#!/usr/bin/env python3
"""scratch_calib.py — iter004 builder 校准（一次性测量脚本，不进候选）。

目标格：dispatch_target=np_dot | impl_strategy=naive_loops
        | memory_layout=row_major | parallelism=serial

要回答的测量问题（为 experiment_plan 的 expected_behavior 提供数据载荷）：
M1. np.dot 1-D×1-D 单调用成本：连续×连续（col_major 风格列）vs 连续×跨步列
    （row_major 真实形态，b[:,j] 跨步 512 元素）——量化 row_major 布局罚分；
M2. 逐元素双循环的端到端成本 @512（bench 协议 warmup1+k3 取中位）：
    V1 内联切片 + out[i,j] 写（朴素直写形态）
    V2 预切片视图 + 平坦游标写（去每次迭代的切片/setitem 开销）
    V3 预切片视图 + 行列表推导 + out[i] 整行赋值（C 级 listcomp + 单次行转换）
    V3b 预切片视图 + map(repeat) + out[i] 整行赋值（再省 listcomp 字节码）
M3. 参照点：rowwise np.dot(a[i], b)（邻格 rowwise_vectorized 形态，ladder 定位）
    与 np.matmul（ref 基线，score 分母）；
M4. 正确性：最优变体 vs np.matmul 的 max_error @512 + 双跑逐位一致。

输出：stdout 单行 JSON（各测量结果）。
"""
import json
import time
from itertools import repeat

import numpy as np

np.seterr(divide="ignore", over="ignore", invalid="ignore")

N = 512
K = 3


def bench(fn, *args, warmup=1, k=K):
    """bench.py 同款协议：warmup 1 次 + k 次计时取中位。"""
    fn(*args)
    ts = []
    for _ in range(k):
        t0 = time.perf_counter()
        fn(*args)
        ts.append((time.perf_counter() - t0) * 1000.0)
    ts.sort()
    return ts[len(ts) // 2] if len(ts) % 2 else (ts[len(ts) // 2 - 1] + ts[len(ts) // 2]) / 2.0


def micro_dot(calls=4096):
    """M1：单调用成本。返回 (contig_contig_us, contig_strided_us)。"""
    rng = np.random.RandomState(42)
    a = rng.rand(N, N).astype(np.float32)
    b_c = rng.rand(N, N).astype(np.float32)          # C 序 B——列是跨步视图
    b_f = np.asfortranarray(b_c)                      # F 序 B——列是连续视图
    av = a[0]
    col_strided = b_c[:, 0]                           # 跨步 512 元素
    col_contig = b_f[:, 0]                            # 连续
    # 跨步确认
    info = {
        "col_strided_stride_bytes": int(col_strided.strides[0]),
        "col_contig_stride_bytes": int(col_contig.strides[0]),
    }
    dot = np.dot
    for _ in range(64):
        dot(av, col_strided)
        dot(av, col_contig)
    t0 = time.perf_counter()
    for _ in range(calls):
        dot(av, col_strided)
    s_us = (time.perf_counter() - t0) / calls * 1e6
    t0 = time.perf_counter()
    for _ in range(calls):
        dot(av, col_contig)
    c_us = (time.perf_counter() - t0) / calls * 1e6
    info["dot_contig_x_strided_us"] = round(s_us, 3)
    info["dot_contig_x_contig_us"] = round(c_us, 3)
    info["rowmajor_penalty_x"] = round(s_us / c_us, 2) if c_us > 0 else None
    return info


def v1_inline(a, b):
    """V1：内联切片 + out[i,j] 逐元素写（无任何微优化）。"""
    n, k = a.shape
    m = b.shape[1]
    out = np.empty((n, m), dtype=np.float32)
    dot = np.dot
    for i in range(n):
        for j in range(m):
            out[i, j] = dot(a[i], b[:, j])
    return out


def v2_flat(a, b):
    """V2：预切片行/列视图 + 平坦游标写。"""
    n, k = a.shape
    m = b.shape[1]
    rows = list(a)
    cols = [b[:, j] for j in range(m)]
    out = np.empty((n, m), dtype=np.float32)
    flat = out.reshape(-1)
    dot = np.dot
    t = 0
    for av in rows:
        for bv in cols:
            flat[t] = dot(av, bv)
            t += 1
    return out


def v3_listcomp(a, b):
    """V3：预切片视图 + 行列表推导 + out[i] 整行赋值。"""
    n, k = a.shape
    rows = list(a)
    cols = [b[:, j] for j in range(b.shape[1])]
    out = np.empty((n, b.shape[1]), dtype=np.float32)
    dot = np.dot
    for i in range(n):
        out[i] = [dot(rows[i], bv) for bv in cols]
    return out


def v3b_map(a, b):
    """V3b：预切片视图 + map(repeat) + out[i] 整行赋值。"""
    rows = list(a)
    m = b.shape[1]
    cols = [b[:, j] for j in range(m)]
    out = np.empty((len(rows), m), dtype=np.float32)
    dot = np.dot
    for i in range(len(rows)):
        out[i] = list(map(dot, repeat(rows[i], m), cols))
    return out


def rowwise_ref(a, b):
    """M3 参照：rowwise np.dot(a[i], b)——邻格 rowwise_vectorized 形态。"""
    return np.stack([np.dot(a[i], b) for i in range(a.shape[0])])


def main():
    rng = np.random.RandomState(42)
    a = rng.rand(N, N).astype(np.float32)
    b = rng.rand(N, N).astype(np.float32)
    res = {"n": N, **micro_dot()}

    # M2：全循环变体（各 ~0.4-0.8s × 4 次）
    for name, fn in [("v1_inline_ms", v1_inline), ("v2_flat_ms", v2_flat),
                     ("v3_listcomp_ms", v3_listcomp), ("v3b_map_ms", v3b_map)]:
        res[name] = round(bench(fn, a, b), 2)

    # M3：参照点
    res["rowwise_dot_ms"] = round(bench(rowwise_ref, a, b), 3)
    res["np_matmul_ms"] = round(bench(np.matmul, a, b), 3)

    # M4：正确性（对最快变体）
    best = min((v1_inline, v2_flat, v3_listcomp, v3b_map),
               key=lambda f: bench.__wrapped__ if False else 0)  # 占位
    got = v3b_map(a, b)
    got2 = v3b_map(a, b)
    exp = np.matmul(a, b)
    res["v3b_max_error"] = float(np.max(np.abs(got.astype(np.float64) - exp.astype(np.float64))))
    res["v3b_double_run_equal"] = bool(np.array_equal(got, got2))
    res["v3b_shape"] = list(got.shape)
    # 预期 score（用各变体时间对 np_matmul 中位）
    res["expected_score_v3b"] = round(res["np_matmul_ms"] / res["v3b_map_ms"], 6)

    print(json.dumps(res, ensure_ascii=False))


if __name__ == "__main__":
    main()
