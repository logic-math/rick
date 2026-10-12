# 依赖关系

task1, task2

# 任务名称

接线层：runner/executor/doing 组合根重构，完成 DIP 全链路

# 任务目标

将 task1 定义的接口和 task2 实现的适配器，通过修改 runner.go / executor.go / doing.go 接线，完成依赖倒置的完整链路。`doing.go` 作为唯一组合根，`runner.go` 和 `executor.go` 不 import `claudecode`。

**DIP 全链路：**
```
doing.go（组合根）
  → import claudecode，创建 claudeExec
  → executor.NewExecutor(..., claudeExec)
    → NewTaskRunner(config, claudeExec)
      → agentExecutor.Execute(promptFile, taskID)
      → actpath.Generate(session, "doing/tasks/{taskID}/act-path.md")
```

# 关键结果

1. **`internal/executor/runner.go` 重构**：
   - `TaskRunner` struct 新增字段 `agentExecutor agent.AgentExecutor`
   - `NewTaskRunner(config *ExecutionConfig, agentExecutor agent.AgentExecutor)` — 必填参数，无默认值
   - `RunTask` 中调用 `tr.agentExecutor.Execute(doingPromptFile, task.ID)` 替代原 `CallClaudeCodeCLI`
   - Execute 完成后调用 `actpath.Generate(session, filepath.Join(tr.config.WorkspaceDir, "tasks", task.ID, "act-path.md"))`
   - `runner.go` import 列表**不出现** `internal/agent/claudecode`

2. **`internal/executor/executor.go` 级联更新**：
   - `NewExecutor` 签名新增 `agentExecutor agent.AgentExecutor`（放在 `existingTasksJSON ...` 之前）
   - 内部第 91 行 `runner := NewTaskRunner(config)` → `runner := NewTaskRunner(config, agentExecutor)`

3. **`internal/cmd/doing.go` 作为组合根**：
   - `import "github.com/sunquan/rick/internal/agent/claudecode"`
   - `executeDoingWorkflow` 中创建：`claudeExec := claudecode.NewExecutor(cfg.ClaudeCodePath)`
   - 传入：`executor.NewExecutor(tasks, execConfig, doingDir, jobID, claudeExec, existingTasksJSON)`

4. **`internal/executor/runner_test.go` 更新**：
   - 新增 `mockAgentSession` struct（实现 `agent.AgentSession`，返回预设固定数据）
   - 新增 `mockAgentExecutor` struct（实现 `agent.AgentExecutor`，返回 `mockAgentSession`）
   - 所有 `NewTaskRunner(config)` 调用改为 `NewTaskRunner(config, &mockAgentExecutor{})`

5. **`internal/executor/executor_test.go` 更新**：
   - 文件顶部新增 `mockAgentExecutor` 定义（与 runner_test.go 共用模式）
   - 所有 `NewExecutor(tasks, config, tmpDir, "job1")` 调用（约 10 处）改为 `NewExecutor(tasks, config, tmpDir, "job1", &mockAgentExecutor{})`

# 测试方法

1. 编译：`go build ./...`，确认零报错（DIP 链路编译通过）
2. DIP 验证：`grep -r "claudecode" internal/executor/` → 空；`grep -r "claudecode" internal/actpath/` → 空
3. 组合根验证：`grep "claudecode" internal/cmd/doing.go` → 有且仅有 doing.go 引用
4. 单元测试：`go test ./internal/executor/... -v`，runner_test.go + executor_test.go 全部通过
5. **KR1 act-path 生成验证**（OKR KR1，最重要）：
   在 `runner_test.go` 新增 `TestRunTask_ActPathGenerated`：
   - mockAgentExecutor.Execute 返回含 1 个 ToolCall、0 个错误的 mockAgentSession，RawLogPath 设为临时文件路径
   - 调用 `RunTask(task, "", "")`
   - 断言 `doing/tasks/{taskID}/act-path.md` 文件存在
   - 断言文件内容包含 `"## 执行摘要"` 和 `"## 行为轨迹"`
   - 断言文件内容包含 `"报错次数: 0"`
   ```
   go test ./internal/executor/... -v -run TestRunTask_ActPathGenerated
   ```
6. doing dry-run：`python3 tools/check_prompt_variables.py --phase doing --keywords "任务目标"`，`{"pass": true}`
