# 依赖关系


# 任务名称
实现 `human-loop` 命令的 prompt 生成函数和 RFC 目录管理

# 任务目标
在 `internal/prompt/` 下新增 `human_loop_prompt.go`，实现 `GenerateHumanLoopPromptFile` 函数，负责将用户输入的主题和 RFC 输出目录路径注入模板，生成临时提示词文件。

同时在 `internal/workspace/paths.go` 中新增 `GetRFCDir` 函数（**不要新建文件**，该文件是所有路径函数的统一归属）。

# 关键结果
1. 新增 `internal/prompt/human_loop_prompt.go`，包含 `GenerateHumanLoopPromptFile(topic string, rfcDir string, manager *PromptManager) (string, error)` 函数
2. 在 `internal/workspace/paths.go` 末尾追加 `GetRFCDir() (string, error)` 函数，返回 `.rick/RFC/` 的绝对路径，风格与同文件其他函数保持一致
3. `GenerateHumanLoopPromptFile` 将 `topic` 和 `rfcDir` 注入模板，调用 `builder.BuildAndSave("human_loop")` 生成临时文件
4. `go build ./...` 编译通过，无错误

# 测试方法
1. 运行 `go build ./...`，确认编译无错误
2. 运行 `go test ./internal/prompt/...`，确认现有测试全部通过
3. 运行 `go test ./internal/workspace/...`，确认 `GetRFCDir` 返回正确路径（包含 `.rick/RFC`）
4. 在单元测试中调用 `GenerateHumanLoopPromptFile("如何重构?", "/tmp/rfc_test", pm)`，确认返回的文件路径存在且内容包含 "如何重构?"
