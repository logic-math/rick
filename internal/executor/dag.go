package executor

import (
	"fmt"
	"github.com/sunquan/rick/internal/parser"
)

// DAG represents a directed acyclic graph of tasks
type DAG struct {
	Tasks map[string]*parser.Task
	Graph map[string][]string // task_id -> list of dependent task_ids
}

// NewDAG creates a new DAG instance from a list of tasks
func NewDAG(tasks []*parser.Task) (*DAG, error) {
	if len(tasks) == 0 {
		return &DAG{
			Tasks: make(map[string]*parser.Task),
			Graph: make(map[string][]string),
		}, nil
	}

	dag := &DAG{
		Tasks: make(map[string]*parser.Task),
		Graph: make(map[string][]string),
	}

	// Add all tasks to the DAG
	for _, task := range tasks {
		if err := dag.AddTask(task); err != nil {
			return nil, err
		}
	}

	// Add all dependencies
	for _, task := range tasks {
		for _, dep := range task.Dependencies {
			if err := dag.AddDependency(dep, task.ID); err != nil {
				return nil, err
			}
		}
	}

	// Validate the DAG
	if err := dag.ValidateDAG(); err != nil {
		return nil, err
	}

	return dag, nil
}

// AddTask adds a task to the DAG
func (d *DAG) AddTask(task *parser.Task) error {
	if task == nil {
		return fmt.Errorf("task cannot be nil")
	}
	if task.ID == "" {
		return fmt.Errorf("task ID cannot be empty")
	}

	d.Tasks[task.ID] = task
	if _, exists := d.Graph[task.ID]; !exists {
		d.Graph[task.ID] = []string{}
	}

	return nil
}

// AddDependency adds a dependency relationship: from -> to (to depends on from)
func (d *DAG) AddDependency(from, to string) error {
	if from == "" || to == "" {
		return fmt.Errorf("task IDs cannot be empty")
	}

	// Check if both tasks exist
	if _, exists := d.Tasks[from]; !exists {
		return fmt.Errorf("source task '%s' not found", from)
	}
	if _, exists := d.Tasks[to]; !exists {
		return fmt.Errorf("target task '%s' not found", to)
	}

	// Add the edge
	if d.Graph[from] == nil {
		d.Graph[from] = []string{}
	}
	d.Graph[from] = append(d.Graph[from], to)

	return nil
}

// ValidateDAG validates the DAG for cycles and consistency
func (d *DAG) ValidateDAG() error {
	// Check for cycles using DFS
	visited := make(map[string]int) // 0: unvisited, 1: visiting, 2: visited
	var cycle []string

	for taskID := range d.Tasks {
		if visited[taskID] == 0 {
			if hasCycle := d.detectCycleDFS(taskID, visited, &cycle); hasCycle {
				return fmt.Errorf("cycle detected in DAG: %v", cycle)
			}
		}
	}

	// Verify all dependencies reference existing tasks
	for taskID, task := range d.Tasks {
		for _, dep := range task.Dependencies {
			if _, exists := d.Tasks[dep]; !exists {
				return fmt.Errorf("task '%s' depends on non-existent task '%s'", taskID, dep)
			}
		}
	}

	return nil
}

// detectCycleDFS detects cycles using depth-first search
// Returns true if a cycle is found, false otherwise
func (d *DAG) detectCycleDFS(taskID string, visited map[string]int, cycle *[]string) bool {
	visited[taskID] = 1 // Mark as visiting

	// Check all tasks that depend on this task
	for _, dependent := range d.Graph[taskID] {
		if visited[dependent] == 1 {
			// Found a back edge, cycle detected
			*cycle = append(*cycle, taskID, "->", dependent)
			return true
		}
		if visited[dependent] == 0 {
			if d.detectCycleDFS(dependent, visited, cycle) {
				*cycle = append([]string{taskID, "->"}, *cycle...)
				return true
			}
		}
	}

	visited[taskID] = 2 // Mark as visited
	return false
}

// GetTaskDependencies returns all direct dependencies of a task
func (d *DAG) GetTaskDependencies(taskID string) ([]string, error) {
	task, exists := d.Tasks[taskID]
	if !exists {
		return nil, fmt.Errorf("task '%s' not found", taskID)
	}
	return task.Dependencies, nil
}

// GetTaskDependents returns all tasks that depend on a given task
func (d *DAG) GetTaskDependents(taskID string) ([]string, error) {
	if _, exists := d.Tasks[taskID]; !exists {
		return nil, fmt.Errorf("task '%s' not found", taskID)
	}

	dependents, exists := d.Graph[taskID]
	if !exists {
		return []string{}, nil
	}
	return dependents, nil
}

// GetAllTasks returns all tasks in the DAG
func (d *DAG) GetAllTasks() []*parser.Task {
	tasks := make([]*parser.Task, 0, len(d.Tasks))
	for _, task := range d.Tasks {
		tasks = append(tasks, task)
	}
	return tasks
}

// TaskCount returns the total number of tasks in the DAG
func (d *DAG) TaskCount() int {
	return len(d.Tasks)
}
