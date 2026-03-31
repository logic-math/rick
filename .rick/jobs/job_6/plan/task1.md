# 依赖关系


# 任务名称
实现 `human-loop` 命令的提示词模板

# 任务目标
在 `internal/prompt/templates/` 下新增 `human_loop.md` 模板文件，内容完全参考 `sense-human-loop` skill 的提示词体系（主控 Agent + sense-think + sense-learn + sense-express 三个子 Agent），并在 `PromptManager` 中注册该模板的 embed 支持。

模板需要包含两个变量占位符：`{{topic}}` 注入用户的思考主题，`{{rfc_dir}}` 告知 AI 思考完成后将文档保存到哪个目录。

⚠️ **注意：模板内容中禁止出现 `{{` `}}` 形式的非变量文本**（如示例代码、说明性文字），否则 `extractVariables` 会将其误识别为变量，导致 `GetMissingVariables()` 报告缺失变量。如需在模板中展示双花括号示例，改用代码块或中文括号替代。

# 关键结果
1. 新增 `internal/prompt/templates/human_loop.md`，包含完整的 SENSE Human Loop 提示词（主控 Agent + sense-think + sense-learn + sense-express 三个子 Agent 规则）
2. 模板中**只有** `{{topic}}` 和 `{{rfc_dir}}` 两个 `{{}}` 占位符，无其他双花括号文本
3. 在 `internal/prompt/manager.go` 中添加 `//go:embed templates/human_loop.md` 和对应的 `humanLoopTemplate` 变量
4. `getEmbeddedTemplate` 方法中注册 `"human_loop"` case
5. `go build ./...` 编译通过，无错误

# 测试方法
1. 运行 `go build ./...`，确认编译无错误
2. 运行 `go test ./internal/prompt/...`，确认现有测试全部通过
3. 在测试中调用 `pm.LoadTemplate("human_loop")`，确认能成功加载模板且不报错
4. 检查加载的模板内容包含 `{{topic}}` 和 `{{rfc_dir}}` 占位符
5. 调用 `builder.GetMissingVariables()`，确认只返回 `["topic", "rfc_dir"]` 两个变量，无多余条目
