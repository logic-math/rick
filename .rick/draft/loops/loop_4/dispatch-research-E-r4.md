# 派发：research subagent — E-r4 并行调研（zero-shot 对比）

E 阶段批判门禁 r4 期间并行调研（human 显式提的调研请求，不中断主流程）。

**先读**（如未在上下文）：
- `loop_4/prompts/skill_research.md`、`loop_4/prompts/research.md`（角色与工作流）
- `loop_4/briefs/research-report-E-r2.md`（上一轮 research，复用其信源经验：WebFetch/WebSearch 被拦截改用 curl+arxiv REST；raw API 被 proxy 拒绝、claude CLI 无 temperature flag）
- `loop_2/judgment.md`（rick/pi 背景）

无 `.rick/config.json`，信源权重默认：代码 0.4 / 运行时 0.3 / 文档 0.2 / 反事实 0.1；高置信 ≥0.8。运行时适配同前：你即调研执行者，可直接用 Read/Grep/Bash/WebFetch/WebSearch，保留尽调树/MECE/加权/R7/落盘/`git restore` 全部约束。

---

## human 调研请求（原话）

> 需要你调研一下，与 zero shot 对比。 rick 的方法可能包含在了训练数据中，但作为通用的模型它并不会选择用这个方法解决问题。 rick 本身确定的选择了如此做事。 它本身已经包含了判断在内。

## 任务（尽调树，MECE 划分）

验证以下事实陈述，建议根节点下第一层（可自行调整）：

- **节点 A**：rick 的方法（doing/learning/dream、sense 5 阶段、plan-do-learn、debug 循环、act-path）是否**在 LLM 训练数据 G 中**？（即这些方法是否被通用工程实践/公开语料隐式覆盖）
  - 信源：文档（plan-do-learn、TDD、sense/dialectical 方法论是否公开语料常见）；代码原文（rick 仓库 doing.md/sense_loop.md 等是否独创编排 vs 通用实践组合）
- **节点 B**：通用 LLM 在 **zero-shot**（无 rick 注入、无 doing.md/sense 系统提示词）下，是否**会选择**用 rick 的方法解决问题？还是会用其他更"默认"的方式（直接答、单轮 CoT、不迭代）？
  - 信源：运行时（如可行：同一任务，对比 bare LLM 调用 vs rick 注入调用，观察是否自发出现 plan-do-learn/sense 多阶段迭代；复用 E-r2 的 runtime 经验，若 API/CLI 受限则记录并降级）；文档（zero-shot vs methodology-injection 文献；prompt-engineering vs 系统性方法论注入的差异）
- **节点 C**：human 论断"rick 确定性地选择此方法（含判断）"——"确定性选择"是否是 rick 相对 zero-shot 的关键差异？即 rick 注入是否把"可选的方法"变成"确定被执行的方法"？
  - 信源：代码原文（rick 系统提示词注入机制——doing/sense 是否被强制注入；ContextManager/GenerateDoingPromptFile）；文档（deterministic methodology injection vs stochastic model choice）
- **节点 D**（关联核心假设）：LLM 对"未见的 G′ 问题"是否"无法一次性解决"——zero-shot 单轮 vs 多轮迭代在解决未见问题上的差异？
  - 信源：文档（in-context learning 局限、iterative problem-solving 文献、self-refine/Reflexion 等"迭代能否解决单轮无法解决的问题"的实证）；运行时（如可行：给 LLM 一个 rick 训练数据外的冷门任务，单轮 vs 多轮迭代观察）

## 安全约束

- 修改代码必须 `git restore` 还原
- 运行程序优先只读命令；复用 E-r2 已验证的 runtime 路径（claude CLI runtime demo 可用，但无 temperature flag）
- 可复用 loop_2/loop_4 已有 research brief 作为信源（标注来源）

## 交付标准

按 research.md"主报告格式"：信源配置 + 尽调树快照 + 节点详情（置信度+各信源 ✅/❌+证据+疑问点）+ R7 上报 + 整合摘要 + **对 human 请求的直接回答**：
- rick 方法是否在 G 中？
- 通用 LLM zero-shot 是否会选择用此方法？
- "确定性选择（含判断）"是否是 rick vs zero-shot 的关键差异？
- LLM 对未见 G′ 是否"无法一次性解决"（单轮 vs 迭代差异）？

## 产物写入

- 主报告：`loop_4/briefs/research-report-E-r4.md`
- 节点详情：`loop_4/briefs/research-E-r4-{A,B,C,D}.md`

## 返回

整合摘要即为最终输出（尽调树快照 + R7 清单 + 对 human 请求的直接回答）。
