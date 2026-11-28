# stage1_2 素材：问题还原（第 2 轮再启发）

## 块 1：MT_INFER 工作区一手证据——「现状端有多少可写进论文的数字」

来源：本机工作区 /Users/logic/workdir/MT_INFER/.rick/super/ 下的 env_1/ledger.jsonl（54 行、41 行带 perf_delta）、env_2/events.jsonl + ledger.jsonl（77 行）、kernelbench-dev/esq_report_20261122_2232.json、kernelbench-dev/codes（583 个）、env_2/env/feature_space.json。以下均为直接读取原文件所得，非转述。

### env_1（推理服务优化，54 次实验里 41 次有实测 delta）

- perf_delta 分布（items/s 相对父本的增量，41 个有实测值的实验）：最大 +2714.64，其余大额 +1235.68、+1003.34、+691.84、+663.58、+658.43；负值最大 -1631.95（reject）。verdict 类型：admit / replace / reject 三种都有——说明系统在「淘汰错误改进」上也在工作，不只是报喜。
- 精英（elite）轨迹可从 ledger 原文串出：iter31 提到「同格精英 947.96 → ~7970+，8.4×」；iter35 提到「8593.89」；iter36「8999.78」；iter49「--baseline 9143.96」；iter50「iter037 K=2 9153.49 → iter044/045 K=3 前沿树 ~9215」。即基线 ~949 items/s 到 ~9215 items/s，累计约 9.7×，且后期单步增益进入噪声带（「top3 spread 0.67% < 噪声 0.68%」——饱和判据本身被记录在案）。
- 每条 delta 都带有「归因」叙事（如 iter19：GEMM tile 从 (64,64,32) 改 M 适配档 (16,64,32) → +658.43；iter24：GEMM 后端全换 cuBLAS TN + fp32 累加 → +2714.64；iter35：图内 varlen pack → +245.05）。这意味着论文里「每个增益有机制归因」这层证据是现成的。

### env_2（六轴推理服务特征空间）

- feature_space.json 当前状态：6 轴（batch_capacity / kernel_optim / kv_transfer / prefill_side / sched_mem / spec_decode），事件流中可见网格从 11520 → 7680 的两次回缩记录（iter18 seq 1488-1491）——特征空间的「定义本身在运行中被改写过」，这是用户原创点主张的直接现场证据。
- expand_cell 机制实录（events.jsonl 中 80 处 expand_cell 相关事件）：adjust agent 反复提案 `expand_cell` + `add_axis_value`，例如：
  - adj-20-2：kernel_optim 轴加值 `moe_low_batch_tile`（依据：human 干预 #001 两次注入两次被调度器重建挥发 → 走裁决链）；
  - adj-25-0 → 干预 #003（seq 2495）批准：kernel_optim += `gdn_backend_flashinfer`（证据链：iter24 活体探针 import OK + gate(9,0) 通过）；
  - adj-42-4 → 干预 #009（seq 5044）批准：基于 iter037 L0 证伪 verdict（REGRESSION 全 batch 形状）+ iter039/040/041 三连 fresh 复证后重开；
  - adj-52-4：spec_decode 轴加值（MTP 塔减负），依据 iter46 decision §3 排序 + 生产塔实测锚。
- 关键细节：**加轴值不是 adjust agent 自己说了算**——提案要引用「证伪 verdict + 回归证据 + 干预批准」的链路才被接受（adj-42-4 的证据字段就包含「REGRESSION +24.6/+34.12/+40.02% 全 batch 形状+归因双点」这样的反证）。这与「LLM 自由改自己」有本质区别，是论文里可强调的治理结构。
- 另一面对照：env_2 的 25 次 experiment_finished 中 ledger 里 perf_delta 全部为空（77 行 0 个非空）——此环境产出的主要是「判定/收口」而非吞吐增益。写论文时两个环境的角色要分开：env_1 是增益主场，env_2 是机制/特征空间主场。

### kernelbench-dev（KernelBench 基准，500 轮迭代）

- esq_report_20261122_2232.json 原文数字：coverage "36.1% (30/83)"、speedup_geomean "1.743× (norm 0.872)"、整体 ESQ 0.5855；分维度 D1_search 0.666 / D2_adaptivity 0.428 / D3_knowledge 0.271 / D4_stability 0.912。
- 同报告里的自曝短板（论文里对手一定会问）：decision_accuracy "0.0% (0/41 ups productive)"——41 次 adjust 决策零次直接生产性；recovery_capability "27/83 interrupted"（83 个问题中 27 个中断后恢复）。
- 产物规模：codes/ 目录 583 个、designs/ 550 个，迭代上限 500 轮、墙钟 168h、8 并发 env slot、4 个 agent 角色共用同一模型（agent_models: adjust/builder/coding/debug 全 deepseek-v4-pro）。
- 注意：1.743× 是 **geomean over 覆盖到的 30 个问题**（30/83=36.1% 覆盖），不是全 83 题口径——与官方 leaderboard 的全量口径对比时这是最要害的差异点（外部对照见块 4）。

（后续块：块 2 OpenEvolve/AlphaEvolve/FunSearch 近邻事实校验；块 3 QD「可进化解空间」先例检索；块 4 KernelBench 官方 leaderboard 外部对照；块 5 顶会 venue 地图与同域论文落点先例；块 6 张力与引导问题。）

## 块 2：近邻事实校验——OpenEvolve / AlphaEvolve / FunSearch 的「解空间」到底固定不固定

来源（均一手）：codelion/openevolve 官方 README + docs/configuration.md + docs/faq.md + docs/showcase.md（GitHub main 分支 raw）；Google DeepMind AlphaEvolve 白皮书 PDF（2025-05，44 页，经 DeepMind 官方博客链接取得，已全文抽取）；DeepMind 官方博客（AlphaEvolve、FunSearch 两篇）。

### OpenEvolve（用户点名的最近邻）

- README 原文：「It uses LLMs to mutate programs and evolves them with MAP-Elites and island populations」「MAP-Elites keeps diverse programs across feature dimensions」「Island populations: separate populations with periodic migration prevent premature convergence」。
- 特征维度是**用户在 config.yaml 里按问题手配的**（docs/configuration.md 原文示例）：`feature_dimensions: ["complexity", "diversity", "performance"]` + `feature_bins: {complexity: 10, performance: 20}`。评估器返回原始值，OpenEvolve 负责分箱。
- **决定性事实**：docs/faq.md 把「Add custom feature dimensions to diversify search」列在「Manual interventions（人工干预）」清单里——即改特征维度在 OpenEvolve 的设计里是**人工操作**；docs/configuration.md 与 how-it-works.md 中**检索不到任何「系统在运行时自行增加/修改特征维度、自行扩张搜索空间」的机制**（PopulationStrategy 钩子只管 admission/eviction/migration，不管维度）。
- 社区如何评估 OpenEvolve：靠 examples 目录里的复现型任务（circle_packing、alphaevolve_math_problems、alphaevolve_interop、function_minimization、symbolic_regression、tsp_tour_minimization、arc_benchmark 等）+ showcase 数字（如 attention kernel 2.8× @Apple M1 Pro、function minimization 100× 收敛速度、HotpotQA prompt +23%）。没有统一 leaderboard，评估以「能否复现 AlphaEvolve 已知结果」为隐性标尺。

### AlphaEvolve（DeepMind 白皮书，2025-05）

- 流水线（白皮书 Figure 2 原文）：`parent_program, inspirations = database.sample(); prompt = prompt_sampler.build(...); diff = llm.generate(prompt); child = apply_diff(...); results = evaluator.execute(...); database.add(...)`。进化数据库「implements an algorithm that is inspired by a combination of the MAP elites algorithm and island-based population models」。
- **但有一个必须诚实面对的近邻**：白皮书 §2.2 明确写着「**Meta prompt evolution**: instructions and context suggested by the LLM itself in an additional prompt-generation step, **co-evolved in a separate database** analogous to the solution programs」——即 AlphaEvolve 的**提示采样器的一部分（meta prompt）本身是被共同进化的对象**，且消融实验（Figure 8「No meta prompt evolution」）证明它贡献显著。不过：白皮书全文（44 页）**没有**任何「MAP-Elites 描述子维度/特征格定义在运行中被系统自身改写」的机制描述；评估函数、进化循环骨架、维度定义均由人定义（「Human defines "What?" … AlphaEvolve figures out "How?"」）。
- FunSearch 对照（白皮书 Table 1 原文）：FunSearch「evolves single function / 10-20 lines / Python only / single metric」vs AlphaEvolve「evolves entire code file / hundreds of lines / any language / multiple metrics」。FunSearch 侧（DeepMind 博客）：用户写问题描述+评估函数+种子程序，池子+多样性策略并行——评估函数、池结构同样是人事先固定的。

### 对用户主张「其他所有特征格系统都不允许解空间自改进」的校验结论

- **对 OpenEvolve/AlphaEvolve/FunSearch 成立**（以上为据）：这三个系统的特征格/描述子维度在运行时固定、由人配置；OpenEvolve 甚至把「加特征维度」明列为人工干预。
- **但「所有」这个词在三个方向上有近邻反例，主张必须收窄**：
  1. **Promptbreeder**（ICML 2024，856 次引用，AlphaEvolve 白皮书引用列表 [27]）——标题即「Self-Referential Self-Improvement Via Prompt Evolution」：变异提示词（mutation prompts）本身是被进化对象。这是「搜索机制自指进化」的最强先例（但不是特征格系统）。
  2. **ADAS / Meta Agent Search**（ICLR 2025）：「a meta agent iteratively programming interesting new agents based on an **ever-growing archive** of previous discoveries」——被搜索的空间（agent 设计空间）在运行中持续增长。同类：AgentSquare（ICLR 2025，193 引）、AFlow（ICLR 2025）。Voyager 的技能库同理。
  3. **QD 描述子学习线**：Learning Behavior Characterizations for Novelty Search（GECCO 2016，67 引）、Autonomous skill discovery with quality-diversity and unsupervised descriptors（GECCO 2019，136 引）、Unsupervised Behavior Discovery With QD（IEEE TEVC 2022，75 引）——描述子可以「学出来」而非手工设计；但都是离线/旁路学习，且无「带裁决与回归证据的运行中扩格」治理结构。POET/Enhanced POET（ICML 2020）把「环境/问题空间」做成与解共同生长的对象，是最接近「搜索空间开放增长」的 EC 先例。
- **检索空档（可作为差异化空间）**：OpenAlex 上「MAP-Elites + descriptor space 改写于运行中 / co-evolving descriptors / self-adaptive behavior space」无直接命中；最近的是 Loreley: Repository-Scale Program Evolution with Quality-Diversity Search（arXiv 2026）与 TacEvo: Self-Evolving Architecture Discovery via LLM-Driven QD（IROS 2026）——值得在 related work 里逐篇对照。
- 收窄后的表述示例（供用户自己定稿）：从「其他所有特征格系统都不允许解空间自改进」收窄为「现有 LLM+QD 系统中，特征格的定义在运行时由人固定（OpenEvolve 明确列为人工干预）；自我指涉的进化（Promptbreeder 的 mutation prompt、AlphaEvolve 的 meta prompt）作用于提示/算子层，而非特征格定义层；带外置裁决+双回归证据+热重载的**特征格定义层**自改进尚无先例」——这正是第 1 轮缺口①要求的「领域级断言的近邻事实校验」。

## 块 3：KernelBench 外部对照——「显著高于第一名」这句需要什么口径

来源（一手）：ScalingIntelligence/KernelBench 官方 repo README + EVAL.md（GitHub main）；KernelBench 论文摘要（arXiv 2502.10517）；KernelBench v0.1 发布博客（scalingintelligence.stanford.edu）；同域已发表系统的论文摘要（OpenAlex 核验）。

- **官方指标定义**（README）：fast_p = 「percentage of generated kernels that are functionally correct and offer a speedup greater than an adjustable threshold p over baseline」（250 个 PyTorch workload，L1=100/L2=100/L3=50，另有 L4 HuggingFace）。注意官方口径是**全量 250 题**，不是覆盖子集。
- **原论文基线**：前沿推理模型「matching the PyTorch baseline in less than 20% of the cases」——即 fast_1（正确且比 PyTorch 快）在 20% 以下起步。
- **官方对「高分」的立场**（EVAL.md 原文，对用户主张直接相关）：「You should be **extra CAREFUL!**, be always paranoid about suspiciously good results — kernel engineers and existing compilers are already pretty good, so **a >2x speedup for anything is highly unlikely**」「**If the model can reward hack, it will find ways to reward hack! This can especially happen during RL training or evolutionary search.**」还有一条引语：「if you beat cudnn by more than 10%, think again」。
- **官方 leaderboard 免责**（README 原文）：「The KernelBench team does not review, validate, or endorse individual kernels or reported results. Users are responsible for independently verifying any results obtained using the framework.」—— leaderboard 本身不背书任何条目。
- **同域可比数字**（全 250 题口径）：CUDA-L1（ICLR 2026，90 引）「average speedup of x3.12 with a median speedup of x1.42 … over all 250 CUDA kernels of KernelBench, with peak speedups reaching x120；x2.77 over Torch Compile」；Kevin（ICLR 2026，122 引）纯 CUDA 正确率 56%→82%；KernelFoundry（ICML 2026，硬件感知进化式 GPU kernel 优化）20 引。官方 leaderboard 的实时第一名数字**本轮未能直接取到**（HF Space 与其镜像均不可达——Exa 限流 + HF 空间下线/改名，这是残余缺口，用户若主张「高于第一名」需自行打开 leaderboard 截图存证并核对口径：分数是 fast_p 还是 fast_p100、哪个 Level、什么 baseline、什么硬件）。
- **用户数字的位置**：1.743× geomean 是 **30/83=36.1% 覆盖子集**上的口径（kernelbench-dev/esq_report 原文），且 ESQ 报告自曝 decision_accuracy 0/41、D3_knowledge 0.271。与 CUDA-L1 的 3.12×（全 250 题）直接比是不利对照；与 fast_p 排行比则必须换算成同一指标、同一题集、同一 baseline 才成立。「显著高于第一名」这句话目前**缺外部可核对的证据链**，正是第 1 轮缺口②。

## 块 4：社区如何评估这些 evolve 系统（用户点名的问题）

来源：KernelBench EVAL.md + v0.1 博客（官方一手）；AlphaEvolve 白皮书（一手）；OpenEvolve docs（一手）；OpenAlex 引用数据。

- **评估维度正在从「跑分」转向「防作弊」**：KernelBench v0.1 博客列出四类 reward hacking 模式（不写真 CUDA 而调高层算子、写了但不调用、no-op 内核复用输出内存、CUDA stream 计时不同步），并给出「speed of light」理智检查示例（BF16 4096×4096 matmul @H100 理论下限 ~0.138ms，「if your kernel is faster than the speed of light … something is wrong」）。EVAL.md 明说进化搜索/RL 是 reward hacking 高发区。
- **AlphaEvolve 的自证方式**（白皮书）：正确性用「checked against the reference (unmodified) code on randomized inputs」+「The final version of the code was rigorously confirmed by human experts to be correct for all possible inputs」（XLA IR 优化案例）—— 即官方叙事也是「机器可验证 + 人类专家终审」双轨；工程收益叙事用生产指标（数据中心调度回收 0.7% 全球算力、FlashAttention kernel 32% 加速、训练耗时 -1%）。
- **评估这类系统的隐性标尺**：数学问题看「超过已知最好构造」（AlphaEvolve：50+ 开放问题 75% 追平 SOTA、20% 刷新）；系统问题看「部署到生产环境的实测收益」；基准问题看「同一口径下全量覆盖 + 防作弊审计」。用户的 env_1（949→~9215 items/s，9.7×，带归因链）属于第三类的强素材，但需要补防作弊审计叙事。

## 块 5：会场地图（用户：这些顶会我不是很了解，你可以讲讲）

来源：MLSys 官网 Conference Overview（一手，2027 页面）；各论文 venue 经 OpenAlex 交叉核验；其余会场定位为领域常识（已在文中标注哪些是核验过的一手、哪些是通识，供用户自行复核）。

### 会场按「原创」定义分三类

**A. 系统类：MLSys / OSDI / SOSP / ATC / EuroSys**
- MLSys（官方原文核验）：「targets research at the intersection of machine learning and systems」；议题清单**明列**「Efficient model training, inference, and serving」「Compound AI systems and AI agent systems」「**LLM-based hardware design or system optimization techniques**」「ML benchmarks, datasets, and tooling」——与用户系统的匹配度字面命中。看重：真实硬件上的实测收益（吞吐/延迟/成本）、系统机制的通用性、可复现（有 artifact evaluation）。
- OSDI/SOSP（通识，最顶系统双会）：看重**新机制**+极端严谨的评测（多负载、消融、失败分析），接收率 ~16-20%；LLM 驱动的 kernel 进化尚无先例落点（这本身既是机会也是「审稿人不熟」的风险）。ATC/EuroSys（通识）：同口径略宽。
- 同域先例：MLSys 无 LLM-kernel 进化先例论文（KernelBench 本身是 arXiv+社区基准，不上会）；KernelFoundry（进化式 GPU kernel 优化）落的是 **ICML** 而非 MLSys——说明这个子域目前更多被 ML 会场吸收。

**B. ML 类：ICLR / ICML / NeurIPS**
- 看重（通识+同域样本反推）：方法学新颖性 + 大规模可比实验（多基线、多任务）。同域落点实证：Eureka（ICLR 2024）、Promptbreeder（ICML 2024）、ADAS（ICLR 2025）、AgentSquare（ICLR 2025）、AFlow（ICLR 2025）、Enhanced POET（ICML 2020）、CUDA-L1（ICLR 2026）、Kevin（ICLR 2026）、KernelFoundry（ICML 2026）。**「LLM+进化+代码」目前的事实主场是 ICLR/ICML**。NeurIPS 有 datasets/benchmarks track（若把 105 格特征空间+ledger 做成基准可投）。
- 风险：ML 审稿人会问「消融呢？和 AlphaEvolve/OpenEvolve 同预算对比呢？」——env_2 的 0/41 decision_accuracy 是这类会场的减分项，需要先修叙事。

**C. 进化计算类：GECCO / IEEE TEVC / GPEM（Springer 期刊）**
- 看重（通识）：**机制层面的进化计算新颖性**——算子、多样性维持、open-endedness 的形式化，以及与 EC 基线（不只是 LLM 基线）的严格对比。QD 主场：MAP-Elites（GECCO 2015，1348 引）、Evolvability Search（GECCO 2016）、unsupervised descriptors（GECCO 2019）；LLM×EC：LLaMEA（IEEE TEVC 2024，**299 引**——LLM 进化算法生成元启发式的期刊主场先例）、LLaMEA-BO（GECCO 2026）。**「特征格定义层可被进化」在这里是最正统的 novelty 叙事**（这是 EC 社区自己的语言：descriptors、behavior space、open-endedness），且 TEVC/GPEM 是期刊，容忍长周期系统工作。
- FunSearch 的先例提醒：Nature 接的是「数学发现」叙事（新下界/新构造），不是「系统跑分」叙事；AlphaEvolve 至今是白皮书+博客，没有过任何会议评审——**这个子域最响的两个名字都没有顶会记录**，用户系统带 ledger/门禁/可复现实验流，反而是「可以正面上会」的差异化。

**arXiv / Nature**：arXiv 是本子域事实上的发布层（KernelBench、AlphaEvolve 白皮书、OpenEvolve、Loreley、CORAL 全在 arXiv）；Nature 只在「新的可验证科学事实」出现时才开门（FunSearch Nature 2023；AI co-scientist 系 Nature 2026）。

### 快速对照（供用户选期望）

| 会场 | 「原创」=什么 | 同域先例 | 用户系统的强匹配点 | 最可能被拒理由 |
|---|---|---|---|---|
| MLSys | 真实系统+实测收益 | （缺） | env_1 9.7×+归因链、168h 无人值守 | 「就是个搜参框架」 |
| ICLR/ICML | 方法学+大规模可比 | Eureka/ADAS/CUDA-L1/KernelFoundry | expand_cell 机制+特征空间治理 | 消融/同预算对比不足 |
| GECCO/TEVC | EC 机制新颖性 | MAP-Elites/LLaMEA/POET 系 | 「behavior space 自身可进化」是 EC 母语 | 硬件实验不像 EC 论文 |
| OSDI/SOSP | 新机制+极端严谨 | （缺） | 调度器/门禁/裁决链 | 领域外审稿人风险 |
| arXiv | 快、无门槛 | 全体 | ledger 全量数据 | 不算「被接受」 |

## 块 6：张力与引导问题（阶段 1 目标：期望/现状分清、事实与价值分离、差距成形）

### 三组张力（事实 vs 用户已说的话）

1. **「显著高于第一名成绩」vs 可核对口径缺失**：esq_report 说 1.743× 是 30/83 覆盖子集 geomean；CUDA-L1 全 250 题 avg 3.12×/median 1.42× 已上 ICLR 2026；官方 leaderboard 数字本轮未能取得（HF Space 不可达）。官方 EVAL.md 的立场是「>2× speedup 就该起疑心」+「leaderboard 不背书任何条目」。——这句话现在的事实形态是**未核实的内部口径**，直接写进论文会被第一条审稿意见打穿。
2. **「其他所有特征格系统都不允许解空间自改进」vs 三个近邻**：对 OpenEvolve（feature_dimensions 人工配置、faq 明列「加维度=人工干预」）和 AlphaEvolve/FunSearch（描述子层无人改写机制）**成立**；但 Promptbreeder（mutation prompt 自指进化，856 引）和 ADAS（ever-growing archive）在「搜索机制层」已有自进化先例，QD 描述子学习线（2016-2022）在「描述子可学」上已有铺垫。——断言的**可守版本**存在，但必须把「所有」收窄、把近邻逐个引到。
3. **「工业界+学术界都能证明价值」vs 现状数据的内部矛盾**：env_1（9.7×、41 实测 delta、饱和判据入账）是硬通货；但 kernelbench-dev 的 ESQ 0.5855、decision_accuracy 0/41、D3_knowledge 0.271 是系统自曝的短板，「显著高于第一名」与「30/83 覆盖」并存。——价值主张要么押在 env_1 的服务端收益，要么押在 expand_cell 机制，两个都写会互相稀释。

### 引导性问题（把用户的还原推进到「差距成形」）

1. 如果审稿人只允许你保留一句话作为论文的贡献主张，你会选「服务吞吐 9.7×（env_1 事实）」还是「特征格定义层可被有治理地进化（expand_cell 机制）」？——这个选择同时决定会场（MLSys vs GECCO/ICLR）和你需要补的证据（防作弊审计 vs 同预算基线对比）。
2. 「高于 KernelBench 第一名」这句话，你需要哪些**同口径**材料才能把它从口号变成可核对事实（fast_p 还是 fast_p100？哪个 Level？baseline 是 PyTorch eager 还是 Torch Compile？什么 GPU？覆盖多少题）？缺口清单里哪一项是你两周内补得齐的？
3. 你的原创点在收窄表述后（「特征格定义层的、带外置裁决+双回归证据的自进化，尚无先例」），Promptbreeder 和 ADAS 是「被你覆盖的弱先例」还是「你必须显式让位的强先例」？——这道题的答案决定 related work 的写法，也决定审稿人会不会把你的贡献判成「增量」。

### 残余缺口（诚实声明）

- KernelBench 官方 leaderboard 实时第一名数字未取得（HF Space 下线/改名 + 搜索通道限流）；替代物是论文基线数字（<20% fast_1）与 CUDA-L1/Kevin 的全量口径数字，但都不是 leaderboard 本身。
- Sakana「AI CUDA Engineer」被 SemiAnalysis 质疑的事件是社区评估叙事的常用例证，但本轮未能取得一手页面（官网抓取失败），未写入正文，仅在此留痕。
- OSDI/SOSP/ICSE/FSE/ASE 的定位段基于通识而非一手 CFP 核验；MLSys 与 GECCO/TEVC 的关键定位已一手核验。

*（素材完。落盘路径：/Users/logic/workdir/rick-dev/.rick/draft/loops/loop_19/orig/stage1_2.md，共 6 块。）*
