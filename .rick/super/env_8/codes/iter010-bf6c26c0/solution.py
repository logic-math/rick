#!/usr/bin/env python3
"""solution.py — M 维四分 np.dot 派发 + out= 直写（iter010，父本 iter001 m_halves）。

策略（target_cell: dispatch_target=np_dot | impl_strategy=m_quarters）：
- edges = [(i*m//4, (i+1)*m//4) for i in range(4)] 行四分边界，每块
  np.dot(a32[s:e], b32, out=c[s:e]) 直写进预分配输出缓冲——沿用 iter001
  （np_dot|m_halves，7.42ms）的派发+直写机制，仅把行二分改为行四分。
- 相比 iter007（einsum|m_quarters，9.31ms 归档）：消除 einsum 包装×4（+0.56ms）
  与 16MB concatenate 拷贝（+0.69ms）两项开销，隔离测量纯 sgemm 分裂损失。
- M 分解不改 K 求和序：逐位一致于 np.matmul 锚（max_error = 0.0）。
- 恰好四次 np.dot 派发；无转置、无 concatenate、不动 K 维。

iter010 实测注记（np_dot 轴 M 粒度阶梯数据点）：
- 同机同负载对照（k=3 中位）：halves 7.51ms / quarters 8.45ms → 2→4 行分裂的
  纯 sgemm 结构性损失 ≈ +0.93ms，与 iter007 三因子归因带（+0.9~1.1ms）一致，
  也与 builder 预研探针（quarters 8.41 vs halves 7.39，+1.0ms）同向同量级。
- quarters 多轮实测 8.45~8.81ms（中位 ~8.6）：落在预期带（7.9-8.6）上缘，
  个别跑次 8.7-8.8 略超上界——候选原因：vecLib/AMX 对 4×512 行块的小 M 内核
  选择差异 + 当前负载波动；未见「quarters 近乎免费」（<7.5ms）的反例。
- 阶梯结论：full 7.01 / halves 7.42（归档）/ quarters ~8.6——M 粒度分裂代价
  随细分数单调上升（1→2 +0.4ms，2→4 +0.93ms），M 方向继续细分应止损。

契约（与 reference.py 一致）：
- matmul(a, b)：接受二维数组/嵌套 list，返回 float32 矩阵积（1-D b 走 gemv 兼容）。
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
    """M 维四分 np.dot + out= 直写（无转置/无拼接/不动 K 维）。

    逐位一致于 np.matmul（BLAS 求和序不变）。
    """
    a32 = np.asarray(a, dtype=np.float32)
    b32 = np.asarray(b, dtype=np.float32)
    m = a32.shape[0]
    if m == 0:
        # 退化守卫：0 行输入直接返回空 zeros（形状 = (0,) + b 列形）
        return np.zeros((0,) + b32.shape[1:], dtype=np.float32)
    # 预分配输出缓冲：形状 = A 行数 × B 列形（兼容 2-D b 与 1-D b gemv）
    c = np.empty((m,) + b32.shape[1:], dtype=np.float32)
    # 行四分边界（块 i 终点 = 块 i+1 起点，恰好无缝覆盖 [0, m)；m<4 时空块跳过）
    edges = [(i * m // 4, (i + 1) * m // 4) for i in range(4)]
    # 每个非空块整阵 np.dot 直写视图（c[s:e] 为 C 连续行切片，out= 合法）
    for s, e in edges:
        if e > s:
            np.dot(a32[s:e], b32, out=c[s:e])
    return c


def main():
    import argparse

    parser = argparse.ArgumentParser(description="m_quarters out= 自检")
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
        "impl": "m_quarters_np_dot_out",
        "runtime_ms": round(dt_ms, 3),
        "max_error_vs_matmul": max_err,
    }))


if __name__ == "__main__":
    main()
