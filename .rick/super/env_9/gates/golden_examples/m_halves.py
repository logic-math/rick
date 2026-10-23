#!/usr/bin/env python3
"""m_halves.py — 金样例 3：M 维二分风格已知好解（与 rowwise/直调不同的风格）。

研究实测：@2048 ~7.6ms（score ~0.90），与 np.matmul 逐位一致（M 维分解不改变
K 求和序，max_error = 0.0），1e-3 容差必过。
作用：门禁升级时防误杀面（M 分解族合法好解必须继续通过）。
"""
import numpy as np

np.seterr(divide="ignore", over="ignore", invalid="ignore")


def matmul(a, b):
    a32 = np.asarray(a, dtype=np.float32)
    b32 = np.asarray(b, dtype=np.float32)
    h = a32.shape[0] // 2
    return np.concatenate([np.matmul(a32[:h], b32), np.matmul(a32[h:], b32)], axis=0)
