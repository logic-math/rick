"""test_experiment_record.py — 实验记录模块单元测试（job_47 迭代8 / task38 / job_56 task4 四节结构）

跑法（仓库根）：
    python3 -m unittest super.tests.test_experiment_record -v

覆盖：四节模板（create，frontmatter + 实验设计子结构）/ 段标题齐备 /
幂等追加（同名替换）/ read 缺文件 / digest（第 1 节+第 3 节）/ recent
（区间过滤+升序+格 id 提取[frontmatter 优先]+预算截断+新目录名解析）。
"""
import os
import tempfile
import unittest

from super.superopt import experiment_record as er


class TestCreate(unittest.TestCase):
    def setUp(self):
        self.tmp = tempfile.mkdtemp(prefix="exp-rec-")
        self.iter_dir = os.path.join(self.tmp, "3-202611300845")
        os.makedirs(self.iter_dir, exist_ok=True)

    def test_create_writes_four_sections_and_frontmatter(self):
        path = er.create(self.iter_dir, {
            "iteration_id": 3,
            "experiment_id": "3-202611300845",
            "cell_id": "impl_strategy=blocked|memory_layout=tiled",
            "idea": "分块降低 cache miss",
            "hypothesis": "分块使 cache miss 降 30% 以上",
            "parents": ["codes/1-202611300812/solution.py"],
        })
        self.assertEqual(path, er.path_for(self.iter_dir))
        text = er.read(self.iter_dir)
        # 四节标题齐备（job_56 task4：完整事实记录——设计/过程/结论/反思）
        for title in er.SECTION_TITLES:
            self.assertIn("## " + title, text)
        self.assertEqual(len(er.SECTION_TITLES), 4)
        # frontmatter（机器可读：experiment_id/cell_id/hypothesis/status）
        self.assertIn("---", text)
        self.assertIn('experiment_id: "3-202611300845"', text)
        self.assertIn('cell_id: "impl_strategy=blocked|memory_layout=tiled"', text)
        self.assertIn('hypothesis: "分块使 cache miss 降 30% 以上"', text)
        self.assertIn("status: running", text)
        # idea 进第 1 节（实验设计·背景）；hypothesis 进实验假设小节
        self.assertIn("分块降低 cache miss", text)
        self.assertIn("分块使 cache miss 降 30% 以上", text)
        # 设计子结构（仿真预演/证伪性论证——job_56 task4 KR-A）
        self.assertIn("### 仿真预演", text)
        self.assertIn("### 证伪性论证", text)
        # 其余三节占位
        self.assertIn("（待 coding 填写）", text)
        self.assertIn("（待 debug 填写）", text)
        # 父本进 meta 块 + 实验 ID 行
        self.assertIn("1-202611300812", text)
        self.assertIn("- 实验 ID：3-202611300845", text)

    def test_create_rejects_non_dict_meta(self):
        with self.assertRaises(ValueError):
            er.create(self.iter_dir, "nope")

    def test_read_missing_returns_empty(self):
        self.assertEqual(er.read(os.path.join(self.tmp, "404-x")), "")


class TestAppendSection(unittest.TestCase):
    def setUp(self):
        self.tmp = tempfile.mkdtemp(prefix="exp-rec-")
        self.iter_dir = os.path.join(self.tmp, "2-202611300822")
        er.create(self.iter_dir, {"iteration_id": 2, "cell_id": "a=x",
                                  "idea": "i", "hypothesis": "h"})

    def test_append_new_section(self):
        ok = er.append_section(self.iter_dir, "附注", "补充说明")
        self.assertTrue(ok)
        text = er.read(self.iter_dir)
        self.assertIn("## 附注", text)
        self.assertIn("补充说明", text)

    def test_append_same_section_is_idempotent_replace(self):
        t = er.SECTION_TITLES[1]  # 2. 执行过程
        er.append_section(self.iter_dir, t, "过程 A")
        er.append_section(self.iter_dir, t, "过程 B（修订）")
        text = er.read(self.iter_dir)
        self.assertEqual(text.count("## " + t), 1, "同名段不得重复")
        self.assertIn("过程 B（修订）", text)
        self.assertNotIn("过程 A", text)

    def test_append_without_file_returns_false(self):
        self.assertFalse(er.append_section(os.path.join(self.tmp, "404-x"),
                                           "t", "b"))


class TestDigestAndRecent(unittest.TestCase):
    def setUp(self):
        self.ws = tempfile.mkdtemp(prefix="exp-rec-ws-")
        self.codes = os.path.join(self.ws, "codes")
        os.makedirs(self.codes, exist_ok=True)

    def _make(self, n, cell="a=x", dirname=None):
        # 新目录名（job_56 task1）：<N>-<YYYYMMDDHHMM>（= 工作目录名）
        d = os.path.join(self.codes,
                         dirname or "%d-2026113008%02d" % (n, n))
        er.create(d, {"iteration_id": n, "experiment_id":
                      os.path.basename(d), "cell_id": cell,
                      "idea": "idea-%d" % n, "hypothesis": "hyp-%d" % n})
        er.append_section(d, er.SECTION_TITLES[1], "过程-%d" % n)
        er.append_section(d, er.SECTION_TITLES[2], "结论-%d" % n)
        return d

    def test_digest_contains_design_and_conclusion(self):
        d = self._make(5)
        dg = er.digest(d)
        self.assertIn("idea-5", dg)                  # 第 1 节（实验设计）
        self.assertIn("结论-5", dg)                  # 第 3 节（实验结论）
        self.assertNotIn("过程-5", dg)               # 第 2 节不在摘要
        # limit 截断
        short = er.digest(d, limit=40)
        self.assertLessEqual(len(short), 40 + len("\n…（截断）"))

    def test_recent_filters_range_and_sorts(self):
        for n in (1, 2, 3, 7, 9):
            self._make(n)
        got = er.recent(self.ws, since_iter=2, until_iter=7)
        self.assertEqual([g["iteration_id"] for g in got], [2, 3, 7])
        # 格 id 提取：frontmatter 优先
        self.assertEqual(got[0]["cell_id"], "a=x")
        self.assertTrue(got[0]["path"].endswith("experiment_record.md"))
        self.assertIn("hyp-2", got[0]["digest"])   # 第 1 节实验假设在摘要内

    def test_recent_parses_legacy_dirname(self):
        # 旧目录名（历史工作区归档直读）：iter{N}-{uuid}
        self._make(11, dirname="iter011-uuid11")
        got = er.recent(self.ws, since_iter=0, until_iter=99)
        self.assertEqual([g["iteration_id"] for g in got], [11])

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
