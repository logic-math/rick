#!/usr/bin/env python3
"""solution.py — np_matmul|rowwise_vectorized：逐行向量化 np.matmul 派发。

迭代 6（codes/iter006-4c29c00a）· 父本 iter003-93ddaebc（np_matmul|full_blas_dispatch，
记录基线 6.859ms）。

单变量结构实验：派发入口保持 np.matmul 不变，只把 impl_strategy 从整阵单次派发
改为逐行向量化——预分配 out 后循环 n 次（1-D 行 × 2-D B，不预转置 B）：

    out = np.empty((n, b32.shape[1]), np.float32)
    for i in range(n):
        out[i] = np.matmul(a32[i], b32)

- 每行派发为 (1,K)×(K,M) gufunc gemm（N=1），与整阵 gemm 同 K 求和序
  → @2048 与 np.matmul 逐位一致（max_error=0.0；预研已实证 1-D×2-D 不转置
  派发与整阵 gemm 同求和序）；@64 ~7.6e-06（vecLib 小规模路径差异，
  远低于 1e-3 容差）。
- 结构定位（非全局竞逐）：rowwise 每输出行完整流过 B（2048×16MB=32GB 流量），
  内存墙结构性下界 ~137ms——比父本 6.859ms 慢 ~20× 是本结构的固有属性，
  不可能接近整阵族。本实验目标是给 dispatch_target=np_matmul ×
  impl_strategy=rowwise_vectorized 冷格建立首条可归因实测证据；关键可证伪
  对照点：与 naive rowwise np.dot（137-143ms）的配对差（预期 0±2ms——
  matmul gufunc 每行派发税 @2048 仅 ~0.2ms 不可见；einsum 同结构
  +23.5ms 包装税是已实测的对照组）。
- 明确不做：不加 out= 参数、不预转置 B、不动 K 维、无模块级缓存或状态。

远离全部已知死族：不用 K 分块/Strassen/转置后 gemv/降精度/多进程/多线程。

契约（与父本/naive_start.py / reference.py 一致）：
- 暴露 ``matmul(a, b)``：接受二维数组/嵌套 list，返回 (n, m) float32 矩阵积。
- 模块级 np.seterr 抑制 macOS vecLib 虚假 FP 标志噪声（数值正常）。

直接运行（自检，默认 64×64；2048 单次 ~140ms）：
    python3 solution.py --n 2048
"""
import json
import time

import numpy as np

# macOS vecLib BLAS 虚假 FP 标志噪声抑制（数值正常；详见 data/ 同注释）。
np.seterr(divide="ignore", over="ignore", invalid="ignore")


def matmul(a, b):
    """逐行向量化 np.matmul 派发（每行 (1,K)×(K,M) gemm，N=1）。K 求和序不变。

    对父本（iter003 整阵单发）的单变量改动：派发入口 np.matmul 不变，
    仅把整阵一次派发改为预分配 out + 逐行循环派发（1-D 行 × 2-D B，不预转置）。
    """
    a32 = np.asarray(a, dtype=np.float32)
    b32 = np.asarray(b, dtype=np.float32)
    n = a32.shape[0]
    out = np.empty((n, b32.shape[1]), dtype=np.float32)
    for i in range(n):
        out[i] = np.matmul(a32[i], b32)
    return out


def main():
    import argparse

    parser = argparse.ArgumentParser(description="np.matmul rowwise vectorized 自检")
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
        "impl": "np_matmul_rowwise_vectorized",
        "runtime_ms": round(dt_ms, 3),
        "max_error_vs_matmul": max_err,
    }))


if __name__ == "__main__":
    main()
