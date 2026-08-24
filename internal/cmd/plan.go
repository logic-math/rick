package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/sunquan/rick/internal/handler"
)

func NewPlanCmd() *cobra.Command {
	planCmd := &cobra.Command{
		Use:   "plan [requirement]",
		Short: "Plan a new job with AI assistance",
		Long:  `Plan a new job by describing your requirement. Rick will use AI to break it down into tasks.`,
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if GetVerbose() {
				fmt.Println("[INFO] Starting plan phase...")
			}

			opts := handler.Options{
				Verbose: GetVerbose(),
				DryRun:  GetDryRun(),
				JobID:   GetJobID(),
			}

			// v4.4.9: --resume job_N 恢复之前的 plan 会话（等价 --job 重进 + session 恢复）。
			if resumeTarget := GetResume(); resumeTarget != "" {
				if GetDryRun() {
					fmt.Printf("[DRY-RUN] Would resume plan session for job: %s\n", resumeTarget)
					return nil
				}
				requirement := ""
				if len(args) > 0 {
					requirement = args[0]
				}
				if requirement == "" {
					requirement = "恢复会话，继续完善此前的规划"
				}
				opts := handler.Options{Verbose: GetVerbose(), DryRun: GetDryRun(), JobID: resumeTarget}
				return handler.ReEnterPlan(resumeTarget, requirement, opts)
			}

			// Check if --job flag is set to re-enter an existing job
			if existingJobID := GetJobID(); existingJobID != "" {
				if GetDryRun() {
					fmt.Printf("[DRY-RUN] Would re-enter plan for job: %s\n", existingJobID)
					return nil
				}
				requirement := ""
				if len(args) > 0 {
					requirement = args[0]
				}
				if requirement == "" {
					requirement = "重新进入已有计划，继续完善任务分解"
				}
				return handler.ReEnterPlan(existingJobID, requirement, opts)
			}

			if GetDryRun() {
				return handler.PlanDryRun()
			}

			// Get requirement from args or interactive input
			requirement := ""
			if len(args) > 0 {
				requirement = args[0]
			} else {
				var err error
				requirement, err = promptForRequirement()
				if err != nil {
					return fmt.Errorf("failed to get requirement: %w", err)
				}
			}

			if requirement == "" {
				return fmt.Errorf("requirement cannot be empty")
			}

			if GetVerbose() {
				fmt.Printf("[INFO] Requirement: %s\n", requirement)
			}

			// Execute planning workflow
			if err := handler.Plan(requirement, opts); err != nil {
				return err
			}

			fmt.Println("Plan created successfully!")
			return nil
		},
	}

	return planCmd
}

// promptForRequirement prompts user for requirement input
func promptForRequirement() (string, error) {
	reader := bufio.NewReader(os.Stdin)
	fmt.Print("Enter your requirement: ")
	requirement, err := reader.ReadString('\n')
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(requirement), nil
}

// (All agent invocation goes through runtime.CallCLI — the unified CLI abstraction.)
