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

	"github.com/sunquan/rick/internal/actpath"
	"github.com/sunquan/rick/internal/agent"
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
	config        *ExecutionConfig
	agentExecutor agent.AgentExecutor
}

// NewTaskRunner creates a new TaskRunner instance
func NewTaskRunner(config *ExecutionConfig, agentExecutor agent.AgentExecutor) *TaskRunner {
	return &TaskRunner{
		config:        config,
		agentExecutor: agentExecutor,
	}
}

// TestResult represents the JSON result from a test script
type TestResult struct {
	Pass   bool     `json:"pass"`
	Errors []string `json:"errors"`
}

// TestGenContext provides optional additional context for test generation prompts.
type TestGenContext struct {
	DebugContent string
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

	// Step 2: Execute task with context (debug.md + test error feedback)
	var lastOutput string

	// Execute once and test
	doingPromptFile, _, err := tr.GenerateDoingPromptFile(task, debugContext, testErrorFeedback)
	if err != nil {
		result.Status = "failed"
		result.Error = fmt.Sprintf("failed to generate doing prompt: %v", err)
		result.EndTime = time.Now()
		return result, nil
	}

	session, err := tr.agentExecutor.Execute(doingPromptFile, task.ID, tr.config.WorkspaceDir, "raw_session_coding.log")
	if err != nil {
		result.Status = "failed"
		result.Error = fmt.Sprintf("agent execution failed: %v", err)
		result.EndTime = time.Now()
		return result, nil
	}

	if session != nil {
		actPathFile := filepath.Join(tr.config.WorkspaceDir, "tasks", task.ID, "act-path.md")
		if genErr := actpath.Generate(session, actPathFile); genErr != nil {
			fmt.Printf("[WARN] failed to generate act-path: %v\n", genErr)
		}
		lastOutput = session.FinalMessage()
	}

	// Run test to validate
	testResult, testOutput, err := tr.ExecuteTestScript(testScriptPath)
	if err != nil {
		result.Status = "failed"
		result.Error = fmt.Sprintf("test execution failed: %v\n\nFull test output:\n%s", err, testOutput)
		result.Output = fmt.Sprintf("Claude output:\n%s\n\nTest output:\n%s", lastOutput, testOutput)
		result.EndTime = time.Now()
		return result, nil
	}

	// Check if test passed
	if testResult.Pass {
		// Run doing_check to validate debug/ format before marking success
		if checkErr := RunDoingCheck(tr.config.WorkspaceDir); checkErr != nil {
			result.Status = "failed"
			result.Error = fmt.Sprintf("doing_check failed: %v", checkErr)
			result.Output = fmt.Sprintf("Claude output:\n%s\n\nTest output:\n%s", lastOutput, testOutput)
		} else {
			result.Status = "success"
			result.Output = fmt.Sprintf("Claude output:\n%s\n\nTest output:\n%s", lastOutput, testOutput)
		}
	} else {
		result.Status = "failed"
		result.Error = fmt.Sprintf("test did not pass: %s\n\nFull test output:\n%s", strings.Join(testResult.Errors, "; "), testOutput)
		result.Output = fmt.Sprintf("Claude output:\n%s\n\nTest output:\n%s", lastOutput, testOutput)
	}

	result.EndTime = time.Now()
	return result, nil
}

// GenerateTestWithAgent generates a Python test script using Claude Agent.
func (tr *TaskRunner) GenerateTestWithAgent(task *parser.Task) (string, error) {
	if task == nil {
		return "", fmt.Errorf("task cannot be nil")
	}

	testsDir := filepath.Join(tr.config.WorkspaceDir, "tests")
	if err := os.MkdirAll(testsDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create tests directory: %w", err)
	}

	testScriptPath := filepath.Join(testsDir, fmt.Sprintf("%s.py", task.ID))

	if _, err := os.Stat(testScriptPath); err == nil {
		return testScriptPath, nil
	}

	genCtx := TestGenContext{
		DebugContent: LoadDebugContext(tr.config.WorkspaceDir),
	}

	testPromptFile, _, err := tr.buildTestGenerationPromptFile(task, testScriptPath, genCtx)
	if err != nil {
		return "", fmt.Errorf("failed to build test prompt: %w", err)
	}

	if _, err := tr.agentExecutor.Execute(testPromptFile, task.ID, tr.config.WorkspaceDir, "raw_session_test_gen.log"); err != nil {
		return "", fmt.Errorf("test generation agent failed: %w", err)
	}

	if _, err := os.Stat(testScriptPath); os.IsNotExist(err) {
		return "", fmt.Errorf("test script was not created at %s", testScriptPath)
	}

	return testScriptPath, nil
}

// buildTestGenerationPromptFile builds a prompt file for test generation.
// Saves to doing/prompts/; files are persistent, no cleanup needed.
func (tr *TaskRunner) buildTestGenerationPromptFile(task *parser.Task, testScriptPath string, genCtx TestGenContext) (string, []string, error) {
	promptsDir, err := prompt.EnsurePromptsDir(tr.config.WorkspaceDir)
	if err != nil {
		return "", nil, fmt.Errorf("failed to create prompts dir: %w", err)
	}

	tddZhFile, err := prompt.WriteSkillFile(promptsDir, "skill_tdd_zh.md", "tdd-zh")
	if err != nil {
		return "", nil, fmt.Errorf("failed to write tdd-zh skill: %w", err)
	}
	testingAntiPatternsFile, err := prompt.WriteSkillFile(promptsDir, "skill_testing_anti_patterns_zh.md", "testing-anti-patterns-zh")
	if err != nil {
		return "", nil, fmt.Errorf("failed to write testing-anti-patterns-zh skill: %w", err)
	}

	promptMgr := prompt.NewPromptManager("")
	tmpl, err := promptMgr.LoadTemplate("test_python")
	if err != nil {
		return "", nil, fmt.Errorf("failed to load test_python template: %w", err)
	}

	builder := prompt.NewPromptBuilder(tmpl)
	builder.SetVariable("task_id", task.ID)
	builder.SetVariable("task_name", task.Name)
	builder.SetVariable("task_goal", task.Goal)
	builder.SetVariable("test_method", task.TestMethod)
	builder.SetVariable("test_script_path", testScriptPath)
	builder.SetVariable("debug_content", genCtx.DebugContent)
	builder.SetVariable("tdd_skill_path", tddZhFile)
	builder.SetVariable("testing_anti_patterns_path", testingAntiPatternsFile)

	promptFile := filepath.Join(promptsDir, fmt.Sprintf("%s_testgen_prompt.md", task.ID))
	if err := builder.SaveToFile(promptFile); err != nil {
		return "", nil, fmt.Errorf("failed to save test generation prompt: %w", err)
	}

	return promptFile, []string{tddZhFile, testingAntiPatternsFile}, nil
}

// GenerateDoingPromptFile generates the doing prompt file for Claude Code CLI.
// Returns the prompt file path, skill tmp files (caller must remove all), and any error.
func (tr *TaskRunner) GenerateDoingPromptFile(task *parser.Task, debugContext string, testErrorFeedback string) (string, []string, error) {
	if task == nil {
		return "", nil, fmt.Errorf("task cannot be nil")
	}

	// Extract jobID from WorkspaceDir (.rick/jobs/job_X/doing → job_X)
	jobID := extractJobIDFromPath(tr.config.WorkspaceDir)

	// Create context manager with actual job ID
	contextMgr := prompt.NewContextManager(jobID)

	// Compute rickDir and jobDir from workspaceDir (.rick/jobs/job_X/doing → .rick)
	rickDir := ""
	if tr.config.WorkspaceDir != "" {
		jobDir := filepath.Dir(tr.config.WorkspaceDir) // .rick/jobs/job_X
		rickDir = filepath.Dir(filepath.Dir(jobDir))   // .rick

		// Debug: prefer debug/ summaries, fall back to debug.md
		contextMgr.SetDebugRaw(LoadDebugContext(tr.config.WorkspaceDir))
	}

	// Create prompt manager (use embedded templates)
	promptMgr := prompt.NewPromptManager("")

	// Generate doing prompt file (pass doingDir for prompts/, rickDir for skills injection)
	doingPromptFile, skillFiles, err := prompt.GenerateDoingPromptFile(task, 0, contextMgr, promptMgr, tr.config.WorkspaceDir, rickDir)
	if err != nil {
		return "", nil, fmt.Errorf("failed to generate doing prompt: %w", err)
	}

	// Append test error feedback if available (not part of normal context; retry-specific)
	if testErrorFeedback != "" {
		content, err := os.ReadFile(doingPromptFile)
		if err != nil {
			return "", nil, fmt.Errorf("failed to read prompt file: %w", err)
		}
		var fb strings.Builder
		fb.WriteString("\n\n## Test Execution Feedback\n\n")
		fb.WriteString("**Previous test execution encountered errors. You may need to fix the test script.**\n\n")
		fb.WriteString("```\n")
		fb.WriteString(testErrorFeedback)
		fb.WriteString("\n```\n")
		if err := os.WriteFile(doingPromptFile, append(content, []byte(fb.String())...), 0644); err != nil {
			return "", nil, fmt.Errorf("failed to append test feedback: %w", err)
		}
	}

	return doingPromptFile, skillFiles, nil
}

// loadFileContent reads a file and returns its content, or "暂无" if the file is absent.
func loadFileContent(path string) string {
	content, err := os.ReadFile(path)
	if err != nil {
		return "暂无"
	}
	return string(content)
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
