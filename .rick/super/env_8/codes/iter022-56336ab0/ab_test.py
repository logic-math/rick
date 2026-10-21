#!/usr/bin/env python3
"""ab_test.py — iter022 A/B 交裁决：不对称 h=7m//16 vs 父本对称 h=m//2。

协议（与 pre_bench3/4 同款）：同会话同进程、round-robin 交错配对、40 轮，
配对差控制会话漂移。父本结构同会话重测作基线（不引用历史数字）。
"""
import statistics
import time

import numpy as np

np.seterr(divide="ignore", over="ignore", invalid="ignore")

N = 2048
ROUNDS = 40


def split2(a32, b32, h):
    m = a32.shape[0]
    c = np.empty((m,) + b32.shape[1:], dtype=np.float32)
    r1 = np.einsum('jk,ij->ik', b32, a32[:h], optimize=True)
    c[:h] = r1
    r2 = np.einsum('jk,ij->ik', b32, a32[h:], optimize=True)
    c[h:] = r2
    return c


def cand(a32, b32):  # 本迭代候选：h = 7m//16
    return split2(a32, b32, 7 * a32.shape[0] // 16)


def parent(a32, b32):  # 父本 iter013：h = m//2
    return split2(a32, b32, a32.shape[0] // 2)


def main():
    rng = np.random.RandomState(42)
    A = rng.rand(N, N).astype(np.float32)
    B = rng.rand(N, N).astype(np.float32)
    ref = np.matmul(A, B)

    o_c, o_p = cand(A, B), parent(A, B)
    assert float(np.max(np.abs(o_c.astype(np.float64) - ref.astype(np.float64)))) == 0.0
    assert float(np.max(np.abs(o_p.astype(np.float64) - ref.astype(np.float64)))) == 0.0
    print("correctness: both max_error=0.0 vs np.matmul")

    fns = {'asym7_16': cand, 'parent_m2': parent, 'ref': lambda a, b: np.matmul(a, b)}
    for _ in range(6):
        for fn in fns.values():
            fn(A, B)
    times = {k: [] for k in fns}
    for _ in range(ROUNDS):
        for k, fn in fns.items():
            t0 = time.perf_counter()
            fn(A, B)
            times[k].append((time.perf_counter() - t0) * 1000.0)

    base = times['parent_m2']
    print(f"\n{'variant':>10}{'median':>9}{'Δmed':>9}{'win%':>7}{'p25':>8}{'p75':>8}")
    for k in ['asym7_16', 'parent_m2', 'ref']:
        ts = sorted(times[k])
        med = statistics.median(ts)
        dmed = med - statistics.median(base) if k != 'parent_m2' else 0.0
        wins = (100 * sum(1 for i in range(ROUNDS) if times[k][i] < base[i]) // ROUNDS
                if k != 'parent_m2' else 100)
        print(f"{k:>10}{med:>9.3f}{dmed:>+9.3f}{wins:>6}%"
              f"{ts[ROUNDS//4]:>8.3f}{ts[3*ROUNDS//4]:>8.3f}")


if __name__ == "__main__":
    main()
