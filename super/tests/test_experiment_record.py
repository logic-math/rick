"""test_experiment_record.py — 实验记录模块单元测试（job_47 迭代8 / task38）

跑法（仓库根）：
    python3 -m unittest super.tests.test_experiment_record -v

覆盖：六段模板（create）/ 段标题齐备 / 幂等追加（同名替换）/ read 缺文件 /
digest（前 3 段+结论段）/ recent（区间过滤+升序+格 id 提取+预算截断）。
"""
import os
import tempfile
import unittest

from super.superopt import experiment_record as er


class TestCreate(unittest.TestCase):
    def setUp(self):
        self.tmp = tempfile.mkdtemp(prefix="exp-rec-")
        self.iter_dir = os.path.join(self.tmp, "iter003-abcd1234")
        os.makedirs(self.iter_dir, exist_ok=True)

    def test_create_writes_six_sections(self):
        path = er.create(self.iter_dir, {
            "iteration_id": 3,
            "cell_id": "impl_strategy=blocked|memory_layout=tiled",
            "idea": "分块降低 cache miss",
            "parents": ["codes/iter001-aaaaaaaa/solution.py"],
        })
        self.assertEqual(path, er.path_for(self.iter_dir))
        text = er.read(self.iter_dir)
        self.assertIn("# 实验记录 iter3 · impl_strategy=blocked|memory_layout=tiled", text)
        for title in er.SECTION_TITLES:
            self.assertIn("## " + title, text)
        # idea 进第 1 段；其余占位
        self.assertIn("分块降低 cache miss", text)
        self.assertIn("（待 builder 填写）", text)
        # 父本进 meta 块
        self.assertIn("iter001-aaaaaaaa", text)

    def test_create_rejects_non_dict_meta(self):
        with self.assertRaises(ValueError):
            er.create(self.iter_dir, "nope")

    def test_read_missing_returns_empty(self):
        self.assertEqual(er.read(os.path.join(self.tmp, "iter999-x")), "")


class TestAppendSection(unittest.TestCase):
    def setUp(self):
        self.tmp = tempfile.mkdtemp(prefix="exp-rec-")
        self.iter_dir = os.path.join(self.tmp, "iter002-bbbb")
        er.create(self.iter_dir, {"iteration_id": 2, "cell_id": "a=x", "idea": "i"})

    def test_append_new_section(self):
        ok = er.append_section(self.iter_dir, "附注", "补充说明")
        self.assertTrue(ok)
        text = er.read(self.iter_dir)
        self.assertIn("## 附注", text)
        self.assertIn("补充说明", text)

    def test_append_same_section_is_idempotent_replace(self):
        t = er.SECTION_TITLES[1]  # 2. 实验假设
        er.append_section(self.iter_dir, t, "假设 A 成立")
        er.append_section(self.iter_dir, t, "假设 B 成立（修订）")
        text = er.read(self.iter_dir)
        self.assertEqual(text.count("## " + t), 1, "同名段不得重复")
        self.assertIn("假设 B 成立（修订）", text)
        self.assertNotIn("假设 A 成立", text)

    def test_append_without_file_returns_false(self):
        self.assertFalse(er.append_section(os.path.join(self.tmp, "iter404-x"),
                                           "t", "b"))


class TestDigestAndRecent(unittest.TestCase):
    def setUp(self):
        self.ws = tempfile.mkdtemp(prefix="exp-rec-ws-")
        self.codes = os.path.join(self.ws, "codes")
        os.makedirs(self.codes, exist_ok=True)

    def _make(self, n, cell="a=x"):
        d = os.path.join(self.codes, "iter%03d-uuid%s" % (n, n))
        er.create(d, {"iteration_id": n, "cell_id": cell, "idea": "idea-%d" % n})
        er.append_section(d, er.SECTION_TITLES[1], "假设-%d" % n)
        er.append_section(d, er.SECTION_TITLES[5], "结论-%d" % n)
        return d

    def test_digest_contains_first_three_and_conclusion(self):
        d = self._make(5)
        dg = er.digest(d)
        self.assertIn("idea-5", dg)                  # 第 1 段
        self.assertIn("假设-5", dg)                  # 第 2 段
        self.assertIn("结论-5", dg)                  # 第 6 段
        self.assertNotIn("具体实验步骤", dg)          # 第 4 段不在摘要
        # limit 截断
        short = er.digest(d, limit=40)
        self.assertLessEqual(len(short), 40 + len("\n…（截断）"))

    def test_recent_filters_range_and_sorts(self):
        for n in (1, 2, 3, 7, 9):
            self._make(n)
        got = er.recent(self.ws, since_iter=2, until_iter=7)
        self.assertEqual([g["iteration_id"] for g in got], [2, 3, 7])
        self.assertEqual(got[0]["cell_id"], "a=x")
        self.assertTrue(got[0]["path"].endswith("experiment_record.md"))
        self.assertIn("假设-2", got[0]["digest"])

    def test_recent_empty_when_no_codes_dir(self):
        self.assertEqual(er.recent(os.path.join(self.ws, "nope"), 0, 99), [])

    def test_recent_budget_keeps_all_entries_compressed(self):
        for n in (1, 2, 3):
            self._make(n)
        got = er.recent(self.ws, 0, 9, limit_chars=300)
        # 预算再紧也不丢条目（区间全景是 adjust 核心输入）
        self.assertEqual([g["iteration_id"] for g in got], [1, 2, 3])
        # digest 被压缩（远小于完整摘要），但条目一个不少
        for g in got:
            self.assertLessEqual(len(g["digest"]), 200, "digest 应被压缩")
        full = er.recent(self.ws, 0, 9, limit_chars=0)
        self.assertGreater(len(full[0]["digest"]), 40, "无预算限制时摘要完整")


if __name__ == "__main__":
    unittest.main()
