package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/sunquan/rick/internal/handler"
)

// NewSuperCmd creates the `rick super` command group (job_47): a control
// session (default) plus start/stop/status/event subcommands over the super
// evolution workspaces. Go only orchestrates and supervises — the evolution
// loop itself lives in the py scheduler (super/superopt).
func NewSuperCmd() *cobra.Command {
	var name string
	var resume bool

	superCmd := &cobra.Command{
		Use:   "super",
		Short: "Super mode: control session + background evolution supervision",
		Long: "rick super — 进化模式中控。\n\n" +
			"默认打开中控会话（super-loop + gen_env skill 默认加载：env 建设/升级、\n" +
			"启停监控后台进化、干预投递）。进化由三层架构后台执行：Go PySupervisor 监督\n" +
			"→ py 调度器四状态循环 → pi agent 池。",
		RunE: func(cmd *cobra.Command, args []string) error {
			opts := handler.Options{
				Verbose: GetVerbose(),
				DryRun:  GetDryRun(),
				JobID:   GetJobID(),
			}
			if name == "" {
				var err error
				name, err = promptForSuperName()
				if err != nil {
					return err
				}
			}
			return handler.SuperSession(opts, name, resume)
		},
	}

	superCmd.Flags().StringVar(&name, "name", "", "Super workspace name (<super_root>/<name>)")
	superCmd.Flags().BoolVar(&resume, "resume", false, "Resume the workspace's latest control session")

	superCmd.AddCommand(newSuperStartCmd())
	superCmd.AddCommand(newSuperStopCmd())
	superCmd.AddCommand(newSuperStatusCmd())
	superCmd.AddCommand(newSuperEventCmd())
	return superCmd
}

// newSuperStartCmd: rick super start --name <n> [--env <path>] [--daemon] ...
func newSuperStartCmd() *cobra.Command {
	var name, env string
	var daemon bool
	var maxWallTime float64
	var maxIterations, adjustInterval int

	c := &cobra.Command{
		Use:   "start",
		Short: "Initialize a super workspace and launch the py scheduler",
		RunE: func(cmd *cobra.Command, args []string) error {
			opts := handler.Options{Verbose: GetVerbose(), DryRun: GetDryRun(), JobID: GetJobID()}
			ov := handler.SuperStartOverrides{
				MaxWallTimeH:   maxWallTime,
				MaxIterations:  maxIterations,
				AdjustInterval: adjustInterval,
			}
			return handler.SuperStart(opts, name, env, daemon, ov)
		},
	}
	c.Flags().StringVar(&name, "name", "", "Super workspace name (required)")
	c.Flags().StringVar(&env, "env", "", "Path to the env directory (gen_env output)")
	c.Flags().BoolVar(&daemon, "daemon", false, "Run detached (setsid + log file; stop via `rick super stop`)")
	c.Flags().Float64Var(&maxWallTime, "max-wall-time", 0, "Wall-clock budget in hours (0 = default 24h)")
	c.Flags().IntVar(&maxIterations, "max-iterations", 0, "Iteration cap (0 = unlimited; tests/e2e use this)")
	c.Flags().IntVar(&adjustInterval, "adjust-interval", 0, "Adjust-agent cadence in iterations (0 = default 5)")
	_ = c.MarkFlagRequired("name")
	return c
}

// newSuperStopCmd: rick super stop --name <n>
func newSuperStopCmd() *cobra.Command {
	var name string
	c := &cobra.Command{
		Use:   "stop",
		Short: "Gracefully stop the supervised scheduler (SIGTERM ladder)",
		RunE: func(cmd *cobra.Command, args []string) error {
			opts := handler.Options{Verbose: GetVerbose(), DryRun: GetDryRun(), JobID: GetJobID()}
			return handler.SuperStop(opts, name)
		},
	}
	c.Flags().StringVar(&name, "name", "", "Super workspace name (required)")
	_ = c.MarkFlagRequired("name")
	return c
}

// newSuperStatusCmd: rick super status --name <n> [--json]
func newSuperStatusCmd() *cobra.Command {
	var name string
	var jsonOut bool
	c := &cobra.Command{
		Use:   "status",
		Short: "Workspace state + heartbeat + north-star summary (from the event stream)",
		RunE: func(cmd *cobra.Command, args []string) error {
			opts := handler.Options{Verbose: GetVerbose(), DryRun: GetDryRun(), JobID: GetJobID()}
			return handler.SuperStatus(opts, name, jsonOut)
		},
	}
	c.Flags().StringVar(&name, "name", "", "Super workspace name (required)")
	c.Flags().BoolVar(&jsonOut, "json", false, "Machine-readable JSON output")
	_ = c.MarkFlagRequired("name")
	return c
}

// newSuperEventCmd: rick super event --name <n> --type <t> [--payload '<json>']
func newSuperEventCmd() *cobra.Command {
	var name, etype, payload string
	c := &cobra.Command{
		Use:   "event",
		Short: "Queue a human intervention (ingested by the scheduler at iteration boundaries)",
		RunE: func(cmd *cobra.Command, args []string) error {
			opts := handler.Options{Verbose: GetVerbose(), DryRun: GetDryRun(), JobID: GetJobID()}
			return handler.SuperEvent(opts, name, etype, payload)
		},
	}
	c.Flags().StringVar(&name, "name", "", "Super workspace name (required)")
	c.Flags().StringVar(&etype, "type", "", "Intervention type (pause/weight/resurrect/env_change/stop/...)")
	c.Flags().StringVar(&payload, "payload", "{}", "JSON payload for the intervention")
	_ = c.MarkFlagRequired("name")
	_ = c.MarkFlagRequired("type")
	return c
}

// promptForSuperName asks for the workspace name when --name is omitted.
func promptForSuperName() (string, error) {
	fmt.Print("Super workspace name: ")
	var name string
	if _, err := fmt.Scanln(&name); err != nil {
		return "", fmt.Errorf("failed to read workspace name: %w", err)
	}
	if name == "" {
		return "", fmt.Errorf("workspace name cannot be empty")
	}
	return name, nil
}
