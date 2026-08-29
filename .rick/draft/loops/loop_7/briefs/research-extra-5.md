# research-extra-5 简报

> loop_7 E 门禁 r1 追加（事实消解）| pi 扩展 server 能力面 / pi-web-ui 架构 / 语义巡检先例 | 2026-08-26 | 本地 pi 源码+web 检索

## ① pi 扩展 server 能力面（问题 4）

- 扩展=普通 TS 模块，jiti 加载进 pi 主进程，**无沙箱、与 pi 同权限**（docs/security.md），node:http 可用 ⇒ **技术上可起 server/监听端口/接受外部连接**（已验证：代码+文档原文）。
- 生命周期**会话级非进程级**：工厂函数禁启后台资源（sockets/processes/timers），须 defer 到 session_start、幂等 session_shutdown 清理；/new /resume /fork /clone 及进程退出均触发 session_shutdown（extensions.md L222-224+生命周期图）⇒ 扩展内 server 只活在「宿主进程+当前会话」。
- 进程级长驻的官方路径**不是扩展**：SDK `createAgentSession`（独立 Node 进程内嵌 agent，文档明示 web UI 用例）或 RPC 模式 `pi --mode rpc`（JSON over stdio）（docs/sdk.md、docs/rpc.md）。

## ② pi-web-ui 架构与成本（问题 4）

- xiaoyc/pi-web-ui（pi.dev 收录）：独立 Node 进程，**pi SDK in-process+WebSocket 推流**；功能含流式聊天/工具调用/内置 terminal/模型与 prompt 管理/skills+extensions 开关。
- Firstp1ck/pi-coding-agent-forge：**扩展形态 web ui 实证**——pi-package-webui+remote-webui，/remote 开 LAN+PIN 认证+QR 手机接入 ⇒ 扩展起 server 接外部连接实践可行。
- ygncode/pi-web：Go+Svelte5，扫 ~/.pi/agent/sessions/，续聊/共享；hyperdreamer/pi-webui：进程级长驻（浏览器断开会话存活、多会话并行监督）。
- 官方第一方：pi-mono `packages/web-ui`（@earendil-works/pi-web-ui，npm 已发版）复用型聊天组件库（mini-lit+Tailwind），基于 pi-ai+agent-core。
- 成本：官方立场「自扩展无需 fork」（README）；SDK/扩展/外部语言三种架构均有第三方实现 ⇒ fork 非必要；**工时数据无公开来源**。

## ③ 语义巡检自动化先例与误判率（问题 2 事实子项）

- 工业 spec 一致性检查**全为结构/语法级**：Pact matching rules（regex/类型匹配）、oasdiff 514 条 OpenAPI 变更规则；语义级契约检查仅学术提案（IJISAE AI-Assisted API Contract Validation），无工业落地。
- LLM-as-judge 自一致性（误判率）已量化：重复同一评估成对偏好平均 **13.6% 翻转**（GPT-4o-mini/4.1-mini，29 任务；arXiv:2606.13685）；21 judges/118 runs：exact-match agreement 系统性高估判别力（arXiv:2606.19544）；SummEval 33–67% 文档存在偏好 3-cycle，被低聚合违规率 0.8–4.1% 掩盖（arXiv:2604.15302）；54 LLM 复现 human 评分用 Cohen's κ（arXiv:2510.09738）；前轮已知 κ≈0.46–0.60。
- 结论：「LLM 判两份文本语义一致」的**日常化自动巡检无工业先例**；单次判定不可靠（13.6% 翻转、κ<0.6），可靠化手段（N-of-M 投票/确定性参数）出自评测文献非生产实践 ⇒ 问题 2 期望的「可测判据先例」未出现。

## ④ 未消解项（R7）

1. 第三方 web ui 成熟度/维护活跃度未核（stars/commits）。
2. fork vs 自写扩展的工时成本无公开数据。
3. xiaoyc/pi-web-ui 快照推送细节未读源码核验（仅 pi.dev 摘要）。
4. arXiv 2606.19544/2604.15302 等系检索摘要级引用，未逐篇核对原文。
5. 「语义一致」可操作判据的定义属判断性，留 human 裁决。

---
*信源：①②=本地代码/文档原文+官方 repo（高置信）；③=论文摘要检索（中置信，编号未逐篇核）。*
