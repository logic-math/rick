#!/usr/bin/env python3
"""ki tail-skip spelling candidate (pre-research only)."""
import numpy as np
np.seterr(divide="ignore", over="ignore", invalid="ignore")

def matmul(a, b):
    a32 = np.asarray(a, dtype=np.float32)
    b32 = np.asarray(b, dtype=np.float32)
    return np.einsum('ij,jk->ki', a32, b32, optimize=True).T
