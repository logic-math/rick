# 依赖关系


# 任务名称
迁移 `.rick/skills/*.py` 到 `tools/` 目录

# 任务目标
创建项目根目录下的 `tools/` 目录，将 `.rick/skills/` 中所有 `.py` 脚本迁移过去，删除 `rick_tools_check_pattern.py`（模式文档，不是工具），并验证 `rick doing --dry-run` 的 tools section 非空。

# 关键结果
1. 创建 `tools/` 目录，包含 5 个迁移的 `.py` 文件：`build_and_get_rick_bin.py`、`check_go_build.py`、`check_prompt_variables.py`、`check_variadic_api.py`、`mock_agent_testing.py`
2. `.rick/skills/` 中不再有任何 `.py` 文件（`rick_tools_check_pattern.py` 已删除）
3. `rick doing job_12 --dry-run` 输出中 tools section 列出 `tools/` 下的工具

# 测试方法
1. 运行 `ls tools/` 验证 5 个 `.py` 文件存在
2. 运行 `ls .rick/skills/*.py 2>/dev/null || echo "no py files"` 验证无 `.py` 文件
3. 运行 `python3 tools/build_and_get_rick_bin.py` 验证脚本可执行（返回 JSON）
4. 运行 `python3 tools/check_go_build.py --help` 或 `python3 tools/check_go_build.py` 验证脚本可执行
5. 构建 rick 并运行 `{bin} doing job_12 --dry-run` 验证 tools section 非空
