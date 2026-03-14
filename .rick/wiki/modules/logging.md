# Logging 模块详解

> 日志系统模块 - 基于 Go 标准库的简化日志系统

## 📋 模块概述

Logging 模块是 Rick CLI 的日志系统，遵循"简化设计"原则，仅使用 Go 标准库 `log` 包，提供四个日志级别（INFO、WARN、ERROR、DEBUG）的文本格式日志。

### 功能职责
- 提供分级日志记录（INFO/WARN/ERROR/DEBUG）
- 支持输出到标准输出或文件
- 自动添加时间戳和日志级别前缀
- 最小化外部依赖

### 模块位置
```
internal/logging/
└── logger.go       # 日志器实现
```

---

## 🏗️ 核心类型和接口

### Logger 结构

```go
type Logger struct {
    infoLogger  *log.Logger  // INFO 级别日志器
    warnLogger  *log.Logger  // WARN 级别日志器
    errorLogger *log.Logger  // ERROR 级别日志器
    debugLogger *log.Logger  // DEBUG 级别日志器
}
```

**设计特点**：
- 每个级别使用独立的 `log.Logger` 实例
- 不同级别有不同的前缀（`[INFO]`, `[WARN]`, `[ERROR]`, `[DEBUG]`）
- 统一使用 `log.LstdFlags` 时间戳格式

---

## 🔧 主要函数说明

### 1. 创建日志器

#### `NewLogger() *Logger`

创建输出到标准输出的日志器。

**默认行为**：
- 输出到 `os.Stdout`
- 包含时间戳（日期 + 时间）
- 自动添加级别前缀

**示例**：
```go
logger := logging.NewLogger()
logger.Info("Application started")
// 输出: [INFO] 2026/03/14 10:30:45 Application started
```

#### `NewLoggerWithWriter(w io.Writer) *Logger`

创建输出到自定义 Writer 的日志器。

**参数**：
- `w io.Writer`: 任何实现了 `io.Writer` 接口的对象

**使用场景**：
- 输出到文件
- 输出到缓冲区（测试）
- 输出到网络连接

**示例 1: 输出到缓冲区（测试）**
```go
var buf bytes.Buffer
logger := logging.NewLoggerWithWriter(&buf)
logger.Info("Test message")
fmt.Println(buf.String())
// 输出: [INFO] 2026/03/14 10:30:45 Test message
```

**示例 2: 输出到多个目标**
```go
// 同时输出到标准输出和文件
file, _ := os.Create("app.log")
multiWriter := io.MultiWriter(os.Stdout, file)
logger := logging.NewLoggerWithWriter(multiWriter)
logger.Info("Logged to both stdout and file")
```

#### `NewLoggerWithFile(filepath string) (*Logger, error)`

创建输出到文件的日志器。

**参数**：
- `filepath string`: 日志文件路径

**文件模式**：
- `os.O_CREATE`: 文件不存在则创建
- `os.O_WRONLY`: 只写模式
- `os.O_APPEND`: 追加模式（不覆盖现有内容）
- 权限: `0644` (rw-r--r--)

**错误处理**：
- 文件打开失败
- 目录不存在（需要先创建目录）

**示例**：
```go
logger, err := logging.NewLoggerWithFile("/var/log/rick.log")
if err != nil {
    log.Fatalf("Failed to create logger: %v", err)
}
logger.Info("Application started")
// 写入到 /var/log/rick.log
```

**注意事项**：
- 文件句柄不会自动关闭
- 需要确保目录存在
- 建议使用绝对路径

### 2. 日志记录方法

#### `Info(format string, args ...interface{})`

记录 INFO 级别日志。

**用途**：
- 正常操作信息
- 重要步骤完成
- 配置加载成功

**示例**：
```go
logger.Info("Starting job execution")
logger.Info("Loaded %d tasks", taskCount)
logger.Info("Job %s completed successfully", jobID)
```

#### `Warn(format string, args ...interface{})`

记录 WARN 级别日志。

**用途**：
- 潜在问题
- 非致命错误
- 降级操作

**示例**：
```go
logger.Warn("Configuration file not found, using defaults")
logger.Warn("Task %s took longer than expected: %v", taskID, duration)
logger.Warn("Retry attempt %d/%d", attempt, maxRetries)
```

#### `Error(format string, args ...interface{})`

记录 ERROR 级别日志。

**用途**：
- 错误信息
- 操作失败
- 异常情况

**示例**：
```go
logger.Error("Failed to load tasks: %v", err)
logger.Error("Task %s execution failed: %v", taskID, err)
logger.Error("Git commit failed: %v", err)
```

#### `Debug(format string, args ...interface{})`

记录 DEBUG 级别日志。

**用途**：
- 调试信息
- 详细的执行步骤
- 变量值跟踪

**示例**：
```go
logger.Debug("Entering function: executeTask")
logger.Debug("Task dependencies: %v", task.Dependencies)
logger.Debug("Config loaded: %+v", cfg)
```

---

## 💡 使用示例

### 示例 1: 基本使用

```go
package main

import (
    "github.com/sunquan/rick/internal/logging"
)

func main() {
    // 创建日志器
    logger := logging.NewLogger()

    // 记录不同级别的日志
    logger.Info("Application started")
    logger.Warn("Configuration file not found, using defaults")
    logger.Error("Failed to connect to database")
    logger.Debug("Variable value: x=%d", 42)
}
```

**输出**：
```
[INFO] 2026/03/14 10:30:45 Application started
[WARN] 2026/03/14 10:30:45 Configuration file not found, using defaults
[ERROR] 2026/03/14 10:30:45 Failed to connect to database
[DEBUG] 2026/03/14 10:30:45 Variable value: x=42
```

### 示例 2: 输出到文件

```go
package main

import (
    "log"
    "os"
    "path/filepath"
    "github.com/sunquan/rick/internal/logging"
)

func main() {
    // 确保日志目录存在
    logDir := "/var/log/rick"
    if err := os.MkdirAll(logDir, 0755); err != nil {
        log.Fatal(err)
    }

    // 创建文件日志器
    logFile := filepath.Join(logDir, "rick.log")
    logger, err := logging.NewLoggerWithFile(logFile)
    if err != nil {
        log.Fatalf("Failed to create logger: %v", err)
    }

    // 使用日志器
    logger.Info("Application started")
    logger.Info("Log file: %s", logFile)
}
```

### 示例 3: 结构化日志（自定义）

```go
package main

import (
    "fmt"
    "github.com/sunquan/rick/internal/logging"
)

type StructuredLogger struct {
    logger *logging.Logger
    fields map[string]interface{}
}

func NewStructuredLogger() *StructuredLogger {
    return &StructuredLogger{
        logger: logging.NewLogger(),
        fields: make(map[string]interface{}),
    }
}

func (sl *StructuredLogger) WithField(key string, value interface{}) *StructuredLogger {
    sl.fields[key] = value
    return sl
}

func (sl *StructuredLogger) Info(msg string) {
    fieldsStr := ""
    for k, v := range sl.fields {
        fieldsStr += fmt.Sprintf(" %s=%v", k, v)
    }
    sl.logger.Info("%s%s", msg, fieldsStr)
}

func main() {
    logger := NewStructuredLogger()
    logger.WithField("job_id", "job_1").
           WithField("task_id", "task1").
           Info("Task execution started")
}
```

**输出**：
```
[INFO] 2026/03/14 10:30:45 Task execution started job_id=job_1 task_id=task1
```

### 示例 4: 条件日志（带详细模式）

```go
package main

import (
    "github.com/sunquan/rick/internal/logging"
)

type Application struct {
    logger  *logging.Logger
    verbose bool
}

func (app *Application) Run() {
    app.logger.Info("Application started")

    if app.verbose {
        app.logger.Debug("Verbose mode enabled")
        app.logger.Debug("Loading configuration...")
    }

    // 业务逻辑
    app.processTask("task1")

    app.logger.Info("Application completed")
}

func (app *Application) processTask(taskID string) {
    if app.verbose {
        app.logger.Debug("Processing task: %s", taskID)
    }

    // 处理任务
    app.logger.Info("Task %s completed", taskID)
}

func main() {
    app := &Application{
        logger:  logging.NewLogger(),
        verbose: true, // 启用详细模式
    }
    app.Run()
}
```

### 示例 5: 错误日志与恢复

```go
package main

import (
    "fmt"
    "github.com/sunquan/rick/internal/logging"
)

func main() {
    logger := logging.NewLogger()

    defer func() {
        if r := recover(); r != nil {
            logger.Error("Panic recovered: %v", r)
        }
    }()

    logger.Info("Starting risky operation")

    // 可能 panic 的操作
    riskyOperation()

    logger.Info("Operation completed")
}

func riskyOperation() {
    // 模拟错误
    panic("something went wrong")
}
```

---

## ❓ 常见问题

### Q1: 如何控制日志级别？

**A**: 当前实现不支持动态日志级别控制。可以通过条件判断实现：

```go
type LogLevel int

const (
    DEBUG LogLevel = iota
    INFO
    WARN
    ERROR
)

type LeveledLogger struct {
    logger *logging.Logger
    level  LogLevel
}

func (ll *LeveledLogger) Debug(format string, args ...interface{}) {
    if ll.level <= DEBUG {
        ll.logger.Debug(format, args...)
    }
}

func (ll *LeveledLogger) Info(format string, args ...interface{}) {
    if ll.level <= INFO {
        ll.logger.Info(format, args...)
    }
}
```

### Q2: 如何关闭日志文件？

**A**: 当前实现不提供文件关闭接口。需要手动管理：

```go
// 自定义实现
type FileLogger struct {
    logger *logging.Logger
    file   *os.File
}

func NewFileLogger(path string) (*FileLogger, error) {
    file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
    if err != nil {
        return nil, err
    }

    return &FileLogger{
        logger: logging.NewLoggerWithWriter(file),
        file:   file,
    }, nil
}

func (fl *FileLogger) Close() error {
    return fl.file.Close()
}

// 使用
func main() {
    logger, _ := NewFileLogger("app.log")
    defer logger.Close()

    logger.logger.Info("Application started")
}
```

### Q3: 如何实现日志轮转？

**A**: 当前实现不支持日志轮转。可以使用外部工具或自定义实现：

**方法 1: 使用 logrotate（Linux）**
```bash
# /etc/logrotate.d/rick
/var/log/rick/*.log {
    daily
    rotate 7
    compress
    missingok
    notifempty
}
```

**方法 2: 自定义实现**
```go
type RotatingLogger struct {
    logger      *logging.Logger
    file        *os.File
    maxSize     int64
    currentSize int64
}

func (rl *RotatingLogger) Write(p []byte) (n int, err error) {
    rl.currentSize += int64(len(p))
    if rl.currentSize > rl.maxSize {
        rl.rotate()
    }
    return rl.file.Write(p)
}

func (rl *RotatingLogger) rotate() {
    // 实现日志轮转逻辑
}
```

### Q4: 如何在测试中捕获日志？

**A**: 使用 `bytes.Buffer` 作为 Writer：

```go
func TestLogging(t *testing.T) {
    var buf bytes.Buffer
    logger := logging.NewLoggerWithWriter(&buf)

    logger.Info("Test message")

    output := buf.String()
    if !strings.Contains(output, "Test message") {
        t.Errorf("Expected log output to contain 'Test message', got: %s", output)
    }
}
```

### Q5: 如何添加自定义字段（如请求 ID）？

**A**: 创建包装器：

```go
type ContextLogger struct {
    logger    *logging.Logger
    requestID string
}

func (cl *ContextLogger) Info(format string, args ...interface{}) {
    msg := fmt.Sprintf(format, args...)
    cl.logger.Info("[%s] %s", cl.requestID, msg)
}

// 使用
func handleRequest(requestID string) {
    logger := &ContextLogger{
        logger:    logging.NewLogger(),
        requestID: requestID,
    }
    logger.Info("Processing request")
    // 输出: [INFO] 2026/03/14 10:30:45 [req-123] Processing request
}
```

---

## 🔗 相关模块

- [Executor 模块](./dag_executor.md) - 使用日志记录任务执行
- [CMD 模块](./cli_commands.md) - 使用日志记录命令执行
- [Git 模块](./git.md) - 使用日志记录 Git 操作

---

## 📚 设计原则

### 1. 简化优先
遵循 Rick CLI 的"简化设计"原则：
- 仅使用 Go 标准库 `log` 包
- 不引入第三方日志框架（如 logrus、zap）
- 文本格式，易于阅读和解析

### 2. 分级日志
提供四个标准日志级别：
- **DEBUG**: 详细调试信息
- **INFO**: 正常操作信息
- **WARN**: 警告信息
- **ERROR**: 错误信息

### 3. 时间戳优先
所有日志自动包含时间戳（`log.LstdFlags`）：
```
[INFO] 2026/03/14 10:30:45 Message
```

### 4. 可扩展性
设计简单但可扩展：
- 支持自定义 Writer
- 可以包装实现高级功能
- 保持核心简单

---

## 🎯 最佳实践

### 1. 日志级别选择

**DEBUG**: 开发调试
```go
logger.Debug("Variable value: x=%d", x)
logger.Debug("Entering function: processTask")
```

**INFO**: 重要操作
```go
logger.Info("Application started")
logger.Info("Task %s completed", taskID)
logger.Info("Job execution finished in %v", duration)
```

**WARN**: 潜在问题
```go
logger.Warn("Configuration file not found, using defaults")
logger.Warn("Retry attempt %d/%d", attempt, maxRetries)
```

**ERROR**: 错误情况
```go
logger.Error("Failed to load tasks: %v", err)
logger.Error("Database connection failed: %v", err)
```

### 2. 日志消息格式

**推荐**：
```go
logger.Info("Task %s completed successfully", taskID)
logger.Error("Failed to execute task %s: %v", taskID, err)
```

**不推荐**：
```go
logger.Info("Task completed")  // 缺少上下文
logger.Error("Error: %v", err) // 缺少操作描述
```

### 3. 错误日志

始终包含错误详情：
```go
if err := doSomething(); err != nil {
    logger.Error("Failed to do something: %v", err)
    return err
}
```

### 4. 性能考虑

避免在循环中频繁记录日志：
```go
// 不推荐
for _, item := range items {
    logger.Debug("Processing item: %v", item)
    process(item)
}

// 推荐
logger.Debug("Processing %d items", len(items))
for _, item := range items {
    process(item)
}
logger.Debug("Processed %d items", len(items))
```

---

## 🔮 未来扩展

### 可能的增强功能

1. **日志级别控制**
```go
type Logger struct {
    level LogLevel
    // ...
}

func (l *Logger) SetLevel(level LogLevel) {
    l.level = level
}
```

2. **结构化日志**
```go
func (l *Logger) InfoWithFields(msg string, fields map[string]interface{}) {
    // 输出: [INFO] 2026/03/14 10:30:45 msg field1=value1 field2=value2
}
```

3. **日志轮转**
```go
type RotatingLogger struct {
    maxSize int64
    maxAge  time.Duration
    // ...
}
```

4. **异步日志**
```go
type AsyncLogger struct {
    logChan chan logEntry
    // ...
}
```

---

*最后更新: 2026-03-14*
