#!/usr/bin/env python3
"""solution.py — m_halves via np.matmul dispatch（M 维二分：两半整阵 gemm 线程并发）。

目标格：dispatch_target=np_matmul|impl_strategy=m_halves（本格精英 484.43ms，iter16 变体）。

形态（与格定义一致）：
- 输入转 float32（参考锚 dtype；换 dtype = 改 K 求和序，会超 1e-3 容差，禁止）。
- A 的行维 M 二分：mid = M // 2（奇数行安全——下半 floor 行、上半 ceil 行），
  每半各自单次 np.matmul 整阵 BLAS gemm 派发，B 各读一遍。
- 每个输出元素的 K 求和序与整阵 np.matmul 完全一致 → 逐位一致（iter2 原型实测
  max_abs_diff = 0.0 @8192，本变体探针复证全臂 bitwise），必过正确性门。
- C 预分配 + out= 行切片直写：省掉 concatenate 的整阵 256MB 额外拷贝。

iter16 变更（单变量：顺序 → 并发；其余与父本 iter004 逐字一致）：
- 两个半阵 gemm 由顺序执行改为两个 Python 线程并发执行（worker 先 seterr、
  join 后返回 c）。探针实证（7 臂交错 5 轮 @8192，负载窗）：并发两半 498.1ms
  vs 顺序 602.4ms（-104ms / -17%，分布无交叠），且同窗低于整阵 np.matmul 91ms。
- 机制：np.matmul 在 BLAS 段释放 GIL，且本机 Accelerate（vecLib）允许两个并发
  sgemm 重叠执行（OpenBLAS#4741 所述「单调用独占线程池」罚项在本库不存在）。

API 依据（检索来源，research_brief focus：out= 用法与限制）：
- numpy.matmul 官方文档：out 必须精确匹配 (n,k),(k,m)->(n,m) 形状签名；
  不匹配/缺失则新建数组返回。
- numpy PR #29179：matmul 的非连续 out= 在 v2.3.0 曾致内存损坏（2.3.1 修复）——
  本实现只用 C 连续行切片 c[:mid] / c[mid:]（Stack Overflow「views vs copy」：
  行切片返回连续视图，无拷贝），不触碰非连续 out 的坑；本 env numpy 2.0.2。
- Stack Overflow「In which cases numpy's out parameter is faster?」：out= 本身
  不必然更快——此处用它只为规避 concatenate 的输出整阵复制（~512MB 流量）。
- numpy 文档（ufunc/错误处理）：np.seterr 是**线程局部**状态——worker 线程
  各自需要 seterr 抑制 vecLib 虚假 FP 警告（探针 F 臂实证：警告只在 worker
  线程冒出；数值 bitwise 正确，警告只是噪声 + 警告机制短暂持 GIL）。
- numpy#8440：out= 与输入别名约束——两 worker 写同一 c 的**不相交**行切片
  （c[:mid] / c[mid:] 无交叠无别名），不触发别名坑。

鲁棒性（奇数行 / 非方阵 / 嵌套 list 输入，见 __main__ 自检）：
- 奇数行 M：mid = M // 2，两半形状 (mid,k) 与 (M-mid,k)，matmul 形状规则仍成立。
- M=1 / M=0：mid=0 出现 (0,k) 空半——np.matmul 对空维合法，直接退化。
- 嵌套 list：np.asarray(a, dtype=np.float32) 先行归一。
"""
import threading

import numpy as np

# macOS vecLib BLAS 虚假 FP 标志噪声抑制（数值正常；与 reference.py 同款）。
np.seterr(divide="ignore", over="ignore", invalid="ignore")


def matmul(a, b):
    """M 维二分 + np.matmul 整阵派发（两半 gemm 双线程并发）。

    a: (M, K) 嵌套 list 或 ndarray；b: (K, N) 同。返回 (M, N) float32 ndarray。
    逐位一致于 np.matmul(a32, b32)（K 求和序不变；两 worker 写同一 c 的不相交
    行切片，线程调度不影响任何输出元素的计算路径）。
    """
    a32 = np.asarray(a, dtype=np.float32)
    b32 = np.asarray(b, dtype=np.float32)
    m = a32.shape[0]
    n = b32.shape[1]
    mid = m // 2  # 奇数行安全：下半 floor / 上半 ceil
    c = np.empty((m, n), dtype=np.float32)  # 预分配输出

    def _gemm_lo():
        # np.seterr 线程局部：worker 内必须先抑制 vecLib 虚假 FP 警告
        # （探针 F 臂实证；警告机制会短暂持 GIL，拖累并发重叠）。
        np.seterr(divide="ignore", over="ignore", invalid="ignore")
        # 行切片是 C 连续视图 → out= 直写合法且无拷贝（见模块 docstring API 依据）。
        np.matmul(a32[:mid], b32, out=c[:mid])    # 下半整阵 gemm

    def _gemm_hi():
        np.seterr(divide="ignore", over="ignore", invalid="ignore")
        np.matmul(a32[mid:], b32, out=c[mid:])    # 上半整阵 gemm

    # 两 worker 写不相交行切片（numpy#8440 别名坑不触发）；np.matmul 在 BLAS 段
    # 释放 GIL → 两半 sgemm 在 Accelerate 上真并发重叠（探针 -104ms 实证）。
    threads = (
        threading.Thread(target=_gemm_lo),
        threading.Thread(target=_gemm_hi),
    )
    try:
        for t in threads:
            t.start()
    finally:
        # join 必须先于任何返回路径完成（线程建销 ~0.1ms 级，对 500ms 量级无感）。
        for t in threads:
            t.join()
    return c


def _self_check():
    """形状鲁棒性自检：方阵 / 奇数行 / 非方阵 / 嵌套 list / 极小 M。"""
    rng = np.random.RandomState(7)
    cases = [
        (8192 // 64, 8192 // 64, "square small"),
        (129, 64, "odd rows, non-square"),
        (64, 127, "odd cols"),
        (1, 32, "single row"),
        (0, 16, "zero rows"),
        (33, 33, "odd square"),
    ]
    results = []
    for m, k, _label in cases:
        a = rng.rand(m, k)
        b = rng.rand(k, k + 3)
        c = matmul(a, b)
        ref = np.matmul(np.asarray(a, np.float32), np.asarray(b, np.float32))
        err = float(np.max(np.abs(c.astype(np.float64) - ref.astype(np.float64)))) if m else 0.0
        results.append((m, k, c.shape == (m, k + 3), err))
    # 嵌套 list 输入
    la = [[1.0, 2.0], [3.0, 4.0], [5.0, 6.0]]
    lb = [[1.0, 0.0, 2.0], [0.0, 1.0, 3.0]]
    c = matmul(la, lb)
    ref = np.matmul(np.asarray(la, np.float32), np.asarray(lb, np.float32))
    results.append(("lists", 3, c.shape == (3, 3), float(np.max(np.abs(c - ref)))))
    for r in results:
        assert r[2], f"shape mismatch: {r}"
        assert r[3] == 0.0, f"not bitwise identical: {r}"
    print("self-check ok (bitwise identical, shapes robust):", results[:3], "...")


if __name__ == "__main__":
    _self_check()
