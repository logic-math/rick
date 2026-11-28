# stage2_1 · 视角生产（阶段 2 首启）：ABCD 架构「理论基础 + 硬技术亮点」候选知识领域地图

> orig 调研素材 · loop_19 · 用户阶段：视角生产（为假设链的每个环节找候选理论领域，强相关/弱相关分开，不选框架、不收敛候选）
> 用户假设链（阶段 1 产出摘要）：自然语言≈语义空间低维流形 → code≈其上离散低维流形 → 进化=低维流形间跳跃找更优解 → LLM 概率分布保证跳跃的最低限度确定性 → 优秀解跨域同构、催化剂式信息驱动解空间跳跃 → 开放信息获取驱动解空间自改进 → 有限 token 预算分配到最可能产生跳跃解的特征格（解空间形状自改进）。

---

## 1. 入口项目学术定位（用户指定的两个项目）

### 1.1 ScholarCatalyst（斯坦福 AI Lab 博客 "Research Intuition as a Retrieval Task"）

- **一手来源**：Stanford AI Lab Blog，Sohyeon Kim & Yoonho Lee，2026-11-09（https://ai.stanford.edu/blog/scholarcatalyst/）；上游论文：**ScholarCatalyst: A Benchmark for Retrieving Papers that Inspire New Research**（arXiv:2610.02202，作者含 Kim, Lee, Bo Liu, Rulin Shao, Seungone Kim, Neubig, Koh, Chowdhery, Asai, Khattab, Yejin Choi, Gunhee Kim, Chelsea Finn；数据在 HuggingFace，代码 github.com/stanford-iris-lab/ScholarCatalyst）。
- **学术定位**：信息检索（IR）基准工作 + 位置论文。它把「研究直觉/taste」重新定义为**检索任务**：给定项目早期的问题（不相似于答案），从 191k 篇 CS 论文语料中检索「催化剂论文」——被 184 位一作（207 个近期 CS 项目）标注为"曾/本可推进本项目"的论文。核心指标 Recall@20。
- **关键实测数据**（一手）：
  - 多轮 agentic search（Claude Sonnet 类）：Recall@20 ≈ **0.42**；同工具的 embedding 检索：**0.48**——**agentic 搜索反而输给了纯嵌入检索**；最强的 Claude Fable 5.1 也仅 **0.51**（且可能训练时见过已完成论文）。
  - **识别不对称**：把被漏掉的催化剂论文加入候选池后，强模型能把 Recall@5 从 ~27% 提到 ~50%——模型「认得出 > 找得到」。作者据此提出：用模型自身作 critic 的识别信号可以逐步 hill-climb 出更好的检索器（「模型当自己的批评者」）。
- **与用户假设链的对接点**：这就是「催化剂式信息是解空间跳跃关键」的**可评测学术对应物**——但它只评「找到催化剂」，不评「用催化剂跳出新解」。博客的四个引用链值得注意：Swanson 1986「undiscovered public knowledge」、de Groot/Simon 的直觉=识别、Kahneman-Klein 的直觉专长条件、以及「检索器=学会的直觉」论点。
- **张力**：ScholarCatalyst 的证据目前是「agentic 开放搜索 < 嵌入相似检索」——这与用户「开放信息获取驱动解空间自改进」的假设方向**相反**（至少在「找催化剂」这一环上，朴素 agentic 搜索还没赢）。用户的 ABCD 若声称开放获取驱动自改进，需要一个比"多轮 agent 检索"更结构化的获取机制来解释为什么它能翻盘。

### 1.2 Jane Street agent_skills / socratic-tutor

- **一手来源**：github.com/janestreet/agent_skills（MIT，2026-11 发布，11.6k+ stars），首个技能 `socratic-tutor/SKILL.md`。
- **学术定位**：**工程产物而非论文**——遵循 Anthropic/AWS 推广的 Agent Skills 文件夹格式（SKILL.md 指令文件）。socratic-tutor 把宿主模型变成"严格的私人教师"：诊断性分级、0-5 级技能地图、间隔复习、最终评估；触发词 'teach me'/'quiz me'，明确不用于一次性 'explain X'。第三方评测（mlllm.io）指出：其有效性**未经任何实证测量**（无前后测、无 baseline 对比）。
- **与用户假设链的对接点**：它是「提问划定探索空间 → 检索引入外部信息 → 识别更新判断 → 判断推动问题修改」这条链里**提问侧**的工程化样本（苏格拉底式提问=在解空间中用问题而非答案导航）。微信文章把它与 ScholarCatalyst 并列，隐含的连接是：**教学式提问与催化剂检索是同一个「引导式信息注入」模式的两种实例**。

---

## 2. 强相关候选领域（收敛性：直接供给「理论基础 + 硬技术亮点」的框架）

### A. QD 理论本身：MAP-Elites 系谱与其理论缺口

- **来源（一手）**：Mouret & Clune, "Illuminating search spaces by mapping elites"（arXiv:1504.04909, 2015，MAP-Elites 原始论文）；CVT-MAP-Elites（Vassiliades, Chatzilygeroudis & Mouret，质心 Voronoi 分箱扩展，2017/2018）；pyribs 库（github.com/icaros-usc/pyribs，QD 标准软件平台）。
- **理论分析现状（一手）**：
  - Dang, Neumann, Neumann, Opris, Sudholt, "Runtime Analysis of Quality Diversity Algorithms"（Algorithmica，doi:10.1007/s00453-024-01254-z）——首批 QD 运行时间定理：伪布尔优化 + OneMax 特征空间的 QD 行为。
  - "Theoretical Analysis of Quality Diversity Algorithms for a Classical Path Planning Problem"（arXiv:2412.11446）。
  - "Hot off the Press: Quality-Diversity Algorithms Can Provably Be Helpful for Optimization"（GECCO 2025，doi:10.1145/3712255.3734227）。
  - "Multi-objective Analysis of MAP-Elites"（arXiv:1803.05174）。
- **解释面向**：特征格/精英档案作为数学对象（覆盖性、收敛性、与优化的关系）的既有正规语言——即用户「解空间形状」假设的直接载体；且 QD 理论目前只有玩具问题级定理（OneMax、路径规划），一般收敛性结果**不存在**。
- **张力**：理论缺口=学术机会：用户有 10× 实测数据（env_1 ledger ~9.7×），恰落在「QD×LLM 为什么有效」零定理区。
- **关键事实（一手）**：AlphaEvolve 原文（arXiv:2506.13131 §2.5 Evolution）自述：evolutionary database 实现的算法「**inspired by a combination of the MAP-Elites algorithm and island-based population models**」——AlphaEvolve 就是 QD 档案 + LLM 变异的工业化实例。

### B. LLM×EC：LLM 作为进化算子

- **来源（一手）**：
  - Lehman, Clune 等, "Evolution through Large Models"（arXiv:2206.08896, 2022）——LLM 作为学习到的变异算子，产生「plausible offspring」（在训练分布内的高质量后代）。
  - Romera-Paredes et al., "Mathematical discoveries from program search with large language models"（FunSearch，Nature 2024, doi:10.1038/s41586-023-06924-6）——island 种群 + 杀死低分岛重建、LLM 采样程序；cap set 20 年来最大改进。
  - Novikov et al., "AlphaEvolve: A coding agent for scientific and algorithmic discovery"（arXiv:2506.13131, 2025）——MAP-Elites+islands 档案、prompt sampler（从档案构建提示）、evaluation cascade（按难度递增的测试级联、早停剪枝）、分布式异步流水线；4×4 复数矩阵乘 48 次乘法（Strassen 49→48）等成果。
  - Zhang, Hu, Lu, Lange, Clune, "Darwin Gödel Machine: Open-Ended Evolution of Self-Improving Agents"（arXiv:2505.22954, 2025）——agent 改写自身代码 + 保存 agent 档案（不丢弃中间体），SWE-bench 等指标持续提升。
  - Ma et al., "Eureka: Human-Level Reward Design via Coding LLMs"（arXiv:2310.12931, 2023）。
  - 综述：arXiv:2401.10034（EC in the era of LLM）；Evolutionary Computation 期刊综述（doi:10.1162/evco.a.370）。
- **解释面向**：「LLM 概率分布保证解间跳跃的最低限度确定性」在文献中的对应表述= LLM 是**学习到的 proposal distribution**（ELM 的 diff 模型表述）；「变体保持可运行/合理」是该领域的经验共识但无理论保证定理。
- **张力**：AlphaEvolve 是用户系统的最近邻（QD 档案+LLM 变异+级联评估），用户声称「agent 协作改进 vs AlphaEvolve」——差异点只能在文献已明确留白的三处：① 档案描述符固定 vs 自改进；② prompt sampler 静态 vs 预算自适应；③ 无开放文献获取 vs 催化剂检索闭环。

### C. 演化生物学：适应性地形 / 基因型网络（「流形间跳跃」的理论母体）

- **来源（一手书+论文）**：
  - Andreas Wagner, "The Origins of Evolutionary Innovations"（OUP 2011）；综述：Wagner, "The role of robustness in phenotypic adaptation and innovation"（PMC3282381）；"The molecular origins of evolutionary innovations"（Trends in Genetics 2011）——**基因型网络**：同一表型的大连通基因型集合（代谢网络/调控回路/大分子三类系统皆验证）；种群沿网络漂移即可触及罕见表型邻域→创新=在穿越点跳入新表型。
  - Schuster & Fontana, "Chance and Necessity in Evolution: Lessons from RNA"（arXiv:physics/9811037）；Fontana/Schuster RNA 折叠图研究——**shape space covering**：任意序列的小邻域即可实现所有典型形状；中性网络漫游 + 罕见换形点。
  - Kauffman & Levin / Kauffman & Weinberger, "Towards a general theory of adaptive walks on rugged landscapes"（J. theor. Biol. 1987）+ NK 模型——tunable ruggedness、adaptive walk 局部最优；ruggedness=为什么必须「跳跃」而非爬山。
- **解释面向**：用户「低维流形间跳跃找更优解」在演化论中有 30 年成熟对应物：**沿中性网络（≈流形）漂移 + 在穿越点换表型（≈跳流形）**；「催化剂」的生物对应物=跨越网络边界的穿越点基因型。robustness↔innovation 理论正好回答「为什么保留次优解（中性）反而加速创新」——即 QD 精英格的生物学辩护。
- **张力**：生物学的「流形」是高维基因型空间中占极小体积的稀疏连通结构，不是欧氏低维流形——「低维」措辞与文献对接需要转译。

### D. 表示几何 / 流形假设 / 信息几何（数学根基 + 反例）

- **流形假设的形式化（一手）**：Fefferman, Liu, Wong, "Testing the Manifold Hypothesis"（arXiv:1310.0425, 2013）——「高维数据落在低维流形附近」的可测试化与复杂度保证。
- **实证支持（一手）**：Ansuini et al., "Intrinsic dimension of data representations in deep neural networks"（arXiv:1905.12784, NeurIPS 2019）——训练后网络各层内在维数（ID）远小于宽度，且**先升后降**（钟形）。
- **实证反例（一手）**："Token embeddings violate the manifold hypothesis"（NeurIPS 2025, dl.acm.org/doi/10.52202/085713-4929）——token 嵌入邻域**不满足**「局部平坦光滑」的流形零假设；"Hardness of Learning Neural Networks under the Manifold Hypothesis"（arXiv:2406.01461）。
- **LLM 语义几何（一手）**：
  - Huh et al., "The Platonic Representation Hypothesis"（ICML 2024 position）——不同模型/模态的表示随规模**收敛**到共享世界模型（→ 用户「跨域同构」在表示层的一手证据）。
  - Park et al., "The Linear Representation Hypothesis and the Geometry of LLMs"（ICML 2024，反事实形式化）；"The Geometry of Categorical and Hierarchical Concepts in LLMs"（arXiv:2406.01506，层级概念呈低维锥/单形）；Karkada, "Symmetry in language statistics shapes the geometry of model representations"（ICML 2026，月份→圆、年份→一维流形等结构源于语言统计的平移对称）。
- **信息几何（一手经典）**：Amari, "Natural Gradient Works Efficiently in Learning"（Neural Computation 1998, doi:10.1162/089976698300017746）；Ollivier, Auger, Hansen 等, "Information-Geometric Optimization Algorithms: A Unifying Picture via Invariance Principles"（JMLR 2017）——**把进化算法统一为概率分布流形上的自然梯度上升**：这本身就是「进化=分布在几何空间上的移动」的严格数学框架。
- **代码几何（一手新证）**："The Natural Geometry of Code: Hyperbolic Representation Learning for Program Reasoning"（ICLR 2026）——代码的树状分层结构对应**双曲几何**（指数体积增长），欧氏嵌入系统性失真；"Lost in the Latent Space: How Code Embeddings Represent Functional Intent and Implementation Fidelity"（arXiv:2610.39519，14 个 code embedding 模型实证）。
- **解释面向**：「自然语言=语义空间低维流形」的数学根基（流形假设+ID 实证+信息几何）；「code=离散低维流形」的文献现状（几何可能是双曲而非欧氏低维——这是用户假设链第 1-2 环最需要小心的转译点）。

### E. 催化剂信息的学术对应物：检索增强发现 × 类比 × 文献型发现

- **ScholarCatalyst**（§1.1 详述）——「催化剂论文检索」已被形式化为可评测任务（Recall@20）。
- **文献型发现 LBD（一手经典）**：Don Swanson, "Undiscovered Public Knowledge"（Library Quarterly 56(2), 1986）——**ABC 模型**：文献 A-B 与 B-C 分立存在而 A-C 无人连接，连接即新知识（经典案例：鱼油↔血液粘度↔雷诺氏征）；Swanson, "Medical literature as a potential source of new knowledge"（PMC225324）；Smalheiser, "Literature-Based Discovery: Beyond the ABCs"（2012）；Arrowsmith 系统。
- **类比计算化（一手经典）**：Gentner, "Structure-Mapping: A Theoretical Framework for Analogy"（1983，系统性原则：优先映射关系系统而非对象属性）；Falkenhainer, Forbus, Gentner, "The Structure-Mapping Engine"（1989，SME）；SMT 四十年综述（doi:10.1177/09637214251395678）。
- **科学发现现场证据（一手）**：Kevin Dunbar 的 in vivo 实验室研究（"How scientists think", 1997；Trends in Cognitive Sciences 2001）——分子生物学实验室会议中：局部类比（同领域）维持假说、跨领域类比改变假说、异常证据驱动假说丢弃。
- **解释面向**：「跨域同构、认知可迁移」= SMT 的关系结构映射（同构=共享关系谓词系统）；「催化剂式信息」= LBD 的桥接节点 B；ScholarCatalyst = 该能力的第一个可评测基准。用户架构的差异化空间：文献全部只测「**找到**催化剂」，无一测「**用**催化剂改变搜索行为」。

### F. 开放式进化 / 自指改进（「解空间自改进」的理论语境）

- **来源（一手）**：
  - Schmidhuber, "A family of Gödel machines"（2003, cs.LO/0309048）——自指+证明搜索的全局最优自改进（理论极限）。
  - Darwin Gödel Machine（arXiv:2505.22954，§B）——可运行的自指改进（改自己的代码+档案）。
  - Wang, Lehman, Clune, POET（arXiv:1901.01753, 2019）——问题与解**共进化**：环境生成器+解优化器+最小生成标准+**跨环境迁移**（迁移=POET 中产生跨地形跳跃的机制）；Clune 的 AI-GAs（AI-generating algorithms, 2019）三支柱。
  - Lehman & Stanley, novelty search（"Abandoning Objectives: Evolution Through the Search for Novelty Alone", Evolutionary Computation 2011；Why Greatness Cannot Be Planned, 2015）——目标悖论：重大发现常是寻找别的东西的副产品。
- **解释面向**：「解空间自改进」在文献中有三个已命名机制可对应：① 共进化（POET：问题侧也进化）② 自指（DGM：改进器本身被改进）③ 目标漂移。POET 的「迁移」与用户「催化剂式信息触发跳跃」结构同源。

### G. 预算分配 / 自适应算子选择（「token 分配到特征格」的现成框架）

- **自适应算子选择 AOS（一手）**：Fialho, Schoenauer, Sebag 等（"Dynamic Multi-Armed Bandits and Extreme Value-based Rewards for Adaptive Operator Selection in EA"，MSR-INRIA）；DaCosta 等 bandit-AOS 分析（Annals of Mathematics and AI, doi:10.1007/s10472-010-9213-y）；AOS 综述（IEEE TAI 2025, doi:10.1109/tai.2025.3545792）；selection hyper-heuristics（EJOR, doi:10.1016/j.ejor.2017.01.042）——每个算子=一个 arm，按信用分配（credit assignment）动态选。
- **连续空间的树搜索预算（一手）**：Couëtoux, Hoock, Sokolovska, Teytaud, Bonnard, "Continuous Upper Confidence Trees"（LION 2011, doi:10.1007/978-3-642-25566-3_32）——progressive widening / double progressive widening：MCTS 在连续/无限动作空间中「节点访问越多、展开的分支越多」——按证据增量分配探索预算的机制。
- **预算化早停（一手）**：Li, Jamieson 等, "Hyperband"（arXiv:1603.06560, JMLR 2018）——任意预算下的自适应资源分配。
- **LLM 侧（一手）**：Snell et al., "Scaling LLM Test-Time Compute Optimally can be More Effective than Scaling Model Parameters"（arXiv:2408.03314, ICLR 2025）——测试时算力的最优分配与难度自适应；Zhou et al., LATS（arXiv:2310.04406, ICML 2024）——LLM agent×MCTS 统一推理-行动-规划。
- **解释面向**：「有限 token 分到最可能产生跳跃解的特征格」有三个现成数学容器：MAB/AOS（格=arm）、progressive widening（格=树节点、token=visit）、BO/主动学习式 acquisition（格=候选点、token=评估预算）。AlphaEvolve 的 evaluation cascade 是这个问题的**评估侧**粗糙工程版（只做难度级联，不做格间分配）。

---

## 3. 弱相关候选领域（发散性：提供创造性视角的远端领域）

### H. 科学学 / 重组式创新
- Weitzman, "Recombinant Growth"（Quarterly Journal of Economics, 1998）——经济思想史的数学模型：创新=现有思想的重组搜索；Kauffman 的 **adjacent possible**（"Investigations", 2000；"At Home in the Universe"）——可实现的下一步只取决于当前实现的边界（催化剂=把某格从 adjacent possible 变为 actual 的事件）；Uzzi et al., "Atypical Combinations and Scientific Impact"（Science, 2013）——引用组合的常规性/非常规性与高影响的关系。
- **解释面向**：为什么「跨域同构解」在宏观统计上有效——重组创新与相邻可能的统计规律；为「催化剂式信息」提供经济学/社会学尺度（而非个体认知尺度）的辩护。

### I. 算法信息论 / 压缩视角
- Solomonoff 归纳（1964）；Delétang et al., "Language Modeling Is Compression"（arXiv:2309.10668, ICLR 2024）——LLM 分布=通用压缩器=对任意数据（含程序）的隐式先验。
- **解释面向**：「LLM 概率分布保证跳跃的最低限度确定性」可翻译为 AIT 语言：LLM=可计算的程序先验；「跳跃落在可运行解流形上」=从先验质量高的区域采样；FunSearch/AlphaEvolve=程序空间上的近似 MDL 搜索。这是把用户直觉链第一条（流形）与第三条（LLM 确定性）焊接起来的备选数学语言。

### J. 认知科学：类比迁移 / 概念流体 / 直觉
- Gick & Holyoak（1980/1983，Dunker 辐射问题的类比迁移实验）；Hofstadter & Mitchell, Copycat（"Fluid Concepts and Creative Analogies", 1995）——概念空间中的「流体滑移」=跳跃算子的认知模型；de Groot / Simon（国际象棋大师直觉=识别：搜索量相同、搜索质量不同——ScholarCatalyst 博客的立论基础）；Kahneman & Klein, "Conditions for Intuitive Expertise"（2009）——直觉专长成立的两个条件：可靠线索+即时反馈（ScholarCatalyst 据此提出「识别不对称」自举训练回路）。
- **解释面向**：「跳跃算子」的认知模型（滑移/再表征）+「直觉=识别」如何被形式化为检索器——用户架构中「提问划定探索空间」一环的认知科学基础。

### K. 苏格拉底教学法 / 智能辅导系统
- Jane Street socratic-tutor（§1.2）——诊断分级、0-5 级技能地图、间隔复习的工程化；其有效性未测量（第三方 mlllm.io 指出无前后测）。
- **解释面向**：与 ScholarCatalyst 并列的另一半——「用问题而非答案导航探索空间」的工程先例；其未测量状态=用户可以做实证差异化贡献的位置。

### L. 自催化化学 / 生命起源（「催化剂」隐喻的数学本源）
- Kauffman, "Autocatalytic Sets of Proteins"（J. theor. Biol., 1986；The Origins of Order, 1993）——催化剂集合自维持并逐步扩张相邻可能；近代形式化（Hordijk & Steel 的 RAF 理论）。
- **解释面向**：用户借用「催化剂」一词的数学本源：自催化集合=「使解空间自身可扩展的信息集合」的化学先例；ABCD 的 D（解空间自改进）与 autocatalytic set 的结构同源性是一个可写进 related work 的美学连接（但注意它是隐喻层而非机制层）。

---

## 4. 跨领域张力清单（材料，非结论）

1. **AlphaEvolve 已经是 QD×LLM**：原文自述 evolutionary database = "MAP-Elites + island-based population models" 组合 + LLM 变异 + 级联评估。用户「改进 AlphaEvolve」的论证必须钉在文献已明确留白的三处之一/多处：催化剂检索闭环（AlphaEvolve 无开放文献获取）、档案格预算自适应（其 prompt sampler 静态、cascade 只在评估侧）、descriptor 自改进（其特征格固定）。
2. **开放获取当前证据为负**：ScholarCatalyst 实测 agentic search 0.42 < embedding 0.48（同工具下）——「开放信息获取驱动自改进」不能默认成立；但「识别不对称」（漏掉的催化剂加入池后 Recall@5 27%→50%）提供了用模型自身作 critic 的自举回路——用户架构若做成该回路的工程化，恰是对这篇论文方向假设的验证性实验。
3. **「低维流形」假设双向证据并存**：支持——深度网络表示内在维数远小于宽度（Ansuini 2019）、语义几何呈低维圆/锥/单形结构（Park/Karkada）；反对——token embeddings 被证违反流形假设（NeurIPS 2025）、代码自然几何是双曲而非欧氏（ICLR 2026）。用户需要决定第 1-2 环假设采纳哪种几何语言（低维欧氏 / 非欧低维 / 分层稀疏连通结构）。
4. **QD 理论恰好在玩具阶段**：OneMax 特征空间 + 路径规划才有定理（Algorithmica 2024；arXiv:2412.11446）；"QD provably helpful for optimization"（GECCO 2025）刚出现。用户的 10× 实测是理论空白区的经验数据——机会（可提定理级命题）与风险（没有现成收敛定理可引用）并存。
5. **Wagner 基因型网络已把「沿流形漂移+穿越点跳新流形」讲了 30 年**：robustness↔innovation 理论+三类分子系统实验验证。用户假设链第 2-4 环的「理论缺基础」在生物学文献中其实有成熟母体——问题从「没有理论」变成「该引用/转译/扩展哪个理论」。
6. **Swanson 的 ABC（1986）**：文献型发现的 ABC 模型与用户的 ABCD 架构命名纯属巧合，但内容同构（B=桥接节点=催化剂）——40 年先例可援引。

## 5. 引导性问题（阶段 2）

把假设链的五个环节逐环映射到上面候选领域之后——每一环现在对你而言属于三种状态中的哪一种：**已有定理可引用**（如流形假设的可测试性、AOS 的 bandit 保证）、**只有经验证据的猜想**（如 LLM 变异的后代合理性、QD 在真实问题上的优势）、**无人提出过的命题**（如「催化剂信息改变解空间形状」的任何形式化）？——你的 ABCD 的「理论基础」应当落在哪一环的哪个领域，而「硬技术亮点」又应当落在哪一环的哪处留白？

---

## 6. 来源索引（分层）

**一手（原始论文/博客/代码库）**：Stanford AI Lab Blog "Research Intuition as a Retrieval Task"（ai.stanford.edu/blog/scholarcatalyst/, 2026-11-09）；ScholarCatalyst（arXiv:2610.02202；github.com/stanford-iris-lab/ScholarCatalyst；HF 数据集）；janestreet/agent_skills socratic-tutor（github.com/janestreet/agent_skills，MIT）；Mouret & Clune MAP-Elites（arXiv:1504.04909）；AlphaEvolve（arXiv:2506.13131，§2.5 Evolution 原文核验）；FunSearch（Nature, doi:10.1038/s41586-023-06924-6）；ELM（arXiv:2206.08896）；Darwin Gödel Machine（arXiv:2505.22954）；Eureka（arXiv:2310.12931）；POET（arXiv:1901.01753）；Gödel Machine（cs.LO/0309048）；Runtime Analysis of QD（Algorithmica, doi:10.1007/s00453-024-01254-z）；QD path planning theory（arXiv:2412.11446）；QD provably helpful（GECCO 2025, doi:10.1145/3712255.3734227）；Fefferman et al.（arXiv:1310.0425）；Ansuini et al.（arXiv:1905.12784）；Token embeddings violate manifold hypothesis（dl.acm.org/doi/10.52202/085713-4929）；Platonic Representation Hypothesis（ICML 2024）；Park et al. LRH（ICML 2024）/ 概念几何（arXiv:2406.01506）；Karkada（ICML 2026）；Hyperbolic code geometry（ICLR 2026）；Lost in the Latent Space（arXiv:2610.39519）；Amari 自然梯度（Neural Computation 1998）；IGO（JMLR 2017）；Swanson 1986（Library Quarterly 56(2)）；SME（Falkenhainer/Forbus/Gentner 1989）；Dunbar in vivo（Trends in Cognitive Sciences 2001）；Couëtoux et al. Continuous UCT（LION 2011）；Hyperband（arXiv:1603.06560）；Snell et al.（arXiv:2408.03314）；LATS（arXiv:2310.04406）；Delétang et al.（arXiv:2309.10668）；Wagner 2011（OUP）；Schuster/Fontana（arXiv:physics/9811037）；Kauffman（J. theor. Biol. 1987, 1986）；Weitzman 1998（QJE）。

**权威二手（核验用）**：GECCO/Nature 论文官方博客（deepmind.google AlphaEvolve/FunSearch）；AOS 综述（IEEE TAI 2025, doi:10.1109/tai.2025.3545792）；EC×LLM 综述（arXiv:2401.10034; doi:10.1162/evco.a.370）；SMT 四十年综述（doi:10.1177/09637214251395678）；LBD 回顾（PMC5771422）。

**标注说明**：微信文章原文（mp.weixin.qq.com/s/7ikOnMPLPsPuK2VFGrahCw）抓取失败，内容依据 main agent 已核验摘要 + 斯坦福博客一手原文交叉确认；Weitzman/Uzzi/Gick&Holyoak/Hofstadter/RAF/Kauffman1986/1987/Solomonoff/Clune2019/novelty search 为经典文献引用（未在线核验原文，但均为领域奠基性文献）。
