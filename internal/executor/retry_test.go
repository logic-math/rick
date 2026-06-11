package executor

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/sunquan/rick/internal/parser"
)

// TestRetryTaskNilTask tests handling of nil task
func TestRetryTaskNilTask(t *testing.T) {
	config := &ExecutionConfig{
		MaxRetries:     5,
		TimeoutSeconds: 30,
	}

	runner := NewTaskRunner(config, &mockAgentExecutor{})
	manager := NewTaskRetryManager(runner, config, "")

	result, err := manager.RetryTask(nil)
	if err == nil {
		t.Errorf("Expected error for nil task, got nil")
	}

	if result != nil {
		t.Errorf("Expected nil result for nil task, got %v", result)
	}
}

// TestRetryTaskNilConfig tests handling of nil config
func TestRetryTaskNilConfig(t *testing.T) {
	runner := NewTaskRunner(&ExecutionConfig{}, &mockAgentExecutor{})
	manager := &TaskRetryManager{
		runner: runner,
		config: nil,
	}

	task := &parser.Task{
		ID:         "task1",
		Name:       "Test Task",
		Goal:       "Test goal",
		TestMethod: "echo PASS",
	}

	result, err := manager.RetryTask(task)
	if err == nil {
		t.Errorf("Expected error for nil config, got nil")
	}

	if result != nil {
		t.Errorf("Expected nil result for nil config, got %v", result)
	}
}

// TestLoadDebugContext tests that loadDebugContext returns summaries from debug/ dir, not full text.
func TestLoadDebugContext(t *testing.T) {
	tmpDir := t.TempDir()
	debugFile := filepath.Join(tmpDir, "debug.md")

	config := &ExecutionConfig{MaxRetries: 3, TimeoutSeconds: 30}
	runner := NewTaskRunner(config, &mockAgentExecutor{})
	manager := NewTaskRetryManager(runner, config, debugFile)

	// No debug/ and no debug.md → empty
	ctx := manager.loadDebugContext(debugFile)
	if ctx != "" {
		t.Errorf("expected empty context when nothing exists, got: %s", ctx)
	}

	// Create debug/bug1-test.md with frontmatter
	debugDir := filepath.Join(tmpDir, "debug")
	if err := os.MkdirAll(debugDir, 0755); err != nil {
		t.Fatal(err)
	}
	bugBody := "---\nsummary: nil pointer in runner\nstatus: resolved\n---\n\nThis full body should NOT appear."
	if err := os.WriteFile(filepath.Join(debugDir, "bug1-test.md"), []byte(bugBody), 0644); err != nil {
		t.Fatal(err)
	}

	ctx = manager.loadDebugContext(debugFile)
	if !strings.Contains(ctx, "nil pointer in runner") {
		t.Errorf("expected summary in context, got: %s", ctx)
	}
	if strings.Contains(ctx, "full body should NOT appear") {
		t.Error("context should contain summary only, not full body text")
	}
}

// TestRetryResultDuration tests duration calculation
func TestRetryResultDuration(t *testing.T) {
	result := &RetryResult{
		TaskID:    "task1",
		StartTime: time.Now(),
		EndTime:   time.Now().Add(5 * time.Second),
	}

	duration := result.Duration()
	if duration < 4*time.Second || duration > 6*time.Second {
		t.Errorf("Expected duration around 5 seconds, got %v", duration)
	}
}

// TestNewTaskRetryManager tests manager creation
func TestNewTaskRetryManager(t *testing.T) {
	config := &ExecutionConfig{
		MaxRetries:     3,
		TimeoutSeconds: 30,
	}

	runner := NewTaskRunner(config, &mockAgentExecutor{})
	manager := NewTaskRetryManager(runner, config, "/tmp/debug.md")

	if manager.runner != runner {
		t.Errorf("Manager runner not set correctly")
	}

	if manager.config != config {
		t.Errorf("Manager config not set correctly")
	}

	if manager.debugFile != "/tmp/debug.md" {
		t.Errorf("Manager debug file not set correctly")
	}
}

// TestLoadDebugContext_EmptyPath tests loading with empty path
func TestLoadDebugContext_EmptyPath(t *testing.T) {
	config := &ExecutionConfig{MaxRetries: 3, TimeoutSeconds: 30}
	runner := NewTaskRunner(config, &mockAgentExecutor{})
	manager := NewTaskRetryManager(runner, config, "")

	context := manager.loadDebugContext("")
	if context != "" {
		t.Errorf("Expected empty context for empty path, got: %s", context)
	}
}

// TestRetryTaskSimple_NilTask tests RetryTaskSimple with nil task
func TestRetryTaskSimple_NilTask(t *testing.T) {
	if os.Getenv("RICK_INTEGRATION_TEST") == "" {
		t.Skip("skipping integration test: set RICK_INTEGRATION_TEST=1 to enable")
	}
	config := &ExecutionConfig{MaxRetries: 1, TimeoutSeconds: 5}
	runner := NewTaskRunner(config, &mockAgentExecutor{})
	_, err := RetryTaskSimple(nil, runner, config, "")
	if err == nil {
		t.Fatal("expected error for nil task")
	}
}

// TestRetryTaskSimple_Valid tests RetryTaskSimple with a valid task (requires claude)
func TestRetryTaskSimple_Valid(t *testing.T) {
	if os.Getenv("RICK_INTEGRATION_TEST") == "" {
		t.Skip("skipping integration test: set RICK_INTEGRATION_TEST=1 to enable")
	}
	config := &ExecutionConfig{MaxRetries: 1, TimeoutSeconds: 30}
	runner := NewTaskRunner(config, &mockAgentExecutor{})
	task := &parser.Task{
		ID:         "task1",
		Name:       "Test Task",
		Goal:       "Goal",
		TestMethod: "echo PASS",
	}
	result, err := RetryTaskSimple(task, runner, config, "")
	if err != nil {
		t.Logf("RetryTaskSimple error (acceptable): %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}
}

// TestRetryTask_WithMockClaude tests RetryTask with a mock claude binary
func TestRetryTask_WithMockClaude(t *testing.T) {
	// Create a mock claude script that exits with 0 but creates no test script
	mockScript := `#!/bin/sh
exit 0
`
	tmpDir := t.TempDir()
	mockPath := filepath.Join(tmpDir, "claude")
	if err := os.WriteFile(mockPath, []byte(mockScript), 0755); err != nil {
		t.Fatal(err)
	}

	config := &ExecutionConfig{
		MaxRetries:     1,
		TimeoutSeconds: 10,
		ClaudeCodePath: mockPath,
	}
	runner := NewTaskRunner(config, &mockAgentExecutor{})
	manager := NewTaskRetryManager(runner, config, "")

	task := &parser.Task{
		ID:         "task1",
		Name:       "Test Task",
		Goal:       "Goal",
		TestMethod: "echo PASS",
	}

	result, err := manager.RetryTask(task)
	// May fail because test script isn't created by mock, but should not panic
	if err != nil {
		t.Logf("RetryTask with mock returned error (acceptable): %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	// Result should have some status
	if result.TaskID != "task1" {
		t.Errorf("expected task_id=task1, got %s", result.TaskID)
	}
}

// TestRetryTaskSimple_WithMockClaude tests the convenience wrapper
func TestRetryTaskSimple_WithMockClaude(t *testing.T) {
	mockScript := "#!/bin/sh\nexit 0\n"
	tmpDir := t.TempDir()
	mockPath := filepath.Join(tmpDir, "claude")
	if err := os.WriteFile(mockPath, []byte(mockScript), 0755); err != nil {
		t.Fatal(err)
	}

	config := &ExecutionConfig{MaxRetries: 1, TimeoutSeconds: 10, ClaudeCodePath: mockPath}
	runner := NewTaskRunner(config, &mockAgentExecutor{})
	task := &parser.Task{ID: "t1", Name: "T", Goal: "G", TestMethod: "echo"}

	result, _ := RetryTaskSimple(task, runner, config, "")
	if result == nil {
		t.Fatal("expected non-nil result from RetryTaskSimple")
	}
	if result.TaskID != "t1" {
		t.Errorf("expected task_id=t1, got %s", result.TaskID)
	}
}
