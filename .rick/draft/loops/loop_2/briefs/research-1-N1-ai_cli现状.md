# research-1-N1-ai_cli 现状

节点路径:[根 > N1-ai_cli 现状]
事实陈述:rick 仓库中 ai_cli 的代码位置、扩展机制、如何启动/调用 claude code

## 执行动作
- Read `internal/agent/interface.go`（AgentExecutor 接口定义）
- Read `internal/agent/claudecode/executor.go`（claude code 实现）
- Grep "ai_cli|aicli|AiCli" 全仓库 Go 代码
- Read `internal/config/config.go`（ClaudeCodePath 配置项）

## 各信源验证结果

### 代码原文 0.4 ✅
- "ai_cli" 字符串在 Go 代码中**零匹配**。human 口语 "ai_cli" 对应代码中的 `internal/agent/` 抽象层
- `internal/agent/interface.go` 定义 `AgentExecutor` 接口（Execute 方法）+ `AgentSession` 接口（ID/Duration/ToolCalls/FinalMessage/RawLogPath）
- `internal/agent/claudecode/executor.go` 提供 `ClaudeCodeExecutor` 实现，构造函数 `NewExecutor(claudePath string)`
- `internal/config/config.go:7` 定义 `ClaudeCodePath string` 配置项

### 运行时行为 0.3 ✅
- `ClaudeCodeExecutor.Execute()` 调用 `exec.Command(e.claudePath, "-p", "--output-format", "stream-json", "--verbose", "--dangerously-skip-permissions", promptFile)`
- 输出通过 `parseStream` 解析 NDJSON（type: system/assistant/user/result）

### 文档 0.2 ✅
- MEMORY.md 记录 "rick 启动不同 claude code 的一种扩展方式"
- 无 ai_cli 设计文档

### 反事实 0.1 ✅
- 测试文件 `executor_test.go` 证明 parseStream 强依赖 claude NDJSON 格式

## 置信度计算
0.4×1 + 0.3×1 + 0.2×1 + 0.1×1 = **1.0（高）**

## 还原确认
本次调研未修改代码（仅 Read/Grep），无需还原。

## 疑问点
无。ai_cli 是 human 口语，代码中对应 `internal/agent/` 抽象层 + `ClaudeCodeExecutor` 实现。
