#!/usr/bin/env python3
"""solution.py — einsum_path|rowwise_vectorized：逐行向量化 einsum 派发。

迭代 4（codes/iter004-09d14ede）· 父本 iter002（einsum canonical 整阵派发 7.28ms）。

单变量结构实验：派发入口保持 einsum optimize=True 不变，只把 impl_strategy 从
整阵单发改为逐行向量化——预分配 out 后循环 n 次：

    out = np.empty((n, b32.shape[1]), np.float32)
    for i in range(n):
        out[i] = np.einsum('j,jk->k', a32[i], b32, optimize=True)

逐行派发解剖（numpy 2.0.2 _core/einsumfunc.py 源码确认 + 本机实测）：
- 2 操作数时 path 固定为 [(0,1)]，逆序 pop 得 [b32, a32[i]]，einsum_str 重排为
  'jk,j->k'。
- _can_dot(['jk','j'], 'k', {'j'}) transpose-both 分支命中 BLAS 路径 →
  tensordot(b32, a32[i], axes=((0,),(0,))) → dot(b.T 视图, a_i 列向量)，即
  T-标志 gemm N=1（与父本整阵同族）；tensor_result 'k' == 输出 'k'，无尾部
  c_einsum。
- K 求和序不变：@2048 双 seed 与 np.matmul 逐位一致（max_error=0.0），
  @64 ~7.6-9.5e-6 与 naive rowwise np.dot 同级（vecLib 小规模路径差异，
  远低于 1e-3 容差）。
- canonical 'j,jk->k' 拼写：预研三种 rowwise 拼写最快（171.5 vs swapped
  173.6 vs (1,n) 切片 174.2 ms）。

结构定位（非全局竞逐）：rowwise 每输出行完整流过 B（2048×16MB=32GB 流量），
内存墙结构性下界 ~130-145ms；叠加 einsum 包装税（@64 微基准 ~11.3μs/调用 vs
np.dot 0.55μs，@2048 折 ~13μs/行 ×2048≈27ms）→ 预期 ~165-185ms。本实验目标是
给 dispatch_target=einsum_path|impl_strategy=rowwise_vectorized 冷格建立首条
可归因实测证据。

远离全部已知死族：不用 K 分块/Strassen/转置后 gemv/降精度/多进程/多线程/out=
（einsum out= 实测病态；本实现预分配 out 后按行赋值，非 einsum 的 out= 参数）。

契约（与父本/naive_start.py 一致）：
- 暴露 ``matmul(a, b)``：接受二维数组/嵌套 list，返回 (n, m) float32 矩阵积。
- 模块级 np.seterr 抑制 macOS vecLib 虚假 FP 标志噪声（数值正常）。

直接运行（自检，默认 64×64；2048 单次 ~170ms）：
    python3 solution.py --n 2048
"""
import json
import time

import numpy as np

# macOS vecLib BLAS 虚假 FP 标志噪声抑制（数值正常；详见 data/ 同注释）。
np.seterr(divide="ignore", over="ignore", invalid="ignore")


def matmul(a, b):
    """逐行向量化 einsum 派发（每行 T-标志 gemm N=1）。K 求和序不变。

    对父本（iter002 整阵单发）的单变量改动：派发入口 einsum optimize=True
    不变，仅把整阵一次派发改为预分配 out + 逐行循环派发。
    """
    a32 = np.asarray(a, dtype=np.float32)
    b32 = np.asarray(b, dtype=np.float32)
    n = a32.shape[0]
    out = np.empty((n, b32.shape[1]), dtype=np.float32)
    for i in range(n):
        out[i] = np.einsum('j,jk->k', a32[i], b32, optimize=True)
    return out


def main():
    import argparse

    parser = argparse.ArgumentParser(description="einsum rowwise vectorized 自检")
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
        "impl": "einsum_rowwise_vectorized",
        "runtime_ms": round(dt_ms, 3),
        "max_error_vs_matmul": max_err,
    }))


if __name__ == "__main__":
    main()
