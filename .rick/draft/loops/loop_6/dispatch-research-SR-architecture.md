# 派发：research subagent — S-R 架构可行性调研（skill+loop 抽象 + runtime 转义层）

**先读**：
- `/workdir/sunquan20/AI_CODING/rick/.rick/draft/loops/loop_6/prompts/research.md` + `skill_research.md`
- `/workdir/sunquan20/AI_CODING/rick/.rick/draft/loops/loop_6/judgment.md`（S/E/N/S-R 收敛 + 本次架构方案）
- `/workdir/sunquan20/AI_CODING/rick/.rick/draft/loops/loop_6/briefs/research-report-S-bestpractice.md`（BP-1~BP-9）
- `/workdir/sunquan20/AI_CODING/rick/.rick/draft/loops/loop_6/briefs/research-report-S-reasons-agent.md`（B1~B4）
- `/workdir/sunquan20/AI_CODING/rick/.rick/draft/loops/loop_6/briefs/research-report-debate-dsh-vs-pi.md`（pi vs dsh 辩论结论）

运行时适配同前：你即调研执行者，直接 Read/Grep/Bash，不再递归派发 subagent，保留 MECE/信源加权/R7/落盘约束。

---

## 五派发要素

**阶段**：S-R 架构可行性调研（human 提问）

**主题**：rick 方法层抽象为「skill + loop」、在 runtime 层加「方法→runtime 转义层」的架构是否可行。

**草稿路径**：`.../draft` | rfc：`.../draft/rfc` | 会话：`.../draft/loops/loop_6`

**前序判断（human 原话要点）**：
- rick 方法层抽象 = **skill + loop**；**skill = wiki + tools**；**loop = workflow + skill**。
- 实现层兼容 pi 语言，但保留 rick 核心独立（便于替换底层 runtime）。
- 在 runtime 层增加「方法→runtime 转义层」；适配其他 runtime = 重定义转义层。

**任务派发**：调研该架构的可行性，分三个分支：

### 分支 A：rick 现有资产与「skill + loop」抽象的映射
1. 核实 rick 方法层现有资产：
   - wiki 部分：`internal/prompt/templates/skills/sense.md / think.md / research.md / exporter.md`（SENSE 方法论 = wiki？）
   - loop/workflow 部分：`templates/sense_loop.md / think.md / research.md / exporter.md`（五阶段流程 = loop？）
   - tools 部分：rick 的 think/research/exporter 各含什么"工具能力"（读/写/调研/打分），skill = wiki + tools 中的 "tools" 对应什么？
2. 判断：「skill = wiki + tools」「loop = workflow + skill」能否覆盖 rick 方法层的全部语义（SENSE 五阶段 + 三角色 + 门禁 + 反向回流 + 判断记录）。

### 分支 B：pi 侧「转义层」的可行性
1. pi 现有承载机制：自定义 agent（frontmatter markdown）、skills（SKILL.md）、workflowScript（编排）。核实这三者能否分别承载 rick 的 skill（wiki+tools）、loop（workflow+skill）、转义层（方法→pi 翻译）。
2. 转义层在 pi 上具体落成什么（例如：rick 的 skill+loop 抽象 → 转义层生成 pi 的 agent frontmatter + SKILL.md + workflowScript 模板）？机制是否够用、有无缺口。
3. 关键约束核实：pi 自定义 agent 空系统提示词起步（需显式注入上下文）、skills 目录约定、workflowScript 无静态校验——这些对转义层的设计有何影响。

### 分支 C：dsh 侧「转义层」的可行性（验证"重定义转义层即可适配其他 runtime"）
1. dsh 的承载机制：plugin（Cordis）、subagent provider、preset/persona/toolFilter。核实能否承载同一套 rick skill+loop 抽象。
2. 判断：「同一 rick 方法层抽象 + 每 runtime 一个转义层」的分层是否成立（即方法层抽象是否足够 runtime 无关，转义层是否只需重定义而不改方法层）。

## 结果核验

- 三分支事实清单（每条含信源+置信度）
- 可行性结论（✅ 可行 / ⚠️ 有条件可行 / ❌ 不可行，附理由与缺口清单）
- R7 上报项（无法达高置信的叶节点）

## 产出与落盘

主报告：`/workdir/sunquan20/AI_CODING/rick/.rick/draft/loops/loop_6/briefs/research-report-SR-architecture.md`

**禁止**：简报含倾向性（只陈述可行性事实，不替 human 决定是否采纳）、替 human 判断、无事实支撑构建选项、跳过 MECE。

## 返回

S-R 架构可行性简报全文（三分支 + 可行性结论 + 缺口清单 + R7）作为最终输出返回 sense_loop。
