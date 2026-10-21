"""最终探针：显式收缩路径 optimize=[(0,1)]（跳过 greedy path 计算）× ki 拼写组合。

perf_profile 实测 einsum_path 4 次调用 0.102ms（~25μs/次）；显式路径列表可跳过
greedy 求解。组合候选：
  P0 parent（canonical + optimize=True + concat F）
  P1 canonical + optimize=[(0,1)]
  P2 ki + optimize=True
  P3 ki + optimize=[(0,1)]（满配）
验证显式路径仍走 tensordot 派发 + 逐位一致。
"""
import time

import numpy as np

np.seterr(divide="ignore", over="ignore", invalid="ignore")

rng = np.random.RandomState(42)
n = 2048
a = rng.rand(n, n).astype(np.float32)
b = rng.rand(n, n).astype(np.float32)

PATH = [(0, 1)]

# 显式路径派发验证
r_auto = np.einsum('ij,jk->ki', a[:512], b, optimize=True)
r_expl = np.einsum('ij,jk->ki', a[:512], b, optimize=PATH)
print("explicit-path bitwise == auto:", np.array_equal(r_auto, r_expl))
import numpy._core.numeric as _numeric  # noqa: F401  (仅确认导入路径)

import cProfile, pstats, io


def _prof(fn):
    pr = cProfile.Profile()
    pr.enable(); fn(); pr.disable()
    s = io.StringIO()
    pstats.Stats(pr, stream=s).sort_stats("cumulative").print_stats(6)
    for line in s.getvalue().splitlines():
        if "einsum_path" in line or "tensordot" in line:
            print("   ", line.strip())


print("profile: einsum('ij,jk->ki', optimize=[(0,1)]) 派发：")
_prof(lambda: np.einsum('ij,jk->ki', a[:512], b, optimize=PATH))


def P0_parent():
    edges = [(i * n // 4, (i + 1) * n // 4) for i in range(4)]
    blocks = []
    for s, e in edges:
        if e > s:
            blocks.append(np.einsum('ij,jk->ik', a[s:e], b, optimize=True))
    return np.concatenate(blocks, axis=0)


def P1_canon_expl():
    edges = [(i * n // 4, (i + 1) * n // 4) for i in range(4)]
    blocks = []
    for s, e in edges:
        if e > s:
            blocks.append(np.einsum('ij,jk->ik', a[s:e], b, optimize=PATH))
    return np.concatenate(blocks, axis=0)


def P2_ki():
    edges = [(i * n // 4, (i + 1) * n // 4) for i in range(4)]
    tb = []
    for s, e in edges:
        if e > s:
            tb.append(np.einsum('ij,jk->ki', a[s:e], b, optimize=True))
    return np.concatenate(tb, axis=1).T


def P3_ki_expl():
    edges = [(i * n // 4, (i + 1) * n // 4) for i in range(4)]
    tb = []
    for s, e in edges:
        if e > s:
            tb.append(np.einsum('ij,jk->ki', a[s:e], b, optimize=PATH))
    return np.concatenate(tb, axis=1).T


ref = np.matmul(a, b)
impls = [P0_parent, P1_canon_expl, P2_ki, P3_ki_expl]
for f in impls:
    r = f()
    assert np.array_equal(r, ref) or float(np.max(np.abs(r.astype(np.float64) - ref.astype(np.float64)))) < 1e-6
for f in impls:
    f()
np.matmul(a, b)

REPS, ROUNDS = 3, 21
summary = {f.__name__: [] for f in impls}
ref_meds = []
for rep in range(REPS):
    times = {f.__name__: [] for f in impls}
    ref_t = []
    for _ in range(ROUNDS):
        t0 = time.perf_counter(); np.matmul(a, b); ref_t.append((time.perf_counter() - t0) * 1000)
        for f in impls:
            t0 = time.perf_counter(); f(); times[f.__name__].append((time.perf_counter() - t0) * 1000)
    ref_meds.append(sorted(ref_t)[ROUNDS // 2])
    for f in impls:
        summary[f.__name__].append(sorted(times[f.__name__])[ROUNDS // 2])
    pd = [times['P0_parent'][i] - times['P3_ki_expl'][i] for i in range(ROUNDS)]
    print("rep%d: ref %.3f | P0 %.3f | P1 %.3f | P2 %.3f | P3 %.3f | paired P0-P3 med %+.3f (%d/%d pos)" % (
        rep, ref_meds[-1], summary['P0_parent'][-1], summary['P1_canon_expl'][-1],
        summary['P2_ki'][-1], summary['P3_ki_expl'][-1],
        sorted(pd)[ROUNDS // 2], sum(1 for x in pd if x > 0), ROUNDS))

print("\n各实现 3 rep 中位汇总:")
ref_med = sorted(ref_meds)[len(ref_meds) // 2]
for f in impls:
    ms = sorted(summary[f.__name__])
    med = ms[len(ms) // 2]
    print("%-14s med-of-med %.3f  range %.3f-%.3f  score~%.3f" % (
        f.__name__, med, ms[0], ms[-1], ref_med / med))
