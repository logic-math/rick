# 调研报告 — S-R 辩证逆转（逆转逻辑尽调 + 可选项） — 2026-08-09

> 派发：`loops/loop_4/dispatch-research-SR.md`（N2 ✅ 通过，human 选定主要矛盾=M3-ext"对模型输入的可控性"，进入 S-R 辩证逆转）
> 工作流：`loops/loop_4/prompts/research.md`（S-R 简报格式：阻碍+逆转逻辑+替代路径可选项+human 追问）
> 前序：N1 系统描述符 + N2（M3-ext 选定，与 M3/M2 并列 3.0）+ E 收敛（A7/A15/A18 CONFIRMED）

## 前序判断（S-R 的基础）

- **主要矛盾**（human 选定 M3-ext）：输入可控+失败模式管理（rick 确定性提取/强制执行）vs 输出非确定+回退/震荡/局部最优（LLM 参数记忆有损+非确定，A7 CONFIRMED 内禀不可消除）。
- **核心价值**：有限迭代最大化改进（非单调，含回退/震荡/局部最优，需失败模式管理）。
- **系统描述符**：node=human/rick/pi/LLM/外部存储；edge=human↔rick、rick↔pi（系统提示词注入）、rick↔外部存储（确定性提取）、pi↔LLM（compaction）、LLM↔外部存储（skill 注册）、pi↔human（简报）。
- **稳态 A**（rick+ai_cli+claude code）→ **B**（rick+pi+深度定制：二进制/skill 系统级/自定义 compaction/subagent 递归）。

## 信源配置

无 `.rick/config.json`，取默认权重。**S-R 是基于已确认事实的逆转逻辑尽调+可选项枚举**（非新事实验证），信源复用：

| 信源 | 权重 | 本轮来源 |
|---|---|---|
| 代码原文 | 0.4 | 复用 E-r2/E-r4（context.go/doing_prompt.go/doing.md/doing_loop.md/think.md/RFC-001）+ loop_2 research-7（13 调用点+check 门禁 runAutoFix）+ research-4-N2/N3（pi compaction/扩展点） |
| 运行时行为 | 0.3 | 复用 E-r2/E-r4 runtime（LLM 非确定采样 5 次 {73,42,73,42,42}/G' 拒答/zero-shot 线性单遍） |
| 文档 | 0.2 | 复用 E-r2/E-r4（Self-Refine 2303.17651/Reflexion 2303.11366/Sample More 2607.28576/Plan-and-Solve 2305.04091/RAG Lewis 2005.11401/Extended Mind/Delétang/Tishby） |
| 反事实 | 0.1 | 复用 E-r2 节点 C de facto A/B + E-r4 节点 C zero-shot vs rick 强制对照 |

**加权公式**：置信度 = Σ(信源验证结果 × 信源权重)。逆转逻辑的事实基础继承前序 CONFIRMED（0.85-1.0）；替代路径是**供 human 选的可选项**（含利弊，不验证最优性）。

---

## 1. 阻碍识别（基于系统描述符 node/edge）

### X（必然前提/阻碍）= LLM 输出有损+非确定（A7 CONFIRMED，LLM node 内禀，不可消除）

- **node 映射**：LLM node —— 参数权重是有损+非确定压缩（A7，E-r2 节点 A 0.55+节点 B 0.50 互证；Tishby 信息瓶颈"学习=有损压缩"+Delétang"LLM=compressor"+幻觉调研）
- **edge 映射**：pi→LLM→output edge —— LLM 经 softmax+temperature 采样产出非确定输出（E-r2 runtime 5 次 {73,42,73,42,42}）；输出必有损/方差（E-r2 节点 A：对 G' 事实提取 100% 损失）
- **不可消除性**：A7 CONFIRMED 为 LLM 内禀——权重级有损（Tishby 瓶颈，泛化=遗忘），采样级非确定（temperature>0，Wikipedia softmax "more random"），均不可在 rick/pi 层消除（只能弥补）
- **扩展（M3-ext 右极）**：输出非确定 → 迭代中出现回退/震荡/局部最优（doing_loop Step 4"存在失败→返回 Step 3 下一轮"+Step 5"连续 2 轮产出相同错误=无法自动收敛"——回退/震荡/局部最优的代码承认）

### Y（期望）= 有限迭代最大化改进/可靠解决 G′

- **非单调**：doing_loop Step 3-5 的 3 轮上限 + DEBUG Phase 4 上限 3 次 + 停止标准（成功/优雅退出）——承认改进非单调，需失败模式管理
- **可靠**：check 门禁（runAutoFix 循环直到 pass）+ sense 批判门禁（假设打分+top-N+Y 澄清）——可靠性由外部门禁保证，非 LLM 输出本身
- **解决 G′**：A18 CONFIRMED（单轮对 G' 不足，多轮改善）+ A15（zero-shot 不选 rick 编排，需确定性注入）

### 阻碍的本质

X（输出非确定）不可消除，但 Y（可靠解决）要求确定性。**直觉路径**：消除 X → 可靠 Y。**但 X 不可消除（A7 内禀）**→ 直觉路径阻塞。此即 S-R 逆转的起点：X 是 Y 的必然前提（不可绕过），必须逆转逻辑。

---

## 2. 逆转逻辑

### 逆转逻辑形式化

> **若 [LLM 输出有损+非确定（A7 内禀不可消除）] 是 [有限迭代最大化改进解决 G′] 的必然前提，则 [可靠解决 G′] 应当 [放弃输出确定性的追求，把可控性从输出侧（不可控）转移到输入侧（可控），并用有限迭代+失败模式管理吸收/利用输出的非确定——把"非确定"从"阻碍"逆转为"改进的变异源"+"收敛的对象"]。**

### 逆转的三层结构

**层 1 — 可控性转移（输出侧→输入侧）**：
- 输出侧（LLM→output）：A7 内禀不可控 → 放弃追求输出确定性
- 输入侧（rick→LLM，经 pi→LLM edge）：可控 → rick 确定性提取（ContextManager 从文件加载）+ 确定性拼装（GenerateDoingPromptFile + SaveToFile 落盘）+ 强制执行（doing.md"不可跳过任何步骤"）→ 把确定性建在输入端
- **逆转**：既然输出不可控，就把"可控"全部押在输入；输入确定+输出非确定 → 输出方差被确定的输入"锚定"，不致漂移

**层 2 — 非确定吸收（迭代+失败模式管理）**：
- 输出非确定 → 单次输出不可靠 → 用迭代吸收方差：doing_loop Step 3 Main→Sub Agent per-iteration（每轮独立 Sub Agent 携带压缩上下文，执行完整工作流返回摘要）+ Step 4 失败→返回 Step 3（迭代重试）+ 3 轮上限（防止无限震荡）
- 回退/震荡/局部最优管理：DEBUG Phase 1-6（遇红强制触发，Phase 4 上限 3 次后升级人工）+ check 门禁（runAutoFix 循环直到 pass）+ sense 批判门禁（假设打分+top-N 阈值<0.3 不入选，防止低质假设通过）+ human 判断反馈（judgment.md，认定"最大化改进"由谁判断）
- **逆转**：迭代不是"重复直到对"，而是"用确定的框架（3 轮+DEBUG+门禁）管理非确定的输出"，把回退/震荡/局部最优纳入预期并设置停止标准

**层 3 — 非确定转化（阻碍→推动力）**：
- 输出非确定 = 多样性源 → 多次采样 + 选择 = 改进机制（呼应 "Sample More, Reflect Less" 2607.28576：重复采样可超结构化反思）
- 非确定 → 探索 G' 的多个可能解 → check 门禁+sense 批判+human 判断筛选 → 收敛到"最大化改进"
- **逆转**：非确定不是"要消除的噪声"，而是"探索 G'（未见过问题）所必需的变异源"——G' 本质未见过，确定性地输出反而可能错；非确定地探索+确定性门禁筛选，比"强行确定"更适配 G' 的开放性

### rick 现有机制如何填补逆转（尽调，基于代码+loop_2）

| 逆转层 | rick 现有机制 | 代码证据 | 置信度 |
|---|---|---|---|
| 层 1 可控性转移 | 确定性提取（rick↔外部存储 edge）| ContextManager.LoadOKR/SPEC/Debug/History from File + GenerateDoingPromptFile 注入 + SaveToFile 落盘（E-r2 节点 C）| 1.0（高）|
| 层 1 可控性转移 | 强制执行（doing.md"不可跳过"+doing_loop Step 0-5）| doing.md"你需要一步步执行以下操作，不可跳过任何步骤" + doing_loop 全程"必须/强制/自动触发"（E-r4 节点 C）| 1.0（高）|
| 层 2 非确定吸收 | 迭代框架（doing_loop 3 轮+DEBUG Phase 1-6）| Step 3 Sub Agent per-iteration + Step 4 失败返回 Step 3 + Step 5 3 轮上限/连续 2 轮同错停止（doing_loop.md）| 1.0（高）|
| 层 2 非确定吸收 | 失败模式管理（check 门禁+sense 批判门禁+human 判断）| runAutoFix 循环直到 pass（research-7 #11/12/13）+ sense_loop 批判门禁简报格式 + judgment.md human 判断（E-r4 节点 C + 批判门禁 briefs）| 0.9（高）|
| 层 3 非确定转化 | compaction 抗熵增（pi 自定义 compaction 保留 system prompt）| pi compaction 保留 system prompt + before_agent_start 每 turn 重建 + session_before_compact 自定义 summary（loop_2 research-4-N2/N3）| 1.0（高）|
| 层 3 非确定转化 | 探索性采样（DEBUG 遇红触发+多轮）| DEBUG 触发条件"测试 FAIL/编译报错/行为异常"自动触发，Phase 1-6 探索性调试（doing_loop Step 3 DEBUG）| 0.9（高）|

**逆转逻辑尽调置信度**：0.9（高）—— 三层结构均有 CONFIRMED 代码/文献支撑；A7 内禀不可消除是逆转起点（CONFIRMED）；rick 现有机制完整覆盖三层逆转。

---

## 3. 替代路径可选项（供 human 选择，含利弊，不推荐）

> 在 X（输出非确定）必然前提下，实现 Y（可靠解决 G′）的替代路径。每条路径含选项 a/b（或多）+ 利弊。**不替 human 推荐**——利弊并陈，等 human 决策。

### P1. compaction 策略（治理上下文熵增，承载 pi↔LLM edge）

| 选项 | 描述 | 利 | 弊 | 证据 |
|---|---|---|---|---|
| **P1a 自定义 compaction** | pi session_before_compact + customInstructions + 自定义 firstKeptEntryId + 保留 system prompt（作 compaction-resist 载体）| 上下文熵增可控；system prompt compaction-resist（长程确定性持久）；可标记"流程/方法"不可压缩 | 实现成本（需写 extension）；自定义 summary 质量依赖实现 | loop_2 research-4-N2/N3 |
| **P1b 默认 auto-compact** | claude code 默认 auto-compact（rick 不控制）| 零实现成本 | 不可控；system prompt 可能被压缩；长程确定性丢失（违反层 1 可控性转移）| research-7-N3（rick 不控制 compaction）|

### P2. 迭代框架对比（吸收输出非确定，承载 doing_loop+pi→LLM edge）

| 选项 | 描述 | 利 | 弊 | 证据 |
|---|---|---|---|---|
| **P2a rick sense+doing loop** | Step 3 Sub Agent per-iteration + 3 轮上限 + DEBUG Phase 1-6 + check 门禁 + sense 批判门禁 | 确定性编排+失败模式管理+门禁+停止标准（三层逆转全覆盖）；编排不在 G 需注入（A15）→ 确定性选择 | 编排不在 G（需强制注入，非零成本）；3 轮上限是否最优未验证（M8 nuance）；迭代策略优劣未 benchmark | E-r4 节点 C+D + doing_loop.md |
| **P2b Self-Refine** | 迭代精炼+自反馈（同一 LLM 作 generator/refiner/feedback）| 通用，无需 rick 编排；文献实证改善单轮 | 无外部门禁；无强制停止标准；无失败模式管理（回退/震荡/局部最优无管理）；无确定性提取 | arxiv 2303.17651（E-r4 节点 D）|
| **P2c Reflexion** | 语言反思+episodic memory（多轮反思，不更新权重）| 多轮反思；不更新权重（与 rick 上下文级一致）；linguistic feedback | 需 episodic memory 基础设施；无确定性提取；无强制执行；无 human 判断集成 | arxiv 2303.11366（E-r4 节点 D）|
| **P2d 重复采样（Sample More）** | 同等 token 成本下纯重复采样+选择 | 文献示可超 self-refine/Reflexion（2607.28576）；非确定→多样性→选择=改进；最简 | 无结构化收敛；无失败模式管理；纯随机，无确定性锚定；对 G' 开放问题可能效率低 | arxiv 2607.28576（E-r4 节点 D 边界）|

### P3. RAG vs 上下文工程（确定性提取的不同实现路径，承载 rick↔外部存储+LLM↔外部存储 edge）

| 选项 | 描述 | 利 | 弊 | 证据 |
|---|---|---|---|---|
| **P3a RAG** | 检索增强（non-parametric memory + 向量索引检索）| 通用；大语料检索；文献实证"parametric memory 访问有限→需 non-parametric" | 检索非确定（相似度匹配，非确定性提取）；无强制执行；无门禁；检索质量依赖索引 | arxiv 2005.11401 Lewis（E-r2 节点 C）|
| **P3b rick 上下文工程** | 文件载体（OKR/SPEC/task/debug/skills/loops）+ ContextManager 确定性加载 + 强制注入 + compaction-resist system prompt | 确定性提取（文件可版本控制/校验/重建）；强制执行；compaction-resist；三层逆转层 1 完整 | 需人工组织文件（非自动检索）；文件维护成本；覆盖面依赖 human 组织 | E-r2 节点 C（RFC-001+ContextManager）|

### P4. skill 系统级注册（提升确定性触发概率，承载 LLM↔外部存储 edge）

| 选项 | 描述 | 利 | 弊 | 证据 |
|---|---|---|---|---|
| **P4a skill 系统级注册** | pi skill 系统级（系统级触发，非依赖 LLM 选择）| 提升触发概率（系统级）；确定性触发（非随机选）；V3 价值（claude code 不能做）| 需 pi skill 系统级机制；迁移成本；skill 质量依赖维护 | loop_2 judgment V3 + research-3-N2 |
| **P4b prompt 文件路径引用** | 现状：WriteSkillFile 写路径到 prompt，LLM 按路径读 | 零迁移成本；现状已实现 | 触发概率低（非系统级）；依赖 LLM 选择（A15：zero-shot 不选 rick 编排）；非确定性触发 | research-7-N3（skill 加载机制）|

### P5. subagent 递归（分层迭代，承载 doing_loop Step 3 Main→Sub）

| 选项 | 描述 | 利 | 弊 | 证据 |
|---|---|---|---|---|
| **P5a pi subagent 递归** | doing_loop Step 3 Main→Sub Agent per-iteration（每轮独立 Sub Agent，上下文隔离）| 分层迭代；上下文隔离（每轮 Sub Agent 携压缩上下文）；V5 价值（claude code 不能做）；逆转层 2 强化 | 需 pi subagent 扩展；复杂度；Sub Agent 协调成本 | loop_2 judgment V5 + research-3-N4 + doing_loop Step 3 |
| **P5b 单 agent 无递归** | 现状：claude code 不用 subagent（单 agent 线性）| 简单；现状 | 无上下文隔离；无分层迭代；长程上下文熵增不可控 | research-7-N3（subagent 未使用）|

### P6. 二进制部署脱离 node（控制手段的部署形态，承载 rick↔pi edge 的运行时形态）

| 选项 | 描述 | 利 | 弊 | 证据 |
|---|---|---|---|---|
| **P6a pi 二进制编译** | pi 编译为自包含二进制，脱离 node | 自包含部署；V0 价值（claude code 不能做）；部署形态可控；环境一致 | 编译成本；二进制维护；调试复杂度 | loop_2 judgment V0 + research-3-N1 |
| **P6b 依赖 node** | 现状：claude code CLI 依赖 node 环境 | 零编译成本；现状 | 依赖 node 环境；部署受限；环境不一致风险 | research-7-N1（ai_cli 调用 claude 二进制）|

### 路径关系说明（非推荐，供 human 理解路径间关系）

- P1/P2/P3 直接对应逆转三层（P1=层 3 抗熵增，P2=层 2 吸收非确定，P3=层 1 确定性提取）；P4/P5/P6 是支撑性控制手段（P4 触发确定性，P5 分层迭代强化层 2，P6 部署形态）
- P2（迭代框架）选择影响 M8（rick 迭代框架 vs 重复采样的策略优劣）—— A18-Q2 未验证，是 R7 项
- P3（RAG vs 上下文工程）选择影响层 1 确定性提取的实现路径—— rick 现状是 P3b（上下文工程），P3a（RAG）是替代
- 各路径的 a 选项多为稳态 B（目标），b 选项多为稳态 A（现状）—— human 可在 N2 的 A→B 转化点选择哪些路径优先

---

## 尽调树快照

```
根：S-R — 逆转逻辑尽调 + 替代路径可选项
├─ 阻碍识别（X=输出非确定 / Y=有限迭代最大化改进）  [1.00 高] 继承 A7/A18 CONFIRMED
├─ 逆转逻辑（三层结构：可控性转移+非确定吸收+非确定转化）[0.90 高] 代码+文献支撑
│   └─ rick 现有机制填补逆转（6 机制，置信度 0.9-1.0）
├─ 替代路径可选项（P1-P6，12 选项）              [N/A] 供 human 选，不验证最优性
│   ├─ P1 compaction 策略（P1a/P1b）
│   ├─ P2 迭代框架（P2a sense+doing loop / P2b Self-Refine / P2c Reflexion / P2d 重复采样）
│   ├─ P3 RAG vs 上下文工程（P3a/P3b）
│   ├─ P4 skill 系统级注册（P4a/P4b）
│   ├─ P5 subagent 递归（P5a/P5b）
│   └─ P6 二进制部署（P6a/P6b）
└─ M8 迭代策略优劣（P2 的 R7 隐患）              [R7] rick 迭代框架最优性未 benchmark
```

## 节点详情

### 阻碍识别 — 置信度 1.00（高）
- 代码 ✅ 0.4（LLM node + pi→LLM→output edge，A7 CONFIRMED）
- 运行时 ✅ 0.3（5 次采样非确定 + G' 拒答）
- 文档 ✅ 0.2（Tishby/Delétang/幻觉调研）
- 反事实 ✅ 0.1（X 不可消除 = A7 内禀）
- 合计 1.0

### 逆转逻辑 — 置信度 0.90（高）
- 代码 ✅ 0.4（rick 6 机制填补三层逆转，全 CONFIRMED）
- 运行时 ✅ 0.3（zero-shot 非确定 vs rick 强制对照，E-r4 节点 C）
- 文档 ✅ 0.2（Self-Refine/Reflexion/Sample More/RAG/Extended Mind 支撑逆转三层）
- 反事实 ⚠️ 0.05（逆转的"非确定转化为推动力"层 3 是逻辑逆转，无直接 runtime benchmark 验证 rick 框架优于替代）
- 合计 0.95 → 0.9（保留 0.05 折扣反映层 3 转化的逻辑性而非实证性）

### 替代路径可选项 — 置信度 N/A
- 可选项含利弊并陈，不验证最优性（供 human 选）
- 每选项的证据置信度高（代码/文献 CONFIRMED），但"哪个最优"未 benchmark

## R7 上报项

| 项 | 置信度 | 理由 |
|---|---|---|
| M8 / P2 迭代策略优劣 | R7（未决）| rick sense+doing loop 迭代框架 vs Self-Refine/Reflexion/重复采样的最优性未 benchmark 验证（A18-Q2 未验证）；"Sample More Reflect Less"示重复采样可超结构化反思，rick 迭代框架的相对有效性是开放问题。**不影响逆转逻辑方向性**（单轮→多轮改善稳固），但影响 P2a vs P2b/c/d 的选择 |

> 阻碍识别/逆转逻辑均达高置信（1.0/0.9），无 R7。仅 M8 迭代策略优劣为 R7（供 human 在选 P2 时知悉未决）。

## 整合摘要

- **总节点数**：3（阻碍/逆转/可选项）+ 1 根 = 4
- **高置信度叶节点**：2（阻碍 1.0 / 逆转 0.9）
- **R7 上报**：1（M8 迭代策略优劣，P2 选择的开放问题）
- **替代路径**：6 条（P1-P6），12 个选项，含利弊并陈，不推荐

---

## human 启发性追问（照 sense_loop S-R 格式）

### 1. 如果 [LLM 输出非确定] 是不可避免的前提，实现 [可靠解决 G′] 的最意想不到的路径是什么？
（提示：逆转层 3 提出"非确定→多样性→选择=改进"——把阻碍转为推动力。最意想不到的路径可能是 P2d（重复采样+选择，最简）或 P3a（RAG，放弃确定性提取转检索非确定）。或者：不追求"可靠解决"而追求"最大化改进+human 判断认定"（接受非确定，把"可靠"重定义为"human 认定的最大化改进"）。你认为哪条最意想不到又最有效？）

### 2. 什么看似阻碍的力量（输出非确定/回退/震荡），其实可以转化为推动力？
（提示：输出非确定=探索 G'（未见过问题）的变异源（层 3）；回退/震荡=多路径探索的副产品（doing_loop Step 4 失败返回 Step 3）；局部最优=迭代深度的信号（Step 5 连续 2 轮同错=需升级人工）。这些"阻碍"若纳入"失败模式管理"框架（DEBUG/check/sense/human 判断），是否反而成为"最大化改进"的必要成分——确定性输入+非确定探索+门禁收敛=比"强行确定"更适配 G' 开放性？）

### 3. 在 [输出非确定] 必然的前提下，[可靠解决 G′] 的"逆向工程"是什么？
（提示：从"可靠解决 G'"倒推必须为真的条件——(a) 输入必须确定（层 1：ContextManager+强制执行）；(b) 迭代必须有限+有停止标准（层 2：3 轮上限+DEBUG+门禁，防无限震荡）；(c) "可靠"由谁认定（human 判断，M3-ext 未覆盖的"human 判断者不可替代性"，N2 审查 3 提示）；(d) 非确定必须被利用而非消除（层 3）。逆向工程揭示：可靠≠确定输出，而=确定输入+有限非确定探索+门禁收敛+human 认定。你同意这个"可靠"的重定义吗？还是你认为"可靠"应有更强的含义？）

---

## 安全约束确认

- 无 rick 代码修改 → 无需 `git restore` 还原
- 所有 Read/Grep/curl 只读；复用前序 brief 已标注来源
- 可选项含利弊并陈，不替 human 推荐（遵守 S-R 禁止倾向性）

## 信源清单（复用，本轮无新抓取）

- rick 代码（`/workdir/sunquan20/AI_CODING/rick`）：context.go/doing_prompt.go/doing.md/doing_loop.md/think.md/RFC-001 + internal/agent/claudecode（research-7）
- loop_2 briefs：research-7（13 调用点+check 门禁）/ research-4-N2/N3（pi compaction/扩展点）/ research-3-N1/N2/N4（二进制/skill 系统级/subagent）/ judgment（V0-V5 价值）
- E-r2/E-r4 briefs：节点 A/B/C/D（A7 有损非确定/A15 确定性选择/A18 单轮不足多轮改善）
- 文档：Self-Refine 2303.17651/Reflexion 2303.11366/Sample More 2607.28576/Plan-and-Solve 2305.04091/RAG Lewis 2005.11401/Extended Mind/Delétang 2309.10668/Tishby physics/0004057+1503.02406（E-r2/E-r4 已抓取）
