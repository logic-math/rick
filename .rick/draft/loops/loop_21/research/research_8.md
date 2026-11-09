# research_8 — 视角候选供给：四领域把「测量/验证工具自身演化」建成本领域问题的方式（SENSE 第 2 步·构建视角——视角候选供给）— 2026-11-08

> 时间基准 2026-11-08。信源等级：A = 一手（论文原文/官方文档/经典专著题录）；B = 权威综述或教材收编口径；C = 二手转述（仅线索）。关键题录本次 web 核对：Cliff-Miller 1995／Watson-Pollack 2001／Ficici-Pollack 1998·2001／Rosin-Belew 1997／De Jong-Pollack 2004／De Jong 2007／Van Valen 1973（常识级）／Floreano-Nolfi 1997／Salimans et al. 2016／arXiv 1811.11083／DeMillo-Lipton-Sayward 1978／Offutt 1992／Just et al. 2014／Kiela et al. 2021（Dynabench）／LiveBench（ICLR 2025）／Sainz et al. 2023／POET（GECCO 2019）／Enhanced POET（ICML 2020）／ACCEL（ICML 2022）。
>
> **使用边界（硬约束执行）**：本材料是 SENSE 第 2 步「构建视角」的**视角候选供给件**——只给四领域把「测量/验证工具自身演化」问题化成自己领域问题的原生方式，供 human 自己产出强相关（收敛）/弱相关（发散）知识领域清单及理由。**不划分强弱、不裁决哪个视角适配 rick super、不裁决孔雀隔离原则对任务集自进化的许可性**；每例只描述特征。不重复 research_6（编译器自举/免疫自耐受/quant walk-forward）与 research_7（控制论能观性/regret 理论/IRT/定量遗传学）已供内容，交叉处仅引用不重述。
>
> **任务锚（human 追问，judgment.md 2026-11-08）**：「构建任务集，这个任务集可以自进化吗？」——任务集既是验证工具又是候选可进化对象。精确问题：测量/验证工具自身演化在哪些领域是原生问题？彼域漂移防护（隔离/锚定）机制是什么？孔雀隔离原则是禁止任务集自进化，还是存在有防护结构的共同进化？

## 使用方式（给 parent/teach）

- 四例各按五件结构：①原生问题化方式（对象/进步的定义/关键结构）②关键机制或定理与为什么成立（标信源等级）③与卡点的结构映射（任务集/选择器/漂移在彼域的对应物）④漂移防护机制（锚/隔离/档案）⑤收敛/发散特征（只描述特征，不划分强弱）。
- 四例分工：**例一（竞争性共同进化）答「无防护的共同进化会怎样」**＝孔雀警句的病理学形式化；**例二（变异测试）答「验证工具自身如何被验证——工具不进化的路线」**；**例三（ML 动态基准）答「工具必须刷新时锚什么、付什么代价」**；**例四（UED/POET）答「有防护结构的任务集共同进化存在吗」**。
- 四例是候选视角库，不是清单本身：强/弱相关划分及理由是 human 构建视角步骤的产出位。

## 视角一｜演化计算的竞争性共同进化（＋GAN＋生物 Red Queen）：cycling／disengagement／平庸稳定态——「选择器与候选共同进化」的病理学与修复学

（五件详述见下文追加块）

## 视角二｜变异测试（mutation testing）：mutant-kill 评分把测试套件本身建成被测物——「谁验证验证者」的工程答案

（见下文追加块）

## 视角三｜ML 动态基准与数据污染：固定基准→饱和/污染 vs 滚动刷新→可比性丢失

（见下文追加块）

## 视角四｜无监督环境设计/开放式共同进化（UED/POET/ACCEL）：minimal criterion＋transfer——「任务集与学习者共同进化」的受防护原生实现

（见下文追加块）

## 四例横向对照表

（见下文追加块）

## 与 research_6/7 的交叉引用边界

（见下文追加块）

## R7 上报项

（见下文追加块）

## Gaps

（见下文追加块）

### 视角一① 原生问题化方式（对象/进步的定义/关键结构）

- 对象：两个以上相互演化、适应度由「与对方对抗的战绩」定义的种群——**评价者（对手＝测试）本身在被候选驱动着演化**。孔雀结构（选择器与候选互相积累偏好）在该域不是警示语而是研究对象本身。
- 原生问题：无外部基准时「进步」如何定义与测量——Cliff-Miller 1995 论文标题即 "Measurements of Adaptive Progress"；**主观适应度（战胜当代对手）与客观适应度（对全体/历史对手）的偏离**被建成核心研究问题。
- 进步的定义：arms race（军备竞赛）——两种群相互驱动对方性能/复杂性递增；失败形态的命名学（见②）就是该域的「验证层病理学」。
- 关键结构：相对适应度评价（评价集＝当代对手种群）＋种群共演化动力学。

### 视角一② 关键机制或定理（为什么成立）

- **cycling（循环）**：不可传递优劣（A 胜 B、B 胜 C、C 却胜 A）→ 种群「追自己尾巴」——对旧冠军的战绩周期性回退、看似每步都在进步。成立因为相对评价只对当代对手定义，环路劣优结构使「进步」无全局序可言。（A 一手：Watson & Pollack, GECCO 2001 "Coevolutionary dynamics in a minimal substrate"——摘要原话 "chases its own tail"；Cliff & Miller, ECAL 1995；Floreano & Nolfi, GP-97 1997 捕食-逃逸机器人实证——"the everchanging fitness landscape caused by the competing species (named the 'Red Queen effect')"）
- **disengagement（丢梯度/脱接触）**：一方碾压时被支配种群失去选择梯度、随机漂移，胜方随之失去改进压力。成立因为适应度信号由双方差距供给，差距过大＝信号消失。（A：Watson & Pollack 2001；二手归引 "the gradient for selection disappears and the populations disengage and drift"）
- **mediocre stable states（平庸稳定态）**：军备竞赛失败时落入互相制服不了对方的平庸均衡。Ficici & Pollack 1998 原文要义：competitors 必须实力足够匹配才能互相逼出改进，一方 overpowered 则对方找不到梯度。（A：Ficici & Pollack, "Challenges in Coevolutionary Learning: Arms-Race Dynamics, Open-Endedness, and Mediocre Stable States", Artificial Life VI 1998）
- **三分类**：Watson-Pollack 2001 把共同进化失败分为 loss of gradient／over-specialization（互相钻营对方特定弱点、无泛化解）／cycling——主观适应度偏离客观适应度的三条已命名机制。（A）
- **生物 Red Queen**：Van Valen 1973（Evolutionary Theory 1:1-30，A 一手/题录常识级）：物种灭绝概率与已存活时长无关——共同演化使适应度地形持续变形，「奔跑只为留在原地」。生物域**无修复机制**：Red Queen 是常态不是病理。
- **修复① Hall of Fame（历代精英档案）**：每代最优个体永久存档，适应度＝对全部历史精英的战绩——评价不再锚在当代对手。机制：把评价集从「当代」扩到「全历史」，当代对手种群的漂移不再单独决定选择信号。（B：概念源于 Cliff-Miller 追逃研究、Rosin & Belew 1997 系统化＋shared sampling〔按区分度选对手评价〕，Evolutionary Computation 5(1):1-29，题录 A 期刊）
- **修复② Pareto 共同进化＋单调档案定理**：把「对每个对手/测试的战绩」当独立维度（不折叠成单标量）——防单轴偏好累积。（A 一手：Ficici & Pollack, "Pareto Optimality in Coevolutionary Learning", ECAL/2001）该系定理：**给定完备评价集，Pareto 共同进化达成理想评价（按问题全部底层目标评价）**（A：De Jong & Pollack, "Ideal Evaluation from Coevolution", Evolutionary Computation 12(2), 2004）。**单调改进档案（IPCA, GECCO 2004；"A Monotonic Archive for Pareto-Coevolution", Evolutionary Computation 15(1):61, 2007）：档案中的解对已收录的全部档案测试不退化有保证；代价＝档案可无限增长（有界变体 LAPCA, CEC 2004 换保证）**。（A 一手）
- **修复非免费**：HoF/档案自身可产生欺骗性适应度近似（"Tripping Over the Past: Measuring Deceptive Progress in Competitive Coevolutionary Algorithms Employing Hall of Fame", App. of Evolutionary Computation 2024，B）——防护有自己的失效面。
- **GAN 同族**：判别器＝共同演化选择器、生成器＝候选；训练是求 Nash 均衡不是单目标优化、可能不收敛；mode collapse＝生成器钻营判别器弱点（over-specialization 的 GAN 形态）。（A：Salimans et al. 2016, arXiv 1606.03498——historical averaging＝把参数拖向历史平均〔fictitious play 式〕；arXiv 1811.11083, NeurIPS 2019——GAN 训练被重述为持续学习问题、判别器对旧分布灾难遗忘，**experience replay＝存过去生成样本混入判别器训练**＝档案机制在 GAN 的独立重发明，A）

### 视角一③ 与卡点的结构映射（任务集/选择器/漂移的对应物）

- 任务集 ↔ 对手/测试种群；选择器 ↔ 相对适应度评价（评价集＝当代对手）；漂移 ↔ cycling／丢梯度／平庸稳定态——「二者互相积累偏好」被拆成**三条可命名、可复现的失效机制**（孔雀警句的病理学形式化）。
- 修复映射：**HoF/archive 是「信息隔离」的第三种实现形态**——research_6 三镜（种子外置/germline 冻结/数据分割）之外的历史档案形态：不切断空间、不分割数据，而是**把选择器锚定在全部历史精英上——评价基面只能单调增长、不可被当代候选改写**（「隔离」从空间隐喻改写成单调不变式）。Pareto/完备评价集给出**选择器可以进化的定理级条件**——孔雀隔离原则在彼域不是禁令而是带条件的许可（完备性＋单调档案）。
- 异质环：彼域理论预设「客观适应度存在」（评价正确性有外部定义，理想评价以全部底层目标为参照）；rick 域 harness 质量无客观适应度——该域可移植的是**结构**（档案不变式、Pareto 维度化）而非**定理**（理想评价依赖完备性）。

### 视角一④ 漂移防护机制（锚/隔离/档案）

- **档案**：Hall of Fame、IPCA 单调档案、GAN experience replay——共同不变式＝**评价基面只增不删**（旧锚不可删除、新锚可以追加；任务集可以进化，但进化只能单调加锚，不能改写既有评价基面）。
- **隔离**：评价与生成的时间错位——评价用历史精英、生成用当代种群（隔离从「空间/数据」实现改为「时间」实现）。
- **锚定**：Pareto 维度化（对每个测试独立记分，防单轴偏好累积）；shared sampling（按区分度选对手，防任务集信息退化——与 research_7 概念三 IRT 项目信息函数独立同构，引用不重述）。

### 视角一⑤ 收敛/发散特征（只描述特征，不划分强弱）

- 特征：病理分类法直接可译（孔雀警句→三条已命名失效模式，且各有文献原型）；修复有定理级版本（单调档案、理想评价）但前提（完备评价集/客观适应度）在 rick 域未定；生物对照给出「无防护的共同进化是常态」的反向参照（生物靠世代时间与生态网兜底、无档案制度）。

### 视角二① 原生问题化方式（对象/进步的定义/关键结构）

- 对象：**测试套件（＝任务集）本身成为被验证物**——普通测试验程序，变异测试验**测试**。原生问题：test adequacy（我的测试够不够好）——「谁验证验证者（quis custodiet ipsos custodes）」的工程答案：用**第二层受控故障（mutant）**验证验证器，mutant 由算法生成、其「存在性/可杀性」先验已知（验证器的验证器有一个已知答案的生成器）。
- 进步的定义：mutation score（被杀掉的非等价 mutant 占比）上升——测试套件充分性被量化为单一分数。
- 关键结构：mutant 算子（对被测程序注入小语法变异＝人工故障）＋ kill（测试使 mutant 失败）＋ surviving mutant（测试盲区）＋ mutation score。

### 视角二② 关键机制或定理（为什么成立）

- **奠基与耦合效应**：DeMillo-Lipton-Sayward 1978 "Hints on Test Data Selection: Help for the Practicing Programmer"（IEEE Computer 11(4):34-41，A）——耦合效应原话："Test data that distinguishes all programs differing from a correct one by only simple errors is so sensitive that it also implicitly distinguishes more complex errors"（经 mutationtesting.uni.lu 学术仓储核对，A/B）。Offutt 1992（ACM TOSEM 1(1):5-20，A）把耦合效应系统化为可验假设并实证支持——**用简单故障代理复杂故障**是该评分器的立身假设。
- **mutant 对真实故障的代表性（实证锚）**：Just et al. 2014（FSE，A）：357 个真实故障、5 个开源应用、321 KLOC——mutant 检测率与真实故障检测率显著相关、**独立于代码覆盖**；结论原文要义：大部分情形 mutant 是真实故障的有效替身，但**部分真实故障不被 mutant 代表**（代理不完备）。
- **等价 mutant 问题**：某些 mutant 与原程序语义等价→任何测试都杀不掉→分母失义；一般情形下判定程序等价**不可判定**（停机问题类）——变异分数自身永远没有完美分母。（A：Kintis et al., IEEE TSE 2013 等价 mutant 系统综述；Jia & Harman 2011 综述 B）
- **SBST（search-based software testing）**：用固定适应度（覆盖/变异分数）**进化测试套件**——候选（测试集）进化、选择器（准则）冻结的工程形态。（B：Harman et al. SBST 综述，STVR 2012）

### 视角二③ 与卡点的结构映射（任务集/选择器/漂移的对应物）

- 任务集 ↔ 测试套件；选择器 ↔ mutant 算子集＋kill 评分；漂移 ↔ 套件对算子过拟合（只杀简单变异不抓真实复杂故障）＋等价 mutant 分母失义（分数自身歧义）。
- 防护的彼域形态：**mutant 算子集冻结**——锚定在真实故障经验上（耦合效应＋Just 2014 实证），不随被测套件/程序共同进化；评分器与被测套件分侧（套件只当输入、不当评分器）。
- 对学生问题给出的是**禁止共同进化路线的工程先例**：验证器不进化，用冻结的故障模型验证之——孔雀隔离原则的保守实现；「任务集自进化」的肯定路线不在该域（在视角一/四）。
- 异质环：「简单错误」在程序域有结构定义（语法变异算子集）；LLM-harness 轨迹的「简单错误」无现成 mutant 算子——耦合效应在 harness 域是**待验假设**非可用定理。

### 视角二④ 漂移防护机制（锚/隔离/档案）

- **锚定**：mutant 算子集冻结、锚在真实故障模型（外部现实锚——与 research_6 编译器镜「ISO 规范外部冻结」同族，引用不重述）。
- **隔离**：评分器与被测套件分侧；套件进化（SBST）时准则不进化。
- **档案**：surviving mutant 清单本身＝**测试盲区档案**（未杀 mutant 显式列出＝任务集缺口的制度化登记——与 research_6 编译器镜 XFAIL「已知失败登记」同族，引用不重述）。
- **未解裂缝**：等价 mutant 不可判定→防护自身无完美判据（变异分数的分母永远含歧义成分）。

### 视角二⑤ 收敛/发散特征（只描述特征，不划分强弱）

- 特征：工程成熟度最高（工具链工业化、标准 fault model、分数二值化可算）；判分 kill/survive 二值＝「信号明确」的极限形态可移植；但 mutant 算子的「简单错误」结构在 harness 域缺位——可移植的是**双层结构**（用受控故障验证验证器）而非算子集本身。

### 视角三① 原生问题化方式（对象/进步的定义/关键结构）

- 对象：benchmark（评测任务集）本身在模型进步下**老化**——三种已命名老化形态：**污染**（测试数据进入训练语料＝候选「见过题」：选择器向候选泄题）、**饱和**（全员高分、区分度→0——IRT 项目信息函数近零，research_7 概念三「群体进步后同一工具自然老化」同构，引用不重述）、**过拟合**（拟合题面风格而非能力）。
- 该域把「任务集可以自进化吗」做成**工程运营问题**：基准必须定期换血否则失义；换血又破坏跨时间可比性——原生 trade-off。
- 关键结构：数据分割（隔离）＋滚动发布（刷新）＋历史档案（可比性）＋judge 机制（谁打分）。

### 视角三② 关键机制或定理（为什么成立）

- **Dynabench**（Kiela et al., NAACL 2021，A）：human-and-model-in-the-loop 逐轮对抗采集——标注者构造「骗过目标模型而骗不过人」的样本；机制原文要义：过程逐轮重复，每轮数据集针对上一轮领先模型采集、模型全部重评（dynabench.org/about＋论文摘要，A）。**轮次结构＝任务集与模型共同进化，但每轮数据一经发布即冻结为档案**。evaluation-as-a-service：模型在云端被评、拿不到测试数据（隔离的工程实现）。动机原话要义："contemporary models quickly achieve outstanding performance on benchmark tasks but nonetheless fail on simple challenges"。
- **LiveBench**（ICLR 2025 Spotlight，A）：**月度滚动新题**（题源＝最近发布的 arXiv/新闻/IMDb——外部熵源）＋**每题可验证客观答案**（判分不依赖 LLM judge）。README 原话："LiveBench limits potential contamination by releasing new questions regularly"（A GitHub）——注意自家机制描述用 limits（限制）而标题用 contamination-free（绝对化声明，见待裁决 15）。
- **污染测量学**：Sainz et al., Findings of EMNLP 2023 "NLP Evaluation in trouble"（A）：最坏情形＝LLM 在测试 split 上训练又在同一基准被评——性能高估且**污染程度本身不可直接测量**。综述（arXiv 2411.03923，A）：污染的定义与测量本身是开放问题（n-gram 重叠/MinHash/成员推断各有口径与失效面）。
- **理论根**：Goodhart 类定律（测度成为目标即失效）＝该域常识级根法则（B，Campbell 1979/Strathern 1997 题录未逐篇核对）；与 research_7 概念二元过拟合/递归自食退化同构（引用不重述——Shumailov 2024 已供）。

### 视角三③ 与卡点的结构映射（任务集/选择器/漂移的对应物）

- 任务集 ↔ benchmark；候选 ↔ 被评模型；选择器 ↔ 评分函数＋judge；漂移 ↔ 污染（「二者互相积累偏好」的 ML 形态：题面与模型共适应）＋饱和＋judge 漂移。
- 该域**已把「任务集自进化」做成日常工程**（rolling benchmark 是行业实践）——对学生问题的直接形态回答：可以且已在做，代价与防护见④⑤。
- 异质环：该域解法的隐藏前提＝**外部世界持续供给新题与客观答案**（arXiv/新闻/IMDb）；rick super 四元组任务的「目标清晰/信号明确」若无外部源持续供给，滚动刷新不可移植。judge 漂移消除的代价＝任务集被锚收窄（只收可验证答案题）。

### 视角三④ 漂移防护机制（锚/隔离/档案）

- **隔离**：held-out/云端评测（test 数据不进训练——research_6 quant 镜数据分割同族，引用不重述）。
- **锚定**：题级客观 ground-truth 答案（判分不随模型/裁判变）；全部历史题目保留＝可比性锚（**结构上是 research_7 概念三 IRT 锚题等值的工程亲缘**：旧题＝锚题、新题＝新量表、跨期分数靠锚题链可比——引用不重述）。
- **档案**：发布即冻结的轮次数据集（Dynabench 每轮数据永久公开＝评价基面只增不删）。
- **刷新**：外部熵源（最近发布语料）——防护的资源依赖（无外部源则此路不通）。

### 视角三⑤ 收敛/发散特征（只描述特征，不划分强弱）

- 特征：与卡点同域近邻（任务集＝评测集、全结构同型）且工程实践现成、成本与失效案例公开；可比性在彼域靠锚题链维持（链越长维护成本越高、误差可累积——同 research_7 锚题等值误差结构，引用）；任务集的「进化」被限定为「外部供给＋锚定判分」，不是内生生成（内生生成路线在视角四）。

### 视角四① 原生问题化方式（对象/进步的定义/关键结构）

- 对象：训练环境/课程（任务集）本身由算法生成并**与学习者共同进化**。该域的原生问题就是学生原话的学术形式化——POET 系原文要义："How can progress in machine learning... be automated to generate its own never-ending curriculum of challenges without human intervention?"（A）
- 进步的定义：已解任务的数量/复杂度增长（open-endedness）＋对未见环境的零样本迁移（transfer）。
- 关键结构：环境-智能体**成对种群**（paired：每环境配属自己的 agent）＋**准入判据**（minimal criterion）＋**迁移**（transfer）＋新颖性（novelty）。

### 视角四② 关键机制或定理（为什么成立）

- **POET**（Wang-Lehman-Clune-Stanley, GECCO 2019 / arXiv 1901.01753，A）：环境-智能体成对共同进化；新环境经变异生成，入种群须过 **minimal criterion：不能太难（至少一个 agent 能学会）也不能太易（至少一个 agent 尚未解决）**＋新颖性判据；transfer：周期性把其他环境的 agent 迁来试跑、更好则替换——**解跨环境保留（档案式）＋跳出局部最优**。（A：GECCO 版 10.1145/3321707.3321799 机制节：环境评估配对 agent、ELIGIBLE_TO_REPRODUCE、子环境与配对 agent 同过准入判据；Clune 讲稿一页版 "add to population IF not too easy, not too hard, novel"）
- **Enhanced POET**（ICML 2020 / arXiv 2003.08536，A）：规模化后揭示与修复新的失效（步长/算力/课程治理）——**防护结构本身随规模继续演化**（受控共同进化的第二层实证：防护不是一次设计定型）。
- **UED/regret 锚**：ACCEL＝"Adversarially Compounding Complexity by Editing Levels"（Parker-Holder et al., ICML 2022 / arXiv 2203.01302，A）：环境设计建成 teacher-student 博弈，teacher 目标＝**最大化 regret**（环境中最优策略与学生表现的差距）→ 任务被生成在「学生能力的前沿」；ACCEL 用进化编辑课程＋regret 排序。PAIRED（Dennis et al., NeurIPS 2020，minimax regret 环境设计）为其 teacher（B，题录未逐篇核对）。**regret 与 research_7 概念二同源**（引用不重述）：「任务集往哪进化」的判据锚定在与基准的差距上。
- **为什么 minimal criterion 是防护**：准入判据**锚定在当前种群的实际可学性**上（不依赖生成器自评）→ 防两类漂移：太难→丢梯度（视角一 disengagement 的课程版）、太易→零信息（IRT 信息函数近零，research_7 引用）；novelty 防同类任务反复生成（偏好累积）。

### 视角四③ 与卡点的结构映射（任务集/选择器/漂移的对应物）

- 任务集 ↔ 环境种群（生成的课程）；候选 ↔ agent 种群；选择器 ↔ 准入判据＋novelty＋regret；漂移 ↔ 无判据时生成器漂移（太难/太平庸任务堆积）。
- **与孔雀原则的正面关系（该域关键分歧点）**：选择器（准入判据）**读得到候选种群状态**（不隔离）——表面违反「信息隔离」；但选择器的**目标函数锚在环境自身冻结的 reward 上**（环境目标不随 agent 改），耦合被压缩到「难度前沿随种群移动」这一受控维度。该域答案是**受控共同进化**而非隔离——孔雀原则在该域被改写成「目标冻结＋通道窄化」，不是禁止。
- 异质环：RL 环境的「可学性/regret」有可算定义（reward、最优策略可求）；harness 任务的「可学性」需先有测量层（依赖序：先 research_7 概念一/三、后本视角）；该域无收敛保证（open-ended＝发散式复杂度增长，「更好」＝可解前沿推进而非回归收敛——进步观与「下次改进的更好」需对齐）。

### 视角四④ 漂移防护机制（锚/隔离/档案）

- **锚定**：minimal criterion（可学性锚在种群当前能力）＋环境 reward 冻结（任务目标不随候选改——四元组「目标清晰」的彼域实现）＋regret（生成判据锚在与基准的差距）。
- **档案**：成对保留＋transfer（历史最优解不丢、可跨环境复用）。
- **隔离**：**弱隔离**——生成器不直接读 agent 内部状态，只通过准入判据间接耦合（通道窄化，非切断）。

### 视角四⑤ 收敛/发散特征（只描述特征，不划分强弱）

- 特征：「任务集可以自进化吗」的最直接**肯定**答案（该域以此为设计目标、有开源实现）；防护判据三件齐全（可学性/regret/novelty）；但「可学性」锚随种群移动——**锚本身是移动的**（该域把这种移动当特性：课程随学生走）；与孔雀「信息隔离」的关系不是违反/遵守二选一，而是**目标冻结＋通道窄化**的第三种结构（材料并列呈现，裁决留给 human）。

## 四例横向对照表

| 视角 | 任务集对应物 | 选择器对应物 | 漂移形态 | 漂移防护（锚/隔离/档案） | 彼域对「任务集可否自进化」的答案 |
|---|---|---|---|---|---|
| 一 竞争性共同进化 | 对手/测试种群 | 相对适应度（评价集＝当代对手） | cycling／丢梯度／平庸稳定态 | HoF＋单调档案（评价基面只增不删）＋Pareto 维度化＋shared sampling | **可——带条件**：评价基面单调增长不可改写（定理级：完备评价集→理想评价） |
| 二 变异测试 | 测试套件 | mutant 算子集＋kill 评分 | 套件过拟合简单变异／等价 mutant 分母失义 | 冻结故障模型（外部现实锚）＋评分器与套件分侧＋surviving mutant 盲区档案 | **否（工程先例）**：验证器不进化，用冻结准则验证之（SBST 允许套件进化、准则冻结） |
| 三 ML 动态基准 | benchmark | 评分函数＋judge | 污染／饱和／judge 漂移 | held-out 隔离＋客观答案锚＋轮次冻结档案＋外部熵源刷新 | **可且已是日常**——代价：可比性靠锚题链、任务被锚收窄、依赖外部源 |
| 四 UED/POET | 环境种群（生成课程） | 准入判据＋novelty＋regret | 生成器漂移（太难/太易任务堆积） | minimal criterion（可学性锚）＋reward 冻结＋transfer 档案＋通道窄化 | **可（原生目标）**：受控共同进化＝目标冻结＋通道窄化，锚随种群受控移动 |

## 与 research_6/7 的交叉引用边界

- research_7 概念三（IRT 锚题等值/项目信息函数）↔ 视角三「历史题＝可比性锚」「饱和＝信息函数近零」——引用不重述。
- research_7 概念二（regret 理论/元过拟合/递归自食退化）↔ 视角三污染（元过拟合同构）、视角四 regret 生成锚——引用不重述。
- research_6 概念二（免疫 germline 冻结＋胸腺隔离）↔ 视角一生物 Red Queen 只补「共同进化病理学」面；免疫镜已供内容不重述。
- research_6 概念三（quant 数据分割/walk-forward）↔ 视角三 held-out 隔离同族——引用不重述。
- 「标准化考试题库更新/renorming」原生域：锚题等值已由 research_7 概念三覆盖，本材料不重复。

## R7 上报项

- Hall of Fame 概念首提归属（Cliff-Miller 1995/1996 提出 vs Rosin-Belew 1997 系统化）未逐页核对原始页码——按期刊题录＋二手归誉，B。
- PAIRED（Dennis et al., NeurIPS 2020）题录与 minimax regret 环境设计机制未逐篇核对——B 线索。
- Goodhart/Campbell/Strathern 原始题录未本次网核——按常识级引用，B。
- GAN experience replay 论文（arXiv 1811.11083）作者题录未核——内容已核 arXiv 页面（内容 A／题录折价）。
- Floreano-Nolfi 1997 cycling 的「对旧冠军战绩周期性回退」曲线数据未取数——题录 A、细节 B。
- 变异测试工业化成本（mutant 数量爆炸→selective mutation/算子裁剪）未调研。
- Enhanced POET 的具体新失效与修复清单未逐节核对。
- Just et al. 2014「部分真实故障不被 mutant 代表」的具体比例未取数。

## Gaps

- **孔雀「信息隔离」与四域防护的结构关系未裁决**：空间隔离（免疫胸腺，research_6）／数据分割（quant，research_6）／历史档案（视角一）／目标冻结＋通道窄化（视角四）——是同一防护的四种实现还是不同防护，材料只并列呈现，是 human 裁决位（硬约束不代答）。
- **视角一「完备评价集」在 rick 域的可满足性未知**（任务全集只能增不能全）——单调档案可能是最可移植形式，代价＝档案无限增长（LAPCA 有界化换损失保证）→ 取舍需 human 决策。
- **视角二对 harness 域的可移植性依赖「LLM-harness 的 mutant 算子」存在**——语义等价变异无现成算子（缺口非结论）。
- **视角三外部熵源前提**：rick super 无 arXiv/新闻类外部源时滚动刷新不可移植——四元组任务的「目标/信号」从何持续供给是缺口。
- **视角四「可学性锚随种群移动」是否算受控选择器漂移**——该域视为特性、孔雀原则视为风险——两种结构的对齐是 human 产出位。
- **各视角收敛/发散特征只描述了特征本身**——哪个进 human 强相关清单、哪个进弱相关清单及理由，是 human 构建视角步骤的产出位（硬约束执行）。

（材料完 — 视角候选供给件，强/弱相关领域清单及理由由 human 自己产出）
