# 依赖关系


# 任务名称
优化 doing 重试循环的测试失败信息传递

# 任务目标
当前 doing 重试循环（`internal/executor/retry.go`）在测试失败时，传递给下一轮 doing agent 的错误信息不够完整：

1. `retry.go:119-121`：`execResult.Output` 被截断到 500 字符（`output[:500] + "... (truncated)"`），可能丢失关键的测试失败信息
2. `runner.go:113`：`result.Error` 只包含 `strings.Join(testResult.Errors, "; ")`（JSON errors 数组的 join），而完整的 `testOutput`（包含 stderr、traceback 等）没有被充分利用
3. `runner.go:103-105`：当 `ExecuteTestScript` 返回 error 时（如脚本语法错误、JSON 解析失败），`result.Error` 只有简单的错误消息，缺乏完整的脚本输出

需要改进失败信息的传递，让 doing agent 在重试时能看到更完整、更有价值的测试失败上下文，从而快速定位和修复问题。

**具体修改点**：

1. `internal/executor/runner.go`：
   - 在 `RunTask()` 的失败路径中，将完整的 `testOutput`（而非只有 `testResult.Errors`）保存到 `result.Error` 或新增 `result.TestOutput` 字段
   - 确保 test 脚本的 stderr 输出、完整错误信息都包含在传递给下一轮的内容中

2. `internal/executor/retry.go`：
   - 移除或大幅提高 500 字符的截断限制（当前 `output[:500] + "... (truncated)"`）
   - 改为智能截断：优先保留错误信息（stderr、errors 数组），截断冗余的 Claude 输出
   - 调整 `testErrorFeedback` 的累积策略：对于多次重试，只保留最近 N 次（如最近 2 次）的完整失败信息，避免 prompt 过长

3. `internal/executor/executor.go`（如需）：确认 doing_dir 路径正确传递给 runner，以便 doing_check 能找到正确的 tasks.json 和 debug.md

**关键约束**：
- 修改必须保持 `TaskExecutionResult` 结构体和 `RetryResult` 结构体的向后兼容性
- 不能破坏现有的 `ExecuteTestScript` 返回值契约（TestResult + rawOutput + error）
- testErrorFeedback 总长度应有合理上限（建议 3000 字符），防止 prompt 过长

# 关键结果
1. `runner.go` 中测试失败时，`result.Error` 或 `result.Output` 包含完整的 testOutput（含 stderr）
2. `retry.go` 中移除 500 字符硬截断，改为基于内容类型的智能截断（优先保留错误信息）
3. `retry.go` 中 testErrorFeedback 累积策略改为只保留最近 2 次完整失败信息（避免 prompt 膨胀）
4. 修改后 `go test ./internal/executor/... -v` 全部通过
5. 通过代码审查确认：doing agent 在第 2 次重试时能看到第 1 次的完整 test 脚本输出（包括 Python traceback、具体的断言失败信息等）

# 测试方法
1. 运行 `go build ./...` 确认编译通过
2. 运行 `go test ./internal/executor/... -v` 确认 executor 相关测试通过
3. 检查 `retry.go` 中不再有 `output[:500]` 的硬截断代码
4. 检查 `runner.go` 中 test 失败路径，`result.Error` 包含 testOutput 的关键内容（不只是 errors join）
5. 运行 `go test ./... -count=1` 确认全量测试通过
