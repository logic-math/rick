#!/usr/bin/env python3
"""ab_test.py — 迭代 8 A/B 归因临时脚本（不改变提交实现）。

目标（plan step 3/5）：
- 正确性 sanity：奇数行 257×130@130×129、非方阵、1-D b、嵌套 list、双跑一致
  ——验证 np.matmul out= 切片直写未脱离 cblas_sgemm 快路径（gufunc 慢回退会在
  runtime 上暴露，>50ms 量级）。
- 配对 A/B（同进程 k=9 中位）：四变体对照
    halves_matmul_out : 本候选（iter001 m_halves 骨架 + np.matmul 派发入口 + out= 直写）
    halves_dot_out    : 父本 iter001（同骨架 + np.dot 派发入口）
    full_matmul       : iter003 形态（单次整阵 np.matmul，派发入口证据源）
    concat_golden     : 金样例（np.matmul 分块 + concatenate 拼接）
- 成功判据（plan step 5）：候选 max_err=0.0、runtime 落 7.2-7.6ms 带、
  比 concat 快 ≥0.25ms、不比 halves_dot_out 慢 >0.2ms。
"""
import time

import numpy as np

np.seterr(divide="ignore", over="ignore", invalid="ignore")

N = 2048
K = 9


def halves_matmul_out(a32, b32):
    m = a32.shape[0]
    h = m // 2
    c = np.empty((m,) + b32.shape[1:], dtype=np.float32)
    np.matmul(a32[:h], b32, out=c[:h])
    np.matmul(a32[h:], b32, out=c[h:])
    return c


def halves_dot_out(a32, b32):
    m = a32.shape[0]
    h = m // 2
    c = np.empty((m,) + b32.shape[1:], dtype=np.float32)
    np.dot(a32[:h], b32, out=c[:h])
    np.dot(a32[h:], b32, out=c[h:])
    return c


def full_matmul(a32, b32):
    return np.matmul(a32, b32)


def concat_golden(a32, b32):
    m = a32.shape[0]
    h = m // 2
    return np.concatenate(
        (np.matmul(a32[:h], b32), np.matmul(a32[h:], b32)), axis=0
    )


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
]
all_ok = True
for name, sa, sb in cases:
    a = rng.rand(*sa).astype(np.float32)
    b = rng.rand(sa[1] if sb is None else sb[0]).astype(np.float32) if sb is None \
        else rng.rand(*sb).astype(np.float32)
    if name == "nested_list_input":
        a_in, b_in = a.tolist(), b.tolist()
    else:
        a_in, b_in = a, b
    c = halves_matmul_out(np.asarray(a_in, dtype=np.float32),
                          np.asarray(b_in, dtype=np.float32))
    ref = np.matmul(np.asarray(a_in, dtype=np.float32),
                    np.asarray(b_in, dtype=np.float32))
    c2 = halves_matmul_out(np.asarray(a_in, dtype=np.float32),
                           np.asarray(b_in, dtype=np.float32))
    assert c.shape == ref.shape and c.dtype == np.float32
    max_err = float(np.max(np.abs(c.astype(np.float64) - ref.astype(np.float64))))
    bitstable = bool(np.array_equal(c, c2))
    ok = (max_err == 0.0) and bitstable
    all_ok = all_ok and ok
    print(f"{name:28s} shape={str(c.shape):12s} max_err={max_err} 双跑一致={bitstable} [{'OK' if ok else 'FAIL'}]")
print("SANITY:", "ALL_PASS" if all_ok else "SOME_FAIL")

# ---------- A/B 计时（plan step 5：同进程 k=9 中位） ----------
print()
print(f"== A/B 计时 @2048 同进程 k={K}（中位） ==")
rng = np.random.RandomState(42)
a = rng.rand(N, N).astype(np.float32)
b = rng.rand(N, N).astype(np.float32)
ref = np.matmul(a, b)

variants = [
    ("halves_matmul_out", halves_matmul_out),
    ("halves_dot_out", halves_dot_out),
    ("full_matmul", full_matmul),
    ("concat_golden", concat_golden),
]
results = {}
for name, fn in variants:
    fn(a, b)  # 预热
    c = fn(a, b)
    max_err = float(np.max(np.abs(c.astype(np.float64) - ref.astype(np.float64))))
    timings = []
    for _ in range(K):
        t0 = time.perf_counter()
        fn(a, b)
        timings.append((time.perf_counter() - t0) * 1000.0)
    timings.sort()
    med = timings[len(timings) // 2]
    results[name] = {"median_ms": med, "min_ms": timings[0], "max_ms": timings[-1],
                     "max_err": max_err}
    print(f"{name:20s} median={med:.3f}ms min={timings[0]:.3f} max={timings[-1]:.3f} max_err={max_err}")

med = results["halves_matmul_out"]["median_ms"]
err = results["halves_matmul_out"]["max_err"]
concat_med = results["concat_golden"]["median_ms"]
dot_med = results["halves_dot_out"]["median_ms"]
full_med = results["full_matmul"]["median_ms"]

ok_err = err == 0.0
ok_band = 7.2 <= med <= 7.6
ok_vs_concat = (concat_med - med) >= 0.25
ok_vs_dot = (med - dot_med) <= 0.2
print()
print(f"判据1 max_err==0.0              : {'PASS' if ok_err else 'FAIL'} ({err})")
print(f"判据2 runtime 7.2-7.6ms 带      : {'PASS' if ok_band else 'FAIL'} ({med:.3f}ms)")
print(f"判据3 比 concat 快 >=0.25ms     : {'PASS' if ok_vs_concat else 'FAIL'} (Δ={concat_med - med:.3f}ms)")
print(f"判据4 不比 halves_dot 慢 >0.2ms : {'PASS' if ok_vs_dot else 'FAIL'} (Δ={med - dot_med:+.3f}ms)")
all_ok = ok_err and ok_band and ok_vs_concat and ok_vs_dot
print("OVERALL:", "PASS" if all_ok else "FAIL")
