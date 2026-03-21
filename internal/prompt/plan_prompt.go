package prompt

import (
	"fmt"
	"strings"

	"github.com/sunquan/rick/internal/parser"
	"github.com/sunquan/rick/internal/workspace"
)

// GeneratePlanPrompt generates the planning phase prompt from user requirement.
// jobPlanDir is the absolute path to the job's plan directory.
func GeneratePlanPrompt(requirement string, jobPlanDir string, contextMgr *ContextManager, manager *PromptManager) (string, error) {
	if requirement == "" {
		return "", fmt.Errorf("requirement cannot be empty")
	}

	if contextMgr == nil {
		return "", fmt.Errorf("context manager cannot be nil")
	}

	if manager == nil {
		return "", fmt.Errorf("prompt manager cannot be nil")
	}

	// Load plan template
	template, err := manager.LoadTemplate("plan")
	if err != nil {
		return "", fmt.Errorf("failed to load plan template: %w", err)
	}

	// Create prompt builder
	builder := NewPromptBuilder(template)

	// Set project information
	projectName, _ := workspace.GetProjectName()
	builder.SetVariable("project_name", projectName)
	builder.SetVariable("project_description", "Context-First AI Coding Framework")

	// Set OKR content (full content, not just list items)
	okrContent := formatOKRContent(contextMgr.GetOKRInfo())
	builder.SetVariable("okr_content", okrContent)

	// Set SPEC content (full content, not just list items)
	specContent := formatSPECContent(contextMgr.GetSPECInfo())
	builder.SetVariable("spec_content", specContent)

	// Set user requirement
	builder.SetVariable("user_requirement", requirement)

	// Set job plan directory
	builder.SetVariable("job_plan_dir", jobPlanDir)

	// Build final prompt
	prompt, err := builder.Build()
	if err != nil {
		return "", fmt.Errorf("failed to build plan prompt: %w", err)
	}

	return prompt, nil
}

// GeneratePlanPromptFile generates the planning phase prompt and saves it to a temporary file.
// jobPlanDir is the absolute path to the job's plan directory (e.g. .rick/jobs/job_1/plan).
// Returns the file path and any error. The caller is responsible for cleaning up the temporary file.
func GeneratePlanPromptFile(requirement string, jobPlanDir string, contextMgr *ContextManager, manager *PromptManager) (string, error) {
	if requirement == "" {
		return "", fmt.Errorf("requirement cannot be empty")
	}

	if contextMgr == nil {
		return "", fmt.Errorf("context manager cannot be nil")
	}

	if manager == nil {
		return "", fmt.Errorf("prompt manager cannot be nil")
	}

	// Load plan template
	template, err := manager.LoadTemplate("plan")
	if err != nil {
		return "", fmt.Errorf("failed to load plan template: %w", err)
	}

	// Create prompt builder
	builder := NewPromptBuilder(template)

	// Set project information
	projectName, _ := workspace.GetProjectName()
	builder.SetVariable("project_name", projectName)
	builder.SetVariable("project_description", "Context-First AI Coding Framework")

	// Set OKR content (full content, not just list items)
	okrContent := formatOKRContent(contextMgr.GetOKRInfo())
	builder.SetVariable("okr_content", okrContent)

	// Set SPEC content (full content, not just list items)
	specContent := formatSPECContent(contextMgr.GetSPECInfo())
	builder.SetVariable("spec_content", specContent)

	// Set user requirement
	builder.SetVariable("user_requirement", requirement)

	// Set job plan directory
	builder.SetVariable("job_plan_dir", jobPlanDir)

	// Build and save to temporary file
	promptFile, err := builder.BuildAndSave("plan")
	if err != nil {
		return "", fmt.Errorf("failed to build and save plan prompt: %w", err)
	}

	return promptFile, nil
}

// formatOKRContent formats OKR information for the prompt
// Changed: Now returns full OKR content instead of formatted list
func formatOKRContent(okrInfo *parser.ContextInfo) string {
	if okrInfo == nil || len(okrInfo.Objectives) == 0 {
		return "暂无项目 OKR 信息"
	}

	// Return the full content (first element contains the complete OKR.md)
	return okrInfo.Objectives[0]
}

// formatSPECContent formats SPEC information for the prompt
// Changed: Now returns full SPEC content instead of formatted list
func formatSPECContent(specInfo *parser.ContextInfo) string {
	if specInfo == nil || len(specInfo.Specifications) == 0 {
		return "暂无项目 SPEC 信息"
	}

	// Return the full content (first element contains the complete SPEC.md)
	return specInfo.Specifications[0]
}

// formatCompletedWork formats completed work history for the prompt
// Deprecated: History/completed work feature removed as per requirements
// Keeping this function for potential future use
func formatCompletedWork(history []string) string {
	if len(history) == 0 {
		return "这是项目的第一阶段规划"
	}

	var content strings.Builder
	content.WriteString("**已完成的工作:**\n")

	for _, item := range history {
		content.WriteString(fmt.Sprintf("- %s\n", item))
	}

	return content.String()
}
