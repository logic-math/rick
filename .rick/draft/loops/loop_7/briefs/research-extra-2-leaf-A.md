# Research: dsh 能做而 pi 做不到/较差的能力差异（尽调事实）

D=github.com/deepseek-ai/deepseek-harness；P=github.com/earendil-works/pi。基线已核实，仅列差异证据。

## ① pi 核心+官方生态+第三方扩展均做不到
1. **运行时替换 agent loop**：dsh 文档明确 agent loop 本身是插件、可从配置替换；pi 扩展 API 无主循环替换接口，深层改动需 fork。[D:docs/architecture.md；P:packages/coding-agent/README.md]
2. **跨产品子代理后端**：dsh 第一方后端可共存：inprocess spawn/fork、ACP、Codex、Claude Code；pi-subagents 子代理皆为 pi 会话，无已知扩展提供 Codex/CC/ACP 子后端（pi-acp 方向相反，系 pi 被编辑器驱动）。[D:packages/subagent/README.md、.agents/notes/implemented/feature/2026-08-04-claude-code-and-codex-subagent-backends.md；github.com/nicobailon/pi-subagents]
3. **三平台内核沙箱+fail-closed**：dsh 覆盖 Linux bwrap→Landlock、macOS Seatbelt、Windows ACL，不可用即 SANDBOX_UNAVAILABLE 拒执行、绝不静默放行；pi 生态沙箱仅 macOS/Linux 且有 fail-open 实现（见②）。[D:packages/sandbox/sandbox-local/README.md]

## ② pi 靠第三方扩展可部分做到但有明确限制
1. **OS 沙箱**：@nqbao/pi-sandbox（macOS/Linux）、rnorth/sandboxed-pi（Docker）、官方示例；限制：无 Windows ACL/Landlock；@jerryan/pi-bash-wrap 明示非 Linux 直接放行（fail-open）。[npmjs @nqbao/pi-sandbox、@jerryan/pi-bash-wrap；P:examples/extensions/sandbox/index.ts]
2. **子代理编排**：pi-subagents 支持异步/并行/worktree/链式/后台作业；限制：单后端、传输不可换、无跨产品路由。[github.com/nicobailon/pi-subagents]
3. **MCP**：pi-mcp-adapter（590K/月下载）、pi-mcp-extension、ElieMessieCode/pi-mcp 均可用；皆第三方，dsh 为第一方 dsh-mcp-client（原生 mcp__server__tool 命名、effect-scoped 断连）。[pi.dev/packages/pi-mcp-adapter；D:packages/mcp/mcp-client/README.md]
4. **ACP 服务端**：pi 经社区 pi-acp（MVP，自述部分 ACP 特性未实现）入 Zed Registry；dsh 第一方 ACP server+按策略机器审批。[github.com/svkozak/pi-acp；P Discussion #4444；dsh-in-depth.com/llm-platform/acp]
5. **审批拦截**：pi 可经 tool_call 拦截 block/改参实现审批；限制：无统一服务/审计/ask-never 策略瀑布，各扩展自行实现；dsh ctx.approval fail-closed+审计对。[P:pi.dev/docs/latest/extensions；D:docs/subsystems/approval.md]
6. **定时任务**：第三方 pi-scheduler 等仅活动会话内生效；dsh-schedule 持久于 session event log、会话复活续跑。[npmjs pi-schedule-prompt、github.com/manojlds/pi-scheduler；D:packages/schedule/schedule/src/index.ts]

## ③ pi 可做到但 dsh 更优（客观依据）
1. **Web UI/自托管**：pi 有社区 pi-web/pi-webui；dsh 第一方 `npx dsh web`（127.0.0.1:3080、SSH 模式、--no-open）。依据：第一方维护 vs 社区分裂多实现。[pi-web.dev；D:README.md]
2. **上下文管理粒度**：pi 可全换 compaction（官方示例+reserveTokens 等）；dsh：LLM 历史由事件 log 派生、压缩经 surfaceOp+sourceEventSeqs 重放校验、事件类型 merge-extensible、ctx.sessionQuery 全文检索；pi 无对应检索（未消解）。[P:docs/compaction.md；D:reference/subsystems/session、.agents/notes/2026-06-18-compaction-capability-seam.md、packages/session-query/session-query/README.md]
3. **插件生命周期与组合**：pi /reload 全量热重载；dsh 单插件可逆 effect 卸载回滚+bundle/profile/patch 组合树+--dump-config 校验。[P:pi.dev/docs/latest/extensions；D:docs/architecture.md、apps/cli/README.md、docs/user/develop/basic/publish.md]
4. **子代理进程模型**：dsh spawn-in-process 同进程建 Agent（源码注释"cheapest transport"）；pi-subagents 每子代理独立 pi 会话。反证：dsh web host 在 fan-out 下 TTFT 中位 49.7s/p90 142s（#3235），非全面占优。[D:packages/subagent/subagent-spawn-in-process/src/index.ts、Discussion #3235]
5. **模型/后端切换**：同源——dsh llm-pi-ai 适配器直接复用 pi-ai（多 provider+OpenAI 兼容端点），无实质差异，非 dsh 独有。[D:packages/llm/llm-pi-ai/README.md]

## 未消解
未见收录的 pi Codex/CC 子代理扩展；nicobailon 版子代理进程模型细节；pi 是否原生并行工具调用（dsh 有 fail-closed 并行调度，Notes 2026-07-10/08-09）；dshdocs/dsh-in-depth/dsh-plugin.net 为社区站，个别事实（ACP server 细节）未经官方文档直接复核。
