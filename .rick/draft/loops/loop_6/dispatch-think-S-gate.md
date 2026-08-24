# 派发：think subagent — 批判门禁（S 阶段 human 回答）

**先读**：
- `/workdir/sunquan20/AI_CODING/rick/.rick/draft/loops/loop_6/prompts/think.md` + `skill_think.md`
- `/workdir/sunquan20/AI_CODING/rick/.rick/draft/loops/loop_6/briefs/research-report-S.md`（S 阶段已证实事实，供更新置信度参考）

运行时适配同前：你即 think 执行者，直接执行 think.md 的 6 步工作流，不再递归派发 subagent。min_assumptions 默认 5，top-N 默认 3。

---

## 阶段：批判门禁（S 问题确认）

**主题**：subagent 在 pi runtime 下触发概率低，优化提示词。

**待审材料（human 本次回答原话）**：

> 现状中 应该调研一下 pi 触发 subagent 搭建工作流的最佳实践是什么
> 期望目标，其实就是通过改动 rick 现有的框架，优化提示词和某些配置，使得 subagent 的触发确定性最大化。以便于让模型更能遵守提示词的约束。
> 差距在于这个最佳实践以及 rick 的使用现状之间的调研，我需要了解到。

**任务**：识别推理过程（演绎/归纳/溯因）→ 提取假设 → 形式化（"如果 X 那么 Y"）→ 4 维打分 → 选 top-N。若 top-N 中有未澄清的 Y，上报需 human 决策点。

**已知事实（research-report-S.md，供置信度更新）**：
- N3.1：rick 模板零引用 pi 触发语法（workflowScript/runs.run/runs.all）。
- N3.2：rick 模板零引用 pi 内置 agent 名（scout/worker/reviewer/researcher/delegate/oracle）。
- N3.4：结构性缺口已证实，但"缺口⇒触发概率低"的因果归属属假设（R7）。

## 交付标准

按 think.md「批判门禁」简报格式：
- 假设列表（按最终分降序，含推理类型/形式化/4 维打分/期望分/最终分）
- top-N 假设的 3 启发性问题（信念/前提/反例，每假设 3 问）
- **门禁结论**：✅ 通过（top-N 假设的 Y 已澄清或显式确认）或 ❌ 未通过（列出未澄清的 Y + 需 human 回答什么）

**禁止**：简报含倾向性、替 human 决策 Y 是否成立、生成价值性假设（只从已有推理提取）、少于 min_assumptions 就输出 top-N、只 1 种推理类型、每假设只 1 问、确认性句式。

## 产物落盘

`/workdir/sunquan20/AI_CODING/rick/.rick/draft/loops/loop_6/briefs/批判门禁-S.md`

## 返回

批判门禁简报全文（假设列表 + top-N 3 问 + 门禁结论）作为最终输出返回 sense_loop。
