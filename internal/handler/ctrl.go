package handler

import (
	"fmt"
	"os"

	"github.com/sunquan/rick/internal/builder"
	"github.com/sunquan/rick/internal/config"
	"github.com/sunquan/rick/internal/runtime"
	"github.com/sunquan/rick/internal/workspace"
)

// Ctrl launches the interactive ctrl monitoring/intervention session for a job
// (migrated from cmd.runCtrl).
func Ctrl(opts Options) error {
	jID := opts.JobID
	if jID == "" {
		return fmt.Errorf("--job flag is required (e.g. rick ctrl --job job_1)")
	}

	rickDir, err := workspace.GetRickDir()
	if err != nil {
		return fmt.Errorf("failed to get rick directory: %w", err)
	}

	promptFile, _, err := builder.NewPIBuilder().SaveCtrlPrompt(jID, rickDir)
	if err != nil {
		return fmt.Errorf("failed to generate ctrl prompt: %w", err)
	}

	cfg, err := config.LoadConfig()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	fmt.Printf("Job: %s\n", jID)
	fmt.Println("🎮 Starting ctrl interactive session...")

	if err := runtime.CallCLI(opts.Verbose, cfg, promptFile, runtime.ModeInteractive); err != nil {
		return fmt.Errorf("ctrl session failed: %w", err)
	}

	return nil
}

// CtrlDryRun generates and prints the ctrl prompt without starting a session
// (migrated from cmd.runCtrlDryRun).
func CtrlDryRun(opts Options) error {
	jID := opts.JobID
	if jID == "" {
		return fmt.Errorf("--job flag is required (e.g. rick ctrl --dry-run --job job_1)")
	}

	rickDir, err := workspace.GetRickDir()
	if err != nil {
		return fmt.Errorf("failed to get rick directory: %w", err)
	}

	promptFile, _, err := builder.NewPIBuilder().SaveCtrlPrompt(jID, rickDir)
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
