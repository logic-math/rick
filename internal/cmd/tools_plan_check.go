package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/sunquan/rick/internal/agent/piagent"
	"github.com/sunquan/rick/internal/executor"
	"github.com/sunquan/rick/internal/parser"
	"github.com/sunquan/rick/internal/workspace"
)

// NewPlanCheckCmd creates the plan_check subcommand
func NewPlanCheckCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "plan_check <job_id>",
		Short: "Validate the plan directory structure for a job",
		Long: `Check the plan directory structure for a job to ensure it is valid for execution.

Arguments:
  job_id    Job identifier (e.g. job_1)

Checks performed:
  - plan/ directory exists
  - at least one task*.md file present
  - each task has required sections: 依赖关系, 任务名称, 任务目标, 关键结果, 测试方法
  - all dependency references exist
  - no circular dependencies

Output:
  ✅ plan check passed: N tasks, dependencies valid
  ❌ plan check failed: <error description>

Exit codes:
  0  all checks passed
  1  one or more checks failed`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			jobID := args[0]

			planDir, err := workspace.GetJobPlanDir(jobID)
			if err != nil {
				return fmt.Errorf("failed to resolve plan directory: %w", err)
			}

			const maxAutoFixAttempts = 3
			var lastErr error

			for attempt := 0; attempt <= maxAutoFixAttempts; attempt++ {
				checkErr := runPlanCheck(planDir)
				if checkErr == nil {
					return nil
				}
				lastErr = checkErr

				if attempt == maxAutoFixAttempts {
					break
				}

				// Auto-fix: ensure pi binary is available
				if _, findErr := piagent.FindBinary(nil); findErr != nil {
					// No pi available; skip auto-fix
					break
				}

				promptFile, writeErr := writePlanCheckFixPrompt(planDir, checkErr)
				if writeErr != nil {
					break
				}
				defer os.Remove(promptFile)

				if fixErr := piagent.CallCLI(GetVerbose(), nil, promptFile, piagent.ModePrint); fixErr != nil {
					break
				}
			}

			fmt.Fprintf(os.Stderr, "❌ plan check failed: %v\n", lastErr)
			os.Exit(1)
			return nil
		},
	}
}

// runPlanCheck performs all structural checks on the plan directory.
// Returns nil on success, or an error describing the first failure.
func runPlanCheck(planDir string) error {
	// 1. plan/ directory exists
	info, err := os.Stat(planDir)
	if err != nil || !info.IsDir() {
		return fmt.Errorf("plan directory does not exist: %s", planDir)
	}

	// 2. At least one task*.md file
	entries, err := os.ReadDir(planDir)
	if err != nil {
		return fmt.Errorf("failed to read plan directory: %w", err)
	}

	var taskFiles []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasPrefix(e.Name(), "task") && strings.HasSuffix(e.Name(), ".md") {
			taskFiles = append(taskFiles, e.Name())
		}
	}

	if len(taskFiles) == 0 {
		return fmt.Errorf("no task*.md files found in %s", planDir)
	}

	// 3. Parse each task file and validate required sections
	tasks := make([]*parser.Task, 0, len(taskFiles))
	taskIDs := make(map[string]bool)

	for _, filename := range taskFiles {
		taskID := strings.TrimSuffix(filename, ".md")
		taskIDs[taskID] = true
	}

	for _, filename := range taskFiles {
		taskID := strings.TrimSuffix(filename, ".md")
		filePath := filepath.Join(planDir, filename)

		// Verify file is inside plan directory (path constraint)
		absFile, err := filepath.Abs(filePath)
		if err != nil {
			return fmt.Errorf("failed to resolve path for %s: %w", filename, err)
		}
		absPlan, err := filepath.Abs(planDir)
		if err != nil {
			return fmt.Errorf("failed to resolve plan dir path: %w", err)
		}
		if !strings.HasPrefix(absFile, absPlan+string(filepath.Separator)) {
			return fmt.Errorf("task file %s is outside plan directory", filename)
		}

		content, err := os.ReadFile(filePath)
		if err != nil {
			return fmt.Errorf("failed to read %s: %w", filename, err)
		}

		// Check required sections
		requiredSections := []string{
			"# 依赖关系",
			"# 任务名称",
			"# 任务目标",
			"# 关键结果",
			"# 测试方法",
		}
		for _, section := range requiredSections {
			if !strings.Contains(string(content), section) {
				return fmt.Errorf("task %s is missing required section: %s", filename, section)
			}
		}

		task, err := parser.ParseTask(string(content))
		if err != nil {
			return fmt.Errorf("failed to parse %s: %w", filename, err)
		}
		task.ID = taskID

		// 4. Dependency references exist
		for _, dep := range task.Dependencies {
			if !taskIDs[dep] {
				return fmt.Errorf("task %s depends on %s, but %s.md does not exist", taskID, dep, dep)
			}
		}

		tasks = append(tasks, task)
	}

	// 5. No circular dependencies (reuse executor.NewDAG)
	_, err = executor.NewDAG(tasks)
	if err != nil {
		return fmt.Errorf("dependency check failed: %w", err)
	}

	fmt.Printf("✅ plan check passed: %d tasks, dependencies valid\n", len(tasks))
	return nil
}

// fileHasMeaningfulContent returns true if the file has content beyond markdown
// structural lines (headers starting with #, table rows starting with |, empty lines).
func fileHasMeaningfulContent(path string) bool {
	content, err := os.ReadFile(path)
	if err != nil {
		return false
	}
	meaningful := 0
	for _, line := range strings.Split(string(content), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, "|") {
			continue
		}
		meaningful += len(line)
	}
	return meaningful >= 10
}

// (All agent invocation goes through piagent.CallCLI / piagent.FindBinary.)

// writePlanCheckFixPrompt writes a prompt file asking pi to fix the plan check errors.
// Returns the path to the temporary prompt file.
func writePlanCheckFixPrompt(planDir string, checkErr error) (string, error) {
	tmpFile, err := os.CreateTemp("", "rick-plan-check-fix-*.md")
	if err != nil {
		return "", fmt.Errorf("failed to create temp prompt file: %w", err)
	}
	defer tmpFile.Close()

	prompt := fmt.Sprintf(`# Fix Plan Check Errors

The following errors were found in the plan directory: %s

## Errors

%v

## Instructions

Please fix the above errors in the plan directory. Make sure:
1. All task*.md files exist in the plan directory
2. Each task*.md contains the required sections: # 依赖关系, # 任务名称, # 任务目标, # 关键结果, # 测试方法
3. All dependency references point to existing task files
4. There are no circular dependencies between tasks
Fix the files in place without changing the task content, only adding missing sections or correcting dependency references.
`, planDir, checkErr)

	if _, err := tmpFile.WriteString(prompt); err != nil {
		return "", fmt.Errorf("failed to write prompt file: %w", err)
	}

	return tmpFile.Name(), nil
}
