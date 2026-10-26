#!/usr/bin/env python3
"""cost_report.py — super 进化 run 的 token 成本分布报告（task46）

用法：
    python3 super/tools/cost_report.py --workdir <ws> [--json] [--top N]

数据源（按可信度）：
  1. `<ws>/events.jsonl`——**权威口径**：`agent_ended.usage`（input/output/
     cache_read/cache_write/total_tokens/cost_usd）+ `wall_ms`/`tool_calls`/
     `truncated`/`model`；`iteration_ended.tokens`/`cost_usd`/`by_role`
  2. pi session 文件（可选增强）：`<agent_dir>/sessions/<slug>/*.jsonl` 的
     `message_end.role==assistant.usage`——看单轮上下文膨胀；agent_dir 由
     `super.json` 的 `pi_path`（`…/agent/runtime/node_modules/.bin/pi`）上推，
     或环境变量 `PI_CODING_AGENT_DIR`。定位失败 → 自动降级为仅事件流。

输出：markdown 表（默认）或 JSON（`--json`）。只读，不修改任何状态。

stdlib-only，Python >=3.9 兼容。
"""
import argparse
import json
import os
import sys

__all__ = ["build_report", "render_markdown", "main"]

_USAGE_KEYS = ("input_tokens", "output_tokens", "cache_read", "cache_write",
               "total_tokens", "cost_usd")


def _read_events(ws):
    """读 events.jsonl；坏行/缺失容错（返回 []）。"""
    path = os.path.join(ws, "events.jsonl")
    out = []
    try:
        with open(path, "r", encoding="utf-8") as f:
            for line in f:
                line = line.strip()
                if not line:
                    continue
                try:
                    ev = json.loads(line)
                except ValueError:
                    continue
                if isinstance(ev, dict):
                    out.append(ev)
    except OSError:
        return []
    return out


def _num(v):
    try:
        return float(v or 0)
    except (TypeError, ValueError):
        return 0.0


def _usage_of(payload):
    u = payload.get("usage") or {}
    if not isinstance(u, dict):
        u = {}
    out = {}
    for k in _USAGE_KEYS:
        out[k] = _num(u.get(k))
    return out


def _agent_calls(events):
    """抽取每次 agent 派发的成本记录（按事件顺序）。"""
    calls = []
    for ev in events:
        if ev.get("type") != "agent_ended":
            continue
        p = ev.get("payload") or {}
        u = _usage_of(p)
        calls.append({
            "seq": ev.get("seq"),
            "iteration_id": p.get("iteration_id"),
            "role": str(p.get("role") or "?"),
            "model": str(p.get("model") or ""),
            "wall_ms": int(_num(p.get("wall_ms"))),
            "tool_calls": len(p.get("tool_calls") or []),
            "truncated": bool(p.get("truncated")),
            "valid": p.get("valid"),
            "input_tokens": int(u["input_tokens"]),
            "output_tokens": int(u["output_tokens"]),
            "cache_read": int(u["cache_read"]),
            "cache_write": int(u["cache_write"]),
            "total_tokens": int(u["total_tokens"]),
            "cost_usd": round(u["cost_usd"], 6),
        })
    return calls


def _iteration_totals(events):
    """每次迭代的合计（优先 iteration_ended 自带口径，缺则用 calls 聚合）。"""
    iters = {}
    for ev in events:
        if ev.get("type") != "iteration_ended":
            continue
        p = ev.get("payload") or {}
        it = p.get("iteration_id")
        if it is None:
            continue
        iters[int(it)] = {
            "iteration_id": int(it),
            "cell_id": p.get("cell_id"),
            "tokens": int(_num(p.get("tokens"))),
            "cost_usd": round(_num(p.get("cost_usd")), 6),
            "wall_ms": int(_num(p.get("wall_ms"))),
            "by_role": p.get("by_role") or {},
        }
    return iters


def _role_stats(calls):
    """按 role 聚合：次数/总量/单次中位/占比。"""
    roles = {}
    for c in calls:
        r = roles.setdefault(c["role"], {
            "calls": 0, "tokens": 0, "input_tokens": 0, "output_tokens": 0,
            "cache_read": 0, "cost_usd": 0.0, "wall_ms": 0, "tool_calls": 0,
            "truncated": 0, "_samples": [],
        })
        r["calls"] += 1
        r["tokens"] += c["total_tokens"]
        r["input_tokens"] += c["input_tokens"]
        r["output_tokens"] += c["output_tokens"]
        r["cache_read"] += c["cache_read"]
        r["cost_usd"] = round(r["cost_usd"] + c["cost_usd"], 6)
        r["wall_ms"] += c["wall_ms"]
        r["tool_calls"] += c["tool_calls"]
        r["truncated"] += 1 if c["truncated"] else 0
        r["_samples"].append(c["total_tokens"])
    total = sum(v["tokens"] for v in roles.values()) or 1
    for r in roles.values():
        r["share_pct"] = round(100.0 * r["tokens"] / total, 1)
        r["tokens_median"] = _median(r.pop("_samples"))
        # pi 的 usage.input 不含缓存部分（cacheRead 单列）——命中率分母须为
        # input + cache_read，否则会出现 >100% 的荒谬值（真实 run 实测）
        _denom = r["input_tokens"] + r["cache_read"]
        r["cache_hit_pct"] = (round(100.0 * r["cache_read"] / _denom, 1)
                              if _denom else 0.0)
    return roles


def _median(vals):
    if not vals:
        return 0
    s = sorted(vals)
    n = len(s)
    mid = n // 2
    if n % 2:
        return int(s[mid])
    return int((s[mid - 1] + s[mid]) / 2)


def _context_growth(calls):
    """每 role 的 input_tokens 序列（上下文膨胀曲线）。"""
    seq = {}
    for c in calls:
        seq.setdefault(c["role"], []).append(c["input_tokens"])
    return seq


def _locate_agent_dir(ws):
    """由 super.json 的 pi_path 或 PI_CODING_AGENT_DIR 推 agent_dir（best-effort）。"""
    env_dir = os.environ.get("PI_CODING_AGENT_DIR")
    if env_dir and os.path.isdir(env_dir):
        return env_dir
    try:
        with open(os.path.join(ws, "super.json"), "r", encoding="utf-8") as f:
            sj = json.load(f)
        pi_path = str((sj.get("config") or {}).get("pi_path") or "")
    except (OSError, ValueError):
        return None
    marker = os.sep + "runtime" + os.sep
    if marker in pi_path:
        cand = pi_path.split(marker, 1)[0]
        if os.path.isdir(cand):
            return cand
    return None


def _session_detail(ws, agent_dir, max_files=200, max_bytes=2 * 1024 * 1024):
    """session 级明细（best-effort）：匹配 codes/iter* 的 session 文件数 + usage 合计。

    定位失败/无匹配 → None（调用方降级为仅事件流）。
    """
    if not agent_dir:
        return None
    sess_root = os.path.join(agent_dir, "sessions")
    if not os.path.isdir(sess_root):
        return None
    matched = []
    try:
        for slug in sorted(os.listdir(sess_root)):
            d = os.path.join(sess_root, slug)
            if not os.path.isdir(d):
                continue
            if "codes-iter" not in slug and "codes_iter" not in slug:
                continue
            for name in sorted(os.listdir(d)):
                if name.endswith(".jsonl"):
                    matched.append(os.path.join(d, name))
    except OSError:
        return None
    if not matched:
        return None
    files = matched[-max_files:]
    totals = {"files": len(files), "assistant_turns": 0,
              "input_tokens": 0, "output_tokens": 0, "total_tokens": 0}
    for path in files:
        try:
            size = os.path.getsize(path)
            if size > max_bytes:
                continue
            with open(path, "r", encoding="utf-8") as f:
                for line in f:
                    line = line.strip()
                    if not line:
                        continue
                    try:
                        ev = json.loads(line)
                    except ValueError:
                        continue
                    if not isinstance(ev, dict):
                        continue
                    m = ev.get("message") or {}
                    if not isinstance(m, dict) or m.get("role") != "assistant":
                        continue
                    u = m.get("usage")
                    if not isinstance(u, dict):
                        continue
                    totals["assistant_turns"] += 1
                    totals["input_tokens"] += int(_num(u.get("input")))
                    totals["output_tokens"] += int(_num(u.get("output")))
                    totals["total_tokens"] += int(_num(u.get("totalTokens")))
        except OSError:
            continue
    return totals


def _waste_flags(calls, iters, roles):
    """浪费模式快速判定（skill 清单 1-5 的可机检子集）。"""
    flags = []
    # ① 重复 benchmark：同一迭代内同 role 多次派发（多次 gate_check/benchmark 的间接信号）
    per_iter_role = {}
    for c in calls:
        if c["iteration_id"] is None:
            continue
        key = (c["iteration_id"], c["role"])
        per_iter_role[key] = per_iter_role.get(key, 0) + 1
    for (it, role), n in sorted(per_iter_role.items()):
        if role == "debug" and n >= 3:
            flags.append("迭代 %s：debug 派发 %d 次（疑似重复复验/瞬时波动）" % (it, n))
    # ② 超长上下文：单次 input_tokens > 200k
    for c in calls:
        if c["input_tokens"] > 200000:
            flags.append("迭代 %s：%s 单次 input %dk（超长上下文，疑似大文件灌入）"
                         % (c["iteration_id"], c["role"], c["input_tokens"] // 1000))
    # ③ 截断
    for c in calls:
        if c["truncated"]:
            flags.append("迭代 %s：%s 撞回合上限截断（truncated）"
                         % (c["iteration_id"], c["role"]))
    # ④ 单迭代 token 异常（> 2× 中位）
    vals = [v["tokens"] for v in iters.values() if v["tokens"] > 0]
    med = _median(vals)
    if med:
        for it, v in sorted(iters.items()):
            if v["tokens"] > 2 * med:
                flags.append("迭代 %s：tokens %dk 超中位 %dk 的 2×（异常昂贵）"
                             % (it, v["tokens"] // 1000, med // 1000))
    # ⑤ cache 命中低
    for role, r in roles.items():
        if r["input_tokens"] >= 50000 and r["cache_hit_pct"] < 30:
            flags.append("%s：cache 命中仅 %.1f%%（前缀不稳定？）"
                         % (role, r["cache_hit_pct"]))
    return flags


def build_report(ws, top=5):
    """组装完整报告 dict（供 markdown / json 两种渲染）。"""
    events = _read_events(ws)
    calls = _agent_calls(events)
    iters = _iteration_totals(events)
    roles = _role_stats(calls)
    # iteration_ended 缺失时用 calls 兜底聚合
    if not iters and calls:
        agg = {}
        for c in calls:
            it = c["iteration_id"]
            if it is None:
                continue
            a = agg.setdefault(int(it), {"iteration_id": int(it), "cell_id": None,
                                         "tokens": 0, "cost_usd": 0.0,
                                         "wall_ms": 0, "by_role": {}})
            a["tokens"] += c["total_tokens"]
            a["cost_usd"] = round(a["cost_usd"] + c["cost_usd"], 6)
            a["wall_ms"] += c["wall_ms"]
            br = a["by_role"].setdefault(c["role"], {"tokens": 0, "wall_ms": 0,
                                                     "tool_calls": 0})
            br["tokens"] += c["total_tokens"]
            br["wall_ms"] += c["wall_ms"]
            br["tool_calls"] += c["tool_calls"]
        iters = agg
    agent_dir = _locate_agent_dir(ws)
    sessions = _session_detail(ws, agent_dir)
    total_tokens = sum(v["tokens"] for v in roles.values())
    total_cost = round(sum(v["cost_usd"] for v in roles.values()), 6)
    total_wall = sum(v["wall_ms"] for v in roles.values())
    expensive = sorted(iters.values(), key=lambda v: v["tokens"], reverse=True)[:top]
    return {
        "workspace": os.path.abspath(ws),
        "events": len(events),
        "iterations": len(iters),
        "total": {"tokens": total_tokens, "cost_usd": total_cost,
                  "wall_ms": total_wall},
        "roles": roles,
        "iterations_detail": sorted(iters.values(), key=lambda v: v["iteration_id"]),
        "top_expensive_iterations": expensive,
        "context_growth": _context_growth(calls),
        "session_detail": sessions,
        "agent_dir": agent_dir,
        "waste_flags": _waste_flags(calls, iters, roles),
        "calls": calls,
    }


def _fmt_k(n):
    n = int(n or 0)
    if n >= 1000000:
        return "%.2fM" % (n / 1000000.0)
    if n >= 1000:
        return "%dk" % (n // 1000)
    return str(n)


def _fmt_wall(ms):
    s = int((ms or 0) / 1000)
    if s >= 60:
        return "%dm%02ds" % (s // 60, s % 60)
    return "%ds" % s


def render_markdown(report):
    """渲染 markdown（表头含 `|`，便于终端/中控会话直接消费）。"""
    lines = []
    lines.append("# super 成本分布报告")
    lines.append("")
    lines.append("- 工作区：`%s`" % report["workspace"])
    lines.append("- 事件数：%d ｜ 迭代数：%d" % (report["events"], report["iterations"]))
    t = report["total"]
    lines.append("- **合计**：tokens %s ｜ cost $%.4f ｜ wall %s"
                 % (_fmt_k(t["tokens"]), t["cost_usd"],
                    _fmt_wall(t["wall_ms"])))
    if report["iterations"]:
        lines.append("- 单迭代平均：tokens %s ｜ wall %s"
                     % (_fmt_k(t["tokens"] / max(1, report["iterations"])),
                        _fmt_wall(t["wall_ms"] / max(1, report["iterations"]))))
    lines.append("")

    lines.append("## 按角色（role）")
    lines.append("")
    lines.append("| role | 派发次数 | tokens | 占比 | tokens中位 | cost_usd | wall | 工具调用 | cache命中 | 截断 |")
    lines.append("|---|---|---|---|---|---|---|---|---|---|")
    for role in sorted(report["roles"], key=lambda r: -report["roles"][r]["tokens"]):
        r = report["roles"][role]
        lines.append("| %s | %d | %s | %.1f%% | %s | %.4f | %s | %d | %.1f%% | %d |"
                     % (role, r["calls"], _fmt_k(r["tokens"]), r["share_pct"],
                        _fmt_k(r["tokens_median"]), r["cost_usd"],
                        _fmt_wall(r["wall_ms"]), r["tool_calls"],
                        r["cache_hit_pct"], r["truncated"]))
    lines.append("")

    lines.append("## 按迭代")
    lines.append("")
    lines.append("| 迭代 | cell | tokens | cost_usd | wall | by_role（tokens） |")
    lines.append("|---|---|---|---|---|---|")
    for v in report["iterations_detail"]:
        br = v.get("by_role") or {}
        br_txt = " ".join("%s=%s" % (k, _fmt_k((br[k] or {}).get("tokens")))
                          for k in sorted(br)) or "-"
        cell = (v.get("cell_id") or "")[:36]
        lines.append("| %s | %s | %s | %.4f | %s | %s |"
                     % (v["iteration_id"], cell, _fmt_k(v["tokens"]),
                        v["cost_usd"], _fmt_wall(v["wall_ms"]), br_txt))
    lines.append("")

    if report["top_expensive_iterations"]:
        lines.append("## Top-%d 昂贵迭代" % len(report["top_expensive_iterations"]))
        lines.append("")
        lines.append("| 排名 | 迭代 | tokens | wall |")
        lines.append("|---|---|---|---|")
        for i, v in enumerate(report["top_expensive_iterations"], 1):
            lines.append("| %d | %s | %s | %s |"
                         % (i, v["iteration_id"], _fmt_k(v["tokens"]),
                            _fmt_wall(v["wall_ms"])))
        lines.append("")

    growth = report["context_growth"]
    if growth:
        lines.append("## 上下文膨胀（每次派发的 input_tokens 序列）")
        lines.append("")
        lines.append("| role | 序列（按发生顺序，k） | 峰值 |")
        lines.append("|---|---|---|")
        for role in sorted(growth):
            seq = growth[role]
            peak = max(seq) if seq else 0
            head = " → ".join(_fmt_k(x) for x in seq[:12])
            more = "" if len(seq) <= 12 else " …（共 %d 次）" % len(seq)
            lines.append("| %s | %s%s | %s |" % (role, head, more, _fmt_k(peak)))
        lines.append("")

    if report["session_detail"]:
        s = report["session_detail"]
        lines.append("## pi session 明细（best-effort）")
        lines.append("")
        lines.append("| 项 | 值 |")
        lines.append("|---|---|")
        lines.append("| agent_dir | `%s` |" % report["agent_dir"])
        lines.append("| 匹配 session 文件 | %d |" % s["files"])
        lines.append("| assistant 回合 | %d |" % s["assistant_turns"])
        lines.append("| input tokens | %s |" % _fmt_k(s["input_tokens"]))
        lines.append("| output tokens | %s |" % _fmt_k(s["output_tokens"]))
        lines.append("")
    else:
        lines.append("> pi session 明细未定位（agent_dir/sessions 不可读）——本报告仅用事件流口径。")
        lines.append("")

    if report["waste_flags"]:
        lines.append("## 浪费模式（命中项）")
        lines.append("")
        for f in report["waste_flags"]:
            lines.append("- %s" % f)
        lines.append("")
    else:
        lines.append("> 未命中可机检的浪费模式（清单见 skill:super-cost-analysis）。")
        lines.append("")
    return "\n".join(lines)


def main(argv=None):
    ap = argparse.ArgumentParser(
        description="super 进化 run 的 token 成本分布报告（只读）")
    ap.add_argument("--workdir", required=True, help="super 工作区（含 events.jsonl）")
    ap.add_argument("--json", action="store_true", help="输出 JSON（默认 markdown）")
    ap.add_argument("--top", type=int, default=5, help="Top-N 昂贵迭代（默认 5）")
    args = ap.parse_args(argv)
    ws = args.workdir
    if not os.path.isdir(ws):
        sys.stderr.write("workdir 不存在: %s\n" % ws)
        return 2
    report = build_report(ws, top=max(1, args.top))
    if args.json:
        # calls 体积大，JSON 里也给（分析需要）；确保可序列化
        print(json.dumps(report, ensure_ascii=False, indent=2))
    else:
        print(render_markdown(report))
    return 0


if __name__ == "__main__":
    sys.exit(main())
