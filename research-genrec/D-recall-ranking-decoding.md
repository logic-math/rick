# 生成式推荐论文深度调研——方向 D：生成式召回与排序（多目标 / 解码 / 工业落地）

调研日期：2026-10-26。五篇论文全部成功抓取原文（arXiv abs + HTML 全文；SiDec 的 ACM DL 页面 403，改用其 arXiv 版 2607.24865 及 RecArch 摘要交叉验证）。

## Summary

2026 年下半年，生成式推荐的研究重心已从"单模型统一召回+排序"转向工业落地的三大瓶颈：多目标控制（Multi-Decoder OneRec 用 LoRA 专家+配额感知解码在快手落地）、大 beam 解码的系统效率（OneLA 针对线性注意力 beam search 提出 1.5–2.5× 加速的解码框架）、解码过程中目标侧特征缺失与可解释性（TacGR 的逐步特征预期、PushDualGen 的 SID+文案双生成）；SiDec（YouTube）则从 I/O 视角证明 SID token 可一码两用，同时充当协同身份与按需重建的内容向量。五篇均附大规模线上 A/B 或生产部署收益。

## 逐篇调研

### 1. Multi-Decoder OneRec: Controllable Generative Retrieval for Multi-Objective Industrial Recommendation（快手，arXiv 2607.26500，2026-07-29）

**①思考背景**：工业推荐长期按"目标专属召回通道 + 显式配额"构建候选池（观看时长/互动/冷启动各一条路），配额可控，但通道一多，建模、训练、服务全面碎片化；SID 生成式检索统一了建模，可单解码器又让各目标策略相互纠缠（一个目标的更新扰动其他目标，产生负迁移），且各目标条件下生成的候选扎堆在同一高概率 SID 区域，浪费固定预算。

**②要解决的核心问题**：如何让共享的生成式检索器同时满足三重耦合需求——共享建模、各目标隔离适配、固定预算下候选互补。难点在于各目标反馈异构（事件型 like/share vs 连续时长），且给每个目标复制完整解码器会重现模型碎片化，此前的 prompt/任务 token 方案只改变条件方式，核心变换仍共享，策略依旧耦合。

**③如何解决**：所有目标共享用户上下文模块和全参 General Decoder（提供通用 SID 生成先验），每个目标只挂一个隔离的轻量专家（LoRA 低秩更新 + 专属 BOS embedding + SID 词表残差 ΔE）。训练用梯度路由隔离：曝光样本 NTP 只更新共享底座；事件型专家用目标过滤样本 SFT；时长专家用用户历史归一化相对奖励做 L-GBPO 策略优化，并以 General Decoder 的 stop-gradient 分布做 KL 正则锚定。推理用 Multi-Decoder Constrained Beam Search（MD-CBS）：按显式配额与优先级逐路解码，后续路由屏蔽已被占用的 SID 前缀（屏蔽粒度可配置），General Decoder 最后回填剩余配额。

**④结果亮点**：公开 Kwai26 多目标基准（13.1 亿原始行为记录、3185 万 Item-ID、2503 万有效 SID 物料，含切分与评测协议）；同等 512 候选预算下四项 Recall@512 较单解码器 OneRec 基线相对提升 1.69%–5.62%；生产 A/B：人均使用时长 +0.37%、7 日留存用户 +0.19%、有分享设备数 +0.19%、新内容冷启动 +2.09%。

来源：[arXiv abs](https://arxiv.org/abs/2607.26500) ｜ [HTML 全文](https://arxiv.org/html/2607.26500v1)

---

### 2. OneLA: Scaling Linear-Attention Decoding to Large Beams in Generative Recommendation（港大 + 快手，arXiv 2609.12399，2026-09-11）

**①思考背景**：生成式推荐需一次产出数百个候选，必须用大 beam 解码；为吸收长用户历史，OneReason 等工业模型转向 Gated DeltaNet（GDN）等线性注意力，把历史压成固定大小递归状态。但现有服务系统（vLLM 的 FullState / ReplaySSM 路径）要么为每个活 beam 物化一份完整 d_v×d_k 状态矩阵、要么按 beam 独立重放 prompt 尾部，内存与访存流量随 beam 宽度急剧膨胀；且 beam 每步全局剪枝/重排/复制，会引发状态拷贝或重算。

**②要解决的核心问题**：如何在大宽度、动态演化的 beam 树下高效执行线性注意力解码，避免 per-beam 状态物化与搬移。此前未解决，是因为通用 LLM 服务框架假设"每条序列独占一份状态"、把每个 beam 当独立请求，没有利用 GR 工况"所有 beam 共享同一长 prompt、只在极短 SID 后缀上分叉"的结构。

**③如何解决**：关键观察是 GDN 每步只通过两个向量投影（S·k_t 与 S·q_t）消费历史状态，状态矩阵本身无需重建。OneLA 每个请求只存一份 prefill 后的共享状态 S_ctx，每步为每个新 beam 追加一条紧凑 GDN 转移记录 GTR (α, δ, k)，存储从 O(d_v·d_k) 降为 O(d_v+d_k)；用轻量 ancestry 索引追踪每条 beam 引用哪些物理槽位，beam 剪枝/重排/分叉只改引用、绝不搬动记录；解码时沿祖先链做向量级"投影重放"（两个累加器递推），配合跨 beam 复用 S_ctx 的融合 GPU kernel，全程片上执行、零状态写回。

**④结果亮点**：在 0.8B 混合注意力工业模型（R=4、W=256、prompt 1K/5K）上端到端解码加速 1.54–2.46×；算子级对 vLLM/SGLang FullState 中位加速约 40×、对 ReplaySSM 17.5×；持久递归状态容量最高降 42.5×、状态写回降 127×、DRAM 流量降约 40–77×，最大数值误差仅 4.88×10⁻⁴。

来源：[arXiv abs](https://arxiv.org/abs/2609.12399) ｜ [HTML 全文](https://arxiv.org/html/2609.12399v1)

---

### 3. UniSGR-V2（TacGR）: Stepwise Target-Side Feature Anticipation and Crossing（Lazada / 阿里国际，arXiv 2610.16132，2026-10-12）

**①思考背景**：判别式排序模型随时可查询目标物料的侧特征（类目/品牌/店铺）并做特征交叉；而 SID 生成式推荐在逐 token 解码时目标物料尚未确定、属性又被压进有限码本，形成"目标侧特征真空"：每层码字条件熵高（第一层尤甚）且错误逐层不可逆级联；排序头只能依赖解码隐状态，缺失目标侧特征交叉。已有方案要么只交互候选表示、要么 SID 完成后回查物料做事后重排（救不了生成过程）；最接近的 UniRec Chain-of-Attribute 生成离散属性 token，但错误属性不可修正、属性维度增多时解码步线性拉长。

**②要解决的核心问题**：如何在解码路径内部重建一个连续的目标侧特征表示，使其同时作为逐层生成条件并直接服务最终排序。此前无人解决，因为对连续向量做回归引入两大新风险：可学习的回归标签会与预测一起塌缩成零信息平凡解；预测噪声会经注意力污染生成路径、诱发自依赖退化。

**③如何解决**：TacGR 在每层 SID 码字解码前插入预测点：第 1 层冷启动用用户历史侧特征的均值先验，之后预测条件随解码推进逐步变富（新解码的码字即刻提升预测粒度）；预测出的连续向量 φ^(l) 作为 token 送入解码器，与语义前缀和用户行为序列做单向交叉，得到的交叉表示既做当前层生成的软条件、又直接拼接进 PLE 多任务排序头（CTR/ATC/CVR）。用语义锚定辅助塔（以真实侧信息监督）防止回归目标塌缩，用单向可见性掩码 UVM（语义 token 不可见预测 token、预测 token 互不可见）阻断误差传播；工程上以门控卷积去噪 GCD 处理千级行为序列噪声，并把 OneTrans 金字塔收缩迁移到 cross-attention 的 memory 侧压时延。

**④结果亮点**：在 24.2 亿样本、646 万用户 / 753 万物料的 Lazada 数据集上，四项检索指标（CLK/PAY 的 HR@100/HR@500）全面超过 OneRec-V1/V2、UniRec(CoA)、UniSGR（对 UniSGR 最高相对 +3.86%），排序 AUC/GAUC 亦提升（最高 +0.79%）；线上 40 天 A/B（仅替换生成召回模型）：IPV +0.79%、订单 +5.55%、GMV +3.09%（均显著）。

来源：[arXiv abs](https://arxiv.org/abs/2610.16132) ｜ [HTML 全文](https://arxiv.org/html/2610.16132v1)

---

### 4. Tokens are All You Need: Dual-purpose Semantic IDs（SiDec）（Google / YouTube，RecSys 2026，arXiv 2607.24865；DOI 10.1145/3773078.3831900）

**①思考背景**：大规模推荐受"存储墙（Memory Wall）"制约：排序/检索模型依赖海量稠密浮点 embedding 表，用户历史序列达 10⁴ 级，训练日志与在线服务都要逐条存储、传输、join 大向量，I/O 与内存带宽成为吞吐瓶颈；生成式检索虽用离散 token 替代物品 ID，但高维上下文与内容特征仍停留在低效的稠密连续格式。作者借鉴 CV 领域 VQ-VAE/VQGAN 的启示：连续高维数据不必以原生浮点形式处理也能保留预测力。

**②要解决的核心问题**：如何让一套 Semantic ID token 序列同时承担"协同身份"与"内容表示"两种职责，从而把推荐系统从浮点 I/O 中彻底解耦。此前未解决，是因为 SID 一直只被当作 ID 的替代品，没有人把它当作可在模型图内按需解码、还原内容向量的压缩码使用。

**③如何解决**：用 RQ-VAE 把预训练内容向量压成 K 个共享码本 token（存储从 d×32 bit 降至 K·log₂V bit，压缩比约 50–100×），在模型图内一码两用：(1) 协同身份流——把 SID 当层级类别特征，提供 unigram / 重叠 bigram / 嵌套 n-gram / SPM 四种嵌入策略，前缀共享天然利于冷启与长尾泛化；(2) SiDec 内容流——用码本查表 φ 加轻量语义解码器 f_θ（MLP 或浅层 Transformer，MSE 对齐原始向量、推理时冻结或小学习率微调）在图内按需重建近似内容向量；历史/候选/正在观看的物料全部以 K 个整型 token 落日志，彻底消除稠密向量的存储、传输与 join。

**④结果亮点**：离线固定步数实验：直接吞 64 维内容向量使训练吞吐跌 28.2%（16.80→12.07 steps/s），SiDec 恢复至 14.53–15.41 steps/s 且 HR@100 反超原始稠密向量（0.2910 vs 0.2844）；已在 YouTube 生产排序与检索模型部署：Watchpage 满意交互 +0.80%、Homepage +0.22%、Homepage 检索 +0.13%（相对已用 SID 的基线，隔离 SiDec 流的净增益），长尾与稀疏新账号受益更大。

来源：[arXiv abs](https://arxiv.org/abs/2607.24865) ｜ [HTML 全文](https://arxiv.org/html/2607.24865) ｜ [ACM DL](https://dl.acm.org/doi/10.1145/3773078.3831900)（403，未能直接抓取）

---

### 5. PushDualGen: Enabling LLMs to Generate Semantic IDs with Interpretable Copy for Industrial Push Recommendation（快手，arXiv 2608.07989，2026-08-08）

**①思考背景**：快手 Push 推送触达近十亿用户，推荐不当会直接触发反感，对准确性与可追溯性的要求高于被动 feed；SID 生成式推荐端到端个性化但属黑盒，推荐逻辑难以解释，阻碍其在 push 场景上线；OneRec-Thinking 在生成 SID 前引入 CoT 提供了解释，但解码开销大增，无法满足 Push 场景的时延与吞吐要求（线上约 10 万 QPS）。

**②要解决的核心问题**：如何以近零的额外推理成本，让轻量 LLM 同时生成个性化 SID 与可解释的推送文案。难点：LLM 词表中不存在 SID、需专门对齐；"先解释再生成"的 CoT 范式代价过高；用户行为序列远超 LLM 上下文窗口，截断/随机采样会丢信号。

**③如何解决**：三段式设计：(1) 并行 SID 上下文压缩——用 CREM 式并行嵌入槽（M=8 槽、K=512 质心、每槽独立 K-means）避开残差量化的误差累积，将长历史压成紧凑 SID 序列；注册 L×K 个特殊 token，用 Text2SID / SID2Text 双向任务微调 Qwen3-0.6B（冻结原词表嵌入）让 LLM 理解并生成 SID。(2) 个性化 SID+文案联合生成——先生成 SID，经 ⟨id2text_sep⟩ 分隔符再生成"可跳过的解释"文案，损失为 L_SID + λ·L_Copy，配合高频 n-gram 合并的 multi-token binding 压算力。(3) 在线表示融合——将生成的 Top-N（N=20）SID 编码的偏好向量与用户特征加权融合后用于 ANN 召回，而非让 LLM 端到端直接服务线上。

**④结果亮点**：已在快手 Push 系统全量部署（服务近十亿用户、约 10 万 QPS）；14 天、约 1.5 亿用户 A/B（CUPED 校正）：有效播放率相对 +8.50%、负反馈（不满）率相对 -37.70%、点击 PV +0.43%、DAU +0.05%（均 p<0.05 显著）；长期看长尾内容曝光更公平、冷启更友好。

来源：[arXiv abs](https://arxiv.org/abs/2608.07989) ｜ [HTML 全文](https://arxiv.org/html/2608.07989v1)

---

## 横向观察（跨论文）

1. **多目标可控性成为新战场**：Multi-Decoder OneRec 证明"共享底座 + 隔离 LoRA 专家 + 配额感知约束解码"可以在生成式范式下保留工业界最看重的配额控制权，是 2026 年生成式召回走向多目标工业落地的代表性方案；TacGR 则补齐生成式模型相对判别式模型的"目标侧特征"短板。
2. **解码效率决定部署可行性**：OneLA（线性注意力大 beam）与 PushDualGen（SID 先行、解释可跳过）都表明，解码路径的系统成本是生成式推荐上线的第一约束——前者从底层算子/状态管理入手，后者通过生成顺序设计（先 SID 后文案）把解释成本变为可选。
3. **SID token 的角色在扩展**：SiDec 把 SID 从"物品标识符"扩展为"内容向量的压缩码"，PushDualGen 用并行 SID 压缩上下文——token 化正在从生成目标扩展为整个推荐系统的 I/O 与表示基础设施。

## Sources

**Kept（采纳）**：
- Multi-Decoder OneRec（arXiv 2607.26500 abs + HTML 全文）——快手官方一手论文，含方法全貌、Kwai26 基准与线上 A/B 数字。
- OneLA（arXiv 2609.12399 abs + HTML 全文）——港大+快手一手论文，含 GTR/ancestry 索引/融合 kernel 设计与全部评测数字。
- UniSGR-V2 / TacGR（arXiv 2610.16132 abs + HTML 全文）——阿里国际（Lazada）一手论文，含方法五模块与离线/在线实验。
- SiDec（arXiv 2607.24865 abs + HTML 全文）——Google/YouTube RecSys 2026 论文的 arXiv 版（ACM DL 403 不可达），方法与离线实验均取自原文；线上数字与 RecArch 摘要页交叉验证。
- PushDualGen（arXiv 2608.07989 abs + HTML 全文）——快手一手论文，全文完整抓取，含三段式方法与 A/B 结果表。

**Dropped（排除）**：
- LinkedIn 帖子（Kuldeep Singh Sidhu 对 SiDec 的转述）——二手评论，无超出原文的信息。
- awesome-all-you-need-papers 等 GitHub 列表——仅标题索引，无内容价值。
- RecSys 2026 program 页——仅作定位 SiDec 会议条目用途，未纳入正文引用（线上数字已交叉核对）。

## Gaps

1. **SiDec 的线上 A/B 细节**：ACM DL 版本无法访问（HTTP 403），arXiv HTML 中 Table 1 的线上具体数字未在正文渲染；Watchpage +0.80% 等数字经 RecArch 二手页面交叉验证，但 A/B 流量比例、实验时长与显著性水平原文均未披露。如需精确引用建议核对正式出版版 PDF。
2. **TacGR / Multi-Decoder OneRec 的离线结果表**：两篇 HTML 中表格数值部分未被文本化提取（如四项 Recall@512 的逐项绝对值、TacGR 主表的绝对 HR/AUC），本简报仅引用正文明确给出的相对提升数字。
3. **PushDualGen 的离线消融（Pass@k）具体数值**同样在表格中未渲染，仅确认各模块（SID 适配、并行 SID、token freeze、multi-token binding）均有正贡献。
4. **OneLA 的生产部署状态**：论文报告的是原型系统在工业 GR 工况上的评测（0.8B 模型、R/W/T 有限配置），未给出真实生产集群的端到端服务指标；与 vLLM 等基线的对比在"受控 FullState"管道下进行。
