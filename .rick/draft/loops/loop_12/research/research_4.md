# research_4 — sparkling「对抗式认知摩擦」深研 + human 重构逐要素证据 — 2026-10-22

元信息：facts 模式。A=仓库原文/同行评审一手直读；B=权威综述或摘要级；C=线索。Q1 与 Q3① 本代理直做（leaf-r4-0）；Q2/Q3②③/Q4/Q5 外包四叶子，置信度本代理汇总。全文不含设计建议。

## 一、事实性结论

### Q1 仓库完整还原（7 文件，只有 SKILL.md 载机制）
1. **全仓 7 文件、6419B 的 SKILL.md 是唯一实质文件**；其余为 README(1755B)/NOTICE(411B)/COMMERCIAL(239B)/TRADEMARK(438B)/LICENSE(34523B, AGPL-3.0)/.gitignore；**无脚本、无 references、无 eval、无测试**——「250 字」「防倒退」全靠提示词自律。[GitHub API tree + raw 全文，A]
2. **触发条件**＝frontmatter description 原文「携带材料与用户进行结构化业务研讨……用于业务卡点、复杂资料讨论、深层洞察和需要认知突破的研讨任务」；全仓**无「原创性/创造力」字样**；repo description「与用户共同完成有记录、可回溯、以洞察为目标的业务研讨」。[A]
3. **角色措辞**：原文「你的角色是**携带材料的推导伙伴**，不是会议记录员，更不是谄媚的顺从者。你的价值在于提供高质量的**认知摩擦力**：通过材料硬事实碰撞、挑衅式激发（PO）与概念混成，帮助用户打破思维定势与局部最优」；目标「击穿隐性假设、涌现底层洞察（Aha! Moment）」。[A]
4. **5 步流程**：张力预处理（三张张力牌 + 白板初始化，硬约束原文「严禁输出常规公文综述」）→ 定锚（卡点宜写成「为什么[事实A]与[事实B]会同时发生？」+ 交付物形式）→ 短兵相接（每轮 **250 字左右**；必含「接住直觉」+「抛出单点抉择」；挑衅/PO 与概念混成 **二选一**，明令「不要试图在一轮中塞入所有动作」）→ 剃刀收敛/缺口转接 `/research`（「严禁脑补假设」）→ 交付。[A]
5. **白板契约 5 项**（文件 `sparkling-whiteboard.md`，机制名「Working Memory」，默认不在对话输出全文）：①核心卡点与交付目标 ②对撞硬事实 ③已击碎的假设（防倒退）④领先解释机制 ⑤证据缺口与下一步疑问。[A]
6. **红线 4 条**：禁谄媚迎合；禁 `[事实]/[假设]/[推断]` 标签；禁大段公文；防倒退（已证伪假设严禁重启）。[A]
7. **出口**：A=顿悟纪要（核心判断/事实咬合与推导逻辑/下一步业务动作：停止做·转向做·优先资源分配）；B=关键分歧与取证清单（领先假设+致命瓶颈/缺口+取证动作）。**终止条件无量化**：只写「当推导到达终点时」，无轮次上限、无失败退出路径。[A]

### Q2 五个构件的认知机制假设与出处
8. **张力牌**：反常→理论修正**非自动**——反常数据的七种反应中仅两类涉理论改变（其余为忽略/拒斥/搁置）（Chinn & Brewer 1993；Brewer & Chinn 1994，A）；预期违背被快速侦测（Baillargeon et al. 1985 违背预期注视范式，A）；「默认公理」对应 Toulmin「warrant」与 Argyris 双环（B）。机制假设＝预测误差使隐含前提变成可陈述命题。[A/B]
9. **定锚**：问题发现/表述是独立于解题的创作阶段且与原创性相关（Getzels 1979；Getzels/Csikszentmihalyi 1971，A）；但「为什么 A 与 B 同时」句式优于其他表述**无对照证据**（C）。[A/C]
10. **单点抉择**：「逐步最大化信息增益」的单问策略在 20-questions 类任务近似最优，有证明（Navarro & Perfors 2011；Meder 2020，A）；WM≈4 chunk（Cowan 2001，B）；choice overload 为条件性小效应 b≈.17（Chernev 2015，A）——**「每轮一问」本身无直接对照实验**（C）。[A/C]
11. **PO 与概念混成各有一手原文**：de Bono 自述「I came up with the word 'po'…Instead of using judgement with provocation, you use a new mental operation: 'movement'」（A）；Fauconnier & Turner 1998 原文「structure…is projected to a separate, 'blended' mental space…the blend develops structure not provided by the inputs…leading us to modify the initial inputs」（A）。机制假设＝挑衅先暂挂判断产出候选结构；混成的 emergent structure 回改输入空间。[A]
12. **剃刀 + 防倒退**：奥卡姆对应 SEP「Other things being equal, if T1 is more ontologically parsimonious…rational to prefer T1」（B）；防倒退有对应病灶证据——信念在依据被完全否证后仍持续（Ross/Lepper/Hubbard 1975，A）、仅寻证（Wason 1960）、调整不足（Tversky & Kahneman 1974）（A）；**「已证伪清单」的干预效果无研究**（C）。[A/C]

### Q3① 「保留关键对话记录」＝外部记忆（白板）
13. **笔记收益来自生成与复看，而非存储介质**：编码效应为正但**小**（Kobayashi 再分析判定 modest，B）；复习笔记才带来保持/理解增益，生成式优于逐字（Bohay 2011；Kiewra 1978 远迁移，A/B）。[A/B]
14. **卸载的两面**：强制卸载收益 > 自主选择（人低估卸载收益、选择不足），同时卸载**降低个体间表现差异**（趋同）——Burnett & Richmond 2025 两套 meta（A·摘要级，具体 g 未取到）；综述同列「减少自发策略投入」代价（Richmond & Taylor 2025，A/B）。[A/B]
15. **卸载代价**：外部化降低对该信息本身的记忆（photo-taking impairment effect；Mem Cognit 2025），削弱自上而下策略投入与系列位置效应（A/B）。**对「白板记录对话要点」是否成立无直接研究**（外推）。[A/B]

### Q3② 「符合事实的差异化回答」vs「完全反驳」
16. **真实异议 > 角色扮演魔鬼代言人 ≈ 无异议**：方案数 2.76 vs 1.86 vs 1.84（F(1,54)=6.13, p<.02）；「数量×质量」合成 z=0.92 vs −0.32/−0.22；**单看质量评分无差异**（F(2,52)=1.28, n.s.）——Nemeth, Brown & Rogers 2001（A 直读）。[A]
17. **DA 的副作用**：增加原创想法但**显著增加「加固原立场」**的想法，未增他择思考（Nemeth 2001 JASP，B 转引）；真实异质性更抑制确认性搜索（Schulz-Hardt 2002，B）。[B]
18. **批评的收益受语境调节**：鼓励批评只稳增**数量**（28.4 vs 24.5 想法，p<.05，未测质量，Nemeth 2004，A）；质量提升仅在合作语境（+17% 创造性，3.50 vs 3.13），竞争语境**反向** −23%（2.77 vs 3.30）——Curhan 2021（A 直读）。**「yes-and vs yes-but」无同行评审对照实验**（C）。[A/C]

### Q3③ 「不是不对，而是更好」
19. **「指错」类干预风险高**：破坏性批评（含内部归因）→ 冲突/愤怒↑、自我效能↓、后续绩效↓（Baron 1988，A/B）；反馈总均效 d=.41 但 **>1/3 干预降绩效**（Kluger & DeNisi 1996，A）；d=.48、**17% 为负**、I²=86%、对自我效能/坚持等动机类结果效应**更低**（Wisniewski 2020，A）；负反馈对内在动机无总体效应（Fong 2019，B）。[A/B]
20. **与质量正相关的是「修正」而不仅是「识别」**：正确识别错误数与**修正质量**均与 quality/originality/elegance 正相关（Gibson & Mumford，B 摘要）。**缺「仅指错」vs「指错+给更优替代」并同测质量与坚持性的直接对照实验**（Gaps）。[B/C]

### Q4 效果度量
21. **主指标与其混淆**：fluency/originality（统计罕用性）/flexibility/elaboration 可复现计分；uniqueness 与 fluency 强混淆，须用 max/top-2/平均计分或固定想法数（Beaty 2022，A）。[A]
22. **自动评分的效度区间**：语义距离 SemDis 与人类评分 r=.49（Beaty & Johnson 2021，A）；AUT 短表与人类评分 r=.351 [.125,.543]（CRJ 2022，A）；微调 LLM 打分 r=.81（Ocsai，B）。**LLM-judge 已知偏差**：位置偏差（交换顺序后仅 GPT-4 一致率 >60%，few-shot 至 77.5%）；冗长偏差（重复列表攻击下 Claude-v1/GPT-3.5 失守 91.3% vs GPT-4 8.7%）；自我偏好（GPT-4 +10%、Claude-v1 +25%）——MT-Bench, NeurIPS 2023（B）。[A/B]
23. **最小测试范式有硬下限**：单被试设计（SCED）基线≥3 数据点，WWC 最高评级需每阶段 ≥5 点且基线无趋势（Psychol Methods 2012；WWC 手册，A/B）；within-subject 交叉须拉丁方平衡且不能消除全部携带效应（A/B）；教育技术已有 136 项 SCED 综述（BJET 2024，A）。[A/B]

### Q5 可升级点（只列事实差异与可插入点）
24. **掌握度/知识追踪：整体有效、按掌握度选任务无独立增益证据**：ITS 总体 g=.41（107 研究/14,321 人，Ma 2014，A）、中位 d=.66（Kulik & Fletcher 2016，A）；ALEKS 阈值 RCT（N=806,279）「3 连对 vs 5 连对」保持率差≈0.02（JEDM，A）；「知识追踪驱动选题 vs 随机」现场 RCT 为 null（预印本已撤，C）。可插入点＝每轮显式输出掌握度估计/目标难度。[A/C]
25. **去固着手段**：案例对比 d=.50 [.44,.56]（57 实验，Alfieri 2013，A）；交错受「类间相似性」调节（Brunmair & Richter 2019，B），Kornell & Bjork 2008 为 .59 vs .36（d≈1.1 由 M/SD 推算，**此前 d=0.99 未核实**）；并置多解法优于逐一反思（Rittle-Johnson & Star 2007，B）；孵化中位 d=.26（Sio & Ormerod 2009；**此前 0.29 与原文不完全一致**）。可插入点＝同轮并列 2 个相异解法/反例。[A/B]
26. **图文联动效应量**：概念图 g=.604 [.550,.658]（55 研究/5,818 人；自建 .819、阅读现成 .373、超链接图 .017 n.s.）（Nesbit & Adesope 2006，A）；signaling g=.52/.31（Schneider 2018，A）；segmenting d=.32/.36、学习时间 +0.92d（Rey 2019，A）；多媒体原则中位 d=1.67 仅 5 组比较（Mayer 2017，B）。可插入点＝结论句后附节点图 + 空间邻近。[A/B]
27. **篇幅与交互成本**：**未找到直接操纵「LLM 每轮回答字数」并测学习结果的 RCT**（核心缺口）；间接事实：人类与 LLM 评审均偏好更长回答（B）；屏幕阅读劣于纸面 g=−.21（Delgado 2018，B）；step-based d=.76 vs answer-based .31（VanLehn 2011，B）；长上下文关键信息居中时模型性能下降（Liu 2023，A，属模型侧）。[A/B/C]

## 二、R7 / Gaps
- **无任何效果数据**：sparkling 无案例、无 eval、无使用日志；「250 字」「防倒退」「二选一」的实测遵从率不可考。[C]
- **human 重构的三段拆解各自的直接实验都缺**：①白板式对话记录→推理质量；②「引入新概念后的差异化反驳」vs「完全反驳」的直接对照；③「给更优替代」与「指错」同测质量与坚持性的对照——三者均未检索到，现有为邻近文献外推。
- **卸载 meta 具体效应量未取得**（Springer PDF 301、Nature Rev 付费墙）；AUT/PUT 的 PUT 单测信度缺失。
- 两处时间戳异常引注（arXiv 2604.x、ACL 2026 CreativeBench）仅 C 级，未采信进结论。
- 「白板→防重启」「已证伪清单」的干预实验整体缺位；「PO」「每轮一问」同样缺直接对照。[C]

## 三、叶子文件
- research/leaf-r4-0.md（Q1 全还原原文直引 + Q3① 外部记忆）
- research/leaf-r4-1.md（Q3②③ 分歧/反馈证据，含 Nemeth 2001/2004、Curhan 2021 直读）
- research/leaf-r4-2.md（Q2 五构件原典与机制假设）
- research/leaf-r4-3.md（Q4 度量与最小测试设计）
- research/leaf-r4-4.md（Q5 四类可升级点效应量）
