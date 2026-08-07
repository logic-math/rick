# research-5 N2-迁移到 pi 的映射关系

节点路径:[根 > Y12 交互协议 > N2-迁移到 pi 的映射关系]
事实陈述:迁移到 pi 时 rick 现有 13 处 exec.Command 调用点如何映射到 pi 的 CLI/RPC 调用,字段/flag/生命周期/错误处理的映射关系,适配层设计

## 执行动作

1. Read `/tmp/pi_repo/packages/coding-agent/src/cli/args.ts`(全部 CLI flag 定义)
2. Read `/tmp/pi_repo/packages/coding-agent/src/main.ts` line 95-180(resolveAppMode / mode 路由)
3. Read `/tmp/pi_repo/packages/coding-agent/src/modes/print-mode.ts`(print/json 模式实现)
4. Read `/tmp/pi_repo/packages/coding-agent/src/modes/rpc/rpc-types.ts`(RPC 命令清单)
5. Read `/tmp/pi_repo/packages/agent/src/types.ts` line 422-437(AgentEvent union)
6. Read `/tmp/pi_repo/packages/coding-agent/src/core/agent-session.ts` line 141-186(AgentSessionEvent)
7. Grep `--mode|--session|--continue|--fork|--system-prompt|--append-system-prompt` 全 coding-agent/src
8. Grep `isError|durationMs|duration|toolName|toolCallId` rpc-types.ts(字段确认)

## 各信源验证结果

### 代码原文(权重 0.4)✅

**pi CLI flag 清单**(`cli/args.ts` Args interface):

| rick 现状 flag(claude code) | pi 对应 flag | 映射难度 | 备注 |
|---|---|---|---|
| `-p` | `-p` / `--print` | 低(同名) | pi 也有 `-p` print 模式 |
| `--output-format stream-json` | `--mode json` | 中(语义对应,flag 不同) | pi 用 `--mode {text\|json\|rpc}` |
| `--verbose` | `--verbose` | 低(同名) | pi 也支持 |
| `--dangerously-skip-permissions` | ❌ **无对应 flag** | 低(直接删除) | pi 默认无 permission popup(Philosophy: No permission popups) |
| `--resume <sessionID>` | `--continue` / `-c` 或 `--resume` / `-r` | 低(pi 同时支持两者) | pi `-c` 续最近会话,`-r` 浏览选择 |
| `--session-id <sessionID> <file>` | `--session <path\|id> <file>` | 低(重命名) | pi `--session` 接受 path 或 id |
| (无对应) | `--fork <id>` | 增强 | pi 原地分支,claude 无 |
| (无对应) | `--mode rpc` | 增强 | pi 长连接 RPC,claude 无 |
| (无对应) | `--system-prompt <text>` | 增强 | pi 静态 system prompt 注入 |
| (无对应) | `--append-system-prompt <text>` | 增强 | pi 追加 system prompt(可重复) |
| (无对应) | `--provider <name> --model <pattern>` | 增强 | pi 多 provider 切换 |

**pi 模式路由**(`main.ts` line 109-120 resolveAppMode):
```typescript
function resolveAppMode(parsed, stdinIsTTY, stdoutIsTTY): AppMode {
    if (parsed.mode === "rpc") return "rpc";
    if (parsed.mode === "json") return "json";
    if (parsed.print || !stdinIsTTY || !stdoutIsTTY) return "print";
    return "interactive";
}
```
- 4 种模式:**interactive**(TUI)/ **print**(单次 text)/ **json**(单次 JSONL 事件流)/ **rpc**(长连接 stdin/stdout JSONL)
- print 模式自动触发条件:`-p` flag 或 stdin/stdout 非 TTY(pipe 模式)

**pi RPC 命令清单**(`rpc-types.ts` line 20-73 RpcCommand union,30+ 命令):

| 类别 | 命令 |
|---|---|
| Prompting | `prompt` / `steer` / `follow_up` / `abort` / `new_session` |
| State | `get_state` |
| Model | `set_model` / `cycle_model` / `get_available_models` |
| Thinking | `set_thinking_level` / `cycle_thinking_level` |
| Queue | `set_steering_mode` / `set_follow_up_mode` |
| Compaction | `compact` / `set_auto_compaction` |
| Retry | `set_auto_retry` / `abort_retry` |
| Bash | `bash` / `abort_bash` |
| Session | `get_session_stats` / `export_html` / `switch_session` / `fork` / `clone` / `get_entries` / `get_tree` / `set_session_name` |
| Messages | `get_messages` / `get_last_assistant_text` |
| Commands | `get_commands` |

**pi RPC 事件清单**(`agent-session.ts` line 141-183 + `agent-loop.ts` AgentEvent union):

| 事件类别 | 事件名 | 字段 |
|---|---|---|
| Agent lifecycle | `agent_start` / `agent_end`(messages + willRetry)/ `agent_settled` | - |
| Turn | `turn_start` / `turn_end`(message + toolResults) | - |
| Message | `message_start` / `message_update`(delta)/ `message_end` | message: AgentMessage |
| Tool execution | `tool_execution_start` / `tool_execution_update` / `tool_execution_end` | **toolCallId + toolName + args + result + isError**(无 duration) |
| Compaction | `compaction_start` / `compaction_end`(reason + result + aborted + willRetry) | - |
| Retry | `auto_retry_start` / `auto_retry_end` / `summarization_retry_*` | - |
| Queue | `queue_update`(steering + followUp) | - |
| Bash | `bash_execution_update`(delta) | - |
| Session | `entry_appended` / `session_info_changed` / `thinking_level_changed` | - |

**字段对齐性表**(rick NDJSON vs pi 事件):

| rick 期望字段 | rick 现状(claude code stream-json) | pi 对应(RPC/json 事件) | 对齐难度 |
|---|---|---|---|
| session_id | `session_id`(ndLine) | `sessionId`(RpcSessionState)/ `id`(session header) | 低(字段重命名 snake→camel) |
| tool_use(content.type) | `type:"tool_use"` + `id` + `name` + `input` | `type:"tool_execution_start"` event + `toolCallId` + `toolName` + `args` | 中(事件 vs content block,字段名不同) |
| tool_result(content.type) | `type:"tool_result"` + `tool_use_id` + `content` + `is_error` | `type:"tool_execution_end"` event + `toolCallId` + `result` + `isError` | 中(事件 vs content block) |
| duration_ms | `duration_ms`(result type) | ❌ **缺失**(pi 不输出 duration) | 中(rick 需自计时:start_time → agent_settled) |
| is_error | `is_error`(result) | `isError`(tool_execution_end) | 低(snake→camel) |
| final text | `assistant.message.content[type=text].text` | `message_end` event 中 `message.content[type=text].text` | 低(事件名变,内部 schema 同) |
| 终止信号 | `type:"result"` 终止行 | `agent_settled` 事件 | 低(事件名变) |

**pi 错误处理**:
- RPC `response` 类型有 `success: false; error: string` 字段(`rpc-types.ts` line 231)
- 非 RPC 模式:exit code + stderr(pi main.ts 标准 Node.js process.exit)
- `extension_error` 事件(扩展运行时错误,非致命)
- 无 `result` type 终止行(用 `agent_settled` 代替)

**pi 生命周期**:
- **per-prompt**(print/json 模式):等同 rick 现状,单次 prompt → 单次进程 → 退出
- **RPC 长连接**(`--mode rpc`):单进程,stdin/stdout JSONL 双向,生命周期由 client 控制(rick 启动 pi rpc 进程后,可发多次 prompt / steer / follow_up 命令,无需反复启动)
- **session resume**:`--session <id>` 加载已有会话(树结构),`--continue` 续最近,`--fork <id>` 分叉
- **无心跳**(RPC 模式靠 stdin/stdout pipe 保活,pipe 断 = 进程退出)

### 运行时行为(权重 0.3)✅

**13 处调用点映射可行性**:

| # | rick 现状 | pi 映射 | 重构难度 |
|---|---|---|---|
| 1 | `claude -p --output-format stream-json --verbose --dangerously-skip-permissions <file>` | `pi --mode json <file>` 或 `pi --mode rpc` + prompt 命令 | 中(解析器重写,字段 schema 不同) |
| 2 | `claude <file>`(interactive) | `pi <file>` | 低(flag 删除) |
| 3 | `claude -p --dangerously-skip-permissions <file>` | `pi -p <file>` | 低(去 flag) |
| 4 | `claude --resume <id>` | `pi --continue` 或 `pi --resume` 或 `pi --session <id>` | 低(重命名) |
| 5 | `claude --session-id <id> <file>` | `pi --session <id> <file>` | 低(重命名) |
| 6 | `claude <file>`(learning interactive) | `pi <file>` | 低(flag 删除) |
| 7 | `claude -p --dangerously-skip-permissions <file>`(dream bg) | `pi -p <file>` | 低(去 flag) |
| 8 | `claude <file>`(dream interactive) | `pi <file>` | 低(flag 删除) |
| 9 | `claude <mainFile>`(human-loop) | `pi <mainFile>` | 低(flag 删除) |
| 10 | `claude <file>`(ctrl) | `pi <file>` | 低(flag 删除) |
| 11 | `claude --dangerously-skip-permissions <file>`(plan check autofix) | `pi -p <file>` | 低(去 flag) |
| 12 | `claude --dangerously-skip-permissions <file>`(runner.go 备用) | `pi -p <file>` | 低(去 flag) |
| 13 | `claudecode.NewExecutor` → 走 #1 | `piagent.NewExecutor` → 走 #1(pi 侧) | 中(新解析器) |

**统计**:8 处低难度(flag 重命名/删除)+ 2 处中难度(解析器重写:#1 + #13,实际是同一个 doing.go 调用链)+ 3 处低难度(interactive 直接换 binary)

**关键差异点**:
- pi 无 `--dangerously-skip-permissions`(默认无 permission popup,Philosophy 段明示)
- pi 无 `--output-format stream-json`(用 `--mode json` 替代)
- pi 无 `--session-id`(用 `--session` 替代,语义更广:接受 path 或 id)
- pi 有 `--mode rpc`(rick 现状无对应,可选增强)
- pi 有 `--fork`(rick 现状无对应,可选增强)
- pi 有 `--system-prompt` / `--append-system-prompt`(rick 现状通过 prompt 文件注入,可选增强)

### 文档(权重 0.2)✅

- `cli/args.ts` Args interface 列出全部 flag,注释清晰
- `main.ts` resolveAppMode 明确 4 模式路由逻辑
- `rpc-types.ts` 30+ RPC 命令 + 20+ 事件类型完整 schema
- `agent-loop.ts` line 109-446 事件 emit 时机完整
- `print-mode.ts` 注释明示 `-p` 和 `--mode json` 的关系
- 无独立"协议文档"(rpc.md 在 docs/ 下,但本轮直接读源码 schema 更精确)

### 反事实(权重 0.1)N/A

- 本节点为外部代码调研,无 rick 代码修改

## 还原确认

无 rick 代码修改,无需还原。

## 关键事实

1. **pi CLI 调用方式**:4 种模式 `interactive` / `print` / `json` / `rpc`,前 3 种 per-prompt 子进程(同构 rick 现状),第 4 种 RPC 长连接(rick 现状无对应,可选增强)
2. **pi 数据格式**:
   - print 模式:纯文本 stdout(一次性最终结果)
   - json 模式:JSONL over stdout(首行 session header,后续 events,等同 claude stream-json 但字段不同)
   - rpc 模式:JSONL 双向(stdin 命令 / stdout 响应+事件)
3. **字段映射**:5 项关键字段全部不对齐(snake_case vs camelCase)+ duration_ms 缺失(需 rick 自计时)
4. **pi 错误处理**:RPC `response` 有 `success:false + error:string`;非 RPC 用 exit code + stderr;无 `result` type 终止行(用 `agent_settled` 事件)
5. **pi 生命周期**:per-prompt(默认)或 RPC 长连接(可选);session resume 通过 `--session` / `--continue` / `--fork`;无心跳
6. **pi flag 映射**:
   - `--dangerously-skip-permissions` → 删除(pi 默认无 permission)
   - `--output-format stream-json` → `--mode json`
   - `--resume` / `--session-id` → `--continue` / `--session`
   - `-p` / `--verbose` → 同名保留
7. **适配层设计**:
   - 新建 `internal/agent/piagent/executor.go` 实现 `AgentExecutor` 接口
   - PiExecutor.Execute 调用 `pi --mode json <promptFile>` 或 `pi --mode rpc` + prompt 命令
   - 新建 pi 事件流解析器(对标 `claudecode.parseStream`),字段映射:
     - `agent_settled` → 终止信号(替代 claude `result` type)
     - `tool_execution_end`(toolCallId + toolName + result + isError) → ToolCall{Output, IsError}
     - `message_end` message.content[type=text].text → FinalMessage
     - session header `id` → sessionID
     - 自维护 startTime → agent_settled → Duration()(替代 duration_ms)
   - 12 处直接 exec.Command:8 处纯 flag 适配层(plan/easy/dream/learning/human_loop/ctrl/tools_plan_check/runner),4 处 interactive 模式只需换 binary 路径

## 疑问点

- rick 是否要将 12 处直接 exec.Command 全部重构为走 AgentExecutor 接口?(架构决策,非事实调研)
- pi RPC 长连接模式下,rick 如何管理 pi 进程生命周期(启动/超时/崩溃恢复/复用)?(实现细节,非事实调研)
- rick 现有 `--resume` 语义(续接会话)与 pi `--continue` 是否完全等价?pi 树结构会话是否影响 rick 线性会话假设?(语义调研:pi `--session <id>` 加载完整会话,树结构兼容线性假设,但 `--fork` 是新能力)

## 置信度评估(由 research 主调度计算)

- 代码原文 ✅ × 0.4 = 0.4
- 运行时行为 ✅ × 0.3 = 0.3
- 文档 ✅ × 0.2 = 0.2
- 反事实 N/A × 0.1 = 0
- 合计 = 0.9(高,≥ 0.8 终止)
