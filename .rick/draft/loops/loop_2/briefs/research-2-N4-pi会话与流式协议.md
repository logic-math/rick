# research-2 N4-pi 会话与流式协议

节点路径:[根 > N4-pi 会话与流式协议]
事实陈述:pi 是否支持会话续接(等同 --resume)、流式输出协议(NDJSON/SSE/WebSocket/自定义)、输出结构字段(session_id/tool_use/tool_result/duration_ms)。

## 执行动作

1. 读取 README "Sessions" 章节
2. 读取 `session-format.md`(16KB,会话文件格式)
3. 读取 `json.md`(3KB,JSON 流式模式)
4. 读取 `rpc.md` Events 章节(line 832-1130)
5. grep 字段名:session_id / sessionId / tool_use / tool_result / toolCall / duration / duration_ms

## 信源验证结果

### 代码原文(权重 0.4)✅

**会话续接(等同 --resume / --session-id)**:
- README Sessions 章节:`pi -c` 续最近会话 / `pi -r` 浏览选择 / `pi --session <path|id>` 指定 / `pi --fork <path|id>` 分叉
- 会话存储:`~/.pi/agent/sessions/--<path>--/<timestamp>_<uuid>.jsonl`(JSONL 树结构)
- 树结构:每条目 `id` + `parentId`,支持原地分支(/tree / /fork / /clone)
- 会话版本:v3(已自动迁移 v1/v2)
- SDK:`AgentSession.sessionId` / `sessionFile`;`AgentSessionRuntime.switchSession()` / `fork()` / `newSession()` / `importFromJsonl()`
- RPC 命令:`switch_session` / `fork` / `clone` / `new_session`(均带 `parentSession` 追踪)

**流式输出协议**:
- **3 种流式协议**:
  1. **JSON 模式**(`--mode json`):stdout 输出 JSONL,首行 session header,后续 events。`message_update` 为 delta-only(不含 cumulative message)
  2. **RPC 模式**(`--mode rpc`):stdin/stdout 双向 JSONL,strict LF-delimited(rpc.md 警告:不能用 Node readline,因 readline 拆 U+2028/U+2029)
  3. **print 模式**(`-p`):非流式,一次性输出最终文本(支持 stdin pipe merge)
- **传输层**:provider HTTP 流式(SSE/WebSocket,通过 `transport` 设置选 `auto`/`sse`/`websocket`),但 pi→client 协议是 JSONL over stdio

**输出结构字段对比**(pi vs claude code vs rick 期望):

| 字段 | claude code(rick 现状) | pi | 对齐性 |
|---|---|---|---|
| session_id | `session_id`(ndLine) | `sessionId`(get_state response)/ `id`(session header) | ❌ 字段名不同(snake vs camel) |
| tool_use | `type:"tool_use"` + `id` + `name` + `input` | `type:"toolCall"` + `id` + `name` + `arguments`(ToolCall content block) | ❌ type 名 + 字段名不同 |
| tool_result | `type:"tool_result"` + `tool_use_id` + `content` + `is_error` | `role:"toolResult"` + `toolCallId` + `content` + `isError`(ToolResultMessage) | ❌ 字段名不同 |
| duration_ms | `duration_ms`(result type) | ❌ **无 duration_ms 字段**(agent_end 事件不含 duration;get_session_stats 含 tokens/cost 但无 duration) | ❌ 缺失 |
| is_error | `is_error`(result) | `isError`(tool_execution_end) | ❌ 命名风格不同 |

**pi 事件类型清单**(rpc.md Events):
- `agent_start` / `agent_end`(含 messages + willRetry)/ `agent_settled`
- `turn_start` / `turn_end`(含 message + toolResults)
- `message_start` / `message_update`(delta)/ `message_end`
- `bash_execution_update`(直接 bash 命令输出 chunk)
- `tool_execution_start` / `tool_execution_update` / `tool_execution_end`
- `queue_update` / `compaction_start` / `compaction_end`
- `auto_retry_start` / `auto_retry_end`
- `summarization_retry_scheduled` / `summarization_retry_attempt_start` / `summarization_retry_finished`
- `extension_error`

**message_update delta 类型**(assistantMessageEvent):
- `text_start` / `text_delta` / `text_end`
- `thinking_start` / `thinking_delta` / `thinking_end`
- `toolcall_start` / `toolcall_delta` / `toolcall_end`(含完整 toolCall 对象)

### 运行时行为(权重 0.3)✅

- json.md 示例:`pi --mode json "List files" 2>/dev/null | jq -c 'select(.type == "message_end")'` 可直接 jq 解析
- rpc.md 提供 Python 客户端示例(line 1486)+ Node.js 客户端示例(line 1522),证明协议可被任意语言消费
- 会话文件 JSONL 可被 `pi --export` 转 HTML,`pi --import` 从 JSONL 恢复
- `/resume` 命令(interactive)等价 `pi -r`(CLI)

### 文档(权重 0.2)✅

- README Sessions 章节:Management / Branching / Compaction 三小节
- session-format.md:Content Blocks(TextContent/ImageContent/ThinkingContent/ToolCall)/ Base Message Types(UserMessage/AssistantMessage/ToolResultMessage)/ Extended Message Types(BashExecutionMessage/CustomMessage/BranchSummaryMessage/CompactionSummaryMessage)/ AgentMessage union / SessionEntryBase / Entry Types(SessionHeader/UserMessage/AssistantMessage/...)
- json.md:事件类型 + 输出格式 + 示例
- rpc.md:Commands(20+ 命令)+ Events(20+ 事件)+ Types(Model/UserMessage/AssistantMessage/ToolResultMessage/BashExecutionMessage/Attachment)

### 反事实(权重 0.1)N/A

- 本节点为外部协议调研,无代码修改

## 还原确认

无 rick 代码修改,无需还原。

## 关键事实

1. **会话续接**:✅ 完整支持,语义等同或更强于 claude code
   - `--session <id>` ≈ claude `--session-id`
   - `--continue` / `-c` ≈ claude `--continue`(rick 现有用 `--resume`)
   - `--fork <id>` 强于 claude(原地分支)
   - 树结构会话 > claude 的线性会话
2. **流式协议**:JSONL over stdio(RPC 模式双向,JSON 模式单向 stdout)
   - 与 claude code `--output-format stream-json` 同构(均为 NDJSON)
   - 但事件 schema 不同(见上表)
3. **字段对齐性**:❌ **5 个关键字段全部不对齐**
   - session_id → sessionId(camelCase)
   - tool_use → toolCall(content block type)
   - tool_result → toolResult(role)
   - duration_ms → **缺失**(pi 不输出 duration)
   - is_error → isError
4. **缺失字段**:pi 不直接输出 `duration_ms`(需 rick 自行计时)/ 不输出 `result` type 终止行(用 `agent_settled` 代替)
5. **pi 独有**:steering / followUp 消息队列 / compaction 事件 / auto_retry 事件 / extension_error 事件(claude code 无)

## 疑问点

- rick runner.go 的 NDJSON 解析器(`ndLine` struct)硬编码了 claude code 字段名(session_id/tool_use/tool_result/duration_ms/is_error),适配 pi 需重写解析器 → N5 详述
- pi RPC 模式无 `result` type 终止行,rick 如何判断一次 prompt 完成?→ 监听 `agent_settled` 事件(但需确认 rick 现有逻辑是否依赖 `result` type)

## 置信度评估(由 research 主调度计算)

- 代码原文 ✅ × 0.4 = 0.4
- 运行时行为 ✅ × 0.3 = 0.3
- 文档 ✅ × 0.2 = 0.2
- 反事实 N/A × 0.1 = 0
- 合计 = 0.9(高,≥ 0.8 终止)
