#!/usr/bin/env python3
"""solution.py — einsum_path|m_quarters：4 块 swapped-einsum 线程池并发 + concatenate 拼接。

迭代 25（codes/iter025-47700efa）· 父本 codes/iter014-134ed9e4/solution.py
（einsum_path|m_quarters 格精英，门禁 K=3 中位 9.17ms）。

单变量改动（本实验核心）：仅把父本 4 个非空块的串行
``np.einsum('jk,ij->ik', b32, a32[s:e], optimize=True)`` 替换为提交到模块级
懒加载 ``ThreadPoolExecutor(max_workers=4)``（每块一线程），主线程依序
``future.result()`` 收敛后 ``np.concatenate(blocks, axis=0)`` 装配（与父本
逐字一致）。M 四分边界、空块跳过、m==0 守卫、swapped 拼写、实现纪律
（禁 out= 直写/不转置/不动 K 维/无 QoS 操作）全部原样保留。

机制归因（为何线程化本格）：
- perf_profile 证实父本 95.8% 时间在 4 次串行 einsum 调用（8.79ms），Python
  解析仅 ~0.1ms——劣化在 BLAS 派发串行而非包装层；单线程 einsum 块各自
  已在 vecLib 内部多线程，但 4 次串行派发存在块间空泡与调度串行开销。
- 移植邻格 iter017（np_dot|m_quarters）验证过的机制：每块一线程
  （max_workers=4 与块数对齐），vecLib GCD 防超订下单调用退化单线程化而
  总吞吐上升；iter022 已证 concurrency=2 争抢劣化（m_halves 双块无可用
  并发度，胜率 2/30）——4 块恰好对齐制胜形态。
- iter024 已证伪 worker QoS/优先级操作（环境态劣化 0.35-0.54ms）：本实现
  不加任何 QoS/优先级操作，池 worker 走默认配置。
- 线程只改调度顺序不改 K 求和序：4 个块输出互不相交（各算 a32[s:e] 的
  行）、B 只读共享，无数据竞争——逐位一致于 np.matmul 锚。

预研数据（builder 配对实测，load~4-5，18 轮交错）：
- 线程化 7.35ms vs 父本顺序形态 8.99ms：配对 Δ中位 −1.67ms、18/18 全胜，
  三臂 max_error 均 0.0（@2048 逐位一致）。
- 参照 iter017 np_dot 同形态 7.10-7.65ms；ref ~7.3-7.56ms。

证伪线（门禁裁决）：
- 门禁 K=3 中位 < 9.17 → 刷新格精英（预期 ~7.3-7.7ms，score 0.92-0.99）。
- 门禁中位 ≥ 9.17 → 证伪归档（并发未迁移至 einsum 形态）。
- 软门禁上限 ~9.63（9.17×1.05）：回退即持平；环境态较父本劣化 >0.2ms 亦证伪。

实现纪律（承父本，硬约束）：
- swapped 拼写 'jk,ij->ik' + optimize=True（einsumfunc 降序弹出交换后还原
  tensordot(a_blk, b)→N-N gemm→C-布局块，无内部拷贝）。
- 装配用 np.concatenate（同族证伪：C-块预分配直写 9.15ms 劣于 concatenate；
  'ki' 拼写+axis-1 拼接、列切片直写均劣化）。
- 禁逐块 out= 直写（F→C 转置拷贝 12.93ms 病态）；不预转置、不动 K 维、
  无结果缓存、全程 float32；不加 QoS/优先级操作（iter024 证伪）。

契约（与父本 naive_start.py→iter014 一致）：
- 暴露 ``matmul(a, b)``：接受二维数组/嵌套 list，返回 (n, m) float32 矩阵积。
- 模块级 np.seterr 抑制 macOS vecLib 虚假 FP 标志噪声（数值正常）。

直接运行（自检，默认 64×64；2048 单次 ~7.3-7.7ms）：
    python3 solution.py --n 2048
"""
import json
import time

import numpy as np

# macOS vecLib BLAS 虚假 FP 标志噪声抑制（数值正常；详见 data/ 同注释）。
np.seterr(divide="ignore", over="ignore", invalid="ignore")

# 模块级懒加载线程池单例：首次 matmul 调用时创建、跨调用复用（摊销线程创建）。
_POOL = None
_POOL_WORKERS = 4  # 主配置：4 workers（每块一线程，与 M 四分块数对齐）


def _get_pool():
    """懒创建 ThreadPoolExecutor 单例；失败返回 None（调用方回退父本顺序路径）。"""
    global _POOL
    if _POOL is None:
        try:
            from concurrent.futures import ThreadPoolExecutor
            _POOL = ThreadPoolExecutor(max_workers=_POOL_WORKERS)
        except Exception:
            _POOL = False  # 标记不可用，避免每次调用重试创建
    return _POOL or None


def matmul(a, b):
    """M 维四分 swapped-einsum + 4 线程并发 + concatenate 拼接。

    逐位一致于 np.matmul（N-N gemm 即 np.matmul 同款调用；M 分解+线程只改
    调度顺序，不改任何输出元素的 K 求和序）。
    """
    a32 = np.asarray(a, dtype=np.float32)
    b32 = np.asarray(b, dtype=np.float32)
    m = a32.shape[0]
    edges = [(i * m // 4, (i + 1) * m // 4) for i in range(4)]
    blocks = [(s, e) for s, e in edges if e > s]  # 空块跳过（m < 4 时部分块退化）
    if not blocks:  # m == 0 退化：无块可拼
        return np.zeros((0,) + b32.shape[1:], dtype=np.float32)
    if len(blocks) <= 1:
        # 退化守卫：非空块 ≤1（m<4）无并发收益，走父本顺序路径（含 concatenate 装配）
        seq = [np.einsum('jk,ij->ik', b32, a32[s:e], optimize=True)
               for s, e in blocks]
        return np.concatenate(seq, axis=0)
    pool = _get_pool()
    if pool is None:
        # 线程池创建失败 → 回退父本顺序路径（iter014 行为，逐字一致）
        seq = [np.einsum('jk,ij->ik', b32, a32[s:e], optimize=True)
               for s, e in blocks]
        return np.concatenate(seq, axis=0)
    # 每个非空行块一线程并发 einsum（只读共享 b32，输出互不相交，无竞争）
    futs = [pool.submit(np.einsum, 'jk,ij->ik', b32, a32[s:e], optimize=True)
            for s, e in blocks]
    # 依序收敛（future.result() 阻塞至该块完成；装配与父本逐字一致）
    out_blocks = [f.result() for f in futs]
    return np.concatenate(out_blocks, axis=0)


def main():
    import argparse

    parser = argparse.ArgumentParser(description="einsum m_quarters 4线程 自检")
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
        "impl": "einsum_operand_flip_m_quarters_threaded4_concat",
        "runtime_ms": round(dt_ms, 3),
        "max_error_vs_matmul": max_err,
    }))


if __name__ == "__main__":
    main()
