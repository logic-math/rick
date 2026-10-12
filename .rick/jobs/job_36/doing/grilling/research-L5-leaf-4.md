# Research: rick web dev 实例的叶子事实（L5-leaf-4）

信源等级：【高】官方文档/源码/规范；【中】issue/维护者回复/技术博客；【低】论坛。不确定处标「未验证」。

## 摘要
workbox 的 `non-precached-url` 在 **SW 脚本顶层求值期同步抛出**，使整份 SW 注册/更新失败（非仅某 route 失效）【高】；vite-plugin-pwa 官方文档要求 globPatterns 必须含 html。Go `net.Listen` 默认已设 SO_REUSEADDR，SO_REUSEPORT 需 `ListenConfig.Control`+`golang.org/x/sys/unix`；fd 传递用 `os/exec ExtraFiles` 即可，无需手写 fcntl。无 systemd 容器拿不到 LISTEN_FDS，socket activation 不可用；air 能保留旧进程、reflex 不能；SSE 自动重连+Last-Event-ID 是规范行为，seq 归零语义规范未定义。

## 1) workbox-precaching / createHandlerBoundToURL
1. **注册（脚本求值）时抛出**：源码 `const cacheKey = this.getCacheKeyForURL(url); if (!cacheKey) throw new WorkboxError('non-precached-url', {url})`【高】https://github.com/GoogleChrome/workbox/blob/v7/packages/workbox-precaching/src/PrecacheController.ts ；错误文案见 https://github.com/GoogleChrome/workbox/issues/2660
2. 生成式 sw.js 顶层即 `registerRoute(new NavigationRoute(createHandlerBoundToURL("index.html")))`（栈含 `sw.js:1:10085`）→ 顶层未捕获异常 = ServiceWorkers「script evaluation failed」：**首次注册失败则整份 SW 不安装、所有 route 全失效**；若是更新失败，旧 SW 继续生效【高（规范语义）】https://www.w3.org/TR/service-workers/ ；「规范算法逐字对应关系」未验证。
3. vite-plugin-pwa：官方文档「If you configure `globPatterns` … you **MUST** include all your assets patterns」，默认 `**/*.{js,css,html}`；globPatterns 去掉 html 就会生成上面那种会崩的 sw.js。issue #120 / #402 / #731（维护者答「问题在你的 workbox.globPatterns」），**无构建期校验/报错**【高（文档）+中（issue）】https://github.com/vite-pwa/vite-plugin-pwa/issues/120

## 2) Go 监听 socket
1. 默认**设** SO_REUSEADDR：`setDefaultListenerSockopts` → `SetsockoptInt(s, SOL_SOCKET, SO_REUSEADDR, 1)`，在 bind/listen 前调用【高（源码）】https://go.dev/src/net/sockopt_linux.go
2. 标准库**不设** SO_REUSEPORT（golang/go#23696 维护者原话「we don't currently use SO_REUSEPORT on linux」）。惯用法：`net.ListenConfig{Control: func(network, address string, c syscall.RawConn) error{...}}`，官方文档：Control「called after creating the network connection but before binding it」；其中 `unix.SetsockoptInt(fd, unix.SOL_SOCKET, unix.SO_REUSEPORT, 1)`。stdlib `syscall` 在 Linux **无** SO_REUSEPORT（#26771 以 Unplanned 关闭；zerrors 仅 SO_REUSEADDR=0x2），故实务依赖 `golang.org/x/sys/unix`（=0xf）【高】+「新版 stdlib 是否已补」未验证。参考 https://github.com/libp2p/go-reuseport/blob/master/control_unix.go
3. 分发语义：「distributed to multiple sockets … using a **4-tuple hash**」（内核提交 c617f398），连接在 SYN 期即绑定某 listener → 旧 listener 仍在组内时**新连接可能被旧 socket 收走**；关闭旧 listener 会 RST 掉握手中/accept 队列里的连接（LWN 853637）【高】https://lwn.net/Articles/853637/ → 平稳重启须先排空，或改用 fd 传递（单一 accept 队列）【中】https://goteleport.com/blog/golang-ssh-bastion-graceful-restarts/

## 3) self-exec + 传 listening fd
1. **cloudflare/tableflip**：spawn argv[0]，`syscall.ProcAttr{Files: fds}`+`syscall.StartProcess`，父进程等子进程 ready 后退出（「like nginx, but as a library」）【高】https://github.com/cloudflare/tableflip/blob/v1.2.3/doc.go
2. **facebookgo/grace**：gracenet 用环境变量记 fd 数、从 fd 3 起 `os.NewFile`+`net.FileListener` 继承，并刻意兼容 systemd socket activation 协议（LISTEN_FDS/LISTEN_PID）【高】https://github.com/facebookarchive/grace/blob/master/readme.md
3. FD_CLOEXEC 官方依据：syscall/exec_unix.go 注释「we mark all file descriptors close-on-exec and then, in the child, explicitly unmark the ones we want the exec'ed program to keep」+ os/exec 文档「ExtraFiles … entry i becomes file descriptor 3+i」【高】https://go.dev/src/syscall/exec_unix.go → 走 ExtraFiles/ProcAttr.Files **无需手写 fcntl**；只有裸 `syscall.Exec` 才需 `unix.FcntlInt(fd, F_SETFD, 0)`（此为**推断/未验证**）。坑：tableflip commit cae714b 记录 `exec.Cmd.Start()` 会对 ExtraFiles 调 `File.Fd()`，清掉 O_NONBLOCK 并禁用 runtime poller，故须传 dup 的 os.File【高】https://github.com/cloudflare/tableflip/commit/cae714b289e199db5da5f08af861ea65be6232c0

## 4) 无 systemd 容器
1. socket activation 是 service manager 行为：fd 经 execve 传入，契约 LISTEN_PID/LISTEN_FDS/LISTEN_FDNAMES、首 fd 为 3；sd-daemon.c 中 `LISTEN_PID != getpid()` 直接返回 0【高】https://www.freedesktop.org/software/systemd/man/latest/sd_listen_fds.html
2. 官方容器接口亦要求容器管理器本身是 systemd service 并转交 fd 给容器 init；Docker 默认 PID1 非 systemd ⇒ 无 LISTEN_*，socket activation 不可用（Podman 3.4+ 靠 conmon 转交才支持）【高】https://systemd.io/CONTAINER_INTERFACE/
3. 常用替代：SO_REUSEPORT / fd 传递 / 前置反向代理持端口（nginx、caddy）【中】https://github.com/containers/podman/discussions/27916

## 5) air vs reflex（能力对比 3 行）
- 构建失败保留旧进程：**air 支持** `stop_on_error = false` 保留上一成功二进制（注意默认值已被 PR #836 改为 true）；**reflex 不支持**——README 明确「重启时先给旧进程 SIGINT，1s 后仍活着则 SIGKILL，旧进程死后才启动新进程」【高】https://github.com/air-verse/air/pull/836 、https://github.com/cespare/reflex
- one-shot build：**两者均无官方 flag**；air workaround `bin="/usr/bin/make"` + `cmd="/usr/bin/true"`（issue #365 已关闭、PR #494 skip_run 未落地）；reflex 可把命令直接设成纯构建命令【中】https://github.com/air-verse/air/issues/365
- 自定义信号重启：**均不支持**；air 仅 `send_interrupt=true` 发 SIGINT（+`kill_delay`，issue #715 仍在请求）；reflex 源码 terminate() 硬编码「写 ^C 到 pty → SIGINT → SIGKILL」【高/中】https://github.com/cespare/reflex/blob/456b3718abbf1922cfbd498521c27851250f5496/reflex.go
- 安装（均需联网）：air `go install github.com/air-verse/air@latest`（Go module proxy）或 install.sh 拉 release；reflex `go install github.com/cespare/reflex@latest` 或发行版包，仅 Unix（Windows 编译失败 #90）【高】https://github.com/air-verse/air

## 6) SSE / EventSource 服务端重启
1. 自动重连 + Last-Event-ID 是**规范行为**：HTML 规范定义 reconnection time（初始值实现自定义，"probably a few seconds"）与 last event ID string，UA 重连时带 `Last-Event-ID` 头；HTTP 204 可令其停止重连【高】https://html.spec.whatwg.org/multipage/server-sent-events.html ；MDN：断线默认自动重连，`retry:` 字段设重连毫秒【高】https://developer.mozilla.org/en-US/docs/Web/API/Server-sent_events/Using_server-sent_events
2. **seq 归零无官方建议**：规范未定义「Last-Event-ID 已不可用」如何处置——whatwg/html#8297 正是在要求补这一点（现状无法告知客户端丢事件）【高（规范缺口）+中（issue）】https://github.com/whatwg/html/issues/8297 → 客户端/服务端需自理：id 全局单调且含 boot epoch（防重启后撞 id / 静默跳变）、服务端做 replay buffer（匹配不上就重置并显式发一次 resync 事件）、重连期间仅 `readyState=CONNECTING`+`error` 事件可用于 UI、服务端要能忽略陈旧 Last-Event-ID【中/低】。**未验证**：是否有浏览器/框架官方对此的额外建议。

## Sources
- 保留：workbox 源码+issue（#2660/#120/#402/#731）、go.dev 源码（sockopt_linux.go、exec_unix.go、exec_linux.go）、golang/go#23696/#26771、LWN 853637、内核提交 c617f398、tableflip doc.go/process.go/commit、grace readme/gracenet、systemd 官方（sd_listen_fds、CONTAINER_INTERFACE）、air/reflex 官方 README+源码+issue、WHATWG HTML SSE 章节+MDN——全部为一手。
- 丢弃：server-sent-events.com / goperf.dev / oneuptime / medium 等 SEO 或二手博客；reflex-dev/reflex（Python 框架，同名不同工具，污染检索）。

## Gaps
1. workbox-build / vite-plugin-pwa 是否在构建期校验 navigateFallback ∈ manifest：未在源码确认（多起 issue 暗示无校验）。
2. 最新版 stdlib `syscall` 是否已补 SO_REUSEPORT：未逐版核对。
3. 裸 `syscall.Exec` 需手工清 FD_CLOEXEC：官方文档无明文，仅 exec_unix.go 注释可推。
4. air PR #494/#836 的合并版本归属未逐一核对。
