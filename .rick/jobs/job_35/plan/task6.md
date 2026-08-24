# 依赖关系
task3, task4, task5

# 任务名称
落地 handler 调度聚合层并让 cli 变薄（plan/doing/easy）

# 任务目标
按 spec（task2）落地 KR2.1 第一部分：新建 `internal/handler` 包作为调度聚合层，编排 env→builder→runtime 并持久化 sessionID。将 plan/doing/easy 三大命令的编排逻辑从 `internal/cmd` 迁移到 handler，cli 变薄为「Cobra 命令 + flag 解析 + 调 handler」。

handler 编排模式（目标态，task8 才完全落地）：`env.Ensure()` → `builder.Build()`（产 method+instance 两份）→ 调用 runtime → 持久化 sessionID 到 job 目录（`session_id`）。本 task 中：doing 仍经 executor 执行（不改调度），plan/easy 仍走 `runtime.CallCLI`（交互 TUI，method 经 `--append-system-prompt` 注入）；`runtime.Run` 结构化签名在 task8 切换 doing 时才启用。handler 函数签名以参数接收 flag 值（如 `Options{Verbose, DryRun, JobID}`），**不得 import `internal/cmd` 的 `GetVerbose`/`GetDryRun`/`GetJobID`**（跨包循环依赖），cmd 在调用点透传。

## 逐命令改动明细（调研结论）

### plan（干净命令，不依赖 executor/parser/git/actpath/logging，迁移无编译断裂）
- cli 保留 `NewPlanCmd`（Use/Short/Args + RunE 解析 `GetJobID`/`GetDryRun`/args）；迁出 `executePlanWorkflow`→`handler.Plan`、`reEnterPlanWorkflow`→`handler.ReEnterPlan`、`runPlanDryRun`→`handler.PlanDryRun`
- **删除死代码 `generateJobID()`**（`time.Now().Unix()` 生成，但主流程用 `workspace.NextJobID()`，从未被调用）；`promptForRequirement` 与 easy.go 共享（cmd 包内）→ 迁 handler 或在 cmd 保留共用（否则 easy 侧编译断裂）
- flags：无自有 flag；全局 `--dry-run`/`--job`/`-v`；`--job` 触发重入分支（先校验 plan 目录存在）

### doing（executor 最大消费者，删包同批在 task8）
- cli 保留 `NewDoingCmd`（`--job`/`--easy`/`--ctx` flags + RunE 路由：`--easy`→handler.Easy、`--dry-run`→handler.DoingDryRun、否则 handler.Doing）；迁出 `executeDoingWorkflow`→`handler.Doing`、`runDoingDryRun`→`handler.DoingDryRun`、`loadTasksFromPlan`/`sortTaskFiles`/`extractTaskNumber`（task8 删，pi 直读）、`printExecutionSummary`、`commitDoingResults`/`ensureGitUserConfigured`/`ensureGitInitialized`（task8 下沉 pi 脚本）
- 本 task 只搬编排到 handler（doing 仍经 executor 执行，不改调度），task8 才删包 + 改 `runtime.Run` + workflowScript 编排

### easy（easy.go 无 parser import，仅 easy_prompt.go 的 loadDebugContextLocal 依赖 parser.ExtractBugFrontmatter）
- cli 保留 `NewEasyCmd`（`--ctx`/`-r/--requirement`/`--resume` flags）；迁出 `runEasyMode`→`handler.Easy`、`resumeEasyMode`→`handler.ResumeEasy`、`startEasySession`→`handler.StartEasySession`、`runEasyDryRun`→`handler.EasyDryRun`、`validateCtxInheritance`（--ctx 防覆盖）、`saveSessionID`/`loadSessionID`、`writeEasyTasksJSON`（map 内联，无 executor 依赖，保留「easy_session success」供 dream 发现）、`generateUUID`
- `loadDebugContextLocal` 的 `parser.ExtractBugFrontmatter` 依赖：三层注入后改「注入 `doing/debug/` 路径、pi 自行 read」消除 parser 依赖（task8 一并解耦）

参考：domain/architecture.md「模块划分」；skill `command_registration_verification_skill`、`verify_go_changes_skill`、`global_ref_sync_skill`。

# 关键结果
1. 新建 `internal/handler/`，迁移上述 plan/doing/easy 编排函数为 handler 方法；handler 编排 = env.Ensure → builder.Build（产 method+instance）→ 调用 runtime → 持久化 sessionID
2. `internal/cmd/{plan,doing,easy}.go` 变薄：仅保留 Cobra 命令 + flag 解析 + 调 handler；全部 flag 行为不变；删除死代码 `generateJobID`
3. 命令注册不变（`grep -n AddCommand internal/cmd/root.go` 8 命令）
4. doing 仍经 executor 执行（本 task 不改调度，仅搬编排）；logging/git/parser/actpath/executor 均保留；随函数迁移同步改写 `internal/cmd/{plan,doing,easy}_test.go` 中引用已迁函数的断言（executePlanWorkflow/executeDoingWorkflow/loadTasksFromPlan 等）；另改写 `tools_test.go`（`runDoingDryRun` 引用）与 `dryrun_integration_test.go`（`runPlanDryRun` 引用）
5. `go build` + `go test ./internal/cmd/... ./internal/handler/... -timeout 60s -v` 全绿

# 测试方法
（本 task 测试用例设计遵循 skill:tdd + skill:testing-anti-patterns：先写失败测试；测试真实 CLI 行为；dry-run 断言先定位 section 再检查——见 testing-conventions.md。）

1. 正常路径：前置条件 = task3/4/5 完成；输入 = 无；操作 = `go build -o bin/rick ./cmd/rick && go test ./internal/cmd/... ./internal/handler/... -timeout 60s -v`；预期 = build 成功，测试全绿。
2. 边界（dry-run 变量替换 + 空 requirement）：前置条件 = build 成功；输入 = `plan --dry-run`、`easy --dry-run -r ""`；操作 = `./bin/rick plan --dry-run | grep -c '{{'`（预期 0）+ `./bin/rick easy --dry-run -r ""`（预期报 `requirement cannot be empty`）；预期 = 无 `{{` 且空 requirement 报错。
3. 异常（无效 job + 重入不存在 plan + --ctx 冲突）：前置条件 = build 成功；输入 = `doing job_nonexistent`、`plan --job job_nonexistent`、`easy --ctx <已有 loops 的路径>`；操作 = 依次运行；预期 = 分别报 `job directory not found`、`plan directory does not exist`、`local context already exists`，均 exit 非 0。
