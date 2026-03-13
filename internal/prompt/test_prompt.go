package prompt

import (
	"fmt"

	"github.com/sunquan/rick/internal/parser"
)

// GenerateTestPrompt generates the test script generation prompt from a task and implementation code
// It includes task information, test methods, implementation code, and test script format specification
func GenerateTestPrompt(task *parser.Task, implementationCode string, contextMgr *ContextManager, manager *PromptManager) (string, error) {
	if task == nil {
		return "", fmt.Errorf("task cannot be nil")
	}

	if implementationCode == "" {
		return "", fmt.Errorf("implementation code cannot be empty")
	}

	if contextMgr == nil {
		return "", fmt.Errorf("context manager cannot be nil")
	}

	if manager == nil {
		return "", fmt.Errorf("prompt manager cannot be nil")
	}

	// Load test template
	template, err := manager.LoadTemplate("test")
	if err != nil {
		return "", fmt.Errorf("failed to load test template: %w", err)
	}

	// Create prompt builder
	builder := NewPromptBuilder(template)

	// Set task information
	builder.SetVariable("task_id", task.ID)
	builder.SetVariable("task_name", task.Name)

	// Set task details
	builder.SetVariable("task_objective", task.Goal)
	builder.SetVariable("key_results", formatKeyResults(task.KeyResults))
	builder.SetVariable("test_methods", task.TestMethod)

	// Set implementation code
	builder.SetVariable("implementation_code", implementationCode)

	// Set project information
	builder.SetVariable("project_name", "Rick CLI")
	builder.SetVariable("project_type", "Command-line Tool")
	builder.SetVariable("project_language", "Go")

	// Set test framework information
	testFramework := formatTestFramework()
	builder.SetVariable("test_framework", testFramework)

	// Set existing test examples
	existingTests := formatExistingTests()
	builder.SetVariable("existing_tests", existingTests)

	// Build final prompt
	prompt, err := builder.Build()
	if err != nil {
		return "", fmt.Errorf("failed to build test prompt: %w", err)
	}

	return prompt, nil
}

// formatTestFramework formats test framework information for the prompt
func formatTestFramework() string {
	return `Rick 项目使用 Go 标准库中的 testing 包进行单元测试：

**测试框架特点**:
- 使用 Go 标准库 testing 包
- 测试文件命名规范：*_test.go
- 测试函数签名：func TestXxx(t *testing.T)
- 使用 t.Errorf() 报告失败
- 使用 t.Fatal() 报告致命错误
- 支持表驱动测试（Table-Driven Tests）

**测试执行**:
- 运行所有测试: go test ./...
- 运行指定包的测试: go test ./internal/prompt
- 运行指定测试: go test -run TestName
- 生成覆盖率报告: go test -cover ./...
- 生成详细覆盖率: go test -coverprofile=coverage.out ./...`
}

// formatExistingTests formats existing test examples for the prompt
func formatExistingTests() string {
	return `**现有测试示例** (internal/prompt/*_test.go):

1. **manager_test.go** - 提示词管理器测试
   - 模板加载测试
   - 模板缓存测试
   - 错误处理测试

2. **builder_test.go** - 提示词构建器测试
   - 变量替换测试
   - 上下文注入测试
   - 提示词构建测试

3. **context_test.go** - 上下文管理器测试
   - 任务加载测试
   - Debug 加载测试
   - OKR/SPEC 加载测试

4. **plan_prompt_test.go** - 规划提示词生成测试
   - 提示词生成测试
   - 变量包含测试
   - 格式验证测试

5. **doing_prompt_test.go** - 执行提示词生成测试
   - 提示词生成测试
   - 重试上下文测试
   - Debug 信息包含测试

**测试模式**:
- 使用表驱动测试设计多个测试用例
- 每个测试包含 Arrange-Act-Assert 三个阶段
- 使用 t.Run() 分组相关测试
- 使用 t.Parallel() 并行运行独立测试`
}
