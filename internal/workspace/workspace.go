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

// InitWorkspace initializes the .rick directory structure
func (w *Workspace) InitWorkspace() error {
	// Create main directories
	directories := []string{
		w.rickDir,
		filepath.Join(w.rickDir, WikiDirName),
		filepath.Join(w.rickDir, SkillsDirName),
		filepath.Join(w.rickDir, JobsDirName),
	}

	for _, dir := range directories {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}

	// Create OKR.md file if it doesn't exist
	okriPath := filepath.Join(w.rickDir, OKRFileName)
	if _, err := os.Stat(okriPath); os.IsNotExist(err) {
		if err := os.WriteFile(okriPath, []byte("# OKR\n\n"), 0644); err != nil {
			return fmt.Errorf("failed to create OKR.md: %w", err)
		}
	}

	// Create SPEC.md file if it doesn't exist
	specPath := filepath.Join(w.rickDir, SpecFileName)
	if _, err := os.Stat(specPath); os.IsNotExist(err) {
		if err := os.WriteFile(specPath, []byte("# SPEC\n\n"), 0644); err != nil {
			return fmt.Errorf("failed to create SPEC.md: %w", err)
		}
	}

	return nil
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

// EnsureDirectories ensures all necessary workspace directories exist
func (w *Workspace) EnsureDirectories() error {
	directories := []string{
		w.rickDir,
		filepath.Join(w.rickDir, WikiDirName),
		filepath.Join(w.rickDir, SkillsDirName),
		filepath.Join(w.rickDir, JobsDirName),
	}

	for _, dir := range directories {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to ensure directory %s: %w", dir, err)
		}
	}

	return nil
}

// GetRickDir returns the .rick directory path
func (w *Workspace) GetRickDir() string {
	return w.rickDir
}
