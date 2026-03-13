package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var (
	verbose bool
	dryRun  bool
	jobID   string
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
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Enable verbose output")
	rootCmd.PersistentFlags().BoolVar(&dryRun, "dry-run", false, "Run in dry-run mode without making changes")
	rootCmd.PersistentFlags().StringVar(&jobID, "job", "", "Specify job ID (e.g., job_1)")

	// Configure version flag
	rootCmd.Flags().BoolP("version", "V", false, "Show version")
	rootCmd.SetVersionTemplate("Rick CLI version {{.Version}}\n")

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

// GetJobID returns the job ID flag value
func GetJobID() string {
	return jobID
}

// validateJobID validates the job ID format
func validateJobID(id string) error {
	if id == "" {
		return fmt.Errorf("job ID cannot be empty")
	}

	// Check if job ID matches expected format (job_N or similar)
	if len(id) < 1 {
		return fmt.Errorf("job ID is too short")
	}

	// Allow alphanumeric characters, underscores, and hyphens
	for _, ch := range id {
		if !((ch >= 'a' && ch <= 'z') ||
			(ch >= 'A' && ch <= 'Z') ||
			(ch >= '0' && ch <= '9') ||
			ch == '_' || ch == '-') {
			return fmt.Errorf("job ID contains invalid characters: %c", ch)
		}
	}

	return nil
}
