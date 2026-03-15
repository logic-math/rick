package executor

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/sunquan/rick/internal/parser"
)

// RetryResult represents the result of a retry operation
type RetryResult struct {
	TaskID         string
	TaskName       string
	Status         string    // success, failed, max_retries_exceeded
	TotalAttempts  int
	LastError      string
	Output         string
	DebugLogsAdded []string // List of debug entries added
	StartTime      time.Time
	EndTime        time.Time
}

// Duration returns the total execution duration
func (rr *RetryResult) Duration() time.Duration {
	return rr.EndTime.Sub(rr.StartTime)
}

// TaskRetryManager manages task retries with debug logging
type TaskRetryManager struct {
	runner    *TaskRunner
	config    *ExecutionConfig
	debugFile string
}

// NewTaskRetryManager creates a new TaskRetryManager instance
func NewTaskRetryManager(runner *TaskRunner, config *ExecutionConfig, debugFile string) *TaskRetryManager {
	return &TaskRetryManager{
		runner:    runner,
		config:    config,
		debugFile: debugFile,
	}
}

// RetryTask executes a task with retry logic following the new workflow:
// 1. Generate test script once (outside retry loop)
// 2. Retry loop: load debug context -> execute task -> run test -> update debug.md if failed
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
		maxRetries = 5 // Default to 5 retries
	}

	var lastExecResult *TaskExecutionResult

	// Retry loop - this implements the "while not pass" logic
	for attempt := 1; attempt <= maxRetries; attempt++ {
		result.TotalAttempts = attempt

		// Load debug context from debug.md if it exists
		debugContext := trm.loadDebugContext(trm.debugFile)

		// Execute the task with debug context
		// This will:
		// 1. Generate doing prompt with task.md + debug.md + OKR.md + SPEC.md
		// 2. Call Claude to execute the task
		// 3. Run the test script (already generated)
		// 4. Return pass/fail result
		execResult, err := trm.runner.RunTask(task, debugContext)
		if err != nil {
			lastExecResult = execResult
			// Continue to next retry
			continue
		}

		lastExecResult = execResult

		// Check if task succeeded
		if execResult.Status == "success" {
			result.Status = "success"
			result.Output = execResult.Output
			result.EndTime = time.Now()
			return result, nil
		}

		// Task failed, update debug.md
		result.LastError = execResult.Error

		// Record failure to debug.md
		if trm.debugFile != "" {
			debugEntry := trm.buildDebugEntry(task, attempt, maxRetries, execResult, debugContext)
			if err := trm.appendToDebugFile(debugEntry); err != nil {
				// Log error but continue with retry
				fmt.Fprintf(os.Stderr, "warning: failed to write debug log: %v\n", err)
			} else {
				result.DebugLogsAdded = append(result.DebugLogsAdded, debugEntry)
			}
		}

		// If this is not the last attempt, continue to next retry
		if attempt < maxRetries {
			// Optional: add delay between retries
			time.Sleep(time.Duration(attempt) * time.Second)
			continue
		}
	}

	// Max retries exceeded
	result.Status = "max_retries_exceeded"
	result.Output = lastExecResult.Output
	result.LastError = fmt.Sprintf("task failed after %d attempts: %s", maxRetries, result.LastError)
	result.EndTime = time.Now()

	return result, nil
}

// loadDebugContext reads the debug.md file and returns its content
// This provides context for retry decisions
func (trm *TaskRetryManager) loadDebugContext(debugFile string) string {
	if debugFile == "" {
		return ""
	}

	content, err := os.ReadFile(debugFile)
	if err != nil {
		// File might not exist yet, which is okay
		return ""
	}

	return string(content)
}

// buildDebugEntry constructs a debug log entry for a failed task
// Format follows the standard debug.md format with detailed error information
func (trm *TaskRetryManager) buildDebugEntry(task *parser.Task, attempt int, maxRetries int, result *TaskExecutionResult, previousContext string) string {
	var entry strings.Builder

	// Get debug number
	debugNum := trm.getNextDebugNumber(previousContext)

	entry.WriteString(fmt.Sprintf("\n## debug%d: Task %s - Attempt %d/%d\n\n", debugNum, task.ID, attempt, maxRetries))

	// Phenomenon (what happened)
	entry.WriteString("**现象 (Phenomenon)**:\n")
	if result.Error != "" {
		entry.WriteString(fmt.Sprintf("- %s\n", result.Error))
	} else {
		entry.WriteString("- Task execution failed without specific error\n")
	}
	entry.WriteString("\n")

	// Reproduction (how to reproduce)
	entry.WriteString("**复现 (Reproduction)**:\n")
	entry.WriteString(fmt.Sprintf("- Task: %s\n", task.Name))
	entry.WriteString(fmt.Sprintf("- Goal: %s\n", task.Goal))
	entry.WriteString(fmt.Sprintf("- Attempt: %d of %d\n", attempt, maxRetries))
	entry.WriteString("\n")

	// Hypothesis (guesses about the cause)
	entry.WriteString("**猜想 (Hypothesis)**:\n")
	hypotheses := trm.analyzeError(result.Error, result.Output)
	entry.WriteString(fmt.Sprintf("- %s\n", hypotheses))
	entry.WriteString("\n")

	// Verification (how to verify)
	entry.WriteString("**验证 (Verification)**:\n")
	entry.WriteString("- Review the output below\n")
	entry.WriteString("- Check if files were created/modified as expected\n")
	entry.WriteString("- Verify test script logic is correct\n")
	entry.WriteString("\n")

	// Fix (what needs to be fixed)
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

	// Progress (current status)
	entry.WriteString("**进展 (Progress)**:\n")
	if attempt == maxRetries {
		entry.WriteString("- Status: ❌ 未解决 - 超过重试限制\n")
	} else {
		entry.WriteString(fmt.Sprintf("- Status: 🔄 重试中 - Attempt %d/%d\n", attempt, maxRetries))
	}
	entry.WriteString("\n")

	// Output (for reference)
	entry.WriteString("**输出 (Output)**:\n")
	entry.WriteString("```\n")
	if result.Output != "" {
		// Limit output to first 1000 characters to avoid huge debug files
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

// getNextDebugNumber extracts the next debug number from previous context
func (trm *TaskRetryManager) getNextDebugNumber(context string) int {
	if context == "" {
		return 1
	}

	// Find the highest debug number in the context
	maxNum := 0
	lines := strings.Split(context, "\n")
	for _, line := range lines {
		if strings.Contains(line, "## debug") {
			// Try to extract debug number
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

// analyzeError analyzes the error and output to generate hypotheses
func (trm *TaskRetryManager) analyzeError(errMsg string, output string) string {
	hypotheses := []string{}

	// Analyze error message
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

	// Analyze output for additional clues
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

// appendToDebugFile appends a debug entry to the debug.md file
// Creates the file if it doesn't exist
func (trm *TaskRetryManager) appendToDebugFile(entry string) error {
	if trm.debugFile == "" {
		return fmt.Errorf("debug file path is not set")
	}

	// Ensure debug directory exists
	debugDir := filepath.Dir(trm.debugFile)
	if err := os.MkdirAll(debugDir, 0755); err != nil {
		return fmt.Errorf("failed to create debug directory: %w", err)
	}

	// Read existing content
	var content string
	if fileInfo, err := os.Stat(trm.debugFile); err == nil && fileInfo.Size() > 0 {
		data, err := os.ReadFile(trm.debugFile)
		if err != nil {
			return fmt.Errorf("failed to read debug file: %w", err)
		}
		content = string(data)
	} else {
		// Create initial header if file doesn't exist
		content = "# Debug Log\n\n"
		content += "This file contains debugging information for failed task executions.\n\n"
	}

	// Append new entry
	if !strings.HasSuffix(content, "\n") {
		content += "\n"
	}
	content += entry

	// Write back to file
	if err := os.WriteFile(trm.debugFile, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to write debug file: %w", err)
	}

	return nil
}

// RetryTaskSimple is a convenience function that creates a TaskRetryManager and retries a task
// It's useful for simple retry operations without managing a separate manager instance
func RetryTaskSimple(task *parser.Task, runner *TaskRunner, config *ExecutionConfig, debugFile string) (*RetryResult, error) {
	manager := NewTaskRetryManager(runner, config, debugFile)
	return manager.RetryTask(task)
}
