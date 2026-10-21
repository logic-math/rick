#!/usr/bin/env python3
"""dryrun_validate.py — builder 侧校准工件（iter002 目标格 np_matmul×naive_loops×col_major×multiprocess）。

用途：用官方门禁协议（env/data/test_correctness.py + env/data/bench.py）端到端验证
实验计划的预期带。coding 按计划实现 solution.py（结构与此镜像一致，可复用）。

结构（与 scratch_calib.py 的 mp_cm 相同）：
- naive_loops × np_matmul：逐输出元素 (i,j) 双循环，k 收缩逐元素派发 np.matmul(1-D, 1-D)
  （内积入口；非 np.dot / 非逐行 / 非整阵）；
- col_major：B 经 np.asfortranarray 一次性 F 序预处理（本循环形态下按列访问的操作数），
  派生前预切 _COLS = list(bf.T) 连续列视图；A 行访问保持 C 序（ascontiguousarray 归一）；
- multiprocess：fork×P=min(6,cpu,n) 连续行块子进程直写共享 RawArray（零结果 pickling），
  Semaphore 汇合，返回前 finally terminate+join 全部子进程；
- 入口先释放旧共享缓冲（iter001 教训：mp heap 整块复用，稳态从第 2 次调用起）。
"""
import ctypes
import multiprocessing
import os

import numpy as np

_CTX = multiprocessing.get_context("fork")
_P_MAX = 6

_ROWS = None   # list：n 个 A 行视图（连续）
_COLS = None   # list：m 个 B 列视图（F 序转置视图，连续）
_OUT = None    # 共享 RawArray(c_float, n*m)


def _partition(n, parts):
    """n 行划分为 parts 个连续行块（均衡、顺序固定）；返回非空 (lo, hi) 列表。"""
    base, rem = divmod(n, parts)
    bounds = []
    start = 0
    for i in range(parts):
        size = base + (1 if i < rem else 0)
        if size > 0:
            bounds.append((start, start + size))
        start += size
    return bounds


def _worker(lo, hi, m, sem):
    """子进程：[lo, hi) 行逐元素 np.matmul 内积写共享输出，完成后 release（finally 兜底）。"""
    try:
        flat = np.frombuffer(_OUT, dtype=np.float32)
        rows = _ROWS
        cols = _COLS
        mm = np.matmul
        k = lo * m
        for i in range(lo, hi):
            av = rows[i]
            for bv in cols:
                flat[k] = mm(av, bv)
                k += 1
    finally:
        sem.release()


def matmul(a, b):
    """逐元素 np.matmul × F 序 B × fork 行块并行（目标格四轴，见模块 docstring）。"""
    global _ROWS, _COLS, _OUT
    # 先释放上一轮共享缓冲再分配（子进程已全 join 无存活读者；mp heap 整块复用）
    _ROWS = _COLS = _OUT = None
    a_arr = np.ascontiguousarray(np.asarray(a, dtype=np.float32))
    b_arr = np.asarray(b, dtype=np.float32)
    if a_arr.ndim != 2 or b_arr.ndim != 2:
        raise ValueError("matmul expects 2-D operands")
    n, k = a_arr.shape
    k2, m = b_arr.shape
    if k != k2:
        raise ValueError("shape mismatch: (%d,%d)x(%d,%d)" % (n, k, k2, m))
    if n == 0 or m == 0:
        return np.zeros((n, m), dtype=np.float32)

    bf = np.asfortranarray(b_arr)  # col_major：列访问操作数 F 序（一次性）
    rows = list(a_arr)             # n 个连续行视图
    cols = list(bf.T)              # m 个连续列视图（F 序转置视图）
    parts = max(1, min(_P_MAX, os.cpu_count() or 1, n))
    blocks = _partition(n, parts)

    out = _CTX.RawArray(ctypes.c_float, n * m)
    _ROWS, _COLS, _OUT = rows, cols, out  # 派生前就位（fork COW 继承）

    sem = _CTX.Semaphore(0)
    procs = []
    try:
        for lo, hi in blocks:
            p = _CTX.Process(target=_worker, args=(lo, hi, m, sem), daemon=True)
            p.start()
            procs.append(p)
        for _ in range(len(procs)):
            sem.acquire()
    finally:
        for p in procs:
            if p.is_alive():
                p.terminate()
        for p in procs:
            p.join()  # 收割全部子进程——无残留（bench 残留清点拒收）
    return np.frombuffer(out, dtype=np.float32, count=n * m).reshape(n, m).copy()
