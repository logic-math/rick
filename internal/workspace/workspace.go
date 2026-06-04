package workspace

import (
	"fmt"
	"os"
	"path/filepath"
)

// Workspace represents the Rick workspace manager
type Workspace struct {
	rickDir string
}

// New creates a new Workspace instance and ensures the workspace exists
func New() (*Workspace, error) {
	rickDir, err := GetRickDir()
	if err != nil {
		return nil, err
	}

	ws := &Workspace{
		rickDir: rickDir,
	}

	// Automatically ensure workspace directories exist
	if err := ws.EnsureDirectories(); err != nil {
		return nil, fmt.Errorf("failed to ensure workspace directories: %w", err)
	}

	return ws, nil
}

// InitWorkspace initializes the .rick directory structure.
// Delegates to EnsureDirectories which handles both dirs and stub files.
func (w *Workspace) InitWorkspace() error {
	return w.EnsureDirectories()
}

// GetJobPath returns the path to a specific job directory
func (w *Workspace) GetJobPath(jobID string) (string, error) {
	if jobID == "" {
		return "", fmt.Errorf("jobID cannot be empty")
	}
	return filepath.Join(w.rickDir, JobsDirName, jobID), nil
}

// CreateJobStructure creates the directory structure for a job
func (w *Workspace) CreateJobStructure(jobID string) error {
	if jobID == "" {
		return fmt.Errorf("jobID cannot be empty")
	}

	jobPath, err := w.GetJobPath(jobID)
	if err != nil {
		return err
	}

	// Create job subdirectories
	directories := []string{
		jobPath,
		filepath.Join(jobPath, PlanDirName),
		filepath.Join(jobPath, DoingDirName),
		filepath.Join(jobPath, LearningDirName),
	}

	for _, dir := range directories {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}

	return nil
}

// EnsureDirectories ensures all necessary workspace directories and stub files exist.
// Idempotent: safe to call on every plan/doing invocation.
func (w *Workspace) EnsureDirectories() error {
	dirs := []string{
		w.rickDir,
		filepath.Join(w.rickDir, WikiDirName),
		filepath.Join(w.rickDir, JobsDirName),
		filepath.Join(w.rickDir, DreamDirName),
	}
	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to ensure directory %s: %w", dir, err)
		}
	}

	// Create stub files only if they don't exist yet
	stubs := map[string]string{
		filepath.Join(w.rickDir, OKRFileName):  "# OKR\n\n",
		filepath.Join(w.rickDir, SpecFileName): "# SPEC\n\n## 调试环境\n\n## 架构设计\n\n## 编译与运行方法\n\n## 观测方法\n\n## 控制方法\n\n## 技能列表\n\n| 名称 | 触发词 | 路径 |\n|------|--------|------|\n",
	}
	for path, content := range stubs {
		if _, err := os.Stat(path); os.IsNotExist(err) {
			if err := os.WriteFile(path, []byte(content), 0644); err != nil {
				return fmt.Errorf("failed to create %s: %w", path, err)
			}
		}
	}

	return nil
}

// GetRickDir returns the .rick directory path
func (w *Workspace) GetRickDir() string {
	return w.rickDir
}
