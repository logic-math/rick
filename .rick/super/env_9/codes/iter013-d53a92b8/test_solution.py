#!/usr/bin/env python3
"""test_solution.py — 迭代 13 TDD 测试（先红后绿）。

针对本次改动（matmul 内核：plain 整阵派发 → 形态③「每次新建 + 预 fault + out=」）：
- test_uses_prefault_out_path 是**区分性测试**：在父本（plain，无 out=）上必红，
  在目标实现上绿——证明测试真的在测本次改动。
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

def test_uses_prefault_out_path():
    """2-D gemm 形态：np.matmul 必须以 out= 派发，且派发时刻 out 缓冲已全 0（预 fault 证据）。"""
    a = np.random.RandomState(42).rand(64, 64).astype(np.float32)
    b = np.random.RandomState(123).rand(64, 64).astype(np.float32)
    calls = []
    orig = np.matmul

    def spy(x1, x2, out=None, **kw):
        calls.append({"has_out": out is not None,
                      "zeroed": bool(out is not None and not out.any()),
                      "shape": None if out is None else tuple(out.shape)})
        return orig(x1, x2, out=out, **kw)

    np.matmul = spy
    try:
        solution.matmul(a, b)
    finally:
        np.matmul = orig
    assert calls, "matmul 未经过 np.matmul 派发"
    assert calls[0]["has_out"], "未使用 out= 派发（父本 plain 形态——本测试应红）"
    assert calls[0]["zeroed"], "派发时刻 out 缓冲非全 0——缺少预 fault（c[:]=0）步骤"
    assert calls[0]["shape"] == (64, 64), "out 形状不符合 (n,k),(k,m)->(n,m) 签名"


# ---------- 契约保持（不变量回归） ----------

def test_bitwise_identity_sq():
    for n in (64, 300, 512):
        for seed in (42, 123):
            rng = np.random.RandomState(seed)
            a = rng.rand(n, n).astype(np.float32)
            b = rng.rand(n, n).astype(np.float32)
            c = solution.matmul(a, b)
            assert _bitwise(c, _ref(a, b)), "n=%d seed=%d 非逐位一致（K 求和序被动）" % (n, seed)


def test_bitwise_identity_rect():
    rng = np.random.RandomState(7)
    a = rng.rand(300, 500).astype(np.float32)
    b = rng.rand(500, 200).astype(np.float32)
    assert _bitwise(solution.matmul(a, b), _ref(a, b))


def test_list_input():
    al = [[1.0, 2.0], [3.0, 4.0]]
    bl = [[5.0, 6.0], [7.0, 8.0]]
    assert _bitwise(solution.matmul(al, bl), _ref(al, bl))


def test_float64_input_coerced():
    rng = np.random.RandomState(5)
    a = rng.rand(64, 64)          # f64
    b = rng.rand(64, 64)          # f64
    assert _bitwise(solution.matmul(a, b), _ref(a, b))


def test_fresh_output_each_call():
    """连续两次调用返回不同对象且值逐位一致——排除共享缓冲（V1 慢模式/覆写风险）。"""
    rng = np.random.RandomState(11)
    a = rng.rand(64, 64).astype(np.float32)
    b = rng.rand(64, 64).astype(np.float32)
    r1 = solution.matmul(a, b)
    v1 = r1.copy()
    r2 = solution.matmul(a, b)
    assert r1 is not r2, "返回了共享缓冲对象（V1 形态——契约风险）"
    assert _bitwise(r1, v1), "第二次调用覆写了第一次的返回内容（缓冲共享）"
    assert _bitwise(r2, v1)


def test_double_run_consistency():
    rng = np.random.RandomState(21)
    a = rng.rand(128, 128).astype(np.float32)
    b = rng.rand(128, 128).astype(np.float32)
    r1 = solution.matmul(a, b).copy()
    r2 = solution.matmul(a, b)
    assert _bitwise(r1, r2), "同输入两遍输出不一致（verify ⑦）"


def test_non_2d_fallback():
    """1-D × 2-D 走泛化兜底，语义与父本一致（np.matmul 规则）。"""
    v = np.ones(4, np.float32)
    m = (np.eye(4, dtype=np.float32) * 3)
    c = solution.matmul(v, m)
    assert np.allclose(c, v * 3) and np.asarray(c).shape == (4,)
    # 列表 1-D 同样兜底
    c2 = solution.matmul([1.0, 2.0], [[1.0, 0.0], [0.0, 1.0]])
    assert np.allclose(c2, [1.0, 2.0])


if __name__ == "__main__":
    fns = [v for k, v in sorted(globals().items()) if k.startswith("test_")]
    failed = 0
    for fn in fns:
        try:
            fn()
            print("PASS %s" % fn.__name__)
        except AssertionError as e:
            failed += 1
            print("FAIL %s: %s" % (fn.__name__, e))
    print("total=%d failed=%d" % (len(fns), failed))
    raise SystemExit(1 if failed else 0)
