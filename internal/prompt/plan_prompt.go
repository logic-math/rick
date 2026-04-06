package prompt

import (
	"fmt"
	"strings"

	"github.com/sunquan/rick/internal/parser"
	"github.com/sunquan/rick/internal/workspace"
)

// GeneratePlanPrompt generates the planning phase prompt from user requirement.
// jobPlanDir is the absolute path to the job's plan directory.
// rickDir is optional: when non-empty, skills index from .rick/skills/index.md is injected.
func GeneratePlanPrompt(requirement string, jobPlanDir string, contextMgr *ContextManager, manager *PromptManager, rickDir ...string) (string, error) {
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

	// Set OKR content: use parsed result, fall back to raw file content
	okrContent := formatOKRContent(contextMgr.GetOKRInfo())
	if okrContent == "暂无项目 OKR 信息" && contextMgr.GetOKRRaw() != "" {
		okrContent = contextMgr.GetOKRRaw()
	}
	builder.SetVariable("okr_content", okrContent)

	// Set SPEC content: use parsed result, fall back to raw file content
	specContent := formatSPECContent(contextMgr.GetSPECInfo())
	if specContent == "暂无项目 SPEC 信息" && contextMgr.GetSPECRaw() != "" {
		specContent = contextMgr.GetSPECRaw()
	}
	builder.SetVariable("spec_content", specContent)

	// Set user requirement
	builder.SetVariable("user_requirement", requirement)

	// Set completed work history
	completedWork := formatCompletedWork(contextMgr.GetHistory())
	builder.SetVariable("completed_work", completedWork)

	// Set job plan directory
	builder.SetVariable("job_plan_dir", jobPlanDir)

	// Set skills index
	resolvedRickDir := ""
	if len(rickDir) > 0 {
		resolvedRickDir = rickDir[0]
	}
	skillsIndex := formatSkillsIndexSection(resolvedRickDir)
	builder.SetVariable("skills_index", skillsIndex)

	// Build final prompt
	prompt, err := builder.Build()
	if err != nil {
		return "", fmt.Errorf("failed to build plan prompt: %w", err)
	}

	return prompt, nil
}

// GeneratePlanPromptFile generates the planning phase prompt and saves it to a temporary file.
// jobPlanDir is the absolute path to the job's plan directory (e.g. .rick/jobs/job_1/plan).
// rickDir is optional: when non-empty, skills index from .rick/skills/index.md is injected.
// Returns the file path and any error. The caller is responsible for cleaning up the temporary file.
func GeneratePlanPromptFile(requirement string, jobPlanDir string, contextMgr *ContextManager, manager *PromptManager, rickDir ...string) (string, error) {
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

	// Set OKR content: use parsed result, fall back to raw file content
	okrContent2 := formatOKRContent(contextMgr.GetOKRInfo())
	if okrContent2 == "暂无项目 OKR 信息" && contextMgr.GetOKRRaw() != "" {
		okrContent2 = contextMgr.GetOKRRaw()
	}
	builder.SetVariable("okr_content", okrContent2)

	// Set SPEC content: use parsed result, fall back to raw file content
	specContent2 := formatSPECContent(contextMgr.GetSPECInfo())
	if specContent2 == "暂无项目 SPEC 信息" && contextMgr.GetSPECRaw() != "" {
		specContent2 = contextMgr.GetSPECRaw()
	}
	builder.SetVariable("spec_content", specContent2)

	// Set user requirement
	builder.SetVariable("user_requirement", requirement)

	// Set completed work history
	completedWork2 := formatCompletedWork(contextMgr.GetHistory())
	builder.SetVariable("completed_work", completedWork2)

	// Set job plan directory
	builder.SetVariable("job_plan_dir", jobPlanDir)

	// Set skills index
	resolvedRickDir2 := ""
	if len(rickDir) > 0 {
		resolvedRickDir2 = rickDir[0]
	}
	skillsIndex2 := formatSkillsIndexSection(resolvedRickDir2)
	builder.SetVariable("skills_index", skillsIndex2)

	// Build and save to temporary file
	promptFile, err := builder.BuildAndSave("plan")
	if err != nil {
		return "", fmt.Errorf("failed to build and save plan prompt: %w", err)
	}

	return promptFile, nil
}

// formatSkillsIndexSection returns the skills index content for injection into plan prompts.
// Returns empty string if rickDir is empty or index.md doesn't exist.
func formatSkillsIndexSection(rickDir string) string {
	if rickDir == "" {
		return ""
	}
	content, err := workspace.LoadSkillsIndex(rickDir)
	if err != nil || content == "" {
		return ""
	}
	return content
}

// formatOKRContent formats OKR information for the prompt
func formatOKRContent(okrInfo *parser.ContextInfo) string {
	if okrInfo == nil || (len(okrInfo.Objectives) == 0 && len(okrInfo.KeyResults) == 0) {
		return "暂无项目 OKR 信息"
	}

	var content strings.Builder

	if len(okrInfo.Objectives) > 0 {
		content.WriteString("**Objectives**:\n")
		for _, obj := range okrInfo.Objectives {
			content.WriteString(fmt.Sprintf("- %s\n", obj))
		}
		content.WriteString("\n")
	}

	if len(okrInfo.KeyResults) > 0 {
		content.WriteString("**Key Results**:\n")
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
	content.WriteString("**Specifications**:\n")
	for _, spec := range specInfo.Specifications {
		content.WriteString(fmt.Sprintf("- %s\n", spec))
	}

	return content.String()
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
