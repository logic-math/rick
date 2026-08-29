# 依赖关系
无

# 写域
internal/runtime/rpc.go
internal/runtime/rpc_test.go

# 任务目标
实现 pi rpc 子进程的 JSONL 协议客户端（纯协议层，不含进程管理——supervisor 在 task4）。

# 关键结果
1. `internal/runtime/rpc.go`：`type RpcClient` 封装 JSONL 命令构建与事件行解析：
   - 命令构建方法：`Prompt(msg)`（含 streamingBehavior 支持 steer/followUp）、`Steer(msg)`、`FollowUp(msg)`、`Abort()`、`NewSession()`、`GetState()`、`GetMessages()`、`GetEntries(since)`、`SwitchSession(path)`、`SetSessionName(name)`、`SendUIResponse(id, kind, value)`（value/confirmed/cancelled 三态）、`GetSessionStats()`——每个方法返回 `(json.RawMessage, error)` 行字节（LF 结尾），由调用方写入 stdin
   - 事件解析：`ParseEventLine(line []byte) (*RpcEvent, error)`——RPC 事件（`type: response|agent_start|agent_end|agent_settled|turn_start|turn_end|message_start|message_update|message_end|tool_execution_start|tool_execution_end|extension_ui_request|session_start|session_end|bash_execution_update|compaction_*`）解为结构体（Type/ID/SessionID/Message/ToolCallID/ToolName/Result/IsError/Data 原始保留）；严格 LF 分帧提示（注释说明 Node readline 不合规历史，Go bufio.Scanner 默认按 LF 即合规）
   - `BuildCommand(type string, payload map[string]any) ([]byte, error)` 通用命令构建（带自增请求 id）
   - 协议依据：pi 官方 rpc.md（本地路径 /home/hadoop-recsys/.rick/pi/agent/runtime/node_modules/@earendil-works/pi-coding-agent/docs/rpc.md，必须先读）；参考 ygncode/pi-web internal/rpc/client.go 模式（调研简报 research-L2.md）
2. `internal/runtime/rpc_test.go`：表驱动测试覆盖——每命令的 JSON 输出精确断言（字段名/嵌套）；事件行解析（各 type 至少 1 例 + 非法 JSON 报错）；参考 rick 现有 executor.go 测试风格（camelCase 字段）
3. `go build ./...` 与 `go test ./internal/runtime/ -run TestRpc -timeout 60s` 通过

# 测试方法
go test ./internal/runtime/ -run TestRpc -v（表驱动，全绿）；go vet ./internal/runtime/

# 上下文提示
- pi rpc 事件字段是 camelCase（sessionId/toolCallId）——与现有 executor.go 的 json 模式一致，可参考其解析风格但 rpc 事件面更宽（rpc.md Event Types 节全量核对）
- ⚠️ bugs.md 坑位：`--session`=加载已有、`--session-id`=创建新（本 task 只做协议层不涉 spawn，但注释里要写清，供 task4/sessions 使用）
