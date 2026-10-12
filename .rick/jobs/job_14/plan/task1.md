# 依赖关系


# 任务名称

定义 AgentSession/AgentExecutor 稳定接口与 act-path 生成器

# 任务目标

建立 agent 执行层的抽象契约和 act-path 生成逻辑，不依赖任何具体 agent 实现：

```
internal/agent/
├── interface.go    # AgentSession、AgentExecutor、ToolCall（稳定契约）
└── session.go      # ToolCall struct 定义（供 claudecode 包复用）

internal/actpath/
└── generator.go    # Generate(session, outputFile) → act-path.md
```

# 关键结果

1. **`internal/agent/interface.go`**：
   ```go
   type ToolCall struct {
       Timestamp time.Time
       LineNo    int     // 在 raw_session.log 中的行号
       Name      string
       Input     string  // 截断 300 字
       Output    string  // 截断 300 字
       IsError   bool
   }
   type AgentSession interface {
       GetSessionID()        string
       GetToolCalls()        []ToolCall
       GetErrorCount()       int
       GetDuration()         time.Duration
       GetFinalMessage()     string  // assistant/text 最后一条，截断 200 字
       GetFinalMessageLine() int     // 该消息在 raw_session.log 中的行号
       GetRawLogPath()       string  // doing/tasks/{taskID}/raw_session.log 绝对路径
   }
   type AgentExecutor interface {
       Execute(promptFile string, taskID string) (AgentSession, error)
   }
   ```

2. **`internal/actpath/generator.go`**：
   - 签名：`func Generate(session agent.AgentSession, outputFile string) error`
   - **不 import `claudecode` 包**（`grep -r "claudecode" internal/actpath/` 应为空）
   - 输出格式：
     ```markdown
     ## 执行摘要
     - Session ID: {id}
     - 耗时: {duration}
     - 工具调用次数: {n}
     - 报错次数: {errorCount}
     - 完整日志: [raw_session.log]({rawLogPath})

     ## Agent 最终输出
     > {finalMessage}
     > ([完整内容 raw_session.log:{line}]({rawLogPath}))

     ## 行为轨迹
     | 行号 | 工具 | 输入摘要 | 输出摘要 | 是否报错 |
     |------|------|----------|----------|----------|
     | [L5]({rawLogPath}) | Bash | echo hello | hello world | ✅ |
     ```
   - outputFile 所在目录不存在时自动 `os.MkdirAll`

3. **`internal/actpath/generator_test.go`**：
   - 定义 `mockSession` struct 实现 `AgentSession` 接口（含 FinalMessage、RawLogPath、LineNo）
   - 验证生成的 act-path.md 包含"执行摘要"、行号链接格式 `[L{n}]`、finalMessage 截断到 200 字以内

# 测试方法

1. 编译：`go build ./internal/agent/... ./internal/actpath/...`，无报错
2. 接口隔离：`grep -r "claudecode" internal/actpath/` → 空
3. **generator 单元测试**（`generator_test.go` 中新增，明确断言）：
   - `TestGenerate_Format`：用 mockSession（2 个 ToolCall，1 个 IsError=true，FinalMessage="done"，FinalMessageLine=42，RawLogPath="/tmp/raw.log"），调用 Generate()，断言输出文件：
     - 包含 `"## 执行摘要"`、`"## 行为轨迹"`、`"## Agent 最终输出"`
     - 包含 `"报错次数: 1"`
     - 包含行号链接格式 `"[L"` （如 `[L5]`）
     - 包含 `"raw_session.log:42"` （FinalMessageLine 正确引用）
     - FinalMessage 截断：若原始超 200 字符，输出 ≤ 200 字符
   - `TestGenerate_EmptyToolCalls`：零 ToolCall 场景，行为轨迹表格仅含表头，不报错
   - `TestGenerate_CreatesDir`：outputFile 指向不存在的目录路径，验证目录被自动创建
   ```
   go test ./internal/actpath/... -v
   ```
