#!/usr/bin/env python3
"""diag_pure_matmul.py — 诊断：纯 matmul 进程（不混 einsum）的水平测量。

目的：区分 A/B 进程中 matmul 变慢的两种解释——
(a) 机器整体变慢（纯 matmul 进程也升到 ~8ms）；
(b) 混跑效应（同进程交错 einsum/matmul 时 matmul 被压制，纯 matmul 仍 ~7ms）。
协议与 ab_test.py 一致：10 轮 × K=3 取中位（仅一个函数，无配对）。
"""
import json
import time

import numpy as np

np.seterr(divide="ignore", over="ignore", invalid="ignore")

N = 2048
K = 3
ROUNDS = 10

rng = np.random.RandomState(42)
a32 = rng.rand(N, N).astype(np.float32)
b32 = rng.rand(N, N).astype(np.float32)


def median3(fn):
    ts = []
    for _ in range(K):
        t0 = time.perf_counter()
        fn(a32, b32)
        ts.append((time.perf_counter() - t0) * 1000.0)
    ts.sort()
    return ts[K // 2]


# 预热 2 次
np.matmul(a32, b32); np.matmul(a32, b32)

medians = [median3(np.matmul) for _ in range(ROUNDS)]
print(json.dumps({
    "pure_matmul_median_of_medians": round(float(np.median(medians)), 3),
    "min": round(min(medians), 3),
    "max": round(max(medians), 3),
    "medians": [round(t, 3) for t in medians],
}, ensure_ascii=False))
