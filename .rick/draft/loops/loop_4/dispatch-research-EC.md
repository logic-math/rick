# 派发：research subagent — EC 颠覆性假设调查（无限上下文+强模型→rick 是否还需要）

EC 阶段 human 自判提出最不安假设，并显式要求"详细调查 + 证实与反驳观点"。这是特殊情况（human 提调研问题），派 research 并行，不中断 EC 主流程。此调查结果可能颠覆前序判断（A7/A17/A18/核心假设），若颠覆则触发 EC 升维/降维→反向回流。

**先读**（如未在上下文）：
- `loop_4/prompts/skill_research.md`、`loop_4/prompts/research.md`
- `loop_4/briefs/批判门禁-E-r5.md`（E 收敛：核心假设 A7+A5+A15+A18 / rick 价值论 D3′ / 架构定位 D3）
- `loop_4/briefs/research-report-E-r2.md`（A7 有损压缩 CONFIRMED）
- `loop_4/briefs/research-report-SR.md`（逆转逻辑三层）
- `loop_2/briefs/research-4-N2-pi-compaction-内容保留策略.md`（compaction 已尽调，复用）

无 `.rick/config.json`，信源权重默认（代码0.4/运行时0.3/文档0.2/反事实0.1），高置信 ≥0.8。运行时适配同前：你即调研执行者，可直接用 Read/Grep/Bash/WebFetch/WebSearch（WebFetch/WebSearch 被拦截改用 curl+arxiv REST），保留尽调树/MECE/加权/R7/落盘/`git restore` 全部约束。

---

## human 调研请求（原话）

> 我认为最不安的假设是 如果模型的上下文足够长，模型智能足够强。是否就不需要管理上下文了？ AI 能够自主检索所有内容，获得最有效的上下文，然后完成任务。 甚至是自动让模型自己淘汰上下文，遗忘不重要的信息，例如 codex 的服务端压缩。 上下文方法压缩 某种意义上就是遗忘，如果存在一个无限度上下文，自我可以不断迭代直到任务完成。 那还需要 rick 吗？ 请你帮我详细调查，然后给出 证实与反驳的观点。

## 前序判断（被挑战的假设）

- A7（CONFIRMED）：LLM 参数权重=有损+非确定压缩（内禀，不可消除）。
- A17：rick 价值=弥补参数记忆有损+非确定；手段=应对上下文熵增。
- A18：单轮不足/多轮改善→需迭代→rick 存在。
- 核心假设：∃G 外 G′（未见过+无法一次性解决）→ 需迭代 → rick 存在。
- S-R 层 1：可控性转移（输出侧→输入侧，确定性提取+强制执行）。
- human 挑战：若上下文无限长+模型足够强→上下文管理/确定性提取/迭代是否失效→rick 是否还需要？

## 任务（尽调树，MECE 划分）

建议根节点下第一层（可自行调整）：

- **节点 A：无限上下文是否可行/可达？** context window 增长趋势（GPT/Claude/Gemini 百万级 token）、注意力复杂度 O(n²) 的理论/工程下限、长上下文衰减（"lost in the middle"）、无限上下文的物理/算力/成本墙。
- **节点 B：无限上下文+强模型能否消除"上下文管理"需求？** 模型自主检索（agentic retrieval/RAG 自动化）、自主淘汰上下文/遗忘（Codex 服务端 compaction、auto-compaction）、"获得最有效上下文"是否本身是判断（A15：编排不在 G，模型 zero-shot 不选）。
- **节点 C：上下文压缩=遗忘？无限上下文是否避免损失？** 关键辨析：A7 是**参数级**（权重）有损压缩；上下文压缩是**非参数级**（context window）。无限上下文是否绕过参数级损失？还是只是把"遗忘"从参数级移到检索级（检索非确定=另一种损失）？复用 E-r2 节点 C（确定性提取需求必然存在）。
- **节点 D：即使无限上下文+强模型，rick 的哪些价值仍不 collapse？** 候选（验证哪些成立）：(1) 确定性编排/强制执行（A15：模型 zero-shot 不选 rick 编排，无限上下文不改变"不选"）；(2) human 判断者不可替代（N2 提示：G′"最大化改进"由谁认定，LLM 无法自证）；(3) 失败模式管理（回退/震荡/局部最优在无限上下文下是否仍存在）；(4) 做事方法/编排不在 G（A15 confirmed，与上下文长度正交）。
- **节点 E：证实观点**（支持"无限上下文+强模型→rick 不需要"）——枚举最强论据。
- **节点 F：反驳观点**（支持"即使无限上下文+强模型→rick 仍需要"）——枚举最强论据，关联 A7/A15/A17/A18 + human 判断者。

## 信源建议

- 文档（0.2）：context window 趋势、long-context 衰减论文（"Lost in the Middle" Liu et al. 2023）、注意力复杂度、agentic RAG、Codex 服务端 compaction 文档、self-eviction/forgetting 文献
- 运行时（0.3）：若可行，跑 long-context demo 验证衰减/检索非确定（复用 E-r2/E-r4 runtime 经验，API/CLI 受限则记录降级）
- 代码原文（0.4）：rick compaction/ContextManager 实现（复用 loop_2 research-4）；codex 服务端压缩机制（若可访问文档）
- 反事实（0.1）：对比"无限上下文单轮" vs "rick 确定性提取+迭代"在 G′ 上的差异（文献推断）

## 安全约束

- 修改代码必须 `git restore` 还原
- 运行程序优先只读命令；复用已有 brief 作为信源（标注来源）

## 交付标准

按 research.md 主报告格式：信源配置 + 尽调树快照 + 节点详情（置信度+各信源 ✅/❌+证据+疑问点）+ R7 上报 + 整合摘要 + **证实与反驳观点对照**（节点 E vs F，逐论据，关联前序假设 A7/A15/A17/A18/核心假设）+ **对 human 请求的直接回答**：在无限上下文+强模型假设下，rick 是否还需要？哪些价值 collapse、哪些保留？

**关键**：此调查可能颠覆前序判断。若 research 发现 rick 核心价值在无限上下文下 collapse（证实强），需明确标注"建议触发 EC 升维/降维反向回流"；若核心价值保留（反驳强），标注"维持"。

## 产物写入

主报告：`loop_4/briefs/research-report-EC.md`；节点详情按需 `research-EC-{A..F}.md`。

## 返回

整合摘要 + 证实/反驳观点对照 + 对 human 请求的直接回答（rick 是否还需要）即为最终输出。
