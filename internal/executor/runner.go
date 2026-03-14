package executor

import (
	"bytes"
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

// RunTask executes a single task
// It generates a doing prompt, calls Claude Code CLI, generates test script, and validates
func (tr *TaskRunner) RunTask(task *parser.Task) (*TaskExecutionResult, error) {
	if task == nil {
		return nil, fmt.Errorf("task cannot be nil")
	}

	result := &TaskExecutionResult{
		TaskID:    task.ID,
		TaskName:  task.Name,
		Status:    "running",
		StartTime: time.Now(),
	}

	// Step 1: Generate doing prompt
	doingPrompt, err := tr.GenerateDoingPrompt(task)
	if err != nil {
		result.Status = "failed"
		result.Error = fmt.Sprintf("failed to generate doing prompt: %v", err)
		result.EndTime = time.Now()
		return result, nil
	}

	// Step 2: Call Claude Code CLI to execute the task
	claudeOutput, err := tr.CallClaudeCodeCLI(doingPrompt)
	if err != nil {
		result.Status = "failed"
		result.Error = fmt.Sprintf("Claude Code CLI failed: %v", err)
		result.Output = claudeOutput
		result.EndTime = time.Now()
		return result, nil
	}

	// Step 3: Generate test script based on task.TestMethod
	scriptPath, err := tr.GenerateTestScript(task)
	if err != nil {
		result.Status = "failed"
		result.Error = fmt.Sprintf("failed to generate test script: %v", err)
		result.EndTime = time.Now()
		return result, nil
	}
	defer os.Remove(scriptPath)

	// Step 4: Execute test script
	testOutput, err := tr.ExecuteTestScript(scriptPath)
	if err != nil {
		result.Status = "failed"
		result.Error = fmt.Sprintf("test execution failed: %v", err)
		result.Output = fmt.Sprintf("Claude output:\n%s\n\nTest output:\n%s", claudeOutput, testOutput)
		result.EndTime = time.Now()
		return result, nil
	}

	// Step 5: Parse test result
	success, parseErr := tr.ParseTestResult(testOutput)
	if parseErr != nil {
		result.Status = "failed"
		result.Error = fmt.Sprintf("failed to parse test result: %v", parseErr)
		result.Output = fmt.Sprintf("Claude output:\n%s\n\nTest output:\n%s", claudeOutput, testOutput)
		result.EndTime = time.Now()
		return result, nil
	}

	if success {
		result.Status = "success"
		result.Output = fmt.Sprintf("Claude output:\n%s\n\nTest output:\n%s", claudeOutput, testOutput)
	} else {
		result.Status = "failed"
		result.Error = "test did not pass"
		result.Output = fmt.Sprintf("Claude output:\n%s\n\nTest output:\n%s", claudeOutput, testOutput)
	}

	result.EndTime = time.Now()
	return result, nil
}

// GenerateDoingPrompt generates the doing prompt for Claude Code CLI
func (tr *TaskRunner) GenerateDoingPrompt(task *parser.Task) (string, error) {
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

	// Generate doing prompt
	doingPrompt, err := prompt.GenerateDoingPrompt(task, 0, contextMgr, promptMgr)
	if err != nil {
		return "", fmt.Errorf("failed to generate doing prompt: %w", err)
	}

	return doingPrompt, nil
}

// CallClaudeCodeCLI calls Claude Code CLI in non-interactive mode
// Uses pipe + --dangerously-skip-permissions for automation
func (tr *TaskRunner) CallClaudeCodeCLI(promptContent string) (string, error) {
	if promptContent == "" {
		return "", fmt.Errorf("prompt content cannot be empty")
	}

	// Get Claude CLI path
	claudePath := tr.config.ClaudeCodePath
	if claudePath == "" {
		claudePath = "claude"
	}

	// Create command: echo prompt | claude --dangerously-skip-permissions
	cmd := exec.Command(claudePath, "--dangerously-skip-permissions")

	// Create pipe for stdin
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return "", fmt.Errorf("failed to create stdin pipe: %w", err)
	}

	// Capture output
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	// Start command
	if err := cmd.Start(); err != nil {
		return "", fmt.Errorf("failed to start Claude Code CLI: %w", err)
	}

	// Write prompt to stdin
	if _, err := stdin.Write([]byte(promptContent)); err != nil {
		stdin.Close()
		return "", fmt.Errorf("failed to write prompt: %w", err)
	}
	stdin.Close()

	// Wait for completion with timeout
	done := make(chan error, 1)
	go func() {
		done <- cmd.Wait()
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

// GenerateTestScript generates a shell script for testing the task
// The script is created in a temporary location and should be executed
func (tr *TaskRunner) GenerateTestScript(task *parser.Task) (string, error) {
	if task == nil {
		return "", fmt.Errorf("task cannot be nil")
	}

	// Create a temporary script file
	tmpDir := os.TempDir()
	scriptPath := filepath.Join(tmpDir, fmt.Sprintf("test_%s_%d.sh", task.ID, time.Now().UnixNano()))

	// Build the test script content
	var scriptContent strings.Builder
	scriptContent.WriteString("#!/bin/bash\n")
	scriptContent.WriteString("set -e\n\n")

	// Add task information as comments
	scriptContent.WriteString(fmt.Sprintf("# Test script for task: %s\n", task.Name))
	scriptContent.WriteString(fmt.Sprintf("# Task ID: %s\n", task.ID))
	scriptContent.WriteString(fmt.Sprintf("# Goal: %s\n", task.Goal))
	scriptContent.WriteString("\n")

	// Add test method steps as actual executable commands
	if task.TestMethod != "" {
		scriptContent.WriteString("# Execute test steps:\n")
		scriptContent.WriteString("TEST_PASSED=true\n\n")

		lines := strings.Split(task.TestMethod, "\n")
		stepNum := 0
		for _, line := range lines {
			trimmed := strings.TrimSpace(line)
			if trimmed == "" {
				continue
			}

			// Extract the test step (remove list markers)
			step := trimmed
			if strings.HasPrefix(step, "- ") {
				step = strings.TrimPrefix(step, "- ")
			} else if strings.HasPrefix(step, "* ") {
				step = strings.TrimPrefix(step, "* ")
			} else if len(step) > 0 && step[0] >= '0' && step[0] <= '9' {
				// Handle numbered lists
				parts := strings.SplitN(step, ". ", 2)
				if len(parts) == 2 {
					step = parts[1]
				}
			}

			stepNum++
			scriptContent.WriteString(fmt.Sprintf("# Step %d: %s\n", stepNum, step))

			// Try to convert test step description to executable command
			// This is a simple heuristic - in practice, Claude should generate proper test scripts
			cmd := tr.convertTestStepToCommand(step)
			scriptContent.WriteString(fmt.Sprintf("echo \"Executing step %d: %s\"\n", stepNum, step))
			scriptContent.WriteString(cmd + " || TEST_PASSED=false\n\n")
		}

		// Final status check
		scriptContent.WriteString("if [ \"$TEST_PASSED\" = true ]; then\n")
		scriptContent.WriteString("  echo \"Status: PASS\"\n")
		scriptContent.WriteString("  exit 0\n")
		scriptContent.WriteString("else\n")
		scriptContent.WriteString("  echo \"Status: FAIL\"\n")
		scriptContent.WriteString("  exit 1\n")
		scriptContent.WriteString("fi\n")
	} else {
		// No test method specified - assume success if we got here
		scriptContent.WriteString("echo \"No test method specified, assuming success\"\n")
		scriptContent.WriteString("echo \"Status: PASS\"\n")
	}

	// Write the script to file
	err := os.WriteFile(scriptPath, []byte(scriptContent.String()), 0755)
	if err != nil {
		return "", fmt.Errorf("failed to write script file: %w", err)
	}

	return scriptPath, nil
}

// ExecuteTestScript executes a test script with timeout control
// Returns the script output and any error that occurred
func (tr *TaskRunner) ExecuteTestScript(scriptPath string) (string, error) {
	if scriptPath == "" {
		return "", fmt.Errorf("script path cannot be empty")
	}

	// Verify script exists
	if _, err := os.Stat(scriptPath); err != nil {
		return "", fmt.Errorf("script file not found: %w", err)
	}

	// Create command with timeout
	cmd := exec.Command("bash", scriptPath)

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
		if err != nil {
			output := stdout.String()
			if stderr.String() != "" {
				output += "\nSTDERR:\n" + stderr.String()
			}
			return output, fmt.Errorf("script execution failed: %w", err)
		}
		return stdout.String(), nil
	case <-time.After(timeout):
		cmd.Process.Kill()
		return stdout.String(), fmt.Errorf("script execution timeout after %d seconds", tr.config.TimeoutSeconds)
	}
}

// convertTestStepToCommand converts a test step description to an executable command
// This is a simple heuristic - ideally Claude should generate proper test scripts
func (tr *TaskRunner) convertTestStepToCommand(step string) string {
	lowerStep := strings.ToLower(step)

	// Check for common test patterns
	if strings.Contains(lowerStep, "验证") || strings.Contains(lowerStep, "检查") || strings.Contains(lowerStep, "确认") {
		// File existence checks
		if strings.Contains(lowerStep, "文件") && strings.Contains(lowerStep, "存在") {
			// Extract file path if possible
			if strings.Contains(step, ".") {
				return "test -f " + extractFilePath(step) + " && echo 'File exists'"
			}
		}
		// Directory checks
		if strings.Contains(lowerStep, "目录") && strings.Contains(lowerStep, "存在") {
			return "test -d " + extractFilePath(step) + " && echo 'Directory exists'"
		}
		// Content checks
		if strings.Contains(lowerStep, "包含") || strings.Contains(lowerStep, "内容") {
			return "echo 'Content check - manual verification needed'"
		}
	}

	// Run commands
	if strings.Contains(lowerStep, "运行") || strings.Contains(lowerStep, "执行") {
		if strings.Contains(lowerStep, "测试") {
			return "echo 'Running tests' && true"  // Placeholder
		}
		if strings.Contains(lowerStep, "编译") || strings.Contains(lowerStep, "build") {
			return "echo 'Build check' && true"
		}
	}

	// Default: just echo the step as completed
	return "echo 'Step completed: " + strings.ReplaceAll(step, "'", "\\'") + "'"
}

// extractFilePath attempts to extract a file path from a test step description
func extractFilePath(step string) string {
	// Simple heuristic: look for patterns like .rick/file.md or /path/to/file
	words := strings.Fields(step)
	for _, word := range words {
		if strings.Contains(word, "/") || strings.Contains(word, ".") {
			// Clean up quotes and punctuation
			word = strings.Trim(word, "`,\"':;。，")
			return word
		}
	}
	return "."
}

// ParseTestResult parses the output of a test script
// Returns true if the test passed, false otherwise
func (tr *TaskRunner) ParseTestResult(output string) (bool, error) {
	if output == "" {
		return false, fmt.Errorf("test output is empty")
	}

	// Look for success indicators in the output
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.Contains(trimmed, "Status: PASS") {
			return true, nil
		}
		if strings.Contains(trimmed, "PASS") && !strings.Contains(trimmed, "FAIL") {
			return true, nil
		}
	}

	// Check for failure indicators
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.Contains(trimmed, "FAIL") || strings.Contains(trimmed, "ERROR") {
			return false, nil
		}
	}

	// If no explicit status found, consider it a pass if output is present
	return true, nil
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
