"""events.py — super 调度器的唯一真相源：事件日志（job_47 task1 · job_56 task1 DB 化）

事件存储迁移到 sqlite（`<ws>/super.db` 的 events 表——job_56 四项重构裁决 Q1：
一次切净，events.jsonl 废弃；旧工作区不迁移不兼容）。py 调度器进程是
唯一写者（进程锁强保证，见 db.acquire_scheduler_lock），本模块不做进程锁。

设计契约（design-tree §2.2 / §2.7，task1 spec，DB 化后语义不变）：
  - 事件信封五字段：{ts, seq, type, source, payload}
    ts   RFC3339 带时区（domain/bugs.md 坑位：无时区会被 Go time.Parse 拒绝；
         项目惯例固定 +08:00，与 archive/config 一致）
    seq  事件表 AUTOINCREMENT 主键（单调递增，DB 保证唯一）
    type ∈ EVENT_TYPES（设计树 §2.2 定稿 19 类 + task19 adjust_decision）
    source ∈ {scheduler, gate, human, adjudicator, agent:<adjust|builder|coding|debug>}
    payload dict，必填字段见 EVENT_SCHEMAS
  - 写前校验：validate_event 是唯一裁决点（坏事件不落盘）
  - 事务性：sqlite WAL（崩溃安全：已提交事务保留，无半行概念——
    jsonl 时代的截断容错/封口逻辑随存储迁移退役）

迭代 10 / task49：**实验维度的并行写入**（human 裁决）
  - 「格子 + 迭代号」= 一次实验（experiment）的唯一标记：
    ``experiment_id = "<cell_safe>#iter<N>"``（cell_safe = 非字母数字 → ``_``），
    由 ``make_experiment_id(cell_id, iteration_id)`` 生成
    （注意：与 experiments 表的 PK「工作目录名」是两种 ID——见 db.py DDL 注释）
  - 实验内序号 ``experiment_seq`` 单调递增（DB MAX+1，跨重启自动续号）
  - 并行写安全：EventWriter 进程内 ``threading.Lock`` 保护 seq 预计算与
    插入的原子性；跨进程由调度器进程锁保证单写者

stdlib-only，Python >=3.9 兼容（禁 PEP604 X|Y 联合类型注解，用 Optional[...]）。
"""
import json
import os
import re
import threading
from datetime import datetime, timezone, timedelta
from typing import Any, Dict, List, Optional

from .db import SuperDB, db_for_ws

__all__ = [
    "EVENT_TYPES",
    "EVENT_SCHEMAS",
    "SOURCES",
    "AGENT_SOURCES",
    "EventWriter",
    "cell_safe",
    "make_experiment_id",
    "validate_event",
]

# 东八区（项目惯例，见 domain/bugs.md：时间戳必须带时区；与 archive.py 一致）
_TZ = timezone(timedelta(hours=8))

# ---- 事件类型常量集（设计树 §2.2 定稿 19 类 + task19 adjust_decision） ----
EVENT_TYPES = frozenset({
    "session_started",            # env/游标
    "session_ended",              # 游标
    "iteration_started",          # iteration_id/cell_id
    "iteration_ended",            # iteration_id/cell_id
    "cell_sampled",               # cell_id/visit/ucb_score/冷启动标记
    "agent_started",              # role/iteration_id
    "agent_ended",                # role/iteration_id/exit_code/wall_ms（成本账本）
    "candidate_produced",         # code_path/self_check 摘要
    "gate_result",                # stage/failure_stage/passed/metrics
    "elite_admitted",             # cell_id/code_path/perf_value/lineage/confidence
    "elite_replaced",             # 同上（同格替换）
    "weight_changed",             # cell_id/old_weight/new_weight/reason（adjust 软动作）
    "proposal",                   # proposal_id/type/证据引用（硬动作提案）
    "adjudicated",                # proposal_id/decision（外置裁决结果）
    "adjust_decision",            # iteration/proposals 摘要（task19：adjust 决策轨迹事件）
    "infra_failure_concluded",
    "infra_failure_recovered",
    "infra_hold_abandoned",    # 基础设施熔断：连续 N 次 agent 0-token 失败 →
    "gate_no_reading_retry",   # humans #27：verify 无读数失败（-6）→ 同候选重试 1 次
    "budget_adjust_skipped",   # humans #32：非白名单/被拒键留痕（观测缺口）
    "experiment_resumed",      # 无缝重启：中断实验断点续跑（coding/产物断点）
    "experiment_resumed_fallback",  # resume 失败回退正常链
    "pending_item_skipped",    # pending 重评估：坏项防御跳过（缺字段/路径不可解析）
    "pending_item_error",      # pending 重评估：单项异常（不炸整轮）
                                  # 优雅停机（认证/配额/供应商级故障，非实验失败）
    "cell_created",               # cell_id/descriptor/name/method（迭代8：adjust new_cell）
    "cell_modified",              # cell_id/method/source_refs/rationale（迭代8：modify_cell）
    "invalidation",               # cell_id/code_path/reason（出档）
    "resurrection",               # cell_id/code_path/reason（复活）
    "intervention",               # human 输入原文+分类（摄取自干预队列）
    "heartbeat",                  # 无必填载荷（信封 ts 即心跳时刻）
    "env_failure_concluded",      # 诊断包 returncode/stderr_tail/cause
    # ---- 迭代9/task45：格级互斥（一个格同时只有一个实验）----
    "cell_lock_resample",         # sampled_cell/locked_to（采样撞锁 → 异格兜底）
    "cell_locked_skip",           # sampled_cell/consecutive_skips（全格被占 → 跳过）
    "cell_lock_overflow",         # consecutive_skips（连续跳过超上限 → 优雅停）
    "cell_lock_stale_recovered",  # cell_id（超 TTL 锁回收）
    "cell_lock_orphans_released",  # count/cells（启动时死 pid 孤儿锁清理）
    "parallel_degraded",          # requested/effective/reason（并发槽降级为串行）
    # ---- 迭代10/task50：并发实验槽（一格 = 一个实验，异格并行）----
    "parallel_slots_enabled",     # max_parallel_experiments（槽池开启）
    "experiment_started",         # slot/iteration_id/cell_id（实验起跑）
    "experiment_finished",        # slot/iteration_id/cell_id/wall_ms（实验收尾）
    "experiment_failed",          # slot/iteration_id/cell_id/error（实验异常）
    # ---- 迭代11/task54：adjust 饥饿修复可观测 ----
    "adjust_dispatched",          # iteration/window_since（跨边界触发证据）
    "adjust_step_started",        # step/role/iteration_id（job_56 task3：流水线步骤起航）
    "adjust_step_ended",          # step/role/valid/wall_ms（步骤完成）
    "adjust_step_degraded",       # step/role/reason（单步失败降级——跳过+默认值继续）
    "env_proposal_queued",        # iteration/kind/title（step7 env 提案入 humans 队列）
    "budget_adjusted",            # iteration/applied（step7 预算热调生效——job_56）
    "budget_adjust_rejected",     # iteration/key/reason（human 限定键拒绝）
    "env_fixed",                  # iteration/fixes（step7 直接修复审计）
    "env_proposal_rejected",      # iteration/kind/reason（非 restart/verifier 类拒绝）
    "experiment_failed_confirmed", # iteration_id/cell_id/attempt/role/reason（job_56 task4：debug 主动上报实验失败——调度器跳过端到端验证）
    # ---- 迭代12/task56：软监控 + 断连 + 约束 ----
    "turn_budget_alert",          # role/consecutive/soft_warn（连续软告警）
    "api_outage_detected",
    "turn_budget_alert",
    "new_cell_rejected",
    "cell_limit_reached",
    "infra_failure_concluded",
    "self_check_skip_debug",      # iteration_id/reason（空 metrics 跳 debug）
    "cell_limit_reached",         # current_cells/max_cells/reason
    "new_cell_rejected",          # reason（空 method）
    # ---- 迭代9：基础设施熔断 ----
    "infra_failure_concluded",    # consecutive_zero_token_fails/diagnosis
    # ---- 迭代13/task57：熔断暂停的探活恢复 + 产物抢救 ----
    "api_outage_recovered",       # consecutive_infra/probes/diagnosis（探活成功 → 恢复迭代）
    "artifact_rescued",           # code_path/reason（agent 无 JSON 但产物落盘 → 抢救）
    # ---- 迭代13/task58：幻影 elite 防线 + 处方-基座相位差 ----
    "elite_measurement_refresh",  # cell_id/code_hash/new_perf/old_perf（同码重测 → 只刷测量值）
    "ref_drift_warning",          # old_ref/new_ref/drift_pct（基准漂移超阈 → 测量分辨率告警）
    # ---- 迭代13/task59：同码双测噪声标定（测量分辨率锚） ----
    "noise_calibrated",           # cell_id/current_perf/archived_perf/noise_pct（冠军精英重测 → 噪声带）
})

# 各事件类型 payload 的必填字段（设计树 §2.2「载荷要点」的最小契约化）。
# 空 tuple = 无必填字段（payload 为任意合法 JSON 对象即可）。
# 下游写事件方（loop.py）与校验方（gate）共用本表。
EVENT_SCHEMAS: Dict[str, tuple] = {
    "session_started": ("env",),
    "session_ended": ("iteration_cursor",),
    "iteration_started": ("iteration_id",),
    "iteration_ended": ("iteration_id",),
    "cell_sampled": ("cell_id", "visit"),
    "agent_started": ("role", "iteration_id"),
    "agent_ended": ("role", "iteration_id", "exit_code", "wall_ms"),
    "candidate_produced": ("code_path",),
    "gate_result": ("stage", "passed"),
    "elite_admitted": ("cell_id", "code_path", "perf_value"),
    "elite_replaced": ("cell_id", "code_path", "perf_value"),
    "weight_changed": ("cell_id", "old_weight", "new_weight", "reason"),
    "proposal": ("proposal_id", "type"),
    "adjudicated": ("proposal_id", "decision"),
    "adjust_decision": ("iteration", "proposals"),
    # 迭代9/task45：格级互斥事件（必填字段最小契约）
    "cell_lock_resample": ("sampled_cell", "locked_to"),
    "cell_locked_skip": ("sampled_cell", "consecutive_skips"),
    "cell_lock_overflow": ("consecutive_skips",),
    "cell_lock_stale_recovered": ("cell_id",),
    "cell_lock_orphans_released": ("count",),
    "parallel_slots_enabled": ("max_parallel_experiments",),
    "experiment_started": ("slot", "iteration_id", "cell_id"),
    "experiment_finished": ("slot", "iteration_id", "cell_id", "wall_ms"),
    "experiment_failed": ("slot", "iteration_id", "cell_id", "error"),
    # 迭代11/task54：adjust 跨边界触发可观测
    "adjust_dispatched": ("iteration", "window_since"),
    # job_56 task3：adjust 7 步流水线（步骤事件 + env 提案入队）
    "adjust_step_started": ("step", "role", "iteration_id"),
    "adjust_step_ended": ("step", "role", "valid", "wall_ms"),
    "adjust_step_degraded": ("step", "role", "reason"),
    "env_proposal_queued": ("iteration", "kind", "title"),
    "budget_adjusted": ("iteration",),
    "budget_adjust_rejected": ("iteration",),
    "env_fixed": ("iteration",),
    "env_proposal_rejected": ("iteration",),
    "experiment_failed_confirmed": ("iteration_id", "cell_id", "attempt",
                                     "role", "reason"),
    "parallel_degraded": ("requested", "effective", "reason"),
    "cell_created": ("cell_id", "descriptor"),
    "cell_modified": ("cell_id", "method"),
    "invalidation": ("cell_id", "reason"),
    "resurrection": ("cell_id", "reason"),
    "intervention": ("text", "category"),
    "heartbeat": (),
    "env_failure_concluded": ("returncode", "stderr_tail", "cause"),
    "api_outage_detected": ("consecutive_empty", "retry_s"),
    # task57 对齐：schema 必填字段必须与 loop.py 实际发射载荷一致——
    # env_11 实测 schema/载荷不一致会在事件触发时直接炸掉调度器
    # （invalid event: payload missing required field → ValueError 未捕获）。
    "api_outage_recovered": ("consecutive_infra", "diagnosis"),
    "artifact_rescued": ("code_path", "reason"),
    "self_check_skip_debug": ("reason",),
    "infra_failure_concluded": ("consecutive_zero_token_fails", "diagnosis"),
    "infra_failure_recovered": ("probes_streak", "diagnosis"),   # humans #26
    "infra_hold_abandoned": ("consecutive_probe_fails", "diagnosis"),  # humans #26
    "gate_no_reading_retry": ("iteration_id", "cell_id", "reason"),  # humans #27
    "budget_adjust_skipped": ("iteration", "key", "reason"),  # humans #32
    "experiment_resumed": ("iteration_id", "cell_id", "resumed_from",
                           "resume_role", "diagnosis"),
    "experiment_resumed_fallback": ("iteration_id", "cell_id", "reason"),
    "pending_item_skipped": ("cell_id", "code_path", "reason"),
    "pending_item_error": ("cell_id", "error", "reason"),
    "cell_limit_reached": ("current_cells", "max_cells"),
    "new_cell_rejected": ("reason",),
    "turn_budget_alert": ("role", "soft_warn"),
    # 迭代13/task58：幻影 elite 防线（同码重测）/ ref 漂移告警
    "elite_measurement_refresh": ("cell_id", "code_hash", "new_perf", "old_perf"),
    "ref_drift_warning": ("old_ref", "new_ref", "drift_pct"),
    # 迭代13/task59：同码双测噪声标定
    "noise_calibrated": ("cell_id", "current_perf", "archived_perf", "noise_pct"),
}

# 事件来源域（设计树 §2.2）
SOURCES = frozenset({"scheduler", "gate", "human", "adjudicator"})
AGENT_SOURCES = frozenset({
    "agent:adjust", "agent:builder", "agent:coding", "agent:debug",
})

# RFC3339 带时区：日期 T 时间（秒后小数可选）+ Z 或 ±HH:MM
_TS_RE = re.compile(
    r"^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}(\.\d+)?(Z|[+-]\d{2}:\d{2})$"
)

_ENVELOPE_KEYS = ("ts", "seq", "type", "source", "payload")
_OPTIONAL_KEYS = ("experiment_id", "experiment_seq")

# experiment_id 形态：<cell_safe 或角色名>#iter<N>（cell_safe 只含 [A-Za-z0-9_]）
_EXP_ID_RE = re.compile(r"^[A-Za-z0-9_]+#iter\d+$")


def cell_safe(cell_id: str) -> str:
    """cell_id → 可安全进 experiment_id / 文件名的形态（非字母数字 → _）。"""
    return re.sub(r"[^A-Za-z0-9]", "_", cell_id or "")


def make_experiment_id(cell_id: str, iteration_id: Any) -> str:
    """「格子 + 迭代号」= 一次实验的唯一标记（task49）。

    例：``make_experiment_id("impl=a|mem=b", 7)`` → ``"impl_a_mem_b#iter7"``。
    adjust 等非实验事件也可用 ``"adjust#iter7"`` 这类角色 id（同一形态）。
    """
    return "%s#iter%s" % (cell_safe(cell_id), iteration_id)

# last_seq 反向扫描的初始窗口（字节），不足时按 8 倍扩到全文件
_SCAN_WINDOW = 4096


def _now() -> str:
    """RFC3339 带时区时间戳（+08:00，秒精度；排序靠 seq，ts 为人读信息）。"""
    return datetime.now(_TZ).isoformat(timespec="seconds")


def _is_int(value: Any) -> bool:
    """真 int（排除 bool——bool 是 int 子类）。"""
    return isinstance(value, int) and not isinstance(value, bool)


def validate_event(ev: Any) -> List[str]:
    """校验一条事件，返回违规说明列表（空列表 = 合法）。

    检查项：
      1. 事件必须是 JSON object（dict）
      2. 信封五字段齐全：ts / seq / type / source / payload
      3. ts 是带时区的 RFC3339 字符串（Z 或 ±HH:MM；裸时间戳是已知坑）
      4. seq 是非负 int（bool 不算）
      5. type ∈ EVENT_TYPES
      6. source ∈ SOURCES ∪ AGENT_SOURCES
      7. payload 是 dict 且含 EVENT_SCHEMAS[type] 全部必填字段
         （额外字段允许——载荷要点是最小契约，不是封闭 schema）
    """
    if not isinstance(ev, dict):
        return ["event must be a JSON object, got %s" % type(ev).__name__]
    errors: List[str] = []
    missing = [k for k in _ENVELOPE_KEYS if k not in ev]
    if missing:
        return ["missing envelope key %r" % k for k in missing]

    ts = ev["ts"]
    if not isinstance(ts, str) or not _TS_RE.match(ts):
        errors.append(
            "ts must be RFC3339 with timezone offset (Z or +HH:MM), got %r" % (ts,))

    seq = ev["seq"]
    if not _is_int(seq) or seq < 0:
        errors.append("seq must be a non-negative int, got %r" % (seq,))

    etype = ev["type"]
    if not isinstance(etype, str) or etype not in EVENT_TYPES:
        errors.append(
            "type must be one of EVENT_TYPES (%d types), got %r"
            % (len(EVENT_TYPES), etype))

    source = ev["source"]
    if not isinstance(source, str) or not (source in SOURCES or source in AGENT_SOURCES):
        errors.append(
            "source must be scheduler/gate/human/adjudicator or "
            "'agent:<adjust|builder|coding|debug>', got %r" % (source,))

    payload = ev["payload"]
    if not isinstance(payload, dict):
        errors.append(
            "payload must be a JSON object, got %s" % type(payload).__name__)
        return errors
    if isinstance(etype, str) and etype in EVENT_SCHEMAS:
        for field in EVENT_SCHEMAS[etype]:
            if field not in payload:
                errors.append(
                    "payload of %r missing required field %r" % (etype, field))

    # ---- 实验维度（task49，可选字段）----
    eid = ev.get("experiment_id")
    eseq = ev.get("experiment_seq")
    if eid is not None:
        if not isinstance(eid, str) or not _EXP_ID_RE.match(eid):
            errors.append(
                "experiment_id must match '<name>#iter<N>', got %r" % (eid,))
        if eseq is None or not _is_int(eseq) or eseq < 1:
            errors.append(
                "experiment_seq must be a positive int when experiment_id "
                "is present, got %r" % (eseq,))
    elif eseq is not None:
        errors.append("experiment_seq requires experiment_id")
    return errors


class EventWriter:
    """事件日志写入器（DB 后端：super.db 的 events 表；单写者由进程锁保证）。

    用法（path=工作区目录，定位 super.db）::

        w = EventWriter(ws_dir)
        seq = w.append("iteration_started", {"iteration_id": 1}, "scheduler")
        ...
        w.close()   # 或 with EventWriter(ws) as w: ...（幂等 no-op）

    崩溃安全：sqlite WAL——已提交事务保留，无半行概念。

    写前校验：append 内部先构造完整事件（seq 预计算）→ validate_event
    （坏事件不落盘）→ INSERT。进程内 _lock 保证 seq 预计算与插入间
    无并发插入（跨进程由调度器进程锁保证单写者）。
    """

    def __init__(self, path: str, event_fsync: bool = True) -> None:
        # path = 工作区目录（定位 super.db）。event_fsync 参数保留但忽略
        # （DB WAL + synchronous=NORMAL 承担持久化语义）。
        self._path = path
        self._ws = os.path.abspath(path)
        self._db = db_for_ws(self._ws)
        self._closed = False
        # 线程锁：**复用 DB 实例锁**（同连接的全部访问互斥——EventWriter 内
        # 的 MAX 查询绕过 db._lock 会与 Archive/db 操作并发 execute 同一
        # 连接，sqlite C 层竞态可致段错误；单锁序列化根治）
        self._lock = self._db._lock

    # ---- 写 ----

    def append(self, type: str, payload: dict, source: str,
               experiment_id: Optional[str] = None,
               fsync: Optional[bool] = None) -> int:
        """追加一条事件，返回其 seq。

        ``experiment_id``（task49）：给定时自动分配该实验的 ``experiment_seq``
        （DB MAX+1，跨重启自动续号）；为 None 时该事件是「运维/全局」事件
        （session/heartbeat/intervention 等），不带实验字段。

        写前用 validate_event 校验：schema 违规抛 ValueError，坏事件不落盘。
        线程安全（心跳线程/并发实验同进程写）。
        """
        if self._closed:
            raise ValueError("EventWriter is closed")
        with self._lock:
            seq = self._db.last_seq() + 1
            ev = {
                "ts": _now(),
                "seq": seq,
                "type": type,
                "source": source,
                "payload": payload,
            }
            if experiment_id is not None:
                if not isinstance(experiment_id, str) or not experiment_id:
                    raise ValueError(
                        "experiment_id must be a non-empty str, got %r"
                        % (experiment_id,))
                row = self._db._conn.execute(
                    "SELECT MAX(experiment_seq) FROM events WHERE experiment_id=?",
                    (experiment_id,)).fetchone()
                base = row[0] if row and row[0] is not None else 0
                ev["experiment_id"] = experiment_id
                ev["experiment_seq"] = base + 1
            errors = validate_event(ev)
            if errors:
                raise ValueError("invalid event: " + "; ".join(errors))
            stored = self._db.append_event(
                ev["type"], ev["source"], ev["payload"],
                experiment_id=ev.get("experiment_id"),
                experiment_seq=ev.get("experiment_seq"),
                ts=ev["ts"])
        return int(stored["seq"])

    # ---- 读（重放/游标） ----

    def read_all(self) -> List[Dict[str, Any]]:
        """全量读取（重放投影用）。只读幂等。"""
        return self._db.read_events()

    def read_experiment(self, experiment_id: str) -> List[Dict[str, Any]]:
        """还原一次实验（格+迭代号）的完整事件流（experiment_seq 升序）。"""
        return self._db.read_experiment_events(experiment_id)

    def experiment_ids(self) -> List[str]:
        """出现过的全部 experiment_id（去重、按首次出现顺序）。"""
        return self._db.experiment_ids()

    def last_seq(self) -> int:
        """最后一条事件的 seq（DB 为准）。"""
        return self._db.last_seq()

    # ---- 生命周期 ----

    def close(self) -> None:
        """关闭（幂等）。DB 连接由单例管理（进程退出统一释放）——本方法
        仅置 closed 标志，防 close 后继续 append。"""
        self._closed = True

    def __enter__(self) -> "EventWriter":
        return self

    def __exit__(self, exc_type, exc, tb) -> None:
        self.close()
