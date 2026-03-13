package executor

import (
	"testing"

	"github.com/sunquan/rick/internal/parser"
)

// TestTopologicalSortLinearDependency tests sorting with linear dependencies
func TestTopologicalSortLinearDependency(t *testing.T) {
	// Create tasks: task1 -> task2 -> task3
	tasks := []*parser.Task{
		{ID: "task1", Name: "Task 1", Goal: "Goal 1", TestMethod: "Test 1", Dependencies: []string{}},
		{ID: "task2", Name: "Task 2", Goal: "Goal 2", TestMethod: "Test 2", Dependencies: []string{"task1"}},
		{ID: "task3", Name: "Task 3", Goal: "Goal 3", TestMethod: "Test 3", Dependencies: []string{"task2"}},
	}

	dag, err := NewDAG(tasks)
	if err != nil {
		t.Fatalf("Failed to create DAG: %v", err)
	}

	result, err := TopologicalSort(dag)
	if err != nil {
		t.Fatalf("TopologicalSort failed: %v", err)
	}

	// Verify result length
	if len(result) != 3 {
		t.Errorf("Expected 3 tasks, got %d", len(result))
	}

	// Verify order
	if result[0] != "task1" || result[1] != "task2" || result[2] != "task3" {
		t.Errorf("Expected [task1, task2, task3], got %v", result)
	}
}

// TestTopologicalSortMultipleDependencies tests sorting with multiple dependencies
func TestTopologicalSortMultipleDependencies(t *testing.T) {
	// Create tasks:
	//   task1, task2 (no deps)
	//   task3 (depends on task1, task2)
	//   task4 (depends on task3)
	tasks := []*parser.Task{
		{ID: "task1", Name: "Task 1", Goal: "Goal 1", TestMethod: "Test 1", Dependencies: []string{}},
		{ID: "task2", Name: "Task 2", Goal: "Goal 2", TestMethod: "Test 2", Dependencies: []string{}},
		{ID: "task3", Name: "Task 3", Goal: "Goal 3", TestMethod: "Test 3", Dependencies: []string{"task1", "task2"}},
		{ID: "task4", Name: "Task 4", Goal: "Goal 4", TestMethod: "Test 4", Dependencies: []string{"task3"}},
	}

	dag, err := NewDAG(tasks)
	if err != nil {
		t.Fatalf("Failed to create DAG: %v", err)
	}

	result, err := TopologicalSort(dag)
	if err != nil {
		t.Fatalf("TopologicalSort failed: %v", err)
	}

	// Verify result length
	if len(result) != 4 {
		t.Errorf("Expected 4 tasks, got %d", len(result))
	}

	// Verify ordering constraints
	task1Idx, task2Idx, task3Idx, task4Idx := -1, -1, -1, -1
	for i, taskID := range result {
		switch taskID {
		case "task1":
			task1Idx = i
		case "task2":
			task2Idx = i
		case "task3":
			task3Idx = i
		case "task4":
			task4Idx = i
		}
	}

	if task1Idx >= task3Idx || task2Idx >= task3Idx {
		t.Errorf("task3 must come after task1 and task2")
	}
	if task3Idx >= task4Idx {
		t.Errorf("task4 must come after task3")
	}
}

// TestTopologicalSortIndependentTasks tests sorting with independent tasks
func TestTopologicalSortIndependentTasks(t *testing.T) {
	// Create tasks with no dependencies
	tasks := []*parser.Task{
		{ID: "task1", Name: "Task 1", Goal: "Goal 1", TestMethod: "Test 1", Dependencies: []string{}},
		{ID: "task2", Name: "Task 2", Goal: "Goal 2", TestMethod: "Test 2", Dependencies: []string{}},
		{ID: "task3", Name: "Task 3", Goal: "Goal 3", TestMethod: "Test 3", Dependencies: []string{}},
	}

	dag, err := NewDAG(tasks)
	if err != nil {
		t.Fatalf("Failed to create DAG: %v", err)
	}

	result, err := TopologicalSort(dag)
	if err != nil {
		t.Fatalf("TopologicalSort failed: %v", err)
	}

	// Verify result length
	if len(result) != 3 {
		t.Errorf("Expected 3 tasks, got %d", len(result))
	}

	// Verify all tasks are present
	taskSet := make(map[string]bool)
	for _, taskID := range result {
		taskSet[taskID] = true
	}

	if len(taskSet) != 3 {
		t.Errorf("Expected 3 unique tasks, got %d", len(taskSet))
	}
}

// TestTopologicalSortComplexDAG tests sorting with complex DAG structure
func TestTopologicalSortComplexDAG(t *testing.T) {
	// Create complex DAG:
	//     task1 -> task3 -> task5
	//     task2 -> task4 -> task5
	tasks := []*parser.Task{
		{ID: "task1", Name: "Task 1", Goal: "Goal 1", TestMethod: "Test 1", Dependencies: []string{}},
		{ID: "task2", Name: "Task 2", Goal: "Goal 2", TestMethod: "Test 2", Dependencies: []string{}},
		{ID: "task3", Name: "Task 3", Goal: "Goal 3", TestMethod: "Test 3", Dependencies: []string{"task1"}},
		{ID: "task4", Name: "Task 4", Goal: "Goal 4", TestMethod: "Test 4", Dependencies: []string{"task2"}},
		{ID: "task5", Name: "Task 5", Goal: "Goal 5", TestMethod: "Test 5", Dependencies: []string{"task3", "task4"}},
	}

	dag, err := NewDAG(tasks)
	if err != nil {
		t.Fatalf("Failed to create DAG: %v", err)
	}

	result, err := TopologicalSort(dag)
	if err != nil {
		t.Fatalf("TopologicalSort failed: %v", err)
	}

	// Verify result length
	if len(result) != 5 {
		t.Errorf("Expected 5 tasks, got %d", len(result))
	}

	// Verify ordering constraints
	indices := make(map[string]int)
	for i, taskID := range result {
		indices[taskID] = i
	}

	if indices["task1"] >= indices["task3"] {
		t.Errorf("task1 must come before task3")
	}
	if indices["task2"] >= indices["task4"] {
		t.Errorf("task2 must come before task4")
	}
	if indices["task3"] >= indices["task5"] || indices["task4"] >= indices["task5"] {
		t.Errorf("task3 and task4 must come before task5")
	}
}

// TestTopologicalSortEmptyDAG tests sorting with empty DAG
func TestTopologicalSortEmptyDAG(t *testing.T) {
	dag, err := NewDAG([]*parser.Task{})
	if err != nil {
		t.Fatalf("Failed to create empty DAG: %v", err)
	}

	result, err := TopologicalSort(dag)
	if err != nil {
		t.Fatalf("TopologicalSort failed: %v", err)
	}

	if len(result) != 0 {
		t.Errorf("Expected empty result, got %v", result)
	}
}

// TestTopologicalSortSingleTask tests sorting with single task
func TestTopologicalSortSingleTask(t *testing.T) {
	tasks := []*parser.Task{
		{ID: "task1", Name: "Task 1", Goal: "Goal 1", TestMethod: "Test 1", Dependencies: []string{}},
	}

	dag, err := NewDAG(tasks)
	if err != nil {
		t.Fatalf("Failed to create DAG: %v", err)
	}

	result, err := TopologicalSort(dag)
	if err != nil {
		t.Fatalf("TopologicalSort failed: %v", err)
	}

	if len(result) != 1 || result[0] != "task1" {
		t.Errorf("Expected [task1], got %v", result)
	}
}

// TestTopologicalSortDiamondDAG tests sorting with diamond-shaped DAG
func TestTopologicalSortDiamondDAG(t *testing.T) {
	// Create diamond DAG:
	//     task1 -> task2, task3
	//     task2, task3 -> task4
	tasks := []*parser.Task{
		{ID: "task1", Name: "Task 1", Goal: "Goal 1", TestMethod: "Test 1", Dependencies: []string{}},
		{ID: "task2", Name: "Task 2", Goal: "Goal 2", TestMethod: "Test 2", Dependencies: []string{"task1"}},
		{ID: "task3", Name: "Task 3", Goal: "Goal 3", TestMethod: "Test 3", Dependencies: []string{"task1"}},
		{ID: "task4", Name: "Task 4", Goal: "Goal 4", TestMethod: "Test 4", Dependencies: []string{"task2", "task3"}},
	}

	dag, err := NewDAG(tasks)
	if err != nil {
		t.Fatalf("Failed to create DAG: %v", err)
	}

	result, err := TopologicalSort(dag)
	if err != nil {
		t.Fatalf("TopologicalSort failed: %v", err)
	}

	// Verify result length
	if len(result) != 4 {
		t.Errorf("Expected 4 tasks, got %d", len(result))
	}

	// Verify ordering constraints
	indices := make(map[string]int)
	for i, taskID := range result {
		indices[taskID] = i
	}

	if indices["task1"] >= indices["task2"] || indices["task1"] >= indices["task3"] {
		t.Errorf("task1 must come before task2 and task3")
	}
	if indices["task2"] >= indices["task4"] || indices["task3"] >= indices["task4"] {
		t.Errorf("task2 and task3 must come before task4")
	}
}

// TestTopologicalSortNilDAG tests sorting with nil DAG
func TestTopologicalSortNilDAG(t *testing.T) {
	result, err := TopologicalSort(nil)
	if err == nil {
		t.Errorf("Expected error for nil DAG, got nil")
	}
	if result != nil {
		t.Errorf("Expected nil result for nil DAG, got %v", result)
	}
}

// TestCalculateInDegrees tests in-degree calculation
func TestCalculateInDegrees(t *testing.T) {
	// Create tasks: task1 -> task2 -> task3
	tasks := []*parser.Task{
		{ID: "task1", Name: "Task 1", Goal: "Goal 1", TestMethod: "Test 1", Dependencies: []string{}},
		{ID: "task2", Name: "Task 2", Goal: "Goal 2", TestMethod: "Test 2", Dependencies: []string{"task1"}},
		{ID: "task3", Name: "Task 3", Goal: "Goal 3", TestMethod: "Test 3", Dependencies: []string{"task2"}},
	}

	dag, err := NewDAG(tasks)
	if err != nil {
		t.Fatalf("Failed to create DAG: %v", err)
	}

	inDegrees := calculateInDegrees(dag)

	// Verify in-degrees
	if inDegrees["task1"] != 0 {
		t.Errorf("Expected in-degree 0 for task1, got %d", inDegrees["task1"])
	}
	if inDegrees["task2"] != 1 {
		t.Errorf("Expected in-degree 1 for task2, got %d", inDegrees["task2"])
	}
	if inDegrees["task3"] != 1 {
		t.Errorf("Expected in-degree 1 for task3, got %d", inDegrees["task3"])
	}
}

// TestCalculateInDegreesMultipleDependencies tests in-degree calculation with multiple dependencies
func TestCalculateInDegreesMultipleDependencies(t *testing.T) {
	// Create tasks:
	//   task1, task2 (no deps)
	//   task3 (depends on task1, task2)
	tasks := []*parser.Task{
		{ID: "task1", Name: "Task 1", Goal: "Goal 1", TestMethod: "Test 1", Dependencies: []string{}},
		{ID: "task2", Name: "Task 2", Goal: "Goal 2", TestMethod: "Test 2", Dependencies: []string{}},
		{ID: "task3", Name: "Task 3", Goal: "Goal 3", TestMethod: "Test 3", Dependencies: []string{"task1", "task2"}},
	}

	dag, err := NewDAG(tasks)
	if err != nil {
		t.Fatalf("Failed to create DAG: %v", err)
	}

	inDegrees := calculateInDegrees(dag)

	// Verify in-degrees
	if inDegrees["task1"] != 0 {
		t.Errorf("Expected in-degree 0 for task1, got %d", inDegrees["task1"])
	}
	if inDegrees["task2"] != 0 {
		t.Errorf("Expected in-degree 0 for task2, got %d", inDegrees["task2"])
	}
	if inDegrees["task3"] != 2 {
		t.Errorf("Expected in-degree 2 for task3, got %d", inDegrees["task3"])
	}
}

// TestTopologicalSortPreservesAllTasks verifies all tasks are in the result
func TestTopologicalSortPreservesAllTasks(t *testing.T) {
	tasks := []*parser.Task{
		{ID: "task1", Name: "Task 1", Goal: "Goal 1", TestMethod: "Test 1", Dependencies: []string{}},
		{ID: "task2", Name: "Task 2", Goal: "Goal 2", TestMethod: "Test 2", Dependencies: []string{}},
		{ID: "task3", Name: "Task 3", Goal: "Goal 3", TestMethod: "Test 3", Dependencies: []string{"task1"}},
		{ID: "task4", Name: "Task 4", Goal: "Goal 4", TestMethod: "Test 4", Dependencies: []string{"task2"}},
		{ID: "task5", Name: "Task 5", Goal: "Goal 5", TestMethod: "Test 5", Dependencies: []string{"task3", "task4"}},
	}

	dag, err := NewDAG(tasks)
	if err != nil {
		t.Fatalf("Failed to create DAG: %v", err)
	}

	result, err := TopologicalSort(dag)
	if err != nil {
		t.Fatalf("TopologicalSort failed: %v", err)
	}

	// Verify all tasks are in result
	resultSet := make(map[string]bool)
	for _, taskID := range result {
		resultSet[taskID] = true
	}

	for _, task := range tasks {
		if !resultSet[task.ID] {
			t.Errorf("Task %s missing from result", task.ID)
		}
	}
}

// TestTopologicalSortResultSatisfiesDependencies verifies dependencies are satisfied
func TestTopologicalSortResultSatisfiesDependencies(t *testing.T) {
	tasks := []*parser.Task{
		{ID: "task1", Name: "Task 1", Goal: "Goal 1", TestMethod: "Test 1", Dependencies: []string{}},
		{ID: "task2", Name: "Task 2", Goal: "Goal 2", TestMethod: "Test 2", Dependencies: []string{"task1"}},
		{ID: "task3", Name: "Task 3", Goal: "Goal 3", TestMethod: "Test 3", Dependencies: []string{"task1", "task2"}},
		{ID: "task4", Name: "Task 4", Goal: "Goal 4", TestMethod: "Test 4", Dependencies: []string{"task3"}},
	}

	dag, err := NewDAG(tasks)
	if err != nil {
		t.Fatalf("Failed to create DAG: %v", err)
	}

	result, err := TopologicalSort(dag)
	if err != nil {
		t.Fatalf("TopologicalSort failed: %v", err)
	}

	// Create index map for quick lookup
	indices := make(map[string]int)
	for i, taskID := range result {
		indices[taskID] = i
	}

	// Verify all dependencies are satisfied
	for _, task := range tasks {
		for _, dep := range task.Dependencies {
			if indices[dep] >= indices[task.ID] {
				t.Errorf("Dependency violated: %s should come before %s", dep, task.ID)
			}
		}
	}
}
