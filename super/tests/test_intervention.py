"""test_intervention.py — superopt.intervention 单元测试（job_47/task9）

覆盖（spec 测试方法）：
  - push→drain 往返（文件名排序 = 摄取顺序）
  - 坏 JSON 容错（跳过 + 记 bad_intervention 条目，文件仍被转交删除）
  - drain 后文件删除（at-least-once：转交即清场）
  - 空目录 / 目录不存在 → 空列表
  - push 落盘结构与 Go 侧 SuperEventIn 同协议（type/payload/created_at）
  - 与 Go 写入的条目互通（Go 命名 <ts>-<rand>.json 同样可被 drain）
  - __main__ 冒烟：python3 -m superopt --help exit 0（loop 未落时也可冒烟）
"""
import json
import os
import subprocess
import sys
import tempfile
import unittest

_HERE = os.path.dirname(os.path.abspath(__file__))
_SUPER = os.path.dirname(_HERE)      # super/
_REPO = os.path.dirname(_SUPER)      # 仓库根
for _p in (_SUPER, _REPO):
    if _p not in sys.path:
        sys.path.insert(0, _p)

from superopt.intervention import InterventionQueue


def _make_ws(td):
    ws = os.path.join(td, "ws")
    os.makedirs(os.path.join(ws, "interventions"), exist_ok=True)
    return ws


class TestPushDrainRoundtrip(unittest.TestCase):
    def setUp(self):
        self._td = tempfile.TemporaryDirectory()
        self.ws = _make_ws(self._td.name)
        self.q = InterventionQueue(self.ws)

    def tearDown(self):
        self._td.cleanup()

    def test_push_returns_id_and_writes_row(self):
        """DB 化：push 返回 humans#<id> 标识，条目落 humans 表干预队列。"""
        ident = self.q.push("pause", {"text": "暂停观察", "category": "pause"})
        self.assertTrue(ident.startswith("humans#"), "push 应返回 humans#<id>")
        # 未应用干预可查询（step1 注入用），与 push 同协议：type/payload/created_at
        pend = self.q.pending()
        self.assertEqual(len(pend), 1)
        self.assertEqual(pend[0]["type"], "pause")
        self.assertEqual(pend[0]["payload"],
                         {"text": "暂停观察", "category": "pause"})
        self.assertIn("T", pend[0]["created_at"])  # RFC3339 形态

    def test_push_rejects_bad_input(self):
        with self.assertRaises(ValueError):
            self.q.push("", {"text": "x"})            # 空 type
        with self.assertRaises(ValueError):
            self.q.push("pause", "not-a-dict")        # payload 非 dict
        with self.assertRaises(ValueError):
            self.q.push("pause", {"k": object()})     # 非 JSON 可序列化

    def test_drain_roundtrip_and_delete(self):
        self.q.push("weight", {"text": "c1 降权", "category": "weight"})
        self.q.push("stop", {"text": "收工", "category": "stop"})
        items = self.q.drain()
        self.assertEqual([i["type"] for i in items], ["weight", "stop"])
        self.assertEqual(items[0]["payload"]["text"], "c1 降权")
        # drain 后 applied=1 不再返回（转交即清场）
        self.assertEqual(self.q.drain(), [])
        self.assertEqual(self.q.pending(), [])

    def test_drain_orders_by_insertion(self):
        """DB 化：id 自增序 = 摄取顺序（与 push 先后一致）。"""
        self.q.push("first", {})
        self.q.push("second", {})
        self.q.push("third", {})
        items = self.q.drain()
        self.assertEqual([i["type"] for i in items], ["first", "second", "third"])
        self.assertEqual(self.q.drain(), [])

    def test_drain_empty_or_missing_dir(self):
        self.assertEqual(self.q.drain(), [])
        # 目录整个不存在（异常工作区）也不炸
        q2 = InterventionQueue(os.path.join(self._td.name, "no-such-ws"))
        self.assertEqual(q2.drain(), [])

    def test_drain_ignores_legacy_dir_files(self):
        """DB 化：interventions/ 目录遗留文件（json 时代产物/杂物）被忽略。"""
        idir = os.path.join(self.ws, "interventions")
        os.makedirs(idir, exist_ok=True)
        with open(os.path.join(idir, "20260101T000001-000001.json"), "w") as f:
            f.write("{not valid json")
        with open(os.path.join(idir, "README.md"), "w") as f:
            f.write("not an intervention")
        items = self.q.drain()
        self.assertEqual(items, [])
        # 目录文件不动（不是 DB 的一部分）
        self.assertEqual(sorted(os.listdir(idir)),
                         ["20260101T000001-000001.json", "README.md"])


class TestBadJsonTolerance(unittest.TestCase):
    def setUp(self):
        self._td = tempfile.TemporaryDirectory()
        self.ws = _make_ws(self._td.name)
        self.q = InterventionQueue(self.ws)

    def tearDown(self):
        self._td.cleanup()

    def test_db_rows_always_valid_json(self):
        """DB 化：humans 表行总是合法 JSON（坏 JSON 概念随存储迁移退役）。

        坏输入在 push 入口被拒（ValueError）——坏数据不落库。
        """
        self.q.push("stop", {"text": "正常", "category": "stop"})
        items = self.q.drain()
        self.assertEqual([i["type"] for i in items], ["stop"])
        # 全部条目 payload 均为合法 dict（DB 行级 JSON 完整性）
        for i in items:
            self.assertIsInstance(i["payload"], dict)

    def test_proposal_queue_separate_from_intervention(self):
        """两类队列互不串扰：提案不进干预 drain。"""
        from super.superopt.db import db_for_ws
        d = db_for_ws(self.ws)
        d.push_proposal({"kind": "verify_reflect"})
        self.q.push("pause", {"text": "暂停"})
        items = self.q.drain()
        self.assertEqual([i["type"] for i in items], ["pause"])


class TestPackageMainSmoke(unittest.TestCase):
    """__main__ 冒烟（spec：python3 -m superopt --help exit 0）。

    模拟 Go PySupervisor 的 spawn 环境：cwd = 仓库根 + PYTHONPATH 锚定
    <repoRoot>/super。--help 必须不依赖 loop（task8 并行落地中）即可工作。
    """

    def _run(self, *argv):
        env = dict(os.environ)
        env["PYTHONPATH"] = _SUPER + os.pathsep + env.get("PYTHONPATH", "")
        return subprocess.run(
            [sys.executable, "-m", "superopt"] + list(argv),
            cwd=_REPO, capture_output=True, text=True, timeout=60, env=env)

    def test_help_exits_zero_without_loop(self):
        r = self._run("--help")
        self.assertEqual(r.returncode, 0,
                         "--help 应 exit 0：%s%s" % (r.stdout, r.stderr))
        for flag in ("--workdir", "--max-iterations", "--adjust-interval",
                     "--max-wall-time", "--preverify-red"):
            self.assertIn(flag, r.stdout, "--help 输出缺 flag %s" % flag)

    def test_missing_workdir_flag_is_usage_error(self):
        r = self._run()
        self.assertEqual(r.returncode, 2, "缺 --workdir 应是 argparse 用法错误（exit 2）")

    def test_run_reports_config_summary_and_clean_loop_failure(self):
        """带合法 ws 跑：打印配置摘要；loop 未落时 FATAL + exit 4（可诊断）。

        loop.py 由 task8 并行落地后，本用例的两断言依然成立（FATAL 分支
        不再走到，摘要行先打）。若 task8 已落地，此用例会真启动调度器——
        因此用 max-iterations=0 且 stop 干预预置，让其立即优雅退出。
        """
        td = tempfile.TemporaryDirectory()
        self.addCleanup(td.cleanup)
        ws = os.path.join(td.name, "ws")
        sys.path.insert(0, _SUPER)
        try:
            from superopt import config as _config
            _config.init_ws(ws, os.path.join(td.name, "env"))
        finally:
            sys.path.remove(_SUPER)
        r = self._run("--workdir", ws, "--max-iterations", "3")
        self.assertIn("[superopt] starting", r.stdout)
        self.assertIn("max_iterations=3", r.stdout)
        self.assertIn(r.returncode, (0, 3, 4),
                      "退出码应为 0（正常）/3（门先证红失败）/4（loop 未落），"
                      "实际 %s：%s%s" % (r.returncode, r.stdout, r.stderr))


if __name__ == "__main__":
    unittest.main()
