#!/usr/bin/env python3
"""esq_report.py — 进化系统质量评估报告（ESQ 四维度指标体系）。

ESQ (Evolutionary System Quality) = D1^0.35 × D2^0.20 × D3^0.15 × D4^0.30
  D1 搜索质量（0.35）: Coverage × Speedup × SampleEff × ConvEff
  D2 自适应能力（0.20）: DecisionAcc × SelfCorrect × BudgetUtil
  D3 知识积累（0.15）: KnowGrowth × TransferRate × PropConsum
  D4 长跑稳定性（0.30）: Monotonic × InfraFail × Recovery

用法：python3 esq_report.py <workspace> [--json]
输出：四维度分数 + ESQ 总分 + 短板分析 + 竞品对比口径
"""
import argparse
import json
import math
import os
import sqlite3
import subprocess
import sys
from typing import Any, Dict, List, Optional, Tuple


def _db(ws):
    return sqlite3.connect(os.path.join(ws, "super.db"))


def _q(conn, sql, default=0):
    try:
        r = conn.execute(sql).fetchone()
        return r[0] if r else default
    except sqlite3.Error:
        return default


def _geomean(values):
    vals = [v for v in values if v > 0]
    if not vals:
        return 0.0
    return math.exp(sum(math.log(v) for v in vals) / len(vals))


def _clamp(v, lo=0.0, hi=1.0):
    return max(lo, min(hi, v))


# ---- D1 搜索质量 ----
def dim1_search(conn, ws) -> Tuple[float, Dict]:
    total_cells = _q(conn, "SELECT count(*) FROM cells")
    covered = _q(conn, "SELECT count(*) FROM cells WHERE best_experiment_id IS NOT NULL")
    coverage = _clamp(covered / total_cells) if total_cells > 0 else 0

    admitted_scores = [r[0] for r in conn.execute(
        "SELECT score FROM experiments WHERE status='admitted' AND score > 0")]
    speedup_geo = _geomean(admitted_scores) if admitted_scores else 0
    speedup_norm = _clamp(speedup_geo / 2.0)  # 封顶 2×=满分

    total_valid = _q(conn, "SELECT count(*) FROM experiments WHERE status IN ('admitted','rejected','failed')")
    admitted = _q(conn, "SELECT count(*) FROM experiments WHERE status='admitted'")
    sample_eff = _clamp(admitted / total_valid) if total_valid > 0 else 0

    # 收敛效率：最近 10 实验的新格发现率
    recent = conn.execute(
        "SELECT cell_id, status FROM experiments ORDER BY iteration DESC LIMIT 20").fetchall()
    recent_cells = set(r[0] for r in recent if r[1] == "admitted")
    old_cells = set(r[0] for r in conn.execute(
        "SELECT DISTINCT cell_id FROM experiments WHERE status='admitted' AND iteration < "
        "(SELECT min(iteration) FROM (SELECT iteration FROM experiments ORDER BY iteration DESC LIMIT 20))"
    ).fetchall()) if recent else set()
    new_discoveries = len(recent_cells - old_cells)
    conv_eff = _clamp(new_discoveries / 10.0) if recent else 0

    d1 = _geomean([coverage, speedup_norm, sample_eff, conv_eff])
    return d1, {
        "coverage": f"{coverage:.1%} ({covered}/{total_cells})",
        "speedup_geomean": f"{speedup_geo:.3f}× (norm {speedup_norm:.3f})",
        "sample_efficiency": f"{sample_eff:.1%} ({admitted}/{total_valid})",
        "convergence_efficiency": f"{new_discoveries} new cells in last 20 exps (norm {conv_eff:.3f})",
        "D1": round(d1, 4),
    }


# ---- D2 自适应能力 ----
def dim2_adaptivity(conn, ws) -> Tuple[float, Dict]:
    weight_changes = conn.execute(
        "SELECT json_extract(payload,'$.cell_id'), json_extract(payload,'$.old_weight'), "
        "json_extract(payload,'$.new_weight') FROM events WHERE event_type='weight_changed' "
        "ORDER BY seq").fetchall()

    # 决策准确率：升权格后续是否有 admitted 产出
    upweighted_cells = set()
    correct_ups, total_ups = 0, 0
    for cell_id, old_w, new_w in weight_changes:
        if new_w and old_w and float(new_w) > float(old_w):
            upweighted_cells.add(cell_id)
            total_ups += 1
            # 检查该格是否有 admitted（D2 修复：参数化查询传实际 cell_id）
            try:
                has_admit = conn.execute(
                    "SELECT count(*) FROM experiments WHERE cell_id=? AND status='admitted'",
                    (cell_id,)).fetchone()[0]
            except Exception:
                has_admit = 0
            if has_admit > 0:
                correct_ups += 1
    decision_acc = _clamp(correct_ups / total_ups) if total_ups > 0 else 0.5

    # 自纠错率：升权→降权回调的次数 / 总升权次数
    self_corrections = sum(1 for _, old_w, new_w in weight_changes
                           if old_w and new_w and float(new_w) < float(old_w)
                           and float(old_w) > 1.0)
    self_correct = _clamp(self_corrections / total_ups) if total_ups > 0 else 0

    # 预算利用率：admitted token / total token
    total_tokens = _q(conn, "SELECT sum(json_extract(payload,'$.total_tokens')) "
                            "FROM events WHERE event_type='agent_ended'") or 0
    admitted_tokens = _q(conn, "SELECT sum(json_extract(payload,'$.total_tokens')) "
                               "FROM events WHERE event_type='agent_ended' AND json_extract(payload,'$.role')='builder'") or 0
    budget_util = _clamp(admitted_tokens / total_tokens) if total_tokens > 0 else 0.5

    d2 = _geomean([decision_acc, self_correct, max(budget_util, 0.1)])
    return d2, {
        "decision_accuracy": f"{decision_acc:.1%} ({correct_ups}/{total_ups} ups productive)",
        "self_correction": f"{self_corrections} callbacks / {total_ups} ups ({self_correct:.3f})",
        "budget_utilization": f"{budget_util:.1%}",
        "D2": round(d2, 4),
    }


# ---- D3 知识积累 ----
def dim3_knowledge(conn, ws) -> Tuple[float, Dict]:
    dk_path = os.path.join(ws, "env", "domain_knowledge.md")
    dk_bytes = os.path.getsize(dk_path) if os.path.isfile(dk_path) else 0
    know_growth = _clamp(dk_bytes / 51200.0)  # 封顶 50KB=满分

    # 迁移率：domain_knowledge 的「调整实验知识沉淀」节数（每轮 adjust 追加一节）
    adjust_rounds = _q(conn, "SELECT count(*) FROM adjusts")
    dk_sections = 0
    if os.path.isfile(dk_path):
        dk_sections = open(dk_path, encoding="utf-8").read().count("## adjust 实验知识沉淀")
    transfer_rate = _clamp(dk_sections / max(adjust_rounds, 1))

    # 提案消费率
    total_props = _q(conn, "SELECT count(*) FROM humans WHERE queue_type='proposal'")
    applied_props = _q(conn, "SELECT count(*) FROM humans WHERE queue_type='proposal' AND applied=1")
    prop_consum = _clamp(applied_props / total_props) if total_props > 0 else 0

    d3 = _geomean([know_growth, max(transfer_rate, 0.1), max(prop_consum, 0.02)])
    return d3, {
        "knowledge_growth": f"{dk_bytes} bytes ({dk_sections} sections, norm {know_growth:.3f})",
        "transfer_rate": f"{dk_sections}/{adjust_rounds} rounds ({transfer_rate:.3f})",
        "proposal_consumption": f"{applied_props}/{total_props} ({prop_consum:.1%})",
        "D3": round(d3, 4),
    }


# ---- D4 长跑稳定性 ----
def dim4_stability(conn, ws) -> Tuple[float, Dict]:
    # 单调性：per-iteration max(score) 不退化的比例
    iter_scores = conn.execute(
        "SELECT iteration, max(score) FROM experiments WHERE status='admitted' "
        "GROUP BY iteration ORDER BY iteration").fetchall()
    if len(iter_scores) < 2:
        monotonic = 0.5
    else:
        non_decreasing = sum(1 for i in range(1, len(iter_scores))
                             if iter_scores[i][1] >= iter_scores[i-1][1] * 0.8)  # 允许20%带
        monotonic = _clamp(non_decreasing / (len(iter_scores) - 1))

    # 基础设施失败率
    infra_fails = _q(conn, "SELECT count(*) FROM events WHERE event_type='gate_result' "
                           "AND json_extract(payload,'$.failure_stage') IN "
                           "('remote_link','environment_oom','env_misroute')")
    total_fails = _q(conn, "SELECT count(*) FROM events WHERE event_type='gate_result' "
                           "AND json_extract(payload,'$.passed')=0")
    infra_fail_inv = _clamp(1 - (infra_fails / total_fails)) if total_fails > 0 else 1.0

    # 恢复能力：interrupted 后下一个 admitted 的间隔
    interrupted = _q(conn, "SELECT count(*) FROM experiments WHERE status='interrupted'")
    total_exps = _q(conn, "SELECT count(*) FROM experiments")
    recovery = _clamp(1 - (interrupted / max(total_exps, 1)) + 0.3) if interrupted > 0 else 1.0

    d4 = _geomean([monotonic, infra_fail_inv, _clamp(recovery)])
    return d4, {
        "monotonic_improvement": f"{monotonic:.1%}",
        "infra_fail_rate": f"{infra_fails}/{total_fails} ({1-infra_fail_inv:.1%} infra)",
        "recovery_capability": f"{interrupted}/{total_exps} interrupted (norm {recovery:.3f})",
        "D4": round(d4, 4),
    }


def compute_esq(ws: str) -> Dict[str, Any]:
    conn = _db(ws)
    d1, d1_detail = dim1_search(conn, ws)
    d2, d2_detail = dim2_adaptivity(conn, ws)
    d3, d3_detail = dim3_knowledge(conn, ws)
    d4, d4_detail = dim4_stability(conn, ws)

    w1, w2, w3, w4 = 0.35, 0.20, 0.15, 0.30
    esq = (d1 ** w1) * (d2 ** w2) * (d3 ** w3) * (d4 ** w4)

    # 短板分析
    dims = {"D1_search": d1, "D2_adaptivity": d2,
            "D3_knowledge": d3, "D4_stability": d4}
    weakest = min(dims, key=dims.get)
    strongest = max(dims, key=dims.get)

    conn.close()
    return {
        "ESQ": round(esq, 4),
        "weights": {"D1": w1, "D2": w2, "D3": w3, "D4": w4},
        "dimensions": dims,
        "details": {"D1": d1_detail, "D2": d2_detail,
                    "D3": d3_detail, "D4": d4_detail},
        "weakest_dimension": weakest,
        "strongest_dimension": strongest,
        "improvement_priority": [
            f"短板：{weakest}（{dims[weakest]:.3f}）——优先改进此维度",
            f"强项：{strongest}（{dims[strongest]:.3f}）",
        ],
    }


def render_report(esq: Dict) -> str:
    lines = [
        "# ESQ 进化系统质量报告",
        "",
        f"**ESQ = {esq['ESQ']:.4f}**（满分 1.0）",
        "",
        "## 四维度分数",
        "",
        "| 维度 | 权重 | 分数 | 说明 |",
        "|---|---|---|---|",
    ]
    dim_names = {
        "D1_search": "搜索质量", "D2_adaptivity": "自适应能力",
        "D3_knowledge": "知识积累", "D4_stability": "长跑稳定性",
    }
    for key, name in dim_names.items():
        w = esq["weights"].get(key[:2], 0)
        score = esq["dimensions"][key]
        lines.append(f"| {name} | {w:.0%} | **{score:.4f}** | {key} |")

    lines += ["", "## 维度详情", ""]
    for dk in ("D1", "D2", "D3", "D4"):
        lines.append(f"### {dk}")
        for k, v in esq["details"][dk].items():
            lines.append(f"- **{k}**: {v}")
        lines.append("")

    lines += [
        "## 短板分析",
        "",
        f"- 🎯 **优先改进**: {esq['weakest_dimension']}（{esq['dimensions'][esq['weakest_dimension']]:.3f}）",
        f"- 💪 **当前强项**: {esq['strongest_dimension']}（{esq['dimensions'][esq['strongest_dimension']]:.3f}）",
        "",
        "## 竞品对比口径",
        "",
        "```",
        "与 one-shot LLM 对比：只比 D1（Coverage + Speedup）——架构增益 = 进化系统D1 / one-shot D1",
        "与 KernelFoundry 对比：D1 同口径（KernelBench 子集）+ D2/D3/D4 为我们的架构优势",
        "ESQ 自身作为新指标定义（填补空白）——开源计算脚本，邀请社区同协议跑",
        "```",
    ]
    return "\n".join(lines)


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument("workspace", help="super 工作区路径（含 super.db）")
    ap.add_argument("--json", action="store_true", help="输出 JSON 而非 markdown")
    args = ap.parse_args()

    if not os.path.isfile(os.path.join(args.workspace, "super.db")):
        print("Error: super.db not found in workspace", file=sys.stderr)
        return 1

    esq = compute_esq(args.workspace)
    if args.json:
        print(json.dumps(esq, ensure_ascii=False, indent=2))
    else:
        print(render_report(esq))
    return 0


if __name__ == "__main__":
    sys.exit(main())
