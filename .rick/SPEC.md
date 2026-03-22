<!-- Updated in learning phase: job_5 (2026-03-22) -->
<!-- Changes: Added tools subcommand conventions, skills script standard, auto-fix pattern -->
# SPEC

## 技术栈

- 语言: Go 1.21+（主程序），Python 3.8+（skills 脚本和测试脚本）
- 框架: Cobra（CLI 命令框架），Goldmark（Markdown 解析）
- 测试: Go testing 标准库，Python unittest，Bash integration tests
- 其他: Git（版本管理），Claude Code CLI（AI agent 集成）

## 架构设计

- 架构风格: 命令行工具，模块化分层架构（cmd → executor → prompt/workspace/git）
- 模块划分: cmd（命令处理）/ executor（任务执行引擎）/ prompt（提示词管理）/ workspace（路径管理）/ parser（内容解析）/ git（Git 操作）/ callcli（Claude 集成）
- 工具链模块: `rick tools` 子命令体系，plan_check/doing_check/learning_check/merge 四个子命令
- 接口设计: check 命令统一输出格式（✅/❌ + 描述），exit code 0=pass / 1=fail

## 开发规范

- 代码风格: Go 标准格式（gofmt），函数命名 camelCase，导出函数 PascalCase
- check 命令规范: 默认只报告问题，`--auto-fix` 标志才触发 Claude 修复，保持确定性
- Skills 脚本规范: Python 文件，argparse 解析参数，JSON 输出结果（`{"pass": bool, "errors": [...]}`）
- 测试要求: 单元测试覆盖核心逻辑，集成测试覆盖 CLI 命令，mock_agent 替代真实 Claude 调用
- 路径规范: 测试脚本位于 `.rick/jobs/job_N/doing/tests/`，需要 6 次 dirname 到达项目根目录

## 工程实践

- 版本控制: Git，每个任务完成后独立 commit（commit message 包含 task ID）
- 知识合并: `rick tools merge <job_id>` 在 `learning/job_N` 分支执行，人工审核后 `git merge --no-ff`
- 持续集成: `go test ./...` 覆盖单元测试，`bash tests/tools_integration_test.sh` 覆盖集成测试
- 发布流程: `./scripts/build.sh` 构建，`./scripts/install.sh` 安装到 `~/.rick/bin/rick`
