package executor

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/sunquan/rick/internal/parser"
)

// TestNewExecutor tests the creation of a new Executor
func TestNewExecutor(t *testing.T) {
	tasks := []*parser.Task{
		{
			ID:           "task1",
			Name:         "Task 1",
			Goal:         "Goal 1",
			KeyResults:   []string{"KR1"},
			TestMethod:   "Test 1",
			Dependencies: []string{},
		},
	}

	config := &ExecutionConfig{
		MaxRetries:     5,
		TimeoutSeconds: 30,
		LogFile:        "/tmp/test.log",
	}

	tmpDir := t.TempDir()

	executor, err := NewExecutor(tasks, config, tmpDir, "job1")
	if err != nil {
		t.Fatalf("Failed to create executor: %v", err)
	}

	if executor == nil {
		t.Fatal("Executor is nil")
	}

	if executor.jobID != "job1" {
		t.Errorf("Expected job ID 'job1', got '%s'", executor.jobID)
	}

	if executor.workspaceDir != tmpDir {
		t.Errorf("Expected workspace dir '%s', got '%s'", tmpDir, executor.workspaceDir)
	}

	if len(executor.sortedTaskIDs) != 1 {
		t.Errorf("Expected 1 sorted task, got %d", len(executor.sortedTaskIDs))
	}
}

// TestNewExecutorWithNilTasks tests error handling with nil tasks
func TestNewExecutorWithNilTasks(t *testing.T) {
	config := &ExecutionConfig{
		MaxRetries:     5,
		TimeoutSeconds: 30,
	}

	tmpDir := t.TempDir()

	_, err := NewExecutor(nil, config, tmpDir, "job1")
	if err == nil {
		t.Fatal("Expected error for nil tasks")
	}
}

// TestNewExecutorWithEmptyTasks tests error handling with empty tasks
func TestNewExecutorWithEmptyTasks(t *testing.T) {
	config := &ExecutionConfig{
		MaxRetries:     5,
		TimeoutSeconds: 30,
	}

	tmpDir := t.TempDir()

	_, err := NewExecutor([]*parser.Task{}, config, tmpDir, "job1")
	if err == nil {
		t.Fatal("Expected error for empty tasks")
	}
}

// TestNewExecutorWithNilConfig tests error handling with nil config
func TestNewExecutorWithNilConfig(t *testing.T) {
	tasks := []*parser.Task{
		{
			ID:           "task1",
			Name:         "Task 1",
			Goal:         "Goal 1",
			KeyResults:   []string{"KR1"},
			TestMethod:   "Test 1",
			Dependencies: []string{},
		},
	}

	tmpDir := t.TempDir()

	_, err := NewExecutor(tasks, nil, tmpDir, "job1")
	if err == nil {
		t.Fatal("Expected error for nil config")
	}
}

// TestNewExecutorWithEmptyWorkspaceDir tests error handling with empty workspace dir
func TestNewExecutorWithEmptyWorkspaceDir(t *testing.T) {
	tasks := []*parser.Task{
		{
			ID:           "task1",
			Name:         "Task 1",
			Goal:         "Goal 1",
			KeyResults:   []string{"KR1"},
			TestMethod:   "Test 1",
			Dependencies: []string{},
		},
	}

	config := &ExecutionConfig{
		MaxRetries:     5,
		TimeoutSeconds: 30,
	}

	_, err := NewExecutor(tasks, config, "", "job1")
	if err == nil {
		t.Fatal("Expected error for empty workspace dir")
	}
}

// TestExecutorWithMultipleTasks tests executor with multiple independent tasks
func TestExecutorWithMultipleTasks(t *testing.T) {
	tasks := []*parser.Task{
		{
			ID:           "task1",
			Name:         "Task 1",
			Goal:         "Goal 1",
			KeyResults:   []string{"KR1"},
			TestMethod:   "echo PASS",
			Dependencies: []string{},
		},
		{
			ID:           "task2",
			Name:         "Task 2",
			Goal:         "Goal 2",
			KeyResults:   []string{"KR2"},
			TestMethod:   "echo PASS",
			Dependencies: []string{},
		},
		{
			ID:           "task3",
			Name:         "Task 3",
			Goal:         "Goal 3",
			KeyResults:   []string{"KR3"},
			TestMethod:   "echo PASS",
			Dependencies: []string{},
		},
	}

	config := &ExecutionConfig{
		MaxRetries:     5,
		TimeoutSeconds: 30,
	}

	tmpDir := t.TempDir()

	executor, err := NewExecutor(tasks, config, tmpDir, "job_multi")
	if err != nil {
		t.Fatalf("Failed to create executor: %v", err)
	}

	if len(executor.sortedTaskIDs) != 3 {
		t.Errorf("Expected 3 sorted tasks, got %d", len(executor.sortedTaskIDs))
	}
}

// TestExecutorWithDependencies tests executor with task dependencies
func TestExecutorWithDependencies(t *testing.T) {
	tasks := []*parser.Task{
		{
			ID:           "task1",
			Name:         "Task 1",
			Goal:         "Goal 1",
			KeyResults:   []string{"KR1"},
			TestMethod:   "echo PASS",
			Dependencies: []string{},
		},
		{
			ID:           "task2",
			Name:         "Task 2",
			Goal:         "Goal 2",
			KeyResults:   []string{"KR2"},
			TestMethod:   "echo PASS",
			Dependencies: []string{"task1"},
		},
		{
			ID:           "task3",
			Name:         "Task 3",
			Goal:         "Goal 3",
			KeyResults:   []string{"KR3"},
			TestMethod:   "echo PASS",
			Dependencies: []string{"task2"},
		},
	}

	config := &ExecutionConfig{
		MaxRetries:     5,
		TimeoutSeconds: 30,
	}

	tmpDir := t.TempDir()

	executor, err := NewExecutor(tasks, config, tmpDir, "job_deps")
	if err != nil {
		t.Fatalf("Failed to create executor: %v", err)
	}

	// Verify topological order: task1 should come before task2, and task2 before task3
	if executor.sortedTaskIDs[0] != "task1" {
		t.Errorf("Expected first task to be 'task1', got '%s'", executor.sortedTaskIDs[0])
	}
	if executor.sortedTaskIDs[1] != "task2" {
		t.Errorf("Expected second task to be 'task2', got '%s'", executor.sortedTaskIDs[1])
	}
	if executor.sortedTaskIDs[2] != "task3" {
		t.Errorf("Expected third task to be 'task3', got '%s'", executor.sortedTaskIDs[2])
	}
}

// TestGetTasksJSON tests retrieving tasks.json
func TestGetTasksJSON(t *testing.T) {
	tasks := []*parser.Task{
		{
			ID:           "task1",
			Name:         "Task 1",
			Goal:         "Goal 1",
			KeyResults:   []string{"KR1"},
			TestMethod:   "Test 1",
			Dependencies: []string{},
		},
	}

	config := &ExecutionConfig{
		MaxRetries:     5,
		TimeoutSeconds: 30,
	}

	tmpDir := t.TempDir()

	executor, err := NewExecutor(tasks, config, tmpDir, "job1")
	if err != nil {
		t.Fatalf("Failed to create executor: %v", err)
	}

	tasksJSON := executor.GetTasksJSON()
	if tasksJSON == nil {
		t.Fatal("GetTasksJSON returned nil")
	}

	if len(tasksJSON.Tasks) != 1 {
		t.Errorf("Expected 1 task in JSON, got %d", len(tasksJSON.Tasks))
	}

	if tasksJSON.Tasks[0].TaskID != "task1" {
		t.Errorf("Expected task ID 'task1', got '%s'", tasksJSON.Tasks[0].TaskID)
	}

	if tasksJSON.Tasks[0].Status != "pending" {
		t.Errorf("Expected status 'pending', got '%s'", tasksJSON.Tasks[0].Status)
	}
}

// TestGetDAG tests retrieving DAG
func TestGetDAG(t *testing.T) {
	tasks := []*parser.Task{
		{
			ID:           "task1",
			Name:         "Task 1",
			Goal:         "Goal 1",
			KeyResults:   []string{"KR1"},
			TestMethod:   "Test 1",
			Dependencies: []string{},
		},
	}

	config := &ExecutionConfig{
		MaxRetries:     5,
		TimeoutSeconds: 30,
	}

	tmpDir := t.TempDir()

	executor, err := NewExecutor(tasks, config, tmpDir, "job1")
	if err != nil {
		t.Fatalf("Failed to create executor: %v", err)
	}

	dag := executor.GetDAG()
	if dag == nil {
		t.Fatal("GetDAG returned nil")
	}

	if len(dag.Tasks) != 1 {
		t.Errorf("Expected 1 task in DAG, got %d", len(dag.Tasks))
	}

	if _, exists := dag.Tasks["task1"]; !exists {
		t.Fatal("Task 'task1' not found in DAG")
	}
}

// TestGetSortedTaskIDs tests retrieving sorted task IDs
func TestGetSortedTaskIDs(t *testing.T) {
	tasks := []*parser.Task{
		{
			ID:           "task1",
			Name:         "Task 1",
			Goal:         "Goal 1",
			KeyResults:   []string{"KR1"},
			TestMethod:   "Test 1",
			Dependencies: []string{},
		},
		{
			ID:           "task2",
			Name:         "Task 2",
			Goal:         "Goal 2",
			KeyResults:   []string{"KR2"},
			TestMethod:   "Test 2",
			Dependencies: []string{"task1"},
		},
	}

	config := &ExecutionConfig{
		MaxRetries:     5,
		TimeoutSeconds: 30,
	}

	tmpDir := t.TempDir()

	executor, err := NewExecutor(tasks, config, tmpDir, "job1")
	if err != nil {
		t.Fatalf("Failed to create executor: %v", err)
	}

	sortedIDs := executor.GetSortedTaskIDs()
	if len(sortedIDs) != 2 {
		t.Errorf("Expected 2 sorted task IDs, got %d", len(sortedIDs))
	}

	if sortedIDs[0] != "task1" {
		t.Errorf("Expected first task ID to be 'task1', got '%s'", sortedIDs[0])
	}

	if sortedIDs[1] != "task2" {
		t.Errorf("Expected second task ID to be 'task2', got '%s'", sortedIDs[1])
	}
}

// TestSaveExecutionLog tests saving execution log to file
func TestSaveExecutionLog(t *testing.T) {
	tasks := []*parser.Task{
		{
			ID:           "task1",
			Name:         "Task 1",
			Goal:         "Goal 1",
			KeyResults:   []string{"KR1"},
			TestMethod:   "Test 1",
			Dependencies: []string{},
		},
	}

	config := &ExecutionConfig{
		MaxRetries:     5,
		TimeoutSeconds: 30,
	}

	tmpDir := t.TempDir()

	executor, err := NewExecutor(tasks, config, tmpDir, "job1")
	if err != nil {
		t.Fatalf("Failed to create executor: %v", err)
	}

	// Add some log entries
	executor.logf("Test log message 1")
	executor.logf("Test log message 2")

	// Save log
	logFilePath := filepath.Join(tmpDir, "execution.log")
	err = executor.SaveExecutionLog(logFilePath)
	if err != nil {
		t.Fatalf("Failed to save execution log: %v", err)
	}

	// Verify file exists
	if _, err := os.Stat(logFilePath); err != nil {
		t.Fatalf("Log file not found: %v", err)
	}

	// Verify content
	content, err := os.ReadFile(logFilePath)
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}

	if len(content) == 0 {
		t.Fatal("Log file is empty")
	}

	if !contains(string(content), "Test log message 1") {
		t.Fatal("Log file does not contain expected message 1")
	}

	if !contains(string(content), "Test log message 2") {
		t.Fatal("Log file does not contain expected message 2")
	}
}

// TestSaveExecutionLogWithEmptyPath tests error handling for empty log path
func TestSaveExecutionLogWithEmptyPath(t *testing.T) {
	tasks := []*parser.Task{
		{
			ID:           "task1",
			Name:         "Task 1",
			Goal:         "Goal 1",
			KeyResults:   []string{"KR1"},
			TestMethod:   "Test 1",
			Dependencies: []string{},
		},
	}

	config := &ExecutionConfig{
		MaxRetries:     5,
		TimeoutSeconds: 30,
	}

	tmpDir := t.TempDir()

	executor, err := NewExecutor(tasks, config, tmpDir, "job1")
	if err != nil {
		t.Fatalf("Failed to create executor: %v", err)
	}

	err = executor.SaveExecutionLog("")
	if err == nil {
		t.Fatal("Expected error for empty log path")
	}
}

// TestExecutionJobResultDuration tests ExecutionJobResult duration calculation
func TestExecutionJobResultDuration(t *testing.T) {
	result := &ExecutionJobResult{
		JobID:     "job1",
		StartTime: time.Now(),
		EndTime:   time.Now().Add(5 * time.Second),
	}

	duration := result.Duration()
	if duration < 5*time.Second || duration > 6*time.Second {
		t.Errorf("Expected duration around 5 seconds, got %v", duration)
	}
}

// TestExecutorLogf tests the logf method
func TestExecutorLogf(t *testing.T) {
	tasks := []*parser.Task{
		{
			ID:           "task1",
			Name:         "Task 1",
			Goal:         "Goal 1",
			KeyResults:   []string{"KR1"},
			TestMethod:   "Test 1",
			Dependencies: []string{},
		},
	}

	config := &ExecutionConfig{
		MaxRetries:     5,
		TimeoutSeconds: 30,
	}

	tmpDir := t.TempDir()

	executor, err := NewExecutor(tasks, config, tmpDir, "job1")
	if err != nil {
		t.Fatalf("Failed to create executor: %v", err)
	}

	executor.logf("Test message: %s", "hello")

	log := executor.getExecutionLog()
	if !contains(log, "Test message: hello") {
		t.Fatal("Log does not contain expected message")
	}

	if !contains(log, "[") || !contains(log, "]") {
		t.Fatal("Log does not contain timestamp")
	}
}

// TestExecutorWithComplexDependencies tests executor with complex dependency graph
func TestExecutorWithComplexDependencies(t *testing.T) {
	tasks := []*parser.Task{
		{
			ID:           "task1",
			Name:         "Task 1",
			Goal:         "Goal 1",
			KeyResults:   []string{"KR1"},
			TestMethod:   "echo PASS",
			Dependencies: []string{},
		},
		{
			ID:           "task2",
			Name:         "Task 2",
			Goal:         "Goal 2",
			KeyResults:   []string{"KR2"},
			TestMethod:   "echo PASS",
			Dependencies: []string{"task1"},
		},
		{
			ID:           "task3",
			Name:         "Task 3",
			Goal:         "Goal 3",
			KeyResults:   []string{"KR3"},
			TestMethod:   "echo PASS",
			Dependencies: []string{"task1"},
		},
		{
			ID:           "task4",
			Name:         "Task 4",
			Goal:         "Goal 4",
			KeyResults:   []string{"KR4"},
			TestMethod:   "echo PASS",
			Dependencies: []string{"task2", "task3"},
		},
	}

	config := &ExecutionConfig{
		MaxRetries:     5,
		TimeoutSeconds: 30,
	}

	tmpDir := t.TempDir()

	executor, err := NewExecutor(tasks, config, tmpDir, "job_complex")
	if err != nil {
		t.Fatalf("Failed to create executor: %v", err)
	}

	sortedIDs := executor.GetSortedTaskIDs()
	if len(sortedIDs) != 4 {
		t.Errorf("Expected 4 sorted tasks, got %d", len(sortedIDs))
	}

	// Verify topological order
	// task1 should come first
	if sortedIDs[0] != "task1" {
		t.Errorf("Expected first task to be 'task1', got '%s'", sortedIDs[0])
	}

	// task4 should come last (depends on task2 and task3)
	if sortedIDs[3] != "task4" {
		t.Errorf("Expected last task to be 'task4', got '%s'", sortedIDs[3])
	}
}

// TestExecutorDAGValidation tests that executor validates DAG correctness
func TestExecutorDAGValidation(t *testing.T) {
	// Create tasks with circular dependency
	tasks := []*parser.Task{
		{
			ID:           "task1",
			Name:         "Task 1",
			Goal:         "Goal 1",
			KeyResults:   []string{"KR1"},
			TestMethod:   "echo PASS",
			Dependencies: []string{"task2"},
		},
		{
			ID:           "task2",
			Name:         "Task 2",
			Goal:         "Goal 2",
			KeyResults:   []string{"KR2"},
			TestMethod:   "echo PASS",
			Dependencies: []string{"task1"},
		},
	}

	config := &ExecutionConfig{
		MaxRetries:     5,
		TimeoutSeconds: 30,
	}

	tmpDir := t.TempDir()

	_, err := NewExecutor(tasks, config, tmpDir, "job_circular")
	if err == nil {
		t.Fatal("Expected error for circular dependency")
	}
}

// TestExecuteJobSingleTask tests executing a single task
func TestExecuteJobSingleTask(t *testing.T) {
	tasks := []*parser.Task{
		{
			ID:           "task1",
			Name:         "Task 1",
			Goal:         "Goal 1",
			KeyResults:   []string{"KR1"},
			TestMethod:   "echo PASS",
			Dependencies: []string{},
		},
	}

	config := &ExecutionConfig{
		MaxRetries:     5,
		TimeoutSeconds: 30,
	}

	tmpDir := t.TempDir()

	executor, err := NewExecutor(tasks, config, tmpDir, "job_single")
	if err != nil {
		t.Fatalf("Failed to create executor: %v", err)
	}

	result, err := executor.ExecuteJob()
	if err != nil {
		t.Logf("ExecuteJob error: %v", err)
	}

	if result == nil {
		t.Fatal("ExecuteJob returned nil result")
	}

	if result.JobID != "job_single" {
		t.Errorf("Expected job ID 'job_single', got '%s'", result.JobID)
	}

	if result.TotalTasks != 1 {
		t.Errorf("Expected 1 total task, got %d", result.TotalTasks)
	}
}

// TestExecuteJobMultipleTasks tests executing multiple tasks
func TestExecuteJobMultipleTasks(t *testing.T) {
	tasks := []*parser.Task{
		{
			ID:           "task1",
			Name:         "Task 1",
			Goal:         "Goal 1",
			KeyResults:   []string{"KR1"},
			TestMethod:   "echo PASS",
			Dependencies: []string{},
		},
		{
			ID:           "task2",
			Name:         "Task 2",
			Goal:         "Goal 2",
			KeyResults:   []string{"KR2"},
			TestMethod:   "echo PASS",
			Dependencies: []string{},
		},
	}

	config := &ExecutionConfig{
		MaxRetries:     5,
		TimeoutSeconds: 30,
	}

	tmpDir := t.TempDir()

	executor, err := NewExecutor(tasks, config, tmpDir, "job_multi")
	if err != nil {
		t.Fatalf("Failed to create executor: %v", err)
	}

	result, err := executor.ExecuteJob()
	if err != nil {
		t.Logf("ExecuteJob error: %v", err)
	}

	if result == nil {
		t.Fatal("ExecuteJob returned nil result")
	}

	if result.TotalTasks != 2 {
		t.Errorf("Expected 2 total tasks, got %d", result.TotalTasks)
	}

	if len(result.TaskResults) != 2 {
		t.Errorf("Expected 2 task results, got %d", len(result.TaskResults))
	}
}

// TestExecuteJobTasksJSONPersistence tests that tasks.json is persisted during execution
func TestExecuteJobTasksJSONPersistence(t *testing.T) {
	tasks := []*parser.Task{
		{
			ID:           "task1",
			Name:         "Task 1",
			Goal:         "Goal 1",
			KeyResults:   []string{"KR1"},
			TestMethod:   "echo PASS",
			Dependencies: []string{},
		},
	}

	config := &ExecutionConfig{
		MaxRetries:     5,
		TimeoutSeconds: 30,
	}

	tmpDir := t.TempDir()

	executor, err := NewExecutor(tasks, config, tmpDir, "job_persist")
	if err != nil {
		t.Fatalf("Failed to create executor: %v", err)
	}

	result, err := executor.ExecuteJob()
	if err != nil {
		t.Logf("ExecuteJob error: %v", err)
	}

	if result == nil {
		t.Fatal("ExecuteJob returned nil result")
	}

	// Check if tasks.json file was created
	tasksJSONPath := filepath.Join(tmpDir, "tasks.json")
	if _, err := os.Stat(tasksJSONPath); err != nil {
		t.Fatalf("tasks.json file not found: %v", err)
	}

	// Load and verify tasks.json
	loadedTasks, err := LoadTasksJSON(tasksJSONPath)
	if err != nil {
		t.Fatalf("Failed to load tasks.json: %v", err)
	}

	if len(loadedTasks.Tasks) != 1 {
		t.Errorf("Expected 1 task in loaded JSON, got %d", len(loadedTasks.Tasks))
	}
}

// TestExecuteJobResultStatus tests ExecutionJobResult status field
func TestExecuteJobResultStatus(t *testing.T) {
	tasks := []*parser.Task{
		{
			ID:           "task1",
			Name:         "Task 1",
			Goal:         "Goal 1",
			KeyResults:   []string{"KR1"},
			TestMethod:   "echo PASS",
			Dependencies: []string{},
		},
	}

	config := &ExecutionConfig{
		MaxRetries:     5,
		TimeoutSeconds: 30,
	}

	tmpDir := t.TempDir()

	executor, err := NewExecutor(tasks, config, tmpDir, "job_status")
	if err != nil {
		t.Fatalf("Failed to create executor: %v", err)
	}

	result, err := executor.ExecuteJob()
	if err != nil {
		t.Logf("ExecuteJob error: %v", err)
	}

	if result == nil {
		t.Fatal("ExecuteJob returned nil result")
	}

	// Status should be one of: completed, failed, partial
	validStatuses := map[string]bool{"completed": true, "failed": true, "partial": true}
	if !validStatuses[result.Status] {
		t.Errorf("Invalid status: %s", result.Status)
	}
}

// TestExecuteJobLogging tests that execution logging works
func TestExecuteJobLogging(t *testing.T) {
	tasks := []*parser.Task{
		{
			ID:           "task1",
			Name:         "Task 1",
			Goal:         "Goal 1",
			KeyResults:   []string{"KR1"},
			TestMethod:   "echo PASS",
			Dependencies: []string{},
		},
	}

	config := &ExecutionConfig{
		MaxRetries:     5,
		TimeoutSeconds: 30,
	}

	tmpDir := t.TempDir()

	executor, err := NewExecutor(tasks, config, tmpDir, "job_logging")
	if err != nil {
		t.Fatalf("Failed to create executor: %v", err)
	}

	result, err := executor.ExecuteJob()
	if err != nil {
		t.Logf("ExecuteJob error: %v", err)
	}

	if result == nil {
		t.Fatal("ExecuteJob returned nil result")
	}

	if result.ExecutionLog == "" {
		t.Fatal("Execution log is empty")
	}

	// Check for expected log entries
	if !contains(result.ExecutionLog, "Starting job execution") {
		t.Fatal("Log does not contain 'Starting job execution'")
	}
}

// TestExecuteJobTimestamps tests that execution timestamps are set
func TestExecuteJobTimestamps(t *testing.T) {
	tasks := []*parser.Task{
		{
			ID:           "task1",
			Name:         "Task 1",
			Goal:         "Goal 1",
			KeyResults:   []string{"KR1"},
			TestMethod:   "echo PASS",
			Dependencies: []string{},
		},
	}

	config := &ExecutionConfig{
		MaxRetries:     5,
		TimeoutSeconds: 30,
	}

	tmpDir := t.TempDir()

	executor, err := NewExecutor(tasks, config, tmpDir, "job_timestamps")
	if err != nil {
		t.Fatalf("Failed to create executor: %v", err)
	}

	result, err := executor.ExecuteJob()
	if err != nil {
		t.Logf("ExecuteJob error: %v", err)
	}

	if result == nil {
		t.Fatal("ExecuteJob returned nil result")
	}

	if result.StartTime.IsZero() {
		t.Fatal("StartTime is zero")
	}

	if result.EndTime.IsZero() {
		t.Fatal("EndTime is zero")
	}

	if result.EndTime.Before(result.StartTime) {
		t.Fatal("EndTime is before StartTime")
	}
}

// TestExecuteJobTaskResultsNotNil tests that task results are not nil
func TestExecuteJobTaskResultsNotNil(t *testing.T) {
	tasks := []*parser.Task{
		{
			ID:           "task1",
			Name:         "Task 1",
			Goal:         "Goal 1",
			KeyResults:   []string{"KR1"},
			TestMethod:   "echo PASS",
			Dependencies: []string{},
		},
	}

	config := &ExecutionConfig{
		MaxRetries:     5,
		TimeoutSeconds: 30,
	}

	tmpDir := t.TempDir()

	executor, err := NewExecutor(tasks, config, tmpDir, "job_results")
	if err != nil {
		t.Fatalf("Failed to create executor: %v", err)
	}

	result, err := executor.ExecuteJob()
	if err != nil {
		t.Logf("ExecuteJob error: %v", err)
	}

	if result == nil {
		t.Fatal("ExecuteJob returned nil result")
	}

	if result.TaskResults == nil {
		t.Fatal("TaskResults is nil")
	}
}

// TestExecuteJobWithDependencies tests execution with task dependencies
func TestExecuteJobWithDependencies(t *testing.T) {
	tasks := []*parser.Task{
		{
			ID:           "task1",
			Name:         "Task 1",
			Goal:         "Goal 1",
			KeyResults:   []string{"KR1"},
			TestMethod:   "echo PASS",
			Dependencies: []string{},
		},
		{
			ID:           "task2",
			Name:         "Task 2",
			Goal:         "Goal 2",
			KeyResults:   []string{"KR2"},
			TestMethod:   "echo PASS",
			Dependencies: []string{"task1"},
		},
	}

	config := &ExecutionConfig{
		MaxRetries:     5,
		TimeoutSeconds: 30,
	}

	tmpDir := t.TempDir()

	executor, err := NewExecutor(tasks, config, tmpDir, "job_deps")
	if err != nil {
		t.Fatalf("Failed to create executor: %v", err)
	}

	result, err := executor.ExecuteJob()
	if err != nil {
		t.Logf("ExecuteJob error: %v", err)
	}

	if result == nil {
		t.Fatal("ExecuteJob returned nil result")
	}

	if result.TotalTasks != 2 {
		t.Errorf("Expected 2 total tasks, got %d", result.TotalTasks)
	}

	if len(result.TaskResults) != 2 {
		t.Errorf("Expected 2 task results, got %d", len(result.TaskResults))
	}
}

// TestExecuteJobCounters tests task success/failure counters
func TestExecuteJobCounters(t *testing.T) {
	tasks := []*parser.Task{
		{
			ID:           "task1",
			Name:         "Task 1",
			Goal:         "Goal 1",
			KeyResults:   []string{"KR1"},
			TestMethod:   "echo PASS",
			Dependencies: []string{},
		},
	}

	config := &ExecutionConfig{
		MaxRetries:     5,
		TimeoutSeconds: 30,
	}

	tmpDir := t.TempDir()

	executor, err := NewExecutor(tasks, config, tmpDir, "job_counters")
	if err != nil {
		t.Fatalf("Failed to create executor: %v", err)
	}

	result, err := executor.ExecuteJob()
	if err != nil {
		t.Logf("ExecuteJob error: %v", err)
	}

	if result == nil {
		t.Fatal("ExecuteJob returned nil result")
	}

	if result.SuccessfulTasks < 0 {
		t.Errorf("SuccessfulTasks cannot be negative: %d", result.SuccessfulTasks)
	}

	if result.FailedTasks < 0 {
		t.Errorf("FailedTasks cannot be negative: %d", result.FailedTasks)
	}

	if result.SuccessfulTasks+result.FailedTasks != result.TotalTasks {
		t.Errorf("Sum of successful and failed tasks (%d) does not equal total tasks (%d)",
			result.SuccessfulTasks+result.FailedTasks, result.TotalTasks)
	}
}

// TestGenerateErrorSummary tests error summary generation
func TestGenerateErrorSummary(t *testing.T) {
	tasks := []*parser.Task{
		{
			ID:           "task1",
			Name:         "Task 1",
			Goal:         "Goal 1",
			KeyResults:   []string{"KR1"},
			TestMethod:   "echo PASS",
			Dependencies: []string{},
		},
	}

	config := &ExecutionConfig{
		MaxRetries:     5,
		TimeoutSeconds: 30,
	}

	tmpDir := t.TempDir()

	executor, err := NewExecutor(tasks, config, tmpDir, "job_error_summary")
	if err != nil {
		t.Fatalf("Failed to create executor: %v", err)
	}

	taskResults := []*RetryResult{
		{
			TaskID:       "task1",
			TaskName:     "Task 1",
			Status:       "failed",
			LastError:    "Test error",
			TotalAttempts: 3,
		},
	}

	summary := executor.generateErrorSummary(taskResults)
	if summary == "" {
		t.Fatal("Error summary is empty")
	}

	if !contains(summary, "task1") {
		t.Fatal("Error summary does not contain task ID")
	}

	if !contains(summary, "Test error") {
		t.Fatal("Error summary does not contain error message")
	}
}

// Helper function to check if string contains substring
func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
