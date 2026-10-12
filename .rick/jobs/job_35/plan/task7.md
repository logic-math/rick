# 依赖关系
task6

# 任务名称
完成 handler 覆盖 human-loop/ctrl/dream/learning 并让 cli 全量变薄

# 任务目标
按 spec（task2）完成 KR2.1：handler 调度聚合层覆盖剩余命令 human-loop、ctrl、dream、learning，`internal/cmd` 全部命令变薄为「路由 + 参数解析 + 调 handler」。所有命令统一走「env.Ensure → builder.Build（产 method+instance 两份）→ 调用 runtime」编排；交互命令保留 `CallCLI`（行为不变，method 经 `--append-system-prompt` 传给 `CallCLI`），`runtime.Run` 结构化签名仅 task8 的 doing 使用。

## 逐命令改动明细（调研结论）

### human-loop（不依赖 executor/parser/git/actpath/logging，迁移无编译断裂）
- cli 保留 `NewHumanLoopCmd`（topic 校验：空则报「topic is required」）；迁出 RunE 内编排 → `handler.HumanLoop`：`GetDraftDir` → `NextLoopID` → 建目录 → `builder.BuildHumanLoop`（产 method+instance）→ `env.Ensure` → `runtime.CallCLI`（交互，注入 method）→ 持久化 sessionID
- flags：无自有 flag；全局 `--dry-run`/`--job`/`-v`
- **task5 包迁移影响**：`human_loop_prompt.go` 依赖 prompt 包的 `WriteSkillFile`/`LoadCoreSkills`（读 skillsFS embed），prompt→builder 迁移后改经 builder 导出函数（`ReadEmbeddedSkill`）；task9 注册 think/research/exporter agent 后，`{{think_skill_path}}` 等 skill 路径变量改由 pi skills 机制承载

### ctrl（不直接 import executor/parser/git/logging/agent 接口/actpath，仅 piagent→runtime）
- cli 保留 `NewCtrlCmd`（`--job` 必传校验 + 调 handler）；迁出 `runCtrl`/`runCtrlDryRun` → `handler.Ctrl`/`handler.CtrlDryRun`
- flags：无自有 flag；全局 `--dry-run`/`--job`（代码强制必传）/`-v`
- **ctrl.md 模板 stale（重要）**：当前描述 claude code 的 NDJSON 格式（`type=system/assistant/user/result`），但 runtime 已用 pi JSONL（`session/agent_settled/message_end/tool_execution_start/tool_execution_end`，camelCase）——本 task 改写 ctrl.md 为 pi JSONL 语义；`act-path.md` 引用改为 runtime trace（task8 一并）
- builder：`GenerateCtrlPromptFile` 拆为 `BuildCtrl` → (method=ctrl.md 角色定义, instance=job_id/doing_dir/plan_dir/tasks_json 路径)，**注入路径而非内容**（去掉 tasks_json_content 快照，pi 自行 read）

### dream（唯一 executor 依赖 = LoadTasksJSON，line 169）
- cli 保留 `NewDreamCmd`（`-p/--background`、`--job_num int`(默认5)）；迁出 `dreamWorkflow`→`handler.Dream`、`runDreamDryRun`→`handler.DreamDryRun`、`selectPendingJobs`/`getDreamProcessedJobs`/`discoverCompletedJobs`/`jobNumber`（4 个确定性扫描过滤函数）
- **扫描过滤留在 Go（确定性输入过滤，决定哪些 job 要 dream）**：`discoverCompletedJobs` 依赖的 `executor.LoadTasksJSON` + `TasksJSON.GetAllTasks()` + `TaskState.Status` 迁 `workspace`（极薄读取器，task8 显式落地）
- `dream.md` Step 2 读 `act-path.md`、`dream_prompt.go` 的 `loadActPaths` 扫 `act-path.md` → task8 改扫 `trace.md`；Step 7 的 subagent_1~4 触发词 → task11 改 workflowScript+runs.run；Step 8 `tools dream_check` 引用保留（dream_check 存活）

### learning（依赖 executor + parser + actpath，task8 一并闭环）
- cli 保留 `NewLearningCmd`（`--job` flag）；迁出 `executeLearningWorkflow`→`handler.Learning`、`runLearningDryRun`→`handler.LearningDryRun`、`collectExecutionData`、`callAgentForAnalysis`、`buildLearningPrompt`（当前内联在 cmd 内，无独立 prompt/*.go → 迁 builder 的 `BuildLearning`）
- 依赖闭环（task8）：`executor.LoadTasksJSON`/`TasksJSON`/`TaskState` + `executor.LoadDebugContext` + `parser.ExtractBugFrontmatter` → 迁 workspace/prompt 极薄实现；`ActPathFiles` glob `act-path.md` → 改 glob `trace.md`
- 模板 `learning.md` 完成要求 `rick tools learning_check {{job_id}}`（learning_check 存活，task8 明确）

参考：domain/commands.md「human-loop」「ctrl」「dream」；skill `command_registration_verification_skill`、`verify_go_changes_skill`、`global_ref_sync_skill`。

# 关键结果
1. `internal/handler` 新增 `HumanLoop`/`Ctrl`/`Dream`/`Learning` 编排函数，从对应 cmd 文件迁移；`internal/cmd/{human_loop,ctrl,dream,learning}.go` 变薄
2. 全部 8 命令 flag 行为不变；随函数迁移同步改写 `human_loop_test.go`/`learning_test.go` 等引用已迁函数的断言；另改写 `tools_test.go`（`collectExecutionData` 引用）与 `dryrun_integration_test.go`（`runLearningDryRun` 引用）；handler 以 `Options{Verbose, DryRun, JobID}` 参数接收 flag（不 import cmd）
3. **ctrl.md 模板改写为 pi JSONL 语义**（去 claude code NDJSON 描述），act-path 引用改 runtime trace 占位
4. **dream 的 4 个扫描函数**（`selectPendingJobs`/`getDreamProcessedJobs`/`discoverCompletedJobs`/`jobNumber`）迁 workspace（本 task 迁编排，tasks.json 极薄读取器类型定义在 task8 落地）
5. `go build` + `go test ./internal/cmd/... ./internal/handler/... -timeout 60s -v` 全绿；`human-loop --dry-run '测试'` 含 `sense_loop`；`ctrl`（无 --job）报错退出

# 测试方法
（本 task 测试用例设计遵循 skill:tdd + skill:testing-anti-patterns：先写失败测试；测试真实 CLI 行为；dry-run 断言先定位 section。）

1. 正常路径：前置条件 = task6 完成；输入 = 无；操作 = `go build -o bin/rick ./cmd/rick && go test ./internal/cmd/... ./internal/handler/... -timeout 60s -v`；预期 = build 成功，测试全绿。
2. 边界（human-loop dry-run + dream 扫描过滤 + ctrl --job 缺失）：前置条件 = build 成功 + 存在「全 success」与「未完成」的 job；输入 = `human-loop --dry-run '测试主题'`、`dream --dry-run`、`ctrl`（无 --job）；操作 = 依次运行；预期 = human-loop 输出含 `sense_loop`；dream 只列「完成且未 dream」的 job（排除未完成/已 dream，按 job 号升序截断）；ctrl 报 `--job flag is required` exit 非 0。
3. 异常（learning 缺数据 + ctrl doing 目录不存在）：前置条件 = build 成功；输入 = `learning job_N`（doing/tasks.json 不存在）、`ctrl --dry-run --job job_N`（doing 目录不存在）；操作 = 运行；预期 = learning 报 `tasks.json not found` exit 非 0 不 panic；ctrl 报 `doing directory not found` exit 非 0。
