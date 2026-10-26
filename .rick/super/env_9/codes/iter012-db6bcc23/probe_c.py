import cProfile, pstats, io, time
import numpy as np
np.seterr(divide="ignore", over="ignore", invalid="ignore")

n = 8192
rng = np.random.RandomState(0)
a = rng.rand(n, n).astype(np.float32)
b = rng.rand(n, n).astype(np.float32)

def einsum_quarters():
    out = np.empty((n, n), np.float32)
    q = n // 4
    for i in range(4):
        out[i*q:(i+1)*q] = np.einsum('ij,jk->ik', a[i*q:(i+1)*q], b, optimize=True)
    return out

# ---- 1) cProfile one einsum_quarters call: where do the extra ~170ms go?
einsum_quarters()  # warm
pr = cProfile.Profile()
pr.enable()
einsum_quarters()
pr.disable()
s = io.StringIO()
ps = pstats.Stats(pr, stream=s).sort_stats("cumulative")
ps.print_stats(18)
print(s.getvalue())

# ---- 2) micro: what does a single einsum call on ONE quarter block cost vs tensordot?
q = n // 4
a_blk = a[q:2*q]
for name, fn in [
    ("tensordot_1blk", lambda: np.tensordot(a_blk, b, axes=((1,), (0,)))),
    ("einsum_1blk    ", lambda: np.einsum('ij,jk->ik', a_blk, b, optimize=True)),
    ("einsum_1blk_noo", lambda: np.einsum('ij,jk->ik', a_blk, b, optimize=False)),
    ("dot_1blk       ", lambda: np.dot(a_blk, b)),
]:
    for _ in range(2):
        fn()
    ts = []
    for _ in range(4):
        t0 = time.perf_counter()
        fn()
        ts.append((time.perf_counter() - t0) * 1000)
    print(f"{name}: min={min(ts):7.1f}ms all={[round(t,1) for t in ts]}")
