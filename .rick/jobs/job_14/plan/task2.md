# 依赖关系

task1

# 任务名称

实现 Claude Code 适配器（NDJSON 解析 + raw_session 双写）

# 任务目标

实现 `internal/agent/claudecode/executor.go`，将 `claude -p --output-format stream-json --verbose` 的流式 NDJSON 输出解析为 `AgentSession`，同时双写原始日志到 `raw_session.log`。

## 关键事实（实测验证）

`--output-format stream-json` **必须加 `--verbose`**，否则直接报错退出：
`Error: When using --print, --output-format=stream-json requires --verbose`

实际 NDJSON 格式（tool_use **嵌套在 message.content[] 内**，不在顶层）：
```
{"type":"system","session_id":"xxx",...}
{"type":"assistant","message":{"content":[{"type":"tool_use","id":"t1","name":"Bash","input":{...}}]},"session_id":"xxx"}
{"type":"user","message":{"content":[{"type":"tool_result","tool_use_id":"t1","content":"ok","is_error":false}]},"session_id":"xxx"}
{"type":"assistant","message":{"content":[{"type":"text","text":"done."}]},"session_id":"xxx"}
{"type":"result","is_error":false,"duration_ms":10864,"session_id":"xxx"}
```

# 关键结果

1. **`internal/agent/claudecode/executor.go`**：
   - 结构体：`ClaudeCodeExecutor{ claudePath string }`，构造函数 `NewExecutor(claudePath string)`
   - 实现 `AgentExecutor` 接口：`Execute(promptFile, taskID string) (agent.AgentSession, error)`
   - 执行流程：
     1. `os.MkdirAll("doing/tasks/{taskID}/", 0755)`
     2. 打开 `doing/tasks/{taskID}/raw_session.log` 写入流（追加模式）
     3. 启动：`claude -p --output-format stream-json --verbose --dangerously-skip-permissions {promptFile}`
     4. `bufio.Scanner` 逐行读取 stdout，**每行先写入 raw_session.log**（`lineNo++`），再解析：
        - `type=system` / `type=result`：提取顶层 `session_id`；`type=result` 还提取 `duration_ms`、`is_error`
        - `type=assistant` → 遍历 `message.content[]`：
          - `type=="tool_use"` → 追加 ToolCall（Name/Input JSON 截断 300 字，记录当前 `lineNo`）
          - `type=="text"` → 覆盖 `finalMessage`（截断 200 字），记录 `finalMessageLine=lineNo`
        - `type=user` → 遍历 `message.content[]`：
          - `type=="tool_result"` → 更新最后一个 ToolCall.Output（截断 300 字）；`is_error==true` 标记 IsError
        - 非 JSON 行：`log.Printf("warn: skip non-json line %d: %s", lineNo, line[:min(60,len)])`，继续，不 panic
     5. 返回 `&claudeSession{...}`（实现 `agent.AgentSession` 接口）

2. **`claudeSession` struct**（文件内私有）：
   - 字段：sessionID、toolCalls、finalMessage、finalMessageLine、rawLogPath、duration、errorCount
   - `GetRawLogPath()` 返回 raw_session.log 绝对路径
   - `GetErrorCount()` 统计 toolCalls 中 IsError==true 的数量

3. **`internal/agent/claudecode/executor_test.go`**：
   - `TestExecute_ParseNDJSON`：构造符合真实格式的 mock NDJSON（含 system/tool_use/tool_result/text/result 各行），通过 pipe 模拟 stdout 注入，验证：
     - sessionID 正确提取
     - ToolCalls 长度和 Name/IsError 字段正确
     - FinalMessage 不超过 200 字
     - raw_session.log 文件存在，每行均可 `json.Unmarshal`
     - FinalMessageLine 与实际 NDJSON 行号一致
   - `TestExecute_SkipNonJSON`：输入中混入非 JSON 行，验证不 panic，其他字段正常解析，非 JSON 行不出现在 raw_session.log 的解析结果中（但仍被写入 raw 文件）

# 测试方法

1. 编译：`go build ./internal/agent/claudecode/...`，无报错
2. **`TestExecute_ParseNDJSON` 断言（明确）**：
   - session_id == "mock-session-001"
   - ToolCalls 长度 == 1，Name == "Bash"，IsError == false
   - FinalMessage == "done."（≤200字）
   - FinalMessageLine == 4（text 行是第 4 行）
   - `raw_session.log` 存在，每行 `json.Unmarshal` 成功，共 5 行
   ```
   go test ./internal/agent/claudecode/... -v -run TestExecute_ParseNDJSON
   ```
3. **`TestExecute_SkipNonJSON` 断言**：
   - 输入中第 3 行为纯文本 `"not json"`，不 panic
   - ToolCalls 正常解析，sessionID 正常提取
   - raw_session.log 存在（非 JSON 行也被写入）
   ```
   go test ./internal/agent/claudecode/... -v -run TestExecute_SkipNonJSON
   ```
4. `go test ./internal/agent/claudecode/... -v`，全部通过
