package git

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// runGitCommand is a helper function to run git commands
func runGitCommand(dir string, args ...string) error {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	return cmd.Run()
}

// TestGitIntegrationFullWorkflow tests the complete Git workflow
// including initialization, file operations, commits, versioning, and rollback
func TestGitIntegrationFullWorkflow(t *testing.T) {
	tmpDir := t.TempDir()
	gm := New(tmpDir)

	// Task 1: Verify Git repository initialization
	t.Run("Task1_RepositoryInitialization", func(t *testing.T) {
		if err := gm.InitRepo(); err != nil {
			t.Fatalf("Failed to initialize repo: %v", err)
		}

		// Configure git user
		configCmd := []string{"config", "user.email", "test@example.com"}
		if err := runGitCommand(tmpDir, configCmd...); err != nil {
			t.Fatalf("Failed to configure git email: %v", err)
		}
		configCmd = []string{"config", "user.name", "Test User"}
		if err := runGitCommand(tmpDir, configCmd...); err != nil {
			t.Fatalf("Failed to configure git name: %v", err)
		}

		gitDir := filepath.Join(tmpDir, ".git")
		if _, err := os.Stat(gitDir); os.IsNotExist(err) {
			t.Errorf("Git directory not created")
		}

		if !gm.IsRepository() {
			t.Error("Repository not recognized as valid")
		}

		// Create initial commit to verify branch works
		file := filepath.Join(tmpDir, "init.txt")
		if err := os.WriteFile(file, []byte("initial"), 0644); err != nil {
			t.Fatalf("Failed to create file: %v", err)
		}

		if err := gm.AddFiles([]string{file}); err != nil {
			t.Fatalf("Failed to add file: %v", err)
		}

		if err := gm.Commit("initial: setup"); err != nil {
			t.Fatalf("Failed to commit: %v", err)
		}

		branch, err := gm.GetCurrentBranch()
		if err != nil || branch == "" {
			t.Error("Failed to get current branch")
		}
	})

	// Task 2: Verify file addition and commit
	t.Run("Task2_FileAdditionAndCommit", func(t *testing.T) {
		// Create test files
		file1 := filepath.Join(tmpDir, "file1.txt")
		file2 := filepath.Join(tmpDir, "file2.txt")

		if err := os.WriteFile(file1, []byte("content1"), 0644); err != nil {
			t.Fatalf("Failed to create file1: %v", err)
		}
		if err := os.WriteFile(file2, []byte("content2"), 0644); err != nil {
			t.Fatalf("Failed to create file2: %v", err)
		}

		// Add files
		if err := gm.AddFiles([]string{file1, file2}); err != nil {
			t.Fatalf("Failed to add files: %v", err)
		}

		// Commit
		if err := gm.Commit("feat: add initial files"); err != nil {
			t.Fatalf("Failed to commit: %v", err)
		}

		// Verify commit was created
		log, err := gm.GetLog(1)
		if err != nil {
			t.Fatalf("Failed to get log: %v", err)
		}

		if len(log) == 0 {
			t.Error("No commits found after adding files")
		}

		if !strings.Contains(log[0].Message, "add initial files") {
			t.Errorf("Commit message not found: %v", log[0].Message)
		}
	})

	// Task 3: Verify auto-commit system
	t.Run("Task3_AutoCommitSystem", func(t *testing.T) {
		// Create a new file for auto-commit test
		file3 := filepath.Join(tmpDir, "file3.txt")
		if err := os.WriteFile(file3, []byte("auto-commit content"), 0644); err != nil {
			t.Fatalf("Failed to create file3: %v", err)
		}

		// Add the file to staging
		if err := gm.AddFiles([]string{file3}); err != nil {
			t.Fatalf("Failed to add file3: %v", err)
		}

		// Use AutoCommitter for auto-commit
		ac := NewAutoCommitter(gm)
		if err := ac.CommitTask("task_1", "Task 1: Add file3"); err != nil {
			t.Fatalf("CommitTask failed: %v", err)
		}

		// Verify the commit
		log, err := gm.GetLog(1)
		if err != nil {
			t.Fatalf("Failed to get log: %v", err)
		}

		if len(log) == 0 {
			t.Error("No commits found after auto-commit")
		}

		if !strings.Contains(log[0].Message, "task_1") {
			t.Errorf("Task commit not found: %v", log[0].Message)
		}
	})

	// Task 4: Verify version management
	t.Run("Task4_VersionManagement", func(t *testing.T) {
		// Create initial version
		vm := NewVersionManager(gm)
		if err := vm.CreateTag("v1.0.0", "Release v1.0.0"); err != nil {
			t.Fatalf("Failed to create tag: %v", err)
		}

		// Get current version
		version, err := vm.GetCurrentVersion()
		if err != nil {
			t.Fatalf("Failed to get current version: %v", err)
		}

		if version != "v1.0.0" {
			t.Errorf("Expected version v1.0.0, got %s", version)
		}

		// List versions
		versions, err := vm.ListVersions()
		if err != nil {
			t.Fatalf("Failed to list versions: %v", err)
		}

		if len(versions) == 0 {
			t.Error("No versions found")
		}

		found := false
		for _, v := range versions {
			if v.Tag == "v1.0.0" {
				found = true
				break
			}
		}

		if !found {
			t.Error("v1.0.0 not found in versions list")
		}
	})

	// Task 5: Verify rollback and recovery
	t.Run("Task5_RollbackAndRecovery", func(t *testing.T) {
		// Create another file and commit
		file4 := filepath.Join(tmpDir, "file4.txt")
		if err := os.WriteFile(file4, []byte("rollback test"), 0644); err != nil {
			t.Fatalf("Failed to create file4: %v", err)
		}

		if err := gm.AddFiles([]string{file4}); err != nil {
			t.Fatalf("Failed to add file4: %v", err)
		}

		if err := gm.Commit("feat: add file4 for rollback test"); err != nil {
			t.Fatalf("Failed to commit file4: %v", err)
		}

		// Get the current commit hash
		log, err := gm.GetLog(1)
		if err != nil {
			t.Fatalf("Failed to get log: %v", err)
		}

		if len(log) == 0 {
			t.Fatal("No commits found")
		}

		currentHash := log[0].Hash

		// Create another file and commit
		file5 := filepath.Join(tmpDir, "file5.txt")
		if err := os.WriteFile(file5, []byte("another file"), 0644); err != nil {
			t.Fatalf("Failed to create file5: %v", err)
		}

		if err := gm.AddFiles([]string{file5}); err != nil {
			t.Fatalf("Failed to add file5: %v", err)
		}

		if err := gm.Commit("feat: add file5"); err != nil {
			t.Fatalf("Failed to commit file5: %v", err)
		}

		// Verify file5 exists
		if _, err := os.Stat(file5); os.IsNotExist(err) {
			t.Error("file5 should exist before rollback")
		}

		// Rollback to previous commit
		rm := NewRollbackManager(gm)
		if err := rm.ResetToCommit(currentHash); err != nil {
			t.Fatalf("Failed to reset to commit: %v", err)
		}

		// Verify file5 is gone
		if _, err := os.Stat(file5); !os.IsNotExist(err) {
			t.Error("file5 should not exist after rollback")
		}

		// Verify we're at the correct commit
		log, err = gm.GetLog(1)
		if err != nil {
			t.Fatalf("Failed to get log after rollback: %v", err)
		}

		if log[0].Hash != currentHash {
			t.Errorf("Expected commit %s, got %s", currentHash, log[0].Hash)
		}
	})

	// Task 6: Verify commit history
	t.Run("Task6_CommitHistoryVerification", func(t *testing.T) {
		// Get full commit history
		log, err := gm.GetLog(100)
		if err != nil {
			t.Fatalf("Failed to get log: %v", err)
		}

		if len(log) == 0 {
			t.Error("No commits found in history")
		}

		// Verify commit structure
		for _, commit := range log {
			if commit.Hash == "" {
				t.Error("Commit hash is empty")
			}
			if commit.Message == "" {
				t.Error("Commit message is empty")
			}
			if commit.Author == "" {
				t.Error("Commit author is empty")
			}
			if commit.Date.IsZero() {
				t.Error("Commit date is zero")
			}
		}

		// Verify commit messages contain expected patterns
		hasInitialCommit := false
		hasTaskCommit := false
		hasFileCommit := false

		for _, commit := range log {
			if strings.Contains(commit.Message, "add initial files") {
				hasInitialCommit = true
			}
			if strings.Contains(commit.Message, "task_1") {
				hasTaskCommit = true
			}
			if strings.Contains(commit.Message, "file4") {
				hasFileCommit = true
			}
		}

		if !hasInitialCommit {
			t.Error("Initial commit not found in history")
		}
		if !hasTaskCommit {
			t.Error("Task commit not found in history")
		}
		if !hasFileCommit {
			t.Error("File commit not found in history")
		}
	})
}

// TestGitIntegrationJobCommit tests job-level commits
func TestGitIntegrationJobCommit(t *testing.T) {
	tmpDir := t.TempDir()
	gm := New(tmpDir)

	if err := gm.InitRepo(); err != nil {
		t.Fatalf("Failed to initialize repo: %v", err)
	}

	// Configure git user
	runGitCommand(tmpDir, "config", "user.email", "test@example.com")
	runGitCommand(tmpDir, "config", "user.name", "Test User")

	// Create test files
	file1 := filepath.Join(tmpDir, "job_file1.txt")
	if err := os.WriteFile(file1, []byte("job content 1"), 0644); err != nil {
		t.Fatalf("Failed to create file: %v", err)
	}

	// Add file to staging
	if err := gm.AddFiles([]string{file1}); err != nil {
		t.Fatalf("Failed to add file: %v", err)
	}

	// Commit job
	ac := NewAutoCommitter(gm)
	if err := ac.CommitJob("job_1", "Job 1: Initial setup"); err != nil {
		t.Fatalf("CommitJob failed: %v", err)
	}

	// Verify commit
	log, err := gm.GetLog(1)
	if err != nil {
		t.Fatalf("Failed to get log: %v", err)
	}

	if len(log) == 0 {
		t.Fatal("No commits found after job commit")
	}

	if !strings.Contains(log[0].Message, "job_1") {
		t.Errorf("Job commit message incorrect: %v", log[0].Message)
	}
}

// TestGitIntegrationVersionCheckout tests version checkout
func TestGitIntegrationVersionCheckout(t *testing.T) {
	tmpDir := t.TempDir()
	gm := New(tmpDir)

	if err := gm.InitRepo(); err != nil {
		t.Fatalf("Failed to initialize repo: %v", err)
	}

	// Create and commit file
	file1 := filepath.Join(tmpDir, "version_test.txt")
	if err := os.WriteFile(file1, []byte("v1 content"), 0644); err != nil {
		t.Fatalf("Failed to create file: %v", err)
	}

	if err := gm.AddFiles([]string{file1}); err != nil {
		t.Fatalf("Failed to add file: %v", err)
	}

	if err := gm.Commit("feat: v1 content"); err != nil {
		t.Fatalf("Failed to commit: %v", err)
	}

	// Create v1.0.0 tag
	vm := NewVersionManager(gm)
	if err := vm.CreateTag("v1.0.0", "Version 1.0.0"); err != nil {
		t.Fatalf("Failed to create tag: %v", err)
	}

	// Modify file and commit
	if err := os.WriteFile(file1, []byte("v2 content"), 0644); err != nil {
		t.Fatalf("Failed to update file: %v", err)
	}

	if err := gm.AddFiles([]string{file1}); err != nil {
		t.Fatalf("Failed to add file: %v", err)
	}

	if err := gm.Commit("feat: v2 content"); err != nil {
		t.Fatalf("Failed to commit: %v", err)
	}

	// Create v2.0.0 tag
	if err := vm.CreateTag("v2.0.0", "Version 2.0.0"); err != nil {
		t.Fatalf("Failed to create tag: %v", err)
	}

	// Checkout v1.0.0
	if err := vm.Checkout("v1.0.0"); err != nil {
		t.Fatalf("Failed to checkout: %v", err)
	}

	// Verify file content
	content, err := os.ReadFile(file1)
	if err != nil {
		t.Fatalf("Failed to read file: %v", err)
	}

	if string(content) != "v1 content" {
		t.Errorf("Expected 'v1 content', got '%s'", string(content))
	}

	// Checkout v2.0.0
	if err := vm.Checkout("v2.0.0"); err != nil {
		t.Fatalf("Failed to checkout: %v", err)
	}

	// Verify file content
	content, err = os.ReadFile(file1)
	if err != nil {
		t.Fatalf("Failed to read file: %v", err)
	}

	if string(content) != "v2 content" {
		t.Errorf("Expected 'v2 content', got '%s'", string(content))
	}
}

// TestGitIntegrationDiffAndHistory tests diff and file history
func TestGitIntegrationDiffAndHistory(t *testing.T) {
	tmpDir := t.TempDir()
	gm := New(tmpDir)

	if err := gm.InitRepo(); err != nil {
		t.Fatalf("Failed to initialize repo: %v", err)
	}

	// Create initial file
	file1 := filepath.Join(tmpDir, "diff_test.txt")
	if err := os.WriteFile(file1, []byte("line 1\nline 2\n"), 0644); err != nil {
		t.Fatalf("Failed to create file: %v", err)
	}

	if err := gm.AddFiles([]string{file1}); err != nil {
		t.Fatalf("Failed to add file: %v", err)
	}

	if err := gm.Commit("feat: initial content"); err != nil {
		t.Fatalf("Failed to commit: %v", err)
	}

	log1, err := gm.GetLog(1)
	if err != nil {
		t.Fatalf("Failed to get log: %v", err)
	}

	commit1 := log1[0].Hash

	// Modify file
	if err := os.WriteFile(file1, []byte("line 1\nline 2\nline 3\n"), 0644); err != nil {
		t.Fatalf("Failed to update file: %v", err)
	}

	if err := gm.AddFiles([]string{file1}); err != nil {
		t.Fatalf("Failed to add file: %v", err)
	}

	if err := gm.Commit("feat: add line 3"); err != nil {
		t.Fatalf("Failed to commit: %v", err)
	}

	log2, err := gm.GetLog(1)
	if err != nil {
		t.Fatalf("Failed to get log: %v", err)
	}

	commit2 := log2[0].Hash

	// Get diff
	rm := NewRollbackManager(gm)
	diffInfo, err := rm.GetDiff(commit1, commit2)
	if err != nil {
		t.Fatalf("Failed to get diff: %v", err)
	}

	if len(diffInfo) == 0 {
		t.Error("Diff should not be empty")
	}

	// Get file history
	history, err := rm.GetFileHistory(file1)
	if err != nil {
		t.Fatalf("Failed to get file history: %v", err)
	}

	if len(history) == 0 {
		t.Error("File history should not be empty")
	}

	// Verify history contains both commits
	hasCommit1 := false
	hasCommit2 := false

	for _, entry := range history {
		if strings.Contains(entry.Hash, commit1) || strings.Contains(entry.Message, "initial content") {
			hasCommit1 = true
		}
		if strings.Contains(entry.Hash, commit2) || strings.Contains(entry.Message, "add line 3") {
			hasCommit2 = true
		}
	}

	if !hasCommit1 {
		t.Error("First commit not found in file history")
	}
	if !hasCommit2 {
		t.Error("Second commit not found in file history")
	}
}

// TestGitIntegrationConcurrentOperations tests that operations work correctly in sequence
func TestGitIntegrationConcurrentOperations(t *testing.T) {
	tmpDir := t.TempDir()
	gm := New(tmpDir)

	if err := gm.InitRepo(); err != nil {
		t.Fatalf("Failed to initialize repo: %v", err)
	}

	// Simulate multiple task commits
	for i := 1; i <= 3; i++ {
		file := filepath.Join(tmpDir, "task_file_"+string(rune('0'+i))+".txt")
		if err := os.WriteFile(file, []byte("task "+string(rune('0'+i))+" content"), 0644); err != nil {
			t.Fatalf("Failed to create file: %v", err)
		}

		if err := gm.AddFiles([]string{file}); err != nil {
			t.Fatalf("Failed to add file: %v", err)
		}

		if err := gm.Commit("feat: task " + string(rune('0'+i))); err != nil {
			t.Fatalf("Failed to commit: %v", err)
		}
	}

	// Verify all commits exist
	log, err := gm.GetLog(10)
	if err != nil {
		t.Fatalf("Failed to get log: %v", err)
	}

	if len(log) < 3 {
		t.Errorf("Expected at least 3 commits, got %d", len(log))
	}

	// Verify commit order (most recent first)
	if !strings.Contains(log[0].Message, "task 3") {
		t.Errorf("Expected most recent commit to be 'task 3', got '%s'", log[0].Message)
	}
}

// TestGitIntegrationErrorHandling tests error conditions
func TestGitIntegrationErrorHandling(t *testing.T) {
	tmpDir := t.TempDir()

	// Test operations on non-git directory
	t.Run("NonGitDirectory", func(t *testing.T) {
		gm := New(tmpDir)

		// GetLog should fail on non-git directory
		_, err := gm.GetLog(1)
		if err == nil {
			t.Error("GetLog should fail on non-git directory")
		}

		// GetCurrentBranch should fail on non-git directory
		branch, err := gm.GetCurrentBranch()
		if err == nil || branch != "" {
			t.Error("GetCurrentBranch should fail on non-git directory")
		}
	})

	// Test invalid version format
	t.Run("InvalidVersionFormat", func(t *testing.T) {
		gm := New(tmpDir)
		if err := gm.InitRepo(); err != nil {
			t.Fatalf("Failed to initialize repo: %v", err)
		}

		// Create a commit first
		file := filepath.Join(tmpDir, "test.txt")
		if err := os.WriteFile(file, []byte("test"), 0644); err != nil {
			t.Fatalf("Failed to create file: %v", err)
		}

		if err := gm.AddFiles([]string{file}); err != nil {
			t.Fatalf("Failed to add file: %v", err)
		}

		if err := gm.Commit("feat: test"); err != nil {
			t.Fatalf("Failed to commit: %v", err)
		}

		// Try to create tag with invalid version
		vm := NewVersionManager(gm)
		err := vm.CreateTag("invalid_version", "Invalid")
		if err == nil {
			t.Error("CreateTag should fail with invalid version format")
		}
	})
}
