package workspace

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// TaskState is the minimal task state read from doing/tasks.json. It is the
// thin reader that replaces the deleted internal/executor.TasksJSON for the
// surviving consumers (dream scan + learning data collection).
type TaskState struct {
	TaskID       string    `json:"task_id"`
	TaskName     string    `json:"task_name"`
	TaskFile     string    `json:"task_file,omitempty"`
	Status       string    `json:"status"` // pending, running, success, failed, retrying
	Dependencies []string  `json:"dependencies"`
	Attempts     int       `json:"attempts"`
	Error        string    `json:"error,omitempty"`
	Output       string    `json:"output,omitempty"`
	CommitHash   string    `json:"commit_hash,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// TasksJSON is the minimal doing/tasks.json envelope.
type TasksJSON struct {
	Version   string      `json:"version"`
	CreatedAt time.Time   `json:"created_at"`
	UpdatedAt time.Time   `json:"updated_at"`
	Tasks     []TaskState `json:"tasks"`

	taskMap map[string]*TaskState
}

// LoadTasksJSON reads and parses doing/tasks.json, rebuilding the lookup map.
func LoadTasksJSON(filePath string) (*TasksJSON, error) {
	if filePath == "" {
		return nil, fmt.Errorf("file path cannot be empty")
	}
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read tasks.json: %w", err)
	}
	var tj TasksJSON
	if err := json.Unmarshal(data, &tj); err != nil {
		return nil, fmt.Errorf("failed to unmarshal tasks.json: %w", err)
	}
	tj.taskMap = make(map[string]*TaskState, len(tj.Tasks))
	for i := range tj.Tasks {
		tj.taskMap[tj.Tasks[i].TaskID] = &tj.Tasks[i]
	}
	return &tj, nil
}

// GetTaskStatus returns the status of a task, or an error if unknown.
func (tj *TasksJSON) GetTaskStatus(taskID string) (string, error) {
	ts, ok := tj.taskMap[taskID]
	if !ok {
		return "", fmt.Errorf("task '%s' not found", taskID)
	}
	return ts.Status, nil
}

// GetTask returns the full task state, or an error if unknown.
func (tj *TasksJSON) GetTask(taskID string) (*TaskState, error) {
	ts, ok := tj.taskMap[taskID]
	if !ok {
		return nil, fmt.Errorf("task '%s' not found", taskID)
	}
	return ts, nil
}

// GetAllTasks returns every task state in slice order.
func (tj *TasksJSON) GetAllTasks() []TaskState {
	return tj.Tasks
}

// GetTaskCount returns the number of tasks.
func (tj *TasksJSON) GetTaskCount() int { return len(tj.Tasks) }

// GetCompletedCount returns the number of success tasks.
func (tj *TasksJSON) GetCompletedCount() int {
	n := 0
	for _, t := range tj.Tasks {
		if t.Status == "success" {
			n++
		}
	}
	return n
}

// ExtractBugFrontmatter parses YAML frontmatter (between --- markers) and
// extracts summary and status fields (the thin replacement for the deleted
// internal/parser.ExtractBugFrontmatter).
func ExtractBugFrontmatter(content string) (summary, status string) {
	lines := strings.Split(content, "\n")
	inFrontmatter := false
	started := false
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "---" {
			if !started {
				inFrontmatter = true
				started = true
				continue
			}
			if inFrontmatter {
				break
			}
		}
		if !inFrontmatter {
			continue
		}
		if strings.HasPrefix(trimmed, "summary:") {
			summary = strings.Trim(strings.TrimSpace(strings.TrimPrefix(trimmed, "summary:")), `"'`)
		} else if strings.HasPrefix(trimmed, "status:") {
			status = strings.Trim(strings.TrimSpace(strings.TrimPrefix(trimmed, "status:")), `"'`)
		}
	}
	return summary, status
}

// LoadDebugDirSummaries scans {doingDir}/debug/, reads bug*.md in lexicographic
// order, and returns a multi-line string of frontmatter summaries.
func LoadDebugDirSummaries(doingDir string) string {
	if doingDir == "" {
		return ""
	}
	debugDir := filepath.Join(doingDir, "debug")
	entries, err := os.ReadDir(debugDir)
	if err != nil {
		return ""
	}
	var files []string
	for _, e := range entries {
		name := e.Name()
		if !e.IsDir() && strings.HasPrefix(name, "bug") && strings.HasSuffix(name, ".md") {
			files = append(files, name)
		}
	}
	if len(files) == 0 {
		return ""
	}
	sort.Strings(files)
	var sb strings.Builder
	for _, name := range files {
		data, err := os.ReadFile(filepath.Join(debugDir, name))
		if err != nil {
			continue
		}
		summary, status := ExtractBugFrontmatter(string(data))
		sb.WriteString(fmt.Sprintf("- [%s] summary: %s | status: %s\n", name, summary, status))
	}
	return sb.String()
}

// LoadDebugContext is the unified debug context loader. Prefers bug*.md
// frontmatter summaries from {doingDir}/debug/; falls back to {doingDir}/debug.md.
// Returns "" (not an error) when doingDir is empty or does not exist.
func LoadDebugContext(doingDir string) string {
	if doingDir == "" {
		return ""
	}
	if _, err := os.Stat(doingDir); os.IsNotExist(err) {
		return ""
	}
	if summaries := LoadDebugDirSummaries(doingDir); summaries != "" {
		return summaries
	}
	data, err := os.ReadFile(filepath.Join(doingDir, "debug.md"))
	if err != nil {
		return ""
	}
	return string(data)
}
