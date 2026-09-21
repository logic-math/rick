package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/sunquan/rick/internal/web"
)

var (
	verbose bool
	dryRun  bool
	jobID   string
	resume  string
)

// BuildID is the build fingerprint, injected at link time:
//
//	-ldflags "-X github.com/sunquan/rick/internal/cmd.BuildID=<sha7>-<YYYYMMDDHHMMSS>"
//
// 默认 "dev"（未注入时）。它让「现在跑的是不是我刚构建的那个二进制」可被一条
// curl 判定（/api/health 免认证携带 build_id；/api/config 与 SSE server_info 同源）
// ——自进化闭环的反馈回路基石（task17 的 dev-web up 与 task21 的 release 都靠它
// 校验「新构建真的在跑」，research-L5 §4）。
var BuildID = "dev"

// FuncVersion returns the effective build fingerprint (empty → "dev"), so
// no caller has to special-case the un-injected case.
func FuncVersion() string {
	if strings.TrimSpace(BuildID) == "" {
		return "dev"
	}
	return BuildID
}

func NewRootCmd(version string) *cobra.Command {
	// 把构建指纹交给 web 层（web 不能反向 import cmd，由组合根注入）——
	// `rick web` 的所有入口都经由本函数，故 /api/health、/api/config、
	// SSE server_info 总能拿到真实指纹。
	web.SetBuildID(FuncVersion())

	rootCmd := &cobra.Command{
		Use:     "rick",
		Short:   "Rick CLI - A powerful command-line tool for managing development workflows",
		Long:    `Rick CLI is a comprehensive tool for managing development workflows, including job planning, task execution, and learning documentation.`,
		Version: version,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return cmd.Help()
			}
			return nil
		},
	}

	// Add global flags
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Enable verbose output")
	rootCmd.PersistentFlags().BoolVar(&dryRun, "dry-run", false, "Run in dry-run mode without making changes")
	rootCmd.PersistentFlags().StringVar(&jobID, "job", "", "Specify job ID (e.g., job_1)")
	rootCmd.PersistentFlags().StringVar(&resume, "resume", "", "Resume a previous pi session (plan/doing: job id; human-loop: loop_N; easy: job id)")

	// Configure version flag
	rootCmd.Flags().BoolP("version", "V", false, "Show version")
	rootCmd.SetVersionTemplate("Rick CLI version {{.Version}}\n")

	// Add subcommands
	rootCmd.AddCommand(NewPlanCmd())
	rootCmd.AddCommand(NewDoingCmd())
	rootCmd.AddCommand(NewLearningCmd())
	rootCmd.AddCommand(NewDreamCmd())
	rootCmd.AddCommand(NewToolsCmd())
	rootCmd.AddCommand(NewHumanLoopCmd())
	rootCmd.AddCommand(NewCtrlCmd())
	rootCmd.AddCommand(NewEasyCmd())
	rootCmd.AddCommand(NewRSICmd())
	rootCmd.AddCommand(NewWebCmd(version))

	return rootCmd
}

// GetVerbose returns the verbose flag value
func GetVerbose() bool {
	return verbose
}

// GetDryRun returns the dry-run flag value
func GetDryRun() bool {
	return dryRun
}

// GetResume returns the --resume flag value (session re-entry target: job id
// for plan/doing, loop_N for human-loop, job id for easy).
func GetResume() string {
	return resume
}

// GetJobID returns the job ID flag value
func GetJobID() string {
	return jobID
}

// validateJobID validates the job ID format
func validateJobID(id string) error {
	if id == "" {
		return fmt.Errorf("job ID cannot be empty")
	}

	// Check if job ID matches expected format (job_N or similar)
	if len(id) < 1 {
		return fmt.Errorf("job ID is too short")
	}

	// Allow alphanumeric characters, underscores, and hyphens
	for _, ch := range id {
		if !((ch >= 'a' && ch <= 'z') ||
			(ch >= 'A' && ch <= 'Z') ||
			(ch >= '0' && ch <= '9') ||
			ch == '_' || ch == '-') {
			return fmt.Errorf("job ID contains invalid characters: %c", ch)
		}
	}

	return nil
}
