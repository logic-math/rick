package executor

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/sunquan/rick/internal/parser"
	"github.com/sunquan/rick/internal/prompt"
)

// ExecutionConfig holds the configuration for task execution
type ExecutionConfig struct {
	MaxRetries      int
	TimeoutSeconds  int
	LogFile         string
	ClaudeCodePath  string
	WorkspaceDir    string
}

// TaskRunner manages the execution of individual tasks
type TaskRunner struct {
	config *ExecutionConfig
}

// NewTaskRunner creates a new TaskRunner instance
func NewTaskRunner(config *ExecutionConfig) *TaskRunner {
	return &TaskRunner{
		config: config,
	}
}

// TestResult represents the JSON result from a test script
type TestResult struct {
	Pass   bool     `json:"pass"`
	Errors []string `json:"errors"`
}

// RunTask executes a single task following the new workflow:
// 1. Generate test script using Agent (test generation phase)
// 2. Execute task -> run test
// Parameters:
//   - task: The task to execute
//   - debugContext: Content from debug.md (managed by Claude)
//   - testErrorFeedback: Previous test execution errors (for test script correction)
func (tr *TaskRunner) RunTask(task *parser.Task, debugContext string, testErrorFeedback string) (*TaskExecutionResult, error) {
	if task == nil {
		return nil, fmt.Errorf("task cannot be nil")
	}

	result := &TaskExecutionResult{
		TaskID:    task.ID,
		TaskName:  task.Name,
		Status:    "running",
		StartTime: time.Now(),
	}

	// Step 1: Generate test script using Agent (test generation phase)
	testScriptPath, err := tr.GenerateTestWithAgent(task)
	if err != nil {
		result.Status = "failed"
		result.Error = fmt.Sprintf("failed to generate test script: %v", err)
		result.EndTime = time.Now()
		return result, nil
	}
	// Keep test script for audit purposes (do not delete)

	// Step 2: Execute task with context (debug.md + test error feedback)
	var lastOutput string

	// Execute once and test
	doingPromptFile, err := tr.GenerateDoingPromptFile(task, debugContext, testErrorFeedback)
	if err != nil {
		result.Status = "failed"
		result.Error = fmt.Sprintf("failed to generate doing prompt: %v", err)
		result.EndTime = time.Now()
		return result, nil
	}
	defer os.Remove(doingPromptFile) // Clean up temporary file

	claudeOutput, err := tr.CallClaudeCodeCLI(doingPromptFile)
	if err != nil {
		result.Status = "failed"
		result.Error = fmt.Sprintf("Claude Code CLI failed: %v", err)
		result.Output = claudeOutput
		result.EndTime = time.Now()
		return result, nil
	}

	lastOutput = claudeOutput

	// Run test to validate
	testResult, testOutput, err := tr.ExecuteTestScript(testScriptPath)
	if err != nil {
		result.Status = "failed"
		result.Error = fmt.Sprintf("test execution failed: %v\n\nFull test output:\n%s", err, testOutput)
		result.Output = fmt.Sprintf("Claude output:\n%s\n\nTest output:\n%s", claudeOutput, testOutput)
		result.EndTime = time.Now()
		return result, nil
	}

	// Check if test passed
	if testResult.Pass {
		result.Status = "success"
		result.Output = fmt.Sprintf("Claude output:\n%s\n\nTest output:\n%s", lastOutput, testOutput)
	} else {
		result.Status = "failed"
		result.Error = fmt.Sprintf("test did not pass: %s\n\nFull test output:\n%s", strings.Join(testResult.Errors, "; "), testOutput)
		result.Output = fmt.Sprintf("Claude output:\n%s\n\nTest output:\n%s", lastOutput, testOutput)
	}

	result.EndTime = time.Now()
	return result, nil
}

// GenerateTestWithAgent generates a Python test script using Claude Agent
// This is the "test generation phase" in the workflow
func (tr *TaskRunner) GenerateTestWithAgent(task *parser.Task) (string, error) {
	if task == nil {
		return "", fmt.Errorf("task cannot be nil")
	}

	// Create tests directory if it doesn't exist
	testsDir := filepath.Join(tr.config.WorkspaceDir, "tests")
	if err := os.MkdirAll(testsDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create tests directory: %w", err)
	}

	testScriptPath := filepath.Join(testsDir, fmt.Sprintf("%s.py", task.ID))

	// Create test prompt file
	testPromptFile, err := tr.buildTestGenerationPromptFile(task, testScriptPath)
	if err != nil {
		return "", fmt.Errorf("failed to build test prompt: %w", err)
	}
	defer os.Remove(testPromptFile) // Clean up temporary file

	// Call Claude to generate the test script
	claudePath := tr.config.ClaudeCodePath
	if claudePath == "" {
		claudePath = "claude"
	}

	// Create command to generate test
	cmd := exec.Command(claudePath, "--dangerously-skip-permissions", testPromptFile)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	// Wait with timeout
	done := make(chan error, 1)
	go func() {
		done <- cmd.Run()
	}()

	timeout := 300 * time.Second // 5 minutes for test generation
	select {
	case err := <-done:
		if err != nil {
			output := stdout.String()
			if stderr.String() != "" {
				output += "\n\nSTDERR:\n" + stderr.String()
			}
			return "", fmt.Errorf("test generation failed: %w\nOutput: %s", err, output)
		}
	case <-time.After(timeout):
		cmd.Process.Kill()
		return "", fmt.Errorf("test generation timeout after %v", timeout)
	}

	// Verify test script was created
	if _, err := os.Stat(testScriptPath); os.IsNotExist(err) {
		return "", fmt.Errorf("test script was not created at %s", testScriptPath)
	}

	return testScriptPath, nil
}

// buildTestGenerationPromptFile builds a prompt file for Claude to generate a Python test script
// Uses the test_python.md template for consistent formatting
func (tr *TaskRunner) buildTestGenerationPromptFile(task *parser.Task, testScriptPath string) (string, error) {
	// Create prompt manager to load template
	promptMgr := prompt.NewPromptManager("")

	// Load test_python template
	template, err := promptMgr.LoadTemplate("test_python")
	if err != nil {
		return "", fmt.Errorf("failed to load test_python template: %w", err)
	}

	// Create prompt builder
	builder := prompt.NewPromptBuilder(template)

	// Set task information
	builder.SetVariable("task_id", task.ID)
	builder.SetVariable("task_name", task.Name)
	builder.SetVariable("task_goal", task.Goal)

	// Set test method
	builder.SetVariable("test_method", task.TestMethod)

	// Set test script path
	builder.SetVariable("test_script_path", testScriptPath)

	// Build prompt
	promptContent, err := builder.Build()
	if err != nil {
		return "", fmt.Errorf("failed to build test generation prompt: %w", err)
	}

	// Create temporary file
	tmpFile, err := os.CreateTemp("", fmt.Sprintf("rick-test-gen-%s-*.md", task.ID))
	if err != nil {
		return "", fmt.Errorf("failed to create temporary file: %w", err)
	}
	defer tmpFile.Close()

	if _, err := tmpFile.WriteString(promptContent); err != nil {
		os.Remove(tmpFile.Name())
		return "", fmt.Errorf("failed to write prompt to file: %w", err)
	}

	return tmpFile.Name(), nil
}

// GenerateDoingPromptFile generates the doing prompt file for Claude Code CLI
// Parameters:
//   - task: The task to execute
//   - debugContext: Content from debug.md (managed by Claude)
//   - testErrorFeedback: Previous test execution errors for test script correction
func (tr *TaskRunner) GenerateDoingPromptFile(task *parser.Task, debugContext string, testErrorFeedback string) (string, error) {
	if task == nil {
		return "", fmt.Errorf("task cannot be nil")
	}

	// Extract jobID from WorkspaceDir (.rick/jobs/job_X/doing → job_X)
	jobID := extractJobIDFromPath(tr.config.WorkspaceDir)

	// Create context manager with actual job ID
	contextMgr := prompt.NewContextManager(jobID)

	// Compute rickDir from workspaceDir (.rick/jobs/job_X/doing → .rick)
	rickDir := ""
	if tr.config.WorkspaceDir != "" {
		jobDir := filepath.Dir(tr.config.WorkspaceDir)  // go up to .rick/jobs/job_X
		rickDir = filepath.Dir(filepath.Dir(jobDir))    // go up to .rick

		// Load job-level OKR from job_N/plan/OKR.md (not global .rick/OKR.md)
		jobOKRPath := filepath.Join(jobDir, "plan", "OKR.md")
		if _, err := os.Stat(jobOKRPath); err == nil {
			contextMgr.LoadOKRFromFile(jobOKRPath)
		}

		specPath := filepath.Join(rickDir, "SPEC.md")
		if _, err := os.Stat(specPath); err == nil {
			contextMgr.LoadSPECFromFile(specPath)
		}
	}

	// Create prompt manager (use embedded templates)
	promptMgr := prompt.NewPromptManager("")

	// Generate doing prompt file (pass rickDir for skills injection)
	doingPromptFile, err := prompt.GenerateDoingPromptFile(task, 0, contextMgr, promptMgr, rickDir)
	if err != nil {
		return "", fmt.Errorf("failed to generate doing prompt: %w", err)
	}

	// Read existing content
	content, err := os.ReadFile(doingPromptFile)
	if err != nil {
		os.Remove(doingPromptFile)
		return "", fmt.Errorf("failed to read prompt file: %w", err)
	}

	var additionalContext strings.Builder

	// Append debug context if available
	if debugContext != "" {
		additionalContext.WriteString("\n\n## Previous Debugging Context\n\n")
		additionalContext.WriteString(debugContext)
		additionalContext.WriteString("\n\nPlease review the debugging context above and avoid the same mistakes.\n")
	}

	// Append test error feedback if available
	if testErrorFeedback != "" {
		additionalContext.WriteString("\n\n## Test Execution Feedback\n\n")
		additionalContext.WriteString("**Previous test execution encountered errors. You may need to fix the test script.**\n\n")
		additionalContext.WriteString("Test error details:\n")
		additionalContext.WriteString("```\n")
		additionalContext.WriteString(testErrorFeedback)
		additionalContext.WriteString("\n```\n\n")
		additionalContext.WriteString("**Action Required**:\n")
		additionalContext.WriteString("1. Review the test script for potential issues\n")
		additionalContext.WriteString("2. Check if the test logic correctly validates the task requirements\n")
		additionalContext.WriteString("3. Fix any bugs in the test script (path issues, logic errors, etc.)\n")
		additionalContext.WriteString("4. Ensure the test script outputs valid JSON format\n")
		additionalContext.WriteString("5. Re-run the task to verify the fix\n")
	}

	// Write back with additional context
	if additionalContext.Len() > 0 {
		if err := os.WriteFile(doingPromptFile, append(content, []byte(additionalContext.String())...), 0644); err != nil {
			os.Remove(doingPromptFile)
			return "", fmt.Errorf("failed to append additional context: %w", err)
		}
	}

	return doingPromptFile, nil
}

// extractJobIDFromPath extracts the job ID (e.g. "job_1") from a workspace directory path.
// Expected format: .rick/jobs/job_N/doing
func extractJobIDFromPath(dirPath string) string {
	parts := strings.Split(filepath.ToSlash(dirPath), "/")
	for i := len(parts) - 1; i >= 0; i-- {
		if strings.HasPrefix(parts[i], "job_") {
			return parts[i]
		}
	}
	return "job_N"
}

// CallClaudeCodeCLI calls Claude Code CLI in non-interactive mode
// promptFile is the path to the prompt file to be loaded by Claude
func (tr *TaskRunner) CallClaudeCodeCLI(promptFile string) (string, error) {
	if promptFile == "" {
		return "", fmt.Errorf("prompt file cannot be empty")
	}

	// Get Claude CLI path
	claudePath := tr.config.ClaudeCodePath
	if claudePath == "" {
		claudePath = "claude"
	}

	// Create command: claude --dangerously-skip-permissions <promptFile>
	cmd := exec.Command(claudePath, "--dangerously-skip-permissions", promptFile)

	// Capture output
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	// Wait for completion with timeout
	done := make(chan error, 1)
	go func() {
		done <- cmd.Run()
	}()

	timeout := time.Duration(tr.config.TimeoutSeconds) * time.Second
	if tr.config.TimeoutSeconds == 0 {
		timeout = 600 * time.Second // Default 10 minutes for Claude
	}

	select {
	case err := <-done:
		output := stdout.String()
		if stderr.String() != "" {
			output += "\n\nSTDERR:\n" + stderr.String()
		}
		if err != nil {
			return output, fmt.Errorf("Claude Code CLI execution failed: %w", err)
		}
		return output, nil
	case <-time.After(timeout):
		cmd.Process.Kill()
		return stdout.String(), fmt.Errorf("Claude Code CLI timeout after %d seconds", tr.config.TimeoutSeconds)
	}
}

// ExecuteTestScript executes a Python test script and parses JSON result
// Returns TestResult, raw output, and any error
func (tr *TaskRunner) ExecuteTestScript(scriptPath string) (*TestResult, string, error) {
	if scriptPath == "" {
		return nil, "", fmt.Errorf("script path cannot be empty")
	}

	// Verify script exists
	if _, err := os.Stat(scriptPath); err != nil {
		return nil, "", fmt.Errorf("script file not found: %w", err)
	}

	// Create command with timeout
	cmd := exec.Command("python3", scriptPath)

	// Capture output
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	// Set timeout if configured
	done := make(chan error, 1)
	go func() {
		done <- cmd.Run()
	}()

	// Wait for completion or timeout
	timeout := time.Duration(tr.config.TimeoutSeconds) * time.Second
	if tr.config.TimeoutSeconds == 0 {
		timeout = 30 * time.Second // Default timeout
	}

	select {
	case err := <-done:
		output := stdout.String()
		if stderr.String() != "" {
			output += "\nSTDERR:\n" + stderr.String()
		}

		// Parse JSON result from stdout
		testResult, parseErr := tr.parseTestResult(stdout.String())
		if parseErr != nil {
			return nil, output, fmt.Errorf("failed to parse test result: %w\nOutput: %s", parseErr, output)
		}

		// If script exited with error but we got valid JSON, use JSON result
		if err != nil && testResult == nil {
			return nil, output, fmt.Errorf("script execution failed: %w", err)
		}

		return testResult, output, nil

	case <-time.After(timeout):
		cmd.Process.Kill()
		return nil, stdout.String(), fmt.Errorf("script execution timeout after %d seconds", tr.config.TimeoutSeconds)
	}
}

// parseTestResult parses JSON test result from script output
func (tr *TaskRunner) parseTestResult(output string) (*TestResult, error) {
	if output == "" {
		return nil, fmt.Errorf("test output is empty")
	}

	// Try to find JSON in the output
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "{") {
			// Try to parse as JSON
			var result TestResult
			if err := json.Unmarshal([]byte(trimmed), &result); err == nil {
				return &result, nil
			}
		}
	}

	return nil, fmt.Errorf("no valid JSON result found in output")
}

// TaskExecutionResult represents the result of a task execution
type TaskExecutionResult struct {
	TaskID    string
	TaskName  string
	Status    string    // running, success, failed
	Error     string
	Output    string
	StartTime time.Time
	EndTime   time.Time
}

// Duration returns the execution duration
func (ter *TaskExecutionResult) Duration() time.Duration {
	return ter.EndTime.Sub(ter.StartTime)
}
