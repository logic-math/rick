#!/usr/bin/env python3
"""solution.py — M 维四分 np.dot 派发 + out= 直写 + 4 线程并发（iter017，父本 iter010 m_quarters）。

策略（target_cell: dispatch_target=np_dot | impl_strategy=m_quarters）：
- 沿用 iter010 的行四分 + np.dot(a32[s:e], b32, out=c[s:e]) 直写结构不变；
  仅把顺序 for 循环替换为 ThreadPoolExecutor(max_workers=4) 并发 submit——
  4 个 512 行块天然互相独立（各写预分配 c 的不相交行切片、只读共享 B），
  且 np.dot 在 BLAS 计算期间释放 GIL，可真正并发执行。
- 线程池为模块级懒加载单例：首次 matmul 调用时创建、跨调用复用（摊销线程
  创建开销，落在 bench 预热调用内）；创建失败 try/except 回退顺序路径。
- M 分解不改 K 求和序：线程只改调度顺序，不改任何输出元素的 BLAS 求和序，
  逐位一致于 np.matmul 锚（max_error = 0.0）；各线程写不相交行切片、只读
  共享 B，无数据竞争。

iter017 实测注记（threaded M 分片 × out= 预分配组合，格内隔离检验；同机含
~270% 背景负载，交替配对对照）：
- 主配置 4 workers（每块一线程）：bench.py k=3 中位 × 5 轮交替配对——
  threaded4 7.16-7.65ms（中位 ~7.25，score 0.91-0.99）vs 父本顺序 8.55-8.71ms
  （中位 ~8.62，score 0.77-0.85）→ 配对差 +0.97~+1.55ms，5/5 轮全部改善。
- 补上领域知识空档「线程化 M 分片与 out= 组合未实测」：组合实测 ~7.2ms，
  好于研究带（7.5-8.0）下缘；陷阱记录的 vecLib GCD 防超订上限（8.4ms）未在
  4w 复现（仅个别跑次 8.0-8.2 波动，与背景负载相关，仍 <8.45 证伪线）。
- 单参数对照 2 workers（4 块入 2 线程池 → 2 波 × 2 路并发 sgemm）：
  8.68/9.07/8.95ms（score 0.77-0.79）——反而劣于父本顺序路径：2 路并发
  sgemm 时 vecLib 内部线程与并发调用争抢（GCD 防超订劣化在此形态显形），
  4 路并发时每调用退化单线程化、总吞吐反而上升——并发度与块数对齐才有效。
- 判定：中位 ~7.25ms < 8.3ms 改进成立线且 5/5 配对一致，本格线程方向有效；
  仍未及整阵派发天花板（~7.0ms），余量 ~0.2ms 为 4×512 行块小 M 内核
  效率损失 + B 重读 4 遍 DRAM 流量（结构性，与父本归因一致）。
- perf_profile：subcalls 10→106（submit/future 机器开销，~μs 级），主线程
  99% 时间阻塞在 future.result() 锁上——BLAS 计算全部移入 worker 线程，
  主线程纯等待，Python 包装开销占比进一步下降。

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

# 模块级懒加载线程池单例：首次 matmul 调用时创建、跨调用复用（摊销线程创建）。
_POOL = None
_POOL_WORKERS = 4  # 主配置：4 workers（每块一线程）


def _get_pool():
    """懒创建 ThreadPoolExecutor 单例；失败返回 None（调用方回退顺序路径）。"""
    global _POOL
    if _POOL is None:
        try:
            from concurrent.futures import ThreadPoolExecutor
            _POOL = ThreadPoolExecutor(max_workers=_POOL_WORKERS)
        except Exception:
            _POOL = False  # 标记不可用，避免每次调用重试创建
    return _POOL or None


def matmul(a, b):
    """M 维四分 np.dot + out= 直写 + 4 线程并发（无转置/无拼接/不动 K 维）。

    逐位一致于 np.matmul（线程只改调度顺序，BLAS 求和序不变）。
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
    blocks = [(s, e) for s, e in edges if e > s]
    if len(blocks) <= 1:
        # 退化守卫：非空块 ≤1（m<4）无并发收益，直接顺序计算
        for s, e in blocks:
            np.dot(a32[s:e], b32, out=c[s:e])
        return c
    pool = _get_pool()
    if pool is None:
        # 线程池创建失败 → 回退顺序路径（父本 iter010 行为）
        for s, e in blocks:
            np.dot(a32[s:e], b32, out=c[s:e])
        return c
    # 每个非空行块一线程并发 np.dot 直写视图（c[s:e] 为 C 连续行切片，out= 合法）
    futs = [pool.submit(np.dot, a32[s:e], b32, c[s:e]) for s, e in blocks]
    # 依序收敛（future.result() 阻塞至该块完成；写不相交切片无竞争）
    for f in futs:
        f.result()
    return c


def main():
    import argparse

    parser = argparse.ArgumentParser(description="m_quarters out= 4线程 自检")
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
        "impl": "m_quarters_np_dot_out_threaded4",
        "runtime_ms": round(dt_ms, 3),
        "max_error_vs_matmul": max_err,
    }))


if __name__ == "__main__":
    main()
