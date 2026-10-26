import time
import numpy as np

print("numpy", np.__version__)
print("dot: builtin, out kwarg supported per docs")

n = 8192
rng = np.random.RandomState(0)
a = rng.rand(n, n).astype(np.float32)
b = rng.rand(n, n).astype(np.float32)
ref = np.matmul(a, b)

def bench(fn, k=3, warm=2):
    for _ in range(warm):
        fn()
    ts = []
    for _ in range(k):
        t0 = time.perf_counter()
        fn()
        ts.append((time.perf_counter() - t0) * 1000.0)
    return min(ts), float(np.median(ts))

def t_dotout():
    out = np.empty((n, n), np.float32)
    q = n // 4
    for i in range(4):
        np.dot(a[i*q:(i+1)*q], b, out=out[i*q:(i+1)*q])
    return out

def t_tensordot_copy():
    out = np.empty((n, n), np.float32)
    q = n // 4
    for i in range(4):
        out[i*q:(i+1)*q] = np.tensordot(a[i*q:(i+1)*q], b, axes=((1,), (0,)))
    return out

def t_matmulout():
    out = np.empty((n, n), np.float32)
    q = n // 4
    for i in range(4):
        np.matmul(a[i*q:(i+1)*q], b, out=out[i*q:(i+1)*q])
    return out

def t_full():
    return np.matmul(a, b)

for name, fn in [("dot_out    ", t_dotout),
                 ("tensor_copy", t_tensordot_copy),
                 ("matmul_out ", t_matmulout),
                 ("full_matmul", t_full)]:
    mn, md = bench(fn)
    r = fn()
    eq = np.array_equal(r, ref)
    mx = float(np.max(np.abs(r.astype(np.float64) - ref.astype(np.float64))))
    print(f"{name}: min={mn:8.1f}ms med={md:8.1f}ms bitwise={eq} max_err={mx}")
