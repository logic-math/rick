#!/usr/bin/env python3
"""test_solution.py — 迭代 7 TDD 测试（dispatch_target=np_dot|full_blas_dispatch）。

判别维度：本格定义形态要求 kernel 入口为 **np.dot**（2-D 整阵 BLAS gemm 直派）。
当前工作目录 solution.py 为 iter003 模板（np.matmul 入口）——
T6（调用路径 spy）与 T7（main JSON impl 标签）必须**先红**，改完 kernel 后转绿；
T1-T5 守契约与数值（逐位一致于 np.matmul 锚），改前改后都应绿（防回归）。

运行（env 三要素同源解释器）：
    env/.venv/bin/python test_solution.py
"""
import importlib.util
import json
import os
import subprocess
import sys
import unittest
from unittest import mock

import numpy as np

HERE = os.path.dirname(os.path.abspath(__file__))
SOLUTION = os.path.join(HERE, "solution.py")


def _load_solution():
    spec = importlib.util.spec_from_file_location("solution_under_test", SOLUTION)
    mod = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(mod)
    return mod


class ContractTests(unittest.TestCase):
    """T1-T2：契约保持（接口 / asarray→f32 / 返回形态）。"""

    def setUp(self):
        self.mod = _load_solution()

    def test_T1_nested_list_input(self):
        a = [[1.0, 2.0], [3.0, 4.0]]
        b = [[5.0, 6.0], [7.0, 8.0]]
        c = self.mod.matmul(a, b)
        self.assertIsInstance(c, np.ndarray)
        self.assertEqual(c.dtype, np.float32)
        self.assertEqual(c.shape, (2, 2))
        self.assertTrue(np.array_equal(
            c, np.array([[19.0, 22.0], [43.0, 50.0]], dtype=np.float32)))

    def test_T2_dtype_normalization(self):
        rng = np.random.RandomState(0)
        a64 = rng.rand(33, 17)          # float64
        b_int = (rng.rand(17, 9) * 10).astype(np.int64)  # int64
        c = self.mod.matmul(a64, b_int)
        self.assertEqual(c.dtype, np.float32)
        ref = np.matmul(np.asarray(a64, dtype=np.float32),
                        np.asarray(b_int, dtype=np.float32))
        self.assertTrue(np.array_equal(c, ref))


class NumericsTests(unittest.TestCase):
    """T3-T5：数值假设（逐位一致于 np.matmul 锚 / 通用形状 / 双跑一致）。"""

    def setUp(self):
        self.mod = _load_solution()

    def test_T3_bitwise_identical_to_matmul_anchor(self):
        for n in (64, 129, 256):
            for seed in (42, 7):
                rng = np.random.RandomState(seed)
                a = rng.rand(n, n).astype(np.float32)
                b = rng.rand(n, n).astype(np.float32)
                c = self.mod.matmul(a, b)
                ref = np.matmul(a, b)  # 锚：K 求和序同源
                max_err = float(np.max(np.abs(
                    c.astype(np.float64) - ref.astype(np.float64))))
                self.assertEqual(max_err, 0.0,
                                 "n=%d seed=%d max_err=%r" % (n, seed, max_err))

    def test_T4_non_square_shapes(self):
        rng = np.random.RandomState(5)
        a = rng.rand(7, 5).astype(np.float32)
        b = rng.rand(5, 11).astype(np.float32)
        c = self.mod.matmul(a, b)
        self.assertEqual(c.shape, (7, 11))
        self.assertTrue(np.array_equal(c, np.matmul(a, b)))

    def test_T5_double_run_consistency(self):
        rng = np.random.RandomState(11)
        a = rng.rand(64, 64).astype(np.float32)
        b = rng.rand(64, 64).astype(np.float32)
        self.assertTrue(np.array_equal(self.mod.matmul(a, b),
                                       self.mod.matmul(a, b)))


class DispatchPathTests(unittest.TestCase):
    """T6：本格定义形态——kernel 恰好一次 np.dot 整阵 2-D 直派、不落 np.matmul。"""

    def setUp(self):
        self.mod = _load_solution()

    def test_T6_kernel_uses_np_dot_entry(self):
        rng = np.random.RandomState(42)
        a = rng.rand(64, 64).astype(np.float32)
        b = rng.rand(64, 64).astype(np.float32)
        with mock.patch("numpy.dot", wraps=np.dot) as dot_spy, \
                mock.patch("numpy.matmul", wraps=np.matmul) as mm_spy:
            c = self.mod.matmul(a, b)
        # 判别断言 1：整阵派发恰好调用一次 np.dot
        self.assertEqual(dot_spy.call_count, 1,
                         "np.dot call_count=%d（应为 1：单次整阵 2-D 直派）"
                         % dot_spy.call_count)
        # 判别断言 2：kernel 不得走 np.matmul 入口
        self.assertEqual(mm_spy.call_count, 0,
                         "np.matmul call_count=%d（应为 0：本格入口为 np.dot）"
                         % mm_spy.call_count)
        # 直派参数形态：两个 2-D float32 阵
        args, kwargs = dot_spy.call_args
        self.assertEqual(len(args), 2)
        for x in args:
            self.assertEqual(x.ndim, 2)
            self.assertEqual(x.dtype, np.float32)
        # 结果仍逐位一致
        self.assertTrue(np.array_equal(c, np.matmul(a, b)))


class MainProtocolTests(unittest.TestCase):
    """T7：main 自检 JSON 协议——impl 标签为本格形态 full_blas_dispatch_np_dot。"""

    def test_T7_impl_label_and_selfcheck_json(self):
        proc = subprocess.run(
            [sys.executable, SOLUTION, "--n", "64"],
            capture_output=True, text=True, timeout=60)
        self.assertEqual(proc.returncode, 0, proc.stderr)
        payload = json.loads(proc.stdout.strip().splitlines()[-1])
        self.assertEqual(payload["impl"], "full_blas_dispatch_np_dot")
        self.assertEqual(payload["n"], 64)
        self.assertEqual(payload["max_error_vs_matmul"], 0.0)
        self.assertGreater(payload["runtime_ms"], 0.0)
        self.assertEqual(payload["ref"], False)


if __name__ == "__main__":
    unittest.main(verbosity=2)
