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

    def test_push_returns_path_and_writes_entry(self):
        path = self.q.push("pause", {"text": "暂停观察", "category": "pause"})
        self.assertTrue(os.path.isabs(path), "push 应返回绝对路径")
        self.assertTrue(path.startswith(os.path.join(self.ws, "interventions")))
        self.assertTrue(os.path.exists(path))
        with open(path, encoding="utf-8") as f:
            entry = json.load(f)
        # 与 Go 侧 SuperEventIn 同协议：type / payload / created_at
        self.assertEqual(entry["type"], "pause")
        self.assertEqual(entry["payload"], {"text": "暂停观察", "category": "pause"})
        self.assertIn("created_at", entry)
        self.assertIn("T", entry["created_at"])  # RFC3339 形态

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
        # drain 后队列目录清空（转交即删）
        self.assertEqual(os.listdir(os.path.join(self.ws, "interventions")), [])
        # 再 drain 无残余（幂等）
        self.assertEqual(self.q.drain(), [])

    def test_drain_orders_by_filename(self):
        """文件名（时间戳）排序 = 摄取顺序，与写入先后无关。"""
        idir = os.path.join(self.ws, "interventions")
        # 逆序手写三个文件（模拟 Go 侧写入/乱序落盘）
        for name, etype in (("20260101T000003-000001.json", "third"),
                            ("20260101T000001-000001.json", "first"),
                            ("20260101T000002-000001.json", "second")):
            with open(os.path.join(idir, name), "w", encoding="utf-8") as f:
                json.dump({"type": etype, "payload": {}, "created_at": "2026-01-01T00:00:00+08:00"}, f)
        items = self.q.drain()
        self.assertEqual([i["type"] for i in items], ["first", "second", "third"])
        self.assertEqual(os.listdir(idir), [])

    def test_drain_empty_or_missing_dir(self):
        self.assertEqual(self.q.drain(), [])
        # 目录整个不存在（异常工作区）也不炸
        q2 = InterventionQueue(os.path.join(self._td.name, "no-such-ws"))
        self.assertEqual(q2.drain(), [])

    def test_drain_skips_non_json_files(self):
        """非 .json 后缀文件不动（目录里的杂物/临时文件不是干预）。"""
        idir = os.path.join(self.ws, "interventions")
        with open(os.path.join(idir, "20260101T000001-000001.json.tmp"), "w") as f:
            f.write("garbage")
        with open(os.path.join(idir, "README.md"), "w") as f:
            f.write("not an intervention")
        items = self.q.drain()
        self.assertEqual(items, [])
        # 非 .json 文件保留
        self.assertEqual(sorted(os.listdir(idir)),
                         ["20260101T000001-000001.json.tmp", "README.md"])


class TestBadJsonTolerance(unittest.TestCase):
    def setUp(self):
        self._td = tempfile.TemporaryDirectory()
        self.ws = _make_ws(self._td.name)
        self.q = InterventionQueue(self.ws)

    def tearDown(self):
        self._td.cleanup()

    def test_bad_json_skipped_and_recorded(self):
        idir = os.path.join(self.ws, "interventions")
        with open(os.path.join(idir, "20260101T000001-000001.json"), "w") as f:
            f.write("{not valid json")
        self.q.push("stop", {"text": "正常", "category": "stop"})  # 命名晚于坏文件
        items = self.q.drain()
        # 好条目照常摄取
        self.assertEqual([i["type"] for i in items if i["type"] != "bad_intervention"],
                         ["stop"])
        # 坏 JSON 以 bad_intervention 条目留痕
        bad = [i for i in items if i["type"] == "bad_intervention"]
        self.assertEqual(len(bad), 1)
        self.assertIn("file", bad[0])
        self.assertIn("20260101T000001-000001.json", bad[0]["file"])
        self.assertIn("error", bad[0])
        # 坏文件同样被转交删除（drain 后清场）
        self.assertEqual(os.listdir(idir), [])

    def test_non_object_json_is_bad(self):
        """合法 JSON 但顶层非对象（数组/字符串）→ bad_intervention。"""
        idir = os.path.join(self.ws, "interventions")
        with open(os.path.join(idir, "20260101T000001-000001.json"), "w") as f:
            json.dump(["not", "an", "object"], f)
        items = self.q.drain()
        self.assertEqual(len(items), 1)
        self.assertEqual(items[0]["type"], "bad_intervention")


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
