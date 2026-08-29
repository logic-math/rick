# rick web E2E 验证记录（job_36 / task15 / 2026-08-30）

> 手册化全链路验证记录（与 gate6 脚本化 E2E 互补）。环境：本机 rick 4.4.15、`go build -o bin/rick ./cmd/rick` 新构建、隔离 HOME（/tmp/rick-e2e-task15/home-\*）、`--port 16376 --token e2etest15`、curl `--noproxy '*'`（本机 http_proxy 会拦截 127.0.0.1）。

## 服务启动

```
$ ./bin/rick web --port 16376 --listen 127.0.0.1 --token e2etest15
[rick-web] listening on http://127.0.0.1:16376
[rick-web] auth: token required (Authorization: Bearer or ?token=)
[rick-web] workspaces: 0 registered
```

启动横幅：监听地址 + 认证提示 + 工作区计数 ✓

## 验证清单（10 项）

| # | 项目 | 结果 | 证据 |
|---|---|---|---|
| ① | GET / 静态首页 | ✅ 200 | `<!doctype html>` + `<div id="root">` + hashed assets 引用 `index-CBhlcu38.js` |
| ② | GET /api/config 认证矩阵 | ✅ | 无 token → 401；Bearer token → 200 `{"version":1,"rick_version":"4.4.15","port":"16376","auth_required":true}` |
| ③ | POST /api/workspaces 注册 | ✅ 201 | `{"id":"a8ceb938","path":"/workdir/.../rick","name":"rick","added_at":"...+08:00"}`（RFC3339 带时区） |
| ④ | GET /api/workspaces 列表 | ✅ 200 | 含注册条目 + `jobs_count:26`（ListJobs join 生效） |
| ⑤ | GET /api/sessions?workspace= | ✅ 200 | `[]`（新建服务无会话，符合预期） |
| ⑥ | SSE /api/events | ⚠️ **部分** | HTTP 200 + `Content-Type: text/event-stream` + `Cache-Control: no-cache` + `X-Accel-Buffering: no` 全部正确；**但连接建立后未发出初始 server_info 事件**（详见下方「发现的缺陷」） |
| ⑦ | GET /api/workspaces/{id}/jobs | ✅ 200 | 26 个 job 真实数据（job_36 含 task1..task14 状态/commit_hash） |
| ⑧ | knowledge tree / file | ✅ 200 | tree 47 文件（domain/loops/skills）；`domain/bugs.md` 内容正常返回 |
| ⑨ | Ctrl+C 优雅退出 + pid 清理 | ✅ | SIGINT → 进程退出 ✓ + `~/.rick/web.pid` 删除 ✓ + 端口释放 ✓（注：kill 需打到真实 rick pid——env 包装进程会吸收信号，测试方法注意） |
| ⑩ | 二次启动 + 状态持久化 | ✅ | 第二次启动 `workspaces: 1 registered`（web.json 持久化 ✓）；health 200；再退出 pid 清理 ✓ |

补充验证：

- **残留进程清理**：发现 L4/L5 worker 的 smoke 测试残留进程（`bin/rick web --port 16390 --token smoke-token`，孤儿）已清理；机器无 `~/.rick/web.json` 污染（该 worker 用了隔离 HOME）。
- **受影响包全回归**：`go test ./internal/cmd/ ./internal/env/ ./internal/web/... ./internal/runtime/ ./internal/handler/...` 全绿。
- **embed 终检**：SrcFS 契约锁定测试（含 public/ PWA 图标）通过；dist 新鲜构建（源码 L3/L4 改动后重建）。

## 发现的缺陷（1 项，需 parent 裁决修复）

**SSE 初始 server_info 缺失**（L4 routes.go 遗留）：

- 现象：连接 `/api/events` 后 3s 内无任何 `data:` 事件（心跳 15s 周期也未到）；HTTP 层全部正确。
- 根因：`internal/web/routes.go` 的 `handleEvents()` 调 `ServeSSE(w, r, deps.Hub, ServeSSEOptions{})`——`ServeSSE` 从不在连接建立时发布初始 `server_info`；`ServerInfoEvent()` 构造器存在但只在测试里手动 Publish 过。api-contract 承诺「server_info 连接建立即发」。
- 影响：① gate6 的 SSE 断言（3s 内 server_info）会红——**L6 层检查点将被阻塞**；② 前端 `api/sse.ts` 以 server_info 作为重连后全量重建信号（`rick-web:sse-resync` 事件），首连无此信号（首连靠 REST 拉全量，影响有限，但契约违约）。
- 建议修复（一行级，写域在 internal/web/routes.go——task13 所有者）：`handleEvents` 在 `ServeSSE` 前往 hub 或订阅写入一个初始 `ServerInfoEvent(1, deps.Version)`；最小改法是在 `ServeSSEOptions` 加 `InitialEvent *Envelope`，`ServeSSE` 在写头后先发它。

## 真实 pi 会话冒烟（human 手动指引）

go test 已用 fake pi 覆盖会话创建/命令/entries 全链路；真实 LLM 会话由 human 手动验证（参考）：

```
1. rick web 启动后浏览器进入
2. 添加工作区（任一含 .rick 的项目）
3. 新建 human-loop 会话（topic 随意）→ 观察聊天窗事件流与 extension_ui 弹窗
4. 新建 doing 会话（选一个已有 plan 的 job）→ 观察监控看板
5. 关闭会话 → 离线浏览 → Resume
```
