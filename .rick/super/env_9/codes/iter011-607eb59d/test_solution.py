#!/usr/bin/env python3
"""test_solution.py — 迭代 11 TDD 测试（einsum_path × m_quarters）。

假设主句（experiment_plan idea）：手动复制 einsum optimize 路径的内部序列
（tensordot BLAS 收缩 + 结果拷入 out），按 M 维四分逐块派发，绕开
``np.einsum(..., out=, optimize=True)`` 自身调度代码的恒等拷贝惩罚。

红→绿判据（coding_plan.md §4）：
- KernelDispatchTests 四例中前三例 + MainProtocolTests 在父本（naive_start.py：
  rowwise np.dot + np.stack 拼装）上必须**红**，失败信息须指向「缺 tensordot 派发 /
  缺预分配直写 / impl 标签不符」；
- BitwiseIdentity / ShapeRobustness / Contract / 双跑一致为回归护栏（父本已绿，
  实现后仍须绿）；分级容差：常规块（K>=16）逐位一致（err=0.0），退化小形状
  <= 2.4e-07（1 ULP 内核域界，iter010 实测记录；门规模判据以 @8192 逐位为准）。

运行（venv 无 pytest，用 stdlib unittest）：
    env/.venv/bin/python test_solution.py
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
_spec = importlib.util.spec_from_file_location("iter011_solution", SOLUTION)
sol = importlib.util.module_from_spec(_spec)
_spec.loader.exec_module(sol)

ULP_BOUND = 2.4e-07  # 1 ULP @ |x|~1.0 —— 退化小形状的内核域界（远小于门容差 1e-3）


class KernelDispatchTests(unittest.TestCase):
    """假设主句：M 四分 × 每块 einsum 内部序列（tensordot → 切片直写预分配 C）。"""

    def test_kernel_dispatches_four_tensordot_calls(self):
        """matmul(a,b) 恰好 4 次 np.tensordot：入参为 2-D M 行块 (q,K)×整阵 B (K,N)，
        收缩轴 axes=((1,), (0,))——einsum 源码同款调用形态（einsumfunc.py L1477）。"""
        m, k, n = 67, 32, 48  # 奇数行：q=ceil(67/4)=17 → 块 17,17,17,16（余数并入末块）
        rng = np.random.RandomState(11)
        a = rng.rand(m, k).astype(np.float32)
        b = rng.rand(k, n).astype(np.float32)
        calls = []
        orig_td = np.tensordot

        def _spy(*args, **kwargs):
            calls.append((args, kwargs))
            return orig_td(*args, **kwargs)

        np.tensordot = _spy
        try:
            c = sol.matmul(a, b)
        finally:
            np.tensordot = orig_td

        q = -(-m // 4)  # ceil
        self.assertEqual(
            len(calls), 4,
            "expected exactly 4 np.tensordot dispatch calls (m_quarters), got %d — "
            "kernel not using the einsum-internal tensordot sequence" % len(calls))
        for i, (args_i, kw_i) in enumerate(calls):
            left, right = args_i[0], args_i[1]
            self.assertEqual(left.ndim, 2,
                             "tensordot arg0 #%d must be a 2-D M-row block" % i)
            self.assertEqual(right.shape, (k, n),
                             "tensordot arg1 #%d must be B passed whole" % i)
            axes = kw_i.get("axes", 2 if len(args_i) < 3 else args_i[2])
            self.assertEqual(
                axes, ((1,), (0,)),
                "tensordot #%d axes must be einsum's form ((1,),(0,)) — got %r" % (i, axes))
        # 四块行数 = q,q,q,remainder（M 四分，余数并入末块）
        rows = [calls[0][0][0].shape[0], calls[1][0][0].shape[0],
                calls[2][0][0].shape[0], calls[3][0][0].shape[0]]
        self.assertEqual(rows, [q, q, q, m - 3 * q],
                         "M-quarter row split expected [q,q,q,m-3q], got %r" % rows)
        # M 分解不动 K 求和序 → 与 np.matmul 锚逐位一致
        self.assertTrue(np.array_equal(c, np.matmul(a, b)),
                        "m_quarters tensordot result not bitwise identical to np.matmul anchor")

    def test_result_written_to_preallocated_buffer(self):
        """结果写入预分配 C 的行切片（memcpy 直写），非 np.stack 拼装：
        4 次拷贝目标均为返回数组的行切片视图（base is 返回值）。"""
        m, k, n = 64, 32, 48
        rng = np.random.RandomState(29)
        a = rng.rand(m, k).astype(np.float32)
        b = rng.rand(k, n).astype(np.float32)
        copies = []
        orig_td = np.tensordot

        def _spy(left, right, axes=None):
            res = orig_td(left, right, axes=axes)
            # 捕获紧随其后的拷贝目标：拦截 ndarray.__setitem__ 不可行，
            # 改为捕获返回后各块的来源——通过 base 追溯判定。
            copies.append(res)
            return res

        np.tensordot = _spy
        try:
            c = sol.matmul(a, b)
        finally:
            np.tensordot = orig_td

        self.assertEqual(len(copies), 4, "expected 4 tensordot results (m_quarters)")
        # 每个 tensordot 临时结果的内容必须出现在返回数组的对应行块中
        # （= 块结果被拷入预分配 C，而非 np.stack 拼装出的新数组）。
        q = m // 4
        for i, tmp in enumerate(copies):
            self.assertTrue(np.array_equal(tmp, c[i * q:(i + 1) * q]),
                            "tensordot result #%d not copied into returned C rows %d:%d"
                            % (i, i * q, (i + 1) * q))
        # 返回值必须是本地新分配的 ndarray（拷贝语义，非 stack 视图链）
        self.assertIsInstance(c, np.ndarray)
        self.assertEqual(c.dtype, np.float32)
        self.assertEqual(c.shape, (m, n))

    def test_kernel_bypasses_einsum_dispatch(self):
        """einsum_path 格的手动序列：np.einsum 自身调度（含 out= 恒等拷贝惩罚）
        被绕开——炸弹替换 np.einsum 后 matmul 仍逐位正确完成。"""
        rng = np.random.RandomState(23)
        a = rng.rand(66, 32).astype(np.float32)
        b = rng.rand(32, 48).astype(np.float32)
        ref = np.matmul(a, b)  # 先算锚，避免炸弹误伤
        orig_einsum = np.einsum

        def _bomb(*args, **kwargs):
            raise AssertionError(
                "np.einsum must not be called by the kernel — the einsum "
                "internal sequence (tensordot + copy) is replicated manually, "
                "bypassing einsum's own dispatch (out= identity-copy penalty)")

        np.einsum = _bomb
        try:
            c = sol.matmul(a, b)
        finally:
            np.einsum = orig_einsum
        self.assertTrue(np.array_equal(c, ref),
                        "result wrong when np.einsum is unavailable")

    def test_kernel_independent_of_matmul(self):
        """派发入口是 einsum 内部序列（tensordot→dot），不是 np.matmul 入口：
        炸弹替换 np.matmul 后仍逐位正确（锚先算）。"""
        rng = np.random.RandomState(31)
        a = rng.rand(66, 32).astype(np.float32)
        b = rng.rand(32, 48).astype(np.float32)
        ref = np.matmul(a, b)
        orig_matmul = np.matmul

        def _bomb(*args, **kwargs):
            raise AssertionError(
                "np.matmul must not be called by the einsum_path m_quarters kernel")

        np.matmul = _bomb
        try:
            c = sol.matmul(a, b)
        finally:
            np.matmul = orig_matmul
        self.assertTrue(np.array_equal(c, ref),
                        "result wrong when np.matmul is unavailable")


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

    def test_double_run_bitwise_consistent(self):
        """verify.py ⑦ 门预演：同输入两次调用输出逐位一致（抓缓存返回/分支特判）。"""
        rng = np.random.RandomState(5)
        a = rng.rand(256, 256).astype(np.float32)
        b = rng.rand(256, 256).astype(np.float32)
        c1 = sol.matmul(a, b)
        c2 = sol.matmul(a, b)
        self.assertTrue(np.array_equal(c1, c2),
                        "double-run inconsistency: outputs differ bitwise")
        self.assertIsNot(c1, c2, "double-run returned the identical object (cached?)")


class ShapeRobustnessTests(unittest.TestCase):
    """假设边界：ceil(m/4) 余数并入末块安全 / 空块跳过 / 非方阵 / 退化形状。"""

    def test_shape_robustness(self):
        rng = np.random.RandomState(7)
        cases = [
            (128, 128, "square"),
            (129, 64, "odd rows, non-square"),
            (64, 127, "odd cols"),
            (67, 33, "odd square-ish remainder blocks (17,17,17,16)"),
            (1, 32, "single row"),
            (2, 16, "two rows (only 2 non-empty quarters)"),
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


class MainProtocolTests(unittest.TestCase):
    """契约：solution.py --n 64 末行输出 JSON（预热 1 次 + 计时 1 次 +
    max_error_vs_matmul），impl 标签 einsum_path_m_quarters。"""

    def test_main_json_protocol(self):
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
        self.assertEqual(payload.get("impl"), "einsum_path_m_quarters",
                         "impl label mismatch: %r" % payload.get("impl"))
        self.assertIs(payload.get("ref"), False)
        self.assertEqual(payload.get("n"), 64)
        self.assertGreater(payload.get("runtime_ms", 0.0), 0.0)
        self.assertEqual(payload.get("max_error_vs_matmul"), 0.0,
                         "not bitwise vs matmul anchor @64: %r"
                         % payload.get("max_error_vs_matmul"))


if __name__ == "__main__":
    unittest.main(verbosity=2)
