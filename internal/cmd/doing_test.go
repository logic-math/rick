package cmd

import (
	"testing"
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
	err := cmd.RunE(cmd, []string{})
	if err == nil {
		t.Error("Expected error when job ID is missing")
	}
}
