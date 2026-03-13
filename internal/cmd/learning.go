package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

func NewLearningCmd() *cobra.Command {
	var jobFlag string

	learningCmd := &cobra.Command{
		Use:   "learning",
		Short: "View or manage learning documentation",
		Long:  `View or manage learning documentation for your Rick workflow.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if GetVerbose() {
				fmt.Println("[INFO] Displaying learning documentation...")
			}

			if GetDryRun() {
				fmt.Println("[DRY-RUN] Would display learning documentation")
				return nil
			}

			if jobFlag != "" {
				fmt.Printf("Learning documentation for job: %s\n", jobFlag)
			} else {
				fmt.Println("Learning documentation")
			}
			return nil
		},
	}

	learningCmd.Flags().StringVar(&jobFlag, "job", "", "Specific job learning documentation to display")

	return learningCmd
}
