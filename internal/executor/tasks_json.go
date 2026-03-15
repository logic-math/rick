package executor

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"time"
)

// TaskState represents the state of a task in tasks.json
type TaskState struct {
	TaskID      string    `json:"task_id"`
	TaskName    string    `json:"task_name"`
	TaskFile    string    `json:"task_file,omitempty"` // task.md filename
	Status      string    `json:"status"` // pending, running, success, failed, retrying
	Dependencies []string `json:"dependencies"`
	Attempts    int       `json:"attempts"`
	Error       string    `json:"error,omitempty"`
	Output      string    `json:"output,omitempty"`
	CommitHash  string    `json:"commit_hash,omitempty"` // Git commit hash when task completed
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// TasksJSON represents the entire tasks.json structure
type TasksJSON struct {
	Version   string       `json:"version"`
	CreatedAt time.Time    `json:"created_at"`
	UpdatedAt time.Time    `json:"updated_at"`
	Tasks     []TaskState  `json:"tasks"`
	taskMap   map[string]*TaskState // internal map for fast lookup
}

// GenerateTasksJSON generates a TasksJSON structure from a DAG and sorted tasks
// dag: the DAG structure containing task definitions
// sortedTasks: task IDs in topological order
func GenerateTasksJSON(dag *DAG, sortedTasks []string) (*TasksJSON, error) {
	if dag == nil {
		return nil, fmt.Errorf("DAG cannot be nil")
	}
	if len(sortedTasks) == 0 {
		return nil, fmt.Errorf("sortedTasks cannot be empty")
	}

	now := time.Now()
	tasksJSON := &TasksJSON{
		Version:   "1.0",
		CreatedAt: now,
		UpdatedAt: now,
		Tasks:     make([]TaskState, 0, len(sortedTasks)),
		taskMap:   make(map[string]*TaskState),
	}

	// Create TaskState for each sorted task
	for _, taskID := range sortedTasks {
		task, exists := dag.Tasks[taskID]
		if !exists {
			return nil, fmt.Errorf("task '%s' not found in DAG", taskID)
		}

		taskState := TaskState{
			TaskID:       taskID,
			TaskName:     task.Name,
			Status:       "pending",
			Dependencies: task.Dependencies,
			Attempts:     0,
			CreatedAt:    now,
			UpdatedAt:    now,
		}

		tasksJSON.Tasks = append(tasksJSON.Tasks, taskState)
		tasksJSON.taskMap[taskID] = &tasksJSON.Tasks[len(tasksJSON.Tasks)-1]
	}

	return tasksJSON, nil
}

// LoadTasksJSON loads a TasksJSON from a file
func LoadTasksJSON(filePath string) (*TasksJSON, error) {
	if filePath == "" {
		return nil, fmt.Errorf("file path cannot be empty")
	}

	data, err := ioutil.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read tasks.json: %w", err)
	}

	var tasksJSON TasksJSON
	if err := json.Unmarshal(data, &tasksJSON); err != nil {
		return nil, fmt.Errorf("failed to unmarshal tasks.json: %w", err)
	}

	// Rebuild the internal task map
	tasksJSON.taskMap = make(map[string]*TaskState)
	for i := range tasksJSON.Tasks {
		tasksJSON.taskMap[tasksJSON.Tasks[i].TaskID] = &tasksJSON.Tasks[i]
	}

	return &tasksJSON, nil
}

// SaveTasksJSON saves a TasksJSON to a file
func SaveTasksJSON(filePath string, tasksJSON *TasksJSON) error {
	if filePath == "" {
		return fmt.Errorf("file path cannot be empty")
	}
	if tasksJSON == nil {
		return fmt.Errorf("tasksJSON cannot be nil")
	}

	// Update the UpdatedAt timestamp
	tasksJSON.UpdatedAt = time.Now()

	data, err := json.MarshalIndent(tasksJSON, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal tasks.json: %w", err)
	}

	if err := ioutil.WriteFile(filePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write tasks.json: %w", err)
	}

	return nil
}

// UpdateTaskStatus updates the status of a task
// taskID: the ID of the task to update
// status: the new status (pending, running, success, failed, retrying)
func (tj *TasksJSON) UpdateTaskStatus(taskID, status string) error {
	if taskID == "" {
		return fmt.Errorf("task ID cannot be empty")
	}

	// Validate status
	validStatuses := map[string]bool{
		"pending":   true,
		"running":   true,
		"success":   true,
		"failed":    true,
		"retrying":  true,
	}
	if !validStatuses[status] {
		return fmt.Errorf("invalid status '%s', must be one of: pending, running, success, failed, retrying", status)
	}

	taskState, exists := tj.taskMap[taskID]
	if !exists {
		return fmt.Errorf("task '%s' not found", taskID)
	}

	taskState.Status = status
	taskState.UpdatedAt = time.Now()

	return nil
}

// UpdateTaskStatusWithError updates the status of a task with error information
func (tj *TasksJSON) UpdateTaskStatusWithError(taskID, status, errorMsg string) error {
	if err := tj.UpdateTaskStatus(taskID, status); err != nil {
		return err
	}

	taskState, _ := tj.taskMap[taskID]
	taskState.Error = errorMsg
	taskState.UpdatedAt = time.Now()

	return nil
}

// UpdateTaskStatusWithOutput updates the status of a task with output information
func (tj *TasksJSON) UpdateTaskStatusWithOutput(taskID, status, output string) error {
	if err := tj.UpdateTaskStatus(taskID, status); err != nil {
		return err
	}

	taskState, _ := tj.taskMap[taskID]
	taskState.Output = output
	taskState.UpdatedAt = time.Now()

	return nil
}

// GetTaskStatus returns the status of a task
// Returns the status string or an error if the task is not found
func (tj *TasksJSON) GetTaskStatus(taskID string) (string, error) {
	if taskID == "" {
		return "", fmt.Errorf("task ID cannot be empty")
	}

	taskState, exists := tj.taskMap[taskID]
	if !exists {
		return "", fmt.Errorf("task '%s' not found", taskID)
	}

	return taskState.Status, nil
}

// GetTask returns the full TaskState for a task
func (tj *TasksJSON) GetTask(taskID string) (*TaskState, error) {
	if taskID == "" {
		return nil, fmt.Errorf("task ID cannot be empty")
	}

	taskState, exists := tj.taskMap[taskID]
	if !exists {
		return nil, fmt.Errorf("task '%s' not found", taskID)
	}

	return taskState, nil
}

// IncrementAttempts increments the attempt count for a task
func (tj *TasksJSON) IncrementAttempts(taskID string) error {
	if taskID == "" {
		return fmt.Errorf("task ID cannot be empty")
	}

	taskState, exists := tj.taskMap[taskID]
	if !exists {
		return fmt.Errorf("task '%s' not found", taskID)
	}

	taskState.Attempts++
	taskState.UpdatedAt = time.Now()

	return nil
}

// GetAllTasks returns all tasks in the TasksJSON
func (tj *TasksJSON) GetAllTasks() []TaskState {
	return tj.Tasks
}

// GetTasksByStatus returns all tasks with a specific status
func (tj *TasksJSON) GetTasksByStatus(status string) []TaskState {
	var result []TaskState
	for _, task := range tj.Tasks {
		if task.Status == status {
			result = append(result, task)
		}
	}
	return result
}

// GetCompletedTasks returns all tasks with status "success"
func (tj *TasksJSON) GetCompletedTasks() []TaskState {
	return tj.GetTasksByStatus("success")
}

// GetFailedTasks returns all tasks with status "failed"
func (tj *TasksJSON) GetFailedTasks() []TaskState {
	return tj.GetTasksByStatus("failed")
}

// GetPendingTasks returns all tasks with status "pending"
func (tj *TasksJSON) GetPendingTasks() []TaskState {
	return tj.GetTasksByStatus("pending")
}

// GetTaskCount returns the total number of tasks
func (tj *TasksJSON) GetTaskCount() int {
	return len(tj.Tasks)
}

// GetCompletedCount returns the number of completed tasks
func (tj *TasksJSON) GetCompletedCount() int {
	return len(tj.GetCompletedTasks())
}

// GetFailedCount returns the number of failed tasks
func (tj *TasksJSON) GetFailedCount() int {
	return len(tj.GetFailedTasks())
}

// GetPendingCount returns the number of pending tasks
func (tj *TasksJSON) GetPendingCount() int {
	return len(tj.GetPendingTasks())
}

// IsAllCompleted checks if all tasks are completed
func (tj *TasksJSON) IsAllCompleted() bool {
	return tj.GetCompletedCount() == tj.GetTaskCount()
}

// IsAnyFailed checks if any task has failed
func (tj *TasksJSON) IsAnyFailed() bool {
	return tj.GetFailedCount() > 0
}

// UpdateTaskCommit updates the commit hash for a task
func (tj *TasksJSON) UpdateTaskCommit(taskID, commitHash string) error {
	if taskID == "" {
		return fmt.Errorf("task ID cannot be empty")
	}

	taskState, exists := tj.taskMap[taskID]
	if !exists {
		return fmt.Errorf("task '%s' not found", taskID)
	}

	taskState.CommitHash = commitHash
	taskState.UpdatedAt = time.Now()

	return nil
}

// UpdateTaskFile updates the task file name for a task
func (tj *TasksJSON) UpdateTaskFile(taskID, taskFile string) error {
	if taskID == "" {
		return fmt.Errorf("task ID cannot be empty")
	}

	taskState, exists := tj.taskMap[taskID]
	if !exists {
		return fmt.Errorf("task '%s' not found", taskID)
	}

	taskState.TaskFile = taskFile
	taskState.UpdatedAt = time.Now()

	return nil
}
