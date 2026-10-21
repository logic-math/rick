#!/usr/bin/env python3
"""scratch_calib2.py — iter003 第二轮微基准：内积原语横向对比（纯 Python 内）。

第一轮结论：map(sum+operator.mul) 逐行向量化 @512 = 1675ms（vs parent 4190ms）。
本轮筛选：
  A sum(map(mul))          —— 第一轮主候选（复测稳定性）
  B fsum(map(mul))         —— math.fsum：C 级浮点专用累加（无逐元素结果对象、无双目协议）
  C sum(map(mul))+locals   —— 默认参数抬为局部名（LOAD_FAST 替代 LOAD_GLOBAL）
  D reduce(add, map(mul))  —— functools.reduce 收缩
  E 父本 naive_start 原样 numpy 输入路径（gate 可见基线，单次）
全部仍为 pure_python × rowwise_vectorized × col_major × serial。
"""
import json
import time
from functools import reduce as _reduce
from operator import add as _add
from operator import mul as _mul

import numpy as np

from math import fsum as _fsum

np.seterr(divide="ignore", over="ignore", invalid="ignore")

ENV = "/Users/logic/workdir/rick-dev/.rick/super/env_7"


def _as_list_of_lists(x):
    tolist = getattr(x, "tolist", None)
    if tolist is not None:
        return tolist()
    return [row if isinstance(row, list) else list(row) for row in x]


def matmul_sum(a, b):
    a_l = _as_list_of_lists(a)
    cols = list(zip(*_as_list_of_lists(b)))
    return [[sum(map(_mul, ai, col)) for col in cols] for ai in a_l]


def matmul_fsum(a, b):
    a_l = _as_list_of_lists(a)
    cols = list(zip(*_as_list_of_lists(b)))
    return [[_fsum(map(_mul, ai, col)) for col in cols] for ai in a_l]


def matmul_locals(a, b, _sum=sum, _map=map, mul=_mul):
    a_l = _as_list_of_lists(a)
    cols = list(zip(*_as_list_of_lists(b)))
    return [[_sum(_map(mul, ai, col)) for col in cols] for ai in a_l]


def matmul_reduce(a, b):
    a_l = _as_list_of_lists(a)
    cols = list(zip(*_as_list_of_lists(b)))
    return [[_reduce(_add, map(_mul, ai, col)) for col in cols] for ai in a_l]


def check(fn, n, seed=42):
    rng = np.random.RandomState(seed)
    a = rng.rand(n, n).astype(np.float32)
    b = rng.rand(n, n).astype(np.float32)
    exp = np.matmul(a, b)
    got = np.asarray(fn(a, b), dtype=np.float64)
    assert got.shape == exp.shape
    return float(np.max(np.abs(got - exp.astype(np.float64))))


def bench(fn, n, warmup=1, k=3, seed=42):
    rng = np.random.RandomState(seed)
    a = rng.rand(n, n).astype(np.float32)
    b = rng.rand(n, n).astype(np.float32)
    for _ in range(warmup):
        fn(a, b)
    ts = sorted((lambda t0: (time.perf_counter() - t0) * 1000.0)(time.perf_counter())
                for _ in range(0))  # placeholder
    ts = []
    for _ in range(k):
        t0 = time.perf_counter()
        fn(a, b)
        ts.append((time.perf_counter() - t0) * 1000.0)
    ts.sort()
    return ts[k // 2], ts


def main():
    out = {}

    # 256 快筛
    for name, fn in [("sum", matmul_sum), ("fsum", matmul_fsum),
                     ("locals", matmul_locals), ("reduce", matmul_reduce)]:
        err = check(fn, 128)
        med, ts = bench(fn, 256, k=2)
        out[name] = {"err_128": err, "ms_256": round(med, 1)}

    # 512 对 top2（fsum vs sum vs locals 按快筛结果取前二 + sum 基线）
    for name, fn in [("sum", matmul_sum), ("fsum", matmul_fsum), ("locals", matmul_locals)]:
        med, ts = bench(fn, 512)
        err = check(fn, 512)
        out[name].update({"ms_512": round(med, 1),
                          "timings_512": [round(t, 1) for t in ts],
                          "err_512": err})

    # 父本 naive_start 原样（numpy 输入路径，gate 可见形态）单次 @512
    import importlib.util
    spec = importlib.util.spec_from_file_location(
        "naive_start", ENV + "/env/code/naive_start.py")
    mod = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(mod)
    rng = np.random.RandomState(42)
    a = rng.rand(512, 512).astype(np.float32)
    b = rng.rand(512, 512).astype(np.float32)
    mod.matmul(a, b)  # warmup
    t0 = time.perf_counter()
    mod.matmul(a, b)
    out["parent_numpy_input_ms_512"] = round((time.perf_counter() - t0) * 1000.0, 1)

    # 双跑确定性（fsum 形态）
    rng = np.random.RandomState(7)
    x = rng.rand(128, 128).astype(np.float32)
    y = rng.rand(128, 128).astype(np.float32)
    r1 = np.asarray(matmul_fsum(x, y), dtype=np.float64)
    r2 = np.asarray(matmul_fsum(x, y), dtype=np.float64)
    out["fsum_deterministic_128"] = bool(np.array_equal(r1, r2))
    r3 = np.asarray(matmul_locals(x, y), dtype=np.float64)
    out["locals_deterministic_128"] = bool(np.array_equal(r1, r3) and np.array_equal(r1, r3))

    print(json.dumps(out, ensure_ascii=False, indent=1))
    with open(__file__.replace(".py", "_result.json"), "w", encoding="utf-8") as f:
        json.dump(out, f, ensure_ascii=False, indent=1)


if __name__ == "__main__":
    main()
