# 调研：AI→human 个性化体验指标度量方法学（尽调事实，仅事实+来源）

## ① Per-user 偏好学习 / 个性化评估（LLM/agent 文献）
- **P-RLHF**：vanilla RLHF 假设偏好同分布；P-RLHF 为每个用户学独立 user model，兼容任意偏好优化算法。[arXiv:2402.05133](https://arxiv.org/abs/2402.05133)
- **变分偏好学习**：将个体偏好差异建模为隐变量分布，避免"平均化"导致子群体奖励失准。[arXiv:2408.10075](https://arxiv.org/abs/2408.10075)（NeurIPS 2024）
- **Reward 特征分解（DeepMind）**：个体偏好 = 通用 reward 特征的线性组合，可特化到个人/群体。[arXiv:2503.17338](https://arxiv.org/abs/2503.17338)（NeurIPS 2025）
- **SynthesizeMe**：从用户交互历史归纳合成 persona，用于个性化 reward model，不依赖人口学标签。[arXiv:2506.05598](https://arxiv.org/abs/2506.05598)
- **个性化评测可靠性**：标准 LLM-as-a-Judge 对个性化任务仅 ~70% 与人类一致（二元选择），persona 稀疏是主要失败因素。[EMNLP 2024 Findings](https://aclanthology.org/2024.findings-emnlp.592.pdf)

## ② NPS/CSAT 方法论与局限
- 学术界长期批评 NPS：任意截断点（6/9）、丢弃部分样本信息、11 点量表坍缩为三分类，学界呼吁弃用。[JBR 2022](https://ideas.repec.org/a/eee/jbrese/v149y2022icp353-362.html)
- NPS 统计性质差（差值统计量方差大、噪声放大），统计学界明确反对单独使用。[arXiv:1806.10452](https://arxiv.org/abs/1806.10452)
- CSAT 回收率仅 10–25%，自选择偏差使极端体验者过采样；响应延迟降低数据质量（漏答、定势偏差上升）。[Response bias](https://pmc.ncbi.nlm.nih.gov/articles/PMC1464019/)
- 主观分数与实际行为/绩效弱关联，须辅以行为指标（任务成功率、任务时长）。[NN/g](https://www.nngroup.com/articles/nps-ux/)
- 结论性事实：NPS/CSAT 为**群体横断面**设计；单用户单次打分无统计意义，横向人群推断方法不适用纵向 N=1。

## ③ 小样本主观评价的统计方法
- **Beta-Binomial 共轭**：单用户"好/差"比率追踪的标准模型，先验+逐次后验更新；贝叶斯版本见 JBES 1987。[tandfonline](https://www.tandfonline.com/doi/abs/10.1080/07350015.1987.10509600)
- **贝叶斯收缩排序**：小样本评分向先验均值收缩，避免"单次满分置顶"。[Evan Miller](https://www.evanmiller.org/ranking-items-with-star-ratings.html)
- **SPRT 序贯检验**：Wald 序贯概率比允许 peeking/提前停止，最小化样本需求，A/B 实验实践已产品化。[arXiv:2606.24871](https://arxiv.org/html/2606.24871)、[Statsig](https://docs.statsig.com/en/experiments/advanced-setup/sprt)
- **贝叶斯 N-of-1**：个体内重复交叉测量+分层先验做个体疗效推断。[PMC10817775](https://pmc.ncbi.nlm.nih.gov/articles/PMC10817775/)

## ④ 单用户纵向指标的成熟实践
- **Single-case design（SCD）**：A=基线、B=干预；WWC 证据标准要求**至少 4 个 A/B 相位**（如 ABAB）方可因果推断。[ies.ed.gov](https://ies.ed.gov/ncee/wwc/Docs/ReferenceResources/wwc_scd.pdf)
- **Alternating Treatments Design**：单被试内快速交替两种处理直接比较。[Barlow & Hayes 1979, PMC1311363](https://pmc.ncbi.nlm.nih.gov/articles/PMC1311363/)
- **视觉分析规范**：level/trend/variability 三要素跨相位比较判定效果。[PMC6745757](https://pmc.ncbi.nlm.nih.gov/articles/PMC6745757/)
- **量化自我**：QuantifyMe 自动化单人实验平台；新手自我实验的结构化方法研究。[MDPI Sensors 2018](https://www.mdpi.com/1424-8220/18/4/1097)、[IMWUT 2017](https://dl.acm.org/doi/10.1145/3130911)

## ⑤ 任务无关 agent 质量指标与难度归一化
- 聚合 pass rate 掩盖任务多样性；**任务级 psychometrics** 预测单任务成败替代单一分数。[arXiv:2604.00594](https://arxiv.org/abs/2604.00594)
- **IRT** 用于估计模型能力+任务难度归一化，但 AI 数据形态（模型少、条目多、能力分布偏态）偏离人类测验假设，需校正。[arXiv:2607.15190](https://arxiv.org/html/2607.15190v2)
- **TASTE**：程序化生成任务并显式控制覆盖度与难度分布。[arXiv:2605.28556](https://arxiv.org/html/2605.28556v1)
- 任务有效性硬伤实例：τ-bench 平凡 agent 可过 38% 任务而无需领域知识。[NeurIPS 2025 D&B](https://proceedings.neurips.cc/paper_files/paper/2025/file/f316275b44ee2de533102913828a8107-Paper-Datasets_and_Benchmarks_Track.pdf)
- 任务数充分性：预算化任务抽样 + 决策误差/覆盖度目标的回放分析。[KDD 2026 workshop](https://kdd-eval-workshop.github.io/agenticai-evaluation-kdd2026/assets/papers/92_How_Many_Tasks_Are_Enough_f.pdf)
