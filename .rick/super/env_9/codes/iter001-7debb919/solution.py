#!/usr/bin/env python3
"""solution.py — matmul m_quarters 候选（M 维四分 + 2-D np.dot 整块派发）。

契约：
- 暴露 ``matmul(a, b)``：C 预分配，M 维 4 个行块各自 ``np.dot(a_blk, b, out=C_blk)``
  直写行切片（**不预转置 B、不 K 分块**——保持 BLAS K 求和序）。
- 接受嵌套 list 或 numpy array，返回 (n, m) ndarray。
- 派生自 naive_start.py（rowwise np.dot 起点）：仅重写 matmul 内核，模块头/签名/自检结构不动。

瓶颈修复（profile 证据：rowwise listcomp 占 99.9%——8192 次 gemv 全流量重读 B）：
- M 维四分后每块走 2-D np.dot 大 GEMM 路径，B 重读次数从 8192 次降到 4 次；
- 行切片 C 连续，满足 np.dot 的 out= 约束，免去每块 64MB 临时拷贝与 stack 拼装。

直接运行（自检，默认 64×64；8192 单次 ~0.45s）：
    python3 code/solution.py --n 8192
"""
import json
import time

import numpy as np

# macOS vecLib BLAS 虚假 FP 标志噪声抑制（数值正常；详见 data/ 同注释）。
np.seterr(divide="ignore", over="ignore", invalid="ignore")


def matmul(a, b):
    """m_quarters：M 四分 + 2-D np.dot 整块派发（out= 直写行切片，K 求和序不变）。"""
    a32 = np.asarray(a, dtype=np.float32)
    b32 = np.asarray(b, dtype=np.float32)
    n = a32.shape[0]
    C = np.empty((n, b32.shape[1]), dtype=np.float32)
    q = 4
    for i in range(q):
        i0 = i * n // q
        i1 = (i + 1) * n // q
        np.dot(a32[i0:i1], b32, out=C[i0:i1])
    return C


def main():
    import argparse

    parser = argparse.ArgumentParser(description="m_quarters matmul 自检")
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
        "impl": "m_quarters",
        "runtime_ms": round(dt_ms, 3),
        "max_error_vs_matmul": max_err,
    }))


if __name__ == "__main__":
    main()
