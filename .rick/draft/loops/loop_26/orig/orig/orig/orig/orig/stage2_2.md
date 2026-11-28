# Stage 2 · 第 2 轮深挖调研素材（orig）

- 阶段：视角生产（阶段 2，第 2 轮）
- 任务：三领域（信息几何/IGO、Wagner 基因型网络、AIT/压缩）深挖 + 交叉空档，对准缺口「每个领域为何强/弱、分别支撑假设链哪一环、如何共同支撑『检索穿越点』亮点」
- 假设链五环：①语言/代码=语义空间低维流形 → ②进化=流形间跳跃找更优解 → ③LLM 概率分布保证跳跃最低限度确定性 → ④跨域同构、催化剂式信息是跳跃关键 → ⑤token 资源分配到最可能跳跃的特征格=解空间形状自改进

---

## A. 信息几何 / IGO

### A1. 可引用定理层

**Ollivier, Arnold, Auger, Hansen, "Information-Geometric Optimization Algorithms: A Unifying Picture via Invariance Principles", JMLR 18(18):1–65, 2017**（https://jmlr.org/papers/v18/14-467.html）
- 核心构造：把任意光滑参数化分布族 {p_θ} 变成 X 上的黑盒优化方法——期望适应度的**自然梯度上升**（Fisher 信息度量下的最陡方向）。
- **三重不变性**（IGO 的设计原则，定理化）：对目标函数的单调变换不变（通过分位数/排名适应度实现）、对搜索空间重参数化不变、对分布族重参数化不变。
- 统一范围：CMA-ES、交叉熵方法（CEM）、多数 EDA 可看作「分布族限制在指数族上时 IGO 流的近似实现」。来源：JMLR 原文 + Hansen 的 IGO 教学讲义（http://www.cmap.polytechnique.fr/~anne.auger/teaching-slides/IGO-2022-slides.pdf，注明「CMA-ES 的某些部分未被 IGO 框架解释」）。
- **注意（对用户重要）**：IGO 的自然梯度流是**单一分布族内部**的上升流——「换族/换支撑」不在其框架内。讲义明确承认 CMA-ES 有些步骤（如协方差矩阵更新中的累积/步长控制）IGO 无法完全解释。

**Akimoto, "Theoretical Foundation for CMA-ES from Information Geometry Perspective", arXiv:1206.0730**（https://arxiv.org/abs/1206.0730）
- 在 Fisher 度量概率流形上重推 CMA-ES：均值更新 = 期望适应度的自然梯度；协方差更新 = 自然梯度的近似。可引用为「CMA-ES = 高斯流形上的自然梯度」的严格出处。

**Amari 信息几何基础（可直接引用的定理）**
- Chentsov 定理：Fisher 信息度量是统计流形上唯一（差常数倍）在充分统计量下不变的度量——「为什么是 Fisher 度量」的唯一性论证。
- Amari 1998 "Natural gradient works efficiently in learning"（Neural Computation）：自然梯度 = Fisher 度量下最陡上升；渐近有效性（达到 Cramér–Rao 界）。
- Amari & Nagaoka 2000《Methods of Information Geometry》：指数族的对偶平坦结构（e-联络/m-联络、Bregman 散度=KL）——「分布族的几何」标准工具箱。
- 相关补充：ICML 2009 "Stochastic Search using the Natural Gradient"；JMLR 2020 "New Insights and Perspectives on the Natural Gradient Method"（https://www.jmlr.org/papers/volume21/17-678/17-678.html）。

### A2. 经验证据层（QD / LLM-进化系统：有几何机器，无 Fisher 论述）

- **MAP-Elites**（Mouret & Clune 2015, arXiv:1504.04909）：档案 = 行为描述子空间上的网格细分，每格存最优个体——产出是「解的集合/地图」，不是分布；无度量结构。
- **DQD**（Fontaine et al., NeurIPS 2021, arXiv:2106.03894）：把目标函数与 measure/描述子函数都当作可微——MEGA 在「目标×描述子」联合空间上构造梯度树——描述子空间上确实存在梯度机器，但全文无 Fisher/信息几何论述。
- **FunSearch**（Romera-Paredes et al., Nature 2024, s41586-023-06924-6）：LLM = 以「最优程序 + 多样程序」构造的 prompt 为条件的 proposal 分布；island 模型维持多样性；程序数据库=档案。整套=「LLM 条件采样 + 精英档案」的 EDA，无几何。
- **AlphaEvolve**（DeepMind 2025, arXiv:2506.13131）：程序数据库 + prompt sampler + LLM 集成（Flash=快速变异、Pro=深变异）+ 自动评估器——「prompt sampler 检索档案喂 LLM」就是其核心循环；无任何几何/分布族论述。
- **ELM**（Große et al. 2022, arXiv:2206.08896）：LLM 当变异算子用于 GP；可对 LLM 变异算子做微调。
- **LMX**（Meyerson et al. 2023, arXiv:2302.12170）：LLM few-shot prompting = 类交叉算子——「语言模型天然实现类交叉的变异」。
- **LLM_GP**（arXiv:2401.07102）：形式化的 LLM 基进化算法，算子设计偏离经典 GP。

**小结（A 域分层）**：定理层 = IGO/自然梯度（含 Akimoto 2012 对 CMA-ES 的严格化）；经验层 = QD 档案、CMA-ME、LLM-进化系统的全部实践；**无人提出** = 把 LLM 条件分布族当 IGO 分布族做自然梯度、QD 档案的 Fisher 度量化。

---

## B. Wagner 基因型网络 / 中性网络

### B1. 可引用定量结果（RNA GP 映射上的组合/统计规律——非普适定理）

- **Schuster, Fontana et al. 1994 "From sequences to shapes and back"**（Proc R Soc B, doi:10.1098/rspb.1994.0040）：结构频率服从广义 Zipf 律（少数常见结构+大量罕见结构）；同结构序列随机散布于全序列空间；**shape space covering**：所有常见结构可在任一随机序列的小 Hamming 邻域内触及。
- 覆盖半径数量级：AUGC n=100 时，随机序列到任一常见结构中性网络的距离约 15 ≪ 序列空间半径 n/2=50（Schuster & Fontana, "Modelling evo-devo with RNA", SFI working paper）。
- **Reidys, Stadler, Schuster 1997**（Bull Math Biol, doi:10.1007/bf02462007）：中性网络作为组合映射对象的数学性质（连通性/巨分量）。
- **Huynen, Stadler, Fontana 1996 "Smoothness within ruggedness"**（PNAS 93:397）：种群在中性网络上分裂成亚群独立扩散；基因型信息丢失阈值与表型信息丢失阈值分离；**中性网络不同位置的可及新表型集合不同**（穿越点位置依赖的直接证据）。
- **van Nimwegen, Crutchfield, Huynen 1999 "Neutral evolution of mutational robustness"**（PNAS 96:9716，**定理级**）：种群在中性网络上的极限分布 = 邻接矩阵主特征向量（只由拓扑决定）→ 中性漂移自发聚到高稳健度区域。配套 "Metastable Evolutionary Dynamics"（arXiv adap-org/9907002）：形式化对比「跨适应度屏障」vs「沿中性路径逃逸」两种相变。
- 跨映射对比（Biophys J 2012, HP25 蛋白 vs GC25 RNA）：蛋白每结构平均 7.1 条序列 vs RNA 每结构 >1000 条——中性网络规模随 GP 映射差数量级。
- 可塑性库测量（Biophys J 2014 "Mutational Robustness Accelerates the Origin of Novel RNA Phenotypes through Phenotypic Plasticity"）：1-邻域的可及表型库 ≈ 基因型自身的 8 倍（平均 525.7 个独特表型）——「漂移一步，可及新表型暴涨」的定量。
- 总纲：**Andreas Wagner 2011《The Origins of Evolutionary Innovations》(Oxford UP)**：稳健性创造可进化性——中性网络大→稳健；网络铺展→可及新表型集合大→创新（穿越）机会多。

### B2. 「预测/检索穿越点」的最近先例（B×C 桥）

- **Schaper & Louis 2014 "The Arrival of the Frequent"**（PLoS ONE, e86635）：变体到达顺序由 GP 映射局部表型频率决定——表型偏差能把种群「引向」特定局部最优。
- **Dingle et al. 2018 "Input–output maps are strongly biased towards simple outputs"**：随机输入产生输出 x 的先验概率随 x 的描述复杂度指数衰减（编码定理型上界）。
- **Johnston, Dingle, Greenbury, Camargo, Doye, Ahnert, Louis 2022 "Symmetry and simplicity spontaneously emerge from the algorithmic nature of evolution"**（PNAS 119(11):e2113883119）：AIT 编码定理 → 对称/简单结构在变异下先验概率指数更高 → 无适应性演化也系统偏向简单表型（用 RNA/蛋白 HP/多面体 GP 映射验证）。
- **"Predicting phenotype transition probabilities via conditional algorithmic probability approximations"**（J R Soc Interface 2023, rsif.2022.0694）：用条件算法概率近似**预测表型跃迁概率**——目前最接近「预测穿越点」的文献（细节见下）。
- **无人提出**：把上述规律组装成「主动检索穿越点的技术」（即不是被动统计偏差，而是可查询的检索器）——文献中未见。

---

## C. AIT / 压缩

### C1. 可引用定理层

- **Solomonoff 完备性/支配性**（Solomonoff 1978, "Complexity-Based Induction Systems: Comparisons and Convergence Theorems", raysolomonoff.com/publications/solo1.pdf）：对任意通用 UIO 机 M 与任意可计算概率测度 P，存在常数 k=P 的描述复杂度量级，使得对所有串 Ph(x(n)) > 2^{-k}·P(x(n))——**通用先验支配一切可计算先验**（差一个与目标相关的常数因子）。
- **Solomonoff 收敛**（同上 + Hutter 讲义 hutter1.net/rsise/2017/slides4.pdf）：对未知真实分布 μ 的序列预测，ε-偏差期望次数有界，界常数 c = ln2·K(μ)——用通用先验的累计预测误差被目标过程的 Kolmogorov 复杂度控制。进一步精细化的界见 Chernov–Hutter–Schmidhuber, "Algorithmic Complexity Bounds on Future Prediction Errors"（arXiv:cs/0701120）。
- **Levin 编码定理**（标准出处 Li & Vitányi《Kolmogorov Complexity and Its Applications》第4章）：−log m(x) = K(x) + O(1)——通用先验质量 = 描述长度的直接换算；「从先验采样」与「偏向短程序」严格等价。
- **MDL/极小极大 regret**（Shtarkov 1987 NML；Rissanen MDL 系列）：任何通用码的 regret 下界；「预测器=码」的 prequential 视角（Barron–Rissanen–Yu 1998）。
- **Chaitin 元生物学**（"A Mathematical Theory of Evolution and Biological Creativity"；"Life as Evolving Software"）：演化 = **软件空间中的随机爬山**；对比定理：随机变异 vs 智能选择变异（oracle 引导的 "intelligent design"）在软件空间可指数级快——「往哪跳」的差别被形式化为指数级（模型 A/B 定理，busy beaver/Omega 界）。
- **Valiant 可进化性**（Evolvability, JACM 2009）：进化 = 受限 PAC 学习的形式模型——「进化=学习」已有严格形式化先例。

### C2. LM-Is-Compression 机制细节（Delétang et al., ICLR 2024, arXiv:2309.10668）

- 机制：算术编码以预测分布 ρ 构造码，长度 ≈ −log ρ(x_{1:n})+1 bit；预测好 ⟺ 压缩好（log-loss 意义）。
- 实测：**Chinchilla 70B（纯文本训练）把 ImageNet patch 压到 43.4%（PNG 58.5%）；LibriSpeech 压到 16.4%（FLAC 30.3%）**——跨模态通用压缩器。
- 论文主张：压缩视角统一解释 scaling laws、tokenization、in-context learning；任何压缩器（如 gzip）反过来可当条件生成模型。
- 关联：LLM 条件分布 = 数据上的**隐式先验**；−log P_LLM(x) = x 在该先验下的码长——「LLM 给的变异先验质量」可直接读作码长。

### C3. 压缩 × 进化搜索 / 新颖性

- **NCD**（Cilibrasi & Vitányi, "Clustering by Compression", IEEE Trans. IT 2005）：归一化压缩距离 = 参数无关的通用相似度；理论基础是归一化信息距离（NID）——注意 NID 本身**不可半计算**，NCD 是其可计算近似（arXiv:1110.4544 "Compression-based Similarity"）。→ 压缩距离做新颖度/异常检测有定理级基础。
- **压缩进度理论**（Schmidhuber 2009 "Driven by Compression Progress..."）：数据在观察者学会更好地压缩它时「暂时有趣」→ 新颖性/兴趣 = 压缩进度的一阶导数——**内在动机的形式理论**；2026 有续作 "Position: Interestingness is an Inductive Heuristic for Future Compression Progress"（PMLR v306）。
- **Novelty search**（Lehman & Stanley 2011, "Abandoning Objectives", Evolutionary Computation 19(2)）：弃目标、搜新颖；新颖度 = 到档案的 k-近邻平均距离——「搜索穿越点」的既有工程，但度量是启发式的。
- **档案几何的最近尝试**（Salehi, GECCO'22 Companion, arXiv:2205.03162）：档案的作用不是单纯防回溯，而是**抵消行为空间度量的畸变/行为映射非线性带来的探索偏差**——档案=对扭曲几何的修正（分析+低维示例，非定理）。

---

## D. 交叉空档地图

### D1. IGO × LLM（LLM 当 proposal 分布的几何优化）
已有人做：
- LLM 当变异/交叉/优化器：FunSearch（Nature 2024, s41586-023-06924-6）、AlphaEvolve（arXiv:2506.13131）、ELM（arXiv:2206.08896）、LMX（arXiv:2302.12170）、LLM_GP（arXiv:2401.07102）、QDAIF（ICLR 2024）、OPRO（ICLR 2024, arXiv:2309.03409）。
- 适应 proposal 分布的手段：档案条件化（prompt=精英+多样样本）；分布匹配微调（GFlowNet-LLM，ICLR 2024，arXiv:2310.04363——按后验采样重写 LLM 分布）。
- 训练侧几何：结构化 Fisher 近似优化器（arXiv:2502.07752）；ENIGMA position（arXiv:2510.11278，政策=信息流形方向）；ICL=梯度下降定理化（von Oswald et al. ICML 2023 等）。
空档：把 {P_LLM(·|context)}（context 为参数）当 IGO 分布族做自然梯度；QD 档案的 Fisher 度量化——未见文献。

### D2. 中性网络 × 代码空间
已有人做：
- GP 中性经验：非有效代码提供中性（Banzhaf 组线性 GP，cs.mun.ca GPTP 系列）；定量研究（GECCO 2006, p895）；综述（Galván-López, Evolving Systems 2011）。
- 可微程序（NN 权重空间）：flat minima=大连通近常误差区域（Hochreiter & Schmidhuber, Neural Computation 1997——其论证恰是 MDL）；mode connectivity（Garipov et al. 2018 / Draxler et al. 2018 方向）；Git Re-Basin（arXiv:2209.04836：排列对称性→近似单盆地、线性连通）；低损空间连续全连通（arXiv:2505.02604, 2025）。
- GP 映射方法迁移先例：量子线路（arXiv:2301.01847）；蛋白结构（bioRxiv 2026.03）。
- 形式障碍：程序等价不可判定 → 等价变异自动检测不可能（Budd & Angluin；转引自 Jia & Harman 综述，mutationtesting.uni.lu TR-09-06）。
空档：离散程序空间中性网络的系统理论；LLM 编辑空间（重构/语义等价编辑）的中性度测量——未见。

### D3. 检索 × 穿越点（催化剂=流形间桥接的形式化）
已有人做：
- 生物侧：条件算法概率→表型跃迁概率预测/上界（J R Soc Interface 19:20220694, 2022；22:20240916, 2025）；表型偏差/arrival of the frequent（PLoS ONE 2014）。
- 工程侧：novelty search 档案（Lehman & Stanley, Evol. Comput. 2011）；档案格选择=多臂老虎机（Monte Carlo Elites, arXiv:2104.08781）；AlphaEvolve prompt sampler / FunSearch islands = 档案检索器现役实现；transfer optimization（MFEA/MFEA-II, IEEE TEVC 2016/2019；Gupta & Ong 综述 TEVC 2017）。
- 认知侧：SME 结构映射（Falkenhainer–Forbus–Gentner 1989, AAAI）；概念整合计算模型（Fauconnier–Turner / Goguen 传统）。
空档：用 LLM 压缩先验当条件复杂度的可计算代理、在程序空间查询「跃迁概率被压低」的位置（=穿越点检索）、催化剂形式化为「压低目标条件码长的信息」——三件拼图从未被接通。

## E. 五环 × 三领域（原料映射：每环各自拿到的是哪一层）

| 环 | 定理层 | 经验层 | 空白 |
|---|---|---|---|
| ① 代码=低维流形 | 无（流形假设本身未证，只有检验方法） | 自然软件（ICSE 2012）；代码嵌入/语义检索；Platonic 表示收敛（Huh et al., ICML 2024）；NN 低损空间连通（2025） | 代码语义流形定理化 |
| ② 进化=流形间跳跃 | IGO=分布流形内自然梯度（JMLR 2017）；中性网络极限分布=主特征向量（PNAS 1999）；Chaitin「跳哪=指数差别」 | RNA 覆盖/邻域/位置依赖/可塑性 8×（1994–2014） | 「流形」双义未统一（分布参数流形 vs 表型连通集） |
| ③ LLM 先验保证最低确定性 | 只对通用先验成立：支配性/编码定理/收敛界（Solomonoff 1978；Levin；Chernov-Hutter-Schmidhuber） | LM-is-compression 跨模态压缩（43.4%/16.4%，ICLR 2024）；压缩-能力相关（arXiv:2406.14171） | LLM 版支配性无定理——链上最依赖类比的一环 |
| ④ 催化剂/跨域同构=关键 | 条件复杂度→跃迁概率界（J R Soc Interface 2022/2025） | 表型偏差（简单表型优先）；transfer optimization；SME/概念整合 | 催化剂 AIT 形式化（K(x|当前+催化剂)≪K(x)）无人写 |
| ⑤ token→最可能跳跃的格 | 无 | AlphaEvolve prompt sampler；FunSearch islands；Monte Carlo Elites bandit | 「解空间形状自改进」无人主张过 |

## F. 张力与引导性问题（交付原样内容）

张力：
1. 两个「流形」：IGO 的流形=分布参数统计流形（点=分布）；Wagner 的中性网络=同表型基因型连通集（点=个体/序列）。「穿越点」只能定义在其中一个上，且两者的可计算性完全不同。
2. 不可判定墙：程序等价不可判定（Budd & Angluin）+ NID 不可半计算 → 「穿越点」不可能被精确检索，只能用可计算代理量（NCD、条件复杂度近似、LLM 码长）——三领域恰在「代理量」上咬合：LLM=不可计算先验的可计算代理。
3. 偏差 vs 催化剂：无条件跃迁被偏差捕获（跳到的大多是简单表型，Johnston 2022）；催化剂要成立必须以「压低条件复杂度」起效——这正是「检索」的定义口。
4. 差异化压力：AlphaEvolve prompt sampler 已是「档案检索喂 LLM」；亮点若只落在「检索」不够，必须落在「穿越点」（按跃迁概率而非分数/多样性检索）；若穿越点=表型连通结构的边，AlphaEvolve 完全没做（无跃迁概率概念）。

引导性问题：见正文（填空式主张模板 + 强/弱相关是否维持）。
