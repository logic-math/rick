package cmd

import (
	"github.com/spf13/cobra"
)

var (
	verbose bool
	dryRun  bool
)

func NewRootCmd(version string) *cobra.Command {
	rootCmd := &cobra.Command{
		Use:     "rick",
		Short:   "Rick CLI - A powerful command-line tool for managing development workflows",
		Long:    `Rick CLI is a comprehensive tool for managing development workflows, including job planning, task execution, and learning documentation.`,
		Version: version,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return cmd.Help()
			}
			return nil
		},
	}

	// Add global flags
	rootCmd.PersistentFlags().BoolVar(&verbose, "verbose", false, "Enable verbose output")
	rootCmd.PersistentFlags().BoolVar(&dryRun, "dry-run", false, "Run in dry-run mode without making changes")

	// Add version flag
	rootCmd.Flags().Bool("version", false, "Show version")

	// Add subcommands
	rootCmd.AddCommand(NewInitCmd())
	rootCmd.AddCommand(NewPlanCmd())
	rootCmd.AddCommand(NewDoingCmd())
	rootCmd.AddCommand(NewLearningCmd())

	return rootCmd
}

// GetVerbose returns the verbose flag value
func GetVerbose() bool {
	return verbose
}

// GetDryRun returns the dry-run flag value
func GetDryRun() bool {
	return dryRun
}
