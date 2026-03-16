package executor

import (
	"fmt"
	"os"
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
	var testErrorFeedback string // Accumulate test errors for feedback

	// Retry loop - this implements the "while not pass" logic
	for attempt := 1; attempt <= maxRetries; attempt++ {
		result.TotalAttempts = attempt

		// Load debug context from debug.md if it exists
		debugContext := trm.loadDebugContext(trm.debugFile)

		// Execute the task with debug context and test error feedback
		// This will:
		// 1. Generate doing prompt with task.md + debug.md + test errors + OKR.md + SPEC.md
		// 2. Call Claude to execute the task (may fix test script if needed)
		// 3. Run the test script
		// 4. Return pass/fail result
		execResult, err := trm.runner.RunTask(task, debugContext, testErrorFeedback)
		if err != nil {
			lastExecResult = execResult
			// Accumulate test error for next retry
			if execResult != nil && execResult.Error != "" {
				testErrorFeedback = fmt.Sprintf("Attempt %d: %s\n%s", attempt, execResult.Error, testErrorFeedback)
			}
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

		// Task failed, record error
		result.LastError = execResult.Error
		// Note: debug.md is now managed by Claude, not by the program

		// Accumulate test error feedback for next retry
		// This allows Claude to see the history of test failures and fix the test script
		if execResult.Error != "" {
			testErrorFeedback = fmt.Sprintf("Attempt %d: %s\n%s", attempt, execResult.Error, testErrorFeedback)
		}
		if execResult.Output != "" {
			// Include test output for context (limit to 500 chars to avoid bloat)
			output := execResult.Output
			if len(output) > 500 {
				output = output[:500] + "... (truncated)"
			}
			testErrorFeedback += fmt.Sprintf("\nOutput:\n%s\n", output)
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

// Note: Debug logging is now handled by Claude, not by the program
// The program only loads debug.md and passes it as context

// RetryTaskSimple is a convenience function that creates a TaskRetryManager and retries a task
// It's useful for simple retry operations without managing a separate manager instance
func RetryTaskSimple(task *parser.Task, runner *TaskRunner, config *ExecutionConfig, debugFile string) (*RetryResult, error) {
	manager := NewTaskRetryManager(runner, config, debugFile)
	return manager.RetryTask(task)
}
