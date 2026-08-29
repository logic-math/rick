# teach-S 教学简报

> 阶段：loop_7 S 问题确认（教学综合，第 1 次有效产出——上次派发因模型空响应未落盘，本次为重派）| 主题：深度学习 pi 与 dsh，判断谁适合作为 rick 的 runtime | 信息来源：research-S.md（第 1 轮事实基线）+ research-S-r2.md（第 2 轮消解）+ think-S.md + think-S-r2.md（最新轮）+ 本地代码引用 | 信源等级标注：[本地代码]=rick 仓库内可验证；[官方文档]=pi/dsh 官方发布物；[第三方]=博客/评测站，可信度需折扣

## 一、发生了什么

loop_6 时 human 已确认：单一 runtime = pi、三层金字塔架构落地、切换前提 = 更强生态 + 更强可定制性、dsh 列为未来深度定制候选。loop_7 重提「pi vs dsh 谁适合做 runtime」。两轮 research+think 之后，事实层面的收获是三件：①runtime 切换成本被量化了——templates 里 128 处 pi 专有触发语法、19/30 文件受染，「切换只改 Go seam、templates 不动」的旧承诺被证伪；②度量方案被证明可行——不需要写任何新埋点，5 类已存在的日志/产物文件就能离线算出触发率、延迟、成功率、成本，这直接针对 loop_6 自认的「无量化评估、靠直觉优化」短板；③dsh 的评测证据出了大问题——Terminal-Bench 2.1 官方 82.70% 与社区复测 0.51 相差两个数量级且未归因，以它作为 dsh 质量证据暂时不可用。think 层面：原 5 个隐含前提问题中唯一的架构级事实问题（模板耦合）已消解，剩下 4 个判断性问题 + 三连总追问 + 4 项未消解。现在所有能靠调研解决的事已解决，剩下的是 human 的判断。

## 二、这个领域的知识是什么样子

### 2.1 agent runtime 的职责本质：从第一性原理讲起

大模型本身是一个无状态的函数：文本进，文本出。它不会记住上一轮说了什么，不能执行命令，不能开子进程。把这样一个函数变成「能干活的 coding agent」，中间需要一个宿主程序持续做四件事，这个宿主就是 agent runtime：

1. **进程编排**：主循环（模型→工具→模型……何时停）、子代理派发（谁在何时被派去干什么、结果如何汇合）、后台任务。这是 rick 最依赖的一层——sense_loop 的 fanout（think/research/exporter 派发与汇合）全部发生在这一层。
2. **上下文管理**：上下文窗口是有限的，但工作跨越几十轮。何时压缩（compaction）、压缩时保留什么、摘要写回哪里——这些决定了模型「记得什么」。压缩策略的差异会直接改变长任务的行为。
3. **工具执行**：把模型的文本输出（tool call）变成真实副作用（bash、文件读写），再把结果以模型可读的形式送回去，包括失败处理。
4. **扩展机制**：宿主程序（rick 这样的上层框架）如何把自己的行为注入进去——注册工具、拦截调用、替换组件。

**「薄接口 vs 深耦合」的边界在哪里？** 直觉上，「runtime 是 rick 的一个可替换后端」听起来像换一个数据库驱动。但耦合不是只存在于 Go 接口那一层。rick 与 runtime 的实际接触面有四层：

- **Go 代码层**：`runtime.Runtime` 接口（Name/Run 两个方法），这是设计好的、真正薄的 seam。[本地代码 internal/runtime/runtime.go]
- **提示词模板层**：模板告诉模型「用 subagent({workflowScript:...}) 派发、agent:'think'、timeoutMs:3600000」——这是模型被教会说的一门**方言**，方言的语法属于谁，耦合就属于谁。
- **产物/数据层**：rick 事后要读回来的文件——`.pi/subagents/artifacts/` 目录布局、meta.json/transcript.jsonl 格式、`~/.rick/pi/agent/sessions/` 会话路径。
- **行为假设层**：模板与代码是否假设了 compaction 的具体语义（压缩何时发生、保留什么）。

「薄接口」这个说法只在第一层为真。第二、三层是否也薄，不取决于接口设计，取决于实际写了什么——这正是第 2.4 节要讲的核心。

### 2.2 pi 的架构事实：极简核心 + 生态补全

**设计哲学**。pi（Mario Zechner / badlogic，npm 包 @earendil-works/pi-coding-agent，MIT）自称「minimal terminal coding harness」，它的哲学是**核心刻意地小**：官方 README 明列「No MCP / No sub-agents / No permission popups / No plan mode / No built-in to-dos / No background bash」——这些能力全部交给扩展层构建，口号是「Adapt pi to your workflows...without having to fork and modify pi internals」。注意这个哲学的隐含交易：**核心小 = 侵扰少、可组合，但常用能力不在核心里 = 你要么自己写扩展、要么依赖第三方扩展**。[官方文档 README，本地 docs 已验证]

**扩展模型**。pi 提供的定制面四件：Extensions（TypeScript 模块，jiti 免编译加载；可 registerTool 自定义工具、registerCommand、拦截 tool_call 事件——可 block、可改参、可改结果、可自定义 compaction、可自定义 UI 渲染、可 appendEntry 持久化会话）、Skills（agentskills.io 标准）、Prompt Templates、Themes（仅 51 个颜色 token）。分发走 Pi Packages（npm/git）。[官方文档 docs/extensions.md]

**运行模式**：interactive / print(-p) / --mode json（JSONL 单向事件流，rick 在用的） / --mode rpc（双向 LF-JSONL，供非 Node 集成）/ SDK（createAgentSession 嵌入）。内置 30+ provider（含 DeepSeek、订阅登录、本地 llama.cpp）。[官方文档 README/SDK/rpc 文档]

**关键事实：pi 没有内建 subagent**。SDK 文档明示子会话由调用方自己建。rick 的 fanout 派发实际踩在一个**三层栈**上：pi 核心扩展 API（最底）→ 第三方扩展 pi-subagents（nicobailon 维护，提供 subagent 工具、workflowScript/runs.run 语法、worker/reviewer/researcher 内置 agent 名）→ rick 模板（最顶，内嵌这套语法）。所以 rick 的「深度定制」不是直接站在 pi 上，而是站在「pi + 第三方扩展」的复合地基上。[本地代码 internal/env/agents.go + 官方 README Philosophy 节]

**生态规模（2026-08-24 核实）**：建仓约 1 年（2025-08-09），GitHub 95,913★ / 11,864 forks；npm 周下载约 217 万（第三方口径 1.4-1.65M，口径未统一）；550 dependents；累计 254 个版本、发布频率约每周 1-3 版（rick 托管 0.84.2 = 最新版）；第三方 pi-package 约 4,300 个（pi.dev/packages 目录）。[GitHub badlogic/pi-mono；npmjs.com；pi.dev——均为公开可复核的第三方/官方页面]

**bus factor = 1 意味着什么**。badlogic 一人占 69% commits；新贡献者的 issue/PR 默认自动关闭、由维护者每日复查。bus factor 是「项目被一辆公交车带走的风险」——单点中断风险。要精确理解它的含义：它**不**说明项目现在不健康（周更 254 版、生态 4300 包是活跃的反证），它说的是**未来不连续性的形状**——如果这个人停更，没有第二个人能同等速度接手。rick 已经实测过的限制也记录在案：pi 不读 PI_* 环境变量（必须 CLI flags）、渲染行为不可主题化（改行为须改 dist JS，rick 已否决 patch 路线）、strict 工具校验曾在 0.51.0 硬失败。[本地 .rick/domain/pi-runtime.md + inspect.software 社区统计]

### 2.3 dsh 的架构事实：一切皆插件 + 未成形的契约

**身份**。dsh = DeepSeek Harness，DeepSeek AI 官方开源（github.com/deepseek-ai/deepseek-harness），2026-08-13 与 DeepSeek V4-Pro 同日发布，MIT，TypeScript/Node（要求 Node ^22.19 或 >=24）。[官方仓库元数据 + deepseek.com/harness]

**「Everything is a Plugin」不是修辞，是结构**。官方架构文档原话：「There is no privileged core to patch: you extend dsh by mounting a plugin」——model adapter、tool registry、session log、**agent loop 本身**都是插件，都可以从配置整体替换。插件是 ESM named-export 的 Service 对象（apply(ctx)），贡献 services / typed events / reversible effects（ctx.effect() 注册的资源随插件卸载自动回滚）；cordis.yml 声明插件清单，boot 时按 profile/bundle/patch 分层组合成「插件树」。内核 Cordis 是 vendored 进 monorepo的自持框架（设计论文《A Programming Paradigm for Spatiotemporal Composability》），管理挂载/卸载/依赖。[官方文档 deepseek-harness.github.io/en/reference/]

**与 pi 的结构差异，一句话说清**：pi = 固定核心 + 扩展**加**能力（扩展能拦截、能增加，不能替换核心 loop）；dsh = 插件组合**即**产品（核心本身也是插件，可从配置换掉）。前者是「钩子」模型，后者是「组合」模型。组合模型的理论上限更高——你可以换掉 loop 本身——但代价是：**没有一个稳定的「核」可以作为兼容性承诺的锚点**。

**能力词汇比 pi 核心广**：subagent（ctx.subagents 多后端：inprocess/acp/codex/claude-code，传输可换）、workflow 引擎（goal rounds + Fresh-agent Ralph）、user approval（fail-closed）、sandbox（bwrap/Landlock/Seatbelt）、MCP client、planning（goals/jobs/todo_write）、JSONL 持久化。这些在 pi 那里要么没有、要么靠第三方扩展。[官方 reference/subsystems + dsh-in-depth.com 第三方站]

**developer preview 阶段在工程上意味着什么**。官方声明原文：「currently in developer preview and iterating rapidly. THERE WILL BE COMPATIBILITY-BREAKING CHANGES」。版本史：全部为 pre-release，v0.1.1-rc.2（08-21）之前是 rc.5→rc.7→rc.8→rc.1 的快迭代，npm latest 仍指 0.1.0-rc.7 而 rc.8 在 next tag 且含 breaking change。已知限制清单包括：session 分支树推迟、fork() 仅限活跃会话、ACP 仅 fresh session、subagent SDK 每次运行新建进程（无池化）、沙箱后端不可用即 fail-closed、Node<22.19 出过含混 ESM 崩溃、Web UI 多轮会话丢 reasoning blocks。这些的**工程含义**：dsh 的 API 现在还不是一份契约，每次 breaking change 的成本会直接砸在基于它构建的集成层（也就是 rick 未来的 dshRuntime + 模板方言）上。[官方仓库 packages/*/README + GitHub Discussions]

**TB2.1 冲突说明了评测的什么问题**。Terminal-Bench 2.1 榜上 DeepSeek 条目 82.70%（V4 Flash、dsh minimal mode、max effort）[tbench.ai，经 phaseo.app 镜像确认]；但社区 Discussion #1107 用 Harbor 跑全量 89 task 只得 **0.51**（对照 Clawcodex 0.75），且 #1089 指出官方未附 artifacts。82.70% 与 0.51 相差两个数量级。这件事的教学价值在于拆开看：一个 benchmark 数字从来不是「dsh 的能力」，而是 f(harness, model, 配置, 任务子集, effort, 环境) 的一个点——官方条目用了 minimal mode + max effort + 特定模型，社区复测是另一组参数，**两者甚至不构成同一实验**；官方自报数字无 artifacts 时不可复核；社区复测也未归因差异来源。旁证：dsh 自托管 web 的 TTFT 约 10s、成熟会话 median 49.7s / p90 142s（#3235）、zero prompt-cache hits + 每轮 6.6K token 的 schema 开销（#3304）；Composio Golden Eval（8 harness×30 task×V4 Flash）里 dsh 因发布晚 2 天未入选，Pi Agent 以 66.7% 最高且 $0.028/task 最便宜（注意是 pi 系，且 66.7% 与 Oh My Pi 硬 fork 的 88% 都不是 pi 本体数据）；阿里云 AgentLoop 在 TB2.1 十任务子集上 dsh 与 Codex 持平。**底线事实：dsh vs pi 的直接受控对比至今不存在**。[tbench.ai/Discussion #1107 #1089 #3235 #3304/composio.dev/segmentfault/atlascloud.ai——第三方评测，可信度需折扣]

### 2.4 模板耦合面：为什么 128 处触发语法意味着 templates 不是中立层

回到 2.1 的四层接触面。第 2 轮 research 对 internal/prompt/templates/（10 主模板 + 20 skills）做了逐文件盘点，结论是「templates 不改」的承诺不成立。先把事实摆出来：

- **写侧（触发语言）**：显式 pi 触发语法关键字 **128 处、遍布 19/30 个文件**，其中 11 个文件含可执行 JS。最重的是 sense_loop.md——18 处，且 L26-60 是一个独立章节「显式触发语法（pi subagent 工具）」；plan.md L139 有八-reviewer 并行派发的 JS。具体形态：`agent:'xxx'` 约 75 处、`timeoutMs` 27 处。agent 名有两类：rick 自有的 think/research/exporter，**以及 pi-subagents 内置的 worker/reviewer/researcher**（agents.ts:38 的 BUILTIN_AGENT_NAMES）——也就是说模板里的词汇有一半来自第三方扩展的词表。
- **读侧（产物格式）**：`.pi/subagents/artifacts/` 布局 + meta.json/transcript.jsonl 格式，约 11 处、4 个文件（ctrl×6、dream×2、doing×1、learning×1）；doing.md L14 还有 `{action:"status",view:"transcript"}` 管理语法。
- **路径侧**：ctrl.md 两处硬编码 `~/.rick/pi/agent/sessions/--<cwd>--/<ts>_<sessionId>.jsonl`。
- **行为假设侧（唯一的缓解项）**：compaction/上下文行为假设 **0 处**——sense_loop.md 里的 "compact" 实为 rick 自有术语「compact contract」，全部模板不假设 pi 的压缩语义。[本地代码 internal/prompt/templates/ 逐文件盘点，research-S-r2 ①]

**为什么这让 templates 不再是中立层？** 因为模板不是「写给模型的散文」，而是**宿主教给模型的 runtime 方言教科书**。loop_6 之前，模板用自然语言描述派发（「派一个研究员去查 X」）——可移植但不可靠，模型可能不触发；loop_6 的修复正是把自然语言换成显式语法（subagent({workflowScript:...}) + runs.run + 真实 agent 名），**用可移植性换确定性**。这笔交易在当时是对症的（触发概率是当时的病），但它把 pi-subagents 的词表浇筑进了 30 个文件里的 19 个。于是切换成本的真实结构变成三层：Go seam（小、设计好的）+ templates 方言（大、逐轮累积的）+ 产物读回格式（中）。而且这个成本是**复利**的——每一轮新模板、每一次 agent 名引用，都在加深它。

**但要精确理解这个发现的边界**：它量化的是「切换成本的下限结构」，不是「切换不可行」。修订方向 research 已给出两条：templates 分 runtime 版本（pi 版/dsh 版并存，维护两套方言），或 dsh 侧提供等价 subagent API + 兼容产物布局（把兼容负担推给 dsh 侧的适配层）。另外 compaction 假设为 0 是重要缓解——耦合不涉及上下文管理语义，也就是说 rick 对 pi 的依赖集中在「怎么派活」和「怎么读结果」，不涉及「模型怎么记忆」，后者恰恰是 runtime 之间差异最大、最难适配的部分。

### 2.5 度量问题：为什么「无量化、靠直觉」是当前判断的根本短板

loop_6 的 judgment 原话：「目前确实缺少量化评估，当前都是靠直觉进行优化」。为什么这 Matters 到「根本短板」的程度？因为本轮的三个问题——loop_6 修复有没有效、短期性能瓶颈在哪、换 runtime 能否改善——**每一个的答案都依赖同一个前提：知道现状的数字**。没有基线，「性能仍需提升」无法与「修复已生效但感知滞后」区分；没有基线，任何 runtime 对比都是无对照的。优化没有度量，在统计意义上是随机游走：你可能好起来了，也可能只是运气。

**为什么 5 类现成数据源就能解**（关键在于它们已经存在，不需要写一行新埋点）：

1. `.pi/subagents/artifacts/*_meta.json`——rick 仓库里已有 296 个文件 = 74 个 runs（worker 44 + reviewer 30），字段含 runId/agent/task 全文/exitCode/usage{cost,turns}/model/durationMs/toolCount。这是每个子代理运行的一手账本。
2. `~/.rick/pi/agent/run-history.jsonl`——agent/taskHash/ts/status/duration（task 已脱敏）。
3. 父会话 JSONL——message 事件含 toolName=subagent/subagent_wait + 时间戳；实测 loop_7 父会话可 join 出 2+1 次调用。
4. `raw_session_coding.log`（pi --mode json 输出）——turn/tool_execution/agent_settled 事件带时间戳。
5. tasks.json（status/attempts）——任务级成败与重试。

由此可算：**触发率** = 预期派发数（briefs/*.md 存在数）÷ 实际派发数（父会话 subagent 计数）；**启动率** = 调用数 vs artifacts run 数；**延迟** = durationMs 分桶 P50/P95；**成功率/成本** = exitCode / usage.cost。覆盖 doing→worker、plan/dream→reviewer、sense_loop→think/research/exporter 全部路径。[本地代码/数据实测，research-S-r2 ②]

**四个限制要如实知道**：run-history 的 task 字段已脱敏、难以归因到具体任务；session_id join 键只有 loop_7 有，历史会话得靠 meta.json 里的 task 文本匹配（task 含路径，可行但费工）；**「预期派发数」没有机器可读定义**——这是度量落地的唯一工程前置（想想看：什么算「本应派发」？得先有一份预期清单的规范）；Trace 目前仅 stdout 未落盘。换句话说：方案可行 ≠ 方案已运行，从「可算」到「算出来」之间还差一个脚本和一份预期派发定义。

### 2.6 常见误区：三个最容易带偏判断的坑

**误区一：把 star 数当采用指标**。dsh 11 天 189,495★，pi 一年 95,913★——如果 star 是采用，dsh 已赢两倍。但 star 度量的是**注意力**，不是**采用**。dsh 的 star 来自「与 V4-Pro 同日官方发布」的发布事件注意力高峰；VentureBeat 自己都注明「数字只是快照、非采用指标」。真正能当采用代理的指标是：npm 周下载（pi 约 217 万，第三方口径 1.4-1.65M）、dependents 数（550）、第三方包存量（约 4,300 个 pi-package）——这些是「有人在真的用并在上面构建」的痕迹。dsh 侧这三个口径的数据：**未发现第三方插件目录或规模统计**，open issues=0（11 天新仓库的正常状态，不是质量信号）。star 是注意力的先行指标、生产采用的后行（甚至缺席）指标——两者在本案例里方向完全相反。[GitHub/npm/inspect.software 公开数据；0xran.com/VentureBeat 第三方评述]

**误区二：把评测分数当能力属性**。2.3 节已拆过 TB2.1 的 82.70% vs 0.51。这里补一条通用规则：**harness benchmark 的分数属于「harness×model×配置」的组合，不属于 harness 单独**。同一 harness 换 model 分数天差地别；自报数字无 artifacts 不可复核；单任务轶事（dsh vs Hermes 的 121s vs 780s）样本量为一；「Oh My Pi 88% 第一」是 pi 的硬 fork，不是 pi 本体。唯一能支撑「A 优于 B」的证据形态是：同 harness 组、同 model、同任务集、受控配置、附 artifacts 的对比——而 dsh vs pi 的这种对比**不存在**。在它出现之前，任何「dsh 跑分更强/pi 跑分更强」的叙述都是在用不构成对照的数据做对照结论。[tbench.ai/composio.dev/Discussions——第三方评测，可信度需折扣]

**误区三：把成熟度风险当静态属性，或干脆不当风险**。两个方向都错。dsh 的 developer preview 是**时间变量**——它今天 0.1.x-rc、官方明示将有 breaking changes，但这不是永恒属性，1.0 会不会来、何时来才是问题；pi 的 bus factor=1 也是时间变量——今天周更 254 版很健康，但单点不连续的形状不会因健康而消失。这两类风险的**性质不同**而非仅大小不同：pi 的风险是「不连续性」（maintainer 停更则断崖），dsh 的风险是「churn」（持续迭代、持续破坏兼容）。把「dsh 还在 preview」当永久否定或当无关紧要，都忽略了它们是会随时间演化的风险形状，而 human 的判断需要给它们各自一个权重与观察触发器（例如：dsh 出 1.0？pi 出现第二梯队维护者？）。另外还有一条常被漏掉的：rick 踩的是「pi + 第三方 pi-subagents」复合栈，bus factor 风险实际是**两层**的——pi 核心一层、pi-subagents（nicobailon 个人维护）一层。[官方仓库声明 + inspect.software + 本地代码 internal/env/agents.go]

## 三、启发式追问

以下追问建立在上面已讲清的知识之上。每条附「改变判断的证据」——不是要你现在回答，而是标明什么证据出现时这个问题应该被重新掂量。

**Q1（重开的触发条件）**：如果「loop_7 现在就要判断谁适合做 runtime」成立，那么也假设了「loop_6 之后出现了足以重开已决事项的新条件」。但两轮调研的事实是：dsh 发布（08-13）早于 loop_6 判断（08-14），其后仅 rc 常规迭代；TB2.1 数据本身可信度存疑（2.3 节）；唯一的新量化（模板耦合 128 处）实际上是把「维持现状的切换成本下限」标得更清楚了，而非提供了切换的理由。那么，本轮重提的真实诉求更接近「用新事实校验既有决定的边界」，还是「预设了要重选、在找理由」？这两种诉求需要的证据强度完全不同。
改变判断的证据：dsh 发 1.0 稳定版 + 插件市场成形；pi 停更或 bus factor 兑现；TB2.1 冲突归因后 dsh 显著领先。

**Q2（「短期性能」的指代与杠杆）**：如果「对比 pi/dsh 能回答短期如何提升性能」成立，那么也假设了「短期性能有明确指代，且 runtime 选择是该维度的有效杠杆」。但 2.5 节表明：度量基线可以纯离线算出却尚未运行——在基线出来之前，「瓶颈在 runtime 层」与「瓶颈在提示词/流程层」无法区分；而 loop_6 已确认的短期抓手（提示词/配置对齐）已落地验收。那么在跑出基线之前，「性能」这个词在本轮语境里指什么——触发确定性？上下文效率？延迟？成本？——由谁、依据什么来定义？
改变判断的证据：R7-4 度量脚本运行后显示瓶颈确实在 runtime 层（如触发不确定性的根因被定位到 pi-subagents）；或出现直接受控对比且 dsh 显著优于 pi。

**Q3（长期架构绑定 vs spec 可切换赌注）**：如果「长期架构发展取决于选谁做 runtime」成立，那么也假设了「loop_6 押注的 spec 可切换性（自然语言方法描述→任意等价实现）不足以消解 runtime 锁定风险」。注意两侧证据形态的不对称：切换成本侧已被量化（128 处方言 + 产物格式 + 路径，且逐轮复利）；可消解侧（spec 赌注成立则切换长期廉价可逆）至今无任何实验验证。这个不对称本身就是一个信息：你现在为一件事拥有数字、为另一件事只有信念。若 spec 赌注成立，runtime 选择长期廉价可逆、本轮对比 stakes 大降；若不成立，每轮模板演化都在加深绑定。两方向不能同时为真——你押哪边？
改变判断的证据：一次小规模 spec→等价重实现实验（用 spec 重建 rick 某模块并过功能验收）成功或失败。

**Q4（评判标准集的完备性）**：如果「按生态 + 可定制性两条标准对比即可得出结论」成立，那么也假设了「这两条构成的集合是完备的」。但 2.6 节展示了至少两个不在集合内的维度：成熟度/stability（dsh 官方明示 breaking changes）与维护者风险（pi bus factor=1，且 rick 踩的是双层栈：pi 核心 + pi-subagents 第三方）。这两个维度的风险性质不同（churn vs 断崖），且都有一票否决的潜能。那么你的标准集应该有几条？每条的真实权重是多少——生态存量（pi 现在领先）与生态增速（dsh 注意力领先）又是否该算同一条「生态」？
改变判断的证据：dsh 发 1.0 且 pi 形成第二梯队核心维护者；TB2.1 冲突归因完成。

**S 三连（协议规定的总追问）**：

**① 现状中最不能忽视的事实是什么，为什么？** 两轮调研摆出了两类候选事实：一类关于外部（dsh 的组合式架构理论上限更高、但 preview 阶段 API 无契约；pi 生态存量厚、但单点维护）；一类关于内部（rick 自身效果无度量——loop_6 修复是否生效未知，「性能仍需提升」目前只来自直觉，2.5 节证明基线可离线算出而未算）。哪一类才是「最不能忽视」的？如果内部度量缺位是主事实，那么本轮最短缺的交付物是 runtime 对比还是度量脚本运行？

**② 如果期望达成，你看到的世界与现在有什么不同？** 把两种可能的期望各描述一遍，对比它们的差异：若期望是（A）重选 runtime——你会看到 templates 分 runtime 版本、dshRuntime 三件套落地、但每轮要消化 dsh 的 breaking changes，且选择依据仍无受控对比支撑；若期望是（B）校验维持 pi + 定义未来切换触发器——你会看到一条明确的触发器清单（dsh 1.0？插件市场成形？pi 停更？TB2.1 归因？），度量基线在跑，templates 继续在 pi 方言上复利。这两个世界对「今天要做什么」的要求不同，对证据强度的要求也不同。

**③ 现状与期望之间真正的阻碍是什么（不是表面差距）？** 表面差距是「不知道 pi 和 dsh 谁更好」。但 2.3 节已表明这个差距在受控对比出现前无法靠调研消解；2.5 节表明另一个差距——rick 自身效果的度量差距——既有方案又有数据、只差一次运行。如果真正的阻碍是后者，那么深度学习 dsh 并不能消解它；如果是前者，那么正确的下一步是 R7-5 spike（dsh headless 实测）而非更多文档调研。真正的阻碍是哪一个？

## 四、延伸学习指导

若你希望在回答追问前系统性补课，以下材料按「读完你能判断什么」挑选：

1. **pi 官方文档三件**：`docs/extensions.md`（扩展 API 全貌）、`docs/sdk.md` + `docs/rpc.md`（嵌入式集成模式）。为什么读：2.2 节的「定制面四件」是我转述的，读完你能亲自判断 pi 的钩子模型天花板——哪些行为你能改、哪些永远改不了（这直接决定 Q4 中「可定制性」这条标准对 pi 打几分）。
2. **dsh 官方架构参考**：deepseek-harness.github.io/en/reference/（「Everything is a Plugin」章节）+ deepseek-harness.github.io/en/develop/（插件 API）。为什么读：读完你能判断「agent loop 可配置替换」在 API 层面长什么样、替换 loop 的实际工程量，而不是停留在架构图印象（这决定 Q4 中「可定制性」对 dsh 打几分，以及 R7-5 spike 该测什么）。
3. **Cordis 设计论文**：《A Programming Paradigm for Spatiotemporal Composability》（dsh 内核的理论来源）。为什么读：dsh 的「无特权核心」承诺的理论边界在哪里、reversible effects 到底保证什么——读完你能判断 dsh 插件模型的健壮性是结构保证还是宣传口径。
4. **耦合度量的经典框架**：Yourdon & Constantine《Structured Design》中 coupling/cohesion 章节（模块间耦合的分类：数据耦合/控制耦合/内容耦合），或 Robert Martin 对稳定依赖原则（SDP）的论述。为什么读：2.4 节的「128 处 = 耦合面」是现象，经典框架给你语言和度量方法——读完你能自己判断 rick 对 pi-subagents 词表的依赖属于哪类耦合、严重度如何，而不依赖我的定性。
5. **评测方法论**：Terminal-Bench 的官方报告（tbench.ai）+ 本次案例本身（GitHub Discussion #1107/#1089 的复现争议全帖）。为什么读：这是活教材——读完你能建立「一个 benchmark 数字何时可用作证据」的检查清单（有无 artifacts、配置是否披露、任务集是否全量、有无独立复现），用于未来一切 harness 评测。
6. **度量方法论**：Douglas Hubbard《How to Measure Anything》第 1-3 章。为什么读：loop_6 的「无量化靠直觉」正是此书定义的问题——它论证「测量 = 不确定度的减少」而非「精确数字」，并给出用廉价既有观察替代昂贵埋点的方法论；rick 的 5 类现成数据源就是这种方法论的一次无意识实践，读完你能判断 R7-4 脚本的设计是否还漏了更便宜的观察点。

## 五、R7 未消解项上报

两轮调研后仍未达高置信度的事实项，逐条标注性质，供知情决策：

1. **dsh vs pi 直接受控对比不存在**——未消解（实验性：须受控实验，非调研可消解；子项「pi 是否有 TB2.1 条目」可再查但对比本身须实验）。
2. **TB2.1 官方 82.70% vs 社区 0.51 未归因**——未消解（实验性：须官方补 artifacts 或第三方复现归因；在此之前以 TB2.1 为 dsh 质量证据不可用）。
3. **R7-4「预期派发数」无机器可读定义**——未消解（工程性：度量脚本落地的唯一前置；属工程任务非调研）。
4. **workflowScript→dsh ctx.subagents 等价性未实测**——未消解（实验性：即 R7-5 spike，Q5 模板耦合结论的 dsh 侧验证）。

另附 think 轮对旧 R7 清单的处置建议（非事实，供参考）：R7-1/R7-2（dsh 第三方插件规模、star 含金量）建议挂起 30 天后复查——11 天项目一次性调研无解；R7-3/R7-7/R7-8（Discord 人数、npm 口径、commit 拆分）对本轮各题均无影响，建议放弃。

---

*本简报仅综合事实与讲解知识，不含对「选 pi 或选 dsh」的倾向性结论；所有判断性问题的答案由 human 给出。*
