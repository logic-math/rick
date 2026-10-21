#!/usr/bin/env python3
"""Pre-research for iter020: np_dot|full_blas_dispatch — swapped-operand transpose-view dispatch.

Tests:
  V0 baseline: np.dot(a32, b32)                       (current elite, NN row-major gemm)
  V1 ref:      np.matmul(a32, b32)                    (axis best 6.859)
  V2 swapped:  np.dot(b32.T, a32.T).T                 (different gemm flag config?)
  V3 out=:     np.empty + np.dot(a32, b32, out=c)
  V4 method:   a32.dot(b32)

Checks bit-identity vs np.matmul at 64/2047/2048 multi-seed, layout of results,
then interleaved timing rounds (medians) at 2048.
"""
import numpy as np
import sys
from time import perf_counter

np.seterr(divide="ignore", over="ignore", invalid="ignore")

VARIANTS = {
    "V0_dot": lambda a, b: np.dot(a, b),
    "V1_matmul": lambda a, b: np.matmul(a, b),
    "V2_swapped": lambda a, b: np.dot(b.T, a.T).T,
    "V3_out": lambda a, b: np.dot(a, b, out=np.empty((a.shape[0], b.shape[1]), dtype=np.float32)),
    "V4_method": lambda a, b: a.dot(b),
}

# ---------- 1. bit-identity ----------
print("=== bit-identity vs np.matmul ===")
for n in (64, 2047, 2048):
    for seed in (42, 123, 777):
        rng = np.random.RandomState(seed)
        a = rng.rand(n, n).astype(np.float32)
        b = rng.rand(n, n).astype(np.float32)
        ref = np.matmul(a, b)
        row = []
        for name, fn in VARIANTS.items():
            got = fn(a, b)
            me = float(np.max(np.abs(got.astype(np.float64) - ref.astype(np.float64))))
            row.append("%s=%.1e" % (name, me))
        print("n=%-5d seed=%-4d %s" % (n, seed, " ".join(row)))

# ---------- 2. layout probe ----------
print("=== layout @2048 ===")
rng = np.random.RandomState(42)
a = rng.rand(2048, 2048).astype(np.float32)
b = rng.rand(2048, 2048).astype(np.float32)
for name, fn in VARIANTS.items():
    r = fn(a, b)
    print("%-11s shape=%s dtype=%s C=%s F=%s base-is-view=%s" % (
        name, r.shape, r.dtype, r.flags["C_CONTIGUOUS"], r.flags["F_CONTIGUOUS"],
        r.base is not None))

# ---------- 3. interleaved timing ----------
print("=== timing @2048 (9 rounds x 5 variants, interleaved; report per-variant median & min) ===")
names = list(VARIANTS.keys())
fns = [VARIANTS[k] for k in names]
for f in fns:
    f(a, b)  # warmup all
samples = {k: [] for k in names}
for rnd in range(9):
    for name, fn in zip(names, fns):
        t0 = perf_counter()
        fn(a, b)
        samples[name].append((perf_counter() - t0) * 1000.0)
for name in names:
    vs = sorted(samples[name])
    print("%-11s median=%.3f min=%.3f all=[%s]" % (
        name, vs[len(vs)//2], vs[0], ", ".join("%.2f" % v for v in vs)))
