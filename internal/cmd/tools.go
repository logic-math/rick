package cmd

import (
	"github.com/spf13/cobra"
)

// NewToolsCmd creates the tools parent command
// This is the meta-skill system for Rick, primarily used by AI agents.
// AI agents invoke `rick tools --help` to discover available tools and
// decide which commands to run during the learning phase.
func NewToolsCmd(version string) *cobra.Command {
	toolsCmd := &cobra.Command{
		Use:   "tools",
		Short: "Meta-skill tools for validating and managing Rick workflows",
		Long: `rick tools provides a set of validation and management commands for Rick workflows.

These commands are primarily used by AI agents during the learning phase.
An AI agent reads 'rick tools --help' to discover all available tools,
then decides which commands to invoke to complete its work.

Available subcommands:
  init-pi         Initialize pi (rick's agent runtime) + subagent extension
  update-pi       Update pi runtime / extensions / model catalogs (+ quick checks)
  theme           Manage pi themes (list / set / reset, applied to rick's managed pi config)
  dump            Pack workspaces + rick state into one portable archive
  import          Unpack a dump archive and make it work on this machine
  learning_check  Validate the learning directory structure for a job (incl. loops/skills format)
  dream_check     Validate dream_run_{job_id}_log.md files in .rick/dream/ (incl. loops/skills format)
  loops_check     Validate .rick/loops/*.md and .rick/skills/*.md format (frontmatter + required sections)
  dev-web         Manage the isolated dev instance (dev worktree + dev HOME) for developing rick itself
  release         Atomically promote the dev build to production, restart it, or roll back
  rsi_check       Validate the rsi-loop evidence of one self-evolution iteration (6 machine checks)
`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
	}

	toolsCmd.AddCommand(NewInitPiCmd())
	toolsCmd.AddCommand(NewUpdatePiCmd())
	toolsCmd.AddCommand(NewThemeCmd())
	toolsCmd.AddCommand(NewDumpCmd(version))
	toolsCmd.AddCommand(NewImportCmd())
	toolsCmd.AddCommand(NewLearningCheckCmd())
	toolsCmd.AddCommand(NewDreamCheckCmd())
	toolsCmd.AddCommand(NewLoopsCheckCmd())
	toolsCmd.AddCommand(NewDevWebCmd())
	toolsCmd.AddCommand(NewReleaseCmd())
	toolsCmd.AddCommand(NewRSICheckCmd())

	return toolsCmd
}
