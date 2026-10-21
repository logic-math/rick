#!/usr/bin/env python3
"""naive_start.py — matmul naive 起点（进化起点；正确但慢，慢是特性）。

契约：
- 暴露 ``matmul(a, b)``：rowwise np.dot——A 逐行与 B 做 np.dot（**不预转置 B**）。
- 接受嵌套 list 或 numpy array，返回 (n, m) ndarray。
- 这是进化的起点：coding agent 从本文件出发沿 m 分解/整阵派发方向优化。
- 正确性必须先过门（与参考实现 max_error ≤ 容差）——正确但不优，是合法起点状态。

为什么是 rowwise 而不是纯 Python 三重循环（env_8 与 env_7 的差异，2048 规模事实）：
- 纯 Python 三重循环 @2048 ≈ 274-550s/次，正确性门 30s 告警必超时死——不可行起点。
- rowwise np.dot（不转置）@2048 ≈ 143ms（score ~0.048），且与 np.matmul 逐位一致
  （保持 BLAS 求和序，max_error = 0），必过 1e-3 容差门。
- ⚠️ 不要给 B 预转置（np.dot(bt, a[i]) 变体）：转置改变求和序，@2048 误差 ~1.7e-3
  超容差会被拒——无转置版才是「正确」的形态。

直接运行（自检，默认 64×64；2048 单次 ~143ms）：
    python3 code/naive_start.py --n 2048
"""
import json
import time

import numpy as np

# macOS vecLib BLAS 虚假 FP 标志噪声抑制（数值正常；详见 data/ 同注释）。
np.seterr(divide="ignore", over="ignore", invalid="ignore")


def matmul(a, b):
    """rowwise np.dot（无转置）。逐位一致于 np.matmul（BLAS 求和序不变）。"""
    a32 = np.asarray(a, dtype=np.float32)
    b32 = np.asarray(b, dtype=np.float32)
    return np.stack([np.dot(a32[i], b32) for i in range(a32.shape[0])])


def main():
    import argparse

    parser = argparse.ArgumentParser(description="naive matmul 自检")
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
        "impl": "rowwise_np_dot",
        "runtime_ms": round(dt_ms, 3),
        "max_error_vs_matmul": max_err,
    }))


if __name__ == "__main__":
    main()
