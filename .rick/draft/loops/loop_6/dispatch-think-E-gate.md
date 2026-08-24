# 派发：think subagent — 批判门禁（E 阶段 human 回答）

**先读**：
- `/workdir/sunquan20/AI_CODING/rick/.rick/draft/loops/loop_6/prompts/think.md` + `skill_think.md`
- `/workdir/sunquan20/AI_CODING/rick/.rick/draft/loops/loop_6/briefs/think-E.md`（V1~V6 打分 + top-N）
- `/workdir/sunquan20/AI_CODING/rick/.rick/draft/loops/loop_6/briefs/research-report-E.md`（视角候选事实）
- `/workdir/sunquan20/AI_CODING/rick/.rick/draft/loops/loop_6/judgment.md`（S 收敛判断）

运行时适配同前：你即 think 执行者，直接执行 6 步工作流，不递归派发 subagent。min_assumptions=5，top-N=3。

---

## 阶段：批判门禁（E 视角生成）

**主题**：subagent 在 pi runtime 下触发概率低，优化提示词（使触发确定性提升到上限内最高）。

**待审材料（human 本次回答原话）**：

> 1. 我觉得这就是一个 协议对齐问题，pi 有自己的最佳实践，rick 有自己的现状。二者迁移后存在兼容性问题，既然我们打算在 pi 框架上长期发展，我们就要高度定制化的改造 rick 的提示词。 因此我们需要基于 pi 的 subagent 最佳触发实战去干燥，这是一个协议兼容的问题。
> 2. 不同的领域我认为就是，类比两个不同的语言体系，二者在各自的领域都是自洽的但合作的时候就会暴露问题。

**已知事实（供置信度更新）**：
- V5 协议对齐是 top-N 之一（最终分 0.625，与 V2 并列第 2）。
- BP-9：模型对 subagent 工具的认知来源 = tool description（协议规范）；D7：rick 提示词与 tool description 语法/agent 名不对齐。
- human 明确选定 V5 方向 + 「两个语言体系」类比 + 「高度定制化改造 rick 提示词」动作。

**任务**：识别推理（演绎/归纳/溯因）→ 提取假设 → 形式化（"如果 X 那么 Y"）→ 4 维打分 → top-N。若 top-N 中有未澄清的 Y，上报需 human 决策点。

## 交付标准

按 think.md 批判门禁简报格式：假设列表 + top-N 3 问 + 门禁结论（✅/❌）。落盘 `.../loop_6/briefs/批判门禁-E.md`。

**禁止**：简报含倾向性、替 human 决策 Y、生成价值性假设（只从已有推理提取）、少于 min_assumptions、只 1 种推理类型、每假设只 1 问、确认性句式。

## 返回

批判门禁简报全文作为最终输出返回 sense_loop。
