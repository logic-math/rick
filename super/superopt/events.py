"""events.py — super 调度器的唯一真相源：append-only 事件日志（job_47 task1）

events.jsonl 一行一事件（紧凑 JSON），py 调度器进程是唯一写者（单写者架构，
人工干预经干预队列由调度器代写），本模块不做进程锁。

设计契约（design-tree §2.2 / §2.7，task1 spec）：
  - 事件信封五字段：{ts, seq, type, source, payload}
    ts   RFC3339 带时区（domain/bugs.md 坑位：无时区会被 Go time.Parse 拒绝；
         项目惯例固定 +08:00，与 archive/config 一致）
    seq  单调递增，从 1 起；重开（模拟重启）从「既有最大 seq+1」续号不回退
    type ∈ EVENT_TYPES（设计树 §2.2 定稿 19 类 + task19 adjust_decision）
    source ∈ {scheduler, gate, human, adjudicator, agent:<adjust|builder|coding|debug>}
    payload dict，必填字段见 EVENT_SCHEMAS
  - 每次 append 后 flush + fsync（崩溃安全：返回即已落盘）
  - 截断容错：读取端跳过坏行/半行；重开时若文件尾部是半写行（无换行结尾），
    先补一个换行符「封口」——恢复后的新事件不与残行粘连成坏行
  - append-only 硬约束：只追加，绝不读改写已有字节

validate_event 是事件 schema 的唯一裁决点：EventWriter.append 写前校验
（违规抛 ValueError，坏事件不落盘），测试与层门禁（gate）复用同一函数。

与并行层（task2 archive / task3 sampler / task4 config·lineage）零 import 耦合：
lineage.py 以字面量消费事件类型名，两边以设计树 §2.2 为对齐事实源。

stdlib-only，Python >=3.9 兼容（禁 PEP604 X|Y 联合类型注解，用 Optional[...]）。
"""
import json
import os
import re
import threading
from datetime import datetime, timezone, timedelta
from typing import Any, Dict, List

__all__ = [
    "EVENT_TYPES",
    "EVENT_SCHEMAS",
    "SOURCES",
    "AGENT_SOURCES",
    "EventWriter",
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
    "cell_created",               # cell_id/descriptor/name/method（迭代8：adjust new_cell）
    "cell_modified",              # cell_id/method/source_refs/rationale（迭代8：modify_cell）
    "invalidation",               # cell_id/code_path/reason（出档）
    "resurrection",               # cell_id/code_path/reason（复活）
    "intervention",               # human 输入原文+分类（摄取自干预队列）
    "heartbeat",                  # 无必填载荷（信封 ts 即心跳时刻）
    "env_failure_concluded",      # 诊断包 returncode/stderr_tail/cause
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
    "cell_created": ("cell_id", "descriptor"),
    "cell_modified": ("cell_id", "method"),
    "invalidation": ("cell_id", "reason"),
    "resurrection": ("cell_id", "reason"),
    "intervention": ("text", "category"),
    "heartbeat": (),
    "env_failure_concluded": ("returncode", "stderr_tail", "cause"),
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

    def __init__(self, path: str) -> None:
        self._path = path
        self._closed = False
        self._fh = open(path, "a", encoding="utf-8")
        self._seal_truncated_tail()
        self._next_seq = _scan_last_seq(path) + 1
        # 线程锁：调度器心跳线程与主循环同进程并发 append（跨进程仍是单写者架构保证）
        self._lock = threading.Lock()

    # ---- 写 ----

    def append(self, type: str, payload: dict, source: str) -> int:
        """追加一条事件，返回其 seq。

        写前用 validate_event 校验：schema 违规抛 ValueError，坏事件不落盘
        （单写者自己写的事件必须当场合规，契约违例要在写入点暴露）。
        写后 flush + fsync——返回即已落盘。
        """
        if self._closed:
            raise ValueError("EventWriter is closed")
        with self._lock:
            seq = self._next_seq
            ev = {
                "ts": _now(),
                "seq": seq,
                "type": type,
                "source": source,
                "payload": payload,
            }
            errors = validate_event(ev)
            if errors:
                raise ValueError("invalid event: " + "; ".join(errors))
            line = json.dumps(ev, ensure_ascii=False, separators=(",", ":"))
            self._fh.write(line + "\n")
            self._fh.flush()
            os.fsync(self._fh.fileno())
            self._next_seq = seq + 1
        return seq

    # ---- 读（重放/游标） ----

    def read_all(self) -> List[Dict[str, Any]]:
        """全量读取（重放投影用）。只读幂等；坏行/半行跳过不抛异常。"""
        events: List[Dict[str, Any]] = []
        try:
            f = open(self._path, "r", encoding="utf-8")
        except FileNotFoundError:
            return []
        with f:
            for line in f:
                stripped = line.strip()
                if not stripped:
                    continue
                try:
                    events.append(json.loads(stripped))
                except ValueError:
                    continue  # 坏行/半行：截断容错，跳过
        return events

    def last_seq(self) -> int:
        """最后一条完整事件的 seq（崩溃恢复游标）。以文件为准（非内存计数）。"""
        return _scan_last_seq(self._path)

    # ---- 生命周期 ----

    def close(self) -> None:
        """关闭写句柄（幂等）。"""
        if not self._closed:
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
