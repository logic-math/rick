#!/usr/bin/env python3
"""out_ab.py — kernel 微变体 A/B（单元格方法描述 ①②③ 的预验证）。

V0: plain np.matmul(a, b)（父本形态 = 基线）
V1: out= 持久预分配 + 预 fault 缓冲（复用形态，域简报称已证伪——作对照）
V2: 每次 np.empty + c[:]=0 预 fault + out=（单元格 ③ 形态——该反转侧从未实测）
V3: np.dot(a, b)（同 cblas 路径直调对照）

单进程顺序执行，两轮（round 交错）以判别信号 vs 噪声；读数即记录。
"""
import time

import numpy as np
np.seterr(divide="ignore", over="ignore", invalid="ignore")

n = 8192
rng = np.random.RandomState(42)
a = rng.rand(n, n).astype(np.float32)
b = rng.rand(n, n).astype(np.float32)

cbuf = np.empty((n, n), np.float32)
cbuf[:] = 0  # 预 fault


def v0():
    return np.matmul(a, b)


def v1():
    np.matmul(a, b, out=cbuf)
    return cbuf


def v2():
    c = np.empty((n, n), np.float32)
    c[:] = 0  # 预 fault（OpenBLAS#3197 / numpy#18669 反转侧）
    np.matmul(a, b, out=c)
    return c


def v3():
    return np.dot(a, b)


def med(fn, k=3):
    ts = []
    for _ in range(k):
        t0 = time.perf_counter()
        fn()
        ts.append((time.perf_counter() - t0) * 1e3)
    ts.sort()
    return ts[len(ts) // 2], ts


arms = [("V0_plain", v0), ("V1_out_reuse", v1), ("V2_prefault_fresh", v2), ("V3_dot", v3)]
for rnd in (1, 2):
    for name, fn in arms:
        fn()  # 预热一次（V2 的预热含分配）
        m, ts = med(fn)
        print("round%d %-18s median=%.1f timings=%s" % (rnd, name, m, [round(x, 1) for x in ts]))
