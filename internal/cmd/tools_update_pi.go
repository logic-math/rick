package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/sunquan/rick/internal/env"
	"github.com/sunquan/rick/internal/handler"
)

// NewUpdatePiCmd creates the update-pi subcommand: updates rick's pi runtime,
// its registered extensions, or model catalogs, then runs a quick post-update
// check (env 职责 1+2 的更新侧 + 职责 4 的快速自检). The actual work lives in
// internal/env — this command is a thin Cobra entry, same pattern as init-pi.
//
// Targets (positional, optional):
//
//	(nothing) / all        pi runtime + all extensions + model catalogs
//	pi / self              pi runtime only
//	extensions / ext       all registered extensions only
//	models                 model catalogs only
//	<extension-name>       one extension (e.g. pi-subagents, npm:pi-web-access)
//
// pi runtime updates use rick's own npm --prefix install for the managed
// runtime (~/.rick/pi/agent/runtime) — identical semantics to init-pi — so the
// update never depends on `pi update --self` succeeding for non-global installs.
// Extension updates run `pi update` against rick's managed agent dir only
// (PI_CODING_AGENT_DIR isolation), never the user's ~/.pi.
//
// After updating, the command redeploys rick's customizations (agents/hooks/
// skills, idempotent) and runs quick checks: pi version, required extensions
// registered, rick agents/hooks deployed, rick-gates helper syntax, and a
// human-loop prompt render smoke test. Check failures are reported as warnings
// (exit 0); only a failed update itself exits 1.
func NewUpdatePiCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "update-pi [target]",
		Short: "Update pi runtime / extensions / model catalogs + quick checks",
		Long: `Update rick's pi agent runtime, its extensions, or model catalogs.

Targets (positional, optional):
  (nothing) | all     Update pi runtime + all extensions + model catalogs
  pi | self           Update pi runtime only
  extensions | ext    Update all registered extensions only
  models             Refresh model catalogs only
  <extension-name>    Update one extension (e.g. pi-subagents)

The managed pi runtime (~/.rick/pi/agent/runtime) is updated via rick's own
npm --prefix install (same as init-pi). Extensions are updated via 'pi update'
against rick's managed agent dir only — the user's own ~/.pi is never touched.

After updating, rick redeploys its customizations (think/research/exporter
agents, rick-gates hook, skills) and runs quick checks (pi version, extension
registration, agent deployment, gates helper syntax, prompt render smoke test).

Exit codes:
  0  update succeeded (check warnings, if any, are printed but non-fatal)
  1  update failed (fix the reported error, then retry)`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			arg := ""
			if len(args) > 0 {
				arg = args[0]
			}
			target, one := env.ParseUpdateTarget(arg)

			res, err := env.UpdatePi(target, one)
			if err != nil {
				fmt.Fprintf(cmd.ErrOrStderr(), "❌ update-pi failed: %v\n", err)
				return err
			}
			printUpdateSummary(res)

			// 更新后快速自检：env 侧（pi 版本/扩展/定制/门禁 helper）+
			// prompt 渲染冒烟（handler 侧）。
			check := env.QuickCheck()
			printQuickCheck(check)
			printRenderSmokeCheck()
			return nil
		},
	}
}

// printUpdateSummary 输出更新摘要（实际发生了什么）。
func printUpdateSummary(res *env.UpdateResult) {
	fmt.Println()
	fmt.Println("✅ update-pi finished")
	if res.PiUpdated {
		scope := "managed runtime"
		if !res.ManagedRuntime {
			scope = "PATH pi (via pi update --self)"
		}
		fmt.Printf("  pi runtime   : updated (%s)  v%s → v%s\n",
			scope, orUnknown(res.PiBefore), orUnknown(res.PiAfter))
	}
	if res.ExtensionsUpdated {
		fmt.Println("  extensions   : all updated (pi update --extensions)")
	}
	if res.OneUpdated != "" {
		fmt.Printf("  extension    : updated %s\n", res.OneUpdated)
	}
	if res.ModelsRefreshed {
		fmt.Println("  model catalogs: refreshed")
	}
	for _, w := range res.Warnings {
		fmt.Printf("  ⚠️  %s\n", w)
	}
}

// printQuickCheck 输出 env 侧快速自检结果。
func printQuickCheck(check env.QuickCheckResult) {
	fmt.Println()
	fmt.Println("── post-update quick check ──")
	fmt.Printf("  pi version        : %s\n", orUnknown(check.PiVersion))
	if len(check.MissingExtensions) == 0 {
		fmt.Println("  extensions        : all registered")
	} else {
		fmt.Printf("  ⚠️  extensions      : missing %s\n",
			strings.Join(check.MissingExtensions, ", "))
	}
	if len(check.NotReady) == 0 {
		fmt.Println("  rick customizations: ready (agents + rick-gates deployed)")
	} else {
		fmt.Printf("  ⚠️  not ready       : %s\n", strings.Join(check.NotReady, ", "))
		fmt.Println("     → run `rick tools init-pi` in the rick repo to fix")
	}
	if check.GatesHelperOK {
		fmt.Println("  rick-gates helper : syntax OK")
	} else {
		fmt.Printf("  ⚠️  rick-gates helper: %s\n", check.GatesHelperNote)
	}
}

// printRenderSmokeCheck 做 human-loop prompt 渲染冒烟：模板能构建、非空、
// 且没有未替换的 {{placeholder}}。
func printRenderSmokeCheck() {
	content, err := handler.RenderHumanLoopPrompt("update-pi smoke check")
	if err != nil {
		fmt.Printf("  ⚠️  prompt render   : failed: %v\n", err)
		return
	}
	if strings.TrimSpace(content) == "" {
		fmt.Println("  ⚠️  prompt render   : empty human-loop prompt")
		return
	}
	if strings.Contains(content, "{{") {
		fmt.Println("  ⚠️  prompt render   : unreplaced {{placeholder}} in human-loop prompt")
		return
	}
	fmt.Println("  prompt render     : OK (human-loop prompt builds, no placeholders)")
}

func orUnknown(s string) string {
	if s == "" {
		return "unknown"
	}
	return s
}
