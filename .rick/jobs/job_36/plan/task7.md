# 依赖关系
task3

# 写域
web/src/api/
web/src/stores/
web/src/types.ts

# 任务目标
前端 API 客户端 + SSE 客户端 + Zustand stores + 全量类型定义（按 api-contract.md 单源）。

# 关键结果
1. `web/src/types.ts`：api-contract.md 全量 TS 类型（WorkspaceEntry/SessionInfo/JobSummary/TaskBrief/SessionType 联合类型+各 type 的 Params 接口/SSE Envelope 与各 event data 类型/WebError）
2. `web/src/api/client.ts`：`class ApiClient`——fetch 封装（baseUrl 默认同源；token 从 localStorage(`rick-web-token`) 注入 Bearer；统一错误解包 WebError→throw ApiError{code,message,status}；401 时触发全局未授权事件→store 置 authRequired）；方法面=契约全集（getWorkspaces/addWorkspace/removeWorkspace/listSessions/createSession/getSession/prompt/steer/abort/closeSession/resumeSession/uiResponse/getEntries/listJobs/getTasks/getJobFile/knowledgeTree/knowledgeFile/getConfig）
3. `web/src/api/sse.ts`：`class SseClient`——EventSource(`/api/events?token=`)；按 envelope.type 分发到 handlers map；**断线重连**（onerror 指数退避 1s→30s 封顶；重连成功后重放期间 UI 乐观保留 + 收到 server_info 后触发全量刷新事件）；Last-Event-ID 由 EventSource 自动携带（服务端 task8 支持）；收到 `frontend_reload` → location.reload()
4. `web/src/stores/`（Zustand）：`workspaces.ts`（列表+增删+loading 态）、`sessions.ts`（按 workspace 分组的会话表+状态同步）、`events.ts`（**流式事件批处理**：session_event 入 ring buffer（每 session 保留最近 500 条）+ rAF/50ms 批量 flush 通知订阅组件——OpenHands 防卡顿模式）、`jobs.ts`（jobs 快照+jobs_update diff 应用）、`ui.ts`（authRequired/全局错误/连接状态）
5. `npm run build` + `npx tsc --noEmit` 通过
6. 未接线页面允许占位（stores 自测用 vitest 可选——不强制；以 tsc 类型自洽为准）

# 测试方法
cd web && npx tsc --noEmit && npm run build（类型+构建全绿）；mock EventSource 手动 smoke（console 驱动）可选

# 上下文提示
- 先读 /workdir/sunquan20/AI_CODING/rick/.rick/jobs/job_36/plan/api-contract.md——types.ts 与其逐字段对齐，不得臆造
- token 首次输入界面在 task12（Settings 页）；本 task 的 authRequired 事件是它的信号源
- Zustand 无 immer 依赖手写更新；SSE envelope.seq 单调——events store 按 seq 去重
