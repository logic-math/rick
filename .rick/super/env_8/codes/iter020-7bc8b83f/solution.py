#!/usr/bin/env python3
"""solution.py — np_dot|full_blas_dispatch + out= 预分配直写（杠杆裁决）。

迭代 20（codes/iter020-7bc8b83f）· 父本 iter005-c8d039f5（np.dot 整阵派发，
门禁中位 6.89ms 量级）。

实现（单变量改动——父本让 C 层内部分配输出，本迭代改为显式预分配直写）：
    a32 = np.asarray(a, dtype=np.float32)
    b32 = np.asarray(b, dtype=np.float32)
    if a32.ndim != 2:
        return np.dot(a32, b32)          # 1-D gemv 语义保持原生派发
    c = np.empty((a32.shape[0],) + b32.shape[1:], dtype=np.float32)
    return np.dot(a32, b32, out=c)

机制假设：整阵派发下输出装配本为零拷贝，out= 只改变「谁分配输出」——
预期分配器暖复用下无收益。

裁决结论（预研 pre_research.py 40 轮交错配对 A/B：父本中位 6.890 vs out= 6.909、
胜率 12/40；本迭代 ab_test.py 同会话复测：P1 父本中位 6.830 vs out= 6.825、
18/40 胜，P2 门禁协议 6.846 vs 6.856）：
- 两次测量 |Δ中位| 均 ≤ 0.02ms ≤ 0.05ms 噪声带 —— 判持平，负载抖动 ±0.3ms
  主导一切分配侧微差异。
- out= 杠杆在本格（dispatch_target=np_dot|impl_strategy=full_blas_dispatch）
  闭合：格地板 = BLAS 平价，分配侧无可榨余量。
- 与父本同 BLAS gemm 调用、同 K 求和序 → 逐位一致（max_error=0.0；
  64/2047/2048 多 seed、1-D b gemv、嵌套 list、退化形状 0×k/k×0 全部 =0.0）。
- 输出仍为 C 布局 float32（np.empty 原生形态）。
- 明确不做：不动 K 维、不转置、不加线程/缓存、无模块级状态。

远离全部已知死族：不用 K 分块/Strassen/转置后 gemv/降精度/多进程/多线程。

契约（与 naive_start.py / reference.py 一致）：
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
    """整阵 np.dot 派发 + out= 预分配直写（与 reference 同 BLAS 路径，求和序不变）。

    a32.ndim != 2 时回退原生 np.dot，保持 1-D gemv 语义不变。
    """
    a32 = np.asarray(a, dtype=np.float32)
    b32 = np.asarray(b, dtype=np.float32)
    if a32.ndim != 2:
        return np.dot(a32, b32)
    c = np.empty((a32.shape[0],) + b32.shape[1:], dtype=np.float32)
    return np.dot(a32, b32, out=c)


def main():
    import argparse

    parser = argparse.ArgumentParser(description="np.dot out= 直写自检")
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
        "impl": "np_dot_full_blas_dispatch_out_prealloc",
        "runtime_ms": round(dt_ms, 3),
        "max_error_vs_matmul": max_err,
    }))


if __name__ == "__main__":
    main()
