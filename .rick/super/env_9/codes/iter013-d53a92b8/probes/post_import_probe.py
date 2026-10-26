import os
import time
import numpy as np
np.seterr(divide="ignore", over="ignore", invalid="ignore")
# numpy ALREADY imported (宿主先 import 的情形模拟)——此刻才设 env
n = int(os.environ.get("PROBE_N", "8192"))
want = os.environ.get("WANT_THREADS", "1")
os.environ["VECLIB_MAXIMUM_THREADS"] = want
rng = np.random.RandomState(42)
a = rng.rand(n, n).astype(np.float32)
b = rng.rand(n, n).astype(np.float32)
np.matmul(a[:512, :512], b[:512, :512])
ts = []
for _ in range(3):
    t0 = time.perf_counter()
    np.matmul(a, b)
    ts.append((time.perf_counter() - t0) * 1e3)
ts.sort()
print("post-import-set WANT=%s n=%d median=%.1f timings=%s" % (want, n, ts[len(ts) // 2], [round(x, 1) for x in ts]))
