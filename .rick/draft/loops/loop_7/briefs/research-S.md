# research-S 简报 — pi vs dsh 作为 rick runtime 的事实基础

> 阶段：S 问题确认（loop_7 第 1 轮）| 时间基准：2026-08-24 | 方法：本地代码尽调 + 3 叶子联网调研（leaf-1 dsh 架构 / leaf-2 生态活跃度 / leaf-3 runtime 职责对照）
> 硬约束：仅事实+前提+来源；「已验证」= 本地代码或官方文档直接证据，「未消解」= R7 上报。

## ① rick 现状（已验证，本地代码证据）

1. **定位**：Go CLI（go 1.25 + cobra + goldmark，仅 3 个直接依赖），「对抗上下文熵增的 AI Coding 控制框架」；pi 为受控执行后端，rick 自身是引导程序（bootloader），不直接调用大模型。[README.md]
2. **三层金字塔架构已落地**（loop_6 RFC O2 已完成）：`internal/{cmd,handler,env,builder,runtime,config,prompt,workspace}` 与架构图逐层对应；重构删除 6 个旧包（executor/parser/actpath/logging/git/agent），dag 调度与门禁下沉 pi（workflowScript 编排 + rick-gates hook）。版本 v4.4.15，最新提交 728bd4a（API key 泄漏修复）。[git log; internal/ 目录]
3. **pi 集成方式**：`runtime.Runtime` 接口（Name/Run）唯一实现 piRuntime——pi 二进制 + `--mode json` JSONL 事件流解析（agent_settled 终止信号）+ 方法层与实例层提示词均走 `--append-system-prompt`（compaction 持久，user 消息仅 bootstrap）+ `PI_CODING_AGENT_DIR=~/.rick/pi/agent` 配置目录隔离 + 自闭环运行时副本（stock 不 patch）。[internal/runtime/runtime.go, executor.go]
4. **env 四职责落地**：pi/扩展（pi-subagents、pi-web-access）安装更新 + rick 自有定制（think/research/exporter 三 agent：fullTools 全量工具 + thinking high + timeoutMs 3600000，`agents/{name}.md` frontmatter 含 systemPromptMode: replace 与 rick-managed: true 覆盖标记）+ 就绪 check。[internal/env/agents.go]
5. **loop_6 性能改造已落地**（task11）：自然语言 subagent 触发词已等价迁移为 pi 显式触发语法——sense_loop.md 触发权归属条款「每次派发必须用 subagent({workflowScript:...}) + runs.run/runs.all + 真实 agent 名（agent:'think'/'research'/'exporter'），不再用自然语言描述触发动作」；plan/doing/dream 模板均含显式派发示例与 timeoutMs 规范；task12 三 O 端到端验收通过。[git log 23d31cc; internal/prompt/templates/sense_loop.md L28]
6. **spec 信息内核已落地**（O1）：`.rick/domain/spec.md`（四要素模板：模块边界/职责/接口契约/验收标准）+ `.rick/domain/rick-spec.md`（四层架构图含 dsh 预留位）。[.rick/domain/]
7. **dsh 扩展 seam 已预留未实现**：Runtime/RuntimeEnv/RuntimeBuilder 三接口 + config.runtime 字段（默认 pi）；rick-spec 明确「dsh = deepseek harness 预留 runtime，当前不写代码」；切换规则 = 只新增 dshRuntime/dshEnv/dshBuilder 并注册，cli/handler/templates 不改。[.rick/domain/rick-spec.md §3.5]
8. **托管 pi 版本**：0.84.2（MIT）。[~/.rick/pi/agent/runtime/node_modules/@earendil-works/pi-coding-agent/package.json]
9. **未量化**：loop_6 subagent 触发概率提升无量化数据——loop_6 judgment 明确「改正后实测」由 human 承接，目前 rick 仓库内无触发概率度量代码/埋点。[loop_6/judgment.md; 本地检索无果]

## ② pi 画像（本地文档已验证 + leaf-2 联网核实）

1. **定位**：「minimal terminal coding harness」，TypeScript/Node，MIT 许可证，npm 包 @earendil-works/pi-coding-agent，作者 Mario Zechner（badlogic，earendil-works），官网 pi.dev。[本地 README.md + package.json L99]
2. **哲学（定制化边界的关键前提）**：核心极简 + 激进可扩展——「No MCP / No sub-agents / No permission popups / No plan mode / No built-in to-dos / No background bash」，这些全部经 extensions/skills/packages 构建；「Adapt pi to your workflows...without having to fork and modify pi internals」。[README Philosophy 节]
3. **定制面四件 + 分发**：Extensions（TS 模块：registerTool 自定义工具/registerCommand/事件拦截 tool_call 可 block/自定义 compaction/自定义 UI 与渲染/appendEntry 会话持久化）、Skills（agentskills.io 标准，/skill:name 或模型自动加载）、Prompt Templates、Themes（仅 51 个颜色 token）；Pi Packages 经 npm/git 分发。[README; docs/extensions.md]
4. **4+1 运行模式**：interactive / print(-p) / --mode json（JSONL 事件流）/ --mode rpc（stdin/stdout LF-JSONL 协议，供非 Node 集成）/ SDK（createAgentSession 嵌入式）。[README; docs/sdk.md, rpc.md]
5. **子代理能力非核心**：pi 无内置 subagent，rick 依赖第三方扩展 pi-subagents（env 安装注册）实现 fanout 派发——rick 的深度定制实际踩在「核心扩展 API + 第三方扩展」两层上。[README Philosophy; internal/env/agents.go]
6. **rick 实测的 known limitations**：pi 不读 PI_PROVIDER/PI_MODEL/PI_API_KEY 环境变量（必须 CLI flags）；主题只能改颜色、渲染行为（diff 反显）不可主题化（无配置口，改行为须改 dist JS，rick 已否决 patch 路线）；pi install 对 user scope 包可能共享全局 npm root 代码（卸载互相影响）；strict 工具校验曾在 0.51.0 因列出未加载扩展的工具硬失败。[.rick/domain/pi-runtime.md]
7. **provider 覆盖**：内置 30+ provider（Anthropic/OpenAI/DeepSeek/Gemini/Bedrock/OpenRouter 等 + 订阅登录 + llama.cpp 本地路由）。[README Providers 节]
8. **活跃度与生态（2026-08-24 核实）**：GitHub（earendil-works/pi，原 badlogic/pi-mono，建仓 2025-08-09，约 1 年）95,913★ / 11,864 forks；npm 周下载 2,173,217（第三方口径 1.4-1.65M）、550 dependents、累计 254 版本，最新 0.84.2（2026-08-14，与 rick 托管版一致），发布频率约每周 1-3 版；第三方 pi-package ≈4,300 个（pi.dev/packages 目录）；pi-subagents 与 pi-web-access 均由 nicobailon 维护且活跃（另存在 @tintinweb/pi-subagents 948★ 同名实现）；bus factor=1（badlogic 占 69% commits）；新贡献者 issue/PR 默认自动关闭、维护者每日复查。[github.com/badlogic/pi-mono; npmjs.com; pi.dev/packages; inspect.software]

## ③ dsh 画像（联网调研，web_search 多信源交叉，截至 2026-08-24）

1. **身份确认**：dsh = DeepSeek Harness，GitHub deepseek-ai/deepseek-harness，DeepSeek AI 官方开源，2026-08-13 与 DeepSeek V4-Pro 同日发布；CLI 名 dsh，npm 包 @deepseek-ai/dsh，入口 `npx @deepseek-ai/dsh web`（启动 Web UI）。[github.com/deepseek-ai/deepseek-harness; deepseek.com/harness/en/]
2. **架构**：「Everything is a Plugin」——插件提供全部 agent 能力：models、tools、skills、sessions、sandboxes、storage、loops、scheduling、UI，全部可换可重组；Cordis 内核 vendored 进 monorepo（@deepseek-ai/cordis，自持框架层，设计论文《A Programming Paradigm for Spatiotemporal Composability》）管理插件挂载/卸载/依赖；「running dsh = 插件树，boot 时由 ordered layers 组合」（profile/bundle/patch 分层覆盖，dsh-base 最先应用，细节见 leaf-1）。[deepseek-harness.github.io/en/reference/; deepseek.com/harness/en/]
3. **无特权核心**：官方架构文档原话「There is no privileged core to patch: you extend dsh by mounting a plugin」——model adapter、tool registry、session log、agent loop 本身都是插件，全部可从配置替换。这是与 pi「核心内置 + 扩展加能力」的关键结构差异。[deepseek-harness.github.io/en/reference/]
4. **插件 API 面**：插件 = Service 对象（ESM named-export apply(ctx)）；插件贡献 services（直接调用）、typed events（tool 结果/model 请求/approval 决策）、reversible effects（注册随插件卸载自动回滚，外部资源须包 ctx.effect()）；cordis.yml 列插件条目（并发启动）。[deepseek-harness.github.io/en/develop/]
5. **能力词汇（较 pi 核心更广）**：subagent（ctx.subagents 多后端可换传输）、workflow 引擎（goal rounds + Fresh-agent Ralph）、user approval（fail-closed）、sandbox、MCP、planning（goals/jobs/todo_write）、JSONL 持久化。[dsh-in-depth.com; reference/subsystems/; examples/headless-agent]
6. **实现语言与要求**：TypeScript/Node.js（^22.19.0 或 >=24.0.0）。[deepseekdocs.com quickstart]
7. **许可证**：MIT。[github repo 元数据]
8. **活跃度**：开源 2 天 96.8k stars（第三方博客 0xran.com 记录），当前 189,495 stars / 21,139 forks（GitHub 快照）；7,090 commits / 34 committers（Ecosyste.ms），top 贡献者 tianyicui 5268 / LegGasai 1506 / imccyu 1262；releases 全为 pre-release，最新 v0.1.1-rc.2（2026-08-21；此前 rc.5 08-13→rc.7 08-17→rc.8 08-19→rc.1 08-21）。[0xran.com/en/blog/dsh-vs-pi/; GitHub releases; summary.ecosyste.ms/projects/378897]
9. **成熟度（重要限制）**：官方声明「DeepSeek Harness is currently in developer preview and is iterating rapidly. THERE WILL BE COMPATIBILITY-BREAKING CHANGES.」；版本 0.1.x-rc 阶段（npm latest 仍指 0.1.0-rc.7，rc.8 在 next tag 且含 breaking change）；各包 README Known Limitations：session 分支树推迟、fork() 仅限活跃会话稳定边界、ACP 仅 fresh session、subagent SDK 每次运行新建进程（无池化）、沙箱后端不可用即 fail-closed、Node<22.19 曾现含混 ESM 崩溃（Discussion #2327）；已知 bug 如 Web UI 多轮会话丢 reasoning blocks（Discussion #231）。[github repo 及 packages/*/README; dshdocs.com; github discussions]

## ④ runtime 职责边界对比（leaf-3 完成，完整表见叶子文件）

| 维度 | pi（本地 docs 已验证） | dsh（官方文档） |
|---|---|---|
| 会话管理 | 树形 JSONL（id/parentId 分支）+ /new /resume /fork /clone；SDK 会话替换 | append-only SessionEvent 日志=唯一事实源；SessionPersistence seam（JSONL 后端+checkpoint+崩溃恢复） |
| 上下文管理 | 内建 compaction（阈值自动+手动 /compact，reserveTokens 默认 16384）；扩展可自定义摘要 | compaction 为可选 capability seam（ctx.compaction）；BasicCompactionEngine（token 预算+KV cache 复用） |
| 工具执行 | 内建 7 工具；扩展 registerTool；tool_call 钩子可 block/改参 | 工具 registry 本身是插件（ctx.tools）；bash 执行独立 seam，多执行器（local/sandbox/terminal） |
| subagent | **无内建**（SDK 文档明示由调用方建子 AgentSession；rick 实际依赖第三方 pi-subagents） | 内建 ctx.subagents 多后端注册（inprocess/acp/codex/claude-code，传输可换） |
| 扩展机制 | TS 扩展（jiti 免编译），事件钩子覆盖全周期；**核心不可替换** | Cordis 一切皆插件（model adapter/tool registry/session log/agent loop 均可配置替换）；**无特权核心** |
| 无头集成 | --mode json（单向事件流）/ --mode rpc（双向）/ SDK 同进程嵌入 | npx dsh web（Web UI）/ dsh --profile headless "task"（headless-runner 插件）/ profile 组合层 |
| MCP | 官方 6 份文档未提及（哲学：CLI tools+README 替代） | @deepseek-ai/dsh-mcp-client（mcp__server__tool 命名，stdio+HTTP） |
| 沙箱/审批 | 无内建（扩展做 permission gate + project_trust 流程） | ctx.approval seam（ask/never 策略+fail-closed）+ dsh-sandbox-local（bwrap/Landlock/Seatbelt/ACL） |
| Planning | 无内建（扩展/TODO.md 模式） | ctx.goals（跨轮目标）+ ctx.jobs（后台任务）+ todo_write 工具 |

结构性差异总结（事实层面）：pi = 「固定核心 + 扩展加能力」（扩展能拦截/增加，不能替换核心 loop）；dsh = 「插件组合即产品」（核心本身也是插件，可从配置整体替换）。

## ⑤ 对比条件核对（loop_6 切换前提 × 两项目现状，仅事实）

**前提 1「新 runtime 带来更强大的生态」**：
- pi 生态：≈1 年，254 版本、npm 周下载 2.17M（第三方口径 1.4-1.65M）、550 dependents、≈4,300 个第三方 pi-package。
- dsh 生态：11 天，189,495★（采用指标存疑，见 R7）、open issues=0、GitHub Discussions + DeepSeek Discord；**未发现第三方 dsh 插件目录或规模统计**；官方文档薄，第三方站（deepseekdocs/dshdocs/dsharness）补位。
- 事实结论：按「可复用第三方扩展存量」口径，pi 目前领先（4,300 vs 未发现）；按「关注度/增长」口径，dsh 领先（11 天 189k★ vs 1 年 96k★）。

**前提 2「更强的可定制性」**：
- dsh 结构性更强：无特权核心（「There is no privileged core to patch」），model adapter/tool registry/session log/agent loop 全部可配置替换；subagent（多后端）/MCP/沙箱（三模式+多平台内核）/审批/planning 内建为插件。
- pi 定制边界：扩展可注册工具/命令/事件拦截（block/改参/改结果/自定义 compaction/UI/渲染），但**核心 agent loop 与默认骨架不可整体替换**；subagent/MCP/沙箱/审批均非内建。
- 事实结论：定制深度上限 dsh 更高（可替换 loop 本身 vs 只能拦截钩子）；但 dsh 处于 0.1.x-rc developer preview，官方明示将有破坏性变更，pi 已连续发布 254 版本。

**「短期性能提升」在 rick 语境的可量化抓手（代码事实）**：
- subagent 触发概率（loop_6 主题）：显式触发语法已落地，但无埋点无量化——可从 raw_session_coding.log（pi JSONL，含 tool_execution 事件）与 trace.ToolCalls 统计派发次数，rick 现无此统计代码。
- 已有可测产物：tasks.json（状态/重试）、Trace（工具调用数/错误数/时长/Settled）、debug-summary.md、dream 淘汰计数（loop/skill 连续 3 次未触发）。
- 已调参数（已做非量化优化）：max_retries=5、agent thinking=high、timeoutMs=3600000、--append-system-prompt（compaction 持久）、hideThinkingBlock、pi_extra_args 模型选择。
- loop_6 判断原文确认：「目前确实缺少量化评估，当前都是靠直觉进行优化」。
- 唯一 adjacent 基准：Composio harness benchmark 中 Oh My Pi（pi 硬 fork）以 88% pass rate 第一（第三方数据，非 dsh vs pi 直接对比）。[0xran.com]

## R7 未消解项（上报）

1. dsh 第三方插件生态规模无可靠数字（11 天，未发现第三方插件目录统计）——缺 dsh 插件市场/索引数据。
2. dsh 189k★ 的实际采用含金量无法澄清（star 增速异常有讨论无定论；VentureBeat 称「数字只是快照、非采用指标」）。
3. pi 与 dsh 的 Discord 社区成员数均无公开数字。
4. loop_6 subagent 触发概率提升效果未量化（human 承接实测未开始，rick 无埋点代码）。
5. dsh 与 rick 集成可行性未实测（本地未安装 dsh；dsh --profile headless 一次性运行模式未验证）。
6. pi 与 dsh 无直接可比的执行性能/质量 benchmark。
7. npm 周下载数口径差异大（1.4M–2.2M），未用 registry API 实时核实。
8. dsh 7,090 commits 含 vendored Cordis 历史，净新增 commit 数无法拆分。

## 叶子文件

- leaf-1（dsh 架构/定制化/成熟度）：briefs/research-S-leaf-1.md
- leaf-2（生态/社区/活跃度对比）：briefs/research-S-leaf-2.md
- leaf-3（runtime 职责边界对照表）：briefs/research-S-leaf-3.md
