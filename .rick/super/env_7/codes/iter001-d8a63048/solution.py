#!/usr/bin/env python3
"""solution.py — matmul 候选（iter001 实验格：rowwise einsum × col_major × multiprocess）。

target_cell: dispatch_target=einsum_path | impl_strategy=rowwise_vectorized
             | memory_layout=col_major | parallelism=multiprocess

结构（按实验计划）：
- 入口 np.asarray(a, float32)；B 一次性预处理为 F 序（np.asfortranarray，col_major 轴）；
- 行块划分：P = min(6, os.cpu_count()) 个连续行块，n < P 时收缩进程数并跳过空块；
- 每块一个 fork 子进程（multiprocessing.get_context('fork')）：子进程内 Python 循环逐行
  np.einsum('j,jk->k', A[i], BF)（默认 optimize=False——单行收缩不走 BLAS 派发），
  结果直接写入共享 _CTX.RawArray(c_float, n*m) 的 frombuffer 视图（无结果 pickling）；
- 子进程写完 release Semaphore；父进程 acquire 全部后在 finally 中 terminate+join
  全部子进程（bench 有残留子进程拒收——保证无残留），最后 frombuffer().reshape(n,m)
  .copy() 以 C 序 float32 ndarray 返回。

确定性与边界：
- 块边界固定、逐行 einsum 顺序固定 → 同输入双跑逐位一致；
- 支持矩形 (n,k)×(k,m)；全程 float32 不升 fp64；
- 异常路径 terminate+join 兜底（子进程 finally release，父进程不悬挂）。
"""
import ctypes
import multiprocessing
import os

import numpy as np

_CTX = multiprocessing.get_context("fork")
_P_MAX = 6  # 计划指定：P = min(6, os.cpu_count())

# fork 继承的模块级工作区（父进程在派生前写入；子进程读 A/BF、只写 OUT 自己的行块）
_A = None    # (n, k) float32
_BF = None   # (k, m) float32 F 序（col_major 轴）
_OUT = None  # 共享 RawArray(c_float, n*m)——匿名共享内存，父子同视图


def _partition(n, parts):
    """n 行划分为 parts 个连续行块（均衡大小、顺序固定）；返回非空 (lo, hi) 列表。"""
    base, rem = divmod(n, parts)
    bounds = []
    start = 0
    for i in range(parts):
        size = base + (1 if i < rem else 0)
        if size > 0:  # 跳过空块（n < parts 收缩后仍可能有空块，双保险）
            bounds.append((start, start + size))
        start += size
    return bounds


def _worker(lo, hi, nrows, ncols, sem):
    """子进程：[lo, hi) 行逐行 einsum 写共享输出缓冲，完成后 release（finally 兜底）。"""
    try:
        out = np.frombuffer(_OUT, dtype=np.float32).reshape(nrows, ncols)
        a = _A
        bf = _BF
        einsum = np.einsum
        for i in range(lo, hi):
            out[i] = einsum("j,jk->k", a[i], bf)
    finally:
        sem.release()


def matmul(a, b):
    """行块 fork 并行 × 逐行 einsum × F 序 B（实验格 cell，见模块 docstring）。"""
    global _A, _BF, _OUT
    # 先释放上一轮共享缓冲再分配新的：上一轮子进程已在 finally 中全部 join，无存活
    # 读者，此时清引用可让 multiprocessing heap 整块复用刚释放的 block——否则新分配
    # 发生在旧 _OUT 仍被持有时，heap 被迫每次 mmap 新 arena（实测前 2 次调用 alloc
    # 10.5ms/3.4ms，稳态才 0.1ms；先释放后分配可从第 2 次调用起即达稳态）。
    _A = _BF = _OUT = None
    a_arr = np.asarray(a, dtype=np.float32)
    b_arr = np.asarray(b, dtype=np.float32)
    if a_arr.ndim != 2 or b_arr.ndim != 2:
        raise ValueError("matmul expects 2-D operands")
    n, k = a_arr.shape
    k2, m = b_arr.shape
    if k != k2:
        raise ValueError("shape mismatch: (%d,%d)x(%d,%d)" % (n, k, k2, m))
    if n == 0 or m == 0:
        return np.zeros((n, m), dtype=np.float32)

    bf = np.asfortranarray(b_arr)  # col_major 轴：一次性 F 序预处理
    parts = max(1, min(_P_MAX, os.cpu_count() or 1, n))  # n < P 时收缩进程数
    blocks = _partition(n, parts)

    out = _CTX.RawArray(ctypes.c_float, n * m)
    _A, _BF, _OUT = a_arr, bf, out  # 派生前就位，fork 子进程直接继承

    sem = _CTX.Semaphore(0)
    procs = []
    try:
        for lo, hi in blocks:
            p = _CTX.Process(target=_worker, args=(lo, hi, n, m, sem), daemon=True)
            p.start()
            procs.append(p)
        for _ in range(len(procs)):
            sem.acquire()  # 全部子进程写完才继续（超时中断路径由 finally 兜底清理）
    finally:
        for p in procs:
            if p.is_alive():
                p.terminate()
        for p in procs:
            p.join()  # 收割全部子进程——无残留（bench 残留清点拒收）
    return np.frombuffer(out, dtype=np.float32, count=n * m).reshape(n, m).copy()


if __name__ == "__main__":
    # 手动自检：n=64 与 np.matmul 对比（正确性门同款输入协议）
    import json

    # macOS vecLib 对 float32 matmul 报虚假 FP 标志（数值正常；门禁同款抑制）
    np.seterr(divide="ignore", over="ignore", invalid="ignore")
    rng = np.random.RandomState(42)
    x = rng.rand(64, 64).astype(np.float32)
    y = rng.rand(64, 64).astype(np.float32)
    r = matmul(x, y)
    print(json.dumps({
        "n": 64,
        "shape": list(r.shape),
        "dtype": str(r.dtype),
        "max_error": float(np.max(np.abs(
            r.astype(np.float64) - np.matmul(x, y).astype(np.float64)))),
    }))
