"""test_lineage.py — superopt.lineage 单元测试（job_47/task4）

覆盖：rebuild_lineage 链序（新→旧）、rebuild_all 全格投影、
project_state 游标重放投影（iteration_cursor/state/cells 的 visit/weight/
pending/last_seq）。事件字典为内联构造（不 import task1/2/3，并行层禁耦合）。
"""
import os
import sys
import unittest

_HERE = os.path.dirname(os.path.abspath(__file__))
_SUPER = os.path.dirname(_HERE)
_REPO = os.path.dirname(_SUPER)
for _p in (_SUPER, _REPO):
    if _p not in sys.path:
        sys.path.insert(0, _p)

from superopt import lineage
from superopt.lineage import rebuild_all, rebuild_lineage, project_state


def ev(seq, typ, payload, source="scheduler"):
    return {"ts": f"2026-10-17T10:{seq // 60:02d}:{seq % 60:02d}+08:00",
            "seq": seq, "type": typ, "source": source, "payload": payload}


def build_fixture():
    """10+ 条事件：2 格（A/B）3 代 + 穿插事件。

    A: c1 → c2 → c3（3 代）
    B: b1（父本取 A 的 c1，跨格重组）
    """
    return [
        ev(1, "session_started", {"env": "/tmp/env", "cursor": 0}),
        ev(2, "iteration_started", {"iteration_id": 1, "cell_id": "A"}),
        ev(3, "cell_sampled", {"cell_id": "A", "visit": 1, "ucb_score": 0.0,
                               "cold_start": True}),
        ev(4, "elite_admitted", {"cell_id": "A", "code_path": "codes/c1/",
                                 "perf_value": 10.0, "parents": [],
                                 "gen": 1}),
        ev(5, "iteration_started", {"iteration_id": 2, "cell_id": "A"}),
        ev(6, "cell_sampled", {"cell_id": "A", "visit": 2, "ucb_score": 0.5,
                               "cold_start": False}),
        ev(7, "weight_changed", {"cell_id": "A", "old_weight": 1.0,
                                 "new_weight": 0.5, "reason": "stagnation"}),
        ev(8, "elite_replaced", {"cell_id": "A", "code_path": "codes/c2/",
                                 "perf_value": 8.0, "parents": ["codes/c1/"],
                                 "gen": 2}),
        ev(9, "iteration_started", {"iteration_id": 3, "cell_id": "B"}),
        ev(10, "cell_sampled", {"cell_id": "B", "visit": 1, "ucb_score": 0.1,
                                "cold_start": True}),
        ev(11, "elite_admitted", {"cell_id": "B", "code_path": "codes/b1/",
                                  "perf_value": 9.0, "parents": ["codes/c1/"],
                                  "gen": 2, "pending": True,
                                  "pending_reason": "tolerance_band"}),
        ev(12, "iteration_started", {"iteration_id": 4, "cell_id": "A"}),
        ev(13, "cell_sampled", {"cell_id": "A", "visit": 3, "ucb_score": 0.7,
                                "cold_start": False}),
        ev(14, "elite_replaced", {"cell_id": "A", "code_path": "codes/c3/",
                                  "perf_value": 6.0, "parents": ["codes/c2/"],
                                  "gen": 3}),
        ev(15, "heartbeat", {"state": "running"}),
    ]


class TestRebuildLineage(unittest.TestCase):
    def test_chain_order_new_to_old(self):
        events = build_fixture()
        self.assertEqual(rebuild_lineage(events, "A"),
                         ["codes/c3/", "codes/c2/", "codes/c1/"])
        self.assertEqual(rebuild_lineage(events, "B"),
                         ["codes/b1/", "codes/c1/"], "跨格父本（B 的父是 A 的 c1）")

    def test_unknown_cell_returns_empty(self):
        events = build_fixture()
        self.assertEqual(rebuild_lineage(events, "nope"), [])

    def test_prefers_payload_lineage_field(self):
        """payload 自带完整 lineage 列表时直接采信（调度器可直写谱系）。"""
        events = [
            ev(1, "elite_admitted", {"cell_id": "X", "code_path": "codes/x2/",
                                     "lineage": ["codes/x1/", "codes/x0/"],
                                     "gen": 2}),
        ]
        self.assertEqual(rebuild_lineage(events, "X"),
                         ["codes/x2/", "codes/x1/", "codes/x0/"])

    def test_cycle_guard(self):
        """父本边成环时截断（防死循环）。"""
        events = [
            ev(1, "elite_admitted", {"cell_id": "X", "code_path": "a",
                                     "parents": ["b"], "gen": 1}),
            ev(2, "elite_replaced", {"cell_id": "X", "code_path": "b",
                                     "parents": ["a"], "gen": 2}),
        ]
        chain = rebuild_lineage(events, "X")
        self.assertEqual(chain[0], "b")
        self.assertLessEqual(len(chain), 4, "环链必须截断")

    def test_rebuild_all(self):
        events = build_fixture()
        all_l = rebuild_all(events)
        self.assertEqual(set(all_l.keys()), {"A", "B"})
        self.assertEqual(all_l["A"], ["codes/c3/", "codes/c2/", "codes/c1/"])
        self.assertEqual(all_l["B"], ["codes/b1/", "codes/c1/"])


class TestProjectState(unittest.TestCase):
    def test_projection_recovers_full_state(self):
        events = build_fixture()
        st = project_state(events)
        self.assertEqual(st["iteration_cursor"], 4, "cursor=最后开始迭代号")
        self.assertEqual(st["state"], "running")
        self.assertEqual(st["last_seq"], 15)
        # cells：visit/weight 恢复
        self.assertEqual(st["cells"]["A"]["visit"], 3)
        self.assertEqual(st["cells"]["A"]["weight"], 0.5)
        self.assertEqual(st["cells"]["B"]["visit"], 1)
        self.assertEqual(st["cells"]["B"]["weight"], 1.0,
                         "无 weight_changed 的格恢复默认权重 1.0")
        # 精英投影（供 lineage/北极星）
        self.assertEqual(st["cells"]["A"]["elite"]["code_path"], "codes/c3/")
        self.assertEqual(st["cells"]["A"]["elite"]["gen"], 3)
        # pending：elite_admitted 带 pending 标志 → 延迟入库队列
        self.assertEqual(len(st["pending"]), 1)
        self.assertEqual(st["pending"][0]["cell_id"], "B")
        self.assertEqual(st["pending"][0]["code_path"], "codes/b1/")
        self.assertEqual(st["pending"][0]["reason"], "tolerance_band")

    def test_empty_events(self):
        st = project_state([])
        self.assertEqual(st["iteration_cursor"], 0)
        self.assertEqual(st["cells"], {})
        self.assertEqual(st["pending"], [])
        self.assertEqual(st["last_seq"], 0)
        self.assertIsNone(st["state"])

    def test_session_ended_sets_stopped(self):
        events = build_fixture() + [ev(16, "session_ended", {"cursor": 4})]
        st = project_state(events)
        self.assertEqual(st["state"], "stopped")
        self.assertEqual(st["last_seq"], 16)

    def test_invalidation_and_resurrection(self):
        """invalidation：精英出档 + pending 结清；resurrection：权重恢复。"""
        events = build_fixture() + [
            ev(16, "invalidation", {"cell_id": "B", "code_path": "codes/b1/",
                                    "reason": "re-eval failed"}),
            ev(17, "weight_changed", {"cell_id": "A", "old_weight": 0.5,
                                      "new_weight": 0.0,
                                      "reason": "functional extinction"}),
            ev(18, "resurrection", {"cell_id": "A", "code_path": "codes/c3/",
                                    "reason": "false kill corrected",
                                    "restored_weight": 1.0}),
        ]
        st = project_state(events)
        self.assertIsNone(st["cells"]["B"]["elite"], "invalidation 后精英出档")
        self.assertEqual(st["pending"], [], "重评估结清 pending")
        self.assertEqual(st["cells"]["A"]["weight"], 1.0,
                         "resurrection 恢复权重")

    def test_events_unsorted_are_ordered_by_seq(self):
        """投影前按 seq 排序（事件流可能乱序读取）。"""
        events = list(reversed(build_fixture()))
        st = project_state(events)
        self.assertEqual(st["last_seq"], 15)
        self.assertEqual(st["cells"]["A"]["visit"], 3)
        self.assertEqual(st["cells"]["A"]["elite"]["code_path"], "codes/c3/")


if __name__ == "__main__":
    unittest.main()
