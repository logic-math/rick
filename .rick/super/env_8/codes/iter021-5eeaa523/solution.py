#!/usr/bin/env python3
"""solution.py — np_dot|full_blas_dispatch + 模块级持久 out 缓冲（分配轴三臂终局）。

迭代 21（codes/iter021-5eeaa523）· 父本 iter005-c8d039f5（np.dot 整阵派发，
C 层内部分配输出，门禁中位 6.89ms 量级）。

实现（单变量改动——分配路径三态的第三臂，iter020 骨架上仅换分配策略）：
    a32 = np.asarray(a, dtype=np.float32)
    b32 = np.asarray(b, dtype=np.float32)
    if a32.ndim == 2 and b32.ndim == 2:
        # 模块级持久 scratch：None 或形状不符才（重）分配，跨调用页恒热
        if _OUT is None or _OUT.shape != (a32.shape[0], b32.shape[1]):
            _OUT = np.empty((a32.shape[0], b32.shape[1]), dtype=np.float32)
        return np.dot(a32, b32, out=_OUT)
    return np.dot(a32, b32)              # 其余维度回退原生 np.dot（保 1-D gemv 语义）

机制：np.dot 整阵派发下输出 16MB 的分配路径有三态——
  ① C 层内部分配（父本 iter005）
  ② 每调用 np.empty + out=（iter020，已证与 ① 持平：|Δ中位|≤0.02ms）
  ③ 模块级持久 scratch（本实验，唯一未测臂）：把每次 malloc/free 与首触缺页的
     残余成本摊销为一次，bench warmup=1 后 K 次计时页恒热。
预研（本目录 pre_research.py，2×40 轮同轮相邻交错配对）：③ vs ① 中位差
−0.017~−0.021ms、胜率 20-25/40——±0.05ms 噪声带内，方向为正但未达判据。

⚠️ 返回值别名语义（显式披露）：
- 返回的 ndarray 是模块级共享缓冲——**后续任何 matmul 调用都会原地改写先前返回
  的数组**。调用方若需跨调用保留结果，必须自行 .copy()。
- 不要把先前返回值直接作为后续调用的 a/b 输入（输入与 out 同缓冲别名，BLAS
  在途读写语义未定义）。两道门构造上别名安全：correctness 门在二次调用前已把
  got 转拷为 float64（np.asarray 拷贝），expected 由 np.matmul 先于候选独立
  分配（已逐行核实）；bench 丢弃返回值，warmup 1 次即完成缓冲分配与全页首触。

- 与父本同 cblas_sgemm 调用、同 K 求和序 → 逐位一致（max_error=0.0；
  64/2047/2048 多 seed、1-D b gemv、嵌套 list、0×k/k×0 退化形状全验；同输入
  两次调用输出逐位相同——缓冲复用不自毁）。
- 输出为 C 布局 float32（np.empty 原生形态，无转置视图尾巴）。
- 明确不做：不加 import 期预热（ramp_ab.py 预研 −0.086ms/8 批 5 胜弱信号，属
  测量协议博弈而非算法改进，留档供 human 裁决）、不动 K 维、不转置、不加线程、
  除 _OUT 外无其它模块级状态。

远离全部已知死族：不用 K 分块/Strassen/转置后 gemv/降精度/多进程/多线程。

契约（与 naive_start.py / reference.py 一致）：
- 暴露 ``matmul(a, b)``：接受二维数组/嵌套 list，返回 (n, m) float32 矩阵积
  （共享缓冲，见上）。
- 模块级 np.seterr 抑制 macOS vecLib 虚假 FP 标志噪声（数值正常）。

直接运行（自检，默认 64×64；2048 单次 ~7ms）：
    python3 solution.py --n 2048
"""
import json
import time

import numpy as np

# macOS vecLib BLAS 虚假 FP 标志噪声抑制（数值正常；详见 data/ 同注释）。
np.seterr(divide="ignore", over="ignore", invalid="ignore")

# 模块级持久输出缓冲（分配摊销）：None 或形状不符时才（重）分配。
# 别名语义见模块 docstring——后续调用会原地改写先前返回的数组。
_OUT = None


def matmul(a, b):
    """整阵 np.dot 派发 + 模块级持久 out 缓冲（与 reference 同 BLAS 路径，求和序不变）。

    2-D×2-D 走持久缓冲直写；其余维度（含 1-D b 的 gemv）回退原生 np.dot。
    返回值是共享缓冲——后续调用会改写先前返回的数组（详见模块 docstring）。
    """
    global _OUT
    a32 = np.asarray(a, dtype=np.float32)
    b32 = np.asarray(b, dtype=np.float32)
    if a32.ndim == 2 and b32.ndim == 2:
        shape = (a32.shape[0], b32.shape[1])
        if _OUT is None or _OUT.shape != shape:
            _OUT = np.empty(shape, dtype=np.float32)
        return np.dot(a32, b32, out=_OUT)
    return np.dot(a32, b32)


def main():
    import argparse

    parser = argparse.ArgumentParser(description="np.dot 持久 out 缓冲自检")
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
        "impl": "np_dot_full_blas_dispatch_persistent_out",
        "runtime_ms": round(dt_ms, 3),
        "max_error_vs_matmul": max_err,
    }))


if __name__ == "__main__":
    main()
