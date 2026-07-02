package prompt

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/sunquan/rick/internal/parser"
	"github.com/sunquan/rick/internal/workspace"
)


// GenerateDoingPrompt generates the execution phase prompt from a task
// It includes task information, test methods, and debug context
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

	builder.SetVariable("task_info_section", formatTaskInfoSection(task))
	builder.SetVariable("requirement", "")
	builder.SetVariable("grilling_section", "")
	builder.SetVariable("import_ctx_content", "")
	builder.SetVariable("session_wrap_section", "")
	builder.SetVariable("doing_loop_content", loadDoingLoopContent())

	// Set loops context from .rick/loops/
	rickDir, _ := workspace.GetRickDir()
	loopsDir := filepath.Join(rickDir, "loops")
	builder.SetVariable("loops_context", LoadLoopsContext(loopsDir))

	// Debug context
	debugContext := contextMgr.GetDebugRaw()
	if debugContext == "" {
		debugContext = formatDebugContext(contextMgr.GetDebug())
	}
	builder.SetVariable("debug_context", debugContext)

	builder.SetVariable("rick_bin_path", resolveRickBinPath())
	builder.SetVariable("check_command", "doing_check")
	jobID := contextMgr.GetJobID()
	if jobID == "" || jobID == "doing" {
		jobID = "job_N"
	}
	builder.SetVariable("job_id", jobID)

	// Build final prompt
	prompt, err := builder.Build()
	if err != nil {
		return "", fmt.Errorf("failed to build doing prompt: %w", err)
	}

	return prompt, nil
}

// GenerateDoingPromptFile generates the execution phase prompt and saves it to doingDir/prompts/.
// doingDir is the job's doing directory; when empty, falls back to a temp directory.
// rickDir is optional: when non-empty, skills from .rick/skills/ are appended to the prompt.
// Files are persistent; no cleanup needed.
func GenerateDoingPromptFile(task *parser.Task, retryCount int, contextMgr *ContextManager, manager *PromptManager, doingDir string, rickDir ...string) (string, []string, error) {
	if task == nil {
		return "", nil, fmt.Errorf("task cannot be nil")
	}

	if contextMgr == nil {
		return "", nil, fmt.Errorf("context manager cannot be nil")
	}

	if manager == nil {
		return "", nil, fmt.Errorf("prompt manager cannot be nil")
	}

	// Determine prompts directory
	promptsDir, err := resolvePromptsDir(doingDir)
	if err != nil {
		return "", nil, fmt.Errorf("failed to resolve prompts dir: %w", err)
	}

	skillFiles := []string{}

	// Load doing template
	template, err := manager.LoadTemplate("doing")
	if err != nil {
		return "", nil, fmt.Errorf("failed to load doing template: %w", err)
	}

	// Create prompt builder
	builder := NewPromptBuilder(template)

	builder.SetVariable("task_info_section", formatTaskInfoSection(task))
	builder.SetVariable("requirement", "")
	builder.SetVariable("grilling_section", "")
	builder.SetVariable("import_ctx_content", "")
	builder.SetVariable("session_wrap_section", "")

	// Set loops context from .rick/loops/
	rickDirVal, _ := workspace.GetRickDir()
	loopsDirVal := filepath.Join(rickDirVal, "loops")
	builder.SetVariable("loops_context", LoadLoopsContext(loopsDirVal))

	// Debug context
	debugContext := contextMgr.GetDebugRaw()
	if debugContext == "" {
		debugContext = formatDebugContext(contextMgr.GetDebug())
	}
	builder.SetVariable("debug_context", debugContext)

	builder.SetVariable("doing_loop_content", loadDoingLoopContent())
	builder.SetVariable("rick_bin_path", resolveRickBinPath())
	builder.SetVariable("check_command", "doing_check")
	jobIDVal := contextMgr.GetJobID()
	if jobIDVal == "" || jobIDVal == "doing" {
		jobIDVal = "job_N"
	}
	builder.SetVariable("job_id", jobIDVal)

	// Save to doing/prompts/
	promptFile := filepath.Join(promptsDir, fmt.Sprintf("%s_doing_prompt.md", task.ID))
	if err := builder.SaveToFile(promptFile); err != nil {
		return "", nil, fmt.Errorf("failed to save doing prompt: %w", err)
	}

	return promptFile, skillFiles, nil
}

// resolvePromptsDir returns doingDir/prompts/ (created), or a temp dir if doingDir is empty.
func resolvePromptsDir(doingDir string) (string, error) {
	if doingDir == "" {
		return os.MkdirTemp("", "rick-doing-prompts-*")
	}
	return EnsurePromptsDir(doingDir)
}

// readAndAppend appends text to a file and returns nil on success
func readAndAppend(filePath, text string) ([]byte, error) {
	f, err := os.OpenFile(filePath, os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	_, err = f.WriteString(text)
	return nil, err
}

// loadDoingLoopContent reads the doing_loop skill and strips its YAML frontmatter,
// returning clean content ready for inline embedding in the doing template.
func loadDoingLoopContent() string {
	raw := LoadCoreSkills([]string{"doing_loop"})
	return stripYAMLFrontmatter(raw)
}

// formatTaskInfoSection builds the ## 任务信息 block for injection into the doing template.
func formatTaskInfoSection(task *parser.Task) string {
	var sb strings.Builder
	sb.WriteString("## 任务信息\n\n")
	sb.WriteString(fmt.Sprintf("**任务 ID**: %s\n", task.ID))
	sb.WriteString(fmt.Sprintf("**任务名称**: %s\n", task.Name))
	sb.WriteString("\n### 任务目标\n")
	sb.WriteString(task.Goal + "\n")
	sb.WriteString("\n### 关键结果\n")
	sb.WriteString(formatKeyResults(task.KeyResults) + "\n")
	sb.WriteString("\n### 测试方法\n")
	sb.WriteString(task.TestMethod + "\n")
	return sb.String()
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
