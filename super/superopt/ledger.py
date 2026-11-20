"""ledger.py — 实验台账（job_47/task26）

env_8 实证：coding agent 自发在迭代目录写 `ab_notes.md`，补系统缺失的
「实验档案」层（idea → 实现 → 门禁结果 → 结论，跨迭代不可查）。本模块把
这一层做成系统能力：**跨迭代可查的实验台账**，供四角色 ctx 消费
（builder 避免重复失败方向 / coding 参考同类教训 / debug 借鉴修复经验 /
adjust 回看自己的历史决策与效果）。

契约：
  - 存 `<ws>/super.db` 的 ledger_entries 内部表（job_56 task1 DB 化：
  jsonl 废弃；py 调度器单写者）。
  - 条目 schema（loop 侧填充，本模块容错消费）::

        {ts, iteration_id, cell_id, idea, parents, plan_steps_count,
         gate: {passed, failure_stage, perf_value},
         verdict: "admit"|"replace"|"reject"|"no_candidate",
         perf_delta, reject_reason,
         agents: {role: {wall_ms, tokens, truncated}}}

  - **perf 语义假设（低优）**：台账的 perf_value 为 runtime_ms 语义（越小越好），
    perf_timeline 的 is_record 判定按低优比较；本模块不引入 perf_desc 依赖
    （当前 super env 全为 low；未来若出现高优任务需扩展为按 perf_desc 判定）。
  - 读取端坏行/半行跳过（截断容错，与 events.py / archive.py 一致）。
  - 台账是**分析副产品**（真相源仍是 events.jsonl）：写失败不炸调用方
    （追加 IO 异常静默吞掉），非法条目/非法字段跳过或字符串化——主循环的
    健壮性优先于台账完整性（缺失可从事件流重建）。

与 loop/agents 零 import 耦合：本模块只吃 dict、吐 dict。

stdlib-only，Python >=3.9 兼容（禁 PEP604 X|Y 联合类型注解，用 Optional[...]）。
"""
import json
import os
from datetime import datetime, timezone, timedelta
from typing import Any, Dict, List, Optional

__all__ = ["Ledger"]

# 东八区（项目惯例，见 domain/bugs.md：时间戳必须带时区；与 archive/events 一致）
_TZ = timezone(timedelta(hours=8))

# brief() 里 idea 的截断长度（prompt 注入时的上下文预算）
_IDEA_LIMIT = 120


def _now() -> str:
    """RFC3339 带时区时间戳（+08:00）。"""
    return datetime.now(_TZ).isoformat(timespec="seconds")


def _num(value: Any) -> Optional[float]:
    """数值提取（排除 bool——True/False 不是 perf 数据）。"""
    if isinstance(value, (int, float)) and not isinstance(value, bool):
        return float(value)
    return None


def _perf_of(entry: Dict[str, Any]) -> Optional[float]:
    """条目里的 perf_value：优先 gate.perf_value，回退顶层 perf_value。"""
    gate = entry.get("gate")
    if isinstance(gate, dict):
        v = _num(gate.get("perf_value"))
        if v is not None:
            return v
    return _num(entry.get("perf_value"))


def _idea_text(entry: Dict[str, Any]) -> str:
    """idea 的人读文本（非 str 字符串化，超长截断）。"""
    idea = entry.get("idea")
    if idea is None:
        return ""
    s = idea if isinstance(idea, str) else str(idea)
    return s[:_IDEA_LIMIT]


class Ledger:
    """跨迭代实验台账（append-only JSONL）。

    用法::

        lg = Ledger(ws)
        lg.append({"iteration_id": 3, "cell_id": "a=x", "idea": "...", ...})
        lg.brief(8)          # 近 8 条精简摘要（prompt 注入）
        lg.perf_timeline(10) # 性能时间线（含 is_record 判定）
        lg.own_decisions("adjust", 5)  # 某角色的历史决策+台账内效果
    """

    def __init__(self, ws: str) -> None:
        self._ws = ws
        # DB 化（job_56 task1）：台账存 super.db 的 ledger_entries 内部表
        from .db import db_for_ws
        self._db = db_for_ws(ws)

    @property
    def path(self) -> str:
        """台账存储路径（super.db——兼容旧属性名，测试/调试用）。"""
        from .db import db_path_for_ws
        return db_path_for_ws(self._ws)

    # ---- 写 ----

    def append(self, entry: Dict[str, Any]) -> None:
        """追加一条台账（容错：非法条目跳过、非法字段字符串化、异常不炸）。

        不修改调用方传入的 dict（复制后补 ts）。
        """
        if not isinstance(entry, dict):
            return  # 非法条目：跳过（不炸调用方）
        item = dict(entry)
        item.setdefault("ts", _now())
        try:
            # DB 行总是合法 JSON（default=str 兜底不可序列化字段）
            json.dumps(item, ensure_ascii=False, default=str)
        except (TypeError, ValueError):
            return  # 极端不可序列化情形：跳过该条（台账非真相源）
        try:
            self._db.append_ledger(item)
        except Exception:  # noqa: BLE001
            # 台账是分析副产品：写失败不影响进化主循环（事件表仍是唯一真相源）
            return

    # ---- 读 ----

    def read_all(self) -> List[Dict[str, Any]]:
        """全量读取（非 dict 行跳过——DB 行总是合法 JSON，防御性保留）。"""
        return self._db.read_ledger()

    def brief(self, n: int = 8) -> List[Dict[str, Any]]:
        """最近 n 条精简摘要（builder/coding/debug/adjust 的 ctx 注入形态）。"""
        if not isinstance(n, int) or n <= 0:
            return []
        out = []
        for e in self.read_all()[-n:]:
            gate = e.get("gate") if isinstance(e.get("gate"), dict) else {}
            out.append({
                "iteration_id": e.get("iteration_id"),
                "cell_id": e.get("cell_id"),
                "idea": _idea_text(e),
                "verdict": e.get("verdict"),
                "perf_value": _perf_of(e),
                "perf_delta": e.get("perf_delta"),
                "reject_reason": e.get("reject_reason"),
            })
        return out

    def perf_timeline(self, n: int = 10) -> List[Dict[str, Any]]:
        """性能时间线（近 n 条带 perf 的条目 + is_record 判定）。

        is_record：该条 perf **严格优于此前所有**带 perf 条目（台账惯例低优：
        runtime_ms 越小越好）。判定在**全量历史**上进行后再截取尾部 n 条——
        保证「历史新纪录」标记不因窗口截断而失真。
        """
        if not isinstance(n, int) or n <= 0:
            return []
        timeline: List[Dict[str, Any]] = []
        best: Optional[float] = None
        for e in self.read_all():
            perf = _perf_of(e)
            if perf is None:
                continue  # 无 perf 的条目（no_candidate 等）不上性能时间线
            is_record = best is None or perf < best
            if is_record:
                best = perf
            timeline.append({
                "iteration_id": e.get("iteration_id"),
                "cell_id": e.get("cell_id"),
                "perf_value": perf,
                "is_record": bool(is_record),
            })
        return timeline[-n:]

    def own_decisions(self, role: str, n: int = 5) -> List[Dict[str, Any]]:
        """某角色参与过的条目摘要（adjust 用：回看自己的历史决策与效果）。

        以条目 ``agents`` 是否含该 role 为判据。
        """
        if not isinstance(role, str) or not role:
            return []
        if not isinstance(n, int) or n <= 0:
            return []
        out = []
        for e in self.read_all():
            agents = e.get("agents")
            if not isinstance(agents, dict):
                continue
            stats = agents.get(role)
            if not isinstance(stats, dict):
                continue
            out.append({
                "iteration_id": e.get("iteration_id"),
                "cell_id": e.get("cell_id"),
                "verdict": e.get("verdict"),
                "perf_delta": e.get("perf_delta"),
                "role_stats": stats,
            })
        return out[-n:]
