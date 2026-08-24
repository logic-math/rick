# 派发：research subagent — E 视角生成（跨领域调研 → 多视角候选）

**先读**：
- `/workdir/sunquan20/AI_CODING/rick/.rick/draft/loops/loop_6/prompts/research.md` + `skill_research.md`
- `/workdir/sunquan20/AI_CODING/rick/.rick/draft/loops/loop_6/judgment.md`（S 阶段 human 判断原话）
- `/workdir/sunquan20/AI_CODING/rick/.rick/draft/loops/loop_6/briefs/research-report-S.md`、`research-report-S-bestpractice.md`、`research-report-S-reasons-agent.md`（S 已证实事实）

运行时适配同前：你即调研执行者，直接 Read/Grep/Bash/WebSearch/WebFetch，不再递归派发 subagent，保留 MECE/信源加权/R7/落盘约束。

---

## 五派发要素

**阶段**：E 视角生成（pErspective）

**主题**：subagent 在 pi runtime 下触发概率低，优化提示词（使触发确定性提升到上限内最高）。

**草稿路径**：`.../draft` | rfc：`.../draft/rfc` | 会话：`.../draft/loops/loop_6`

**前序判断**（S 收敛，human 原话要点）：
- 现状：rick 各命令（human-loop/plan/doing/easy/learning/dream/ctrl）提示词明确要求用 subagent，但 pi runtime 下触发概率低。
- 期望：改动 rick 框架（提示词+配置），使 subagent 触发确定性**提升到上限内最高**，让模型更遵守提示词约束。
- 差距：pi 触发 subagent 的最佳实践 与 rick 现状 之间（rick 提示词零触发语法、零内置 agent 名，150 处自然语言 subagent 术语）。
- 因果：提示词未对齐 pi 触发机制是主要原因（human 确认）。
- 方向：按 pi 方式把几个 subagent 内置为系统级 agent，定义好 agent，用明确提示词触发。

**任务派发**：**跨领域调研**，引用不同领域的理论，产出**多视角候选**（每个视角含：来源理论 + 事实支撑 + 融贯性[自洽/他洽/续洽]），供 human 综合出原创视角。不替 human 选择视角。

**结果核验**：多视角候选列表，每个候选含来源理论/事实支撑/融贯性三要素。

---

## 视角候选方向（供参考，可自行补充/替换，需跨领域、有理论来源）

核心问题可被多个领域的理论重新表述，例如（非穷举，research 自行取舍）：

1. **可靠性/控制论视角**：触发 = 控制系统的确定性响应；如何让"要求→触发"的映射从概率性变为确定性（反馈/前馈/强制约束）。
2. **人机交互/提示工程视角**：指令遵循（instruction following）——模型遵守提示词约束的条件是什么；软自然语言 vs 显式规范（schema/语法）的遵循度差异。
3. **组织管理/委托代理（principal-agent）视角**：父 agent 委托子 agent 时，如何通过契约（contract）+ 边界（authority boundary）保证代理执行；激励/约束相容。
4. **编程语言/编译器视角**：确定性派发（dispatch）——编译器如何把符号引用解析到确定实现（符号表、显式绑定、名字解析），对比自然语言"角色名"的模糊引用。
5. **协议设计视角**：双方（模型↔工具）通信需要协议对齐（协议即工具描述 + 触发语法）；未对齐协议 = 握手失败。
6. **认知科学视角**：人类/LLM 遵循指令的认知机制；显式外部化规则 vs 内隐语境。

每个候选视角需给出：**来源理论**（理论名+领域+可引用出处，可 WebSearch）、**事实支撑**（rick/pi 的已证实事实如何支撑该视角）、**融贯性**（自洽：视角内部一致 / 他洽：与已证实事实一致 / 续洽：能预测后续改进效果）。

## 产出与落盘

主报告：`/workdir/sunquan20/AI_CODING/rick/.rick/draft/loops/loop_6/briefs/research-report-E.md`

**禁止**：简报含倾向性（不推荐某视角）、替 human 选择视角、无事实支撑构建选项、跳过融贯性三要素。

## 返回

多视角候选列表（每候选含来源理论/事实支撑/融贯性）作为最终输出返回 sense_loop。
