# research-extra-2 简报 — pi/dsh 优劣势对比 + 判断力度量 + 个性化指标

> 阶段：S 门禁轮临时调研（human 明确请求）| 基准：2026-08-25 | 方法：本地代码尽调（C）+ 自查 web（B）+ 3 联网叶子（A/D/E，见文末叶子清单）| 前序：research-S.md / research-S-r2.md（从略部分不重复）
> 硬约束：仅事实+来源；分层清单是事实呈现不是选边；「未消解」集中列 ⑥。

## ① dsh 独有能力清单（human 问 3，三层；D=deepseek-ai/deepseek-harness，P=earendil-works/pi）

**第①层：pi 核心+官方生态+第三方扩展均做不到**

1. **替换 agent loop 本身**：dsh 官方架构文档明示 agent loop 是插件、可从配置替换；pi 扩展 API（registerTool/registerCommand/事件拦截/自定义渲染）无主循环替换接口，README 以「without having to fork」划界，更深改动须 fork。[D:docs/architecture.md；P:packages/coding-agent/README.md]
2. **跨产品子代理后端**：dsh 第一方多后端共存：inprocess spawn/fork、ACP、Codex（app-server --stdio）、Claude Code；pi-subagents 子代理均为 pi 子会话，已知生态无任何扩展把 Codex/Claude Code/ACP agent 作为子后端（社区 pi-acp 方向相反：pi 被编辑器驱动）。[D:packages/subagent/README.md+2026-08-04 feature note；github.com/nicobailon/pi-subagents；github.com/svkozak/pi-acp]
3. **三平台内核沙箱+fail-closed 语义**：dsh-sandbox-local 覆盖 Linux bwrap→Landlock 探测链/macOS Seatbelt/Windows ACL restricted-token，不可用即拒绝执行「never silently falls through unconfined」；pi 生态沙箱仅 macOS/Linux 且存在 fail-open 实现（@jerryan/pi-bash-wrap 非 Linux 平台原样放行）。[D:packages/sandbox/sandbox-local/README.md；npmjs @jerryan/pi-bash-wrap]

**第②层：pi 靠第三方扩展可部分做到（附限制）**

1. OS 沙箱：@nqbao/pi-sandbox（macOS sandbox-exec/Linux bubblewrap，基于 @anthropic-ai/sandbox-runtime）、rnorth/sandboxed-pi（Docker 包裹）、官方示例 sandbox；限制：无 Windows ACL、无 Landlock。[npmjs/pi.dev packages]
2. 子代理编排：pi-subagents 支持异步/并行/worktree 隔离/链式流水线/后台作业；限制：单后端（皆为 pi 会话）、传输不可换、无跨产品路由。[github.com/nicobailon/pi-subagents]
3. MCP：pi-mcp-adapter（590K/月，单代理工具 ~200 token）、pi-mcp-extension、ElieMessieCode/pi-mcp（HTTP/SSE）；皆第三方，dsh 为第一方 dsh-mcp-client（effect-scoped 生命周期自动断连）。[pi.dev/packages/pi-mcp-adapter；D:packages/mcp/mcp-client/README.md]
4. ACP 服务端：pi 经社区 pi-acp 适配器（spawn pi --mode rpc 桥接，自述「Some ACP features may be not implemented」）；dsh 第一方 ACP server+按策略一次性机器审批。[github.com/svkozak/pi-acp；P Discussion #4444；dsh-in-depth.com/llm-platform/acp]
5. 审批/权限：pi 经 tool_call 拦截可 block/改参（pi-permission-gate/@pi-lab/permissions/pi-agent-review 等）；限制：无统一审批服务、无审计日志、无 per-session ask/never 策略瀑布；dsh ctx.approval fail-closed+审计对+每会话策略。[pi.dev/docs/extensions；D:docs/subsystems/approval.md]
6. 定时任务：pi-schedule-prompt/manojlds/pi-scheduler 等仅活动会话内生效；dsh-schedule 状态持久于 session event log、会话复活续跑。[npmjs；D:packages/schedule/src/index.ts]
7. Web UI/后台任务：pi-web-ui（16.9K/月）、pi-web.dev（server-side 工作区，Docker/systemd 部署）、pi-reactor（cron+webhook+花费上限）、pi-background-tasks（81K/月）；均第三方多实现分裂。[pi.dev/packages；pi-web.dev]

**第③层：pi 可做到但 dsh 更优（客观依据）**

1. Web UI/自托管：dsh 第一方 `npx @deepseek-ai/dsh web`（127.0.0.1:3080/SSH 启动）vs pi 社区多实现分裂。[D:README.md；pi-web.dev]
2. 会话/上下文控制粒度：pi 可全量替换 compaction（官方 custom-compaction 示例+reserveTokens/keepRecentTokens）；dsh 为事件溯源 session——LLM 历史由 log 派生不独立存储、压缩经 surfaceOp replace+sourceEventSeqs 重放校验、事件类型 merge-extensible、ctx.sessionQuery 会话全文检索（pi 无对应）。[P:docs/compaction.md；D:reference/subsystems/session、packages/session-query]
3. 插件生命周期：pi /reload 热重载+jiti；dsh Cordis 插件=可逆 effect（卸载自动回滚）+bundle/profile/patch 组合树+--dump-config 组合校验。[P:pi.dev/docs/extensions；D:docs/architecture.md]
4. 子代理进程模型：dsh subagent-spawn-in-process 同进程新建 Agent（源码注释「cheapest transport」）；pi-subagents 每子代理独立 pi 子会话。反证：dsh Web host fan-out/成熟会话 TTFT 退化至中位 49.7s/p90 142s（网关仅 2.1s）——非全面占优。[D:packages/subagent/subagent-spawn-in-process/src/index.ts；Discussion #3235]
5. 模型/后端切换：**无实质差异**——dsh llm-pi-ai 适配器直接复用 @earendil-works/pi-ai（pi 的多 provider 层），不构成 dsh 独有能力。[D:packages/llm/llm-pi-ai/README.md]

## ② 灵活性代价对照（human 问 2）

**dsh 侧代价（已验证）**

| 代价项 | 事实与量级 | 来源 |
|---|---|---|
| 破坏性变更承诺 | README：「developer preview...THERE WILL BE COMPATIBILITY-BREAKING CHANGES」；开源 11 天 rc.5→rc.8 共 5 个 rc | D repo README |
| 破坏性变更实例 | rc.8 SQLite 存储格式与 rc.7 不兼容（改的是会话数据存储，该条列在 "Chores" 段性能行末一句带过，升级前须手动备份 ~/.dsh）；rc.6→rc.8 keyed slots 须 options.key | dshdocs.com/guides/changelog; Discussion #3691 |
| 版本发布混乱 | npm dist-tags latest=rc.7 / next=rc.8，普通 @latest 安装拿到 rc.7；GitHub 全部标 pre-release | dshdocs.com FAQ how-to-update |
| 安装开销 | rc.8 依赖树 60+ 包，低内存机（7.7GB RAM）npm install 触发 V8 heap OOM（exit 134） | Discussion #3691 |
| 插件树调试复杂度 | 实测故障：一个不兼容插件可阻止 DSH Web shell 挂载，同时移除诊断/卸载该插件所需的设置入口（插件管理 UI 与坏插件同树同殁）；社区已提案 PoC「保证可选插件不兼容时仍可启动」 | Discussion #4175（out-of-tree 插件维护者实测+PoC） |
| 学习曲线 | 插件开发前须学 Cordis（官方 primer+tutorial）；架构文档明示「改 packages/ 前先读，假设你已懂 Cordis」「建议用 agent 探索代码库」；文档靠第三方站补位 | deepseek-harness.github.io reference/develop |
| 性能开销 | subagent SDK 每次运行新建进程（无池化，但另有 in-process 后端，见①-③-4）；自托管 web TTFT~10s、成熟会话 median 49.7s/p90 142s（#3235）；zero prompt-cache hit+6.6K token schema 开销使每轮 3.5s→53s（#3304） | packages README; research-S-r2 ③-4 |
| Cordis vendored 自持性 | Cordis vendored 进 monorepo+vendor/README.md 同步规程：不受上游破坏影响但自担同步负担；7,090 commits 含 vendored 历史不可拆分 | repo vendor/README.md; research-S ③-8 |

**pi 侧代价（已验证，本地）**

| 代价项 | 事实 | 来源 |
|---|---|---|
| 核心固定不可替换 | agent loop/默认骨架不可整体替换，扩展只能拦截钩子（tool_call block/改参/改结果）+加能力；主题仅 51 色彩 token，diff 反显等渲染行为无配置口（改行为须改 dist JS，rick 已否决 patch 路线） | pi README Philosophy; .rick/domain/pi-runtime.md |
| 第三方扩展依赖 | subagent fanout 踩在 pi-subagents（nicobailon 单人维护）上；存在 @tintinweb 同名实现易混淆 | internal/env/extensions.go; research-S ②-5/8 |
| 不读环境变量 | PI_PROVIDER/PI_MODEL/PI_API_KEY 不生效，必须 CLI flags | .rick/domain/pi-runtime.md |
| 包管理边界 | pi install 对 user scope 包可能共享全局 npm root，卸载互相影响；0.51.0 曾因 strict 工具校验硬失败 | .rick/domain/pi-runtime.md |
| bus factor | badlogic 占 69% commits；新贡献者 issue/PR 默认自动关闭 | research-S ②-8 |

**对 rick 规模的可支付性（仅事实量级，不下结论）**：rick 是单人项目，runtime 相关 Go 代码 ~5,365 行（internal/runtime+env+builder），已付的 pi 代价=触发语法迁移（loop_6 已完成）+自闭环 runtime 副本（0.84.2 锁版本，stock 不 patch）；若引入 dsh 并保持 human Q4 要求的「双 runtime 等价+可验证 spec」，按 rc 节奏（11 天 5 个 rc）每个 rc 需重验等价性，且已知至少 1 次存储格式破坏（会话数据层）+1 次插件树整体不可启动故障模式。

## ③ rick 定制需求盘点（human 问 1：限度在哪）

**今天实际用到的定制点**（全部来自本地代码，pi 均满足，标注依赖层）：

1. CLI flags 面：`-p` / `--append-system-prompt <file>`（方法+实例层提示词注入）/ `--session-id` / PiExtraArgs（provider/model/api-key）——pi **核心**。[internal/runtime/cli.go:60-80, runtime.go:118-122]
2. `PI_CODING_AGENT_DIR` 隔离托管目录（~/.rick/pi/agent）——pi **核心**。[internal/runtime/agentdir.go:64-69]
3. `--mode json` JSONL 事件流+agent_settled 终止信号——pi **核心**。[internal/runtime/executor.go]
4. compaction 持久（--append-system-prompt 使协议不遗忘）——pi **核心行为**，rick 依赖。[internal/runtime/runtime.go:42]
5. subagent fanout（workflowScript+runs.run/runs.all+agent 名，templates 内 128 处关键字）——pi **第三方扩展 pi-subagents**。[internal/env/extensions.go:10; research-S-r2 ①-1]
6. 自定义 agent（agents/*.md frontmatter：tools/thinking/timeoutMs/systemPromptMode replace）——pi **核心 agents 目录+pi-subagents 约定**。[internal/env/agents.go]
7. 扩展安装/更新/settings.json 托管——pi **核心（pi install）**。[internal/env/settings.go, update.go]
8. 产物读回（sessions/--cwd--/*.jsonl、.pi/subagents/artifacts/*_meta.json+transcript.jsonl）——pi 核心+**第三方格式耦合**（research-S-r2：切换规则「templates 不改」不成立）。

**结论（事实）**：rick 今天 8 个定制点中 6 个落在 pi 核心（稳定 API），2 个落在第三方扩展（pi-subagents）；无任何一个需要「替换 agent loop」级别的灵活性——dsh 的差异化上限（loop 可替换）在 rick 现有需求清单中**零命中**。

**规划中未落地的定制需求**（谁满足）：

1. 双 runtime 等价（human Q4：CLI 用 pi、web 用 dsh）——pi/dsh 均需，seam 已预留（Runtime/RuntimeEnv/RuntimeBuilder 三接口），dshRuntime 未写。[internal/builder/xxxxbuilder.go; rick-spec §3.5]
2. 判断力度量采集（human-loop 判断→dream 回收）——数据源已存在（judgment.md 文本、meta.json toolCount/durationMs/cost、父会话 JSONL 工具序列、tasks.json），**pi 满足（离线脚本，无需 runtime 定制）**；dsh 侧 session log 插件可换可自定义采集（理论更灵活，未实测）。[research-S-r2 ②]
3. 触发概率埋点——pi-subagents artifacts 已含 usage/toolCount，离线可算，无需定制。
4. 主观评价采集（每任务好/差、验收质量分）——需 rick 层新增（judgment/loop 结构），**runtime 无关**。
5. 工具出错次数/异常路径/纠正耗时——工具出错可从会话 JSONL isError 字段派生（本轮已验证）；debug skill 调用次数不可直接派生（无 skill 一等事件，见 ⑤ 表/⑥-4）。

## ④ human 判断力度量方法学（human 请求「怎么评价人的判断」）

成熟方法七种对照（细节与来源见 leaf-D）：

1. **Tetlock/GJP**：IARPA 锦标赛以 Brier 分对「概率+截止日+可判对错」问题排名；超级预测者=前 2% 且跨年稳定；CHAMPS 去偏差训练（<1h）提升 Brier 6–11%。数据要求：每人每期数十~上百条可解决预测。映射：rick 拍板须转成「概率+截止日+可验证命题」。[Mellers et al. 2014]
2. **Brier/校准曲线**：二次严格 proper 规则（诚实报概率最优策略）；分解=校准+分辨力+不确定度；可靠性图每档需数十条。human 的「正确性归一化得分」可直接用 2(p−y)² 或对数分——**前提是拍板存数字概率**。[Gneiting & Raftery 2007]
3. **预测市场赔率**：押反共识方向且命中，回报 ∝ (1−p)/p——human 的「赔率」项有标准数学对应，无需新发明。[Hanson 2002 LMSR]
4. **Dalio believability 加权**：可信度=既往战绩×能讲清因果逻辑，桥水专有无公开公式；单用户场景退化为「用本人历史准确率给未来判断置信度加权」，依赖②③先行累计。
5. **DQ 六要素**（Spetzler/SDG）：评**决策时点的过程质量**，无结果数据也可打分（0–10 快检）；映射 dream 复盘的过程侧量表，防「好过程坏结果」被误罚。[sdg.com]
6. **AAR 四问/联想复盘四步**：单事件可执行、零样本量要求；与 rick dream 复盘结构一一对应——「判断→结果回收」闭环最直接的方法论载体。[TC 25-20]
7. **VoI/EV**：「影响力=历史价值+未来期望值」对应 EV 加权，但需显式决策树建模（备选/概率/效用），文本拍板无法直接算。

**框架级事实**（leaf-D 数学分析）：human 候选公式的「正确性×赔率」两项有成熟数学对应（proper scoring/博彩回报），但**三项连乘**偏离两类成熟范式（proper 分数只能是所报概率×结果的函数；EV 是加性结构），且乘积形式激励虚报影响力；标准做法是「分数（proper）+影响力（EV）」分列而非乘积。

**rick 现状对照（本地验证）**：判断语料=6 个 judgment.md（687 行，human 原话+拍板文本，**无数值概率/截止日字段**）；dream 回收窗口 1–2 周（短于 GJP 典型期限，命题设计可行）；按 GJP 口径（每人每期数十~上百条）当前判断密度低 1–2 个数量级。所有成熟方法共同前置=拍板结构化（概率+截止日+可验证+影响力预估）——属 rick 层格式扩展，runtime 无关。

## ⑤ AI 对 human 个性化体验指标度量（human 请求）

方法学（细节见 leaf-E）：

- **per-user 偏好学习**：P-RLHF（每用户独立 user model）[arXiv:2402.05133]、变分偏好建模 [arXiv:2408.10075]、reward 特征线性组合 [arXiv:2503.17338]、SynthesizeMe 从交互历史合成 persona [arXiv:2506.05598]——共同数据要求=**用户交互历史**，rick 已有（会话 JSONL/artifacts）。
- **NPS/CSAT 局限**：群体横断面设计；任意截断（6/9）、信息丢弃、差值方差大 [JBR 2022; arXiv:1806.10452]；回收率 10–25% 自选择偏差；主观分与行为绩效弱关联——**单用户单次打分无统计意义，不适合 rick N=1 纵向**。
- **单用户纵向统计**（匹配 rick）：Beta-Binomial 好/差比率追踪+贝叶斯收缩（防单次满分置顶）；SPRT 序贯检验（允许 peeking/提前停止）；贝叶斯 N-of-1；**SCD 单被试设计：WWC 证据标准要求 ≥4 个 A/B 相位（ABAB）方可因果推断**；交替处理设计=单被试内快速 A/B 两种配置。
- **任务无关指标+难度归一化**：IRT/任务级 psychometrics 预测单任务成败 [arXiv:2604.00594; 2607.15190]；TASTE 程序化生成控制难度覆盖 [arXiv:2605.28556]；反面实例：τ-bench 平凡 agent 无领域知识可过 38%。

**human 四指标 → rick 数据源映射（本地已验证）**：

| human 指标 | 数据源 | 状态 |
|---|---|---|
| ①主观好/差 | judgment.md 拍板（Beta-Binomial 纵向追踪） | 需新增结构化字段（rick 层） |
| ②工具出错次数 | 会话 JSONL 工具结果事件 isError 字段 | ✅ 已验证可派生 |
| ②debug skill 调用次数 | JSONL 无 skill 一等事件（仅文本内出现）；jobs/*/doing/debug/bug*.md 现为 0 文件 | ❌ 未消解 |
| ②异常路径/纠正耗时 | transcript 工具序列失败-修复循环（dream 模板已按报错/重试排序读取） | 部分可派生（需解析器） |
| ③任务总工时 | meta.json durationMs 汇总+父会话时间戳 | ✅ 已验证 |
| ④验收质量评价 | meta.json acceptance.childReport（criteriaSatisfied/runtimeChecks/evidence 已自动采集） | ✅ AI 侧已有；human 侧需新增 |

「dream 中真实度量」可行设计模式（方法组合，事实性）：SCD 相位设计（配置变更=干预，≥4 相位）× SPRT 序贯检验 × 行为指标（isError/durationMs）辅助主观分（NPS/CSAT 与行为弱关联的教训）× Beta-Binomial 纵向追踪单用户比率。

## ⑥ 未消解项

1. **①层清单非穷举封闭**：pi 生态（~4,300 包）无全文索引，未被 pi.dev 目录收录的 Codex/Claude Code 子代理扩展无法穷举排除；nicobailon pi-subagents 子代理进程模型细节（README 仅「focused child Pi session」）未核实。[leaf-A]
2. **pi 是否原生并行执行工具调用**未核实（dsh 有 fail-closed 并行调度器）。[leaf-A]
3. **dsh 部分事实来自社区站点**（dsh-in-depth.com/dshdocs.com）：ACP server 细节、schedule 会话复活续跑未在官方文档直接复核。[leaf-A]
4. **debug skill 调用次数不可直接派生**：pi 会话 JSONL 无 skill 一等事件（skill 名仅出现在文本/工具输出中）；jobs/*/doing/debug/bug*.md 现为 0 文件（模板有、实践未产生）。
5. **「赔率/反共识」的单用户操作缺口**：预测市场以群体概率为共识基准，rick 单用户无群体基准——需引入外部基准（如 AI 判断分布/社区共识数据源），此为设计决策非事实可消解项。
6. **dsh 侧度量采集能力未实测**：session log 是插件理论上可自定义采集（pi 侧需第三方扩展 appendEntry 或离线脚本），本地无 dsh 无从验证（R7-5 spike 范畴）。
7. dsh 文档 API 表面积无量化可比口径（文档页数/API 端点数无统一统计）。

## 叶子文件

- leaf-A（dsh 独有能力三层清单）：briefs/research-extra-2-leaf-A.md
- leaf-D（判断力度量方法学七法）：briefs/research-extra-2-leaf-D.md
- leaf-E（个性化体验指标方法学）：briefs/research-extra-2-leaf-E.md
- 本简报 B（代价）自查来源：dshdocs.com changelog/FAQ、D Discussions #3691/#4175、deepseek-harness.github.io、pi.dev/packages 目录（MCP/沙箱/审批/Web UI/后台任务扩展存在性均经 pi.dev/packages 与 npmjs 双重核实）；C（rick 定制需求）全部来自本地代码。
