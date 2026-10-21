"""终裁探针：全部结构变体头对头（5 rep × 15 轮交错，控制负载漂移）。

变体（全部 einsum optimize=True 派发 + M 四分，格内合法）：
  P0 parent    : 'ij,jk->ik' + concat(F, axis=0)                    [iter007 现状]
  S1 swap_cat  : 'jk,ij->ik' (N-gemm C 块) + concat(axis=0)
  S2 swap_out  : 'jk,ij->ik' + 预分配 c 直写 c[s:e]=blk
  K2 ki_cat    : 'ij,jk->ki' (T-gemm (k,i) C 块，无内部拷贝) + concat(axis=1) + .T
  K3 ki_out    : 'ij,jk->ki' + 预分配 (N,m) 缓冲列切片直写 + .T
"""
import time

import numpy as np

np.seterr(divide="ignore", over="ignore", invalid="ignore")

rng = np.random.RandomState(42)
n = 2048
a = rng.rand(n, n).astype(np.float32)
b = rng.rand(n, n).astype(np.float32)
EDGES = [(i * n // 4, (i + 1) * n // 4) for i in range(4)]


def P0_parent():
    blocks = []
    for s, e in EDGES:
        if e > s:
            blocks.append(np.einsum('ij,jk->ik', a[s:e], b, optimize=True))
    return np.concatenate(blocks, axis=0)


def S1_swap_cat():
    blocks = []
    for s, e in EDGES:
        if e > s:
            blocks.append(np.einsum('jk,ij->ik', b, a[s:e], optimize=True))
    return np.concatenate(blocks, axis=0)


def S2_swap_out():
    c = np.empty((n, n), dtype=np.float32)
    for s, e in EDGES:
        if e > s:
            c[s:e] = np.einsum('jk,ij->ik', b, a[s:e], optimize=True)
    return c


def K2_ki_cat():
    tb = []
    for s, e in EDGES:
        if e > s:
            tb.append(np.einsum('ij,jk->ki', a[s:e], b, optimize=True))
    return np.concatenate(tb, axis=1).T


def K3_ki_out():
    ct = np.empty((n, n), dtype=np.float32)
    for s, e in EDGES:
        if e > s:
            ct[:, s:e] = np.einsum('ij,jk->ki', a[s:e], b, optimize=True)
    return ct.T


ref = np.matmul(a, b)
impls = [P0_parent, S1_swap_cat, S2_swap_out, K2_ki_cat, K3_ki_out]
for f in impls:
    r = f()
    err = float(np.max(np.abs(r.astype(np.float64) - ref.astype(np.float64))))
    assert r.shape == ref.shape and err == 0.0, (f.__name__, err)
print("全部变体 @2048 逐位一致于 np.matmul ✓ (max_error=0.0)")
for f in impls:
    f()
np.matmul(a, b)

REPS, ROUNDS = 5, 15
meds = {f.__name__: [] for f in impls}
ref_meds = []
pair_p0 = {f.__name__: [] for f in impls}
for rep in range(REPS):
    times = {f.__name__: [] for f in impls}
    rt = []
    for _ in range(ROUNDS):
        t0 = time.perf_counter(); np.matmul(a, b); rt.append((time.perf_counter() - t0) * 1000)
        for f in impls:
            t0 = time.perf_counter(); f(); times[f.__name__].append((time.perf_counter() - t0) * 1000)
    ref_meds.append(sorted(rt)[ROUNDS // 2])
    line = "rep%d ref %.3f:" % (rep, ref_meds[-1])
    for f in impls:
        m = sorted(times[f.__name__])[ROUNDS // 2]
        meds[f.__name__].append(m)
        pd = [times['P0_parent'][i] - times[f.__name__][i] for i in range(ROUNDS)]
        pair_p0[f.__name__].append((sorted(pd)[ROUNDS // 2], sum(1 for x in pd if x > 0)))
        line += "  %s %.3f" % (f.__name__[:9], m)
    print(line)

ref_med = sorted(ref_meds)[len(ref_meds) // 2]
print("\n汇总 (med-of-med, ref=%.3f):" % ref_med)
for f in impls:
    ms = sorted(meds[f.__name__])
    mm = ms[len(ms) // 2]
    pds = sorted(x[0] for x in pair_p0[f.__name__])
    poss = sum(x[1] for x in pair_p0[f.__name__])
    print("%-12s %.3f ms (range %.3f-%.3f) score~%.3f | paired P0-本实现 中位 %+.3f, 胜率 %d/%d" % (
        f.__name__, mm, ms[0], ms[-1], ref_med / mm, pds[len(pds) // 2], poss, REPS * ROUNDS))
