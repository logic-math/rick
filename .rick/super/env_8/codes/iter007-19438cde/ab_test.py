#!/usr/bin/env python3
"""ab_test.py — 迭代 7 A/B 归因临时脚本（不改变提交实现）。

目标（plan step 4）：把 m_quarters（候选）比父本 full 慢的差距归因到三因子：
- sgemm 分裂损失（4×512 行 vs 1×2048 行，vecLib/AMX 小 M 内核效率）
- einsum 包装 ×4（parse+path+tensordot 派发，每块一次）
- concatenate 16MB 拼接拷贝

设计：
- 配对 A/B：进程内、每轮随机次序 10 对、每对 K=3 取中位
  （candidate: m_quarters einsum+concat；parent: 整阵 canonical einsum）
- 微基准三档分解（每档 warmup2 + K=3 × 10 组取中位之中位）：
    gemm4_noassemble : 纯 4×512 sgemm（np.dot T-标志，无组装、无 einsum 包装）
    matmul_q_concat  : np.matmul 分块 + concatenate（无 einsum 包装，有拼接）
    einsum_q_concat  : einsum 分块 + concatenate（= 提交实现路径）
  对照锚：full_matmul / full_einsum（父本路径）
- 正确性 sanity：逐位/容差校验（嵌套 list、非方阵、m=1/2/3 退化、双跑一致）
"""
import json
import random
import time

import numpy as np

np.seterr(divide="ignore", over="ignore", invalid="ignore")

N = 2048
K = 3
PAIRS = 10

rng = np.random.RandomState(42)
a32 = rng.rand(N, N).astype(np.float32)
b32 = rng.rand(N, N).astype(np.float32)

# ---------------- 正确性 sanity ----------------
def _edges(m):
    return [(i * m // 4, (i + 1) * m // 4) for i in range(4)]

def candidate(a, b):
    a32_ = np.asarray(a, dtype=np.float32)
    b32_ = np.asarray(b, dtype=np.float32)
    m = a32_.shape[0]
    blocks = [np.einsum('ij,jk->ik', a32_[s:e], b32_, optimize=True)
              for s, e in _edges(m) if e > s]
    if not blocks:
        return np.zeros((0, b32_.shape[1]), dtype=np.float32)
    return np.concatenate(blocks, axis=0)

def parent_full(a, b):
    a32_ = np.asarray(a, dtype=np.float32)
    b32_ = np.asarray(b, dtype=np.float32)
    return np.einsum('ij,jk->ik', a32_, b32_, optimize=True)

# @2048 逐位一致 + 双跑一致
r_c, r_p, r_m = candidate(a32, b32), parent_full(a32, b32), np.matmul(a32, b32)
bitwise_2048 = bool(np.array_equal(r_c, r_m))
double_run = bool(np.array_equal(candidate(a32, b32), r_c))
# 嵌套 list
cl = candidate([[1.0, 2.0], [3.0, 4.0]], [[5.0, 6.0], [7.0, 8.0]])
nested_ok = bool(np.array_equal(cl, np.matmul(np.float32([[1, 2], [3, 4]]), np.float32([[5, 6], [7, 8]]))))
# 非方阵 + M 不整除
a_ns = rng.rand(100, 37).astype(np.float32); b_ns = rng.rand(37, 50).astype(np.float32)
ns_err = float(np.max(np.abs(candidate(a_ns, b_ns).astype(np.float64)
                             - np.matmul(a_ns, b_ns).astype(np.float64))))
# m=1/2/3 退化块
deg_err = 0.0
for m in (1, 2, 3):
    am = rng.rand(m, 5).astype(np.float32); bm = rng.rand(5, 7).astype(np.float32)
    deg_err = max(deg_err, float(np.max(np.abs(candidate(am, bm).astype(np.float64)
                                              - np.matmul(am, bm).astype(np.float64)))))
print("sanity: bitwise@2048=%s double_run=%s nested_list=%s "
      "nonsquare_err=%.2e degenerate_err=%.2e"
      % (bitwise_2048, double_run, nested_ok, ns_err, deg_err))

# ---------------- 微基准三档分解 ----------------
def gemm4_noassemble(a, b):
    """纯 4×512 sgemm（T-标志 dot：dot(b.T 视图, blk.T 视图)），无组装无 einsum 包装。"""
    m = a.shape[0]
    out = []
    for s, e in _edges(m):
        if e > s:
            out.append(np.dot(b.T, a[s:e].T))
    return out  # 故意不组装（结果丢弃由 bench 层使用，勿作正确性比较）

def matmul_q_concat(a, b):
    m = a.shape[0]
    blocks = [np.matmul(a[s:e], b) for s, e in _edges(m) if e > s]
    return np.concatenate(blocks, axis=0)

def einsum_q_concat(a, b):
    m = a.shape[0]
    blocks = [np.einsum('ij,jk->ik', a[s:e], b, optimize=True)
              for s, e in _edges(m) if e > s]
    return np.concatenate(blocks, axis=0)

FUNCS = {
    "gemm4_noassemble": gemm4_noassemble,
    "matmul_q_concat": matmul_q_concat,
    "einsum_q_concat": einsum_q_concat,
    "full_matmul": np.matmul,
    "full_einsum": parent_full,
}

for f in FUNCS.values():
    f(a32, b32); f(a32, b32)  # 预热 ×2

def median_k(fn):
    ts = []
    for _ in range(K):
        t0 = time.perf_counter()
        fn(a32, b32)
        ts.append((time.perf_counter() - t0) * 1000.0)
    ts.sort()
    return ts[K // 2]

# 三档微基准：随机次序 ×10 组
micro_results = {name: [] for name in FUNCS}
order = random.Random(321)
for _ in range(PAIRS):
    names = list(FUNCS)
    order.shuffle(names)
    for name in names:
        micro_results[name].append(median_k(FUNCS[name]))

# ---------------- 配对 A/B：candidate vs parent ----------------
ab = {"candidate_m_quarters": [], "parent_full_einsum": []}
order_ab = random.Random(123)
for _ in range(PAIRS):
    names = list(ab)
    order_ab.shuffle(names)
    for name in names:
        fn = candidate if name.startswith("candidate") else parent_full
        ab[name].append(median_k(fn))

summary = {"sanity": {"bitwise_2048": bitwise_2048, "double_run": double_run,
                      "nested_list": nested_ok, "nonsquare_max_error": ns_err,
                      "degenerate_max_error": deg_err}}
for name, ts in micro_results.items():
    summary[name] = {"median_of_medians": round(float(np.median(ts)), 3),
                     "min": round(min(ts), 3), "max": round(max(ts), 3)}
for name, ts in ab.items():
    summary[name] = {"median_of_medians": round(float(np.median(ts)), 3),
                     "min": round(min(ts)), "max": round(max(ts)),
                     "medians": [round(t, 3) for t in ts]}

diffs = [p - c for c, p in zip(ab["candidate_m_quarters"], ab["parent_full_einsum"])]
summary["parent_minus_candidate_per_pair"] = [round(d, 3) for d in diffs]
summary["candidate_slower_mean_ms"] = round(float(np.mean(diffs)), 3)
summary["candidate_slower_median_ms"] = round(float(np.median(diffs)), 3)
summary["pairs_candidate_slower"] = int(sum(1 for d in diffs if d > 0))

print(json.dumps(summary, ensure_ascii=False))
