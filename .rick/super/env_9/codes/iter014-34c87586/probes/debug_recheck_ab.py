#!/usr/bin/env python3
"""debug_recheck_ab.py — 容差带内复验（iter14 attempt 200）专用 A/B。

问题：recheck perf_gain_pct=-2.161（候选 440.14 vs 精英 430.83）是形态固有劣化，
还是跨会话机器漂移/噪声？persist 形态有没有被噪声掩盖的收益？

设计（fresh 进程，无顺序偏置）：
- 两臂交错 3 轮（正/反/正序），每臂每轮 1 预热 + 5 计时取中位。
- plain   = np.matmul(a, b)（精英形态原样，含 fresh 分配+缺页）
- persist = 惰性 per-shape 持久缓冲 + out=（solution.py 真实行为；分配发生在首调=预热内，
  计时窗只含复用派发）
- 判据：① 逐轮中位差与 pooled 差的符号/幅度；② 各臂 floor（全 timings 最小值）是否相同；
  ③ 有无任何计时 < 409ms（430.83×0.95 = 带外更优线，收益被掩盖检验）；
  ④ P1 与 P0 逐位一致（uint32 视图，out= 用法正确性）。
"""
import time

import numpy as np

np.seterr(divide="ignore", over="ignore", invalid="ignore")

n = 8192
rng = np.random.RandomState(42)
a = rng.rand(n, n).astype(np.float32)
b = rng.rand(n, n).astype(np.float32)

_BUFS = {}


def _get_buf(n_, m_):
    buf = _BUFS.get((n_, m_))
    if buf is None:
        buf = np.empty((n_, m_), dtype=np.float32)
        buf[:] = 0
        _BUFS[(n_, m_)] = buf
    return buf


def arm_plain():
    return np.matmul(a, b)


def arm_persist():
    c = _get_buf(n, n)
    np.matmul(a, b, out=c)
    return c


# ④ 正确性预检：逐位一致
r0 = arm_plain()
r1 = arm_persist()
print("bitwise persist_vs_plain=%s" % np.array_equal(
    r0.view(np.uint32), r1.view(np.uint32)))
del r0


def med5(fn):
    fn()  # 预热（persist 的首次分配在此窗口外）
    ts = []
    for _ in range(5):
        t0 = time.perf_counter()
        fn()
        ts.append((time.perf_counter() - t0) * 1e3)
    ts.sort()
    return ts[2], ts


arms = [("plain", arm_plain), ("persist", arm_persist)]
allts = {"plain": [], "persist": []}
for rnd, order in enumerate([arms, arms[::-1], arms], 1):
    for name, fn in order:
        m, ts = med5(fn)
        allts[name].extend(ts)
        print("round%d %-8s median=%.1f timings=%s" % (
            rnd, name, m, [round(x, 1) for x in ts]))

for name in ("plain", "persist"):
    v = allts[name]
    v_sorted = sorted(v)
    print("POOLED %-8s median=%.1f floor=%.1f max=%.1f n=%d" % (
        name, v_sorted[len(v) // 2], v_sorted[0], v_sorted[-1], len(v)))

BETTER_LINE = 430.8255829999998 * 0.95
below = [t for v in allts.values() for t in v if t < BETTER_LINE]
print("better_line=%.1fms readings_below=%d" % (BETTER_LINE, len(below)))
