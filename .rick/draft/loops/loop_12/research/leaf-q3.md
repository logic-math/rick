# 叶子调研 Q3：AI 辅导业界现状与有效性实证

> 勘误：任务所给 arXiv:2407.10087 实为量子物理综述（weak-value amplification），非 LearnLM 报告；正确编号为 arXiv:2407.12687（姊妹篇 arXiv:2412.16429）。仅记已核验事实；信源等级 A/B/C=同行评审一手/官方文档/媒体线索。

## ① Google LearnLM（技术报告 arXiv:2407.12687，2024）
- 解决什么问题：通用 LLM 按「直接给答案」的 helpful 优化，与教学法冲突（可绕过练习、生掌握错觉）；教学法直觉难写成 prompt；教育评测缺失。
- 机制：①「教学法指令遵循」(pedagogical instruction following)：训练/评测样本附系统级指令，描述期望的教学法属性（不绑定特定教学法）；②SFT 微调 Gemini（M0→M4，M4=LearnLM-Tutor，基于 Gemini 1.0），数据含真人辅导、AI 角色扮演、GSM8k 苏格拉底对话、教师精写「黄金对话」；③对齐五原则：鼓励主动学习、管理认知负荷、深化元认知、激发好奇与动机、适应学习者目标与需求；④配套 7 个教育评测基准。
- 效果证据：专家/学习者多数维度偏好 LearnLM-Tutor 胜 prompt 版 Gemini 1.0（并排盲评 5 准则 4 项显著，n=189，p≤0.05；准确性无差）；回复刻意更短（均长 297.6 vs 423.0 tokens）以降认知负荷；姊妹篇（Gemini 1.5 Pro 版）：专家平均偏好强度 +31% vs GPT-4o、+11% vs Claude 3.5 Sonnet、+13% vs Gemini 1.5 Pro；英国 Eedi 探索性 RCT（N=165，导师在环审每条草稿）：76.4% 草稿零/微改通过，新题解决率 66.2% vs 纯人类导师 60.7%。
- 一手信源：arXiv:2407.12687 [B] https://arxiv.org/abs/2407.12687 ；arXiv:2412.16429 [B] https://arxiv.org/abs/2412.16429 ；arXiv:2512.23633 [B] https://arxiv.org/abs/2512.23633

## ② Khanmigo（可汗学院）
- 解决什么问题：通用聊天 AI 直接给答案、有作弊与内容风险；Khanmigo 定位教育专用 AI 导师+教师助手。
- 机制（官方描述）：不直接给答案——"never giving you the answer…prompted to think deeply and critically"，以引导性提问+提示约束输出；与可汗学院内容库绑定（grounding）；媒体常称苏格拉底式（C）。
- 效果证据：Common Sense Media 2023 年底首届 AI 评级 4 星（教育 AI 工具最高档之一，高于 ChatGPT/Bard；针对透明/安全/隐私/学习，非效果 RCT）；官方博客引试点教师定性反馈「学生更独立答题」；本次检索范围内未见针对 Khanmigo 的同行评审 RCT（见 Gaps）。
- 一手信源：khanmigo.ai 官方 FAQ/learners [B] https://www.khanmigo.ai/ ；Khan Academy 官方博客 [B] https://blog.khanacademy.org/four-stars-for-khanmigo-common-sense-media-rates-ai-tools-for-learning/

## ③ 哈佛物理课 AI 导师 RCT（Kestin et al. 2025，Scientific Reports）
- 设计：哈佛本科物理大课 N=194，交叉设计：两主题连续两周，组1某周在家用 AI 导师、组2同周课堂主动学习，次周对调；前后测+体验量表。AI 导师=GPT-4+教学法系统提示（主动学习/认知负荷/成长型思维）；平台按题分步 scaffold；prompt 内置教师分步标准答案保准确。
- 数值结论：后测中位分 AI 组 4.5（N=142）vs 课堂组 3.5（N=174），前测基线 2.75（N=316）；AI 组中位学习增益>课堂组 2 倍；Mann-Whitney z=-5.6，p<10⁻⁸；回归效应量 0.63（受天花板效应低估），分位数回归 0.73–1.3 SD；AI 组中位用时 49 min vs 课堂 60 min；投入感 4.1 vs 3.6（p<0.0001）、动机 3.4 vs 3.1（p<0.001）；83% 学生认为 AI 讲解不逊人类讲师。作者限定：仅测初学新内容（Bloom 理解/应用/分析层），不外推高阶综合。
- 一手信源：Kestin, Miller, Klales, Milbourne & Ponti (2025), Sci Rep 15 [A]，DOI:10.1038/s41598-025-97652-6 https://www.nature.com/articles/s41598-025-97652-6

## ④ Socratic 引导 vs 直接讲授：实证争议
- 争论焦点：探究/建构式教学是否因缺引导而低效。
- KSC 2006 主张 [A]：极简引导违背认知架构——工作记忆有限、专家-新手差异、认知负荷；半世纪实证一致显示不如强引导教学；仅当学习者先验知识够高（自带「内部引导」）时优势才消退。Educational Psychologist 41(2):75-86，DOI:10.1207/s15326985ep4102_1
- HSC 2007 回应 [A]：KSC 把 PBL/探究学习与「无引导发现学习」混同；实际二者高度支架化（广泛用 scaffolding、降认知负荷），证据显示有效，且承载内容知识、认识论实践与协作/自主学习等目标。EP 42(2):99-107，DOI:10.1080/00461520701263368
- 后续边界条件：worked example 效应（Sweller & Cooper 1985：新手学样例比常规解题更省时、错更少）DOI:10.1207/s1532690xci0201_3；专长逆转效应（Kalyuga et al. 2003：对新手高效的技术对有经验者可失效甚至负效）DOI:10.1207/s15326985ep3801_4；KSC 2007 回应文 EP 42:115-121，DOI:10.1080/00461520701263426。均 [A] 同行评审。

## Gaps
- Khanmigo「无同行评审 RCT」是本次检索范围内结论（搜索通道受限），非穷尽证明；媒体试点报道（Hobbs/Newark 等）未核验未写入。
- LearnLM 逐维度偏好值存于论文图表，未取到逐项精确数。

## 结论
1. LearnLM：教学法指令遵循+微调，专家偏好 +31% 胜 GPT-4o，英国 RCT 亦正向。
2. Khanmigo 官方约束「不给答案、引导自答」；效果证据仅评级与定性反馈，无同行评审 RCT。
3. 引导之争边界清晰：极简引导对新手低效，支架式探究有效，引导量随专长调整。
