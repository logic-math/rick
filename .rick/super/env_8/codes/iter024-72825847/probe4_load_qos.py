#!/usr/bin/env python3
"""iter024 builder 探针 4：受控合成负载下 q4w vs q4w_qos(interactive) vs full。

动机：环境负载下（探针 3 会话，load~4-5）q4w 中位 7.71 vs full 9.66（1.25x）——
threaded 形态在负载下反而占优。若 worker 提升 QoS(interactive) 能进一步
巩固/放大负载下的优势（P 核调度优先于默认 QoS 的背景线程），则
「QoS 加固」是一个有非对称收益的格内实验。

合成负载：6 个默认 QoS 忙循环子进程（模拟框架/系统背景负载，load≈6）。
交替测量 full / q4w / q4w_qos，REPS=9 取中位。负载前后各测一轮空载对照。
"""
import ctypes
import subprocess
import sys
import time

import numpy as np

np.seterr(divide="ignore", over="ignore", invalid="ignore")

N = 2048
REPS = 9
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


def w_plain(a, b, c, s, e):
    np.dot(a[s:e], b, out=c[s:e])


def w_int(a, b, c, s, e):
    set_qos(QOS_USER_INTERACTIVE)
    np.dot(a[s:e], b, out=c[s:e])


def main():
    from concurrent.futures import ThreadPoolExecutor

    rng = np.random.RandomState(42)
    a = rng.rand(N, N).astype(np.float32)
    b = rng.rand(N, N).astype(np.float32)
    ref = np.matmul(a, b)
    quarters = [(i * N // 4, (i + 1) * N // 4) for i in range(4)]
    pool = ThreadPoolExecutor(max_workers=4)

    def full(a_, b_):
        return np.dot(a_, b_)

    def q4w(a_, b_):
        c = np.empty((N, N), dtype=np.float32)
        futs = [pool.submit(w_plain, a_, b_, c, s, e) for (s, e) in quarters]
        for f in futs:
            f.result()
        return c

    def q4w_qos(a_, b_):
        c = np.empty((N, N), dtype=np.float32)
        futs = [pool.submit(w_int, a_, b_, c, s, e) for (s, e) in quarters]
        for f in futs:
            f.result()
        return c

    variants = {"full": full, "q4w": q4w, "q4w_qos": q4w_qos}

    for name, fn in variants.items():
        c = fn(a, b)
        err = float(np.max(np.abs(c.astype(np.float64) - ref.astype(np.float64))))
        assert err == 0.0, (name, err)

    spinners = []

    def start_load(n=6):
        code = "while True: pass"
        for _ in range(n):
            spinners.append(subprocess.Popen([sys.executable, "-c", code]))

    def stop_load():
        for p in spinners:
            p.terminate()
        for p in spinners:
            p.wait()
        spinners.clear()

    def measure(tag):
        results = {k: [] for k in variants}
        for _ in range(REPS):
            for name, fn in variants.items():
                t0 = time.perf_counter()
                fn(a, b)
                results[name].append((time.perf_counter() - t0) * 1000.0)
        print(f"\n--- {tag} ---")
        print(f"{'variant':<10} {'median_ms':>10} {'min_ms':>8} {'max_ms':>8}")
        base = sorted(results["full"])[REPS // 2]
        for name in variants:
            ts = sorted(results[name])
            med = ts[REPS // 2]
            print(f"{name:<10} {med:>10.3f} {ts[0]:>8.3f} {ts[-1]:>8.3f}  vs full {base / med:.3f}x")
        return {k: sorted(v)[REPS // 2] for k, v in results.items()}

    measure("空载对照 pre-load")
    start_load(6)
    time.sleep(1.0)
    r_load = measure("合成负载 6x default-QoS spinners")
    stop_load()
    time.sleep(1.0)
    measure("空载对照 post-load")

    print("\n结论要点：负载下 q4w_qos vs q4w 中位差 = %.3fms" %
          (r_load["q4w"] - r_load["q4w_qos"]))


if __name__ == "__main__":
    main()
