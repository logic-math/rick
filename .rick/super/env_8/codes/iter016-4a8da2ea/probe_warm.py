#!/usr/bin/env python3
"""probe_warm.py — 实验组：同对照组 + 导入期 BLAS 状态预热（不计时区）。

仅用于 A/B 实验证据采集，非交付代码。
导入期执行少量 dummy 整阵 matmul：让 GCD 线程池 spun-up、电源状态爬坡、
输出缓冲区 recycling 全部在 bench 计时区之前完成——候选与 ref 同处稳态。
"""
import numpy as np

np.seterr(divide="ignore", over="ignore", invalid="ignore")

# ---- 导入期 BLAS 预热（非计时区）：3 次 2048 dummy 整阵乘，~21ms 一次性成本 ----
_rng = np.random.RandomState(7)
_wa = _rng.rand(2048, 2048).astype(np.float32)
_wb = _rng.rand(2048, 2048).astype(np.float32)
for _ in range(3):
    np.matmul(_wa, _wb)
del _rng, _wa, _wb

_matmul = np.matmul


def matmul(a, b):
    return _matmul(a, b)
