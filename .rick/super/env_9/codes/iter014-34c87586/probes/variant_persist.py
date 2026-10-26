#!/usr/bin/env python3
"""variant_persist.py — P1 形态：持久 per-shape 输出缓冲 + out= 派发。

首次调用某形状时分配 + 写入式预 fault（c[:]=0），此后每次调用零分配零缺页，
sgemm 直接写入常驻页。返回同一缓冲对象（每次全量重算，非缓存结果）。
"""
import numpy as np

np.seterr(divide="ignore", over="ignore", invalid="ignore")

_BUFS = {}


def _get_buf(n, m):
    buf = _BUFS.get((n, m))
    if buf is None:
        buf = np.empty((n, m), dtype=np.float32)
        buf[:] = 0  # 写入式预 fault（OpenBLAS#3197 / numpy#18669 预 fault 侧）
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
