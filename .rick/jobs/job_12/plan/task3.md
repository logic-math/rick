# 依赖关系


# 任务名称
更新 learning 提示词模板区分 tools 和 skills

# 任务目标
修改 `internal/prompt/templates/learning.md`，将"按需产出四类知识文档"中的 `skills/*.py` 改为 `tools/*.py`，并新增 `skills/*.md` 产出类型，明确区分两者的定义、格式、输出目录和触发条件，防止未来 AI 重蹈覆辙。

# 关键结果
1. learning.md 中"按需产出"部分将 `skills/*.py` 改为 `tools/*.py`，描述为"确定性工具脚本"
2. learning.md 中新增 `skills/*.md` 产出类型，描述为"组合技能说明书，描述在特定场景下如何组合使用 tools"
3. learning.md 中明确 tools 输出目录为 `{{learning_dir}}/tools/*.py`，skills 输出目录为 `{{learning_dir}}/skills/*.md`
4. learning.md 中的 checklist 和其他引用 `skills/*.py` 的地方同步更新

# 测试方法
1. 检查 `internal/prompt/templates/learning.md` 不包含 `skills/*.py` 字样（grep 验证）
2. 检查 learning.md 包含 `tools/*.py` 字样（grep 验证）
3. 检查 learning.md 包含 `skills/*.md` 字样（grep 验证）
4. 检查 learning.md 中 tools 和 skills 的定义区分清晰（人工阅读验证）
5. 运行 `go build ./...` 确保模板变更不影响 Go 编译
