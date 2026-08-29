# pi 生态 Web UI/远程访问调研（L1-r2-leaf-2）

## 官方集成面（本地 docs：extensions.md / rpc.md / packages.md / sdk.md）
三轨：①Extension API（TS 扩展热加载，事件拦截+自定义工具/命令；ctx.ui 对话框在 RPC 模式经 `extension_ui_request` 子协议外发，可被外部 UI 接管）；②RPC 模式（`pi --mode rpc`：JSONL over stdio，prompt/steer/bash/get_tree 等命令+流式事件，官方定位即"嵌入 IDE/自定义 UI"）；③SDK（`createAgentSession()` 进程内 AgentSession + subscribe() 事件流，官方用例即"构建 Web UI"，OpenClaw 为点名参考实现）。分发：`pi install npm:` + pi.dev/packages 画廊。上游路线图信号：#2737 会话 RPC 服务器化（TUI 变订阅者，"server mode"开发中）、#8481 RemoteSession（本地 TUI+远端会话）、#5142 非TUI远程客户端扩展 API、Discussion #4444 ACP 协议支持。旧 monorepo badlogic/pi-mono 已迁 earendil-works/pi（org 另有 absurd、gondolin microvm agent 沙箱）。

## 第三方工具

### 1. PI WEB（[jmfederico/pi-web](https://github.com/jmfederico/pi-web)）⭐620
架构：长驻 session 守护进程（持会话）+Web/API 服务双进程，自动选 systemd/launchd 用户服务，浏览器断线会话存活；亦发布为 pi 包（/pi-web 命令）。功能：真实 repo/git worktree 工作区、多会话监督/重定向/评审、跨设备。成熟度：⭐620/fork121，npm @jmfederico/pi-web 周下载~2.7k，pi-web.dev。借鉴：会话所有权与 UI 进程分离。

### 2. pi-agent-dashboard（[BlackBeltTechnology/pi-agent-dashboard](https://github.com/BlackBeltTechnology/pi-agent-dashboard)）⭐253
架构：三件套=全局桥接扩展（每会话注入、识别 TUI/Zed/tmux 来源并转发事件）+HTTP/WS 网关（:9999）+浏览器端，与 pi TUI 共存。功能：多会话视图、聊天镜像、集成终端、diff 查看器、pi-flows、插件系统、中文 UI、mDNS/zrok 移动远程。成熟度：⭐253，npm @blackbelt-technology/pi-agent-dashboard。借鉴：扩展桥接+WS 网关模式、免公网远程（mDNS/zrok）。

### 3. pi-web-ui（[xing-shuyin/pi-web-ui](https://github.com/xing-shuyin/pi-web-ui)）⭐41，npm 月下载~2万
架构：pi SDK 进程内运行 agent，WebSocket 流式推送到浏览器；一键启动，Docker/systemd/launchd 部署。功能：thinking 块+工具调用渲染、内置终端、模型/系统提示词管理、技能与扩展开关、设置预设、CSS 主题、插件安装器（`pi-web-ui install owner/repo`）。成熟度：⭐41，2026-08 创建但迭代极快（v0.47）。借鉴：SDK 直嵌最省事、主题/插件化。

### 4. pi-remote-web-ui（[VVander/pi-remote-web-ui](https://github.com/VVander/pi-remote-web-ui)）⭐33
架构：服务仅绑 127.0.0.1:8080，SSH 隧道访问；单个进程内 AgentSession（SDK、无子进程）跨标签页共享。功能：极简安全 GUI，多会话 tab 开发中（PR#17）。成熟度：⭐33，由 pi-gui 改名。借鉴：loopback+SSH 隧道安全模型。

### 5. @firstpick/pi-package-webui（[Firstp1ck/pi-coding-agent-forge](https://github.com/Firstp1ck/pi-coding-agent-forge)）
架构：pi install 扩展包+独立 pi-webui CLI（--cwd/--pi/--host）控制 spawned pi 会话，默认绑 127.0.0.1。功能：多 tab 聊天/流式/模型控制/上传/slash 命令；姊妹包 @firstpick/pi-package-remote-webui 提供 /remote 命令：LAN+PIN+二维码手机直连（周下载 363）。借鉴：扩展包形态分发、PIN+QR 移动接入。

### 6. OpenClaw（[openclaw/openclaw](https://github.com/openclaw/openclaw)）⭐~38.7万（SDK 嵌入参考实现）
架构：pi SDK 进程内 `createAgentSession()` 嵌入自研 Gateway，无子进程/RPC。功能：WhatsApp/Telegram/Discord/iMessage 常驻个人助理。借鉴：SDK 深度嵌入+多渠道网关的成熟范本。

### 其他小型
hyperdreamer/pi-webui（⭐4，npm @hyperdreamer/pi-webui 638/月，pi 包形态，会话存活于真实工作区，配置含绑定地址/可信主机/session-daemon，驱动细节未明）；isr4el-silv4/pi-web-ui（Chrome 侧栏 UI+本地桥接扩展，已上架 Chrome 商店）；xuzhixiangya/pi-web-ui（浏览器远程控制）。npm 生态另有大量非 GUI 扩展包（pi-agent-suite、pi-agent-extensions、@narumitw/* 等），见 pi.dev/packages。

## 【未找到】
earendil-works 共 5 个公开仓，仅确认 pi/absurd/gondolin，余 2 仓未查到；官方第一方 web UI 未见（issue #4225 带"web-ui:"前缀但归属未证实）；isr4el-silv4、xuzhixiangya 两项目 star 未获取。

## 【结论】
1. pi 无第一方 WebUI；集成面=Extension/RPC/SDK 三轨，远程化已在上游路线图。
2. 最成熟第三方：jmfederico/pi-web（⭐620）与 BlackBelt dashboard（⭐253）。
3. 自研最优：SDK 进程内 AgentSession+WebSocket，配 systemd 守护+隧道/PIN 鉴权。
