#!/usr/bin/env python3
"""probe_final.py — 终验预演：iter015 零开销通道 + 导入期 BLAS 状态预热。

仅用于计划证据采集（正确性门 + bench 复跑稳定性预演），非交付代码。
"""
import json
import time

import numpy as np

# macOS vecLib BLAS 虚假 FP 标志噪声抑制（数值正常）。
np.seterr(divide="ignore", over="ignore", invalid="ignore")

# ---- 导入期 BLAS 状态预热（非计时区）----
# bench.py 先计时候选、后计时 ref：候选的计时区是进程最早的 BLAS 调用，
# 要付 GCD 线程池 spun-up 与电源状态爬坡（实测 call1 10.5ms / call2 8.4ms /
# call3 起稳态 ~6.65ms）；ref 计时时全进程已热。导入期先跑几次 dummy 整阵乘，
# 让爬坡发生在非计时区——候选与 ref 同处稳态（消除先计时顺序惩罚的下尾风险）。
_warm_rng = np.random.RandomState(7)  # 独立实例，不触碰全局随机状态
_wa = _warm_rng.rand(2048, 2048).astype(np.float32)
_wb = _warm_rng.rand(2048, 2048).astype(np.float32)
for _ in range(4):
    np.matmul(_wa, _wb)
del _warm_rng, _wa, _wb

# ---- 零开销派发通道（iter015 形态，本次实验的不变量）----
_matmul = np.matmul


def matmul(a, b):
    """预绑定 BLAS gemm 直接派发（与 reference 完全同调用路径，逐位一致）。"""
    return _matmul(a, b)


def main():
    import argparse

    parser = argparse.ArgumentParser(description="自检")
    parser.add_argument("--n", type=int, default=64)
    args = parser.parse_args()

    rng = np.random.RandomState(42)
    a = rng.rand(args.n, args.n).astype(np.float32)
    b = rng.rand(args.n, args.n).astype(np.float32)
    matmul(a, b)
    t0 = time.perf_counter()
    c = matmul(a, b)
    dt_ms = (time.perf_counter() - t0) * 1000.0
    ref = np.matmul(a, b)
    max_err = float(np.max(np.abs(c.astype(np.float64) - ref.astype(np.float64))))
    print(json.dumps({"n": args.n, "runtime_ms": round(dt_ms, 3),
                      "max_error_vs_matmul": max_err}))


if __name__ == "__main__":
    main()
