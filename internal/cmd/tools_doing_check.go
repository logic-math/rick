package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/sunquan/rick/internal/executor"
	"github.com/sunquan/rick/internal/runtime"
	"github.com/sunquan/rick/internal/workspace"
)

// NewDoingCheckCmd creates the doing_check subcommand
func NewDoingCheckCmd() *cobra.Command {
	var autoFix bool

	cmd := &cobra.Command{
		Use:   "doing_check <job_id>",
		Short: "Validate the doing directory structure for a job",
		Long: `Check the doing directory structure for a job to ensure it completed correctly.

Arguments:
  job_id    Job identifier (e.g. job_1)

Checks performed:
  - tasks.json exists and is parseable
  - debug.md exists (mandatory work log)
  - no tasks in "running" zombie state
  - all success tasks have a non-empty commit_hash

Output:
  ✅ doing check passed: N/N tasks succeeded
  ❌ doing check failed: <error description>

Exit codes:
  0  all checks passed
  1  one or more checks failed`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			jobID := args[0]

			doingDir, err := workspace.GetJobDoingDir(jobID)
			if err != nil {
				return fmt.Errorf("failed to resolve doing directory: %w", err)
			}

			if !autoFix {
				// No auto-fix: run once and report
				if checkErr := runDoingCheck(doingDir); checkErr != nil {
					fmt.Fprintf(os.Stderr, "❌ doing check failed: %v\n", checkErr)
					os.Exit(1)
				}
				return nil
			}

			const maxAutoFixAttempts = 3
			var lastErr error

			for attempt := 0; attempt <= maxAutoFixAttempts; attempt++ {
				checkErr := runDoingCheck(doingDir)
				if checkErr == nil {
					return nil
				}
				lastErr = checkErr

				if attempt == maxAutoFixAttempts {
					break
				}

				if _, findErr := runtime.FindBinary(nil); findErr != nil {
					break
				}

				promptFile, writeErr := writeDoingCheckFixPrompt(doingDir, checkErr)
				if writeErr != nil {
					break
				}
				defer os.Remove(promptFile)

				if fixErr := runtime.CallCLI(GetVerbose(), nil, promptFile, runtime.ModePrint); fixErr != nil {
					break
				}
			}

			fmt.Fprintf(os.Stderr, "❌ doing check failed: %v\n", lastErr)
			os.Exit(1)
			return nil
		},
	}

	cmd.Flags().BoolVar(&autoFix, "auto-fix", false, "Attempt to auto-fix errors using pi")
	return cmd
}

// NewEasyCheckCmd creates the easy_check subcommand (debug/ format only, no task checks).
func NewEasyCheckCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "easy_check <job_id>",
		Short: "Validate the doing directory structure for an easy mode job",
		Long: `Check the doing directory for an easy mode job (debug/ format only, no task checks).

Arguments:
  job_id    Job identifier (e.g. job_1)

Checks performed:
  - debug/bug*.md file name format
  - Required # sections and ## attempt blocks inside each bug file
  - frontmatter status field present and not "🔄 进行中"

Exit codes:
  0  all checks passed
  1  one or more checks failed`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			jobID := args[0]
			doingDir, err := workspace.GetJobDoingDir(jobID)
			if err != nil {
				return fmt.Errorf("failed to resolve doing directory: %w", err)
			}
			if checkErr := executor.RunEasyCheck(doingDir); checkErr != nil {
				fmt.Fprintf(os.Stderr, "❌ easy check failed: %v\n", checkErr)
				os.Exit(1)
			}
			fmt.Println("✅ easy check passed")
			return nil
		},
	}
	return cmd
}

// runDoingCheck performs all structural checks on the doing directory.
func runDoingCheck(doingDir string) error {
	if err := executor.RunDoingCheck(doingDir); err != nil {
		return err
	}
	tasksJSONPath := filepath.Join(doingDir, "tasks.json")
	tasksJSON, err := executor.LoadTasksJSON(tasksJSONPath)
	if err != nil {
		return err
	}

	for _, task := range tasksJSON.GetAllTasks() {
		if task.Status == "running" {
			return fmt.Errorf("task %s is in zombie 'running' state — change to 'failed' if stuck", task.TaskID)
		}
		if task.Status == "success" && task.CommitHash == "" {
			return fmt.Errorf("task %s has status=success but missing commit_hash", task.TaskID)
		}
	}

	successCount := tasksJSON.GetCompletedCount()
	totalCount := tasksJSON.GetTaskCount()
	fmt.Printf("✅ doing check passed: %d/%d tasks succeeded\n", successCount, totalCount)
	return nil
}

// writeDoingCheckFixPrompt writes a prompt file asking pi to fix doing check errors.
func writeDoingCheckFixPrompt(doingDir string, checkErr error) (string, error) {
	tmpFile, err := os.CreateTemp("", "rick-doing-check-fix-*.md")
	if err != nil {
		return "", fmt.Errorf("failed to create temp prompt file: %w", err)
	}
	defer tmpFile.Close()

	prompt := fmt.Sprintf(`# Fix Doing Check Errors

The following errors were found in the doing directory: %s

## Errors

%v

## Instructions

Please fix the above errors in the doing directory. Make sure:
1. tasks.json exists and is valid JSON with proper task states
2. debug.md exists, is non-empty, and contains at least one "## task" section recording the execution
3. No tasks are in "running" zombie state (change to "failed" if stuck)
4. All tasks with status="success" have a non-empty commit_hash field

Fix the metadata files in place without re-running tasks.
`, doingDir, checkErr)

	if _, err := tmpFile.WriteString(prompt); err != nil {
		return "", fmt.Errorf("failed to write prompt file: %w", err)
	}

	return tmpFile.Name(), nil
}
