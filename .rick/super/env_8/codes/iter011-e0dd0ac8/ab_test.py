#!/usr/bin/env python3
"""ab_test.py — 迭代 11 A/B 归因临时脚本（不改变提交实现）。

目标（plan step 3/4/5）：
- 正确性 sanity：奇数行 257×130@130×129、非方阵、1-D b、嵌套 list、双跑一致
  ——验证行四分边界无缝覆盖 + np.matmul out= 切片直写未脱离 cblas_sgemm 快路径
  （gufunc 慢回退会在 runtime 上暴露，>50ms 量级）。
- 配对 A/B（同进程 k=9 中位 × 多轮）：四变体对照
    quarters_matmul_out : 本候选（iter008 m_halves 骨架 + np.matmul 派发 + 行四分）
    halves_matmul_out   : 父本 iter008 形态（同派发入口 + 行二分）
    quarters_dot_out    : iter010 形态（np.dot 派发 + 行四分）
    full_matmul         : 阶梯锚点（单次整阵 np.matmul）
- 两个量级判断（plan step 4）：
    ① 2→4 分裂结构性损失：quarters_matmul − halves_matmul ≈ +0.8~1.1ms
    ② np.dot→np.matmul 派发层差异：quarters_matmul − quarters_dot ≈ −0.05~−0.15ms
- 阶梯对照：full ~6.9 / halves ~7.4 / quarters ~8.4（M 粒度单调上升验证）。
"""
import time

import numpy as np

np.seterr(divide="ignore", over="ignore", invalid="ignore")

N = 2048
K = 9
ROUNDS = 5


def quarters_matmul_out(a32, b32):
    """本候选形态：行四分 + np.matmul out= 直写。"""
    m = a32.shape[0]
    if m == 0:
        return np.zeros((0,) + b32.shape[1:], dtype=np.float32)
    c = np.empty((m,) + b32.shape[1:], dtype=np.float32)
    edges = [(i * m // 4, (i + 1) * m // 4) for i in range(4)]
    for s, e in edges:
        if e > s:
            np.matmul(a32[s:e], b32, out=c[s:e])
    return c


def halves_matmul_out(a32, b32):
    """父本 iter008 形态：行二分 + np.matmul out= 直写。"""
    m = a32.shape[0]
    h = m // 2
    c = np.empty((m,) + b32.shape[1:], dtype=np.float32)
    np.matmul(a32[:h], b32, out=c[:h])
    np.matmul(a32[h:], b32, out=c[h:])
    return c


def quarters_dot_out(a32, b32):
    """iter010 形态：行四分 + np.dot out= 直写。"""
    m = a32.shape[0]
    if m == 0:
        return np.zeros((0,) + b32.shape[1:], dtype=np.float32)
    c = np.empty((m,) + b32.shape[1:], dtype=np.float32)
    edges = [(i * m // 4, (i + 1) * m // 4) for i in range(4)]
    for s, e in edges:
        if e > s:
            np.dot(a32[s:e], b32, out=c[s:e])
    return c


def full_matmul(a32, b32):
    """阶梯锚点：单次整阵 np.matmul。"""
    return np.matmul(a32, b32)


# ---------- 正确性 sanity（plan step 3） ----------
print("== 正确性 sanity ==")
rng = np.random.RandomState(7)
cases = [
    ("odd_rows_257x130@130x129", (257, 130), (130, 129)),
    ("tall_130x64@64x200", (130, 64), (64, 200)),
    ("wide_64x300@300x32", (64, 300), (300, 32)),
    ("square_2048", (2048, 2048), (2048, 2048)),
    ("1d_b_128x64@64", (128, 64), None),
    ("odd_rows_257x64_1d_b", (257, 64), None),
    ("nested_list_input", (100, 50), (50, 40)),
    ("m_lt4_3x5@5x7", (3, 5), (5, 7)),
    ("m_eq0_0x8@8x9", (0, 8), (8, 9)),
]
all_ok = True
for name, sa, sb in cases:
    a = rng.rand(*sa).astype(np.float32)
    if sb is None:
        b = rng.rand(sa[1]).astype(np.float32)
    else:
        b = rng.rand(*sb).astype(np.float32)
    if name == "nested_list_input":
        a_in, b_in = a.tolist(), b.tolist()
    else:
        a_in, b_in = a, b
    c = quarters_matmul_out(np.asarray(a_in, dtype=np.float32),
                            np.asarray(b_in, dtype=np.float32))
    ref = np.matmul(np.asarray(a_in, dtype=np.float32),
                    np.asarray(b_in, dtype=np.float32))
    c2 = quarters_matmul_out(np.asarray(a_in, dtype=np.float32),
                             np.asarray(b_in, dtype=np.float32))
    assert c.shape == ref.shape and c.dtype == np.float32
    if c.size == 0:
        max_err = 0.0  # 空输出（m==0 退化守卫）无可比元素
    else:
        max_err = float(np.max(np.abs(c.astype(np.float64) - ref.astype(np.float64))))
    bitstable = bool(np.array_equal(c, c2))
    # 判据 = env 容差（1e-3）+ 双跑一致；位精确性单独标注（sgemm 主路径应为 0.0）
    ok = (max_err <= 1e-3) and bitstable
    bitexact = (max_err == 0.0)
    all_ok = all_ok and ok
    print(f"{name:28s} shape={str(c.shape):12s} max_err={max_err:.3e} 双跑一致={bitstable} "
          f"位精确={'Y' if bitexact else 'N'} [{'OK' if ok else 'FAIL'}]")
print("SANITY:", "ALL_PASS" if all_ok else "SOME_FAIL",
      "（判据 = env 容差 1e-3 + 双跑一致；2-D sgemm 主路径应为位精确 0.0，"
      "1-D gemv 32 行块 / m<4 单行块的 ε 级差异见 ab_notes 归因）")

# ---------- A/B 计时（plan step 4：同进程 k=9 中位 × 5 轮，轮内固定次序） ----------
print()
print(f"== A/B 计时 @2048 同进程 k={K} × {ROUNDS} 轮（中位） ==")
rng = np.random.RandomState(42)
a = rng.rand(N, N).astype(np.float32)
b = rng.rand(N, N).astype(np.float32)
ref = np.matmul(a, b)

variants = [
    ("quarters_matmul_out", quarters_matmul_out),
    ("halves_matmul_out", halves_matmul_out),
    ("quarters_dot_out", quarters_dot_out),
    ("full_matmul", full_matmul),
]

# 预热全部变体（首轮计时前消除 import/JIT 差异）
for _, fn in variants:
    fn(a, b)

names = [name for name, _ in variants]
rounds = []
for r in range(ROUNDS):
    row = {"round": r + 1}
    for name, fn in variants:
        c = fn(a, b)
        max_err = float(np.max(np.abs(c.astype(np.float64) - ref.astype(np.float64))))
        timings = []
        for _ in range(K):
            t0 = time.perf_counter()
            fn(a, b)
            timings.append((time.perf_counter() - t0) * 1000.0)
        timings.sort()
        med = timings[len(timings) // 2]
        row[name] = {"median_ms": med, "min_ms": timings[0], "max_ms": timings[-1],
                     "max_err": max_err}
    rounds.append(row)
    print(f"轮 {r+1}: " + "  ".join(
        f"{name}={row[name]['median_ms']:.3f}" for name in names))

# ---------- 汇总（轮中位的中位） ----------
def med_of_meds(name):
    meds = sorted(row[name]["median_ms"] for row in rounds)
    return meds[len(meds) // 2]

summary = {name: med_of_meds(name) for name in names}
all_err_zero = all(row[n]["max_err"] == 0.0 for row in rounds for n in names)
qm, hm, qd, fm = (summary[n] for n in names)

print()
print(f"== 汇总（{ROUNDS} 轮中位之中位） ==")
for name in names:
    print(f"{name:20s} median_of_medians={summary[name]:.3f}ms")
print(f"全部变体 max_err==0.0 : {'PASS' if all_err_zero else 'FAIL'}")

d_split = qm - hm    # 判据①：2→4 分裂结构性损失
d_dispatch = qm - qd  # 判据②：np.dot→np.matmul 派发层差异（负=matmul 快）
print()
print(f"判据① 2→4 分裂损失（quarters_matmul − halves_matmul）: {d_split:+.3f}ms "
      f"(预期 +0.8~+1.1) [{'PASS' if 0.8 <= d_split <= 1.1 else 'OUT_OF_BAND'}]")
print(f"判据② 派发层差异（quarters_matmul − quarters_dot）  : {d_dispatch:+.3f}ms "
      f"(预期 −0.05~−0.15) [{'PASS' if -0.15 <= d_dispatch <= -0.05 else 'OUT_OF_BAND'}]")
print(f"阶梯: full={fm:.3f} < halves={hm:.3f} < quarters={qm:.3f} "
      f"(单调性 {'成立' if fm < hm < qm else '不成立'})")
print(f"候选 runtime 带（8.2~8.9）: {qm:.3f}ms "
      f"[{'PASS' if 8.2 <= qm <= 8.9 else 'OUT_OF_BAND'}]")
