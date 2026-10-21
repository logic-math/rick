#!/usr/bin/env python3
"""iter019 builder 预研 VI（终验）：canonical-TT + concat-T 装配的最终核验。

V1: 64/2048/嵌套 list/双跑一致性 正确性核验（门协议模拟）
V2: W8（canonical + concat-T）vs V0（父本）新鲜配对 ×4 轮——确定 expected_behavior 量级
"""
import time
import statistics as st

import numpy as np

np.seterr(divide="ignore", over="ignore", invalid="ignore")

N = 2048
H = N // 2


def v0_parent(a, b):
    m = a.shape[0]
    h = m // 2
    c = np.empty((m,) + b.shape[1:], dtype=np.float32)
    r1 = np.einsum('jk,ij->ik', b, a[:h], optimize=True)
    c[:h] = r1
    r2 = np.einsum('jk,ij->ik', b, a[h:], optimize=True)
    c[h:] = r2
    return c


def w8_candidate(a, b):
    a32 = np.asarray(a, dtype=np.float32)
    b32 = np.asarray(b, dtype=np.float32)
    m = a32.shape[0]
    h = m // 2
    r1 = np.einsum('ij,jk->ik', a32[:h], b32, optimize=True)
    r2 = np.einsum('ij,jk->ik', a32[h:], b32, optimize=True)
    return np.concatenate((r1.T, r2.T), axis=1).T


print("== V1 正确性核验（门协议模拟） ==")
for n in (64, 2048):
    for seed in (42, 914037288):
        rng = np.random.RandomState(seed)
        a = rng.rand(n, n).astype(np.float32)
        b = rng.rand(n, n).astype(np.float32)
        ref = np.matmul(a, b)
        c1 = w8_candidate(a, b)
        c2 = w8_candidate(a, b)
        err = float(np.max(np.abs(c1.astype(np.float64) - ref.astype(np.float64))))
        same = np.array_equal(c1, c2)
        print("  n=%-5d seed=%-10d max_err=%-8g double_run_bitwise=%s F_contig=%s" %
              (n, seed, err, same, c1.flags['F_CONTIGUOUS']))
# 嵌套 list 输入
rng = np.random.RandomState(7)
al = [[float(x) for x in row] for row in rng.rand(64, 64)]
bl = [[float(x) for x in row] for row in rng.rand(64, 64)]
ref = np.matmul(np.asarray(al, dtype=np.float32), np.asarray(bl, dtype=np.float32))
err = float(np.max(np.abs(w8_candidate(al, bl).astype(np.float64) - ref.astype(np.float64))))
print("  nested-list max_err=%g" % err)
# 奇数规模
rng = np.random.RandomState(3)
a = rng.rand(65, 67).astype(np.float32)
b = rng.rand(67, 63).astype(np.float32)
ref = np.matmul(a, b)
err = float(np.max(np.abs(w8_candidate(a, b).astype(np.float64) - ref.astype(np.float64))))
print("  odd-shape(65x67@67x63) max_err=%g" % err)

print("\n== V2 W8 vs V0 新鲜配对 ×4 轮（2048） ==")
rng = np.random.RandomState(42)
A = rng.rand(N, N).astype(np.float32)
B = rng.rand(N, N).astype(np.float32)
REF = np.matmul(A, B)
assert np.array_equal(w8_candidate(A, B), REF)
assert np.array_equal(v0_parent(A, B), REF)

REPS = 11


def med(fn):
    for _ in range(2):
        fn()
    ts = []
    for _ in range(REPS):
        t0 = time.perf_counter()
        fn()
        ts.append((time.perf_counter() - t0) * 1000.0)
    return st.median(ts)


for rnd in range(4):
    m0 = med(lambda: v0_parent(A, B))
    m8 = med(lambda: w8_candidate(A, B))
    m0b = med(lambda: v0_parent(A, B))
    print("  round%d  V0=%7.3f/%7.3f  W8=%7.3f  (Δ=%+.3f vs V0 均值)" % (rnd, m0, m0b, m8, m8 - (m0 + m0b) / 2))
