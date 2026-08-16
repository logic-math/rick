package workspace

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

// SelectPendingJobs returns up to jobNum completed jobs not yet processed by
// dream (migrated from cmd.selectPendingJobs). The scan/filtering stays in Go:
// it is the deterministic input filter that decides which jobs dream should
// process.
func SelectPendingJobs(rickDir string, jobNum int) []string {
	completed := DiscoverCompletedJobs(rickDir)
	processed := GetDreamProcessedJobs(rickDir)

	var pending []string
	for _, id := range completed {
		if !processed[id] {
			pending = append(pending, id)
		}
	}

	if len(pending) > jobNum {
		pending = pending[:jobNum]
	}
	return pending
}

// GetDreamProcessedJobs returns the set of job IDs that already have a
// dream_run_{job_id}_log.md file in .rick/dream/ (migrated from
// cmd.getDreamProcessedJobs).
func GetDreamProcessedJobs(rickDir string) map[string]bool {
	processed := make(map[string]bool)
	dreamDir := filepath.Join(rickDir, DreamDirName)
	entries, err := os.ReadDir(dreamDir)
	if err != nil {
		return processed
	}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		// Expected format: dream_run_job_N_log.md
		if !strings.HasPrefix(name, "dream_run_") || !strings.HasSuffix(name, "_log.md") {
			continue
		}
		jobID := strings.TrimPrefix(name, "dream_run_")
		jobID = strings.TrimSuffix(jobID, "_log.md")
		if strings.HasPrefix(jobID, "job_") {
			processed[jobID] = true
		}
	}
	return processed
}

// thinTaskState is the minimal tasks.json reader needed by dream's scan: it
// only decodes task status. The full thin reader type (with a stable public
// API) lands in task8; this local helper keeps workspace from importing
// executor (which would form workspace→executor→prompt→workspace).
type thinTaskState struct {
	TaskID string `json:"task_id"`
	Status string `json:"status"`
}

type thinTasksJSON struct {
	Tasks []thinTaskState `json:"tasks"`
}

func loadTasksJSONStatuses(path string) (*thinTasksJSON, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var tj thinTasksJSON
	if err := json.Unmarshal(data, &tj); err != nil {
		return nil, err
	}
	return &tj, nil
}

// DiscoverCompletedJobs scans .rick/jobs/*/doing/tasks.json and returns jobs
// where all tasks have status "success", sorted by job number ascending
// (migrated from cmd.discoverCompletedJobs).
func DiscoverCompletedJobs(rickDir string) []string {
	pattern := filepath.Join(rickDir, JobsDirName, "job_*", DoingDirName, "tasks.json")
	files, err := filepath.Glob(pattern)
	if err != nil {
		return nil
	}

	var completed []string
	for _, f := range files {
		tj, err := loadTasksJSONStatuses(f)
		if err != nil {
			continue
		}
		tasks := tj.Tasks
		if len(tasks) == 0 {
			continue
		}
		allDone := true
		for _, t := range tasks {
			if t.Status != "success" {
				allDone = false
				break
			}
		}
		if allDone {
			// Extract job ID from path: .rick/jobs/job_N/doing/tasks.json
			jobID := filepath.Base(filepath.Dir(filepath.Dir(f)))
			completed = append(completed, jobID)
		}
	}

	sort.Slice(completed, func(i, j int) bool {
		return JobNumber(completed[i]) < JobNumber(completed[j])
	})
	return completed
}

// JobNumber extracts the numeric part from "job_N" (migrated from
// cmd.jobNumber). Returns 0 for malformed IDs.
func JobNumber(jobID string) int {
	n, _ := strconv.Atoi(strings.TrimPrefix(jobID, "job_"))
	return n
}
