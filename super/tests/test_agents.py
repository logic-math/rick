"""test_agents.py — job_47 task5 单元测试

PiAgentClient（pi 池客户端）行为契约（mock_pi 六场景全覆盖）：
  - ok_builder：usage 聚合精确数值（两条 assistant message_end 的 input 相加）；
    **user echo 的伪造 usage 不计数**（bugs.md：pi 对 user/assistant 都发
    message_end）；六字段从 ```json 围栏提取；角色增量（proposals）保留在 data
  - ok_coding：tool_calls 去重列表（首现顺序）；六字段 + diff_path/self_check
  - zero_usage：message_end 无 usage → 兜底取最后一次非零 message_update 顶层
    usage；user echo 的 message_update usage 同样被过滤
  - all_zero：全程零 → usage 全 0（降级路径），valid 仍 True
  - bad_json：final 无 JSON → valid=False，summary=final 前 200 字；usage 照常计
  - timeout_sleep：客户端超时 kill（TERM→grace→KILL）→ exit_code 非 0，
    valid=False，wall_ms ≥ timeout
  - _extract_six_fields 单元：裸 JSON / 围栏 JSON / 无 JSON / 带角色增量字段
"""
import json
import os
import sys
import time
import unittest
from unittest import mock

sys.path.insert(0, os.path.abspath(os.path.join(os.path.dirname(__file__), os.pardir, os.pardir)))

from super.superopt.agents import (  # noqa: E402
    AgentResult,
    PiAgentClient,
    _extract_six_fields,
)

ROOT = os.path.abspath(os.path.join(os.path.dirname(__file__), os.pardir, os.pardir))
MOCK_PI = os.path.join(ROOT, "super", "tests", "mock_pi.py")


def make_client():
    """PiAgentClient 指向 mock_pi.py（exec 形式：可执行文件 + shebang）。"""
    return PiAgentClient(pi_path=MOCK_PI)


def run_scenario(scenario, timeout_s=30, role="builder", agent_dir=None,
                 max_turns=None):
    """在 MOCK_SCENARIO 下跑一次客户端（临时 prompt 文件 + 临时 workdir）。"""
    import tempfile
    with tempfile.TemporaryDirectory() as td:
        prompt = os.path.join(td, "prompt_builder.md")
        with open(prompt, "w", encoding="utf-8") as f:
            f.write("# 生成 experiment_plan\n按系统提示词输出约定 JSON。\n")
        with mock.patch.dict(os.environ, {"MOCK_SCENARIO": scenario}):
            client = make_client()
            if agent_dir is not None:
                client = PiAgentClient(pi_path=MOCK_PI, agent_dir=agent_dir)
            return client.run(role=role, prompt_path=prompt,
                              workdir=td, timeout_s=timeout_s,
                              max_turns=max_turns)


class TestOkBuilder(unittest.TestCase):
    def test_usage_aggregation_and_role_filter(self):
        r = run_scenario("ok_builder")
        self.assertTrue(r.valid)
        # 聚合：assistant #1(input=100,cacheRead=1000) + #2(input=200,cacheRead=2000,cacheWrite=10)
        self.assertEqual(r.usage["input_tokens"], 300)
        self.assertEqual(r.usage["output_tokens"], 130)
        self.assertEqual(r.usage["cache_read"], 3000)
        self.assertEqual(r.usage["cache_write"], 10)
        self.assertEqual(r.usage["total_tokens"], 3440)
        self.assertAlmostEqual(r.usage["cost_usd"], 0.03, places=6)
        # user echo 的伪造 usage（input=999999）绝不可混入
        self.assertNotEqual(r.usage["input_tokens"], 999999 + 300)

    def test_six_fields_and_role_extras(self):
        r = run_scenario("ok_builder")
        self.assertTrue(r.valid)
        self.assertAlmostEqual(r.score, 0.85)
        self.assertAlmostEqual(r.perf_value, 123.4)
        self.assertEqual(r.perf_type, "runtime_ms")
        self.assertEqual(r.perf_desc, "low")
        self.assertEqual(r.summary, "建议在 blocked 格尝试转置缓存友好布局")
        # 角色增量（adjust 的 proposals）保留在 data（任务目标：六字段基座 + 角色增量）
        self.assertIn("proposals", r.data)
        self.assertEqual(r.data["proposals"][0]["type"], "soft_weight")

    def test_final_message_and_meta(self):
        r = run_scenario("ok_builder")
        self.assertIn("结论如下", r.final_message)
        self.assertEqual(r.role, "builder")
        self.assertEqual(r.exit_code, 0)
        self.assertEqual(r.tool_calls, [])
        self.assertGreaterEqual(r.wall_ms, 0)


class TestOkCoding(unittest.TestCase):
    def test_tool_calls_dedup_order(self):
        r = run_scenario("ok_coding", role="coding")
        self.assertEqual(r.tool_calls, ["read", "bash", "write"])
        self.assertTrue(r.valid)
        self.assertEqual(r.usage["input_tokens"], 300)
        self.assertEqual(r.usage["output_tokens"], 150)
        # coding 角色增量
        self.assertEqual(r.data["diff_path"], "codes/abc123/solution.py")
        self.assertTrue(r.data["self_check"]["passed"])


class TestZeroUsageFallback(unittest.TestCase):
    def test_fallback_last_nonzero_message_update(self):
        r = run_scenario("zero_usage")
        # message_update 兜底：取最后一次非零（500/60），user echo 的 999999 被过滤
        self.assertEqual(r.usage["input_tokens"], 500)
        self.assertEqual(r.usage["output_tokens"], 60)
        self.assertAlmostEqual(r.usage["cost_usd"], 0.008, places=6)
        self.assertTrue(r.valid)  # JSON 正常解析，与 usage 无关


class TestAllZero(unittest.TestCase):
    def test_degraded_all_zero(self):
        r = run_scenario("all_zero")
        self.assertEqual(r.usage["input_tokens"], 0)
        self.assertEqual(r.usage["output_tokens"], 0)
        self.assertEqual(r.usage["cache_read"], 0)
        self.assertEqual(r.usage["cache_write"], 0)
        self.assertEqual(r.usage["total_tokens"], 0)
        self.assertEqual(r.usage["cost_usd"], 0.0)
        self.assertTrue(r.valid)  # 六字段照常提取


class TestBadJson(unittest.TestCase):
    def test_graceful_degradation(self):
        r = run_scenario("bad_json")
        self.assertFalse(r.valid)
        self.assertIsNone(r.score)
        # summary = final_message 前 200 字（不炸循环）
        self.assertEqual(r.summary, r.final_message[:200])
        self.assertLessEqual(len(r.summary), 200)
        # usage 照常计量
        self.assertEqual(r.usage["input_tokens"], 50)
        self.assertEqual(r.usage["output_tokens"], 30)


class TestTimeoutKill(unittest.TestCase):
    def test_timeout_ladder_and_nonzero_exit(self):
        t0 = time.monotonic()
        r = run_scenario("timeout_sleep", timeout_s=2)
        elapsed = time.monotonic() - t0
        self.assertGreaterEqual(elapsed, 1.8)
        self.assertLess(elapsed, 30)  # TERM 立即生效，不该走满 grace+KILL
        self.assertNotEqual(r.exit_code, 0)
        self.assertFalse(r.valid)
        self.assertEqual(r.usage["total_tokens"], 0)
        self.assertGreaterEqual(r.wall_ms, 1800)


class TestTurnBudget(unittest.TestCase):
    """改动②（env_8 实测成本失控）：回合预算——NDJSON 计数 assistant 回合，
    达上限走 kill 梯优雅终止；已产出内容全部保留（截断不等于无产出）。"""

    def test_truncated_at_budget(self):
        r = run_scenario("many_turns", timeout_s=60, max_turns=5)
        self.assertTrue(r.truncated)
        # summary 前缀标注（spec：[turn-budget-exceeded: N] ）
        self.assertTrue((r.summary or "").startswith("[turn-budget-exceeded: 5]"),
                        r.summary)
        # 不是等到墙钟超时（many_turns 全跑完约 2s；截断应在 ~1s 内）
        self.assertLess(r.wall_ms, 30000)
        # 进程被 kill 梯终止（SIGTERM → 非零退出）
        self.assertNotEqual(r.exit_code, 0)
        # 截断不等于无产出：已产出的回合内容/usage/tool_calls 保留
        self.assertIn("回合分析中", r.final_message)
        self.assertGreater(r.usage["total_tokens"], 0)
        self.assertIn("read", r.tool_calls)
        # payload 携带 truncated（事件可观测性）
        self.assertTrue(r.to_payload()["truncated"])
        # 证明早期终止：未跑满 20 回合（每回合 input=100）
        self.assertLess(r.usage["input_tokens"], 20 * 100)

    def test_no_budget_runs_to_completion(self):
        r = run_scenario("many_turns", timeout_s=60, max_turns=None)
        self.assertFalse(r.truncated)
        self.assertEqual(r.exit_code, 0)
        # 完整跑完 20 回合：20 × (input=100, output=20)
        self.assertEqual(r.usage["input_tokens"], 20 * 100)
        self.assertEqual(r.usage["output_tokens"], 20 * 20)
        # 最后回合的六字段照常提取
        self.assertTrue(r.valid)
        self.assertAlmostEqual(r.score, 0.85)

    def test_zero_budget_means_unlimited(self):
        r = run_scenario("many_turns", timeout_s=60, max_turns=0)
        self.assertFalse(r.truncated)
        self.assertEqual(r.exit_code, 0)


class TestExtractSixFields(unittest.TestCase):
    def test_bare_json(self):
        payload = {"valid": True, "score": 0.5, "summary": "裸 JSON 场景"}
        got, data = _extract_six_fields(json.dumps(payload))
        self.assertTrue(got["valid"])
        self.assertAlmostEqual(got["score"], 0.5)
        self.assertEqual(got["summary"], "裸 JSON 场景")
        self.assertEqual(data, payload)

    def test_fenced_json_with_prose(self):
        payload = {"valid": True, "perf_value": 1.0}
        text = "分析如下：\n```json\n%s\n```\n以上。" % json.dumps(payload)
        got, data = _extract_six_fields(text)
        self.assertTrue(got["valid"])
        self.assertAlmostEqual(got["perf_value"], 1.0)
        self.assertEqual(data, payload)

    def test_no_json(self):
        got, data = _extract_six_fields("纯文本废话，没有任何结构化输出。")
        self.assertFalse(got["valid"])
        self.assertIsNone(got["score"])
        self.assertEqual(got["summary"], "纯文本废话，没有任何结构化输出。")
        self.assertEqual(data, {})

    def test_missing_fields_default(self):
        got, data = _extract_six_fields('{"valid": true}')
        self.assertTrue(got["valid"])
        self.assertIsNone(got["score"])
        self.assertIsNone(got["perf_value"])
        self.assertIsNone(got["perf_type"])
        self.assertIsNone(got["perf_desc"])
        self.assertIsNone(got["summary"])

    def test_extra_fields_preserved_in_data(self):
        payload = {"valid": True, "experiment_plan": {"idea": "x"}, "fix_report": {"attempts": 1}}
        got, data = _extract_six_fields(json.dumps(payload))
        self.assertIn("experiment_plan", data)
        self.assertIn("fix_report", data)

    def test_text_before_json_array_ignored(self):
        # 数组不是对象——不应被当六字段载体
        got, data = _extract_six_fields("[1, 2, 3]")
        self.assertFalse(got["valid"])
        self.assertEqual(data, {})

    def test_summary_truncated_to_200(self):
        long_text = "长" * 500
        got, _ = _extract_six_fields(long_text)
        self.assertEqual(len(got["summary"]), 200)


class TestAgentClientConstruction(unittest.TestCase):
    def test_pi_path_required(self):
        with self.assertRaises(TypeError):
            PiAgentClient()  # noqa: E501 缺 pi_path

    def test_agent_dir_env_injected(self):
        # agent_dir 传递 → PI_CODING_AGENT_DIR 进子进程环境（经 ok_coding 场景验证不炸）
        r = run_scenario("ok_coding", role="coding", agent_dir="/tmp/fake-agent-dir")
        self.assertTrue(r.valid)

    def test_workdir_must_exist(self):
        client = make_client()
        with self.assertRaises(ValueError):
            client.run(role="builder", prompt_path=MOCK_PI,
                       workdir="/nonexistent/workdir/xyz", timeout_s=5)


class TestPerRoleModel(unittest.TestCase):
    """改动①（优化空间①）：per-role 模型档位——--model 参数 + 可观测性。"""

    def _argv_recorder(self, td):
        """写一个 fake pi：把 argv 落盘后输出最小合法 NDJSON（六字段 JSON）。"""
        script = os.path.join(td, "fake_argv_pi.py")
        body = (
            "#!/usr/bin/env python3\n"
            "import json, sys\n"
            "with open(sys.argv[0] + '.argv', 'w') as g:\n"
            "    json.dump(sys.argv[1:], g)\n"
            "final = '```json\\n' + json.dumps({'valid': True}) + '\\n```'\n"
            "ev = {'type': 'message_end', 'message': {'role': 'assistant',\n"
            "      'content': [{'type': 'text', 'text': final}]}}\n"
            "print(json.dumps(ev))\n"
        )
        with open(script, "w", encoding="utf-8") as f:
            f.write(body)
        os.chmod(script, 0o755)
        return script

    def test_model_flag_passed_and_observable(self):
        import tempfile
        with tempfile.TemporaryDirectory() as td:
            pi = self._argv_recorder(td)
            prompt = os.path.join(td, "prompt_builder.md")
            with open(prompt, "w", encoding="utf-8") as f:
                f.write("# test\n")
            client = PiAgentClient(pi_path=pi)
            r = client.run(role="builder", prompt_path=prompt, workdir=td,
                           timeout_s=30, model="catpaw-proxy/deepseek-v4-flash")
            # 结果可观测：AgentResult.model 回填 + payload 序列化
            self.assertEqual(r.model, "catpaw-proxy/deepseek-v4-flash")
            self.assertEqual(r.to_payload()["model"],
                             "catpaw-proxy/deepseek-v4-flash")
            # argv 实证：--model <id> 追加在命令尾
            with open(pi + ".argv", encoding="utf-8") as f:
                argv = json.load(f)
            self.assertIn("--model", argv)
            self.assertEqual(argv[argv.index("--model") + 1],
                             "catpaw-proxy/deepseek-v4-flash")

    def test_no_model_no_flag(self):
        import tempfile
        with tempfile.TemporaryDirectory() as td:
            pi = self._argv_recorder(td)
            prompt = os.path.join(td, "prompt_builder.md")
            with open(prompt, "w", encoding="utf-8") as f:
                f.write("# test\n")
            client = PiAgentClient(pi_path=pi)
            r = client.run(role="builder", prompt_path=prompt, workdir=td,
                           timeout_s=30)
            self.assertEqual(r.model, "")           # 默认模型：空串
            with open(pi + ".argv", encoding="utf-8") as f:
                argv = json.load(f)
            self.assertNotIn("--model", argv)        # 不传 flag


if __name__ == "__main__":
    unittest.main()
