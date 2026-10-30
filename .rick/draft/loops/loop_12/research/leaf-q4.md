# leaf-q4：原创性如何发生 + tools-for-thought 传统（事实记录，无建议）

调研方式：2026-10 在线核验（Crossref/OpenAlex 书目与被引记录、官方存档全文、作者站点、原文 PDF）。等级：A=一手著作/同行评审；B=权威综述/官方文档，或作者一手但非同行评审（注明）；C=仅线索。

## ① 创造力研究核心结论

1. **Boden 三分类［A］**：组合型（熟悉概念的新组合）、探索型（按既有规则在概念空间内搜索）、变革型（改变空间自身的生成规则，使旧空间中"不可能"的想法变为可能；impossibilist vs improbabilist）。变革型被论证为罕见：须对规则动手术（通常一次只改一个维度），历史级实例（放弃调性、非欧几何）极少，多数创造属前两类；另区分个人原创/历史原创。该框架为概念分析而非实验结论。
   信源：Boden, *The Creative Mind*（1990；2nd ed. *Myths and Mechanisms*, Routledge 2004）；Boden, "Creativity and artificial intelligence", *Artificial Intelligence* 103(1-2):347-356 (1998), DOI 10.1016/S0004-3702(98)00055-1（被引 840+，OpenAlex）。

2. **Gentner 结构映射［A］**：类比迁移的是关系结构而非对象表面属性；合法映射须满足结构一致性（一一对应＋平行连通性）与系统性原则——由高阶关系统摄的关系系统优先被映射、更易习得与传达。证据来自类比解释、匹配任务与发展研究。
   信源：Gentner, "Structure-Mapping: A Theoretical Framework for Analogy", *Cognitive Science* 7(2):155-170 (1983), DOI 10.1207/s15516709cog0702_3。

3. **Dunbar 原位研究［A］**：对 4 个分子生物学实验室例会做约一年录音编码：真实科研中类比高频出现；但绝大多数为"局部"类比（同领域邻近问题），跨领域远程类比很少直接生成假说、多用于向听众解释，用于解题时常失败。此即"类比悖论"：自然情境中类比易而有效，实验室中自发迁移率低（归因于局部性与编码深度差异）。
   信源：Dunbar, "How Scientists Really Reason", in Sternberg & Davidson (eds.), *The Nature of Insight*, MIT Press, 365-396 (1995), DOI 10.7551/mitpress/4879.003.0017（被引 759+）；Dunbar, "The Analogical Paradox", in *The Analogical Mind*, MIT Press, 313-334 (2001), DOI 10.7551/mitpress/1251.003.0013。

4. **Sio & Ormerod 酝酿效应元分析［A］**：总体存在正的酝酿效应；调节变量——发散思维任务获益大于语言/视觉顿悟任务；酝酿前准备期越长效应越大；酝酿期高认知负荷填充任务削弱效应；低负荷任务优于纯休息（语言顿悟任务）。作者结论：存在多重机制（知识性 vs 策略性过程），作为实用技术须谨慎设计条件。
   信源：Sio & Ormerod, "Does incubation enhance problem solving? A meta-analytic review", *Psychological Bulletin* 135(1):94-120 (2009), DOI 10.1037/a0014212；官方摘要经 Lancaster EPrints 核验；被引 547+（OpenAlex）。

5. **Kauffman 相邻可能 / Johnson 平台论证［A / B(大众非虚构)］**：Kauffman——生物圈与经济持续扩张进"相邻可能"（由当前状态一步可达的新可能性集合），创新是逐级解锁而非凭空跳跃。Johnson 将其引入创新史：好点子来自相邻可能的解锁、缓慢直觉在"液态网络"中的碰撞；"平台"（珊瑚礁、城市、GitHub、YouTube 的层叠技术栈）使他人可在其上再建造，指数级打开相邻可能。
   信源：Kauffman, *Investigations*, Oxford University Press (2000)；Johnson, *Where Good Ideas Come From*, Riverhead (2010)。

## ② tools-for-thought 传统

6. **Engelbart 1962［A］**：智力效能是"人＋语言＋artifacts＋方法＋训练"系统（H-LAM/T）的涌现属性，个体现有效能"几乎不可能是原生智力受限"；放大而非替代——"intelligence amplification 并不意指提升原生智力"，而是将能力组织进更高层协同结构；bootstrap：先取最快收益，再用提升后的效能追求后续收益（工具开发↔工具使用的再生循环，报告第 IV 节 Basic Regenerative Feature）。
   信源：Engelbart, *Augmenting Human Intellect: A Conceptual Framework*, SRI Summary Report AFOSR-3223 (1962)，原文 PDF（144 页）在线核验；实证载体 NLS：Engelbart & English, "A Research Center for Augmenting Human Intellect", AFIPS FJCC (1968), DOI 10.1145/1476589.1476645。

7. **Licklider 1960［A］**：人机共生是介于"机械延伸人"与"纯 AI"之间的过渡期形态；分工——人设定目标、假说、标准并做评估，机器承担可程序化的例行工作，使计算机进入"formulative"（问题成形）阶段而非只解已形式化问题；其 1957 年自我时间分析：约 85% "思考时间"耗于为思考做准备的文书性操作；列出分时、记忆组织（trie memory）、目标式指令语言、I/O 设备为实现前提。
   信源：Licklider, "Man-Computer Symbiosis", *IRE Transactions on Human Factors in Electronics* HFE-1:4-11 (1960), DOI 10.1109/THFE2.1960.4503259；全文经 MIT CSAIL 存档核验。

8. **Michael Nielsen［B(作者一手自出版，非同行评审)］**：(a) mnemonic medium——Quantum Country（与 Matuschak, 2019, quantum.country）将间隔重复嵌入阅读，核心宣称"媒介使记忆成为选择"；(b) 记忆工程学（ltm.html, 2018）——作者自报数据：约 10,000 卡/2.5 年、日均 15-20 分钟、20 年保持成本约 4-7 分钟/卡（常规复习 2 小时+/卡），并论证记忆是问题求解与创造力的组件；(c) "Thought as a Technology"（外部表征/记号系统改变可思考内容的边界）——原 michaelnielsen.org/blog 链接已失效（域名现停放），未能在线核验，按二手线索记录［C］。
   信源：quantum.country/qcvc（在线核验）；Nielsen, "Augmenting Long-term Memory" (2018), augmentingcognition.com/ltm.html（在线核验）；"Thought as a Technology"（自出版，链接失效）。

9. **Andy Matuschak［B，作者一手工作笔记，非同行评审］**：evergreen notes——笔记原子化、概念导向、密集链接、跨项目演化；主张"重点是更好思考而非更好记笔记"；自认承自 Zettelkasten（Ahrens 2017；Luhmann 1992）。mnemonic medium 首用于 Quantum Country，扩展为 Orbit（withorbit.com，嵌入式 SRS 平台）。作者自报效果证据：完成复习会话者可靠保持内容；2019H1 读完文内层的读者中 29% 完成 1 个月复习层；至 2020Q1 数百用户展示 1+ 月保持；单条提示中位复习 6 秒；代价 35-50% 阅读时间开销；"跨领域泛化尚无证据"；采用障碍包括习惯负担与内容临界规模。
   信源：notes.andymatuschak.org（Evergreen_notes、Mnemonic_medium、Orbit、"Effects of the mnemonic medium on reader memory"，在线核验）；Nielsen & Matuschak, "How can we develop transformative tools for thought?" (2019, numinous.productions/ttt，现 404)。

## 结论
1. 原创性实证主要由组合、探索、类比迁移与相邻可能解锁构成；变革型创造被论证为罕见。
2. 酝酿效应真实但条件敏感：充分准备、低负荷间歇、发散任务时效应最大。
3. TFT 传统一致主张放大人机系统而非替代个体；已验证杠杆是记忆与外部表征。
