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

{{task_info_section}}

{{loops_context}}

{{debug_context}}`

	doingPath := filepath.Join(tmpDir, "doing.md")
	if err := os.WriteFile(doingPath, []byte(doingTemplate), 0644); err != nil {
		t.Fatalf("Failed to create doing template: %v", err)
	}

	// Create prompt manager
	manager := NewPromptManager(tmpDir)

	// Create context manager
	contextMgr := NewContextManager("job_1")

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

	// Verify loops_context is injected (not left as literal)
	if strings.Contains(prompt, "{{loops_context}}") {
		t.Error("Expected loops_context variable to be replaced")
	}
	if !strings.Contains(prompt, "可用的项目 Loops") {
		t.Error("Expected prompt to contain '可用的项目 Loops' from loops_context")
	}

}

func TestGenerateDoingPrompt_WithRetry(t *testing.T) {
	// Create temporary template directory
	tmpDir := t.TempDir()

	// Create doing.md template
	doingTemplate := `# Rick 项目执行阶段提示词

{{task_info_section}}

## Job 上下文

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

func TestGenerateDoingPrompt_LoopsContextInLoopList(t *testing.T) {
	tmpDir := t.TempDir()

	doingTemplate := `## Loop 列表
{{loops_context}}`

	doingPath := filepath.Join(tmpDir, "doing.md")
	if err := os.WriteFile(doingPath, []byte(doingTemplate), 0644); err != nil {
		t.Fatalf("Failed to create doing template: %v", err)
	}

	manager := NewPromptManager(tmpDir)
	contextMgr := NewContextManager("job_1")
	task := &parser.Task{ID: "task1", Name: "Test", Goal: "Test goal"}

	prompt, err := GenerateDoingPrompt(task, 0, contextMgr, manager)
	if err != nil {
		t.Fatalf("GenerateDoingPrompt failed: %v", err)
	}

	if strings.Contains(prompt, "{{loops_context}}") {
		t.Error("loops_context must be replaced (not left as literal)")
	}
	if !strings.Contains(prompt, "可用的项目 Loops") {
		t.Error("Expected prompt to contain loops_context output")
	}
}

func TestGenerateDoingPrompt_NoKeyResults(t *testing.T) {
	tmpDir := t.TempDir()

	doingTemplate := `# Rick 项目执行阶段提示词

{{task_info_section}}`

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

func TestGenerateDoingPrompt_LoopsContextInjected(t *testing.T) {
	tmpDir := t.TempDir()

	doingTemplate := `## 项目背景
{{loops_context}}`

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

	// loops_context must be replaced (not left as literal placeholder)
	if strings.Contains(prompt, "{{loops_context}}") {
		t.Error("Expected loops_context variable to be replaced, but literal {{loops_context}} found in output")
	}
	// LoadLoopsContext always returns a header, even on empty/missing dir
	if !strings.Contains(prompt, "可用的项目 Loops") {
		t.Error("Expected prompt to contain '可用的项目 Loops' from loops_context injection")
	}
}

func TestGenerateDoingPrompt_CompleteFlow(t *testing.T) {
	tmpDir := t.TempDir()

	doingTemplate := `# Rick 项目执行阶段提示词

{{task_info_section}}

## Loop 列表

{{loops_context}}

## Job 上下文

{{debug_context}}`

	doingPath := filepath.Join(tmpDir, "doing.md")
	if err := os.WriteFile(doingPath, []byte(doingTemplate), 0644); err != nil {
		t.Fatalf("Failed to create doing template: %v", err)
	}

	manager := NewPromptManager(tmpDir)
	contextMgr := NewContextManager("job_1")

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
		"可用的项目 Loops",
	}

	for _, content := range requiredContent {
		if !strings.Contains(prompt, content) {
			t.Errorf("Expected prompt to contain: %s", content)
		}
	}
}
