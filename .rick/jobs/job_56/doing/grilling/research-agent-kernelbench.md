# research-agent-kernelbench — Agent 级 GPU kernel 优化评测榜单调研

> 2026-11-22 · 调研触发：job_56（super 进化系统需要与同类 agent 系统对比——非 one-shot LLM 模型榜）
> 方法：web_search 3 轮（Exa）+ arXiv/GitHub/官方榜直抓 · 信源等级见各条

---

## 核心结论

1. **不存在专门的「agent 级 GPU kernel 优化」评测榜单**——这是一个**空白**。所有现有 kernel benchmark（KernelBench/KernelBench-Verified/KernelBenchX/MultiKernelBench 等）都是 one-shot LLM 生成评测。
2. **最接近的是 KernelGYM 的 RL 训练评测协议**——但它是训练环境基建（reward 函数+GPU 调度），不是「agent 榜单」。
3. **通用 agent benchmark（SWE-bench/Terminal-Bench/AgentBench/OSWorld）不含 kernel 优化子集**——kernel 编程是垂直域，通用 agent 榜不覆盖。
4. **AlphaEvolve/OpenEvolve/KernelFoundry 等进化系统用「KernelBench 子集 + 自定义报告」做内部对比**——没有统一的进化系统评测协议。
5. **如果我们定义一个「agent-level kernel optimization evaluation protocol」，这是原创贡献**——填补空白。

---

## Q1: agent/agentic kernel 优化评测

### 1.1 没有专门的 agent kernel 优化 benchmark

**结论：无强证据（检索 3 轮 20+ 检索词均未发现）**。

现有 kernel benchmark 的共同特征：
- KernelBench (arXiv:2502.10517) — one-shot LLM 生成，无工具使用
- KernelBench-Verified (arXiv:2607.16241) — 同上 + 反 reward hacking
- KernelBenchX (arXiv:2605.04956) — 同上 + outlier 模式
- MultiKernelBench (arXiv:2507.17773) — 多平台（CUDA/Triton/HIP），仍 one-shot
- KernelGenBench (GitHub: flagos-ai) — 210 算子多芯片，one-shot

**没有任何一个允许 agent 使用工具（跑代码/读 profile/搜索文档）或多轮迭代。**

### 1.2 KernelGYM + Dr.Kernel（最接近的）

- **KernelGYM** (arXiv:2602.05885, ICML 2026, HKUST + TikTok)：分布式 GPU 评测/RL 训练环境——支持 KernelBench 作为 backend，CUDA event 计时 + decoy kernel detection。但它是**训练基建**（Redis 队列/VERL 集成/reward hacking detection），不是评测榜单。
- **Dr.Kernel-14B**（同论文）：用 KernelGYM 训练的 RL agent——在标准 KernelBench 上报告（1.2× fast 指标与 Claude 4.5/GPT-5 同档）。**报告口径仍是 KernelBench（模型榜单），不是 agent 榜单**——Dr.Kernel 的 RL 多步推理在评测时被**折叠为单次生成**（推理时的多步思考不算 agent 行为）。
- **关键区别**：Dr.Kernel 的 RL 推理是「多次前向传播后输出最终 kernel」——评测时仍算 one-shot（LLM 内部多步≠外部 agent 循环）。**没有工具使用、没有代码执行、没有迭代修正。**

### 1.3 AI Coding Agent 在 kernel 优化上的评测

**无强证据**——SWE-agent/Devin/Cursor 等通用 coding agent 的评测（SWE-bench 等）不含 kernel 优化子集。没有任何公开的「用 SWE-agent 优化 CUDA kernel」的评测结果。

---

## Q2: 通用 agent benchmark 中的 kernel 子集

| Benchmark | kernel/CUDA 子集？ | 来源 |
|---|---|---|
| SWE-bench / SWE-bench-Verified | ❌ 无（Python repo 修 bug/feature） | swebench.com |
| Terminal-Bench | ❌ 无（terminal 任务：文件/Git/DevOps） | tbench.ai |
| AgentBench | ❌ 无（8 环境：OS/DB/Web/Game 等） | arXiv:2308.03688 |
| OSWorld | ❌ 无（桌面 GUI 操作） | arXiv:2404.07972 |
| GAIA | ❌ 无（通用助理问答） | arXiv:2311.12983 |
| MLE-bench | ❌ 无（Kaggle ML 竞赛） | arXiv:2410.07095 |
| **结论** | **零个通用 agent benchmark 含 kernel 优化** | — |

---

## Q3: 进化搜索/QD 系统的评测

### 3.1 各系统各自的评测方式（无统一协议）

| 系统 | 评测方式 | 来源 |
|---|---|---|
| **FunSearch** (DeepMind) | 自定义数学问题（cap set/bin packing）——每个问题单独报告，无标准化 benchmark | Nature 2024 |
| **AlphaEvolve** (DeepMind) | 内部 Google 问题（矩阵乘/scheduling/data center）——**不公开题库，不可复现** | arXiv:2506.13131 |
| **OpenEvolve** (开源) | 复现 AlphaEvolve 的 circle packing + 简单算法——**无 GPU kernel 评测** | GitHub: codelion/openevolve |
| **KernelFoundry** (Basis) | **用 KernelBench 做评测**——但用进化系统生成（map + improve 循环）| arXiv:2610.16281 |
| **CompoundingKernel** | 用自建的「kernel optimization benchmark」（内部题库）| arXiv:2611.19230 |

### 3.2 KernelFoundry 的评测——最接近我们场景

- **KernelFoundry** (arXiv:2610.16281, Basis Research)：进化搜索系统（map→improve 循环 + LLM mutation），在 **KernelBench L1** 上评测。
- 评测方式：从 KernelBench L1 100 题中选子集，用进化系统迭代生成 kernel，报告最优 speedup——**但报告口径仍是 KernelBench 标准指标（fast_p）**，没有定义「进化系统的专属评测协议」（如迭代次数限制/预算公平/知识迁移度量）。
- **关键局限**：KernelFoundry 的报告没有对比「同预算 one-shot vs 进化」的公平实验——只报进化系统的绝对分数。

### 3.3 CompoundingKernel 的做法

- 用自建内部 benchmark（不公开）——评测结果不可复现/不可比较
- 报告指标：Quant/030 深跌→graft 新高 = 片间跳跃教科书——这是内部轨迹分析，不是标准榜单

---

## Q4: 学术界的 agent kernel 评测框架

**结论：没有专门的「agent-level kernel optimization evaluation」协议论文。**

原因分析（为什么这个空白存在）：
1. **评测成本**：agent 系统每题跑 10-30 分钟（vs one-shot 秒级）——100 题 × 5 基线 = 天级评测
2. **公平性困难**：agent 可以用工具（搜索/执行/profile）——需要控制「工具调用预算」
3. **知识积累维度**：agent 可以跨题学习——需要定义「题间隔离 vs 允许迁移」两种模式
4. **社区尚未成熟**：agentic coding 是 2025 年热点，但 kernel 优化的 agent 化才刚起步

**最接近的学术讨论**（非完整协议）：
- arXiv:2510.07356 (ConCuR)：提出 LLM kernel 生成需要「多步推理 + 反馈」——但只讨论了可行性，没有提出评测协议
- arXiv:2609.22220 (Measuring the Checker)：讨论了验证器的质量对 kernel 评测的影响——但没有涉及 agent 维度

---

## Q5: 我们的最佳对比基线（如果要发论文/报告）

### 5.1 可直接对比的系统

| 对比对象 | 为什么可比 | 怎么比 |
|---|---|---|
| **One-shot LLM (GPT-5.5/我们的 LLM)** | 同底座 LLM、同题库、同验证协议 | 同一道题：one-shot 生成 vs 进化系统——隔离「架构增益」 |
| **Best-of-N 采样** | 控制尝试次数 | 同题：one-shot × N 次取最优 vs 进化系统同预算（token/GPU 时间） |
| **KernelFoundry** | 唯一公开的进化 kernel 系统（用 KernelBench） | 如果我们能复现其 L1 结果——同题同协议 |
| **Dr.Kernel-14B** | RL agent（内部多步推理） | 在 KernelBench 上报了分数——但它的 RL 推理不算外部 agent 循环 |

### 5.2 最有说服力的对比实验设计

```
三臂对照（同底座 LLM + 同题库 + 同验证协议）：

臂 1: One-shot（基线）—— LLM 直接生成 kernel，每题 1 次
臂 2: Best-of-N（采样基线）—— 同题采样 N=5/10/20 次，取最优
臂 3: Super 进化系统（我们）—— builder→coding→debug→gate_check + QD archive + adjust

控制变量：
- 总 token 预算相同（如每题 100K token）
- 总 GPU 时间相同（如每题 5 分钟）
- 验证协议相同（KernelBench-Verified TF32 + 四分布）

报告维度：
- Correctness%（三臂同口径）
- Correct Speedup geomean（三臂同口径）
- 迭代效率曲线（进化系统的 per-iteration 改进 vs Best-of-N 的采样改进）
- 知识迁移增益（进化系统第 N 题的起点是否优于第 1 题——控制题序效应）
```

### 5.3 原创贡献机会

**如果我们定义并开源「Agent Kernel Optimization Benchmark (AKOB)」，这是填补空白：**

核心设计维度（区别于 KernelBench 的 one-shot）：
1. **工具使用预算**：限制 agent 的工具调用次数/token/GPU 时间
2. **多轮迭代**：允许 agent 实验→验证→修正循环（每轮记录 wall time + token cost）
5. **知识积累**：定义「题间隔离模式」（每题从零开始）和「迁移模式」（允许跨题学习）两种评测模式
3. **公平性控制**：总资源预算相同（token + GPU 时间 + 工具调用数）
4. **轨迹评测**：报告 per-iteration 改进曲线（不只报最终分数——进化系统的搜索效率也是评测维度）

---

## R7 上报项

1. KernelFoundry 的详细评测数字（L1 各题 speedup/正确率）不可得——arXiv 论文有摘要但完整数据表未在 GitHub 开源
2. Dr.Kernel-14B 的具体 KernelBench 分数（fast_p 各档）在 KernelGYM 论文中——需要全文精读
3. CompoundingKernel 的内部 benchmark 题目完全不可得（私有）

## Gaps

- **AlphaEvolve 不公开**（Google 内部）——不可复现/不可比较
- **OpenEvolve 没有 kernel 评测**——只有 circle packing/排序等
- **没有统一的进化系统 kernel 评测协议**——各家自定义题库和报告格式，不可横向比较
- **「agent 评测」的定义本身模糊**：RL 内部多步推理（Dr.Kernel）vs 外部 agent 循环（我们的 builder→coding→debug）——学界尚无统一区分标准

---

## 信源索引

| 来源 | 等级 |
|---|---|
| arXiv:2502.10517 (KernelBench) | A |
| arXiv:2607.16241 (KernelBench-Verified) | A |
| arXiv:2602.05885 (KernelGYM + Dr.Kernel, ICML 2026) | A |
| arXiv:2610.16281 (KernelFoundry, Basis) | A |
| arXiv:2611.19230 (CompoundingKernel) | A |
| arXiv:2506.13131 (AlphaEvolve, DeepMind) | A |
| GitHub: codelion/openevolve | B |
| swebench.com / tbench.ai / agentbench.ai | B |
