# Research: [B] 学习环节「大纲生成 + 教学内容生成」技术盘点（仅还原事实）

## 一、课程序列化与学习路径规划

1. KST/ALEKS 用「知识状态集 + 题目偏序（surmise relation）」建模，单学科知识结构约 200–1000 题；前提=题目可枚举、偏序可专家标定，再用贝叶斯由作答反推最可能状态。[A] aleks.com/about_aleks/Science_Behind_ALEKS.pdf
2. ALEKS 之后改用「学习序列」（antimatroid 的 basic words）替代纯偏序，可表达任意外延知识空间，两条序列即可生成整空间。[A] arXiv:0803.4030
3. 概念级先修自动推断：用 MIT OCW/Caltech/CMU/Princeton 课程目录，MIT 上 CGL.Rank 达 AUC 0.96 / MAP 0.46，优于 1NN(0.76)、SVM(0.78)；前提=有课程描述文本+部分课程级标注。[A] JAIR 10.1613/jair.5002
4. 先修关系公开基准：654 门课、861 条课程级边、1008 对人工标注概念对（11 所美国高校 CS 目录）。[A] AAAI 2017 (ojs.aaai.org 10550)
5. 路径推荐综述拆为三组件：知识追踪/认知诊断定状态 → 图结构生成/优化路径 → 多维评测；算法族=图搜索、MDP+RL、层次 RL。[B] MDPI Electronics 15(1):238；CIKM'23 Graph-Enhanced HRL
6. 可汗学院以「skill 达 proficient/mastery」而非时长度量；三年准实验（学生自身对照）显示每年多掌握 1 个 skill ≈ 目标增长达成率 +0.5 个百分点。[B] blog.khanacademy.org + EC-TEL 2024
7. Duolingo 早期 skill tree=语法/主题技能+strength meter，2022 后改线性 path 并刻意混排概念；HLR 半衰期回归使回忆预测误差降 45%+。[A/B] ACL 2016 (P16-1174)；blog.duolingo.com
8. EduPlanner 用多 agent（evaluator/optimizer/question analyst）+Skill-Tree+CIDDP 五维评价，在 GSM8K/Algebra 验证「评价+优化」闭环；但为 LLM 自评，未测学习增益。[B] arXiv:2504.05370。检索未发现名为「EDUPLAN」的课纲 benchmark。
9. LLM 生成整专业课程表/大纲的横向评测多用「与现有大纲相似度 + 专家 Likert 评分」；GPT-3.5/4、Bard、Gemini 产出结构完整但趋同、缺校本特色。[B] IEEE TLT 2025 (10.1109/tlt.2025.3587081)；Peer.ASEE 56052

## 二、图解 + 文字的教学内容生成

10. Mayer 原则 2025 元分析（92 篇/181 研究/591 效应）：总体 g=0.37；最强=去除 seductive details 1.00，其次 modality 0.82、personalization 0.70、multimedia（语词+图）0.68、句级 coherence 0.63；文本+图 0.45；VR 0.17 不显著；segmenting、contiguity、cueing、voice、redundancy 不显著或很小；主动生成类（self-explanation 0.46、testing 0.41）强于版式类（scaffolding 0.38）。[A] Educ. Res. Rev. 49:100730
11. 同作者早期自报中位数更高：multimedia d=1.67、coherence 0.97、modality 1.02；差异前提=新元分析纳入更多媒体类型与人群，引旧值会高估。[A] Mayer, JCAL 2017
12. 静态图形对阅读理解：39 项实验、2103 人，Hedges g=0.39（95%CI 0.26–0.51）；年级不显著调节；开放题/混合题型收益大于判断题；图片/图示/流程图之间无显著差异。[A] AERA Open 10.1177/2332858420901696
13. Graphic organizer 元分析：学生自建 g=0.59（记忆,k=44）/0.63（理解,k=68）；教师提供 g=0.70（记忆,k=37）/0.53（理解,k=49）；有发表偏倚与小样本效应，效应量可能被高估。[A] Educ. Psychol. Rev. 10.1007/s10648-025-10084-6
14. dual coding/picture superiority 的记忆优势稳定，但机制有争议（distinctiveness 可替代解释；aphantasia 者仍现该效应）——「图有效」不能直接归因双通道编码。[A] QJEP 2024 (10.1177/17470218241235520)；Neuropsychologia 2026
15. 学术图示代码生成：MathemaTikZ（TikZ，源自 Illustrative Mathematics）最好模型仅 73.9% 成功率，按可视化类型差异显著。[A] ACM L@S 10.1145/3698205.3729558
16. LLM 直出 SVG 数学提示图（GPT-4o+ICL，VQA 评测）：语义 0.80（真值 0.85）、句法 0.76（0.79）；改 DALL·E 3 直出像素图基本不可用（臆造数字、伪 3D）。失败模式=紧贴题面文字、丢失教学意图（如未把 2/2 拆成 1/2+1/2）。[A] arXiv:2503.07429
17. chart-to-code：ChartMimic（4800 三元组）上 GPT-4o Direct Mimic 82.2、Customized Mimic 61.6（0–100）。[A] ICLR 2025
18. NL→Mermaid 已有专用基准 MermaidSeqBench（132 样本、LLM-as-judge，指标含语法正确性/激活处理/错误处理/可用性），报告模型间显著差距。[B] arXiv:2511.14967
19. 物理图生成经专家评审仍出错：ChatGPT/Gemini/Copilot 的受力图存在力的识别/指向错误与要素缺失。[A] Phys. Educ. 10.1088/1361-6552/ae56aa、ae4c11

## Gaps

- 「EDUPLAN」作为课纲 benchmark 未检索到，最接近者为 EduPlanner（arXiv:2504.05370），需确认指代。
- Mayer 新元分析中 segmenting/contiguity/spatial contiguity 的具体 g 被付费墙遮挡，仅知「不显著或很小」。
- 缺「LLM 生成课纲 → 学生学习增益」RCT；现有评测仅专家评分或 LLM 自评。
- Manim/Graphviz/Excalidraw 缺专项量化评测，现有评测集中于 TikZ/SVG/Mermaid/图表代码。
- Coursera 是否公开先修图、Khan Academy 知识图谱结构均无一手文档，仅产品描述层证据。
