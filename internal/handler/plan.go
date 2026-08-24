package handler

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/sunquan/rick/internal/builder"
	"github.com/sunquan/rick/internal/config"
	"github.com/sunquan/rick/internal/runtime"
	"github.com/sunquan/rick/internal/workspace"
)

// Plan executes the complete planning workflow for a new job (migrated from
// cmd.executePlanWorkflow). It loads config, ensures the workspace exists,
// allocates the next job ID, builds the plan prompt, and launches the
// interactive pi session.
func Plan(requirement string, opts Options) error {
	// Step 1: Load configuration and ensure workspace exists
	cfg, err := config.LoadConfig()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Ensure workspace exists (auto-create if needed)
	ws, err := workspace.New()
	if err != nil {
		return fmt.Errorf("failed to create workspace: %w", err)
	}

	rickDir, err := workspace.GetRickDir()
	if err != nil {
		return fmt.Errorf("failed to get rick directory: %w", err)
	}

	if opts.Verbose {
		fmt.Printf("[INFO] Using workspace: %s\n", rickDir)
		fmt.Printf("[INFO] Workspace initialized: %v\n", ws != nil)
	}

	// Step 2: Determine next job ID and create job/plan directory
	jobID, err := workspace.NextJobID()
	if err != nil {
		return fmt.Errorf("failed to determine next job ID: %w", err)
	}

	jobPlanDir, err := workspace.GetJobPlanDir(jobID)
	if err != nil {
		return fmt.Errorf("failed to get job plan directory: %w", err)
	}

	if err := os.MkdirAll(jobPlanDir, 0755); err != nil {
		return fmt.Errorf("failed to create job plan directory: %w", err)
	}

	if opts.Verbose {
		fmt.Printf("[INFO] Created job directory: %s\n", jobPlanDir)
	}

	fmt.Printf("Job ID: %s\n", jobID)
	fmt.Printf("Plan directory: %s\n", jobPlanDir)

	// Step 3: Generate planning prompt
	if opts.Verbose {
		fmt.Println("[INFO] Generating planning prompt...")
	}

	// Generate plan prompt and save to jobPlanDir/prompts/
	planPromptFile, _, err := builder.NewPIBuilder().SavePlanPrompt(requirement, jobPlanDir, rickDir)
	if err != nil {
		return fmt.Errorf("failed to generate plan prompt: %w", err)
	}

	if opts.Verbose {
		fmt.Printf("[INFO] Planning prompt saved to: %s\n", planPromptFile)
	}

	// Step 4: Call pi with planning prompt file (interactive mode)
	if opts.Verbose {
		fmt.Println("[INFO] Calling pi for planning...")
	}

	// v4.4.9: 会话持久化——首次生成 uuid（--session-id 创建），落盘 plan/session_id；
	// 后续 `rick plan --resume job_N` 读同一 id 恢复完整会话。
	sessionID, err := ensureSessionID(jobPlanDir)
	if err != nil {
		return fmt.Errorf("ensure plan session id: %w", err)
	}
	fmt.Printf("Session ID: %s\n", sessionID)

	if err := runtime.CallCLI(opts.Verbose, cfg, planPromptFile, runtime.ModeInteractive, "--session-id", sessionID); err != nil {
		return fmt.Errorf("failed to call pi: %w", err)
	}

	fmt.Printf("\nPlanning session completed! Job: %s\n", jobID)
	fmt.Println("Please review the generated task files and then run:")
	fmt.Printf("  rick doing %s\n", jobID)

	return nil
}

// ReEnterPlan re-enters a planning session for an existing job (migrated from
// cmd.reEnterPlanWorkflow).
func ReEnterPlan(existingJobID string, requirement string, opts Options) error {
	jobPlanDir, err := workspace.GetJobPlanDir(existingJobID)
	if err != nil {
		return fmt.Errorf("failed to get job plan directory: %w", err)
	}

	if _, err := os.Stat(jobPlanDir); os.IsNotExist(err) {
		return fmt.Errorf("job %s plan directory does not exist, use 'rick plan' to create a new job", existingJobID)
	}

	cfg, err := config.LoadConfig()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	rickDir, err := workspace.GetRickDir()
	if err != nil {
		return fmt.Errorf("failed to get rick directory: %w", err)
	}

	if opts.Verbose {
		fmt.Printf("[INFO] Re-entering plan for job: %s\n", existingJobID)
		fmt.Printf("[INFO] Plan directory: %s\n", jobPlanDir)
	}

	fmt.Printf("Job ID: %s\n", existingJobID)
	fmt.Printf("Plan directory: %s\n", jobPlanDir)

	planPromptFile, _, err := builder.NewPIBuilder().SavePlanPrompt(requirement, jobPlanDir, rickDir)
	if err != nil {
		return fmt.Errorf("failed to generate plan prompt: %w", err)
	}

	if opts.Verbose {
		fmt.Printf("[INFO] Planning prompt saved to: %s\n", planPromptFile)
	}

	// v4.4.9: --resume 恢复——plan/session_id 存在则恢复同一 pi 会话（完整
	// 历史与上下文：此前澄清过的设计树/流水线讨论都在）；无记录则新建并落盘。
	sessionID, err := ensureSessionID(jobPlanDir)
	if err != nil {
		return fmt.Errorf("ensure plan session id: %w", err)
	}
	fmt.Printf("Session ID: %s\n", sessionID)

	if err := runtime.CallCLI(opts.Verbose, cfg, planPromptFile, runtime.ModeInteractive, "--session-id", sessionID); err != nil {
		return fmt.Errorf("failed to call pi: %w", err)
	}

	fmt.Printf("\nPlanning session completed! Job: %s\n", existingJobID)
	fmt.Println("Please review the generated task files and then run:")
	fmt.Printf("  rick doing %s\n", existingJobID)

	return nil
}

// PlanDryRun generates and prints the plan prompt without executing it
// (migrated from cmd.runPlanDryRun). Used for inspection and testing.
func PlanDryRun() error {
	rickDir, err := workspace.GetRickDir()
	if err != nil {
		fmt.Printf("[DRY-RUN] failed to get rick dir: %v\n", err)
		return nil
	}

	jobPlanDir := filepath.Join(rickDir, "jobs", "job_N", "plan")
	_, promptContent, err := builder.NewPIBuilder().BuildPlan("dry-run requirement", map[string]string{
		"rick_dir":     rickDir,
		"job_plan_dir": jobPlanDir,
	})
	if err != nil {
		fmt.Printf("[DRY-RUN] failed to generate prompt: %v\n", err)
		return nil
	}

	fmt.Printf("[DRY-RUN] Plan prompt:\n\n")
	fmt.Println(promptContent)
	return nil
}
