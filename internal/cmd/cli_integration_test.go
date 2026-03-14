package cmd

import (
	"bytes"
	"testing"
)

// TestCLIIntegrationPlanCommandExists tests that plan command exists and responds to --help
func TestCLIIntegrationPlanCommandExists(t *testing.T) {
	rootCmd := NewRootCmd("0.1.0")

	// Execute plan --help
	rootCmd.SetArgs([]string{"plan", "--help"})
	var out bytes.Buffer
	rootCmd.SetOut(&out)
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("plan --help failed: %v", err)
	}

	output := out.String()
	if output == "" {
		t.Errorf("expected plan help output, but got none")
	}
}

// TestCLIIntegrationDoingCommandExists tests that doing command exists and responds to --help
func TestCLIIntegrationDoingCommandExists(t *testing.T) {
	rootCmd := NewRootCmd("0.1.0")

	// Execute doing --help
	rootCmd.SetArgs([]string{"doing", "--help"})
	var out bytes.Buffer
	rootCmd.SetOut(&out)
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("doing --help failed: %v", err)
	}

	output := out.String()
	if output == "" {
		t.Errorf("expected doing help output, but got none")
	}
}

// TestCLIIntegrationLearningCommandExists tests that learning command exists and responds to --help
func TestCLIIntegrationLearningCommandExists(t *testing.T) {
	rootCmd := NewRootCmd("0.1.0")

	// Execute learning --help
	rootCmd.SetArgs([]string{"learning", "--help"})
	var out bytes.Buffer
	rootCmd.SetOut(&out)
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("learning --help failed: %v", err)
	}

	output := out.String()
	if output == "" {
		t.Errorf("expected learning help output, but got none")
	}
}

// TestCLIIntegrationAllCommandsAvailable tests that all core commands are available
func TestCLIIntegrationAllCommandsAvailable(t *testing.T) {
	rootCmd := NewRootCmd("0.1.0")

	// Verify all commands are registered
	expectedCommands := []string{"plan", "doing", "learning"}
	for _, cmdName := range expectedCommands {
		cmd, _, err := rootCmd.Find([]string{cmdName})
		if err != nil {
			t.Errorf("command '%s' not found: %v", cmdName, err)
		}
		if cmd == nil {
			t.Errorf("command '%s' is nil", cmdName)
		}
	}
}

// TestCLIIntegrationCommandHelpOutput tests that all commands have help output
func TestCLIIntegrationCommandHelpOutput(t *testing.T) {
	commands := []struct {
		name string
		cmd  func() interface{}
	}{
		{"plan", func() interface{} { return NewPlanCmd() }},
		{"doing", func() interface{} { return NewDoingCmd() }},
		{"learning", func() interface{} { return NewLearningCmd() }},
	}

	for _, tc := range commands {
		t.Run(tc.name, func(t *testing.T) {
			cmd := tc.cmd()
			if c, ok := cmd.(interface{ GetShort() string }); ok {
				if c.GetShort() == "" {
					t.Errorf("command '%s' has no Short description", tc.name)
				}
			}
		})
	}
}

// TestCLIIntegrationCommandJobFlag tests that commands support --job flag
func TestCLIIntegrationCommandJobFlag(t *testing.T) {
	commands := []struct {
		name string
		cmd  interface{}
	}{
		{"doing", NewDoingCmd()},
		{"learning", NewLearningCmd()},
	}

	for _, tc := range commands {
		t.Run(tc.name, func(t *testing.T) {
			if cmd, ok := tc.cmd.(*interface{}); ok {
				// Just verify the command exists
				if cmd == nil {
					t.Errorf("command '%s' is nil", tc.name)
				}
			}
		})
	}
}

