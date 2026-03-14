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
	var skipExplore bool

	initCmd := &cobra.Command{
		Use:   "init",
		Short: "Initialize a new Rick workspace",
		Long:  `Initialize a new Rick workspace with all necessary directories, configuration files, and global context through automated exploration.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if GetVerbose() {
				fmt.Println("[INFO] Initializing Rick workspace...")
			}

			if GetDryRun() {
				fmt.Println("[DRY-RUN] Would initialize workspace with exploration")
				return nil
			}

			// Step 1: Create workspace manager and initialize
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

			// Step 2: Initialize default config
			defaultConfig := config.GetDefaultConfig()
			if err := config.SaveConfig(defaultConfig); err != nil {
				return fmt.Errorf("failed to save default config: %w", err)
			}

			if GetVerbose() {
				fmt.Println("[INFO] Config file saved")
			}

			// Step 3: Initialize Git repository in the .rick directory
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

			fmt.Println("✅ Workspace initialized successfully")

			// Step 4: Automatic exploration if not skipped
			if skipExplore {
				if GetVerbose() {
					fmt.Println("[INFO] Skipping automatic exploration (--skip-explore flag set)")
				}
				fmt.Println("\n📝 To generate global context, run:")
				fmt.Println("   rick plan \"深度探索项目源码结构和架构设计\"")
				fmt.Println("   rick doing job_0")
				fmt.Println("   rick learning job_0")
				return nil
			}

			fmt.Println("\n🔍 Starting automatic exploration to generate global context...")
			fmt.Println("   This will execute: plan → doing → learning")
			fmt.Println("")

			// Execute automatic exploration workflow
			if err := executeInitExploration(); err != nil {
				fmt.Printf("\n⚠️  Exploration encountered an error: %v\n", err)
				fmt.Println("   You can manually complete the exploration by running:")
				fmt.Println("   rick plan \"深度探索项目源码结构和架构设计\"")
				fmt.Println("   rick doing job_0")
				fmt.Println("   rick learning job_0")
				return nil // Don't fail init, just skip exploration
			}

			fmt.Println("\n✅ Workspace initialization and exploration completed successfully!")
			fmt.Println("   Global context has been generated:")
			fmt.Println("   - .rick/OKR.md")
			fmt.Println("   - .rick/SPEC.md")
			fmt.Println("   - .rick/wiki/")
			fmt.Println("   - .rick/skills/")
			return nil
		},
	}

	initCmd.Flags().BoolVar(&skipExplore, "skip-explore", false, "Skip automatic exploration and only initialize workspace")

	return initCmd
}

// executeInitExploration performs automatic exploration to generate global context
func executeInitExploration() error {
	// Step 1: Execute plan workflow to generate job_0
	if GetVerbose() {
		fmt.Println("[INFO] Step 1/3: Executing plan workflow...")
	}
	fmt.Println("Step 1/3: Planning source code exploration...")

	requirement := "深度探索项目源码结构和架构设计，分析项目的整体架构、核心模块、关键依赖和最佳实践"
	if err := executePlanWorkflow(requirement); err != nil {
		return fmt.Errorf("failed to execute plan workflow: %w", err)
	}

	fmt.Println("✓ Plan completed: job_0 created")

	// Step 2: Execute doing workflow for job_0
	if GetVerbose() {
		fmt.Println("[INFO] Step 2/3: Executing doing workflow...")
	}
	fmt.Println("Step 2/3: Executing source code exploration tasks...")

	if err := executeDoingWorkflow("job_0"); err != nil {
		return fmt.Errorf("failed to execute doing workflow: %w", err)
	}

	fmt.Println("✓ Exploration completed: tasks executed")

	// Step 3: Execute learning workflow for job_0
	if GetVerbose() {
		fmt.Println("[INFO] Step 3/3: Executing learning workflow...")
	}
	fmt.Println("Step 3/3: Generating global context from exploration results...")

	if err := executeLearningWorkflow("job_0"); err != nil {
		return fmt.Errorf("failed to execute learning workflow: %w", err)
	}

	fmt.Println("✓ Learning completed: global context generated")

	return nil
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
