package prompt

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sunquan/rick/internal/parser"
)

func TestGenerateTestPrompt_Success(t *testing.T) {
	// Create temporary template directory
	tmpDir := t.TempDir()

	// Create test.md template
	testTemplate := `# Rick 项目测试脚本生成提示词

你是一个资深的测试工程师和质量保证专家。你的任务是根据任务描述和实现代码，生成全面的测试脚本。

## 任务信息

**任务 ID**: {{task_id}}
**任务名称**: {{task_name}}

### 任务目标
{{task_objective}}

### 关键结果
{{key_results}}

### 测试方法
{{test_methods}}

## 实现代码

` + "```" + `
{{implementation_code}}
` + "```" + `

## 项目背景

**项目名称**: {{project_name}}
**项目类型**: {{project_type}}
**项目语言**: {{project_language}}

### 测试框架
{{test_framework}}

### 现有测试示例
{{existing_tests}}

## 测试脚本生成要求

1. **覆盖所有功能**: 测试应该覆盖实现的所有功能
2. **边界条件**: 包括边界情况和异常情况的测试
3. **集成测试**: 测试与其他模块的集成
4. **性能测试**: 如果需要，包括性能测试
5. **可读性**: 测试代码应该清晰易读`

	testPath := filepath.Join(tmpDir, "test.md")
	if err := os.WriteFile(testPath, []byte(testTemplate), 0644); err != nil {
		t.Fatalf("Failed to create test template: %v", err)
	}

	// Create prompt manager
	manager := NewPromptManager(tmpDir)

	// Create context manager
	contextMgr := NewContextManager("test_job")

	// Create test task
	task := &parser.Task{
		ID:          "task_1",
		Name:        "Test Task",
		Goal:        "Implement test functionality",
		KeyResults:  []string{"Result 1", "Result 2"},
		TestMethod:  "Run: go test ./...",
		Dependencies: []string{},
	}

	// Test implementation code
	implementationCode := `func TestExample(t *testing.T) {
	result := Add(1, 2)
	if result != 3 {
		t.Errorf("Expected 3, got %d", result)
	}
}`

	// Generate test prompt
	prompt, err := GenerateTestPrompt(task, implementationCode, contextMgr, manager)
	if err != nil {
		t.Fatalf("Failed to generate test prompt: %v", err)
	}

	// Verify prompt is not empty
	if prompt == "" {
		t.Error("Generated prompt is empty")
	}

	// Verify prompt contains task information
	if !strings.Contains(prompt, "task_1") {
		t.Error("Prompt should contain task ID")
	}

	if !strings.Contains(prompt, "Test Task") {
		t.Error("Prompt should contain task name")
	}

	// Verify prompt contains task objective
	if !strings.Contains(prompt, "Implement test functionality") {
		t.Error("Prompt should contain task objective")
	}

	// Verify prompt contains key results
	if !strings.Contains(prompt, "Result 1") {
		t.Error("Prompt should contain key results")
	}

	// Verify prompt contains test method
	if !strings.Contains(prompt, "go test") {
		t.Error("Prompt should contain test method")
	}

	// Verify prompt contains implementation code
	if !strings.Contains(prompt, "TestExample") {
		t.Error("Prompt should contain implementation code")
	}

	// Verify prompt contains project information
	if !strings.Contains(prompt, "Rick CLI") {
		t.Error("Prompt should contain project name")
	}

	if !strings.Contains(prompt, "Go") {
		t.Error("Prompt should contain project language")
	}

	// Verify prompt contains test framework information
	if !strings.Contains(prompt, "testing") {
		t.Error("Prompt should contain test framework information")
	}

	// Verify prompt contains existing tests information
	if !strings.Contains(prompt, "manager_test.go") {
		t.Error("Prompt should contain existing tests information")
	}
}

func TestGenerateTestPrompt_NilTask(t *testing.T) {
	tmpDir := t.TempDir()
	manager := NewPromptManager(tmpDir)
	contextMgr := NewContextManager("test_job")

	_, err := GenerateTestPrompt(nil, "code", contextMgr, manager)
	if err == nil {
		t.Error("Expected error for nil task")
	}
	if !strings.Contains(err.Error(), "task cannot be nil") {
		t.Errorf("Unexpected error message: %v", err)
	}
}

func TestGenerateTestPrompt_EmptyCode(t *testing.T) {
	tmpDir := t.TempDir()
	manager := NewPromptManager(tmpDir)
	contextMgr := NewContextManager("test_job")

	task := &parser.Task{
		ID:   "task_1",
		Name: "Test Task",
	}

	_, err := GenerateTestPrompt(task, "", contextMgr, manager)
	if err == nil {
		t.Error("Expected error for empty implementation code")
	}
	if !strings.Contains(err.Error(), "implementation code cannot be empty") {
		t.Errorf("Unexpected error message: %v", err)
	}
}

func TestGenerateTestPrompt_NilContextManager(t *testing.T) {
	tmpDir := t.TempDir()
	manager := NewPromptManager(tmpDir)

	task := &parser.Task{
		ID:   "task_1",
		Name: "Test Task",
	}

	_, err := GenerateTestPrompt(task, "code", nil, manager)
	if err == nil {
		t.Error("Expected error for nil context manager")
	}
	if !strings.Contains(err.Error(), "context manager cannot be nil") {
		t.Errorf("Unexpected error message: %v", err)
	}
}

func TestGenerateTestPrompt_NilPromptManager(t *testing.T) {
	contextMgr := NewContextManager("test_job")

	task := &parser.Task{
		ID:   "task_1",
		Name: "Test Task",
	}

	_, err := GenerateTestPrompt(task, "code", contextMgr, nil)
	if err == nil {
		t.Error("Expected error for nil prompt manager")
	}
	if !strings.Contains(err.Error(), "prompt manager cannot be nil") {
		t.Errorf("Unexpected error message: %v", err)
	}
}

func TestGenerateTestPrompt_MissingTemplate(t *testing.T) {
	tmpDir := t.TempDir()
	manager := NewPromptManager(tmpDir)
	contextMgr := NewContextManager("test_job")

	task := &parser.Task{
		ID:   "task_1",
		Name: "Test Task",
	}

	_, err := GenerateTestPrompt(task, "code", contextMgr, manager)
	if err == nil {
		t.Error("Expected error for missing template")
	}
	if !strings.Contains(err.Error(), "failed to load test template") {
		t.Errorf("Unexpected error message: %v", err)
	}
}

func TestGenerateTestPrompt_WithComplexCode(t *testing.T) {
	// Create temporary template directory
	tmpDir := t.TempDir()

	// Create test.md template
	testTemplate := `# Test Prompt

Task: {{task_id}}
Code: {{implementation_code}}
Framework: {{test_framework}}
Tests: {{existing_tests}}`

	testPath := filepath.Join(tmpDir, "test.md")
	if err := os.WriteFile(testPath, []byte(testTemplate), 0644); err != nil {
		t.Fatalf("Failed to create test template: %v", err)
	}

	manager := NewPromptManager(tmpDir)
	contextMgr := NewContextManager("test_job")

	task := &parser.Task{
		ID:   "task_1",
		Name: "Test Task",
		Goal: "Implement complex functionality",
	}

	// Complex implementation code with multiple functions
	complexCode := `package example

import "testing"

func TestAdd(t *testing.T) {
	cases := []struct {
		a, b, want int
	}{
		{1, 2, 3},
		{0, 0, 0},
		{-1, 1, 0},
	}
	for _, c := range cases {
		if got := Add(c.a, c.b); got != c.want {
			t.Errorf("Add(%d, %d) = %d, want %d", c.a, c.b, got, c.want)
		}
	}
}

func TestMultiply(t *testing.T) {
	result := Multiply(3, 4)
	if result != 12 {
		t.Errorf("Expected 12, got %d", result)
	}
}`

	prompt, err := GenerateTestPrompt(task, complexCode, contextMgr, manager)
	if err != nil {
		t.Fatalf("Failed to generate test prompt: %v", err)
	}

	// Verify complex code is preserved in prompt
	if !strings.Contains(prompt, "TestAdd") {
		t.Error("Prompt should contain TestAdd function")
	}

	if !strings.Contains(prompt, "TestMultiply") {
		t.Error("Prompt should contain TestMultiply function")
	}

	if !strings.Contains(prompt, "table-driven test") || !strings.Contains(prompt, "cases") {
		// Code structure should be preserved
		if !strings.Contains(prompt, "cases :=") {
			t.Error("Prompt should preserve code structure")
		}
	}
}

func TestGenerateTestPrompt_WithMultipleKeyResults(t *testing.T) {
	tmpDir := t.TempDir()

	testTemplate := `# Test Prompt

Task: {{task_id}}
Objective: {{task_objective}}
Key Results: {{key_results}}
Test Methods: {{test_methods}}`

	testPath := filepath.Join(tmpDir, "test.md")
	if err := os.WriteFile(testPath, []byte(testTemplate), 0644); err != nil {
		t.Fatalf("Failed to create test template: %v", err)
	}

	manager := NewPromptManager(tmpDir)
	contextMgr := NewContextManager("test_job")

	task := &parser.Task{
		ID:   "task_1",
		Name: "Test Task",
		Goal: "Implement comprehensive testing",
		KeyResults: []string{
			"All functions have unit tests",
			"Test coverage >= 80%",
			"Edge cases are covered",
			"Performance tests pass",
		},
		TestMethod: "go test -cover ./internal/prompt",
	}

	prompt, err := GenerateTestPrompt(task, "func TestExample(t *testing.T) {}", contextMgr, manager)
	if err != nil {
		t.Fatalf("Failed to generate test prompt: %v", err)
	}

	// Verify all key results are included
	for i, kr := range task.KeyResults {
		if !strings.Contains(prompt, kr) {
			t.Errorf("Prompt should contain key result %d: %s", i+1, kr)
		}
	}

	// Verify test method is included
	if !strings.Contains(prompt, "go test -cover") {
		t.Error("Prompt should contain test method")
	}
}

func TestGenerateTestPrompt_PromptStructure(t *testing.T) {
	tmpDir := t.TempDir()

	// Create test.md template with all sections
	testTemplate := `# Rick 项目测试脚本生成提示词

## 任务信息

Task ID: {{task_id}}
Task Name: {{task_name}}

## 实现代码

{{implementation_code}}

## 项目信息

Name: {{project_name}}
Type: {{project_type}}
Language: {{project_language}}

## 测试框架

{{test_framework}}

## 现有测试

{{existing_tests}}`

	testPath := filepath.Join(tmpDir, "test.md")
	if err := os.WriteFile(testPath, []byte(testTemplate), 0644); err != nil {
		t.Fatalf("Failed to create test template: %v", err)
	}

	manager := NewPromptManager(tmpDir)
	contextMgr := NewContextManager("test_job")

	task := &parser.Task{
		ID:   "task_1",
		Name: "Test Task",
	}

	prompt, err := GenerateTestPrompt(task, "code", contextMgr, manager)
	if err != nil {
		t.Fatalf("Failed to generate test prompt: %v", err)
	}

	// Verify prompt has expected sections
	if !strings.Contains(prompt, "## 任务信息") {
		t.Error("Prompt should contain task information section")
	}

	if !strings.Contains(prompt, "## 实现代码") {
		t.Error("Prompt should contain implementation code section")
	}

	if !strings.Contains(prompt, "## 项目信息") {
		t.Error("Prompt should contain project information section")
	}

	if !strings.Contains(prompt, "## 测试框架") {
		t.Error("Prompt should contain test framework section")
	}

	if !strings.Contains(prompt, "## 现有测试") {
		t.Error("Prompt should contain existing tests section")
	}
}

func TestFormatTestFramework(t *testing.T) {
	framework := formatTestFramework()

	// Verify framework information is present
	if !strings.Contains(framework, "testing") {
		t.Error("Framework should mention testing package")
	}

	if !strings.Contains(framework, "testing.T") {
		t.Error("Framework should mention testing.T")
	}

	if !strings.Contains(framework, "*_test.go") {
		t.Error("Framework should mention test file naming convention")
	}

	if !strings.Contains(framework, "go test") {
		t.Error("Framework should mention go test command")
	}

	if !strings.Contains(framework, "t.Errorf") {
		t.Error("Framework should mention error reporting")
	}
}

func TestFormatExistingTests(t *testing.T) {
	tests := formatExistingTests()

	// Verify existing tests information is present
	if !strings.Contains(tests, "manager_test.go") {
		t.Error("Should mention manager_test.go")
	}

	if !strings.Contains(tests, "builder_test.go") {
		t.Error("Should mention builder_test.go")
	}

	if !strings.Contains(tests, "context_test.go") {
		t.Error("Should mention context_test.go")
	}

	if !strings.Contains(tests, "plan_prompt_test.go") {
		t.Error("Should mention plan_prompt_test.go")
	}

	if !strings.Contains(tests, "doing_prompt_test.go") {
		t.Error("Should mention doing_prompt_test.go")
	}

	if !strings.Contains(tests, "表驱动测试") {
		t.Error("Should mention table-driven tests")
	}

	if !strings.Contains(tests, "Arrange-Act-Assert") {
		t.Error("Should mention AAA pattern")
	}
}

func TestGenerateTestPrompt_VariableReplacement(t *testing.T) {
	tmpDir := t.TempDir()

	// Create test.md template with all variables
	testTemplate := `# Test Prompt

Task: {{task_id}} - {{task_name}}
Objective: {{task_objective}}
Results: {{key_results}}
Methods: {{test_methods}}
Code: {{implementation_code}}
Project: {{project_name}} ({{project_type}}, {{project_language}})
Framework: {{test_framework}}
Tests: {{existing_tests}}`

	testPath := filepath.Join(tmpDir, "test.md")
	if err := os.WriteFile(testPath, []byte(testTemplate), 0644); err != nil {
		t.Fatalf("Failed to create test template: %v", err)
	}

	manager := NewPromptManager(tmpDir)
	contextMgr := NewContextManager("test_job")

	task := &parser.Task{
		ID:         "test_task_123",
		Name:       "My Test Task",
		Goal:       "Test the system",
		KeyResults: []string{"KR1", "KR2"},
		TestMethod: "go test",
	}

	code := "func Test() {}"

	prompt, err := GenerateTestPrompt(task, code, contextMgr, manager)
	if err != nil {
		t.Fatalf("Failed to generate test prompt: %v", err)
	}

	// Verify all variables are replaced
	if strings.Contains(prompt, "{{") {
		t.Error("Prompt should not contain unreplaced variables")
	}

	// Verify specific replacements
	if !strings.Contains(prompt, "test_task_123") {
		t.Error("Task ID should be replaced")
	}

	if !strings.Contains(prompt, "My Test Task") {
		t.Error("Task name should be replaced")
	}

	if !strings.Contains(prompt, "Test the system") {
		t.Error("Task objective should be replaced")
	}

	if !strings.Contains(prompt, "Command-line Tool") {
		t.Error("Project type should be replaced")
	}
}
