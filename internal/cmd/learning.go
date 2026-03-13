package cmd

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/sunquan/rick/internal/config"
	"github.com/sunquan/rick/internal/git"
	"github.com/sunquan/rick/internal/workspace"
)

func NewLearningCmd() *cobra.Command {
	var jobID string

	learningCmd := &cobra.Command{
		Use:   "learning [job_id]",
		Short: "Perform knowledge accumulation and learning for a completed job",
		Long:  `Perform knowledge accumulation and learning for a completed job. Analyzes execution results and generates learning documentation.`,
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if GetVerbose() {
				fmt.Println("[INFO] Starting learning phase...")
			}

			if GetDryRun() {
				fmt.Println("[DRY-RUN] Would perform learning")
				return nil
			}

			// Get job ID from args or flag
			if len(args) > 0 {
				jobID = args[0]
			}

			if jobID == "" {
				return fmt.Errorf("job ID is required. Usage: rick learning [job_id] or rick learning --job job_id")
			}

			if GetVerbose() {
				fmt.Printf("[INFO] Performing learning for job: %s\n", jobID)
			}

			// Execute learning workflow
			if err := executeLearningWorkflow(jobID); err != nil {
				return err
			}

			fmt.Printf("Learning phase for job %s completed!\n", jobID)
			return nil
		},
	}

	learningCmd.Flags().StringVar(&jobID, "job", "", "Job ID to perform learning on")

	return learningCmd
}

// executeLearningWorkflow executes the complete learning workflow
func executeLearningWorkflow(jobID string) error {
	// Step 1: Load configuration and workspace
	cfg, err := config.LoadConfig()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	_, err = workspace.New()
	if err != nil {
		return fmt.Errorf("failed to create workspace: %w", err)
	}

	rickDir, err := workspace.GetRickDir()
	if err != nil {
		return fmt.Errorf("failed to get rick directory: %w", err)
	}

	if GetVerbose() {
		fmt.Printf("[INFO] Using workspace: %s\n", rickDir)
	}

	// Step 2: Validate job directory structure
	jobDir := filepath.Join(rickDir, "jobs", jobID)
	doingDir := filepath.Join(jobDir, "doing")
	learningDir := filepath.Join(jobDir, "learning")

	if _, err := os.Stat(jobDir); os.IsNotExist(err) {
		return fmt.Errorf("job directory not found: %s", jobDir)
	}

	if _, err := os.Stat(doingDir); os.IsNotExist(err) {
		return fmt.Errorf("doing directory not found: %s (job may not have been executed)", doingDir)
	}

	// Create learning directory if it doesn't exist
	if err := os.MkdirAll(learningDir, 0755); err != nil {
		return fmt.Errorf("failed to create learning directory: %w", err)
	}

	if GetVerbose() {
		fmt.Printf("[INFO] Job directory: %s\n", jobDir)
		fmt.Printf("[INFO] Doing directory: %s\n", doingDir)
		fmt.Printf("[INFO] Learning directory: %s\n", learningDir)
	}

	// Step 3: Load job execution results
	if GetVerbose() {
		fmt.Println("[INFO] Loading job execution results...")
	}

	executionResults, err := loadExecutionResults(doingDir, jobID)
	if err != nil {
		return fmt.Errorf("failed to load execution results: %w", err)
	}

	if GetVerbose() {
		fmt.Printf("[INFO] Execution results loaded: %d tasks executed\n", executionResults.TotalTasks)
	}

	// Step 4: Generate learning prompt
	if GetVerbose() {
		fmt.Println("[INFO] Generating learning prompt...")
	}

	learningPrompt, err := generateLearningPrompt(jobID, cfg, executionResults)
	if err != nil {
		return fmt.Errorf("failed to generate learning prompt: %w", err)
	}

	if GetVerbose() {
		fmt.Printf("[INFO] Learning prompt generated (length: %d bytes)\n", len(learningPrompt))
	}

	// Step 5: Call Claude Code CLI for learning summary
	if GetVerbose() {
		fmt.Println("[INFO] Calling Claude Code CLI for learning summary...")
	}

	learningResult, err := callClaudeCodeForLearning(cfg, learningPrompt, learningDir)
	if err != nil {
		return fmt.Errorf("failed to generate learning summary: %w", err)
	}

	if GetVerbose() {
		fmt.Printf("[INFO] Learning summary generated (length: %d bytes)\n", len(learningResult))
	}

	// Step 6: Parse learning results and update documentation
	if GetVerbose() {
		fmt.Println("[INFO] Parsing learning results and updating documentation...")
	}

	if err := updateDocumentation(rickDir, learningResult, learningDir); err != nil {
		return fmt.Errorf("failed to update documentation: %w", err)
	}

	if GetVerbose() {
		fmt.Println("[INFO] Documentation updated successfully")
	}

	// Step 7: Commit learning results
	if GetVerbose() {
		fmt.Println("[INFO] Committing learning results...")
	}

	if err := commitLearningResults(jobID); err != nil {
		fmt.Printf("[WARN] Failed to commit learning results: %v\n", err)
	}

	return nil
}

// ExecutionResults represents the execution results of a job
type ExecutionResults struct {
	JobID            string
	TotalTasks       int
	SuccessfulTasks  int
	FailedTasks      int
	ExecutionLog     string
	DebugRecords     string
	GitHistory       string
}

// loadExecutionResults loads execution results from the doing directory
func loadExecutionResults(doingDir string, jobID string) (*ExecutionResults, error) {
	results := &ExecutionResults{
		JobID: jobID,
	}

	// Load execution log
	logPath := filepath.Join(doingDir, "execution.log")
	if data, err := os.ReadFile(logPath); err == nil {
		results.ExecutionLog = string(data)
	}

	// Load debug records from debug.md if it exists
	debugPath := filepath.Join(doingDir, "debug.md")
	if data, err := os.ReadFile(debugPath); err == nil {
		results.DebugRecords = string(data)
	}

	// Get git history
	cwd, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("failed to get current working directory: %w", err)
	}

	gm := git.New(cwd)
	commits, err := gm.GetLog(20)
	if err == nil {
		var historyBuilder strings.Builder
		for _, commit := range commits {
			historyBuilder.WriteString(fmt.Sprintf("%s - %s (%s)\n", commit.Hash[:7], commit.Message, commit.Date.Format("2006-01-02")))
		}
		results.GitHistory = historyBuilder.String()
	}

	// Try to parse execution log to get task counts
	if results.ExecutionLog != "" {
		// Simple parsing to extract task counts
		lines := strings.Split(results.ExecutionLog, "\n")
		for _, line := range lines {
			if strings.Contains(line, "Total Tasks") {
				fmt.Sscanf(line, "Total Tasks: %d", &results.TotalTasks)
			} else if strings.Contains(line, "Successful Tasks") {
				fmt.Sscanf(line, "Successful Tasks: %d", &results.SuccessfulTasks)
			} else if strings.Contains(line, "Failed Tasks") {
				fmt.Sscanf(line, "Failed Tasks: %d", &results.FailedTasks)
			}
		}
	}

	return results, nil
}

// generateLearningPrompt generates the learning prompt from execution results
func generateLearningPrompt(jobID string, cfg *config.Config, results *ExecutionResults) (string, error) {
	// Create a simple learning prompt that includes all execution results
	var promptBuilder strings.Builder

	promptBuilder.WriteString("# Learning Summary for Job: " + jobID + "\n\n")

	promptBuilder.WriteString("## Execution Summary\n")
	promptBuilder.WriteString(fmt.Sprintf("- Total Tasks: %d\n", results.TotalTasks))
	promptBuilder.WriteString(fmt.Sprintf("- Successful Tasks: %d\n", results.SuccessfulTasks))
	promptBuilder.WriteString(fmt.Sprintf("- Failed Tasks: %d\n\n", results.FailedTasks))

	if results.ExecutionLog != "" {
		promptBuilder.WriteString("## Execution Log\n")
		promptBuilder.WriteString("```\n")
		promptBuilder.WriteString(results.ExecutionLog)
		promptBuilder.WriteString("\n```\n\n")
	}

	if results.DebugRecords != "" {
		promptBuilder.WriteString("## Debug Records\n")
		promptBuilder.WriteString(results.DebugRecords)
		promptBuilder.WriteString("\n\n")
	}

	if results.GitHistory != "" {
		promptBuilder.WriteString("## Git History\n")
		promptBuilder.WriteString("```\n")
		promptBuilder.WriteString(results.GitHistory)
		promptBuilder.WriteString("\n```\n\n")
	}

	promptBuilder.WriteString("## Learning Task\n")
	promptBuilder.WriteString("Based on the above execution results, please provide a comprehensive learning summary that includes:\n")
	promptBuilder.WriteString("1. Key achievements and milestones\n")
	promptBuilder.WriteString("2. Problems encountered and their solutions\n")
	promptBuilder.WriteString("3. Technical insights and lessons learned\n")
	promptBuilder.WriteString("4. Recommendations for future improvements\n")
	promptBuilder.WriteString("5. Knowledge that should be documented for team reference\n")

	return promptBuilder.String(), nil
}

// callClaudeCodeForLearning calls Claude Code CLI to generate learning summary
func callClaudeCodeForLearning(cfg *config.Config, learningPrompt string, learningDir string) (string, error) {
	// Create a temporary file for the prompt
	tmpFile, err := os.CreateTemp("", "rick-learning-*.md")
	if err != nil {
		return "", fmt.Errorf("failed to create temporary file: %w", err)
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.WriteString(learningPrompt); err != nil {
		return "", fmt.Errorf("failed to write prompt to temporary file: %w", err)
	}
	tmpFile.Close()

	// Call Claude Code CLI
	cmd := exec.Command("claude", "code", tmpFile.Name())
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("claude code execution failed: %w", err)
	}

	// Read the result from the temporary file (if updated)
	content, err := os.ReadFile(tmpFile.Name())
	if err != nil {
		return "", fmt.Errorf("failed to read result file: %w", err)
	}

	return string(content), nil
}

// updateDocumentation updates OKR.md, SPEC.md, and wiki with learning results
func updateDocumentation(rickDir string, learningResult string, learningDir string) error {
	// Save learning summary to learning directory
	learningPath := filepath.Join(learningDir, "learning_summary.md")
	if err := os.WriteFile(learningPath, []byte(learningResult), 0644); err != nil {
		return fmt.Errorf("failed to save learning summary: %w", err)
	}

	// Extract key insights from learning result and append to OKR.md
	okriPath := filepath.Join(rickDir, "OKR.md")
	if err := appendToFile(okriPath, "\n## Learning Insights\n\n"+extractKeyInsights(learningResult)); err != nil {
		return fmt.Errorf("failed to update OKR.md: %w", err)
	}

	// Append to SPEC.md
	specPath := filepath.Join(rickDir, "SPEC.md")
	if err := appendToFile(specPath, "\n## Implementation Notes\n\n"+extractImplementationNotes(learningResult)); err != nil {
		return fmt.Errorf("failed to update SPEC.md: %w", err)
	}

	return nil
}

// extractKeyInsights extracts key insights from learning result
func extractKeyInsights(learningResult string) string {
	// Simple extraction: look for "Key achievements", "lessons learned", etc.
	lines := strings.Split(learningResult, "\n")
	var insights strings.Builder

	inInsightsSection := false
	for _, line := range lines {
		if strings.Contains(strings.ToLower(line), "achievement") || strings.Contains(strings.ToLower(line), "lesson") || strings.Contains(strings.ToLower(line), "insight") {
			inInsightsSection = true
		}

		if inInsightsSection {
			insights.WriteString(line + "\n")
			if strings.HasPrefix(strings.TrimSpace(line), "##") && !strings.Contains(line, "achievement") && !strings.Contains(line, "lesson") {
				break
			}
		}
	}

	if insights.Len() == 0 {
		return "- Learning documentation generated\n"
	}

	return insights.String()
}

// extractImplementationNotes extracts implementation notes from learning result
func extractImplementationNotes(learningResult string) string {
	// Simple extraction: look for technical notes, recommendations, etc.
	lines := strings.Split(learningResult, "\n")
	var notes strings.Builder

	inNotesSection := false
	for _, line := range lines {
		if strings.Contains(strings.ToLower(line), "recommendation") || strings.Contains(strings.ToLower(line), "improvement") || strings.Contains(strings.ToLower(line), "technical") {
			inNotesSection = true
		}

		if inNotesSection {
			notes.WriteString(line + "\n")
			if strings.HasPrefix(strings.TrimSpace(line), "##") && !strings.Contains(line, "recommendation") && !strings.Contains(line, "improvement") {
				break
			}
		}
	}

	if notes.Len() == 0 {
		return "- Implementation notes documented\n"
	}

	return notes.String()
}

// appendToFile appends content to a file
func appendToFile(filePath string, content string) error {
	file, err := os.OpenFile(filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	if _, err := file.WriteString(content); err != nil {
		return fmt.Errorf("failed to write to file: %w", err)
	}

	return nil
}

// commitLearningResults commits the learning results to git
func commitLearningResults(jobID string) error {
	// Get current working directory
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current working directory: %w", err)
	}

	// Create git manager
	gm := git.New(cwd)

	// Create auto committer
	ac := git.NewAutoCommitter(gm)

	// Generate commit message
	commitMsg := fmt.Sprintf("morty: learning %s - COMPLETED\n\nLearning phase completed with knowledge documentation.\n\nCo-Authored-By: Claude Haiku 4.5 <noreply@anthropic.com>",
		jobID)

	// Commit using auto committer
	if err := ac.CommitJob(jobID, commitMsg); err != nil {
		return fmt.Errorf("failed to commit changes: %w", err)
	}

	return nil
}

// promptForLearningConfirmation prompts user to confirm learning phase
func promptForLearningConfirmation() (bool, error) {
	reader := bufio.NewReader(os.Stdin)
	fmt.Print("Do you want to proceed with learning phase? (y/n): ")
	response, err := reader.ReadString('\n')
	if err != nil {
		return false, err
	}

	response = strings.TrimSpace(strings.ToLower(response))
	return response == "y" || response == "yes", nil
}
