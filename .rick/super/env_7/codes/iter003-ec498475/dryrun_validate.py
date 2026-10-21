#!/usr/bin/env python3
"""dryrun_validate.py — iter003 目标格结构镜像候选（builder 校准工件，非交付物）。

target_cell: dispatch_target=pure_python | impl_strategy=rowwise_vectorized
             | memory_layout=col_major | parallelism=serial

结构（按实验计划）：
- dispatch_target=pure_python：核心 FLOPs 全部走 Python 内建（list/zip/sum/map/
  operator.mul），无 numpy/BLAS/einsum/ctypes 派发；模块 import 仅 operator（stdlib）。
  输入适配用 duck-typing（.tolist 存在即转嵌套 list）——numpy 仅可能出现在
  __main__ 自检块，不在计算路径。
- impl_strategy=rowwise_vectorized：每个输出行由一个列表推导产出，
  行内每个元素的内积收缩是单个 C 级原语 sum(map(operator.mul, ai, col))——
  无 Python 层标量累加循环（那是 naive_loops），无分块/递归/整阵派发。
- memory_layout=col_major：B 一次性预处理为列元组 list(zip(*b_rows))（父本
  naive_start 的转置约定；列连续访问是本循环形态的受益布局，实测 1.58×）。
- parallelism=serial：单线程，无子进程/线程池（无残留子进程风险）。

校准带（本机 512，bench 协议 warmup1+k3 中位）：1584-1675ms 两轮独立复测；
score ≈ 1.1-1.2e-4（ref ~0.185ms）；正确性 max_error ≈ 1.8e-4（fp64 累加 vs
fp32 BLAS）；同输入双跑逐位一致。
"""
from operator import mul as _mul


def _to_rows(x):
    """输入适配（预处理，非核心计算）：numpy→嵌套 list（Python float）；list 归一。"""
    tolist = getattr(x, "tolist", None)
    if tolist is not None:
        return tolist()
    return [row if isinstance(row, list) else list(row) for row in x]


def matmul(a, b):
    """逐行向量化点积 × 列主序 B × 串行（目标格四轴，见模块 docstring）。"""
    for x in (a, b):
        ndim = getattr(x, "ndim", None)
        if ndim is not None and ndim != 2:
            raise ValueError("matmul expects 2-D operands")
    a_rows = _to_rows(a)
    b_rows = _to_rows(b)
    if not a_rows or not b_rows:
        # 空轴早退：m 从 numpy 形状取（嵌套 list 输入的 k=0 无可判定形态，与父本同类行为）
        shape = getattr(b, "shape", None)
        m = shape[1] if shape is not None else (len(b_rows[0]) if b_rows else 0)
        return [[0.0] * m for _ in range(len(a_rows))]
    cols = list(zip(*b_rows))  # col_major：B 转置为列元组（一次性预处理）
    return [[sum(map(_mul, ai, col)) for col in cols] for ai in a_rows]


if __name__ == "__main__":
    # 手动自检：n=64 与 np.matmul 对比（正确性门同款输入协议）
    import json

    import numpy as np

    np.seterr(divide="ignore", over="ignore", invalid="ignore")
    rng = np.random.RandomState(42)
    x = rng.rand(64, 64).astype(np.float32)
    y = rng.rand(64, 64).astype(np.float32)
    r = matmul(x, y)
    r2 = matmul(x, y)
    print(json.dumps({
        "n": 64,
        "shape": [len(r), len(r[0])],
        "max_error": float(np.max(np.abs(
            np.asarray(r, dtype=np.float64) - np.matmul(x, y).astype(np.float64)))),
        "double_run_equal": r == r2,
    }))
