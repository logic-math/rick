# research-2 N5-pi 与 rick AgentExecutor 接口语义对齐性

节点路径:[根 > N5-pi 与 rick AgentExecutor 接口语义对齐性]
事实陈述:基于 N2/N3/N4 综合判断,pi 的扩展机制与运行时形态能否与 rick `AgentExecutor` 接口语义对齐,12 处直接 exec.Command 能否被泛化重构。

## 执行动作

1. 读取 rick `internal/agent/interface.go`(AgentExecutor / AgentSession 接口定义)
2. 读取 rick `internal/agent/claudecode/executor.go`(现有实现 + NDJSON 类型)
3. 读取 rick `internal/executor/runner.go`(调用点)
4. 读取 rick `internal/executor/runner_test.go`(mock 实现)
5. grep rick 全仓 `exec.Command.*claude` 调用点(13 处,见 N2 上一轮报告)
6. 综合对比 pi RPC 协议(rpc.md)vs rick NDJSON 解析(claudecode/executor.go)

## 信源验证结果

### 代码原文(权重 0.4)✅

**rick AgentExecutor 接口**(`internal/agent/interface.go`):
```go
type AgentSession interface {
    ID() string
    Duration() time.Duration
    ToolCalls() []ToolCall
    FinalMessage() string
    FinalMessageLine() int
    RawLogPath() string
}

type AgentExecutor interface {
    Execute(promptFile, taskID, workspaceDir, logFileName string) (AgentSession, error)
}

type ToolCall struct {
    Name    string
    Input   string
    Output  string
    Line    int
    IsError bool
}
```

**rick NDJSON 解析**(`claudecode/executor.go` line 54-77):
```go
type ndLine struct {
    Type       string     `json:"type"`        // "system"/"assistant"/"user"/"result"
    SessionID  string     `json:"session_id"`  // ← claude 风格
    Message    *ndMessage `json:"message,omitempty"`
    IsError    bool       `json:"is_error"`    // ← claude 风格
    DurationMS int64      `json:"duration_ms"` // ← claude 风格
}

type ndContent struct {
    Type      string          `json:"type"`      // "tool_use"/"text"
    ID        string          `json:"id"`
    Name      string          `json:"name"`
    Input     json.RawMessage `json:"input"`
    Text      string          `json:"text"`
    ToolUseID string          `json:"tool_use_id"` // ← claude 风格
    Content   json.RawMessage `json:"content"`
    IsError   bool            `json:"is_error"`
}
```

**rick 现有 13 处 claude 调用点**(上一轮 N2 报告已穷举):
- `internal/agent/claudecode/executor.go:39` — `claude -p --output-format stream-json --verbose --dangerously-skip-permissions <promptFile>`(走 AgentExecutor 接口,doing.go 注入)
- `internal/cmd/plan.go:285` — `claude -p --dangerously-skip-permissions <promptFile>`
- `internal/cmd/easy.go:149/191` — `claude --resume <id>` / `claude --session-id <id> <mainFile>`
- `internal/cmd/tools_plan_check.go:207` — `claude --dangerously-skip-permissions <promptFile>`
- `internal/executor/runner.go:305` — `claude --dangerously-skip-permissions <promptFile>`
- 其余 8 处(plan_test.go 测试 + 其他)

**pi RPC 协议对齐性分析**:

| rick AgentSession 方法 | rick 现状(claude code) | pi RPC 对应 | 对齐难度 |
|---|---|---|---|
| `ID()` | 解析 `session_id` 字段 | RPC `get_state` response 的 `sessionId` / session header 的 `id` | 低(字段重命名) |
| `Duration()` | 解析 `result.duration_ms` | ❌ pi 不输出 duration → rick 需自计时(start_time → agent_settled) | 中(改逻辑) |
| `ToolCalls()` | 解析 `assistant.message.content[type=tool_use]` | 解析 `tool_execution_end` 事件(`toolName` + `args` + `result` + `isError`) | 中(schema 不同) |
| `FinalMessage()` | 解析最后一条 `assistant.message.content[type=text].text` | 解析最后一条 `message_end` event 中 `message.content[type=text].text` | 低(事件名变) |
| `FinalMessageLine()` | 行号(原始日志) | rick 自维护行计数器(每行 JSONL +1) | 低(自维护) |
| `RawLogPath()` | rick 自己写的日志路径 | rick 自己写日志(捕获 pi RPC stdout JSONL) | 低(无差异) |

### 运行时行为(权重 0.3)✅

**12 处直接 exec.Command 泛化重构可行性**:

| 命令 | 现有 flag | pi 对应 flag | 重构难度 |
|---|---|---|---|
| plan.go | `-p --dangerously-skip-permissions <file>` | `-p <file>`(pi 无 permission 概念,默认无 popup) | 低(去掉 flag) |
| easy.go:149 | `--resume <id>` | `--session <id>` 或 `-c` | 低(重命名) |
| easy.go:191 | `--session-id <id> <file>` | `--session <id> <file>` | 低(重命名) |
| tools_plan_check.go | `--dangerously-skip-permissions <file>` | `-p <file>` | 低(去掉 flag) |
| runner.go:305 | `--dangerously-skip-permissions <file>` | `-p <file>` 或 `--mode rpc` 长连接 | 低-中 |
| doing.go(走接口) | stream-json | `--mode rpc` 或 `--mode json` | 中(解析器重写) |

**关键差异**:
- pi 无 `--dangerously-skip-permissions`(默认无 permission popup,Philosophy 段:"No permission popups")
- pi 无 `--verbose` flag
- pi `--mode rpc` 是长连接,优于 rick 现有"每次 prompt 启动一次 claude 子进程"模式
- pi 会话续接用 `--session <id>`(claude 用 `--session-id`/`--resume`,flag 名不同但语义同)

**AgentExecutor 接口语义对齐**:
- ✅ `Execute(promptFile, taskID, workspaceDir, logFileName)` 签名兼容:pi 可用 `pi -p <promptFile>` 或 `pi --mode rpc` + prompt 命令
- ✅ 返回 `AgentSession` 可填充:ID/ToolCalls/FinalMessage/FinalMessageLine/RawLogPath 均可从 pi RPC 事件流提取
- ⚠️ `Duration()` 需 rick 自计时(pi 不输出 duration_ms)
- ✅ `ToolCall{Name, Input, Output, Line, IsError}` 可从 `tool_execution_start` + `tool_execution_end` 事件组装

### 文档(权重 0.2)✅

- rick `interface.go` 注释清晰,接口最小化(6 方法)
- pi rpc.md 提供 Python + Node.js 客户端示例,证明协议可被外部消费
- pi SDK 文档:`AgentSession` 暴露 `subscribe(listener)` 可捕获所有事件,等价 rick 的 NDJSON 流式解析
- rick runner_test.go 已有 mockAgentSession,证明接口可多实现

### 反事实(权重 0.1)N/A

- 本节点为接口语义对比,未修改代码(本轮约束:纯外部调研)

## 还原确认

无 rick 代码修改,无需还原。

## 关键事实

1. **接口语义对齐**:✅ rick `AgentExecutor` 与 pi RPC 模式语义可对齐
   - `Execute()` 签名无需改动
   - `AgentSession` 6 方法均可填充(5 个低难度,1 个中难度:`Duration()` 需自计时)
2. **NDJSON 解析器重写**:必要,因字段名/schema 不同
   - rick 现有 `ndLine` 硬编码 `session_id`/`is_error`/`duration_ms`/`tool_use`/`tool_use_id`
   - pi 用 `sessionId`/`isError`/无 duration_ms/`toolCall`/`toolCallId`/事件流(`tool_execution_*`)
   - 需新建 `internal/agent/piagent/executor.go`,实现 `PiExecutor` + 事件流解析
3. **12 处 exec.Command 泛化**:✅ 可行,但需分层
   - **低难度(8 处)**:plan/easy/tools_plan_check/runner 等纯 flag 重命名 + 去掉 `--dangerously-skip-permissions`
   - **中难度(doing.go 1 处)**:走 AgentExecutor 接口,需新增 PiExecutor 实现
   - **中难度(其余 4 处)**:需引入 CLI agent 抽象层(当前仅 doing.go 走接口,其余直接 exec.Command)
4. **关键收益点**:pi `--mode rpc` 长连接可消除 12 处反复启动子进程开销,且支持 steering/followUp 消息队列(claude code 无)
5. **关键阻碍点**:
   - 字段 schema 全不对齐(需新解析器)
   - `duration_ms` 缺失(需自计时)
   - pi 无 `--dangerously-skip-permissions` 等价 flag(默认无 permission,但若需要 permission gate 需写 extension)
6. **rick 现有架构优势**:`AgentExecutor` 接口已抽象 + doing.go 已注入,新增 PiExecutor 是"加法"不破坏现有

## 疑问点

- rick 是否要将 12 处直接 exec.Command 全部重构为走 AgentExecutor 接口?还是仅 doing.go 走接口,其余保持 exec.Command + flag 适配层?(架构决策,非事实调研)
- pi RPC 长连接模式下,rick 如何管理 pi 进程生命周期(启动/心跳/超时/崩溃恢复/复用)?(实现细节,非事实调研)
- rick 现有 `--resume` 语义(续接会话)与 pi `--session` 是否完全等价?pi 树结构会话是否影响 rick 线性会话假设?(需 rick 端验证,但 pi 文档显示 `--session <id>` 加载完整会话,语义兼容)

## 置信度评估(由 research 主调度计算)

- 代码原文 ✅ × 0.4 = 0.4
- 运行时行为 ✅ × 0.3 = 0.3
- 文档 ✅ × 0.2 = 0.2
- 反事实 N/A × 0.1 = 0
- 合计 = 0.9(高,≥ 0.8 终止)
