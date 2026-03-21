# 依赖关系
task5, task6

# 任务名称
实现 mock_agent 测试工具并完成端到端集成测试

# 任务目标
在测试环境中无法递归启动 Claude Code（`rick plan` / `rick doing` / `rick learning` 会尝试调用 `claude` CLI，在 CI 或嵌套执行中不可用）。需要实现一个 `mock_agent` 工具，模拟 AI agent 在各阶段的交付产物，支持正常和异常两类场景，使完整的 plan→doing→learning→merge 工作流可在无 AI 依赖的环境中测试。同时完成相关文档更新。

# 关键结果

## 1. 实现 tests/mock_agent/mock_agent.py

mock_agent 是一个 Python 脚本，通过环境变量 `RICK_MOCK_AGENT` 控制行为，替代真实的 `claude` CLI 被 rick 调用。

**接口设计**：rick 调用 `claude <prompt_file>` 时，mock_agent 读取 prompt_file 判断阶段，然后按配置生成对应产物。

**调用方式**：在 config.json 中将 `claude_code_path` 设为 `python3 tests/mock_agent/mock_agent.py`，rick 会用它替代真实 claude。

**支持的模拟场景**（通过环境变量 `MOCK_SCENARIO` 控制）：

```
MOCK_SCENARIO=plan_success
  → 在 prompt 指定的 job plan 目录下生成合法的 task1.md～task3.md 和 tasks.json
  → 每个 task.md 包含完整的五个章节，依赖关系合法

MOCK_SCENARIO=plan_missing_section
  → 生成的 task1.md 缺少 "# 关键结果" 章节
  → 用于测试 plan_check 的检测能力

MOCK_SCENARIO=plan_circular_dep
  → 生成 task1.md（依赖 task2）和 task2.md（依赖 task1）
  → 用于测试 plan_check 的循环依赖检测

MOCK_SCENARIO=doing_success
  → 在 doing 目录下生成 tasks.json（所有 task status=success，含 commit_hash）
  → 生成 debug.md（含完整的四段式工作日志）
  → 模拟 git commit（创建一个空 commit）

MOCK_SCENARIO=doing_no_debug
  → 生成 tasks.json 但不生成 debug.md
  → 用于测试 doing_check 的 debug.md 检测

MOCK_SCENARIO=doing_zombie_task
  → 生成 tasks.json，其中一个 task status=running（僵尸状态）
  → 用于测试 doing_check 的僵尸状态检测

MOCK_SCENARIO=learning_success
  → 在 learning 目录下生成：
      SUMMARY.md（含 APPROVED: true 在第一行）
      wiki/dag_execution.md（含合法 Markdown + Mermaid）
      skills/check_go_build.py（含 # Description: 检查 Go 项目编译，语法合法）
      OKR.md（完整格式，含变更注释）
      SPEC.md（完整格式，含变更注释）

MOCK_SCENARIO=learning_bad_skill
  → 生成 learning/skills/bad_skill.py，内含 Python 语法错误
  → 用于测试 learning_check 的语法检查

MOCK_SCENARIO=learning_no_summary
  → 不生成 SUMMARY.md
  → 用于测试 learning_check 的必需文件检测

MOCK_SCENARIO=claude_timeout
  → sleep 999（模拟超时，rick 应在 timeout 后终止）

MOCK_SCENARIO=claude_exit_nonzero
  → 直接 exit(1)（模拟 claude CLI 崩溃）

MOCK_SCENARIO=claude_bad_output
  → 不生成任何产物，只打印乱码到 stdout
  → 用于测试 rick 对无效输出的容错
```

## 2. 完成 tests/tools_integration_test.sh

使用 mock_agent 覆盖以下场景（全部使用 `./bin/rick`，不使用系统安装版本）：

**plan_check 测试**：
- `MOCK_SCENARIO=plan_success` → 运行 plan → 运行 `./bin/rick tools plan_check job_N` → 期望通过
- `MOCK_SCENARIO=plan_missing_section` → 运行 plan → 运行 plan_check → 期望报错含"关键结果"
- `MOCK_SCENARIO=plan_circular_dep` → 运行 plan → 运行 plan_check → 期望报错含"circular"

**doing_check 测试**：
- `MOCK_SCENARIO=doing_success` → 运行 doing → 运行 `./bin/rick tools doing_check job_N` → 期望通过
- `MOCK_SCENARIO=doing_no_debug` → 运行 doing → 运行 doing_check → 期望报错含"debug.md"
- `MOCK_SCENARIO=doing_zombie_task` → 运行 doing → 运行 doing_check → 期望报错含"running"

**learning_check + merge 测试**：
- `MOCK_SCENARIO=learning_success` → 运行 learning → 运行 `./bin/rick tools learning_check job_N` → 期望通过 → 运行 `./bin/rick tools merge job_N` → 验证 git branch 创建、.rick/ 下文件更新
- `MOCK_SCENARIO=learning_bad_skill` → 运行 learning → 运行 learning_check → 期望报错含语法错误信息
- `MOCK_SCENARIO=learning_no_summary` → 运行 learning → 运行 learning_check → 期望报错含"SUMMARY.md"

**AI 异常场景测试**：
- `MOCK_SCENARIO=claude_exit_nonzero` → 运行 doing → 期望 rick 返回非零 exit code，不 panic
- `MOCK_SCENARIO=claude_bad_output` → 运行 doing → 期望 rick 正确处理无效输出，输出可读错误信息
- `MOCK_SCENARIO=claude_timeout` → 运行 doing（设置短 timeout）→ 期望 rick 在超时后终止进程，返回 timeout 错误

**skills 注入测试**：
- 在 `.rick/skills/` 放入 mock skill 脚本 → 运行 `./bin/rick doing job_N --dry-run`（或检查生成的临时提示词文件）→ 验证提示词包含 skills 列表

## 3. 完成文档更新
- `wiki/modules/cmd.md`：添加 tools 子命令说明（plan_check/doing_check/learning_check/merge）
- `README.md`：在命令列表中添加 `rick tools` 说明

## 4. 完成所有 Go 单元测试通过
- `go test ./...` 零失败
- 关键模块覆盖率 >= 70%

# 测试方法
1. 运行 `go build -o bin/rick ./cmd/rick/`，验证编译零错误
2. 运行 `go test ./...`，验证所有单元测试通过
3. 运行 `python3 tests/mock_agent/mock_agent.py --self-test`，验证 mock_agent 各场景可正常生成产物
4. 运行 `bash tests/tools_integration_test.sh`，验证所有集成测试通过（exit code 0）
5. 运行 `go test -cover ./internal/cmd/... ./internal/executor/... ./internal/prompt/...`，验证覆盖率 >= 70%
6. 运行 `./bin/rick tools --help`，验证显示完整的子命令列表
7. 运行 `./bin/rick --help`，验证 tools 命令出现在命令列表中
