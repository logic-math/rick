# 派发：think subagent — 批判门禁（S 阶段 human 回答 · 第 2 轮）

**先读**：
- `/workdir/sunquan20/AI_CODING/rick/.rick/draft/loops/loop_6/prompts/think.md` + `skill_think.md`
- `/workdir/sunquan20/AI_CODING/rick/.rick/draft/loops/loop_6/briefs/批判门禁-S.md`（第 1 轮门禁：7 假设 + top-N + ❌ 结论 + 5 未澄清点）
- `/workdir/sunquan20/AI_CODING/rick/.rick/draft/loops/loop_6/briefs/research-report-S-reasons-agent.md`（A/B/C 三分支事实，用于更新置信度）
- `/workdir/sunquan20/AI_CODING/rick/.rick/draft/loops/loop_6/briefs/research-report-S-bestpractice.md`（BP-1~BP-9）

运行时适配同前：你即 think 执行者，直接执行 6 步工作流，不递归派发 subagent。min_assumptions=5，top-N=3。

---

## 阶段：批判门禁（S 问题确认 · 第 2 轮）

**主题**：subagent 在 pi runtime 下触发概率低，优化提示词。

**待审材料（human 对第 1 轮 5 个未澄清点的回答，原话）**：

> 1. 我认为这个假设是成立的,可以继续推进
> 2. 是的，应改为提升到上限内最高
> 3. 其实没有排除，有可能是这个原因，需要实际改正完后再验证，或者你可以调研一下有什么事实可以佐证，可以搜索确认。
> 4. 我认为都是提示词未对齐 pi 的触发机制，这个主要原因。其他原因你可以调研一下，帮我确认。
> 5. 是的，关键应该是按 pi 的方式将其中几个 subagent 直接内置未系统级的 agent，定义好这些 agent，然后用明确的提示词进行触发。这样可以保证触发概率。这个可以调研一下

**第 1 轮 5 个未澄清点的对应关系**：
- 第 1 点 → A1 因果归属（缺口是主要原因）
- 第 2 点 → A2 「最大化」可达性
- 第 3 点 → A6 模型能力边界
- 第 4 点 → A5 共性根因
- 第 5 点 → A4 最佳实践 + 方向（系统级 agent）

**调研结果（用于更新置信度，事实性）**：
- A2 配置层已排除：扩展已注册、subagent 工具可用、默认 full 工具描述、无禁用项（高置信）。
- A1 模型 tool-calling 能力差异：本地无 benchmark 信源，未验证 → R7。
- 分支 B：自定义 agent = frontmatter markdown，注册作用域 builtin/package/user/project，触发用 `runs.run(agent:'name')`；rick 现状零自定义 agent、think/research/exporter 仅为普通 markdown（高置信）。
- 分支 C：pi 官方要求显式 workflowScript + agent 名；官方旁证「软性指令不可靠、需显式提示」（高置信）。

## 任务

重新执行 6 步工作流：识别推理 → 提取假设 → 形式化 → 4 维打分（置信度用调研结果更新）→ top-N → 门禁结论。

**门禁结论**：✅ 通过（top-N 假设的 Y 已澄清或显式确认）或 ❌ 未通过（列出仍未澄清的 Y + 需 human 回答什么）。

## 交付标准

按 think.md 批判门禁简报格式：假设列表 + top-N 3 问 + 门禁结论。落盘 `.../loop_6/briefs/批判门禁-S-r2.md`。

**禁止**：简报含倾向性、替 human 决策 Y、生成价值性假设、少于 min_assumptions、只 1 种推理类型、每假设只 1 问、确认性句式。

## 返回

批判门禁简报全文作为最终输出返回 sense_loop。
