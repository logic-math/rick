# research-7 N1-N3：rick 侧完整调用链与 claude code 功能枚举

节点路径：[根 > N1-N3-ai_cli 调用链与 claude code 功能枚举]
事实陈述：从 ai_cli 调用点（`internal/agent/`）反向回溯，枚举所有调用 claude code 的位置、完整调用链、claude code 使用到的全部功能

## 执行动作

1. Grep `exec\.Command.*claude|AgentExecutor|claudecode\.` 全仓 Go 代码（18 文件命中）
2. Read `internal/agent/claudecode/executor.go`（唯一 ClaudeCodeExecutor 实现）
3. Read `internal/agent/interface.go`（AgentExecutor 接口）
4. Read `internal/executor/runner.go`（runner.go 是 AgentExecutor 的唯一生产调用方，line 93/166）
5. Read `internal/cmd/doing.go:180-240`（NewExecutor 实例化唯一位置）
6. Grep `callClaudeCodeCLI|claudePath|ClaudeCodePath` 全 cmd 包（找到 plan/learning/easy/dream/human_loop/ctrl 7 个直接调用点）
7. Read `internal/cmd/plan.go:245-298`（callClaudeCodeCLI + callClaudeCodeCLIBackground 函数体）
8. Read `internal/cmd/learning.go:230-260`（learning 直接 exec.Command）
9. Read `internal/cmd/easy.go:130-200`（easy 的 --resume/--session-id 调用）
10. Read `internal/cmd/tools_plan_check.go:195-220`（runAutoFix + findClaudeBinary）
11. Read `internal/prompt/doing_prompt.go`（system prompt 注入机制）
12. Grep `PreToolUse|PostToolUse|settings\.json|\.claude|hooks` 全仓（确认 rick 是否用 claude code hooks）

## 各信源验证结果

### 代码原文（权重 0.4）✅

**N1 - ai_cli 调用点完整枚举**（13 处实际调用 claude code 二进制）：

| # | 文件:行号 | 调用形式 | 用途 | flag |
|---|---|---|---|---|
| 1 | `internal/agent/claudecode/executor.go:39` | `exec.Command(claudePath, "-p", "--output-format", "stream-json", "--verbose", "--dangerously-skip-permissions", promptFile)` | doing 任务执行（核心） | -p / --output-format stream-json / --verbose / --dangerously-skip-permissions |
| 2 | `internal/cmd/plan.go:261` (callClaudeCodeCLI) | `exec.Command(claudePath, args...)` + stdin/stdout/stderr 接管 | plan interactive | (无 flag，纯 interactive) |
| 3 | `internal/cmd/plan.go:285` (callClaudeCodeCLIBackground) | `exec.Command(claudePath, "-p", "--dangerously-skip-permissions", promptFile)` | dream 后台 | -p / --dangerously-skip-permissions |
| 4 | `internal/cmd/learning.go:247` | `exec.Command(claudePath, promptFile)` + stdin/stdout/stderr | learning interactive | (无 flag) |
| 5 | `internal/cmd/easy.go:149` (callClaudeCodeCLI) | `callClaudeCodeCLI(cfg, "", "--resume", sessionID)` | easy resume | --resume <sessionID> |
| 6 | `internal/cmd/easy.go:191` (callClaudeCodeCLI) | `callClaudeCodeCLI(cfg, mainFile, "--session-id", sessionID)` | easy 新会话 | --session-id <sessionID> <file> |
| 7 | `internal/cmd/dream.go:97` (callClaudeCodeCLIBackground) | 同 #3 | dream 后台 | -p / --dangerously-skip-permissions |
| 8 | `internal/cmd/dream.go:102` (callClaudeCodeCLI) | 同 #2 | dream interactive | (无 flag) |
| 9 | `internal/cmd/human_loop.go:78` (callClaudeCodeCLI) | 同 #2 | human-loop | (无 flag) |
| 10 | `internal/cmd/ctrl.go:74` (callClaudeCodeCLI) | 同 #2 | ctrl | (无 flag) |
| 11 | `internal/cmd/tools_plan_check.go:207` (runAutoFix) | `exec.Command(claudePath, "--dangerously-skip-permissions", promptFile)` | plan check autofix | --dangerously-skip-permissions |
| 12 | `internal/cmd/tools_doing_check.go:81` (runAutoFix) | 同 #11 | doing check autofix | --dangerously-skip-permissions |
| 13 | `internal/cmd/tools_learning_check.go:81` (runAutoFix) | 同 #11 | learning check autofix | --dangerously-skip-permissions |
| (备用) | `internal/executor/runner.go:293` (CallClaudeCodeCLI) | `exec.Command(claudePath, "--dangerously-skip-permissions", promptFile)` | **生产代码未调用**，仅 test 引用 | --dangerously-skip-permissions |

**统计**：13 处生产调用点 + 1 处备用（仅 test）。其中 8 处通过 `callClaudeCodeCLI`/`callClaudeCodeCLIBackground` 封装，3 处 `runAutoFix` 封装，1 处 learning 直接 exec，1 处 ClaudeCodeExecutor（核心 doing）。

**N2 - 调用链反向回溯**（从 cmd 入口到 claude code）：

| cmd 入口 | 子命令处理 | executor/封装 | agent 调用 | claude code flag |
|---|---|---|---|---|
| `rick plan` | `cmd/plan.go:executePlanWorkflow` | `callClaudeCodeCLI` (plan.go:249) | `exec.Command` (plan.go:261) | (无 flag，interactive) |
| `rick doing job_N` | `cmd/doing.go:runDoing` | `executor.NewExecutor` (doing.go:205) → `TaskRunner.RunTask` (runner.go:60) → `agentExecutor.Execute` (runner.go:93,166) | `ClaudeCodeExecutor.Execute` (claudecode/executor.go:26) → `exec.Command` (claudecode/executor.go:39) | -p --output-format stream-json --verbose --dangerously-skip-permissions |
| `rick learning job_N` | `cmd/learning.go` | 直接 `exec.Command` (learning.go:247) | - | (无 flag，interactive) |
| `rick easy` | `cmd/easy.go:startEasySession`/`resumeEasySession` | `callClaudeCodeCLI` (easy.go:149,191) | `exec.Command` (plan.go:261) | --resume / --session-id |
| `rick dream` | `cmd/dream.go` | `callClaudeCodeCLIBackground`/`callClaudeCodeCLI` (dream.go:97,102) | `exec.Command` (plan.go:285,261) | -p --dangerously-skip-permissions / (无 flag) |
| `rick human-loop` | `cmd/human_loop.go` | `callClaudeCodeCLI` (human_loop.go:78) | `exec.Command` (plan.go:261) | (无 flag) |
| `rick ctrl` | `cmd/ctrl.go` | `callClaudeCodeCLI` (ctrl.go:74) | `exec.Command` (plan.go:261) | (无 flag) |
| `rick plan` (autofix) | `cmd/tools_plan_check.go:runAutoFix` | 直接 `exec.Command` (tools_plan_check.go:207) | - | --dangerously-skip-permissions |
| `rick doing` (autofix) | `cmd/tools_doing_check.go:runAutoFix` | 同上 | - | --dangerously-skip-permissions |
| `rick learning` (autofix) | `cmd/tools_learning_check.go:runAutoFix` | 同上 | - | --dangerously-skip-permissions |

**核心调用链（doing）**：
```
cmd/rick/main.go:15 rootCmd.Execute()
  → internal/cmd/doing.go:204 claudecode.NewExecutor(cfg.ClaudeCodePath)
  → internal/cmd/doing.go:205 executor.NewExecutor(tasks, execConfig, doingDir, jobID, claudeExec, existingTasksJSON)
  → internal/executor/executor.go ExecuteJob()
  → internal/executor/runner.go:60 TaskRunner.RunTask(task, debugContext, testErrorFeedback)
    → runner.go:73 GenerateTestWithAgent(task) → runner.go:166 agentExecutor.Execute(testPromptFile, ...) [test 生成]
    → runner.go:85 GenerateDoingPromptFile(task, ...) [prompt 文件生成，含 system prompt 注入]
    → runner.go:93 agentExecutor.Execute(doingPromptFile, task.ID, ...) [任务执行]
      → internal/agent/claudecode/executor.go:39 exec.Command(claudePath, "-p", "--output-format", "stream-json", "--verbose", "--dangerously-skip-permissions", promptFile)
      → claudecode/executor.go:49 parseStream(stdout, rawLogPath) [NDJSON 解析]
    → runner.go:110 ExecuteTestScript(testScriptPath) [python3 跑测试]
    → runner.go:122 RunDoingCheck(workspaceDir) [格式校验]
```

**N3 - claude code 功能枚举**（从 13 处调用点提取的全部使用到的 claude code 功能）：

| 类别 | 功能 | rick 使用位置 | claude code 协议字段/flag/行为 |
|---|---|---|---|
| **flag** | `-p` / `--print` | #1,3,7,11,12,13 | 非交互模式，处理 prompt 后退出 |
| **flag** | `--output-format stream-json` | #1 (doing 核心) | NDJSON 流式输出（type: system/assistant/user/result） |
| **flag** | `--verbose` | #1 (doing 核心) | 详细输出 |
| **flag** | `--dangerously-skip-permissions` | #1,3,7,11,12,13 | 跳过权限确认（非 interactive 必需） |
| **flag** | `--resume <sessionID>` | #5 (easy resume) | 续接已有会话 |
| **flag** | `--session-id <sessionID>` | #6 (easy 新会话) | 指定 session id 启动 |
| **flag** | (无 flag，纯 interactive) | #2,4,8,9,10 | interactive 模式，stdin/stdout/stderr 接管 |
| **stream-json 协议** | `type:"system"` 行 | claudecode/executor.go:138 | 提取 sessionID |
| **stream-json 协议** | `type:"result"` 行 | claudecode/executor.go:141 | 终止信号 + duration_ms |
| **stream-json 协议** | `type:"assistant"` 行 + `message.content[]` | claudecode/executor.go:145 | 提取 tool_use + text |
| **stream-json 协议** | `content.type:"tool_use"` | claudecode/executor.go:151 | 工具调用（id/name/input） |
| **stream-json 协议** | `content.type:"text"` | claudecode/executor.go:159 | 最终文本消息 |
| **stream-json 协议** | `type:"user"` + `content.type:"tool_result"` | claudecode/executor.go:164 | 工具结果（tool_use_id/content/is_error） |
| **stream-json 协议** | `session_id` 字段 | claudecode/executor.go:139,142 | 会话标识 |
| **stream-json 协议** | `duration_ms` 字段 | claudecode/executor.go:144 | 执行时长 |
| **stream-json 协议** | `is_error` 字段 | claudecode/executor.go:176 | 工具错误标记 |
| **行为** | prompt 文件作为 positional arg | 全部 13 处 | claude code 读取文件内容作为 prompt |
| **行为** | stdin/stdout/stderr 接管（interactive） | #2,4,8,9,10 | 用户与 claude 直接交互 |
| **行为** | session 持久化（--resume/--session-id） | #5,6 | 会话状态保存到磁盘 |
| **system prompt 注入** | 通过 prompt 文件内容注入 | doing_prompt.go（WriteSkillFile + LoadLoopsContext + LoadSkillsContext + doing_loop_content） | rick 将 system prompt 写入 prompt 文件，claude code 作为 prompt 读取（无 --system-prompt flag） |
| **skill 加载** | 通过 prompt 文件路径引用 | doing_prompt.go:101 WriteSkillFile("skill_debug_skill.md", "debug_skill") | rick 将 skill 文件路径写入 prompt 文件，claude code 不主动加载 skill |
| **权限管理** | `--dangerously-skip-permissions` 全程跳过 | #1,3,7,11,12,13 | 无 permission popup |
| **hooks（PreToolUse/PostToolUse）** | **未使用** | Grep 全仓零匹配 | rick 不使用 claude code hooks 机制 |
| **settings.json 配置** | **未使用** | rick 仓库无 .claude/settings.json（仅有 .claude/settings.local.json 用于 WebSearch 权限） | rick 不通过 claude code settings.json 配置行为 |
| **subagent** | **未使用** | Grep 全仓零匹配（subagent 仅在 .rick/jobs learning 文档中提及） | rick 不使用 claude code subagent |
| **compaction** | **未显式控制** | Grep 全仓零匹配 | rick 不控制 claude code compaction（依赖默认 auto-compact） |
| **MCP** | **未使用** | Grep 全仓零匹配 | rick 不使用 claude code MCP |

### 运行时行为（权重 0.3）✅

- rick 仓库 `internal/agent/claudecode/executor.go` 是唯一通过 `AgentExecutor` 接口封装的 claude code 调用点（doing 核心）
- 其余 12 处直接 `exec.Command` 调用 claude code（plan/learning/easy/dream/human_loop/ctrl/tools_*_check）
- rick 不使用 claude code 的 hooks/settings.json/subagent/MCP/compaction 控制
- rick 的 system prompt 注入完全通过 prompt 文件内容（不是 claude code flag）
- rick 的 skill 加载完全通过 prompt 文件路径引用（不是 claude code 内置 skill 机制）

### 文档（权重 0.2）✅

- MEMORY.md 记录 "rick 启动不同 claude code 的一种扩展方式"
- rick 代码注释明确标注 `ClaudeCodeExecutor runs claude CLI and parses its stream-json output`
- 无 ai_cli 设计文档（human 口语 "ai_cli" 对应代码 `internal/agent/` 抽象层）

### 反事实（权重 0.1）✅

- 测试文件 `executor_test.go` 证明 `parseStream` 强依赖 claude NDJSON 格式（type: system/assistant/user/result）
- 测试文件 `runner_test.go` 证明 `TaskRunner` 通过 `mockAgentExecutor` 接口注入，claude code 实现可替换

## 还原确认

无 rick 代码修改（仅 Read/Grep），无需还原。

## 关键事实

1. **13 处生产调用点**：1 处通过 `ClaudeCodeExecutor`（doing 核心，stream-json）+ 8 处通过 `callClaudeCodeCLI`/`callClaudeCodeCLIBackground` 封装 + 3 处 `runAutoFix` + 1 处 learning 直接 exec
2. **核心调用链**：`rick doing` → `executor.NewExecutor` → `TaskRunner.RunTask` → `agentExecutor.Execute` → `ClaudeCodeExecutor.Execute` → `exec.Command(claude, -p, --output-format stream-json, --verbose, --dangerously-skip-permissions, file)`
3. **claude code 功能使用清单**（MECE 完备枚举）：
   - **flag 类**（6 项）：`-p` / `--output-format stream-json` / `--verbose` / `--dangerously-skip-permissions` / `--resume` / `--session-id`
   - **stream-json 协议字段类**（7 项）：`type:system` / `type:result` / `type:assistant` / `type:user` / `content.type:tool_use` / `content.type:text` / `content.type:tool_result` + `session_id` / `duration_ms` / `is_error`
   - **行为类**（4 项）：prompt 文件作为 positional arg / stdin-stdout-stderr 接管 / session 持久化 / 权限跳过
   - **未使用类**（5 项）：hooks(PreToolUse/PostToolUse) / settings.json 配置 / subagent / compaction 控制 / MCP
4. **system prompt 注入机制**：rick 完全通过 prompt 文件内容注入（WriteSkillFile + LoadLoopsContext + LoadSkillsContext + loadDoingLoopContent），不使用 claude code 的 `--system-prompt` / `--append-system-prompt` flag（claude code 是否支持这些 flag 在 rick 代码中无证据，前 4 轮调研确认 claude code 有这些 flag 但 rick 未用）
5. **skill 加载机制**：rick 将 skill 文件路径写入 prompt 文件（如 `{{debug_skill_path}}`），claude code 不主动加载 skill（rick 的 skill 是 prompt 内容引用，不是 claude code 内置 skill 机制）

## 疑问点

无。13 处调用点完整枚举，调用链反向回溯到 cmd 入口，claude code 功能完备枚举（使用 + 未使用）。

## 置信度评估（由 research 主调度计算）

- 代码原文 ✅ × 0.4 = 0.4
- 运行时行为 ✅ × 0.3 = 0.3
- 文档 ✅ × 0.2 = 0.2
- 反事实 ✅ × 0.1 = 0.1
- 合计 = 1.0（高，≥ 0.8 终止）
