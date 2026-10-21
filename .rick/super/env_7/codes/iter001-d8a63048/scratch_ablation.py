#!/usr/bin/env python3
"""scratch_ablation.py — 一次性校准消融（实验计划 plan_step 5；不进候选，diff_path=solution.py）。

目的：量化本实验格（einsum_path × rowwise × col_major × multiprocess）的两个可证伪载荷：
  ① fork 建进程+join 开销（预期 ~7ms 主导端到端）；
  ② F 序 B（col_main 轴）对逐行 einsum 的布局罚分（预期 serial ~1.5×，mp 并行仅回收部分）。

计时协议（对齐 bench.py）：warmup 1 + 5 次计时取中位数，n=512，float32，seed=42。
"""
import ctypes
import importlib.util
import json
import os
import time

import numpy as np

import multiprocessing

# macOS vecLib 虚假 FP 标志噪声抑制（门禁同款）
np.seterr(divide="ignore", over="ignore", invalid="ignore")

_CTX = multiprocessing.get_context("fork")
N = 512
P = 6  # 与候选一致：min(6, cpu_count)=6（本机 cpu_count=18）

# 模块级工作区（与 solution.py 同构：fork 派生前写入，子进程直接继承）
_A = None
_B = None
_OUT = None


def _timed(fn, k=5, warmup=1):
    for _ in range(warmup):
        fn()
    ts = []
    for _ in range(k):
        t0 = time.perf_counter()
        fn()
        ts.append((time.perf_counter() - t0) * 1000.0)
    ts.sort()
    return ts[len(ts) // 2]


def _worker(lo, hi, nrows, ncols, sem):
    """子进程：[lo, hi) 行逐行 einsum 写共享缓冲（与候选同构）。"""
    try:
        out = np.frombuffer(_OUT, dtype=np.float32).reshape(nrows, ncols)
        a = _A
        b = _B
        einsum = np.einsum
        for i in range(lo, hi):
            out[i] = einsum("j,jk->k", a[i], b)
    finally:
        sem.release()


def _mp_blocks(a, B):
    """mp 结构（与候选同构：行块 × fork × 共享 RawArray × Semaphore）。"""
    global _A, _B, _OUT
    o = _CTX.RawArray(ctypes.c_float, N * N)
    _A, _B, _OUT = a, B, o
    sem = _CTX.Semaphore(0)
    base, rem = divmod(N, P)
    procs = []
    try:
        start = 0
        for i in range(P):
            size = base + (1 if i < rem else 0)
            lo, hi = start, start + size
            start = hi
            p = _CTX.Process(target=_worker, args=(lo, hi, N, N, sem), daemon=True)
            p.start()
            procs.append(p)
        for _ in range(len(procs)):
            sem.acquire()
    finally:
        for p in procs:
            if p.is_alive():
                p.terminate()
        for p in procs:
            p.join()
    return np.frombuffer(o, dtype=np.float32, count=N * N).reshape(N, N).copy()


def main():
    rng = np.random.RandomState(42)
    a = rng.rand(N, N).astype(np.float32)
    b = rng.rand(N, N).astype(np.float32)
    expected = np.matmul(a, b)
    bF = np.asfortranarray(b)

    # ---- serial rowwise einsum：C 序 vs F 序（布局罚分） ----
    def serial(B):
        out = np.empty((N, N), dtype=np.float32)
        einsum = np.einsum
        for i in range(N):
            out[i] = einsum("j,jk->k", a[i], B)
        return out

    r_c = serial(b)
    r_f = serial(bF)
    ms_c = _timed(lambda: serial(b))
    ms_f = _timed(lambda: serial(bF))

    # ---- fork+join 纯开销（6 个 noop 子进程，无工作载荷） ----
    def noop():
        pass

    def fork_only():
        ps = [_CTX.Process(target=noop, daemon=True) for _ in range(P)]
        for p in ps:
            p.start()
        for p in ps:
            p.join()

    ms_fork = _timed(fork_only)

    # ---- mp 候选本体（F 序 B，即 solution.py 的行为） ----
    here = os.path.dirname(os.path.abspath(__file__))
    spec = importlib.util.spec_from_file_location("solution", os.path.join(here, "solution.py"))
    mod = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(mod)
    r_mp = mod.matmul(a, b)
    ms_mp = _timed(lambda: mod.matmul(a, b))

    # ---- 附加诊断：同 mp 结构 + C 序 B（隔离「并行收益」与「布局罚分」） ----
    r_mp_c = _mp_blocks(a, b)
    ms_mp_c = _timed(lambda: _mp_blocks(a, b))

    err = lambda r: float(np.max(np.abs(r.astype(np.float64) - expected.astype(np.float64))))
    report = {
        "n": N, "p": P,
        "serial_einsum_B_C_ms": round(ms_c, 3),
        "serial_einsum_B_F_ms": round(ms_f, 3),
        "fork_join_noop_x6_ms": round(ms_fork, 3),
        "mp_einsum_B_F_ms": round(ms_mp, 3),
        "mp_einsum_B_C_ms": round(ms_mp_c, 3),
        "max_error_serial_C": err(r_c),
        "max_error_serial_F": err(r_f),
        "max_error_mp_F": err(r_mp),
        "max_error_mp_C": err(r_mp_c),
    }
    # 派生量：布局罚分、fork 开销占比、并行回收率
    report["layout_penalty_ms"] = round(ms_f - ms_c, 3)
    report["layout_penalty_ratio"] = round(ms_f / ms_c, 3)
    report["mp_overhead_vs_work_ms"] = round(ms_mp - ms_f / P, 3)  # 端到端 - 并行化纯工作
    report["mp_overhead_share"] = round((ms_mp - ms_f / P) / ms_mp, 3)
    report["mp_F_vs_serial_C"] = round(ms_mp / ms_c, 3)
    report["f_penalty_recovered_ms"] = round(ms_f - ms_mp, 3)  # mp 相对 serial F 序回收的罚分
    print(json.dumps(report, ensure_ascii=False))


if __name__ == "__main__":
    main()
