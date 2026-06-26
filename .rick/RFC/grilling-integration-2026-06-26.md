# RFC: 将 grilling 机制融合到 rick cli（easy 和 plan 命令）

**日期**：2026-06-26  
**状态**：草稿（S 阶段完成，P/J 阶段待补充）  
**记录者**：express subagent

---

## Subject（现状与期望）

### 现状

- "肯定是 doing，doing 是最重的环节，doing 完成后我都是在 learning 中发现 doing 做的不符合我的预期。"
- "都有，做了我不想要，或者没做我想要的都存在。"
- "我认为 plan 生成的就不行，而 plan 生成的不行其背后的原因是，human-loop 执行的就不行。"

### 期望（用户选择的方向）

- "我觉得 grill 这段提示词可以内嵌到 easy 和 plan 里效果会比较不错。"
- "可以将其抽象出来，分别内嵌到 easy 和 plan 的第一步。"
- "让用户澄清自身方案的细节，通过追问与给出推荐答案的方式帮助提升计划的质量。"
- "最后对于 plan 阶段生成 task 分析好的一堆带执行的 plan 文件，对于 easy 来说就是重写 requirement.md 澄清丰富需求与方案即可。"

### 关键设计决策

- "内嵌到同一个会话中"（grilling 和后续执行在同一 Claude 会话里）

---

## Perspective（多视角分析）

[本阶段未完成，需补充]

---

## Judgment（决策）

[本阶段未完成，需补充]

---

## 调研事实

### grilling 提示词

来源：`builder/skills/grilling/SKILL.md`

```
Interview me relentlessly about every aspect of this plan until we reach a shared understanding. 
Walk down each branch of the design tree, resolving dependencies between decisions one-by-one. 
For each question, provide your recommended answer.
Ask the questions one at a time, waiting for feedback on each question before continuing.
If a question can be answered by exploring the codebase, explore the codebase instead.
```

### rick easy 现状

单次 Claude 会话，easy.md 模板，{{requirement}} 变量，TDD 编码，完成后自动触发 learning。

### rick plan 现状

调用 callClaudeCodeCLI()，生成 tasks.json。

---

## 排除的方案

用户未选择以下方案：

- 新增独立 `rick grill` 命令
- 独立前置会话后再启动新会话
- 在 human-loop 中加 L0 层级
- 升级 human-loop think subagent
