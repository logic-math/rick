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
	data := &ExecutionData{
		JobID:        "job_0",
		DebugContent: "## debug1: test error\n**现象**: something went wrong",
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

	learningDir := t.TempDir()
	promptsDir := filepath.Join(learningDir, "prompts")
	if err := os.MkdirAll(promptsDir, 0755); err != nil {
		t.Fatal(err)
	}

	promptFile, err := buildLearningPrompt(data, learningDir, promptsDir)
	if err != nil {
		t.Fatalf("buildLearningPrompt failed: %v", err)
	}

	if promptFile == "" {
		t.Fatal("Prompt file path should not be empty")
	}

	content, err := os.ReadFile(promptFile)
	if err != nil {
		t.Fatalf("failed to read prompt file: %v", err)
	}
	prompt := string(content)

	for _, want := range []string{"task1", "abc123"} {
		if !strings.Contains(prompt, want) {
			t.Errorf("Prompt missing expected content: %s", want)
		}
	}
}

func TestExecutionDataStruct(t *testing.T) {
	data := &ExecutionData{
		JobID:        "job_0",
		DebugContent: "## debug1: test error",
	}

	if data.JobID != "job_0" {
		t.Errorf("Expected JobID 'job_0', got '%s'", data.JobID)
	}

	if data.DebugContent != "## debug1: test error" {
		t.Errorf("Expected DebugContent, got '%s'", data.DebugContent)
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
