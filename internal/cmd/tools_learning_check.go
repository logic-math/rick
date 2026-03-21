package cmd

import (
	"fmt"
	"os"
	"os/exec"
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
  - learning/SUMMARY.md exists
  - if learning/skills/*.py exist, each passes Python syntax check
  - if learning/OKR.md exists, it contains required sections
  - if learning/SPEC.md exists, it contains required sections

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

// runLearningCheck performs all structural checks on the learning directory.
func runLearningCheck(learningDir string) error {
	// 1. SUMMARY.md exists
	summaryPath := filepath.Join(learningDir, "SUMMARY.md")
	if _, err := os.Stat(summaryPath); os.IsNotExist(err) {
		return fmt.Errorf("SUMMARY.md not found in %s", learningDir)
	}

	// 2. If learning/skills/*.py exist, each must pass Python syntax check
	skillsDir := filepath.Join(learningDir, "skills")
	if info, err := os.Stat(skillsDir); err == nil && info.IsDir() {
		entries, err := os.ReadDir(skillsDir)
		if err != nil {
			return fmt.Errorf("failed to read skills directory: %w", err)
		}
		for _, e := range entries {
			if e.IsDir() || !strings.HasSuffix(e.Name(), ".py") {
				continue
			}
			pyFile := filepath.Join(skillsDir, e.Name())
			checkCmd := exec.Command("python3", "-c",
				fmt.Sprintf("import ast; ast.parse(open(%q).read())", pyFile))
			if out, err := checkCmd.CombinedOutput(); err != nil {
				return fmt.Errorf("Python syntax error in skills/%s: %s", e.Name(), strings.TrimSpace(string(out)))
			}
		}
	}

	// 3. If learning/OKR.md exists, check required sections
	okrPath := filepath.Join(learningDir, "OKR.md")
	if _, err := os.Stat(okrPath); err == nil {
		if checkErr := checkOKRSections(okrPath); checkErr != nil {
			return checkErr
		}
	}

	// 4. If learning/SPEC.md exists, check required sections
	specPath := filepath.Join(learningDir, "SPEC.md")
	if _, err := os.Stat(specPath); err == nil {
		if checkErr := checkSPECSections(specPath); checkErr != nil {
			return checkErr
		}
	}

	fmt.Printf("✅ learning check passed\n")
	return nil
}

// checkOKRSections verifies OKR.md contains the required sections.
func checkOKRSections(okrPath string) error {
	content, err := os.ReadFile(okrPath)
	if err != nil {
		return fmt.Errorf("failed to read OKR.md: %w", err)
	}
	text := string(content)

	// Must contain a line starting with "## O"
	hasObjective := false
	for _, line := range strings.Split(text, "\n") {
		if strings.HasPrefix(strings.TrimSpace(line), "## O") {
			hasObjective = true
			break
		}
	}
	if !hasObjective {
		return fmt.Errorf("OKR.md is missing required section: ## O (objective heading)")
	}

	if !strings.Contains(text, "### 关键结果") {
		return fmt.Errorf("OKR.md is missing required section: ### 关键结果")
	}

	return nil
}

// checkSPECSections verifies SPEC.md contains the four required sections.
func checkSPECSections(specPath string) error {
	content, err := os.ReadFile(specPath)
	if err != nil {
		return fmt.Errorf("failed to read SPEC.md: %w", err)
	}
	text := string(content)

	requiredSections := []string{
		"## 技术栈",
		"## 架构设计",
		"## 开发规范",
		"## 工程实践",
	}
	for _, section := range requiredSections {
		if !strings.Contains(text, section) {
			return fmt.Errorf("SPEC.md is missing required section: %s", section)
		}
	}

	return nil
}

// writeLearningCheckFixPrompt writes a prompt file asking claude to fix learning check errors.
func writeLearningCheckFixPrompt(learningDir string, checkErr error) (string, error) {
	tmpFile, err := os.CreateTemp("", "rick-learning-check-fix-*.md")
	if err != nil {
		return "", fmt.Errorf("failed to create temp prompt file: %w", err)
	}
	defer tmpFile.Close()

	prompt := fmt.Sprintf(`# Fix Learning Check Errors

The following errors were found in the learning directory: %s

## Errors

%v

## Instructions

Please fix the above errors in the learning directory. Make sure:
1. SUMMARY.md exists with a summary of the job execution
2. Any .py files in skills/ are valid Python (fix syntax errors)
3. If OKR.md exists, it contains a "## O..." objective heading and "### 关键结果" section
4. If SPEC.md exists, it contains all four sections: ## 技术栈, ## 架构设计, ## 开发规范, ## 工程实践

Fix the documents and scripts in place, preserving existing content where possible.
`, learningDir, checkErr)

	if _, err := tmpFile.WriteString(prompt); err != nil {
		return "", fmt.Errorf("failed to write prompt file: %w", err)
	}

	return tmpFile.Name(), nil
}
