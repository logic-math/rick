#!/usr/bin/env python3
"""test_solution.py — iter017 候选单元测试（TDD：先红后绿）。

针对本次改动（kernel：逐行 np.dot + np.stack → 单次整阵 np.dot(a32, b32)）：
- 结构判据：kernel 必须是「单次 np.dot 调用、双 2-D 参数、无 np.stack 装配」
  ——区分改前（父本 rowwise）/改后的核心测试（改前红）。
- 行为判据：M=0 空行合法（父本 np.stack([]) 抛 ValueError——改前红）。
- 数值判据：与 np.matmul(f32,f32) 锚逐位一致（K≥16 逐位；K<16 容 1-ULP 内核域界）。
- 契约判据：嵌套 list 输入 / f64→f32 归一 / 输出 dtype float32 / 双跑确定性。

运行：env/.venv/bin/python test_solution.py -v
"""
import os
import sys
import unittest
from unittest import mock

import numpy as np

sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
import solution  # noqa: E402

ULP1 = 2.4e-07  # float32 1-ULP 界（vecLib 小矩阵内核域效应，iter005/iter010 实测）


def _f32(m, k, seed):
    return np.random.RandomState(seed).rand(m, k).astype(np.float32)


class KernelDispatchTests(unittest.TestCase):
    """结构判据：单次整阵 2-D np.dot 直派（本格定义形态 full_blas_dispatch）。"""

    def test_single_2d_dot_call(self):
        """kernel 恰派发一次 np.dot，且两侧均为整阵 2-D（gemm，非逐行 gemv）。"""
        a = _f32(64, 64, 1)
        b = _f32(64, 64, 2)
        real = np.dot
        seen = []

        def spy(x, y, out=None):
            seen.append((np.ndim(x), np.ndim(y), tuple(np.shape(x)), tuple(np.shape(y))))
            return real(x, y) if out is None else real(x, y, out=out)

        with mock.patch("numpy.dot", side_effect=spy):
            c = solution.matmul(a, b)
        self.assertEqual(len(seen), 1,
                         "kernel 应只派发一次 np.dot（整阵直派），实际 %d 次" % len(seen))
        self.assertEqual(seen[0][:2], (2, 2),
                         "np.dot 须收双 2-D 阵（gemm 派发），实际 ndim=%r" % (seen[0][:2],))
        self.assertEqual(seen[0][2], (64, 64), "首个参数须为整阵 A：%r" % (seen[0][2],))
        self.assertEqual(seen[0][3], (64, 64), "第二参数须为整阵 B：%r" % (seen[0][3],))
        self.assertEqual(c.shape, (64, 64))

    def test_no_stack_assembly(self):
        """kernel 不得用 np.stack 装配逐行结果（父本 10.1ms 装配热点 + 空维脆弱点）。"""
        a = _f32(32, 32, 3)
        b = _f32(32, 32, 4)
        with mock.patch("numpy.stack", wraps=np.stack) as stack_spy:
            solution.matmul(a, b)
        self.assertEqual(
            stack_spy.call_count, 0,
            "kernel 不得用 np.stack 装配（实际调用 %d 次）" % stack_spy.call_count)


class EdgeShapeTests(unittest.TestCase):
    """边界形状：M=0 空行（父本 np.stack([]) 抛 ValueError——改前红点）。"""

    def test_zero_rows(self):
        a = np.zeros((0, 16), dtype=np.float32)
        b = _f32(16, 8, 21)
        try:
            c = solution.matmul(a, b)
        except Exception as e:  # noqa: BLE001 —— 把父本错误转成断言失败（TDD 红的规范形态）
            self.fail("M=0 应合法返回 (0, 8)，实现抛了异常：%r" % (e,))
            return
        self.assertEqual(c.shape, (0, 8))
        self.assertEqual(c.dtype, np.float32)


class NumericalParityTests(unittest.TestCase):
    """与 np.matmul(f32,f32) 锚的一致性（K 求和序不变 ⇒ 逐位一致）。

    用例表对齐 iter010 _self_check（本 env 已实测：K>=16 逐位、K<16 ≤1-ULP）。
    """

    def _parity(self, m, k, n, seed, tol):
        a = _f32(m, k, seed)
        b = _f32(k, n, seed + 1000)
        c = solution.matmul(a, b)
        ref = np.matmul(a, b)
        self.assertEqual(c.shape, (m, n), "形状 (M,N)=(%d,%d)，实际 %r" % (m, n, c.shape))
        self.assertEqual(c.dtype, np.float32, "输出 dtype 须 float32")
        err = (float(np.max(np.abs(c.astype(np.float64) - ref.astype(np.float64))))
               if m and n else 0.0)
        self.assertLessEqual(err, tol, "max_error %.3e > tol %.3e（K=%d）" % (err, tol, k))

    def test_bitwise_square_64(self):
        self._parity(64, 64, 64, 11, 0.0)

    def test_bitwise_square_128(self):
        self._parity(128, 128, 128, 12, 0.0)

    def test_bitwise_odd_rows_nonsquare(self):
        self._parity(129, 128, 131, 13, 0.0)

    def test_bitwise_single_row(self):
        self._parity(1, 32, 35, 14, 0.0)

    def test_bitwise_odd_square(self):
        self._parity(33, 33, 36, 16, 0.0)

    def test_small_k_1ulp(self):
        self._parity(33, 8, 11, 15, ULP1)


class ContractTests(unittest.TestCase):
    """契约不回归：嵌套 list / f64→f32 / 双跑确定性。"""

    def test_nested_lists(self):
        rng = np.random.RandomState(51)
        la = rng.rand(3, 16).tolist()          # 嵌套 list，K=16 → 逐位
        lb = rng.rand(16, 4).tolist()
        c = solution.matmul(la, lb)
        ref = np.matmul(np.asarray(la, np.float32), np.asarray(lb, np.float32))
        self.assertEqual(c.shape, (3, 4))
        self.assertEqual(c.dtype, np.float32)
        self.assertTrue(np.array_equal(c, ref),
                        "嵌套 list 输入须与 matmul(f32) 逐位一致")

    def test_f64_input_converted(self):
        a = np.random.RandomState(61).rand(17, 32)   # f64
        b = np.random.RandomState(62).rand(32, 9)    # f64
        c = solution.matmul(a, b)
        ref = np.matmul(a.astype(np.float32), b.astype(np.float32))
        self.assertEqual(c.dtype, np.float32, "f64 输入须归一到 f32 路径")
        self.assertEqual(c.shape, (17, 9))
        self.assertTrue(np.array_equal(c, ref),
                        "f64 输入转换后须与 matmul(f32) 逐位一致")

    def test_deterministic_double_run(self):
        a = _f32(64, 64, 31)
        b = _f32(64, 64, 32)
        c1 = solution.matmul(a, b)
        c2 = solution.matmul(a, b)
        self.assertTrue(np.array_equal(c1, c2), "同输入两次调用须逐位一致")


if __name__ == "__main__":
    unittest.main(verbosity=2)
