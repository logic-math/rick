#!/usr/bin/env python3
"""solution.py — einsum_path|m_halves：M 维不对称二分（h=7m/16）swapped einsum 派发。

迭代 22（codes/iter022-56336ab0）· 父本 codes/iter013-0a966897/solution.py（对称
h=m//2 + 预分配 + 趁热逐块拷贝，格精英 7.83875ms）。

单变量改动（本实验核心）：块规模不对称化，切分点 h = m // 2 → h = 7 * m // 16。
块数仍为 2、仍是 M 维二分两次整阵乘；@2048 → 上块 896 行 / 下块 1152 行
（7:9 偏离规范的上下对半，作为二分家族内块规模调优提出；按比例适配任意 m，
n=64 → 28/36 切分）。其余全部原样保留：swapped 拼写 'jk,ij->ik' + optimize=True、
预分配 c + 趁热逐块拷贝、r1/r2 保持绑定不提前释放、不动 K 维、不转置、
无模块级缓存。

机制归因（pre_bench3/4 实测，@2048 f32，同进程交错配对）：
- 单块 einsum t(M) 稳态曲线：M=256→1.171 / 512→2.124 / 768→2.845 / 1024→3.655
  / 1536→5.222 / 1792→6.099 / 2048→6.919 ms。
- µs/行：4.58→4.15→3.70→3.57→3.40→3.40→3.38：1024 以上近线性（每调用固定开销
  c(M) 随 M 消失：c(512)≈0.39ms、c(768)≈0.25、c(1024)≈0.19、c(1536+)≈0），
  512 以下每行成本陡增。
- 端到端二分组合 h 轴扫描（40 轮 vs h=1024 基线 7.755，配对 Δ中位/胜率）：
  h=640 +0.140/7%；h=704 −0.054/67%；h=768 −0.086/87%；**h=896 −0.110/75%**；
  h=1024 基线；h=1280 +0.016/48%；h=1536 +0.198/8%；h=1792 +0.021/48%；
  h=1984 +0.531/0%。
- 解读：h∈[700,900] 是宽优区（两轮独立会话一致：pre_bench3 h=768 −0.078/68%，
  pre_bench4 h=768 −0.086/87%），非单点噪声。上块落 896 行（小 M 固定开销低谷
  +每行成本已近渐近）、下块 1152 行（纯线性区），净省 ~0.1ms。h≤640 上块进入
  高每行成本区、h≥1536 退回对称位形态，均劣化。

正确性：M 维行分组任意切分点不改 K 求和序 → 逐位一致于 np.matmul（全切分点
实测 max_error=0.0，@64/2047/2048 与嵌套 list 输入均验证）。

本迭代预研证伪清单（勿再触碰）：
- 线程三形态全灭（pre_bench.py，30 轮交错配对）：2 路并发 gemm 中位 8.309
  胜率 2/30（vecLib/GCD 争抢）；拷贝重叠 8.327 胜率 9/30 p75 9.1（memcpy 线程
  扰动 BLAS 线程队）；双拷贝进 worker 7.901 胜率 10/30。m_halves（2 块）无任何
  可用并发度；iter017 的 4 线程机制需 4 块对齐，越格。
- 邻接零拷贝装配不可能（pre_bench2.py）：macOS 大分配（8/16MB）各自独立 vm 区
  不拆分不邻接——探针后裸 np.empty 相邻 0/20、真实 einsum 内部分配相邻 0/20、
  稳态 0/20；ctypes 跨分配视图方案（若邻接成立本可省 0.30ms 拷贝）判死刑。

历史已闭合（勿重开）：拼写轴（canonical-TT 8.465 被拒）；einsum out=（病态 8.33，
c_einsum 恒等拷贝）；显式 path（零收益）；提前释放 r1（缺页劣化）；双 gemm 后
concatenate（冷拷贝劣化，已被趁热拷贝替代）；K 维任何操作（容差死族）。

裁决线：门禁 K=3 中位 < 7.83875 → 刷新格精英（预期 7.65-7.80）；≥ 7.83875 →
证伪归档：块规模轴闭合，本格六轴（拼写/包装/装配/线程/分配/块规模）全关，
结构地板 7.75-7.95（2×tensordot gemm 7.2-7.4 + 最小装配 ~0.3 + einsum 包装
0.12）终局确认，供 scheduler 权重→0。软门禁上限 ~8.23（7.83875×1.05），
回退即持平，余量充足。

派发解剖（承父本结论，本格不变）：
- 2 操作数 path 固定 [(0,1)]，swapped 拼写（'jk,ij->ik', (b, a_blk)）重排后走
  tensordot(a_blk, b) 即 C-标志 gemm；块规模 C 连续 → 块拷贝入位退化为纯 memcpy。

实现纪律（预研实测陷阱，硬约束——全部原样保留）：
- swapped 拼写与 optimize=True；h = 7m//16 二分两次整阵乘。
- r1/r2 局部变量保持绑定到函数返回（不做 del/None 提前释放——预研实测提前释放使
  gemm2 缺页劣化）。
- 不用 einsum 的 out= 参数、不转置、不动 K 维、无模块级缓存或状态、全程 float32。

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
    """M 维不对称二分 swapped einsum 派发 + 装配交错化（逐位一致于 np.matmul）。

    块规模不对称化（本迭代单变量改动）：切分点 h = 7m//16（@2048 → 896/1152，
    7:9），落单块 t(M) 固定开销低谷（c(768)≈0.25ms）与近线性渐近区的组合，
    vs 对称 h=m//2 净省 ~0.1ms（pre_bench4 实测 h=896 配对 Δ−0.110ms / 75% 胜率）。
    块数仍为 2；M 行分组不改 K 求和序 → 逐位一致。其余承父本：swapped 拼写
    np.einsum('jk,ij->ik', b32, a_blk)（C-标志 gemm，块结果 C 连续）、预分配 c、
    每块 einsum 完成立即趁热拷贝入位。实现纪律：不用 out=、不转置、不动 K 维、
    无缓存；r1/r2 保持绑定到函数返回（不提前释放——预研实测提前释放使 gemm2
    缺页劣化）。
    """
    a32 = np.asarray(a, dtype=np.float32)
    b32 = np.asarray(b, dtype=np.float32)
    m = a32.shape[0]
    h = 7 * m // 16  # 不对称切分点：@2048 → 896/1152（7:9），块数仍为 2
    # 预分配输出（一次性 mmap，替代 concatenate 的二次分配 + 冷拷贝）。
    c = np.empty((m,) + b32.shape[1:], dtype=np.float32)
    # 上下两块 swapped 拼写派发（C-标志 gemm，块结果 C 连续）+ 趁热立即拷贝入位。
    r1 = np.einsum('jk,ij->ik', b32, a32[:h], optimize=True)
    c[:h] = r1
    r2 = np.einsum('jk,ij->ik', b32, a32[h:], optimize=True)
    c[h:] = r2
    return c


def main():
    import argparse

    parser = argparse.ArgumentParser(description="einsum swapped m_halves 不对称 7:9 二分自检")
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
        "impl": "einsum_swapped_m_halves_asym7_16",
        "runtime_ms": round(dt_ms, 3),
        "max_error_vs_matmul": max_err,
    }))


if __name__ == "__main__":
    main()
