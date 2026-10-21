#!/usr/bin/env python3
"""solution.py — M 维四分 np.matmul 派发 + out= 直写 + 4 线程并发（iter018，父本 iter011）。

策略（target_cell: dispatch_target=np_matmul | impl_strategy=m_quarters）：
- 沿用父本 iter011 的行四分边界 + np.matmul(a32[s:e], b32, out=c[s:e]) 直写
  预分配输出缓冲的机制完全不变；仅把顺序 for 循环替换为
  ThreadPoolExecutor(max_workers=4) 并发 submit（单变量变更）——
  4 个 512 行块天然互相独立（各写预分配 c 的不相交行切片、只读共享 B），
  且 np.matmul 在 BLAS 计算期间释放 GIL，可真正并发执行。
- 机制供体 iter017（np_dot|m_quarters 精英）已证明 4-worker 线程池
  （每块一线程、与块数对齐）可让 4 块在 vecLib GCD 防超订下各自单线程化
  并行执行、总吞吐上升（np_dot 轴 5/5 配对改善 +0.97~1.55ms）；本实验把
  该机制移植进 np_matmul 派发格（builder 本机探针：4 线程行四分 np.matmul
  中位 7.58ms vs 顺序 8.46ms，5/5 配对改善）。2-worker 形态已被 iter017
  证伪（8.68-9.07ms 反劣），worker 数固定 4 与块数对齐。
- 线程池为模块级懒加载单例：首次 matmul 调用时创建、跨调用复用（摊销线程
  创建开销，落在 bench 预热调用内）；创建失败 try/except 回退顺序路径。
- worker 为小包装函数而非裸 submit(np.matmul,...)：函数体内先
  np.seterr(divide/over/invalid='ignore') 再 np.matmul——np.seterr 是
  线程局部状态，模块级设置不覆盖新建 worker 线程（builder 探针观测到
  worker 内偶发 vecLib 虚假 FP RuntimeWarning，防御性一行，零成本）。
- M 分解不改 K 求和序：线程只改调度顺序，不改任何输出元素的 BLAS 求和序，
  逐位一致于 np.matmul 锚（max_error = 0.0）；各线程写不相交行切片、只读
  共享 B，无数据竞争。

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
_POOL_WORKERS = 4  # 主配置：4 workers（每块一线程，与块数对齐；2w 已被 iter017 证伪）


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


def _matmul_worker(a_blk, b32, c_blk):
    """worker 包装：np.seterr 是线程局部状态，须在每个 worker 线程内设置。"""
    np.seterr(divide="ignore", over="ignore", invalid="ignore")
    np.matmul(a_blk, b32, out=c_blk)


def matmul(a, b):
    """M 维四分 np.matmul + out= 直写 + 4 线程并发（无转置/无拼接/不动 K 维）。

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
        # 退化守卫：非空块 ≤1（m<4）无并发收益，直接顺序计算（父本行为）
        for s, e in blocks:
            np.matmul(a32[s:e], b32, out=c[s:e])
        return c
    pool = _get_pool()
    if pool is None:
        # 线程池创建失败 → 回退顺序路径（父本 iter011 行为）
        for s, e in blocks:
            np.matmul(a32[s:e], b32, out=c[s:e])
        return c
    # 每个非空行块一线程并发 np.matmul 直写视图（c[s:e] 为 C 连续行切片，out= 合法）
    futs = [pool.submit(_matmul_worker, a32[s:e], b32, c[s:e]) for s, e in blocks]
    # 依序收敛（future.result() 阻塞至该块完成；写不相交切片无竞争）
    for f in futs:
        f.result()
    return c


def main():
    import argparse

    parser = argparse.ArgumentParser(description="m_quarters np_matmul 4线程 自检")
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
        "impl": "m_quarters_np_matmul_out_threaded4",
        "runtime_ms": round(dt_ms, 3),
        "max_error_vs_matmul": max_err,
    }))


if __name__ == "__main__":
    main()
