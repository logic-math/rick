# 依赖关系
task2

# 任务名称
新增 tools/ 扫描机制并注入 plan/doing prompt

# 任务目标
RFC-001 要求在用户项目根目录建立 `tools/` 目录，存放确定性 Python 工具脚本。这些 tools 是项目交付物的一部分，人和 AI 都可以使用。

当前 rick 没有任何 tools 扫描机制。本任务目标：
1. 在 `workspace` 包新增 `LoadToolsList(projectRoot string)` 函数，扫描项目根目录 `tools/*.py`，提取每个脚本顶部注释（`# Description: ...`）作为描述
2. 在 `doing_prompt.go` 新增 `formatToolsSection(projectRoot string)` 函数，生成 tools 列表注入 doing prompt
3. 在 `plan_prompt.go` 同样注入 tools 列表（模板变量 `{{tools_list}}`）
4. 在 `plan.md` 和 `doing.md` 模板中添加对应的 tools 章节，强制要求 agent 优先使用已有 tools

**注意**：`tools/` 在用户项目根目录（`os.Getwd()`），不在 `.rick/` 下。

# 关键结果
1. `workspace/tools.go` 新增 `ToolInfo` 结构体和 `LoadToolsList(projectRoot string) ([]ToolInfo, error)` 函数，扫描 `projectRoot/tools/*.py`
2. `doing_prompt.go` 新增 `formatToolsSection()` 并在 `GenerateDoingPromptFile()` 中调用，注入 tools 列表
3. `plan_prompt.go` 新增 tools 注入，在 `GeneratePlanPromptFile()` 中调用
4. `doing.md` 模板新增"可用的项目 Tools"章节，强调优先使用
5. `plan.md` 模板新增"可用的项目 Tools"章节
6. `go test ./...` 全部通过

# 测试方法
1. 运行 `go build ./...` 确认编译通过
2. 在测试目录创建 `tools/sample_tool.py`（含 `# Description: 示例工具`），运行 `go test ./internal/workspace/ -run TestLoadToolsList -v` 确认扫描正确
3. 运行 `go test ./internal/prompt/ -v` 确认 prompt 相关测试通过
4. 运行 `go test ./...` 确认全量测试通过
5. 构建后运行 `rick doing --dry-run job_9`，检查输出 prompt 包含 tools 列表章节
