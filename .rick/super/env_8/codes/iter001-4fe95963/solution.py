#!/usr/bin/env python3
"""solution.py — M 维二分 np.dot 派发 + out= 直写（iter001，父本 naive_start.py）。

策略（target_cell: dispatch_target=np_dot | impl_strategy=m_halves）：
- h = m//2 分上下半两次 2-D np.dot 派发，各自用 out=c[:h] / out=c[h:] 直写进
  预分配的输出缓冲——省掉 concatenate 路径的两个 8MB 中间分配与 16MB 拷贝。
- M 分解不改 K 求和序：逐位一致于 np.matmul 锚（max_error = 0.0）。
- 恰好两次 np.dot 派发；无转置、无 concatenate、不动 K 维。

契约（与 reference.py 一致）：
- matmul(a, b)：接受二维数组/嵌套 list，返回 float32 矩阵积。
- 全程 float32（f32 BLAS 求和序与锚一致）。

直接运行（自检，默认 64×64）：
    python3 solution.py --n 2048
"""
import json
import time

import numpy as np

# macOS vecLib BLAS 虚假 FP 标志噪声抑制（数值正常；与 reference/naive_start 同款）。
np.seterr(divide="ignore", over="ignore", invalid="ignore")


def matmul(a, b):
    """M 维二分 np.dot + out= 直写（无转置/无拼接/不动 K 维）。

    逐位一致于 np.matmul（BLAS 求和序不变）。
    """
    a32 = np.asarray(a, dtype=np.float32)
    b32 = np.asarray(b, dtype=np.float32)
    m = a32.shape[0]
    h = m // 2
    # 预分配输出缓冲：形状 = A 行数 × B 列形（兼容 2-D b 与 1-D b）
    c = np.empty((m,) + b32.shape[1:], dtype=np.float32)
    # 上下半各自整阵 np.dot 直写视图（c[:h]/c[h:] 均为 C 连续，out= 合法）
    np.dot(a32[:h], b32, out=c[:h])
    np.dot(a32[h:], b32, out=c[h:])
    return c


def main():
    import argparse

    parser = argparse.ArgumentParser(description="m_halves out= 自检")
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
        "impl": "m_halves_np_dot_out",
        "runtime_ms": round(dt_ms, 3),
        "max_error_vs_matmul": max_err,
    }))


if __name__ == "__main__":
    main()
