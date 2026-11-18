# 方向 B 第二轮深挖简报：扩散 / 流式生成推荐（r2-B-deepen）

> 调研日期：2026-10-26。承接首轮《B-diffusion-flow.md》。本轮三件事：**A** 逐条补齐首轮 Gaps；**B** 对本方向最重要的 3 篇（DPE-Rec、FairDiff、ANR-DiffRec）做近可复现级深挖（架构 / 公式 / 算法伪代码 / 训练配方 / 部署细节）；**C** 补扫 2026-07 下旬 ~ 2026-10-26 首轮与顶会篇均未覆盖的新论文。所有事实来自本轮一手抓取（arXiv HTML 全文逐字符读取、OpenAlex API、GitHub、ACM 元数据页），抓不到的均明确标注原因，无编造。

## Summary

首轮 5 条 Gaps 中 4 条已实质补齐（ScoreRec 出版元数据与引用数核验、DPE-Rec 延迟表与消融数字全量提取、FairDiff 数据集/基线/KL 数值/理论全量补齐、引用数与代码可用性核验），1 条（ScoreRec 正文逐格数值）因 ACM 全端点 403 确认无法补齐。三篇深挖论文呈现出清晰的方法谱系：**DPE-Rec** 用「RL 管理探索预算分配 + 轨迹子空间蒸馏」攻克离散扩散解码的多样性-效率两难；**FairDiff** 用「推理期 score 引导 + 一步 OT 校准」从生成动力学层面拆解扩散推荐的自强化马太效应（并给出两偏差正交乘性复合的理论）；**ANR-DiffRec** 把 item-based CF 信号同时写进 SID 表示与去噪权重（结构几何衰减 + 行为注意力，复杂度不变）。遗漏扫描确认 5 篇窗口内未覆盖新论文（EPIC、Eval4DiRec、OT-FlowRec、MIRAGE、UHIFlow），其中 Eval4DiRec（TKDD）首次为扩散推荐建立 14 模型 × 5 场景的统一评测基准，是本方向的重要基础设施。

---

## A. 首轮 Gaps 逐条补齐

### Gap 1：ScoreRec 正文数字未逐条核验 → **部分补齐，正文数值确认不可达（付费墙）**

- **本轮再试三种通道均 403**：`dl.acm.org/doi/10.1145/3831688`、`dl.acm.org/doi/epdf/...`、`dlnext.acm.org/doi/epdf/...` 全部 HTTP 403 Forbidden（本轮直抓两次 + 首轮一次）。检索确认 **ScoreRec 无 arXiv 预印本版本**，无法走 arXiv 通道。
- **已核验的元数据**（来源：ACM DL 页面元数据经搜索索引/摘要页核验，[dl.acm.org/doi/10.1145/3831688](https://dl.acm.org/doi/10.1145/3831688)）：TOIS **Volume 44, Issue 8**；**Accepted 16 May 2026**；**Online 2026-09-30**；**期刊卷期正式出版 2026-10-21**。即首轮"2026-09-30 上线 / 2026-10-21"两个日期均正确，分别对应 online AM 与卷期出版。
- **引用数**：OpenAlex API 查询（2026-10-26）`cited_by_count = 0`（[api.openalex.org/works/doi:10.1145/3831688](https://api.openalex.org/works/doi:10.1145/3831688)）；Semantic Scholar 经搜索索引亦显示 "0 citations"。
- **残余**：消融、长尾、跨域、显著性检验的逐格数值仍不可核验——**ACM 付费墙/反爬（403），无 arXiv 版本**。如需精确引用，须机构订阅下载 PDF 或向作者（UESTC Qiao Liu 组）索取；官方 GitHub README（[AONE-NLP/ScoreRec](https://github.com/AONE-NLP/ScoreRec)，首轮已与摘要交叉验证）仍是方法级最优替代来源。

### Gap 2：DPE-Rec 主表逐格分值未完全展开 → **大部分补齐（Table 2 全量 + 消融数字；Table 1/3/4/5 仍图片化）**

本轮完整精读 arXiv HTML v1 全文（73,227 字符全部读取，[arxiv.org/html/2610.34700v1](https://arxiv.org/html/2610.34700v1)）：

- **Table 2（延迟表）逐格全量提取**（ms/bundle，Spotify K=30/60/90、POG K=4、IQON3000 K=8）：

| 模型 | S30 | S60 | S90 | POG | IQON |
|---|---|---|---|---|---|
| BERT4Rec | 1.66 | 1.93 | 2.27 | 1.39 | 1.51 |
| GRU4Rec | 13.18 | 26.17 | 39.10 | 1.53 | 5.92 |
| HSTU | 18.54 | 36.16 | 54.91 | 2.72 | 5.26 |
| RecJPQ | 22.15 | 44.03 | 66.01 | 3.16 | 6.01 |
| SASRec | 26.82 | 55.84 | 82.95 | 3.90 | 7.71 |
| FDSA | 37.79 | 75.53 | 113.41 | 5.24 | 9.77 |
| DiffGRM | 367.89 | 736.45 | 1267.27 | 68.47 | 91.40 |
| UniGRec | 593.76 | 1168.84 | 1792.86 | 80.35 | 154.68 |
| DDBC | 2930.87 | 5822.53 | 8262.87 | 549.53 | 872.57 |
| **DPE-Rec** | 443.59 | 903.41 | 1340.05 | 59.38 | 112.51 |
| **DPE-Rec-Fast** | **62.34** | **65.79** | **66.72** | **46.49** | **58.32** |

  DPE-Rec 相对 9 基线加速 1.01/0.98/0.97/1.34/1.14×；Fast 变体加速 7.15/13.46/19.45/1.71/2.20×；**Fast 的延迟随 bundle 从 30→90 仅从 62.34 涨到 66.72 ms**。
- **消融数字补齐**（正文文字，Table 3 数值仍图片化）：fork-aware GRPO 使 F1 / Jaccard 分别 **+28.86% / +33.29%**（不增推理延迟）；伪轨迹蒸馏（PTD）再使延迟 **−89.52%**、三项质量指标仅降约 **2.5%**。
- **63% / 25.6× 口径澄清**：63% = 相对全部 9 个基线、5 个评测设置、3 个指标的平均相对提升；25.6× = Fast 变体相对 3 个生成式基线（UniGRec/DiffGRM/DDBC）的平均加速。**稳健性复核**：剔除 4 个弱基线后（仅对 FDSA/HSTU/UniGRec/DiffGRM/DDBC 5 个强基线），质量平均提升仍有 **26.9%**、加速 **15.6×**——首轮"论文自报口径"疑虑可部分解除。
- **附录新增关键数字**：训练成本摊销——GRPO+PTD 额外训练开销在生成 **1.3K–5.0K** 个 bundle 后即被推理节省抵消；多样性——目录覆盖为 DiffGRM/UniGRec 的 **2.2–3.4×**、cluster entropy 全场景最高；SID 碰撞消解——第 4 token 按目录顺序消歧，最大碰撞数 Spotify 74 / POG 53 / IQON3000 171。
- **残余**：Table 1（质量主表）、Table 3（消融逐格）、Table 4（超参表）、Table 5（GPU-hours 成本表）在 arXiv HTML 中**渲染为图片、不进文本层**，逐格数值需下载 PDF（CC-BY 4.0）。

### Gap 3：FairDiff 主表数值与数据集清单 → **数据集/基线/KL/理论全量补齐（主表逐格仍图片化）**

本轮完整精读 arXiv HTML v2 全文（143,591 字符全部读取，[arxiv.org/html/2609.36671v2](https://arxiv.org/html/2609.36671v2)）：

- **数据集清单补齐**（7 个 + 扩展场景）：
  - 顺序推荐主表（Table 1）：**Amazon Beauty、Amazon Toys、Steam**；
  - 顺序推荐另含 **MovieLens-1M**（设置节列出）；时序建模实验：**Toy / Game / Book**；
  - 多模态：**TikTok、Baby、Sports**（V/A/T 三模态统计见附录 Table 6）；
  - 跨域冷启：**Movie→Music、Book→Movie、Book→Music**（20%/50%/80% 稀疏度，3 场景 × 9 设置 × MAE/RMSE 共 27 项指标全胜最强基线）；CDR 附录场景 **Toy→Game / Game→Toy**。
- **基线清单补齐**（附录"Introduction and Implement of Baselines"逐个列出）：顺序推荐 = 判别式（SASRec、BERT4Rec、CSRec、TiMiRec、BASRec、HSTU）+ DRM（**DiffuRec、DreamRec、CDiff4Rec**）+ 生成式（ACVAE、TIGER）；多模态 = BPR、LightGCN、MMGCN、DualGNN、MGCN、Freedom、VBPR、SLMRec、BM3、**DiffMM、DiffCL**；跨域 = PDRec(M)、T-DiffRec(M)、TI-DiffRec(M)、SASRec(M)。消融骨干为 **CDiff4Rec**。
- **KL 数值精确化（4 位小数）**：前向终止分布 vs 反向初始分布 KL **10.6447 → 0.2475**；原始分布 vs 重建分布 KL **8.2825 → 0.2383**（RQ5 可视化研究节，经经验采样估计）。首轮"10.64→0.25"与此一致。
- **显著性更正**：v2 Table 1 注明 t 检验 **p<10⁻²（⋆）/ p<10⁻⁴（⋆⋆）**——首轮写的"p<0.05"应更正为 v2 口径（部分对比达 p<10⁻⁴）。
- **消融补齐**（CDiff4Rec 骨干）：仅注入流行度条件（PC）可提效用但纠偏有限；**w_PCG=0.5** 中等引导强度下效用与公平双赢（推荐值）；**w=0.0**（完全流行度无关）更公平但略损效用；SC 单独使用一致提升且支持截断采样跳过冗余早期步（T→T/2 段可跳过）；**PCG+SC 组合全面最优**。
- **理论部分补齐**（附录 Theory）：命题 0.2 梯度支配（E‖∇θL_i‖ ∝ f(i)）；定理 0.5 反向初始化偏差（从各向同性先验出发，P(generate i) ≤ C·p(i)，尾物品吸引盆测度趋 0）；定理 0.8 两偏差**正交且乘性复合**；定理 0.16 **PCG ≡ 从重加权分布 p̃_t(x) ∝ p_t(x)·p_t(c|x)^λ 采样**（λ<0 时显式惩罚高流行区域）；结构差距用 W₂² 度量并给出下界（渐近不消失）。
- **工程信息补齐**：全部实验在 **A100 GPU** 上运行；每实验 **5 个随机种子**取平均。
- **残余**：Table 1/2/3/4 逐格数值图片化不可提取；FairDiff 自身超参（lr、扩散步数 T、w 网格扫描细节）未在文本层展开（附录列的是各基线超参）。需 PDF（CC-BY 4.0）。

### Gap 4：引用数为 0 / 无第三方复现；DPE-Rec 与 FairDiff 无代码 → **已核验**

- **引用数（OpenAlex API，2026-10-26 查询）**：DPE-Rec（[arXiv:2610.34700](https://api.openalex.org/works/doi:10.48550/arXiv.2610.34700)）**0**；FairDiff（[arXiv:2609.36671](https://api.openalex.org/works/doi:10.48550/arXiv.2609.36671)）**0**；ANR-DiffRec（[arXiv:2608.23400](https://api.openalex.org/works/doi:10.48550/arXiv.2608.23400)）**0**；ScoreRec（[DOI:10.1145/3831688](https://api.openalex.org/works/doi:10.1145/3831688)）**0**。结论维持：**四篇均暂无第三方复现/独立验证**（注：Semantic Scholar Graph API 本轮持续 429 限流，改用 OpenAlex；ScoreRec 另有搜索索引中 S2 "0 citations" 佐证）。
- **代码可用性核验**：ANR-DiffRec 官方代码在论文摘要中直接给出（[github.com/CalmaQi/ANR-DiffRec](https://github.com/CalmaQi/ANR-DiffRec)）；ScoreRec 官方 PyTorch 实现（[github.com/AONE-NLP/ScoreRec](https://github.com/AONE-NLP/ScoreRec)）；**DPE-Rec 论文内无代码链接，GitHub 检索未发现官方实现**（检出的 [241416/DDBC](https://github.com/241416/DDBC) 是其前作 DDBC 的仓库，非 DPE-Rec）；**FairDiff 论文内无代码链接，检索未发现官方实现**（检出的 [danielemalitesta/FairDiffRec](https://github.com/danielemalitesta/FairDiffRec) 是 RecSys 2025 LBR 另一篇"How Fair is Your Diffusion Recommender Model?"，[zihanlizi/A2G-DiffRec](https://github.com/zihanlizi/A2G-DiffRec) 是 SIGIR 2026 另一篇，均非本文，注意区分）。

### Gap 5：建议补充谱系对比 → **已补（本轮 Part B/C 覆盖）**

- DPE-Rec 直接前作 **DDBC**（ICLR 2026 poster，[OpenReview dKyhgfe50H](https://openreview.net/forum?id=dKyhgfe50H)，2026-01-26 发布；DPE-Rec 沿用其预训练配方与 SID→物品合法化策略）——见 B1。
- DDM 骨干谱系确认：**LLaDA-Rec**（EMNLP 2026，[aclanthology 2026.emnlp-main.616](https://aclanthology.org/2026.emnlp-main.616.pdf)，arXiv 2511.06254）、**DiffGRM**（WWW 2026，DOI 10.1145/3774904.3792156）、**MaskGR**（Snap，[arXiv 2511.23021](https://arxiv.org/html/2511.23021)）、**MDGR**（RecSys 2026，见 C 节边界补充）——ANR-DiffRec/EPIC 均以 LLaDA-Rec 为最强基线/骨干。
- 连续谱系（DreamRec/DiffuRec/CDiff4Rec/DiffRec/CF-Diff/DDRM/DiffMM/CCDRec/Diff-POI/DiffCDR 等 14 模型 × 5 场景）已由 **Eval4DiRec**（C2）系统化给出；流式侧代表为 FAVE（SIGIR 2026）、FlowRec（2025-08）、本轮新发现的 MIRAGE/OT-FlowRec/UHIFlow。
- 线上收益证据：四篇首轮论文仍均无 A/B；窗口内最接近的是 **MDGR**（RecSys 2026，在线广告平台部署营收 +1.20%，见 C 节）。

---

## B. 方法内部深挖（3 篇，近可复现级）

### B1. DPE-Rec：扩散语言模型做 bundle 构造的多样性保持与高效解码

**元信息**：arXiv 2610.34700 v1（2026-10-26），ASU + Stevens；全文 73,227 字符逐段读取。[abs](https://arxiv.org/abs/2610.34700) | [HTML v1](https://arxiv.org/html/2610.34700v1)

#### 架构组成
1. **骨干**：masked-diffusion 语言模型（DLM，双向注意力），Spotify/POG 上按 DDBC 完全相同配方从头预训练；序列布局 `[BOS],[BOI,z_{1,1..4}],…,[BOI,z_{K,1..4}],[EOS]`（长度 5K+2）；评测时观测 bundle 段可见、目标段全 `[MASK]`。
2. **SID 分词**：4 token/物品。前 3 token 由**残差量化（RVQ）**对固定 **64 维 CLHE** 物品表示生成（码本大小 C=256，Spotify/IQON3000；C=128，POG）；第 4 token 专用于**碰撞消歧**（同前缀物品按目录顺序赋不同值）。词表 = 4C 个 SID token + 3 特殊 token + [MASK]；语义层 m 的 token 映射到不相交词表区间 `tok(z,m)=z+(m−1)C`。
3. **两个后训练组件**：semantic-fork-aware GRPO（教师）→ 伪轨迹蒸馏（学生）。自回归顺序**仅作后训练脚手架**，骨干双向架构与并行解码能力不动。

#### 关键公式与损失函数
- **DLM 前向**（式 1）：`q(x_{t,k}|x_{0,k})`：每个 token 独立以概率 t 替换为 [MASK]。
- **DLM 训练损失**（式 2，加权 CE / ELBO 简化）：`L_DLM = −E_{t~U[0,1], x_t~q}[ (1/t) Σ_k I[x_{t,k}=<m>] log p_θ(x_{0,k}|x_t) ]`。
- **诱导自回归策略**（式 8）：从去噪器 logits 导出带层级温度的策略 `π_θ(o_k=v|o_{<k},c) = exp(s_{θ,k}(v|x^(k))/τ_{m(k)}) / Σ_{v'∈V_{m(k)}} exp(…)`——即对**语义层 m(k) 使用不同温度**的 softmax；整个输出序列可精确链式分解（式 9），为 GRPO 提供可处理的 token 似然。
- **层级集合奖励**（式 11）：`r(o_i) = w₁·|Ẑ_i^2 ∩ Z*^2|/N + w₂·|Ẑ_i^M ∩ Z*^M|/N`——生成的与真值 SID 在 **2-token 前缀级**与**完整 SID 级**的集合交集率。按集合计分 ⇒ 重复预测无额外奖励；粗层给稀疏正样本提供可学信号。默认 (w₁, w₂, G)=(0.2, 0.5, 16)。
- **GRPO 目标**（式 3–5）：组内奖励归一化优势 `Â_i=(r(o_i)−μ_G)/σ_G` + clipped surrogate（ρ_{i,k} 为 token 级重要性比）+ KL 正则。
- **教师轨迹提取**（式 12–13）：置信度 `κ_k^(j)=max_v p_θ(v|x^(j))`，每步只揭示 `π_j=argmax_k κ_k^(j)` 一个位置——**任意序但串行**，保留 token 依赖。
- **轨迹诱导状态**（式 14–15）：对掩码率 t 保留前 `r(t)=⌊(1−t)MN⌋` 个教师决策、其余 MASK——学生只在这个**紧凑轨迹子空间**上训练，而非随机掩码的指数级组合空间。
- **课程学习**（式 16）：`t ~ U[t_min+u(t_max−t_min), t_max]`（u=训练进度）——先易（少掩码）后难（长程去噪）。
- **蒸馏损失**（式 17–18）：`L_TD = L_CE + L_CE^comp`——主视图从轨迹中间状态恢复教师后续决策 + 互补视图（MASK 前 r(t) 个决策、揭示其余）双份监督。
- **并行解码**（式 19–20）：每轮揭示集合 `S_s = {k: max_v p_ψ(v|x^(s)) ≥ η} ∪ Top_{q_s}(M_s)`，其中 `q_s=⌈|M_s|/(F−s)⌉` 保证在步数预算 F 内完成——**置信度阈值 η + 预算兜底**。

#### 算法流程（伪代码级）
```
阶段0 预训练:   DDBC 配方, L_DLM, 从头训练双向 DLM
阶段1 GRPO:    for 每个训练 bundle 上下文 c:
                 按 fork-first 序列化目标段 o=[o_1..o_{MN}]（每个物品槽内先 SID 首 token 后细化 token）
                 从旧策略采样 G=16 个 rollout（组预算 512/update）; τ₁=0.6, τ₂=τ₃=τ₄=0.2
                 计算 r(o_i)（式11, 集合交集）; 组归一化; 更新 GRPO 目标
               按验证集平均 rollout 奖励选 checkpoint
阶段2 蒸馏:    教师 = GRPO 模型, 置信度引导 AO 解码逐 token 出高质量轨迹 (x̃₀, π)
               for 训练进度 u: 采样 t（式16 课程）; 构造轨迹诱导状态 x̃_t（式15）
               学生(从教师初始化) 以 L_TD = L_CE + L_CE^comp 训练
阶段3 推理:    目标段全 MASK; 每轮计算 p_ψ, 揭示 S_s（式19, 阈值 η ∪ Top_{q_s}）直到完成
               SID→物品合法化（DDBC 策略）: 精确命中用之; 否则码本重建表示 + 候选集余弦最近邻; 已选候选移除
```

#### 训练配方与超参
- **硬件**：单卡 NVIDIA A100-80GB（所有实验）。
- 关键超参：(w₁,w₂,G)=(0.2,0.5,16)；rollout 总预算 512/update；τ₁=0.6、τ₂₋₄=0.2（图 5 温度扫描：首 token 峰值 0.6，其余在最低档 0.2 最佳且随温度单调下降）；奖励权重与组大小在测试范围内**不敏感**（图 9）。其余（lr、步数等）在 Table 4，**HTML 图片化未能提取**（残余 gap）。
- 数据增强：每个训练 bundle 生成 ⌊r|B|⌋ 个增广样本（先随机交换一对相邻物品、再取随机连续 K 长子序列）；r=0.4（Spotify）、0.8（POG/IQON3000）；验证/测试不增强。
- 评测协议：候选集 = 全部真值目标 + 均匀负采样，总大小 = **100× 目标数**；指标 F1/Jaccard/OAS（最优一一匹配平均余弦，匈牙利算法）；5 seeds 均值±std。
- 数据规模：Spotify 254,155 tracks；POG 31,217 商品；IQON3000 715,883 时尚物品（去缺图 outfit 后）。

#### 部署工程细节
- **训练成本与摊销**：GRPO 占额外训练成本大头（长 bundle 的串行 rollout）；额外成本在生成 **1.3K–5.0K 个 bundle** 后即被 Fast 变体的推理节省摊销（`N_amort=(T_GRPO+T_PTD)/(L_DDBC−L_Fast)`）。
- **延迟行为**：Fast 变体 62–67 ms 且随 bundle 尺寸几乎不涨（表 2）；相对 3 个生成式基线平均加速 25.6×（对 5 强基线子集 15.6×）。
- **多样性**：生成物品成对余弦相似度在所有对比方法中最低；cluster entropy 全场景最高；目录覆盖 2.2–3.4×（vs DiffGRM/UniGRec）。
- **合法 SID 率**：同等去噪步数下蒸馏学生反超教师（图 7）——蒸馏学到的轨迹状态使多 token 并行预测可靠。

#### 主结果（数字口径）
相对 9 基线（GRU4Rec/SASRec/BERT4Rec/FDSA/RecJPQ/HSTU/UniGRec/DiffGRM/DDBC）× 5 设置 × 3 指标平均 **+63%**；Fast 平均加速 **25.6×**、质量损失极小；强基线子集复核 +26.9% / 15.6×。逐格质量数值（Table 1）需 PDF（残余 gap）。

---

### B2. FairDiff：扩散推荐自强化马太效应的过程层治理

**元信息**：arXiv 2609.36671 v2（v1 2026-09-29，v2 2026-10-19），清华 + 华为诺亚 + USTC；全文 143,591 字符逐段读取（含附录理论）。[abs](https://arxiv.org/abs/2609.36671) | [HTML v2](https://arxiv.org/html/2609.36671v2)

#### 架构组成
- **骨干设定**（Preliminary）：物品嵌入 z₀ 从预定义 embedding 表 E∈R^{n×d} 查表，编码为潜变量 x₀；前向 `q(x_t|x_{t−1})=N(√(1−β_t)x_{t−1}, β_t I)`；反向 `p_θ(x_{t−1}|x_t,c)=N(μ_θ(x_t,t,c), Σ_t I)`，条件 c 为经**双向 Transformer** 编码的序列上下文；训练 = 噪声预测 MSE（式 5，等价学习 score ∇log p_t(x_t|c)）；推理从 N(0,I) 迭代去噪得 user/item 嵌入，与候选内积取 top-K。
- **PCG 模块**（推理期，零参数改动）：去噪器输入端**追加显式流行度条件 p**（由历史交互频次归一化）⇒ `ε_θ(x_t,t,c,p)`。
- **SC 模块**（一次离线拟合）：轻量 MLP 参数化的静态 OT 映射 T。
- 两模块**即插即用**，不改骨干架构与训练目标。

#### 关键公式与损失函数
- **PCG 引导噪声**（式 6–8）：
  - 流行度无关估计：`ε^agn_θ(x_t,t,c) = E_{p̂~q(p)}[ε_θ(x_t,t,c,p̂)]`（实践用**单样本 Monte Carlo** 近似）；
  - 引导合成：`ε̂_θ = ε^agn + w_PCG·(ε^pop − ε^agn)`，w_PCG∈[0,1]；w=0 完全流行度无关、w=1 退化为标准条件去噪；引导后按式 3 常规更新反向均值。
  - 理论等价（定理 0.16）：该引导 ≡ 从 `p̃_t(x) ∝ p_t(x)·p_t(c|x)^λ` 采样——**λ<0 时显式压低高流行区域似然**。
- **SC 静态 OT**（式 9–11）：`min_T E_{x̂_T~N(0,I)} ‖T(x̂_T) − stop_grad(x_T)‖²`（x_T = 真实数据前向扩散到 T 的样本）；推理初始化 `x̃_T = T(x̂_T)`。
- **截断采样**：x̃_T 已对齐数据分布 ⇒ 反向过程早期混沌段（T→T/2）冗余，可直接从 T′<T 起步——**降延迟 + 去掉最易坍向热门的早段**。
- **理论（附录）**：命题 0.2（梯度支配）`E‖∇θL_i‖ ∝ f(i)`；定理 0.5（反向初始化偏差）`P(generate i) ≤ C·p(i)`（长尾先验下尾物品吸引盆测度趋零）；定理 0.8 两偏差**正交且乘性复合**（oracle 训练/采样反事实各自消不掉另一偏差）；结构差距 `Δ_struct = W₂²(q(x_T), N(0,I))` 有下界且 **T→∞ 不消失**（协方差失配）。

#### 算法流程（伪代码级）
```
阶段0: 任意 DRM 骨干按原样训练（去噪网络输入端加流行度条件通道）
阶段1 SC 离线拟合: 对真实用户数据跑前向扩散到 T 得 {x_T}
        训练 MLP 映射 T: N(0,I) → 经验终止分布（MSE, stop_grad 目标）
阶段2 推理: x̂_T ~ N(0,I); x̃_T = T(x̂_T)（可截断: 直接从 T′ 起步）
        for t = T′ … 1:
          ε_pop = ε_θ(x_t, t, c, p)（真实流行度条件）
          ε_agn = ε_θ(x_t, t, c, p̂)（采样辅助条件, 1 次 MC）
          ε̂ = ε_agn + w_PCG(ε_pop − ε_agn);  按式3更新反向均值, 采样 x_{t−1}
        生成嵌入与候选集内积 → top-K
```

#### 训练配方与超参
- 硬件 **A100**；每实验 **5 seeds** 取平均；消融骨干 **CDiff4Rec**；w_PCG 讨论 {0, 0.5, 1}（推荐 0.5）。
- 数据集与基线：见 Part A Gap 3（7 数据集 + CDR 场景；30+ 基线分组列出）。
- 指标：H@K/N@K（K=5,10,20）+ 消费者公平 ΔRecall/ΔNDCG（top-20 列表）+ 提供者公平 ΔExp（曝光不均）/ APLT（长尾覆盖）。
- FairDiff 自身 lr/步数 T 等**未在文本层展开**（残余 gap）。

#### 部署工程细节
- **纯推理期干预**：不改训练管线；PCG 每步仅多一次条件前向（MC 单样本）；SC 一次离线拟合；截断采样反而**减步数**——论文称"几乎不引入额外计算开销"（效率表 Table 7 数值图片化未提取）。
- **公平-效用双赢证据**：所有数据集 ΔExp↓、APLT↑ 且效用同步提升；按流行度分层（Toys）冷段（bottom 20%）增益最大、热段也提升（PCG 解耦而非压制流行度信号）；跨域 3 场景 27 指标全胜。
- **诊断证据**：图 3——随训练进行所有模型冷物品性能持续恶化、热物品持续改善，DRM 两段式放大最显著、FairDiff 使其平稳；图 2 + KL：SC 使前向终止-反向初始分布 KL 10.6447→0.2475、原始-重建 8.2825→0.2383。

---

### B3. ANR-DiffRec：把 item-based 协同结构注入离散扩散的表示与去噪权重

**元信息**：arXiv 2608.23400 v1（2026-08-24），同济 + 华为 + 复旦；全文 108,725 字符逐段读取（含算法伪代码、效率表、案例）。[abs](https://arxiv.org/abs/2608.23400) | [HTML v1](https://arxiv.org/html/2608.23400v1) | [代码](https://github.com/CalmaQi/ANR-DiffRec)

#### 架构组成
1. **共现引导 SID 生成**：Sentence-T5 语义嵌入（物品标题/描述）⊕ **SVD 协作嵌入**——物品-物品共现矩阵 `C∈R^{|I|×|I|}`（C_ij=共现次数）截断 SVD `C≈U_kΣ_kV_k^T`，`e_i^coll=U_k(i,:)Σ_k`（**k=64 维**，消融确定 32→64 升、128/256 反降）→ 拼接 `e_i=[e^sem; e^coll]` → **RQ-KMeans**（L=4 层、每层码本 K=256）残差量化得 SID。
2. **去噪网络**：双向 Transformer encoder（8 heads、hidden 256、4 层；Game 数据集扩到 6 层）；两类掩码训练（用户历史级 + 目标物品级）。
3. **推理**：全 MASK 目标初始化 + LLaDA-Rec 式 **beam search** + **SID 合法性约束解码**——维护候选物品集 `C^t = {i: SID(i)[m]=x_{n,m}, ∀ 已揭示 (m,x)}`，每步只在合法 token 集 `X_m^t` 上重归一化预测分布后再选 token。

#### 关键公式与损失函数
- **自适应噪声重调度核心**（式 12–16）：
  - 结构项（几何衰减）：`W^struct_ij = φ(|I(i)−I(j)|) + 1[I(i)=I(j)]·ψ(|T(i)−T(j)|)`，φ/ψ 均为核 `p(1−p)^{d−1}`（**p=0.1，所有数据集**；网格 {0.1,0.5,0.9} 后取 0.1），分别刻画 inter-item / intra-item 上下文可恢复性；
  - 行为项（注意力）：物品表示 `e_m` = 最后层 token 隐状态均值池化（**detach** 防退化），`A^item_mn = softmax(e_m^T W_Q^T W_K e_n/√d)`，按物品索引投影回 token 对；
  - 合成：`W_ij = W^struct_ij + α·A_ij`（α≥0 可学习）；聚合 `W_i = Σ_j 1[未MASK]·W_ij`——**去噪权重 = 有效噪声水平的逆**：上下文支撑强的 token 权重大（更易恢复）。
- **整体损失**（式 17–19）：`L = λL₁ + L₂`；L₁ = 重加权历史重建 CE（`(1/τ)Σ W_i·1[mask]·log p_θ`），L₂ = 目标物品预测 CE（历史全可见）；**λ 网格 {1..5}**，各数据集取 3/5/2/3/3（Scientific/Instrument/Game/MovieLens/Steam）。

#### 算法流程（论文自带 Algorithm 1/2，伪代码级）
```
离线: 共现矩阵 C → SVD → 64 维协作嵌入 → 与 Sentence-T5 语义拼接 → RQ-KMeans(L=4,K=256) → SID 表
训练(每 batch): 采样历史 S_H + 目标 x_n;  τ₁,τ₂ ~ U(0,1]
  用户历史级: 掩 S_H(τ₁) → 提取最后层物品隐状态(detach) → 注意力 A_ij → 结构权重 W^struct → W_ij
             → 重加权损失 L₁
  目标级: 掩 x_n(τ₂), 历史可见 → L₂;  总损失 L=λL₁+L₂; AdamW 更新
推理: x_n^T 全 MASK + 历史 S_H → for t=T..1:
        预测所有掩码位分布 → 合法 token 集 X_m^t 过滤（候选集 C^t 一致性）
        扩展 top 概率 token → 保留 top-B 累计对数似然 → 更新 C^t
      返回 top-1 beam 对应物品
```

#### 训练配方与超参
- **150 epochs + early stopping**；**AdamW**；**batch 1024**；lr ∈ {0.005, 0.003, 0.001}、weight decay ∈ {0.05, 0.005, 0.001} 网格精调；硬件 **Intel Xeon Gold 6226R + 2× NVIDIA RTX 4090**。
- leave-one-out 协议；R@K/N@K（K=1,5,10）；**16 基线**（ID 系：GRU4Rec/SASRec/BERT4Rec/FMLP-Rec/LRURec/DreamRec/DiffuRec/PreferGrow；SID 系：VQ-Rec/TIGER/TIGER-SAS/LETTER/LC-Rec/RPG/DiffGRM/LLaDA-Rec）；paired t-test p<0.05。
- 数据集统计：Scientific 50,985 用户/25,848 物品/412,947 交互；Instrument 57,439/24,587/511,836；Game 94,762/25,612/814,586；Steam 39,795/9,265/2,949,605；MovieLens 6,040/3,883/1,001,456。
- 残余：扩散步数 T 与 beam size B 的具体取值未在正文给出（算法输入参数级），需查 GitHub 代码默认值。

#### 部署/效率工程细节
- **复杂度不变**：整体 O(M²d)（M=(N−1)L 序列长），结构项 O(M²)、注意力复用最后层隐状态 O(M²d)——与 BERT4Rec/LLaDA-Rec 同阶。
- **tokenization 成本 0.059×**：RQ-VAE 1.799 h vs 本方法 0.106 h（Scientific，含共现矩阵+SVD+RQ-KMeans；RQ-VAE 需昂贵预训练）。
- **训练开销 +1%**：LLaDA-Rec 87.402 → ANR-DiffRec 88.351 s/epoch（总时长 3.641→3.681 h，即 1.011×）。
- **推理近零开销**：beam 1→50 线性扩展；beam=50 时 10.18 vs 10.06 ms/user（vs LLaDA-Rec，Scientific 测试集）。
- **主结果**：Scientific R@1 **0.0122**（LLaDA-Rec 0.0098）；Game R@1 **0.0214**；MovieLens N@5 **0.1468**（LLaDA-Rec 0.1359）；Steam N@10 **0.0526**——5 数据集全指标第一。
- **消融**（Scientific NDCG@1）：Full 0.0122 > RQ-VAE 替换 0.0120 > w/o II（去协作特征）0.0109 > w/o Schedule（均匀噪声）0.0099 > 双去掉 0.0097——**噪声重调度是第一引擎**（论文口径"降幅超 15%"；表值降幅 18.9%）；组件细分：去局部可恢复性估计（w/o LRS）降幅最大 > 去 BDM；inter-item > intra-item。协作信号形式：SVD ≈ LightGCN > Node2Vec。
- **案例**：共现邻居命中（Top-20 内真值共现物品数）SID 空间 #710: 1→8、#6278: 4→12（3×）、#219: 3→7。

---

## C. 遗漏扫描：窗口内（2026-07 下旬 ~ 2026-10-26）未覆盖新论文

首轮与顶会篇均未覆盖、本轮确认的 5 篇（日期均已核验 arXiv 提交页）：

### C1. EPIC: Explicit Posterior Item Conditioning for Semantic ID Diffusion Recommendation
**arXiv 2609.03522，2026-09-03 提交**（v1）；VinUniversity / Griffith / Aalborg；11 页 7 图 3 表。[abs](https://arxiv.org/abs/2609.03522) | [HTML v1](https://arxiv.org/html/2609.03522v1)（本轮已读方法+主结果全文）

**①思考背景**：SID 掩码扩散方法（LLaDA-Rec、DiffGRM、MaskGR、MDGR）改进了 token 生成方式——双向上下文、置信度驱动的灵活揭示顺序，但决策接口始终停留在 token 级，而推荐的最终对象是完整目录物品；每个部分 SID 定义一个随 token 承诺不断收缩的**可行候选物品集**。
**②要解决的核心问题**："token–item 推断层差"——每一次 token 承诺同时是一次物品级淘汰决策，局部合理的 token 会把有希望的物品提前挤出可行集，被挤出的物品后续去噪与事后重排都无法救回；物品级证据必须在候选仍在竞争、仍可达时介入。
**③如何解决**：EPIC 冻结预训练骨干（LLaDA-Rec 的 Sentence-T5 分词器 + 双向 Transformer + 4 步传输调度），仅训一个轻量 item-reasoning adapter：每步用骨干 token 似然（不可微）选 Top-M 候选支撑，构造候选-历史**个性化转移证据**（注意力 + 一致/差异特征）得到归一化物品后验，再把后验**边缘化投影回未解析 SID 位置**，经歧义感知门控（可行性集大小/后验熵/top-2 margin/掩码率/token 熵六特征）以 clipped log 残差注入骨干 logits；**前沿感知学习**只在 2≤|C_t|≤M 的竞争状态施加物品监督（重掩码状态用 SID 距离软标签退火）；全程复用骨干隐状态与 logits，无额外骨干前向。
**④结果亮点**：4 个 Amazon 基准（Beauty/Sports/Toys/Games）**16 个数据集-指标组合全胜**，相对次优 +1.2%~16.8%（Toys/Games NDCG@5 +16.8%），Holm 校正后全部 padj<0.05；inline 反馈比 post-hoc 打分 NDCG@5 高 15.7–26.8%，且存在"控制组最终候选池外"的救回（25/48/49 例）——证明增益来自改变去噪轨迹而非事后重排；换用户历史降 12.4–17.9%（个性化证据是主源）；adapter 仅 0.391M 参数（比全量微调少 94.3%）、训练吞吐 +42.7%、峰值内存 −55.9%，推理延迟 +34.2%（93.0→124.8 ms/user，Tesla T4），解析 2 个 SID 位即可得最佳质量-延迟点（再快 1.72×）。

### C2. Eval4DiRec: A Unified and Systematic Evaluation Framework for Diffusion-based Recommender Systems
**arXiv 2609.34404，v1 2026-09-28 / v2 2026-10-10**；齐鲁工业大学（山东省计算中心）+ UTS + 同济；**已被 ACM TKDD 接收**；代码开源。[abs](https://arxiv.org/abs/2609.34404) | [HTML v2](https://arxiv.org/html/2609.34404v2)（本轮已读框架+实验+发现） | [GitHub](https://github.com/wangcong2001/Eval4DiRec)

**①思考背景**：扩散推荐两年爆发（论文统计了年度发文量与 venue 分布），但各研究在数据处理、切分、训练配置、推理过程、评测协议上互不一致，导致结果不可复现、比较不公平；已有可复现性研究（Benigni et al. 2026）显示 SIGIR 2023/24 的 9 个 DDPM 推荐算法仅少数可完全复现、调优后的简单基线常更强。
**②要解决的核心问题**：RecBole/ReChorus/DaisyRec 等通用框架不暴露也不统一**扩散特有配置**（噪声调度、扩散步数、噪声分布、反向采样过程）——这些设置被内嵌在各模型实现里，逐模型差异使任何跨研究比较都被推理配置混杂。
**③如何解决**：首个扩散推荐专用统一开源评测框架：Data / DiRec / Execution / Utils 四模块，统一数据加载与 leave-one-out 切分、YAML 分组配置、训练-验证-测试管线与 Top-K 指标计算，并把扩散过程与反向采样配置提升为**框架级组件**；当前收录 **14 个代表性扩散模型 × 5 场景**（CF：CODIGEM/DiffRec/CF-Diff/DDRM；序列：DiffuRec/DiffuASR/T-DiffRec/CaDiRec/CDDRec；多模态：MCDRec/DiffMM/CCDRec；POI：Diff-POI；跨域：DiffCDR），配非扩散基线（BPR-MF/LightGCN/BERT4Rec/STOSA/BM3/STAN/PTUPCDR）；RTX 3090 + PyTorch 2.6，超参按验证性能统一精调（lr∈{1e-5..1e-3}、隐层∈{64,128,256}、噪声调度 4 种、步数∈{2..1000}），5 seeds。
**④结果亮点**：统一协议下的系统发现——CF 场景 DiffRec（Yelp/ML-1M 最佳）与 DDRM（Beauty 最佳）最强；序列场景 **CDDRec 全部 3 数据集最佳**；多模态 DiffMM（TikTok）/CCDRec（Baby/Sports）；**扩散步数并非越多越好**（小步数常最优、加步反而掉点）；linear/exponential 噪声调度最稳；高斯噪声最稳、其他分布场景依赖；缺数据（10–50% 删除）与噪声注入（10–50%）下扩散模型退化均缓于非扩散基线；长尾组性能落差更小。为方向 B 提供了首个公平比较基座与"配置敏感性"实证。

### C3. OT-FlowRec: Optimal Transport-Guided Multi-Bridge Flow for Multimodal Sequential Recommendation
**arXiv 2610.16038，2026-10-12 提交**（v1）；cs.IR/cs.AI。[abs](https://arxiv.org/abs/2610.16038)（摘要级调研，HTML 未精读）

**①思考背景**：电商会话内用户偏好动态演化、呈**并发多中心**的多兴趣分布；现有多模态序列推荐用全局注意力压缩历史交互，既抓不住多兴趣行为的并发性，也建模不了用户意图的多中心分布。
**②要解决的核心问题**：单一生成路径（单条流/单一扩散轨迹）在拟合全局分布时会把多峰偏好坍缩成单峰平均——多兴趣表示被"平均化"。
**③如何解决**：OT 引导的**多桥流匹配**：构造并行、兴趣特定的传输流混合体，每条流经一个最优传输计划保持一个独立兴趣簇；再设计几何传输机制在潜多兴趣分布与目标偏好原型间建立最小代价路径，提供多兴趣感知；**渐进时间采样 warmup** 把优化权重动态偏向生成过程末段，弥合"连续全局分布拟合"与"离散 top-K 排序指标"的错位。
**④结果亮点**：3 个公开数据集上超过现有 SOTA 基线；实现显式兴趣解耦，保持多峰偏好表示而非坍缩为单峰（论文自报口径，逐格数值待精读全文核验）。

### C4. MIRAGE（Escaping the Euclidean Void: Manifold-Informed Flow Matching for Sequential Recommendation）
**arXiv 2607.23762，2026-07-26 提交**（v1，窗口起点内）；cs.IR/cs.AI。[abs](https://arxiv.org/abs/2607.23762)（摘要级调研）

**①思考背景**：连续生成式推荐用流匹配把初始噪声沿嵌入空间中的中间状态逐步塑造成确定的下一物品表示；但物品目录离散且支撑稀疏——即使一条"直"的欧氏传输路径也会穿过几乎不含有效物品语义证据的连续区域。
**②要解决的核心问题**：把这一失败形式化为"**欧氏虚空**（Euclidean void）"：流匹配的插值路径状态缺乏物品流形支撑，训练信号与推理质量受损。
**③如何解决**：MIRAGE（Manifold-Informed Rectification）在**不改变直线概率路径**的前提下校正嵌入几何：以**物品共现图**作为底层语义流形的代理，将插值路径状态与局部锚点对齐、重组嵌入空间，把轨迹锚定在有效物品支撑上；关键工程取舍——**图只在训练期使用**，推理保持原概率路径，从而支持准确且高效的**一步推理**。
**④结果亮点**：4 个真实数据集一致超 SOTA；稀疏观测目标（长尾/低支撑物品）提升尤为显著、整体精度稳健（与 FairDiff 的"尾物品失配"诊断在流式侧形成呼应）。

### C5. UHIFlow（Anatomy of a Decision: Uncertainty-aware Hierarchical Intent Learning via Flow Matching for Multimodal Recommendation）
**arXiv 2609.29609，2026-08-30 提交**（v1）；**已被 WISE 2026 接收**。[abs](https://arxiv.org/abs/2609.29609)（摘要级调研）

**①思考背景**：意图建模对推荐至关重要，但现方法依赖聚类或原型学习得到**静态**意图集合，忽视两件事：多模态特征的内在不确定性，以及用户兴趣的动态层级结构。
**②要解决的核心问题**：静态、平坦的意图结构无法适应用户决策确定性的变化——同一套意图粒度同时服务"犹豫型"与"明确型"用户必然次优。
**③如何解决**：两个模块。**CUSM**（跨模态不确定性协同建模）用条件流匹配分别量化视觉/文本模态的不确定性并协同对齐；**UHIG**（不确定性引导的层级意图生成）用量化出的不确定性动态构建个性化意图层级——不确定用户生成**粗粒度**意图、偏好明确用户生成**细粒度**意图。
**④结果亮点**：3 个真实数据集上显著超 SOTA（论文自报口径；逐格数值待精读全文）。

### 边界与关联补充（一句话，均未计入正式 5 篇）

- **PrismRec**（Preference Flow Matching with Spectral Factorization，[arXiv 2608.26579](https://arxiv.org/abs/2608.26579)，2026-08-27，窗口内）：微视频推荐；SSF 用先验引导的可学频率掩码在时频域把帧级表示分解为静态语义/动态因子，CPM 按用户敏感度加权并作为结构化条件注入匹配轨迹；4 数据集（两平台）超 SOTA 最高 **+22.65%**，推理成本与峰值内存最低。（摘要级；偏多模态流匹配，列为方向 B 关联）
- **MDGR**（Masked Diffusion Generative Recommendation，RecSys 2026 论文集 **2026-09-27 出版**，[DOI 10.1145/3773078.3831862](https://dl.acm.org/doi/full/10.1145/3773078.3831862)；arXiv 2601.19501 为 2026-01-27，**窗口外**故不入正式列表）：并行码本 + 时间/样本双维度自适应掩码监督 + warm-up 两阶段并行解码；公开+工业数据集超 10 基线最高 +10.78%，**在线广告平台部署营收 +1.20%**（阿里国际）——是本方向窗口内少有的工业线上收益证据。
- **MaskPoison**（[DOI 10.1145/3773078.3831778](https://doi.org/10.1145/3773078.3831778)，RecSys 2026，2026-09-25 出版）：经掩码离散扩散对序列推荐发起意图引导投毒攻击——扩散推荐安全侧首个工作（摘要级）。
- **A2G-DiffRec**（Adaptive Autoguidance for Item-Side Fairness in Diffusion Recommender Systems，SIGIR 2026；arXiv [2602.14706](https://arxiv.org/abs/2602.14706) 为 2026-02，窗口外）：自适应自引导增强扩散推荐物品侧公平——与 FairDiff 同主题的前置工作，建议对照阅读（[GitHub](https://github.com/zihanlizi/A2G-DiffRec)）。
- **索引边界现象**（标注日期晚于今天 2026-10-26，未纳入）：DARTS（2610.36954，标注 10-27，首轮已在 research.md G 节列出）、Latent Residual Reasoning（2610.36890，标注 10-27）、SourceFlow（2611.04455，标注 11-03）。
- 窗口外相关但常被混检：FAVE（SIGIR 2026，arXiv 2604.04427，4 月）、FlowRec（2508.17618，2025-08）、TA-Rec（2510.17245，2025-10）、IGDMRec（2512.19983，2025-12）、Continuous-time Discrete-space Diffusion（WSDM 2026，arXiv 2511.12114）、FlowConsist（2602.06346，2 月）——均已按日期排除。

---

## 来源清单

**保留（本轮一手）**：
- [DPE-Rec arXiv HTML v1](https://arxiv.org/html/2610.34700v1)（73,227 字符全读）— 公式 1–29、延迟表全量、消融/成本/多样性数字、附录 8.1–8.7
- [FairDiff arXiv HTML v2](https://arxiv.org/html/2609.36671v2)（143,591 字符全读）— 公式 1–11、理论附录（命题/定理 0.2–0.16）、数据集/基线全清单、KL 四位小数、消融
- [ANR-DiffRec arXiv HTML v1](https://arxiv.org/html/2608.23400v1)（108,725 字符全读）— 公式 1–22、Algorithm 1/2、主表/消融/效率表逐格、案例
- [EPIC arXiv HTML v1](https://arxiv.org/html/2609.03522v1)（方法+主结果全读）+ [abs 页](https://arxiv.org/abs/2609.03522)（日期核验：2026-09-03）
- [Eval4DiRec arXiv HTML v2](https://arxiv.org/html/2609.34404v2)（框架+实验+发现全读）+ [abs 页](https://arxiv.org/abs/2609.34404)（日期：v1 09-28 / v2 10-10，TKDD 接收）
- [OT-FlowRec abs](https://arxiv.org/abs/2610.16038)（2026-10-12）、[MIRAGE abs](https://arxiv.org/abs/2607.23762)（2026-07-26）、[UHIFlow abs](https://arxiv.org/abs/2609.29609)（2026-08-30，WISE 2026）、[PrismRec abs](https://arxiv.org/abs/2608.26579)（2026-08-27）— 日期与摘要逐页核验
- OpenAlex API 4 查询（[2610.34700](https://api.openalex.org/works/doi:10.48550/arXiv.2610.34700) / [2609.36671](https://api.openalex.org/works/doi:10.48550/arXiv.2609.36671) / [2608.23400](https://api.openalex.org/works/doi:10.48550/arXiv.2608.23400) / [10.1145/3831688](https://api.openalex.org/works/doi:10.1145/3831688)）— 引用数=0 核验
- [ACM DL ScoreRec 页](https://dl.acm.org/doi/10.1145/3831688)（元数据经搜索索引核验：accepted 2026-05-16 / online 2026-09-30 / 卷期 2026-10-21）
- GitHub：[CalmaQi/ANR-DiffRec](https://github.com/CalmaQi/ANR-DiffRec)、[AONE-NLP/ScoreRec](https://github.com/AONE-NLP/ScoreRec)、[wangcong2001/Eval4DiRec](https://github.com/wangcong2001/Eval4DiRec)、[241416/DDBC](https://github.com/241416/DDBC)（前作对照）
- [MDGR ACM DL](https://dl.acm.org/doi/full/10.1145/3773078.3831862) + [arXiv 2601.19501](https://arxiv.org/abs/2601.19501)（出版日期边界核验）

**排除（及原因）**：
- `dl.acm.org/doi`、`dl.acm.org/doi/epdf`、`dlnext.acm.org/doi/epdf`（ScoreRec 正文）— 三次尝试均 HTTP 403，正文不可达
- Semantic Scholar Graph API — 持续 429 限流（"Too Many Requests"），引用数改由 OpenAlex 覆盖
- ebiotrade / scholarbro / arxiv.science / paperlayer — 二手转述或聚合站
- [danielemalitesta/FairDiffRec](https://github.com/danielemalitesta/FairDiffRec)（RecSys 2025 LBR"How Fair is Your Diffusion Recommender Model?"）、[zihanlizi/A2G-DiffRec](https://github.com/zihanlizi/A2G-DiffRec)（SIGIR 2026）— 与 FairDiff 同名/同主题但**不同论文**，仅作区分标注不作来源

---

## 剩余 Gaps（本轮之后）

1. **ScoreRec 正文逐格数值（硬缺口）**：ACM 三个端点均 403、无 arXiv 版本——需机构订阅下载 PDF 或联系作者；GitHub README + 摘要交叉验证仍为最优替代。已确认出版链：2026-05-16 录取 → 2026-09-30 online → 2026-10-21 卷期出版。
2. **DPE-Rec Table 1/3/4/5、FairDiff Table 1–4 逐格数值**：arXiv HTML 中这些表格渲染为图片、不进文本层（同批文档中 DPE-Rec Table 2 与 ANR-DiffRec 全部表格均可提取，说明是特定表格的渲染问题）——两篇均 CC-BY 4.0，下载 PDF 即可补齐；FairDiff 自身超参（lr、T、w 网格明细）与 DPE-Rec Table 4 超参明细同此。
3. **ANR-DiffRec 的扩散步数 T 与 beam size B 取值**：正文只到"算法输入参数"级，具体值需读 GitHub 代码默认配置。
4. **OT-FlowRec / MIRAGE / UHIFlow / PrismRec 仅摘要级**：本轮重心在 A/B 两项；四篇如需可复现级细节，其 arXiv HTML 均已存在可下轮精读。
5. **引用数时效**：OpenAlex 2026-10-26 快照四篇均 0；三篇 arXiv 论文发表不足 3 个月（DPE-Rec 仅 1 天），"无第三方复现"是时点结论，1–2 个月后建议复查。
6. **线上收益证据仍缺**：DPE-Rec/FairDiff/ANR-DiffRec/ScoreRec 四篇均无 A/B；方向 B 窗口内最接近线上证据的是 MDGR（在线广告营收 +1.20%，但其 arXiv 日期在窗口外）。

