# 依赖关系
task2

# 任务名称
更新 Go embed 和 human_loop_prompt.go，注入 sub agent 路径

# 任务目标
按照 plan_prompt.go 的注入模式，更新 `internal/prompt/manager.go` 注册三个新 embed，更新 `internal/prompt/human_loop_prompt.go` 将三个 sub agent 模板各自写到 tmp 文件，再把 tmp 路径注入主模板的 `{{think_agent_path}}`、`{{learn_agent_path}}`、`{{express_agent_path}}`。`human_loop.go` 负责在会话结束后清理全部 tmp 文件（主 prompt + 三个 sub agent）。同时修复 dry-run：当前只打印一行占位消息，需改为输出完整 prompt 内容（对齐 plan 的 dry-run 规范）。

# 关键结果
1. `manager.go` 新增三个 embed 声明（`humanLoopThinkTemplate`、`humanLoopLearnTemplate`、`humanLoopExpressTemplate`）并在 `getEmbeddedTemplate` 中注册
2. `human_loop_prompt.go` 的 `GenerateHumanLoopPromptFile` 返回值扩展为 `(mainFile string, subAgentFiles []string, err error)`，subAgentFiles 包含三个 sub agent 的 tmp 路径
3. 新增 `GenerateHumanLoopPrompt` 函数（返回 string，供 dry-run 使用），dry-run 时路径显示为占位描述（如 `<tmp>/human_loop_think_*.md`）
4. `human_loop.go` 的 dry-run 分支改为调用 `GenerateHumanLoopPrompt` 并打印完整内容；正常分支在 `defer` 中清理所有 tmp 文件
5. `go build ./...` 编译通过

# 测试方法
1. 编译检查：`python3 tools/check_go_build.py`
2. dry-run 输出包含 think 路径关键词：`python3 tools/check_prompt_variables.py --command "$(python3 tools/build_and_get_rick_bin.py) human-loop --dry-run '测试主题'" --variables "think_agent_path"`
3. dry-run 输出包含 learn 路径关键词：`python3 tools/check_prompt_variables.py --command "$(python3 tools/build_and_get_rick_bin.py) human-loop --dry-run '测试主题'" --variables "learn_agent_path"`
4. dry-run 输出包含 express 路径关键词：`python3 tools/check_prompt_variables.py --command "$(python3 tools/build_and_get_rick_bin.py) human-loop --dry-run '测试主题'" --variables "express_agent_path"`
5. 验证路径真实可读：dry-run 之外实际运行时，执行 `rick human-loop '测试主题' --dry-run` 后检查输出中的路径是临时文件路径且文件存在（通过在 dry-run 模式下临时保留文件并打印路径来验证）
