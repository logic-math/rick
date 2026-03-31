# 依赖关系


# 任务名称
为 `rick plan` 命令增加 `--job` flag 支持重进已有 job

# 任务目标
当前 `rick plan` 每次都会创建新的 job（调用 `NextJobID()`）。新增 `--job <job_id>` flag，当用户指定已有 job ID 时，直接复用该 job 的 plan 目录，不创建新 job，重新进入 Claude 交互式规划会话。

使用场景：用户对某个 job 的规划不满意，想重新和 AI 讨论，执行 `rick plan --job job_1` 即可。

⚠️ **注意：`root.go` 已通过 `PersistentFlags()` 定义了全局 `--job` flag。Cobra 不允许子命令用 `LocalFlags()` 再定义同名 flag，否则运行时 panic：`flag redefined: job`。**

正确做法：**不新增本地 flag**，直接复用全局 `--job` flag，在 `plan.go` 的 `RunE` 中通过 `GetJobID()` 读取其值。

# 关键结果
1. `plan.go` 的 `RunE` 中，在调用 `executePlanWorkflow` 前先检查 `GetJobID()`：非空则走"重进已有 job"路径，空则走原有"创建新 job"路径
2. "重进已有 job"路径：跳过 `NextJobID()` 和目录创建，直接用 `GetJobPlanDir(jobID)` 获取已有 plan 目录
3. 若指定的 job plan 目录不存在，返回明确错误：`"job job_X plan directory does not exist, use 'rick plan' to create a new job"`
4. 未指定 `--job` 时，行为与原来完全一致（创建新 job）
5. **不新增任何新 flag**，`go build ./...` 编译通过，无 panic

# 测试方法
1. 运行 `go build ./...`，确认编译无错误，无 `flag redefined` panic
2. 运行 `go test ./internal/cmd/...`，确认所有测试通过
3. 在 dry-run 模式测试：`rick --job job_1 plan --dry-run "需求"`，应输出 "[DRY-RUN] Would re-enter plan for job: job_1" 并退出
4. 指定不存在的 job：`rick --job job_999 plan "需求"`，应返回错误 "job job_999 plan directory does not exist..."
5. 不带 `--job` 执行 `rick plan --dry-run "需求"`，确认仍创建新 job，行为不变
