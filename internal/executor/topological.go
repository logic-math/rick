package executor

import (
	"fmt"
)

// TopologicalSort performs a topological sort on the DAG using Kahn's algorithm
// Returns a slice of task IDs in topological order
// Returns an error if a cycle is detected in the DAG
func TopologicalSort(dag *DAG) ([]string, error) {
	if dag == nil {
		return nil, fmt.Errorf("DAG cannot be nil")
	}

	// Calculate in-degrees for all tasks
	inDegrees := calculateInDegrees(dag)

	// Initialize queue with all tasks that have in-degree 0
	queue := make([]string, 0)
	for taskID, degree := range inDegrees {
		if degree == 0 {
			queue = append(queue, taskID)
		}
	}

	// Process tasks from queue
	result := make([]string, 0, len(dag.Tasks))
	for len(queue) > 0 {
		// Dequeue a task with in-degree 0
		current := queue[0]
		queue = queue[1:]
		result = append(result, current)

		// For each dependent of the current task
		dependents, err := dag.GetTaskDependents(current)
		if err != nil {
			return nil, err
		}

		for _, dependent := range dependents {
			// Decrease in-degree of dependent
			inDegrees[dependent]--

			// If in-degree becomes 0, add to queue
			if inDegrees[dependent] == 0 {
				queue = append(queue, dependent)
			}
		}
	}

	// Check if all tasks were processed (no cycles)
	if len(result) != len(dag.Tasks) {
		// Find tasks that were not processed (part of cycle)
		processedSet := make(map[string]bool)
		for _, taskID := range result {
			processedSet[taskID] = true
		}

		var cycledTasks []string
		for taskID := range dag.Tasks {
			if !processedSet[taskID] {
				cycledTasks = append(cycledTasks, taskID)
			}
		}

		return nil, fmt.Errorf("cycle detected in DAG: tasks %v form a cycle", cycledTasks)
	}

	return result, nil
}

// calculateInDegrees calculates the in-degree for each task in the DAG
// In-degree is the number of tasks that a task depends on
func calculateInDegrees(dag *DAG) map[string]int {
	inDegrees := make(map[string]int)

	// Initialize all tasks with in-degree 0
	for taskID := range dag.Tasks {
		inDegrees[taskID] = 0
	}

	// For each task, count how many other tasks it depends on
	for taskID, task := range dag.Tasks {
		for range task.Dependencies {
			inDegrees[taskID]++
		}
	}

	return inDegrees
}
