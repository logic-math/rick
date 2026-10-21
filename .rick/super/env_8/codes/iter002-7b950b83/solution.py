#!/usr/bin/env python3
"""solution.py — einsum_path|full_blas_dispatch：单次整阵 canonical einsum 派发。

迭代 2（codes/iter002-7b950b83）· 父本 naive_start.py（rowwise np.dot 143ms）。

实现（一行核心）：
    return np.einsum('ij,jk->ik', a32, b32, optimize=True)

派发解剖（numpy 2.0.2 _core/einsumfunc.py:950 起源码确认 + 本机实测）：
- 2 操作数时 path 固定为 [(0,1)]，逆序排序为 (1,0)——`tmp_operands = [operands.pop(x) for
  x in (1,0)]` 得 [b, a]，einsum_str 重排为 'jk,ij->ik'。
- BLAS 分支：tensordot(b, a, axes=((0,),(1,)))——转置视图零拷贝（dot(b.T 视图, a.T 视图)），
  即 T-标志 gemm；尾部 c_einsum('ki->ik') 仅返回转置视图（np.shares_memory=True，
  无 16MB 拷贝，实测 ~0.0002ms）。
- 输出为 F-布局，但与 np.matmul 同底 BLAS、同一 K 求和序 → 逐位一致（max_error=0.0，
  64/2048 双 seed、嵌套 list、非方阵、双跑一致均实测通过）。
- canonical 拼写（'ij,jk->ik'）走 T-标志 gemm；swapped 拼写（einsum('jk,ij->ik', b, a)）
  走 C-标志 gemm——本机负载态 canonical 稳定快 0.2-0.6ms（见 ab_notes.md）。

远离全部已知死族：不用 K 分块/Strassen/转置后 gemv/降精度/多进程/多线程/out=（einsum
out= 实测 16.8ms 病态）。

契约（与 naive_start.py 一致）：
- 暴露 ``matmul(a, b)``：接受二维数组/嵌套 list，返回 (n, m) float32 矩阵积。
- 模块级 np.seterr 抑制 macOS vecLib 虚假 FP 标志噪声（数值正常）。

直接运行（自检，默认 64×64；2048 单次 ~7ms）：
    python3 solution.py --n 2048
"""
import json
import time

import numpy as np

# macOS vecLib BLAS 虚假 FP 标志噪声抑制（数值正常；详见 data/ 同注释）。
np.seterr(divide="ignore", over="ignore", invalid="ignore")


def matmul(a, b):
    """整阵 canonical einsum 派发（T-标志 gemm）。逐位一致于 np.matmul（求和序不变）。"""
    a32 = np.asarray(a, dtype=np.float32)
    b32 = np.asarray(b, dtype=np.float32)
    return np.einsum('ij,jk->ik', a32, b32, optimize=True)


def main():
    import argparse

    parser = argparse.ArgumentParser(description="einsum canonical 派发自检")
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
        "impl": "einsum_canonical_full_blas_dispatch",
        "runtime_ms": round(dt_ms, 3),
        "max_error_vs_matmul": max_err,
    }))


if __name__ == "__main__":
    main()
