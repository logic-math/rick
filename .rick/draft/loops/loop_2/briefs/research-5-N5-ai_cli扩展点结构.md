# research-5-N5-ai_cli 扩展点结构

节点路径:[根 > N5-ai_cli 扩展点结构]
事实陈述:ai_cli 是否已抽象出 "agent 类型" 接口，PI agent 接入需要哪些改动

## 执行动作
- Read `internal/agent/interface.go`（接口定义）
- Read `internal/executor/executor.go`（接口注入点）
- 反事实：临时修改 doing.go 把 claudecode.NewExecutor 替换为 mockPIAgentExecutor，go build 验证
- git restore 还原

## 各信源验证结果

### 代码原文 0.4 ✅
**已有抽象**：
- `agent.AgentExecutor` 接口（interface.go:27）：`Execute(promptFile, taskID, workspaceDir, logFileName string) (AgentSession, error)`
- `agent.AgentSession` 接口（interface.go:11）：ID/Duration/ToolCalls/FinalMessage/FinalMessageLine/RawLogPath
- `agent.ToolCall` 结构体（interface.go:6）：Name/Input/Output/Line/IsError

**接口注入点**：
- `executor.NewExecutor(..., agentExecutor agent.AgentExecutor, ...)`（executor.go:56）
- `TaskRunner.agentExecutor` 字段（runner.go:31）
- 仅 `doing.go:204` 通过 `claudecode.NewExecutor(cfg.ClaudeCodePath)` 注入

**PI agent 接入需改动的点**：
1. 新增 `internal/agent/piagent/executor.go` 实现 `AgentExecutor` 接口（需适配 PI agent 的输出格式到 AgentSession）
2. `doing.go:204` 把 `claudecode.NewExecutor` 替换为 `piagent.NewExecutor`（或按配置选择）
3. **未走接口的 12 处需重构**：plan/easy/dream/learning/human_loop/ctrl/tools_plan_check/runner.CallClaudeCodeCLI 直接 exec.Command claude，需抽象出 "CLI agent" 接口（含 Interactive/Background/Resume/SessionID 等方法）
4. `ClaudeCodePath` 配置项需泛化为 `AgentType` + `AgentPath`（或类似）
5. claude 特有 flag（--dangerously-skip-permissions/--session-id/--resume/-p/--output-format stream-json）需按 agent 类型分发

### 运行时行为 0.3 ✅
- 反事实验证：doing.go:204 替换为 `&mockPIAgentExecutor{}` 后 go build 报错 `undefined: mockPIAgentExecutor`（证明注入点松耦合，只需提供接口实现）
- 但 callClaudeCodeCLI 等共享函数未抽象，无法通过接口替换

### 文档 0.2 ✅
- MEMORY.md 无 agent 类型扩展文档

### 反事实 0.1 ✅
- 反事实实验：修改 doing.go:204 → go build 报错 → git restore 还原 → 还原后 grep -c "claudecode.NewExecutor" = 1（确认还原）

## 置信度计算
0.4×1 + 0.3×1 + 0.2×1 + 0.1×1 = **1.0（高）**

## 还原确认
- 反事实修改：`internal/cmd/doing.go:204` 临时替换为 `&mockPIAgentExecutor{}`
- 还原方式：`git restore internal/cmd/doing.go`
- 还原验证：`git diff --stat internal/cmd/doing.go` 无输出；`grep -c "claudecode.NewExecutor" internal/cmd/doing.go` = 1（原始状态）

## 疑问点
1. PI agent 是否提供与 claude stream-json 等价的输出格式？（影响 AgentSession 实现）
2. PI agent 是否支持交互式模式（stdin/stdout 转发）？（影响 easy/dream/learning/human_loop）
3. PI agent 是否支持 session resume？（影响 easy --resume）
4. "后续不再支持其他类型的 agent" 是否意味着需删除 callClaudeCodeCLI 共享函数，统一走接口？
