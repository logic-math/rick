#!/usr/bin/env python3
"""ab_test.py — 迭代 4 A/B 归因临时脚本（不改变提交实现）。

设计：进程内配对、随机次序 8 对、每对 K=3 取中位：
- candidate: solution.py matmul —— 逐行向量化 einsum（'j,jk->k' canonical，逐行 T-标志 gemm N=1）
- naive:     naive_start.py matmul —— rowwise np.dot（不预转置 B）
- parent:    iter002 matmul —— 整阵 canonical einsum（'ij,jk->ik'，T-标志 gemm）

两因素量级模型（可证伪）：
- rowwise 内存墙：每输出行完整流过 B（2048×16MB=32GB 流量）→ naive ~144ms 结构性下界；
- einsum 包装税：@64 微基准 ~11.3μs/调用 vs np.dot 0.55μs，@2048 折 ~13μs/行 ×2048≈27ms；
- 预期：candidate ≈ naive + 20-35ms（配对差带），且 ≈ parent + ~160ms。
"""
import importlib.util
import json
import os
import random
import time

import numpy as np

np.seterr(divide="ignore", over="ignore", invalid="ignore")

HERE = os.path.dirname(os.path.abspath(__file__))
N = 2048
K = 3
PAIRS = 8

# 候选实现：从 solution.py 加载（提交实现本体）
spec = importlib.util.spec_from_file_location("candidate", os.path.join(HERE, "solution.py"))
cand_mod = importlib.util.module_from_spec(spec)
spec.loader.exec_module(cand_mod)


def candidate(a, b):
    return cand_mod.matmul(a, b)


def naive(a, b):
    """naive_start.py 的 matmul：rowwise np.dot（不预转置 B）。"""
    a32 = np.asarray(a, dtype=np.float32)
    b32 = np.asarray(b, dtype=np.float32)
    return np.stack([np.dot(a32[i], b32) for i in range(a32.shape[0])])


def parent(a, b):
    """iter002 的 matmul：整阵 canonical einsum 派发（T-标志 gemm）。"""
    a32 = np.asarray(a, dtype=np.float32)
    b32 = np.asarray(b, dtype=np.float32)
    return np.einsum('ij,jk->ik', a32, b32, optimize=True)


FUNCS = {"candidate": candidate, "naive": naive, "parent": parent, "matmul": np.matmul}

rng = np.random.RandomState(42)
a32 = rng.rand(N, N).astype(np.float32)
b32 = rng.rand(N, N).astype(np.float32)

# 预热各 2 次
for f in FUNCS.values():
    f(a32, b32); f(a32, b32)

# 正确性 sanity：全部逐位一致于 np.matmul
r_c, r_n, r_p, r_m = (f(a32, b32) for f in (candidate, naive, parent, np.matmul))
print("sanity bitwise: cand==mat %s, naive==mat %s, parent==mat %s" % (
    np.array_equal(r_c, r_m), np.array_equal(r_n, r_m), np.array_equal(r_p, r_m)))


def median3(fn):
    ts = []
    for _ in range(K):
        t0 = time.perf_counter()
        fn(a32, b32)
        ts.append((time.perf_counter() - t0) * 1000.0)
    ts.sort()
    return ts[K // 2]


results = {name: [] for name in FUNCS}
order = random.Random(404)
for pair in range(PAIRS):
    names = list(FUNCS)
    order.shuffle(names)
    for name in names:
        results[name].append(median3(FUNCS[name]))

summary = {}
for name, ts in results.items():
    summary[name] = {
        "median_of_medians": round(float(np.median(ts)), 3),
        "min": round(min(ts), 3),
        "max": round(max(ts), 3),
        "medians": [round(t, 3) for t in ts],
    }

# 配对差：包装税（candidate − naive）与结构差（candidate − parent）
tax = [c - n for c, n in zip(results["candidate"], results["naive"])]
wall = [c - p for c, p in zip(results["candidate"], results["parent"])]
summary["candidate_minus_naive_per_pair"] = [round(d, 3) for d in tax]
summary["candidate_minus_parent_per_pair"] = [round(d, 3) for d in wall]
summary["wrapper_tax_mean_ms"] = round(float(np.mean(tax)), 3)
summary["wrapper_tax_min_ms"] = round(float(min(tax)), 3)
summary["wrapper_tax_max_ms"] = round(float(max(tax)), 3)
summary["pairs_candidate_slower_than_naive"] = int(sum(1 for d in tax if d > 0))
summary["structure_gap_mean_ms"] = round(float(np.mean(wall)), 3)

print(json.dumps(summary, ensure_ascii=False))
