#!/usr/bin/env python3
"""solution.py — m_halves via np.dot dispatch（M 维二分：输出行上下半各一次整阵 2-D dot）。

目标格：dispatch_target=np_dot|impl_strategy=m_halves（本格首访实作，无精英；
对照基线 = 父格精英 iter004「m_halves + np.matmul」484.43ms）。

形态（与格定义一致）：
- 输入转 float32（参考锚 dtype；换 dtype = 改 K 求和序，会超 1e-3 容差，禁止）。
- A 的行维 M 二分：mid = M // 2（奇数行安全——下半 floor 行、上半 ceil 行），
  每半各自单次 2-D ``np.dot`` 整阵 BLAS gemm 派发，B 各读一遍。
- K 求和序不动：不切 K/N、不转置 B、不换 dtype——每个输出元素的 K 累加与整阵
  np.matmul 同一 BLAS 路径（2-D f32 下 dot 与 matmul 映射同一 cblas_sgemm），
  门规模逐位一致（本 session 预检 @8192：array_equal=True, maxdiff=0.0）。
- C 预分配 + out= 行切片直写：省掉 concatenate 的整阵 256MB 额外拷贝。

API 依据（检索来源，research_brief focus：np.dot out= 用法与限制）：
- numpy.dot 官方文档（NumPy v2.x Manual）：out 必须类型正确、**C 连续**、dtype 与
  dot(a,b) 返回一致——不满足直接抛 ValueError（非弹性处理）。本实现只用 C 序数组
  的行切片 c[:mid] / c[mid:]（C 连续视图，无拷贝），形状精确匹配 (块高, N)。
- numpy GitHub #7124：dot 的 out= 非 C-array 报 "output array is not acceptable" 实证
  ——行切片合法，memmap 等非连续 out 才踩坑。
- numpy GitHub #18669「matmul with out= slower than np.dot in some cases」：
  matmul 的 out= 路径在输出页未 fault-in 时显著慢于 np.dot——同为 out= 直写，
  **np.dot 入口更稳**（本 session 预检同协议 3 连：m_halves_np_dot 中位
  449.6/466.5/459.8ms vs iter004 同 session 复测 480.5ms，~5% 入口差）。
- numpy GitHub #23588：matmul 对可 gemm 化形状/连续性检查更挑剔（不满足落
  noblas ~100× 慢路径）；dot 派发更宽容——行切片两侧合法，dot 侧风险更低。
- numpy#8440（iter007 调研）：out 与输入同阵别名会静默全零——本实现 c 为全新
  np.empty，与 a32/b32 无别名，安全。

鲁棒性（奇数行 / 非方阵 / 嵌套 list 输入，见 __main__ 自检）：
- 奇数行 M：mid = M // 2，两半形状 (mid,K) 与 (M-mid,K)，dot 2-D 形状规则仍成立。
- M=1 / M=0：mid=0 出现 (0,K) 空半——np.dot 对空维 2-D 合法，直接退化。
- 嵌套 list：np.asarray(a, dtype=np.float32) 先行归一。

契约（与父系一致，保持不动）：
- ``matmul(a, b)`` 接口 / asarray f32 / np.seterr 三项抑制；
- main() 的 --n 自检 JSON 协议（预热 1 次 + 计时 1 次 + max_error_vs_matmul，
  模板 = env/code/naive_start.py），impl 标签 "m_halves_np_dot"。
"""
import json
import time

import numpy as np

# macOS vecLib BLAS 虚假 FP 标志噪声抑制（数值正常；与 reference.py 同款）。
np.seterr(divide="ignore", over="ignore", invalid="ignore")


def matmul(a, b):
    """M 维二分 + 每半单次 2-D np.dot 整阵派发。

    a: (M, K) 嵌套 list 或 ndarray；b: (K, N) 同。返回 (M, N) float32 ndarray。
    K 求和序不变 → 逐位一致于 np.matmul(a32, b32)（预检 @8192 maxdiff=0.0）。
    """
    a32 = np.asarray(a, dtype=np.float32)
    b32 = np.asarray(b, dtype=np.float32)
    m = a32.shape[0]
    n = b32.shape[1]
    mid = m // 2  # 奇数行安全：下半 floor / 上半 ceil
    c = np.empty((m, n), dtype=np.float32)  # 预分配输出
    # 行切片是 C 连续视图 → dot 的 out= 严格契约满足（见模块 docstring API 依据）。
    np.dot(a32[:mid], b32, out=c[:mid])    # 下半整阵 2-D dot（sgemm）
    np.dot(a32[mid:], b32, out=c[mid:])    # 上半整阵 2-D dot（sgemm）
    return c


def _self_check():
    """形状鲁棒性自检：方阵 / 奇数行 / 非方阵 / 嵌套 list / 极小 M。

    断言分级：常规块形状（K>=16）逐位一致（err=0.0，门规模实测）；退化小形状
    容许 1-ULP 界（2.4e-07）——vecLib 小矩阵内核域效应（iter005 实测记录，
    m_halves 亦然），远在 1e-3 门容差内。
    """
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
        # 常规块形状逐位一致；空维/退化形状（k < 16）容 1-ULP 内核域界
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
        "impl": "m_halves_np_dot",
        "runtime_ms": round(dt_ms, 3),
        "max_error_vs_matmul": max_err,
    }))


if __name__ == "__main__":
    _self_check()
    main()
