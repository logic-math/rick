#!/usr/bin/env python3
"""ab_test.py — 迭代 2 A/B 归因临时脚本（不改变提交实现）。

设计：进程内配对、随机次序 ≥8 对、每对 K=3 取中位：
- canonical: np.einsum('ij,jk->ik', a32, b32, optimize=True)   → T-标志 gemm
- swapped:   np.einsum('jk,ij->ik', b32, a32, optimize=True)   → C-标志 gemm
- matmul:    np.matmul(a32, b32)                                → 基线

假设：负载态 canonical 比 swapped 快 ≥0.2ms。
"""
import json
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


def canonical(a, b):
    return np.einsum('ij,jk->ik', a, b, optimize=True)


def swapped(a, b):
    return np.einsum('jk,ij->ik', b, a, optimize=True)


FUNCS = {"canonical": canonical, "swapped": swapped, "matmul": np.matmul}

# 预热各 2 次
for f in FUNCS.values():
    f(a32, b32); f(a32, b32)

# 正确性 sanity：三种实现逐位一致
r_can, r_swa, r_mat = canonical(a32, b32), swapped(a32, b32), np.matmul(a32, b32)
print("sanity bitwise: can==mat %s, swa==mat %s" %
      (np.array_equal(r_can, r_mat), np.array_equal(r_swa, r_mat)))


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

summary = {}
for name, ts in results.items():
    summary[name] = {
        "median_of_medians": round(float(np.median(ts)), 3),
        "min": round(min(ts), 3),
        "max": round(max(ts), 3),
        "medians": [round(t, 3) for t in ts],
    }

diffs = [s - c for c, s in zip(results["canonical"], results["swapped"])]
summary["swapped_minus_canonical_per_pair"] = [round(d, 3) for d in diffs]
summary["canonical_faster_mean_ms"] = round(float(np.mean(diffs)), 3)
summary["canonical_faster_min_ms"] = round(float(min(diffs)), 3)
summary["pairs_canonical_faster"] = int(sum(1 for d in diffs if d > 0))

print(json.dumps(summary, ensure_ascii=False))
