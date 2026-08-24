# 派发：think subagent — 批判门禁（S-R 第 2 轮）

**先读**：
- `/workdir/sunquan20/AI_CODING/rick/.rick/draft/loops/loop_6/prompts/think.md` + `skill_think.md`
- `/workdir/sunquan20/AI_CODING/rick/.rick/draft/loops/loop_6/briefs/批判门禁-SR.md`（第 1 轮：❌ + 4 未澄清点）
- `/workdir/sunquan20/AI_CODING/rick/.rick/draft/loops/loop_6/judgment.md`（S/E/N/S-R 全部收敛 + 本轮澄清）

运行时适配同前：你即 think 执行者，直接执行 6 步工作流，不递归派发 subagent。min_assumptions=5，top-N=3。

---

## 阶段：批判门禁（S-R 第 2 轮）

**主题**：subagent 在 pi runtime 下触发概率低，优化提示词。

**待审材料（human 对第 1 轮 4 个未澄清点的回答，原话）**：

> 1. 近似测试通过了所有的功能验收，就算是一致的。 自然语言表达力是有的，可以认为是可以无歧义的表达的，自然语言描述总是会进行信息压缩的，但只要把关键信息描述正确，就可以刻画功能，只要功能等价，就认为是效果等价的。
> 2. 这是一个必然会发生的事实，模型现在就有能力，这是一个可以悬置的假设。
> 3. 嗯不会随意切换，只会保留一个runtime，只是未来 dsh 可以承接更深度的定制化开发，这个时候我们才需要做架构升级。

**第 1 轮 4 个未澄清点对应关系**：
- 第 1 点 → ① A4 核心假设（等价=功能等价；自然语言可无歧义；信息压缩可接受）
- 第 2 点 → ② A1 未来预测（必然事实，可悬置）
- 第 3 点 → ③ A7 tools 隐含 + ④ A3 随意切换（不随意切换、只保留一个 runtime；dsh 更深定制化时才架构升级）

**任务**：重新执行 6 步工作流：识别推理 → 提取假设 → 形式化 → 4 维打分（置信度用澄清结果更新）→ top-N → 门禁结论。

**门禁结论**：✅ 通过（top-N 假设的 Y 已澄清或显式确认）或 ❌ 未通过（列出仍未澄清的 Y + 需 human 回答什么）。

## 交付标准

按 think.md 批判门禁简报格式：假设列表 + top-N 3 问 + 门禁结论。落盘 `.../loop_6/briefs/批判门禁-SR-r2.md`。

**禁止**：简报含倾向性、替 human 决策 Y、生成价值性假设、少于 min_assumptions、只 1 种推理类型、每假设只 1 问、确认性句式。

## 返回

批判门禁简报全文作为最终输出返回 sense_loop。
