#!/usr/bin/env python3
"""debug A/B 探针：同窗对照 顺序两半 vs 并发两半（当前候选形态）。

目的：区分「机制失效/过订阅反噬」（并发 ≥ 顺序）与「机制有效」（并发显著低于顺序）。
交错 5 轮，n=8192 seed=42，与 iter16 探针同型。
"""
import time

import numpy as np

np.seterr(divide="ignore", over="ignore", invalid="ignore")

import threading


def matmul_seq(a, b):
    a32 = np.asarray(a, dtype=np.float32)
    b32 = np.asarray(b, dtype=np.float32)
    m = a32.shape[0]
    n = b32.shape[1]
    mid = m // 2
    c = np.empty((m, n), dtype=np.float32)
    np.matmul(a32[:mid], b32, out=c[:mid])
    np.matmul(a32[mid:], b32, out=c[mid:])
    return c


def matmul_conc(a, b):
    a32 = np.asarray(a, dtype=np.float32)
    b32 = np.asarray(b, dtype=np.float32)
    m = a32.shape[0]
    n = b32.shape[1]
    mid = m // 2
    c = np.empty((m, n), dtype=np.float32)

    def lo():
        np.seterr(divide="ignore", over="ignore", invalid="ignore")
        np.matmul(a32[:mid], b32, out=c[:mid])

    def hi():
        np.seterr(divide="ignore", over="ignore", invalid="ignore")
        np.matmul(a32[mid:], b32, out=c[mid:])

    ts = [threading.Thread(target=lo), threading.Thread(target=hi)]
    try:
        for t in ts:
            t.start()
    finally:
        for t in ts:
            t.join()
    return c


def main():
    n = 8192
    rng = np.random.RandomState(42)
    a32 = rng.rand(n, n).astype(np.float32)
    b32 = rng.rand(n, n).astype(np.float32)

    # 整阵参考（窗口定标锚）
    np.matmul(a32, b32)  # warmup
    t0 = time.perf_counter()
    np.matmul(a32, b32)
    t_full = (time.perf_counter() - t0) * 1e3

    matmul_seq(a32, b32)
    matmul_conc(a32, b32)  # warmup both

    seq_t, conc_t = [], []
    for _ in range(5):
        for name, fn, acc in (("S", matmul_seq, seq_t), ("C", matmul_conc, conc_t)):
            t0 = time.perf_counter()
            c = fn(a32, b32)
            acc.append((time.perf_counter() - t0) * 1e3)

    # bitwise 校验
    ref = np.matmul(a32, b32)
    ok_seq = bool(np.array_equal(matmul_seq(a32, b32), ref))
    ok_conc = bool(np.array_equal(matmul_conc(a32, b32), ref))

    print(f"full_matrix_once_ms={t_full:.1f}")
    print(f"seq_ms={[round(x, 1) for x in seq_t]} median={sorted(seq_t)[2]:.1f}")
    print(f"conc_ms={[round(x, 1) for x in conc_t]} median={sorted(conc_t)[2]:.1f}")
    print(f"bitwise: seq={ok_seq} conc={ok_conc}")
    print(f"conc_minus_seq_median={sorted(conc_t)[2] - sorted(seq_t)[2]:.1f}ms")


if __name__ == "__main__":
    main()
