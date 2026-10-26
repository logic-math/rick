#!/usr/bin/env python3
"""solution.py — matmul 单次整阵 np.matmul 派发（iter003，父本 naive_start.py）。

契约（与父本一致，全部保持不动）：
- 暴露 ``matmul(a, b)``：接受嵌套 list 或 numpy array，asarray 转 f32，
  返回 (n, m) ndarray。
- main 的预热/计时/自检 JSON 输出协议与 np.seterr 设置保持不动。

本迭代改动（iter13，唯一一处，kernel 内部——builder 探针 out_ab.py 预验证的形态③）：
- 每次调用新建输出缓冲并以写入式预 fault（``c[:] = 0``，非 calloc 惰性页），
  再 ``np.matmul(a32, b32, out=c)`` 派发——numpy#18669/OpenBLAS#3197 的「预 fault 反转侧」。
- 仍为单次整阵 BLAS gemm 派发（本格定义形态）；beta=0 全量覆写，预填 0 不参与结果，
  K 求和序与参考锚逐位一致（max_abs_diff = 0.0）。
- 不用持久共享缓冲（本机实测恒慢 ~14%：out_ab A/B 498-523ms vs plain 436ms）。
- 非 2-D 输入走泛化兜底 plain np.matmul，契约与父本一致。
- 实测结论（iter13）：预 fault 反转侧与 plain 平价（436 vs 436ms）——无净增益，收口证伪；
  详见 experiment_record.md 第 6 段。

直接运行（自检，默认 64×64）：
    python3 code/solution.py --n 8192
"""
import json
import time

import numpy as np

# macOS vecLib BLAS 虚假 FP 标志噪声抑制（数值正常；详见 data/ 同注释）。
np.seterr(divide="ignore", over="ignore", invalid="ignore")


def matmul(a, b):
    """整阵 BLAS gemm 派发（形态③：新建输出 + 预 fault + out=）。

    逐位一致于参考锚：同一 cblas_sgemm 调用、beta=0 全量覆写，K 求和序不变。
    out 形状按 (n,k),(k,m)->(n,m) 签名精确构造（numpy 手册约束），f32 入/f32 出无 cast。
    """
    a32 = np.asarray(a, dtype=np.float32)
    b32 = np.asarray(b, dtype=np.float32)
    if a32.ndim == 2 and b32.ndim == 2:
        c = np.empty((a32.shape[0], b32.shape[1]), dtype=np.float32)
        c[:] = 0  # 写入式预 fault（OpenBLAS#3197：多线程缺页争内核锁；numpy#18669 反转侧）
        np.matmul(a32, b32, out=c)
        return c
    return np.matmul(a32, b32)  # 非 2-D 泛化兜底（契约与父本一致）


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
        "impl": "prefault_out_np_matmul",
        "form": "fresh_alloc_prefault_out",
        "runtime_ms": round(dt_ms, 3),
        "max_error_vs_matmul": max_err,
    }))


if __name__ == "__main__":
    main()
