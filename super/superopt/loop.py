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
import inspect
import json
import os
import re
import shutil
import signal
import subprocess
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
from superopt.cell_lock import CellLocks
from superopt.config import DEFAULTS
from superopt.events import EventWriter, make_experiment_id
from superopt.feature_space import FeatureSpace
from superopt import experiment_record
from superopt.gates import CascadeGate
from superopt.intervention import InterventionQueue
from superopt.ledger import Ledger as ExperimentLedger
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

# 相邻格实验记录摘要上限（迭代10/task51 降本）：相邻格只作灵感源，不让
# 其二篇完整记录撑大 builder 上下文（目标格精英记录仍用 1200）。
_NEIGHBOR_RECORD_LIMIT = 400

# task57 R1：infra_noop 判定的秒退墙（agent 总墙钟 < 5s 且无产出 →\# 基础设施空转形态，非实验失败）
_INFRA_NOOP_WALL_MS = 5000

# task57 R1：探活消息（极小 agent 派发——成功即 API 恢复）
_PROBE_PROMPT_TEXT = "回复 OK（探活消息，不要做任何其它事情）\n"


def _fmt_num(v: float) -> str:
    """性能数字的人读格式：整数不带尾零；浮点保留 1 位小数。"""
    if v == int(v):
        return "%d" % int(v)
    return "%.1f" % v


def _now() -> str:
    return datetime.now(_TZ).isoformat(timespec="seconds")


def _safe_fs_name(cell_id: str) -> str:
    """cell_id → 文件系统安全的目录/文件名片段。

    cell_id 形如 `impl_strategy=blocked|memory_layout=tiled`（含 `|`/`=`），
    在 macOS/Linux 下可直接做目录名；仅替换路径分隔符与残余控制字符，
    保持可读（profile_history/<cell>/ 与 designs/iter{N}_<cell>.json）。
    """
    return re.sub(r"[/\\\x00-\x1f]", "_", str(cell_id))


def _parse_last_json_line(text: str) -> Optional[Dict[str, Any]]:
    """从子进程 stdout 提取最后一行合法 JSON（三要素脚本的标准输出协议）。

    宽松匹配：取最后一个能 json.loads 成 dict 的非空行（脚本可能有 print
    前言或 stderr 混入 stdout）。解析不出 → None。
    """
    if not isinstance(text, str):
        return None
    for line in reversed(text.strip().splitlines()):
        line = line.strip()
        if not line or not line.startswith("{"):
            continue
        try:
            val = json.loads(line)
        except ValueError:
            continue
        if isinstance(val, dict):
            return val
    return None


def _coerce_text(v: Any) -> str:
    """subprocess 输出可能是 bytes/None → str（text=True 下正常是 str）。"""
    if v is None:
        return ""
    if isinstance(v, bytes):
        return v.decode("utf-8", "replace")
    return str(v)


def _profile_facts(profile: Optional[Dict[str, Any]]) -> List[str]:
    """profile 输出 → 可读事实摘要（designs 落盘的 profile_facts）。

    兼容热点列表（hotspots: [{func/name, pct/percent/ratio}]）与环境事实
    （env_facts: {k: v}）两种常见形状；未知形状不编造（返回空列表）。
    """
    if not isinstance(profile, dict):
        return []
    facts = []  # type: List[str]
    hs = profile.get("hotspots")
    if isinstance(hs, list):
        for h in hs[:3]:
            if isinstance(h, dict):
                # profile.py（三要素版）用 function 键；兼容 func/name（旧版/他域）
                name = (h.get("function") or h.get("func") or h.get("name")
                        or "?")
                val = h.get("pct")
                if val is None:
                    val = h.get("percent")
                if val is None:
                    val = h.get("ratio")
                facts.append("%s=%s" % (name, val if val is not None else ""))
            else:
                facts.append(str(h)[:80])
    ef = profile.get("env_facts")
    if isinstance(ef, dict):
        for k in list(ef.keys())[:4]:
            facts.append("%s=%s" % (k, ef.get(k)))
    return facts


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


# --------------------------------------------------------------------------- #
# 并发槽（task50）：每迭代可变状态的线程隔离
# --------------------------------------------------------------------------- #
# 并发实验时，迭代内的可变状态（成本累计 / 实验标记 / builder 产物 / TDD 痕迹…）
# 必须按线程隔离：worker 线程各持一份，主线程也持一份（adjust 收尾等）。
# 实现为 property → threading.local()，既有代码里的 self._xxx 读写全部不变。
_ITER_TL_FIELDS = (
    "_iter_cost", "_iter_agents", "_cur_exp_id", "_builder_outputs",
    "_tdd_trace", "_record_updated", "_recheck_done_this_iter",
    "_recheck_verdict", "_last_admit_verdict", "_last_reject_reason",
    "_record_this_iter", "_best_perf_before_iter",
    "_iter_self_check_real",  # task57 R1：agent 是否真实报告了自检内容（outcome 分类）
)


def _iter_tl_property(name: str) -> Any:
    """把 ``self.<name>`` 落到 threading.local()（并发槽的迭代态隔离）。"""
    key = "_tl_" + name

    def _get(self: Any) -> Any:
        return getattr(self._tl, key)

    def _set(self: Any, value: Any) -> None:
        setattr(self._tl, key, value)

    return property(_get, _set)


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

    # ---- 线程隔离的迭代态（task50：并发槽下每线程一份）----
    _iter_cost = _iter_tl_property('_iter_cost')
    _iter_agents = _iter_tl_property('_iter_agents')
    _cur_exp_id = _iter_tl_property('_cur_exp_id')
    _builder_outputs = _iter_tl_property('_builder_outputs')
    _tdd_trace = _iter_tl_property('_tdd_trace')
    _record_updated = _iter_tl_property('_record_updated')
    _recheck_done_this_iter = _iter_tl_property('_recheck_done_this_iter')
    _recheck_verdict = _iter_tl_property('_recheck_verdict')
    _last_admit_verdict = _iter_tl_property('_last_admit_verdict')
    _last_reject_reason = _iter_tl_property('_last_reject_reason')
    _record_this_iter = _iter_tl_property('_record_this_iter')
    _best_perf_before_iter = _iter_tl_property('_best_perf_before_iter')
    _iter_self_check_real = _iter_tl_property('_iter_self_check_real')

    def __init__(self, ws: str, cfg: Dict[str, Any],
                 agent_client: Optional[Any] = None,
                 gate: Optional[Any] = None,
                 event_writer: Optional[EventWriter] = None,
                 archive: Optional[Archive] = None,
                 feature_space: Optional[FeatureSpace] = None,
                 sampler: Optional[Sampler] = None,
                 ledger: Optional[CostLedger] = None,
                 queue: Optional[InterventionQueue] = None,
                 exp_ledger: Optional[Any] = None) -> None:
        self.ws = os.path.abspath(ws)
        self.cfg = cfg
        conf = dict(DEFAULTS)
        conf.update(cfg.get("config") or {})
        self.conf = conf

        # ---- 并发槽（task50）----
        # _tl：每线程的迭代态（见 _ITER_TL_FIELDS）；_state_lock：共享计数
        # （iteration_cursor / 饱和信号）的临界区；_ledger_lock：CostLedger 与
        # 实验台账的串行写（两者内部无锁，多 worker 并发 record 会丢更新）。
        self._tl = threading.local()
        self._state_lock = threading.Lock()
        self._ledger_lock = threading.Lock()

        self.events_path = os.path.join(self.ws, "events.jsonl")
        self.archive_path = os.path.join(self.ws, "archive.json")
        self.env_dir = cfg.get("env_path") or os.path.join(self.ws, "env")
        self.gates_dir = os.path.join(self.ws, "gates")
        self.codes_dir = os.path.join(self.ws, "codes")
        self.proposals_dir = os.path.join(self.ws, "proposals")
        self.heartbeat_path = os.path.join(self.ws, "heartbeat.json")
        # 三要素之「现状」的时间序列（迭代6：每迭代自动采集 profile）与
        # builder 决策落盘（designs/iter{N}_{cell}.json）——懒建目录
        self.profile_history_dir = os.path.join(self.ws, "profile_history")
        self.designs_dir = os.path.join(self.ws, "designs")
        # adjust 决策记录（迭代8/KR3）：<ws>/decisions/iter{N}.md——
        # 可解释性 + builder 执行参考（J-i8-4 裁决）
        self.decisions_dir = os.path.join(self.ws, "decisions")
        os.makedirs(self.codes_dir, exist_ok=True)
        os.makedirs(self.proposals_dir, exist_ok=True)
        os.makedirs(self.gates_dir, exist_ok=True)
        os.makedirs(self.decisions_dir, exist_ok=True)
        # 格级互斥（迭代9/task45，human fix 2）：一个格同时只有一个实验。
        # 内存集合 + 文件锁双保险（跨进程）；陈旧锁按 TTL 回收、启动时清孤儿。
        # 回调在调用线程（主线程/采样处）落事件——本模块自身不写事件。
        self.cell_locks = CellLocks(
            self.ws,
            ttl_s=float(conf.get("cell_lock_ttl_s", 3600) or 3600),
            on_stale=self._on_cell_lock_stale)
        self.max_parallel_experiments = max(
            1, int(conf.get("max_parallel_experiments", 1) or 1))
        # 连续「全格被占」跳过计数（超过 cell_lock_max_skips → 优雅停，防空转）
        self._lock_skip_count = 0

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
            # event_fsync（迭代10/task51）：默认 True；高频/并发实验可关（省 IO，
            # 崩溃可能丢尾行——events 读取有截断容错）
            self.events = event_writer or EventWriter(
                self.events_path,
                event_fsync=bool(conf.get("event_fsync", True)))
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
            # 实验台账（task26）：跨迭代实验档案（idea→实现→门禁→结论）。
            # 注意：self.ledger 已被 CostLedger（成本账本）占用——实验台账用
            # self.exp_ledger 区分（名字不同但语义即规格里的 Ledger）。
            # 台账是分析副产品，写失败不影响主循环。
            self.exp_ledger = exp_ledger or ExperimentLedger(self.ws)
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
        # 基础设施熔断（env_11 实测三连坑：配额耗尽→agent 0-token 秒退→
        # 调度器无熔断空转烧迭代号与冷格——最重一次 18 分钟 2108 次失败、
        # 386 个迭代号、全部未访格被烧毁）。连续 N 次「0-token 失败」→
        # 优雅停 + infra_failure_concluded 事件（human/中控据此换号/修号）。
        self._consec_zero_token_fails = 0
        self._infra_failure = False
        # task56 D 段：API 断连检测 / task56 A 段：turn_warn 连续告警
        # task57 R1 升级：连续 N 次 infra 失败（配额报错/空回）→ 熔断打开
        # （_api_outage_active）→ 循环层暂停（不推进 cursor、不建 iterNNN
        # 目录、不写空壳决策）+ 周期探活（_probe_api_recovery）恢复。
        # _consec_infra_fails：连续 infra 失败计数（任何非 infra 结果重置）。
        self._consec_infra_fails = 0
        self._api_outage_active = False
        self._turn_warn_streak = {}
        # ---- 饱和信号状态（task23 E 段）----
        # _best_perf：历史最优 perf（方向感知）；_stale_iters：自上次刷新的
        # 完成迭代数；_champion_misses：冠军格连续采样未刷新次数；
        # _record_this_iter：本迭代是否刷新了纪录（_step 尾更新 stale）
        self._best_perf = None  # type: Optional[float]
        self._stale_iters = 0
        self._champion_misses = 0
        self._record_this_iter = False
        # ---- 本迭代 agent 统计（ledger 条目用：role → {wall_ms, tokens, truncated}）
        self._iter_agents = {}  # type: Dict[str, Dict[str, Any]]
        # adjust 跨迭代并行：统计在收集时暂存，归属到「收集发生的迭代」
        self._adjust_stats_carry = {}  # type: Dict[str, Dict[str, Any]]
        # 本迭代入格结果（ledger.verdict：admit|replace|reject|no_candidate）
        self._last_admit_verdict = ""
        self._last_reject_reason = ""
        # ---- 迭代8：adjust 四步 / 实验记录 / debug 三触发 ----
        # 上一次 adjust 的迭代号：全局实验分析的区间起点（since_iter）
        self._last_adjust_iter = 0
        # 本迭代 builder 产物校验（research_report / record_updated）
        self._builder_outputs = {}  # type: Dict[str, Any]
        # 本迭代 coding 的 TDD 痕迹（test_solution.py 存在）与实验记录更新
        self._tdd_trace = None  # type: Optional[bool]
        self._record_updated = None  # type: Optional[bool]
        # 本迭代是否已做异常收益/容差带复验（每迭代至多 1 次）
        self._recheck_done_this_iter = False
        self._recheck_verdict = None  # type: Optional[str]
        self._recover_saturation_state()

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
        # ---- 格级互斥启动清理（迭代9/task45）----
        # 上个进程被 SIGKILL 时可能留下锁文件：持有者 pid 已死 → 立即清理
        # （不必等 TTL），避免恢复后的首个迭代空等。
        released = self.cell_locks.release_orphans()
        if released:
            self.events.append("cell_lock_orphans_released", {
                "count": len(released), "cells": released[:20]}, "scheduler")
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
                            "budget_exhausted": bool(self._budget_exhausted),
                            "infra_failure": bool(self._infra_failure)},
                           "scheduler")
        self._set_super_state("stopped")
        self._write_heartbeat(force=True, state="stopped")
        return EXIT_OK

    def _loop(self) -> None:
        """主循环分发（task50）：并发槽 >1 → 槽池；否则串行（原行为不变）。"""
        if self.max_parallel_experiments > 1:
            return self._loop_parallel()
        return self._loop_serial()

    def _alloc_cursor(self) -> int:
        """原子分配迭代号（槽池并发起槽时唯一号源）。"""
        with self._state_lock:
            self.iteration_cursor += 1
            return self.iteration_cursor

    def _adjust_due(self, adjust_every: int) -> bool:
        """跨边界触发判定（task54）：cursor - 上次 adjust >= adjust_every。

        等值判断（cursor % N == 0）在并发 cursor 跳跃（如 3→7 直接跳过 4）时会
        漏判；改为区间判断后两路径（串行/并发）语义一致。首轮 _last_adjust_iter=0
        → cursor 达到 N 即触发。
        """
        if adjust_every <= 0 or self.iteration_cursor <= 0:
            return False
        return (self.iteration_cursor - self._last_adjust_iter) >= adjust_every

    def _loop_parallel(self) -> None:
        """并发槽池（task50）：异格并行实验，一格同时只有一个实验。

        职责划分（保正确性的关键）：
        - **主线程**：采样 + 格锁、分配迭代号、写 ``experiment_started``、
          干预摄取 + **adjust 调度**（后台线程，与 worker 并行安全——task54 修复
          饥饿：原实现把 adjust 放在 ``if not slots`` 内，槽池每 tick 自动补满
          → 门永不打开 → 21 迭代 0 次 adjust 实证。adjust 是只读快照 + 后台
          线程：``_adjust_lock`` 单飞、archive RLock 快照、events 只读）；
        - **全局态操作**（热重载/裁决消费/epoch 复评）仍在**无在飞槽**时执行
          （它们改全局面貌，与 worker 的 admit 交叠有风险）；
        - **worker 线程**：``_step_locked``（builder/coding/debug/gate/admit），
          事件并行写（EventWriter 锁 + experiment_id，O_APPEND 原子），
          archive 写有 RLock，CostLedger/台账写经 ``_ledger_lock``；
        - 格锁在 worker 的 finally 释放（异常也不漏）。
        """
        max_iter = int(self.conf.get("max_iterations") or 0)
        adjust_every = int(self.conf.get("adjust_interval") or 0)
        poll_s = max(0.05, float(self.conf.get("slot_poll_s", 0.2) or 0.2))
        slots = {}  # type: Dict[int, Dict[str, Any]]
        next_slot = 0
        self.events.append("parallel_slots_enabled", {
            "max_parallel_experiments": self.max_parallel_experiments,
        }, "scheduler")
        while True:
            # ① adjust 收菜（后台线程结果 → 主线程应用）
            self._collect_adjust()
            # ② 收割已完成槽
            self._reap_slots(slots)
            if self.stop_requested:
                break
            if not self.ledger.within_budget():
                self._budget_exhausted = True
                break
            # ③ 干预摄取每个 tick 都做（human 的 stop/降权不该被在飞槽拖延；
            #    写入方是主线程，事件单写者语义不变）
            self._drain_interventions()
            if self.stop_requested:
                break
            # ③′ 熔断暂停（task57 R1）：断连/配额 → 停止补槽（在飞槽自然收尾），
            # 探活恢复后再继续（cursor / iterNNN 目录 / 决策都不再新增）
            if self._api_outage_active:
                if not self._probe_api_recovery():
                    break
            # ③″ adjust 调度：移出 if not slots（task54：并发下槽池总被补满，
            # 等值 + 无在飞槽的门 → adjust 永久饥饿）。跨边界判断 + 后台线程
            # 单飞；ctx 快照在主线程组装，与 worker 并行安全。
            if self._adjust_due(adjust_every):
                self._maybe_start_adjust()
            # ③‴ 其余全局态操作（热重载/裁决/epoch）仍在无在飞槽时执行
            if not slots:
                self._maybe_reload_feature_space()
                self._drain_decisions()
                if (self.iteration_cursor > 0
                        and self.iteration_cursor % EPOCH_ITERATIONS == 0):
                    self._reevaluate_pending()
            # ④ 补满在飞槽（每槽 = 一个实验：独占一格）
            while (len(slots) < self.max_parallel_experiments
                   and not self.stop_requested
                   and (max_iter <= 0 or self.iteration_cursor < max_iter)):
                cell_id = self._sample_and_lock()
                if cell_id is None:
                    break  # 全格被占/溢出停机（事件已落）
                cursor = self._alloc_cursor()
                exp_id = make_experiment_id(cell_id, cursor)
                slot = next_slot
                next_slot += 1
                self.events.append("experiment_started", {
                    "slot": slot, "iteration_id": cursor, "cell_id": cell_id,
                }, "scheduler", experiment_id=exp_id)
                th = threading.Thread(
                    target=self._slot_worker,
                    args=(slot, cell_id, cursor, exp_id),
                    name="superopt-slot-%d" % slot, daemon=True)
                slots[slot] = {"thread": th, "cell_id": cell_id,
                               "iteration_id": cursor, "started": time.time()}
                th.start()
            # ⑤ 退出判据：达到迭代上限且无在飞槽
            if (max_iter > 0 and self.iteration_cursor >= max_iter
                    and not slots):
                break
            if not slots and self.stop_requested:
                break
            time.sleep(poll_s)
        # ⑥ 收尾：等在飞槽优雅退出（grace 后记录超时；daemon 随进程退出）
        self._drain_slots(slots)

    def _slot_worker(self, slot: int, cell_id: str, cursor: int,
                     exp_id: str) -> None:
        """槽 worker：跑一次实验（已持格锁），结束时释放并落收尾事件。"""
        t0 = time.time()
        try:
            self._step_locked(cell_id, cursor=cursor, exp_id=exp_id)
        except Exception as exc:  # noqa: BLE001 —— 单实验失败不杀调度器
            try:
                self.events.append("experiment_failed", {
                    "slot": slot, "iteration_id": cursor, "cell_id": cell_id,
                    "error": str(exc)[:400],
                }, "scheduler", experiment_id=exp_id)
            except Exception:  # noqa: BLE001
                pass
        finally:
            try:
                self.cell_locks.release(cell_id)
            except Exception:  # noqa: BLE001
                pass
            try:
                self.events.append("experiment_finished", {
                    "slot": slot, "iteration_id": cursor, "cell_id": cell_id,
                    "wall_ms": int((time.time() - t0) * 1000),
                }, "scheduler", experiment_id=exp_id)
            except Exception:  # noqa: BLE001
                pass

    def _reap_slots(self, slots: Dict[int, Dict[str, Any]]) -> None:
        """收割已结束的槽（非阻塞）。"""
        for slot in list(slots):
            th = slots[slot]["thread"]
            if not th.is_alive():
                th.join(timeout=0.1)
                del slots[slot]

    def _drain_slots(self, slots: Dict[int, Dict[str, Any]]) -> None:
        """停止时等在飞槽收尾（grace 内）；超期记录 slot_stop_timeout。"""
        grace_s = float(self.conf.get("slot_stop_grace_s", 120) or 0)
        deadline = time.time() + max(0.0, grace_s)
        for slot in list(slots):
            th = slots[slot]["thread"]
            th.join(timeout=max(0.1, deadline - time.time()))
        leftover = [s for s, v in slots.items() if v["thread"].is_alive()]
        if leftover:
            try:
                self.events.append("slot_stop_timeout", {
                    "slots": leftover}, "scheduler")
            except Exception:  # noqa: BLE001
                pass
        else:
            for slot in list(slots):
                slots[slot]["thread"].join(timeout=0.1)
                del slots[slot]

    def _loop_serial(self) -> None:
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
            # ①′ 熔断暂停（task57 R1）：API 断连/配额耗尽 → 不推进 cursor、
            # 不建 iterNNN 目录、不写空壳决策——探活恢复后续跑（期间仍摄取
            # 干预，human 可停）
            if self._api_outage_active:
                if not self._probe_api_recovery():
                    break
            # ② 裁决决策消费（外置 adjudicator 的 decision 文件）
            self._drain_decisions()
            # ③ adjust（每 N 迭代；cursor 为已完成迭代数）——跨边界触发
            # （task54：cursor % N 在跳跃时漏判；串行/并发两路径语义统一）
            if self._adjust_due(adjust_every):
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
        """单迭代 = 一个实验：先抢占格锁（一个格同时只有一个实验）。"""
        cell_id = self._sample_and_lock()
        if cell_id is None:
            return  # 全格被其它实验占用（事件已落）；外层等待后重试
        try:
            self._step_locked(cell_id)
        finally:
            self.cell_locks.release(cell_id)

    def _sample_and_lock(self) -> Optional[str]:
        """采样 + 格锁（迭代9/task45，human fix 2）。

        - 随机采样撞锁 → 按权重降序尝试其它存活格（确定性兜底：并发时各
          实例/槽自然分道——**不同格可并行**）；
        - 全部被占 → 记 ``cell_locked_skip``，短暂等待后返回 None（**不消耗
          迭代号**：跳过的不是实验）；连续跳过超 ``cell_lock_max_skips``
          → 记 ``cell_lock_overflow`` 并优雅停（防死等空转）。
        """
        cell = self.sampler.sample_cell()  # 无可用格时抛 ValueError（原语义）
        if self.cell_locks.acquire(cell):
            self._lock_skip_count = 0
            return cell
        snap = _cells_snapshot(self.archive)
        alive = [cid for cid in self.fs.all_cells()
                 if cid != cell
                 and float((snap.get(cid) or {}).get("weight") or 0.0) > 0.0]
        alive.sort(key=lambda cid: -float(
            (snap.get(cid) or {}).get("weight") or 0.0))
        for cid in alive:
            if self.cell_locks.acquire(cid):
                self._lock_skip_count = 0
                self.events.append("cell_lock_resample", {
                    "iteration_cursor": self.iteration_cursor,
                    "sampled_cell": cell, "locked_to": cid,
                }, "scheduler")
                return cid
        # 全部候选格被其它实验占用
        self._lock_skip_count += 1
        self.events.append("cell_locked_skip", {
            "iteration_cursor": self.iteration_cursor,
            "sampled_cell": cell,
            "consecutive_skips": self._lock_skip_count,
        }, "scheduler")
        max_skips = int(self.conf.get("cell_lock_max_skips", 60) or 60)
        if max_skips > 0 and self._lock_skip_count >= max_skips:
            self.events.append("cell_lock_overflow", {
                "iteration_cursor": self.iteration_cursor,
                "consecutive_skips": self._lock_skip_count,
            }, "scheduler")
            self.stop_requested = True
            return None
        sleep_s = float(self.conf.get("cell_lock_skip_sleep_s", 1.0) or 0.0)
        if sleep_s > 0:
            time.sleep(sleep_s)
        return None

    def _on_cell_lock_stale(self, cell_id: str) -> None:
        """格锁陈旧回收上报（CellLocks 回调）→ 事件（可观测）。"""
        events = getattr(self, "events", None)
        if events is None:
            return
        try:
            events.append("cell_lock_stale_recovered",
                          {"cell_id": cell_id}, "scheduler")
        except Exception:  # noqa: BLE001 —— 事件写失败不破坏锁语义
            pass

    # ------------------------------------------------------------------ #
    # task57 R1：失败分类 + 熔断探活
    # ------------------------------------------------------------------ #

    def _iteration_outcome(self, candidate: Optional[str]) -> str:
        """三态失败分类（task57 R1）：iteration_ended.outcome 判据。

        delivered      有候选交框架终验并入库（admit/replace/延迟入库 pending）
        infra_noop     基础设施空转（agent 带 infra_failure 标记 / 全程
                       0-token / 总墙钟 < 5s 且无产出）——非实验失败，
                       度量上应从「失败率」里剔除（env_11 实测 68% 迭代
                       属此类，被旧口径计入「80% 无产出」）
        turn_truncated agent 回合预算截断（真实工作被攔断——修复方向是
                       提预算/产物抢救，不是「agent 失败」）
        agent_failed   真实实验失败（自检有内容但 fail / 门禁拒绝）
        no_output      其它无产出（如 agent 未报自检内容地空手而归）
        """
        if candidate is not None:
            return "delivered"
        agents = self._iter_agents or {}
        if any(bool(a.get("infra_failure")) for a in agents.values()):
            return "infra_noop"
        total_tokens = sum(int(a.get("tokens") or 0) for a in agents.values())
        total_wall = sum(int(a.get("wall_ms") or 0) for a in agents.values())
        if total_tokens <= 0 or total_wall < _INFRA_NOOP_WALL_MS:
            return "infra_noop"
        if any(bool(a.get("truncated")) for a in agents.values()):
            return "turn_truncated"
        if self._iter_self_check_real:
            return "agent_failed"
        return "no_output"

    def _probe_api_recovery(self) -> bool:
        """task57 R1：熔断暂停期间的探活循环（成功 → 恢复迭代推进）。

        - 每 api_outage_retry_s 探一次（极小 agent「回复 OK」，
          超时 api_outage_probe_timeout_s）；
        - 探活周期内仍摄取干预（human 可停——暂停不等于聋）；
        - 持续失败计入 0-token 熔断后备（agent_failure_breaker →
          infra_failure_concluded 优雅停，等 human 换号/修号）。

        返回 False = 停止请求（或熔断停机），外层退出主循环。
        """
        retry_s = int(self.conf.get("api_outage_retry_s", 60) or 0)
        probes = 0
        while not self.stop_requested:
            self._sleep_interruptible(retry_s)
            if self.stop_requested:
                return False
            self._drain_interventions()
            if self.stop_requested:
                return False
            probes += 1
            if self._run_api_probe():
                self.events.append("api_outage_recovered", {
                    "consecutive_infra": self._consec_infra_fails,
                    "probes": probes,
                    "diagnosis": "探活成功——API 恢复，继续迭代推进（暂停期间"
                                "cursor/格状态未被烧）",
                }, "scheduler")
                self._api_outage_active = False
                self._consec_infra_fails = 0
                self._consec_zero_token_fails = 0
                return True
            # 探活失败计入 0-token 熔断后备（持续断连 → 停机等 human）
            self._consec_zero_token_fails += 1
            if self._maybe_fire_infra_breaker():
                return False
        return False

    def _run_api_probe(self) -> bool:
        """task57 R1：探活——极小 agent 派发，成功即 API 恢复。

        有回复（valid 或 token>0）= 恢复；空回/异常 = 仍断连。探活不进
        agent_ended/账本（它不是实验派发，独立于迭代事件流）。
        """
        timeout_s = int(self.conf.get("api_outage_probe_timeout_s", 30) or 30)
        probe_prompt = os.path.join(self.ws, "probe_prompt.md")
        try:
            if not os.path.isfile(probe_prompt):
                with open(probe_prompt, "w", encoding="utf-8") as f:
                    f.write(_PROBE_PROMPT_TEXT)
            result = self._call_client("probe", probe_prompt, self.ws,
                                       timeout_s, None, None, turn_soft_warn=0)
        except Exception:  # noqa: BLE001 —— 探活异常 = 仍断连
            return False
        total_tokens = float((getattr(result, "usage", None) or
                              {}).get("total_tokens") or 0)
        return bool(getattr(result, "valid", False)) or total_tokens > 0.0

    def _sleep_interruptible(self, seconds: float) -> None:
        """可中断睡眠（stop_requested 即刻醒——探活周期不聋）。"""
        deadline = time.monotonic() + max(0.0, float(seconds or 0.0))
        while not self.stop_requested:
            remain = deadline - time.monotonic()
            if remain <= 0:
                return
            time.sleep(min(0.05, remain))

    def _maybe_fire_infra_breaker(self) -> bool:
        """task57 R1（factor 自原内联块）：0-token 熔断后备。

        连续 agent_failure_breaker 次 0-token 失败 → infra_failure_concluded
        + 优雅停（返回 True = 触发）。0-token 秒退是认证/配额/供应商级故障
        的稳定信号；真实验失败都烧 token。
        """
        breaker = int(self.conf.get("agent_failure_breaker", 8) or 0)
        if breaker <= 0:
            return False
        if self._consec_zero_token_fails < breaker or self._infra_failure:
            return False
        self._infra_failure = True
        self.stop_requested = True
        self.events.append("infra_failure_concluded", {
            "consecutive_zero_token_fails": self._consec_zero_token_fails,
            "threshold": breaker,
            "diagnosis": "连续 %d 次 agent 0-token 失败——认证/配额/"
                          "供应商级故障（非实验失败）。调度器优雅停机；"
                          "修复凭据后 rick super start 续跑（工作区状态"
                          "全持久）。" % self._consec_zero_token_fails,
        }, "scheduler")
        return True

    def _track_infra_failure(self, result: Any) -> None:
        """task57 R1：agent 结果的 infra 计数 + 断连熔断 + 0-token 后备。

        infra 判定：client 层配额标记（AgentResult.infra_failure）或空回
        形态（0-token + exit 0 + 秒退）。连续 api_outage_threshold 次 →
        熔断打开（_api_outage_active）：本迭代剩余派发被短路，循环层
        暂停 + 探活（替代 task56 D 的内联 sleep——那会阻塞事件流与心跳）。
        """
        total_tokens = float((result.usage or {}).get("total_tokens") or 0)
        # 0-token 熔断后备（与原内联块同语义）
        breaker = int(self.conf.get("agent_failure_breaker", 8) or 0)
        if breaker > 0:
            if (not result.valid) and total_tokens <= 0.0:
                self._consec_zero_token_fails += 1
            elif result.valid or total_tokens > 0.0:
                self._consec_zero_token_fails = 0
            self._maybe_fire_infra_breaker()
        # 断连/配额熔断（R1 主判据）
        outage_thr = int(self.conf.get("api_outage_threshold", 3) or 0)
        if outage_thr > 0:
            is_infra = bool(getattr(result, "infra_failure", False)) or (
                (not result.valid) and total_tokens <= 0.0
                and result.exit_code in (0, None)
                and int(result.wall_ms or 0) < 10000)
            if is_infra:
                self._consec_infra_fails += 1
            else:
                self._consec_infra_fails = 0
            if self._consec_infra_fails >= outage_thr \
                    and not self._api_outage_active:
                self._api_outage_active = True
                retry_s = int(self.conf.get("api_outage_retry_s", 60) or 60)
                self.events.append("api_outage_detected", {
                    "consecutive_empty": self._consec_infra_fails,
                    "threshold": outage_thr,
                    "retry_s": retry_s,
                    "diagnosis": "连续 %d 次 infra 失败（配额报错/空回："
                                "0-token/exit=0/秒退）——疑似 API 断连或额度"
                                "耗尽。熔断：暂停迭代推进（不烧 cursor/不建"
                                "目录），每 %ds 探活，恢复后续跑。"
                                % (self._consec_infra_fails, retry_s),
                }, "scheduler")

    def _rescue_artifact(self, iter_dir: str, since_epoch: float) -> Optional[str]:
        """task57 R2：产物抢救——找 agent 启动后新落盘的候选文件。

        契约（与 agents._find_fresh_artifact 同规）：solution.*/candidate.*
        源文件、非空、mtime ≥ since_epoch。种子父本（copy2 保留源 mtime）
        天然排除——只有本轮 agent 写的才算「做完了但没说」的物证。
        """
        try:
            names = sorted(os.listdir(iter_dir))
        except OSError:
            return None
        for name in names:
            if not (name.startswith("solution") or name.startswith("candidate")):
                continue
            if not name.endswith((".py", ".cpp", ".cc", ".cxx", ".c")):
                continue
            p = os.path.join(iter_dir, name)
            try:
                if not os.path.isfile(p) or os.path.getsize(p) <= 0:
                    continue
                if os.path.getmtime(p) < since_epoch:
                    continue
            except OSError:
                continue
            return p
        return None

    def _step_locked(self, cell_id: str,
                     cursor: Optional[int] = None,
                     exp_id: Optional[str] = None) -> None:
        """单迭代主体（已持有 cell_id 的格锁；_step 的 finally 保证释放）。

        task49：本迭代 = 一次实验，唯一标记 ``experiment_id``（格+迭代号）；
        迭代内全部事件都带该标记（并行写入时用 read_experiment / grep 还原
        单实验事件流），运维事件（心跳/干预/裁决）不带。
        """
        # 并发槽（task50）：迭代号与实验标记由主线程在起槽时分配（原子），
        # 串行路径（_step）未传入时在此分配——两条路径语义一致。
        if cursor is None:
            cursor = self._alloc_cursor()
        if exp_id is None:
            exp_id = make_experiment_id(cell_id, cursor)
        # 实验标记（格 + 迭代号；cell_safe 形态，见 events.make_experiment_id）
        self._cur_exp_id = exp_id
        iter_started = time.time()
        self._iter_cost = {"wall_ms": 0.0, "tool_calls": 0.0,
                           "tokens": 0.0, "cost_usd": 0.0}
        with self._state_lock:
            # adjust 收集侧归属：拷贝并清空（多槽并发时只归属到第一个开跑的槽，
            # 避免重复计入）
            self._iter_agents = dict(self._adjust_stats_carry)
            self._adjust_stats_carry = {}
        self._best_perf_before_iter = self._best_perf  # ledger perf_delta 基准
        self._last_admit_verdict = ""
        self._last_reject_reason = ""
        self._record_this_iter = False
        # 迭代8：每迭代状态重置（builder 产物校验 / TDD 痕迹 / 复验）
        self._builder_outputs = {}
        self._tdd_trace = None
        self._record_updated = None
        self._recheck_done_this_iter = False
        self._recheck_verdict = None
        # task57 R1：outcome 分类的本迭代标记（agent 是否真实报告了自检内容）
        self._iter_self_check_real = False
        self.events.append("iteration_started",
                           {"iteration_id": cursor, "cell_id": cell_id},
                           "scheduler", experiment_id=self._cur_exp_id)

        # ---- 状态 1：visit 递增 + cell_sampled 事件（采样与格锁已在上游
        #      _sample_and_lock 完成：一个格同时只有一个实验；visit 唯一递增点） ----
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
        }, "scheduler", experiment_id=self._cur_exp_id)

        # ---- 迭代工作区（codes/{uuid}/ 隔离快照） ----
        iter_dir = os.path.join(
            self.codes_dir, "iter%03d-%s" % (cursor, uuid.uuid4().hex[:8]))
        os.makedirs(iter_dir, exist_ok=True)

        # ---- 状态 2：builder（选父格 → experiment_plan） ----
        plan = self._run_builder(cell_id, cursor, iter_dir)

        # ---- 状态 3：coding（自检仅参考）→ debug（≤M 次） ----
        candidate = self._run_coding(plan, cell_id, cursor, iter_dir)

        # ---- 状态 4：框架验证 + 入格 ----
        gr = None  # type: Optional[Any]
        if candidate is not None:
            gr = self._verify_and_admit(candidate, cell_id, cursor, plan)
            # 框架门禁失败 → debug 接管（task23 D 段）：env_8 实测 it19
            # soft_gate 失败后直接丢弃——但门禁给了比自检更权威的
            # failure_stage/metrics，本可修复。每迭代仅 1 次（防死循环）。
            if (gr is not None and not gr.passed
                    and self._gate_fixable(gr.failure_stage)):
                fixed = self._run_gate_debug(gr, plan, cell_id, cursor, iter_dir)
                if fixed is not None:
                    gr = self._verify_and_admit(fixed, cell_id, cursor, plan)
                    if gr is not None and gr.passed:
                        candidate = fixed
                    else:
                        candidate = None  # 降级：重验仍挂 → 本迭代无产出
                else:
                    candidate = None
        # else：coding/debug 全失败 → 已记失败事件，跳过终验（本迭代无产出）

        # ---- 饱和状态维护（task23 E 段：每迭代完一次；并发槽下共享计数加锁）----
        with self._state_lock:
            if not self._record_this_iter:
                self._stale_iters += 1
            else:
                self._stale_iters = 0
            # 冠军格连续未刷新计数：本迭代采了当前冠军格且无新纪录 → miss+1
            champ = self._champion_cell()
            if (champ is not None and champ == cell_id
                    and not self._record_this_iter):
                self._champion_misses += 1
            elif self._record_this_iter:
                self._champion_misses = 0

        # ---- 收尾 ----
        wall_ms = int((time.time() - iter_started) * 1000)
        self.events.append("iteration_ended", {
            "iteration_id": cursor, "cell_id": cell_id,
            "wall_ms": wall_ms + int(self._iter_cost.get("wall_ms", 0.0)),
            "tool_calls": int(self._iter_cost.get("tool_calls", 0.0)),
            "tokens": int(self._iter_cost.get("tokens", 0.0)),
            "cost_usd": round(float(self._iter_cost.get("cost_usd", 0.0)), 6),
            "had_candidate": candidate is not None,
            # 成本可解释性（task46）：本迭代按角色的 token/wall/tool_calls 归属
            # ——每次迭代的成本可归因到 agent（cost_report 与 adjust 决策共用）
            "by_role": {r: {"tokens": int(a.get("tokens") or 0),
                             "wall_ms": int(a.get("wall_ms") or 0),
                             "tool_calls": int(a.get("tool_calls") or 0)}
                        for r, a in self._iter_agents.items()},
            # 迭代8：builder 产物校验 / coding TDD 痕迹 / debug 复验结论
            # （事件标注，缺失即质量缺口——trajectory 分析与自改进读这些）
            "builder_outputs": dict(self._builder_outputs),
            "tdd_trace": self._tdd_trace,
            "record_updated": self._record_updated,
            "recheck_verdict": self._recheck_verdict,
            # task57 R1：三态失败分类（度量口径分离——复盘从「猜」变成「读」）
            "outcome": self._iteration_outcome(candidate),
        }, "scheduler", experiment_id=self._cur_exp_id)
        # 每迭代落盘档案：bump_visit 只改内存，若本迭代无 admit/epoch 则到收尾都
        # 不 save——kill -9 后重启虽能从事件恢复，但「正常收尾的档案」会缺新采样
        # 格的 visit（job_47 e2e 实测投影1≠档案0）。每迭代一次 save，崩溃窗口
        # 从「上次 admit」缩到「本迭代」（事件流仍是唯一真相，恢复幂等）。
        self.archive.save()
        # 实验台账（task23 G 段）：跨迭代档案——idea→实现→门禁→结论一行
        self._append_ledger_entry(cursor, cell_id, plan, gr)

    # ---- builder ----

    def _run_builder(self, cell_id: str, cursor: int, iter_dir: str) -> Dict[str, Any]:
        entry = self._cell_entry(cell_id)
        elite = entry.get("elite") or {}
        cmds = self._self_check_and_profile_cmds()
        elems = self._element_cmds()
        # 三要素之「现状」（迭代6）：每迭代 builder 前自动采集 profile 归档——
        # 形成 profile_history 时间序列（失败静默不阻断）
        profile = self._collect_profile(cell_id, cursor)
        ctx = {
            "task": self.task_name, "perf_desc": self.perf_desc,
            "perf_type": self.perf_type,
            "feature_axes": self.fs.axes,
            "iteration_id": cursor,
            "cell_id": cell_id,
            "cell_descriptor": self.fs.describe_cell(cell_id) or {},
            "cell_elite": self._or_default(elite or None, "（无精英——本格首访）"),
            "cell_visit": entry.get("visit", 0),
            "cell_weight": entry.get("weight", 1.0),
            "neighbor_elites": self._neighbor_elites(cell_id),
            "history_failures": self._recent_gate_failures(5),
            # 优化空间③（job_47 首跑实测）：同轴历史性能对照——避免重复已被
            # 轴级证据证伪的方向（如「row_major 劣势已在两分支独立复现」）
            "axis_perf": self._axis_perf_summary(),
            # ---- 迭代5（env_8 信息审计）：消除硬编码 + 注入上下文 ----
            "env_dir": self.env_dir,
            "self_check_cmd": cmds["self_check_cmd"],
            "perf_profile_cmd": cmds["perf_profile_cmd"],
            "saturation": self._or_default(self._saturation_signal(), "（未饱和）"),
            "domain_brief": self._domain_brief(),
            "perf_timeline": self._perf_timeline(10),
            "budget_status": self._budget_status(),
            "ledger_brief": self._ledger_brief(8),
            "gate_defs": self._gate_defs(),
            "interventions": self._recent_interventions(3),
            # ---- 迭代6：三要素命令 + profile 历史轨迹 + 决策落盘 ----
            "profile_cmd": elems["profile_cmd"],
            "benchmark_cmd": elems["benchmark_cmd"],
            "verify_cmd": elems["verify_cmd"],
            "gate_check_cmd": elems["gate_check_cmd"],
            "target_profile_history": self._or_default(
                self._profile_history(cell_id, 3), "（本格暂无 profile 历史）"),
            "parent_profile_history": self._or_default(
                self._parent_profile_history(cell_id), "（无跨格父本）"),
            "inspiration_profile_history": self._or_default(
                self._inspiration_profile_history(cell_id),
                "（无邻格灵感数据）"),
            "designs_dir": self.designs_dir,
            "recent_designs": self._or_default(
                self._recent_designs(3), "（暂无历史决策）"),
            # ---- 迭代7：检索（第四类资产）——命令 + 预生成 brief ----
            "research_cmd": self._research_cmd("builder", cell_id),
            "research_brief": self._or_default(
                self._research_brief("builder", cell_id),
                "（未提供检索指引——跳过检索）"),
            # ---- 迭代8：实验记录（跨 agent 恒常上下文）+ 相邻格样本 + 落盘路径 ----
            "target_record": self._or_default(
                self._record_for_code_path((elite or {}).get("code_path")),
                "（本格首访——无历史实验记录）"),
            "neighbor_samples": self._or_default(
                self._neighbor_samples(cell_id, 2),
                "（无相邻格样本——全库仅本格）"),
            "research_report_path": os.path.join(iter_dir,
                                                 "research_report.md"),
            "experiment_record_path": experiment_record.path_for(iter_dir),
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
        # 迭代6：builder 决策落盘（designs/iter{N}_{cell}.json）——
        # 论文/灵感/推理/计划/现状事实的完整决策档案，供 gen_env 与中控
        # 会话读进化轨迹（super-loop Step 3/7）。写失败不影响主循环。
        self._write_design(cell_id, cursor, result, plan, profile)
        # 迭代8：builder 产物校验（调研报告 / 实验记录）——缺失即质量缺口，
        # 记进 iteration_ended.builder_outputs（轨迹分析与自改进读它）
        rr_path = os.path.join(iter_dir, "research_report.md")
        rec_path = experiment_record.path_for(iter_dir)
        self._builder_outputs = {
            "research_report": os.path.isfile(rr_path),
            "record_updated": os.path.isfile(rec_path),
            "experiment_record_path": rec_path,
            "research_report_path": rr_path,
        }
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
        cmds = self._self_check_and_profile_cmds()
        self_check_cmd = cmds["self_check_cmd"]
        elems = self._element_cmds()
        ctx = {
            "task": self.task_name, "perf_desc": self.perf_desc,
            "perf_type": self.perf_type,
            "iteration_id": cursor,
            "experiment_plan": plan,
            "workdir": iter_dir,
            "env_dir": self.env_dir,
            "self_check_cmd": self_check_cmd,
            # ---- 迭代5：门禁定义/基线/工作区文件/台账（env_8 审计） ----
            "gate_defs": self._gate_defs(),
            "perf_baseline": self._or_default(self._perf_baseline(cell_id), "（本格暂无精英基线）"),
            "workdir_files": self._workdir_files(iter_dir),
            "ledger_brief": self._ledger_brief(8),
            "saturation": self._or_default(self._saturation_signal(), "（未饱和）"),
            # ---- 迭代6：三要素命令（与框架同一套资产；自行按需调用） ----
            "profile_cmd": elems["profile_cmd"],
            "benchmark_cmd": elems["benchmark_cmd"],
            "verify_cmd": elems["verify_cmd"],
            "gate_check_cmd": elems["gate_check_cmd"],
            # ---- 迭代7：检索（role=coding：使用手册/代码样例） ----
            "research_cmd": self._research_cmd("coding", cell_id),
            "research_brief": self._or_default(
                self._research_brief("coding", cell_id),
                "（未提供检索指引——跳过检索）"),
            # ---- 迭代8：实验记录（恒常上下文）+ builder 调研报告 + 落盘路径 ----
            "experiment_record": self._or_default(
                experiment_record.read(iter_dir),
                "（experiment_record.md 缺失——builder 未落盘）"),
            "research_report": self._or_default(
                self._read_text(os.path.join(iter_dir, "research_report.md"),
                                3000),
                "（research_report.md 缺失——builder 未落盘调研报告）"),
            "coding_plan_path": os.path.join(iter_dir, "coding_plan.md"),
            "test_path": os.path.join(iter_dir, "test_solution.py"),
        }
        prompt = render("coding", self.ws, ctx)
        prompt_path = write_prompt(iter_dir, "coding", prompt)
        # 迭代8：实验记录 mtime 基线（coding 是否把执行过程追加进记录）
        rec_path = experiment_record.path_for(iter_dir)
        rec_mtime_before = self._mtime_ns(rec_path)
        # task57 R2：产物新鲜度基线（agent 启动前——种子父本 copy2 保留旧
        # mtime，本轮新落盘的才算「做完了但没说」的物证）
        agent_t0 = time.time()
        result = self._dispatch_agent("coding", prompt_path, iter_dir, cursor)
        # TDD 痕迹：测试文件是否落盘（模板要求先写测试再实现）
        self._tdd_trace = os.path.isfile(os.path.join(iter_dir,
                                                      "test_solution.py"))
        rec_mtime_after = self._mtime_ns(rec_path)
        if rec_mtime_before is not None and rec_mtime_after is not None:
            self._record_updated = rec_mtime_after > rec_mtime_before
        elif rec_mtime_after is not None:
            self._record_updated = True
        else:
            self._record_updated = False

        candidate = self._candidate_path(result, iter_dir)
        self_check = self._self_check_of(result)
        # task57 R1：outcome 分类标记（agent 是否真实报告了自检内容）。
        # 仅计 data.self_check 里的真实报告（有 metrics/stage 内容）；
        # _self_check_of 的合成 fallback（stage 恒为 self_check）不算——
        # 没报告过内容 → no_output 而非 agent_failed。
        reported_sc = result.data.get("self_check")
        self._iter_self_check_real = (
            isinstance(reported_sc, dict) and "passed" in reported_sc
            and bool(reported_sc.get("metrics") or reported_sc.get("stage")))
        self.events.append("candidate_produced", {
            "iteration_id": cursor, "cell_id": cell_id,
            "role": "coding",
            "code_path": candidate or "",
            "self_check": self_check,
            "diff_path": result.data.get("diff_path"),
        }, "agent:coding", experiment_id=self._cur_exp_id)

        if self_check.get("passed"):
            return candidate

        # ---- task57 R2：产物抢救——「做完了但没说」不再等于「没做」----
        # agent 未完成输出协议（result 无效/截断——无 JSON）但工作区有
        # 本轮新落盘的候选产物 → 不触发 debug 重做（debug 从头重做一份
        # 实现是 env_11 实测最大的 token 浪费源），产物直接交框架
        # gate_check 终验（与正常 candidate 同一路径——终验才是真判据）。
        # valid=True 的诚实自检失败不抢：agent 协议正常，走既有 debug 循环。
        if (not result.valid or result.truncated):
            rescued = self._rescue_artifact(iter_dir, agent_t0)
            if rescued is not None:
                self.events.append("artifact_rescued", {
                    "iteration_id": cursor, "cell_id": cell_id, "role": "coding",
                    "code_path": rescued,
                    "reason": "agent_completed_but_no_json",
                    "agent_valid": bool(result.valid),
                    "agent_truncated": bool(result.truncated),
                }, "scheduler", experiment_id=self._cur_exp_id)
                return rescued

        # task56 C 段：self_check 空 metrics 不触发 debug——agent 被截断/没
        # 跑完自检（不是实验失败），触发 debug 只会浪费 token。直接返回
        # candidate 让框架 gate_check 做终验（终验才是真判据）。
        if not self_check.get("passed") and not self_check.get("metrics") \
                and not self_check.get("stage"):
            self.events.append("self_check_skip_debug", {
                "iteration_id": cursor, "cell_id": cell_id,
                "reason": "empty_metrics_no_debug",
                "detail": "agent 未完成自检（截断/中断）——跳过 debug 循环，"
                          "候选直接交框架 gate_check 终验",
            }, "scheduler", experiment_id=self._cur_exp_id)
            return candidate

        # ---- debug 循环（≤M 次；每次一条 fix_report 载体事件） ----
        m = int(self.conf.get("debug_max_attempts") or 0)
        last_err = json.dumps(self_check, ensure_ascii=False)
        for attempt in range(1, m + 1):
            # task57 R2：debug 侧新鲜度基线（本轮 debug 内的写盘才算修复产物；
            # coding 阶段的旧文件 mtime 早于 debug_t0，不会被误抢）
            debug_t0 = time.time()
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
            }, "agent:debug", experiment_id=self._cur_exp_id)
            if dcheck.get("passed"):
                return dpath or candidate
            # task57 R2：debug 产物抢救——debug agent 改了代码但没发 JSON
            #（截断/空回/无效）→ 修复产物直接交终验，不再追加 debug 重做
            if not debug_result.valid or debug_result.truncated:
                d_rescued = self._rescue_artifact(iter_dir, debug_t0)
                if d_rescued is not None:
                    self.events.append("artifact_rescued", {
                        "iteration_id": cursor, "cell_id": cell_id,
                        "role": "debug", "attempt": attempt,
                        "code_path": d_rescued,
                        "reason": "agent_completed_but_no_json",
                        "agent_valid": bool(debug_result.valid),
                        "agent_truncated": bool(debug_result.truncated),
                    }, "scheduler", experiment_id=self._cur_exp_id)
                    return d_rescued
            last_err = json.dumps(dcheck, ensure_ascii=False)
        # M 次仍未修复 → 记失败（iteration_ended.had_candidate=False 由调用方收尾）
        return None

    def _run_debug(self, plan: Dict[str, Any], cell_id: str, cursor: int,
                   iter_dir: str, attempt: int, max_attempts: int,
                   self_check: Dict[str, Any], last_err: str,
                   gate_failure: Optional[Dict[str, Any]] = None,
                   perf_gain_pct: Optional[float] = None,
                   recheck_reason: Optional[str] = None) -> Any:
        cmds = self._self_check_and_profile_cmds()
        elems = self._element_cmds()
        ctx = {
            "task": self.task_name, "perf_desc": self.perf_desc,
            "perf_type": self.perf_type,
            "iteration_id": cursor,
            "experiment_plan": plan,
            "workdir": iter_dir,
            "env_dir": self.env_dir,
            "self_check_cmd": cmds["self_check_cmd"],
            "attempt": attempt, "max_attempts": max_attempts,
            "self_check_result": self_check,
            "failure_stage": self_check.get("stage"),
            "stderr_tail": last_err[-400:],
            # ---- 迭代5：框架门禁失败上下文 + 共用资产 ----
            "gate_failure": self._or_default(gate_failure or None, "（无——本次为自检失败触发）"),
            "perf_profile_cmd": cmds["perf_profile_cmd"],
            "gate_defs": self._gate_defs(),
            "ledger_brief": self._ledger_brief(8),
            "saturation": self._or_default(self._saturation_signal(), "（未饱和）"),
            # ---- 迭代6：三要素命令（debug 用三者分段定位失败） ----
            "profile_cmd": elems["profile_cmd"],
            "benchmark_cmd": elems["benchmark_cmd"],
            "verify_cmd": elems["verify_cmd"],
            "gate_check_cmd": elems["gate_check_cmd"],
            # ---- 迭代7：检索（role=debug：错误语义/正确用法/已知坑） ----
            "research_cmd": self._research_cmd("debug", cell_id),
            "research_brief": self._or_default(
                self._research_brief("debug", cell_id),
                "（未提供检索指引——跳过检索）"),
            # ---- 迭代8：实验记录（先读它再判断）+ 复验上下文 ----
            "experiment_record": self._or_default(
                experiment_record.read(iter_dir),
                "（experiment_record.md 缺失——builder 未落盘）"),
            "perf_gain_pct": self._or_default(
                perf_gain_pct, "（不适用——非异常收益复验）"),
            "recheck_reason": self._or_default(
                recheck_reason, "（无——本次为自检/门禁失败触发）"),
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
        sc = os.path.join(self.env_dir, "skills", "self_check.py")
        return "%s %s --candidate ." % (self._python_cmd(), sc)

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
                          plan: Dict[str, Any]) -> Any:
        """框架终验 + 入格。返回 GateResult（供 _step 的 gate→debug 判断）。"""
        entry = self._cell_entry(cell_id)
        elite = entry.get("elite")
        history = [h.get("perf_value") for h in (entry.get("history") or [])
                   if h.get("perf_value") is not None]
        gr = self.gate.verify(candidate,
                              elite_perf=(elite or {}).get("perf_value"),
                              history=history)
        self._emit_gate_result(gr, cursor, cell_id)

        if gr.passed:
            # 迭代8/KR6：门禁通过后仍可能复验——①异常收益（超 human 确认阈值，
            # 防作弊/误测的性能结论进精英地图）②软门禁容差带内（无明显收益）。
            # 每迭代至多 1 次；debug 结论 keep_original → 原候选照常入格。
            reason = self._recheck_reason(gr, cell_id)
            if reason is not None and not self._recheck_done_this_iter:
                self._recheck_done_this_iter = True
                iter_dir = os.path.dirname(os.path.abspath(candidate))
                fixed, verdict = self._run_recheck_debug(
                    plan, cell_id, cursor, iter_dir, gr, reason)
                self._recheck_verdict = verdict
                if fixed is not None:
                    gr2 = self.gate.verify(
                        fixed, elite_perf=(elite or {}).get("perf_value"),
                        history=history)
                    self._emit_gate_result(gr2, cursor, cell_id,
                                           recheck=True)
                    if gr2.passed:
                        candidate, gr = fixed, gr2
            self._admit(candidate, cell_id, cursor, plan, gr, had_elite=bool(elite))
            self.env_fail_streak = 0
            return gr

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
            }, "scheduler", experiment_id=getattr(self, "_cur_exp_id", None))
            if self.env_fail_streak >= ENV_FAIL_STREAK_LIMIT:
                self.stop_requested = True
        else:
            # 正常实验失败（正确性/性能不达标）：记失败继续下迭代
            self.env_fail_streak = 0
            self._last_admit_verdict = "reject"
            self._last_reject_reason = str(gr.failure_stage or "gate_failed")
        return gr

    # 框架门禁失败→debug 的可修阶段（stage1 是环境坏，交给环境失败结论化）
    _GATE_FIXABLE_STAGES = ("stage2_correctness", "stage3_perf", "soft_gate")

    @classmethod
    def _gate_fixable(cls, failure_stage: Any) -> bool:
        return isinstance(failure_stage, str) and \
            failure_stage in cls._GATE_FIXABLE_STAGES

    def _run_gate_debug(self, gr: Any, plan: Dict[str, Any], cell_id: str,
                        cursor: int, iter_dir: str) -> Optional[str]:
        """框架门禁失败 → debug 修复（task23 D 段：每迭代仅 1 次）。

        attempt=100 起编号（与自检 debug 的 1..M 区分，事件可归因）；
        返回候选路径（无法产出 → None）。
        """
        failure = {
            "failure_stage": gr.failure_stage,
            "metrics": gr.metrics or {},
            "details": (gr.details or "")[-600:],
        }
        self_check = {"passed": False, "stage": gr.failure_stage or "gate",
                      "metrics": gr.metrics or {}}
        last_err = json.dumps(failure, ensure_ascii=False)
        result = self._run_debug(plan, cell_id, cursor, iter_dir,
                                 attempt=100, max_attempts=1,
                                 self_check=self_check, last_err=last_err,
                                 gate_failure=failure)
        dpath = self._candidate_path(result, iter_dir)
        self.events.append("candidate_produced", {
            "iteration_id": cursor, "cell_id": cell_id,
            "role": "debug", "attempt": 100,
            "code_path": dpath or "",
            "fix_report": result.data.get("fix_report")
            or {"attempts": 100, "root_cause": None, "changes": []},
            "self_check": {"passed": bool(dpath), "stage": "gate_fix",
                           "metrics": {}},
        }, "agent:debug")
        return dpath

    def _append_ledger_entry(self, cursor: int, cell_id: str,
                             plan: Dict[str, Any], gr: Any) -> None:
        """迭代收尾写实验台账（task23 G 段）：跨迭代档案。"""
        gate = None  # type: Optional[Dict[str, Any]]
        if gr is not None:
            perf = (getattr(gr, "metrics", None) or {}).get("perf") or {}
            gate = {
                "passed": bool(getattr(gr, "passed", False)),
                "failure_stage": getattr(gr, "failure_stage", None),
                "perf_value": perf.get("value"),
            }
        verdict = self._last_admit_verdict or (
            "reject" if (gr is not None and not gr.passed) else "no_candidate")
        prev_best = getattr(self, "_best_perf_before_iter", None)
        perf_value = (gate or {}).get("perf_value")
        perf_delta = None
        if isinstance(perf_value, (int, float)):
            elite = self._cell_entry(cell_id).get("elite") or {}
            base = elite.get("perf_value")
            # 同格同解时 base==perf（无 delta 信息）；用历史最优作参照
            ref = prev_best if prev_best is not None else None
            if isinstance(base, (int, float)) and isinstance(ref, (int, float)):
                perf_delta = round(float(perf_value) - float(ref), 6)
        reject_reason = None
        if gate is not None and not gate["passed"]:
            reject_reason = str(gate.get("failure_stage") or "unknown")
        if reject_reason is None and self._last_reject_reason:
            reject_reason = self._last_reject_reason
        idea = plan.get("idea") if isinstance(plan, dict) else None
        entry = {
            "iteration_id": cursor,
            "cell_id": cell_id,
            "idea": idea,
            "parents": (plan.get("parents") if isinstance(plan, dict) else None),
            "plan_steps_count": (len(plan.get("plan_steps") or [])
                                 if isinstance(plan, dict) else 0),
            "gate": gate,
            "verdict": verdict,
            "perf_delta": perf_delta,
            "reject_reason": reject_reason,
            "agents": dict(self._iter_agents),
        }
        with self._ledger_lock:  # 台账是共享文件：并发槽串行追加
            self.exp_ledger.append(entry)

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
        # 五要素（task44）：入档同时关联该精英的实验记录（elite_record_path=
        # codes/iter{N}/experiment_record.md）——builder 后续读「精英代码 + 其实验记录」
        record_path = experiment_record.path_for(
            os.path.dirname(os.path.abspath(candidate)))
        admitted = self.archive.admit(cell_id, entry, record_path=record_path)
        if not admitted:
            # 更差解：同格替换条件不满足（事件流无 elite_*——诚实记录）
            self._last_admit_verdict = "reject"
            self._last_reject_reason = "worse_than_elite"
            return
        self._last_admit_verdict = "replace" if had_elite else "admit"
        # 饱和状态：历史最优刷新（方向感知）
        self._note_perf_record(perf_value)

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
            # 事件先于档案（§2.5 原则）：add_pending 在 elite 事件之后——
            # 原实现 add_pending 先于事件写，kill 落在两若之间会产生「档案有 pending、
            # 事件流无记录」→ 重启投影丢失该项（e2e kill 场景 flaky 实测）
            pass

        _exp = getattr(self, "_cur_exp_id", None)
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
            }, "scheduler", experiment_id=_exp)
        if reason:
            self.archive.add_pending(cell_id, candidate, reason)
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
            # 迭代5（env_8 审计 P0）：recent_events 给真值（原实现只给字段名，
            # 「决策须引用事件证据」的纪律无从执行）；条数 12 → 15
            "recent_events": self._recent_events_summary(15),
            # 迭代5：饱和信号 + 历史/预算/领域/台账/自己的决策史
            "saturation": self._or_default(self._saturation_signal(), "（未饱和）"),
            "perf_timeline": self._perf_timeline(10),
            "budget_status": self._budget_status(),
            "domain_brief": self._domain_brief(),
            "ledger_brief": self._ledger_brief(8),
            "own_decisions": self._or_default(self._own_decisions("adjust", 5), "（暂无历史决策）"),
            "interventions": self._recent_interventions(3),
            # 迭代6：三要素门禁命令（可选——判断某方向是否还有潜力时可实证）
            "gate_check_cmd": self._element_cmds()["gate_check_cmd"],
            # 迭代7：检索（role=adjust：创新点/研究趋势；cell 空=全局视野）
            "research_cmd": self._research_cmd("adjust"),
            "research_brief": self._or_default(
                self._research_brief("adjust"),
                "（未提供检索指引——跳过检索）"),
            # ---- 迭代8：四步 loop 的输入（全局实验分析 → SENSE → research → 格管理） ----
            # 区间：上次 adjust（含）→ 本次（全局实验分析的范围，规格 a）
            "experiment_records": self._or_default(
                self._experiment_records(self._last_adjust_iter,
                                         self.iteration_cursor),
                "（区间内尚无实验记录——首轮或 builder 未落盘）"),
            "experiment_sessions": self._experiment_sessions_count(self._last_adjust_iter, self.iteration_cursor),
            "decision_history": self._or_default(
                self._decision_history(3), "（暂无历史决策记录）"),
            "cell_descriptions": self._cell_descriptions(),
            "decisions_path": self._decisions_path(self.iteration_cursor),
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
        }, "scheduler", experiment_id=make_experiment_id("adjust", iteration_id))
        model = (self.conf.get("agent_models") or {}).get("adjust")
        timeout_s = int(self.conf.get("agent_timeout_s", 1800))
        max_turns = self._role_max_turns("adjust")

        def _worker():
            try:
                result = self._call_client("adjust", prompt_path, adjust_dir,
                                           timeout_s, model, max_turns)
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
        # 区间起点推进（下一次 adjust 的全局实验分析从本次起算）
        old_last = self._last_adjust_iter
        self._last_adjust_iter = iteration_id
        # task54 可观测性：何时触发、距上次多少迭代（并发饥饿修复的回归证据）
        self.events.append("adjust_dispatched", {
            "iteration": iteration_id,
            "window_since": iteration_id - old_last,
        }, "scheduler")

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
        with self._ledger_lock:
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
        self.events.append("agent_ended", payload, "scheduler",
                           experiment_id=make_experiment_id("adjust",
                                                            iteration_id))
        # adjust 统计暂存（下一个 _step 头部并入 _iter_agents；加锁防并发槽竞争）
        with self._state_lock:
            self._adjust_stats_carry["adjust"] = {
                "wall_ms": int(pending.wall_ms or 0),
                "tokens": int((pending.usage or {}).get("total_tokens") or 0),
                "tool_calls": len(pending.tool_calls or []),
                "truncated": bool(getattr(pending, "truncated", False)),
            }
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
        # 决策记录（迭代8/KR3）：agent 应写 decisions/iter{N}.md；未写 →
        # 框架按提案合成最小记录（标注 framework_synthesized），保证
        # 「每次决策都有可解释档案」不被 agent 的疏漏破坏
        decision_path = self._ensure_decision_file(
            iteration_id, proposals, bool(pending.valid))
        # adjust_decision 事件（task19）：结构化决策日志——输入（cells_status
        # 摘要）+ 输出（proposals 摘要）同帧落事件流；自改进分析（adjust_trace）
        # 与崩溃恢复均可消费
        self.events.append("adjust_decision", {
            "iteration": iteration_id,
            "agent_valid": bool(pending.valid),
            "proposals": self._proposal_digest(proposals),
            "cells_status_summary": self._cells_status_digest(),
            "decision_path": decision_path,
        }, "scheduler",
            experiment_id=make_experiment_id("adjust", iteration_id))
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
        if ptype == "terminate_recommend":
            # 建议性动作（task23 F 段）：不进裁决自动执行——落提案文件
            # （needs_human=true）+ proposal 事件，供中控会话/人决策收手或换轴
            pid = str(p.get("proposal_id") or
                      "term-%d-%d" % (self.iteration_cursor, index))
            record = {"proposal_id": pid, "type": "terminate_recommend",
                      "target_cell": target, "action": p.get("action"),
                      "evidence": reason,
                      "rationale": str(p.get("rationale") or "")[:400],
                      "rule_id": p.get("rule_id"),
                      "needs_human": True,
                      "proposed_at": _now(),
                      "iteration": self.iteration_cursor}
            os.makedirs(self.proposals_dir, exist_ok=True)
            with open(os.path.join(self.proposals_dir, pid + ".json"), "w",
                      encoding="utf-8") as f:
                json.dump(record, f, ensure_ascii=False, indent=2)
            self.events.append("proposal", {
                "proposal_id": pid, "type": "terminate_recommend",
                "target_cell": target, "action": p.get("action"),
                "evidence": reason, "rule_id": p.get("rule_id"),
                "needs_human": True,
            }, "agent:adjust")
            return
        if ptype == "new_cell":
            # task56 F 段：method 必填（空 method → 拒绝，不建没有方向描述的格）
            method = str(p.get("method") or "").strip()
            if not method:
                self.events.append("new_cell_rejected", {
                    "reason": "empty_method",
                    "detail": "new_cell 必须带 method（方法描述）——"
                              "没有方向描述的格无法被 builder 理解",
                    "descriptor": p.get("descriptor"),
                }, "agent:adjust")
                return
            # 新格（迭代8/KR2+KR3）：descriptor → feature_space 热重载加值
            # （复用 expand_cell 的 add_cell 链）+ 格描述落档 + cell_created 事件
            descriptor = p.get("descriptor")
            if not isinstance(descriptor, dict) or not descriptor:
                return
            applied = self._apply_expand_cell(
                {"action": "add_cell", "descriptor": descriptor})
            if applied:
                # 立即热重载：让新轴值进入 fs 取值域，cell_id 才能解析
                self._maybe_reload_feature_space()
            try:
                cell_id = self.fs.cell_id(descriptor)
            except ValueError:
                return  # 脏描述子（未知轴/缺轴）——不落半成品
            description = {}
            for key in ("name", "method"):
                val = p.get(key)
                if isinstance(val, str) and val.strip():
                    description[key] = val.strip()
            refs = p.get("source_refs")
            if isinstance(refs, (list, tuple)) and refs:
                description["source_refs"] = [str(r) for r in refs]
            if description:
                self.archive.set_description(cell_id, description)
            self.events.append("cell_created", {
                "cell_id": cell_id, "descriptor": descriptor,
                "name": (description or {}).get("name"),
                "method": str((description or {}).get("method") or "")[:400],
                "source_refs": (description or {}).get("source_refs") or [],
                "orthogonality_rationale": str(
                    p.get("orthogonality_rationale") or "")[:400],
                "applied": bool(applied),
                "rule_id": p.get("rule_id"),
            }, "agent:adjust")
            self.archive.save()
            return
        if ptype == "modify_cell":
            # 改进格描述（迭代8/KR2）：只改 description（method/source_refs），
            # 不改格定义本身（改定义仍是 hard_modify_cell 提案制）
            if not target:
                return
            current = self._cell_entry(target).get("description")
            merged = dict(current) if isinstance(current, dict) else {}
            method = p.get("method")
            if isinstance(method, str) and method.strip():
                merged["method"] = method.strip()
            refs = p.get("source_refs")
            if isinstance(refs, (list, tuple)) and refs:
                merged["source_refs"] = [str(r) for r in refs]
            name = p.get("name")
            if isinstance(name, str) and name.strip():
                merged["name"] = name.strip()
            if not merged:
                return
            self.archive.set_description(target, merged)
            self.events.append("cell_modified", {
                "cell_id": target,
                "method": str(merged.get("method") or "")[:400],
                "source_refs": merged.get("source_refs") or [],
                "rationale": str(p.get("rationale") or "")[:300],
                "rule_id": p.get("rule_id"),
            }, "agent:adjust")
            self.archive.save()
            return
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
        """格子现状（adjust 决策依据）。红黑树 env_11 实测：完整 JSON 拼进 ctx
        导致 prompt 88KB → input 膨胀 → 截断失败。压缩为精简元信息：
        cell_id + description.name + weight + visit + score。需要详细
        descriptor/elite 时 agent 自己 read archive.json（它的决策）。"""
        out = []
        for cid in self.fs.all_cells():
            e = self._cell_entry(cid)
            desc = e.get("description") or {}
            elite = e.get("elite") or {}
            out.append({"cell_id": cid,
                        "name": desc.get("name") or cid.split("|")[0].split("=")[-1],
                        "visit": e.get("visit", 0),
                        "weight": e.get("weight", 1.0),
                        "perf": elite.get("perf_value") if elite else None})
        return out

    def _recent_events_summary(self, n: int = 15) -> List[Dict[str, Any]]:
        """近期事件摘要——**真值**（task23 A 段）。

        env_8 实测：原实现只给 `payload_keys`（字段名列表、零值），adjust 的
        「每次权重变动必须引用 ≥1 条具体事件证据」纪律无从执行（它看不到
        failure_stage / perf_value / self_check 结果）。本版按事件类型提取
        归因必需字段；heartbeat 过滤（运维噪音，占事件流 70%+）。
        """
        items = []
        for e in self.events.read_all():
            typ = e.get("type")
            p = e.get("payload") or {}
            if typ == "heartbeat":
                continue
            item = {"type": typ, "seq": e.get("seq")}  # type: Dict[str, Any]
            if typ == "gate_result":
                metrics = p.get("metrics") or {}
                perf = metrics.get("perf") or {}
                soft = metrics.get("soft") or {}
                item.update({
                    "iteration_id": p.get("iteration_id"),
                    "cell_id": p.get("cell_id"),
                    "passed": p.get("passed"),
                    "stage": p.get("stage"),
                    "failure_stage": p.get("failure_stage"),
                    "perf_value": perf.get("value"),
                    "soft_verdict": soft.get("verdict"),
                })
            elif typ == "agent_ended":
                usage = p.get("usage") or {}
                item.update({
                    "iteration_id": p.get("iteration_id"),
                    "role": p.get("role"),
                    "wall_s": round(float(p.get("wall_ms") or 0) / 1000.0),
                    "total_tokens": usage.get("total_tokens"),
                    "truncated": p.get("truncated"),
                    "valid": p.get("valid"),
                })
            elif typ == "cell_sampled":
                item.update({
                    "iteration_id": p.get("iteration_id"),
                    "cell_id": p.get("cell_id"),
                    "weight_share": p.get("weight_share"),
                    "cold_start": p.get("cold_start"),
                    "visit": p.get("visit"),
                })
            elif typ in ("elite_admitted", "elite_replaced"):
                item.update({
                    "iteration_id": p.get("iteration_id"),
                    "cell_id": p.get("cell_id"),
                    "perf_value": p.get("perf_value"),
                    "gen": p.get("gen"),
                    "score": p.get("score"),
                })
            elif typ == "weight_changed":
                item.update({
                    "cell_id": p.get("cell_id"),
                    "old_weight": p.get("old_weight"),
                    "new_weight": p.get("new_weight"),
                    "rule_id": p.get("rule_id"),
                })
            elif typ == "candidate_produced":
                sc = p.get("self_check") or {}
                item.update({
                    "iteration_id": p.get("iteration_id"),
                    "cell_id": p.get("cell_id"),
                    "role": p.get("role"),
                    "self_check_passed": sc.get("passed"),
                    "diff_path": p.get("diff_path"),
                })
            elif typ == "adjust_decision":
                props = p.get("proposals") or []
                item.update({
                    "iteration": p.get("iteration"),
                    "proposals": props[:4],
                })
            elif typ == "intervention":
                item.update({"raw": str(p.get("text") or "")[:200],
                             "category": p.get("category")})
            elif typ in ("resurrection", "invalidation", "env_failure_concluded"):
                item.update({
                    "cell_id": p.get("cell_id"),
                    "reason": str(p.get("reason") or p.get("cause") or "")[:120],
                })
            elif typ in ("iteration_started", "iteration_ended",
                         "session_started", "session_ended"):
                item.update({"iteration_id": p.get("iteration_id")})
            elif typ in ("proposal", "adjudicated"):
                item.update({
                    "proposal_id": p.get("proposal_id"),
                    "ptype": p.get("type"),
                    "decision": p.get("decision"),
                })
            else:
                item.update({"payload": {k: p.get(k) for k in list(p)[:4]}})
            items.append(item)
        return items[-n:]

    # ---- ctx 组装辅助（task23 B 段：四角色共用的剩余盲点修复） ----

    def _perf_timeline(self, n: int = 10) -> List[Dict[str, Any]]:
        """性能时间线（近 n 条带 perf 的台账条目 + is_record）。"""
        return self.exp_ledger.perf_timeline(n)

    def _budget_status(self) -> Dict[str, Any]:
        """预算余额（迭代/墙钟/token 三轨——agent 不知道剩余预算就无法权衡）。"""
        try:
            wall_min = round((time.time() - self.ledger.wall_start) / 60.0, 1)
        except (AttributeError, TypeError):
            wall_min = 0.0
        tokens_used = int(self.ledger.total_tokens() or 0)
        max_tokens = int(self.ledger.max_total_tokens or 0)
        return {
            "iterations_used": int(self.iteration_cursor),
            "max_iterations": int(self.conf.get("max_iterations") or 0),
            "wall_minutes": wall_min,
            "max_wall_h": float(self.ledger.max_wall_time_h or 0),
            "tokens_used": tokens_used,
            "max_tokens": max_tokens,
            "tokens_pct": (round(100.0 * tokens_used / max_tokens, 1)
                           if max_tokens > 0 else None),
        }

    def _gate_defs(self) -> Dict[str, Any]:
        """env 的门禁定义关键值（coding 需要知道「改善多少才算赢」）。"""
        meta = self._load_env_meta()
        tol = meta.get("tolerance") if isinstance(meta.get("tolerance"), dict) else {}
        bench = meta.get("bench") if isinstance(meta.get("bench"), dict) else {}
        gates = meta.get("gates") if isinstance(meta.get("gates"), dict) else {}
        return {
            "tolerance": tol,
            "bench": {k: bench.get(k) for k in ("size", "k", "timeout_s")
                      if k in bench},
            "gates": {k: gates.get(k) for k in
                      ("score_max", "runtime_ms_min", "ref_runtime_band_ms")
                      if k in gates},
        }

    def _domain_brief(self, limit: int = 2000) -> str:
        """领域知识摘要（priority 行优先：SOP/死族/天花板/容差）。

        env_8 实测：agent 每轮都 read 整个 domain_knowledge.md（吃回合预算、
        且内容随 env 升级漂移）；摘要直接进 ctx 省回合。
        """
        path = os.path.join(self.env_dir, "domain_knowledge.md")
        try:
            with open(path, "r", encoding="utf-8") as f:
                text = f.read()
        except OSError:
            return ""
        if limit <= 0 or len(text) <= limit:
            return text
        keys = ("SOP", "死族", "天花板", "score 阶梯", "容差", "score_max",
                "BLAS", "无法", "上限")
        lines = text.splitlines()
        picked = []  # type: List[str]
        for i, ln in enumerate(lines):
            if any(k in ln for k in keys):
                picked.extend(lines[max(0, i - 2):i + 6])
        out = "\n".join(picked).strip()
        if len(out) < 200:
            return text[:limit]
        return out[:limit]

    def _python_cmd(self) -> str:
        """env venv python 优先，回退 python3。"""
        venv = os.path.join(self.env_dir, ".venv", "bin", "python")
        return venv if os.path.exists(venv) else "python3"

    def _self_check_and_profile_cmds(self) -> Dict[str, str]:
        """返回 (self_check_cmd, perf_profile_cmd)；perf_profile 缺失 → 空串。"""
        self_check_cmd = self._self_check_cmd()
        py = self._python_cmd()
        prof = os.path.join(self.env_dir, "skills", "perf_profile.py")
        profile_cmd = ("%s %s --candidate <path>" % (py, prof)
                       if os.path.exists(prof) else "")
        return {"self_check_cmd": self_check_cmd,
                "perf_profile_cmd": profile_cmd}

    # ------------------------------------------------------------------ #
    # 三要素（迭代6）：profile / benchmark / verify / gate_check
    # ------------------------------------------------------------------ #

    def _element_cmds(self) -> Dict[str, str]:
        """三要素命令（迭代6 契约）。

        profile=事实获取 / benchmark=评估方法 / verify=约束 / gate_check=三者封装
        （框架与 agent 共用同一套资产）。每项 = `<python> <env>/skills/<n>.py`
        （文件存在才给）；缺失 → 空串（legacy env 无 skills 四件，模板侧
        「未提供则跳过」兜底）。gate_check 附 --baseline 用法（agent 可自判
        软门禁时需要该格精英 perf）。
        """
        py = self._python_cmd()
        skills = os.path.join(self.env_dir, "skills")
        out = {}  # type: Dict[str, str]
        for name in ("profile", "benchmark", "verify"):
            script = os.path.join(skills, name + ".py")
            out[name + "_cmd"] = ("%s %s --candidate <path>" % (py, script)
                                  if os.path.exists(script) else "")
        gc = os.path.join(skills, "gate_check.py")
        out["gate_check_cmd"] = (
            "%s %s --candidate <path> [--baseline <该格精英 perf，可选>]"
            % (py, gc) if os.path.exists(gc) else "")
        return out

    # ------------------------------------------------------------------ #
    # 检索（迭代7）：research skill —— 三要素（方法/信源/关注点）注入
    # ------------------------------------------------------------------ #

    def _research_cmd(self, role: str,
                      cell_id: Optional[str] = None) -> str:
        """research 命令（第四类 env 资产）：供 agent 换角色/换格再查。

        文件缺失（legacy env 无 skills/research.py）→ 空串，模板侧
        「未提供则跳过检索」兜底。
        """
        script = os.path.join(self.env_dir, "skills", "research.py")
        if not os.path.exists(script):
            return ""
        cmd = "%s %s --role %s" % (self._python_cmd(), script, role)
        if cell_id:
            cmd += ' --cell "%s"' % cell_id
        return cmd

    def _research_brief(self, role: str,
                        cell_id: Optional[str] = None) -> Optional[Dict[str, Any]]:
        """预生成检索指引（框架代调 research.py，省 agent 回合）。

        返回 {role, cell, method, sources, focus, suggested_queries,
        discipline}；脚本缺失/超时/非 JSON/带 error → None（静默，不阻断
        迭代——检索是认知辅助，不是门禁）。
        """
        script = os.path.join(self.env_dir, "skills", "research.py")
        if not os.path.isfile(script):
            return None
        argv = [self._python_cmd(), script, "--role", role]
        if cell_id:
            argv += ["--cell", cell_id]
        try:
            r = subprocess.run(argv, capture_output=True, text=True,
                               timeout=30)
        except (OSError, subprocess.SubprocessError):
            return None
        parsed = _parse_last_json_line(_coerce_text(r.stdout))
        if not isinstance(parsed, dict) or parsed.get("error"):
            return None
        return parsed

    def _profile_candidate(self, cell_id: str) -> Optional[str]:
        """profile 采集对象：该格精英代码 → naive_start.py → reference.py。"""
        elite = self._cell_entry(cell_id).get("elite") or {}
        cp = elite.get("code_path")
        if isinstance(cp, str) and cp and os.path.isfile(cp):
            return cp
        for name in ("naive_start.py", "reference.py"):
            p = os.path.join(self.env_dir, "code", name)
            if os.path.isfile(p):
                return p
        return None

    def _collect_profile(self, cell_id: str, cursor: int) -> Optional[Dict[str, Any]]:
        """每迭代自动采集 profile（三要素之「现状」）→ 时间序列归档。

        对父本候选（该格精英 / naive_start / reference）跑 profile 脚本，
        存 `<ws>/profile_history/<cell>/iter{N}.json`：
            {ts, iteration_id, cell_id, candidate, profile}
        失败静默（profile 是辅助事实，不阻断迭代）。
        """
        script = os.path.join(self.env_dir, "skills", "profile.py")
        if not os.path.isfile(script):
            return None
        candidate = self._profile_candidate(cell_id)
        if not candidate:
            return None
        try:
            r = subprocess.run(
                [self._python_cmd(), script, "--candidate", candidate],
                capture_output=True, text=True, timeout=120)
        except (OSError, subprocess.SubprocessError):
            return None
        parsed = _parse_last_json_line(_coerce_text(r.stdout))
        if parsed is None:
            return None
        payload = {"ts": _now(), "iteration_id": cursor, "cell_id": cell_id,
                   "candidate": candidate, "profile": parsed}
        d = os.path.join(self.profile_history_dir, _safe_fs_name(cell_id))
        try:
            os.makedirs(d, exist_ok=True)
            with open(os.path.join(d, "iter%d.json" % cursor), "w",
                      encoding="utf-8") as f:
                json.dump(payload, f, ensure_ascii=False, indent=2)
        except OSError:
            pass  # 归档失败不影响迭代（事实已进 ctx 供本次使用）
        return parsed

    def _profile_history(self, cell_id: str, n: int = 3) -> List[Dict[str, Any]]:
        """某格最近 n 条 profile 历史（iter 编号降序）。"""
        d = os.path.join(self.profile_history_dir, _safe_fs_name(cell_id))
        try:
            names = [x for x in os.listdir(d)
                     if x.startswith("iter") and x.endswith(".json")]
        except OSError:
            return []

        def _key(name: str) -> int:
            m = re.search(r"iter(\d+)\.json$", name)
            return int(m.group(1)) if m else -1

        out = []  # type: List[Dict[str, Any]]
        for name in sorted(names, key=_key, reverse=True)[:max(1, n)]:
            try:
                with open(os.path.join(d, name), "r", encoding="utf-8") as f:
                    out.append(json.load(f))
            except (OSError, ValueError):
                continue
        return out

    def _cell_of_code_path(self, code_path: Optional[str]) -> Optional[str]:
        """code_path 属于哪个格（扫 archive 快照的精英引用）。"""
        if not isinstance(code_path, str) or not code_path:
            return None
        for cid, cell in _cells_snapshot(self.archive).items():
            elite = (cell or {}).get("elite") or {}
            if elite.get("code_path") == code_path:
                return cid
        return None

    def _parent_profile_history(self, cell_id: str) -> List[Dict[str, Any]]:
        """父格 profile 历史（父本跨格时；父本即本格 → 空）。"""
        elite = self._cell_entry(cell_id).get("elite") or {}
        owner = self._cell_of_code_path(elite.get("code_path"))
        if not owner or owner == cell_id:
            return []
        return [{"cell_id": owner, "entries": self._profile_history(owner, 2)}]

    def _inspiration_profile_history(self, cell_id: str,
                                     n_cells: int = 3,
                                     per: int = 2) -> List[Dict[str, Any]]:
        """灵感格（近邻）各自最近 per 条 profile 历史。"""
        out = []  # type: List[Dict[str, Any]]
        for nb in self._neighbor_elites(cell_id, limit=n_cells):
            cid = nb.get("cell_id")
            if not isinstance(cid, str):
                continue
            entries = self._profile_history(cid, per)
            if entries:
                out.append({"cell_id": cid, "entries": entries})
        return out

    def _write_design(self, cell_id: str, cursor: int, result: Any,
                      plan: Dict[str, Any],
                      profile: Optional[Dict[str, Any]]) -> None:
        """builder 决策落盘（designs/iter{N}_{cell}.json）。写失败不影响循环。"""
        data = getattr(result, "data", None)
        data = data if isinstance(data, dict) else {}
        papers = data.get("papers")
        inspirations = data.get("inspirations")
        reasoning = data.get("reasoning")
        design = {
            "iteration_id": cursor,
            "cell_id": cell_id,
            "ts": _now(),
            "profile_facts": _profile_facts(profile),
            "papers": papers if isinstance(papers, list) else [],
            "inspirations": inspirations if isinstance(inspirations, list) else [],
            "reasoning": (reasoning if isinstance(reasoning, str)
                          else (getattr(result, "summary", "") or "")),
            "experiment_plan": plan,
        }
        try:
            os.makedirs(self.designs_dir, exist_ok=True)
            path = os.path.join(
                self.designs_dir,
                "iter%d_%s.json" % (cursor, _safe_fs_name(cell_id)))
            with open(path, "w", encoding="utf-8") as f:
                json.dump(design, f, ensure_ascii=False, indent=2)
        except OSError:
            pass  # 决策档案是分析副产品；失败不回滚迭代

    def _recent_designs(self, n: int = 3) -> List[Dict[str, Any]]:
        """近期 builder 决策摘要（避免重复方向；gen_env 轨迹分析也读 designs/）。"""
        try:
            names = [x for x in os.listdir(self.designs_dir)
                     if x.endswith(".json")]
        except OSError:
            return []

        def _key(name: str) -> int:
            m = re.match(r"iter(\d+)_", name)
            return int(m.group(1)) if m else -1

        out = []  # type: List[Dict[str, Any]]
        for name in sorted(names, key=_key, reverse=True)[:max(1, n)]:
            try:
                with open(os.path.join(self.designs_dir, name), "r",
                          encoding="utf-8") as f:
                    d = json.load(f)
            except (OSError, ValueError):
                continue
            plan = d.get("experiment_plan") if isinstance(
                d.get("experiment_plan"), dict) else {}
            out.append({
                "iteration_id": d.get("iteration_id"),
                "cell_id": d.get("cell_id"),
                "idea": plan.get("idea"),
                "papers": [p.get("arxiv_id") or p.get("title")
                           for p in (d.get("papers") or [])
                           if isinstance(p, dict)],
                "reasoning": str(d.get("reasoning") or "")[:200],
            })
        return out

    def _workdir_files(self, iter_dir: str) -> List[str]:
        """迭代工作区文件清单（父本/solution——coding 不用先 ls 一遍）。"""
        try:
            names = sorted(os.listdir(iter_dir))
        except OSError:
            return []
        return [n for n in names if not n.startswith(".")]

    def _recent_interventions(self, n: int = 3) -> List[Dict[str, Any]]:
        """human 干预（原文）——人能引导 agent 的最小通道。"""
        out = []
        for e in self.events.read_all():
            if e.get("type") != "intervention":
                continue
            p = e.get("payload") or {}
            out.append({"seq": e.get("seq"),
                        "text": p.get("text"),
                        "category": p.get("category")})
        return out[-n:]

    def _ledger_brief(self, n: int = 8) -> List[Dict[str, Any]]:
        """实验台账摘要（跨迭代记忆：避免重复已失败方向）。"""
        return self.exp_ledger.brief(n)

    def _own_decisions(self, role: str = "adjust", n: int = 5) -> List[Dict[str, Any]]:
        """某角色历史参与条目的摘要（adjust 回看自己的决策与效果）。"""
        return self.exp_ledger.own_decisions(role, n)

    def _perf_baseline(self, cell_id: str) -> Dict[str, Any]:
        """该格当前精英 perf 基线（coding 的改善判据参照）。"""
        elite = self._cell_entry(cell_id).get("elite") or {}
        pv = elite.get("perf_value")
        return {"cell_id": cell_id,
                "elite_perf": pv if isinstance(pv, (int, float)) else None,
                "perf_type": self.perf_type,
                "perf_desc": self.perf_desc}

    # ------------------------------------------------------------------ #
    # 迭代8：实验记录 / adjust 四步 / debug 三触发 辅助
    # ------------------------------------------------------------------ #

    @staticmethod
    def _mtime_ns(path: str) -> Optional[int]:
        try:
            return os.stat(path).st_mtime_ns
        except OSError:
            return None

    @staticmethod
    def _read_text(path: str, limit: int = 0) -> str:
        """读文本文件（缺文件 → ""；limit>0 时截断）。"""
        try:
            with open(path, "r", encoding="utf-8") as f:
                text = f.read()
        except OSError:
            return ""
        if limit and limit > 0 and len(text) > limit:
            return text[:limit]
        return text

    def _record_for_code_path(self, code_path: Any,
                              limit: int = 1200) -> str:
        """精英解所在迭代目录的实验记录摘要（builder/coding 的「必读上下文」）。

        limit：摘要字符上限。目标格精英记录用 1200（完整上下文）；**相邻格
        只作灵感源，用 400（降本，迭代10/task51：相邻格×2 的上下文是 builder
        成本的大头之一）**。
        """
        if not isinstance(code_path, str) or not code_path:
            return ""
        iter_dir = os.path.dirname(os.path.abspath(code_path))
        return experiment_record.digest(iter_dir, limit=limit)

    def _neighbor_samples(self, cell_id: str, k: int = 2) -> List[Dict[str, Any]]:
        """两个随机相邻格样本（J-i8-3）——**摘要形态**（迭代10/task51 降本）。

        每项只给：cell_id / description / elite_code_path（**路径引用，不内联
        代码正文**）/ elite_perf / record_digest（≤400 字）。需要看代码或完整
        记录时由 agent 用 read 自取——避免把相邻格全文塞进 builder 上下文。
        """
        out = []  # type: List[Dict[str, Any]]
        for cid in self.fs.sample_neighbors(cell_id, k):
            entry = self._cell_entry(cid)
            elite = entry.get("elite") or {}
            desc = entry.get("description")
            if not isinstance(desc, dict):
                desc = {}
            out.append({
                "cell_id": cid,
                "description": desc,
                "elite_code_path": elite.get("code_path"),
                "elite_perf": elite.get("perf_value"),
                "record_digest": self._or_default(
                    self._record_for_code_path(elite.get("code_path"),
                                               limit=_NEIGHBOR_RECORD_LIMIT),
                    "（该格无精英或记录缺失）"),
            })
        return out

    # ---- adjust 四步上下文（KR3） ----

    def _experiment_records(self, since_iter: int,
                            until_iter: int) -> List[Dict[str, Any]]:
        """区间内全部实验记录摘要（上次 adjust → 本次）。"""
        since = max(0, int(since_iter or 0))
        until = max(since, int(until_iter or 0))
        return experiment_record.recent(self.ws, since, until)

    def _experiment_sessions_count(self, since: int, until: int) -> str:
        """区间 agent session 可用性提示（不预列完整路径——ctx 降本）。"""
        n = until - max(0, since)
        if n <= 0:
            return "（本区间无新实验 session）"
        return "（%d 次实验的 pi session 可用——需要补细节时 read codes/iter{N}/ 下对应 prompt 的 workdir）" % n

    def _experiment_sessions(self, since_iter: int,
                             until_iter: int) -> List[str]:
        """区间内 agent 的 pi session 目录（best-effort：按编码后的 cwd 名匹配）。

        记录不足时 adjust 可按需 read 这些目录下的 jsonl 补细节（规格 a）。
        找不到（路径不同/未启用 session）→ []，不阻断。
        """
        roots = []  # type: List[str]
        agent_dir = os.environ.get("RICK_PI_AGENT_DIR")
        if agent_dir:
            roots.append(os.path.join(agent_dir, "sessions"))
        roots.append(os.path.join(os.path.expanduser("~"), ".rick", "pi",
                                  "agent", "sessions"))
        found = []  # type: List[str]
        seen = set()
        for root in roots:
            try:
                names = os.listdir(root)
            except OSError:
                continue
            for name in names:
                m = re.search(r"codes-iter(\d+)-", name)
                if not m:
                    continue
                n = int(m.group(1))
                if n < since_iter or n > until_iter:
                    continue
                path = os.path.join(root, name)
                if path in seen:
                    continue
                seen.add(path)
                found.append(path)
        return sorted(found)

    def _decisions_path(self, iteration_id: int) -> str:
        return os.path.join(self.decisions_dir, "iter%d.md" % iteration_id)

    def _decision_history(self, n: int = 3) -> List[Dict[str, Any]]:
        """近期 adjust 决策记录摘要（防震荡/防重复处方）。"""
        try:
            names = [x for x in os.listdir(self.decisions_dir)
                     if x.endswith(".md")]
        except OSError:
            return []

        def _key(name: str) -> int:
            m = re.match(r"iter(\d+)\.md$", name)
            return int(m.group(1)) if m else -1

        out = []  # type: List[Dict[str, Any]]
        for name in sorted(names, key=_key, reverse=True)[:max(1, n)]:
            path = os.path.join(self.decisions_dir, name)
            text = self._read_text(path, 800)
            out.append({"path": path, "digest": text})
        return out

    def _cell_descriptions(self) -> Dict[str, Any]:
        """各格 description（精简版：name + method 前 200 字 + source_refs 数量）。

        红黑树 env_11 实测：完整 description JSON 使 prompt 达 88KB → input
        膨胀 → 截断。精简到 200 字后仍能支撑「改进 method」的判断（方向在，
        细节靠 archive.json 自查）。"""
        out = {}  # type: Dict[str, Any]
        for cid in self.fs.all_cells():
            desc = self._cell_entry(cid).get("description")
            if not isinstance(desc, dict):
                out[cid] = {}
                continue
            method = str(desc.get("method") or "")
            out[cid] = {
                "name": desc.get("name") or "",
                "method_brief": method[:200] + ("…" if len(method) > 200 else ""),
                "source_refs": desc.get("source_refs") or [],
            }
        return out

    def _ensure_decision_file(self, iteration_id: int,
                              proposals: List[Any], valid: bool) -> str:
        """决策记录保底（KR3）：agent 未落盘 → 框架按提案合成最小记录。

        返回 decisions 文件路径（存在即返回；不存在则合成后返回）。
        """
        path = self._decisions_path(iteration_id)
        if os.path.isfile(path):
            return path
        lines = ["# adjust 决策记录 · 迭代 %d" % iteration_id, "",
                 "> framework_synthesized: true"
                 "（agent 未落盘决策记录，框架按提案合成）", "",
                 "- agent_valid: %s" % bool(valid), "", "## 动作"]
        if not proposals:
            lines.append("- （无提案）")
        for p in proposals:
            if not isinstance(p, dict):
                continue
            lines.append("- **%s** %s：%s"
                         % (p.get("type"),
                            p.get("target_cell") or p.get("descriptor") or "",
                            str(p.get("action") or "")[:200]))
            ev = p.get("evidence") or p.get("rationale")
            if ev:
                lines.append("  - 依据：%s" % str(ev)[:300])
        try:
            os.makedirs(self.decisions_dir, exist_ok=True)
            with open(path, "w", encoding="utf-8") as f:
                f.write("\n".join(lines) + "\n")
        except OSError:
            return path
        return path

    # ---- debug 三触发（KR6） ----

    def _debug_policy(self) -> Dict[str, Any]:
        """debug 复验策略：env.json debug_policy 覆盖 conf（gen_env 追问 human）。"""
        meta = self._load_env_meta()
        dp = meta.get("debug_policy") if isinstance(
            meta.get("debug_policy"), dict) else {}
        gain = dp.get("gain_threshold_pct",
                      self.conf.get("debug_gain_threshold_pct", 20))
        try:
            gain = float(gain)
        except (TypeError, ValueError):
            gain = 20.0
        recheck = dp.get("recheck_tolerance_band",
                         self.conf.get("debug_recheck_tolerance_band", True))
        return {"gain_threshold_pct": gain,
                "recheck_tolerance_band": bool(recheck)}

    def _perf_gain_pct(self, gr: Any, cell_id: str) -> Optional[float]:
        """本候选相对基线（该格精英 / 全局最优）的收益百分比（方向感知）。"""
        perf = (getattr(gr, "metrics", None) or {}).get("perf") or {}
        pv = perf.get("value")
        if not isinstance(pv, (int, float)) or isinstance(pv, bool):
            return None
        ref = None  # type: Optional[float]
        elite = self._cell_entry(cell_id).get("elite") or {}
        ep = elite.get("perf_value")
        if isinstance(ep, (int, float)) and not isinstance(ep, bool):
            ref = float(ep)
        elif isinstance(getattr(self, "_best_perf_before_iter", None),
                        (int, float)):
            ref = float(self._best_perf_before_iter)
        elif isinstance(self._best_perf, (int, float)):
            ref = float(self._best_perf)
        if ref is None or ref == 0:
            return None
        pv = float(pv)
        if self.perf_desc == "high":
            return round((pv - ref) / abs(ref) * 100.0, 3)
        return round((ref - pv) / abs(ref) * 100.0, 3)

    def _recheck_reason(self, gr: Any, cell_id: str) -> Optional[str]:
        """复验原因（None=不复验）：suspicious_gain / tolerance_band_recheck。"""
        policy = self._debug_policy()
        gain = self._perf_gain_pct(gr, cell_id)
        thr = float(policy["gain_threshold_pct"])
        if isinstance(gain, (int, float)) and thr > 0 and gain >= thr:
            return "suspicious_gain"
        if bool(getattr(gr, "tolerance_band", False)) and \
                policy["recheck_tolerance_band"]:
            return "tolerance_band_recheck"
        return None

    def _run_recheck_debug(self, plan: Dict[str, Any], cell_id: str,
                           cursor: int, iter_dir: str, gr: Any,
                           reason: str) -> Any:
        """异常收益/容差带复验（attempt=200）：返回 (fixed_path|None, verdict)。

        verdict ∈ {keep_original, fixed}（agent 契约）；缺省按 keep_original
        处理——复验的目的是防错杀，不是制造新失败。
        """
        gain = self._perf_gain_pct(gr, cell_id)
        failure = {
            "failure_stage": None,
            "metrics": getattr(gr, "metrics", None) or {},
            "details": "recheck(%s): perf_gain_pct=%s" % (reason, gain),
        }
        self_check = {"passed": True, "stage": "recheck",
                      "metrics": getattr(gr, "metrics", None) or {}}
        result = self._run_debug(
            plan, cell_id, cursor, iter_dir, attempt=200, max_attempts=1,
            self_check=self_check,
            last_err=json.dumps(failure, ensure_ascii=False),
            gate_failure=failure, perf_gain_pct=gain,
            recheck_reason=reason)
        data = getattr(result, "data", None)
        data = data if isinstance(data, dict) else {}
        fix_report = data.get("fix_report") if isinstance(
            data.get("fix_report"), dict) else {}
        verdict = str(data.get("verdict") or fix_report.get("verdict")
                      or "keep_original")
        fixed = None  # type: Optional[str]
        if verdict == "fixed":
            fixed = self._candidate_path(result, iter_dir)
        self.events.append("candidate_produced", {
            "iteration_id": cursor, "cell_id": cell_id,
            "role": "debug", "attempt": 200,
            "code_path": fixed or "",
            "recheck_reason": reason,
            "verdict": verdict,
            "fix_report": fix_report or
            {"attempts": 200, "root_cause": None, "changes": []},
            "self_check": {"passed": verdict != "failed",
                           "stage": "recheck", "metrics": {}},
        }, "agent:debug")
        return fixed, verdict

    def _emit_gate_result(self, gr: Any, cursor: int, cell_id: str,
                          recheck: bool = False) -> None:
        """gate_result 事件（复验时带 recheck 标记）。"""
        payload = {
            "iteration_id": cursor, "cell_id": cell_id,
            "stage": getattr(gr, "failure_stage", None) or "passed",
            "failure_stage": getattr(gr, "failure_stage", None),
            "passed": bool(getattr(gr, "passed", False)),
            "metrics": getattr(gr, "metrics", None) or {},
            "details": (getattr(gr, "details", "") or "")[-400:],
        }
        if recheck:
            payload["recheck"] = True
        self.events.append("gate_result", payload, "gate",
                           experiment_id=getattr(self, "_cur_exp_id", None))

    # ---- 饱和信号（task23 E 段） ----

    def _recover_saturation_state(self) -> None:
        """从事件流恢复历史最优 perf（崩溃重启后 stale 计数不从头来）。"""
        best = None  # type: Optional[float]
        try:
            events = self.events.read_all()
        except (AttributeError, OSError):
            return
        for ev in events:
            if ev.get("type") not in ("elite_admitted", "elite_replaced"):
                continue
            pv = (ev.get("payload") or {}).get("perf_value")
            if not isinstance(pv, (int, float)):
                continue
            v = float(pv)
            if best is None:
                best = v
            else:
                best = max(best, v) if self.perf_desc == "high" else min(best, v)
        self._best_perf = best

    def _champion_cell(self) -> Optional[str]:
        """当前全局最优格（按 perf 方向）。"""
        snap = _cells_snapshot(self.archive)
        low = (self.perf_desc != "high")
        best_cid = None  # type: Optional[str]
        best = None  # type: Optional[float]
        for cid in self.fs.all_cells():
            elite = (snap.get(cid) or {}).get("elite") or {}
            pv = elite.get("perf_value")
            if not isinstance(pv, (int, float)):
                continue
            v = float(pv)
            if best is None or (v < best if low else v > best):
                best, best_cid = v, cid
        return best_cid

    @staticmethod
    def _or_default(value, default):
        """ctx 值的可读兜底：None/空 → default 文案。

        render 的宽松语义会保留未替换的 {{key}}（agent 看到裸占位符会困惑），
        因此在组装 ctx 时就把 None/空值转成人读文案（env_8/迭代5 冒烟实测）。
        """
        if value is None:
            return default
        if isinstance(value, (list, dict)) and not value:
            return default
        return value

    def _saturation_signal(self) -> Optional[Dict[str, Any]]:
        """数据饱和信号（env_8 实测：18 轮无新纪录仍只调权重、0 次加格）。

        三条件全满足才返回非空：
          - stale_iterations >= 3（自上次全局最优刷新）
          - all_visited：所有 weight>0 的格 visit >= 2
          - champion_misses >= 3（冠军格近 3 次采样未刷新）
        """
        try:
            snap = _cells_snapshot(self.archive)
            cells = list(self.fs.all_cells())
        except AttributeError:
            return None
        alive = [c for c in cells
                 if float((snap.get(c) or {}).get("weight", 1.0) or 0) > 0]
        if not alive:
            return None
        all_visited = all(int((snap.get(c) or {}).get("visit") or 0) >= 2
                          for c in alive)
        if not (self._stale_iters >= 3 and all_visited
                and self._champion_misses >= 3):
            return None
        return {
            "stale_iterations": int(self._stale_iters),
            "all_visited": True,
            "champion_cell": self._champion_cell(),
            "best_perf": self._best_perf,
            "alive_cells": len(alive),
        }

    def _note_perf_record(self, perf_value: Any) -> None:
        """入格时维护历史最优/静止计数（方向感知）。"""
        if not isinstance(perf_value, (int, float)):
            return
        v = float(perf_value)
        with self._state_lock:
            improved = (self._best_perf is None or
                        (v > self._best_perf if self.perf_desc == "high"
                         else v < self._best_perf))
            if improved:
                self._best_perf = v
                self._record_this_iter = True
                self._champion_misses = 0


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

        task56 E 段：先检查格子总量软约束（> max_cells=100 → 拒绝 +
        cell_limit_reached 事件）——红黑树 env_11 实测 135 格使 adjust
        prompt 膨胀到 88KB。
        """
        max_cells = int(self.conf.get("max_cells", 100) or 0)
        if max_cells > 0 and len(self.fs.all_cells()) >= max_cells:
            self.events.append("cell_limit_reached", {
                "current_cells": len(self.fs.all_cells()),
                "max_cells": max_cells,
                "action": proposal.get("action"),
                "reason": "格子总数已达上限——红黑树 env_11 实测 135 格使 "
                          "adjust prompt 膨胀。建议先灭绝无效格（weight=0）"
                          "或收敛轴。",
            }, "scheduler")
            return None

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

    def _role_max_turns(self, role: str) -> Optional[int]:
        """per-role 回合预算（<role>_max_turns；0/缺失 = 不限）。"""
        raw = self.conf.get("%s_max_turns" % role)
        try:
            n = int(raw or 0)
        except (TypeError, ValueError):
            n = 0
        return n if n > 0 else None

    def _call_client(self, role: str, prompt_path: str, workdir: str,
                     timeout_s: int, model: Optional[str],
                     max_turns: Optional[int],
                     turn_soft_warn: int = 0) -> Any:
        """调 client.run，按签名能力选择传参。

        注入的客户端（测试 fake / 第三方实现）可能没有新形参——
        用签名探测逐个尝试（task56 的 turn_soft_warn 也要兼容旧 fake）。
        """
        kwargs = {"model": model}
        try:
            params = inspect.signature(self.client.run).parameters
            has_var_kw = any(
                p.kind == inspect.Parameter.VAR_KEYWORD for p in params.values())
            if "max_turns" in params or has_var_kw:
                kwargs["max_turns"] = max_turns
            if turn_soft_warn and ("turn_soft_warn" in params or has_var_kw):
                kwargs["turn_soft_warn"] = turn_soft_warn
        except (TypeError, ValueError):
            # 签名探测失败 → 尝试传全部（旧客户端若有 **kwargs 则能接）
            kwargs["max_turns"] = max_turns
            if turn_soft_warn:
                kwargs["turn_soft_warn"] = turn_soft_warn
        return self.client.run(role, prompt_path, workdir, timeout_s, **kwargs)

    def _dispatch_agent(self, role: str, prompt_path: str, workdir: str,
                        iteration_id: int) -> Any:
        self.events.append("agent_started", {
            "role": role, "iteration_id": iteration_id,
            "workdir": workdir,
        }, "scheduler", experiment_id=getattr(self, "_cur_exp_id", None))
        # task57 R1：熔断已打开 → 本迭代剩余派发快速失败（不再起 pi 进程。
        # env_11 实测一轮配额空转烧满 4 角色×重试链；熔断后短路是主要止损）
        if self._api_outage_active:
            from superopt.agents import AgentResult as _AR
            result = _AR(role=role, valid=False, infra_failure=True,
                         summary="api_outage_active: dispatch skipped"
                                 "（熔断暂停中，探活恢复后自动续跑）")
            self._finish_dispatch(role, result, iteration_id, prompt_path,
                                  workdir)
            return result
        timeout_s = int(self.conf.get("agent_timeout_s", 1800))
        # per-role 模型档位（优化空间①：builder 等角色可降档省 token）
        model = (self.conf.get("agent_models") or {}).get(role)
        # per-role 回合预算（task56 A 段：默认 0=不限，软告警在 client 层）
        max_turns = self._role_max_turns(role)
        turn_soft_warn = int(self.conf.get("turn_soft_warn", 20) or 0)
        try:
            result = self._call_client(role, prompt_path, workdir, timeout_s,
                                       model, max_turns,
                                       turn_soft_warn=turn_soft_warn)
        except Exception as exc:  # agent 超时/崩溃 → 记失败继续（spec）
            from superopt.agents import AgentResult as _AR
            result = _AR(role=role, valid=False, summary="agent error: %s" % exc)
        return self._finish_dispatch(role, result, iteration_id, prompt_path,
                                     workdir)

    def _finish_dispatch(self, role: str, result: Any, iteration_id: int,
                         prompt_path: str, workdir: str) -> Any:
        """派发收尾（task57 拆出）：记账 + 事件 + infra 计数 + 物证落盘。

        （原本体内联；拆出后熔断短路路径与正常路径共用同一收尾——
        账本/事件/分类计数语义一致。）
        """
        # 本迭代 agent 统计（ledger 条目用；同 role 多次运行累加）
        # tool_calls 亦累加——iteration_ended.by_role 的成本可解释性需要
        # 「每角色花了多少 token / 刷了多少工具调用」同帧可读（task46）
        acc = self._iter_agents.get(role) or {"wall_ms": 0, "tokens": 0,
                                              "tool_calls": 0, "truncated": False}
        acc["wall_ms"] = int(acc["wall_ms"]) + int(getattr(result, "wall_ms", 0) or 0)
        acc["tokens"] = int(acc["tokens"]) + int(
            (result.usage or {}).get("total_tokens") or 0)
        acc["tool_calls"] = int(acc.get("tool_calls") or 0) + len(
            result.tool_calls or [])
        acc["truncated"] = bool(acc["truncated"]) or bool(
            getattr(result, "truncated", False))
        # task57 R1：infra 标记进入 by_role 内部口径（outcome 分类用；
        # iteration_ended.by_role 载荷仍只投影 tokens/wall_ms/tool_calls）
        acc["infra_failure"] = bool(acc.get("infra_failure")) or bool(
            getattr(result, "infra_failure", False))
        self._iter_agents[role] = acc
        # 记账 + 事件
        with self._ledger_lock:  # CostLedger 内部无锁；并发槽需串行化
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
        self.events.append("agent_ended", payload, "scheduler",
                           experiment_id=getattr(self, "_cur_exp_id", None))
        # task57 R1：infra 计数 + 断连熔断 + 0-token 后备（factor 自原内联块；
        # 熔断后的暂停/探活在循环层 _probe_api_recovery——不再内联 sleep
        # 阻塞事件流与心跳）
        self._track_infra_failure(result)

        # task56 A 段：turn_warn 连续告警——中控/human 感知后裁决
        if getattr(result, "turn_warn", False):
            self._turn_warn_streak[role] = \
                self._turn_warn_streak.get(role, 0) + 1
            if self._turn_warn_streak[role] >= 3:
                soft = int(self.conf.get("turn_soft_warn", 20) or 20)
                self.events.append("turn_budget_alert", {
                    "role": role,
                    "consecutive": self._turn_warn_streak[role],
                    "soft_warn": soft,
                    "avg_turns_hint": "agent 回合数连续 ≥%d 达软告警阈值"
                                      "——中控/human 裁决是否提高预算或介入排查"
                                      % soft,
                }, "scheduler")
                self._turn_warn_streak[role] = 0  # 重置（告警一次即可）
        else:
            self._turn_warn_streak[role] = 0

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
