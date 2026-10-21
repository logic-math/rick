#!/usr/bin/env python3
"""iter024 builder 探针：6 P 核（M5 Max）下并发 sgemm 的可扩展性。

背景：父本 iter017 = M 四分 × 4 workers（每块一线程）→ 只占 4/6 P 核。
本探针测：并发单线程 sgemm 是否随并发度 4→6 扩展？M 四分结构不变的前提下，
块内 N 维细分成 12 任务 × 6 workers 能否吃到 6 核收益？

变体（全部 np.dot + out= 直写、M/N 分解不改 K 求和序 → 逐位一致）：
  full          单次整阵 np.dot（天花板参照）
  seq_q         顺序 M 四分（父本 iter010 形态）
  q4w           M 四分 × 4 workers（父本 iter017 形态，当前精英）
  s6w           M 六分 × 6 workers（越格探针：纯测 6 核可扩展性）
  q_n3_6w       M 四分 × N 三分 = 12 任务 × 6 workers（格内候选：M 边界仍四分）
  q_n2_8w       M 四分 × N 二分 = 8 任务 × 8 workers（E 核拖尾检验）
  e8w           M 八分 × 8 workers（E 核拖尾检验）
  q_n3_12w      12 任务 × 12 workers（E 核拖尾检验）
"""
import time

import numpy as np

np.seterr(divide="ignore", over="ignore", invalid="ignore")

N = 2048
REPS = 11  # 每变体次数（交替轮转摊平背景负载），取中位


def msplit(m, k):
    return [(i * m // k, (i + 1) * m // k) for i in range(k)]


def nsplit(n, k):
    return [(j * n // k, (j + 1) * n // k) for j in range(k)]


def tasks_m(mblocks):
    return [(s, e, None, None) for s, e in mblocks]


def tasks_mn(mblocks, nstrips):
    return [(s, e, j0, j1) for (s, e) in mblocks for (j0, j1) in nstrips]


def dot_task(a32, b32, c, t):
    s, e, j0, j1 = t
    if j0 is None:
        np.dot(a32[s:e], b32, out=c[s:e])
    else:
        # np.dot 的 out= 只接受 C 连续数组 → N 条带经临时阵 + 视图回拷
        c[s:e, j0:j1] = np.dot(a32[s:e], b32[:, j0:j1])


def matmul_task(a32, b32, c, t):
    s, e, j0, j1 = t
    if j0 is None:
        np.matmul(a32[s:e], b32, out=c[s:e])
    else:
        # np.matmul 的 gufunc out= 接受跨步视图（免拷贝上界参照，越格探针）
        np.matmul(a32[s:e], b32[:, j0:j1], out=c[s:e, j0:j1])


def make_threaded(nworkers, task_list_fn, task_fn=dot_task):
    from concurrent.futures import ThreadPoolExecutor

    pool = ThreadPoolExecutor(max_workers=nworkers)

    def fn(a, b):
        a32 = np.asarray(a, dtype=np.float32)
        b32 = np.asarray(b, dtype=np.float32)
        c = np.empty((a32.shape[0],) + b32.shape[1:], dtype=np.float32)
        ts = task_list_fn(a32.shape[0], b32.shape[-1] if b32.ndim == 2 else None)
        if b32.ndim == 1:
            ts = [(s, e, None, None) for s, e, _, _ in ts]
        futs = [pool.submit(task_fn, a32, b32, c, t) for t in ts]
        for f in futs:
            f.result()
        return c

    return fn


def main():
    rng = np.random.RandomState(42)
    a = rng.rand(N, N).astype(np.float32)
    b = rng.rand(N, N).astype(np.float32)
    ref = np.matmul(a, b)

    def full(a, b):
        return np.dot(a, b)

    def seq_q(a, b):
        a32 = np.asarray(a, dtype=np.float32)
        b32 = np.asarray(b, dtype=np.float32)
        c = np.empty((a32.shape[0],) + b32.shape[1:], dtype=np.float32)
        for s, e in msplit(a32.shape[0], 4):
            np.dot(a32[s:e], b32, out=c[s:e])
        return c

    variants = {
        "full": full,
        "seq_q": seq_q,
        "q4w": make_threaded(4, lambda m, n: tasks_m(msplit(m, 4))),
        "s6w": make_threaded(6, lambda m, n: tasks_m(msplit(m, 6))),
        "q_n3_6w": make_threaded(6, lambda m, n: tasks_mn(msplit(m, 4), nsplit(n, 3))),
        "q_n3_6w_mm": make_threaded(6, lambda m, n: tasks_mn(msplit(m, 4), nsplit(n, 3)), matmul_task),
        "q_n2_8w": make_threaded(8, lambda m, n: tasks_mn(msplit(m, 4), nsplit(n, 2))),
        "e8w": make_threaded(8, lambda m, n: tasks_m(msplit(m, 8))),
        "q_n3_12w": make_threaded(12, lambda m, n: tasks_mn(msplit(m, 4), nsplit(n, 3))),
    }

    # 预热（池创建 + BLAS 线程态）
    for name, fn in variants.items():
        c = fn(a, b)
        err = float(np.max(np.abs(c.astype(np.float64) - ref.astype(np.float64))))
        assert c.shape == ref.shape, (name, c.shape)
        print(f"warmup {name}: max_err={err:g}")

    # 交替轮转测量
    results = {k: [] for k in variants}
    for rep in range(REPS):
        for name, fn in variants.items():
            t0 = time.perf_counter()
            c = fn(a, b)
            results[name].append((time.perf_counter() - t0) * 1000.0)

    print(f"\n{'variant':<10} {'median_ms':>10} {'min_ms':>8} {'max_ms':>8}  (n={N}, reps={REPS}, interleaved)")
    base = sorted(results["full"])[REPS // 2]
    for name in variants:
        ts = sorted(results[name])
        med = ts[REPS // 2]
        print(f"{name:<10} {med:>10.3f} {ts[0]:>8.3f} {ts[-1]:>8.3f}  vs full {base / med:.3f}x (score≈)")


if __name__ == "__main__":
    main()
