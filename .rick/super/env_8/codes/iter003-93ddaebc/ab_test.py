#!/usr/bin/env python3
"""ab_test.py — 迭代 3 A/B 归因临时脚本（不改变提交实现）。

设计：进程内配对、随机次序 ≥10 对、每对 K=3 取中位：
- candidate: np.matmul(a32, b32)                                → 本迭代（直接整阵派发）
- parent:    np.einsum('ij,jk->ik', a32, b32, optimize=True)    → 父本（canonical einsum）

假设：候选不劣于父本（低负载态可见削掉 einsum 包装层 ~0.1-0.15ms，
负载态持平于噪声带内）。可证伪点：若候选系统性慢于父本 >0.1ms（中位之差）
则主张被证伪。
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


def candidate(a, b):
    return np.matmul(a, b)


def parent(a, b):
    return np.einsum('ij,jk->ik', a, b, optimize=True)


FUNCS = {"candidate": candidate, "parent": parent}

# 预热各 2 次
for f in FUNCS.values():
    f(a32, b32); f(a32, b32)

# 正确性 sanity：两实现逐位一致
r_can, r_par = candidate(a32, b32), parent(a32, b32)
print("sanity bitwise: candidate == parent %s" % np.array_equal(r_can, r_par))


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

diffs = [p - c for c, p in zip(results["candidate"], results["parent"])]
summary["parent_minus_candidate_per_pair"] = [round(d, 3) for d in diffs]
summary["candidate_faster_mean_ms"] = round(float(np.mean(diffs)), 3)
summary["candidate_faster_min_ms"] = round(float(min(diffs)), 3)
summary["pairs_candidate_faster"] = int(sum(1 for d in diffs if d > 0))

print(json.dumps(summary, ensure_ascii=False))
