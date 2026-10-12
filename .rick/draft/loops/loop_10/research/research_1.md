# research_1 — OneTrans-V2 领域全景调研 — 2026-10-11

时间基准：2026-10-11（UTC）；论文 arXiv:2609.28589v1 [cs.IR]，v1 提交 2026-09-23。

## 一、这篇论文是什么

- **身份**：OneTrans-V2: Unifying Retrieval, Pre-rank, and Fine-rank with One Transformer in Industrial Recommender。作者为 ByteDance Foundation/Architecture 两队共 21 人（通讯作者 liushaowei.nphard@bytedance.com）；正文未印机构，邮箱后缀推断字节跳动/抖音。来源=论文原文（A；机构为 A 级证据+推断）。
- **要解决的问题**：工业召回→粗排→精排级联各阶段独立建模/服务，导致用户行为序列被每阶段重复编码、孤立优化无法共享表示与监督、工程重复、容量碎片化；前作 OneTrans(WWW'26) 只统一了单阶段内建模。来源=原文 §1（A）。
- **方法一句话**：把召回/粗排/精排当作一个因果 Transformer 的三个任务，共享"候选无关的用户行为序列编码(user context)"，各阶段以 token 级独立参数+阶段可见性掩码保留候选特征；召回用 DCGR（先预测 purchase/discovery/sponsored/spending 决策前缀，再条件生成 3 级 SID），训练用 SNT（按用户窗口组织、序列只编码一次）。来源=原文 §4（A）。
- **结果一句话**：线上 A/B，OneTrans-V2(L) 替换三级联后 GMV/用户 +9.74%、活跃天数 +0.25%，同硬件预算 QPS 3.2×；离线召回 HR@1 +69.36%（0.0545→0.0923）。来源=原文 §6.2/6.3（A）。

## 二、领域核心资料（A 级优先）

- **奠基论文（A）**：TIGER(NeurIPS'23, arXiv:2305.05065) 生成式检索+语义 ID 源头；HSTU(ICML'24, arXiv:2402.17152)，开源 meta-recsys/generative-recommenders；OneRec(arXiv:2506.13695)；OneTrans(WWW'26) 本作前身；MTGR(CIKM'25, arXiv:2505.18654, 美团) 工业生成式推荐，与 human 场景直接相关。
- **级联各阶段经典（A）**：DeepFM / DIN(KDD'18) / SIM(CIKM'20) / DCN V2(WWW'21)、COLD(粗排, arXiv:2007.16122)、RankFlow(SIGIR'22)、COPR(CIKM'23)。来源=原文 §2 引用（A）。
- **教程/综述（A/B）**：WWW'25 生成式推荐 Tutorial（large-genrec.github.io）+ 配套《Generative Recommendation with Semantic IDs: A Practitioner's Handbook》(ACM)；LLM 时代生成式搜推综述 arXiv:2404.16924（B）。
- **可复现代码（A=源码）**：NVIDIA/recsys-examples(sid_gr/hstu)、Kuaishou-OneRec/OpenOneRec、AkaliKong/MiniOneRec、phonism/genrec、RecBole(RUCAIBox)。
- **缩放基石（A）**：DeepSeekMoE/DeepSeek-V3（稀疏 MoE+sigmoid 路由）、Tensor Programs μP（Stability-AI/mup 官方仓库）、FlashAttention。来源=原文 §4.3 引用+官方仓库（A）。

## 三、专家公认的学习/练习方法

- **主线＝手写最小生成式检索**：教程与 Handbook 分两阶段——①用 RQ-VAE/RQ-KMeans 把 item 量化成语义 ID；②搭 autoregressive decoder 用 beam search 解码 SID。来源=WWW'25 slides+Handbook（A/B）。
- **工业级联按"漏斗经济学"逐层练**：召回（亿级候选→数千）→粗排（单 token 轻量）→精排（多 token 富特征），每层有独立 latency/吞吐预算，"一次编码、多候选复用"；对照 MTGR/COLD 的 per-stage 预算练。来源=原文 §1/§4.2.2+教程（A/B）。
- **缩放要"先验超参迁移"**：用 μP 仓库小模型调参→放大后超参不变；MoE 用细粒度 expert+sigmoid 路由+shared expert。来源=Stability-AI/mup+原文 §4.3/§7.7（A）。

## 四、世界观 + 主要争论

- **核心世界观**：推荐正从"多模型级联"走向"单一生成式/序列 transduction 模型"（HSTU/OneRec/OneTrans-V2 一线），把推荐重铸为 next-item 预测，共享 user context 一次编码被多阶段复用。来源=原文 §1-2+综述（A/B）。
- **争论1：端到端替换级联 vs 保留级联统一**。OneRec 路线删级联（风险：丢失候选富特征）；OneTrans-V2/UniR²/UniPinRec 保留级联只共享 context。原文明确选后者。来源=原文 §1 Fig.1（A）。
- **争论2：生成式检索 vs 判别式双塔/ANN**。《The Case Against Generation for Retrieval》(arXiv:2607.25346)主张 LLM 作双塔骨架更省算力；GenRec(arXiv:2604.14878) 列 GR 工业三难题（翻页不一致、长序列编码成本、策略对齐）。来源=两篇预印本（B）。

## R7 上报项

- 机构归属：论文正文未印机构，仅由通讯邮箱 @bytedance.com 推断字节跳动/抖音，缺官方声明。
- 论文无开源代码/checkpoint 链接（OneTrans-V2 未开源），落地需自行复现，缺官方实现。
- "与 OneTrans 是否同团队、是否含字节外合作"未在署名中澄清，缺归属说明。

## Gaps

- 离线 GFLOPs 口径（662/239 vs 级联 275）如何换算到 Meituan 自身栈，需 human 结合自家预算判断。
- DCGR 四决策维在 Meituan 搜推的对应物（GMV/发现/广告/供给）需 human 映射，本调研未覆盖 Meituan 内部目标定义。
- 论文为 2026-09 极新预印本，社区独立复现/质疑（除自引用外）尚缺，需后续跟踪。
