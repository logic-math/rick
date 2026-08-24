# 派发：research subagent — S 追加调研（触发概率低的其他原因 + pi 自定义 agent 机制）

**先读**：
- `/workdir/sunquan20/AI_CODING/rick/.rick/draft/loops/loop_6/prompts/research.md` + `skill_research.md`
- `/workdir/sunquan20/AI_CODING/rick/.rick/draft/loops/loop_6/briefs/research-report-S.md`（N3.1/N3.2 已证实缺口）
- `/workdir/sunquan20/AI_CODING/rick/.rick/draft/loops/loop_6/briefs/research-report-S-bestpractice.md`（BP-1~BP-9 最佳实践）

运行时适配同前：你即调研执行者，直接 Read/Grep/Bash/WebFetch/WebSearch，不再递归派发 subagent，保留 MECE/信源加权/R7/落盘约束。

---

## 五派发要素

**阶段**：S 问题确认 · 追加调研（human 追问）

**主题**：① 触发概率低除「提示词未对齐 pi 触发机制」外还有哪些原因；② pi 自定义/系统级 agent 的机制（能否把 think/research/exporter 等内置为系统级 agent，用明确提示词触发）。

**草稿路径**：`.../draft` | rfc：`.../draft/rfc` | 会话：`.../draft/loops/loop_6`

**前序判断**（human 原话，已确认）：
- A1 因果归属成立（结构性缺口是主要原因），继续推进。
- A2 期望改为「触发确定性提升到上限内最高」（非"最大化"）。
- A5 各命令低触发是同一根因（提示词未对齐 pi 触发机制）；其他原因需调研确认。
- A4 最佳实践存在；方向：按 pi 方式把几个 subagent 直接内置为系统级 agent，定义好 agent，用明确提示词触发，以保证触发概率。
- A6 模型能力/pi 配置等因素尚未排除，需改正后验证，或调研佐证事实。

**任务派发**：按尽调树 MECE 划分，调研以下三个分支，逐叶节点信源加权验证，产出事实 + R7。

**结果核验**：三分支的事实清单（每条含信源 + 置信度）+ 与 rick 现状的关联 + R7 上报项。

---

## 调研要点（三分支）

### 分支 A：除「提示词未对齐」外，还有哪些因素会影响 subagent 触发概率
1. **模型 tool-calling 能力差异**：不同模型（deepseek / claude / gemini / gpt 等）对函数调用/工具调用的遵循度差异，是否影响 subagent 工具被调用的概率。（可 WebSearch 佐证：如 tool-calling 遵循度 benchmark、agentic tool use 可靠性研究。）
2. **pi 运行时/配置因素**：`toolDescriptionMode`（full/compact/custom）、工具描述文本长度/截断、`toolBudget`/`tool allowlist` 是否含 subagent、skill 是否加载、context 压缩（compaction）是否丢失工具说明、`subagent` 工具是否被 disable 等。（Read pi-subagents 源码 + pi 文档 + rick 的 settings.json / tools_init_pi.go。）
3. **提示词本身的结构性因素**（非"对齐"层面）：自然语言触发词是否过于模糊、触发条件是否写了"当 X 时用 subagent"这类模型易忽略的软条件、提示词是否被压缩截断丢失 subagent 段。

### 分支 B：pi 自定义 agent / 系统级 agent 机制
1. **自定义 agent 定义方式**：pi-subagents 如何定义自定义 agent（agents/*.md frontmatter 的 name/description/model/tools/context 等字段？`{action:"create"}` 配置？`config.json` 的 agents 段？）。读 `pi-subagents/skills/pi-subagents/SKILL.md` + `references/*.md` + `src/agents/*.ts` + pi 文档（docs/subagents.md / docs/models.md / docs/configuration 若存在）。
2. **系统级 vs 项目级 vs 用户级注册**：agent 注册的作用域与加载路径（user / project / session）。
3. **如何用明确提示词触发自定义 agent**：定义好 agent 后，parent 用 `runs.run(key, {agent:'自定义名', task:'...'})` 触发？提示词中如何显式引导模型调用？
4. **rick 现状对照**：`internal/prompt/human_loop_prompt.go` 已生成 think/research/exporter 三个 subagent prompt 文件（落盘路径、用途）——这些 prompt 文件能否/如何注册为 pi 自定义 agent？rick 是否有现成的 agent 注册配置（.rick/config.json / pi settings）？

### 分支 C：佐证「缺口是主要原因」的旁证
1. rick 150 处自然语言 subagent 术语、0 处触发语法/agent 名（已证实 N3.1/N3.2）——是否有同类项目/文档指出「自然语言触发不可靠、显式工具调用才可靠」的佐证（可 WebSearch）。
2. pi 官方是否明确「提示词中描述 subagent 时应写显式语法/agent 名」（BP-9 已证 tool description 是模型认知来源）——补充分支 A 的第 2 条。

## 产出与落盘

主报告：`/workdir/sunquan20/AI_CODING/rick/.rick/draft/loops/loop_6/briefs/research-report-S-reasons-agent.md`

**禁止**：简报含倾向性、替 human 判断、无事实支撑构建选项、跳过 MECE。

## 返回

三分支事实清单 + 与 rick 现状关联 + R7 上报项，作为最终输出返回 sense_loop。
