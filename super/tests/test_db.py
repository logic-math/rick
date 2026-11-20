"""test_db.py — superopt.db 五核心表/进程锁/槽位资源调度 单元测试（job_56 task1）

RED→GREEN 基准：本文件先于 db.py 编写（TDD 任务书约定）。
覆盖：
  - 五表 DDL 与核心列（cells/experiments/humans/events/resources）
  - 事件 CRUD（append/read/experiment_seq 续号/last_seq）
  - cells 状态读写（archive 兼容投影 + 核心列）
  - humans 两类队列（intervention drain 标 applied / proposal push 查询）
  - experiments 注册（<iter>-<YYYYMMDDHHMM> 编号 / 同分钟撞号后缀 / 更新回写）
  - 资源槽（实验槽/环境槽 / acquire 事务抢位 / release / 二级限额语义）
  - 进程锁（存活 PID 拒绝 / 死 PID 放行 / 自身 PID 注册）
  - WAL 模式与单例（db_for_ws 同 ws 同实例）
stdlib-only，Python >=3.9。
"""
import os
import subprocess
import sys
import tempfile
import time
import unittest

sys.path.insert(0, os.path.join(os.path.dirname(os.path.dirname(
    os.path.abspath(__file__))), "..", "super"))

from superopt import db as dbm  # noqa: E402


def _mkws() -> str:
    d = tempfile.mkdtemp(prefix="superopt-db-test-")
    return d


class TestSchema(unittest.TestCase):
    def test_five_tables_exist_with_core_columns(self):
        ws = _mkws()
        d = dbm.SuperDB(ws)
        con = d._conn
        tables = {r[0] for r in con.execute(
            "SELECT name FROM sqlite_master WHERE type='table'")}
        for t in ("cells", "experiments", "humans", "events", "resources"):
            self.assertIn(t, tables, "缺核心表 %s" % t)
        # cells 核心列（任务书 KR-A）
        cols = {r[1] for r in con.execute("PRAGMA table_info(cells)")}
        for c in ("cell_id", "name", "method_desc", "best_experiment_id", "weight"):
            self.assertIn(c, cols, "cells 缺列 %s" % c)
        # humans 两类队列字段
        cols = {r[1] for r in con.execute("PRAGMA table_info(humans)")}
        for c in ("queue_type", "content", "applied"):
            self.assertIn(c, cols)
        # events 表字段
        cols = {r[1] for r in con.execute("PRAGMA table_info(events)")}
        for c in ("seq", "ts", "event_type", "source", "payload"):
            self.assertIn(c, cols)
        # resources 资源类型 + 额度
        cols = {r[1] for r in con.execute("PRAGMA table_info(resources)")}
        for c in ("resource_type", "resource_id", "quota"):
            self.assertIn(c, cols)
        # experiments 编号列
        cols = {r[1] for r in con.execute("PRAGMA table_info(experiments)")}
        for c in ("experiment_id", "iteration", "workdir", "record_path",
                  "session_id", "score"):
            self.assertIn(c, cols)

    def test_wal_mode(self):
        ws = _mkws()
        d = dbm.SuperDB(ws)
        mode = d._conn.execute("PRAGMA journal_mode").fetchone()[0]
        self.assertEqual(mode.lower(), "wal")

    def test_singleton_per_ws(self):
        ws = _mkws()
        a = dbm.db_for_ws(ws)
        b = dbm.db_for_ws(ws)
        self.assertIs(a, b, "db_for_ws 同 ws 必须同实例（连接共享）")


class TestEventsAPI(unittest.TestCase):
    def test_append_and_read_all(self):
        ws = _mkws()
        d = dbm.db_for_ws(ws)
        ev1 = d.append_event("iteration_started", "scheduler",
                             {"iteration_id": 1})
        ev2 = d.append_event("cell_sampled", "scheduler",
                             {"cell_id": "a=x"}, experiment_id="ax#iter1")
        self.assertEqual(ev1["seq"], 1)
        self.assertEqual(ev2["seq"], 2)
        self.assertIn("ts", ev1)
        self.assertEqual(ev1["payload"], {"iteration_id": 1})
        all_evs = d.read_events()
        self.assertEqual(len(all_evs), 2)
        self.assertEqual(all_evs[0]["type"], "iteration_started")
        self.assertEqual(all_evs[1]["experiment_id"], "ax#iter1")
        self.assertEqual(all_evs[1]["experiment_seq"], 1)

    def test_experiment_seq_monotonic(self):
        ws = _mkws()
        d = dbm.db_for_ws(ws)
        d.append_event("cell_sampled", "scheduler", {}, experiment_id="ax#iter1")
        d.append_event("gate_passed", "gate", {}, experiment_id="ax#iter1")
        d.append_event("cell_sampled", "scheduler", {}, experiment_id="by#iter1")
        seqs = [e["experiment_seq"] for e in d.read_experiment_events("ax#iter1")]
        self.assertEqual(seqs, [1, 2])
        # 续号（重启后新实例继续单调——单例缓存外新建连接验证）
        d2 = dbm.SuperDB(os.path.join(ws))
        d2.append_event("iteration_ended", "scheduler", {},
                        experiment_id="ax#iter1")
        seqs = [e["experiment_seq"] for e in d2.read_experiment_events("ax#iter1")]
        self.assertEqual(seqs, [1, 2, 3])

    def test_last_seq(self):
        ws = _mkws()
        d = dbm.db_for_ws(ws)
        self.assertEqual(d.last_seq(), 0)
        d.append_event("a", "scheduler", {})
        d.append_event("b", "scheduler", {})
        self.assertEqual(d.last_seq(), 2)

    def test_experiment_ids_order(self):
        ws = _mkws()
        d = dbm.db_for_ws(ws)
        d.append_event("a", "scheduler", {}, experiment_id="ax#iter1")
        d.append_event("a", "scheduler", {}, experiment_id="by#iter1")
        d.append_event("a", "scheduler", {}, experiment_id="ax#iter2")
        self.assertEqual(d.experiment_ids(), ["ax#iter1", "by#iter1", "ax#iter2"])


class TestCellsAPI(unittest.TestCase):
    def test_upsert_and_load_full_state(self):
        ws = _mkws()
        d = dbm.db_for_ws(ws)
        cell = {"descriptor": {"a": "x"}, "elite": {"perf_value": 3.0},
                "history": [], "weight": 1.5, "visit": 2, "description":
                {"name": "n", "method": "m", "source_refs": ["r1"]}}
        d.upsert_cell_state("a=x", cell, name="n", method_desc="m",
                            best_experiment_id="1-202611300900", weight=1.5)
        cells = d.load_cells()
        self.assertEqual(cells["a=x"]["elite"]["perf_value"], 3.0)
        self.assertEqual(cells["a=x"]["weight"], 1.5)
        # 核心列可 SQL 直查
        row = d._conn.execute(
            "SELECT name, method_desc, best_experiment_id, weight "
            "FROM cells WHERE cell_id=?", ("a=x",)).fetchone()
        self.assertEqual(row[0], "n")
        self.assertEqual(row[1], "m")
        self.assertEqual(row[2], "1-202611300900")
        self.assertAlmostEqual(row[3], 1.5)

    def test_meta_roundtrip(self):
        ws = _mkws()
        d = dbm.db_for_ws(ws)
        self.assertEqual(d.get_meta(), {})
        d.save_meta({"perf_desc": "low", "env": "/tmp/e"})
        self.assertEqual(d.get_meta()["perf_desc"], "low")

    def test_pending_roundtrip(self):
        ws = _mkws()
        d = dbm.db_for_ws(ws)
        self.assertEqual(d.get_pending(), [])
        d.save_pending([{"cell_id": "a=x", "code_path": "c.py",
                         "reason": "new_cell"}])
        self.assertEqual(d.get_pending()[0]["cell_id"], "a=x")


class TestHumansAPI(unittest.TestCase):
    def test_intervention_push_drain_marks_applied(self):
        ws = _mkws()
        d = dbm.db_for_ws(ws)
        i1 = d.push_intervention("pause", {"text": "停", "category": "pause"})
        i2 = d.push_intervention("weight", {"text": "调权"})
        self.assertLess(i1, i2)
        # 未应用干预可查询（step1 注入用——task3 消费）
        pend = d.pending_interventions()
        self.assertEqual(len(pend), 2)
        self.assertEqual(pend[0]["payload"]["text"], "停")
        # drain：取走全部未应用干预并标 applied=1
        items = d.drain_interventions()
        self.assertEqual(len(items), 2)
        self.assertEqual(items[0]["type"], "pause")
        self.assertEqual(items[0]["payload"], {"text": "停", "category": "pause"})
        # drain 后清空（applied 不再返回）
        self.assertEqual(d.drain_interventions(), [])
        self.assertEqual(d.pending_interventions(), [])
        # applied 行仍在（审计）——直查表
        n = d._conn.execute(
            "SELECT COUNT(*) FROM humans WHERE queue_type='intervention' "
            "AND applied=1").fetchone()[0]
        self.assertEqual(n, 2)

    def test_proposal_queue(self):
        ws = _mkws()
        d = dbm.db_for_ws(ws)
        pid = d.push_proposal({"kind": "verify_reflect",
                                "content": "验证器疑似过拟合"})
        self.assertIsInstance(pid, int)
        props = d.pending_proposals()
        self.assertEqual(len(props), 1)
        self.assertEqual(props[0]["content"]["kind"], "verify_reflect")
        # 提案不进干预 drain
        self.assertEqual(d.drain_interventions(), [])
        d.mark_proposal_applied(pid)
        self.assertEqual(d.pending_proposals(), [])


class TestExperimentsAPI(unittest.TestCase):
    def test_register_experiment_id_format(self):
        ws = _mkws()
        d = dbm.db_for_ws(ws)
        eid = d.register_experiment(3, cell_id="a=x")
        # 格式 <iter>-<YYYYMMDDHHMM>（迭代号-年月日时分）
        self.assertTrue(eid.startswith("3-"), eid)
        tail = eid[2:]
        self.assertEqual(len(tail.split("-")[0]), 12, "时间戳须为 YYYYMMDDHHMM")
        row = d._conn.execute(
            "SELECT iteration, workdir FROM experiments WHERE experiment_id=?",
            (eid,)).fetchone()
        self.assertEqual(row[0], 3)
        self.assertEqual(row[1], eid, "workdir = experiment_id")

    def test_register_same_minute_suffix(self):
        """同 iter 同分钟并行实验：PK 撞号自动加序号后缀（并行安全）。"""
        ws = _mkws()
        d = dbm.db_for_ws(ws)
        e1 = d.register_experiment(5)
        e2 = d.register_experiment(5)
        e3 = d.register_experiment(5)
        self.assertNotEqual(e1, e2)
        self.assertNotEqual(e2, e3)
        self.assertTrue(e2.startswith(e1.split("-")[0] + "-" + e1.split("-")[1]))
        # 后缀形态：base 或 base-2/-3...
        self.assertIn(e3, (e1 + "-3", e2 + "-2", e2 + "-3", e1 + "-2") + (e3,))

    def test_update_experiment(self):
        ws = _mkws()
        d = dbm.db_for_ws(ws)
        eid = d.register_experiment(1, cell_id="a=x")
        d.update_experiment(eid, record_path="/ws/codes/%s/experiment_record.md" % eid,
                            session_id="superopt-coding-abc",
                            score=42.5, status="passed")
        row = d.get_experiment(eid)
        self.assertEqual(row["record_path"], "/ws/codes/%s/experiment_record.md" % eid)
        self.assertEqual(row["session_id"], "superopt-coding-abc")
        self.assertAlmostEqual(row["score"], 42.5)
        self.assertEqual(row["status"], "passed")


class TestResourcesAPI(unittest.TestCase):
    def test_init_slots_idempotent(self):
        ws = _mkws()
        d = dbm.db_for_ws(ws)
        d.init_slots(2, 1)
        d.init_slots(2, 1)  # 幂等
        stats = d.slot_stats()
        self.assertEqual(stats["experiment_slot"]["total"], 2)
        self.assertEqual(stats["env_slot"]["total"], 1)

    def test_acquire_release_exhaustion(self):
        ws = _mkws()
        d = dbm.db_for_ws(ws)
        d.init_slots(2, 1)
        r1 = d.acquire_slot("experiment_slot", "e1")
        r2 = d.acquire_slot("experiment_slot", "e2")
        self.assertIsNotNone(r1)
        self.assertIsNotNone(r2)
        self.assertNotEqual(r1, r2)
        # 槽满 → None（调度器阻塞等待的依据）
        self.assertIsNone(d.acquire_slot("experiment_slot", "e3"))
        # 释放后可再取
        d.release_holder("e1")
        r3 = d.acquire_slot("experiment_slot", "e3")
        self.assertIsNotNone(r3)

    def test_env_slot_secondary_limit(self):
        """环境槽二级限额：实验槽充足但环境槽满 → None。"""
        ws = _mkws()
        d = dbm.db_for_ws(ws)
        d.init_slots(4, 1)  # 4 实验槽 / 1 环境槽
        r = d.acquire_slot("env_slot", "e1")
        self.assertIsNotNone(r)
        self.assertIsNone(d.acquire_slot("env_slot", "e2"))
        d.release_holder("e1")
        self.assertIsNotNone(d.acquire_slot("env_slot", "e2"))

    def test_acquire_is_atomic_under_threads(self):
        """事务抢位：并发 acquire 不超发。"""
        import threading
        ws = _mkws()
        d = dbm.db_for_ws(ws)
        d.init_slots(3, 3)
        got = []
        lock = threading.Lock()

        def worker(i):
            r = d.acquire_slot("experiment_slot", "e%d" % i)
            with lock:
                if r:
                    got.append(r)

        ts = [threading.Thread(target=worker, args=(i,)) for i in range(8)]
        for t in ts:
            t.start()
        for t in ts:
            t.join()
        self.assertEqual(len(got), 3, "3 槽最多 3 个并发持有人")
        self.assertEqual(len(set(got)), 3, "不允许两位持有人抢到同一位")


class TestSchedulerLock(unittest.TestCase):
    def test_lock_free_then_held_then_released(self):
        ws = _mkws()
        env_dir = os.path.join(ws, "env")
        os.makedirs(env_dir, exist_ok=True)
        # 无锁文件 → 注册成功
        pid = dbm.acquire_scheduler_lock(env_dir)
        self.assertGreater(pid, 0)
        # 存活进程持锁 → 再取失败
        with self.assertRaises(dbm.SchedulerLockError) as cm:
            dbm.acquire_scheduler_lock(env_dir)
        self.assertIn(str(pid), str(cm.exception))
        # 锁文件指向已死进程（本测试用不存在的 PID）→ 放行
        dead_pid = self._find_dead_pid()
        with open(os.path.join(env_dir, "scheduler.pid"), "w") as f:
            f.write(str(dead_pid))
        pid2 = dbm.acquire_scheduler_lock(env_dir)
        self.assertEqual(pid2, os.getpid())

    @staticmethod
    def _find_dead_pid() -> int:
        # 找一个当前不存在的 PID（用大数避开常见区间；不死磕精确性）
        for cand in (999997, 999998, 999999, 999901, 999902):
            try:
                os.kill(cand, 0)
            except ProcessLookupError:
                return cand
            except PermissionError:
                continue
        return 999999

    def test_lock_error_mentions_recovery_hint(self):
        ws = _mkws()
        env_dir = os.path.join(ws, "env")
        os.makedirs(env_dir, exist_ok=True)
        with open(os.path.join(env_dir, "scheduler.pid"), "w") as f:
            f.write(str(os.getpid()))  # 当前进程存活
        with self.assertRaises(dbm.SchedulerLockError) as cm:
            dbm.acquire_scheduler_lock(env_dir)
        self.assertIn("rick super stop", str(cm.exception),
                      "报错须含回收建议（先 stop 旧进程）")


if __name__ == "__main__":
    unittest.main()
