#!/usr/bin/env python3
"""solution.py — einsum_path|full_blas_dispatch：单次整阵 swapped einsum 派发。

迭代 12（codes/iter012-d7d3748e）· 父本 iter002（canonical 拼写，在库精英 7.2757ms）。

实现（一行核心，单变量改动：拼写 canonical → swapped）：
    return np.einsum('jk,ij->ik', b32, a32, optimize=True)

派发解剖（numpy 2.0.2 _core/einsumfunc.py 源码确认 + 本机实测，见 pre_research.md）：
- swapped 拼写 'jk,ij->ik'（操作数 b, a）：路径反转后 tmp_operands=[a, b]、einsum_str
  重排为 'ij,jk->ik' → BLAS 分支 `tensordot(a, b, axes=((1,),(0,)))` = dot(a, b)
  **纯 C-标志 gemm**（与 np.matmul 同内核族，无转置视图入参）；tensor_result 'ik'
  == results 'ik' → **跳过尾部 c_einsum**，原生 C 序直接返回（无转置视图链）。
- 父本 canonical 拼写：`tensordot(b, a, axes=((0,),(1,)))` = dot(b.T 视图, a.T 视图)
  T-标志 gemm + 尾部 'ki->ik' 转置视图（F 序输出）。
- K 求和序与 np.matmul 一致 → 逐位一致（max_error=0.0，64/2048 双 seed、嵌套 list、
  双跑一致均实测通过）。
- 预研证据（本机 2026-10-21）：进程内配对 TT−CC gemm 差 +0.23ms（tdot_TT 6.983 vs
  tdot_CC 6.758）；einsum 机器开销仅 ~0.03ms（perf_profile：99.2% 时间在 tensordot 内）；
  门协议 ×3 swapped 中位 7.008 vs canonical 7.191。旧主张「CC 先计时退化 7.5-8.6ms」
  未复现。风险：T/C 差随负载态漂移（父本两天前测得反向 +0.04-0.07ms）——若门中位
  不优于父本即证伪回退（见 ab_notes.md）。

远离全部已知死族：不用 K 分块/Strassen/转置后 gemv/降精度/多进程/多线程/out=（einsum
out= 实测 16.8ms 病态）。

契约（与父本一致）：
- 暴露 ``matmul(a, b)``：接受二维数组/嵌套 list，返回 (n, m) float32 矩阵积。
- 模块级 np.seterr 抑制 macOS vecLib 虚假 FP 标志噪声（数值正常）。

直接运行（自检，默认 64×64；2048 单次 ~7ms）：
    python3 solution.py --n 2048
"""
import json
import time

import numpy as np

# macOS vecLib BLAS 虚假 FP 标志噪声抑制（数值正常；详见 data/ 同注释）。
np.seterr(divide="ignore", over="ignore", invalid="ignore")


def matmul(a, b):
    """整阵 swapped einsum 派发（C-标志 gemm，无尾部视图）。逐位一致于 np.matmul（求和序不变）。"""
    a32 = np.asarray(a, dtype=np.float32)
    b32 = np.asarray(b, dtype=np.float32)
    return np.einsum('jk,ij->ik', b32, a32, optimize=True)


def main():
    import argparse

    parser = argparse.ArgumentParser(description="einsum swapped 派发自检")
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
        "impl": "einsum_swapped_full_blas_dispatch",
        "runtime_ms": round(dt_ms, 3),
        "max_error_vs_matmul": max_err,
    }))


if __name__ == "__main__":
    main()
