package cmd

import (
	"github.com/spf13/cobra"
)

// NewToolsCmd creates the tools parent command
// This is the meta-skill system for Rick, primarily used by AI agents.
// AI agents invoke `rick tools --help` to discover available tools and
// decide which commands to run during the learning phase.
func NewToolsCmd() *cobra.Command {
	toolsCmd := &cobra.Command{
		Use:   "tools",
		Short: "Meta-skill tools for validating and managing Rick workflows",
		Long: `rick tools provides a set of validation and management commands for Rick workflows.

These commands are primarily used by AI agents during the learning phase.
An AI agent reads 'rick tools --help' to discover all available tools,
then decides which commands to invoke to complete its work.

Available subcommands:
  plan_check    Validate the plan directory structure for a job`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
	}

	toolsCmd.AddCommand(NewPlanCheckCmd())

	return toolsCmd
}
