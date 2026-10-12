# 依赖关系
task16

# 写域
internal/web/registry.go
internal/web/sessions.go
internal/web/recovery.go
internal/web/routes.go
internal/handler/doing.go
internal/web/recovery_test.go
internal/web/sessions_test.go
internal/handler/doing_test.go

# 任务目标
把「平台升级重启」的语义从含混的 `error` 升级为**明确的挂起（suspend）**，并提供**人工一键恢复**入口与恢复报告——**绝不自动恢复/自动续跑**（human 裁决 J-L6-6/J-L6-7：避免重复副作用；调研实测 pi 不修复悬挂 toolCall、配额耗尽不报错，自动续跑风险不可接受）。

# 关键结果
1. `internal/web/registry.go`：新增状态常量 `SessionStatusSuspended = "suspended"`（注释写清语义：进程不在、状态完整、可一键恢复；与 `error`=执行失败、`closed`=正常结束区分）。
2. `internal/web/recovery.go`（新）：
   - `type SuspendRecord struct { SessionID, Type, PISessionID, WorkspaceID, JobID, LastEntryID string; Busy bool; At time.Time }`
   - 关停时（`Server` graceful 路径 / `ShutdownWorkers` 之前）把「当前在跑（active/running）」的会话写入 `<stateDir>/suspend.json`（原子 tmp+rename），并把注册表状态改为 `suspended`（**写盘**，不是只发事件）；`reason="platform upgrade (release)"` 或 `"server restart"`。
   - `ReconcileOnStart` 改造：active/running 且无 worker → 标 `suspended`（而不是 `error`），reason 落盘（新增 `SessionEntry.LastReason` 字段，`json:"last_reason,omitempty"`，向后兼容）；已 `suspended` 的行保持不动。
   - `WriteRecoveryReport(stateDir string, items []RecoveryItem) error` + `ReadRecoveryReport(stateDir string) (Report, error)`：`<stateDir>/recovery-report.json`，`Report{at, suspended[], recovered[], failed[]}`。
3. `internal/web/sessions.go`：
   - `SessionResume`：允许 `suspended`（与 error/closed 同路径 spawn `--session <pi_id>`）；成功后写回 report（recovered）。
   - **doing/dream 的可恢复化**（原 `doing` 直接 409）：`resume` 对 doing 走「续跑」语义 → 归一化 `<ws>/.rick/jobs/<job>/doing/tasks.json` 中遗留 `running → pending`（调研 F8：门禁把遗留 running 判为 zombie 而失败），再复用既有 `DoingIn` 重新执行剩余 task；dream 后台同理（记录：不自动、仅人工触发）。
   - `SessionContinue`（新 handler，`POST /api/sessions/{id}/continue`）：语义 = 「我确认继续」——对 `suspended` 的交互型会话等价 resume；对 doing/dream 执行上面的归一化 + 续跑；对 active 返回 202 幂等。
4. `internal/web/routes.go`：挂 `POST /api/sessions/{id}/continue` 与 `GET /api/recovery`（读恢复报告，供前端展示）。
5. `internal/handler/doing.go`：暴露 `NormalizeRunningTasks(rickDir, jobID) (moved []string, err error)`（幂等；只动 `running`，不动 success/error/pending）。
6. 测试：
   - `internal/web/recovery_test.go`：suspend 写入/读取；ReconcileOnStart 把 active→suspended（**不是 error**）；报告读写。
   - `internal/web/sessions_test.go`：`/continue` 对 suspended 交互会话 → 202/200 且状态回 active（fake supervisor）；doing 归一化后剩余 task 被续跑（fake DoingRunner 断言收到的 job 与「running 已变 pending」）；**反向断言：启动过程不自动 spawn 任何 worker**（自动恢复必须不存在）。
   - `internal/handler/doing_test.go`：`NormalizeRunningTasks` 幂等与作用域。

# 测试方法
go test ./internal/web/ ./internal/handler/ -timeout 600s
go build ./... && go vet ./internal/web/ ./internal/handler/
python3 .rick/jobs/job_36/plan/gates/gate8.py

# 上下文提示
- 调研依据：research-L6 §3（intent 持久化方案 / jsonl 形状判定 / 续跑策略对比 —— **本 task 采信其「不自动续跑」结论**）、F8（doing 归一化）、F3（reason 未落盘）；research-L5 §6（dev 实例重启后同样走挂起+人工）。
- 幂等与安全：`/continue` 重复点击不得重复起 worker（复用既有 dup 检查）；归一化必须原子（tmp+rename），且**不得**把 `success` 改回 `pending`。
- 前端类型 `SessionStatus` 新增 `suspended` 由 task20 同步（并行层，契约先定：状态字符串 `suspended`）。
