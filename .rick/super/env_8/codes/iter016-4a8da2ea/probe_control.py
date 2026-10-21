#!/usr/bin/env python3
"""probe_control.py — 对照组：iter015 零开销预绑定通道原样（无导入期预热）。

仅用于 A/B 实验证据采集，非交付代码。
"""
import numpy as np

np.seterr(divide="ignore", over="ignore", invalid="ignore")

_matmul = np.matmul


def matmul(a, b):
    return _matmul(a, b)
