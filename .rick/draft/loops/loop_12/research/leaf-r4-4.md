# R4-4 事实核查：四个可升级点（只列事实与可插入点）

## (a) 学习者模型 / ZPD 掌握度接入
- ITS 整体效应：g=0.41（107 研究 / 14,321 人，vs 非 ITS）[Ma et al. 2014, **A**]；中位 d=0.66（50 项对照评价）[Kulik & Fletcher 2016, **A**]。二者均为「系统整体」效应，非「按掌握度选任务」的独立效应。
- 掌握阈值随机实验（ALEKS，2023–2024）：阈值「3 连对 vs 5 连对」随机分配，806,279 名学生；长期保持率差 ≈0.02（基线≈0.6），高阈值需 7.4 vs 5.4 题 [JEDM, **A**]。
- 85% 规则：最优错误率 15.87%（≈85% 正确率），仅来自理论推导 + 仿真，无直接人类/动物实验 [Nat Commun 2019, **B**]。
- 「按知识追踪选任务 vs 随机」现场 RCT：4,365 人，null 结果；该预印本已被 arXiv 撤下 [arXiv:2407.03118, **C**]。
- 掌握驱动 tutor 控制实验（n=196）：改为学生自选概念后，学习量与速度无差异，已学概念成绩提升 d=0.43 [Kumar, **B**]。
- 可插入点：每轮显式写出「当前掌握度估计 / 目标难度」字段。

## (b) 去固着：并置对比 / 变式
- 案例对比元分析：d=0.50，95%CI [0.44, 0.56]，57 实验 / 336 检验 [Alfieri et al. 2013, **A**]。
- 交错 vs 阻断：Exp1b 正确率 .59 vs .36，t(70)=4.94，p<.0001；由文中 M/SD 推算 d≈1.1（原文未报 d；任务给的 0.99 未在原文核实）[Kornell & Bjork 2008, **A**/推算]。
- 交错元分析（59 研究 / 238 效应量）：效应存在，受「类间相似性」调节；总体 g 未取到 [Brunmair & Richter 2019, **B**]。
- 并置多解法 > 逐一反思（七年级 n=70，概念与程序后测更优）[Rittle-Johnson & Star 2007, **B**]。
- 对照案例后置讲授 > 前置讲授 [Schwartz & Bransford 1998；Kapur, **B**，仅方向]。
- 孵化：中位 d=0.26，均值 d≈0.32（任务给的 0.29 与原文报告不一致）[Sio & Ormerod 2009, **A/B**]。
- 可插入点：同轮并列 2 个相异解法/反例，而非串行给结论。

## (c) 图解 + 文字联动
- 多媒体原则（文字+图 > 纯文字）中位 d=1.67，但仅 5 组比较、作者自评 [Mayer 2017, **B**]。
- 概念图/知识图：总 g=0.604 [0.550, 0.658]，67 效应量 / 55 研究 / 5,818 人；学生自建 g=0.819；阅读现成 g=0.373；超链接图 g=0.017（不显著）[Nesbit & Adesope 2006, **A**]。
- 信号化（signaling）：保持 g=.52 [.44,.60]，迁移 g=.31；95 研究 / 11,499 人 [Schneider et al. 2018, **A**]。
- 分段（segmenting）：保持 d=0.32 [0.20,0.43]，迁移 d=0.36 [0.24,0.48]，认知负荷 d=0.23，学习时间多 0.92 d；56 研究 / 7,713 人 [Rey et al. 2019, **A**]。
- 可插入点：结论句后附节点图/箭头结构，并做空间邻近排版。

## (d) 篇幅与交互成本
- Lost in the Middle：关键信息位于长上下文中部时性能显著下降，首尾最好 [Liu et al. 2023 TACL, **A**]——此为模型侧事实，不等于人类阅读。
- 屏幕 vs 纸：g=-0.21（2000–2017），时间压力下差距扩大；Clinton 2019 g=-0.25 [Delgado et al. 2018, **B**]。
- 长度偏好偏差：人类与 LLM 评审均偏好更长回答（verbosity bias）[arXiv:2310.10076, **B**]；未找到「回答长度→学习结果」实验。
- 交互粒度：step-based d=0.76，answer-based d=0.31 [VanLehn 2011, **B**]。
- AI 辅导现场 RCT（约 1,000 高中生）：无护栏 GPT-4 练习分 +48%、撤除后考试 -17%；护栏式提示版考试≈对照 [PNAS 2025, **A**]。
- 可插入点：每轮固定「要点数上限 + 1 个待答问题」。

## 结论 3 条
1. 掌握度/知识追踪无独立学习增益证据，阈值实验差≈0.02。
2. 并置对比 d≈0.5 稳定；交错效应大但受相似性调节。
3. 图文联动 g≈0.4–0.6；「每轮字数」缺直接证据。

## 信源（kept）
- Ma et al. 2014 ITS meta: https://doi.org/10.1037/a0037123
- Kulik & Fletcher 2016: https://doi.org/10.3102/0034654315581420
- ALEKS 阈值 RCT (JEDM): https://jedm.educationaldatamining.org/index.php/JEDM/article/download/900/254
- 85% rule: https://www.nature.com/articles/s41467-019-12552-4
- Kumar adaptive tutor: https://par.nsf.gov/servlets/purl/10155756
- Alfieri et al. 2013: https://doi.org/10.1080/00461520.2013.775712
- Kornell & Bjork 2008: https://web.williams.edu/Psychology/Faculty/Kornell/Publications/Kornell.Bjork.2008a.pdf
- Brunmair & Richter 2019: https://doi.org/10.1037/bul0000209
- Sio & Ormerod 2009: https://doi.org/10.1037/a0014212
- Nesbit & Adesope 2006: https://www.sfu.ca/~jcnesbit/research/NesbitAdesope2006.pdf
- Schneider et al. 2018 signaling: https://doi.org/10.1016/j.edurev.2017.11.001
- Rey et al. 2019 segmenting: https://doi.org/10.1007/s10648-018-9456-4
- Mayer 2017 mm e-learning: https://doi.org/10.1111/jcal.12197
- Liu et al. 2023: https://arxiv.org/abs/2307.03172
- Delgado et al. 2018: https://doi.org/10.1016/j.edurev.2018.09.003
- VanLehn 2011: https://doi.org/10.1080/00461520.2011.611369
- Bastani et al. PNAS 2025: https://www.pnas.org/doi/10.1073/pnas.2422633122

## 信源（dropped）
- Noetel et al. 2022 元-元分析：正文数值未能抓取（JS 渲染），只保留定性。
- arXiv:2407.03118：已被撤下，仅作 C 级旁证。
- SEO/二手博客：无原始统计量，弃用。

## Gaps
- 未找到任何直接操纵「LLM 每轮回答字数」并测学习结果的 RCT（d 项核心缺口）。
- Brunmair & Richter 总体 g、Rittle-Johnson & Star 具体效应量未取到。
- 任务给的两个数值（交错 d=0.99、孵化 d=0.29）与原文报告不一致，未采信。
- 缺中文/中文母语学习者样本证据。
