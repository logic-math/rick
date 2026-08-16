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

// TestPlanWorkflow tests the planning workflow
// Note: This test is skipped because it requires actual pi interaction
func TestPlanWorkflow(t *testing.T) {
	t.Skip("Skipping integration test that requires pi - run manually if needed")
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
