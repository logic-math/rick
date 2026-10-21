#!/usr/bin/env python3
"""Rigorous A/B: parent iter005 (plain np.dot, C-layer internal alloc) vs
iter021 candidate (module-level persistent out buffer) @2048.

P1: 40 interleaved single-shot rounds -> per-variant median + paired win rate.
P2: bench-protocol simulation (warmup 1 + k=3, median), alternating repeats
    per variant + ref matmul -> distribution of medians + would-be score.

Candidate arm loads the actual submitted solution.py (module-level _OUT included);
parent arm re-implements iter005 inline (a32/b32 asarray + np.dot).
"""
import importlib.util
import os

import numpy as np
from time import perf_counter

np.seterr(divide="ignore", over="ignore", invalid="ignore")

spec = importlib.util.spec_from_file_location("cand21", "solution.py")
cand = importlib.util.module_from_spec(spec)
spec.loader.exec_module(cand)
cand_persist = cand.matmul


def cand_parent(a, b):
    a32 = np.asarray(a, dtype=np.float32)
    b32 = np.asarray(b, dtype=np.float32)
    return np.dot(a32, b32)


rng = np.random.RandomState(42)
a = rng.rand(2048, 2048).astype(np.float32)
b = rng.rand(2048, 2048).astype(np.float32)

# ---- correctness spot-checks (bit-identical + boundary semantics) ----
ref = np.matmul(a, b)
assert np.array_equal(cand_parent(a, b), ref)
c1 = cand_persist(a, b).copy()          # copy before next call overwrites buffer
assert np.array_equal(c1, ref)
assert np.array_equal(cand_persist(a, b), ref)  # buffer reuse not self-destructive
for n in (64, 2047):
    r2 = np.random.RandomState(n)
    aa, bb = r2.rand(n, n).astype(np.float32), r2.rand(n, n).astype(np.float32)
    rr = np.matmul(aa, bb)
    assert np.array_equal(cand_parent(aa, bb), rr)
    assert np.array_equal(cand_persist(aa, bb), rr)
v = np.ones(2048, dtype=np.float32)
assert np.array_equal(cand_persist(a, v), np.matmul(a, v))  # 1-D b gemv (fallback)
al = [[1.0, 2.0], [3.0, 4.0]]
bl = [[5.0, 6.0], [7.0, 8.0]]
assert np.array_equal(cand_persist(al, bl), np.matmul(np.array(al), np.array(bl)))
a0 = np.zeros((0, 8), dtype=np.float32)
assert np.array_equal(cand_persist(a0, b[:8, :4]), np.matmul(a0, b[:8, :4]))  # 0xk
bz = np.zeros((2048, 0), dtype=np.float32)
assert np.array_equal(cand_persist(a, bz), np.matmul(a, bz))                 # kx0
print("correctness spot-checks: all bit-identical OK")
print("loadavg(1m) at start: %.2f" % os.getloadavg()[0])

# ---- P1: interleaved single shots, 40 rounds (parent first each round) ----
cand_parent(a, b); cand_persist(a, b)  # warmup both arms
pa, pb = [], []
for _ in range(40):
    t0 = perf_counter(); cand_parent(a, b); pa.append((perf_counter() - t0) * 1e3)
    t0 = perf_counter(); cand_persist(a, b); pb.append((perf_counter() - t0) * 1e3)
pa_s, pb_s = sorted(pa), sorted(pb)
wins = sum(1 for x, y in zip(pa, pb) if y < x)
print("P1 parent  : median=%.3f mean=%.3f min=%.3f max=%.3f" % (
    pa_s[20], sum(pa) / 40, pa_s[0], pa_s[-1]))
print("P1 persist : median=%.3f mean=%.3f min=%.3f max=%.3f" % (
    pb_s[20], sum(pb) / 40, pb_s[0], pb_s[-1]))
print("P1 persist wins %d/40 rounds (paired); delta median = %+.3f ms" % (
    wins, pb_s[20] - pa_s[20]))

# ---- P2: bench protocol (warmup 1 + k=3 median), alternating, 12 repeats ----
def bench_sim(fn):
    fn(a, b)
    ts = []
    for _ in range(3):
        t0 = perf_counter(); fn(a, b); ts.append((perf_counter() - t0) * 1e3)
    ts.sort()
    return ts[1]


mA, mB, mR = [], [], []
for i in range(12):
    mR.append(bench_sim(np.matmul))
    if i % 2 == 0:
        mA.append(bench_sim(cand_parent)); mB.append(bench_sim(cand_persist))
    else:
        mB.append(bench_sim(cand_persist)); mA.append(bench_sim(cand_parent))
for name, ms in (("parent", mA), ("persist", mB), ("ref matmul", mR)):
    ms.sort()
    print("P2 %-11s medians: med=%.3f min=%.3f max=%.3f spread=%.3f" % (
        name, ms[6], ms[0], ms[-1], ms[-1] - ms[0]))
medA, medB, medR = sorted(mA)[6], sorted(mB)[6], sorted(mR)[6]
print("P2 would-be score: parent=%.4f persist=%.4f (ref/cand, central estimates)" % (
    medR / medA, medR / medB))
print("loadavg(1m) at end: %.2f" % os.getloadavg()[0])
