# research-extra-3 简报 — loop 定制化 + 判断力公式 + 赔率 vs 影响力

> 阶段：S 门禁第 3 轮临时调研（human 明确请求 loop_7）| 基准：2026-08-25 | 方法：本地源码考古（A1/B 自查）+ 3 叶子（A2A3/A4C/A5 联网+本地，见文末）| 前序：research-extra-2.md（从略不重复）
> 硬约束：仅事实+方法学+来源；不给推荐倾向；区分已验证/未消解。

## ① pi agent loop 源码考古（human 问「pi 的 loop 里有什么做了哪些事」）

pi 的 agent loop 分三层（本地已读全部相关源码，版本 0.84.2）：

**L1 纯循环（pi-agent-core/dist/agent-loop.js，552 行）**——「loop 内做了哪些事」逐项：

1. **双层 while 结构**：外层处理 follow-up 队列续跑（agent 本要停止时查 getFollowUpMessages）；内层处理工具调用与 steering 消息注入。[runLoop()]
2. **事件发射固定序列**：agent_start→turn_start→prompt 消息→每轮〔流式 assistant→工具执行×N→turn_end〕→agent_end。[agent-loop.js]
3. **LLM 流式调用**：transformContext 钩子（AgentMessage[]→AgentMessage[]）→ convertToLlm（→LLM Message[]）→ getApiKey 解析 → streamFunction 流式（text/thinking/toolcall delta 事件逐个透传）。[agent-loop.js streamAssistantResponse()]
4. **工具调用执行**：默认**并行**（Promise.all，executeToolCallsParallel）；当 config.toolExecution==="sequential" 或任一工具 executionMode==="sequential" 时改串行。**顺带消解 extra-2 ⑥-2：pi 有原生并行工具执行（已验证源码）**。[agent-loop.js executeToolCalls()]
5. **截断保护**：stopReason==="length"（输出被 token 上限切断）→ 该消息所有工具调用全部标记失败不执行（参数可能被截断，宁可报错让模型重发）。[failToolCallsFromTruncatedMessage()]
6. **终止条件（四条）**：① assistant stopReason error/aborted → 立即 agent_end；② 每轮后 shouldStopAfterTurn 钩子返回 true → agent_end；③ 本批全部工具结果 terminate===true → 终止内层循环；④ 无工具调用+无 pending steering+无 followUp → 退出外层。[runLoop()]
7. **轮间快照可变**：prepareNextTurn 钩子可替换下一轮的 context/model/thinkingLevel（模型热切换入口）。[runLoop() 尾部]
8. **钩子全表（loop 暴露给外部的全部接口）**：getSteeringMessages/getFollowUpMessages/transformContext/convertToLlm/getApiKey/beforeToolCall（可 block+terminate）/afterToolCall（可改 result/terminate/isError）/prepareNextTurn/shouldStopAfterTurn/tool.prepareArguments/streamFn 整体替换。[agent-loop.js+agent.js createLoopConfig()]

**L2 Agent 包装（pi-agent-core/dist/agent.js，421 行）**：单活动 run 保护（"Agent is already processing"）/AbortController/steering+followUp 队列/失败时合成 error assistant 消息走完整事件序列/事件状态归约（pendingToolCalls 集合）。[runWithLifecycle()/handleRunFailure()/processEvents()]

**L3 AgentSession（pi-coding-agent/dist/core/agent-session.js，2686 行）**：loop 外围 session 级编排——① **自动 compaction**：agent_end 后检查；overflow/可恢复截断→compact-and-retry 一次（_overflowRecoveryAttempted 防死循环）；阈值触发 shouldCompact()。② **自动重试**：可重试错误按 maxRetries 重试（auto_retry_start/end 事件，成功重置）。③ **扩展事件桥**：全部 loop 事件转发 ExtensionRunner。④ **agent_settled**：重试/compaction/followUp 均无剩余才算 idle——rick json mode 终止信号即此。[agent-session.js:396-600, 1530-1590]

**固有 vs 扩展边界（已验证）**：主循环结构（双层 while/事件序列/终止条件/并行策略）是**固有**，扩展不可替换；可定制面=上表钩子+扩展事件（tool_call/tool_result 拦截、before_provider_request/headers、自定义 compaction、session_before_switch/fork、registerTool/Command/Provider）。[docs/extensions.md 事件目录；loop 源码]

## ② 定制 loop 好处/代价 + 稳定内核架构学（本地已验证部分；业界案例见 leaf-1）

**pi/dsh 两侧的隔离边界（本地源码已验证，回应「loop 稳定是否= 不会出现 dsh 那种插件错误系统性崩溃」）**：

1. **pi 扩展异常不进主循环**：ExtensionRunner.emit() 对每个扩展 handler 独立 try/catch，异常转 emitError（含 extensionPath+stack）通知，主循环继续执行；扩展代码不在 loop 包内（pi-agent-core 零扩展依赖），仅经钩子/事件桥交互。[dist/core/extensions/runner.js emit()]
2. **pi 扩展加载失败不阻断启动**：loadExtensionsInternal 对单个扩展加载错误 push 进 errors 数组后 continue，其余扩展照常加载。[dist/core/extensions/loader.js:432-452]
3. **dsh 的 loop 是插件**（复确认本地）：architecture.md:11「产品的每一部分都是插件，包括…agent loop 本身，可从配置替换」；core/agent-loop 是实现 ctx.agentLoop 接口的默认驱动器。[docs/architecture.md:49；docs/capability-seams.md]
4. **事实推论（非推荐）**：pi 故障域=「handler 级隔离+加载级跳过」，loop 结构性 bug 理论上只来自 pi-agent-core 本身；dsh 故障域包含「插件树组合启动」（#4175 一坏插件阻 Web shell 挂载且卸载入口同树不可用）——隔离边界与 loop 是否可替换是两个正交维度，loop 稳定≠自动获得插件隔离，但 pi 实现恰两者兼备（1/2 条均源码已验证）。
5. **AI 自改进关注点分离（本地事实）**：pi 下自改进面=扩展（registerTool/Command/Provider/事件 handler，JS 可热重载 /reload），loop 结构不在改进面内，验证面=扩展自身单测+pi 稳定 API 契约；dsh 下 loop 亦在可改面内（配置替换 ctx.agentLoop），改进面更大验证面相应更大（自改进一般结论见 leaf-2：固定不变量+证明/对拍闸门+改扩展不改内核，把验证面压到差分面）。

**业界案例（leaf-1，联网已核）**：

6. **Loop 定制业界形态**：LangGraph=显式 StateGraph+条件边+checkpointer 中断恢复（代价：schema 变更破坏历史 checkpoint 兼容）；OpenHands=agent.step(State)→Action+append-only 事件流（痛点：模块级函数难覆写）；AutoGPT 弃用 legacy Plan-Act REPL loop 官方归因「缺抽象边界/难演进」，失败模式=无限循环+token 失控；smolagents/CrewAI 收敛于固定步循环+过程层编排。[docs.langchain.com；OpenHands issue#8025；AutoGPT ARCHITECTURE_NOTES.md]
7. **单插件阻塞启动被业界视为设计缺陷**：OSGi/Eclipse（与 dsh 插件树最同构）非可选 import 不满足时仅该 bundle 停 INSTALLED、框架照常启动+lazy 激活；Chromium 以多进程切故障域+插件独立进程；VS Code 扩展全走独立 Extension Host 进程+官方 Bisect；Firefox 停用进程内旧插件只留受限 API；Linux 模块异常只 taint 不阻断启动。[OSGi Core 8 §3.2.1.1；chromium.org；code.visualstudio.com；docs.kernel.org]
8. **微内核史实**：Mach3 实测内存系统性能显著差于单核 Ultrix（Chen&Bershad 1993）；L4 将 IPC 降至 45–121 cycles 证伪「微内核必慢」——可行性成立但工程代价真实。[CB93；Liedtke L4 HotOS-VI]

## ③ dsh 插件规模化风险（human 问「插件越来越多会不会系统性混乱/增删需耦合」；leaf-3 本地源码）

1. **声明与加载机制**：无单一 cordis.yml；插件树=四层有序 patch 组合（profile bundles 列表→profile cordis.patch.yml→home 级→--patch，后层按 row id 整体替换不深合并）；加载顺序非手动：插件声明 inject 所需服务、服务出现才激活（fiber 状态机 PENDING→LOADING→ACTIVE/FAILED）。[docs/architecture.md:15-27；vendor/cordis/src/fiber.ts:140-155]
2. **增删插件=profile/pnpm+patch 层操作**：dsh plugin 转发 pnpm，add 自动追加 bundle 层、remove 同步移除层+依赖；无 bundle 声明的包只装为普通依赖不激活。[docs/user/develop/basic/publish.md:77-110]——声明面薄（增删本身不需手改其他插件），耦合在运行时依赖而非声明。
3. **可逆 effect 边界**：注册类操作（工具 schema/prompt section/适配器/listener/服务）经 ctx.effect()/ctx.on() 的 disposer 逆序回滚；**不可回滚**：① session 事件日志（append-only）；② Cordis API 外副作用（写文件/起进程须插件自包 ctx.effect 否则泄漏）；③ 动态包内存态（重启即失）；④ profile 磁盘状态。[docs/cordis-primer.md:13,44；cordis-tutorial/02-lifecycle；packages/extensions/tool-cordis/README.md]
4. **系统性风险（human 问的直接答案，本地源码级证据）**：① **boot 是全树事务**：任一 entry 激活失败→整树 dispose 抛「plugin tree failed to load」，无单插件故障隔离（与 OSGi 的 bundle 级隔离形成对照）[packages/boot/app-boot/src/index.ts:757-800]；② **设置 UI 自身是插件**（inject 连接/远程服务），坏依赖可带走卸载入口本身——#4175 故障模式有源码同构证据 [packages/client/ui-settings-plugins/src/client/index.ts:51-53]；③ **级联卸载**：被依赖服务变化→inject 它的全部插件自动 UNLOAD→PENDING（依赖链传导非局部隔离）[vendor/cordis/src/fiber.ts _setEpoch]；④ **组合缺陷测试盲区**：postmortem 0001 同一插件多出口形态使 Loader 丢弃 inject，178 绿灯单测+100% 行覆盖未拦截、真实连接即崩 [docs/postmortem/0001]；⑤ HMR 用户补丁层有独立容错（坏 patch 保留最后好树广播失败事件），但这是配置层非插件代码层。[packages/boot/app-boot/README.md:45]

## ④ 判断力候选公式（human 问「直接给我一个可排序的公式+数据从哪来」）

**前置结构（拍板时须记四字段，公式共用输入）**：命题 M（可判对错）、概率 p（human 报 0–1）、截止日 D、影响力预估 I₀+可选关联 job 集。缺此四字段则公式退化为事后叙事（GJP 同构前提：概率+截止日+可解决命题）。[Mellers et al. 2014]

**候选 F1：字典序向量（与 human 拍板「正确性>赔率>影响力」直接兼容）**
- 记录三元组 V=(C, O, I)，排序规则：先比 C，C 同（容差 ε 内）比 O，再同比 I。
- C（正确性，proper score）：命题到期后按 y∈{0,1} 回收，C = 1 − (p−y)²（归一化 Brier，1 为全对）；或对数分 C = y·ln p + (1−y)·ln(1−p)（严 proper，对过度自信惩罚更陡）。Brier/对数分均为严格 proper scoring rule——诚实报 p 是最优策略。[Gneiting & Raftery 2007 JASA]
- O（赔率/非共识）：共识基准 q 取 N 个独立 subagent 概率中位数（N2 已拍板）；命中时 O=(1−q)/q（押注 1 单位的赔率回报，LMSR 市场定价同构）；未命中 O=0。[Hanson 2002 LMSR]
- I（影响力）：I=H+F；H=历史价值（关联 job 的 durationMs/成本/任务状态变化，meta.json 可派生）；F=未来期望值（需概率树 ΣP(oᵢ)·V(oᵢ)，唯一需建模项）。
- 排序性质：全序+传递；**无跨维补偿**——正确性不可用影响力买（Goodhart 鲁棒）；缺点：ε 容差是设计参数，ε→0 时微小概率差压倒一切；字典序不连续不可微，但作排序函数完备良定义。

**候选 F2：加权合成标量**
- V = w₁C + w₂O + w₃I，w₁>w₂>w₃（编码优先级），O/I 为归一化后值。连续、可补偿；风险：**Goodhart/博弈**——三项异量纲必须先归一化，归一化界本身是可被博弈的设计参数；可用低 C 换高 I。[Goodhart 1975；Strathern 1997]；F2 是 F1 的「软化」（补偿性换连续性）。

**候选 F3：分列记账（非合成公式，账本式）**
- 不合成标量：每条判断记 (M, p, q, y, D, H, F, t)；汇总层只做分列统计——校准曲线（分档 p vs 实际命中率，每档≥10 条）、分辨力、命中判断的赔率加权和 ΣO、影响力加权和 ΣI。排序=「先按校准带，再按赔率和」；无单一全序但每列独立可审计；GJP 实际排名即用单一 Brier——分列是学界默认形态。[Mellers et al. 2014]

**数据来源逐项映射（B2，全部本地已验证）**：

| 字段 | 来源 | 状态/成本 |
|---|---|---|
| M/p/D/I₀ | judgment.md 新增结构化字段 | **需新增**（human 拍板时写 4 字段，成本最低） |
| q 共识基准 | 派 N 个独立 subagent 报概率（runs.all 现成）+ human 审核（N2 已拍板） | **需新增流程**，技术现成 |
| y 回收 | dream 复盘按 D 检查命题（5-job 窗口已拍板 N4） | **需新增**（dream 模板加「判断回收」节） |
| H 历史价值 | meta.json（durationMs/cost/toolCount/usage.turns 已实测含全部字段）+ tasks.json 状态 + 关联 job 映射 | 机器侧✅可派生；判断→job 关联字段需新增 |
| F 未来估值 | 无现成数据源，需人工概率树建模 | 成本最高，可先置空只用 H |
| 汇总统计 | dream log（现成结构：反思发现/变更记录） | 模板扩展 |

**human 三维模型对齐**：「正确性归一化得分」=Brier/对数分；「赔率」=赔率回报 (1−q)/q（皆有成熟数学对应）；「三维加权平均」前轮 leaf-D 已论证偏离 proper 范式且激励虚报影响力——F1 字典序/F3 分列是与之兼容且数学上站得住的两种形态。

## ⑤ 赔率 vs 影响力（human 问「哪个更重要」；leaf-2 一手来源已核，正反并列）

**支持「赔率（非共识）优先」的论证**：

1. Marks second-level thinking：超额收益要求「非共识且正确」；Steinhardt variant perception 自述为「唯一管用的分析工具」（28 年费后年化≈24.5%）。[《The Most Important Thing》ch.1；《No Bull》2001]
2. **决策论精确对应 human「正确的废话」**：Grossman-Stiglitz（1980 AER）——已完全反映进价格（共识）的信息无人能从昂贵获取中获得回报，即共识性判断的**边际认知价值≈0**（注意：是边际认知价值，非总价值）。[Grossman & Stiglitz 1980 AER 70(3)]
3. 库恩/普朗克：认知增量存在于范式（共识）之外，范式内解题是常规科学；反共识新范式兑现以代际计（Planck 原则）。[《科学革命的结构》ch.IV/IX；Planck 1949]

**支持「影响力（规模/仓位）独立价值」的论证**：

4. **VoI 视角**：信息价值=对决策期望值的改进，「不改变决策的信息价值为零」——信息优势必须经可变动作（仓位/规模）才兑现；无影响力的正确反共识判断兑现不了价值。[Howard 1966 IEEE SSC-2(1)]
5. **预测市场形式化**：价格≈财富加权平均信念——个体的「影响力」（下注财富）就是其信念在共识中的权重；仓位同时是赔率的对手方与影响力的载体，两者在定价机制中不可分离。[Wolfers & Zitzewitz JEP 18(2) 2004]
6. Soros/Druckenmiller 转述（二手）：「关键不是对错，而是对时赚多少、错时亏多少」——收益=正确性×仓位大小。[《The Alchemy of Finance》1987]
7. 共识性大判断的价值形态=**执行价值**（范式内执行生产力、大仓位可执行低边际风险），与认知价值是两种形态——「正确的废话」低估了执行价值这一独立形态（leaf-2 综合推断，已标注为推断）。[leaf-2 #17]

**human 子判断等价论证的成立/失效条件（方法学分析）**：
- 成立前提：①「难而正确」确由某因果瓶颈子判断驱动且该子判断当时反共识；②价值形态是纯认知性。
- 失效面：①**分解不唯一**：哪个子判断是「关键的」是事后回溯归因，同一成功可分解出多个子判断集；②执行价值低估：若价值形态是执行（组织资源/坚持/规模），价值不在任何认知子判断中（如巴菲特式买好生意的成功更多在执行纪律与规模）；③**共识性时变**：赔率取决于度量时点（共识形成前 vs 后，预测市场早期仓位高赔率），等价论证隐含「共识性固定」假设。

## ⑥ 未消解项（R7）

1. 「未来 rick 会不会有定制 loop 需求」是前瞻问题非事实可消解：现有 8 定制点零命中 loop 替换（extra-2 ③），依赖 human Q1 决策后的架构走向。
2. dsh 插件规模化长期实证数据不存在：项目开源仅数周，只有机制级证据（全树事务/级联卸载/postmortem 0001）。
3. pi「坏扩展跳过继续」vs dsh「全树事务 fail-loud」是设计取舍，无第三方基准量化孰优。
4. leaf-2 部分表述为二手/推断（Soros 赚亏语、「执行价值」综合），已在叶子内标注。
5. F1/F2/F3 未在真实判断数据上试算（judgment.md 无结构化概率字段，N=6 循环）——可排序性成立，区分度待数据。
6. 赔率基准 q 的 subagent 群体概率有假独立性偏置面（模型同源/提示词同构），校准方法未定。

## 叶子文件

- leaf-1（loop 定制业界实践+稳定内核/插件隔离）：briefs/research-extra-3-leaf-1.md
- leaf-2（自改进系统+正确×非共识+预测市场/科学）：briefs/research-extra-3-leaf-2.md
- leaf-3（dsh 插件规模化本地源码）：briefs/research-extra-3-leaf-3.md
- ①②自查源：pi-agent-core/dist/{agent-loop,agent}.js、pi-coding-agent/dist/core/{agent-session,extensions/runner,extensions/loader}.js、dsh docs/{architecture,capability-seams,cordis-primer}.md、packages/boot/app-boot/；④数据源：.pi/subagents/artifacts/*_meta.json、judgment.md/tasks.json/dream log 实测。
