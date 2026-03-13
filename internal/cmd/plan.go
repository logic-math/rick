package cmd

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/sunquan/rick/internal/config"
	"github.com/sunquan/rick/internal/executor"
	"github.com/sunquan/rick/internal/git"
	"github.com/sunquan/rick/internal/parser"
	"github.com/sunquan/rick/internal/prompt"
	"github.com/sunquan/rick/internal/workspace"
)

func NewPlanCmd() *cobra.Command {
	planCmd := &cobra.Command{
		Use:   "plan [requirement]",
		Short: "Plan a new job with AI assistance",
		Long:  `Plan a new job by describing your requirement. Rick will use AI to break it down into tasks.`,
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if GetVerbose() {
				fmt.Println("[INFO] Starting plan phase...")
			}

			if GetDryRun() {
				fmt.Println("[DRY-RUN] Would create a plan")
				return nil
			}

			// Get requirement from args or interactive input
			requirement := ""
			if len(args) > 0 {
				requirement = args[0]
			} else {
				var err error
				requirement, err = promptForRequirement()
				if err != nil {
					return fmt.Errorf("failed to get requirement: %w", err)
				}
			}

			if requirement == "" {
				return fmt.Errorf("requirement cannot be empty")
			}

			if GetVerbose() {
				fmt.Printf("[INFO] Requirement: %s\n", requirement)
			}

			// Execute planning workflow
			if err := executePlanWorkflow(requirement); err != nil {
				return err
			}

			fmt.Println("Plan created successfully!")
			return nil
		},
	}

	return planCmd
}

// promptForRequirement prompts user for requirement input
func promptForRequirement() (string, error) {
	reader := bufio.NewReader(os.Stdin)
	fmt.Print("Enter your requirement: ")
	requirement, err := reader.ReadString('\n')
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(requirement), nil
}

// executePlanWorkflow executes the complete planning workflow
func executePlanWorkflow(requirement string) error {
	// Step 1: Load configuration and workspace
	cfg, err := config.LoadConfig()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	ws, err := workspace.New()
	if err != nil {
		return fmt.Errorf("failed to create workspace: %w", err)
	}

	rickDir, err := workspace.GetRickDir()
	if err != nil {
		return fmt.Errorf("failed to get rick directory: %w", err)
	}

	if GetVerbose() {
		fmt.Printf("[INFO] Using workspace: %s\n", rickDir)
	}

	// Step 2: Generate planning prompt
	if GetVerbose() {
		fmt.Println("[INFO] Generating planning prompt...")
	}

	contextMgr := prompt.NewContextManager("plan")

	// Load OKR and SPEC
	okriPath := filepath.Join(rickDir, workspace.OKRFileName)
	if _, err := os.Stat(okriPath); err == nil {
		if err := contextMgr.LoadOKRFromFile(okriPath); err != nil && GetVerbose() {
			fmt.Printf("[WARN] Failed to load OKR: %v\n", err)
		}
	}

	specPath := filepath.Join(rickDir, workspace.SpecFileName)
	if _, err := os.Stat(specPath); err == nil {
		if err := contextMgr.LoadSPECFromFile(specPath); err != nil && GetVerbose() {
			fmt.Printf("[WARN] Failed to load SPEC: %v\n", err)
		}
	}

	// Create prompt manager
	templateDir := filepath.Join(rickDir, "templates")
	if _, err := os.Stat(templateDir); os.IsNotExist(err) {
		// Use default template directory from internal/prompt/templates
		templateDir = "" // Will use embedded templates
	}

	promptMgr := prompt.NewPromptManager(templateDir)

	planPrompt, err := prompt.GeneratePlanPrompt(requirement, contextMgr, promptMgr)
	if err != nil {
		return fmt.Errorf("failed to generate plan prompt: %w", err)
	}

	if GetVerbose() {
		fmt.Println("[INFO] Planning prompt generated")
	}

	// Step 3: Call Claude Code CLI with planning prompt
	if GetVerbose() {
		fmt.Println("[INFO] Calling Claude Code CLI for planning...")
	}

	claudeOutput, err := callClaudeCodeCLI(cfg, planPrompt)
	if err != nil {
		return fmt.Errorf("failed to call Claude Code CLI: %w", err)
	}

	if GetVerbose() {
		fmt.Printf("[INFO] Claude output length: %d bytes\n", len(claudeOutput))
	}

	// Step 4: Parse Claude output and generate task files
	if GetVerbose() {
		fmt.Println("[INFO] Parsing Claude output and generating task files...")
	}

	jobID, tasks, err := parseClaudeOutputAndGenerateTasks(claudeOutput, ws)
	if err != nil {
		return fmt.Errorf("failed to parse Claude output: %w", err)
	}

	if GetVerbose() {
		fmt.Printf("[INFO] Created job: %s with %d tasks\n", jobID, len(tasks))
	}

	// Step 5: Generate tasks.json with DAG and topological sort
	if GetVerbose() {
		fmt.Println("[INFO] Generating tasks.json with DAG...")
	}

	jobPath, err := ws.GetJobPath(jobID)
	if err != nil {
		return fmt.Errorf("failed to get job path: %w", err)
	}

	if err := generateTasksJSON(jobPath, tasks); err != nil {
		return fmt.Errorf("failed to generate tasks.json: %w", err)
	}

	if GetVerbose() {
		fmt.Println("[INFO] tasks.json generated successfully")
	}

	// Step 6: Auto-commit planning results
	if GetVerbose() {
		fmt.Println("[INFO] Committing planning results...")
	}

	if err := commitPlanResults(rickDir, jobID); err != nil {
		if GetVerbose() {
			fmt.Printf("[WARN] Failed to commit: %v\n", err)
		}
	}

	return nil
}

// callClaudeCodeCLI calls Claude Code CLI with the planning prompt
func callClaudeCodeCLI(cfg *config.Config, prompt string) (string, error) {
	// Create temporary prompt file
	tmpFile, err := os.CreateTemp("", "plan-prompt-*.md")
	if err != nil {
		return "", fmt.Errorf("failed to create temp prompt file: %w", err)
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.WriteString(prompt); err != nil {
		tmpFile.Close()
		return "", fmt.Errorf("failed to write prompt to file: %w", err)
	}
	tmpFile.Close()

	// Call Claude Code CLI
	claudePath := cfg.ClaudeCodePath
	if claudePath == "" {
		claudePath = "claude"
	}

	cmd := exec.Command(claudePath, tmpFile.Name())
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("Claude Code CLI failed: %w\nOutput: %s", err, string(output))
	}

	return string(output), nil
}

// parseClaudeOutputAndGenerateTasks parses Claude output and generates task files
func parseClaudeOutputAndGenerateTasks(claudeOutput string, ws *workspace.Workspace) (string, []*parser.Task, error) {
	// Generate job ID (job_N format)
	jobID := generateJobID()

	// Create job structure
	if err := ws.CreateJobStructure(jobID); err != nil {
		return "", nil, fmt.Errorf("failed to create job structure: %w", err)
	}

	jobPath, err := ws.GetJobPath(jobID)
	if err != nil {
		return "", nil, err
	}

	planDir := filepath.Join(jobPath, workspace.PlanDirName)
	planTasksDir := filepath.Join(planDir, "tasks")

	// Parse Claude output to extract tasks
	// This is a simplified parser - in production, you'd have more sophisticated parsing
	tasks, err := parseTasksFromClaudeOutput(claudeOutput, planTasksDir, jobID)
	if err != nil {
		return "", nil, fmt.Errorf("failed to parse tasks from Claude output: %w", err)
	}

	if len(tasks) == 0 {
		return "", nil, fmt.Errorf("no tasks found in Claude output")
	}

	return jobID, tasks, nil
}

// parseTasksFromClaudeOutput extracts tasks from Claude output and creates task files
func parseTasksFromClaudeOutput(output string, tasksDir string, jobID string) ([]*parser.Task, error) {
	if err := os.MkdirAll(tasksDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create tasks directory: %w", err)
	}

	// Split output by task markers (e.g., "## Task 1:", "### Task 2:", etc.)
	var tasks []*parser.Task
	taskBlocks := strings.Split(output, "\n## Task ")

	taskIndex := 0
	for i, block := range taskBlocks {
		if i == 0 && !strings.Contains(block, "Task") {
			// Skip header if not a task
			continue
		}

		// Clean up block
		if i > 0 {
			block = "## Task " + block
		}

		// Create task from block
		task, err := createTaskFromBlock(block, taskIndex, jobID)
		if err != nil {
			if GetVerbose() {
				fmt.Printf("[WARN] Failed to parse task block %d: %v\n", i, err)
			}
			continue
		}

		// Write task file
		taskFile := filepath.Join(tasksDir, fmt.Sprintf("task_%d.md", taskIndex))
		if err := os.WriteFile(taskFile, []byte(block), 0644); err != nil {
			return nil, fmt.Errorf("failed to write task file: %w", err)
		}

		tasks = append(tasks, task)
		taskIndex++
	}

	return tasks, nil
}

// createTaskFromBlock creates a Task struct from a task block
func createTaskFromBlock(block string, index int, jobID string) (*parser.Task, error) {
	task := &parser.Task{
		ID:   fmt.Sprintf("task_%d", index),
		Name: extractField(block, "# 任务名称", "任务"),
	}

	if task.Name == "" {
		task.Name = fmt.Sprintf("Task %d", index)
	}

	task.Goal = extractField(block, "# 任务目标", "")
	task.TestMethod = extractField(block, "# 测试方法", "")

	// Parse dependencies
	depStr := extractField(block, "# 依赖关系", "")
	if depStr != "" {
		task.Dependencies = strings.Split(depStr, ",")
		for i := range task.Dependencies {
			task.Dependencies[i] = strings.TrimSpace(task.Dependencies[i])
		}
	}

	// Parse key results
	krStr := extractField(block, "# 关键结果", "")
	if krStr != "" {
		lines := strings.Split(krStr, "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, "-") {
				line = strings.TrimPrefix(line, "-")
				line = strings.TrimSpace(line)
			}
			if line != "" {
				task.KeyResults = append(task.KeyResults, line)
			}
		}
	}

	return task, nil
}

// extractField extracts a field value from markdown block
// For multi-line fields, returns all lines until the next field marker
func extractField(block string, fieldName string, defaultValue string) string {
	lines := strings.Split(block, "\n")
	var result []string
	found := false

	for i, line := range lines {
		if strings.Contains(line, fieldName) {
			found = true
			// Start collecting from the next line
			for j := i + 1; j < len(lines); j++ {
				nextLine := strings.TrimSpace(lines[j])
				// Stop at the next field marker (starts with #)
				if strings.HasPrefix(nextLine, "#") {
					break
				}
				if nextLine != "" {
					result = append(result, nextLine)
				}
			}
			break
		}
	}

	if !found {
		return defaultValue
	}

	if len(result) == 0 {
		return defaultValue
	}

	return strings.Join(result, "\n")
}

// generateTasksJSON generates tasks.json with DAG and topological sort
func generateTasksJSON(jobPath string, tasks []*parser.Task) error {
	// Create DAG
	dag, err := executor.NewDAG(tasks)
	if err != nil {
		return fmt.Errorf("failed to create DAG: %w", err)
	}

	// Validate DAG (check for cycles)
	if err := dag.ValidateDAG(); err != nil {
		return fmt.Errorf("DAG validation failed: %w", err)
	}

	// Topological sort
	sortedTasks, err := executor.TopologicalSort(dag)
	if err != nil {
		return fmt.Errorf("topological sort failed: %w", err)
	}

	// Generate tasks.json
	tasksJSON, err := executor.GenerateTasksJSON(dag, sortedTasks)
	if err != nil {
		return fmt.Errorf("failed to generate tasks.json: %w", err)
	}

	// Save tasks.json
	planDir := filepath.Join(jobPath, workspace.PlanDirName)
	tasksJSONPath := filepath.Join(planDir, "tasks.json")
	if err := executor.SaveTasksJSON(tasksJSONPath, tasksJSON); err != nil {
		return fmt.Errorf("failed to save tasks.json: %w", err)
	}

	return nil
}

// commitPlanResults commits the planning results to git
func commitPlanResults(rickDir string, jobID string) error {
	gm := git.New(rickDir)

	// Check if repository exists, if not initialize
	if !gm.IsRepository() {
		if err := gm.InitRepo(); err != nil {
			return fmt.Errorf("failed to initialize git repo: %w", err)
		}
	}

	// Add job directory
	jobsDir := filepath.Join(rickDir, workspace.JobsDirName)
	if err := gm.AddFiles([]string{filepath.Join(jobsDir, jobID)}); err != nil {
		return fmt.Errorf("failed to add job files: %w", err)
	}

	// Commit
	commitMsg := fmt.Sprintf("plan: %s - Planning completed", jobID)
	if err := gm.Commit(commitMsg); err != nil {
		// Ignore commit error if nothing to commit
		if !strings.Contains(err.Error(), "nothing to commit") {
			return err
		}
	}

	return nil
}

// generateJobID generates a new job ID (job_N format)
func generateJobID() string {
	// Use timestamp-based ID for uniqueness
	return fmt.Sprintf("job_%d", time.Now().Unix())
}
