# 运行时模块（internal/runtime）

## 职责

`internal/runtime` 是三层金字塔第三层「执行」的一员，对 agent runtime（当前只有 pi）的调用逻辑做封装。它不安装 pi、不拼提示词、不维护 dag 调度与门禁。

- 参数解析 + 调用 pi（合并 provider/model/api-key 等 flags）
- 内部校验 session 就绪
- 采集行为轨迹（trace）
- 返回 `(sessionID, trace)`

## Runtime 契约

```go
type Runtime interface {
    Name() string
    Run(methodText string, promptFile string, cfg *config.Config) (sessionID string, trace *Trace, err error)
}
```

```go
type Trace struct {
    SessionID    string
    ToolCalls    []ToolCall
    FinalMessage string
    RawLogPath   string
    Duration     time.Duration
    Settled      bool
}
```

`piRuntime` 是当前唯一实现。`Run` 返回非空 sessionID 表示运行成功；若 JSONL 流从未产生 session id 或从未发出 `agent_settled`（会话未 settle），`Run` 返回错误，由 handler 应用重试安全网。

## 三层注入

`Run` 把方法层（methodText）写临时文件，经 `--append-system-prompt <methodFile>` 注入，保留 pi 默认骨架；promptFile 作为 user prompt（实例上下文）最后传入。

## pi 配置目录隔离

所有 pi 调用入口注入 `PI_CODING_AGENT_DIR=~/.rick/pi/agent`（`AgentEnv()`），与用户 `~/.pi` 完全隔离。

## pi 解析优先级（全链路一致）

`cfg.PiPath` → 托管运行时（`RuntimeBin()` = `~/.rick/pi/agent/runtime/node_modules/.bin/pi`）→ PATH `pi`。

## 相关文件

| 文件 | 职责 |
|------|------|
| `runtime.go` | Runtime 接口 + piRuntime 实现 + Trace/ToolCall |
| `executor.go` | pi `--mode json` 事件流解析（session/tool/agent_settled） |
| `cli.go` | CallCLI 交互入口 + pi 路径解析 |
| `agentdir.go` | 共享路径工具（AgentDir/RuntimeDir/RuntimeBin/AgentEnv） |
