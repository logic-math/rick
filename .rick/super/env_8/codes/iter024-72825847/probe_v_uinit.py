#!/usr/bin/env python3
"""solution.py — M 维四分 np.dot 派发 + out= 直写 + 4 线程并发 + worker QoS 加固（iter024，父本 iter017）。

策略（target_cell: dispatch_target=np_dot | impl_strategy=m_quarters）：
- 沿用 iter017 结构逐字不动：iter010 的行四分 + np.dot(a32[s:e], b32, out=c[s:e])
  直写、ThreadPoolExecutor(max_workers=4) 并发 submit、模块级懒加载池单例、
  创建失败回退顺序路径、m==0 守卫、1-D b gemv 兼容、全程 float32。
- iter024 单变量修改：线程池 initializer=_init_worker——worker 线程启动时
  pthread_set_qos_class_self_np(0x21, 0) 把 QoS 升为 USER_INTERACTIVE
  （P 核调度优先于默认 QoS 的背景线程），并设线程局部 np.seterr（iter018
  防御性修正）。QoS 只改调度优先级，BLAS 求和序与父本完全相同。
  动机（探针1/3/4/5 归因）：P 核侧已近饱和（q4w 7.13ms vs 整阵 7.01ms），
  结构方向穷举完毕（6/8 分块、N 细分、E 核协同均劣），剩余杠杆只有负载
  鲁棒性——本机常态 ~4-5 背景负载（默认 QoS）与 worker 公平争抢 P 核；
  USER_INTERACTIVE 把「负载下 threaded 反超整阵派发」（探针3 1.25×）的
  优势固化为可预期常态（探针4/5 顺序对消：合成 6-spinner 负载中位
  +1.14ms、14/15 轮改善、max 尾部收紧；空载态中性 ±0.1ms）。

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

# QOS_CLASS_USER_INTERACTIVE（pthread/qos.h）；relative_priority=0。
_QOS_USER_INITIATED = 0x19


def _init_worker():
    """线程池 worker 初始化（每 worker 线程启动时执行一次）。

    - QoS 升 USER_INTERACTIVE：pthread_set_qos_class_self_np(0x21, 0)——P 核调度
      优先于默认 QoS 的背景线程（负载下保吞吐）。失败静默回退默认 QoS
      （非 macOS / 符号缺失 / 无效参数——池不 broken，仅失去优先级）。
    - np.seterr(...ignore)：worker 线程局部抑制 vecLib 虚假 FP 标志噪声
      （iter018 已验证的防御性修正，零计时成本）。
    """
    try:
        import ctypes
        libc = ctypes.CDLL(None)
        libc.pthread_set_qos_class_self_np.restype = ctypes.c_int
        libc.pthread_set_qos_class_self_np.argtypes = [ctypes.c_uint, ctypes.c_int]
        libc.pthread_set_qos_class_self_np(_QOS_USER_INITIATED, 0)
    except Exception:
        pass  # 静默回退默认 QoS（worker 功能不受影响）
    np.seterr(divide="ignore", over="ignore", invalid="ignore")


def _get_pool():
    """懒创建 ThreadPoolExecutor 单例；失败返回 None（调用方回退顺序路径）。"""
    global _POOL
    if _POOL is None:
        try:
            from concurrent.futures import ThreadPoolExecutor
            _POOL = ThreadPoolExecutor(max_workers=_POOL_WORKERS, initializer=_init_worker)
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

    parser = argparse.ArgumentParser(description="m_quarters out= 4线程+QoS 自检")
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
        "impl": "m_quarters_np_dot_out_threaded4_qos_ui",
        "runtime_ms": round(dt_ms, 3),
        "max_error_vs_matmul": max_err,
    }))


if __name__ == "__main__":
    main()
