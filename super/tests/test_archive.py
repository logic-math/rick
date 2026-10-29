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


class TestFiveElements(unittest.TestCase):
    """task44：特征格五要素——elite_code_path / elite_record_path 与并发安全。"""

    def setUp(self):
        self.tmp = tempfile.mkdtemp(prefix="archive-five-")
        self.path = os.path.join(self.tmp, "archive.json")

    def tearDown(self):
        import shutil
        shutil.rmtree(self.tmp, ignore_errors=True)

    def test_admit_syncs_elite_code_path_and_record_path(self):
        a = Archive(self.path, perf_desc="low")
        rec = os.path.join(self.tmp, "codes", "iter001-x", "experiment_record.md")
        ok = a.admit("c1", entry(5.0, code_path="codes/iter001-x/solution.py"),
                     record_path=rec)
        self.assertTrue(ok)
        cell = a.cells["c1"]
        self.assertEqual(cell["elite_code_path"], "codes/iter001-x/solution.py")
        self.assertEqual(cell["elite_record_path"], rec)
        self.assertEqual(a.get_record_path("c1"), rec)
        # 副本安全：外部改副本不影响内部
        cell["elite_record_path"] = "hacked"
        self.assertEqual(a.get_record_path("c1"), rec)

    def test_replace_updates_both_paths(self):
        a = Archive(self.path, perf_desc="low")
        a.admit("c1", entry(9.0, code_path="codes/iter001-x/s"))
        a.admit("c1", entry(5.0, code_path="codes/iter002-y/s"),
                record_path="/w/codes/iter002-y/experiment_record.md")
        cell = a.cells["c1"]
        self.assertEqual(cell["elite_code_path"], "codes/iter002-y/s")
        self.assertEqual(cell["elite_record_path"],
                         "/w/codes/iter002-y/experiment_record.md")

    def test_admit_without_record_path_keeps_none(self):
        a = Archive(self.path, perf_desc="low")
        a.admit("c1", entry(5.0, code_path="codes/iter003-z/s"))
        cell = a.cells["c1"]
        self.assertEqual(cell["elite_code_path"], "codes/iter003-z/s")
        self.assertIsNone(cell["elite_record_path"])
        self.assertIsNone(a.get_record_path("c1"))

    def test_get_record_path_unknown_cell_none(self):
        a = Archive(self.path, perf_desc="low")
        self.assertIsNone(a.get_record_path("nope"))

    def test_old_archive_without_five_elements_backfilled(self):
        """旧档格无 elite_code_path/elite_record_path → load 后补 None。"""
        os.makedirs(self.tmp, exist_ok=True)
        with open(self.path, "w", encoding="utf-8") as f:
            json.dump({
                "meta": {"env": "", "feature_axes": [], "perf_desc": "low",
                         "updated_at": "2026-10-01T00:00:00+08:00"},
                "cells": {"c1": {"descriptor": {"a": "x"},
                                 "elite": {"code_path": "codes/old/solution.py",
                                           "perf_value": 5.0},
                                 "history": [], "weight": 1.0, "visit": 1}},
                "pending": [],
            }, f)
        a = Archive(self.path, perf_desc="low")
        a.load()
        cell = a.cells["c1"]
        self.assertIn("elite_code_path", cell)
        self.assertIsNone(cell["elite_code_path"])
        self.assertIsNone(cell["elite_record_path"])
        self.assertIsNone(a.get_record_path("c1"))

    def test_concurrent_admits_do_not_break(self):
        """20 线程并发写不同格 + 并发读——不炸且全部落档（task45 前置）。"""
        import threading
        a = Archive(self.path, perf_desc="low")
        errs = []

        def worker(i):
            try:
                for k in range(5):
                    a.admit("c%d" % i, entry(10.0 - k, code_path="codes/c%d-%d" % (i, k)),
                            record_path="/w/c%d/%d.md" % (i, k))
                    a.set_weight("c%d" % i, 1.0 - 0.1 * k)
                    a.set_visit("c%d" % i, k)
                    _ = a.cells
                    _ = a.get_record_path("c%d" % i)
            except Exception as exc:  # noqa: BLE001
                errs.append(exc)

        threads = [threading.Thread(target=worker, args=(i,)) for i in range(20)]
        for t in threads:
            t.start()
        for t in threads:
            t.join()
        self.assertEqual(errs, [])
        cells = a.cells
        self.assertEqual(len(cells), 20)
        for i in range(20):
            self.assertEqual(cells["c%d" % i]["elite_code_path"], "codes/c%d-4" % i)
            self.assertEqual(cells["c%d" % i]["elite_record_path"], "/w/c%d/4.md" % i)

    def test_concurrent_save_and_mutate_serialized(self):
        """并发 save + mutate 串行化（不产生半写档案）。"""
        import threading
        a = Archive(self.path, perf_desc="low")
        errs = []

        def writer(i):
            try:
                for k in range(3):
                    a.admit("d%d" % i, entry(20.0 - k))
                    a.save()
            except Exception as exc:  # noqa: BLE001
                errs.append(exc)

        threads = [threading.Thread(target=writer, args=(i,)) for i in range(6)]
        for t in threads:
            t.start()
        for t in threads:
            t.join()
        self.assertEqual(errs, [])
        b = Archive(self.path, perf_desc="low")
        b.load()  # 合法 JSON（未半写）
        self.assertEqual(len(b.cells), 6)


class TestCodeHashAndStale(unittest.TestCase):
    """task58/R3+R4：elite_code_hash（同码判别锚）+ method_stale（相位差标记）。"""

    def setUp(self):
        self.tmp = tempfile.mkdtemp(prefix="archive-task58-")
        self.path = os.path.join(self.tmp, "archive.json")

    def tearDown(self):
        import shutil
        shutil.rmtree(self.tmp, ignore_errors=True)

    def test_admit_stores_code_hash_and_replaces_on_new(self):
        a = Archive(self.path, perf_desc="low")
        a.admit("c1", entry(9.0, code_path="codes/i1/s"), code_hash="h1")
        self.assertEqual(a.get_code_hash("c1"), "h1")
        # 换代（不同码）→ 新哈希落格
        a.admit("c1", entry(5.0, code_path="codes/i2/s"), code_hash="h2")
        self.assertEqual(a.get_code_hash("c1"), "h2")

    def test_admit_without_code_hash_keeps_none(self):
        a = Archive(self.path, perf_desc="low")
        a.admit("c1", entry(5.0))
        self.assertIsNone(a.get_code_hash("c1"))
        self.assertFalse(a.is_stale("c1"))

    def test_get_code_hash_unknown_cell_none(self):
        a = Archive(self.path, perf_desc="low")
        self.assertIsNone(a.get_code_hash("nope"))

    def test_set_stale_roundtrip_and_unknown_cell(self):
        a = Archive(self.path, perf_desc="low")
        self.assertFalse(a.is_stale("c1"))   # 未知格 → False（不炸）
        a.set_stale("c1", True)
        self.assertTrue(a.is_stale("c1"))
        self.assertIn("c1", a.cells)          # 骨架创建（同 set_weight 语义）
        a.set_stale("c1", False)
        self.assertFalse(a.is_stale("c1"))

    def test_refresh_measurement_updates_value_not_gen(self):
        """同码重测：只刷测量值（gen/位次/方法基座/history 全不动）。"""
        a = Archive(self.path, perf_desc="low")
        a.admit("c1", entry(5.0, code_path="codes/i1/s", gen=4),
                code_hash="h1")
        a.set_stale("c1", False)
        ok = a.refresh_measurement("c1", 4.7, ref_runtime_ms=5200.0)
        self.assertTrue(ok)
        el = a.get_elite("c1")
        self.assertEqual(el["perf_value"], 4.7)          # 测量值刷新
        self.assertEqual(el["gen"], 4)                    # gen 不动
        self.assertEqual(el["code_path"], "codes/i1/s")  # 位次/代码不动
        self.assertEqual(el["ref_runtime_ms"], 5200.0)   # ref 一并刷新
        self.assertEqual(a.cells["c1"]["history"], [])    # 不进 history
        self.assertEqual(a.get_code_hash("c1"), "h1")     # 哈希锚不动
        self.assertFalse(a.is_stale("c1"))                # 不触发相位差

    def test_refresh_measurement_unknown_cell_false(self):
        a = Archive(self.path, perf_desc="low")
        self.assertFalse(a.refresh_measurement("nope", 1.0))

    def test_refresh_measurement_without_ref_keeps_old(self):
        a = Archive(self.path, perf_desc="low")
        e = entry(5.0)
        e["elite"]["ref_runtime_ms"] = 5000.0
        a.admit("c1", e)
        a.refresh_measurement("c1", 4.8)   # 不带 ref
        el = a.get_elite("c1")
        self.assertEqual(el["perf_value"], 4.8)
        self.assertEqual(el["ref_runtime_ms"], 5000.0)  # 旧 ref 保留

    def test_ref_runtime_ms_carried_in_elite_and_history(self):
        """R3：ref_runtime_ms 属精英字段——admit 携带、换代后旧值入 history。"""
        a = Archive(self.path, perf_desc="low")
        e1 = entry(9.0, code_path="codes/i1/s", gen=1)
        e1["elite"]["ref_runtime_ms"] = 5000.0
        a.admit("c1", e1)
        e2 = entry(5.0, code_path="codes/i2/s", gen=2)
        e2["elite"]["ref_runtime_ms"] = 5300.0
        a.admit("c1", e2)
        el = a.get_elite("c1")
        self.assertEqual(el["ref_runtime_ms"], 5300.0)
        self.assertEqual(a.cells["c1"]["history"][0]["ref_runtime_ms"], 5000.0)

    def test_old_archive_without_task58_fields_backfilled(self):
        """旧档格无 elite_code_hash/method_stale → load 后补默认。"""
        with open(self.path, "w", encoding="utf-8") as f:
            json.dump({
                "meta": {"env": "", "feature_axes": [], "perf_desc": "low",
                         "updated_at": "2026-10-01T00:00:00+08:00"},
                "cells": {"c1": {"descriptor": {"a": "x"},
                                 "elite": {"code_path": "codes/old/s.py",
                                           "perf_value": 5.0},
                                 "history": [], "weight": 1.0, "visit": 1}},
                "pending": [],
            }, f)
        a = Archive(self.path, perf_desc="low")
        a.load()
        cell = a.cells["c1"]
        self.assertIn("elite_code_hash", cell)
        self.assertIsNone(cell["elite_code_hash"])
        self.assertIn("method_stale", cell)
        self.assertFalse(cell["method_stale"])
        self.assertIsNone(a.get_code_hash("c1"))
        self.assertFalse(a.is_stale("c1"))

    def test_set_stale_persisted_and_restored(self):
        a = Archive(self.path, perf_desc="low")
        a.admit("c1", entry(5.0), code_hash="h1")
        a.set_stale("c1", True)
        a.save()
        b = Archive(self.path, perf_desc="low")
        b.load()
        self.assertTrue(b.is_stale("c1"))
        self.assertEqual(b.get_code_hash("c1"), "h1")
