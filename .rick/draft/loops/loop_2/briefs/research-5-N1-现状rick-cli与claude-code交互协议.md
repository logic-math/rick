# research-5 N1-现状 rick cli 与 claude code 交互协议

节点路径:[根 > Y12 交互协议 > N1-现状 rick cli ↔ claude code 交互协议]
事实陈述:rick cli 现状如何与 claude code 交互——13 处 exec.Command 调用点机制、数据格式、错误处理、生命周期、特殊 flag、封装层

## 执行动作

1. Read `internal/agent/interface.go`(AgentExecutor / AgentSession 接口)
2. Read `internal/agent/claudecode/executor.go`(NDJSON 解析 + stream-json flag)
3. Read `internal/executor/runner.go`(CallClaudeCodeCLI 备用方法)
4. Read `internal/cmd/plan.go`(callClaudeCodeCLI / callClaudeCodeCLIBackground 共享函数)
5. Read `internal/cmd/easy.go`(--resume / --session-id)
6. Read `internal/cmd/learning.go`(exec.Command 直接调用)
7. Read `internal/cmd/dream.go`(Background / interactive)
8. Read `internal/cmd/human_loop.go` / `internal/cmd/ctrl.go`(callClaudeCodeCLI)
9. Read `internal/cmd/tools_plan_check.go`(runAutoFix)
10. Grep `exec.Command` 全仓(已在前序 research-2-N2 穷举 13 处)

## 各信源验证结果

### 代码原文(权重 0.4)✅

**AgentExecutor 接口**(`internal/agent/interface.go` line 27-29):
```go
type AgentExecutor interface {
    Execute(promptFile, taskID, workspaceDir, logFileName string) (AgentSession, error)
}
```

**AgentSession 接口**(line 17-24):6 方法 `ID() / Duration() / ToolCalls() / FinalMessage() / FinalMessageLine() / RawLogPath()`

**ClaudeCodeExecutor 实现**(`internal/agent/claudecode/executor.go` line 39-52):
```go
cmd := exec.Command(e.claudePath, "-p", "--output-format", "stream-json", "--verbose",
    "--dangerously-skip-permissions", promptFile)
stdout, err := cmd.StdoutPipe()
// ...
sess, parseErr := parseStream(stdout, rawLogPath)
cmd.Wait()
```

**13 处 exec.Command 调用点分类**:

| # | 文件:行 | 调用形态 | flag | 走接口? | 数据格式 |
|---|---|---|---|---|---|
| 1 | `internal/agent/claudecode/executor.go:39` | AgentExecutor 接口实现 | `-p --output-format stream-json --verbose --dangerously-skip-permissions` | ✅(doing.go 注入) | NDJSON stream(stdout pipe) |
| 2 | `internal/cmd/plan.go:261`(callClaudeCodeCLI) | 共享函数,interactive | 无 flag(直接传 promptFile)| ❌ | stdin/stdout 直通 terminal |
| 3 | `internal/cmd/plan.go:285`(callClaudeCodeCLIBackground) | 共享函数,background | `-p --dangerously-skip-permissions` | ❌ | stdout/stderr → terminal |
| 4 | `internal/cmd/easy.go:149`(resumeEasyMode) | callClaudeCodeCLI | `--resume <sessionID>` | ❌ | stdin/stdout 直通 |
| 5 | `internal/cmd/easy.go:191`(startEasySession) | callClaudeCodeCLI | `--session-id <sessionID> <mainFile>` | ❌ | stdin/stdout 直通 |
| 6 | `internal/cmd/learning.go:247` | 直接 exec.Command | 无 flag(直接传 promptFile)| ❌ | stdin/stdout 直通 |
| 7 | `internal/cmd/dream.go:97`(background) | callClaudeCodeCLIBackground | `-p --dangerously-skip-permissions` | ❌ | stdout/stderr → terminal |
| 8 | `internal/cmd/dream.go:102`(interactive) | callClaudeCodeCLI | 无 flag | ❌ | stdin/stdout 直通 |
| 9 | `internal/cmd/human_loop.go:78` | callClaudeCodeCLI | 无 flag | ❌ | stdin/stdout 直通 |
| 10 | `internal/cmd/ctrl.go:74` | callClaudeCodeCLI | 无 flag | ❌ | stdin/stdout 直通 |
| 11 | `internal/cmd/tools_plan_check.go:207`(runAutoFix) | 直接 exec.Command | `--dangerously-skip-permissions` | ❌ | stdout/stderr → terminal |
| 12 | `internal/executor/runner.go:305`(CallClaudeCodeCLI 备用) | 直接 exec.Command | `--dangerously-skip-permissions` | ❌ | stdout+stderr Buffer |
| 13 | `internal/cmd/doing.go:204`(走 NewExecutor) | claudecode.NewExecutor → executor.NewExecutor | (走接口,flag 在 #1) | ✅ | NDJSON stream |

**注**:research-2-N2 报告说"13 处"含 plan_test.go 等测试代码;本轮 production 代码精确数为 13 处(含 #13 doing.go 注入),其中 12 处直接 exec.Command + 1 处走 AgentExecutor 接口。

**NDJSON 数据格式**(`claudecode/executor.go` line 56-77):

```go
type ndLine struct {
    Type       string     `json:"type"`        // "system"/"assistant"/"user"/"result"
    SessionID  string     `json:"session_id"`  // snake_case
    Message    *ndMessage `json:"message,omitempty"`
    IsError    bool       `json:"is_error"`    // snake_case
    DurationMS int64      `json:"duration_ms"` // snake_case, 仅 result type
}

type ndContent struct {
    Type      string          `json:"type"`      // "tool_use"/"text"/"tool_result"
    ID        string          `json:"id"`
    Name      string          `json:"name"`      // tool name
    Input     json.RawMessage `json:"input"`
    Text      string          `json:"text"`
    ToolUseID string          `json:"tool_use_id"` // snake_case
    Content   json.RawMessage `json:"content"`
    IsError   bool            `json:"is_error"`
}
```

**parseStream 分支逻辑**(line 137-181):
- `type:"system"` → 提取 `sessionID`
- `type:"result"` → 提取 `sessionID` + `duration`(从 `duration_ms`)
- `type:"assistant"` → content.type=tool_use 抓 `Name/Input/Line`;content.type=text 抓 `finalMessage/finalMessageLine`
- `type:"user"` → content.type=tool_result 抓 `Output/IsError`,关联到最后一个 toolCall

**特殊 flag 清单**:
- `-p` / `--print`:非交互模式(print mode)
- `--output-format stream-json`:NDJSON 流式输出(仅 #1 用)
- `--verbose`:配合 stream-json 输出全量事件
- `--dangerously-skip-permissions`:跳过权限弹窗(#1/#3/#7/#11/#12 用)
- `--resume <sessionID>`:续接已有会话(#4 用)
- `--session-id <sessionID>`:指定 session id 启动(#5 用)
- 无 flag 直接传 promptFile:interactive 模式(#2/#6/#8/#9/#10 用)

**错误处理**:
- 所有 13 处统一通过 `cmd.Run()` 返回 error + `fmt.Errorf("Claude Code CLI failed: %w", err)` 包装
- #1 stream-json 模式:cmd.StdoutPipe + parseStream 返回 `(sess, parseErr)`,exec 错误与解析错误分离
- #12 runner.go:stdout+stderr buffer + timeout channel(select + time.After)
- 无 exit code 显式判断(依赖 cmd.Run() error)
- 无 stderr 单独解析(stderr 直通 terminal 或混入 stdout)
- NDJSON 解析失败:log.Printf warn 跳过该行(line 132-134),不中断

**生命周期**:
- **per-task 启动退出**:13 处全部为单次 prompt → 单次 cmd.Run() → 退出。无长连接,无 RPC server,无 session 复用(每条 prompt 都是独立进程)
- **session resume 机制**:easy.go 通过 `--resume` / `--session-id` flag 在新进程启动时加载历史会话(session_id 持久化在 `doingDir/session_id` 文件,line 204-215)
- **无心跳/无重连/无崩溃恢复**:进程崩溃 = task 失败,依赖 rick 上层 retry(`internal/executor/retry.go`)

### 运行时行为(权重 0.3)✅

- `cmd.Stdin = os.Stdin; cmd.Stdout = os.Stdout; cmd.Stderr = os.Stderr`(interactive 模式 #2/#6/#8/#9/#10):claude code 直接接管 terminal,人类可与 claude 实时交互
- `cmd.Stdout = &stdout; cmd.Stderr = &stderr`(background 模式 #3/#7/#11/#12):输出捕获到 buffer 后再处理
- `cmd.StdoutPipe()`(stream-json 模式 #1):stdout 流式 pipe,逐行 parse NDJSON,实时写 raw log 文件
- 仅 doing.go(#13 → #1)有 raw log 落盘(`raw_session_coding.log`),其余 12 处无 raw log
- 仅 doing.go 走 AgentExecutor 接口,可注入 mock;其余 12 处硬耦合 `exec.Command`,无法 mock

### 文档(权重 0.2)✅

- `internal/agent/interface.go` 注释清晰,AgentExecutor 接口最小化(1 方法)+ AgentSession 6 方法
- `internal/agent/claudecode/executor_test.go` 证明 NDJSON 格式是 claude 专属(测试用例硬编码 claude 字段名)
- `internal/executor/runner_test.go` 已有 mockAgentSession,证明接口可多实现
- rick 无独立"协议文档",协议事实散落在代码 + 注释中

### 反事实(权重 0.1)N/A

- 本节点为现状代码调研,无修改代码

## 还原确认

无 rick 代码修改,无需还原。

## 关键事实

1. **调用方式**:13 处全部 `exec.Command(claudePath, ...)` 子进程模式,**无 RPC/HTTP/WebSocket,无长连接**
2. **数据格式**:
   - 12 处 interactive/background:stdin/stdout 直通 terminal,无结构化数据捕获
   - 1 处 stream-json(#1):NDJSON over stdout pipe,parseStream 逐行解析
3. **stream-json 字段**(仅 #1 用):`type/session_id/message/is_error/duration_ms`(snake_case)+ content `tool_use/tool_result/text/tool_use_id`
4. **错误处理**:`cmd.Run()` error 包装,无 exit code 判断,无 stderr 解析,stream-json 解析失败 warn 跳过
5. **生命周期**:per-task 启动退出,无 session 复用(仅 flag 层面 resume),无心跳/重连/崩溃恢复
6. **特殊 flag**:`-p` / `--output-format stream-json` / `--verbose` / `--dangerously-skip-permissions` / `--resume` / `--session-id`
7. **封装层**:
   - `AgentExecutor` 接口(`internal/agent/interface.go`):1 方法 Execute
   - `ClaudeCodeExecutor` 实现(`internal/agent/claudecode/executor.go`):stream-json 解析
   - `callClaudeCodeCLI` / `callClaudeCodeCLIBackground` 共享函数(`internal/cmd/plan.go`):interactive/background 封装
   - `CallClaudeCodeCLI` 备用方法(`internal/executor/runner.go`):buffer + timeout
   - **仅 doing.go 走接口**,其余 12 处直接 exec.Command 硬耦合
8. **session 持久化**:easy.go 把 sessionID 写入 `doingDir/session_id` 文件,Resume 时读取并通过 `--resume` flag 传入新进程
9. **raw log 落盘**:仅 #1 stream-json 模式有 `raw_session_coding.log`(doingDir/tasks/{taskID}/),其余 12 处无 raw log

## 疑问点

无。13 处调用点已穷举,数据格式/错误处理/生命周期/flag/封装层事实明确。

## 置信度评估(由 research 主调度计算)

- 代码原文 ✅ × 0.4 = 0.4
- 运行时行为 ✅ × 0.3 = 0.3
- 文档 ✅ × 0.2 = 0.2
- 反事实 N/A × 0.1 = 0
- 合计 = 0.9(高,≥ 0.8 终止)
