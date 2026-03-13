package git

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// TestCommitTask tests the CommitTask function
func TestCommitTask(t *testing.T) {
	// Create a temporary directory for testing
	tmpDir, err := os.MkdirTemp("", "rick-test-")
	if err != nil {
		t.Fatalf("failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Initialize a Git repository
	gm := New(tmpDir)
	if err := gm.InitRepo(); err != nil {
		t.Fatalf("failed to initialize repo: %v", err)
	}

	// Configure git user for testing
	configureGitUser(t, tmpDir)

	// Create a test file
	testFile := filepath.Join(tmpDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("test content"), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	// Add and commit the file
	if err := gm.AddFiles([]string{"test.txt"}); err != nil {
		t.Fatalf("failed to add file: %v", err)
	}

	ac := NewAutoCommitter(gm)

	// Test CommitTask
	taskID := "task_1"
	taskName := "Implement basic feature"
	if err := ac.CommitTask(taskID, taskName); err != nil {
		t.Fatalf("CommitTask failed: %v", err)
	}

	// Verify the commit message
	logs, err := gm.GetLog(1)
	if err != nil {
		t.Fatalf("failed to get log: %v", err)
	}

	if len(logs) == 0 {
		t.Fatal("no commits found")
	}

	expectedMessage := fmt.Sprintf("feat(%s): %s", taskID, taskName)
	if logs[0].Message != expectedMessage {
		t.Errorf("expected message '%s', got '%s'", expectedMessage, logs[0].Message)
	}
}

// TestCommitTask_EmptyParams tests CommitTask with empty parameters
func TestCommitTask_EmptyParams(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "rick-test-")
	if err != nil {
		t.Fatalf("failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	gm := New(tmpDir)
	if err := gm.InitRepo(); err != nil {
		t.Fatalf("failed to initialize repo: %v", err)
	}

	ac := NewAutoCommitter(gm)

	// Test with empty taskID
	if err := ac.CommitTask("", "task name"); err == nil {
		t.Error("expected error for empty taskID, got nil")
	}

	// Test with empty taskName
	if err := ac.CommitTask("task_1", ""); err == nil {
		t.Error("expected error for empty taskName, got nil")
	}
}

// TestCommitJob tests the CommitJob function
func TestCommitJob(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "rick-test-")
	if err != nil {
		t.Fatalf("failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	gm := New(tmpDir)
	if err := gm.InitRepo(); err != nil {
		t.Fatalf("failed to initialize repo: %v", err)
	}

	configureGitUser(t, tmpDir)

	// Create a test file
	testFile := filepath.Join(tmpDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("test content"), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	if err := gm.AddFiles([]string{"test.txt"}); err != nil {
		t.Fatalf("failed to add file: %v", err)
	}

	ac := NewAutoCommitter(gm)

	// Test CommitJob
	jobID := "job_1"
	jobName := "Implement Git integration"
	if err := ac.CommitJob(jobID, jobName); err != nil {
		t.Fatalf("CommitJob failed: %v", err)
	}

	// Verify the commit message
	logs, err := gm.GetLog(1)
	if err != nil {
		t.Fatalf("failed to get log: %v", err)
	}

	expectedMessage := fmt.Sprintf("feat(%s): %s", jobID, jobName)
	if logs[0].Message != expectedMessage {
		t.Errorf("expected message '%s', got '%s'", expectedMessage, logs[0].Message)
	}
}

// TestCommitDebug tests the CommitDebug function
func TestCommitDebug(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "rick-test-")
	if err != nil {
		t.Fatalf("failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	gm := New(tmpDir)
	if err := gm.InitRepo(); err != nil {
		t.Fatalf("failed to initialize repo: %v", err)
	}

	configureGitUser(t, tmpDir)

	// Create a test file
	testFile := filepath.Join(tmpDir, "debug.txt")
	if err := os.WriteFile(testFile, []byte("debug info"), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	if err := gm.AddFiles([]string{"debug.txt"}); err != nil {
		t.Fatalf("failed to add file: %v", err)
	}

	ac := NewAutoCommitter(gm)

	// Test CommitDebug
	jobID := "job_1"
	debugInfo := "Fixed issue with file detection"
	if err := ac.CommitDebug(jobID, debugInfo); err != nil {
		t.Fatalf("CommitDebug failed: %v", err)
	}

	// Verify the commit message
	logs, err := gm.GetLog(1)
	if err != nil {
		t.Fatalf("failed to get log: %v", err)
	}

	expectedMessage := fmt.Sprintf("debug(%s): %s", jobID, debugInfo)
	if logs[0].Message != expectedMessage {
		t.Errorf("expected message '%s', got '%s'", expectedMessage, logs[0].Message)
	}
}

// TestGetModifiedFiles tests the GetModifiedFiles function
func TestGetModifiedFiles(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "rick-test-")
	if err != nil {
		t.Fatalf("failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	gm := New(tmpDir)
	if err := gm.InitRepo(); err != nil {
		t.Fatalf("failed to initialize repo: %v", err)
	}

	ac := NewAutoCommitter(gm)

	// Create a test file
	testFile := filepath.Join(tmpDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("test content"), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	// Get modified files
	files, err := ac.GetModifiedFiles()
	if err != nil {
		t.Fatalf("GetModifiedFiles failed: %v", err)
	}

	// Should find the test file
	found := false
	for _, f := range files {
		if f == "test.txt" {
			found = true
			break
		}
	}

	if !found {
		t.Errorf("expected to find test.txt in modified files, got %v", files)
	}
}

// TestGetStagedFiles tests the GetStagedFiles function
func TestGetStagedFiles(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "rick-test-")
	if err != nil {
		t.Fatalf("failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	gm := New(tmpDir)
	if err := gm.InitRepo(); err != nil {
		t.Fatalf("failed to initialize repo: %v", err)
	}

	// Create and stage a test file
	testFile := filepath.Join(tmpDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("test content"), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	if err := gm.AddFiles([]string{"test.txt"}); err != nil {
		t.Fatalf("failed to add file: %v", err)
	}

	ac := NewAutoCommitter(gm)

	// Get staged files
	files, err := ac.GetStagedFiles()
	if err != nil {
		t.Fatalf("GetStagedFiles failed: %v", err)
	}

	// Should find the test file
	found := false
	for _, f := range files {
		if f == "test.txt" {
			found = true
			break
		}
	}

	if !found {
		t.Errorf("expected to find test.txt in staged files, got %v", files)
	}
}

// TestGetUntrackedFiles tests the GetUntrackedFiles function
func TestGetUntrackedFiles(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "rick-test-")
	if err != nil {
		t.Fatalf("failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	gm := New(tmpDir)
	if err := gm.InitRepo(); err != nil {
		t.Fatalf("failed to initialize repo: %v", err)
	}

	// Create an untracked file
	testFile := filepath.Join(tmpDir, "untracked.txt")
	if err := os.WriteFile(testFile, []byte("untracked content"), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	ac := NewAutoCommitter(gm)

	// Get untracked files
	files, err := ac.GetUntrackedFiles()
	if err != nil {
		t.Fatalf("GetUntrackedFiles failed: %v", err)
	}

	// Should find the untracked file
	found := false
	for _, f := range files {
		if f == "untracked.txt" {
			found = true
			break
		}
	}

	if !found {
		t.Errorf("expected to find untracked.txt in untracked files, got %v", files)
	}
}

// TestAutoAddAndCommitTask tests the AutoAddAndCommitTask function
func TestAutoAddAndCommitTask(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "rick-test-")
	if err != nil {
		t.Fatalf("failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	gm := New(tmpDir)
	if err := gm.InitRepo(); err != nil {
		t.Fatalf("failed to initialize repo: %v", err)
	}

	configureGitUser(t, tmpDir)

	// Create test files
	file1 := filepath.Join(tmpDir, "file1.txt")
	file2 := filepath.Join(tmpDir, "file2.txt")
	if err := os.WriteFile(file1, []byte("content1"), 0644); err != nil {
		t.Fatalf("failed to create file1: %v", err)
	}
	if err := os.WriteFile(file2, []byte("content2"), 0644); err != nil {
		t.Fatalf("failed to create file2: %v", err)
	}

	ac := NewAutoCommitter(gm)

	// Test AutoAddAndCommitTask
	taskID := "task_1"
	taskName := "Add multiple files"
	if err := ac.AutoAddAndCommitTask(taskID, taskName); err != nil {
		t.Fatalf("AutoAddAndCommitTask failed: %v", err)
	}

	// Verify the commit
	logs, err := gm.GetLog(1)
	if err != nil {
		t.Fatalf("failed to get log: %v", err)
	}

	expectedMessage := fmt.Sprintf("feat(%s): %s", taskID, taskName)
	if logs[0].Message != expectedMessage {
		t.Errorf("expected message '%s', got '%s'", expectedMessage, logs[0].Message)
	}
}

// TestAutoAddAndCommitJob tests the AutoAddAndCommitJob function
func TestAutoAddAndCommitJob(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "rick-test-")
	if err != nil {
		t.Fatalf("failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	gm := New(tmpDir)
	if err := gm.InitRepo(); err != nil {
		t.Fatalf("failed to initialize repo: %v", err)
	}

	configureGitUser(t, tmpDir)

	// Create a test file
	testFile := filepath.Join(tmpDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("test content"), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	ac := NewAutoCommitter(gm)

	// Test AutoAddAndCommitJob
	jobID := "job_1"
	jobName := "Complete job"
	if err := ac.AutoAddAndCommitJob(jobID, jobName); err != nil {
		t.Fatalf("AutoAddAndCommitJob failed: %v", err)
	}

	// Verify the commit
	logs, err := gm.GetLog(1)
	if err != nil {
		t.Fatalf("failed to get log: %v", err)
	}

	expectedMessage := fmt.Sprintf("feat(%s): %s", jobID, jobName)
	if logs[0].Message != expectedMessage {
		t.Errorf("expected message '%s', got '%s'", expectedMessage, logs[0].Message)
	}
}

// TestAutoAddAndCommitDebug tests the AutoAddAndCommitDebug function
func TestAutoAddAndCommitDebug(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "rick-test-")
	if err != nil {
		t.Fatalf("failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	gm := New(tmpDir)
	if err := gm.InitRepo(); err != nil {
		t.Fatalf("failed to initialize repo: %v", err)
	}

	configureGitUser(t, tmpDir)

	// Create a test file
	testFile := filepath.Join(tmpDir, "debug.txt")
	if err := os.WriteFile(testFile, []byte("debug content"), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	ac := NewAutoCommitter(gm)

	// Test AutoAddAndCommitDebug
	jobID := "job_1"
	debugInfo := "Fixed critical bug"
	if err := ac.AutoAddAndCommitDebug(jobID, debugInfo); err != nil {
		t.Fatalf("AutoAddAndCommitDebug failed: %v", err)
	}

	// Verify the commit
	logs, err := gm.GetLog(1)
	if err != nil {
		t.Fatalf("failed to get log: %v", err)
	}

	expectedMessage := fmt.Sprintf("debug(%s): %s", jobID, debugInfo)
	if logs[0].Message != expectedMessage {
		t.Errorf("expected message '%s', got '%s'", expectedMessage, logs[0].Message)
	}
}

// TestCommitTaskWithFiles tests the CommitTaskWithFiles function
func TestCommitTaskWithFiles(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "rick-test-")
	if err != nil {
		t.Fatalf("failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	gm := New(tmpDir)
	if err := gm.InitRepo(); err != nil {
		t.Fatalf("failed to initialize repo: %v", err)
	}

	configureGitUser(t, tmpDir)

	// Create multiple test files
	file1 := filepath.Join(tmpDir, "file1.txt")
	file2 := filepath.Join(tmpDir, "file2.txt")
	if err := os.WriteFile(file1, []byte("content1"), 0644); err != nil {
		t.Fatalf("failed to create file1: %v", err)
	}
	if err := os.WriteFile(file2, []byte("content2"), 0644); err != nil {
		t.Fatalf("failed to create file2: %v", err)
	}

	ac := NewAutoCommitter(gm)

	// Test CommitTaskWithFiles with specific files
	taskID := "task_2"
	taskName := "Commit specific files"
	if err := ac.CommitTaskWithFiles(taskID, taskName, []string{"file1.txt"}); err != nil {
		t.Fatalf("CommitTaskWithFiles failed: %v", err)
	}

	// Verify the commit
	logs, err := gm.GetLog(1)
	if err != nil {
		t.Fatalf("failed to get log: %v", err)
	}

	expectedMessage := fmt.Sprintf("feat(%s): %s", taskID, taskName)
	if logs[0].Message != expectedMessage {
		t.Errorf("expected message '%s', got '%s'", expectedMessage, logs[0].Message)
	}
}

// TestCommitTaskWithFiles_EmptyFiles tests CommitTaskWithFiles with no files
func TestCommitTaskWithFiles_EmptyFiles(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "rick-test-")
	if err != nil {
		t.Fatalf("failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	gm := New(tmpDir)
	if err := gm.InitRepo(); err != nil {
		t.Fatalf("failed to initialize repo: %v", err)
	}

	ac := NewAutoCommitter(gm)

	// Test with empty files list
	if err := ac.CommitTaskWithFiles("task_1", "task name", []string{}); err == nil {
		t.Error("expected error for empty files list, got nil")
	}
}

// TestHasChanges tests the HasChanges function
func TestHasChanges(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "rick-test-")
	if err != nil {
		t.Fatalf("failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	gm := New(tmpDir)
	if err := gm.InitRepo(); err != nil {
		t.Fatalf("failed to initialize repo: %v", err)
	}

	ac := NewAutoCommitter(gm)

	// Initially, no changes
	hasChanges, err := ac.HasChanges()
	if err != nil {
		t.Fatalf("HasChanges failed: %v", err)
	}

	if hasChanges {
		t.Error("expected no changes initially")
	}

	// Create a file
	testFile := filepath.Join(tmpDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("test content"), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	// Now there should be changes
	hasChanges, err = ac.HasChanges()
	if err != nil {
		t.Fatalf("HasChanges failed: %v", err)
	}

	if !hasChanges {
		t.Error("expected changes after creating file")
	}
}

// TestGetWorkingDirectoryStatus tests the GetWorkingDirectoryStatus function
func TestGetWorkingDirectoryStatus(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "rick-test-")
	if err != nil {
		t.Fatalf("failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	gm := New(tmpDir)
	if err := gm.InitRepo(); err != nil {
		t.Fatalf("failed to initialize repo: %v", err)
	}

	configureGitUser(t, tmpDir)

	// Create and stage a file
	file1 := filepath.Join(tmpDir, "file1.txt")
	if err := os.WriteFile(file1, []byte("content1"), 0644); err != nil {
		t.Fatalf("failed to create file1: %v", err)
	}

	if err := gm.AddFiles([]string{"file1.txt"}); err != nil {
		t.Fatalf("failed to add file1: %v", err)
	}

	// Commit the file
	if err := gm.Commit("initial commit"); err != nil {
		t.Fatalf("failed to commit: %v", err)
	}

	// Modify the file (unstaged)
	if err := os.WriteFile(file1, []byte("modified content1"), 0644); err != nil {
		t.Fatalf("failed to modify file1: %v", err)
	}

	// Create a new untracked file
	file2 := filepath.Join(tmpDir, "file2.txt")
	if err := os.WriteFile(file2, []byte("content2"), 0644); err != nil {
		t.Fatalf("failed to create file2: %v", err)
	}

	ac := NewAutoCommitter(gm)

	// Get working directory status
	status, err := ac.GetWorkingDirectoryStatus()
	if err != nil {
		t.Fatalf("GetWorkingDirectoryStatus failed: %v", err)
	}

	// Check unstaged files
	if len(status["unstaged"]) == 0 {
		t.Error("expected unstaged files")
	}

	// Check untracked files
	if len(status["untracked"]) == 0 {
		t.Error("expected untracked files")
	}
}

// Helper function to configure git user for testing
func configureGitUser(t *testing.T, repoPath string) {
	// Configure git user name
	cmd1 := exec.Command("git", "config", "user.name", "Test User")
	cmd1.Dir = repoPath
	if err := cmd1.Run(); err != nil {
		t.Logf("warning: failed to configure git user name: %v", err)
	}

	// Configure git user email
	cmd2 := exec.Command("git", "config", "user.email", "test@example.com")
	cmd2.Dir = repoPath
	if err := cmd2.Run(); err != nil {
		t.Logf("warning: failed to configure git user email: %v", err)
	}
}
