#!/usr/bin/env python3
"""solution.py — matmul：einsum optimize 路径内部序列的手动复制 × M 维四分（iter011）。

契约（与父本 naive_start.py 一致，全部保持不动）：
- 暴露 ``matmul(a, b)``：接受嵌套 list 或 numpy array，asarray 归一 f32，
  返回 (m, n) float32 ndarray。
- main 的预热/计时/自检 JSON 输出协议与 np.seterr 设置保持不动。

本迭代改动（唯一一处，kernel 内部）：
- 父本 rowwise np.dot listcomp（@8192 实测 11516-11657ms，profile 热点 99.9%）→
  **einsum optimize 路径内部序列的手动复制，按 M 维四分逐块派发**（本格定义形态：
  dispatch_target=einsum_path × impl_strategy=m_quarters）。

机制（numpy 2.0.2 源码级依据，numpy/_core/einsumfunc.py L1418-1503）：
``np.einsum('ij,jk->ik', a, b, out=dst, optimize=True)`` 的内部执行序列为
  ① einsum_path 生成单步收缩（blas=True）；
  ② tensordot(a, b, axes=((1,), (0,))) → np.dot → BLAS sgemm（K 求和序与 np.matmul 一致）；
  ③ handle_out=True → c_einsum('ik->ik', tmp, out=dst) —— **恒等收缩走 C einsum 通用
     迭代器逐元素拷贝**（不是 memcpy）：@8192² 实测整链 735-832ms，比 ②+memcpy 慢
     +275-380ms——开销在 einsum 自身调度代码（本实验要绕开的对象）。
手动复制该序列（②tensordot + ③换成 memcpy 速度的切片赋值，256MB ≈ 16.6ms @35.6GB/s）
整阵实测 458ms；按 M 四分逐块派发实测 475.5ms——落 BLAS 平价带
（同族 np_dot m_halves 462.8 / np_matmul m_quarters 481.8），且与 np.matmul 锚逐位一致
（max_error = 0.0，@8192 实测 array_equal=True）。

正确性纪律（domain_knowledge 合法族约束）：
- 只动 M 维（行四分：q = ceil(m/4)，余数并入末块，空块跳过）；不动 K 维、不转置 B、
  不低精度——K 求和序 = BLAS sgemm 原序。
- tensordot 的 transpose/reshape 在 2-D C 连续输入 + axes=((1,),(0,)) 下均为恒等视图
  （numeric.py L1140-1156，零拷贝）。

来源与依据（检索纪律，role=coding 落地点，≤3 源）：
1. numpy 2.0.2 源码（本机 env_9 venv）einsumfunc.py L1418-1503（einsum optimize+out 的
   内部序列与 c_einsum 恒等拷贝）+ numeric.py L1108-1163（tensordot 的恒等视图与 dot 派发）。
2. numpy 官方文档（einsum out= 语义：结果拷入给定数组；tensordot axes 序列语义）。
3. 本 env 实测台账（builder 计划 + 本机复测）：einsum(out=, optimize=True) 735-832ms；
   手动序列 458-479ms；m_quarters 475.5ms；@8192 全部与锚逐位一致。

正确性纪律补充（debug attempt 1，实测依据）：m<4 时四分退化为 1 行块，M=1 sgemm
（GEMV 型内核）与整阵 gemm 累加序偶差 1 ULP——回退整阵单次 tensordot（einsum 原生
形态）恢复逐位一致；m>=4 的门规模路径（8192→4×2048、64→4×16）不变、实测逐位。

直接运行（自检，默认 64×64）：
    python3 solution.py --n 8192
"""
import json
import time

import numpy as np

# macOS vecLib BLAS 虚假 FP 标志噪声抑制（数值正常；详见 data/ 同注释）。
np.seterr(divide="ignore", over="ignore", invalid="ignore")


def matmul(a, b):
    """einsum optimize 内部序列（tensordot BLAS 收缩 + 结果拷入预分配 C）× M 四分。

    逐位一致于 np.matmul 锚（K 求和序不动：tensordot→np.dot→sgemm 同派发路径）。
    """
    a32 = np.asarray(a, dtype=np.float32)
    b32 = np.asarray(b, dtype=np.float32)
    m = a32.shape[0]
    n = b32.shape[1]
    # C 预分配（对应 einsum out= 的目标数组）
    out = np.empty((m, n), dtype=np.float32)
    # M 四分退化（m<4 行，凑不出 4 个非空块）：四分退化为 1 行块——M=1 的 sgemm
    # 在 Accelerate 走 GEMV 型内核，累加序与锚的多行 gemm 偶差 1 ULP（本机实测
    # (2,16) err=4.77e-07 = 1ULP@|x|~4；SO#76707696「np.dot 两段算结果不同」、
    # numpy#26791「Accelerate 形状相关精度」同现象）。einsum 内部序列对整阵本来就
    # 是**单次** tensordot（einsum 从不切 M）——m<4 回退整阵单发即其原生形态，
    # 与 np.matmul 锚逐位一致（实测 0.0）。门规模路径（m>=4）不受影响。
    if m < 4:
        out[:] = np.tensordot(a32, b32, axes=((1,), (0,)))
        return out
    # M 维四分：q = ceil(m/4)（8192→4×2048；余数并入末块）
    q = -(-m // 4)
    for i in range(4):
        lo = min(i * q, m)
        hi = min((i + 1) * q, m)
        if lo < hi:
            # einsum 内部序列 ②③：tensordot BLAS 收缩（einsumfunc.py L1477 同款
            # axes=((1,),(0,))）+ 恒等拷入 out——einsum 用 c_einsum 通用迭代器拷贝
            # （慢路径，本实验绕开），此处为 memcpy 速度的切片赋值（快路径）。
            out[lo:hi] = np.tensordot(a32[lo:hi], b32, axes=((1,), (0,)))
    return out


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
        "impl": "einsum_path_m_quarters",
        "runtime_ms": round(dt_ms, 3),
        "max_error_vs_matmul": max_err,
    }))


if __name__ == "__main__":
    main()
