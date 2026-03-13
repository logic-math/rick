package git

import (
	"fmt"
	"os/exec"
	"strings"
)

// CommitMessage represents a formatted commit message
type CommitMessage struct {
	Type        string // feat, fix, refactor, test, docs, etc.
	Scope       string // task_id, job_id, etc.
	Description string // commit description
}

// AutoCommitter handles automatic commits for tasks and jobs
type AutoCommitter struct {
	gm *GitManager
}

// NewAutoCommitter creates a new AutoCommitter instance
func NewAutoCommitter(gm *GitManager) *AutoCommitter {
	return &AutoCommitter{
		gm: gm,
	}
}

// CommitTask commits a completed task with standardized message format
// Message format: feat(task_id): task_name
func (ac *AutoCommitter) CommitTask(taskID, taskName string) error {
	if taskID == "" || taskName == "" {
		return fmt.Errorf("task ID and name cannot be empty")
	}

	message := fmt.Sprintf("feat(%s): %s", taskID, taskName)
	return ac.gm.Commit(message)
}

// CommitJob commits an entire job with standardized message format
// Message format: feat(job_id): job_name
func (ac *AutoCommitter) CommitJob(jobID, jobName string) error {
	if jobID == "" || jobName == "" {
		return fmt.Errorf("job ID and name cannot be empty")
	}

	message := fmt.Sprintf("feat(%s): %s", jobID, jobName)
	return ac.gm.Commit(message)
}

// CommitDebug commits debug/problem records with standardized message format
// Message format: debug(job_id): debug_description
func (ac *AutoCommitter) CommitDebug(jobID, debugInfo string) error {
	if jobID == "" || debugInfo == "" {
		return fmt.Errorf("job ID and debug info cannot be empty")
	}

	message := fmt.Sprintf("debug(%s): %s", jobID, debugInfo)
	return ac.gm.Commit(message)
}

// GetModifiedFiles returns a list of modified files in the working directory
func (ac *AutoCommitter) GetModifiedFiles() ([]string, error) {
	cmd := exec.Command("git", "status", "--porcelain")
	cmd.Dir = ac.gm.repoPath
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get modified files: %w", err)
	}

	var files []string
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	for _, line := range lines {
		if line == "" {
			continue
		}
		// Format: "XY filename"
		// X = staged status, Y = unstaged status
		parts := strings.Fields(line)
		if len(parts) >= 2 {
			// Join all parts after the status to handle filenames with spaces
			filename := strings.Join(parts[1:], " ")
			files = append(files, filename)
		}
	}

	return files, nil
}

// GetStagedFiles returns a list of staged files ready for commit
func (ac *AutoCommitter) GetStagedFiles() ([]string, error) {
	cmd := exec.Command("git", "diff", "--cached", "--name-only")
	cmd.Dir = ac.gm.repoPath
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get staged files: %w", err)
	}

	var files []string
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	for _, line := range lines {
		if line != "" {
			files = append(files, line)
		}
	}

	return files, nil
}

// GetUnstagedFiles returns a list of unstaged files
func (ac *AutoCommitter) GetUnstagedFiles() ([]string, error) {
	cmd := exec.Command("git", "diff", "--name-only")
	cmd.Dir = ac.gm.repoPath
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get unstaged files: %w", err)
	}

	var files []string
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	for _, line := range lines {
		if line != "" {
			files = append(files, line)
		}
	}

	return files, nil
}

// GetUntrackedFiles returns a list of untracked files
func (ac *AutoCommitter) GetUntrackedFiles() ([]string, error) {
	cmd := exec.Command("git", "ls-files", "--others", "--exclude-standard")
	cmd.Dir = ac.gm.repoPath
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get untracked files: %w", err)
	}

	var files []string
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	for _, line := range lines {
		if line != "" {
			files = append(files, line)
		}
	}

	return files, nil
}

// AutoAddAndCommitTask automatically adds modified files and commits a task
func (ac *AutoCommitter) AutoAddAndCommitTask(taskID, taskName string) error {
	// Get all modified files (including untracked)
	modified, err := ac.GetModifiedFiles()
	if err != nil {
		return fmt.Errorf("failed to get modified files: %w", err)
	}

	untracked, err := ac.GetUntrackedFiles()
	if err != nil {
		return fmt.Errorf("failed to get untracked files: %w", err)
	}

	// Combine all files that need to be added
	allFiles := append(modified, untracked...)

	if len(allFiles) > 0 {
		// Add all files to staging area
		if err := ac.gm.AddFiles(allFiles); err != nil {
			return fmt.Errorf("failed to add files: %w", err)
		}
	}

	// Commit the task
	return ac.CommitTask(taskID, taskName)
}

// AutoAddAndCommitJob automatically adds modified files and commits a job
func (ac *AutoCommitter) AutoAddAndCommitJob(jobID, jobName string) error {
	// Get all modified files
	modified, err := ac.GetModifiedFiles()
	if err != nil {
		return fmt.Errorf("failed to get modified files: %w", err)
	}

	untracked, err := ac.GetUntrackedFiles()
	if err != nil {
		return fmt.Errorf("failed to get untracked files: %w", err)
	}

	// Combine all files that need to be added
	allFiles := append(modified, untracked...)

	if len(allFiles) > 0 {
		// Add all files to staging area
		if err := ac.gm.AddFiles(allFiles); err != nil {
			return fmt.Errorf("failed to add files: %w", err)
		}
	}

	// Commit the job
	return ac.CommitJob(jobID, jobName)
}

// AutoAddAndCommitDebug automatically adds modified files and commits debug info
func (ac *AutoCommitter) AutoAddAndCommitDebug(jobID, debugInfo string) error {
	// Get all modified files
	modified, err := ac.GetModifiedFiles()
	if err != nil {
		return fmt.Errorf("failed to get modified files: %w", err)
	}

	untracked, err := ac.GetUntrackedFiles()
	if err != nil {
		return fmt.Errorf("failed to get untracked files: %w", err)
	}

	// Combine all files that need to be added
	allFiles := append(modified, untracked...)

	if len(allFiles) > 0 {
		// Add all files to staging area
		if err := ac.gm.AddFiles(allFiles); err != nil {
			return fmt.Errorf("failed to add files: %w", err)
		}
	}

	// Commit the debug info
	return ac.CommitDebug(jobID, debugInfo)
}

// CommitTaskWithFiles commits specific files for a task
func (ac *AutoCommitter) CommitTaskWithFiles(taskID, taskName string, files []string) error {
	if taskID == "" || taskName == "" {
		return fmt.Errorf("task ID and name cannot be empty")
	}

	if len(files) == 0 {
		return fmt.Errorf("no files to commit")
	}

	// Add specified files
	if err := ac.gm.AddFiles(files); err != nil {
		return fmt.Errorf("failed to add files: %w", err)
	}

	// Commit the task
	return ac.CommitTask(taskID, taskName)
}

// HasChanges checks if there are any changes in the working directory
func (ac *AutoCommitter) HasChanges() (bool, error) {
	cmd := exec.Command("git", "status", "--porcelain")
	cmd.Dir = ac.gm.repoPath
	output, err := cmd.Output()
	if err != nil {
		return false, fmt.Errorf("failed to check status: %w", err)
	}

	return len(strings.TrimSpace(string(output))) > 0, nil
}

// GetWorkingDirectoryStatus returns the current working directory status
// Returns a map with keys: staged, unstaged, untracked
func (ac *AutoCommitter) GetWorkingDirectoryStatus() (map[string][]string, error) {
	status := make(map[string][]string)

	// Get staged files
	staged, err := ac.GetStagedFiles()
	if err != nil {
		return nil, err
	}
	status["staged"] = staged

	// Get unstaged files
	unstaged, err := ac.GetUnstagedFiles()
	if err != nil {
		return nil, err
	}
	status["unstaged"] = unstaged

	// Get untracked files
	untracked, err := ac.GetUntrackedFiles()
	if err != nil {
		return nil, err
	}
	status["untracked"] = untracked

	return status, nil
}
