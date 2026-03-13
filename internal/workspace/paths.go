package workspace

import (
	"os"
	"path/filepath"
)

// Path constants for the Rick workspace structure
const (
	RickDirName    = ".rick"
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

// GetRickDir returns the path to the .rick directory in the user's home
func GetRickDir() (string, error) {
	home, err := GetHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, RickDirName), nil
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
