#!/usr/bin/env python3
"""iter024 builder 探针 3（修正版）：E 核（background QoS）协同 M 四分的可行域。

修正探针 2 的闭包 bug（worker 显式传参 a/b/c，不再闭包引用）。
假设链：
  H1: background QoS worker 被调度器钉到 E 核（单任务 26.9ms ≈ 0.16 TF/s 节流态）
  H2: 4 个 bg-QoS 并发 sgemm 在 E 核上并发执行（不是串行排队）→ 聚合 ~0.6 TF/s
  H3: 4 个 interactive-QoS P 任务 + 4 个 bg-QoS E 任务并发时，P 侧速率不掉
      （GCD 防超订对 QoS 分层的并发调用不作恶）
  H4: 组合墙钟 = max(P 大块, E 尾块) < q4w 7.13ms → 突破整阵派发天花板 6.94ms

变体：
  full        单次整阵 np.dot
  q4w         4 块 × 4 workers（默认 QoS；父本形态）
  q4w_qos     4 块 × 4 workers（interactive QoS）
  e4w_bg      4 块 × 4 workers（background QoS；测 E 聚合速率）
  hetero_fXX  每四分块 [0,f) 行给 P worker + [f,1) 行给 bg-QoS E worker
"""
import ctypes
import time

import numpy as np

np.seterr(divide="ignore", over="ignore", invalid="ignore")

N = 2048
REPS = 9

QOS_USER_INTERACTIVE = 0x21
QOS_BACKGROUND = 0x09

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


def msplit(m, k):
    return [(i * m // k, (i + 1) * m // k) for i in range(k)]


def gflop_of(rows, n):
    return 2.0 * rows * n * n / 1e9


def worker_dot(a, b, c, s, e):
    np.dot(a[s:e], b, out=c[s:e])


def worker_dot_qos(cls):
    def w(a, b, c, s, e):
        set_qos(cls)
        np.dot(a[s:e], b, out=c[s:e])
    return w


def make_pool(nw, task_fn):
    from concurrent.futures import ThreadPoolExecutor
    return ThreadPoolExecutor(max_workers=nw), task_fn


def run_pool(pool, task_fn, a, b, c, tasks):
    futs = [pool.submit(task_fn, a, b, c, s, e) for (s, e) in tasks]
    for f in futs:
        f.result()


def main():
    rng = np.random.RandomState(42)
    a = rng.rand(N, N).astype(np.float32)
    b = rng.rand(N, N).astype(np.float32)
    ref = np.matmul(a, b)

    from concurrent.futures import ThreadPoolExecutor
    pool4 = ThreadPoolExecutor(max_workers=4)
    poolP = ThreadPoolExecutor(max_workers=4)
    poolE = ThreadPoolExecutor(max_workers=4)

    w_plain = worker_dot
    w_int = worker_dot_qos(QOS_USER_INTERACTIVE)
    w_bg = worker_dot_qos(QOS_BACKGROUND)

    quarters = msplit(N, 4)

    def full(a_, b_):
        return np.dot(a_, b_)

    def q4w(a_, b_):
        c = np.empty((N, N), dtype=np.float32)
        run_pool(pool4, w_plain, a_, b_, c, quarters)
        return c

    def q4w_qos(a_, b_):
        c = np.empty((N, N), dtype=np.float32)
        run_pool(pool4, w_int, a_, b_, c, quarters)
        return c

    def e4w_bg(a_, b_):
        c = np.empty((N, N), dtype=np.float32)
        run_pool(poolE, w_bg, a_, b_, c, quarters)
        return c

    def make_hetero(f):
        def fn(a_, b_):
            c = np.empty((N, N), dtype=np.float32)
            futs = []
            for (s, e) in quarters:
                cut = s + int((e - s) * f)
                if cut > s:
                    futs.append(poolP.submit(w_int, a_, b_, c, s, cut))
                if e > cut:
                    futs.append(poolE.submit(w_bg, a_, b_, c, cut, e))
            for fu in futs:
                fu.result()
            return c
        return fn

    variants = {
        "full": full,
        "q4w": q4w,
        "q4w_qos": q4w_qos,
        "e4w_bg": e4w_bg,
    }
    for f in [0.92, 0.88, 0.84, 0.80, 0.75, 0.70]:
        variants[f"h_{f:.2f}"] = make_hetero(f)

    # 预热 + 正确性（每变体核对 err==0 —— 行分裂保 K 求和序）
    for name, fn in variants.items():
        c = fn(a, b)
        err = float(np.max(np.abs(c.astype(np.float64) - ref.astype(np.float64))))
        flag = "OK " if err == 0.0 else "BAD"
        print(f"  warmup {name:<10} max_err={err:g} {flag}")

    results = {k: [] for k in variants}
    for _ in range(REPS):
        for name, fn in variants.items():
            t0 = time.perf_counter()
            fn(a, b)
            results[name].append((time.perf_counter() - t0) * 1000.0)

    print(f"\n{'variant':<10} {'median_ms':>10} {'min_ms':>8} {'max_ms':>8}  TF/s   vs_full")
    base = sorted(results["full"])[REPS // 2]
    for name in variants:
        ts = sorted(results[name])
        med = ts[REPS // 2]
        tf = gflop_of(N, N) / (med / 1000.0)
        print(f"{name:<10} {med:>10.3f} {ts[0]:>8.3f} {ts[-1]:>8.3f}  {tf:.2f}  {base / med:.3f}x")


if __name__ == "__main__":
    main()
