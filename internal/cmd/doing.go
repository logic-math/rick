package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/sunquan/rick/internal/handler"
)

func NewDoingCmd() *cobra.Command {
	var jobID string
	var easy bool
	var ctxPath string

	doingCmd := &cobra.Command{
		Use:   "doing [job_id]",
		Short: "Execute tasks in a job",
		Long:  `Execute tasks in a job. Supports retry mechanism and automatic commits.`,
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if GetVerbose() {
				fmt.Println("[INFO] Starting doing phase...")
			}

			opts := handler.Options{
				Verbose: GetVerbose(),
				DryRun:  GetDryRun(),
				JobID:   GetJobID(),
			}

			// Easy mode: args[0] is requirement (not job_id); --job flag is for resume
			if easy {
				requirement := ""
				if len(args) > 0 {
					requirement = args[0]
				}
				if GetDryRun() {
					return handler.EasyDryRun(requirement, ctxPath)
				}
				// --job flag explicitly set → resume existing session
				if jobID == "" {
					jobID = GetJobID()
				}
				if jobID != "" {
					return handler.ResumeEasy(jobID, opts)
				}
				return handler.Easy(requirement, ctxPath, opts)
			}

			// Normal doing: args[0] is job_id
			if len(args) > 0 {
				jobID = args[0]
			} else if jobID == "" {
				jobID = GetJobID()
			}

			if GetDryRun() {
				return handler.DoingDryRun(jobID)
			}

			if jobID == "" {
				return fmt.Errorf("job ID is required. Usage: rick doing [job_id] or rick doing --job job_id")
			}

			// Validate job ID format
			if err := validateJobID(jobID); err != nil {
				return err
			}

			if GetVerbose() {
				fmt.Printf("[INFO] Executing job: %s\n", jobID)
			}

			// Execute doing workflow
			if err := handler.Doing(jobID, opts); err != nil {
				return err
			}

			fmt.Printf("Job %s execution completed!\n", jobID)
			return nil
		},
	}

	doingCmd.Flags().StringVar(&jobID, "job", "", "Job ID to execute")
	doingCmd.Flags().BoolVar(&easy, "easy", false, "Easy mode: skip plan, start interactive pi session")
	doingCmd.Flags().StringVar(&ctxPath, "ctx", "", "Inherit context from specified .rick directory (easy mode only)")

	return doingCmd
}
