#!/usr/bin/env python3
"""test_solution.py — 迭代 14 TDD 测试（先红后绿）。

针对本次改动（matmul 内核输出缓冲管理：每次 fresh 分配 → per-shape 持久预 fault 缓冲 + out= 复用）：
- test_persistent_buffer_reuse / test_reuse_zero_alloc_second_call 是**区分性测试**：
  在父本（iter003 plain，每次 fresh 返回新数组）上必红，在目标实现上绿——
  证明测试真的在测本次改动（持久缓冲结构本身）。
- 其余为契约保持与泛化边界（父本上即绿，属不变量回归）。

运行：/Users/logic/workdir/rick-dev/.rick/super/env_9/env/.venv/bin/python -m pytest test_solution.py -q
（或直接 python test_solution.py）
"""
import numpy as np
import solution

np.seterr(divide="ignore", over="ignore", invalid="ignore")


def _ref(a, b):
    a32 = np.asarray(a, dtype=np.float32)
    b32 = np.asarray(b, dtype=np.float32)
    return np.matmul(a32, b32)


def _bitwise(x, y):
    return np.array_equal(np.asarray(x).view(np.uint32), np.asarray(y).view(np.uint32))


# ---------- 区分性测试（父本红 / 目标绿） ----------

def test_persistent_buffer_reuse():
    """同形状两次调用必须共享同一底层缓冲（持久复用）——父本 fresh 分配必红。"""
    rng = np.random.RandomState(42)
    a = rng.rand(64, 64).astype(np.float32)
    b = rng.rand(64, 64).astype(np.float32)
    c1 = solution.matmul(a, b)
    c2 = solution.matmul(a, b)
    assert np.shares_memory(c1, c2), (
        "两次调用未共享输出缓冲（父本 plain 形态 fresh 分配——本测试应红）")


def test_reuse_zero_alloc_second_call():
    """第二次同形状调用零新分配：派发时 out 即为上一调用返回的同一缓冲对象。"""
    rng = np.random.RandomState(42)
    a = rng.rand(64, 64).astype(np.float32)
    b = rng.rand(64, 64).astype(np.float32)
    c1 = solution.matmul(a, b)
    seen_outs = []
    orig = np.matmul

    def spy(x1, x2, out=None, **kw):
        seen_outs.append(None if out is None else (id(out), out.shape))
        return orig(x1, x2, out=out, **kw)

    np.matmul = spy
    try:
        c2 = solution.matmul(a, b)
    finally:
        np.matmul = orig
    assert seen_outs, "2-D gemm 未经过 np.matmul 派发"
    out_id, out_shape = seen_outs[0]
    assert out_id == id(c1), "复用派发的 out 不是上一调用的缓冲（非持久形态）"
    assert out_shape == (64, 64), "out 形状不符合 (n,k),(k,m)->(n,m) 签名"
    assert np.shares_memory(c2, c1)


# ---------- 契约保持（不变量回归） ----------

def test_contract_list_input():
    a = [[1.0, 2.0], [3.0, 4.0]]
    b = [[5.0, 6.0], [7.0, 8.0]]
    got = solution.matmul(a, b)
    ref = _ref(a, b)
    assert isinstance(got, np.ndarray)
    assert got.dtype == np.float32
    assert got.shape == (2, 2)
    assert _bitwise(got, ref)


def test_bitwise_vs_blas_anchor():
    """K 求和序不变：双规模双 seed 与 f32 BLAS 锚逐位一致（正确性核心）。"""
    for n in (64, 300, 512):
        for seed in (42, 123):
            rng = np.random.RandomState(seed)
            a = rng.rand(n, n).astype(np.float32)
            b = rng.rand(n, n).astype(np.float32)
            got = solution.matmul(a, b)
            assert _bitwise(got, _ref(a, b)), "非逐位一致 @ n=%d seed=%d" % (n, seed)


def test_recompute_not_cached():
    """每次全量重算：换输入后新调用结果=新锚；旧引用内容同步被覆写（共享缓冲语义）。"""
    rng = np.random.RandomState(7)
    a = rng.rand(64, 64).astype(np.float32)
    b1 = rng.rand(64, 64).astype(np.float32)
    b2 = rng.rand(64, 64).astype(np.float32)
    c1 = solution.matmul(a, b1)
    assert _bitwise(c1, _ref(a, b1))
    c2 = solution.matmul(a, b2)
    assert _bitwise(c2, _ref(a, b2)), "返回缓存旧结果（红线：预存结果）"
    assert _bitwise(c1, _ref(a, b2)), "旧引用未被新结果覆写（与共享缓冲语义矛盾）"


def test_shapes_independent():
    """per-shape 键控：不同形状各自独立缓冲，互不别名，各自正确。"""
    rng = np.random.RandomState(42)
    a64 = rng.rand(64, 64).astype(np.float32)
    a512 = rng.rand(512, 512).astype(np.float32)
    b64 = rng.rand(64, 64).astype(np.float32)
    b512 = rng.rand(512, 512).astype(np.float32)
    c64 = solution.matmul(a64, b64)
    c512 = solution.matmul(a512, b512)
    assert not np.shares_memory(c64, c512)
    assert _bitwise(c64, _ref(a64, b64))
    assert _bitwise(c512, _ref(a512, b512))


def test_rectangular_shapes():
    """矩形（n,k),(k,m) 签名：out 形状精确构造（numpy 手册约束）。"""
    rng = np.random.RandomState(5)
    a = rng.rand(37, 53).astype(np.float32)
    b = rng.rand(53, 29).astype(np.float32)
    got = solution.matmul(a, b)
    assert got.shape == (37, 29)
    assert _bitwise(got, _ref(a, b))


def test_non_2d_fallback():
    """1-D×2-D（matvec）走泛化兜底，结果与 np.matmul 一致。"""
    rng = np.random.RandomState(11)
    v = rng.rand(16).astype(np.float32)
    m = rng.rand(16, 8).astype(np.float32)
    got = solution.matmul(v, m)
    ref = np.matmul(np.asarray(v, np.float32), np.asarray(m, np.float32))
    assert _bitwise(np.asarray(got), ref)


def test_double_run_bitwise():
    """镜像 verify 加固⑦：同输入两遍逐位一致（抓缓存/分支特判）。"""
    rng = np.random.RandomState(99)
    a = rng.rand(128, 128).astype(np.float32)
    b = rng.rand(128, 128).astype(np.float32)
    g1 = solution.matmul(a, b)
    g2 = solution.matmul(a, b)
    assert _bitwise(g1, g2)


if __name__ == "__main__":
    fns = [v for k, v in sorted(globals().items()) if k.startswith("test_") and callable(v)]
    failed = 0
    for fn in fns:
        try:
            fn()
            print("PASS %s" % fn.__name__)
        except AssertionError as e:
            failed += 1
            print("FAIL %s: %s" % (fn.__name__, e))
    print("%d/%d passed" % (len(fns) - failed, len(fns)))
    raise SystemExit(1 if failed else 0)
