package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/spf13/cobra"
	"github.com/sunquan/rick/internal/workspace"
)

var validJobIDPattern = regexp.MustCompile(`^job_\d+$`)

// NewDreamCheckCmd creates the dream_check subcommand
func NewDreamCheckCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "dream_check",
		Short: "Validate dream_run_{job_id}_log.md files in .rick/dream/",
		Long: `Check .rick/dream/ to ensure dream log files are valid.

Checks performed:
  - .rick/dream/ directory exists
  - each dream_run_*_log.md filename contains a valid job_N job ID
  - no duplicate job IDs across log files
  - each recorded job ID has a corresponding .rick/jobs/ directory
  - .rick/loops/*.md: frontmatter (name, trigger) + 5 sections (目标/上下文管理/可调用工具/产出评估/停止标准)
  - .rick/skills/*.md: frontmatter (name, description) + 4 sections (When to Use/Procedure/Pitfalls/Verification)
  - README.md in each dir is skipped

Exit codes:
  0  all checks passed
  1  one or more checks failed`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			rickDir, err := workspace.GetRickDir()
			if err != nil {
				return fmt.Errorf("failed to resolve rick directory: %w", err)
			}
			if err := runDreamCheck(rickDir); err != nil {
				fmt.Fprintf(os.Stderr, "❌ dream check failed: %v\n", err)
				os.Exit(1)
			}
			return nil
		},
	}
}

func runDreamCheck(rickDir string) error {
	dreamDir := filepath.Join(rickDir, workspace.DreamDirName)

	if _, err := os.Stat(dreamDir); os.IsNotExist(err) {
		fmt.Printf("✅ dream check passed: dream directory not yet created (no runs yet)\n")
		return nil
	}

	jobIDs, err := parseDreamLogJobIDs(dreamDir)
	if err != nil {
		return err
	}

	for _, id := range jobIDs {
		jobDir := filepath.Join(rickDir, "jobs", id)
		if _, err := os.Stat(jobDir); os.IsNotExist(err) {
			return fmt.Errorf("log file references job %q but directory not found: %s", id, jobDir)
		}
	}

	lsErrs := runLoopsAndSkillsCheck(rickDir)
	if len(lsErrs) > 0 {
		return fmt.Errorf("loops/skills check errors:\n  - %s", strings.Join(lsErrs, "\n  - "))
	}

	fmt.Printf("✅ dream check passed: %d processed job(s) recorded\n", len(jobIDs))
	return nil
}

// parseDreamLogJobIDs scans dreamDir for dream_run_{job_id}_log.md files,
// validates naming, and returns the list of job IDs. Errors on invalid or duplicate names.
func parseDreamLogJobIDs(dreamDir string) ([]string, error) {
	entries, err := os.ReadDir(dreamDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read dream directory: %w", err)
	}

	var jobIDs []string
	seen := make(map[string]bool)

	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if !strings.HasPrefix(name, "dream_run_") || !strings.HasSuffix(name, "_log.md") {
			continue
		}
		jobID := strings.TrimPrefix(name, "dream_run_")
		jobID = strings.TrimSuffix(jobID, "_log.md")
		if !validJobIDPattern.MatchString(jobID) {
			return nil, fmt.Errorf("invalid log filename %q: job ID part %q does not match job_N format", name, jobID)
		}
		if seen[jobID] {
			return nil, fmt.Errorf("duplicate log file for job %q in dream directory", jobID)
		}
		seen[jobID] = true
		jobIDs = append(jobIDs, jobID)
	}
	return jobIDs, nil
}
