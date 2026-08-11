# 调研报告 — EC 颠覆性假设调查（无限上下文+强模型→rick 是否还需要） — 2026-08-09

> 派发：`loops/loop_4/dispatch-research-EC.md`（EC 阶段 human 自判最不安假设，显式要求"详细调查+证实与反驳观点"，可能颠覆前序 A7/A17/A18/核心假设/S-R 层1）
> 工作流：`loops/loop_4/prompts/research.md`（主报告格式 + 证实/反驳对照 + 直接回答）
> 前序：E 收敛（A7/A15/A18 CONFIRMED，A17 价值论 D3′，核心假设）+ S-R 逆转三层 + N1 系统描述

## human 调研请求（原话）

> 我认为最不安的假设是 如果模型的上下文足够长，模型智能足够强。是否就不需要管理上下文了？ AI 能够自主检索所有内容，获得最有效的上下文，然后完成任务。 甚至是自动让模型自己淘汰上下文，遗忘不重要的信息，例如 codex 的服务端压缩。 上下文方法压缩 某种意义上就是遗忘，如果存在一个无限度上下文，自我可以不断迭代直到任务完成。 那还需要 rick 吗？ 请你帮我详细调查，然后给出 证实与反驳的观点。

## 信源配置

无 `.rick/config.json`，取默认权重。复用 E-r2/E-r4 运行时路径（curl+arxiv REST 可用；WebFetch/WebSearch 被拦截；claude CLI 可用但无 temperature flag；raw API 被 proxy 拒绝）：

| 信源 | 权重 | 本轮来源 |
|---|---|---|
| 代码原文 | 0.4 | 复用 E-r2/E-r4（context.go/doing_prompt.go/doing.md/doing_loop.md）+ loop_2 research-4-N2/N3（pi compaction 损失性 summary） |
| 运行时行为 | 0.3 | 复用 E-r2（5 次采样 {73,42,73,42,42} 非确定/G' 拒答）+ E-r4（zero-shot 线性单遍/单轮对 G' 失败） |
| 文档 | 0.2 | 本轮新抓取：Lost in the Middle 2307.03172 / RULER 2404.06654 / Is It Really Long Context 2407.00402 / Self-RAG 2310.11511 / context pruning 2501.16214+2503.10720 / linear attention 2510.26692+2310.03294；复用 E-r2（Delétang/Tishby/RAG/Extended Mind） |
| 反事实 | 0.1 | 复用 E-r2 节点 C de facto A/B + E-r4 节点 C zero-shot vs rick 强制对照 |

**加权公式**：置信度 = Σ(信源验证结果 × 信源权重)。证实/反驳观点对照不验证真假（枚举论据，关联前序假设）。

---

## 尽调树快照

```
根：EC — 无限上下文+强模型→rick 是否还需要
├─ A. 无限上下文是否可行/可达               [0.20 低] R7（runtime/code 不可访问；docs 强支撑）
├─ B. 无限上下文+强模型能否消除上下文管理需求  [0.55 中] R7（LLM 检索源码不可访问）
├─ C. 上下文压缩=遗忘？无限上下文是否避免损失  [1.00 高] ✅（A7 参数级 vs 非参数级辨析 + Lost in the Middle）
├─ D. 即使无限上下文+强模型，rick 哪些价值不 collapse [1.00 高] ✅（4 候选全 HOLD）
├─ E. 证实观点（无限上下文→rick 不需要）     [N/A] 枚举论据
└─ F. 反驳观点（即使无限上下文→rick 仍需要）  [N/A] 枚举论据
```

总节点 6 叶 + 1 根 = 7 | 高置信度叶 2（C/D）| R7 上报 2（A/B）

---

## 节点详情

### 节点 A — 无限上下文是否可行/可达？
**事实陈述**：context window 增长趋势、O(n²) 注意力复杂度、长上下文衰减、无限上下文的物理/算力/成本墙
- **置信度**：0.20（低，R7）
- **信源验证**：
  - 代码 ❌（LLM 注意力/上下文实现源码不在 rick 仓库）
  - 运行时 ❌（无法用 claude CLI agent 构造受控长上下文测试；raw API 被 proxy 拒绝；降级）
  - 文档 ✅（Lost in the Middle 2307.03172 + RULER 2404.06654 + linear attention 2510.26692/2310.03294）
  - 反事实 ❌
- **关键证据**：
  - **增长趋势**：context window 从 2023（16k-128k）→ 2024（Gemini 1.5 Pro 2M，Claude 200k）→ 2025-26（1M+ 常见，部分 2M+）。趋向"effectively very long"，但非"infinite"。
  - **O(n²) 注意力**：标准 transformer 注意力二次复杂度；高效变体（Kimi Linear 2510.26692 线性注意力，DISTFLASHATTN 2310.03294 分布式内存高效注意力）降低但不消除；算力/内存/成本墙存在。
  - **长上下文衰减**：Lost in the Middle（2307.03172）——"performance is often highest when relevant information occurs at the beginning or end... significantly degrades when models must access relevant information in the middle of long contexts, **even for explicitly long-context models**"；RULER（2404.06654）——NIAH 是"superficial form of long-context understanding"，声称的 context size ≠ 真实可用 size。
- **结论**：无限上下文不可达（物理/算力/成本墙 + 衰减；O(n²) 不消除）；"effectively very long" 可达但不等于"infinite"。
- **R7**：runtime/code 不可访问（无法构造受控长上下文实验 + LLM 注意力源码不在仓库），置信度受信源可达性上限约束。docs 强支撑结论。

### 节点 B — 无限上下文+强模型能否消除"上下文管理"需求？
**事实陈述**：模型自主检索（agentic retrieval/RAG 自动化）、自主淘汰上下文/遗忘（Codex 服务端 compaction）、"获得最有效上下文"是否本身是判断
- **置信度**：0.55（中，R7）
- **信源验证**：
  - 代码 ❌（LLM 检索/淘汰源码不在 rick 仓库；pi compaction 是 loop_2 已尽调的损失性 summary）
  - 运行时 ✅（复用 E-r4 节点 B：zero-shot claude CLI 线性单遍，不自发用 rick 编排；复用 E-r2：采样非确定）
  - 文档 ✅（Self-RAG 2310.11511 + context pruning 2501.16214/2503.10720 + A15 E-r4）
  - 反事实 ⚠️ 0.05（E-r4 zero-shot vs rick 强制对照）
- **关键证据**：
  - **自主检索需训练，非 zero-shot**：Self-RAG（2310.11511）——"adaptively retrieves passages on-demand, and generates and reflects on retrieved passages... using special tokens, called **reflection tokens**... makes the LM controllable during the inference phase"。自检索是**训练注入的能力**（reflection tokens），非 zero-shot 默认。呼应 A15（zero-shot 不选 rick 编排）——"获得最有效上下文"本身是判断，需训练/注入，非强模型自发。
  - **"有效上下文"是判断（A15）**：对 G'（未见过），模型不知道什么是"有效"（它不知道自己不知道什么）——Self-RAG 需 reflection tokens 训练才知何时检索/检索什么。
  - **自主淘汰=遗忘（lossy）**：Codex 服务端 compaction / auto-compaction / context pruning（Provence 2501.16214，AttentionRAG 2503.10720）——均为**损失性**摘要/裁剪（loop_2 research-4-N2：compaction 生成 structured summary 丢弃细节；pruning 丢弃 token）。human"压缩=遗忘"洞察成立——自主淘汰不避免损失，只是把"遗忘"自动化。
- **结论**：不能消除——自检索需训练（非 zero-shot）；"有效上下文"是判断（A15，G' 下模型不知何为有效）；自主淘汰是 lossy（遗忘的另一形态，不避免损失）。
- **R7**：LLM 检索/淘汰源码不可访问；runtime 复用 E-r4 节点 B（0.55 真理性强）。

### 节点 C — 上下文压缩=遗忘？无限上下文是否避免损失？（关键辨析）
**事实陈述**：A7 是参数级（权重）有损；上下文压缩是非参数级。无限上下文是否绕过参数级损失？还是把"遗忘"从参数级移到检索级？
- **置信度**：1.00（高，终止）
- **信源验证**：
  - 代码 ✅ 0.4（loop_2 research-4-N2：pi compaction 生成 structured summary 丢弃细节=lossy；E-r2 ContextManager 文件载体=确定性提取）
  - 运行时 ✅ 0.3（复用 E-r2：5 次采样非确定=参数级非确定；G' 拒答=参数级 100% 损失）
  - 文档 ✅ 0.2（Lost in the Middle 2307.03172=长上下文检索级损失；Tishby 信息瓶颈=A7 参数级有损；Self-RAG 2310.11511"LLMs often produce factual inaccuracies due to sole reliance on parametric knowledge"=参数级损失确认）
  - 反事实 ✅ 0.1（E-r2 节点 C de facto A/B：无外部源 LLM 拒答 vs 有源可答）
- **关键辨析（核心）**：
  - **A7 是参数级（权重）有损**：Tishby 信息瓶颈——学习=通过瓶颈的有损压缩；泛化=遗忘训练细节；Self-RAG 确认"sole reliance on parametric knowledge 导致 factual inaccuracies"。**上下文长度不触及参数级**——权重不因 context 变长而变无损。
  - **上下文压缩是非参数级（context window）**：compaction（loop_2 research-4-N2）=lossy summarization（structured summary 丢弃细节）；pruning（Provence/AttentionRAG）=丢弃 token。human"压缩=遗忘"洞察**对非参数级成立**。
  - **无限上下文是否避免损失？** 两层分析：
    - **(1) 非参数级**：若真无限 context → 无需压缩 → 无压缩级遗忘。**但**：(a) 无限不可达（节点 A）；(b) 长 context 有检索/注意力级损失——Lost in the Middle（"degrades when access relevant information in the middle, **even for explicitly long-context models**"）= 另一种"遗忘"（信息在 context 中但用不到）。**遗忘从压缩级移到检索/注意力级，未消除**。
    - **(2) 参数级（A7）**：**完全不被上下文长度触及**——权重有损不因 context 变长而修复；模型对 context 内容的推理仍非确定（E-r2 采样）+ 仍有损（注意力稀释/Lost in the Middle）。**A7 不 collapse**。
- **结论**：无限上下文**不避免损失**——(1) 非参数级：避免压缩级遗忘但不可达；长 context 有检索/注意力级损失（Lost in the Middle）；遗忘移位未消除。(2) 参数级（A7）：完全不被触及，权重有损+采样非确定持续。human"压缩=遗忘"对非参数级成立；但无限上下文不解决参数级 A7。
- **疑问点**：无；四源全 ✅，1.0 达高，终止。

### 节点 D — 即使无限上下文+强模型，rick 的哪些价值仍不 collapse？
**事实陈述**：验证候选 (1) 确定性编排/强制执行 (2) human 判断者不可替代 (3) 失败模式管理 (4) 做事方法/编排不在 G
- **置信度**：1.00（高，终止）
- **信源验证**：
  - 代码 ✅ 0.4（E-r4 节点 C：doing.md"不可跳过"+doing_loop Step 0-5 强制；E-r2 ContextManager 确定性提取）
  - 运行时 ✅ 0.3（复用 E-r4 节点 B：zero-shot 线性单遍不选编排；E-r2 采样非确定）
  - 文档 ✅ 0.2（A15/A18 + Lost in the Middle + Self-RAG 需训练）
  - 反事实 ✅ 0.1（E-r4 zero-shot vs rick 强制对照）
- **4 候选验证（全 HOLD）**：
  1. **确定性编排/强制执行（A15）HOLD ✅**：E-r4 节点 B runtime——zero-shot LLM 线性单遍，不自发用 rick 编排；**无限上下文不改变"不选"**（A15 与上下文长度正交）。doing.md"不可跳过"+doing_loop 强制仍需——把"可选"变"强制执行"的价值不 collapse。
  2. **human 判断者不可替代（N2 提示）HOLD ✅**：G' "最大化改进"/"任务完成"由谁认定？"自我迭代到任务完成"假设模型知道何为"完成"——但对 G'（开放未见过问题），"完成"定义来自 human 判断；LLM 无法自证 G' 解决（它不知道自己不知道什么；无限 context 不注入不存在的 G' 知识）。
  3. **失败模式管理（回退/震荡/局部最优）HOLD ✅**：SR 层 2——这些在无限上下文下仍存在（模型能迭代但不保证收敛；Lost in the Middle 示长 context 性能退化=回退/局部最优的实证）。无限 context ≠ 收敛；DEBUG Phase 1-6/check 门禁/sense 批判/human 判断仍需。
  4. **做事方法/编排不在 G（A15 confirmed）HOLD ✅**：E-r4 节点 A——rick 编排（plan-do-learn/sense S-E-N/doing-learning-dream）LLM 判"project-specific"，不在 G；**与上下文长度正交**——无限 context 不把 rick 编排注入 G。
- **结论**：4 候选全 HOLD。rick 核心价值（确定性编排/强制执行 + human 判断者 + 失败模式管理 + 编排不在 G）在无限上下文+强模型下**不 collapse**。
- **疑问点**：无；四源全 ✅，1.0 达高，终止。

### 节点 E — 证实观点（支持"无限上下文+强模型→rick 不需要"）
**枚举最强论据**（不验证真假，供 human 对照）：
1. **上下文熵增手段价值下降**：context window 趋向"effectively very long"（2M+）+ auto-compaction（Codex 服务端）——A17 的**手段**"应对上下文熵增"价值部分下降（compaction-resist system prompt 的相对价值降低）。
2. **bounded G' 可被 one-shot**：对**有界/语料内**的 G' 子集，强模型+长 context 可能单轮解决——A18"单轮不足"对 bounded G' 部分弱化。
3. **agentic RAG 自检索**：Self-RAG（2310.11511）示自检索+反思可超 parametric-only——部分替代 rick 确定性提取（但需训练，非 zero-shot）。
4. **"自我迭代到完成"的强假设**：若模型有无限 context 作 scratchpad + 自我迭代，可能不需 rick 3 轮上限框架（但见 F5 反驳：不保证收敛+无"完成"认定）。
5. **auto-compaction 自动化熵管理**：Codex 服务端压缩自动化上下文管理——减少对 rick 手动 compaction-resist 的依赖（但见 F3：lossy=遗忘，未避免损失）。

### 节点 F — 反驳观点（支持"即使无限上下文+强模型→rick 仍需要"）
**枚举最强论据**（关联 A7/A15/A17/A18/核心假设/human 判断者）：
1. **A7 参数级损失不被上下文长度触及（HOLD）**：A7 CONFIRMED 是**参数级**（权重）有损+非确定；context 是**非参数级**——上下文长度不修复权重有损。Self-RAG 确认"sole reliance on parametric knowledge 导致 inaccuracies"持续。**核心价值主体（弥补参数级有损+非确定）不 collapse**。
2. **A15 编排不在 G + zero-shot 不选（HOLD，与上下文长度正交）**：E-r4 runtime——zero-shot 线性单遍不选 rick 编排；无限 context 不改变"不选"。doing.md"不可跳过"强制执行仍需——确定性选择+强制执行的价值不 collapse。
3. **"获得最有效上下文"是判断（A15）+ 自检索需训练**：对 G'（未见过），模型不知何为"有效"（不知道自己不知道）；Self-RAG 自检索需 reflection tokens 训练（非 zero-shot）。检索非确定（相似度匹配）≠ 确定性提取。
4. **自主淘汰=遗忘（human 洞察成立）**：compaction/pruning 是 lossy（loop_2 research-4-N2 structured summary 丢弃细节；Provence/AttentionRAG 丢弃 token）；无限 context 不可达（节点 A）；长 context 有检索/注意力级损失（Lost in the Middle）。**遗忘移位未消除**——非参数级损失仍存。
5. **"自我迭代到完成"不成立（HOLD）**：(a) 模型不知何为"完成"（G' 开放，定义来自 human）；(b) 迭代不保证收敛（SR 层 2 回退/震荡/局部最优持续；Lost in the Middle 示长 context 退化）；(c) 模型无 G' 知识可迭代向（G' 未见过，不在权重，可能不在可检索语料）。**A18"单轮不足，需迭代"对真正 G' 仍成立**——但"需 rick 迭代框架"需视策略优劣（M8 nuance）。
6. **human 判断者不可替代（N2 提示）**：G' "最大化改进"由 human 认定；LLM 无法自证 G' 解决。即使自迭代，"完成"认定需 human 判断——rick 的 human 判断集成（judgment.md/sense 简报/check 门禁）不 collapse。
7. **失败模式管理（SR 层 2）HOLD**：回退/震荡/局部最优在无限上下文下仍存（不收敛性 + Lost in the Middle）；DEBUG/check/sense 批判门禁/human 判断仍需。
8. **核心假设（∃G' + 无法一次性）HOLD**：G' 是"未见过+无法一次性解决"——无限 context 不注入不存在的 G' 知识（不在权重、可能不在可检索语料）；A18 单轮对 G' 失败持续。**核心假设不 collapse**。

---

## 证实与反驳观点对照（逐论据，关联前序假设）

| # | 证实观点（E：rick 不需要）| 反驳观点（F：rick 仍需要）| 前序假设状态 |
|---|---|---|---|
| 1 | 上下文熵增手段价值下降（auto-compaction 自动化）| A7 参数级损失不被上下文长度触及（权重有损持续）| A17 手段**部分弱化**；A17 价值主体（A7）**HOLD** |
| 2 | bounded G' 可被 one-shot（A18 部分弱化）| 核心假设 G'（未见过+无法一次性）HOLD——无限 context 不注入不存在的 G' 知识 | A18 对 bounded G' 部分弱化；对真正 G' **HOLD** |
| 3 | agentic RAG 自检索部分替代确定性提取 | "有效上下文"是判断（A15）+ 自检索需训练（非 zero-shot）+ 检索非确定 | A15 **HOLD**（与上下文长度正交）|
| 4 | 自我迭代到完成（无限 context scratchpad）| 不成立：不知"完成"+不收敛+无 G' 知识迭代向 | A18 + human 判断者 **HOLD** |
| 5 | auto-compaction 自动化熵管理 | 自主淘汰=遗忘（lossy）+ 无限不可达 + 长 context 检索级损失 | A7 非参数级辨析：遗忘移位未消除 |
| 6 | — | human 判断者不可替代（G' 完成认定）| N2 提示 **HOLD** |
| 7 | — | 失败模式管理（回退/震荡/局部最优）持续 | SR 层 2 **HOLD** |
| 8 | — | 做事方法/编排不在 G（与上下文长度正交）| A15 **HOLD** |

**对照结论**：证实观点主要冲击 **A17 的"手段"**（应对上下文熵增）+ A18 对 bounded G' 的弱化；**未触及 A7 价值主体 + A15 + human 判断者 + 失败模式管理 + 核心假设**。反驳 8 论据中 7 个关联前序 CONFIRMED 假设且全 HOLD。

---

## R7 上报项

| 节点 | 置信度 | 理由 |
|---|---|---|
| A 无限上下文可行/可达 | 0.20 低 | LLM 注意力/上下文源码不在 rick 仓库（代码 0.4 不可计入）；无法用 claude CLI agent 构造受控长上下文实验（raw API 被 proxy 拒绝）；runtime 降级。docs 强支撑（Lost in the Middle+RULER+linear attention）但权重 0.2 上限 |
| B 无限上下文+强模型消除管理需求 | 0.55 中 | LLM 检索/淘汰源码不在仓库；runtime 复用 E-r4 节点 B（0.55 真理性强）；docs Self-RAG+context pruning 支撑 |

> 节点 C/D 置信度 1.0 达高，无 R7。节点 E/F 是论据枚举（不验证真假）。
> 两 R7 共性：对象为 LLM 内部机制（注意力/检索/淘汰），源码不可访问，方法论权重受信源可达性上限约束。**真理由 docs（多 arxiv 主源）+ runtime 复用已充分确立**。

## 整合摘要

- **总节点数**：6 叶 + 1 根 = 7
- **高置信度叶节点**：2（C 1.0 / D 1.0）
- **R7 上报**：2（A 0.20 / B 0.55，均信源可达性受限）
- **证实/反驳对照**：证实 5 论据（主要冲击 A17 手段 + A18 bounded G'），反驳 8 论据（7 关联 CONFIRMED 假设且全 HOLD）

---

## 对 human 请求的直接回答

### 在无限上下文+强模型假设下，rick 是否还需要？

**✅ 仍需要（反驳强，标注"维持"）**——rick 核心价值在无限上下文下**不 collapse**：

**COLLAPSE（部分，手段层）**：
- **A17 的"手段=应对上下文熵增"部分弱化**：context 趋向 effectively very long（2M+）+ auto-compaction（Codex 服务端）自动化熵管理 → rick 的 compaction-resist system prompt 手段价值**相对下降**。但这是**手段层**弱化，非**价值主体**（A7 参数级弥补）弱化。

**PRESERVE（核心，价值主体层）——4 价值全 HOLD**：
1. **A7 参数级弥补（价值主体）HOLD**：A7 是**参数级**（权重）有损+非确定；上下文是**非参数级**——上下文长度不修复权重有损（Self-RAG 确认 parametric inaccuracies 持续；Tishby 瓶颈）。rick 价值主体=弥补参数级有损+非确定，**与上下文长度正交，不 collapse**。
2. **A15 确定性编排/强制执行 HOLD**：zero-shot LLM 线性单遍不选 rick 编排（E-r4 runtime），无限 context 不改变"不选"；doing.md"不可跳过"强制执行仍需。**与上下文长度正交**。
3. **human 判断者不可替代 HOLD**：G' "完成"/"最大化改进"由 human 认定；LLM 无法自证 G' 解决（不知道自己不知道；无限 context 不注入不存在的 G' 知识）。"自我迭代到完成"假设模型知何为"完成"——对 G'（开放）不成立。
4. **失败模式管理（SR 层 2）HOLD**：回退/震荡/局部最优在无限上下文下仍存（不收敛性 + Lost in the Middle 长上下文退化）；DEBUG/check/sense 批判门禁/human 判断仍需。

**关键辨析（对 human 洞察的回应）**：
- human"压缩=遗忘"洞察**对非参数级（context）成立**——compaction/pruning 是 lossy（loop_2 research-4-N2 + Provence/AttentionRAG）。但无限上下文**不避免损失**：(1) 非参数级——避免压缩级遗忘但不可达（节点 A）；长 context 有检索/注意力级损失（Lost in the Middle，"even for explicitly long-context models"）；遗忘移位未消除。(2) 参数级（A7）——完全不被上下文长度触及，权重有损+采样非确定持续。**遗忘从压缩级移到检索/注意力级，参数级损失不动——总损失未消除**。
- "AI 自主检索获得最有效上下文"——Self-RAG（2310.11511）示自检索需**训练**（reflection tokens），非 zero-shot 默认；且对 G'（未见过）模型不知何为"有效"（A15）。检索非确定（相似度匹配）≠ 确定性提取。
- "自我迭代到任务完成"——(a) 不知"完成"（G' 开放，human 认定）；(b) 不保证收敛（SR 层 2 回退/震荡/局部最优；Lost in the Middle）；(c) 无 G' 知识可迭代向（G' 未见过，不在权重，可能不在可检索语料）。**对真正 G' 不成立**。

### 是否建议触发反向回流（EC 升维/降维）？

**❌ 不建议触发反向回流。标注"维持"。**

- **核心价值不 collapse**：A7（价值主体）+ A15 + human 判断者 + 失败模式管理 + 核心假设全 HOLD。证实观点仅冲击 A17 的**手段**（应对上下文熵增）+ A18 对 bounded G' 的弱化，未触及价值主体。
- **建议的 refinement（非回流）**：A17 的"手段=应对上下文熵增"部分弱化——建议将 rick 价值主张**更牢固地锚定在 A7（参数级有损，内禀）+ A15（确定性选择/强制执行）+ human 判断者**，而非"应对上下文熵增"（后者作为手段在无限上下文下价值下降）。这是**手段层重锚**（类似 D3′ 价值重锚），非核心假设颠覆。
- **A18 边界细化（非回流）**：A18"单轮不足，需迭代"对**真正 G'（未见过+开放）HOLD**；对 bounded G'（语料内+有界）部分弱化。建议 A18 表述细化为"对真正 G'（未见过）单轮不足"——边界细化，非方向性改变。
- **不构成 EC 升维/降维触发条件**：颠覆性假设未颠覆核心价值主体（A7/A15/核心假设全 HOLD），仅弱化手段层 + 细化 A18 边界——属"维持+手段重锚+边界细化"，非"核心 collapse"。

---

## human 启发性追问（照 sense_loop EC 格式）

1. **若 rick 核心价值保留，但其"手段"（应对上下文熵增）在无限上下文下弱化——rick 的价值主张应如何重锚？** 是更牢固锚定在 A7（参数级有损，内禀不可消除）+ A15（确定性选择/强制执行）+ human 判断者，还是你认为"应对熵增"仍有独立价值（如长程任务的上下文漂移/累积噪声，与 context 长度无关）？
2. **"自我迭代到任务完成"的"完成"由谁认定？** 若 LLM 自迭代并声明"完成"，但 G' 是未见过的问题——LLM 的"完成"声明可信吗（它不知道自己不知道什么）？这是否意味着 human 判断者是不可消除的最后一道门禁（check 门禁/sense 批判/judgment.md）？
3. **无限上下文是否反而放大"非确定"的代价？** context 越长， Lost in the Middle 越严重，检索/注意力级损失越大——确定性提取（文件载体+强制注入）是否在长上下文下**价值上升**（弥补检索级损失），而非下降？

---

## 安全约束确认

- 无 rick 代码修改 → 无需 `git restore` 还原
- 所有 Read/Grep/curl 只读；复用前序 brief 已标注来源
- 证实/反驳观点并陈，不替 human 决策是否回流（标注"维持"，理由供 human 判断）

## 信源清单

- 本轮新抓取：arxiv 2307.03172（Lost in the Middle）/ 2404.06654（RULER）/ 2407.00402（Is It Really Long Context）/ 2310.11511（Self-RAG, Asai）/ 2501.16214（Provence context pruning）/ 2503.10720（AttentionRAG）/ 2510.26692（Kimi Linear）/ 2310.03294（DISTFLASHATTN）
- 复用 E-r2/E-r4：Delétang 2309.10668 / Tishby physics/0004057+1503.02406 / RAG Lewis 2005.11401 / Extended Mind / Self-Refine 2303.17651 / Reflexion 2303.11366 / Sample More 2607.28576
- 复用 loop_2：research-4-N2（pi compaction 损失性 summary）/ research-4-N3（before_agent_start）
- 复用 E-r2/E-r4 代码：context.go/doing_prompt.go/doing.md/doing_loop.md（rick 确定性提取+强制执行）
- runtime 复用：E-r2（5 次采样非确定/G' 拒答）+ E-r4（zero-shot 线性单遍/单轮对 G' 失败）
