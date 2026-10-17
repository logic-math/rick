"""test_config.py — superopt.config 单元测试（job_47/task4）

覆盖：init_ws 目录骨架与 super.json 初始化、幂等语义、load 缺键补 DEFAULTS、
merge_cli 覆盖语义（非 None 才覆盖；--max-wall-time 同步 budget 段）、
文件缺失兜底、坏 JSON 报错。
"""
import json
import os
import sys
import tempfile
import unittest
from argparse import Namespace

_HERE = os.path.dirname(os.path.abspath(__file__))
_SUPER = os.path.dirname(_HERE)      # super/
_REPO = os.path.dirname(_SUPER)      # 仓库根
for _p in (_SUPER, _REPO):
    if _p not in sys.path:
        sys.path.insert(0, _p)

from superopt import config
from superopt.config import DEFAULTS


class TestDefaults(unittest.TestCase):
    def test_defaults_values_match_go_side_contract(self):
        """与 Go 侧 task11 对齐的默认值（spec 硬约束）。"""
        self.assertEqual(DEFAULTS["adjust_interval"], 5)
        self.assertEqual(DEFAULTS["debug_max_attempts"], 3)
        self.assertEqual(DEFAULTS["soft_gate_k"], 3)
        self.assertEqual(DEFAULTS["heartbeat_interval_s"], 30)
        self.assertEqual(DEFAULTS["heartbeat_timeout_s"], 90)
        self.assertEqual(DEFAULTS["term_grace_s"], 5)
        self.assertEqual(DEFAULTS["max_wall_time_h"], 24.0)
        self.assertEqual(DEFAULTS["max_iterations"], 0)
        self.assertEqual(DEFAULTS["python_path"], "python3")
        self.assertEqual(DEFAULTS["pi_path"], "")


class TestInitWs(unittest.TestCase):
    def setUp(self):
        self._td = tempfile.TemporaryDirectory()
        self.ws = os.path.join(self._td.name, "ws")
        os.makedirs(self.ws)
        self.env = os.path.join(self._td.name, "env")

    def tearDown(self):
        self._td.cleanup()

    def test_init_ws_creates_skeleton(self):
        cfg = config.init_ws(self.ws, self.env)
        for d in ("interventions", "codes", "prompts"):
            self.assertTrue(os.path.isdir(os.path.join(self.ws, d)),
                            f"缺目录 {d}")
        self.assertTrue(os.path.exists(os.path.join(self.ws, "events.jsonl")))
        self.assertEqual(os.path.getsize(os.path.join(self.ws, "events.jsonl")), 0,
                         "events.jsonl 应为空文件")
        arch = json.load(open(os.path.join(self.ws, "archive.json")))
        self.assertEqual(arch, {"cells": {}, "pending": []},
                         "archive.json 初始结构")

    def test_init_ws_super_json_fields(self):
        cfg = config.init_ws(self.ws, self.env)
        self.assertEqual(cfg["state"], "idle")
        self.assertEqual(cfg["iteration_cursor"], 0)
        self.assertEqual(cfg["env_path"], self.env)
        self.assertIsNone(cfg["pid"])
        self.assertEqual(cfg["budget"]["wall_start"], None)
        self.assertEqual(cfg["budget"]["max_wall_time_h"], 24.0)
        self.assertEqual(cfg["config"]["adjust_interval"], 5)
        self.assertEqual(cfg["config"]["max_iterations"], 0)
        # heartbeat_path 指向工作区内
        self.assertEqual(cfg["heartbeat_path"],
                         os.path.join(self.ws, "heartbeat.json"))
        # 落盘内容 == 返回内容
        on_disk = json.load(open(os.path.join(self.ws, "super.json")))
        self.assertEqual(on_disk, cfg)
        # 时间戳带时区（bugs.md：RFC3339 完整格式）
        self.assertIn("+", cfg["updated_at"])

    def test_init_ws_idempotent_keeps_state(self):
        config.init_ws(self.ws, self.env)
        # 模拟运行后状态被改
        p = os.path.join(self.ws, "super.json")
        d = json.load(open(p))
        d["state"] = "running"
        d["iteration_cursor"] = 7
        json.dump(d, open(p, "w"))
        cfg2 = config.init_ws(self.ws, self.env)
        self.assertEqual(cfg2["state"], "running", "幂等：不重置已运行状态")
        self.assertEqual(cfg2["iteration_cursor"], 7)
        self.assertEqual(cfg2["env_path"], self.env)


class TestLoad(unittest.TestCase):
    def setUp(self):
        self._td = tempfile.TemporaryDirectory()
        self.ws = os.path.join(self._td.name, "ws")
        os.makedirs(self.ws)

    def tearDown(self):
        self._td.cleanup()

    def test_load_missing_file_returns_defaults(self):
        cfg = config.load(self.ws)
        self.assertEqual(cfg["state"], None)
        self.assertEqual(cfg["iteration_cursor"], 0)
        self.assertEqual(cfg["config"], dict(DEFAULTS))
        self.assertEqual(cfg["budget"]["max_wall_time_h"], 24.0)

    def test_load_roundtrip_and_fill_missing_keys(self):
        config.init_ws(self.ws, "/tmp/env")
        p = os.path.join(self.ws, "super.json")
        d = json.load(open(p))
        # 抽掉一个 config 键 → load 应补 DEFAULTS（内存补全，不写回）
        del d["config"]["adjust_interval"]
        json.dump(d, open(p, "w"))
        cfg = config.load(self.ws)
        self.assertEqual(cfg["config"]["adjust_interval"], 5)
        # 不写回：磁盘上仍缺
        d2 = json.load(open(p))
        self.assertNotIn("adjust_interval", d2["config"])

    def test_load_bad_json_raises(self):
        with open(os.path.join(self.ws, "super.json"), "w") as f:
            f.write("{not json")
        with self.assertRaises(ValueError):
            config.load(self.ws)


class TestMergeCli(unittest.TestCase):
    def _base(self):
        return config.init_ws(tempfile.mkdtemp(), "/tmp/env")

    def test_merge_overrides_only_non_none(self):
        cfg = self._base()
        args = Namespace(max_iterations=3, adjust_interval=None,
                         max_wall_time=None)
        out = config.merge_cli(cfg, args)
        self.assertEqual(out["config"]["max_iterations"], 3)
        self.assertEqual(out["config"]["adjust_interval"], 5, "None 不覆盖")
        # 原对象不被就地修改（返回新 dict）
        self.assertEqual(cfg["config"]["max_iterations"], 0)

    def test_merge_max_wall_time_syncs_budget(self):
        cfg = self._base()
        args = Namespace(max_iterations=None, adjust_interval=None,
                         max_wall_time=2.5)
        out = config.merge_cli(cfg, args)
        self.assertEqual(out["config"]["max_wall_time_h"], 2.5)
        self.assertEqual(out["budget"]["max_wall_time_h"], 2.5,
                         "预算参数覆盖须同步 budget 段（两层一致）")

    def test_merge_ignores_unknown_keys(self):
        cfg = self._base()
        args = Namespace(max_iterations=1, adjust_interval=None,
                         max_wall_time=None, something_else="x")
        out = config.merge_cli(cfg, args)  # 不应抛错
        self.assertNotIn("something_else", out["config"])

    def test_merge_accepts_dict(self):
        """merge_cli 对 dict 形式的 args 同样可用（可测性）。"""
        cfg = self._base()
        out = config.merge_cli(cfg, {"max_iterations": 9})
        self.assertEqual(out["config"]["max_iterations"], 9)


if __name__ == "__main__":
    unittest.main()
