#!/usr/bin/env python3
"""pre_bench2.py — iter022 预研 II：r1/r2 邻接零拷贝装配可行性。

假设链（逐项裁决）：
- T1: 预释放 16MB 探针后，两次 np.empty(8MB) 是否落在相邻地址？
- T2: 真实 einsum 调用（内部分配 8MB 输出）在探针后是否相邻？
     （关键：gemm1 内部若有 vecLib 侧 malloc，可能偷走余下半区破坏邻接）
- T3: 无探针稳态下（模拟 bench 循环：结果丢弃、r1/r2 每次释放回收）
     相邻率是否自持？
- T4: 端到端零拷贝变体（邻接→ctypes 视图直返；不邻接→回退逐块拷贝）
     vs 父本 v0 交错配对计时 + 正确性。

构造（邻接成立时）：
    probe = np.empty(2*half_bytes); del probe     # 塑形空闲区间
    r1 = einsum(gemm1)                            # 落在区间前半
    r2 = einsum(gemm2)                            # 落在区间后半（余下精确匹配）
    if r2.data == r1.data + half_bytes:           # 运行时验证，不成立即回退
        buf = (ctypes.c_char * total).from_address(r1.ctypes.data)
        buf._own = (r1, r2)                       # ctypes 对象持引用防 GC
        return np.ndarray(shape, np.float32, buffer=buf)   # 零拷贝直返
    else:
        c = np.empty(...); c[:h] = r1; c[h:] = r2; return c  # 父本路径
"""
import ctypes
import statistics
import time

import numpy as np

np.seterr(divide="ignore", over="ignore", invalid="ignore")

N = 2048
HALF_BYTES = None


def ein_half(b32, a_blk):
    return np.einsum('jk,ij->ik', b32, a_blk, optimize=True)


def t1_raw_adjacency(a32, b32, reps=20):
    """探针后两次裸 np.empty(8MB) 的相邻率。"""
    m = a32.shape[0]
    h = m // 2
    hb = h * a32.shape[1] * 4
    ok = 0
    for _ in range(reps):
        probe = np.empty(2 * hb, dtype=np.uint8)
        del probe
        r1 = np.empty(hb, dtype=np.uint8)
        r2 = np.empty(hb, dtype=np.uint8)
        if r2.ctypes.data == r1.ctypes.data + hb:
            ok += 1
        del r1, r2
    return ok, reps


def t2_einsum_adjacency_probe(a32, b32, reps=20):
    """探针 + 真实两次 einsum（gemm 在两次分配之间跑）的相邻率。"""
    m = a32.shape[0]
    h = m // 2
    hb = h * a32.shape[1] * 4
    ok = 0
    for _ in range(reps):
        probe = np.empty(2 * hb, dtype=np.uint8)
        del probe
        r1 = ein_half(b32, a32[:h])
        r2 = ein_half(b32, a32[h:])
        if r2.ctypes.data == r1.ctypes.data + hb:
            ok += 1
        del r1, r2
    return ok, reps


def t3_einsum_adjacency_steady(a32, b32, reps=20):
    """无探针稳态（结果即刻丢弃）相邻率。"""
    m = a32.shape[0]
    h = m // 2
    hb = h * a32.shape[1] * 4
    ok = 0
    for _ in range(reps):
        r1 = ein_half(b32, a32[:h])
        r2 = ein_half(b32, a32[h:])
        if r2.ctypes.data == r1.ctypes.data + hb:
            ok += 1
        del r1, r2
    return ok, reps


def v0_base(a32, b32):
    m = a32.shape[0]
    h = m // 2
    c = np.empty((m,) + b32.shape[1:], dtype=np.float32)
    r1 = ein_half(b32, a32[:h])
    c[:h] = r1
    r2 = ein_half(b32, a32[h:])
    c[h:] = r2
    return c


def vzc_zero_copy(a32, b32):
    """邻接→零拷贝视图直返；不邻接→回退父本拷贝路径。"""
    m = a32.shape[0]
    h = m // 2
    n = b32.shape[1] if b32.ndim == 2 else b32.shape[0]
    hb = h * n * 4
    total = m * n * 4
    probe = np.empty(total, dtype=np.uint8)   # 塑形空闲区间（等价目标大小）
    del probe
    r1 = ein_half(b32, a32[:h])
    r2 = ein_half(b32, a32[h:])
    if r2.ctypes.data == r1.ctypes.data + hb and r1.ctypes.data % 16 == 0:
        buf = (ctypes.c_char * total).from_address(r1.ctypes.data)
        buf._own = (r1, r2)                   # 防 GC：视图生命周期内钉住两块
        return np.ndarray(shape=(m, n), dtype=np.float32, buffer=buf)
    # 回退：父本逐块拷贝路径
    c = np.empty((m,) + b32.shape[1:], dtype=np.float32)
    c[:h] = r1
    c[h:] = r2
    return c


def main():
    rng = np.random.RandomState(42)
    A = rng.rand(N, N).astype(np.float32)
    B = rng.rand(N, N).astype(np.float32)
    ref = np.matmul(A, B)

    ok, reps = t1_raw_adjacency(A, B)
    print(f"T1 探针后裸 np.empty 相邻: {ok}/{reps}")
    ok, reps = t2_einsum_adjacency_probe(A, B)
    print(f"T2 探针后真实 einsum 相邻: {ok}/{reps}")
    ok, reps = t3_einsum_adjacency_steady(A, B)
    print(f"T3 无探针稳态相邻: {ok}/{reps}")

    # 正确性：零拷贝视图逐位一致 + 返回数组可独立存活（r1/r2 不被提前回收）
    out = vzc_zero_copy(A, B)
    err = float(np.max(np.abs(out.astype(np.float64) - ref.astype(np.float64))))
    print(f"T4a 零拷贝正确性 max_error = {err}")
    assert err == 0.0
    # 视图存活检验：构造后做一轮大量分配，再校验内容不变（_own 钉住生效）
    import gc
    stress = [np.empty((512, 512), dtype=np.float32) for _ in range(64)]
    del stress
    gc.collect()
    err2 = float(np.max(np.abs(out.astype(np.float64) - ref.astype(np.float64))))
    print(f"T4b 视图存活（分配压力后）max_error = {err2}")
    assert err2 == 0.0
    # 双跑一致 + 老结果不被新调用破坏（无跨调用别名）
    o1 = vzc_zero_copy(A, B)
    o2 = vzc_zero_copy(A, B)
    e1 = float(np.max(np.abs(o1.astype(np.float64) - ref.astype(np.float64))))
    e2 = float(np.max(np.abs(o2.astype(np.float64) - ref.astype(np.float64))))
    print(f"T4c 双跑一致 max_error = {e1}/{e2}")
    assert e1 == 0.0 and e2 == 0.0

    # 端到端交错配对
    ROUNDS = 30
    fns = [("v0_base", v0_base), ("vzc_zero_copy", vzc_zero_copy),
           ("ref_matmul", lambda a, b: np.matmul(a, b))]
    for _ in range(5):
        for _, fn in fns:
            fn(A, B)
    times = {n: [] for n, _ in fns}
    for _ in range(ROUNDS):
        for name, fn in fns:
            t0 = time.perf_counter()
            fn(A, B)
            times[name].append((time.perf_counter() - t0) * 1000.0)
    print(f"\n{'variant':<16}{'median':>9}{'mean':>9}{'min':>9}{'p75':>9}")
    for name, _ in fns:
        ts = sorted(times[name])
        print(f"{name:<16}{statistics.median(ts):>9.3f}{statistics.mean(ts):>9.3f}"
              f"{ts[0]:>9.3f}{ts[3*len(ts)//4]:>9.3f}")
    wins = sum(1 for i in range(ROUNDS)
               if times['vzc_zero_copy'][i] < times['v0_base'][i])
    diffs = [times['v0_base'][i] - times['vzc_zero_copy'][i] for i in range(ROUNDS)]
    print(f"\npaired: vzc wins {wins}/{ROUNDS}, Δmedian {statistics.median(diffs):+.3f}ms")
    print(f"ref median {statistics.median(times['ref_matmul']):.3f}ms（负载带 6.6-7.0）")


if __name__ == "__main__":
    main()
