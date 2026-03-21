package prompt

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sunquan/rick/internal/parser"
)

func TestGenerateLearningPrompt_Success(t *testing.T) {
	// Create temporary template directory
	tmpDir := t.TempDir()

	// Create learning.md template
	learningTemplate := `# Rick 项目学习阶段提示词

你是一个资深的技术文档专家和知识管理专家。你的任务是根据项目执行过程，总结知识、经验和教训。

## 项目信息

**项目名称**: {{project_name}}
**项目描述**: {{project_description}}
**执行周期**: {{job_id}}

## 执行摘要

### 已完成的工作
{{completed_work_summary}}

### 任务执行结果
{{task_execution_results}}

### 遇到的问题和解决方案

#### 问题记录
{{debug_records}}

#### 解决方案总结
{{solutions_summary}}

## Git 提交历史

` + "```" + `
{{git_history}}
` + "```" + `

## 代码变更分析

### 新增功能
{{new_features}}

### 代码改进
{{code_improvements}}

### 技术债务
{{technical_debt}}`

	learningPath := filepath.Join(tmpDir, "learning.md")
	if err := os.WriteFile(learningPath, []byte(learningTemplate), 0644); err != nil {
		t.Fatalf("Failed to create learning template: %v", err)
	}

	// Create prompt manager
	manager := NewPromptManager(tmpDir)

	// Create context manager
	contextMgr := NewContextManager("job_7")

	// Load task
	task := &parser.Task{
		ID:         "task_1",
		Name:       "Learning Task",
		Goal:       "Summarize project execution",
		KeyResults: []string{"KR1", "KR2"},
		TestMethod: "Review learning summary",
	}
	contextMgr.LoadTask(task)

	// Load history
	history := []string{
		"Implemented learning prompt generation",
		"Created context manager",
		"Added test cases",
	}
	contextMgr.LoadHistory(history)

	// Generate learning prompt
	prompt, err := GenerateLearningPrompt("job_7", contextMgr, manager)
	if err != nil {
		t.Fatalf("Failed to generate learning prompt: %v", err)
	}

	// Verify prompt is not empty
	if prompt == "" {
		t.Error("Generated prompt is empty")
	}

	// Verify prompt contains project information (project_name is dynamically resolved)
	if strings.Contains(prompt, "{{project_name}}") {
		t.Error("Prompt should have project_name variable replaced")
	}

	if !strings.Contains(prompt, "job_7") {
		t.Error("Prompt should contain job ID")
	}

	// Verify prompt contains completed work
	if !strings.Contains(prompt, "Implemented learning prompt generation") {
		t.Error("Prompt should contain completed work")
	}

	// Verify prompt contains task information
	if !strings.Contains(prompt, "task_1") {
		t.Error("Prompt should contain task ID")
	}

	if !strings.Contains(prompt, "Learning Task") {
		t.Error("Prompt should contain task name")
	}

	// Verify prompt contains sections
	if !strings.Contains(prompt, "已完成的工作") {
		t.Error("Prompt should contain completed work section")
	}

	if !strings.Contains(prompt, "任务执行结果") {
		t.Error("Prompt should contain task execution results section")
	}

	if !strings.Contains(prompt, "问题记录") {
		t.Error("Prompt should contain debug records section")
	}

	if !strings.Contains(prompt, "Git 提交历史") {
		t.Error("Prompt should contain git history section")
	}

	if !strings.Contains(prompt, "新增功能") {
		t.Error("Prompt should contain new features section")
	}

	if !strings.Contains(prompt, "代码改进") {
		t.Error("Prompt should contain code improvements section")
	}

	if !strings.Contains(prompt, "技术债务") {
		t.Error("Prompt should contain technical debt section")
	}
}

func TestGenerateLearningPrompt_EmptyJobID(t *testing.T) {
	tmpDir := t.TempDir()
	manager := NewPromptManager(tmpDir)
	contextMgr := NewContextManager("test_job")

	_, err := GenerateLearningPrompt("", contextMgr, manager)
	if err == nil {
		t.Error("Expected error for empty job ID")
	}
	if !strings.Contains(err.Error(), "job ID cannot be empty") {
		t.Errorf("Unexpected error message: %v", err)
	}
}

func TestGenerateLearningPrompt_NilContextManager(t *testing.T) {
	tmpDir := t.TempDir()
	manager := NewPromptManager(tmpDir)

	_, err := GenerateLearningPrompt("job_1", nil, manager)
	if err == nil {
		t.Error("Expected error for nil context manager")
	}
	if !strings.Contains(err.Error(), "context manager cannot be nil") {
		t.Errorf("Unexpected error message: %v", err)
	}
}

func TestGenerateLearningPrompt_NilPromptManager(t *testing.T) {
	contextMgr := NewContextManager("test_job")

	_, err := GenerateLearningPrompt("job_1", contextMgr, nil)
	if err == nil {
		t.Error("Expected error for nil prompt manager")
	}
	if !strings.Contains(err.Error(), "prompt manager cannot be nil") {
		t.Errorf("Unexpected error message: %v", err)
	}
}

func TestGenerateLearningPrompt_MissingTemplate(t *testing.T) {
	// This test is now obsolete because we have embedded templates as fallback
	t.Skip("Skipping test - embedded templates now provide fallback")

	tmpDir := t.TempDir()
	manager := NewPromptManager(tmpDir)
	contextMgr := NewContextManager("test_job")

	// With embedded templates, this should now succeed
	_, err := GenerateLearningPrompt("job_1", contextMgr, manager)
	if err != nil {
		t.Errorf("Unexpected error with embedded template fallback: %v", err)
	}
}

func TestGenerateLearningPrompt_WithDebugRecords(t *testing.T) {
	tmpDir := t.TempDir()

	learningTemplate := `# Learning Prompt

Job: {{job_id}}

## Debug Records
{{debug_records}}

## Solutions
{{solutions_summary}}`

	learningPath := filepath.Join(tmpDir, "learning.md")
	if err := os.WriteFile(learningPath, []byte(learningTemplate), 0644); err != nil {
		t.Fatalf("Failed to create learning template: %v", err)
	}

	manager := NewPromptManager(tmpDir)
	contextMgr := NewContextManager("job_7")

	// Load debug info with proper format
	debugContent := "**调试日志**:\n- debug1: Template loading failed, Call LoadTemplate with invalid name, Template file not found, Check file existence, Add error handling, 已修复"
	contextMgr.LoadDebugFromContent(debugContent)

	prompt, err := GenerateLearningPrompt("job_7", contextMgr, manager)
	if err != nil {
		t.Fatalf("Failed to generate learning prompt: %v", err)
	}

	// Verify prompt contains debug records
	if !strings.Contains(prompt, "Template loading failed") {
		t.Error("Prompt should contain debug records")
	}

	if !strings.Contains(prompt, "已修复") {
		t.Error("Prompt should contain solution status")
	}
}

func TestGenerateLearningPrompt_WithEmptyHistory(t *testing.T) {
	tmpDir := t.TempDir()

	learningTemplate := `# Learning Prompt

## Completed Work
{{completed_work_summary}}`

	learningPath := filepath.Join(tmpDir, "learning.md")
	if err := os.WriteFile(learningPath, []byte(learningTemplate), 0644); err != nil {
		t.Fatalf("Failed to create learning template: %v", err)
	}

	manager := NewPromptManager(tmpDir)
	contextMgr := NewContextManager("job_7")

	// Don't load any history

	prompt, err := GenerateLearningPrompt("job_7", contextMgr, manager)
	if err != nil {
		t.Fatalf("Failed to generate learning prompt: %v", err)
	}

	// Verify prompt handles empty history
	if !strings.Contains(prompt, "暂无已完成的工作") {
		t.Error("Prompt should indicate no completed work")
	}
}

func TestGenerateLearningPrompt_WithMultipleHistoryItems(t *testing.T) {
	tmpDir := t.TempDir()

	learningTemplate := `# Learning Prompt

## Completed Work
{{completed_work_summary}}`

	learningPath := filepath.Join(tmpDir, "learning.md")
	if err := os.WriteFile(learningPath, []byte(learningTemplate), 0644); err != nil {
		t.Fatalf("Failed to create learning template: %v", err)
	}

	manager := NewPromptManager(tmpDir)
	contextMgr := NewContextManager("job_7")

	// Load multiple history items
	history := []string{
		"Task 1: Implemented feature A",
		"Task 2: Fixed bug B",
		"Task 3: Added test C",
		"Task 4: Refactored module D",
	}
	contextMgr.LoadHistory(history)

	prompt, err := GenerateLearningPrompt("job_7", contextMgr, manager)
	if err != nil {
		t.Fatalf("Failed to generate learning prompt: %v", err)
	}

	// Verify all history items are included
	for _, item := range history {
		if !strings.Contains(prompt, item) {
			t.Errorf("Prompt should contain history item: %s", item)
		}
	}

	// Verify items are numbered
	if !strings.Contains(prompt, "1.") && !strings.Contains(prompt, "2.") {
		t.Error("History items should be numbered")
	}
}

func TestGenerateLearningPrompt_VariableReplacement(t *testing.T) {
	tmpDir := t.TempDir()

	learningTemplate := `# Learning Prompt

Project: {{project_name}} - {{project_description}}
Job: {{job_id}}
Work: {{completed_work_summary}}
Results: {{task_execution_results}}
Debug: {{debug_records}}
Solutions: {{solutions_summary}}
History: {{git_history}}
Features: {{new_features}}
Improvements: {{code_improvements}}
Debt: {{technical_debt}}`

	learningPath := filepath.Join(tmpDir, "learning.md")
	if err := os.WriteFile(learningPath, []byte(learningTemplate), 0644); err != nil {
		t.Fatalf("Failed to create learning template: %v", err)
	}

	manager := NewPromptManager(tmpDir)
	contextMgr := NewContextManager("job_7")

	prompt, err := GenerateLearningPrompt("job_7", contextMgr, manager)
	if err != nil {
		t.Fatalf("Failed to generate learning prompt: %v", err)
	}

	// Verify all variables are replaced
	if strings.Contains(prompt, "{{") {
		t.Error("Prompt should not contain unreplaced variables")
	}

	// Verify specific replacements (project_name is dynamically resolved)
	if strings.Contains(prompt, "{{project_name}}") {
		t.Error("Project name variable should be replaced")
	}

	if !strings.Contains(prompt, "job_7") {
		t.Error("Job ID should be replaced")
	}

	if !strings.Contains(prompt, "Context-First AI Coding Framework") {
		t.Error("Project description should be replaced")
	}
}

func TestFormatCompletedWorkSummary(t *testing.T) {
	// Test with empty history
	result := formatCompletedWorkSummary([]string{})
	if !strings.Contains(result, "暂无已完成的工作") {
		t.Error("Should indicate no completed work for empty history")
	}

	// Test with items
	history := []string{"Item 1", "Item 2", "Item 3"}
	result = formatCompletedWorkSummary(history)

	for _, item := range history {
		if !strings.Contains(result, item) {
			t.Errorf("Should contain history item: %s", item)
		}
	}

	// Verify numbering
	if !strings.Contains(result, "1.") || !strings.Contains(result, "2.") || !strings.Contains(result, "3.") {
		t.Error("Items should be numbered")
	}
}

func TestFormatTaskExecutionResults(t *testing.T) {
	// Test with nil task
	result := formatTaskExecutionResults(nil)
	if !strings.Contains(result, "暂无任务执行结果") {
		t.Error("Should indicate no results for nil task")
	}

	// Test with task
	task := &parser.Task{
		ID:         "task_1",
		Name:       "Test Task",
		Goal:       "Test objective",
		KeyResults: []string{"KR1", "KR2"},
		TestMethod: "go test",
	}

	result = formatTaskExecutionResults(task)

	if !strings.Contains(result, "task_1") {
		t.Error("Should contain task ID")
	}

	if !strings.Contains(result, "Test Task") {
		t.Error("Should contain task name")
	}

	if !strings.Contains(result, "Test objective") {
		t.Error("Should contain task goal")
	}

	if !strings.Contains(result, "KR1") || !strings.Contains(result, "KR2") {
		t.Error("Should contain key results")
	}

	if !strings.Contains(result, "go test") {
		t.Error("Should contain test method")
	}
}

func TestFormatLearningDebugRecords(t *testing.T) {
	// Test with nil debug info
	result := formatLearningDebugRecords(nil)
	if !strings.Contains(result, "暂无问题记录") {
		t.Error("Should indicate no records for nil debug info")
	}

	// Test with debug entries
	debugInfo := &parser.DebugInfo{
		Entries: []parser.DebugEntry{
			{
				ID:         1,
				Phenomenon: "Issue 1",
				Reproduce:  "Reproduce 1",
				Hypothesis: "Hypothesis 1",
				Fix:        "Fix 1",
				Progress:   "已修复",
			},
			{
				ID:         2,
				Phenomenon: "Issue 2",
				Reproduce:  "Reproduce 2",
			},
		},
	}

	result = formatLearningDebugRecords(debugInfo)

	if !strings.Contains(result, "Issue 1") {
		t.Error("Should contain issue 1")
	}

	if !strings.Contains(result, "Issue 2") {
		t.Error("Should contain issue 2")
	}

	if !strings.Contains(result, "已修复") {
		t.Error("Should contain progress status")
	}
}

func TestFormatSolutionsSummary(t *testing.T) {
	// Test with nil debug info
	result := formatSolutionsSummary(nil)
	if !strings.Contains(result, "暂无解决方案") {
		t.Error("Should indicate no solutions for nil debug info")
	}

	// Test with no solved issues
	debugInfo := &parser.DebugInfo{
		Entries: []parser.DebugEntry{
			{
				ID:         1,
				Phenomenon: "Issue 1",
				Fix:        "Fix 1",
				Progress:   "待修复",
			},
		},
	}

	result = formatSolutionsSummary(debugInfo)
	if !strings.Contains(result, "暂无已解决的问题") {
		t.Error("Should indicate no solved issues")
	}

	// Test with solved issues
	debugInfo = &parser.DebugInfo{
		Entries: []parser.DebugEntry{
			{
				ID:         1,
				Phenomenon: "Issue 1",
				Fix:        "Fix 1",
				Progress:   "已修复",
			},
			{
				ID:         2,
				Phenomenon: "Issue 2",
				Fix:        "Fix 2",
				Progress:   "已修复",
			},
		},
	}

	result = formatSolutionsSummary(debugInfo)

	if !strings.Contains(result, "Fix 1") {
		t.Error("Should contain solution 1")
	}

	if !strings.Contains(result, "Fix 2") {
		t.Error("Should contain solution 2")
	}

	if !strings.Contains(result, "Issue 1") {
		t.Error("Should contain issue reference")
	}
}

func TestFormatGitHistory(t *testing.T) {
	result := formatGitHistory()

	if !strings.Contains(result, "Git") {
		t.Error("Should mention Git")
	}

	if !strings.Contains(result, "提交") {
		t.Error("Should mention commits")
	}
}

func TestFormatNewFeatures(t *testing.T) {
	result := formatNewFeatures()

	if !strings.Contains(result, "学习阶段提示词生成") {
		t.Error("Should mention learning prompt generation")
	}

	if !strings.Contains(result, "learning_prompt.go") {
		t.Error("Should mention learning_prompt.go")
	}

	if !strings.Contains(result, "GenerateLearningPrompt") {
		t.Error("Should mention GenerateLearningPrompt function")
	}
}

func TestFormatCodeImprovements(t *testing.T) {
	result := formatCodeImprovements()

	if !strings.Contains(result, "提示词管理模块") {
		t.Error("Should mention prompt manager module")
	}

	if !strings.Contains(result, "上下文管理器") {
		t.Error("Should mention context manager")
	}

	if !strings.Contains(result, "测试覆盖") {
		t.Error("Should mention test coverage")
	}
}

func TestFormatTechnicalDebt(t *testing.T) {
	result := formatTechnicalDebt()

	if !strings.Contains(result, "技术债务") {
		t.Error("Should mention technical debt")
	}

	if !strings.Contains(result, "Git") {
		t.Error("Should mention Git-related debt")
	}

	if !strings.Contains(result, "性能优化") {
		t.Error("Should mention performance optimization")
	}
}
