#!/usr/bin/env python3
"""reference.py — matmul 参考实现（正确性锚点 + 性能基线）。

契约：
- 暴露 ``matmul(a, b)``：接受两个二维数组（numpy array 或嵌套 list），
  返回矩阵积（形态与输入匹配）。
- 正确性以本实现为锚：test_correctness.py 用固定 seed 生成输入，
  对比 candidate.matmul 与 np.matmul 的 max_error ≤ env.json 容差。
- 性能以本实现为基线：bench.py 内部同时计时本实现与候选，
  score = ref_median / candidate_median（加速比）。

直接运行（自检）：
    .venv/bin/python code/reference.py        # 2048×2048 计时输出
"""
import json
import time

import numpy as np

# macOS vecLib BLAS 虚假 FP 标志噪声抑制（数值正常；详见 data/ 同注释）。
np.seterr(divide="ignore", over="ignore", invalid="ignore")


def matmul(a, b):
    """参考实现：numpy.matmul（底层 BLAS）。"""
    return np.matmul(a, b)


def main():
    n = 8192
    rng = np.random.RandomState(42)
    a = rng.rand(n, n).astype(np.float32)
    b = rng.rand(n, n).astype(np.float32)
    # 预热 1 次 + 计时 3 次取中位数（与 bench.py 协议一致）
    matmul(a, b)
    timings = []
    for _ in range(3):
        t0 = time.perf_counter()
        matmul(a, b)
        timings.append((time.perf_counter() - t0) * 1000.0)
    timings.sort()
    print(json.dumps({
        "ref": True,
        "n": n,
        "dtype": "float32",
        "runtime_ms": timings[1],
        "timings": timings,
    }))


if __name__ == "__main__":
    main()
