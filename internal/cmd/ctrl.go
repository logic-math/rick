package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/sunquan/rick/internal/agent/piagent"
	"github.com/sunquan/rick/internal/config"
	"github.com/sunquan/rick/internal/prompt"
	"github.com/sunquan/rick/internal/workspace"
)

func NewCtrlCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "ctrl",
		Short: "Monitor and intervene in a background doing session",
		Long:  `Launch an interactive pi agent that monitors doing progress and applies human interventions via task.md and tasks.json.`,
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if GetDryRun() {
				return runCtrlDryRun()
			}
			return runCtrl(cmd, args)
		},
	}
}

func runCtrlDryRun() error {
	jID := GetJobID()
	if jID == "" {
		return fmt.Errorf("--job flag is required (e.g. rick ctrl --dry-run --job job_1)")
	}
	rickDir, err := workspace.GetRickDir()
	if err != nil {
		return fmt.Errorf("failed to get rick directory: %w", err)
	}
	promptFile, err := prompt.GenerateCtrlPromptFile(jID, rickDir)
	if err != nil {
		return fmt.Errorf("failed to generate ctrl prompt: %w", err)
	}
	data, err := os.ReadFile(promptFile)
	if err != nil {
		return fmt.Errorf("failed to read ctrl prompt: %w", err)
	}
	fmt.Printf("[DRY-RUN] ctrl prompt for %s:\n\n", jID)
	fmt.Print(string(data))
	return nil
}

func runCtrl(cmd *cobra.Command, args []string) error {
	jID := GetJobID()
	if jID == "" {
		return fmt.Errorf("--job flag is required (e.g. rick ctrl --job job_1)")
	}

	rickDir, err := workspace.GetRickDir()
	if err != nil {
		return fmt.Errorf("failed to get rick directory: %w", err)
	}

	promptFile, err := prompt.GenerateCtrlPromptFile(jID, rickDir)
	if err != nil {
		return fmt.Errorf("failed to generate ctrl prompt: %w", err)
	}

	cfg, err := config.LoadConfig()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	fmt.Printf("Job: %s\n", jID)
	fmt.Println("🎮 Starting ctrl interactive session...")

	if err := piagent.CallCLI(GetVerbose(), cfg, promptFile, piagent.ModeInteractive); err != nil {
		return fmt.Errorf("ctrl session failed: %w", err)
	}

	return nil
}
