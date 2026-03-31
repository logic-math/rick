package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
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
			// Require the topic argument
			if len(args) == 0 || args[0] == "" {
				return fmt.Errorf("topic is required")
			}
			topic := args[0]

			if GetDryRun() {
				fmt.Printf("[DRY-RUN] Would start human-loop session for topic: %s\n", topic)
				return nil
			}

			// Get RFC directory and auto-create it
			rfcDir, err := workspace.GetRFCDir()
			if err != nil {
				return fmt.Errorf("failed to get RFC directory: %w", err)
			}
			if err := os.MkdirAll(rfcDir, 0755); err != nil {
				return fmt.Errorf("failed to create RFC directory: %w", err)
			}

			// Load config
			cfg, err := config.LoadConfig()
			if err != nil {
				return fmt.Errorf("failed to load config: %w", err)
			}

			// Generate human_loop prompt file
			promptMgr := prompt.NewPromptManager()
			promptFile, err := prompt.GenerateHumanLoopPromptFile(topic, rfcDir, promptMgr)
			if err != nil {
				return fmt.Errorf("failed to generate human-loop prompt: %w", err)
			}
			defer os.Remove(promptFile)

			if GetVerbose() {
				fmt.Printf("[INFO] Human-loop prompt saved to: %s\n", promptFile)
				fmt.Printf("[INFO] RFC directory: %s\n", rfcDir)
			}

			// Start Claude Code CLI interactive session
			if err := callClaudeCodeCLI(cfg, promptFile); err != nil {
				return fmt.Errorf("failed to start Claude Code CLI: %w", err)
			}

			fmt.Println("思考记录已保存到 .rick/RFC/ 目录（如果 AI 已执行 sense-express）")
			return nil
		},
	}

	return humanLoopCmd
}
