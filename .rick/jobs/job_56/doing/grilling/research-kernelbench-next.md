# research-kernelbench-next — KernelBench 之后的新一代榜单全景与升级评估

> 2026-11-22 · 调研触发：job_56 第七轮（是否升级替代 KernelBench）
> 方法：web_search 2 轮（Exa）+ arXiv/官方 Leaderboard/GitHub API 直抓核实 · 信源等级见各条
> 消费方：human（升级裁决）+ super env 协议改造

---

## 核心结论（先读）

1. **你记忆中的「Anthropic 的 KernelGym」不存在**——KernelGYM 是 **HKUST**（香港科技大学）的分布式 GPU 评测/RL 训练环境（ICML 2026，非 Anthropic、非题库榜单）；Anthropic 确有一个 kernels task 但**任务不公开**（Claude system card 内部 eval），无法采用。
2. **KernelBench 题库仍是事实标准且活跃维护**（2026-03 仍有 commit：UV 迁移/AMD HIP backend/NCU profiling；官方 v0.1 已自我修复随机化测试问题）——**不需要换题库**。
3. **真正值得升级的是验证协议**：Stanford+Meta 官方续作 **KernelBench-Verified**（原班人马 Azalia Mirhoseini 等，arXiv:2607.16241 + 官方 Leaderboard 2026-05 评测）——修复了 reward hacking（四分布隐藏测试）、baseline 计时（TF32-enabled）、新增内存效率指标。**其官方榜就跑在 H200 上（与我们 H20 同 Hopper 代）**。
4. **推荐动作**：题库不动（KernelBench 250），验证协议升级到 Verified 标准（四分布测试+TF32 baseline+内存指标）——协议清晰可自实现（Verified 代码未独立开源），迁移成本低（改 verify_impl/benchmark_impl 即可），报告格式对齐官方新榜。

---

## 1. 「Anthropic KernelGym」的事实核查（用户记忆纠偏）

| 名字 | 实际是什么 | 发布方 | 与 Anthropic 关系 |
|---|---|---|---|
| **KernelGYM** | 分布式 GPU 评测/训练环境（数千任务并行、Redis 队列、VERL 集成、decoy kernel detection）+ RL 训练方法 Dr. Kernel（arXiv:2602.05885）——**环境基建，非题库榜单**；支持 KernelBench 作为 backend（CUDA/Triton） | **HKUST**（Junxian He 组）+ TikTok/CUHK(SZ)/NTU | 无关（ICML 2026） |
| **Kernels task (Anthropic)** | Claude system card 内部评测（2025-05，agents 优化 compute kernels，按 speedup 评分）——**任务不公开** | Anthropic | 是 Anthropic 但**无法使用**（私有） |
| Claude Mythos/AMD 新闻 | Tom's Hardware 报道 Anthropic Claude「三天写 AMD GPU kernel」——新闻非榜单 | Anthropic | 可能的混淆源之一 |

记忆混淆的可能来源：KernelGYM（HKUST）+ Anthropic kernels task（私有）+ KernelBench 作者 Azalia Mirhoseini（Stanford，曾与 Anthropic 合作 AlphaCode 类工作）三者混叠。

## 2. KernelBench 原版现状（维护活跃度实测，B 级——GitHub API 直查）

- **v0.1（2025-07，官方自我升级）**：①固定问题规模+改进数值（防 launch 开销占比过大与 trivial kernel）②支持正态分布之外的**随机化测试**③结果有效性分析指南。HF dataset 同步更新到 v0.1。（官方博客 scalingintelligence.stanford.edu/blogs/kernelbenchv01 + Benchmark Almanac，A 级）
- **维护活跃**：最近 commits——2026-03-05 UV 迁移、2026-02-27 AMD HIP backend、2026-01-20 独立 reference timing、2026-01-08 NCU profiling API；Fall 2025 维护 roadmap（issue #74）更新至 2026-04-15（Stanford 本科生团队核心维护）。（GitHub API 实测，A 级）
- **仍在被厂商刷榜**：MiniMax-M3 模型卡引用（28.8%）、xAI 报内部衍生版 KernelBenchInternal v1.1、Inspect Evals（UK BEIS）收录为标准 eval。（Benchmark Almanac + GitHub，B 级）

## 3. 新榜单全景（2025-02 之后，按与我们的相关度排序）

| 榜单 | 发布方/时间 | 核心改进 | 任务规模 | 评分协议 | 开源 | 对我们的价值 |
|---|---|---|---|---|---|---|
| **KernelBench-Verified**（arXiv:2607.16241） | **Stanford+Meta 官方**（Azalia Mirhoseini 原班）2026-05 | ①四分布**隐藏测试套件**对抗硬编码 reward hacking ②**TF32-enabled baseline**（原版未开 TF32，对现代 GPU 不真实）③**内存效率指标**（速度-内存权衡；28% 的模型 kernel 反而增加峰值显存） | KernelBench 题库 | Correct Speedup↑=geomean（正确题上的 baseline/kernel）；Correctness%（best@k）；Fast@1；Correct Mem Eff↑；FP32/BF16 双榜；best-of-5 | **代码未独立开源**（404 实测）；协议论文+官方榜可复现 | **最高**（官方续作、H200 榜与我们 H20 同代、修复的正是我们已部署协议的弱点） |
| KernelGYM + Dr. Kernel（arXiv:2602.05885，ICML 2026） | HKUST+TikTok | 分布式 GPU 环境（数千任务并行/Redis/VERL 集成）+ RL 训练配方（reward hacking detection/decoy kernel detection/long-horizon rollouts） | 复用 KernelBench 等题库 | CUDA event 计时+自定义 tolerance+decoy 检测 | ✅ GitHub | 中（环境基建我们已有；decoy 检测与 reward hacking detection 思路可吸收；Dr.Kernel-14B 与 Claude 4.5 Sonnet/GPT-5 同档的数据可作参考基线） |
| KernelBenchX（arXiv:2605.04956） | 学术 | outlier 模式（0.1% 概率 50× 离群值）——**我们已移植** | KernelBench 超集 | fast_p+ | ? | 已吸收 |
| AgentKernelArena（arXiv:2605.16819） | 学术 | unseen-configuration 泛化测试 | - | - | ? | 低（泛化维度可后补） |
| CANN Bench（arXiv:2607.20518） | 学术 | hidden split（public/hidden 同构） | - | - | ? | 低（思想已被 Verified 的隐藏测试覆盖） |
| gpuemu（arXiv:2606.27396） | 学术 | op-schema-aware seeded fuzzer 的 shape 变异输入 | - | - | ? | 低 |
| MultiKernelBench（arXiv:2507.17773） | 学术 | 多平台（CUDA/Triton/HIP 多芯片）、细粒度算子分类、均衡覆盖 | 多来源 | - | ✅ | 低（我们单 NVIDIA 平台） |
| KernelGenBench | flagos-ai（国产） | 210 算子（ATen 110/vLLM 50/cuBLAS 50）、多芯片（含昇腾/MUSA/海光/摩尔线程） | 210 | 双模式 | ✅ GitHub | 低（NVIDIA 之外的芯片域） |
| AscendGYM | slai-aitp | 昇腾真实训练/推理 workload（951 cases）、Speedup-Token Score（性能-成本联合） | 951 | STS | ✅ | 低（昇腾域） |
| EdgeKernelBench（arXiv:2610.24648） | 学术 | 边缘后端（异构硬件/碎片化栈/紧预算） | - | 统一 IR | ? | 低 |
| HPCAgent-Bench（arXiv:2610.20673） | 学术 | 689 kernels 三轨（loop 优化/HPC 提取/ML 模型）、统一 NumPy 规格多框架派生、agent/skill 成本核算 | 689 | 分轨 | ? | 低（HPC 域，非 LLM kernel 主流） |
| **验证器研究线**（非榜单） | 学术 2026 | Contract-Grade Verifier（2608.12700：few random inputs+loose tolerance 会静默放过 NaN/非确定/形状脆弱/fp16 累积错误）；KernelAudit（2611.15506：审计验证器契约）；Measuring the Checker（2609.22220：mutation analysis 评 oracle 质量） | - | - | 论文 | 中（Verified 协议的学术印证+补充检查项清单） |

## 4. KernelBench-Verified 官方榜数据（抓取自官方 Leaderboard，A 级）

- 协议：**Best-of-5**、Baseline=PyTorch reference **on H200**、2026-05 评测、FP32/BF16 双 tab、Level 1/2/3 分层
- 指标：Correct Speedup↑ = geomean(baseline/kernel) over correct problems only｜Correctness%↑ = best@k｜Fast@1↑ = best correct speedup>1× 的比例｜Correct Mem Eff↑ = geomean(baseline/kernel memory)｜Mem Efficient%↑
- **FP32 榜关键事实：没有任何模型的 correct speedup geomean 超过 1.0**（最强 GPT-5.5：L1 0.90 / L2 0.85 / L3 0.92；Claude Opus 4.7 L1 仅 0.46）——TF32-enabled baseline 下「打过 PyTorch」尚未有模型做到；BF16 榜 GPT-5.5 L2 1.63（bf16 有真实融合空间）
- 论文数字：GPT-5.5 在 verified 协议下 0.88x geomean（标准协议 1.43x——**reward hacking+弱 baseline 虚增了 63%**）；最佳模型 28% 的 kernel 增加峰值显存

## 5. 升级建议（针对 super 场景）

**判定：不换题库；升级验证协议到 KernelBench-Verified 标准。**

| 决策点 | 建议 | 理由 |
|---|---|---|
| 题库 | **保持 KernelBench 250**（v0.1 dataset） | 事实标准、活跃维护、厂商仍刷榜、我们基建已对齐 |
| 验证协议 | **升级到 Verified**：①四分布隐藏测试（正态+现有 outlier 之外补均匀/重尾/边界值分布，seed 隐藏）②**TF32-enabled baseline**（reference 计时开 `torch.backends.cuda.matmul.allow_tf32=True`——H20 上与官方 H200 榜对齐）③**内存效率指标**（`torch.cuda.max_memory_allocated` 比值） | 修复的正是我们协议的已知弱点（我们已遇 passthrough/全零，四分布是对硬编码的更强对抗）；Verified 是官方新标准，报告直接可比 |
| 报告格式 | 对齐 Verified Leaderboard：Correct Speedup geomean + Correctness% + Fast@1 + Correct Mem Eff（比 fast_p 曲线信息量大，官方现役格式）+ 标注 best-of-evolution 口径 | 与官方榜同格式可横向参照（GPT-5.5 L2 FP32 0.85 = 我们的参照锚点） |
| 多维目标 | speedup 0.7 + mem_eff 0.3 加权融合（v2 融合分原生支持）——或纯 speedup+mem_eff 作为第二指标并列报告 | 官方把内存效率作为独立指标；融合权重是设计决策（human 裁决） |
| KernelGYM | 不引入（环境基建我们已有；进化框架≠RL 训练）；吸收其 decoy kernel detection 思路（已部分覆盖：passthrough 静态检测） | 价值在 RL 训练配方，非评测协议 |
| 参照基线 | Dr.Kernel-14B（open-weights RL 模型，与 Claude 4.5/GPT-5 同档于 1.2× fast 指标）与 GPT-5.5 官方榜数字可作我们进化系统的外部锚点 | 报告可比性 |
| 迁移成本 | 低：改 env/gates/verify_impl.py（四分布）+ benchmark_impl.py（TF32 baseline+max_memory 计量）+ gate_check.py（mem_eff 指标）+ report.py（新格式）——**题库/远程执行链/作弊样本全不动** | 协议级改动，非架构级 |

## R7 上报项

1. KernelBench-Verified **代码未独立开源**（GitHub 404 实测）——四分布测试的具体分布参数/隐藏 seed 管理需按论文自行实现（协议描述清晰，实现无障碍，但与官方的精确对齐有细微自由度）
2. TF32 baseline 的精确设置（`allow_tf32` 全局开关 vs `torch.set_float32_matmul_precision('high')`）官方实现细节未开源——需选一种并在报告中标注（建议后者，语义更明确）
3. Verified 榜为 best-of-5 single-turn；我们 best-of-evolution（archive 取最优）——口径差异必须在报告标注（不可直接比大小，但可作上界参照）
4. Dr.Kernel/KernelGYM 的 decoy kernel detection 具体算法未细读（README 截断）——若采纳需补一次源码级调研

## Gaps

- Verified 论文全文细节（四分布的精确定义、内存指标的归一化公式）仅从摘要+Leaderboard 页面还原——落地前建议全文精读 arXiv:2607.16241（下一步动作）
- xAI KernelBenchInternal v1.1 等厂商私有衍生版细节不可得（私有）
- Anthropic kernels task 的协议细节不可得（私有）
