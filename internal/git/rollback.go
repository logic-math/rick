package git

import (
	"fmt"
	"os/exec"
	"strings"
)

// RollbackManager handles rollback and recovery operations
type RollbackManager struct {
	gm *GitManager
}

// NewRollbackManager creates a new RollbackManager instance
func NewRollbackManager(gm *GitManager) *RollbackManager {
	return &RollbackManager{
		gm: gm,
	}
}

// ResetToCommit resets the repository to a specific commit hash
// This performs a hard reset, discarding all changes
func (rm *RollbackManager) ResetToCommit(hash string) error {
	if hash == "" {
		return fmt.Errorf("commit hash cannot be empty")
	}

	// Safety check: verify commit exists
	if err := rm.commitExists(hash); err != nil {
		return fmt.Errorf("commit %s does not exist: %w", hash, err)
	}

	// Check for uncommitted changes
	if err := rm.checkUncommittedChanges(); err != nil {
		return fmt.Errorf("cannot reset: %w", err)
	}

	// Perform hard reset
	cmd := exec.Command("git", "reset", "--hard", hash)
	cmd.Dir = rm.gm.repoPath
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to reset to commit %s: %w", hash, err)
	}

	return nil
}

// ResetToVersion resets the repository to a specific version tag
// This performs a hard reset, discarding all changes
func (rm *RollbackManager) ResetToVersion(version string) error {
	if version == "" {
		return fmt.Errorf("version cannot be empty")
	}

	// Safety check: verify version tag exists
	vm := NewVersionManager(rm.gm)
	exists, err := vm.TagExists(version)
	if err != nil {
		return fmt.Errorf("failed to check version: %w", err)
	}

	if !exists {
		return fmt.Errorf("version tag %s does not exist", version)
	}

	// Check for uncommitted changes
	if err := rm.checkUncommittedChanges(); err != nil {
		return fmt.Errorf("cannot reset: %w", err)
	}

	// Get commit hash for the version
	hash, err := vm.getTagHash(version)
	if err != nil {
		return fmt.Errorf("failed to get commit hash for version %s: %w", version, err)
	}

	// Perform hard reset to the version
	cmd := exec.Command("git", "reset", "--hard", hash)
	cmd.Dir = rm.gm.repoPath
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to reset to version %s: %w", version, err)
	}

	return nil
}

// DiffInfo represents the difference between two commits
type DiffInfo struct {
	FilePath string
	Status   string // "A" (added), "M" (modified), "D" (deleted)
	Changes  string // detailed diff content
}

// GetDiff returns the differences between two commits
// fromCommit and toCommit can be commit hashes or tags
func (rm *RollbackManager) GetDiff(fromCommit, toCommit string) ([]DiffInfo, error) {
	if fromCommit == "" || toCommit == "" {
		return nil, fmt.Errorf("both fromCommit and toCommit cannot be empty")
	}

	// Get list of changed files
	cmd := exec.Command("git", "diff", "--name-status", fromCommit, toCommit)
	cmd.Dir = rm.gm.repoPath
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get diff: %w", err)
	}

	var diffs []DiffInfo
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")

	for _, line := range lines {
		if line == "" {
			continue
		}

		parts := strings.Fields(line)
		if len(parts) < 2 {
			continue
		}

		status := parts[0]
		filePath := parts[1]

		// Get detailed diff for this file
		diffCmd := exec.Command("git", "diff", fromCommit, toCommit, "--", filePath)
		diffCmd.Dir = rm.gm.repoPath
		diffOutput, err := diffCmd.Output()
		if err != nil {
			// Continue on error for individual files
			continue
		}

		diffs = append(diffs, DiffInfo{
			FilePath: filePath,
			Status:   status,
			Changes:  string(diffOutput),
		})
	}

	return diffs, nil
}

// FileHistoryEntry represents a single commit in a file's history
type FileHistoryEntry struct {
	Hash    string
	Message string
	Author  string
	Date    string
}

// GetFileHistory returns the commit history for a specific file
func (rm *RollbackManager) GetFileHistory(filePath string) ([]FileHistoryEntry, error) {
	if filePath == "" {
		return nil, fmt.Errorf("file path cannot be empty")
	}

	// Get log for the file
	format := "%H%n%s%n%an%n%ai"
	cmd := exec.Command("git", "log", "--follow", fmt.Sprintf("--format=%s", format), "--", filePath)
	cmd.Dir = rm.gm.repoPath
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get file history: %w", err)
	}

	var history []FileHistoryEntry
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")

	for i := 0; i < len(lines); i += 4 {
		if i+3 >= len(lines) {
			break
		}

		history = append(history, FileHistoryEntry{
			Hash:    lines[i],
			Message: lines[i+1],
			Author:  lines[i+2],
			Date:    lines[i+3],
		})
	}

	return history, nil
}

// commitExists checks if a commit hash exists in the repository
func (rm *RollbackManager) commitExists(hash string) error {
	cmd := exec.Command("git", "cat-file", "-t", hash)
	cmd.Dir = rm.gm.repoPath
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("commit does not exist")
	}
	return nil
}

// checkUncommittedChanges checks if there are uncommitted changes
// Returns error if there are uncommitted changes
func (rm *RollbackManager) checkUncommittedChanges() error {
	cmd := exec.Command("git", "status", "--porcelain")
	cmd.Dir = rm.gm.repoPath
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("failed to check status: %w", err)
	}

	if strings.TrimSpace(string(output)) != "" {
		return fmt.Errorf("working directory has uncommitted changes; please commit or stash them first")
	}

	return nil
}

// HasUncommittedChanges returns true if there are uncommitted changes
func (rm *RollbackManager) HasUncommittedChanges() (bool, error) {
	cmd := exec.Command("git", "status", "--porcelain")
	cmd.Dir = rm.gm.repoPath
	output, err := cmd.Output()
	if err != nil {
		return false, fmt.Errorf("failed to check status: %w", err)
	}

	return strings.TrimSpace(string(output)) != "", nil
}

// GetCurrentCommitHash returns the current commit hash
func (rm *RollbackManager) GetCurrentCommitHash() (string, error) {
	cmd := exec.Command("git", "rev-parse", "HEAD")
	cmd.Dir = rm.gm.repoPath
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get current commit hash: %w", err)
	}

	return strings.TrimSpace(string(output)), nil
}

// ValidateCommit validates if a commit hash is valid (can be short or full hash)
func (rm *RollbackManager) ValidateCommit(hash string) (string, error) {
	if hash == "" {
		return "", fmt.Errorf("commit hash cannot be empty")
	}

	// Get the full hash from the short hash
	cmd := exec.Command("git", "rev-parse", hash)
	cmd.Dir = rm.gm.repoPath
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("invalid commit hash: %s", hash)
	}

	return strings.TrimSpace(string(output)), nil
}
