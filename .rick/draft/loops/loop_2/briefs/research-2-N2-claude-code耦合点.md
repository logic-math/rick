# research-2-N2-claude code 耦合点

节点路径:[根 > N2-claude code 耦合点]
事实陈述:rick 哪些模块依赖 claude code（命令调用、prompt 模板、subagent 派发、skill 加载等）

## 执行动作
- Grep "callClaudeCodeCLI|callClaudeCodeCLIBackground|claudecode.NewExecutor|exec.Command(claudePath" 全 cmd/executor 目录
- Grep "stream-json|--output-format|tool_use|tool_result|--dangerously-skip-permissions|--session-id|--resume" 全 internal
- Grep "claude|subagent|Task tool" prompt/templates/

## 各信源验证结果

### 代码原文 0.4 ✅
**claude code 调用点（13 处）**：
| 文件 | 行 | 调用方式 | 走接口? |
|---|---|---|---|
| cmd/plan.go:249 | callClaudeCodeCLI（共享函数） | exec.Command | ❌ |
| cmd/plan.go:279 | callClaudeCodeCLIBackground | exec.Command -p --skip-perms | ❌ |
| cmd/plan.go:159,209 | plan 命令调用 | callClaudeCodeCLI | ❌ |
| cmd/doing.go:204 | claudecode.NewExecutor → executor.NewExecutor | ✅ AgentExecutor 接口 |
| cmd/easy.go:149,191 | callClaudeCodeCLI(--resume/--session-id) | exec.Command | ❌ |
| cmd/dream.go:97,102 | callClaudeCodeCLI(Background) | exec.Command | ❌ |
| cmd/learning.go:247 | exec.Command(claudePath, promptFile) | exec.Command | ❌ |
| cmd/human_loop.go:78 | callClaudeCodeCLI | exec.Command | ❌ |
| cmd/ctrl.go:74 | callClaudeCodeCLI | exec.Command | ❌ |
| cmd/tools_plan_check.go:207 | exec.Command(claudePath, --skip-perms) | exec.Command | ❌ |
| executor/runner.go:305 | CallClaudeCodeCLI（备用方法） | exec.Command | ❌ |

**claude 特有协议耦合**：
- `claudecode/executor.go:39-40`: `--output-format stream-json --verbose --dangerously-skip-permissions` flag 硬编码
- `claudecode/executor.go:56-77`: NDJSON 结构体（ndLine/ndMessage/ndContent）强依赖 claude 输出格式
- `claudecode/executor.go:137-181`: parseStream 按 type=system/assistant/user/result 分支解析，content.type=tool_use/tool_result/text
- `easy.go:149,191`: `--resume <sessionID>` / `--session-id <sessionID>` 是 claude 特有 flag

### 运行时行为 0.3 ✅
- only doing.go 走 AgentExecutor 接口；其余 12 处直接 exec.Command 硬耦合

### 文档 0.2 ✅
- prompt 模板中 "subagent" 是 rick 自己的概念（sense_loop 派发 think/research/exporter），通过 prompt 文件 + claude CLI 实现，非 claude code 内置 subagent 机制
- 模板无 claude 专有 API，均为 prompt 文本 + 路径占位符

### 反事实 0.1 ✅
- 测试文件 executor_test.go 证明 NDJSON 格式是 claude 专属

## 置信度计算
0.4×1 + 0.3×1 + 0.2×1 + 0.1×1 = **1.0（高）**

## 还原确认
未修改代码，无需还原。

## 疑问点
无。耦合点已穷举（13 处调用 + NDJSON 协议 + claude flag）。
