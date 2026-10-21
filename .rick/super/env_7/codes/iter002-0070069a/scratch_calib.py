#!/usr/bin/env python3
"""scratch_calib.py — iter002 builder 校准测量（目标格 np_matmul×naive_loops×col_major×multiprocess）。

一次性校准脚本，不进候选。测量：
  1. ref：整阵 np.matmul（基线 ~0.19ms）
  2. per-call 开销：np.matmul(1-D,1-D) 连续 vs 跨步 vs np.dot（µs/call）
  3. 串行全循环：col_major（B F 序预切连续列视图） vs row_major（B C 序跨步列视图）
  4. mp 版（fork×6，目标结构镜像）：call1/2/3 分相
  5. noop fork×6 + semaphore + join（进程开销）
  6. asfortranarray 转换成本
  7. mp 结果 vs np.matmul 正确性
"""
import ctypes
import json
import multiprocessing
import os
import time

import numpy as np

np.seterr(divide="ignore", over="ignore", invalid="ignore")
_CTX = multiprocessing.get_context("fork")
N = 512
rng = np.random.RandomState(42)
A = rng.rand(N, N).astype(np.float32)
B = rng.rand(N, N).astype(np.float32)


def timed_ms(fn, once=False):
    ts = []
    reps = 1 if once else 3
    for _ in range(reps):
        t0 = time.perf_counter()
        fn()
        ts.append((time.perf_counter() - t0) * 1e3)
    return sorted(ts)[reps // 2], ts


out = {"cpu_count": os.cpu_count()}

# ---- 1. ref ----
np.matmul(A, B)
ref_ms, _ = timed_ms(lambda: np.matmul(A, B))
out["ref_matmul_ms"] = round(ref_ms, 4)

# ---- 2. per-call 开销 ----
ai = A[0]
bF = np.asfortranarray(B)
bj_c = bF.T[0]     # 连续视图（col_major）
bj_s = B.T[0]      # 跨步视图（row_major）
reps = 20000
np.matmul(ai, bj_c); np.dot(ai, bj_c); np.matmul(ai, bj_s)

t0 = time.perf_counter()
for _ in range(reps):
    np.matmul(ai, bj_c)
out["percall_matmul_contig_us"] = round((time.perf_counter() - t0) / reps * 1e6, 3)

t0 = time.perf_counter()
for _ in range(reps):
    np.dot(ai, bj_c)
out["percall_dot_contig_us"] = round((time.perf_counter() - t0) / reps * 1e6, 3)

t0 = time.perf_counter()
for _ in range(reps):
    np.matmul(ai, bj_s)
out["percall_matmul_strided_us"] = round((time.perf_counter() - t0) / reps * 1e6, 3)

t0 = time.perf_counter()
for _ in range(reps):
    B[:, 0]
out["percall_colslice_us"] = round((time.perf_counter() - t0) / reps * 1e6, 3)

# ---- 3. 串行全循环（512×512，每元素一次 np.matmul）----
def serial_cm():
    bF2 = np.asfortranarray(B)
    rows = list(A)
    cols = list(bF2.T)
    flat = np.empty(N * N, dtype=np.float32)
    mm = np.matmul
    k = 0
    for av in rows:
        for bv in cols:
            flat[k] = mm(av, bv)
            k += 1
    return flat.reshape(N, N)


def serial_rm():
    rows = list(A)
    cols = list(B.T)  # 跨步列视图
    flat = np.empty(N * N, dtype=np.float32)
    mm = np.matmul
    k = 0
    for av in rows:
        for bv in cols:
            flat[k] = mm(av, bv)
            k += 1
    return flat.reshape(N, N)


serial_cm_ms, _ = timed_ms(serial_cm, once=True)
serial_rm_ms, _ = timed_ms(serial_rm, once=True)
out["serial_col_major_ms"] = round(serial_cm_ms, 2)
out["serial_row_major_ms"] = round(serial_rm_ms, 2)
out["layout_ratio_rm_over_cm"] = round(serial_rm_ms / serial_cm_ms, 3)
out["per_element_ns"] = round(serial_cm_ms * 1e6 / (N * N), 1)

# ---- 4. mp 版（目标结构镜像：fork×6，共享 RawArray，semaphore 汇合）----
_G = {"ROWS": None, "COLS": None, "OUT": None}


def _worker(lo, hi, m, sem):
    try:
        flat = np.frombuffer(_G["OUT"], dtype=np.float32)
        rows = _G["ROWS"]
        cols = _G["COLS"]
        mm = np.matmul
        k = lo * m
        for i in range(lo, hi):
            av = rows[i]
            for bv in cols:
                flat[k] = mm(av, bv)
                k += 1
    finally:
        sem.release()


def mp_cm():
    # iter001 教训：入口先释放旧共享缓冲（mp heap 整块复用，避免 arena 反复 mmap）
    _G["ROWS"] = _G["COLS"] = _G["OUT"] = None
    bF2 = np.asfortranarray(B)
    rows = list(A)
    cols = list(bF2.T)
    raw = _CTX.RawArray(ctypes.c_float, N * N)
    _G["ROWS"], _G["COLS"], _G["OUT"] = rows, cols, raw
    P = 6
    base, rem = divmod(N, P)
    sem = _CTX.Semaphore(0)
    procs = []
    lo = 0
    try:
        for i in range(P):
            size = base + (1 if i < rem else 0)
            if size == 0:
                continue
            p = _CTX.Process(target=_worker, args=(lo, lo + size, N, sem),
                             daemon=True)
            p.start()
            procs.append(p)
            lo += size
        for _ in range(len(procs)):
            sem.acquire()
    finally:
        for p in procs:
            if p.is_alive():
                p.terminate()
        for p in procs:
            p.join()
    return np.frombuffer(raw, dtype=np.float32, count=N * N).reshape(N, N).copy()


mp_calls = []
for _ in range(4):
    t0 = time.perf_counter()
    R = mp_cm()
    mp_calls.append(round((time.perf_counter() - t0) * 1e3, 2))
out["mp_col_major_calls_ms"] = mp_calls
out["mp_col_major_steady_ms"] = sorted(mp_calls[1:])[1]

# ---- 5. noop fork×6 ----
def _noop(sem):
    sem.release()


def noop_forks():
    sem = _CTX.Semaphore(0)
    procs = []
    try:
        for _ in range(6):
            p = _CTX.Process(target=_noop, args=(sem,), daemon=True)
            p.start()
            procs.append(p)
        for _ in range(6):
            sem.acquire()
    finally:
        for p in procs:
            if p.is_alive():
                p.terminate()
        for p in procs:
            p.join()


noop_ms, _ = timed_ms(noop_forks)
out["fork_join_noop_x6_ms"] = round(noop_ms, 2)

# ---- 6. asfortranarray 成本 ----
t0 = time.perf_counter()
np.asfortranarray(B)
out["asfortranarray_512_ms"] = round((time.perf_counter() - t0) * 1e3, 3)

# ---- 7. 正确性（mp 结果 vs np.matmul）----
ref = np.matmul(A, B)
err = float(np.max(np.abs(R.astype(np.float64) - ref.astype(np.float64))))
out["mp_max_error_vs_ref"] = err
R2 = mp_cm()
out["mp_double_run_bitwise"] = bool(np.array_equal(R2, R))

out["score_prediction_steady"] = round(ref_ms / out["mp_col_major_steady_ms"], 6)
out["serial_over_mp_speedup"] = round(serial_cm_ms / out["mp_col_major_steady_ms"], 2)
out["mp_overhead_share"] = round(noop_ms / out["mp_col_major_steady_ms"], 3)

print(json.dumps(out, ensure_ascii=False, indent=2))
