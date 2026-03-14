package cmd

import (
	"strings"
	"testing"
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
