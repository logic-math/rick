#!/usr/bin/env python3
"""solution.py — matmul 单次整阵 np.dot 派发（iter017；父系 naive_start.py，iter003/iter007 契约模板）。

目标格：dispatch_target=np_dot|impl_strategy=full_blas_dispatch（本格 visit=2：
iter007 同一设计预检全绿但死于 turn-budget 流程断裂，verdict=no_candidate——
本轮为流程修复重跑，kernel 与契约不变，钉死「每工具至多跑一次」最小动作链）。

契约（与父本一致，全部保持不动）：
- 暴露 ``matmul(a, b)``：接受嵌套 list 或 numpy array，asarray 转 f32，
  返回 (M, N) float32 ndarray。
- main 的预热/计时/自检 JSON 输出协议与 np.seterr 三项抑制保持不动。

本迭代改动（唯一一处，kernel）：
- 逐行 listcomp + np.stack（iter17 profile @8192：listcomp self 10400.1ms 占 99.8%、
  stack 10.1ms、dot 仅 1.23ms——瓶颈全在 Python 层装配，BLAS 内核本身只值 ~440ms）
  → **单次整阵 ``np.dot(a32, b32)`` 2-D cblas_sgemm 直派**（本格定义形态）。
- K 求和序不动：numpy.dot 官方文档——both 2-D 即 matrix multiplication，f32 连续阵下
  与 np.matmul 映射同一 BLAS gemm 调用（iter007/iter010 预检 @8192 array_equal=True、
  maxdiff=0.0；iter007 同进程交错预检 dot 437.4 vs matmul 433.1ms，±1% 噪声带）。
- 不用 out=（numpy#7124：out 须类型/维数/C 连续精确匹配，非弹性约束；
  numpy#18669：matmul+out 在未 fault-in 输出页上显著慢——plain 调用两入口都在入口
  新分配输出，入口差异在噪声带内）。
- 不预转置 B（转置改求和序，@8192 误差 ~1.4e-2 超容差——naive_start 实测记录）、
  不切 K/N、不做 1-D 化（numpy#13732：gemm/gemv 派发由 numpy 决定，父本逐行 1-D
  gemv + B 全量重读正是 ~25x 慢的结构根源）、无中间拷贝。

鲁棒性自检（_self_check，用例表移植自 iter010）：方阵/奇数行/非方阵/M=1/M=0/嵌套 list，
K>=16 逐位一致、退化形 ≤ 1-ULP（2.4e-07，vecLib 小矩阵内核域效应，远在 1e-3 门内）。

直接运行（自检，默认 64x64；8192 预期 ~434-488ms、max_error_vs_matmul=0.0）：
    python3 code/solution.py --n 8192
"""
import json
import time

import numpy as np

# macOS vecLib BLAS 虚假 FP 标志噪声抑制（数值正常；与 reference.py 同款）。
np.seterr(divide="ignore", over="ignore", invalid="ignore")


def matmul(a, b):
    """单次整阵 np.dot（2-D BLAS gemm）直派。逐位一致于 np.matmul 锚（同一 cblas 调用）。"""
    a32 = np.asarray(a, dtype=np.float32)
    b32 = np.asarray(b, dtype=np.float32)
    return np.dot(a32, b32)


def _self_check():
    """形状鲁棒性自检（用例表移植自 iter010）：方阵/奇数行/非方阵/M=1/M=0/嵌套 list。

    断言分级：常规块形状（K>=16）逐位一致（err=0.0，门规模实测）；退化小形状容
    1-ULP 界（2.4e-07）——vecLib 小矩阵内核域效应（iter005/iter010 实测记录），
    远在 1e-3 门容差内。
    """
    rng = np.random.RandomState(7)
    cases = [
        (128, 128, "square small"),
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
    results.append(("lists", 2, c.shape == (3, 3), float(np.max(np.abs(c - ref)))))
    for r in results:
        assert r[2], f"shape mismatch: {r}"
        # 常规块形状逐位一致；退化形状（K < 16）容 1-ULP 内核域界
        tol = 0.0 if r[1] >= 16 else 2.4e-07
        assert r[3] <= tol, f"error {r[3]} > tol {tol}: {r}"
    print("self-check ok (gate shapes bitwise, robust shapes <= 1 ULP):",
          results[:3], "...")


def main():
    import argparse

    parser = argparse.ArgumentParser(description="solution matmul 自检")
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
        "impl": "full_blas_dispatch_np_dot",
        "runtime_ms": round(dt_ms, 3),
        "max_error_vs_matmul": max_err,
    }))


if __name__ == "__main__":
    _self_check()
    main()
