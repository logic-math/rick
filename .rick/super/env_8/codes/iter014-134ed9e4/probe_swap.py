"""预研探针：einsum 操作数顺序对派发层的影响（einsum_path|m_quarters 格内）。

假设（源码阅读 numpy 2.0.2 einsumfunc）：
- canonical: einsum('ij,jk->ik', A_blk, B, optimize=True)
  → tensordot(B, A_blk, axes=([0],[1])) → dot(B.T, A_blk) = T-标志 gemm
  → 结果 F-布局 + c_einsum('ki->ik') 额外 4MB 转置拷贝
- swapped:   einsum('jk,ij->ik', B, A_blk, optimize=True)
  → tensordot(A_blk, B, axes=([1],[0])) → dot(A_blk, B) = N-标志 gemm
  → 结果 C-布局，无 c_einsum 步骤

验证：flags / bitwise 正确性 / 配对计时（parent 结构 vs swapped 直写 vs swapped concat
vs np_dot out= 直写参照）。
"""
import time

import numpy as np

np.seterr(divide="ignore", over="ignore", invalid="ignore")

rng = np.random.RandomState(42)
n = 2048
a = rng.rand(n, n).astype(np.float32)
b = rng.rand(n, n).astype(np.float32)

# ---- 1. 单块：flags 与正确性 ----
blk_canon = np.einsum('ij,jk->ik', a[:512], b, optimize=True)
blk_swap = np.einsum('jk,ij->ik', b, a[:512], optimize=True)
print("canon flags C/F:", blk_canon.flags['C_CONTIGUOUS'], blk_canon.flags['F_CONTIGUOUS'])
print("swap  flags C/F:", blk_swap.flags['C_CONTIGUOUS'], blk_swap.flags['F_CONTIGUOUS'])
ref_blk = np.matmul(a[:512], b)
print("canon max_err vs matmul:", float(np.max(np.abs(blk_canon.astype(np.float64) - ref_blk.astype(np.float64)))))
print("swap  max_err vs matmul:", float(np.max(np.abs(blk_swap.astype(np.float64) - ref_blk.astype(np.float64)))))
print("swap  bitwise == np.dot:", np.array_equal(blk_swap, np.dot(a[:512], b)))
print("swap  bitwise == matmul:", np.array_equal(blk_swap, ref_blk))


# ---- 2. 整体结构：四种实现配对计时 ----
def parent_impl():  # iter007 现状：canonical + concatenate
    edges = [(i * n // 4, (i + 1) * n // 4) for i in range(4)]
    blocks = []
    for s, e in edges:
        if e > s:
            blocks.append(np.einsum('ij,jk->ik', a[s:e], b, optimize=True))
    return np.concatenate(blocks, axis=0)


def swapped_out():  # swapped + 预分配直写（无 concatenate）
    edges = [(i * n // 4, (i + 1) * n // 4) for i in range(4)]
    c = np.empty((n, n), dtype=np.float32)
    for s, e in edges:
        if e > s:
            c[s:e] = np.einsum('jk,ij->ik', b, a[s:e], optimize=True)
    return c


def swapped_concat():  # swapped + concatenate（对照：拼接成本 vs 直写）
    edges = [(i * n // 4, (i + 1) * n // 4) for i in range(4)]
    blocks = []
    for s, e in edges:
        if e > s:
            blocks.append(np.einsum('jk,ij->ik', b, a[s:e], optimize=True))
    return np.concatenate(blocks, axis=0)


def dot_out():  # iter010 np_dot 参照（性能锚，非本格实现）
    edges = [(i * n // 4, (i + 1) * n // 4) for i in range(4)]
    c = np.empty((n, n), dtype=np.float32)
    for s, e in edges:
        if e > s:
            np.dot(a[s:e], b, out=c[s:e])
    return c


impls = [parent_impl, swapped_out, swapped_concat, dot_out]
# 正确性
ref_full = np.matmul(a, b)
for f in impls:
    r = f()
    print(f.__name__, "bitwise==matmul:", np.array_equal(r, ref_full),
          "max_err:", float(np.max(np.abs(r.astype(np.float64) - ref_full.astype(np.float64)))))

# 预热
for f in impls:
    f()
np.matmul(a, b)

rounds = 21
times = {f.__name__: [] for f in impls}
ref_times = []
for _ in range(rounds):
    t0 = time.perf_counter()
    np.matmul(a, b)
    ref_times.append((time.perf_counter() - t0) * 1000)
    for f in impls:
        t0 = time.perf_counter()
        f()
        times[f.__name__].append((time.perf_counter() - t0) * 1000)

ref_med = sorted(ref_times)[rounds // 2]
print("\nref np.matmul median: %.3f ms" % ref_med)
for f in impls:
    ts = sorted(times[f.__name__])
    med = ts[rounds // 2]
    print("%-16s median %.3f ms  min %.3f  max %.3f  (score~%.3f)" % (
        f.__name__, med, ts[0], ts[-1], ref_med / med))

# 配对差（每轮同序对比，消除负载漂移）
pd = [times['parent_impl'][i] - times['swapped_out'][i] for i in range(rounds)]
print("\npaired parent - swapped_out: median %.3f ms, pos %d/%d" % (
    sorted(pd)[rounds // 2], sum(1 for x in pd if x > 0), rounds))
