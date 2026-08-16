package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/sunquan/rick/internal/handler"
)

// NewEasyCmd creates the `rick easy` command for interactive AI coding sessions.
func NewEasyCmd() *cobra.Command {
	var requirement string
	var ctxPath string
	var resumeJobID string

	easyCmd := &cobra.Command{
		Use:   "easy",
		Short: "Start an interactive easy AI coding session",
		RunE: func(cmd *cobra.Command, args []string) error {
			opts := handler.Options{
				Verbose: GetVerbose(),
				DryRun:  GetDryRun(),
				JobID:   GetJobID(),
			}

			if GetDryRun() {
				return handler.EasyDryRun(requirement, ctxPath)
			}
			if resumeJobID != "" {
				return handler.ResumeEasy(resumeJobID, opts)
			}
			if len(args) > 0 && requirement == "" {
				requirement = args[0]
			}
			// Resolve interactive requirement (preserved from runEasyMode)
			if requirement == "" {
				var err error
				requirement, err = promptForRequirement()
				if err != nil {
					return fmt.Errorf("failed to get requirement: %w", err)
				}
			}
			return handler.Easy(requirement, ctxPath, opts)
		},
	}

	easyCmd.Flags().StringVarP(&requirement, "requirement", "r", "", "Requirement for the easy session")
	easyCmd.Flags().StringVar(&ctxPath, "ctx", "", "Path to a .rick directory to inherit context from")
	easyCmd.Flags().StringVar(&resumeJobID, "resume", "", "Resume an existing easy session by job ID")

	return easyCmd
}
