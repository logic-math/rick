"""S1 终验预演：正确性全电池（双 seed、双跑一致、退化形状）+ self_check 口径单次计时。

S1 = einsum('jk,ij->ik', b, a[s:e], optimize=True)（tensordot(A_blk,B)→N-N gemm→C 块）
   + np.concatenate(blocks, axis=0)
"""
import time

import numpy as np

np.seterr(divide="ignore", over="ignore", invalid="ignore")


def s1_matmul(a, b):
    a32 = np.asarray(a, dtype=np.float32)
    b32 = np.asarray(b, dtype=np.float32)
    m = a32.shape[0]
    edges = [(i * m // 4, (i + 1) * m // 4) for i in range(4)]
    blocks = []
    for s, e in edges:
        if e > s:
            blocks.append(np.einsum('jk,ij->ik', b32, a32[s:e], optimize=True))
    if not blocks:
        return np.zeros((0,) + b32.shape[1:], dtype=np.float32)
    return np.concatenate(blocks, axis=0)


# ---- 正确性电池：多形状 × 多 seed × 双跑一致 ----
print("正确性电池 (容差 1e-3，门禁口径 sizes=[64,2048]+双seed+双跑):")
worst = 0.0
ok_all = True
for (m_, k_, n2_) in [(64, 64, 64), (2048, 2048, 2048), (65, 63, 67), (2, 5, 3), (1, 1, 1),
                      (3, 7, 5), (7, 3, 11), (5, 5, 5), (0, 4, 4), (128, 256, 64),
                      (2048, 2048, 64), (64, 2048, 2048)]:
    for seed in (42, 996785307 % (2**31 - 1)):
        rng = np.random.RandomState(seed + m_ + k_)
        aa = rng.rand(m_, k_).astype(np.float32)
        bb = rng.rand(k_, n2_).astype(np.float32)
        got = s1_matmul(aa, bb)
        got2 = s1_matmul(aa, bb)
        exp = np.matmul(aa, bb)
        err = float(np.max(np.abs(got.astype(np.float64) - exp.astype(np.float64)))) if got.size else 0.0
        consistent = np.array_equal(got, got2)
        ok = got.shape == exp.shape and err <= 1e-3 and consistent
        ok_all &= ok
        worst = max(worst, err)
        if not ok or err > 0:
            print("  m=%-5d k=%-5d n=%-5d seed=%-10d err=%.2e consistent=%s ok=%s" % (
                m_, k_, n2_, seed, err, consistent, ok))
print("全部通过:", ok_all, " 最差 err: %.2e (容差 1e-3)" % worst)

# ---- self_check 口径：2048 单次（预热后）×5 ----
rng = np.random.RandomState(42)
a = rng.rand(2048, 2048).astype(np.float32)
b = rng.rand(2048, 2048).astype(np.float32)
s1_matmul(a, b)
ts = []
for _ in range(5):
    t0 = time.perf_counter()
    s1_matmul(a, b)
    ts.append((time.perf_counter() - t0) * 1000)
ts.sort()
ref = np.matmul(a, b)
np.matmul(a, b)
rt = []
for _ in range(5):
    t0 = time.perf_counter(); np.matmul(a, b); rt.append((time.perf_counter() - t0) * 1000)
rt.sort()
print("S1 @2048: median %.3f ms (range %.3f-%.3f) | ref %.3f | score~%.3f" % (
    ts[2], ts[0], ts[-1], rt[2], rt[2] / ts[2]))
print("bitwise vs matmul @2048:", np.array_equal(s1_matmul(a, b), ref))
