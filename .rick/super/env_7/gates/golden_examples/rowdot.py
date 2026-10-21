#!/usr/bin/env python3
"""rowdot.py — 金样例 2：逐行 np.dot 风格已知好解（与参考不同的风格）。

研究实测：score ~0.15（一步向量化形态），max_error ~7.6e-6（正确性门通过）。
作用：门禁升级时防误杀面（不同风格的合法好解必须继续通过）。
"""
import numpy as np

np.seterr(divide="ignore", over="ignore", invalid="ignore")


def matmul(a, b):
    a32 = np.asarray(a, dtype=np.float32)
    b32 = np.asarray(b, dtype=np.float32)
    return np.stack([np.dot(a32[i], b32) for i in range(a32.shape[0])])
