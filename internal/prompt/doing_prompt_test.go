package prompt

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sunquan/rick/internal/parser"
)

func TestGenerateDoingPrompt_Success(t *testing.T) {
	// Create temporary template directory
	tmpDir := t.TempDir()

	// Create doing.md template
	doingTemplate := `# Rick 项目执行阶段提示词

你是一个资深的软件工程师。你的任务是执行规划好的任务，完成具体的编码工作。

## 任务信息

**任务 ID**: {{task_id}}
**任务名称**: {{task_name}}
**重试次数**: {{retry_count}}

### 任务目标
{{task_objective}}

### 关键结果
{{key_results}}

### 测试方法
{{test_methods}}

## 项目背景

**项目名称**: {{project_name}}
**项目描述**: {{project_description}}

### 项目 SPEC
{{spec_content}}

### 项目架构
{{project_architecture}}

## 执行上下文

### 已完成的任务
{{completed_tasks}}

### 任务依赖
{{task_dependencies}}

{{#if retry_count > 0}}
### 前次执行的问题记录

根据前次执行遇到的问题，请重点关注以下内容：

{{debug_context}}

请确保这次执行能够解决之前遇到的问题。
{{/if}}

## 执行要求

1. **理解需求**: 仔细阅读任务目标和关键结果
2. **设计方案**: 根据项目架构和现有代码，设计实现方案
3. **编写代码**: 实现所有必要的功能
4. **测试验证**: 按照测试方法验证功能的正确性
5. **提交代码**: 使用 git 提交代码，提交信息应该清晰明确`

	doingPath := filepath.Join(tmpDir, "doing.md")
	if err := os.WriteFile(doingPath, []byte(doingTemplate), 0644); err != nil {
		t.Fatalf("Failed to create doing template: %v", err)
	}

	// Create prompt manager
	manager := NewPromptManager(tmpDir)

	// Create context manager
	contextMgr := NewContextManager("job_1")

	// Load SPEC
	specContent := "# Specifications\n- Use Go language\n- Support DAG execution"
	contextMgr.LoadSPECFromContent(specContent)

	// Load history
	contextMgr.LoadHistory([]string{"Module 1 completed", "Module 2 completed"})

	// Create a task
	task := &parser.Task{
		ID:           "task1",
		Name:         "实现提示词构建器",
		Goal:         "实现动态提示词构建功能",
		KeyResults:   []string{"完成 PromptBuilder 类型定义", "实现 Build() 方法", "编写单元测试"},
		TestMethod:   "运行 go test ./internal/prompt",
		Dependencies: []string{},
	}

	// Generate doing prompt
	prompt, err := GenerateDoingPrompt(task, 0, contextMgr, manager)

	if err != nil {
		t.Fatalf("GenerateDoingPrompt failed: %v", err)
	}

	// Verify prompt contains task information
	if !strings.Contains(prompt, "task1") {
		t.Error("Expected prompt to contain task ID")
	}

	if !strings.Contains(prompt, "实现提示词构建器") {
		t.Error("Expected prompt to contain task name")
	}

	if !strings.Contains(prompt, "实现动态提示词构建功能") {
		t.Error("Expected prompt to contain task goal")
	}

	// Verify prompt contains key results
	if !strings.Contains(prompt, "完成 PromptBuilder 类型定义") {
		t.Error("Expected prompt to contain key results")
	}

	// Verify prompt contains test method
	if !strings.Contains(prompt, "go test") {
		t.Error("Expected prompt to contain test method")
	}

	// Verify prompt contains project information
	if !strings.Contains(prompt, "Rick CLI") {
		t.Error("Expected prompt to contain project name")
	}

	// Verify prompt contains SPEC information
	if !strings.Contains(prompt, "Use Go language") {
		t.Error("Expected prompt to contain SPEC information")
	}

	// Verify prompt contains completed tasks
	if !strings.Contains(prompt, "Module 1 completed") {
		t.Error("Expected prompt to contain completed tasks")
	}
}

func TestGenerateDoingPrompt_WithRetry(t *testing.T) {
	// Create temporary template directory
	tmpDir := t.TempDir()

	// Create doing.md template
	doingTemplate := `# Rick 项目执行阶段提示词

## 任务信息

**任务 ID**: {{task_id}}
**任务名称**: {{task_name}}
**重试次数**: {{retry_count}}

### 任务目标
{{task_objective}}

### 关键结果
{{key_results}}

### 测试方法
{{test_methods}}

## 项目背景

**项目名称**: {{project_name}}
**项目描述**: {{project_description}}

### 项目 SPEC
{{spec_content}}

### 项目架构
{{project_architecture}}

## 执行上下文

### 已完成的任务
{{completed_tasks}}

### 任务依赖
{{task_dependencies}}

### 前次执行的问题记录

根据前次执行遇到的问题，请重点关注以下内容：

{{debug_context}}`

	doingPath := filepath.Join(tmpDir, "doing.md")
	if err := os.WriteFile(doingPath, []byte(doingTemplate), 0644); err != nil {
		t.Fatalf("Failed to create doing template: %v", err)
	}

	// Create prompt manager
	manager := NewPromptManager(tmpDir)

	// Create context manager
	contextMgr := NewContextManager("job_1")

	// Load SPEC
	specContent := "# Specifications\n- Use Go language"
	contextMgr.LoadSPECFromContent(specContent)

	// Load debug information
	debugContent := `**调试日志**:
- debug1: 编译错误, 执行 make 时报错, 猜想: 缺少导入包, 验证: 检查导入, 修复: 添加 import "fmt", 已修复`
	contextMgr.LoadDebugFromContent(debugContent)

	// Create a task
	task := &parser.Task{
		ID:           "task2",
		Name:         "实现上下文管理器",
		Goal:         "实现执行上下文管理功能",
		KeyResults:   []string{"完成 ContextManager 类型定义", "实现 Load 方法"},
		TestMethod:   "运行 go test ./internal/prompt",
		Dependencies: []string{"task1"},
	}

	// Generate doing prompt with retry
	prompt, err := GenerateDoingPrompt(task, 1, contextMgr, manager)

	if err != nil {
		t.Fatalf("GenerateDoingPrompt failed: %v", err)
	}

	// Verify retry count is set
	if !strings.Contains(prompt, "1") {
		t.Error("Expected prompt to contain retry count")
	}

	// Verify debug context is included
	if !strings.Contains(prompt, "debug1") || !strings.Contains(prompt, "编译错误") {
		t.Error("Expected prompt to contain debug information")
	}

	// Verify task dependency is included
	if !strings.Contains(prompt, "task1") {
		t.Error("Expected prompt to contain task dependency")
	}
}

func TestGenerateDoingPrompt_NilTask(t *testing.T) {
	tmpDir := t.TempDir()
	manager := NewPromptManager(tmpDir)
	contextMgr := NewContextManager("job_1")

	_, err := GenerateDoingPrompt(nil, 0, contextMgr, manager)

	if err == nil {
		t.Error("Expected error for nil task")
	}

	if !strings.Contains(err.Error(), "task cannot be nil") {
		t.Errorf("Expected error message about nil task, got: %v", err)
	}
}

func TestGenerateDoingPrompt_NilContextManager(t *testing.T) {
	tmpDir := t.TempDir()
	manager := NewPromptManager(tmpDir)
	task := &parser.Task{
		ID:   "task1",
		Name: "Test Task",
		Goal: "Test goal",
	}

	_, err := GenerateDoingPrompt(task, 0, nil, manager)

	if err == nil {
		t.Error("Expected error for nil context manager")
	}

	if !strings.Contains(err.Error(), "context manager cannot be nil") {
		t.Errorf("Expected error message about nil context manager, got: %v", err)
	}
}

func TestGenerateDoingPrompt_NilPromptManager(t *testing.T) {
	contextMgr := NewContextManager("job_1")
	task := &parser.Task{
		ID:   "task1",
		Name: "Test Task",
		Goal: "Test goal",
	}

	_, err := GenerateDoingPrompt(task, 0, contextMgr, nil)

	if err == nil {
		t.Error("Expected error for nil prompt manager")
	}

	if !strings.Contains(err.Error(), "prompt manager cannot be nil") {
		t.Errorf("Expected error message about nil prompt manager, got: %v", err)
	}
}

func TestGenerateDoingPrompt_MissingTemplate(t *testing.T) {
	// This test is now obsolete because we have embedded templates as fallback
	// Even if the template directory is empty, the embedded template will be used
	t.Skip("Skipping test - embedded templates now provide fallback")

	tmpDir := t.TempDir()
	manager := NewPromptManager(tmpDir)
	contextMgr := NewContextManager("job_1")

	task := &parser.Task{
		ID:   "task1",
		Name: "Test Task",
		Goal: "Test goal",
	}

	// With embedded templates, this should now succeed
	_, err := GenerateDoingPrompt(task, 0, contextMgr, manager)

	if err != nil {
		t.Errorf("Unexpected error with embedded template fallback: %v", err)
	}
}

func TestGenerateDoingPrompt_WithDependencies(t *testing.T) {
	tmpDir := t.TempDir()

	doingTemplate := `# Rick 项目执行阶段提示词

## 任务信息

**任务 ID**: {{task_id}}
**任务名称**: {{task_name}}

### 任务目标
{{task_objective}}

### 任务依赖
{{task_dependencies}}`

	doingPath := filepath.Join(tmpDir, "doing.md")
	if err := os.WriteFile(doingPath, []byte(doingTemplate), 0644); err != nil {
		t.Fatalf("Failed to create doing template: %v", err)
	}

	manager := NewPromptManager(tmpDir)
	contextMgr := NewContextManager("job_1")

	task := &parser.Task{
		ID:             "task3",
		Name:           "Test Task",
		Goal:           "Test goal",
		Dependencies:   []string{"task1", "task2"},
		KeyResults:     []string{},
		TestMethod:     "",
	}

	prompt, err := GenerateDoingPrompt(task, 0, contextMgr, manager)

	if err != nil {
		t.Fatalf("GenerateDoingPrompt failed: %v", err)
	}

	// Verify task dependencies are included
	if !strings.Contains(prompt, "task1") || !strings.Contains(prompt, "task2") {
		t.Error("Expected prompt to contain all task dependencies")
	}

	if !strings.Contains(prompt, "该任务依赖以下任务的完成") {
		t.Error("Expected prompt to contain dependency header")
	}
}

func TestGenerateDoingPrompt_NoKeyResults(t *testing.T) {
	tmpDir := t.TempDir()

	doingTemplate := `# Rick 项目执行阶段提示词

### 关键结果
{{key_results}}`

	doingPath := filepath.Join(tmpDir, "doing.md")
	if err := os.WriteFile(doingPath, []byte(doingTemplate), 0644); err != nil {
		t.Fatalf("Failed to create doing template: %v", err)
	}

	manager := NewPromptManager(tmpDir)
	contextMgr := NewContextManager("job_1")

	task := &parser.Task{
		ID:           "task1",
		Name:         "Test Task",
		Goal:         "Test goal",
		KeyResults:   []string{},
		TestMethod:   "",
		Dependencies: []string{},
	}

	prompt, err := GenerateDoingPrompt(task, 0, contextMgr, manager)

	if err != nil {
		t.Fatalf("GenerateDoingPrompt failed: %v", err)
	}

	if !strings.Contains(prompt, "暂无关键结果") {
		t.Error("Expected prompt to contain 'no key results' message")
	}
}

func TestGenerateDoingPrompt_CompleteFlow(t *testing.T) {
	tmpDir := t.TempDir()

	doingTemplate := `# Rick 项目执行阶段提示词

## 任务信息

**任务 ID**: {{task_id}}
**任务名称**: {{task_name}}
**重试次数**: {{retry_count}}

### 任务目标
{{task_objective}}

### 关键结果
{{key_results}}

### 测试方法
{{test_methods}}

## 项目背景

**项目名称**: {{project_name}}
**项目描述**: {{project_description}}

### 项目 SPEC
{{spec_content}}

### 项目架构
{{project_architecture}}

## 执行上下文

### 已完成的任务
{{completed_tasks}}

### 任务依赖
{{task_dependencies}}`

	doingPath := filepath.Join(tmpDir, "doing.md")
	if err := os.WriteFile(doingPath, []byte(doingTemplate), 0644); err != nil {
		t.Fatalf("Failed to create doing template: %v", err)
	}

	manager := NewPromptManager(tmpDir)
	contextMgr := NewContextManager("job_1")

	// Load SPEC
	specContent := "# Specifications\n- Use Go language\n- Support DAG execution"
	contextMgr.LoadSPECFromContent(specContent)

	// Load history
	contextMgr.LoadHistory([]string{"Infrastructure module completed", "Parser module completed"})

	task := &parser.Task{
		ID:           "task1",
		Name:         "实现提示词构建器",
		Goal:         "实现动态提示词构建功能",
		KeyResults:   []string{"完成 PromptBuilder 类型定义", "实现 Build() 方法", "编写单元测试"},
		TestMethod:   "运行 go test ./internal/prompt",
		Dependencies: []string{},
	}

	prompt, err := GenerateDoingPrompt(task, 0, contextMgr, manager)

	if err != nil {
		t.Fatalf("GenerateDoingPrompt failed: %v", err)
	}

	// Comprehensive verification
	requiredContent := []string{
		"task1",
		"实现提示词构建器",
		"实现动态提示词构建功能",
		"完成 PromptBuilder 类型定义",
		"实现 Build() 方法",
		"编写单元测试",
		"go test",
		"Rick CLI",
		"Context-First AI Coding Framework",
		"Use Go language",
		"Support DAG execution",
		"Infrastructure module completed",
		"Parser module completed",
		"该任务无依赖关系",
	}

	for _, content := range requiredContent {
		if !strings.Contains(prompt, content) {
			t.Errorf("Expected prompt to contain: %s", content)
		}
	}
}
