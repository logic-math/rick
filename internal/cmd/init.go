package cmd

import (
	"fmt"

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

			// Initialize default config
			defaultConfig := config.GetDefaultConfig()
			if err := config.SaveConfig(defaultConfig); err != nil {
				return fmt.Errorf("failed to save default config: %w", err)
			}

			fmt.Println("Workspace initialized successfully")
			return nil
		},
	}

	return initCmd
}
