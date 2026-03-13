package cmd

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/spf13/cobra"
	"github.com/sunquan/rick/internal/config"
	"github.com/sunquan/rick/internal/workspace"
)

func NewInitCmd() *cobra.Command {
	initCmd := &cobra.Command{
		Use:   "init",
		Short: "Initialize a new Rick workspace",
		Long:  `Initialize a new Rick workspace with all necessary directories and configuration files.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if GetVerbose() {
				fmt.Println("[INFO] Initializing Rick workspace...")
			}

			if GetDryRun() {
				fmt.Println("[DRY-RUN] Would initialize workspace")
				return nil
			}

			// Create workspace manager and initialize
			ws, err := workspace.New()
			if err != nil {
				return fmt.Errorf("failed to create workspace manager: %w", err)
			}

			if err := ws.InitWorkspace(); err != nil {
				return fmt.Errorf("failed to initialize workspace: %w", err)
			}

			if GetVerbose() {
				fmt.Println("[INFO] Workspace directories created")
			}

			// Initialize default config
			defaultConfig := config.GetDefaultConfig()
			if err := config.SaveConfig(defaultConfig); err != nil {
				return fmt.Errorf("failed to save default config: %w", err)
			}

			if GetVerbose() {
				fmt.Println("[INFO] Config file saved")
			}

			// Initialize Git repository in the .rick directory
			rickDir, err := workspace.GetRickDir()
			if err != nil {
				return fmt.Errorf("failed to get rick directory: %w", err)
			}

			if err := initGitRepo(rickDir); err != nil {
				if GetVerbose() {
					fmt.Printf("[WARN] Failed to initialize Git repository: %v\n", err)
				}
			} else if GetVerbose() {
				fmt.Println("[INFO] Git repository initialized")
			}

			fmt.Println("Workspace initialized successfully")
			return nil
		},
	}

	return initCmd
}

// initGitRepo initializes a git repository in the given directory
func initGitRepo(dir string) error {
	// Check if git is already initialized
	gitDir := fmt.Sprintf("%s/.git", dir)
	if _, err := os.Stat(gitDir); err == nil {
		// Git repository already exists
		return nil
	}

	// Initialize git repository
	cmd := exec.Command("git", "init")
	cmd.Dir = dir
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to run git init: %w", err)
	}

	// Create initial .gitignore if it doesn't exist
	gitignorePath := fmt.Sprintf("%s/.gitignore", dir)
	if _, err := os.Stat(gitignorePath); os.IsNotExist(err) {
		content := "# Rick workspace gitignore\n*.log\n.DS_Store\n"
		if err := os.WriteFile(gitignorePath, []byte(content), 0644); err != nil {
			return fmt.Errorf("failed to create .gitignore: %w", err)
		}
	}

	return nil
}
