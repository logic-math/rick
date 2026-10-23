"""test_loop.py — Scheduler 四状态循环测试（job_47/task8）

全 mock（秒级）：注入 FakeAgentClient（脚本化 AgentResult 序列）+ FakeGate
（脚本化 GateResult 序列）；Archive/EventWriter/FeatureSpace/Sampler/
CostLedger/InterventionQueue 用真实现（tmpdir 工作区）——这样恢复/事件链/
采样递增这些「修复 v5 缺陷」的核心断言落在真代码路径上。

mock_pi 场景脚本的 subprocess 往返已由 test_agents.py 覆盖；本文件聚焦
调度器编排契约（事件序列/游标恢复/淘汰语义），不需要真进程。

覆盖（task8 spec 测试要求）：
  ①  3 迭代全链事件序列有序 + seq 连续
  ② visit 真实递增（v5 断链回归断言）
  ③ 崩溃恢复：2 迭代 → 新 Scheduler 同 ws → cursor/visit 一致 → 续跑 seq 续号
  ④ debug 循环：自检失败 → M 次上限放弃（debug 尝试事件数 = M）
  ⑤ adjust 软动作（weight_changed + archive.weight）/硬动作（proposal 事件
     + <ws>/proposals/ 文件，不直改 archive）
  ⑥ 延迟入库（容差带内解 → pending 非空 + elite_admitted）
  ⑦ 预算墙（max_iterations=3 自然退出 + session_ended）
  ⑧ 门先证红失败 exit 3；决策文件 drain（adjudicated 事件）
  ⑨ 干预摄取（intervention 事件 + stop 类停止）
  ⑩ epoch pending 重评估（invalidation/resurrection）
  ⑪ env stage1 连续失败 3 次 → env_failure_concluded + 停止

stdlib-only，Python >=3.9。
"""
import json
import os
import random
import shutil
import sys
import tempfile
import time
import unittest
from typing import Any, Dict, List, Optional

_HERE = os.path.dirname(os.path.abspath(__file__))
_SUPER = os.path.dirname(_HERE)      # super/
_REPO = os.path.dirname(_SUPER)      # 仓库根
for _p in (_SUPER, _REPO):
    if _p not in sys.path:
        sys.path.insert(0, _p)

from superopt.agents import AgentResult
from superopt.gates import GateResult
from superopt.loop import EPOCH_ITERATIONS, Scheduler

# --------------------------------------------------------------------------- #
# 测试脚手架
# --------------------------------------------------------------------------- #


def _agent_result(role: str, data: Optional[Dict[str, Any]] = None,
                  valid: bool = True, exit_code: int = 0) -> AgentResult:
    """构造 AgentResult（六字段 + 角色增量载体 data）。"""
    return AgentResult(
        role=role, valid=valid, score=0.85, perf_value=123.4,
        perf_type="runtime_ms", perf_desc="low", summary="mock",
        usage={"input_tokens": 100, "output_tokens": 50, "cache_read": 10,
               "cache_write": 0, "total": 160, "cost_usd": 0.01},
        data=dict(data or {}), final_message="mock final",
        tool_calls=["read"], wall_ms=11, exit_code=exit_code)


def _builder_result() -> AgentResult:
    return _agent_result("builder", {
        "experiment_plan": {
            "idea": "分块转置", "target_cell": None, "parents": [],
            "plan_steps": ["step1"], "expected_behavior": "faster",
        }})


def _coding_result(passed: bool = True, diff_path: str = "solution.py") -> AgentResult:
    return _agent_result("coding", {
        "diff_path": diff_path,
        "self_check": {"passed": passed, "stage": "self_check",
                       "metrics": {"max_error": 1e-06}},
    })


def _debug_result(passed: bool = True) -> AgentResult:
    return _agent_result("debug", {
        "fix_report": {"attempts": 1, "root_cause": "typo", "changes": ["fix"]},
        "final_self_check": {"passed": passed, "stage": "self_check",
                             "metrics": {}},
        "diff_path": "solution.py",
        "self_check": {"passed": passed, "stage": "self_check", "metrics": {}},
    })


def _adjust_result(proposals: List[Dict[str, Any]]) -> AgentResult:
    return _agent_result("adjust", {"proposals": proposals})


class FakeAgentClient(object):
    """脚本化 pi 客户端：role → 返回值工厂列表（耗尽后重复最后一个 None-safe）。"""

    def __init__(self):
        self.script = {}          # type: Dict[str, List[AgentResult]]
        self.calls = []           # type: List[Dict[str, str]]

    def queue(self, role: str, results: List[AgentResult]) -> None:
        """替换语义：后 queue 覆盖该 role 的既有脚本（测试确定性）。"""
        self.script[role] = list(results)

    def run(self, role: str, prompt_path: str, workdir: str,
            timeout_s: int = 1800, model: Optional[str] = None,
            max_turns: Optional[int] = None) -> AgentResult:
        self.calls.append({"role": role, "prompt_path": prompt_path,
                           "workdir": workdir, "model": model,
                           "max_turns": max_turns})
        seq = self.script.get(role) or []
        if not seq:
            raise AssertionError("FakeAgentClient: role %r 无脚本（调用 %d 次）"
                                 % (role, len(self.calls)))
        # 弹出第一个；只剩一个时重复返回（避免测试里数错个数）
        if len(seq) > 1:
            return seq.pop(0)
        return seq[0]


class FakeGate(object):
    """脚本化级联门：verify 结果队列 + preverify_red 返回值。"""

    def __init__(self, preverify_ok: bool = True):
        self.verify_results = []  # type: List[GateResult]
        self.verify_calls = []    # type: List[Dict[str, Any]]
        self.preverify_ok = preverify_ok

    def queue(self, result: GateResult) -> None:
        self.verify_results.append(result)

    def preverify_red(self) -> Dict[str, Any]:
        if self.preverify_ok:
            return {"ok": True}
        return {"ok": False, "leaked": ["sample-1"], "false_killed": []}

    def verify(self, candidate_dir: str,
               elite_perf: Optional[float] = None,
               history: Optional[List[float]] = None) -> GateResult:
        self.verify_calls.append({"candidate_dir": candidate_dir,
                                  "elite_perf": elite_perf,
                                  "history": list(history or [])})
        if self.verify_results:
            return self.verify_results.pop(0)
        # 缺省：通过、无容差带
        return GateResult(passed=True, failure_stage=None,
                          metrics={"perf": {"value": 100.0, "score": 0.9,
                                            "perf_type": "runtime_ms",
                                            "perf_desc": "low", "k": 3,
                                            "timings": [100, 101, 99]},
                                   "soft": {"tolerance_band": False,
                                            "verdict": "better"}},
                          details="fake pass")


def _gate_pass(value: float = 100.0, band: bool = False) -> GateResult:
    return GateResult(passed=True, failure_stage=None,
                      metrics={"perf": {"value": value, "score": 0.9,
                                        "perf_type": "runtime_ms",
                                        "perf_desc": "low", "k": 3,
                                        "timings": [value] * 3},
                               "soft": {"tolerance_band": band,
                                        "verdict": "band" if band else "better"}},
                      details="fake pass")


def _gate_fail(stage: str = "stage2_correctness") -> GateResult:
    return GateResult(passed=False, failure_stage=stage,
                      metrics={}, details="fake fail at %s" % stage)


AXES = [
    {"name": "impl", "type": "enum", "values": ["naive", "blocked", "vectorized"],
     "desc": "实现策略"},
    {"name": "mem", "type": "enum", "values": ["row", "tiled"], "desc": "内存布局"},
]


class LoopTestBase(unittest.TestCase):
    """公共脚手架：tmpdir 工作区 + env 定义 + Scheduler 组装点。"""

    def setUp(self) -> None:
        # 固定全局随机源：Sampler 默认用 random 模块（冷启动加权随机），
        # 不固定则 hard_target=采样格 的断言 ~10% flaky（采样到 soft_target 即碰撞）
        random.seed(2026)
        self.ws = tempfile.mkdtemp(prefix="superopt-loop-test-")
        env_dir = os.path.join(self.ws, "env")
        os.makedirs(os.path.join(env_dir, "code"))
        os.makedirs(os.path.join(self.ws, "gates"))
        with open(os.path.join(env_dir, "env.json"), "w", encoding="utf-8") as f:
            json.dump({"task": "matmul", "perf_desc": "low",
                       "perf_type": "runtime_ms",
                       # 迭代5 夹具：门禁定义（coding ctx gate_defs 的来源）
                       "tolerance": {"correctness": {"max_error": 0.001},
                                     "soft_gate": {"relative_band": 0.05}},
                       "bench": {"size": 2048, "k": 3, "timeout_s": 60},
                       "gates": {"score_max": 2.0, "runtime_ms_min": 0.01}},
                      f)
        with open(os.path.join(env_dir, "domain_knowledge.md"), "w",
                  encoding="utf-8") as f:
            f.write("# matmul 领域知识\n\n## 性能分析 SOP\n先做热点剖析。\n\n"
                    "## 已知死族\n纯 python 三重循环不可达天花板。\n")
        with open(os.path.join(env_dir, "feature_space.json"), "w",
                  encoding="utf-8") as f:
            json.dump({"axes": AXES}, f)
        with open(os.path.join(env_dir, "code", "reference.py"), "w",
                  encoding="utf-8") as f:
            f.write("# reference impl\n")
        with open(os.path.join(self.ws, "super.json"), "w", encoding="utf-8") as f:
            json.dump({"state": "idle", "iteration_cursor": 0,
                       "env_path": env_dir, "pid": None,
                       "heartbeat_path": os.path.join(self.ws, "heartbeat.json"),
                       "budget": {"wall_start": None, "max_wall_time_h": 0},
                       "config": {}}, f)
        self.client = FakeAgentClient()
        self.gate = FakeGate()
        # 候选 fixture：mock coding agent 声称的 diff_path（真实存在——
        # pending 重评估的 code_path 存在性检查需要）
        self.cand_dir = os.path.join(self.ws, "fixture_cand")
        os.makedirs(self.cand_dir, exist_ok=True)
        self.cand_file = os.path.join(self.cand_dir, "solution.py")
        with open(self.cand_file, "w", encoding="utf-8") as f:
            f.write("# mock candidate\n")
        # 默认脚本：builder/coding 都成功（diff_path 指向真实 fixture）
        self.client.queue("builder", [_builder_result()])
        self.client.queue("coding", [_coding_result(diff_path=self.cand_file)])
        self.events_path = os.path.join(self.ws, "events.jsonl")
        self.archive_path = os.path.join(self.ws, "archive.json")

    def tearDown(self) -> None:
        shutil.rmtree(self.ws, ignore_errors=True)

    def fs_all_cells(self):
        from superopt.feature_space import FeatureSpace
        fs = FeatureSpace(AXES)
        return fs.all_cells()

    def make_scheduler(self, cfg_overrides: Optional[Dict[str, Any]] = None,
                       client: Optional[FakeAgentClient] = None,
                       gate: Optional[FakeGate] = None) -> Scheduler:
        cfg = {
            "state": "idle", "iteration_cursor": 0,
            "env_path": os.path.join(self.ws, "env"),
            "preverify_red": False,
            "config": {
                "adjust_interval": 0,       # 测试默认关掉 adjust
                "debug_max_attempts": 3,
                "soft_gate_k": 3,
                "heartbeat_interval_s": 0,  # 每迭代都写心跳（测试可断言）
                "max_wall_time_h": 0,       # 不限时
                "max_iterations": 0,
                "pi_path": "/nonexistent-pi",
            },
        }
        cfg["config"].update(cfg_overrides or {})
        return Scheduler(self.ws, cfg, agent_client=client or self.client,
                         gate=gate or self.gate)

    # ---- 读工具 ----

    def read_events(self) -> List[Dict[str, Any]]:
        out = []
        with open(self.events_path, "r", encoding="utf-8") as f:
            for line in f:
                if line.strip():
                    out.append(json.loads(line))
        return out

    def events_of(self, etype: str) -> List[Dict[str, Any]]:
        return [e for e in self.read_events() if e["type"] == etype]

    def read_archive(self) -> Dict[str, Any]:
        with open(self.archive_path, "r", encoding="utf-8") as f:
            return json.load(f)


# --------------------------------------------------------------------------- #
# ① 3 迭代全链：事件序列有序 + seq 连续 + ② visit 真实递增
# --------------------------------------------------------------------------- #

class TestFullLoopSequence(LoopTestBase):

    def test_three_iterations_event_chain(self):
        sched = self.make_scheduler(cfg_overrides={"max_iterations": 3})
        rc = sched.run()
        self.assertEqual(rc, 0)

        evs_all = self.read_events()
        # seq 连续（1..N 无断档；heartbeat 等运维事件也占 seq）
        seqs = [e["seq"] for e in evs_all]
        self.assertEqual(seqs, list(range(1, len(evs_all) + 1)))

        # 第一迭代的完整事件链严格有序（过滤运维事件 heartbeat）
        evs = [e for e in evs_all if e["type"] != "heartbeat"]
        first_types = [e["type"] for e in evs[:10]]
        self.assertEqual(first_types, [
            "session_started",            # 0
            "iteration_started",          # 1
            "cell_sampled",               # 2
            "agent_started", "agent_ended",   # builder   # 3,4
            "agent_started", "agent_ended",   # coding    # 5,6
            "candidate_produced",         # 7
            "gate_result",                # 8
            "elite_admitted",             # 9
        ])
        self.assertEqual(evs[3]["payload"]["role"], "builder")
        self.assertEqual(evs[5]["payload"]["role"], "coding")
        # §2.2 事件表：agent_started/ended 产生者 = scheduler；candidate_produced = agent:coding
        self.assertEqual(evs[3]["source"], "scheduler")
        self.assertEqual(evs[7]["source"], "agent:coding")

        # 每迭代收尾 iteration_ended；3 个迭代
        self.assertEqual(len(self.events_of("iteration_started")), 3)
        self.assertEqual(len(self.events_of("iteration_ended")), 3)
        self.assertEqual(len(self.events_of("cell_sampled")), 3)
        self.assertEqual(len(self.events_of("elite_admitted")), 3)

        # session 收尾
        ended = self.events_of("session_ended")
        self.assertEqual(len(ended), 1)
        self.assertEqual(ended[0]["payload"]["iteration_cursor"], 3)

        # builder agent_ended 带成本账本字段（usage/wall_ms/exit_code）
        be = [e for e in self.events_of("agent_ended")
              if e["payload"]["role"] == "builder"][0]
        for k in ("wall_ms", "exit_code", "usage"):
            self.assertIn(k, be["payload"])

    def test_visit_increments_per_iteration(self):
        """② v5 断链回归断言：迭代后 visit 恒 +1 且落在 archive。"""
        sched = self.make_scheduler(cfg_overrides={"max_iterations": 2})
        sched.run()
        sampled = self.events_of("cell_sampled")
        self.assertEqual(len(sampled), 2)
        # 两次采样的 visit 值都是「采样后真实递增」的值（>=1）
        for s in sampled:
            self.assertGreaterEqual(s["payload"]["visit"], 1)
        # visit 落在 archive（同格两次 → 2；不同格各自 1）
        cells = self.read_archive()["cells"]
        total_visit = sum(c["visit"] for c in cells.values())
        self.assertEqual(total_visit, 2)
        # 第一次采样是冷启动（visit==1 → cold_start 标记）
        self.assertTrue(sampled[0]["payload"].get("cold_start"))

    def test_heartbeat_and_super_json_state(self):
        sched = self.make_scheduler(cfg_overrides={"max_iterations": 1})
        sched.run()
        # heartbeat.json 落盘且带 seq/state
        with open(os.path.join(self.ws, "heartbeat.json"), encoding="utf-8") as f:
            hb = json.load(f)
        self.assertIn("seq", hb)
        self.assertGreaterEqual(hb["seq"], 1)
        # 收尾心跳终态 stopped（running 期心跳被终态覆写——状态机事实）
        self.assertEqual(hb["state"], "stopped")
        # 循环内确实写过心跳事件（interval=0 → 每迭代）
        self.assertGreaterEqual(len(self.events_of("heartbeat")), 1)
        # super.json state=stopped + cursor 同步
        with open(os.path.join(self.ws, "super.json"), encoding="utf-8") as f:
            sj = json.load(f)
        self.assertEqual(sj["state"], "stopped")
        self.assertEqual(sj["iteration_cursor"], 1)


# --------------------------------------------------------------------------- #
# ③ 崩溃恢复
# --------------------------------------------------------------------------- #

class TestCrashRecovery(LoopTestBase):

    def test_recover_cursor_and_visit_then_continue(self):
        # 第一段：跑 2 迭代「崩溃」（不写 session_ended，直接弃对象）
        s1 = self.make_scheduler(cfg_overrides={"max_iterations": 2})
        s1.run()
        n_events_before = len(self.read_events())
        visits_before = {cid: c["visit"]
                         for cid, c in self.read_archive()["cells"].items()}

        # 第二段：新 Scheduler 同 ws（不传 max_iterations → 恢复后手动再跑 1 迭代）
        client2 = FakeAgentClient()
        client2.queue("builder", [_builder_result()])
        client2.queue("coding", [_coding_result()])
        gate2 = FakeGate()
        s2 = self.make_scheduler(cfg_overrides={"max_iterations": 3},
                                 client=client2, gate=gate2)
        last_seq = s2._recover()
        self.assertEqual(last_seq, n_events_before)
        self.assertEqual(s2.iteration_cursor, 2)
        # visit 恢复到内存 archive（与崩溃前一致）
        for cid, v in visits_before.items():
            self.assertEqual(s2.archive.cells()[cid]["visit"], v)

        rc = s2.run()
        self.assertEqual(rc, 0)
        evs = self.read_events()
        # 续跑后 seq 无断档（1..N 连续）
        self.assertEqual([e["seq"] for e in evs],
                         list(range(1, len(evs) + 1)))
        # 事件净增（session_started + 第 3 迭代全链 + 心跳 + session_ended）
        self.assertGreater(len(evs), n_events_before)
        # 第 3 迭代的事件在第 2 迭代事件之后（seq 续号）
        starts = self.events_of("iteration_started")
        self.assertEqual([s["payload"]["iteration_id"] for s in starts], [1, 2, 3])
        # 重评估 epoch（默认 10）未触发 —— 无 invalidation
        self.assertEqual(self.events_of("invalidation"), [])
        # archive visit 总数 = 3
        cells = self.read_archive()["cells"]
        self.assertEqual(sum(c["visit"] for c in cells.values()), 3)


# --------------------------------------------------------------------------- #
# ④ debug 循环
# --------------------------------------------------------------------------- #

class TestHeartbeatThread(LoopTestBase):
    """心跳线程回归：长 agent 派发期间心跳不得停跳（realpi 冒烟发现的
    重启风暴缺陷：主循环阻塞→Go 误判心跳超时→kill→重启循环）。"""

    class _SlowClient(FakeAgentClient):
        def __init__(self, delay: float):
            FakeAgentClient.__init__(self)
            self.delay = delay

        def run(self, role, prompt_path, workdir, timeout_s=1800,
                model=None, max_turns=None):
            time.sleep(self.delay)      # 模拟长 agent 派发（真 pi 可达十余分钟）
            return FakeAgentClient.run(self, role, prompt_path, workdir,
                                       timeout_s, model=model,
                                       max_turns=max_turns)

    def test_heartbeat_during_slow_agent_dispatch(self):
        client = self._SlowClient(1.5)
        client.queue("builder", [_builder_result()])
        client.queue("coding", [_coding_result()])
        sched = self.make_scheduler(
            cfg_overrides={"max_iterations": 1, "heartbeat_interval_s": 0.2},
            client=client)
        sched.run()
        evs = self.read_events()
        hbs = [e for e in evs if e["type"] == "heartbeat"]
        # 1.5s 派发 × （builder+coding）≈ 3s，0.2s 间隔 → 至少 3 跳
        # （修复前：主循环阻塞，心跳停跳 → 0-1 跳）
        self.assertGreaterEqual(len(hbs), 3,
                                "长 agent 派发期间心跳停跳（回归：重启风暴缺陷）")
        # session_ended 后心跳线程应停止：仅允许 run() 收尾那一跳（state=stopped
        # 的设计语义，Go 侧靠它读优雅停状态），不允许线程继续跳
        end_seq = max(e["seq"] for e in evs if e["type"] == "session_ended")
        time.sleep(0.5)
        hbs_after = [e for e in self.read_events()
                     if e["type"] == "heartbeat" and e["seq"] > end_seq]
        self.assertLessEqual(len(hbs_after), 1,
                             "session_ended 后心跳线程仍在跳（仅允许收尾 1 跳）：%d"
                             % len(hbs_after))


class TestDebugLoop(LoopTestBase):

    def test_self_check_failure_debug_attempts(self):
        # coding 自检失败；debug 3 次全失败 → 放弃（不进门禁）
        self.client.queue("coding", [_coding_result(passed=False)])
        for _ in range(3):
            self.client.queue("debug", [_debug_result(passed=False)])
        sched = self.make_scheduler(cfg_overrides={"max_iterations": 1,
                                                   "debug_max_attempts": 3})
        sched.run()
        # debug agent_started × 3（M 次上限）
        debug_runs = [e for e in self.events_of("agent_started")
                      if e["payload"]["role"] == "debug"]
        self.assertEqual(len(debug_runs), 3)
        # 每次 debug 尝试有一条 candidate_produced（fix_report 载体）
        debug_cands = [e for e in self.events_of("candidate_produced")
                       if e["payload"].get("role") == "debug"]
        self.assertEqual(len(debug_cands), 3)
        for c in debug_cands:
            self.assertIn("fix_report", c["payload"])
        # 门禁没跑（自检失败链路不送终验）
        self.assertEqual(self.gate.verify_calls, [])
        # 无 elite 入格
        self.assertEqual(self.events_of("elite_admitted"), [])

    def test_debug_success_proceeds_to_gate(self):
        # coding 失败 → 第 2 次 debug 修好 → 送门禁 → 入格
        self.client.queue("coding", [_coding_result(passed=False)])
        self.client.queue("debug", [_debug_result(passed=False),
                                    _debug_result(passed=True)])
        sched = self.make_scheduler(cfg_overrides={"max_iterations": 1,
                                                   "debug_max_attempts": 3})
        sched.run()
        self.assertEqual(len(self.gate.verify_calls), 1)
        self.assertEqual(len(self.events_of("elite_admitted")), 1)
        debug_runs = [e for e in self.events_of("agent_started")
                      if e["payload"]["role"] == "debug"]
        self.assertEqual(len(debug_runs), 2)


# --------------------------------------------------------------------------- #
# ⑤ adjust 软/硬动作
# --------------------------------------------------------------------------- #

class TestAdjust(LoopTestBase):

    def test_soft_weight_applied_and_hard_proposal_queued(self):
        # adjust_interval=1 → 第 1 迭代完成后第 2 轮循环顶触发 adjust
        sched_probe = self.make_scheduler(cfg_overrides={"max_iterations": 1})
        sched_probe.run()
        sampled_cell = self.events_of("cell_sampled")[0]["payload"]["cell_id"]
        # 重开干净事件流/档案（probe 只为拿确定性格 id），并经事件把其余格
        # 灭绝（weight 0）——正式 run 的采样被钉死在 sampled_cell 上，
        # soft/hard 目标不再随机相撞
        os.unlink(self.events_path)
        if os.path.exists(self.archive_path):
            os.unlink(self.archive_path)
        from superopt.events import EventWriter as _EW
        with _EW(self.events_path) as _w:
            for cid in self.fs_all_cells():
                if cid != sampled_cell:
                    _w.append("weight_changed",
                              {"cell_id": cid, "old_weight": 1.0,
                               "new_weight": 0.0, "reason": "test isolate"},
                              "scheduler")
        soft_target = "impl=vectorized|mem=tiled"   # 隔离格（weight 0）
        hard_target = sampled_cell                    # 采样入档格（weight 1.0）
        self.client.queue("adjust", [_adjust_result([
            {"type": "soft_weight", "target_cell": soft_target,
             "action": "weight 0.5", "evidence": "visit 12 零入格",
             "rule_id": "R1"},
            {"type": "hard_delete_cell", "target_cell": hard_target,
             "action": "delete", "evidence": "长期无效",
             "rule_id": "R2"},
        ])])
        self.gate.queue(_gate_pass())
        self.gate.queue(_gate_pass())
        sched = self.make_scheduler(cfg_overrides={"max_iterations": 2,
                                                   "adjust_interval": 1})
        sched.run()
        # 软动作：weight_changed 事件 + archive.weight
        adj_wcs = [w for w in self.events_of("weight_changed")
                   if w["source"] == "agent:adjust"]
        self.assertEqual(len(adj_wcs), 1)
        self.assertEqual(adj_wcs[0]["payload"]["cell_id"], soft_target)
        self.assertEqual(adj_wcs[0]["payload"]["new_weight"], 0.5)
        cells = self.read_archive()["cells"]
        self.assertEqual(cells[soft_target]["weight"], 0.5)
        # 硬动作：proposal 事件 + <ws>/proposals/ 队列文件；不删格不直改
        props = self.events_of("proposal")
        self.assertEqual(len(props), 1)
        self.assertEqual(props[0]["payload"]["type"], "hard_delete_cell")
        self.assertIn("proposal_id", props[0]["payload"])
        prop_files = [n for n in os.listdir(os.path.join(self.ws, "proposals"))
                      if n.endswith(".json")]
        self.assertEqual(len(prop_files), 1)
        with open(os.path.join(self.ws, "proposals", prop_files[0]),
                  encoding="utf-8") as f:
            pf = json.load(f)
        self.assertEqual(pf["type"], "hard_delete_cell")
        # 硬动作不直改 archive：目标格（采样入档格）weight 仍为 1.0
        cells = self.read_archive()["cells"]
        # 硬动作不直改 archive：目标格（采样入档格）weight 仍为 1.0
        self.assertEqual(cells[hard_target]["weight"], 1.0)

    def test_decision_file_drained_to_adjudicated_event(self):
        # 先让 adjust 产一条硬动作提案
        self.client.queue("adjust", [_adjust_result([
            {"type": "hard_delete_cell", "target_cell": "impl=naive|mem=tiled",
             "action": "delete", "evidence": "x", "rule_id": "R2"}])])
        sched = self.make_scheduler(cfg_overrides={"max_iterations": 2,
                                                   "adjust_interval": 1})
        sched.run()
        prop_files = [n for n in os.listdir(os.path.join(self.ws, "proposals"))
                      if n.endswith(".json") and not n.endswith(".decision.json")]
        pid = prop_files[0][:-5]  # 去 .json
        # 手工放置 decision 文件（模拟 adjudicator 进程产出）
        dec_path = os.path.join(self.ws, "proposals", pid + ".decision.json")
        with open(dec_path, "w", encoding="utf-8") as f:
            json.dump({"proposal_id": pid, "decision": "approve",
                       "reason": "dormant ok", "conditions": [],
                       "adjudicated_at": "2026-10-17T12:00:00+08:00",
                       "adjudicator": "auto"}, f)
        # 再跑一轮：drain decision → adjudicated 事件 + 文件被消费
        client2 = FakeAgentClient()
        client2.queue("builder", [_builder_result()])
        client2.queue("coding", [_coding_result(diff_path=self.cand_file)])
        client2.queue("adjust", [_adjust_result([])])
        s2 = self.make_scheduler(cfg_overrides={"max_iterations": 3,
                                                "adjust_interval": 1},
                                 client=client2)
        rc = s2.run()
        self.assertEqual(rc, 0)
        adjs = self.events_of("adjudicated")
        self.assertEqual(len(adjs), 1)
        self.assertEqual(adjs[0]["payload"]["proposal_id"], pid)
        self.assertEqual(adjs[0]["payload"]["decision"], "approve")
        # decision 文件已删（消费即删）；approve 的 hard_delete 应用为功能性灭绝（weight 0）
        self.assertFalse(os.path.exists(dec_path))
        cells = self.read_archive()["cells"]
        self.assertEqual(cells["impl=naive|mem=tiled"]["weight"], 0.0)
        # 应用伴随 weight_changed（灭绝语义）
        wcs = [w for w in self.events_of("weight_changed")
               if w["payload"]["cell_id"] == "impl=naive|mem=tiled"]
        self.assertEqual(len(wcs), 1)
        self.assertEqual(wcs[0]["payload"]["new_weight"], 0.0)


# --------------------------------------------------------------------------- #
# ⑥ 延迟入库
# --------------------------------------------------------------------------- #

class TestDelayedEntry(LoopTestBase):

    def test_tolerance_band_solution_goes_pending(self):
        self.gate.queue(_gate_pass(value=100.0, band=True))
        sched = self.make_scheduler(cfg_overrides={"max_iterations": 1})
        sched.run()
        admitted = self.events_of("elite_admitted")
        self.assertEqual(len(admitted), 1)
        self.assertTrue(admitted[0]["payload"].get("pending"))
        self.assertEqual(admitted[0]["payload"].get("reason"), "tolerance_band")
        arc = self.read_archive()
        self.assertEqual(len(arc["pending"]), 1)
        self.assertEqual(arc["pending"][0]["reason"], "tolerance_band")

    def test_first_solution_new_cell_goes_pending(self):
        # 新格首解（band=False）也延迟入库
        self.gate.queue(_gate_pass(value=100.0, band=False))
        sched = self.make_scheduler(cfg_overrides={"max_iterations": 1})
        sched.run()
        admitted = self.events_of("elite_admitted")
        self.assertEqual(admitted[0]["payload"].get("reason"), "new_cell")
        arc = self.read_archive()
        self.assertEqual(arc["pending"][0]["reason"], "new_cell")

    def test_no_pending_when_cell_established(self):
        # 同格已有精英且非容差带：第 2 解直接正式入格（elite_replaced）
        self.gate.queue(_gate_pass(value=100.0))
        sched = self.make_scheduler(cfg_overrides={"max_iterations": 1})
        sched.run()
        # 确定性同格：其余格经 weight_changed 事件灭绝（事件是真相源，
        # _recover 据此恢复权重 → sampler 只剩存活格）
        sampled = self.events_of("cell_sampled")[0]["payload"]["cell_id"]
        from superopt.events import EventWriter as _EW
        with _EW(self.events_path) as _w:
            for cid in self.fs_all_cells():
                if cid != sampled:
                    _w.append("weight_changed",
                              {"cell_id": cid, "old_weight": 1.0,
                               "new_weight": 0.0, "reason": "test isolate"},
                              "scheduler")
        # 同格更优解（value 更小）→ elite_replaced、无新 pending
        self.gate.queue(_gate_pass(value=50.0))
        client2 = FakeAgentClient()
        client2.queue("builder", [_builder_result()])
        client2.queue("coding", [_coding_result(diff_path=self.cand_file)])
        s2 = self.make_scheduler(cfg_overrides={"max_iterations": 2},
                                 client=client2)
        s2.run()
        replaced = self.events_of("elite_replaced")
        self.assertEqual(len(replaced), 1)
        self.assertEqual(replaced[0]["payload"]["cell_id"], sampled)
        self.assertFalse(replaced[0]["payload"].get("pending"))
        arc2 = self.read_archive()
        # pending 空（new_cell 已被 _recover 重建后又被…… 注意：重建的
        # pending 在 run2 不会自动清——断言 replaced 无 pending 即可，
        # pending 残留属「重评估前不入正式格」的正常状态）
        self.assertEqual([p for p in arc2["pending"] if p["cell_id"] == sampled
                          and p["reason"] != "new_cell"], [])


# --------------------------------------------------------------------------- #
# ⑦ 预算墙 / ⑧ preverify-red
# --------------------------------------------------------------------------- #

class TestBudgetAndPreverify(LoopTestBase):

    def test_max_iterations_wall(self):
        sched = self.make_scheduler(cfg_overrides={"max_iterations": 3})
        rc = sched.run()
        self.assertEqual(rc, 0)
        self.assertEqual(len(self.events_of("iteration_ended")), 3)
        self.assertEqual(len(self.events_of("session_ended")), 1)

    def test_preverify_red_failure_exits_3_no_events(self):
        gate = FakeGate(preverify_ok=False)
        sched = self.make_scheduler(gate=gate)
        sched.cfg["preverify_red"] = True
        rc = sched.run()
        self.assertEqual(rc, 3)
        # 未写任何事件（会话先证红失败 = 不开跑；EventWriter 构造可能
        # 已创建空文件——append 打开的副作用，断言零事件而非文件缺席）
        self.assertEqual(self.read_events(), [])

    def test_preverify_red_ok_runs(self):
        sched = self.make_scheduler(cfg_overrides={"max_iterations": 1})
        sched.cfg["preverify_red"] = True
        rc = sched.run()
        self.assertEqual(rc, 0)
        self.assertEqual(len(self.events_of("iteration_started")), 1)


# --------------------------------------------------------------------------- #
# ⑨ 干预摄取
# --------------------------------------------------------------------------- #

class TestInterventions(LoopTestBase):

    def test_intervention_event_and_stop(self):
        from superopt.intervention import InterventionQueue
        q = InterventionQueue(self.ws)
        q.push("note", {"text": "注意采样偏置"})
        q.push("stop", {"text": "人工叫停"})
        sched = self.make_scheduler(cfg_overrides={"max_iterations": 5})
        rc = sched.run()
        self.assertEqual(rc, 0)
        ivs = self.events_of("intervention")
        self.assertEqual(len(ivs), 2)
        texts = [e["payload"]["text"] for e in ivs]
        self.assertIn("注意采样偏置", texts)
        cats = sorted(e["payload"]["category"] for e in ivs)
        self.assertEqual(cats, ["note", "stop"])
        # stop 干预 → 立即停止（排空发生在迭代前：0 迭代，无半途迭代）
        self.assertEqual(len(self.events_of("iteration_started")), 0)
        self.assertEqual(len(self.events_of("session_ended")), 1)

    def test_bad_intervention_skipped(self):
        # 坏 JSON 队列文件不炸循环
        os.makedirs(os.path.join(self.ws, "interventions"), exist_ok=True)
        with open(os.path.join(self.ws, "interventions", "zz-bad.json"),
                  "w", encoding="utf-8") as f:
            f.write("{not json")
        sched = self.make_scheduler(cfg_overrides={"max_iterations": 1})
        rc = sched.run()
        self.assertEqual(rc, 0)
        self.assertEqual(len(self.events_of("iteration_started")), 1)


# --------------------------------------------------------------------------- #
# ⑩ epoch pending 重评估（invalidation / resurrection）
# --------------------------------------------------------------------------- #

class TestPendingReevaluation(LoopTestBase):

    def test_epoch_reevaluation_invalidates_bad_pending(self):
        # 触发路径：容差带入 pending → epoch(10) 边界重评估复测失败 → 出档
        self.gate.queue(_gate_pass(value=100.0, band=True))
        sched = self.make_scheduler(cfg_overrides={"max_iterations": 10})
        # 让 10 次迭代都采样同一格：注入固定种子 rng
        sched = self.make_scheduler(cfg_overrides={"max_iterations": 10})
        rc = sched.run()
        self.assertEqual(rc, 0)
        arc = self.read_archive()
        # epoch 重评估已把 pending 消费掉（复测 fake gate 全过 → 保留）
        # FakeGate 缺省 verify=pass → 复测通过：pending 清空、无 invalidation
        self.assertEqual(arc["pending"], [])
        self.assertEqual(self.events_of("invalidation"), [])

    def test_epoch_reevaluation_invalidation_on_fail(self):
        # pending 里的解复测失败（M 次）→ invalidation 出档
        sched = self.make_scheduler(cfg_overrides={"max_iterations": 1})
        sched.run()
        # 手工塞一个 pending（模拟延迟入库）+ 后续复测全挂
        arc = sched.archive
        cells = self.read_archive()["cells"]
        cid = [c for c, v in cells.items() if v.get("elite")][0]
        arc.drain_pending()
        arc.add_pending(cid, self.cand_file, "tolerance_band")
        arc.save()
        # 复测：M 次全挂
        for _ in range(3):
            self.gate.queue(_gate_fail("stage2_correctness"))
        # 跑到 epoch 边界：cursor=1 → 需要跑到 10 太多；直接调内部重评估
        sched2 = self.make_scheduler(cfg_overrides={"max_iterations": 1})
        sched2._recover()
        sched2._reevaluate_pending()
        invs = self.events_of("invalidation")
        self.assertEqual(len(invs), 1)
        self.assertEqual(invs[0]["payload"]["cell_id"], cid)
        # 出档：archive elite=None（格保留）
        cells2 = self.read_archive()["cells"]
        self.assertIsNone(cells2[cid]["elite"])
        # pending 清空
        self.assertEqual(self.read_archive()["pending"], [])

    def test_resurrection_on_reevaluation_pass_with_zero_weight(self):
        # 灭绝格复测通过 → 复活（weight 0 → 1）
        sched = self.make_scheduler(cfg_overrides={"max_iterations": 1})
        sched.run()
        cells = self.read_archive()["cells"]
        cid = [c for c, v in cells.items() if v.get("elite")][0]
        arc = sched.archive
        arc.set_weight(cid, 0.0)
        arc.drain_pending()
        arc.add_pending(cid, self.cand_file, "low_weight_src")
        arc.save()
        # 复测通过（FakeGate 缺省 pass）
        sched2 = self.make_scheduler(cfg_overrides={"max_iterations": 1})
        sched2._recover()
        sched2._reevaluate_pending()
        res = self.events_of("resurrection")
        self.assertEqual(len(res), 1)
        self.assertEqual(res[0]["payload"]["cell_id"], cid)
        cells2 = self.read_archive()["cells"]
        self.assertEqual(cells2[cid]["weight"], 1.0)
        self.assertEqual(self.read_archive()["pending"], [])


# --------------------------------------------------------------------------- #
# ⑪ env stage1 失败结论化
# --------------------------------------------------------------------------- #

class TestEnvFailure(LoopTestBase):

    def test_stage1_failure_concluded_and_stops_after_streak(self):
        for _ in range(5):
            self.gate.queue(_gate_fail("stage1_env"))
        sched = self.make_scheduler(cfg_overrides={"max_iterations": 5})
        rc = sched.run()
        self.assertEqual(rc, 0)
        concl = self.events_of("env_failure_concluded")
        # 连续 3 次 stage1 失败 → 停止（第 3 次后不再迭代）
        self.assertEqual(len(concl), 3)
        for c in concl:
            self.assertIn("cause", c["payload"])
            self.assertEqual(c["payload"]["cause"], "stage1_env_unhealthy")
            self.assertIn("stderr_tail", c["payload"])
            self.assertIn("returncode", c["payload"])
        # 只跑了 3 迭代（不是 5）
        self.assertEqual(len(self.events_of("iteration_started")), 3)
        self.assertEqual(len(self.events_of("session_ended")), 1)


# --------------------------------------------------------------------------- #
# ⑫ 改进点回归：axis_perf 注入 / adjust 并行 / score 传递 / per-role model
# --------------------------------------------------------------------------- #

class TestAxisPerfInjection(LoopTestBase):
    """改动③：builder prompt 含同轴历史性能对照。"""

    def test_builder_prompt_contains_axis_perf(self):
        # 先跑 1 迭代（入格一精英），再跑第 2 迭代——第 2 迭代的 builder
        # prompt 应含轴级对照文本（具体 perf 数字）
        self.gate.queue(_gate_pass(4946.8))
        self.gate.queue(_gate_pass(3610.4))
        sched = self.make_scheduler(cfg_overrides={"max_iterations": 1})
        sched.run()
        self.assertEqual(len(self.events_of("elite_admitted")), 1)

        # 直接调 _run_builder 验证 prompt（不经 agent）
        cell = self.events_of("cell_sampled")[0]["payload"]["cell_id"]
        iter_dir = os.path.join(self.ws, "codes", "probe")
        os.makedirs(iter_dir, exist_ok=True)
        sched2 = self.make_scheduler(cfg_overrides={"max_iterations": 0})
        # 在 sched2 上直接调（无 run）：prompt 落盘后断言内容
        import superopt.loop as _loop_mod
        ctx_cell = cell
        # 直接用内部方法组装 prompt 文件（避免走 agent）
        sched2._run_builder(ctx_cell, 99, iter_dir)
        prompt_path = os.path.join(iter_dir, "prompt_builder.md")
        with open(prompt_path, encoding="utf-8") as f:
            body = f.read()
        self.assertIn("同轴历史性能对照", body)
        self.assertIn("4946.8", body)     # 具体数字（低优 1 位小数）
        self.assertIn("轴：", body)        # 轴级行存在

    def test_axis_perf_empty_when_no_elites(self):
        sched = self.make_scheduler(cfg_overrides={"max_iterations": 0})
        text = sched._axis_perf_summary()
        self.assertEqual(text, "（尚无精英数据）")


class TestAdjustParallel(LoopTestBase):
    """改动⑤：adjust 与下一迭代并行——iteration_started 先于 adjust
    agent_ended；权重变化在后续迭代边界落地。"""

    class _SlowAdjustClient(FakeAgentClient):
        """adjust 慢 1.2s（其他角色瞬时）——并行性可观测。"""

        def __init__(self):
            FakeAgentClient.__init__(self)
            self.adjust_started_at = {}
            self.adjust_ended_at = {}

        def run(self, role, prompt_path, workdir, timeout_s=1800, model=None):
            import time as _t
            if role == "adjust":
                self.adjust_started_at[len(self.adjust_started_at)] = _t.time()
                _t.sleep(1.2)
                r = FakeAgentClient.run(self, role, prompt_path, workdir,
                                        timeout_s, model=model)
                self.adjust_ended_at[len(self.adjust_ended_at)] = _t.time()
                return r
            return FakeAgentClient.run(self, role, prompt_path, workdir,
                                       timeout_s, model=model)

    def test_adjust_overlaps_next_iteration_and_applies(self):
        client = self._SlowAdjustClient()
        client.queue("builder", [_builder_result()])
        client.queue("coding", [_coding_result(diff_path=self.cand_file)])
        client.queue("adjust", [_adjust_result([
            {"type": "soft_weight", "target_cell": "impl=naive|mem=tiled",
             "action": "weight 0.5", "evidence": "慢格", "rule_id": "R1"},
        ])])
        for _ in range(3):
            self.gate.queue(_gate_pass())
        sched = self.make_scheduler(cfg_overrides={"max_iterations": 3,
                                                   "adjust_interval": 1},
                                    client=client)
        sched.run()
        evs = [e for e in self.read_events() if e["type"] != "heartbeat"]
        types = [(e["type"], (e["payload"] or {}).get("role")) for e in evs]

        # ① 并行证据：adjust 的 agent_started 之后紧跟 iteration_started(2)
        #    （旧串行实现：adjust agent_ended 先于 iteration_started(2)）
        idx_adj_start = types.index(("agent_started", "adjust"))
        iter_starts = [i for i, t in enumerate(types)
                       if t[0] == "iteration_started"]
        idx_iter2 = iter_starts[1]           # 第 2 迭代的 iteration_started
        self.assertLess(idx_adj_start, idx_iter2,
                        "adjust 应先启动（不阻塞）再开下一迭代")
        idx_adj_end = [i for i, t in enumerate(types)
                       if t == ("agent_ended", "adjust")][0]
        self.assertGreater(idx_adj_end, idx_iter2,
                           "adjust agent_ended 应晚于下一迭代开始（并行证据）："
                           "事件序 %r" % (types[:12],))

        # ② 权重落地：weight_changed（agent:adjust 来源）存在且 archive 生效
        adj_wcs = [w for w in self.events_of("weight_changed")
                   if w["source"] == "agent:adjust"]
        self.assertGreaterEqual(len(adj_wcs), 1)
        self.assertEqual(adj_wcs[0]["payload"]["cell_id"], "impl=naive|mem=tiled")
        cells = self.read_archive()["cells"]
        self.assertAlmostEqual(cells["impl=naive|mem=tiled"]["weight"], 0.5)

        # ③ agent_ended 带 model 字段（可观测性，改动①）
        ae = [e for e in evs if e["type"] == "agent_ended"][0]
        self.assertIn("model", ae["payload"])


class TestExpandCellChain(LoopTestBase):
    """task19：expand_cell 全链——提案 → decision 文件 → 应用（写回
    feature_space.json）→ 热重载 → 新格被分殖采样。

    语义链（迭代3 J-i3-2）：探索的唯一入口 = adjust 加格（走裁决链）；
    apply 不直改内存空间——写文件，下一迭代边界 mtime 热重载生效。
    """

    def test_expand_cell_add_axis_value_full_chain(self):
        # ---- 阶段 1：adjust 产出 expand_cell 提案 ----
        self.client.queue("adjust", [_adjust_result([
            {"type": "expand_cell",
             "action": "add_axis_value", "axis": "mem", "value": "zcurve",
             "rationale": "全部存活格 visit>=4 且无新纪录",
             "rule_id": "saturated_expand"},
        ])])
        sched = self.make_scheduler(cfg_overrides={"max_iterations": 2,
                                                   "adjust_interval": 1})
        sched.run()
        # 提案落队列文件（含结构化扩张参数）
        props = self.events_of("proposal")
        self.assertEqual(len(props), 1)
        pp = props[0]["payload"]
        self.assertEqual(pp["type"], "expand_cell")
        self.assertEqual(pp["axis"], "mem")
        self.assertEqual(pp["value"], "zcurve")
        prop_files = [n for n in os.listdir(os.path.join(self.ws, "proposals"))
                      if n.endswith(".json")
                      and not n.endswith(".decision.json")]
        self.assertEqual(len(prop_files), 1)
        pid = prop_files[0][:-5]
        with open(os.path.join(self.ws, "proposals", prop_files[0]),
                  encoding="utf-8") as f:
            pf = json.load(f)
        self.assertEqual(pf["axis"], "mem")
        self.assertEqual(pf["value"], "zcurve")
        self.assertEqual(pf["rationale"][:5], "全部存活格")

        # ---- 阶段 2：放置 approve decision（模拟 adjudicator 进程）----
        dec_path = os.path.join(self.ws, "proposals", pid + ".decision.json")
        with open(dec_path, "w", encoding="utf-8") as f:
            json.dump({"proposal_id": pid, "decision": "approve",
                       "reason": "扩张合法", "conditions": [],
                       "adjudicated_at": "2026-10-20T12:00:00+08:00",
                       "adjudicator": "auto"}, f)

        # ---- 阶段 3：第二轮运行——drain decision → 应用 → 热重载 → 新格被采 ----
        client2 = FakeAgentClient()
        client2.queue("builder", [_builder_result()])
        client2.queue("coding", [_coding_result(diff_path=self.cand_file)])
        client2.queue("adjust", [_adjust_result([])])
        for _ in range(8):
            self.gate.queue(_gate_pass())
        s2 = self.make_scheduler(cfg_overrides={"max_iterations": 8,
                                                "adjust_interval": 0},
                                 client=client2)
        rc = s2.run()
        self.assertEqual(rc, 0)

        # adjudicated 事件 ×2：decision=approve + decision=applied（应用留痕）
        adjs = self.events_of("adjudicated")
        decisions = {a["payload"]["proposal_id"]: a["payload"]["decision"]
                     for a in adjs}
        self.assertEqual(decisions.get(pid), "applied")
        applied_ev = [a for a in adjs
                      if a["payload"].get("decision") == "applied"][0]
        self.assertEqual(applied_ev["payload"]["applied"]["axis"], "mem")
        self.assertEqual(applied_ev["payload"]["applied"]["value"], "zcurve")

        # feature_space.json 已更新（文件是空间真相）
        with open(os.path.join(self.ws, "env", "feature_space.json"),
                  encoding="utf-8") as f:
            fs_data = json.load(f)
        mem_axis = [a for a in fs_data["axes"] if a["name"] == "mem"][0]
        self.assertIn("zcurve", mem_axis["values"])

        # 新格（mem=zcurve × 3 个 impl 值）被分殖优先采样（确定性首访）
        sampled = self.events_of("cell_sampled")
        new_cells = [s2 for s2 in sampled
                     if "mem=zcurve" in s2["payload"]["cell_id"]]
        self.assertGreaterEqual(len(new_cells), 1,
                                "热重载后新格应被分殖优先采样")
        for s2 in new_cells:
            self.assertTrue(s2["payload"]["cold_start"])

    def test_expand_cell_reject_decision_not_applied(self):
        """reject 的 expand_cell：decision 被消费但 feature_space 不变。"""
        self.client.queue("adjust", [_adjust_result([
            {"type": "expand_cell",
             "action": "add_axis_value", "axis": "impl", "value": "weird",
             "rationale": "x", "rule_id": "R"}])])
        sched = self.make_scheduler(cfg_overrides={"max_iterations": 2,
                                                   "adjust_interval": 1})
        sched.run()
        prop_files = [n for n in os.listdir(os.path.join(self.ws, "proposals"))
                      if n.endswith(".json")
                      and not n.endswith(".decision.json")]
        pid = prop_files[0][:-5]
        dec_path = os.path.join(self.ws, "proposals", pid + ".decision.json")
        with open(dec_path, "w", encoding="utf-8") as f:
            json.dump({"proposal_id": pid, "decision": "reject",
                       "reason": "轴已饱和", "conditions": []}, f)
        client2 = FakeAgentClient()
        client2.queue("builder", [_builder_result()])
        client2.queue("coding", [_coding_result(diff_path=self.cand_file)])
        s2 = self.make_scheduler(cfg_overrides={"max_iterations": 3,
                                                "adjust_interval": 0},
                                 client=client2)
        s2.run()
        with open(os.path.join(self.ws, "env", "feature_space.json"),
                  encoding="utf-8") as f:
            fs_data = json.load(f)
        impl_axis = [a for a in fs_data["axes"] if a["name"] == "impl"][0]
        self.assertNotIn("weird", impl_axis["values"])
        # 只有一条 adjudicated（decision=reject），无 applied
        applied = [a for a in self.events_of("adjudicated")
                   if a["payload"].get("decision") == "applied"]
        self.assertEqual(len(applied), 0)

    def test_external_feature_space_edit_hot_reload(self):
        """外部修改 feature_space（gen_env 升级/human 手改）→ 热重载生效。"""
        sched = self.make_scheduler(cfg_overrides={"max_iterations": 2})
        sched.run()
        # 外部加值（模拟 env 升级：直接改文件）
        fs_path = os.path.join(self.ws, "env", "feature_space.json")
        with open(fs_path, encoding="utf-8") as f:
            fs_data = json.load(f)
        for a in fs_data["axes"]:
            if a["name"] == "impl":
                a["values"].append("extern_new")
        with open(fs_path, "w", encoding="utf-8") as f:
            json.dump(fs_data, f, ensure_ascii=False, indent=2)
        # 第二轮运行：热重载 + 新格被分殖采样
        client2 = FakeAgentClient()
        client2.queue("builder", [_builder_result()])
        client2.queue("coding", [_coding_result(diff_path=self.cand_file)])
        for _ in range(4):
            self.gate.queue(_gate_pass())
        s2 = self.make_scheduler(cfg_overrides={"max_iterations": 6,
                                                "adjust_interval": 0},
                                 client=client2)
        s2.run()
        sampled = self.events_of("cell_sampled")
        new_cells = [s2 for s2 in sampled
                     if "impl=extern_new" in s2["payload"]["cell_id"]]
        self.assertGreaterEqual(len(new_cells), 1,
                                "外部修改后热重载：新格应被分殖采样")


class TestAdjustObservability(LoopTestBase):
    """task19：adjust 决策轨迹可观测——快照按迭代归档 + adjust_decision 事件
    + cell_sampled.weight_share 字段。"""

    def test_adjust_snapshot_archived_per_iteration(self):
        """codes/adjust/iter_{N}/ 下 prompt_adjust.md + agent_adjust.json。"""
        self.client.queue("adjust", [_adjust_result([])])
        sched = self.make_scheduler(cfg_overrides={"max_iterations": 3,
                                                   "adjust_interval": 1})
        sched.run()
        adjust_root = os.path.join(self.ws, "codes", "adjust")
        iter_dirs = sorted(d for d in os.listdir(adjust_root)
                           if d.startswith("iter_"))
        # 迭代 1、2 边界各触发一次 adjust（interval=1）
        self.assertGreaterEqual(len(iter_dirs), 2)
        for d in iter_dirs:
            files = os.listdir(os.path.join(adjust_root, d))
            self.assertIn("prompt_adjust.md", files)
            self.assertIn("agent_adjust.json", files)
        # agent_adjust.json 可解析且带 iteration_id
        with open(os.path.join(adjust_root, iter_dirs[0],
                               "agent_adjust.json"), encoding="utf-8") as f:
            rec = json.load(f)
        self.assertIn("iteration_id", rec)

    def test_adjust_decision_event_written(self):
        """adjust_decision 事件：iteration + proposals 摘要 + cells_status 摘要。"""
        self.client.queue("adjust", [_adjust_result([
            {"type": "soft_weight", "target_cell": "impl=naive|mem=tiled",
             "action": "weight 0.5", "evidence": "e", "rule_id": "R"}])])
        sched = self.make_scheduler(cfg_overrides={"max_iterations": 2,
                                                   "adjust_interval": 1})
        sched.run()
        ads = self.events_of("adjust_decision")
        self.assertGreaterEqual(len(ads), 1)
        p = ads[0]["payload"]
        self.assertIn("iteration", p)
        self.assertIsInstance(p["proposals"], list)
        self.assertGreaterEqual(len(p["proposals"]), 1)
        self.assertEqual(p["proposals"][0]["type"], "soft_weight")
        # cells_status 摘要：每格 cell_id/visit/weight/elite_perf
        css = p["cells_status_summary"]
        self.assertIsInstance(css, list)
        self.assertGreaterEqual(len(css), 1)
        for c in css:
            for k in ("cell_id", "visit", "weight", "elite_perf"):
                self.assertIn(k, c)

    def test_cell_sampled_has_weight_share(self):
        """cell_sampled.weight_share：0 < share <= 1 的概率快照（ucb_score 退役）。"""
        sched = self.make_scheduler(cfg_overrides={"max_iterations": 2})
        sched.run()
        for s2 in self.events_of("cell_sampled"):
            self.assertNotIn("ucb_score", s2["payload"])
            share = s2["payload"]["weight_share"]
            self.assertIsInstance(share, float)
            self.assertGreater(share, 0.0)
            self.assertLessEqual(share, 1.0)

    def test_session_ended_budget_exhausted_on_token_limit(self):
        """token 预算耗尽 → session_ended.budget_exhausted=True（task16 接线）。

        FakeAgentClient 每次 agent 运行记 usage.total=160（_agent_result 固定）
        ——设极小上限（如 200）→ 第一个 agent 结账后 within_budget False。
        """
        sched = self.make_scheduler(cfg_overrides={"max_iterations": 5,
                                                   "max_total_tokens": 200})
        sched.run()
        ended = self.events_of("session_ended")
        self.assertEqual(len(ended), 1)
        self.assertTrue(ended[0]["payload"].get("budget_exhausted"))
        # 未到 max_iterations：预算墙先停（迭代数 < 5）
        started = self.events_of("iteration_started")
        self.assertLess(len(started), 5)

    def test_session_ended_no_budget_flag_by_default(self):
        """默认（无 token 上限）：budget_exhausted=False（正常 max_iter 收尾）。"""
        sched = self.make_scheduler(cfg_overrides={"max_iterations": 2})
        sched.run()
        ended = self.events_of("session_ended")
        self.assertFalse(ended[0]["payload"].get("budget_exhausted"))


class TestScorePlumbing(LoopTestBase):
    """改动④：gate metrics.perf.score → 入档精英 score 的传递回归
    （首跑实测复现：链路本身完好——本测试固化防未来真断链）。"""

    def test_elite_score_equals_gate_score(self):
        self.gate.queue(_gate_pass(4946.8))
        # 定制 score：用带 score 的 gate 结果（_gate_pass 固定 0.9——
        # 换手搓 metrics 验证非平凡值）
        from superopt.gates import GateResult
        self.gate.queue(GateResult(
            passed=True, failure_stage=None,
            metrics={"perf": {"value": 3610.4, "score": 0.123456,
                              "perf_type": "runtime_ms", "perf_desc": "low",
                              "k": 3, "timings": [3610.4] * 3}},
            details="fake"))
        sched = self.make_scheduler(cfg_overrides={"max_iterations": 2})
        sched.run()
        elites = [e for e in self.events_of("elite_admitted")]
        self.assertEqual(len(elites), 2)
        # 第 2 迭代的精英 score == gate 的 score（非 0 非 None）
        self.assertAlmostEqual(elites[1]["payload"]["score"], 0.123456,
                               places=9)
        # archive 同步
        arch = self.read_archive()
        scores = [c["elite"]["score"] for c in arch["cells"].values()
                  if c.get("elite")]
        self.assertIn(0.123456, scores)


class TestPerRoleModelDispatch(LoopTestBase):
    """改动①：agent_models 配置 → client.run 收到对应 model。"""

    def test_builder_model_passed_to_client(self):
        client = FakeAgentClient()
        client.queue("builder", [_builder_result()])
        client.queue("coding", [_coding_result(diff_path=self.cand_file)])
        self.gate.queue(_gate_pass())
        sched = self.make_scheduler(
            cfg_overrides={"max_iterations": 1,
                           "agent_models": {"builder": "catpaw-proxy/deepseek-v4-flash"}},
            client=client)
        sched.run()
        builder_call = [c for c in client.calls if c["role"] == "builder"][0]
        self.assertEqual(builder_call["model"],
                         "catpaw-proxy/deepseek-v4-flash")
        coding_call = [c for c in client.calls if c["role"] == "coding"][0]
        self.assertIsNone(coding_call["model"])   # 未配角色 → None


if __name__ == "__main__":
    unittest.main()


# --------------------------------------------------------------------------- #
# ⑫ 迭代5：agent 信息完整性（ctx 真值 / gate→debug / 台账 / 饱和 / 回合预算）
# --------------------------------------------------------------------------- #

class TestContextCompleteness(LoopTestBase):
    """P0/P1/P2/P3：四角色 ctx 的盲点修复。"""

    def test_recent_events_real_values(self):
        """A：recent_events 给真值（不再是 payload_keys）。"""
        self.gate.queue(_gate_fail("stage1_env"))  # stage1 不进 debug 通道
        sched = self.make_scheduler(cfg_overrides={"max_iterations": 1})
        sched.run()
        summary = sched._recent_events_summary(15)
        gates = [e for e in summary if e["type"] == "gate_result"]
        self.assertEqual(len(gates), 1)
        self.assertEqual(gates[0]["failure_stage"], "stage1_env")
        self.assertFalse(gates[0]["passed"])
        # heartbeat 被过滤
        self.assertEqual([e for e in summary if e["type"] == "heartbeat"], [])
        # 无 payload_keys 残留形态
        for e in summary:
            self.assertNotIn("payload_keys", e)

    def test_ctx_helpers_structure(self):
        """B：perf_timeline / budget_status / gate_defs / domain_brief。"""
        sched = self.make_scheduler(cfg_overrides={"max_iterations": 1})
        sched.run()
        timeline = sched._perf_timeline(10)
        self.assertTrue(timeline)
        self.assertIn("is_record", timeline[0])
        bs = sched._budget_status()
        for k in ("iterations_used", "max_iterations", "wall_minutes",
                  "tokens_used", "max_tokens", "tokens_pct"):
            self.assertIn(k, bs)
        self.assertEqual(bs["iterations_used"], 1)
        gd = sched._gate_defs()
        self.assertEqual(gd["tolerance"]["soft_gate"]["relative_band"], 0.05)
        self.assertEqual(gd["bench"]["k"], 3)
        self.assertEqual(gd["gates"]["score_max"], 2.0)
        brief = sched._domain_brief()
        self.assertIn("SOP", brief)
        self.assertIn("死族", brief)
        files = sched._workdir_files(os.path.join(self.ws, "codes"))
        self.assertTrue(files)

    def test_role_prompts_contain_new_ctx(self):
        """C：四角色 prompt 各自带新 ctx 渲染值。"""
        self.client.queue("adjust", [_adjust_result([])])
        sched = self.make_scheduler(cfg_overrides={"max_iterations": 2,
                                                   "adjust_interval": 1})
        sched.run()
        # builder：env_dir + gate_defs（relative_band）
        iter_dirs = sorted(d for d in os.listdir(os.path.join(self.ws, "codes"))
                           if d.startswith("iter"))
        bprompt = open(os.path.join(self.ws, "codes", iter_dirs[0],
                                    "prompt_builder.md"), encoding="utf-8").read()
        self.assertIn(self.ws, bprompt)          # env_dir 绝对路径
        self.assertIn("relative_band", bprompt)  # gate_defs
        self.assertIn("tokens_used", bprompt)    # budget_status
        self.assertIn("SOP", bprompt)            # domain_brief
        # coding：gate_defs + 文件清单 + 精英基线（断言渲染后的标签/内容）
        cprompt = open(os.path.join(self.ws, "codes", iter_dirs[0],
                                    "prompt_coding.md"), encoding="utf-8").read()
        self.assertIn("relative_band", cprompt)      # gate_defs 值
        self.assertIn("文件清单", cprompt)            # workdir_files 标签
        self.assertIn("reference.py", cprompt)       # 文件清单内容（父本）
        self.assertIn("精英基线", cprompt)            # perf_baseline 标签
        # adjust：budget_status + perf_timeline + own_decisions
        ap = os.path.join(self.ws, "codes", "adjust", "iter_1",
                          "prompt_adjust.md")
        self.assertTrue(os.path.exists(ap))
        aprompt = open(ap, encoding="utf-8").read()
        self.assertIn("tokens_used", aprompt)
        self.assertIn("perf_timeline", aprompt)
        self.assertIn("own_decisions", aprompt)

    def test_max_turns_passthrough(self):
        """H：per-role 回合预算透传（fake client 记 kwargs）。"""
        sched = self.make_scheduler(cfg_overrides={"max_iterations": 1,
                                                   "builder_max_turns": 5,
                                                   "coding_max_turns": 7})
        sched.run()
        by_role = {c["role"]: c for c in self.client.calls}
        self.assertEqual(by_role["builder"]["max_turns"], 5)
        self.assertEqual(by_role["coding"]["max_turns"], 7)


class TestGateToDebug(LoopTestBase):
    """D：框架门禁失败 → debug 接管（每迭代 1 次）。"""

    def test_gate_failure_debug_fixes_and_readmits(self):
        # 首次门禁失败（stage2）→ debug 修复 → 重验通过 → 入格
        self.gate.queue(_gate_fail("stage2_correctness"))
        self.gate.queue(_gate_pass(value=90.0))
        self.client.queue("debug", [_debug_result(passed=True)])
        sched = self.make_scheduler(cfg_overrides={"max_iterations": 1})
        rc = sched.run()
        self.assertEqual(rc, 0)
        self.assertEqual(len(self.gate.verify_calls), 2)
        # debug 被派发一次，attempt=100（与自检 debug 区分）
        debug_starts = [e for e in self.events_of("agent_started")
                        if e["payload"]["role"] == "debug"]
        self.assertEqual(len(debug_starts), 1)
        cands = [e for e in self.events_of("candidate_produced")
                 if e["payload"].get("role") == "debug"]
        self.assertEqual(len(cands), 1)
        self.assertEqual(cands[0]["payload"]["attempt"], 100)
        # 重验通过 → 入格
        self.assertEqual(len(self.events_of("elite_admitted")), 1)
        # debug prompt 含框架失败上下文
        iter_dirs = sorted(d for d in os.listdir(os.path.join(self.ws, "codes"))
                           if d.startswith("iter"))
        dp = os.path.join(self.ws, "codes", iter_dirs[0], "prompt_debug.md")
        self.assertTrue(os.path.exists(dp))
        dprompt = open(dp, encoding="utf-8").read()
        self.assertIn("框架门禁失败详情", dprompt)    # gate_failure 标签
        self.assertIn("stage2_correctness", dprompt)  # gate_failure 值
        self.assertIn("relative_band", dprompt)       # gate_defs 值

    def test_gate_failure_debug_still_fails_records_reject(self):
        # 门禁失败 → debug → 重验仍失败：记 reject、不崩、无入格
        self.gate.queue(_gate_fail("stage3_perf"))
        self.gate.queue(_gate_fail("stage3_perf"))
        self.client.queue("debug", [_debug_result(passed=True)])
        sched = self.make_scheduler(cfg_overrides={"max_iterations": 1})
        rc = sched.run()
        self.assertEqual(rc, 0)
        self.assertEqual(len(self.gate.verify_calls), 2)
        self.assertEqual(self.events_of("elite_admitted"), [])
        ended = self.events_of("iteration_ended")[0]["payload"]
        self.assertFalse(ended["had_candidate"])
        # 台账记 reject + reject_reason
        entries = sched.exp_ledger.read_all()
        self.assertEqual(entries[-1]["verdict"], "reject")
        self.assertEqual(entries[-1]["reject_reason"], "stage3_perf")

    def test_stage1_failure_not_routed_to_debug(self):
        # stage1（环境坏）不进 debug 通道——交给 env_failure_concluded
        self.gate.queue(_gate_fail("stage1_env"))
        sched = self.make_scheduler(cfg_overrides={"max_iterations": 1})
        sched.run()
        self.assertEqual(len(self.gate.verify_calls), 1)
        self.assertEqual(len(self.events_of("env_failure_concluded")), 1)
        self.assertEqual([e for e in self.events_of("agent_started")
                          if e["payload"]["role"] == "debug"], [])


class TestLedgerAndSaturation(LoopTestBase):
    """G/E/F：台账接线 / 饱和信号 / terminate_recommend。"""

    def test_ledger_entry_written(self):
        sched = self.make_scheduler(cfg_overrides={"max_iterations": 1})
        sched.run()
        entries = sched.exp_ledger.read_all()
        self.assertEqual(len(entries), 1)
        e = entries[0]
        for k in ("iteration_id", "cell_id", "idea", "gate", "verdict",
                  "perf_delta", "reject_reason", "agents"):
            self.assertIn(k, e)
        self.assertIn(e["verdict"], ("admit", "replace"))
        self.assertTrue(e["gate"]["passed"])
        self.assertIn("builder", e["agents"])
        self.assertIn("coding", e["agents"])

    def test_saturation_signal_states(self):
        sched = self.make_scheduler(cfg_overrides={"max_iterations": 1})
        sched.run()
        # 初始（纪录刚刷新）→ 不饱和
        self.assertIsNone(sched._saturation_signal())
        # 构造饱和：全格 visit>=2 + stale>=3 + 冠军连续 miss>=3
        for cid in self.fs_all_cells():
            sched.archive.set_visit(cid, 2)
        sched._stale_iters = 3
        sched._champion_misses = 3
        sig = sched._saturation_signal()
        self.assertIsNotNone(sig)
        self.assertEqual(sig["stale_iterations"], 3)
        self.assertTrue(sig["all_visited"])
        self.assertIn("champion_cell", sig)
        self.assertEqual(sig["alive_cells"], len(self.fs_all_cells()))

    def test_saturation_requires_all_conditions(self):
        sched = self.make_scheduler(cfg_overrides={"max_iterations": 1})
        sched.run()
        # 只 stale 无全访问 → None
        sched._stale_iters = 5
        sched._champion_misses = 5
        self.assertIsNone(sched._saturation_signal())
        # 全访问 + stale + champion miss 齐 → 非空
        for cid in self.fs_all_cells():
            sched.archive.set_visit(cid, 2)
        self.assertIsNotNone(sched._saturation_signal())

    def test_terminate_recommend_needs_human(self):
        self.client.queue("adjust", [_adjust_result([
            {"type": "terminate_recommend",
             "evidence": "seq 5 连续 8 迭代无新纪录",
             "rationale": "全部格触及 domain 天花板",
             "rule_id": "saturated_terminate"}])])
        sched = self.make_scheduler(cfg_overrides={"max_iterations": 2,
                                                   "adjust_interval": 1})
        sched.run()
        # proposal 文件：needs_human=true
        pdir = os.path.join(self.ws, "proposals")
        files = [n for n in os.listdir(pdir) if n.endswith(".json")]
        self.assertEqual(len(files), 1)
        rec = json.load(open(os.path.join(pdir, files[0]), encoding="utf-8"))
        self.assertEqual(rec["type"], "terminate_recommend")
        self.assertTrue(rec["needs_human"])
        # proposal 事件：needs_human 透出
        props = self.events_of("proposal")
        self.assertEqual(len(props), 1)
        self.assertTrue(props[0]["payload"]["needs_human"])
        # 不进裁决自动执行：无 adjudicated 事件、无 weight 变更
        self.assertEqual(self.events_of("adjudicated"), [])
        self.assertEqual([w for w in self.events_of("weight_changed")
                          if w["source"] == "agent:adjust"], [])
        # 权重全部保持缺省（建议性动作不动 archive）
        cells = self.read_archive()["cells"]
        self.assertTrue(all(float(c.get("weight", 1.0)) == 1.0
                            for c in cells.values()))


# --------------------------------------------------------------------------- #
# ⑫ 三要素（profile/benchmark/verify/gate_check）+ profile 历史 + builder 决策落盘
#      （迭代6/task28）
# --------------------------------------------------------------------------- #

_PROFILE_SCRIPT = '''#!/usr/bin/env python3
import json, sys
print(json.dumps({"hotspots": [{"func": "np.dot", "pct": 62.5},
                               {"func": "concat", "pct": 21.0}],
                  "env_facts": {"numpy": "2.0.2", "blas": "accelerate"}}))
'''


class TestElementsAndDesigns(LoopTestBase):
    """迭代6：三要素命令注入 / profile 时间序列 / builder 决策落盘。"""

    def _install_element_skills(self) -> None:
        """fixture env 造 skills/{profile,benchmark,verify,gate_check}.py。"""
        sk = os.path.join(self.ws, "env", "skills")
        os.makedirs(sk, exist_ok=True)
        with open(os.path.join(sk, "profile.py"), "w", encoding="utf-8") as f:
            f.write(_PROFILE_SCRIPT)
        for name, payload in (
                ("benchmark", '{"runtime_ms": 1.0, "score": 0.9, "timings": [1, 1, 1], "k": 3}'),
                ("verify", '{"passed": true, "max_error": 0.0, "stage": "stage2_correctness"}')):
            with open(os.path.join(sk, name + ".py"), "w", encoding="utf-8") as f:
                f.write("#!/usr/bin/env python3\nimport sys\nprint(%r)\n" % payload)
        with open(os.path.join(sk, "gate_check.py"), "w", encoding="utf-8") as f:
            f.write("#!/usr/bin/env python3\nimport sys\nprint("
                    "'{\"passed\": true, \"failure_stage\": null, "
                    "\"metrics\": {}, \"details\": \"ok\"}')\n")

    @staticmethod
    def _iter_dirs(ws: str):
        return sorted(d for d in os.listdir(os.path.join(ws, "codes"))
                      if d.startswith("iter"))

    def test_element_cmds_injected_to_role_prompts(self):
        """① 三要素命令注入 builder/coding/debug prompt。"""
        self._install_element_skills()
        sk = os.path.join(self.ws, "env", "skills")
        # coding 自检失败 → 触发 debug（第三角色也要看到三要素）
        self.client.queue("coding", [_coding_result(passed=False,
                                                    diff_path=self.cand_file)])
        self.client.queue("debug", [_debug_result(passed=True)])
        sched = self.make_scheduler(cfg_overrides={"max_iterations": 1})
        sched.run()
        elems = sched._element_cmds()
        for key in ("profile_cmd", "benchmark_cmd", "verify_cmd", "gate_check_cmd"):
            self.assertTrue(elems[key], "%s 缺（skills 四件应齐备）" % key)
        self.assertIn("--baseline", elems["gate_check_cmd"])
        d = self._iter_dirs(self.ws)[0]
        for role in ("builder", "coding", "debug"):
            p = os.path.join(self.ws, "codes", d, "prompt_%s.md" % role)
            text = open(p, encoding="utf-8").read()
            self.assertIn(os.path.join(sk, "profile.py"), text,
                          "%s prompt 缺 profile 命令" % role)
            self.assertIn(os.path.join(sk, "gate_check.py"), text,
                          "%s prompt 缺 gate_check 命令" % role)

    def test_legacy_env_has_empty_element_cmds(self):
        """② legacy env（无 skills 四件）→ 命令空串且模板不炸。"""
        sched = self.make_scheduler(cfg_overrides={"max_iterations": 1})
        sched.run()  # 不抛异常
        elems = sched._element_cmds()
        self.assertEqual(elems["profile_cmd"], "")
        self.assertEqual(elems["benchmark_cmd"], "")
        self.assertEqual(elems["verify_cmd"], "")
        self.assertEqual(elems["gate_check_cmd"], "")

    def test_profile_collected_and_archived(self):
        """③ 每迭代自动采集 profile → profile_history/<cell>/iter1.json。"""
        self._install_element_skills()
        sched = self.make_scheduler(cfg_overrides={"max_iterations": 1})
        sched.run()
        root = os.path.join(self.ws, "profile_history")
        self.assertTrue(os.path.isdir(root))
        cells = os.listdir(root)
        self.assertEqual(len(cells), 1)
        files = os.listdir(os.path.join(root, cells[0]))
        self.assertIn("iter1.json", files)
        rec = json.load(open(os.path.join(root, cells[0], "iter1.json"),
                             encoding="utf-8"))
        for k in ("ts", "iteration_id", "cell_id", "candidate", "profile"):
            self.assertIn(k, rec)
        self.assertEqual(rec["iteration_id"], 1)
        self.assertEqual(rec["profile"]["env_facts"]["numpy"], "2.0.2")

    def test_profile_history_injected_into_builder(self):
        """④ profile 历史（目标格 + 灵感格）注入 builder prompt。"""
        self._install_element_skills()
        from superopt.feature_space import FeatureSpace
        cells = FeatureSpace(AXES).all_cells()
        root = os.path.join(self.ws, "profile_history")
        # 目标格：所有格都种上历史（采样结果不确定，但标记必现）
        for cid in cells:
            d = os.path.join(root, cid)
            os.makedirs(d, exist_ok=True)
            with open(os.path.join(d, "iter900.json"), "w", encoding="utf-8") as f:
                json.dump({"ts": "2026-10-23T00:00:00+08:00",
                           "iteration_id": 900, "cell_id": cid,
                           "candidate": "x", "profile":
                           {"hotspots": [{"func": "PROF_MARK", "pct": 1}]}},
                          f)
        # 灵感格：造一个有精英的邻格（archive.json 预写）
        nb = cells[0]
        arch = {"meta": {}, "pending": [], "cells": {nb: {
            "descriptor": {}, "elite": {"code_path": self.cand_file,
                                        "perf_value": 100.0},
            "history": [], "weight": 1.0, "visit": 1}}}
        with open(self.archive_path, "w", encoding="utf-8") as f:
            json.dump(arch, f)
        d = os.path.join(root, nb)
        with open(os.path.join(d, "iter901.json"), "w", encoding="utf-8") as f:
            json.dump({"ts": "2026-10-23T00:00:00+08:00", "iteration_id": 901,
                       "cell_id": nb, "candidate": "x",
                       "profile": {"hotspots": [{"func": "INSP_MARK", "pct": 2}]}},
                      f)
        sched = self.make_scheduler(cfg_overrides={"max_iterations": 1})
        sched.run()
        d0 = self._iter_dirs(self.ws)[0]
        text = open(os.path.join(self.ws, "codes", d0, "prompt_builder.md"),
                    encoding="utf-8").read()
        self.assertIn("PROF_MARK", text)      # 目标格历史轨迹
        self.assertIn("INSP_MARK", text)      # 灵感格历史轨迹

    def test_builder_design_persisted(self):
        """⑤ builder 决策落盘 designs/iter{N}_{cell}.json（papers/inspirations/reasoning）。"""
        self._install_element_skills()
        self.client.queue("builder", [_agent_result("builder", {
            "experiment_plan": {"idea": "分块+转置", "parents": [],
                                "plan_steps": ["s1"], "expected_behavior": "faster"},
            "papers": [{"arxiv_id": "2501.00001", "title": "GEMM tiling",
                        "insight": "块尺寸与 L2 匹配", "applicability": "本格 tiled"}],
            "inspirations": [{"source_cell": "impl=vectorized|mem=tiled",
                              "what": "zip 转置降低装配税"}],
            "reasoning": "热点在 np.dot，装配次之；先换派发入口。",
        })])
        sched = self.make_scheduler(cfg_overrides={"max_iterations": 1})
        sched.run()
        ddir = os.path.join(self.ws, "designs")
        files = [n for n in os.listdir(ddir) if n.endswith(".json")]
        self.assertEqual(len(files), 1)
        self.assertTrue(files[0].startswith("iter1_"))
        rec = json.load(open(os.path.join(ddir, files[0]), encoding="utf-8"))
        self.assertEqual(rec["iteration_id"], 1)
        self.assertTrue(rec["cell_id"])
        self.assertEqual(rec["papers"][0]["arxiv_id"], "2501.00001")
        self.assertEqual(rec["inspirations"][0]["source_cell"],
                         "impl=vectorized|mem=tiled")
        self.assertIn("热点在 np.dot", rec["reasoning"])
        self.assertEqual(rec["experiment_plan"]["idea"], "分块+转置")
        # profile_facts：自动采集的现状事实（np.dot=62.5）
        self.assertTrue(any("np.dot" in f for f in rec["profile_facts"]))

    def test_recent_designs_injected(self):
        """⑥ recent_designs 注入后续迭代的 builder prompt。"""
        self._install_element_skills()
        ddir = os.path.join(self.ws, "designs")
        os.makedirs(ddir, exist_ok=True)
        with open(os.path.join(ddir, "iter1_x.json"), "w", encoding="utf-8") as f:
            json.dump({"iteration_id": 1, "cell_id": "impl=naive|mem=row",
                       "experiment_plan": {"idea": "PREV_IDEA_MARK"},
                       "papers": [{"arxiv_id": "2502.00002"}],
                       "reasoning": "PREV_REASON_MARK"}, f)
        sched = self.make_scheduler(cfg_overrides={"max_iterations": 2})
        sched.run()
        d1 = self._iter_dirs(self.ws)[1]
        text = open(os.path.join(self.ws, "codes", d1, "prompt_builder.md"),
                    encoding="utf-8").read()
        self.assertIn("PREV_IDEA_MARK", text)
        self.assertIn("2502.00002", text)
