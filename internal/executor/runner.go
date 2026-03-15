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
// 2. Enter execution loop: execute task -> run test -> retry if failed
func (tr *TaskRunner) RunTask(task *parser.Task, debugContext string) (*TaskExecutionResult, error) {
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
	defer os.Remove(testScriptPath)

	// Step 2: Execution loop - keep trying until test passes
	// This implements: while not pass: execute -> test -> retry
	var lastOutput string

	// Execute once and test
	doingPromptFile, err := tr.GenerateDoingPromptFile(task, debugContext)
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
		result.Error = fmt.Sprintf("test execution failed: %v", err)
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
		result.Error = fmt.Sprintf("test did not pass: %s", strings.Join(testResult.Errors, "; "))
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
func (tr *TaskRunner) buildTestGenerationPromptFile(task *parser.Task, testScriptPath string) (string, error) {
	var prompt strings.Builder
	prompt.WriteString("# Test Generation Task\n\n")
	prompt.WriteString("You need to generate a Python test script based on the task's test method.\n\n")
	prompt.WriteString("## Task Information\n\n")
	prompt.WriteString(fmt.Sprintf("**Task ID**: %s\n", task.ID))
	prompt.WriteString(fmt.Sprintf("**Task Name**: %s\n", task.Name))
	prompt.WriteString(fmt.Sprintf("**Task Goal**: %s\n\n", task.Goal))

	prompt.WriteString("## Test Method\n\n")
	prompt.WriteString(task.TestMethod)
	prompt.WriteString("\n\n")

	prompt.WriteString("## Requirements\n\n")
	prompt.WriteString(fmt.Sprintf("1. Create a Python test script at: `%s`\n", testScriptPath))
	prompt.WriteString("2. The script MUST return a JSON result in this format:\n")
	prompt.WriteString("   ```json\n")
	prompt.WriteString("   {\"pass\": true/false, \"errors\": [\"error1\", \"error2\"]}\n")
	prompt.WriteString("   ```\n")
	prompt.WriteString("3. Implement each test step from the test method above\n")
	prompt.WriteString("4. The script should be executable with: `python3 " + testScriptPath + "`\n")
	prompt.WriteString("5. Make sure to handle errors gracefully and report them in the errors array\n")
	prompt.WriteString("6. Use absolute paths when checking files\n\n")

	prompt.WriteString("## Example Test Script Structure\n\n")
	prompt.WriteString("```python\n")
	prompt.WriteString("#!/usr/bin/env python3\n")
	prompt.WriteString("import json\n")
	prompt.WriteString("import sys\n")
	prompt.WriteString("import os\n\n")
	prompt.WriteString("def main():\n")
	prompt.WriteString("    errors = []\n")
	prompt.WriteString("    \n")
	prompt.WriteString("    # Test step 1\n")
	prompt.WriteString("    if not os.path.exists('file.txt'):\n")
	prompt.WriteString("        errors.append('file.txt does not exist')\n")
	prompt.WriteString("    \n")
	prompt.WriteString("    # Test step 2\n")
	prompt.WriteString("    # ...\n")
	prompt.WriteString("    \n")
	prompt.WriteString("    result = {\n")
	prompt.WriteString("        'pass': len(errors) == 0,\n")
	prompt.WriteString("        'errors': errors\n")
	prompt.WriteString("    }\n")
	prompt.WriteString("    print(json.dumps(result))\n")
	prompt.WriteString("    sys.exit(0 if result['pass'] else 1)\n\n")
	prompt.WriteString("if __name__ == '__main__':\n")
	prompt.WriteString("    main()\n")
	prompt.WriteString("```\n\n")

	prompt.WriteString("Please generate the test script now. Do NOT execute the task itself, ONLY generate the test script.\n")

	// Create temporary file
	tmpFile, err := os.CreateTemp("", fmt.Sprintf("rick-test-gen-%s-*.md", task.ID))
	if err != nil {
		return "", fmt.Errorf("failed to create temporary file: %w", err)
	}
	defer tmpFile.Close()

	if _, err := tmpFile.WriteString(prompt.String()); err != nil {
		os.Remove(tmpFile.Name())
		return "", fmt.Errorf("failed to write prompt to file: %w", err)
	}

	return tmpFile.Name(), nil
}

// GenerateDoingPromptFile generates the doing prompt file for Claude Code CLI
func (tr *TaskRunner) GenerateDoingPromptFile(task *parser.Task, debugContext string) (string, error) {
	if task == nil {
		return "", fmt.Errorf("task cannot be nil")
	}

	// Create context manager
	contextMgr := prompt.NewContextManager("doing")

	// Load OKR and SPEC if available
	if tr.config.WorkspaceDir != "" {
		rickDir := filepath.Dir(tr.config.WorkspaceDir) // workspaceDir is .rick/jobs/job_X/doing
		rickDir = filepath.Dir(rickDir)                  // go up to .rick/jobs/job_X
		rickDir = filepath.Dir(rickDir)                  // go up to .rick/jobs
		rickDir = filepath.Dir(rickDir)                  // go up to .rick

		okriPath := filepath.Join(rickDir, "OKR.md")
		if _, err := os.Stat(okriPath); err == nil {
			contextMgr.LoadOKRFromFile(okriPath)
		}

		specPath := filepath.Join(rickDir, "SPEC.md")
		if _, err := os.Stat(specPath); err == nil {
			contextMgr.LoadSPECFromFile(specPath)
		}
	}

	// Create prompt manager (use embedded templates)
	promptMgr := prompt.NewPromptManager("")

	// Generate doing prompt file
	doingPromptFile, err := prompt.GenerateDoingPromptFile(task, 0, contextMgr, promptMgr)
	if err != nil {
		return "", fmt.Errorf("failed to generate doing prompt: %w", err)
	}

	// Append debug context if available
	if debugContext != "" {
		// Read existing content
		content, err := os.ReadFile(doingPromptFile)
		if err != nil {
			os.Remove(doingPromptFile)
			return "", fmt.Errorf("failed to read prompt file: %w", err)
		}

		// Append debug context
		debugSection := "\n\n## Previous Debugging Context\n\n" + debugContext +
			"\n\nPlease review the debugging context above and avoid the same mistakes.\n"

		// Write back
		if err := os.WriteFile(doingPromptFile, append(content, []byte(debugSection)...), 0644); err != nil {
			os.Remove(doingPromptFile)
			return "", fmt.Errorf("failed to append debug context: %w", err)
		}
	}

	return doingPromptFile, nil
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
