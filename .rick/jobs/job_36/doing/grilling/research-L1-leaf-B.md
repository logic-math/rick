# Research: coding agent web UI 架构模式（B 组）

## 架构模式对比

- **OpenHands**：(b) headless。React SPA + Python 后端；agent 跑在 Docker Runtime（client-server），前端经 HTTP/WebSocket 订阅 EventStream（append-only 类型化事件日志）。[官方文档]
- **goose（block）**：(b)。Rust 核心；`goosed` 后端提供 REST + SSE 流式消息；Electron 桌面（Node 主进程拉起 goosed，React 渲染）+ CLI；官方明确支持 Web/Mobile 自建 UI 接 goosed。[官方文档]
- **aider**：(b)。`aider --browser` 实验性 Web UI，Python 内置 web server，自绘聊天界面，非终端转发。[官方文档]
- **claude-code-webui（sugyan）**：(b)。Node/Deno + Hono 后端调 Claude Agent SDK `query()`（无头进程产 stream-json），HTTP POST 返回 NDJSON 流；React 自绘消息/工具调用卡片，无 xterm.js；权限用 allowedTools/permissionMode。[源码]
- **opcode（winfunc）**：(b)。Tauri 桌面（Rust+React）；web 模式为 Rust Axum + WebSocket，spawn `claude` 子进程、解析 stdout JSONL 转 DOM 事件推给 React；web 模式曾直接 `--dangerously-skip-permissions`，权限交互是痛点；早期版本有会话串扰/无法取消进程等坑。[源码]
- **omnara**：(b)。开源 agent 托管平台：SDK 无头执行+状态管理，Web 仪表盘 + 移动端推送/审批，强调手机体验。[官方 README]
- **Claude Code on the web（官方）**：(b)。Anthropic 托管云沙箱运行，GitHub 仓库集成；会话云端持久化、Claude 手机 App 可监控；`--cloud/--teleport` 在 web 与终端间迁移会话。[官方文档]
- **终端路线实际案例**：ragent（xterm.js↔WebSocket↔PTY↔Claude CLI，Docker 隔离）、agentboard / agentdeck / mulmoterminal / web_terminal_acp（tmux + node-pty + xterm.js，主打手机/平板远程看 agent TUI）。[社区源码]

## PTY 终端转发细节

**可行性：成熟。** ttyd（C/libuv，内置 xterm.js，支持 CJK/IME）、wetty（Node+node-pty）、gotty（Go 19.5k star，WebTTY 协议含 resize 边带命令）均将 PTY 主端桥接到 WebSocket。Go 侧用 creack/pty：`pty.Start()` 起子进程；示例官方给 `pty.InheritSize()` + 监听 SIGWINCH 同步尺寸——web 化时改为把 xterm.js `onResize` 经 WS 发到后端调 `pty.Setsize()`。256 色/真彩由 xterm.js 配主题渲染，子进程 TERM 设 xterm-256color 即可。

**坑清单**：
1. 移动键盘：Android/GBoard 组合输入致乱序/重复字符（xterm.js#3600）；触摸滚动/选中差（#5377、#1101）。
2. 软键盘无 Esc/Ctrl/方向键，须自绘按键条（agentboard 专为 iOS Safari 做了适配）。
3. 软键盘遮挡视口，需 VisualViewport 处理。
4. resize：旋转屏抖动、断线重连后须重发尺寸（gotty 协议已含此边带）。
5. 回放：xterm.js scrollback 有限，长会话需 tmux capture 或 JSONL 外置落盘。
6. 安全：暴露 PTY≈给 shell，需鉴权（ttyd SSL/basic auth）+ 容器隔离（ragent 用 Docker）。

## 结论

1. 主流项目清一色 (b) headless 事件流+自绘 UI，无一内嵌终端。（官方文档+源码）
2. pi 的 JSONL 无头流契合 (b)，rick 可直接做事件网关+React UI。（源码）
3. PTY+xterm.js 可行但移动端硬伤，只宜作兜底通道。（社区）

## 信源链接

- OpenHands 架构: https://docs.openhands.dev/openhands/usage/architecture/runtime ; https://docs.openhands.dev/sdk/arch/events.md
- goose 架构/server: https://block-goose.mintlify.app/concepts/architecture ; https://block-goose.mintlify.app/advanced/server-deployment
- goose Desktop: https://deepwiki.com/block/goose/3.1-desktop-application
- aider browser UI: https://aider.chat/docs/usage/browser.html ; https://github.com/Aider-AI/aider/issues/481
- sugyan/claude-code-webui: https://github.com/sugyan/claude-code-webui ; https://deepwiki.com/sugyan/claude-code-webui/3.4-communication-protocol
- winfunc/opcode web 设计: https://github.com/winfunc/opcode/blob/main/web_server.design.md
- omnara: https://github.com/omnara-ai/omnara
- Claude Code on the web: https://www.anthropic.com/news/claude-code-on-the-web ; https://code.claude.com/docs/en/claude-code-on-the-web
- Agent SDK hosting/streaming: https://code.claude.com/docs/en/agent-sdk/hosting ; https://code.claude.com/docs/en/agent-sdk/streaming-vs-single-mode
- ttyd: https://github.com/tsl0922/ttyd ; wetty: https://github.com/butlerx/wetty ; gotty: https://github.com/yudai/gotty
- creack/pty: https://github.com/creack/pty
- xterm.js 移动端 issues: https://github.com/xtermjs/xterm.js/issues/3600 ; /issues/5377 ; /issues/1101 ; /issues/2403
- 终端路线项目: https://github.com/Chris-bzst/ragent ; https://github.com/gbasin/agentboard ; https://github.com/AliceLJY/agentdeck ; https://github.com/receptron/mulmoterminal ; https://github.com/boydfd/web_terminal_acp
