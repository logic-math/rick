#!/usr/bin/env python3
"""scratch_calib.py — iter003 builder 校准：pure_python × rowwise_vectorized × col_major × serial。

目的（先测量，后动手）：
- 目标格结构：B 经 zip(*b) 转置为列元组（col_major，与父本 naive_start 同约定），
  每个输出行用一个列表推导产出，行内每个元素的内积收缩用 C 级内建
  sum(map(operator.mul, ai, col))（rowwise_vectorized 的纯 Python 表达），
  单线程串行（serial）。
- 消融矩阵（归因载荷）：
  T_parent : 父本 naive_start 原样（zip 转置 + 双下标标量内层循环）——gate 可见基线
  T_noT    : 无转置（b[t][j] 双下标）+ 标量循环 ——隔离 memory_layout（转置）效应
  T_zip    : zip 转置 + zip 标量循环 ——隔离「下标 vs 迭代」效应（布局不变）
  T_map    : zip 转置 + sum(map(mul)) 逐行向量化 ——主候选（隔离向量化效应）
  T_lcomp  : zip 转置 + sum([x*y for zip]) 逐行向量化变体 ——内积原语横向对比
  T_axpy   : 行累加 axpy 形态（row-major 友好）——记录用，预期与 col_major 不匹配且更慢
- 正确性：全部变体 vs np.matmul（fp64 比较，容差 1e-3）。
- 规模：先 256 快筛，主候选再按 bench 协议（warmup1+k3 中位）测 512。

输出：scratch_calib_result.json（同目录）。
"""
import json
import time
from operator import mul as _mul

import numpy as np

np.seterr(divide="ignore", over="ignore", invalid="ignore")

ENV = "/Users/logic/workdir/rick-dev/.rick/super/env_7"


def _as_list_of_lists(x):
    """numpy ndarray → 嵌套 list（Python float）；已是 list 的归一为 list-of-list。"""
    tolist = getattr(x, "tolist", None)
    if tolist is not None:
        return tolist()
    return [row if isinstance(row, list) else list(row) for row in x]


# ---- 变体 ----

def matmul_parent(a, b):
    """父本 naive_start 原样（zip 转置 + ai[t]*bj[t] 双下标标量循环）。"""
    a_l = _as_list_of_lists(a)
    b_l = _as_list_of_lists(b)
    n, k, m = len(a_l), len(b_l), len(b_l[0]) if b_l else 0
    bt = list(zip(*b_l))
    c = []
    for i in range(n):
        ai = a_l[i]
        row = [0.0] * m
        for j in range(m):
            bj = bt[j]
            s = 0.0
            for t in range(k):
                s += ai[t] * bj[t]
            row[j] = s
        c.append(row)
    return c


def matmul_noT(a, b):
    """无转置：b[t][j] 双下标 + 标量循环（row_major-ish 布局的标量形态）。"""
    a_l = _as_list_of_lists(a)
    b_l = _as_list_of_lists(b)
    n, k, m = len(a_l), len(b_l), len(b_l[0]) if b_l else 0
    c = []
    for i in range(n):
        ai = a_l[i]
        row = [0.0] * m
        for j in range(m):
            s = 0.0
            for t in range(k):
                s += ai[t] * b_l[t][j]
            row[j] = s
        c.append(row)
    return c


def matmul_zip(a, b):
    """zip 转置 + zip(ai,col) 标量循环（布局同主候选，循环层级同为 Python 标量）。"""
    a_l = _as_list_of_lists(a)
    b_l = _as_list_of_lists(b)
    cols = list(zip(*b_l))
    c = []
    for ai in a_l:
        row = []
        for col in cols:
            s = 0.0
            for x, y in zip(ai, col):
                s += x * y
            row.append(s)
        c.append(row)
    return c


def matmul_map(a, b):
    """主候选：zip 转置（col_major）+ 每输出行一个列表推导，元素内积
    sum(map(operator.mul, ai, col))（C 级收缩 = rowwise_vectorized 纯 Python 表达）。"""
    a_l = _as_list_of_lists(a)
    b_l = _as_list_of_lists(b)
    cols = list(zip(*b_l))
    return [[sum(map(_mul, ai, col)) for col in cols] for ai in a_l]


def matmul_lcomp(a, b):
    """变体：内积原语换 sum([x*y for x,y in zip(ai,col)])。"""
    a_l = _as_list_of_lists(a)
    b_l = _as_list_of_lists(b)
    cols = list(zip(*b_l))
    return [[sum([x * y for x, y in zip(ai, col)]) for col in cols] for ai in a_l]


def matmul_axpy(a, b):
    """行累加形态（B 行消费 = row_major 友好；记录用）。"""
    a_l = _as_list_of_lists(a)
    b_l = _as_list_of_lists(b)
    out = []
    for ai in a_l:
        it = iter(b_l)
        acc = [ai[0] * x for x in next(it)]
        for f, brow in zip(ai[1:], it):
            acc = [x + f * y for x, y in zip(acc, brow)]
        out.append(acc)
    return out


VARIANTS = [
    ("parent", matmul_parent),
    ("noT", matmul_noT),
    ("zip_scalar", matmul_zip),
    ("map", matmul_map),
    ("lcomp", matmul_lcomp),
    ("axpy", matmul_axpy),
]


def check_correctness(fn, n, seed=42):
    rng = np.random.RandomState(seed)
    a = rng.rand(n, n).astype(np.float32)
    b = rng.rand(n, n).astype(np.float32)
    expected = np.matmul(a, b)
    got = fn(a, b)
    got_arr = np.asarray(got, dtype=np.float64)
    if got_arr.shape != expected.shape:
        return None, "shape %s vs %s" % (got_arr.shape, expected.shape)
    err = float(np.max(np.abs(got_arr - expected.astype(np.float64))))
    return err, None


def time_once(fn, a, b):
    t0 = time.perf_counter()
    fn(a, b)
    return (time.perf_counter() - t0) * 1000.0


def bench_protocol(fn, n, seed=42, warmup=1, k=3):
    rng = np.random.RandomState(seed)
    a = rng.rand(n, n).astype(np.float32)
    b = rng.rand(n, n).astype(np.float32)
    for _ in range(warmup):
        fn(a, b)
    ts = sorted(time_once(fn, a, b) for _ in range(k))
    return ts[k // 2], ts


def main():
    result = {"n_fast": 256, "n_main": 512, "variants": {}, "notes": []}

    # 1) 快筛 n=256：每变体 1 次预热 + 2 次计时取小（粗筛，误差可接受）
    for name, fn in VARIANTS:
        err, err_msg = check_correctness(fn, 128)
        if err is None:
            result["variants"][name] = {"correctness_error_128": None, "error": err_msg}
            continue
        med, ts = bench_protocol(fn, 256, warmup=1, k=2)
        result["variants"][name] = {
            "correctness_error_128": err,
            "ms_256": round(med, 1),
            "timings_256": [round(t, 1) for t in ts],
        }

    # 2) 主候选与父本 n=512（bench 协议 warmup1+k3 中位）
    for name in ("parent", "map", "lcomp"):
        fn = dict(VARIANTS)[name]
        med, ts = bench_protocol(fn, 512)
        err, _ = check_correctness(fn, 512)
        result["variants"][name].update({
            "ms_512": round(med, 1),
            "timings_512": [round(t, 1) for t in ts],
            "correctness_error_512": err,
        })

    # 3) 预处理成本拆解（tolist + zip(*b) @512）
    rng = np.random.RandomState(42)
    a = rng.rand(512, 512).astype(np.float32)
    b = rng.rand(512, 512).astype(np.float32)
    t0 = time.perf_counter()
    for _ in range(5):
        a_l = _as_list_of_lists(a)
        b_l = _as_list_of_lists(b)
        cols = list(zip(*b_l))
    prep_ms = (time.perf_counter() - t0) * 1000.0 / 5
    result["prep_ms_512"] = round(prep_ms, 2)

    # 4) 双跑逐位一致性（确定性 sanity，128）
    rng = np.random.RandomState(7)
    x = rng.rand(128, 128).astype(np.float32)
    y = rng.rand(128, 128).astype(np.float32)
    r1 = np.asarray(matmul_map(x, y), dtype=np.float64)
    r2 = np.asarray(matmul_map(x, y), dtype=np.float64)
    result["deterministic_128"] = bool(np.array_equal(r1, r2))

    # 5) 派生量
    v = result["variants"]
    if "ms_512" in v.get("map", {}) and "ms_512" in v.get("parent", {}):
        result["speedup_vs_parent_512"] = round(v["parent"]["ms_512"] / v["map"]["ms_512"], 2)
        result["score_pred"] = round(0.185 / (v["map"]["ms_512"] / 1000.0), 8)

    print(json.dumps(result, ensure_ascii=False, indent=1))
    with open(__file__.replace(".py", "_result.json"), "w", encoding="utf-8") as f:
        json.dump(result, f, ensure_ascii=False, indent=1)


if __name__ == "__main__":
    main()
