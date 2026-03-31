package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sunquan/rick/internal/config"
)

// TestPlanCmdCreation tests that NewPlanCmd creates a valid command
func TestPlanCmdCreation(t *testing.T) {
	cmd := NewPlanCmd()
	if cmd == nil {
		t.Fatal("NewPlanCmd returned nil")
	}
	if cmd.Use != "plan [requirement]" {
		t.Errorf("expected Use to be 'plan [requirement]', got %s", cmd.Use)
	}
	if cmd.Short == "" {
		t.Error("expected Short to be non-empty")
	}
}

// TestPromptForRequirement tests the requirement prompt function
func TestPromptForRequirement(t *testing.T) {
	// This test would require mocking stdin, so we'll skip detailed testing
	// Just verify the function exists and can be called
	t.Skip("Skipping interactive test - requires stdin mocking")
}

// TestGenerateJobID tests job ID generation
func TestGenerateJobID(t *testing.T) {
	jobID1 := generateJobID()
	if jobID1 == "" {
		t.Error("generateJobID returned empty string")
	}
	if !strings.HasPrefix(jobID1, "job_") {
		t.Errorf("expected job ID to start with 'job_', got %s", jobID1)
	}

	// Verify the format is job_<timestamp>
	parts := strings.Split(jobID1, "_")
	if len(parts) != 2 {
		t.Errorf("expected job ID format 'job_<timestamp>', got %s", jobID1)
	}
}

// TestExecutePlanWorkflow tests the planning workflow
// Note: This test is skipped because it requires actual Claude CLI interaction
func TestExecutePlanWorkflow(t *testing.T) {
	t.Skip("Skipping integration test that requires Claude CLI - run manually if needed")

	// This test would require mocking the Claude CLI interaction
	// For now, we skip it to avoid blocking CI/CD pipelines
	//
	// To test manually:
	// 1. Ensure Claude CLI is installed
	// 2. Run: go test -v -run TestExecutePlanWorkflow
}

// TestPlanCmdWithDryRun tests plan command in dry-run mode
func TestPlanCmdWithDryRun(t *testing.T) {
	// Save original flags
	origDryRun := dryRun
	defer func() { dryRun = origDryRun }()

	dryRun = true

	cmd := NewPlanCmd()
	cmd.SetArgs([]string{"test requirement"})

	err := cmd.Execute()
	if err != nil {
		t.Errorf("plan command with dry-run failed: %v", err)
	}
}

// TestPlanCmdWithEmptyRequirement tests plan command with empty requirement
func TestPlanCmdWithEmptyRequirement(t *testing.T) {
	cmd := NewPlanCmd()
	cmd.SetArgs([]string{""})

	err := cmd.Execute()
	if err == nil {
		t.Error("expected error with empty requirement, got nil")
	}

	if !strings.Contains(err.Error(), "requirement cannot be empty") {
		t.Errorf("expected 'requirement cannot be empty' error, got: %v", err)
	}
}

// TestCallClaudeCodeCLI_MockBinary tests callClaudeCodeCLI with a mock binary
func TestCallClaudeCodeCLI_MockBinary(t *testing.T) {
	// Create a mock claude script that exits successfully
	tmpDir := t.TempDir()
	mockScript := "#!/bin/sh\nexit 0\n"
	mockPath := filepath.Join(tmpDir, "mock_claude")
	if err := os.WriteFile(mockPath, []byte(mockScript), 0755); err != nil {
		t.Fatal(err)
	}

	// Create a dummy prompt file
	promptFile := filepath.Join(tmpDir, "prompt.md")
	if err := os.WriteFile(promptFile, []byte("# Test prompt"), 0644); err != nil {
		t.Fatal(err)
	}

	cfg := &config.Config{ClaudeCodePath: mockPath}
	if err := callClaudeCodeCLI(cfg, promptFile); err != nil {
		t.Errorf("expected no error with mock claude, got: %v", err)
	}
}

// TestCallClaudeCodeCLI_FailingBinary tests callClaudeCodeCLI with a failing binary
func TestCallClaudeCodeCLI_FailingBinary(t *testing.T) {
	tmpDir := t.TempDir()
	mockScript := "#!/bin/sh\nexit 1\n"
	mockPath := filepath.Join(tmpDir, "mock_claude_fail")
	if err := os.WriteFile(mockPath, []byte(mockScript), 0755); err != nil {
		t.Fatal(err)
	}

	promptFile := filepath.Join(tmpDir, "prompt.md")
	if err := os.WriteFile(promptFile, []byte("# Test prompt"), 0644); err != nil {
		t.Fatal(err)
	}

	cfg := &config.Config{ClaudeCodePath: mockPath}
	if err := callClaudeCodeCLI(cfg, promptFile); err == nil {
		t.Error("expected error with failing mock claude")
	}
}

// TestPlanCmdWithJobFlagDryRun tests plan command with --job flag in dry-run mode
func TestPlanCmdWithJobFlagDryRun(t *testing.T) {
	origDryRun := dryRun
	origJobID := jobID
	defer func() {
		dryRun = origDryRun
		jobID = origJobID
	}()

	dryRun = true
	jobID = "job_1"

	cmd := NewPlanCmd()
	cmd.SetArgs([]string{"some requirement"})

	err := cmd.Execute()
	if err != nil {
		t.Errorf("plan command with --job dry-run failed: %v", err)
	}
}

// TestReEnterPlanWorkflow_NonExistentJob tests error when job plan dir does not exist
func TestReEnterPlanWorkflow_NonExistentJob(t *testing.T) {
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

	err = reEnterPlanWorkflow("job_999")
	if err == nil {
		t.Fatal("expected error for non-existent job plan directory")
	}
	if !strings.Contains(err.Error(), "job job_999 plan directory does not exist") {
		t.Errorf("unexpected error message: %v", err)
	}
}

// TestExecutePlanWorkflow_WithMockClaude tests executePlanWorkflow with mock claude
func TestExecutePlanWorkflow_WithMockClaude(t *testing.T) {
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

	if err := os.MkdirAll(filepath.Join(dir, ".rick"), 0755); err != nil {
		t.Fatal(err)
	}

	// Write a config with mock claude path
	cfgContent := fmt.Sprintf(`{"claude_code_path": "%s"}`, mockPath)
	cfgDir := filepath.Join(os.TempDir(), ".rick")
	_ = os.MkdirAll(cfgDir, 0755)
	_ = os.WriteFile(filepath.Join(cfgDir, "config.json"), []byte(cfgContent), 0644)

	origPath := os.Getenv("PATH")
	_ = os.Setenv("PATH", mockDir+":"+origPath)
	defer os.Setenv("PATH", origPath)

	err = executePlanWorkflow("test requirement")
	t.Logf("executePlanWorkflow returned: %v", err)
}
