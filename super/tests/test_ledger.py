"""test_ledger.py — job_47 task26 单元测试

Ledger（实验台账）行为契约：
  - append：自动补 ts / 不污染调用方 dict / 非法条目与非序列化字段容错
  - read_all：坏行与空行跳过；文件缺失返回空
  - brief：最近 n 条精简摘要（idea 截 120；perf_value 取 gate 优先）
  - perf_timeline：is_record 按「低优 + 严格优于历史全部」判定；截断窗口
    不影响历史判定；无 perf 条目不上线
  - own_decisions：按 agents 含 role 过滤，返回台账内效果字段
  - 懒建目录：ws 不存在时 append 自动建
"""
import json
import os
import tempfile
import unittest

from super.superopt.ledger import Ledger


def entry(iteration_id, cell_id="c1", verdict="admit", perf_value=100.0,
          idea="idea", perf_delta=None, reject_reason=None, agents=None,
          failure_stage=None):
    """构造一条符合 schema 约定的台账条目。"""
    return {
        "iteration_id": iteration_id,
        "cell_id": cell_id,
        "idea": idea,
        "parents": [],
        "plan_steps_count": 3,
        "gate": {
            "passed": verdict in ("admit", "replace"),
            "failure_stage": failure_stage,
            "perf_value": perf_value,
        },
        "verdict": verdict,
        "perf_delta": perf_delta,
        "reject_reason": reject_reason,
        "agents": agents or {},
    }


class LedgerTestBase(unittest.TestCase):
    def setUp(self):
        self.tmp = tempfile.TemporaryDirectory()
        self.ws = self.tmp.name
        self.ledger = Ledger(self.ws)

    def tearDown(self):
        self.tmp.cleanup()


class TestAppendRead(LedgerTestBase):
    def test_append_and_read_roundtrip(self):
        self.ledger.append(entry(1, "a=x", perf_value=10.0))
        self.ledger.append(entry(2, "a=y", verdict="reject", perf_value=12.0,
                                 reject_reason="soft_gate"))
        rows = self.ledger.read_all()
        self.assertEqual(len(rows), 2)
        self.assertEqual(rows[0]["iteration_id"], 1)
        self.assertEqual(rows[1]["verdict"], "reject")
        self.assertEqual(rows[1]["reject_reason"], "soft_gate")
        # ts 自动补（RFC3339 带时区）
        self.assertIn("+08:00", rows[0]["ts"])

    def test_append_does_not_mutate_caller_dict(self):
        src = entry(1)
        self.ledger.append(src)
        self.assertNotIn("ts", src)  # 复制后补 ts，原 dict 不动

    def test_append_tolerates_invalid_entries(self):
        for bad in ("not-a-dict", None, 123, ["x"]):
            self.ledger.append(bad)  # 不应抛异常
        self.assertEqual(self.ledger.read_all(), [])

    def test_append_coerces_unserializable_fields(self):
        e = entry(1)
        e["extra"] = {1, 2, 3}  # set 不可直接 JSON 序列化
        self.ledger.append(e)
        rows = self.ledger.read_all()
        self.assertEqual(len(rows), 1)
        self.assertIsInstance(rows[0]["extra"], str)  # default=str 兜底

    def test_lazy_dir_creation(self):
        ws = os.path.join(self.ws, "nested", "dir")
        lg = Ledger(ws)
        lg.append(entry(1))
        self.assertTrue(os.path.isfile(os.path.join(ws, "ledger.jsonl")))
        self.assertEqual(len(lg.read_all()), 1)


class TestReadAllTolerance(LedgerTestBase):
    def test_skips_bad_and_blank_lines(self):
        path = os.path.join(self.ws, "ledger.jsonl")
        with open(path, "w", encoding="utf-8") as f:
            f.write(json.dumps(entry(1), ensure_ascii=False) + "\n")
            f.write("not-json-at-all\n")          # 坏行
            f.write("\n")                          # 空行
            f.write('{"iteration_id": 3, "truncated_writ\n')  # 半写行
            f.write(json.dumps(entry(4), ensure_ascii=False) + "\n")
            f.write("[1,2,3]\n")                   # 合法 JSON 但非 dict
        rows = self.ledger.read_all()
        self.assertEqual([r["iteration_id"] for r in rows], [1, 4])

    def test_missing_file_returns_empty(self):
        self.assertEqual(self.ledger.read_all(), [])
        self.assertEqual(self.ledger.brief(), [])
        self.assertEqual(self.ledger.perf_timeline(), [])
        self.assertEqual(self.ledger.own_decisions("adjust"), [])


class TestBrief(LedgerTestBase):
    def test_brief_fields_and_truncation(self):
        long_idea = "x" * 300
        self.ledger.append(entry(1, "a=x", perf_value=9.5,
                                 idea=long_idea, perf_delta=-0.4))
        self.ledger.append(entry(2, "a=y", verdict="reject",
                                 perf_value=11.0, reject_reason="stage2"))
        brief = self.ledger.brief(2)
        self.assertEqual(len(brief), 2)
        self.assertEqual(len(brief[0]["idea"]), 120)
        self.assertEqual(brief[0]["verdict"], "admit")
        self.assertEqual(brief[0]["perf_value"], 9.5)
        self.assertEqual(brief[0]["perf_delta"], -0.4)
        self.assertEqual(brief[1]["reject_reason"], "stage2")
        for key in ("iteration_id", "cell_id", "idea", "verdict",
                    "perf_value", "perf_delta", "reject_reason"):
            self.assertIn(key, brief[0])

    def test_brief_n_limit_and_invalid_n(self):
        for i in range(1, 5):
            self.ledger.append(entry(i))
        self.assertEqual([b["iteration_id"] for b in self.ledger.brief(2)],
                         [3, 4])
        self.assertEqual(self.ledger.brief(0), [])
        self.assertEqual(self.ledger.brief(-1), [])


class TestPerfTimeline(LedgerTestBase):
    def test_is_record_low_better(self):
        # 低优：越小越好 → 10(记录) 8(记录) 12(非) 7(记录) 9(非)
        for it, perf, verdict in [(1, 10.0, "admit"), (2, 8.0, "admit"),
                                  (3, 12.0, "reject"), (4, 7.0, "replace"),
                                  (5, 9.0, "reject")]:
            self.ledger.append(entry(it, verdict=verdict, perf_value=perf))
        tl = self.ledger.perf_timeline(10)
        self.assertEqual([t["iteration_id"] for t in tl], [1, 2, 3, 4, 5])
        self.assertEqual([t["is_record"] for t in tl],
                         [True, True, False, True, False])
        self.assertEqual(tl[3]["perf_value"], 7.0)
        self.assertEqual(tl[0]["cell_id"], "c1")

    def test_record_judgment_uses_full_history_when_windowed(self):
        # 12 条：1 是最初记录；10 是唯一后来刷新（6.0）的新记录
        perfs = [100.0, 99, 98, 97, 96, 95, 94, 93, 92, 6.0, 91, 90]
        for i, perf in enumerate(perfs, start=1):
            self.ledger.append(entry(i, perf_value=perf))
        tl = self.ledger.perf_timeline(3)  # 只取尾部 3 条（10/11/12）
        self.assertEqual([t["iteration_id"] for t in tl], [10, 11, 12])
        # 第 10 条虽然窗口外的历史里它是刷新 → 仍需 True
        self.assertTrue(tl[0]["is_record"])
        self.assertFalse(tl[1]["is_record"])
        self.assertFalse(tl[2]["is_record"])

    def test_entries_without_perf_are_skipped(self):
        self.ledger.append(entry(1, verdict="no_candidate", perf_value=None))
        self.ledger.append(entry(2, perf_value=5.0))
        tl = self.ledger.perf_timeline(10)
        self.assertEqual([t["iteration_id"] for t in tl], [2])
        self.assertTrue(tl[0]["is_record"])


class TestOwnDecisions(LedgerTestBase):
    def test_filters_by_role_and_returns_stats(self):
        self.ledger.append(entry(1, agents={"adjust": {"wall_ms": 1000,
                                                       "tokens": 500,
                                                       "truncated": False}}))
        self.ledger.append(entry(2, agents={"builder": {"wall_ms": 200,
                                                        "tokens": 10,
                                                        "truncated": False}}))
        self.ledger.append(entry(3, verdict="replace",
                                 perf_delta=0.2,
                                 agents={"adjust": {"wall_ms": 900,
                                                    "tokens": 420,
                                                    "truncated": True}}))
        out = self.ledger.own_decisions("adjust", 5)
        self.assertEqual([o["iteration_id"] for o in out], [1, 3])
        self.assertEqual(out[1]["verdict"], "replace")
        self.assertEqual(out[1]["perf_delta"], 0.2)
        self.assertEqual(out[1]["role_stats"]["tokens"], 420)
        self.assertTrue(out[1]["role_stats"]["truncated"])

    def test_own_decisions_limits_and_invalid_args(self):
        for i in range(1, 4):
            self.ledger.append(entry(i, agents={"adjust": {"wall_ms": i,
                                                           "tokens": i,
                                                           "truncated": False}}))
        self.assertEqual([o["iteration_id"]
                          for o in self.ledger.own_decisions("adjust", 2)],
                         [2, 3])
        self.assertEqual(self.ledger.own_decisions("", 2), [])
        self.assertEqual(self.ledger.own_decisions("adjust", 0), [])


if __name__ == "__main__":
    unittest.main()
