# 方向 C 第二轮深挖简报：端到端 / 自回归工业生成推荐与规模化（HSTU/LRM 系）

> 调研日期：2026-10-26。本文档为 `research-genrec/C-e2e-industrial.md`（首轮）的二轮深挖：**A** 逐条补全首轮 Gaps；**B** 对本方向最核心的 3 篇（TGR、Kunlun、MTGenRec）做接近可复现级深挖；**C** 补扫 2026-07 下旬 ~ 2026-10-26 窗口内首轮与顶会篇（`F-conference-papers.md`）均未覆盖的新论文。每条事实标注来源；无法核实的明确标注，未编造。

## Summary

首轮 5 篇论文的 Gaps 本轮全部处置完毕：**Kunlun** 的 ACM 正式版已通过 OpenAlex 核实（KDD '26 V.2，pp.7379–7390），部署叙事确认只到"major Meta Ads models + topline +1.2%"（论文匿名化，具体指标与模型清单不可得）；**MTGenRec** 补齐 A/B 分桶（2% 流量）与全部实验细节，但 A/B 时长与 110G 变体 PV_CTR +1.90% 确认论文未报告/未文本化（二手存疑）；**TGR** 补齐 CCFormer 公开基准全表、五场景在线 A/B 全指标、BARGE/HiGR 分场景数字与 Appendix C 默认超参；**Deco** 补齐作者（Gusak/Volodkevich/Oseledets/Frolov）与全部实现细节（机构仍非一手）；**LoopGR** 补齐附录 A.1/A.2/A.4（工业数据集规模确认论文未披露）。深挖 3 篇给出伪代码级算法流程、关键公式与训练配方。遗漏扫描发现 5 篇窗口内未覆盖新论文：**CCFormer、BARGE、OneTrans-V2、TransX、LazFormer**，另有 6 项边界注记。

---

## A. 首轮 Gaps 逐条补全（对照 `C-e2e-industrial.md` 末尾「Gaps」节）

### A.1 Kunlun（Meta，arXiv 2602.10016）

| 原 Gap | 处置结果 |
|---|---|
| ACM DOI 10.1145/3770855.3818438 未直接核对 | ✅ **已核实**（ACM DL 403，改走 OpenAlex）：正式收录于 *Proceedings of the 32nd ACM SIGKDD Conference on Knowledge Discovery and Data Mining* **V.2**，出版日期 2026-08-07，pp.7379–7390，CC-BY（hybrid OA），26 位作者全部 Meta（另 2 位署 OpenAI、工作在 Meta 期间完成），第一作者 Bojian Hou。来源：[OpenAlex 记录](https://api.openalex.org/works/doi:10.1145/3770855.3818438)、[DOI](https://doi.org/10.1145/3770855.3818438) |
| "topline 指标 +1.2%"的指标定义 | ❌ **论文未披露**：§5.5 原文为 "Kunlun has been deployed across major Meta Ads models, delivering a 1.2% improvement in topline metrics and demonstrating superior ROI at scale"——topline 指标按 Meta 惯例匿名化，无定义。来源：[arXiv HTML v3 §5.5](https://arxiv.org/html/2602.10016v3) |
| 部署的广告模型清单 | ❌ **论文未披露**：仅表述为 "major Meta Ads models"（匿名化，无具体模型名）。来源同上 |
| （新增补全）scaling law 与消融细节 | ✅ 详见 B.2 深挖节 |

### A.2 MTGenRec（美团+武大，arXiv 2505.12663，KDD '26 V.1）

| 原 Gap | 处置结果 |
|---|---|
| A/B 分桶数 | ✅ **已补**：美团外卖（Take-away）应用 **2% 流量** A/B，对照基线为已**持续在线学习 2 年**的 DLRM。来源：[arXiv HTML v3 §4.2](https://arxiv.org/html/2505.12663v3) |
| A/B 时长 | ❌ **论文未报告**（全文检索 "traffic" 等关键词仅见 2% 流量表述，无实验周期） |
| 每应用部署清单 | ❌ **论文未披露**：仅"deployed for many applications in Meituan…handling hundreds of millions of requests daily"，唯一具名应用为外卖平台。来源：[abs](https://arxiv.org/abs/2505.12663) |
| GRM 推理侧指标 | ❌ **论文未报告**（本文为训练系统论文，聚焦训练侧）。来源同上 |
| 110G 变体 PV_CTR +1.90%（仅二手 recarch） | ⚠️ **一手未能确认**：arXiv v3 全文检索 "1.90" 无匹配；Table 1（离线/在线对比表）在 HTML 中为图片，无法文本化。该数字**仅存在于二手资料页** [recarch](https://jdding.github.io/recarch/en/systems/meituan-mtgenrec-2025/)，保留存疑标注。论文正文明确报告的是：离线 CTCVR GAUC +2.88pp、外卖订单 +1.22%、PV_CTR +1.31%（未指明变体）。来源：[arXiv HTML v3](https://arxiv.org/html/2505.12663v3) |
| （新增补全）模型变体与硬件 | ✅ GRM 变体按**每前向 GFLOPs** 命名：4G / 110G（110G 为复杂度 27.5×）；embedding 按维度因子 1D/8D/64D（64D 为 32×）。硬件：最多 16 节点 × 8 = **128 张 A100 SXM4 80GB**，节点内 NVLink 600 GB/s、节点间 InfiniBand 200 GB/s；优化器 Adam。序列均衡对 GRM 110G 平均 +26.5% 吞吐（64 GPU 峰值 +33.5%）；128 GPU 时达到理想加速的 62.75%–78.5%。来源：[arXiv HTML v3 §4.1–4.3](https://arxiv.org/html/2505.12663v3) |

### A.3 TGR（腾讯 PCG，arXiv 2609.00986）

| 原 Gap | 处置结果 |
|---|---|
| CCFormer 公开基准（vs STCA）明细 | ✅ **已补**：Taobao 上 CCFormer AUC **93.67±0.32**，超最强序列基线 STCA（92.81±0.50）0.86 点；KuaiRec 上 **83.35±0.29**，超该集最强基线 OneTrans（82.76±0.59）0.59 点；相对 DIN 的 ΔAUC 分别 +13.93 / +10.36（STCA +11.69、OneTrans +8.41）。工业数据集（**44B+ 样本、30M 用户、10M 物品、序列长 1000**）：CCFormer AUC 77.94 / GAUC 71.36，超 HSTU（77.66/70.86）+1.01%/+2.40% 相对，超 STCA（77.73/70.95）+0.21/+0.41 点。超参鲁棒：压缩核 k 从 2→7 时 AUC 仅在 77.92–77.95% 波动。来源：[arXiv HTML v1 §3.4 Table 2–3](https://arxiv.org/html/2609.00986v1) |
| BARGE/HiGR 分场景 A/B 完整数字 | ✅ **已补**（详见 B.1）：CCFormer 五场景全指标（Table 6）；BARGE 双场景离线全表 + 在线（6% 流量：CTR +0.60%、点击 UV +1.34%、阅读时长 +1.70%，后全量）；HiGR 三场景（Table 14：停留 +1.03 / 观看时长 +1.22 / 播放量 +1.73 / 请求 +1.57；CTR +0.68 / 广告收入 +0.56；观看时长 +1.14 / 播放量 +0.88，5% 流量）。来源：[arXiv HTML v1 §3.6/§4.2/§4.3](https://arxiv.org/html/2609.00986v1) |
| Appendix C 默认配置 | ✅ **已补**（部分）：见 B.1 训练配方节（BARGE 训练硬件一栏抓取片段截断，未完整获取）。来源：[arXiv HTML v1 Appendix C Table 19](https://arxiv.org/html/2609.00986v1) |

### A.4 Deco（arXiv 2610.32980）

| 原 Gap | 处置结果 |
|---|---|
| 作者机构 | ⚠️ **部分补全**：作者为 **Danil Gusak、Anna Volodkevich、Ivan Oseledets、Evgeny Frolov**（v1 2026-10-23、v2 2026-10-26），经 arXiv API 核实；但 abs 页**未标注机构**。旁证：RecSys '26 会务页显示 Gusak/Frolov 隶属莫斯科 "AI Center"（页面匿名化为 AXXX），同组在 RecSys '26 另有 CoSID 与 Codebook-Based SIDs 立场论文——机构归属仍非一手确认。来源：[arXiv API](http://export.arxiv.org/api/query?id_list=2610.32980)、[RecSys'26 机构页](https://programs.sigchi.org/recsys/2026/people/affiliations/axxx)、[ACM 作者页](https://dlnext.acm.org/profile/99661361226) |
| "production-style serving pipeline"性质 | ✅ **确认为模拟**：单张 **H100 80GB**、bf16 权重、内核编译、CUDA graphs、beam 共享历史缓存、beam 宽 100——为统一 serving 管线模拟，**非真实线上部署**，线上收益未报告（与首轮判断一致，维持标注）。来源：[arXiv HTML v2 §4.1.4/App C](https://arxiv.org/html/2610.32980v2) |
| （新增补全）实现细节 | ✅ 全量补齐（见下）：SID K=4 级、V=2048 码，RQ-KMeans 基于多语 E5-large 文本向量（Yambda 用官方 128 维物品向量）+ 流行度碰撞消解；骨干 8 层 Qwen2 风格 decoder、宽 640、10 头；轻量解码器 2 层、宽 256、4 头、M=2 概念向量；从头训练骨干 S/M/L = 4M/20M/69M（宽 128/384/640，FFN 512/1536/2560，2/4/8 层，2/6/10 头）；LLM 骨干用 Qwen2.5 0.5B/1.5B/3B + LoRA（先做元数据-SID 双向 grounding，LoRA 合并后计时）；Adam + warmup-cosine，验证集 N@10（beam 10 选点、beam 100 测试）；KD 权重一次性设定 w_ce=1/(3ℓ̄_ce)、w_kd=2/(3ℓ̄_kd)，τ=1（Beauty/MovieLens τ=2）；Yambda 全模型加 copy 指针。来源：[arXiv HTML v2 §4.1.4/App C](https://arxiv.org/html/2610.32980v2) |

### A.5 LoopGR（腾讯 PCG+中山大学，arXiv 2610.34157）

| 原 Gap | 处置结果 |
|---|---|
| 工业数据集规模细节（附录 A.1） | ✅ **已抓取并确认**：附录 A.1 实为**公开数据集统计**——Beauty 22,363 用户/12,101 物品/198,502 交互（均长 8.88）、Sports 35,598/18,357/296,337（8.32）、CDs 75,258/64,443/1,097,592（14.58）；**工业数据集规模论文未披露**（正文仅称"collected from Tencent's production recommendation system"，附录无补充）——确认为论文未报告，非抓取遗漏。来源：[arXiv HTML v1 附录 A.1 Table 5](https://arxiv.org/html/2610.34157v1) |
| （新增补全）RTSD 压缩效率（附录 A.2） | ✅ 教师从 16 循环压到 8/4 循环：计算 17.408→8.415/3.918 GFLOPs，batch=1 延迟 211.08→104.01/49.82 ms（约 2.0×/4.2× 加速），精度保持（4 循环学生在 Sports 四指标全超 16 循环教师，8 循环学生在 Beauty 取得最佳 R@10/N@10）。来源：[arXiv HTML v1 附录 A.2 Table 6](https://arxiv.org/html/2610.34157v1) |
| （新增补全）上下文更新器消融（附录 A.4 + Table 4） | ✅ 工业数据集 R@5：Identity 0.5513 / GRU 0.5626 / Linear-Attn 0.5556 / **Mixing 0.5691**（默认）；相对 Lazy Decoder/TIGER/BARGE 最大提升 11.47%/7.13%/3.91%。token 级更新器（线性注意力 O(L_u d²)、MLP、恒等）与物品级更新器（GRU：均值池化+双向 GRU+广播，O(nd²)；Mixing：H×d 分组通道混合+层级专属 FFN，O(L_u d²)）公式均已获取。来源：[arXiv HTML v1 §5.5 Table 4 + 附录 A.4](https://arxiv.org/html/2610.34157v1) |

---

## B. 方法内部深挖（3 篇，接近可复现级）

选篇理由：**TGR** 是本方向目前最完整的"全栈工业化"报告（排序→生成→推理三线全部署）；**Kunlun** 是 Meta HSTU/LRM 谱系在"序列+上下文联合建模 scaling law"上的直系推进；**MTGenRec** 是 GRM 百卡级分布式训练系统的工程答案——三者合起来覆盖方向 C 的"端到端 / 自回归 / 规模化"三个关键词。

### B.1 TGR：腾讯生成推荐全栈（arXiv 2609.00986，2026-09-01）

#### B.1.1 架构组成（三层栈 + 共享底座）

- **TGR-GenRank（CCFormer）**：保留级联、替换排序器。输入特征空间分为三场：用户画像 token **U∈R^{Lu×d}**、行为序列 token **S∈R^{Ls×d}**（每历史物品 1 个细粒度 token）、目标商品 token **T∈R^{Lt×d}**；堆叠 L 个交互块迭代精化三场，末层 T 表示进任务专属多头，输出**逐商品多任务分数**（drop-in 兼容现网）。[来源](https://arxiv.org/html/2609.00986v1)
- **TGR-GenRec（BARGE / HiGR）**：BARGE = NTP（逐物品）范式生产模型；HiGR = NSP（整页 slate）范式生产模型，两者各有独立 tokenizer（见下）。[来源](https://arxiv.org/html/2609.00986v1)
- **TGR-Reason**：LatentRec 训练的离线 Think 模型导出"reason tokens"（完整 L 级 SID），在线经 Direct Reasoning Injection 注入生成器，请求路径零推理 rollout。[来源](https://arxiv.org/html/2609.00986v1)
- **共享底座**：SID tokenizer 家族（slate 生成用 PCRQ-VAE、NTP 生成用 OSQ-VAE）+ 统一用户/上下文编码接口 + Numerous-Torch 训练推理基础设施（Megatron 式 3D 并行 + 跨 GPU embedding 缓存；默认 NVIDIA H20 节点集群，节点内 NVLink、节点间 400 Gbps）。[来源](https://arxiv.org/html/2609.00986v1)

#### B.1.2 关键机制与公式

**CCFormer 四机制**（每块内替代全局自注意力，从不计算拼接 token 集上的 self-attention）：
1. **特征场分离交叉注意力**：三条有向流实现排序所需的异构交互——用户→序列（全历史偏好检索）、目标→序列（目标-历史相关性匹配）、目标→用户（抓取片段在该处截断，按上下文推断；以原文为准）；目标 token 互不 attend → 无跨目标泄漏，支持单遍并行打分全部候选。
2. **相对时间-位置编码（RTPE）**：局部 m 个行为的组内偏置 = 可学习时间衰减 **α·β^{|ti−tj|^γ}**（β∈(0,1)，时间邻近行为权重更大）+ 可学习相对位置偏置，代价 **O(Ls·m)**（注意力矩阵偏置为 O(Ls²)）。
3. **长序列子空间 token 混合（STM）**：序列张量重塑为 (m 相邻行为 × n 隐藏通道) 紧凑子空间，每个通道组由独立**门控 per-channel FFN** 处理，token 级与通道级信号在局部子空间直接交互，代价对 Ls 线性（论文称首次将该结构用于该目的，原文该句被截断）。
4. **卷积序列压缩**：每块后 strided 卷积（核 k=3、步幅 s=2），下一块序列长度几何递减——深层块以极低成本保有全历史感受野。整体复杂度：CCFormer 避免 O(Ls²)，同容量下每样本 GFLOPs 约为 HSTU 一半（d=256：18.28 vs 38.37；d=512：36.14 vs 63.47）。[来源](https://arxiv.org/html/2609.00986v1)

**BARGE（NTP）**：
- **OSQ-VAE（正交分裂量化）**：可学习 Householder 参数化旋转 R（构造上 R^T R=I），把旋转后的物品表示切到两个正交坐标子空间，各自独立残差码本栈量化 → 每物品两条 SID 通道；共享编码器 + 两个通道专属解码塔（各配私有输出头）。
- **ICA（物品上下文感知注意力）**：编码器内恢复被压平破坏的物品级结构；门控设计（门趋零时退化为恒等路径），公开数据上四层 SID 平均门激活 0.35–0.38（适度注入而非覆写）。
- **HPR（层级路径重排）**：第 l 层用双塔打分器评估解码器预生成隐状态 **h₀** 与累计路径嵌入 **p^(l)=Σ_{j=1..l} e_{c_j}** 的兼容性；与主模型联合训练（对称 InfoNCE，label-free）；推理时对 Top-N 池（N=400）重排、保留 Top-B（beam 20），出束宽度不变。
- **DPD + OR-fusion**：双通道并行解码、Top-K 列表用 LSE（log-sum-exp 软 OR）融合；辅助对称 InfoNCE + NTP 损失。[来源](https://arxiv.org/html/2609.00986v1)

**HiGR（NSP）**：
- **PCRQ-VAE**：标准 RQ-VAE 骨干 + 两项塑形——全局量化项（聚合码字与 latent 对齐，防深层码本残差消失，承诺强度 η=0.1）+ 前缀对比项（**仅前 D−1 层**码本把语义/协同相关物品拉近；λ1=0.1、λ2=0.01、(w1,w2)=(1,0.1)，InfoNCE 温度 τ、逐层权重 w_d）；消融证伪扩展到叶层（碰撞率 2.37%→8.73%，前 D−1 层方案：碰撞 2.37%、浓度 93%、一致性 66.47%）。
- **HSD（层级 slate 解码器）**：粗粒度 **slate planner**（l_slate=14 层）在紧凑偏好嵌入空间自回归（每 slate 位 1 个嵌入）锁定全局列表意图 + 共享细粒度 **item generator**（l_item=2 层）把每个偏好落地为短 SID 序列；planned 几何上加**列表内多样性（ILD）正则**保持位置区分（去掉则多样性 0.62→0.59）；推理时 planner 贪心、beam search 只在物品级短序列内进行——把 M×D 长 token 序列分解为 M 步规划 + 批量短序列，训练/推理均大幅省算。
- **ORPO 列表级对齐**：三目标 slate 偏好对（排序保真、真实兴趣、多样性），系数 α=0.1；消融去掉 ORPO 后列表指标明显下降（如 HiGR-25M 首列指标 0.2825→0.2641）。[来源](https://arxiv.org/html/2609.00986v1)

**TGR-Reason（LatentRec）**：
- **Continuous Thought Module**：从用户上下文隐状态 h₀ 展开 K 步潜推理（仅训练期；K_train=2）；**Soft Token Selection（STS）**：每步隐状态投影回输入 embedding 分布再进入骨干；**Per-step Reasoning Loss（PRL）**：经共享预测头对**每个中间状态**监督（对齐目标 SID 第一级），温度 T_b=1、T_s=5、损失权重 λ=0.1；服务期 K=0 直接丢弃推理模块。
- **训练三阶段**：词表扩展（SID token 入骨干）→ 内容对齐（可训练 token 微调对齐平台内容）→ 推理阶段（**只训 LoRA 适配器 + CTM**，骨干冻结）；Think 模型周期性离线导出每用户 top-Kr 完整 SID reason tokens；**DRI** 注入：reason tokens 作为层级对齐提示条件在线解码；可另查询 **Group Memory**（前向窗口 W、每 reason-token 查询保留 M 个后继；该路径为设计扩展，未进入部署配置）。[来源](https://arxiv.org/html/2609.00986v1)

#### B.1.3 算法流程（伪代码级）

```text
# CCFormer 单块前向（按原文机制拼装；块内精确串行次序以原文为准）
输入: U∈R^{Lu×d}(用户画像), S∈R^{Ls×d}(行为序列), T∈R^{Lt×d}(目标商品)
对 l = 1..L(默认 8):
  1) 特征场分离交叉注意力(三条有向流):
     S ← S + CrossAttn(Q=S, K=V=U)      # 用户→序列: 全历史偏好检索
     T ← T + CrossAttn(Q=T, K=V=S)      # 目标→序列: 目标-历史相关性
     T ← T + CrossAttn(Q=T, K=V=U)      # 目标→用户(片段截断, 按上下文推断)
     # T 内部互不 attend → 单遍并行打分所有候选
  2) RTPE: S 组内偏置 += α·β^{|ti−tj|^γ} + 可学习相对位置偏置   # O(Ls·m)
  3) STM: reshape(S, m 行为 × n 通道) → 各通道组门控 per-channel FFN → 逆 reshape  # O(Ls)
  4) 卷积压缩: S ← StridedConv(S; k=3, s=2)   # 下一块序列几何变短
输出: 末层 T → 任务专属多头 → 逐商品多任务分数

# BARGE 解码(OSQ-VAE 双通道 + HPR)
tokenizer: R(Householder, R^TR=I) → 旋转表示切两个正交子空间 → 两套残差码本 → SID 通道 A/B
共享编码器(含 ICA 门控注入物品上下文) → 解码塔 A、B 并行逐码生成
每 SID 层 l: 双塔打分 score(h₀, p^(l)=Σ_{j≤l} e_{c_j})
  → 对 Top-N(400) 池重排, 保留 Top-B(20) 路径 → 层间不扩束
终层: 两通道 Top-K 列表 LSE OR-fusion 合并 → 候选物品
训练: NTP + 对称 InfoNCE(HPR 双塔, label-free)

# HiGR 解码(PCRQ-VAE + HSD + ORPO)
tokenizer: RQ-VAE + 全局量化损失(η) + 前缀对比(仅前 D−1 层, λ1/λ2/w_d)
planner(14 层): 在偏好嵌入空间自回归生成 M 个 slate 位置嵌入(贪心)
  + ILD 正则保持位置几何可分
item generator(2 层, 所有位置共享): 每个偏好嵌入 → beam search 短 D 级 SID 序列
后训练: ORPO 列表级偏好对(排序保真/真实兴趣/多样性, α=0.1), 3% 预训练样本

# TGR-Reason(离线 Think + 在线注入)
训练: 词表扩展 → 内容对齐 → 冻结骨干, 只训 LoRA+CTM(PRL 监督每步, K_train=2)
离线周期管线: Think 模型标准解码导出每用户 top-Kr 完整 SID reason tokens
在线: reason tokens 作为层级对齐提示前缀注入 TGR-GenRec 解码(零请求期 rollout)
```

#### B.1.4 训练配方与超参（Appendix C Table 19 + 正文）

| 组件 | 默认配置 |
|---|---|
| CCFormer（工业设置） | 特征 token 维 d=256；8 个 CCFormer 块；子空间组 m=8、n=16；压缩核 k=3、步幅 s=2；Adam lr=1e-4；batch 4096；16×NVIDIA H20 统一硬/软件栈 |
| BARGE（公开基准设置；线上通道见 §4.2.5） | SID 深度 L=4，码本 (512,256,128,64) 全层可学习；编码/解码塔各 2 层 4 头（共享编码器、双解码器）；embedding/attention/FFN 维 128/512/1024；Adam+warmup，batch 256，200 epochs（早停，3 seeds）；beam B=20，HPR 池 N=400，融合权重 λ=0.25；OR-fusion=LSE；训练硬件原文为"2…"（片段截断，未完整获取） |
| HiGR | d_model=512、d_FFN=2048；SID 深度 D=3、每层码本 1024；planner l=14、generator l=2；PCRQ-VAE η=0.1、λ1=0.1、λ2=0.01、(w1,w2)=(1,0.1)；ORPO α=0.1；1B 预训练样本（3% 用于 ORPO 后训练）；部署规模 25M/100M，scaling 研究 0.05B–2B；H20 训练、L20 serving（KV 缓存） |
| TGR-Reason | K_train=2、PRL 权重 λ=0.1、T_b=1、T_s=5；骨干冻结 + LoRA；服务期 K=0 |
| 基础设施 | Numerous-Torch：混合精度训练；ID embedding 表 **INT8 对称线性量化（约 70% 压缩）+ double-hashing（再省约 50% 稀疏参数）**；单遍并行候选打分；H20 节点（NVLink + 400 Gbps） |

来源：[arXiv HTML v1 §3.3/§4.2/§4.3/§5.3/App C](https://arxiv.org/html/2609.00986v1)

#### B.1.5 部署与线上证据（分场景全量数字）

- **CCFormer（Table 6，五场景；每组 >100 万日曝光用户、两周窗口、双样本 t 检验 p<0.05）**：S1（视频推荐，已全量）：CTR +3.57、UCTR +1.93、3 日活跃 +1.93、视频观看 +3.47、独立观众 +2.61、PV +3.86、观看时长 +1.29、广告收入 +1.64；S2（广告排序，已全量）：广告曝光 +1.43、**广告收入 +1.71**；S3：CTR +2.02、UCTR +1.67、新用户 +4.01、PV +1.29、广告收入 +1.29；S4：PV +0.55、完播 +0.98、观看时长 +0.42、关注 +2.18；S5：PV +0.45、观看时长 +1.26、分享 +3.49、CTR +1.01。
- **Scaling**：序列长 0.5k→2k，AUC 77.72→78.17、GAUC 70.95→71.57（对 HSTU 平均相对 +1.08%/+2.37%，且差距随变长拉大；**CCFormer@0.5k 已追平 HSTU@2k 的 AUC、反超 GAUC**）；模型维 128/256/512 单调升（77.65→78.13 AUC）。消融：去序列压缩 → 加速 2.21×→1.29×（AUC 基本不变，效率杠杆）；去 STM → 质量最大损失（−0.21 AUC/−0.34 GAUC），换回全局自注意力仅 +0.02 AUC、反丢 0.15 GAUC 且加速降至 1.41×（质量主力）。
- **BARGE**：离线两场景 Hit@5 较最强基线 OneRec +10.2%（S1：0.6015 vs 0.5459）/ +16.9%（S2：0.0976 vs 0.0835），Hit@50 仍 +12.6%；在线 6% 流量：CTR +0.60%、点击 UV +1.34%、阅读时长 +1.70%（独立论文摘要交叉确认），后全量且收益更大更稳；参数/beam/延迟零增量。
- **HiGR**：离线整页质量较等容量 OneRec +15.9–21.3%，推理 5× 加速、P99<50ms、GPU 需求约 −60%；0.05B–2B 训练损失与 NDCG@5 呈幂律；在线三场景（5% 流量）见 A.3 表。
- **TGR-Reason**：离线冷启动新用户 Hit@1 +477.8%；线上 +1.75% 有效消费率、新用户曝光→转化 +13.09%。

来源：[arXiv HTML v1 §3.5–3.6/§4.2.3–4.3.5/§5.5](https://arxiv.org/html/2609.00986v1)；BARGE 独立版交叉核对 [arXiv 2607.21028](https://arxiv.org/abs/2607.21028)、CCFormer 独立版 [arXiv 2607.28070](https://arxiv.org/abs/2607.28070)

### B.2 Kunlun：Meta 海量推荐 scaling law（arXiv 2602.10016，KDD '26 V.2）

#### B.2.1 架构组成

基于 Interformer 框架做模型-效率协同设计，L 层，每层两块：**Kunlun Transformer 块**（GDPA 增强的 PFFN + 滑窗多头自注意力 MHA）+ **Kunlun Interaction 块**（Weight Generation：从非序列特征生成 PFFN 个性化权重；HSP：层级序列摘要；Global Interaction：序列摘要与非序列特征及非序列特征间的交互）。特征预处理：稠密特征拼接投影、稀疏查表、多事件类型序列沿 embedding 维拼接经 MLP 融合；**ROTE 旋转时间编码**扩展 RoPE：τ_t=log(1+Δt/τ_scale)，旋转矩阵元素 cos/sin(tθ_i+τ_t·φ_i) 同时编码位置频率 θ_i 与时间频率 φ_i。垂直（层数）与水平（宽度）均可扩展。[来源](https://arxiv.org/html/2602.10016v3)

#### B.2.2 关键公式与损失

- **Scaling law**：NE(C)=NE₀−η·log(C/C₀)；scaling efficiency = η/η_baseline，分解为算法收益（每 FLOP 的 NE 改善）× 硬件利用率（MFU）。主指标为 **NE（Normalized Entropy，越低越好）**（选 NE 而非 AUC：NE 直接反映生产排序关键的校准质量）。
- **GDPA**（把 PFFN 重构为多头注意力式算子，可 FlashAttention 式融成单 kernel）：每头 GDPA_h(Q,K,V)=Activation_h(QK^T/τ)·V，其中 **Q=S^(l)（序列 token 作 query）、K=w1^(h)(X_sum)、V=w2^(h)(X_sum)（非序列上下文的两套可学习投影）、温度 τ=maxlen(seq)**（实证优于 d 或 √d）；残差连接使 PFFN 可多层堆叠（原 PFFN 不可堆叠）；PFFN 块 MFU 最高提升 6×。
- **HSP**（替代 PMA 随机 query 池化）：三阶段——①可学习种子 E←LearnableSeeds(n_s,d)（批间共享）；②种子注意力 H_s←MHA(Norm(E),S,S)（种子 attend 全序列）；③**SumKronLinear** 压缩 H←Σ_{i=1..k} Z_i^T·H_s·W_i → n_t 个输出 token。
- **滑窗注意力**：Q_t 只 attend K/V∈[t−w,t+w]，复杂度 O(T²)→O(T·w)，窗口 w 按事件类型可配（+31.1% QPS）。
- **CompSkip**：隔层模式——偶数层跳过自注意力但计算新 HSP 摘要；奇数层跳过 HSP（复用上层摘要）与 PFFN 但计算自注意力；自注意力/HSP/PFFN 的 FLOPs 各省约 50%，总 FLOPs −43.1%。
- **事件级个性化**：按事件类型重要性（点击>曝光）分配资源——每事件类型独立配置 d_model^(e)、n_heads^(e)、窗口 w^(e) 等（配置项列表在抓取片段处截断）。
- 复杂度表：GDPA O(T·N·d)、滑窗 O(T·w·d)、全注意力 O(T²·d)、HSP O(n_s·T·d)、全局交互 O(M·(N+n_t)²·d)。
- 附录 C 提供 4 个完整算法：层前向（含 skip 逻辑）、HSP、GDPA、CompSkip 配置生成。[来源](https://arxiv.org/html/2602.10016v3)

#### B.2.3 伪代码（层前向 + CompSkip 配置，按附录 C 算法 1/4）

```text
CompSkip 配置(L 层): for l=0..L−1: 偶数层{skip_self_attn=True, skip_hsp=False};
                               奇数层{skip_self_attn=False, skip_hsp=True, skip_pffn=True}

Kunlun 层前向(l):
  X_sum ← HSP(S) (若 skip_hsp: 复用 H_prev)
  for h = 1..H:  # GDPA, 融合单 kernel
    K_h ← W_h^K·X_sum;  V_h ← W_h^V·X_sum;  Q_h ← W_h^Q·S
    O_h ← Act_h(Q_h·K_h^T / τ)·V_h        # τ = maxlen(seq)
  S̃ ← S + W^O[O_1|…|O_H]                  # 残差使 PFFN 可堆叠
  (若 skip_pffn: 跳过上步)
  全局交互: X^(l+1) ← GlobalInteraction(X_sum, X)   # 非序列↔序列摘要
  S^(l+1) ← SlidingWindowMHA(S̃)+S̃ (若 skip_self_attn: S^(l+1)←S̃)
  return X^(l+1), S^(l+1), H_curr
```

#### B.2.4 实验与训练配置

- 数据：Meta Ads 内部数据集，**70B+ 样本**、数百~数千非序列特征、10+ 条序列特征（长数百~数千、多事件类型：点击/转化等）。
- 消融（同硬件同训练配置，B200 上测 MFU；ΔNE 正=更差，全模型 154.9 GFLOPs/样本）：w/o GDPA：MFU 34.0%、QPS −8%；w/o HSP（换 PMA）：MFU 30.21%、QPS +8.8%、NE +0.08%、GFLOPs −25%（HSP 是花 QPS 买质量的投资）；w/o SumKronLinear：34.55%、GFLOPs −7.5%、NE +0.03%；w/o 滑窗：32.90%、QPS −31.1%、GFLOPs +29.5%；w/o CompSkip：34.50%、QPS −35%、GFLOPs +43.1%；w/o 事件个性化：35.7%、QPS −13%、GFLOPs +11%、NE −0.02%；w/o 专家并行：35.5%、QPS −4%；**全模型 MFU 37.0%**（较起点 17% 翻倍以上）。
- Scaling 对比（对 Wukong 基线的 NE 增益，6/60/180 GFLOPs 三档）：Wukong+PMA 0.16%/0.11%/0.20%；InterFormer 0.31%/0.36%/0.50%；**Kunlun 0.31%/0.66%/0.79%**（180 档 FLOPs 还更低：154.9 vs 172.3，1.58× 优势）→ scaling 效率 2×。
- 逐层 scaling：1→6 层，ΔNE_l≈c/log(l+1)，收益递减但可预测。
- 部署：major Meta Ads models，topline +1.2%（匿名化，见 A.1）。[来源](https://arxiv.org/html/2602.10016v3)

### B.3 MTGenRec：美团 GRM 分布式训练系统（arXiv 2505.12663，KDD '26 V.1）

#### B.3.1 模型与系统架构

- **GRM 模型**：输入序列 T=[T_con(上下文/用户特征), T_hst(点击/购买等历史), T_exp(候选曝光)] → embedding 层 φ_emb → 多层 **HSTU** → **MMoE** top-k 专家聚合 → 交叉熵损失。HSTU 单层：U,Q,K,V=Split(φ1(MLP(E)))；O=φ2(QK^T)V；H=MLP(Norm(O⊙U))（φ1/φ2 为 SiLU）。MMoE：y=Σ_{i=1..k} g_i(H)·Expert_i(H)。平均序列长度 600。
- **分布式策略**：稀疏模型并行（单卡装不下全表）+ 稠密数据并行；端到端同步训练；embedding 查找 All-to-all 两次交换（ID 交换 + 向量交换）；反向异构更新（稀疏表按模型并行本地聚合梯度更新分片，稠密参数 All-Reduce 同步）；下一批次数据预处理与当前计算流水线化。[来源](https://arxiv.org/html/2505.12663v3)

#### B.3.2 核心机制（公式级）

1. **动态哈希 embedding 表**：MurmurHash3（4 字节块多级混合、雪崩效应）+ 开放寻址 + **分组并行探测**：S=(k mod (M/threads−1)+1 | 1)·threads（| 为按位或；基步长强制奇数 + 步长绑定 key + 乘线程组号）。**定理 3.1**：表大小 M=2^n、探测步长 S 为奇数 ⟺ 探测序列覆盖全部 M 个槽位。键（ID→指针）与值（向量+时间戳等辅助数据）解耦存储、chunk 化分配（每 chunk 批量插删）；负载因子 >0.75 时键结构倍增扩容（只迁键不迁值），双 chunk 预分配保插删即时。查表五步：hash → 探测定 index → p=st_add+index·row_offset+pointer_offset → 取向量。
2. **自动表合并**：FeatureConfig 统一接口（特征名/维度/查找表），按相同 embedding 维度等模式自动合并；动态表用位运算生成全局唯一 ID：k=⌈log2(m+1)⌉ 位高位作表标识，**ID=(i≪(63−k))|x**（最高位置 0 保正，例：3 表 k=2，Table 2/3 偏移 2^59/2^60）。
3. **两阶段 ID 去重**：ID 通信前（本地去重省 ID 通信）+ 通信后（跨设备重复再去重，主要省 embedding 向量回传带宽）。
4. **动态序列分桶**：每 GPU 维护 Hive chunk 缓冲 Q；目标 token 数 N=平均序列长(600)×batch；对缓冲内序列 token 数累加和 S 二分查找最接近 N 的 k，弹出前 k 条成批；不足则并入下一缓冲迭代。注意：各设备样本数不同，直接 All-Reduce 平均梯度有偏（论文讨论了相应处理）。[来源](https://arxiv.org/html/2505.12663v3)

#### B.3.3 伪代码（训练工作流 + 动态分桶）

```text
# MTGenRec 训练步(混合并行)
1. 数据 I/O: HDFS 分区 Hive 列存, GRM 按"用户↔多条目序列"组样(避免用户特征重复计算)
2. Embedding 查找: 本地 ID 去重 → All-to-all ID 交换 → 接收端再去重
   → All-to-all embedding 向量交换 → 重组
3. 前向: HSTU 层×N → MMoE top-k → CE 损失
4. 反向: 稀疏表本地梯度聚合+分片更新(模型并行); 稠密 All-Reduce 梯度同步
5. 流水线: 与 2-4 并发预取/预处理下一批

# 动态序列分桶(每 GPU)
Q ← 从 Hive chunk 填充序列缓冲
loop: S ← Q 内 token 数前缀和; k ← 二分找最接近 N=600×batch 的位置
      若 k<len(Q): 输出批 B=Q[:k]; 否则 Q 并入下一 chunk 继续
附加: checkpoint 续训、混合精度、梯度累积、算子融合
```

#### B.3.4 实验与部署

- 评估数据：美团一周内 **2 亿真实用户序列**；硬件见 A.2（128×A100）。GRM 变体 4G/110G（GFLOPs/前向）、1D/8D/64D。
- vs TorchRec：整体训练时间缩短 **1.6–2.44×**（含 lookup/forward/backward 各阶段）；128 GPU 达理想加速 62.75%–78.5%；复杂度 ×27.5（4G→110G）与 embedding 维 ×32（2D→64D）只造成轻微加速衰减。
- 消融：序列均衡整体 **1.75× 吞吐**（GRM 110G 平均 +26.5%、64 GPU 峰值 +33.5%，且拉满 GPU 显存利用率）；ID 去重 +53% 吞吐；合并表/去重/均衡增益随复杂度上升而放大。
- 业务：GRM 与 DRM 训练成本相近下离线 CTCVR GAUC **+2.88pp**；外卖平台 2% 流量 A/B（对照持续学习 2 年的 DLRM）：订单 **+1.22%**、PV_CTR **+1.31%**。[来源](https://arxiv.org/html/2505.12663v3)

---

## C. 遗漏扫描：窗口内（2026-07 下旬 ~ 2026-10-26）首轮与顶会篇均未覆盖的新论文

> 检索方式：TGR/LoopGR/Kunlun 引文图谱顺藤 + arXiv API 日期核实 + 多引擎检索。以下 5 篇均经 arXiv API 核实提交日期，均未被 `C-e2e-industrial.md`、`F-conference-papers.md`、`research.md` 覆盖。

### C.1 CCFormer：腾讯工业推荐的高效跨场交互与层级序列压缩（arXiv 2607.28070，2026-07-30 提交；v2 2026-10-29 更新）

**①思考背景**：工业排序须在严格算力预算下同时处理长行为历史与异构用户/商品特征；HSTU/OneTrans 证明了统一 token + Transformer 可换来 scaling，但全局自注意力 O(Ls²) 成本迫使截断或预压缩，牺牲跨字段细粒度交互。**②核心问题**：构造把"跨字段检索"与"历史内建模"分离的结构化交互架构，在次平方成本下保留全历史访问，同时仍输出逐商品分数以便 drop-in 替换现网排序器。**③如何解决**：特征场分离交叉注意力（用户/目标 query 读取不断演进的历史层级）+ 局部子空间 token 混合（独立于候选目标地为历史 token 注入上下文）+ 层级序列压缩（多尺度历史上下文），从不计算拼接 token 集上的全局自注意力。**④结果亮点**：两个公开基准平均 AUC 超所评基线；大规模工业数据集上精度-算力权衡与训练吞吐均优于 HSTU（TGR 主文：2.21× 训练加速、约半 GFLOPs）；在线 A/B 四个推荐/广告场景最高 CTR +3.57%、PV +3.86%、广告收入 +1.71%，v2 摘要称已四场景全量部署（与 TGR 主文"五场景两全量"构成部署时间线互补）。注：本文即 TGR-GenRank 生产模型的独立论文版（TGR 引文 [22]）。来源：[arXiv abs](https://arxiv.org/abs/2607.28070)（日期经 [arXiv API](http://export.arxiv.org/api/query?id_list=2607.28070) 核实）

### C.2 BARGE / Bridging the Structural Gap：适配自回归生成的推荐（腾讯，arXiv 2607.21028，2026-07-23 提交；v3 2026-08-20）

**①思考背景**：层级 SID 逐 token 生成的 GR 存在两个结构性缺口——多 token 物品压平进单一序列破坏物品级结构；训练与推理在层级码本上的不一致（早期前缀错误不可逆传播）造成语义漂移。**②核心问题**：不加大模型、不加宽 beam，弥平这两个缺口。**③如何解决**：ICA（物品上下文感知注意力，门控注入、四层平均门激活 0.35–0.38）恢复编码侧物品结构；HPR（层级路径重排：双塔打分 h₀ 与累计路径嵌入的兼容性，对称 InfoNCE 联训，Top-N 池重排保 Top-B）+ DPD（OSQ-VAE 正交 Householder 旋转双通道解码）从解码侧互补抑制漂移。**④结果亮点**：公开基准与大规模离线测试全面占优（TGR 场景 2 Hit@5 较 OneRec +16.9%）；腾讯平台在线 A/B：CTR +0.60%、点击 UV +1.34%、阅读时长 +1.70%（6% 流量，后全量）。注：TGR-GenRec NTP 生产模型独立论文版（TGR 引文 [23]）。来源：[arXiv abs](https://arxiv.org/abs/2607.21028)

### C.3 OneTrans-V2：一个 Transformer 统一召回/粗排/精排（arXiv 2609.28589，2026-09-23，Hannan Cao 等 21 人）

**①思考背景**：工业推荐的召回/粗排/精排级联通常各自独立训练与 serving，造成用户序列被反复编码、优化互相隔离、工程重复；OneTrans V1 只完成模型级统一。**②核心问题**：用一个 Transformer 统一整条级联，同时保留各阶段特有的候选特征与计算，并在规模化时保持稳定。**③如何解决**：用户行为序列只编码一次为共享上下文；三阶段联合训练互相增强 + 模型内精排→粗排知识蒸馏；稀疏 MoE 扩骨干（容量增而有界激活）+ μP 式参数化稳 scaling；DCGR（决策条件生成检索：先预测描述即将发生交互的"决策前缀"再条件生成物品，业务目标操控单一生成过程）；SNT（序列原生训练：围绕用户终身行为序列组织训练、跨曝光摊销编码）。**④结果亮点**：部署于大规模工业推荐系统全部三阶段：**GMV +9.74%**；协同设计 serving 栈在同等硬件预算下吞吐为被替换级联的 **3.2×**。来源：[arXiv abs](https://arxiv.org/abs/2609.28589)（论文未披露具体平台）

### C.4 TransX：面向推荐的序列转导规模化技术报告（arXiv 2610.20913，2026-10-14，Da Xu 等 8 人）

**①思考背景**：序列转导（HSTU 式）推荐基础模型的规模化是一重挑战，在生产级延迟/成本约束下跑起来是另一重——模型容量增长不应成比例推高在线服务成本。**②核心问题**：如何显式控制"模型-算力分配"与"执行摊销"，实现容量与服务成本解耦。**③如何解决**：把序列编码、行为记忆、候选条件融合拆为独立可缩放组件；训练把因果轨迹计算摊销到行为前缀、记忆状态扇出到候选；serving 解耦 bulk prefill / 近线刷新 / 在线预测（紧凑行为记忆 KV）；实证发现"各向异性 scaling 回报"——编码器深度超中等后饱和、监督增多把分配前沿推向候选条件融合、保质量所需记忆分辨率随容量增长大体稳定。**④结果亮点**：生产 A/B：核心模型 **CTR +6.02%、转化 +4.45%**；后训练召回/偏好排序变体 CTR +2.96%/转化 +1.33%；训练侧监督目标吞吐 +84%；serving 解耦较共置**总成本降近 80%**，同等总成本下每请求模型 FLOPs 达 HSTU 式生产基线约 **5×**、同等每请求 FLOPs 下每 H200 吞吐约 **4×**。来源：[arXiv abs](https://arxiv.org/abs/2610.20913)、[HTML](https://arxiv.org/html/2610.20913)（作者 Da Xu 在 [LinkedIn](https://www.linkedin.com/posts/da-xu-62106b159_transx-technical-report-scaling-sequence-activity-7518358552734838784-UDZu) 官宣；论文未明示公司）

### C.5 LazFormer：经可迁移生成式预训练缩放工业推荐 Transformer（arXiv 2609.14978，2026-09-14，Xiaodong Li 等 11 人）

**①思考背景**：工业推荐 Transformer 通常靠单一排序模型从头同时优化稀疏+稠密参数，算力消耗大、收敛慢；预训练可提供良好初始化，但直接迁移有两大限制。**②核心问题**：(1) 预训练与排序的输入特征不一致，稠密参数直接迁移会负迁移；(2) 排序阶段多 epoch 训练稀疏参数易过拟合，冻结又限制对排序目标的适配。**③如何解决**：生成式预训练模块自回归生成序列特征（同时初始化稀疏+稠密参数）；可迁移残差适配器以残差方式注入排序特有特征（解负迁移）；请求感知排序模块集成长序列压缩、混合稀疏注意力与请求感知范式高效建模长序列；非对称多 epoch 策略（epoch 间重置稀疏参数、持续累积稠密参数）缓解稀疏过拟合。**④结果亮点**：论文报告在多数据集与工业部署中验证各组件有效（本条为摘要级；正文具体数字未在本轮抓取范围内展开，如需可下钻 HTML 全文）。来源：[arXiv abs](https://arxiv.org/abs/2609.14978)、[HTML](https://arxiv.org/html/2609.14978)

### 边界注记（窗口内外但与本方向强相关，未计入上述 5 篇）

- **LaRec**（arXiv 2607.24617，2026-07-27，窗口内）：LLM 潜空间推理 GR——潜空间预训练（step-level + 过程方向对齐）+ 个性化高斯混合分布采样多样推理起点 + RL-tuning。更偏 LLM 推理侧（方向 E），但与 TGR-Reason/LatentRec 同题，可作方向 E 补充。来源：[arXiv abs](https://arxiv.org/abs/2607.24617)
- **CoSID: Concept-Conditioned Semantic-ID Decoding**（RecSys '26，2026-09，Gusak/Volodkevich/Frolov）：Deco 同组会议版"概念条件 SID 解码"，F 篇未覆盖；与 Deco（2610.32980）互为姊妹篇。来源：[ACM 作者页](https://dlnext.acm.org/profile/99661361226)
- **IBA / Where Reasoning Matters**（arXiv 2607.12425，2026-07-14）：SID 各位置信息增益分析（早层高、晚层低）+ 潜推理步预算按位置分配——窗口早 7 天，勉强界外。来源：[arXiv abs](https://arxiv.org/abs/2607.12425)
- **GR4AD 可靠/可扩/价值对齐广告 GR**（arXiv 2610.38378，提交 2026-10-28）：快手广告 GR 框架后续——**晚于 2026-10-26**，属索引边界现象（同首轮 G 节处理方式），建议下轮纳入。来源：[arXiv abs](https://arxiv.org/abs/2610.38378)
- **HiGR 独立论文版**（arXiv 2512.24787，2025-12-31，窗口外）：TGR-GenRec NSP 生产模型的独立论文（"Industrial-Scale Hierarchical Generative Slate Recommendation Framework in Tencent"），TGR 主文已覆盖其核心内容。来源：[arXiv abs](https://arxiv.org/abs/2512.24787)
- **NVIDIA 技术博客：用 Dynamo-Triton 部署 HSTU 生成推荐器**（2026-09-30，非论文）：HSTU 生产部署的工程参考，可与 MTGenRec/TGR 部署节互相印证。来源：[NVIDIA Developer Blog](https://developer.nvidia.com/blog/deploying-an-hstu-generative-recommender-with-nvidia-dynamo-triton/)

---

## Sources

**保留（一手为主）**：
- TGR：[arXiv abs](https://arxiv.org/abs/2609.00986) + [HTML v1 全文](https://arxiv.org/html/2609.00986v1)（§3–5、Table 2–7/11/14/19、附录 C 逐段抓取）
- Kunlun：[arXiv abs](https://arxiv.org/abs/2602.10016) + [HTML v3 全文](https://arxiv.org/html/2602.10016v3)（方法/实验/附录 C 算法/附录 E 表）；[OpenAlex DOI 记录](https://api.openalex.org/works/doi:10.1145/3770855.3818438)（正式版页码/日期/作者核实，ACM DL 403 的替代通道）
- MTGenRec：[arXiv abs](https://arxiv.org/abs/2505.12663) + [HTML v3 全文](https://arxiv.org/html/2505.12663v3)（§3–4、附录 B 算法）；[DOI](https://doi.org/10.1145/3770854.3783925)
- LoopGR：[arXiv abs](https://arxiv.org/abs/2610.34157) + [HTML v1 全文](https://arxiv.org/html/2610.34157v1)（§4–5、附录 A.1–A.4）
- Deco：[arXiv abs](https://arxiv.org/abs/2610.32980) + [HTML v2 全文](https://arxiv.org/html/2610.32980v2)（§3–4、附录 A/C）+ [arXiv API 作者核实](http://export.arxiv.org/api/query?id_list=2610.32980)
- 新论文：CCFormer [abs](https://arxiv.org/abs/2607.28070)、BARGE [abs](https://arxiv.org/abs/2607.21028)、OneTrans-V2 [abs](https://arxiv.org/abs/2609.28589)、TransX [abs](https://arxiv.org/abs/2610.20913)、LazFormer [abs](https://arxiv.org/abs/2609.14978)、LaRec [abs](https://arxiv.org/abs/2607.24617)、IBA [abs](https://arxiv.org/abs/2607.12425)、HiGR [abs](https://arxiv.org/abs/2512.24787)、GR4AD [abs](https://arxiv.org/abs/2610.38378)——日期均经 [arXiv API](http://export.arxiv.org/api/query) 核实
- 旁证：[RecSys '26 机构页](https://programs.sigchi.org/recsys/2026/people/affiliations/axxx)、[Gusak ACM 主页](https://dlnext.acm.org/profile/99661361226)（CoSID/立场论文）、[Da Xu LinkedIn](https://www.linkedin.com/posts/da-xu-62106b159_transx-technical-report-scaling-sequence-activity-7518358552734838784-UDZu)、[NVIDIA HSTU 部署博客](https://developer.nvidia.com/blog/deploying-an-hstu-generative-recommender-with-nvidia-dynamo-triton/)、[recarch MTGenRec 页](https://jdding.github.io/recarch/en/systems/meituan-mtgenrec-2025/)（二手，仅 110G +1.90% 一项存疑引用）

**剔除**：dl.acm.org 直连（一律 403，改用 OpenAlex/arXiv 替代通道）；emergentmind/paper-daily/knowthepaper/chiosa 等聚合站（SEO 二手，事实以 arXiv 原文/API 为准）；LinkedIn 仅用于 TransX 官宣佐证，不作数字来源。

---

## 剩余 Gaps（本轮之后仍未解决）

1. **MTGenRec 110G 变体 PV_CTR +1.90%**：一手（arXiv v3）无法确认——全文无 "1.90" 匹配，Table 1 为图片未文本化；仅有 recarch 二手来源。**建议**：获取论文 PDF 原表或向作者确认。
2. **MTGenRec A/B 实验时长**：论文未报告（仅 2% 流量与两周级信息缺失）；**每应用部署清单**仅"many applications"，未具名。
3. **Kunlun topline +1.2% 的指标定义与部署模型清单**：论文匿名化披露（"major Meta Ads models"），一手不可得；实验模型的具体层数/维度配置表在抓取片段中亦未完整出现（仅有 1–6 层 scaling 与 154.9 GFLOPs 全模型）。
4. **TGR Appendix C Table 19 的 BARGE 训练硬件条目**：抓取片段在"Training hardware 2…"处截断，未完整获取；CCFormer 第三条注意力流（目标→用户）的原文表述在抓取片段中被截断，本报告按上下文推断并已标注。
5. **Deco 作者机构**：abs 页与 HTML 均未标注；RecSys '26 会务页显示莫斯科 "AI Center"（匿名 AXXX），机构归属非一手确认。
6. **LoopGR 工业数据集规模**：确认论文未披露（附录 A.1 仅为公开数据集统计），非抓取遗漏。
7. **LazFormer 正文数字**：本轮仅摘要级覆盖（C.5 已标注）；如需可下钻其 HTML 全文。
8. **索引边界现象**：CCFormer v2（10-29 更新）、GR4AD（10-28 提交）等日期晚于"今天 2026-10-26"，与首轮 research.md G 节现象一致，建议以 arXiv 页面为准复核。
