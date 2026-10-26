"""lineage.py — 谱系重建与游标重放投影（job_47/task4）

修复 v5 缺陷：lineage 恒 None（parent_hash 无消费方）——本模块从事件流
重建精英谱系；`project_state` 是 task8 Scheduler._recover 的恢复基座
（崩溃重启后从 events.jsonl 游标重放投影系统状态）。

事件协议约定（task8 写事件时遵守，本模块消费）：
- ``elite_admitted`` / ``elite_replaced``：
  payload 必含 cell_id/code_path；谱系边可选 ``parents: [code_path, ...]``
  （直接父本，来自 experiment_plan.parents）或 ``lineage: [code_path, ...]``
  （完整祖先链，新→旧，调度器已维护时直接采信）。两者都有时 parents 优先。
  延迟入库（L2-Q6）经 ``pending: true`` + ``pending_reason`` 标志表达。
- ``invalidation`` / ``resurrection``：payload 含 cell_id；resurrection 可带
  ``restored_weight``（恢复采样权）。两者都会结清该格的 pending（重评估已结）。
- ``cell_sampled``：payload 含 cell_id/visit（采样后的最新值，last-wins 采信）。
- ``weight_changed``：payload 含 cell_id/new_weight（last-wins 采信）。
- ``session_started`` → state="running"；``session_ended`` → state="stopped"。

stdlib-only，Python >=3.9 兼容。事件类型以字符串字面量引用（不 import
task1 的 EVENT_TYPES——并行层禁耦合）。
"""
from typing import Any, Dict, List, Optional

# 消费的事件类型（与 events.py 的 EVENT_TYPES 字面量对齐）。
_ELITE_EVENTS = ("elite_admitted", "elite_replaced")
_WEIGHT_EVENT = "weight_changed"
_SAMPLED_EVENT = "cell_sampled"
_INVALIDATION_EVENT = "invalidation"
_RESURRECTION_EVENT = "resurrection"
_STARTED_EVENT = "iteration_started"
_ENDED_EVENT = "iteration_ended"

# 谱系回溯深度上限（防父本边成环导致死循环）。
_MAX_LINEAGE_DEPTH = 100


def _cell_safe(cell_id: str) -> str:
    """cell_id → experiment_id 安全形态（与 events.cell_safe 同规则，本模块内联
    一份以保持零 import 耦合）：非字母数字 → ``_``。"""
    import re as _re
    return _re.sub(r"[^A-Za-z0-9]", "_", cell_id or "")


def _sorted_events(events: List[Dict[str, Any]]) -> List[Dict[str, Any]]:
    """按 (ts, seq) 稳定排序（task49：seq 在并行写入下不再全局单调）。

    单写者场景下 (ts, seq) 与 seq 序一致；并行场景下同秒事件按 seq 排序，
    seq 回退时按 ts 先顺序。完全相等时保持文件原序（stable sort）。
    """
    def _key(ev):
        ts = ev.get("ts")
        seq = ev.get("seq")
        return (ts if isinstance(ts, str) else "",
                seq if isinstance(seq, (int, float)) else 0)
    return sorted(events, key=_key)


def _elite_chain(events: List[Dict[str, Any]],
                 cell_id: str) -> List[str]:
    """单格谱系核心：从最后一条精英事件回溯 parents 边到根（新→旧）。

    - payload.lineage（完整链）存在时直接采信（前面拼当前 code_path）。
    - 否则按 parents 边回溯：child → parent 映射取「最近一次出现」。
    - 环/缺失截断（_MAX_LINEAGE_DEPTH 兜底）。
    """
    relevant = [ev for ev in events
                if ev.get("type") in _ELITE_EVENTS
                and (ev.get("payload") or {}).get("cell_id") == cell_id]
    if not relevant:
        return []

    # payload.lineage 直采信（最后一条为准）
    last = relevant[-1].get("payload") or {}
    if isinstance(last.get("lineage"), list) and last["lineage"]:
        chain = [last.get("code_path")] + list(last["lineage"])
        seen = set()
        out = []
        for cp in chain:
            if cp is None or cp in seen:
                continue
            seen.add(cp)
            out.append(cp)
        return out

    # parents 边回溯：code_path → parents（最近一次出现优先）
    edges = {}  # type: Dict[str, List[str]]
    for ev in relevant:
        p = ev.get("payload") or {}
        cp = p.get("code_path")
        if cp is None:
            continue
        parents = p.get("parents")
        if isinstance(parents, list):
            edges[cp] = [x for x in parents if isinstance(x, str)]
        elif cp not in edges:
            edges[cp] = []

    current = last.get("code_path")
    if current is None:
        return []
    chain = [current]
    seen = {current}
    while len(chain) < _MAX_LINEAGE_DEPTH:
        parents = edges.get(current)
        if not parents:
            break
        nxt = None
        for cand in parents:      # 取第一个未访问的父本（多父本重组：主父线）
            if cand not in seen:
                nxt = cand
                break
        if nxt is None:
            break
        seen.add(nxt)
        chain.append(nxt)
        current = nxt
    return chain


def rebuild_lineage(events: List[Dict[str, Any]],
                    cell_id: str) -> List[str]:
    """从事件流重建某格精英谱系（新→旧的 code_path 祖先链）。

    未知格/无精英事件 → 空列表。
    """
    return _elite_chain(_sorted_events(events), cell_id)


def rebuild_all(events: List[Dict[str, Any]]) -> Dict[str, List[str]]:
    """全格谱系投影：{cell_id: [code_path, ...]}（重放恢复与北极星共用）。"""
    ordered = _sorted_events(events)
    cells = set()
    for ev in ordered:
        if ev.get("type") in _ELITE_EVENTS:
            cid = (ev.get("payload") or {}).get("cell_id")
            if isinstance(cid, str):
                cells.add(cid)
    return {cid: _elite_chain(ordered, cid) for cid in sorted(cells)}


def _new_cell_state() -> Dict[str, Any]:
    return {"visit": 0, "weight": 1.0, "elite": None}


def project_state(events: List[Dict[str, Any]]) -> Dict[str, Any]:
    """游标重放投影：events → 调度器恢复所需的最小系统状态。

    返回：
        {
          "iteration_cursor": int,   # **已完成**迭代的 max iteration_id
                                     # （无 iteration_ended 的旧日志回退到
                                     #  max iteration_started；无则 0）
          "state": "running"|"stopped"|None,
          "last_seq": int,           # 最后事件 seq（空流=0；并行下仅参考）
          "cells": {cell_id: {"visit": int, "weight": float,
                              "elite": {"code_path", "perf_value", "gen"}|None}},
          "pending": [{"cell_id", "code_path", "reason"}],
          "interrupted_experiments": [experiment_id, ...],
                                     # 有 iteration_started/cell_sampled
                                     # 但无 iteration_ended 的实验（task49：
                                     # 崩溃恢复时用于清理格锁/记事件）
        }

    投影规则（全部 last-wins，事件流是唯一真相源）：
    - cell_sampled → cells[cell].visit = payload.visit
    - weight_changed → cells[cell].weight = payload.new_weight
    - elite_admitted/replaced → cells[cell].elite = {code_path, perf_value, gen}；
      payload.pending=true → pending 追加 {cell_id, code_path, reason}
    - invalidation → cells[cell].elite = None（出档）+ pending 结清
    - resurrection → cells[cell].weight = payload.restored_weight（缺省 1.0）
      + pending 结清
    - session_started → state="running"；session_ended → state="stopped"

    游标语义（task49）：以「**已完成**迭代」为准——中断的迭代（started 但无
    ended）不推进 cursor，恢复后重跑该迭代号（旧行为取 max started 会把
    中断迭代当成已完成，导致该迭代被跳过）。旧日志（无 iteration_ended）
    回退到 max started，保持向后兼容。
    """
    ordered = _sorted_events(events)

    state = None               # type: Optional[str]
    iteration_cursor = 0
    last_seq = 0
    cells = {}                 # type: Dict[str, Dict[str, Any]]
    pending = []               # type: List[Dict[str, Any]]
    pending_cells = set()      # type: set
    # task49：迭代完成度记录（it -> {cell, started, ended}）
    iter_rec = {}              # type: Dict[int, Dict[str, Any]]

    def _rec(it):
        if it not in iter_rec:
            iter_rec[it] = {"cell": None, "started": True, "ended": False}
        return iter_rec[it]

    def cell(cid):
        if cid not in cells:
            cells[cid] = _new_cell_state()
        return cells[cid]

    for ev in ordered:
        seq = ev.get("seq")
        if isinstance(seq, (int, float)):
            last_seq = max(last_seq, int(seq))
        typ = ev.get("type")
        p = ev.get("payload") or {}

        if typ == "session_started":
            state = "running"
        elif typ == "session_ended":
            state = "stopped"
        elif typ in (_STARTED_EVENT, _ENDED_EVENT):
            it = p.get("iteration_id")
            if isinstance(it, (int, float)):
                rec = _rec(int(it))
                if typ == _ENDED_EVENT:
                    rec["ended"] = True
                cid0 = p.get("cell_id")
                if isinstance(cid0, str) and not rec.get("cell"):
                    rec["cell"] = cid0
        elif typ == _SAMPLED_EVENT:
            cid = p.get("cell_id")
            if isinstance(cid, str):
                visit = p.get("visit")
                c = cell(cid)
                if isinstance(visit, (int, float)) and visit >= 0:
                    c["visit"] = int(visit)
                elif c["visit"] is None:
                    c["visit"] = 0
                # 采样事件带 iteration_id 时用于推断中断实验的格归属
                it = p.get("iteration_id")
                if isinstance(it, (int, float)):
                    rec = _rec(int(it))
                    if not rec.get("cell"):
                        rec["cell"] = cid
        elif typ == _WEIGHT_EVENT:
            cid = p.get("cell_id")
            nw = p.get("new_weight")
            if isinstance(cid, str) and isinstance(nw, (int, float)):
                cell(cid)["weight"] = float(nw)
        elif typ in _ELITE_EVENTS:
            cid = p.get("cell_id")
            if isinstance(cid, str):
                elite = {
                    "code_path": p.get("code_path"),
                    "perf_value": p.get("perf_value"),
                    "gen": p.get("gen"),
                }
                cell(cid)["elite"] = elite
                if p.get("pending"):
                    reason = p.get("pending_reason") or "tolerance_band"
                    # last-wins：同格再次 pending（复测保留后重新入库）时替换旧项——
                    # 档案语义是 drain 后 add 新 code_path，投影若按 cell 去重丢弃
                    # 新项会与档案不一致（e2e 实测：iter008 项丢失）
                    if cid not in pending_cells:
                        pending_cells.add(cid)
                    else:
                        pending = [q for q in pending if q["cell_id"] != cid]
                    pending.append({
                        "cell_id": cid,
                        "code_path": p.get("code_path"),
                        "reason": reason,
                    })
        elif typ in (_INVALIDATION_EVENT, _RESURRECTION_EVENT):
            cid = p.get("cell_id")
            if isinstance(cid, str):
                if typ == _INVALIDATION_EVENT:
                    cell(cid)["elite"] = None
                else:  # resurrection
                    rw = p.get("restored_weight")
                    cell(cid)["weight"] = float(rw) if isinstance(
                        rw, (int, float)) else 1.0
                # 重评估已结 → pending 结清（保序移除该格全部项）
                if cid in pending_cells:
                    pending_cells.discard(cid)
                    pending = [q for q in pending if q["cell_id"] != cid]

    # ---- task49：游标 = 已完成迭代的 max；中断实验单独上报 ----
    ended_iters = [it for it, rec in iter_rec.items() if rec["ended"]]
    started_iters = [it for it, rec in iter_rec.items() if rec["started"]]
    if ended_iters:
        iteration_cursor = max(ended_iters)
    elif started_iters:
        # 旧日志兼容（无 iteration_ended 事件）：回退到 max started
        iteration_cursor = max(started_iters)
    interrupted = []  # type: List[str]
    for it in sorted(iter_rec):
        rec = iter_rec[it]
        if rec["ended"]:
            continue
        cid = rec.get("cell")
        interrupted.append("%s#iter%d" % (_cell_safe(cid) if cid else "unknown", it))

    return {
        "iteration_cursor": iteration_cursor,
        "state": state,
        "last_seq": last_seq,
        "cells": cells,
        "pending": pending,
        "interrupted_experiments": interrupted,
    }
