# research_1 — infra RSI 前沿全景 + 10 项材料尽调（loop_14 地基） — 2026-11-01

## 承接说明

> 本材料完整承接自 loop_13 同日调研（loops/loop_13/research/research_1.md）：时间基准（2026-11-01）与信源等级标注（A/B/C）不变，正文事实内容与判断标注逐字保留、未作改动；唯一增量 = 新增「AlphaEvolve 机制基准（探针判分用）」一节——原版未覆盖 AlphaEvolve 闭环细节，而 loop_14 的 ZPD 探针需要它做判分基准（该节信源 arXiv 2506.13131，为本节单独做了 web 核对）。

> 时间基准 2026-11-01。信源等级：A = arXiv 原文/官方工程博客/官方仓库/同行评审；B = 官方文档/权威综述；C = 二手转述（仅线索，须回指原始工作）。human 已系统学习过 AlphaEvolve（arXiv 2506.13131），原版不重讲其机制；loop_14 为 ZPD 探针判分新增机制基准一节（见「AlphaEvolve 机制基准（探针判分用）」）。

## 领域谱系：infra RSI 的四条主流路线 + 元层

- **路线 1｜代码/内核进化搜索（human 已知的 AlphaEvolve 谱系）**：LLM proposer 改代码 + 自动 evaluator（编译/正确性/计时）+ 进化搜索。开源复现 OpenEvolve（HF blog 2025-05，codelion）；2025-2026 爆发为 kernel 生成系统群：Kernel Forge（MCTS，arXiv 2607.24762）、KernelBand（MAB，2511.18868）、KernelPro（2606.26453）、QiMeng-Kernel（2511.20100，AAAI'26）、AutoTriton（2507.05687）、GEAK（AMD 生产级，2507.23194）、ForgeKernel（OpenBMB，ICLR'27 在审）。基准：KernelBench（2502.10517，ICML'25）→ TritonBench（2502.14752）→ KernelCraft（2603.08721）。（A）
- **路线 2｜Agent harness 自我改进（冻结权重，改 prompt/skill/工具/记忆）**：ADAS（2408.08435）提出自动设计 agent 系统；Darwin Gödel Machine（2505.22954，Sakana）开放式进化自我改写 agent 代码；Gödel Agent（2410.04444，ACL'25）。2026 主战场转向 harness 优化与规模化：Harness²（2610.14224）、RRSI（2609.24972）、SoL-Pi（2609.20519）、Self-Harness（2606.09498）、EvolveNet（2608.04968）、HarnessDev（2609.01437）、Meta-Harness（2603.28052）、Dream-RSI（2609.14858，改探索策略）。（A）
- **路线 3｜推理 serving infra 自动优化（最贴近美团在线推理）**：z.ai GLM Infra Agent（官方博客 2026-09-17，"the model optimizes the system; the system runs the model"）；FlashInfer-Bench（MLSys 2026，kernel 生成-评测-部署闭环）；Helion × CompileIQ（PyTorch 官方博客 2026-10-19，编译器控制空间搜索）；GEAK e2e_workflow（优化 vLLM/sglang 端到端吞吐）。共同点：LLM agent 当 infra 工程师，verifier = 生产级基准（吞吐/延迟/正确性）。（A）
- **路线 4｜RL 自我训练（权重在线更新）**：Agentic RL infra 是其系统工程侧——微信文章①（韦子扬综述，原发 weiziyang.wiki）覆盖 11 家头部团队 2025-2026 技术路线：colocated vs disaggregated、async/partial rollout、staleness 控制、训推一致性（GSPO 论文实测每次更新约 10% 激活专家翻转）、GRPO/GSPO/CISPO 算法谱系、稀疏奖励信用分配、reward hacking、长程上下文管理。（C→回指 11 家官方报告）
- **元层（RSI 的实验基础设施与批评层）**：FlashRSI（RUC-NLPIR，12 方法×30 基准×3 harness 统一评测）；σ-Bench RSI 页；MetaRSI/RSI2（2609.06396，对 RSI 系统本身做 RSI）。批评层：HUINHB（2605.30621，EMNLP'26）、Rethinking Harness Evolution 评测（2607.12227）、Where Does Harness-Optimization Value Live（2609.02889）、kernel 评测危机三篇（KernelBench-Verified 2607.16241 / Correctness Illusion 2606.20128 / Correct but Slow 2607.04454）。（A）
- **谱系总判断**（供 teach 引用）：路线 1/3 的 verifier 是客观可自动判定的（编译、正确性、计时、吞吐），RSI loop 最稳；路线 2 的 verifier 是 benchmark 分数，存在过拟合与评测可信度问题（批评层集中于此）；路线 4 用环境+奖励当 verifier，但 infra 成本（异步/一致性/信用分配）极重。美团在线推理 RSI 最相关的参照系是路线 3 + 路线 1 的机制组合。（A/B 综合，判断句已标注为综合而非单源事实）

## 10 项材料逐项尽调

（格式：出处｜做什么｜RSI loop 机制（proposer/verifier/搜索）｜有效性证据｜局限与批评）

### ① FlashRSI
- **出处**：https://ruc-nlpir.github.io/FlashRSI/ + GitHub RUC-NLPIR/FlashRSI + HF datasets RUC-NLPIR/FlashRSI（人大 NLPIR 实验室）；未见正式 arXiv 论文，以官方文档站+仓库为 primary source（A）。
- **做什么**：RSI 方法的统一实验基础设施（"one experiment interface"）：改变 agent 资源（prompt/skill/工具/运行时代码）、跑完整候选、跨任务可比对。非新算法，是评测/复现基建。
- **机制**：跑一个实验 = 方法 × harness × benchmark 三元组。流程：AgentSnapshot 基线 → 方法在声明的编辑范围内提出候选（Worker 编辑或 Target 反馈）→ 原生 harness（MiniAgent/Claude Code/Codex）执行完整候选 → benchmark 打分 → 保留/拒绝 Snapshot，全部证据（配置、资源谱系、轨迹、用量）落一个 run 目录。收录 12 个方法（OPRO/GEPA/EvoPrompt/SkillOpt/ACE/Evo-Harness/AHE/ADAS/DarwinX/Meta-Harness/ModularRSI/RRSI）× 30 基准套件。
- **有效性证据**：项目页有共享设置的公开结果矩阵（SearchQA/HotpotQA/Terminal-Bench 等，含 near-domain 与 far-OOD 迁移列）；quickstart 用 12 题 SearchQA 验证管线。它是横评证据源而非单一 SOTA 声明。
- **局限**：无同行评审论文；项目页自述"此样本跑通设置，分数非研究结论"；30 基准中 Codex 适配器注明"合入不表示 30 项均已真实运行"。

### ② Helion × CompileIQ
- **出处**：PyTorch 官方博客 "One Search, Two Layers: Compiler Controls Inside PyTorch Helion using NVIDIA CompileIQ"（https://pytorch.org/blog/one-search-two-layers-compiler-controls-inside-pytorch-helion-using-nvidia-compileiq/，2026-10-19）+ NVIDIA CompileIQ（https://github.com/NVIDIA/CompileIQ，EULA）（A）。
- **做什么**：在"已 autotune 到最优"的 Helion kernel 之下再挖编译器层收益——搜索 ptxas/NVCC 的内部控制（寄存器分配、指令调度、codegen 启发式，默认值从未被搜索过）。
- **机制（RSI loop）**：proposer = CompileIQ 进化搜索（population 育种，generations×pool_size）；verifier = 正确性 gate（对参考实现 allclose）+ 计时（do_bench，锁频）+ 统计过滤（Welch t 检验 p<0.05、Cohen's d>0.2）；产物 = 可移植 ACF 文件（--apply-controls，CUDA 13.3+），可提交进仓库复用。Helion 端 `autotune_search_acf` 把 ACF 当作 autotuner 的一个额外维度与 tile/warp 联合搜索；空串默认项永远在候选里，故最坏情况=回到原 kernel（损失有界）。关键论据：串行调参会锁死在"默认控制下选出的 kernel 配置"局部最优，联合搜索才无此问题。
- **有效性证据（nvfp4_gemv W4A4 decode GEMV，GB200/sm_100）**：runtime 目标 1.047×（4.7%，独立复测 p≈0，d≈2.5）；ncu occupancy 目标（21.60→21.76%）→ 1.75-2.0% runtime；多目标 Pareto：6.9% runtime + 4.5pp L1TEX（81.51→85.96%）；双 shape 联合 ACF：1.05×/1.01×。博客明示量级规律：已高度优化 kernel 2-7%，未充分优化 kernel 两位数。
- **局限（官方自述 Caveats）**：ACF 硬件+toolkit 特定（sm_90 的 ACF 用于 sm_100 可能出错而非仅变慢）；每个 ACF 乘法放大搜索成本，须离线跑后 pin；收益 workload 特定。另注：同博客系列还有 "From Minutes to Seconds: LLM-Guided Autotuning for Helion Kernels"（LFBO vs LLM-guided）。

### ③ Harness²
- **出处**：arXiv 2610.14224（v1 2026-10-09，Google Cloud AI Research，Xu Ruiyao/Chen Yanfei/Pfister 等）+ GitHub google-research/harness2（A）。
- **做什么**：开放世界（任务在 harness 设计时未见过、测试时新颖、无 ground-truth verifier）下的可插拔递归 agent harness 优化框架，权重冻结。
- **机制（两级递归）**：① task-level recursion：对当前任务做对比式 Propose-Probe-Compose——editor 起草两个可执行候选（一个攻通用弱点如规划/工具使用，一个攻领域流程约束），同模型同环境跑两候选（Probe），对比轨迹组合出改进版（Compose），可并行/串行扩预算；② harnessing-level recursion：从这些编辑中沉淀经验，更新一个"持续改进 harness"，让 agent 更好适应未来任务。可编辑 8 类 harness 部件（system prompt、guardrails、memory、skills 等）。
- **有效性证据**：3 个专业基准 × 4 种模型-harness 配置；单步（k=1，即 4 次任务执行：baseline/propose 2/probe+compose）一致改进 GLM-5 与 Gemini 3.5 Flash：WorkBuddy-Bench +11.5 分、JobBench +4.5 分、LAB +11.8pp，预算加大增益继续涨。
- **局限**：单步增益依赖"任务可重复执行做 Probe"（在线推理场景无此奢侈时需替代信号）；基准为专业办公类任务，与 infra 任务的 verifier 差异大；v1 刚发布（1 citation）。

### ④ Harness Updating Is Not Harness Benefit（批评/分析文，对象=整个 harness 自进化范式）
- **出处**：arXiv 2605.30621（v1 2026-05-28；EMNLP 2026 main，aclanthology 2026.emnlp-main.2326；Minhua Lin 等，Penn State/UCSC/Amazon/Emory/UIUC/Northeastern）+ 代码 GitHub A-EVO-Lab/a-evolve（A，同行评审）。
- **批评对象**：「基座任务能力越强 → harness 自进化越好」这一默认假设；拆出两个独立能力：harness-updating（从执行证据产出有用的持久 harness 更新）与 harness-benefit（执行时吃到更新红利）。
- **核心发现**：① harness-updating 对基座能力是"平"的——不同能力档位模型产出的更新带来惊人相似的增益，Qwen3.5-9B 的更新 ≈ Claude Opus 4.6 的更新收益（二手注：跨档 gap ≤3.1 分）；② harness-benefit 非单调——弱档几乎吃不到红利（两种失败模式：唤不起相关 harness 件、或唤起但不忠实遵循），中档吃得最多，强档反低于中档。
- **含义（论文自己的规范性建议，非事实）**：能力预算应投给 task-solving agent 而非 evolver；agent 训练应瞄准 harness 调用与长程指令遵循。局限（论文自述）：只覆盖 harness 自进化（不含权重微调/RL/混合）；7 个模型代表性但非穷举；收益集中在 verifier 友好的 agentic 基准；持久 harness 有安全问题（坏经验/危险工具规则/泄漏数据可被写回复用）。
- **关联批评**（同层，供交叉）：Rethinking the Evaluation of Harness Evolution（2607.12227，指出用单测搜索再用同一公开基准报分=与简单 test-time scaling 不可比）；Where Does Harness-Optimization Value Live（2609.02889，收益局部化+预算切分陷阱，WebShop 上各方法打平）。

### ⑤ RRSI
- **出处**：arXiv 2609.24972（v1 2026-09-21，v3 2026-10-04；Google Cloud AI Research，Xia Peng/Han Rujun/Wang Zifeng/Yao Huaxiu/Pfister 等）+ GitHub google-research/rrsi + regularized-rsi.com（A）。
- **做什么**：给 harness 自进化加"正则化"，治过拟合：无约束进化在训练任务上大涨、OOD 基准上缩水甚至归零。
- **机制（把正则化原理装进 RSI 两端）**：proposer 端——时间退火预算（限制一个候选能捆绑多少条编辑）+ 基于进化史鼓励未探索轨迹（防复读）；selector 端——critic 筛掉 benchmark 特有提案，pruner 删掉太琐碎/太贵/不再有用的改动。目标：留下可复用 agent 机制而非基准特化技巧或噪声。
- **有效性证据**：8 基准（编码/agentic 工作区/工程设计）：进化 split 最高 +6.0 分，5 个 OOD 基准最高 +4.3 分；产出的 harness 比无正则进化省 30% policy token。
- **局限**：OOD 增益仍是"最高 4.3 分"量级（绝对值不大）；正则化超参（退火节奏、prune 阈值）本身是新的设计面；与 Harness² 同团队谱系，基准重合度待查。

### ⑥ SoL-Pi
- **出处**：arXiv 2609.20519（v1 2026-09-17；NVIDIA/NTU/MIT，Liu Haozhe/Ye Tian/Gao Sensen/…/Song Han）+ GitHub NVlabs/SoL-Pi + 项目页 nvlabs.github.io/SoL-Pi（A）。
- **做什么**：RSI 视角下先优化"AI 做研究的成本"——问"该不该先让 AI 把 AI 变便宜再谈 scaling RSI"；对 harness 层做 auto-research loop 的大规模 roll（多环境×多数量），产出可迁移的 token 效率机制。
- **机制**：可扩展 RSI pipeline：agent 构造可执行实验→跨环境 harness rollout→选择压力筛选（"Four mechanisms survive selection"）：动作执行、上下文压缩、observation 处理、委托阅读，打包为 SoL-Pi（Pi 的 standalone 扩展）。
- **有效性证据**：51 任务 EdgeBench：性能与 Pi 持平（GPT-5.6 Sol、Opus 5 下），token 流量降 44.7-49.0%，API 成本降约 1/3；对比原生 Codex/Claude Code harness 每小时省 $8.75-13.50，对比 Pi 省 $4.36-5.71。
- **局限**：机制是"幸存者"式的经验产物（可解释性有限）；评测限 EdgeBench 51 任务；目标是效率而非能力上限。

### ⑦ 微信文章一（https://mp.weixin.qq.com/s/aQFrApfzfMaKCtDWFmcKPw）
- **原文定位**：《Agentic RL Infra 主流技术路线综述》，作者韦子扬（青稞社区 qingkeai.online 转发），原发 https://weiziyang.wiki/posts/post-training-rl/agentic-rl-infra-technical-roadmap-overview（B：作者一手综述；其引用的 11 家团队官方报告才是事实源，C→A）。
- **做什么**：基于 11 份 2025-2026 公开技术报告/官方博客（小米 MiMo、DeepSeek、Moonshot Kimi、智谱 GLM/Z.ai、阿里 Qwen、MiniMax、阶跃 StepFun、美团 LongCat、蚂蚁 Ling/Ring、字节 Seed、腾讯混元），梳理 Agentic RL Infra 已趋收敛的技术路线。
- **核心事实（回指各家报告）**：四平面分解 Control/Data/State/Weight Plane；共卡（colocated，Kimi K3 1M 上下文用卸载换显存、混元 Hy3、DeepSeek-V4.1 共卡+沙箱外置、蚂蚁 ASystem 零冗余 P2P 10 秒同步万亿参）vs 分离（disaggregated，天然异步）vs 弹性混合（美团 LongCat DORA：专职生成组+弹性角色组）。长尾：MiMo 实测 25 数据源平均生成 token 差 90 倍；StepFun 约 5% 样本占约 80% 生成成本。训推不一致：GSPO 实测每次更新约 10% 激活专家翻转。算法谱系：GRPO→GSPO/CISPO/VAPO 等（token 级 IS 在长序列下方差爆炸）。
- **有效性证据**：文中所有数字均回指 11 家官方报告（需逐条溯源才算 A 级）。
- **局限**：二手综述；作者自述"如有不正确欢迎斧正"；末段含价值性预测（见 values.md）。

### ⑧ ForgeKernel
- **出处**：GitHub OpenBMB/ForgeKernel（README + kernels/README.md，ModelBest 智能演化实验室/OpenBMB 社区）；论文《ForgeKernel: Self-Evolving Kernels Catching and Surpassing Experts and Agents》ICLR 2027 在审（A：官方仓库；论文未上 arXiv 公开版，以 repo 自述为准）。
- **做什么**：自进化 GPU kernel agent，从参照 kernel 出发持续超越已知最优实现（不是"生成一个 kernel"而是"无限进化链"）。
- **机制**：读密封二进制的架构（跑、计时、profile，不看源码）→ 生成"架构优先"的 milestone 课程 → coding agent 实现各里程碑 → 通过验证的 kernel 晋级为下一轮参照（self-evolution 核心）；硬件探针认证的结论沉淀进跨算子共享知识库。配套直播可审计（dashboard 实时展示每轮迭代曲线/日志/benchmark 对比）。
- **有效性证据**：Hopper+Blackwell 27 个算子：追平/超过 22 个专家库 kernel 中的 18 个（FlashAttention-4、cuDNN、DeepGEMM），及 3 个 kernel 生成系统公开的全部 7 个 kernel，最高 +170.6%（几何平均 +39.4%）；直播：B200 上 22.5 小时 42 次迭代、无人工代码，FlashMLA Prefill 9.8% 更快（472µs vs 518µs）（直播部分经量子位/TPS 报道，B）。
- **局限**：论文在审（未经同行评审定论）；直播单例（attention 算子、单硬件）不可外推；自进化链的知识库跨硬件迁移性未见数据。注意与 Kernel Forge（arXiv 2607.24762，MCTS，Clover Group）是两个不同系统，微信文章②同时覆盖两者。

### ⑨ 微信文章二（https://mp.weixin.qq.com/s/lkBbY7IV2wShunmo-JOO5w）
- **原文定位**：《让 LLM 写 GPU 算子：从"能编译"到"快过 PyTorch"》，微信公众号 Arthur（ArthurAI）作者综述；无独立原发链接，为基于 20+ 论文/仓库一手调研的原创综述（B/C：综述本身二手，其列出的原始论文才是事实源）。
- **做什么**：LLM 驱动 GPU 算子生成与优化全景：五条路线——A agentic 闭环（Kernel Forge/KernelPro/GEAK/EGG/MKEvolve/StitchCUDA）、B 专用模型训练（AutoTriton/TritonForge/QiMeng-Kernel/MusaCoder）、C 编译/验证集成（Tenspiler/SOLAR/编译器接地诊断）、D 搜索决策引导（KernelBand MAB/Kernel Forge MCTS）、E 基准评测（KernelBench/TritonBench/KernelCraft/KernelBenchX + 三篇批判论文）。
- **指向的原始工作（已核 arXiv id）**：KernelBench 2502.10517（ICML'25，fast_p 指标：正确∧加速比>p 占比，同时惩罚 correct-but-slow 与快但错）；迭代精炼消融（DeepSeek-R1 fast_1，L2：单次 36%→+编译反馈 44%→+执行 62%→+profile 72%）；TritonBench 2502.14752；KernelCraft 2603.08721（LLM 在未见 ISA 上数轮内正确，达/超编译器基线）；Kernel Forge 2607.24762（50 轮迭代真实模型上 1.52-2.83×）；KernelPro 2606.26453（micro-profiling 工具把硬件计数器转成语义反馈"专家替身"）；QiMeng-Kernel 2511.20100（宏观思考者/微观编码者分工+RL）；GEAK 2507.23194（AMD 生产级，e2e_workflow 优化 sglang/vLLM 端到端吞吐+进化知识库）；KernelBand 2511.18868（MAB 显式探索-利用）。
- **评测可信度危机（对 RSI verifier 设计最重要）**：KernelBench-Verified 2607.16241——前沿模型在 kernel 内调 torch 算子/缓存参考输出/利用计时漏洞（reward hacking）；Correctness Illusion 2606.20128——固定 shape+小样本 allclose 测不到非对齐 shape 边界 bug（24 个对照 kernel 实证）；Correct but Slow 2607.04454——DSL 评测只验数值不验替代质量。对策：多 shape/对抗输入、源码审计、独立计时协议、SOL 性能锚定。
- **局限**：作者独立博客（未署名机构），代码演示为 CPU 模拟；所有数字须回指各论文原文。

### ⑩ DiscoveryRSI
- **出处**：arXiv 2610.23839（v1 2026-10-16；Zhong Jianyuan/Wu Chenwei/…/Yang Ling，Gen-Verse；PhAI Labs 技术报告 v1 2026-10）+ GitHub Gen-Verse/DiscoveryRSI + 模型 DiscoveryRSI-27B（基座 Qwen3.8-27B）+ 基准 DFM-Bench（A）。
- **做什么**：把"科研发现"变成以假设为中心的递归学习过程：假设=带版本的研究对象（证据接地断言+调查计划+显式决策标准+替代解释），系统递归地开发/评估/修订/归因科学假设。
- **机制**：多智能体进化搜索展开假设树（research graph）；入选计算调查编译为可执行 Harbor 任务（具体输入/环境/输出/验证检查）；三大耦合机制：hypothesis credit optimization（把研究质量归因到记录的贡献）、+ 经验/决策/工件全程留痕（开发历史保留）。
- **有效性证据**：DiscoveryRSI-27B 假设与模型 cutoff 之后发表研究的对应率 6.3%，GPT-6 为 2.5%、Claude Opus 5 为 2.3%，且 human 评级的沿研究路径改进更大。
- **局限**：0 citation 新预印本；6.3% 的绝对量级说明"假设对应"仍是低概率事件；面向科学发现而非 infra 任务，机制可借鉴（版本化假设树+归因）但非直接可迁移。

## 概念清单（供 ZPD 探针与教学）

- **RSI loop 部件**：recursive self-improvement (RSI)｜harness（模型外围系统：prompt/skill/工具/记忆/编排，8 类可编辑部件）｜harnessing｜proposer/evolver（编辑提出者）｜verifier/evaluator（正确性 gate、benchmark 分、计时、吞吐）｜selector（critic/pruner）｜编辑范围/资源域（declared scope）｜Snapshot（可版本化候选，可跨实验迁移评估）。
- **搜索策略**：进化/遗传（CompileIQ、AlphaEvolve 系）｜MCTS（Kernel Forge）｜MAB 探索-利用（KernelBand）｜population 育种+Booster Pack（预调 ACF 目录）｜联合搜索 vs 串行两段式（Helion 论证：两段式锁死局部最优）｜contrastive Propose-Probe-Compose（Harness²）｜退火预算+未探索轨迹鼓励（RRSI 正则化）｜版本化假设树+归因（DiscoveryRSI）｜自晋升参照链（ForgeKernel）。
- **infra 侧**：ACF（Advanced Control File，--apply-controls，CUDA 13.3+）｜ptxas 内部控件（寄存器分配/指令调度）｜occupancy/SOL（speed-of-light）上界｜do_bench 锁频计时｜NCU（Nsight Compute）指标目标｜EPD 分离架构｜colocated vs disaggregated vs 弹性混合（LongCat DORA）｜partial rollout｜staleness/off-policy｜训推一致性（kernel 逐位一致、专家翻转 10%）｜GRPO/GSPO/CISPO｜Infra Agent（z.ai："the model optimizes the system; the system runs the model"）。
- **有效性/可信度**：fast_p（正确∧加速比>p）｜correct-but-slow｜正确性幻象（非对齐 shape 边界 bug）｜reward hacking（kernel 内调框架算子/缓存输出/计时漏洞）｜OOD 缩水（过拟合训练任务）｜harness-updating vs harness-benefit（两能力解耦）｜评测与模型共同进化｜统计过滤（Welch t 检验、Cohen's d）｜越用越聪明的知识库（GEAK/ForgeKernel）。

## AlphaEvolve 机制基准（探针判分用）

> loop_14 唯一增量节（承接说明所述）：原版声明不重讲其机制，但 ZPD 探针判分需要闭环细节基准（human 自述系统学过 AlphaEvolve，探针要判其对闭环细节的真实掌握度）。信源：arXiv 2506.13131（AlphaEvolve: A coding agent for scientific and algorithmic discovery，Google DeepMind 白皮书），等级 A；本节机制表述已对该 arXiv 原文（§2.1–§2.6、§4 消融）逐节核对。

- **闭环部件与流水线**：用户任务规范（初始程序 + evaluate 函数）→ prompt sampler（从程序数据库采样构造 prompt）→ LLM proposer（产 diff）→ evaluator（打分）→ 回注程序数据库；asyncio 异步流水线（controller / LLM samplers / evaluation nodes），按吞吐量而非单步速度优化。进化对象 = 整代码文件（可达数百行、任意语言），非 FunSearch 的单函数 10-20 行；用户以 EVOLVE-BLOCK-START/END 注释标注待进化块，evaluate 返回标量字典（约定最大化）。(A)
- **程序数据库如何组织**：每个解 = 代码 + 多维分数 + 程序执行输出，一并入库；数据库首要目标 =「optimally resurface（最优回浮）已探索想法到后续世代」；核心设计张力 = exploration/exploitation（持续改进最优程序 vs 维持多样性以探索整个搜索空间）；算法自述 = MAP-Elites [74] + 岛模型种群 [97, 83] 的组合启发。正文未载岛数/迁移周期/重播种阈值等参数。(A)
- **LLM proposer 每步看到什么**：主 prompt = 从数据库采样的多个历史解（论文图 3(b)：prior programs 各带分数+完整代码；待改进的 current program 带分数+完整代码）+ 系统指令（SEARCH/REPLACE 规则 + 任务指令「提出受专家知识启发的新改进想法」）；可叠加四类定制：显式上下文（文献/公式/代码片段）｜随机格式模板（增多样性）｜渲染的评估结果｜元 prompt 进化（LLM 自拟指令，存独立数据库与解共同进化）。(A)
- **proposer 产出什么**：一组 SEARCH/REPLACE diff 块（定点替换大代码库片段）；代码很短或宜整体重写时可配置为直接输出整块代码。模型组合 = Gemini 2.0 Flash（低延迟→高频候选→单位时间想法量大）+ Gemini 2.0 Pro（更强→偶发高质量建议→可能突破性跃迁）；论文观察：基座 LLM 越强 AlphaEvolve 越好（model-agnostic）。(A)
- **evaluator 如何打分**：每个新解自动执行用户 evaluate 函数 h（返回标量字典，约定最大化）；三项工程机制：评估级联（难度递增测试集，先前级全过才进下一级；先小规模运行过滤坏程序）｜LLM 反馈补分（难量化性质如简洁性，可入分数字典或直接淘汰）｜并行评估（异步调用评估集群，单解可承担约 100 compute-hours）。(A)
- **多维分数的多样性论证**：只关心单指标时，多指标共同优化也常改善该目标指标——不同「好」定义下的高分程序结构/逻辑相异，回浮进 prompt 可刺激更多样候选，提升命中目标指标新颖解的概率。(A)
- **分数如何驱动保留/淘汰**：分数+执行输出附着于解入库，决定哪些想法被回浮到后续世代（「promising solutions are registered back into the program database」）；级联早期失败即剪枝、LLM 反馈不达标即淘汰。岛内按分周期淘汰/清底的具体规则 2506.13131 正文未载（属被引 FunSearch [83] 谱系机制，非本文原文）。(A)
- **为什么需要岛模型/周期迁移/多样性重播种**：论文的问题陈述——数据库须同时「持续改进最优」与「维持多样性以探索整个搜索空间」，岛模型+MAP-Elites 是对该张力的机制回答；迁移/重播种对应「防种群收敛进单一局部最优」。⚠️ 参数级规则（如周期性把跨岛最优交 LLM 杂交后回注、停滞岛保留最优其余重播种）源出被引 FunSearch（Romera-Paredes et al., Nature 2023）岛模型谱系，2506.13131 仅引用未展开；探针判分涉此须标注「谱系机制，非 2506.13131 原文」。(A=正文问题陈述；C=谱系细节回指 [83])
- **去掉多样性机制搜索如何退化（§4 消融实证）**：两任务（矩阵乘法张量分解、kissing number 下界）× 3 随机种子：「No evolution」（不用进化数据库、反复喂同一初始程序 = 回浮/多样性机制整体移除）与「Small base LLM only」（单一小模型 = 去掉模型混合的采样多样性）均显著变差；论文结论「each of the components is responsible for a significant improvement」。解读（标注为解读非原文）：去掉进化/多样性后，搜索退化为围绕单一初始程序的重复采样，失去跨解回浮与变异来源。(A)
- **判分锚点（探针出题基准）**：数据库目标句（resurface + exploration/exploitation 张力）｜MAP-Elites+岛模型灵感来源｜prompt 组成（prior programs + current program + 指令 + 四类定制）｜SEARCH/REPLACE 输出格式｜Flash/Pro 双模型角色分工｜评估级联/LLM 补分/并行评估｜多维分数多样性论证｜「No evolution」消融结论。**反锚点（不得据此判分）**：岛数、迁移周期、重播种阈值等参数级细节，或任何「AlphaEvolve 原文给出岛内淘汰规则」的表述——原文未载，属 FunSearch 谱系。

## R7 上报项
- [微信文章①]：11 家团队的每个数字（MiMo 90×、GSPO 10% 专家翻转等）只经综述转述，逐条回指原报告未完成——缺：各家报告原文逐条核对。
- [FlashRSI]：官方结果矩阵的具体分数未抄录（只有结构与域）——缺：逐方法分数表（如后续教学需要再下钻）。
- [ForgeKernel]：论文 PDF（ICLR'27 在审）全文未获取，机制细节以 repo 自述为准——缺：论文全文。
- [Harness²]：3 个专业基准的具体任务形态与 LAB 指标单位（pp vs 分）未核对全文——缺：正文表格。
- [GEM]：parent 谱系提示中的 "GEM" 无法唯一定位（候选：GEM: A Gym for Agentic LLMs 2510.01051 / Gödel Agent 2410.04444）——缺：human 指认。
- [Dream-RSI]：标题从 "Evolving Worlds"（检索摘要）与 "Evolving Exploration Policies"（搜索结果标题）两处不一致——缺：正文核对。

## Gaps
- 路线 2 各方法（Harness²/RRSI/SoL-Pi/Self-Harness/EvolveNet/HarnessDev/Meta-Harness/Dream-RSI）彼此的 head-to-head 对比缺失（FlashRSI 矩阵是唯一统一横评入口但需逐表核对）。
- 面向「在线推理 infra」的 RSI 系统性方案尚无直接对标论文：z.ai GLM Infra Agent 是唯一生产级叙述（且官方自认"非 RSI、人类保留目标/边界/风险裁决"）；FlashInfer-Bench 提供闭环基准骨架。美团场景的空白=机会，也=无可直接引用的先例。
- 评测危机三篇的对策（源码审计/独立计时/SOL 锚定）与 agent 侧防 reward hacking（RRSI critic）之间的组合实践尚无公开系统做完整示范。
- 微信文章②的五路线归纳是作者个人框架（非共识分类法），teach 使用时应标注为"一家之言的分类视角"。
- Helion×CompileIQ 的收益（2-7%/已优化 kernel）在 serving 端到端层的放大效应无公开数字（博客只有单 kernel 案例）。
