#!/usr/bin/env python3
"""ab_test.py — 迭代 6 A/B 归因临时脚本（不改变提交实现）。

设计：进程内配对、随机次序 ≥10 对、每对 K=3 取中位。变体（完成 2×2 因子格
+ 骨架对照）：

- candidate:        预分配 out + 逐行 np.matmul(a32[i], b32)   → 本迭代（np_matmul|rowwise_vectorized）
- naive_dot:        np.stack([np.dot(a32[i], b32) ...])        → naive_start.py（np_dot|rowwise 原生形态）
- naive_dot_pre:    预分配 out + 逐行 np.dot(a32[i], b32)      → 骨架对照（隔离纯入口税：与 candidate 仅差派发调用）
- parent:           np.matmul(a32, b32)                        → 父本 iter003（np_matmul|full_blas_dispatch）
- einsum_rowwise:   预分配 out + 逐行 np.einsum('j,jk->k', ...) → iter004（einsum_path|rowwise_vectorized，对照包装税）

假设：
1. rowwise 结构税：candidate − parent ≈ +135ms（内存墙，每行完整流过 B）。
2. matmul 入口税：candidate − naive_dot* ≈ 0±2ms（gufunc 每行派发税 @2048
   折 ~0.2ms 不可见）——与 einsum 同结构 +23.5ms 包装税构成可证伪对照。
可证伪点：若 candidate − naive_dot* > +5ms，则 gufunc 派发税在 rowwise 下可见，
量级模型需修正。
"""
import json
import platform
import random
import time

import numpy as np

np.seterr(divide="ignore", over="ignore", invalid="ignore")

N = 2048
K = 3
PAIRS = 10

rng = np.random.RandomState(42)
a32 = rng.rand(N, N).astype(np.float32)
b32 = rng.rand(N, N).astype(np.float32)


def candidate(a, b):
    """本迭代：预分配 out + 逐行 np.matmul（1-D × 2-D，不转置）。"""
    n = a.shape[0]
    out = np.empty((n, b.shape[1]), dtype=np.float32)
    for i in range(n):
        out[i] = np.matmul(a[i], b)
    return out


def naive_dot(a, b):
    """naive_start.py 原生形态：np.stack + 列表推导逐行 np.dot。"""
    return np.stack([np.dot(a[i], b) for i in range(a.shape[0])])


def naive_dot_pre(a, b):
    """骨架对照：与 candidate 仅差派发调用（np.matmul → np.dot）。"""
    n = a.shape[0]
    out = np.empty((n, b.shape[1]), dtype=np.float32)
    for i in range(n):
        out[i] = np.dot(a[i], b)
    return out


def parent(a, b):
    """父本 iter003：整阵单次 np.matmul。"""
    return np.matmul(a, b)


def einsum_rowwise(a, b):
    """iter004：预分配 out + 逐行 einsum('j,jk->k', optimize=True)。"""
    n = a.shape[0]
    out = np.empty((n, b.shape[1]), dtype=np.float32)
    for i in range(n):
        out[i] = np.einsum('j,jk->k', a[i], b, optimize=True)
    return out


FUNCS = {
    "candidate": candidate,
    "naive_dot": naive_dot,
    "naive_dot_pre": naive_dot_pre,
    "parent": parent,
    "einsum_rowwise": einsum_rowwise,
}

# 预热各 2 次
for f in FUNCS.values():
    f(a32, b32); f(a32, b32)

# 正确性 sanity：全部变体 vs np.matmul 锚（@2048 应逐位一致；naive 家族亦然）
anchor = np.matmul(a32, b32)
sanity = {name: bool(np.array_equal(f(a32, b32), anchor)) for name, f in FUNCS.items()}


def median3(fn):
    ts = []
    for _ in range(K):
        t0 = time.perf_counter()
        fn(a32, b32)
        ts.append((time.perf_counter() - t0) * 1000.0)
    ts.sort()
    return ts[K // 2]


results = {name: [] for name in FUNCS}
order = random.Random(123)
for pair in range(PAIRS):
    names = list(FUNCS)
    order.shuffle(names)
    for name in names:
        results[name].append(median3(FUNCS[name]))

summary = {"sanity_bitwise_vs_matmul": sanity, "n": N, "pairs": PAIRS, "k": K}
for name, ts in results.items():
    summary[name] = {
        "median_of_medians": round(float(np.median(ts)), 3),
        "min": round(min(ts), 3),
        "max": round(max(ts), 3),
    }

# 归因量（逐对配对差，均值/中位）
def paired(base, other):
    diffs = [b - o for b, o in zip(results[base], results[other])]
    return {
        "mean": round(float(np.mean(diffs)), 3),
        "median": round(float(np.median(diffs)), 3),
        "min": round(min(diffs), 3),
        "max": round(max(diffs), 3),
        "per_pair": [round(d, 3) for d in diffs],
    }


summary["attribution"] = {
    "rowwise_structure_tax_candidate_minus_parent": paired("candidate", "parent"),
    "matmul_entry_tax_candidate_minus_naive_dot_pre": paired("candidate", "naive_dot_pre"),
    "matmul_entry_tax_candidate_minus_naive_dot": paired("candidate", "naive_dot"),
    "einsum_packaging_tax_einsum_minus_candidate": paired("einsum_rowwise", "candidate"),
    "naive_skeleton_tax_naive_dot_minus_naive_dot_pre": paired("naive_dot", "naive_dot_pre"),
}

print(json.dumps(summary, ensure_ascii=False))
