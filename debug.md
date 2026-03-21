# debug1: executor tests hanging due to claude calls

**现象 (Phenomenon)**:
- `go test ./...` 在 `internal/executor` 包卡住，超过60秒 timeout
- `TestExecuteJobMultipleTasks` 等测试尝试调用真实的 claude CLI

**复现 (Reproduction)**:
- 运行 `go test -timeout 60s ./...`

**猜想 (Hypothesis)**:
- `TestExecuteJob*` 系列测试调用 `ExecuteJob()`，后者调用真实的 claude CLI
- 在 CI 环境或无 claude 的环境中会卡住

**验证 (Verification)**:
- 确认 `executor.go` 中 `ExecuteJob()` 最终调用 `exec.Command(claudePath, ...)`

**修复 (Fix)**:
- 在所有 `TestExecuteJob*` 函数中添加 `skipIfNoClaude(t)` 
- 使用 `RICK_INTEGRATION_TEST=1` 环境变量控制是否跳过
- 将 `skipIfNoClaude` 改为检查 `RICK_INTEGRATION_TEST` 而非 claude 是否在 PATH

**进展 (Progress)**:
- ✅ 已解决 - `go test -timeout 60s ./...` 零失败

# debug2: internal/cmd 覆盖率低（<70%）

**现象 (Phenomenon)**:
- `go test -cover ./internal/cmd/...` 显示 17.8%，远低于 70% 要求
- 主要原因：`executeDoingWorkflow`、`executeLearningWorkflow`、`executePlanWorkflow` 等函数调用 claude，无法直接测试

**复现 (Reproduction)**:
- 运行 `go test -cover ./internal/cmd/...`

**猜想 (Hypothesis)**:
- 工作流函数直接调用 `claude` CLI，需要 mock 来测试

**验证 (Verification)**:
- 使用 `go test -coverprofile` 分析每个函数的覆盖率

**修复 (Fix)**:
- 创建 `tools_test.go`：为 `runPlanCheck`、`runDoingCheck`、`runLearningCheck` 等纯文件系统函数添加测试
- 添加 merge helper 测试：`checkApproved`、`copyFile`、`copyDir`、`generateWikiREADME`
- 添加 git helper 测试：`getCurrentBranch`、`gitCreateAndSwitch`、`gitCheckout`、`runGit`
- 添加 workflow 测试：使用 mock claude binary 测试 `executeDoingWorkflow`、`executeLearningWorkflow`、`executePlanWorkflow`
- 添加 `collectExecutionData` 和 `runMerge` 测试
- 添加 `commitDoingResults` 和 `ensureGitUserConfigured` 测试
- 添加 Command `RunE` 函数测试

**进展 (Progress)**:
- ✅ 已解决 - internal/cmd: 71.4%, internal/executor: 72.9%, internal/prompt: 76.1%
