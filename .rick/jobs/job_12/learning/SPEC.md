<!-- 变更说明：本次 job_12 执行后更新
- 新增：开发规范 - Skills/Tools 分离规范（原因：RFC-002 落地，明确 .py 放 tools/，.md 放 skills/）
- 新增：开发规范 - Mock Agent 同步要求（原因：发现 mock_agent 与 check 命令格式不同步导致集成测试失败）
- 新增：路径约定 - tools/ 目录说明补充（原因：tools/ 是项目级工具，区别于 .rick/skills/）
- 修改：路径约定 - .rick/skills/ 说明（原因：skills/ 现在只含 .md，不含 .py）
-->
# SPEC

## 技术栈

- 语言: Go 1.21+（主程序），Python 3.8+（tools 脚本和测试脚本）
- 框架: Cobra（CLI 命令框架），Goldmark（Markdown 解析）
- 测试: Go testing 标准库，Python unittest，Bash integration tests
- 其他: Git（版本管理），Claude Code CLI（AI agent 集成）

## 架构设计

- 架构风格: 命令行工具，模块化分层架构（cmd → executor → prompt/workspace/git）
- 模块划分: cmd（命令处理）/ executor（任务执行引擎）/ prompt（提示词管理）/ workspace（路径管理）/ parser（内容解析）/ git（Git 操作）/ callcli（Claude 集成）
- 工具链模块: `rick tools` 子命令体系，plan_check/doing_check/learning_check/merge 四个子命令
- 接口设计: check 命令统一输出格式（✅/❌ + 描述），exit code 0=pass / 1=fail
- human-loop 模块: `rick human-loop <topic>` 命令，通过 SENSE 方法论模板引导 Claude 对复杂主题进行深度分析，产出存入 `.rick/RFC/` 目录
- tools 扫描模块: `workspace/tools.go` 扫描 `projectRoot/tools/*.py`，提取 `# Description:` 注释，注入 plan/doing 提示词
- skills 注入模块: `workspace/skills.go` 优先读取 `.rick/skills/index.md` 全文，注入 plan/doing 提示词

## 开发规范

- 代码风格: Go 标准格式（gofmt），函数命名 camelCase，导出函数 PascalCase
- check 命令规范: 默认只报告问题，`--auto-fix` 标志才触发 Claude 修复，保持确定性
- **Skills/Tools 分离规范**:
  - `tools/*.py`：确定性工具脚本，原子化，单一职责，JSON 输出，文件首行必须有 `# Description:` 注释
  - `.rick/skills/*.md`：组合技能说明书，描述在特定场景下如何组合使用 tools，必须包含"触发场景"、"使用的 Tools"、"执行步骤"三节
  - 严禁在 `.rick/skills/` 放 `.py` 文件，严禁在 `tools/` 放 `.md` 文件
- Tools 脚本规范: Python 文件，argparse 解析参数，JSON 输出结果（`{"pass": bool, "errors": [...]}`）
- 测试要求: 单元测试覆盖核心逻辑，集成测试覆盖 CLI 命令，mock_agent 替代真实 Claude 调用
- **Mock Agent 同步要求**: `tests/mock_agent/mock_agent.py` 和 `tools/mock_agent_testing.py` 的 mock 输出格式必须与 doing_check/learning_check 期望严格对齐；当 check 命令格式规范变更时，两个 mock_agent 文件需同步更新
- 路径规范: 测试脚本位于 `.rick/jobs/job_N/doing/tests/`，需要 6 次 dirname 到达项目根目录
- Go variadic 改造模式: 当需要让现有必传参数变为可选时，使用 variadic（`...T`）而非新增无参构造函数，保持接口唯一性；调用方无需修改
- 包内函数共享: 同一 Go 包内的函数（如 `callClaudeCodeCLI`）可在多个文件中直接调用，不需要重新声明或导出
- Dry-run 规范: `--dry-run` 标志必须输出完整的 prompt 内容（而非占位消息），便于调试和验证上下文注入效果
- **测试断言精确性**: dry-run 输出包含大量上下文文本，断言需先定位 section（如 `## 可用的项目 Skills`）再检查内容，避免全文搜索误判

## 工程实践

- 版本控制: Git，每个任务完成后独立 commit（commit message 包含 task ID）
- 知识合并: `rick tools merge <job_id>` 在 `learning/job_N` 分支执行，人工审核后 `git merge --no-ff`
- 持续集成: `go test ./...` 覆盖单元测试，`bash tests/tools_integration_test.sh` 覆盖集成测试
- 发布流程: `./scripts/build.sh` 构建，`./scripts/install.sh` 安装到 `~/.rick/bin/rick`

## 路径约定

- `.rick/RFC/`: human-loop 会话产出文档目录，由 `GetRFCDir()` 管理，`rick human-loop` 执行时自动创建
- `.rick/jobs/job_N/`: 每次 job 的工作目录，包含 plan/doing/learning 三个子目录
- `.rick/jobs/job_N/plan/OKR.md`: job 级 OKR，由 plan 阶段 Claude 生成，doing/learning 阶段读取
- `.rick/skills/`: 可复用技能说明书（**只含 `.md` 文件**），doing/plan 阶段自动注入提示词；`.py` 脚本必须放 `tools/`
- `.rick/skills/index.md`: Skills 主索引文件（优先于 README.md），含触发场景列，由人工维护或 `GenerateSkillsIndex()` 生成；格式为 `| Skill | 描述 | 触发场景 |` 三列表格
- `.rick/wiki/`: 系统运行原理文档，供人类阅读
- `<projectRoot>/tools/`: 项目特定 Python 工具脚本（**只含 `.py` 文件**），plan/doing 阶段自动扫描并注入提示词；每个脚本首行必须有 `# Description:` 注释

## 命令规范

### rick human-loop

- 必须提供 topic 参数（位置参数），否则返回 "topic is required" 错误
- 支持 `--dry-run` 标志，输出 `[DRY-RUN] Would start human-loop session for topic: <topic>` 后退出
- 自动创建 `.rick/RFC/` 目录（MkdirAll，幂等）
- 复用 `callClaudeCodeCLI`（plan.go 中定义，同包内共享，不重复声明）
- 会话结束后打印提示，引导用户查看 `.rick/RFC/` 目录

### rick plan --job

- `--job <job_id>` 为全局 flag（定义在 root.go），plan.go 通过 `GetJobID()` 读取，不在 plan.go 中重复定义
- 指定 `--job` 时跳过 `NextJobID()`，直接复用已有 job 的 plan 目录
- plan 目录不存在时返回明确错误，不自动创建

### rick plan --dry-run

- 生成完整 plan prompt 并打印到 stdout（通过 `runPlanDryRun()` 函数）
- 不调用 Claude，不创建任何文件
- 输出包含所有注入内容：skills_index、tools_list、job_plan_dir 等

### rick doing --dry-run

- 打印完整 doing prompt 内容到 stdout
- 不调用 Claude，不执行任何任务
- 展示第一个非 success 状态的任务（从 tasks.json 读取，不硬编码 task1）

### rick learning --dry-run

- 生成完整 learning prompt 并打印到 stdout（通过 `runLearningDryRun()` 函数）
- 不调用 Claude，不创建任何文件
- 输出包含所有注入内容：okr_content、task_md_content、debug 记录等
