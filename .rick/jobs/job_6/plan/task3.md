# 依赖关系
task1, task2

# 任务名称
实现 `rick human-loop` CLI 命令

# 任务目标
在 `internal/cmd/` 下新增 `human_loop.go`，实现 `NewHumanLoopCmd()` 函数，并在 `root.go` 中注册该命令。

命令行为：
1. 接收一个必选参数 `topic`（用户的思考主题，如 `'如何重构?'`）
2. 获取 RFC 目录路径（`GetRFCDir()`），并自动创建该目录
3. 生成 human_loop 提示词临时文件（调用 `GenerateHumanLoopPromptFile`）
4. 启动 Claude Code CLI 交互式会话（**直接复用 `plan.go` 中已有的 `callClaudeCodeCLI` 函数，不要重复定义**，两者在同一 `cmd` 包内）
5. 会话结束后提示用户："思考记录已保存到 .rick/RFC/ 目录（如果 AI 已执行 sense-express）"

⚠️ **注意：`callClaudeCodeCLI` 已定义在 `plan.go` 中，`human_loop.go` 直接调用即可。若重复定义会导致编译报错 `callClaudeCodeCLI redeclared in this block`。**

# 关键结果
1. 新增 `internal/cmd/human_loop.go`，包含 `NewHumanLoopCmd()` 函数
2. `root.go` 中 `rootCmd.AddCommand(NewHumanLoopCmd())` 注册命令
3. `human_loop.go` 中调用 `callClaudeCodeCLI`，不重新定义该函数
4. `rick human-loop '如何重构?'` 能成功启动 Claude 交互式会话
5. `rick human-loop` 无参数时输出 "topic is required" 错误
6. `go build ./...` 编译通过，`go test ./internal/cmd/...` 测试通过

# 测试方法
1. 运行 `go build ./...`，确认编译无错误（特别验证无 `redeclared` 错误）
2. 运行 `go test ./internal/cmd/...`，确认所有测试通过
3. 在 dry-run 模式下测试：`rick human-loop --dry-run '如何重构?'`，应输出 "[DRY-RUN] Would start human-loop session for topic: 如何重构?" 并退出
4. 无参数运行 `rick human-loop`，应返回错误 "topic is required"
5. 用 mock binary 测试 Claude 调用：创建 `exit 0` 的 shell script，通过 `config.ClaudeCodePath` 注入，验证命令能正常走完流程
6. 检查 `.rick/RFC/` 目录在命令执行时被自动创建
