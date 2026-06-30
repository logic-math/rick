package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/sunquan/rick/internal/workspace"
)

// NewLearningCheckCmd creates the learning_check subcommand
func NewLearningCheckCmd() *cobra.Command {
	var autoFix bool

	cmd := &cobra.Command{
		Use:   "learning_check <job_id>",
		Short: "Validate the learning directory structure for a job",
		Long: `Check the learning directory structure for a job to ensure it is complete and well-formed.

Arguments:
  job_id    Job identifier (e.g. job_1)

Checks performed:
  - learning/SUMMARY.md exists and contains a "# Job" heading
  - .rick/loops/*.md: frontmatter (name, trigger) + 5 sections (目标/上下文管理/可调用工具/产出评估/停止标准)
  - .rick/skills/*.md: frontmatter (name, description) + 4 sections (When to Use/Procedure/Pitfalls/Verification)
  - README.md in each dir is skipped

Output:
  ✅ learning check passed
  ❌ learning check failed: <error description>

Exit codes:
  0  all checks passed
  1  one or more checks failed`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			jobID := args[0]

			learningDir, err := workspace.GetJobLearningDir(jobID)
			if err != nil {
				return fmt.Errorf("failed to resolve learning directory: %w", err)
			}

			if !autoFix {
				// No auto-fix: run once and report
				if checkErr := runLearningCheck(learningDir); checkErr != nil {
					fmt.Fprintf(os.Stderr, "❌ learning check failed: %v\n", checkErr)
					os.Exit(1)
				}
				return nil
			}

			const maxAutoFixAttempts = 3
			var lastErr error

			for attempt := 0; attempt <= maxAutoFixAttempts; attempt++ {
				checkErr := runLearningCheck(learningDir)
				if checkErr == nil {
					return nil
				}
				lastErr = checkErr

				if attempt == maxAutoFixAttempts {
					break
				}

				claudePath, findErr := findClaudeBinary()
				if findErr != nil {
					break
				}

				promptFile, writeErr := writeLearningCheckFixPrompt(learningDir, checkErr)
				if writeErr != nil {
					break
				}
				defer os.Remove(promptFile)

				if fixErr := runAutoFix(claudePath, promptFile); fixErr != nil {
					break
				}
			}

			fmt.Fprintf(os.Stderr, "❌ learning check failed: %v\n", lastErr)
			os.Exit(1)
			return nil
		},
	}

	cmd.Flags().BoolVar(&autoFix, "auto-fix", false, "Attempt to auto-fix errors using Claude")
	return cmd
}

// runLearningCheck checks that SUMMARY.md exists in the learning directory.
func runLearningCheck(learningDir string) error {
	summaryPath := filepath.Join(learningDir, "SUMMARY.md")
	if _, err := os.Stat(summaryPath); os.IsNotExist(err) {
		return fmt.Errorf("SUMMARY.md not found in %s", learningDir)
	}
	summaryContent, err := os.ReadFile(summaryPath)
	if err != nil {
		return fmt.Errorf("failed to read SUMMARY.md: %w", err)
	}
	if len(strings.TrimSpace(string(summaryContent))) == 0 || !strings.Contains(string(summaryContent), "# Job") {
		return fmt.Errorf("SUMMARY.md exists but is empty or missing required '# Job' heading")
	}

	rickDir, err := workspace.GetRickDir()
	if err != nil {
		return fmt.Errorf("failed to resolve rick dir for loops/skills check: %w", err)
	}
	lsErrs := runLoopsAndSkillsCheck(rickDir)
	if len(lsErrs) > 0 {
		return fmt.Errorf("loops/skills check errors:\n  - %s", strings.Join(lsErrs, "\n  - "))
	}

	fmt.Printf("✅ learning check passed\n")
	return nil
}

// writeLearningCheckFixPrompt writes a prompt file asking claude to fix SUMMARY.md.
func writeLearningCheckFixPrompt(learningDir string, checkErr error) (string, error) {
	tmpFile, err := os.CreateTemp("", "rick-learning-check-fix-*.md")
	if err != nil {
		return "", fmt.Errorf("failed to create temp prompt file: %w", err)
	}
	defer tmpFile.Close()

	content := fmt.Sprintf(`# Fix Learning Check Errors

The following error was found in the learning directory: %s

## Error

%v

## Instructions

Please fix the above error. Ensure SUMMARY.md exists in %s, is non-empty, and contains a "# Job" heading summarizing the job execution.
`, learningDir, checkErr, learningDir)

	if _, err := tmpFile.WriteString(content); err != nil {
		return "", fmt.Errorf("failed to write prompt file: %w", err)
	}

	return tmpFile.Name(), nil
}
