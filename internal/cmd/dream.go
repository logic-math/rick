package cmd

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/sunquan/rick/internal/config"
	"github.com/sunquan/rick/internal/prompt"
	"github.com/sunquan/rick/internal/workspace"
)

const dreamReadmeDefaultContent = "## 已处理 Jobs\n\n（空）\n\n## 待处理 Jobs\n"

func NewDreamCmd() *cobra.Command {
	dreamCmd := &cobra.Command{
		Use:   "dream",
		Short: "Cross-job global reflection and skill evolution",
		Long:  `Perform cross-job global reflection, evolve skills, and maintain .rick knowledge base.`,
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if GetDryRun() {
				return runDreamDryRun()
			}
			return dreamWorkflow()
		},
	}

	return dreamCmd
}

func runDreamDryRun() error {
	rickDir, err := workspace.GetRickDir()
	if err != nil {
		fmt.Printf("[DRY-RUN] failed to get rick dir: %v\n", err)
		return nil
	}

	if err := ensureDreamReadme(rickDir); err != nil {
		fmt.Printf("[DRY-RUN] failed to ensure readme.md: %v\n", err)
	}

	jobIDs := getPendingJobIDs(rickDir, 5)

	promptFile, err := prompt.GenerateDreamPromptFile(jobIDs, rickDir)
	if err != nil {
		fmt.Printf("[DRY-RUN] failed to generate dream prompt: %v\n", err)
		return nil
	}
	defer os.Remove(promptFile)

	content, err := os.ReadFile(promptFile)
	if err != nil {
		fmt.Printf("[DRY-RUN] failed to read prompt file: %v\n", err)
		return nil
	}

	fmt.Printf("[DRY-RUN] Dream prompt:\n\n")
	fmt.Print(string(content))
	return nil
}

func dreamWorkflow() error {
	fmt.Println("\n=== Dream Workflow ===")

	rickDir, err := workspace.GetRickDir()
	if err != nil {
		return fmt.Errorf("failed to get rick directory: %w", err)
	}

	if err := ensureDreamReadme(rickDir); err != nil {
		return fmt.Errorf("failed to ensure dream readme: %w", err)
	}

	jobIDs := getPendingJobIDs(rickDir, 5)
	if len(jobIDs) == 0 {
		fmt.Println("No pending jobs found in .rick/dream/readme.md")
		fmt.Println("Add job IDs under '## 待处理 Jobs' section to proceed.")
		return nil
	}

	fmt.Printf("Processing %d pending job(s): %s\n", len(jobIDs), strings.Join(jobIDs, ", "))

	promptFile, err := prompt.GenerateDreamPromptFile(jobIDs, rickDir)
	if err != nil {
		return fmt.Errorf("failed to generate dream prompt: %w", err)
	}
	defer os.Remove(promptFile)

	cfg, err := config.LoadConfig()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	fmt.Println("🤖 Starting Claude Code CLI for dream phase...")
	if err := callClaudeCodeCLI(cfg, promptFile); err != nil {
		return fmt.Errorf("Claude Code CLI failed: %w", err)
	}

	fmt.Println("\n✅ Dream phase completed!")
	return nil
}

// ensureDreamReadme creates .rick/dream/readme.md if it doesn't exist.
func ensureDreamReadme(rickDir string) error {
	dreamDir := filepath.Join(rickDir, workspace.DreamDirName)
	if err := os.MkdirAll(dreamDir, 0755); err != nil {
		return err
	}
	readmePath := filepath.Join(dreamDir, "readme.md")
	if _, err := os.Stat(readmePath); os.IsNotExist(err) {
		return os.WriteFile(readmePath, []byte(dreamReadmeDefaultContent), 0644)
	}
	return nil
}

// getPendingJobIDs reads .rick/dream/readme.md and returns up to maxCount pending job IDs.
func getPendingJobIDs(rickDir string, maxCount int) []string {
	readmePath := filepath.Join(rickDir, workspace.DreamDirName, "readme.md")
	f, err := os.Open(readmePath)
	if err != nil {
		return nil
	}
	defer f.Close()

	var jobIDs []string
	inPendingSection := false

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "## 待处理 Jobs") {
			inPendingSection = true
			continue
		}
		if inPendingSection {
			if strings.HasPrefix(line, "## ") {
				break
			}
			if strings.HasPrefix(line, "- ") {
				id := strings.TrimSpace(strings.TrimPrefix(line, "- "))
				if id != "" {
					jobIDs = append(jobIDs, id)
					if len(jobIDs) >= maxCount {
						break
					}
				}
			}
		}
	}
	return jobIDs
}
