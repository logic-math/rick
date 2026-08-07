# research-3-N3-agent 类型与启动路径

节点路径:[根 > N3-agent 类型与启动路径]
事实陈述:rick 当前支持哪些 agent 类型/启动路径（easy/doing/learning/dream 等）

## 执行动作
- Grep "func New.*Cmd" internal/cmd/
- Read 各 cmd 文件 RunE 入口

## 各信源验证结果

### 代码原文 0.4 ✅
**rick 命令（agent 启动路径）**：
| 命令 | 入口 | claude 调用 | 模式 |
|---|---|---|---|
| plan | plan.go:24 RunE | callClaudeCodeCLI（行 159,209） | 交互式 |
| doing | doing.go:30 RunE | claudecode.NewExecutor → 接口（行 204） | 非交互（stream-json） |
| easy | easy.go:26 RunE | callClaudeCodeCLI(--session-id/--resume)（行 149,191） | 交互式 |
| learning | learning.go:25 RunE | exec.Command(claudePath)（行 247） | 交互式 |
| dream | dream.go:27 RunE | callClaudeCodeCLI(Background)（行 97,102） | 交互/后台 |
| human-loop | human_loop.go:20 RunE | callClaudeCodeCLI（行 78） | 交互式 |
| ctrl | ctrl.go:19 RunE | callClaudeCodeCLI（行 74） | 交互式 |
| tools | tools.go:28 RunE | 子命令（plan/doing/learning/dream/easy check） | 校验 |
| tools-plan-check | tools_plan_check.go:41 | exec.Command(claudePath)（行 207） | 交互式 |

**agent 类型抽象**：
- `internal/agent/interface.go` 定义 `AgentExecutor` 接口（仅 1 个 Execute 方法）
- 仅 1 个实现：`internal/agent/claudecode/executor.go` 的 `ClaudeCodeExecutor`
- 接口注入点仅 `doing.go:204`（通过 `executor.NewExecutor` 传入）

### 运行时行为 0.3 ✅
- 9 个命令入口，8 个直接 exec.Command claude，1 个走接口

### 文档 0.2 ✅
- MEMORY.md 记录三阶段工作流 plan → doing → learning + easy + dream

### 反事实 0.1 ✅
- executor_test.go 用 mockAgentExecutor 验证接口可替换

## 置信度计算
0.4×1 + 0.3×1 + 0.2×1 + 0.1×1 = **1.0（高）**

## 还原确认
未修改代码，无需还原。

## 疑问点
无。启动路径已穷举。
