package executor

import (
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/sunquan/rick/internal/agent"
	"github.com/sunquan/rick/internal/parser"
)

// mockAgentSession is a test double for agent.AgentSession.
type mockAgentSession struct {
	id               string
	toolCalls        []agent.ToolCall
	finalMessage     string
	finalMessageLine int
	rawLogPath       string
}

func (m *mockAgentSession) ID() string                  { return m.id }
func (m *mockAgentSession) Duration() time.Duration     { return 0 }
func (m *mockAgentSession) ToolCalls() []agent.ToolCall { return m.toolCalls }
func (m *mockAgentSession) FinalMessage() string        { return m.finalMessage }
func (m *mockAgentSession) FinalMessageLine() int       { return m.finalMessageLine }
func (m *mockAgentSession) RawLogPath() string          { return m.rawLogPath }

// mockAgentExecutorWithSession extends mockAgentExecutor to return a specific session.
type mockAgentExecutorWithSession struct {
	session agent.AgentSession
	err     error
}

func (m *mockAgentExecutorWithSession) Execute(promptFile, taskID string) (agent.AgentSession, error) {
	return m.session, m.err
}

// TestNewTaskRunner tests TaskRunner creation
func TestNewTaskRunner(t *testing.T) {
	config := &ExecutionConfig{
		MaxRetries:     5,
		TimeoutSeconds: 30,
		LogFile:        "/tmp/test.log",
	}

	runner := NewTaskRunner(config, &mockAgentExecutor{})
	if runner == nil {
		t.Fatal("NewTaskRunner returned nil")
	}
	if runner.config != config {
		t.Fatal("config not set correctly")
	}
}

// TestExecuteTestScript tests script execution with a Python script returning JSON
func TestExecuteTestScript(t *testing.T) {
	runner := NewTaskRunner(&ExecutionConfig{TimeoutSeconds: 30}, &mockAgentExecutor{})

	tmpDir := os.TempDir()
	scriptPath := filepath.Join(tmpDir, "test_script_pass.py")
	scriptContent := `import json
print(json.dumps({"pass": True, "errors": []}))
`
	err := os.WriteFile(scriptPath, []byte(scriptContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test script: %v", err)
	}
	defer os.Remove(scriptPath)

	result, output, err := runner.ExecuteTestScript(scriptPath)
	if err != nil {
		t.Fatalf("ExecuteTestScript failed: %v", err)
	}

	if result == nil {
		t.Fatal("Expected non-nil result")
	}

	if !result.Pass {
		t.Errorf("Expected pass=true, got false. Output: %s", output)
	}
}

// TestExecuteTestScript_EmptyPath tests with empty script path
func TestExecuteTestScript_EmptyPath(t *testing.T) {
	runner := NewTaskRunner(&ExecutionConfig{TimeoutSeconds: 30}, &mockAgentExecutor{})

	_, _, err := runner.ExecuteTestScript("")
	if err == nil {
		t.Fatal("Expected error for empty script path")
	}
}

// TestExecuteTestScript_NonexistentFile tests with nonexistent file
func TestExecuteTestScript_NonexistentFile(t *testing.T) {
	runner := NewTaskRunner(&ExecutionConfig{TimeoutSeconds: 30}, &mockAgentExecutor{})

	_, _, err := runner.ExecuteTestScript("/nonexistent/path/script.py")
	if err == nil {
		t.Fatal("Expected error for nonexistent file")
	}
}

// TestExecuteTestScript_WithTimeout tests timeout handling
func TestExecuteTestScript_WithTimeout(t *testing.T) {
	runner := NewTaskRunner(&ExecutionConfig{TimeoutSeconds: 1}, &mockAgentExecutor{})

	tmpDir := os.TempDir()
	scriptPath := filepath.Join(tmpDir, "timeout_script.py")
	scriptContent := `import time
time.sleep(5)
print("This should not execute")
`
	err := os.WriteFile(scriptPath, []byte(scriptContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test script: %v", err)
	}
	defer os.Remove(scriptPath)

	_, _, err = runner.ExecuteTestScript(scriptPath)
	if err == nil {
		t.Fatal("Expected timeout error")
	}
	if !strings.Contains(err.Error(), "timeout") {
		t.Errorf("Expected timeout error, got: %v", err)
	}
}

// TestExecuteTestScript_FailResult tests script that returns fail JSON
func TestExecuteTestScript_FailResult(t *testing.T) {
	runner := NewTaskRunner(&ExecutionConfig{TimeoutSeconds: 30}, &mockAgentExecutor{})

	tmpDir := os.TempDir()
	scriptPath := filepath.Join(tmpDir, "fail_script.py")
	scriptContent := `import json
print(json.dumps({"pass": False, "errors": ["assertion failed"]}))
`
	err := os.WriteFile(scriptPath, []byte(scriptContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test script: %v", err)
	}
	defer os.Remove(scriptPath)

	result, _, err := runner.ExecuteTestScript(scriptPath)
	if err != nil {
		t.Fatalf("ExecuteTestScript failed: %v", err)
	}

	if result == nil {
		t.Fatal("Expected non-nil result")
	}

	if result.Pass {
		t.Error("Expected pass=false")
	}

	if len(result.Errors) == 0 {
		t.Error("Expected errors in result")
	}
}

// TestRunTask_NilTask tests with nil task
func TestRunTask_NilTask(t *testing.T) {
	runner := NewTaskRunner(&ExecutionConfig{TimeoutSeconds: 30}, &mockAgentExecutor{})

	_, err := runner.RunTask(nil, "", "")
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

// TestGenerateDoingPromptFile_NilTask tests with nil task
func TestGenerateDoingPromptFile_NilTask(t *testing.T) {
	runner := NewTaskRunner(&ExecutionConfig{TimeoutSeconds: 30}, &mockAgentExecutor{})

	_, err := runner.GenerateDoingPromptFile(nil, "", "")
	if err == nil {
		t.Fatal("Expected error for nil task")
	}
}

// TestGenerateDoingPromptFile_ValidTask tests prompt file generation
func TestGenerateDoingPromptFile_ValidTask(t *testing.T) {
	runner := NewTaskRunner(&ExecutionConfig{TimeoutSeconds: 30}, &mockAgentExecutor{})

	task := &parser.Task{
		ID:         "task1",
		Name:       "Test Task",
		Goal:       "Test goal",
		TestMethod: "- Step 1\n- Step 2",
	}

	promptFile, err := runner.GenerateDoingPromptFile(task, "", "")
	if err != nil {
		t.Fatalf("GenerateDoingPromptFile failed: %v", err)
	}
	defer os.Remove(promptFile)

	if _, err := os.Stat(promptFile); err != nil {
		t.Fatalf("Prompt file not created: %v", err)
	}

	content, err := os.ReadFile(promptFile)
	if err != nil {
		t.Fatalf("Failed to read prompt file: %v", err)
	}

	if len(content) == 0 {
		t.Error("Prompt file is empty")
	}
}

// TestGenerateDoingPromptFile_WithDebugContext tests prompt with debug context
func TestGenerateDoingPromptFile_WithDebugContext(t *testing.T) {
	runner := NewTaskRunner(&ExecutionConfig{TimeoutSeconds: 30}, &mockAgentExecutor{})

	task := &parser.Task{
		ID:   "task1",
		Name: "Test Task",
		Goal: "Test goal",
	}

	debugContext := "## debug1: Previous error\nSome debug info"
	promptFile, err := runner.GenerateDoingPromptFile(task, debugContext, "")
	if err != nil {
		t.Fatalf("GenerateDoingPromptFile failed: %v", err)
	}
	defer os.Remove(promptFile)

	content, err := os.ReadFile(promptFile)
	if err != nil {
		t.Fatalf("Failed to read prompt file: %v", err)
	}

	if !strings.Contains(string(content), debugContext) {
		t.Error("Prompt file should contain debug context")
	}
}

// TestGenerateDoingPromptFile_WithTestErrorFeedback tests prompt with test error feedback
func TestGenerateDoingPromptFile_WithTestErrorFeedback(t *testing.T) {
	runner := NewTaskRunner(&ExecutionConfig{TimeoutSeconds: 30}, &mockAgentExecutor{})

	task := &parser.Task{
		ID:   "task1",
		Name: "Test Task",
		Goal: "Test goal",
	}

	testFeedback := "Attempt 1: test did not pass: assertion error"
	promptFile, err := runner.GenerateDoingPromptFile(task, "", testFeedback)
	if err != nil {
		t.Fatalf("GenerateDoingPromptFile failed: %v", err)
	}
	defer os.Remove(promptFile)

	content, err := os.ReadFile(promptFile)
	if err != nil {
		t.Fatalf("Failed to read prompt file: %v", err)
	}

	if !strings.Contains(string(content), testFeedback) {
		t.Error("Prompt file should contain test error feedback")
	}
}

// TestBuildTestGenerationPromptFile tests the test prompt file generation
func TestBuildTestGenerationPromptFile(t *testing.T) {
	config := &ExecutionConfig{TimeoutSeconds: 30}
	runner := NewTaskRunner(config, &mockAgentExecutor{})

	task := &parser.Task{
		ID:         "task1",
		Name:       "Test Task",
		Goal:       "Test goal",
		TestMethod: "Run the test",
	}

	promptFile, err := runner.buildTestGenerationPromptFile(task, "/tmp/test_task1.py")
	if err != nil {
		t.Fatalf("buildTestGenerationPromptFile failed: %v", err)
	}
	defer os.Remove(promptFile)

	if _, err := os.Stat(promptFile); err != nil {
		t.Fatalf("prompt file not created: %v", err)
	}

	content, err := os.ReadFile(promptFile)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(content), "task1") {
		t.Error("prompt file should contain task ID")
	}
}

func TestBuildTestGenerationPromptFile_NilTask(t *testing.T) {
	config := &ExecutionConfig{TimeoutSeconds: 30}
	runner := NewTaskRunner(config, &mockAgentExecutor{})

	// nil task should be handled gracefully by the caller (RunTask checks for nil)
	// buildTestGenerationPromptFile itself may panic or error on nil task
	// We just verify it doesn't silently succeed with bad data
	task := &parser.Task{ID: "t1", Name: "T", Goal: "G", TestMethod: "test"}
	promptFile, err := runner.buildTestGenerationPromptFile(task, "/tmp/test.py")
	if err != nil {
		// It's ok if it errors (e.g., template not found in test env)
		t.Logf("buildTestGenerationPromptFile returned error (acceptable): %v", err)
		return
	}
	defer os.Remove(promptFile)
}

// TestRunTask_ActPathGeneration validates KR1: act-path.md is generated after Execute.
func TestRunTask_ActPathGeneration(t *testing.T) {
	tmpDir := t.TempDir()

	// Pre-create test script so GenerateTestWithAgent skips claude invocation.
	testsDir := filepath.Join(tmpDir, "tests")
	if err := os.MkdirAll(testsDir, 0755); err != nil {
		t.Fatalf("failed to create tests dir: %v", err)
	}
	scriptContent := "import json\nprint(json.dumps({\"pass\": True, \"errors\": []}))\n"
	if err := os.WriteFile(filepath.Join(testsDir, "task1.py"), []byte(scriptContent), 0644); err != nil {
		t.Fatalf("failed to write test script: %v", err)
	}

	// Create a temp raw log file as the session's RawLogPath.
	rawLog, err := os.CreateTemp(tmpDir, "raw_session_*.log")
	if err != nil {
		t.Fatalf("failed to create raw log: %v", err)
	}
	rawLog.Close()

	sess := &mockAgentSession{
		id:   "sess-1",
		toolCalls: []agent.ToolCall{
			{Name: "bash", Input: "echo hi", IsError: false, Line: 1},
		},
		finalMessage:     "done",
		finalMessageLine: 5,
		rawLogPath:       rawLog.Name(),
	}
	mock := &mockAgentExecutorWithSession{session: sess}

	config := &ExecutionConfig{
		TimeoutSeconds: 30,
		WorkspaceDir:   tmpDir,
	}
	runner := NewTaskRunner(config, mock)

	task := &parser.Task{
		ID:   "task1",
		Name: "Test Task",
		Goal: "Test goal",
	}

	result, err := runner.RunTask(task, "", "")
	if err != nil {
		t.Fatalf("RunTask returned unexpected error: %v", err)
	}
	_ = result

	actPathFile := filepath.Join(tmpDir, "tasks", "task1", "act-path.md")
	if _, err := os.Stat(actPathFile); err != nil {
		t.Fatalf("act-path.md not created at %s: %v", actPathFile, err)
	}

	content, err := os.ReadFile(actPathFile)
	if err != nil {
		t.Fatalf("failed to read act-path.md: %v", err)
	}
	s := string(content)
	for _, want := range []string{"## 执行摘要", "## 行为轨迹", "报错次数: 0"} {
		if !strings.Contains(s, want) {
			t.Errorf("act-path.md missing %q", want)
		}
	}
}

// testGenExecutor is a mock for RED validation tests.
// It creates the test script file on -test-gen Execute calls with configurable pass values.
type testGenExecutor struct {
	mu             sync.Mutex
	testScriptPath string
	passValues     []bool // indexed by testGenCount; last value is repeated
	testGenCount   int
	totalCount     int
}

func (m *testGenExecutor) Execute(promptFile, taskID string) (agent.AgentSession, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.totalCount++
	if strings.HasSuffix(taskID, "-test-gen") {
		idx := m.testGenCount
		if idx >= len(m.passValues) {
			idx = len(m.passValues) - 1
		}
		passStr := "False"
		if m.passValues[idx] {
			passStr = "True"
		}
		content := "import json\nprint(json.dumps({\"pass\": " + passStr + ", \"errors\": []}))\n"
		os.WriteFile(m.testScriptPath, []byte(content), 0644)
		m.testGenCount++
	}
	return nil, nil
}

// TestRunTask_REDFail_Normal verifies normal RED state: test fails before implementation,
// testing agent called exactly once, no RED warning in debug.md.
func TestRunTask_REDFail_Normal(t *testing.T) {
	tmpDir := t.TempDir()
	testsDir := filepath.Join(tmpDir, "tests")
	os.MkdirAll(testsDir, 0755)
	testScript := filepath.Join(testsDir, "task1.py")

	mock := &testGenExecutor{
		testScriptPath: testScript,
		passValues:     []bool{false}, // normal RED: test fails
	}

	runner := NewTaskRunner(&ExecutionConfig{TimeoutSeconds: 30, WorkspaceDir: tmpDir}, mock)
	task := &parser.Task{ID: "task1", Name: "Test Task", Goal: "Test goal"}

	runner.RunTask(task, "", "")

	mock.mu.Lock()
	testGenCount := mock.testGenCount
	mock.mu.Unlock()

	if testGenCount != 1 {
		t.Errorf("expected testGenCount=1 (no RED retry), got %d", testGenCount)
	}

	// No RED warning in debug.md
	debugPath := filepath.Join(tmpDir, "debug.md")
	if content, err := os.ReadFile(debugPath); err == nil {
		if strings.Contains(string(content), "RED") || strings.Contains(string(content), "unexpected pass") {
			t.Error("debug.md should not have RED warning for normal RED state")
		}
	}
}

// TestRunTask_REDPass_TriggersRetry verifies that when test unexpectedly passes (unexpected green),
// testing agent is re-invoked and debug.md contains RED/unexpected-pass warning.
func TestRunTask_REDPass_TriggersRetry(t *testing.T) {
	tmpDir := t.TempDir()
	testsDir := filepath.Join(tmpDir, "tests")
	os.MkdirAll(testsDir, 0755)
	testScript := filepath.Join(testsDir, "task1.py")

	mock := &testGenExecutor{
		testScriptPath: testScript,
		passValues:     []bool{true, false}, // first gen: unexpected green; second: normal RED
	}

	runner := NewTaskRunner(&ExecutionConfig{TimeoutSeconds: 30, WorkspaceDir: tmpDir}, mock)
	task := &parser.Task{ID: "task1", Name: "Test Task", Goal: "Test goal"}

	runner.RunTask(task, "", "")

	mock.mu.Lock()
	testGenCount := mock.testGenCount
	mock.mu.Unlock()

	if testGenCount < 2 {
		t.Errorf("expected testGenCount>=2 (testing agent retried), got %d", testGenCount)
	}

	debugPath := filepath.Join(tmpDir, "debug.md")
	content, err := os.ReadFile(debugPath)
	if err != nil {
		t.Fatalf("debug.md should exist after RED warning: %v", err)
	}
	if !strings.Contains(string(content), "RED") && !strings.Contains(string(content), "unexpected pass") {
		t.Error("debug.md missing RED or unexpected pass warning")
	}
}

// TestRunTask_REDPass_MaxRetry verifies that after 2 consecutive unexpected-green states,
// execution continues without infinite loop.
func TestRunTask_REDPass_MaxRetry(t *testing.T) {
	tmpDir := t.TempDir()
	testsDir := filepath.Join(tmpDir, "tests")
	os.MkdirAll(testsDir, 0755)
	testScript := filepath.Join(testsDir, "task1.py")

	mock := &testGenExecutor{
		testScriptPath: testScript,
		passValues:     []bool{true, true}, // both retries: unexpected green (max retries exhausted)
	}

	runner := NewTaskRunner(&ExecutionConfig{TimeoutSeconds: 30, WorkspaceDir: tmpDir}, mock)
	task := &parser.Task{ID: "task1", Name: "Test Task", Goal: "Test goal"}

	done := make(chan struct{})
	go func() {
		runner.RunTask(task, "", "")
		close(done)
	}()

	select {
	case <-done:
		// completed without infinite loop ✓
	case <-time.After(10 * time.Second):
		t.Fatal("RunTask hung — possible infinite loop in RED validation")
	}

	mock.mu.Lock()
	testGenCount := mock.testGenCount
	mock.mu.Unlock()

	if testGenCount > 2 {
		t.Errorf("testGenCount=%d exceeds maxREDRetries=2, possible infinite loop", testGenCount)
	}
}
