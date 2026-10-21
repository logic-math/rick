#!/usr/bin/env python3
"""solution.py — M 维四分 np.matmul 派发 + out= 直写（iter011，父本 iter008 的 m_halves 骨架）。

策略（target_cell: dispatch_target=np_matmul | impl_strategy=m_quarters）：
- edges = [(i*m//4, (i+1)*m//4) for i in range(4)] 行四分边界（取自 iter010），
  每个非空块 np.matmul(a32[s:e], b32, out=c[s:e]) 直写进预分配输出缓冲——
  沿用 iter008 父本（np_matmul|m_halves，7.30ms）的派发+直写机制，
  仅把行二分改为行四分（单变量变更 = 分裂粒度 2→4）。
- 恰好四次 np.matmul 派发（m%4 余行自动归入靠后块，块 i 终点 = 块 i+1 起点，
  无缝覆盖 [0, m)；m<4 时空块跳过，m==0 走退化守卫返回空数组）。
- M 分解不改 K 求和序：逐位一致于 np.matmul 锚（max_error = 0.0）。
- 无转置、无 concatenate、不动 K 维。

iter011 实验定位（格子填充与 M 粒度阶梯测量，非最优冲刺）：
- 预期 runtime（2048×2048 f32，k=3 中位）8.2~8.9ms 带（builder 探针 8.36ms）：
  比父本 halves_matmul_out 7.39ms 慢约 +0.9~1.0ms（2→4 行分裂结构性损失，
  与 np_dot 轴阶梯数据点 +0.93ms 同量级）；比 np_dot|quarters 精英 8.53ms
  持平或略优 0.05~0.15ms（np.matmul 直派发省 np.dot 包装层差异）。
- 可证伪点：实测 <7.8ms → 「分裂代价随 M 粒度单调上升」在 np_matmul 轴被证伪；
  >9.0ms 或正确性非零 → out= 切片快路径或无缝覆盖假设被证伪，需归因重设计。

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
    """M 维四分 np.matmul + out= 直写（无转置/无拼接/不动 K 维）。

    逐位一致于 np.matmul（BLAS 求和序不变）。
    """
    a32 = np.asarray(a, dtype=np.float32)
    b32 = np.asarray(b, dtype=np.float32)
    m = a32.shape[0]
    if m == 0:
        # 退化守卫（取自 iter010）：0 行输入直接返回空 zeros（形状 = (0,) + b 列形）
        return np.zeros((0,) + b32.shape[1:], dtype=np.float32)
    # 预分配输出缓冲：形状 = A 行数 × B 列形（兼容 2-D b 与 1-D b gemv）
    c = np.empty((m,) + b32.shape[1:], dtype=np.float32)
    # 行四分边界（块 i 终点 = 块 i+1 起点，恰好无缝覆盖 [0, m)；m<4 时空块跳过）
    edges = [(i * m // 4, (i + 1) * m // 4) for i in range(4)]
    # 每个非空块整阵 np.matmul 直写视图（c[s:e] 为 C 连续行切片，out= 合法）
    for s, e in edges:
        if e > s:
            np.matmul(a32[s:e], b32, out=c[s:e])
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
        "impl": "m_quarters_np_matmul_out",
        "runtime_ms": round(dt_ms, 3),
        "max_error_vs_matmul": max_err,
    }))


if __name__ == "__main__":
    main()
