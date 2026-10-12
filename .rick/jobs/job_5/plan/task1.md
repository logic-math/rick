# 依赖关系
无

# 任务名称
清理无用依赖和通用化 project_name

# 任务目标
移除未使用的 go-git 外部依赖和 pkg/feedback 模块，消除 doing.go 中的死代码，将提示词中硬编码的 project_name 改为从 workspace 动态读取，使 Rick 成为真正通用的 AI Coding Framework。

# 关键结果
1. 完成 go.mod / go.sum 中 go-git/go-git/v5 及其所有间接依赖的移除，`go build ./...` 通过
2. 完成 pkg/feedback/ 整个目录的删除，internal/cmd/feedback_helper.go 和对应测试文件删除
3. 完成 internal/cmd/doing.go 中死代码 callClaudeCodeForTask 和 promptForRetry 函数的移除
4. 完成 internal/cmd/learning.go.backup 文件删除
5. 完成 internal/prompt/plan_prompt.go 和 doing_prompt.go 中 project_name 的动态化：新增 workspace.GetProjectName() 函数，优先读取 .rick/PROJECT.md 第一行，fallback 为 go.mod module 名，再 fallback 为 filepath.Base(cwd)
6. 所有现有 Go 单元测试通过（go test ./...）

# 测试方法
1. 运行 `go build ./...`，验证编译成功，无 go-git 相关错误
2. 运行 `go test ./...`，验证所有测试通过
3. 运行 `grep -r "go-git" go.mod go.sum`，验证结果为空
4. 运行 `grep -r "pkg/feedback" internal/`，验证结果为空
5. 运行 `ls pkg/`，验证 feedback 目录不存在（或 pkg/ 目录为空）
6. 在项目根目录运行以下验证：`MOCK_SCENARIO=plan_success claude_code_path=python3 ./bin/rick plan "test"`（或直接在代码中单元测试 `workspace.GetProjectName()`），验证返回值为动态项目名而非 "Rick CLI"；同时通过 `go test ./internal/workspace/... -run TestGetProjectName` 覆盖三种 fallback 场景
