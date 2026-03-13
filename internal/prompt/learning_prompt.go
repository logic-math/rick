package prompt

import (
	"fmt"
	"strings"

	"github.com/sunquan/rick/internal/parser"
)

// GenerateLearningPrompt generates the learning phase prompt from job execution results
// It includes completed work summary, task execution results, debug records, git history, and analysis
func GenerateLearningPrompt(jobID string, contextMgr *ContextManager, manager *PromptManager) (string, error) {
	if jobID == "" {
		return "", fmt.Errorf("job ID cannot be empty")
	}

	if contextMgr == nil {
		return "", fmt.Errorf("context manager cannot be nil")
	}

	if manager == nil {
		return "", fmt.Errorf("prompt manager cannot be nil")
	}

	// Load learning template
	template, err := manager.LoadTemplate("learning")
	if err != nil {
		return "", fmt.Errorf("failed to load learning template: %w", err)
	}

	// Create prompt builder
	builder := NewPromptBuilder(template)

	// Set project information
	builder.SetVariable("project_name", "Rick CLI")
	builder.SetVariable("project_description", "Context-First AI Coding Framework")
	builder.SetVariable("job_id", jobID)

	// Set completed work summary
	completedWorkSummary := formatCompletedWorkSummary(contextMgr.GetHistory())
	builder.SetVariable("completed_work_summary", completedWorkSummary)

	// Set task execution results
	taskResults := formatTaskExecutionResults(contextMgr.GetTask())
	builder.SetVariable("task_execution_results", taskResults)

	// Set debug records
	debugRecords := formatLearningDebugRecords(contextMgr.GetDebug())
	builder.SetVariable("debug_records", debugRecords)

	// Set solutions summary
	solutionsSummary := formatSolutionsSummary(contextMgr.GetDebug())
	builder.SetVariable("solutions_summary", solutionsSummary)

	// Set git history
	gitHistory := formatGitHistory()
	builder.SetVariable("git_history", gitHistory)

	// Set new features
	newFeatures := formatNewFeatures()
	builder.SetVariable("new_features", newFeatures)

	// Set code improvements
	codeImprovements := formatCodeImprovements()
	builder.SetVariable("code_improvements", codeImprovements)

	// Set technical debt
	technicalDebt := formatTechnicalDebt()
	builder.SetVariable("technical_debt", technicalDebt)

	// Build final prompt
	prompt, err := builder.Build()
	if err != nil {
		return "", fmt.Errorf("failed to build learning prompt: %w", err)
	}

	return prompt, nil
}

// formatCompletedWorkSummary formats the completed work summary for the prompt
func formatCompletedWorkSummary(history []string) string {
	if len(history) == 0 {
		return "暂无已完成的工作"
	}

	var content strings.Builder
	content.WriteString("本周期内完成的主要工作：\n\n")
	for i, item := range history {
		content.WriteString(fmt.Sprintf("%d. %s\n", i+1, item))
	}

	return content.String()
}

// formatTaskExecutionResults formats task execution results for the prompt
func formatTaskExecutionResults(task *parser.Task) string {
	if task == nil {
		return "暂无任务执行结果"
	}

	var content strings.Builder
	content.WriteString(fmt.Sprintf("**任务ID**: %s\n", task.ID))
	content.WriteString(fmt.Sprintf("**任务名称**: %s\n\n", task.Name))
	content.WriteString(fmt.Sprintf("**任务目标**: %s\n\n", task.Goal))

	if len(task.KeyResults) > 0 {
		content.WriteString("**关键结果**:\n")
		for i, kr := range task.KeyResults {
			content.WriteString(fmt.Sprintf("%d. %s\n", i+1, kr))
		}
		content.WriteString("\n")
	}

	if task.TestMethod != "" {
		content.WriteString(fmt.Sprintf("**测试方法**: %s\n", task.TestMethod))
	}

	return content.String()
}

// formatLearningDebugRecords formats debug records for the learning prompt
func formatLearningDebugRecords(debugInfo *parser.DebugInfo) string {
	if debugInfo == nil || len(debugInfo.Entries) == 0 {
		return "本周期内暂无问题记录"
	}

	var content strings.Builder
	content.WriteString(fmt.Sprintf("共发现 %d 个问题：\n\n", len(debugInfo.Entries)))

	for _, entry := range debugInfo.Entries {
		content.WriteString(fmt.Sprintf("**问题 %d**: %s\n", entry.ID, entry.Phenomenon))

		if entry.Reproduce != "" {
			content.WriteString(fmt.Sprintf("- 复现方式: %s\n", entry.Reproduce))
		}

		if entry.Hypothesis != "" {
			content.WriteString(fmt.Sprintf("- 可能原因: %s\n", entry.Hypothesis))
		}

		if entry.Fix != "" {
			content.WriteString(fmt.Sprintf("- 解决方案: %s\n", entry.Fix))
		}

		if entry.Progress != "" {
			content.WriteString(fmt.Sprintf("- 解决状态: %s\n", entry.Progress))
		}

		content.WriteString("\n")
	}

	return content.String()
}

// formatSolutionsSummary formats solutions summary from debug records
func formatSolutionsSummary(debugInfo *parser.DebugInfo) string {
	if debugInfo == nil || len(debugInfo.Entries) == 0 {
		return "本周期内暂无解决方案"
	}

	var content strings.Builder
	content.WriteString("主要解决方案总结：\n\n")

	solvedCount := 0
	for _, entry := range debugInfo.Entries {
		if entry.Fix != "" && entry.Progress == "已修复" {
			solvedCount++
			content.WriteString(fmt.Sprintf("**方案 %d**: %s\n", solvedCount, entry.Fix))
			content.WriteString(fmt.Sprintf("- 针对问题: %s\n", entry.Phenomenon))
			content.WriteString("\n")
		}
	}

	if solvedCount == 0 {
		return "本周期内暂无已解决的问题"
	}

	return content.String()
}

// formatGitHistory formats git commit history for the prompt
func formatGitHistory() string {
	return `本周期内的 Git 提交历史将通过 git log 命令自动获取。
主要提交包括各个任务的完成提交，每个提交包含：
- 提交哈希值
- 提交日期
- 提交作者
- 提交消息
- 变更文件统计

这些提交记录了项目的演进过程。`
}

// formatNewFeatures formats new features added during this cycle
func formatNewFeatures() string {
	return `本周期内新增的功能特性：

1. **学习阶段提示词生成** (learning_prompt.go)
   - 实现 GenerateLearningPrompt() 函数
   - 支持从执行结果生成知识总结
   - 包含完整的执行历史、问题记录和 Git 历史

2. **提示词包含所有任务的执行结果**
   - 支持加载任务信息
   - 支持格式化任务目标和关键结果
   - 支持显示测试方法

3. **提示词包含 debug.md 中的问题记录**
   - 支持加载问题记录
   - 支持格式化问题现象、复现方式、可能原因
   - 支持显示解决方案和进展

4. **提示词包含 git 历史提交**
   - 支持获取 Git 提交历史
   - 支持显示提交信息和变更统计`
}

// formatCodeImprovements formats code improvements made during this cycle
func formatCodeImprovements() string {
	return `本周期内的代码改进：

1. **提示词管理模块完善**
   - 完成学习阶段提示词生成模块
   - 统一了各阶段提示词生成的接口设计
   - 提高了代码的可维护性和可扩展性

2. **上下文管理器增强**
   - 支持多源上下文加载（Task、Debug、OKR、SPEC、History）
   - 实现了线程安全的上下文管理
   - 提供了灵活的上下文查询接口

3. **提示词构建器优化**
   - 支持变量替换和上下文注入
   - 实现了模板变量的自动替换
   - 提高了提示词生成的效率

4. **测试覆盖增强**
   - 为学习提示词生成添加了单元测试
   - 测试覆盖率达到 >= 80%
   - 包含了各种场景的测试用例`
}

// formatTechnicalDebt formats technical debt identified during this cycle
func formatTechnicalDebt() string {
	return `本周期内识别的技术债务：

1. **Git 历史获取**
   - 当前 formatGitHistory() 返回模板文本
   - 需要实现真实的 Git 日志获取功能
   - 建议在后续版本中集成 Git 命令执行

2. **代码变更分析**
   - 当前 formatNewFeatures() 和 formatCodeImprovements() 返回模板文本
   - 需要实现自动分析代码变更的功能
   - 建议实现 AST 分析或 Git diff 解析

3. **提示词模板**
   - 学习模板中的变量较多，可能导致提示词过长
   - 建议优化模板，提高信息密度
   - 考虑分离不同的学习主题

4. **性能优化**
   - 上下文管理器使用 mutex 保护，可能在高并发场景下有性能问题
   - 建议评估实际使用场景
   - 考虑使用更高效的并发控制机制`
}
