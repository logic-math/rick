"""组装结构探针：canonical T-gemm 块（F-布局）的最廉价拼装方式。

已知（probe_decompose）：canonical 块 1.955ms（最快 gemm），F-块 concatenate(axis=0)
0.813ms 是主要可攻击开销；C-块级拷贝只要 0.234ms。F-块的 .T 是零成本 C-连续视图。

候选组装：
A) concat([blk.T ...], axis=1).T     —— 拼转置再免费转置返回（F-序返回）
B) 预分配 c，c[:, s:e] = blk.T       —— 列切片直写（C-序返回）
C) concat(F-块, axis=0, out=F-序缓冲) —— F-序 out= 拼接
D) 对照：parent 的 concat(F-块, axis=0)
"""
import time

import numpy as np

np.seterr(divide="ignore", over="ignore", invalid="ignore")

rng = np.random.RandomState(42)
n = 2048
a = rng.rand(n, n).astype(np.float32)
b = rng.rand(n, n).astype(np.float32)

blocks = [np.einsum('ij,jk->ik', a[i * n // 4:(i + 1) * n // 4], b, optimize=True) for i in range(4)]
assert all(x.flags['F_CONTIGUOUS'] for x in blocks)

print("concat(F,axis=0) 输出 flags C/F:",
      np.concatenate(blocks, axis=0).flags['C_CONTIGUOUS'],
      np.concatenate(blocks, axis=0).flags['F_CONTIGUOUS'])


def bench(fn, rounds=40):
    fn()
    ts = []
    for _ in range(rounds):
        t0 = time.perf_counter()
        fn()
        ts.append((time.perf_counter() - t0) * 1000)
    ts.sort()
    return ts[rounds // 2]


def assy_A():
    return np.concatenate([blk.T for blk in blocks], axis=1).T


def assy_B():
    c = np.empty((n, n), dtype=np.float32)
    for i, blk in enumerate(blocks):
        c[:, i * n // 4:(i + 1) * n // 4] = blk.T
    return c


def assy_C():
    out = np.empty((n, n), dtype=np.float32, order='F')
    return np.concatenate(blocks, axis=0, out=out)


def assy_D():
    return np.concatenate(blocks, axis=0)


ref = np.matmul(a, b)
for name, f in [("A concatT+.T", assy_A), ("B colslice", assy_B),
                ("C concatF/out=F", assy_C), ("D concatF (parent)", assy_D)]:
    r = f()
    ok = np.array_equal(np.ascontiguousarray(r), ref)
    print("%-18s ok=%s  ret C/F=%s/%s  median %.3f ms" % (
        name, ok, r.flags['C_CONTIGUOUS'], r.flags['F_CONTIGUOUS'], bench(f)))

# ---- 完整候选 vs parent 配对 ----
def parent_impl():
    edges = [(i * n // 4, (i + 1) * n // 4) for i in range(4)]
    blks = []
    for s, e in edges:
        if e > s:
            blks.append(np.einsum('ij,jk->ik', a[s:e], b, optimize=True))
    return np.concatenate(blks, axis=0)


def cand_impl():  # A 组装
    edges = [(i * n // 4, (i + 1) * n // 4) for i in range(4)]
    blks = []
    for s, e in edges:
        if e > s:
            blks.append(np.einsum('ij,jk->ik', a[s:e], b, optimize=True))
    if not blks:
        return np.zeros((0, b.shape[1]), dtype=np.float32)
    return np.concatenate([blk.T for blk in blks], axis=1).T


def cand_B_impl():  # B 组装
    edges = [(i * n // 4, (i + 1) * n // 4) for i in range(4)]
    c = np.empty((n, b.shape[1]), dtype=np.float32)
    for s, e in edges:
        if e > s:
            c[:, s:e] = np.einsum('ij,jk->ik', a[s:e], b, optimize=True).T
    return c


impls = [parent_impl, cand_impl, cand_B_impl]
for f in impls:
    f()
print()
rounds = 25
times = {f.__name__: [] for f in impls}
ref_t = []
for _ in range(rounds):
    t0 = time.perf_counter(); np.matmul(a, b); ref_t.append((time.perf_counter() - t0) * 1000)
    for f in impls:
        t0 = time.perf_counter(); f(); times[f.__name__].append((time.perf_counter() - t0) * 1000)
ref_med = sorted(ref_t)[rounds // 2]
print("ref median %.3f" % ref_med)
for f in impls:
    ts = sorted(times[f.__name__])
    print("%-14s median %.3f  min %.3f  max %.3f  score~%.3f" % (
        f.__name__, ts[rounds // 2], ts[0], ts[-1], ref_med / ts[rounds // 2]))
pd = [times['parent_impl'][i] - times['cand_impl'][i] for i in range(rounds)]
print("paired parent-candA: median %.3f ms  pos %d/%d" % (sorted(pd)[rounds // 2], sum(1 for x in pd if x > 0), rounds))
