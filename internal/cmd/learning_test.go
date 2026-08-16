package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sunquan/rick/internal/executor"
	"github.com/sunquan/rick/internal/prompt"
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

// TestLearningTemplateHasDraftDir verifies the embedded learning template declares {{draft_dir}}.
func TestLearningTemplateHasDraftDir(t *testing.T) {
	pm := prompt.NewPromptManager("")
	tmpl, err := pm.LoadTemplate("learning")
	if err != nil {
		t.Fatalf("failed to load learning template: %v", err)
	}
	if !strings.Contains(tmpl.Content, "{{draft_dir}}") {
		t.Errorf("learning.md template does not contain '{{draft_dir}}'")
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

	err = executeLearningWorkflow("job_test")
	t.Logf("executeLearningWorkflow returned: %v", err)
}
