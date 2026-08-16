package cmd

import (
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
