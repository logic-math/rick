#!/usr/bin/env python3
"""iter019 builder 预研微基准：分解 einsum|m_halves 父本（iter013）的剩余开销。

分解目标（@2048 f32，稳态中位）：
  V0 父本形态：预分配 c + 交错趁热拷贝
  V1 纯两半 gemm（einsum swapped，无装配——只算不拼）
  V2 纯装配（预分配 + 两块拷贝，r1/r2 预先用同结构数组代替）
  V3 双 gemm 后 concatenate（iter009 形态，对照）
  V4 np.dot halves + out= 直写（iter001 形态，跨格参照 7.42）
  V5 np.matmul halves + out= 直写（跨格参照）
  V6 单次整阵 einsum swapped（iter012 形态，参照 ~7.0）
  V7 预分配 + 双 gemm 后双拷贝（冷拷贝序，对照交错）
  V8 V0 但 c 用 np.zeros（calloc 路径对照）
"""
import time
import statistics as st

import numpy as np

np.seterr(divide="ignore", over="ignore", invalid="ignore")

N = 2048
REPS = 9
WARM = 2

rng = np.random.RandomState(42)
A = rng.rand(N, N).astype(np.float32)
B = rng.rand(N, N).astype(np.float32)
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


def v1_gemm_only(a, b):
    m = a.shape[0]
    h = m // 2
    r1 = np.einsum('jk,ij->ik', b, a[:h], optimize=True)
    r2 = np.einsum('jk,ij->ik', b, a[h:], optimize=True)
    return r2  # 只算不拼（r1 计算量照付，返回避免被优化掉）


def v2_assembly_only(a, b):
    # r1/r2 用一次性预生成的同结构数组（模拟热块不现实，测纯拷贝+分配成本）
    r1 = _R1
    r2 = _R2
    m = a.shape[0]
    h = m // 2
    c = np.empty((m,) + b.shape[1:], dtype=np.float32)
    c[:h] = r1
    c[h:] = r2
    return c


def v3_concat(a, b):
    m = a.shape[0]
    h = m // 2
    r1 = np.einsum('jk,ij->ik', b, a[:h], optimize=True)
    r2 = np.einsum('jk,ij->ik', b, a[h:], optimize=True)
    return np.concatenate((r1, r2), axis=0)


def v4_dot_out(a, b):
    m = a.shape[0]
    h = m // 2
    c = np.empty((m,) + b.shape[1:], dtype=np.float32)
    np.dot(a[:h], b, out=c[:h])
    np.dot(a[h:], b, out=c[h:])
    return c


def v5_matmul_out(a, b):
    m = a.shape[0]
    h = m // 2
    c = np.empty((m,) + b.shape[1:], dtype=np.float32)
    np.matmul(a[:h], b, out=c[:h])
    np.matmul(a[h:], b, out=c[h:])
    return c


def v6_full_einsum(a, b):
    return np.einsum('jk,ij->ik', b, a, optimize=True)


def v7_coldcopy(a, b):
    m = a.shape[0]
    h = m // 2
    c = np.empty((m,) + b.shape[1:], dtype=np.float32)
    r1 = np.einsum('jk,ij->ik', b, a[:h], optimize=True)
    r2 = np.einsum('jk,ij->ik', b, a[h:], optimize=True)
    c[:h] = r1
    c[h:] = r2
    return c


def v8_zeros(a, b):
    m = a.shape[0]
    h = m // 2
    c = np.zeros((m,) + b.shape[1:], dtype=np.float32)
    r1 = np.einsum('jk,ij->ik', b, a[:h], optimize=True)
    c[:h] = r1
    r2 = np.einsum('jk,ij->ik', b, a[h:], optimize=True)
    c[h:] = r2
    return c


# 预生成 r1/r2 同结构数组（供 V2）
_R1 = np.ascontiguousarray(A[:H] @ B)
_R2 = np.ascontiguousarray(A[H:] @ B)

VARIANTS = [
    ("V0_parent_interleaved", v0_parent),
    ("V1_gemm_only", v1_gemm_only),
    ("V2_assembly_only", v2_assembly_only),
    ("V3_concat", v3_concat),
    ("V4_dot_out", v4_dot_out),
    ("V5_matmul_out", v5_matmul_out),
    ("V6_full_einsum", v6_full_einsum),
    ("V7_coldcopy_after", v7_coldcopy),
    ("V8_zeros_c", v8_zeros),
]


def bench(fn):
    for _ in range(WARM):
        fn(A, B)
    ts = []
    for _ in range(REPS):
        t0 = time.perf_counter()
        fn(A, B)
        ts.append((time.perf_counter() - t0) * 1000.0)
    return st.median(ts), min(ts), ts


# 正确性抽检（V1/V2 是分解探针除外）
ref = np.matmul(A, B)
for name, fn in VARIANTS:
    if name in ("V1_gemm_only", "V2_assembly_only"):
        continue
    err = float(np.max(np.abs(fn(A, B).astype(np.float64) - ref.astype(np.float64))))
    assert err == 0.0, (name, err)

results = {}
for name, fn in VARIANTS:
    med, mn, ts = bench(fn)
    results[name] = (med, mn)
    print("%-26s median=%7.3f  min=%7.3f  spread=%.3f" % (name, med, mn, max(ts) - min(ts)))

# 二轮：交叉顺序复测关键对（排除块间顺序漂移）
print("\n-- 二轮（顺序反转，测漂移）--")
for name, fn in reversed(VARIANTS):
    med, mn, ts = bench(fn)
    print("%-26s median=%7.3f  min=%7.3f" % (name, med, mn))
