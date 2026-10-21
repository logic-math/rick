#!/usr/bin/env python3
"""swapped spelling candidate (pre-research only)."""
import numpy as np
np.seterr(divide="ignore", over="ignore", invalid="ignore")

def matmul(a, b):
    a32 = np.asarray(a, dtype=np.float32)
    b32 = np.asarray(b, dtype=np.float32)
    return np.einsum('jk,ij->ik', b32, a32, optimize=True)
