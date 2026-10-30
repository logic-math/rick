#!/usr/bin/env python3
"""naive_start.py — matmul naive 起点（进化起点；正确但慢，bench 慢是特性）。

契约：
- 暴露 ``matmul(a, b)``：纯 Python 三重循环（无 numpy 依赖），
  接受嵌套 list 或 numpy array（逐元素索引），返回嵌套 list。
- 这是进化的起点：coding agent 从本文件出发向 blocked_loops/rowwise_vectorized/full_blas_dispatch 方向优化（特征空间 3×3=6 格，见 domain_knowledge.md）。
- 正确性必须先过门（与参考实现 max_error ≤ 容差）——正确但不优，是合法起点状态。

直接运行（自检，默认 64×64——512 纯 Python 单次 15s+，手动跑大矩阵自己加 --n）：
    python3 code/naive_start.py --n 64
"""
import json
import random
import time


def matmul(a, b):
    """朴素三重循环。转置视图（zip(*b)）让内层沿行访问列，比 b[t][j] 快数倍。"""
    n = len(a)
    m = len(b[0])
    k = len(b)
    bt = list(zip(*b))  # 转置：bt[j] 是 b 的第 j 列
    c = []
    for i in range(n):
        ai = a[i]
        row = [0.0] * m
        for j in range(m):
            bj = bt[j]
            s = 0.0
            for t in range(k):
                s += ai[t] * bj[t]
            row[j] = s
        c.append(row)
    return c


def main():
    import argparse

    parser = argparse.ArgumentParser(description="naive matmul 自检")
    parser.add_argument("--n", type=int, default=64, help="矩阵规模（默认 64）")
    args = parser.parse_args()

    random.seed(42)
    a = [[random.random() for _ in range(args.n)] for _ in range(args.n)]
    b = [[random.random() for _ in range(args.n)] for _ in range(args.n)]
    t0 = time.perf_counter()
    c = matmul(a, b)
    dt_ms = (time.perf_counter() - t0) * 1000.0
    print(json.dumps({
        "ref": False,
        "n": args.n,
        "impl": "naive_loops",  # 起点自称（不在精选特征轴内——进化即离开此格）,
        "runtime_ms": round(dt_ms, 3),
        "checksum": sum(c[0]) if c else 0.0,
    }))


if __name__ == "__main__":
    main()
