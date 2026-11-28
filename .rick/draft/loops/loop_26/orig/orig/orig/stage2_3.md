# Stage 2-3 视角生产·第3轮再启发 调研素材（orig agent）

任务：围绕用户视角句「在精英解组成的稀疏低维流形空间上，以命中流形穿越点的代理量为可计算指标，检索使解的复杂度被压低的过程」的四处缺口（①代理量循环定义 ②复杂度→穿越概率的因果链 ③无条件 vs 条件复杂度 ④四空位文献划界 + 三领域标注是否换成流形内/流形间轴），按概念跳跃法检索。

**线索总表（持续更新）：**

| 概念 | 最相关一手来源 | 对哪个缺口 |
|---|---|---|
| 条件算法概率预测表型跃迁概率 | Dingle, Novev, Ahnert, Louis, "Predicting phenotype transition probabilities via conditional algorithmic probability approximations", J R Soc Interface 19(197):20220694 (2022). https://doi.org/10.1098/rsif.2022.0694 ；biorxiv PDF: https://www.biorxiv.org/content/10.1101/2022.09.21.508902v2.full.pdf | ②③ |
| 条件复杂度给出跃迁概率上界 | Dingle, Hagolani, Zimm, Umar, O'Sullivan, Louis, "Bounding phenotype transition probabilities via conditional complexity", J R Soc Interface 22(231):20240916 (2025-10-08). https://doi.org/10.1098/rsif.2024.0916 ；biorxiv: https://www.biorxiv.org/content/10.1101/2024.12.18.629197v3 | ②③ |
| 表型偏差=无条件编码定理 | Johnston, Dingle, Greenbury, Camargo, Ahnert, Louis, "Symmetry and simplicity spontaneously emerge from the algorithmic nature of evolution", PNAS 119(11):e2113883119 (2022). https://doi.org/10.1073/pnas.2113883119 | ③ |
| 简单性偏差普适形式 P(x) ~ 2^-K(x) | Dingle, Camargo, Louis, "Input–output maps are strongly biased towards simple outputs", Nat Mach Intell 2:83-89 (2020); 早期版 Nat Commun 9:761 (2018). https://www.nature.com/articles/s41467-018-03101-6 | ②③ |
| LLM 作为条件压缩器/条件码长 K_M(x|y) | Delétang et al., "Language Modeling Is Compression", ICLR 2024, arXiv:2309.10668. https://arxiv.org/abs/2309.10668 ；"The Value of a Prompt: An LLM-Relative Kolmogorov-Complexity Approach", arXiv:2608.16438 | ①③ |
| LLM≈Solomonoff/算法概率可计算近似 | "Large Language Models as Computable Approximations to Solomonoff Induction", arXiv:2505.15784 | ① |
| NCD 的条件版本 NCCD | Pratas et al., "A conditional compression distance that unveils insights of the genomic evolution", arXiv:1401.4134 (NCCD：用条件信息量替代联信息量) | ① |
| QD 选父=多臂老虎机 | Lee, Cully et al., "Monte Carlo Elites: Quality-Diversity Selection as a Multi-Armed Bandit Problem", GECCO 2021, arXiv:2104.08781 | ①④ |
| novelty=到档案 k 近邻平均距离 | Lehman & Stanley, "Abandoning Objectives: Evolution Through the Search for Novelty Alone", Evolutionary Computation 19(2):189-223 (2011)（稀疏度 ρ(y)=到档案 A 的 k 近邻平均距离 δ） | ①④ |
| QDAIF：LLM 打分选格 | Bradley et al., "Quality-Diversity through AI Feedback", ICLR 2024. https://qdaif.github.io/ | ①④ |
| OPRO：优化轨迹进 prompt | Yang et al., "Large Language Models as Optimizers", ICLR 2024, arXiv:2309.03409 | ①④ |

---

## 块1：Dingle-Ahnert-Louis 条件算法概率文献串的定位（定位见上表；细节展开在下方块2）

---

## 块2（已补）：Dingle–Novev–Ahnert–Louis 条件算法概率文献串：他们到底用什么可计算量

**来源（一手）**：Dingle, Novev, Ahnert & Louis, "Predicting phenotype transition probabilities via conditional algorithmic probability approximations", J. R. Soc. Interface 19(197):20220694 (2022)（全文 PDF 取自牛津 ORA 档案 https://ora.ox.ac.uk/objects/uuid:8b0cf447-0484-49fa-a24c-334f64149f33）；Dingle, Hagolani, Zimm, Umar, O'Sullivan & Louis, "Bounding phenotype transition probabilities via conditional complexity", J. R. Soc. Interface 22(231):20240916 (2025)（全文取自 Europe PMC, PMCID: PMC12503940）。

### 2022 文献串的具体数学与可计算近似（原文 equation 编号）

1. **无条件编码定理**（Levin）：P(x) = 2^(−K(x)+O(1))（eq. 3.3）。**条件编码定理**：P(y|x) = 2^(−K(y|x)+O(1))（eq. 3.4）——条件码长短的输出概率高；K(y|x) 低意味着「y 本身简单」**或**「y 与 x 相似」（共享子序列）。
2. **应用到跃迁**：P(x→y) ≲ 2^(−K(y|x,f)+O(1))（eq. 4.1）。若 GP 映射 f 本身复杂度低（K(f)=O(1)，即映射规则简单），则 P(x→y) ≲ 2^(−K(y|x)+O(1))（eq. 4.2）——**这正是「码长↓⟹跃迁概率指数↑」的因果链定理化表述：条件复杂度每降 1 bit，跃迁概率上界翻倍**。
3. **可计算近似**（关键，对应缺口①）：
   - K̃(y|x) ≈ K̃(xy) − K̃(x)（eq. 4.4）——**拼接串压缩长度减去 x 压缩长度**；
   - K̃ 用 **Lempel–Ziv 复杂度估计**（gzip 家族的源头）逐字符串计算；
   - 复杂度按 0 ≤ K̃(y|x) ≤ log2(Np) 缩放（Np = 可达表型数 N_y(x) 的估计），缩放对了则默认 a=1、b=0；
   - 实用形式：**P(x→y) ≲ 2^(−K̃(y|x))**（eq. 4.5）。拟合值 RNA 上 a≈1.1、b=0。
   - 2025 文献串沿用同一套：K̃(y|x) = K̃(xy) − K̃(x)，LZ 度量 C_LZ；并在附录给出信息论旁证——universal prediction 文献（Merhav & Cohen 的密码猜测）里就有条件编码定理预测器 2^(−LZ(y|x))，即「用压缩器算条件概率」在密码学/预测理论里已成惯例。
4. **测试系统**：2022 = RNA 二级结构（L=40）+ 蛋白二级结构（Porter5，CDRH3 抗体区）；2025 = 昼夜节律 ODE、基因调控网络矩阵乘法模型、环斑海豹牙齿形态发育模型、polyomino 自组装、HP 格点蛋白——**五个系统全是「自然映射」检验，全部不涉及任何检索系统设计**。
5. **实证精度（对缺口①「可计算指标」的现实校准）**：预测两个候选 y 哪个跃迁概率更高——RNA 86%（概率加权采样）/79%（均匀采样），蛋白 79%/77%；概率比值预测 Pearson r=0.91（RNA）/0.93（蛋白）。2025 定义三级预测标准：Level I（Spearman ρ<0 显著，负相关）→ Level II（上界线性，R²>0.5）→ Level III（斜率也对，bootstrap 检验）。
6. **两个直接命中缺口③的原文证据**：
   - 2022 Fig 1c：**无条件复杂度 K̃(y) 不能预测跃迁概率，必须用条件复杂度 K̃(y|x)**（原文："the unconditional complexity ~K(y) does not predict the transition probabilities well…this demonstrates that the conditional complexities are needed"）；
   - Fig 1d：少数表型的 K̃(y|x) 远低于 K̃(y)，**这些恰是高概率输出**——「相似性」是跃迁概率的来源，不是「简单性」本身。
7. **§4.6 对「流形内 vs 流形间」轴的直接理论支撑（对应缺口④第二问）**：原文指出 AIT 里几乎所有表型对 x、y 互不共享信息，故 K(y|x) ≈ K(y)（eq. 4.6）——**对「无关的远端表型」，条件复杂度退化为无条件复杂度**；但概率质量集中在 K(y|x) 低的少数输出上。即：**流形内（y 与 x 相似）跃迁由 K(y|x) 主导；流形间（y 与 x 无关）跃迁由 K(y) 主导，无引导时被简单性偏差锁死**。这条是文献里现成的「二分轴」理论根据。
8. **这串文献停在哪一步（划界证据）**：2022 讨论节明说「Another area of potential applicability is in genetic algorithms for optimization. Indeed, Hu et al….some target phenotypes are harder to find than others…In the future work, it would be interesting to assess if these mutational connections are related to conditional complexity, as our theory would suggest」——**他们预测概率，但从未提出「设计一个过程压低 K(y|x)」（催化剂）**；也无任何主动干预（变异算子/上下文/环境）改变条件复杂度的机制研究。2025 只是扩系统验证边界，同样止步于「预测」。
9. **相关引用可继续跳的点**：Hu, Tomassini & Banzhaf, "A network perspective on genotype–phenotype mapping in genetic programming", Genet. Prog. Evol. Mach. 21 (2020)（GP 中表型可达性研究，Dingle 2022 引作 GA 侧最近文献）；Schaper & Louis, "The arrival of the frequent: how bias in genotype-phenotype maps can steer evolution", PLoS Comput. Biol. 10:e1003549 (2014)。

---

## 块3：LLM 条件码长 −log P_LLM(x|context) 在进化循环里怎么算 + 现役系统的可计算选择指标目录

### 3.1 LLM 条件码长的理论地位（预测=压缩）

**来源**：Delétang, Ruoss, Grau-Moya et al., "Language Modeling Is Compression", ICLR 2024, arXiv:2309.10668（Google DeepMind）。核心：预测-压缩等价——任何预测模型经算术编码即成无损压缩器，压缩长度 ≈ −log2 P(x|context)；LLM 在文本/图像/音频上全部超过 gzip/PNG/FLAC 等经典压缩器；且"任何压缩器都可以变成条件生成模型"——**压缩视角把「上下文里的父代程序+档案」变成条件分布 P(·|context)，子代码长就是 −log2 P(child|context)**。这是缺口①「可计算指标」的理论通行证。

**来源**：Rafael Pass, "The Value of a Prompt: An LLM-Relative Kolmogorov-Complexity Approach", arXiv:2608.16438 (2026)。把经典定义里的通用图灵机换成 LLM 本身，定义 LLM 相对的概率版 Levin-Kolmogorov 复杂度 pKt（把模型思考当随机带、按 Levin 方式对数计费）；**prompt 的价值 b bit = 与产物的算法互信息 = 让产物「好获得 2^b 倍」**（成功概率×2^b 或所需计算÷2^b 或两者的权衡），且该互信息**可有效估计**（经典算法互信息不可计算）。这等于把缺口②的因果链（码长↓b bit ⟹ 命中概率×2^b）在 LLM 上重新定理化了一遍——b bit 的上下文价值直接翻译成指数级命中增益。

### 3.2 现役 LLM 进化系统实际用的可计算选择/选样指标（逐个核账）

| 系统 | 可计算选择量 | 来源 |
|---|---|---|
| FunSearch | 评估器分数 h；采样偏好「更高分**且更短**的程序」；岛屿模型保多样性；best-shot prompt = k=2 个程序排序后拼 prompt | Romera-Paredes et al., "Mathematical discoveries from program search with large language models", Nature 625, 468-475 (2024)（PMC10794145 全文核） |
| AlphaEvolve | 多标量评估函数 h（可加 LLM-judge 的「程序简单性」打分项！）；程序库 = MAP-Elites + 岛屿混合；LLM 出 diff；Gemini Flash（吞吐）+ Pro（偶发突破）混合；meta-prompt 与解共进化 | Novikov et al., "AlphaEvolve: A coding agent for scientific and algorithmic discovery", arXiv:2506.13131 (2025) 全文核 |
| OPRO | meta-prompt 内嵌「优化轨迹」= 历史解+分数按升序排列；LLM 读轨迹生成新解；选择全靠外部分数，无码长 | Yang et al., "Large Language Models as Optimizers", ICLR 2024, arXiv:2309.03409 |
| QDAIF | 质量 = LLM-as-judge 自然语言打分；多样性 = LLM 生成的定性描述轴（观点/体裁/结局）上的档案格；基于 ELM 的进化循环 | Bradley, Dai, Lehman, Stanley et al., "Quality-Diversity through AI Feedback", ICLR 2024（qdaif.github.io 核） |
| Monte Carlo Elites | 选父 = 多臂老虎机 UCB：奖励 = 该父代采出的子代进档案的比例（archive-add rate）+ 未尝试次数→∞ 优先 | Lee, Cully et al., GECCO 2021, arXiv:2104.08781；emitter 版本 UCB1 选 emitter（arXiv:2007.05352） |
| novelty search | 稀疏度 ρ(x) = 到档案 A 的 k 近邻平均距离 δ；超阈值入档案 | Lehman & Stanley, Evolutionary Computation 19(2) (2011)（ρ_y := (1/k)Σ δ(x_i^A, y)，MIT Press 版核） |
| MAP-Elites | 档案格内均匀随机选父 + 分数留档 | Mouret & Clune, GECCO 2015 |

**核账结论（对缺口①）**：现役系统的选择量分三类——外部评估分数、统计探索量（UCB/k-NN 距离）、LLM 定性打分。**没有一个系统用「条件码长」或任何它的函数做选择量**；更刺眼的是 FunSearch 偏好「更短程序」、AlphaEvolve 可把「简单性」加进分数——**两个现役旗舰都在用无条件复杂度当偏好方向**（按 Dingle 2022 Fig 1c 的证据，无条件码长不预测跃迁概率；按缺口③的顾虑，这会放大简单性偏差而不是打开稀有流形）。−log P_LLM(child|parent 上下文) 在循环里**可算**（logprob API/算术编码/NCD 皆可），但把它当选择指标**没有先例**——这是「必须自己发明」侧的最硬证据。

### 3.3 无引导 LLM 变异算子自带偏差（「表型偏差」的 LLM 版，对缺口③④）

**来源**：Gurkan, Stonedahl, Wilensky, "Mutation Without Variation: Convergence Dynamics in LLM-Driven Program Evolution", GECCO Companion 2026, arXiv:2606.05408。无选择压力下让 LLM 反复变异 DSL 程序、变 prompt 设计/模型家族/随机复制：LLM 变异**一致收敛到程序空间的受限吸引子区域**；87% 的链在结构层面收敛（93% 的变异重访已见过的结构形式），剩余变异多为「模板内换终端常量」；**经典 GP 子树变异无此收敛**。结论原文：使 LLM 具备语义感知变换能力的机制，**同时携带朝结构同质化的系统性偏差**。——这正好是 Johnston PNAS 2022 表型偏差（自然变异偏向简单/低复杂度表型）在 LLM 进化循环里的镜像：**无引导（无上下文/无催化剂）跃迁塌缩到流形内已有结构**。

---

## 块4：条件复杂度 K(x|y) 的可计算近似谱系 + 催化剂形式化查证（有没有人做过）

### 4.1 可计算条件复杂度近似的三条现成路径

1. **LZ/压缩器拼接差**（现役、已被验证）：K̃(y|x) = K̃(xy) − K̃(x)，K̃ 用 Lempel-Ziv；Dingle 2022/2025 用它在 RNA/蛋白/海豹牙/polyomino/HP/昼夜节律 ODE/GRN 七类系统上验证 P(x→y) ≲ 2^(−a·K̃(y|x)−b)（Level I/II 全过，Level III 部分过）。信息论侧旁证：universal prediction / 密码猜测文献（Merhav & Cohen）里 2^(−LZ(y|x)) 已是惯例性的条件概率赋值（Dingle 2025 附录引）。
2. **LLM logprob / 算术编码码长**（理论完备、工程可算、无人用于选择）：−log2 P_LLM(y|context)（Delétang 2023 的预测-压缩等价；Pass 2026 的 pKt 形式化+有效估计）。工程上=对候选子程序跑一遍带上下文的打分取 logprob，或用 LLM 算术编码压缩量。
3. **NCD 家族**：C(y|x) = C(xy) − C(x)（Li, Chen, Li, Ma, Vitányi, "Clustering by Compression", IEEE TIT 2004）；条件版 NCCD（Pratas et al., arXiv:1401.4134，用条件信息量替代联信息量、真条件压缩器）。尺度大、无需评分 API，但分辨率粗。

### 4.2 「催化剂 = 压低条件复杂度的东西」有没有人形式化过

**最接近的一篇（2026 预印本）**：Elija Perrier, "Watts-per-Intelligence Part II: Algorithmic Catalysis", arXiv:2604.20897 (2026-04)。在瓦特-智能框架里给「算法催化剂」下了正式定义，其中结构性选择条件为：**K_U(σ(C) | desc(H_exec,cat)) ≤ K_U(σ(C)) − η**——任务类描述符在催化剂执行结构的辅助下条件复杂度至少降 η bit；并证定理：类特异加速 Γ(T) 满足 log2 Γ(T) ≤ I_alg(催化剂执行结构 : 任务类描述符) + O(1)，即**加速上限 = 算法互信息位数**；另有部署视界耦合定理（催化剂要在能耗上划算所需的部署时长下界）。**但**：这是单作者热力学计算框架的预印本，落在 Landauer 擦除能耗语境，**不涉及进化循环、GP 映射、QD 档案，更没接 Dingle 的跃迁概率上界**。

**历史源头**：Walter Fontana, "Algorithmic Chemistry"（Physica D 1991 / Artificial Life II 1992 起；AlChemy 项目）——λ 演算项当分子、归约当反应，研究「组织 = 自维持的变换网络」；动机是生物缺一个「可能性的理论」。这是「把化学反应写成程序/计算」的开山，但度量是组合代数动力学，**没有条件复杂度概念**。

**生命起源侧**：Cronin, Walker, "The Algorithmic Origins of Life" (2012, arXiv:1207.4803)；Adami, "Information-theoretic considerations concerning the origin of life" (arXiv:1409.0590，把自发分子自复制概率写成信息量函数)；Kauffman 自催化集、hypercycle 一族（综述 Life 8:62）。都在讲「催化让不可能变可几」，**但没有一个给出 K(x|current+catalyst) ≪ K(x|current) 这种算法信息论形式的机制定义**。

**结论（划界）**：催化剂的形式化碎片存在（Perrier 的结构性选择条件几乎就是那句话），Pass 的 prompt 价值把「上下文=催化剂、bit=加速倍数」定理化，但**「以压低条件复杂度为检索目标、以跃迁概率上界为反馈的进化检索循环」两半各有文献、中间无人焊接**。

---

## 块5：四空位「文献现成可填 vs 必须自己发明」划界 + 「流形内/流形间」轴的证据

（空位按用户视角句结构读出：A 空间=精英解稀疏低维流形；B 指标=命中流形穿越点的代理量；C 检索对象=压低解复杂度的过程；D 因果运动=复杂度压低→穿越点命中。）

### 5.1 空位A（空间）：**大部分现成，一处要拼**
现成：Wagner「基因型网络/中性网络」——同表型基因型构成互连网络、可在基因型空间中延伸很远（Wagner, *The Origins of Evolutionary Innovations* 2011；PMC3282381）；QD 档案=行为描述子空间里的精英格（Mouret & Clune 2015；CVT-MAP-Elites 的低维质心格）。要拼：**「档案=流形、穿越点=网络交集」的合并视角**——穿越点有现成的生物学对应物：Bendixsen & Hayden, "Genotype network intersections promote evolutionary innovation", PLOS Biology (2019, PMC6555535)：两个 RNA 酶（HDV 自切/连接酶）的基因型网络**交集=双功能序列**，16484 个变体里 9032 个双功能；中性漂移 400 代后才能登顶无漂移时到不了的峰。把 QD 档案格间移动读成「网络交集穿越」这一步没有人做过。

### 5.2 空位B（指标）：**零件现成，成品没有**
现成零件：(i) K̃(y|x) LZ 版——预测两个候选哪个更可达 79-86% 准确率（Dingle 2022）；(ii) −log P_LLM(child|parent 上下文)——理论+工程皆可算（Delétang/Pass），无人当选择量用；(iii) MCE 的 archive-add rate——**经验版的「命中新格率」**，但它只能事后统计，不能先验预测；(iv) novelty 的 k-NN 距离——「落点稀疏度」，不是「跳得过去」。必须发明：把 (i)/(ii) 接到 (iii)/(iv) 上——例如用 2^(−K̃(y|x)) 当 archive-add rate 的先验预测器、或定义「双低」候选（对两个档案区域的条件码长同时低=潜在交集成员）。任何一种组装在文献里都查无此物。

### 5.3 空位C（检索对象=催化剂过程）：**必须自己发明**
全部近邻都停在外面：Dingle 串只预测不干预（2022 讨论节明说 GA 应用是 future work）；Perrier 2026 有催化剂定义但在热力学语境、无进化循环；Pass 2026 评估 prompt 价值、不构造催化剂；Fontana 1992 有「分子=程序」无复杂度度量；FunSearch/AlphaEvolve 的「上下文=父代+灵感程序+评估回显」实际上**是**在当催化剂用，但没有任何一处把它的效果度量成条件码长差。「检索使 K(y|x,c) ≪ K(y|x) 成立的 c」作为检索目标本身无人提出。

### 5.4 空位D（因果运动）：**定理现成，接线自己来**
编码定理环节的每一环都有定理与实证：K(f)=O(1) 简单映射 ⟹ P(x→y) ≲ 2^(−K(y|x))（Dingle 2022 eq 4.2/4.3/4.5；经验 a≈1.1, b≈0；Level I-III 检验）；**条件码长每降 1 bit，跃迁概率上界×2**；LLM 侧：b bit prompt 价值 ⟹ ×2^b 易获得（Pass 2026）。缺的只是把它讲成检索循环的因果骨架：催化剂 c 把 K(y|x) 降 b bit ⟹ 命中穿越点的概率上界×2^b ⟹ archive-add rate/新格发现率的先验预测。这条叙事线文献里没有连成句。

### 5.5 「三领域强/弱标注」换「流形内 vs 流形间」轴：证据支持换
- Dingle 2022 §4.6+Fig 1c/1d 的数学本身就是这个轴：无关表型对 K(y|x)≈K(y)（流形间跃迁退化为无条件复杂度支配=简单性偏差锁死），概率质量集中在 K(y|x)≪K(y) 的少数高概率输出（流形内/近邻）。**轴不是比喻，是 K(y|x) 与 K(y) 差值的符号结构**。
- Gurkan 2026：无引导 LLM 变异=流形内塌缩（结构同质化 87-93%），经典 GP 子树变异无此塌缩——「哪个领域强」取决于变异算子的引导结构，不取决于领域标签。
- Bendixsen & Hayden：创新发生在交集=同时在两个流形内的点——「流形间」的通道由「对两侧都低条件码长」的序列承担。
- Johnston PNAS 2022（PMC8763027）：自然 RNA 结构对形态空间的占据由表型偏差（发育偏差）主导，选择难以翻越——无引导时**所有领域**被同一无条件偏差支配，领域强弱标注会把机制差异（引导）误标成领域差异。
结论：换成「流形内（K(y|x) 支配）vs 流形间（K(y) 支配）」轴后，四个空位、缺口②③的因果链全部能落在这根轴上，且有现成数学（K(y|x)/K(y) 比值）可直接计算。

---

## 块6：补充信源台账（本轮新增，按一手>二手分层）

| 条目 | 来源 | 分层 |
|---|---|---|
| 条件编码定理 P(y\|x)=2^(−K(y\|x)+O(1)) 及全部推导 | Dingle et al. 2022, J R Soc Interface 19:20220694（ORA 全文 PDF 逐节核：eq. 3.4/4.1/4.2/4.3/4.4/4.5/4.6/4.8、Fig 1a-f、§4.6、讨论节） | 一手 |
| 条件复杂度界在 7 类 GP 映射上的验证 + Level I/II/III 标准 + LZ 度量 C_LZ + Merhav&Cohen 旁证 | Dingle et al. 2025, J R Soc Interface 22:20240916（Europe PMC PMC12503940 全文核） | 一手 |
| 表型偏差/简单性偏差 P(x)∝2^(−aK̃(x)−b) | Dingle, Camargo, Louis, Nat Mach Intell 2:83 (2020)；Johnston et al., PNAS 119:e2113883119 (2022)（PMC8763027: 自然 RNA 形态空间占据由偏差主导） | 一手 |
| 预测=压缩、算术编码、LLM 跨模态压缩 | Delétang et al., "Language Modeling Is Compression", ICLR 2024 (arXiv:2309.10668) | 一手 |
| LLM 相对 pKt、prompt 价值 b bit ⟹ 2^b、可有效估计 | R. Pass, arXiv:2608.16438 (2026, v2) | 一手预印本 |
| 算法催化剂正式定义（结构性选择 K_U(σ(C)\|desc(H))≤K_U(σ(C))−η；加速≤算法互信息；耦合定理） | E. Perrier, "Watts-per-Intelligence Part II: Algorithmic Catalysis", arXiv:2604.20897 (2026-04) | 一手预印本（单作者、热力学框架） |
| 基因型网络交集=双功能序列、9032/16484 双功能、中性漂移 400 代登顶 | Bendixsen & Hayden, PLOS Biology, PMC6555535 (2019) | 一手 |
| 基因型网络跨基因型空间延伸、近邻表型多样 | Wagner, The Origins of Evolutionary Innovations (OUP 2011)；PMC3282381 | 一手（书/综述） |
| LLM 变异吸引子收敛 87%/93%、GP 子树变异对照 | Gurkan, Stonedahl, Wilensky, GECCO Companion 2026, arXiv:2606.05408 | 一手 |
| FunSearch 采样偏「更高分且更短」、best-shot k=2、岛屿模型 | Romera-Paredes et al., Nature 625:468 (2024)（PMC10794145 全文核） | 一手 |
| AlphaEvolve 程序库=MAP-Elites+岛屿、LLM-judge 简单性打分、diff 变异、Flash+Pro 混合、meta-prompt 共进化 | Novikov et al., arXiv:2506.13131 (2025)（arXiv HTML 全文核） | 一手 |
| MCE UCB/奖励=archive-add 比例 | Lee et al., GECCO 2021 (arXiv:2104.08781)；emitter-UCB1: arXiv:2007.05352 | 一手 |
| novelty ρ(y)=k 近邻平均距离 | Lehman & Stanley, Evol. Comput. 19(2) (2011)；sparseness 定义另见 Evol. Comput. 32(3):249 | 一手 |
| QDAIF LLM-as-judge 质量+定性描述轴多样性 | Bradley et al., ICLR 2024 (arXiv:2310.13032, qdaif.github.io) | 一手 |
| OPRO 轨迹升序入 meta-prompt | Yang et al., ICLR 2024 (arXiv:2309.03409) | 一手 |
| NCD C(y\|x)=C(xy)−C(x) | Li, Chen, Li, Ma, Vitányi, IEEE TIT 50(6) (2004) | 一手 |
| NCCD 条件压缩距离 | Pratas et al., arXiv:1401.4134 | 一手 |
| Algorithmic Chemistry | Fontana, Physica D 1991 / Artificial Life II (AlChemy, SFI) | 一手（历史） |
| 算法生命起源 / 自复制概率=信息量函数 | Cronin & Walker, arXiv:1207.4803；Adami, arXiv:1409.0590 | 一手综述/预印本 |
| GP 侧表型可达性 | Hu, Tomassini, Banzhaf, Genet. Prog. Evol. Mach. 21 (2020)（Dingle 2022 引文 83） | 一手 |
| 分子结构 Boltzmann 分布预测随机突变可能变化 | PMC10698324 (2023) | 一手 |

**本轮检索未找到（=「必须自己发明」的负证据）**：
1. 任何用 −log P_LLM(子代|父代上下文) 或 LZ 条件码长做进化选择指标的已发表系统；
2. 任何把 QD archive-add rate 与 2^(−K̃(y|x)) 先验上界接起来的工作；
3. 任何在进化检索循环里以「压低 K(y|x)」为显式检索目标/奖励的「催化剂检索」工作；
4. QD 文献里「流形内 vs 流形间」的显式命名轴（数学对应物 K(y|x) vs K(y) 存在于 AIT 文献，但 QD 侧无人这样切分）。
