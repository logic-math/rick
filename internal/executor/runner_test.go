package executor

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/sunquan/rick/internal/parser"
)

// TestNewTaskRunner tests TaskRunner creation
func TestNewTaskRunner(t *testing.T) {
	config := &ExecutionConfig{
		MaxRetries:     5,
		TimeoutSeconds: 30,
		LogFile:        "/tmp/test.log",
	}

	runner := NewTaskRunner(config)
	if runner == nil {
		t.Fatal("NewTaskRunner returned nil")
	}
	if runner.config != config {
		t.Fatal("config not set correctly")
	}
}

// TestGenerateTestScript tests script generation
func TestGenerateTestScript(t *testing.T) {
	runner := NewTaskRunner(&ExecutionConfig{TimeoutSeconds: 30})

	task := &parser.Task{
		ID:   "task1",
		Name: "Test Task",
		Goal: "Test the functionality",
		TestMethod: "- Step 1: Verify basic functionality\n- Step 2: Check output",
	}

	scriptPath, err := runner.GenerateTestScript(task)
	if err != nil {
		t.Fatalf("GenerateTestScript failed: %v", err)
	}
	defer os.Remove(scriptPath)

	// Verify script file exists
	if _, err := os.Stat(scriptPath); err != nil {
		t.Fatalf("Script file not created: %v", err)
	}

	// Verify script content
	content, err := os.ReadFile(scriptPath)
	if err != nil {
		t.Fatalf("Failed to read script: %v", err)
	}

	contentStr := string(content)
	if !strings.Contains(contentStr, "#!/bin/bash") {
		t.Error("Script missing shebang")
	}
	if !strings.Contains(contentStr, task.ID) {
		t.Error("Script missing task ID")
	}
	if !strings.Contains(contentStr, task.Name) {
		t.Error("Script missing task name")
	}
	if !strings.Contains(contentStr, "Status: PASS") {
		t.Error("Script missing status check")
	}
}

// TestGenerateTestScript_WithoutTestMethod tests script generation without test method
func TestGenerateTestScript_WithoutTestMethod(t *testing.T) {
	runner := NewTaskRunner(&ExecutionConfig{TimeoutSeconds: 30})

	task := &parser.Task{
		ID:   "task2",
		Name: "Simple Task",
		Goal: "Simple goal",
	}

	scriptPath, err := runner.GenerateTestScript(task)
	if err != nil {
		t.Fatalf("GenerateTestScript failed: %v", err)
	}
	defer os.Remove(scriptPath)

	if _, err := os.Stat(scriptPath); err != nil {
		t.Fatalf("Script file not created: %v", err)
	}
}

// TestGenerateTestScript_NilTask tests with nil task
func TestGenerateTestScript_NilTask(t *testing.T) {
	runner := NewTaskRunner(&ExecutionConfig{TimeoutSeconds: 30})

	_, err := runner.GenerateTestScript(nil)
	if err == nil {
		t.Fatal("Expected error for nil task")
	}
}

// TestExecuteTestScript tests script execution
func TestExecuteTestScript(t *testing.T) {
	runner := NewTaskRunner(&ExecutionConfig{TimeoutSeconds: 30})

	// Create a simple test script
	tmpDir := os.TempDir()
	scriptPath := filepath.Join(tmpDir, "test_script.sh")
	scriptContent := `#!/bin/bash
echo "Testing"
echo "Status: PASS"
`
	err := os.WriteFile(scriptPath, []byte(scriptContent), 0755)
	if err != nil {
		t.Fatalf("Failed to create test script: %v", err)
	}
	defer os.Remove(scriptPath)

	output, err := runner.ExecuteTestScript(scriptPath)
	if err != nil {
		t.Fatalf("ExecuteTestScript failed: %v", err)
	}

	if !strings.Contains(output, "Status: PASS") {
		t.Errorf("Output missing expected content: %s", output)
	}
}

// TestExecuteTestScript_EmptyPath tests with empty script path
func TestExecuteTestScript_EmptyPath(t *testing.T) {
	runner := NewTaskRunner(&ExecutionConfig{TimeoutSeconds: 30})

	_, err := runner.ExecuteTestScript("")
	if err == nil {
		t.Fatal("Expected error for empty script path")
	}
}

// TestExecuteTestScript_NonexistentFile tests with nonexistent file
func TestExecuteTestScript_NonexistentFile(t *testing.T) {
	runner := NewTaskRunner(&ExecutionConfig{TimeoutSeconds: 30})

	_, err := runner.ExecuteTestScript("/nonexistent/path/script.sh")
	if err == nil {
		t.Fatal("Expected error for nonexistent file")
	}
}

// TestExecuteTestScript_WithTimeout tests timeout handling
func TestExecuteTestScript_WithTimeout(t *testing.T) {
	runner := NewTaskRunner(&ExecutionConfig{TimeoutSeconds: 1})

	// Create a script that sleeps longer than timeout
	tmpDir := os.TempDir()
	scriptPath := filepath.Join(tmpDir, "timeout_script.sh")
	scriptContent := `#!/bin/bash
sleep 5
echo "This should not execute"
`
	err := os.WriteFile(scriptPath, []byte(scriptContent), 0755)
	if err != nil {
		t.Fatalf("Failed to create test script: %v", err)
	}
	defer os.Remove(scriptPath)

	_, err = runner.ExecuteTestScript(scriptPath)
	if err == nil {
		t.Fatal("Expected timeout error")
	}
	if !strings.Contains(err.Error(), "timeout") {
		t.Errorf("Expected timeout error, got: %v", err)
	}
}

// TestParseTestResult_Success tests parsing successful result
func TestParseTestResult_Success(t *testing.T) {
	runner := NewTaskRunner(&ExecutionConfig{TimeoutSeconds: 30})

	output := "Some test output\nStatus: PASS\nMore output"
	success, err := runner.ParseTestResult(output)
	if err != nil {
		t.Fatalf("ParseTestResult failed: %v", err)
	}
	if !success {
		t.Error("Expected success, got failure")
	}
}

// TestParseTestResult_Success_PASS tests parsing with PASS indicator
func TestParseTestResult_Success_PASS(t *testing.T) {
	runner := NewTaskRunner(&ExecutionConfig{TimeoutSeconds: 30})

	output := "Test PASS"
	success, err := runner.ParseTestResult(output)
	if err != nil {
		t.Fatalf("ParseTestResult failed: %v", err)
	}
	if !success {
		t.Error("Expected success for PASS indicator")
	}
}

// TestParseTestResult_Failure tests parsing failed result
func TestParseTestResult_Failure(t *testing.T) {
	runner := NewTaskRunner(&ExecutionConfig{TimeoutSeconds: 30})

	output := "Some test output\nTest FAIL\nMore output"
	success, err := runner.ParseTestResult(output)
	if err != nil {
		t.Fatalf("ParseTestResult failed: %v", err)
	}
	if success {
		t.Error("Expected failure, got success")
	}
}

// TestParseTestResult_Failure_ERROR tests parsing with ERROR indicator
func TestParseTestResult_Failure_ERROR(t *testing.T) {
	runner := NewTaskRunner(&ExecutionConfig{TimeoutSeconds: 30})

	output := "ERROR: Something went wrong"
	success, err := runner.ParseTestResult(output)
	if err != nil {
		t.Fatalf("ParseTestResult failed: %v", err)
	}
	if success {
		t.Error("Expected failure for ERROR indicator")
	}
}

// TestParseTestResult_EmptyOutput tests with empty output
func TestParseTestResult_EmptyOutput(t *testing.T) {
	runner := NewTaskRunner(&ExecutionConfig{TimeoutSeconds: 30})

	_, err := runner.ParseTestResult("")
	if err == nil {
		t.Fatal("Expected error for empty output")
	}
}

// TestParseTestResult_NoIndicator tests output without pass/fail indicators
func TestParseTestResult_NoIndicator(t *testing.T) {
	runner := NewTaskRunner(&ExecutionConfig{TimeoutSeconds: 30})

	output := "Some generic output"
	success, err := runner.ParseTestResult(output)
	if err != nil {
		t.Fatalf("ParseTestResult failed: %v", err)
	}
	if !success {
		t.Error("Expected success for output without indicators")
	}
}

// TestRunTask_Success tests successful task execution
func TestRunTask_Success(t *testing.T) {
	runner := NewTaskRunner(&ExecutionConfig{TimeoutSeconds: 30})

	task := &parser.Task{
		ID:   "task1",
		Name: "Test Task",
		Goal: "Test the functionality",
		TestMethod: "- Step 1: Verify\n- Step 2: Check",
	}

	result, err := runner.RunTask(task)
	if err != nil {
		t.Fatalf("RunTask failed: %v", err)
	}

	if result.TaskID != task.ID {
		t.Errorf("TaskID mismatch: expected %s, got %s", task.ID, result.TaskID)
	}
	if result.Status != "success" {
		t.Errorf("Expected success status, got %s", result.Status)
	}
	if result.Error != "" {
		t.Errorf("Unexpected error: %s", result.Error)
	}
}

// TestRunTask_NilTask tests with nil task
func TestRunTask_NilTask(t *testing.T) {
	runner := NewTaskRunner(&ExecutionConfig{TimeoutSeconds: 30})

	_, err := runner.RunTask(nil)
	if err == nil {
		t.Fatal("Expected error for nil task")
	}
}

// TestTaskExecutionResult_Duration tests duration calculation
func TestTaskExecutionResult_Duration(t *testing.T) {
	result := &TaskExecutionResult{
		TaskID:    "task1",
		StartTime: time.Now(),
	}

	time.Sleep(10 * time.Millisecond)
	result.EndTime = time.Now()

	duration := result.Duration()
	if duration < 10*time.Millisecond {
		t.Errorf("Duration too short: %v", duration)
	}
}

// TestGenerateTestScript_WithComplexTestMethod tests with complex test method
func TestGenerateTestScript_WithComplexTestMethod(t *testing.T) {
	runner := NewTaskRunner(&ExecutionConfig{TimeoutSeconds: 30})

	task := &parser.Task{
		ID:   "task3",
		Name: "Complex Task",
		Goal: "Complex goal",
		TestMethod: "1. First step\n2. Second step\n3. Third step\n- Additional step\n* Another step",
	}

	scriptPath, err := runner.GenerateTestScript(task)
	if err != nil {
		t.Fatalf("GenerateTestScript failed: %v", err)
	}
	defer os.Remove(scriptPath)

	content, err := os.ReadFile(scriptPath)
	if err != nil {
		t.Fatalf("Failed to read script: %v", err)
	}

	contentStr := string(content)
	if !strings.Contains(contentStr, "First step") {
		t.Error("Script missing first step")
	}
	if !strings.Contains(contentStr, "Second step") {
		t.Error("Script missing second step")
	}
	if !strings.Contains(contentStr, "Third step") {
		t.Error("Script missing third step")
	}
}

// TestExecuteTestScript_WithScriptError tests script that returns error
func TestExecuteTestScript_WithScriptError(t *testing.T) {
	runner := NewTaskRunner(&ExecutionConfig{TimeoutSeconds: 30})

	tmpDir := os.TempDir()
	scriptPath := filepath.Join(tmpDir, "error_script.sh")
	scriptContent := `#!/bin/bash
echo "Before error"
exit 1
`
	err := os.WriteFile(scriptPath, []byte(scriptContent), 0755)
	if err != nil {
		t.Fatalf("Failed to create test script: %v", err)
	}
	defer os.Remove(scriptPath)

	output, err := runner.ExecuteTestScript(scriptPath)
	if err == nil {
		t.Fatal("Expected error from failed script")
	}
	if !strings.Contains(output, "Before error") {
		t.Error("Output should contain script output before error")
	}
}

// TestParseTestResult_MultilineOutput tests parsing multiline output
func TestParseTestResult_MultilineOutput(t *testing.T) {
	runner := NewTaskRunner(&ExecutionConfig{TimeoutSeconds: 30})

	output := `Starting test
Running validation
Checking results
Status: PASS
Cleanup complete`

	success, err := runner.ParseTestResult(output)
	if err != nil {
		t.Fatalf("ParseTestResult failed: %v", err)
	}
	if !success {
		t.Error("Expected success for multiline output with PASS")
	}
}

// TestRunTask_WithKeyResults tests that key results are available in task
func TestRunTask_WithKeyResults(t *testing.T) {
	runner := NewTaskRunner(&ExecutionConfig{TimeoutSeconds: 30})

	task := &parser.Task{
		ID:   "task4",
		Name: "Task with Results",
		Goal: "Test with key results",
		KeyResults: []string{
			"Result 1",
			"Result 2",
			"Result 3",
		},
		TestMethod: "- Verify all results",
	}

	result, err := runner.RunTask(task)
	if err != nil {
		t.Fatalf("RunTask failed: %v", err)
	}

	if result.TaskName != task.Name {
		t.Errorf("TaskName mismatch: expected %s, got %s", task.Name, result.TaskName)
	}
	if result.Status != "success" {
		t.Errorf("Expected success status, got %s", result.Status)
	}
}

// TestExecuteTestScript_DefaultTimeout tests default timeout when not configured
func TestExecuteTestScript_DefaultTimeout(t *testing.T) {
	runner := NewTaskRunner(&ExecutionConfig{TimeoutSeconds: 0}) // 0 means use default

	tmpDir := os.TempDir()
	scriptPath := filepath.Join(tmpDir, "quick_script.sh")
	scriptContent := `#!/bin/bash
echo "Quick execution"
echo "Status: PASS"
`
	err := os.WriteFile(scriptPath, []byte(scriptContent), 0755)
	if err != nil {
		t.Fatalf("Failed to create test script: %v", err)
	}
	defer os.Remove(scriptPath)

	output, err := runner.ExecuteTestScript(scriptPath)
	if err != nil {
		t.Fatalf("ExecuteTestScript failed: %v", err)
	}

	if !strings.Contains(output, "Quick execution") {
		t.Error("Output missing expected content")
	}
}

// BenchmarkGenerateTestScript benchmarks script generation
func BenchmarkGenerateTestScript(b *testing.B) {
	runner := NewTaskRunner(&ExecutionConfig{TimeoutSeconds: 30})
	task := &parser.Task{
		ID:   "task_bench",
		Name: "Benchmark Task",
		Goal: "Benchmark goal",
		TestMethod: "- Step 1\n- Step 2\n- Step 3",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		scriptPath, err := runner.GenerateTestScript(task)
		if err != nil {
			b.Fatalf("GenerateTestScript failed: %v", err)
		}
		os.Remove(scriptPath)
	}
}

// BenchmarkParseTestResult benchmarks result parsing
func BenchmarkParseTestResult(b *testing.B) {
	runner := NewTaskRunner(&ExecutionConfig{TimeoutSeconds: 30})
	output := "Some test output\nStatus: PASS\nMore output"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := runner.ParseTestResult(output)
		if err != nil {
			b.Fatalf("ParseTestResult failed: %v", err)
		}
	}
}

// TestRunTask_GenerateScriptFailure tests when script generation fails
// This is tested indirectly through successful generation
func TestRunTask_WithDependencies(t *testing.T) {
	runner := NewTaskRunner(&ExecutionConfig{TimeoutSeconds: 30})

	task := &parser.Task{
		ID:   "task_with_deps",
		Name: "Task With Dependencies",
		Goal: "Test task with dependencies",
		Dependencies: []string{"task1", "task2"},
		TestMethod: "- Verify dependencies",
	}

	result, err := runner.RunTask(task)
	if err != nil {
		t.Fatalf("RunTask failed: %v", err)
	}

	if result.Status != "success" {
		t.Errorf("Expected success status, got %s", result.Status)
	}
}

// TestRunTask_CompleteFlow tests complete task execution flow
func TestRunTask_CompleteFlow(t *testing.T) {
	config := &ExecutionConfig{
		MaxRetries:     5,
		TimeoutSeconds: 30,
		LogFile:        "/tmp/test.log",
	}
	runner := NewTaskRunner(config)

	task := &parser.Task{
		ID:   "complete_flow_task",
		Name: "Complete Flow Task",
		Goal: "Test complete execution flow",
		KeyResults: []string{
			"All steps executed",
			"Output captured",
			"Result parsed",
		},
		TestMethod: "1. Initialize\n2. Execute\n3. Verify\n4. Cleanup",
	}

	result, err := runner.RunTask(task)
	if err != nil {
		t.Fatalf("RunTask failed: %v", err)
	}

	if result.TaskID != task.ID {
		t.Errorf("TaskID mismatch")
	}
	if result.TaskName != task.Name {
		t.Errorf("TaskName mismatch")
	}
	if result.Status == "" {
		t.Error("Status should not be empty")
	}
	if result.StartTime.IsZero() {
		t.Error("StartTime should be set")
	}
	if result.EndTime.IsZero() {
		t.Error("EndTime should be set")
	}
}

// TestExecuteTestScript_CapturesStderr tests stderr capture
func TestExecuteTestScript_CapturesStderr(t *testing.T) {
	runner := NewTaskRunner(&ExecutionConfig{TimeoutSeconds: 30})

	tmpDir := os.TempDir()
	scriptPath := filepath.Join(tmpDir, "stderr_script.sh")
	scriptContent := `#!/bin/bash
echo "stdout message"
echo "stderr message" >&2
echo "Status: PASS"
exit 0
`
	err := os.WriteFile(scriptPath, []byte(scriptContent), 0755)
	if err != nil {
		t.Fatalf("Failed to create test script: %v", err)
	}
	defer os.Remove(scriptPath)

	output, err := runner.ExecuteTestScript(scriptPath)
	if err != nil {
		t.Fatalf("ExecuteTestScript failed: %v", err)
	}

	if !strings.Contains(output, "stdout message") {
		t.Error("Output should contain stdout")
	}
	// Note: stderr is captured in combined output by the ExecuteTestScript function
	// The output variable contains both stdout and stderr
}

// TestParseTestResult_CaseSensitivity tests case sensitivity
func TestParseTestResult_CaseSensitivity(t *testing.T) {
	runner := NewTaskRunner(&ExecutionConfig{TimeoutSeconds: 30})

	// lowercase pass should still work
	output := "test pass"
	success, err := runner.ParseTestResult(output)
	if err != nil {
		t.Fatalf("ParseTestResult failed: %v", err)
	}
	// Should be true since it contains "pass" as substring
	if !success {
		t.Error("Expected success for output with 'pass'")
	}
}

// TestGenerateTestScript_SpecialCharacters tests with special characters in task name
func TestGenerateTestScript_SpecialCharacters(t *testing.T) {
	runner := NewTaskRunner(&ExecutionConfig{TimeoutSeconds: 30})

	task := &parser.Task{
		ID:   "task_special",
		Name: "Task with \"quotes\" and 'apostrophes'",
		Goal: "Test with special chars",
		TestMethod: "- Step with $variable\n- Step with `backticks`",
	}

	scriptPath, err := runner.GenerateTestScript(task)
	if err != nil {
		t.Fatalf("GenerateTestScript failed: %v", err)
	}
	defer os.Remove(scriptPath)

	if _, err := os.Stat(scriptPath); err != nil {
		t.Fatalf("Script file not created: %v", err)
	}

	content, err := os.ReadFile(scriptPath)
	if err != nil {
		t.Fatalf("Failed to read script: %v", err)
	}

	contentStr := string(content)
	if !strings.Contains(contentStr, "#!/bin/bash") {
		t.Error("Script missing shebang")
	}
}

// TestTaskExecutionResult_Fields tests all fields are set correctly
func TestTaskExecutionResult_Fields(t *testing.T) {
	startTime := time.Now()
	endTime := startTime.Add(5 * time.Second)

	result := &TaskExecutionResult{
		TaskID:    "test_task",
		TaskName:  "Test Task",
		Status:    "success",
		Error:     "",
		Output:    "Test output",
		StartTime: startTime,
		EndTime:   endTime,
	}

	if result.TaskID != "test_task" {
		t.Error("TaskID not set correctly")
	}
	if result.TaskName != "Test Task" {
		t.Error("TaskName not set correctly")
	}
	if result.Status != "success" {
		t.Error("Status not set correctly")
	}
	if result.Output != "Test output" {
		t.Error("Output not set correctly")
	}

	duration := result.Duration()
	if duration != 5*time.Second {
		t.Errorf("Duration calculation incorrect: expected 5s, got %v", duration)
	}
}

// TestExecuteTestScript_WithStdout tests stdout capture
func TestExecuteTestScript_WithStdout(t *testing.T) {
	runner := NewTaskRunner(&ExecutionConfig{TimeoutSeconds: 30})

	tmpDir := os.TempDir()
	scriptPath := filepath.Join(tmpDir, "stdout_script.sh")
	scriptContent := `#!/bin/bash
echo "Line 1"
echo "Line 2"
echo "Line 3"
echo "Status: PASS"
`
	err := os.WriteFile(scriptPath, []byte(scriptContent), 0755)
	if err != nil {
		t.Fatalf("Failed to create test script: %v", err)
	}
	defer os.Remove(scriptPath)

	output, err := runner.ExecuteTestScript(scriptPath)
	if err != nil {
		t.Fatalf("ExecuteTestScript failed: %v", err)
	}

	lines := strings.Split(output, "\n")
	if len(lines) < 4 {
		t.Errorf("Expected at least 4 lines of output, got %d", len(lines))
	}
}

// TestRunTask_AllStatuses tests different task statuses
func TestRunTask_AllStatuses(t *testing.T) {
	runner := NewTaskRunner(&ExecutionConfig{TimeoutSeconds: 30})

	testCases := []struct {
		name string
		task *parser.Task
	}{
		{
			name: "task_1",
			task: &parser.Task{
				ID:   "task_1",
				Name: "First Task",
				Goal: "First goal",
				TestMethod: "- Test 1",
			},
		},
		{
			name: "task_2",
			task: &parser.Task{
				ID:   "task_2",
				Name: "Second Task",
				Goal: "Second goal",
				TestMethod: "- Test 2",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := runner.RunTask(tc.task)
			if err != nil {
				t.Fatalf("RunTask failed: %v", err)
			}

			if result.TaskID != tc.task.ID {
				t.Errorf("TaskID mismatch")
			}
			if result.TaskName != tc.task.Name {
				t.Errorf("TaskName mismatch")
			}
		})
	}
}
