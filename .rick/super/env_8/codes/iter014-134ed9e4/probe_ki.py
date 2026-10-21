"""决定性探针：'ki' 输出拼写（跳过内部转置拷贝）vs parent canonical。

结构对比（numpy 2.0.2 源码追踪 + 实测）：
- canonical 'ij,jk->ik'：tensordot(B,A)→T-gemm→(k,i) C 块→c_einsum('ki->ik') 转置拷贝→F 块
                        →concatenate(F, axis=0) 0.755ms（2KB 粒度散读）
- ki 拼写    'ij,jk->ki'：同一 tensordot(B,A)→T-gemm→(k,i) C 块【无内部拷贝】
                        →concatenate(C, axis=1) 0.78ms→免费 .T 视图返回（F-序返回，门为值比较，合法）
- swapped    'jk,ij->ik'：tensordot(A,B)→N-gemm（实测更慢 2.115 vs 1.955）→弃

本探针：块级计时 + 全结构配对 + 多规模正确性（含 65/1/2 边界）。
"""
import time

import numpy as np

np.seterr(divide="ignore", over="ignore", invalid="ignore")


def bench(fn, rounds=40):
    fn()
    ts = []
    for _ in range(rounds):
        t0 = time.perf_counter()
        fn()
        ts.append((time.perf_counter() - t0) * 1000)
    ts.sort()
    return ts[rounds // 2]


rng = np.random.RandomState(42)
n = 2048
a = rng.rand(n, n).astype(np.float32)
b = rng.rand(n, n).astype(np.float32)

# ---- 块级 ----
s, e = 0, 512
a_blk = a[s:e]
r_canon = np.einsum('ij,jk->ik', a_blk, b, optimize=True)
r_ki = np.einsum('ij,jk->ki', a_blk, b, optimize=True)
print("ki 块 flags C/F:", r_ki.flags['C_CONTIGUOUS'], r_ki.flags['F_CONTIGUOUS'], "shape", r_ki.shape)
print("ki 值 == canonical.T:", np.array_equal(r_ki, r_canon.T))
print("ki 值 bitwise == matmul.T:", np.array_equal(r_ki, np.matmul(a_blk, b).T))
t_canon = bench(lambda: np.einsum('ij,jk->ik', a_blk, b, optimize=True))
t_ki = bench(lambda: np.einsum('ij,jk->ki', a_blk, b, optimize=True))
t_dotT = bench(lambda: np.dot(b.T, a_blk.T))  # T-gemm 裸成本参照（np_dot 派发，仅归因用）
t_dotN = bench(lambda: np.dot(a_blk, b))    # N-gemm 裸成本参照
print("块级 median ms: canonical %.3f | ki %.3f | dot(b.T,a) %.3f | dot(a,b) %.3f" % (
    t_canon, t_ki, t_dotT, t_dotN))


# ---- 全结构 ----
def parent_impl():
    edges = [(i * n // 4, (i + 1) * n // 4) for i in range(4)]
    blocks = []
    for s_, e_ in edges:
        if e_ > s_:
            blocks.append(np.einsum('ij,jk->ik', a[s_:e_], b, optimize=True))
    return np.concatenate(blocks, axis=0)


def ki_impl():
    m = a.shape[0]
    edges = [(i * m // 4, (i + 1) * m // 4) for i in range(4)]
    tblocks = []
    for s_, e_ in edges:
        if e_ > s_:
            tblocks.append(np.einsum('ij,jk->ki', a[s_:e_], b, optimize=True))
    if not tblocks:
        return np.zeros((0, b.shape[1]), dtype=np.float32)
    return np.concatenate(tblocks, axis=1).T


ref = np.matmul(a, b)
for f in (parent_impl, ki_impl):
    r = f()
    print(f.__name__, "bitwise==matmul:", np.array_equal(r, ref),
          "ret C/F:", r.flags['C_CONTIGUOUS'], r.flags['F_CONTIGUOUS'])

impls = [parent_impl, ki_impl]
for f in impls:
    f()
np.matmul(a, b)
rounds = 31
times = {f.__name__: [] for f in impls}
ref_t = []
for _ in range(rounds):
    t0 = time.perf_counter(); np.matmul(a, b); ref_t.append((time.perf_counter() - t0) * 1000)
    for f in impls:
        t0 = time.perf_counter(); f(); times[f.__name__].append((time.perf_counter() - t0) * 1000)
ref_med = sorted(ref_t)[rounds // 2]
print("\nref median %.3f ms" % ref_med)
for f in impls:
    ts = sorted(times[f.__name__])
    print("%-13s median %.3f  min %.3f  max %.3f  score~%.3f" % (
        f.__name__, ts[rounds // 2], ts[0], ts[-1], ref_med / ts[rounds // 2]))
pd = [times['parent_impl'][i] - times['ki_impl'][i] for i in range(rounds)]
print("paired parent-ki: median %.3f ms  pos %d/%d" % (
    sorted(pd)[rounds // 2], sum(1 for x in pd if x > 0), rounds))


# ---- 多规模正确性（含边界：65 非整除 / 2 / 1 / 0）----
def ki_matmul(a, b):
    a32 = np.asarray(a, dtype=np.float32)
    b32 = np.asarray(b, dtype=np.float32)
    m = a32.shape[0]
    edges = [(i * m // 4, (i + 1) * m // 4) for i in range(4)]
    tblocks = []
    for s_, e_ in edges:
        if e_ > s_:
            tblocks.append(np.einsum('ij,jk->ki', a32[s_:e_], b32, optimize=True))
    if not tblocks:
        return np.zeros((0, b32.shape[1]), dtype=np.float32)
    return np.concatenate(tblocks, axis=1).T


print("\n多规模正确性:")
ok_all = True
for m_, k_, n2_ in [(64, 64, 64), (2048, 2048, 2048), (65, 63, 67), (2, 5, 3), (1, 1, 1),
                    (3, 7, 5), (7, 3, 11), (0, 4, 4), (128, 256, 64)]:
    rng2 = np.random.RandomState(m_ * 1000 + k_)
    aa = rng2.rand(m_, k_).astype(np.float32)
    bb = rng2.rand(k_, n2_).astype(np.float32)
    got = ki_matmul(aa, bb)
    exp = np.matmul(aa, bb)
    err = float(np.max(np.abs(got.astype(np.float64) - exp.astype(np.float64)))) if got.size else 0.0
    ok = got.shape == exp.shape and err == 0.0
    ok_all &= ok
    print("  m=%-5d k=%-5d n=%-5d shape=%s bitwise0=%s err=%.2e" % (m_, k_, n2_, got.shape, ok, err))
print("ALL OK:", ok_all)
