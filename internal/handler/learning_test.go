package handler

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sunquan/rick/internal/executor"
)

func writeTasksJSON(t *testing.T, dir string, tasks []executor.TaskState) {
	t.Helper()
	now := "2026-01-01T00:00:00Z"
	data := map[string]interface{}{
		"version":    "1.0",
		"created_at": now,
		"updated_at": now,
		"tasks":      tasks,
	}
	b, err := json.Marshal(data)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "tasks.json"), b, 0644); err != nil {
		t.Fatal(err)
	}
}

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

// TestLearning_NoDoingDir tests Learning with missing doing dir.
func TestLearning_NoDoingDir(t *testing.T) {
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

	err = Learning("job_test", Options{})
	if err == nil {
		t.Fatal("expected error for missing doing dir")
	}
}

// TestBuildLearningPromptInjectsDraftDir verifies draft_dir is fully resolved in the output prompt.
func TestBuildLearningPromptInjectsDraftDir(t *testing.T) {
	tmpDir := t.TempDir()
	rickDir := filepath.Join(tmpDir, ".rick")
	doingDir := filepath.Join(rickDir, "jobs", "job_test", "doing")
	if err := os.MkdirAll(doingDir, 0755); err != nil {
		t.Fatalf("MkdirAll doingDir: %v", err)
	}

	oldDir, _ := os.Getwd()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("Chdir: %v", err)
	}
	defer os.Chdir(oldDir) //nolint:errcheck

	data := &ExecutionData{
		JobID:        "job_test",
		RickDir:      rickDir,
		TaskMDPaths:  []string{},
		ActPathFiles: []string{},
	}

	learningDir := filepath.Join(rickDir, "jobs", "job_test", "learning")
	promptsDir := filepath.Join(learningDir, "prompts")
	if err := os.MkdirAll(promptsDir, 0755); err != nil {
		t.Fatalf("MkdirAll promptsDir: %v", err)
	}

	promptFile, err := buildLearningPrompt(data, learningDir, promptsDir)
	if err != nil {
		t.Fatalf("buildLearningPrompt error: %v", err)
	}

	content, err := os.ReadFile(promptFile)
	if err != nil {
		t.Fatalf("read prompt file: %v", err)
	}

	s := string(content)
	if strings.Contains(s, "{{draft_dir}}") {
		t.Errorf("prompt still contains unreplaced {{draft_dir}}")
	}
	expectedDraftPath := filepath.Join(rickDir, "draft")
	if !strings.Contains(s, expectedDraftPath) {
		snippet := s
		if len(snippet) > 300 {
			snippet = snippet[:300]
		}
		t.Errorf("prompt does not contain draft path %q; snippet:\n%s", expectedDraftPath, snippet)
	}
}

// TestLearning_WithMockPi tests Learning with mock pi.
func TestLearning_WithMockPi(t *testing.T) {
	mockDir := t.TempDir()
	mockScript := "#!/bin/sh\nexit 0\n"
	mockPath := filepath.Join(mockDir, "pi")
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

	// Isolate HOME and point pi_path at the mock binary so the workflow calls
	// the mock directly — never the real managed runtime or PATH pi (learning
	// drives runtime.CallCLI, not the legacy claude CLI).
	home := t.TempDir()
	t.Setenv("HOME", home)
	cfgDir := filepath.Join(home, ".rick")
	if err := os.MkdirAll(cfgDir, 0755); err != nil {
		t.Fatal(err)
	}
	cfgContent := fmt.Sprintf(`{"pi_path": "%s"}`, mockPath)
	if err := os.WriteFile(filepath.Join(cfgDir, "config.json"), []byte(cfgContent), 0644); err != nil {
		t.Fatal(err)
	}

	err = Learning("job_test", Options{})
	t.Logf("Learning returned: %v", err)
}

func TestCollectExecutionData_NoDoingDir(t *testing.T) {
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

	_, err = collectExecutionData("job_test")
	if err == nil {
		t.Fatal("expected error for missing doing dir")
	}
}

func TestCollectExecutionData_WithData(t *testing.T) {
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

	doingDir := filepath.Join(dir, ".rick", "jobs", "job_test", "doing")
	if err := os.MkdirAll(doingDir, 0755); err != nil {
		t.Fatal(err)
	}
	// Create debug.md
	if err := os.WriteFile(filepath.Join(doingDir, "debug.md"), []byte("## task1: did work\nsome content"), 0644); err != nil {
		t.Fatal(err)
	}
	// Create tasks.json
	writeTasksJSON(t, doingDir, []executor.TaskState{
		{TaskID: "task1", TaskName: "T1", Status: "success", CommitHash: "abc"},
	})
	data, err := collectExecutionData("job_test")
	if err != nil {
		t.Errorf("expected no error, got: %v", err)
	}
	if data == nil {
		t.Fatal("expected non-nil data")
	}
	if data.JobID != "job_test" {
		t.Errorf("expected job_id=job_test, got %s", data.JobID)
	}
}

func TestCollectExecutionData_NoDebugMD(t *testing.T) {
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

	doingDir := filepath.Join(dir, ".rick", "jobs", "job_test", "doing")
	if err := os.MkdirAll(doingDir, 0755); err != nil {
		t.Fatal(err)
	}
	// Create tasks.json but no debug.md
	writeTasksJSON(t, doingDir, []executor.TaskState{
		{TaskID: "task1", TaskName: "T1", Status: "success", CommitHash: "abc"},
	})
	data, err := collectExecutionData("job_test")
	if err != nil {
		t.Errorf("expected no error even without debug.md, got: %v", err)
	}
	if data != nil && data.DebugContent != "" {
		t.Logf("debug content unexpectedly set: %s", data.DebugContent)
	}
}
