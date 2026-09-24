# research-L1 叶子B：rick Web UI 功能面现状（job_43）

> 范围：internal/web/*.go（忽略 _test）+ web/src/（67 文件）。信源=代码原文（最高置信）。

## 1. internal/web 文件职责

- `web.go`：包总纲——机器级单例状态（工作区/会话注册表+SSE 总线+API+静态资源）+ BuildID
- `server.go`：http.Server 生命周期（NewServer→Serve，10s 优雅关闭）
- `routes.go`(969L)：全部 HTTP 路由（api-contract.md 单源，Go1.22 mux + authWrap）
- `sessions.go`(2313L)：会话执行心脏——pi rpc 子进程 Supervisor + 编排 + SSE 汇合
- `jobs.go`(609L)：job 列表（归档过滤/dream 自动归档）+ tasks + job 文件树/读 + knowledge 树/读
- `registry.go`(506L)：工作区注册表；`recovery.go`(361L)：suspend.json+recovery-report.json（禁自动恢复）
- `sse.go`(432L)：SSE Hub（连接即发 server_info）；`auth.go`：TokenAuth（Bearer/?token=）
- `static.go`(241L)：静态资源（~/.rick/web/dist 覆盖层优先，SPA fallback）
- `watcher.go`(297L)：轮询 watcher（dist 变更→frontend_reload；tasks.json 变更→jobs_update）
- `statedir.go`：~/.rick web 状态目录；`jobnames.go`：job 别名；`archived.go`：软归档（manual/dream/done）
- `rel-state-004/`：**空目录**（遗留）

## 2. API 路由面（routes.go）

health（免认证）+ config；workspaces：list/add/browse/create/delete/order + fs：list/status/mkdir；sessions 20 个：CRUD/import/prompt/steer/abort/close/archive/unarchive/resume/**continue**/ui_response/model/thinking/compact/entries/models/prompt + /api/recovery；jobs：list(include_archived)/rename/archive/unarchive/tasks/file/files；knowledge：tree/file；/workspaces/{ws}/file（白名单硬化读）；web customize/reset；SSE /api/events；SPA fallback

## 3. 前端功能入口

路由 `/ws/:wsId/{sessions|jobs|dreams}`、`/session/:id`、`/settings`。**NewSessionModal 支持 7 种会话类型**：plan/easy/doing/ctrl/human-loop/learning/dream（dream 带 job_num+mode；NewSessionModal.tsx:50-113）→ CLI 全命令面已入 web。监控：TaskBoard+GateResult+EventStream；聊天：ChatView（steer/模型切换/thinking/compact）；Jobs 页：文件浏览含 doing 下 act-path.md 与 raw_session_coding.log（JobFiles.tsx:8）

## 4. 能力确认（有证据）

- 多 workspace：registry.go+WorkspaceTree.tsx ✅
- 多设备：SSE+会话跑在 Go server 进程内（App.tsx ConnectionBanner 注释）✅
- 挂起恢复：recovery.go+sessions.go:1033 SessionContinue（仅人工）+RecoveryBanner ✅
- daemon：internal/cmd/web.go:61-107（setsid 自 exec）✅
- 自迭代生效：watcher.go dist 变更 500ms→reload ✅

## 5. 候选缺口

- **B1 dream 日志无浏览接口**：Dreams.tsx:6,28 前端占位「待后端接口支持」；KnowledgeTree 只走 domain/loops/skills（jobs.go:404-407）。小
- **B2 learning 产物不在 job 文件白名单**：handleJobFileList 只走 plan/doing/grilling/prompts（jobs.go:385 whitelist=plan/**+doing/**）→ learning/SUMMARY.md 不可浏览。小
- **B3 rick tools 类操作无 web 入口**（checks/release/dev-web/theme CLI only）。中
- **B4 draft/rfc 产物不可发现**（KnowledgeRoots 不含 draft，仅可 /file 直读）。小
- **B5 空目录 rel-state-004**。极小

## 结论（3 条）

- web 已覆盖全部 7 种会话命令+多工作区/多设备/恢复/daemon，功能面完整度高
- 最大缺口：dream 日志浏览（前端已留占位）与 learning 产物（白名单遗漏）
- tools 类管理操作与 draft 发现是次要缺口
