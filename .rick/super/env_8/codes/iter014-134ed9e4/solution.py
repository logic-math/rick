#!/usr/bin/env python3
"""solution.py — einsum_path|m_quarters：M 维四分 einsum 派发（操作数翻转→N-N gemm）+ concatenate 拼接。

迭代 14（codes/iter014-134ed9e4）· 父本 codes/iter007-19438cde/solution.py（m_quarters canonical）。

实现（单变量改动：操作数顺序与下标同步翻转，数学收缩不变）：
    m = a32.shape[0]
    edges = [(i * m // 4, (i + 1) * m // 4) for i in range(4)]   # M 行四分边界
    blocks = [einsum(b32, a32[s:e]) for s, e in edges if e > s]   # 空块跳过
    return np.concatenate(blocks, axis=0)

每块调用 np.einsum('jk,ij->ik', b32, a32[s:e], optimize=True)（父本为
'ij,jk->ik', a_blk, b——同一收缩、同一 M 四分结构、同一 einsum optimize 派发层）。

派发层归因（操作数顺序翻转为何省时）：
- numpy einsum 的 optimize 路径按 contract_inds 降序弹出操作数（einsumfunc.py
  L963/L1455），因此父本 canonical 拼写 einsum('ij,jk->ik', a_blk, b) 被交换成
  tensordot(b, a_blk)→dot(b.T, a_blk.T)（T-T gemm）——块结果是 F-布局转置视图，
  且每块多付一次 4MB c_einsum('ki->ik') 内部转置拷贝，随后 concatenate 还要做
  F-输入的跨步拼接（实测 0.76ms vs C-输入 0.23ms）。
- 翻转拼写 einsum('jk,ij->ik', b, a_blk) 操作数被交换后还原为
  tensordot(a_blk, b)→dot(a_blk, b)——N-N gemm 直接产出 (512, 2048) C-布局块，
  无内部拷贝；np.concatenate(blocks, axis=0) 变成 4 段连续 4MB 流式拷贝。

预研实测（iter014 probe_showdown / probe_s1_final / 末次复确认）：
- 父本 9.03-9.11ms → 本变体 8.83-8.90ms（配对中位 -0.19~-0.22ms，同向胜率
  40/41、69/75、109/116 轮）。
- 正确性：@2048 与 np.matmul 逐位一致（max_error=0.0）；小规模/退化形状
  （m=65、m<4、m=0、非方阵）最差 2.4e-7（容差 1e-3 内，与父本同量级），双跑一致。

同族已证伪变体（后续迭代止损，勿再试）：
- 'ki' 输出拼写 + axis-1 拼接（9.09ms，不优于父本）；
- C-块预分配直写 c[s:e]=blk（9.15ms，劣于 concatenate）；
- 列切片直写（9.29ms，劣于 concatenate）。
→ 拼接用 np.concatenate 而非直写是实测结论（C-块下 concatenate 比直写快 ~0.26ms）。

实现纪律（预研实测陷阱，硬约束）：
- 禁止逐块 out[s:e]=res 直写（F→C 转置拷贝 12.93ms 病态；C-块直写亦劣于
  concatenate，见上同族证伪）。
- 不预转置、不动 K 维、无模块级缓存、全程 float32。

远离全部已知死族：不用 einsum out=/K 分块/Strassen/转置后 gemv/降精度/多进程/多线程。

契约（与父本 naive_start.py→iter007 一致）：
- 暴露 ``matmul(a, b)``：接受二维数组/嵌套 list，返回 (n, m) float32 矩阵积。
- 模块级 np.seterr 抑制 macOS vecLib 虚假 FP 标志噪声（数值正常）。

直接运行（自检，默认 64×64；2048 单次 ~8.8-9.0ms）：
    python3 solution.py --n 2048
"""
import json
import time

import numpy as np

# macOS vecLib BLAS 虚假 FP 标志噪声抑制（数值正常；详见 data/ 同注释）。
np.seterr(divide="ignore", over="ignore", invalid="ignore")


def matmul(a, b):
    """M 维四分 einsum 派发（操作数翻转→N-N gemm→C-块）+ concatenate 拼接。

    逐位一致于 np.matmul（N-N gemm 即 np.matmul 同款调用；M 分解不改 K 求和序）。
    """
    a32 = np.asarray(a, dtype=np.float32)
    b32 = np.asarray(b, dtype=np.float32)
    m = a32.shape[0]
    edges = [(i * m // 4, (i + 1) * m // 4) for i in range(4)]
    blocks = []
    for s, e in edges:
        if e > s:  # 空块跳过（m < 4 时部分块退化）
            # 操作数顺序与下标同步翻转（vs 父本 'ij,jk->ik', a_blk, b）：
            # einsumfunc 降序弹出交换后还原 tensordot(a_blk, b)→N-N gemm→C-布局块。
            # 实现纪律：禁止 out[s:e]=res 逐块直写（同族对照已证伪，见头注）。
            blocks.append(np.einsum('jk,ij->ik', b32, a32[s:e], optimize=True))
    if not blocks:  # m == 0 退化：无块可拼
        return np.zeros((0,) + b32.shape[1:], dtype=np.float32)
    return np.concatenate(blocks, axis=0)


def main():
    import argparse

    parser = argparse.ArgumentParser(description="einsum m_quarters（操作数翻转）自检")
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
        "impl": "einsum_operand_flip_m_quarters_concat",
        "runtime_ms": round(dt_ms, 3),
        "max_error_vs_matmul": max_err,
    }))


if __name__ == "__main__":
    main()
