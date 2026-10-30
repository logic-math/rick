# Research: 如何度量一个启发式 skill 是否有效

等级说明：A=同行评审期刊/官方标准文档；B=同行评审会议论文/机构技术文档；C=预印本或未评审。

## Summary
发散思维任务可给出可复现的原始分数（fluency/flexibility/originality/elaboration），自动化语义距离与 LLM 打分已被多次与人类评分对照，相关值多为小到中等（r≈.35）或高（r≈.81，监督微调模型）；LLM-as-judge 存在可量化的位置、冗长、自偏好偏差；最小有效测试可用 within-subject 交叉或单被试（SCED）设计，后者有明确的每阶段数据点下限。

## Findings
1. **四指标定义与计分**：fluency=想法总数；originality=想法的统计罕用性（低频/独特计分）；flexibility=想法所属概念类别数；elaboration=细节丰富度。[A] [NeuroImage 2021](https://www.sciencedirect.com/science/article/pii/S1053811920308119)
2. **主观评分信度**：CAT 式多评分者平均 ICC 在每次测评中均 >.90；但文献同时记录评分者不一致与疲劳会压低信度。[A] [JOCB 2020](https://doi.org/10.1002/jocb.462) / [A] [BRM 2021](https://pmc.ncbi.nlm.nih.gov/articles/PMC8062332/)
3. **罕用性计分混淆**：uniqueness 计分与 fluency 关系极强，故须用 max/top-2/average 计分或固定想法数来控制数量-质量混淆。[A] [Beaty et al. 2022, CRJ](https://par.nsf.gov/servlets/purl/10350927)
4. **语义距离与人类评分（AUT）**：SemDis 潜变量与人类创意评分 r=.49（词联想任务 novelty r=.73，appropriateness r=−.93；另一研究 originality r=.85）。[A] [BRM 2021](https://doi.org/10.3758/s13428-020-01453-w)
5. **AUT 条目与信度**：13 条目短表（belt…sock），Cronbach α=.638–.785（ω 同量级），与人类评分均值 r=.351，95%CI [.125,.543]；建议条目尽量多、指令用"be creative"。[A] [CRJ 2022](https://doi.org/10.1080/10400419.2022.2025720)
6. **监督式 LLM 打分**：Ocsai（微调 LLM，1–5 分）与人类评分相关最高 r=.81，优于纯语义距离。[B] [Open Creativity Scoring](https://openscoring.du.edu/ocsai) / [B] [ERIC ED629879](https://files.eric.ed.gov/fulltext/ED629879.pdf)
7. **零样本 LLM 评 AUT**：模型间 Spearman 平均 >0.7，与"oracle"分级 >0.77；未见自我偏好。[C] [arXiv 2411.15560](https://arxiv.org/html/2411.15560v2)
8. **位置偏差**：成对评判交换顺序后，仅 GPT-4 一致率 >60%（65.0%，few-shot 提至 77.5%）；GPT-4 与人类一致率 85%，人类间 81%。[B] [NeurIPS 2023 / arXiv 2306.05685](https://arxiv.org/html/2306.05685v4)
9. **冗长偏差**：重复列表攻击下 Claude-v1、GPT-3.5 失守率均 91.3%，GPT-4 为 8.7%；自我偏好：GPT-4 自身胜率高 10%，Claude-v1 高 25%。[B] 同上
10. **风格而非概念**：LLM 输出常被评为高 originality 但多样性低（"LLM creativity paradox"），自动评分可能追踪风格特征。[C] [PsyArXiv](https://doi.org/10.31234/osf.io/p6dyu_v1)；同向证据见 elaboration 偏差 [C] [arXiv 2604.20569](https://arxiv.org/html/2604.20569v1)
11. **创意基准**：Creativity Benchmark（营销创意）用 678 名从业者、11,012 次匿名成对比较 + Bradley-Terry，模型表现紧密聚集、无模型通吃。[C] [arXiv 2509.09702](https://arxiv.org/pdf/2509.09702)；CreativeBench（代码创意，Boden 框架，Combo/Explore 两子集）以可执行正确性+新颖性为 ground truth。[B] [ACL 2026 Findings](https://aclanthology.org/2026.findings-acl.1546/)
12. **单被试设计下限**：WWC、APA DIV12、DIV16 一致要求基线≥3 个数据点，WWC 最高评级需每阶段≥5 个点，且基线须稳定、无趋势、与干预期不重叠。[A] [Psychol Methods 2012 综述](https://pmc.ncbi.nlm.nih.gov/articles/PMC3652808/) / [B] [WWC SCD 手册](https://ies.ed.gov/ncee/wwc/Docs/ReferenceResources/wwc_scd.pdf)
13. **SCED 在教育技术中的实际使用**：系统综述纳入 136 项 TEL 研究，多集中于特殊教育/计算机辅助教学；另有 CHI 论文提出 N-of-1 评估框架，并指出 RCT 在移动应用场景的局限。[A] [BJET 2024](https://doi.org/10.1111/bjet.13468) / [B] [CHI EA 2019](https://doi.org/10.1145/3290607.3312923)
14. **within-subject 交叉的已知风险**：顺序/携带效应须用拉丁方平衡，且平衡不能消除全部携带效应，需更大样本与严格流程。[B] [IEEE VR 2024](https://doi.org/10.1109/vrw62533.2024.00007) / [A] [ACM 2025 within-subject 陷阱](https://doi.org/10.1145/3772363.3799010)
15. **会话内想法差异性的量化**：(i) distinct idea count（去重后计数，即 fluency 的清洗版）；(ii) 语义嵌入两两距离/方差（idea diversity，与 SemDis/forward flow 同族）；(iii) 词汇多样性 MTLD、MATTR 在心理测量上优于原始 TTR；(iv) 设计学界的 novelty/variety/quality/quantity 四指标。[A] [BRM 2021](https://doi.org/10.3758/s13428-020-01453-w) / [A] [JSLHR 2015](https://pmc.ncbi.nlm.nih.gov/articles/PMC4490052/) / [A] [Design Studies 2003](https://doi.org/10.1016/s0142-694x(02)00034-0)

## Sources
- Kept: Beaty et al. 2022 CRJ（SemDis 条目/信度/效度数值）；Beaty & Johnson 2021 BRM（潜变量相关值）；MT-Bench 论文（三类偏差的具体百分比）；WWC/DIV16 与 Psychol Methods 综述（SCED 数据点下限）；BJET 2024 TEL-SCED 综述；Organisciak Ocsai 文档（r=.81）；Shah 2003 Design Studies（四指标）。
- Dropped: 各类 SEO/工具站与博客转述；ResearchGate 镜像（与原件重复）；无可核数值的二手摘要。

## Gaps
- 未取到 PUT（Product Improvement Test）单独的 fluency/flexibility 信度数值；本报告以 AUT 为主，PUT 结论为类推。
- 未获得 LLM judge 在"创意"任务上位置偏差的专用数值（现有百分比多来自通用聊天评判）。
- HCI/教育技术中"创意结果"研究的样本量经验值缺乏统一定量标准；可操作替代是按效应量做先验功效分析，或改用 SCED 的 3–5 点/阶段规则。
- 两处时间戳异常条目（arXiv 2604.20569、ACL 2026 CreativeBench）仅作 C/B 级参考，需人工核对。

## 结论3条
1. 主指标用语义距离（人类评分 r≈.35）或微调 LLM 评分（r≈.81），人工评分只做抽样校准。
2. 会话内差异性报「去重想法数 + 嵌入两两距离」，并同时报 fluency 防数量-质量混淆。
3. 最小设计取 within-subject 交叉或 SCED，每阶段≥3–5 点并交换呈现位置。
