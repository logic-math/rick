# Research: 生成式推荐系统近 3 个月新论文（2026-07 下旬 ~ 2026-10-26）

## Summary
过去 3 个月生成式推荐研究高度活跃：① 语义 ID / 生成式检索（TIGER 系）转向 token 效率与结构对齐（Pinterest PinGER、SAHL、RecSys 2026 多篇 tokenization 论文）；② 扩散/流式生成推荐明显升温（DPE-Rec、FairDiff、DARTS）；③ 工业界端到端生成推荐密集落地并公开系统细节（快手 OneRec 系、腾讯 TGR 系、美团 MTGenRec、YouTube SiDec、Lazada UniSGR-V2）；④ RecSys 2026（9 月）与 KDD 2026（8 月）均设生成式推荐专场。本简报按子方向整理约 20 篇窗口内论文（均标注确切日期）+ 顶会动态 + 窗口边界刚出现的稿件。

## Findings

### A. 语义 ID / 生成式检索（TIGER 系后续）
1. **PinGER: Token-Efficient Generative Retrieval with Semantic IDs**（Pinterest）— arXiv 2610.33435 — **2026-10-24** — 生产部署的生成式检索：训练时 SID+稠密 token 交错并配非对称注意力掩码，serving 时每历史物品只需 1 个稠密 token 但保留 SID 输出与监督；Related Pins 线上新 Pin 收藏 +3.60%、整体 saves +0.62%，端到端时延 -0.7% 且省掉 ANN 索引。[link](https://arxiv.org/abs/2610.33435)
2. **SAHL: Structure-Aligned History Learning with Subspace-Structured Semantic IDs** — arXiv 2610.25093 — **2026-10-18** — 语义残差平衡量化（SRBQ）构建 token–子空间–物品层级，并让历史重建遵循量化器结构；4 个 Amazon 类目 NDCG@10 较 RPG +18.6~30.0%。[link](https://arxiv.org/abs/2610.25093)
3. **Topology-Aware Tokenization for Generative Recommendation** — arXiv 2607.18600 — 2026-07 中下旬（RecSys 2026 接收）— 指出物品 tokenization 的"拓扑失真"（预训练语义空间的邻接关系被量化破坏）并提出保持拓扑的 token 化方法。[link](https://arxiv.org/abs/2607.18600)
4. **EGR: Embedding-Native Generative Retrieval with a Shared LLM** — arXiv 2607.23038 — 2026-07 下旬（RecSys 2026 接收）— 免量化的 embedding-native 生成式检索，绕开 SID 离散化瓶颈，面向大规模推荐/广告。[link](https://arxiv.org/abs/2607.23038)
5. **CoFiRec: Coarse-to-Fine Tokenization for Generative Recommendation**（Amazon）— RecSys 2026（DOI 10.1145/3773078.3831740，pp.115-124）— **2026-09-27** — 粗到细的层级 tokenization，对齐用户"从类目到具体物品"的兴趣收敛过程。[link](https://dl.acm.org/doi/10.1145/3773078.3831740)

### B. 扩散 / 流式生成推荐
6. **DPE-Rec: Diversity-Preserving and Efficient Diffusion Language Model Recommendation for Bundle Construction** — arXiv 2610.34700 — **2026-10-26** — 首批把扩散 LLM 用于（无序）bundle 补全的工作：semantic-fork-aware GRPO 保多样性 + 伪轨迹蒸馏解锁多 token 解码；平均性能 +63%，快版推理 25.6× 加速。[link](https://arxiv.org/abs/2610.34700)
7. **FairDiff: Mitigating the Self-Reinforcing Matthew Effect in Diffusion Recommender Models**（清华/华为诺亚）— arXiv 2609.36671 — **2026-09-29**（v2 10-19）— 揭示扩散推荐器因"先验失配"自我强化流行度偏差；用推理期流行度条件引导（PCG）+ 一步最优传输语义校准加以抑制。[link](https://arxiv.org/abs/2609.36671)
8. **Adaptive Item-based Collaborative Structures via Noise Rescheduling in Diffusion for Generative Recommendation** — arXiv 2608.23400 — **2026-08-24** — 离散扩散（DDM）推荐中通过噪声重调度显式注入 item-based 协同过滤结构。[link](https://arxiv.org/abs/2608.23400)
9. （期刊）**ScoreRec: Reallocating Probability Mass via Score-based Diffusion Modeling** — ACM TOIS 44(8) — **2026-10-21** — 把推荐重构为连续时间 SDE score 生成过程，纠正 DDPM 模式坍缩到热门物品的质量分配。[link](https://dlnext.acm.org/doi/10.1145/3831688)

### C. 端到端 / 自回归工业生成推荐与规模化（HSTU/LRM 系）
10. **TGR: Advancing Industrial Recommendation from Generative-Paradigm Ranking toward Unified Generation and Reasoning**（腾讯 TGR 团队）— arXiv 2609.00986 — **2026-09-01** — 工业级联架构（召回/预排/精排/重排）向"统一生成 + 推理"范式的系统性演进报告。[link](https://arxiv.org/abs/2609.00986)
11. **LoopGR: Parameter-Efficient Scaling for Generative Recommendation via Recurrent Depth**（腾讯，TGR 系列）— arXiv 2610.34157 — **2026-10-25** — 层级循环深度（共享解码器 + 轻量上下文更新器）+ 循环轨迹自蒸馏，把"深度"与"参数量"解耦：同等性能下模型最多小 52%。[link](https://arxiv.org/abs/2610.34157)
12. **MTGenRec**（美团 + 武汉大学）— KDD 2026（论文集 2026-08-08）— HSTU+MMoE 生成推荐模型的生产训练系统：动态哈希 embedding 表 + 变长序列均衡分桶；100+ A100 上比 TorchRec 快 1.6–2.4×，外卖 A/B 订单 +1.22%。[link](https://jdding.github.io/recarch/en/systems/meituan-mtgenrec-2025/)
13. **Kunlun: Establishing Scaling Laws for Massive-Scale Recommendation Systems through Unified Architecture Design** — KDD 2026（论文集 2026-08-08；arXiv 2602.10016）— 统一架构设计下的大规模推荐 scaling law 系统研究（HSTU/LRM 谱系在 KDD 的集中呈现）。[link](https://dl.acm.org/doi/10.1145/3770855.3818438)
14. **Deco: Separating Thinking from Spelling for Efficient Generative Recommendation** — arXiv 2610.32980 — **2026-10-23** — 把"读历史"（压缩成单个 next-item concept）与"写 SID"（轻量 KV 缓存解码器跑全部 beam search）分离；50ms p95 下吞吐 3.2–8.6×。[link](https://arxiv.org/abs/2610.32980)

### D. 生成式召回与排序（多目标 / 解码 / 工业落地）
15. **Multi-Decoder OneRec: Controllable Generative Retrieval for Multi-Objective Industrial Recommendation**（快手）— arXiv 2607.26500 — **2026-07-29** — 共享用户上下文 + General Decoder，每目标挂 LoRA 专家，多解码器受限 beam search 控制配额；发布 13.1 亿条记录的 Kwai26 基准；A/B 使用时长 +0.37%、新内容冷启 +2.09%。[link](https://arxiv.org/abs/2607.26500)
16. **OneLA: Scaling Linear-Attention Decoding to Large Beams in Generative Recommendation**（港大+快手）— arXiv 2609.12399 — **2026-09-11** — 面向大 beam 解码的线性注意力服务框架（共享 prompt 状态 + 追加式分歧记录 + 融合 GPU kernel），端到端解码 1.54–2.46× 加速。[link](https://arxiv.org/abs/2609.12399)
17. **UniSGR-V2（TacGR）: Stepwise Target-Side Feature Anticipation and Crossing**（Lazada 电商场景）— arXiv 2610.16132 — **2026-10-12** — 在解码每级 SID 前"预期"一个连续目标侧特征向量并与语义前缀/行为序列交叉，填补目标侧特征真空；在线 A/B：IPV +0.79%、订单 +5.55%、GMV +3.09%。[link](https://arxiv.org/abs/2610.16132)
18. **Tokens are All You Need: Dual-purpose Semantic IDs（SiDec）**（Google/YouTube）— RecSys 2026（DOI 10.1145/3773078.3831900）— **2026-09-25** — 双语义 ID 把稠密内容 embedding 压成 token 并在模型内即时重建，突破 embedding 表"I/O 记忆墙"；YouTube watchpage 满意度 +0.80%。[link](https://doi.org/10.1145/3773078.3831900)
19. **PushDualGen: Enabling LLMs to Generate Semantic IDs with Interpretable Copy for Industrial Push Recommendation** — arXiv 2608.07989 — 2026-08 上旬 — 先生成 SID、再产出可跳过的可解释文案，替代"先 CoT 后 SID"的高成本方案，已工业部署。[link](https://arxiv.org/html/2608.07989v1)

### E. LLM-based 推荐（推理 / 蒸馏 / 语义融合）
20. **SelfDR: Self-Distillation from Reasoning for LLM-Based Recommendation** — arXiv 2609.03313 — **2026-09-03** — 把推理（reasoning）能力自蒸馏进 LLM 推荐主干，推理期免显式思维链开销。[link](https://arxiv.org/abs/2609.03313)
21. **Learning Better Reasoning for Generative Recommendation with Semantic IDs** — arXiv 2609.29973 — **2026-09-24** — 针对 SID 生成式推荐的推理质量改进（训练侧）。[link](https://arxiv.org/abs/2609.29973)
22. **IMFuse: Instance-Aware Multi-Layer Fusion for LLM-Enhanced Sequential Recommendation** — arXiv 2607.27002 — 2026-07 下旬 — 发现深层 LLM embedding 谱坍缩、中间层语义互补，用实例感知 MoE 融合全部层，即插即用平均 +6.72%。[link](https://arxiv.org/abs/2607.27002)
23. **SARA: Scaling Articulated Rationales for MLLM-based Recommendation**（快手）— arXiv 2609.17639 — 2026-09 中旬 — 把稀疏、噪声大的用户自述"喜欢/不喜欢理由"规模化用于多模态 LLM 推荐训练。[link](https://arxiv.org/abs/2609.17639)

### F. 顶会动态（近 3 个月）
- **RecSys 2026**（第 20 届，2026-09，论文集 9 月下旬出版）：设专场 "Generative Recommendation: Retrieval, Ranking & Content Generation"。窗口内亮点：CoFiRec（Amazon，09-27）、SiDec（YouTube，09-25）、EGR、Topology-Aware Tokenization、Addressing Cross-Stage Decoupling of Semantic and Collaborative Signals in GR（快手系）、Empowering CDSR with Hybrid Tokenization and Serial-Parallel Decoding、PROMISE（快手，过程奖励模型解锁 GR 测试时 scaling，poster）、Netflix 工业级生成推荐（offset scaling law + 多 token 预测，2026-08-06 官宣接收）。[accepted list](https://recsys.acm.org/recsys26/contributions/) | [program](https://recsys.acm.org/recsys26/recsys-26-program/)
- **KDD 2026**（2026-08-09~13 济州；research track 录取率 21%）：设 "Generative Recommendation & Reranking" 专场。代表：Uncertainty-aware GR；Unifying Behavior Modeling and Semantic Generation（腾讯）；ReSOT（最优传输重平衡语义 ID，阿里系）；Breaking the Likelihood Trap（图结构一致生成式重排）；Stochastic Primal-Dual Decoding for Multiobjective GR（Spotify）；A Unified Model for Personalization: Language-Steerable GR/Search/User Understanding（Spotify）；Kunlun；MTGenRec（美团）；Think2Go（POI+LLM 推理，oral）；小红书搜索 RL 生成式排序；Stop Treating Collisions Equally: Qualification-Aware Semantic ID（工业规模）；DiKGRec（扩散+知识图谱）；On the Memorization and Generalization of GR。[KDD 亮点汇总](https://www.themoonlight.io/en/explore/kdd2026)
- **SIGIR 2026**（2026-07-20~24 墨尔本，恰在窗口起点）：Differentiable Semantic ID for GR；One-Pass Decoding with WFST-Constrained A* Search；RIGER（推理驱动意图注入，解决意图冷启）；Beyond Item IDs（YouTube 短视频语义原生长序列建模）；MVIGER；Lost in Decoding?（生成检索 look-ahead 先验的复现与压力测试）；GenRec（偏好导向大规模生成框架）；Diffusion/Flow-based Recommendation 专场。[program PDF](https://sigir2026.org/SIGIR2026_program.pdf)
- **NeurIPS 2026**（已接收，2026-12 开会）：Echoes in Filter Bubble（GR 流行度偏差的诊断与治理）；TRACER（SID token 重分配实现概念擦除/公平性）；Not All Slots Are Equal（非共进 Markov 桥做 bundle 构建）。[poster 1](https://neurips.cc/virtual/2026/poster/148459) | [poster 2](https://neurips.cc/virtual/2026/poster/154740) | [poster 3](https://neurips.cc/virtual/2026/poster/154810)
- **WWW 2026**：会议在 2026-04（窗口外），近 3 个月无新论文；最近的代表作：PLUM（Google，预训练 LLM 工业化生成推荐）、Token-level Collaborative Alignment for LLM-based GR、Diffusion GR with Continuous Tokens、ThinkRec、Meituan DOS 双流正交语义 ID（short）。[WWW26 short papers](https://www2026.thewebconf.org/accepted/short-papers.html)
- **Meta 专项说明**：窗口内未见全新独立论文；该系（HSTU/GEAR 谱系）最近代表作为 Kunlun（KDD 2026，见上）与 ICLR 2026 的 Massive Memorization with Hundreds of Trillions of Parameters for Sequential Transducer Generative Recommenders（窗口外）。

### G. 补充：检索索引中日期紧邻/略晚于窗口（2026-10-27 ~ 11 月）的稿件（未计入主列表，建议复核日期）
- **REGER: Modeling inter-item Relations with Semantic IDs** — arXiv 2610.37584 — 标注 2026-10-27 — 生成式语义检索预训练继承物品间关系 + 前缀树偏好分解理论。[link](https://arxiv.org/abs/2610.37584)
- **LESSRec: Level-Specific History Modeling** — arXiv 2610.37286 — 标注 2026-10-27 — 目标 token 只看同语义层级历史 token。[link](https://arxiv.org/abs/2610.37286)
- **DARTS: When Diffusion Recommenders Stop Sampling** — arXiv 2610.36954 — 标注 2026-10-27 — 扩散推荐器采样坍缩诊断 + 三阶段潜空间扩散。[link](https://arxiv.org/abs/2610.36954)
- **RecoGEM**（快手）— arXiv 2611.12296 — 生产 GR serving 系统：runtime-engine 解耦，部署 9 个月、10+ 模型家族，单实例 >1000 QPS、P99 <35ms、吞吐 2.88×。[link](https://arxiv.org/abs/2611.12296)
- **ZeroGR**（快手+港大）— arXiv 2611.04715 — filtering-first 受限解码，管线加速最高 9.9×。[link](https://arxiv.org/html/2611.04715v1)
- **ReSID** — arXiv 2611.16404 — 长尾生成式检索（并行 SID + 部分 SID 匹配），已部署主流量。[link](https://arxiv.org/html/2611.16404v1)

## Sources
**Kept:**
- arXiv 各论文 abs 页（2607.26500 / 2609.12399 / 2609.36671 / 2610.16132 / 2610.32980 等）— 已逐一抓取核验确切提交日期（精确到日）
- RecSys 2026 官网 accepted list 与 program — 一手会议接收信息
- ACM DL（KDD '26 / RecSys '26 / SIGIR '26 / TOIS）— 顶会论文集一手出处与出版日期
- NeurIPS 2026 virtual poster 页 — 已接收论文一手信息
- RecArch（jdding.github.io）— 工业系统论文（MTGenRec、SiDec）的结构化事实核对
- Kuaishou-OneRec/OpenOneRec GitHub、ACL Anthology（ONEREC-THINK）— 工业界一手资料
**Dropped:**
- notes.swasilewski.com（HSTU/LRM 概念笔记）— 个人笔记，仅用于概念背景，不作为论文出处
- bytebytego / aboutamazon / ZenML / MarkTechPost 等博客 — 营销向二手内容，事实可由一手来源覆盖
- LinkedIn 帖（Netflix RecSys 2026）— 仅用于确认接收事实，不作为内容来源

## Gaps
1. **逐日 arXiv 扫描未做**：本调研基于多引擎检索式抽样，2026-10 中旬 cs.IR 仍可能有零星遗漏；建议对 arXiv listing（cs.IR, 10 月）做逐周补扫。
2. **日期矛盾提示**：G 节稿件在检索索引中的标注日期（10-27 ~ 11 月）晚于"今天 2026-10-26"，属索引边界现象，已单列并建议以 arXiv 页面为准复核。
3. **部分论文仅有月份级日期**：Topology-Aware（2607.18600）、EGR（2607.23038）、IMFuse（2607.27002）、PushDualGen（2608.07989）、SARA（2609.17639）、Think2Go（2607.28997）确切"日"未核（仅 arXiv ID 月份推断），如需精确日期可逐一抓取 abs 页。
4. **RecSys/KDD 接收清单为摘要级汇总**：未逐一展开全部相关论文（如 RecSys 2026 cross-stage decoupling、KDD 2026 Xiaohongshu RL 排序等仅一句话概括），可按链接下钻。
5. **WWW 2026 与 Meta 在窗口内无新论文**（已按要求注明并给出最近代表作）。
