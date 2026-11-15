# 调研简报:生成式推荐深度调研——方向 B(扩散 / 流式生成推荐)

调研日期:2026-10-26 | 论文数:4 | 全部成功获取原文(3 篇 arXiv HTML 全文 + 1 篇经 Semantic Scholar API 与官方 GitHub 交叉验证)

## 总体摘要

四篇论文代表了 2026 年扩散式生成推荐从"可行性验证"走向"精细化设计"的四个互补切面:**解码策略与推理效率**(DPE-Rec)、**生成动力学的公平性/马太效应**(FairDiff)、**协同过滤信号注入离散扩散**(ANR-DiffRec)、**连续时间 score/ODE 建模**(ScoreRec)。其中 DPE-Rec 与 ANR-DiffRec 属离散 Semantic-ID 掩码扩散路线,FairDiff 与 ScoreRec 属连续嵌入扩散路线;共同趋势是把扩散过程的内在结构(熵分布、先验匹配、噪声调度、score 场)变成可显式操控的设计杠杆,而非仅当作黑盒生成器。

---

## 逐篇分析

### 1) DPE-Rec: Diversity-Preserving and Efficient Diffusion Language Model Recommendation for Bundle Construction

**元信息**:arXiv 2610.34700,2026-10-26 提交(v1);Pingchuan Ma、Bohan Jiang、Chengshuai Zhao、Dawei Li、Zhen Tan(Stevens)、Huan Liu(主要来自 Arizona State University)。任务:生成式 bundle 构造(补全部分可见 bundle 的缺失物品)。

**① 思考背景**:生成式 bundle 补全的输出本质是无序集合——自回归 LLM 强加从左到右的生成顺序,归纳偏置与任务错位;扩散语言模型(DLM)双向注意力天然契合无序生成,迭代去噪还可每步揭示多个 token 以降低延迟。但作者发现"灵活性陷阱":置信度引导解码总先揭示低熵的细化 token,把决定粗语义分支的"语义分叉"(SID 首 token)推迟到上下文已被锁定之后,分叉处熵值坍缩(entropy degradation),推荐集中于少数分支、生成物品高度重复。

**② 要解决的核心问题**:在 DLM 推荐解码中,如何在保持语义分叉处探索性(多样性)的同时可靠地并行揭示多个 token(效率)?此前未解,因为并行解码须以条件独立近似替代链式分解,各自合理的 token 会拼成目录外的非法 SID,质量-延迟被迫二选一。

**③ 如何解决**:两个组件。(1) **semantic-fork-aware GRPO**:从预训练去噪器导出自回归 SID 策略,仅作后训练脚手架(不改双向架构与并行解码能力),对分叉位置用高温、细化位置用低温采样(τ=0.6 vs 0.2),配层级化集合奖励(粗 SID 层部分得分 + 精确命中,按集合计分故重复无奖励),用 GRPO 迫使模型在不确定性尚存时直面分叉;(2) **伪轨迹蒸馏**:GRPO 优化后的教师按置信度任意序逐 token 解码出高质量轨迹,仅在教师轨迹诱导的紧凑状态子空间(而非随机掩码的巨大组合空间)上、以渐进难度课程训练学生从中间状态并行恢复多个未来 token,从而解锁置信度阈值化的多 token 并行解码。本质区别:以 RL 显式管理"探索预算的分配位置",以轨迹子空间蒸馏替代全空间掩码监督。

**④ 结果亮点**:在 Spotify(30/60/90)、POG、IQON3000 五种设定下较 9 个基线(ID 序列模型、表示增强判别式、SID 生成式)平均相对提升约 63%;Fast 变体较生成式基线平均加速 25.6×(如 Spotify90:66.7ms vs DDBC 8262.9ms)且质量损失极小;生成物品成对余弦相似度在所有对比方法中最低(多样性最佳),蒸馏学生在同等去噪步数下合法 SID 比例反超教师。

**来源**:[arXiv abs](https://arxiv.org/abs/2610.34700) | [HTML 全文 v1](https://arxiv.org/html/2610.34700v1)

---

### 2) FairDiff: Mitigating the Self-Reinforcing Matthew Effect in Diffusion Recommender Models

**元信息**:arXiv 2609.36671(v1 2026-09-29,v2 2026-10-19);Song-Li Wu、Xianquan Wang、Zhaocheng Du(通讯)、Weinan Gan、Jingyi Wang 等;清华大学 + 华为诺亚方舟实验室 + 中国科学技术大学。

**① 思考背景**:马太效应与过滤气泡通常被视为"结果层"偏差;作者观察到扩散推荐模型(DRM)会通过生成动力学**自我强化**放大它——DRM 整体指标的提升主要由热门物品贡献,而在冷门物品上持续劣于其他模型,即"自强化马太效应",而非单纯继承数据不平衡。

**② 要解决的核心问题**:如何拆解 DRM 特有的自强化循环?难在两个叠加机制:(1) 优化损失被高频物品支配(各推荐范式共有);(2) DRM 独有的**结构先验失配**——长尾数据的前向终止分布显著偏离标准高斯先验,从标准高斯噪声出发的反向轨迹自第一步起就坍向高密度热门模式,冷门物品生成被结构性压制。以往纠偏工作只作用于结果层(曝光控制、正则、事后多样化),未触及生成过程本身。

**③ 如何解决**:FairDiff 是即插即用、纯推理期的公平性框架,不改骨干架构与训练目标。(1) **PCG(Popularity Condition Guidance)**:把归一化流行度作为显式条件输入去噪器,利用真实流行度条件与采样辅助流行度条件下输出之差,在 score 梯度场中数学地惩罚高流行区域、把轨迹导向冷门语义;引导强度 w 可控公平-精度权衡(w=0.5 较优,w=0 更公平但略损精度)。(2) **SC(Semantic Calibration)**:以一步静态最优传输(OT)映射把反向初始分布对齐到经验前向终止分布,修复先验失配,同时支持截断采样跳过冗余早期去噪步。本质区别:在"过程层"重塑生成轨迹,而非事后重加权。

**④ 结果亮点**:在多个顺序推荐与跨域多模态数据集、多个扩散骨干(如 CDiff4Rec)上取得一致且统计显著(p<0.05)的提升,同时曝光不均(ΔExp)下降、长尾覆盖(APLT)上升,公平与效用兼得;SC 使前向终止-反向初始分布的 KL 散度从 10.64 降至 0.25,且几乎不引入额外计算开销;消融显示 PCG+SC 组合最优。

**来源**:[arXiv abs](https://arxiv.org/abs/2609.36671) | [HTML 全文 v2](https://arxiv.org/html/2609.36671v2)

---

### 3) ANR-DiffRec: Adaptive Item-based Collaborative Structures via Noise Rescheduling in Diffusion for Generative Recommendation

**元信息**:arXiv 2608.23400,2026-08-24 提交(v1);Jiaqi Wang、Tianying Liu、Heng Chang(通讯)、Jihong Guan、Wengen Li、Shuigeng Zhou;同济大学 + 华为 + 复旦大学。代码:[github.com/CalmaQi/ANR-DiffRec](https://github.com/CalmaQi/ANR-DiffRec)。

**① 思考背景**:离散扩散模型(LLaDA-Rec、DiffGRM)把用户历史建模为迭代去噪的 token 生成,擅长用户级序列模式,但作者指出其丢掉了经典协同过滤的核心信号——item-based 协同结构,体现为两点:(1) SID 只由文本语义嵌入构成,编码不了"语义不同却高频共现"的物品关系;(2) 均匀噪声调度对所有掩码 token 一视同仁,忽视各 token 上下文可恢复性的巨大差异。

**② 要解决的核心问题**:如何把 item-based 协同结构显式注入离散扩散的物品表示与训练目标?此前未解,因为 DDM 方法在去噪时无法区分易恢复/难恢复 token,且 SID 空间本身缺乏协作先验可供利用。

**③ 如何解决**:ANR-DiffRec 双管齐下。(1) **共现引导的 SID 生成**:从历史交互构建物品-物品共现矩阵,经 SVD 分解得 64 维协作嵌入,与 Sentence-T5 语义特征融合后,用 RQ-KMeans(L=4 层、每层码本 256)离散化为 SID,使其同时携带语义相似性与隐式转移模式;(2) **item-based 自适应噪声重调度**:联合"局部可恢复性估计"(intra-item 与 inter-item 几何上下文依赖)与"行为感知依赖建模"(注意力交互)动态调整每个掩码 token 的去噪权重——上下文支撑强的 token 获更大权重(等价更低有效噪声)。与已有方法的本质区别:把 CF 信号同时写进表示(共现先验)与目标(结构感知去噪权重),复杂度仍保持 O(M²d)。

**④ 结果亮点**:5 个数据集(Amazon Scientific/Instrument/Game、Steam、MovieLens)× 16 个基线(GRU4Rec、SASRec、BERT4Rec、DreamRec、DiffuRec、PreferGrow、TIGER、LC-Rec、RPG、DiffGRM、LLaDA-Rec 等)上全指标第一(如 Scientific R@1 0.0122 vs LLaDA-Rec 0.0098,MovieLens NDCG@5 0.1468 vs 0.1359);消融显示去掉噪声重调度降幅最大(Scientific NDCG@1 跌逾 15%);训练开销仅约 +1%,tokenization 成本仅为 RQ-VAE 的 0.059×;案例显示 SID 空间共现邻居数最多增至 3 倍,中频物品收益最明显。

**来源**:[arXiv abs](https://arxiv.org/abs/2608.23400) | [HTML 全文 v1](https://arxiv.org/html/2608.23400v1)

---

### 4) ScoreRec: Reallocating Probability Mass in Sequential Recommendation via Score-based Diffusion Modeling

**元信息**:ACM TOIS Vol. 44, No. 8,DOI [10.1145/3831688](https://doi.org/10.1145/3831688);2026-05-16 录取、2026-09-30 上线;Peng He、Yao Liu、Tong Luo、Yanglei Gan、Tingting Dai、Run Lin、Qiao Liu(通讯),电子科技大学(UESTC)。代码:[github.com/AONE-NLP/ScoreRec](https://github.com/AONE-NLP/ScoreRec)。注:ACM DL 直接抓取返回 403,本文细节经 Semantic Scholar Graph API(完整摘要)与官方 GitHub README(方法、超参、主结果表)交叉验证,两者一致。

**① 思考背景**:扩散模型虽能捕获复杂偏好分布,作者发现现有 DM 推荐器存在关键缺陷——**分布失配**:mode-seeking 的去噪过程把概率质量不成比例地分配给热门物品,偏离用户真实偏好;且这些方法基于固定时间步近似的离散 DDPM,难以在嵌入流形的稀疏(长尾)区域提供有效训练信号。

**② 要解决的核心问题**:如何把概率质量重新分配到低频但个人相关的物品,同时兼顾排序准确率与推理效率?固定离散调度在稀疏区缺乏有效监督信号,此前方法未从生成过程层面解决。

**③ 如何解决**:ScoreRec 把序列推荐重构为**连续时间 score 生成过程**。(1) 以随机微分方程(SDE)与连续噪声调度(噪声强度随局部嵌入密度自适应)训练分数网络,直接估计条件对数密度梯度(去噪分数匹配),获得"无限时间分辨率";(2) **双流历史编码器**:Transformer 上下文流捕获长程依赖 + DFT 频谱流把偏好分解为低频(长期)/高频(短期)信号,拼接为生成条件;(3) 推理时用**自适应步长概率流 ODE**(SciPy RK45)确定性积分出目标物品嵌入,再与候选嵌入做内积排序取 top-K。与 DDPM 式推荐器(DreamRec/DiffuRec/PreferDiff)的本质区别:从固定离散转移改为连续 score 场 + 确定性 ODE 采样,从机制上缓解 mode-seeking。

**④ 结果亮点**:Zhihu、YooChoose、Sports & Outdoors 三大基准 × 11 个 SOTA 基线上准确率与多样性双赢:YooChoose 上 HR@5 相对提升 27.9%、NDCG@5 提升 32.7%;Gini@5 在三个数据集全部最低(曝光分配最均衡),ILD-E@5 在两数据集最高;推理效率上 Zhihu 为 30.35 s/epoch,约为 DiffuRec(281.2 s/epoch)的 1/9;另含 5 个 Amazon 2018 域的跨域验证与长尾、敏感性、FLOPs 等分析,代码开源。

**来源**:[ACM DL(403,经引用解析)](https://dl.acm.org/doi/10.1145/3831688) | [Semantic Scholar 记录](https://api.semanticscholar.org/graph/v1/paper/DOI:10.1145/3831688) | [官方 GitHub README](https://github.com/AONE-NLP/ScoreRec)

---

## 来源清单

**保留(一手/官方)**:
- DPE-Rec arXiv abs + HTML v1(2610.34700)— 方法、实验全细节的直接来源
- FairDiff arXiv abs + HTML v2(2609.36671)— 含 v2 修订,动机/方法/消融/KL 定量结果
- ANR-DiffRec arXiv abs + HTML v1(2608.23400)— 含 16 基线主表、消融、效率分析
- ScoreRec Semantic Scholar Graph API(DOI:10.1145/3831688)— ACM 403 时的完整摘要与元数据替代
- ScoreRec 官方 GitHub(AONE-NLP/ScoreRec)raw README — 官方方法说明、最优超参与主结果表,与摘要交叉一致

**排除**:
- dl.acm.org 直接抓取 — HTTP 403 Forbidden,改走 API/镜像
- paddadata.com(DPE-Rec 解读博客)— 二手 SEO 内容,被一手来源完全覆盖
- arxivdaily.com、descubridor.ipchile.cl、arxivtldr.org、doi.org/10.48550 镜像页 — 聚合/镜像站点,信息冗余
- github.com/AONE-NLP/ScoreRec 网页版抓取失败 — 已改用 raw.githubusercontent.com 的 README 成功获取

## 未能确认的信息与缺口(Gaps)

1. **ScoreRec 正文数字未逐条核验**:ACM DL 403,消融、长尾、跨域、显著性检验的逐格数值来自官方 README 转述(与摘要一致),如需精确引用建议下载 ACM PDF(开放获取,CC-BY)。
2. **DPE-Rec 主表(Table 1)逐格分值未完全展开**:HTML 中表格数值渲染不全,已核对"约 63% 平均提升"与延迟表(Table 2)的原文表述;63%/25.6× 为论文自报口径。
3. **FairDiff 主表(Table 1/2)逐格数值未在 HTML 文本中展开**:已核对的结论为正文文字描述(一致显著提升、ΔExp↓、APLT↑、KL 10.64→0.25);所用顺序推荐数据集为 Amazon Toy/Game/Book 等(附录表 5/6 为图形式),跨域多模态数据集清单未逐项提取。
4. **四篇均为 2026 年 8–10 月新作,引用数为 0**,尚无第三方复现或独立验证;DPE-Rec 与 FairDiff 未在文中提供公开代码链接(ANR-DiffRec 与 ScoreRec 有 GitHub)。
5. 后续建议:方向 B 可补充对比 DDBC(DPE-Rec 的直接前作)、LLaDA-Rec/DiffGRM(离散扩散推荐主干)、DreamRec/PreferDiff(连续扩散推荐主干)以建立完整谱系;如需线上收益证据,四篇均未报告 A/B 实验结果。
