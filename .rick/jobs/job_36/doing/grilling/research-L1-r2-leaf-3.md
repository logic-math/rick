# 调研：手机伴侣/远程会话工具架构 → rick web ui（Go 服务 + pi rpc 子进程 + 多端浏览器 + 单用户 token）可借鉴先例

**① Happy Coder**（happy.engineering ｜ github.com/slopus/happy）
- 架构：三层分布式。本地 `happy-cli`（Node.js，包装 Claude Code/Codex 子进程执行并流式采集终端状态）→ 云中继 `happy-server`（Node.js+Fastify，PostgreSQL+Prisma、Redis Pub/Sub）→ 多端客户端（React Native iOS/Android + macOS 桌面 + Web），支持自托管。
- session 同步/relay：CLI 与手机经加密中继双向实时同步；同一 session 可手机↔终端无缝接力继续。
- 断线重连/加密：全链路端到端加密，服务器仅转发、不可读明文；会话持久化于 PostgreSQL，客户端重连后拉取恢复；agent 需权限/提问时推送通知，手机回复后送回。
- 借鉴：权限请求推送通知；会话状态锚定在服务端而非前端；单用户场景可去掉云中继、Go 服务直连多端。

**② omlet**【未找到】未检索到名为"omlet"的手机伴侣/远程会话工具。近似项均非目标：Project-Omelet/claude-code（Claude Code harness 的 clean-room Python 重写，非远程客户端）；omlet.co（Open Metrics Logs Events Traces 可观测性/AI SRE 平台）。

**③ setsudo**【未找到】仅找到 Mac 专注提示小工具 Setsudo(節度)（notes.rodrigofranco.com），与编码代理/远程会话无关。

**④ 其他 Claude Code/Aider/Codex GUI 或远程方案**
- **opcode**（github.com/winfunc/opcode）｜架构：Tauri2 桌面 GUI，Rust 后端管理 Claude Code 进程；web server 模式＝REST API+WebSocket 完整镜像桌面能力，手机/浏览器接入（Browser UI⇄WS⇄Rust Backend⇄子进程）｜session：会话归后端所有，桌面/Web 均为视图｜重连/加密细节未能获取（fetch 受阻）｜借鉴：后端独占子进程与会话状态、多端仅 WS 视图，形态最贴 rick。
- **Conductor**（conductor.build）｜架构：Mac 原生 app，git worktree 隔离并行多个 Claude Code/Codex agent，复用 ~/.claude（skills/hooks/MCP）、auth passthrough，集中 review/merge｜非远程工具｜借鉴：worktree 隔离并行 agent、凭据直通不中转。
- **Sst Opencode**（opencode.ai/docs/server）｜架构：client-server；`opencode serve` 起 headless HTTP 服务（OpenAPI+SSE/WS），`opencode attach` 让 TUI 连远程 daemon，`opencode web` 另起独立实例；OPENCODE_SERVER_PASSWORD 环境变量＝单密码 HTTP basic auth；桌面"web server mirror"用 TCP 反代实现桌面 UI 1:1 镜像（PR#12000）｜坑：web 为独立进程，内存态/SSE 事件与桌面不共享｜借鉴：serve/attach 分离、环境变量单用户认证、SSE 事件流+状态快照重连、避免双实例状态分裂。
- **Claude Code 官方 Remote Control**（code.claude.com/docs/en/remote-control）｜本地 session 经 claude.ai/code 或 Claude app 从手机/平板/任意浏览器接力；agent 始终跑在本机，文件系统/MCP/配置保留｜借鉴："会话锚定服务端、客户端即插即用"的产品定位。
- **轻量自托管**：fafawlf/claude-code-web（SSH 隧道自托管 web UI）、CodeRemote（coderemote.dev，Tailscale 内网直连、无云服务）、C3（getc3.app，自托管 hub+E2E 加密+多 agent）｜借鉴：rick 单用户可用"token+内网/隧道直连"，无需公网中继与 E2E。

【结论】
1. Go 服务独占 pi 子进程与会话状态，浏览器仅作 WS/SSE 视图（opcode/opencode 模式）。
2. 断线重连靠服务端事件日志+全量快照回放，状态勿存前端。
3. 单用户 token 认证已足够；E2E 加密仅在公网中继场景必要。
