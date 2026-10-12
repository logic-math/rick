# research_4 — 模块3「共享 context 的统一架构」— 2026-10-11

时间基准 2026-10-11(UTC)。一手原文 arxiv.org/html/2609.28589v1（§3/§4/§5/§6/§7）。A=原文；转述=据 related work。

## 一、候选无关 user context：定义与一次编码复用

1. **user context = 候选无关的用户行为序列** S=(s1,…,sL)：按到达顺序 append-only 的用户历史事件，不含候选 item 或非序列特征；候选特征分属三阶段 NS_r/NS_p/NS_f。"候选无关"因候选随请求与阶段变化，行为序列一次请求内对全部候选、三阶段不变，是唯一可跨级联共享的部分。— §4.1+§5.1（A）
2. **一次编码复用**：每请求先把 S 经共享 causal Transformer 编码成 cache C_u，同一 C_u 被三阶段消费——"每请求编码一次而非每阶段一次"；各阶段把 stage token 挂在 context 后、在掩码下 attend 可见前缀，behavior token 从不 attend stage token，故 context 一次编码、三阶段只读。— §4.1+§5.1（A）
3. **与 HSTU M-FALCON**：M-FALCON 复用缓存历史态跨候选 microbatch 与请求；SNT 把该原则推进到跨级联——同用户窗口内不同请求+阶段的 exposure 共享一份序列编码。（转述）— §2.4+§4.4（A）

## 二、token 级独立参数（mixed parameterization）

4. **源自 OneTrans 并落地 V2**：行为 token 同质共享一套 Q/K/V+FFN，非序列 token 用 non-shared、token-specific Q/K/V+FFN；V2 每层 TransBlock 内行为 token 共享一套、各阶段 stage token 各自用 token-specific 的 Q/K/V+FFN。— §3+§4.1+Fig.2(b)（A）
5. **独立的是"参数"而非"token 本身"**：Fig.2(b) 明说行为 token 共享参数、stage token 每 token 独立参数；token 是数据、参数是权重入口，backbone 层不变、按 token 类型分流权重。— Fig.2(b)（A）
6. **GQA 边界与"无独立公式"**：stage token 与行为序列间只需 KV head 数、head dim 一致，query head 数可不同；混合参数化是参数路由规则（Fig.2(b)+Alg.1 的 fused W^{QKVG}），公式对象是 block 前向(Alg.1)与各阶段损失(Eq.6/7/9)。— §4.1/§4.3（A）

## 三、阶段可见性掩码（stage visibility mask）

7. **四条规则**（Fig.2(c)+§4.1）：①行为 token 在 S 内 causal attend；②stage token 只 attend 可见行为前缀(到 anchor)+自己+同 stage/exposure 中前置 token；③不同 stage/exposure 互 masked；④behavior 从不 attend stage token。— §4.1（A）
8. **anchor 与分层隔离**：stage token 锚定在请求时间戳前最后到达的行为 a_e，只读 a_e 前缀，同请求各 exposure 共享 anchor，新行为 append 不改旧前缀→编码一次后各 exposure 仍等于在线所见。掩码让每阶段只看到"共享 context+自己候选 token"，看不到他阶段特征、不能改 context→三任务独立前向（与 UniR² 相反）。— §2.2+§4.1+§4.4（A）
9. **工程实现**：该 attention 跑在 FlashAttention 式 kernel 上，内置 mask，一次调用覆盖一用户全部候选，无 padding、不物化 attention 矩阵。— §5.2（A）

## 四、共享而不破坏分层：候选特征与预算隔离

10. **召回=生成**（无"候选 token"）：NS_r 概括成 CTX token 接在 S 后，自回归生成 DT^p→DT^d→SID0-2，beam search 出 top-K 交粗排。— §4.2.1（A）
11. **粗排单 token vs 精排多 token**：粗排预算最紧，限轻量子集 NS_p⊂NS_f 压成单 token，heads 预测 CTR/CVR(Eq.6)；精排只评存活候选，用全 NS_f 保持多 token 保特征异质性，heads 预测更多目标(聚焦 CTR/CVR)。— §4.2.2（A）
12. **预算隔离与成本**：共享 backbone+context 下各阶段 token 预算独立；serving 仍走三级联，每阶段只执行自己 candidate-side 计算。现有级联每请求 275 GFLOPs，V2 S 三阶段 239（略低仍超各基线）、L 2.4×，粗排评数千、精排评数百候选。— §4.1+§5.1+§6.1（A）
13. **共享带来的耦合=一致性**：共享 context+模型内精排→粗排蒸馏(Eq.8，无独立 teacher)拉近两阶段排序，inversion rate −6.7%、top-10 coverage +31.4%。— §4.2.2+§7.2（A）

## 五、human 直觉对照

human："把共享上下文提取出来，各自阶段生成独立 token，再拼接上去"。

14. **「提取共享 context」成立**：候选无关行为序列一次编码成 C_u、三阶段只读复用。✅
15. **「各阶段生成独立 token」方向对、出入在"独立"**：token 由各阶段 tokenizer 从 NS 特征产出挂到 S 后（成立）；但"独立"是 token 级独立参数（每 token 自己的 Q/K/V+FFN），非"每阶段一个表示向量"，粒度 per-token 非 per-stage。— Fig.2(b)/§4.1（A）
16. **「再拼接」成立但缺掩码**：token 确实 append 在 context 后；无 visibility mask 则跨阶段/exposure 互看、behavior 被反向修改，破坏"一次编码只读"。掩码是关键，human 直觉未含此层。— §4.1（A）
17. **候选 token 挂法——召回例外**：粗/精排是候选特征 token 化挂载评分；召回无候选 token，是"CTX+决策前缀→生成 SID"，候选由 beam search 产出。— §4.2.1（A）
18. **「独立 token」≠「独立 backbone」**：backbone（层/残差/归一化/MoE）全共享，独立的是 token 参数入口+各自 heads+各自损失(Eq.9)。— §4.1+§4.2.3（A）

## R7 上报项

- 混合参数化/掩码精确实现（W^{QKVG} 如何按 token 路由、mask kernel 布局）无开源代码，仅据 Fig.2/Alg.1 文字。
- "token-specific 参数"粒度（每 token 一套 vs 同阶段同类型共享一套）未给参数配置/数量。
- beam width k、top-K、各阶段候选精确数（除"粗排数千/精排数百"）论文只给符号。
- HSTU M-FALCON 原文未读，其与 SNT 关系基于 §2.4 转述；behavior 参数梯度隔离细节论文未细述。

## Gaps

- 掩码+mixed parameterization 在自家栈的工程落地（kernel 如何按 token 分流权重）需 human 结合实现判断，无代码对照。
- "token-specific 参数"粒度与参数量增长需下轮或 human 确认。
- 各阶段候选数/beam width/top-K 落地口径需结合论文 §5 与自家漏斗换算（沿用 research_1/3）。
