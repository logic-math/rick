#!/usr/bin/env python3
"""iter024 builder 探针 5（终判）：q4w vs q4w_qos(interactive) 高重复 A/B。

设计要点：
- QoS 经 ThreadPoolExecutor(initializer=...) 线程创建时设置一次（干净形态）
- 交替配对 31 轮（空载）+ 15 轮（合成负载 6 spinners），报告配对差分布
- 结论标准：负载轮配对差中位 > 0 且 ≥60% 轮改善、空载轮 |差| < 0.1ms → QoS 实验成立
"""
import ctypes
import subprocess
import sys
import time

import numpy as np

np.seterr(divide="ignore", over="ignore", invalid="ignore")

N = 2048
QOS_USER_INTERACTIVE = 0x21

_libc = None


def set_qos(cls):
    global _libc
    try:
        if _libc is None:
            _libc = ctypes.CDLL(None)
            _libc.pthread_set_qos_class_self_np.restype = ctypes.c_int
            _libc.pthread_set_qos_class_self_np.argtypes = [ctypes.c_int]
        return _libc.pthread_set_qos_class_self_np(cls) == 0
    except Exception:
        return False


def w(a, b, c, s, e):
    np.dot(a[s:e], b, out=c[s:e])


def main():
    from concurrent.futures import ThreadPoolExecutor

    rng = np.random.RandomState(42)
    a = rng.rand(N, N).astype(np.float32)
    b = rng.rand(N, N).astype(np.float32)
    ref = np.matmul(a, b)
    quarters = [(i * N // 4, (i + 1) * N // 4) for i in range(4)]

    def init_qos():
        set_qos(QOS_USER_INTERACTIVE)

    pool_plain = ThreadPoolExecutor(max_workers=4)
    pool_qos = ThreadPoolExecutor(max_workers=4, initializer=init_qos)

    def q4w(a_, b_):
        c = np.empty((N, N), dtype=np.float32)
        futs = [pool_plain.submit(w, a_, b_, c, s, e) for (s, e) in quarters]
        for f in futs:
            f.result()
        return c

    def q4w_qos(a_, b_):
        c = np.empty((N, N), dtype=np.float32)
        futs = [pool_qos.submit(w, a_, b_, c, s, e) for (s, e) in quarters]
        for f in futs:
            f.result()
        return c

    for name, fn in [("q4w", q4w), ("q4w_qos", q4w_qos)]:
        c = fn(a, b)
        err = float(np.max(np.abs(c.astype(np.float64) - ref.astype(np.float64))))
        assert err == 0.0, (name, err)

    def paired(tag, rounds):
        diffs = []
        ts_a, ts_b = [], []
        for _ in range(rounds):
            t0 = time.perf_counter(); q4w(a, b); ta = (time.perf_counter() - t0) * 1e3
            t0 = time.perf_counter(); q4w_qos(a, b); tb = (time.perf_counter() - t0) * 1e3
            ts_a.append(ta); ts_b.append(tb); diffs.append(ta - tb)
        ds = sorted(diffs)
        med = ds[len(ds) // 2]
        improved = sum(1 for d in diffs if d > 0)
        print(f"{tag}: q4w={sorted(ts_a)[rounds//2]:.3f}ms q4w_qos={sorted(ts_b)[rounds//2]:.3f}ms "
              f"配对差中位={med:+.3f}ms 改善轮占比={improved}/{rounds}")
        return med

    paired("空载  ", 31)

    spinners = [subprocess.Popen([sys.executable, "-c", "while True: pass"]) for _ in range(6)]
    time.sleep(1.0)
    paired("负载6 ", 15)
    for p in spinners:
        p.terminate()
    for p in spinners:
        p.wait()
    time.sleep(1.0)
    paired("空载2 ", 15)


if __name__ == "__main__":
    main()
