#!/usr/bin/env python3
"""solution.py — np_matmul|full_blas_dispatch：零开销派发通道（模块级预绑定）。

迭代 15（codes/iter015-614d495c）· 父本 iter003-93ddaebc（两次 asarray 强转 + np.matmul）。

实现（单变量替换——只把派发入口换成模块级预绑定，不动其余一切）：
    _matmul = np.matmul        # 模块导入期一次性解析并绑定
    def matmul(a, b):
        return _matmul(a, b)

相对父本消除的两类 per-call 开销：
1. 两次 ``np.asarray(a/b, dtype=np.float32)`` 强转——门禁输入恒为 float32 ndarray
   （bench.py / test_correctness.py 均为 rng.rand(n,n).astype(np.float32），
   asarray 是无操作拷贝守卫；reference.py 锚本身也不做强转，直接 ``np.matmul(a,b)``，
   故语义与锚完全一致（含嵌套 list 输入行为：交给 np.matmul 原生处理）。
2. 每次调用的 ``np.matmul`` 属性解析链（np 模块属性查找）——预绑定后函数体只剩
   一次全局 LOAD + CALL。

归因链（父本剖析 + 本机微基准）：
- 父本剖析：subcalls=4、100% 时间在 BLAS 内核内——wrapper 层是唯一可动余量。
- n=64 微基准（本迭代实测，5 组×2000 次取中位数）：raw np.matmul 1.3138µs/call；
  本实现 1.3204µs/call（wrapper 开销 +0.007µs）；父本 1.4505µs/call（wrapper 开销
  +0.137µs）——预绑定通道把 wrapper 消减 ~20×（0.137→0.007µs/call）。
- @2048 该开销占 ~0.002%——本实验的真正命题是归因验证：score 缺口主体是测量
  噪声（sim_bench 两次复现 1.0081/0.9883 跨 1.0、纯循环顺序效应 2.02%、
  per-call 抖动 ±3%），预期 3 次 bench 的 score 在 0.99-1.02 跨 1.0 抖动。
- 逐位一致：与锚走完全相同的 BLAS gemm 调用路径与 K 求和序 → max_error=0.0。

明确不做（实测+语义风险排除）：
- 无 out= 常驻缓冲（仅 0.24% 且别名语义押门禁结构巧合）；
- 无模块级缓存/状态（两遍一致性天然通过）；
- 不动 K 维求和序、不做转置/分块/降精度/多进程/多线程（已知死族）。

契约（与 naive_start.py / reference.py 一致）：
- 暴露 ``matmul(a, b)``：接受二维数组/嵌套 list，返回 (n, m) 矩阵积
  （float32 ndarray 输入 → float32 输出，与 np.matmul 语义完全一致）。
- 模块级 np.seterr 抑制 macOS vecLib 虚假 FP 标志噪声（数值正常）。

直接运行（自检，默认 64×64；2048 单次 ~7ms）：
    python3 solution.py --n 2048
"""
import json
import time

import numpy as np

# macOS vecLib BLAS 虚假 FP 标志噪声抑制（数值正常；详见 data/ 同注释）。
np.seterr(divide="ignore", over="ignore", invalid="ignore")

# 零开销派发通道：导入期一次性预绑定，函数体无 asarray 强转、无属性解析。
_matmul = np.matmul


def matmul(a, b):
    """预绑定 BLAS gemm 直接派发（与 reference 完全同调用路径，逐位一致）。"""
    return _matmul(a, b)


def main():
    import argparse

    parser = argparse.ArgumentParser(description="零开销派发自检")
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
        "impl": "np_matmul_full_blas_dispatch_prebound",
        "runtime_ms": round(dt_ms, 3),
        "max_error_vs_matmul": max_err,
    }))


if __name__ == "__main__":
    main()
