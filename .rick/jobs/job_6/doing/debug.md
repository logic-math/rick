## task2: 实现 human-loop 命令的 prompt 生成函数和 RFC 目录管理

**分析过程 (Analysis)**:
- 阅读了 `internal/prompt/doing_prompt.go` 了解现有 prompt 生成函数的模式
- 阅读了 `internal/prompt/manager.go` 了解模板加载机制（支持嵌入模板）
- 阅读了 `internal/workspace/paths.go` 了解路径函数风格（均使用 `GetRickDir()` 作为基础）
- 发现 `NewPromptManager` 签名为 `NewPromptManager(templateDir string)`，但 task2.py 测试脚本调用 `prompt.NewPromptManager()`（无参数），需要改为可变参数
- 选择将 `NewPromptManager` 改为 variadic 函数，保持向后兼容

**实现步骤 (Implementation)**:
1. 在 `internal/workspace/paths.go` 末尾追加 `GetRFCDir()` 函数，返回 `.rick/RFC` 绝对路径
2. 创建 `internal/prompt/templates/human_loop.md` 模板，包含 `{{topic}}` 和 `{{rfc_dir}}` 变量
3. 在 `internal/prompt/manager.go` 中添加 `human_loop.md` 的 embed 声明，并在 `getEmbeddedTemplate` 中注册
4. 将 `NewPromptManager(templateDir string)` 改为 `NewPromptManager(templateDir ...string)` 以支持无参调用
5. 创建 `internal/prompt/human_loop_prompt.go`，实现 `GenerateHumanLoopPromptFile(topic, rfcDir string, manager *PromptManager) (string, error)`
6. 在 `internal/workspace/workspace_test.go` 中添加 `TestGetRFCDir` 测试，并补充 `strings` import

**遇到的问题 (Issues)**:
- task2.py 测试脚本路径计算有误（多了一层 `..`），导致 project_root 指向父目录，无法直接通过脚本测试；已通过手动运行 go 命令验证
- `NewPromptManager` 原为必须传参，但 task2.py 动态生成的测试代码调用无参版本，需改为 variadic

**验证结果 (Verification)**:
- 测试命令：`go build ./...` 及 `go test ./internal/prompt/... ./internal/workspace/...`
- 测试输出：
  ```
  BUILD OK
  ok  	github.com/sunquan/rick/internal/prompt	0.029s
  ok  	github.com/sunquan/rick/internal/workspace	0.015s
  ```
- 结论：✅ 通过
