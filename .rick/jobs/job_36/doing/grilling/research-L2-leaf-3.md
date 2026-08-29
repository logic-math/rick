# Research: A) xing-shuyin/pi-web-ui 与 B) VVander/pi-remote-web-ui 深挖（为 rick 评估可复用资产）

> 环境限制：本运行环境对 github.com/npmjs.com/pi.dev 的直连 DNS 均被阻断（fetch_content ENOTFOUND），全部证据来自 web_search 提供商缓存快照（含 GitHub 页面、README/AGENTS.md 全文、npm 元数据、issue/PR 详情）。B 的 package.json 原文未能直抓。

## A) xing-shuyin/pi-web-ui（npm 实际包名：`pi-web-ui`，作者 npm 账号 xingshuyin）

**① License**：MIT（npm 元数据 "License MIT"；README 徽章链接 LICENSE 文件）。

**② 技术栈**（package.json + AGENTS.md）：
- 后端：Node ≥22.19 + Express 4（静态+/api/health）+ ws 8（/ws WebSocket）+ node-pty（服务端 PTY）+ compression + typebox。
- 前端：React 18 + Vite 6；react-markdown 9 + remark-gfm 4 + rehype-highlight 7 + highlight.js 11（github-dark.css）；@xterm/xterm 6 + @xterm/addon-fit；react-icons。
- 智能体：`@earendil-works/pi-coding-agent` ^0.84.2，SDK 进程内运行，读 `~/.pi/agent` 配置。
- 构建/测试：TypeScript 5.7、tsx、vitest、playwright-core、concurrently；dist/ 与 web/dist/ 经 npm `files` 白名单入包，prepublishOnly 自动构建。

**③ pi SDK 进程内架构**（server/agent-service.ts）：每客户端一个 ClientSession；`convs: Map<convId, Conversation>`，每对话独立 AgentSessionRuntime（new_chat 不杀旧对话、switch_conversation 只换 activeId，后台继续跑）；全部对话共享一个 ModelRuntime（首个对话创建时经 makeRuntimeFactory 播种复用，顶栏换模型全局生效）；消息序列化缓存按对话隔离；事件以快照（60ms 节流）推 WebSocket；edit_message→runtime.fork 分支重问；会话 JSONL 持久化。协议事实源 `server/protocol.ts`，`web/src/types.ts` 手工镜像，双端各加 switch 分支。
**主题**：每主题=完整独立样式表（内置深色 `web/src/styles.css` 整份副本换配色，不做 CSS 变量抽取）；内置主题随 npm `themes/` 分发；用户 CSS 丢 `~/.pi-web/themes/*.css` 免重启生效（文件名=id，须自包含、覆盖 `.hljs`，`--term-*` 变量控制 xterm ANSI 配色）；选择存 localStorage。
**插件**：`pi-web-ui install owner/repo|URL|子目录|#tag|本地目录`，浅克隆定位 manifest.json 拷到 `<dataDir>/plugins/<id>/`，以 tab 形式出现；设置面板可按客户端隐藏；uninstall 删目录但 config.json 升级保留。
**聊天 UI 组件可移植性**（`web/src/components/`）：FilePreview、LeftPanel（运行中对话列表）、RightPanel（文件树+fs.watch）、ChatInput（斜杠命令选择器/附件 chips/steer 补充）、Message/MessageList（30 条折叠、问题导航）、ToolCallBlock、ThinkingBlock、BashBlock、TerminalPanel/TermXterm、TopBar/FooterBar、Dialog、ModelConfigModal/PiSetupModal、Markdown/Dropdown/copy-button；配 App.tsx + use-chat.ts（WS 连接+reducer 状态机）+ 单文件 styles.css（CSS 变量分区）。React18 组件+单 CSS，移植到自建 React 前端成本低：仅需把 use-chat.ts 的 WS 客户端换成 rick 的 SSE 客户端。

## B) VVander/pi-remote-web-ui（原 pi-gui，2026-02-26 改名；仓库存在非 404）

**① License**：未检出 LICENSE 文件（GitHub 仓库侧栏元数据无 License 字段，搜索快照均未显示）——按 GitHub 默认"保留所有权利"处理，代码不可直接复制，仅可借鉴架构。

**② 技术栈**：
- 后端：Node + TypeScript，单文件 `server/index.ts`（WebSocket + HTTP 静态服务）；pi SDK 的 AgentSession 进程内（issue#3 代码 `import { AgentSession } from "@mariozechner/pi-coding-agent"`，README 链 badlogic/pi-mono SDK 文档）。
- 前端：无框架，vanilla TypeScript（`src/main.ts` + `src/style.css` 深色主题）。
- 构建链：Vite（vite.config.ts：root=.、outDir dist、dev 代理 `/ws`→`ws://127.0.0.1:8080`）；服务端 tsc（tsconfig.server.json→dist-server/）；dev 用 tsx watch（`npm run dev`）。
- 部署：systemd unit（pi-remote-web-ui.service，开机自启+自动重启）。
- package.json 原文未抓到（DNS 阻断），确定含 vite/tsx/typescript/ws/pi SDK。

**④ loopback 模型与跨标签页共享**：服务器只绑 `127.0.0.1:8080`，绝不暴露公网；访问靠 SSH 隧道（`ssh -L 8080:localhost:8080`），认证完全由 SSH 密钥承担，无密码/token/TLS；README 给 ~/.ssh/config LocalForward 自动化。单一 AgentSession 实例在服务器进程内（启动时 createAgentSession()，无子进程）。跨 tab 共享=WebSocket 广播：`session.subscribe(event)`→`JSON.stringify`→遍历 `wss.clients`（readyState===OPEN）逐个 send；新 tab 连接先收 `{type:"state_sync", messages: session.messages}` 全量历史。命令 prompt（流式中 `streamingBehavior:"followUp"` 排队）/abort/new_session 对所有 tab 生效。动机（pi-gui issue#3）：替代 `pi --mode rpc` 子进程方案——3 tab 由 3+进程/约 165MB 额外内存降为单进程零额外；typed API 替代 stdin/stdout JSON。

**⑤ Markdown/高亮/ANSI/diff**：
- A：react-markdown + remark-gfm + rehype-highlight + highlight.js（github-dark.css，浅色主题需覆盖 .hljs）；终端 ANSI 由 xterm.js 渲染（--term-* 变量）；未见专用 diff 库（bash/工具输出走 BashBlock 卡片）。
- B：无 markdown/高亮/ANSI/diff 库；issue#9（open，2026-02-26）请求"更丰富 markdown、代码块复制按钮、语法高亮主题（对齐 pi-web-ui）"，现状为基础渲染。

**⑥ 活跃度**：
- A：npm `pi-web-ui` 2026-08-04 首发 0.1.1 → 2026-08-28 发 0.47.0，约 25 天 47 个 minor 版本，20.1K 下载/月（5,350/周）；GitHub 公开仓库 main 分支持续更新。单作者高速迭代。
- B：仓库 2026-02-23 创建，33★/5 fork/12 open issues；贡献者 VVander(3)+wanders-bot(2)；最后可证 commit 2026-02-28（改名+多会话 tab bar，PR#17 open）；PR#18（移动端布局+图片附件+会话列表）2026-05-12；无 npm 发布，低频维护。

## 对 rick（Go+REST+SSE，前端自建）的启示
A 的 React 组件族+单 CSS 主题体系为 MIT 可直接移植；B 的 state_sync+广播模式与 rick 的 SSE 天然同构（连接时全量快照、后续增量广播，Go 侧实现极简）；B 的 loopback+SSH 免认证部署模式可参考；高亮栈 react-markdown+remark-gfm+rehype-highlight+highlight.js 已被 A 验证；diff 渲染两者皆缺，rick 需自选（react-diff-view / shiki diff / diff2html）。

## 结论（8 条）
1. A=MIT 可复用；B 未检出 LICENSE 文件，代码勿抄、只借鉴架构。
2. A：每对话独立 AgentSessionRuntime+全对话共享一个 ModelRuntime，事件 60ms 节流快照推送。
3. A 主题=整份独立 CSS 热加载（~/.pi-web/themes/）；插件=GitHub 目录+manifest.json 装入 dataDir/plugins。
4. A 组件在 web/src/components/，十余个 React18 组件+单文件 CSS，移植成本低。
5. B 只绑 127.0.0.1:8080，SSH 隧道+密钥即全部认证，systemd 部署。
6. B 跨 tab：session.subscribe→遍历 wss.clients 广播；新连接先发 state_sync 全量历史。
7. 渲染栈：A 用 react-markdown+remark-gfm+rehype-highlight+highlight.js+xterm(ANSI)；B 无渲染库。
8. 活跃度：A 极高（25 天 47 版、20K/月下载）；B 低（最后可证活动 2026-05-12，12 个 open issue）。

## 信源
- A 仓库：https://github.com/xing-shuyin/pi-web-ui
- A AGENTS.md（架构/目录/组件表）：https://github.com/xing-shuyin/pi-web-ui/blob/main/AGENTS.md
- A README：https://github.com/xing-shuyin/pi-web-ui/blob/main/README.md
- A npm（依赖表/版本史/MIT）：https://www.npmjs.com/package/pi-web-ui
- A pi.dev 包页：https://pi.dev/packages/pi-web-ui
- B 仓库：https://github.com/VVander/pi-remote-web-ui
- B README（安全模型/架构图/目录）：https://github.com/VVander/pi-remote-web-ui/blob/main/README.md
- B issue#3（AgentSession 进程内方案+广播代码）：https://github.com/VVander/pi-gui/issues/3
- B issue#9（markdown 渲染现状）：https://github.com/VVander/pi-remote-web-ui/issues/9
- B PR#18（最新活动）：https://github.com/VVander/pi-remote-web-ui/pull/18
- B vite.config.ts：https://github.com/VVander/pi-remote-web-ui/blob/main/vite.config.ts
