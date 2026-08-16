package cmd

import (
	"github.com/spf13/cobra"
	"github.com/sunquan/rick/internal/handler"
)

func NewDreamCmd() *cobra.Command {
	var jobNum int
	var background bool

	dreamCmd := &cobra.Command{
		Use:   "dream",
		Short: "Cross-job global reflection and skill evolution",
		Long:  `Perform cross-job global reflection, evolve skills, and maintain .rick knowledge base.`,
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if GetDryRun() {
				return handler.DreamDryRun(jobNum)
			}

			opts := handler.Options{
				Verbose: GetVerbose(),
				DryRun:  GetDryRun(),
				JobID:   GetJobID(),
			}
			return handler.Dream(jobNum, background, opts)
		},
	}

	dreamCmd.Flags().IntVar(&jobNum, "job_num", 5, "Number of jobs to process per dream run")
	dreamCmd.Flags().BoolVarP(&background, "background", "p", false, "Run in background (non-interactive, skip-permissions)")

	return dreamCmd
}
