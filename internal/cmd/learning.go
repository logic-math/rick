package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/sunquan/rick/internal/handler"
)

func NewLearningCmd() *cobra.Command {
	var jobID string

	learningCmd := &cobra.Command{
		Use:   "learning [job_id]",
		Short: "Analyze and document learnings from job execution",
		Long:  `Analyze execution results and update loops, skills, and SUMMARY.md.`,
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if GetVerbose() {
				fmt.Println("[INFO] Starting learning phase...")
			}

			if len(args) > 0 {
				jobID = args[0]
			}
			if jobID == "" {
				jobID = GetJobID()
			}

			if GetDryRun() {
				return handler.LearningDryRun(jobID)
			}

			if jobID == "" {
				return fmt.Errorf("job ID is required. Usage: rick learning [job_id] or rick learning --job job_id")
			}

			if err := validateJobID(jobID); err != nil {
				return err
			}

			if GetVerbose() {
				fmt.Printf("[INFO] Analyzing learnings for job: %s\n", jobID)
			}

			opts := handler.Options{
				Verbose: GetVerbose(),
				DryRun:  GetDryRun(),
				JobID:   GetJobID(),
			}
			if err := handler.Learning(jobID, opts); err != nil {
				return err
			}

			fmt.Printf("✅ Learning phase completed for job %s!\n", jobID)
			return nil
		},
	}

	learningCmd.Flags().StringVar(&jobID, "job", "", "Job ID to analyze")

	return learningCmd
}
