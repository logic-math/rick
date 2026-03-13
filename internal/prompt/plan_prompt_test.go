package prompt

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sunquan/rick/internal/parser"
)

func TestGeneratePlanPrompt_Success(t *testing.T) {
	// Create temporary template directory
	tmpDir := t.TempDir()

	// Create plan.md template
	planTemplate := `# Rick 项目规划阶段提示词

## 项目背景

**项目名称**: {{project_name}}
**项目描述**: {{project_description}}

### 项目 OKR
{{okr_content}}

### 项目 SPEC
{{spec_content}}

### 已完成的工作
{{completed_work}}

## 规划任务

根据上述背景，请为以下需求生成详细的规划文档：

**用户需求**: {{user_requirement}}`

	planPath := filepath.Join(tmpDir, "plan.md")
	if err := os.WriteFile(planPath, []byte(planTemplate), 0644); err != nil {
		t.Fatalf("Failed to create plan template: %v", err)
	}

	// Create prompt manager
	manager := NewPromptManager(tmpDir)

	// Create context manager
	contextMgr := NewContextManager("job_1")

	// Load OKR
	okrContent := "# Objectives\n- Build Rick CLI\n\n# Key Results\n- Complete 8 modules"
	contextMgr.LoadOKRFromContent(okrContent)

	// Load SPEC
	specContent := "# Specifications\n- Use Go language\n- Support DAG execution"
	contextMgr.LoadSPECFromContent(specContent)

	// Load history
	contextMgr.LoadHistory([]string{"Module 1 completed", "Module 2 completed"})

	// Generate plan prompt
	requirement := "Implement prompt management system"
	prompt, err := GeneratePlanPrompt(requirement, contextMgr, manager)

	if err != nil {
		t.Fatalf("GeneratePlanPrompt failed: %v", err)
	}

	// Verify prompt contains project information
	if !strings.Contains(prompt, "Rick CLI") {
		t.Error("Expected prompt to contain project name")
	}

	if !strings.Contains(prompt, "Context-First AI Coding Framework") {
		t.Error("Expected prompt to contain project description")
	}

	// Verify prompt contains user requirement
	if !strings.Contains(prompt, requirement) {
		t.Error("Expected prompt to contain user requirement")
	}

	// Verify prompt contains OKR information
	if !strings.Contains(prompt, "Build Rick CLI") {
		t.Error("Expected prompt to contain OKR information")
	}

	// Verify prompt contains SPEC information
	if !strings.Contains(prompt, "Use Go language") {
		t.Error("Expected prompt to contain SPEC information")
	}

	// Verify prompt contains completed work
	if !strings.Contains(prompt, "Module 1 completed") {
		t.Error("Expected prompt to contain completed work")
	}
}

func TestGeneratePlanPrompt_EmptyRequirement(t *testing.T) {
	tmpDir := t.TempDir()
	manager := NewPromptManager(tmpDir)
	contextMgr := NewContextManager("job_1")

	prompt, err := GeneratePlanPrompt("", contextMgr, manager)

	if err == nil {
		t.Error("Expected error for empty requirement, got nil")
	}

	if prompt != "" {
		t.Error("Expected empty prompt for empty requirement")
	}
}

func TestGeneratePlanPrompt_NilContextManager(t *testing.T) {
	tmpDir := t.TempDir()
	manager := NewPromptManager(tmpDir)

	prompt, err := GeneratePlanPrompt("test requirement", nil, manager)

	if err == nil {
		t.Error("Expected error for nil context manager, got nil")
	}

	if prompt != "" {
		t.Error("Expected empty prompt for nil context manager")
	}
}

func TestGeneratePlanPrompt_NilManager(t *testing.T) {
	contextMgr := NewContextManager("job_1")

	prompt, err := GeneratePlanPrompt("test requirement", contextMgr, nil)

	if err == nil {
		t.Error("Expected error for nil manager, got nil")
	}

	if prompt != "" {
		t.Error("Expected empty prompt for nil manager")
	}
}

func TestGeneratePlanPrompt_NoOKRInfo(t *testing.T) {
	tmpDir := t.TempDir()

	// Create plan.md template
	planTemplate := `## 项目 OKR
{{okr_content}}

**用户需求**: {{user_requirement}}`

	planPath := filepath.Join(tmpDir, "plan.md")
	if err := os.WriteFile(planPath, []byte(planTemplate), 0644); err != nil {
		t.Fatalf("Failed to create plan template: %v", err)
	}

	manager := NewPromptManager(tmpDir)
	contextMgr := NewContextManager("job_1")

	prompt, err := GeneratePlanPrompt("test requirement", contextMgr, manager)

	if err != nil {
		t.Fatalf("GeneratePlanPrompt failed: %v", err)
	}

	// Verify prompt contains default OKR message
	if !strings.Contains(prompt, "暂无项目 OKR 信息") {
		t.Error("Expected prompt to contain default OKR message when no OKR is loaded")
	}
}

func TestGeneratePlanPrompt_NoSPECInfo(t *testing.T) {
	tmpDir := t.TempDir()

	// Create plan.md template
	planTemplate := `## 项目 SPEC
{{spec_content}}

**用户需求**: {{user_requirement}}`

	planPath := filepath.Join(tmpDir, "plan.md")
	if err := os.WriteFile(planPath, []byte(planTemplate), 0644); err != nil {
		t.Fatalf("Failed to create plan template: %v", err)
	}

	manager := NewPromptManager(tmpDir)
	contextMgr := NewContextManager("job_1")

	prompt, err := GeneratePlanPrompt("test requirement", contextMgr, manager)

	if err != nil {
		t.Fatalf("GeneratePlanPrompt failed: %v", err)
	}

	// Verify prompt contains default SPEC message
	if !strings.Contains(prompt, "暂无项目 SPEC 信息") {
		t.Error("Expected prompt to contain default SPEC message when no SPEC is loaded")
	}
}

func TestGeneratePlanPrompt_NoHistory(t *testing.T) {
	tmpDir := t.TempDir()

	// Create plan.md template
	planTemplate := `### 已完成的工作
{{completed_work}}

**用户需求**: {{user_requirement}}`

	planPath := filepath.Join(tmpDir, "plan.md")
	if err := os.WriteFile(planPath, []byte(planTemplate), 0644); err != nil {
		t.Fatalf("Failed to create plan template: %v", err)
	}

	manager := NewPromptManager(tmpDir)
	contextMgr := NewContextManager("job_1")

	prompt, err := GeneratePlanPrompt("test requirement", contextMgr, manager)

	if err != nil {
		t.Fatalf("GeneratePlanPrompt failed: %v", err)
	}

	// Verify prompt contains default completed work message
	if !strings.Contains(prompt, "这是项目的第一阶段规划") {
		t.Error("Expected prompt to contain default completed work message when no history is loaded")
	}
}

func TestFormatOKRContent_WithData(t *testing.T) {
	okrInfo := &parser.ContextInfo{
		Objectives: []string{"Build Rick CLI", "Improve performance"},
		KeyResults: []string{"Complete 8 modules", "Achieve 90% test coverage"},
	}

	content := formatOKRContent(okrInfo)

	if !strings.Contains(content, "Objectives") {
		t.Error("Expected content to contain 'Objectives'")
	}

	if !strings.Contains(content, "Build Rick CLI") {
		t.Error("Expected content to contain objective")
	}

	if !strings.Contains(content, "Key Results") {
		t.Error("Expected content to contain 'Key Results'")
	}

	if !strings.Contains(content, "Complete 8 modules") {
		t.Error("Expected content to contain key result")
	}
}

func TestFormatOKRContent_Empty(t *testing.T) {
	okrInfo := &parser.ContextInfo{}
	content := formatOKRContent(okrInfo)

	if content != "暂无项目 OKR 信息" {
		t.Errorf("Expected default message, got %s", content)
	}
}

func TestFormatOKRContent_Nil(t *testing.T) {
	content := formatOKRContent(nil)

	if content != "暂无项目 OKR 信息" {
		t.Errorf("Expected default message, got %s", content)
	}
}

func TestFormatSPECContent_WithData(t *testing.T) {
	specInfo := &parser.ContextInfo{
		Specifications: []string{"Use Go language", "Support DAG execution", "Minimal dependencies"},
	}

	content := formatSPECContent(specInfo)

	if !strings.Contains(content, "Specifications") {
		t.Error("Expected content to contain 'Specifications'")
	}

	if !strings.Contains(content, "Use Go language") {
		t.Error("Expected content to contain specification")
	}
}

func TestFormatSPECContent_Empty(t *testing.T) {
	specInfo := &parser.ContextInfo{}
	content := formatSPECContent(specInfo)

	if content != "暂无项目 SPEC 信息" {
		t.Errorf("Expected default message, got %s", content)
	}
}

func TestFormatCompletedWork_WithHistory(t *testing.T) {
	history := []string{"Module 1 completed", "Module 2 completed", "Tests added"}
	content := formatCompletedWork(history)

	if !strings.Contains(content, "已完成的工作") {
		t.Error("Expected content to contain '已完成的工作'")
	}

	if !strings.Contains(content, "Module 1 completed") {
		t.Error("Expected content to contain history item")
	}
}

func TestFormatCompletedWork_Empty(t *testing.T) {
	history := []string{}
	content := formatCompletedWork(history)

	if content != "这是项目的第一阶段规划" {
		t.Errorf("Expected default message, got %s", content)
	}
}

func TestGeneratePlanPrompt_IncludesDependencyInfo(t *testing.T) {
	tmpDir := t.TempDir()

	// Create plan.md template with dependency info
	planTemplate := `# 规划阶段

## 项目背景

**项目名称**: {{project_name}}

## 规划任务

**用户需求**: {{user_requirement}}

## 任务格式规范

### 依赖关系

任务可以有依赖关系，格式如下：

` + "`" + `
# 依赖关系
task1, task2, ...
` + "`" + `

### 任务目标

每个任务应该有明确的目标和验收标准。`

	planPath := filepath.Join(tmpDir, "plan.md")
	if err := os.WriteFile(planPath, []byte(planTemplate), 0644); err != nil {
		t.Fatalf("Failed to create plan template: %v", err)
	}

	manager := NewPromptManager(tmpDir)
	contextMgr := NewContextManager("job_1")

	prompt, err := GeneratePlanPrompt("test requirement", contextMgr, manager)

	if err != nil {
		t.Fatalf("GeneratePlanPrompt failed: %v", err)
	}

	// Verify prompt includes dependency information
	if !strings.Contains(prompt, "依赖关系") {
		t.Error("Expected prompt to include dependency relationship information")
	}

	if !strings.Contains(prompt, "任务目标") {
		t.Error("Expected prompt to include task objective information")
	}
}

func TestGeneratePlanPrompt_FormatCorrect(t *testing.T) {
	tmpDir := t.TempDir()

	// Create plan.md template
	planTemplate := `# 规划提示词

项目: {{project_name}}
需求: {{user_requirement}}
OKR: {{okr_content}}
SPEC: {{spec_content}}
历史: {{completed_work}}`

	planPath := filepath.Join(tmpDir, "plan.md")
	if err := os.WriteFile(planPath, []byte(planTemplate), 0644); err != nil {
		t.Fatalf("Failed to create plan template: %v", err)
	}

	manager := NewPromptManager(tmpDir)
	contextMgr := NewContextManager("job_1")

	contextMgr.LoadOKRFromContent("Objective: Test")
	contextMgr.LoadSPECFromContent("Spec: Test")
	contextMgr.LoadHistory([]string{"Task 1"})

	prompt, err := GeneratePlanPrompt("test requirement", contextMgr, manager)

	if err != nil {
		t.Fatalf("GeneratePlanPrompt failed: %v", err)
	}

	// Verify all variables are replaced
	if strings.Contains(prompt, "{{") {
		t.Error("Expected all variables to be replaced, found unreplaced variables")
	}

	// Verify no empty placeholders
	if strings.Contains(prompt, "}}") {
		t.Error("Expected no unreplaced placeholders")
	}
}
