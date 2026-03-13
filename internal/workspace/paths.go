package workspace

import (
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

// GetRickDir returns the path to the .rick directory in the user's home
func GetRickDir() (string, error) {
	home, err := GetHomeDir()
	if err != nil {
		return "", err
	}
	rickDirName := getRickDirName()
	return filepath.Join(home, rickDirName), nil
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
