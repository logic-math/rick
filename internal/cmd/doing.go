package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/sunquan/rick/internal/config"
	"github.com/sunquan/rick/internal/executor"
	"github.com/sunquan/rick/internal/git"
	"github.com/sunquan/rick/internal/parser"
	"github.com/sunquan/rick/internal/workspace"
)

func NewDoingCmd() *cobra.Command {
	var jobID string

	doingCmd := &cobra.Command{
		Use:   "doing [job_id]",
		Short: "Execute tasks in a job",
		Long:  `Execute tasks in a job. Supports retry mechanism and automatic commits.`,
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if GetVerbose() {
				fmt.Println("[INFO] Starting doing phase...")
			}

			if GetDryRun() {
				fmt.Println("[DRY-RUN] Would execute job")
				return nil
			}

			// Get job ID from args, local flag, or global flag
			if len(args) > 0 {
				jobID = args[0]
			} else if jobID == "" {
				jobID = GetJobID()
			}

			if jobID == "" {
				return fmt.Errorf("job ID is required. Usage: rick doing [job_id] or rick doing --job job_id")
			}

			// Validate job ID format
			if err := validateJobID(jobID); err != nil {
				return err
			}

			if GetVerbose() {
				fmt.Printf("[INFO] Executing job: %s\n", jobID)
			}

			// Execute doing workflow
			if err := executeDoingWorkflow(jobID); err != nil {
				return err
			}

			fmt.Printf("Job %s execution completed!\n", jobID)
			return nil
		},
	}

	doingCmd.Flags().StringVar(&jobID, "job", "", "Job ID to execute")

	return doingCmd
}

// executeDoingWorkflow executes the complete doing workflow
func executeDoingWorkflow(jobID string) error {
	// Step 1: Load configuration and workspace
	cfg, err := config.LoadConfig()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	_, err = workspace.New()
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

	// Step 1.5: Auto-initialize Git repository if not exists
	if err := ensureGitInitialized(rickDir); err != nil {
		if GetVerbose() {
			fmt.Printf("[WARN] Failed to initialize Git repository: %v\n", err)
		}
	}

	// Step 2: Validate job directory structure
	jobDir := filepath.Join(rickDir, "jobs", jobID)
	planDir := filepath.Join(jobDir, "plan")
	doingDir := filepath.Join(jobDir, "doing")

	if _, err := os.Stat(jobDir); os.IsNotExist(err) {
		return fmt.Errorf("job directory not found: %s", jobDir)
	}

	if _, err := os.Stat(planDir); os.IsNotExist(err) {
		return fmt.Errorf("plan directory not found: %s", planDir)
	}

	// Create doing directory if it doesn't exist
	if err := os.MkdirAll(doingDir, 0755); err != nil {
		return fmt.Errorf("failed to create doing directory: %w", err)
	}

	if GetVerbose() {
		fmt.Printf("[INFO] Job directory: %s\n", jobDir)
		fmt.Printf("[INFO] Plan directory: %s\n", planDir)
		fmt.Printf("[INFO] Doing directory: %s\n", doingDir)
	}

	// Step 3: Load tasks from plan directory
	if GetVerbose() {
		fmt.Println("[INFO] Loading tasks from plan directory...")
	}

	tasks, err := loadTasksFromPlan(planDir)
	if err != nil {
		return fmt.Errorf("failed to load tasks: %w", err)
	}

	if len(tasks) == 0 {
		return fmt.Errorf("no tasks found in plan directory")
	}

	if GetVerbose() {
		fmt.Printf("[INFO] Loaded %d tasks\n", len(tasks))
		for i, task := range tasks {
			fmt.Printf("  [%d] %s: %s\n", i+1, task.ID, task.Name)
		}
	}

	// Step 4: Create executor with execution config
	execConfig := &executor.ExecutionConfig{
		MaxRetries:     cfg.MaxRetries,
		TimeoutSeconds: 3600,
		LogFile:        filepath.Join(doingDir, "execution.log"),
		ClaudeCodePath: cfg.ClaudeCodePath,
		WorkspaceDir:   doingDir,
	}

	if GetVerbose() {
		fmt.Printf("[INFO] Execution config: MaxRetries=%d, TimeoutSeconds=%d\n",
			execConfig.MaxRetries, execConfig.TimeoutSeconds)
	}

	exec, err := executor.NewExecutor(tasks, execConfig, doingDir, jobID)
	if err != nil {
		return fmt.Errorf("failed to create executor: %w", err)
	}

	// Step 5: Execute job
	if GetVerbose() {
		fmt.Println("[INFO] Starting task execution...")
	}

	result, err := exec.ExecuteJob()
	if err != nil {
		return fmt.Errorf("execution error: %w", err)
	}

	// Step 6: Print execution summary
	printExecutionSummary(result)

	// Step 7: Handle execution results
	if result.Status == "completed" {
		// All tasks succeeded, commit the results
		if GetVerbose() {
			fmt.Println("[INFO] All tasks completed successfully, committing results...")
		}

		if err := commitDoingResults(jobID, result); err != nil {
			fmt.Printf("[WARN] Failed to commit results: %v\n", err)
		}

		return nil
	} else if result.Status == "partial" {
		// Some tasks succeeded, some failed
		fmt.Printf("\n⚠ Job execution completed with partial success (%d/%d tasks)\n",
			result.SuccessfulTasks, result.TotalTasks)
		fmt.Println("Please review the failed tasks and run 'rick doing job_id' again to retry.")

		// Commit partial results
		if GetVerbose() {
			fmt.Println("[INFO] Committing partial results...")
		}

		if err := commitDoingResults(jobID, result); err != nil {
			fmt.Printf("[WARN] Failed to commit partial results: %v\n", err)
		}

		return fmt.Errorf("job execution incomplete: %d/%d tasks failed", result.FailedTasks, result.TotalTasks)
	} else {
		// All tasks failed
		fmt.Printf("\n✗ Job execution failed (%d/%d tasks failed)\n",
			result.FailedTasks, result.TotalTasks)
		fmt.Println("Please review the errors and run 'rick doing job_id' again to retry.")

		return fmt.Errorf("job execution failed: all tasks failed")
	}
}

// loadTasksFromPlan loads all task.md files from the plan directory
func loadTasksFromPlan(planDir string) ([]*parser.Task, error) {
	tasks := make([]*parser.Task, 0)

	// List all task*.md files in the plan directory
	entries, err := os.ReadDir(planDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read plan directory: %w", err)
	}

	taskFiles := make([]string, 0)
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasPrefix(entry.Name(), "task") && strings.HasSuffix(entry.Name(), ".md") {
			taskFiles = append(taskFiles, entry.Name())
		}
	}

	if len(taskFiles) == 0 {
		return nil, fmt.Errorf("no task*.md files found in plan directory")
	}

	// Sort task files by name to ensure consistent order
	sortTaskFiles(taskFiles)

	// Parse each task file
	for _, taskFile := range taskFiles {
		taskPath := filepath.Join(planDir, taskFile)

		content, err := os.ReadFile(taskPath)
		if err != nil {
			return nil, fmt.Errorf("failed to read task file %s: %w", taskFile, err)
		}

		task, err := parser.ParseTask(string(content))
		if err != nil {
			return nil, fmt.Errorf("failed to parse task file %s: %w", taskFile, err)
		}

		// Extract task ID from filename (task1.md -> task1)
		taskID := strings.TrimSuffix(taskFile, ".md")
		task.ID = taskID

		tasks = append(tasks, task)
	}

	return tasks, nil
}

// sortTaskFiles sorts task files in numerical order (task1.md, task2.md, ...)
func sortTaskFiles(files []string) {
	// Simple bubble sort for small lists
	for i := 0; i < len(files); i++ {
		for j := i + 1; j < len(files); j++ {
			// Extract numbers from filenames
			numI := extractTaskNumber(files[i])
			numJ := extractTaskNumber(files[j])
			if numJ < numI {
				files[i], files[j] = files[j], files[i]
			}
		}
	}
}

// extractTaskNumber extracts the numeric part from a task filename
func extractTaskNumber(filename string) int {
	// task1.md -> 1, task2.md -> 2, etc.
	base := strings.TrimSuffix(filename, ".md")
	base = strings.TrimPrefix(base, "task")

	var num int
	fmt.Sscanf(base, "%d", &num)
	return num
}

// printExecutionSummary prints a summary of the execution results
func printExecutionSummary(result *executor.ExecutionJobResult) {
	separator := strings.Repeat("=", 60)
	fmt.Printf("\n%s\n", separator)
	fmt.Println("Execution Summary")
	fmt.Printf("%s\n", separator)
	fmt.Printf("Job ID:           %s\n", result.JobID)
	fmt.Printf("Status:           %s\n", result.Status)
	fmt.Printf("Duration:         %v\n", result.Duration())
	fmt.Printf("Total Tasks:      %d\n", result.TotalTasks)
	fmt.Printf("Successful Tasks: %d\n", result.SuccessfulTasks)
	fmt.Printf("Failed Tasks:     %d\n", result.FailedTasks)
	fmt.Println()

	if len(result.TaskResults) > 0 {
		fmt.Println("Task Details:")
		for i, tr := range result.TaskResults {
			status := "✓"
			if tr.Status != "success" {
				status = "✗"
			}
			fmt.Printf("  [%d] %s %s (%s, %d attempts)\n",
				i+1, status, tr.TaskID, tr.Status, tr.TotalAttempts)
			if tr.Status != "success" && tr.LastError != "" {
				fmt.Printf("       Error: %s\n", tr.LastError)
			}
		}
		fmt.Println()
	}

	fmt.Printf("%s\n", separator)
}

// commitDoingResults commits the execution results to git
func commitDoingResults(jobID string, result *executor.ExecutionJobResult) error {
	// Get current working directory
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current working directory: %w", err)
	}

	// Create git manager
	gm := git.New(cwd)

	// Create auto committer
	ac := git.NewAutoCommitter(gm)

	// Generate commit message based on execution result
	var commitMsg string
	if result.Status == "completed" {
		commitMsg = fmt.Sprintf("morty: doing %s - COMPLETED\n\nAll %d tasks executed successfully.\n\nCo-Authored-By: Claude Haiku 4.5 <noreply@anthropic.com>",
			jobID, result.TotalTasks)
	} else if result.Status == "partial" {
		commitMsg = fmt.Sprintf("morty: doing %s - PARTIAL\n\n%d/%d tasks completed successfully.\n\nCo-Authored-By: Claude Haiku 4.5 <noreply@anthropic.com>",
			jobID, result.SuccessfulTasks, result.TotalTasks)
	} else {
		commitMsg = fmt.Sprintf("morty: doing %s - FAILED\n\n%d/%d tasks failed.\n\nCo-Authored-By: Claude Haiku 4.5 <noreply@anthropic.com>",
			jobID, result.FailedTasks, result.TotalTasks)
	}

	// Check if there are any changes to commit
	hasChanges, err := ac.HasChanges()
	if err != nil {
		return fmt.Errorf("failed to check for changes: %w", err)
	}

	if !hasChanges {
		if GetVerbose() {
			fmt.Println("[INFO] No changes to commit")
		}
		return nil
	}

	// Add all files before committing
	if err := ac.AutoAddAndCommitJob(jobID, commitMsg); err != nil {
		return fmt.Errorf("failed to commit changes: %w", err)
	}

	return nil
}

// ensureGitUserConfigured ensures Git user is configured for the repository
// Reads user.name and user.email from global config (~/.rick/config.json)
func ensureGitUserConfigured(projectRoot string) error {
	// Load global config
	cfg, err := config.LoadConfig()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Check if user.name is configured in the repository
	cmd := exec.Command("git", "config", "user.name")
	cmd.Dir = projectRoot
	if output, err := cmd.Output(); err != nil || strings.TrimSpace(string(output)) == "" {
		// Set user.name from global config
		userName := cfg.Git.UserName
		if userName == "" {
			if projectName, err := workspace.GetProjectName(); err == nil && projectName != "" {
				userName = projectName
			} else {
				userName = "Rick"
			}
		}
		cmd = exec.Command("git", "config", "user.name", userName)
		cmd.Dir = projectRoot
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to set git user.name: %w", err)
		}
		if GetVerbose() {
			fmt.Printf("[INFO] Set git user.name to '%s'\n", userName)
		}
	}

	// Check if user.email is configured in the repository
	cmd = exec.Command("git", "config", "user.email")
	cmd.Dir = projectRoot
	if output, err := cmd.Output(); err != nil || strings.TrimSpace(string(output)) == "" {
		// Set user.email from global config
		userEmail := cfg.Git.UserEmail
		if userEmail == "" {
			userEmail = "rick@localhost" // Fallback default
		}
		cmd = exec.Command("git", "config", "user.email", userEmail)
		cmd.Dir = projectRoot
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to set git user.email: %w", err)
		}
		if GetVerbose() {
			fmt.Printf("[INFO] Set git user.email to '%s'\n", userEmail)
		}
	}

	return nil
}

// ensureGitInitialized checks if Git is initialized in the project root directory
// and initializes it if not present
func ensureGitInitialized(rickDir string) error {
	// Get current working directory (project root)
	projectRoot, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	// Check if .git directory exists in project root
	gitDir := filepath.Join(projectRoot, ".git")
	if _, err := os.Stat(gitDir); err == nil {
		// Git already initialized
		if GetVerbose() {
			fmt.Println("[INFO] Git repository already initialized in project root")
		}
		return nil
	}

	// Initialize Git repository in project root
	if GetVerbose() {
		fmt.Printf("[INFO] Initializing Git repository in project root: %s\n", projectRoot)
	}

	cmd := exec.Command("git", "init")
	cmd.Dir = projectRoot
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to run git init: %w\nOutput: %s", err, string(output))
	}

	// Configure Git user if not already configured
	if err := ensureGitUserConfigured(projectRoot); err != nil {
		return fmt.Errorf("failed to configure git user: %w", err)
	}

	// Create initial .gitignore if it doesn't exist
	gitignorePath := filepath.Join(projectRoot, ".gitignore")
	if _, err := os.Stat(gitignorePath); os.IsNotExist(err) {
		content := "# Project gitignore\n*.log\n.DS_Store\n"
		if err := os.WriteFile(gitignorePath, []byte(content), 0644); err != nil {
			return fmt.Errorf("failed to create .gitignore: %w", err)
		}
		if GetVerbose() {
			fmt.Println("[INFO] Created .gitignore file")
		}
	}

	fmt.Printf("✅ Git repository initialized in project root: %s\n", projectRoot)
	return nil
}
