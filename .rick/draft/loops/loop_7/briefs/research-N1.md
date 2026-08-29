# research-N1 简报

阶段：N1·事实基础 | 主题：五要素刻画 rick | 基准 2026-08-27 | 视角：契约/制度

## ① node 清单（契约角色）

1. **human**：缔约方+巡检终审者。judgment.md 判断、门禁拍板、协议巡检终审（diff 报告+裁决+误判风险记录）。系统组件而非外部评测者（benchmark=human+agent）。
2. **rick agent 层（internal/ 七包）**：履约者。cmd（CLI：plan/doing/learning/dream/ctrl/easy/human-loop+init-pi/update-pi 等 tools）；handler；builder（PIBuilder：method 层 --append-system-prompt 注入+instance 层提示词文件）；prompt（templates+skills 共 30 文件）；env（agents.go 落盘自定义 agent；extensions.go 注册 pi-subagents/pi-web-access；PI_CODING_AGENT_DIR 隔离；update.go 版本管理）；runtime（Runtime 接口 Name/Run→Trace，唯一 Go seam）；workspace（dream 选 job）。
3. **pi runtime**：缔约对方。契约物=12 钩子封闭清单（源码考古全表见 research-extra-3.md：steering/context 转换/模型与轮次/工具拦截/终止/流式/并行策略共 12 面）+扩展事件；主循环结构（事件序列/终止组合/while 拓扑）固有不可替换。**pi 在系统内，其决策权（pi 社区）在系统外。**
4. **templates**：履约工具（方法层载体）。128 处 pi 触发方言（agent:'xxx'≈75、timeoutMs 27、19/30 文件）——最大耦合面，非中立层。
5. **sense loop**：parent 编排者+think/research/exporter 三 agent；5 阶段四段链+重试/回流；单写者文件隔离。
6. **dream/loop 机制**：履约监测（部分）。5 job 窗口+判断回收（X5）+dream 日志；未来承载协议巡检+度量（规划中）。
7. **human-loop**：判断记录器。judgment.md 逐条 human 原话。
8. **.rick/ 形态**：domain/（12 文件，pi-runtime.md=协议雏形）、draft/loops（loop_1-7）、jobs/（job_N/{plan,doing,learning}）、skills/（8 件）、dream/、RFC/。
9. **第三方扩展 pi-subagents/pi-web-access**：转包方。fanout 与 web 访问依赖；worker/reviewer/researcher 为其内置名。agent 实例（think/research/exporter 等）=履约分包者，由 env 落盘、templates 定义角色。

## ② input/output

输入：任务需求（job/loop）；human 判断（judgment.md/门禁/X3 反共识自答）；上游模型能力（deepseek 等，pi_extra_args 透传）；pi 版本更新（update-pi 拉入=单方改约）；社区动态（pi/npm 生态、dsh 观测、benchmark 环境监测项）。
输出：工程产物（jobs/{plan,doing}、代码）；judgment 记录；skill 沉淀（.rick/skills/）；domain 知识；dream 复盘日志；度量数据（规划中，已存在日志离线算、无需新埋点）。

## ③ inner/edge（契约权利义务）

- **human↔agent（主契约）**：human 义务=判断+终审+反共识基准；agent 义务=调研/教学/不替 human 决策。载体=human-loop judgment.md+dream 回收（X5）。
- **agent↔pi（执行契约）**：agent 权利=经 12 钩子+扩展事件干预；pi 义务=公开 API 稳定。载体=Runtime seam+templates 方言。
- **rick↔pi 版本（最脆弱边）**：pi 对 rick 无义务（单方改约自由；npm 包共享全局安装有卸载连坐风险）；rick 补救=协议清单巡检（规划中）+update-pi 版本控制。
- **parent↔三 agent（sense loop 内部）**：单写者文件隔离+timeoutMs 60min+fresh context+重试/回流上限。
- **dream↔jobs**：SelectPendingJobs 5 job 窗口扫描。
- **templates↔agent 实例**：模板定义角色/工具/超时。

## ④ 系统边界与外部环境

边界内：human+agent 层+templates+pi runtime 本体+.rick/ 全形态+第三方扩展行为。
**关键事实：边界切开契约——pi runtime 在内，pi 的意思机关（社区决策权）在外；契约漂移是结构性产物而非偶发事故。**
系统外：pi 社区（缔约对方意志主体）、模型厂商、npm 扩展作者、dsh（被排除、观察中）、benchmark 生态（环境监测）、美团内网网关（实测不可复用）。

## ⑤ 规划中组件（拍板未落地）

度量体系（三维向量正确率>赔率>影响力独立可微+统计值周/月环比+反共识基线 X3/Y5 并行冲突上报+四字段 X8）；协议清单巡检（domain 清单+dream 巡检 diff+human 终审+误判风险记录）；X6 实验前置；层 B 触发器≡Y1（12 钩子外骨架替换→全池重评：fork/自研/新 runtime）；web ui（pi 扩展、进程级 SDK/RPC）；度量数据流（dream 度量+主观评价）。

## 事实来源

internal/ 代码+.rick/ 实测；judgment.md 五轮拍板；12 钩子表溯 research-extra-3.md；128 处方言溯 research-S-r2.md。全部可溯源；规划中项已标注。
