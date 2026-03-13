package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/sunquan/rick/internal/config"
	"github.com/sunquan/rick/internal/workspace"
)

// TestNewInitCmd tests that NewInitCmd creates a valid cobra command
func TestNewInitCmd(t *testing.T) {
	cmd := NewInitCmd()
	if cmd == nil {
		t.Fatal("NewInitCmd returned nil")
	}
	if cmd.Use != "init" {
		t.Errorf("expected Use to be 'init', got %s", cmd.Use)
	}
	if cmd.Short == "" {
		t.Error("expected Short to be non-empty")
	}
	if cmd.Long == "" {
		t.Error("expected Long to be non-empty")
	}
}

// TestInitCmdDryRun tests that init command works in dry-run mode
func TestInitCmdDryRun(t *testing.T) {
	// Save original values
	origVerbose := verbose
	origDryRun := dryRun
	defer func() {
		verbose = origVerbose
		dryRun = origDryRun
	}()

	// Set dry-run mode
	dryRun = true

	cmd := NewInitCmd()
	err := cmd.RunE(cmd, []string{})
	if err != nil {
		t.Fatalf("init command failed in dry-run mode: %v", err)
	}
}

// TestInitCmdVerbose tests that init command works in verbose mode
func TestInitCmdVerbose(t *testing.T) {
	// Save original values
	origVerbose := verbose
	defer func() {
		verbose = origVerbose
	}()

	// Set verbose mode
	verbose = true

	cmd := NewInitCmd()
	err := cmd.RunE(cmd, []string{})
	if err != nil {
		t.Fatalf("init command failed in verbose mode: %v", err)
	}
}

// TestInitCmdBasicExecution tests that init command can be executed
func TestInitCmdBasicExecution(t *testing.T) {
	cmd := NewInitCmd()
	if cmd.RunE == nil {
		t.Error("expected RunE to be set")
	}
}

// TestInitGitRepo tests the initGitRepo function
func TestInitGitRepo(t *testing.T) {
	tmpDir := t.TempDir()

	// Test initializing git in a new directory
	err := initGitRepo(tmpDir)
	if err != nil {
		t.Fatalf("initGitRepo failed: %v", err)
	}

	// Verify .git directory exists
	gitDir := filepath.Join(tmpDir, ".git")
	if _, err := os.Stat(gitDir); os.IsNotExist(err) {
		t.Error("expected .git directory to be created")
	}

	// Verify .gitignore was created
	gitignorePath := filepath.Join(tmpDir, ".gitignore")
	if _, err := os.Stat(gitignorePath); os.IsNotExist(err) {
		t.Error("expected .gitignore to be created")
	}
}

// TestInitGitRepoIdempotent tests that initGitRepo can be called multiple times
func TestInitGitRepoIdempotent(t *testing.T) {
	tmpDir := t.TempDir()

	// First call
	err := initGitRepo(tmpDir)
	if err != nil {
		t.Fatalf("first initGitRepo failed: %v", err)
	}

	// Second call (should not fail)
	err = initGitRepo(tmpDir)
	if err != nil {
		t.Fatalf("second initGitRepo failed: %v", err)
	}

	// Verify .git still exists
	gitDir := filepath.Join(tmpDir, ".git")
	if _, err := os.Stat(gitDir); os.IsNotExist(err) {
		t.Error("expected .git directory to still exist")
	}
}

// TestInitGitignoreContent tests that .gitignore has correct content
func TestInitGitignoreContent(t *testing.T) {
	tmpDir := t.TempDir()

	err := initGitRepo(tmpDir)
	if err != nil {
		t.Fatalf("initGitRepo failed: %v", err)
	}

	gitignorePath := filepath.Join(tmpDir, ".gitignore")
	content, err := os.ReadFile(gitignorePath)
	if err != nil {
		t.Fatalf("failed to read .gitignore: %v", err)
	}

	contentStr := string(content)
	if contentStr == "" {
		t.Error(".gitignore is empty")
	}

	// Check for expected entries
	expectedEntries := []string{"*.log", ".DS_Store"}
	for _, entry := range expectedEntries {
		if !contains(contentStr, entry) {
			t.Errorf("expected .gitignore to contain %q", entry)
		}
	}
}

// TestWorkspaceIntegration tests workspace initialization
func TestWorkspaceIntegration(t *testing.T) {
	// Create workspace
	ws, err := workspace.New()
	if err != nil {
		t.Fatalf("failed to create workspace: %v", err)
	}

	if ws == nil {
		t.Error("workspace is nil")
	}
}

// TestConfigIntegration tests config initialization
func TestConfigIntegration(t *testing.T) {
	cfg := config.GetDefaultConfig()
	if cfg == nil {
		t.Error("failed to get default config")
	}

	if cfg.MaxRetries != 5 {
		t.Errorf("expected MaxRetries to be 5, got %d", cfg.MaxRetries)
	}
}

// Helper function to check if string contains substring
func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
