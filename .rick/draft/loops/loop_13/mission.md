# 学习使命 — loop_13

主题：学习当前 infra RSI 领域的最新进展论文，建设在线推理（online inference）领域的 RSI 系统

## 为什么学（human 原话）
在美团真实落地，带来业务收益与技术影响力。

## 学习目标与考核标准（human 原话）
能对当下领域最前沿哪些做法有效，哪些没效有一个认知。并且转化为建设美团内部 infer RSI 的原创思考。

（展开：① 对 infra RSI 前沿做法形成「有效 / 无效 + 为什么」的判断力；② 该判断转化为美团在线推理场景 RSI 系统的建设方案原创思考——最终检验是这套思考能否支撑真实落地。）

## 起点（human 原话）
我对这个领域怎么做有实践经验，但是缺少系统性认知。在线推理 infra 我非常熟悉。

我系统学习过，alphaevloe 论文，对这些内容不熟，现在主要想学习下 FlashRSI，Helion × CompileIQ，Harness²，Harness Updating Is Not Harness Benefit，RRSI ，SoL-Pi，https://mp.weixin.qq.com/s/aQFrApfzfMaKCtDWFmcKPw ，ForgeKernel，https://mp.weixin.qq.com/s/lkBbY7IV2wShunmo-JOO5w ，DiscoveryRSI

## 起点（parent 备注，仅供 research / teach 校准）
- human 是美团 ML 系统架构师：在线推理 infra（serving / KV cache / 路由 / 评测等）专业水平，教学无需从入门讲起
- AlphaEvolve 已系统学习过——进化搜索 + LLM proposer + evaluator 闭环相关概念可直接引用，无需重讲
- 本次学习核心范围是 human 点名的 10 份材料（8 个名称 + 2 篇微信文章），需逐一定位原始出处与核心机制
- 产出导向：不是综述式学习，而是「有效性判断 + 美团 infer RSI 建设原创思考」，指向真实落地
