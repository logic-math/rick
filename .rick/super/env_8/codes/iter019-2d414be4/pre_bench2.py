#!/usr/bin/env python3
"""iter019 builder 预研 II：机制核验 + 剩余微变体裁决。

C1 机制核验：swapped/canonical 两种拼写的输出连续性 + tensordot 参数形态
C2 拼写 A/B（halves，干净配对中位）
C4 装配变体：交错拷贝 vs 分块交错（b 驻留管理假设）
C5 显式 path vs optimize=True（源码预测零收益——实测确认）
C6 concatenate(out=c) 装配 vs 切片赋值
C7 gemm 顺序反转（下块先算）——malloc 地址复用/页温效应探针
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


def med(fn, *args):
    for _ in range(WARM):
        fn(*args)
    ts = []
    for _ in range(REPS):
        t0 = time.perf_counter()
        fn(*args)
        ts.append((time.perf_counter() - t0) * 1000.0)
    return st.median(ts), min(ts)


# ---------- C1 机制核验 ----------
print("== C1 机制核验 ==")
r_swap = np.einsum('jk,ij->ik', B, A[:H], optimize=True)
r_canon = np.einsum('ij,jk->ik', A[:H], B, optimize=True)
print("swapped  r.flags C=%s F=%s  owndata=%s" % (r_swap.flags['C_CONTIGUOUS'], r_swap.flags['F_CONTIGUOUS'], r_swap.flags['OWNDATA']))
print("canonical r.flags C=%s F=%s  owndata=%s" % (r_canon.flags['C_CONTIGUOUS'], r_canon.flags['F_CONTIGUOUS'], r_canon.flags['OWNDATA']))
print("swapped==canonical bitwise:", np.array_equal(r_swap, r_canon))
print("swapped==ref bitwise:", np.array_equal(r_swap, REF[:H]))
# list-form 拼写（跳过字符串解析）
r_list = np.einsum(B, [0, 1], A[:H], [2, 0], [2, 1], optimize=True)
print("list-form ok, bitwise==swapped:", np.array_equal(r_list, r_swap))

# ---------- C2 拼写 A/B ----------
print("\n== C2 拼写 A/B（halves，配对中位 ×3 轮）==")


def two_halves_swapped(a, b):
    h = a.shape[0] // 2
    r1 = np.einsum('jk,ij->ik', b, a[:h], optimize=True)
    r2 = np.einsum('jk,ij->ik', b, a[h:], optimize=True)
    return r2


def two_halves_canonical(a, b):
    h = a.shape[0] // 2
    r1 = np.einsum('ij,jk->ik', a[:h], b, optimize=True)
    r2 = np.einsum('ij,jk->ik', a[h:], b, optimize=True)
    return r2


def two_halves_listform(a, b):
    h = a.shape[0] // 2
    r1 = np.einsum(b, [0, 1], a[:h], [2, 0], [2, 1], optimize=True)
    r2 = np.einsum(b, [0, 1], a[h:], [2, 0], [2, 1], optimize=True)
    return r2


for rnd in range(3):
    m_s, _ = med(two_halves_swapped, A, B)
    m_c, _ = med(two_halves_canonical, A, B)
    m_l, _ = med(two_halves_listform, A, B)
    print("round%d  swapped=%7.3f  canonical=%7.3f  listform=%7.3f" % (rnd, m_s, m_c, m_l))

# ---------- C4/C6/C7 完整装配变体 ----------
print("\n== 完整装配变体（gemm+装配端到端）==")


def v0_parent(a, b):
    m = a.shape[0]
    h = m // 2
    c = np.empty((m,) + b.shape[1:], dtype=np.float32)
    r1 = np.einsum('jk,ij->ik', b, a[:h], optimize=True)
    c[:h] = r1
    r2 = np.einsum('jk,ij->ik', b, a[h:], optimize=True)
    c[h:] = r2
    return c


def v_chunk(a, b):
    # 分块交错：gemm1 → 拷 r1 前半 → gemm2 → 拷 r1 后半 + r2
    # 假设：copy 前半只逐出一半 b，gemm2 的 b 再读省一半；代价是 r1 后半冷拷
    m = a.shape[0]
    h = m // 2
    q = h // 2
    c = np.empty((m,) + b.shape[1:], dtype=np.float32)
    r1 = np.einsum('jk,ij->ik', b, a[:h], optimize=True)
    c[:q] = r1[:q]
    r2 = np.einsum('jk,ij->ik', b, a[h:], optimize=True)
    c[q:h] = r1[q:]
    c[h:] = r2
    return c


def v_concat_out(a, b):
    m = a.shape[0]
    h = m // 2
    c = np.empty((m,) + b.shape[1:], dtype=np.float32)
    r1 = np.einsum('jk,ij->ik', b, a[:h], optimize=True)
    r2 = np.einsum('jk,ij->ik', b, a[h:], optimize=True)
    np.concatenate((r1, r2), axis=0, out=c)
    return c


def v_rev(a, b):
    # 下块先算（malloc 地址序 / 页温探针）
    m = a.shape[0]
    h = m // 2
    c = np.empty((m,) + b.shape[1:], dtype=np.float32)
    r2 = np.einsum('jk,ij->ik', b, a[h:], optimize=True)
    c[h:] = r2
    r1 = np.einsum('jk,ij->ik', b, a[:h], optimize=True)
    c[:h] = r1
    return c


def v_listfull(a, b):
    # list-form 拼写 + 父本装配
    m = a.shape[0]
    h = m // 2
    c = np.empty((m,) + b.shape[1:], dtype=np.float32)
    r1 = np.einsum(b, [0, 1], a[:h], [2, 0], [2, 1], optimize=True)
    c[:h] = r1
    r2 = np.einsum(b, [0, 1], a[h:], [2, 0], [2, 1], optimize=True)
    c[h:] = r2
    return c


VARIANTS = [
    ("V0_parent", v0_parent),
    ("V_chunk_interleave", v_chunk),
    ("V_concat_out", v_concat_out),
    ("V_reversed", v_rev),
    ("V_listform_full", v_listfull),
]
for name, fn in VARIANTS:
    err = float(np.max(np.abs(fn(A, B).astype(np.float64) - REF.astype(np.float64))))
    assert err == 0.0, (name, err)

for rnd in range(3):
    print("-- round %d --" % rnd)
    for name, fn in VARIANTS:
        m, mn = med(fn, A, B)
        print("  %-20s median=%7.3f  min=%7.3f" % (name, m, mn))
