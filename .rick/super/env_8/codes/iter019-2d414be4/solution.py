#!/usr/bin/env python3
"""solution.py — einsum_path|m_halves：M 维二分 canonical einsum 派发 + concat-T 装配。

迭代 19（codes/iter019-2d414be4）· 父本 codes/iter013-0a966897/solution.py（swapped 拼写
+ 预分配趁热拷贝装配，门中位 7.83875ms）。

实现（单变量改动：matmul 内核 = 拼写轴 swapped → canonical + 装配联动换最优实测形式）：
    h = a32.shape[0] // 2
    r1 = np.einsum('ij,jk->ik', a32[:h], b32, optimize=True)   # TT gemm，(h,N) F 连续视图
    r2 = np.einsum('ij,jk->ik', a32[h:], b32, optimize=True)
    return np.concatenate((r1.T, r2.T), axis=1).T               # (N,M) C 连续装配 + .T 视图

实验目的（W8，拼写轴终局裁决）：
- canonical 拼写（'ij,jk->ik', (a_blk, b)）走 tensordot(a_blk.T 隐式, b.T ...) 即 TT gemm；
  今日负载态 gemm-only 实测快 0.21ms（历史 T/C 漂移幅度 ±0.25ms）。
- 但 TT gemm 输出 F 连续块 → 父本的「预分配 + 趁热 memcpy」对 F 连续块退化为 2.6ms 级
  散布（死，W5 实测 +4.7~+5.1）；最优装配形式为 concat-T：r1/r2 的 .T 基视图（零拷贝）
  沿 axis=1 concatenate 成 (N,M) C 连续（每行 2×4KB 段拷贝），返回 .T 视图（(M,N) F
  连续数组，值语义完全合法）。预研 6 轮实测装配税 +0.47ms（负载无关）。
- 裁决问题：gemm 级 T/C 漂移收益（今日 −0.21，历史 ±0.25 摆动）能否覆盖 F 序装配税
  （+0.47）？预研端到端 +0.14~+0.22 劣化、翻盘需历史幅度 2.2 倍未见先例——本实验用
  门禁（K=3 中位、独立负载态）做拼写轴在本格的终局裁决，证伪则与 pre_research.md 的
  全设计空间分解一起归档格地板。

派发解剖（承 iter002/iter013 结论，本格不变）：
- 2 操作数 path 固定 [(0,1)]，canonical 拼写重排后走 tensordot → TT 标志 gemm；
  M=1024 半块规模下 T/C 优劣随负载态漂移（正是本实验要裁决的对象）。
- concat-T 装配是纯重排，不改 K 求和序 → 逐位一致于 np.matmul（预研 @64/2047/2048、
  嵌套 list、奇数形状均 max_error=0.0、双跑逐位一致）。

结构定位（target_cell: dispatch_target=einsum_path | impl_strategy=m_halves）：
- einsum M 粒度阶梯（@2048 f32，本机）：full 7.28 → halves ~7.8-8.0（本格）→ quarters 9.31。
- 本实验是「拼写轴 T/C 漂移收益 vs F 序装配税」的裁决点：门中位 < 7.83875 → 采纳并
  改写装配范式；≥ 7.83875 → 拼写轴闭合，父本即全构造空间最优点，格地板确认
  （gemm 7.46 + 装配 0.24 + 交互 0.19，全部结构项），scheduler 可降权本格。

实现纪律（预研实测陷阱，硬约束——全部原样保留）：
- canonical 拼写与 optimize=True；h = m//2 上下半分两次整阵乘。
- r1/r2 局部变量保持绑定到函数返回（不做 del/None 提前释放——预研实测提前释放使
  gemm2 缺页劣化）。
- 不用 einsum 的 out= 参数、不动 K 维、无模块级缓存或状态、全程 float32。
- 装配必须用 concat-T（r.T 零拷贝基视图 + axis=1 concatenate + .T 返回视图），
  不用 F 序散布拷贝 / order='C' / 预分配交错拷贝（对 F 连续块均为死路，W2/W5/W6 实测）。

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
    """M 维二分 canonical einsum 派发 + concat-T 装配（逐位一致于 np.matmul）。

    canonical 拼写 np.einsum('ij,jk->ik', a_blk, b32)——path [(0,1)] 重排后走
    tensordot 即 TT 标志 gemm，半块结果为 (h,N) F 连续视图。装配用最优实测形式
    concat-T：r1/r2 的 .T 基视图（零拷贝）沿 axis=1 concatenate 成 (N,M) C 连续
    数组，返回其 .T 视图（(M,N) F 连续）。
    """
    a32 = np.asarray(a, dtype=np.float32)
    b32 = np.asarray(b, dtype=np.float32)
    m = a32.shape[0]
    h = m // 2
    # 上下半两块 canonical 拼写派发（TT gemm，输出 (h,N) F 连续视图）。
    # 实现纪律：不用 out=、不动 K 维、无缓存；r1/r2 保持绑定到函数返回
    # （不提前释放——预研实测提前释放使 gemm2 缺页劣化）。
    r1 = np.einsum('ij,jk->ik', a32[:h], b32, optimize=True)
    r2 = np.einsum('ij,jk->ik', a32[h:], b32, optimize=True)
    # 装配：.T 基视图（零拷贝）沿 axis=1 concatenate 成 (N,M) C 连续（每行 2×4KB
    # 段拷贝），返回 .T 视图——(M,N) F 连续数组，值语义合法。
    return np.concatenate((r1.T, r2.T), axis=1).T


def main():
    import argparse

    parser = argparse.ArgumentParser(description="einsum canonical m_halves concat-T 自检")
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
        "impl": "einsum_canonical_m_halves_concat_T",
        "runtime_ms": round(dt_ms, 3),
        "max_error_vs_matmul": max_err,
    }))


if __name__ == "__main__":
    main()
