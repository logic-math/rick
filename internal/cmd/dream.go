package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
	"github.com/sunquan/rick/internal/config"
	"github.com/sunquan/rick/internal/executor"
	"github.com/sunquan/rick/internal/prompt"
	"github.com/sunquan/rick/internal/workspace"
)

func NewDreamCmd() *cobra.Command {
	var jobNum int
	var background bool

	dreamCmd := &cobra.Command{
		Use:   "dream",
		Short: "Cross-job global reflection and skill evolution",
		Long:  `Perform cross-job global reflection, evolve skills, and maintain .rick knowledge base.`,
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if GetDryRun() {
				return runDreamDryRun(jobNum)
			}
			return dreamWorkflow(jobNum, background)
		},
	}

	dreamCmd.Flags().IntVar(&jobNum, "job_num", 5, "Number of jobs to process per dream run")
	dreamCmd.Flags().BoolVarP(&background, "background", "p", false, "Run in background (non-interactive, skip-permissions)")

	return dreamCmd
}

func runDreamDryRun(jobNum int) error {
	rickDir, err := workspace.GetRickDir()
	if err != nil {
		fmt.Printf("[DRY-RUN] failed to get rick dir: %v\n", err)
		return nil
	}

	jobIDs := selectPendingJobs(rickDir, jobNum)
	fmt.Printf("[DRY-RUN] Pending jobs (%d): %v\n", len(jobIDs), jobIDs)

	content, err := prompt.GenerateDreamPrompt(jobIDs, rickDir)
	if err != nil {
		fmt.Printf("[DRY-RUN] failed to generate dream prompt: %v\n", err)
		return nil
	}

	fmt.Printf("[DRY-RUN] Dream prompt:\n\n")
	fmt.Print(content)
	return nil
}

func dreamWorkflow(jobNum int, background bool) error {
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

	jobIDs := selectPendingJobs(rickDir, jobNum)
	if len(jobIDs) == 0 {
		fmt.Println("No new completed jobs to process.")
		fmt.Println("All jobs have already been dreamed, or no jobs are fully completed yet.")
		return nil
	}

	fmt.Printf("Processing %d job(s): %s\n", len(jobIDs), strings.Join(jobIDs, ", "))

	promptFile, _, err := prompt.GenerateDreamPromptFile(jobIDs, rickDir)
	if err != nil {
		return fmt.Errorf("failed to generate dream prompt: %w", err)
	}

	cfg, err := config.LoadConfig()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	if background {
		fmt.Println("🤖 Starting Claude Code CLI (background mode)...")
		if err := callClaudeCodeCLIBackground(cfg, promptFile); err != nil {
			return fmt.Errorf("Claude Code CLI failed: %w", err)
		}
	} else {
		fmt.Println("🤖 Starting Claude Code CLI (interactive mode)...")
		if err := callClaudeCodeCLI(cfg, promptFile); err != nil {
			return fmt.Errorf("Claude Code CLI failed: %w", err)
		}
	}

	fmt.Println("\n✅ Dream phase completed!")
	fmt.Println("Agent should have written dream_run_{job_id}_log.md for each processed job.")
	return nil
}

// selectPendingJobs returns up to jobNum completed jobs not yet processed by dream.
func selectPendingJobs(rickDir string, jobNum int) []string {
	completed := discoverCompletedJobs(rickDir)
	processed := getDreamProcessedJobs(rickDir)

	var pending []string
	for _, id := range completed {
		if !processed[id] {
			pending = append(pending, id)
		}
	}

	if len(pending) > jobNum {
		pending = pending[:jobNum]
	}
	return pending
}

// getDreamProcessedJobs returns the set of job IDs that already have a
// dream_run_{job_id}_log.md file in .rick/dream/.
func getDreamProcessedJobs(rickDir string) map[string]bool {
	processed := make(map[string]bool)
	dreamDir := filepath.Join(rickDir, workspace.DreamDirName)
	entries, err := os.ReadDir(dreamDir)
	if err != nil {
		return processed
	}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		// Expected format: dream_run_job_N_log.md
		if !strings.HasPrefix(name, "dream_run_") || !strings.HasSuffix(name, "_log.md") {
			continue
		}
		jobID := strings.TrimPrefix(name, "dream_run_")
		jobID = strings.TrimSuffix(jobID, "_log.md")
		if strings.HasPrefix(jobID, "job_") {
			processed[jobID] = true
		}
	}
	return processed
}

// discoverCompletedJobs scans .rick/jobs/*/doing/tasks.json and returns
// jobs where all tasks have status "success", sorted by job number ascending.
func discoverCompletedJobs(rickDir string) []string {
	pattern := filepath.Join(rickDir, "jobs", "job_*", "doing", "tasks.json")
	files, err := filepath.Glob(pattern)
	if err != nil {
		return nil
	}

	var completed []string
	for _, f := range files {
		tj, err := executor.LoadTasksJSON(f)
		if err != nil {
			continue
		}
		tasks := tj.GetAllTasks()
		if len(tasks) == 0 {
			continue
		}
		allDone := true
		for _, t := range tasks {
			if t.Status != "success" {
				allDone = false
				break
			}
		}
		if allDone {
			// Extract job ID from path: .rick/jobs/job_N/doing/tasks.json
			jobID := filepath.Base(filepath.Dir(filepath.Dir(f)))
			completed = append(completed, jobID)
		}
	}

	sort.Slice(completed, func(i, j int) bool {
		return jobNumber(completed[i]) < jobNumber(completed[j])
	})
	return completed
}

// jobNumber extracts the numeric part from "job_N".
func jobNumber(jobID string) int {
	n, _ := strconv.Atoi(strings.TrimPrefix(jobID, "job_"))
	return n
}

