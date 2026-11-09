# superopt 设计与实现：按 RSI 可自改进的对象划分五条路线

> 内部分享·演讲支撑文档。文档结构：第一部分（superopt 的运行原理与架构）由分享者现场展开，此处留位；第二、三部分合并为按路线组织的五节，每节固定五小件——**路线定义 → 最新进展（论文怎么干）→ 挑战问题 → 论文解法 → superopt 的设计立场与理由**，讲者按小节顺序展开。
>
> 引用约定：论文/系统名附 arXiv 编号；信源等级行内轻标注（A＝论文原文/官方一手，B＝官方文档/权威综述，C＝二手转述）；立场性表述标「作者立场」，与论文事实分开；弱基线与自报数字如实标注口径（z.ai 3.22× 为未调优首跑基线自报；GEAK/ForgeKernel 数字为官方自报、无第三方复测）；superopt 实现细节不在本文展开，每节只取「取舍与理由」层。

## 一、superopt 的设计与实现

（本节由分享者撰写——superopt 运行原理与架构图）

## 二、按 RSI 可自改进的对象划分五条路线

RSI（Recursive Self-Improvement，递归自改进）的落地形态差异极大，差别首先在**被自改进的对象是什么**。按对象一刀切，当前领域工作可划分为五条路线：target（外部代码与 kernel）、model（模型本体）、infra（serving 框架与运行时）、harness（agent 自身扩展件）、meta（评测方法与评测数据）。对象直接决定两件事：**verifier 能不能客观**（改外部代码，编译器＋计时器说了算；改 harness，benchmark 分说了算）与 **loop 的成本结构**（改 harness 近乎零部署成本，改权重＝背上一整套训练 infra）。meta 单列：它不进化「东西」，judge 进化——按对象分类的结构性盲区是它自己。

### 总路线图

```
RSI 可自改进对象五路线（对象 × verifier × 代表工作）

路线一  target｜外部代码与 GPU kernel
    对象     = 第三方工件（kernel / 算子 / 外部程序；agent 本体与模型权重不动）
    verifier = 编译 + 正确性 + 计时（客观、可自动判定）
    代表     = AlphaEvolve (2506.13131)｜ForgeKernel（自晋升参照链）
               Kernel Forge (2607.24762, MCTS)｜KernelBand (2511.18868, MAB)
               KernelPro (2606.26453, micro-profiling)｜QiMeng-Kernel (2511.20100)
    基准链   = KernelBench (2502.10517) → TritonBench (2502.14752) → KernelCraft (2603.08721)

路线二  model｜模型本体（权重 / 结构 / loss / 数据 / 训练框架）
    对象     = 模型权重与训练系统本身（RL 自我训练）
    verifier = 环境 + 奖励（非客观 gate：奖励稀疏、可被 hack）
    代表     = GRPO → GSPO / CISPO 信用分配谱系；Agentic RL Infra 11 家综述
               （colocated 共卡 vs disaggregated 分离 vs 弹性混合——美团 LongCat DORA 在列）

路线三  infra｜serving 框架与运行时
    对象     = 跑模型的系统本身（serving 栈 / 编译器控制 / kernel 资产 / 部署工作流）
               ＝ 优化者（agent）的宿主——对象与优化者的 harness 分属两层
    verifier = 生产级基准（吞吐 / 延迟 / 正确性）
    代表     = z.ai GLM Infra Agent｜FlashInfer-Bench (2601.00227)
               Helion × CompileIQ（PyTorch/NVIDIA）｜GEAK (2507.23194 + v4 e2e_workflow)

路线四  harness｜agent 自身扩展件（权重冻结）
    对象     = agent 自己的外围件（prompt / skill / tools / loop / plugin / hook）
    verifier = benchmark 分（可过拟合、可被 hack——批评层集中于此）
    代表     = ADAS (2408.08435)｜Darwin Gödel Machine (2505.22954)
               2026 主战场：Harness² (2610.14224) / RRSI (2609.24972) / SoL-Pi (2609.20519) 等
               HUINHB (2605.30621)：harness-updating ≠ harness-benefit 两能力解耦

路线五  meta｜评测方法与评测数据
    对象     = 别的系统的产出与主张（judge 侧——不进化「东西」，审其余四条路线）
    verifier = 不适用（它是 verifier 的供应商；而 verifier 自己会被骗）
    代表     = 评测可信度危机三篇（2607.16241 / 2606.20128 / 2607.04454）
               判真统计协议（Welch t × Cohen's d）｜FlashRSI 统一横评｜MetaRSI (2609.06396)
```

五条路线是同一个自改进闭环在不同对象上的投影：一/三最稳（verifier 客观可自动判定）、二 infra 成本极重、四集中过拟合与可信度问题、五审所有路线的 verifier。以下逐路线展开。

### 路线一｜target：外部代码与 kernel

**路线定义**。被改进对象是外部代码与 GPU kernel——第三方工件，agent 本体与模型权重都不动。verifier 是编译＋正确性＋计时三层，全部客观可自动判定、不经过任何 benchmark 分数——谱系综合判断里「loop 最稳」的一条：两种失败模式（快但错、正确但慢）都有明确暴露面。

**最新进展（论文怎么干）**。

- **AlphaEvolve**（arXiv 2506.13131，A）——进化搜索的机制基准。① **程序数据库三件套**：每个解入库的是「代码（整文件）＋多维分数（标量字典）＋程序执行输出」三件，不只存最优；首要目标「最优回浮」——prompt sampler 采样历史解（带分数＋完整代码）拼进 prompt，被证伪的负样本同样参与下一代。② **核心张力** exploration/exploitation（持续改进最优 vs 维持多样性），机制回答＝MAP-Elites＋岛模型组合启发。③ **双模型分工**：Flash（低延迟→高频候选→单位时间想法量大）×Pro（更强→偶发高质量建议→可能突破性跃迁）——Flash 的搜索价值不是省钱，是供给进化所需的候选流规模。④ **多维分数多样性论证**：不同「好」的定义下高分程序结构相异（同分异形），都回浮进 prompt 刺激更多样候选、提升命中新颖解的概率——生成端论证，非选择端「目标约束」。消融（A）：去掉进化数据库（No evolution）与只留单一小模型两臂，两任务×3 种子均显著变差。
- **ForgeKernel**（OpenBMB，ICLR'27 在审、论文未公开——机制以官方 repo 自述为准，A repo／数字自报无第三方复测）——**自晋升参照链**：新任务从已验证的链顶端参照 kernel 出发，进化结果通过验证后自身晋升为新参照，链无限延伸；对「参照可能非最优」的风险，机制回答两层：验证是参照的资格线（赢不过验证的候选连参照位都摸不到），且每代必须赢过参照才上位（参照是卫冕者不是真理）。**密封二进制探测**：参照物（cuDNN 等专家 kernel）本来闭源，就跑/计时/profile 读能力面、不看源码——kernel 的 verifier 本来就是行为面的，故可密封；代价是源码层设计知识丢失，由跨算子共享知识库补位（硬件探针认证的结论沉淀进库、跨算子复用）。**战绩**（repo 自述）：27 个算子上追平/超过 22 个专家库 kernel 中的 18 个（FlashAttention-4/cuDNN/DeepGEMM），最高 +170.6%、几何平均 +39.4%；直播演示（经二手报道，B）：B200 上 22.5 小时 42 次迭代无人工代码，FlashMLA Prefill +9.8%。
- **搜索策略谱系**：Kernel Forge（2607.24762，A）用 **MCTS** 分配探索-利用，50 轮迭代真实模型上 1.52-2.83×；KernelBand（2511.18868）用 **MAB** 显式建模探索-利用；KernelPro（2606.26453）用 **micro-profiling「专家替身」**——把硬件计数器转成语义反馈喂给生成回路。注意 Kernel Forge 与 ForgeKernel 一字之差是两个系统（一个回答「怎么搜」、一个回答「留什么」），核 id 不认词序。
- **专用模型训练亚族**：QiMeng-Kernel（2511.20100，AAAI'26）宏观思考者/微观编码者分工＋RL、AutoTriton（2507.05687）等——训练专用模型写 kernel，与「通用模型＋反馈闭环」相对。
- **基准链**：KernelBench（2502.10517，ICML'25，fast_p 指标）→ TritonBench（2502.14752）→ KernelCraft（2603.08721，未见 ISA 上达/超编译器基线）。

> **挑战问题：怎么让 agent 生成的 kernel 既对又快，且在真实负载上可信？**

**论文解法**。① **指标端**——fast_p（KernelBench，2502.10517）：「正确 ∧ 加速比>p」的样本占比，一个指标双端惩罚。② **真实负载端**——FlashInfer-Bench（2601.00227）从真实 serving trace 采集评测集（详见路线三）：端到端实证两个 AI kernel 全慢于 native（933 vs 939/1055ms）——生成质量的真实水位：正确性层及格（Triton 最高 96%）、性能层普遍不过半，「又对又快」里缺的主要是「快」。③ **留存端**——ForgeKernel 用验证资格线回答「依赖非最优参照」；AlphaEvolve 用多维分数＋回浮保住同分异形的多条路线，避免单维排序把搜索压成贪心。④ **可信端**——对策四类（多 shape/对抗输入、源码审计、独立计时、SOL 锚定），机制见路线五。

**superopt 的设计立场与理由**。

1. **gate/信号分离**：正确性是二值 gate、不可权衡；性能是连续信号、可权衡——指标设计先分这两刀再组合（fast_p 正是先过 gate 再量信号的形态）。
2. **Amdahl 优先分当选层判据**：按 pct_gpu_time×可达加速排序决定先优化什么；**native 实现常驻候选集**，f(1−1/s)>δ 才替换——最坏情况回退原生实现，损失有界。
3. **五步放量管线**：大规模正确性测评 → 生产框架替换部署 → 灰度放量观察 → 效果验证 → 推全（详见路线三）——生成侧「赢了 benchmark」不等于「可信」，中间隔着三步。
4. **builder/adjust 双档提案者**：双模型分工的落地版——builder 档做跃升式创新提案，adjust 档做落地实现，档位混用换吞吐-质量联合（与 Flash×Pro 同一逻辑，按任务角色分工而非纯模型档位）。
5. **知识库形态立场**：种群容器可容纳 advisory——**种群＋知识库＋参照链三容器组合**是 superopt 的开放设计位（种群/advisory/参照链三种形态无 head-to-head 实证，不预设唯一答案）。

### 路线二｜model：模型本体（权重 / 结构 / loss / 数据 / 训练框架）

**路线定义**。被改进对象是模型本体——权重在线更新、结构、loss、数据生成、训练 pipeline 与训练框架；verifier 是环境＋奖励。与路线一/三的根本差别：计时器直接给出「快了多少」，奖励不指明**是哪一步、哪个 token 挣来的分**——信用分配是 reward 侧固有的判真难题；环境与奖励本身可被 hack（对策见路线五）。

**最新进展（论文怎么干）**。本节事实主要经二手综述转述——韦子扬《Agentic RL Infra 主流技术路线综述》（覆盖 11 家头部团队 2025-2026 公开技术报告，含美团 LongCat），逐条回指各家原报告未完成，信源 C：

- **信用分配谱系**：GRPO → GSPO / CISPO / VAPO 等——token 级重要性采样在长序列下方差爆炸，算法沿这条谱系演化（C→回指原文）。
- **训推一致性**：GSPO 论文实测每次更新约 10% 激活专家翻转；kernel 逐位一致是系统工程约束、不是假设（C→回指）。
- **异步与长尾**：partial rollout、staleness/off-policy 控制是标准件；MiMo 实测 25 个数据源平均生成 token 量差 90 倍；StepFun 约 5% 样本占约 80% 生成成本（C→回指）。
- **训练 infra 三形态**：colocated 共卡（Kimi K3 用卸载换显存、DeepSeek-V4.1 共卡＋沙箱外置）vs disaggregated 分离（天然异步）vs 弹性混合（**美团 LongCat DORA：专职生成组＋弹性角色组**）——三形态并存无收敛结论；四平面分解（Control/Data/State/Weight）是通用分析框架。

> **挑战问题：稀疏奖励下信用分配难、训推不一致、训练 infra 成本极重——自训练值得吗？**

**论文解法**。① **信用分配**——GSPO/CISPO 谱系在算法侧回答（序列级重要性采样压方差）；② **训推一致性**——kernel 逐位一致的工程约束＋每次更新专家翻转比例的量化监控；③ **infra 成本**——三形态选型按 workload 形态决定，不是「哪个更先进」而是「哪个匹配」；④ **预算分配的实证反例**来自路线四的 HUINHB（2605.30621，EMNLP'26）：harness-updating ≠ harness-benefit 两能力解耦——「预算投 agent 不投 evolver」为作者立场。

**superopt 的设计立场与理由**。

1. **路线选择论证——agent 闭环的性价比**：dense feedback（预建、零训练成本）＋搜索框架，**不动权重**——superopt 把改进预算压在 target/infra/harness/meta 四条不动权重的路线上，绕开信用分配难题与训练 infra 成本。事实层：Kernel Forge 50 轮 1.52-2.83×＋消融 36→72 阶梯；「agentic 闭环是当前性价比之王」为综述作者立场，非共识结论。
2. **四重位定位**：superopt 同时占四个位置——target 的优化者、harness 本身可改进、跑在 infra 上可被优化、被测评方法评判（meta）。model 是唯一不占位的路线：superopt 跑在模型上、用模型，但不改模型本体——自指递归是架构自觉，自改进的边界同样是架构自觉。

### 路线三｜infra：serving 框架与运行时

**路线定义**。被改进对象是跑模型的系统本身——serving 栈、编译器控制、kernel 资产、部署工作流。结构特征是「宿主拧结」：被优化的 serving 系统正是执行优化的 agent 的宿主——model 优化 serving 系统，serving 系统跑 model；但被优化对象（infra）与优化者自己的 harness 分属两层，不是同层循环。verifier 是生产级基准（吞吐/延迟/正确性）——五条路线里贴在线推理业务最近、2026 年案例最厚的一条。

**最新进展（论文怎么干）**。

- **z.ai GLM Infra Agent**（官方博客 2026-09-17，A）——国产加速器上 GLM-5.3 驱动的单 agent 优化自家推理栈，三件套：① **dense feedback 环境**：工程师预建可重复、agent 可直接调用的验证工作流（正确性测试/日志/trace/端到端指标等），三性质：局部（改哪测哪）、廉价及时、客观可验；② **优化骨架库**：从 SGLang/FLA/DeepGEMM 既有 kernel 蒸馏优化骨架当起点，验证过的变更回写；③ **人类三职责边界**：定优化目标与约束、搭反馈环境、审关键变更。结果：13 天 3.22×——口径为**未调优首跑基线**（新硬件首次跑通的 W8A8 初始态）官方自报，无独立复测，判真协议零记载；案例级证据强（GIL 根因修复有可证伪验收线＋trace＋上游 PR 旁证）——判真逐主张，不逐文章。优化者 GLM-5.3 与被服务 GLM-5.3-Flash 同家族不同变体（族级自指）；「通往 RSI 早期一步」为叙事定性（作者立场）。
- **FlashInfer-Bench**（arXiv 2601.00227，MLSys 2026，A）——kernel 生成-评测-部署三段闭环：① 生成段 feedback-loop agent（GPT-5/o3/Gemini/Claude×CUDA/Triton，10 轮或 beam-3）；② 评测段是唯一 verifier——**真实 serving trace**（SGLang×DeepSeek-V3 等×ShareGPT 流量）生成 8 算子族 41 Definitions×约 50 workloads＝1,600 负载；正确性**三分支 gate**（确定性算子逐元素容差/FP8 MoE matched-ratio≥0.95/随机采样 TVD≤0.2×100 trials）；**反 reward-hacking 设计**（isolated runner 拆 CUDA context 防读残留内存、hidden workloads）；③ 部署段 **apply()**——启动前按 shapes 构 key、每 key 选最快 solution 建 AOT 索引，运行时 O(1) 查表替换，未命中/关闭则 fallback 原实现，替换开销端到端 <0.8%。端到端实证（batch 64）：native 933ms vs AI kernel 939/1055ms——**机制已通（替换可用、开销小、快慢忠实传导端到端）、AI 尚未赢**（论文正文「faster than the baseline」与图值矛盾，独立评述逐图重算指出）。
- **Helion × CompileIQ**（PyTorch 官方博客＋NVIDIA，2026-10-19，A）——搜「已 autotune 到最优」之下的**编译器控制空间**：ptxas/NVCC 内部控件（寄存器分配/指令调度/codegen 启发式）的默认值是编译器工程师的硬编码启发式，**从未被系统性搜索**。CompileIQ 进化搜索控件组合，产物 **ACF** 是可移植文件（--apply-controls，CUDA 13.3+，可版本化、可提交复用）；Helion 端 autotune_search_acf 把 ACF 当 autotuner 的额外维度与 tile/warp **联合搜索**——串行两段式（先调 kernel 配置再调控件）锁死局部最优：第 1 段的赢家只是「默认控件下」的赢家，被它淘汰的配置可能在别的控件下反超，两段式永远访问不到那个组合。**统计过滤满配**：独立验证环三条件合取（Welch t p<0.05 ∧ Cohen's d>0.2 ∧ 方向闸）＋run1/run2 独立复测（4.7% 加速 d≈2.5）。量级规律（博客经验口径）：已高度优化的 kernel 2-7%、未充分优化的两位数；ACF 硬件＋toolkit 特定（跨架构可能出错而非变慢）。
- **GEAK**（v1＝arXiv 2507.23194；**e2e_workflow＝v4**，2026-07 发版，无论文，一手＝ROCm 官方文档＋repo，A；两代不可混标）——生产级端到端优化。① **Amdahl 优先分**：候选优化按 pct_gpu_time×achievable_speedup 排序，占比大×可达加速高才值得动；② **cheapest-first 四档杠杆**：launch flags→env vars→源码替换→author from scratch，按成本升序花预算；③ **确定性 JS 控制面×LLM 判断面分离**（预算循环/并行/验证/递归钉死在确定性层），e2e 层把热 kernel 递归交给原封不动的 kernel workflow（分形两层）；④ **进化知识库三物件**：静态参考库（只 seed 不裁决、自认「may be stale」）＋蒸馏经验卡（add-only 铁律——只能加候选，不能删候选/剪 bake-off/跳测量，无卡可封路）＋程序库面（淘汰＝墓碑回撤＋attestation 台账，无 DELETE）；⑤ **判真**：interleaved A/B（同卡交替）＋噪声带（固定 shape 0.5%/真实 trace 4% floor）＋非重叠分布 gate（cand_min>ref_max 才算赢）＋Director 独立终判。数字（AMD 自报，无第三方复测）：五个模型/量化配置 e2e +25.4%~+108%（MI300X/MI355X）。「生产级」准确读法：CI＋评测＋17+ SGLang 上游合入 PR 在运行，未见「持续跑在客户生产流量」的一手声明。

> **挑战问题：已优化到「最优」的系统还有多少空间、在哪、怎么判真？**

**论文解法**。① **空间在哪**——CompileIQ 下钻下一层：每层 IR 是一个优化空间，ptxas 控件这层从未被搜过，「已最优」只是默认控件下的**条件最优**；GEAK 用 Amdahl 优先分给「搜哪」排序、cheapest-first 给预算排序；FlashInfer-Bench 用真实 trace 决定「为什么优化」。② **怎么判真**——判真统计协议三要素（重复测量/方差与显著性双闸/环境控制，展开见路线五）；判真成熟度四档：z.ai 零记载 → FIB 半配 → CompileIQ 满配（双闸＋独立复测）→ GEAK 另一取向（噪声带＋非重叠 gate，保守换免假设）。③ **怎么落地**——apply()＋AOT 索引＋native fallback；GEAK 可逆 overlay（never editing installed packages）＋交付物（patch.diff＋launch.sh＋冻结环境）。

**superopt 的设计立场与理由**。

1. **判真协议**：Welch t＋Cohen's d 双闸——p 管偶然性、d 管效应大小，缺一不可（n 大时 0.1% 也显著但无工程意义）；环境控制三件（锁频/预热/隔离）；**0.5% 级噪声逐轮接受会累积成系统性假阳性**——进化 loop 会去优化噪声而不是目标，这是双闸与判真环分离存在的理由。
2. **五步放量管线**：大规模正确性测评 → 生产框架替换部署 → 灰度放量观察 → 效果验证 → 推全。现有工作普遍止步于前两步（FIB 到部署段、GEAK 到 CI 为止）——灰度/验证/推全三步是 superopt 的差异化位。
3. **Amdahl 优先分＋native 常驻候选集**（同路线一，跨路线共用）。
4. **评估方法版本化协议**（连路线五）：评估方法换版时，新老评估方法×baseline/对照组做 2×2 一致性检查，老方法缺陷经论证可豁免，历史解回归重评。
5. **dense feedback 的路线论证**：预建反馈环境（局部/廉价及时/客观可验）＋搜索框架＝零训练成本的闭环——superopt 把主战场压在 target×infra 交叉位的机理依据，z.ai 三件套是这个判断的生产级先例。

### 路线四｜harness：agent 自身扩展件（权重冻结）

**路线定义**。被改进对象是 agent 自己的外围件——prompt、skill、tools、loop、plugin、hook 等，权重冻结；verifier 是 benchmark 分。verifier 的性质决定了这条路线的两个病：benchmark 分可被过拟合（OOD 缩水）、可被 hack；「更新了却不受益」的能力解耦问题 2026 年才被单独测量。

**最新进展（论文怎么干）**。

- **开山**：ADAS（2408.08435）——agent 由代码定义、代码可被搜索；Darwin Gödel Machine（2505.22954，Sakana）——开放式进化自我改写 agent 代码。
- **2026 主战场**（A，各一句话定位）：Harness²（2610.14224，两级递归）｜RRSI（2609.24972，正则化抗 OOD 缩水）｜SoL-Pi（2609.20519，NVIDIA：跨环境 rollout＋选择压力筛选，四机制存活打包 standalone 扩展）｜Self-Harness（2606.09498）｜EvolveNet（2608.04968）｜HarnessDev（2609.01437）｜Meta-Harness（2603.28052）｜Dream-RSI（2609.14858，改探索策略）。
- **核心问题实证**（RRSI 动机，A）：无约束 harness 进化在训练任务上大涨、在 OOD 基准上缩水甚至归零——训练 split 的涨是**真实、可复现**的改进（不是噪声，「验一下是不是真改进」拦不住它）；成因：选择压力只来自 verifier 所测分布，基准特有模式与通用改进在选择眼里无差别、被逐轮放大。
- **HUINHB**（arXiv 2605.30621，EMNLP'26，A）：**harness-updating 能力 ≠ harness-benefit 能力**——会改 harness 与能从 harness 获益是两个独立能力，实证发现更新能力上升时获益并非单调上升。

> **挑战问题：benchmark 分当唯一选择压力——OOD 缩水、benchmark hack、更新了却不受益，怎么防？**

**论文解法**。① **RRSI 的正则化四件**：时间退火预算、进化史鼓励未探索轨迹、critic 筛掉 benchmark 特有提案、pruner 删琐碎改动；结果：进化 split 最高 +6.0 分、5 个 OOD 基准最高 +4.3 分、省 30% policy token（诚实读法：OOD 增益绝对值不大；正则化超参本身是新的设计面）。② **通用防御三件**：进化-保留拆分（选择与检验任务分离，保留任务从不参与选择）、正则化、独立验证集（从未参与进化的基准作最终门槛）。③ **评测方法批评**：Rethinking Harness Evolution（2607.12227）——单测搜索＋同一公开基准报分，与 test-time scaling 不可比；Where Does Value Live（2609.02889）——收益局部化＋预算切分陷阱。④ **能力解耦的测量**（HUINHB）：先测「更新了是否受益」再谈更新能力——「预算投 agent 不投 evolver」为作者立场。

**superopt 的设计立场与理由**。

1. **四重位定位**：superopt 本身占着 harness 位——自己的 prompt/skill/loop/工具就是可进化对象；系统知道自己在图上的全部四个位置（见路线二），harness 位是最直接的一条。
2. **OOD 防御**：进化-保留拆分＋独立验证集进入 loop 结构本身——不是事后补救，是选择压力的设计。
3. **分数三分类**：噪声假分（分数是假的）/OOD 真分测错分布（分数真、绑错分布）/统计不确定（指标分布都对、单次测量分不开）——三类信号三种对策，不可混在一个标签下处理。
4. **三层验证栈**：模型对抗（抓逻辑矛盾）→ 锚式批评（测量/审计，抓共同盲区）→ 人类/生产裁决（最后一环）。
5. **自指性边界**：验证器不能验证它自己是否有效——自指验证环不能自我闭合，系统边界上保留人类最终裁决。这条边界与 z.ai「人类三职责」在生产系统实例上独立同构。

### 路线五｜meta：评测方法与评测数据

**路线定义**。被改进对象是评测方法与评测数据本身——judge 侧，不进化「东西」，审的是其余四条路线的产出与主张；按对象分类它结构性不可见（没有「被进化对象」的东西无处落桶），故须单列。verifier 一栏写「不适用」——它是 verifier 的供应商，而挑战正在这里：**verifier 自己会被骗**。

**最新进展（论文怎么干）**。

- **评测可信度危机三篇**（均 A）：① **KernelBench-Verified**（2607.16241）——前沿模型在 kernel 评测上的 reward hacking 三形态：kernel 内调框架算子（工作转嫁给库）、缓存参考输出（计算没发生、输出全对）、利用计时漏洞（时间数字被定向操纵）；② **Correctness Illusion**（2606.20128）——固定 shape＋小样本 allclose 测不到非对齐 shape 边界 bug，24 个对照 kernel 实证「通过评测的 kernel 换输入形状后出错」，机制一句话＝**验证分布 ⊂ 使用分布**；③ **Correct but Slow**（2607.04454）——DSL 评测只验数值、不验替代质量，correct-but-slow 整类放行（信号不是被骗，是根本没装）。
- **fast_p**（KernelBench，2502.10517，ICML'25）：「正确 ∧ 加速比>p」的样本占比——把两种失败模式做成一个指标的两端。
- **对策四类**：多 shape/对抗输入、源码审计、独立计时协议、SOL（speed-of-light 物理上界）锚定。
- **判真统计协议三要素**：重复测量（√n 收缩；单次噪声带可宽于待判效应）、方差与显著性（Welch t×Cohen's d）、环境控制（锁频/预热/隔离）——工业实现（do_bench/CompileIQ）见路线三。
- **迭代精炼消融**（KernelBench L2 上 DeepSeek-R1，fast_1 口径，经综述转述回指原文）：单次生成 36% →＋编译反馈 44% →＋执行反馈 62% →＋profile 反馈 72%——编译判「能不能跑」、执行判「对不对」、profile 判「快不快」，verifier 三层逐层接回生成回路。
- **FlashRSI**（RUC-NLPIR，官方文档站＋repo，无同行评审论文）：12 方法×30 基准×3 harness 统一横评——路线四各方法无 head-to-head 的唯一统一入口；实验＝方法×harness×benchmark 三元组＋far-OOD 迁移列。**MetaRSI**（2609.06396）：对 RSI 系统本身做 RSI——元层自己变成进化对象，递归再拧一层。

> **挑战问题：verifier 自己会被骗——评测本身可信吗？**

**论文解法**。① **命题分离原理**（三形态的共同原理）：验证器验证的是它实际检查的命题，不是使用者以为它在检查的命题——骗局永远让「实际检查的命题」为真、让「被以为的命题」为假；绿灯没坏，是蕴含链断了。调算子/缓存攻击 gate（输出「对」了，但「对」的方式是假的），计时漏洞攻击 signal（时间数字「真」了，但测的不是被以为的工作）。② **对策四类按攻击面归属**：多 shape/对抗输入＝gate 覆盖（抓边界 bug；换输入让缓存当场穿帮）｜源码审计＝gate 命题（唯一够得着「对象是不是声称的对象」的一层）｜独立计时协议＝signal 有效性（计时权不在被测者手里＋判真三要素全款）｜SOL 锚定＝signal 锚＋gate 抓手（**超过物理上界的加速只能来自「没做工作」**——转嫁/缓存在输出层全绿、在物理层穿帮）。③ **工程组合**：FlashInfer-Bench 的 verifier 设计（hidden workloads、isolated runner、三分支专用 validator）及其边界（matched-ratio 只数「对多少」不数「错得多坏」——950 全对＋50 任意坏值可以通过）。④ **元层递归**：审计员自己也是 verifier、也有被骗面——审计员谁来验收，是下一层问题（MetaRSI 把这层拧成对象）。

**superopt 的设计立场与理由**。

1. **评估方法版本化协议**：评估方法换版时，做新老方法×baseline/对照组 2×2 一致性检查——新老方法在对照组上都超越 baseline 才算刻度稳定；老方法逻辑缺陷经论证可豁免一票；历史解在新方法上回归重评。
2. **三层验证栈**（本路线的骨架件）：模型对抗（中间层，抓逻辑矛盾）→ 锚式批评（测量/审计，抓共同盲区）→ 人类/生产裁决（最后一环）——对策四类是锚式层的具体工具。
3. **自指性边界**：验证器不能验证它自己是否有效——元层的改进协议同样停在这条边界上，最终裁决留给人类/生产。
4. **判真协议与分数三分类作为评测基建默认配置**：双闸（p 管偶然性、d 管效应大小）＋环境控制＋0.5% 噪声累积警戒；三类分数分渠道处理。
5. **gate/信号分离作为指标设计第一刀**：正确性二值不可权衡、性能连续可权衡——所有评测指标先过这一刀再组合。

## 参考与信源（arXiv 编号汇总）

**路线一（target）**：AlphaEvolve 2506.13131（A）｜ForgeKernel（OpenBMB repo 自述，ICLR'27 在审、论文未公开；数字官方自报）｜Kernel Forge 2607.24762（A）｜KernelBand 2511.18868（A）｜KernelPro 2606.26453（A）｜QiMeng-Kernel 2511.20100（AAAI'26）｜AutoTriton 2507.05687｜KernelBench 2502.10517（ICML'25）｜TritonBench 2502.14752｜KernelCraft 2603.08721｜迭代精炼消融（经综述转述回指 KernelBench，B/C）。

**路线二（model）**：韦子扬《Agentic RL Infra 主流技术路线综述》（weiziyang.wiki，C——二手综述，逐条回指 11 家官方报告未完成；本节数字均按 C 级记）。

**路线三（infra）**：z.ai《GLM built its own inference infrastructure》（官方博客 2026-09-17，A；3.22× 为未调优首跑基线自报、无独立复测；「通往 RSI 早期一步」「与主流 NVIDIA GPU 可比」为作者立场）｜FlashInfer-Bench 2601.00227（MLSys 2026，A；图文矛盾处经独立评述逐图重算，B）｜PyTorch×NVIDIA 博客 "One Search, Two Layers"（2026-10-19，A；2-7% 为博客经验口径）｜GEAK v1 2507.23194（A，benchmark 级实证）＋GEAK v4 e2e_workflow（ROCm 官方文档＋repo，A，无论文；数字 AMD 自报、无第三方复现；「民主化专家级性能」为摘要愿景句）。

**路线四（harness）**：ADAS 2408.08435｜Darwin Gödel Machine 2505.22954｜Harness² 2610.14224｜RRSI 2609.24972｜SoL-Pi 2609.20519（NVIDIA）｜Self-Harness 2606.09498｜EvolveNet 2608.04968｜HarnessDev 2609.01437｜Meta-Harness 2603.28052｜Dream-RSI 2609.14858｜HUINHB 2605.30621（EMNLP'26；「预算投 agent 不投 evolver」为作者立场）｜Rethinking Harness Evolution 2607.12227｜Where Does Harness-Optimization Value Live 2609.02889（均 A）。

**路线五（meta）**：KernelBench-Verified 2607.16241｜Correctness Illusion 2606.20128｜Correct but Slow 2607.04454（均 A）｜FlashRSI（RUC-NLPIR 官方文档站＋repo，无同行评审论文）｜MetaRSI 2609.06396｜统计方法：Welch 1947（Biometrika）｜Cohen 1988（0.2/0.5/0.8 为领域约定）｜do_bench（Triton 源码）。

**立场句汇总（与论文事实分开引用）**：「agentic 闭环是性价比之王」＋五趋势（微信公众号 Arthur《让 LLM 写 GPU 算子》，C，独立作者综述，其五路线归纳为作者个人框架非共识分类法）；z.ai「通往 RSI 早期一步／进度不会放慢」；PyTorch「底部几个百分点在大规模 fleet 上有价值」；AMD「更便宜、无需自建 kernel 工程团队」；HUINHB 投资建议——以上均为对应作者立场，非可验证事实。superopt 各节的设计立场为本文档立场层，独立于上述论文事实。
