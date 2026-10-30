# 研究：学习环节「测评与自适应」事实盘点（leaf-r3-1）

硬事实集中在 CAT 效率、KT 精度与前提、阈值来源薄弱；80–90% 掌握阈值与"FSRS 少复习 20–30%"属约定/流传数字，缺一手实证。A=同行评审一手，B=权威文档/综述，C=线索。

## 1 目标对齐与需求引出
1. ABCD 目标（Mager 1962）：目标须含可观测行为+条件+达标标准，把"学会"变为可判定观测项；前提是行为可外显。[B]
2. Bloom 修订版（2001）：记忆→创造 6 认知层×事实/概念/程序/元认知 4 类知识，只定层级，不产生阈值。[B]
3. 需求分析＝现状与目标差距（gap analysis），无统一量化标准；GenMentor（WWW'25）以"目标+简历"推技能集 S′，与自评 S₀ 求差 ΔS=S′−S₀ 后人工确认，**无初始测验**；目标→技能 Recall .67/Precision .63，人工研究 n=20。[B: 小样本]

## 2 ZPD 初始化：诊断测评
4. CAT 优于固定测验的前提：一维 IRT+已标定题库+按 SE 终止；实测 CAT 均 5 题（SD1.9）vs 7 题短表，r=.90、T 分差<0.5，仅量表两端更准 → 能力居中时收益≈0。[A]
5. CAT 终止取 SE≤3.0（≈信度 .90）或最大题量；放宽 SE 后题量收益递减。[A]
6. BKT 每 KC 四参数（L0/T/G/S），需 KC 标注；良好调参 AUC .73，加遗忘 .83，按题目索引 .90 ≥ DKT .86。[A]
7. DKT 原报 ASSISTments AUC .86，复现指其高估；AKT（KDD'20）最多 +6% AUC，需海量交互、解释性弱。[A/B]
8. LLM-KT（LKT）用题面文本缓解冷启动，ASSIST09 AUC≈.83–.84，增益约 2–5%，跨榜切分不一致。[B/C]
9. 前置关系可由作答数据做结构/因果发现或取自 KST；无 KC 标注须先建 KC 图。[B]
10. pretest 非中性：先测后学对 ≥24h 保持有 d=.44~>2.0 增益，但主要限于被测项，概念性 pretest 可失败 → 诊断前测会抬高基线。[B]
11. 口头回忆/教别人：自解释质量与理解增益相关（过程指标，非能力估计）；learning-by-teaching g=.27（39 研究）、准备教 .35、教+.56（28 研究）；teach-back 16 项 RCT 仅 8 项（N=721）测知识获得且方法学受限 → 作能力测评缺效度证据。[A 收益/B 效度缺位]

## 3 掌握度判断与推进/倒退
12. Bloom（1968）仅称"90% 以上学生可掌握"，未定数值阈值；Guskey：80% 是常见约定，百分比切点本质任意。[A/B]
13. 阈值实证弱：70/80/90% 三组对照，90% 仅改善部分测验，**期末成绩无差异**。[A]
14. Khan：<70%=Attempted、70–99%=Familiar、满分→Proficient/Mastered，掌握度只计后两级；"90%"为官方口径，非实验结论。[B/C]
15. ITS 推进：Corbett & Anderson 1994 model tracing＝步级诊断+知识追踪概率，用于即时提示/跳练；前提是领域生产式规则库。[A]
16. KST/ALEKS：知识状态=题目子集；外缘=可学、内缘=刚会易失；Markov 自适应 25–30 题定状态（Knowledge Check 20–30 题）；需先建知识结构。[A/B]
17. SM-2：EF′=EF+(0.1−(5−q)(0.08+(5−q)0.02))，EF≥1.3；I₁=1、I₂=6、I(n)=I(n−1)·EF，q<3 重置；不输出概率。[A]
18. HLR：p=2^(−Δ/h)，h=2^(Θ·x)；Duolingo 数据预测误差降 45%+。[A: ACL16]
19. FSRS：R=(1+FACTOR·t/S)^(−w20)，FACTOR=0.9^(1/decay)−1，S=遗忘至 90% 天数；349.9M 复习上 FSRS-7 logloss .3363/AUC .7243；对 Anki SM-2 在 99.6% 用户 logloss 更优（SM-2 非概率模型，比较有保留）。[B]
20. 85% 规则：特定学习算法类下正确率≈85%（错误率 15.87%）学习最快，可作难度靶，不可当掌握阈值。[A]

## 来源
Mager/ABCD https://www.td.org/templates-and-tools/magers-model-for-writing-learning-objectives ·
NIU https://www.niu.edu/citl/resources/guides/instructional-guide/writing-goals-and-objectives.shtml ·
GenMentor https://arxiv.org/abs/2501.15749 ·
PLUS-M CAT https://pmc.ncbi.nlm.nih.gov/articles/PMC5832493/ ·
CAT 终止 https://doi.org/10.7333/jcat.v1i1.16 ·
How deep is KT https://www.educationaldatamining.org/EDM2016/proceedings/paper_144.pdf ·
AKT https://arxiv.org/abs/2007.12324 · LKT https://arxiv.org/abs/2406.02893 ·
KST https://arxiv.org/abs/1511.06757 · ALEKS https://www.aleks.com/about_aleks/knowledge_space_theory ·
pretest 综述 https://link.springer.com/article/10.1007/s10648-023-09814-5 ·
Kobayashi https://doi.org/10.1111/jpr.12221 · teach-back https://www.hsrd.research.va.gov/publications/esp/teach-back.cfm ·
Bloom1968 https://www.uky.edu/~gmswan3/EDC608/Bloom_1968.pdf ·
Guskey https://www.ascd.org/el/articles/in-search-of-a-useful-definition-of-mastery ·
阈值实验 https://link.springer.com/article/10.1007/BF02768360 ·
Khan https://support.khanacademy.org/hc/en-us/articles/5548760867853 ·
Corbett&Anderson https://link.springer.com/doi/10.1007/BF01099821 ·
SM-2 https://www.super-memory.com/english/ol/sm2.htm · HLR https://aclanthology.org/P16-1174.pdf ·
FSRS 基准 https://github.com/open-spaced-repetition/srs-benchmark · FSRS 公式 https://github.com/open-spaced-repetition/awesome-fsrs/wiki/The-Algorithm ·
85% 规则 https://www.nature.com/articles/s41467-019-12552-4

丢弃：Expertium/Quanta/Verbamor/Yazu 博客（仅核验"20–30% 少复习"溯源失败，未采信数值）；sota2 榜单（切分不可核）；ResearchGate/知乎转载。

## Gaps
1. "80–90% 掌握阈值"无一手实验推导：Bloom/Guskey 视之为约定，唯一直接对照实验显示 90% 对期末无增益 → 属工程选择。
2. "FSRS 比 SM-2 少 20–30% 复习"溯源失败（官方基准当前无 SM-2 行，SM-2 不输出概率），不应引用。
3. free recall/teach-back/protégé 作测量工具的效度（信度、与 IRT 一致性）无系统证据；文献测的是学习收益。
4. "goal elicitation"缺同行评审规范术语与基准；LLM 侧样本普遍 <100，指标多为 LLM 判分。
5. CAT 在学科测评（非 PROMIS/PLUS-M 类）中"何时优于固定测验"的课堂头对头证据不足；建议补检索 EM:IP、JEM 2020 后。
