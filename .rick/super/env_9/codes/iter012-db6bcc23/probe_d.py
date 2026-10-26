"""probe_d — 分解 einsum_noout(4) 的 653ms 到底慢在哪。

probe_c 证据：einsum_1blk=127.1ms / tensordot_1blk=120.1ms（微循环，同一块）；
cProfile：einsum 子树 128.5ms/调用（路径机制≈0），帧自身 tottime 216ms
→ 疑点在切片赋值 out[...] = einsum_result。
本探针逐相位计时（einsum 调用 vs 拷贝 vs 释放），并对照 tensordot 版本。
"""
import time
import numpy as np

np.seterr(divide="ignore", over="ignore", invalid="ignore")

n = 8192
rng = np.random.RandomState(0)
a = rng.rand(n, n).astype(np.float32)
b = rng.rand(n, n).astype(np.float32)
q = n // 4


def decompose(fn_name, call_fn, rounds=4):
    """逐块计时：call（收缩）与 copy（写 out）分离。"""
    print(f"--- decompose {fn_name} ---")
    all_t = []
    for r in range(rounds):
        out = np.empty((n, n), np.float32)
        t_alloc = time.perf_counter()
        times = []
        for i in range(4):
            lo, hi = i * q, (i + 1) * q
            t0 = time.perf_counter()
            tmp = call_fn(a[lo:hi], b)
            t1 = time.perf_counter()
            out[lo:hi] = tmp
            t2 = time.perf_counter()
            times.append(((t1 - t0) * 1000, (t2 - t1) * 1000))
        t_end = time.perf_counter()
        all_t.append(times)
        if r == rounds - 1:
            for i, (c, cp) in enumerate(times):
                print(f"  blk{i}: call={c:7.1f}ms copy={cp:6.1f}ms")
            print(f"  total={sum(c+cp for c,cp in times):7.1f}ms "
                  f"(+alloc {(t_alloc and 0) or 0})")
    # 汇总（每轮 4 块）
    calls = [c for r in all_t for c, _ in r]
    copies = [cp for r in all_t for _, cp in r]
    print(f"  summary: call min={min(calls):.1f} med={sorted(calls)[len(calls)//2]:.1f} | "
          f"copy min={min(copies):.1f} med={sorted(copies)[len(copies)//2]:.1f} | "
          f"sum med={sorted(c+cp for r in all_t for c,cp in r)[4]:.1f}ms/round")


# 检查 einsum 结果的内存属性
r_ein = np.einsum('ij,jk->ik', a[q:2 * q], b, optimize=True)
r_td = np.tensordot(a[q:2 * q], b, axes=((1,), (0,)))
print("einsum result:", type(r_ein).__name__, r_ein.dtype, "strides:", r_ein.strides,
      "C:", r_ein.flags['C_CONTIGUOUS'], "OWNS:", r_ein.flags['OWNDATA'])
print("tensordot result:", type(r_td).__name__, r_td.dtype, "strides:", r_td.strides,
      "C:", r_td.flags['C_CONTIGUOUS'], "OWNS:", r_td.flags['OWNDATA'])
print("array_equal:", np.array_equal(r_ein, r_td))

# 纯拷贝微基准（同一 src 多次拷到不同 out 的同一切片）
out = np.empty((n, n), np.float32)
for name, src in [("einsum_res ", r_ein), ("tdot_res   ", r_td)]:
    ts = []
    for _ in range(5):
        t0 = time.perf_counter()
        out[q:2 * q] = src
        ts.append((time.perf_counter() - t0) * 1000.0)
    print(f"copy {name}: {[round(t, 1) for t in ts]}")

# 逐相位分解（先预热一轮）
decompose("einsum(opt=T)", lambda x, y: np.einsum('ij,jk->ik', x, y, optimize=True))
decompose("tensordot    ", lambda x, y: np.tensordot(x, y, axes=((1,), (0,))))

# 变体：np.copyto / 拷到 out 的独立切片目标 / einsum 结果先转 np.array (no-op)?
def decompose_copyto(rounds=3):
    print("--- decompose einsum + copyto ---")
    for r in range(rounds):
        out = np.empty((n, n), np.float32)
        times = []
        for i in range(4):
            lo, hi = i * q, (i + 1) * q
            t0 = time.perf_counter()
            tmp = np.einsum('ij,jk->ik', a[lo:hi], b, optimize=True)
            t1 = time.perf_counter()
            np.copyto(out[lo:hi], tmp)
            t2 = time.perf_counter()
            times.append(((t1 - t0) * 1000, (t2 - t1) * 1000))
        if r == rounds - 1:
            for i, (c, cp) in enumerate(times):
                print(f"  blk{i}: call={c:7.1f}ms copyto={cp:6.1f}ms")
            print(f"  total={sum(c+cp for c, cp in times):7.1f}ms")

decompose_copyto()
