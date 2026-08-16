package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/sunquan/rick/internal/handler"
)

func NewHumanLoopCmd() *cobra.Command {
	humanLoopCmd := &cobra.Command{
		Use:   "human-loop [topic]",
		Short: "Start a human-loop thinking session with AI assistance",
		Long:  `Start an interactive thinking session guided by the SENSE methodology. Provide a topic to think through deeply.`,
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 || args[0] == "" {
				return fmt.Errorf("topic is required")
			}
			topic := args[0]

			if GetDryRun() {
				return handler.HumanLoopDryRun(topic)
			}

			opts := handler.Options{
				Verbose: GetVerbose(),
				DryRun:  GetDryRun(),
				JobID:   GetJobID(),
			}
			return handler.HumanLoop(topic, opts)
		},
	}

	return humanLoopCmd
}
