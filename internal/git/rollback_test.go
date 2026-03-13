package git

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// TestResetToCommit tests the ResetToCommit function
func TestResetToCommit(t *testing.T) {
	// Setup: Create temporary directory
	tmpDir := t.TempDir()

	// Initialize repository
	gm := New(tmpDir)
	if err := gm.InitRepo(); err != nil {
		t.Fatalf("Failed to initialize repo: %v", err)
	}

	// Configure git
	configureGitRepo(t, tmpDir)

	// Create first commit
	file1 := filepath.Join(tmpDir, "file1.txt")
	if err := os.WriteFile(file1, []byte("content1"), 0644); err != nil {
		t.Fatalf("Failed to write file: %v", err)
	}
	if err := gm.AddFiles([]string{"file1.txt"}); err != nil {
		t.Fatalf("Failed to add file: %v", err)
	}
	if err := gm.Commit("First commit"); err != nil {
		t.Fatalf("Failed to commit: %v", err)
	}

	// Get first commit hash
	logs, err := gm.GetLog(1)
	if err != nil || len(logs) == 0 {
		t.Fatalf("Failed to get first commit")
	}
	firstHash := logs[0].Hash

	// Create second commit
	file2 := filepath.Join(tmpDir, "file2.txt")
	if err := os.WriteFile(file2, []byte("content2"), 0644); err != nil {
		t.Fatalf("Failed to write file: %v", err)
	}
	if err := gm.AddFiles([]string{"file2.txt"}); err != nil {
		t.Fatalf("Failed to add file: %v", err)
	}
	if err := gm.Commit("Second commit"); err != nil {
		t.Fatalf("Failed to commit: %v", err)
	}

	// Test ResetToCommit
	rm := NewRollbackManager(gm)
	if err := rm.ResetToCommit(firstHash); err != nil {
		t.Fatalf("Failed to reset to commit: %v", err)
	}

	// Verify we're at first commit
	currentLogs, err := gm.GetLog(1)
	if err != nil || len(currentLogs) == 0 {
		t.Fatalf("Failed to get current commit")
	}
	if currentLogs[0].Hash != firstHash {
		t.Errorf("Expected hash %s, got %s", firstHash, currentLogs[0].Hash)
	}

	// Verify file2 is gone
	if _, err := os.Stat(file2); err == nil {
		t.Errorf("file2 should not exist after reset")
	}
}

// TestResetToCommitInvalid tests ResetToCommit with invalid hash
func TestResetToCommitInvalid(t *testing.T) {
	tmpDir := t.TempDir()
	gm := New(tmpDir)
	if err := gm.InitRepo(); err != nil {
		t.Fatalf("Failed to initialize repo: %v", err)
	}

	rm := NewRollbackManager(gm)

	// Test with invalid hash
	if err := rm.ResetToCommit("invalid_hash_12345"); err == nil {
		t.Errorf("Expected error for invalid hash, got nil")
	}

	// Test with empty hash
	if err := rm.ResetToCommit(""); err == nil {
		t.Errorf("Expected error for empty hash, got nil")
	}
}

// TestResetToVersion tests the ResetToVersion function
func TestResetToVersion(t *testing.T) {
	tmpDir := t.TempDir()
	gm := New(tmpDir)
	if err := gm.InitRepo(); err != nil {
		t.Fatalf("Failed to initialize repo: %v", err)
	}

	configureGitRepo(t, tmpDir)

	// Create first commit with version tag
	file1 := filepath.Join(tmpDir, "file1.txt")
	if err := os.WriteFile(file1, []byte("content1"), 0644); err != nil {
		t.Fatalf("Failed to write file: %v", err)
	}
	if err := gm.AddFiles([]string{"file1.txt"}); err != nil {
		t.Fatalf("Failed to add file: %v", err)
	}
	if err := gm.Commit("First commit"); err != nil {
		t.Fatalf("Failed to commit: %v", err)
	}

	// Create version tag
	vm := NewVersionManager(gm)
	if err := vm.CreateTag("v1.0.0", "Release 1.0.0"); err != nil {
		t.Fatalf("Failed to create tag: %v", err)
	}

	// Create second commit
	file2 := filepath.Join(tmpDir, "file2.txt")
	if err := os.WriteFile(file2, []byte("content2"), 0644); err != nil {
		t.Fatalf("Failed to write file: %v", err)
	}
	if err := gm.AddFiles([]string{"file2.txt"}); err != nil {
		t.Fatalf("Failed to add file: %v", err)
	}
	if err := gm.Commit("Second commit"); err != nil {
		t.Fatalf("Failed to commit: %v", err)
	}

	// Test ResetToVersion
	rm := NewRollbackManager(gm)
	if err := rm.ResetToVersion("v1.0.0"); err != nil {
		t.Fatalf("Failed to reset to version: %v", err)
	}

	// Verify file2 is gone
	if _, err := os.Stat(file2); err == nil {
		t.Errorf("file2 should not exist after reset to v1.0.0")
	}

	// Verify file1 still exists
	if _, err := os.Stat(file1); err != nil {
		t.Errorf("file1 should exist after reset to v1.0.0: %v", err)
	}
}

// TestResetToVersionInvalid tests ResetToVersion with invalid version
func TestResetToVersionInvalid(t *testing.T) {
	tmpDir := t.TempDir()
	gm := New(tmpDir)
	if err := gm.InitRepo(); err != nil {
		t.Fatalf("Failed to initialize repo: %v", err)
	}

	rm := NewRollbackManager(gm)

	// Test with invalid version
	if err := rm.ResetToVersion("v99.99.99"); err == nil {
		t.Errorf("Expected error for non-existent version, got nil")
	}

	// Test with empty version
	if err := rm.ResetToVersion(""); err == nil {
		t.Errorf("Expected error for empty version, got nil")
	}
}

// TestGetDiff tests the GetDiff function
func TestGetDiff(t *testing.T) {
	tmpDir := t.TempDir()
	gm := New(tmpDir)
	if err := gm.InitRepo(); err != nil {
		t.Fatalf("Failed to initialize repo: %v", err)
	}

	configureGitRepo(t, tmpDir)

	// Create first commit
	file1 := filepath.Join(tmpDir, "file1.txt")
	if err := os.WriteFile(file1, []byte("content1"), 0644); err != nil {
		t.Fatalf("Failed to write file: %v", err)
	}
	if err := gm.AddFiles([]string{"file1.txt"}); err != nil {
		t.Fatalf("Failed to add file: %v", err)
	}
	if err := gm.Commit("First commit"); err != nil {
		t.Fatalf("Failed to commit: %v", err)
	}

	logs1, err := gm.GetLog(1)
	if err != nil || len(logs1) == 0 {
		t.Fatalf("Failed to get first commit")
	}
	firstHash := logs1[0].Hash

	// Create second commit with modifications
	if err := os.WriteFile(file1, []byte("modified content1"), 0644); err != nil {
		t.Fatalf("Failed to write file: %v", err)
	}
	file2 := filepath.Join(tmpDir, "file2.txt")
	if err := os.WriteFile(file2, []byte("content2"), 0644); err != nil {
		t.Fatalf("Failed to write file: %v", err)
	}
	if err := gm.AddFiles([]string{"file1.txt", "file2.txt"}); err != nil {
		t.Fatalf("Failed to add files: %v", err)
	}
	if err := gm.Commit("Second commit"); err != nil {
		t.Fatalf("Failed to commit: %v", err)
	}

	logs2, err := gm.GetLog(1)
	if err != nil || len(logs2) == 0 {
		t.Fatalf("Failed to get second commit")
	}
	secondHash := logs2[0].Hash

	// Test GetDiff
	rm := NewRollbackManager(gm)
	diffs, err := rm.GetDiff(firstHash, secondHash)
	if err != nil {
		t.Fatalf("Failed to get diff: %v", err)
	}

	if len(diffs) == 0 {
		t.Errorf("Expected diffs, got none")
	}

	// Check that we have both modified and added files
	hasModified := false
	hasAdded := false
	for _, diff := range diffs {
		if diff.Status == "M" && diff.FilePath == "file1.txt" {
			hasModified = true
		}
		if diff.Status == "A" && diff.FilePath == "file2.txt" {
			hasAdded = true
		}
	}

	if !hasModified {
		t.Errorf("Expected modified file1.txt in diff")
	}
	if !hasAdded {
		t.Errorf("Expected added file2.txt in diff")
	}
}

// TestGetDiffInvalid tests GetDiff with invalid commits
func TestGetDiffInvalid(t *testing.T) {
	tmpDir := t.TempDir()
	gm := New(tmpDir)
	if err := gm.InitRepo(); err != nil {
		t.Fatalf("Failed to initialize repo: %v", err)
	}

	rm := NewRollbackManager(gm)

	// Test with empty commits
	if _, err := rm.GetDiff("", "abc123"); err == nil {
		t.Errorf("Expected error for empty fromCommit, got nil")
	}

	if _, err := rm.GetDiff("abc123", ""); err == nil {
		t.Errorf("Expected error for empty toCommit, got nil")
	}
}

// TestGetFileHistory tests the GetFileHistory function
func TestGetFileHistory(t *testing.T) {
	tmpDir := t.TempDir()
	gm := New(tmpDir)
	if err := gm.InitRepo(); err != nil {
		t.Fatalf("Failed to initialize repo: %v", err)
	}

	configureGitRepo(t, tmpDir)

	// Create first commit
	file1 := filepath.Join(tmpDir, "file1.txt")
	if err := os.WriteFile(file1, []byte("content1"), 0644); err != nil {
		t.Fatalf("Failed to write file: %v", err)
	}
	if err := gm.AddFiles([]string{"file1.txt"}); err != nil {
		t.Fatalf("Failed to add file: %v", err)
	}
	if err := gm.Commit("First commit"); err != nil {
		t.Fatalf("Failed to commit: %v", err)
	}

	// Modify and commit again
	if err := os.WriteFile(file1, []byte("content1 modified"), 0644); err != nil {
		t.Fatalf("Failed to write file: %v", err)
	}
	if err := gm.AddFiles([]string{"file1.txt"}); err != nil {
		t.Fatalf("Failed to add file: %v", err)
	}
	if err := gm.Commit("Second commit"); err != nil {
		t.Fatalf("Failed to commit: %v", err)
	}

	// Test GetFileHistory
	rm := NewRollbackManager(gm)
	history, err := rm.GetFileHistory("file1.txt")
	if err != nil {
		t.Fatalf("Failed to get file history: %v", err)
	}

	if len(history) != 2 {
		t.Errorf("Expected 2 history entries, got %d", len(history))
	}

	// Verify entries are in correct order (newest first)
	if history[0].Message != "Second commit" {
		t.Errorf("Expected first entry to be 'Second commit', got '%s'", history[0].Message)
	}
	if history[1].Message != "First commit" {
		t.Errorf("Expected second entry to be 'First commit', got '%s'", history[1].Message)
	}
}

// TestGetFileHistoryInvalid tests GetFileHistory with invalid file
func TestGetFileHistoryInvalid(t *testing.T) {
	tmpDir := t.TempDir()
	gm := New(tmpDir)
	if err := gm.InitRepo(); err != nil {
		t.Fatalf("Failed to initialize repo: %v", err)
	}

	configureGitRepo(t, tmpDir)

	// Create at least one commit so the repository is not empty
	file := filepath.Join(tmpDir, "test.txt")
	if err := os.WriteFile(file, []byte("test"), 0644); err != nil {
		t.Fatalf("Failed to write file: %v", err)
	}
	if err := gm.AddFiles([]string{"test.txt"}); err != nil {
		t.Fatalf("Failed to add file: %v", err)
	}
	if err := gm.Commit("Initial commit"); err != nil {
		t.Fatalf("Failed to commit: %v", err)
	}

	rm := NewRollbackManager(gm)

	// Test with empty path
	if _, err := rm.GetFileHistory(""); err == nil {
		t.Errorf("Expected error for empty path, got nil")
	}

	// Test with non-existent file - should return empty history (file never existed)
	history, err := rm.GetFileHistory("non_existent.txt")
	if err != nil {
		// It's acceptable to get an error or empty history for non-existent file
		t.Logf("Got error for non-existent file: %v (acceptable)", err)
	} else if len(history) != 0 {
		t.Errorf("Expected empty history for non-existent file, got %d entries", len(history))
	}
}

// TestHasUncommittedChanges tests the HasUncommittedChanges function
func TestHasUncommittedChanges(t *testing.T) {
	tmpDir := t.TempDir()
	gm := New(tmpDir)
	if err := gm.InitRepo(); err != nil {
		t.Fatalf("Failed to initialize repo: %v", err)
	}

	configureGitRepo(t, tmpDir)

	// Create initial commit
	file1 := filepath.Join(tmpDir, "file1.txt")
	if err := os.WriteFile(file1, []byte("content1"), 0644); err != nil {
		t.Fatalf("Failed to write file: %v", err)
	}
	if err := gm.AddFiles([]string{"file1.txt"}); err != nil {
		t.Fatalf("Failed to add file: %v", err)
	}
	if err := gm.Commit("Initial commit"); err != nil {
		t.Fatalf("Failed to commit: %v", err)
	}

	rm := NewRollbackManager(gm)

	// Check no uncommitted changes
	hasChanges, err := rm.HasUncommittedChanges()
	if err != nil {
		t.Fatalf("Failed to check uncommitted changes: %v", err)
	}
	if hasChanges {
		t.Errorf("Expected no uncommitted changes, got true")
	}

	// Modify file
	if err := os.WriteFile(file1, []byte("modified"), 0644); err != nil {
		t.Fatalf("Failed to modify file: %v", err)
	}

	// Check uncommitted changes
	hasChanges, err = rm.HasUncommittedChanges()
	if err != nil {
		t.Fatalf("Failed to check uncommitted changes: %v", err)
	}
	if !hasChanges {
		t.Errorf("Expected uncommitted changes, got false")
	}
}

// TestGetCurrentCommitHash tests the GetCurrentCommitHash function
func TestGetCurrentCommitHash(t *testing.T) {
	tmpDir := t.TempDir()
	gm := New(tmpDir)
	if err := gm.InitRepo(); err != nil {
		t.Fatalf("Failed to initialize repo: %v", err)
	}

	configureGitRepo(t, tmpDir)

	// Create commit
	file1 := filepath.Join(tmpDir, "file1.txt")
	if err := os.WriteFile(file1, []byte("content1"), 0644); err != nil {
		t.Fatalf("Failed to write file: %v", err)
	}
	if err := gm.AddFiles([]string{"file1.txt"}); err != nil {
		t.Fatalf("Failed to add file: %v", err)
	}
	if err := gm.Commit("Test commit"); err != nil {
		t.Fatalf("Failed to commit: %v", err)
	}

	rm := NewRollbackManager(gm)
	hash, err := rm.GetCurrentCommitHash()
	if err != nil {
		t.Fatalf("Failed to get current commit hash: %v", err)
	}

	// Verify hash is valid (40 hex characters for SHA-1)
	if len(hash) != 40 {
		t.Errorf("Expected 40 character hash, got %d: %s", len(hash), hash)
	}

	// Verify it matches the log
	logs, err := gm.GetLog(1)
	if err != nil || len(logs) == 0 {
		t.Fatalf("Failed to get log")
	}
	if hash != logs[0].Hash {
		t.Errorf("Expected hash %s, got %s", logs[0].Hash, hash)
	}
}

// TestValidateCommit tests the ValidateCommit function
func TestValidateCommit(t *testing.T) {
	tmpDir := t.TempDir()
	gm := New(tmpDir)
	if err := gm.InitRepo(); err != nil {
		t.Fatalf("Failed to initialize repo: %v", err)
	}

	configureGitRepo(t, tmpDir)

	// Create commit
	file1 := filepath.Join(tmpDir, "file1.txt")
	if err := os.WriteFile(file1, []byte("content1"), 0644); err != nil {
		t.Fatalf("Failed to write file: %v", err)
	}
	if err := gm.AddFiles([]string{"file1.txt"}); err != nil {
		t.Fatalf("Failed to add file: %v", err)
	}
	if err := gm.Commit("Test commit"); err != nil {
		t.Fatalf("Failed to commit: %v", err)
	}

	logs, err := gm.GetLog(1)
	if err != nil || len(logs) == 0 {
		t.Fatalf("Failed to get log")
	}
	fullHash := logs[0].Hash
	shortHash := fullHash[:7]

	rm := NewRollbackManager(gm)

	// Test with full hash
	validatedHash, err := rm.ValidateCommit(fullHash)
	if err != nil {
		t.Fatalf("Failed to validate full hash: %v", err)
	}
	if validatedHash != fullHash {
		t.Errorf("Expected %s, got %s", fullHash, validatedHash)
	}

	// Test with short hash
	validatedHash, err = rm.ValidateCommit(shortHash)
	if err != nil {
		t.Fatalf("Failed to validate short hash: %v", err)
	}
	if validatedHash != fullHash {
		t.Errorf("Expected %s, got %s", fullHash, validatedHash)
	}

	// Test with invalid hash
	if _, err := rm.ValidateCommit("invalid_hash"); err == nil {
		t.Errorf("Expected error for invalid hash, got nil")
	}

	// Test with empty hash
	if _, err := rm.ValidateCommit(""); err == nil {
		t.Errorf("Expected error for empty hash, got nil")
	}
}

// Helper function to configure git repository
func configureGitRepo(t *testing.T, repoPath string) {
	configs := [][]string{
		{"config", "user.name", "Test User"},
		{"config", "user.email", "test@example.com"},
	}

	for _, config := range configs {
		cmd := exec.Command("git", config...)
		cmd.Dir = repoPath
		if err := cmd.Run(); err != nil {
			t.Fatalf("Failed to configure git: %v", err)
		}
	}
}

// BenchmarkResetToCommit benchmarks the ResetToCommit function
func BenchmarkResetToCommit(b *testing.B) {
	tmpDir := b.TempDir()
	gm := New(tmpDir)
	if err := gm.InitRepo(); err != nil {
		b.Fatalf("Failed to initialize repo: %v", err)
	}

	configureGitRepo(&testing.T{}, tmpDir)

	// Create commits
	for i := 0; i < 5; i++ {
		file := filepath.Join(tmpDir, "file.txt")
		if err := os.WriteFile(file, []byte(string(rune(i))), 0644); err != nil {
			b.Fatalf("Failed to write file: %v", err)
		}
		if err := gm.AddFiles([]string{"file.txt"}); err != nil {
			b.Fatalf("Failed to add file: %v", err)
		}
		if err := gm.Commit("Commit " + string(rune(i))); err != nil {
			b.Fatalf("Failed to commit: %v", err)
		}
	}

	logs, err := gm.GetLog(1)
	if err != nil || len(logs) == 0 {
		b.Fatalf("Failed to get log")
	}
	targetHash := logs[0].Hash

	rm := NewRollbackManager(gm)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Reset to a previous commit and back
		if err := rm.ResetToCommit(targetHash); err != nil {
			b.Fatalf("Failed to reset: %v", err)
		}
	}
}
