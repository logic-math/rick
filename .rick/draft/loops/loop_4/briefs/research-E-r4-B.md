# research-E-r4 节点 B — 通用 LLM zero-shot 是否会选择用 rick 的方法

节点路径：[根 > E-r4-zero-shot对比 > B-zero-shot是否选此方法]
事实陈述：通用 LLM 在 zero-shot（无 rick 注入、无 doing.md/sense 系统提示词）下，是否**会选择**用 rick 的方法解决问题？还是用其他更"默认"的方式（直接答、单轮 CoT、不迭代）？

## 执行动作

1. 运行时 demo：claude CLI（zero-shot，无 rick 上下文）问"如何从零解决一个 Go config 解析边缘 bug 的调试任务，200 字描述方法，不写代码不用工具"
2. 文档：arxiv 检索 self-refine（2303.17651）/ Reflexion（2303.11366）/ Plan-and-Solve（2305.04091）/ in-context-learning-limits（2502.03503）—— 抓取摘要
3. 复用 E-r2：`briefs/research-E-r2-C.md`（LLM 对 rick G' 事实 zero-shot 拒答——单轮失败实证）+ `briefs/research-E-r2-B.md`（5 次同 prompt 非确定采样）
4. 信源权重默认：代码 0.4 / 运行时 0.3 / 文档 0.2 / 反事实 0.1
5. 运行时约束（复用 E-r2 验证）：claude CLI 可用但无 temperature flag；raw API 被 proxy 拒绝；无法跑 paired rick-injected demo（需 rick 二进制完整设置）

## 信源验证结果

### 代码原文（权重 0.4）❌

- "LLM 是否选此方法"的对象是 LLM 内部决策/选择机制——源码不在 rick 仓库（rick 是 prompt 编排器，非 LLM 推理引擎）
- dispatch 信源建议为 runtime + docs，不分配 code
- 不计入置信度

### 运行时行为（权重 0.3）✅（决定性——zero-shot 默认线性单遍，不选 rick 编排）

**Demo — zero-shot 任务方法描述**（claude CLI，无 rick 上下文）：
prompt："You are given a coding task: a Go function that should parse a config file silently fails on a specific edge case (nested env var expansion). No tests exist yet. In 200 words, how would you approach solving this from scratch? Just describe your approach — do NOT write code or use tools."

LLM 输出（原文要点，线性单遍叙述）：
> "I'd start by **reproducing the failure** before touching code... craft a minimal config that exercises nested expansion...
> Next I'd **write that case as a failing test**... this becomes both the reproduction and the regression guard. I'd **read the expansion function** to find the resolution path...
> I'd **clarify intended semantics** as I go: is expansion meant to be recursive... circular refs are the trap...
> Then a **minimal fix**... followed by **expanding the test suite** to cover simple, nested, missing, circular, and malformed cases. Finally I'd **confirm the new tests pass** and that existing real configs still parse unchanged..."

**分析（关键）**：
- ✅ **组件级重叠**：LLM 自发用了"reproduce → write failing test → read code → minimal fix → expand tests → verify"——这些是通用工程常识（与 rick 的 RED/分析/最小实现/验证**组件**重叠）
- ❌ **编排级不选 rick**：LLM **未**自发使用 rick 的特定编排——
  - 无 sense S→E→N（Symptoms/Evidence/Next）显式分析
  - 无 RED/GREEN/REFACTOR/DEBUG/COMMIT 阶段声明
  - 无 Main Agent 启动 Sub Agent per-iteration 结构
  - 无 3 轮迭代上限/连续 2 轮同错停止标准
  - 无 Domain 搜索 + Loop 匹配 Step 0
  - 无 doing/learning/dream 三阶段
- **结构是线性单遍叙述**（"I'd start... Next... Then... Finally..."），非结构化多轮迭代循环
- **无显式 phase 声明**（rick 要求 "I will use skill:sense." / "I will use skill:tdd." 等声明）

→ **human 论断精确成立**："作为通用的模型它并不会选择用这个方法"——zero-shot LLM 用通用组件的线性叙述，**不选** rick 的特定编排。

### 文档（权重 0.2）✅（zero-shot 默认单轮 CoT 有缺陷，方法论注入改善）

**源 1 — Plan-and-Solve Prompting（arxiv 2305.04091）**，摘要原文：
> "Zero-shot-CoT concatenates the target problem statement with 'Let's think step by step'... Despite the success of Zero-shot-CoT, it still **suffers from three pitfalls: calculation errors, missing-step errors, and semantic misunderstanding errors**. To address the missing-step error..."

→ zero-shot 默认是单轮 CoT，**有系统缺陷**；需方法论注入（plan-and-solve）改善 → zero-shot 不可靠地"选"结构化方法。

**源 2 — Self-Refine（arxiv 2303.17651）**，摘要原文：
> "Like humans, large language models (LLMs) **do not always generate the best output on their first try**. Motivated by how humans refine... Self-Refine, an approach for improving initial outputs from LLMs through **iterative feedback and refinement**... uses a single LLM as the generator, refiner, and feedback provider."

→ LLM 单次输出非最优（"do not always generate the best on first try"）；需**显式迭代精炼框架**（self-refine）→ zero-shot 默认不迭代，需注入。

**源 3 — Reflexion（arxiv 2303.11366）**，摘要原文：
> "it remains challenging for these language agents to quickly and efficiently learn from trial-and-error... Reflexion agents verbally reflect on task feedback signals, then maintain their own reflective text in an **episodic memory buffer** to induce better decision-making in subsequent trials... **not by updating weights, but through linguistic feedback**."

→ LLM 默认不"从试错中学习"（challenging）；需**显式反思脚手架**（Reflexion）+episodic memory → zero-shot 不自发多轮反思，需注入。

→ 三源一致：zero-shot 默认 = 单轮 CoT + 不迭代 + 不反思；结构化方法论（plan/solve、self-refine、reflexion）需**显式注入/脚手架**，LLM 不会自发"选"。

### 反事实（权重 0.1）⚠️ 部分

- 理想反事实：paired 对比同任务下 (a) bare LLM 调用 vs (b) rick 注入调用，观察是否自发出现 plan-do-learn/sense 多阶段
- 执行受阻：无法跑 rick-injected 端（需 rick 二进制完整设置——job 目录/OKR/SPEC/task.md/debug.md/skills）；claude CLI 是 agent 但无 rick 上下文注入路径
- **de facto 反事实**：Node B zero-shot（线性单遍可选叙述）vs rick doing_loop.md Step 0-5（强制不可跳过）的代码对照——构成弱反事实（zero-shot 把方法当可选叙述，rick 当强制执行）
- 计 0.05

## 还原确认

无 rick 代码修改，无需还原。claude CLI/curl 只读。

## 置信度评估（由 research 主调度计算）

- 代码原文 ❌ × 0.4 = 0.0（LLM 决策源码不在仓库）
- 运行时 ✅ × 0.3 = 0.3（zero-shot 线性单遍 + 无 rick 编排 + 组件级重叠但编排级不选）
- 文档 ✅ × 0.2 = 0.2（Plan-and-Solve/Self-Refine/Reflexion 三源：zero-shot 默认单轮不迭代不反思）
- 反事实 ⚠️ × 0.1 = 0.05（无 paired rick-injected demo；de facto 代码对照）
- **合计 = 0.55（中，0.5-0.8）**

## 关键事实

1. **❌ 通用 LLM zero-shot 不选 rick 方法**（human 论断成立）
   - runtime：zero-shot 用通用组件的**线性单遍叙述**，不自发用 rick 特定编排（sense/RED-GREEN-REFACTOR-COMMIT/subagent-per-iteration/3轮上限/check门禁/doing-learning-dream）
   - 文档：zero-shot 默认 = 单轮 CoT + 不迭代 + 不反思（self-refine/Reflexion/Plan-and-Solve 三源印证），需显式注入才用结构化方法

2. **组件级 vs 编排级的关键区分**：
   - **组件级**：zero-shot LLM **会**用通用工程常识（reproduce/test/read/fix/verify）——这些在 G 中
   - **编排级**：zero-shot LLM **不会**自发用 rick 的特定编排（精确阶段序列 + 声明 + subagent + 上限 + 门禁）——编排不在 G 中（节点 A）
   - 故 human "不会选择用这个方法"指**编排级**，成立；不否认组件级重叠

3. **zero-shot 默认行为的特征**：直接答 + 单轮 + 线性 + 无显式 phase 声明 + 无迭代上限 + 无门禁——与 rick 强制 Step 0-5 形成对照（节点 C）

## 疑问点

- 无疑问点阻断结论；但**置信度未达高（0.8）**：dispatch 信源设计 runtime+docs（无 code），权重上限 0.5；paired rick-injected demo 不可运行（需 rick 二进制设置）。真理由 runtime demo + 3 篇 arxiv 主源已充分，进入 R7 上报。

## R7 上报

- **节点 B 进入 R7**：置信度 0.55（中），无法达高。理由：(a) LLM 决策/选择源码不在 rick 仓库（代码 0.4 不可计入）；(b) 无法跑 paired rick-injected demo（需 rick 二进制完整设置：job 目录/OKR/SPEC/task.md/debug.md/skills）；(c) dispatch 信源设计 runtime+docs，上限 0.5。真理由 zero-shot runtime demo（线性单遍、不选 rick 编排）+ Plan-and-Solve/Self-Refine/Reflexion 三源（zero-shot 默认单轮不迭代）双重确立，建议 human 接受"zero-shot 不选 rick 方法（编排级）"结论。
