#!/usr/bin/env python3
"""iter019 builder 预研 IV：拷贝次序/分配次序裁决（交互项管理）。

模型：V0 的交互项 ~0.2ms ≈ copy1 的 16MB 流量把 b 逐出 L2 → gemm2 冷读 b。
X(f) 族：gemm1 → 拷 r1 前 f 部分 → gemm2 → copy2（热）→ 拷 r1 余下（冷）
  f=0   → X0: gemm1, gemm2, copy2(热), copy1(冷)
  f=0.5 → X50
  f=1   → V0 父本（gemm1, copy1(热), gemm2, copy2(热)）
模型预测全族 ≈ wash（±0.08ms）；实测裁决。
另测：晚分配 c（地址循环改变）、X0 与 V0 交替配对（抗漂移）。
"""
import time
import statistics as st

import numpy as np

np.seterr(divide="ignore", over="ignore", invalid="ignore")

N = 2048
H = N // 2
rng = np.random.RandomState(42)
A = rng.rand(N, N).astype(np.float32)
B = rng.rand(N, N).astype(np.float32)
REF = np.matmul(A, B)

REPS = 11
WARM = 2


def med(fn):
    for _ in range(WARM):
        fn()
    ts = []
    for _ in range(REPS):
        t0 = time.perf_counter()
        fn()
        ts.append((time.perf_counter() - t0) * 1000.0)
    return st.median(ts), min(ts)


def v0():
    c = np.empty((N, N), dtype=np.float32)
    r1 = np.einsum('jk,ij->ik', B, A[:H], optimize=True)
    c[:H] = r1
    r2 = np.einsum('jk,ij->ik', B, A[H:], optimize=True)
    c[H:] = r2
    return c


def x0():
    # gemm1 → gemm2 → copy2(热) → copy1(冷)
    c = np.empty((N, N), dtype=np.float32)
    r1 = np.einsum('jk,ij->ik', B, A[:H], optimize=True)
    r2 = np.einsum('jk,ij->ik', B, A[H:], optimize=True)
    c[H:] = r2
    c[:H] = r1
    return c


def x50():
    q = H // 2
    c = np.empty((N, N), dtype=np.float32)
    r1 = np.einsum('jk,ij->ik', B, A[:H], optimize=True)
    c[:q] = r1[:q]
    r2 = np.einsum('jk,ij->ik', B, A[H:], optimize=True)
    c[H:] = r2
    c[q:H] = r1[q:]
    return c


def x0_late_c():
    # 同 X0 但 c 晚分配（malloc 地址循环改变）
    r1 = np.einsum('jk,ij->ik', B, A[:H], optimize=True)
    r2 = np.einsum('jk,ij->ik', B, A[H:], optimize=True)
    c = np.empty((N, N), dtype=np.float32)
    c[H:] = r2
    c[:H] = r1
    return c


VARIANTS = [
    ("V0_parent", v0),
    ("X0_gg_c2c1", x0),
    ("X50", x50),
    ("X0_lateC", x0_late_c),
]
for name, fn in VARIANTS:
    r = fn()
    assert np.array_equal(r, REF), name

# 配对交替 4 轮（V0 与各变体紧邻配对，抗慢漂移）
for rnd in range(4):
    print("-- round %d --" % rnd)
    m0, _ = med(v0)
    for name, fn in VARIANTS[1:]:
        m, mn = med(fn)
        d = m - m0
        print("  V0=%7.3f  %-12s=%7.3f (Δ=%+.3f, min=%7.3f)" % (m0, name, m, d, mn))
