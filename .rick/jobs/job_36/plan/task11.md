# 依赖关系
task4, task2, task8

# 写域
internal/web/sessions.go
internal/web/sessions_test.go

# 任务目标
会话 REST 层：创建/命令/生命周期/离线回放 + pi rpc supervisor 接线 + doing/dream 后台执行桥（本 task 产出 http.HandlerFunc，路由挂载在 task13）。

# 关键结果
1. `internal/web/sessions.go`：
   - `type SessionManager struct`（依赖注入：SessionRegistry/WorkspaceRegistry/Supervisor/Hub/builder 所需 cfg）+ `NewSessionManager(...)`
   - 创建会话 `CreateSession(w, r)`：ValidateSessionRequest（task5）→ 按 type 分派：
     - **plan**：`planCreate(ws)`——workspace 路径下产 prompt 文件（NextJobIDIn + `prompt.GeneratePlanPromptFile`——若 task2 导出的 prepare 粒度够就复用；注意：PlanIn 会启动交互 CLI，web 不能用！web 只消费「只产出 prompt 文件」的部分）→ SpawnSpec{Dir: wsPath, MethodFile, PromptFile, CreateNew: true, SessionIDFlag: uuid} → supervisor.Spawn → 发 bootstrap prompt（`runtime.BootstrapMessage`——task4 导出）→ SessionEntry{PISessionID: uuid, Status: active} 落盘 + Hub 发 session_state
     - **easy/ctrl/human-loop/learning**：同款——easy=`prompt.GenerateEasyPromptFile`+uuid；ctrl=`prompt.GenerateCtrlPromptFile`（已有 job 校验）；human-loop=humanLoopCore 的 prompt 产出部分；learning=collectExecutionData+buildLearningPrompt 部分（注意：builder 真实函数名以 internal/prompt/*_prompt.go 导出面为准——SavePlanPrompt 实为 `prompt.GeneratePlanPromptFile`，dream 为 `prompt.GenerateDreamPromptFile`）
     - **dream(interactive)**：dream prompt 产出 + spawn；**dream(background)**：不 spawn——goroutine 跑 dreamCore(ctx, rickDir, jobNum, progress→Hub) → Status: running→closed
     - **doing**：goroutine 跑 doingCore(ctx, rickDir, job, opts, progress→Hub)（task2 已加 ctx/progress 形态）→ running→closed/error
   - 命令面：`SessionPrompt/SessionSteer/SessionAbort/SessionClose/SessionResume/SessionUIResponse`（worker stdin 写 RpcClient 命令；非 active 会话 409；Resume=closed→SpawnSpec{CreateNew: false, SessionIDFlag: pi_session_id}）+ `SessionClose` 幂等（已 closed 仍 202）
   - `SessionEntries(w, r)`：active→rpc GetEntries(since) 透传；closed→离线读 pi session JSONL（`~/.rick/pi/agent/sessions/**/<pi_session_id>.jsonl` 定位：按目录枚举匹配文件名前缀；解析为 entries 数组——session-format v3 行结构，读本 task 内实现轻量解析：每行 json，取 id/parentId/message）；解析结果 {entries, leaf_id}
   - extension_ui 事件回程：Supervisor worker Events() 的 extension_ui_request → Hub session_event（前端弹窗）→ SessionUIResponse 写回
   - 事件泵：每 worker 起 goroutine 把 Events() 全量转 Hub.Publish（session_event）+ session_state 变更（agent_settled→idle 提示）+ LastEntryID 更新
2. `internal/web/sessions_test.go`：fake pi（同 task4 模式）+ httptest：创建 plan 会话（fake pi 收到 --mode rpc 进程 + bootstrap prompt stdin 断言）；prompt/steer/abort/close/resume 命令路由；closed 会话 entries 离线解析（造 fixture jsonl）；doing 后台型（fake doingCore？——doingCore 依赖重，用注入函数接口 `DoingRunner func(ctx, rickDir, job, progress) error` 使 SessionManager 可测：fake runner 发 progress 事件→断言 Hub 收到 jobs_update）；409 矩阵
3. `go build ./...` + `go test ./internal/web/ -run TestSession -timeout 180s` 通过
4. ⚠️ 测试隔离：RICK_PI_AGENT_DIR/HOME 双隔离（supervisor spawn fake pi + registry 落 tmp）

# 测试方法
go test ./internal/web/ -run TestSession -v 全绿；ps 无残留 fake pi 进程

# 上下文提示
- 本 task 是后端心脏：Supervisor（task4）+ handler core（task2）+ Hub（task8）三线汇合；写域只有 sessions*.go——若 task2 导出粒度不够（无法只产 prompt 不启动 CLI），**不要改 handler/**（写域越界）——改为 import internal/builder 直接产出（builder 本就路径参数化）+ 在 task 回执中记录该缺口（task13 前统一收口）
- ⚠️ pi 会话文件定位：sessions/<encoded-workdir>/<timestamp>_<uuid>.jsonl——按 uuid 文件名前缀枚举（不可假设 workdir 编码规则稳定，直接全 sessions 树 find）
- dream interactive 与 background 的 prompt 产出复用 dreamCore 的前置（builder.SaveDreamPrompt 若有——自查 internal/prompt/dream_prompt.go）
