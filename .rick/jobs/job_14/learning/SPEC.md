<!-- 变更说明：本次 job_14 执行后更新
- 新增：架构设计 - agent 接口层和 DIP 组合根模式（原因：job_14 建立了 AgentSession/AgentExecutor 接口契约）
- 新增：架构设计 - act-path 生成模块（原因：新增 internal/actpath/ 和 internal/agent/ 两个模块）
- 新增：开发规范 - embed.FS 目录嵌入规则（原因：core-skills 用 embed.FS，与现有 string 变量共存）
- 新增：开发规范 - json.dumps ensure_ascii=False 约定（原因：job_14 因 unicode 转义导致字符串匹配失败）
- 新增：开发规范 - 接口签名协商要求（原因：并行 task 接口不匹配导致 task6 修复成本）
- 修改：架构设计 - 补充 dream/actpath/agent 模块（原因：新增三个模块）
- 修改：路径约定 - 新增 act-path/raw_session/dream 路径（原因：v2 新增产物）
- 修改：命令规范 - 新增 rick dream（原因：新增命令）
-->
# SPEC

## 技术栈

- 语言: Go 1.21+（主程序），Python 3.8+（tools 脚本和测试脚本）
- 框架: Cobra（CLI 命令框架），Goldmark（Markdown 解析）
- 测试: Go testing 标准库，Python unittest，Bash integration tests
- 其他: Git（版本管理），Claude Code CLI（AI agent 集成）

## 架构设计

- 架构风格: 命令行工具，模块化分层架构（cmd → executor → prompt/workspace/git）
- 模块划分: cmd（命令处理）/ executor（任务执行引擎）/ prompt（提示词管理）/ workspace（路径管理）/ parser（内容解析）/ git（Git 操作）/ callcli（Claude 集成）/ agent（接口契约）/ actpath（act-path 生成）
- 工具链模块: `rick tools` 子命令体系，plan_check/doing_check/learning_check/merge 四个子命令
- 接口设计: check 命令统一输出格式（✅/❌ + 描述），exit code 0=pass / 1=fail
- human-loop 模块: `rick human-loop <topic>` 命令，通过 SENSE 方法论模板引导 Claude 对复杂主题进行深度分析，产出存入 `.rick/RFC/` 目录；三个 sub agent 模板通过 Go embed 编译进二进制，运行时写出到 tmp 文件，路径注入主控 prompt
- tools 扫描模块: `workspace/tools.go` 扫描 `projectRoot/tools/*.py`，提取 `# Description:` 注释，注入 plan/doing 提示词
- skills 注入模块: `workspace/skills.go` 优先读取 `.rick/skills/index.md` 全文，注入 plan/doing 提示词
- **agent 接口模块** (`internal/agent/`): 定义 `AgentSession` / `AgentExecutor` 接口契约和 `ToolCall` struct；`claudecode` 子包为唯一实现，只在 `doing.go` 组合根中实例化
- **act-path 生成模块** (`internal/actpath/`): `Generate(session AgentSession, outputFile string) error`，不 import 任何具体 agent 实现，输出含执行摘要/行为轨迹/Agent 最终输出三节
- **DIP 组合根模式**: `doing.go` 是唯一 import `internal/agent/claudecode` 的地方；runner/executor/actpath 仅依赖 `internal/agent` 接口，保证可单元测试
- **dream 模块**: `internal/cmd/dream.go` 实现 `rick dream` 命令（交互式），不生成 act-path，读取 `.rick/dream/readme.md` 提取待处理 jobs，限制修改范围为 wiki/tools/SPEC.md

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
- **task.md 测试方法精确性**: task.md 中"测试方法"描述的命令行调用必须基于工具**实际存在的参数接口**，不得引用尚未实现的参数。plan 阶段生成测试脚本前应验证 `tools/` 下对应工具的 `--help` 输出
- **embed.FS 目录嵌入**: `//go:embed dir`（目录）必须绑定 `embed.FS` 类型；`//go:embed file`（单文件）可绑定 `string`；两者可在同一文件共存。`_ "embed"` 改为 `"embed"` 才能使用 `embed.FS`
- **JSON 输出编码约定**: 所有 Python 工具/测试脚本的 `json.dumps()` 调用必须加 `ensure_ascii=False`，避免中文字符被转义为 `\uXXXX` 导致字符串匹配失败
- **接口签名协商**: 并行 task 中若涉及接口定义和实现，接口 task 应先完成后实现 task 才开始；或在 plan 阶段明确接口签名（不含 context.Context，避免标准库强制依赖）
- **同包测试 mock 命名**: 同一 Go 包的多个测试文件共享命名空间；mock struct 应使用区分前缀（如 `runnerMockExecutor` vs `executorMockExecutor`）避免冲突

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
- `.rick/dream/readme.md`: dream 阶段待处理/已处理 jobs 列表，不存在时自动创建
- `.rick/dream/run_log_{n}.md`: learning 阶段 Step 6 写入的度量文件，格式 `| Job | 模型 | 错误次数 | 工具调用轮次 | 备注 |`
- `<projectRoot>/tools/`: 项目特定 Python 工具脚本（**只含 `.py` 文件**），plan/doing 阶段自动扫描并注入提示词；每个脚本首行必须有 `# Description:` 注释
- `doing/tasks/{taskID}/act-path.md`: 任务执行后自动生成的行为轨迹文件，含工具调用、报错次数、执行时长
- `doing/tasks/{taskID}/raw_session.log`: Claude Code NDJSON 原始流式输出，每行一个 JSON 对象（非 JSON 行也写入）

## 命令规范

### rick doing（DIP 全链路）

- `doing.go` 是唯一 import `internal/agent/claudecode` 的地方（**组合根**）
- `runner.go` 和 `executor.go` 只依赖 `internal/agent` 接口，不 import claudecode
- `actpath.Generate(session, outputFile)` 在每个 task 的 `agentExecutor.Execute` 完成后调用
- session 为 nil 时跳过 act-path 生成（nil guard），不 panic

### rick doing（RED 验证）

- testing agent 生成测试脚本后立即运行 `ExecuteTestScript`
- 若 `pass==true`（意外绿态）→ 写入 debug.md 警告 + 重新触发 test generation
- 最多重试 2 次（`maxREDRetries=2`），2 次后仍绿态则记录 warn 继续执行，不阻断

### rick dream

- 读取 `.rick/dream/readme.md` 的"待处理 Jobs"列表，最多取 5 个
- readme.md 不存在时自动创建默认内容（幂等）
- 交互式调用 `callClaudeCodeCLI`（不生成 act-path）
- `--dry-run` 输出完整提示词含 sense + evolve-skills core-skills
- **变更约束**: 仅允许修改 `wiki/`、`tools/`、`SPEC.md`，严禁修改业务代码

### NDJSON 解析规范

- Claude Code `--output-format stream-json` **必须加 `--verbose`**，否则报错退出
- `tool_use`/`tool_result` 嵌套在 `message.content[]` 内，不在顶层
- 非 JSON 行: `log.Printf("warn: skip non-json line %d: %s")` 后继续，不 panic
- 截断规范: Input/Output 截断 300 字符，FinalMessage 截断 200 字符，用 `[]rune` 处理 Unicode

### human-loop 规范

- dry-run 输出中 sub agent 路径为占位符格式（如 `<tmp>/human_loop_think_*.md`），不含真实 `/tmp/` 路径
- 三个 sub agent 模板（think/learn/express）通过 Go embed 编译进二进制，运行时写出到系统 tmp，路径注入主控 prompt
- 自动创建 `.rick/RFC/` 目录（MkdirAll，幂等）
- 复用 `callClaudeCodeCLI`（plan.go 中定义，同包内共享，不重复声明）
- 会话结束后 defer 清理所有 tmp 文件（主 prompt + 三个 sub agent）
- 验证 human-loop dry-run 输出：`python3 tools/check_prompt_variables.py --phase human-loop --topic '测试主题' --keywords human_loop_think`

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
- 输出包含所有注入内容：okr_content、task_md_content、debug 记录、act_path_content 等
