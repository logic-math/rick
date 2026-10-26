#!/usr/bin/env python3
"""test_solution.py — 迭代 10 TDD 测试（m_halves + np.dot 派发）。

运行（venv 无 pytest，用 stdlib unittest）：
    env/.venv/bin/python test_solution.py

红→绿判据（coding_plan.md §3）：
- KernelDispatchTests 三例在父本（iter004：np.matmul kernel、无 main()）上必须**红**，
  失败信息须指向「缺 np.dot 派发 / 缺 main 协议」；
- Bitwise / ShapeRobustness / Contract 三例为回归护栏（父本已绿，实现后仍须绿）；
  分级断言：常规块（K>=16）逐位一致（err=0.0），退化小形状 <= 2.4e-07（1 ULP 内核域界，
  iter005 实测记录；门规模判据以 @8192 逐位为准，见实验计划第 5 段）。
"""
import importlib.util
import json
import os
import subprocess
import sys
import unittest

import numpy as np

HERE = os.path.dirname(os.path.abspath(__file__))
SOLUTION = os.path.join(HERE, "solution.py")

# 与门禁（verify.py/benchmark.py）同源的加载方式：importlib 按路径加载。
_spec = importlib.util.spec_from_file_location("iter010_solution", SOLUTION)
sol = importlib.util.module_from_spec(_spec)
_spec.loader.exec_module(sol)

ULP_BOUND = 2.4e-07  # 1 ULP @ |x|~1.0 —— 退化小形状的内核域界（远小于门容差 1e-3）


class KernelDispatchTests(unittest.TestCase):
    """假设主句：M 维二分 + 每半单次 2-D np.dot 整阵派发（C 预分配 + 行切片 out= 直写）。"""

    def test_kernel_dispatches_two_np_dot_calls(self):
        """matmul(a,b) 恰好 2 次 np.dot：入参为 2-D 半块 (mid,K)/(m-mid,K)，
        out= 为预分配 C 的行切片 (mid,N)/(m-mid,N)——奇数行 mid 安全。"""
        m, k, n = 67, 32, 48  # 奇数行：mid=33 / 34，钉住「二分 + 余数并入上半」
        rng = np.random.RandomState(11)
        a = rng.rand(m, k).astype(np.float32)
        b = rng.rand(k, n).astype(np.float32)
        calls = []
        orig_dot = np.dot

        def _spy(*args, **kwargs):
            calls.append((args, kwargs))
            return orig_dot(*args, **kwargs)

        np.dot = _spy
        try:
            c = sol.matmul(a, b)
        finally:
            np.dot = orig_dot

        mid = m // 2
        self.assertEqual(
            len(calls), 2,
            "expected exactly 2 np.dot dispatch calls (m_halves), got %d — "
            "kernel not using the np.dot entry" % len(calls))
        (args0, kw0), (args1, kw1) = calls
        # 两次调用均为 2-D 整块（不是逐行 1-D gemv，也不是单次整阵）
        self.assertEqual(args0[0].ndim, 2, "first np.dot arg must be 2-D")
        self.assertEqual(args1[0].ndim, 2, "second np.dot arg must be 2-D")
        self.assertEqual(args0[0].shape, (mid, k), "lower half shape (mid, K)")
        self.assertEqual(args1[0].shape, (m - mid, k), "upper half shape (m-mid, K)")
        self.assertEqual(args0[1].shape, (k, n), "b passed whole (no N/K split)")
        self.assertEqual(args1[1].shape, (k, n), "b passed whole (no N/K split)")
        # out= 必须是预分配 C 的行切片直写
        self.assertIn("out", kw0, "first call must pass out= (preallocated row slice)")
        self.assertIn("out", kw1, "second call must pass out= (preallocated row slice)")
        self.assertEqual(kw0["out"].shape, (mid, n))
        self.assertEqual(kw1["out"].shape, (m - mid, n))
        self.assertIs(kw0["out"].base, c, "out0 must be a row-slice view of the returned C")
        self.assertIs(kw1["out"].base, c, "out1 must be a row-slice view of the returned C")
        # M 分解不动 K 求和序 → 与 np.matmul 锚逐位一致
        self.assertTrue(np.array_equal(c, np.matmul(a, b)),
                        "m_halves np.dot result not bitwise identical to np.matmul anchor")

    def test_kernel_independent_of_matmul(self):
        """np.dot 派发格：kernel 不依赖 np.matmul 入口（替换为炸弹后仍正确完成）。"""
        rng = np.random.RandomState(23)
        a = rng.rand(66, 32).astype(np.float32)
        b = rng.rand(32, 48).astype(np.float32)
        orig_matmul = np.matmul

        def _bomb(*args, **kwargs):
            raise AssertionError(
                "np.matmul must not be called by the m_halves_np_dot kernel")

        np.matmul = _bomb
        try:
            c = sol.matmul(a, b)
        finally:
            np.matmul = orig_matmul
        ref = orig_matmul(a, b)
        self.assertTrue(np.array_equal(c, ref),
                        "result wrong when np.matmul is unavailable")

    def test_main_json_protocol(self):
        """契约：`solution.py --n 64` 末行输出 JSON（预热 1 次 + 计时 1 次 +
        max_error_vs_matmul），impl 标签为 m_halves_np_dot。"""
        proc = subprocess.run(
            [sys.executable, SOLUTION, "--n", "64"],
            capture_output=True, text=True, timeout=120, cwd=HERE)
        self.assertEqual(proc.returncode, 0,
                         "exit %s; stderr tail: %r" % (proc.returncode, proc.stderr[-400:]))
        lines = [ln.strip() for ln in proc.stdout.splitlines() if ln.strip()]
        self.assertTrue(lines, "no stdout from solution.py --n 64")
        last = lines[-1]
        self.assertTrue(last.startswith("{"),
                        "no JSON line in output (main --n protocol missing?); "
                        "last line: %r" % last[:160])
        payload = json.loads(last)
        self.assertEqual(payload.get("impl"), "m_halves_np_dot",
                         "impl label mismatch: %r" % payload.get("impl"))
        self.assertIs(payload.get("ref"), False)
        self.assertEqual(payload.get("n"), 64)
        self.assertGreater(payload.get("runtime_ms", 0.0), 0.0)
        self.assertEqual(payload.get("max_error_vs_matmul"), 0.0,
                         "not bitwise vs matmul anchor @64: %r"
                         % payload.get("max_error_vs_matmul"))


class BitwiseIdentityTests(unittest.TestCase):
    """假设数值部分：M 分解不动 K 求和序 → 与 np.matmul 锚逐位一致（max_error=0.0）。"""

    def test_bitwise_identical_to_matmul_anchor(self):
        for seed in (42, 123):
            rng = np.random.RandomState(seed)
            a = rng.rand(512, 512).astype(np.float32)
            b = rng.rand(512, 512).astype(np.float32)
            c = sol.matmul(a, b)
            self.assertEqual(c.dtype, np.float32)
            self.assertTrue(np.array_equal(c, np.matmul(a, b)),
                            "seed %d: not bitwise identical to np.matmul anchor" % seed)


class ShapeRobustnessTests(unittest.TestCase):
    """假设边界：mid=m//2 奇数行安全 / 空半合法 / 非方阵 / 嵌套 list 输入。"""

    def test_shape_robustness(self):
        rng = np.random.RandomState(7)
        cases = [
            (128, 128, "square"),
            (129, 64, "odd rows, non-square"),
            (64, 127, "odd cols"),
            (1, 32, "single row"),
            (0, 16, "zero rows"),
            (33, 33, "odd square"),
        ]
        for m_, k_, label in cases:
            a = rng.rand(m_, k_)
            b = rng.rand(k_, k_ + 3)
            c = sol.matmul(a, b)
            ref = np.matmul(np.asarray(a, np.float32), np.asarray(b, np.float32))
            self.assertEqual(c.shape, (m_, k_ + 3), "shape mismatch (%s)" % label)
            if m_ == 0:
                continue
            err = float(np.max(np.abs(
                c.astype(np.float64) - ref.astype(np.float64))))
            tol = 0.0 if k_ >= 16 else ULP_BOUND  # 常规块逐位；退化小形状 1-ULP 界
            self.assertLessEqual(err, tol, "label=%s err=%r > tol=%r" % (label, err, tol))


class ContractTests(unittest.TestCase):
    """契约护栏：嵌套 list 输入归一 / 输出 f32 ndarray。"""

    def test_contract_dtype_and_input(self):
        la = [[1.0, 2.0], [3.0, 4.0], [5.0, 6.0]]
        lb = [[1.0, 0.0, 2.0], [0.0, 1.0, 3.0]]
        c = sol.matmul(la, lb)
        self.assertIsInstance(c, np.ndarray)
        self.assertEqual(c.dtype, np.float32, "output must be float32 (anchor dtype)")
        self.assertEqual(c.shape, (3, 3))
        ref = np.matmul(np.asarray(la, np.float32), np.asarray(lb, np.float32))
        self.assertTrue(np.array_equal(c, ref))
        # f64 输入亦归一为 f32（契约：asarray f32）
        a64 = np.eye(4, dtype=np.float64) * 2.0
        c2 = sol.matmul(a64, a64)
        self.assertEqual(c2.dtype, np.float32)
        self.assertTrue(np.array_equal(c2, np.eye(4, dtype=np.float32) * 4.0))


if __name__ == "__main__":
    unittest.main(verbosity=2)
