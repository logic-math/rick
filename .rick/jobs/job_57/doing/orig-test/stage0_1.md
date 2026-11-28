# stage0_1 调研素材——原创性思考与研究品味（实测首启）

任务：为正在设计「启发原创性思考」会话系统（SENSE 五阶段思考门禁 loop）的用户检索原创性/科研品味方向的启发素材。检索按四类启发组织：方法型/现象型/局限型/跨领域。

---

## 0. 先核用户引用的一手来源（ScholarCatalyst）

**ScholarCatalyst: A Benchmark for Retrieving Papers that Inspire New Research**（Wei Koh, Aakanksha Chowdhery, Akari Asai, Omar Khattab, Yejin Choi 等, Stanford IRIS Lab, arXiv:2610.02202, 2026）

- 定义 catalyst paper：「一篇其思想确实/本可以在项目早期推动项目前进的论文」。由 184 位项目负责人（亲手做过研究、了解项目每个环节的 lead authors）标注自己项目中真正起催化作用的论文，包括当时未遇到的论文。
- 关键数据（多来源交叉验证一致）：
  - **49.5% 的 gold/catalyst 论文来自项目自身主题领域之外**（"Inspiration arrives from the field next door in nearly half of all cases"）
  - 43.6% 的 subfield 级 positives 从未被源论文引用；uncited positives 中 67.8% 与源论文零参考文献重叠
  - 引用了的 catalyst 里，只有 46.2%（CoreQ）/34.0%（SubQ）获得 "uses"/"extends" 的引用意图标签
  - **硬负例（hard negatives）在 lexical 与 embedding 相似度上与 catalyst 论文同样接近甚至更接近 query**——主题相似度和引用状态都捕捉不到「启发性」
  - 现有系统（含 agent search）在此基准上表现差（"Why Agent Search Trails"）

来源：
- arXiv 原文: https://arxiv.org/html/2610.02202
- 项目页: https://ohmyksh.github.io/project/ScholarCatalyst/
- GitHub: https://stanford-iris-lab/ScholarCatalyst
- 解读: https://blog.pebblous.ai/blog/scholarcatalyst-inspiring-paper-benchmark/en/
- 作者页: https://dayoon-ko.github.io/publication/2026-arxiv-scholarcatalyst

用户转述的两点（49.5% 领域外；识别不对称）中：49.5% 属实且为论文一手数据；「识别不对称」更接近用户从 benchmark 设计（人能事后指认 catalyst，但机器难以检索到）提炼的机制性推论——ScholarCatalyst 的标注方式本身就是这个不对称的工程化：**研究者事后能识别哪篇论文催化了自己，却无法事前搜索到它**。

---

## 1. 现象型启发（一）：AI 生成研究思想的实验数据

**Can LLMs Generate Novel Research Ideas? A Large-Scale Human Study with 100+ NLP Researchers**（Chenglei Si, Diyi Yang, Tatsunori Hashimoto, Stanford, ICLR 2025, arXiv:2409.04109）

- 100+ NLP 研究者盲评 LLM ideation agent vs 专家自己写的 idea
- 结论：**AI 生成的 idea 在「novelty」上被评为显著高于人类专家，feasibility 略弱**
- 但去重后 agent 只产出约 200 个独特 idea 就开始重复——**多样性坍缩 / idea homogenization**；作者明确提出若学界依赖此类系统会导致思想同质化的风险
- 论文同时记录了评估方法的局限（idea 不执行难以评价值，即 novelty 判断 ≠ 价值判断）

来源：
- 原文: https://arxiv.org/abs/2409.04109
- ICLR proceedings PDF: https://proceedings.iclr.cc/paper_files/paper/2025/file/ea94957d81b1c1caf87ef5319fa6b467-Paper-Conference.pdf

## 2. 现象型启发（二）：新颖性-常规性双高的大规模证据

**Atypical Combinations and Scientific Impact**（Brian Uzzi, Satyam Mukherjee, Michael Stringer, Benjamin Jones, *Science* 342:468-472, 2013）

- 17.9M 篇论文（Web of Science 1950-2000 全库）的引用配对 z-score 分析
- 几乎普适模式：**最高影响力的科学主要以高度常规的既有知识组合为基础，同时嵌入极少数（在 90% 常规配对之上的 10%）高度非典型的知识组合**——新颖与常规必须双高，纯新颖组合反而失败
- 团队比独行作者高 37.7% 的概率在熟悉知识中插入新颖组合

来源：
- Science 原文: https://www.science.org/doi/10.1126/science.1240474
- PDF: https://www.kellogg.northwestern.edu/faculty/uzzi/htm/papers/Science-2013-Uzzi-468-72.pdf

---

## 3. 方法型启发（一）：问题选择的结构化启发式

**How To Choose a Good Scientific Problem**（Uri Alon, *Molecular Cell*, 2009）

- 给出一个二维问题选择图：**可行性× 兴趣/重要性**。右上区域（既可行又重要）是常见误区区……不，右上才是理想区；误区是留在「可行但无趣」的安全区
- 核心论点：选择问题是在这两个轴上做显式权衡，多数研究者卡在可行性强但增量性高的区域；「自我维持的激励结构」让研究者倾向于可行而非重要
- 值得注意：Alon 把问题选择描述为可教、可显式化的判断结构（而非神秘品味）

来源: https://www.cell.com/molecular-cell/fulltext/S1097-2765(09)00641-8

**Eric Kandel（2000 诺奖）访谈原话（NobelPrize.org）**：
> "Becoming a scientist … is a question of developing taste. … you develop taste by seeing what other people do, what are the really important questions, what's trivial and making important distinctions between them."
- 品味 = 通过大量看别人的工作，在「重要/琐碎」之间做出区分的能力；品味是被训练出来的区分能力，不是规则遵循。
- Tsung-Dao Lee 访谈则给出反面：问什么问题「cannot be specific. Each person is different」——提问无法被模板化。

来源:
- Kandel: https://www.nobelprize.org/prizes/medicine/2000/kandel/25039-interview-transcript-2000/
- Lee: https://www.nobelprize.org/prizes/physics/1957/lee/interview/
- 诺奖级直觉访谈研究: "A Nobel's eye view of scientific intuition" (Int. J. Sci. Ed., 1994), https://doi.org/10.1080/0950069940160406
- 诺奖得主问题选择实证: https://doi.org/10.51408/issi2025_092

## 4. 现象型启发（三）/局限型（一）：原创性的宏观成本曲线

**Are Ideas Getting Harder to Find?**（Bloom, Jones, Van Reenen, Webb, *American Economic Review* 110(4):1104-1144, 2020）

- 多行业/产品/企业证据：研究投入大幅上升的同时研究生产率急剧下降；维持摩尔定律所需的研究人员数量约为 1970 年代初的 **18 倍**
- 配套机制：**The Burden of Knowledge and the Death of the Renaissance Man**（Ben Jones, *Review of Economic Studies* 76(1):283-317, 2009）——知识存量越大，创新者教育负担越重，补偿方式是领域窄化+团队化

来源:
- AER 原文: https://web.stanford.edu/~chadj/IdeaPF.pdf
- Jones 2009: https://academic.oup.com/restud/article-abstract/76/1/283/1577537

→ 与用户「识别不对称」的张力：宏观上「主动找到」正变得越来越贵（ideas harder to find），这正是催化剂检索/启发系统存在的结构性理由。

## 5. 跨领域（一）：组织学习的探索-利用权衡

**Exploration and Exploitation in Organizational Learning**（James G. March, *Organization Science* 2(1):71-87, 1991）

- 探索= search, variation, risk taking, experimentation, play, discovery, innovation；利用= refinement, choice, efficiency, selection, execution
- 关键结构：**探索的回报不确定、遥远、常常为负**，利用的回报确定、即时、局部 → 任何评估系统（奖励/绩效评估）天然偏向利用，探索会被系统性挤出（"crowding out of the exploration of more distant opportunities"）
- 延伸（Levinthal & March；Ocasio & Rhee 2020）：组织学习的失灵常不是不会探索，而是评估机制奖励短期确定回报

来源:
- 原文: https://pubsonline.informs.org/doi/10.1287/orsc.2.1.71
- 综述: https://faculty.wharton.upenn.edu/wp-content/uploads/2013/04/Advancing.pdf
- March 与组织智能: https://bpb-us-e2.wpmucdn.com/faculty.sites.uci.edu/dist/a/737/files/2024/09/ocasiorhee2020.pdf

→ 与用户 SENSE 门禁设计的张力：五阶段门禁本质是一个「评估机制」；March 的警告是：**任何以「过关」为形式的选择机制都会偏袒确定、即时、局部的回报，从而系统性杀死远距离探索**——用户的门禁如果设计不当，会过滤掉它本想催生的原创性。

---

## 6. 现象型启发（四）：认知科学的原创性机制——酝酿效应与问题发现

**Does incubation enhance problem solving? A meta-analytic review**（Ut Na Sio & Thomas C. Ormerod, *Psychological Bulletin* 135(1):94-120, 2009）

- 117 项独立研究的元分析：**整体存在正的酝酿效应**（问题搁置一段时间后解题率上升），但效应受调节变量影响大：线索的存在（外展线索比无线索强）、问题类型、准备期长度
- 机制之争：选择性遗忘 vs 选择性激活/线索敏化 vs 无意识工作。后续实验：「酝酿收益只在被误导后才出现」（Ellwood et al. 2009, "Incubation benefits only after people have been misdirected"）——**酝酿首先解除的是错误定势，而不是无意识加工出新解**

来源:
- 原文: https://doi.org/10.1037/a0014212 ; PubMed: https://pubmed.ncbi.nlm.nih.gov/19210055/
- 机制研究: https://escholarship.org/uc/item/9q03k7z8
- 误导实验: https://pubmed.ncbi.nlm.nih.gov/17848028/

→ 与用户 loop 的张力：SENSE 的「判断矛盾→辩证逆转」两段在结构上对应「定势解除」；酝酿效应文献说这个环节的机制不是注入新内容，而是**解除已建立的错误定势**。

**The Creative Vision: A Longitudinal Study of Problem Finding in Art**（Jacob W. Getzels & Mihaly Csikszentmihalyi, Wiley, 1976；20 年追踪到 1985 报告）

- 1960 年代芝加哥艺术学院 31-36 名毕业年级学生实验：观察他们「摆静物—作画—完成」全过程的行为
- 核心区分：**问题求解者 vs 问题发现者**——问题发现者在作画过程中持续重构问题（中途调整物体、改变取景），不把「画什么」当作已给定
- 追踪结果：**在校期间的问题发现行为预测了 7 年后与 18-20 年后的艺术成就**（成为职业艺术家/艺术教师 vs 放弃艺术），而当时的画技评分不预测（20 年时唯问题发现行为显著区分仍在艺术界者）
- 相关综述：Reiter-Palmon & Robinson (2009), "Problem identification and construction: What do we know, what is the future?"

来源:
- MIT Press Reader 解读（Keith Sawyer）: https://thereader.mitpress.mit.edu/the-1960s-art-school-experiment-that-redefined-creativity/
- 原书: https://archive.org/details/creativevisionlo0000getz
- 20 年追踪报告（Getzels 1985）: https://journals.sagepub.com/doi/10.1177/019263658506948208
- 综述: https://digitalcommons.unomaha.edu/cgi/viewcontent.cgi?article=1027&context=psychfacpub

→ 跨领域张力（艺术创作领域）：用户的「问题还原」阶段若被设计为「把问题表述清楚」，与 Getzels 的发现冲突——问题发现是**持续重构**的过程而非一次性表述，且它本身可被观察、被外显为行为指标。

---

## 7. ScholarCatalyst 一手细节（识别 vs 搜索的实验证据，项目页+论文）

- Agent search 并不优于 embedding retrieval：0.42 vs 0.48 Recall@20；基于 Claude Fable 5.1 的 agent（训练中可能见过已完成的论文）也仅 0.51 R@20
- 最好的一般 dense retriever 只把 39%（Core Query）/51%（Subfield Query）的 catalyst 论文放进 top-20；grep 式搜索最差（8% gold 覆盖）
- **Gold injection 实验**：把漏掉的 gold catalyst 换入 reranker 的 top-20 候选后，强 reranker 大幅提升（Claude Fable 从 ~27% 升到 50% Recall@5）→ **摆在面前时能认出来，是搜索/召回环节找不到**
- 论文结论原话："**researchers may therefore be limited more by what they have seen than by how well they judge it**"（研究者更受限于见过什么，而非判断力如何）
- 启发类型学：catalyst 的 rationale 分九种类型；55.8% 的 rationale 同时适用多种类型——「启发性」是多维的，单轴相似度注定失效

来源: https://ohmyksh.github.io/project/ScholarCatalyst/ ; https://arxiv.org/html/2610.02202

→ 方法型启发（可迁移实验范式）：**gold-injection 对照**——检验一个系统瓶颈在「召回」还是「判断」，就把 gold 塞进候选池看判断端能吃到多少增益。这个范式可直接迁移到用户 SENSE 门禁的验收测试：门禁失败时，先测是「没摆到他面前」还是「摆到面前仍不认」。

---

## 8. 局限型启发（二）：识别不对称自身的边界——迟滞承认现象

**Sleeping Beauties in science**（Anthony F. J. van Raan, *Scientometrics*, 2004；扩展 Garfield 1970 的 delayed recognition 概念）

- 「睡美人」论文 = 发表后长期无人问津、之后几乎突然获得大量引用的论文（EPR 悖论论文为典型）。基于百万级论文的测量给出「唤醒概率函数」；后续 PNAS 2015（Ke et al.）与 2024 大规模复检确认其普遍性
- 张力：**识别不对称假设「摆在面前能认出价值」——但科学史上系统性存在『摆在面前也认不出』的时刻**，唤醒往往需要领域外的 prince（跨域引用者）到来

来源:
- van Raan 2004: https://link.springer.com/article/10.1023/B:SCIE.0000018543.82441.f1
- PNAS 2015: https://www.pnas.org/doi/full/10.1073/pnas.1424329112
- 2024 大规模复检: https://doi.org/10.1177/01655515241244462

## 9. 现象型启发（五）：LLM 对人类创意的同质化效应（人-机共创实验）

- **Homogenization Effects of LLMs on Human Creative Ideation**（Anderson et al., ACM Creativity & Cognition 2024，36 人对照实验）：LLM 当创意支持工具让用户**自评更有创意**、个体内多样性略增，但**用户之间**的想法显著同质化
- **Human Creativity in the Age of LLMs**（CHI 2025 随机实验）：暴露于 LLM 生成策略的参与者在**后续**独立任务中仍持续产出更相似的想法——同质化是**残留性**的，不随会话结束
- 大样本动态实验（800+ 参与者，40+ 国家，2025）：高 AI 示例暴露 → 人类想法多样性显著下降；meta 分析（19 研究 61 效应量）确认「个体略升、群体显著同质化」的小而稳健的模式

来源:
- https://dl.acm.org/doi/fullHtml/10.1145/3635636.3656204
- https://dl.acm.org/doi/full/10.1145/3706598.3714198
- https://dl.acm.org/doi/full/10.1145/3715928.3737481
- meta: https://www.tandfonline.com/doi/full/10.1080/0144929X.2026.2726451

## 10. 跨领域（二）：科学哲学——常规科学与革命的「本质张力」

**The Essential Tension: Tradition and Innovation in Scientific Research**（Thomas S. Kuhn, 1959 演讲 / 1977 文集）

- Kuhn：科学教育本质上是**收敛式**的（「教条式启蒙」，教科书+标准习题），常规科学的特征问题「几乎总是既有问题的微小修改重复」；但科学发现恰恰要求**发散式**思维能力
- 核心论点：**两种能力的张力不可消除**——只收敛则只见常规反常，只发散则无从积累深度；创新出现在「对传统的深度承诺」与「打破传统的能力」同时成立之处。Kuhn 原话大意：在科学里「用现有工具尽力而为几乎总是优于停下来 contemplating divergent approaches」——收敛是常态、发散是例外时刻
- Hanne Andersen 等分析：本质张力=常规科学的保存传统活动 vs 革命的破坏传统活动之间的张力

来源:
- Kuhn 原文 PDF: https://public.econ.duke.edu/~kdh9/Courses/Philosophy%20of%20Economics/Spring%202007/Kuhn%20Essential%20Tension.pdf
- 书目: https://press.uchicago.edu/ucp/books/book/chicago/E/bo5970650.html
- Andersen 分析: https://curis.ku.dk/ws/portalfiles/portal/226281123/Andersen_Second_Essential_Tension.pdf
- Kuhn 论创造力与传统教育: https://researchprofiles.ku.dk/en/publications/kuhn-on-creativity-and-tradition-in-education-2/

---

## 11. 候选张力映射（给最终回复用）

对用户两句话（A. catalyst 论文 49.5% 领域外；B. 识别不对称：摆到面前判断价值比主动找到容易）：

1. B 的可行性支持：ScholarCatalyst 的 gold-injection 实验（识别端 27%→50% R@5）+ 论文结论原话「研究者更受限于见过什么而非判断力」——不对称被实验量化，且架构上可利用（机器管召回，人管识别）→【可行性支持】
2. B 的方向调整：睡美人现象——识别依赖图式与时点，「摆在面前也认不出」系统性地存在 → 识别不对称有时效维度，门禁只和用户当下图式一样宽 →【方向调整】
3. A 的扩展/调整：Uzzi 2013——最高影响=高度常规+少量(≤10%)非典型组合；纯领域外/纯新颖组合反而失败 → 领域外素材需要领域内常规骨架做锚 →【方向调整】
4. SENSE 门禁 vs March：门禁=评估机制，March 的机制分析：任何过关型选择机制偏向确定/即时/局部回报，远距离探索被挤出 → 五段门禁若评的是「当下可辩护性」，会系统性杀死它想催生的原创性 →【方向调整】
5. 视角生产外包给 AI 的边界：Si et al. 多样性坍缩 + CHI 2025 残留同质化（AI 示例暴露后的想法趋同在会后仍残留）→ AI 给视角 vs AI 给张力素材是两种不同的系统 →【方向调整/方法借用】
6. 问题还原 vs Getzels：问题发现是持续重构过程且可被行为化观察，一次性「还原表述」可能固化问题 →【方法借用】
7. 判断矛盾/辩证逆转 vs 酝酿效应：逆机制的实验证据指向「解除错误定势」而非「注入新内容」（misdirected 后收益最大）→【方法借用/可行性支持】
8. 良质批判 vs Kandel：品味=可训练的区分能力（看大量他人工作+做重要/琐碎区分）→ 门禁本身可被设计成品味训练器而不仅是过滤器 →【方法借用】
9. Kuhn 本质张力 vs 辩证逆转：收敛（常规科学/范式深度承诺）与发散同时需要——辩证逆转不是抛弃传统而是「在深度承诺中打破」→【方法借用】

## 12. 引导性问题候选

ScholarCatalyst 证明「人更受限于见过什么而非判断力」（识别>搜索）；March 证明「任何过关型评估机制天然偏袒确定、即时、局部的回报」（评估杀死远距离探索）。SENSE 的门禁同时是这两者：它既是评估机制（有 March 风险），又把宝押在人的识别力上（有 ScholarCatalyst 支持）。当门禁在判断矛盾/良质批判阶段评估一个「远距离视角」时，它评的是用户的判断力，还是用户当下图式恰好装得下的东西？loop 里哪个机制负责让「此刻还认不出价值」的东西活到下一轮？

---

## 13. 方法型启发（二）：研究品味的操作化与可学习性（AI for research 前沿）

**AI Can Learn Scientific Taste**（arXiv:2603.14473）
- 把 scientific taste 显式定义为「**判断和提出具有长期科学影响潜力的研究想法的能力**」；提出 RLCF（Reinforcement Learning from Community Feedback）：用大规模科学社区反馈信号训练，证据显示 AI 可学判断+构思两种品味能力
- 来源: https://arxiv.org/html/2603.14473v2

**SciTaste-Bench**（arXiv:2610.28653）
- 六学科基准测 LLM 对论文的「研究品味」判断（300 题+5,700 备选池，1996-2023）——品味判断已成为可基准化的对象
- 来源: https://arxiv.org/abs/2610.28653

**TIGER: Learning from Researcher Taste to Generate Tailored Research Ideas**（arXiv:2611.02234）
- 前提：「LLM 能生成研究想法，但输出常不匹配**个体**研究者的品味」；用 400 真实研究请求 + 2,000 候选 idea，按每位研究者的标注训 per-researcher 奖励模型做个性化生成——品味是个体性的、可建模的
- 来源: https://arxiv.org/abs/2611.02234

**SciMuse**（arXiv:2405.17044）：58M 论文知识图谱+LLM 生成个性化研究想法，100 位研究组长评估。来源: https://arxiv.org/abs/2405.17044

→ 张力（【方向调整】）：RLCF/SciTaste 学的是**社区共识品味**；而 ScholarCatalyst 恰恰证明 49.5% 的催化剂在领域外、共识信号（引用、相似度）捕捉不到它们——用「学到的社区品味」当门禁/检索器，等于用导致催化剂稀缺的机制去找催化剂。TIGER 证明了品味可个体化建模，这是另一条路线。

---

## 14. 检索覆盖核对（四类启发 × 至少一检）

- 方法型：第 3 节、第 7 节、第 13 节（gold-injection / 问题选择图 / RLCF-SciTaste-TIGER / 人类盲评协议）
- 现象型：第 1、2、4、6、9 节（Si et al. 数据、Uzzi 双高峰、酝酿效应、问题发现预测生涯、同质化实验、ideas harder to find）
- 局限型：第 4、5、8 节 + 第 7 节 + 第 9 节（检索信号失效、agent search 不及 embedding、识别不对称的睡美人边界、多样性坍缩、novelty≠value）
- 跨领域：第 5 节（组织科学 March）、第 10 节（科学哲学 Kuhn）、第 6 节 Getzels（艺术创作）、第 3 节 Kandel 访谈（神经科学史）

落盘完毕。共 8 次检索 + 1 次 fetch（ScholarCatalyst 项目页）+ 1 次多源交叉验证。
