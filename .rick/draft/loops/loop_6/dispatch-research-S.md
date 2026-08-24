# 派发：research subagent — S 问题确认（调研现状事实）

**先读**（进入本任务前）：
- `/workdir/sunquan20/AI_CODING/rick/.rick/draft/loops/loop_6/prompts/research.md`（角色与工作流）
- `/workdir/sunquan20/AI_CODING/rick/.rick/draft/loops/loop_6/prompts/skill_research.md`（尽调树+信源加权方法论）

运行时适配（同 loop_4）：你即调研执行者，可直接用 Read / Grep / Bash / WebFetch / WebSearch 直接调研，同时保留尽调树 / MECE / 信源加权 / R7 上报 / 落盘 / `git restore` 全部约束。**不再递归派发 subagent**（上下文隔离由你直接执行调研动作替代）。无 `.rick/config.json` 时信源权重默认：代码原文 0.4 / 运行时行为 0.3 / 文档 0.2 / 反事实 0.1，高置信 ≥ 0.8。

---

## 五派发要素

**阶段**：S 问题确认（Subject：还原 + 追问现状事实）

**主题**：subagent 在 pi runtime 下触发概率比较低，想要优化提示词。

**草稿路径**：`/workdir/sunquan20/AI_CODING/rick/.rick/draft` | rfc：`.../draft/rfc` | 本次会话：`.../draft/loops/loop_6`

**前序判断**（human 已确认，原话）：
1. human-loop 命令内部的三个 subagent（think / research / exporter）触发概率低。
2. plan / doing / easy / learning / dream / ctrl 等命令，在提示词中已明确说要用 subagent 的情况下，触发概率也是比较低的。

**任务派发**：调研现状事实，用尽调树。根节点 = 主题「subagent 在 pi runtime 下触发概率低（优化提示词）」，按 MECE 划分第一层子节点（自行选择维度，例如「命令/场景」「提示词如何要求 subagent」「pi runtime 的 subagent 触发机制」「main agent 执行行为」等），逐叶节点用信源加权验证，直至置信度达高或 R7 上报。

**结果核验（交付标准）**：按 research.md 的 S 阶段简报格式输出——尽调树快照（标注每个叶节点置信度）+ 节点详情 + R7 上报项 + 整合摘要。追加 sense_loop S 阶段三连启发性追问（见下）。

---

## 具体调研要点（供 MECE 划分参考，非穷举）

1. **各命令提示词如何要求 subagent**（代码原文 + 文档信源）：
   - human-loop：`internal/prompt/human_loop_prompt.go`（think/research/exporter subagent prompt 的生成与落盘）+ `internal/prompt/templates/sense_loop.md / research.md / think.md / exporter.md`（这些模板内如何描述"派发 subagent"）。核实：main agent 收到的提示词里，触发 subagent 的指令长什么样、有无明确触发条件。
   - plan：`internal/prompt/plan_prompt.go` + `templates/plan.md`（六维评审 subagent_1~6，"每个 subagent 独立启动，串行执行"）。
   - dream：`internal/prompt/dream_prompt.go` + `templates/dream.md`（subagent_1~4）。
   - doing / easy / ctrl / learning：`internal/prompt/doing_prompt.go / easy_prompt.go / ctrl_prompt.go`，learning 的 prompt 来源需核实（cmd/learning.go 引用哪个 prompt builder）。
2. **pi runtime 的 subagent 触发机制**（代码 + 文档信源）：
   - `internal/agent/piagent/`（executor / agentdir）——rick 迁移到 pi 后如何发起会话、subagent 能力是否/如何透传。
   - pi 运行时文档/技能：`/home/hadoop-recsys/.rick/pi/agent/npm/node_modules/pi-subagents/skills/pi-subagents/SKILL.md`、`/home/hadoop-recsys/.rick/pi/agent/runtime/node_modules/@earendil-works/pi-coding-agent/docs/subagents.md`（若存在）——subagent 在 pi 下的触发条件、工具形态、约束。
3. **提示词与 runtime 之间的缺口证据**（反事实/运行时信源，可运行只读命令）：提示词要求用 subagent，但 pi runtime 下 main agent 触发概率低——找出可验证的缺口（例如：提示词用"派发 subagent"这种自然语言，而 pi 的 subagent 触发有硬性工具/配置前提；或提示词未给出触发 subagent 所需的显式工具名/参数）。

## 产出与落盘

- 主报告：`/workdir/sunquan20/AI_CODING/rick/.rick/draft/loops/loop_6/briefs/research-report-S.md`
- 节点详情按需：`.../loop_6/briefs/research-S-{N}-{节点路径}.md`
- R7 上报项（无法达高置信度的叶节点）单独列出理由，供 human 决策。

**禁止**：简报含倾向性、替 human 判断、无事实支撑构建选项、跳过 MECE、subagent 私自计算置信度（置信度只在整合处算）。

## 返回

S 阶段简报全文（尽调树快照 + 节点详情 + R7 上报项 + 整合摘要 + 三连启发性追问），作为最终输出返回给 sense_loop。三连追问照 sense_loop S 格式：

① 现状中，你认为最不能忽视的事实是什么？为什么？
② 如果期望达成，你看到的世界与现在有什么不同？
③ 现状与期望之间，真正的阻碍是什么？（不是表面差距）
