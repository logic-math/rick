# loop_13 ZPD 记录（zpd-assessor 独写；跨 loop 记忆文件）

> loop_13 前无 anchors/zpd 留存，本文件为冷启动首块。期望 = mission 目标（infra RSI 前沿有效性判断 + 美团在线推理 RSI 建设原创思考）；现状 = 待探针测量。本节记录基线校准探针轮。

## [2026-11-01 19:52] probe（冷启动基线校准） — 探针轮：AlphaEvolve 强项核验 · RSI 部件语言 · verifier 迁移面

### human 自报基线摘要（来源：mission.md）
- 在线推理 infra 专业水准（serving / KV cache / 路由 / 评测），教学无需入门讲起
- AlphaEvolve 已系统学习：进化搜索 + LLM proposer + evaluator 闭环，相关概念可直接引用
- 不熟（= 本会话教学对象，不作探题）：FlashRSI、Helion×CompileIQ、Harness²、HUINHB、RRSI、SoL-Pi、ForgeKernel、DiscoveryRSI 及 2 篇微信文章
- 自评：对该领域怎么做有实践经验，但缺系统性认知
- 产出导向：前沿做法「有效 / 无效 + 为什么」的判断力 → 美团 infer RSI 建设方案原创思考

### 探针设计依据
- 探题全部踩在 mission 自报强项 + research_1「概念清单」的 RSI loop 部件通用词汇上；新工作细节一律不考
- 3 题对应三个校准面：① 自报强项（AlphaEvolve）核验 ② 通用部件语言深度 ③ infra 经验迁移面
- 难度对齐专业水准，不出入门题；允许遗忘——回忆过程本身即知识再创造，不苛求完整

### 探针轮记录（探题 + 考察目的）

**探针 1｜AlphaEvolve 闭环机制——核验自报强项**

题目：凭记忆讲 AlphaEvolve 的闭环，重点在机制而非印象：
① 程序数据库怎么组织？LLM 作为 proposer 在每一步看到什么输入、产出什么？
② evaluator 的分数如何驱动候选的保留与淘汰？
③ 为什么要维护 population（岛模型、周期迁移、多样性重播种这类强制多样性设计），而不是只保留当前最优解？去掉这套多样性机制，搜索行为会退化成什么样？

考察目的：mission 自报「AlphaEvolve 已系统学习」是本会话最重要的对照锚——待学新工作（FlashRSI、ForgeKernel 等）大量以其为基线变体。此题核验自报是否达到机制级（部件怎么运转 + 设计为什么存在），直接决定 lesson-writer 能否引用 AlphaEvolve 作对照下钻、还是需要先补讲。

**探针 2｜RSI 部件通用概念——语言深度**

题目：不引用任何具体论文，每个概念用 2-3 句独立说清：
① harness——RSI / agent 语境里指什么？典型的可编辑部件有哪些（举几类即可）？
② proposer 与 verifier 在 loop 里的分工是什么？verifier 的「正确性 gate」与「性能信号」为什么必须分开看？
③ OOD 过拟合（在系统 / 代码优化 loop 里常称「OOD 缩水」）——怎么发生、典型形态、通用防御手段？

考察目的：这组部件词汇（harness、proposer、verifier/evaluator、OOD 缩水）是 10 份新材料的公共承重语言。能独立说清 → 教学可直接用术语并下钻新机制；含糊 → 需先统一部件级语言再进入新工作。是详略档位的核心输入之一。

**探针 3｜在线推理 verifier 迁移面——实践经验 → 新领域**

题目：用你在美团在线推理的实操经验回答：
① 在线推理场景里，哪些量天然适合作 RSI loop 的 verifier（既当正确性 gate 又当性能信号）？
② 哪些量看起来可用但实际不能直接用？为什么（如在线指标噪声、流量波动、混杂因素）？
③ 挑一个你认为最可靠的 verifier，说明一个「聪明但错误」的候选——钻计时空子、缓存输出、把算子转嫁给框架侧——会在它面前怎么暴露？

考察目的：mission 的落点是「美团 infer RSI 建设原创思考」。此题测 infra 经验能否直接迁移成 verifier 设计判断——这是从「学习新工作」到「原创方案」最关键的迁移面；回答也将成为后续教学「结合你的场景」的锚点素材。

### 待 human 回答（判档位留空）
- [ ] 探针 1 回忆原文：待 human 回答
- [ ] 探针 2 回忆原文：待 human 回答
- [ ] 探针 3 回忆原文：待 human 回答
- [ ] 档位（强 / 中 / 弱）：待判——回答到达后按 rubric 判定，并补写对错条目、缺口清单、详略建议（给 lesson-writer）、节奏建议（给 parent：推进 | 同层复测 | 二次探针后回退）
