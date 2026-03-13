package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

func NewPlanCmd() *cobra.Command {
	var jobFlag string

	planCmd := &cobra.Command{
		Use:   "plan",
		Short: "Display or manage job plans",
		Long:  `Display or manage job plans for your Rick workflow.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if GetVerbose() {
				fmt.Println("[INFO] Displaying plan...")
			}

			if GetDryRun() {
				fmt.Println("[DRY-RUN] Would display plan")
				return nil
			}

			if jobFlag != "" {
				fmt.Printf("Plan for job: %s\n", jobFlag)
			} else {
				fmt.Println("Overall plan")
			}
			return nil
		},
	}

	planCmd.Flags().StringVar(&jobFlag, "job", "", "Specific job to display plan for")

	return planCmd
}
