#!/usr/bin/env python3
"""solution.py — matmul 单次整阵 np.dot 派发（iter007，父系 naive_start.py / iter003 模板）。

契约（与父本一致，全部保持不动）：
- 暴露 ``matmul(a, b)``：接受嵌套 list 或 numpy array，asarray 转 f32，
  返回 (n, m) ndarray。
- main 的预热/计时/自检 JSON 输出协议与 np.seterr 设置保持不动。

本迭代改动（唯一一处，kernel 入口）：
- 2-D×2-D 整阵 ``np.dot(a32, b32)`` 直派 cblas_sgemm——本格定义形态
  （dispatch_target=np_dot / impl_strategy=full_blas_dispatch）。
- 父本逐行 listcomp + np.stack（profile @8192：listcomp self 99.9%、runtime 11612ms）
  是纯 Python 层装配开销；本格把 kernel 收敛为一次 BLAS 调用（~26× 塌缩）。
- K 求和序不动：np.dot 与 np.matmul 在 2-D float32 连续阵下映射同一 cblas_sgemm
  （调研报告 §3 / 1911.09421；本 session 预检 array_equal=True、maxdiff 0.0）。
- 不用 out=（numpy#8440：out 与输入同阵静默全零——直接返回新分配数组）、
  不预转置 B（转置改求和序，@8192 误差 ~1.4e-2 超容差）、不切 K/N、无中间拷贝。

直接运行（自检，默认 64×64）：
    python3 code/solution.py --n 8192
"""
import json
import time

import numpy as np

# macOS vecLib BLAS 虚假 FP 标志噪声抑制（数值正常；详见 data/ 同注释）。
np.seterr(divide="ignore", over="ignore", invalid="ignore")


def matmul(a, b):
    """单次整阵 np.dot（2-D BLAS gemm）直派。逐位一致于 np.matmul 锚（同一 cblas 调用）。"""
    a32 = np.asarray(a, dtype=np.float32)
    b32 = np.asarray(b, dtype=np.float32)
    return np.dot(a32, b32)


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
        "impl": "full_blas_dispatch_np_dot",
        "runtime_ms": round(dt_ms, 3),
        "max_error_vs_matmul": max_err,
    }))


if __name__ == "__main__":
    main()
