"""loop.py — superopt 调度器心脏：四状态循环 + 崩溃游标恢复（job_47/task8）

循环形态（design-tree §2.5）::

    LOOP（每迭代）:
      drain 干预队列 → intervention 事件（stop 类 → 优雅停止）
      drain 裁决决策 → adjudicated 事件（approve 的 hard_delete 应用为灭绝）
      每 N 迭代 → adjust agent（软动作直接应用 / 硬动作提案走外置裁决；
                    expand_cell 裁决通过后写回 feature_space.json，热重载生效）
      [加权随机采样] increment_visit（唯一递增点，修复 v5 db.py:201 断链；
                    概率 ∝ weight——决策权在 adjust，迭代3 J-i3-1）
      [builder]  选父格 → experiment_plan
      [coding]   codes/{uuid}/ 隔离快照 → 自检 env skills 脚本（仅参考）
        └ 自检不符 → [debug] 同 worktree 修复（≤M 次，每次一条 fix_report
          载体事件 = candidate_produced{role:debug, fix_report}）
      [框架验证] CascadeGate.verify（以框架为准，J6）→ gate_result
        ├ 通过 → MAP-Elites admit（同格替换）+ 延迟入库判定（L2-Q6 三者任一）
        │         → elite_admitted/replaced（payload.pending 供投影重建）
        └ stage1 失败 → env_failure_concluded（诊断包）→ 连续 3 次停止
      每 epoch（EPOCH_ITERATIONS=10）末 → pending 重评估（M 次复测）
        ├ 多数过 → 保留 / 复活（resurrection，weight 0→1）
        └ 多数挂 → invalidation 出档（elite=None，格保留）
      heartbeat（events + heartbeat.json + super.json state/cursor）

架构纪律：
  - **事件先于档案**：events.jsonl 是唯一真相源；archive 是投影（save 滞后
    于事件追加，崩溃后 _recover 从事件流重建 visit/weight/pending）
  - **visit 递增只经 Sampler.increment_visit**（on_visit 回调接
    _RuntimeArchive.bump_visit）——agent 无直改通道（KR5）
  - 门禁阈值判断零出现（CascadeGate 职责，L3-Q2）
  - debug 尝试的事件载体：EVENT_TYPES 无 fix_report 类型（task1 冻结 19 类），
    以 candidate_produced{role:"debug", fix_report:{...}} 承载（测试与
    lineage 消费同一约定）

运行时 Archive 扩展（_RuntimeArchive）：
  task2 的 Archive 承载 visit 字段但无递增方法（visit 的唯一真相在
  cell_sampled.payload.visit）；本子类补齐 bump_visit/set_visit，save() 后
  archive.json 的 visit 与事件流一致（e2e 一致性断言的载体）。属同包内的
  运行时视图扩展，不改 archive.py 契约。

退出码（__main__ 契约）：0=正常（预算墙/优雅停止）；3=preverify-red 失败。

stdlib-only，Python >=3.9（禁 PEP604 X|Y）。
"""
import copy
import json
import os
import re
import shutil
import signal
import sys
import tempfile
import threading
import time
import uuid
from datetime import datetime, timezone, timedelta
from typing import Any, Callable, Dict, List, Optional

from superopt.agents import PiAgentClient
from superopt.archive import Archive
from superopt.budget import CostLedger
from superopt.config import DEFAULTS
from superopt.events import EventWriter
from superopt.feature_space import FeatureSpace
from superopt.gates import CascadeGate
from superopt.intervention import InterventionQueue
from superopt.lineage import project_state
from superopt.prompts import render, write_prompt
from superopt.sampler import Sampler

__all__ = ["Scheduler", "EPOCH_ITERATIONS", "ENV_FAIL_STREAK_LIMIT",
           "EXIT_OK", "EXIT_PREVERIFY_RED"]

# pending 重评估的 epoch 边界（设计树 §2.5：每 epoch 末重评估）
EPOCH_ITERATIONS = 10

# stage1 环境失败连续次数上限（超过即停止：环境坏了继续跑是浪费）
ENV_FAIL_STREAK_LIMIT = 3

# 退出码（与 __main__.py / Go 侧 PySupervisor 消费方对齐）
EXIT_OK = 0
EXIT_PREVERIFY_RED = 3

_TZ = timezone(timedelta(hours=8))

# adjust 软动作 action 字符串里的权重解析（如 "weight 0.5" / "0.5"）
_WEIGHT_RE = re.compile(r"(\d+(?:\.\d+)?)")


def _fmt_num(v: float) -> str:
    """性能数字的人读格式：整数不带尾零；浮点保留 1 位小数。"""
    if v == int(v):
        return "%d" % int(v)
    return "%.1f" % v


def _now() -> str:
    return datetime.now(_TZ).isoformat(timespec="seconds")


class _RuntimeArchive(Archive):
    """Archive 的运行时视图：cells 方法化 + visit 递增/恢复入口。

    - cells()：方法形式（Sampler 鸭子类型契约：callable(cells)）；
      基类是 property，本类覆写为同名方法返回同一快照
    - bump_visit：Sampler 的 on_visit 落点（visit/sample_count/last_sampled_at）
    - set_visit：_recover 从事件流投影恢复用（不走 bump 累加语义）
    visit 的唯一真相在 cell_sampled 事件；本视图保证 archive.json 侧一致。
    """

    def cells(self) -> Dict[str, Any]:
        # 基类 property 的值（深拷贝快照）——方法形式满足 Sampler 契约
        return Archive.cells.fget(self)  # type: ignore[attr-defined]

    def _cell(self, cell_id: str) -> Dict[str, Any]:
        cell = self._data["cells"].get(cell_id)
        if cell is None:
            cell = {"descriptor": {}, "elite": None, "history": [],
                    "weight": 1.0, "visit": 0, "sample_count": 0,
                    "last_sampled_at": None}
            self._data["cells"][cell_id] = cell
        return cell

    def bump_visit(self, cell_id: str) -> int:
        cell = self._cell(cell_id)
        cell["visit"] = int(cell.get("visit") or 0) + 1
        cell["sample_count"] = int(cell.get("sample_count") or 0) + 1
        cell["last_sampled_at"] = _now()
        return cell["visit"]

    def set_visit(self, cell_id: str, visit: int) -> None:
        cell = self._cell(cell_id)
        cell["visit"] = max(0, int(visit))

    def cell_entry(self, cell_id: str) -> Dict[str, Any]:
        return copy.deepcopy(self._data["cells"].get(cell_id) or {})


def _cells_snapshot(archive: Any) -> Dict[str, Any]:
    """兼容两种 cells 形态（property / 方法）的快照读取。"""
    c = getattr(archive, "cells", None)
    if callable(c):
        return c()
    return c or {}


class Scheduler:
    """四状态循环调度器（superopt 心脏）。

    构造（依赖注入——测试覆盖 agent_client/gate；生产全真件）::

        sched = Scheduler(ws, cfg)
        rc = sched.run()          # int 退出码

    cfg 结构（config.load + __main__ 装配）：
        {"preverify_red": bool, "env_path": Optional[str],
         "config": {adjust_interval, debug_max_attempts, soft_gate_k,
                    heartbeat_interval_s, max_wall_time_h, max_iterations,
                    pi_path, ...}}
    """

    def __init__(self, ws: str, cfg: Dict[str, Any],
                 agent_client: Optional[Any] = None,
                 gate: Optional[Any] = None,
                 event_writer: Optional[EventWriter] = None,
                 archive: Optional[Archive] = None,
                 feature_space: Optional[FeatureSpace] = None,
                 sampler: Optional[Sampler] = None,
                 ledger: Optional[CostLedger] = None,
                 queue: Optional[InterventionQueue] = None) -> None:
        self.ws = os.path.abspath(ws)
        self.cfg = cfg
        conf = dict(DEFAULTS)
        conf.update(cfg.get("config") or {})
        self.conf = conf

        self.events_path = os.path.join(self.ws, "events.jsonl")
        self.archive_path = os.path.join(self.ws, "archive.json")
        self.env_dir = cfg.get("env_path") or os.path.join(self.ws, "env")
        self.gates_dir = os.path.join(self.ws, "gates")
        self.codes_dir = os.path.join(self.ws, "codes")
        self.proposals_dir = os.path.join(self.ws, "proposals")
        self.heartbeat_path = os.path.join(self.ws, "heartbeat.json")
        os.makedirs(self.codes_dir, exist_ok=True)
        os.makedirs(self.proposals_dir, exist_ok=True)
        os.makedirs(self.gates_dir, exist_ok=True)

        # ---- env 装配（坏 env 不炸构造：run() 先检 env_error 给可诊断退出码） ----
        self.env_error = None  # type: Optional[str]
        self.task_name = ""
        self.perf_desc = "low"
        self.perf_type = "runtime_ms"
        try:
            env_meta = self._load_env_meta()
            self.task_name = env_meta.get("task", "")
            self.perf_desc = env_meta.get("perf_desc", "low")
            self.perf_type = env_meta.get("perf_type", "runtime_ms")

            # ---- 组装（可注入） ----
            self.events = event_writer or EventWriter(self.events_path)
            self.archive = archive or _RuntimeArchive(
                self.archive_path, perf_desc=self.perf_desc)
            self.archive.load()
            self.fs = feature_space or FeatureSpace(
                self._load_feature_axes())
            self.fs.validate_orthogonal()
            on_visit = getattr(self.archive, "bump_visit", None)
            if not callable(on_visit):
                # 注入的外部 Archive 无 bump_visit：内存补一层（visit 真相在事件）
                self._visit_shadow = {}  # type: Dict[str, int]
                on_visit = self._shadow_bump
            self.sampler = sampler or Sampler(
                self.archive, self.fs, on_visit=on_visit)
            self.client = agent_client or PiAgentClient(
                pi_path=str(conf.get("pi_path") or ""))
            self.gate = gate or CascadeGate(self.env_dir, self.gates_dir,
                                            k=int(conf.get("soft_gate_k", 3)))
            self.ledger = ledger or CostLedger(
                max_wall_time_h=float(conf.get("max_wall_time_h", 0) or 0),
                max_total_tokens=float(conf.get("max_total_tokens", 0) or 0))
            self.queue = queue or InterventionQueue(self.ws)
        except (OSError, ValueError) as exc:
            # env 缺失/损坏（如 feature_space.json 不存在）——不炸构造，
            # run() 对此返回可诊断退出码（与 __main__ 的 0/3/4 契约对齐）
            self.env_error = str(exc)

        # ---- 运行态 ----
        self.iteration_cursor = int(cfg.get("iteration_cursor") or 0)
        self.stop_requested = False
        self.env_fail_streak = 0
        self._iter_cost = {}  # type: Dict[str, float]  # 本迭代累计（iteration_ended 载荷）
        self._last_heartbeat_ts = 0.0
        self._prev_handler = None  # type: Optional[Any]
        # adjust 并行（优化空间⑤）：在飞线程 + 结果移交（主线程消费）
        self._adjust_thread = None  # type: Optional[threading.Thread]
        self._adjust_lock = threading.Lock()
        self._pending_adjust = None  # type: Optional[Dict[str, Any]]
        self._adjust_iteration = 0
        self._adjust_dir = ""  # type: str  # 本轮在飞 adjust 的快照目录（iter_N）
        # feature_space 热重载基线（task19：expand_cell 裁决应用/外部修改后
        # 迭代边界重建 self.fs 并重接 sampler）
        self._fs_mtime = None  # type: Optional[int]
        try:
            self._fs_mtime = os.stat(
                os.path.join(self.env_dir, "feature_space.json")).st_mtime_ns
        except OSError:
            self._fs_mtime = None
        # 预算耗尽标注（session_ended.budget_exhausted：task16 接线）
        self._budget_exhausted = False

    # ------------------------------------------------------------------ #
    # 主入口
    # ------------------------------------------------------------------ #

    def run(self) -> int:
        """主循环。返回退出码（0=正常；3=preverify-red 失败；4=env 不可用）。"""
        if self.env_error is not None:
            # env 配置缺失/损坏：不开跑、不写事件（__main__ 契约的可诊断退出）
            print("[superopt] FATAL: env 配置不可用（%s）: %s"
                  % (self.env_dir, self.env_error), file=sys.stderr, flush=True)
            return 4
        if self.cfg.get("preverify_red"):
            report = self._safe_preverify_red()
            if not report.get("ok"):
                # 会话前失败：不开跑、不写事件（Go 侧由退出码感知）
                print("[superopt] preverify-red FAILED: %s"
                      % json.dumps(report, ensure_ascii=False), flush=True)
                return EXIT_PREVERIFY_RED

        self.events.append("session_started",
                           {"env": self.env_dir, "pid": os.getpid()},
                           "scheduler")
        self._recover()
        self._install_signal_handlers()
        # 心跳线程：长 agent 派发（真 pi 单次可达十余分钟）会阻塞主循环，
        # 若心跳只在主循环 tick 写，Go PySupervisor 会误判心跳超时→kill→重启
        # →重启风暴（realpi 冒烟实测）。后台线程独立跳心，与主循环并发 append
        # 由 EventWriter 线程锁保护。
        self._hb_stop = threading.Event()
        self._hb_thread = threading.Thread(
            target=self._heartbeat_loop, name="superopt-heartbeat", daemon=True)
        self._hb_thread.start()
        try:
            self._write_heartbeat(force=True)
            self._loop()
        finally:
            self._hb_stop.set()
            self._hb_thread.join(timeout=5)
            self._join_adjust()   # 在飞 adjust 收尾（事件先于 session_ended）
            self._restore_signal_handlers()
        self.events.append("session_ended",
                           {"iteration_cursor": self.iteration_cursor,
                            "budget_exhausted": bool(self._budget_exhausted)},
                           "scheduler")
        self._set_super_state("stopped")
        self._write_heartbeat(force=True, state="stopped")
        return EXIT_OK

    def _loop(self) -> None:
        max_iter = int(self.conf.get("max_iterations") or 0)
        adjust_every = int(self.conf.get("adjust_interval") or 0)
        while not self.stop_requested:
            # ⓪ 收菜上一轮在飞的 adjust（优化空间⑤：adjust 与下一迭代并行；
            # 主线程在此消费结果——保证权重对本次采样生效）
            self._collect_adjust()
            if not self.ledger.within_budget():
                # 预算耗尽（墙钟/token 任一）——session_ended 标注可观测（task19）
                self._budget_exhausted = True
                break
            # ⓪′ feature_space 热重载（task19）：expand_cell 裁决应用或外部修改
            # （gen_env 升级/human 手改）后，在本迭代边界重建空间并重接 sampler——
            # 新格 visit=0 自然进入分殖优先，获得确定性首访
            self._maybe_reload_feature_space()
            if max_iter > 0 and self.iteration_cursor >= max_iter:
                break
            # ① 干预摄取（human 经队列；单写者纪律：代写入事件）
            self._drain_interventions()
            if self.stop_requested:
                break
            # ② 裁决决策消费（外置 adjudicator 的 decision 文件）
            self._drain_decisions()
            # ③ adjust（每 N 迭代；cursor 为已完成迭代数）——后台并行派发
            if adjust_every > 0 and self.iteration_cursor > 0 \
                    and self.iteration_cursor % adjust_every == 0:
                self._maybe_start_adjust()
            # ④ 单迭代
            self._step()
            # ⑤ epoch 边界：pending 重评估
            if (self.iteration_cursor > 0
                    and self.iteration_cursor % EPOCH_ITERATIONS == 0):
                self._reevaluate_pending()
            # ⑥ 心跳由后台线程独立跳（长 agent 派发不中断）

    def _heartbeat_loop(self) -> None:
        """后台心跳：周期性跳，与主循环并发。

        interval<=0（原「每迭代一跳」测试语义）时用 10s 地板：长 agent 派发
        （真 pi 单次可达十余分钟）主循环阻塞，若无线程跳心，Go PySupervisor
        会误判心跳超时→kill→重启风暴（realpi 冒烟实测）。地板值远低于任何
        心跳超时（生产默认 90s / 测试 30s）。
        """
        interval = float(self.conf.get("heartbeat_interval_s", 30) or 0)
        if interval <= 0:
            interval = 10.0
        while not self._hb_stop.wait(interval):
            try:
                self._write_heartbeat(force=True)
            except Exception as exc:  # noqa: BLE001
                # 心跳失败不杀主循环（如事件写器被外部关闭的窗口）；下一轮再试
                print("[superopt] heartbeat thread error: %s" % exc,
                      file=sys.stderr, flush=True)

    # ------------------------------------------------------------------ #
    # 恢复
    # ------------------------------------------------------------------ #

    def _recover(self) -> int:
        """事件流游标重放：恢复 cursor/visit/weight/pending（返回 last_seq）。

        空事件流 = 全新开始。恢复以事件为准（archive.json 可能滞后于崩溃点）：
        visit/weight 用投影覆盖内存档案；pending 重建；elite 保留磁盘档案
        （投影只含最小 elite 字段；罕见窗口的 invalidation-后-崩溃由
        pending 重评估自愈——已知限制，记录于此）。
        """
        events = self.events.read_all()
        if not events:
            return 0
        proj = project_state(events)
        self.iteration_cursor = int(proj.get("iteration_cursor") or 0)
        # 只对「确实发过 weight_changed 事件的格」应用投影权重：投影对
        # 无事件格缺省 1.0，会覆盖磁盘档案的真实 0.0（灭绝）——visit 同理
        # 但缺省 0 与磁盘一致，无需特判。
        weighted = set()
        for ev in events:
            if ev.get("type") == "weight_changed":
                cid = (ev.get("payload") or {}).get("cell_id")
                if isinstance(cid, str):
                    weighted.add(cid)
        for cid, st in (proj.get("cells") or {}).items():
            visit = int(st.get("visit") or 0)
            if hasattr(self.archive, "set_visit"):
                self.archive.set_visit(cid, visit)
            if cid in weighted and isinstance(st.get("weight"), (int, float)):
                self.archive.set_weight(cid, float(st["weight"]))
        # pending 重建（事件流是真相）
        if hasattr(self.archive, "drain_pending"):
            self.archive.drain_pending()
        for p in (proj.get("pending") or []):
            try:
                self.archive.add_pending(p.get("cell_id", ""),
                                         p.get("code_path", ""),
                                         p.get("reason", "tolerance_band"))
            except ValueError:
                continue  # 投影脏数据容忍
        self.archive.save()
        return int(proj.get("last_seq") or 0)

    # ------------------------------------------------------------------ #
    # 单迭代（四状态）
    # ------------------------------------------------------------------ #

    def _step(self) -> None:
        self.iteration_cursor += 1
        cursor = self.iteration_cursor
        iter_started = time.time()
        self._iter_cost = {"wall_ms": 0.0, "tool_calls": 0.0,
                           "tokens": 0.0, "cost_usd": 0.0}
        self.events.append("iteration_started",
                           {"iteration_id": cursor}, "scheduler")

        # ---- 状态 1：采样（纯加权随机，决策权在 adjust；visit 唯一递增点） ----
        cell_id = self.sampler.sample_cell()
        visit_before = int(self._cell_entry(cell_id).get("visit") or 0)
        cold_start = visit_before == 0
        self.sampler.increment_visit(cell_id)
        visit_now = int(self._cell_entry(cell_id).get("visit")
                        or visit_before + 1)
        snap = _cells_snapshot(self.archive)
        total_visit = sum(int((snap.get(c) or {}).get("visit") or 0)
                          for c in self.fs.all_cells())
        # weight_share：采样时刻的实际概率分布快照（task19：ucb_score 字段退役，
        # 与 task15 的 sampler.weight_share() 对齐）
        share = self.sampler.weight_share(cell_id)
        self.events.append("cell_sampled", {
            "iteration_id": cursor, "cell_id": cell_id, "visit": visit_now,
            "weight_share": round(share, 6), "cold_start": cold_start,
            "total_visit": total_visit,
        }, "scheduler")

        # ---- 迭代工作区（codes/{uuid}/ 隔离快照） ----
        iter_dir = os.path.join(
            self.codes_dir, "iter%03d-%s" % (cursor, uuid.uuid4().hex[:8]))
        os.makedirs(iter_dir, exist_ok=True)

        # ---- 状态 2：builder（选父格 → experiment_plan） ----
        plan = self._run_builder(cell_id, cursor, iter_dir)

        # ---- 状态 3：coding（自检仅参考）→ debug（≤M 次） ----
        candidate = self._run_coding(plan, cell_id, cursor, iter_dir)

        # ---- 状态 4：框架验证 + 入格 ----
        if candidate is not None:
            self._verify_and_admit(candidate, cell_id, cursor, plan)
        # else：coding/debug 全失败 → 已记失败事件，跳过终验（本迭代无产出）

        # ---- 收尾 ----
        wall_ms = int((time.time() - iter_started) * 1000)
        self.events.append("iteration_ended", {
            "iteration_id": cursor, "cell_id": cell_id,
            "wall_ms": wall_ms + int(self._iter_cost.get("wall_ms", 0.0)),
            "tool_calls": int(self._iter_cost.get("tool_calls", 0.0)),
            "tokens": int(self._iter_cost.get("tokens", 0.0)),
            "cost_usd": round(float(self._iter_cost.get("cost_usd", 0.0)), 6),
            "had_candidate": candidate is not None,
        }, "scheduler")
        # 每迭代落盘档案：bump_visit 只改内存，若本迭代无 admit/epoch 则到收尾都
        # 不 save——kill -9 后重启虽能从事件恢复，但「正常收尾的档案」会缺新采样
        # 格的 visit（job_47 e2e 实测投影1≠档案0）。每迭代一次 save，崩溃窗口
        # 从「上次 admit」缩到「本迭代」（事件流仍是唯一真相，恢复幂等）。
        self.archive.save()

    # ---- builder ----

    def _run_builder(self, cell_id: str, cursor: int, iter_dir: str) -> Dict[str, Any]:
        entry = self._cell_entry(cell_id)
        elite = entry.get("elite") or {}
        ctx = {
            "task": self.task_name, "perf_desc": self.perf_desc,
            "perf_type": self.perf_type,
            "feature_axes": self.fs.axes,
            "iteration_id": cursor,
            "cell_id": cell_id,
            "cell_descriptor": self.fs.describe_cell(cell_id) or {},
            "cell_elite": elite or None,
            "cell_visit": entry.get("visit", 0),
            "cell_weight": entry.get("weight", 1.0),
            "neighbor_elites": self._neighbor_elites(cell_id),
            "history_failures": self._recent_gate_failures(5),
            # 优化空间③（job_47 首跑实测）：同轴历史性能对照——避免重复已被
            # 轴级证据证伪的方向（如「row_major 劣势已在两分支独立复现」）
            "axis_perf": self._axis_perf_summary(),
        }
        prompt = render("builder", self.ws, ctx)
        prompt_path = write_prompt(iter_dir, "builder", prompt)
        result = self._dispatch_agent("builder", prompt_path, iter_dir, cursor)
        plan = result.data.get("experiment_plan")
        if not isinstance(plan, dict):
            plan = {"idea": result.summary or "", "target_cell": cell_id,
                    "parents": [], "plan_steps": [],
                    "expected_behavior": None}
        plan.setdefault("target_cell", cell_id)
        plan.setdefault("parents", [])
        # 迭代物证：plan 落盘（lineage 重建的旁证）
        with open(os.path.join(iter_dir, "experiment_plan.json"), "w",
                  encoding="utf-8") as f:
            json.dump(plan, f, ensure_ascii=False, indent=2)
        return plan

    def _axis_perf_summary(self) -> str:
        """同轴历史性能对照（优化空间③）：每轴一段「值 → 最优精英 perf」。

        汇总自 archive 有精英的格；无任何精英时返回占位（尚无精英数据）。
        低优 perf（越小越好）取 min，高优取 max——方向感知与 archive 一致。
        """
        snap = _cells_snapshot(self.archive)
        lines = []
        have_any = False
        low = (self.perf_desc != "high")  # 缺省/low 都是低优
        for axis in self.fs.axes:
            name = axis.get("name") or ""
            if not name:
                continue
            parts = []
            values = axis.get("values") or []
            if not values:
                # 非枚举轴（数值/布尔）：从描述子里拿实际值集
                seen = set()
                for cid in self.fs.all_cells():
                    v = (self.fs.describe_cell(cid) or {}).get(name)
                    if v is not None and v not in seen:
                        seen.add(v)
                values = sorted(seen, key=str)
            for v in values:
                best = None
                count = 0
                for cid in self.fs.all_cells():
                    desc = self.fs.describe_cell(cid) or {}
                    if desc.get(name) != v:
                        continue
                    e = (snap.get(cid) or {}).get("elite") or {}
                    pv = e.get("perf_value")
                    if not isinstance(pv, (int, float)):
                        continue
                    count += 1
                    if best is None:
                        best = float(pv)
                    else:
                        best = min(best, float(pv)) if low else max(best, float(pv))
                if count > 0:
                    have_any = True
                    parts.append("%s→最优 %s（%d 格）"
                                 % (v, _fmt_num(best), count))
                else:
                    parts.append("%s→无数据" % (v,))
            if parts:
                lines.append("%s轴：%s" % (name, "；".join(parts)))
        if not lines or not have_any:
            return "（尚无精英数据）"
        return "\n".join(lines)

    def _neighbor_elites(self, cell_id: str, limit: int = 3) -> List[Dict[str, Any]]:
        """邻格精英（跨格重组灵感）：同轴单值不同的格，取 visit 最高的几个。"""
        desc = self.fs.describe_cell(cell_id) or {}
        if not desc:
            return []
        neighbors = []
        for cid in self.fs.all_cells():
            if cid == cell_id:
                continue
            nd = self.fs.describe_cell(cid) or {}
            diff = sum(1 for k in desc if nd.get(k) != desc.get(k))
            if diff == 1:
                e = self._cell_entry(cid)
                if e.get("elite"):
                    neighbors.append({
                        "cell_id": cid, "descriptor": nd,
                        "elite": {"code_path": e["elite"].get("code_path"),
                                  "perf_value": e["elite"].get("perf_value")},
                        "visit": e.get("visit", 0),
                    })
        neighbors.sort(key=lambda n: -int(n.get("visit") or 0))
        return neighbors[:limit]

    def _recent_gate_failures(self, n: int) -> List[Dict[str, Any]]:
        out = []
        for ev in self.events.read_all()[-80:]:
            if ev.get("type") == "gate_result" and not ev["payload"].get("passed"):
                out.append({"iteration_id": ev["payload"].get("iteration_id"),
                            "failure_stage": ev["payload"].get("failure_stage")})
        return out[-n:]

    # ---- coding / debug ----

    def _run_coding(self, plan: Dict[str, Any], cell_id: str, cursor: int,
                    iter_dir: str) -> Optional[str]:
        """实现 + 自检 + debug 循环。返回候选绝对路径（全失败 → None）。"""
        self._seed_parent_code(plan, cell_id, iter_dir)
        self_check_cmd = self._self_check_cmd()
        ctx = {
            "task": self.task_name, "perf_desc": self.perf_desc,
            "perf_type": self.perf_type,
            "iteration_id": cursor,
            "experiment_plan": plan,
            "workdir": iter_dir,
            "env_dir": self.env_dir,
            "self_check_cmd": self_check_cmd,
        }
        prompt = render("coding", self.ws, ctx)
        prompt_path = write_prompt(iter_dir, "coding", prompt)
        result = self._dispatch_agent("coding", prompt_path, iter_dir, cursor)

        candidate = self._candidate_path(result, iter_dir)
        self_check = self._self_check_of(result)
        self.events.append("candidate_produced", {
            "iteration_id": cursor, "cell_id": cell_id,
            "role": "coding",
            "code_path": candidate or "",
            "self_check": self_check,
            "diff_path": result.data.get("diff_path"),
        }, "agent:coding")

        if self_check.get("passed"):
            return candidate

        # ---- debug 循环（≤M 次；每次一条 fix_report 载体事件） ----
        m = int(self.conf.get("debug_max_attempts") or 0)
        last_err = json.dumps(self_check, ensure_ascii=False)
        for attempt in range(1, m + 1):
            debug_result = self._run_debug(plan, cell_id, cursor, iter_dir,
                                           attempt, m, self_check, last_err)
            dpath = self._candidate_path(debug_result, iter_dir)
            dcheck = self._final_check_of(debug_result)
            self.events.append("candidate_produced", {
                "iteration_id": cursor, "cell_id": cell_id,
                "role": "debug", "attempt": attempt,
                "code_path": dpath or "",
                "fix_report": debug_result.data.get("fix_report")
                or {"attempts": attempt, "root_cause": None, "changes": []},
                "self_check": dcheck,
            }, "agent:debug")
            if dcheck.get("passed"):
                return dpath or candidate
            last_err = json.dumps(dcheck, ensure_ascii=False)
        # M 次仍未修复 → 记失败（iteration_ended.had_candidate=False 由调用方收尾）
        return None

    def _run_debug(self, plan: Dict[str, Any], cell_id: str, cursor: int,
                   iter_dir: str, attempt: int, max_attempts: int,
                   self_check: Dict[str, Any], last_err: str) -> Any:
        ctx = {
            "task": self.task_name, "perf_desc": self.perf_desc,
            "perf_type": self.perf_type,
            "iteration_id": cursor,
            "experiment_plan": plan,
            "workdir": iter_dir,
            "env_dir": self.env_dir,
            "self_check_cmd": self._self_check_cmd(),
            "attempt": attempt, "max_attempts": max_attempts,
            "self_check_result": self_check,
            "failure_stage": self_check.get("stage"),
            "stderr_tail": last_err[-400:],
        }
        prompt = render("debug", self.ws, ctx)
        prompt_path = write_prompt(iter_dir, "debug", prompt)
        return self._dispatch_agent("debug", prompt_path, iter_dir, cursor)

    def _seed_parent_code(self, plan: Dict[str, Any], cell_id: str,
                          iter_dir: str) -> None:
        """把父本代码拷进迭代工作区（coding 提示词的「父本已就位」契约）。"""
        parents = [p for p in (plan.get("parents") or []) if isinstance(p, str)]
        if not parents:
            elite = self._cell_entry(cell_id).get("elite") or {}
            cp = elite.get("code_path")
            if isinstance(cp, str) and cp:
                parents = [cp]
        copied = False
        for p in parents:
            if os.path.isfile(p):
                shutil.copy2(p, os.path.join(iter_dir, os.path.basename(p)))
                copied = True
        if not copied:
            # 无父本：拷 env 参考实现起步（进化从 naive/参考起步，L3-Q6）
            env_code = os.path.join(self.env_dir, "code")
            if os.path.isdir(env_code):
                for name in sorted(os.listdir(env_code)):
                    src = os.path.join(env_code, name)
                    if os.path.isfile(src):
                        shutil.copy2(src, os.path.join(iter_dir, name))

    def _self_check_cmd(self) -> str:
        py = "python3"
        venv = os.path.join(self.env_dir, ".venv", "bin", "python")
        if os.path.exists(venv):
            py = venv
        sc = os.path.join(self.env_dir, "skills", "self_check.py")
        return "%s %s --candidate ." % (py, sc)

    @staticmethod
    def _candidate_path(result: Any, iter_dir: str) -> Optional[str]:
        diff = result.data.get("diff_path")
        if isinstance(diff, str) and diff.strip():
            p = diff if os.path.isabs(diff) else os.path.join(iter_dir, diff)
            if os.path.exists(p):
                return p
            return p  # agent 声称的路径（门禁 stage1 会验存在性）
        # 缺省：找工作区里最像候选的 py
        for name in ("solution.py", "candidate.py"):
            p = os.path.join(iter_dir, name)
            if os.path.exists(p):
                return p
        return None

    @staticmethod
    def _self_check_of(result: Any) -> Dict[str, Any]:
        sc = result.data.get("self_check")
        if isinstance(sc, dict) and "passed" in sc:
            return sc
        # coding 契约：valid=True 即完成实现与自检（自检结果如实填 self_check）
        return {"passed": bool(result.valid), "stage": "self_check",
                "metrics": {}}

    @staticmethod
    def _final_check_of(result: Any) -> Dict[str, Any]:
        fc = result.data.get("final_self_check")
        if isinstance(fc, dict) and "passed" in fc:
            return fc
        sc = result.data.get("self_check")
        if isinstance(sc, dict) and "passed" in sc:
            return sc
        return {"passed": False, "stage": "self_check", "metrics": {}}

    # ---- 验证 + 入格 ----

    def _verify_and_admit(self, candidate: str, cell_id: str, cursor: int,
                          plan: Dict[str, Any]) -> None:
        entry = self._cell_entry(cell_id)
        elite = entry.get("elite")
        history = [h.get("perf_value") for h in (entry.get("history") or [])
                   if h.get("perf_value") is not None]
        gr = self.gate.verify(candidate,
                              elite_perf=(elite or {}).get("perf_value"),
                              history=history)
        self.events.append("gate_result", {
            "iteration_id": cursor, "cell_id": cell_id,
            "stage": gr.failure_stage or "passed",
            "failure_stage": gr.failure_stage,
            "passed": bool(gr.passed),
            "metrics": gr.metrics,
        }, "gate")

        if gr.passed:
            self._admit(candidate, cell_id, cursor, plan, gr, had_elite=bool(elite))
            self.env_fail_streak = 0
            return

        stage = gr.failure_stage or ""
        if stage.startswith("stage1"):
            # 环境失败结论化：诊断包 + 连续计数（超限停止）
            self.env_fail_streak += 1
            self.events.append("env_failure_concluded", {
                "iteration_id": cursor, "cell_id": cell_id,
                "returncode": None,
                "stderr_tail": (gr.details or "")[-400:],
                "cause": "stage1_env_unhealthy",
                "streak": self.env_fail_streak,
            }, "scheduler")
            if self.env_fail_streak >= ENV_FAIL_STREAK_LIMIT:
                self.stop_requested = True
        else:
            # 正常实验失败（正确性/性能不达标）：记失败继续下迭代
            self.env_fail_streak = 0

    def _admit(self, candidate: str, cell_id: str, cursor: int,
               plan: Dict[str, Any], gr: Any, had_elite: bool) -> None:
        perf = (gr.metrics or {}).get("perf") or {}
        perf_value = perf.get("value")
        if perf_value is None:
            perf_value = gr.metrics.get("correctness", {}).get("max_error")
        parents = [p for p in (plan.get("parents") or [])
                   if isinstance(p, str)]
        parent_elite = self._cell_entry(cell_id).get("elite") or {}
        if not parents:
            pp = parent_elite.get("code_path")
            if isinstance(pp, str) and pp:
                parents = [pp]
        gen = int(parent_elite.get("gen") or 0) + 1
        entry = {
            "descriptor": self.fs.describe_cell(cell_id) or {},
            "elite": {
                "code_path": candidate,
                "perf_value": perf_value,
                "perf_type": self.perf_type,
                "perf_desc": self.perf_desc,
                "score": perf.get("score"),
                "lineage": parents,
                "confidence": 0.5 if gr.tolerance_band else 0.8,
                "admitted_at": _now(),
                "gen": gen,
            },
        }
        admitted = self.archive.admit(cell_id, entry)
        if not admitted:
            return  # 更差解：同格替换条件不满足（事件流无 elite_*——诚实记录）

        # ---- 延迟入库判定（L2-Q6：三者任一） ----
        _w = self._cell_entry(cell_id).get("weight")
        weight = float(_w) if isinstance(_w, (int, float)) else 1.0
        if gr.tolerance_band:
            reason = "tolerance_band"
        elif not had_elite:
            reason = "new_cell"
        elif weight < 1.0:
            reason = "low_weight_src"
        else:
            reason = None
        if reason:
            self.archive.add_pending(cell_id, candidate, reason)

        self.events.append(
            "elite_replaced" if had_elite else "elite_admitted",
            {
                "iteration_id": cursor, "cell_id": cell_id,
                "code_path": candidate,
                "perf_value": perf_value,
                "score": perf.get("score"),
                "lineage": parents, "gen": gen,
                "confidence": entry["elite"]["confidence"],
                "pending": bool(reason),
                # reason：人读；pending_reason：lineage.project_state 的
                # 投影重建契约字段（缺省 fallback 是 tolerance_band——必须显式）
                "reason": reason, "pending_reason": reason,
            }, "scheduler")
        self.archive.save()

    # ------------------------------------------------------------------ #
    # adjust（软动作直改 / 硬动作提案）
    # ------------------------------------------------------------------ #

    def _maybe_start_adjust(self) -> None:
        """边界触发：后台派发 adjust agent（优化空间⑤：与下一迭代并行）。

        行为约束：同一时刻至多 1 个在飞；在飞未收时到边界不重触发；
        ctx 组装与 agent_started 事件在主线程（快且共享状态安全），
        线程只跑 client.run（独立子进程，无共享可变状态）。
        """
        if self.stop_requested:
            return
        with self._adjust_lock:
            if self._adjust_thread is not None:
                return  # 在飞未收：不重触发
        ctx = {
            "task": self.task_name, "perf_desc": self.perf_desc,
            "perf_type": self.perf_type,
            "feature_axes": self.fs.axes,
            "iteration_id": self.iteration_cursor,
            "cells_status": self._cells_status(),
            "recent_events": self._recent_events_summary(12),
        }
        prompt = render("adjust", self.ws, ctx)
        # 决策快照按迭代归档（task19：codes/adjust/iter_{N}/，不覆盖写——
        # 轨迹可回溯是自改进的地基；agent 工作目录与快照同目录）
        iteration_id = self.iteration_cursor
        adjust_dir = os.path.join(self.codes_dir, "adjust",
                                  "iter_%d" % iteration_id)
        os.makedirs(adjust_dir, exist_ok=True)
        prompt_path = write_prompt(adjust_dir, "adjust", prompt)
        # agent_started 在主线程（事件序确定性：started 先于下一迭代的任何事件）
        self.events.append("agent_started", {
            "role": "adjust", "iteration_id": iteration_id,
            "workdir": adjust_dir,
        }, "scheduler")
        model = (self.conf.get("agent_models") or {}).get("adjust")
        timeout_s = int(self.conf.get("agent_timeout_s", 1800))

        def _worker():
            try:
                result = self.client.run("adjust", prompt_path, adjust_dir,
                                         timeout_s, model=model)
            except Exception as exc:  # noqa: BLE001 线程绝不向上抛
                from superopt.agents import AgentResult as _AR
                result = _AR(role="adjust", valid=False,
                             summary="agent error: %s" % exc)
            with self._adjust_lock:
                self._pending_adjust = result

        with self._adjust_lock:
            self._adjust_iteration = iteration_id
            self._adjust_dir = adjust_dir
            self._adjust_thread = threading.Thread(
                target=_worker, name="superopt-adjust", daemon=True)
            self._adjust_thread.start()

    def _collect_adjust(self) -> None:
        """主线程消费在飞 adjust 的结果（agent_ended + 记账 + 提案应用）。

        无在飞/未完成 → 空操作；消费后清线程引用（下一边界可重触发）。
        """
        with self._adjust_lock:
            thread = self._adjust_thread
            pending = self._pending_adjust
        if thread is None:
            return
        if pending is None:
            if thread.is_alive():
                return  # 还在飞：下一边界再收
            # 线程结束但无结果（异常路径未写 pending——防御）
            with self._adjust_lock:
                self._adjust_thread = None
            return
        thread.join(timeout=5)
        with self._adjust_lock:
            self._adjust_thread = None
            self._pending_adjust = None
            iteration_id = self._adjust_iteration
            adjust_dir = self._adjust_dir or os.path.join(
                self.codes_dir, "adjust", "iter_%d" % iteration_id)
        # ---- 以下与 _dispatch_agent 的收尾同构（主线程执行） ----
        self.ledger.record_agent("adjust", int(pending.wall_ms),
                                 len(pending.tool_calls or []),
                                 dict(pending.usage or {}))
        self._iter_cost["wall_ms"] = self._iter_cost.get("wall_ms", 0.0) \
            + float(pending.wall_ms or 0)
        self._iter_cost["tool_calls"] = self._iter_cost.get("tool_calls", 0.0) \
            + float(len(pending.tool_calls or []))
        _u = pending.usage or {}
        self._iter_cost["tokens"] = self._iter_cost.get("tokens", 0.0) \
            + float(_u.get("total_tokens", 0) or 0)
        self._iter_cost["cost_usd"] = self._iter_cost.get("cost_usd", 0.0) \
            + float(_u.get("cost_usd", 0) or 0)
        payload = pending.to_payload()
        payload["iteration_id"] = iteration_id
        # prompt 物证：本轮快照目录（iter_N）下的 prompt_adjust.md
        payload["prompt_path"] = os.path.join(adjust_dir, "prompt_adjust.md")
        self.events.append("agent_ended", payload, "scheduler")
        # 决策快照按迭代归档（task19）：agent_adjust.json 落 iter_N 目录
        # （不覆盖写——历史轨迹完整保留，自改进分析的物证）
        try:
            with open(os.path.join(adjust_dir, "agent_adjust.json"), "w",
                      encoding="utf-8") as f:
                json.dump(payload, f, ensure_ascii=False, indent=2)
        except OSError:
            pass
        proposals = pending.data.get("proposals")
        if not isinstance(proposals, list):
            proposals = []
        # adjust_decision 事件（task19）：结构化决策日志——输入（cells_status
        # 摘要）+ 输出（proposals 摘要）同帧落事件流；自改进分析（adjust_trace）
        # 与崩溃恢复均可消费
        self.events.append("adjust_decision", {
            "iteration": iteration_id,
            "agent_valid": bool(pending.valid),
            "proposals": self._proposal_digest(proposals),
            "cells_status_summary": self._cells_status_digest(),
        }, "scheduler")
        for i, p in enumerate(proposals):
            if not isinstance(p, dict):
                continue
            self._apply_proposal(p, i)

    def _join_adjust(self) -> None:
        """run() 收尾：等在飞 adjust 线程并消费（事件序：adjust 归档先于
        session_ended）。超时 5s 未收 → 放弃（daemon 线程不阻退出）。"""
        with self._adjust_lock:
            thread = self._adjust_thread
        if thread is not None:
            thread.join(timeout=5)
        self._collect_adjust()

    def _apply_proposal(self, p: Dict[str, Any], index: int) -> None:
        ptype = p.get("type")
        target = p.get("target_cell") or p.get("cell_id") or ""
        reason = str(p.get("evidence") or p.get("reason") or "")[:200]
        if ptype == "soft_weight":
            # 软动作：直接应用（weight_changed 事件 + archive）
            m = _WEIGHT_RE.search(str(p.get("action") or ""))
            if not m:
                return
            new_w = float(m.group(1))
            old = self._cell_entry(target).get("weight")
            old_w = float(old) if isinstance(old, (int, float)) else 1.0
            self.archive.set_weight(target, new_w)
            self.events.append("weight_changed", {
                "cell_id": target, "old_weight": old_w,
                "new_weight": new_w, "reason": reason,
                "rule_id": p.get("rule_id"),
            }, "agent:adjust")
            self.archive.save()
            return
        # 硬动作（及未知类型——保守走提案通道）：proposal 事件 + 投递裁决队列
        pid = str(p.get("proposal_id") or
                  "adj-%d-%d" % (self.iteration_cursor, index))
        record = {"proposal_id": pid, "type": str(ptype or "unknown"),
                  "target_cell": target, "action": p.get("action"),
                  "evidence": reason, "rule_id": p.get("rule_id"),
                  "proposed_at": _now(),
                  "iteration": self.iteration_cursor}
        # expand_cell 提案（task19）：携带结构化扩张参数——裁决与应用侧
        # （adjudicator 校验 axis 存在/值不重复；approve 后 loop 写回
        # feature_space.json）依赖这些字段，缺字段裁决按保守拒绝
        if ptype == "expand_cell":
            record["axis"] = p.get("axis")
            record["value"] = p.get("value")
            record["descriptor"] = p.get("descriptor")
            record["rationale"] = str(p.get("rationale") or "")[:400]
        os.makedirs(self.proposals_dir, exist_ok=True)
        with open(os.path.join(self.proposals_dir, pid + ".json"), "w",
                  encoding="utf-8") as f:
            json.dump(record, f, ensure_ascii=False, indent=2)
        prop_payload = {
            "proposal_id": pid, "type": record["type"],
            "target_cell": target, "action": p.get("action"),
            "evidence": reason, "rule_id": p.get("rule_id"),
        }
        if ptype == "expand_cell":
            prop_payload["axis"] = p.get("axis")
            prop_payload["value"] = p.get("value")
            prop_payload["descriptor"] = p.get("descriptor")
        self.events.append("proposal", prop_payload, "agent:adjust")

    def _cells_status(self) -> List[Dict[str, Any]]:
        out = []
        for cid in self.fs.all_cells():
            e = self._cell_entry(cid)
            out.append({"cell_id": cid,
                        "descriptor": self.fs.describe_cell(cid) or {},
                        "visit": e.get("visit", 0),
                        "weight": e.get("weight", 1.0),
                        "elite": ({"code_path": e["elite"].get("code_path"),
                                   "perf_value": e["elite"].get("perf_value")}
                                  if e.get("elite") else None)})
        return out

    def _recent_events_summary(self, n: int) -> List[Dict[str, Any]]:
        return [{"type": e.get("type"),
                 "seq": e.get("seq"),
                 "payload_keys": sorted((e.get("payload") or {}).keys())[:6]}
                for e in self.events.read_all()[-n:]]

    def _proposal_digest(self, proposals: List[Any]) -> List[Dict[str, Any]]:
        """proposals 的紧凑摘要（adjust_decision 事件载荷）。

        只保留归因分析需要的字段：type / target_cell / action / rule_id /
        evidence（截断）/ expand_cell 的 axis/value/descriptor。
        """
        out = []
        for p in proposals:
            if not isinstance(p, dict):
                continue
            item = {"type": p.get("type"),
                    "target_cell": p.get("target_cell") or p.get("cell_id"),
                    "action": p.get("action"),
                    "rule_id": p.get("rule_id"),
                    "evidence": str(p.get("evidence") or p.get("reason")
                                    or "")[:200]}
            if p.get("type") == "expand_cell":
                item["axis"] = p.get("axis")
                item["value"] = p.get("value")
                item["descriptor"] = p.get("descriptor")
            out.append(item)
        return out

    def _cells_status_digest(self) -> List[Dict[str, Any]]:
        """cells_status 的紧凑摘要（adjust_decision 事件载荷：决策输入快照）。

        cell_id / visit / weight / 精英 perf——自改进分析（决策→效果对照）
        需要的最小字段，不含 descriptor 全文与代码路径。
        """
        out = []
        for cid in self.fs.all_cells():
            e = self._cell_entry(cid)
            elite = e.get("elite") or {}
            out.append({"cell_id": cid,
                        "visit": e.get("visit", 0),
                        "weight": e.get("weight", 1.0),
                        "elite_perf": elite.get("perf_value")})
        return out

    # ---- expand_cell 应用 + feature_space 热重载（task19） ----

    def _apply_expand_cell(self, proposal: Dict[str, Any]) -> Optional[Dict[str, Any]]:
        """approve 的 expand_cell → 写回 ws/env/feature_space.json。

        - add_axis_value：向既有 axes[].values 追加值（去重；轴缺失/值已在
          → 不改文件，返回 None——裁决侧本已校验，这里是应用侧防御）
        - add_cell：descriptor 每个值补进对应轴（新值追加，已存在的跳过）
        原子写（tmp+rename）；返回 applied 摘要（{action, axis, value} 或
        {action, added_values}），失败/无变化返回 None。
        """
        path = os.path.join(self.env_dir, "feature_space.json")
        try:
            with open(path, "r", encoding="utf-8") as f:
                data = json.load(f)
        except (OSError, ValueError):
            return None
        axes = data.get("axes") if isinstance(data, dict) else None
        if not isinstance(axes, list):
            return None
        action = str(proposal.get("action") or "")
        changed = False
        applied = None  # type: Optional[Dict[str, Any]]

        def _axis_by_name(name):
            for ax in axes:
                if isinstance(ax, dict) and ax.get("name") == name:
                    return ax
            return None

        if action == "add_axis_value":
            axis_name = proposal.get("axis")
            value = proposal.get("value")
            if not isinstance(axis_name, str) or not axis_name:
                return None
            if value is None or (isinstance(value, str) and not value.strip()):
                return None
            ax = _axis_by_name(axis_name)
            if ax is None:
                return None
            values = ax.get("values")
            if not isinstance(values, list):
                return None
            norm = str(value)
            if norm in [str(v) for v in values]:
                return None  # 已存在：无变化
            values.append(value)
            changed = True
            applied = {"action": "add_axis_value", "axis": axis_name,
                       "value": norm}
        elif action == "add_cell":
            descriptor = proposal.get("descriptor")
            if not isinstance(descriptor, dict) or not descriptor:
                return None
            added = {}  # type: Dict[str, str]
            for axis_name, value in descriptor.items():
                if not isinstance(axis_name, str) or value is None:
                    return None
                ax = _axis_by_name(axis_name)
                if ax is None:
                    return None  # 未知轴：整体不改（裁决侧已校验）
                values = ax.get("values")
                if not isinstance(values, list):
                    return None
                norm = str(value)
                if norm not in [str(v) for v in values]:
                    values.append(value)
                    added[axis_name] = norm
                    changed = True
            if not changed:
                return None
            applied = {"action": "add_cell", "added_values": added}
        else:
            return None

        if changed:
            try:
                self._atomic_json(path, data)
            except OSError:
                return None
        return applied

    def _maybe_reload_feature_space(self) -> None:
        """迭代边界热重载：feature_space.json mtime 变化 → 重建 fs + 重接 sampler。

        变更来源：expand_cell 裁决应用（本调度器写回）或外部修改（gen_env
        升级 / human 手改）。重建失败（半写/坏 JSON/正交性破坏）→ 保留旧空间
        打 stderr，下一边界重试（文件是真相，加载防御性跳过不炸主循环）。
        """
        path = os.path.join(self.env_dir, "feature_space.json")
        try:
            mtime = os.stat(path).st_mtime_ns
        except OSError:
            return
        if self._fs_mtime is None or mtime == self._fs_mtime:
            if self._fs_mtime is None:
                self._fs_mtime = mtime
            return
        try:
            with open(path, "r", encoding="utf-8") as f:
                data = json.load(f)
            axes = data.get("axes") if isinstance(data, dict) else None
            if not isinstance(axes, list):
                raise ValueError("feature_space.json 缺 axes")
            new_fs = FeatureSpace(axes)
            new_fs.validate_orthogonal()
        except (OSError, ValueError) as exc:
            print("[superopt] feature_space 热重载失败（保留旧空间，下边界重试）: %s"
                  % exc, file=sys.stderr, flush=True)
            return
        self.fs = new_fs
        self._fs_mtime = mtime
        # 重接 sampler 的空间引用（set_feature_space 仅换引用，采样语义不变）
        self.sampler.set_feature_space(new_fs)
        # 新格进分殖优先（visit=0）——无需显式处理：sampler 的分殖规则天然覆盖

    # ------------------------------------------------------------------ #
    # 决策消费 / 干预摄取 / pending 重评估
    # ------------------------------------------------------------------ #

    def _drain_decisions(self) -> None:
        """消费 <ws>/proposals/*.decision.json → adjudicated 事件 + 应用。"""
        try:
            names = sorted(os.listdir(self.proposals_dir))
        except FileNotFoundError:
            return
        for name in names:
            if not name.endswith(".decision.json"):
                continue
            path = os.path.join(self.proposals_dir, name)
            try:
                with open(path, "r", encoding="utf-8") as f:
                    rec = json.load(f)
            except (OSError, ValueError):
                continue  # 半写 decision：留待下次（adjudicator 原子写，罕见）
            pid = str(rec.get("proposal_id") or name[:-len(".decision.json")])
            decision = str(rec.get("decision") or "reject")
            self.events.append("adjudicated", {
                "proposal_id": pid, "decision": decision,
                "reason": rec.get("reason"),
                "conditions": rec.get("conditions"),
            }, "adjudicator")
            # 应用：approve 的 hard_delete_cell = 功能性灭绝（不删格）；
            #         approve 的 expand_cell = 写回 feature_space.json（task19：
            #         热重载在下一迭代边界生效——新格进分殖优先）
            if decision == "approve":
                proposal = self._load_proposal(pid)
                if proposal and proposal.get("type") == "hard_delete_cell":
                    target = proposal.get("target_cell") or ""
                    old = self._cell_entry(target).get("weight")
                    old_w = float(old) if isinstance(old, (int, float)) else 1.0
                    self.archive.set_weight(target, 0.0)
                    self.events.append("weight_changed", {
                        "cell_id": target, "old_weight": old_w,
                        "new_weight": 0.0,
                        "reason": "adjudicated hard_delete_cell → 功能性灭绝",
                        "rule_id": pid,
                    }, "scheduler")
                    self.archive.save()
                elif proposal and proposal.get("type") == "expand_cell":
                    # 应用 = 写回 feature_space.json（append-only 事件流纪律：
                    # 文件是空间真相，热重载在下一迭代边界生效——新格 visit=0
                    # 自然进分殖优先）；应用结果以追加事件留痕（不改写历史）
                    applied = self._apply_expand_cell(proposal)
                    if applied:
                        self.events.append("adjudicated", {
                            "proposal_id": pid, "decision": "applied",
                            "reason": "expand_cell 应用完成（feature_space "
                                      "已更新，下一迭代边界热重载生效）",
                            "conditions": [],
                            "applied": applied,
                        }, "scheduler")
            try:
                os.unlink(path)
            except OSError:
                pass

    def _load_proposal(self, pid: str) -> Optional[Dict[str, Any]]:
        path = os.path.join(self.proposals_dir, pid + ".json")
        try:
            with open(path, "r", encoding="utf-8") as f:
                data = json.load(f)
            return data if isinstance(data, dict) else None
        except (OSError, ValueError):
            return None

    def _drain_interventions(self) -> None:
        """干预队列 → intervention 事件（stop 类 → 优雅停止）。"""
        for item in self.queue.drain():
            if item.get("type") == "bad_intervention":
                continue  # 坏文件已在队列层留痕（文件已删），不产生事件
            payload = item.get("payload") or {}
            text = payload.get("text") if isinstance(payload, dict) else None
            if not isinstance(text, str):
                text = json.dumps(payload, ensure_ascii=False)
            category = str(item.get("type") or "note")
            self.events.append("intervention", {
                "text": text, "category": category,
                "created_at": item.get("created_at"),
            }, "human")
            if category == "stop":
                self.stop_requested = True

    def _reevaluate_pending(self) -> None:
        """epoch 末 pending 重评估：M 次复测 → 保留/复活/出档。"""
        pending = self.archive.drain_pending()
        if not pending:
            return
        m = max(1, int(self.conf.get("debug_max_attempts") or 1))
        for item in pending:
            cid = item.get("cell_id") or ""
            code_path = item.get("code_path") or ""
            if not cid or not code_path or not os.path.exists(code_path):
                continue
            passes = 0
            for _ in range(m):
                gr = self.gate.verify(code_path)
                if gr.passed:
                    passes += 1
            majority = passes * 2 > m
            entry = self._cell_entry(cid)
            if majority:
                _w = entry.get("weight")
                weight = float(_w) if isinstance(_w, (int, float)) else 1.0
                if weight <= 0.0:
                    # 复活：灭绝来源格的解复测通过 → 恢复采样权
                    self.archive.set_weight(cid, 1.0)
                    self.events.append("resurrection", {
                        "cell_id": cid, "code_path": code_path,
                        "reason": "pending 复测多数通过（%d/%d）→ 恢复采样权"
                                  % (passes, m),
                        "restored_weight": 1.0,
                    }, "scheduler")
                # 保留：pending 结清（drain 已清），无事件
            else:
                # 出档：elite 清空（格保留）
                if hasattr(self.archive, "_data"):
                    cell = self.archive._data["cells"].get(cid)
                    if cell is not None:
                        cell["elite"] = None
                self.events.append("invalidation", {
                    "cell_id": cid, "code_path": code_path,
                    "reason": "pending 复测多数失败（%d/%d）→ 出档"
                              % (passes, m),
                }, "scheduler")
        self.archive.save()

    # ------------------------------------------------------------------ #
    # agent 派发（事件包夹 + 记账）
    # ------------------------------------------------------------------ #

    def _dispatch_agent(self, role: str, prompt_path: str, workdir: str,
                        iteration_id: int) -> Any:
        self.events.append("agent_started", {
            "role": role, "iteration_id": iteration_id,
            "workdir": workdir,
        }, "scheduler")
        timeout_s = int(self.conf.get("agent_timeout_s", 1800))
        # per-role 模型档位（优化空间①：builder 等角色可降档省 token）
        model = (self.conf.get("agent_models") or {}).get(role)
        try:
            result = self.client.run(role, prompt_path, workdir, timeout_s,
                                     model=model)
        except Exception as exc:  # agent 超时/崩溃 → 记失败继续（spec）
            result = PiAgentClient.__new__(PiAgentClient)  # 不用——见下
            from superopt.agents import AgentResult as _AR
            result = _AR(role=role, valid=False, summary="agent error: %s" % exc)
        # 记账 + 事件
        self.ledger.record_agent(role, int(result.wall_ms),
                                 len(result.tool_calls or []),
                                 dict(result.usage or {}))
        self._iter_cost["wall_ms"] = self._iter_cost.get("wall_ms", 0.0) \
            + float(result.wall_ms or 0)
        self._iter_cost["tool_calls"] = self._iter_cost.get("tool_calls", 0.0) \
            + float(len(result.tool_calls or []))
        u = result.usage or {}
        self._iter_cost["tokens"] = self._iter_cost.get("tokens", 0.0) \
            + float(u.get("total_tokens", 0) or 0)
        self._iter_cost["cost_usd"] = self._iter_cost.get("cost_usd", 0.0) \
            + float(u.get("cost_usd", 0) or 0)
        payload = result.to_payload()
        payload["iteration_id"] = iteration_id
        payload["prompt_path"] = prompt_path
        # 事件表（§2.2）：agent_started/ended 产生者 = scheduler（调度器的观察）
        self.events.append("agent_ended", payload, "scheduler")
        # 迭代物证：agent final 落盘
        try:
            with open(os.path.join(workdir, "agent_%s.json" % role), "w",
                      encoding="utf-8") as f:
                json.dump(payload, f, ensure_ascii=False, indent=2)
        except OSError:
            pass
        return result

    # ------------------------------------------------------------------ #
    # 心跳 / super.json / 信号
    # ------------------------------------------------------------------ #

    def _write_heartbeat(self, force: bool = False, state: str = "running") -> None:
        interval = float(self.conf.get("heartbeat_interval_s", 30) or 0)
        now = time.time()
        if not force and interval > 0 and (now - self._last_heartbeat_ts) < interval:
            return
        self._last_heartbeat_ts = now
        seq = self.events.last_seq()
        hb = {"ts": _now(), "seq": seq, "state": state,
              "iteration_cursor": self.iteration_cursor}
        self._atomic_json(self.heartbeat_path, hb)
        self.events.append("heartbeat", {"seq": seq,
                                         "iteration_cursor": self.iteration_cursor},
                           "scheduler")
        self._set_super_state(state, cursor=self.iteration_cursor)

    def _set_super_state(self, state: str, cursor: Optional[int] = None) -> None:
        path = os.path.join(self.ws, "super.json")
        data = {}
        try:
            with open(path, "r", encoding="utf-8") as f:
                data = json.load(f)
        except (OSError, ValueError):
            data = {}
        if not isinstance(data, dict):
            data = {}
        data["state"] = state
        data["updated_at"] = _now()
        data["pid"] = os.getpid()
        data["heartbeat_path"] = self.heartbeat_path
        if cursor is not None:
            data["iteration_cursor"] = cursor
        self._atomic_json(path, data)

    @staticmethod
    def _atomic_json(path: str, data: Dict[str, Any]) -> None:
        d = os.path.dirname(os.path.abspath(path)) or "."
        fd, tmp = tempfile.mkstemp(prefix=".hb-", suffix=".tmp", dir=d)
        try:
            with os.fdopen(fd, "w", encoding="utf-8") as f:
                json.dump(data, f, ensure_ascii=False)
                f.flush()
                os.fsync(f.fileno())
            os.replace(tmp, path)
        except BaseException:
            try:
                os.unlink(tmp)
            except OSError:
                pass
            raise

    def _install_signal_handlers(self) -> None:
        """SIGTERM/SIGINT → 优雅停止（落完当前事件再退）。"""
        def handler(signum, frame):  # noqa: ARG001
            self.stop_requested = True
        try:
            self._prev_handler = signal.signal(signal.SIGTERM, handler)
            signal.signal(signal.SIGINT, handler)
        except ValueError:
            # 非主线程（测试）——无信号安装，直接轮询 stop_requested
            self._prev_handler = None

    def _restore_signal_handlers(self) -> None:
        if self._prev_handler is not None:
            try:
                signal.signal(signal.SIGTERM, self._prev_handler)
            except ValueError:
                pass
            self._prev_handler = None
        try:
            signal.signal(signal.SIGINT, signal.default_int_handler)
        except ValueError:
            pass

    # ------------------------------------------------------------------ #
    # env 元信息
    # ------------------------------------------------------------------ #

    def _load_env_meta(self) -> Dict[str, Any]:
        path = os.path.join(self.env_dir, "env.json")
        try:
            with open(path, "r", encoding="utf-8") as f:
                data = json.load(f)
            return data if isinstance(data, dict) else {}
        except (OSError, ValueError):
            return {}

    def _load_feature_axes(self) -> List[Dict[str, Any]]:
        path = os.path.join(self.env_dir, "feature_space.json")
        with open(path, "r", encoding="utf-8") as f:
            data = json.load(f)
        axes = data.get("axes") if isinstance(data, dict) else None
        if not isinstance(axes, list):
            raise ValueError("feature_space.json 缺 axes: %s" % path)
        return axes

    def _cell_entry(self, cell_id: str) -> Dict[str, Any]:
        """读单格快照（兼容 _RuntimeArchive.cell_entry 与基类 property cells）。"""
        if hasattr(self.archive, "cell_entry"):
            return self.archive.cell_entry(cell_id)
        return (_cells_snapshot(self.archive).get(cell_id) or {})

    # ---- 注入外部 Archive 无 bump_visit 时的内存影子（visit 真相在事件） ----

    def _shadow_bump(self, cell_id: str) -> None:
        self._visit_shadow[cell_id] = self._visit_shadow.get(cell_id, 0) + 1

    # ------------------------------------------------------------------ #
    # 门先证红（异常防御：门禁框架抛错视同失败）
    # ------------------------------------------------------------------ #

    def _safe_preverify_red(self) -> Dict[str, Any]:
        try:
            report = self.gate.preverify_red()
            return report if isinstance(report, dict) else {"ok": False}
        except Exception as exc:
            return {"ok": False, "error": "preverify_red raised: %s" % exc}
