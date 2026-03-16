package git

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// CommitInfo represents information about a Git commit
type CommitInfo struct {
	Hash    string
	Message string
	Author  string
	Date    time.Time
}

// GitManager manages Git operations for a repository
type GitManager struct {
	repoPath string
}

// New creates a new GitManager instance
func New(repoPath string) *GitManager {
	return &GitManager{
		repoPath: repoPath,
	}
}

// InitRepo initializes a Git repository at the given path
func (gm *GitManager) InitRepo() error {
	if err := os.MkdirAll(gm.repoPath, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	cmd := exec.Command("git", "init")
	cmd.Dir = gm.repoPath
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to initialize git repo: %w", err)
	}

	return nil
}

// AddFiles adds files to the staging area
func (gm *GitManager) AddFiles(paths []string) error {
	if len(paths) == 0 {
		// Not an error - just no files to add
		return nil
	}

	args := append([]string{"add"}, paths...)
	cmd := exec.Command("git", args...)
	cmd.Dir = gm.repoPath
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to add files: %w (output: %s)", err, string(output))
	}

	return nil
}

// Commit commits changes with the given message
func (gm *GitManager) Commit(message string) error {
	if message == "" {
		return fmt.Errorf("commit message cannot be empty")
	}

	cmd := exec.Command("git", "commit", "-m", message)
	cmd.Dir = gm.repoPath
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to commit: %w", err)
	}

	return nil
}

// GetLog retrieves commit history with the given limit
func (gm *GitManager) GetLog(limit int) ([]CommitInfo, error) {
	if limit <= 0 {
		limit = 10
	}

	format := "%H%n%s%n%an%n%ai"
	cmd := exec.Command("git", "log", fmt.Sprintf("--max-count=%d", limit), fmt.Sprintf("--format=%s", format))
	cmd.Dir = gm.repoPath
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get log: %w", err)
	}

	commits := []CommitInfo{}
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")

	for i := 0; i < len(lines); i += 4 {
		if i+3 >= len(lines) {
			break
		}

		dateStr := lines[i+3]
		date, err := time.Parse("2006-01-02 15:04:05 -0700", dateStr)
		if err != nil {
			date = time.Now()
		}

		commits = append(commits, CommitInfo{
			Hash:    lines[i],
			Message: lines[i+1],
			Author:  lines[i+2],
			Date:    date,
		})
	}

	return commits, nil
}

// GetCurrentBranch returns the name of the current branch
func (gm *GitManager) GetCurrentBranch() (string, error) {
	cmd := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD")
	cmd.Dir = gm.repoPath
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get current branch: %w", err)
	}

	branch := strings.TrimSpace(string(output))
	return branch, nil
}

// IsRepository checks if the given path is a Git repository
func (gm *GitManager) IsRepository() bool {
	gitDir := filepath.Join(gm.repoPath, ".git")
	_, err := os.Stat(gitDir)
	return err == nil
}

// GetRepoPath returns the repository path
func (gm *GitManager) GetRepoPath() string {
	return gm.repoPath
}

// GetDiff returns the diff for a specific commit
func (gm *GitManager) GetDiff(commitHash string) (string, error) {
	if commitHash == "" {
		return "", fmt.Errorf("commit hash cannot be empty")
	}

	cmd := exec.Command("git", "show", commitHash)
	cmd.Dir = gm.repoPath
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get diff: %w", err)
	}

	return string(output), nil
}

// GetCommitsBetween returns commits between two references
func (gm *GitManager) GetCommitsBetween(from, to string) ([]CommitInfo, error) {
	if from == "" || to == "" {
		return nil, fmt.Errorf("from and to references cannot be empty")
	}

	format := "%H%n%s%n%an%n%ai"
	revRange := fmt.Sprintf("%s..%s", from, to)
	cmd := exec.Command("git", "log", revRange, fmt.Sprintf("--format=%s", format))
	cmd.Dir = gm.repoPath
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get commits: %w", err)
	}

	commits := []CommitInfo{}
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")

	for i := 0; i < len(lines); i += 4 {
		if i+3 >= len(lines) {
			break
		}

		dateStr := lines[i+3]
		date, err := time.Parse("2006-01-02 15:04:05 -0700", dateStr)
		if err != nil {
			date = time.Now()
		}

		commits = append(commits, CommitInfo{
			Hash:    lines[i],
			Message: lines[i+1],
			Author:  lines[i+2],
			Date:    date,
		})
	}

	return commits, nil
}

// GetCommitsByGrep returns commits matching a pattern in the message
func (gm *GitManager) GetCommitsByGrep(pattern string, limit int) ([]CommitInfo, error) {
	if pattern == "" {
		return nil, fmt.Errorf("pattern cannot be empty")
	}

	if limit <= 0 {
		limit = 100
	}

	format := "%H%n%s%n%an%n%ai"
	cmd := exec.Command("git", "log", "--grep", pattern, fmt.Sprintf("--max-count=%d", limit), fmt.Sprintf("--format=%s", format))
	cmd.Dir = gm.repoPath
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get commits: %w", err)
	}

	commits := []CommitInfo{}
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")

	for i := 0; i < len(lines); i += 4 {
		if i+3 >= len(lines) {
			break
		}

		dateStr := lines[i+3]
		date, err := time.Parse("2006-01-02 15:04:05 -0700", dateStr)
		if err != nil {
			date = time.Now()
		}

		commits = append(commits, CommitInfo{
			Hash:    lines[i],
			Message: lines[i+1],
			Author:  lines[i+2],
			Date:    date,
		})
	}

	return commits, nil
}
