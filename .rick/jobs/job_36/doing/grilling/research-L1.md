# research-L1 简报：rick web ui L1 架构调研

元信息：L1 总体架构层 | 2026-08-29 | 详录：同目录 research-L1-A-detail.md + 3 叶子文件（自查 pi 文档 + fanout 联网）

## 事实性结论（12 条）

1. pi SDK 完备支持自定义 UI：createAgentSession/ModelRuntime/SessionManager/DefaultResourceLoader 公开导出，官方用例明列 web/desktop/mobile UI【高｜官方 sdk.md】
2. session.subscribe 覆盖 text/thinking/toolcall delta、tool_execution 三段（含累积 partialResult）、消息/轮次/agent 生命周期、steer 队列、compaction/retry；无独立 subagent 事件——子代理=subagent 工具的 tool_execution_* + extensions/subagent/runs/<id>/events.jsonl 细粒度轨迹【高｜官方】
3. SDK 多 session 并发架构可行（session 相互独立、无单例），但无官方多 session 先例佐证；进程级共享=globalThis.fetch 幂等补丁（undici 池，idle timeout 全局不可 per-session）【中高 0.7｜源码+文档推断】
4. --mode rpc 协议完备：prompt/steer/abort/bash 流式/switch_session/fork/get_entries（since 游标跨客户端重启增量）/extension_ui 对话框桥接；严格 LF JSONL（Node readline 不合规）；官方定位=跨语言子进程集成，Go 可直接 spawn【高｜官方 rpc.md】
5. extension 可起 HTTP/WS server，但生命周期=session 存活期且 pi 一进程一活动 session ⇒ extension 路线只能覆盖单会话，「web UI 作为 pi extension 无法承载完整 rick 多 job」判断成立【高｜官方 extensions.md】
6. 断线重连恢复是现成能力：session JSONL 树(v3)+get_entries since 游标+leafId 单往返判活+SessionManager.open/fork 重建；历史 job 会话可离线重放（rick 已落盘 ~/.rick/pi/agent）【高｜官方+rick 代码】
7. 竞品清一色 headless 事件流+自绘 UI：OpenHands(React+WS 事件日志)/goose(goosed REST+SSE)/claude-code-webui(SDK NDJSON)/opcode(spawn CLI 解析 JSONL)/Claude Code on the web(云沙箱)，无一内嵌终端【高｜官方文档+源码】
8. PTY+xterm.js（ttyd/wetty/gotty+creack/pty，SIGWINCH/尺寸协议成熟）可行，但移动端硬伤：软键盘无 Esc/Ctrl、IME 乱序、遮挡视口；仅社区小项目走此路，宜作兜底通道【高｜官方仓库+issue】
9. 文件级双向同步遇并发写皆留冲突：git 两端并发 commit→后 push 者被拒须手工 rebase；Syncthing 留 .sync-conflict 副本；rclone bisync 默认无胜者。.rick/jobs/*/doing/tasks.json 双端并发写无自动解法【高｜官方文档】
10. 先例共识：静态规则走 git（cursor rules），运行态走结构化/append-only 存储（Claude Agent SDK SessionStore 追加分块 part-*.jsonl 天然免冲突；continue=Hub 云端控制面）【高｜官方文档+SDK 源码】
11. Go embed SPA 有「预构建 dist 提交仓库」先例（AdguardHome/kubeaquarium embed 提交产物、Vaultwarden 独立仓库 bw_web_builds、Syncthing vanilla JS embed）——node 不进 rick 构建链可守【高｜项目仓库】
12. 传输选型：标准库 SSE（http.Flusher）+POST 命令=LLM 流式事实标准，代理缓冲坑可解（FlushInterval=-1/X-Accel-Buffering:no）；确需 WS 时选 coder/websocket（活跃）或 gorilla（2023 复活）【高｜Go 官方 issue+仓库】

## R7 上报项

1. SDK 多 session 并发无官方先例/文档——需原型验证（缺官方示例）
2. 每 job 一个 pi rpc 子进程的常驻内存开销未实测（缺 RSS 数据）
3. .rick 运行态改 append-only/单写点是设计决策非事实裁决——交判断节点
4. Claude Code on the web 多设备 session 同步机制细节未公开

## 对 L1 判断节点的输入：架构选项可行性矩阵

| 选项 | 多 job 常驻 | 事件流完备度 | rick 哲学契合 | 复杂度 | 判定 |
|---|---|---|---|---|---|
| A pi extension 内嵌 web server | ✗ 单 session 生命周期(结论5) | 中（单会话） | 中 | 低 | 仅单会话 debug 视图，非主架构 |
| B 独立 Node 常驻服务(SDK) | ✓ | 高(结论1-3) | 低-中：多一常驻 Node 服务 | 中 | 技术最优但引入第二运行时 |
| C Go 服务+pi rpc/json 子进程 | ✓ 每 job 一子进程 | 高(结论4,6) | 高：node 仅用户依赖=pi 本体 | 中 | 主候选 |
| D 混合 Go 主+Node SDK 子服务 | ✓ | 最高 | 中 | 高：两服务生命周期+IPC | C 不足时的升级路径 |

矩阵补充：B/C/D 共用同一前端与同步层（结论 9-12）；C 直接复用 pi 断线重连/历史回放（结论 6）；rick-spec 已预留 WEB-UI 为第一层入口且允许跨层直连 pi。

## 产出文件

- 详录 A：research-L1-A-detail.md（pi 一手细节+信源）
- 叶子 B：research-L1-leaf-B.md（竞品+PTY 坑+链接）
- 叶子 C：research-L1-leaf-C.md（同步冲突机制+先例+链接）
- 叶子 D：research-L1-leaf-D.md（Go 前端栈+WS/SSE+链接）
