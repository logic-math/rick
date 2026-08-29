# 调研：human 判断力度量方法学（成熟方法盘点，仅事实+来源）

**总判**：候选框架中「正确性×赔率」有成熟数学对应（proper scoring rules／博彩回报），「影响力」对应决策分析的期望值/VoI 加权；AAR/复盘即 dream 回收环节的直接对应；但三项连乘偏离两类成熟范式（proper 评分只能是「所报概率×结果」的函数；EV 是加性加权），且会激励虚报影响力。

① **Tetlock / GJP**：IARPA ACE 锦标赛以 Brier 分对大量「可判对错、有截止日」的地缘问题排名；CHAMPS KNOW 去偏差训练（<1 小时）持续提升 Brier 6–11%；超级预测者=按 Brier 取前 2% 且跨年成绩稳定。数据要求：每人每期数十~上百条可解决概率预测。映射：rick 拍板须转成「概率+截止日+可验证命题」才可入库；1–2 周回收窗口比 GJP 典型期限短，可行但需命题设计。来源：Mellers et al. 2014, https://journals.sagepub.com/doi/10.1177/0956797614524255 ；训练效果 https://www.sas.upenn.edu/~baron/journal/16/16511/jdm16511.html ；超级预测者 https://stanford.edu/~knutson/jdm/mellers15.pdf

② **Brier 分数／校准曲线**：二次严格 proper 规则（诚实报概率是最优策略）；可分解为校准(reliability)+分辨力(resolution)+不确定度；可靠性图需按概率分档累计样本（每档若干十条）。映射：rick 的「归一化正确性得分」可直接用 2(p−y)² 或对数分；前提是拍板时存了数字概率。来源：Gneiting & Raftery 2007, https://sites.stat.washington.edu/people/raftery/Research/PDF/Gneiting2007jasa.pdf ；分解 https://www.cawcr.gov.au/projects/verification/reliability_resolution.html

③ **预测市场赔率**：价格=群体概率；Hanson LMSR 以对数评分做市。押注反共识方向且命中，回报 ∝ (1−p)/p，即「赔率×正确性」的标准实现；log 评分对低概率命中给对数级大分。映射：rick 的赔率项等价于「以共识价为基准的下注损益」，数学成熟，无需额外发明。来源：Hanson 2002, https://mason.gmu.edu/~rhanson/mktscore.pdf

④ **Dalio believability 加权**：可信度=「该领域既往战绩×能讲清因果逻辑」，经 baseball card/点收集器长期积累，属桥水专有机制，未公开打分公式。数据要求：年级别的决策记录+同侪评分。映射：单人场景退化为「用该人历史准确率给未来判断的置信度加权」，需要先有②③的累计分数。来源：https://www.principles.com/principles/633d5d13-8610-425f-ad62-cd62347d9165/

⑤ **DQ（Spetzler/SDG）**：六要素=恰当框架、可行多元备选、可靠信息、清晰价值权衡、逻辑健全、承诺行动；评「决策时点的过程质量」而非结果，无结果数据也可打分（0–10 快检）。映射：dream 复盘时的过程侧量表，与结果回收互补，可防「好过程坏结果」被误罚。来源：https://sdg.com/decision-quality/ ；书 https://www.wiley.com/en-us/Decision+Quality%3A+Value+Creation+from+Better+Business+Decisions-p-9781119144670

⑥ **AAR／复盘**：美军 TC 25-20 四问=原定发生什么／实际发生什么／为何有差异／下次怎么改；联想四步=回顾目标→评估结果→分析原因→总结经验。单事件即可执行，零样本量要求。映射：与 rick dream 复盘结构一一对应，是「判断→结果回收」闭环最直接的方法论载体。来源：https://nick.groenen.me/attachments/public/gitignored/TC%2025-20%20A%20Leader%27s%20Guide%20to%20After-Action%20Reviews.pdf ；联想 https://www.chrm.cn/article/show.php?itemid=24557

⑦ **VoI／决策价值**：EVPI/EVSI=「有 vs 无该信息时最优期望值之差」，需显式决策树（备选方案/概率/效用）。映射：「影响力=历史价值+未来期望值」对应决策的 EV 加权，但必须建模，文本拍板无法直接算。事实提醒：EV 是加性结构，proper 分数不乘权重——三项连乘既非②③也非⑦的标准形式，若保留建议改为「分数（proper）+ 影响力（EV）分列或分层」而非乘积。来源：https://en.wikipedia.org/wiki/Value_of_information ；ISPOR https://pmc.ncbi.nlm.nih.gov/articles/PMC7373630/

**操作性小结**：低样本（早期）→ 用⑥AAR 结构+⑤DQ 量表；积累后 → 拍板格式化为「概率+截止日+可验证+影响力预估」，用②③计分，用④做自加权，用⑦评估是否值得追问。
