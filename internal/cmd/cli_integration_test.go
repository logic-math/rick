package cmd

import (
	"bytes"
	"testing"
)

// TestCLIIntegrationPlanCommandExists tests that plan command exists and responds to --help
// TestCLIIntegrationAllCommandsAvailable tests that remaining core commands are
// available and the removed ones (job_53 Bug #5: plan/doing/learning/ctrl) are gone.
func TestCLIIntegrationAllCommandsAvailable(t *testing.T) {
	rootCmd := NewRootCmd("0.1.0")

	// Verify remaining commands are registered
	expectedCommands := []string{"easy", "human-loop", "dream", "super", "web", "tools"}
	for _, cmdName := range expectedCommands {
		cmd, _, err := rootCmd.Find([]string{cmdName})
		if err != nil {
			t.Errorf("command '%s' not found: %v", cmdName, err)
		}
		if cmd == nil {
			t.Errorf("command '%s' is nil", cmdName)
		}
	}
	// Removed commands must be gone (unknown command)
	for _, cmdName := range []string{"plan", "doing", "learning", "ctrl"} {
		if cmd, _, err := rootCmd.Find([]string{cmdName}); err == nil && cmd != nil && cmd.Name() == cmdName {
			t.Errorf("removed command '%s' still registered", cmdName)
		}
	}
}

// TestCLIIntegrationCommandHelpOutput tests that all commands have help output
// TestCLIIntegrationCommandJobFlag tests that commands support --job flag

// TestCLIIntegrationUpdatePiCommandExists tests that tools update-pi exists and
// responds to --help (v4.1.0 feature: pi runtime / extensions / models update).
func TestCLIIntegrationUpdatePiCommandExists(t *testing.T) {
	rootCmd := NewRootCmd("0.1.0")

	rootCmd.SetArgs([]string{"tools", "update-pi", "--help"})
	var out bytes.Buffer
	rootCmd.SetOut(&out)
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("tools update-pi --help failed: %v", err)
	}

	output := out.String()
	if output == "" {
		t.Errorf("expected update-pi help output, but got none")
	}
	for _, want := range []string{"update-pi", "pi-subagents", "extensions"} {
		if !bytes.Contains([]byte(output), []byte(want)) {
			t.Errorf("update-pi help should mention %q", want)
		}
	}
}
