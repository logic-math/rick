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

	// Step 2: Generate planning prompt
	if GetVerbose() {
		fmt.Println("[INFO] Generating planning prompt...")
	}

	contextMgr := prompt.NewContextManager("plan")

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

	planPrompt, err := prompt.GeneratePlanPrompt(requirement, contextMgr, promptMgr)
	if err != nil {
		return fmt.Errorf("failed to generate plan prompt: %w", err)
	}

	if GetVerbose() {
		fmt.Println("[INFO] Planning prompt generated")
	}

	// Step 3: Call Claude Code CLI with planning prompt (interactive mode)
	if GetVerbose() {
		fmt.Println("[INFO] Calling Claude Code CLI for planning...")
	}

	if err := callClaudeCodeCLI(cfg, planPrompt); err != nil {
		return fmt.Errorf("failed to call Claude Code CLI: %w", err)
	}

	fmt.Println("\nPlanning session completed!")
	fmt.Println("Please review the generated task files and then run:")
	fmt.Println("  rick doing <job_id>")

	return nil
}


// generateJobID generates a new job ID (job_N format)
func generateJobID() string {
	// Use timestamp-based ID for uniqueness
	return fmt.Sprintf("job_%d", time.Now().Unix())
}

// callClaudeCodeCLI calls Claude Code CLI in interactive mode
func callClaudeCodeCLI(cfg *config.Config, prompt string) error {
	// Get Claude CLI path from config
	claudePath := cfg.ClaudeCodePath
	if claudePath == "" {
		claudePath = "claude"
	}

	// Call Claude Code CLI in interactive mode
	// Pass prompt via stdin for interactive session
	cmd := exec.Command(claudePath, "--permission-mode", "plan")

	// Create pipe for stdin
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdin pipe: %w", err)
	}

	// Connect stdout and stderr to terminal
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if GetVerbose() {
		fmt.Printf("[INFO] Executing: %s --permission-mode plan\n", claudePath)
	}

	// Start the command
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start Claude Code CLI: %w", err)
	}

	// Write prompt to stdin
	if _, err := stdin.Write([]byte(prompt)); err != nil {
		return fmt.Errorf("failed to write prompt: %w", err)
	}
	stdin.Close()

	// Wait for completion
	if err := cmd.Wait(); err != nil {
		return fmt.Errorf("Claude Code CLI failed: %w", err)
	}

	return nil
}
