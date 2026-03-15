package prompt

import (
	"fmt"
	"strings"

	"github.com/sunquan/rick/internal/parser"
)

// GenerateDoingPrompt generates the execution phase prompt from a task
// It includes task information, test methods, and debug context if it's a retry
func GenerateDoingPrompt(task *parser.Task, retryCount int, contextMgr *ContextManager, manager *PromptManager) (string, error) {
	if task == nil {
		return "", fmt.Errorf("task cannot be nil")
	}

	if contextMgr == nil {
		return "", fmt.Errorf("context manager cannot be nil")
	}

	if manager == nil {
		return "", fmt.Errorf("prompt manager cannot be nil")
	}

	// Load doing template
	template, err := manager.LoadTemplate("doing")
	if err != nil {
		return "", fmt.Errorf("failed to load doing template: %w", err)
	}

	// Create prompt builder
	builder := NewPromptBuilder(template)

	// Set task information
	builder.SetVariable("task_id", task.ID)
	builder.SetVariable("task_name", task.Name)
	builder.SetVariable("retry_count", fmt.Sprintf("%d", retryCount))

	// Set task details
	builder.SetVariable("task_objective", task.Goal)
	builder.SetVariable("key_results", formatKeyResults(task.KeyResults))
	builder.SetVariable("test_methods", task.TestMethod)

	// Set project information
	builder.SetVariable("project_name", "Rick CLI")
	builder.SetVariable("project_description", "Context-First AI Coding Framework")

	// Set SPEC content
	specContent := formatSPECContent(contextMgr.GetSPECInfo())
	builder.SetVariable("spec_content", specContent)

	// Set project architecture
	projectArch := formatProjectArchitecture()
	builder.SetVariable("project_architecture", projectArch)

	// Set completed tasks
	completedTasks := formatCompletedTasks(contextMgr.GetHistory())
	builder.SetVariable("completed_tasks", completedTasks)

	// Set task dependencies
	taskDeps := formatTaskDependencies(task.Dependencies)
	builder.SetVariable("task_dependencies", taskDeps)

	// If it's a retry, include debug context
	if retryCount > 0 {
		debugContext := formatDebugContext(contextMgr.GetDebug())
		builder.SetVariable("debug_context", debugContext)
	} else {
		// For non-retry, set empty debug context
		builder.SetVariable("debug_context", "")
	}

	// Build final prompt
	prompt, err := builder.Build()
	if err != nil {
		return "", fmt.Errorf("failed to build doing prompt: %w", err)
	}

	return prompt, nil
}

// GenerateDoingPromptFile generates the execution phase prompt and saves it to a temporary file
// Returns the file path and any error
// The caller is responsible for cleaning up the temporary file
func GenerateDoingPromptFile(task *parser.Task, retryCount int, contextMgr *ContextManager, manager *PromptManager) (string, error) {
	if task == nil {
		return "", fmt.Errorf("task cannot be nil")
	}

	if contextMgr == nil {
		return "", fmt.Errorf("context manager cannot be nil")
	}

	if manager == nil {
		return "", fmt.Errorf("prompt manager cannot be nil")
	}

	// Load doing template
	template, err := manager.LoadTemplate("doing")
	if err != nil {
		return "", fmt.Errorf("failed to load doing template: %w", err)
	}

	// Create prompt builder
	builder := NewPromptBuilder(template)

	// Set task information
	builder.SetVariable("task_id", task.ID)
	builder.SetVariable("task_name", task.Name)
	builder.SetVariable("retry_count", fmt.Sprintf("%d", retryCount))

	// Set task details
	builder.SetVariable("task_objective", task.Goal)
	builder.SetVariable("key_results", formatKeyResults(task.KeyResults))
	builder.SetVariable("test_methods", task.TestMethod)

	// Set project information
	builder.SetVariable("project_name", "Rick CLI")
	builder.SetVariable("project_description", "Context-First AI Coding Framework")

	// Set SPEC content
	specContent := formatSPECContent(contextMgr.GetSPECInfo())
	builder.SetVariable("spec_content", specContent)

	// Set project architecture
	projectArch := formatProjectArchitecture()
	builder.SetVariable("project_architecture", projectArch)

	// Set completed tasks
	completedTasks := formatCompletedTasks(contextMgr.GetHistory())
	builder.SetVariable("completed_tasks", completedTasks)

	// Set task dependencies
	taskDeps := formatTaskDependencies(task.Dependencies)
	builder.SetVariable("task_dependencies", taskDeps)

	// If it's a retry, include debug context
	if retryCount > 0 {
		debugContext := formatDebugContext(contextMgr.GetDebug())
		builder.SetVariable("debug_context", debugContext)
	} else {
		// For non-retry, set empty debug context
		builder.SetVariable("debug_context", "")
	}

	// Build and save to temporary file
	promptFile, err := builder.BuildAndSave(fmt.Sprintf("doing-%s", task.ID))
	if err != nil {
		return "", fmt.Errorf("failed to build and save doing prompt: %w", err)
	}

	return promptFile, nil
}

// formatKeyResults formats key results for the prompt
func formatKeyResults(keyResults []string) string {
	if len(keyResults) == 0 {
		return "暂无关键结果"
	}

	var content strings.Builder
	for i, kr := range keyResults {
		content.WriteString(fmt.Sprintf("%d. %s\n", i+1, kr))
	}

	return content.String()
}

// formatProjectArchitecture formats project architecture information
func formatProjectArchitecture() string {
	return `Rick 项目采用模块化架构设计：

**核心模块**:
- infrastructure: 基础设施模块（Go 项目初始化、CLI、工作空间、配置、日志）
- parser: 内容解析模块（Markdown、task.md、debug.md、OKR/SPEC 解析）
- dag_executor: DAG 执行模块（DAG 构建、拓扑排序、任务执行、重试机制）
- prompt_manager: 提示词管理模块（模板、构建、上下文、各阶段提示词生成）
- cli_commands: 命令处理模块（init、plan、doing、learning 命令）

**关键设计**:
- 使用 Go 标准库为主，最小化外部依赖
- 提示词管理是核心创新，支持多阶段提示词生成
- 任务执行采用 DAG 拓扑排序，支持并行和串行执行
- 失败重试机制，超过限制后需人工干预`
}

// formatCompletedTasks formats completed tasks for the prompt
func formatCompletedTasks(history []string) string {
	if len(history) == 0 {
		return "暂无已完成的任务"
	}

	var content strings.Builder
	for _, item := range history {
		content.WriteString(fmt.Sprintf("- %s\n", item))
	}

	return content.String()
}

// formatTaskDependencies formats task dependencies for the prompt
func formatTaskDependencies(dependencies []string) string {
	if len(dependencies) == 0 {
		return "该任务无依赖关系"
	}

	var content strings.Builder
	content.WriteString("该任务依赖以下任务的完成：\n")
	for _, dep := range dependencies {
		content.WriteString(fmt.Sprintf("- %s\n", dep))
	}

	return content.String()
}

// formatDebugContext formats debug information for retry prompts
func formatDebugContext(debugInfo *parser.DebugInfo) string {
	if debugInfo == nil || len(debugInfo.Entries) == 0 {
		return "暂无问题记录"
	}

	var content strings.Builder
	for _, entry := range debugInfo.Entries {
		content.WriteString(fmt.Sprintf("**debug%d: %s**\n", entry.ID, entry.Phenomenon))

		if entry.Reproduce != "" {
			content.WriteString(fmt.Sprintf("- 复现: %s\n", entry.Reproduce))
		}

		if entry.Hypothesis != "" {
			content.WriteString(fmt.Sprintf("- 猜想: %s\n", entry.Hypothesis))
		}

		if entry.Verify != "" {
			content.WriteString(fmt.Sprintf("- 验证: %s\n", entry.Verify))
		}

		if entry.Fix != "" {
			content.WriteString(fmt.Sprintf("- 修复: %s\n", entry.Fix))
		}

		if entry.Progress != "" {
			content.WriteString(fmt.Sprintf("- 进展: %s\n", entry.Progress))
		}

		content.WriteString("\n")
	}

	return content.String()
}
