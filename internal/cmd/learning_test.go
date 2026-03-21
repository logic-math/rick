package cmd

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sunquan/rick/internal/executor"
)

func TestBuildLearningPrompt(t *testing.T) {
	// Create test data
	data := &ExecutionData{
		JobID:        "job_0",
		DebugContent: "Test debug content",
		TasksJSON: &executor.TasksJSON{
			Version: "1.0",
			Tasks: []executor.TaskState{
				{
					TaskID:     "task1",
					TaskName:   "Test Task",
					TaskFile:   "task1.md",
					Status:     "success",
					CommitHash: "abc123",
					Attempts:   1,
				},
			},
		},
	}
	
	// Build prompt
	prompt, err := buildLearningPrompt(data, "/tmp/test-learning")
	if err != nil {
		t.Fatalf("buildLearningPrompt failed: %v", err)
	}

	// Verify prompt contains expected sections
	if prompt == "" {
		t.Fatal("Prompt should not be empty")
	}

	// Check for key sections
	expectedSections := []string{
		"task1",
		"abc123",
	}
	
	for _, section := range expectedSections {
		if !strings.Contains(prompt, section) {
			t.Errorf("Prompt missing expected section: %s", section)
		}
	}
}

func TestExecutionDataStruct(t *testing.T) {
	data := &ExecutionData{
		JobID:        "job_0",
		DebugContent: "test",
		TasksJSON:    nil,
	}
	
	if data.JobID != "job_0" {
		t.Errorf("Expected JobID 'job_0', got '%s'", data.JobID)
	}
	
	if data.DebugContent != "test" {
		t.Errorf("Expected DebugContent 'test', got '%s'", data.DebugContent)
	}
}

// TestExecuteLearningWorkflow_NoDoingDir tests executeLearningWorkflow with missing doing dir
func TestExecuteLearningWorkflow_NoDoingDir(t *testing.T) {
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

	err = executeLearningWorkflow("job_test")
	if err == nil {
		t.Fatal("expected error for missing doing dir")
	}
}

// TestExecuteLearningWorkflow_WithMockClaude tests executeLearningWorkflow with mock claude
func TestExecuteLearningWorkflow_WithMockClaude(t *testing.T) {
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

	// Set up workspace with doing dir and tasks.json
	doingDir := filepath.Join(dir, ".rick", "jobs", "job_test", "doing")
	if err := os.MkdirAll(doingDir, 0755); err != nil {
		t.Fatal(err)
	}

	now := "2026-01-01T00:00:00Z"
	tasksData := map[string]interface{}{
		"version": "1.0", "created_at": now, "updated_at": now,
		"tasks": []map[string]interface{}{
			{"task_id": "task1", "task_name": "T1", "status": "success",
				"commit_hash": "abc", "dependencies": []string{}, "attempts": 1,
				"created_at": now, "updated_at": now},
		},
	}
	tasksJSON, _ := json.Marshal(tasksData)
	if err := os.WriteFile(filepath.Join(doingDir, "tasks.json"), tasksJSON, 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(doingDir, "debug.md"), []byte("# debug"), 0644); err != nil {
		t.Fatal(err)
	}

	origPath := os.Getenv("PATH")
	_ = os.Setenv("PATH", mockDir+":"+origPath)
	defer os.Setenv("PATH", origPath)

	err = executeLearningWorkflow("job_test")
	t.Logf("executeLearningWorkflow returned: %v", err)
}
