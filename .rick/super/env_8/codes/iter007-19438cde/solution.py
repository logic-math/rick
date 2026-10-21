#!/usr/bin/env python3
"""solution.py — einsum_path|m_quarters：M 维四分 einsum 派发 + concatenate 拼接。

迭代 7（codes/iter007-19438cde）· 父本 codes/iter002-7b950b83/solution.py（einsum full）。

实现（单变量结构改动 full→m_quarters）：
    m = a32.shape[0]
    edges = [(i * m // 4, (i + 1) * m // 4) for i in range(4)]   # M 行四分边界
    blocks = [einsum(a32[s:e], b32) for s, e in edges if e > s]   # 空块跳过
    return np.concatenate(blocks, axis=0)

每块仍是 canonical 拼写 np.einsum('ij,jk->ik', a32[s:e], b32, optimize=True)——与父本
同一派发层（2 操作数 path [(0,1)] → tensordot(b, a_blk) → T-标志 gemm），仅 M 粒度从
整阵 2048 降到 4×512。M 分解不改 K 求和序 → 逐位一致于 np.matmul（预研实测 @2048
max_error=0.0；小规模/非方阵/M 不整除 ≤1.2e-7）。

结构定位（本格目标：M 粒度阶梯数据点，非全局竞速）：
- M 粒度阶梯（@2048 f32，本机）：
    einsum full      7.16 ms 预研 / 7.21-7.27 ms 本格复测（父本 iter002，整阵 canonical）
    einsum halves    8.23 ms 预研（2×1024 行）
    einsum quarters  9.19 ms 预研 / 9.33 ms 本格 bench 5 轮中位（4×512 行，本格；
                     配对 vs 父本 +2.10~+2.30ms，10/10 对同向，预期带 +1.9~2.3ms 内）
- 三因子归因（本格复测，vs full_einsum 总差距 +2.15ms）：4×512 行 sgemm 分裂
  损失 ≈ +0.9~1.1ms（vecLib/AMX 小 M 内核效率，结构性；预研估 +1.57ms，
  幅度随负载态漂移、方向稳定）+ einsum 包装 ×4 ≈ +0.56ms + concatenate
  16MB 拼接拷贝 ≈ +0.69ms（含标志差上界）。详见 ab_notes.md。
- 本格为谱系「冷格」数据点：确认 M 粒度-代价曲线（quarters 每份 ~+0.55ms），
  M 方向继续细分预期单调劣化。

实现纪律（预研实测陷阱，硬约束）：
- 禁止逐块 out[s:e]=res 直写——einsum 块结果是 F-布局转置视图，直写触发 F→C
  转置拷贝，实测 12.93ms 病态；np.concatenate 实测最快（预研 9.19ms / 本格
  bench 5 轮中位 9.33ms，备选 z 转置缓冲法 9.43ms 不采用）。
- 不预转置、不动 K 维、无模块级缓存。

远离全部已知死族：不用 einsum out=/K 分块/Strassen/转置后 gemv/降精度/多进程/多线程。

契约（与父本 naive_start.py→iter002 一致）：
- 暴露 ``matmul(a, b)``：接受二维数组/嵌套 list，返回 (n, m) float32 矩阵积。
- 模块级 np.seterr 抑制 macOS vecLib 虚假 FP 标志噪声（数值正常）。

直接运行（自检，默认 64×64；2048 单次 ~9.3ms）：
    python3 solution.py --n 2048
"""
import json
import time

import numpy as np

# macOS vecLib BLAS 虚假 FP 标志噪声抑制（数值正常；详见 data/ 同注释）。
np.seterr(divide="ignore", over="ignore", invalid="ignore")


def matmul(a, b):
    """M 维四分 canonical einsum 派发 + concatenate 拼接（逐位一致于 np.matmul）。"""
    a32 = np.asarray(a, dtype=np.float32)
    b32 = np.asarray(b, dtype=np.float32)
    m = a32.shape[0]
    edges = [(i * m // 4, (i + 1) * m // 4) for i in range(4)]
    blocks = []
    for s, e in edges:
        if e > s:  # 空块跳过（m < 4 时部分块退化）
            # 实现纪律：禁止 out[s:e]=res 逐块直写（F→C 转置拷贝 12.93ms 陷阱）。
            blocks.append(np.einsum('ij,jk->ik', a32[s:e], b32, optimize=True))
    if not blocks:  # m == 0 退化：无块可拼
        return np.zeros((0, b32.shape[1]), dtype=np.float32)
    return np.concatenate(blocks, axis=0)


def main():
    import argparse

    parser = argparse.ArgumentParser(description="einsum m_quarters 自检")
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
        "impl": "einsum_canonical_m_quarters_concat",
        "runtime_ms": round(dt_ms, 3),
        "max_error_vs_matmul": max_err,
    }))


if __name__ == "__main__":
    main()
