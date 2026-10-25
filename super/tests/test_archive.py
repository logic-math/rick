"""test_archive.py — job_47 task2 单元测试

Archive（精英格档案）行为契约：
  - admit：新格入档 / 同格更优替换（perf_desc 方向感知）/ 同格更差拒绝
  - history：替换时旧精英入 history，上限 20（最旧被挤出）
  - pending：add/drain 往返
  - set_weight：降 0 不删格（功能性灭绝）
  - load/save 往返一致；save 原子（失败不破坏旧档）
  - cells 访问器返回副本（外部改不影响内部）
"""
import copy
import json
import os
import tempfile
import unittest
from unittest import mock

from super.superopt.archive import Archive


def entry(perf_value, code_path="codes/x/", gen=1, score=0.5):
    """构造一个合法的 admit entry（descriptor + elite 六字段）。"""
    return {
        "descriptor": {"impl_strategy": "naive_loops", "memory_layout": "row_major"},
        "elite": {
            "code_path": code_path,
            "perf_value": perf_value,
            "perf_type": "runtime_ms",
            "perf_desc": "low",
            "score": score,
            "lineage": [],
            "confidence": 0.8,
            "admitted_at": "2026-10-17T10:00:00+08:00",
            "gen": gen,
        },
    }


class TestAdmitBasics(unittest.TestCase):
    def setUp(self):
        self.tmp = tempfile.TemporaryDirectory()
        self.path = os.path.join(self.tmp.name, "archive.json")

    def tearDown(self):
        self.tmp.cleanup()

    def test_admit_new_cell_succeeds_and_creates_cell_defaults(self):
        a = Archive(self.path)
        a.load()
        ok = a.admit("cell_a", entry(6.0))
        self.assertTrue(ok)
        el = a.get_elite("cell_a")
        self.assertIsNotNone(el)
        self.assertEqual(el["perf_value"], 6.0)
        cell = a.cells["cell_a"]
        self.assertEqual(cell["weight"], 1.0)
        self.assertEqual(cell["visit"], 0)
        self.assertEqual(cell["sample_count"], 0)
        self.assertEqual(cell["history"], [])
        self.assertEqual(cell["descriptor"]["impl_strategy"], "naive_loops")

    def test_admit_missing_descriptor_stores_empty(self):
        a = Archive(self.path)
        a.load()
        e = entry(6.0)
        del e["descriptor"]
        ok = a.admit("cell_b", e)
        self.assertTrue(ok)
        self.assertEqual(a.cells["cell_b"]["descriptor"], {})

    def test_admit_no_prior_elite_always_succeeds(self):
        # 格存在但 elite 为空（如 invalidation 出档后）——视同空格，必入
        a = Archive(self.path)
        a.load()
        a.admit("cell_c", entry(9.0))
        a.cells  # cells 内部
        a._data["cells"]["cell_c"]["elite"] = None
        ok = a.admit("cell_c", entry(3.0))
        self.assertTrue(ok)
        self.assertEqual(a.get_elite("cell_c")["perf_value"], 3.0)


class TestAdmitDirection(unittest.TestCase):
    def setUp(self):
        self.tmp = tempfile.TemporaryDirectory()
        self.path = os.path.join(self.tmp.name, "archive.json")

    def tearDown(self):
        self.tmp.cleanup()

    def test_low_direction_smaller_replaces(self):
        a = Archive(self.path, perf_desc="low")
        a.load()
        a.admit("c", entry(6.0))
        ok = a.admit("c", entry(5.0))
        self.assertTrue(ok)
        self.assertEqual(a.get_elite("c")["perf_value"], 5.0)

    def test_low_direction_larger_rejected(self):
        a = Archive(self.path, perf_desc="low")
        a.load()
        a.admit("c", entry(5.0))
        ok = a.admit("c", entry(6.0))
        self.assertFalse(ok)
        self.assertEqual(a.get_elite("c")["perf_value"], 5.0)

    def test_high_direction_larger_replaces(self):
        a = Archive(self.path, perf_desc="high")
        a.load()
        a.admit("c", entry(6.0))
        ok = a.admit("c", entry(7.0))
        self.assertTrue(ok)
        self.assertEqual(a.get_elite("c")["perf_value"], 7.0)

    def test_high_direction_smaller_rejected(self):
        a = Archive(self.path, perf_desc="high")
        a.load()
        a.admit("c", entry(7.0))
        ok = a.admit("c", entry(6.0))
        self.assertFalse(ok)

    def test_equal_perf_not_replaced(self):
        a = Archive(self.path, perf_desc="low")
        a.load()
        a.admit("c", entry(5.0, code_path="codes/first/"))
        ok = a.admit("c", entry(5.0, code_path="codes/second/"))
        self.assertFalse(ok)
        self.assertEqual(a.get_elite("c")["code_path"], "codes/first/")

    def test_perf_desc_persisted_in_meta(self):
        # 方向读 env 的 perf_desc 并存入 archive meta
        a = Archive(self.path, perf_desc="low")
        a.load()
        a.admit("c", entry(6.0))
        a.save()
        with open(self.path, encoding="utf-8") as f:
            data = json.load(f)
        self.assertEqual(data["meta"]["perf_desc"], "low")


class TestHistory(unittest.TestCase):
    def setUp(self):
        self.tmp = tempfile.TemporaryDirectory()
        self.path = os.path.join(self.tmp.name, "archive.json")

    def tearDown(self):
        self.tmp.cleanup()

    def test_replace_appends_old_elite_to_history(self):
        a = Archive(self.path, perf_desc="low")
        a.load()
        a.admit("c", entry(6.0, code_path="codes/gen1/"))
        a.admit("c", entry(5.0, code_path="codes/gen2/"))
        hist = a.cells["c"]["history"]
        self.assertEqual(len(hist), 1)
        self.assertEqual(hist[0]["code_path"], "codes/gen1/")
        self.assertEqual(a.get_elite("c")["code_path"], "codes/gen2/")

    def test_history_capped_at_20_oldest_dropped(self):
        a = Archive(self.path, perf_desc="low")
        a.load()
        for i in range(25):
            a.admit("c", entry(float(100 - i), code_path="codes/gen%d/" % i))
        hist = a.cells["c"]["history"]
        # 24 次替换 → 24 条 history，上限 20 → 最旧 4 条（gen0..gen3）被挤出
        self.assertEqual(len(hist), 20)
        self.assertEqual(hist[0]["code_path"], "codes/gen4/")
        self.assertEqual(hist[-1]["code_path"], "codes/gen23/")
        # 当前精英是最后一次 admit（gen24，值最小）
        self.assertEqual(a.get_elite("c")["code_path"], "codes/gen24/")


class TestPending(unittest.TestCase):
    def setUp(self):
        self.tmp = tempfile.TemporaryDirectory()
        self.path = os.path.join(self.tmp.name, "archive.json")

    def tearDown(self):
        self.tmp.cleanup()

    def test_add_and_drain_roundtrip_ordered(self):
        a = Archive(self.path)
        a.load()
        a.add_pending("c1", "codes/u1/", "new_cell")
        a.add_pending("c2", "codes/u2/", "tolerance_band")
        a.add_pending("c3", "codes/u3/", "low_weight_src")
        got = a.drain_pending()
        self.assertEqual([p["cell_id"] for p in got], ["c1", "c2", "c3"])
        self.assertEqual(got[0]["reason"], "new_cell")
        self.assertEqual(got[0]["code_path"], "codes/u1/")
        self.assertIn("added_at", got[0])
        # drain 后清空
        self.assertEqual(a.drain_pending(), [])

    def test_drain_does_not_require_save_to_empty_memory(self):
        # drain 是取走语义（内存队列立即清空；持久化由调用方 save 决定）
        a = Archive(self.path)
        a.load()
        a.add_pending("c1", "codes/u1/", "new_cell")
        a.drain_pending()
        self.assertEqual(a._data["pending"], [])

    def test_pending_persisted_and_restored(self):
        a = Archive(self.path)
        a.load()
        a.add_pending("c1", "codes/u1/", "new_cell")
        a.save()
        b = Archive(self.path)
        b.load()
        got = b.drain_pending()
        self.assertEqual(len(got), 1)
        self.assertEqual(got[0]["cell_id"], "c1")


class TestWeight(unittest.TestCase):
    def setUp(self):
        self.tmp = tempfile.TemporaryDirectory()
        self.path = os.path.join(self.tmp.name, "archive.json")

    def tearDown(self):
        self.tmp.cleanup()

    def test_set_weight_zero_cell_remains(self):
        # 降 0 = 功能性灭绝，不删格
        a = Archive(self.path)
        a.load()
        a.admit("c", entry(6.0))
        a.set_weight("c", 0.0)
        self.assertIn("c", a.cells)
        self.assertEqual(a.cells["c"]["weight"], 0.0)
        # 精英仍在
        self.assertEqual(a.get_elite("c")["perf_value"], 6.0)

    def test_set_weight_unknown_cell_creates_cell(self):
        # adjust 可对未入档的新格预设权重（格骨架即存在）
        a = Archive(self.path)
        a.load()
        a.set_weight("ghost", 0.5)
        self.assertEqual(a.cells["ghost"]["weight"], 0.5)
        self.assertIsNone(a.cells["ghost"]["elite"])
        self.assertEqual(a.cells["ghost"]["visit"], 0)


class TestPersistence(unittest.TestCase):
    def setUp(self):
        self.tmp = tempfile.TemporaryDirectory()
        self.path = os.path.join(self.tmp.name, "archive.json")

    def tearDown(self):
        self.tmp.cleanup()

    def test_load_save_roundtrip(self):
        a = Archive(self.path, perf_desc="low")
        a.load()
        a.admit("c1", entry(6.0, gen=1))
        a.admit("c2", entry(3.2, gen=2))
        a.admit("c1", entry(5.0, gen=2))
        a.add_pending("c2", "codes/p1/", "tolerance_band")
        a.set_weight("c2", 0.4)
        a.save()
        b = Archive(self.path, perf_desc="low")
        b.load()
        self.assertEqual(b.get_elite("c1")["perf_value"], 5.0)
        self.assertEqual(b.cells["c1"]["history"][0]["perf_value"], 6.0)
        self.assertEqual(b.cells["c2"]["weight"], 0.4)
        self.assertEqual(len(b.drain_pending()), 1)
        self.assertIn("updated_at", b._data["meta"])

    def test_load_missing_file_starts_empty(self):
        a = Archive(self.path)
        a.load()
        self.assertEqual(a.cells, {})
        self.assertEqual(a.drain_pending(), [])

    def test_load_corrupt_file_raises_or_starts_empty(self):
        # 档案是单写者真相源的投影——损坏文件不应静默当作空档（宁可炸）
        with open(self.path, "w", encoding="utf-8") as f:
            f.write("{not json")
        a = Archive(self.path)
        with self.assertRaises(json.JSONDecodeError):
            a.load()

    def test_save_atomic_failure_leaves_old_file_intact(self):
        a = Archive(self.path, perf_desc="low")
        a.load()
        a.admit("c", entry(6.0))
        a.save()
        with open(self.path, encoding="utf-8") as f:
            before = f.read()
        # 再改内存，然后让 save 中途失败（os.replace 抛错）
        a.admit("c", entry(5.0))
        a.add_pending("c", "codes/px/", "tolerance_band")
        with mock.patch("os.replace", side_effect=OSError("disk full")):
            with self.assertRaises(OSError):
                a.save()
        # 旧档未被破坏：内容与 save 前一致，可正常 load
        with open(self.path, encoding="utf-8") as f:
            after = f.read()
        self.assertEqual(before, after)
        b = Archive(self.path, perf_desc="low")
        b.load()
        self.assertEqual(b.get_elite("c")["perf_value"], 6.0)

    def test_save_writes_no_tmp_leftover(self):
        a = Archive(self.path)
        a.load()
        a.admit("c", entry(6.0))
        a.save()
        leftovers = [f for f in os.listdir(self.tmp.name) if f != "archive.json"]
        self.assertEqual(leftovers, [])


class TestCellsAccessor(unittest.TestCase):
    def setUp(self):
        self.tmp = tempfile.TemporaryDirectory()
        self.path = os.path.join(self.tmp.name, "archive.json")

    def tearDown(self):
        self.tmp.cleanup()

    def test_cells_returns_copy(self):
        a = Archive(self.path)
        a.load()
        a.admit("c", entry(6.0))
        cells = a.cells
        cells["hacked"] = {"weight": 99}
        cells["c"]["weight"] = 99
        self.assertNotIn("hacked", a.cells)
        self.assertEqual(a.cells["c"]["weight"], 1.0)

    def test_get_elite_returns_copy(self):
        a = Archive(self.path)
        a.load()
        a.admit("c", entry(6.0))
        el = a.get_elite("c")
        el["perf_value"] = 0.0
        self.assertEqual(a.get_elite("c")["perf_value"], 6.0)

    def test_get_elite_missing_returns_none(self):
        a = Archive(self.path)
        a.load()
        self.assertIsNone(a.get_elite("nope"))


if __name__ == "__main__":
    unittest.main()


class TestCellDescription(unittest.TestCase):
    """特征格四要素之「方法描述」（迭代8 / KR2）：description 落盘与兼容。"""

    def setUp(self):
        self.tmp = tempfile.mkdtemp(prefix="archive-desc-")
        self.path = os.path.join(self.tmp, "archive.json")

    def test_new_cell_skeleton_has_empty_description(self):
        a = Archive(self.path, perf_desc="low")
        a.admit("c1", entry(1.0))
        self.assertEqual(a.cells["c1"]["description"], {})
        a.set_weight("c2", 0.5)
        self.assertEqual(a.cells["c2"]["description"], {})
        a.set_visit("c3", 1)
        self.assertEqual(a.cells["c3"]["description"], {})

    def test_set_description_roundtrip(self):
        a = Archive(self.path, perf_desc="low")
        a.set_description("c1", {
            "name": "M 维四分 + 2-D np.dot",
            "method": "把 B 的全量重读降到 4 次；见 arxiv:2108.02025",
            "source_refs": ["arxiv:2108.02025", "  ", "docs/perf.md"],
        })
        d = a.cells["c1"]["description"]
        self.assertEqual(d["name"], "M 维四分 + 2-D np.dot")
        self.assertIn("2108.02025", d["method"])
        # source_refs 规范化：去空白项、全 str
        self.assertEqual(d["source_refs"], ["arxiv:2108.02025", "docs/perf.md"])
        # 落盘往返
        a.save()
        b = Archive(self.path, perf_desc="low")
        b.load()
        self.assertEqual(b.cells["c1"]["description"]["name"], d["name"])

    def test_set_description_unknown_keys_dropped_and_scalar_refs(self):
        a = Archive(self.path, perf_desc="low")
        a.set_description("c1", {"name": "x", "method": "y",
                                 "source_refs": "arxiv:1", "extra": "nope"})
        d = a.cells["c1"]["description"]
        self.assertNotIn("extra", d)
        self.assertEqual(d["source_refs"], ["arxiv:1"])

    def test_set_description_rejects_non_dict(self):
        a = Archive(self.path, perf_desc="low")
        with self.assertRaises(ValueError):
            a.set_description("c1", "not-a-dict")

    def test_old_archive_without_description_is_backfilled(self):
        """旧档（迭代8 之前）格无 description → load 后补 {}。"""
        os.makedirs(self.tmp, exist_ok=True)
        with open(self.path, "w", encoding="utf-8") as f:
            json.dump({
                "meta": {"env": "", "feature_axes": [], "perf_desc": "low",
                         "updated_at": "2026-10-01T00:00:00+08:00"},
                "cells": {"c1": {"descriptor": {"a": "x"}, "elite": None,
                                 "history": [], "weight": 1.0, "visit": 0}},
                "pending": [],
            }, f)
        a = Archive(self.path, perf_desc="low")
        a.load()
        self.assertEqual(a.cells["c1"]["description"], {})
