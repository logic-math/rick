package prompt

import (
	"fmt"
	"strings"

	"github.com/sunquan/rick/internal/parser"
)

// GeneratePlanPrompt generates the planning phase prompt from user requirement
// It includes project context (OKR, SPEC) and task format specification
func GeneratePlanPrompt(requirement string, contextMgr *ContextManager, manager *PromptManager) (string, error) {
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
	builder.SetVariable("project_name", "Rick CLI")
	builder.SetVariable("project_description", "Context-First AI Coding Framework")

	// Set OKR content
	okrContent := formatOKRContent(contextMgr.GetOKRInfo())
	builder.SetVariable("okr_content", okrContent)

	// Set SPEC content
	specContent := formatSPECContent(contextMgr.GetSPECInfo())
	builder.SetVariable("spec_content", specContent)

	// Set completed work
	completedWork := formatCompletedWork(contextMgr.GetHistory())
	builder.SetVariable("completed_work", completedWork)

	// Set user requirement
	builder.SetVariable("user_requirement", requirement)

	// Build final prompt
	prompt, err := builder.Build()
	if err != nil {
		return "", fmt.Errorf("failed to build plan prompt: %w", err)
	}

	return prompt, nil
}

// formatOKRContent formats OKR information for the prompt
func formatOKRContent(okrInfo *parser.ContextInfo) string {
	if okrInfo == nil || (len(okrInfo.Objectives) == 0 && len(okrInfo.KeyResults) == 0) {
		return "暂无项目 OKR 信息"
	}

	var content strings.Builder

	// Add objectives
	if len(okrInfo.Objectives) > 0 {
		content.WriteString("**Objectives:**\n")
		for _, obj := range okrInfo.Objectives {
			content.WriteString(fmt.Sprintf("- %s\n", obj))
		}
		content.WriteString("\n")
	}

	// Add key results
	if len(okrInfo.KeyResults) > 0 {
		content.WriteString("**Key Results:**\n")
		for _, kr := range okrInfo.KeyResults {
			content.WriteString(fmt.Sprintf("- %s\n", kr))
		}
	}

	return content.String()
}

// formatSPECContent formats SPEC information for the prompt
func formatSPECContent(specInfo *parser.ContextInfo) string {
	if specInfo == nil || len(specInfo.Specifications) == 0 {
		return "暂无项目 SPEC 信息"
	}

	var content strings.Builder
	content.WriteString("**Specifications:**\n")

	for _, spec := range specInfo.Specifications {
		content.WriteString(fmt.Sprintf("- %s\n", spec))
	}

	return content.String()
}

// formatCompletedWork formats completed work history for the prompt
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
