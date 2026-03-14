# 失败重试模式 - 实现细节

## 核心数据结构

### RetryResult 结构
```go
type RetryResult struct {
    TaskID         string      // 任务 ID
    TaskName       string      // 任务名称
    Status         string      // success, failed, max_retries_exceeded
    TotalAttempts  int         // 总尝试次数
    LastError      string      // 最后一次错误信息
    Output         string      // 最后一次输出
    DebugLogsAdded []string    // 添加的调试日志列表
    StartTime      time.Time   // 开始时间
    EndTime        time.Time   // 结束时间
}

// Duration 返回总执行时间
func (rr *RetryResult) Duration() time.Duration {
    return rr.EndTime.Sub(rr.StartTime)
}
```

### TaskRetryManager 结构
```go
type TaskRetryManager struct {
    runner    *TaskRunner      // 任务执行器
    config    *ExecutionConfig // 执行配置（包含 MaxRetries）
    debugFile string           // debug.md 文件路径
}
```

## 重试逻辑实现

### 完整代码
```go
package executor

import (
    "fmt"
    "os"
    "path/filepath"
    "strings"
    "time"
)

// NewTaskRetryManager 创建 TaskRetryManager 实例
func NewTaskRetryManager(runner *TaskRunner, config *ExecutionConfig, debugFile string) *TaskRetryManager {
    return &TaskRetryManager{
        runner:    runner,
        config:    config,
        debugFile: debugFile,
    }
}

// RetryTask 执行任务，包含重试逻辑
// 流程：
// 1. 生成测试脚本（一次性，在重试循环外）
// 2. 重试循环：加载 debug.md -> 执行任务 -> 运行测试 -> 更新 debug.md（如果失败）
func (trm *TaskRetryManager) RetryTask(task *parser.Task) (*RetryResult, error) {
    if task == nil {
        return nil, fmt.Errorf("task cannot be nil")
    }

    if trm.config == nil {
        return nil, fmt.Errorf("execution config is required")
    }

    result := &RetryResult{
        TaskID:        task.ID,
        TaskName:      task.Name,
        Status:        "running",
        TotalAttempts: 0,
        StartTime:     time.Now(),
    }

    maxRetries := trm.config.MaxRetries
    if maxRetries <= 0 {
        maxRetries = 5 // 默认 5 次
    }

    var lastExecResult *TaskExecutionResult

    // 重试循环
    for attempt := 1; attempt <= maxRetries; attempt++ {
        result.TotalAttempts = attempt

        // 加载 debug.md 的历史上下文
        debugContext := trm.loadDebugContext(trm.debugFile)

        // 执行任务（包含调用 Claude Code + 运行测试）
        execResult, err := trm.runner.RunTask(task, debugContext)
        if err != nil {
            lastExecResult = execResult
            continue
        }

        lastExecResult = execResult

        // 检查是否成功
        if execResult.Status == "success" {
            result.Status = "success"
            result.Output = execResult.Output
            result.EndTime = time.Now()
            return result, nil
        }

        // 任务失败，记录到 debug.md
        result.LastError = execResult.Error

        if trm.debugFile != "" {
            debugEntry := trm.buildDebugEntry(task, attempt, maxRetries, execResult, debugContext)
            if err := trm.appendToDebugFile(debugEntry); err != nil {
                fmt.Fprintf(os.Stderr, "warning: failed to write debug log: %v\n", err)
            } else {
                result.DebugLogsAdded = append(result.DebugLogsAdded, debugEntry)
            }
        }

        // 如果不是最后一次尝试，继续重试
        if attempt < maxRetries {
            // 渐进式延迟：第 1 次重试延迟 1 秒，第 2 次延迟 2 秒，...
            time.Sleep(time.Duration(attempt) * time.Second)
            continue
        }
    }

    // 超过最大重试次数
    result.Status = "max_retries_exceeded"
    result.Output = lastExecResult.Output
    result.LastError = fmt.Sprintf("task failed after %d attempts: %s", maxRetries, result.LastError)
    result.EndTime = time.Now()

    return result, nil
}
```

## 关键实现细节

### 1. 加载调试上下文
```go
// loadDebugContext 读取 debug.md 文件内容
// 这为 AI Agent 提供了历史失败的上下文
func (trm *TaskRetryManager) loadDebugContext(debugFile string) string {
    if debugFile == "" {
        return ""
    }

    content, err := os.ReadFile(debugFile)
    if err != nil {
        // 文件可能不存在（第一次执行），这是正常的
        return ""
    }

    return string(content)
}
```

### 2. 构建调试日志
```go
// buildDebugEntry 构建结构化的调试日志条目
// 格式：现象 -> 复现 -> 猜想 -> 验证 -> 修复 -> 进展 -> 输出
func (trm *TaskRetryManager) buildDebugEntry(
    task *parser.Task,
    attempt int,
    maxRetries int,
    result *TaskExecutionResult,
    previousContext string,
) string {
    var entry strings.Builder

    // 获取 debug 编号（自动递增）
    debugNum := trm.getNextDebugNumber(previousContext)

    entry.WriteString(fmt.Sprintf("\n## debug%d: Task %s - Attempt %d/%d\n\n",
        debugNum, task.ID, attempt, maxRetries))

    // 现象（发生了什么）
    entry.WriteString("**现象 (Phenomenon)**:\n")
    if result.Error != "" {
        entry.WriteString(fmt.Sprintf("- %s\n", result.Error))
    } else {
        entry.WriteString("- Task execution failed without specific error\n")
    }
    entry.WriteString("\n")

    // 复现（如何复现）
    entry.WriteString("**复现 (Reproduction)**:\n")
    entry.WriteString(fmt.Sprintf("- Task: %s\n", task.Name))
    entry.WriteString(fmt.Sprintf("- Goal: %s\n", task.Goal))
    entry.WriteString(fmt.Sprintf("- Attempt: %d of %d\n", attempt, maxRetries))
    entry.WriteString("\n")

    // 猜想（错误原因的假设）
    entry.WriteString("**猜想 (Hypothesis)**:\n")
    hypotheses := trm.analyzeError(result.Error, result.Output)
    entry.WriteString(fmt.Sprintf("- %s\n", hypotheses))
    entry.WriteString("\n")

    // 验证（如何验证假设）
    entry.WriteString("**验证 (Verification)**:\n")
    entry.WriteString("- Review the output below\n")
    entry.WriteString("- Check if files were created/modified as expected\n")
    entry.WriteString("- Verify test script logic is correct\n")
    entry.WriteString("\n")

    // 修复（如何修复）
    entry.WriteString("**修复 (Fix)**:\n")
    if attempt == maxRetries {
        entry.WriteString("- ⚠️ Max retries exceeded - manual intervention required\n")
        entry.WriteString("- Review task.md and test method\n")
        entry.WriteString("- Update task requirements if needed\n")
    } else {
        entry.WriteString("- Will retry with updated context\n")
        entry.WriteString("- Agent should learn from this failure\n")
    }
    entry.WriteString("\n")

    // 进展（当前状态）
    entry.WriteString("**进展 (Progress)**:\n")
    if attempt == maxRetries {
        entry.WriteString("- Status: ❌ 未解决 - 超过重试限制\n")
    } else {
        entry.WriteString(fmt.Sprintf("- Status: 🔄 重试中 - Attempt %d/%d\n",
            attempt, maxRetries))
    }
    entry.WriteString("\n")

    // 输出（参考信息）
    entry.WriteString("**输出 (Output)**:\n")
    entry.WriteString("```\n")
    if result.Output != "" {
        // 限制输出长度，避免 debug.md 过大
        output := result.Output
        if len(output) > 1000 {
            output = output[:1000] + "\n... (truncated)"
        }
        entry.WriteString(output)
    } else {
        entry.WriteString("(no output)")
    }
    entry.WriteString("\n```\n")

    return entry.String()
}
```

### 3. 错误分析
```go
// analyzeError 分析错误消息和输出，生成假设
func (trm *TaskRetryManager) analyzeError(errMsg string, output string) string {
    hypotheses := []string{}

    // 分析错误消息
    if strings.Contains(errMsg, "timeout") {
        hypotheses = append(hypotheses, "执行超时 - 可能是任务太复杂或资源不足")
    } else if strings.Contains(errMsg, "not found") || strings.Contains(errMsg, "does not exist") {
        hypotheses = append(hypotheses, "文件或资源不存在 - 可能是路径错误或文件未创建")
    } else if strings.Contains(errMsg, "permission") {
        hypotheses = append(hypotheses, "权限不足 - 需要检查文件/目录权限")
    } else if strings.Contains(errMsg, "connection") {
        hypotheses = append(hypotheses, "网络连接失败 - 检查网络或服务可用性")
    } else if strings.Contains(errMsg, "test did not pass") {
        hypotheses = append(hypotheses, "测试未通过 - 任务执行结果不符合预期")
    } else if strings.Contains(errMsg, "failed to generate test script") {
        hypotheses = append(hypotheses, "测试脚本生成失败 - 检查测试方法定义")
    } else {
        hypotheses = append(hypotheses, "未知错误 - 需要详细分析输出日志")
    }

    // 分析输出中的线索
    if strings.Contains(output, "FAIL") {
        hypotheses = append(hypotheses, "测试断言失败")
    }
    if strings.Contains(output, "ERROR") {
        hypotheses = append(hypotheses, "运行时错误")
    }
    if strings.Contains(output, "SyntaxError") {
        hypotheses = append(hypotheses, "Python语法错误")
    }
    if strings.Contains(output, "ImportError") || strings.Contains(output, "ModuleNotFoundError") {
        hypotheses = append(hypotheses, "缺少Python模块依赖")
    }

    if len(hypotheses) == 0 {
        return "未知错误 - 需要人工分析"
    }

    return strings.Join(hypotheses, "; ")
}
```

### 4. 追加调试日志
```go
// appendToDebugFile 追加调试条目到 debug.md
// 如果文件不存在，会自动创建
func (trm *TaskRetryManager) appendToDebugFile(entry string) error {
    if trm.debugFile == "" {
        return fmt.Errorf("debug file path is not set")
    }

    // 确保目录存在
    debugDir := filepath.Dir(trm.debugFile)
    if err := os.MkdirAll(debugDir, 0755); err != nil {
        return fmt.Errorf("failed to create debug directory: %w", err)
    }

    // 读取现有内容
    var content string
    if fileInfo, err := os.Stat(trm.debugFile); err == nil && fileInfo.Size() > 0 {
        data, err := os.ReadFile(trm.debugFile)
        if err != nil {
            return fmt.Errorf("failed to read debug file: %w", err)
        }
        content = string(data)
    } else {
        // 文件不存在，创建初始头部
        content = "# Debug Log\n\n"
        content += "This file contains debugging information for failed task executions.\n\n"
    }

    // 追加新条目
    if !strings.HasSuffix(content, "\n") {
        content += "\n"
    }
    content += entry

    // 写回文件
    if err := os.WriteFile(trm.debugFile, []byte(content), 0644); err != nil {
        return fmt.Errorf("failed to write debug file: %w", err)
    }

    return nil
}
```

### 5. 获取下一个 debug 编号
```go
// getNextDebugNumber 从现有上下文中提取最大的 debug 编号
func (trm *TaskRetryManager) getNextDebugNumber(context string) int {
    if context == "" {
        return 1
    }

    // 查找最高的 debug 编号
    maxNum := 0
    lines := strings.Split(context, "\n")
    for _, line := range lines {
        if strings.Contains(line, "## debug") {
            // 提取 debug 编号
            parts := strings.Split(line, "debug")
            if len(parts) > 1 {
                numStr := strings.TrimSpace(strings.Split(parts[1], ":")[0])
                var num int
                fmt.Sscanf(numStr, "%d", &num)
                if num > maxNum {
                    maxNum = num
                }
            }
        }
    }

    return maxNum + 1
}
```

## 使用示例

### 基本用法
```go
package main

import (
    "fmt"
    "github.com/sunquan/rick/internal/executor"
    "github.com/sunquan/rick/internal/parser"
)

func main() {
    // 1. 创建任务
    task := &parser.Task{
        ID:   "task1",
        Name: "创建输出文件",
        Goal: "生成 output.txt 文件",
    }

    // 2. 创建任务执行器
    runner := executor.NewTaskRunner()

    // 3. 创建执行配置
    config := &executor.ExecutionConfig{
        MaxRetries: 5, // 最多重试 5 次
    }

    // 4. 创建重试管理器
    debugFile := ".rick/jobs/job_1/doing/debug.md"
    manager := executor.NewTaskRetryManager(runner, config, debugFile)

    // 5. 执行任务（自动重试）
    result, err := manager.RetryTask(task)
    if err != nil {
        fmt.Printf("Error: %v\n", err)
        return
    }

    // 6. 打印结果
    fmt.Printf("Task: %s\n", result.TaskID)
    fmt.Printf("Status: %s\n", result.Status)
    fmt.Printf("Attempts: %d\n", result.TotalAttempts)
    fmt.Printf("Duration: %v\n", result.Duration())

    if result.Status != "success" {
        fmt.Printf("Last Error: %s\n", result.LastError)
    }
}
```

### 简化用法（一次性）
```go
// 如果不需要复用 TaskRetryManager，可以使用便捷函数
result, err := executor.RetryTaskSimple(task, runner, config, debugFile)
```

## 最佳实践

### 1. 合理设置重试次数
```go
// 对于 AI Agent 任务，5 次通常足够
config := &ExecutionConfig{
    MaxRetries: 5,
}

// 对于网络请求，3 次可能就够了
config := &ExecutionConfig{
    MaxRetries: 3,
}
```

### 2. 使用渐进式延迟
```go
// 每次重试之间增加延迟时间
// 第 1 次重试：1 秒
// 第 2 次重试：2 秒
// 第 3 次重试：3 秒
// ...
time.Sleep(time.Duration(attempt) * time.Second)
```

### 3. 限制输出长度
```go
// 避免 debug.md 过大
output := result.Output
if len(output) > 1000 {
    output = output[:1000] + "\n... (truncated)"
}
```

### 4. 清晰的错误分类
```go
// 根据错误类型给出针对性的建议
if strings.Contains(errMsg, "timeout") {
    return "执行超时 - 可能是任务太复杂或资源不足"
} else if strings.Contains(errMsg, "not found") {
    return "文件或资源不存在 - 可能是路径错误或文件未创建"
}
```

## 测试用例

### 测试1: 成功重试
```go
func TestRetryTask_Success(t *testing.T) {
    task := &parser.Task{
        ID:   "task1",
        Name: "Test Task",
        Goal: "Complete successfully",
    }

    runner := NewMockTaskRunner(2) // 第 2 次尝试成功
    config := &ExecutionConfig{MaxRetries: 5}
    manager := NewTaskRetryManager(runner, config, "debug.md")

    result, err := manager.RetryTask(task)
    assert.NoError(t, err)
    assert.Equal(t, "success", result.Status)
    assert.Equal(t, 2, result.TotalAttempts)
}
```

### 测试2: 超过重试限制
```go
func TestRetryTask_MaxRetriesExceeded(t *testing.T) {
    task := &parser.Task{
        ID:   "task1",
        Name: "Test Task",
        Goal: "Always fail",
    }

    runner := NewMockTaskRunner(-1) // 永远失败
    config := &ExecutionConfig{MaxRetries: 3}
    manager := NewTaskRetryManager(runner, config, "debug.md")

    result, err := manager.RetryTask(task)
    assert.NoError(t, err)
    assert.Equal(t, "max_retries_exceeded", result.Status)
    assert.Equal(t, 3, result.TotalAttempts)
}
```

### 测试3: debug.md 生成
```go
func TestRetryTask_DebugLogGeneration(t *testing.T) {
    task := &parser.Task{
        ID:   "task1",
        Name: "Test Task",
        Goal: "Fail and log",
    }

    runner := NewMockTaskRunner(-1)
    config := &ExecutionConfig{MaxRetries: 2}
    debugFile := "test_debug.md"
    defer os.Remove(debugFile)

    manager := NewTaskRetryManager(runner, config, debugFile)
    result, err := manager.RetryTask(task)
    assert.NoError(t, err)

    // 检查 debug.md 文件是否生成
    content, err := os.ReadFile(debugFile)
    assert.NoError(t, err)
    assert.Contains(t, string(content), "## debug1:")
    assert.Contains(t, string(content), "## debug2:")
}
```

## 扩展: 高级重试策略

### 指数退避
```go
// 每次重试之间的延迟呈指数增长
func exponentialBackoff(attempt int) time.Duration {
    baseDelay := 1 * time.Second
    maxDelay := 32 * time.Second
    delay := baseDelay * time.Duration(1<<uint(attempt-1))
    if delay > maxDelay {
        delay = maxDelay
    }
    return delay
}

// 使用
time.Sleep(exponentialBackoff(attempt))
```

### 带抖动的退避
```go
// 添加随机抖动，避免多个客户端同时重试
func backoffWithJitter(attempt int) time.Duration {
    delay := exponentialBackoff(attempt)
    jitter := time.Duration(rand.Int63n(int64(delay / 2)))
    return delay + jitter
}
```

### 断路器模式集成
```go
// 如果连续失败次数过多，暂时停止重试
type CircuitBreaker struct {
    failures   int
    threshold  int
    resetTime  time.Time
}

func (cb *CircuitBreaker) ShouldRetry() bool {
    if time.Now().Before(cb.resetTime) {
        return false // 断路器打开，拒绝重试
    }
    return true
}
```

---

*参考: Rick CLI `internal/executor/retry.go`*
*最后更新: 2026-03-14*
