import time
import numpy as np
np.seterr(divide="ignore", over="ignore", invalid="ignore")

n = 8192
rng = np.random.RandomState(0)
a = rng.rand(n, n).astype(np.float32)
b = rng.rand(n, n).astype(np.float32)
ref = np.matmul(a, b)

def tensor_copy(k=4):
    out = np.empty((n, n), np.float32)
    q = n // k
    for i in range(k):
        out[i*q:(i+1)*q] = np.tensordot(a[i*q:(i+1)*q], b, axes=((1,), (0,)))
    return out

def dot_out(k=4):
    out = np.empty((n, n), np.float32)
    q = n // k
    for i in range(k):
        np.dot(a[i*q:(i+1)*q], b, out=out[i*q:(i+1)*q])
    return out

def matmul_out(k=4):
    out = np.empty((n, n), np.float32)
    q = n // k
    for i in range(k):
        np.matmul(a[i*q:(i+1)*q], b, out=out[i*q:(i+1)*q])
    return out

def einsum_noout_assign(k=4):
    out = np.empty((n, n), np.float32)
    q = n // k
    for i in range(k):
        out[i*q:(i+1)*q] = np.einsum('ij,jk->ik', a[i*q:(i+1)*q], b, optimize=True)
    return out

def einsum_full_noout():
    return np.einsum('ij,jk->ik', a, b, optimize=True)

def full_matmul():
    return np.matmul(a, b)

ARMS = [
    ("tensor_copy(4)", lambda: tensor_copy(4)),
    ("dot_out(4)     ", lambda: dot_out(4)),
    ("matmul_out(4)  ", lambda: matmul_out(4)),
    ("einsum_noout(4)", lambda: einsum_noout_assign(4)),
    ("dot_out(2)     ", lambda: dot_out(2)),
    ("dot_out(8)     ", lambda: dot_out(8)),
    ("einsum_full    ", einsum_full_noout),
    ("full_matmul    ", full_matmul),
]

# bitwise checks (once)
for name, fn in ARMS:
    r = fn()
    eq = np.array_equal(r, ref)
    print(f"bitwise {name}: {eq}")

# interleaved timing: 2 warmup rounds + 6 measured rounds
for _ in range(2):
    for _, fn in ARMS:
        fn()
times = {name: [] for name, _ in ARMS}
for _ in range(6):
    for name, fn in ARMS:
        t0 = time.perf_counter()
        fn()
        times[name].append((time.perf_counter() - t0) * 1000.0)
for name, _ in ARMS:
    ts = sorted(times[name])
    print(f"{name}: min={ts[0]:7.1f} med={ts[len(ts)//2]:7.1f} all={[round(t,1) for t in ts]}")
