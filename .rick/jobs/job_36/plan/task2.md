# 依赖关系
无

# 写域
internal/handler/
internal/cmd/doing.go
internal/cmd/easy.go
internal/cmd/plan.go
internal/cmd/ctrl.go
internal/cmd/human_loop.go
internal/cmd/learning.go
internal/cmd/dream.go
internal/runtime/runtime.go

# 任务目标
handler 层「路径参数化 core 重构」：把 7 个 handler 从「cwd 隐式解析 rickDir」改为「显式路径参数 core + CLI 包装」，为 web 多工作区注入铺路。**CLI 行为零变化**。

# 关键结果
1. internal/handler/ 内 7 个编排函数抽 core：`planCore(rickDir string, ...)`、`easyCore(rickDir, ...)`、`ctrlCore(rickDir, jobID, ...)`、`humanLoopCore(rickDir, ...)`、`learningCore(rickDir, ...)`、`dreamCore(rickDir, ...)`、`doingCore(rickDir, jobID, opts, ...)`——core 接收 rickDir（及各 cmd 特有参数），内部不再调用 workspace.GetRickDir()
2. 对外导出两种入口：现有 `Plan(...)`/`Easy(...)` 等（cwd 解析后调 core，签名不变，CLI 调用点零改动）+ 新导出 `PlanIn(rickDir, ...)` 等 web 用的显式版本（后续 task11 消费）
3. doing/dream 增加可取消与进度回调形态：`doingCore` 接受 `ctx context.Context` 与 `progress func(DoingEvent)`（task 状态 diff 事件：{job_id, task_id, from, to}；watchTasksJSON 已有 watcher 逻辑改造为回调注入）——CLI 入口传 context.Background() 与 stderr 打印回调（行为不变）
4. runtime.Runtime 接口加目录参数变体：`RunIn(dir string, methodText, promptFile string, cfg) (...)`（默认 Run 委托 RunIn("", ...) 即当前 cwd 行为）；piRuntime 实现里 exec.Cmd 设 cmd.Dir=dir（空=现状）
5. core 内路径派生硬约束：一律 `filepath.Join(rickDir, ...)`；workspace.NextJobID/GetJobPlanDir/GetDraftDir/GetJobsDir 等 cwd 依赖函数**禁入 core**——在 handler 内自实现 `nextJobIDIn(rickDir)`（扫 `<rickDir>/jobs` 取最大 job_N+1）；SelectPendingJobs(rickDir)/LoadTasksJSON(path)/NextLoopID(draftDir) 已路径参数化可直接复用
6. 全量回归：`go build ./...` + `go test ./internal/handler/... -timeout 120s` + `go test ./internal/cmd/... -timeout 60s` 全绿（现有测试是行为零变化的保证）

# 测试方法
go test ./internal/handler/... ./internal/cmd/... -timeout 180s -v（现有测试全绿 = 重构安全）；临时对照：dry-run 输出与重构前一致（plan/easy/learning/ctrl/dream 都有 --dry-run，可 diff）

# 上下文提示
- GetRickDir() = cwd + ".rick"（paths.go:48，不向上回溯），handler 内 16 处调用全部要消除（core 化）；另注意 workspace.NextJobID/GetJobPlanDir/GetDraftDir/GetJobsDir 也是 cwd 依赖（core 内必须自实现路径版）
- 这是机械重构：移动代码 + 参数化，不改逻辑；builder 已路径参数化无需动
- ⚠️ 保持 DIP 组合根模式：doing.go（cmd 层）是唯一 import runtime 具体实现的地方，别破坏
