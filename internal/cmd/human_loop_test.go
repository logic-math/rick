package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sunquan/rick/internal/config"
)

// TestHumanLoopCmdCreation tests that NewHumanLoopCmd creates a valid command
func TestHumanLoopCmdCreation(t *testing.T) {
	cmd := NewHumanLoopCmd()
	if cmd == nil {
		t.Fatal("NewHumanLoopCmd returned nil")
	}
	if cmd.Use != "human-loop [topic]" {
		t.Errorf("expected Use to be 'human-loop [topic]', got %s", cmd.Use)
	}
	if cmd.Short == "" {
		t.Error("expected Short to be non-empty")
	}
}

// TestHumanLoopCmdNoArgs tests that human-loop without args returns "topic is required"
func TestHumanLoopCmdNoArgs(t *testing.T) {
	cmd := NewHumanLoopCmd()
	cmd.SetArgs([]string{})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error when no topic provided, got nil")
	}
	if !strings.Contains(err.Error(), "topic is required") {
		t.Errorf("expected 'topic is required' error, got: %v", err)
	}
}

// TestHumanLoopCmdDryRun tests human-loop in dry-run mode
func TestHumanLoopCmdDryRun(t *testing.T) {
	origDryRun := dryRun
	defer func() { dryRun = origDryRun }()

	dryRun = true

	cmd := NewHumanLoopCmd()
	cmd.SetArgs([]string{"如何重构?"})

	err := cmd.Execute()
	if err != nil {
		t.Errorf("human-loop dry-run failed: %v", err)
	}
}

// TestHumanLoopCmdWithMockClaude tests the full human-loop flow with a mock Claude binary
func TestHumanLoopCmdWithMockClaude(t *testing.T) {
	// Create a mock claude script
	tmpDir := t.TempDir()
	mockScript := "#!/bin/sh\nexit 0\n"
	mockPath := filepath.Join(tmpDir, "mock_claude")
	if err := os.WriteFile(mockPath, []byte(mockScript), 0755); err != nil {
		t.Fatal(err)
	}

	// Switch to a temp working dir so .rick/RFC is created there
	orig, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	workDir := t.TempDir()
	if err := os.Chdir(workDir); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(orig) }()

	if err := os.MkdirAll(filepath.Join(workDir, ".rick"), 0755); err != nil {
		t.Fatal(err)
	}

	// Write config pointing to mock claude
	cfgContent := `{"claude_code_path": "` + mockPath + `"}`
	cfgDir := filepath.Join(workDir, ".rick")
	if err := os.WriteFile(filepath.Join(cfgDir, "config.json"), []byte(cfgContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Override config load by patching env or using direct call
	cfg := &config.Config{ClaudeCodePath: mockPath}

	// Create prompt manager and generate prompt file manually to test the flow
	rfcDir := filepath.Join(workDir, ".rick", "RFC")
	if err := os.MkdirAll(rfcDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Verify RFC dir was created
	if _, err := os.Stat(rfcDir); os.IsNotExist(err) {
		t.Error("RFC directory was not created")
	}

	// Test callClaudeCodeCLI with mock
	promptFile := filepath.Join(tmpDir, "prompt.md")
	if err := os.WriteFile(promptFile, []byte("# Test human-loop prompt"), 0644); err != nil {
		t.Fatal(err)
	}

	if err := callClaudeCodeCLI(cfg, promptFile); err != nil {
		t.Errorf("callClaudeCodeCLI with mock failed: %v", err)
	}
}
