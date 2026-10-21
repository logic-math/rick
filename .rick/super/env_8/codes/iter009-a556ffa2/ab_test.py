#!/usr/bin/env python3
"""ab_test.py — 迭代 9 A/B 归因临时脚本（不改变提交实现）。

目标（plan step 5）：把 swapped 拼写 halves（候选）vs canonical 拼写 halves
（iter007 预研基线 8.23ms）的差归因到两因子：
- gemm 标志差（swapped → C-标志 gemm vs canonical → T-标志 gemm，M=1024 半块规模）
- 装配差（C 连续块 concatenate = 纯 memcpy 0.234ms vs F 转置视图拼接 0.648ms）

设计：
- 配对 A/B：进程内、每轮随机次序 10 轮、每轮各变体 K=3 取中位
  （candidate: swapped halves einsum+concat；baseline: canonical halves einsum+concat）
- 归因矩阵变体：
    full_einsum_canonical : 父本 iter002 路径（整阵 canonical einsum，T-标志 gemm）
    full_einsum_swapped   : 整阵 swapped einsum（iter002 预研：canonical 快 ~0.15ms）
    gemm2_canonical       : 纯 2×1024 行 T-标志 gemm（dot(b.T, blk.T)，无包装无组装）
    gemm2_swapped         : 纯 2×1024 行 C-标志 gemm（matmul(blk, b)，无包装无组装）
    einsum2_swapped       : 两块 swapped einsum 无 concat（分离 einsum 包装层）
    halves_canonical_concat : iter007 预研 halves 基线形态（canonical 块 + concat F 视图）
    halves_swapped_concat : 候选 = 提交实现路径（swapped 块 + concat C 块）
    concat_CC_micro       : 微基准——两个预制 C 连续 (1024,2048) 块的 concatenate
    concat_FF_micro       : 微基准——两个预制 F 布局转置视图的 concatenate
- 正确性 sanity：@2048 逐位一致、双跑一致、嵌套 list、奇数 M（65 方阵 / 101×37
  非方阵）、m=1/2/3 退化、m=0。
"""
import json
import random
import time

import numpy as np

np.seterr(divide="ignore", over="ignore", invalid="ignore")

N = 2048
K = 3
ROUNDS = 10

rng = np.random.RandomState(42)
a32 = rng.rand(N, N).astype(np.float32)
b32 = rng.rand(N, N).astype(np.float32)

# ---------------- 候选与基线实现（与 solution.py 同式） ----------------
def candidate(a, b):
    """swapped 拼写 halves einsum + concat（= solution.py 提交路径）。"""
    a32_ = np.asarray(a, dtype=np.float32)
    b32_ = np.asarray(b, dtype=np.float32)
    h = a32_.shape[0] // 2
    r1 = np.einsum('jk,ij->ik', b32_, a32_[:h], optimize=True)
    r2 = np.einsum('jk,ij->ik', b32_, a32_[h:], optimize=True)
    return np.concatenate([r1, r2], axis=0)

def halves_canonical(a, b):
    """canonical 拼写 halves einsum + concat（iter007 预研基线 8.23ms 形态）。"""
    a32_ = np.asarray(a, dtype=np.float32)
    b32_ = np.asarray(b, dtype=np.float32)
    h = a32_.shape[0] // 2
    r1 = np.einsum('ij,jk->ik', a32_[:h], b32_, optimize=True)
    r2 = np.einsum('ij,jk->ik', a32_[h:], b32_, optimize=True)
    return np.concatenate([r1, r2], axis=0)

def full_canonical(a, b):
    return np.einsum('ij,jk->ik', a, b, optimize=True)

def full_swapped(a, b):
    return np.einsum('jk,ij->ik', b, a, optimize=True)

def gemm2_canonical(a, b):
    """纯 2×1024 行 T-标志 gemm（tensordot(b, blk) 等价 dot(b.T, blk.T)），无组装。"""
    h = a.shape[0] // 2
    return [np.dot(b.T, a[:h].T), np.dot(b.T, a[h:].T)]

def gemm2_swapped(a, b):
    """纯 2×1024 行 C-标志 gemm（tensordot(blk, b) 等价 matmul(blk, b)），无组装。"""
    h = a.shape[0] // 2
    return [np.matmul(a[:h], b), np.matmul(a[h:], b)]

def einsum2_swapped(a, b):
    """两块 swapped einsum 无 concat（分离 einsum 包装层）。"""
    h = a.shape[0] // 2
    return [np.einsum('jk,ij->ik', b, a[:h], optimize=True),
            np.einsum('jk,ij->ik', b, a[h:], optimize=True)]

# ---------------- 正确性 sanity ----------------
r_c = candidate(a32, b32)
r_m = np.matmul(a32, b32)
bitwise_2048 = bool(np.array_equal(r_c, r_m))
double_run = bool(np.array_equal(candidate(a32, b32), r_c))
# 布局断言：swapped 块 C 连续、canonical 块 F 转置视图
h = N // 2
blk_sw = np.einsum('jk,ij->ik', b32, a32[:h], optimize=True)
blk_ca = np.einsum('ij,jk->ik', a32[:h], b32, optimize=True)
layout_sw_c = bool(blk_sw.flags['C_CONTIGUOUS'])
layout_ca_f = bool(blk_ca.flags['F_CONTIGUOUS'] and not blk_ca.flags['C_CONTIGUOUS'])
# 嵌套 list
cl = candidate([[1.0, 2.0], [3.0, 4.0]], [[5.0, 6.0], [7.0, 8.0]])
nested_ok = bool(np.array_equal(cl, np.matmul(np.float32([[1, 2], [3, 4]]),
                                              np.float32([[5, 6], [7, 8]]))))
# 奇数 M：65 方阵 + 101×37 非方阵
odd_err = 0.0
for sa, sb in (((65, 65), (65, 65)), ((101, 37), (37, 50))):
    am = rng.rand(*sa).astype(np.float32)
    bm = rng.rand(*sb).astype(np.float32)
    odd_err = max(odd_err, float(np.max(np.abs(candidate(am, bm).astype(np.float64)
                                               - np.matmul(am, bm).astype(np.float64)))))
# m=1/2/3 退化 + m=0
deg_err = 0.0
for m in (0, 1, 2, 3):
    am = rng.rand(m, 5).astype(np.float32)
    bm = rng.rand(5, 7).astype(np.float32)
    out = candidate(am, bm)
    assert out.shape == (m, 7)
    if m:
        deg_err = max(deg_err, float(np.max(np.abs(out.astype(np.float64)
                                                   - np.matmul(am, bm).astype(np.float64)))))
print("sanity: bitwise@2048=%s double_run=%s nested_list=%s oddM_err=%.2e "
      "degenerate_err=%.2e | layout: swapped_blk_C=%s canonical_blk_F=%s"
      % (bitwise_2048, double_run, nested_ok, odd_err, deg_err,
         layout_sw_c, layout_ca_f))

# ---------------- concat 微基准（预制缓冲，只计时拼接） ----------------
cc1 = np.matmul(a32[:h], b32)            # (1024, 2048) C 连续
cc2 = np.matmul(a32[h:], b32)
ff1 = np.dot(b32.T, a32[:h].T).T          # (1024, 2048) F 布局转置视图
ff2 = np.dot(b32.T, a32[h:].T).T
assert cc1.flags['C_CONTIGUOUS'] and ff1.flags['F_CONTIGUOUS']
assert np.array_equal(cc1.astype(np.float64), ff1.astype(np.float64))

def concat_cc(a=None, b=None):
    return np.concatenate([cc1, cc2], axis=0)

def concat_ff(a=None, b=None):
    return np.concatenate([ff1, ff2], axis=0)

# ---------------- 计时框架 ----------------
FUNCS = {
    "full_einsum_canonical": full_canonical,
    "full_einsum_swapped": full_swapped,
    "gemm2_canonical": gemm2_canonical,
    "gemm2_swapped": gemm2_swapped,
    "einsum2_swapped": einsum2_swapped,
    "halves_canonical_concat": halves_canonical,
    "halves_swapped_concat": candidate,
    "concat_CC_micro": concat_cc,
    "concat_FF_micro": concat_ff,
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

results = {name: [] for name in FUNCS}
order = random.Random(321)
for _ in range(ROUNDS):
    names = list(FUNCS)
    order.shuffle(names)
    for name in names:
        results[name].append(median_k(FUNCS[name]))

summary = {"sanity": {"bitwise_2048": bitwise_2048, "double_run": double_run,
                      "nested_list": nested_ok, "odd_m_max_error": odd_err,
                      "degenerate_max_error": deg_err,
                      "swapped_blk_C_contig": layout_sw_c,
                      "canonical_blk_F_layout": layout_ca_f}}
for name, ts in results.items():
    summary[name] = {"median_of_medians": round(float(np.median(ts)), 3),
                     "min": round(min(ts), 3), "max": round(max(ts), 3)}

# 配对差（同轮同序消负载漂移）
def diffs(na, nb):
    return [round(b - a, 3) for a, b in zip(results[na], results[nb])]

summary["canonical_minus_swapped_halves_per_round"] = diffs(
    "halves_swapped_concat", "halves_canonical_concat")
summary["swapped_faster_rounds"] = int(sum(
    1 for d in summary["canonical_minus_swapped_halves_per_round"] if d > 0))
summary["full_minus_candidate_per_round"] = diffs(
    "halves_swapped_concat", "full_einsum_canonical")

print(json.dumps(summary, ensure_ascii=False))
