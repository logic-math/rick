# 派发：research subagent — S 追加调研（pi 触发 subagent 搭建工作流的最佳实践）

**先读**：
- `/workdir/sunquan20/AI_CODING/rick/.rick/draft/loops/loop_6/prompts/research.md` + `skill_research.md`
- 前序调研结果：`/workdir/sunquan20/AI_CODING/rick/.rick/draft/loops/loop_6/briefs/research-report-S.md`（已证实的缺口 N3.1/N3.2）

运行时适配同前：你即调研执行者，直接 Read/Grep/Bash，不再递归派发 subagent，保留 MECE/信源加权/R7/落盘约束。

---

## 五派发要素

**阶段**：S 问题确认 · 追加调研（human 提问）

**主题**：pi 下触发 subagent、搭建多 agent 工作流的**最佳实践**是什么；与 rick 现状的差距是什么。

**草稿路径**：`.../draft` | rfc：`.../draft/rfc` | 会话：`.../draft/loops/loop_6`

**前序判断**（human 原话）：
- 现状：应调研 pi 触发 subagent 搭建工作流的最佳实践是什么。
- 期望：通过改动 rick 现有框架，优化提示词和某些配置，使 subagent 触发确定性最大化，让模型更能遵守提示词约束。
- 差距：最佳实践 与 rick 使用现状 之间的调研。

**任务派发**：调研 pi runtime 官方/推荐的最佳实践，产出「最佳实践清单」，并与 rick 现状（research-report-S.md 的 G2/G3 层）逐项对照，标出差距。

**结果核验**：最佳实践清单（每条含来源信源 + 置信度）+ 与 rick 现状的差距对照表 + R7 上报项。

---

## 调研要点

1. **pi-subagents 扩展的官方推荐写法**（代码原文 + 文档信源）：
   - `/home/hadoop-recsys/.rick/pi/agent/npm/node_modules/pi-subagents/skills/pi-subagents/SKILL.md`（完整读）
   - 同目录 `references/` 下所有文件（如 `execution-controls.md`、`prompting-and-roles.md`、工作流编排等），提取：如何让模型确定地触发 subagent、任务文本怎么写、agent 怎么选、gate/acceptance/async/worktree 等参数的最佳用法。
   - `pi-subagents/src/extension/tool-description.ts`、`src/agents/*.ts`（触发语法与内置 agent 定义，确认是否已有最佳实践说明）。
2. **pi 官方文档**（文档信源）：
   - `/home/hadoop-recsys/.rick/pi/agent/runtime/node_modules/@earendil-works/pi-coding-agent/docs/subagents.md`（若存在）、`docs/packages.md`、`docs/prompt-templates.md`、`docs/skills.md` 中与 subagent 触发/编排相关的部分。
3. **触发确定性手段**（归纳各信源）：官方如何确保「提示词要求用 subagent ⇒ 模型真的调用 subagent 工具」——例如：显式写 `subagent({workflowScript:...})` 语法、显式列 agent 名、给出触发条件/决策树、还是存在更软的约定。
4. **与 rick 现状对照**：rick 模板（`internal/prompt/templates/`）用自然语言「派发 subagent / 子 Agent / SPAWN Sub Agent」且零引用 pi 触发语法与内置 agent 名（N3.1/N3.2）——逐项指出「最佳实践要求 X，rick 现状是 Y，差距 = Z」。

## 产出与落盘

主报告：`/workdir/sunquan20/AI_CODING/rick/.rick/draft/loops/loop_6/briefs/research-report-S-bestpractice.md`

**禁止**：简报含倾向性（只陈述最佳实践与差距，不推荐"怎么改"）、替 human 判断、无事实支撑构建选项、跳过 MECE。

## 返回

最佳实践清单 + 差距对照表 + R7 上报项，作为最终输出返回 sense_loop。
