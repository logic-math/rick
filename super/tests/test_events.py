"""test_events.py — job_47 task1 单元测试

EventWriter（append-only 事件日志，单写者真相源）行为契约：
  - append：seq 连续单调（1 起）；每行合法 JSON；ts 带时区（RFC3339）；
    紧凑单行写出（separators=(',',':')，重序列化逐字节一致）
  - 重开（模拟重启）：seq 续号不回退
  - read_all：幂等、只读；坏行/半行跳过不抛异常（截断容错）
  - last_seq：最后完整事件 seq（游标）；空文件 = 0
  - 截断封口：尾部半行后重开 writer → 补换行封口，新事件不与残行粘连
  - append 写前校验：schema 违规抛 ValueError，坏事件不落盘
  - validate_event：信封五字段/时区/source 域/payload 必填字段
"""
import json
import os
import sys
import tempfile
import unittest

sys.path.insert(0, os.path.abspath(os.path.join(os.path.dirname(__file__), os.pardir, os.pardir)))

from super.superopt.events import (  # noqa: E402
    EVENT_SCHEMAS,
    EVENT_TYPES,
    EventWriter,
    cell_safe,
    make_experiment_id,
    validate_event,
)


def valid_event(**overrides):
    """构造一条合法事件（cell_sampled）。"""
    ev = {
        "ts": "2026-10-17T12:00:00+08:00",
        "seq": 1,
        "type": "cell_sampled",
        "source": "scheduler",
        "payload": {"cell_id": "impl_strategy=naive_loops|memory_layout=tiled", "visit": 3},
    }
    ev.update(overrides)
    return ev


class TestConstants(unittest.TestCase):
    def test_event_types_is_frozenset_with_design_tree_types(self):
        expected = {
            "session_started", "session_ended",
            "iteration_started", "iteration_ended",
            "cell_sampled", "agent_started", "agent_ended",
            "candidate_produced", "gate_result",
            "elite_admitted", "elite_replaced",
            "weight_changed", "proposal", "adjudicated",
            "invalidation", "resurrection", "intervention",
            "heartbeat", "env_failure_concluded",
        "api_outage_detected",
        "turn_budget_alert",
        "new_cell_rejected",
        "cell_limit_reached",
        "infra_failure_concluded",
        "self_check_skip_debug",
        # 迭代13/task57：熔断探活恢复 + 产物抢救
        "api_outage_recovered",
        "artifact_rescued",
        # 迭代13/task58：幻影 elite 防线 + ref 漂移告警
        "elite_measurement_refresh",
        "ref_drift_warning",
            "adjust_decision",   # task19：adjust 决策轨迹事件
            "cell_created",      # 迭代8：adjust new_cell（特征格新建）
            "cell_modified",     # 迭代8：adjust modify_cell（特征格描述改进）
            # 迭代9/task45：格级互斥（一格一实验）
            "cell_lock_resample", "cell_locked_skip", "cell_lock_overflow",
            "cell_lock_stale_recovered", "cell_lock_orphans_released",
            "parallel_degraded",
            # 迭代10/task50：并发实验槽
            "parallel_slots_enabled", "experiment_started",
            "experiment_finished", "experiment_failed",
            # 迭代11/task54：adjust 饥饿修复可观测
            "adjust_dispatched",
        }
        self.assertIsInstance(EVENT_TYPES, frozenset)
        self.assertEqual(EVENT_TYPES, frozenset(expected))

    def test_schemas_cover_all_types(self):
        self.assertEqual(set(EVENT_SCHEMAS), set(EVENT_TYPES))
        for etype, fields in EVENT_SCHEMAS.items():
            self.assertIsInstance(fields, tuple, "schema of %s must be tuple" % etype)

    def test_lineage_module_alignment(self):
        """task4 lineage.py 以字面量消费的事件类型必须在 EVENT_TYPES 内（跨模块对齐）。"""
        for etype in ("elite_admitted", "elite_replaced", "weight_changed",
                      "cell_sampled", "invalidation", "resurrection"):
            self.assertIn(etype, EVENT_TYPES)


class TestValidateEvent(unittest.TestCase):
    def test_valid_event_passes(self):
        self.assertEqual(validate_event(valid_event()), [])

    def test_heartbeat_empty_payload_ok(self):
        self.assertEqual(validate_event(valid_event(type="heartbeat", payload={})), [])

    def test_non_dict_rejected(self):
        errs = validate_event("not-an-event")
        self.assertTrue(errs and "object" in errs[0])

    def test_missing_envelope_keys(self):
        errs = validate_event({"ts": "2026-10-17T12:00:00+08:00", "seq": 1})
        self.assertEqual(len(errs), 3)  # type / source / payload 三项缺失

    def test_naive_ts_rejected(self):
        errs = validate_event(valid_event(ts="2026-10-17T12:00:00"))
        self.assertTrue(any("ts" in e for e in errs))

    def test_zulu_ts_ok(self):
        self.assertEqual(validate_event(valid_event(ts="2026-10-17T04:00:00Z")), [])

    def test_bad_ts_type_rejected(self):
        errs = validate_event(valid_event(ts=12345))
        self.assertTrue(any("ts" in e for e in errs))

    def test_bad_seq_rejected(self):
        for bad in ("1", 1.5, True, -1, None):
            errs = validate_event(valid_event(seq=bad))
            self.assertTrue(errs, "seq=%r 应被拒绝" % (bad,))

    def test_unknown_type_rejected(self):
        errs = validate_event(valid_event(type="explosion"))
        self.assertTrue(any("type" in e for e in errs))

    def test_source_domain(self):
        ok = ["scheduler", "gate", "human", "adjudicator",
              "agent:adjust", "agent:builder", "agent:coding", "agent:debug"]
        for s in ok:
            self.assertEqual(validate_event(valid_event(source=s)), [], s)
        for s in ["", "robot", "agent:", "agent:pilot", "Agent:adjust", 42]:
            self.assertTrue(validate_event(valid_event(source=s)), "source=%r 应被拒绝" % (s,))

    def test_payload_missing_required_field(self):
        errs = validate_event(valid_event(payload={"cell_id": "x"}))  # 缺 visit
        self.assertTrue(any("visit" in e for e in errs))

    def test_payload_must_be_dict(self):
        errs = validate_event(valid_event(payload=["not", "dict"]))
        self.assertTrue(any("payload" in e for e in errs))

    def test_extra_payload_fields_allowed(self):
        ev = valid_event(payload={"cell_id": "x", "visit": 1, "ucb_score": 0.7, "cold_start": True})
        self.assertEqual(validate_event(ev), [])


class TestEventWriter(unittest.TestCase):
    def setUp(self):
        self._tmp = tempfile.TemporaryDirectory()
        self.path = os.path.join(self._tmp.name, "events.jsonl")

    def tearDown(self):
        self._tmp.cleanup()

    def test_append_monotonic_seq_valid_lines_compact(self):
        w = EventWriter(self.path)
        seqs = [w.append("iteration_started", {"iteration_id": i + 1}, "scheduler")
                for i in range(5)]
        w.close()
        self.assertEqual(seqs, [1, 2, 3, 4, 5])  # 连续单调，1 起
        with open(self.path, encoding="utf-8") as f:
            lines = [ln.rstrip("\n") for ln in f]
        self.assertEqual(len(lines), 5)
        for i, line in enumerate(lines):
            ev = json.loads(line)  # 每行合法 JSON
            self.assertEqual(ev["seq"], i + 1)
            self.assertEqual(ev["type"], "iteration_started")
            self.assertEqual(ev["source"], "scheduler")
            self.assertEqual(ev["payload"], {"iteration_id": i + 1})
            self.assertEqual(validate_event(ev), [])  # 含 ts 带时区
            # 紧凑写出：重序列化逐字节一致（separators=(',',':')，无多余空白）
            self.assertEqual(
                json.dumps(ev, ensure_ascii=False, separators=(",", ":")), line)

    def test_ts_has_timezone_offset(self):
        with EventWriter(self.path) as w:
            w.append("heartbeat", {}, "scheduler")
        with open(self.path, encoding="utf-8") as f:
            ts = json.loads(f.readline())["ts"]
        self.assertRegex(ts, r"(Z|[+-]\d{2}:\d{2})$")  # 带时区，无裸时间戳

    def test_reopen_continues_seq(self):
        with EventWriter(self.path) as w:
            for _ in range(3):
                w.append("heartbeat", {}, "scheduler")
        with EventWriter(self.path) as w2:  # 模拟重启重开
            self.assertEqual(w2.append("heartbeat", {}, "scheduler"), 4)  # 续号不回退

    def test_read_all_idempotent_and_last_seq(self):
        with EventWriter(self.path) as w:
            for i in range(3):
                w.append("iteration_started", {"iteration_id": i + 1}, "scheduler")
            self.assertEqual(w.last_seq(), 3)
            first, second = w.read_all(), w.read_all()
        self.assertEqual(first, second)  # 只读幂等
        self.assertEqual([e["seq"] for e in first], [1, 2, 3])

    def test_fresh_file_empty_read_and_zero_cursor(self):
        with EventWriter(self.path) as w:  # 打开即建空文件
            self.assertEqual(w.read_all(), [])
            self.assertEqual(w.last_seq(), 0)
            self.assertEqual(w.append("session_started", {"env": "matmul"}, "scheduler"), 1)

    def test_truncated_tail_tolerated(self):
        with EventWriter(self.path) as w:
            for i in range(3):
                w.append("iteration_started", {"iteration_id": i + 1}, "scheduler")
        with open(self.path, "a", encoding="utf-8") as f:
            f.write('{"ts":"2026-10-17T12:00:00+08:00","seq":4,"typ')  # 半行，无换行
        with EventWriter(self.path) as r:
            self.assertEqual(len(r.read_all()), 3)  # 跳过坏行不抛异常
            self.assertEqual(r.last_seq(), 3)      # 游标=最后完整行

    def test_reopen_seals_truncated_tail_then_append_continues(self):
        with EventWriter(self.path) as w:
            w.append("heartbeat", {}, "scheduler")
            w.append("iteration_started", {"iteration_id": 1}, "scheduler")
        with open(self.path, "a", encoding="utf-8") as f:
            f.write('{"ts":"2026-10-17T12:00:00+08:0')  # 半行残骸
        with EventWriter(self.path) as w2:  # 重开触发封口
            self.assertEqual(
                w2.append("iteration_ended", {"iteration_id": 1}, "scheduler"), 3)
            events = w2.read_all()
        self.assertEqual(len(events), 3)  # 2 完整 + 新 1（残行被跳过）
        self.assertEqual([e["seq"] for e in events], [1, 2, 3])
        self.assertEqual(events[-1]["type"], "iteration_ended")

    def test_append_rejects_invalid_and_keeps_file_intact(self):
        with EventWriter(self.path) as w:
            w.append("heartbeat", {}, "scheduler")
            with self.assertRaises(ValueError):
                w.append("explosion", {}, "scheduler")  # 未知 type
            with self.assertRaises(ValueError):
                w.append("cell_sampled", {"cell_id": "x"}, "scheduler")  # 缺 visit
            with self.assertRaises(ValueError):
                w.append("heartbeat", {}, "robot")  # 非法 source
            w.append("heartbeat", {}, "scheduler")  # 拒绝不消耗 seq
            self.assertEqual([e["seq"] for e in w.read_all()], [1, 2])

    def test_append_after_close_raises(self):
        with EventWriter(self.path) as w:
            w.append("heartbeat", {}, "scheduler")
        with self.assertRaises(ValueError):
            w.append("heartbeat", {}, "scheduler")

    def test_agent_source_rich_payload_roundtrip(self):
        with EventWriter(self.path) as w:
            w.append(
                "agent_ended",
                {"role": "coding", "iteration_id": 7, "exit_code": 0, "wall_ms": 12345,
                 "usage": {"total_tokens": 8192, "cost_usd": 0.42}},
                "agent:coding")
            ev = w.read_all()[0]
        self.assertEqual(ev["source"], "agent:coding")
        self.assertEqual(ev["payload"]["usage"]["total_tokens"], 8192)

    def test_unicode_payload_roundtrip(self):
        with EventWriter(self.path) as w:
            w.append("intervention",
                    {"text": "暂停进化：人工检查", "category": "pause"}, "human")
            ev = w.read_all()[0]
        self.assertEqual(ev["payload"]["text"], "暂停进化：人工检查")
        self.assertEqual(ev["source"], "human")

    def test_last_seq_after_seal_equals_appended(self):
        with EventWriter(self.path) as w:
            for i in range(4):
                w.append("cell_sampled",
                         {"cell_id": "c%d" % i, "visit": i, "ucb_score": 0.1 * i},
                         "scheduler")
            self.assertEqual(w.last_seq(), 4)
        with EventWriter(self.path) as w2:
            self.assertEqual(w2.last_seq(), 4)  # 重开后游标一致


# --------------------------------------------------------------------------- #
# task49：实验维度（experiment_id = 格+迭代号）+ 并行写入
# --------------------------------------------------------------------------- #

class TestExperimentDimension(unittest.TestCase):
    """task49：一次实验 = 格+迭代号；实验内序号单调、跨重启续号、可还原。"""

    def setUp(self):
        self.tmp = tempfile.TemporaryDirectory()
        self.path = os.path.join(self.tmp.name, "events.jsonl")

    def tearDown(self):
        self.tmp.cleanup()

    def test_cell_safe_and_experiment_id_shape(self):
        self.assertEqual(cell_safe("impl=a|mem=b"), "impl_a_mem_b")
        self.assertEqual(make_experiment_id("impl=a|mem=b", 7),
                         "impl_a_mem_b#iter7")
        self.assertEqual(cell_safe(""), "")

    def test_experiment_seq_monotonic_per_experiment(self):
        exp = make_experiment_id("A", 1)
        with EventWriter(self.path) as w:
            for i in range(3):
                w.append("cell_sampled", {"cell_id": "A", "visit": i + 1},
                         "scheduler", experiment_id=exp)
            w.append("heartbeat", {"seq": 0, "iteration_cursor": 1}, "scheduler")
        evs = EventWriter(self.path).read_all()
        seqs = [e.get("experiment_seq") for e in evs if e.get("experiment_id") == exp]
        self.assertEqual(seqs, [1, 2, 3])
        # 运维事件不带实验字段
        hb = [e for e in evs if e["type"] == "heartbeat"][0]
        self.assertNotIn("experiment_id", hb)
        self.assertNotIn("experiment_seq", hb)

    def test_experiment_seq_restored_across_reopen(self):
        exp = make_experiment_id("A", 2)
        with EventWriter(self.path) as w:
            w.append("iteration_started", {"iteration_id": 2}, "scheduler",
                     experiment_id=exp)
            w.append("cell_sampled", {"cell_id": "A", "visit": 1}, "scheduler",
                     experiment_id=exp)
        with EventWriter(self.path) as w2:   # 模拟重启：同实验续号不回退
            w2.append("iteration_ended", {"iteration_id": 2}, "scheduler",
                      experiment_id=exp)
        seqs = [e["experiment_seq"] for e in EventWriter(self.path).read_experiment(exp)]
        self.assertEqual(seqs, [1, 2, 3])

    def test_read_experiment_restores_single_flow(self):
        a1, b1 = make_experiment_id("A", 1), make_experiment_id("B", 1)
        with EventWriter(self.path) as w:
            w.append("iteration_started", {"iteration_id": 1}, "scheduler",
                     experiment_id=a1)
            w.append("cell_sampled", {"cell_id": "A", "visit": 1}, "scheduler",
                     experiment_id=a1)
            w.append("iteration_started", {"iteration_id": 1}, "scheduler",
                     experiment_id=b1)
            w.append("iteration_ended", {"iteration_id": 1}, "scheduler",
                     experiment_id=b1)
            w.append("iteration_ended", {"iteration_id": 1}, "scheduler",
                     experiment_id=a1)
        flows = EventWriter(self.path).read_experiment(a1)
        self.assertEqual([e["type"] for e in flows],
                         ["iteration_started", "cell_sampled", "iteration_ended"])
        self.assertEqual([e["experiment_seq"] for e in flows], [1, 2, 3])
        # 未知实验 → 空
        self.assertEqual(EventWriter(self.path).read_experiment("nope#iter9"), [])

    def test_experiment_ids_listing(self):
        with EventWriter(self.path) as w:
            w.append("iteration_started", {"iteration_id": 1}, "scheduler",
                     experiment_id="A#iter1")
            w.append("iteration_started", {"iteration_id": 1}, "scheduler",
                     experiment_id="A#iter1")
            w.append("iteration_started", {"iteration_id": 2}, "scheduler",
                     experiment_id="B#iter2")
        self.assertEqual(EventWriter(self.path).experiment_ids(),
                         ["A#iter1", "B#iter2"])

    def test_parallel_append_no_corruption(self):
        """100 线程并发 append（不同实验）→ 行完整、实验内序号连续无重。"""
        import threading
        n_threads, per_thread = 20, 5
        errors = []

        def worker(idx):
            try:
                for i in range(per_thread):
                    exp = make_experiment_id("cell%d" % idx, 1)
                    w.append("cell_sampled", {"cell_id": "cell%d" % idx,
                                              "visit": i + 1},
                             "scheduler", experiment_id=exp)
            except Exception as exc:  # noqa: BLE001
                errors.append(exc)

        w = EventWriter(self.path, event_fsync=False)
        threads = [threading.Thread(target=worker, args=(i,))
                   for i in range(n_threads)]
        for t in threads:
            t.start()
        for t in threads:
            t.join()
        w.close()
        self.assertEqual(errors, [])
        evs = EventWriter(self.path).read_all()
        self.assertEqual(len(evs), n_threads * per_thread, "行数不丢")
        # 每个实验的序号 1..per_thread 连续无重
        by_exp = {}
        for e in evs:
            by_exp.setdefault(e["experiment_id"], []).append(e["experiment_seq"])
        for exp, seqs in by_exp.items():
            self.assertEqual(sorted(seqs), list(range(1, per_thread + 1)),
                             "实验 %s 序号应连续" % exp)
        # 每行都是合法 JSON 且能过 schema 校验
        for e in evs:
            self.assertEqual(validate_event(e), [])

    def test_fsync_disabled_still_readable(self):
        with EventWriter(self.path, event_fsync=False) as w:
            w.append("heartbeat", {"seq": 0, "iteration_cursor": 0}, "scheduler")
        self.assertEqual(len(EventWriter(self.path).read_all()), 1)

    def test_validate_event_experiment_fields(self):
        base = valid_event()
        # 合法实验字段
        ok = dict(base, experiment_id="A#iter1", experiment_seq=1)
        self.assertEqual(validate_event(ok), [])
        # experiment_id 形态非法
        bad = dict(base, experiment_id="no-iter-suffix", experiment_seq=1)
        self.assertTrue(any("experiment_id" in e for e in validate_event(bad)))
        # experiment_seq 缺失
        bad2 = dict(base, experiment_id="A#iter1")
        self.assertTrue(any("experiment_seq" in e for e in validate_event(bad2)))
        # experiment_seq 无 experiment_id
        bad3 = dict(base, experiment_seq=2)
        self.assertTrue(any("experiment_id" in e for e in validate_event(bad3)))
        # 运维事件（无实验字段）仍合法
        self.assertEqual(validate_event(base), [])

    def test_append_rejects_bad_experiment_id(self):
        with EventWriter(self.path) as w:
            with self.assertRaises(ValueError):
                w.append("heartbeat", {}, "scheduler", experiment_id="")
        self.assertEqual(EventWriter(self.path).read_all(), [])


class TestTask57EventContracts(unittest.TestCase):
    """task57（R1+R2）：新事件 schema 与发射方载荷对齐。

    env_11 实测教训：schema 必填字段与 loop.py 实际发射载荷不一致时，
    事件在触发路径上直接扙 ValueError 炸掉调度器（seq 18117 同型崩溃）。
    这里用「发射方真实载荷形态」逐一 append——过不了校验 = 契约破了。
    """

    def setUp(self):
        self._tmp = tempfile.TemporaryDirectory()
        self.path = os.path.join(self._tmp.name, "events.jsonl")

    def tearDown(self):
        self._tmp.cleanup()

    def test_new_event_types_present(self):
        # task57 新事件类型齐备（EVENT_TYPES + EVENT_SCHEMAS 双登记）
        for etype in ("api_outage_recovered", "artifact_rescued"):
            self.assertIn(etype, EVENT_TYPES)
            self.assertIn(etype, EVENT_SCHEMAS)
        # task58 新事件类型齐备（幻影 elite 防线 + ref 漂移告警）
        for etype in ("elite_measurement_refresh", "ref_drift_warning"):
            self.assertIn(etype, EVENT_TYPES)
            self.assertIn(etype, EVENT_SCHEMAS)

    def test_api_outage_recovered_accepts_emitter_payload(self):
        with EventWriter(self.path) as w:
            w.append("api_outage_recovered", {
                "consecutive_infra": 3, "probes": 2,
                "diagnosis": "探活成功——API 恢复，继续迭代推进",
            }, "scheduler")
        evs = EventWriter(self.path).read_all()
        self.assertEqual(evs[-1]["type"], "api_outage_recovered")

    def test_artifact_rescued_accepts_emitter_payload(self):
        with EventWriter(self.path) as w:
            w.append("artifact_rescued", {
                "iteration_id": 7, "cell_id": "impl=naive|mem=row",
                "role": "coding", "code_path": "/ws/codes/iter007-x/solution.py",
                "reason": "agent_completed_but_no_json",
                "agent_valid": False, "agent_truncated": True,
            }, "scheduler", experiment_id="impl_naive_mem_row#iter7")
        evs = EventWriter(self.path).read_all()
        self.assertEqual(evs[-1]["payload"]["reason"],
                         "agent_completed_but_no_json")

    def test_fixed_schemas_match_loop_emitters(self):
        """task57 对齐修复的三个事件：发射方真实载荷必须过校验。

        - infra_failure_concluded：consecutive_zero_token_fails/diagnosis
          （旧 schema 要求 consecutive_empty/detail——发射方没有，触发即炸）
        - turn_budget_alert：role/soft_warn（旧 schema 要求 avg_turns）
        - new_cell_rejected：reason（旧 schema 要求 target_cell）
        """
        with EventWriter(self.path) as w:
            w.append("infra_failure_concluded", {
                "consecutive_zero_token_fails": 8, "threshold": 8,
                "diagnosis": "连续 8 次 agent 0-token 失败",
            }, "scheduler")
            w.append("turn_budget_alert", {
                "role": "coding", "consecutive": 3, "soft_warn": 20,
                "avg_turns_hint": "agent 回合数连续达软告警阈值",
            }, "scheduler")
            w.append("new_cell_rejected", {
                "reason": "empty_method", "detail": "new_cell 必须带 method",
                "descriptor": {},
            }, "agent:adjust")
        self.assertEqual(len(EventWriter(self.path).read_all()), 3)


if __name__ == "__main__":
    unittest.main()
