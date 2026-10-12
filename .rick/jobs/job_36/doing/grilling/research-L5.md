# research-L5 简报 —— 开发闭环层（改动如何生效 + 自举问题）

元信息：阶段 L5 / 主题「dev-web 闭环：前端生效、后端重建重启、自举不断会、反馈回路、构建依赖、自动恢复」/ 时间基准 2026-09-20 22:15-22:50 (+08) / 代码基准 git HEAD=a60035a，生产进程 pid 172761（`./bin/rick web --listen 0.0.0.0 --port 8413`，21:59 启动）
证据等级：**[码]** 仓库代码（文件:行号）；**[测]** 本机实测（命令+输出，均在隔离 HOME=/tmp/l5devhome、端口 18413 上做，未触碰生产）；**[档]** 外部文档（叶子-4）。
confidence：除标注外均为高（≥0.8）。

## 0. 结论速览（事实 14 条；本节即 ≤3000 字执行层，其余各节为证据细节）

1. 前端热更今天就能用，且**不需要 dev 实例**：静态层逐请求 Lstat 覆盖层 `~/.rick/web/dist/index.html`（internal/web/static.go:71-99），watcher 每 2s 轮询该 dist 树并在**稳定后**推 `frontend_reload`（internal/web/watcher.go:27-105）。**[码]**
2. 但 `frontend_reload` 的实测延迟是 **6.8s**（不是注释声称的 0.5-2.5s）：500ms debounce 分支实际不可达，真正的触发条件是「距首次变更 ≥5s 的那一次轮询」（watcher.go:88-105）。**[测]** touch index.html → SSE 收到 `frontend_reload` +6.83s。
3. 启动时会有一次**无变更的伪 reload**（≈4.7-7s）：`lastSig` 初值为空串，首轮签名即被判为「变更」（watcher.go:78-99）。**[测]** 两次实测启动后 4.7s / <14s 各出现一次 reload。
4. 生产覆盖层当前 = 仓库 HEAD 的 dist（`~/.rick/web/dist` 由 `~/.rick/deploy-web.sh` 用 `npm run build` + cp 投放，**不重启服务**）；`~/.rick/web/` 里**没有** `.rick-managed` 标记也没有 `src/`，即线上并非走 `rick web customize` 链路。**[测]** ls ~/.rick/web + 读 deploy-web.sh。
5. PWA service worker 在本仓库**注册即失败**（也因此目前不会造成「改了看不到」）：`web/dist/sw.js` 的 precache 清单**不含 index.html**（globPatterns 仅 `assets/**`，vite.config.ts:55），而它同时调用 `createHandlerBoundToURL("index.html")`，workbox 该函数在 url 不在 precache 时**抛 non-precached-url**。**已用 Node/vm 重放真实 `web/dist/sw.js` + `workbox-9c191d2f.js` 复现**：`UNHANDLED REJECTION: non-precached-url :: [{"url":"index.html"}]`（harness 在 /tmp/swtest/harness.js）→ 浏览器里 SW 脚本求值失败、注册不成立。f88bc82 起所有历史构建同样如此。confidence 0.9（剩浏览器 DevTools 终校）。**[码]+[测]**
6. 无 SSE 客户端时，dev 实例 `kill -TERM` → 进程 8.7ms 退出、exit=0；**有 SSE 长连接时 → 9.91s 才退出且 exit=1**，日志 `Error: graceful shutdown: context deadline exceeded`（server.go:139-146 的 10s Shutdown 超时 → Close + 返回 error）。**[测]**
7. 端口交接不是瓶颈：`kill -9` 后立刻重启，**42ms** 就 health 200（Go listener 默认 SO_REUSEADDR，叶子-2 独立实测重绑 2ms、TIME_WAIT 不阻塞）；不依赖 systemd socket activation（本机虽有 systemd 249 但 `is-system-running=starting`，unit 托管不可靠；`systemd-socket-activate` 二进制本身可用）。**[测]**
8. dev 实例与生产**可零改码并存**：所有 web 状态路径都跟随 `$HOME`（internal/web/web.go:12-68 明说；handler/web.go:112-121，os.UserHomeDir 实测尊重 $HOME），singleton pid 也因此隔离。**[测]** 用 HOME=/tmp/l5devhome 起 18413 实例，健康检查 OK、workspaces=0（生产 7 个）、serve 自己的 dist。
9. 但 HOME 换掉会连带换掉 GOPATH/GOMODCACHE/GOCACHE/GOENV：实测覆盖 HOME 后 `go run` 触发 `go: downloading go1.25.0`（缓存失效、需网络）。=> 隔离方案必须同时固定 GOPATH/GOCACHE（或直接 `env -i` 白名单）。**[测]**
10. pi worker 的输入输出是**匿名 pipe**（生产 worker pid 176315 的 fd 0/1 → `pipe:[...]`，supervisor.go:218-230 StdinPipe/StdoutPipe），且 worker `Setpgid` 自成进程组（ppid=web pid、pgid=自身，实测）。=> 新 web 进程**无法接管**既有 worker（无路径可重连）；且优雅退出会主动 `kill(-pgid, SIGTERM)` 收尸（server.go:129-131 + sessions.go:1745-1756 + supervisor.go:492-536）。**[测]+[码]**
11. `spec.Dir = ws.Path`（sessions.go:437-444、984-993），实测生产 worker 的 cwd = 其工作区路径 `/workdir/sunquan20/MOFANG_SEARCH`。=> 「生产托管 + 工作目录=dev worktree」的架构**现有代码即可实现**，无需改动。**[测]**
12. 重启后自动恢复所需字段**缺失**：`SessionEntry.Busy` 是 `json:"-"` 刻意不落盘（registry.go:247-253），重启只能把所有 active/running 标成 error（sessions.go:154-176 ReconcileOnStart），恢复要人工点 Resume；doing/dream 后台型会话明确不可 resume（sessions.go:1017-1026 返 409）。**[码]**
13. **高危**：dev 实例**绝不能只换端口共用 HOME**——生产当前没有 `~/.rick/web.pid`（L6 亦实测），singleton 门禁失效；同 HOME 的第二实例启动时 `ReconcileOnStart` 会把 prod 的 active 会话改写为 error 并落盘。隔离必须走 `HOME=$DEVHOME`（同时固定 GOPATH/GOCACHE/GOENV，`RICK_PI_AGENT_DIR` 指向 prod 的 agent 目录）。**[测/高危]**
14. 两条已验证的落地细节：① 覆盖层可以是**目录软链**（实测 `$DEVHOME/.rick/web/dist → <devtree>/web/dist` 时 `GET /` 正确返回软链内 index.html，带 no-cache/Last-Modified）；② 生产覆盖层与仓库 dist **逐字节相同**（sha256 `d03b90a1…`），即线上前端就是 HEAD 构建，无手改痕迹。**[测]**

## 问题索引

- Q1 前端改动生效（overlay/watcher/缓存/SW）→ §1
- Q2 后端/CLI/runtime 改动生效（5 方案对比）→ §2
- Q3 自举问题（4 架构方案对比 + 推荐）→ §3
- Q4 反馈回路（会话如何知道 dev 已重启成功/失败）→ §4
- Q5 构建与依赖约束（go/npm/embed/无网）→ §5
- Q6 dev 实例重启后托管会话如何恢复（持久化字段）→ §6
- R7 上报项 → §7；叶子证据文件 → §8


## §1 Q1 前端改动生效（overlay + watcher + 缓存 + SW）

**事实**
- 覆盖层优先规则是 all-or-nothing：`~/.rick/web/dist/index.html` 存在 → 整棵 dist 树生效；否则回内嵌 baseline（static.go:71-99 注释明确「半个新半个旧更糟」）。覆盖层缺某文件时：asset 路径 404、非 asset 路径回 index.html（static.go:88-96）。**[码]**
- `StateDir` 来源 `web.WebStateDir()` = `$HOME/.rick/web`（cmd/web.go:117-124 → server.go:44-46,157-167）；`StartWatchers` 监听同一 `StateDir/dist`（server.go:157-167）。=> **dev 实例用不同 HOME 就自动拿到自己的 dist 覆盖层**（实测：HOME=/tmp/l5devhome 时 18413 服务的是 /tmp/l5devhome/.rick/web/dist）。**[测]**
- 缓存策略：`assets/*` 与根级 `workbox-*.js` → `max-age=31536000, immutable`；`index.html/manifest/sw.js/registerSW.js/favicon.ico` → `no-cache`（static.go:18-27,120-134）。hash 化产物改名即换 URL，不会脏缓存。**[码]**
- 前端在收到 `frontend_reload` 时 `window.location.reload()`，并用 sessionStorage 记 seq 去重防重放循环（web/src/api/sse.ts:11,53,336,422-427）。**[码]**
- 实测延迟：touch 覆盖层 index.html → `frontend_reload` 到达 **+6.83s**（watcher 2s 轮询 + 「距首次变更 ≥5s 才 flush」，watcher.go:88-105）；启动时另有一次**伪 reload**（≈4.7-7s，watcher.go:78-99 的 lastSig 初值问题）。**[测]**
- SW：`web/dist/sw.js` precache 清单 6 条（assets×2、图标×3、manifest），**不含 index.html**；同一行还调用 `createHandlerBoundToURL("index.html")`，该函数在 url 不在 precache 时抛 `non-precached-url`（bundle 内实证）→ SW 脚本求值期拒绝注册。**实测复现（Node/vm 重放真实 sw.js + workbox bundle，/tmp/swtest/harness.js）**：`UNHANDLED REJECTION: non-precached-url :: [{"url":"index.html"}]`，confidence 0.9（待浏览器 DevTools 终校）。`registerType:"autoUpdate"` + `skipWaiting/clientsClaim` 已生成（vite.config.ts:12-46）。**[码]+[测]**
  - 复现命令：`node /tmp/swtest/harness.js`（vm 上下文里 stub `self/caches/skipWaiting/location`，用 `importScripts` 按 URL 加载同目录 workbox 模块后 eval `web/dist/sw.js`）。附带修法：把 `index.html` 加进 precache（或去掉 `createHandlerBoundToURL`）；**修复后 SW 才真的会接管页面**，届时必须同步解决「SW 缓存导致改了看不到」的体验问题，否则前端热更会退化。
- 生产覆盖层与仓库 dist 逐字节一致：`sha256sum` 实测 `~/.rick/web/dist/index.html` == `web/dist/index.html` = `d03b90a1…`；entry JS 同为 `c8b71cd1…`。**[测]**

**推荐（dev 实例）**
1. 隔离层用 `HOME=$DEVHOME`（+ 固定 GOPATH/GOCACHE，见 §5），把 `$DEVHOME/.rick/web/dist` 做成**指向 dev worktree `web/dist` 的目录软链**：`ln -s <devtree>/web/dist $DEVHOME/.rick/web/dist`。这样 `npm run build` 产物零拷贝即可被 dev 实例服务（`serveOverlay` 只对**请求到的文件**做 Lstat 拒软链，目录级软链可通过）。代价：软链路径绕过 `resolveUnder` 的前缀语义（仍是规范内行为），需在实现时实测一次。**[推荐]**
2. 需要「立即生效」时不要等 watcher：直接把 dev 实例的 reload 当**可选**，人或 AI 手动 `curl -H "Authorization: Bearer <tok>"` 无关——最省事的是浏览器 F5（index.html no-cache 必然拿到新 asset 名）。若嫌 6.8s 慢，改 watcher 的 debounce 分支（真 debounce 500ms）是一处小改（低风险、可独立成 job）。**[推荐]**
3. 「改了看不到」的三个真实成因与对策：① 覆盖层残留旧 dist（部署脚本 `rm -rf` 后再 cp，现有 deploy-web.sh 已如此）；② 浏览器装过**早期构建**的 SW——当前构建的 SW 必然注册失败（§0.5），但**若未来修好 globPatterns，SW 就会真的接管页面**，届时必须同步解决缓存问题；对策：dev 构建禁用 PWA（`VitePWA({disable:true})`）或在 dev 的 index.html 注入 self-destroying SW，并在排障手册写「先 Unregister SW + 硬刷新」；③ 标签页 SSE 断开（dev 重启）→ 手动 F5。**[推荐]**
4. **已实测**：覆盖层可以是**目录软链**——`$DEVHOME/.rick/web/dist → <devtree>/web/dist` 时，`GET /` 正确返回软链目标内的 index.html（响应带 `Cache-Control: no-cache` + `Last-Modified`，证明走的是覆盖层而非内嵌 baseline）。=> 推荐 #1 的软链方案成立（`serveOverlay` 只对**请求到的文件**做 Lstat 拒软链，目录级软链不触发）。**[测]**

## §2 Q2 后端/CLI/runtime 改动生效：5 方案对比

前提事实：Go 二进制无热加载；dev 实例重启的**实测代价**是——无 SSE 客户端 8.7ms 退出/exit 0；有 SSE 客户端 9.91s 退出/**exit 1**（10s graceful 超时，server.go:139-146）；`kill -9` 后立刻重启 **42ms** 就绪（SO_REUSEADDR，无需 systemd）。优雅退出会**顺序**收掉每个 worker（`CloseAll` 顺序 for，supervisor.go:347-357；单个 worker 预算 TERM→5s→KILL，supervisor.go:47,492-536），所以停机时间随活跃 worker 数线性增长。**[测]+[码]**

| 方案 | 端口交接 | in-flight HTTP/SSE 中断代价 | 实现复杂度 | 对现有代码侵入 | 失败恢复 | 判定 |
|---|---|---|---|---|---|---|
| (a) 手动/API 触发 `go build` + 重启进程 | 关闭后立即重绑，实测 42ms（`kill -9` 后） | SSE 全断（客户端自动重连+resync，sse.ts 已处理）；若等 graceful 则 10s + exit 1 | 最低（一个 shell 脚本/Bash 工具调用） | 零 | 构建失败则旧进程仍在（build 与 restart 分两步最稳） | **推荐为 dev 默认**，配 §4 的脚本 |
| (b) 启动器常驻（wrapper 持端口或纯 supervisor） | wrapper 若持**监听 socket** 就需要 fd 传递（同 (d)）；若只做 supervisor（子进程持有端口）则重启子进程=短暂 EADDRINUSE 窗口 | 同 (a) | 中（wrapper 状态机 + 健康等待） | 零（外部脚本）/ 小（内建 `rick web --dev-supervise`） | wrapper 自身崩溃 → 全灭；需 wrapper 也常驻 | 可作为 (a) 的加固版：wrapper 只做「build → 探测旧 pid → TERM → 起新 → 等 health」，**不持有端口** |
| (c) 文件监听自动重建重启（air/reflex/自研 watcher） | 同 (a) | 同 (a)，且**每次保存都触发**（编译期中断频繁） | 中（本机无 air/reflex，`which air reflex` 空；自研 watcher 已有 polling 先例 watcher.go） | 小（新子命令）或零（外部工具+配置文件）；air 需联网 `go install` | **air 支持「构建失败保留旧进程」**（`stop_on_error=false`，注意默认已改为 true）；**reflex 不支持**（先 SIGINT→1s→SIGKILL 旧进程，再起新）；两者都无 one-shot build flag/自定义信号（§11） | 适合作为 dev 的**可选加速**，不建议作为唯一闭环（多次中间态重启会打断浏览器会话） |
| (d) `syscall.Exec` 自替换（保持 pid） | 保持同一 socket 只需把监听 fd 传给新映像（fd 默认 CLOEXEC，需 fcntl 清位；Go 里 `net.FileListener(os.NewFile(n))` 接管）。**叶子-2 已实测可行**：清 CLOEXEC 后 exec，pid 不变（192524）、fd 继承、accept 继续成功。若改为「启动子进程接管」，用 `os/exec ExtraFiles` 即可、**无需手写 fcntl**（§11） | 仍在 exec 瞬间丢弃 accept 队列之外的内存态；in-flight 请求随旧映像消失（可先 `Shutdown` 再 exec 则退化为 (a) 的语义） | 高（fd 号传递协议 + 双入口 argv 约定 + 新旧版本状态兼容） | 大（listener 创建、Serve、信号处理都要改） | 新映像启动失败 = 服务永久失联（无回退进程） | **不建议本期做**；收益仅是「pid 不变」，而 pid 不变对 dev 无实际价值（dev 可断） |
| (e) 代理前置（浏览器只连固定端口） | 代理端口固定，后端换端口后 reload 配置即可 | 代理层可对 SSE 透明（需 `X-Accel-Buffering:no` 已具备，sse.go:271），切换瞬间 in-flight 断开 | 中（多一个常驻代理进程 + 端口分配） | 零 | 代理挂了则全断 | 仅当前端固定端口/多实例分流有需求时值得；本场景 (a) 已够 |

**SO_REUSEPORT / socket activation 在本环境（叶子-2 实测）**：`syscall.SO_REUSEPORT` 常量在 Go 1.25 的 syscall 包里**不存在**（`undefined: syscall.SO_REUSEPORT`），必须硬编码 `0xf`（或引 x/sys）；用 `net.ListenConfig.Control + SetsockoptInt(fd, SOL_SOCKET, 0xf, 1)` 两个 listener 同端口**都能成功 bind**（对照组报 `address already in use`）。它的分发语义是内核按四元组 hash 选 socket，**不适合「新进程接管既有连接」**（老连接仍归老进程），本场景也不需要。socket activation：pid1 实测是 **systemd 249**（`is-system-running=starting`，unit 托管不可靠），但 `systemd-socket-activate` 二进制**可用**，实测 LISTEN_FDS/fd3 继承成功——机制等价于「fd 交接」，可自建不依赖 unit。**[测]**

**推荐结论**：dev 闭环 = (a) 为默认，脚本化「build 成功才重启 + 等 /api/health + 打印构建日志尾部 + 记录 build 指纹」（§4）；(c) 作为可选开发体验（只在需要高频试错时开）；(b) 只有当你希望「dev 自己重启自己」也无人值守多轮迭代时再加；(d)(e) 本期不做。**[推荐]**

## §3 Q3 自举问题（开发会话不能死在 dev 重启里）

**先把「会不会死」钉死**
- worker 是 web 进程的**直接子进程**且自成进程组：实测生产 web pid 172761 的子进程 176315（ppid=172761、pgid=176315 自身、`sess`=172761）。**[测]**
- worker 的 0/1 是**匿名 pipe**：`/proc/176315/fd/0 -> pipe:[483394971]`、`fd/1 -> pipe:[483972]`；代码同源 `cmd.StdinPipe()/cmd.StdoutPipe()`（supervisor.go:218-230）。匿名 pipe 无路径、无名字，**新进程无法重新打开或接管**。**[测]+[码]**
- pi 没有 socket/attach 模式：`pi --help` 只有 `--mode text|json|rpc` 与 `--session/--session-id/--fork`，无 listen/attach/socket 选项（实测 help 全文）。=> (iii) 的「新实例接管既有 worker」在不改 pi 传输层的前提下**不可能**。**[测]**
- 即使「不优雅退出」也救不了 worker：管道读端关闭后，worker 下一次写 stdout 就 EPIPE 自杀——叶子-2 用同构实验复现（node 子进程 stdout 接管道，父进程 SIGKILL 后子进程打印 `Error: write EPIPE (Unhandled 'error' event)` 随即退出）。**[测 叶子-2]**
- 退出路径是**主动杀 worker**：serve 的 ctx 取消 → `Sessions.ShutdownWorkers()`（server.go:129-131）→ `Supervisor.CloseAll()`（sessions.go:1745-1756）→ 每个 worker `abort → kill(-pgid, SIGTERM) → 5s → SIGKILL`（supervisor.go:492-536）。**[码]**
- 反例（不优雅退出）也不是出路：`kill -9` 会留下**孤儿 pi**，继续占着会话文件，导致下次 resume「新 worker 起来就死」——这正是代码注释里 job_69 的实测事故（sessions.go:1745-1750）。**[码]**
- 附加约束：worker 空闲 30 分钟（无非 response 事件）会被 `reapLoop` 主动 Close 并标 dead（supervisor.go:808-835 + 700-708「heartbeat 响应不算活动」）；`IdleTimeout` 在 web 组合根里未暴露开关（cmd/web.go:105-110 用默认）。=> 即使是「托管在 prod 的开发会话」，人离开 30 分钟也会掉线，需要 Resume（或后续加环境变量/flag）。**[码]**

**四方案对比**

| 方案 | 能否 dev 随便重启而开发会话不断 | 关键机制/前提 | 代价与代价证据 |
|---|---|---|---|
| (i) 开发会话托管在 **prod**，worker 的工作目录指向 dev worktree | **能**（dev 重启只影响 dev 进程自身） | `spec.Dir = ws.Path`（sessions.go:437-444/984-993），实测 worker cwd=其工作区路径；只需把 dev worktree 注册进 prod 的 `~/.rick/web.json`（`Add` 要求该目录含 `.rick/`，registry.go:107-118；dev worktree 若来自 git worktree 自带 `.rick/`） | prod 侧占用 1 个 MaxActive 名额（默认 8，supervisor.go:43）；会话出现在 prod UI/注册表；dev worktree 的 `.rick/jobs` 会与 prod 共用同一个工作区目录（因为 cwd=dev worktree，job 文件写在 dev worktree 内 → 反而更干净）；人类必须在 **prod** 的 UI 里看着这个会话 |
| (ii) 独立第三个常驻 supervisor（不经任何会被重启的实例） | **能**，且比 (i) 更强：prod 也可以在 dev 期间重启 | 起一个专用 `rick web`（第三个 HOME + 第三端口 + 独立 pid/注册表），它从不重启；开发会话在它名下 | 多一份常驻进程与状态；三个实例 → 三份注册表/覆盖层，人类要记「哪个 UI 管什么」；它的二进制**不能**被替换，否则等于重启（它托管的会话就断了）；仍受 30 分钟 idle reap |
| (iii) worker 与 web 解耦（setsid 脱离 + 新实例接管/重连） | 理论能，**当前实现不能** | 需要改传输层：把 web→worker 的 stdio 换成有名字的通道（unix socket/FIFO/一对稳定的 pipe fd 由 broker 持有），或引入 broker 进程（web 死、broker 与 pi 都活着，新 web 重连 broker） | 侵入面大（supervisor.go Spawn/Worker/scanLoop/pumpLoop 全改，runtime 契约变化）；pi 侧无需改，但 rick 侧要新增进程角色与握手/重放协议（events 的 Last-Event-ID 语义要重做）；失败模式更多（broker 崩溃=全灭） |
| (iv) 接受「重启即中断」，靠自动 resume 恢复 | **不能**（进行中的回合必丢） | 现状：`ReconcileOnStart` 只把 active/running 标 error（sessions.go:154-176），人工点 Resume 重新 `--session <piID>`（sessions.go:1050-1065）——**不回放被打断的回合** | 正在生成的回答/工具调用丢失，任务可能需要重跑；doing/dream 后台型会话连 Resume 都没有（409，sessions.go:1017-1026） |

**推荐（含备选与代价）**
- 主选 **(i)**：开发会话（improve rick web 的那个 AI 会话）跑在 **prod 实例**下，`workspace = dev worktree`。这样 dev 实例可以随便 `go build`+重启（甚至 `kill -9`），开发会话的进程关系完全不受影响；开发会话的所有工具调用（bash 里重启 dev 实例、npm build、curl dev 健康检查）都是 prod worker 的子进程，不会被 dev 重启波及。
- 备选 **(ii)**：若「prod 在开发期也可能被替换/重启」是硬需求，才上 (ii)——代价是多一个常驻实例和一层心智负担，收益是把 prod 也变成可重启对象。**注意 (ii) 的常驻实例自身就是当前架构下的单点**：它一旦重启，其托管会话同样全断（除非 (iii)）。
- **不要**把开发会话托管在 dev 实例上（否则每次后端试错都自杀一次）。
- (iv) 仍然必须实现（L6/L7 的 auto-resume）：虽然它救不了「开发会话」，但**最终把 prod 二进制替换上线时，prod 上所有 in-flight 任务都会被打断**——用户要求的「替换后主动恢复运行中的 job」只能靠它。两者叠加，本层对 L6 的依赖是硬依赖。
- 顺带：如果实现 (i)，**dev 实例可以使用 `kill -9` 快速重启**（实测 42ms 就绪），避免 10s graceful 长尾——前提是 dev 实例上不托管任何需要 resume 的会话（否则孤儿 pi 会污染 resume）。**[推荐]**

## §4 Q4 反馈回路：AI 会话如何可靠得知「dev 已重建重启成功/失败」

**现成机制清单（事实）**
| 机制 | 内容 | 可用性 |
|---|---|---|
| `GET /api/health` | 仅 `{"status":"ok"}`，免认证（routes.go:105-107、849） | 能证明**端口活**，不能证明**是哪个构建**。**[测]** |
| `GET /api/config`（需 token） | `{version:1, rick_version:"4.4.15", port, auth_required}`（routes.go:69-104）；`rick_version` 来自 `cmd/rick/main.go:10 const VERSION="4.4.15"`（静态常量，无 git sha/构建时间） | 只能区分版本号，区分不了「同版本的两次构建」。**[测]** |
| SSE `server_info` | 连接即发 `{rick_version, time, version}`（sse.go:14,417-425）；实测 payload | 浏览器/脚本都能拿，但同样无构建指纹。**[测]** |
| SSE `frontend_reload` | 覆盖层 dist 变更（watcher，延迟实测 6.8s） | 只对**前端**重建有意义。**[测]** |
| SSE `jobs_update` | 各工作区 tasks.json diff（watcher.go:107-181） | 可用于 doing 任务进度反馈。**[码]** |
| 构建产物本身 | `go build`/`npm run build` 的 stdout/stderr，全在 AI 会话的 bash 工具里 | 构建失败的**唯一**可靠信源。**[码]** |

**推荐封装（零改码版，立即可用）**
`scripts/dev-web.sh`（dev 实例专用，动作幂等、失败保留旧进程）：
1. `go build -o $DEVHOME/bin/rick.dev.<sha7>-<ts> ./cmd/rick`（**唯一文件名即构建指纹**）；失败 → 打印 stderr 尾部 30 行，`exit 2`，**不动旧进程**。
2. 成功后记 `$DEVHOME/build.json`（path/sha/size/built_at）。
3. 停旧进程：读 `$DEVHOME/.rick/web.pid`（singleton pid 文件，handler/web.go:112-121）→ `kill -TERM` → 最多等 12s（覆盖 SSE 场景的 10s graceful）→ 仍在则 `kill -9`（会留孤儿 pi，dev 实例上应确保无托管会话，§3）。
4. 启动新进程（`setsid nohup ... &`，env 白名单，§5）。
5. 等健康：轮询 `GET /api/health` 直到 200（超时 15s，失败 exit 3 并打印新进程日志尾部）。
6. **证明是新构建**（零改码技巧）：`readlink /proc/$(cat $DEVHOME/.rick/web.pid)/exe` 应指向第 1 步的唯一文件名；比对 `/proc/<pid>/exe` 的 sha256 与目标二进制（本机无 `diff`/`cmp`，用 `sha256sum`）。
7. 一行回执给 AI：`DEV_UP bin=rick.dev.<sha7> pid=<p> health_ms=<n> build_tail=<3 行>`；失败输出 `DEV_FAIL stage=build|stop|start|health detail=...`。
**建议的少量代码改动（更高保真）**：`-ldflags "-X main.VERSION=dev+<sha7>-<ts>"` + 在 `/api/config` 增 `build_id`（或让 `/api/health` 返回 `build_id`，仍免认证）——这样「新构建真的在跑」可以一条 curl 判定，且 SSE `server_info` 也会带上（sse.go:419 已传 rick_version）。**[推荐]**
**不要让 AI 依赖 SSE 判断重启**：SSE 是给人看的（浏览器 reload）；AI 侧只依赖脚本 exit code + health/构建指纹轮询。**[推荐]**

## §6 Q6 dev/prod 重启后「运行中的 job」如何主动恢复（与 L6 复用）

**现状可持久化字段**（SessionEntry，registry.go:234-254；sessions.json 实测样本一致）：`id / workspace_id / type / title / params{_method_file,_prompt_file,_job_id,job,requirement|topic|...} / status / pi_session_id / created_at / closed_at / archived`。resume 需要的三要素已有：`pi_session_id`（`--session <id>`，sessions.go:1050-1057）、`_prompt_file`/`_method_file`（重新注入 system prompt）、workspace 注册（`ws.Path` → worker cwd）。**[码]**

**缺口（必须新增，否则「主动恢复」无从下手）**
1. **「被打断时是否在回合中」无法判定**：`Busy` 是 `json:"-"`，注释明说「刻意不持久化」（registry.go:247-253）。缺一方位：`_busy_at`（RFC3339，agent_start 写、agent_end/agent_settled 清）。
2. **被打断的回合不会回放**：`--session` 只加载历史；要继续必须**再发一次 prompt**（nudge）。缺：`_last_user_prompt`（或消息 id）+ 一个固定「续跑」提示词模板（可用 `findSessionJSONL`/`parseSessionEntries`（sessions.go:1538/1570）离线读尾部自行推断，但不可靠，建议显式存）。
3. **最可靠的是「旧进程主动留遗嘱」**：在 `ShutdownWorkers()` **之前**把恢复清单原子写盘（`writeFileAtomic` 已存在，registry.go:355）到 `~/.rick/web/restart-manifest.json`，每条含 `{session_id, type, workspace_id, dir, pi_session_id, _method_file, _prompt_file, busy, busy_at, last_event_seq, job, job_num}`。新进程启动时：`ReconcileOnStart()`（cmd/web.go:111）→ 读清单 → 按类型自动 resume。这比「重启后反推」准确得多。**[推荐]**
4. **doing/dream 后台型会话目前不可恢复**：`SessionResume` 直接 409（sessions.go:1017-1026），因为它们是进程内 goroutine（`startBackground`，sessions.go:465-544），进程一死就没了。要「主动恢复运行中的 job」必须新增路径：用 `params.job/job_num` + `jobs/<job>/doing/tasks.json` 的任务前沿重跑剩余阶段。runner 侧没问题：组合根传 nil 时 `NewSessionManager` 会默认包 `handler.DoingIn/DreamIn`（sessions.go:95-121，cmd/web.go:107 传 nil 是有意为之）。**[码]**
5. 顺序与竞态：自动 resume 必须在 `ReconcileOnStart` 之后、且要限速（MaxActive=8，supervisor.go:43）——重启时若 10+ 会话同时 spawn，超出上限的会 `ErrMaxActive` 失败，需要排队重试。
6. dev 实例专用简化：dev 的托管会话**可以**直接不恢复（人类刷新即可），把 auto-resume 只在 prod 打开（用 HOME 区分或配置开关）。**[推荐]**

## §5 Q5 构建与依赖约束（叶子-1 实测，全部在 /tmp 拷贝里做）

| 约束 | 实测/事实 | 对 dev 循环的含义 |
|---|---|---|
| Go 工具链 | PATH 上的 `go` 是 **1.22.4**（`/tmp` 下 `go version`），仓库 `go.mod` 要求 `go 1.25.0` → `GOTOOLCHAIN=auto` 自动切到已缓存的 1.25.0 工具链（`~/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.25.0.linux-amd64`，**214MB**）；在仓库目录内 `go version` 显示 1.25.0。**[测]** | dev 构建脚本**不要**设 `GOTOOLCHAIN=local`（会报 `go.mod requires go >= 1.25.0`）；也不要清空 HOME 派生缓存（否则触发 214MB 工具链下载，见 §0.9） |
| `go build` 需要 dist | `//go:embed all:dist`（web/embed.go:35）在 `web/dist` 缺失/为空时**编译期硬失败**：`pattern all:dist: no matching files found`，无占位文件、无降级。dist 的 12 个产物文件**已入库**（`git ls-files web/dist`）且未被忽略。**[测]** | dev worktree 用 `git worktree` 即可自带 dist；**任何「清理 dist/加 .gitignore」的改动都会让全仓 `go build` 挂掉**（要改须同时加占位文件） |
| `go build` 耗时 | 热 0.44s；新 GOCACHE + `GOPROXY=off` 6.70s；冷机无网（无工具链/模块、无 vendor）**不可构建**。**[测]** | 后端重建可放进「保存即重启」循环无压力 |
| `npm ci` | 需要网或预热 cache：热 cache 2.74s；冷 cache 联网 8.71s；`--offline` 空 cache → `ENOTCACHED`；`--offline` + 预热 cache → 2.2s。node_modules 187MB/353 包。**[测]** | dev 首次用 `npm ci`，之后复用 `web/node_modules`（vite build 完全离线可跑） |
| `npm run build` | 3.7s（vite 2.36s），产出 dist 8 根文件 + assets 2 个。**[测]** | 前端 loop 秒级 |
| **必须在 `web/` 内就地构建**（重要） | 在 /tmp 干净副本构建出的 hash 与仓库/生产不同；把仓库 `web/dist` 一起拷进构建树后即复现仓库 hash。根因：`web/dist` 入库且未被忽略，**Tailwind v4 自动源扫描把旧 dist 里的类名也算进 CSS**（唯一规则数 976 vs 901，多 80 条）→ CSS hash 变 → Vite 把 CSS 依赖名内联进 entry JS → JS hash 也变。**[测]** | dev 实例的覆盖层若指向 worktree 的 `web/dist`，务必在 worktree 的 `web/` 里 `npm run build`（不要在 /tmp 副本里构建后拷贝，也不要删 worktree 的 dist） |
| 构建失败信号 | go/npm 的 stdout/stderr 只存在于调用者的 bash 工具 | 见 §4 的 `scripts/dev-web.sh`（失败打印尾部 30 行 + 保留旧进程） |

## §7 R7 上报项（无法在本环境澄清 / 需 human 或后续 job 决定）

1. **浏览器侧 SW 终校缺位**：本机无 chromium/playwright（`which chromium` 空）。原代码级推断已升级为 Node/vm 重放实证（`non-precached-url :: index.html`，confidence 0.9），但**浏览器内**的最终确认（DevTools → Application → Service Workers 是否控制页面）仍需人工做一次。**缺**：一次浏览器访问。
2. **`pi --mode rpc` 是否把「被打断回合」写进 session jsonl**（决定自动恢复是「nudge 续跑」还是「重发原 prompt」）：未做实验（需真实模型调用）。**缺**：一次受控的中断-恢复实验。
3. **prod 上「运行中的 job」现状口径**：实测 prod 6 个 web 会话中仅 1 个 active（easy job_72，busy=false），另有一个 CLI 会话 `rick easy --resume job_36`（pid 38109，**不在 web 注册表内**）。human 需要确认「所有在 rick web 上运行的任务不停止」是否包含 CLI 侧 job_36 这个会话（它当前天然不受 web 重启影响；若要搬到 web UI 会变成新 pi 会话，见 §3 建议）。**缺**：human 的口径确认。
4. **SO_REUSEPORT 的实际分流行为**（新旧 listener 谁收新连接）未测，仅验证双 bind 成功；本期方案 (a) 不需要它，故不阻塞。
5. **exec 自替换后 http.Server/Registry/Supervisor 的重建约定**未设计（方案 (d) 本期不做）；若后续要做，需要一份 fd 交接 + 状态重建清单。
6. **doing/dream 自动恢复的「任务前沿」语义**：`jobs/<job>/doing/tasks.json` 能否无损续跑（部分任务已跑一半、有副作用）需按 doing 的 stage 语义单独调研（L1/L6 范围）。
7. **`IdleTimeout` 无开关**：30 分钟空闲 reap（supervisor.go:44,808-835）会让「托管在 prod 的开发会话」掉线；要不要加环境变量/flag 调整，需 human 定夺（影响「会话不断」的可达性）。
8. **10s graceful + exit=1 是否要改**：有 SSE 时 dev 重启要多等 10s 且退出码非零（§0.6）。可选修法（SSE handler 监听 server 关闭信号 / 缩短 grace / wrapper 用 SIGKILL 兜底）各有取舍，需 human 选。

## §8 叶子证据文件

- `/workdir/sunquan20/AI_CODING/rick/.rick/jobs/job_36/doing/grilling/research-L5-leaf-1.md` — 构建/npm/embed/Tailwind 就地构建（叶子-1，worker）
- `.../research-L5-leaf-2.md` — SO_REUSEPORT/端口交接/syscall.Exec 保 pid/socket activation/进程树 EPIPE 实验（叶子-2，worker）
- `.../research-L5-leaf-4.md` — 外部文档：workbox non-precached-url、Go SO_REUSEADDR、tableflip/grace、air/reflex、EventSource 重连（叶子-4，researcher）

## §9 落地蓝图（推荐架构 + 可直接施工的参数）

**目标形态**：dev 实例与 prod 实例**同机共存、零代码改动隔离**；开发会话托管在 prod；dev 可随时重建重启。

```
prod 实例（不可动）: ./bin/rick web --listen 0.0.0.0 --port 8413     HOME=$REAL_HOME
                     └─ 托管开发会话的 pi worker（cwd = dev worktree）  ← 开发会话不断
dev  实例（随便重启）: <devbin> web --listen 127.0.0.1 --port 8414    HOME=$DEVHOME(.rick=隔离态)
                     ├─ dist 覆盖层 = $DEVHOME/.rick/web/dist → 软链 worktree/web/dist
                     └─ 不托管任何需要 resume 的会话（允许 kill -9 快重启）
```

**dev 实例启动命令（实测可用的隔离参数）**
```bash
REAL=$HOME; DEVHOME=/workdir/sunquan20/AI_CODING/rick-dev/.devhome   # 任意目录
env -i \
  HOME=$DEVHOME \
  PATH="$PATH" \
  RICK_PI_AGENT_DIR=$REAL/.rick/pi/agent \      # 复用 pi 配置/鉴权；dev 自己的 pi 目录无 provider 配置
  GOPATH=$REAL/go GOMODCACHE=$REAL/go/pkg/mod GOCACHE=$REAL/.cache/go-build GOENV=$REAL/.config/go/env \
  setsid nohup <devbin> web --listen 127.0.0.1 --port 8414 > $DEVHOME/web.log 2>&1 &
mkdir -p $DEVHOME/.rick/web; ln -sfn <devtree>/web/dist $DEVHOME/.rick/web/dist
cp $REAL/.rick/config.json $DEVHOME/.rick/config.json    # pi_extra_args(provider/model) + 可选 web_token
```
要点：`GOPATH/GOCACHE/GOENV` 必须**显式固定到真实 HOME**（否则每次 `go build` 丢缓存、可能重新下载 214MB 工具链，§0.9 实测）；`RICK_PI_AGENT_DIR` 固定到 prod 的 agent 目录（内部既有 `RICK_PI_AGENT_DIR` 开关，internal/runtime/agentdir.go:10-30）。singleton 冲突天然消失（pid 文件跟随 HOME，handler/web.go:112-121）。

**把开发会话挂到 prod（现有 API 即可，零改码）**
1. `POST /api/workspaces {path: <devtree>}`（prod，需 token）——要求目录含 `.rick/`（registry.go:107-118；git worktree 自带）。
2. 在 prod UI 里对 `AI_CODING/devtree` 工作区新建会话（类型建议 `easy`/`plan`）→ worker cwd=devtree（实测机制见 §0.11）。
3. 人类在 **prod** 的 UI 跟踪该会话；dev 实例的 UI 只用于验证被改造的界面。

**每次改动的闭环（AI 会话在 prod 上执行）**
| 改动类型 | 动作 | 生效路径 | 备注 |
|---|---|---|---|
| 前端（web/src） | 在 `<devtree>/web` 里 `npm run build`（3.7s） | worktree dist 变化 → dev 实例覆盖层（软链）→ dev 浏览器 `frontend_reload`（实测 +6.8s）| 必须在 worktree 内构建（Tailwind 扫 dist，§5）；prod 前端上线另走 `~/.rick/deploy-web.sh`（拷贝到 `~/.rick/web/dist`，**不重启**）|
| 后端/CLI（*.go） | `scripts/dev-web.sh`：`go build`（热 0.44s）→ TERM 旧 dev → 起新 → 等 health → 校验 `/proc/pid/exe` 指纹（§4）| dev 实例新二进制 | 无 SSE 客户端时重启仅 ~10ms；有 SSE 时 10s 且 exit=1（§0.6）|
| pi runtime/扩展 | 改 `$REAL/.rick/pi/agent/{settings.json,extensions,...}` 后重启 dev（pi 在 spawn 时读取，runtime/agentdir.go:68-71）| dev 实例 | 注意 prod 的 worker 也读同一份配置 → 改 pi 配置**会影响 prod**：建议先复制成 dev 专用 agent 目录再切换 `RICK_PI_AGENT_DIR`（本次未验证 pi 配置最小集）|
| 上线到 prod | 人工确认 → ① 前端 deploy-web.sh；② 后端：更新 `<repo>/bin/rick` + 写 restart-manifest + TERM prod + 起新 + auto-resume | prod | 依赖 L6 的 auto-resume；prod 上有 SSE 时停机 ≥10s（可接受，但要写进操作手册）|

**监控需求（dev 实例建议加的最小遥测，便于 AI 判断）**：① `/api/config` 增 `build_id`（或 health 带上）；② dev 实例日志单独文件 + `build.json`；③ 可选：`POST /api/dev/restart`（只在 dev 实例注册，由 HOME/flag 门控）让 AI 不必依赖 shell 脚本管理进程。**[推荐]**

## §10 跨层约束与 blocker（与 L6/L1 的接口）

1. **blocker：dev 实例绝不能只换端口、共用 HOME。** 同 HOME 时第二个实例不会被告警拦住——生产当前**没有** `~/.rick/web.pid`（`ls ~/.rick/web.pid` → No such file，而 prod 172761 正在跑；原因未定，L6 亦实测同结论），因此 singleton 门禁失效；更严重的是第二实例启动时 `ReconcileOnStart()` 会把注册表里 active/running 的会话**改写为 error 并落盘** `sessions.json`（sessions.go:158-176 → registry.save），即「起个 dev 看看」会污染生产会话状态。**必须 HOME 隔离**（实测：HOME=/tmp/… 时 dev 自建 pidfile、workspaces=0，prod 的 `~/.rick` 原封不动）。**[高危/测]**
2. **给 L6 的三处输入**：① 提升（promote）那天 prod 必重启，若开发会话托管在 prod 会一起死 → 要么开发会话改用 (ii) 第三常驻 supervisor，要么接受「最后一次提升前把开发会话收尾」；② 恢复清单必须在 `ShutdownWorkers()` **之前**写盘（server.go:129-131 的顺序是 worker 收尸先于 HTTP drain）；③ 有 SSE 连接时 prod 停机 ≥10s 且进程退出码 1（§0.6），门禁/操作手册不能把 exit=1 当失败信号。
3. **与 L6 的结论一致性核对（无冲突）**：浏览器侧 SSE 断线自愈（L6 §6 ↔ 本简报 §2）；`Busy` 不落盘（L6 §7 ↔ §6 缺口 1）；per-turn 中断判定（L6 §9 用 pi jsonl 的 `stopReason=toolUse` 无后续 toolResult）可以把 §6 缺口 2 从「必须新增 `_last_user_prompt`」降级为「jsonl 判定 + 固定 nudge」；但 L6 也提示**重复副作用**风险 → 自动续跑应默认关闭，只自动恢复空闲 worker。
4. **构建输出路径的碰撞**：prod 的运行体就是仓库里的 `bin/rick`（`/proc/172761/exe → /workdir/.../rick/bin/rick`，start-web.sh 硬编码 `cd <repo> && exec ./bin/rick`）。dev 构建若直接写 `<repo>/bin/rick`，虽不影响已在跑的进程，但会**覆盖待回滚/待提升的产物**。建议 dev 一律输出到 `<repo>/bin/rick.dev.<sha7>-<ts>`（唯一名同时充当构建指纹，§4），提升时再由 L6 的原子替换流程变成 `bin/rick`。**[推荐]**
5. **ai 会话可用的最省心闭环（一句话）**：`go build -o bin/rick.dev.<sha7>` → `scripts/dev-web.sh` 重启 dev（TERM→等健康→校验 /proc/pid/exe）→ 在 **prod** 的 UI 里继续对话；前端改动用 `npm run build`（worktree 内）+ 软链覆盖层，浏览器 ~7s 自动刷新（或 F5）。

## §11 外部信源交叉验证（叶子-4，均为一手文档/源码）

- **SW 结论互证**：workbox 源码 `if (!cacheKey) throw new WorkboxError('non-precached-url',{url})` 在 **sw.js 顶层求值期同步抛出** → ServiceWorkers 规范语义「script evaluation failed」= **整份 SW 首次注册失败、所有 route 失效**（若为更新场景则旧 SW 继续生效）。vite-plugin-pwa 官方文档明确：配置 `globPatterns` 时 **MUST include all your assets patterns**（默认 `**/*.{js,css,html}`），去掉 html 就会生成会崩的 sw.js，且**构建期无校验**（issue #120/#402/#731）。=> 本仓现状（sw.js 必然抛错）**不是环境偶发，是配置缺陷**；修法：`globPatterns: ["assets/**","index.html"]`（或去掉 `createHandlerBoundToURL` 调用）。**[档 高]**
- **Go 监听**：`net` 在 bind 前显式 `SetsockoptInt(SO_REUSEADDR,1)`（go.dev/src/net/sockopt_linux.go）→ 我实测的「SIGKILL 后 42ms 重绑 / TIME_WAIT 不阻塞」得到机制解释；标准库**不设** SO_REUSEPORT（golang/go#23696 维护者原话），且 `syscall` 在 Linux 无该常量（#26771 Unplanned）→ 实务用 `ListenConfig.Control` + `golang.org/x/sys/unix`，或如叶子-2 实测**硬编码 0xf**（零新依赖）。SO_REUSEPORT 的内核语义是 4-tuple hash 分发、SYN 期即绑定某 listener → 旧 listener 在组内时新连接可能被旧 socket 收走 → **不适合「新进程接管」**，平稳重启应改用 fd 传递（单一 accept 队列）。**[档 高]**
- **fd 交接的实现选项**：若要启动**子进程**接管监听 socket，Go 官方路径是 `os/exec ExtraFiles`（entry i → fd 3+i，runtime 会自动处理 CLOEXEC），**无需手写 fcntl**；启动器范式参考 cloudflare/tableflip（父进程等子进程 ready 再退出）与 facebookgo/grace（从 fd 3 起 `net.FileListener` 继承，兼容 LISTEN_FDS）；坑：`exec.Cmd.Start()` 会对 ExtraFiles 调 `File.Fd()` 清 O_NONBLOCK 并禁用 runtime poller，须传 dup 后的 os.File。裸 `syscall.Exec` 则必须自己清 FD_CLOEXEC（叶子-2 已实测可行：pid 不变 + accept 继续）。**[档 高 + 测]**
- **socket activation**：是 service manager 行为（LISTEN_PID/LISTEN_FDS、首 fd=3；`LISTEN_PID != getpid()` 即返回 0）；容器默认 PID1 非 systemd ⇒ 不可用（systemd.io/CONTAINER_INTERFACE/）。本机虽有 systemd 249 但 `is-system-running=starting`，叶子-2 用 `systemd-socket-activate` 独立验证了机制。=> **不要依赖 unit 托管**，要平滑交接就用 fd 传递。**[档 高]**
- **air vs reflex（方案 (c) 的能力边界）**：air 可「构建失败保留旧进程」（`stop_on_error=false`；注意默认值已被 PR #836 改为 true，需显式关掉）；**reflex 不支持**——重启时先 SIGINT 旧进程、1s 后 SIGKILL，旧进程死后才启动新的（即前端热更会经历「先停后起」的中断）。**两者都没有 one-shot build 官方 flag，也都不支持自定义重启信号**（air 只能 SIGINT）。=> 若走 (c)，air 更贴合「构建失败不破服务」的要求，但都要额外引入二进制（离线需预装）。**[档 高/中]**
- **SSE 重连**：自动重连 + `Last-Event-ID` 是 HTML 规范行为（MDN/WHATWG），但**规范未定义「Last-Event-ID 已不可用（如 seq 归零）」的处置**（whatwg/html#8297 仍在要求补）→ 本仓自理的方案（服务端 replay buffer 失配就回 `server_info{reason:"replay_overflow"}`，客户端清游标 + resync）与「id 单调且含 boot epoch」的建议方向一致；L6 §6 已确认前端自愈路径。**[档 高 + 码]**
