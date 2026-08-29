# research-L2 简报：pi 生态 web ui 逐项目深挖（可复用性判定）

元信息：L2 | 主题：pi-web（jmfederico）/ ygncode-pi-web（新发现）/ pi-agent-dashboard / pi-web-ui / pi-remote-web-ui / @firstpick 系 / 上游 pi-server 系——license·技术栈·组件结构·对接方式·可复用资产·活跃度 | 时间基准 2026-08-29 | 方法：4 叶子 fanout（researcher 联网源码级取证）+ research 独立 web_search 交叉验证（ygncode 全部由 research 自查）| 上轮 L1-r2 盘点为基础

## 1. pi-web（jmfederico）——生态最成熟，Node+SDK 进程内
**事实（置信 0.85｜npm 依赖实测+dist 文件树+PR/releases+官网文档；leaf-1）**：
- **MIT**（仓库/npm/README 三处一致）｜⭐538-620/fork108-121｜2026-05-07 建｜活跃：npm 1.202608.2（2026-08-24），CalVer 3-5 天一发，25+ releases，周下载 ~2.7k，1005 commits 单人主导（巴士因子）
- 技术栈：后端 Node≥22.19+TS+**Fastify 5**（websocket/static/compress）+typebox+node-pty+undici；前端 **Lit 3 Web Components**（非 React/Vue/Svelte）+Vite 8+CodeMirror 6 全家桶+@xterm/xterm 6；**marked 18**+**jsdiff 9**；无数据库（JSON+内存+pi JSONL）
- 对接：**pi SDK 进程内嵌**（peerDep ≥0.84，piSessionService.ts 封装 SessionManager），PR#98 明确拒绝 spawn CLI 子进程路线——与 rick 的 rpc 子进程裁决相反，后端代码不可复用
- API 面：REST 按域拆分（sessions/machines/projects/workspaces/git/terminals/activity/diagnostics/storage/realtime/sessiond/config）；**实时流=WebSocket 非 SSE**（webSocketBridge 桥 sessiond Unix socket ~/.pi-web/sessiond.sock）
- daemon：**双进程 sessiond（长驻持全部会话运行时）+ web-server（可随意重启）**；pi-web install 自动选 systemd user/LaunchAgent+PATH 预检+doctor CLI；默认 127.0.0.1:8504；联邦=本地实例 undici 代理远程机器 HTTP+WS（machines）
- 前端结构：components/（PiWebApp/ChatView/SessionList/PromptEditor/appShell/）+controllers+api clients/parsers+chatTranscriptStore+plugins（浏览器插件 API v2）+798 前端测试；组件=标准 custom element 但强绑定自家 api/store 边界（禁裸 fetch/WS）
- 渲染：marked（流式+表格横滚）+CodeMirror 6（CodeViewer/编辑）+jsdiff（edit 卡片 diff+词级高亮+hunk 暂存）+xterm 全彩；无专用 ANSI→HTML 库

## 2. ygncode/pi-web（L2 新发现）——Go+SSE+rpc worker，与 rick 架构同卵 ★重点
> 与 jmfederico/pi-web 同名不同项目；L1 盘点遗漏，L2 由 research 独立发现并源码级核实
**事实（置信 0.9｜backend.md/frontend.md/system-overview.md/data-flow.md/manager.go/client.go/commits 多源交叉，research 自查）**：
- **MIT**｜⭐89/fork14｜2026-05-06 建｜活跃：最新 commit 2026-08-23（manual context compaction）、beta.36（2026-08-13）、PR 至 #108、双作者（setkyar/beilo）
- 架构=rick 已裁决方案逐条命中：**Go 1.25+ HTTP 服务 + 每 session 一 `pi --mode rpc` 子进程 worker**（internal/workers/manager.go：workers map[string]ChatWorker+Factory；ChatWorker 接口=Prompt/SetModel/SetThinkingLevel/Abort/GetState/GetCommands/Close）+ internal/rpc/client.go（JSONL 命令构建）+ **SSE**（GET /events?id=、sse registry、sse_format.go、stream.go 累积器）+ REST（/api/sessions、/api/session、/api/new、/api/projects?filtered=1 enabled-projects allowlist）
- 前端：**Svelte 5 SPA**（routes/components/index/session/settings/shared）+ Vite→web/dist→**//go:embed**（+.vite/manifest.json，internal/frontend/assets.go）+**PWA**；SessionsPage.svelte（会话列表/卡片/命令面板/新建 modal/项目管理 modal）、ArtifactPanel.svelte（纯函数 artifacts 检测+glob 过滤）；正从 imperative JS 渐进迁移 Svelte 5
- 功能：扫 ~/.pi/agent/sessions/ 离线浏览+继续聊天（文本/图片附件）+对任意 project path 开新会话+多 session 并行+worker 状态（idle/running/error）**崩溃自动恢复**+逐会话模型/thinking 切换+session 分享+静态导出+git 集成（分支/开 PR）
- 认证：**PI_WEB_TOKEN**（LAN 暴露默认强制）；渲染：marked+highlight.js（lazy）；分发：npm 包=installer（postinstall 从 GitHub Releases 下载多平台 Go 二进制→~/.pi/agent/bin/pi-web+自启动+注册 pi /web /remote /refresh 命令）
- 缺口：无 .rick 多工作区概念（仅 projects allowlist）、无 rick cmd 类型语义、无 jobs/知识库视图、beta 期成熟度

## 3. pi-agent-dashboard（BlackBeltTechnology）——extension 桥接+WS 网关，React 组件供给最富
**事实（置信 0.85｜bridge.ts/source-detector.ts/server.ts/pnpm-lock/chat-embed 文档源码级；leaf-2）**：
- **MIT**（© 2026 Robert Csakany，全部子包 MIT；注意 npm 另有 @fsabado 停更 fork 勿装错）｜⭐253｜2026-03-23 建｜活跃：v0.8.0（2026-08-26），2-6 周一版，双核心贡献者
- 技术栈：monorepo（shared/extension/server/client/electron/dashboard-plugin-runtime+~30 插件包）；后端 Node≥22.18+TS+Fastify 5+ws 8 **双 WS 网关**（Pi Gateway :9999 extension 侧 / Browser Gateway :8000）+node-pty+jsonwebtoken+mDNS+zrok+Electron 壳；前端 **React 19+Tailwind 4+Vite 6**+wouter+@tanstack/react-virtual
- 桥接：服务端把 extension 写入 pi 全局 settings.json 自动注入每个 pi 会话；bridge.ts WS 连 :9999 指数退避重连+事件缓冲+15s 心跳/60s 看门狗；来源识别=环境变量（ZED_TERM/TMUX）+hasUI+.meta.json sidecar+spawn 注入 PI_DASHBOARD_SPAWNED；UI 对话框走 PromptBus 双向
- 协议（最值得抄）：**事件带序号+subscribe{sessionId,lastSeq} 断线续传**；内存 LRU 100 会话×5000 事件、发送缓冲 >4MB 丢帧、大 payload 截断
- 前端：App.tsx 1437 行正拆 DesktopLayout/MobileLayout+SessionDetailView；80+ 组件；**chat-embed 子路径官方支持外部 React 嵌入**（ChatView/useSessionState/ThemeProvider 等），但 npm 只发 dist/，摘组件须 git vendor+约 24 依赖
- 渲染：react-markdown 10+remark-gfm 4+remark-math 6+rehype-katex 7+rehype-raw+**dompurify**；react-syntax-highlighter 16；**ansi-to-react 6**；**@git-diff-view** 系+diff 8；mermaid 11+Monaco 0.52+xterm 6

## 4. pi-web-ui（xing-shuyin）——SDK 直嵌+React 18 组件族
**事实（置信 0.85｜AGENTS.md+npm+仓库；leaf-3）**：
- **MIT**｜⭐41｜2026-08-04 建｜**极活跃**：25 天 0.1.1→0.47.0 共 47 版，~20K 下载/月（4998/周），单作者
- 技术栈：Node≥22.19+Express 4+ws 8+node-pty；前端 **React 18+Vite 6**+react-markdown 9+remark-gfm 4+rehype-highlight 7+highlight.js 11+@xterm/xterm 6；pi SDK ^0.84.2 进程内
- 架构：每对话独立 AgentSessionRuntime+全对话共享一个 ModelRuntime；事件 60ms 节流快照推 WS；主题=整份独立 CSS 热加载（~/.pi-web/themes/）；插件=GitHub+manifest.json 装入 dataDir/plugins
- 组件（web/src/components/）：ChatInput（斜杠命令/附件 chips/steer）、Message/MessageList（30 条折叠）、ToolCallBlock、ThinkingBlock、BashBlock、TerminalPanel、RightPanel（文件树+fs.watch）、LeftPanel、ModelConfigModal 等+use-chat.ts（WS reducer 状态机）——**MIT React 组件移植成本低**（换 SSE 客户端即可）

## 5. pi-remote-web-ui（VVander）——loopback+SSH 极简
**事实（置信 0.75｜README/issues/PR/vite.config；package.json 未直抓；leaf-3）**：
- **无 LICENSE 文件**（GitHub 侧栏无 License 字段）→ **代码不可抄**，仅设计借鉴｜⭐33｜2026-02-23 建｜低频（最后可证活动 2026-05-12，12 open issues）
- vanilla TS+WS 单文件服务端；只绑 127.0.0.1:8080+SSH 隧道（认证=SSH 密钥）；单 AgentSession 跨 tab=subscribe→遍历 wss.clients 广播，新连接先收 state_sync 全量历史；动机 issue#3：弃 rpc 子进程省内存（3 tab 3 进程 165MB→1 进程）——对 rick 无效（Go 无 Node SDK，必须子进程）

## 6. @firstpick 系 + 上游 pi-server 系
**@firstpick（置信 0.8｜npm+TECHNICAL.md+commits；leaf-4）**：pi-package-webui **MIT** v0.10.0（2026-08-24，~1.5k/周）+remote-webui MIT v0.1.9；Node 无框架 HTTP+**每 tab 一 pi JSONL RPC 子进程+detached supervisor（重启 HTTP 不杀 pi）**+原生 JS PWA；/remote=4 位随机 PIN+QR 内嵌 #pin= 免输+POST /api/network/open 重绑 0.0.0.0——受信 LAN 便捷门禁非强认证
**上游 @earendil-works/pi-server/client/protocol（置信 0.85｜README/CHANGELOG/PR#7344#7409；leaf-4+research 交叉）**：三包 MIT、monorepo 锁步 v0.84.2/3（2026-08-14）；#7344 已合并（CBOR+长度前缀分帧+125 协议测试）、#7409 已合并（SessionLease exclusive/shared）；**全部标 Experimental，wire protocol/transport/lease 均不稳定，6 周 20+ 版，0.84.0 已含 breaking**；pi-server 无 CLI 无内置 coding-agent service（须自实现 PiServerService）、默认 Unix socket、SQLite 后端；**生态主流（firstpick/pi-desktop）均走 JSONL RPC 稳定层**——rick 自建 rpc supervisor 当下正确，抽象传输层留退路

## 7. rick web 前端复用策略矩阵（四档判定）
| 项目 | 判定 | 理由 |
|---|---|---|
| **ygncode/pi-web** | **摘代码移植（首选）+架构样板** | MIT；Go+SSE+rpc worker+embed+token+PWA 与 rick 同卵；internal/rpc/client.go、workers/manager.go、sse registry、assets.go manifest 方案可直接改造抄入 rick runtime 层；不宜整体 fork（rick 有自有四层架构+jobs 语义），摘代码+抄结构 |
| **pi-agent-dashboard** | **摘组件移植（React 侧）+借协议** | MIT；chat-embed React 组件族+渲染栈可 vendor；事件序号+subscribe(lastSeq) 断线续传协议直接映射 rick SSE Last-Event-ID 设计；后端 Fastify/WS 异构不复用 |
| **pi-web-ui（xing-shuyin）** | **摘组件移植（React 侧）** | MIT；React 18 组件族（ChatInput/ToolCallBlock/ThinkingBlock/BashBlock/MessageList）+单 CSS 主题体系移植成本低（换 SSE 客户端）；迭代过快，摘静态组件不依赖其包 |
| **pi-web（jmfederico）** | **借设计自建** | MIT 但后端 Node+SDK 进程内+WS+Lit 3（小众框架）+组件耦合自家 api/store——直接复用成本>收益；抄其 sessiond 双进程模型、安装器（doctor/CLI）、API 域拆分、798 测试工程化 |
| **pi-remote-web-ui** | **借设计** | 无 LICENSE 代码勿抄；state_sync 全量+增量广播模式与 rick SSE 同构可参考 |
| **@firstpick 系** | **借设计** | MIT；PIN+QR+LAN 开放模式留给 rick 未来手机接入；detached supervisor 思路有价值；pi 包分发形态 rick 不需要 |
| **上游 pi-server 系** | **不用（现在）** | Experimental+breaking 频繁+须自实现 service；JSONL RPC 是稳定层；rick 驱动层做接口抽象，盯 PROTOCOL_VERSION 与 #8481 类提案 |

## 8. 渲染库推荐（各项目用库+rick 组合）
- 生态用库盘点：markdown=react-markdown+remark-gfm（dashboard/pi-web-ui）或 marked（jmfederico/ygncode）；高亮=highlight.js（rehype-highlight 或 hljs）或 react-syntax-highlighter 16（dashboard）或 CodeMirror 6（jmfederico，兼编辑器）；ANSI=ansi-to-react 6（dashboard）或 xterm.js 全彩（各家终端面板）；diff=@git-diff-view 系（dashboard）或 jsdiff 9（jmfederico）；XSS 清洗=dompurify（dashboard）；公式=remark-math+rehype-katex（dashboard）
- **rick 推荐（React 路线）**：react-markdown+remark-gfm+rehype-highlight(highlight.js)+**ansi-to-react**（bash 工具输出内联卡）+**@git-diff-view/react**（edit 工具卡 diff）+dompurify；终端兜底 @xterm/xterm 6+addon-fit；暂不引 CodeMirror/Monaco（无编辑需求，省 ~950KB vendor chunk）
- **rick 推荐（Svelte 路线，若摘 ygncode 前端）**：marked+highlight.js（lazy）+ansi 处理用 ansi-to-html 自封装+diff 自选 diff2html/@git-diff-view 风格自绘——ygncode 无 diff 库，需补
- 框架建议：**倾向 React**——组件供给面最广（dashboard chat-embed+pi-web-ui 组件族可直接摘，LibreChat/OpenHands 等先例全 React），且 rick 摘 ygncode 的主力在 Go 侧（与前端框架无关）；若走「fork ygncode 改造」路线则顺从 Svelte 5 亦可（组件面小、dist 更小）。两案均成立，React 供给面更宽

## 9. 生态状态一句话 + rick top 可复用资产
**一句话**：pi web ui 生态=官方 server 尚在实验期（CBOR+lease 不稳定、无 CLI），第三方已百花齐放且全部 MIT——Node 系三强（jmfederico 的 sessiond 分离/dashboard 的桥接网关/pi-web-ui 的 SDK 直嵌）+ Go 系一匹与 rick 完全同构的黑马（ygncode），rick 需要的每一层都有源码级先例可抄。
**top 资产（按优先级）**：
1. ygncode Go 侧全套：internal/rpc/client.go（JSONL 命令构建）、internal/workers/manager.go（每 session worker+Factory+崩溃恢复）、SSE registry/sse_format.go、internal/frontend/assets.go（Vite manifest+go:embed）——摘代码移植首选
2. dashboard 协议设计：事件序号+subscribe(lastSeq) 断线续传（直接映射 rick SSE Last-Event-ID+重放缓冲）+渲染栈组合（react-markdown 系+ansi-to-react+@git-diff-view+dompurify）
3. pi-web-ui React 组件族：ChatInput/ToolCallBlock/ThinkingBlock/BashBlock/MessageList+单 CSS 主题体系
4. jmfederico 工程化：sessiond 双进程模型、安装器（systemd/launchd 自适配+doctor+CLI）、REST 域拆分、前端测试体系
5. firstpick /remote：PIN+QR+LAN 开放（rick 未来手机接入直接照抄模式）
6. 主题=整份 CSS 热加载（pi-web-ui/VVander 先例，rick 主题机制可循）

## R7 上报项
1. VVander/pi-remote-web-ui 无 LICENSE 文件——代码不可抄仅设计借鉴；若需代码须联系作者授权
2. ygncode/pi-web npm 周下载量未获权威数值；star 89/59 两快照有时滞
3. 本环境对 github/npm 直连 DNS 阻断，全部证据经 web_search 快照+jsdelivr CDN（叶子同限）——commit 哈希/版本/stars 为快照值，落地抄码前应再核对当时 HEAD
4. dashboard chat-embed npm 只发 dist/，src 摘取须 git vendor+锁版本（约 24 依赖），集成成本中等
5. 上游 pi-server 稳定时间表未公开；PROTOCOL_VERSION 协商细节未逐条核读——rick 留接口抽象即可，不阻塞
6. ygncode 前端无 diff 渲染库——rick edit 工具卡 diff 需自选库（React 路线推荐 @git-diff-view）
7. 同名项目混淆风险：ygncode/pi-web（Go+Svelte+SSE）与 jmfederico/pi-web（Node+Lit+WS）同名但为独立项目（建仓日仅差一天）——两者是否有历史关联未逐 commit 核，不影响各自结论，但 rick 内部引用时应带全名区分

## 叶子文件清单
- research-L2-leaf-1.md（jmfederico/pi-web 源码级：license/栈/结构/API/daemon/渲染/活跃度）
- research-L2-leaf-2.md（pi-agent-dashboard 源码级：桥接/协议/组件/渲染栈/活跃度）
- research-L2-leaf-3.md（pi-web-ui+pi-remote-web-ui：架构/组件/主题插件/安全模型）
- research-L2-leaf-4.md（firstpick 系+上游 pi-server/client/protocol：分发/协议稳定性/切换成本）
- （research-L2-leaf-N.out.md 为运行时输出副本；leaf-1.out 已修剪为终稿）
