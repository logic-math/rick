package prompt

import (
	"fmt"
	"strings"

	"github.com/sunquan/rick/internal/parser"
	"github.com/sunquan/rick/internal/workspace"
)

// GenerateLearningPrompt generates the learning phase prompt from job execution results
// It includes completed work summary, task execution results, debug records, git history, and analysis
func GenerateLearningPrompt(jobID string, contextMgr *ContextManager, manager *PromptManager) (string, error) {
	if jobID == "" {
		return "", fmt.Errorf("job ID cannot be empty")
	}

	if contextMgr == nil {
		return "", fmt.Errorf("context manager cannot be nil")
	}

	if manager == nil {
		return "", fmt.Errorf("prompt manager cannot be nil")
	}

	// Load learning template
	template, err := manager.LoadTemplate("learning")
	if err != nil {
		return "", fmt.Errorf("failed to load learning template: %w", err)
	}

	// Create prompt builder
	builder := NewPromptBuilder(template)

	// Set project information
	projectName, err := workspace.GetProjectName()
	if err != nil {
		projectName = "unknown"
	}
	builder.SetVariable("project_name", projectName)
	builder.SetVariable("project_description", "Context-First AI Coding Framework")
	builder.SetVariable("job_id", jobID)

	// Set completed work summary
	completedWorkSummary := formatCompletedWorkSummary(contextMgr.GetHistory())
	builder.SetVariable("completed_work_summary", completedWorkSummary)

	// Set task execution results
	taskResults := formatTaskExecutionResults(contextMgr.GetTask())
	builder.SetVariable("task_execution_results", taskResults)

	// Set debug records
	debugRecords := formatLearningDebugRecords(contextMgr.GetDebug())
	builder.SetVariable("debug_records", debugRecords)

	// Set solutions summary
	solutionsSummary := formatSolutionsSummary(contextMgr.GetDebug())
	builder.SetVariable("solutions_summary", solutionsSummary)

	// Build final prompt
	prompt, err := builder.Build()
	if err != nil {
		return "", fmt.Errorf("failed to build learning prompt: %w", err)
	}

	return prompt, nil
}

// formatCompletedWorkSummary formats the completed work summary for the prompt
func formatCompletedWorkSummary(history []string) string {
	if len(history) == 0 {
		return "暂无已完成的工作"
	}

	var content strings.Builder
	content.WriteString("本周期内完成的主要工作：\n\n")
	for i, item := range history {
		content.WriteString(fmt.Sprintf("%d. %s\n", i+1, item))
	}

	return content.String()
}

// formatTaskExecutionResults formats task execution results for the prompt
func formatTaskExecutionResults(task *parser.Task) string {
	if task == nil {
		return "暂无任务执行结果"
	}

	var content strings.Builder
	content.WriteString(fmt.Sprintf("**任务ID**: %s\n", task.ID))
	content.WriteString(fmt.Sprintf("**任务名称**: %s\n\n", task.Name))
	content.WriteString(fmt.Sprintf("**任务目标**: %s\n\n", task.Goal))

	if len(task.KeyResults) > 0 {
		content.WriteString("**关键结果**:\n")
		for i, kr := range task.KeyResults {
			content.WriteString(fmt.Sprintf("%d. %s\n", i+1, kr))
		}
		content.WriteString("\n")
	}

	if task.TestMethod != "" {
		content.WriteString(fmt.Sprintf("**测试方法**: %s\n", task.TestMethod))
	}

	return content.String()
}

// formatLearningDebugRecords formats debug records for the learning prompt
func formatLearningDebugRecords(debugInfo *parser.DebugInfo) string {
	if debugInfo == nil || len(debugInfo.Entries) == 0 {
		return "本周期内暂无问题记录"
	}

	var content strings.Builder
	content.WriteString(fmt.Sprintf("共发现 %d 个问题：\n\n", len(debugInfo.Entries)))

	for _, entry := range debugInfo.Entries {
		content.WriteString(fmt.Sprintf("**问题 %d**: %s\n", entry.ID, entry.Phenomenon))

		if entry.Reproduce != "" {
			content.WriteString(fmt.Sprintf("- 复现方式: %s\n", entry.Reproduce))
		}

		if entry.Hypothesis != "" {
			content.WriteString(fmt.Sprintf("- 可能原因: %s\n", entry.Hypothesis))
		}

		if entry.Fix != "" {
			content.WriteString(fmt.Sprintf("- 解决方案: %s\n", entry.Fix))
		}

		if entry.Progress != "" {
			content.WriteString(fmt.Sprintf("- 解决状态: %s\n", entry.Progress))
		}

		content.WriteString("\n")
	}

	return content.String()
}

// formatSolutionsSummary formats solutions summary from debug records
func formatSolutionsSummary(debugInfo *parser.DebugInfo) string {
	if debugInfo == nil || len(debugInfo.Entries) == 0 {
		return "本周期内暂无解决方案"
	}

	var content strings.Builder
	content.WriteString("主要解决方案总结：\n\n")

	solvedCount := 0
	for _, entry := range debugInfo.Entries {
		if entry.Fix != "" && entry.Progress == "已修复" {
			solvedCount++
			content.WriteString(fmt.Sprintf("**方案 %d**: %s\n", solvedCount, entry.Fix))
			content.WriteString(fmt.Sprintf("- 针对问题: %s\n", entry.Phenomenon))
			content.WriteString("\n")
		}
	}

	if solvedCount == 0 {
		return "本周期内暂无已解决的问题"
	}

	return content.String()
}

