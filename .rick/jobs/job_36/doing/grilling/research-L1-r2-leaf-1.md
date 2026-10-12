# pi examples 本地考察（research-L1-r2 leaf-1）

基路径 `$PI=node_modules/@earendil-works/pi-coding-agent/examples/`（v0.84.2）。extensions/README.md 与 sdk/README.md 官方含每示例一句话描述，以下为浓缩+深读结果。

## 一、示例清单

**examples/ 顶层**：rpc-extension-ui.ts → 独立 TUI 聊天客户端，spawn `pi --mode rpc` 子进程，走全 JSONL 协议（含 extension UI 对话框桥接）。

**examples/sdk/**（13 个，演示 createAgentSession 编程面）：01-minimal 全默认；02-custom-model 选模型/thinking；03-custom-prompt 改系统提示；04-skills 技能发现过滤；05-tools 工具白名单；06-extensions 事件拦截/改结果；07-context-files AGENTS.md；08-prompt-templates 文件斜杠命令；09-api-keys-and-oauth 密钥/OAuth；10-settings 压缩/重试覆盖；11-sessions 内存/持久/continue/list；12-full-control 全替换禁发现；13-session-runtime createAgentSessionRuntime 会话热替换（new/switch/fork 复用同一 runtime 工厂）。

**examples/extensions/**（73 项，按主题归组）：
- 安全门禁：permission-gate（危险 bash 确认）、protected-paths、confirm-destructive、dirty-repo-guard、project-trust、sandbox/（@anthropic-ai/sandbox-runtime OS 级沙箱，覆盖内置 bash）、gondolin/（QEMU 微 VM 内跑全部内置工具）
- 自定义工具：todo、hello（最小）、question、questionnaire、tool-override、dynamic-tools（session_start 后注册）、kimi-deferred-tools、structured-output（terminate:true 结束轮次）、built-in-tool-renderer、minimal-mode、truncated-tool（包装 rg+50KB 截断）、ssh.ts、subagent/
- 命令与 UI：preset、plan-mode/（只读模式+步骤跟踪）、tools、handoff、qna、status-line、github-issue-autocomplete（预载 gh issue 列表做补全）、widget-placement、hidden-thinking-label、working-indicator、model-status、snake、tic-tac-toe、space-invaders、send-user-message、timed-confirm、rpc-demo.ts、modal-editor、rainbow-editor、notify（OSC777/99 桌面通知）、titlebar-spinner、summarize、custom-footer、custom-header、border-status-editor、overlay-test、overlay-qa-tests、doom-overlay/（35FPS 游戏 overlay）、shutdown-command、reload-runtime、interactive-shell（user_bash 挂起 TUI 跑 vim）、inline-bash、input-transform(-streaming)、commands、bash-spawn-hook（createBashTool spawnHook 改 cmd/env）
- 提示/压缩/资源：pirate、claude-rules、custom-compaction、trigger-compact、prompt-customizer、system-prompt-header、mac-system-theme、dynamic-resources/（resources_discover 动态注入 skill/prompt/theme）、message-renderer、entry-renderer、event-bus（pi.events 总线）、session-name、bookmark
- Git：git-checkpoint、auto-commit-on-exit、git-merge-and-resolve
- Provider：custom-provider-anthropic/（自定义 streamSimple+OAuth，fetch 仅作 LLM 客户端）、custom-provider-gitlab-duo/
- 其他：with-deps/（扩展自带 npm 依赖）、file-trigger、provider-payload（before_provider_request 落盘）、working-message-test

## 二、集成类示例实现模式详解

**关键否定性结论：全部 examples 无一启动 HTTP/websocket server，无端口无鉴权。** 外部集成全部走「spawn 子进程」或「文件/事件钩子」；pi 官方对外集成范式 = rpc/json 子进程 + extension 事件钩子。

1. **rpc-extension-ui.ts（$PI/rpc-extension-ui.ts）— 外挂 UI 参考架构**
   - `spawn(node,[cli.js,--mode,rpc,--no-session,--no-extension,--extension,rpc-demo.ts],{stdio:[pipe,pipe,pipe]})`；stdin 写 JSONL 命令，stdout 逐行读事件
   - 启动探活：spawn 后 500ms 查 `exitCode!==null`，stderr 缓存用于报错；退出走 SIGTERM
   - 双向桥：收 `extension_ui_request{id,method}`（select/confirm/input/editor）→ UI 渲染 → 回写 `extension_ui_response{id,value|confirmed|cancelled}`；notify/setStatus/setWidget/set_editor_text 为单向
   - 注意 rpc.md：Node readline 按 U+2028/29 切分不合规，须自实现 `\n` 分帧（Go bufio.Scanner 无此问题）
2. **subagent/index.ts — pi 官方子进程管理范本（最成熟）**
   - 每任务 `pi --mode json -p --no-session` 一进程，按配置追加 `--model/--thinking/--tools/--append-system-prompt 临时文件`
   - JSONL 手工分帧：buffer+=data→split("\n")→残尾保留→逐行 parse（吞错）；聚合 message_end usage
   - abort：AbortSignal→SIGTERM→5s→SIGKILL；MAX_PARALLEL_TASKS=8 / MAX_CONCURRENCY=4；单任务输出 50KB 上限
   - `getPiInvocation()`：dev 用 execPath+脚本，否则裸 `pi` —— rick Go 侧同样需二选一解析 pi 可执行路径
3. **ssh.ts — 工具执行位置整体替换范式**：createRead/Write/Edit/BashTool(cwd,{operations}) 可插拔 ops 把四类工具委托给远程 `spawn("ssh",[remote,cmd])`（免密 key）；证明工具层可整体远程化
4. **file-trigger.ts — 无子进程的外部注入**：session_start 里 `fs.watch` 触发文件 → `pi.sendMessage({customType,display:true},{triggerTurn:true})` 注入并触发 LLM 轮次；读后清空
5. **event-bus.ts / dynamic-resources/**：pi.events.on/emit 扩展间总线；resources_discover 返回路径动态注入资源

## 三、【结论】
1. pi 示例零 HTTP/WS server 先例，web 服务须 Go 侧自建，pi 只能经 rpc 子进程桥接（与 rick 既定架构一致）。
2. subagent+rpc-extension-ui 给出完整子进程生命周期范本：JSONL 分帧、SIGTERM→5s→SIGKILL、启动探活、并发/输出上限，可直接移植 Go。
3. extension UI 请求经 extension_ui_request/response 走 stdio，web 端必须实现该双向对话框桥才能承载 human-loop 会话。
