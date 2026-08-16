package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/sunquan/rick/internal/env"
)

// NewInitPiCmd creates the init-pi subcommand: ensures pi (rick's agent
// runtime), the subagent + web-access extensions, rick's customizations, and
// rick's managed pi config (settings + theme) are in place. The actual work is
// delegated to internal/env (env 四职责收口) — this command is a thin Cobra
// entry. Idempotent — each step checks before acting and skips what is done.
func NewInitPiCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "init-pi",
		Short: "Initialize pi (rick's agent runtime) + extensions + theme",
		Long: `Ensure pi is installed, required extensions are registered, and rick's
managed pi config is set up.

rick drives pi (@earendil-works/pi-coding-agent) as its agent runtime. This
command guarantees the runtime is ready: it installs pi if missing (via the
official installer), registers pi-subagents (Sub Agent delegation), registers
pi-web-access (external web search/fetch), keeps the managed config free of the
Tokyo Night package, and sets rick's managed theme. A final verification step
confirms everything is registered.

Idempotent: every step checks first and skips what is already satisfied.
Non-fatal: a missing extension/theme does not block rick; pi being entirely
missing is the only fatal condition.

Exit codes:
  0  pi environment ready (or ready enough to run rick)
  1  pi could not be installed/found (rick cannot run)`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := env.Ensure(); err != nil {
				fmt.Fprintf(os.Stderr, "❌ init-pi failed: %v\n", err)
				os.Exit(1)
			}
			return nil
		},
	}
}
