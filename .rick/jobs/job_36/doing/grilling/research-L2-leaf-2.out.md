# Research: BlackBeltTechnology/pi-agent-dashboard 深挖

**npm**: `@blackbelt-technology/pi-agent-dashboard` v0.8.0（metapackage）｜**默认分支**: develop｜253★ / 38 fork

## 摘要

pi-agent-dashboard 是 pi 编码代理的实时 Web 仪表盘：全局 extension 注入每个 pi 会话转发事件，Node/Fastify 双 WebSocket 网关聚合，React 前端展示。MIT 许可、活跃维护；技术栈与 rick（Go+REST+SSE）不重合，可复用点集中在协议设计、渲染库选型与 chat-embed 前端组件。

## 详解

**① License/商用**：MIT（LICENSE 文件，Copyright (c) 2026 Robert Csakany）。允许商用、嵌入、修改、再分发、再许可，唯一义务是保留版权与许可声明。npm 全部子包均 MIT。注意 npm 另有 `@fsabado` 同名包（v0.5.5，2026-06 后停更，为贡献者个人 fork），依赖务必指向 @blackbelt-technology。

**② 技术栈**：workspace monorepo（历史 pnpm-workspace，commit 8ce152c 重构为 npm workspaces；packages/shared｜extension｜server｜client｜electron｜dashboard-plugin-runtime + 约 30 个插件包）。后端：Node≥22.18 + TypeScript 5.9 + **Fastify 5**（@fastify/compress/cookie/cors/static/websocket/http-proxy/reply-from）+ **ws 8** 双 WS 网关；jiti 直跑 TS 免构建；node-pty 终端、jsonwebtoken 鉴权、bonjour-service(mDNS)、zrok 隧道、Electron 桌面壳。前端：**React 19 + Tailwind 4 + Vite 6** + wouter 路由 + @tanstack/react-virtual；vitest 4 / Biome / Playwright。

**③ Extension 桥接**：注入方式=服务端启动时 `extension-register.ts` 把桥接 extension 自动写入 pi 全局设置 `~/.pi/agent/settings.json`（dev 模式跳过），此后每个 pi 进程加载它（`pi install npm:@blackbelt-technology/pi-agent-dashboard` 同理）。`bridge.ts` 以 WS 连 `ws://localhost:9999`（`PI_DASHBOARD_URL` 可覆盖），指数退避重连+事件缓冲；15s 心跳（CPU/RSS/heap）+60s 看门狗。事件转发：经 `pi.events`（EventBus facade）按声明通道订阅（FLOW/SUBAGENT_EVENT_MAP），封装为 `event_forward` 消息；agent_*/message_*/tool_execution_* 等逐类加富，`context`、`before_provider_request` 因体积排除。**来源识别**（`source-detector.ts` 的 `detectSessionSource(hasUI, sessionFile)` → tui|zed|tmux|dashboard）：按优先级探测环境变量 `ZED_TERM`→zed、`TMUX`→tmux，hasUI→tui，并读会话 `.jsonl` 旁 `.meta.json` sidecar；dashboard 会话靠 spawn 时注入 `PI_DASHBOARD_SPAWNED=1` 与一次性 `PI_DASHBOARD_SPAWN_TOKEN`。命令回传：浏览器 `send_prompt` → bridge 解析 `!!`/`!`/斜杠命令；UI 对话框走 PromptBus（prompt_request/response/dismiss/cancel）。

**④ WS 网关 API**：Pi Gateway **:9999**（extension 侧）、Browser Gateway **:8000**（浏览器 WS + REST + 静态资源）；`--port/--pi-port` 或 `PI_DASHBOARD_PORT`/`PI_DASHBOARD_PI_PORT`；默认绑 127.0.0.1。Extension→Server：session_register/unregister、event_forward、heartbeat(+ack)、commands_list、flows_list、models_list、providers_list、git_info_update、replay_complete、prompt_response、spawn_new_session、dispatch_extension_command。Browser→Server：`subscribe{sessionId,lastSeq}`（事件序号断线续传）、send_prompt、session_view/unview。Server→Browser：`event`（带序号）、session_updated、prompt_request/dismiss/cancel、openspec_update。REST（Fastify 路由组 session/git/file/grep/openspec）：`/api/session/:id/*`（prompt/abort/spawn/resume/rename/flow-control/model 等）、`/api/health`、`/api/config`、`/api/providers`、`/api/restart`。内存 LRU：100 会话 × 5000 事件/会话，发送缓冲 >4MB 丢帧，大 payload 截断。

**⑤ 前端结构**：`packages/client/src`——App.tsx（1437 行，正按 openspec「app-decomposition」拆分为 DesktopLayout/MobileLayout 与 SessionDetailView=SessionHeader+TokenStatsBar+内容路由+StatusBar+CommandInput）；侧栏 SessionList/SessionCard（目录分组+pin）；`lib/event-reducer` 纯函数事件归约器；hooks（WS、useSessionState、useViewDispatcher）；共 80+ 组件。内容视图互斥：ChatView（默认）、FileDiffView、FlowDashboard、OpenSpecPreview 等。插件系统：10 个 React slot（session-card-badge/content-view/settings-section/tool-renderer 等），package.json 声明 `pi-dashboard-plugin` manifest，Vite 构建期生成 plugin-registry.tsx。**可复用面**：`pi-dashboard-web/chat-embed` 子路径导出 ChatView/useSessionState/applySessionMessage/ThemeProvider/ApiContext 等，官方支持嵌入外部 React 应用；但 client 包只发布 dist/，src 子路径 npm 安装不可解析——须 git vendor 源码并自担约 24 个外部依赖（react/wouter/xterm/@git-diff-view/markdown 栈）。

**⑥ 活跃度**：高。仓库 2026-03-23 创建；release 从 v0.2.9 到 **v0.8.0（2026-08-26）**，约 2-6 周一版；核心作者 robertcsakany（874 贡献）、molnar-botond（239）；2026-07 下旬仍密集合 PR；npm metapackage 周下载 221。

**⑦ 渲染库**：Markdown=react-markdown 10 + remark-gfm 4 + remark-math 6 + rehype-katex 7 + katex + rehype-raw + dompurify（XSS 清洗）；代码高亮=react-syntax-highlighter 16；ANSI=ansi-to-react 6；diff=@git-diff-view/{core,file,lowlight,react} + diff 8；另 mermaid 11、Monaco 0.52、xterm 6。

## 结论

1. MIT 许可可商用/嵌入/修改，仅需保留版权声明；依赖须用 @blackbelt-technology 范围（@fsabado 为停更 fork）。
2. 后端 Node/Fastify/WS 与 rick 的 Go+REST+SSE 异构，服务端代码不可直接复用，仅可借鉴架构。
3. 最值得抄：事件序号 + subscribe(lastSeq) 断线续传协议，与双网关（extension 侧/浏览器侧）分离设计。
4. 来源识别纯靠环境变量（ZED_TERM/TMUX）+ hasUI + .meta.json sidecar，思路可移植到 rick 事件打标。
5. 前端 chat-embed 官方支持外部 React 嵌入，但 npm 装不到 src 子路径，需 git vendor + 约 24 个依赖。
6. 渲染栈选型可照抄：react-markdown+GFM+katex、react-syntax-highlighter、ansi-to-react、@git-diff-view、dompurify。
7. 深度绑定 pi 生态（@earendil-works/pi-coding-agent 的 EventBus/extension API），整体移植不现实。
8. 活跃度高：约每 2-6 周一版、最新 v0.8.0（2026-08-26）、253★，核心作者持续投入，维护风险低。

## 信源

- GitHub 仓库（README/结构）: https://github.com/BlackBeltTechnology/pi-agent-dashboard
- LICENSE（develop）: https://github.com/BlackBeltTechnology/pi-agent-dashboard/blob/develop/LICENSE
- docs/architecture.md（双网关/协议/事件流/REST）: https://github.com/BlackBeltTechnology/pi-agent-dashboard/blob/HEAD/docs/architecture.md
- extension 源码 bridge.ts: https://cdn.jsdelivr.net/npm/@blackbelt-technology/pi-agent-dashboard@0.7.0/packages/extension/src/bridge.ts
- extension 源码 source-detector.ts: https://cdn.jsdelivr.net/npm/@blackbelt-technology/pi-agent-dashboard@0.7.0/packages/extension/src/source-detector.ts
- 来源识别规格: https://github.com/BlackBeltTechnology/pi-agent-dashboard/blob/develop/openspec/specs/bridge-source-detection/spec.md
- server 源码 server.ts（Fastify 注册）: https://cdn.jsdelivr.net/npm/@blackbelt-technology/pi-agent-dashboard@0.7.0/packages/server/src/server.ts
- server 包依赖（npm package.json）: https://www.npmjs.com/package/@blackbelt-technology/pi-dashboard-server
- web 包依赖（渲染库清单）: https://www.npmjs.com/package/@blackbelt-technology/pi-dashboard-web
- pnpm-lock.yaml（版本锁定）: https://github.com/BlackBeltTechnology/pi-agent-dashboard/blob/develop/pnpm-lock.yaml
- chat-embed 嵌入文档: https://github.com/BlackBeltTechnology/pi-agent-dashboard/blob/develop/docs/embedding-chat-view.md
- chat-embed 集成示例: https://github.com/BlackBeltTechnology/pi-agent-dashboard/blob/develop/examples/chat-embed-tester/INTEGRATION.md
- 前端拆分规格: https://github.com/BlackBeltTechnology/pi-agent-dashboard/blob/develop/openspec/specs/app-decomposition/spec.md
- Releases（活跃度）: https://github.com/BlackBeltTechnology/pi-agent-dashboard/releases
- npm metapackage: https://www.npmjs.com/package/@blackbelt-technology/pi-agent-dashboard
- @fsabado fork（佐证）: https://www.npmjs.com/package/@fsabado/pi-agent-dashboard
