package workspace

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Path constants for the Rick workspace structure
const (
	RickDirName    = ".rick"
	RickDevDirName = ".rick_dev"
	OKRFileName    = "OKR.md"
	SpecFileName   = "SPEC.md"
	WikiDirName    = "wiki"
	SkillsDirName  = "skills"
	JobsDirName    = "jobs"
	PlanDirName    = "plan"
	DoingDirName   = "doing"
	LearningDirName = "learning"
)

// GetHomeDir returns the user's home directory
func GetHomeDir() (string, error) {
	return os.UserHomeDir()
}

// getRickDirName returns the appropriate rick directory name based on the binary name
func getRickDirName() string {
	// Get the binary name from os.Args[0]
	binaryPath := os.Args[0]
	binaryName := filepath.Base(binaryPath)

	// If the binary name ends with _dev, use .rick_dev
	if strings.HasSuffix(binaryName, "_dev") {
		return RickDevDirName
	}
	return RickDirName
}

// GetRickDir returns the path to the .rick directory in the current working directory
func GetRickDir() (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	rickDirName := getRickDirName()
	return filepath.Join(cwd, rickDirName), nil
}

// GetJobsDir returns the path to the jobs directory
func GetJobsDir() (string, error) {
	rickDir, err := GetRickDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(rickDir, JobsDirName), nil
}

// GetJobDir returns the path to a specific job directory
func GetJobDir(jobID string) (string, error) {
	jobsDir, err := GetJobsDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(jobsDir, jobID), nil
}

// GetJobPlanDir returns the path to the plan directory for a job
func GetJobPlanDir(jobID string) (string, error) {
	jobDir, err := GetJobDir(jobID)
	if err != nil {
		return "", err
	}
	return filepath.Join(jobDir, PlanDirName), nil
}

// GetJobDoingDir returns the path to the doing directory for a job
func GetJobDoingDir(jobID string) (string, error) {
	jobDir, err := GetJobDir(jobID)
	if err != nil {
		return "", err
	}
	return filepath.Join(jobDir, DoingDirName), nil
}

// GetJobLearningDir returns the path to the learning directory for a job
func GetJobLearningDir(jobID string) (string, error) {
	jobDir, err := GetJobDir(jobID)
	if err != nil {
		return "", err
	}
	return filepath.Join(jobDir, LearningDirName), nil
}

// GetProjectName returns the project name by reading .rick/PROJECT.md (first line),
// falling back to the module name in go.mod, then to filepath.Base(cwd).
func GetProjectName() (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return filepath.Base("."), nil
	}

	// 1. Try .rick/PROJECT.md first line
	rickDirName := getRickDirName()
	projectMDPath := filepath.Join(cwd, rickDirName, "PROJECT.md")
	if f, err := os.Open(projectMDPath); err == nil {
		defer f.Close()
		scanner := bufio.NewScanner(f)
		if scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			line = strings.TrimPrefix(line, "#")
			line = strings.TrimSpace(line)
			if line != "" {
				return line, nil
			}
		}
	}

	// 2. Try go.mod module name
	goModPath := filepath.Join(cwd, "go.mod")
	if data, err := os.ReadFile(goModPath); err == nil {
		for _, line := range strings.Split(string(data), "\n") {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, "module ") {
				modulePath := strings.TrimSpace(strings.TrimPrefix(line, "module "))
				if modulePath != "" {
					return filepath.Base(modulePath), nil
				}
			}
		}
	}

	// 3. Fallback to directory base name
	return filepath.Base(cwd), nil
}

// GetRFCDir returns the path to the RFC directory under .rick
func GetRFCDir() (string, error) {
	rickDir, err := GetRickDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(rickDir, "RFC"), nil
}

// NextJobID scans the jobs directory and returns the next job_N id.
// If no jobs exist yet, returns "job_1".
func NextJobID() (string, error) {
	jobsDir, err := GetJobsDir()
	if err != nil {
		return "", err
	}

	entries, err := os.ReadDir(jobsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return "job_1", nil
		}
		return "", fmt.Errorf("failed to read jobs directory: %w", err)
	}

	maxN := 0
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		var n int
		if _, err := fmt.Sscanf(e.Name(), "job_%d", &n); err == nil && n > 0 && n <= 9999 && n > maxN {
			maxN = n
		}
	}
	return fmt.Sprintf("job_%d", maxN+1), nil
}
