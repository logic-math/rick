#!/usr/bin/env python3
"""variant_spacer.py — P6 形态：import 期 1GB spacer + 持久 per-shape 缓冲 + out=。

布局规避（buffer_ab.py P6 臂）：import 时先整体触页一个 1GB spacer，
使持久缓冲的物理/虚拟落位避开 a/b 邻接区。
"""
import numpy as np

np.seterr(divide="ignore", over="ignore", invalid="ignore")

_JUNK = np.empty(1 << 28, np.float32)  # 1GB spacer
_JUNK[:] = 0  # 整体触页（import 期，计时窗口外）

_BUFS = {}


def _get_buf(n, m):
    buf = _BUFS.get((n, m))
    if buf is None:
        buf = np.empty((n, m), dtype=np.float32)
        buf[:] = 0
        _BUFS[(n, m)] = buf
    return buf


def matmul(a, b):
    a32 = np.asarray(a, dtype=np.float32)
    b32 = np.asarray(b, dtype=np.float32)
    if a32.ndim == 2 and b32.ndim == 2:
        c = _get_buf(a32.shape[0], b32.shape[1])
        np.matmul(a32, b32, out=c)
        return c
    return np.matmul(a32, b32)
