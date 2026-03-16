package git

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestInitRepo(t *testing.T) {
	tmpDir := t.TempDir()
	gm := New(tmpDir)

	if err := gm.InitRepo(); err != nil {
		t.Fatalf("InitRepo failed: %v", err)
	}

	gitDir := filepath.Join(tmpDir, ".git")
	if _, err := os.Stat(gitDir); os.IsNotExist(err) {
		t.Errorf("Git directory not created at %s", gitDir)
	}

	if !gm.IsRepository() {
		t.Error("Repository not recognized as valid Git repository")
	}
}

func TestInitRepoInvalidPath(t *testing.T) {
	// Use a path that cannot be created
	invalidPath := "/invalid/path/that/cannot/be/created/repo"
	gm := New(invalidPath)

	err := gm.InitRepo()
	if err == nil {
		t.Error("InitRepo should fail with invalid path")
	}
}

func TestAddFiles(t *testing.T) {
	tmpDir := t.TempDir()
	gm := New(tmpDir)

	// Initialize repo first
	if err := gm.InitRepo(); err != nil {
		t.Fatalf("Failed to initialize repo: %v", err)
	}

	// Create test files
	testFile := filepath.Join(tmpDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("test content"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Add files
	if err := gm.AddFiles([]string{"test.txt"}); err != nil {
		t.Fatalf("AddFiles failed: %v", err)
	}
}

func TestAddFilesEmpty(t *testing.T) {
	tmpDir := t.TempDir()
	gm := New(tmpDir)

	if err := gm.InitRepo(); err != nil {
		t.Fatalf("Failed to initialize repo: %v", err)
	}

	// Empty file list should not fail (just do nothing)
	err := gm.AddFiles([]string{})
	if err != nil {
		t.Errorf("AddFiles with empty list should not fail, got: %v", err)
	}
}

func TestCommit(t *testing.T) {
	tmpDir := t.TempDir()
	gm := New(tmpDir)

	// Initialize repo
	if err := gm.InitRepo(); err != nil {
		t.Fatalf("Failed to initialize repo: %v", err)
	}

	// Create and add a file
	testFile := filepath.Join(tmpDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("test content"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	if err := gm.AddFiles([]string{"test.txt"}); err != nil {
		t.Fatalf("Failed to add files: %v", err)
	}

	// Commit
	if err := gm.Commit("Initial commit"); err != nil {
		t.Fatalf("Commit failed: %v", err)
	}
}

func TestCommitEmptyMessage(t *testing.T) {
	tmpDir := t.TempDir()
	gm := New(tmpDir)

	if err := gm.InitRepo(); err != nil {
		t.Fatalf("Failed to initialize repo: %v", err)
	}

	err := gm.Commit("")
	if err == nil {
		t.Error("Commit should fail with empty message")
	}
}

func TestGetLog(t *testing.T) {
	tmpDir := t.TempDir()
	gm := New(tmpDir)

	// Initialize repo
	if err := gm.InitRepo(); err != nil {
		t.Fatalf("Failed to initialize repo: %v", err)
	}

	// Create and commit a file
	testFile := filepath.Join(tmpDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("test content"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	if err := gm.AddFiles([]string{"test.txt"}); err != nil {
		t.Fatalf("Failed to add files: %v", err)
	}

	if err := gm.Commit("Initial commit"); err != nil {
		t.Fatalf("Failed to commit: %v", err)
	}

	// Get log
	logs, err := gm.GetLog(5)
	if err != nil {
		t.Fatalf("GetLog failed: %v", err)
	}

	if len(logs) != 1 {
		t.Errorf("Expected 1 commit, got %d", len(logs))
	}

	if logs[0].Message != "Initial commit" {
		t.Errorf("Expected message 'Initial commit', got '%s'", logs[0].Message)
	}

	if logs[0].Hash == "" {
		t.Error("Commit hash is empty")
	}

	if logs[0].Author == "" {
		t.Error("Commit author is empty")
	}
}

func TestGetLogWithLimit(t *testing.T) {
	tmpDir := t.TempDir()
	gm := New(tmpDir)

	// Initialize repo
	if err := gm.InitRepo(); err != nil {
		t.Fatalf("Failed to initialize repo: %v", err)
	}

	// Create multiple commits
	for i := 1; i <= 5; i++ {
		testFile := filepath.Join(tmpDir, "test.txt")
		content := []byte("test content " + string(rune(i)))
		if err := os.WriteFile(testFile, content, 0644); err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}

		if err := gm.AddFiles([]string{"test.txt"}); err != nil {
			t.Fatalf("Failed to add files: %v", err)
		}

		if err := gm.Commit("Commit " + string(rune(i))); err != nil {
			t.Fatalf("Failed to commit: %v", err)
		}
	}

	// Get log with limit
	logs, err := gm.GetLog(3)
	if err != nil {
		t.Fatalf("GetLog failed: %v", err)
	}

	if len(logs) != 3 {
		t.Errorf("Expected 3 commits with limit=3, got %d", len(logs))
	}
}

func TestGetLogDefaultLimit(t *testing.T) {
	tmpDir := t.TempDir()
	gm := New(tmpDir)

	// Initialize repo
	if err := gm.InitRepo(); err != nil {
		t.Fatalf("Failed to initialize repo: %v", err)
	}

	// Create and commit a file
	testFile := filepath.Join(tmpDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("test content"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	if err := gm.AddFiles([]string{"test.txt"}); err != nil {
		t.Fatalf("Failed to add files: %v", err)
	}

	if err := gm.Commit("Initial commit"); err != nil {
		t.Fatalf("Failed to commit: %v", err)
	}

	// Get log with default limit (0 or negative should use default)
	logs, err := gm.GetLog(0)
	if err != nil {
		t.Fatalf("GetLog failed: %v", err)
	}

	if len(logs) != 1 {
		t.Errorf("Expected 1 commit, got %d", len(logs))
	}
}

func TestGetCurrentBranch(t *testing.T) {
	tmpDir := t.TempDir()
	gm := New(tmpDir)

	// Initialize repo
	if err := gm.InitRepo(); err != nil {
		t.Fatalf("Failed to initialize repo: %v", err)
	}

	// Create and commit a file
	testFile := filepath.Join(tmpDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("test content"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	if err := gm.AddFiles([]string{"test.txt"}); err != nil {
		t.Fatalf("Failed to add files: %v", err)
	}

	if err := gm.Commit("Initial commit"); err != nil {
		t.Fatalf("Failed to commit: %v", err)
	}

	// Get current branch
	branch, err := gm.GetCurrentBranch()
	if err != nil {
		t.Fatalf("GetCurrentBranch failed: %v", err)
	}

	if branch != "master" && branch != "main" {
		t.Errorf("Expected branch 'master' or 'main', got '%s'", branch)
	}
}

func TestIsRepository(t *testing.T) {
	tmpDir := t.TempDir()
	gm := New(tmpDir)

	// Before initialization
	if gm.IsRepository() {
		t.Error("Should not be recognized as repository before initialization")
	}

	// After initialization
	if err := gm.InitRepo(); err != nil {
		t.Fatalf("Failed to initialize repo: %v", err)
	}

	if !gm.IsRepository() {
		t.Error("Should be recognized as repository after initialization")
	}
}

func TestGetRepoPath(t *testing.T) {
	tmpDir := t.TempDir()
	gm := New(tmpDir)

	if gm.GetRepoPath() != tmpDir {
		t.Errorf("Expected repo path %s, got %s", tmpDir, gm.GetRepoPath())
	}
}

func TestCommitInfoDateParsing(t *testing.T) {
	tmpDir := t.TempDir()
	gm := New(tmpDir)

	// Initialize repo
	if err := gm.InitRepo(); err != nil {
		t.Fatalf("Failed to initialize repo: %v", err)
	}

	// Create and commit a file
	testFile := filepath.Join(tmpDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("test content"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	if err := gm.AddFiles([]string{"test.txt"}); err != nil {
		t.Fatalf("Failed to add files: %v", err)
	}

	if err := gm.Commit("Initial commit"); err != nil {
		t.Fatalf("Failed to commit: %v", err)
	}

	// Get log
	logs, err := gm.GetLog(1)
	if err != nil {
		t.Fatalf("GetLog failed: %v", err)
	}

	if logs[0].Date.IsZero() {
		t.Error("Commit date should not be zero")
	}

	// Check that date is recent (within last minute)
	now := time.Now()
	if logs[0].Date.After(now.Add(1 * time.Minute)) {
		t.Error("Commit date is in the future")
	}

	if logs[0].Date.Before(now.Add(-1 * time.Minute)) {
		t.Error("Commit date is too old")
	}
}
