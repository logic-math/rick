# teach-gate-S-r2 教学简报

> 阶段:loop_7 S 门禁第 2 轮(教学综合)|主题:pi vs dsh 谁适合作为 rick 的 runtime(结合 rick 未来发展方向)|信息来源:research-extra-2.md(+leaf-A/D/E)1 轮 + think-gate-S-r2.md 1 轮;前轮 teach-gate-S.md(D1-D11)已讲过的从略引用|信源等级:[本地代码]=仓库内可验证;[arXiv/DOI]=编号可查论文;[官方]=厂商发布物(README/docs/repo);[第三方]=社区站/博客,可信度需折扣|事实与观点分开:带来源标注的是事实与领域知识,判断项全部归入第五节由 human 拍板

## 一、发生了什么

**human 对 teach-gate-S(D1-D11)的拍板构成四个转变**(judgment 转述):

1. **Q1 接受 spec 降级**:有限穷举+语义契约——前轮教学标记为「最危险认知支点」的「spec 可穷举验证」被 human 主动收敛,D2 就此关闭。
2. **Q2 从「双 runtime 都要」转向「做出取舍」**:runtime 决策的第二次反转(loop_6 单 pi → S 门禁双 runtime → 本轮取舍),human 同时请求对 pi/dsh 做细致调研。
3. **Q3 提出度量体系**:human 判断力三维模型(正确性/赔率/影响力加权)+ agent 四指标(主观评价/工具出错与 debug 调用与异常纠正耗时/工时/验收质量)——度量对象从 agent 侧扩展到 human 自身的判断力。
4. **Q5 拒绝现成 benchmark**:benchmark=human+agent 个性化评价体系;行动请求=完成度量研究后裁决 DSH vs PI。

**门禁 r2 的发现**(think-gate-S-r2):行动请求把「度量设计→runtime 裁决」串成前提链,链上有五个未经检验的隐含假设(top-5:因果链单调性/判断归因无对照/反共识无共识基线/样本量与数据可得性/度量判别力可能为零);决策表核对=D1/D2/D8/D9 已覆盖,D3/D7/D10 悬置,D4/D5 条件失效,D6 部分覆盖;另指出三维模型是投资框架向 n=1 决策的迁移、且维度非独立。

**调研的顶线事实**(research-extra-2):dsh 独有能力实为三层(3 项结构性差异/7 项 pi 靠第三方部分做到/4 项程度差异/1 项无实质差异);rick 今天 8 个定制点中 6 个落在 pi 核心、2 个在第三方扩展,dsh 的差异化上限「替换 agent loop」在 rick 现有+已规划需求中零命中;判断力度量七种成熟方法与 N=1 纵向统计工具均就绪,但全部以「拍板结构化」为共同前置——这属 rick 层工程,runtime 无关。

## 二、这个领域的知识是什么样子

### 2.1 「灵活→掌控→性能」因果链:控制收益的真实形状

**这条因果链的结构。**「dsh 更灵活(可定制)→对 LLM 行为掌控更绝对→产出性能更好→更适合 rick」——这条链每一环都有独立的领域知识要讲透,混在一起就会变成「更灵活=更好」的口号。

**第一环:灵活性到底差在哪(精确化「更灵活」)。** pi 的扩展面=registerTool/registerCommand/事件拦截(tool_call 可 block/改参/改结果)/自定义 compaction(全量可换)/自定义渲染,边界=主循环与默认骨架不可替换(README 以「without having to fork」划界)。[P:packages/coding-agent/README.md] dsh 的 agent loop 本身是插件、可从配置替换。[D:docs/architecture.md] 所以「dsh 更灵活」精确指:pi 的扩展面 + 1 个 dsh 独有的自由度(loop 替换)+ 第一方 vs 第三方的生态差异(2.2 详述)——不是一个均匀的「灵活度」标量。

**第二环:掌控与产出的关系——控制收益非单调。**「对 LLM 更绝对掌控→产出更好」假设控制收益单调递增。人类参与度的对照实验给出形状反例:HAS-Bench(397 任务,5 级 agency×3 通道)中,A3(平等伙伴)比全自动 A1:Pass@1 +8.4、安全率 +26.9、可恢复 65.4% 自主失败;但 A3→A4(更高参与)边际递减(救 27 破 13),且弱模型上人类参与≈0 增益。[arXiv 2607.04329] 这条数据讲的是 human 参与,但机制可迁移:控制(无论来自 human 还是 runtime 配置)的收益有饱和点,且取决于被控对象(模型)利用控制的能力。

**为什么控制收益会饱和(第一性原理)。** 控制的作用机制=把不确定的行为空间约束到期望子空间。**已经到位的约束,再加控制的边际收益≈0**。rick 的控制点(templates 提示词/gates 确定性脚本/spec 契约)本来就架在 runtime 之上,经扩展 API 注入;「替换 agent loop 本身」新增的自由度,只有在「现有注入点无法表达期望约束」时才有价值——而 rick 的 8 个在用定制点(2.3)没有一个撞上这个条件。控制还有反例成本:配置面越大故障面越大——dsh 的「插件树整体不可启动」故障(2.3)就是灵活性自身引入的失败模式,不是外部强加的。

**第三环:性能证据的现状。** 直接对比不存在:无「同 model 同任务集 pi vs dsh」受控数据。dsh 侧有自身绝对数:自托管 web 在 fan-out 下成熟会话 TTFT 中位 49.7s/p90 142s(Discussion #3235);zero prompt-cache+6.6K token schema 开销使每轮 3.5s→53s(#3304)。[D Discussion #3235/#3304] 换言之:因果链第三环目前既无正向证据也无反向证据,只有 dsh 自身的已知性能债。

**需求清单对照法(本轮调研的方法论核心)。**「谁更灵活」是抽象争论;「我的需求落在谁的 API 面上」是可逐项核对的问题。方法:列出 runtime 相关全部定制点→逐项标注落在 pi 核心/pi 第三方/dsh 核心/dsh 插件→统计命中与缺口。**零命中 loop 替换的含义**:dsh 相对 pi 的结构性差异化上限(loop 可替换)在 rick 现有+已规划需求中没有对应需求。这不是说该能力无用——它是**期权**:若未来出现「必须改 loop/session log」的需求,pi 只剩 fork 一条路,dsh 有配置层入口。期权的价值=未来需求概率×命中时收益−持有限制性配置的持续成本;定价是判断项(第五节 Q1)。

### 2.2 dsh 独有能力三层清单:结构性差异 vs 生态位差异 vs 程度差异 vs 无差异

**为什么要分层。**「dsh 能做 pi 做不到的事」这句话把四种性质不同的差异混在一起了。本轮调研把它们拆开(全部带来源):

**第①层:结构性差异——pi 无论装什么扩展都做不到(除非 fork)。** 共 3 项:

1. **替换 agent loop 本身**:dsh 的 loop 是插件,可从配置替换;pi 扩展 API 无主循环替换接口。[D:docs/architecture.md;P:packages/coding-agent/README.md]
2. **跨产品子代理后端**:dsh 第一方多后端共存——inprocess spawn/fork、ACP、Codex(app-server --stdio)、Claude Code,可把 Claude Code/Codex 当子代理跑;pi 生态的 pi-subagents 子代理全是 pi 子会话,已知生态无跨产品子后端扩展(pi-acp 方向相反:pi 被编辑器驱动)。[D:packages/subagent/README.md+feature note 2026-08-04;github.com/nicobailon/pi-subagents;github.com/svkozak/pi-acp]
3. **三平台内核沙箱+fail-closed 语义**:Linux bwrap→Landlock 探测链/macOS Seatbelt/Windows ACL restricted-token;沙箱不可用即拒绝执行,「never silently falls through unconfined」;pi 生态沙箱仅 macOS/Linux,且存在 fail-open 实现(@jerryan/pi-bash-wrap 非 Linux 平台原样放行)。[D:packages/sandbox/sandbox-local/README.md;npmjs @jerryan/pi-bash-wrap]

**fail-closed vs fail-open(安全工程基础概念,值得单独讲)。** 二者描述系统在「防护机制本身失效」时的默认行为:fail-closed=默认拒绝(牺牲可用性保安全——沙箱坏了就不跑);fail-open=默认放行(保可用性丢安全——沙箱坏了照跑不误)。对能执行任意 shell 的 agent runtime,这是实质语义差异:问题不是「有没有沙箱」,而是「沙箱坏了会发生什么」。

**第②层:生态位差异——pi 靠第三方扩展可部分做到,附限制。** 共 7 项:①OS 沙箱(@nqbao/pi-sandbox、rnorth/sandboxed-pi、官方示例;限制=无 Windows ACL/Landlock);②子代理编排(pi-subagents 支持异步/并行/worktree/链式/后台作业;限制=单后端、传输不可换、无跨产品路由);③MCP(pi-mcp-adapter 590K/月下载、pi-mcp-extension、ElieMessieCode/pi-mcp 皆第三方,vs dsh 第一方 dsh-mcp-client、effect-scoped 生命周期自动断连);④ACP 服务端(pi 经社区 pi-acp 桥接,自述「Some ACP features may not be implemented」,vs dsh 第一方 ACP server+按策略一次性机器审批);⑤审批/权限(pi 经 tool_call 拦截可 block/改参,但无统一审批服务、无审计日志、无 per-session ask/never 策略瀑布,vs dsh ctx.approval fail-closed+审计对);⑥定时任务(pi-schedule-prompt 等仅活动会话内生效,vs dsh-schedule 状态持久于 session event log、会话复活续跑);⑦Web UI/后台任务(pi-web-ui 16.9K/月、pi-reactor、pi-background-tasks 81K/月等多实现分裂)。[pi.dev/packages;D 各包 README]

**「第一方 vs 第三方」的真实含义(不是修辞)。** 第一方=同 repo、同版本节奏、同 API 治理、同一套文档;第三方=独立维护者、独立版本、可能随主版本断裂、多实现分裂(pi 的 web UI 至少 4 个第三方实现并存)。但第一方的代价同样真实:breaking 由官方节奏决定——rc.8 的 SQLite 格式破坏(2.3)就是第一方干的。所以这个差异的正确读法是「**风险形状不同**」(谁破坏你、多久一次、破坏多大),不是「有无风险」。

**第③层:程度差异——双方都能做,一方更优(各附反证)。** ①Web UI 自托管:dsh 第一方 `npx @deepseek-ai/dsh web`(127.0.0.1:3080/SSH 模式)vs pi 社区分裂(反证:dsh web fan-out TTFT 退化);②会话/上下文控制:dsh 事件溯源 session——LLM 历史由 log 派生不独立存储、压缩经 surfaceOp replace+sourceEventSeqs 重放校验、ctx.sessionQuery 会话全文检索(pi 无对应);pi 侧=可全量替换 compaction(官方示例+reserveTokens/keepRecentTokens);③插件生命周期:pi /reload 热重载+jiti vs dsh Cordis 可逆 effect(卸载自动回滚)+bundle/profile/patch 组合树+--dump-config 校验;④子代理进程模型:dsh subagent-spawn-in-process 同进程新建 Agent(源码注释「cheapest transport」)vs pi-subagents 每子代理独立 pi 会话(反证:dsh web fan-out TTFT 中位 49.7s——非全面占优)。[D reference/subsystems/session、packages/subagent/subagent-spawn-in-process/src/index.ts;P:docs/compaction.md]

**第④层(容易被误读为差异的):无实质差异。** 模型/后端切换——dsh 的 llm-pi-ai 适配器直接复用 pi 的多 provider 层 @earendil-works/pi-ai,同源,不构成任何一方的差异化。[D:packages/llm/llm-pi-ai/README.md]

**清单的非穷举性(未消解项,如实标注)**:pi 生态约 4,300 包无全文索引,未被 pi.dev 目录收录的跨产品子代理扩展无法穷举排除;nicobailon pi-subagents 子代理进程模型细节未核实;pi 是否原生并行执行工具调用未核实(dsh 有 fail-closed 并行调度器);dsh 部分事实(ACP server 细节/schedule 会话复活续跑)来自社区站 dsh-in-depth/dshdocs,未经官方文档直接复核。[research-extra-2 ⑥]

### 2.3 灵活性代价的真实结构:churn 与断崖两种时间形状

**第一性原理:灵活性=吸收上游变化的义务。** 一个系统的可定制面越大,它的行为就有越多部分由「你的定制」和「上游的演进」共同决定——上游每次变化,breaking 的可能性先落在定制点上。灵活性从来不是免费属性,它是把上游的变化接进自己系统的义务。双方代价都有实测记录,但**时间形状不同**。

**dsh 侧代价(已验证实例,每条可溯源)**:

- **存储格式破坏**:rc.8 SQLite 存储与 rc.7 不兼容——改的是会话数据层;该条列在 changelog 的「Chores」段性能行末尾一句带过,升级前须手动备份 ~/.dsh。[dshdocs.com/guides/changelog;Discussion #3691]
- **dist-tag 混乱**:npm latest=rc.7、next=rc.8——普通 @latest 安装拿到 rc.7;GitHub 全部标 pre-release。[dshdocs.com FAQ how-to-update]
- **插件树整体不可启动**:实测故障——一个不兼容插件可阻止 DSH Web shell 挂载,且插件管理 UI(诊断/卸载入口)与坏插件同树同殁;社区已提案 PoC「保证可选插件不兼容时仍可启动」。[Discussion #4175,out-of-tree 插件维护者实测+PoC]
- **安装开销**:rc.8 依赖树 60+ 包,7.7GB RAM 机器 npm install 触发 V8 heap OOM(exit 134)。[Discussion #3691]
- **承诺本身**:README 明示「developer preview...THERE WILL BE COMPATIBILITY-BREAKING CHANGES」;开源 11 天 rc.5→rc.8 共 5 个 rc。[D repo README]
- **学习曲线与自持负担**:插件开发前须学 Cordis(官方文档明示「改 packages/ 前先读,假设你已懂 Cordis」「建议用 agent 探索代码库」);Cordis vendored 进 monorepo 自担同步负担,7,090 commits 含 vendored 历史不可拆分;文档靠第三方站补位。[deepseek-harness.github.io reference/develop;repo vendor/README.md]
- **性能开销**:subagent SDK 每次运行新建进程(无池化);自托管 web TTFT~10s、成熟会话 median 49.7s/p90 142s;zero prompt-cache hit+6.6K token schema 开销使每轮 3.5s→53s。[D packages README;Discussion #3235/#3304]

**pi 侧代价(本地已验证)**:

- **核心固定不可替换**:agent loop/默认骨架/渲染行为无配置口——主题仅 51 色彩 token,diff 反显等须改 dist JS(rick 已否决 patch 路线)。[pi README Philosophy;.rick/domain/pi-runtime.md]
- **bus factor 双层**:核心层=badlogic 占 69% commits、新贡献者 issue/PR 默认自动关闭;扩展层=subagent fanout 踩在 pi-subagents 上(nicobailon 单人维护,另有 @tintinweb 同名实现易混淆)。[research-S ②-8;internal/env/extensions.go]
- **小项**:不读环境变量(PI_PROVIDER/PI_MODEL/PI_API_KEY 无效,必须 CLI flags);pi install 对 user scope 包可能共享全局 npm root,卸载互相影响;0.51.0 曾因 strict 工具校验硬失败。[.rick/domain/pi-runtime.md]

**churn vs 断崖:依赖风险的两种时间形状(本节核心概念)。** churn 形=高频小破坏:每次升级都可能有适配工作,但单次工作量小、可预算;断崖形=长期无事,一旦发生就是全量级重做(存储格式迁移/插件树修复),不可预算但间隔长。dsh 的实测形状=rc 节奏断崖(11 天 5 个 rc,其中至少 1 次会话数据层格式破坏+1 次插件树整体故障模式);pi 的形状=核心 API 冻结(rick 以 0.84.2 锁版本+stock 不 patch 已隔离此层)+bus factor 断崖(维护者消失是不可预算事件,但发生前系统稳定)+第三方扩展随主版本断裂的风险。**没有无风险选项,只有风险形状的偏好**——偏好哪种形状是判断项(第五节)。

**对 rick 规模的可支付性(事实量级,不下结论)**:rick 是单人项目,runtime 相关 Go 代码约 5,365 行(internal/runtime+env+builder);已付的 pi 代价=触发语法迁移(loop_6 已完成)+自闭环 runtime 副本(0.84.2 锁版本)。若引入 dsh 并保持「双 runtime 等价」,按 rc 节奏(11 天 5 rc)每个 rc 需重验等价性,且已知至少 1 次会话数据层格式破坏+1 次插件树整体不可启动故障模式。[research-extra-2 ②]

### 2.4 判断力度量科学:正确性与赔率有成熟数学,三项连乘没有

human 提出三维模型(正确性/赔率/影响力加权)。本节讲透:前两维有几十年成熟的数学,第三维有决策分析的标准工具;但「三项连乘成一个标量」偏离两类成熟范式——这是数学事实,不是对三维直觉的否定。

**proper scoring rules:为什么「正确性」有标准答案。** 给概率判断打分,难在防博弈:若规则设计不当,报分者可通过虚报概率得利(线性计分下,报 100% 对自己有利的事件、报 0% 对自己不利的,分数更好看)。**proper scoring rule** 的定义:诚实报告真实概率是期望分数最优策略;strictly proper 则诚实是唯一最优。Brier 分=(p−y)²、对数分=−(y·ln p+(1−y)·ln(1−p))均二次/严格 proper。[Gneiting & Raftery 2007, JASA] 这就是 GJP(IARPA 锦标赛)用 Brier 给几千名预测者排名的原因:**分数不可被话术操纵**。Brier 可分解为校准(报 70% 的事件是否七成发生)+分辨力(能否区分会发生/不发生)+不确定度;可靠性图按概率分档累计,每档需数十条才有读数。[cawcr.gov.au verification]

**预测市场赔率:为什么「赔率」也有标准答案。** 预测市场(Hanson LMSR 做市)中,价格=群体概率的聚合。押反共识方向且命中,回报 ∝ (1−p)/p——共识越强(p 越大)你越敢反着押、你对了回报越大;log 评分对低概率命中给对数级大分。[Hanson 2002] 所以「赔率=反共识定价」在数学上完备。**但它需要什么**:一个「群体概率」作为共识基准。rick 是 n=1 决策,无判断者群体——这是三维模型落地的真实缺口(**数据源缺口,非数学缺口**)。

**三项连乘的数学问题(两个偏离+一个激励扭曲)。**

1. **偏离 proper 范式**:proper 分数的定义域只能是「所报概率×结果」的函数——乘上别的系数(影响力)后规则不再 proper,诚实报概率不再是最优策略,博弈空间重新打开。[leaf-D 数学分析]
2. **偏离 EV 范式**:期望值是**加性结构**(决策树的效用按概率加权求和),不是连乘;影响力若对应 EV/VoI,标准形态是分列加权而非乘子。[leaf-D]
3. **激励扭曲**:乘积形式激励虚报影响力——影响力是自报且难以即时验证的维度,作为乘数时,报高一点总分按比例放大:三维中唯一可被单方面操纵的维度恰好获得杠杆位置。[leaf-D]

**领域共识形态**:「分数(proper)+影响力(EV/分级)」**分列**记录或分层使用,不合成单标量。「正确、反共识、重要」三维**直觉**本身有投资领域的对应(正确且非共识才有超额收益——Kahn 式表述),问题不在直觉,在**合成公式的形态**。[leaf-D]

**Dalio believability:自加权需要什么。** 可信度=「该领域既往战绩×能讲清因果逻辑」,桥水经 baseball card/点收集器年级别积累,打分公式未公开(专有机制)。单人场景退化为「用本人历史准确率给未来判断的置信度加权」——**依赖 Brier/校准分数先行累计**,即先有②③的数据积累,④才有输入。[principles.com]

**DQ 与 AAR:过程与结果的解耦(度量体系的另一半)。** 决策质量 DQ(Spetzler/SDG)六要素=恰当框架/可行多元备选/可靠信息/清晰价值权衡/逻辑健全/承诺行动——评「**决策时点的过程质量**」,零结果数据也可打 0–10 快检。它解决一个真实问题:「好过程+坏运气」不应被结果回收误罚、「坏过程+好运气」不应被误奖——结果度量必须配过程度量才公平。[sdg.com] AAR(美军 TC 25-20)四问=原定发生什么/实际发生什么/为何有差异/下次怎么改——单事件即可执行、零样本量要求,与 rick dream 复盘结构一一对应,是「判断→结果回收」闭环最直接的方法论载体。[TC 25-20] VoI/EVPI 则需显式决策树(备选/概率/效用),文本拍板无法直接算——「影响力」维度若走 EV 路线,需要建模前置。[ISPOR]

**GJP 的数据要求(用来校准期望)**:每人每期数十~上百条「概率+截止日+可判对错」的命题;CHAMPS 去偏差训练(<1h)提升 Brier 6–11%;超级预测者=前 2% 且跨年稳定。[Mellers et al. 2014] rick 的 dream 回收窗口 1–2 周,短于 GJP 典型期限,命题设计可行——瓶颈在密度(2.6)。

### 2.5 N=1 度量的正确姿势:单用户纵向统计工具箱

human 的 agent 四指标含主观评价(每任务好/差)——这是 N=1(单 human)纵向场景。有一整套与「群体横断面」不同的工具,先讲为什么常用工具不适合。

**为什么 NPS/CSAT 不适合。** NPS/CSAT 为群体横断面设计:11 点量表任意截断(6/9)、丢弃部分样本信息、差值统计量方差大[JBR 2022;arXiv:1806.10452];回收率仅 10–25% 且自选择偏差(极端体验者过采样)[PMC1464019];主观分与实际行为/绩效弱关联[NN/g]。核心问题:**横断面的统计前提**(N 大、个体间可交换、一次性快照)在 N=1 纵向场景全部不成立——单用户单次打分无统计意义。这不是说主观评价没用,是群体工具用错了场景。

**N=1 的四件工具(均有文献与实践)**:

1. **Beta-Binomial 共轭追踪**:单用户好/差比率的标准后验更新模型(先验+逐次更新)[JBES 1987];配**贝叶斯收缩**防「单次满分置顶」——1 条 5 星不该排在 500 条 4.8 之上,收缩把小样本评分向先验均值拉回。[evanmiller.org]
2. **SPRT 序贯检验**:Wald 序贯概率比检验,允许 peeking(随时看中间结果)与提前停止,最小化样本需求——N=1 场景数据来得慢,经典固定样本量设计不可行,序贯是正确形态;A/B 实验实践已产品化。[arXiv:2606.24871;Statsig docs]
3. **SCD 单被试设计**:A=基线、B=干预;WWC(WWC/IES 官方证据标准)要求**至少 4 个 A/B 相位(ABAB)**方可因果推断;交替处理设计(ATD)=单被试内快速交替两种处理直接比较。[ies.ed.gov wwc_scd.pdf;Barlow & Hayes 1979] **对 rick 的含义**:配置变更(如 runtime 切换、模板改版)是「干预」——单次切换的前后对比不构成因果证据,因果推断需要 ABAB 相位摆动。
4. **难度归一化(IRT/任务级 psychometrics)**:难任务的差评可能是任务难、不是 agent 差;IRT 同时估计能力与任务难度,可归一化。AI 场景需校正(模型少/条目多/分布偏态偏离人类测验假设)。[arXiv:2604.00594;2607.15190] 反面参照:τ-bench 平凡 agent 无领域知识可过 38%——未难度校准的 pass rate 会系统性误导。[NeurIPS 2025 D&B]

**「dream 中真实度量」的可行组合模式**(方法组合的事实呈现):SCD 相位设计(配置变更=干预,≥4 相位)× SPRT 序贯检验 × 行为指标(isError/durationMs)辅助主观分 × Beta-Binomial 纵向追踪——因「主观分与行为弱关联」的教训,主观/行为两轨并行而非互替。[leaf-E] 另外 per-user 偏好学习路线(P-RLHF 每用户独立 user model [arXiv:2402.05133]、变分偏好 [arXiv:2408.10075]、SynthesizeMe 从交互历史合成 persona [arXiv:2506.05598])的共同数据要求=用户交互历史——rick 已有(会话 JSONL/artifacts),该路线与上述统计工具互补。

### 2.6 四指标数据可得性矩阵 + 判断语料现状

**human 四指标→rick 数据源映射(本地逐项验证)**:

| human 指标 | 数据源 | 状态 |
|---|---|---|
| ①主观好/差 | judgment.md 拍板(Beta-Binomial 纵向追踪) | 需新增结构化字段(rick 层) |
| ②工具出错次数 | 会话 JSONL 工具结果事件 isError 字段 | ✅ 已验证可派生 |
| ②debug skill 调用次数 | JSONL 无 skill 一等事件(仅文本内出现);jobs/*/doing/debug/bug*.md 现为 0 文件 | ❌ 不可派生(未消解) |
| ②异常路径/纠正耗时 | transcript 工具序列失败-修复循环(dream 模板已按报错/重试排序读取) | 部分可派生(需解析器) |
| ③任务总工时 | meta.json durationMs 汇总+父会话时间戳 | ✅ 已验证 |
| ④验收质量评价 | meta.json acceptance.childReport(criteriaSatisfied/runtimeChecks/evidence 已自动采集) | ✅ AI 侧已有;human 侧需新增 |

**判断语料现状(本地清点)**:6 个 judgment.md 共 687 行,human 原话+拍板文本,**无数值概率/截止日字段**;dream 回收窗口 1–2 周(短于 GJP 典型期限,命题设计可行);按 GJP 口径(每人每期数十~上百条)当前判断密度低 1–2 个数量级。[research-extra-2 ④]

**本轮最关键的贯穿性事实**:七种成熟方法(GJP-Brier/校准曲线/预测市场/Dalio/DQ/AAR/VoI)与 N=1 工具的**共同前置=拍板结构化**(概率+截止日+可验证命题+影响力预估)——这属 rick 层的 judgment 格式扩展,**runtime 无关**。度量采集所需数据源(会话 JSONL/artifacts/meta.json)pi 已满足(离线脚本即可);dsh 侧 session log 是插件、理论可自定义采集但未实测(未消解项)。这条事实直接决定裁决链条的结构:若度量所需能力双方都满足,「度量后裁决」的判别力为零,裁决依据退回成熟度/生态/维护/风险形状维度(第五节 N5)。

### 2.7 常见误区

**误区一:把「dsh 更灵活」当均匀标量。** 它实际是三层异质差异:1 项结构性自由度(loop 替换)+7 项生态位差异+4 项程度差异+1 项无差异(2.2)——其中 rick 现有需求真正触到的只有第三方扩展层的 2 个点(pi-subagents)。「更灵活」的争论若不落到「哪层灵活、为哪个需求」,就是空转。

**误区二:把第一方/第三方当「有无风险」。** 第一方的 breaking 由官方节奏决定(rc.8 SQLite 破坏),第三方的 breaking 由维护者存续决定(pi-subagents 单人)——是风险形状不同(2.3),不是有无。

**误区三:把三维直觉与三维连乘混为一谈。** 「正确且反共识才有超额收益」的直觉有投资领域对应;数学问题只在合成公式形态(连乘破坏 proper 性+激励虚报影响力)——修形态不等于否定直觉(2.4)。

**误区四:把群体横断面工具搬进 N=1。** NPS/CSAT 的统计前提(大 N、可交换、快照)在单用户纵向全部不成立;正确工具是 Beta-Binomial/SPRT/SCD(2.5)。

**误区五:把「度量后裁决」当必然有判别力。** 若度量所需能力 pi/dsh 都满足(当前证据指向如此:数据源已存在、离线脚本即可),度量研究本身不构成裁决依据——裁决退化为风险形状偏好。这不是失败,是裁决结构的澄清:承认「裁决依据=风险偏好而非能力差异」与「假装度量会给出答案」是两条不同的路(第五节 N5)。

## 三、启发式追问

以下追问建立在 2.1–2.7 已讲清的知识之上,承接 think-gate-S-r2 top-5,每条附「改变判断的证据」。只问,不替答。

**Q1 掌控度的限度与零命中含义:期权怎么定价。** 2.1 讲了控制收益非单调(A3→A4 边际递减、弱模型≈0)+rick 需求清单零命中 loop 替换。那么「为未来保留 loop 替换灵活性」的价值该按什么框架评估——期权框架(未来出现需替换 loop 需求的概率×命中时收益−持有限制性配置的持续成本)还是 YAGNI(需要时再评估迁移、现在不为此付任何代价)?两个框架在「未来需求概率足够低」时给出相同答案,分歧出现在概率多高、命中收益多大的什么区间?
改变判断的证据:路线图出现必须替换 agent loop/session log 的具体功能(期权框架胜出);或 R7-4 基线显示瓶颈全在 runtime 之上的层(YAGNI 证据增强)。

**Q2 判断的可归因子集:划在哪条线上。** 2.4 讲了无对照归因不可靠(LLM 判分 κ≈0.46)、结果=判断×执行×环境的混杂;2.6 讲了判断密度低 GJP 口径 1–2 个数量级。若把「结果归因」限定在有机器可读信号的子集(验收通过/失败、工时、成本——这些是真值),子集外只做 DQ 过程评分不做结果归因,这个划界你能接受吗?子集占比小到什么程度时,「判断力度量」项目本身应降级为纯过程复盘(DQ/AAR)?
改变判断的证据:试评历史 6 个 judgment——「有验收信号」子集占比,及对该子集归因的一致率(一致率高=划界可行;占比低于阈值=项目降级为过程复盘)。

**Q3 反共识的操作化:接受哪个基线。** 2.4 讲了赔率数学需要「群体概率」做共识基准,而 rick 是 n=1、无判断者群体。现成候选:以 agent 初步建议为共识基线(human 偏离且正确=反共识价值,human-loop 内可测、零新增基建)。但这个基线有已知偏差:agent 建议不是群体共识,是单一模型分布的样本——「偏离 agent」测到的是「与模型的分歧」,不是「与市场的分歧」。接受这个代理(附口径说明),还是等外部共识数据源(AI 判断分布/社区共识)可采样再做,还是删/降权该维度?
改变判断的证据:校准实验——把若干历史拍板让多个独立模型给建议,看「human vs agent 分歧率」是否稳定可复现(稳定=代理可用;不稳=该维度在本数据源下无法操作化)。

**Q4 三维模型的形态:向量还是标量,权重谁定。** 2.4 讲了连乘破坏 proper 性+激励虚报影响力,领域共识形态是分列;2.5 讲了 Goodhart 类风险。那么:三维分别记录(向量画像,不做合成)vs 分列后按场景加权合成 vs 连乘(须显式接受数学偏离)?若合成,权重谁定、多久校准一次?Strathern:「当度量成为目标,它就不再是好度量」——一旦合成分数用于裁决 runtime/选 agent/自评,被测维度会被无意识优化。先向量记录若干 epoch 再决定合成式,是不是一个信息成本更低的路径?
改变判断的证据:向量记录若干 epoch 后观察——若没有任何维度实际进入过决策,合成公式失去意义;若单一维度主导使用,加权设计需重想。

**Q5 度量判别力为零时:接受裁决降级吗。** 2.6 讲了拍板结构化 runtime 无关+数据源 pi 已满足(离线脚本)。若逐项核对度量所需能力(事件访问/无头集成/产物持久化/结构化采集)发现 pi/dsh 都满足,则「完成度量研究后裁决 DSH vs PI」的前提链空转——裁决依据从能力差异退回成熟度(0.1.x-rc breaking vs 254 版稳定)/生态(约 4,300 包 vs 无目录)/维护(bus factor 1 vs 34 人)/风险形状(churn vs 断崖,2.3)。这个降级结构你能**预先**接受吗(即预先承认:裁决依据=风险偏好而非能力差异)?还是要求先做能力判别力核对(R7-5 spike:列 hard-constraints 清单逐项核对两 runtime)再定裁决结构?
改变判断的证据:能力核对出现仅一者支持的项(如需替换 session log 自定义采集→仅 dsh;如需 /reload 热重载→仅 pi)——此时度量重新获得判别力,链条不空转。

**Q6 dream 样本量与 epoch 口径。** 2.5 讲了 SCD 需 ABAB≥4 相位、判断密度低 GJP 1–2 个数量级;2.6 讲了语料 687 行无数值字段。那么「epoch」的口径定为什么(每 loop/每月/每季)?若每 loop 可归因判断数<5,趋势判读须按月/季聚合——观测窗拉长后,配置变更(干预)频率还够在窗内摆出 ABAB 相位吗?还是承认 rick 的度量目标是「过程质量趋势」(DQ/AAR 可承载、零样本量要求)而非「效果因果推断」(SCD 承载、需相位设计),把因果推断从目标里显式删掉?
改变判断的证据:清点每 loop 可归因判断数——≥5/loop 则 loop 口径可行;<5 则必须月/季聚合,且 ABAB 相位设计的干预频率需重排(或放弃因果推断目标)。

## 四、延伸学习指导

**判断力度量(对应 2.4)**

1. Tetlock《Superforecasting》第 1-5 章(或 Mellers et al. 2014, journals.sagepub.com/doi/10.1177/0956797614524255)——为什么读:GJP 全景(超级预测者如何选出、Brier 如何用于排名、CHAMPS 训练为何有效);读完你能判断 rick 拍板结构化应长什么样、判断密度要求是否现实。
2. Gneiting & Raftery 2007《Strictly Proper Scoring Rules, Prediction, and Estimation》(JASA,sites.stat.washington.edu PDF)——为什么读:proper scoring 的严格数学;读完你能自己推导为什么连乘破坏 proper 性、为什么 Brier/对数分是标准选择。
3. Hanson 2002 LMSR(mason.gmu.edu/~rhanson/mktscore.pdf)——为什么读:市场赔率=反共识定价的机制数学;读完你能判断「赔率」维度操作化需要什么基线、代理基线的偏差在哪。
4. Spetzler et al.《Decision Quality: Value Creation from Better Business Decisions》(Wiley)——为什么读:DQ 六要素量表原文;读完你能直接设计 dream 复盘的过程侧评分表,与结果回收互补。
5. Dalio《Principles》believability weighted decision-making 章节(principles.com/633d...)——为什么读:可信度加权的原型机制;读完你能判断自加权(用自己的历史准确率)需要什么数据积累、何时可以启动。
6. TC 25-20 AAR(美军 Leader's Guide,经 nick.groenen.me 镜像)——为什么读:复盘四问原始出处;读完你能对照 dream 模板查漏补缺。

**N=1 统计(对应 2.5)**

7. WWC Single-Case Design standards(ies.ed.gov/ncee/wwc wwc_scd.pdf)——为什么读:ABAB≥4 相位证据标准的官方文件;读完你能判断 rick 的配置变更实验有没有资格做因果推断。
8. Evan Miller《Ranking Items with Star Ratings》(evanmiller.org)——为什么读:贝叶斯收缩的具体公式与直觉;读完你能防「单次满分置顶」类错误。
9. Barlow & Hayes 1979 交替处理设计(PMC1311363)——为什么读:单被试内快速 A/B 两处理的原典;读完你能设计「两配置直接对比」而不必等完整 ABAB。
10. Wald SPRT 综述(arXiv:2606.24871;Statsig docs)——为什么读:序贯检验允许 peeking/提前停止的产品化实践;读完你能定 N=1 场景的判读规则。

**runtime 工程判断(对应 2.1–2.3)**

11. pi README Philosophy 节+packages/coding-agent/README.md——为什么读:「without having to fork」划界声明的一手原文;读完你能自己判断任一定制需求落在 pi 界内还是界外。
12. dsh docs/architecture.md+Cordis primer(deepseek-harness.github.io)——为什么读:loop-as-plugin 与可逆 effect 的一手描述;读完你能判断 dsh 灵活性的真实边界与学习成本。
13. dshdocs.com changelog+Discussion #3691/#4175/#3235——为什么读:破坏性变更、插件树故障、fan-out 性能的一手记录;读完你能对「断崖形风险」形成自己的量级感。

## 五、决策汇总表:需 human 逐项拍板

**前轮 D1-D11 覆盖状态(事实核对,含 think-gate-S-r2 判定)+ 本轮新决策项 N1-N7**:

| # | 拍板项 | 状态/背景(来源) | 选项(不含倾向) | 关联 |
|---|---|---|---|---|
| D1 | runtime 取舍 | **已拍板转变**:「都要」→「做出取舍」(human Q2,第二次反转);取舍依据结构未定 | 取舍依据=能力差异(2.2 三层清单)还是风险形状偏好(2.3)——见 N5 | N5 |
| D2 | spec 可验证性形态 | **已拍板**:有限穷举+语义契约(human Q1);关闭 | — | — |
| D3 | spec 层实验角色 | **悬置**:降级后 spec 支点作用消失,实验角色需重定义(think-gate-S-r2) | 重定义为通用工程实践 / 明确放弃 / 继续挂起 | 门禁 |
| D4 | 双 runtime 等价机制 | **条件失效**:仅双 runtime 下需要;D1 已转取舍,think-gate-S-r2 建议明确关闭 | 明确关闭 / 保留(若未来反转回双 runtime) | D1 |
| D5 | 双实现成本归属 | **条件失效**:同 D4(think-gate-S-r2 建议明确关闭) | 明确关闭 / 保留 | D1 |
| D6 | 指标体系 | **部分覆盖**:四指标+数据源矩阵就绪(2.6);权重/基线/归因/形态未定 | 见 N1-N4/N6/N7 | N1-N7 |
| D7 | 先度量 vs 先支持 | **挂起**:等 Q2/Q5 澄清(think-gate-S-r2) | 挂起 / 随裁决一并定 | Q5 |
| D8 | benchmark 用途 | **已拍板**:拒绝现成 benchmark;benchmark=human+agent 个性化评价体系(human Q5) | — | D6 |
| D9 | benchmark 选型 | **已关闭**(D8 转向个性化体系) | — | D8 |
| D10 | 决策撤销条件 | **悬置**:两次反转未记撤销条件;第三次决策(runtime 裁决)将至,应前置(think-gate-S-r2) | 本轮裁决时一并写撤销条件 / 不写 | 门禁 |
| D11 | R7 未消解项 | **部分**:本轮新增=清单非穷举/dsh 社区站事实未复核/debug 指标不可派生/反共识基线缺口/dsh 度量采集未实测(research-extra-2 ⑥) | 挂起 / R7-5 spike / 放弃 | — |

**本轮新决策项**:

| # | 拍板项 | 背景事实(来源) | 选项(不含倾向) | 关联 |
|---|---|---|---|---|
| N1 | 三维模型形态 | 连乘破坏 proper 性+激励虚报影响力;领域共识=分列;Goodhart 风险(2.4) | 向量分别记录 / 分列+场景加权 / 连乘(须显式接受数学偏离) | Q4 |
| N2 | 反共识基线 | 赔率数学需群体共识;n=1 无群体;候选=agent 建议为基线(已知偏差:测的是与模型的分歧)(2.4) | agent 建议为基线 / 等外部共识数据源 / 删除或降权该维度 | Q3 |
| N3 | 可归因子集 | 无对照归因 κ≈0.46;「有验收信号」子集=机器可读真值(2.4/2.6) | 限定子集归因+其余 DQ 过程评分 / 全量尝试归因 / 放弃结果归因只做过程 | Q2 |
| N4 | epoch 口径 | 判断密度低 GJP 1–2 个数量级;SCD 需 ABAB≥4 相位(2.5/2.6) | loop / 月 / 季;并决定是否显式放弃因果推断目标(降级为趋势+过程质量) | Q6 |
| N5 | 裁决结构(降级预案) | 度量共同前置=拍板结构化,runtime 无关;若判别力为零,裁决退回成熟度/生态/维护/风险形状(2.6/2.3) | 预先接受降级结构 / 要求先做能力判别力核对(R7-5 spike)再定 | Q5 |
| N6 | 拍板结构化格式 | 所有度量方法的共同前置(概率+截止日+可验证+影响力预估);judgment.md 现无此字段(2.6) | 采纳为 judgment 格式扩展 / 部分采纳(如仅概率+截止日) / 不采纳 | Q2-Q4 |
| N7 | debug skill 指标处置 | 无 skill 一等事件不可派生;jobs/*/doing/debug/ 现为 0 文件(2.6) | 放弃该指标 / 新增采集(pi 扩展 appendEntry 一等事件) / 文本启发式近似(附噪声标注) | Q4 |

---

*本简报仅综合事实与讲解知识,不含对「选 pi / 选 dsh / 选某度量形态」的倾向性结论;事实带来源标注,判断项全部归入第五节由 human 逐项拍板。R7 未消解项已如实标注「未消解」(2.2 末/2.6/D11/N2/N7)。*
