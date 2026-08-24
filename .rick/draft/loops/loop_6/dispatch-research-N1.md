# 派发：research subagent — N1 矛盾生成（系统论描述符）

**先读**：
- `/workdir/sunquan20/AI_CODING/rick/.rick/draft/loops/loop_6/prompts/research.md` + `skill_research.md`
- `/workdir/sunquan20/AI_CODING/rick/.rick/draft/loops/loop_6/judgment.md`（S + E 收敛判断）
- `/workdir/sunquan20/AI_CODING/rick/.rick/draft/loops/loop_6/briefs/research-report-E.md`（视角候选 V1~V6 + 事实基础）
- `/workdir/sunquan20/AI_CODING/rick/.rick/draft/loops/loop_6/briefs/research-report-S-bestpractice.md` + `research-report-S-reasons-agent.md`（BP-1~BP-9 / B1~B4 / A/C 分支事实）

运行时适配同前：你即调研执行者，直接 Read/Grep/Bash，不再递归派发 subagent，保留 MECE/信源加权/R7/落盘约束。

---

## 五派发要素

**阶段**：N1 矛盾生成（基于已确认视角，用系统论描述符描述系统，分析稳态，列举矛盾状态）

**主题**：subagent 在 pi runtime 下触发概率低，优化提示词（使触发确定性提升到上限内最高）。

**草稿路径**：`.../draft` | rfc：`.../draft/rfc` | 会话：`.../draft/loops/loop_6`

**前序判断**（S + E 收敛，human 原话要点）：
- S：现状 = rick 各命令提示词明确要求用 subagent 但 pi runtime 下触发概率低；期望 = 触发确定性提升到上限内最高；差距 = pi 最佳实践 vs rick 现状；因果 = 提示词未对齐 pi 触发机制（主要原因，human 确认）。
- E 视角 = **协议对齐/兼容**（V5）；类比 = **两个各自自洽、合作时暴露问题的语言体系**；动作 = 在 pi 框架长期发展前提下，**高度定制化改造 rick 提示词**，基于 pi 的 subagent 最佳触发实战对齐。

**任务派发**：基于「协议对齐」视角，用**系统论描述符（5 要素）**描述「rick + pi + LLM(main agent) + human + 外部存储（提示词模板/agent 定义/配置）」这一系统，分析稳态（A→B 所需控制手段），列举多种相互矛盾的状态供 human 选择。

**结果核验**：系统论描述符列表 + ASCII 图 + 稳态分析 + 矛盾状态列表。

---

## 系统论描述符（5 要素）

| 要素 | 含义 | 需识别（协议对齐视角下） |
|---|---|---|
| node | 系统组件 | human / rick（命令+提示词+方法层·引导程序）/ main agent（pi 下 LLM 会话）/ pi-subagents 运行时（subagent 工具 + tool description + agent 注册表）/ 子 agent（think/research/exporter 或 pi 内置/自定义 agent）/ 外部存储（rick 模板、agent 定义 markdown、pi 配置） |
| input | 系统输入 | human 的任务需求/命令；pi 协议规范（tool description 定义的触发语法/agent 名） |
| output | 系统输出 | 子 agent 被触发并产出结果；rick 命令交付物 |
| inner | 系统内部协作 input/output | rick 提示词→main agent（指令注入）；main agent→subagent 工具（调用）；tool description→main agent（协议认知）；agent 定义→运行时（符号表）；子 agent→main agent（结果回流） |
| edge | node 间协作关系 | 承载 inner_input/inner_output 的边（如 rick↔main agent 的「提示词协议」边、main agent↔pi-subagents 的「工具调用协议」边） |

## 具体调研

1. **node/input/output/inner/edge 列表 + ASCII 图**：基于已证实事实（N3.1/N3.2 零触发语法/零内置名、D1~D7 七项差距、BP-1~BP-9、B1~B4）识别 5 要素，画系统图，突出「协议不对齐」在哪些 edge 上发生。
2. **稳态分析**：当前稳态 A（rick 提示词自然语言软触发、零 pi 触发语法/agent 名、main agent 触发概率低）→ 目标稳态 B（rick 提示词按 pi 协议对齐：显式触发语法 + 真实 agent 名 + 系统级 agent 注册，触发确定性达上限内最高）所需控制手段。
3. **多种相互矛盾的状态**（供 human 在 N2 选择主要矛盾，自行补充）：
   - rick 自然语言软触发（自洽但不确定）vs pi 显式强制触发（确定但需改造）
   - rick 自造角色名（表达 rick 方法论）vs pi 真实 agent 名（协议约束）
   - rick 方法层独立性（引导程序/sense 方法论）vs pi 协议要求（按 pi 触发语法改造）
   - 高度定制化改造（长期跟随 pi）vs rick 框架稳定性/独立性
   - 触发确定性（协议对齐可解决）vs 模型能力上限（不可控残余，R7）
   - 统一改造覆盖各命令 vs 各命令特殊场景（ctrl 无 subagent 要求、dream 有 4 子 agent 轮询等）
4. **human 启发性追问**（简报末尾，照 sense_loop N1 格式）：
   - 在这个系统中，你看到哪两股力量在拉扯？
   - 如果系统继续按现状运行，3 年后会发生什么？
   - 系统的哪个节点，如果消失，整个系统会重组？

## 产出与落盘

主报告：`/workdir/sunquan20/AI_CODING/rick/.rick/draft/loops/loop_6/briefs/research-report-N1.md`

**禁止**：简报含倾向性（不推荐某矛盾为主要）、替 human 选择矛盾、无事实支撑构建选项、跳过 MECE。

## 返回

N1 简报全文（系统论描述符列表 + ASCII 图 + 稳态分析 + 矛盾状态列表 + human 启发性追问）作为最终输出返回 sense_loop。
