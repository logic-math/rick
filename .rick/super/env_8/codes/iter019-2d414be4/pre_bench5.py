#!/usr/bin/env python3
"""iter019 builder 预研 V：交互项归因诊断（b 逐出 vs 缺页 vs 其它）。

D1: gemm2 在「无拷贝间隔」vs「有 copy1 间隔」下的耗时差 → b 逐出效应
D2: copy1 写入 fresh c vs 写入 persistent c（不可入格，纯诊断）→ 缺页效应
D3: gemm-only 总时间 vs gemms+allocs 分解
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

REPS = 15
WARM = 3


def med(fn):
    for _ in range(WARM):
        fn()
    ts = []
    for _ in range(REPS):
        t0 = time.perf_counter()
        fn()
        ts.append((time.perf_counter() - t0) * 1000.0)
    return st.median(ts), min(ts)


# 持久缓冲（仅诊断用，不入格）
PERSIST_C = np.empty((N, N), dtype=np.float32)
PERSIST_R1 = np.empty((H, N), dtype=np.float32)


def g2_after_g1():
    r1 = np.einsum('jk,ij->ik', B, A[:H], optimize=True)
    t0 = time.perf_counter()
    r2 = np.einsum('jk,ij->ik', B, A[H:], optimize=True)
    dt = (time.perf_counter() - t0) * 1000.0
    return r2, dt


def g2_after_g1_copy():
    c = np.empty((N, N), dtype=np.float32)
    r1 = np.einsum('jk,ij->ik', B, A[:H], optimize=True)
    c[:H] = r1
    t0 = time.perf_counter()
    r2 = np.einsum('jk,ij->ik', B, A[H:], optimize=True)
    dt = (time.perf_counter() - t0) * 1000.0
    return r2, dt


def g2_after_g1_copypersist():
    # copy1 写入持久 c（无 fresh 页缺页，但同样有 16MB 拷贝流量逐出 b）
    r1 = np.einsum('jk,ij->ik', B, A[:H], optimize=True)
    PERSIST_C[:H] = r1
    t0 = time.perf_counter()
    r2 = np.einsum('jk,ij->ik', B, A[H:], optimize=True)
    dt = (time.perf_counter() - t0) * 1000.0
    return r2, dt


def copy_fresh():
    c = np.empty((N, N), dtype=np.float32)
    c[:H] = PERSIST_R1
    return c


def copy_persist():
    PERSIST_C[:H] = PERSIST_R1
    return None


print("== D1: gemm2 前置状态效应（15 次中位） ==")
for name, fn in [("g2_after_g1(无拷贝)", g2_after_g1),
                 ("g2_after_g1+copy_fresh", g2_after_g1_copy),
                 ("g2_after_g1+copy_persist", g2_after_g1_copypersist)]:
    dts = []
    for _ in range(REPS):
        _, dt = fn()
        dts.append(dt)
    print("  %-28s gemm2 median=%7.3f  min=%7.3f" % (name, st.median(dts), min(dts)))

print("\n== D2: copy1 目标页状态效应 ==")
for name, fn in [("copy→fresh c(含缺页)", copy_fresh),
                 ("copy→persist c(无缺页)", copy_persist)]:
    m, mn = med(fn)
    print("  %-28s median=%7.3f  min=%7.3f" % (name, m, mn))

print("\n== D3: 全分解复测（今日负载态） ==")


def v0():
    c = np.empty((N, N), dtype=np.float32)
    r1 = np.einsum('jk,ij->ik', B, A[:H], optimize=True)
    c[:H] = r1
    r2 = np.einsum('jk,ij->ik', B, A[H:], optimize=True)
    c[H:] = r2
    return c


def v1():
    r1 = np.einsum('jk,ij->ik', B, A[:H], optimize=True)
    r2 = np.einsum('jk,ij->ik', B, A[H:], optimize=True)
    return r2


m0, _ = med(v0)
m1, _ = med(v1)
print("  V0(parent)=%7.3f  V1(gemm_only)=%7.3f  → 装配+交互=%6.3f" % (m0, m1, m0 - m1))
