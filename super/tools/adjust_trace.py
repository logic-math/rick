#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""adjust_trace.py — adjust 决策-效果对照工具（《自改进的进化搜索算法》归因层）。

用法::

    python3 super/tools/adjust_trace.py --workdir <ws> [--window 5]

解析 <ws>/events.jsonl，输出 markdown 表到 stdout：每次 adjust 决策一行，
含「决策内容（迭代号 / proposals 摘要：格+旧新权重 / 决策时格状态）」与
「效果（决策后 K 迭代窗口内受影响格的：是否被采 / gate 通过率 / perf 变化）」。

数据源（两级）：
  1. adjust_decision 事件（task19 落地：iteration/agent_valid/proposals/
     cells_status_summary——输入输出同帧快照）；
  2. 回退：无 adjust_decision 事件时，从 weight_changed（source=agent:adjust）
     / proposal 事件序列重构决策（旧版事件流兼容）。

效果推导（事件流为唯一真相）：
  - 是否被采：窗口内 cell_sampled（cell_id 命中）
  - gate 通过率：窗口内 gate_result（cell_id 命中）passed 比例
  - perf 变化：窗口内 elite_admitted/elite_replaced 的 perf_value 与决策时
    cells_status 中该格精英 perf 的差值（低优=负数=改善）

stdlib-only，Python 3.9 兼容。
"""
import argparse
import json
import os
import sys
from typing import Any, Dict, List, Optional

# ---- 事件常量（与 super/superopt/events.py 字符串字面量保持一致；零 import 耦合） ----
EV_ADJUST_DECISION = "adjust_decision"
EV_WEIGHT_CHANGED = "weight_changed"
EV_PROPOSAL = "proposal"
EV_CELL_SAMPLED = "cell_sampled"
EV_GATE_RESULT = "gate_result"
EV_ELITE_ADMITTED = "elite_admitted"
EV_ELITE_REPLACED = "elite_replaced"
EV_ITERATION_ENDED = "iteration_ended"


def _read_events(workdir: str) -> List[Dict[str, Any]]:
    """全量读取 events.jsonl（坏行跳过——与 EventWriter.read_all 同容错）。"""
    path = os.path.join(workdir, "events.jsonl")
    out: List[Dict[str, Any]] = []
    if not os.path.exists(path):
        return out
    with open(path, "r", encoding="utf-8") as f:
        for line in f:
            line = line.strip()
            if not line:
                continue
            try:
                ev = json.loads(line)
                if isinstance(ev, dict) and "type" in ev:
                    out.append(ev)
            except ValueError:
                continue  # 半写行/坏行
    return out


def _payload(ev: Dict[str, Any]) -> Dict[str, Any]:
    p = ev.get("payload")
    return p if isinstance(p, dict) else {}


def _fmt_perf(value: Any) -> str:
    """perf 值的紧凑展示（None → -）。"""
    if isinstance(value, (int, float)):
        return ("%.4g" % float(value))
    return "-"


# --------------------------------------------------------------------------- #
# 决策帧提取
# --------------------------------------------------------------------------- #

def _decisions_from_adjust_events(events: List[Dict[str, Any]]
                                  ) -> List[Dict[str, Any]]:
    """主路径：adjust_decision 事件 → 决策帧。"""
    frames = []
    for ev in events:
        if ev.get("type") != EV_ADJUST_DECISION:
            continue
        p = _payload(ev)
        frames.append({
            "seq": ev.get("seq"),
            "iteration": p.get("iteration"),
            "agent_valid": p.get("agent_valid"),
            "proposals": p.get("proposals") or [],
            "cells_status": p.get("cells_status_summary") or [],
        })
    return frames


def _decisions_from_legacy_events(events: List[Dict[str, Any]]
                                  ) -> List[Dict[str, Any]]:
    """回退路径：weight_changed / proposal（agent:adjust 来源）→ 重构决策帧。

    重构规则：同一迭代边界产生的连续 weight_changed/proposal 事件归为一帧
    （以事件序列中第一个触发项开帧，遇到非 weight_changed/proposal 且非
    heartbeat 的事件关帧）。
    """
    frames: List[Dict[str, Any]] = []
    current: Optional[Dict[str, Any]] = None
    for ev in events:
        typ = ev.get("type")
        p = _payload(ev)
        if typ == EV_WEIGHT_CHANGED and ev.get("source") == "agent:adjust":
            if current is None:
                current = {"seq": ev.get("seq"), "iteration": None,
                           "agent_valid": None, "proposals": [],
                           "cells_status": []}
                frames.append(current)
            current["proposals"].append({
                "type": "soft_weight",
                "target_cell": p.get("cell_id"),
                "action": "weight %s -> %s" % (p.get("old_weight"),
                                               p.get("new_weight")),
                "evidence": str(p.get("reason") or "")[:200],
            })
        elif typ == EV_PROPOSAL:
            if current is None:
                current = {"seq": ev.get("seq"), "iteration": None,
                           "agent_valid": None, "proposals": [],
                           "cells_status": []}
                frames.append(current)
            current["proposals"].append({
                "type": p.get("type"),
                "target_cell": p.get("target_cell"),
                "action": p.get("action"),
                "evidence": str(p.get("evidence") or "")[:200],
            })
        elif typ == EV_ITERATION_ENDED:
            # 迭代边界 → 关帧（决策归属其后的迭代号）
            if current is not None:
                current["iteration"] = p.get("iteration_id")
                current = None
        elif typ == EV_CELL_SAMPLED:
            # 新迭代开始 → 关帧
            if current is not None:
                current = None
    return frames


def extract_decisions(events: List[Dict[str, Any]]
                      ) -> List[Dict[str, Any]]:
    """决策帧提取：adjust_decision 优先，无则回退 legacy 重构。"""
    frames = _decisions_from_adjust_events(events)
    if frames:
        return frames
    return _decisions_from_legacy_events(events)


# --------------------------------------------------------------------------- #
# 效果推导
# --------------------------------------------------------------------------- #

def _affected_cells(decision: Dict[str, Any]) -> List[str]:
    """决策涉及的格（proposals 的 target_cell 去重保序）。"""
    seen = set()
    out = []
    for p in decision.get("proposals") or []:
        cid = p.get("target_cell")
        if isinstance(cid, str) and cid and cid not in seen:
            seen.add(cid)
            out.append(cid)
    return out


def _iteration_at_seq(events: List[Dict[str, Any]], seq: Any
                      ) -> Optional[int]:
    """决策帧 seq 落在哪个迭代（最近的 iteration_started/ended 的 id）。"""
    try:
        seq_i = int(seq)
    except (TypeError, ValueError):
        return None
    it = None
    for ev in events:
        try:
            s = int(ev.get("seq"))
        except (TypeError, ValueError):
            continue
        if s > seq_i:
            break
        if ev.get("type") in ("iteration_started", EV_ITERATION_ENDED):
            v = _payload(ev).get("iteration_id")
            if isinstance(v, int):
                it = v
    return it


def effect_for_cell(events: List[Dict[str, Any]], cell_id: str,
                    start_iter: int, window: int) -> Dict[str, Any]:
    """决策后 K 迭代窗口内该格的效果：是否被采 / gate 通过率 / perf 变化。"""
    lo, hi = start_iter, start_iter + window  # (start_iter, start_iter+window]
    sampled = 0
    gate_total = 0
    gate_pass = 0
    perf_best = None          # type: Optional[float]
    elite_events = 0
    for ev in events:
        p = _payload(ev)
        typ = ev.get("type")
        it = p.get("iteration_id")
        if not isinstance(it, int) or not (lo < it <= hi):
            continue
        if typ == EV_CELL_SAMPLED and p.get("cell_id") == cell_id:
            sampled += 1
        elif typ == EV_GATE_RESULT and p.get("cell_id") == cell_id:
            gate_total += 1
            if p.get("passed"):
                gate_pass += 1
        elif typ in (EV_ELITE_ADMITTED, EV_ELITE_REPLACED) \
                and p.get("cell_id") == cell_id:
            elite_events += 1
            v = p.get("perf_value")
            if isinstance(v, (int, float)):
                if perf_best is None or v < perf_best:
                    perf_best = float(v)  # 低优：小=好（perf_desc=low 场景）
    return {
        "sampled": sampled,
        "gate_total": gate_total,
        "gate_pass": gate_pass,
        "perf_best": perf_best,
        "elite_events": elite_events,
    }


def _baseline_perf(decision: Dict[str, Any], cell_id: str) -> Optional[float]:
    """决策时该格的精英 perf（cells_status 快照；无则 None）。"""
    for c in decision.get("cells_status") or []:
        if c.get("cell_id") == cell_id:
            v = c.get("elite_perf")
            if isinstance(v, (int, float)):
                return float(v)
    return None


def _cells_status_line(decision: Dict[str, Any]) -> str:
    """决策时格状态摘要（紧凑：cell=visit/w/perf ...）。"""
    parts = []
    for c in (decision.get("cells_status") or [])[:8]:
        parts.append("%s(v%s/w%s/p%s)" % (
            str(c.get("cell_id", ""))[:36],
            c.get("visit", "?"), c.get("weight", "?"),
            _fmt_perf(c.get("elite_perf"))))
    if len(decision.get("cells_status") or []) > 8:
        parts.append("…+%d" % (len(decision["cells_status"]) - 8))
    return " ".join(parts) if parts else "-"


def _proposals_line(decision: Dict[str, Any]) -> str:
    """proposals 摘要（每条：type@cell action）。"""
    parts = []
    for p in (decision.get("proposals") or [])[:6]:
        cid = str(p.get("target_cell") or "")[:36]
        action = str(p.get("action") or "")[:40]
        parts.append("[%s] %s: %s" % (p.get("type") or "?", cid, action))
    if len(decision.get("proposals") or []) > 6:
        parts.append("…+%d 条" % (len(decision["proposals"]) - 6))
    return "<br>".join(parts) if parts else "（无提案）"


# --------------------------------------------------------------------------- #
# 输出
# --------------------------------------------------------------------------- #

def render_markdown(events: List[Dict[str, Any]], window: int) -> str:
    """主渲染：决策-效果对照 markdown 表。"""
    decisions = extract_decisions(events)
    if not decisions:
        return ("# adjust 决策-效果对照\n\n（无 adjust 决策事件："
                "本工作区尚未触发过 adjust agent）\n")

    lines: List[str] = []
    lines.append("# adjust 决策-效果对照")
    lines.append("")
    lines.append("窗口：决策后 %d 迭代；perf 为低优语义（小=好，Δ 负=改善）。"
                 % window)
    lines.append("")

    # 表 1：决策总表
    lines.append("## 决策轨迹")
    lines.append("")
    lines.append("| # | 迭代 | 决策时格状态 | proposals |")
    lines.append("|---|------|--------------|-----------|")
    for i, d in enumerate(decisions, 1):
        it = d.get("iteration")
        if it is None:
            it = _iteration_at_seq(events, d.get("seq"))
        valid = "" if d.get("agent_valid") in (None, True) else " ⚠invalid"
        lines.append("| %d | %s%s | %s | %s |" % (
            i, it, valid, _cells_status_line(d), _proposals_line(d)))
    lines.append("")

    # 表 2：决策-效果对照（受影响格逐行）
    lines.append("## 决策-效果对照（受影响格）")
    lines.append("")
    lines.append("| 决策# | 迭代 | 格 | 决策时perf | 窗口内被采 | gate通过 | "
                 "窗口最优perf | perf Δ |")
    lines.append("|-------|------|----|-----------|-----------|---------|"
                 "-------------|--------|")
    for i, d in enumerate(decisions, 1):
        it = d.get("iteration")
        if it is None:
            it = _iteration_at_seq(events, d.get("seq")) or 0
        for cid in _affected_cells(d):
            eff = effect_for_cell(events, cid, int(it), window)
            base = _baseline_perf(d, cid)
            delta = "-"
            if base is not None and eff["perf_best"] is not None:
                delta = "%+.4g" % (eff["perf_best"] - base)
            gate = "-"
            if eff["gate_total"] > 0:
                gate = "%d/%d" % (eff["gate_pass"], eff["gate_total"])
            lines.append("| %d | %s | %s | %s | %d | %s | %s | %s |" % (
                i, it, cid[:44], _fmt_perf(base), eff["sampled"],
                gate, _fmt_perf(eff["perf_best"]), delta))
    lines.append("")
    return "\n".join(lines)


def main() -> int:
    ap = argparse.ArgumentParser(
        description="adjust 决策-效果对照（自改进归因工具）")
    ap.add_argument("--workdir", required=True,
                    help="super 工作区目录（含 events.jsonl）")
    ap.add_argument("--window", type=int, default=5,
                    help="效果观察窗口（决策后 K 迭代，默认 5）")
    args = ap.parse_args()

    ws = os.path.abspath(args.workdir)
    if not os.path.isdir(ws):
        print("error: workdir 不存在: %s" % ws, file=sys.stderr)
        return 2
    events = _read_events(ws)
    if not events:
        print("error: %s 无可读事件（events.jsonl 缺失或为空）"
              % os.path.join(ws, "events.jsonl"), file=sys.stderr)
        return 2

    print(render_markdown(events, max(1, args.window)))
    return 0


if __name__ == "__main__":
    sys.exit(main())
