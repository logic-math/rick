"""probe_e — 验证 einsum_path×m_quarters 修复设计：

np.einsum('ij,jk->ki', a_blk, b, optimize=True)：
  输出下标 'ki' == einsum 内部 tensordot 交换序的 tensor_result → 跳过转置视图，
  直接返回 (n, q) C 连续的转置积（bitwise 已由 probe_b 验证 einsum 家族逐位一致）。
然后写 outT[:, lo:hi]（内轴双侧 unit-stride → 8KB 连续 memcpy 快路径），返回 outT.T。

对照臂：
  A: outT 预分配 + 切片写 + return outT.T（主设计）
  B: np.concatenate(blocks, axis=1) + .T
  P: 父本 tensor_copy(4)（参照，~472-482ms）
  全阵 einsum 'ki' 单发 + .T（参照——full dispatch 形态）
逐块分解计时 + bitwise 一致性（8192 全阵 + 64 小阵 + m<4 边界）。
"""
import time
import numpy as np

np.seterr(divide="ignore", over="ignore", invalid="ignore")

n = 8192
rng = np.random.RandomState(0)
a = rng.rand(n, n).astype(np.float32)
b = rng.rand(n, n).astype(np.float32)
ref = np.matmul(a, b)
q = n // 4


def form_A():
    """outT 预分配 + einsum('ki') 块写入 + return outT.T"""
    outT = np.empty((n, n), dtype=np.float32)
    for i in range(4):
        lo, hi = i * q, (i + 1) * q
        outT[:, lo:hi] = np.einsum('ij,jk->ki', a[lo:hi], b, optimize=True)
    return outT.T


def form_B():
    """concatenate 装配 + .T"""
    blocks = [np.einsum('ij,jk->ki', a[i * q:(i + 1) * q], b, optimize=True)
              for i in range(4)]
    return np.concatenate(blocks, axis=1).T


def form_P():
    """父本参照：tensordot(a_blk, b) + C 布局切片写"""
    out = np.empty((n, n), dtype=np.float32)
    for i in range(4):
        lo, hi = i * q, (i + 1) * q
        out[lo:hi] = np.tensordot(a[lo:hi], b, axes=((1,), (0,)))
    return out


def form_full_ki():
    """全阵 einsum('ki') 单发 + .T（full dispatch 参照）"""
    return np.einsum('ij,jk->ki', a, b, optimize=True).T


# ---- bitwise / 属性检查 ----
for name, fn in [("form_A", form_A), ("form_B", form_B),
                 ("form_P", form_P), ("full_ki", form_full_ki)]:
    r = fn()
    eq = np.array_equal(r, ref)
    mx = float(np.max(np.abs(r.astype(np.float64) - ref.astype(np.float64))))
    print(f"bitwise {name}: {eq} max_err={mx:.3g} "
          f"shape={r.shape} strides={r.strides} C={r.flags['C_CONTIGUOUS']}")

# ---- 逐块分解（form_A）----
print("--- decompose form_A (einsum 'ki' call vs outT slice copy) ---")
for r in range(3):
    outT = np.empty((n, n), dtype=np.float32)
    times = []
    for i in range(4):
        lo, hi = i * q, (i + 1) * q
        t0 = time.perf_counter()
        tmp = np.einsum('ij,jk->ki', a[lo:hi], b, optimize=True)
        t1 = time.perf_counter()
        outT[:, lo:hi] = tmp
        t2 = time.perf_counter()
        times.append(((t1 - t0) * 1000, (t2 - t1) * 1000))
    if r == 2:
        for i, (c, cp) in enumerate(times):
            print(f"  blk{i}: einsum_ki={c:6.1f}ms copyT={cp:5.1f}ms")
        print(f"  4blk total={sum(c + cp for c, cp in times):6.1f}ms")

# ---- 交错计时 ----
ARMS = [("form_A", form_A), ("form_B", form_B),
        ("form_P", form_P), ("full_ki", form_full_ki),
        ("full_matmul", lambda: np.matmul(a, b))]
for _ in range(2):
    for _, fn in ARMS:
        fn()
times = {name: [] for name, _ in ARMS}
for _ in range(6):
    for name, fn in ARMS:
        t0 = time.perf_counter()
        fn()
        times[name].append((time.perf_counter() - t0) * 1000.0)
for name, _ in ARMS:
    ts = sorted(times[name])
    print(f"{name:12s}: min={ts[0]:7.1f} med={ts[len(ts) // 2]:7.1f} "
          f"all={[round(t, 1) for t in ts]}")

# ---- 小规模 / 边界 bitwise ----
print("--- small/edge bitwise ---")
for (m, kk, nn) in [(64, 64, 64), (2, 16, 16), (1, 8, 8), (3, 5, 7),
                    (5, 33, 17), (100, 64, 32)]:
    rng2 = np.random.RandomState(m * 1000 + kk)
    aa = rng2.rand(m, kk).astype(np.float32)
    bb = rng2.rand(kk, nn).astype(np.float32)
    rref = np.matmul(aa, bb)
    # quarters 形态（q=ceil(m/4)）
    qq = -(-m // 4)
    oT = np.empty((nn, m), dtype=np.float32)
    for i in range(4):
        lo, hi = min(i * qq, m), min((i + 1) * qq, m)
        if lo < hi:
            oT[:, lo:hi] = np.einsum('ij,jk->ki', aa[lo:hi], bb, optimize=True)
    rA = oT.T
    # 全阵 einsum 'ki'
    rF = np.einsum('ij,jk->ki', aa, bb, optimize=True).T
    print(f"m={m:3d} k={kk:3d} n={nn:3d}: quarters bitwise={np.array_equal(rA, rref)} "
          f"full_ki bitwise={np.array_equal(rF, rref)} "
          f"err_q={float(np.max(np.abs(rA.astype(np.float64) - rref.astype(np.float64)))):.3g}")
