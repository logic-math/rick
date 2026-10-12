# Research: jmfederico/pi-web（Pi Coding Agent 的 Web UI）

## 摘要
jmfederico/pi-web（npm：@jmfederico/pi-web）是单人主导但非常活跃的 Pi Coding Agent 网页控制台：Node.js+Fastify 后端、Lit Web Components 前端、WebSocket 实时流；核心是「长驻 sessiond 拥有会话 + 可随时重启的 web/API」双进程架构，会话在浏览器断开后继续运行。与 rick（Go+REST+SSE）的问题域高度重合，MIT 许可允许直接复用代码与设计。

## ① License
GitHub 仓库 License 字段、npm License 字段、README 徽章三处一致：MIT（"MIT © 2026 Federico Jaramillo Martinez"）。商用嵌入零障碍，无 copyleft/额外条款；可直接抄代码、设计、文档。

## ② 技术栈（npm package.json 实测）
- 后端：Node.js≥22.19+TypeScript；Fastify ^5.10（+@fastify/websocket ^11.3、static ^10、compress ^9）；schema 校验 typebox；终端 node-pty ^1.1；ws ^8.21；联邦代理 undici ^8.5。无数据库：状态=JSON 文件+内存+pi 的 JSONL 会话存储。
- 前端：Lit ^3.3.3（LitElement Web Components，非 React/Vue/Svelte）；构建 Vite ^8；编辑/高亮 CodeMirror 6 全家桶（go/python/rust/json/html/css/js/markdown+legacy-modes）；终端 @xterm/xterm ^6+addon-fit；Markdown 用 marked ^18；diff 用 diff ^9（jsdiff）。
- Pi SDK 为 peerDependencies：@earendil-works/pi-agent-core、pi-ai、pi-coding-agent ≥0.84.0。

## ③ 前端结构（src/client/src/）
components/：PiWebApp.ts（根 LitElement）、ChatView.ts（聊天流）、SessionList.ts（会话列表）、MachineList.ts、PromptEditor.ts（输入区，支持粘贴/拖拽图片）、appShell/（AppNavigationPanel 等）、shared.ts（--pi-* CSS 变量主题）。controllers/（session/machine）；api.ts+api/clients+api/parsers；appState.ts、chatTranscriptStore.ts、chatHistoryCache.ts、chatMessages.ts；route.ts（前端路由）；formatting/markdown.ts；plugins/（registry、core/actions，浏览器插件 API v2）。设置页=Settings 深链 UI（General/Session daemon/Pi packages/Plugins/Keyboard 标签页）。构建按需分包：TerminalPanel、CodeViewer、vendor-editor(~950KB)、vendor-terminal。组件是标准 custom element，理论可移植，但强绑定自家 api client/store（AGENTS.md 规定浏览器侧禁裸 fetch/WebSocket，必须走 request()/resolveAppUrl()/resolveAppWebSocketUrl() 边界）。前端 798 个测试。

## ④ 与 pi 的对接
进程内嵌 SDK（import @earendil-works/pi-coding-agent），非 spawn CLI、非 extension 协议。src/server/sessions/piSessionService.ts（编译后 52KB，核心服务）封装 SessionManager.create/open/list/resume，沿用 pi 默认 JSONL 会话存储与 PI_CODING_AGENT_SESSION_DIR 目录优先级；会话运行时由 sessiond 进程持有（piSessionManagerGateway 网关）。PR#98 明确拒绝「另起 pi CLI 进程」方案（pure-Pi runtime 方向）。附件走 pi 原生 ImageContent+resizeImage；亦发布为 Pi 包（pi 内 /pi-web 命令，安装与启用分离）。

## ⑤ API 面
REST（/api/ 前缀）按域拆分（dist/server 目录取证）：sessions/（sessionRoutes、authRoutes、oauthLoginFlowService、sessionArchiveStore…）、machines/（machineRoutes、machineProxyRoutes、machinePluginProxyRoutes…）、projects/、workspaces/、git/、terminals/、activity/、diagnostics/、storage/、realtime/、sessiond/、configRoutes。已知路由如 POST /api/…/sessions/:sessionId/reload、GET /pi-web-plugins/manifest.json。实时流=WebSocket（非 SSE）：官方文档「Chat with Pi Coding Agent through realtime WebSocket events」；@fastify/websocket 承接浏览器，web 进程 webSocketBridge.js 桥接到 sessiond（默认 Unix socket ~/.pi-web/sessiond.sock，可切 TCP）。联邦：本地实例当网关，undici 服务端到服务端代理远程机器的 HTTP+WS（含终端/插件）。前后端共享类型 src/shared/apiTypes.ts。

## ⑥ daemon 模式
双进程：pi-web-sessiond（长驻、拥有全部活动会话运行时、Restart=no 不热重载）+ pi-web-server（web/API/UI，可随意重启，dev 下 Vite HMR）。pi-web install 自动选原生 per-user 服务：Linux systemd user service / macOS LaunchAgent，bash -lc 登录 shell 启动并做 PATH 预检；无 systemd 环境（旧 WSL/容器）手动跑两个进程。会话存活：运行时驻 sessiond 内存，浏览器断开、web/API 重启均不中断；历史落 pi JSONL。配置：~/.config/pi-web/config.json（PI_WEB_CONFIG 覆盖）+项目级 .pi-web/config.json；数据目录 ~/.pi-web（projects.json、machines.json、sessiond.sock、plugins/）。CLI：install/doctor/status/logs/restart/version/uninstall；bin：pi-web-server、pi-web-sessiond；默认 127.0.0.1:8504。

## ⑦ 活跃度
建仓 2026-05-07；538 stars/108 forks/38 open issues；jmfederico 1005 commits（绝对主导）。npm 最新 1.202608.2（2026-08-24 发布），CalVer 1.YYYYMM.N，3 个多月 25+ 个 release（约 3-5 天一发）；周下载约 2.7k（月约 7.9k）。结论：非常活跃、单人巴士因子风险。

## ⑧ 渲染细节
Markdown：marked ^18（formatting/markdown.ts；流式渲染；表格横向滚动容器；复制时带原始 markdown；mermaid 未实现，仅 issue #116 请求）。代码高亮：CodeMirror 6（CodeViewer 组件+vendor-editor 语言包 chunk，文件查看与编辑）。diff：jsdiff ^9——edit 工具卡片渲染 diff+行内词级高亮（v1.202605.9 起贴近 TUI）；文件 diff 视图支持 hunk 暂存。ANSI：xterm.js 6 终端面板全彩（node-pty 后端）；依赖中无专用 ANSI→HTML 库（无 ansi-to-html/ansi_up）。

## 结论
1. MIT 许可，商用嵌入零障碍，代码/设计可直接移植。
2. 核心可借鉴：长驻 sessiond 拥有会话+可重启 web/API 的双进程模型。
3. 实时流用 WebSocket 而非 SSE，webSocketBridge 桥接 sessiond Unix socket。
4. pi 对接=进程内 SDK（peer dep ≥0.84），复用 pi JSONL 会话存储。
5. 前端 Lit 3 Web Components，标准 custom element 但耦合其 api/store 边界。
6. ChatView/SessionList/PromptEditor/CodeMirror/xterm 资产可参考，惜非 React。
7. 安装器（systemd/launchd 自适配+doctor 预检+CLI）设计成熟，可整体借鉴。
8. 渲染栈 marked+CodeMirror6+jsdiff+xterm；无独立 ANSI 转换库。

## 信源
- GitHub 仓库：https://github.com/jmfederico/pi-web
- README：https://github.com/jmfederico/pi-web/blob/main/README.md
- AGENTS.md：https://github.com/jmfederico/pi-web/blob/main/AGENTS.md
- npm（依赖表/版本史）：https://www.npmjs.com/package/@jmfederico/pi-web
- 官网文档：https://pi-web.dev/ 、/install、/config、/plugins、/faq、/machines
- Releases：https://github.com/jmfederico/pi-web/releases
- 源码 commits：f65cb87、d17050e、c0d1222、0405b38、111db63、82ba2e0、4bc390a
- jsDelivr dist 文件树：https://cdn.jsdelivr.net/npm/@jmfederico/pi-web@1.202606.3/dist/
- PR#98：https://github.com/jmfederico/pi-web/pull/98
- Pi SDK 文档：https://github.com/earendil-works/pi/blob/v0.84.1/packages/coding-agent/docs/sdk.md
