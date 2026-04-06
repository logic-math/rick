package cmd

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/sunquan/rick/internal/executor"
	"github.com/sunquan/rick/internal/parser"
)

func TestNewDoingCmd(t *testing.T) {
	cmd := NewDoingCmd()
	if cmd == nil {
		t.Fatal("NewDoingCmd returned nil")
	}

	if cmd.Use != "doing [job_id]" {
		t.Errorf("Expected Use 'doing [job_id]', got '%s'", cmd.Use)
	}

	if cmd.Short != "Execute tasks in a job" {
		t.Errorf("Expected Short 'Execute tasks in a job', got '%s'", cmd.Short)
	}

	if cmd.RunE == nil {
		t.Error("Expected RunE to be defined")
	}
}

func TestDoingCmdFlags(t *testing.T) {
	cmd := NewDoingCmd()

	// Check for --job flag
	jobFlag := cmd.Flags().Lookup("job")
	if jobFlag == nil {
		t.Error("Expected --job flag to be defined")
	}

	if jobFlag.Usage != "Job ID to execute" {
		t.Errorf("Expected --job usage 'Job ID to execute', got '%s'", jobFlag.Usage)
	}
}

func TestExtractTaskNumber(t *testing.T) {
	tests := []struct {
		filename string
		expected int
	}{
		{"task1.md", 1},
		{"task2.md", 2},
		{"task10.md", 10},
		{"task99.md", 99},
		{"task.md", 0},
		{"notask.md", 0},
	}

	for _, tt := range tests {
		t.Run(tt.filename, func(t *testing.T) {
			result := extractTaskNumber(tt.filename)
			if result != tt.expected {
				t.Errorf("extractTaskNumber(%q) = %d, want %d", tt.filename, result, tt.expected)
			}
		})
	}
}

func TestSortTaskFiles(t *testing.T) {
	tests := []struct {
		name     string
		input    []string
		expected []string
	}{
		{
			name:     "unsorted files",
			input:    []string{"task3.md", "task1.md", "task2.md"},
			expected: []string{"task1.md", "task2.md", "task3.md"},
		},
		{
			name:     "already sorted",
			input:    []string{"task1.md", "task2.md", "task3.md"},
			expected: []string{"task1.md", "task2.md", "task3.md"},
		},
		{
			name:     "reverse order",
			input:    []string{"task5.md", "task4.md", "task3.md", "task2.md", "task1.md"},
			expected: []string{"task1.md", "task2.md", "task3.md", "task4.md", "task5.md"},
		},
		{
			name:     "single file",
			input:    []string{"task1.md"},
			expected: []string{"task1.md"},
		},
		{
			name:     "empty list",
			input:    []string{},
			expected: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sortTaskFiles(tt.input)
			if len(tt.input) != len(tt.expected) {
				t.Errorf("Length mismatch: got %d, want %d", len(tt.input), len(tt.expected))
			}
			for i, v := range tt.input {
				if v != tt.expected[i] {
					t.Errorf("Element %d: got %q, want %q", i, v, tt.expected[i])
				}
			}
		})
	}
}

func TestLoadTasksFromPlan(t *testing.T) {
	// Create temporary directory structure
	tmpDir := t.TempDir()
	planDir := filepath.Join(tmpDir, "plan")
	if err := os.MkdirAll(planDir, 0755); err != nil {
		t.Fatalf("Failed to create plan directory: %v", err)
	}

	// Create sample task files
	task1Content := `# 依赖关系


# 任务名称
Task 1

# 任务目标
Complete task 1

# 关键结果
1. Result 1
2. Result 2

# 测试方法
1. Test 1
2. Test 2
`

	task2Content := `# 依赖关系
task1

# 任务名称
Task 2

# 任务目标
Complete task 2

# 关键结果
1. Result 1

# 测试方法
1. Test 1
`

	if err := os.WriteFile(filepath.Join(planDir, "task1.md"), []byte(task1Content), 0644); err != nil {
		t.Fatalf("Failed to write task1.md: %v", err)
	}

	if err := os.WriteFile(filepath.Join(planDir, "task2.md"), []byte(task2Content), 0644); err != nil {
		t.Fatalf("Failed to write task2.md: %v", err)
	}

	// Test loading tasks
	tasks, err := loadTasksFromPlan(planDir)
	if err != nil {
		t.Fatalf("loadTasksFromPlan failed: %v", err)
	}

	if len(tasks) != 2 {
		t.Errorf("Expected 2 tasks, got %d", len(tasks))
	}

	// Verify task1
	if tasks[0].ID != "task1" {
		t.Errorf("Expected task1 ID, got %s", tasks[0].ID)
	}
	if tasks[0].Name != "Task 1" {
		t.Errorf("Expected 'Task 1', got %s", tasks[0].Name)
	}
	if len(tasks[0].Dependencies) != 0 {
		t.Errorf("Expected no dependencies for task1, got %v", tasks[0].Dependencies)
	}

	// Verify task2
	if tasks[1].ID != "task2" {
		t.Errorf("Expected task2 ID, got %s", tasks[1].ID)
	}
	if tasks[1].Name != "Task 2" {
		t.Errorf("Expected 'Task 2', got %s", tasks[1].Name)
	}
	if len(tasks[1].Dependencies) != 1 || tasks[1].Dependencies[0] != "task1" {
		t.Errorf("Expected task1 dependency for task2, got %v", tasks[1].Dependencies)
	}
}

func TestLoadTasksFromPlanNoFiles(t *testing.T) {
	tmpDir := t.TempDir()
	planDir := filepath.Join(tmpDir, "plan")
	if err := os.MkdirAll(planDir, 0755); err != nil {
		t.Fatalf("Failed to create plan directory: %v", err)
	}

	// Try to load from empty directory
	tasks, err := loadTasksFromPlan(planDir)
	if err == nil {
		t.Error("Expected error for empty plan directory")
	}
	if tasks != nil {
		t.Errorf("Expected nil tasks for empty directory, got %v", tasks)
	}
}

func TestLoadTasksFromPlanNotFound(t *testing.T) {
	nonexistentDir := filepath.Join(t.TempDir(), "nonexistent")

	tasks, err := loadTasksFromPlan(nonexistentDir)
	if err == nil {
		t.Error("Expected error for nonexistent plan directory")
	}
	if tasks != nil {
		t.Errorf("Expected nil tasks, got %v", tasks)
	}
}

func TestPrintExecutionSummary(t *testing.T) {
	result := &executor.ExecutionJobResult{
		JobID:           "job1",
		Status:          "completed",
		TotalTasks:      3,
		SuccessfulTasks: 3,
		FailedTasks:     0,
		TaskResults: []*executor.RetryResult{
			{
				TaskID:       "task1",
				TaskName:     "Task 1",
				Status:       "success",
				TotalAttempts: 1,
			},
			{
				TaskID:       "task2",
				TaskName:     "Task 2",
				Status:       "success",
				TotalAttempts: 1,
			},
			{
				TaskID:       "task3",
				TaskName:     "Task 3",
				Status:       "success",
				TotalAttempts: 1,
			},
		},
	}

	// This should not panic
	printExecutionSummary(result)
}

func TestPrintExecutionSummaryWithFailures(t *testing.T) {
	result := &executor.ExecutionJobResult{
		JobID:           "job1",
		Status:          "partial",
		TotalTasks:      3,
		SuccessfulTasks: 2,
		FailedTasks:     1,
		TaskResults: []*executor.RetryResult{
			{
				TaskID:       "task1",
				TaskName:     "Task 1",
				Status:       "success",
				TotalAttempts: 1,
			},
			{
				TaskID:        "task2",
				TaskName:      "Task 2",
				Status:        "failed",
				TotalAttempts: 3,
				LastError:     "Connection timeout",
			},
			{
				TaskID:       "task3",
				TaskName:     "Task 3",
				Status:       "success",
				TotalAttempts: 1,
			},
		},
	}

	// This should not panic
	printExecutionSummary(result)
}

func TestDoingCmdMissingJobID(t *testing.T) {
	cmd := NewDoingCmd()

	// Execute without job ID should fail
	err := cmd.RunE(cmd, []string{})
	if err == nil {
		t.Error("Expected error when job ID is missing")
	}
}

func TestDoingCmdWithJobIDArg(t *testing.T) {
	cmd := NewDoingCmd()

	// This will fail because the workspace doesn't exist, but it should
	// at least validate the job ID argument
	err := cmd.RunE(cmd, []string{"job1"})
	// Error is expected due to missing workspace, but not due to invalid arguments
	if err != nil && err.Error() == "job ID is required. Usage: rick doing [job_id] or rick doing --job job_id" {
		t.Error("Should not complain about job ID when it's provided as argument")
	}
}

func TestTaskStruct(t *testing.T) {
	task := &parser.Task{
		ID:           "task1",
		Name:         "Test Task",
		Goal:         "Complete the test",
		KeyResults:   []string{"Result 1", "Result 2"},
		TestMethod:   "Run tests",
		Dependencies: []string{},
	}

	if task.ID != "task1" {
		t.Errorf("Expected task ID task1, got %s", task.ID)
	}

	if task.Name != "Test Task" {
		t.Errorf("Expected task name 'Test Task', got %s", task.Name)
	}

	if len(task.KeyResults) != 2 {
		t.Errorf("Expected 2 key results, got %d", len(task.KeyResults))
	}
}

func TestExecutionJobResultDuration(t *testing.T) {
	now := time.Now()
	result := &executor.ExecutionJobResult{
		StartTime: now,
		EndTime:   now.Add(5 * time.Second),
	}

	duration := result.Duration()
	if duration.Seconds() != 5.0 {
		t.Errorf("Expected duration 5s, got %v", duration)
	}
}

// TestExecuteDoingWorkflow_NoJobDir tests executeDoingWorkflow with missing job dir
func TestExecuteDoingWorkflow_NoJobDir(t *testing.T) {
	orig, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(orig) }()

	if err := os.MkdirAll(filepath.Join(dir, ".rick"), 0755); err != nil {
		t.Fatal(err)
	}

	err = executeDoingWorkflow("job_test")
	if err == nil {
		t.Fatal("expected error for missing job dir")
	}
}

// TestExecuteDoingWorkflow_NoPlanDir tests executeDoingWorkflow with missing plan dir
func TestExecuteDoingWorkflow_NoPlanDir(t *testing.T) {
	orig, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(orig) }()

	if err := os.MkdirAll(filepath.Join(dir, ".rick", "jobs", "job_test"), 0755); err != nil {
		t.Fatal(err)
	}

	err = executeDoingWorkflow("job_test")
	if err == nil {
		t.Fatal("expected error for missing plan dir")
	}
}

// TestExecuteDoingWorkflow_NoTasks tests executeDoingWorkflow with empty plan dir
func TestExecuteDoingWorkflow_NoTasks(t *testing.T) {
	orig, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(orig) }()

	if err := os.MkdirAll(filepath.Join(dir, ".rick", "jobs", "job_test", "plan"), 0755); err != nil {
		t.Fatal(err)
	}

	err = executeDoingWorkflow("job_test")
	if err == nil {
		t.Fatal("expected error for no tasks")
	}
}

// TestExecuteDoingWorkflow_ResumesFromTasksJSON verifies that when doing/tasks.json
// already exists with task1=success, executeDoingWorkflow loads it and skips task1.
// We confirm this by checking that the execution log contains the skip message.
func TestExecuteDoingWorkflow_ResumesFromTasksJSON(t *testing.T) {
	orig, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(orig) }()

	// Set up plan directory with two tasks
	planDir := filepath.Join(dir, ".rick", "jobs", "job_resume", "plan")
	if err := os.MkdirAll(planDir, 0755); err != nil {
		t.Fatal(err)
	}
	task1 := "# 依赖关系\n\n# 任务名称\nTask1\n# 任务目标\nGoal1\n# 关键结果\n1. KR1\n# 测试方法\nTest1\n"
	task2 := "# 依赖关系\ntask1\n# 任务名称\nTask2\n# 任务目标\nGoal2\n# 关键结果\n1. KR2\n# 测试方法\nTest2\n"
	if err := os.WriteFile(filepath.Join(planDir, "task1.md"), []byte(task1), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(planDir, "task2.md"), []byte(task2), 0644); err != nil {
		t.Fatal(err)
	}

	// Pre-create doing/tasks.json with task1 already succeeded
	doingDir := filepath.Join(dir, ".rick", "jobs", "job_resume", "doing")
	if err := os.MkdirAll(doingDir, 0755); err != nil {
		t.Fatal(err)
	}
	existingJSON := `{
  "version": "1.0",
  "created_at": "2024-01-01T00:00:00Z",
  "updated_at": "2024-01-01T00:00:00Z",
  "tasks": [
    {"task_id": "task1", "task_name": "Task1", "status": "success",
     "dependencies": [], "attempts": 1,
     "created_at": "2024-01-01T00:00:00Z", "updated_at": "2024-01-01T00:00:00Z"},
    {"task_id": "task2", "task_name": "Task2", "status": "pending",
     "dependencies": ["task1"], "attempts": 0,
     "created_at": "2024-01-01T00:00:00Z", "updated_at": "2024-01-01T00:00:00Z"}
  ]
}`
	if err := os.WriteFile(filepath.Join(doingDir, "tasks.json"), []byte(existingJSON), 0644); err != nil {
		t.Fatal(err)
	}

	// Use a mock claude that exits 0 but never creates a test script,
	// so task2 will fail — but task1 must be skipped entirely.
	mockDir := t.TempDir()
	mockScript := "#!/bin/sh\nexit 0\n"
	mockPath := filepath.Join(mockDir, "claude")
	if err := os.WriteFile(mockPath, []byte(mockScript), 0755); err != nil {
		t.Fatal(err)
	}
	origPath := os.Getenv("PATH")
	_ = os.Setenv("PATH", mockDir+":"+origPath)
	defer os.Setenv("PATH", origPath)

	// Run the workflow — it will fail on task2 (mock claude doesn't create test script),
	// but the key assertion is about task1 being skipped.
	_ = executeDoingWorkflow("job_resume")

	// Read the persisted tasks.json and verify task1 is still "success"
	loaded, err := executor.LoadTasksJSON(filepath.Join(doingDir, "tasks.json"))
	if err != nil {
		t.Fatalf("failed to load tasks.json after workflow: %v", err)
	}
	status, err := loaded.GetTaskStatus("task1")
	if err != nil {
		t.Fatalf("GetTaskStatus task1 failed: %v", err)
	}
	if status != "success" {
		t.Errorf("expected task1 to remain 'success' after resume, got '%s'", status)
	}
}

// TestExecuteDoingWorkflow_WithMockClaude tests executeDoingWorkflow with mock claude
func TestExecuteDoingWorkflow_WithMockClaude(t *testing.T) {
	// Create mock claude that exits 0 but creates no test script
	mockDir := t.TempDir()
	mockScript := "#!/bin/sh\nexit 0\n"
	mockPath := filepath.Join(mockDir, "claude")
	if err := os.WriteFile(mockPath, []byte(mockScript), 0755); err != nil {
		t.Fatal(err)
	}

	orig, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(orig) }()

	// Set up workspace
	planDir := filepath.Join(dir, ".rick", "jobs", "job_test", "plan")
	if err := os.MkdirAll(planDir, 0755); err != nil {
		t.Fatal(err)
	}
	task1 := "# 依赖关系\n无\n# 任务名称\nTask1\n# 任务目标\nGoal\n# 关键结果\n1. KR1\n# 测试方法\nTest\n"
	if err := os.WriteFile(filepath.Join(planDir, "task1.md"), []byte(task1), 0644); err != nil {
		t.Fatal(err)
	}

	// Set ClaudeCodePath via config
	origPath := os.Getenv("PATH")
	_ = os.Setenv("PATH", mockDir+":"+origPath)
	defer os.Setenv("PATH", origPath)

	// executeDoingWorkflow will fail when ExecuteJob tries to run claude
	// but it should reach that point (covering the workflow setup code)
	err = executeDoingWorkflow("job_test")
	// Error is expected (mock claude doesn't create test scripts)
	t.Logf("executeDoingWorkflow returned: %v", err)
}
