package cmd

import (
	"github.com/spf13/cobra"
	"github.com/sunquan/rick/internal/handler"
)

func NewCtrlCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "ctrl",
		Short: "Monitor and intervene in a background doing session",
		Long:  `Launch an interactive pi agent that monitors doing progress and applies human interventions via task.md and tasks.json.`,
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			opts := handler.Options{
				Verbose: GetVerbose(),
				DryRun:  GetDryRun(),
				JobID:   GetJobID(),
			}
			if GetDryRun() {
				return handler.CtrlDryRun(opts)
			}
			return handler.Ctrl(opts)
		},
	}
}
