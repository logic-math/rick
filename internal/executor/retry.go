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

// RetryTask executes a task with retry logic
// It retries failed tasks up to MaxRetries times, loading debug context on each retry
// Returns RetryResult with execution status and debug information
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

	// Retry loop
	for attempt := 1; attempt <= maxRetries; attempt++ {
		result.TotalAttempts = attempt

		// Load debug context from debug.md if it exists
		debugContext := ""
		if trm.debugFile != "" {
			debugContext = trm.loadDebugContext(trm.debugFile)
		}

		// Execute the task
		execResult, err := trm.runner.RunTask(task)
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

		// Task failed, prepare for retry
		result.LastError = execResult.Error

		// Record failure to debug.md if file is specified
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
// Format: "debug_N: [现象], [复现], [猜想], [验证], [修复], [进展]"
func (trm *TaskRetryManager) buildDebugEntry(task *parser.Task, attempt int, maxRetries int, result *TaskExecutionResult, previousContext string) string {
	// Extract error information
	phenomenon := result.Error
	if phenomenon == "" {
		phenomenon = "task execution failed"
	}

	// Build reproduction steps
	reproduction := fmt.Sprintf("Attempt %d of %d: %s", attempt, maxRetries, task.Name)

	// Build hypothesis (guesses about the cause)
	hypothesis := trm.analyzeError(result.Error, result.Output)

	// Build verification steps
	verification := fmt.Sprintf("Review output and error logs from attempt %d", attempt)

	// Build fix description (placeholder - to be filled by human)
	fix := "待人工审查"

	// Build progress status
	progress := "未解决"
	if attempt == maxRetries {
		progress = "超过重试限制，需要人工干预"
	}

	// Format as debug entry
	debugNum := trm.getNextDebugNumber(previousContext)
	entry := fmt.Sprintf("- debug%d: %s, %s, %s, %s, %s, %s",
		debugNum, phenomenon, reproduction, hypothesis, verification, fix, progress)

	return entry
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
		if strings.Contains(line, "- debug") {
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
		hypotheses = append(hypotheses, "1)执行超时 2)资源不足 3)外部依赖不可用")
	} else if strings.Contains(errMsg, "not found") {
		hypotheses = append(hypotheses, "1)文件不存在 2)命令找不到 3)依赖缺失")
	} else if strings.Contains(errMsg, "permission") {
		hypotheses = append(hypotheses, "1)权限不足 2)文件权限错误 3)目录访问被拒")
	} else if strings.Contains(errMsg, "connection") {
		hypotheses = append(hypotheses, "1)网络连接失败 2)服务不可用 3)DNS 解析失败")
	} else if strings.Contains(errMsg, "script execution failed") {
		hypotheses = append(hypotheses, "1)脚本语法错误 2)环境变量缺失 3)依赖命令不可用")
	} else {
		hypotheses = append(hypotheses, "1)脚本执行异常 2)测试逻辑错误 3)环境配置问题")
	}

	// Analyze output for additional clues
	if strings.Contains(output, "FAIL") {
		hypotheses = append(hypotheses, "4)测试断言失败")
	}
	if strings.Contains(output, "ERROR") {
		hypotheses = append(hypotheses, "4)运行时错误")
	}

	if len(hypotheses) == 0 {
		return "1)未知错误 2)环境问题 3)脚本问题"
	}

	return hypotheses[0]
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
	}

	// Append new entry with newline
	if content != "" && !strings.HasSuffix(content, "\n") {
		content += "\n"
	}
	content += entry + "\n"

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
