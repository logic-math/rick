#!/usr/bin/env python3
"""solution.py — np_matmul|full_blas_dispatch：单次整阵 np.matmul 直接派发。

迭代 3（codes/iter003-93ddaebc）· 父本 iter002-7b950b83（einsum canonical 7.28ms）。

实现（一行核心，单变量替换——只换派发入口，不动其余一切）：
    a32 = np.asarray(a, dtype=np.float32)
    b32 = np.asarray(b, dtype=np.float32)
    return np.matmul(a32, b32)

归因假设（iter002 ab_notes 实测）：父本骨架已触及单次整阵派发天花板附近，其相对
ref 的剩余损耗集中在 einsum 包装层——字符串 parse + path 计算 + tensordot 转置视图
派发，@2048 约 0.1-0.15ms。本实验把派发入口换成 np.matmul 直接整阵派发（本格代码
形态的定义本身），消除全部中间包装，直达与 reference 相同的 BLAS gemm 调用路径。

- 与锚同 K 求和序 → 逐位一致（max_error=0.0；64/2048 双 seed、嵌套 list、双跑一致）。
- 输出为 C-布局（np.matmul 原生形态，无 einsum 的 F-布局转置视图尾巴）。
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
    """整阵 np.matmul 直接派发（与 reference 同 BLAS gemm 路径，求和序不变）。"""
    a32 = np.asarray(a, dtype=np.float32)
    b32 = np.asarray(b, dtype=np.float32)
    return np.matmul(a32, b32)


def main():
    import argparse

    parser = argparse.ArgumentParser(description="np.matmul 直接派发自检")
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
        "impl": "np_matmul_full_blas_dispatch",
        "runtime_ms": round(dt_ms, 3),
        "max_error_vs_matmul": max_err,
    }))


if __name__ == "__main__":
    main()
