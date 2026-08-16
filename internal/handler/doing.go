package handler

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/sunquan/rick/internal/builder"
	"github.com/sunquan/rick/internal/config"
	"github.com/sunquan/rick/internal/runtime"
	"github.com/sunquan/rick/internal/workspace"
)

// Doing executes the complete doing workflow. Scheduling is now delegated to
// pi's workflowScript orchestration (parent single-writer + runs.run with await),
// and the gate is a deterministic rick-side script run after the pi session
// settles. rick's retry loop is only a safety net: it regenerates an
// orchestration of the remaining pending tasks on each attempt, bounded by
// cfg.MaxRetries.
func Doing(jobID string, opts Options) error {
	cfg, err := config.LoadConfig()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	rickDir, err := workspace.GetRickDir()
	if err != nil {
		return fmt.Errorf("failed to get rick directory: %w", err)
	}

	jobDir := filepath.Join(rickDir, "jobs", jobID)
	planDir := filepath.Join(jobDir, "plan")
	doingDir := filepath.Join(jobDir, "doing")

	if _, err := os.Stat(jobDir); os.IsNotExist(err) {
		return fmt.Errorf("job directory not found: %s", jobDir)
	}
	if _, err := os.Stat(planDir); os.IsNotExist(err) {
		return fmt.Errorf("plan directory not found: %s", planDir)
	}
	if err := os.MkdirAll(doingDir, 0755); err != nil {
		return fmt.Errorf("failed to create doing directory: %w", err)
	}

	if opts.Verbose {
		fmt.Printf("[INFO] Job directory: %s\n", jobDir)
		fmt.Printf("[INFO] Plan directory: %s\n", planDir)
		fmt.Printf("[INFO] Doing directory: %s\n", doingDir)
	}

	// Initial tasks.json draft: builder scans plan/task*.md (all pending).
	if err := builder.EnsureTasksJSON(doingDir, planDir); err != nil {
		return fmt.Errorf("failed to initialize tasks.json: %w", err)
	}

	maxRetries := cfg.MaxRetries
	if maxRetries <= 0 {
		maxRetries = 3
	}

	piRuntime := runtime.NewPiRuntime(cfg.PiPath, cfg.PiExtraArgs...)

	for attempt := 1; attempt <= maxRetries; attempt++ {
		if opts.Verbose {
			fmt.Printf("[INFO] Attempt %d/%d\n", attempt, maxRetries)
		}

		// builder regenerates an orchestration of only the remaining pending
		// tasks (completed tasks are filtered out at generation time).
		promptFile, method, _, err := builder.NewPIBuilder().SaveDoingPrompt(doingDir, planDir, rickDir, jobID)
		if err != nil {
			return fmt.Errorf("failed to build doing prompt: %w", err)
		}

		_, _, err = piRuntime.Run(method, promptFile, cfg)
		if err != nil {
			fmt.Printf("[WARN] pi run did not settle (attempt %d/%d): %v\n", attempt, maxRetries, err)
			if attempt < maxRetries {
				continue
			}
			return fmt.Errorf("pi run failed after %d attempts: %w", maxRetries, err)
		}

		// Deterministic gate after the session settles (agent_settled).
		if gateErr := runDoingGate(rickDir, doingDir); gateErr != nil {
			fmt.Printf("[WARN] gate failed (attempt %d/%d): %v\n", attempt, maxRetries, gateErr)
			if attempt < maxRetries {
				continue
			}
			return fmt.Errorf("doing gate failed after %d attempts: %w", maxRetries, gateErr)
		}

		fmt.Printf("Job %s execution completed!\n", jobID)
		return nil
	}

	return fmt.Errorf("job execution incomplete after %d attempts", maxRetries)
}

// runDoingGate runs the deterministic rick-side gate script:
// python3 .rick/skills/rick-gates/helper.py <doingDir>. Exit non-zero = gate
// failure (unparseable tasks.json / zombie running / success without commit_hash).
func runDoingGate(rickDir, doingDir string) error {
	helper := filepath.Join(rickDir, "skills", "rick-gates", "helper.py")
	cmd := exec.Command("python3", helper, doingDir)
	out, err := cmd.CombinedOutput()
	if err != nil {
		msg := strings.TrimSpace(string(out))
		if msg == "" {
			msg = err.Error()
		}
		return fmt.Errorf("%s", msg)
	}
	return nil
}

// DoingDryRun generates and prints the doing prompt (with the pi workflowScript
// orchestration) without executing it.
func DoingDryRun(jobID string) error {
	if jobID == "" {
		fmt.Println("[DRY-RUN] No job ID provided")
		return nil
	}

	rickDir, err := workspace.GetRickDir()
	if err != nil {
		fmt.Printf("[DRY-RUN] failed to get rick dir: %v\n", err)
		return nil
	}

	planDir := filepath.Join(rickDir, "jobs", jobID, "plan")
	if _, err := os.Stat(planDir); os.IsNotExist(err) {
		doingDirFallback := filepath.Join(rickDir, "jobs", jobID, "doing")
		if _, e := os.Stat(filepath.Join(doingDirFallback, "requirement.md")); e == nil {
			fmt.Printf("[DRY-RUN] %s is an easy mode job (no plan/). Use: rick doing --easy --dry-run\n", jobID)
		} else {
			fmt.Printf("[DRY-RUN] plan directory not found: %s\n", planDir)
		}
		return nil
	}

	doingDir := filepath.Join(rickDir, "jobs", jobID, "doing")

	promptFile, _, _, err := builder.NewPIBuilder().SaveDoingPrompt(doingDir, planDir, rickDir, jobID)
	if err != nil {
		fmt.Printf("[DRY-RUN] failed to generate prompt: %v\n", err)
		return nil
	}

	content, err := os.ReadFile(promptFile)
	if err != nil {
		fmt.Printf("[DRY-RUN] failed to read prompt file: %v\n", err)
		return nil
	}

	fmt.Printf("[DRY-RUN] Doing prompt:\n\n")
	fmt.Println(string(content))
	return nil
}
