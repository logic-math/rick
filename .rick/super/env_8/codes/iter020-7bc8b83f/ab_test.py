#!/usr/bin/env python3
"""Rigorous A/B: parent (plain np.dot) vs out= preallocation variant @2048.

P1: 40 interleaved single-shot rounds -> per-variant median + win rate.
P2: bench-protocol simulation (warmup + k=3, median), 12 alternating repeats
    per variant + ref matmul -> distribution of medians + would-be score.
"""
import numpy as np
from time import perf_counter

np.seterr(divide="ignore", over="ignore", invalid="ignore")


def cand_parent(a, b):
    a32 = np.asarray(a, dtype=np.float32)
    b32 = np.asarray(b, dtype=np.float32)
    return np.dot(a32, b32)


def cand_out(a, b):
    a32 = np.asarray(a, dtype=np.float32)
    b32 = np.asarray(b, dtype=np.float32)
    if a32.ndim != 2:
        return np.dot(a32, b32)
    c = np.empty((a32.shape[0],) + b32.shape[1:], dtype=np.float32)
    return np.dot(a32, b32, out=c)


rng = np.random.RandomState(42)
a = rng.rand(2048, 2048).astype(np.float32)
b = rng.rand(2048, 2048).astype(np.float32)

# correctness spot-check (multi-size/seed + 1-D b + nested list)
ref = np.matmul(a, b)
assert np.array_equal(cand_parent(a, b), ref)
assert np.array_equal(cand_out(a, b), ref)
for n in (64, 2047):
    r2 = np.random.RandomState(n)
    aa, bb = r2.rand(n, n).astype(np.float32), r2.rand(n, n).astype(np.float32)
    assert np.array_equal(cand_out(aa, bb), np.matmul(aa, bb))
v = np.ones(2048, dtype=np.float32)
assert np.array_equal(cand_out(a, v), np.matmul(a, v))  # 1-D b gemv
al = [[1.0, 2.0], [3.0, 4.0]]
bl = [[5.0, 6.0], [7.0, 8.0]]
assert np.array_equal(cand_out(al, bl), np.matmul(np.array(al), np.array(bl)))
print("correctness spot-checks: all bit-identical OK")

# ---- P1: interleaved single shots ----
cand_parent(a, b); cand_out(a, b)
pa, pb = [], []
for _ in range(40):
    t0 = perf_counter(); cand_parent(a, b); pa.append((perf_counter()-t0)*1e3)
    t0 = perf_counter(); cand_out(a, b);   pb.append((perf_counter()-t0)*1e3)
pa_s, pb_s = sorted(pa), sorted(pb)
wins = sum(1 for x, y in zip(pa, pb) if y < x)
print("P1 parent : median=%.3f mean=%.3f min=%.3f max=%.3f" % (
    pa_s[20], sum(pa)/40, pa_s[0], pa_s[-1]))
print("P1 out=   : median=%.3f mean=%.3f min=%.3f max=%.3f" % (
    pb_s[20], sum(pb)/40, pb_s[0], pb_s[-1]))
print("P1 out= wins %d/40 rounds (paired)" % wins)

# ---- P2: bench protocol (warmup + k=3 median), alternating, 12 repeats ----
def bench_sim(fn):
    fn(a, b)
    ts = []
    for _ in range(3):
        t0 = perf_counter(); fn(a, b); ts.append((perf_counter()-t0)*1e3)
    ts.sort()
    return ts[1]

mA, mB, mR = [], [], []
for i in range(12):
    mR.append(bench_sim(np.matmul))
    if i % 2 == 0:
        mA.append(bench_sim(cand_parent)); mB.append(bench_sim(cand_out))
    else:
        mB.append(bench_sim(cand_out)); mA.append(bench_sim(cand_parent))
for name, ms in (("parent", mA), ("out=", mB), ("ref matmul", mR)):
    ms.sort()
    print("P2 %-11s medians: med=%.3f min=%.3f max=%.3f spread=%.3f" % (
        name, ms[6], ms[0], ms[-1], ms[-1]-ms[0]))
medA, medB, medR = sorted(mA)[6], sorted(mB)[6], sorted(mR)[6]
print("P2 would-be score: parent=%.4f out=%.4f (ref/cand, central estimates)" % (
    medR/medA, medR/medB))
