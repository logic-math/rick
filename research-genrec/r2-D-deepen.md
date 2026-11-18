# 方向 D 第二轮深挖简报：生成式召回与排序（多目标 / 解码 / 工业落地）

调研日期：2026-10-26。范围：首轮方向 D 文件（D-recall-ranking-decoding.md）的 Gaps 逐条补齐 + 3 篇可复现级深挖 + 窗口内（2026-07 下旬 ~ 2026-10-26）遗漏论文扫描。

**方法说明**：首轮调研中 arXiv HTML 的"readable"正文抽取丢掉了全部表格单元格（只留表题），本轮改用 raw 模式抓取五篇论文的 arXiv HTML 源码（LaTeXML 输出中表格数值为纯文本 `<td>`），成功恢复了全部关键表格数值；所有数字均可通过论文 HTML 页面直接核验，未做任何推算（TacGR 的 OneRec-V2 / TIGER 行、MD-OneRec 的 HSTU 行仅部分恢复，已在文中注明）。

## Summary

首轮 4 条 Gaps 已补齐 3.5 条：SiDec 线上 Table 1 数值已从 arXiv HTML 原文恢复（Watchpage 排序 +0.80%、Homepage 排序 +0.22%、检索 +0.13%/+0.09%，但仍未披露 A/B 流量比例/时长/p 值）；TacGR 与 Multi-Decoder OneRec 的离线主表绝对值、PushDualGen 的 Pass@k 消融表已全部恢复；OneLA 确认为研究原型（无生产部署声明）。深挖三篇（Multi-Decoder OneRec / OneLA / TacGR）已到公式+伪代码+训练配方级。遗漏扫描发现 5 篇重要新论文（SnapLGR、OGR、GRACE、WeGR、AR-BS vs MTP）及 6 篇一行级补充，显示"解码系统工程"（GPU 约束解码、宽 beam serving）成为窗口内新增的独立战线。

---

## A. 首轮 Gaps 逐条补齐

### Gap 1：SiDec（YouTube）线上 A/B 细节 —— 部分补齐

- **线上 Table 1（arXiv HTML §3.4.2 Table 1，已从 raw HTML 恢复）**：[来源](https://arxiv.org/html/2607.24865)
  | 部署位 | 使用的 SID 特征 | Online Satisfied Engagement 相对提升 |
  |---|---|---|
  | Watchpage 排序模型 | 正在观看视频 + 候选视频 + 观看历史的 SID | **+0.09% Sitewide，+0.80% Watchpage** |
  | Homepage 排序模型 | 候选视频 + 观看历史的 SID | **+0.08% Sitewide，+0.22% Homepage** |
  | 检索模型 | 观看历史的 SID | **+0.13% Homepage，+0.09% Watchpage** |
- **部署范围与设定（§4.2）**：同时部署于多任务生产排序模型（Singh et al., 2024 架构）与基础 Transformer 检索模型；排序侧用三类 SID 特征（候选视频、正在观看视频、用户观看历史），协同身份流采用嵌套 n-gram 或 SPM 嵌入策略；检索侧 SiDec 仅作用于用户观看历史（用于在不违反时延/资源约束下为 ID embedding 桥接语义鸿沟）。[来源](https://arxiv.org/html/2607.24865)
- **净增益隔离**：基线系统已含协同身份流（SID token embedding），论文明确所报指标"仅隔离新增 SiDec 内容重建流的纯提升"；线上指标为专有复合指标 Online Satisfied Engagement（反映长观看时长与点赞/收藏等正互动）。[来源](https://arxiv.org/html/2607.24865)
- **离线消融 Table 3（CTR AUC 相对变化，已恢复）**：仅保留候选+正在观看的 SID（去掉观看历史 SiDec 流）→ **−0.04%**；仅保留观看历史 SID → **−0.03%**；去掉轻量语义解码器 f_θ → **−0.01%**——绝大多数增益来自观看历史 SID（历史条数多，最吃 SiDec 的 I/O 效率）。[来源](https://arxiv.org/html/2607.24865)
- **仍未披露**：A/B 流量比例、实验时长、显著性水平（原文仅定性表述"at YouTube's massive scale ... highly statistically significant"，无 p 值/置信区间）；ACM DL 正式版（[DOI 10.1145/3773078.3831900](https://dl.acm.org/doi/10.1145/3773078.3831900)）仍 403 不可达。此为**未能补齐部分**。

### Gap 2：TacGR / Multi-Decoder OneRec 离线结果表 —— 已补齐

**TacGR 主表（Table 1，raw HTML 恢复，列序：CLK HR@100 / PAY HR@100 / CLK HR@500 / PAY HR@500 / CLK AUC / PAY AUC / CLK GAUC / PAY GAUC）**：[来源](https://arxiv.org/html/2610.16132v1)
| 模型 | CLK HR@100 | PAY HR@100 | CLK HR@500 | PAY HR@500 | CLK AUC | PAY AUC | CLK GAUC | PAY GAUC |
|---|---|---|---|---|---|---|---|---|
| OneRec-V1 | 0.3007 | 0.3711 | 0.5580 | 0.6207 | — | — | — | — |
| UniSGR | 0.3031 | 0.3808 | 0.5645 | 0.6321 | 0.6750 | 0.8438 | 0.6222 | 0.5989 |
| **TacGR (Ours)** | **0.3148** | **0.3857** | **0.5849** | **0.6490** | **0.6786** | **0.8505** | **0.6261** | **0.6018** |

（TacGR 行全部加粗最优；相对 UniSGR：检索四指标最高 +3.86%，AUC +0.53%/+0.79%，GAUC +0.63%/+0.48%；TIGER / OneRec-V2 / UniRec(CoA) 行未逐格恢复，正文相对数字为：对 UniRec(CoA) 四检索指标 +1.50%~+3.11%。）

**Multi-Decoder OneRec 主表（Table 1，raw HTML 恢复，列序：Exposure / Long-View / Like / Watch-time 的 Recall@512）**：[来源](https://arxiv.org/html/2607.26500v1)
| 模型 | Exp. | Long-View | Like | WT |
|---|---|---|---|---|
| DSSM（ANN Top-512 全库） | 0.0046 | 0.0050 | 0.0039 | 0.0050 |
| SASRec（ANN） | 0.0310 | 0.0329 | （未恢复） | （未恢复） |
| OneRec（单解码器基线） | 0.1539 | 0.1833 | 0.1318 | 0.1923 |
| **Multi-Decoder OneRec (Ours)** | **0.1565** | **0.1907** | **0.1391** | **0.2031** |

（相对单解码器 OneRec：+1.69% / +4.04% / +5.54% / +5.62%，与正文一致；HSTU / TIGER 行仅部分恢复（TIGER 行可见 0.1613 / 0.1161 / 0.1660 三个值，列位未完全确认，不作引用）。）

### Gap 3：PushDualGen 离线消融（Pass@k）—— 已补齐

Table 2（SID 预测任务，Pass@1 / Pass@4 / Pass@20，raw HTML 恢复）：[来源](https://arxiv.org/html/2608.07989v1)
| 变体 | Pass@1 | Pass@4 | Pass@20 |
|---|---|---|---|
| **PushDualGen（完整）** | **0.335** | **0.391** | **0.422** |
| w/o Token Freeze | 0.333 | 0.386 | 0.412 |
| w/o Multi-token Binding | 0.332 | 0.384 | 0.416 |
| w/o SID Adaptation | 0.319 | 0.352 | 0.366 |
| w/o Parallel SID（换用 OneRec 的 SID） | 0.326 | 0.379 | 0.401 |
| Qwen3-0.6B（原始模型直接在 OneRec SID 上训练） | 0.312 | 0.347 | 0.358 |

结论细化：**SID Adaptation 是最关键模块**（Pass@1 掉 0.016，LLM 预训练从未见过 SID，必须专门适配）；Token Freeze 与 Multi-token Binding 主要优化训练过程，增益相对较小但为正；各模块可叠加。线上数字维持首轮（14 天、约 1.5 亿用户、CUPED 校正：有效播放率相对 +8.50%、负反馈率相对 −37.70%，均 p<0.05）。[来源](https://arxiv.org/html/2608.07989v1)

### Gap 4：OneLA 生产部署状态 —— 已补齐（确认为研究原型）

- 全文（abs + HTML）**无 "production / deployment" 声明**（raw 全文检索无命中）；评测为原型系统在工业 GR 工况上的受控实验：0.8B Qwen3.5 混合注意力模型（6 层全注意力 + 18 层 GDN）、R=4、W=256、prompt 1K/5K、O=3–7，对比未修改的 vLLM/SGLang FullState/ReplaySSM 内核、FlashInfer、TensorRT-LLM。[来源](https://arxiv.org/html/2609.12399v1)
- 受控基线有效性已验证：其 Controlled FullState 管道与原生 vLLM FullState 差距仅 6.9–9.3%，说明 1.54–2.46× 端到端加速归因于 OneLA 的状态管理而非工程差异。[来源](https://arxiv.org/html/2609.12399v1)
- 生态背景（OneLA 引文）：GR serving 相关线已有 xGR（[arXiv 2512.11529](https://arxiv.org/abs/2512.11529)，2025-12，窗口外）；GDN+beam 端到端仅 vLLM 原生支持，SGLang/FlashInfer/TRT-LLM 仅算子级——OneLA 填补的是"线性注意力 + 大 beam"这一空档。[来源](https://arxiv.org/html/2609.12399v1)

---

## B. 方法内部深挖（三篇，至可复现级）

选取依据：三篇分别对应本方向三大支柱——多目标（Multi-Decoder OneRec）、解码（OneLA）、工业落地中的目标侧特征（TacGR）。

### B1. Multi-Decoder OneRec（快手，[arXiv 2607.26500](https://arxiv.org/abs/2607.26500)，2026-07-29）

**① 架构组成**
- **共享用户上下文模块（encoder）**：每个请求构造两条时序截断行为视图——最近 20 条交互 + 256 条 Long-View 视频；每条视频为五字段特征元组（Item-ID、author ID、tag、时间差、观看时长）→ embedding → 4 个 encoder block（每块 = 特征级 cross-attention + 序列级 self-attention），输出 H_u = Enc(H_u^τ)，**每请求计算一次**，作为所有解码路由 cross-attention 的 K/V 复用。
- **General Decoder（全参，Decoder0）**：4 个 Transformer decoder block（masked self-attention + cross-attention + FFN）；输入 = BOS embedding ⊕ 右移目标 SID 序列。
- **目标专家（每目标轻量增量）**：LoRA 挂在 masked self-attention 与 cross-attention 的 Q/K/V 投影（式 8：W^(t)=W₀+B^(t)A^(t)）＋独立可学习 BOS embedding b_t ＋ SID 词表残差 ΔE_t（式 2：e_{r,ℓ}(z)=(E_{gen,ℓ}+ΔE_{r,ℓ})[z]，ΔE_{gen}=0）。可训练状态 φ_t={A^(t),B^(t),b_t,ΔE_t}，与底座及其他专家完全隔离。
- **路由集合**：R={gen}∪T；离线 T={Long-View, Like, Watch-time}，**线上 T={Watch-time, Share, Cold-Start}**（仅标签与监督不同）。

**② 关键公式与损失（编号同原文）**
- 基座 NTP（式 10）：L_base = −Σ_{ℓ=0}^{L−1} log p_gen(y_ℓ | c, y_{<ℓ})（曝光样本）。
- 目标过滤 SFT（式 11）：L_SFT^(t) = −Σ_i Σ_ℓ m_i^(t) log p_t(y_{i,ℓ} | c_i, y_{i,<ℓ})（Like 解码器只用 like 样本、Long-View 只用长播样本）。
- 连续时长目标的标准化相对奖励（式 12）：r_i = (w_i − μ(G_i)) / (σ(G_i)+ε)，G_i = 当前观看时长 ∪ 用户最近 K 条有效历史观看时长（把 GRPO 的组内归一化改造为"用户历史内归一化"）。
- **L-GBPO**（式 13–14）：stop-gradient 分母 p_old,i,ℓ = max(p_ref,i,ℓ, sg(p_t,i,ℓ))（r≥0）或 max(p_ref,i,ℓ, 1−sg(p_t,i,ℓ))（r<0）；L_L-GBPO = −E[r_i · p_t,i,ℓ/(p_old,i,ℓ+ε)]；参考分布 p_ref 由**当前 General Decoder 的 stop-gradient 前向**提供。
- KL 锚定（式 15–16）：ρ=p_ref/p_t，L_KL = E[ρ − log ρ − 1]（非负 token 级 KL 代理）；总 RL 损失 L_RL = λ_L-GBPO·L_L-GBPO + λ_KL·L_KL（式 17）。
- **梯度路由总损失（式 18）**：L = L_base(θ₀) + λ_SFT Σ_t L_SFT^(t)(sg(θ₀), φ_t) + Σ_t L_RL^(t)(sg(θ₀), φ_t)——基座 NTP 是 θ₀ 的唯一更新来源；各目标损失只更新自己的 φ_t（并发训练、批次可交错，靠梯度路由而非时间冻结隔离）。

**③ 算法流程（MD-CBS 伪代码级）**
```
输入：路由优先序 π（General Decoder 永远最后），各路由 (r, b_r, q_r)，
      约束层级 d（默认 d = L−1 = 2，即以完整 SID 三元组为去重键）
K ← ∅  # 已占用键集合
for r in π[0..M-1]:           # 依次执行各目标路由
    C_r ← ∅
    对 r 做 beam_size=b_r 的受限 beam search：
        score s'_r(z_{0:d}) = -∞  若 z_{0:d} ∈ K   # 屏蔽已被占用的 d 级前缀
        否则 s'_r(z_{0:d}) = s_r(z_{0:d})
    跳过被屏蔽候选，沿 beam 列表继续，直到接受 q_r 个唯一候选或 beam 耗尽
    K ← K ∪ {本轮接受的键}；记录 C_r
r ← gen：同样屏蔽已占键，回填剩余配额（最多 q_gen 个未见完整 SID）
合并：Merge_π({(R_r, q_r)})，按路由序去重，得到 |C(u,B)| ≤ B 的候选池
```
约束层级 d 的权衡：d 小 → 更早剪掉重复前缀、省解码算力，但过早封死同粗语义区的其他物品；d 大 → 保留搜索广度、解码更贵。消融：无 CBS 时去重后平均只剩 **207.67** 个唯一候选；L2/L3 CBS 都能恢复满 512 预算，L3（完整 SID 键）在全部四指标上最优，故为默认。

**④ 训练配方与超参**
- 模型：hidden size 256；encoder/decoder 各 4 块、8 头；SID 3 层、每层 8,192 码字；与 OneRec/TIGER 基线共享 SID 词表与时序切分。
- 数据（Kwai26，[GitHub 数据管线](https://github.com/liuzhao09/Kwai26-Data-Pipeline)）：60 天工业短视频日志；13.1 亿原始记录 → 8.2184 亿正播放交互 → 1.2526 亿训练 session；每用户留 1 个 held-out session（5 万测试用户）；目录 25.03M 有效 SID 物料（31.85M Item-ID）。
- 训练：单轮时序遍历（one chronological pass），全局 batch 1,024；LoRA rank **32**（消融：rank 4/64 在不同指标上列第二，16 全指标最差，32 全指标最优）；KL 权重 **1.0** 最优（去掉 KL → 召回与 SID 合法率严重坍缩）；奖励历史 K=**500**（K=8→500 使 WT Recall 0.2003→0.2031，+1.40%）。
- 推理：beam sizes（Long-View/Like/Watch-time/General）= 86/171/256/512，配额 = 86/85/85/256（总预算 512）。
- 开销：3 个 LoRA 专家仅 +20% 参数；训练与四路推理 FLOPs 分别 1.54× / 2.23×（按 session 计）。
- **未披露**：优化器类型、学习率、warmup（已抓取正文中无相关段落）。

**⑤ 部署工程**
- Serving：所有解码器共享**一份物化的 encoder 表示**；每个解码器只附加自己的 BOS/SID 残差/LoRA 状态/beam/配额；**目标集合与配额可在线调整而不动 General Decoder 与其他专家**。
- 生产环境：快手巴西短视频全量管线，生成式召回贡献全站 57% 曝光；A/B 7 天，按设备 ID 随机化，对照组 14.64% DID（单解码器 OneRec）vs 实验组 7.32% DID（增加 Watch-time/Share/Cold-Start 三解码器与显式配额），其余排序/重排完全一致。
- 线上结果（Table 4，全部 p<0.05）：人均使用时长 **+0.37%**、Day-1/7 留存 +0.12%/+0.19%、互动 +0.19%~+0.52%、新内容冷启动 **+2.09%**；后验分析：Share 解码器在 FTR（首次触发率）领先、Cold-Start 解码器在冷启率领先、Watch-time 解码器在平均观看时长领先——证实解码器分工在下游 serving 中成立。
- 配额方向控制（Fig.4）：配额向 Long-View 倾斜 → Long-View Recall +0.10% 但 Exp/Like 微降；向 WT 倾斜 → WT +0.31%（最大定向变化）；**全部路由 beam 扩到 512 → 四指标全部为正**（配额控方向、beam 控质量）。

**⑥ 可复现要点**：Kwai26 数据集与切分/评测协议公开（泄漏安全处理、时序训练布局见论文附录 B）；SID 词表、batch、beam/配额、LoRA rank、KL 权重、奖励历史长度全部可得；缺优化器与学习率细节。

### B2. OneLA（港大+快手，[arXiv 2609.12399](https://arxiv.org/abs/2609.12399)，2026-09-11）

**① 问题设定与工作负载特征**
- GR 解码工况（承 xGR 刻画）：长共享 prompt + 短 SID 后缀 + 大而动态演化的 beam，即 **L_prompt ≫ T 且 W_t ≫ T**；每步全局 top-k 剪枝/重排/复制形成动态 beam 树（父映射 π_t(b) 每步变化）。
- GDN（Gated DeltaNet）每 token 一次状态更新（式 1）：S_t = α_t·S_{t−1} + β_t(v_t − α_t·S_{t−1}k_t)k_t^T，o_t = S_t·q_t；S_t ∈ R^{d_v×d_k}。现有服务的两条路径都不适配：vLLM FullState 为每个活 beam 物化完整状态矩阵（Θ(w·d_v·d_k) 持久容量）；vLLM ReplaySSM 只在 prompt 块边界做检查点、把每个 beam 当独立请求重放 prompt 尾部。beam 每步重排/复制导致状态拷贝或重算。

**② 核心表示：GDN Transition Record（GTR）与投影重放**
- 把一次 GDN 更新改写（式 2）：S_t = α_t·S_{t−1} + δ_t·k_t^T，其中 δ_t = β_t(v_t − α_t·S_{t−1}k_t)；**GTR r_t = (α_t, δ_t, k_t)**——标量衰减 + d_v 维更新向量 + d_k 维键向量，存储 O(d_v+d_k)（对比完整状态 O(d_v·d_k)）。存 δ_t 而非 (β_t, v_t) 的原因：重放永远只需要状态在当前 q/k 上的两个投影，δ_t 的贡献只差一个点积；存 (β_t,v_t) 则依赖父状态、必须重建矩阵，违背设计初衷。
- **投影重放**（式 3–4）：沿当前 beam 祖先链 r_0..r_{t−1}，两个累加器递推
  u_q^(−1)=S_ctx·q_t，u_q^(j) = α_j·u_q^(j−1) + δ_j(k_j^T·q_t)；u_k 同理（换成 k_t）；
  则当前步 δ_t = β_t(v_t − α_t·u_k^(t−1))，o_t = α_t·u_q^(t−1) + δ_t(k_t^T·q_t)。
  **每 beam 每 value 头只需 2 个 d_v 维累加器，全程不物化任何 d_v×d_k 矩阵**；S_ctx（prompt prefill 后状态）每请求只存一份。

**③ 动态 beam：ancestry 索引（式 5）**
```
h_t(b, j) = h_{t-1}(π_t(b), j)   若 0 ≤ j < t-1   # 继承被选中父 beam 的索引
h_t(b, j) = π_t(b)               若 j = t-1       # 记录父的物理槽位
```
GTR 追加写入当前步物理槽位后**永不迁移**；逻辑 beam 树完全由 ancestry 索引表达：剪枝=丢弃引用、重排=改引用顺序、分叉=只复制索引（不拷贝记录链）。

**④ 融合 GPU kernel**
- 每个 thread block 负责 value 维一个 B_V 行 tile + 一组 beam；**S_ctx[B_V, d_k] tile 只从 HBM 加载一次**、被块内所有 beam tile 复用；单一 kernel 内融合"共享上下文投影 → 按 ancestry 索引收集 GTR 并重放 → 当前 GDN 更新 → 输出"，中间值全在片上，**只追加写回紧凑 GTR**（无全状态写回）。

**⑤ 评测配置与数字（复现基准）**
- 全模型：0.8B Qwen3.5（6 全注意力层 + 18 GDN 层）；R=4、W=256、prompt 1K/5K、O=3–7；Controlled FullState 与原生 vLLM FullState 差 6.9–9.3%（受控有效性）。
- 算子套件：405 个形状（R=4/8/16、W=128/256/512、T=2–6、9 种循环几何 H_K=4–32 / H_V=4–64 / K=128–256 / V=128–512），配对对比未修改的 vLLM/SGLang FullState+ReplaySSM、FlashInfer、TensorRT-LLM。
- 结果：端到端解码 **1.54–2.46×**；算子级中位加速 40.5×/42.5×（vLLM/SGLang FullState）、17.5×（两 ReplaySSM）、41.5×/46.3×（FlashInfer/TRT-LLM）；最大 Qwen 几何（R=16,W=512,H=16,K=V=128）下 0.212ms@O=3 / 0.990ms@O=7（对该形状四基线 16.0–56.5×）；**最大绝对数值误差 4.88×10⁻⁴**（对 FP32 FullState）。
- 记忆/流量（R=4,W=512）：持久循环状态 36.3/101 MiB@O=3/7（较 FullState 降 56.5×/20.3×）；状态写入 32.3/96.8 MiB（**降 127×**）；O=3 时 DRAM 物理流量 161 MiB（对 FullState/ReplaySSM 降 76.9×/39.9×）；算术强度 40.0 FLOP/byte（高 45.6×/20.6×）。
- **未披露**：kernel 的 B_V tile 尺寸、SM 占用等实现级参数；生产集群指标（确认为原型）。

### B3. TacGR / UniSGR-V2（阿里国际 Lazada，[arXiv 2610.16132](https://arxiv.org/abs/2610.16132)，2026-10-12）

**① 五模块架构**
1. **Memory encoder**（含 GCD 去噪）：行为序列（倒序，最新在前）→ 长期记忆 M ∈ R^{n×d_m}；
2. **MoE decoder**：GQA 注意力 + 共享 SwiGLU-MoE block，经 cross-attention 读 M（可见范围由 memory funnel 调度），自回归生成 3 层 SID；
3. **Cascaded sideinfo anticipator（CSA）**：每个 SID 码字前一个预测点；
4. **UVM 单向可见性掩码** + **PLE 多任务排序头**（CTR/ATC/CVR，输入 [h_dec; ResProj(φ̃^(L))]）；
5. **锚定辅助塔**（仅训练期存在，推理时完全移除）。

**② CSA 与 in-decoder crossing（核心公式）**
- 预测点放在 s^{l−1} 之后、s^l 之前（s⁰:=BOS），每 item 的 decoder 输入从 L+1 token 扩为 **2L+1 token**；预测条件逐级变富（式 3）：φ^(1) = c_u（冷启动先验：历史点击物品侧特征的 masked 均值池化，式 4，不参与回归损失）；l≥2 时 φ^(l) = Pred^(l)([h^(0);…;h^(l−2)], e(s^{l−1}))——刚解码出的 s^{l−1} 的 embedding 查表立即使预测粒度从 s^{l−2} 提升到 s^{l−1}；每级预测点有独立 token-type embedding 与输出投影。
- Crossing：ψ(φ^(l)) 作为 token 插入 decoder 序列；在 UVM 下经 self-attention 读语义前缀、cross-attention 读 M，得到交叉表示 φ̃^(l)（**单向**：预测 token 可读语义/记忆，语义 token 不可读预测 token）。
- 双用途（式 5–6）：生成侧 P(s^l|·) = softmax(W_si^(l)[h^(l−1); φ̃^(l)])（级专属投影，梯度可回传到预测分支）；排序侧 x_rank = [h_dec; z_L]，z_L = ResProj(φ̃^(L)) → PLE 三头。
- 为什么连续向量而非离散属性 token（vs UniRec CoA）：① 解码步数与属性数解耦（毫秒级 SLA 下属性 token 线性拉长不可行）；② 错误容忍（连续位移=软扰动，离散错 token=不可逆换路径）；③ 信号带宽（可表达"介于两类目之间"等不可离散化的语义梯度）。

**③ 两个退化风险与对策**
- **回归标签塌缩**（Proposition 1）：标签 φ_θ(v) 本身可学习，若 θ 只从 L_feat = Σ_{l=2}^L ‖φ^(l) − φ_θ(v)‖₂²（式 7）收梯度，存在常数映射平凡解（L_feat=0 且 I(φ_θ(v); v)=0）。对策：**锚定辅助塔**（式 8）——以真值 φ_θ(v) 为输入、独立 PLE + 三头，BCE(ctr/atc/cvr) 直接给 θ 业务对齐梯度；φ_θ 塌缩则辅助塔退化为只依赖 h_dec 的预测器、L_anchor 无法再降，塌缩被惩罚；仅训练期存在、零在线开销。
- **误差传播/自依赖退化**：UVM（式 9）：query i 可见 key j ⟺ j=BOS ∨ j=i ∨ (j<i ∧ j∈S)——BOS 全局可见、token 可见自身、只能回看**语义** token；预测 token 互不可见、语义 token 永不看预测 token。误差隔离分析：h^(l) 与所有 ε 独立（语义路径如同预测 token 不存在）；每级预测的输入（缓存隐状态 + 真值码 embedding，teacher forcing 下）均无预测误差，**误差逐级隔离、不在层级间滚雪球**。

**④ 训练目标（式 10–13）**
L = λ_gr·L_GR + λ_rank·L_rank + λ_feat·L_feat + λ_anchor·L_anchor；
- L_GR：**行为加权** SID 交叉熵（式 11），w_pv < w_click < w_atc < w_pay（按样本最高价值行为加权，偏置生成分布向高价值行为）；
- L_rank：多任务 BCE，ATC/CVR 以点击为条件、只对被点击样本回传（式 12–13）。
- **未披露**：λ 各权重具体取值。

**⑤ 推理优化：paired forward（零额外前向次数）**
φ^(l) 只依赖 KV cache 中的历史隐状态 + s^{l−1} 的 embedding 查表 → s^{l−1} 采样后即可先算好 ψ(φ^(l))，把 (s^{l−1}, ψ(φ^(l))) 作为两 token 打包进**同一次** decoder 前向（复合注意力掩码控制两者可见性）；2L 次前向 → **L 次**，与无侧信息预测的基线完全同数。

**⑥ 长序列两工程模块**
- **GCD 门控卷积去噪**（式 14–15）：掩码置零 padding → depthwise Conv1D（核宽 k、same padding、**均值滤波核初始化 w_i=1/k、可学习**）逐通道时间平滑 → 门控 α=σ(g)，x̃=(1−α)x′+α·s；M = [x; x̃]·W_M（W_M ∈ R^{2d×d_m}）——平滑视图与原始视图拼接投影，保留直达原信号路径；只在 memory 编码时执行一次，不随 beam 宽/decoder 深度重复。
- **金字塔 memory funnel**：D 层 decoder 定义单调递减记忆调度 τ=[τ₁,…,τ_D]，第 i 层 cross-attention 只读 M 的前 τ_i 个位置（序列倒序 → 前缀=最近交互）；借鉴 OneTrans 的金字塔思想但把收缩从 query 侧移到 **memory（K/V）侧**；作者明确定位为满足时延的工程适配而非独立方法创新。

**⑦ 训练配方、数据与结果**
- 配置：8 层 decoder、注意力维 512、GQA（4 query 头共享 1 KV 头）、SwiGLU-MoE 64 专家 top-2 路由；embedding 维与侧信息维均 128（侧信息 = leaf 类目/L1 类目/shop/brand 四个 32 维 embedding 拼接）；3 层 SID、每层码本 2,048；行为序列最长 1024。
- 数据：Lazada 工业日志，2.42B 样本、6.46M 用户、7.53M 物料，四行为（PV/CLK/ATC/PAY）。
- 主表：见上文 Gap 2 表（TacGR 全指标最优）。
- 消融（相对 Full 的下降）：去 CSA+crossing → CLK HR@100 −2.19%、CLK AUC −0.29%；去锚定塔 → −1.33%/GAUC −0.30%；去 UVM → −1.30%/−0.06%；两者同去 → −1.75%/−0.29%（互补性成立）；去 GCD → −2.10%/−0.16%。
- 重建质量分析：四个属性的预测-真值余弦相似度均单调 φ^(1)<φ^(2)<φ^(3)；L1 类目从冷启动到全条件**翻倍以上**（基数小、embedding 空间稀疏均匀）；brand 全程最低（top 5% brand 占 83.1% 训练样本，头重脚轻、空间稠密难分）。
- 线上：Lazada 首页 Guess-You-Like，**40 天、10% 流量**，仅把上一代生成召回 UniSGR 换成 TacGR（其余在线配置不变）：IPV +0.79%、Order **+5.55%**、GMV +3.09%（三项均显著）。

---

## C. 遗漏扫描：窗口内（2026-07 下旬 ~ 2026-10-26）首轮与顶会篇均未覆盖的新论文

### C1. SnapLGR：LLM-Based Generative Retrieval for Snapchat Content Recommendation（Snap，[arXiv 2607.28895](https://arxiv.org/abs/2607.28895)，v1 2026-07-30，最新 v4 09-25）

**①思考背景**：预训练 LLM 具备语义先验、序列建模与良好 scaling，是理想的检索引擎；但要变成生产检索器，必须学会预训练中从未见过的"物品词表"，并在严格时延与成本约束下生成合法物品标识符。
**②核心问题**：如何在短视频推荐场景把预训练 LLM 改造成可上线、可比 TIGER 式生成检索基线更优的生产级生成式召回。
**③如何解决**：三件套——多模态物品 embedding 构造 SID 并用 **Personalized PageRank 共同消费对比学习**增强（改善码本利用率、减少碰撞、注入协同信号）；**SID token 先持续预训练（CPT）接地、再在交互序列上 SFT**（先让 LLM"认识"新词表再学任务）；serving 用 **TensorRT-LLM CUDA beam search + 去中心化 worker-loop 架构**满足时延。
**④结果亮点**：已上线；live A/B 对比 TIGER 式生成检索基线：View Time +0.37%、Time Spent +0.09%、Deep Sessions +0.18%、Deep Sessions Unique User +0.11%；并固定 tokenizer 分解离线差距，量化架构/scaling/预训练各自贡献。

### C2. OGR：Once Generated, Ranked——端到端生成式整版（slate）推荐（快手系，[arXiv 2608.17613](https://arxiv.org/abs/2608.17613)，v1 2026-08-18）

**①思考背景**：slate 推荐以整版而非单品为推荐单元，需联合优化物品间互动与整版效用；现有方案"召回-排序"两段式，优化被困在已召回候选内；SID 生成式推荐提供了端到端路径，但 SID 构造缺推荐感知语义与局部协同信号，NTP 又与 slate 级目标错位。
**②核心问题**：如何让模型直接生成**有序整版**（一次生成即完成排序），同时建模全局偏好与物品间依赖。
**③如何解决**：**TUSID** 把物品专属语义与局部协同信息自适应融合进层级 SID；**list-wise 偏好规划 + 流水线式按位置 SID 解码**建模全局偏好与物品依赖；**SPA** 奖励引导的保守策略优化使生成整版对齐用户偏好而不仅模仿似然。
**④结果亮点**：离线 NDCG@5 相对提升 **+48.2%**（工业数据集）/ **+27.2%**（公开数据集）；快手在线 A/B：有效观看（Effective Views）**+1.120%**。

### C3. GRACE：实时广告生成式召回的 serving 加速引擎（[arXiv 2608.00938](https://arxiv.org/abs/2608.00938)，v1 2026-08-02 / v2 08-04）

**①思考背景**：把生成式召回产品化到高频实时广告检索有两大 serving 挑战——**资格性**（每个生成广告须满足广告主定向规则）与**算力**（严格时延与 GPU 成本下仍需宽 beam 每请求生成数千条广告）。
**②核心问题**：目录合法的约束解码只保证"存在于目录"，不保证"对该请求定向合法"；事后过滤浪费算力还可能凑不够结果；LLM 级解码器又太重。
**③如何解决**：**GTM（Generative Target Matching）**把目录合法约束解码扩展为对 SID 前缀的个性化过滤（定向规则导出的 bitmask + Bloom filter 匹配器）；解码器面向"宽 beam + 短序列"工况重设计（attention kernel、KV cache、beam search 全链路）。
**④结果亮点**：SID 级 GTM 把最终广告级定向匹配通过率从 **23.55% 提到 40.42%**；NVIDIA GH200 上对 FlashAttention-2/3 中较快者：cross-attention 延迟 **68.0×**、self-attention **23.4–25.8×**，整体解码器延迟 **11.1×**。

### C4. WeGR：统一生成式检索的行为感知 tokenization 与偏好对齐（[arXiv 2610.16979](https://arxiv.org/abs/2610.16979)，v1 2026-10-12 / v3 10-21）

**①思考背景**：工业搜索的生成式检索要利用用户行为，但 SID tokenization 难以在保持码本均衡利用的同时保留协同信号；生成 SID 与正样本不匹配时 RL 奖励极度稀疏。
**②核心问题**：如何在统一 GR 框架内同时解决"行为感知的 SID 构造"与"稀疏奖励下的偏好对齐"。
**③如何解决**：**CoCo-VAE** 在量化前把 query-item、item-item 协同信号注入连续表示，配合首层码本刷新（重激活低利用码字）与保真反坍缩正则（深 层碰撞惩罚），得到行为感知且利用均衡的 SID；**DHPO（Dynamic Horizon Policy Optimization）**逐步扩展奖励评估与策略优化视野，从 SID 前缀到完整物品做粗到细偏好学习，无需辅助奖励模型。
**④结果亮点**：大型电商搜索平台 **6 周**在线 A/B：订单 **+2.539%**、CTR +1.508%、CVR +1.381%，同时 CPU/GPU 用量降 25.5%/26.1%；支撑多搜索场景、日 PV 数千万。

### C5. Personalized LLM Recommendation with Semantic IDs：AR Beam Search vs. Multi-Token Prediction（工业流媒体团队，[arXiv 2610.15864](https://arxiv.org/abs/2610.15864)，v1 2026-10-11，NeurIPS 2026 PALM Workshop）

**①思考背景**：大规模个性化正从 embedding 表检索转向"生成 SID"的 LLM 推荐器；单模型统一检索、内容理解与语言，可凭内容冷启推荐全新条目——但"S 该怎么生成"（自回归 beam search vs 多 token 并行预测 MTP）缺乏工业级同口径对比。
**②核心问题**：在同一多任务 LLM（个性化检索 + 内容问答 + 指令跟随、大型流媒体曲库）内，公平比较单头 AR 解码与 MTP 的质量与训练/推理成本。
**③如何解决**：工业级 apples-to-apples 对比设计；为 MTP 引入**基于联合 SID 分数的辅助排序损失**；围绕该发现给出高效训练与推理栈；SID 词表扩张引起的语言能力回退用 RL 后训练修复。
**④结果亮点**：结论反直觉——**单头 beam search 在多数产品关键指标（排序、内容 QA）上稳定胜过 MTP**，且在 Qwen3-8B 规模训练美元成本约低 **2.6×**；MTP 仅在原始推理吞吐上占优；相对前一代非 SID 基线：物品冷启排序 **+77%**、内容 QA 准确率 +22%、训练快 **2.8×**。

### 一行带过（窗口内、未覆盖、方向 D 相关）

- **Beyond Semantic IDs / CQG**（[2610.12977](https://arxiv.org/abs/2610.12977)，10-09）：指出有限宽 beam 对 SID 的"可达性"约束（held-out 目标进不了有限返回列表），提出连续查询生成 CQG-Single / CQG-AR，去掉 SID 词表与离散 beam 扩展，全部物品经稠密/ANN 直接可打分；统一协议下 CQG-Single 七次全胜 TIGER 式基线且参数量仅 55.3%。
- **SPAR**（优酷，[2610.27898](https://arxiv.org/abs/2610.27898)，10-20）：PATH Decoder 从压缩长期记忆检索前缀相关证据指导每步 token 决策 + RQ-PIB 前缀均衡 SID；优酷生产数据平均 HitRate +2.66%、目标前缀保留 +5.79pp、最大子分支流量份额 −32.84%；7 天 10% 流量 A/B：人均观看时长 +3.95%、播放转化率 +3.38%。
- **TAGR**（[2608.24034](https://arxiv.org/abs/2608.24034)，v1 08-25，WSDM 2027 Main）：直播广告三级时效自适应——LSID 按直播场景/带货商品周期性刷新在投广告 SID（保持词表稳定）、IAG 多粒度意图建模 + 业务价值加权 NTP、IOPO 间歇式在线策略偏好优化；线上直播间进入率 +8.5%、购物车点击 +7.4%、收入 +16.1%。（注：abs 页显示 v2 标注 2026-11-04，晚于本窗口，引用以 v1 为准。）
- **LIGE-GR**（Meta，[2609.18148](https://arxiv.org/abs/2609.18148)，v1 09-16）：不推倒重来，把成熟 pointwise 排序系统泛化为 **listwise 生成+评估**框架，兼容既有模型/价值函数/serving；Instagram Reels 时长 +1.14%、Facebook Video +0.72%，推理资源增量适中。
- **SWIM**（[2608.25104](https://arxiv.org/abs/2608.25104)，v1 08-25）：生成式重排 G-E 框架的列表级评估器——把用户行为建模为有限视野前缀 session 级生存过程，因子化为递归生存分布 + 到位条件奖励，因果掩码 Transformer 并行估计满足工业时延。
- **MetaStrategy**（[2608.09440](https://arxiv.org/abs/2608.09440)，08-10）：不生成物品序列而生成**可执行的排序策略**（LLM policy 输出结构化策略），便于与成熟预测模型、运营规则与字段级护栏集成。

### 边界与窗口外相关（供参照，不计入本窗口）

- **FlashTrie**（[2607.10044](https://arxiv.org/abs/2607.10044)，v1 2026-07-10，略早于窗口）：GPU 化约束 beam search——整数感知 succinct trie + 协作 CUDA kernel 全设备执行；800M 关键词、beam≤1000 下 trie 搜索 <3ms、对多线程 CPU 基线最高 24×；商业搜索引擎 A/B 收入 +0.71%。
- **Vectorizing the Trie**（[2602.22647](https://arxiv.org/abs/2602.22647)，2026-02）：TPU/GPU 向量化 trie 约束解码（同属解码系统线，窗口外）。
- **RoutingTrie**（[2611.11016](https://arxiv.org/abs/2611.11016)，2026-11，窗口外索引边界）：beam 选择前先做元数据路由约束。
- **xGR**（[2512.11529](https://arxiv.org/abs/2512.11529)，2025-12）：GR serving 工况刻画（长 prompt/短输出/宽 beam），被 OneLA 引用为工作负载依据。
- 另有方向 D 邻近但偏 tokenization/重排的窗口内新作：SimHash 并行解码 SID 构造（[2610.07402](https://arxiv.org/abs/2610.07402)，10-05）、Reasoning Reranking with SIDs（[2610.22797](https://arxiv.org/abs/2610.22797)，10-16）。

---

## Sources

**Kept（采纳，均为一手来源）**
- [Multi-Decoder OneRec arXiv abs](https://arxiv.org/abs/2607.26500) ｜ [HTML 全文](https://arxiv.org/html/2607.26500v1) ｜ [Kwai26 数据管线](https://github.com/liuzhao09/Kwai26-Data-Pipeline) —— 深挖主源；式 8–19、训练配方、Table 1/4/6、A/B 细节均自原文（表格数值取自 raw HTML）。
- [OneLA arXiv abs](https://arxiv.org/abs/2609.12399) ｜ [HTML 全文](https://arxiv.org/html/2609.12399v1) —— GDN/GTR/投影重放/ancestry/融合 kernel 全部公式与 405 形状评测数字。
- [TacGR arXiv abs](https://arxiv.org/abs/2610.16132) ｜ [HTML 全文](https://arxiv.org/html/2610.16132v1) —— 五模块架构、式 1–15、消融、40 天 10% 流量 A/B。
- [SiDec arXiv abs](https://arxiv.org/abs/2607.24865) ｜ [HTML 全文](https://arxiv.org/html/2607.24865) —— 线上 Table 1（raw HTML 恢复）、Table 3 消融、§4.2 部署细节。
- [PushDualGen arXiv abs](https://arxiv.org/abs/2608.07989) ｜ [HTML 全文](https://arxiv.org/html/2608.07989v1) —— Table 2 消融全表（raw HTML 恢复）。
- 遗漏扫描五篇 + 一行级各篇：SnapLGR（[2607.28895](https://arxiv.org/abs/2607.28895)）、OGR（[2608.17613](https://arxiv.org/abs/2608.17613)）、GRACE（[2608.00938](https://arxiv.org/abs/2608.00938)）、WeGR（[2610.16979](https://arxiv.org/abs/2610.16979)）、AR-BS vs MTP（[2610.15864](https://arxiv.org/abs/2610.15864)）、CQG（[2610.12977](https://arxiv.org/abs/2610.12977)）、SPAR（[2610.27898](https://arxiv.org/abs/2610.27898)）、TAGR（[2608.24034](https://arxiv.org/abs/2608.24034)）、LIGE-GR（[2609.18148](https://arxiv.org/abs/2609.18148)）、SWIM（[2608.25104](https://arxiv.org/abs/2608.25104)）、MetaStrategy（[2608.09440](https://arxiv.org/abs/2608.09440)）—— 均已逐一抓取 abs 页核验提交日期（精确到日）与摘要。

**Dropped（排除）**
- arxiv.science / searcharxiv.org / alphaxiv 等镜像聚合站（arXiv 官方 abs 页可达，无需二手镜像）。
- dl.acm.org 直链（SiDec DOI 10.1145/3773078.3831900 仍 403）。
- LinkedIn / 博客类转述（首轮已排除，本轮未再使用；SiDec 线上数字已由 arXiv HTML 原文直接确认，不再依赖 RecArch 二手交叉验证）。

---

## 剩余 Gaps / 未能确认

1. **SiDec A/B 实验设计参数（未披露）**：流量比例、实验时长、p 值/置信区间在 arXiv 版中均未给出（仅定性"highly statistically significant"）；ACM DL 正式版 403。若需精确引用实验设计，需机构订阅访问正式版 PDF。
2. **三篇深挖论文的部分实现超参（未披露）**：MD-OneRec 的优化器/学习率/warmup；TacGR 的 λ_gr/λ_rank/λ_feat/λ_anchor 具体取值、memory funnel 的 τ 调度具体取值、GCD 核宽 k；OneLA 的 kernel tile 尺寸（B_V）与 SM 占用配置。以上在已抓取正文（含附录可读部分）中均未出现，标注定为"论文未披露"，未做任何猜测。
3. **MD-OneRec Table 1 的 HSTU/TIGER 行、TacGR Table 1 的 TIGER/OneRec-V2/UniRec(CoA) 行**：raw HTML 中仅部分单元格恢复（列位未能完全确认），本简报未引用这些不完整数值；如需完整表格建议直接下载官方 PDF。
4. **OneLA / GRACE / SWIM 等系统类论文无线上 A/B**（GRACE 有算子级与通过率指标、SWIM 称"工业时延约束下有效"但未见公开 A/B 数字）；生产级端到端 serving 指标仍以 RecoGEM（2611.12296，窗口边界外）为唯一公开样本。
5. **遗漏扫描覆盖度**：本轮仍为多引擎检索式抽样（arXiv cs.IR/cs.LG 逐日 listing 未扫）；窗口内可能仍有零星遗漏，尤其 10 月中下旬 cs.LG 侧（如 FlashTrie 同类的系统论文）。建议对 arXiv listing 2026-08、2026-10 做逐周补扫。
