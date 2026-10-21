#!/usr/bin/env python3
"""pre_bench.py — iter022 预研：einsum|m_halves 装配副本线程化重叠的可行性测量。

背景（父本 iter013 剖析 @2048）：
- 端到端 7.87ms = 2×tensordot(gemm) 7.50 + 副本 0.30 + einsum 包装 0.09。
- 拆分税（2×M=1024 vs 1×M=2048）+0.55ms 为结构成本；副本 0.30ms 是最后一块
  「纯搬运」开销——单线程串行时它裸露在关键路径上。

假设（本预研裁决）：副本是纯 memcpy（带宽型），gemm 是 SME 计算型——两者重叠
（gemm2 在主线程跑、c[:h]=r1 的副本在 worker 线程跑）可把 copy1 完全藏进
gemm2 的 3.75ms 里，端到端省 ~0.15ms（copy1 的一半）。
关键区别于 iter017 的 2-worker 劣化形态：任一时刻只有 1 个 sgemm 在跑
（gemm1 → [gemm2 ∥ copy1] → copy2），不触发 2 路并发 sgemm 的 GCD 争抢。

变体：
- V0 base    ：父本结构（顺序 gemm + 趁热逐块拷贝）——基线
- V1 overlap ：gemm1 → submit(copy1) → gemm2 → copy2 → join（副本与 gemm2 重叠）
- V1b both   ：两个副本都进 worker（gemm1 → submit copy1 → gemm2 → submit copy2 → join×2）
- V2 concur2 ：两个 gemm 并发（2 路并发 sgemm，预期劣化——iter017 对照复现）
- Vgemm      ：纯 gemm 拆解（2×tensordot 无副本）
- Vfull      ：整阵 einsum 派发（地板参照）
- ref        ：np.matmul 同会话计时（负载归一）

协议：同进程交错配对 ×N 轮，中位 + 配对胜率。
"""
import json
import statistics
import time
from concurrent.futures import ThreadPoolExecutor

import numpy as np

np.seterr(divide="ignore", over="ignore", invalid="ignore")

N = 2048
ROUNDS = 30

pool = ThreadPoolExecutor(max_workers=2)


def _einsum_half(b32, a_blk):
    """worker 内执行的半块 swapped einsum（与主线程同拼写同 optimize）。"""
    return np.einsum('jk,ij->ik', b32, a_blk, optimize=True)


def _prep():
    a32 = np.asarray(A, dtype=np.float32)
    b32 = np.asarray(B, dtype=np.float32)
    return a32, b32


def v0_base(a32, b32):
    m = a32.shape[0]
    h = m // 2
    c = np.empty((m,) + b32.shape[1:], dtype=np.float32)
    r1 = np.einsum('jk,ij->ik', b32, a32[:h], optimize=True)
    c[:h] = r1
    r2 = np.einsum('jk,ij->ik', b32, a32[h:], optimize=True)
    c[h:] = r2
    return c


def v1_overlap(a32, b32):
    m = a32.shape[0]
    h = m // 2
    c = np.empty((m,) + b32.shape[1:], dtype=np.float32)
    r1 = np.einsum('jk,ij->ik', b32, a32[:h], optimize=True)
    fut = pool.submit(c[:h].__setitem__, slice(None), r1)  # copy1 与 gemm2 重叠
    r2 = np.einsum('jk,ij->ik', b32, a32[h:], optimize=True)
    c[h:] = r2
    fut.result()
    return c


def v1b_both(a32, b32):
    m = a32.shape[0]
    h = m // 2
    c = np.empty((m,) + b32.shape[1:], dtype=np.float32)
    r1 = np.einsum('jk,ij->ik', b32, a32[:h], optimize=True)
    f1 = pool.submit(c[:h].__setitem__, slice(None), r1)
    r2 = np.einsum('jk,ij->ik', b32, a32[h:], optimize=True)
    f2 = pool.submit(c[h:].__setitem__, slice(None), r2)
    f1.result()
    f2.result()
    return c


def v2_concur2(a32, b32):
    m = a32.shape[0]
    h = m // 2
    c = np.empty((m,) + b32.shape[1:], dtype=np.float32)
    f1 = pool.submit(_einsum_half, b32, a32[:h])
    f2 = pool.submit(_einsum_half, b32, a32[h:])
    r1 = f1.result()
    c[:h] = r1
    r2 = f2.result()
    c[h:] = r2
    return c


def vgemm(a32, b32):
    m = a32.shape[0]
    h = m // 2
    r1 = np.einsum('jk,ij->ik', b32, a32[:h], optimize=True)
    r2 = np.einsum('jk,ij->ik', b32, a32[h:], optimize=True)
    return r1, r2


def vfull(a32, b32):
    return np.einsum('jk,ij->ik', b32, a32, optimize=True)


VARIANTS = [
    ("v0_base", v0_base),
    ("v1_overlap", v1_overlap),
    ("v1b_both", v1b_both),
    ("v2_concur2", v2_concur2),
    ("vgemm_only", vgemm),
    ("vfull", vfull),
    ("ref_matmul", lambda a, b: np.matmul(a, b)),
]


def main():
    global A, B
    rng = np.random.RandomState(42)
    A = rng.rand(N, N).astype(np.float32)
    B = rng.rand(N, N).astype(np.float32)
    a32, b32 = A, B

    # 正确性快检：所有变体 vs np.matmul 逐位一致
    ref = np.matmul(a32, b32)
    for name, fn in VARIANTS:
        out = fn(a32, b32)
        if isinstance(out, tuple):
            out = np.concatenate(out, axis=0)
        err = float(np.max(np.abs(out.astype(np.float64) - ref.astype(np.float64))))
        assert err == 0.0, f"{name} max_error={err}"
    print("correctness: all variants bitwise-identical to np.matmul ✓")

    # 预热
    for _ in range(5):
        for _, fn in VARIANTS:
            fn(a32, b32)

    # 交错配对计时
    times = {name: [] for name, _ in VARIANTS}
    for rnd in range(ROUNDS):
        for name, fn in VARIANTS:
            t0 = time.perf_counter()
            fn(a32, b32)
            times[name].append((time.perf_counter() - t0) * 1000.0)

    print(f"\n{'variant':<14}{'median':>9}{'mean':>9}{'min':>9}{'p25':>9}{'p75':>9}")
    med = {}
    for name, _ in VARIANTS:
        ts = sorted(times[name])
        m = statistics.median(ts)
        med[name] = m
        print(f"{name:<14}{m:>9.3f}{statistics.mean(ts):>9.3f}{ts[0]:>9.3f}"
              f"{ts[len(ts)//4]:>9.3f}{ts[3*len(ts)//4]:>9.3f}")

    # 配对胜率 vs v0（同轮交错，控制负载漂移）
    print(f"\npaired vs v0_base (win = variant faster that round):")
    names = [n for n, _ in VARIANTS if n != "v0_base"]
    for name in names:
        wins = sum(1 for i in range(ROUNDS) if times[name][i] < times["v0_base"][i])
        diffs = [times["v0_base"][i] - times[name][i] for i in range(ROUNDS)]
        print(f"  {name:<14} wins {wins}/{ROUNDS}  Δmedian {statistics.median(diffs):+.3f}ms")

    print(f"\nload check: ref_matmul median {med['ref_matmul']:.3f}ms (历史带 6.6-7.0)")


if __name__ == "__main__":
    main()
