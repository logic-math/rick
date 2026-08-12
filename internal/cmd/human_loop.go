package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/sunquan/rick/internal/agent/piagent"
	"github.com/sunquan/rick/internal/config"
	"github.com/sunquan/rick/internal/prompt"
	"github.com/sunquan/rick/internal/workspace"
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

			draftDir, err := workspace.GetDraftDir()
			if err != nil {
				return fmt.Errorf("failed to get draft directory: %w", err)
			}
			rfcDir := draftDir + "/rfc"

			// Allocate next loop_N directory
			loopID, err := workspace.NextLoopID(draftDir)
			if err != nil {
				return fmt.Errorf("failed to allocate loop id: %w", err)
			}
			loopDir := filepath.Join(draftDir, "loops", loopID)

			for _, sub := range []string{
				draftDir,
				rfcDir,
				draftDir + "/concepts",
				draftDir + "/human-learning",
				filepath.Join(draftDir, "loops"),
			} {
				if err := os.MkdirAll(sub, 0755); err != nil {
					return fmt.Errorf("failed to create directory %s: %w", sub, err)
				}
			}

			promptMgr := prompt.NewPromptManager()

			if GetDryRun() {
				content, err := prompt.GenerateHumanLoopPrompt(topic, rfcDir, draftDir, promptMgr)
				if err != nil {
					return fmt.Errorf("failed to generate human-loop prompt: %w", err)
				}
				fmt.Fprint(cmd.OutOrStdout(), content)
				return nil
			}

			cfg, err := config.LoadConfig()
			if err != nil {
				return fmt.Errorf("failed to load config: %w", err)
			}

			mainFile, _, _, _, err := prompt.GenerateHumanLoopPromptFile(topic, rfcDir, draftDir, loopDir, promptMgr)
			if err != nil {
				return fmt.Errorf("failed to generate human-loop prompt: %w", err)
			}

			if GetVerbose() {
				fmt.Printf("[INFO] Loop directory: %s\n", loopDir)
				fmt.Printf("[INFO] Human-loop prompt saved to: %s\n", mainFile)
				fmt.Printf("[INFO] rfc directory: %s\n", rfcDir)
			}

			if err := piagent.CallCLI(GetVerbose(), cfg, mainFile, piagent.ModeInteractive); err != nil {
				return fmt.Errorf("failed to start pi CLI: %w", err)
			}

			fmt.Printf("思考记录已保存到 %s\n", loopDir)
			return nil
		},
	}

	return humanLoopCmd
}
