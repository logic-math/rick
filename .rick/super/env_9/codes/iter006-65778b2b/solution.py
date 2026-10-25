#!/usr/bin/env python3
"""solution.py — matmul 单次整阵 np.einsum optimize-path 派发（iter006，父本 iter003 np_matmul 整阵派发）。

契约（与父本一致，全部保持不动）：
- 暴露 ``matmul(a, b)``：接受嵌套 list 或 numpy array，asarray 转 f32，
  返回 (n, m) ndarray。
- main 的预热/计时/自检 JSON 输出协议与 np.seterr 设置保持不动。

本迭代改动（唯一一处，kernel 内部）：
- ``np.matmul(a32, b32)`` → ``np.einsum('ij,jk->ik', a32, b32, optimize=_EINSUM_PATH)``
  ——本格定义形态（dispatch_target=einsum_path / impl_strategy=full_blas_dispatch）：
  einsum optimize 路径对 'ij,jk->ik' 走 tensordot→np.dot→BLAS sgemm 整阵派发。
- 模块级预计算收缩 path：``np.einsum_path('ij,jk->ik', ..., optimize='greedy')[0]``，
  调用时传 ``optimize=path`` 跳过每次调用的 greedy 路径搜索（输出不变，仅省每调用开销）。
  path 为 ``['einsum_path', (0, 1)]``——2 操作数单步收缩，结构与矩阵规模无关，
  故用 2x2 占位阵预计算即可对任意 (n, k)/(k, m) 生效（实测 n=64/1024/8192 均逐位一致）。
- K 求和序不动：与 np.matmul 同一 BLAS sgemm 调用路径，逐位一致
  （实测 n=64/1024/8192 ``np.array_equal`` 均为 True，max_abs_diff = 0.0）。
- 不预转置 B、不用 out=、不引入 reshape/中间拷贝/额外 dtype 转换。

来源与依据（检索纪律，role=coding 落地点）：
- numpy 官方文档 einsum/optimize：optimize=True 或显式 path 均经 einsum_path 派发；
  2 操作数 'ij,jk->ik' 收缩命中 tensordot 快路径（BLAS），求和序与 np.dot/np.matmul 一致。
- 预检（本机 env_9 venv）：einsum(optimize=path) 与 np.matmul 逐位一致（array_equal=True），
  @8192 warm 单发 450.9/464.7/554.8ms，落 BLAS 平价带。

直接运行（自检，默认 64×64）：
    python3 code/solution.py --n 8192
"""
import json
import time

import numpy as np

# macOS vecLib BLAS 虚假 FP 标志噪声抑制（数值正常；详见 data/ 同注释）。
np.seterr(divide="ignore", over="ignore", invalid="ignore")

# 模块级预计算收缩 path：2 操作数收缩 path（['einsum_path', (0, 1)]）与规模无关，
# 用 2x2 占位阵算一次，每次调用传 optimize=_EINSUM_PATH 跳过 greedy 搜索开销。
_EINSUM_PATH = np.einsum_path(
    "ij,jk->ik",
    np.zeros((2, 2), dtype=np.float32),
    np.zeros((2, 2), dtype=np.float32),
    optimize="greedy",
)[0]


def matmul(a, b):
    """单次整阵 np.einsum optimize-path 派发（tensordot→BLAS gemm）。逐位一致于参考锚。"""
    a32 = np.asarray(a, dtype=np.float32)
    b32 = np.asarray(b, dtype=np.float32)
    return np.einsum("ij,jk->ik", a32, b32, optimize=_EINSUM_PATH)


def main():
    import argparse

    parser = argparse.ArgumentParser(description="solution matmul 自检")
    parser.add_argument("--n", type=int, default=64, help="矩阵规模（默认 64）")
    args = parser.parse_args()

    rng = np.random.RandomState(42)
    a = rng.rand(args.n, args.n).astype(np.float32)
    b = rng.rand(args.n, args.n).astype(np.float32)
    matmul(a, b)  # 预热
    t0 = time.perf_counter()
    c = matmul(a, b)
    dt_ms = (time.perf_counter() - t0) * 1000.0
    ref = np.matmul(a, b)
    max_err = float(np.max(np.abs(c.astype(np.float64) - ref.astype(np.float64))))
    print(json.dumps({
        "ref": False,
        "n": args.n,
        "impl": "full_blas_dispatch_einsum_path",
        "runtime_ms": round(dt_ms, 3),
        "max_error_vs_matmul": max_err,
    }))


if __name__ == "__main__":
    main()
