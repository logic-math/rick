"""分解探针：单块 gemm 成本（canonical/swapped/dot）× 拷贝结构成本。

分解 iter014 预研数据：parent 9.10 / swapped_out 9.21 / swapped_concat 8.90 / dot_out 8.49。
问题：为何 swapped 直写反而比 swapped+concat 慢 0.3ms？T-标志 gemm 是否其实更快？
"""
import time

import numpy as np

np.seterr(divide="ignore", over="ignore", invalid="ignore")

rng = np.random.RandomState(42)
n = 2048
a = rng.rand(n, n).astype(np.float32)
b = rng.rand(n, n).astype(np.float32)


def bench(fn, rounds=40):
    fn()
    ts = []
    for _ in range(rounds):
        t0 = time.perf_counter()
        fn()
        ts.append((time.perf_counter() - t0) * 1000)
    ts.sort()
    return ts[rounds // 2]


s, e = 0, 512
a_blk = a[s:e]

# ---- 单块计算（含各自内部拷贝）----
t_canon_blk = bench(lambda: np.einsum('ij,jk->ik', a_blk, b, optimize=True))
t_swap_blk = bench(lambda: np.einsum('jk,ij->ik', b, a_blk, optimize=True))
t_dot_blk = bench(lambda: np.dot(a_blk, b))
t_dot_out_blk = bench(lambda: np.dot(a_blk, b, out=np.empty((512, n), dtype=np.float32)))
print("单块(512x2048x2048) median ms:")
print("  einsum canonical (T-gemm + F块 + 内部转置拷贝): %.3f" % t_canon_blk)
print("  einsum swapped   (N-gemm + C块, 无内部拷贝)   : %.3f" % t_swap_blk)
print("  np.dot           (N-gemm + C块)               : %.3f" % t_dot_blk)
print("  np.dot out=预热缓冲 (N-gemm 直写)             : %.3f" % t_dot_out_blk)

# ---- 拷贝结构（给定 4 个块结果）----
blocks_F = [np.einsum('ij,jk->ik', a[i * n // 4:(i + 1) * n // 4], b, optimize=True) for i in range(4)]
blocks_C = [np.einsum('jk,ij->ik', b, a[i * n // 4:(i + 1) * n // 4], optimize=True) for i in range(4)]
assert all(x.flags['F_CONTIGUOUS'] for x in blocks_F)
assert all(x.flags['C_CONTIGUOUS'] for x in blocks_C)

cbuf = np.empty((n, n), dtype=np.float32)


def concat_F():
    return np.concatenate(blocks_F, axis=0)


def concat_C():
    return np.concatenate(blocks_C, axis=0)


def direct_C():
    for i in range(4):
        cbuf[i * n // 4:(i + 1) * n // 4] = blocks_C[i]
    return cbuf


def direct_F():
    for i in range(4):
        cbuf[i * n // 4:(i + 1) * n // 4] = blocks_F[i]
    return cbuf


def copyto_C():
    for i in range(4):
        np.copyto(cbuf[i * n // 4:(i + 1) * n // 4], blocks_C[i])
    return cbuf


def alloc_direct_C():
    c = np.empty((n, n), dtype=np.float32)
    for i in range(4):
        c[i * n // 4:(i + 1) * n // 4] = blocks_C[i]
    return c


print("\n拷贝结构 median ms (16MB 输出):")
print("  concatenate(F-块)  : %.3f" % bench(concat_F))
print("  concatenate(C-块)  : %.3f" % bench(concat_C))
print("  直写 c[s:e]=C-块   : %.3f" % bench(direct_C))
print("  直写 c[s:e]=F-块   : %.3f" % bench(direct_F))
print("  copyto C-块        : %.3f" % bench(copyto_C))
print("  alloc+直写 C-块    : %.3f" % bench(alloc_direct_C))

# ---- 组合预估 ----
print("\n组合预估 (4×单块 + 拷贝):")
print("  parent   = 4*canon + concat_F = %.3f" % (4 * t_canon_blk + bench(concat_F)))
print("  swap_cat = 4*swap  + concat_C = %.3f" % (4 * t_swap_blk + bench(concat_C)))
print("  swap_out = 4*swap  + direct_C = %.3f" % (4 * t_swap_blk + bench(direct_C)))
print("  dot_out  = 4*dot_out          = %.3f" % (4 * t_dot_out_blk))
