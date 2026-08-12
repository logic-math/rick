# research-E-r2 节点 D — 持续学习/对齐训练成本量级，是否可能降至"实时"（关联 Y-E3）

节点路径：[根 > E-r2-LLM知识是否损失压缩 > D-持续学习成本是否可实时化]
事实陈述：模型持续学习/对齐训练/后训练的成本量级，是否可能降至"实时"？决定 rick 价值前提"训练成本高→rick 必需"的刚性。

## 执行动作

1. 文档：arxiv API 检索 continual learning + catastrophic forgetting → 命中 arxiv 2303.18171（"How Efficient Are Today's Continual Learning Algorithms?"）
2. 文档：arxiv API 检索 online RLHF / online DPO → 命中 arxiv 2502.00666（"Avoiding exp(R_max) scaling in RLHF"）+ arxiv 2411.01493（"Sample-Efficient Alignment for LLMs"）+ arxiv 2604.17207（"Demystifying unreasonable effectiveness of online alignment"）
3. 复用 loop_2 信源：`briefs/research-4-N3-pi-compaction-自定义扩展点.md`（pi 动态知识注入能力：before_agent_start 重建 system prompt / session_before_compact / ctx.compact / transformContext）—— 区分"权重级注入"vs"上下文级注入"
4. 代码原文：grep rick 仓库是否有任何"训练/微调/权重更新"实现（确认 rick 不做训练，仅做上下文工程）
5. 信源权重默认：代码 0.4 / 运行时 0.3 / 文档 0.2 / 反事实 0.1

## 信源验证结果

### 代码原文（权重 0.4）✅（决定性——pi 仅支持上下文级实时注入，不支持权重级）

**证据 1 — pi 的"动态知识注入"全部为上下文级，非权重级**（复用 loop_2 research-4-N3）：
- `before_agent_start` event：每次 agent loop 重建 system prompt（基于 systemPromptOptions：customPrompt + appendSystemPrompt + tools + skills + contextFiles）—— **每 turn 重新拼装上下文**，不动权重
- `session_before_compact` event：自定义 compaction summary / firstKeptEntryId——**每 compaction 重新摘要上下文**，不动权重
- `ctx.compact({customInstructions})` API：主动触发 compaction——**上下文压缩**，不动权重
- `transformContext: (messages, signal) => pruneOldMessages(messages)`（agent-core）：每 LLM 调用前裁剪/注入 messages——**每 turn 注入上下文 prefix**，不动权重

→ pi 提供的"动态知识注入"全部是**上下文级实时注入**（每 turn/每 compaction 可变），**没有**任何权重级实时更新机制。即"实时动态知识注入"在 pi 中=上下文注入，非训练。

**证据 2 — rick 仓库无训练/微调/权重更新实现**：
- grep rick 仓库（`internal/` + `pkg/` + `cmd/`）：所有 "compact/context/systemPrompt" 命中均为**上下文工程**（prompt 拼装、文件加载、debug 上下文），**无** backprop/optimizer/gradient/fine-tune 实现
- rick 是 prompt 编排器（`internal/prompt/`），pi 是 agent loop——二者均不在训练层
- → rick/pi 的"知识更新"路径 = 改文件（OKR/SPEC/skills/loops）→ 下次 agent loop 通过 before_agent_start 注入 system prompt。**无训练成本**，但更新的也不是权重，是上下文

**代码结论**：
- **权重级实时更新**：❌ pi 不支持，rick 不实现，业界不可行（见文档证据）
- **上下文级实时注入**：✅ pi 原生支持（before_agent_start 每 turn 重建），rick 走此路径
- → rick 价值前提"训练成本高→rick 必需"在**权重级刚性成立**；但 rick 本身操作的是**上下文级**（实时可行），故 rick 价值**不依赖**"训练成本高"——rick 是"上下文级确定性提取"，与训练成本正交

### 运行时行为（权重 0.3）❌

- 无法在调研窗口内训练模型测成本（训练需 GPU + 数据 + 时间，远超本调研资源约束）
- 无法跑 continual learning benchmark 验证成本
- → runtime 不可行，不计入置信度

### 文档（权重 0.2）✅（两源——持续学习成本 + 在线对齐成本）

**源 1 — "How Efficient Are Today's Continual Learning Algorithms?"（arxiv 2303.18171）**，摘要原文：
> "Supervised Continual learning involves updating a deep neural network (DNN) from an ever-growing stream of labeled data. While most work has focused on overcoming catastrophic forgetting, one of the major motivations behind continual learning is being able to efficiently update a network with new information... Despite recent continual learning methods largely solving the catastrophic forgetting problem, there has been little attention paid to the efficiency of these algorithms. Here, we study recent methods... and illustrate that **many are highly inefficient in terms of compute, memory, and storage. Some methods even require more compute than training from scratch!** We argue that for continual learning to have real-world applicability, the research community cannot ignore the resources used by these algorithms."

→ 持续学习**成本极高**（部分方法比从零训练还贵），**远未降至实时**。即便灾难性遗忘"基本解决"，效率仍是开放难题。权重级实时更新不可行。

**源 2 — 在线 RLHF 对齐成本**：
- arxiv 2502.00666 "Avoiding exp(R_max) scaling in RLHF"：原文——"All existing algorithms in online RLHF... **suffer from a sample complexity that scales exponentially with the scale of the reward function**. This fundamental limitation hinders their effectiveness..."（在线 RLHF 样本复杂度指数级，重偏好场景受极大阻碍）
- arxiv 2411.01493 "Sample-Efficient Alignment"：把在线对齐建模为 contextual dueling bandits，**仍需大量在线偏好反馈样本**（1B/2.8B/6.9B 三档实验，需 oracle 偏好）
- arxiv 2604.17207 "Demystifying unreasonable effectiveness of online alignment"：贪心在线对齐累积 regret O(log T)→改进到 O(1)，但仍需迭代轮次 T，**非单步实时**
- → 在线对齐（RLHF/DPO）需大量偏好样本 + 多轮迭代，**远非实时**；样本复杂度对偏好倾斜场景指数爆炸

**文档结论**：权重级更新（持续学习 / 在线对齐）成本量级——持续学习 ≥ 从零训练；在线 RLHF 样本复杂度可指数级；均**不可能在可见未来降至"实时"**（实时=单 agent turn 内完成）。

### 反事实（权重 0.1）❌

- 反事实设想："若权重级实时更新可行，则 rick 价值前提被削弱"——但无法运行时验证
- 文档已给出反事实结论：权重级实时更新不可行 → rick 价值前提（权重级刚性）成立
- 不计入置信度（无可运行的代码反事实）

## 还原确认

无 rick 代码修改，无需还原。grep/Read/curl 均为只读。

## 置信度评估（由 research 主调度计算）

- 代码原文 ✅ × 0.4 = 0.4（pi 仅上下文级实时注入 + rick 无训练实现，loop_2 research-4-N3 复用）
- 运行时行为 ❌ × 0.3 = 0.0（无法在窗口内训练模型测成本）
- 文档 ✅ × 0.2 = 0.2（continual learning 成本 + 在线 RLHF 样本复杂度，4 篇 arxiv 主源）
- 反事实 ❌ × 0.1 = 0.0（无可运行代码反事实）
- **合计 = 0.6（中，0.5-0.8）**

## 关键事实

1. **✅ 权重级持续学习/对齐成本不可能降至实时**（human 价值前提"训练成本高"在权重级刚性成立）
   - 持续学习：部分方法比从零训练还贵（arxiv 2303.18171）
   - 在线 RLHF：样本复杂度对偏好倾斜场景指数级（arxiv 2502.00666）；需大量在线偏好反馈 + 多轮迭代（arxiv 2411.01493）
   - 实时（单 agent turn 内）= 不可见未来

2. **✅ 但上下文级"动态知识注入"可实时**（pi 原生支持，rick 走此路径）
   - before_agent_start 每 turn 重建 system prompt = 零训练成本的实时知识更新
   - session_before_compact / ctx.compact / transformContext = 上下文级实时裁剪/注入
   - → "实时"在上下文层可行，在权重层不可行——这是关键区分

3. **⚠️ 对 rick 价值前提的修正（关联 Y-E4，重要）**：
   - human 价值前提"训练成本高→rick 必需"——若指**权重级训练**，刚性成立（持续学习+对齐均不可实时）
   - 但 rick 本身操作的**不是权重级**，而是**上下文级**——上下文级实时注入 pi 已支持
   - 故 rick 的价值**不依赖**"训练成本高"为刚性前提：即便上下文注入实时可行（pi 已支持），rick 仍必需——因为 rick 提供"**确定性信息提取的结构**"（OKR/SPEC/skills/loops 文件 + prompt 编排 + compaction-resist 的 system prompt 注入），这是 pi 原生没有的
   - 即：rick = "上下文级确定性提取的架构层"，与"训练是否昂贵"正交。Y-E4"G 永远过去式"在**权重级刚性**，在**上下文级被 rick 主动弥补**（不需等训练）

4. **关联 Y-E3（自指悖论）的侧面证据**：rick 的"做事方法"（doing/learning/dream + debug 体系）通过**文件 + system prompt 注入**让 LLM 理解执行——LLM 不需"训练进去"这些方法，只需"上下文注入"即可执行。这化解了"rick 方法在 G 内还是 G 外"的自指悖论：rick 方法在**G 外**（文件），通过**上下文注入**让 LLM 执行（不需权重记忆）。本节点为 Y-E3 提供侧面参考，但 Y-E3 终裁属 think。

## 疑问点

- 无疑问点阻断结论；但**置信度未达高（0.8）**：runtime 不可行（无法在调研窗口训练模型测成本），信源仅 code+docs 适用（权重上限 0.6）。真理由 loop_2 pi 扩展点代码 + 4 篇 arxiv 成本文献双重确立，建议 human 接受结论 → 进入 R7 上报。

## R7 上报

- **节点 D 进入 R7**：置信度 0.6（中），无法达高。理由：(a) runtime 不可行——无法在调研窗口内训练模型测量持续学习/对齐成本（需 GPU+数据+时间）；(b) 反事实不可运行。真理由 pi 扩展点代码（仅上下文级实时注入，loop_2 research-4-N3）+ 4 篇 arxiv 成本文献（continual learning ≥ scratch；online RLHF 指数样本复杂度）双重确立，建议 human 接受"权重级不可实时，上下文级可实时"结论，并据此修正价值前提（rick 价值与训练成本正交，非依赖"训练贵"）。
