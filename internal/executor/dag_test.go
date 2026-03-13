package executor

import (
	"github.com/sunquan/rick/internal/parser"
	"testing"
)

// Helper function to create a test task
func createTestTask(id, name, goal string, deps []string) *parser.Task {
	return &parser.Task{
		ID:           id,
		Name:         name,
		Goal:         goal,
		Dependencies: deps,
		KeyResults:   []string{},
		TestMethod:   "test method",
	}
}

// Test NewDAG with empty task list
func TestNewDAGEmpty(t *testing.T) {
	dag, err := NewDAG([]*parser.Task{})
	if err != nil {
		t.Fatalf("NewDAG failed with empty list: %v", err)
	}
	if dag.TaskCount() != 0 {
		t.Errorf("Expected 0 tasks, got %d", dag.TaskCount())
	}
}

// Test NewDAG with single task
func TestNewDAGSingleTask(t *testing.T) {
	task := createTestTask("task1", "Task 1", "Goal 1", []string{})
	dag, err := NewDAG([]*parser.Task{task})
	if err != nil {
		t.Fatalf("NewDAG failed: %v", err)
	}
	if dag.TaskCount() != 1 {
		t.Errorf("Expected 1 task, got %d", dag.TaskCount())
	}
	if _, exists := dag.Tasks["task1"]; !exists {
		t.Error("Task 'task1' not found in DAG")
	}
}

// Test NewDAG with multiple independent tasks
func TestNewDAGMultipleIndependentTasks(t *testing.T) {
	tasks := []*parser.Task{
		createTestTask("task1", "Task 1", "Goal 1", []string{}),
		createTestTask("task2", "Task 2", "Goal 2", []string{}),
		createTestTask("task3", "Task 3", "Goal 3", []string{}),
	}
	dag, err := NewDAG(tasks)
	if err != nil {
		t.Fatalf("NewDAG failed: %v", err)
	}
	if dag.TaskCount() != 3 {
		t.Errorf("Expected 3 tasks, got %d", dag.TaskCount())
	}
}

// Test NewDAG with linear dependency chain
func TestNewDAGLinearDependencies(t *testing.T) {
	tasks := []*parser.Task{
		createTestTask("task1", "Task 1", "Goal 1", []string{}),
		createTestTask("task2", "Task 2", "Goal 2", []string{"task1"}),
		createTestTask("task3", "Task 3", "Goal 3", []string{"task2"}),
	}
	dag, err := NewDAG(tasks)
	if err != nil {
		t.Fatalf("NewDAG failed: %v", err)
	}
	if dag.TaskCount() != 3 {
		t.Errorf("Expected 3 tasks, got %d", dag.TaskCount())
	}

	// Check dependencies
	deps, err := dag.GetTaskDependencies("task2")
	if err != nil {
		t.Fatalf("GetTaskDependencies failed: %v", err)
	}
	if len(deps) != 1 || deps[0] != "task1" {
		t.Errorf("Expected task2 to depend on task1, got %v", deps)
	}
}

// Test NewDAG with multiple dependencies
func TestNewDAGMultipleDependencies(t *testing.T) {
	tasks := []*parser.Task{
		createTestTask("task1", "Task 1", "Goal 1", []string{}),
		createTestTask("task2", "Task 2", "Goal 2", []string{}),
		createTestTask("task3", "Task 3", "Goal 3", []string{"task1", "task2"}),
	}
	dag, err := NewDAG(tasks)
	if err != nil {
		t.Fatalf("NewDAG failed: %v", err)
	}

	deps, err := dag.GetTaskDependencies("task3")
	if err != nil {
		t.Fatalf("GetTaskDependencies failed: %v", err)
	}
	if len(deps) != 2 {
		t.Errorf("Expected 2 dependencies, got %d", len(deps))
	}
}

// Test AddTask
func TestAddTask(t *testing.T) {
	dag := &DAG{
		Tasks: make(map[string]*parser.Task),
		Graph: make(map[string][]string),
	}

	task := createTestTask("task1", "Task 1", "Goal 1", []string{})
	err := dag.AddTask(task)
	if err != nil {
		t.Fatalf("AddTask failed: %v", err)
	}

	if _, exists := dag.Tasks["task1"]; !exists {
		t.Error("Task not added to DAG")
	}
}

// Test AddTask with nil task
func TestAddTaskNil(t *testing.T) {
	dag := &DAG{
		Tasks: make(map[string]*parser.Task),
		Graph: make(map[string][]string),
	}

	err := dag.AddTask(nil)
	if err == nil {
		t.Error("Expected error for nil task")
	}
}

// Test AddTask with empty ID
func TestAddTaskEmptyID(t *testing.T) {
	dag := &DAG{
		Tasks: make(map[string]*parser.Task),
		Graph: make(map[string][]string),
	}

	task := &parser.Task{ID: "", Name: "Task", Goal: "Goal"}
	err := dag.AddTask(task)
	if err == nil {
		t.Error("Expected error for empty task ID")
	}
}

// Test AddDependency
func TestAddDependency(t *testing.T) {
	dag := &DAG{
		Tasks: make(map[string]*parser.Task),
		Graph: make(map[string][]string),
	}

	task1 := createTestTask("task1", "Task 1", "Goal 1", []string{})
	task2 := createTestTask("task2", "Task 2", "Goal 2", []string{})
	dag.AddTask(task1)
	dag.AddTask(task2)

	err := dag.AddDependency("task1", "task2")
	if err != nil {
		t.Fatalf("AddDependency failed: %v", err)
	}

	dependents, err := dag.GetTaskDependents("task1")
	if err != nil {
		t.Fatalf("GetTaskDependents failed: %v", err)
	}
	if len(dependents) != 1 || dependents[0] != "task2" {
		t.Errorf("Expected task1 to have dependent task2, got %v", dependents)
	}
}

// Test AddDependency with non-existent source task
func TestAddDependencyMissingSource(t *testing.T) {
	dag := &DAG{
		Tasks: make(map[string]*parser.Task),
		Graph: make(map[string][]string),
	}

	task2 := createTestTask("task2", "Task 2", "Goal 2", []string{})
	dag.AddTask(task2)

	err := dag.AddDependency("task1", "task2")
	if err == nil {
		t.Error("Expected error for missing source task")
	}
}

// Test AddDependency with non-existent target task
func TestAddDependencyMissingTarget(t *testing.T) {
	dag := &DAG{
		Tasks: make(map[string]*parser.Task),
		Graph: make(map[string][]string),
	}

	task1 := createTestTask("task1", "Task 1", "Goal 1", []string{})
	dag.AddTask(task1)

	err := dag.AddDependency("task1", "task2")
	if err == nil {
		t.Error("Expected error for missing target task")
	}
}

// Test ValidateDAG with valid DAG
func TestValidateDAGValid(t *testing.T) {
	tasks := []*parser.Task{
		createTestTask("task1", "Task 1", "Goal 1", []string{}),
		createTestTask("task2", "Task 2", "Goal 2", []string{"task1"}),
		createTestTask("task3", "Task 3", "Goal 3", []string{"task2"}),
	}
	dag, err := NewDAG(tasks)
	if err != nil {
		t.Fatalf("NewDAG failed: %v", err)
	}

	err = dag.ValidateDAG()
	if err != nil {
		t.Fatalf("ValidateDAG failed: %v", err)
	}
}

// Test ValidateDAG with cycle (simple cycle)
func TestValidateDAGSimpleCycle(t *testing.T) {
	tasks := []*parser.Task{
		createTestTask("task1", "Task 1", "Goal 1", []string{"task2"}),
		createTestTask("task2", "Task 2", "Goal 2", []string{"task1"}),
	}
	_, err := NewDAG(tasks)
	if err == nil {
		t.Error("Expected error for DAG with cycle")
	}
}

// Test ValidateDAG with self-cycle
func TestValidateDAGSelfCycle(t *testing.T) {
	tasks := []*parser.Task{
		createTestTask("task1", "Task 1", "Goal 1", []string{"task1"}),
	}
	_, err := NewDAG(tasks)
	if err == nil {
		t.Error("Expected error for DAG with self-cycle")
	}
}

// Test ValidateDAG with complex cycle
func TestValidateDAGComplexCycle(t *testing.T) {
	tasks := []*parser.Task{
		createTestTask("task1", "Task 1", "Goal 1", []string{}),
		createTestTask("task2", "Task 2", "Goal 2", []string{"task1"}),
		createTestTask("task3", "Task 3", "Goal 3", []string{"task2"}),
		createTestTask("task4", "Task 4", "Goal 4", []string{"task3", "task1"}),
		createTestTask("task5", "Task 5", "Goal 5", []string{"task4"}),
		createTestTask("task1_cycle", "Task 1 Cycle", "Goal", []string{"task5"}), // Creates cycle back to task1
	}
	// Update task1 to depend on task1_cycle
	tasks[0].Dependencies = []string{"task1_cycle"}

	_, err := NewDAG(tasks)
	if err == nil {
		t.Error("Expected error for DAG with complex cycle")
	}
}

// Test ValidateDAG with non-existent dependency
func TestValidateDAGMissingDependency(t *testing.T) {
	dag := &DAG{
		Tasks: make(map[string]*parser.Task),
		Graph: make(map[string][]string),
	}

	task := createTestTask("task1", "Task 1", "Goal 1", []string{"task2"})
	dag.AddTask(task)

	err := dag.ValidateDAG()
	if err == nil {
		t.Error("Expected error for missing dependency")
	}
}

// Test GetTaskDependencies
func TestGetTaskDependencies(t *testing.T) {
	tasks := []*parser.Task{
		createTestTask("task1", "Task 1", "Goal 1", []string{}),
		createTestTask("task2", "Task 2", "Goal 2", []string{}),
		createTestTask("task3", "Task 3", "Goal 3", []string{"task1", "task2"}),
	}
	dag, err := NewDAG(tasks)
	if err != nil {
		t.Fatalf("NewDAG failed: %v", err)
	}

	deps, err := dag.GetTaskDependencies("task3")
	if err != nil {
		t.Fatalf("GetTaskDependencies failed: %v", err)
	}
	if len(deps) != 2 {
		t.Errorf("Expected 2 dependencies, got %d", len(deps))
	}
}

// Test GetTaskDependencies for non-existent task
func TestGetTaskDependenciesNotFound(t *testing.T) {
	dag := &DAG{
		Tasks: make(map[string]*parser.Task),
		Graph: make(map[string][]string),
	}

	_, err := dag.GetTaskDependencies("task1")
	if err == nil {
		t.Error("Expected error for non-existent task")
	}
}

// Test GetTaskDependents
func TestGetTaskDependents(t *testing.T) {
	tasks := []*parser.Task{
		createTestTask("task1", "Task 1", "Goal 1", []string{}),
		createTestTask("task2", "Task 2", "Goal 2", []string{"task1"}),
		createTestTask("task3", "Task 3", "Goal 3", []string{"task1"}),
	}
	dag, err := NewDAG(tasks)
	if err != nil {
		t.Fatalf("NewDAG failed: %v", err)
	}

	dependents, err := dag.GetTaskDependents("task1")
	if err != nil {
		t.Fatalf("GetTaskDependents failed: %v", err)
	}
	if len(dependents) != 2 {
		t.Errorf("Expected 2 dependents, got %d", len(dependents))
	}
}

// Test GetAllTasks
func TestGetAllTasks(t *testing.T) {
	tasks := []*parser.Task{
		createTestTask("task1", "Task 1", "Goal 1", []string{}),
		createTestTask("task2", "Task 2", "Goal 2", []string{"task1"}),
		createTestTask("task3", "Task 3", "Goal 3", []string{"task2"}),
	}
	dag, err := NewDAG(tasks)
	if err != nil {
		t.Fatalf("NewDAG failed: %v", err)
	}

	allTasks := dag.GetAllTasks()
	if len(allTasks) != 3 {
		t.Errorf("Expected 3 tasks, got %d", len(allTasks))
	}
}

// Test TaskCount
func TestTaskCount(t *testing.T) {
	tasks := []*parser.Task{
		createTestTask("task1", "Task 1", "Goal 1", []string{}),
		createTestTask("task2", "Task 2", "Goal 2", []string{"task1"}),
	}
	dag, err := NewDAG(tasks)
	if err != nil {
		t.Fatalf("NewDAG failed: %v", err)
	}

	if dag.TaskCount() != 2 {
		t.Errorf("Expected 2 tasks, got %d", dag.TaskCount())
	}
}

// Test complex DAG with multiple paths
func TestComplexDAGMultiplePaths(t *testing.T) {
	tasks := []*parser.Task{
		createTestTask("task1", "Task 1", "Goal 1", []string{}),
		createTestTask("task2", "Task 2", "Goal 2", []string{"task1"}),
		createTestTask("task3", "Task 3", "Goal 3", []string{"task1"}),
		createTestTask("task4", "Task 4", "Goal 4", []string{"task2", "task3"}),
		createTestTask("task5", "Task 5", "Goal 5", []string{"task4"}),
	}
	dag, err := NewDAG(tasks)
	if err != nil {
		t.Fatalf("NewDAG failed: %v", err)
	}

	if dag.TaskCount() != 5 {
		t.Errorf("Expected 5 tasks, got %d", dag.TaskCount())
	}

	// Verify task4 has 2 dependencies
	deps, err := dag.GetTaskDependencies("task4")
	if err != nil {
		t.Fatalf("GetTaskDependencies failed: %v", err)
	}
	if len(deps) != 2 {
		t.Errorf("Expected 2 dependencies for task4, got %d", len(deps))
	}

	// Verify task1 has 2 dependents
	dependents, err := dag.GetTaskDependents("task1")
	if err != nil {
		t.Fatalf("GetTaskDependents failed: %v", err)
	}
	if len(dependents) != 2 {
		t.Errorf("Expected 2 dependents for task1, got %d", len(dependents))
	}
}
