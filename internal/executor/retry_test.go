package executor

import (
	"os"
	"path/filepath"
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

	runner := NewTaskRunner(config)
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
	runner := NewTaskRunner(&ExecutionConfig{})
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

// TestLoadDebugContext tests loading debug context
func TestLoadDebugContext(t *testing.T) {
	tmpDir := t.TempDir()
	debugFile := filepath.Join(tmpDir, "debug.md")

	config := &ExecutionConfig{
		MaxRetries:     3,
		TimeoutSeconds: 30,
	}

	runner := NewTaskRunner(config)
	manager := NewTaskRetryManager(runner, config, debugFile)

	// Test loading non-existent file
	context := manager.loadDebugContext(debugFile)
	if context != "" {
		t.Errorf("Expected empty context for non-existent file, got: %s", context)
	}

	// Create debug file with content
	testContent := "- debug1: test entry"
	err := os.WriteFile(debugFile, []byte(testContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Test loading existing file
	context = manager.loadDebugContext(debugFile)
	if context != testContent {
		t.Errorf("Expected '%s', got '%s'", testContent, context)
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

	runner := NewTaskRunner(config)
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
	runner := NewTaskRunner(config)
	manager := NewTaskRetryManager(runner, config, "")

	context := manager.loadDebugContext("")
	if context != "" {
		t.Errorf("Expected empty context for empty path, got: %s", context)
	}
}
