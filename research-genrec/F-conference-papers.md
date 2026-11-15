# 方向 F｜生成式推荐系统顶会专篇：RecSys / KDD / SIGIR / NeurIPS 2026（未被 A–E 覆盖论文）

> 调研日期：2026-10-26。范围：任务清单中 21 篇论文（CoFiRec、SiDec、EGR、Topology-Aware Tokenization、Kunlun、MTGenRec 已由方向 A–E 覆盖，按约跳过）。选取 12 篇最重要的做四段深度调研（每篇①思考背景②核心问题③如何解决④结果亮点），其余 9 篇一行带过。所有事实均来自 arXiv 原文/ACM DL 摘要/官方仓库/官方博客，未编造细节；受付费墙限制未获取全文处已在 Gaps 中注明。

## Summary

2026 年四会上未被 A–E 覆盖的生成式推荐（GR）论文呈现四条清晰主线：**①工业规模化与生产约束**（Netflix offset scaling law、YouTube 超长序列、快手 PRM 测试时扩展、Spotify 统一模型）；**②解码层成为新战场**（过程奖励引导束搜索、原始-对偶多目标解码、WFST+A* 单遍解码、图结构重排）；**③语义 ID（SID）学习精细化**（协同信号注入、最优传输重平衡、碰撞分级治理、可微化）；**④科学理解与偏差治理**（记忆 vs 泛化、流行度偏差机理、概念遗忘、不确定性）。两处信息修正：ReSOT 实为浙大×OPPO（非阿里系）；Beyond Item IDs 作者来自 Google（YouTube）。21 篇全部覆盖：12 篇深度 + 9 篇一行。

---

## 一、深度调研（12 篇）

### A. RecSys 2026

#### 1. Towards Generalizable and Efficient Large-Scale Generative Recommenders（Netflix Research）

**①思考背景**：Netflix 发现生成式推荐的预训练收益不会自动迁移到生产任务：任务天花板、重训成本、服务延迟、物品新鲜度都在侵蚀收益。把 backbone 从 2M 扩到 1B 参数（不含 embedding/解码层）后，观察到任务依赖的 scaling 行为——有的任务已接近经验天花板，有的仍在持续受益，"该不该继续加参数"成了必须回答的问题。
**②核心问题**：如何用 scaling 拟合诊断"再加参数值不值"，并同步解决每轮约 2 万亿行为 token 的高频重训效率、缓存服务导致 next-token 标签过时、新内容冷启动三大生产约束。难点：传统 scaling law 只给单调外推，回答不了"任务天花板在哪"。
**③如何解决**：Offset scaling law 拟合 P(N)=P₀−(N/N₀)^(−a)，以饱和水平 P₀ 做任务级天花板诊断（三个任务 RMSE 均优于 log-linear 拟合）；带指数时间衰减（半衰期 1 小时）加权的多 token 预测（MTP）对齐缓存服务延迟语义；1% 负采样 softmax＋4096→512 投影解码头（估算 FLOPs/token 降约 35.5×）压低重训成本；语义 item 塔（KG 消息传递＋LLM2Vec＋编辑标注）按实测线上冷启动率对协同 embedding 做掩码，新内容仅靠语义表示打分。
**④结果亮点**：百万用户、一周生产影子评估中，1B backbone 在全部任务上 MRR 超过 2M 基线；48 小时缓存场景下最优 5-token MTP 带来相对 MRR +22.1%/+27.8%/+27.9%，但 p95<1s 的在线服务中仅短视野任务受益（+19.6%）——MTP 需按任务谨慎启用。
来源：[arXiv:2605.23312](https://arxiv.org/abs/2605.23312)（RecSys '26，DOI [10.1145/3773078.3831905](https://doi.org/10.1145/3773078.3831905)）

#### 2. PROMISE: Process Reward Models Unlock Test-Time Scaling Laws in Generative Recommendations（快手，poster）

**①思考背景**：GR 把推荐重构为层级语义 ID 的序列生成，但错误集中在早期高层 token：开头方向一旦出错，整条生成轨迹被不可逆拖入无关语义子空间——作者命名为 Semantic Drift（语义漂移）。常规束搜索只看局部 token 概率，对这种漂移毫无察觉。
**②核心问题**：如何把 LLM 推理中"过程奖励模型（PRM）逐步验证"的思想移植到推荐生成，在解码时评估中间步骤质量并剪枝。难点：推荐 token 没有显式"步骤正确性"标注，且线上延迟预算严格。
**③如何解决**：训练轻量 PRM 稠密地评估中间生成步骤质量，配合 PRM 引导的束搜索，用逐步反馈动态剪除错误分支。本质区别是把验证信号从"最终序列是否正确"提前到"每个生成步骤"，以推理算力换精度，首次在推荐系统上解锁测试时扩展律。
**④结果亮点**：快手与快手极速版大规模在线 A/B（约 2000 万用户）验证显著提升精度并缓解语义漂移：总 App 使用时长 +0.13%、人均使用时长 +0.16%、总观看时长 +0.40%、单次观看时长 +0.30%；小模型增加推理算力可追平甚至超越大模型。
来源：[arXiv:2601.04674](https://arxiv.org/abs/2601.04674)（RecSys '26 poster，DOI 10.1145/3773078.3831741）

#### 3. Addressing Cross-Stage Decoupling of Semantic and Collaborative Signals（SCRec，快手）

**①思考背景**：现有 GR 的两阶段流水线互相"看不见"对方：分词阶段被文本语义主导，协同信号与交互相似性注入有限，码本分配与下游生成错位；生成阶段又基于交互数据重新 embed 码序列，丢掉了原始语义信息。这种跨阶段信息解耦同时伤害语义连贯性与推荐精度。
**②核心问题**：如何在不引入额外对齐任务、只付出最小训练/推理开销的前提下，让语义与协同信号在分词与生成两阶段双向流动。难点：离散码本索引空间与稠密连续语义空间存在几何失配，硬对齐会破坏一方。
**③如何解决**：SCRec 三组件构成双向补信息框架：(i) 协同增强分词——把文本化的协同信号显式注入语义 tokenization；(ii) 语义引导生成——用可学习码 embedding 在生成阶段动态校准语义先验；(iii) 流形对齐——弥合离散码本索引 embedding 空间与稠密语义空间的几何失配。
**④结果亮点**：多数据集大量实验验证了有效性、鲁棒性与可泛化性，额外训练与推理成本极小；作为通用框架可套用在已有 GR 管线上。
来源：[arXiv:2609.13678](https://arxiv.org/abs/2609.13678)（RecSys '26 主会，DOI 10.1145/3773078.3831736）

### B. SIGIR 2026

#### 4. Beyond Item IDs: Scaling Short-Form-Video Recommendation via Semantic-Native Long Sequence Modeling（Google/YouTube）

**①思考背景**：短视频推荐需要捕捉超长观看历史，但序列长度被两个瓶颈卡死：正交 Video ID 语义稀疏、embedding 表随语料基数膨胀且表达不了内容间关系；Transformer 自注意力二次复杂度在工业延迟/资源约束下限制最大序列长度。
**②核心问题**：如何在十亿用户级生产系统中同时突破表示瓶颈与算力瓶颈、以可负担成本支持超长行为序列。此前工作要么只改物品表示、要么截断序列，缺少生产可部署的联合方案。
**③如何解决**：生产已部署框架两步走：用深度截断的粗粒度语义 ID（内容原生 SID）替代 Video ID，使 embedding 表规模与语料基数解耦，共享语义前缀天然泛化到冷启动内容；再引入 Global-Aware Compression Transformer，以非参数时间折叠＋统一全局 query 集成压缩序列，绕开标准自注意力的内存与算力瓶颈。
**④结果亮点**：自建算力设施上离线 profiling 显示峰值内存降一个数量级、计算开销大幅下降，从而在生产成本内支持更长序列；大规模在线 A/B 带来满意度互动与满意内容消费的显著提升。
来源：[arXiv:2606.07546](https://arxiv.org/abs/2606.07546)（SIGIR '26，DOI 10.1145/3805712.3808503）

### C. KDD 2026

#### 5. Uncertainty-aware Generative Recommendation（UGR，中科大）

**①思考背景**：GR 的偏好优化普遍依赖二元"结果对错"信号，作者指出系统性缺陷"不确定性盲视"（uncertainty blindness）：模型自身生成置信度、样本学习难度差异、显式置信表达能力全被忽略，直接导致训练动态不稳定、决策风险不可量化。
**②核心问题**：如何把不确定性作为一等公民信号纳入生成式推荐的优化。难点：此前方法既没有置信度进入 reward 的通道，也没有机制区分"自信的错误"与"正常的难样本"。
**③如何解决**：UGR 三机制协同：(1) 不确定性加权奖励，重罚自信错误；(2) 难度感知优化动力学，防止过早收敛；(3) 显式置信对齐，赋予模型置信表达能力。本质区别：reward 与优化梯度都按不确定性自适应调节，而非对所有错误一视同仁。
**④结果亮点**：推荐性能更优且训练根本性稳定（避免了标准方法常见的性能退化）；学到的置信度可直接支撑下游风险感知应用（如按置信度路由/兜底）。
来源：[arXiv:2602.11719](https://arxiv.org/abs/2602.11719)、[GitHub](https://github.com/cxfann/UGR)（KDD '26，DOI 10.1145/3770855.3817975）

#### 6. Breaking the Likelihood Trap: Consistent Generative Recommendation with Graph-structured Model（CONGRATS，北大×快手）

**①思考背景**：生成式重排把重排建模为整体序列生成、隐式建模物品依赖，但作者发现"似然陷阱"（likelihood trap）：模型偏爱的高似然序列往往重复、被人类感知为低质量——最大化似然与用户感知质量在重排场景系统性错位，限制用户参与度。
**②核心问题**：如何让生成式重排跳出"高似然＝高质量"的错位，在扩大多样性的同时保持与用户偏好一致。难点：训练目标（似然）与业务目标（用户反馈）不一致，自由生成与约束之间难以两全。
**③如何解决**：CONGRATS 提出图结构模型，让生成沿图上多条路径探索——既扩大解码空间促进多样性，又通过图转移显式建模物品依赖提升精度；再设计带 evaluator 的一致性可微训练方法，让模型直接从用户偏好（而非仅似然）学习。
**④结果亮点**：离线实验全面优于 SOTA 重排方法；已在 3 亿+ DAU 的快手大规模验证，推荐质量与多样性同时显著提升。
来源：[arXiv:2510.10127](https://arxiv.org/abs/2510.10127)（KDD '26，DOI 10.1145/3770855.3818326）

#### 7. Stochastic Primal-Dual Decoding for Multiobjective Generative Recommender Systems（SPDD，Spotify）

**①思考背景**：实际推荐常要生成满足多目标（相关性之外的属性约束、公平性约束）的有序 slate；现有做法要么用为非生成式设计的事后处理（忽略生成过程的序列依赖），要么把辅助目标训进模型——大规模系统里目标一变就要重训，不现实。
**②核心问题**：如何在不动、不重训底层生成式推荐模型的前提下，在自回归解码时动态平衡相关性与多个带约束的辅助目标。
**③如何解决**：把解码建模为在线约束优化问题：逐个生成物品时依据各目标的剩余约束余量动态调节权衡，用随机原始-对偶近似方案在生成过程中在线求解。纯推理时的轻量解码层，附约束违反与 regret 的理论保证——与"改训练目标"路线的本质区别是零重训、即插即用。
**④结果亮点**：离线实验与大规模线上 A/B（真实推荐系统、playlist ranking 场景）一致改善多目标权衡：相比固定权重生成多获得 +1.8pp 辅助目标收益，用户满意度零损失。
来源：[arXiv:2607.19357](https://arxiv.org/abs/2607.19357)（KDD '26，DOI 10.1145/3770855.3817806；[Spotify 官方博客](https://research.atspotify.com/2026/8/balancing-multiple-objectives-in-generative-recommendations-with-adaptive-decoding)）

#### 8. A Unified Model for Personalization: Language-Steerable Generative Recommendation, Search, and User Understanding（NEO，Spotify）

**①思考背景**：LLM 日益用于推荐/检索/推理，但单一端到端模型要在千万级异构目录上同时支持三者极难：必须生成指向真实物品的无歧义引用、处理多种实体类型、满足严格延迟与可靠性——纯文本生成做不到，工具增强方案又引入编排复杂度且无法端到端优化。
**②核心问题**：如何把预训练 decoder-only LLM 改造成免工具、目录锚定的统一生成器，让自然语言指令能控制任务、目标实体类型与输出格式（ID、文本或混合）。
**③如何解决**：NEO 用语义 ID 表示物品、把 SID 视作一种独立"模态"（借鉴多模态对齐经验做分阶段对齐＋指令微调），训练单一模型在共享序列中交错生成自然语言与类型化物品标识；约束解码保证生成的物品 ID 始终目录合法且不限制自由文本输出，作者称之为 language-steerability（语言可操控性）。
**④结果亮点**：在 1000 万+ 物品、多媒介类型的真实目录上，推荐/搜索/用户理解三类任务全面超过强任务专用基线，并呈现正跨任务迁移，给出把大规模发现能力合并进单一语言可操控生成模型的可行路径。
来源：[Spotify Research](https://research.atspotify.com/publications/unified-model-personalization-language-steerable-generative-recommendation-search-user-understanding)（KDD '26，DOI 10.1145/3770855.3817955）

#### 9. On the Memorization and Generalization of Generative Recommendation（MemGen-GR，UCSD McAuley 组等）

**①思考背景**："GR 优于传统 ID 序列推荐是因为泛化更强"是被广泛接受却从未被系统验证的假设；此前只有整体指标对比，无法区分记忆与泛化各自贡献了什么。
**②核心问题**：如何把每个测试实例按"答对所需能力"细粒度归类——记忆（复用训练中见过的物品转移模式）vs 泛化（组合已知模式预测未见转移）——并解释两种范式为何各有所长。此前没有这样的评估框架。
**③如何解决**：提出细粒度评估框架逐实例分类，再把分析从物品层下沉到 token 层：发现 GR 的"物品级泛化"很大程度上可还原为"token 级记忆"；基于置信度的记忆感知指标（memorization-aware indicator）按实例自适应组合 GR 与传统 ID 模型。
**④结果亮点**：7 个开源数据集上证实：GR 在需泛化的实例上更强、ID 模型在需记忆的实例上更强、二者互补；自适应 ensemble 提升整体性能。发布实例级标注与模型 checkpoint（HuggingFace），可复用性强。
来源：[arXiv:2603.19809](https://arxiv.org/abs/2603.19809)、[GitHub](https://github.com/Jamesding000/MemGen-GR)（KDD '26，DOI 10.1145/3770855.3818148）

#### 10. ReSOT: Re-balance Semantic ID with Optimal Transport for Generative Recommendation（浙大×OPPO）

**①思考背景**：物品 tokenization（RQ-VAE/残差 K-means）大多以逐点或启发式方式优化，作者指出这会造成"关系结构失真"（Relational Structure Distortion）：物品-物品关系在量化后的码空间被扭曲，而生成恰恰依赖码前缀承载的结构信息；同时码本使用不均衡、碰撞缺乏原则性处理。
**②核心问题**：如何在学习语义 ID 时同时保持物品关系结构、重平衡码本分布并感知碰撞。难点：逐点重建损失天然只看单物品、贪心分配天然失衡，此前方法（TIGER/LETTER 等）各解决一个维度，缺统一原则。
**③如何解决**：ReSOT 把 tokenization 变成全局结构保持的传输问题：用（Fused）Gromov-Wasserstein OT 保持物品-物品关系结构（BADMM 求解），用非平衡 OT（UOT）在最后一层学习碰撞感知、语义一致的软码分配（镜像梯度迭代），配合多样性损失；与逐点量化是本质区别。
**④结果亮点**：T5 主干＋Amazon 2018 文本/图像（LLaMA＋CLIP）embedding 设置下，在与 TIGER、LC-Rec、LETTER、MME-SID、MQL4GRec、MACRec 等 SOTA 分词基线的对比实验及 6 组消融（GWOT/UOT/软量化各组件）中验证有效；官方仓库开源端到端管线。
来源：[GitHub](https://github.com/grw-zju/ReSOT)、[ACM DL](https://dl.acm.org/doi/10.1145/3770855.3817834)（KDD '26，pp.1274-1285）

### D. NeurIPS 2026

#### 11. Echoes in Filter Bubble: Diagnosing and Curing Popularity Bias in Generative Recommenders（Ghost）

**①思考背景**：流行度偏差是推荐的老问题，但把传统去偏方法直接搬到 GR 收效甚微；GR 到底在哪一层、因何种机制放大流行度偏差，此前缺乏解释，导致治理只能"隔靴搔痒"。
**②核心问题**：从生成框架的优化目标与基于语义索引的物品 tokenization 两个核心环节，定位 GR 流行度偏差的根本成因，并据此设计 GR 原生的治理方案。
**③如何解决**：理论分析把根因归结为"token 级优化缺陷与物品 tokenization 无差别性的合流"；据此提出 Ghost 系统双管齐下：非对称反似然优化（asymmetric unlikelihood optimization，对流行/非流行方向施加不对称惩罚）＋骨架化 tokenization（skeleton-founded tokenization，从码本结构上削弱无差别高频 token 的优势）。
**④结果亮点**：3 个数据集、多个 SOTA 基线上显著缓解流行度偏差、推荐分布更公平，整体推荐效用仅有轻微下降。
来源：[arXiv:2605.16825](https://arxiv.org/abs/2605.16825)（NeurIPS 2026 [poster](https://neurips.cc/virtual/2026/poster/148459)）

#### 12. TRACER: Token ReAssignment for Concept ERasure in Generative Recommendation

**①思考背景**：隐私与安全要求推荐系统对敏感/有害概念做"遗忘"（concept unlearning）；但 GR 的 SID 是抽象标识符，常被"遗忘集"与"保留集"物品共享，直接套用 LLM 遗忘方法（抑制相关 token）会连带摧毁保留物品的生成——概念擦除与推荐效用保留发生结构性冲突。
**②核心问题**：如何在移除目标概念的同时最大限度保住推荐效用。难点：SID 无显式语义、概念与 token 的绑定是多对多共享，无法像词 token 那样直接定位删除。
**③如何解决**：TRACER 不抑制共享 SID，而是做 token 重分配（reassignment）：把概念相关物品重新指派到更利于遗忘的替代 token 上，并以相干性正则器保持保留物品间的语义一致。本质是把"删除/抑制"变成"改址"，用有意义的替代品而非破坏生成质量。
**④结果亮点**：真实数据集上有效擦除目标概念，效用保留显著优于现有 unlearning 基线：保留约 97% 原推荐效用，替代 token 语义合理（语义相似度 0.834），且速度快。
来源：[arXiv:2606.07688](https://arxiv.org/abs/2606.07688)（NeurIPS 2026 [poster](https://neurips.cc/virtual/2026/poster/154740)）

---

## 二、其余论文一行带过（9 篇）

1. **Empowering CDSR with Hybrid Tokenization and Serial-Parallel Decoding（GenCDSR，香港城大×字节，RecSys 2026）**：指出跨域 GR 两大缺口——分词忽略跨域协同关联、束搜索式低效解码阻碍实时部署；提出多塔跨域混合分词（共享+特定层级码本）与利用 SID 层级结构部分并行化的串行-并行解码，T5/Qwen 主干在 Clothing–Sports、Electronics–Phone、Book–Movie 三组跨域设置验证。[arXiv:2607.28659](https://arxiv.org/abs/2607.28659)、[GitHub](https://github.com/Applied-Machine-Learning-Lab/RecSys2026_GenCDSR)
2. **Unifying Behavior Modeling and Semantic Generation（OMG，华南理工×腾讯，KDD 2026）**：定义"语义偏好偏差"（GR 偏好语义相似物品、忽略弱语义相似但行为相关的物品），用混合行为-语义编码＋行为-语义渐进推理（迭代精炼、从行为空间渐进过渡到语义空间）统一行为建模与语义生成。[ACM DL](https://dl.acm.org/doi/10.1145/3770855.3818045)
3. **Stop Treating Collisions Equally: Qualification-Aware Semantic ID（QuaSID，快手，KDD 2026）**：指出 SID 碰撞并非一律有害，提出碰撞"资格"分级：Hamming 引导的边际排斥（按碰撞严重度缩放几何约束）＋冲突感知合法配对掩码（滤掉采样协议导致的良性重叠）＋双塔对比注入协同信号；公开数据集 top-K 排序较最强基线 +5.9%，快手电商 A/B（5% 流量）GMV-S2 +2.38%、冷启动检索成单最高 +6.42%，排斥损失即插即用。[arXiv:2603.00632](https://arxiv.org/abs/2603.00632)
4. **DiKGRec: Generative Recommender Model with Diffusion and Knowledge Graph-Based Reasoning（Bosch AI×奥斯陆大学×清华，KDD 2026）**：双架构把扩散生成与 KG 推理（刻画偏好在 KG 上的传播）结合，缓解交互稀疏、增强可解释性并借 KG 语义连接自信地推荐冷启动新物品；3 个基准上 recall@20 平均超 SOTA 10%+。[ACM DL](http://dl.acm.org/doi/10.1145/3770854.3780190)
5. **Differentiable Semantic ID for Generative Recommendation（DIGER，格拉斯哥×山大×Amazon，SIGIR 2026）**：把 SID 从"冻结的重建目标"变为可微——推荐梯度直接更新 tokenizer；用 Gumbel 噪声鼓励早期码本探索、两种不确定性衰减策略实现探索→收敛平滑过渡，破解可微化普遍伴随的 codebook collapse，多公开数据集一致提升。[arXiv:2601.19711](https://arxiv.org/abs/2601.19711)、[GitHub](https://github.com/junchen-fu/DIGER)
6. **One-Pass Decoding for GR with WFST-Constrained A* Search（OneGR，中科院计算所，SIGIR 2026 short）**：把 SID 预测形式化为单遍、结构约束解码问题：单次前向计算全局一致的位置级分数，把合法 SID 空间编码为确定性加权有限状态转换器（WFST）并用单遍加性路径代价的 A* 搜索替代 trie 前缀束搜索——只探索可达前缀、消除迭代假设扩展、保证最优预测，缓解早期次优前缀的不可逆误差传播；多基准上稳定超过强基线。[ACM DL](https://dl.acm.org/doi/10.1145/3805712.3809884)
7. **Reasoning-Grounded Intent Injection for GR（RIGER，JD.com 等，SIGIR 2026）**：面向"意图冷启动"（行为驱动的 SID 系统难以在显式信号出现前激活潜在需求），两阶段把离线 LLM 推理注入在线 GR：SID 空间层级意图对齐探索奖励＋KL 正则保留原利用行为；线上点击 +1.6%、广告主消耗亦有提升（具体值未获取到）。[ACM DL](https://dl.acm.org/doi/10.1145/3805712.3808423)
8. **Lost in Decoding? Reproducing and Stress-Testing the Look-Ahead Prior in Generative Retrieval（阿姆斯特丹大学等，SIGIR 2026 可复现赛道）**：复现 PAG（look-ahead 先验引导解码）确认其在 MS MARCO/TREC-DL 19/20 的有效性与延迟-束宽权衡；但压力测试发现规划信号对词面扰动脆弱——保意图的 typo 即可触发 plan collapse、look-ahead 奖励失效退化为无引导搜索；跨语言（mMARCO 查询×英文索引）场景下查询翻译是最有效的免重建索引缓解。[arXiv:2604.23396](https://arxiv.org/abs/2604.23396)
9. **Not All Slots Are Equal: Non-Co-Progressive Markov Bridge for Bundle Construction（NonMBB，NeurIPS 2026 poster）**：把 bundle 构造（从零构建或补全组合）建模为离散状态空间的非共进 Markov 桥：对高兼容 slot 与弱关联 slot 赋不同时间步调度（避免高兼容 slot 在相同噪声级参照无信息上下文）＋调度重加权约束最大时间步差；Spotify/POG 数据集上显著改进最优二部匹配与 embedding 空间对齐。[NeurIPS poster](https://neurips.cc/virtual/2026/poster/154810)

---

## 三、横向趋势观察

1. **工业界全面入场且带生产证据**：Netflix（scaling 诊断+MTP）、Google/YouTube（超长序列+内存压缩）、快手（PRM 测试时扩展、图结构重排、SID 碰撞治理）、Spotify（多目标解码、统一模型）均给出 A/B 或生产影子评估数字——2026 年的叙事从"GR 能不能行"转向"GR 在什么约束下值得部署"。
2. **解码层成为独立创新层**：PROMISE（过程奖励引导剪枝）、SPDD（原始-对偶在线约束求解）、OneGR（WFST+A* 单遍最优）、Congrats（图结构多路径）显示：不动模型、只改解码即可拿到可观收益，且与测试时扩展天然契合。
3. **语义 ID 学习进入"第二阶段"**：从"如何量化"（TIGER/RQ-VAE）转向协同信号注入（SCRec/QuaSID）、结构保持（ReSOT）、碰撞分级（QuaSID）、可微端到端（DIGER）——tokenization 与推荐目标的一致性成为核心矛盾。
4. **科学审视与治理并行**：MemGen-GR 拆穿"泛化神话"（物品级泛化≈token 级记忆）、Echoes 定位流行度偏差机理、TRACER 做概念遗忘、UGR 引入不确定性、Lost in Decoding 做可复现性审计——社区开始对 GR 做批判性验证。
5. **Netflix 的 offset scaling law 与 PROMISE 的测试时扩展互为镜像**：前者回答"参数还该不该加"，后者证明"推理算力也能换性能"——GR 的 scaling 讨论正从单轴（参数）走向双轴（参数×推理算力）。

---

## 四、来源清单（保留）

- [arXiv:2605.23312](https://arxiv.org/abs/2605.23312) Netflix 论文（含 [HTML 全文](https://arxiv.org/html/2605.23312v2)，方法细节主要来源）
- [arXiv:2601.04674](https://arxiv.org/abs/2601.04674) PROMISE（A/B 数字另参 [arXiv HTML](https://arxiv.org/html/2601.04674) 与 [SOTA2 基准页](https://www.sota2.com/research/sota/recommendation-on-kuaishou-lite-online-a-b-test)）
- [arXiv:2609.13678](https://arxiv.org/abs/2609.13678) SCRec；[arXiv:2606.07546](https://arxiv.org/abs/2606.07546) Beyond Item IDs；[arXiv:2602.11719](https://arxiv.org/abs/2602.11719) UGR；[arXiv:2510.10127](https://arxiv.org/abs/2510.10127) CONGRATS；[arXiv:2607.19357](https://arxiv.org/abs/2607.19357) SPDD；[arXiv:2603.19809](https://arxiv.org/abs/2603.19809) MemGen-GR；[arXiv:2605.16825](https://arxiv.org/abs/2605.16825) Ghost；[arXiv:2606.07688](https://arxiv.org/abs/2606.07688) TRACER；[arXiv:2607.28659](https://arxiv.org/abs/2607.28659) GenCDSR；[arXiv:2603.00632](https://arxiv.org/abs/2603.00632) QuaSID；[arXiv:2601.19711](https://arxiv.org/abs/2601.19711) DIGER；[arXiv:2604.23396](https://arxiv.org/abs/2604.23396) Lost in Decoding
- [Spotify Research：NEO 论文页](https://research.atspotify.com/publications/unified-model-personalization-language-steerable-generative-recommendation-search-user-understanding)、[SPDD 博客](https://research.atspotify.com/2026/8/balancing-multiple-objectives-in-generative-recommendations-with-adaptive-decoding)、[From Models to Products](https://research.atspotify.com/2026/8/from-models-to-products-llms-for-recommendation-at-spotify-scale)
- [ReSOT GitHub](https://github.com/grw-zju/ReSOT)（方法与消融细节）、[MemGen-GR GitHub](https://github.com/Jamesding000/MemGen-GR)、[UGR GitHub](https://github.com/cxfann/UGR)、[DIGER GitHub](https://github.com/junchen-fu/DIGER)、[GenCDSR GitHub](https://github.com/Applied-Machine-Learning-Lab/RecSys2026_GenCDSR)
- [ACM DL DOI](https://dl.acm.org/doi/10.1145/3770855.3818045)（OMG/腾讯）、[ACM DL DOI](http://dl.acm.org/doi/10.1145/3770854.3780190)（DiKGRec）、[ACM DL DOI](https://dl.acm.org/doi/10.1145/3805712.3809884)（OneGR）、[ACM DL DOI](https://dl.acm.org/doi/10.1145/3805712.3808423)（RIGER）——摘要级信息（正文 403）
- [NeurIPS 2026 虚拟海报页](https://neurips.cc/virtual/2026/poster/148459)：Echoes（148459）、TRACER（154740）、NonMBB（154810）
- [OpenAlex API](https://api.openalex.org/works/doi:10.1145/3805712.3809884)：补全 OneGR/DiKGRec 完整摘要（ACM 被 403 时的替代通道）

**已剔除**：catalyzex / arxivdaily / theflow / knowthepaper / scholarbro 等聚合站（SEO 内容、与 arXiv 原文重复）；dl.acm.org PDF 直链（一律 403）；dinq.me 学者页（冗余）；hypepaper / luneresearch（429/403，仅经搜索快照使用了其中摘要片段并已与 arXiv 原文交叉核对）。

---

## 五、未能确认的信息（Gaps）

1. **ACM DL 付费墙（中等影响）**：ReSOT、OMG（腾讯）、RIGER、OneGR、DiKGRec 的正式版正文不可达（HTTP 403），以上论文仅有摘要级信息；ReSOT 无 arXiv 版本，其实验具体数字（Hit@k/NDCG@k 提升幅度）未获取。
2. **RIGER 线上收益不完整（低影响）**：摘要显示"点击 +1.6%、广告主消耗提升"，但消耗提升的具体百分比在可获取片段中截断。
3. **Netflix 任务匿名化（低影响）**：论文将下游任务匿名为 Task A/B/C，无法对应具体产品场景；offset scaling law 拟合参数（P₀ 等）仅对这三个匿名任务报告。
4. **版本差异（低影响）**：MemGen-GR 的 arXiv 标题（"How Well Does GR Generalize?"）与 KDD 正式标题不同；KDD V.1/V.2 与 arXiv 版内容可能存在差异（如 UGR v1→v2 修订）。
5. **PROMISE 的 PRM 构造细节（低影响）**：PRM 训练标注来源与蒸馏方式未读全文，仅有框架级描述。

**建议后续动作**：如需 ReSOT/OMG/RIGER 的实验细节，可通过机构订阅访问 ACM DL 或联系作者索取；PROMISE、Netflix 建议精读 arXiv HTML 全文补齐方法实现细节。
