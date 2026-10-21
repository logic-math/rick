#!/usr/bin/env python3
"""ab_micro.py — einsum dispatch-chain decomposition @2048 (builder 预研，不提交).

配对交叉计时（每轮随机次序），K=5 每变体取中位，10 轮取中位之中位。
变体：
  matmul    : np.matmul(a, b)                     — CC gemm 基线（ref）
  dot       : np.dot(a, b)                        — CC gemm（np_dot 格）
  canonical : einsum('ij,jk->ik', a, b, True)      — 父本（TT gemm + 尾部 ki->ik 视图）
  ki_T      : einsum('ij,jk->ki', a, b, True).T    — 尾跳过变体（TT gemm + 免费转置视图）
  swapped   : einsum('jk,ij->ik', b, a, True)      — CC gemm 无尾（已知慢路）
  tdot_TT   : tensordot(b, a, ((0,),(1,)))         — canonical 的 gemm 裸调（无 einsum 机器）
  tdot_CC   : tensordot(a, b, ((1,),(0,)))         — swapped 的 gemm 裸调
"""
import time

import numpy as np

np.seterr(divide="ignore", over="ignore", invalid="ignore")

N = 2048
rng = np.random.RandomState(42)
a = rng.rand(N, N).astype(np.float32)
b = rng.rand(N, N).astype(np.float32)

VARIANTS = {
    "matmul":    lambda: np.matmul(a, b),
    "dot":       lambda: np.dot(a, b),
    "canonical": lambda: np.einsum('ij,jk->ik', a, b, optimize=True),
    "ki_T":      lambda: np.einsum('ij,jk->ki', a, b, optimize=True).T,
    "swapped":   lambda: np.einsum('jk,ij->ik', b, a, optimize=True),
    "tdot_TT":   lambda: np.tensordot(b, a, axes=((0,), (1,))),
    "tdot_CC":   lambda: np.tensordot(a, b, axes=((1,), (0,))),
}

# ---- 正确性 sanity：全部与 np.matmul 逐位一致？----
ref = np.matmul(a, b)
for name, fn in VARIANTS.items():
    c = fn()
    bit = bool(np.array_equal(np.ascontiguousarray(c), np.ascontiguousarray(ref)))
    print("correct %s: bitwise=%s dtype=%s C=%s F=%s" % (
        name, bit, c.dtype, c.flags.c_contiguous, c.flags.f_contiguous))

# ---- @64 机器成本（与规模无关的 Python 机器开销）----
n = 64
rng = np.random.RandomState(7)
a64 = rng.rand(n, n).astype(np.float32)
b64 = rng.rand(n, n).astype(np.float32)
fns64 = {
    "matmul":    lambda: np.matmul(a64, b64),
    "canonical": lambda: np.einsum('ij,jk->ik', a64, b64, optimize=True),
    "ki_T":      lambda: np.einsum('ij,jk->ki', a64, b64, optimize=True).T,
    "tdot_TT":   lambda: np.tensordot(b64, a64, axes=((0,), (1,))),
}
for name, fn in fns64.items():
    for _ in range(50):
        fn()
    ts = []
    for _ in range(200):
        t0 = time.perf_counter()
        fn()
        ts.append((time.perf_counter() - t0) * 1e6)
    print("cost64 %s: %.1f us" % (name, sorted(ts)[len(ts)//2]))

# ---- @2048 配对交叉 ----
ROUNDS, K = 10, 5
import random
medians = {k: [] for k in VARIANTS}
for r in range(ROUNDS):
    order = list(VARIANTS.items())
    random.shuffle(order)
    for name, fn in order:
        fn()  # warm
        ts = []
        for _ in range(K):
            t0 = time.perf_counter()
            fn()
            ts.append((time.perf_counter() - t0) * 1000.0)
        medians[name].append(sorted(ts)[K // 2])

print("\n@2048 median-of-medians (ms), %d rounds x K=%d:" % (ROUNDS, K))
base = sorted(medians["matmul"])[ROUNDS // 2]
for name in VARIANTS:
    m = sorted(medians[name])[ROUNDS // 2]
    allm = sorted(medians[name])
    print("  %-9s %.3f  (delta vs matmul: %+.3f)  spread %.3f-%.3f" % (
        name, m, m - base, allm[0], allm[-1]))

# 逐对差（canonical - ki_T）：尾部跳过的净效应
pair_d = [c - k for c, k in zip(medians["canonical"], medians["ki_T"])]
print("\nper-round (canonical - ki_T) ms:", ["%+.3f" % x for x in pair_d])
print("mean: %+.4f ms" % (sum(pair_d) / len(pair_d)))

pair_d2 = [m - t for m, t in zip(medians["matmul"], medians["tdot_TT"])]
print("per-round (matmul - tdot_TT) ms:", ["%+.3f" % x for x in pair_d2])
print("mean: %+.4f ms" % (sum(pair_d2) / len(pair_d2)))
