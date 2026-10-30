"""test_prompts.py — superopt.prompts 单元测试（job_47/task6）

覆盖：
- 三级模板查找链优先级：<ws>/prompts > <ws>/env/prompts > 仓内 super/prompts（默认）
- 占位符替换：默认模板占位符全集可替换（自动发现，无 {{ 残留）；
  缺占位数据保留原样不炸（宽松替换）；值格式化（str 原样 / 复杂结构 JSON）
- write_prompt 落盘契约：codes 工作目录内 prompt_{role}.md + 返回路径 + 幂等覆写
- 角色名校验（防路径注入）与全链缺失时的 FileNotFoundError（含三条查找路径）
- 四默认模板内容契约：角色关键词（proposals/experiment_plan/diff_path/fix_report 等，
  与 gate2 断言同源）+ 输出契约段（```json）+ 不含 matmul 特定细节（模板通用性）
"""
import os
import sys
import tempfile
import unittest

_HERE = os.path.dirname(os.path.abspath(__file__))
_SUPER = os.path.dirname(_HERE)      # super/
_REPO = os.path.dirname(_SUPER)      # 仓库根
for _p in (_SUPER, _REPO):
    if _p not in sys.path:
        sys.path.insert(0, _p)

from superopt import prompts
from superopt.prompts import DEFAULT_TEMPLATE_DIR, ROLES


def _probe_value(key):
    """按占位符键名给出类型合理的探针值（供全替换测试）。"""
    if key in ("visit", "attempt", "max_attempts", "iteration_id"):
        return 7
    if key == "weight":
        return 0.5
    if key in ("cell_descriptor", "cell_elite", "experiment_plan",
               "self_check_result", "neighbor_elites", "history_failures",
               "cells_status", "feature_axes"):
        return {"probe": [key, 1, True]}
    if key == "recent_events":
        return ["evt-1", "evt-2"]
    return "probe-" + key


class _TmpWsTestCase(unittest.TestCase):
    """每个用例一个空工作区（无任何模板覆盖 → 走仓内默认）。"""

    def setUp(self):
        self._td = tempfile.TemporaryDirectory()
        self.ws = os.path.join(self._td.name, "ws")
        os.makedirs(self.ws)

    def tearDown(self):
        self._td.cleanup()

    def _write(self, rel, content):
        path = os.path.join(self.ws, rel)
        os.makedirs(os.path.dirname(path), exist_ok=True)
        with open(path, "w", encoding="utf-8") as f:
            f.write(content)
        return path


class TestTemplateCandidates(_TmpWsTestCase):
    def test_candidates_priority_order(self):
        cands = prompts.template_candidates("adjust", self.ws)
        self.assertEqual(cands[0], os.path.join(self.ws, "prompts", "adjust.md"))
        self.assertEqual(cands[1], os.path.join(self.ws, "env", "prompts", "adjust.md"))
        self.assertEqual(cands[2], os.path.join(DEFAULT_TEMPLATE_DIR, "adjust.md"))

    def test_find_template_missing_raises_with_all_paths(self):
        with self.assertRaises(FileNotFoundError) as cm:
            prompts.find_template("no_such_role", self.ws)
        msg = str(cm.exception)
        # 报错信息列出全部三条查找路径（排障友好）
        self.assertIn(os.path.join(self.ws, "prompts"), msg)
        self.assertIn(os.path.join(self.ws, "env", "prompts"), msg)
        self.assertIn(DEFAULT_TEMPLATE_DIR, msg)


class TestLookupChain(_TmpWsTestCase):
    def test_ws_override_wins_over_env_and_default(self):
        self._write("prompts/adjust.md", "WS_OVERRIDE")
        self._write("env/prompts/adjust.md", "ENV_OVERRIDE")
        self.assertEqual(prompts.render("adjust", self.ws, {}), "WS_OVERRIDE")

    def test_env_override_beats_default(self):
        self._write("env/prompts/adjust.md", "ENV_OVERRIDE")
        self.assertEqual(prompts.render("adjust", self.ws, {}), "ENV_OVERRIDE")

    def test_fallback_to_repo_default(self):
        out = prompts.render("adjust", self.ws, {})
        self.assertIn("角色使命", out)
        self.assertIn("proposals", out)

    def test_env_override_only_for_that_role(self):
        # env 覆盖 builder，不影响 adjust 走默认
        self._write("env/prompts/builder.md", "ENV_BUILDER")
        self.assertEqual(prompts.render("builder", self.ws, {}), "ENV_BUILDER")
        self.assertIn("角色使命", prompts.render("adjust", self.ws, {}))


class TestPlaceholderReplacement(_TmpWsTestCase):
    def test_all_placeholders_in_default_templates_replaceable(self):
        """自动发现四默认模板的占位符全集，逐一给值后渲染必须零 {{ 残留。"""
        keys = set()
        for role in ROLES:
            path = os.path.join(DEFAULT_TEMPLATE_DIR, role + ".md")
            with open(path, encoding="utf-8") as f:
                keys |= set(prompts.extract_placeholders(f.read()))
        self.assertTrue(keys, "默认模板应使用占位符")
        ctx = dict((k, _probe_value(k)) for k in keys)
        for role in ROLES:
            out = prompts.render(role, self.ws, ctx)
            self.assertNotIn("{{", out,
                             "%s 模板存在未替换占位符" % role)

    def test_probe_values_actually_reach_output(self):
        out = prompts.render("adjust", self.ws, {"task": "probe-task"})
        self.assertIn("probe-task", out)

    def test_missing_ctx_keeps_placeholder_verbatim(self):
        """缺占位数据 → 占位符原样保留，不抛异常（宽松替换）。"""
        out = prompts.render("coding", self.ws, {})
        self.assertIn("{{experiment_plan}}", out)
        self.assertIn("{{", out)

    def test_none_value_keeps_placeholder(self):
        out = prompts.apply_placeholders(
            "x={{task}} y={{absent}}",
            {"task": None})
        self.assertEqual(out, "x={{task}} y={{absent}}")

    def test_value_formatting_str_vs_json(self):
        out = prompts.apply_placeholders(
            "s=[{{s}}] i=[{{i}}] f=[{{f}}] b=[{{b}}] d=[{{d}}] l=[{{l}}]",
            {"s": "文本", "i": 5, "f": 1.5, "b": True,
             "d": {"k": "值"}, "l": [1, "a"]})
        self.assertEqual(
            out,
            's=[文本] i=[5] f=[1.5] b=[true] d=[{"k": "值"}] l=[[1, "a"]]')

    def test_json_braces_in_template_not_confused_as_placeholder(self):
        """模板中的 JSON 示例大括号不参与替换（自定义 {{key}} 语义的初衷）。"""
        tpl = '输出示例：{"a": {"b": 1}} 与 {{real_key}}'
        out = prompts.apply_placeholders(tpl, {"real_key": "OK"})
        self.assertEqual(out, '输出示例：{"a": {"b": 1}} 与 OK')

    def test_extract_placeholders_dedup_in_order(self):
        self.assertEqual(
            prompts.extract_placeholders("{{b}} {{a}} {{b}} {{c}}"),
            ["b", "a", "c"])


class TestRenderSemantics(_TmpWsTestCase):
    def test_debug_attempt_placeholders_rendered(self):
        out = prompts.render("debug", self.ws,
                             {"attempt": 2, "max_attempts": 3})
        self.assertIn("第 2/3 次", out)
        self.assertIn('"attempts": 2', out)

    def test_render_with_none_ctx(self):
        out = prompts.render("builder", self.ws, None)
        self.assertIn("{{cell_id}}", out)  # 宽松：None 视为空 ctx


class TestWritePrompt(unittest.TestCase):
    def test_write_prompt_creates_file_and_returns_path(self):
        with tempfile.TemporaryDirectory() as td:
            wd = os.path.join(td, "codes", "uuid-1")
            path = prompts.write_prompt(wd, "coding", "CONTENT")
            self.assertEqual(path, os.path.join(wd, "prompt_coding.md"))
            self.assertTrue(os.path.isfile(path))
            with open(path, encoding="utf-8") as f:
                self.assertEqual(f.read(), "CONTENT")

    def test_write_prompt_overwrites_idempotently(self):
        with tempfile.TemporaryDirectory() as td:
            wd = os.path.join(td, "codes", "uuid-2")
            prompts.write_prompt(wd, "adjust", "V1")
            path = prompts.write_prompt(wd, "adjust", "V2")
            with open(path, encoding="utf-8") as f:
                self.assertEqual(f.read(), "V2")


class TestRoleValidation(_TmpWsTestCase):
    def test_bad_role_name_raises_valueerror(self):
        for bad in ("../etc/passwd", "With Space", "UPPER", "", "9start",
                    "slash/role"):
            with self.assertRaises(ValueError):
                prompts.render(bad, self.ws, {})

    def test_unknown_role_raises_filenotfound(self):
        # 名字形状合法但三级链均无模板
        with self.assertRaises(FileNotFoundError):
            prompts.render("no_such_role_template", self.ws, {})


class TestDefaultTemplateContracts(unittest.TestCase):
    """四默认模板内容契约（与 gate2 断言同源；模板通用性红线）。"""

    def _tpl(self, role):
        path = os.path.join(DEFAULT_TEMPLATE_DIR, role + ".md")
        self.assertTrue(os.path.isfile(path), "缺默认模板 %s" % path)
        with open(path, encoding="utf-8") as f:
            return f.read()

    def test_roles_constant(self):
        self.assertEqual(ROLES, ("adjust", "builder", "coding", "debug"))

    def test_adjust_keywords(self):
        s = self._tpl("adjust")
        for kw in ("proposals", "soft_weight", "hard_delete_cell",
                   "hard_modify_cell", "hard_modify_gate",
                   "软动作", "硬动作", "正交", "不删格"):
            self.assertIn(kw, s)

    def test_builder_keywords(self):
        s = self._tpl("builder")
        for kw in ("experiment_plan", "parents", "idea", "plan_steps",
                   "expected_behavior", "target_cell"):
            self.assertIn(kw, s)

    def test_coding_keywords(self):
        s = self._tpl("coding")
        for kw in ("diff_path", "self_check", "框架", "作弊", "预存",
                   "篡改计时", "降精度"):
            self.assertIn(kw, s)

    def test_debug_keywords(self):
        s = self._tpl("debug")
        for kw in ("fix_report", "root_cause", "attempt", "final_self_check",
                   "failure_stage"):
            self.assertIn(kw, s)

    def test_output_contract_section_in_all(self):
        for role in ROLES:
            s = self._tpl(role)
            self.assertIn("输出契约", s, "%s 缺输出契约段" % role)
            self.assertIn("```json", s, "%s 缺 JSON 代码块" % role)
            # 六字段基座声明（与 task5 AgentResult 提取对齐）
            for field in ("valid", "summary"):
                self.assertIn(field, s)

    def test_no_matmul_specifics(self):
        """默认模板必须通用：不得写死任务域细节（env 上下文经占位符注入）。"""
        for role in ROLES:
            s = self._tpl(role)
            for bad in ("matmul", "numpy", "512", "float32", "strassen"):
                self.assertNotIn(bad, s,
                                 "%s 模板含任务域特定内容 %r" % (role, bad))


if __name__ == "__main__":
    unittest.main()
