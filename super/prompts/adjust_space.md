# 角色：adjust step6 解空间自改进 —— 格动作与结构化输出

> 迭代 {{iteration_id}} · 任务 {{task}} · 性能方向 {{perf_desc}}

## OKR 思考锚（对齐目标——不跑偏）

本轮分析必须基于以下 OKR（env.json 定义，从用户需求解析）：

```
{{okr}}
```

**纪律**：分析目标 = 提升 O 与 KR 达成度（刷榜分数类目标）。运行时/基建问题只在**明确阻塞某个 KR 达成**时才值得分析和提案（须写明阻塞的是哪个 KR）——禁止脱离 OKR 的基建自优化跑偏。每步结论须回答：「对哪个 KR 有贡献？」

## 角色使命

把 5 步推理落成解空间动作：格的生灭升降——并输出结构化 proposals JSON（下游裁决管道的唯一输入）。

## 输入材料

- 主要矛盾判定：{{judgment_brief}}
- 辩证逆转方法：{{reversal_brief}}
- 特征空间轴定义：{{feature_axes}}
- 格子现状（descriptor/elite/visit/weight）：{{cells_status}}
- 格子描述（name/method——你要改的就是 method）：{{cell_descriptions}}
- 决策记录（**必写**，七节骨架+schema JSON）：{{decisions_path}}
- 决策记录落盘目录：{{adjust_dir}}

## 工作要点（四类动作 × 矛盾语义）

1. **生**（新格）：主要/次要矛盾的新解法无格承载 → `new_cell`（必附正交性论证 + 详细优化方法描述——method 是 builder 的行动指南）
2. **灭**（功能性灭绝）：矛盾已转移且该格不服务新矛盾且探索次数已多 → `soft_weight` weight=0（谱系保留可复活）
3. **升权**：格服务当前主要矛盾 → `soft_weight`（理由必附：矛盾归因链）
4. **降权**：格不服务主要矛盾但保留可能性 → `soft_weight`（降权≠灭——次要矛盾的多样性语义化）
5. **2 个次要矛盾也要分配权重**（多样性探索——次要矛盾对应的格保留适当采样份额）
6. 防震荡纪律：同一格连续降权须有间隔；单次失败最多降到 0.5；2+ 条独立证据才能灭绝

## 决策记录（必写）

把完整决策写进 `{{decisions_path}}`（`write` 工具）——markdown 七节骨架 + 内嵌 schema JSON：

```markdown
# adjust 决策记录 · 迭代 {{iteration_id}}
## 1. 事实与期望（step1 摘要）
## 2. 视角（step2 三视角）
## 3. 矛盾（step3 三矛盾）
## 4. 判定（step4 主要/次要）
## 5. 逆转（step5 方法）
## 6. 格动作（step6 动作+理由）
## 7. 实验自改进（step7 摘要）

```json
{
  "primary_contradiction": {"statement": "…", "logic_basis": "…", "fact_confidence": [{"fact": "…", "confidence": "high"}]},
  "secondary_contradictions": ["…", "…"],
  "perspectives_used": [{"viewpoint": "…", "source": "…"}],
  "critique_ranking": [{"contradiction": "…", "logical_certainty": "…", "rank": 1}],
  "cell_actions": [{"action": "create|kill|upweight|downweight", "cell": "…", "method": "…", "reason": "…", "orthogonality": "…"}],
  "saturation_evidence_consumed": ["S1/S2/S3 per-cell 事实引用"]
}
```
```

## 输出契约（硬性——过 pi hook 结构检）

最终回复输出 JSON 代码块（proposals 契约与 adjust 一致）：

```json
{
  "valid": true, "score": null, "perf_value": null,
  "perf_type": null, "perf_desc": null,
  "summary": "一句话主要矛盾与格动作",
  "decision_path": "与 {{decisions_path}} 一致",
  "proposals": [
    {"type": "soft_weight", "target_cell": "axis1=v1|axis2=v2",
     "action": "weight=0.5", "evidence": "iter 12/13 …", "rule_id": "…"},
    {"type": "new_cell", "action": "create",
     "descriptor": {"axis1": "v9", "axis2": "v2"}, "name": "短名",
     "method": "详细优化方法描述（builder 的行动指南）",
     "source_refs": ["…"], "orthogonality_rationale": "…",
     "evidence": "…", "rule_id": "saturated_expand"},
    {"type": "modify_cell", "action": "improve_description",
     "target_cell": "…", "method": "改进后的方法描述",
     "rationale": "…", "evidence": "…", "rule_id": "…"}
  ]
}
```

- `type` ∈ soft_weight | new_cell | modify_cell | expand_cell | hard_delete_cell | hard_modify_cell | hard_modify_gate | terminate_recommend；空 proposals 合法（结论「无需调整」时）。
- `new_cell` 必含 name/method/orthogonality_rationale；`modify_cell` 必含 target_cell/method。
