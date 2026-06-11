# Rick 项目执行阶段提示词

**YOU MUST declare at the start: "I will use skill:tdd for implementation. I will use skill:super-debugging for any unexpected behavior."**

## 核心 Skills（必须加载）

在开始任何工作之前，必须读取以下 skill 文件：

- skill:tdd（测试驱动开发）：`/Users/sunquan/ai_coding/CODING/rick/.rick/jobs/job_16/doing/prompts/skill_tdd_zh.md`
- skill:testing-anti-patterns（测试反模式）：`/Users/sunquan/ai_coding/CODING/rick/.rick/jobs/job_16/doing/prompts/skill_testing_anti_patterns_zh.md`
- skill:super-debugging（超级调试框架）：`/Users/sunquan/ai_coding/CODING/rick/.rick/jobs/job_16/doing/prompts/skill_super_debugging_zh.md`

你是一个资深的软件工程师。你的任务是执行规划好的任务，完成具体的编码工作。

## 任务信息

**任务 ID**: task4
**任务名称**: 全局替换 debug.md 读取：7 处上下文加载改为 debug/ 优先、回退 debug.md 的兼容策略

### 任务目标
项目中有 7 处生产代码读取 `debug.md` 作为上下文注入。统一改为新策略：**优先**扫描 `debug/bug*.md`（只读 frontmatter 中的 summary+status，不加载全文）；若 debug/ 为空或不存在，则**回退**读取 `debug.md`（旧格式，保障历史上下文不丢失）。回退逻辑用 TODO 注释标记，2026-08 后重构时删除。

## 7 处变更位置（执行前逐一阅读实际代码确认行号）

| # | 文件 | 大致位置 | 变更说明 |
|---|------|---------|---------|
| 1 | `internal/executor/retry.go` | `loadDebugContext()` 函数体 | 函数签名不变，内部逻辑改为：`return LoadDebugContext(filepath.Dir(debugFile))`（`filepath.Dir(debugFile)` = workspaceDir，因为 debugFile = `workspaceDir/debug.md`）；原有的 `os.ReadFile(debugFile)` 逻辑全部删除，由 `LoadDebugContext` 统一处理 |
| 2 | `internal/executor/runner.go` | `DebugContent:` 赋值处（TestGenContext） | 替换为 `LoadDebugContext(tr.config.WorkspaceDir)` |
| 3 | `internal/executor/runner.go` | `contextMgr.LoadDebugFromFile(debugMdPath)` 处 | 用 `builder.SetVariable("debug_context", LoadDebugContext(tr.config.WorkspaceDir))` 替换 `contextMgr.LoadDebugFromFile` + 后续的 `formatDebugContext` 调用；`doing.md` 模板无需改动 |
| 4 | `internal/cmd/learning.go` | L102-103（主路径，使用 `jobDir`） | 统一改为 `data.DebugContent = executor.LoadDebugContext(doingDir)`（`doingDir` 在 learning.go 中已定义为 `filepath.Join(jobDir, "doing")`，不得再用 `filepath.Join(jobDir, "doing")` 手动拼接） |
| 5 | `internal/cmd/learning.go` | L164-171（主执行 + dry-run，使用 `doingDir`） | `data.DebugContent = executor.LoadDebugContext(doingDir)`，删除原 `os.ReadFile(debugPath)` 逻辑 |
| 6 | `internal/prompt/easy_prompt.go` | `debugContent := readFileOrDefault(...)` 处 | 替换为 `debugContent := executor.LoadDebugContext(doingDir)`；**注意**：easy 模式下 `doingDir` 可能不存在，`LoadDebugContext` 内部需容错（目录不存在时返回空字符串，不 panic） |
| 7 | `internal/prompt/easy_prompt.go` | `buildEasyLearningPrompt` 的"数据来源"和"Step 1"文字 | 更新"读取 debug.md"的描述为"优先读取 debug/ 下的 bug*.md 摘要，若无则读取 debug.md" |

### 关键结果
1. **新增文件 `internal/executor/debug_dir.go`**，包含三个函数：
2. `extractBugFrontmatter(content string) (summary, status string)`（私有）：解析文件首段 YAML frontmatter（两个 `---` 之间），提取 `summary:` 和 `status:` 字段值（去掉引号和多余空格）；frontmatter 缺失或字段不存在时返回空字符串
3. `LoadDebugDirSummaries(workspaceDir string) string`（**导出**）：扫描 `{workspaceDir}/debug/`，按字典序读取所有 `bug*.md`，对每个文件调用 `extractBugFrontmatter`，返回格式：
4. `LoadDebugContext(workspaceDir string) string`（**导出**，统一入口，所有调用方使用此函数）：
5. `workspaceDir` 为空或目录不存在时均返回空字符串，不 panic（easy 模式下 doingDir 可能尚未创建）
6. **上述 7 处全部替换**（见任务目标表格），每处仅保留一次 `LoadDebugContext(...)` 调用，不再直接读 debug.md
7. **更新受影响的测试**：
8. 新增 `TestExtractBugFrontmatter`：正常 frontmatter、缺失 frontmatter、字段缺失三种情况
9. 新增 `TestLoadDebugDirSummaries`：bug*.md 被读取、非 bug*.md 被跳过、目录不存在返回空
10. 新增 `TestLoadDebugContext_WithDebugDir`：debug/ 有内容时返回摘要（不回退）
11. 新增 `TestLoadDebugContext_Fallback`：debug/ 为空时回退读取 debug.md，返回其内容
12. `executor/retry_test.go` — `TestLoadDebugContext`：在 tmpDir 下创建 `debug/bug1-test.md`（含 frontmatter），验证返回摘要而非全文
13. `executor/runner_test.go` — `TestGenerateDoingPromptFile_WithDebugContext`：在 workspace 创建 `debug/bug1-test.md`，验证 prompt 含摘要不含 bug 正文；删除旧的 debug.md 写入逻辑（或保留作为回退测试）
14. `cmd/learning_test.go`：`DebugContent` 断言改为兼容 debug/ 摘要格式
15. `go build ./...` 无编译错误
16. `go test ./internal/executor/... ./internal/cmd/... ./internal/prompt/...` 全部通过


### 测试方法
预期：LoadDebugContext ≥ 5 次，无残留 ✅
预期：✅
预期：无 FAIL 行
预期：✅ TODO 注释存在

## 项目背景

**项目名称**: rick
**项目描述**: Context-First AI Coding Framework

### Job OKR
# Job OKR: 实现 RFC-debugging，建立三阶段科学调试体系

## 目标 (Objective)
将 Rick 的调试能力从"盲目重试"升级为基于状态机理论的科学调试——三阶段 SOP（源码推理→增量调试→科学实验）+ review debug agent + 运行时工具指引，消除调试上下文的恶性循环。

## 关键结果 (Key Results)
- KR1: `internal/prompt/templates/skills/debug_skill.md` 存在，包含准备阶段、三阶段 SOP（含回滚约束、循环上限）、review debug agent 协议（两个触发点）、运行时观察工具指引、debug/ 目录文件格式
- KR2: `super-debugging-zh.md` 已删除；`doing.md` 和 `plan.md` 模板中所有 `super_debugging*` 引用替换为 `debug_skill_path`；doing.md 的 debug{N} 调试记录格式替换为 debug_skill 加载指令
- KR3: `doing_prompt.go`、`plan_prompt.go`、`easy_prompt.go` 的 WriteSkillFile/SetVariable 调用全部从 "super-debugging-zh"/"super_debugging_path"/"super_debugging_skill_path" 切换到 "debug_skill"/"debug_skill_path"；`go test ./internal/prompt/...` 全部通过
- KR4: `internal/executor/runner.go` 的重试上下文加载逻辑从仅读 `debug.md` 扩展为同时扫描 `debug/` 目录下所有 `bug*.md` 文件；`go test ./internal/executor/...` 全部通过


### 项目 SPEC
# SPEC

## 技术栈

- 语言: Go 1.21+（主程序），Python 3.8+（tools 脚本和测试脚本）
- 框架: Cobra（CLI 命令框架），Goldmark（Markdown 解析）
- 测试: Go testing 标准库，Python unittest，Bash integration tests
- 其他: Git（版本管理），Claude Code CLI（AI agent 集成）

## 架构设计

- 架构风格: 命令行工具，模块化分层架构（cmd → executor → prompt/workspace/git）
- 模块划分: cmd（命令处理）/ executor（任务执行引擎）/ prompt（提示词管理）/ workspace（路径管理）/ parser（内容解析）/ git（Git 操作）/ callcli（Claude 集成）/ agent（接口契约）/ actpath（act-path 生成）
- 工具链模块: `rick tools` 子命令体系，plan_check/doing_check/learning_check/dream_check 四个子命令
- 接口设计: check 命令统一输出格式（✅/❌ + 描述），exit code 0=pass / 1=fail
- human-loop 模块: `rick human-loop <topic>` 命令，通过 SENSE 方法论模板引导 Claude 对复杂主题进行深度分析，产出存入 `.rick/RFC/` 目录；三个 sub agent 模板通过 Go embed 编译进二进制，运行时写出到 tmp 文件，路径注入主控 prompt
- tools 模块: `.rick/tools/*.py` 存放确定性工具脚本，agent 通过 `python3 .rick/tools/<file>.py` 调用
- **agent 接口模块** (`internal/agent/`): 定义 `AgentSession` / `AgentExecutor` 接口契约和 `ToolCall` struct；`claudecode` 子包为唯一实现，只在 `doing.go` 组合根中实例化
- **act-path 生成模块** (`internal/actpath/`): `Generate(session AgentSession, outputFile string) error`，不 import 任何具体 agent 实现，输出含执行摘要/行为轨迹/Agent 最终输出三节
- **DIP 组合根模式**: `doing.go` 是唯一 import `internal/agent/claudecode` 的地方；runner/executor/actpath 仅依赖 `internal/agent` 接口，保证可单元测试；验证: `grep -r "claudecode" internal/executor/ internal/actpath/` 应为空
- **dream 模块**: `internal/cmd/dream.go` 实现 `rick dream` 命令，不生成 act-path，自动扫描 `.rick/jobs/*/doing/tasks.json` 发现已完成 jobs、与 `dream_run_*_log.md` 对比得出待处理列表；支持 `--background`/`-p` 背景模式（`--dangerously-skip-permissions`），限制修改范围为 `.rick/wiki/`/`.rick/tools/`/`.rick/SPEC.md`

## 开发规范

- 代码风格: Go 标准格式（gofmt），函数命名 camelCase，导出函数 PascalCase
- check 命令规范: 默认只报告问题，`--auto-fix` 标志才触发 Claude 修复，保持确定性
- **三层上下文结构**（`.rick/` 内部）:
  - `SPEC.md`：规范与约束，agent 上下文的唯一入口
  - `wiki/*.md`：系统原理文档 + 技能说明书，供人类阅读和 dream 阶段参考
  - `.rick/tools/*.py`：确定性工具脚本，原子化，单一职责，JSON 输出（`{"pass": bool, "errors": [...]}`），文件首行必须有 `# Description:` 注释，调用方式 `python3 .rick/tools/<file>.py`
- Tools 脚本规范: Python 文件，argparse 解析参数，JSON 输出结果（`{"pass": bool, "errors": [...]}`）
- 测试要求: 单元测试覆盖核心逻辑，集成测试覆盖 CLI 命令，mock_agent 替代真实 Claude 调用
- **Mock Agent 同步要求**: `tests/mock_agent/mock_agent.py` 和 `.rick/tools/mock_agent_testing.py` 的 mock 输出格式必须与 doing_check/learning_check 期望严格对齐；当 check 命令格式规范变更时，两个 mock_agent 文件需同步更新
- 路径规范: 测试脚本位于 `.rick/jobs/job_N/doing/tests/`，需要 6 次 dirname 到达项目根目录
- **测试脚本 binary 规范**: 测试脚本调用 rick 命令验证新实现的功能时，必须先调用 `.rick/tools/build_and_get_rick_bin.py` 构建本地 `./bin/rick` 并使用返回的 `bin_path`，不得直接调用系统安装版（系统版不含当前任务的新代码）
- **Cobra flag 定义规范**: 全局 flag（跨命令共享，如 `--job`、`--dry-run`）用 `rootCmd.PersistentFlags()`，在 `root.go` 定义；命令级 flag 用 `cmd.Flags()`，在对应命令文件定义；全局 flag 通过 `GetXxx()` 函数统一暴露
- Go variadic 改造模式: 当需要让现有必传参数变为可选时，使用 variadic（`...T`）而非新增无参构造函数，保持接口唯一性；调用方无需修改
- 包内函数共享: 同一 Go 包内的函数（如 `callClaudeCodeCLI`）可在多个文件中直接调用，不需要重新声明或导出
- Dry-run 规范: `--dry-run` 标志必须输出完整的 prompt 内容（而非占位消息），便于调试和验证上下文注入效果
- **测试断言精确性**: dry-run 输出包含大量上下文文本，断言需先定位 section（如 `## 可用的项目 Skills`）再检查内容，避免全文搜索误判
- **task.md 测试方法精确性**: task.md 中"测试方法"描述的命令行调用必须基于工具**实际存在的参数接口**，不得引用尚未实现的参数。plan 阶段生成测试脚本前应验证 `.rick/tools/` 下对应工具的 `--help` 输出
- **embed.FS 目录嵌入**: `//go:embed dir`（目录）必须绑定 `embed.FS` 类型；`//go:embed file`（单文件）可绑定 `string`；两者可在同一文件共存。`_ "embed"` 改为 `"embed"` 才能使用 `embed.FS`
- **JSON 输出编码约定**: 所有 Python 工具/测试脚本的 `json.dumps()` 调用必须加 `ensure_ascii=False`，避免中文字符被转义为 `\uXXXX` 导致字符串匹配失败
- **接口签名协商**: 并行 task 中若涉及接口定义和实现，接口 task 应先完成后实现 task 才开始；或在 plan 阶段明确接口签名（不含 context.Context，避免标准库强制依赖）
- **同包测试 mock 命名**: 同一 Go 包的多个测试文件共享命名空间；mock struct 应使用区分前缀（如 `runnerMockExecutor` vs `executorMockExecutor`）避免冲突

## 工程实践

- 版本控制: Git，每个任务完成后独立 commit（commit message 包含 task ID）
- 知识合并: learning 产出经人工审核后手动 `git merge --no-ff`（`rick tools merge` 命令尚未实现，见 RFC-005）
- 持续集成: `go test ./...` 覆盖单元测试，`bash tests/tools_integration_test.sh` 覆盖集成测试
- 发布流程: `./scripts/build.sh` 构建，`./scripts/install.sh` 安装到 `~/.rick/bin/rick`

## 路径约定

- `.rick/RFC/`: human-loop 会话产出文档目录，由 `GetRFCDir()` 管理，`rick human-loop` 执行时自动创建
- `.rick/jobs/job_N/`: 每次 job 的工作目录，包含 plan/doing/learning 三个子目录
- `.rick/jobs/job_N/plan/OKR.md`: job 级 OKR，由 plan 阶段 Claude 生成，doing/learning 阶段读取
- `.rick/wiki/`: 系统原理文档 + 技能说明书（`.md`），供人类阅读和 dream 阶段参考；`wiki/README.md` 为所有文档索引
- `.rick/dream/`: dream 目录，存放 `dream_run_*_log.md` 和 `prompts/`；待处理 jobs 由程序自动扫描 tasks.json 发现，无需手工维护索引文件
- `.rick/dream/run_log_{n}.md`: learning 阶段 Step 6 写入的度量文件，格式 `| Job | 模型 | 错误次数 | 工具调用轮次 | 备注 |`
- `.rick/tools/`: 确定性 Python 工具脚本（**只含 `.py` 文件**）；每个脚本首行必须有 `# Description:` 注释；调用方式 `python3 .rick/tools/<file>.py`
- `doing/tasks/{taskID}/act-path.md`: 任务执行后自动生成的行为轨迹文件，含工具调用、报错次数、执行时长
- `doing/tasks/{taskID}/raw_session.log`: Claude Code NDJSON 原始流式输出，每行一个 JSON 对象（非 JSON 行也写入）

## 命令规范

### rick doing（DIP 全链路）

- `doing.go` 是唯一 import `internal/agent/claudecode` 的地方（**组合根**）
- `runner.go` 和 `executor.go` 只依赖 `internal/agent` 接口，不 import claudecode
- `actpath.Generate(session, outputFile)` 在每个 task 的 `agentExecutor.Execute` 完成后调用
- session 为 nil 时跳过 act-path 生成（nil guard），不 panic

### rick dream

- 自动扫描 `.rick/jobs/*/doing/tasks.json` 发现所有 tasks 均 "success" 的 jobs（已完成）
- 对比 `.rick/dream/dream_run_*_log.md` 排除已处理 jobs，取最多 5 个待处理 jobs
- `--job_num <n>`：调整每次处理的 job 数量（默认 5）
- `--background`/`-p`：背景模式，使用 `--dangerously-skip-permissions` 非交互执行
- `--dry-run`：输出完整提示词（含 sense + evolve-skills core-skills），不调用 Claude
- **变更约束**: 仅允许修改 `.rick/wiki/`、`.rick/tools/`、`.rick/SPEC.md`，严禁修改业务代码

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
- 验证 human-loop dry-run 输出：`python3 .rick/tools/check_prompt_variables.py --phase human-loop --topic '测试主题' --keywords human_loop_think`

### rick plan --job

- `--job <job_id>` 为全局 flag（定义在 root.go），plan.go 通过 `GetJobID()` 读取，不在 plan.go 中重复定义
- 指定 `--job` 时跳过 `NextJobID()`，直接复用已有 job 的 plan 目录
- plan 目录不存在时返回明确错误，不自动创建

### rick plan --dry-run

- 生成完整 plan prompt 并打印到 stdout（通过 `runPlanDryRun()` 函数）
- 不调用 Claude，不创建任何文件
- 输出包含所有注入内容：job_plan_dir、SPEC 路径等

### rick doing --dry-run

- 打印完整 doing prompt 内容到 stdout
- 不调用 Claude，不执行任何任务
- 展示第一个非 success 状态的任务（从 tasks.json 读取，不硬编码 task1）

### rick learning --dry-run

- 生成完整 learning prompt 并打印到 stdout（通过 `runLearningDryRun()` 函数）
- 不调用 Claude，不创建任何文件
- 输出包含所有注入内容：okr_content、task_md_content、debug 记录、act_path_content 等

### rick ctrl

- `--job <job_id>` 为必传参数，无默认值
- 调用 `GenerateCtrlPromptFile(jobID, rickDir)` 生成 prompt，写入 `doing/prompts/ctrl_prompt.md`，返回路径
- `callClaudeCodeCLI(cfg, promptFile)` 启动交互式 Claude 会话（与 plan/human-loop 共用同一函数）
- ctrl 与 doing 之间**仅通过文件通信**：reading tasks.json + raw_session_coding.log，writing tasks.json + plan/task<N>.md
- 场景 A（追加指令）：在 `plan/task<N>.md` 末尾追加 `## 干预指令 (Intervention)` 章节写入人类指令，通常同时执行场景 B
- 场景 B（重置 task）：将 `status` 改为 `"pending"`，清空 `error` 字段，更新 `updated_at`；若目标 task 正在运行（`running`），直接重置无效，需告知人类先 Ctrl+C 停止 doing
- **变更约束**：只能修改 `doing/` 和 `plan/` 下的文件，不得修改 `.rick/` 其他目录
- dry-run 输出完整 prompt（通过 `runCtrlDryRun()`），需指定 `--job` 否则报错退出


### 项目架构
Rick 项目采用模块化架构设计：

**核心模块**:
- infrastructure: 基础设施模块（Go 项目初始化、CLI、工作空间、配置、日志）
- parser: 内容解析模块（Markdown、task.md、debug.md、OKR/SPEC 解析）
- dag_executor: DAG 执行模块（DAG 构建、拓扑排序、任务执行、重试机制）
- prompt_manager: 提示词管理模块（模板、构建、上下文、各阶段提示词生成）
- cli_commands: 命令处理模块（init、plan、doing、learning 命令）

**关键设计**:
- 使用 Go 标准库为主，最小化外部依赖
- 提示词管理是核心创新，支持多阶段提示词生成
- 任务执行采用 DAG 拓扑排序，支持并行和串行执行
- 失败重试机制，超过限制后需人工干预

## 执行上下文

### 已完成的任务
暂无已完成的任务

### 任务依赖
该任务依赖以下任务的完成：
- task3


### 问题记录

以下是执行过程中遇到的问题记录，请重点关注避免重复错误：

暂无问题记录

## Cialdini 合规原则

### 权威（Authority）

**YOU MUST follow TDD. No exceptions.**

在开始任何实现之前，必须先编写失败的测试（RED phase）。这是不可协商的工程规范。

#### TDD 铁律（Three Laws）

1. **RED（先红）**: 先运行测试，确认测试失败（证明测试有效）
2. **GREEN（再绿）**: 编写最少代码让测试通过
3. **REFACTOR（再重构）**: 在测试通过的前提下改善代码质量

**不得跳过任何阶段。** 未经 RED 验证直接写实现，视为违反 TDD 铁律。

#### DEBUG 铁律

**When encountering ANY bug, YOU MUST declare: "I will use skill:super-debugging." No random fixes. No exceptions.**

遇到任何不符合预期的行为时，必须：
1. 声明 `"I will use skill:super-debugging."`
2. 走 super-debugging 五阶段流程：S（还原问题）→ E（视角分析）→ N（验证假设）→ 修复 → 3 次失败则找人类
3. 不得随机修改代码（no random fixes）

### 承诺（Commitment）

在开始实现前，声明你将使用的 skills：

```
I will use skill:tdd for implementation.
I will use skill:super-debugging for any unexpected behavior.
```

明确的承诺能提升 skill 合规率，防止任务执行过程中遗忘关键工程实践。

### 稀缺（Scarcity）

**Before proceeding to next task, verify: all tests pass.**

**Immediately after test failure, run super-debugging Phase 1 (S：还原问题).**

每次推进都有且仅有一次机会通过检查。未通过则必须先修复，不可跳过。

---

## 做事方法

1. **理解需求**: 仔细阅读任务目标和关键结果
2. **设计方案**: 根据项目架构和现有代码，设计实现方案
3. **实现代码**: 实现所有必要的功能
4. **测试验证**: 按照测试方法验证功能的正确性
5. **记录工作日志**: 在 git commit 之前，**必须**更新 debug.md（强制要求，非可选）
6. **提交代码**: 使用 git 提交代码，提交信息应该清晰明确


## 具体步骤

请按照以下步骤执行任务：

1. **分析**: 基于目标和关键结果彻底分析既有事实现状
2. **设计**: 针对目标和关键结果规划实现方案
3. **实现**: 完全具体实现工作
4. **测试**: 根据测试方法对交付的结果进行测试,代码必须能在生产环境正确工作
5. **记录**: **在 git commit 之前必须先更新 debug.md**（强制，详见下方"工作日志规范"）
6. **提交**: 使用 git 将这次任务变更进行提交,务必遵循项目规范进行提交

## 工作日志规范

**debug.md 是强制工作日志，无论任务是否顺利，都必须在 git commit 之前记录完整的执行过程。**

这是每次任务执行的硬约束，不可跳过。debug.md 是 learning 阶段提取有价值 skills 的核心数据源。

### debug.md 文件位置
- 路径：`{{doing_dir}}/debug.md`
- 如果文件不存在，请创建它

### 强制记录格式

每次任务执行，使用以下格式追加记录（按顺序递增编号）：

```markdown
## task{N}: {任务名称简述}

**分析过程 (Analysis)**:
- 分析了哪些现有代码/文件
- 发现了哪些关键约束或依赖
- 选择了什么实现方案，为什么

**实现步骤 (Implementation)**:
1. 步骤1：做了什么
2. 步骤2：做了什么
3. ...

**遇到的问题 (Issues)**:
- 无（如果没有遇到任何问题，写"无"）
- 或者列出遇到的问题及解决方法

**验证结果 (Verification)**:
- 测试命令：`{实际运行的测试命令}`
- 测试输出：
  ```
  {粘贴实际测试输出}
  ```
- 结论：✅ 通过 / ❌ 失败
```

### 遇到问题时的详细记录

如果"遇到的问题"不为空，在 debug.md 中**额外追加**以下格式的详细问题记录：

```markdown
## debug{N}: 问题简要描述

**现象 (Phenomenon)**:
- 描述观察到的问题现象
- 包括错误信息、测试失败信息等

**复现 (Reproduction)**:
- 如何复现这个问题
- 相关的操作步骤

**猜想 (Hypothesis)**:
- 对问题原因的分析和猜测
- 可能的根本原因

**验证 (Verification)**:
- 如何验证猜想是否正确
- 进行了哪些验证操作

**修复 (Fix)**:
- 采取的修复措施
- 修改了哪些代码或配置

**进展 (Progress)**:
- 当前状态：✅ 已解决 / 🔄 进行中 / ❌ 未解决
- 如果未解决，说明下一步计划
```

### 示例

```markdown
## task1: 实现用户认证模块

**分析过程 (Analysis)**:
- 阅读了 internal/auth/ 目录下的现有代码
- 发现 JWT 库已在 go.mod 中声明，无需新增依赖
- 选择在现有 middleware.go 中扩展，避免创建新文件

**实现步骤 (Implementation)**:
1. 在 middleware.go 中添加 ValidateToken 函数
2. 修改 router.go 注册认证中间件
3. 更新 config.go 添加 JWT secret 配置项

**遇到的问题 (Issues)**:
- 无

**验证结果 (Verification)**:
- 测试命令：`go test ./internal/auth/... -v`
- 测试输出：
  ```
  --- PASS: TestValidateToken (0.00s)
  --- PASS: TestMiddleware (0.01s)
  PASS
  ok  	project/internal/auth	0.023s
  ```
- 结论：✅ 通过
```

## 行为约束

1. **强制工作日志**: **在 git commit 之前必须先更新 debug.md**，这是硬约束，不可跳过
2. **四个必填部分**: 分析过程、实现步骤、遇到的问题（无则写"无"）、验证结果（含测试命令和实际输出）
3. **测试通过**: 确保所有测试都通过后才能提交代码
4. **生产就绪**: 代码应该能够在生产环境中正确运行
5. **明确阻碍**: 如果无法完成任务，请在 debug.md 中详细记录阻碍因素
6. **优先使用 tools**: 如果项目根目录存在 `tools/` 目录，优先使用其中的 Python 工具脚本完成任务（tools 列表会在 prompt 末尾动态注入）
7. **强制 doing check**: 在 git commit 之后，**必须**运行以下命令验证产出：
   ```bash
   /Users/sunquan/ai_coding/CODING/rick/bin/rick tools doing_check job_16
   ```
   如果 check 失败，根据错误信息修复（如补充 debug.md、解决 zombie 任务等），修复后重新运行，循环直到 check 通过。**check 通过后才算任务完成**，不可跳过。


## Test Execution Feedback

**Previous test execution encountered errors. You may need to fix the test script.**

```

**遇到的问题 (Issues)**:
- 无

**验证结果 (Verification)**:
- 测试命令：`go test ./internal/auth/... -v`
- 测试输出：
  ```
  --- PASS: TestValidateToken (0.00s)
  --- PASS: TestMiddleware (0.01s)
  PASS
  ok  	project/internal/auth	0.023s
  ```
- 结论：✅ 通过
```

## 行为约束

1. **强制工作日志**: **在 git commit 之前必须先更新 debug.md**，这是硬约束，不可跳过
2. **四个必填部分**: 分析过程、实现步骤、遇到的问题（无则写"无"）、验证结果（含测试命令和实际输出）
3. **测试通过**: 确保所有测试都通过后才能提交代码
4. **生产就绪**: 代码应该能够在生产环境中正确运行
5. **明确阻碍**: 如果无法完成任务，请在 debug.md 中详细记录阻碍因素
6. **优先使用 tools**: 如果项目根目录存在 `tools/` 目录，优先使用其中的 Python 工具脚本完成任务（tools 列表会在 prompt 末尾动态注入）
7. **强制 doing check**: 在 git commit 之后，**必须**运行以下命令验证产出：
   ```bash
   rick tools doing_check job_N
   ```
   如果 check 失败，根据错误信息修复（如补充 debug.md、解决 zombie 任务等），修复后重新运行，循环直到 check 通过。**check 通过后才算任务完成**，不可跳过。

✅ plan check passed: 1 tasks, dependencies valid
✅ doing check passed: 1/1 tasks succeeded
✅ learning check passed
✅ Loaded debug context (31 bytes)
✅ Loaded tasks.json (1 tasks)
⚠ OKR.md not found (skipping)
⚠ No task*.md files found in plan directory
✅ Found 0 act-path.md files
✅ Loaded debug context (0 bytes)
✅ Loaded tasks.json (1 tasks)
⚠ OKR.md not found (skipping)
⚠ No task*.md files found in plan directory
✅ Found 0 act-path.md files
Error: accepts 1 arg(s), received 0
Usage:
  plan_check <job_id> [flags]

Flags:
  -h, --help   help for plan_check

Error: accepts 1 arg(s), received 0
Usage:
  doing_check <job_id> [flags]

Flags:
      --auto-fix   Attempt to auto-fix errors using Claude
  -h, --help       help for doing_check

Error: accepts 1 arg(s), received 0
Usage:
  learning_check <job_id> [flags]

Flags:
      --auto-fix   Attempt to auto-fix errors using Claude
  -h, --help       help for learning_check

Invalid API key · Fix external API key
❌ plan check failed: OKR.md has no meaningful content (only stub headers): /private/var/folders/c2/9ln3nxr55z7fqd9jpxscvg9w0000gn/T/TestNewPlanCheckCmd_RunE_WithWorkspace1732783721/001/.rick/jobs/job_test/plan/OKR.md
FAIL	github.com/sunquan/rick/internal/cmd	29.472s
ok  	github.com/sunquan/rick/internal/config	(cached)
ok  	github.com/sunquan/rick/internal/executor	1.362s
ok  	github.com/sunquan/rick/internal/git	(cached)
ok  	github.com/sunquan/rick/internal/logging	(cached)
ok  	github.com/sunquan/rick/internal/parser	(cached)
ok  	github.com/sunquan/rick/internal/prompt	(cached)
ok  	github.com/sunquan/rick/internal/workspace	(cached)
FAIL



```
