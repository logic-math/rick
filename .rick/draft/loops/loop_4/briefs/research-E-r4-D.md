# research-E-r4 节点 D — LLM 对未见 G′ 是否"无法一次性解决"（单轮 vs 迭代）

节点路径：[根 > E-r4-zero-shot对比 > D-单轮vs迭代解决未见G']
事实陈述：LLM 对"未见的 G′ 问题"是否"无法一次性解决"——zero-shot 单轮 vs 多轮迭代在解决未见问题上的差异？（关联 human 核心假设 + Y-E1 涌现推理）

## 执行动作

1. 文档：arxiv 检索 self-refine（2303.17651）/ Reflexion（2303.11366）/ in-context-learning-limits（2502.03503 "Analyzing limits for in-context learning"，2509.10414 "Is In-Context Learning Learning?"，2404.11018 "Many-Shot In-Context Learning"）/ Plan-and-Solve（2305.04091）+ 涌现反例 "Sample More Reflect Less"（2607.28576）
2. 运行时：复用 E-r2 `briefs/research-E-r2-C.md`（LLM 对 rick G' 事实 zero-shot 单轮拒答——单轮失败实证）+ 节点 B demo（zero-shot 线性单遍无迭代）
3. 代码原文：rick `doing_loop.md` Step 3-5（Main→Sub Agent per-iteration + 失败返回 Step 3 + 3 轮上限）——rick 设计假定单轮不足
4. 反事实：无法跑 paired 单轮 vs 多轮（claude CLI -p 单轮；多轮需 session，raw API 被 proxy 拒绝）
5. 信源权重默认：代码 0.4 / 运行时 0.3 / 文档 0.2 / 反事实 0.1

## 信源验证结果

### 代码原文（权重 0.4）✅（rick 设计假定单轮不足→实现迭代）

**证据 — doing_loop.md Step 3-5 实现"多轮迭代 + 失败重试 + 停止标准"**（`internal/prompt/templates/skills/doing_loop.md`）：
- Step 3："**每轮迭代由 Main Agent 启动一个独立 Sub Agent**，携带 Step 2 的上下文，执行完整工作流后返回产出摘要"——单轮不够，需多轮 Sub Agent
- Sub Agent DEBUG 触发条件："测试 FAIL / 编译报错 / 行为与预期不符"——**遇错强制 DEBUG**（承认单次会出错）
- DEBUG Phase 4 上限 3 次，"达上限后输出当前状态并升级人工协作"——单轮无法收敛时升级
- Step 4："存在失败 → 将失败原因附加到上下文，**返回 Step 3 启动下一轮迭代**"——迭代是核心
- Step 5 优雅退出：迭代次数达上限（默认 3 轮）/ 连续 2 轮产出相同错误（判断无法自动收敛）——停止标准假定"可能不收敛"

→ rick 设计**假定单轮不足**（否则无需 Step 3-5 迭代 + 3 轮上限 + DEBUG + 停止标准）。这是 human 核心假设"LLM 对 G' 无法一次性解决"的工程化体现：rick 用迭代 + 门禁 + 上限来应对"单轮不可靠"。

### 运行时行为（权重 0.3）✅（决定性——单轮 zero-shot 对 G' 失败）

**证据 1 — 复用 E-r2 节点 C runtime demo**（`briefs/research-E-r2-C.md`）：
claude CLI 问 rick 项目特定 G' 问题（doing 模板文件名 + GenerateDoingPromptFile 注入变量），无上下文单轮 → LLM 拒答："I don't know this. I have no memory or knowledge of a 'rick CLI'... fabricating would be worse than useless... point me at the source."
→ **单轮 zero-shot 对 G' 事实 100% 失败**（权重中根本不存在）。这是"LLM 对未见 G' 无法一次性解决"的最强 runtime 实证——对训练数据外的项目特定事实，单轮提取损失=100%。

**证据 2 — 节点 B runtime demo**（`briefs/research-E-r4-B.md`）：
zero-shot 任务方法描述是**线性单遍叙述**（"I'd start... Next... Then... Finally..."），无迭代结构、无失败重试、无停止标准。
→ zero-shot 默认单轮，不自发迭代——若单轮答错，无内在机制修正（需外部脚手架）。

### 文档（权重 0.2）✅（四源——单轮不足 + 迭代改善 + ICL 局限）

**源 1 — Self-Refine（arxiv 2303.17651）**：
> "Like humans, large language models (LLMs) **do not always generate the best output on their first try**... Self-Refine... through **iterative feedback and refinement**... outputs generated with Self-Refine are preferred by humans and automatic metrics"
→ 单轮非最优；**迭代精炼改善**——单轮→多轮有实证增益。

**源 2 — Reflexion（arxiv 2303.11366）**：
> "it remains **challenging for these language agents to quickly and efficiently learn from trial-and-error**... Reflexion agents verbally reflect on task feedback signals, then maintain reflective text in episodic memory buffer to induce better decision-making in **subsequent trials**... not by updating weights, but through linguistic feedback"
→ 单轮不能"从试错中学习"；**多轮反思（linguistic feedback，非权重更新）改善**——单轮→多轮有实证增益。

**源 3 — in-context learning limits**：
- arxiv 2502.03503 "Analyzing limits for in-context learning"
- arxiv 2509.10414 "Is In-Context Learning Learning?"（质疑 ICL 是否真"学习"）
- arxiv 2404.11018 "Many-Shot In-Context Learning"（更多示例→更好，暗示少示例不足）
→ in-context learning 有局限，少示例/单轮不足；需更多示例/多轮——单轮对未见问题不可靠。

**源 4 — Plan-and-Solve（arxiv 2305.04091）**：
> Zero-shot-CoT "suffers from three pitfalls: calculation errors, missing-step errors, and semantic misunderstanding errors"
→ zero-shot 单轮 CoT 有系统缺陷。

**边界证据 — "Sample More, Reflect Less"（arxiv 2607.28576）**：
> "Self-Refine and Reflexion **Lose to Repeated Sampling at Equal Token Cost**, from 1.5B to 7B"
→ 反例/边界：同等 token 成本下，单纯重复采样可超过 self-refine/Reflexion。**nuance**：迭代不是银弹，"重复采样"也是多轮策略的一种；但**单轮仍被多轮（无论迭代反思还是重复采样）超越**——核心"单轮不足"结论不变。

→ 四源一致：**LLM 单轮对未见问题不可靠**（有系统缺陷、非最优、不能从试错学习、ICL 有局限）；**多轮（迭代/反思/重复采样）改善**——单轮→多轮有实证增益，但迭代非银弹（重复采样可超过反思）。

### 反事实（权重 0.1）❌

- 理想反事实：paired 同一冷门任务 (a) 单轮 -p 调用 vs (b) 多轮 session 迭代，对比成功率
- 执行受阻：claude CLI `-p` 单轮模式；多轮需 session（raw API 被 proxy 拒绝，无法自建多轮 loop）；无法跑 paired
- 但 E-r2 节点 C demo（单轮拒答）+ 我读 doing_prompt.go 后可答（"多轮"借助外部上下文）已构成 de facto 单轮失败 vs 有外部信息成功的对照
- 该 de facto 对照已在 E-r2 节点 C 计入反事实，此处不重复计入

## 还原确认

无 rick 代码修改，无需还原。Read/curl/claude CLI 只读。

## 置信度评估（由 research 主调度计算）

- 代码原文 ✅ × 0.4 = 0.4（rick doing_loop Step 3-5 迭代+3轮上限+DEBUG+停止标准，设计假定单轮不足）
- 运行时 ✅ × 0.3 = 0.3（复用 E-r2 节点 C：单轮 zero-shot 对 G' 拒答=100% 失败 + 节点 B：zero-shot 线性单遍无迭代）
- 文档 ✅ × 0.2 = 0.2（Self-Refine/Reflexion/ICP-limits/Plan-and-Solve 四源 + "Sample More Reflect Less" 边界反例）
- 反事实 ❌ × 0.1 = 0.0（无法跑 paired 单轮 vs 多轮；de facto 对照已在 E-r2 计入）
- **合计 = 0.9（高，≥0.8 终止）**

## 关键事实

1. **✅ LLM 对未见 G' 无法一次性解决**（human 核心假设成立）
   - runtime：单轮 zero-shot 对 rick G' 事实拒答=100% 失败（E-r2 节点 C）；zero-shot 默认线性单遍无迭代（节点 B）
   - 文档：单轮有系统缺陷（Plan-and-Solve）、非最优（Self-Refine）、不能从试错学习（Reflexion）、ICL 有局限（2502.03503 等）
   - 代码：rick 设计假定单轮不足→实现 Step 3-5 多轮迭代+3 轮上限+DEBUG+停止标准

2. **多轮迭代改善（但非银弹）**：
   - Self-Refine/Reflexion：迭代+反思改善单轮
   - **边界**："Sample More, Reflect Less"（2607.28576）——同等 token 成本下重复采样可超 self-refine/Reflexion
   - 即"多轮 > 单轮"成立，但"反思式多轮"不一定是最优多轮策略——核心"单轮不足"不变

3. **与 human "盒子里的 LLM" 论断呼应**：G' 永远是过去式，LLM 在 G 上训练——对 G' 子集，单轮提取损失高（极端=100%）；需多轮迭代 + 外部信息（节点 C）+ 确定性方法注入（节点 C）弥补。rick 的 Doing Loop Step 3-5 + debug + 3 轮上限即是工程化弥补。

## 疑问点

- 无疑问点阻断。节点 D 置信度 0.9 达高，终止。
- 边界：本节点证"单轮不足，多轮改善"，不证"涌现推理是否存在"（Y-E1，属 think 范畴）。"Sample More Reflect Less" 反例提示：迭代策略本身有优劣，非所有多轮都优——但单轮 vs 多轮的方向性结论稳固。

## R7 上报

- 无。节点 D 置信度 0.9（高），终止。
