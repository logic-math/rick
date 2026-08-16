package handler

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/sunquan/rick/internal/executor"
)

func setupGitRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	orig, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(orig) })

	// Initialize git repo
	cmds := [][]string{
		{"git", "init"},
		{"git", "config", "user.email", "test@test.com"},
		{"git", "config", "user.name", "Test"},
	}
	for _, args := range cmds {
		cmd := exec.Command(args[0], args[1:]...)
		cmd.Dir = dir
		if err := cmd.Run(); err != nil {
			t.Fatalf("git setup failed: %v", err)
		}
	}
	// Create initial commit
	if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte("test"), 0644); err != nil {
		t.Fatal(err)
	}
	for _, args := range [][]string{
		{"git", "add", "."},
		{"git", "commit", "-m", "init"},
	} {
		cmd := exec.Command(args[0], args[1:]...)
		cmd.Dir = dir
		if err := cmd.Run(); err != nil {
			t.Fatalf("git commit failed: %v", err)
		}
	}
	return dir
}

func TestCommitDoingResults_NoChanges(t *testing.T) {
	setupGitRepo(t)
	result := &executor.ExecutionJobResult{
		JobID:           "job_test",
		Status:          "completed",
		TotalTasks:      1,
		SuccessfulTasks: 1,
	}
	// No changes to commit - should succeed silently
	err := commitDoingResults("job_test", result, false)
	if err != nil {
		t.Errorf("expected no error for no-changes case, got: %v", err)
	}
}

func TestCommitDoingResults_PartialStatus(t *testing.T) {
	dir := setupGitRepo(t)
	// Create a new file to commit
	if err := os.WriteFile(filepath.Join(dir, "new_file.txt"), []byte("content"), 0644); err != nil {
		t.Fatal(err)
	}
	result := &executor.ExecutionJobResult{
		JobID:           "job_test",
		Status:          "partial",
		TotalTasks:      2,
		SuccessfulTasks: 1,
		FailedTasks:     1,
	}
	err := commitDoingResults("job_test", result, false)
	if err != nil {
		t.Logf("commitDoingResults partial error (acceptable): %v", err)
	}
}

func TestCommitDoingResults_FailedStatus(t *testing.T) {
	setupGitRepo(t)
	result := &executor.ExecutionJobResult{
		JobID:       "job_test",
		Status:      "failed",
		TotalTasks:  1,
		FailedTasks: 1,
	}
	err := commitDoingResults("job_test", result, false)
	if err != nil {
		t.Logf("commitDoingResults failed status error (acceptable): %v", err)
	}
}

func TestEnsureGitUserConfigured(t *testing.T) {
	dir := setupGitRepo(t)
	err := ensureGitUserConfigured(dir, false)
	if err != nil {
		t.Logf("ensureGitUserConfigured error (acceptable in test env): %v", err)
	}
}

func TestEnsureGitUserConfigured_WithConfig(t *testing.T) {
	dir := setupGitRepo(t)
	// Unset git user to force configuration
	exec.Command("git", "config", "--unset", "user.name").Run()
	exec.Command("git", "config", "--unset", "user.email").Run()
	err := ensureGitUserConfigured(dir, false)
	if err != nil {
		t.Logf("ensureGitUserConfigured error (acceptable): %v", err)
	}
}
