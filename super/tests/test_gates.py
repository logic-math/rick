"""test_gates.py — job_47 task7 单元测试

CascadeGate（级联门框架）行为契约——全部用 tmpdir 内自建 fixture env（真 subprocess 跑，秒级）：
  - stage1 环境健康：env.json 可读 / data 脚本与 reference 存在 / 解释器探活（env/.venv 优先，
    坏 venv 报 stage1）/ 参考实现过正确性门（参考挂=环境坏）/ candidate 存在
  - stage2 正确性：协议 JSON 解析 / 坏解拒绝 / 子进程超时 / 非 JSON 输出 / 级联短路（bench 零调用）
  - stage3 性能：K 次独立进程计时取中位数 / bench 超时与失败协议 / score 缺省时的方向归一兜底
  - 软门禁：容差带内标记 tolerance_band / 带外劣化 soft_gate 拒 / 带外更优放行 /
    基线解析（elite → 同格历史中位数 → env.json baseline → 首解豁免）
  - 门先证红 preverify_red：exploit 全拒 + 金样例全过 → ok；泄漏 / 误杀 / 注册表缺失、空、
    malformed、样本路径不存在 → not ok；相对 sample_path 按 gates_dir → ws 两级解析
  - **框架零硬编码证明**：同一坏解，改 env.json 容差（1e-3→1e-9）从通过变拒绝；
    同一劣化解，改 relative_band（0.05→0.5）从 soft_gate 拒变带内通过
    （复用同一 CascadeGate 实例——env.json 每次 verify 重读，热更新生效）
"""
import json
import os
import shlex
import sys
import tempfile
import unittest

from super.superopt.gates import (
    CascadeGate,
    GateResult,
    STAGE1_ENV,
    STAGE2_CORRECTNESS,
    STAGE3_PERF,
    SOFT_GATE,
)


# ---- fixture env 资产（目录协议与 matmul 模板同构，脚本极度简化） ----

CORRECTNESS_SRC = '''#!/usr/bin/env python3
"""fixture 正确性门：candidate.compute() 与固定期望值 100.0 比较，容差读 env.json。"""
import importlib.util
import json
import os
import sys

HERE = os.path.dirname(os.path.abspath(__file__))
ENV_JSON = os.path.join(HERE, "..", "env.json")
EXPECTED = 100.0


def _emit(payload):
    print(json.dumps(payload))


def _tolerance():
    try:
        with open(ENV_JSON, "r", encoding="utf-8") as f:
            cfg = json.load(f)
        return float(cfg["tolerance"]["correctness"]["max_error"])
    except Exception:
        return 1e-3


def main():
    candidate = None
    args = sys.argv[1:]
    for i in range(len(args) - 1):
        if args[i] == "--candidate":
            candidate = args[i + 1]
    if not candidate:
        _emit({"passed": False, "error": "missing --candidate"})
        sys.exit(1)
    tol = _tolerance()
    try:
        spec = importlib.util.spec_from_file_location("candidate", os.path.abspath(candidate))
        mod = importlib.util.module_from_spec(spec)
        spec.loader.exec_module(mod)
        if getattr(mod, "_emit_garbage", False):
            print("garbage not json")
            sys.exit(3)
        got = float(mod.compute())
    except Exception as e:
        _emit({"passed": False, "error": "load/run: %s" % e})
        sys.exit(1)
    err = abs(got - EXPECTED)
    passed = err <= tol
    _emit({"passed": passed, "max_error": err, "tolerance": tol})
    sys.exit(0 if passed else 1)


if __name__ == "__main__":
    main()
'''

BENCH_SRC = '''#!/usr/bin/env python3
"""fixture 性能门：输出由 data/bench_schedule.json 驱动（第 n 次调用返回 schedule[n]）。"""
import json
import os
import sys

HERE = os.path.dirname(os.path.abspath(__file__))


def main():
    schedule = [100.0]
    try:
        with open(os.path.join(HERE, "bench_schedule.json"), "r", encoding="utf-8") as f:
            schedule = [float(v) for v in json.load(f)]
    except Exception:
        pass
    counter_path = os.path.join(HERE, "bench_counter")
    n = 0
    try:
        with open(counter_path, "r", encoding="utf-8") as f:
            n = int(f.read().strip() or "0")
    except Exception:
        n = 0
    try:
        with open(counter_path, "w", encoding="utf-8") as f:
            f.write(str(n + 1))
    except Exception:
        pass
    s = schedule[n % len(schedule)]
    print(json.dumps({"runtime_ms": s, "score": s, "ref_runtime_ms": 100.0, "n": 8, "k": 1}))
    sys.exit(0)


if __name__ == "__main__":
    main()
'''

HANGING_BENCH_SRC = "import time\n\ntime.sleep(5.0)\n"

FAILURE_BENCH_SRC = (
    "import json\n"
    "print(json.dumps({'passed': False, 'error': 'timeout'}))\n"
)

NOSCORE_BENCH_SRC = '''import json
import os

HERE = os.path.dirname(os.path.abspath(__file__))
schedule = [100.0]
try:
    with open(os.path.join(HERE, "bench_schedule.json"), "r", encoding="utf-8") as f:
        schedule = [float(v) for v in json.load(f)]
except Exception:
    pass
n = 0
try:
    with open(os.path.join(HERE, "bench_counter"), "r", encoding="utf-8") as f:
        n = int(f.read().strip() or "0")
except Exception:
    pass
try:
    with open(os.path.join(HERE, "bench_counter"), "w", encoding="utf-8") as f:
        f.write(str(n + 1))
except Exception:
    pass
s = schedule[n % len(schedule)]
print(json.dumps({"runtime_ms": s, "ref_runtime_ms": 100.0}))
'''

CAND_GOOD = "def compute():\n    return 100.0\n"
CAND_SLIGHTLY_OFF = "def compute():\n    return 100.0005\n"
CAND_BAD = "def compute():\n    return 42.0\n"
CAND_SLOW = "import time\n\n\ndef compute():\n    time.sleep(3.0)\n    return 100.0\n"
CAND_NOISY = "_emit_garbage = True\n\n\ndef compute():\n    return 100.0\n"


def default_env_json(**over):
    """fixture env.json（gate 定义在 env 侧——框架零硬编码的被测对象）。"""
    cfg = {
        "task": "fixture",
        "perf_type": "runtime_ms",
        "perf_desc": "low",
        "tolerance": {
            "correctness": {"max_error": 1e-3},
            "soft_gate": {"relative_band": 0.05},
        },
        "gates": {"correctness_timeout_s": 10, "bench_timeout_s": 10},
        "description": "fixture mini env",
    }
    cfg.update(over)
    return cfg


def _write_file(path, content, mode=0o644):
    os.makedirs(os.path.dirname(path), exist_ok=True)
    with open(path, "w", encoding="utf-8") as f:
        f.write(content)
    os.chmod(path, mode)


class FixtureEnv(object):
    """tmpdir 内自建 mini env + mini gates/（门先证红注册表与金样例）。"""

    def __init__(self, root):
        self.root = root
        self.env_dir = os.path.join(root, "env")
        self.gates_dir = os.path.join(root, "gates")
        self.candidates_dir = os.path.join(root, "candidates")

        _write_file(os.path.join(self.env_dir, "data", "test_correctness.py"), CORRECTNESS_SRC)
        _write_file(os.path.join(self.env_dir, "data", "bench.py"), BENCH_SRC)
        _write_file(os.path.join(self.env_dir, "data", "bench_schedule.json"), json.dumps([100.0]))
        _write_file(os.path.join(self.env_dir, "code", "reference.py"), CAND_GOOD)

        _write_file(os.path.join(self.gates_dir, "exploits", "zero_output.py"),
                    "def compute():\n    return 0.0\n")
        _write_file(os.path.join(self.gates_dir, "exploits", "cached_result.py"), CAND_GOOD)
        _write_file(os.path.join(self.gates_dir, "golden_examples", "good.py"), CAND_GOOD)

        self.write_env(default_env_json())
        self.write_registry([
            {"id": "zero_output", "name": "零输出作弊", "pattern": "returns 0.0",
             "sample_path": os.path.join(self.gates_dir, "exploits", "zero_output.py")},
        ])

        self.cand_good = self.candidate("good.py", CAND_GOOD)
        self.cand_slightly_off = self.candidate("slightly_off.py", CAND_SLIGHTLY_OFF)
        self.cand_bad = self.candidate("bad.py", CAND_BAD)
        self.cand_slow = self.candidate("slow.py", CAND_SLOW)
        self.cand_noisy = self.candidate("noisy.py", CAND_NOISY)

    # ---- 写入器（测试改 env/gates 定义用） ----

    def write_env(self, cfg):
        _write_file(os.path.join(self.env_dir, "env.json"), json.dumps(cfg, indent=1))

    def write_registry(self, entries):
        _write_file(os.path.join(self.gates_dir, "exploit_registry.json"),
                    json.dumps(entries, indent=1))

    def write_bench(self, src):
        _write_file(os.path.join(self.env_dir, "data", "bench.py"), src)

    def candidate(self, name, body):
        path = os.path.join(self.candidates_dir, name)
        _write_file(path, body)
        return path

    # ---- bench 控制 ----

    def set_schedule(self, values):
        _write_file(os.path.join(self.env_dir, "data", "bench_schedule.json"),
                    json.dumps(list(values)))

    def reset_bench(self):
        try:
            os.remove(os.path.join(self.env_dir, "data", "bench_counter"))
        except FileNotFoundError:
            pass

    def bench_calls(self):
        try:
            with open(os.path.join(self.env_dir, "data", "bench_counter"),
                      "r", encoding="utf-8") as f:
                return int(f.read().strip() or "0")
        except FileNotFoundError:
            return 0


class GatesTestCase(unittest.TestCase):
    def setUp(self):
        self.tmp = tempfile.TemporaryDirectory()
        self.fx = FixtureEnv(self.tmp.name)
        self.gate = CascadeGate(self.fx.env_dir, self.fx.gates_dir)

    def tearDown(self):
        self.tmp.cleanup()


# ---- stage1 环境健康 ----

class TestStage1EnvHealth(GatesTestCase):

    def test_missing_bench_script_fails_stage1(self):
        os.remove(os.path.join(self.fx.env_dir, "data", "bench.py"))
        r = self.gate.verify(self.fx.cand_good)
        self.assertFalse(r.passed)
        self.assertEqual(r.failure_stage, STAGE1_ENV)
        self.assertIn("data/bench.py", r.details)

    def test_missing_reference_impl_fails_stage1(self):
        os.remove(os.path.join(self.fx.env_dir, "code", "reference.py"))
        r = self.gate.verify(self.fx.cand_good)
        self.assertFalse(r.passed)
        self.assertEqual(r.failure_stage, STAGE1_ENV)
        self.assertIn("reference", r.details)

    def test_missing_candidate_fails_stage1(self):
        r = self.gate.verify(os.path.join(self.fx.root, "nope.py"))
        self.assertFalse(r.passed)
        self.assertEqual(r.failure_stage, STAGE1_ENV)
        self.assertIn("candidate", r.details)

    def test_broken_env_json_fails_stage1(self):
        _write_file(os.path.join(self.fx.env_dir, "env.json"), "not json {{{")
        r = self.gate.verify(self.fx.cand_good)
        self.assertFalse(r.passed)
        self.assertEqual(r.failure_stage, STAGE1_ENV)
        self.assertIn("env.json", r.details)

    def test_invalid_perf_desc_fails_stage1(self):
        self.fx.write_env(default_env_json(perf_desc="up"))
        r = self.gate.verify(self.fx.cand_good)
        self.assertFalse(r.passed)
        self.assertEqual(r.failure_stage, STAGE1_ENV)
        self.assertIn("perf_desc", r.details)

    def test_reference_fails_correctness_gate_fails_stage1(self):
        # 参考实现挂 = 环境坏（v5 环境失败 78% 的头部防线）
        _write_file(os.path.join(self.fx.env_dir, "code", "reference.py"),
                    "def compute():\n    return 999.0\n")
        r = self.gate.verify(self.fx.cand_good)
        self.assertFalse(r.passed)
        self.assertEqual(r.failure_stage, STAGE1_ENV)
        self.assertIn("reference", r.details)

    def test_venv_interpreter_preferred(self):
        marker = os.path.join(self.fx.root, "venv_used.marker")
        venv_bin = os.path.join(self.fx.env_dir, ".venv", "bin")
        os.makedirs(venv_bin)
        venv_py = os.path.join(venv_bin, "python")
        with open(venv_py, "w", encoding="utf-8") as f:
            f.write('#!/bin/sh\necho used >> %s\nexec %s "$@"\n'
                    % (shlex.quote(marker), shlex.quote(sys.executable)))
        os.chmod(venv_py, 0o755)
        r = self.gate.verify(self.fx.cand_good)
        self.assertTrue(r.passed, r.details)
        self.assertTrue(os.path.exists(marker), "env/.venv/bin/python 未被优先使用")

    def test_broken_venv_interpreter_fails_stage1(self):
        venv_bin = os.path.join(self.fx.env_dir, ".venv", "bin")
        os.makedirs(venv_bin)
        venv_py = os.path.join(venv_bin, "python")
        _write_file(venv_py, "not an executable\n")
        r = self.gate.verify(self.fx.cand_good)
        self.assertFalse(r.passed)
        self.assertEqual(r.failure_stage, STAGE1_ENV)
        self.assertIn("interpreter", r.details)


# ---- stage2 正确性 ----

class TestStage2Correctness(GatesTestCase):

    def test_good_solution_passes_full_cascade(self):
        self.fx.reset_bench()
        r = self.gate.verify(self.fx.cand_good)
        self.assertTrue(r.passed, r.details)
        self.assertIsNone(r.failure_stage)
        self.assertTrue(r.metrics["correctness"]["passed"])
        self.assertIn("max_error", r.metrics["correctness"])
        self.assertEqual(r.metrics["perf"]["value"], 100.0)
        self.assertEqual(r.metrics["soft"]["exempt"], True)
        self.assertTrue(r.details)

    def test_bad_solution_rejected_and_stage3_not_run(self):
        # 级联短路：stage2 挂 → bench 零调用
        self.fx.reset_bench()
        r = self.gate.verify(self.fx.cand_bad)
        self.assertFalse(r.passed)
        self.assertEqual(r.failure_stage, STAGE2_CORRECTNESS)
        self.assertFalse(r.metrics["correctness"]["passed"])
        self.assertNotIn("perf", r.metrics)
        self.assertEqual(self.fx.bench_calls(), 0, "stage2 失败后不得再跑 bench")

    def test_candidate_timeout_rejected_at_stage2(self):
        self.fx.write_env(default_env_json(
            gates={"correctness_timeout_s": 1, "bench_timeout_s": 10}))
        r = self.gate.verify(self.fx.cand_slow)
        self.assertFalse(r.passed)
        self.assertEqual(r.failure_stage, STAGE2_CORRECTNESS)
        self.assertIn("timeout", r.details)

    def test_non_json_output_rejected_at_stage2(self):
        r = self.gate.verify(self.fx.cand_noisy)
        self.assertFalse(r.passed)
        self.assertEqual(r.failure_stage, STAGE2_CORRECTNESS)
        self.assertIn("non-JSON", r.details)
        self.assertIn("garbage", r.details)

    def test_correctness_tolerance_is_env_driven(self):
        # 框架零硬编码证明：同一坏解（err=5e-4），env 容差 1e-3 通过、1e-9 拒绝
        r1 = self.gate.verify(self.fx.cand_slightly_off)
        self.assertTrue(r1.passed, r1.details)
        self.assertLessEqual(r1.metrics["correctness"]["max_error"], 1e-3)

        cfg = default_env_json()
        cfg["tolerance"]["correctness"]["max_error"] = 1e-9
        self.fx.write_env(cfg)
        # 同一 gate 实例：env.json 每次 verify 重读（热更新）
        r2 = self.gate.verify(self.fx.cand_slightly_off)
        self.assertFalse(r2.passed)
        self.assertEqual(r2.failure_stage, STAGE2_CORRECTNESS)


# ---- stage3 性能 ----

class TestStage3Perf(GatesTestCase):

    def test_median_of_k_independent_runs(self):
        self.fx.set_schedule([100.0, 300.0, 200.0])
        self.fx.reset_bench()
        r = self.gate.verify(self.fx.cand_good)
        self.assertTrue(r.passed, r.details)
        perf = r.metrics["perf"]
        self.assertEqual(perf["timings"], [100.0, 300.0, 200.0])
        self.assertEqual(perf["value"], 200.0)
        self.assertEqual(perf["score"], 200.0)
        self.assertEqual(perf["k"], 3)
        self.assertEqual(perf["perf_type"], "runtime_ms")
        self.assertEqual(self.fx.bench_calls(), 3)

    def test_k_constructor_controls_timing_count(self):
        gate2 = CascadeGate(self.fx.env_dir, self.fx.gates_dir, k=2)
        self.fx.set_schedule([100.0, 300.0])
        self.fx.reset_bench()
        r = gate2.verify(self.fx.cand_good)
        self.assertTrue(r.passed, r.details)
        self.assertEqual(r.metrics["perf"]["value"], 200.0)
        self.assertEqual(r.metrics["perf"]["k"], 2)
        self.assertEqual(self.fx.bench_calls(), 2)

    def test_bench_subprocess_timeout_fails_stage3(self):
        self.fx.write_bench(HANGING_BENCH_SRC)
        self.fx.write_env(default_env_json(
            gates={"correctness_timeout_s": 10, "bench_timeout_s": 1}))
        r = self.gate.verify(self.fx.cand_good)
        self.assertFalse(r.passed)
        self.assertEqual(r.failure_stage, STAGE3_PERF)
        self.assertIn("timeout", r.details)

    def test_bench_reported_failure_fails_stage3(self):
        self.fx.write_bench(FAILURE_BENCH_SRC)
        r = self.gate.verify(self.fx.cand_good)
        self.assertFalse(r.passed)
        self.assertEqual(r.failure_stage, STAGE3_PERF)
        self.assertIn("timeout", r.details)

    def test_score_fallback_direction_normalized_when_bench_omits_score(self):
        # bench 不输出 score → 框架按 perf_desc 方向归一兜底（low → 取负，越高越好）
        self.fx.write_bench(NOSCORE_BENCH_SRC)
        self.fx.set_schedule([100.0, 300.0, 200.0])
        self.fx.reset_bench()
        r = self.gate.verify(self.fx.cand_good)
        self.assertTrue(r.passed, r.details)
        self.assertEqual(r.metrics["perf"]["value"], 200.0)
        self.assertEqual(r.metrics["perf"]["score"], -200.0)


# ---- 软门禁 ----

class TestSoftGate(GatesTestCase):

    def test_in_band_marks_tolerance_band(self):
        self.fx.set_schedule([102.0, 102.0, 102.0])
        self.fx.reset_bench()
        r = self.gate.verify(self.fx.cand_good, elite_perf=100.0)
        self.assertTrue(r.passed, r.details)
        self.assertIsNone(r.failure_stage)
        self.assertTrue(r.tolerance_band)
        soft = r.metrics["soft"]
        self.assertEqual(soft["baseline"], 100.0)
        self.assertEqual(soft["baseline_source"], "elite")
        self.assertEqual(soft["verdict"], "band")
        self.assertAlmostEqual(soft["ratio"], 1.02)

    def test_worse_beyond_band_fails_soft_gate(self):
        self.fx.set_schedule([120.0, 120.0, 120.0])
        self.fx.reset_bench()
        r = self.gate.verify(self.fx.cand_good, elite_perf=100.0)
        self.assertFalse(r.passed)
        self.assertEqual(r.failure_stage, SOFT_GATE)
        self.assertEqual(r.metrics["soft"]["verdict"], "worse")
        self.assertFalse(r.tolerance_band)
        # perf 本身测量成功（soft 拒 ≠ 测量失败）
        self.assertEqual(r.metrics["perf"]["value"], 120.0)

    def test_better_beyond_band_passes_without_band_marker(self):
        self.fx.set_schedule([80.0, 80.0, 80.0])
        self.fx.reset_bench()
        r = self.gate.verify(self.fx.cand_good, elite_perf=100.0)
        self.assertTrue(r.passed, r.details)
        self.assertFalse(r.tolerance_band)
        self.assertEqual(r.metrics["soft"]["verdict"], "better")

    def test_history_median_used_as_baseline_when_no_elite(self):
        # invalidation 后精英清空、history 尚在的场景：同格历史中位数为比较锚点
        self.fx.set_schedule([102.0, 102.0, 102.0])
        self.fx.reset_bench()
        r = self.gate.verify(self.fx.cand_good, history=[100.0, 100.0, 100.0])
        self.assertTrue(r.passed, r.details)
        self.assertTrue(r.tolerance_band)
        self.assertEqual(r.metrics["soft"]["baseline_source"], "history")
        self.assertEqual(r.metrics["soft"]["baseline"], 100.0)

    def test_env_json_baseline_used_when_no_elite_no_history(self):
        cfg = default_env_json()
        cfg["tolerance"]["soft_gate"]["baseline"] = 100.0
        self.fx.write_env(cfg)
        self.fx.set_schedule([102.0, 102.0, 102.0])
        self.fx.reset_bench()
        r = self.gate.verify(self.fx.cand_good)
        self.assertTrue(r.passed, r.details)
        self.assertTrue(r.tolerance_band)
        self.assertEqual(r.metrics["soft"]["baseline_source"], "env")
        self.assertEqual(r.metrics["soft"]["baseline"], 100.0)

    def test_first_solution_exempt_when_no_baseline(self):
        self.fx.set_schedule([500.0, 500.0, 500.0])
        self.fx.reset_bench()
        r = self.gate.verify(self.fx.cand_good)
        self.assertTrue(r.passed, r.details)
        self.assertEqual(r.metrics["soft"]["exempt"], True)
        self.assertFalse(r.tolerance_band)
        self.assertIsNone(r.metrics["soft"]["baseline"])

    def test_relative_band_is_env_driven(self):
        # 框架零硬编码证明：同一劣化解（ratio=1.2），band 0.05 拒、0.5 带内通过
        self.fx.set_schedule([120.0, 120.0, 120.0])
        self.fx.reset_bench()
        r1 = self.gate.verify(self.fx.cand_good, elite_perf=100.0)
        self.assertFalse(r1.passed)
        self.assertEqual(r1.failure_stage, SOFT_GATE)

        cfg = default_env_json()
        cfg["tolerance"]["soft_gate"]["relative_band"] = 0.5
        self.fx.write_env(cfg)
        self.fx.reset_bench()
        r2 = self.gate.verify(self.fx.cand_good, elite_perf=100.0)
        self.assertTrue(r2.passed, r2.details)
        self.assertTrue(r2.tolerance_band)

    def test_high_direction_perf_type(self):
        # perf_desc=high + perf_type=score：perf 值直接取 bench 输出的 score 字段，方向反转
        self.fx.write_env(default_env_json(perf_type="score", perf_desc="high"))
        self.fx.set_schedule([50.0, 50.0, 50.0])

        self.fx.reset_bench()
        r_worse = self.gate.verify(self.fx.cand_good, elite_perf=100.0)
        self.assertFalse(r_worse.passed)
        self.assertEqual(r_worse.failure_stage, SOFT_GATE)
        self.assertEqual(r_worse.metrics["perf"]["value"], 50.0)

        self.fx.reset_bench()
        r_better = self.gate.verify(self.fx.cand_good, elite_perf=10.0)
        self.assertTrue(r_better.passed, r_better.details)
        self.assertEqual(r_better.metrics["soft"]["verdict"], "better")


# ---- 门先证红 ----

class TestPreverifyRed(GatesTestCase):

    def test_all_exploits_rejected_and_goldens_pass(self):
        out = self.gate.preverify_red()
        self.assertTrue(out["ok"], out)
        self.assertEqual(out["leaked"], [])
        self.assertEqual(out["false_killed"], [])
        self.assertGreaterEqual(out["exploits_checked"], 1)
        self.assertGreaterEqual(out["goldens_checked"], 1)

    def test_exploit_leak_makes_not_ok(self):
        self.fx.write_registry([
            {"id": "zero_output",
             "sample_path": os.path.join(self.fx.gates_dir, "exploits", "zero_output.py")},
            {"id": "cached_result",
             "sample_path": os.path.join(self.fx.gates_dir, "exploits", "cached_result.py")},
        ])
        out = self.gate.preverify_red()
        self.assertFalse(out["ok"])
        self.assertEqual(out["leaked"], ["cached_result"])
        self.assertEqual(out["false_killed"], [])

    def test_golden_false_killed_makes_not_ok(self):
        _write_file(os.path.join(self.fx.gates_dir, "golden_examples", "bad_golden.py"),
                    "def compute():\n    return 55.0\n")
        out = self.gate.preverify_red()
        self.assertFalse(out["ok"])
        self.assertEqual(out["false_killed"], ["bad_golden.py"])
        self.assertEqual(out["leaked"], [])

    def test_missing_registry_makes_not_ok(self):
        os.remove(os.path.join(self.fx.gates_dir, "exploit_registry.json"))
        out = self.gate.preverify_red()
        self.assertFalse(out["ok"])
        self.assertIn("error", out)
        self.assertIn("registry", out["error"])

    def test_empty_registry_makes_not_ok(self):
        self.fx.write_registry([])
        out = self.gate.preverify_red()
        self.assertFalse(out["ok"])
        self.assertIn("error", out)
        self.assertIn("empty", out["error"])

    def test_malformed_registry_entry_makes_not_ok(self):
        self.fx.write_registry([{"id": "x"}])
        out = self.gate.preverify_red()
        self.assertFalse(out["ok"])
        self.assertIn("error", out)
        self.assertIn("malformed", out["error"])

    def test_registry_sample_path_not_found_makes_not_ok(self):
        self.fx.write_registry([{"id": "gone", "sample_path": "/no/such/file.py"}])
        out = self.gate.preverify_red()
        self.assertFalse(out["ok"])
        self.assertIn("error", out)
        self.assertIn("not found", out["error"])

    def test_relative_sample_path_resolved_against_gates_dir(self):
        self.fx.write_registry([
            {"id": "zero_output", "sample_path": "exploits/zero_output.py"},
        ])
        out = self.gate.preverify_red()
        self.assertTrue(out["ok"], out)
        self.assertEqual(out["exploits_checked"], 1)


# ---- GateResult 结构 ----

class TestGateResultShape(unittest.TestCase):

    def test_defaults_and_tolerance_band_property(self):
        r = GateResult(passed=True)
        self.assertIsNone(r.failure_stage)
        self.assertEqual(r.metrics, {})
        self.assertEqual(r.details, "")
        self.assertFalse(r.tolerance_band)

        r2 = GateResult(passed=True, metrics={"soft": {"tolerance_band": True}})
        self.assertTrue(r2.tolerance_band)

    def test_invalid_k_rejected(self):
        with self.assertRaises(ValueError):
            CascadeGate("env", "gates", k=0)


if __name__ == "__main__":
    unittest.main()
