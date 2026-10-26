#!/usr/bin/env python3
"""buffer_ab.py — iter14 builder 探针：V1（持久 out=）「恒慢 ~14%」异常的机制定位 A/B。

背景（iter13 out_ab.py 读数，同进程交错）：
  V0 plain            436ms
  V1 持久预 fault out= 498-523ms（+62-87ms）
  V2 fresh+memset+out= 436ms（平价）
矛盾：V1 与 V2 同为 matmul+out= 同 sgemm，唯一差异「复用同一物理页 vs 每次新页」。
本探针定位机制：物理布局（缓冲分配位置） vs numpy matmul-out 派发机制 vs 长驻脏页写带宽。

臂（同进程交错，round 正/反/正序消位置偏置；每臂每轮 1 预热 + 5 计时取中位 + ru_minflt/call）：
  P0 plain    : np.matmul(a, b)               ——基线（精英形态；每次调用内部 fresh 分配+缺页）
  P1 post     : matmul(a,b,out=cbuf_post)     ——iter13 V1 精确复刻（缓冲分配在 b 之后=候选真实情境）
  P2 fresh    : np.empty+c[:]=0+out=          ——iter13 V2（已知平价参照）
  P5 pre      : matmul(a,b,out=cbuf_pre)      ——布局对照（缓冲分配在 a/b 之前；候选不可实现，纯机制判别）
  P6 spacer   : matmul(a,b,out=cbuf_sp)       ——候选可实现布局规避（1GB spacer 后分配）
  P7 dot_post : np.dot(a,b,out=cbuf_post)     ——机制判别：dot 直调路径+同 P1 缓冲（快→matmul out= 机制层；慢→物理层）

正确性预检：P1 输出与 P0 逐位一致（uint32 视图）。读数即记录，不复跑。
"""
import resource
import time

import numpy as np

np.seterr(divide="ignore", over="ignore", invalid="ignore")

n = 8192
rng = np.random.RandomState(42)

# ---- 分配顺序决定物理布局 ----
cbuf_pre = np.empty((n, n), np.float32)
cbuf_pre[:] = 0  # 预 fault（在 a/b 之前）

a = rng.rand(n, n).astype(np.float32)
b = rng.rand(n, n).astype(np.float32)

cbuf_post = np.empty((n, n), np.float32)
cbuf_post[:] = 0  # 预 fault（紧随 b 之后 = iter13 V1 布局 = 候选真实情境）

junk = np.empty(1 << 28, np.float32)  # 1GB spacer（整体触页）
junk[:] = 0
cbuf_sp = np.empty((n, n), np.float32)
cbuf_sp[:] = 0  # 预 fault（spacer 之后）


def rusage_pages():
    ru = resource.getrusage(resource.RUSAGE_SELF)
    return ru.ru_minflt, ru.ru_majflt


def med5(fn):
    fn()  # 预热
    m0 = rusage_pages()
    ts = []
    for _ in range(5):
        t0 = time.perf_counter()
        fn()
        ts.append((time.perf_counter() - t0) * 1e3)
    m1 = rusage_pages()
    ts.sort()
    return ts[2], ts, (m1[0] - m0[0]) / 5.0, (m1[1] - m0[1]) / 5.0


# ---- 正确性预检：P1 vs P0 逐位 ----
r0 = np.matmul(a, b)
np.matmul(a, b, out=cbuf_post)
print("bitwise P1_vs_P0=%s" % np.array_equal(r0.view(np.uint32), cbuf_post.view(np.uint32)))
del r0


def p0():
    return np.matmul(a, b)


def p1():
    np.matmul(a, b, out=cbuf_post)


def p2():
    c = np.empty((n, n), np.float32)
    c[:] = 0
    np.matmul(a, b, out=c)


def p5():
    np.matmul(a, b, out=cbuf_pre)


def p6():
    np.matmul(a, b, out=cbuf_sp)


def p7():
    np.dot(a, b, out=cbuf_post)


def memcpy_marker():
    src = np.empty((n, n), np.float32)
    src[:] = 1.0
    dst = np.empty((n, n), np.float32)
    t0 = time.perf_counter()
    dst[:] = src
    dt = (time.perf_counter() - t0) * 1e3
    return dt


print("drift_start memcpy256MB=%.1fms" % memcpy_marker())

arms = [("P0_plain", p0), ("P1_post", p1), ("P2_fresh", p2),
        ("P5_pre", p5), ("P6_spacer", p6), ("P7_dot_post", p7)]
for rnd, order in enumerate([arms, arms[::-1], arms], 1):
    for name, fn in order:
        m, ts, mf, mj = med5(fn)
        print("round%d %-11s median=%.1f timings=%s minflt/call=%.0f majflt/call=%.0f"
              % (rnd, name, m, [round(x, 1) for x in ts], mf, mj))

print("drift_end memcpy256MB=%.1fms" % memcpy_marker())
print("numpy=%s pagesize=%d" % (np.__version__, resource.getpagesize()))
