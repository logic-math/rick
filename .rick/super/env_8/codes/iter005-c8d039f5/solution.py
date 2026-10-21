#!/usr/bin/env python3
"""solution.py — np_dot|full_blas_dispatch：单次整阵 np.dot 直接派发。

迭代 5（codes/iter005-c8d039f5）· 父本 iter003-93ddaebc（np.matmul canonical 6.859ms）。

实现（一行核心，单变量替换——只换派发入口，不动其余一切）：
    a32 = np.asarray(a, dtype=np.float32)
    b32 = np.asarray(b, dtype=np.float32)
    return np.dot(a32, b32)

归因假设：np.matmul 走 gufunc 机制（类型解析 + 广播 + 内循环匹配），np.dot 是更薄的
C 层包装直接判维度进 BLAS；@2048 计算由 BLAS gemm 主导，入口层开销差预期为
0.05-0.2ms 量级。本实验把派发入口从 np.matmul 换成 np.dot，检验 gufunc 包装开销
是否可见。2-D 输入下 np.dot 与 np.matmul 走同一 gemm 调用、同一 K 求和序。

- 与锚同 K 求和序 → 逐位一致（max_error=0.0；64/2048 双 seed、嵌套 list、双跑一致）。
- 输出为 C-布局 float32（np.dot 原生形态，无转置视图尾巴）。
- 语义边界：1-D b 时 np.dot 走 gemv（等价 a @ b），2-D b 走 gemm；输出形状
  与 reference 契约一致（(n,)+b32.shape[1:]）。
- 明确不做：不加 out=、不做转置、不动 K 维、无任何模块级缓存或状态。

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
    """整阵 np.dot 直接派发（与 reference 同 BLAS gemm 路径，求和序不变）。"""
    a32 = np.asarray(a, dtype=np.float32)
    b32 = np.asarray(b, dtype=np.float32)
    return np.dot(a32, b32)


def main():
    import argparse

    parser = argparse.ArgumentParser(description="np.dot 直接派发自检")
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
        "impl": "np_dot_full_blas_dispatch",
        "runtime_ms": round(dt_ms, 3),
        "max_error_vs_matmul": max_err,
    }))


if __name__ == "__main__":
    main()
