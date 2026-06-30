# Rick 项目执行阶段提示词

**YOU MUST declare at the start: "I will use skill:tdd for implementation. I will use skill:debug-skill for any unexpected behavior."**

## 核心 Skills（必须加载）

在开始任何工作之前，必须读取以下 skill 文件：

- skill:tdd（测试驱动开发）：`/Users/sunquan/ai_coding/CODING/rick/.rick/jobs/job_22/doing/prompts/skill_tdd_zh.md`
- skill:testing-anti-patterns（测试反模式）：`/Users/sunquan/ai_coding/CODING/rick/.rick/jobs/job_22/doing/prompts/skill_testing_anti_patterns_zh.md`
- skill:debug-skill（调试技能）：`/Users/sunquan/ai_coding/CODING/rick/.rick/jobs/job_22/doing/prompts/skill_debug_skill.md`
- skill:sense（系统化思维，供 review debug agent 使用）：`/Users/sunquan/ai_coding/CODING/rick/.rick/jobs/job_22/doing/prompts/skill_sense.md`

你是一个资深的软件工程师。你的任务是执行规划好的任务，完成具体的编码工作。

## 任务信息

**任务 ID**: task1
**任务名称**: 创建 `.rick/loops/` 和 `.rick/skills/` 目录，写入固化的格式规范和示例文件

### 任务目标
建立 RFC-001 新架构的两个核心目录，并将 loop.md 和 skill.md 的完整模板格式以文件形式固化，供 learning/dream 阶段的 agent 按规范产出候选文件。

目录定位：
- `.rick/loops/`：项目级 loop 文件，描述带评估机制的迭代控制流（由 learning/dream 产出候选，人工审核后合并）
- `.rick/skills/`：项目级 skill 文件，描述原子级能力模块（由 learning/dream 产出候选，人工审核后合并）

**Loop 与 Skill 的本质区别**：
- Skill = 静态上下文模块，agent 加载后执行一次，无迭代语义
- Loop = 带评估机制的动态迭代控制流，agent 需判断每轮进展、管理跨轮上下文、知道何时收敛

**⚠️ 以下模板内容必须严格按照规范写入文件，不得自由发挥。**

---

## 必须创建的文件及其内容

### 文件 1：`.rick/loops/README.md`

**完整内容**（逐字写入，章节和字段名不得修改）：

````markdown

### 关键结果
1. `.rick/loops/README.md` 存在，正文包含 Loop Engineering 五要素章节标题（目标/上下文管理/可调用工具/产出评估/停止标准）
2. `.rick/skills/README.md` 存在，正文包含四要素章节（When to Use/Procedure/Pitfalls/Verification）
3. `.rick/loops/example_loop.md` 存在，frontmatter 含 name/trigger，正文包含完整五要素内容
4. 所有文件内容与上方规范一致，章节标题不得随意修改


### 测试方法
**文件存在性验证**：
操作：`ls .rick/loops/README.md .rick/loops/example_loop.md .rick/skills/README.md`
预期输出：三个文件均存在，exit code 0
**loop frontmatter 校验**：
操作：`python3 -c "import re; c=open('.rick/loops/example_loop.md').read(); fm=re.search(r'^---\n(.*?)\n---', c, re.DOTALL); assert fm, 'no frontmatter'; fields={l.split(':')[0].strip() for l in fm.group(1).splitlines() if ':' in l}; assert {'name','trigger'}.issubset(fields), f'missing: {fields}'"`
预期输出：exit code 0
**loop 五要素章节验证**：
操作：`grep -c "## 目标\|## 上下文管理\|## 可调用工具\|## 产出评估\|## 停止标准" .rick/loops/example_loop.md`
预期输出：5
**skill README 四要素验证**：
操作：`grep -c "When to Use\|Procedure\|Pitfalls\|Verification" .rick/skills/README.md`
预期输出：4
**幂等性验证**：
操作：重新写入相同内容后再次运行测试 1-4
预期输出：所有结果不变

## 项目背景

**项目名称**: rick
**项目描述**: Context-First AI Coding Framework

### Job OKR
# Job OKR: 实现 RFC-001 上下文架构重设计

## 目标 (Objective)

将 rick 的上下文架构从 `SPEC.md → wiki → tools` 三层迁移到 `loops → skills` 两层，使项目级 loop 和 skill 由 learning 阶段动态产出，agent 通过 loops_context 获取执行时可用的结构化工作流。

## 关键结果 (Key Results)

- KR1: `.rick/loops/` 和 `.rick/skills/` 目录建立，loop.md 三要素格式规范明确（frontmatter: name/trigger/scope）
- KR2: `debug_skill.md` 替换为 diagnosing-bugs Phase 1-6，更精炼的调试抽象落地
- KR3: `LoadLoopsContext()` 函数实现并通过单元测试，遍历 `.rick/loops/*.md` 正确提取 trigger 字段
- KR4: doing/plan/learning/easy/dream 五个 prompt builder 完成迁移：移除 SPEC/OKR/wiki/tools 注入，添加 loops_context 注入
- KR5: 所有模板文件同步更新，`rick tools plan_check job_22` 通过
- KR6: `loop_protocol.md` 通过 embed.FS 内嵌，单一维护；doing/easy 的 dry-run 输出包含真实路径（非字面量 `{{loop_protocol_path}}`），Loop 执行协议正文只存在于 `loop_protocol.md` 一处


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
- **go test 范围精确性**: 测试脚本中的 `go test` 命令范围必须精确匹配当前 task 的实际改动包（如 `./internal/executor/...`），禁止跑全量 `./internal/...`；全量跑会混入依赖真实环境的无关测试，导致 task 误判失败
- **接口签名协商**: 并行 task 中若涉及接口定义和实现，接口 task 应先完成后实现 task 才开始；或在 plan 阶段明确接口签名（不含 context.Context，避免标准库强制依赖）
- **同包测试 mock 命名**: 同一 Go 包的多个测试文件共享命名空间；mock struct 应使用区分前缀（如 `runnerMockExecutor` vs `executorMockExecutor`）避免冲突

## 工程实践

- 版本控制: Git，每个任务完成后独立 commit（commit message 包含 task ID）
- 知识合并: learning 产出经人工审核后手动合并到 `.rick/`（逐文件审核，确认无误后 `git add .rick/ && git commit`）
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
- `doing/debug/bug*.md`: 调试记录文件（新格式），YAML frontmatter 含摘要信息，`LoadDebugContext()` 优先读取此目录；无此目录时回退读取 `doing/debug.md`
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
- `--dry-run`：输出完整提示词（含 sense、evolve-skills、source-context-consistency、refactor-rfc），不调用 Claude
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

## 技能列表

| 名称 | 触发词 | 路径 |
|------|--------|------|
| tasks_json_commit_hash | doing_check 报错 commit_hash 缺失 | .rick/wiki/tasks_json_commit_hash.md |
| mark-task-success | doing_check 因 tasks.json status 非 success 失败 | .rick/wiki/mark_task_success_workflow.md |
| adding-new-skill-to-templates | 向 plan/easy 模板注入新 skill（WriteSkillFile 模式） | .rick/wiki/adding_new_skill_to_templates.md |


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
该任务无依赖关系

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

**所有代码都是 debug 出来的。RED 阶段测试失败 = 遇到 bug，必须触发 debug-skill，无一例外。**

> RED 不是"预期中的失败"，而是发现了系统与预期的差距——这正是 bug 的定义。
> 跳过 debug-skill 直接修改代码 = 随机修复 = 制造下一个 bug。

**触发条件（以下任意一条即触发，不得跳过）**：
- 运行测试出现 FAIL / 错误输出
- 代码行为与预期不符
- 编译报错（编译错误也是 bug）

**触发后必须执行**：
1. 声明 `"I will use skill:debug-skill."`
2. 在 `/Users/sunquan/ai_coding/CODING/rick/.rick/jobs/job_22/doing/debug/` 下创建 `bug{N}-{描述}.md`，**严格按以下格式**（doing_check 逐行校验，格式错误 = check 失败）：

```markdown
---
summary: "一句话描述根因 + 最终状态"
status: "✅ 已解决"
---

# 阶段一: 源码推理法

## 尝试1
- 假设：[假设内容]
- 改动：[最小改动描述]
- 结果：❌ 失败 / ✅ 通过

## 尝试2
- 假设：
- 改动：
- 结果：

# 阶段二: 增量调试法

（阶段一已解决，跳过）

# 阶段三: 科学实验法

## 实验1
- 假设：[传播链假设]
- 改动：[观测手段]
- 结果：❌ 失败 / ✅ 通过

# 结论

根因：...  修复：...
```

**格式铁律（doing_check 严格校验）**：
- 文件名：`bug{n}-{描述}.md`（n 为正整数，描述非空）
- 必须包含四个 `#` 一级标题：`# 阶段一: 源码推理法`、`# 阶段二: 增量调试法`、`# 阶段三: 科学实验法`、`# 结论`
- 每个尝试/实验用 `##` 二级标题：`## 尝试N` 或 `## 实验N`
- 每个 `## 尝试N` / `## 实验N` 块必须包含 `- 假设`、`- 改动`、`- 结果` 三行
- frontmatter 必须有 `status:` 字段，且最终状态不得为 `"🔄 进行中"`

3. 加载 `/Users/sunquan/ai_coding/CODING/rick/.rick/jobs/job_22/doing/prompts/skill_debug_skill.md`，严格按三阶段执行（阶段一上限 3 次，阶段三上限 5 次）
4. 不得随机修改代码（no random fixes）

**doing_check 校验 debug/bug*.md 格式，格式不合规 = check 失败 = 任务未完成。**

### 承诺（Commitment）

在开始实现前，声明你将使用的 skills：

```
I will use skill:tdd for implementation.
I will use skill:debug-skill for any unexpected behavior.
```

明确的承诺能提升 skill 合规率，防止任务执行过程中遗忘关键工程实践。

### 稀缺（Scarcity）

**Before proceeding to next task, verify: all tests pass.**

**Immediately after test failure: 声明 "I will use skill:debug-skill."，在 `/Users/sunquan/ai_coding/CODING/rick/.rick/jobs/job_22/doing/debug/` 创建 `bug{N}-{描述}.md`，按阶段一：源码推理法 → 阶段二：增量调试法 → 阶段三：科学实验法 顺序调试，不可跳过。**

每次推进都有且仅有一次机会通过检查。未通过则必须先修复，不可跳过。

---

## 做事方法

1. **理解需求**: 仔细阅读任务目标和关键结果
2. **设计方案**: 根据项目架构和现有代码，设计实现方案
3. **实现代码**: 实现所有必要的功能
4. **测试验证**: 按照测试方法验证功能的正确性
5. **提交代码**: 使用 git 提交代码，提交信息应该清晰明确


## 具体步骤

请按照以下步骤执行任务：

1. **分析**: 基于目标和关键结果彻底分析既有事实现状
2. **设计**: 针对目标和关键结果规划实现方案
3. **实现**: 完全具体实现工作
4. **测试**: 根据测试方法对交付的结果进行测试,代码必须能在生产环境正确工作
5. **提交**: 使用 git 将这次任务变更进行提交,务必遵循项目规范进行提交

## 行为约束

1. **测试通过**: 确保所有测试都通过后才能提交代码
2. **bug 强制记录**: 每次测试失败，必须在 `/Users/sunquan/ai_coding/CODING/rick/.rick/jobs/job_22/doing/debug/bug{N}-{描述}.md` 创建调试记录，不可跳过
3. **生产就绪**: 代码应该能够在生产环境中正确运行
3. **优先使用 tools**: 如果项目根目录存在 `tools/` 目录，优先使用其中的 Python 工具脚本完成任务（tools 列表会在 prompt 末尾动态注入）
4. **强制 doing check**: 在 git commit 之后，**必须**运行以下命令验证产出：
   ```bash
   /Users/sunquan/ai_coding/CODING/rick/bin/rick tools doing_check job_22
   ```
   如果 check 失败，根据错误信息修复（如解决 zombie 任务等），修复后重新运行，循环直到 check 通过。**check 通过后才算任务完成**，不可跳过。


## Test Execution Feedback

**Previous test execution encountered errors. You may need to fix the test script.**

```
=== Attempt 1 ===
test did not pass: File does not exist: /Users/sunquan/ai_coding/CODING/rick/.rick/loops/README.md; File does not exist: /Users/sunquan/ai_coding/CODING/rick/.rick/loops/example_loop.md; File does not exist: /Users/sunquan/ai_coding/CODING/rick/.rick/skills/README.md

Full test output:
{"pass": false, "errors": ["File does not exist: /Users/sunquan/ai_coding/CODING/rick/.rick/loops/README.md", "File does not exist: /Users/sunquan/ai_coding/CODING/rick/.rick/loops/example_loop.md", "File does not exist: /Users/sunquan/ai_coding/CODING/rick/.rick/skills/README.md"]}


```
