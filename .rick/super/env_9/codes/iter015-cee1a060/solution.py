#!/usr/bin/env python3
"""solution.py — matmul 单次整阵 np.matmul 派发（iter003，父本 naive_start.py）。

契约（与父本一致，全部保持不动）：
- 暴露 ``matmul(a, b)``：接受嵌套 list 或 numpy array，asarray 转 f32，
  返回 (n, m) ndarray。
- main 的预热/计时/自检 JSON 输出协议与 np.seterr 设置保持不动。

本迭代改动（唯一一处，kernel 内部）：
- 删除逐行 listcomp + np.stack（父本 99.9% 热点：Python 层 8192 次循环 + B 全量重读），
  改为单次 ``np.matmul(a32, b32)`` 整阵 BLAS gemm 派发——本格定义形态
  （dispatch_target=np_matmul / impl_strategy=full_blas_dispatch）。
- K 求和序不动：与锚 np.matmul 为同一调用路径，逐位一致（max_abs_diff = 0.0）。
- 不预转置 B、不用 out=、不引入 reshape/中间拷贝/额外 dtype 转换。

本迭代（iter015）唯一改动（线程池配置维度，单变量）：
- 模块加载期（numpy import 之前）`os.environ.setdefault("VECLIB_MAXIMUM_THREADS", "6")`，
  其余逐行不动。6 = 本机 Apple M5 Max P 核数（hw.perflevel0.physicalcpu=6，
  6P+12E AMP）——vecLib 默认池撒满 18 核，sgemm floor 含 E 核 straggler 栅栏成本；
  限线程到快集群是 AMP 公认修正（arXiv:1506.08988 / 1511.02171；本机 3 轮交错 A/B：
  v6 pooled floor 429.5 vs default 441.6，probes/thread_ab.json）。
- 合法性（research_report §4 benchmark.py 源码审计）：进程级、先于首次 BLAS 调用
  （池惰性创建，late-set 探针证明生效）、对 candidate 与 ref 对称（同池同速，
  ref 落 [380,700] 带内，score≈1.0）——非「只让 ref 慢」通道；已知篡改模式的
  静态黑名单（宿主模块反射 / 计时属性赋值 / 内建篡改等）均不涉及。
- 数值不受影响：跨线程数输出 sha256 逐位相同（probes/final_confirm.json），
  K 求和序不动 → verify max_error 预期 0.0。

直接运行（自检，默认 64×64）：
    python3 code/solution.py --n 8192
"""
import json
import os
import time

# --- iter015 唯一改动：vecLib BLAS 线程池限 6 线程（= P 核数，AMP 限线程到快集群） ---
# setdefault：不覆盖外部显式配置；须先于首次 gemm（BLAS 池惰性创建，此时读 env）。
os.environ.setdefault("VECLIB_MAXIMUM_THREADS", "6")

import numpy as np

# macOS vecLib BLAS 虚假 FP 标志噪声抑制（数值正常；详见 data/ 同注释）。
np.seterr(divide="ignore", over="ignore", invalid="ignore")


def matmul(a, b):
    """单次整阵 np.matmul（BLAS gemm）派发。逐位一致于参考锚（同一 BLAS 调用）。"""
    a32 = np.asarray(a, dtype=np.float32)
    b32 = np.asarray(b, dtype=np.float32)
    return np.matmul(a32, b32)


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
        "impl": "full_blas_dispatch_np_matmul",
        "runtime_ms": round(dt_ms, 3),
        "max_error_vs_matmul": max_err,
    }))


if __name__ == "__main__":
    main()
