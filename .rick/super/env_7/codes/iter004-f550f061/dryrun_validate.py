#!/usr/bin/env python3
"""dryrun_validate.py — iter004 目标格结构镜像候选（builder 校准工件，非交付物）。

target_cell: dispatch_target=np_dot | impl_strategy=naive_loops
             | memory_layout=row_major | parallelism=serial

结构（四轴，相对父本 env/code/naive_start.py 的单轴结构替换实验）：
- dispatch_target=np_dot：内层 k 收缩从纯 Python 标量累加改为逐元素派发
  ``np.dot(av, bv)``（1-D×1-D 内积入口——非 np.matmul / 非逐行 1-D×2-D /
  非整阵派发，那些属其它格）；
- impl_strategy=naive_loops：保持逐输出元素 (i,j) 双循环粒度不变
  （与 iter002 np_matmul|naive_loops 同粒度先例）——显式 for i / for j 双层循环，
  每次循环恰好产出一个输出标量（262144 次 1-D×1-D 派发），
  无分块 / 无逐行批量收缩 / 无整阵派发 / 无递归分治；
- memory_layout=row_major：操作数归一为 C 序行主序（ascontiguousarray，
  无 asfortranarray / 无转置预处理——区别于 iter001/002 的 col_major）；
  B 的列以跨步视图 b_arr[:, j] 访问（stride 512 元素，校准实测单调用罚分 2.21×）；
  视图预切片（一次性 list(a) / [b[:,j]...]）是循环开销剥离，不改变数据布局；
- parallelism=serial：单线程串行，无子进程/线程池（无残留子进程风险）。

校准带（本机 512，bench 协议 warmup1+k3 中位）：135-145ms；
score ≈ 1.25-1.35e-3（ref ~0.185ms）；正确性 max_error ≈ 2e-4（fp32 逐元素
sdot 累加序 vs BLAS 分块序，容差 1e-3）；同输入双跑逐位一致。
"""
import numpy as np


def matmul(a, b):
    """逐元素 np.dot × C 序行主序 × 串行（目标格四轴，见模块 docstring）。"""
    a_arr = np.ascontiguousarray(np.asarray(a, dtype=np.float32))
    b_arr = np.ascontiguousarray(np.asarray(b, dtype=np.float32))
    if a_arr.ndim != 2 or b_arr.ndim != 2:
        raise ValueError("matmul expects 2-D operands")
    n, k = a_arr.shape
    k2, m = b_arr.shape
    if k != k2:
        raise ValueError("shape mismatch: (%d,%d)x(%d,%d)" % (n, k, k2, m))
    if n == 0 or m == 0:
        return np.zeros((n, m), dtype=np.float32)

    rows = list(a_arr)                          # n 个连续行视图（C 序，row_major）
    cols = [b_arr[:, j] for j in range(m)]      # m 个跨步列视图（C 序原位，row_major）
    out = np.empty((n, m), dtype=np.float32)
    flat = out.reshape(-1)
    dot = np.dot
    t = 0
    for i in range(n):                          # naive_loops：显式 (i,j) 双循环
        av = rows[i]
        for bv in cols:
            flat[t] = dot(av, bv)               # 逐元素 1-D×1-D np.dot 派发
            t += 1
    return out


if __name__ == "__main__":
    # 手动自检：n=64 与 np.matmul 对比（正确性门同款输入协议）
    import json

    np.seterr(divide="ignore", over="ignore", invalid="ignore")
    rng = np.random.RandomState(42)
    x = rng.rand(64, 64).astype(np.float32)
    y = rng.rand(64, 64).astype(np.float32)
    r = matmul(x, y)
    r2 = matmul(x, y)
    print(json.dumps({
        "n": 64,
        "shape": list(r.shape),
        "dtype": str(r.dtype),
        "max_error": float(np.max(np.abs(
            r.astype(np.float64) - np.matmul(x, y).astype(np.float64)))),
        "double_run_equal": bool(np.array_equal(r, r2)),
    }))
