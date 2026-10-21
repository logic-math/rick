#!/usr/bin/env python3
"""pre_bench4.py — iter022 预研 IV：不对称二分 h 轴终局裁决（新进程、40 轮、聚焦）。

pre_bench3 信号：h=768 配对 Δ中位 −0.078ms / 胜率 68%（25 轮）——噪声边缘。
本裁决：40 轮交错配对，聚焦 h ∈ {640, 704, 768, 896, 1024(基线)}，
判据：某 h Δ中位 ≤ −0.05 且胜率 ≥ 60% → 立项；否则 h 轴闭合、格地板终局确认。
"""
import statistics
import time

import numpy as np

np.seterr(divide="ignore", over="ignore", invalid="ignore")

N = 2048
ROUNDS = 40


def ein_half(b32, a_blk):
    return np.einsum('jk,ij->ik', b32, a_blk, optimize=True)


def split2(a32, b32, h):
    m = a32.shape[0]
    c = np.empty((m,) + b32.shape[1:], dtype=np.float32)
    r1 = ein_half(b32, a32[:h])
    c[:h] = r1
    r2 = ein_half(b32, a32[h:])
    c[h:] = r2
    return c


def main():
    rng = np.random.RandomState(42)
    A = rng.rand(N, N).astype(np.float32)
    B = rng.rand(N, N).astype(np.float32)
    ref = np.matmul(A, B)
    for h in [640, 704, 768, 896, 1024]:
        out = split2(A, B, h)
        assert float(np.max(np.abs(out.astype(np.float64) - ref.astype(np.float64)))) == 0.0
    print("correctness ✓")

    hs = [640, 704, 768, 896, 1024]
    fns = {h: (lambda a, b, h=h: split2(a, b, h)) for h in hs}
    fns['ref'] = lambda a, b: np.matmul(a, b)
    for _ in range(6):
        for _, fn in fns.items():
            fn(A, B)
    times = {k: [] for k in fns}
    for _ in range(ROUNDS):
        for k, fn in fns.items():
            t0 = time.perf_counter()
            fn(A, B)
            times[k].append((time.perf_counter() - t0) * 1000.0)

    base = times[1024]
    print(f"\n{'h':>6}{'median':>9}{'Δmed':>9}{'win%':>7}{'p25':>8}{'p75':>8}")
    for h in hs:
        ts = sorted(times[h])
        med = statistics.median(ts)
        wins = sum(1 for i in range(ROUNDS) if times[h][i] < base[i])
        print(f"{h:>6}{med:>9.3f}{med-statistics.median(base):>+9.3f}"
              f"{100*wins//ROUNDS:>6}%{ts[ROUNDS//4]:>8.3f}{ts[3*ROUNDS//4]:>8.3f}")
    print(f"{'ref':>6}{statistics.median(times['ref']):>9.3f}   (负载带 6.6-7.0)")


if __name__ == "__main__":
    main()
