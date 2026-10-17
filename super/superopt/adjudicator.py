"""adjudicator.py — 外置裁决（job_47/task9，独立进程）

提案-裁决分离（design-tree KR5 / §3.5）：adjust agent 的**硬动作**（删格/改格
定义/修门）只产提案（loop 投递到 `<ws>/proposals/`），裁决在调度器进程之外
完成——本模块是**自动裁决规则**的实现；human 也可手改 decision 文件覆盖
（文件驱动天然支持 human-in-loop）。

裁决规则（内置，硬动作保守）：
  - hard_delete_cell：weight==0 **且** 连续 ≥20 迭代未被采样 → approve；
    否则 reject「先降权观察」。目标格的精英若是金样例（golden_examples
    引用）→ reject（金样例格不可删）。未知格 → reject（无证据不裁）。
  - hard_modify_cell / hard_modify_gate：**强制双回归证据**
    （proposal.evidence.regression = {elite_replay, exploit_regression}——
    gate4/task14 harness 的调用协议）：缺 → reject「缺双回归证据」；
    两者全过 → approve（conditions 附双回归结论）；任一失败 → reject。
  - soft_weight：软动作不走裁决（loop 直接应用）→ reject（路由错误提示）。
  - 未知/畸形提案 → reject。

进程契约：
  - `python3 -m superopt.adjudicator --proposal <path> --workdir <ws>`
  - 读提案文件 → adjudicate → 写回 `<ws>/proposals/<proposal_id>.decision.json`
  - 返回码：approve=0 / reject=1 / 参数或 IO 错误=2
  - **不写 events.jsonl**（单写者 = py 调度器；adjudicated 事件由调度器在
    下一迭代 drain decision 文件时写入）
  - 文件驱动、无锁：调度器或 human 在任意时刻跑本进程都安全

证据读取（只读）：
  - `<ws>/events.jsonl`：迭代游标与各格末次采样迭代（连续未采样间隔）、
    weight_changed 兜底
  - `<ws>/archive.json`：格现状（weight/visit/精英）
  - `<ws>/gates/golden_examples/`：金样例保护比对
  - `<ws>/gates/exploit_registry.json`：exploit 引用（提示性，不阻断）

stdlib-only，Python >=3.9 兼容（禁 PEP604 X|Y 联合类型注解）。
"""
import argparse
import json
import os
import sys
import time
from typing import Any, Dict, List, Optional, Tuple

__all__ = ["adjudicate", "main", "DECISION_APPROVE", "DECISION_REJECT",
           "DORMANT_ITERATIONS_REQUIRED"]

DECISION_APPROVE = "approve"
DECISION_REJECT = "reject"

# 删格观察期：连续 N 迭代未被采样才可批（spec：weight=0 且连续 20 迭代未采样）
DORMANT_ITERATIONS_REQUIRED = 20

# 硬动作提案类型（裁决对象；soft_weight 不在其列）
_HARD_TYPES = frozenset({
    "hard_delete_cell",
    "hard_modify_cell",
    "hard_modify_gate",
})

# 双回归证据的两个子键（task14 harness 协议）
_REGRESSION_KEYS = ("elite_replay", "exploit_regression")


# --------------------------------------------------------------------------
# 证据读取（全部只读，容忍缺失/损坏——坏档按「无证据」保守处理）
# --------------------------------------------------------------------------

def _read_events(ws: str) -> List[Dict[str, Any]]:
    """逐行读 events.jsonl（坏行跳过——与 EventWriter.read_all 同容错，但
    **只读不落句柄**：裁决进程不得对真相源有任何写副作用）。"""
    path = os.path.join(ws, "events.jsonl")
    events = []            # type: List[Dict[str, Any]]
    try:
        f = open(path, "r", encoding="utf-8")
    except OSError:
        return events
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
                events.append(ev)
    return events


def _read_archive_cells(ws: str) -> Dict[str, Dict[str, Any]]:
    """读 archive.json 的 cells（损坏/缺失 → 空表，由事件流兜底）。"""
    path = os.path.join(ws, "archive.json")
    try:
        with open(path, "r", encoding="utf-8") as f:
            data = json.load(f)
    except (OSError, ValueError):
        return {}
    if not isinstance(data, dict):
        return {}
    cells = data.get("cells")
    return cells if isinstance(cells, dict) else {}


def _scan_cell_facts(events: List[Dict[str, Any]],
                     cell_id: str) -> Dict[str, Any]:
    """从事件流提取目标格的裁决事实。

    返回 {"iteration_cursor": int, "last_sampled_iteration": Optional[int],
          "weight": Optional[float]}：
      - iteration_cursor：最后一个 iteration_started 的 iteration_id（无则 0）
      - last_sampled_iteration：目标格最后一次被采样时所处的迭代号
        （cell_sampled 自带 iteration_id 时优先；否则取当时最近的
        iteration_started；从未被采样 = None）
      - weight：最后一个针对该格的 weight_changed 的 new_weight（无 = None）
    """
    cursor = 0
    current_iter = None                     # type: Optional[int]
    last_sampled_iter = None                # type: Optional[int]
    weight = None                           # type: Optional[float]
    for ev in events:
        typ = ev.get("type")
        p = ev.get("payload") or {}
        if not isinstance(p, dict):
            continue
        if typ == "iteration_started":
            it = p.get("iteration_id")
            if isinstance(it, (int, float)) and not isinstance(it, bool):
                it = int(it)
                cursor = max(cursor, it)
                current_iter = it
        elif typ == "cell_sampled" and p.get("cell_id") == cell_id:
            it = p.get("iteration_id")
            if not (isinstance(it, (int, float)) and not isinstance(it, bool)):
                it = current_iter
            if it is not None:
                last_sampled_iter = max(last_sampled_iter or 0, int(it))
        elif typ == "weight_changed" and p.get("cell_id") == cell_id:
            nw = p.get("new_weight")
            if isinstance(nw, (int, float)) and not isinstance(nw, bool):
                weight = float(nw)
    return {
        "iteration_cursor": cursor,
        "last_sampled_iteration": last_sampled_iter,
        "weight": weight,
    }


def _golden_example_hit(ws: str, elite: Optional[dict]) -> bool:
    """目标格精英的 code_path 是否被 golden_examples 引用（金样例保护）。"""
    if not isinstance(elite, dict):
        return False
    code_path = elite.get("code_path")
    if not isinstance(code_path, str) or not code_path:
        return False
    gold_dir = os.path.join(ws, "gates", "golden_examples")
    if not os.path.isdir(gold_dir):
        return False
    base = os.path.basename(code_path.rstrip("/")) or code_path
    for name in os.listdir(gold_dir):
        if not name.endswith(".json"):
            continue
        try:
            with open(os.path.join(gold_dir, name), "r", encoding="utf-8") as f:
                entry = json.load(f)
        except (OSError, ValueError):
            continue
        if not isinstance(entry, dict):
            continue
        ref = entry.get("code_path")
        if isinstance(ref, str) and ref and ref == code_path:
            return True
        # 容错：金样例条目只写代号（basename 相同即视为引用）
        ref_base = os.path.basename(ref.rstrip("/")) if isinstance(ref, str) and ref else ""
        if ref_base and ref_base == base:
            return True
    return False


# --------------------------------------------------------------------------
# 裁决规则
# --------------------------------------------------------------------------

def _reject(reason: str, conditions: Optional[List[str]] = None) -> Dict[str, Any]:
    return {"decision": DECISION_REJECT, "reason": reason,
            "conditions": list(conditions or [])}


def _approve(reason: str, conditions: Optional[List[str]] = None) -> Dict[str, Any]:
    return {"decision": DECISION_APPROVE, "reason": reason,
            "conditions": list(conditions or [])}


def _adjudicate_delete(proposal: dict, ws: str) -> Dict[str, Any]:
    """hard_delete_cell：weight=0 且连续 ≥20 迭代未采样才可批。"""
    target = proposal.get("target_cell")
    if not isinstance(target, str) or not target:
        return _reject("删格提案缺 target_cell")

    cells = _read_archive_cells(ws)
    cell = cells.get(target)
    events = _read_events(ws)
    facts = _scan_cell_facts(events, target)

    # weight：档案优先，事件流兜底；两边都无证据 → 保守拒绝
    weight = None                            # type: Optional[float]
    if isinstance(cell, dict):
        w = cell.get("weight")
        if isinstance(w, (int, float)) and not isinstance(w, bool):
            weight = float(w)
    if weight is None:
        weight = facts["weight"]
    if weight is None:
        return _reject("目标格 %s 无 weight 证据（档案与事件流均无记录）——"
                       "无证据不裁，先降权观察" % target)
    if weight > 0:
        return _reject("目标格 %s weight=%.3f 仍活跃——先降权观察"
                       "（软动作 set_weight 0 后再提案）" % (target, weight))

    # 连续未采样间隔：游标 - 末次采样迭代；从未采样 = 间隔无穷大
    cursor = facts["iteration_cursor"]
    last_sampled = facts["last_sampled_iteration"]
    if last_sampled is None:
        gap = None  # 从未采样
    else:
        gap = cursor - last_sampled
    if last_sampled is not None and gap < DORMANT_ITERATIONS_REQUIRED:
        return _reject("目标格 %s 末次采样在 %d 迭代前（游标 %d），"
                       "未满 %d 迭代观察期——先降权观察"
                       % (target, gap, cursor, DORMANT_ITERATIONS_REQUIRED))

    # 金样例保护
    elite = cell.get("elite") if isinstance(cell, dict) else None
    if _golden_example_hit(ws, elite):
        return _reject("目标格 %s 的精英是金样例（golden_examples 引用）——"
                       "金样例格不可删" % target)

    dormant_note = ("从未被采样" if last_sampled is None
                    else "连续 %d 迭代未采样" % gap)
    return _approve(
        "目标格 %s weight=0 且%s（游标 %d）——休眠格可删"
        % (target, dormant_note, cursor),
        conditions=["删除后格定义仍保留在事件流（append-only 可追溯）",
                    "若需恢复，从 events.jsonl 重放重建"])


def _regression_evidence(proposal: dict) -> Optional[Dict[str, Any]]:
    """提取提案的双回归证据（宽容定位，严格验证）。

    定位顺序（提案形状宽容——投递方可能把证据挂在不同层）：
      1. proposal.evidence.regression（spec 主契约，task14 harness 协议）
      2. proposal.regression（顶层直接携带）
      3. proposal.evidence 自身即回归结果 dict（同形状误挂）
    验证严格：两子键（elite_replay + exploit_regression）齐全且各自含
    passed 真值，否则视为缺证据（None）。
    """
    if not isinstance(proposal, dict):
        return None
    ev = proposal.get("evidence")
    candidates = []
    if isinstance(ev, dict):
        if isinstance(ev.get("regression"), dict):
            candidates.append(ev["regression"])
        candidates.append(ev)          # evidence 自身即回归 dict（同形状）
    if isinstance(proposal.get("regression"), dict):
        candidates.append(proposal["regression"])

    for reg in candidates:
        keys_present = all(k in reg for k in _REGRESSION_KEYS)
        if not keys_present:
            continue
        out = {}
        ok = True
        for key in _REGRESSION_KEYS:
            part = reg.get(key)
            if not isinstance(part, dict) or "passed" not in part:
                ok = False
                break
            out[key] = part
        if ok:
            return out
    return None


def _adjudicate_modify(proposal: dict, ws: str) -> Dict[str, Any]:
    """hard_modify_cell / hard_modify_gate：强制双回归证据，按证据裁决。"""
    reg = _regression_evidence(proposal)
    if reg is None:
        return _reject("缺双回归证据（proposal.evidence.regression 需含 "
                       "elite_replay + exploit_regression 结果）——修门/改格定义"
                       "必须先过双回归：elite 回放防误杀 + exploit 回归防放水")
    conditions = []
    for key in _REGRESSION_KEYS:
        passed = bool(reg[key].get("passed"))
        if not passed:
            detail = reg[key]
            bad = detail.get("killed") or detail.get("leaked") or []
            return _reject("双回归未全过：%s failed（%s）——按证据拒绝"
                           % (key, json.dumps(bad, ensure_ascii=False)[:200]))
        conditions.append("%s passed" % key)
    return _approve("双回归全过（%s）——按证据批准" % " + ".join(conditions),
                    conditions=conditions)


def adjudicate(proposal: dict, ws: str) -> Dict[str, Any]:
    """裁决一条硬动作提案（只读 ws 证据，无任何写副作用）。

    参数：
      proposal：提案 dict（adjust agent 输出的 proposal 项 + proposal_id 等
                元信息；字段宽容——缺关键字段按保守拒绝处理）
      ws：super 工作区路径（读 events.jsonl / archive.json / gates/）

    返回：{"decision": "approve"|"reject", "reason": str, "conditions": [str]}
    """
    if not isinstance(proposal, dict):
        return _reject("提案必须是 JSON 对象，得到 %s" % type(proposal).__name__)
    ptype = proposal.get("type")
    if not isinstance(ptype, str) or not ptype:
        return _reject("提案缺 type 字段（畸形提案）")
    if ptype == "soft_weight":
        return _reject("软动作（soft_weight）不走裁决——由调度器直接应用"
                       "（weight_changed 事件）；仅硬动作（%s）进入外置裁决"
                       % "/".join(sorted(_HARD_TYPES)))
    if ptype not in _HARD_TYPES:
        return _reject("未知提案类型 %r（合法硬动作：%s）" % (ptype, sorted(_HARD_TYPES)))
    if ptype == "hard_delete_cell":
        return _adjudicate_delete(proposal, ws)
    return _adjudicate_modify(proposal, ws)


# --------------------------------------------------------------------------
# 独立进程入口
# --------------------------------------------------------------------------

def _now_rfc3339() -> str:
    """RFC3339 带本地时区（domain/bugs.md：无时区会被 Go time.Parse 拒绝）。"""
    return time.strftime("%Y-%m-%dT%H:%M:%S", time.localtime()) + _tz_suffix()


def _tz_suffix() -> str:
    off = time.timezone if time.daylight == 0 else time.altzone
    if off == 0:
        return "Z"
    sign = "-" if off > 0 else "+"
    off = abs(off)
    return "%s%02d:%02d" % (sign, off // 3600, (off % 3600) // 60)


def _proposal_id(proposal: dict, proposal_path: str) -> str:
    """提案 id：内容 proposal_id 优先，缺则用文件名 stem。"""
    pid = proposal.get("proposal_id")
    if isinstance(pid, str) and pid.strip():
        return pid.strip()
    return os.path.splitext(os.path.basename(proposal_path))[0]


def main(argv: Optional[List[str]] = None) -> int:
    """CLI 入口：python3 -m superopt.adjudicator --proposal <f> --workdir <ws>。

    返回码：approve=0 / reject=1 / 参数或 IO 错误=2。
    产物：<ws>/proposals/<proposal_id>.decision.json
    """
    parser = argparse.ArgumentParser(
        prog="superopt.adjudicator",
        description="superopt 外置裁决：硬动作提案（删格/改格定义/修门）的独立"
                    "裁决进程。文件驱动、无锁、不写 events.jsonl。")
    parser.add_argument("--proposal", required=True,
                        help="提案文件路径（<ws>/proposals/<id>.json）")
    parser.add_argument("--workdir", required=True,
                        help="super 工作区路径")
    args = parser.parse_args(argv)

    try:
        with open(args.proposal, "r", encoding="utf-8") as f:
            proposal = json.load(f)
    except (OSError, ValueError) as e:
        print("读取提案失败 %s: %s" % (args.proposal, e), file=sys.stderr)
        return 2
    if not isinstance(proposal, dict):
        # 畸形提案也走裁决语义（保守拒绝），产出 decision 文件留痕
        proposal = {"proposal_id": None, "type": None, "_malformed": True}

    decision = adjudicate(proposal, args.workdir)

    pid = _proposal_id(proposal, args.proposal)
    out_dir = os.path.join(args.workdir, "proposals")
    os.makedirs(out_dir, exist_ok=True)
    out_path = os.path.join(out_dir, pid + ".decision.json")
    record = {
        "proposal_id": pid,
        "decision": decision["decision"],
        "reason": decision["reason"],
        "conditions": decision["conditions"],
        "adjudicated_at": _now_rfc3339(),
        "adjudicator": "auto",   # 自动规则裁决；human 手改本文件即覆盖
    }
    with open(out_path, "w", encoding="utf-8") as f:
        json.dump(record, f, ensure_ascii=False, indent=2)
    print("%s %s: %s" % (decision["decision"].upper(), pid, decision["reason"]))
    return 0 if decision["decision"] == DECISION_APPROVE else 1


if __name__ == "__main__":
    sys.exit(main())
