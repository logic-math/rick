# 派发：research subagent — E-r2 并行调研（human 调研请求）

你是 **research subagent**。这是 sense_loop E 阶段批判门禁 r2 期间的并行调研（human 显式提出的调研请求，不中断主流程）。

**第一步：必须先读以下文件了解你的角色与工作流**：
- `/workdir/sunquan20/AICODING/rick/.rick/draft/loops/loop_4/prompts/skill_research.md`
- `/workdir/sunquan20/AICODING/rick/.rick/draft/loops/loop_4/prompts/research.md`
- `/workdir/sunquan20/AICODING/rick/.rick/draft/loops/loop_2/judgment.md`（前序判断，了解 rick/pi 迁移背景）

无 `.rick/config.json`，信源权重取默认：代码原文 0.4 / 运行时 0.3 / 文档 0.2 / 反事实 0.1；高置信度阈值 ≥0.8；树规模 深度≤5 / 每层≤7 / 总≤30。

> 运行时适配：protocol 的 research 设计为"派发 subagent 实现上下文隔离"。在本运行时中，你即是调研执行者——可直接用 Read/Grep/Bash/WebFetch/WebSearch 执行各信源验证，无需再派子层 agent（节点级上下文隔离已由本派发的作用域保证）。保留尽调树/MECE/加权/R7/落盘/`git restore` 还原等全部方法论约束。

---

## 五派发要素

**阶段**：E 批判门禁 r2 并行调研（human 提的调研问题，不中断主流程）
**主题**：继续 loop_2 未思考完的问题，推进 sense 思考过程
**草稿**：`.../draft/loops/loop_4`
**前序判断**（loop_2 S+E 已确认，详见 `loop_2/judgment.md`）：S 已收敛（rick→pi 迁移全部事实已尽调，8 轮 research）；E human 原创视角"盒子里的 LLM"，核心论断之一为"解决 G' 的方法无法被 LLM 训练覆盖"。

## human 调研请求（原话）

> 这里的假设在于，LLM 的参数权重本质上是一种非确定性的信息压缩，提取时也会具有一定的损失。请你帮我验证这一点，LLM 存储的知识是否是一种有损信息压缩。如果是的话，那么对于确定性的信息提取的需求就会一定存在，LLM 就必然依赖外部信息进行思考。

## 任务

构建尽调树（MECE 划分），验证以下事实陈述。建议根节点下第一层划分（你可自行按 MECE 调整）：

- **节点 A**：LLM 参数权重是否"有损"信息压缩（vs 无损）？知识存储是否存在提取损失？
  - 信源：文档（信息论率失真；LLM-as-compressor 文献如 Delétang et al. 2023 "Language Modeling Is Compression"；知识冲突/幻觉文献如 "Survey of Hallucination"）；运行时（同 prompt 多次采样看输出方差）
- **节点 B**：LLM 知识提取是否"非确定性"（同输入不同输出）？损失是否随提取方式（prompt/温度/top-k）变化？
  - 信源：运行时（温度采样 demo）；文档（采样理论、校准文献）
- **节点 C**：若有损+非确定，"确定性的信息提取需求"是否必然存在 → LLM 是否必然依赖外部信息（扩展心智/RAG/上下文工程）思考？
  - 信源：文档（扩展心智理论 Clark & Chalmers 1998 "The Extended Mind"；RAG 动机文献 Lewis et al. 2020）；代码原文（rick 仓库内 compaction/上下文工程实现，grep "compact" "context" "system prompt"，验证 rick 现有"确定性信息提取"机制是否存在）
- **节点 D**（关联 Y-E3）：模型持续学习/对齐训练/后训练的成本量级，是否可能降至"实时"？（决定 rick 价值前提"训练成本高→rick 必需"的刚性）
  - 信源：文档（continual learning / online RLHF 成本文献；pi 是否支持动态知识注入——查 loop_2 research-4-N3-pi-compaction-自定义扩展点.md）

## 信源建议

- 文档（0.2）：WebFetch/WebSearch 上述论文与理论；pi 文档（持续学习/动态注入能力，可复用 loop_2 已有 research brief）
- 运行时（0.3）：Bash 跑简单 demo——同一 prompt 多次采样验证提取不确定性（若有可用 LLM CLI/API，如 rick 自身或本地小模型）；若无可跳过此项并在报告中说明
- 代码原文（0.4）：rick 仓库 grep "compact" "context" "systemPrompt" 等，验证 rick 现有"确定性信息提取/上下文工程"机制；参考 loop_2 research-4-N2-pi-compaction-内容保留策略.md
- 反事实（0.1）：如有条件，对比有/无外部上下文的输出差异；否则跳过

## 安全约束

- 修改代码必须 `git restore` 还原，整合前检查"还原确认"段
- 运行程序优先只读命令，写命令需先备份
- 可复用 loop_2 已有 research brief 作为信源（标注来源），避免重复调研

## 交付标准

按 research.md"主报告格式"产出，必须包含：
- 信源配置（权重表+加权公式+阈值）
- 尽调树快照（每叶节点置信度）
- 节点详情（置信度+各信源 ✅/❌+证据+疑问点）
- R7 上报项（无法达高置信度的叶节点+理由）
- 整合摘要（总节点/高置信/R7 数）
- **对 human 请求的直接回答**：有损压缩是否成立？下游推论（确定性提取需求必然存在→LLM 必然依赖外部信息）是否成立？逐节点置信度。

## 产物写入

- 主报告：`/workdir/sunquan20/AICODING/rick/.rick/draft/loops/loop_4/briefs/research-report-E-r2.md`
- 各节点详细报告：`.../briefs/research-E-r2-{节点A/B/C/D}.md`

## 返回

整合摘要即为你的最终输出（尽调树快照+R7 清单+对 human 请求的直接回答）。
