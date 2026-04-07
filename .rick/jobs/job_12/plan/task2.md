# 依赖关系
task1

# 任务名称
重建 `.rick/skills/` 为 Markdown 技能说明书

# 任务目标
将 `.rick/skills/` 重建为只含 Markdown 文件的技能说明书目录。创建 2 个 skill Markdown 文件（`verify_rick_check_commands.md` 和 `test_go_project_changes.md`），并更新 `index.md` 使每个 skill 都有明确的触发场景。

# 关键结果
1. 创建 `.rick/skills/verify_rick_check_commands.md`，描述如何验证 rick check 命令行为，包含触发场景、使用的 tools、执行步骤、示例
2. 创建 `.rick/skills/test_go_project_changes.md`，描述如何测试 Go 项目代码变更，包含触发场景、使用的 tools、执行步骤、示例
3. 更新 `.rick/skills/index.md`，格式为 `| Skill | 描述 | 触发场景 |` 三列表格，每行触发场景列非空
4. `.rick/skills/` 中只有 `.md` 文件，无 `.py` 文件
5. 更新 `tests/tools_integration_test.sh` 的 scenario 10（skills injection dry-run），改为创建 `.md` skill 文件并验证 skills section 显示 Markdown 名称而非 `.py` 列表

# 测试方法
1. 运行 `ls .rick/skills/` 验证只有 `.md` 文件
2. 检查 `index.md` 包含三列表格且每个 skill 的触发场景列非空（无空的 `| |` 列）
3. 检查 `verify_rick_check_commands.md` 包含"触发场景"、"使用的 Tools"、"执行步骤"三个 section
4. 检查 `test_go_project_changes.md` 包含"触发场景"、"使用的 Tools"、"执行步骤"三个 section
5. 构建 rick 并运行 `{bin} doing job_12 --dry-run` 验证 skills section 显示 Markdown skill 名称和触发场景（而非 `.py` 文件列表）
