#!/usr/bin/env python3
"""variant_plain.py — 对照：父本精英形态（单次整阵 np.matmul，无 out=）。"""
import numpy as np

np.seterr(divide="ignore", over="ignore", invalid="ignore")


def matmul(a, b):
    a32 = np.asarray(a, dtype=np.float32)
    b32 = np.asarray(b, dtype=np.float32)
    return np.matmul(a32, b32)
