#!/usr/bin/env python3
"""ab_test.py — 迭代 12 A/B 归因临时脚本（不改变提交实现）。

单变量假设（本迭代 idea）：swapped 拼写（C-标志 gemm，无尾部视图）在当前机器态
比父本 canonical（T-标志 gemm + 尾部 ki->ik 视图）快 ~0.17-0.23ms。

设计（复用父本 ab_test.py 方法 + ab_micro.py 口径，K=5）：
- canonical: np.einsum('ij,jk->ik', a32, b32, optimize=True)   → T-标志 gemm + 尾部视图
- swapped:   np.einsum('jk,ij->ik', b32, a32, optimize=True)   → C-标志 gemm 无尾部（本迭代提交实现）
- matmul:    np.matmul(a32, b32)                                → CC gemm 基线

进程内配对、每轮随机次序 10 对、每对 K=5 取中位。
"""
import json
import random
import time

import numpy as np

np.seterr(divide="ignore", over="ignore", invalid="ignore")

N = 2048
K = 5
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


def median_k(fn):
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
        results[name].append(median_k(FUNCS[name]))

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
summary["swapped_faster_mean_ms"] = round(float(np.mean(diffs)), 3)
summary["swapped_faster_min_ms"] = round(float(min(diffs)), 3)
summary["pairs_swapped_faster"] = int(sum(1 for d in diffs if d < 0))

print(json.dumps(summary, ensure_ascii=False))
