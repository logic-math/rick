"""events.py — super 调度器的唯一真相源：append-only 事件日志（job_47 task1）

events.jsonl 一行一事件（紧凑 JSON），py 调度器进程是唯一写者（单写者架构，
人工干预经干预队列由调度器代写），本模块不做进程锁。

设计契约（design-tree §2.2 / §2.7，task1 spec）：
  - 事件信封五字段：{ts, seq, type, source, payload}
    ts   RFC3339 带时区（domain/bugs.md 坑位：无时区会被 Go time.Parse 拒绝；
         项目惯例固定 +08:00，与 archive/config 一致）
    seq  进程内尽力递增，从 1 起；重开（模拟重启）从「既有最大 seq+1」续号
    type ∈ EVENT_TYPES（设计树 §2.2 定稿 19 类 + task19 adjust_decision）
    source ∈ {scheduler, gate, human, adjudicator, agent:<adjust|builder|coding|debug>}
    payload dict，必填字段见 EVENT_SCHEMAS
  - 每次 append 后 flush（+ fsync，可关：见下）
  - 截断容错：读取端跳过坏行/半行；重开时若文件尾部是半写行（无换行结尾），
    先补一个换行符「封口」——恢复后的新事件不与残行粘连成坏行
  - append-only 硬约束：只追加，绝不读改写已有字节

迭代 10 / task49：**实验维度的并行写入**（human 裁决）
  - 「格子 + 迭代号」= 一次实验（experiment）的唯一标记：
    ``experiment_id = "<cell_safe>#iter<N>"``（cell_safe = 非字母数字 → ``_``），
    由 ``make_experiment_id(cell_id, iteration_id)`` 生成
  - 实验内序号 ``experiment_seq`` 单调递增（1 起，重开时从既有事件扫描恢复）
  - **seq 语义降级声明**：``seq`` 仅在单写者进程内尽力递增；**并行写入时不再
    保证全局唯一/单调**。定位与还原请用 ``(experiment_id, experiment_seq)``
    （或直接 ``read_experiment(experiment_id)``）；全局时间顺序看 ``ts``。
    既有消费方：gates.py / adjust_trace.py / cost_report.py 只读 payload（不受
    影响）；lineage.project_state 的游标已改为「已完成迭代的 max iteration_id」，
    不依赖全局 seq 单调。
  - **并行写安全**：一行一次 ``write``（O_APPEND 下 POSIX 单行 <4KB 原子）；
    进程内仍用 ``threading.Lock`` 保护多线程 append 的 experiment_seq 计数
  - ``event_fsync`` 实例开关（默认 True）：高频实验事件可关 fsync，退出时统一
    flush（``close()`` 已 flush；崩溃时未 fsync 的尾行可能丢——由截断容错容忍）

validate_event 是事件 schema 的唯一裁决点：EventWriter.append 写前校验
（违规抛 ValueError，坏事件不落盘），测试与层门禁（gate）复用同一函数。

stdlib-only，Python >=3.9 兼容（禁 PEP604 X|Y 联合类型注解，用 Optional[...]）。
"""
import json
import os
import re
import threading
from datetime import datetime, timezone, timedelta
from typing import Any, Dict, List, Optional

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
    "infra_failure_concluded",    # 基础设施熔断：连续 N 次 agent 0-token 失败 →
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
    "cell_limit_reached": ("current_cells", "max_cells"),
    "new_cell_rejected": ("reason",),
    "turn_budget_alert": ("role", "soft_warn"),
    # 迭代13/task58：幻影 elite 防线（同码重测）/ ref 漂移告警
    "elite_measurement_refresh": ("cell_id", "code_hash", "new_perf", "old_perf"),
    "ref_drift_warning": ("old_ref", "new_ref", "drift_pct"),
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


def _scan_last_seq(path: str) -> int:
    """从文件尾部反向找最后一条完整（以换行结尾且可解析）事件的 seq。

    空文件 / 文件不存在 / 无完整行 → 0。尾部半行（崩溃残留）自然被忽略；
    窗口从 4KB 起、按 8 倍扩到全文件（事件行数量级在 KB 级，首窗即命中）。
    """
    try:
        f = open(path, "rb")
    except FileNotFoundError:
        return 0
    with f:
        f.seek(0, os.SEEK_END)
        size = f.tell()
        window = _SCAN_WINDOW
        while True:
            start = max(0, size - window)
            f.seek(start)
            tail = f.read()
            lines = tail.split(b"\n")
            # split 后最后一段若非空 = 无换行结尾的半行，丢弃；
            # 其余各段均以 \n 结尾 = 完整行候选。窗口首段可能是被截断的
            # 半行——从后往前解析时最后才轮到它，且半行 JSON 不可解析。
            for raw in reversed(lines[:-1]):
                if not raw.strip():
                    continue
                try:
                    ev = json.loads(raw.decode("utf-8"))
                except (ValueError, UnicodeDecodeError):
                    continue
                if isinstance(ev, dict) and _is_int(ev.get("seq")):
                    return ev["seq"]
            if start == 0:
                return 0
            window *= 8


def _iter_file_events(path: str):
    """逐行读事件文件（坏行/半行跳过，不抛异常）。文件不存在 → 无输出。

    读取端容错：追加中的半行（无换行）与非法 JSON 行都被跳过；
    并行写入下也只影响正在写的那一行。
    """
    try:
        f = open(path, "r", encoding="utf-8")
    except FileNotFoundError:
        return
    with f:
        for line in f:
            stripped = line.strip()
            if not stripped:
                continue
            try:
                ev = json.loads(stripped)
            except ValueError:
                continue
            if isinstance(ev, dict):
                yield ev


def _scan_experiment_seqs(path: str) -> Dict[str, int]:
    """扫描既有事件，恢复每个 experiment_id 的当前最大 experiment_seq。

    task49：实验内序号必须跨重启继续（同一次实验不会因进程重启而重号）。
    """
    out = {}  # type: Dict[str, int]
    for ev in _iter_file_events(path):
        eid = ev.get("experiment_id")
        seq = ev.get("experiment_seq")
        if not isinstance(eid, str) or not _is_int(seq):
            continue
        if seq > out.get(eid, 0):
            out[eid] = seq
    return out


class EventWriter:
    """append-only 事件日志写入器（单写者，不做进程锁——架构保证）。

    用法::

        w = EventWriter(ws_dir + "/events.jsonl")
        seq = w.append("iteration_started", {"iteration_id": 1}, "scheduler")
        ...
        w.close()   # 或 with EventWriter(path) as w: ...

    崩溃安全（design-tree §2.7）：
      - 每次 append：校验 → 写一行（紧凑 JSON + \\n）→ flush → fsync → 返回 seq；
      - __init__ 重开时若文件尾部是半写行（无换行结尾），先补一个换行符封口
        （这一字节是追加而非改写），再从既有最大 seq+1 续号；
      - 崩溃重启后由调用方用 read_all()/last_seq() 做游标重放投影。
    """

    def __init__(self, path: str, event_fsync: bool = True) -> None:
        self._path = path
        self._closed = False
        self._fh = open(path, "a", encoding="utf-8")
        self._seal_truncated_tail()
        self._next_seq = _scan_last_seq(path) + 1
        # task49：实验内序号计数（跨重启从既有事件恢复，同实验不重号）
        self._exp_seq = _scan_experiment_seqs(path)  # type: Dict[str, int]
        # fsync 开关（默认 True 保守；高频实验事件可关，由 close/flush 兼容）
        self._event_fsync = bool(event_fsync)
        # 线程锁：调度器心跳线程与主循环同进程并发 append（跨进程仍是单写者架构保证）
        self._lock = threading.Lock()

    # ---- 写 ----

    def append(self, type: str, payload: dict, source: str,
               experiment_id: Optional[str] = None,
               fsync: Optional[bool] = None) -> int:
        """追加一条事件，返回其 seq。

        ``experiment_id``（task49）：给定时自动分配该实验的 ``experiment_seq``
        （1 起、单调、跨重启续号）；为 None 时该事件是「运维/全局」事件
        （session/heartbeat/intervention 等），不带实验字段。

        ``fsync``：None → 用实例默认（``event_fsync``）；True/False 覆盖。

        写前用 validate_event 校验：schema 违规抛 ValueError，坏事件不落盘。
        线程安全（心跳线程/并发实验同进程写）；**seq 仅进程内尽力递增**，
        并行场景请用 (experiment_id, experiment_seq) 定位事件流
        （见模块 docstring 的 seq 语义降级声明）。
        """
        if self._closed:
            raise ValueError("EventWriter is closed")
        do_fsync = self._event_fsync if fsync is None else bool(fsync)
        with self._lock:
            seq = self._next_seq
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
                eseq = self._exp_seq.get(experiment_id, 0) + 1
                ev["experiment_id"] = experiment_id
                ev["experiment_seq"] = eseq
            errors = validate_event(ev)
            if errors:
                raise ValueError("invalid event: " + "; ".join(errors))
            line = json.dumps(ev, ensure_ascii=False, separators=(",", ":"))
            # 单次 write 一行：O_APPEND 下 POSIX 单行 <4KB 写入原子
            # （并行实验各写各的，不会互相截断成坏行）
            self._fh.write(line + "\n")
            self._fh.flush()
            if do_fsync:
                os.fsync(self._fh.fileno())
            self._next_seq = seq + 1
            if experiment_id is not None:
                self._exp_seq[experiment_id] = ev["experiment_seq"]
        return seq

    # ---- 读（重放/游标） ----

    def read_all(self) -> List[Dict[str, Any]]:
        """全量读取（重放投影用）。只读幂等；坏行/半行跳过不抛异常。"""
        return list(_iter_file_events(self._path))

    def read_experiment(self, experiment_id: str) -> List[Dict[str, Any]]:
        """还原一次实验（格+迭代号）的完整事件流（task49）。

        按 ``experiment_seq`` 升序返回该实验的事件；未知实验 → 空列表。
        等价于 ``grep '"experiment_id":"<id>"' events.jsonl`` 后按实验内
        序号排序。
        """
        if not isinstance(experiment_id, str) or not experiment_id:
            return []
        out = [ev for ev in _iter_file_events(self._path)
               if ev.get("experiment_id") == experiment_id]
        out.sort(key=lambda ev: ev.get("experiment_seq") or 0)
        return out

    def experiment_ids(self) -> List[str]:
        """文件中出现过的全部 experiment_id（去重、按首次出现顺序）。"""
        seen = []  # type: List[str]
        known = set()
        for ev in _iter_file_events(self._path):
            eid = ev.get("experiment_id")
            if isinstance(eid, str) and eid not in known:
                known.add(eid)
                seen.append(eid)
        return seen

    def last_seq(self) -> int:
        """最后一条完整事件的 seq（崩溃恢复游标）。以文件为准（非内存计数）。"""
        return _scan_last_seq(self._path)

    # ---- 生命周期 ----

    def close(self) -> None:
        """关闭写句柄（幂等）。关前 flush（event_fsync=False 时也保证已入 OS 缓冲）。"""
        if not self._closed:
            try:
                self._fh.flush()
            except (OSError, ValueError):
                pass
            self._fh.close()
            self._closed = True

    def __enter__(self) -> "EventWriter":
        return self

    def __exit__(self, exc_type, exc, tb) -> None:
        self.close()

    # ---- 内部 ----

    def _seal_truncated_tail(self) -> None:
        """文件非空且不以换行结尾（崩溃半写行）→ 追加一个换行符封口。

        封口保证恢复后的新事件从新行开始，不与残行粘连成不可解析的坏行
        （否则重开后的第一条事件会被残行吞掉）。只追加一个字节，不改写。
        """
        try:
            with open(self._path, "rb") as f:
                f.seek(0, os.SEEK_END)
                size = f.tell()
                if size == 0:
                    return
                f.seek(size - 1)
                last = f.read(1)
        except FileNotFoundError:
            return
        if last != b"\n":
            self._fh.write("\n")
            self._fh.flush()
            os.fsync(self._fh.fileno())
