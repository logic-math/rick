# research-L1-r2 简报：pi 生态 web UI 实现方案与成熟架构参考

元信息：L1-r2（架构裁决 C 方案后的补充调研）| 主题：pi 生态 web ui 工具 + session chat UI 模式 + 子进程/SSE 架构细节 | 时间基准 2026-08-29 | 方法：自查 pi 一手文档/示例 + 4 叶子 fanout（清单见末尾）

## A. pi 生态 web UI / 远程访问

### 1. pi 生态现有 web ui / 远程工具盘点（human 核心诉求；leaf-2 + research 独立复核）

**总判断（高置信 0.85）**：pi 无第一方 web UI；官方集成面=Extension/RPC/SDK 三轨；远程化在上游路线图中。第三方生态已长出一批 web ui（npm+pi.dev/packages 画廊），最成熟两强=pi-web 与 pi-agent-dashboard。

| 工具 | 架构（怎么起服务/怎么连 pi） | 成熟度 | 可借鉴点 |
|---|---|---|---|
| **pi-web**（github.com/jmfederico/pi-web）【已复核】 | 长驻 session 守护进程（持会话）+Web/API 服务双进程；systemd/launchd 托管；浏览器断线会话存活；亦以 pi 包分发（/pi-web 命令） | ⭐620/fork121，npm 周下载~2.7k，2026-05 创建，活跃（pi-web.dev） | **会话所有权与 UI 进程分离**：会话活在工作区守护进程，UI 只是监督/重定向/评审视图 |
| **pi-agent-dashboard**（github.com/BlackBeltTechnology/pi-agent-dashboard） | 三件套：全局桥接 extension（每会话注入、识别 TUI/Zed/tmux 来源转发事件）+HTTP/WS 网关（:9999）+浏览器端；与 pi TUI 共存 | ⭐253，npm 在架 | extension 桥接+WS 网关模式；mDNS/zrok 免公网移动远程 |
| **pi-web-ui**（xing-shuyin） | pi SDK 进程内跑 agent（无子进程），WebSocket 推流；一键启动+Docker/systemd | ⭐41 但 npm 月下载~2 万、迭代极快（v0.47） | SDK 直嵌最省事；主题/插件化 |
| **pi-remote-web-ui**（VVander） | 服务仅绑 127.0.0.1:8080，SSH 隧道访问；单进程内 AgentSession 跨标签页共享 | ⭐33 | loopback+SSH 隧道安全模型（单用户最简形态） |
| **@firstpick/pi-package-webui** | pi install 扩展包+独立 pi-webui CLI 控制 spawned pi 会话，默认 127.0.0.1；姊妹包 /remote=LAN+PIN+二维码手机直连 | 周下载 363（remote 包） | pi 包形态分发；PIN+QR 移动接入 |
| **OpenClaw**（openclaw/openclaw） | pi SDK createAgentSession() 进程内嵌入自研 Gateway（官方 sdk.md 点名参考） | 高星（数值未核） | SDK 深度嵌入+多渠道网关成熟范本 |

**上游路线图信号（战略输入，research 已独立复核）**：pi 官方在建 `@earendil-works/pi-server`/`pi-client`/`pi-protocol` 分布式会话架构（CBOR 协议+lease 管理，README 标 Experimental）；issue #8481（TUI 本地跑、RemoteSession 服务端持会话/工具/文件）、PR #7344（remote session wire protocol）、issue #4737（TUI 连 RPC backend）。**pi 官方 server mode 方向明确但未稳定** ⇒ rick Go+rpc 子进程架构当下正确（rpc 协议已稳定），但 pi 侧驱动层应做成可替换接口，为未来切官方 PiServer 留退路【高｜github.com/earendil-works/pi issues/PR/packages 实证】

### 2. 本地官方 examples 盘点（leaf-1 全量 + research 自查锚点）

- **关键否定性结论：官方 ~75 个 extensions 示例 + 13 个 sdk 示例无一启动 HTTP/WS server（零端口零鉴权）**——pi 官方对外集成范式=rpc/json 子进程+extension 事件钩子；web 服务层只能宿主侧自建【高｜examples 全量盘点，代码原文 0.4】
- examples/rpc-extension-ui.ts = 官方「自建 UI on RPC」参考实现：spawn `pi --mode rpc --no-session --extension <path>`，stdin JSONL 命令/stdout 行读事件；演示 extension_ui_request/response 子协议（extension 的 select/confirm/input/editor 桥接为客户端对话框，web 端可映射弹窗；notify/setStatus/setWidget/setTitle/setEditorText 单向推送）【高｜源码+rpc.md L1264-1345】
- subagent/index.ts = pi 官方子进程管理范本：每任务 `pi --mode json -p --no-session` 一进程；JSONL 手工分帧（buffer+split("\n")+残尾保留）；abort=SIGTERM→5s→SIGKILL；MAX_PARALLEL_TASKS=8/MAX_CONCURRENCY=4；单任务输出 50KB 上限；启动探活 500ms 查 exitCode+stderr 缓存；getPiInvocation() 区分 dev/prod 解析 pi 路径——全部可直接移植 rick Go 侧【高｜源码】
- examples/sdk/ 13 个（01-minimal…13-session-runtime）全为 Node 进程内 SDK 用法，Rick 已裁决 Go+rpc，仅作能力面参考【高｜examples/sdk/README.md】
- 其余集成类：ssh.ts（工具 ops 可插拔整体远程化）、gondolin（QEMU 微 VM）、notify.ts（OSC 通知）、event-bus.ts（pi.events 扩展间总线）、file-trigger.ts（fs.watch→pi.sendMessage 注入）、dynamic-resources、with-deps【高｜源码】

### 3. 手机伴侣 / 远程会话类工具（leaf-3，与「云端同步多端适配」同构先例）

**结论（中高置信 0.75）**：
- **Happy Coder**（happy.engineering，slopus/happy）：三层=本地 happy-cli（包装 Claude Code/Codex 子进程并流式采集）→ 云中继 happy-server（Fastify+PostgreSQL+Redis Pub/Sub）→ 多端（RN iOS/Android+macOS+Web），可自托管；全链路端到端加密（服务器仅转发不可读明文）；会话持久化 PG、重连拉取恢复；agent 权限请求→推送通知→手机回复送回。借鉴：**权限请求推送通知**、会话状态锚定服务端；单用户可去云中继【高｜官网+GitHub】
- **omlet / setsudo：未找到**（近似项均非目标）【低——R7】
- **opcode**（winfunc/opcode）：Tauri2 桌面+Rust 后端管 Claude 进程；web 模式=REST+WS 完整镜像桌面能力；**会话归后端所有，桌面/Web 均为视图**——形态最贴 rick【中高｜源码】
- **Sst Opencode**：`opencode serve` 起 headless HTTP 服务（OpenAPI+SSE/WS），`opencode attach` 让 TUI 连远端 daemon；OPENCODE_SERVER_PASSWORD=单密码 basic auth。反面教材：web 独立进程致内存态/SSE 与桌面不共享（双实例状态分裂）——**印证 rick 中心实例单写点裁决**【高｜官方 docs】
- **Claude Code 官方 Remote Control**：本地 session 经 claude.ai 从手机/任意浏览器接力，agent 始终跑本机——「会话锚定服务端、客户端即插即用」同 rick 定位【高｜官方 docs】
- 轻量自托管：fafawlf/claude-code-web（SSH 隧道）、CodeRemote（Tailscale 直连）、C3（自托管 hub+E2E）。**单用户 token+内网/隧道直连即够，无需公网中继与 E2E**【中｜GitHub】

## B. session 中心化 chat UI 交互模式

### 4. 主流 agent web UI 的 session 管理（leaf-4；OpenHands/claude-code-webui/goose/LibreChat）

**结论（中高置信 0.75｜deepwiki+源码）**：
- **新建流程**：主流均「先进聊天后补配置」，参数作为会话属性随会话持久化（LibreChat conversationPreset；OpenHands 启动表单选 repo+profile 属少数派）⇒ **rick 的 cmd 类型+参数宜做成会话属性（窗内可改），而非前置表单门槛**；新建向导仅作快捷入口
- **resume 数据流**：统一「侧栏列表（时间/项目分组+状态点）→按 id 拉事件/消息重放重建」（OpenHands WS 回放/claude-code-webui --resume+回读 JSONL/goose SQLite sessions.db/LibreChat 分页拉消息树）——与 pi session JSONL+get_entries(since) 游标天然对齐
- **渲染分层共识**：文本 delta 直渲（Zustand 批处理防高频卡顿）＋工具调用折叠分组卡（≥2 连续自动折叠）＋thinking 独立折叠＋错误二分（banner 可展示错误 vs AgentErrorEvent 内联聊天）＋流式与最终消息合并去重
- **传输**：claude-code-webui/LibreChat 用**可恢复 SSE**（POST 发起+GET EventSource 订阅分离、断线指数退避、导航离开不中断生成）；OpenHands 用 WS 双连接——SSE 可恢复流是成熟标配，与 r1 选型一致

### 5. 移动端适配具体做法（leaf-4）

**结论（中高置信 0.7）**：断点共识 768px（LibreChat useMediaQuery）/1024px（OpenHands）切 desktop/mobile 布局；移动端侧栏折叠为 drawer 覆盖态或状态点；输入区常驻底部（claude-code-webui 移动体验为官方卖点）；**Tailwind 断点须与 JS 断点对齐**（LibreChat 曾有 bug）、跨断点 resize 避免整页重载（OpenHands 已知 bug：长会话 30s+ 重载）——rick 前端验收项。输入区 fixed 底部+VisualViewport 处理软键盘（承接 r1 PTY 结论）

## C. 架构细节补充（research 自查）

### 6. 多 pi rpc 子进程管理实践

**结论（高置信 0.85｜官方 rpc.md+extensions.md+Go 惯用法检索）**：
- pi rpc 无 quit/ping/keepalive 命令（rpc.md 命令全集核对）：健康检查用 `get_state` 心跳（含 isStreaming/isCompacting/sessionId/sessionFile，恰是 kill-safety 与路由所需）；进程退出只能靠信号【官方文档】
- switch_session 语义=同进程内**替换**活动 session（非并存）：旧 session 走 session_shutdown、runtime 拆除重建、旧绑定对象作废 ⇒ **单子进程只能串行浏览，无法并发承载多聊天窗口事件流**（pi 一进程一活动 session；多窗口并发必须一 session 一进程，与已裁决一致）。正确用途：①空闲进程 resume 历史 session（省冷启动）②read-only 预览，配合 get_entries(since) 增量回放【官方】
- Go 侧成熟模式：exec.Cmd+ctx（Go1.20+ Cancel/WaitDelay：ctx 取消先 SIGTERM、超时自动 SIGKILL）；**Setpgid 进程组+kill(-pgid)** 防 pi 孙进程（bash 工具）成孤儿；stdin/stdout/stderr 三管道各配常驻 reader goroutine（不读满管道死锁子进程）；Wait 必被调用防僵尸；崩溃恢复=respawn+switch_session(原 jsonl)+get_entries(since) 补流【Go 标准库+社区 supervisor 实践】

### 7. Go 侧 SSE 实现要点

**结论（高置信 0.9｜Go 官方 issue+SSE 专题站+业界惯例）**：
- 标准库即可：handler 设 `Content-Type: text/event-stream`，逐事件写+`http.Flusher.Flush()`（net/http 默认缓冲，不 flush 客户端看不到）；每连接 goroutine select{ctx.Done(), per-client chan, heartbeat ticker}
- 心跳：15-30s 写 SSE 注释行（`: ping`）防中间层掐空闲连接、检测死对端
- 断线重连：EventSource 自动重连并回带 `Last-Event-ID` 头——服务端每 session 维护单调 id+重放环形缓冲，重连按 Last-Event-ID 补发；EventSource 遇 HTTP 错误码会停止重连，handler 须一直 200
- 多路汇聚：每 pi 子进程一个 stdout scanner goroutine→按 sessionId 分发进 per-session 事件总线（chan fan-in）→每 SSE 连接一个 buffered chan 订阅；慢消费者背压=丢旧保新（快照类事件可合并）或踢线重连【blog.pranshu-raj.in 28K 连接实践】
- 代理坑（承接 r1）：自反代 FlushInterval=-1，nginx 加 X-Accel-Buffering:no；**HTTP/1.1 每域名 6 连接上限——多窗口多 SSE 需 HTTP/2 或单流复用（L2 决策点）**【Go issue #47359/#64045】

## 对 L1 终判与 L2 设计的输入

### ① pi 生态 web ui 工具借鉴清单（哪些架构模式直接可用）
1. **会话所有权与 UI 进程分离**（pi-web 核心模式）：会话活在工作区/守护层，UI 只是视图——rick「Go 服务持 pi 子进程+浏览器纯视图」与生态最强先例一致，终判可固守
2. **opcode/opencode 模式**：后端独占子进程与会话状态，桌面/Web 仅 WS/SSE 视图；opencode 双实例状态分裂是反面教材，印证中心实例单写点裁决
3. **subagent/index.ts 子进程范本直接移植 Go**：JSONL 手工分帧、SIGTERM→5s→SIGKILL、并发/输出上限、启动探活
4. **extension_ui_request/response 双向桥必须实现**（映射 web 弹窗/确认卡），否则 human-loop 会话 web 端走不通；permission-gate 类审批即走此通道
5. **Happy Coder 推送通知模式**：agent 权限/提问→push 手机→回复送回——rick 后续可加 Web Push，是 human-loop 移动体验关键
6. **访问安全三档**：loopback+SSH 隧道（最简）→LAN+PIN+QR（firstpick）→公网中继+E2E（Happy）——rick 单用户 token 对应第一档+可选第二档
7. **上游 PiServer 实验包**：pi 侧驱动层做成接口抽象，未来可换官方 server mode；勿在 Go 侧做 rpc 协议之外的状态假设
8. 交互设计：cmd 类型+参数=会话属性（窗内可改）不作前置表单；渲染分层照 leaf-4 共识（delta 直渲/工具折叠卡/thinking 折叠/错误二分/流式与终稿合并去重）

### ② env 管理前端源码 vs embed dist 提交仓库（建议）
**建议：前端源码同仓库 + 预构建 dist 提交仓库（AdGuardHome 模式）+ 开发态 vite dev server 代理**。理由：a) r1 已证先例充分（AdguardHome embed 预构建 build/、kubeaquarium 为 go install 免 node 提交 dist、Vaultwarden 独立仓库 bw_web_builds、CI 可校验 dist 新鲜度）；b) pi-web/pi-web-ui 等同类项目均 node 技术栈直接构建，rick 差异化恰是「node 不进 rick 构建链」——提交 dist 是唯一同时满足 go install 可用+无 node 依赖的方案；c) 源码同仓库避免双仓库同步成本（bw_web_builds 模式适合大团队，rick 单人过重）。配套：改前端后 `make web-dist` 重生成随源码提交；CI 校验重构建一致性（可选）。

### ③ session 子进程管理模型建议
1. **每 active chat session 一个 `pi --mode rpc` 子进程**（终判维持）；close=abort（若 isStreaming）→SIGTERM→grace 5s→SIGKILL，全程 Setpgid 进程组防孙进程孤儿
2. **Supervisor 结构**：registry(map[sessionID]*Proc)；每 Proc=exec.Cmd+stdin writer+stdout 行扫描 goroutine+stderr drain goroutine+pending request map(id→chan)+idle timer；三管道全配常驻 reader
3. **健康检查**：无官方 ping——get_state 心跳（10-30s）；stdout EOF=进程死亡→按需 respawn+switch_session(原 jsonl)+get_entries(since) 补流
4. **switch_session 只作串行复用**（空闲进程 resume 历史/read-only 预览）；不得用于并发多窗口
5. **closed 会话浏览不起进程**：直接读 session JSONL（~/.rick/pi/agent/sessions/，pi session-format v3）离线渲染；resume 才 spawn 新进程
6. **上限与回收**：参照 pi 官方 subagent 量级（MAX_PARALLEL=8）设 active 进程上限+idle 超时（如 30min）自动 close；doing 一次性任务维持现状 --mode json 无需常驻

## R7 上报项
1. omlet / setsudo 未检索到（leaf-3【未找到】）——若 human 有确切项目名/链接请提供
2. OpenClaw star 数值未核实（leaf 报 38.7 万，量级存疑）——不影响架构结论，仅成熟度描述
3. pi 上游 PiServer/pi-protocol 实验包稳定时间表未公开——影响未来切换官方 server mode 时机
4. pi rpc 无官方 keepalive/ping：get_state 心跳为推断方案，协议层无此语义，未经官方确认
5. HTTP/1.1 浏览器 6 连接上限对「多窗口多 SSE」影响需 HTTP/2 或单流复用决策——L2 设计点，未原型验证
6. pi-web/pi-agent-dashboard 源码级进程拓扑未逐行核读——star/文档级证据已足，L2 需要时再深挖

## 叶子文件清单（均含结论与信源链接）
- research-L1-r2-leaf-1.md（本地 examples 全量盘点+集成模式详解）
- research-L1-r2-leaf-2.md（pi 生态 web ui/远程工具盘点，含上游路线图）
- research-L1-r2-leaf-3.md（Happy Coder/opcode/opencode/Remote Control 等）
- research-L1-r2-leaf-4.md（OpenHands/claude-code-webui/goose/LibreChat+移动端）
（research-L1-r2-leaf-N.out.md 为各叶子原始回执副本，内容同 leaf-N.md）
