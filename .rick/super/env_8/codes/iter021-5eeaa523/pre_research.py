#!/usr/bin/env python3
"""pre_research.py — 迭代21预研：持久 out 缓冲 vs 每调用分配的配对 A/B。

机制问题：np.dot 整阵派发下，输出 16MB 的「分配 + 首触缺页」成本是 0（libmalloc
大块回收、页保持热）还是 >0.1ms（每次 mmap 重缺页）？iter020 的 out= 实验用
「每调用 np.empty」——与内部分配同付分配税，无法区分。本预研加入第三个变量：
模块级持久缓冲（跨调用复用，页恒热）。

三臂：
  P  父本机制：np.dot(a32, b32)          —— C 层内部分配（bench 丢弃返回值 → 每次分配+释放）
  E  iter020：  np.empty + np.dot(out=)  —— 每调用显式分配
  C  本预研：  持久 buf + np.dot(out=)   —— 只分配一次，页恒热

交错配对 40 轮（P/E/C 同轮相邻，负载态共享），报中位与胜率。
"""
import json
import time

import numpy as np

np.seterr(divide="ignore", over="ignore", invalid="ignore")

N = 2048
ROUNDS = 40

rng = np.random.RandomState(42)
a = rng.rand(N, N).astype(np.float32)
b = rng.rand(N, N).astype(np.float32)

_buf = np.empty((N, N), dtype=np.float32)  # 持久缓冲（预触：写一次让页全部就位）
_buf[:] = 0.0


def p_impl(a, b):
    return np.dot(a, b)


def e_impl(a, b):
    c = np.empty((N, N), dtype=np.float32)
    return np.dot(a, b, out=c)


def c_impl(a, b):
    return np.dot(a, b, out=_buf)


def time_once(fn):
    t0 = time.perf_counter()
    fn(a, b)
    return (time.perf_counter() - t0) * 1000.0


# 预热：每臂 3 次（BLAS 初始化 + 分配器稳态）
for fn in (p_impl, e_impl, c_impl):
    for _ in range(3):
        fn(a, b)

res = {"P": [], "E": [], "C": []}
for _ in range(ROUNDS):
    for key, fn in (("P", p_impl), ("E", e_impl), ("C", c_impl)):
        res[key].append(time_once(fn))


def median(v):
    s = sorted(v)
    m = len(s)
    return s[m // 2] if m % 2 else (s[m // 2 - 1] + s[m // 2]) / 2.0


out = {"rounds": ROUNDS, "n": N}
for k in ("P", "E", "C"):
    out[k + "_median_ms"] = round(median(res[k]), 4)
    out[k + "_min_ms"] = round(min(res[k]), 4)
    out[k + "_max_ms"] = round(max(res[k]), 4)
out["C_vs_P_delta_ms"] = round(median(res["C"]) - median(res["P"]), 4)
out["C_beats_P_rounds"] = sum(1 for pc, cc in zip(res["P"], res["C"]) if cc < pc)
out["E_vs_P_delta_ms"] = round(median(res["E"]) - median(res["P"]), 4)
print(json.dumps(out))
