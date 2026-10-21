#!/usr/bin/env python3
"""iter019 builder 预研 III：canonical-TT halves + F 序装配 全管线裁决。

背景（pre_bench2 实测）：
- canonical 'ij,jk->ik' (a_blk, b) halves gemm-only = 7.35ms（TT gemm，输出 F 连续视图）
- swapped 'jk,ij->ik' (b, a_blk) halves gemm-only = 7.57ms（CC gemm，输出 C 连续）
- 今日负载态与 iter013 预研（swapped 7.44 / canonical 7.58）相反——T/C 漂移，门禁裁决

W 变体（端到端含装配，全部逐位一致）：
  W0 父本：swapped + C 序预分配 + 交错拷贝（对照）
  W2 canonical + c F 序预分配 + 交错列段拷贝 + 返回 F 序
  W2b canonical + d=(N,M) C 序预分配 + d[:, :h]=r1.T 行段拷贝 + 返回 d.T
  W8 canonical + np.concatenate((r1.T, r2.T), axis=1) + 返回 .T
  W5 canonical + C 序 c + c[:h]=r1（散布拷贝，预期死路——验证）
  W6 canonical + einsum(order='C') 物化 + C 序切片赋值（双拷贝，验证）
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


def w0_parent():
    c = np.empty((N, N), dtype=np.float32)
    r1 = np.einsum('jk,ij->ik', B, A[:H], optimize=True)
    c[:H] = r1
    r2 = np.einsum('jk,ij->ik', B, A[H:], optimize=True)
    c[H:] = r2
    return c


def w2_forder():
    c = np.empty((N, N), dtype=np.float32, order='F')
    r1 = np.einsum('ij,jk->ik', A[:H], B, optimize=True)
    c[:H] = r1
    r2 = np.einsum('ij,jk->ik', A[H:], B, optimize=True)
    c[H:] = r2
    return c


def w2b_transposed():
    d = np.empty((N, N), dtype=np.float32)  # d = c.T（C 序）
    r1 = np.einsum('ij,jk->ik', A[:H], B, optimize=True)
    d[:, :H] = r1.T
    r2 = np.einsum('ij,jk->ik', A[H:], B, optimize=True)
    d[:, H:] = r2.T
    return d.T


def w8_concat():
    r1 = np.einsum('ij,jk->ik', A[:H], B, optimize=True)
    r2 = np.einsum('ij,jk->ik', A[H:], B, optimize=True)
    return np.concatenate((r1.T, r2.T), axis=1).T


def w5_scatter():
    c = np.empty((N, N), dtype=np.float32)
    r1 = np.einsum('ij,jk->ik', A[:H], B, optimize=True)
    c[:H] = r1
    r2 = np.einsum('ij,jk->ik', A[H:], B, optimize=True)
    c[H:] = r2
    return c


def w6_orderC():
    c = np.empty((N, N), dtype=np.float32)
    r1 = np.einsum('ij,jk->ik', A[:H], B, optimize=True, order='C')
    c[:H] = r1
    r2 = np.einsum('ij,jk->ik', A[H:], B, optimize=True, order='C')
    c[H:] = r2
    return c


VARIANTS = [
    ("W0_parent_swapped", w0_parent),
    ("W2_canon_Forder", w2_forder),
    ("W2b_canon_dT", w2b_transposed),
    ("W8_canon_concatT", w8_concat),
    ("W5_canon_scatter", w5_scatter),
    ("W6_canon_orderC", w6_orderC),
]

# 正确性（逐位）+ 返回序标记
for name, fn in VARIANTS:
    r = fn()
    err = float(np.max(np.abs(r.astype(np.float64) - REF.astype(np.float64))))
    print("%-22s max_err=%g  C=%s F=%s" % (name, err, r.flags['C_CONTIGUOUS'], r.flags['F_CONTIGUOUS']))
    assert err == 0.0, name

# A/B 交替 3 轮（抗漂移）
print()
order = [0, 1, 2, 3, 4, 5]
for rnd in range(3):
    print("-- round %d --" % rnd)
    for i in order:
        name, fn = VARIANTS[i]
        m, mn = med(fn)
        print("  %-22s median=%7.3f  min=%7.3f" % (name, m, mn))
    order = order[::-1]

# 装配单项成本：canonical gemm 后单独计时拷贝（用热 r1/r2）
print("\n== 装配单项（canonical 热块） ==")
r1 = np.einsum('ij,jk->ik', A[:H], B, optimize=True)
r2 = np.einsum('ij,jk->ik', A[H:], B, optimize=True)


def asm_f():
    c = np.empty((N, N), dtype=np.float32, order='F')
    c[:H] = r1
    c[H:] = r2
    return c


def asm_c():
    c = np.empty((N, N), dtype=np.float32)
    c[:H] = r1
    c[H:] = r2
    return c


def asm_d():
    d = np.empty((N, N), dtype=np.float32)
    d[:, :H] = r1.T
    d[:, H:] = r2.T
    return d


def asm_concat():
    return np.concatenate((r1.T, r2.T), axis=1)


for name, fn in [("asm_F", asm_f), ("asm_C_scatter", asm_c), ("asm_dT", asm_d), ("asm_concatT", asm_concat)]:
    m, mn = med(fn)
    print("  %-16s median=%7.3f  min=%7.3f" % (name, m, mn))
