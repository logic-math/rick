"""test_adjudicator.py — superopt.adjudicator 单元测试（job_47/task9）

覆盖（spec 测试方法）：
  - 删格（weight=0 + 连续 20 迭代未采样）→ approve
  - 活跃格删格（weight>0 / 近期被采样）→ reject「先降权观察」
  - 修门/改格定义：无 evidence.regression 双回归证据 → reject「缺双回归证据」
  - 带证据 → 按证据（双过 → approve；任一失败 → reject）
  - main 文件往返：proposal 文件 → decision 文件 + 返回码（approve=0/reject=1）
  - 防御路径：soft_weight 不走裁决 / 未知类型 / 畸形 proposal / 金样例格保护
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

from superopt import adjudicator


def _rfc3339(n: int) -> str:
    """第 n 秒的 RFC3339 时间戳（带时区，供事件序列构造）。"""
    return "2026-10-17T10:%02d:%02d+08:00" % (n // 60, n % 60)


def _ev(seq: int, typ: str, payload: dict) -> dict:
    return {"ts": _rfc3339(seq), "seq": seq, "type": typ,
            "source": "scheduler", "payload": payload}


def _write_events(ws: str, events) -> None:
    with open(os.path.join(ws, "events.jsonl"), "w", encoding="utf-8") as f:
        for ev in events:
            f.write(json.dumps(ev, ensure_ascii=False, separators=(",", ":")) + "\n")


def _write_archive(ws: str, cells: dict) -> None:
    with open(os.path.join(ws, "archive.json"), "w", encoding="utf-8") as f:
        json.dump({"meta": {"perf_desc": "low"}, "cells": cells, "pending": []}, f,
                  ensure_ascii=False, indent=2)


def _cell(weight=1.0, visit=5, elite=None) -> dict:
    return {"descriptor": {}, "elite": elite, "history": [],
            "weight": weight, "visit": visit, "sample_count": visit,
            "last_sampled_at": None}


def _proposal(ptype, target="axis1=v1", **extra) -> dict:
    p = {"proposal_id": "p-001", "type": ptype, "target_cell": target,
         "action": "delete", "evidence": "连续多轮无产出", "rule_id": "dead_axis"}
    p.update(extra)
    return p


class AdjudicatorTestCase(unittest.TestCase):
    """公共：临时 ws + 事件流 + 档案构造器。"""

    def setUp(self):
        self._td = tempfile.TemporaryDirectory()
        self.ws = os.path.join(self._td.name, "ws")
        os.makedirs(self.ws)
        os.makedirs(os.path.join(self.ws, "proposals"), exist_ok=True)

    def tearDown(self):
        self._td.cleanup()

    def build(self, events, cells) -> str:
        _write_events(self.ws, events)
        _write_archive(self.ws, cells)
        return self.ws

    def write_feature_space(self, axes) -> None:
        """expand_cell 裁决证据：标准布局 <ws>/env/feature_space.json。"""
        env_dir = os.path.join(self.ws, "env")
        os.makedirs(env_dir, exist_ok=True)
        with open(os.path.join(env_dir, "feature_space.json"), "w",
                  encoding="utf-8") as f:
            json.dump({"axes": axes}, f, ensure_ascii=False, indent=2)


class TestDeleteCellRules(AdjudicatorTestCase):
    def _events(self, last_sampled_iter, cursor):
        """迭代 1..cursor；target 格在 last_sampled_iter 被采样（None=从未）。"""
        evs = [_ev(1, "session_started", {"env": "e"})]
        seq = 2
        for it in range(1, cursor + 1):
            evs.append(_ev(seq, "iteration_started", {"iteration_id": it})); seq += 1
            if last_sampled_iter is not None and it == last_sampled_iter:
                evs.append(_ev(seq, "cell_sampled",
                               {"cell_id": "axis1=v1", "visit": 7})); seq += 1
            evs.append(_ev(seq, "iteration_ended", {"iteration_id": it})); seq += 1
        return evs

    def test_delete_extinct_cell_approved(self):
        """weight=0 且连续 20 迭代未采样 → approve。"""
        # 游标 25，最后一次采样在迭代 3 → 间隔 22 ≥ 20
        self.build(self._events(last_sampled_iter=3, cursor=25),
                   {"axis1=v1": _cell(weight=0.0, visit=7)})
        out = adjudicator.adjudicate(_proposal("hard_delete_cell"), self.ws)
        self.assertEqual(out["decision"], "approve")
        self.assertIn("conditions", out)
        self.assertIn("reason", out)

    def test_delete_boundary_exactly_20(self):
        """间隔恰 20（游标 23，末采样 3）→ approve（≥20 即『连续 20 迭代未采样』）。"""
        self.build(self._events(last_sampled_iter=3, cursor=23),
                   {"axis1=v1": _cell(weight=0.0, visit=7)})
        out = adjudicator.adjudicate(_proposal("hard_delete_cell"), self.ws)
        self.assertEqual(out["decision"], "approve")

    def test_delete_active_weight_rejected(self):
        """weight>0（活跃格）→ reject『先降权观察』。"""
        self.build(self._events(last_sampled_iter=3, cursor=25),
                   {"axis1=v1": _cell(weight=0.5, visit=7)})
        out = adjudicator.adjudicate(_proposal("hard_delete_cell"), self.ws)
        self.assertEqual(out["decision"], "reject")
        self.assertIn("降权", out["reason"])

    def test_delete_recently_sampled_rejected(self):
        """weight=0 但 5 迭代前刚被采样 → reject（未满 20 迭代观察期）。"""
        self.build(self._events(last_sampled_iter=20, cursor=25),
                   {"axis1=v1": _cell(weight=0.0, visit=7)})
        out = adjudicator.adjudicate(_proposal("hard_delete_cell"), self.ws)
        self.assertEqual(out["decision"], "reject")
        self.assertIn("降权", out["reason"])

    def test_delete_never_sampled_zero_weight_approved(self):
        """从未被采样（新格预设了 weight=0）+ weight=0 → approve。"""
        self.build(self._events(last_sampled_iter=None, cursor=25),
                   {"axis1=v1": _cell(weight=0.0, visit=0)})
        out = adjudicator.adjudicate(_proposal("hard_delete_cell"), self.ws)
        self.assertEqual(out["decision"], "approve")

    def test_delete_unknown_cell_rejected(self):
        """目标格不在档案 → reject（无证据可判，保守）。"""
        self.build(self._events(last_sampled_iter=3, cursor=25),
                   {"other-cell": _cell(weight=0.0, visit=0)})
        out = adjudicator.adjudicate(_proposal("hard_delete_cell", target="ghost"),
                                     self.ws)
        self.assertEqual(out["decision"], "reject")

    def test_weight_from_events_fallback(self):
        """档案缺格时用事件流最后一个 weight_changed 兜底（weight=0 → approve）。"""
        evs = self._events(last_sampled_iter=3, cursor=25)
        evs.append(_ev(999, "weight_changed",
                       {"cell_id": "axis1=v1", "old_weight": 1.0,
                        "new_weight": 0.0, "reason": "extinct"}))
        self.build(evs, {})  # 空 cells
        out = adjudicator.adjudicate(_proposal("hard_delete_cell"), self.ws)
        self.assertEqual(out["decision"], "approve")

    def test_golden_example_cell_protected(self):
        """目标格精英是金样例（golden_examples 引用）→ 拒删。"""
        self.build(self._events(last_sampled_iter=3, cursor=25),
                   {"axis1=v1": _cell(weight=0.0, visit=7,
                                      elite={"code_path": "codes/abc/", "perf_value": 1.0})})
        gold = os.path.join(self.ws, "gates", "golden_examples")
        os.makedirs(gold, exist_ok=True)
        with open(os.path.join(gold, "abc.json"), "w") as f:
            json.dump({"code_path": "codes/abc/"}, f)
        out = adjudicator.adjudicate(_proposal("hard_delete_cell"), self.ws)
        self.assertEqual(out["decision"], "reject")
        self.assertIn("金样例", out["reason"])


class TestModifyRules(AdjudicatorTestCase):
    def setUp(self):
        super().setUp()
        _write_events(self.ws, [_ev(1, "session_started", {"env": "e"})])
        _write_archive(self.ws, {"axis1=v1": _cell()})

    def test_modify_gate_without_regression_rejected(self):
        out = adjudicator.adjudicate(_proposal("hard_modify_gate"), self.ws)
        self.assertEqual(out["decision"], "reject")
        self.assertIn("双回归", out["reason"])

    def test_modify_cell_without_regression_rejected(self):
        """改格定义同样强制双回归证据（spec 重点）。"""
        out = adjudicator.adjudicate(_proposal("hard_modify_cell"), self.ws)
        self.assertEqual(out["decision"], "reject")
        self.assertIn("双回归", out["reason"])

    def test_modify_with_both_regressions_passing_approved(self):
        p = _proposal("hard_modify_gate",
                      evidence="收紧 stage3 容差",
                      regression={"elite_replay": {"passed": True, "killed": []},
                                  "exploit_regression": {"passed": True, "leaked": []}})
        out = adjudicator.adjudicate(p, self.ws)
        self.assertEqual(out["decision"], "approve")
        self.assertTrue(any("regression" in str(c) or "回归" in str(c)
                            for c in out.get("conditions", [])),
                        "approve 应附双回归通过的条件")

    def test_modify_with_failing_regression_rejected(self):
        p = _proposal("hard_modify_gate",
                      regression={"elite_replay": {"passed": True, "killed": []},
                                  "exploit_regression": {"passed": False,
                                                         "leaked": ["cheat-1"]}})
        out = adjudicator.adjudicate(p, self.ws)
        self.assertEqual(out["decision"], "reject")
        self.assertIn("回归", out["reason"])

    def test_regression_wrong_shape_rejected(self):
        """evidence.regression 形状不对（缺子键）→ 按缺证据拒绝。"""
        p = _proposal("hard_modify_gate", regression={"elite_replay": {"passed": True}})
        out = adjudicator.adjudicate(p, self.ws)
        self.assertEqual(out["decision"], "reject")


_EXPAND_AXES = [
    {"name": "impl", "type": "enum", "values": ["naive", "blocked"],
     "desc": "实现策略"},
    {"name": "mem", "type": "enum", "values": ["row", "tiled"], "desc": "布局"},
]


class TestExpandCellRules(AdjudicatorTestCase):
    """task19：expand_cell（扩张性硬动作）——宽松规则。

    对比参照：hard_delete 需 weight=0+20 迭代休眠；hard_modify 需双回归
    证据。expand_cell 只做结构校验（轴存在/值不重复）——加格是扩张性
    动作，风险低（裁决语义按 task19 规格）。
    """

    def test_add_axis_value_valid_approved(self):
        self.write_feature_space(_EXPAND_AXES)
        _write_events(self.ws, [_ev(1, "session_started", {"env": "e"})])
        _write_archive(self.ws, {})
        p = _proposal("expand_cell", target="", action="add_axis_value",
                      axis="mem", value="zcurve")
        out = adjudicator.adjudicate(p, self.ws)
        self.assertEqual(out["decision"], "approve")

    def test_add_axis_value_duplicate_rejected(self):
        self.write_feature_space(_EXPAND_AXES)
        _write_events(self.ws, [_ev(1, "session_started", {"env": "e"})])
        _write_archive(self.ws, {})
        p = _proposal("expand_cell", target="", action="add_axis_value",
                      axis="mem", value="row")   # 已有值
        out = adjudicator.adjudicate(p, self.ws)
        self.assertEqual(out["decision"], "reject")
        self.assertIn("已存在", out["reason"])

    def test_add_axis_value_unknown_axis_rejected(self):
        self.write_feature_space(_EXPAND_AXES)
        _write_events(self.ws, [_ev(1, "session_started", {"env": "e"})])
        _write_archive(self.ws, {})
        p = _proposal("expand_cell", target="", action="add_axis_value",
                      axis="nonexistent_axis", value="v")
        out = adjudicator.adjudicate(p, self.ws)
        self.assertEqual(out["decision"], "reject")
        self.assertIn("不存在", out["reason"])

    def test_add_axis_value_missing_fields_rejected(self):
        self.write_feature_space(_EXPAND_AXES)
        _write_events(self.ws, [_ev(1, "session_started", {"env": "e"})])
        _write_archive(self.ws, {})
        # 缺 value
        p1 = _proposal("expand_cell", target="", action="add_axis_value",
                       axis="mem")
        self.assertEqual(adjudicator.adjudicate(p1, self.ws)["decision"], "reject")
        # 缺 axis
        p2 = _proposal("expand_cell", target="", action="add_axis_value",
                       value="v")
        self.assertEqual(adjudicator.adjudicate(p2, self.ws)["decision"], "reject")

    def test_add_cell_new_values_approved(self):
        """add_cell 的 descriptor 值可在取值域外——新值即扩张点。"""
        self.write_feature_space(_EXPAND_AXES)
        _write_events(self.ws, [_ev(1, "session_started", {"env": "e"})])
        _write_archive(self.ws, {})
        p = _proposal("expand_cell", target="", action="add_cell",
                      descriptor={"impl": "fused", "mem": "swizzled"})
        out = adjudicator.adjudicate(p, self.ws)
        self.assertEqual(out["decision"], "approve")

    def test_add_cell_unknown_axis_rejected(self):
        self.write_feature_space(_EXPAND_AXES)
        _write_events(self.ws, [_ev(1, "session_started", {"env": "e"})])
        _write_archive(self.ws, {})
        p = _proposal("expand_cell", target="", action="add_cell",
                      descriptor={"impl": "x", "bad_axis": "y"})
        out = adjudicator.adjudicate(p, self.ws)
        self.assertEqual(out["decision"], "reject")
        self.assertIn("未知轴", out["reason"])

    def test_add_cell_empty_descriptor_rejected(self):
        self.write_feature_space(_EXPAND_AXES)
        _write_events(self.ws, [_ev(1, "session_started", {"env": "e"})])
        _write_archive(self.ws, {})
        p = _proposal("expand_cell", target="", action="add_cell",
                      descriptor={})
        self.assertEqual(adjudicator.adjudicate(p, self.ws)["decision"], "reject")

    def test_feature_space_missing_rejected(self):
        """feature_space 不可读 → 无法校验轴存在性 → 保守拒绝。"""
        _write_events(self.ws, [_ev(1, "session_started", {"env": "e"})])
        _write_archive(self.ws, {})
        p = _proposal("expand_cell", target="", action="add_axis_value",
                      axis="mem", value="v")
        out = adjudicator.adjudicate(p, self.ws)
        self.assertEqual(out["decision"], "reject")
        self.assertIn("不可读", out["reason"])

    def test_unknown_expand_action_rejected(self):
        self.write_feature_space(_EXPAND_AXES)
        _write_events(self.ws, [_ev(1, "session_started", {"env": "e"})])
        _write_archive(self.ws, {})
        p = _proposal("expand_cell", target="", action="replace_axis",
                      axis="mem", value="v")
        out = adjudicator.adjudicate(p, self.ws)
        self.assertEqual(out["decision"], "reject")

    def test_expand_in_hard_types_no_longer_unknown(self):
        """expand_cell 进入 _HARD_TYPES：不再落「未知提案类型」分支。"""
        self.write_feature_space(_EXPAND_AXES)
        _write_events(self.ws, [_ev(1, "session_started", {"env": "e"})])
        _write_archive(self.ws, {})
        p = _proposal("expand_cell", target="", action="add_axis_value",
                      axis="mem", value="zcurve")
        out = adjudicator.adjudicate(p, self.ws)
        self.assertNotIn("未知提案类型", out["reason"])


class TestDefensiveRules(AdjudicatorTestCase):
    def setUp(self):
        super().setUp()
        _write_events(self.ws, [_ev(1, "session_started", {"env": "e"})])
        _write_archive(self.ws, {})

    def test_soft_weight_not_routed_here(self):
        """软动作不走裁决（loop 直接应用）→ reject 说明路由错误。"""
        out = adjudicator.adjudicate(_proposal("soft_weight"), self.ws)
        self.assertEqual(out["decision"], "reject")
        self.assertIn("软动作", out["reason"])

    def test_unknown_type_rejected(self):
        out = adjudicator.adjudicate(_proposal("hard_???"), self.ws)
        self.assertEqual(out["decision"], "reject")
        self.assertIn("未知", out["reason"])

    def test_malformed_proposal_rejected(self):
        out = adjudicator.adjudicate({"proposal_id": "p-x"}, self.ws)
        self.assertEqual(out["decision"], "reject")
        out2 = adjudicator.adjudicate("not-a-dict", self.ws)
        self.assertEqual(out2["decision"], "reject")

    def test_missing_workspace_files_tolerated(self):
        """ws 缺 events/archive 文件不炸（保守 reject）。"""
        empty = os.path.join(self._td.name, "empty-ws")
        os.makedirs(empty)
        out = adjudicator.adjudicate(_proposal("hard_delete_cell"), empty)
        self.assertEqual(out["decision"], "reject")


class TestMainRoundtrip(AdjudicatorTestCase):
    def _write_proposal(self, p) -> str:
        path = os.path.join(self.ws, "proposals", p["proposal_id"] + ".json")
        with open(path, "w", encoding="utf-8") as f:
            json.dump(p, f, ensure_ascii=False)
        return path

    def _run_main(self, proposal_path):
        env = dict(os.environ)
        env["PYTHONPATH"] = _SUPER + os.pathsep + env.get("PYTHONPATH", "")
        return subprocess.run(
            [sys.executable, "-m", "superopt.adjudicator",
             "--proposal", proposal_path, "--workdir", self.ws],
            cwd=_REPO, capture_output=True, text=True, timeout=60, env=env)

    def test_main_approve_roundtrip(self):
        _write_events(self.ws, [_ev(1, "session_started", {"env": "e"})])
        _write_archive(self.ws, {"axis1=v1": _cell(weight=0.0, visit=0)})
        path = self._write_proposal(_proposal("hard_delete_cell"))
        r = self._run_main(path)
        self.assertEqual(r.returncode, 0, "approve 应返回 0：%s%s" % (r.stdout, r.stderr))
        decision_file = os.path.join(self.ws, "proposals", "p-001.decision.json")
        self.assertTrue(os.path.exists(decision_file), "decision 文件未落盘")
        with open(decision_file, encoding="utf-8") as f:
            d = json.load(f)
        self.assertEqual(d["proposal_id"], "p-001")
        self.assertEqual(d["decision"], "approve")
        self.assertIn("adjudicated_at", d)          # RFC3339 带时区
        self.assertIn("T", d["adjudicated_at"])
        self.assertTrue(("+" in d["adjudicated_at"]) or d["adjudicated_at"].endswith("Z"))
        self.assertIn("reason", d)

    def test_main_reject_roundtrip(self):
        _write_events(self.ws, [_ev(1, "session_started", {"env": "e"})])
        _write_archive(self.ws, {"axis1=v1": _cell(weight=1.0, visit=9)})
        path = self._write_proposal(_proposal("hard_delete_cell"))
        r = self._run_main(path)
        self.assertEqual(r.returncode, 1, "reject 应返回 1：%s%s" % (r.stdout, r.stderr))
        with open(os.path.join(self.ws, "proposals", "p-001.decision.json"),
                  encoding="utf-8") as f:
            d = json.load(f)
        self.assertEqual(d["decision"], "reject")

    def test_main_proposal_id_from_filename_fallback(self):
        """proposal 内容缺 proposal_id → 用文件名 stem。"""
        p = _proposal("hard_delete_cell")
        p.pop("proposal_id")
        path = os.path.join(self.ws, "proposals", "p-fallback.json")
        with open(path, "w", encoding="utf-8") as f:
            json.dump(p, f)
        r = self._run_main(path)
        self.assertIn(r.returncode, (0, 1))
        self.assertTrue(os.path.exists(
            os.path.join(self.ws, "proposals", "p-fallback.decision.json")))

    def test_main_missing_proposal_file_errors(self):
        r = self._run_main(os.path.join(self.ws, "proposals", "nope.json"))
        self.assertNotEqual(r.returncode, 0)
        self.assertNotEqual(r.returncode, 1)   # 参数/IO 错误码 ≠ 裁决语义

    def test_main_does_not_touch_events(self):
        """单写者纪律：裁决进程不得写 events.jsonl。"""
        events_path = os.path.join(self.ws, "events.jsonl")
        _write_events(self.ws, [_ev(1, "session_started", {"env": "e"})])
        before = open(events_path, "rb").read()
        _write_archive(self.ws, {"axis1=v1": _cell(weight=0.0, visit=0)})
        path = self._write_proposal(_proposal("hard_delete_cell"))
        self._run_main(path)
        after = open(events_path, "rb").read()
        self.assertEqual(before, after, "adjudicator 不得写 events.jsonl")


if __name__ == "__main__":
    unittest.main()
