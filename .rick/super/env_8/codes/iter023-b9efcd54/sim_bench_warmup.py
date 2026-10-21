#!/usr/bin/env python3
"""sim_bench_warmup.py — 模拟 bench.py 协议，对比预热安放位置的变体分布。

镜像 data/bench.py 的精确顺序（rng/a/b 创建 → 加载候选模块级代码 → 候选 warmup+K=3 计时
→ ref warmup+K=3 计时），在独立进程中逐一测四种形态：
  A: iter003 形态（asarray+matmul，无预热）
  B: iter016 形态（预绑定 + 导入期 dummy 预热 4 次）
  C: 新形态（预绑定 + 首调自预热：warmup 调用内对真实 a/b 追加 3 次整阵乘）
  D: B+C 双保险（导入期 dummy 预热 + 首调自预热）

用法：python3 sim_bench_warmup.py <variant A|B|C|D>
输出：单行 JSON {variant, cand_timings, cand_median, ref_timings, ref_median, score}
"""
import json
import sys
from time import perf_counter as _perf

import numpy as np

np.seterr(divide="ignore", over="ignore", invalid="ignore")

variant = sys.argv[1] if len(sys.argv) > 1 else "A"
n, k = 2048, 3

# ---- bench.py 顺序：先建 rng/a/b，再加载候选（模块级代码此刻执行）----
rng = np.random.RandomState(42)
a = rng.rand(n, n).astype(np.float32)
b = rng.rand(n, n).astype(np.float32)

# ---- 候选模块级代码（模拟 import 期执行）----
if variant in ("B", "D"):
    _w = np.random.RandomState(7)
    _wa = _w.rand(n, n).astype(np.float32)
    _wb = _w.rand(n, n).astype(np.float32)
    for _ in range(4):
        np.matmul(_wa, _wb)
    del _w, _wa, _wb

_mm = np.matmul
_warmed = False


def cand_matmul(a, b):
    global _warmed
    if not _warmed:
        _warmed = True
        if variant in ("C", "D"):
            for _ in range(3):  # 首调自预热：真实输入，紧贴计时区
                _mm(a, b)
        elif variant == "A":
            pass
    if variant == "A":
        return _mm(np.asarray(a, dtype=np.float32), np.asarray(b, dtype=np.float32))
    return _mm(a, b)


def _median(vs):
    vs = sorted(vs)
    m = len(vs)
    return vs[m // 2] if m % 2 else (vs[m // 2 - 1] + vs[m // 2]) / 2.0


fn = cand_matmul
fn(a, b)  # 候选 warmup（未计时；C/D 在此内部追加自预热）
timings = []
for _ in range(k):
    t0 = _perf()
    fn(a, b)
    timings.append((_perf() - t0) * 1000.0)
cand_median = _median(timings)

ref = np.matmul
ref(a, b)
ref_timings = []
for _ in range(k):
    t0 = _perf()
    ref(a, b)
    ref_timings.append((_perf() - t0) * 1000.0)
ref_median = _median(ref_timings)

print(json.dumps({
    "variant": variant,
    "cand_timings": [round(t, 3) for t in timings],
    "cand_median": round(cand_median, 3),
    "ref_timings": [round(t, 3) for t in ref_timings],
    "ref_median": round(ref_median, 3),
    "score": round(ref_median / cand_median, 4),
}))
