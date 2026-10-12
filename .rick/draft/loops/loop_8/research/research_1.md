# research_1 — OneTrans-V2：一个 Transformer 统一召回/粗排/精排 — 2026-09

时间基准：arXiv 2609.28589v1（2026-09-25）。定位：只讲「如何做」。

## 事实性结论

1. **统一形态 = 共享用户上下文 + 阶段隔离，保留级联** — 把召回/粗排/精排表述为**同一个因果 Transformer 的三个任务**；candidate-independent 的用户行为序列每请求只编码一次为「共享用户上下文」，三阶段各自附加候选 token 读取它，阶段 token 相互 mask。服务仍按召回→粗排→精排走、候选逐级收缩；作者明确拒绝 OneRec 式端到端生成替掉级联（排序阶段依赖丰富候选特征）。— arxiv §1/§4/§5（A）
2. **混合参数化让异构 token 共用一个 backbone（继承 V1）** — 每层块内：行为 S-token 共享同一套 Q/K/V 与 FFN 权重；各阶段 NS-token 用各自 token-specific 的 Q/K/V 与 FFN。三阶段共用 Transformer 而不抹平异构特征。— arxiv §4（A）；V1 §3（A）
3. **输入统一：两分法 + 各阶段自持 tokenizer** — 输入 = 共享 S（append-only 行为序列）+ 阶段专属 NS_r/NS_p/NS_f（各自 tokenizer 生成 stage token）。粗排把特征子集压成**单个候选 token**；精排用**多个 token**（压成一个抹掉特征异质性）。stage token anchor 到请求时刻可见的行为前缀，不读未来。— arxiv §4.1/§4.2（A）
4. **召回输出 = 生成式（非打分）** — item 表示为三级语义标识 SID（RQ-KMeans 量化多模态 embedding，每级 codebook 8192），粗→细自回归生成，beam search 宽度 k 在码树解码，同 SID 的 item 全返回。— arxiv §4.2.1（A）
5. **DCGR：先预测决策前缀再生成 SID** — 决策四维：z_oc(订单数,3值)、z_disc(品类新颖度)、z_ad(有机/赞助,2值)、z_aov(客单价,依赖 z_oc)。前三者并行、各自轻量 head 从 CTX 隐藏态预测后求和成一个 DT^p；依赖维 z_aov 放第二个 DT^d；序列 CTX→DT^p→DT^d→SID0-2。业务目标=决策空间某区域，解码时给前缀分数加业务偏置 β·φ(z)（β 在线可调），一个生成过程替代多条召回通道。— arxiv §4.2.1（A）
6. **粗/精排输出 = 非自回归打分 head** — 在共享上下文缓存上逐候选打分：粗排用 NS_p 打召回候选、精排只用 NS_f 打存活候选；head 预测 CTR/CVR 二分类（T_p/T_f），BCE 损失 L_p/L_f。— arxiv §4.2.2（A）
7. **训练统一：加权联合损失 + 一次反传共塑共享上下文** — L = λ_r·L_r + λ_p·L_p + λ_f·L_f + λ_kd·L_kd。阶段 token 仍隔离，三任务只经共享上下文+共享 embedding 互影响。消融：召回梯度截断→召回 HR@1 +7.63%、粗/精排 CTR UAUC +0.19%/+0.21%；放行回传→+15.44%、+0.28%、+0.31%。— arxiv §7.1（A）
8. **模型内蒸馏 精排→粗排（KD）** — 精排 teacher、粗排 student，teacher logits detach，均值中心化+固定温度 τ 的 BCE（L_kd 对 T_p 求和），同一反传完成，无需独立 teacher。去掉 L_kd：粗排 CTR AUC −0.55%、UAUC −0.97%，精排几乎不动；线上逆序率 −6.7%、Top-10 覆盖 +31.4%。— arxiv §4.2.3/§7（A）
9. **SNT：按用户窗口组织训练，序列编码摊销** — append-only 序列每窗口因果编码一次，各曝光记录挂到其请求时刻可见前缀；请求 query 与行为 key 按各自时间戳旋转表达 recency。日窗口：训练 4.4× 加速、序列特征存储 −95%（更长窗口未测）。— arxiv §4.4/§7（A）
10. **Scaling 配方（可迁移经验）** — 稀疏 MoE(细粒度+sigmoid router)解耦容量与激活算力；gated attention、QKNorm、GQA；width-μP/depth-μP 使加宽加深免重搜超参；AdamW 替 RMSProp（后者缺偏差校正吞加宽收益）。多模态 embedding 直接注入：小模型有害、大模型有益。— arxiv §4.3/§7.8（A）
11. **V1 vs V2 演进 + 基线** — V1(2510.26104)=model-level 统一（序列建模+特征交互并入单排序模型，GMV +5.68%）；V2=cascade-level 统一（三阶段三任务）。基线：OneTrans-GR(独立 SID 召回,202G)、OneTrans-Lite(紧凑粗排,4G)、OneTrans(V1 精排,69G)，级联合计 275 GFLOPs；另比 UniSGR/UniPinRec。— arxiv §6.1（A）；2510.26104（A）
12. **结果与边界** — 离线：V2S(239G，略低于级联)三阶段全面超基线（召回 HR@1 0.0660 vs 0.0545=+21.1%；精排 CVR UAUC 0.7589 vs 0.7428=+2.17%）；V2L(662G=2.4×)全指标领先（HR@1 0.0923、粗排 CTR UAUC 0.7409、精排 CVR UAUC 0.7682）。线上 A/B(用户级 50/50)：GMV/u +9.74%、活跃天数 +0.25%、同硬件 QPS 3.2×。边界：3.2×≈2.16×(kernel融合+FP8) × 1.53×(共享上下文)；V2L 每请求 FLOPs 更高，QPS 是系统吞吐非算力下降；+9.74% 混杂 2.4×算力+scaling+联合训练+KD，论文无在线分解。— arxiv §6/§7.9（A）；arxivsignals 转述（B）

## R7 上报项
- 作者团队归属：二手源标「字节全球电商推荐 Foundation Team」，arXiv 页面未核实到署名团队。
- 召回损失 L_r 精确形式（决策 CE 与 SID CE 组合公式）在抓取片段中未显式出现。
- z_disc/z_aov 精确词表大小未核实（二手源仅称「各几个值」）。
- V2S 各阶段算力分配未披露（原级联召回独占 202G、精排仅 69G）。
- 3.2× QPS 逐项分解(2.16× × 1.53×)为二手转述，原文 §7.9 未逐字核对。
- 无公开代码/checkpoint，缺 A 级代码验证。

## Gaps
- +9.74% GMV 的归因混杂如何向同事如实表述，需 human 判断口径。
- 待下一轮：L_r 精确形式、决策词表值、beam 宽度 k、SNT 窗口长度。
- 推理引擎迁移点（KV cache 复用、混合参数化对 batching 影响、FP8/融合 kernel、每请求 GFLOPs 预算）留待 teach 展开，本材料只供事实。
