#!/usr/bin/env python3
"""solution.py — einsum_path|m_halves：M 维二分 swapped einsum 派发 + 装配阶段交错化。

迭代 13（codes/iter013-0a966897）· 父本 codes/iter009-a556ffa2/solution.py（swapped 拼写
+ 双 gemm 后 concatenate）。

实现（单变量改动：装配阶段从「双 gemm 后 concatenate」改为「预分配 + 逐块趁热拷贝」）：
    h = a32.shape[0] // 2
    c = np.empty((m,) + b32.shape[1:], dtype=np.float32)   # 预分配输出
    r1 = np.einsum('jk,ij->ik', b32, a32[:h], optimize=True)
    c[:h] = r1                                              # 趁 gemm 刚写完块热立即拷贝
    r2 = np.einsum('jk,ij->ik', b32, a32[h:], optimize=True)
    c[h:] = r2
    return c

装配温度回收假设（本实验核心）：
- 父本 8.08ms 中装配（双 gemm 后 concatenate）in-context 实测 0.44ms，其中 ~0.15-0.2ms
  是 r1 在 gemm2 期间被逐出 cache 的温度损耗；逐块「einsum 完成立即拷贝」让拷贝发生在
  块刚写完仍热时，把装配压向 ~0.30ms 的 2×8MB memcpy 下限。
- 预研排除的方向不再触碰：einsum out=（实测 8.33ms 病态，c_einsum 恒等拷贝 0.41ms/8MB）、
  提前释放 r1（缺页劣化 +0.15~0.3ms）、显式 path（2 操作数无搜索零收益）、batched
  'aij,jk->aik' 单 gemm（语义上属 full_blas_dispatch，越格）。
- 拷贝入位是纯重排，不改 K 求和序 → 逐位一致于 np.matmul（预研 @64/2047/2048 与嵌套
  list 输入均 max_error=0.0）。

派发解剖（承父本 iter002 结论，本格不变）：
- 2 操作数 path 固定 [(0,1)]，swapped 拼写（'jk,ij->ik', (b, a_blk)）重排后走
  tensordot(a_blk, b) 即 C-标志 gemm；M=1024 半块规模下 C-标志反超 T-标志
  （预研 swapped 7.44ms vs canonical 7.58ms）。
- C-标志 gemm 块结果天然 C 连续 → 块拷贝入位退化为纯 8MB 块 memcpy。

结构定位（target_cell: dispatch_target=einsum_path | impl_strategy=m_halves）：
- einsum M 粒度阶梯（@2048 f32，本机）：full 7.28 → halves ~8.0（本格）→ quarters 9.31。
- 本实验是「装配开销可压缩」假设的裁决点：若门中位不优于父本 → 证伪，归档
  「einsum_path|m_halves 结构 floor：2×半阵 gemm 7.2-7.4ms + 最小装配 ~0.3ms +
  einsum 包装 0.12ms ≈ 7.75-7.95ms，非 gemm 开销已最小化」供 scheduler 降权该格。

实现纪律（预研实测陷阱，硬约束——全部原样保留）：
- swapped 拼写与 optimize=True；h = m//2 上下半分两次整阵乘。
- r1/r2 局部变量保持绑定到函数返回（不做 del/None 提前释放——预研实测提前释放使
  gemm2 缺页劣化）。
- 不用 einsum 的 out= 参数、不转置、不动 K 维、无模块级缓存或状态、全程 float32。

远离全部已知死族：K 维分块/Strassen/转置后 gemv/降精度/多进程/多线程。

契约（与 reference.py / naive_start.py 一致）：
- 暴露 ``matmul(a, b)``：接受二维数组/嵌套 list，返回 float32 矩阵积。
- 模块级 np.seterr 抑制 macOS vecLib 虚假 FP 标志噪声（数值正常）。

直接运行（自检，默认 64×64；2048 单次 ~8ms）：
    python3 solution.py --n 2048
"""
import json
import time

import numpy as np

# macOS vecLib BLAS 虚假 FP 标志噪声抑制（数值正常；与 reference/naive_start 同款）。
np.seterr(divide="ignore", over="ignore", invalid="ignore")


def matmul(a, b):
    """M 维二分 swapped einsum 派发 + 装配交错化（逐位一致于 np.matmul）。

    swapped 拼写 np.einsum('jk,ij->ik', b32, a_blk)——path [(0,1)] 逆序重排后走
    tensordot(a_blk, b) 即 C-标志 gemm，块结果天然 C 连续。预分配 c 后每半块
    einsum 完成立即拷贝入位（趁块热），替代双 gemm 后 concatenate 的冷拷贝。
    """
    a32 = np.asarray(a, dtype=np.float32)
    b32 = np.asarray(b, dtype=np.float32)
    m = a32.shape[0]
    h = m // 2
    # 预分配输出（一次性 mmap，替代 concatenate 的二次分配 + 冷拷贝）。
    c = np.empty((m,) + b32.shape[1:], dtype=np.float32)
    # 上下半两块 swapped 拼写派发（C-标志 gemm，块结果 C 连续）+ 趁热立即拷贝入位。
    # 实现纪律：不用 out=、不转置、不动 K 维、无缓存；r1/r2 保持绑定到函数返回
    # （不提前释放——预研实测提前释放使 gemm2 缺页劣化）。
    r1 = np.einsum('jk,ij->ik', b32, a32[:h], optimize=True)
    c[:h] = r1
    r2 = np.einsum('jk,ij->ik', b32, a32[h:], optimize=True)
    c[h:] = r2
    return c


def main():
    import argparse

    parser = argparse.ArgumentParser(description="einsum swapped m_halves 装配交错化自检")
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
        "impl": "einsum_swapped_m_halves_interleaved_copy",
        "runtime_ms": round(dt_ms, 3),
        "max_error_vs_matmul": max_err,
    }))


if __name__ == "__main__":
    main()
