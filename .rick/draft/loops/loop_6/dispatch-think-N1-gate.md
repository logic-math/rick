# 派发：think subagent — 批判门禁（N1 阶段 human 回答）

**先读**：
- `/workdir/sunquan20/AI_CODING/rick/.rick/draft/loops/loop_6/prompts/think.md` + `skill_think.md`
- `/workdir/sunquan20/AI_CODING/rick/.rick/draft/loops/loop_6/briefs/research-report-N1.md`（系统论描述符 + 稳态 + K1~K7 矛盾）
- `/workdir/sunquan20/AI_CODING/rick/.rick/draft/loops/loop_6/judgment.md`（S + E 收敛判断）

运行时适配同前：你即 think 执行者，直接执行 6 步工作流，不递归派发 subagent。min_assumptions=5，top-N=3。

---

## 阶段：批判门禁（N1 矛盾生成）

**主题**：subagent 在 pi runtime 下触发概率低，优化提示词（使触发确定性提升到上限内最高）。

**待审材料（human 本次回答原话）**：

> 这个本质上就是改造深度与独立性之间的矛盾，在我看来，我们完全偏向与深度改造，不考虑独立性。
> 这个系统运行 3 年，最大的问题就是失去先进性，随着模型迭代，harness 效果变差。
> 我认为 rick 是最核心的定义，如果 rick 节点消息，这个系统就会退化为平庸的 pi agent

**已知事实（供置信度更新）**：
- N1 已产出 K1~K7 七个矛盾状态（K4 = 改造深度 vs 框架独立性）。
- human 回答指向 K4 为主要矛盾方向，并明确「完全偏向深度改造，不考虑独立性」。
- 注：原文「rick 节点消息」疑为「rick 节点**消失**」之误（语境：消失后系统退化为平庸 pi agent），供 human 确认，不影响门禁判断。

**任务**：识别推理（演绎/归纳/溯因）→ 提取假设 → 形式化（"如果 X 那么 Y"）→ 4 维打分 → top-N。若 top-N 中有未澄清的 Y，上报需 human 决策点。

## 交付标准

按 think.md 批判门禁简报格式：假设列表 + top-N 3 问 + 门禁结论（✅/❌）。落盘 `.../loop_6/briefs/批判门禁-N1.md`。

**禁止**：简报含倾向性、替 human 决策 Y、生成价值性假设（只从已有推理提取）、少于 min_assumptions、只 1 种推理类型、每假设只 1 问、确认性句式。

## 返回

批判门禁简报全文作为最终输出返回 sense_loop。
