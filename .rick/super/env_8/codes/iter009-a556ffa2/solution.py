#!/usr/bin/env python3
"""solution.py — einsum_path|m_halves：M 维二分 swapped einsum 派发 + concatenate。

迭代 9（codes/iter009-a556ffa2）· 父本 codes/iter008-1446e1c9/solution.py（m_halves 骨架）
+ codes/iter007-19438cde/solution.py（einsum M 分裂装配纪律）+ codes/iter002-7b950b83/
solution.py（einsum 派发解剖：canonical/swapped 拼写与 gemm 标志差异）。

实现（单变量改动：m_halves 骨架的派发入口换成 swapped 拼写 einsum）：
    h = a32.shape[0] // 2
    r1 = np.einsum('jk,ij->ik', b32, a32[:h], optimize=True)
    r2 = np.einsum('jk,ij->ik', b32, a32[h:], optimize=True)
    return np.concatenate([r1, r2], axis=0)

派发解剖（numpy 2.0.2 _core/einsumfunc.py 源码 + 本机实测，父本 iter002 结论）：
- 2 操作数时 path 固定 [(0,1)]，`tmp_operands = [operands.pop(x) for x in (1,0)]` 逆序
  重排——swapped 拼写（'jk,ij->ik', (b, a_blk)）重排后走 tensordot(a_blk, b)，
  即 C-标志 gemm；canonical 拼写（'ij,jk->ik', (a_blk, b)）走 tensordot(b, a_blk)，
  即 T-标志 gemm。
- **规模依赖的拼写翻转（本实验核心假设）**：M=1024 半块规模下 C-标志 gemm 反超
  T-标志（预研配对实测 swapped 2×1024 行 7.44ms vs canonical 7.58ms），而 iter002
  整阵 2048 行时 canonical 快 +0.15ms——结论随 M 规模翻转。
- C-标志 gemm 块结果天然 C 连续 → 两半块 concatenate(axis=0) 退化为纯 2×8MB 块
  memcpy（预研 0.234ms），而 canonical 的 F 布局转置视图拼接要 0.648ms——装配环节
  再省 ~0.41ms。
- M 分解不改 K 求和序 → 逐位一致于 np.matmul（预研 @2048 max_error=0.0；奇数 M
  与小非方阵误差 ~1e-7 量级，远低于 1e-3 容差）。

结构定位（target_cell: dispatch_target=einsum_path | impl_strategy=m_halves，冷格首建）：
- einsum M 粒度阶梯（@2048 f32，本机）：full 7.28 → **halves ~8.0（本格）** →
  quarters 9.31——本格数据点补全阶梯单调性。
- vs iter007 预研 canonical+concatenate halves 基线 8.23ms，预期快 ~0.3ms
  （官方 bench 口径 7.9-8.1ms）。
- 不预期击败全局精英 np_matmul|full（6.86ms）；本实验价值是冷格首建 +
  「einsum 拼写选择随 M 规模翻转」这一可复用归因结论。

实现纪律（预研实测陷阱，硬约束）：
- 不用 einsum 的 out= 参数（16.8/9.4ms 病态）；不做 C 缓冲 out[s:e]=res 逐块直写
  （12.9ms F→C 转置拷贝陷阱）；不转置、不动 K 维、无模块级缓存或状态。

远离全部已知死族：K 维分块/Strassen/转置后 gemv/降精度/多进程/多线程。

契约（与 reference.py / naive_start.py 一致）：
- 暴露 ``matmul(a, b)``：接受二维数组/嵌套 list，返回 float32 矩阵积。
- 模块级 np.seterr 抑制 macOS vecLib 虚假 FP 标志噪声（数值正常）。

直接运行（自检，默认 64×64；2048 单次 ~8ms）：
    python3 solution.py --n 2048
"""
import json
import time

import numpy as np

# macOS vecLib BLAS 虚假 FP 标志噪声抑制（数值正常；与 reference/naive_start 同款）。
np.seterr(divide="ignore", over="ignore", invalid="ignore")


def matmul(a, b):
    """M 维二分 swapped einsum 派发 + concatenate（逐位一致于 np.matmul）。

    swapped 拼写 np.einsum('jk,ij->ik', b32, a_blk)——path [(0,1)] 逆序重排后走
    tensordot(a_blk, b) 即 C-标志 gemm，块结果天然 C 连续，两半块拼接退化为纯
    2×8MB 块 memcpy。
    """
    a32 = np.asarray(a, dtype=np.float32)
    b32 = np.asarray(b, dtype=np.float32)
    h = a32.shape[0] // 2
    # 上下半两块 swapped 拼写派发（C-标志 gemm，块结果 C 连续）。
    # 实现纪律：不用 out=、不做逐块直写、不转置、不动 K 维、无缓存。
    r1 = np.einsum('jk,ij->ik', b32, a32[:h], optimize=True)
    r2 = np.einsum('jk,ij->ik', b32, a32[h:], optimize=True)
    # 两块均 C 连续 → concatenate 退化为纯 2×8MB 块 memcpy（预研 0.234ms）。
    return np.concatenate([r1, r2], axis=0)


def main():
    import argparse

    parser = argparse.ArgumentParser(description="einsum swapped m_halves 自检")
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
        "impl": "einsum_swapped_m_halves_concat",
        "runtime_ms": round(dt_ms, 3),
        "max_error_vs_matmul": max_err,
    }))


if __name__ == "__main__":
    main()
