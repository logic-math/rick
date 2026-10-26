#!/usr/bin/env python3
"""test_solution.py — iter016 TDD 测试（先红后绿）。

被测候选：同目录 solution.py（iter16：m_halves 两半 gemm 由顺序改并发）。

覆盖（对齐实验计划 §4 步骤 2 与 verify ⑤⑥⑦）：
1. bitwise == np.matmul 锚 @ 64/512/8192 × 双 seed（42 / 42+17）。
2. 奇数行 / 奇列 / 非方阵 / M=1 / M=0 / 嵌套 list 形状鲁棒性（契约逐字保留）。
3. 同输入双调用逐位一致（verify ⑦：抓缓存返回 / 条件分支特判）。
4. **判别测试**：两次半阵 gemm 必须在主线程之外的 2 个 worker 线程中执行——
   父本（顺序两半）上必红，候选（线程并发）上绿，确保测试真的在测本次改动。

运行：python3 test_solution.py（纯 Python runner，无 pytest 依赖）。
"""
import importlib.util
import os
import threading

import numpy as np

# macOS vecLib BLAS 虚假 FP 标志噪声抑制（与 solution.py 同款；数值正常）。
np.seterr(divide="ignore", over="ignore", invalid="ignore")

HERE = os.path.dirname(os.path.abspath(__file__))
CANDIDATE = os.path.join(HERE, "solution.py")


def _load_solution():
    spec = importlib.util.spec_from_file_location("solution_under_test", CANDIDATE)
    mod = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(mod)
    return mod


sol = _load_solution()


def test_bitwise_vs_np_matmul_anchor():
    """候选输出必须逐位 == np.matmul(f32 锚)，双规模双 seed。"""
    for n in (64, 512, 8192):
        for seed in (42, 42 + 17):
            rng = np.random.RandomState(seed)
            a = rng.rand(n, n).astype(np.float32)
            b = rng.rand(n, n).astype(np.float32)
            c = sol.matmul(a, b)
            assert isinstance(c, np.ndarray), f"not ndarray @n={n}"
            assert c.dtype == np.float32, f"dtype drifted: {c.dtype} @n={n}"
            ref = np.matmul(a, b)
            assert np.array_equal(c, ref), (
                f"not bitwise identical @n={n}, seed={seed}: "
                f"max_err={float(np.max(np.abs(c.astype(np.float64) - ref.astype(np.float64))))}"
            )


def test_shape_robustness():
    """奇数行/奇列/非方阵/M=1/M=0/嵌套 list 全部 bitwise == 锚且形状正确。"""
    rng = np.random.RandomState(7)
    for m, k in [(128, 128), (129, 64), (64, 127), (1, 32), (0, 16), (33, 33)]:
        a = rng.rand(m, k)
        b = rng.rand(k, k + 3)
        c = sol.matmul(a, b)
        ref = np.matmul(np.asarray(a, np.float32), np.asarray(b, np.float32))
        assert c.shape == (m, k + 3), f"shape mismatch @({m},{k}): {c.shape}"
        if m:
            assert np.array_equal(c, ref), f"not bitwise @({m},{k})"
    # 嵌套 list 输入
    la = [[1.0, 2.0], [3.0, 4.0], [5.0, 6.0]]
    lb = [[1.0, 0.0, 2.0], [0.0, 1.0, 3.0]]
    c = sol.matmul(la, lb)
    ref = np.matmul(np.asarray(la, np.float32), np.asarray(lb, np.float32))
    assert c.shape == (3, 3)
    assert np.array_equal(c, ref), "nested-list input not bitwise identical"


def test_double_call_bitwise_consistent():
    """同输入两次调用输出逐位一致（线程调度不得引入任何非确定性）。"""
    rng = np.random.RandomState(123)
    a = rng.rand(512, 512).astype(np.float32)
    b = rng.rand(512, 512).astype(np.float32)
    c1 = sol.matmul(a, b)
    c2 = sol.matmul(a, b)
    assert np.array_equal(c1, c2), "two calls on same input differ bitwise"


def test_halves_run_concurrently_off_main_thread():
    """判别测试：两次半阵 gemm 必须在主线程之外的 2 个 worker 线程执行。

    spy 包裹 np.matmul 记录每次派发的调用线程 ident：
    - 父本（顺序两半）：两次调用都在主线程 → caller_idents == {main} → 红。
    - 候选（线程并发）：两次调用分属 2 个 worker 线程 ident（≠ 主线程）→ 绿。
    """
    rng = np.random.RandomState(42)
    n = 512
    a = rng.rand(n, n).astype(np.float32)
    b = rng.rand(n, n).astype(np.float32)
    caller_idents = set()
    orig_matmul = np.matmul

    def _spy(x, y, out=None):
        caller_idents.add(threading.get_ident())
        return orig_matmul(x, y, out=out)

    np.matmul = _spy
    try:
        c = sol.matmul(a, b)
    finally:
        np.matmul = orig_matmul
    # spy 注入不破坏正确性
    assert np.array_equal(c, orig_matmul(a, b)), "spy run broke correctness"
    main_ident = threading.get_ident()
    assert len(caller_idents) >= 2, (
        f"expected 2 half-gemm dispatches from >=2 threads, got callers={sorted(caller_idents)}"
    )
    worker_idents = caller_idents - {main_ident}
    assert len(worker_idents) == 2, (
        f"both half-gemm must run off the main thread in 2 worker threads; "
        f"callers={sorted(caller_idents)}, main={main_ident}"
    )


def main():
    tests = [
        test_bitwise_vs_np_matmul_anchor,
        test_shape_robustness,
        test_double_call_bitwise_consistent,
        test_halves_run_concurrently_off_main_thread,
    ]
    failed = []
    for t in tests:
        try:
            t()
            print("PASS", t.__name__, flush=True)
        except AssertionError as exc:
            failed.append(t.__name__)
            print("FAIL", t.__name__, "->", exc, flush=True)
    if failed:
        raise SystemExit(f"RED — failing tests: {failed}")
    print("ALL GREEN")


if __name__ == "__main__":
    main()
