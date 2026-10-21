#!/usr/bin/env python3
"""pre_bench3.py — iter022 预研 III：einsum 单块 M 规模曲线 + 不对称二分组合。

背景：粒度谱系（einsum 轴，含装配）：full(2048) 7.26 / halves(1024×2) 7.86 /
quarters(512×4) 9.17 —— 每行成本随 M 递减（凸性）。若 t(M) 凸性显著，
不对称二分（如 1792+256、1536+512）块数仍为 2（m_halves 家族内），
但把行从「贵」的小块挪进「便宜」的大块，可能回收拆分税的一部分。

测量：
- t(M)：单块 einsum('jk,ij->ik', b[:2048,:2048... a[:M]) 稳态中位（M 扫描）
- 组合端到端：v(h) = 二分 (h, m-h) + 趁热逐块拷贝（父本装配），h ∈ {512, 768,
  1024(基线), 1280, 1536, 1792, 1984}
- 判据：某 h 的端到端中位比 1024 基线好 ≥0.15ms 且配对胜率 ≥60% → 值得立项；
  否则块规模轴闭合。
"""
import statistics
import time

import numpy as np

np.seterr(divide="ignore", over="ignore", invalid="ignore")

N = 2048
ROUNDS = 25


def ein_half(b32, a_blk):
    return np.einsum('jk,ij->ik', b32, a_blk, optimize=True)


def split2(a32, b32, h):
    """二分 (h, m-h) + 父本装配（预分配 + 趁热逐块拷贝）。"""
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

    # 正确性：所有 h 变体逐位一致
    for h in [512, 768, 1024, 1280, 1536, 1792, 1984]:
        out = split2(A, B, h)
        err = float(np.max(np.abs(out.astype(np.float64) - ref.astype(np.float64))))
        assert err == 0.0, f"h={h} err={err}"
    print("correctness: all split points bitwise-identical ✓")

    # t(M) 单块曲线（稳态）
    print(f"\n{'M':>6}{'t(M) med':>10}{'µs/row':>9}")
    for M in [256, 512, 768, 1024, 1536, 1792, 2048]:
        ts = []
        for _ in range(ROUNDS):
            t0 = time.perf_counter()
            ein_half(B, A[:M])
            ts.append((time.perf_counter() - t0) * 1000.0)
        med = statistics.median(ts)
        print(f"{M:>6}{med:>10.3f}{med/M*1000:>9.2f}")

    # 端到端二分组合（交错配对）
    hs = [512, 768, 1024, 1280, 1536, 1792, 1984]
    fns = {h: (lambda a, b, h=h: split2(a, b, h)) for h in hs}
    fns['ref'] = lambda a, b: np.matmul(a, b)
    for _ in range(4):
        for _, fn in fns.items():
            fn(A, B)
    times = {k: [] for k in fns}
    for _ in range(ROUNDS):
        for k, fn in fns.items():
            t0 = time.perf_counter()
            fn(A, B)
            times[k].append((time.perf_counter() - t0) * 1000.0)

    base = times[1024]
    print(f"\n{'h':>6}{'median':>9}{'Δmed vs 1024':>14}{'win%':>7}")
    for h in hs:
        med = statistics.median(times[h])
        wins = sum(1 for i in range(ROUNDS) if times[h][i] < base[i])
        print(f"{h:>6}{med:>9.3f}{med-statistics.median(base):>+14.3f}{100*wins//ROUNDS:>6}%")
    print(f"{'ref':>6}{statistics.median(times['ref']):>9.3f}"
          f"{'':>14}{'':>7}   (负载带 6.6-7.0)")


if __name__ == "__main__":
    main()
