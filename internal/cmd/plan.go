package cmd

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/sunquan/rick/internal/config"
	"github.com/sunquan/rick/internal/prompt"
	"github.com/sunquan/rick/internal/workspace"
)

func NewPlanCmd() *cobra.Command {
	planCmd := &cobra.Command{
		Use:   "plan [requirement]",
		Short: "Plan a new job with AI assistance",
		Long:  `Plan a new job by describing your requirement. Rick will use AI to break it down into tasks.`,
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if GetVerbose() {
				fmt.Println("[INFO] Starting plan phase...")
			}

			if GetDryRun() {
				fmt.Println("[DRY-RUN] Would create a plan")
				return nil
			}

			// Get requirement from args or interactive input
			requirement := ""
			if len(args) > 0 {
				requirement = args[0]
			} else {
				var err error
				requirement, err = promptForRequirement()
				if err != nil {
					return fmt.Errorf("failed to get requirement: %w", err)
				}
			}

			if requirement == "" {
				return fmt.Errorf("requirement cannot be empty")
			}

			if GetVerbose() {
				fmt.Printf("[INFO] Requirement: %s\n", requirement)
			}

			// Execute planning workflow
			if err := executePlanWorkflow(requirement); err != nil {
				return err
			}

			fmt.Println("Plan created successfully!")
			return nil
		},
	}

	return planCmd
}

// promptForRequirement prompts user for requirement input
func promptForRequirement() (string, error) {
	reader := bufio.NewReader(os.Stdin)
	fmt.Print("Enter your requirement: ")
	requirement, err := reader.ReadString('\n')
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(requirement), nil
}

// executePlanWorkflow executes the complete planning workflow
func executePlanWorkflow(requirement string) error {
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

	if GetVerbose() {
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

	if GetVerbose() {
		fmt.Printf("[INFO] Created job directory: %s\n", jobPlanDir)
	}

	fmt.Printf("Job ID: %s\n", jobID)
	fmt.Printf("Plan directory: %s\n", jobPlanDir)

	// Step 3: Generate planning prompt
	if GetVerbose() {
		fmt.Println("[INFO] Generating planning prompt...")
	}

	contextMgr := prompt.NewContextManager(jobID)

	// Load OKR and SPEC
	okriPath := filepath.Join(rickDir, workspace.OKRFileName)
	if _, err := os.Stat(okriPath); err == nil {
		if err := contextMgr.LoadOKRFromFile(okriPath); err != nil && GetVerbose() {
			fmt.Printf("[WARN] Failed to load OKR: %v\n", err)
		}
	}

	specPath := filepath.Join(rickDir, workspace.SpecFileName)
	if _, err := os.Stat(specPath); err == nil {
		if err := contextMgr.LoadSPECFromFile(specPath); err != nil && GetVerbose() {
			fmt.Printf("[WARN] Failed to load SPEC: %v\n", err)
		}
	}

	// Create prompt manager
	templateDir := filepath.Join(rickDir, "templates")
	if _, err := os.Stat(templateDir); os.IsNotExist(err) {
		// Use default template directory from internal/prompt/templates
		templateDir = "" // Will use embedded templates
	}

	promptMgr := prompt.NewPromptManager(templateDir)

	// Generate plan prompt and save to temporary file
	planPromptFile, err := prompt.GeneratePlanPromptFile(requirement, jobPlanDir, contextMgr, promptMgr)
	if err != nil {
		return fmt.Errorf("failed to generate plan prompt: %w", err)
	}
	defer os.Remove(planPromptFile) // Clean up temporary file

	if GetVerbose() {
		fmt.Printf("[INFO] Planning prompt saved to: %s\n", planPromptFile)
	}

	// Step 4: Call Claude Code CLI with planning prompt file (interactive mode)
	if GetVerbose() {
		fmt.Println("[INFO] Calling Claude Code CLI for planning...")
	}

	if err := callClaudeCodeCLI(cfg, planPromptFile); err != nil {
		return fmt.Errorf("failed to call Claude Code CLI: %w", err)
	}

	fmt.Printf("\nPlanning session completed! Job: %s\n", jobID)
	fmt.Println("Please review the generated task files and then run:")
	fmt.Printf("  rick doing %s\n", jobID)

	return nil
}


// generateJobID generates a new job ID (job_N format)
func generateJobID() string {
	// Use timestamp-based ID for uniqueness
	return fmt.Sprintf("job_%d", time.Now().Unix())
}

// callClaudeCodeCLI calls Claude Code CLI in interactive mode
// promptFile is the path to the prompt file to be loaded by Claude
func callClaudeCodeCLI(cfg *config.Config, promptFile string) error {
	// Get Claude CLI path from config
	claudePath := cfg.ClaudeCodePath
	if claudePath == "" {
		claudePath = "claude"
	}

	// Call Claude Code CLI in interactive mode with prompt file
	// Claude will load the prompt from the file
	cmd := exec.Command(claudePath, promptFile)

	// Connect stdin, stdout and stderr to terminal for interactive session
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if GetVerbose() {
		fmt.Printf("[INFO] Executing: %s %s\n", claudePath, promptFile)
	}

	// Run the command
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("Claude Code CLI failed: %w", err)
	}

	return nil
}
