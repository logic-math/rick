package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

func NewDoingCmd() *cobra.Command {
	var jobFlag string

	doingCmd := &cobra.Command{
		Use:   "doing",
		Short: "Execute or track job execution",
		Long:  `Execute or track job execution in your Rick workflow.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if GetVerbose() {
				fmt.Println("[INFO] Executing job...")
			}

			if GetDryRun() {
				fmt.Println("[DRY-RUN] Would execute job")
				return nil
			}

			if jobFlag != "" {
				fmt.Printf("Executing job: %s\n", jobFlag)
			} else {
				fmt.Println("No job specified")
			}
			return nil
		},
	}

	doingCmd.Flags().StringVar(&jobFlag, "job", "", "Job to execute")

	return doingCmd
}
