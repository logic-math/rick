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
				if GetResume() == "" {
					return fmt.Errorf("topic is required (or use --resume loop_N to resume a session)")
				}
				return nil
			}
			topic := args[0]

			// v4.4.9: --resume loop_N 恢复之前的 human-loop 会话（topic 参数忽略）。
			if resumeTarget := GetResume(); resumeTarget != "" {
				if GetDryRun() {
					fmt.Printf("[DRY-RUN] Would resume human-loop session: %s\n", resumeTarget)
					return nil
				}
				opts := handler.Options{Verbose: GetVerbose(), DryRun: GetDryRun(), JobID: GetJobID()}
				return handler.ResumeHumanLoop(resumeTarget, opts)
			}

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
