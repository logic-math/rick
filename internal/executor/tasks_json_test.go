package executor

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/sunquan/rick/internal/parser"
)

// TestGenerateTasksJSON tests the GenerateTasksJSON function
func TestGenerateTasksJSON(t *testing.T) {
	// Create a simple DAG
	tasks := []*parser.Task{
		{
			ID:           "task1",
			Name:         "Task 1",
			Goal:         "Goal 1",
			Dependencies: []string{},
		},
		{
			ID:           "task2",
			Name:         "Task 2",
			Goal:         "Goal 2",
			Dependencies: []string{"task1"},
		},
		{
			ID:           "task3",
			Name:         "Task 3",
			Goal:         "Goal 3",
			Dependencies: []string{"task1", "task2"},
		},
	}

	dag, err := NewDAG(tasks)
	if err != nil {
		t.Fatalf("Failed to create DAG: %v", err)
	}

	sortedTasks := []string{"task1", "task2", "task3"}

	tasksJSON, err := GenerateTasksJSON(dag, sortedTasks)
	if err != nil {
		t.Fatalf("GenerateTasksJSON failed: %v", err)
	}

	// Validate the generated TasksJSON
	if tasksJSON.Version != "1.0" {
		t.Errorf("Expected version 1.0, got %s", tasksJSON.Version)
	}

	if len(tasksJSON.Tasks) != 3 {
		t.Errorf("Expected 3 tasks, got %d", len(tasksJSON.Tasks))
	}

	// Check task properties
	if tasksJSON.Tasks[0].TaskID != "task1" {
		t.Errorf("Expected task1, got %s", tasksJSON.Tasks[0].TaskID)
	}

	if tasksJSON.Tasks[0].Status != "pending" {
		t.Errorf("Expected status 'pending', got %s", tasksJSON.Tasks[0].Status)
	}

	if tasksJSON.Tasks[0].Attempts != 0 {
		t.Errorf("Expected 0 attempts, got %d", tasksJSON.Tasks[0].Attempts)
	}
}

// TestGenerateTasksJSONWithNilDAG tests error handling for nil DAG
func TestGenerateTasksJSONWithNilDAG(t *testing.T) {
	_, err := GenerateTasksJSON(nil, []string{"task1"})
	if err == nil {
		t.Error("Expected error for nil DAG, got nil")
	}
}

// TestGenerateTasksJSONWithEmptySortedTasks tests error handling for empty sorted tasks
func TestGenerateTasksJSONWithEmptySortedTasks(t *testing.T) {
	tasks := []*parser.Task{
		{
			ID:           "task1",
			Name:         "Task 1",
			Goal:         "Goal 1",
			Dependencies: []string{},
		},
	}

	dag, _ := NewDAG(tasks)
	_, err := GenerateTasksJSON(dag, []string{})
	if err == nil {
		t.Error("Expected error for empty sorted tasks, got nil")
	}
}

// TestGenerateTasksJSONWithMissingTask tests error handling for missing task
func TestGenerateTasksJSONWithMissingTask(t *testing.T) {
	tasks := []*parser.Task{
		{
			ID:           "task1",
			Name:         "Task 1",
			Goal:         "Goal 1",
			Dependencies: []string{},
		},
	}

	dag, _ := NewDAG(tasks)
	_, err := GenerateTasksJSON(dag, []string{"task1", "task2"})
	if err == nil {
		t.Error("Expected error for missing task, got nil")
	}
}

// TestSaveAndLoadTasksJSON tests saving and loading tasks.json
func TestSaveAndLoadTasksJSON(t *testing.T) {
	// Create a temporary file
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "tasks.json")

	// Create a simple DAG and generate TasksJSON
	tasks := []*parser.Task{
		{
			ID:           "task1",
			Name:         "Task 1",
			Goal:         "Goal 1",
			Dependencies: []string{},
		},
		{
			ID:           "task2",
			Name:         "Task 2",
			Goal:         "Goal 2",
			Dependencies: []string{"task1"},
		},
	}

	dag, _ := NewDAG(tasks)
	tasksJSON, _ := GenerateTasksJSON(dag, []string{"task1", "task2"})

	// Save to file
	if err := SaveTasksJSON(filePath, tasksJSON); err != nil {
		t.Fatalf("SaveTasksJSON failed: %v", err)
	}

	// Check file exists
	if _, err := os.Stat(filePath); err != nil {
		t.Fatalf("File not created: %v", err)
	}

	// Load from file
	loadedJSON, err := LoadTasksJSON(filePath)
	if err != nil {
		t.Fatalf("LoadTasksJSON failed: %v", err)
	}

	// Verify loaded data
	if len(loadedJSON.Tasks) != 2 {
		t.Errorf("Expected 2 tasks, got %d", len(loadedJSON.Tasks))
	}

	if loadedJSON.Tasks[0].TaskID != "task1" {
		t.Errorf("Expected task1, got %s", loadedJSON.Tasks[0].TaskID)
	}

	if loadedJSON.Tasks[1].TaskID != "task2" {
		t.Errorf("Expected task2, got %s", loadedJSON.Tasks[1].TaskID)
	}
}

// TestUpdateTaskStatus tests updating task status
func TestUpdateTaskStatus(t *testing.T) {
	tasks := []*parser.Task{
		{
			ID:           "task1",
			Name:         "Task 1",
			Goal:         "Goal 1",
			Dependencies: []string{},
		},
	}

	dag, _ := NewDAG(tasks)
	tasksJSON, _ := GenerateTasksJSON(dag, []string{"task1"})

	// Update status to running
	if err := tasksJSON.UpdateTaskStatus("task1", "running"); err != nil {
		t.Fatalf("UpdateTaskStatus failed: %v", err)
	}

	status, _ := tasksJSON.GetTaskStatus("task1")
	if status != "running" {
		t.Errorf("Expected status 'running', got %s", status)
	}

	// Update status to success
	if err := tasksJSON.UpdateTaskStatus("task1", "success"); err != nil {
		t.Fatalf("UpdateTaskStatus failed: %v", err)
	}

	status, _ = tasksJSON.GetTaskStatus("task1")
	if status != "success" {
		t.Errorf("Expected status 'success', got %s", status)
	}
}

// TestUpdateTaskStatusWithInvalidStatus tests error handling for invalid status
func TestUpdateTaskStatusWithInvalidStatus(t *testing.T) {
	tasks := []*parser.Task{
		{
			ID:           "task1",
			Name:         "Task 1",
			Goal:         "Goal 1",
			Dependencies: []string{},
		},
	}

	dag, _ := NewDAG(tasks)
	tasksJSON, _ := GenerateTasksJSON(dag, []string{"task1"})

	// Try to update with invalid status
	err := tasksJSON.UpdateTaskStatus("task1", "invalid")
	if err == nil {
		t.Error("Expected error for invalid status, got nil")
	}
}

// TestUpdateTaskStatusWithNonExistentTask tests error handling for non-existent task
func TestUpdateTaskStatusWithNonExistentTask(t *testing.T) {
	tasks := []*parser.Task{
		{
			ID:           "task1",
			Name:         "Task 1",
			Goal:         "Goal 1",
			Dependencies: []string{},
		},
	}

	dag, _ := NewDAG(tasks)
	tasksJSON, _ := GenerateTasksJSON(dag, []string{"task1"})

	// Try to update non-existent task
	err := tasksJSON.UpdateTaskStatus("task2", "running")
	if err == nil {
		t.Error("Expected error for non-existent task, got nil")
	}
}

// TestGetTaskStatus tests getting task status
func TestGetTaskStatus(t *testing.T) {
	tasks := []*parser.Task{
		{
			ID:           "task1",
			Name:         "Task 1",
			Goal:         "Goal 1",
			Dependencies: []string{},
		},
	}

	dag, _ := NewDAG(tasks)
	tasksJSON, _ := GenerateTasksJSON(dag, []string{"task1"})

	status, err := tasksJSON.GetTaskStatus("task1")
	if err != nil {
		t.Fatalf("GetTaskStatus failed: %v", err)
	}

	if status != "pending" {
		t.Errorf("Expected status 'pending', got %s", status)
	}
}

// TestGetTaskStatusWithNonExistentTask tests error handling for non-existent task
func TestGetTaskStatusWithNonExistentTask(t *testing.T) {
	tasks := []*parser.Task{
		{
			ID:           "task1",
			Name:         "Task 1",
			Goal:         "Goal 1",
			Dependencies: []string{},
		},
	}

	dag, _ := NewDAG(tasks)
	tasksJSON, _ := GenerateTasksJSON(dag, []string{"task1"})

	_, err := tasksJSON.GetTaskStatus("task2")
	if err == nil {
		t.Error("Expected error for non-existent task, got nil")
	}
}

// TestIncrementAttempts tests incrementing task attempts
func TestIncrementAttempts(t *testing.T) {
	tasks := []*parser.Task{
		{
			ID:           "task1",
			Name:         "Task 1",
			Goal:         "Goal 1",
			Dependencies: []string{},
		},
	}

	dag, _ := NewDAG(tasks)
	tasksJSON, _ := GenerateTasksJSON(dag, []string{"task1"})

	// Increment attempts
	if err := tasksJSON.IncrementAttempts("task1"); err != nil {
		t.Fatalf("IncrementAttempts failed: %v", err)
	}

	task, _ := tasksJSON.GetTask("task1")
	if task.Attempts != 1 {
		t.Errorf("Expected 1 attempt, got %d", task.Attempts)
	}

	// Increment again
	if err := tasksJSON.IncrementAttempts("task1"); err != nil {
		t.Fatalf("IncrementAttempts failed: %v", err)
	}

	task, _ = tasksJSON.GetTask("task1")
	if task.Attempts != 2 {
		t.Errorf("Expected 2 attempts, got %d", task.Attempts)
	}
}

// TestTasksJSONGetAllTasks tests retrieving all tasks
func TestTasksJSONGetAllTasks(t *testing.T) {
	tasks := []*parser.Task{
		{
			ID:           "task1",
			Name:         "Task 1",
			Goal:         "Goal 1",
			Dependencies: []string{},
		},
		{
			ID:           "task2",
			Name:         "Task 2",
			Goal:         "Goal 2",
			Dependencies: []string{"task1"},
		},
	}

	dag, _ := NewDAG(tasks)
	tasksJSON, _ := GenerateTasksJSON(dag, []string{"task1", "task2"})

	allTasks := tasksJSON.GetAllTasks()
	if len(allTasks) != 2 {
		t.Errorf("Expected 2 tasks, got %d", len(allTasks))
	}
}

// TestGetTasksByStatus tests filtering tasks by status
func TestGetTasksByStatus(t *testing.T) {
	tasks := []*parser.Task{
		{
			ID:           "task1",
			Name:         "Task 1",
			Goal:         "Goal 1",
			Dependencies: []string{},
		},
		{
			ID:           "task2",
			Name:         "Task 2",
			Goal:         "Goal 2",
			Dependencies: []string{"task1"},
		},
		{
			ID:           "task3",
			Name:         "Task 3",
			Goal:         "Goal 3",
			Dependencies: []string{"task1"},
		},
	}

	dag, _ := NewDAG(tasks)
	tasksJSON, _ := GenerateTasksJSON(dag, []string{"task1", "task2", "task3"})

	// Update some task statuses
	tasksJSON.UpdateTaskStatus("task1", "success")
	tasksJSON.UpdateTaskStatus("task2", "success")
	tasksJSON.UpdateTaskStatus("task3", "failed")

	// Get tasks by status
	completedTasks := tasksJSON.GetTasksByStatus("success")
	if len(completedTasks) != 2 {
		t.Errorf("Expected 2 completed tasks, got %d", len(completedTasks))
	}

	failedTasks := tasksJSON.GetTasksByStatus("failed")
	if len(failedTasks) != 1 {
		t.Errorf("Expected 1 failed task, got %d", len(failedTasks))
	}
}

// TestGetCompletedTasks tests retrieving completed tasks
func TestGetCompletedTasks(t *testing.T) {
	tasks := []*parser.Task{
		{
			ID:           "task1",
			Name:         "Task 1",
			Goal:         "Goal 1",
			Dependencies: []string{},
		},
		{
			ID:           "task2",
			Name:         "Task 2",
			Goal:         "Goal 2",
			Dependencies: []string{"task1"},
		},
	}

	dag, _ := NewDAG(tasks)
	tasksJSON, _ := GenerateTasksJSON(dag, []string{"task1", "task2"})

	tasksJSON.UpdateTaskStatus("task1", "success")

	completedTasks := tasksJSON.GetCompletedTasks()
	if len(completedTasks) != 1 {
		t.Errorf("Expected 1 completed task, got %d", len(completedTasks))
	}
}

// TestGetFailedTasks tests retrieving failed tasks
func TestGetFailedTasks(t *testing.T) {
	tasks := []*parser.Task{
		{
			ID:           "task1",
			Name:         "Task 1",
			Goal:         "Goal 1",
			Dependencies: []string{},
		},
		{
			ID:           "task2",
			Name:         "Task 2",
			Goal:         "Goal 2",
			Dependencies: []string{"task1"},
		},
	}

	dag, _ := NewDAG(tasks)
	tasksJSON, _ := GenerateTasksJSON(dag, []string{"task1", "task2"})

	tasksJSON.UpdateTaskStatus("task2", "failed")

	failedTasks := tasksJSON.GetFailedTasks()
	if len(failedTasks) != 1 {
		t.Errorf("Expected 1 failed task, got %d", len(failedTasks))
	}
}

// TestGetPendingTasks tests retrieving pending tasks
func TestGetPendingTasks(t *testing.T) {
	tasks := []*parser.Task{
		{
			ID:           "task1",
			Name:         "Task 1",
			Goal:         "Goal 1",
			Dependencies: []string{},
		},
		{
			ID:           "task2",
			Name:         "Task 2",
			Goal:         "Goal 2",
			Dependencies: []string{"task1"},
		},
	}

	dag, _ := NewDAG(tasks)
	tasksJSON, _ := GenerateTasksJSON(dag, []string{"task1", "task2"})

	tasksJSON.UpdateTaskStatus("task1", "success")

	pendingTasks := tasksJSON.GetPendingTasks()
	if len(pendingTasks) != 1 {
		t.Errorf("Expected 1 pending task, got %d", len(pendingTasks))
	}

	if pendingTasks[0].TaskID != "task2" {
		t.Errorf("Expected task2, got %s", pendingTasks[0].TaskID)
	}
}

// TestGetTaskCount tests getting task count
func TestGetTaskCount(t *testing.T) {
	tasks := []*parser.Task{
		{
			ID:           "task1",
			Name:         "Task 1",
			Goal:         "Goal 1",
			Dependencies: []string{},
		},
		{
			ID:           "task2",
			Name:         "Task 2",
			Goal:         "Goal 2",
			Dependencies: []string{"task1"},
		},
	}

	dag, _ := NewDAG(tasks)
	tasksJSON, _ := GenerateTasksJSON(dag, []string{"task1", "task2"})

	count := tasksJSON.GetTaskCount()
	if count != 2 {
		t.Errorf("Expected 2 tasks, got %d", count)
	}
}

// TestGetCompletedCount tests getting completed count
func TestGetCompletedCount(t *testing.T) {
	tasks := []*parser.Task{
		{
			ID:           "task1",
			Name:         "Task 1",
			Goal:         "Goal 1",
			Dependencies: []string{},
		},
		{
			ID:           "task2",
			Name:         "Task 2",
			Goal:         "Goal 2",
			Dependencies: []string{"task1"},
		},
	}

	dag, _ := NewDAG(tasks)
	tasksJSON, _ := GenerateTasksJSON(dag, []string{"task1", "task2"})

	tasksJSON.UpdateTaskStatus("task1", "success")

	count := tasksJSON.GetCompletedCount()
	if count != 1 {
		t.Errorf("Expected 1 completed task, got %d", count)
	}
}

// TestGetFailedCount tests getting failed count
func TestGetFailedCount(t *testing.T) {
	tasks := []*parser.Task{
		{
			ID:           "task1",
			Name:         "Task 1",
			Goal:         "Goal 1",
			Dependencies: []string{},
		},
		{
			ID:           "task2",
			Name:         "Task 2",
			Goal:         "Goal 2",
			Dependencies: []string{"task1"},
		},
	}

	dag, _ := NewDAG(tasks)
	tasksJSON, _ := GenerateTasksJSON(dag, []string{"task1", "task2"})

	tasksJSON.UpdateTaskStatus("task2", "failed")

	count := tasksJSON.GetFailedCount()
	if count != 1 {
		t.Errorf("Expected 1 failed task, got %d", count)
	}
}

// TestGetPendingCount tests getting pending count
func TestGetPendingCount(t *testing.T) {
	tasks := []*parser.Task{
		{
			ID:           "task1",
			Name:         "Task 1",
			Goal:         "Goal 1",
			Dependencies: []string{},
		},
		{
			ID:           "task2",
			Name:         "Task 2",
			Goal:         "Goal 2",
			Dependencies: []string{"task1"},
		},
	}

	dag, _ := NewDAG(tasks)
	tasksJSON, _ := GenerateTasksJSON(dag, []string{"task1", "task2"})

	tasksJSON.UpdateTaskStatus("task1", "success")

	count := tasksJSON.GetPendingCount()
	if count != 1 {
		t.Errorf("Expected 1 pending task, got %d", count)
	}
}

// TestIsAllCompleted tests checking if all tasks are completed
func TestIsAllCompleted(t *testing.T) {
	tasks := []*parser.Task{
		{
			ID:           "task1",
			Name:         "Task 1",
			Goal:         "Goal 1",
			Dependencies: []string{},
		},
		{
			ID:           "task2",
			Name:         "Task 2",
			Goal:         "Goal 2",
			Dependencies: []string{"task1"},
		},
	}

	dag, _ := NewDAG(tasks)
	tasksJSON, _ := GenerateTasksJSON(dag, []string{"task1", "task2"})

	// Not all completed
	if tasksJSON.IsAllCompleted() {
		t.Error("Expected not all completed, but got true")
	}

	// Mark all as completed
	tasksJSON.UpdateTaskStatus("task1", "success")
	tasksJSON.UpdateTaskStatus("task2", "success")

	if !tasksJSON.IsAllCompleted() {
		t.Error("Expected all completed, but got false")
	}
}

// TestIsAnyFailed tests checking if any task has failed
func TestIsAnyFailed(t *testing.T) {
	tasks := []*parser.Task{
		{
			ID:           "task1",
			Name:         "Task 1",
			Goal:         "Goal 1",
			Dependencies: []string{},
		},
		{
			ID:           "task2",
			Name:         "Task 2",
			Goal:         "Goal 2",
			Dependencies: []string{"task1"},
		},
	}

	dag, _ := NewDAG(tasks)
	tasksJSON, _ := GenerateTasksJSON(dag, []string{"task1", "task2"})

	// No failures
	if tasksJSON.IsAnyFailed() {
		t.Error("Expected no failures, but got true")
	}

	// Mark one as failed
	tasksJSON.UpdateTaskStatus("task2", "failed")

	if !tasksJSON.IsAnyFailed() {
		t.Error("Expected failure, but got false")
	}
}

// TestUpdateTaskStatusWithError tests updating task status with error
func TestUpdateTaskStatusWithError(t *testing.T) {
	tasks := []*parser.Task{
		{
			ID:           "task1",
			Name:         "Task 1",
			Goal:         "Goal 1",
			Dependencies: []string{},
		},
	}

	dag, _ := NewDAG(tasks)
	tasksJSON, _ := GenerateTasksJSON(dag, []string{"task1"})

	errorMsg := "Test error message"
	if err := tasksJSON.UpdateTaskStatusWithError("task1", "failed", errorMsg); err != nil {
		t.Fatalf("UpdateTaskStatusWithError failed: %v", err)
	}

	task, _ := tasksJSON.GetTask("task1")
	if task.Status != "failed" {
		t.Errorf("Expected status 'failed', got %s", task.Status)
	}

	if task.Error != errorMsg {
		t.Errorf("Expected error '%s', got '%s'", errorMsg, task.Error)
	}
}

// TestUpdateTaskStatusWithOutput tests updating task status with output
func TestUpdateTaskStatusWithOutput(t *testing.T) {
	tasks := []*parser.Task{
		{
			ID:           "task1",
			Name:         "Task 1",
			Goal:         "Goal 1",
			Dependencies: []string{},
		},
	}

	dag, _ := NewDAG(tasks)
	tasksJSON, _ := GenerateTasksJSON(dag, []string{"task1"})

	output := "Test output"
	if err := tasksJSON.UpdateTaskStatusWithOutput("task1", "success", output); err != nil {
		t.Fatalf("UpdateTaskStatusWithOutput failed: %v", err)
	}

	task, _ := tasksJSON.GetTask("task1")
	if task.Status != "success" {
		t.Errorf("Expected status 'success', got %s", task.Status)
	}

	if task.Output != output {
		t.Errorf("Expected output '%s', got '%s'", output, task.Output)
	}
}

// TestSaveAndLoadPreservesData tests that save and load preserves all data
func TestSaveAndLoadPreservesData(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "tasks.json")

	tasks := []*parser.Task{
		{
			ID:           "task1",
			Name:         "Task 1",
			Goal:         "Goal 1",
			Dependencies: []string{},
		},
		{
			ID:           "task2",
			Name:         "Task 2",
			Goal:         "Goal 2",
			Dependencies: []string{"task1"},
		},
	}

	dag, _ := NewDAG(tasks)
	tasksJSON, _ := GenerateTasksJSON(dag, []string{"task1", "task2"})

	// Modify some data
	tasksJSON.UpdateTaskStatus("task1", "success")
	tasksJSON.UpdateTaskStatusWithError("task2", "failed", "Test error")
	tasksJSON.IncrementAttempts("task2")
	tasksJSON.IncrementAttempts("task2")

	// Save
	SaveTasksJSON(filePath, tasksJSON)

	// Load
	loadedJSON, _ := LoadTasksJSON(filePath)

	// Verify data is preserved
	task1, _ := loadedJSON.GetTask("task1")
	if task1.Status != "success" {
		t.Errorf("Expected task1 status 'success', got %s", task1.Status)
	}

	task2, _ := loadedJSON.GetTask("task2")
	if task2.Status != "failed" {
		t.Errorf("Expected task2 status 'failed', got %s", task2.Status)
	}

	if task2.Error != "Test error" {
		t.Errorf("Expected task2 error 'Test error', got %s", task2.Error)
	}

	if task2.Attempts != 2 {
		t.Errorf("Expected task2 attempts 2, got %d", task2.Attempts)
	}
}

// TestSaveTasksJSONWithNilTasksJSON tests error handling for nil TasksJSON
func TestSaveTasksJSONWithNilTasksJSON(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "tasks.json")

	err := SaveTasksJSON(filePath, nil)
	if err == nil {
		t.Error("Expected error for nil TasksJSON, got nil")
	}
}

// TestLoadTasksJSONWithEmptyPath tests error handling for empty path
func TestLoadTasksJSONWithEmptyPath(t *testing.T) {
	_, err := LoadTasksJSON("")
	if err == nil {
		t.Error("Expected error for empty path, got nil")
	}
}

// TestLoadTasksJSONWithNonExistentFile tests error handling for non-existent file
func TestLoadTasksJSONWithNonExistentFile(t *testing.T) {
	_, err := LoadTasksJSON("/non/existent/path/tasks.json")
	if err == nil {
		t.Error("Expected error for non-existent file, got nil")
	}
}

// TestTimestampUpdates tests that timestamps are updated correctly
func TestTimestampUpdates(t *testing.T) {
	tasks := []*parser.Task{
		{
			ID:           "task1",
			Name:         "Task 1",
			Goal:         "Goal 1",
			Dependencies: []string{},
		},
	}

	dag, _ := NewDAG(tasks)
	tasksJSON, _ := GenerateTasksJSON(dag, []string{"task1"})

	originalUpdatedAt := tasksJSON.Tasks[0].UpdatedAt

	// Wait a bit and update status
	time.Sleep(10 * time.Millisecond)
	tasksJSON.UpdateTaskStatus("task1", "running")

	newUpdatedAt := tasksJSON.Tasks[0].UpdatedAt

	if !newUpdatedAt.After(originalUpdatedAt) {
		t.Error("Expected UpdatedAt to be updated, but it wasn't")
	}
}

// TestGetTask tests retrieving a specific task
func TestGetTask(t *testing.T) {
	tasks := []*parser.Task{
		{
			ID:           "task1",
			Name:         "Task 1",
			Goal:         "Goal 1",
			Dependencies: []string{},
		},
	}

	dag, _ := NewDAG(tasks)
	tasksJSON, _ := GenerateTasksJSON(dag, []string{"task1"})

	task, err := tasksJSON.GetTask("task1")
	if err != nil {
		t.Fatalf("GetTask failed: %v", err)
	}

	if task.TaskID != "task1" {
		t.Errorf("Expected task1, got %s", task.TaskID)
	}

	if task.TaskName != "Task 1" {
		t.Errorf("Expected 'Task 1', got %s", task.TaskName)
	}
}

// TestUpdateTaskCommit tests updating commit hash
func TestUpdateTaskCommit(t *testing.T) {
	tasks := []*parser.Task{{ID: "task1", Name: "T1", Dependencies: []string{}}}
	dag, _ := NewDAG(tasks)
	tj, _ := GenerateTasksJSON(dag, []string{"task1"})

	if err := tj.UpdateTaskCommit("task1", "abc123"); err != nil {
		t.Fatalf("UpdateTaskCommit failed: %v", err)
	}
	state, _ := tj.GetTask("task1")
	if state.CommitHash != "abc123" {
		t.Errorf("expected commit_hash=abc123, got %s", state.CommitHash)
	}
}

func TestUpdateTaskCommit_EmptyID(t *testing.T) {
	tasks := []*parser.Task{{ID: "task1", Name: "T1", Dependencies: []string{}}}
	dag, _ := NewDAG(tasks)
	tj, _ := GenerateTasksJSON(dag, []string{"task1"})

	if err := tj.UpdateTaskCommit("", "abc123"); err == nil {
		t.Fatal("expected error for empty task ID")
	}
}

func TestUpdateTaskCommit_NotFound(t *testing.T) {
	tasks := []*parser.Task{{ID: "task1", Name: "T1", Dependencies: []string{}}}
	dag, _ := NewDAG(tasks)
	tj, _ := GenerateTasksJSON(dag, []string{"task1"})

	if err := tj.UpdateTaskCommit("nonexistent", "abc123"); err == nil {
		t.Fatal("expected error for nonexistent task")
	}
}

// TestUpdateTaskFile tests updating task file name
func TestUpdateTaskFile(t *testing.T) {
	tasks := []*parser.Task{{ID: "task1", Name: "T1", Dependencies: []string{}}}
	dag, _ := NewDAG(tasks)
	tj, _ := GenerateTasksJSON(dag, []string{"task1"})

	if err := tj.UpdateTaskFile("task1", "task1.md"); err != nil {
		t.Fatalf("UpdateTaskFile failed: %v", err)
	}
	state, _ := tj.GetTask("task1")
	if state.TaskFile != "task1.md" {
		t.Errorf("expected task_file=task1.md, got %s", state.TaskFile)
	}
}

func TestUpdateTaskFile_EmptyID(t *testing.T) {
	tasks := []*parser.Task{{ID: "task1", Name: "T1", Dependencies: []string{}}}
	dag, _ := NewDAG(tasks)
	tj, _ := GenerateTasksJSON(dag, []string{"task1"})

	if err := tj.UpdateTaskFile("", "task1.md"); err == nil {
		t.Fatal("expected error for empty task ID")
	}
}

func TestUpdateTaskFile_NotFound(t *testing.T) {
	tasks := []*parser.Task{{ID: "task1", Name: "T1", Dependencies: []string{}}}
	dag, _ := NewDAG(tasks)
	tj, _ := GenerateTasksJSON(dag, []string{"task1"})

	if err := tj.UpdateTaskFile("nonexistent", "task1.md"); err == nil {
		t.Fatal("expected error for nonexistent task")
	}
}

// TestGetTask_EmptyID tests GetTask with empty ID
func TestGetTask_EmptyID(t *testing.T) {
	tasks := []*parser.Task{{ID: "task1", Name: "T1", Dependencies: []string{}}}
	dag, _ := NewDAG(tasks)
	tj, _ := GenerateTasksJSON(dag, []string{"task1"})

	_, err := tj.GetTask("")
	if err == nil {
		t.Fatal("expected error for empty task ID")
	}
}

// TestGetTask_NotFound tests GetTask with nonexistent ID
func TestGetTask_NotFound(t *testing.T) {
	tasks := []*parser.Task{{ID: "task1", Name: "T1", Dependencies: []string{}}}
	dag, _ := NewDAG(tasks)
	tj, _ := GenerateTasksJSON(dag, []string{"task1"})

	_, err := tj.GetTask("nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent task")
	}
}
