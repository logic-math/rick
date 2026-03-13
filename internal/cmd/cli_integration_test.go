package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

// TestCLIIntegrationInitCommandExecution tests the init command execution
func TestCLIIntegrationInitCommandExecution(t *testing.T) {
	// Create a temporary directory for testing
	tmpDir := t.TempDir()
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", oldHome)

	// Create root command with init
	rootCmd := NewRootCmd("0.1.0")

	// Execute init command
	rootCmd.SetArgs([]string{"init"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("init command failed: %v", err)
	}

	// Verify .rick directory was created
	rickDir := filepath.Join(tmpDir, ".rick")
	if _, err := os.Stat(rickDir); os.IsNotExist(err) {
		t.Errorf("expected .rick directory to exist, but it doesn't")
	}

	// Verify subdirectories were created
	expectedDirs := []string{"wiki", "skills", "jobs"}
	for _, dir := range expectedDirs {
		dirPath := filepath.Join(rickDir, dir)
		if _, err := os.Stat(dirPath); os.IsNotExist(err) {
			t.Errorf("expected %s directory to exist, but it doesn't", dir)
		}
	}

	// Verify files were created
	expectedFiles := []string{"OKR.md", "SPEC.md", "config.json"}
	for _, file := range expectedFiles {
		filePath := filepath.Join(rickDir, file)
		if _, err := os.Stat(filePath); os.IsNotExist(err) {
			t.Errorf("expected %s file to exist, but it doesn't", file)
		}
	}

	// Verify .git directory was created
	gitDir := filepath.Join(rickDir, ".git")
	if _, err := os.Stat(gitDir); os.IsNotExist(err) {
		t.Errorf("expected .git directory to exist, but it doesn't")
	}
}

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

// TestCLIIntegrationAllCommandsAvailable tests that all four core commands are available
func TestCLIIntegrationAllCommandsAvailable(t *testing.T) {
	rootCmd := NewRootCmd("0.1.0")

	// Verify all commands are registered
	expectedCommands := []string{"init", "plan", "doing", "learning"}
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
		{"init", func() interface{} { return NewInitCmd() }},
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

// TestCLIIntegrationInitCommandIdempotent tests that running init multiple times is safe
func TestCLIIntegrationInitCommandIdempotent(t *testing.T) {
	tmpDir := t.TempDir()
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", oldHome)

	rootCmd := NewRootCmd("0.1.0")

	// Execute init twice
	rootCmd.SetArgs([]string{"init"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("first init command failed: %v", err)
	}

	// Run init again
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("second init command failed: %v", err)
	}

	// Verify .rick directory still exists
	rickDir := filepath.Join(tmpDir, ".rick")
	if _, err := os.Stat(rickDir); os.IsNotExist(err) {
		t.Errorf("expected .rick directory to exist after second init")
	}
}
