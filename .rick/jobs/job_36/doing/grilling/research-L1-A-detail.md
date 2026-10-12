# research-L1 附录 A：pi 生态支撑能力详录（供简报引用）

元信息：L1 总体架构层调研 | 时间基准 2026-08-29 | 调研员 research（fanout：3 叶子 + 自查一手文档）

## A. pi 生态对 web UI 的支撑能力

### A1. pi SDK 能力边界（置信度：高 0.9）
- createAgentSession/ModelRuntime/SessionManager/DefaultResourceLoader 均为公开导出，明确以「Build a custom UI (web, desktop, mobile)」为官方用例。来源：docs/sdk.md（官方文档 0.2 + dist 导出源码 0.4）。
- 事件订阅 session.subscribe 覆盖：message_update（text/thinking/toolcall delta + 逐 token usage）、tool_execution_start/update/end（含 partialResult 累积输出）、message/turn/agent 生命周期、queue_update（steer/followUp 队列）、compaction/auto_retry/summarization_retry 全套。**没有独立 subagent 事件类型**——子代理以 subagent 工具的 tool_execution_* 呈现；子代理细粒度轨迹落在 `extensions/subagent/runs/<runId>/events.jsonl`（wrapper 事件 + child Pi JSON 事件带 run/step 元数据），可通过 pi.events 总线或读文件获取。来源：docs/sdk.md + pi-subagents/docs/observability.md（本地一手）。
- ResourceLoader 加载 extensions/skills/prompts/themes/context files 全支持（additionalExtensionPaths/extensionFactories/skillsOverride/themePaths），可逐 session 定制。来源：docs/sdk.md（官方文档）。
- **多 session 并发**：无官方"multi-session"字样声明，但架构上每个 createAgentSession 返回独立 session（独立 SessionManager/订阅），无文档化单例；官方 RPC/SDK 对比段写明 SDK 适用于 same-process embedding。已知进程级共享点：① http-dispatcher 幂等替换 globalThis.fetch（undici 连接池，idle timeout 取全局 settings——不同 session 无法各配）；② ModelRuntime 可多 session 共享（设计如此）。判定：**一进程多并发 session 可行（置信度中高 0.7）**，残余风险=无官方并发多 session 的先例佐证（官方 mode 均单活动 session），需原型验证。
- AgentSessionRuntime（newSession/switchSession/fork/import）承担"替换活动 session"，订阅需重绑——多 job 场景应每 job 独立 AgentSession 而非共享一个 runtime。

### A2. pi --mode rpc（置信度：高 0.95）
- 协议：stdin/stdout JSONL（严格 LF 分帧，Node readline 不合规——U+2028/2029 坑）。命令全集：prompt（含 images/streamingBehavior）/steer/follow_up/abort/new_session/get_state/get_messages/set_model/cycle_model/get_available_models/set_thinking_level/bash（流式 bash_execution_update + id 关联）/abort_bash/get_session_stats/compact/switch_session/fork/clone/get_entries（**since 游标增量，跨客户端重启有效**）/get_tree/get_fork_messages/set_session_name/get_commands/export_html。
- 事件流：与 SDK 同源事件 + agent_settled + extension_ui_request/response 子协议（extension 的 select/confirm/input/editor 对话框桥接到客户端）——这是 web UI 做「扩展交互审批」的现成通道。
- vs SDK：rpc=进程隔离 + 语言无关（Go 可直接 spawn）+ 单 pi 进程单活动 session；SDK=同进程、类型安全、可程序化定制 tools/extensions、直接访问 agent state。SDK 官方定位为 Node 内嵌首选，rpc 为跨语言子进程首选。来源：docs/rpc.md（官方文档+协议规范）。

### A3. pi extension 能否承载 web UI（置信度：高 0.9）
- extension = TS 模块（jiti 加载）运行在 pi 进程内，可用 node:net/node:http 起 HTTP/WebSocket server——**技术上可行**。但官方明确：factory 内禁止起后台资源，须 defer 到 session_start 并注册 session_shutdown 清理（生命周期 = session 存活期 = pi 进程存活期）。
- pi 进程模型 = **一进程一活动 session**（interactive 单会话；/resume //fork 走 AgentSessionRuntime 替换）。⇒ 「web UI 作为 pi extension」只能暴露**该 pi 进程内的那一个会话**，无法常驻承载 rick 多 job（多 job = 多 pi 进程）。要覆盖多 job 需每 job 一个 extension server（端口碎片化）或 extension 反向连接中央服务（extension 退化为 agent-side adapter，不是 web UI 宿主）。
- **判断成立**：extension 路线只适合"单会话远程查看/操控"（如 job 内 debug 视图），不能作为 rick web UI 的主架构。来源：docs/extensions.md 生命周期节 + AgentSessionRuntime 节（官方文档 0.2 + 0.4 源码结构）。

### A4. session 持久化与恢复（置信度：高 0.95）
- 格式：JSONL 树（v3），首行 session header（id/cwd/parentSession），条目 id/parentId 链接，支持 message/model_change/compaction（retainedTail 自包含检查点）/branch_summary/custom/custom_message/label/session_info。文件位于 `<agentDir>/sessions/--<path>--/<timestamp>_<uuid>.jsonl`。
- **断线重连恢复 = 现成能力**：rpc get_entries 支持 `since` 游标增量拉取（官方原文"durable cursor ... even across client restarts"）+ leafId 单往返判活；SessionManager.open/switch_session/fork 可重建任意会话；buildContextEntries 处理 compaction 截断。web UI 重连后可用 get_entries(since=lastSeenId) 精确补齐 + get_tree 恢复树视图。来源：docs/session-format.md + docs/sessions.md + rpc.md（官方文档）。
- rick 现状对照：doing 用 `--mode json` 一次性进程（agent_settled 终止），session 文件已由 pi 落盘（PI_CODING_AGENT_DIR=~/.rick/pi/agent 隔离）——历史 job 会话可离线重放。来源：internal/runtime/*.go + .rick/domain/pi-runtime.md（代码原文 0.4）。

## A 组小结（对 L1 判断节点的直接输入）
pi 生态对 web UI 的支撑是**完备的**：SDK（Node 常驻多 session）/ RPC（Go 子进程驱动单 session×N）/ session 文件重放三条通道都通。真正约束不在 pi 能力，而在宿主形态（extension 不可承载多 job 常驻，A3）与 rick 哲学（node 不进 rick 构建链）。
