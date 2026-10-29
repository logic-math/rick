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


class _FakePiScript:
    """task57：按需构造 fake pi（TestPerRoleModel 同范式——不依赖 mock_pi）。

    body 里可用 $COUNT 写调用计数文件（$SCRIPT.count.json），供断言
    「派发了几次」（quota 不 resume / repair resume 一次）。计数文件写在
    脚本同目录，避免污染测试 workdir。
    """

    def __init__(self, td: str, name: str, body: str):
        self.path = os.path.join(td, name)
        with open(self.path, "w", encoding="utf-8") as f:
            f.write(body)
        os.chmod(self.path, 0o755)

    @property
    def count_path(self) -> str:
        return self.path + ".count.json"

    def count(self) -> int:
        import json
        try:
            with open(self.count_path, encoding="utf-8") as f:
                return int(json.load(f))
        except (OSError, ValueError):
            return 0


# 模板：计数 + 单条 assistant message_end（text/usage 自定义）
_EMIT_TMPL = """#!/usr/bin/env python3
import json, os, sys
_p = os.path.abspath(sys.argv[0]) + ".count.json"
_n = 0
try:
    _n = json.load(open(_p)) + 1
except Exception:
    _n = 1
json.dump(_n, open(_p, "w"))
%s
print(json.dumps(ev, ensure_ascii=False))
sys.stdout.flush()
"""


class TestQuotaError(unittest.TestCase):
    """task57 R1：配额/额度报错识别——infra_failure 标记 + 不 resume。

    env_11 实测形态：pi 正常退出 0、回复是额度报错文案、usage 全零。
    旧行为：走 failure-resume（同一凭据重试必然再失败——白烧一次派发）。
    新行为：标 infra_failure=True 直接返回，调度器走熔断/探活路径。
    """

    def _quota_pi(self, td: str) -> _FakePiScript:
        msg = ("您的2026-10-29额度已使用完毕，当前剩余额度为 0。"
               "额度将于次日 00:00:00自动重置。如需申请提额："
               "https://credit-hub.sankuai.com/credit-mgmt/credit-apply")
        emit = ("ev = {'type': 'message_end', 'message': {'role': 'assistant',\n"
                "      'content': [{'type': 'text', 'text': %r}]}}" % msg)
        return _FakePiScript(td, "quota_pi.py", _EMIT_TMPL % emit)

    def test_quota_error_flags_infra_and_skips_resume(self):
        import tempfile
        with tempfile.TemporaryDirectory() as td:
            pi = self._quota_pi(td)
            prompt = os.path.join(td, "prompt_builder.md")
            with open(prompt, "w", encoding="utf-8") as f:
                f.write("# test\n")
            client = PiAgentClient(pi_path=pi.path)
            r = client.run(role="builder", prompt_path=prompt,
                           workdir=td, timeout_s=30)
            # 配额识别：infra 标记 + 0 token + 无 JSON
            self.assertTrue(r.infra_failure, "配额报错应标 infra_failure")
            self.assertFalse(r.valid)
            self.assertEqual(r.usage["total_tokens"], 0)
            # 不 resume（只派发 1 次）+ 不修复（json_repair=0）
            self.assertEqual(pi.count(), 1,
                             "配额报错不该 resume——同一凭据重试只会再失败")
            self.assertEqual(r.json_repair, 0)
            # session 已建（若未来走 resume 仍有会话句柄）+ payload 可观测
            self.assertTrue(r.session_id)
            self.assertTrue(r.to_payload()["infra_failure"])
            # 诊断线索保留：额度文案进 summary（first 200 chars）
            self.assertIn("额度", r.summary or "")

    def test_non_quota_marker_not_flagged(self):
        """非配额文案（普通 bad-json）不误标 infra。"""
        import tempfile
        with tempfile.TemporaryDirectory() as td:
            emit = ("ev = {'type': 'message_end', 'message': {'role': 'assistant',\n"
                    "      'content': [{'type': 'text', 'text': '目标格父本缺失，"
                    "无法完成实现。'}]}}")
            pi = _FakePiScript(td, "plain_pi.py", _EMIT_TMPL % emit)
            prompt = os.path.join(td, "prompt_builder.md")
            with open(prompt, "w", encoding="utf-8") as f:
                f.write("# test\n")
            client = PiAgentClient(pi_path=pi.path)
            r = client.run(role="builder", prompt_path=prompt,
                           workdir=td, timeout_s=30)
            self.assertFalse(r.infra_failure)


class TestJsonRepairEmptyFinalMessage(unittest.TestCase):
    """task57 R2：json_repair 触发条件修正——去掉 final_message 非空前置。

    截断/空回场景 final_message 可能为空（agent 没来得及说话），但 JSON
    契约输出仍值得补发一次（env_11 实测：coding 截断后 final_message=None
    → 旧条件不触发 → 产物全靠 debug 代偿）。
    """

    def test_empty_final_message_still_repairs(self):
        import tempfile
        with tempfile.TemporaryDirectory() as td:
            # assistant 轮带 usage 但无文本（final_message=""）
            emit = ("ev = {'type': 'message_end', 'message': {'role': 'assistant',\n"
                    "      'content': [],\n"
                    "      'usage': {'input': 10, 'output': 5, 'cacheRead': 0,\n"
                    "                'cacheWrite': 0, 'totalTokens': 15,\n"
                    "                'cost': {'total': 0.001}}}}")
            pi = _FakePiScript(td, "empty_pi.py", _EMIT_TMPL % emit)
            prompt = os.path.join(td, "prompt_builder.md")
            with open(prompt, "w", encoding="utf-8") as f:
                f.write("# test\n")
            client = PiAgentClient(pi_path=pi.path)
            r = client.run(role="builder", prompt_path=prompt,
                           workdir=td, timeout_s=30)
            # 旧条件（final_message 非空才修复）不会 resume；新条件必须 resume 一次
            self.assertEqual(pi.count(), 2,
                             "空 final_message + 无 JSON 应触发一次修复 resume")
            self.assertEqual(r.json_repair, 1)
            self.assertEqual(r.usage["total_tokens"], 15)


class TestArtifactPath(unittest.TestCase):
    """task57 R2：AgentResult.artifact_path——本轮新落盘候选的可观测物证。"""

    def _artifact_pi(self, td: str, write_solution: bool) -> _FakePiScript:
        """生成脚本零手写反斜杠（chr(10) 代换行——多层转义在生成式测试里易错）。

        write_solution=True：先向 cwd（= workdir）写 solution.py（模拟
        agent 产物落盘），再发一条无 JSON 的 message_end。
        """
        sol_src = "# agent-written candidate" + chr(10) + "print(42)" + chr(10)
        parts = []
        if write_solution:
            parts.append(
                "open(os.path.join(os.getcwd(), 'solution.py'), 'w')"
                ".write(%r)" % sol_src)
        parts += [
            "ev = {'type': 'message_end', 'message': {'role': 'assistant',",
            "      'content': [{'type': 'text', 'text': %r}],"
            % "已完成实现（未输出 JSON 契约）",
            "      'usage': {'input': 30, 'output': 10, 'cacheRead': 0,",
            "                'cacheWrite': 0, 'totalTokens': 40,",
            "                'cost': {'total': 0.002}}}}",
        ]
        emit = chr(10).join(parts)
        return _FakePiScript(td, "artifact_pi.py", _EMIT_TMPL % emit)

    def test_fresh_artifact_located(self):
        import tempfile
        with tempfile.TemporaryDirectory() as td:
            pi = self._artifact_pi(td, write_solution=True)
            prompt = os.path.join(td, "prompt_builder.md")
            with open(prompt, "w", encoding="utf-8") as f:
                f.write("# test\n")
            client = PiAgentClient(pi_path=pi.path)
            r = client.run(role="coding", prompt_path=prompt,
                           workdir=td, timeout_s=30)
            # 产物被定位（mtime ≥ 运行起点；含 repair resume 后的覆盖写）
            self.assertTrue(r.artifact_path.endswith("solution.py"),
                            r.artifact_path)
            self.assertTrue(os.path.isfile(r.artifact_path))
            # repair 照常触发（无 JSON）——产物定位不受 resume 影响
            self.assertEqual(r.json_repair, 1)
            self.assertTrue(r.to_payload()["artifact_path"])

    def test_no_artifact_empty_string(self):
        import tempfile
        with tempfile.TemporaryDirectory() as td:
            pi = self._artifact_pi(td, write_solution=False)
            prompt = os.path.join(td, "prompt_builder.md")
            with open(prompt, "w", encoding="utf-8") as f:
                f.write("# test\n")
            client = PiAgentClient(pi_path=pi.path)
            r = client.run(role="coding", prompt_path=prompt,
                           workdir=td, timeout_s=30)
            self.assertEqual(r.artifact_path, "")

    def test_seeded_parent_not_mistaken_as_artifact(self):
        """种子父本（旧 mtime）不算本轮产物——mtime 基线的判别力。"""
        import tempfile
        import time as _time
        with tempfile.TemporaryDirectory() as td:
            # 预置 solution.py（模拟 _seed_parent_code 的父本，mtime 在过去）
            seeded = os.path.join(td, "solution.py")
            with open(seeded, "w", encoding="utf-8") as f:
                f.write("# parent seed\n")
            past = time.time() - 3600
            os.utime(seeded, (past, past))
            pi = self._artifact_pi(td, write_solution=False)
            prompt = os.path.join(td, "prompt_builder.md")
            with open(prompt, "w", encoding="utf-8") as f:
                f.write("# test\n")
            client = PiAgentClient(pi_path=pi.path)
            r = client.run(role="coding", prompt_path=prompt,
                           workdir=td, timeout_s=30)
            self.assertEqual(r.artifact_path, "",
                             "旧 mtime 的种子父本不该被当成 agent 产物")
