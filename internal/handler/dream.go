package handler

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/sunquan/rick/internal/builder"
	"github.com/sunquan/rick/internal/config"
	"github.com/sunquan/rick/internal/runtime"
	"github.com/sunquan/rick/internal/workspace"
)

// Dream executes the cross-job global reflection workflow (migrated from
// cmd.dreamWorkflow). The pending-job scan/filtering lives in workspace.
func Dream(jobNum int, background bool, opts Options) error {
	fmt.Println("\n=== Dream Workflow ===")

	rickDir, err := workspace.GetRickDir()
	if err != nil {
		return fmt.Errorf("failed to get rick directory: %w", err)
	}

	// Ensure dream directory exists (agent writes log files here)
	dreamDir := filepath.Join(rickDir, workspace.DreamDirName)
	if err := os.MkdirAll(dreamDir, 0755); err != nil {
		return fmt.Errorf("failed to create dream directory: %w", err)
	}

	jobIDs := workspace.SelectPendingJobs(rickDir, jobNum)
	if len(jobIDs) == 0 {
		fmt.Println("No new completed jobs to process.")
		fmt.Println("All jobs have already been dreamed, or no jobs are fully completed yet.")
		return nil
	}

	fmt.Printf("Processing %d job(s): %s\n", len(jobIDs), strings.Join(jobIDs, ", "))

	promptFile, _, err := builder.NewPIBuilder().SaveDreamPrompt(jobIDs, rickDir)
	if err != nil {
		return fmt.Errorf("failed to generate dream prompt: %w", err)
	}

	cfg, err := config.LoadConfig()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	if background {
		fmt.Println("🤖 Starting pi (background mode)...")
		if err := runtime.CallCLI(opts.Verbose, cfg, promptFile, runtime.ModePrint); err != nil {
			return fmt.Errorf("pi failed: %w", err)
		}
	} else {
		fmt.Println("🤖 Starting pi (interactive mode)...")
		if err := runtime.CallCLI(opts.Verbose, cfg, promptFile, runtime.ModeInteractive); err != nil {
			return fmt.Errorf("pi failed: %w", err)
		}
	}

	fmt.Println("\n✅ Dream phase completed!")
	fmt.Println("Agent should have written dream_run_{job_id}_log.md for each processed job.")
	return nil
}

// DreamDryRun generates and prints the dream prompt (and the pending-job list)
// without executing it (migrated from cmd.runDreamDryRun).
func DreamDryRun(jobNum int) error {
	rickDir, err := workspace.GetRickDir()
	if err != nil {
		fmt.Printf("[DRY-RUN] failed to get rick dir: %v\n", err)
		return nil
	}

	jobIDs := workspace.SelectPendingJobs(rickDir, jobNum)
	fmt.Printf("[DRY-RUN] Pending jobs (%d): %v\n", len(jobIDs), jobIDs)

	_, content, err := builder.NewPIBuilder().BuildDream(map[string]string{
		"rick_dir": rickDir,
		"job_ids":  strings.Join(jobIDs, ","),
	})
	if err != nil {
		fmt.Printf("[DRY-RUN] failed to generate dream prompt: %v\n", err)
		return nil
	}

	fmt.Printf("[DRY-RUN] Dream prompt:\n\n")
	fmt.Print(content)
	return nil
}
