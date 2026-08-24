# 派发：think subagent — 批判门禁（N2 阶段 human 回答）

**先读**：
- `/workdir/sunquan20/AI_CODING/rick/.rick/draft/loops/loop_6/prompts/think.md` + `skill_think.md`
- `/workdir/sunquan20/AI_CODING/rick/.rick/draft/loops/loop_6/briefs/think-N2.md`（三维打分 + top-N + human 指向标注）
- `/workdir/sunquan20/AI_CODING/rick/.rick/draft/loops/loop_6/briefs/research-report-N1.md`（K1~K7 + 系统描述符 + 稳态）
- `/workdir/sunquan20/AI_CODING/rick/.rick/draft/loops/loop_6/judgment.md`（S/E/N1 收敛判断）

运行时适配同前：你即 think 执行者，直接执行 6 步工作流，不递归派发 subagent。min_assumptions=5，top-N=3。

---

## 阶段：批判门禁（N2 主要矛盾判断）

**主题**：subagent 在 pi runtime 下触发概率低，优化提示词（使触发确定性提升到上限内最高）。

**待审材料（human 本次回答原话）**：

> 我认为是理解清楚 pi 的触发语言是什么，深度跟 pi 的 subagent 体系结合才行。
> 理解了他的触发语言，这个作为主要的控制手段，我们以此改造 prompt 即可。
> 理解触发语言应该作为关键的控制方法，从 A 到 B 的关键转换其实就是改造深度。 这点毋庸置疑
> 而为了实现这一点，我们应该先从理解 pi 的行为开始。

**已知事实（供置信度更新）**：
- N2 三维打分：K1（触发语言）/ K2（角色命名）/ K4（改造深度）并列满分 3.0。
- human N1 指向 K4，本回答显式确认「从 A 到 B 的关键转换就是改造深度，这点毋庸置疑」→ 选定主要矛盾 = K4。
- human 同时给出控制手段 = 理解 pi 触发语言（K1）+ 以此改造 prompt；方法起点 = 先理解 pi 的行为。

**任务**：识别推理（演绎/归纳/溯因）→ 提取假设 → 形式化（"如果 X 那么 Y"）→ 4 维打分 → top-N。若 top-N 中有未澄清的 Y，上报需 human 决策点。

## 交付标准

按 think.md 批判门禁简报格式：假设列表 + top-N 3 问 + 门禁结论（✅/❌）。落盘 `.../loop_6/briefs/批判门禁-N2.md`。

**禁止**：简报含倾向性、替 human 决策 Y、生成价值性假设（只从已有推理提取）、少于 min_assumptions、只 1 种推理类型、每假设只 1 问、确认性句式。

## 返回

批判门禁简报全文作为最终输出返回 sense_loop。
