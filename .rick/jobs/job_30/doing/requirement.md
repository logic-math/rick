参考 draft 中的 rpc cluade 迁移 pi，以及 loop4 内的各种上下文信息，本次 job 要重构 rick 使其基于 pi 完成任务，将 claude code 全部移除

---

## Grilling 澄清结论（2026-08-11）

> 基于 loop_2 研究 brief（research-2-N2/N3/N5、research-5-N1/N2、research-7-N4，置信度 0.9-1.0）+ loop_4 RFC（rick=引导程序，pi=实现方式）+ 代码探索（13 处 claude 调用点、AgentExecutor 接口、config）结构化追问得出。

### 设计树总览

```
Layer 0 迁移策略：rick(引导程序) → piagent(pi 调用适配) → pi(agent runtime, --mode json) → 任务执行
Layer 1 适配器架构：config(pi_path) → [piagent.Executor(doing.go) | piagent.CallCLI(12 处)] → pi 子进程 → 解析会话/结果
Layer 2 叶子层：executor.go(PiExecutor+JSONL解析) / cli.go(CallCLI+FindBinary) / config.go(pi_path) / 命名+文档清理
```

### Layer 0 决策（迁移策略）

| 决策项 | 结论 | 依据 |
|---|---|---|
| D1 pi 可用性 | 人类迭代中安装 pi；我先按研究 brief 协议设计解析器 + mock pi 单测，真实 pi 装好后校准+集成测试 | pi 当前未安装（不在 PATH、无 ~/.pi、/tmp/pi_repo 已删）；研究 brief 已完整捕获 pi 协议 schema |
| D2 移除范围 | 全量移除（代码 + config + 命名 + 文档） | 用户选定；符合「将 claude code 全部移除」 |
| D3 迁移阶段 | 仅 Phase 1：1:1 功能迁移（pi --mode json，per-prompt 子进程，同构现状 claude stream-json） | research-7-N4 建议；不加 pi 增强（rpc/hooks/subagent/compaction/--system-prompt flag 留 Phase 2） |
| D4 12 处架构 | 重构为统一 CLI 抽象层，集中在 piagent 包 | 用户选定；research-2-N5 将此列为架构决策 |

### Layer 1 决策（适配器架构）

| 模块 | 决策 |
|---|---|
| M1 piagent 包结构 | **集中**：`internal/agent/piagent/` 同时含 Executor（AgentExecutor 实现，doing.go 路径，JSONL 解析）+ CallCLI（12 处统一调用）。cmd 文件 import piagent 做简单调用 |
| M2 config 迁移 | `claude_code_path`→`pi_path`，**干净重命名不向后兼容**（config.go struct tag + loader 验证 + config.example.json + 测试 + ~/.rick/config.json key 同步） |
| M3 pi 安装时机 | 迭代中装；先按研究设计，mock pi 单测先行，真实 pi 装好后校准 |

### Layer 2 叶子层（实现规格，四维度）

#### Module A — `internal/agent/piagent/executor.go`（PiExecutor，doing.go 路径）

- **关键代码实现**：
  - `func NewExecutor(piPath string) *Executor`
  - `func (e *Executor) Execute(promptFile, taskID, workspaceDir, logFileName string) (agent.AgentSession, error)`
  - JSONL 解析器 `func parseStream(r io.Reader, rawLogPath string) (*piSession, error)`
  - `piSession` 实现 `agent.AgentSession`（ID/Duration/ToolCalls/FinalMessage/FinalMessageLine/RawLogPath 6 方法）
- **工具调用**：`exec.Command(piPath, "--mode", "json", promptFile)`（per-prompt 子进程，同构现状）
- **字段映射**（claude NDJSON → pi JSONL 事件，来源 research-5-N2/research-7-N4）：
  - `type:system`/`session_id` → session header 首行 `id` → `ID()`
  - `type:result`+`duration_ms` → `agent_settled` 事件（无 duration）→ startTime 自计时 → `Duration()`
  - `type:assistant`+`content[type=text]` → `message_end` 事件 `message.content[type=text].text` → `FinalMessage()`
  - `content[type=tool_use]`(id/name/input) → `tool_execution_start`(toolCallId/toolName/args)
  - `content[type=tool_result]`(tool_use_id/content/is_error) → `tool_execution_end`(toolCallId/result/isError) → `ToolCall{Name,Input,Output,Line,IsError}`
  - snake_case → camelCase
- **环境依赖**：pi 二进制（迭代中安装）；pi --mode json 输出 JSONL

#### Module B — `internal/agent/piagent/cli.go`（12 处统一调用）

- **关键代码实现**：
  - `type CLIMode int`（`ModeInteractive` / `ModePrint`）
  - `func CallCLI(cfg *config.Config, promptFile string, mode CLIMode, extraArgs ...string) error`
    - Interactive：`pi <file> [extra]`，继承 stdin/stdout/stderr
    - Print：`pi -p <file> [extra]`，捕获输出
  - `func FindBinary(cfg *config.Config) string`（cfg.PiPath 或 PATH 中的 `pi`）
- **12 处改造点**：
  - `cmd/plan.go`：`callClaudeCodeCLI`→`piagent.CallCLI(Interactive)`；`callClaudeCodeCLIBackground`→`piagent.CallCLI(Print)`（去 `--dangerously-skip-permissions`）
  - `cmd/easy.go:149`：`--resume <id>`→`--continue <id>`（extraArg）；`easy.go:191`：`--session-id <id>`→`--session <id>`
  - `cmd/dream.go:97,102`：同 plan（Print/Interactive）
  - `cmd/human_loop.go:78`、`cmd/ctrl.go:74`、`cmd/learning.go:247`：→`piagent.CallCLI(Interactive)`
  - `cmd/tools_plan_check.go`、`cmd/tools_doing_check.go`、`cmd/tools_learning_check.go`：`runAutoFix`+`findClaudeBinary`→`piagent.CallCLI(Print)`+`piagent.FindBinary`（去 `--dangerously-skip-permissions`）
  - `internal/executor/runner.go:293-305`：`CallClaudeCodeCLI`→`piagent.CallCLI(Print)`

#### Module C — config 迁移

- **文件结构**：
  - `internal/config/config.go`：`ClaudeCodePath string \`json:"claude_code_path"\`` → `PiPath string \`json:"pi_path"\``
  - `internal/config/loader.go`：字段引用 + 验证错误信息（`ClaudeCodePath does not exist`→`PiPath does not exist`）+ 默认值
  - `config.example.json`：`"claude_code_path"`→`"pi_path"`
  - `~/.rick/config.json`：key 同步（值待 pi 装好后填路径，或留空走 PATH 的 `pi`）
  - `internal/config/loader_test.go`：所有 `ClaudeCodePath` 引用更新
- **环境依赖**：pi 装好后 `pi_path` 指向 pi 二进制

#### Module D — 命名 + 文档清理

- **文件结构**：
  - **删除** `internal/agent/claudecode/`（executor.go + executor_test.go）
  - `internal/executor/retry.go`：去除 claude 引用
  - `internal/executor/runner.go`：`CallClaudeCodeCLI` 方法去除/改用 piagent
- **文档**：
  - `wiki/architecture.md`：`callcli "Claude Code 集成"`→`"pi 集成"`；mermaid 节点 `EXT_CLAUDE`→`EXT_PI`；"调用 Claude Code"→"调用 pi"
  - `README.md`：`claude_code_path`→`pi_path`；"Claude 会话 UUID"→"pi 会话 UUID"；"调用 Claude"→"调用 pi"
  - 历史 job prompts（job_15/16/17/19、dream）为冻结制品，**不改**

### 测试策略

1. **mock pi 单测（先行，不依赖 pi 安装）**：写 mock pi 脚本按研究 brief 的文档化 JSONL 事件 schema emit 输出，跑 piagent 解析器单测
2. **真实 pi 集成测试（pi 装好后校准）**：用真实 `pi --mode json` 输出校准解析器字段映射，补集成测试
3. **编译 + 现有测试**：`go build ./...` + `go test ./...` 确保无回归

### 风险

- **R1**：研究 brief 与真实 pi 输出可能有偏差 → 迭代中用真实 pi 校准（M3 策略已覆盖）
- **R2**：全量移除 + 统一抽象跨 15 Go 文件 + config + 2 文档，改动面大 → Phase 1 1:1 迁移限定范围，easy loop 3 轮迭代收敛
- **R3**：~/.rick/config.json 现有 `claude_code_path: "ai_cli"` 不向后兼容 → 干净重命名，同步更新 key

---

## 端到端校准结论（2026-08-11，pi v0.84.1 安装后）

### 已校准（真实 pi 输出）

- **session header** `{"type":"session","version":3,"id":"..."}` → `ID()` ✅
- **agent_start / turn_start / turn_end / agent_end / agent_settled** 事件序列 ✅
- **message_end 双发**：pi 对 user 和 assistant 轮次**都**发 `message_end` → **发现并修复 bug**：原解析器把 user 输入误当 FinalMessage，已加 `message.role=="assistant"` 过滤（回归测试 `TestParseStream_UserMessageNotFinalMessage` 覆盖）
- **duration 自计时** ✅（pi 不输出 duration_ms，startTime→agent_settled）
- **真实 pi 二进制 smoke**（`realpi` build tag）：rick 真实 exec `pi --mode json` + 解析真实 session，ID/duration/无崩溃全通过
- **mock-pi 端到端**（`executor_e2e_test.go`）：Execute→parse→AgentSession→actpath 全链路，含 tool_execution_start/end 真实 schema

### 未完全校准（环境限制）

- ~~**真实工具调用流**未验证~~ → **已解决**（见下方"完整端到端验证"）
- 历史障碍：本环境 anthropic 网关（mcli.sankuai.com）是 claude-code 专用（需 X-Client-Token 等 pi 发不出的动态 header → 403）；早期 DeepSeek 调用因后台运行 + 缓冲问题疑似不稳定
- 突破：后台 tee（管道保行缓冲）+ 600s 超时后，真实 deepseek-v4-flash 工具调用流成功捕获（88 行，含 tool_execution_start/end）

### 完整端到端验证（2026-08-11，真实 LLM）

- **`TestRealPi_RealToolCall`**（`realpi` tag）：真实 pi 二进制 + 真实 DeepSeek + 真实 Read 工具，经 rick `Execute()` 全链路 → **PASS**（4.4s）
  - rick exec `pi --mode json --provider deepseek --model deepseek-v4-flash --api-key ... <file>`
  - 真实 LLM 调真实 Read 工具，rick 解析器捕获 2 个 tool calls + FinalMessage="DONE"
- **真实工具事件字段确认**：`tool_execution_start{toolCallId,toolName,args}` + `tool_execution_end{toolCallId,toolName,result,isError}` 与研究 brief 完全一致
  - `toolName` 实际为小写（`read`，非 `Read`）——不影响解析
  - `result` 是 JSON 对象（`{"content":[{"type":"text","text":...}]}`），`rawToString` 正确 unwrap
- **`TestParseStream_RealDeepSeekToolCall`**：用真实捕获的 88 行流跑解析器单测 → PASS

### 发现并修复的配置缺口（端到端验证副产品）

- **问题**：pi 不读 `PI_PROVIDER`/`PI_MODEL`/`PI_API_KEY` 环境变量，只认命令行 `--provider/--model/--api-key`。rick 原 `Execute()` 只发 `pi --mode json <file>`，用户无法配置非默认 provider
- **修复**：config 加 `pi_extra_args`（[]string），`Execute()` 和 `CallCLI()` 透传到 pi 命令行（global flags 在前，per-call flags 在后，prompt 文件最后）
- **验证**：`TestRealPi_RealToolCall` 用此机制注入 deepseek provider/model/key，真实工具调用通过

### rick 命令级端到端验证（真实 pi + 真实 DeepSeek LLM）

> 与之前 piagent 包内部验证不同：此次直接跑真实 `rick <cmd>`，验证 rick 命令层 + prompt 落盘到 job 目录 + pi 调用全链路。prompt 全部落盘到 rick 标准目录（非 /tmp），完全对齐 rick 功能。

| 命令 | 调 pi 方式 | prompt 落盘 | 结果 |
|---|---|---|---|
| `rick plan` | CallCLI(Interactive) | `job_N/plan/prompts/plan_prompt.md` | ✅ pi 返回 PLAN_OK，prompt 落盘正确 |
| `rick doing` | Execute(--mode json) | `job_N/doing/prompts/{task}_doing_prompt.md` | ✅ pi 真实执行任务（建文件+commit 4a23447），解析器正确，task success |
| `rick easy` | CallCLI(Interactive, --session-id) | `job_N/doing/prompts/easy_prompt.md` | ✅ 修复 bug 后通过（见下） |
| `rick learning` | CallCLI(Interactive) | `job_N/learning/prompts/learning_prompt.md` | ✅ pi 返回 LEARNING_OK |
| `rick human-loop` | CallCLI(Interactive) | `draft/loops/loop_N/prompts/sense_loop.md` | ✅ pi 返回 HUMANLOOP_OK，"思考记录已保存" |
| `rick ctrl --job` | CallCLI(Interactive) | `job_N/doing/prompts/ctrl_prompt.md` | ✅ pi 返回 CTRL_OK |
| `rick dream --background` | CallCLI(Print) | `.rick/dream/prompts/dream_prompt.md` | ⚠️ prompt 落盘+pi 启动正确；LLM 任务量大（5 job 反思）deepseek 超时未跑完（链路已验证，非代码问题） |

### 发现并修复的 easy 命令 bug（命令级验证副产品）

- **问题**：easy.go 用 `pi --session <UUID>` 创建新会话，但 pi 的 `--session` 是**加载已有会话**（找不到报错 "No session found matching"）。迁移时误把 claude 的 `--session-id`→pi 的 `--session`，语义不同
- **修复**：改回 `--session-id`（pi 同时支持 `--session-id`，专门用于"指定 ID 创建新会话"，"creating a new session with that id"）
- **验证**：修复后 `rick easy` 真实跑通（pi 创建会话 + 返回 EASY_OK）
- **根因**：研究 brief（research-5-N2）说 pi `--session` "接受 path 或 id"，但未明确"加载 vs 创建"语义差异；命令级验证才发现此偏差——证明**真实命令级端到端验证的必要性**

### 配置持久化

- DeepSeek key 已写入 `~/.bashrc`（`export DEEPSEEK_API_KEY=...`，家目录不进 repo）。新开 shell 后 `pi --provider deepseek --model deepseek-v4-flash` 可用
- rick config `pi_path: ""`（走 PATH 的 `pi`）

### 后续若需完整校准工具事件

- 等 DeepSeek 网关稳定后，重跑 `DEEPSEEK_API_KEY=... pi --provider deepseek --model deepseek-v4-flash --mode json -p "Read /tmp/x.txt"`，抓到 `tool_execution_start/end` 真实行后与解析器对照
- 或换一个对 pi 流式输出更稳定的 provider（需对应 API key）

