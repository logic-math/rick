#!/usr/bin/env python3
"""thread_sweep.py — 单进程单线程数计时（须在独立进程中运行，每配置一次）。

读数即记录：打印 median 与全部 timings，不重复跑。
"""
import os
import time

import numpy as np
np.seterr(divide="ignore", over="ignore", invalid="ignore")

n = 8192
rng = np.random.RandomState(42)
a = rng.rand(n, n).astype(np.float32)
b = rng.rand(n, n).astype(np.float32)

# 触发 BLAS 线程池初始化（小矩阵一次）
np.matmul(a[:512, :512], b[:512, :512])

ts = []
for _ in range(4):
    t0 = time.perf_counter()
    np.matmul(a, b)
    ts.append((time.perf_counter() - t0) * 1e3)
ts.sort()
k = os.environ.get("VECLIB_MAXIMUM_THREADS", "unset")
tag = os.environ.get("SWEEP_TAG", "pre")
print("tag=%s VECLIB=%s median=%.1f timings=%s" % (tag, k, ts[len(ts) // 2], [round(x, 1) for x in ts]))
