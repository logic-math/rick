#!/usr/bin/env python3
"""solution.py — matmul 候选（iter002 实验格：naive_loops × np_matmul × col_major × multiprocess）。

target_cell: dispatch_target=np_matmul | impl_strategy=naive_loops
             | memory_layout=col_major | parallelism=multiprocess

结构（相对父本 naive_start.py 保持逐输出元素 (i,j) 双循环粒度不变，只换派发/布局/并行三轴）：
- dispatch_target=np_matmul：内层 k 收缩从纯 Python 标量累加改为逐元素派发
  ``np.matmul(av, bv)``（1-D×1-D 内积入口，显式 np.matmul 调用——非 np.dot /
  非逐行 / 非整阵派发，那些属其它格）；
- memory_layout=col_major：B 经 np.asfortranarray 一次性 F 序预处理，派发前预切
  ``_COLS = list(bf.T)`` 连续列视图（本循环形态按列访问的操作数受益 ~1.9×；
  A 行访问保持 C 序，ascontiguousarray 归一）；
- parallelism=multiprocess：fork×P=min(6,cpu,n) 连续行块子进程直写共享 RawArray
  （零结果 pickling），Semaphore 汇合，返回前 finally terminate 存活者+join 全部
  （bench 残留子进程拒收——保证无残留）；
- 入口先释放旧共享缓冲再分配（iter001 教训：multiprocessing heap 整块复用，
  稳态从第 2 次调用起，bench warmup 吸收首发）。

确定性与边界：
- 块边界与遍历顺序固定 + 每次调用新 RawArray → 同输入双跑逐位一致；
- 矩形 (n,k)×(k,m)、k=0（空内积 np.matmul 结果 0.0 → 全零输出）、n<P 收缩进程数、
  嵌套 list 输入、n/m=0 早退 zeros；
- 全程 float32 不升 fp64；不改动输入数组；返回 C 序 float32 ndarray。
"""
import ctypes
import multiprocessing
import os

import numpy as np

_CTX = multiprocessing.get_context("fork")
_P_MAX = 6

_ROWS = None   # list：n 个 A 行视图（连续，C 序）
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
    cols = list(bf.T)              # m 个连续列视图（F 序转置视图，零拷贝）
    parts = max(1, min(_P_MAX, os.cpu_count() or 1, n))
    blocks = _partition(n, parts)

    out = _CTX.RawArray(ctypes.c_float, n * m)
    _ROWS, _COLS, _OUT = rows, cols, out  # 派发前就位（fork COW 继承）

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
            p.join()  # 收割全部子进程——无残留（bench 残留子进程拒收）
    return np.frombuffer(out, dtype=np.float32, count=n * m).reshape(n, m).copy()
