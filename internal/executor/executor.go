package executor

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/sunquan/rick/internal/parser"
)

// ExecutionJobResult represents the result of executing a job
type ExecutionJobResult struct {
	JobID              string
	Status             string // completed, failed, partial
	TotalTasks         int
	SuccessfulTasks    int
	FailedTasks        int
	StartTime          time.Time
	EndTime            time.Time
	TaskResults        []*RetryResult
	ExecutionLog       string
	ErrorSummary       string
}

// Duration returns the total execution duration
func (ejr *ExecutionJobResult) Duration() time.Duration {
	return ejr.EndTime.Sub(ejr.StartTime)
}

// Executor manages the execution of all tasks in a job
type Executor struct {
	dag              *DAG
	tasksJSON        *TasksJSON
	sortedTaskIDs    []string
	runner           *TaskRunner
	retryManager     *TaskRetryManager
	config           *ExecutionConfig
	executionLog     []string
	jobID            string
	workspaceDir     string
	tasksJSONPath    string
}

// NewExecutor creates a new Executor instance
func NewExecutor(
	tasks []*parser.Task,
	config *ExecutionConfig,
	workspaceDir string,
	jobID string,
) (*Executor, error) {
	if len(tasks) == 0 {
		return nil, fmt.Errorf("tasks list cannot be empty")
	}
	if config == nil {
		return nil, fmt.Errorf("execution config cannot be nil")
	}
	if workspaceDir == "" {
		return nil, fmt.Errorf("workspace directory cannot be empty")
	}

	// Build DAG from tasks
	dag, err := NewDAG(tasks)
	if err != nil {
		return nil, fmt.Errorf("failed to build DAG: %w", err)
	}

	// Perform topological sort
	sortedTaskIDs, err := TopologicalSort(dag)
	if err != nil {
		return nil, fmt.Errorf("failed to perform topological sort: %w", err)
	}

	// Generate tasks.json
	tasksJSON, err := GenerateTasksJSON(dag, sortedTaskIDs)
	if err != nil {
		return nil, fmt.Errorf("failed to generate tasks.json: %w", err)
	}

	// Create task runner
	runner := NewTaskRunner(config)

	// Prepare paths
	tasksJSONPath := filepath.Join(workspaceDir, "tasks.json")

	// Create retry manager
	debugFile := filepath.Join(workspaceDir, "debug.md")
	retryManager := NewTaskRetryManager(runner, config, debugFile)

	executor := &Executor{
		dag:           dag,
		tasksJSON:     tasksJSON,
		sortedTaskIDs: sortedTaskIDs,
		runner:        runner,
		retryManager:  retryManager,
		config:        config,
		executionLog:  []string{},
		jobID:         jobID,
		workspaceDir:  workspaceDir,
		tasksJSONPath: tasksJSONPath,
	}

	return executor, nil
}

// ExecuteJob executes all tasks in the job
// It performs serial execution based on topological order
// Returns ExecutionJobResult with overall status and per-task results
func (e *Executor) ExecuteJob() (*ExecutionJobResult, error) {
	startTime := time.Now()
	result := &ExecutionJobResult{
		JobID:       e.jobID,
		StartTime:   startTime,
		TotalTasks:  len(e.sortedTaskIDs),
		TaskResults: make([]*RetryResult, 0),
	}

	e.logf("Starting job execution: %s", e.jobID)
	e.logf("Total tasks: %d", result.TotalTasks)
	e.logf("Task execution order: %v", e.sortedTaskIDs)

	// Save initial tasks.json
	if err := SaveTasksJSON(e.tasksJSONPath, e.tasksJSON); err != nil {
		e.logf("ERROR: Failed to save initial tasks.json: %v", err)
		result.Status = "failed"
		result.ErrorSummary = fmt.Sprintf("Failed to save initial tasks.json: %v", err)
		result.EndTime = time.Now()
		return result, err
	}

	// Execute each task in order
	for i, taskID := range e.sortedTaskIDs {
		e.logf("[%d/%d] Executing task: %s", i+1, result.TotalTasks, taskID)

		// Update task status to running
		if err := e.tasksJSON.UpdateTaskStatus(taskID, "running"); err != nil {
			e.logf("ERROR: Failed to update task status to running: %v", err)
		}
		if err := SaveTasksJSON(e.tasksJSONPath, e.tasksJSON); err != nil {
			e.logf("ERROR: Failed to save tasks.json: %v", err)
		}

		// Get the task from DAG
		task, exists := e.dag.Tasks[taskID]
		if !exists {
			e.logf("ERROR: Task not found in DAG: %s", taskID)
			if err := e.tasksJSON.UpdateTaskStatusWithError(taskID, "failed", "Task not found in DAG"); err != nil {
				e.logf("ERROR: Failed to update task status: %v", err)
			}
			result.FailedTasks++
			continue
		}

		// Execute task with retry logic
		retryResult, err := e.retryManager.RetryTask(task)
		if err != nil {
			e.logf("ERROR: Failed to execute task: %v", err)
			retryResult = &RetryResult{
				TaskID:        taskID,
				TaskName:      task.Name,
				Status:        "failed",
				LastError:     fmt.Sprintf("Execution error: %v", err),
				StartTime:     time.Now(),
				EndTime:       time.Now(),
			}
		}

		result.TaskResults = append(result.TaskResults, retryResult)

		// Update task status based on result
		if retryResult.Status == "success" {
			e.logf("✓ Task succeeded: %s (attempts: %d)", taskID, retryResult.TotalAttempts)
			if err := e.tasksJSON.UpdateTaskStatus(taskID, "success"); err != nil {
				e.logf("ERROR: Failed to update task status: %v", err)
			}
			result.SuccessfulTasks++
		} else {
			e.logf("✗ Task failed: %s (status: %s, attempts: %d)", taskID, retryResult.Status, retryResult.TotalAttempts)
			if err := e.tasksJSON.UpdateTaskStatusWithError(taskID, "failed", retryResult.LastError); err != nil {
				e.logf("ERROR: Failed to update task status: %v", err)
			}
			result.FailedTasks++
		}

		// Save updated tasks.json
		if err := SaveTasksJSON(e.tasksJSONPath, e.tasksJSON); err != nil {
			e.logf("ERROR: Failed to save tasks.json: %v", err)
		}
	}

	// Determine overall status
	if result.FailedTasks == 0 {
		result.Status = "completed"
		e.logf("✓ Job execution completed successfully")
	} else if result.SuccessfulTasks > 0 {
		result.Status = "partial"
		e.logf("⚠ Job execution completed with partial success")
	} else {
		result.Status = "failed"
		e.logf("✗ Job execution failed")
	}

	result.EndTime = time.Now()
	result.ExecutionLog = e.getExecutionLog()

	// Generate error summary if there are failures
	if result.FailedTasks > 0 {
		result.ErrorSummary = e.generateErrorSummary(result.TaskResults)
	}

	return result, nil
}

// logf logs a message with timestamp
func (e *Executor) logf(format string, args ...interface{}) {
	timestamp := time.Now().Format("2006-01-02 15:04:05")
	message := fmt.Sprintf("[%s] %s", timestamp, fmt.Sprintf(format, args...))
	e.executionLog = append(e.executionLog, message)
	fmt.Println(message)
}

// getExecutionLog returns the full execution log as a string
func (e *Executor) getExecutionLog() string {
	result := ""
	for _, line := range e.executionLog {
		result += line + "\n"
	}
	return result
}

// generateErrorSummary generates a summary of errors from task results
func (e *Executor) generateErrorSummary(taskResults []*RetryResult) string {
	summary := "Failed Tasks Summary:\n"
	for _, tr := range taskResults {
		if tr.Status != "success" {
			summary += fmt.Sprintf("- %s (%s): %s (attempts: %d)\n",
				tr.TaskID, tr.TaskName, tr.LastError, tr.TotalAttempts)
		}
	}
	return summary
}

// GetTasksJSON returns the current tasks.json
func (e *Executor) GetTasksJSON() *TasksJSON {
	return e.tasksJSON
}

// GetDAG returns the DAG
func (e *Executor) GetDAG() *DAG {
	return e.dag
}

// GetSortedTaskIDs returns the task IDs in topological order
func (e *Executor) GetSortedTaskIDs() []string {
	return e.sortedTaskIDs
}

// SaveExecutionLog saves the execution log to a file
func (e *Executor) SaveExecutionLog(logFilePath string) error {
	if logFilePath == "" {
		return fmt.Errorf("log file path cannot be empty")
	}

	// Create directory if it doesn't exist
	dir := filepath.Dir(logFilePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Write log to file
	logContent := e.getExecutionLog()
	if err := os.WriteFile(logFilePath, []byte(logContent), 0644); err != nil {
		return fmt.Errorf("failed to write execution log: %w", err)
	}

	return nil
}
