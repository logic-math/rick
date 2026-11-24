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
import hashlib
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


class _PipelineBudgetExhausted(Exception):
    """adjust 流水线总预算耗尽（job_56 task3）——已完成步骤产物保留，
    跳至 step6/收尾（保守空提案合成）。"""


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
    visit 的唯一真相在 cell_sampled 事件；本视图保证 super.db cells 表侧一致。
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


def _code_digest(path: str) -> Optional[str]:
    """候选代码的内容摘要（md5，task58/R3 幻影 elite 判别的锚）。

    - 文件 → 内容 md5；
    - 目录 → 有序文件树摘要（相对路径 + 内容，dirs/files 排序——确定性）；
    - 不存在 / 读不可 → None（调用方视为无可比锚，走正常入格路径）。
    语义：同一 digest = 同一解（同码重测不是新纪录——env_11 iter552 实测
    阴性结果 keep_original 的同码在更有利机器态下被当新纪录入账）。
    """
    if not path or not os.path.exists(path):
        return None
    h = hashlib.md5()
    if os.path.isfile(path):
        with open(path, "rb") as f:
            h.update(f.read())
        return h.hexdigest()
    for root, dirs, files in os.walk(path):
        dirs.sort()
        for name in sorted(files):
            fp = os.path.join(root, name)
            h.update(os.path.relpath(fp, path).encode("utf-8"))
            try:
                with open(fp, "rb") as f:
                    h.update(f.read())
            except OSError:
                continue
    return h.hexdigest()


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
    "_chain_slot",             # job_56 task4：本实验链的槽位号（ctx 注入用）
    "_exp_failed",             # job_56 task4：debug 确认实验失败（跳过终验标记）
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
    _cur_workdir_id = _iter_tl_property('_cur_workdir_id')
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
    _chain_slot = _iter_tl_property('_chain_slot')
    _exp_failed = _iter_tl_property('_exp_failed')

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

        # DB 时代：events 存 super.db（五表）；EventWriter 的 path 参数
        # 兼容目录直传（jsonl 文件名形态仅为旧签名兼容，见 events.py）。
        self.events_path = self.ws
        # DB 时代：格状态存 super.db（cells 表）；Archive 的 path 参数
        # 兼容目录直传（json 文件名形态仅为旧签名兼容，见 archive.py）。
        self.archive_path = self.ws
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
            # DB 单库（job_56 task1）：全部存储经 super.db（五表）——
            # EventWriter/Archive/Ledger/InterventionQueue 内部转 DB 后端
            from .db import db_for_ws as _db_for_ws
            self.db = _db_for_ws(self.ws)
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
        # A-02 修复：恢复后上一进程的 pipeline 可能仍在飞——时间窗口内
        # 不重触发（防同 iter pipeline 多次并发——iter_14 被 dispatch 9 次）
        self._adjust_inflight_until = 0.0  # type: float
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
        # ---- task59 R5：同码双测噪声带（测量分辨率锚） ----
        # _noise_band：最近一次冠军精英同码重测的相对差（0 = 未标定/标定关闭，
        # 噪声饱和判据失活）；_noise_calib_last：上次标定时的迭代号（幂等防重）
        self._noise_band = 0.0  # type: float
        self._noise_calib_last = 0  # type: int
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
        # A-02 修复：恢复时检查 adjusts 表——最近一轮的 created_at 距今 <
        # total_timeout + 300s → 标记在飞等待窗口
        try:
            _adjs = self.db.list_adjusts(n=1)
            if _adjs:
                import datetime as _dt
                _ts = _adjs[0].get("created_at") or ""
                _t = _dt.datetime.fromisoformat(_ts).timestamp() if _ts else 0
                _total = int(self.conf.get("adjust_total_timeout_s", 10800)
                             or 10800)
                if _t > 0 and time.time() - _t < _total + 300:
                    self._adjust_inflight_until = _t + _total + 300
        except Exception:  # noqa: BLE001
            pass
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
        # 资源槽初始化（job_56 task1 KR-C）：实验槽/环境槽按 config 落
        # resources 表；启动即清残留占用（进程锁保证单写者——残留 holder
        # 必是上个死进程的遗留）
        self._init_resource_slots()
        self._recover()
        # ---- 格级互斥启动清理（迭代9/task45）----
        # 上个进程被 SIGKILL 时可能留下锁文件：持有者 pid 已死 → 立即清理
        # （不必等 TTL），避免恢复后的首个迭代空等。
        released = self.cell_locks.release_orphans()
        if released:
            self.events.append("cell_lock_orphans_released", {
                "count": len(released), "cells": released[:20]}, "scheduler")
        # P4（step7 提案4）：会话重启吞噬在飞实验 → 台账死行清理——
        # 恢复时把 status=running 但无 record_path 的死行标 interrupted
        # （不残留 running 污染采样决策；visit 账实由 archive.load 的
        # 事件重放投影校正）
        try:
            dead = self.db.interrupt_stale_running()
            if dead:
                self.events.append("stale_running_interrupted", {
                    "experiment_ids": dead}, "scheduler")
                # #9-B：死行对应的槽位也要释放（防泄漏——review E-B07）
                for _eid in dead:
                    try:
                        self.db.release_holder(_eid)
                    except Exception:  # noqa: BLE001
                        pass
        except Exception:  # noqa: BLE001 DB 故障不炸启动
            pass
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

    def _unalloc_cursor(self, cursor: int) -> None:
        """槽位占位失败回退迭代号（env_9：防 cursor 跳号堆积空目录）。"""
        with self._state_lock:
            if cursor == self.iteration_cursor:
                self.iteration_cursor -= 1

    def _adjust_due(self, adjust_every: int) -> bool:
        """跨边界触发判定（task54）：cursor - 上次 adjust >= adjust_every。

        等值判断（cursor % N == 0）在并发 cursor 跳跃（如 3→7 直接跳过 4）时会
        漏判；改为区间判断后两路径（串行/并发）语义一致。首轮 _last_adjust_iter=0
        → cursor 达到 N 即触发。
        """
        if adjust_every <= 0 or self.iteration_cursor <= 0:
            return False
        return (self.iteration_cursor - self._last_adjust_iter) >= adjust_every

    def _probe_dbg(self, tag, extra=""):
        """临时调试探针（job_56 僵死循环定位——写 /tmp/superopt_dbg.log，定位后撤）。"""
        try:
            import time as _t
            with open("/tmp/superopt_dbg.log", "a", encoding="utf-8") as _f:
                _f.write("%s [dbg] %s %s\n" % (_t.strftime("%H:%M:%S"), tag, extra))
        except Exception:
            pass

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
            self._probe_dbg("tick", "cursor=%d" % self.iteration_cursor)
            if not slots:
                self._probe_dbg("tick-noslots")
                self._maybe_reload_feature_space()
                self._drain_decisions()
                # task59 R5：噪声标定（无在飞实验时对冠军精英同码重测；
                # 幂等触发见方法内——cursor % interval == 0 才跑）
                self._maybe_run_noise_calibration()
                if (self.iteration_cursor > 0
                        and self.iteration_cursor % EPOCH_ITERATIONS == 0):
                    self._probe_dbg("reeval-pending")
                    self._reevaluate_pending()
            # ④ 补满在飞槽（每槽 = 一个实验：独占一格）
            while (len(slots) < self.max_parallel_experiments
                   and not self.stop_requested
                   and (max_iter <= 0 or self.iteration_cursor < max_iter)):
                # humans #37：backlog-aware——槽位释放先读积压清单直派
                cell_id = self._backlog_dispatch_and_lock()
                if cell_id is None:
                    cell_id = self._sample_and_lock()
                if cell_id is None:
                    break  # 全格被占/溢出停机（事件已落）
                cursor = self._alloc_cursor()
                exp_id = make_experiment_id(cell_id, cursor)
                # 资源槽占用（job_56 task1 KR-C）：实验槽+环境槽（二级限额）。
                # env_9 实测 bug 修复：**先占资源位，后注册实验行建目录**——
                # 旧顺序（register→acquire）在槽满时每 tick 泄漏一个实验
                # 行+目录（env_slots 收紧后目录轰炸：数分钟 1300+ 空目录）。
                # 占位标记用 cursor 序号（占位失败不消耗时间戳实验 ID）。
                _hold = "pending-%d" % cursor
                exp_slot = self.db.acquire_slot("experiment_slot", _hold)
                if exp_slot is None:
                    self._unalloc_cursor(cursor)
                    self.cell_locks.release(cell_id)
                    break  # 实验槽满：等在飞释放
                env_slot = self.db.acquire_slot("env_slot", _hold)
                if env_slot is None:
                    self.db.release_holder(_hold)  # 归还实验槽
                    self._unalloc_cursor(cursor)
                    self.cell_locks.release(cell_id)
                    break  # 环境槽满：等在飞释放
                workdir_id = self._register_experiment_row(cursor, cell_id)
                # 占位改持真实实验 ID（槽位持有者从 pending-N 换绑）
                self.db.rebind_holder(exp_slot, workdir_id)
                self.db.rebind_holder(env_slot, workdir_id)
                slot = next_slot
                next_slot += 1
                self.events.append("experiment_started", {
                    "slot": slot, "iteration_id": cursor, "cell_id": cell_id,
                    "workdir_experiment_id": workdir_id,
                    "exp_slot": exp_slot, "env_slot": env_slot,
                }, "scheduler", experiment_id=exp_id)
                th = threading.Thread(
                    target=self._slot_worker,
                    args=(slot, cell_id, cursor, exp_id, workdir_id),
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
                     exp_id: str, workdir_id: Optional[str] = None) -> None:
        """槽 worker：跑一次实验（已持格锁），结束时释放并落收尾事件。

        workdir_id（job_56 task1）：工作目录实验 ID（experiments 表 PK）——
        资源位持有标识；结束时归还资源位（实验槽+环境槽）。
        slot（job_56 task4）：实验槽位号（ctx 注入 builder/coding/debug）。
        """
        t0 = time.time()
        try:
            self._step_locked(cell_id, cursor=cursor, exp_id=exp_id,
                              workdir_id=workdir_id, slot=slot)
        except Exception as exc:  # noqa: BLE001 —— 单实验失败不杀调度器
            try:
                self.events.append("experiment_failed", {
                    "slot": slot, "iteration_id": cursor, "cell_id": cell_id,
                    "error": str(exc)[:400],
                }, "scheduler", experiment_id=exp_id)
            except Exception:  # noqa: BLE001
                pass
        finally:
            if workdir_id:
                try:
                    self.db.release_holder(workdir_id)  # 归还实验槽+环境槽
                except Exception:  # noqa: BLE001
                    pass
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
            # ④′ task59 R5：迭代边界噪声标定（cursor 已推进；无在飞实验，
            #     冠军精英可安全同码重测——幂等触发见方法内）
            self._maybe_run_noise_calibration()
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

        空事件流 = 全新开始。恢复以 super.db events 表为准（cells 表为投影，事务一致性无滞后）：
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

    def _backlog_dispatch_and_lock(self) -> Optional[str]:
        """humans #37：backlog-aware dispatch——积压清单优先直派。

        槽位释放路径先读积压清单（env/skills/backlog_manifest.json——
        机器可读 priority 序）：非空按 priority 序直派（跳过已锁/无效项），
        消费后清单同步出队（写回原子替换）；清单空/不可读 → None（回归
        纯随机采样）。risk 缓解：清单消费后即出队（防清单吞没随机探索）。
        """
        try:
            mp = os.path.join(self.env_dir, "skills",
                              "backlog_manifest.json")
            if not os.path.isfile(mp):
                return None
            with open(mp, "r", encoding="utf-8") as f:
                manifest = json.load(f)
        except (OSError, ValueError):
            return None
        items = manifest if isinstance(manifest, list) else (
            manifest.get("items") if isinstance(manifest, dict) else None)
        if not isinstance(items, list) or not items:
            return None
        taken = None
        for it in items:
            cid = it.get("cell_id") if isinstance(it, dict) else None
            if not isinstance(cid, str) or not cid:
                continue
            if cid not in self.fs.all_cells():
                continue  # 无效项（空间外）：跳过不消费（留 manifest 维护方清理）
            if self.cell_locks.acquire(cid):
                taken = cid
                break
            # 已锁项跳过（别的槽在跑）——不消费（下轮再看）
        if taken is None:
            return None
        # 消费出队（写回剩余清单：原子替换）
        try:
            remaining = [it for it in items
                         if not (isinstance(it, dict)
                                 and it.get("cell_id") == taken)]
            tmp = mp + ".tmp"
            with open(tmp, "w", encoding="utf-8") as f:
                json.dump(remaining, f, ensure_ascii=False, indent=2)
            os.replace(tmp, mp)
        except OSError:
            pass
        self.events.append("backlog_dispatch", {
            "iteration_cursor": self.iteration_cursor,
            "cell_id": taken,
            "manifest_remaining": len(remaining),
            "reason": "humans #37 backlog-aware 直派（清单消费出队）",
        }, "scheduler")
        return taken

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
        + **探活驻留**（humans #26 根修：不退出进程——旧行为优雅停机后
        11h 空窗止损实证）。驻留循环周期探活，连续 infra_recovery_probes
        次成功 → 自动恢复迭代；stop_requested（human 干预）→ 优雅停。
        0-token 秒退是认证/配额/供应商级故障的稳定信号；真实验失败都烧 token。
        """
        breaker = int(self.conf.get("agent_failure_breaker", 8) or 0)
        if breaker <= 0:
            return False
        if self._consec_zero_token_fails < breaker or self._infra_failure:
            return False
        self._infra_failure = True
        self.events.append("infra_failure_concluded", {
            "consecutive_zero_token_fails": self._consec_zero_token_fails,
            "threshold": breaker,
            "diagnosis": "连续 %d 次 agent 0-token 失败——认证/配额/"
                          "供应商级故障（非实验失败）。调度器进入探活驻留"
                          "（humans #26：不退出进程，探活 %d 次成功自动续跑；"
                          "human 可随时 stop 干预）。" % (
                              self._consec_zero_token_fails,
                              int(self.conf.get("infra_recovery_probes", 3)
                                  or 3)),
        }, "scheduler")
        return self._infra_hold_loop()

    def _infra_hold_loop(self) -> bool:
        """humans #26：infra_failure 探活驻留（不退出进程）。

        - 每 api_outage_retry_s 探活一次（复用 _run_api_probe 极小 agent）；
        - 连续 infra_recovery_probes 次成功（默认 3×60s）→ 自动恢复迭代
          （重置熔断/断连计数，infra_failure_recovered 事件）→ 返回 False；
        - stop_requested（human stop / SIGTERM）→ 返回 True（优雅停保持
          既有语义）；
        - 驻留期间 heartbeat 持续（watchdog 巡检可见驻留态非死机）。
        """
        probes_needed = int(self.conf.get("infra_recovery_probes", 3) or 3)
        retry_s = int(self.conf.get("api_outage_retry_s", 60) or 0)
        # 驻留上限（真·配额耗尽/供应商长瘫：探活连续失败 infra_hold_max_fails
        # 次 → 放弃驻留优雅停机〔旧语义兜底——提案 #26 risk 预案的落地〕，
        # 默认 30 次 × 60s = 30 分钟）
        max_fails = int(self.conf.get("infra_hold_max_fails", 30) or 30)
        ok_streak = fail_streak = 0
        while not self.stop_requested:
            self._sleep_interruptible(retry_s)
            if self.stop_requested:
                break
            self._drain_interventions()
            if self.stop_requested:
                break
            if self._run_api_probe():
                ok_streak += 1
                if ok_streak >= probes_needed:
                    self.events.append("infra_failure_recovered", {
                        "probes_streak": ok_streak,
                        "diagnosis": "探活连续 %d 次成功——API 恢复，自动"
                                     "续跑（humans #26 探活驻留：无空窗）"
                                     % ok_streak,
                    }, "scheduler")
                    self._infra_failure = False
                    self._api_outage_active = False
                    self._consec_infra_fails = 0
                    self._consec_zero_token_fails = 0
                    return False
                fail_streak = 0
            else:
                ok_streak = 0
                fail_streak += 1
                if fail_streak >= max_fails:
                    self.events.append("infra_hold_abandoned", {
                        "consecutive_probe_fails": fail_streak,
                        "diagnosis": "探活连续 %d 次失败（~%d 分钟）——真·"
                                     "配额耗尽/供应商长瘫，放弃驻留优雅停机"
                                     "（human 换号/修号后 rick super start 续跑）"
                                     % (fail_streak, fail_streak * retry_s // 60),
                    }, "scheduler")
                    self.stop_requested = True
                    break
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

    # ------------------------------------------------------------------ #
    # task59 R5：同码双测噪声标定 + 噪声带饱和判据
    # ------------------------------------------------------------------ #

    def _maybe_run_noise_calibration(self) -> None:
        """task59 R5：迭代边界的同码双测噪声标定（测量分辨率锚）。

        触发：cursor % noise_calibration_interval == 0（默认 10；0 = 关闭）。
        幂等：_noise_calib_last 记上次标定迭代号，同一 cursor 不重标。

        标的：当前冠军格的在位精英（code_path + 档案 perf_value）——
        同码重测（与 gate_check 同协议：gate.verify 同一候选路径）→
        相对差 |重测 - 档案| / |档案| → noise_calibrated 事件 →
        self._noise_band（饱和判据的分辨率锚，见 _saturation_signal）。

        失败静默（gate 挂/精英缺失/非数值 → 不打断主循环——标定是增强
        信号，非关键路径）；noise_pct=0 是合法读数（测量稳定）。
        """
        interval = int(self.conf.get("noise_calibration_interval", 10) or 0)
        cursor = int(self.iteration_cursor or 0)
        if interval <= 0 or cursor <= 0:
            return
        if cursor % interval != 0 or self._noise_calib_last == cursor:
            return
        self._noise_calib_last = cursor

        champ = self._champion_cell()
        if champ is None:
            return
        elite = (self._cell_entry(champ).get("elite") or {})
        code_path = elite.get("code_path")
        archived = elite.get("perf_value")
        if (not isinstance(code_path, str) or not code_path
                or not isinstance(archived, (int, float))
                or isinstance(archived, bool) or float(archived) == 0.0
                or not os.path.exists(code_path)):
            return
        try:
            gr = self.gate.verify(code_path)
        except Exception:  # noqa: BLE001 —— 标定失败不破坏主循环
            return
        perf = ((getattr(gr, "metrics", None) or {}).get("perf") or {})
        current = perf.get("value") if isinstance(perf, dict) else None
        if (not getattr(gr, "passed", False)
                or not isinstance(current, (int, float))
                or isinstance(current, bool)):
            return
        noise_pct = abs(float(current) - float(archived)) / abs(float(archived))
        self._noise_band = noise_pct
        self.events.append("noise_calibrated", {
            "iteration_id": cursor,
            "cell_id": champ,
            "code_path": code_path,
            "current_perf": float(current),
            "archived_perf": float(archived),
            "noise_pct": round(noise_pct, 6),
        }, "gate")

    def _identified_headroom(self, snap: Dict[str, Any]) -> Optional[float]:
        """task59 R5：已识别剩余空间（相对当前最优，无据可算 → None）。

        代理口径：冠军格精英时间序（history 旧→新 + 在位）最近 3 次
        gen-over-gen 增量绝对值合计 / |当前最优|。env_11 判例语义：已识别
        （armed）杠杆合计 2-4ns < 跨会话噪声 12-18% → 继续迭代的期望收益
        低于测量分辨率——框架可机读的近似 = 近期机制收益在收敛。
        """
        champ = self._champion_cell()
        if champ is None:
            return None
        entry = snap.get(champ) or {}
        current = (entry.get("elite") or {}).get("perf_value")
        if (not isinstance(current, (int, float)) or isinstance(current, bool)
                or float(current) == 0.0):
            return None
        seq = [h.get("perf_value") for h in (entry.get("history") or [])]
        seq = [float(v) for v in seq
               if isinstance(v, (int, float)) and not isinstance(v, bool)]
        seq.append(float(current))  # 时间序：history（旧→新）+ 在位精英
        deltas = [abs(seq[i] - seq[i - 1])
                  for i in range(max(1, len(seq) - 3), len(seq))]
        if not deltas:
            return None
        return sum(deltas) / abs(float(current))

    def _noise_saturation_evidence(
            self, snap: Dict[str, Any]) -> Optional[Dict[str, Any]]:
        """task59 R5：噪声带饱和证据（两支判据任一触发才非 None）。

        - top3_spread < noise_band：头三名格精英差距小于噪声带 →
          位次不可区分（排序键分辨率不足）；
        - identified_headroom < noise_band：已识别剩余空间合计小于
          噪声带 → 收益不可认证（继续迭代的期望收益为负）。
        触发 → 附 stop_reason="headroom_below_noise"（adjust 可据此提
        terminate_recommend——停止判据从「预算跑道」改锚到「测量分辨率」）。
        """
        band = float(self._noise_band or 0.0)
        if band <= 0.0:
            return None
        low = (self.perf_desc != "high")
        values = []  # type: List[float]
        for cid in self.fs.all_cells():
            pv = ((snap.get(cid) or {}).get("elite") or {}).get("perf_value")
            if isinstance(pv, (int, float)) and not isinstance(pv, bool):
                values.append(float(pv))
        if len(values) < 3:
            return None  # 头三名无从谈起
        values.sort(reverse=not low)
        best = values[0]
        top3_spread = (abs(values[2] - best) / abs(best)
                       if best != 0.0 else None)
        headroom = self._identified_headroom(snap)
        evidence = {"noise_band": round(band, 6)}  # type: Dict[str, Any]
        triggered = False
        if top3_spread is not None:
            evidence["top3_spread"] = round(top3_spread, 6)
            if top3_spread < band:
                triggered = True
        if headroom is not None:
            evidence["identified_headroom"] = round(headroom, 6)
            if headroom < band:
                triggered = True
        if not triggered:
            return None
        evidence["stop_reason"] = "headroom_below_noise"
        return evidence

    def _init_resource_slots(self) -> None:
        """资源表初始化 + 残留占用清理（KR-C：实验槽/环境槽二级限额）。

        实验槽 = max_parallel_experiments 个；环境槽 = env_slots_per_env 个
        （调度器仅管资源分配——两实验可否同 env 并发；同环境内分时复用
        是 ENV 设计责任/gen_env 规则，task2 落文档）。
        """
        try:
            # 环境槽：取值链 conf.env_slots_per_env（显式覆盖）> env.json.env_slots
            # （env 声明——gen_env 产物）> 实验槽数（不构成额外瓶颈）。
            # （Q7/Q8 裁决；env_9 实测 bug：conf 键与 env.json 键不一致致
            # env_slots=1 声明未生效、4 agent 同环境并发污染测量）
            raw_env_slots = int(self.conf.get("env_slots_per_env", 0) or 0)
            if raw_env_slots <= 0:
                try:
                    raw_env_slots = int(
                        self._load_env_meta().get("env_slots") or 0)
                except (TypeError, ValueError):
                    raw_env_slots = 0
            env_slots = raw_env_slots if raw_env_slots > 0 \
                else self.max_parallel_experiments
            self.db.init_slots(self.max_parallel_experiments, env_slots)
            # 清残留占用（上个进程崩溃遗留——进程锁保证此刻无他人持槽）
            import sqlite3 as _sq
            with self.db._lock:
                self.db._conn.execute(
                    "UPDATE resources SET holder_experiment_id=NULL, "
                    "acquired_at=NULL WHERE holder_experiment_id IS NOT NULL")
                self.db._conn.commit()
        except Exception:  # noqa: BLE001 资源表故障降级内存槽池（不炸启动）
            pass

    def _register_experiment_row(self, cursor: int,
                                 cell_id: Optional[str] = None) -> str:
        """experiments 表登记（job_56 task1 KR-E）：返回工作目录实验 ID。

        `<iteration>-<YYYYMMDDHHMM>`（迭代号-年月日时分；同分钟并行撞号
        自动加序号后缀——db.make_experiment_timestamp_id / register）。
        存储调用行封装（task4 的 sub loop 写回 record_path/session_id/score）。
        """
        try:
            return self.db.register_experiment(int(cursor), cell_id=cell_id)
        except Exception:  # noqa: BLE001 DB 故障不炸实验链（兜底旧目录名）
            return "iter%03d-%s" % (cursor, uuid.uuid4().hex[:8])

    def _step_locked(self, cell_id: str,
                     cursor: Optional[int] = None,
                     exp_id: Optional[str] = None,
                     workdir_id: Optional[str] = None,
                     slot: Optional[int] = None) -> None:
        """单迭代主体（已持有 cell_id 的格锁；_step 的 finally 保证释放）。

        task49：本迭代 = 一次实验，唯一标记 ``experiment_id``（格+迭代号）；
        迭代内全部事件都带该标记（并行写入时用 read_experiment / grep 还原
        单实验事件流），运维事件（心跳/干预/裁决）不带。
        slot（job_56 task4 KR-F）：实验槽位号（ctx 注入三角色；串行路径
        None = 串行模式独占）。
        """
        # 并发槽（task50）：迭代号与实验标记由主线程在起槽时分配（原子），
        # 串行路径（_step）未传入时在此分配——两条路径语义一致。
        if cursor is None:
            cursor = self._alloc_cursor()
        if exp_id is None:
            exp_id = make_experiment_id(cell_id, cursor)
        # 实验标记（格 + 迭代号；cell_safe 形态，见 events.make_experiment_id）
        self._cur_exp_id = exp_id
        if workdir_id:
            self._cur_workdir_id = workdir_id
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
        # job_56 task4：本实验链的槽位号（ctx 注入）+ debug 确认失败标记
        self._chain_slot = slot
        self._exp_failed = False
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

        # ---- 迭代工作区（codes/{experiment_id}/ 隔离快照；job_56 task1 KR-E：
        # 实验表 PK = <iteration>-<YYYYMMDDHHMM> = 工作目录名；串行/旧路径
        # 未传 workdir_id 时兜底 register（同分钟撞号自动后缀）） ----
        if workdir_id is None:
            workdir_id = self._register_experiment_row(cursor, cell_id)
        iter_dir = os.path.join(self.codes_dir, workdir_id)
        os.makedirs(iter_dir, exist_ok=True)

        # ---- 状态 2+3（无缝重启 resume 分支：中断实验从断点续跑）----
        resumed = self._try_resume_experiment(cell_id, cursor, iter_dir)
        if resumed is not None:
            plan, candidate = resumed
        else:
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
        # sub loop 写回 DB（job_56 task4 KR-E）：实验表 UPDATE
        # （record_path/session_id/score/status）——失败路径（debug 确认失败
        # /无候选）也写回（KR-D：status='failed'，score=NULL）
        self._writeback_experiment(workdir_id, iter_dir, gr, candidate)

    def _writeback_experiment(self, workdir_id: Optional[str],
                              iter_dir: str, gr: Any,
                              candidate: Optional[str]) -> None:
        """实验链收尾写回 experiments 表（job_56 task4 KR-E）。

        状态语义（KR-D）：
          - admitted：候选入格（gate 融合分写入 score）
          - rejected：候选存在但门禁拒（score=NULL）
          - failed：无候选 / debug 确认失败（experiment_failed_confirmed
            事件已落——区别于普通失败，可查 events 表）
        session_id：实验链主 agent 的 pi session（优先 debug > coding >
        builder——最后行为轨迹的会话；_finish_dispatch 存入 by_role）。
        """
        if not workdir_id:
            return
        try:
            status = "failed"
            score = None
            if candidate is not None and gr is not None and gr.passed:
                status = "admitted"
                perf = (getattr(gr, "metrics", None) or {}).get("perf") or {}
                score = perf.get("score")
                if score is None:
                    score = perf.get("value")
                if isinstance(score, (int, float)):
                    score = float(score)
            elif candidate is not None:
                status = "rejected"
            # 三角色 session 分字段（job_56 第三轮审计：builder/coding/debug
            # 各自的 pi session——行为轨迹还原；单列覆盖退役）
            per_role_sessions = {}
            for role in ("builder", "coding", "debug"):
                acc = self._iter_agents.get(role) or {}
                sid = acc.get("session_id") or ""
                if sid:
                    per_role_sessions["%s_session_id" % role] = sid
            record_path = experiment_record.path_for(iter_dir)
            self.db.update_experiment(
                workdir_id,
                record_path=record_path if os.path.isfile(record_path)
                else "",
                score=score,
                status=status,
                **per_role_sessions)
        except Exception:  # noqa: BLE001 DB 故障不炸实验链（事件已可诊断）
            pass

    def _slot_info_text(self) -> str:
        """实验槽位信息（ctx 注入三角色，job_56 task4 KR-F）。"""
        try:
            slot = self._chain_slot
        except AttributeError:
            slot = None
        if slot is None:
            return "串行模式（独占运行环境，无并发实验）"
        return "实验槽位 %d（并行实验位，当前实验池上限 %d）" % (
            int(slot), self.max_parallel_experiments)

    def _env_slot_info_text(self) -> str:
        """运行环境槽位信息（ctx 注入三角色，job_56 task4 KR-F）。

        调度器仅管资源分配（两实验可否同环境并发）；同环境实验如何交错
        使用独占资源（分时复用）是 ENV 的设计问题（见 domain_knowledge.md）。
        """
        raw = int(self.conf.get("env_slots_per_env", 0) or 0)
        if raw <= 0:
            try:
                raw = int(self._load_env_meta().get("env_slots") or 0)
            except (TypeError, ValueError):
                raw = 0
        n = raw if raw > 0 else self.max_parallel_experiments
        return ("运行环境：本 env 工作区（%s）；同环境并发实验上限 %d——"
                "与其他实验共享本环境时的独占资源分时复用规则，见"
                " domain_knowledge.md（调度器仅管资源分配，复用机制由 env"
                " 设计定义）" % (os.path.basename(self.ws.rstrip("/")), n))

    # ---- builder ----

    def _problem_source_for_cell(self, cell_id: str) -> str:
        """题库模式：cell_id=problem=<题名> → 题目 py 全文注入 builder ctx。

        job_56 刷榜模式（KernelBench level2）：题目是实验的「规格」——
        builder 必须看到题目源码（Model/get_inputs/get_init_inputs）。
        非题库 env（cell_id 无 problem= 前缀）→ 空串（模板侧给缺省说明）。
        """
        try:
            prefix = "problem="
            if not cell_id.startswith(prefix):
                return ""
            name = cell_id[len(prefix):]
            for sub in ("level2", "level1", "level3"):
                p = os.path.join(self.env_dir, "problems", sub,
                                 name + ".py")
                if os.path.isfile(p):
                    return open(p, encoding="utf-8").read()
        except Exception:  # noqa: BLE001 题源注入失败不炸实验链
            pass
        return ""

    def _try_resume_experiment(self, cell_id: str, cursor: int,
                                iter_dir: str):
        """无缝重启（job_56 resume）：同 cell 的 interrupted 实验断点续跑。

        coding 断点：pi --session-id 续会话（上下文在）+ 续跑指令——省
        builder 一轮 + coding 已完成部分；coding_done 断点：产物直交后续
        链（省 builder+coding 两轮——debug/gate 走正常链）。
        返回 (plan, candidate) 或 None（无可续跑 → 正常链）。
        """
        self._probe_dbg("resume-try", cell_id[:44])
        try:
            r = self.db.find_resumable_experiment(cell_id)
        except Exception:  # noqa: BLE001
            return None
        if not r:
            return None
        wd = r["workdir"]
        # plan 从旧 workdir 读（builder 产物——resume 时 builder 不重跑）
        plan = {"idea": "resumed", "target_cell": cell_id,
                "parents": [], "plan_steps": []}
        try:
            with open(os.path.join(wd, "experiment_plan.json"),
                      encoding="utf-8") as f:
                p2 = json.load(f)
            if isinstance(p2, dict):
                plan = p2
        except Exception:  # noqa: BLE001 旧 plan 读失败用兜底
            pass
        plan.setdefault("target_cell", cell_id)
        self.events.append("experiment_resumed", {
            "iteration_id": cursor, "cell_id": cell_id,
            "resumed_from": r["experiment_id"],
            "resume_role": r["role"], "coding_done": bool(r["coding_done"]),
            "workdir": wd,
            "diagnosis": "无缝重启：中断实验从断点续跑（%s 断点）"
                         % r["role"],
        }, "scheduler", experiment_id=self._cur_exp_id)
        if r["coding_done"]:
            # coding 已完成 → 产物直交（candidate=workdir 内最新候选）
            cand = self._latest_candidate_in(wd)
            if cand:
                return plan, cand
            # coding_done 但产物不在（被清）→ 回退正常链
            self.events.append("experiment_resumed_fallback", {
                "iteration_id": cursor, "cell_id": cell_id,
                "reason": "coding_done 但候选产物不在盘——回退正常链",
            }, "scheduler", experiment_id=self._cur_exp_id)
            return None
        # coding 断点：续会话跑完
        try:
            result = self.client.run_resumed(
                "coding", r["session_id"], wd,
                timeout_s=int(self.conf.get("agent_timeout_s", 1800) or 1800))
        except Exception as exc:  # noqa: BLE001 续跑失败回退正常链
            self.events.append("experiment_resumed_fallback", {
                "iteration_id": cursor, "cell_id": cell_id,
                "reason": "resume 派发异常: %s" % str(exc)[:200],
            }, "scheduler", experiment_id=self._cur_exp_id)
            return None
        self._finish_dispatch("coding", result, cursor, None, wd)
        cand = (result.artifact_path
                or self._latest_candidate_in(wd) or "")
        if result.valid and cand:
            return plan, cand
        # 续跑仍未完成（valid=False 且无产物）→ 回退正常链
        self.events.append("experiment_resumed_fallback", {
            "iteration_id": cursor, "cell_id": cell_id,
            "reason": "resume 后仍无有效产物——回退正常链",
        }, "scheduler", experiment_id=self._cur_exp_id)
        return None

    def _latest_candidate_in(self, workdir: str) -> Optional[str]:
        """workdir 内最新候选产物（solution.*/candidate.*——mtime 最新）。"""
        best, best_mt = None, -1.0
        try:
            for name in os.listdir(workdir):
                low = name.lower()
                if (low.startswith("solution.") or
                        low.startswith("candidate.")) and \
                        low.endswith((".py", ".cu", ".cc", ".cpp")):
                    p = os.path.join(workdir, name)
                    try:
                        mt = os.path.getmtime(p)
                    except OSError:
                        continue
                    if os.path.getsize(p) > 0 and mt > best_mt:
                        best, best_mt = p, mt
        except OSError:
            return None
        return best

    def _run_builder(self, cell_id: str, cursor: int, iter_dir: str) -> Dict[str, Any]:
        entry = self._cell_entry(cell_id)
        elite = entry.get("elite") or {}
        # task58/R4：处方-基座相位差注入——method_stale 的格，builder 的
        # 方法描述落后于当前精英（env_11 实测滞后 1-2 代 → 按过期处方做
        # 实验，30-60min 级空转）。[STALE] 标注 + 强制以 elite_code_path
        # 实际代码为第一事实源（描述只作历史参考）。
        elite_ctx = dict(elite) if elite else {}
        # 读侧存在性校验（env_9 实测：DB 与磁盘失配时 ENOENT 注入）
        _cp = elite_ctx.get("code_path")
        if isinstance(_cp, str) and _cp and not os.path.exists(_cp):
            self.events.append("elite_path_missing", {
                "cell_id": cell_id, "missing_path": _cp,
                "downgrade": "cell_elite 置空（首访语义）",
            }, "scheduler")
            elite_ctx = {}
        if self.archive.is_stale(cell_id) and elite_ctx:
            elite_ctx["method_stale"] = True
            elite_ctx["stale_note"] = (
                "[STALE] 本格 method 描述落后于当前精英（精英已换代）——"
                "以 elite_code_path 的实际代码为第一事实源，method 描述"
                "仅作历史参考")
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
            # job_56 task4 KR-F：实验 ID（=工作目录名）+ 资源槽位注入
            "experiment_id": getattr(self, "_cur_workdir_id", "")
            or getattr(self, "_cur_exp_id", "") or "",
            # 题库模式（job_56 刷榜）：题目源码全文注入（规格即输入）
            "problem_source": self._problem_source_for_cell(cell_id),
            "trace_id": getattr(self, "_cur_exp_id", "") or "",
            "slot_info": self._slot_info_text(),
            "env_slot_info": self._env_slot_info_text(),
            "cell_id": cell_id,
            "cell_descriptor": self.fs.describe_cell(cell_id) or {},
            # 特征格方法描述（cells 表 method_desc——本次实验要推进的方法论方向）
            "cell_method_desc": str(
                (entry.get("description") if isinstance(
                    entry.get("description"), dict) else {}).get("method")
                or "") or (
                "（无 method 描述——按描述子与精英代码推断，并在 reasoning 里说明）"),
            "cell_elite": self._or_default(elite_ctx or None, "（无精英——本格首访）"),
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
                    _cp = e["elite"].get("code_path")
                    if isinstance(_cp, str) and _cp and not os.path.exists(_cp):
                        self.events.append("elite_path_missing", {
                            "cell_id": cid, "missing_path": _cp,
                            "downgrade": "邻格精英跳过",
                        }, "scheduler")
                        continue
                    neighbors.append({
                        "cell_id": cid, "descriptor": nd,
                        "elite": {"code_path": _cp,
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
            # job_56 task4 KR-F：实验 ID + 资源槽位注入（coding）
            "experiment_id": getattr(self, "_cur_workdir_id", "")
            or getattr(self, "_cur_exp_id", "") or "",
            # 题库模式（job_56 刷榜）：题目源码全文注入（规格即输入）
            "problem_source": self._problem_source_for_cell(cell_id),
            "trace_id": getattr(self, "_cur_exp_id", "") or "",
            "slot_info": self._slot_info_text(),
            "env_slot_info": self._env_slot_info_text(),
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

        candidate = self._candidate_path(result, iter_dir, self.ws)
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
            dpath = self._candidate_path(debug_result, iter_dir, self.ws)
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
            # job_56 task4 KR-C/D：debug 反思后确认实验失败（主动上报）→
            # 跳过 gate_check 端到端验证（debug 已完成完整反思，终验不会再
            # 产生新信息；省部署/预热/测试全流程，加速迭代）。实验表收尾
            # 记 status='failed'（score=NULL）+实验记录 path 写回。
            if debug_result.data.get("experiment_failed"):
                self._exp_failed = True
                self.events.append("experiment_failed_confirmed", {
                    "iteration_id": cursor, "cell_id": cell_id,
                    "attempt": attempt, "role": "debug",
                    "reason": "debug agent 反思后确认实验失败（完整四节"
                              "实验记录已落盘），跳过端到端验证",
                }, "agent:debug", experiment_id=self._cur_exp_id)
                return None
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
            # job_56 task4 KR-F：实验 ID + 资源槽位注入（debug）
            "experiment_id": getattr(self, "_cur_workdir_id", "")
            or getattr(self, "_cur_exp_id", "") or "",
            # 题库模式（job_56 刷榜）：题目源码全文注入（规格即输入）
            "problem_source": self._problem_source_for_cell(cell_id),
            "trace_id": getattr(self, "_cur_exp_id", "") or "",
            "slot_info": self._slot_info_text(),
            "env_slot_info": self._env_slot_info_text(),
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
        """把父本代码拷进迭代工作区（coding 提示词的「父本已就位」契约）。

        E-B04 修复（job_56 review）：fallback 链扩展——
        plan.parents → elite.code_path → **该格最近一次尝试的 solution.py**
        （从 experiments 表按 iteration DESC 找该格最新有 code 的实验——
        无精英格（全失败历史）时历史尝试代码仍被继承）。
        """
        parents = [p for p in (plan.get("parents") or []) if isinstance(p, str)]
        if not parents:
            elite = self._cell_entry(cell_id).get("elite") or {}
            cp = elite.get("code_path")
            if isinstance(cp, str) and cp:
                parents = [cp]
        if not parents:
            # E-B04：无精英时从 experiments 表找最近尝试的 code_path
            try:
                rows = self.db.recent_experiments(n=50)
                for r in rows:
                    if r.get("cell_id") == cell_id and r.get("record_path"):
                        code_p = os.path.join(
                            os.path.dirname(r["record_path"]),
                            "solution.py")
                        if os.path.isfile(code_p):
                            parents = [code_p]
                            break
            except Exception:  # noqa: BLE001 DB 故障退化为 env 参考
                pass
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
    def _candidate_path(result: Any, iter_dir: str,
                        ws_root: str = "") -> Optional[str]:
        diff = result.data.get("diff_path")
        if isinstance(diff, str) and diff.strip():
            if os.path.isabs(diff):
                if os.path.exists(diff):
                    return diff
                return diff  # agent 声称的路径（门禁 stage1 会验存在性）
            # humans #39①：相对 diff 多基解析（iter155 路径双写全损判例：
            # ws 根相对路径盲 join iter_dir → 双写路径 → stage1 not found
            # → 全损）。三级回退：iter_dir/diff → iter_dir/basename →
            # ws_root/diff（覆盖 ws 根相对形态）；全部不存在才返回声称路径。
            for cand in (os.path.join(iter_dir, diff),
                         os.path.join(iter_dir, os.path.basename(diff)),
                         (os.path.join(ws_root, diff)
                          if ws_root else "")):
                if cand and os.path.exists(cand):
                    return cand
            return os.path.join(iter_dir, diff)  # 声称路径（stage1 兜底验存在）
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
        self._gpu_window_advisory(cell_id, cursor)
        self._probe_dbg("verify-admit", "%s %s" % (cell_id[:28], str(candidate)[-36:]))
        entry = self._cell_entry(cell_id)
        elite = entry.get("elite")
        history = [h.get("perf_value") for h in (entry.get("history") or [])
                   if h.get("perf_value") is not None]
        gr = self.gate.verify(candidate,
                              elite_perf=(elite or {}).get("perf_value"),
                              history=history)
        # humans #27②：无读数失败（score=-6：verify 传输/崩溃/OOM——非候选
        # 缺陷）→ 同候选重试一次（与超差失败分离——无读数重测有意义，
        # 真超差重测无意义；限 1 次防慢候选墙放大）。
        _no_reading = (not gr.passed
                       and getattr(gr, "score", None) is not None
                       and float(getattr(gr, "score")) == -6.0)
        if _no_reading:
            self.events.append("gate_no_reading_retry", {
                "iteration_id": cursor, "cell_id": cell_id,
                "subclass": getattr(gr, "metrics", None) or None,
                "reason": "verify 无读数失败（humans #27）——同候选重试 1 次",
            }, "scheduler", experiment_id=self._cur_exp_id)
            gr = self.gate.verify(candidate,
                                  elite_perf=(elite or {}).get("perf_value"),
                                  history=history)
        self._emit_gate_result(gr, cursor, cell_id)

        if gr.passed:
            # 迭代8/KR6：门禁通过后仍可能复验——①异常收益（超 human 确认阈值，
            # 防作弊/误测的性能结论进精英地图）②软门禁容差带内（无明显收益）。
            # 每迭代至多 1 次；debug 结论 keep_original → 原候选照常入格。
            reason = self._recheck_reason(gr, cell_id)
            # humans #38：零改动/同码复验直通——tolerance_band 且候选 MD5 ==
            # 精英 MD5（客观判据）⟹ 直通 measurement_refresh 免 debug 派发
            # （A/A 族 debug 税结构性根除：同码 gain=0 by design ⟹ delta 必
            # 落容差带 ⟹ 结论可先验；MD5 不同维持既有复验路径不动）。
            if (reason == "tolerance_band_recheck"
                    and self._same_code_as_elite(candidate, cell_id)):
                self._recheck_verdict = "keep_original_zero_change"
                self.events.append("zero_change_aa_fastpath", {
                    "iteration_id": cursor, "cell_id": cell_id,
                    "code_path": candidate,
                    "gate_perf": (gr.metrics or {}).get("perf", {}).get("value"),
                    "elite_perf": (elite or {}).get("perf_value"),
                    "reason": "zero_change_aa（humans #38：同码容差带直通"
                              " measurement_refresh，免 debug 派发）",
                }, "scheduler", experiment_id=getattr(self, "_cur_exp_id", None))
                self._admit(candidate, cell_id, cursor, plan, gr,
                            had_elite=bool(elite))
                self.env_fail_streak = 0
                return gr
            if reason is not None and not self._recheck_done_this_iter:
                self._recheck_done_this_iter = True
                iter_dir = os.path.dirname(os.path.abspath(candidate))
                fixed, verdict, recheck_valid = self._run_recheck_debug(
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
                elif verdict == "keep_original" and recheck_valid:
                    # humans #36①：debug verdict=keep_original ⟹ 不 replace
                    # （幻增益防线：cell 51 判例——debug 判幻增益/测噪但写侧
                    # 仍替换了精英值，噪声棘轮自增强）。同码 → _admit 的
                    # measurement_refresh 分支（只刷测量值不 replace）；
                    # 异码 keep_original → 维持原精英（不替换，事件留痕）。
                    if not self._same_code_as_elite(candidate, cell_id):
                        self._last_admit_verdict = "keep_original"
                        self._last_reject_reason = (
                            "debug_verdict_keep_original（humans #36："
                            "复验否决替换——维持原精英）")
                        self.events.append("elite_replace_vetoed", {
                            "iteration_id": cursor, "cell_id": cell_id,
                            "code_path": candidate,
                            "vetoed_perf": (gr.metrics or {}).get(
                                "perf", {}).get("value"),
                            "elite_perf": (elite or {}).get("perf_value"),
                            "reason": "debug_verdict_keep_original",
                        }, "scheduler",
                            experiment_id=getattr(self, "_cur_exp_id", None))
                        self.env_fail_streak = 0
                        return gr
                    # 同码 keep_original：走 _admit（同码分支 → refresh）
            self._admit(candidate, cell_id, cursor, plan, gr, had_elite=bool(elite))
            self.env_fail_streak = 0
            return gr

        stage = gr.failure_stage or ""
        if stage.startswith("stage1"):
            # humans #39②：stage1「candidate not found」类 env 路由失败 →
            # 产物抢救（iter155 判例：真文件在盘而路径双写误报 → 全损）。
            # fresh-mtime 候选在盘即代位重跑 gate（终验才是真判据——与
            # agent_completed_but_no_json 抢救同一路径）。
            _detail = str(gr.details or "")
            if "candidate not found" in _detail.lower():
                iter_dir = os.path.dirname(os.path.abspath(candidate))
                rescued = self._rescue_artifact(iter_dir, time.time() - 86400)
                if rescued is not None and rescued != candidate:
                    self.events.append("artifact_rescued", {
                        "iteration_id": cursor, "cell_id": cell_id,
                        "code_path": rescued,
                        "reason": "stage1_candidate_not_found_rescue"
                                  "（humans #39：路径解析失败产物代位）",
                        "original_claim": str(candidate)[:200],
                    }, "scheduler",
                        experiment_id=getattr(self, "_cur_exp_id", None))
                    gr2 = self.gate.verify(
                        rescued,
                        elite_perf=(elite or {}).get("perf_value"),
                        history=history)
                    self._emit_gate_result(gr2, cursor, cell_id)
                    if gr2.passed:
                        self._admit(rescued, cell_id, cursor, plan, gr2,
                                    had_elite=bool(elite))
                        self.env_fail_streak = 0
                        return gr2
                    # 代位也失败 → 走原 stage1 结论化（真实失败）
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
        # job_56 task4 KR-D：门禁失败触发的 debug 也可能确认实验失败
        #（例如根因是验证机制问题/作弊行为——终验不会再产生新信息）
        if result.data.get("experiment_failed"):
            self._exp_failed = True
            self.events.append("experiment_failed_confirmed", {
                "iteration_id": cursor, "cell_id": cell_id,
                "attempt": 100, "role": "debug",
                "reason": "门禁失败 debug 反思后确认实验失败，跳过端到端重验",
            }, "agent:debug", experiment_id=self._cur_exp_id)
            return None
        dpath = self._candidate_path(result, iter_dir, self.ws)
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

        # ---- task58/R3：同码等价检查（幻影 elite 防线）----
        # env_11 实测判例：阴性结果（keep_original，交付形态=elite 等价零回归）
        # 的同码在更有利机器态下重测读数更优 → gate 入账为全场新纪录。
        # 正确语义：同码 = 同解 → 拒绝 elite_replaced，只刷新测量值，
        # gen/位次/方法基座/历史最优（饱和信号）全部不动。
        new_hash = _code_digest(candidate)
        old_hash = self.archive.get_code_hash(cell_id)
        if had_elite and new_hash and old_hash == new_hash:
            self.archive.refresh_measurement(
                cell_id, perf_value, ref_runtime_ms=perf.get("ref_value"))
            self._last_admit_verdict = "measurement_refresh"
            self._last_reject_reason = None
            self.events.append("elite_measurement_refresh", {
                "iteration_id": cursor, "cell_id": cell_id,
                "code_hash": new_hash,
                "new_perf": perf_value,
                "old_perf": parent_elite.get("perf_value"),
                "ref_runtime_ms": perf.get("ref_value"),
                "gen": int(parent_elite.get("gen") or 0),
                "reason": "same_code_remeasure",
            }, "scheduler",
                experiment_id=getattr(self, "_cur_exp_id", None))
            self.archive.save()
            return

        entry["elite"]["ref_runtime_ms"] = perf.get("ref_value")
        admitted = self.archive.admit(cell_id, entry, record_path=record_path,
                                      code_hash=new_hash)
        if not admitted:
            # 更差解：同格替换条件不满足（事件流无 elite_*——诚实记录）
            self._last_admit_verdict = "reject"
            self._last_reject_reason = "worse_than_elite"
            return
        self._last_admit_verdict = "replace" if had_elite else "admit"
        # task58/R4：精英换代 → method 基座可能落后于新精英（处方-基座相位差）。
        # env_11 实测 3 格在案：method 基座滞后 elite 1-2 代 → 高份额格按过期
        # 处方做实验（30-60min 级空转）。adjust 的 modify_cell 更新 method 后清除。
        if had_elite:
            self.archive.set_stale(cell_id, True)
        # 饱和状态：历史最优刷新（方向感知）
        self._note_perf_record(perf_value)

        # ---- 延迟入库判定（L2-Q6：三者任一） ----
        _w = self._cell_entry(cell_id).get("weight")
        weight = float(_w) if isinstance(_w, (int, float)) else 1.0
        # humans #36②：k=1 单读数 replace（suspicious_gain 复验通过的替换单）
        # 一律 pending 化（confidence<0.8）直至 A/A 复验落带解除——与既有
        # pending new_cell 机制同构（噪声棘轮防线：单读数幻增益不直接换精英）
        _k1_replace = bool(had_elite and getattr(self, "_recheck_verdict", None)
                           and self._recheck_verdict != "keep_original")
        if gr.tolerance_band:
            reason = "tolerance_band"
        elif _k1_replace:
            reason = "k1_recheck_replace"
            entry["elite"]["confidence"] = 0.5
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

    def _record_adjust_started(self, iteration_id: int, adjust_dir: str) -> None:
        """BUG-009 修复：流水线开始时写 adjusts 表（status=started）——
        中断的 pipeline 也有记录（decisions 有但 adjusts 无的对称性问题）。"""
        try:
            self.db.record_adjust(iteration_id, adjust_dir, sessions={},
                                  decisions_path=None, agent_valid=None)
        except Exception:  # noqa: BLE001
            pass

    def _maybe_start_adjust(self) -> None:
        """边界触发：后台派发 adjust 决策层（job_56 task3 双模式）。

        - ``adjust_mode=pipeline``（默认）：7 步 subagent 串行流水线（step3 每
          视角并行 subagent）——每步独立上下文、产物落盘传递（answer-isolation）
        - ``adjust_mode=legacy``：原单 agent 单次调用（兼容回退，deprecated）

        行为约束（两模式共用）：同一时刻至多 1 个在飞；在飞未收时到边界
        不重触发；线程只跑 client.run 子进程链（无共享可变状态，除 agg
        聚合器——线程内串行访问，主线程不碰）；agent_started 总事件在
        主线程（事件序确定性）。
        """
        if self.stop_requested:
            return
        if time.time() < self._adjust_inflight_until:
            return  # A-02：上代 pipeline 可能仍在飞（缓冲期内不重触发）
        with self._adjust_lock:
            if self._adjust_thread is not None:
                return  # 在飞未收：不重触发
        iteration_id = self.iteration_cursor
        adjust_dir = os.path.join(self.codes_dir, "adjust",
                                  "iter_%d" % iteration_id)
        os.makedirs(adjust_dir, exist_ok=True)
        mode = "pipeline"
        # ---- 共享 ctx（pipeline 按步分发）----
        shared = {
            "task": self.task_name, "perf_desc": self.perf_desc,
            "perf_type": self.perf_type,
            # OKR 思考锚（human 裁决：adjust 基于 OKR 思考不跑偏——目标从
            # 用户需求解析，env.json okr 字段独立要素；缺省时降级提示）
            "okr": self._okr_text(),
            "feature_axes": self.fs.axes,
            "iteration_id": iteration_id,
            "cells_status": self._cells_status(),
            "recent_events": self._recent_events_summary(15),
            "saturation": self._or_default(self._saturation_signal(), "（未饱和）"),
            "perf_timeline": self._perf_timeline(10),
            "budget_status": self._budget_status(),
            "domain_brief": self._domain_brief(),
            "ledger_brief": self._ledger_brief(8),
            "own_decisions": self._or_default(self._own_decisions("adjust", 5), "（暂无历史决策）"),
            "interventions": self._recent_interventions(3),
            "gate_check_cmd": self._element_cmds()["gate_check_cmd"],
            "research_cmd": self._research_cmd("adjust"),
            "research_brief": self._or_default(
                self._research_brief("adjust"),
                "（未提供检索指引——跳过检索）"),
            "experiment_records": self._or_default(
                self._experiment_records(self._last_adjust_iter,
                                         self.iteration_cursor),
                "（区间内尚无实验记录——首轮或 builder 未落盘）"),
            "experiment_sessions": self._experiment_sessions_count(
                self._last_adjust_iter, self.iteration_cursor),
            "decision_history": self._or_default(
                self._decision_history(3), "（暂无历史决策记录）"),
            "cell_descriptions": self._cell_descriptions(),
            "decisions_path": self._decisions_path(iteration_id),
            "adjust_dir": adjust_dir,
        }
        # agent_started（总事件，role=adjust——聚合视图；各步另有细粒度
        # agent_started/adjust_step_* 事件）在主线程（事件序确定性）
        self.events.append("agent_started", {
            "role": "adjust", "iteration_id": iteration_id,
            "workdir": adjust_dir, "mode": mode,
        }, "scheduler", experiment_id=make_experiment_id("adjust", iteration_id))

        def _worker():
            try:
                result = self._run_adjust_pipeline(
                    adjust_dir, iteration_id, shared)
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
            # BUG-009 修复：开始时也写 adjusts 表
            self._record_adjust_started(iteration_id, adjust_dir)
        # 区间起点（A-04 修复：dispatch 时不再推进 _last_adjust_iter——
        # 推进移到 _collect_adjust 成功消费后。旧实现在 dispatch 时同步
        # 更新导致多次 dispatch 竞争时区间起点乱序（iter_8 先于 iter_7 触发））
        old_last = self._last_adjust_iter
        # task54 可观测性：何时触发、距上次多少迭代（并发饥饿修复的回归证据）
        self.events.append("adjust_dispatched", {
            "iteration": iteration_id,
            "window_since": iteration_id - old_last,
            "mode": mode,
        }, "scheduler")

    # ---- adjust 7 步流水线（job_56 task3；pipeline 模式）----

    def _adjust_step_model(self, role: str) -> Optional[str]:
        """步骤超参数（KR-C）：env.json agent_hyperparams > agent_models 链。

        hyperparams[role] = {model, thinking}：两者都有 → ``--model id:thinking``
        （pi 官方语法）；仅 model → model；仅 thinking → 无法脱离 model 传递
        （跟随 agent_models 链，prompt 老化风险由 gen_env 校验提示）；
        无配置 → agent_models[role] → agent_models["adjust"] → None。
        """
        hp = self._load_env_meta().get("agent_hyperparams")
        h = hp.get(role) if isinstance(hp, dict) and isinstance(
            hp.get(role), dict) else {}
        model = str(h.get("model") or "").strip() or None
        thinking = str(h.get("thinking") or "").strip() or None
        if model and thinking:
            return "%s:%s" % (model, thinking)
        if model:
            return model
        am = self.conf.get("agent_models") or {}
        return am.get(role) or am.get("adjust")

    def _adjust_step_resume_check(self, role: str, prompt_suffix: str,
                                  adjust_dir: str, iteration_id: int,
                                  step_no: int) -> Any:
        """humans #35：已完成步骤检测（双判据）→ 保真重建或 None（重派）。

        判据（提案 risk①：adjust_step_ended 事件为完成权威，文件存在不可
        信——半写可能）：①agent_{role}{suffix}.json 在盘且可解析且 valid=true
        ②事件流有同 iteration 的 adjust_step_ended（step 对应）valid=true。
        data 从 JSON 读回（#35 落盘增强后的新 JSON；旧 JSON 无 data → None
        → 下游走标准产物路径 fallback——向后兼容）。
        """
        prompt_role = ("%s%s" % (role, prompt_suffix)) if prompt_suffix else role
        jp = os.path.join(adjust_dir, "agent_%s.json" % prompt_role)
        try:
            with open(jp, "r", encoding="utf-8") as f:
                j = json.load(f)
        except (OSError, ValueError):
            return None
        if not isinstance(j, dict) or not j.get("valid"):
            return None
        # 事件权威（半写 JSON 防线）：同 iteration + step 的 valid ended
        try:
            ended = [ev for ev in self.events.read_all()
                     if ev.get("type") == "adjust_step_ended"
                     and (ev.get("payload") or {}).get("step") == step_no
                     and (ev.get("payload") or {}).get("valid")
                     and (ev.get("payload") or {}).get("iteration_id")
                     == iteration_id]
            if not ended:
                return None
        except Exception:  # noqa: BLE001 事件读失败保守重派
            return None
        from superopt.agents import AgentResult as _AR
        data = j.get("data")
        return _AR(
            role=role, valid=True,
            summary=("[resumed] %s" % str(j.get("summary") or ""))[:400],
            usage=dict(j.get("usage") or {}),
            wall_ms=int(j.get("wall_ms") or 0),
            exit_code=0, tool_calls=[], model=str(j.get("model") or ""),
            truncated=bool(j.get("truncated")), turn_warn=bool(j.get("turn_warn")),
            json_repair=int(j.get("json_repair") or 0),
            session_id=str(j.get("session_id") or ""),
            data=(data if isinstance(data, dict) else None))

    def _pipeline_step(self, step_no: int, role: str, ctx: Dict[str, Any],
                       adjust_dir: str, iteration_id: int,
                       step_timeout: int, deadline: float,
                       agg: Dict[str, Any],
                       prompt_suffix: str = "",
                       step_sessions: Optional[Dict[str, Any]] = None,
                       session_key: str = "") -> Any:
        """单步派发（含事件与降级）：render → write_prompt → client.run。

        prompt_suffix：步骤内并行的文件名隔离（step3 每视角
        ``_{index}``——同 role 并行写 prompt_{role}.md 会覆盖冲突）。

        失败降级（KR-A）：跳过+默认值继续+``adjust_step_degraded`` 事件记账
        （不整链重跑）；总预算耗尽 → 抛 _PipelineBudgetExhausted 由调用方收尾。
        events.append 线程安全（EventWriter 复用 DB 实例锁——task1）。
        """
        from superopt.agents import AgentResult as _AR
        try:
            remaining = deadline - time.time()
            if remaining <= 120:
                raise _PipelineBudgetExhausted(
                    "adjust 总预算耗尽（剩余 %.0fs）——跳至收尾" % remaining)
            # ---- humans #35：步骤级断点续跑（杀波损失从整链降为当前步）----
            # 双判据（提案 risk①：事件为完成权威——半写 JSON 不可信）：
            # agent JSON 在盘且可解析且 valid=true + adjust_step_ended
            # 事件（同 iteration/step）valid=true → 跳过重派，JSON data 保真
            # 重建（改动前旧 JSON 无 data 字段 → 空 data 走下游 fallback）。
            resumed = self._adjust_step_resume_check(
                role, prompt_suffix, adjust_dir, iteration_id, step_no)
            if resumed is not None:
                self.events.append("adjust_step_resumed", {
                    "step": step_no, "role": role,
                    "iteration_id": iteration_id,
                    "reason": "humans #35：已完成步骤跳过重派（断点续跑）",
                }, "scheduler", experiment_id=make_experiment_id(
                    role, iteration_id))
                return resumed
            timeout = int(min(step_timeout, remaining))
            prompt = render(role, self.ws, ctx)
            prompt_role = ("%s%s" % (role, prompt_suffix)) if prompt_suffix else role
            prompt_path = write_prompt(adjust_dir, prompt_role, prompt)
            self.events.append("adjust_step_started", {
                "step": step_no, "role": role,
                "iteration_id": iteration_id,
            }, "scheduler", experiment_id=make_experiment_id(
                role, iteration_id))
            self.events.append("agent_started", {
                "role": role, "iteration_id": iteration_id,
                "workdir": adjust_dir, "step": step_no,
            }, "scheduler", experiment_id=make_experiment_id(
                role, iteration_id))
            result = self._call_client(
                role, prompt_path, adjust_dir, timeout,
                self._adjust_step_model(role), self._role_max_turns(role))
            ep_payload = result.to_payload()
            # agent_ended 必填字段（events.py 校验）：iteration_id（原管道由
            # _collect_adjust/_dispatch_agent 在 append 前补——流水线同构）
            ep_payload["iteration_id"] = iteration_id
            ep_payload["step"] = step_no
            self.events.append("agent_ended", ep_payload,
                               "scheduler",
                               experiment_id=make_experiment_id(
                                   role, iteration_id))
            self.events.append("adjust_step_ended", {
                "step": step_no, "role": role,
                "valid": bool(result.valid),
                "wall_ms": int(result.wall_ms or 0),
            }, "scheduler", experiment_id=make_experiment_id(
                role, iteration_id))
            # 聚合（agg 仅本流水线线程访问——step3 并行子线程也只写自己段）
            u = result.usage or {}
            agg["wall_ms"] += float(result.wall_ms or 0)
            agg["tokens"] += float(u.get("total_tokens", 0) or 0)
            agg["cost_usd"] += float(u.get("cost_usd", 0) or 0)
            agg["tool_calls"] += float(len(result.tool_calls or []))
            agg["truncated"] = agg["truncated"] or bool(result.truncated)
            if not result.valid:
                agg["valid"] = False
            # session_id 收集（job_56 第三轮审计：adjusts 表数据源；
            # step3 并行以 session_key 区分——数组聚合）
            if step_sessions is not None:
                sid = getattr(result, "session_id", "") or ""
                if sid:
                    key = session_key or ("step%d" % step_no)
                    if session_key:
                        step_sessions.setdefault("step%d" % step_no,
                                                 []).append(sid)
                    else:
                        step_sessions[key] = sid
            return result
        except _PipelineBudgetExhausted:
            raise
        except Exception as exc:  # noqa: BLE001 单步失败不杀流水线
            self.events.append("adjust_step_degraded", {
                "step": step_no, "role": role,
                "reason": str(exc)[:300],
            }, "scheduler", experiment_id=make_experiment_id(
                role, iteration_id))
            agg["degraded_steps"] = agg.get("degraded_steps", 0) + 1
            return _AR(role=role, valid=False,
                       summary="step degraded: %s" % exc)

    def _hindsight_evidence(self) -> List[Dict[str, Any]]:
        """上轮 adjust 决策的事后验证（KR-G）——主要矛盾判断对错的证据。

        读 super.db events 表：最近一次 adjust_decision 的 proposals digest →
        对每个 target cell 查该决策之后的采样/入格/权重事件 →
        被升权格后续产出 / 被杀格事后引用 / 新格存活，注入下轮 step1 ctx。
        """
        try:
            # A-08 修复：只读最近 500 条事件（不全量——10万+事件线性退化）
            evs = self.db.read_events()[-500:]
        except Exception:  # noqa: BLE001 事后验证是辅助证据——失败不炸主链
            return []
        last_seq = None
        proposals = []  # type: List[Dict[str, Any]]
        for e in reversed(evs):
            if e.get("type") == "adjust_decision":
                last_seq = int(e.get("seq") or 0)
                pd = (e.get("payload") or {}).get("proposals")
                if isinstance(pd, list):
                    proposals = [p for p in pd if isinstance(p, dict)]
                break
        if last_seq is None:
            return []
        # 决策后事件索引（cell_id → 采样/入格计数）
        sampled = {}  # type: Dict[str, int]
        admitted = {}  # type: Dict[str, int]
        for e in evs:
            if int(e.get("seq") or 0) <= last_seq:
                continue
            p = e.get("payload") or {}
            cid = p.get("cell_id")
            if not cid:
                continue
            if e.get("type") == "cell_sampled":
                sampled[cid] = sampled.get(cid, 0) + 1
            elif e.get("type") in ("elite_admitted", "elite_replaced"):
                admitted[cid] = admitted.get(cid, 0) + 1
        out = []
        for p in proposals:
            cell = p.get("target_cell") or ""
            if not cell:
                d = p.get("descriptor")
                if isinstance(d, dict):
                    cell = "|".join("%s=%s" % kv for kv in sorted(d.items()))
            if not cell:
                continue
            action = str(p.get("action") or p.get("type") or "")
            out.append({
                "cell": cell, "last_action": action,
                "subsequent_sampled": sampled.get(cell, 0),
                "subsequent_admitted": admitted.get(cell, 0),
                # 语义：升权后无产出=判断存疑；杀格后被采样=误杀候选；
                # 新格存活=方向正确（下游编码事实由 step1 归纳）
            })
        return out

    def _okr_text(self) -> str:
        """OKR 思考锚（env.json okr 字段：O=目标 / KRs=关键结果）。

        human 裁决（job_56）：adjust 的分析目标 = 提升 O（刷榜分数类目标），
        不是优化运行时基建——基建只在「阻塞 KR 达成」时才值得提案。
        """
        try:
            okr = (self._load_env_meta() or {}).get("okr") or {}
            o = str(okr.get("O") or "").strip()
            krs = okr.get("KRs") or []
            if not o and not krs:
                return ("（env 未定义 OKR——本 env 应在 env.json 补 okr 字段："
                        "O=从用户需求解析的目标 / KRs=关键结果）")
            lines = ["O: %s" % o] if o else []
            for kr in (krs if isinstance(krs, list) else [krs]):
                t = str(kr).strip()
                if t:
                    lines.append("- %s" % t)
            return "\n".join(lines)
        except Exception:  # noqa: BLE001 OKR 缺省不炸 adjust
            return "（OKR 读取失败——按既有上下文思考）"

    def _run_adjust_pipeline(self, adjust_dir: str, iteration_id: int,
                             shared: Dict[str, Any]) -> Any:
        """7 步流水线主体（线程内执行；输出合成 AgentResult role=adjust）。

        步骤串行（step3 每 perspective 并行 subagent）；产物落盘传递
        （answer-isolation）；step6 的 proposals 是下游消费管道的输入
        （_collect_adjust 的 proposals 应用/裁决链零改动）；step7 在
        step6 后执行（env 知识沉淀+提案入队，不阻塞 proposals）。
        """
        from superopt.agents import AgentResult as _AR
        agg = {"wall_ms": 0.0, "tokens": 0.0, "cost_usd": 0.0,
               "tool_calls": 0.0, "truncated": False, "valid": True,
               "degraded_steps": 0}  # type: Dict[str, Any]
        # 各步 pi session_id（job_56 第三轮审计：adjusts 表数据源——
        # step3 三视角并行是数组）
        step_sessions = {}  # type: Dict[str, Any]
        step_timeout = max(60, int(self.conf.get("adjust_step_timeout_s", 900)))
        deadline = time.time() + max(
            120, int(self.conf.get("adjust_total_timeout_s", 5400)))
        briefs = []  # type: List[Dict[str, Any]]
        try:
            # ---- step1 问题还原（humans 表未应用干预 = human 价值判断）----
            pi = []  # type: List[Dict[str, Any]]
            try:
                pi = self.db.pending_interventions()
            except Exception:  # noqa: BLE001 humans 表是期望锚点——失败降级
                pass
            ctx1 = {**shared,
                    "pending_interventions": self._or_default(
                        pi, "（无未应用干预）"),
                    "hindsight_evidence": self._or_default(
                        self._hindsight_evidence(), "（首轮——无事后验证证据）")}
            r1 = self._pipeline_step(1, "adjust_fact", ctx1, adjust_dir,
                                     iteration_id, step_timeout, deadline, agg,
                                     step_sessions=step_sessions)
            facts_path = str((r1.data or {}).get("facts_path") or
                             os.path.join(adjust_dir, "facts.md"))
            briefs.append({"step": 1, "role": "adjust_fact",
                           "valid": bool(r1.valid),
                           "summary": str(r1.summary or "")[:200]})

            # ---- step2 视角生成（启发式跳跃点）----
            ctx2 = {"task": shared["task"],
                    "iteration_id": iteration_id,
                    "facts_path": facts_path,
                    "research_brief": shared["research_brief"],
                    "research_cmd": shared["research_cmd"],
                    "adjust_dir": adjust_dir}
            r2 = self._pipeline_step(2, "adjust_perspective", ctx2,
                                     adjust_dir, iteration_id, step_timeout,
                                     deadline, agg, step_sessions=step_sessions)
            perspectives_path = str((r2.data or {}).get(
                "perspectives_path") or os.path.join(
                    adjust_dir, "perspectives.md"))
            perspectives = (r2.data or {}).get("perspectives")
            if not isinstance(perspectives, list) or not perspectives:
                # A-05 修复：step2 失败时空列表——不再用占位视角跑 step3×3
                # subagent（浪费 LLM 调用产出垃圾矛盾），step3/4 自然跳过
                perspectives = []  # 空列表 → step3/4 跳过
            perspectives = [str(p) for p in perspectives][:3]
            while len(perspectives) < 3:
                perspectives.append("（视角 %d 降级默认）" % (len(perspectives) + 1))
            briefs.append({"step": 2, "role": "adjust_perspective",
                           "valid": bool(r2.valid),
                           "perspectives": perspectives})

            # ---- step3 矛盾生成（每视角 1 个并行 subagent）----
            step3_results = [None] * len(perspectives)

            def _run_one(idx: int, perspective: str) -> None:
                ctx3 = {"task": shared["task"],
                        "iteration_id": iteration_id,
                        "perspective_index": idx + 1,
                        "perspective": perspective,
                        "facts_path": facts_path,
                        "perspectives_path": perspectives_path,
                        "adjust_dir": adjust_dir}
                step3_results[idx] = self._pipeline_step(
                    3, "adjust_contradiction", ctx3, adjust_dir,
                    iteration_id, step_timeout, deadline, agg,
                    prompt_suffix="_%d" % (idx + 1),
                    step_sessions=step_sessions,
                    session_key="step3_p%d" % (idx + 1))

            ths = [threading.Thread(target=_run_one, args=(i, p),
                                    name="superopt-adjust-c%d" % (i + 1),
                                    daemon=True)
                   for i, p in enumerate(perspectives)]
            for t in ths:
                t.start()
            for t in ths:
                t.join()
            contradictions_brief = []  # type: List[Dict[str, Any]]
            for i, r3 in enumerate(step3_results):
                d = (r3.data or {}) if r3 is not None else {}
                contradictions_brief.append({
                    "perspective_index": i + 1,
                    "perspective": perspectives[i],
                    "primary_contradiction": d.get("primary_contradiction")
                    or "（该视角矛盾生成失败降级）",
                    "logic_basis": d.get("logic_basis") or "",
                })
            briefs.append({"step": 3, "role": "adjust_contradiction",
                           "parallel": len(perspectives),
                           "contradictions": contradictions_brief})

            # ---- step4 主要矛盾判断 ----
            ctx4 = {"task": shared["task"],
                    "iteration_id": iteration_id,
                    "contradiction_path_1": os.path.join(
                        adjust_dir, "contradiction_1.md"),
                    "contradiction_path_2": os.path.join(
                        adjust_dir, "contradiction_2.md"),
                    "contradiction_path_3": os.path.join(
                        adjust_dir, "contradiction_3.md"),
                    "contradictions_brief": contradictions_brief,
                    "facts_path": facts_path,
                    "decision_history": shared["decision_history"],
                    "adjust_dir": adjust_dir}
            r4 = self._pipeline_step(4, "adjust_judge", ctx4, adjust_dir,
                                     iteration_id, step_timeout, deadline, agg,
                                     step_sessions=step_sessions)
            judgment_path = str((r4.data or {}).get("judgment_path") or
                                os.path.join(adjust_dir, "judgment.md"))
            judgment_brief = {
                "primary_contradiction": (r4.data or {}).get(
                    "primary_contradiction") or "（判定降级——step4 失败）",
                "secondary_contradictions": (r4.data or {}).get(
                    "secondary_contradictions") or [],
                "summary": str(r4.summary or "")[:200],
            }
            briefs.append({"step": 4, "role": "adjust_judge",
                           "valid": bool(r4.valid), "judgment": judgment_brief})

            # ---- step5 辩证逆转 ----
            ctx5 = {"task": shared["task"],
                    "iteration_id": iteration_id,
                    "judgment_path": judgment_path,
                    "judgment_brief": judgment_brief,
                    "facts_path": facts_path,
                    "adjust_dir": adjust_dir}
            r5 = self._pipeline_step(5, "adjust_reversal", ctx5, adjust_dir,
                                     iteration_id, step_timeout, deadline, agg,
                                     step_sessions=step_sessions)
            reversal_brief = {
                "methods": (r5.data or {}).get("methods") or [],
                "summary": str(r5.summary or "")[:200],
            }
            briefs.append({"step": 5, "role": "adjust_reversal",
                           "valid": bool(r5.valid), "reversal": reversal_brief})

            # ---- step6 解空间自改进（输出步：proposals）----
            ctx6 = {"task": shared["task"],
                    "perf_desc": self.perf_desc,
                    "iteration_id": iteration_id,
                    "judgment_brief": judgment_brief,
                    "reversal_brief": reversal_brief,
                    "feature_axes": shared["feature_axes"],
                    "cells_status": shared["cells_status"],
                    "cell_descriptions": shared["cell_descriptions"],
                    "decisions_path": shared["decisions_path"],
                    "adjust_dir": adjust_dir}
            r6 = self._pipeline_step(6, "adjust_space", ctx6, adjust_dir,
                                     iteration_id, step_timeout, deadline, agg,
                                     step_sessions=step_sessions)
            proposals = (r6.data or {}).get("proposals")
            if not isinstance(proposals, list):
                proposals = []
            briefs.append({"step": 6, "role": "adjust_space",
                           "valid": bool(r6.valid),
                           "n_proposals": len(proposals)})
        except _PipelineBudgetExhausted as exc:
            # 总预算耗尽：已完成步骤产物保留；step6 未跑 → 保守空提案合成
            briefs.append({"step": 0, "role": "pipeline",
                           "degraded": "budget_exhausted",
                           "reason": str(exc)[:200]})
            proposals = []
            r6 = None  # type: ignore
        # ---- step7 实验自改进（env 沉淀+提案；预算外也尽力跑——快步）----
        env_proposals = []  # type: List[Dict[str, Any]]
        try:
            ctx7 = {"task": shared["task"],
                    "iteration_id": iteration_id,
                    "experiment_records": shared["experiment_records"],
                    "recent_events": shared["recent_events"],
                    "domain_knowledge_path": os.path.join(
                        self.env_dir, "domain_knowledge.md"),
                    "hindsight_evidence": self._or_default(
                        self._hindsight_evidence(), "（无）"),
                    # A-07 修复：step7 加 cells_status（预算调节需格权重上下文）
                    "cells_status": shared.get("cells_status", []),
                    "budget_status": shared.get("budget_status", {}),
                    "adjust_dir": adjust_dir}
            r7 = self._pipeline_step(7, "adjust_env", ctx7, adjust_dir,
                                     iteration_id, step_timeout, deadline, agg,
                                     step_sessions=step_sessions)
            ep = (r7.data or {}).get("env_proposals")
            if isinstance(ep, list):
                env_proposals = [p for p in ep if isinstance(p, dict)]
            briefs.append({"step": 7, "role": "adjust_env",
                           "valid": bool(r7.valid),
                           "n_env_proposals": len(env_proposals),
                           "domain_knowledge_appended": bool(
                               (r7.data or {}).get("domain_knowledge_appended"))})
        except _PipelineBudgetExhausted:
            briefs.append({"step": 7, "role": "adjust_env",
                           "degraded": "budget_exhausted"})
        except Exception as exc:  # noqa: BLE001
            briefs.append({"step": 7, "role": "adjust_env",
                           "degraded": str(exc)[:200]})
        # env_proposals → humans 表提案队列（job_56 权限边界收紧：仅
        # restart/verifier 两类入队——其余类是 step7 该自修的，入队=违规拒绝）
        for p in env_proposals:
            kind = str(p.get("kind") or "")
            if kind not in ("restart", "verifier"):
                self.events.append("env_proposal_rejected", {
                    "iteration": iteration_id, "kind": kind,
                    "title": p.get("title"),
                    "reason": "权限边界：非 restart/verifier 类应 step7 直接修复",
                }, "scheduler")
                continue
            try:
                self.db.push_proposal({
                    "title": str(p.get("title") or "")[:200],
                    "kind": kind,
                    "finding": str(p.get("finding") or "")[:2000],
                    "proposal": str(p.get("proposal") or "")[:2000],
                    "risk": str(p.get("risk") or "")[:1000],
                    "iteration": iteration_id,
                })
                self.events.append("env_proposal_queued", {
                    "iteration": iteration_id,
                    "kind": kind, "title": p.get("title"),
                }, "scheduler")
            except Exception:  # noqa: BLE001 提案入队失败不炸流水线
                pass
        # env_fixes：step7 直接修复的审计记录（自修类——无需重启即生效）
        try:
            fixes = (r7.data or {}).get("env_fixes") \
                if isinstance(r7.data, dict) else None
            if isinstance(fixes, list) and fixes:
                self.events.append("env_fixed", {
                    "iteration": iteration_id, "fixes": fixes[:12],
                }, "scheduler")
        except Exception:  # noqa: BLE001
            pass
        # budget_adjust：预算热调（step7 自调节权——迭代边界应用无需重启；
        # human 限定键（HUMAN_PINNED_BUDGET_KEYS）拒绝）
        try:
            ba = (r7.data or {}).get("budget_adjust") \
                if isinstance(r7.data, dict) else None
            if isinstance(ba, dict):
                applied = self._apply_budget_adjust(ba, iteration_id)
                if applied:
                    self.events.append("budget_adjusted", {
                        "iteration": iteration_id, "applied": applied,
                    }, "scheduler")
        except Exception:  # noqa: BLE001 预算调节失败不炸流水线
            pass
        # ---- 合成 AgentResult（role=adjust——_collect_adjust 消费管道零改动）----
        # step7 的 data 搬运（job_56 框架 bug 修复：budget_adjust/env_fixes/
        # env_proposals 在 r7.data 里——之前合成时被丢弃导致 step7 的预算
        # 热调/直接修复审计全部丢失（budget_adjusted 事件全史为零的根因））
        step7_data = r7.data if isinstance(r7.data, dict) else {}
        data = {
            "proposals": proposals,
            "decision_path": shared["decisions_path"],
            # step7 搬运（budget_adjust 消费方在上方 _apply_budget_adjust；
            # env_fixes 消费方在上方 env_fixed 事件；两者此前从 r7.data 读
            # 到 None 因为这段代码还没执行——消费代码在合成**前**跑，读
            # r7.data 是对的但 r7 可能是降级空壳——真正的 bug 是合成后的
            # AgentResult 的 data 丢失这些键，导致 _collect_adjust 的下游
            # （decisions 记录/humans 提案）看不到 step7 的产出）
            "budget_adjust": step7_data.get("budget_adjust"),
            "env_fixes": step7_data.get("env_fixes"),
            "env_proposals": step7_data.get("env_proposals"),
            "domain_knowledge_appended": step7_data.get(
                "domain_knowledge_appended"),
            "pipeline": {
                "mode": "pipeline",
                "steps": briefs,
                "degraded_steps": agg.get("degraded_steps", 0),
                "step_sessions": step_sessions,
            },
        }
        return _AR(
            role="adjust",
            valid=bool(agg["valid"]) and (r6 is not None and r6.valid),
            summary=str((r6.summary if r6 is not None else "") or
                        "adjust pipeline（预算耗尽降级——空提案）")[:500],
            data=data,
            wall_ms=int(agg["wall_ms"]),
            usage={"input_tokens": 0, "output_tokens": 0,
                   "total_tokens": int(agg["tokens"]),
                   "cost_usd": float(agg["cost_usd"])},
            tool_calls=[], truncated=bool(agg["truncated"]))

    # 预算热调白名单（step7 可调；human 限定的键在 super.json config
    # 的 human_pinned_budget 列表中声明——声明过的拒绝 step7 调节）
    _BUDGET_ADJUSTABLE = ("agent_timeout_s", "adjust_step_timeout_s",
                          "adjust_total_timeout_s", "max_tokens",
                          "agent_timeout_s_per_role", "adjust_interval",
                          "max_parallel_experiments", "max_wall_time_h")

    # adjust 自调上限（用户裁决 job_56：墙预算 7 天 / token 预算 1000M
    # 以内 adjust 自行调整——预算是运行资源不是实验变量，不该卡 human 审批；
    # 超上限才需 human 审批（提案队列）——防失控）。
    _BUDGET_CEILINGS = {"max_wall_time_h": 168.0,   # 7 天
                        "max_tokens": 1_000_000_000}  # 1000M

    def _writeback_config_key(self, key: str, value) -> None:
        """BUG-007 修复：热调后同步写回 super.json config 段（防 Go daemon
        重启时用 goSuperDefaults 内存缓存覆盖 py 热调值）。"""
        try:
            import json as _json
            p = os.path.join(self.ws, "super.json")
            d = _json.load(open(p, encoding="utf-8"))
            d.setdefault("config", {})[key] = value
            with open(p, "w", encoding="utf-8") as f:
                _json.dump(d, f, ensure_ascii=False, indent=2)
        except Exception:  # noqa: BLE001 写回失败不炸热调
            pass

    def _apply_budget_adjust(self, ba: Dict[str, Any],
                             iteration_id: int) -> Dict[str, Any]:
        """step7 预算热调（无人值守长程设计）：白名单键 → conf 即时生效。

        human 限定：super.json config.human_pinned_budget 键列表（human
        明确限定的预算不可被 step7 调节——用户裁决权高于自调节权）。
        返回实际应用的 {key: new_value}（被拒/非法键不返回）。
        """
        pinned = self.conf.get("human_pinned_budget") or []
        applied = {}
        for k, v in ba.items():
            if k not in self._BUDGET_ADJUSTABLE:
                # humans #32：非白名单键静默丢弃零留痕 → 观测事件
                self.events.append("budget_adjust_skipped", {
                    "iteration": iteration_id, "key": k,
                    "reason": "not_whitelisted（非白名单键）",
                }, "scheduler")
                continue
            if k in pinned:
                self.events.append("budget_adjust_rejected", {
                    "iteration": iteration_id, "key": k,
                    "reason": "human_pinned（human 限定键）",
                }, "scheduler")
                continue
            # 上限保护（用户裁决：7 天/1000M 内自调，超上限 human 审批）
            ceiling = self._BUDGET_CEILINGS.get(k)
            if ceiling is not None:
                try:
                    if float(v) > float(ceiling):
                        self.events.append("budget_adjust_rejected", {
                            "iteration": iteration_id, "key": k,
                            "value": v, "ceiling": ceiling,
                            "reason": "ceiling_exceeded（超 %s=%s 上限——"
                                      "需 human 审批，已入提案队列）" % (k, ceiling),
                        }, "scheduler")
                        try:
                            self.db._push_human("proposal", {
                                "kind": "budget_ceiling_exceeded",
                                "key": k, "value": v, "ceiling": ceiling,
                                "iteration": iteration_id,
                                "ask": "预算热调超上限（%s %s > %s）——是否批准？"
                                       % (k, v, ceiling),
                            })
                        except Exception:  # noqa: BLE001 提案失败不炸热调
                            pass
                        continue
                except (TypeError, ValueError):
                    continue
            try:
                v2 = type(self.conf.get(k) or v)(v)
                if v2 <= 0:
                    continue
                self.conf[k] = v2
                applied[k] = v2
                # A-09 修复：同步更新固化属性（主循环读 self 而非 conf）
                if k == "max_parallel_experiments":
                    self.max_parallel_experiments = max(1, int(v2))
                # 墙预算热调：同步 CostLedger 运行时生效 + budget 段写回
                # （config/budget 两层必须一致——_BUDGET_SYNC_KEY 契约）
                if k == "max_wall_time_h":
                    self.ledger.max_wall_time_h = float(v2)
                    self._writeback_budget_segment(v2)
                # BUG-007 修复：热调后写回 super.json（防 Go daemon 重启时
                # 用旧值覆盖——py 是 super.json 的合法写者之一[config 段]）
                self._writeback_config_key(k, v2)
            except (TypeError, ValueError):
                continue
        return applied

    def _writeback_budget_segment(self, wall_h: float) -> None:
        """墙预算热调：同步 super.json 顶层 budget 段（两层一致契约）。"""
        try:
            import json as _json
            p = os.path.join(self.ws, "super.json")
            d = _json.load(open(p, encoding="utf-8"))
            d.setdefault("budget", {})["max_wall_time_h"] = float(wall_h)
            with open(p, "w", encoding="utf-8") as f:
                _json.dump(d, f, ensure_ascii=False, indent=2)
        except Exception:  # noqa: BLE001 写回失败不炸热调
            pass

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
        # prompt 物证：legacy=prompt_adjust.md；pipeline=各步 prompt_adjust_*.md
        # （adjust_dir 内）——目录本身即快照根（prompt_path 指向目录）
        payload["prompt_path"] = os.path.join(adjust_dir, "prompt_adjust.md")
        if not os.path.isfile(payload["prompt_path"]):
            payload["prompt_path"] = adjust_dir
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
            iteration_id, proposals, bool(pending.valid),
            pipeline=(pending.data or {}).get("pipeline")
            if isinstance(pending.data, dict) else None)
        # A-04 修复：collect 成功后才推进 _last_adjust_iter（下一次 adjust
        # 的区间起点）——dispatch 时不推进（防多次 dispatch 竞争乱序）
        self._last_adjust_iter = iteration_id
        # adjusts 表（job_56 第三轮审计）：每次 adjust 运行一行——各步 pi
        # session_id + 迭代目录（行为轨迹观察入口；step3 三视角是数组）
        try:
            self.db.record_adjust(
                iteration_id, adjust_dir,
                sessions=((pending.data or {}).get("pipeline") or {}).get(
                    "step_sessions", {}),
                decisions_path=decision_path,
                agent_valid=bool(pending.valid))
        except Exception:  # noqa: BLE001 观察表失败不炸主链（事件已可诊断）
            pass
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
            method_updated = isinstance(method, str) and bool(method.strip())
            self.archive.set_description(target, merged)
            # task58/R4：modify_cell 更新了 method → 处方-基座相位差解除
            # （method 是描述基座的本体；只改 source_refs/name 不解除——
            # [STALE] 会持续提示 adjust 补上真正的方法更新）
            stale_cleared = False
            if method_updated:
                stale_cleared = self.archive.is_stale(target)
                self.archive.set_stale(target, False)
            self.events.append("cell_modified", {
                "cell_id": target,
                "method": str(merged.get("method") or "")[:400],
                "source_refs": merged.get("source_refs") or [],
                "rationale": str(p.get("rationale") or "")[:300],
                "rule_id": p.get("rule_id"),
                "stale_cleared": stale_cleared,
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
        descriptor/elite 时 agent 自己查 super.db cells 表（它的决策）。"""
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
        """profile 采集对象：该格精英代码 → 本题 naive 副本 → reference.py。

        humans #28 根修（题库模式跨格污染）：旧行为 elite-less 格回落
        env/code/naive_start.py（单题模式遗留文件——题库模式下注入的是
        13_Matmul 的 profile 读数，27/27 条 profile_history 实证）。
        题库模式改为本题 problems/level2/<题名>.py（与 gate_check 同源）。
        """
        elite = self._cell_entry(cell_id).get("elite") or {}
        cp = elite.get("code_path")
        if isinstance(cp, str) and cp and os.path.isfile(cp):
            return cp
        prob = self._cell_problem(cell_id)
        if prob:
            pq = os.path.join(self.env_dir, "problems", "level2",
                              "%s.py" % prob)
            if os.path.isfile(pq):
                return pq
        for name in ("naive_start.py", "reference.py"):
            p = os.path.join(self.env_dir, "code", name)
            if os.path.isfile(p):
                return p
        return None

    def _cell_problem(self, cell_id: str) -> Optional[str]:
        """cell_id → 题名（problem=<题名> 解析；非题库格返回 None）。"""
        m = re.match(r"^problem=(.+)$", cell_id or "")
        return m.group(1) if m else None

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
        # humans #28 根修：题库模式显式传 --problem（旧行为不带参——
        # 单题模式 profile 的是 reference.py 而非本格题目）
        cmd = [self._python_cmd(), script, "--candidate", candidate]
        prob = self._cell_problem(cell_id)
        if prob:
            cmd += ["--problem", prob]
        try:
            r = subprocess.run(cmd,
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
            # 读侧存在性校验（env_9 实测 bug：DB 重置/清理后 elite 路径
            # 指向已删文件——注入 ENOENT 路径误导 agent 以陈旧精英为父本）
            code_path = elite.get("code_path")
            if isinstance(code_path, str) and code_path \
                    and not os.path.exists(code_path):
                self.events.append("elite_path_missing", {
                    "cell_id": cid, "missing_path": code_path,
                    "downgrade": "elite_code_path 置空",
                }, "scheduler")
                code_path = None
            out.append({
                "cell_id": cid,
                "description": desc,
                "elite_code_path": code_path,
                "elite_perf": elite.get("perf_value"),
                "record_digest": self._or_default(
                    self._record_for_code_path(code_path,
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
        return ("（%d 次实验的 pi session 可用——按需 SQL 查 experiments 表"
                "（experiment_id/record_path/session_id 列）或 ls codes/ 目录"
                "定位工作目录，pi session 文件按 workdir 归档于"
                " ~/.rick/pi/agent/sessions/）" % n)

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
        细节靠 super.db cells 表自查）。

        task58/R4：method_stale 的格带 [STALE] 标注（处方-基座相位差警示）——
        adjust 看到 [STALE] → 用 modify_cell 更新 method（更新即清除标记）。
        """
        out = {}  # type: Dict[str, Any]
        for cid in self.fs.all_cells():
            entry = self._cell_entry(cid)
            stale = bool(entry.get("method_stale"))
            desc = entry.get("description")
            if not isinstance(desc, dict):
                if stale:
                    out[cid] = {"stale": True}
                else:
                    out[cid] = {}
                continue
            method = str(desc.get("method") or "")
            brief = method[:200] + ("…" if len(method) > 200 else "")
            if stale:
                # [STALE] 前缀：method 基座落后于当前精英——需 modify_cell 更新
                brief = "[STALE] " + brief
            out[cid] = {
                "name": desc.get("name") or "",
                "method_brief": brief,
                "source_refs": desc.get("source_refs") or [],
            }
            if stale:
                out[cid]["stale"] = True
        return out

    def _ensure_decision_file(self, iteration_id: int,
                              proposals: List[Any], valid: bool,
                              pipeline: Optional[Dict[str, Any]] = None) -> str:
        """决策记录保底（KR3/KR-F）：agent 未落盘 → 框架按提案合成最小记录。

        pipeline 模式（job_56 task3）：七节骨架 + 内嵌 schema JSON（从流水线
        briefs 提取——上游步骤产物已落盘，schema 块按步骤摘要合成）。
        返回 decisions 文件路径（存在即返回；不存在则合成后返回）。
        """
        path = self._decisions_path(iteration_id)
        if os.path.isfile(path):
            return path
        steps = ((pipeline or {}).get("steps")) or []
        def _step_brief(step_no: int, key: str, default: str = "（步骤降级/未产出）") -> str:
            for s in steps:
                if s.get("step") == step_no:
                    v = s.get(key)
                    return str(v) if v not in (None, "") else default
            return default
        judgment = {}
        for s in steps:
            if s.get("step") == 4:
                judgment = s.get("judgment") or {}
                break
        schema = {
            "primary_contradiction": {
                "statement": judgment.get("primary_contradiction")
                or "（框架合成——step4 未产出）",
                "logic_basis": _step_brief(4, "summary", ""),
                "fact_confidence": [],
            },
            "secondary_contradictions": judgment.get(
                "secondary_contradictions") or [],
            "perspectives_used": [
                {"viewpoint": p.get("perspective"), "source": "pipeline step2"}
                for p in steps if p.get("step") == 3
                for _ in [p]  # noqa: FURB118
            ][:0] or [
                {"viewpoint": str(c.get("perspective")),
                 "source": "pipeline step3"}
                for s in steps if s.get("step") == 3
                for c in (s.get("contradictions") or [])
            ],
            "critique_ranking": [
                {"contradiction": c.get("primary_contradiction"),
                 "logical_certainty": c.get("logic_basis") or "",
                 "rank": i + 1}
                for s in steps if s.get("step") == 3
                for i, c in enumerate(s.get("contradictions") or [])
            ],
            "cell_actions": [
                {"action": {"new_cell": "create", "soft_weight": "upweight"}.get(
                    str(p.get("type")), str(p.get("action") or "adjust")),
                 "cell": p.get("target_cell") or p.get("descriptor") or "",
                 "reason": str(p.get("evidence") or p.get("rationale") or "")[:300]}
                for p in proposals if isinstance(p, dict)
            ],
            "saturation_evidence_consumed": [],
        }
        lines = [
            "# adjust 决策记录 · 迭代 %d" % iteration_id, "",
            "> framework_synthesized: true"
            "（agent 未落盘决策记录，框架按提案与流水线步骤摘要合成）", "",
            "- agent_valid: %s" % bool(valid),
            "- mode: %s" % ((pipeline or {}).get("mode") or "legacy"), "",
            "## 1. 事实与期望（step1 摘要）", "",
            "- %s" % _step_brief(1, "summary"), "",
            "## 2. 视角（step2/step3 摘要）", "",
        ]
        for s in steps:
            if s.get("step") == 3:
                for i, c in enumerate(s.get("contradictions") or []):
                    lines.append("- 视角 %d：%s" % (
                        i + 1, str(c.get("perspective"))[:150]))
        lines += ["", "## 3. 矛盾（step3 摘要）", ""]
        for s in steps:
            if s.get("step") == 3:
                for i, c in enumerate(s.get("contradictions") or []):
                    lines.append("- 矛盾 %d：%s" % (
                        i + 1, str(c.get("primary_contradiction"))[:200]))
        lines += [
            "", "## 4. 判定（step4 摘要）", "",
            "- 主要矛盾：%s" % str(
                judgment.get("primary_contradiction") or "（降级）")[:300],
            "- 次要矛盾：%s" % str(
                judgment.get("secondary_contradictions") or [])[:300], "",
            "## 5. 逆转（step5 摘要）", "",
            "- %s" % _step_brief(5, "summary"), "",
            "## 6. 格动作", "",
        ]
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
        lines += ["", "## 7. 实验自改进（step7 摘要）", "",
                  "- %s" % _step_brief(7, "summary"), "",
                  "## schema（机器可读——adjust_trace/Step7 复盘消费）", "",
                  "```json",
                  json.dumps(schema, ensure_ascii=False, indent=2),
                  "```", ""]
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

    def _gpu_window_advisory(self, cell_id: str, cursor: int) -> None:
        """humans #43（advisory 版）：E2E 派发前 GPU 窗口拥挤提示。

        iter175 判例：8/8 满并发挤 5 机 H20 池 → E2E 三连 >3600s 全损。
        advisory 档（先观察再收紧）：本地信号（在飞实验数 vs 5 机池 +
        近窗 E2E 无读数/超时事件）拥挤判定 → log-only 事件（不阻塞不
        延迟派发）。远程 GPU 余量实读通道（env/skills/gpu_window.py）
        部署后接线升级 blocking 档（三证伪条款见 cell 40 v4.2）。
        """
        try:
            pool = int(self.conf.get("gpu_pool_size", 5) or 5)
            inflight = 0
            for ev in self.events.read_all()[-200:]:
                t = ev.get("type")
                if t == "experiment_started":
                    inflight += 1
                elif t == "experiment_finished":
                    inflight = max(0, inflight - 1)
            recent_no_reading = sum(
                1 for ev in self.events.read_all()[-50:]
                if ev.get("type") == "gate_result"
                and (ev.get("payload") or {}).get("score") == -6)
            crowded = inflight >= pool or recent_no_reading >= 2
            # 实读通道（humans #43）：gpu_window.py 远程探针（~10s 只读）——
            # advisory 档关窗读数也 log-only（先观察再收紧；升级 blocking 档
            # 的三证伪条款见 cell 40 v4.2）。探针失败/超时 → 本地信号兜底。
            window = None
            try:
                gw = os.path.join(self.env_dir, "skills", "gpu_window.py")
                if os.path.isfile(gw):
                    r = subprocess.run(
                        [self._python_cmd(), gw], capture_output=True,
                        text=True, timeout=45)
                    for line in reversed((r.stdout or "").splitlines()):
                        line = line.strip()
                        if line.startswith("{"):
                            try:
                                w = json.loads(line)
                                if isinstance(w, dict) and \
                                        "window_open" in w:
                                    window = w
                                    break
                            except ValueError:
                                continue
            except Exception:  # noqa: BLE001 探针失败零影响
                window = None
            window_closed = bool(window and not window.get("window_open"))
            if crowded or window_closed:
                self.events.append("gpu_window_advisory", {
                    "iteration_id": cursor, "cell_id": cell_id,
                    "inflight_experiments": inflight,
                    "gpu_pool_size": pool,
                    "recent_no_reading_gates": recent_no_reading,
                    "gpu_window_open": (None if window is None
                                        else bool(window.get("window_open"))),
                    "eligible_machines": ((window or {}).get(
                        "eligible_machines")
                        if window else None),
                    "mode": "advisory（humans #43：log-only 不阻塞——关窗"
                            "拥挤读数落档，升级 blocking 档经三证伪裁决）",
                }, "scheduler",
                    experiment_id=getattr(self, "_cur_exp_id", None))
        except Exception:  # noqa: BLE001 advisory 失败零影响
            pass

    def _same_code_as_elite(self, candidate: str, cell_id: str) -> bool:
        """humans #38/#36：候选与该格精英同码（MD5 客观判据）。"""
        try:
            old_hash = self.archive.get_code_hash(cell_id)
            if not old_hash:
                return False
            return _code_digest(candidate) == old_hash
        except Exception:  # noqa: BLE001
            return False

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
        # humans #36① 语义边界：keep_original 仅在**有效复验**（result.valid
        # =true，debug 真跑并明确判定）时是否决证据；复验失败/降级
        # （valid=false——agent 异常/无脚本）的缺省 keep_original 维持旧行为
        # （原候选照常入格——复验失败≠幻增益否决证据）。
        recheck_valid = bool(result.valid)
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
        return fixed, verdict, recheck_valid

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
        # ---- task58/R3：ref 漂移防线（测量分辨率告警，不硬拒）----
        # env_11 实测：ref_runtime_ms 跨会话摆动 4999→6028（+20.6%），大于
        # 头三名候选间 4.4% 差异 → 排序键分辨率不足。本迭代基准与该格精英
        # 入档基准偏差超阈 → 告警 + 本条 gate_result 标注 measurement_degraded
        # （消费方按 env tolerance 处置；不阻断入格）。
        perf = ((getattr(gr, "metrics", None) or {}).get("perf") or {})
        new_ref = perf.get("ref_value") if isinstance(perf, dict) else None
        old_ref = ((self._cell_entry(cell_id).get("elite") or {})
                   .get("ref_runtime_ms"))
        if isinstance(new_ref, (int, float)) and not isinstance(new_ref, bool) \
                and isinstance(old_ref, (int, float)) \
                and not isinstance(old_ref, bool) and float(old_ref) != 0.0:
            drift = abs(float(new_ref) - float(old_ref)) / abs(float(old_ref))
            threshold = float(self.conf.get("ref_drift_threshold", 0.15) or 0.15)
            if drift > threshold:
                payload["measurement_degraded"] = True
                self.events.append("ref_drift_warning", {
                    "iteration_id": cursor, "cell_id": cell_id,
                    "old_ref": float(old_ref), "new_ref": float(new_ref),
                    "drift_pct": round(drift, 4),
                    "threshold": threshold,
                }, "gate",
                    experiment_id=getattr(self, "_cur_exp_id", None))
        if recheck:
            payload["recheck"] = True
        self.events.append("gate_result", payload, "gate",
                           experiment_id=getattr(self, "_cur_exp_id", None))

    # ---- 饱和信号（task23 E 段） ----

    def _recover_saturation_state(self) -> None:
        """从事件流恢复历史最优 perf（崩溃重启后 stale 计数不从头来）。

        task59 R5：同帧恢复噪声带（最近一次 noise_calibrated 的 noise_pct）
        与标定迭代号（幂等防重——重启后同 cursor 不重标）。
        """
        best = None  # type: Optional[float]
        try:
            events = self.events.read_all()
        except (AttributeError, OSError):
            return
        for ev in events:
            if ev.get("type") not in ("elite_admitted", "elite_replaced"):
                if ev.get("type") == "noise_calibrated":
                    p = ev.get("payload") or {}
                    noise = p.get("noise_pct")
                    it = p.get("iteration_id")
                    if isinstance(noise, (int, float)) \
                            and not isinstance(noise, bool):
                        self._noise_band = float(noise)
                    if isinstance(it, int) and not isinstance(it, bool):
                        self._noise_calib_last = int(it)
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

        原三条件全满足（task23 E 段，保留不变）：
          - stale_iterations >= 3（自上次全局最优刷新）
          - all_visited：所有 weight>0 的格 visit >= 2
          - champion_misses >= 3（冠军格近 3 次采样未刷新）

        task59 R5 新增独立触发支（噪声带判据）：
          - noise_band > 0 且 top3_spread < noise_band（位次不可区分）
          - identified_headroom < noise_band（收益不可认证）
          → 载荷附 noise_band/top3_spread/identified_headroom +
            stop_reason="headroom_below_noise"（adjust 可据此提
            terminate_recommend——停止判据从「预算跑道」改锚「测量分辨率」）。
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
        original = (self._stale_iters >= 3 and all_visited
                    and self._champion_misses >= 3)
        noise = self._noise_saturation_evidence(snap)
        if not original and noise is None:
            return None
        return {
            "stale_iterations": int(self._stale_iters),
            "all_visited": all_visited,
            "champion_cell": self._champion_cell(),
            "best_perf": self._best_perf,
            "alive_cells": len(alive),
            **(noise or {}),
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
        """epoch 末 pending 重评估：M 次复测 → 保留/复活/出档。

        僵死循环修复（job_56 实测 1.5h 空转）：①老 pending 项 code_path
        是相对名（codes/N-xxx）——cwd 相对解析恒失败 → 秒级快败循环；
        ②item 缺字段 → invalidation append schema raise → 异常吞掉 →
        pending 未持久清空 → 每 tick 重入同批 → 死循环。修复：路径
        绝对化兜底 + per-item 防御（单项失败不炸整轮）+ drain 后立即
        save（持久清空防重入）+ 幂等标记（同 cursor 只跑一轮）。
        """
        self._probe_dbg("reeval", "enter")
        pending = self.archive.drain_pending()
        # drain 即持久化（防中途异常导致下 tick 重入同批——死循环根因）
        try:
            self.archive.save()
        except Exception:  # noqa: BLE001
            pass
        if not pending:
            return
        m = max(1, int(self.conf.get("debug_max_attempts") or 1))
        for item in pending:
            try:
                cid = item.get("cell_id") or ""
                code_path = item.get("code_path") or ""
                # 路径绝对化兜底（老存档相对名：codes/N-xxx → <ws>/codes/N-xxx）
                if code_path and not os.path.isabs(code_path):
                    for base in (self.codes_dir, self.ws):
                        cand = os.path.join(base, code_path)
                        if os.path.exists(cand):
                            code_path = cand
                            break
                if not cid or not code_path or not os.path.exists(code_path):
                    self.events.append("pending_item_skipped", {
                        "cell_id": cid, "code_path": str(item.get("code_path"))[:120],
                        "reason": "bad_item（缺字段/路径不可解析——防御跳过）",
                    }, "scheduler")
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
            except Exception as exc:  # noqa: BLE001 单项失败不炸整轮（防死循环）
                self.events.append("pending_item_error", {
                    "cell_id": str(item.get("cell_id") or "")[:80],
                    "error": str(exc)[:200],
                    "reason": "重评估单项异常（防御跳过——drain 已持久清空）",
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
        # job_56 task4 KR-E：pi session id 存入 by_role（实验链收尾写回
        # experiments 表 session_id 列——行为轨迹还原的入口）
        _sid = getattr(result, "session_id", "") or ""
        if _sid:
            acc["session_id"] = _sid
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

        # 迭代物证：agent final 落盘（humans #35：data 保真——事件 payload
        # 不带 data 防膨胀，落盘 JSON 带完整 data 供步骤级断点续跑保真重建）
        try:
            dump_payload = dict(payload)
            dump_payload["data"] = getattr(result, "data", None)
            with open(os.path.join(workdir, "agent_%s.json" % role), "w",
                      encoding="utf-8") as f:
                json.dump(dump_payload, f, ensure_ascii=False, indent=2)
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
        """SIGTERM/SIGINT → 优雅停止（杀在飞 pi agent + 落完当前事件再退）。

        env_9 实测修复：Go force-stop 的 SIGTERM→SIGKILL 梯中，py 无机会等
        分钟级 LLM 调用自然收尾——SIGTERM 时立即杀在飞 pi（killpg 全树），
        实验按中断收尾；残活 pi 会与下代调度器并发写工作区（多写者风险）。
        """
        def handler(signum, frame):  # noqa: ARG001
            self.stop_requested = True
            try:
                from .agents import PiAgentClient as _PAC
                _PAC.kill_all_inflight()
            except Exception:  # noqa: BLE001 杀不干净不阻停机
                pass
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
