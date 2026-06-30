package prompt

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/sunquan/rick/internal/parser"
)

// GenerateEasyPromptFile generates the easy mode interactive prompt.
// Persisted to doingDir/prompts/ so it survives session exits.
// ctxPath is optional; when non-empty the prompt includes ctx-inheritance instructions.
// Returns mainFile (easy_prompt.md), skill tmp files, error.
func GenerateEasyPromptFile(jobID, requirement, rickDir, ctxPath string) (string, []string, error) {
	doingDir := filepath.Join(rickDir, "jobs", jobID, "doing")

	promptsDir, err := EnsurePromptsDir(doingDir)
	if err != nil {
		return "", nil, fmt.Errorf("failed to create prompts dir: %w", err)
	}
	tddFile, err := WriteSkillFile(promptsDir, "skill_tdd_zh.md", "tdd-zh")
	if err != nil {
		return "", nil, err
	}
	senseFile, err := WriteSkillFile(promptsDir, "skill_sense.md", "sense")
	if err != nil {
		return "", nil, err
	}
	debugSkillFile, err := WriteSkillFileWithVars(promptsDir, "skill_debug_skill.md", "debug_skill", map[string]string{
		"sense_skill_path": senseFile,
	})
	if err != nil {
		return "", nil, err
	}
	grillingFile, err := WriteSkillFile(promptsDir, "skill_grilling.md", "grilling")
	if err != nil {
		return "", nil, err
	}
	skillFiles := []string{tddFile, debugSkillFile, senseFile, grillingFile}

	debugContent := loadDebugContextLocal(doingDir)
	if debugContent == "" {
		debugContent = "暂无（首次会话）"
	}

	mgr := NewPromptManager()
	tmpl, err := mgr.LoadTemplate("easy")
	if err != nil {
		return "", nil, fmt.Errorf("failed to load easy template: %w", err)
	}

	projectRoot, _ := os.Getwd()
	rickBinPath := filepath.Join(projectRoot, "bin", "rick")
	learningPromptPath := filepath.Join(promptsDir, "easy_learning_prompt.md")
	loopsDir := filepath.Join(rickDir, "loops")

	builder := NewPromptBuilder(tmpl)
	builder.SetVariable("loops_context", LoadLoopsContext(loopsDir))
	builder.SetVariable("debug_content", debugContent)
	builder.SetVariable("requirement", requirement)
	builder.SetVariable("doing_dir", doingDir)
	builder.SetVariable("tdd_skill_path", tddFile)
	builder.SetVariable("debug_skill_path", debugSkillFile)
	builder.SetVariable("sense_skill_path", senseFile)
	builder.SetVariable("grilling_skill_path", grillingFile)
	builder.SetVariable("learning_prompt_path", learningPromptPath)
	builder.SetVariable("rick_bin_path", rickBinPath)
	builder.SetVariable("job_id", jobID)
	builder.SetVariable("ctx_section", buildCtxSection(ctxPath, rickDir))

	mainContent, err := builder.Build()
	if err != nil {
		return "", nil, fmt.Errorf("failed to build easy prompt: %w", err)
	}

	mainFile := filepath.Join(promptsDir, "easy_prompt.md")
	if err := os.WriteFile(mainFile, []byte(mainContent), 0644); err != nil {
		return "", nil, fmt.Errorf("failed to write easy prompt: %w", err)
	}

	return mainFile, skillFiles, nil
}

// GenerateEasyPrompt generates the easy prompt content with placeholder paths (for dry-run).
// Does not create any directories or write any files.
func GenerateEasyPrompt(requirement, rickDir, ctxPath string) (string, error) {
	if requirement == "" {
		requirement = "<requirement>"
	}

	mgr := NewPromptManager()
	tmpl, err := mgr.LoadTemplate("easy")
	if err != nil {
		return "", fmt.Errorf("failed to load easy template: %w", err)
	}

	projectRoot, _ := os.Getwd()
	rickBinPath := filepath.Join(projectRoot, "bin", "rick")
	doingDir := filepath.Join(rickDir, "jobs", "job_N", "doing")
	promptsDir := filepath.Join(doingDir, "prompts")
	loopsDir := filepath.Join(rickDir, "loops")

	builder := NewPromptBuilder(tmpl)
	builder.SetVariable("loops_context", LoadLoopsContext(loopsDir))
	builder.SetVariable("debug_content", "暂无（首次会话）")
	builder.SetVariable("requirement", requirement)
	builder.SetVariable("doing_dir", doingDir)
	builder.SetVariable("tdd_skill_path", filepath.Join(promptsDir, "skill_tdd_zh.md"))
	builder.SetVariable("debug_skill_path", filepath.Join(promptsDir, "skill_debug_skill.md"))
	builder.SetVariable("sense_skill_path", filepath.Join(promptsDir, "skill_sense.md"))
	builder.SetVariable("grilling_skill_path", filepath.Join(promptsDir, "skill_grilling.md"))
	builder.SetVariable("learning_prompt_path", filepath.Join(promptsDir, "easy_learning_prompt.md"))
	builder.SetVariable("rick_bin_path", rickBinPath)
	builder.SetVariable("job_id", "job_N")
	builder.SetVariable("ctx_section", buildCtxSection(ctxPath, rickDir))

	return builder.Build()
}

// GenerateEasyLearningPromptFile generates the learning prompt after a session ends.
// Called after the easy session completes so doingDir is fully populated (debug/, tasks.json, etc.).
// Reuses the same learning.md template as the standard learning phase.
// Returns the path to the generated prompt file.
func GenerateEasyLearningPromptFile(jobID, rickDir string) (string, error) {
	doingDir := filepath.Join(rickDir, "jobs", jobID, "doing")
	learningDir := filepath.Join(rickDir, "jobs", jobID, "learning")
	if err := os.MkdirAll(learningDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create learning dir: %w", err)
	}

	promptsDir, err := EnsurePromptsDir(doingDir)
	if err != nil {
		return "", fmt.Errorf("failed to create prompts dir: %w", err)
	}

	mgr := NewPromptManager()
	tmpl, err := mgr.LoadTemplate("learning")
	if err != nil {
		return "", fmt.Errorf("failed to load learning template: %w", err)
	}

	builder := NewPromptBuilder(tmpl)
	builder.SetVariable("job_id", jobID)
	builder.SetVariable("learning_dir", learningDir)

	okrContent := readFileOrDefault(filepath.Join(rickDir, "OKR.md"), "（本 job 无 OKR.md）")
	builder.SetVariable("okr_content", okrContent)

	loopsDir := filepath.Join(rickDir, "loops")
	skillsDir := filepath.Join(rickDir, "skills")
	builder.SetVariable("loops_dir", loopsDir)
	builder.SetVariable("skills_dir", skillsDir)
	builder.SetVariable("loops_context", LoadLoopsContext(loopsDir))

	debugContent := loadDebugContextLocal(doingDir)
	if debugContent == "" {
		debugContent = "（本次 job 无 debug 记录）"
	}
	builder.SetVariable("debug_content", debugContent)

	// easy mode has no task*.md files
	builder.SetVariable("task_md_files", "  （easy 模式无 task*.md 文件）")

	// easy mode has no per-task act-path.md
	builder.SetVariable("act_path_files", "  （easy 模式无 act-path.md 文件）")

	// tasks.json: easy session writes a single easy_session task
	builder.SetVariable("task_execution_results", buildEasyTaskResults(doingDir))

	projectRoot, _ := os.Getwd()
	builder.SetVariable("rick_bin_path", filepath.Join(projectRoot, "bin", "rick"))

	promptFile := filepath.Join(promptsDir, "easy_learning_prompt.md")
	if err := builder.SaveToFile(promptFile); err != nil {
		return "", fmt.Errorf("failed to save easy learning prompt: %w", err)
	}

	return promptFile, nil
}

// buildEasyTaskResults formats the task execution table for easy mode (single easy_session task).
func buildEasyTaskResults(doingDir string) string {
	header := "| Task ID | 任务名称 | 状态 | Commit Hash | 重试次数 |\n|---------|---------|------|-------------|----------|\n"
	data, err := os.ReadFile(filepath.Join(doingDir, "tasks.json"))
	if err != nil {
		return header + "| easy_session | Easy Mode Session | unknown | N/A | 1 |\n"
	}
	// Simple extraction: tasks.json has one task with status "success"
	status := "success"
	if strings.Contains(string(data), `"status": "failed"`) {
		status = "failed"
	}
	return header + fmt.Sprintf("| easy_session | Easy Mode Session | %s | N/A | 1 |\n", status)
}

// buildCtxSection renders the easy_ctx.md template with ctxPath and localRickDir.
// Returns empty string when ctxPath is empty (no inheritance).
func buildCtxSection(ctxPath, localRickDir string) string {
	if ctxPath == "" {
		return ""
	}
	mgr := NewPromptManager()
	tmpl, err := mgr.LoadTemplate("easy_ctx")
	if err != nil {
		return ""
	}
	builder := NewPromptBuilder(tmpl)
	builder.SetVariable("ctx_path", ctxPath)
	builder.SetVariable("local_rick_dir", localRickDir)
	content, err := builder.Build()
	if err != nil {
		return ""
	}
	return content
}

// loadDebugContextLocal mirrors executor.LoadDebugContext without creating a circular import.
// Prefers bug*.md frontmatter summaries from doingDir/debug/; falls back to doingDir/debug.md.
// Returns "" when doingDir is empty or does not exist.
//
// TODO(2026-08): remove fallback to debug.md after full migration to debug/ dir format.
func loadDebugContextLocal(doingDir string) string {
	if doingDir == "" {
		return ""
	}
	if _, err := os.Stat(doingDir); os.IsNotExist(err) {
		return ""
	}
	debugDir := filepath.Join(doingDir, "debug")
	entries, err := os.ReadDir(debugDir)
	if err == nil {
		var files []string
		for _, e := range entries {
			name := e.Name()
			if !e.IsDir() && strings.HasPrefix(name, "bug") && strings.HasSuffix(name, ".md") {
				files = append(files, name)
			}
		}
		if len(files) > 0 {
			sort.Strings(files)
			var sb strings.Builder
			for _, name := range files {
				data, err := os.ReadFile(filepath.Join(debugDir, name))
				if err != nil {
					continue
				}
				summary, status := parser.ExtractBugFrontmatter(string(data))
				sb.WriteString(fmt.Sprintf("- [%s] summary: %s | status: %s\n", name, summary, status))
			}
			if result := sb.String(); result != "" {
				return result
			}
		}
	}
	// TODO(2026-08): remove fallback after full migration
	data, err := os.ReadFile(filepath.Join(doingDir, "debug.md"))
	if err != nil {
		return ""
	}
	return string(data)
}

// readFileOrDefault reads a file and returns its content, or the default string if absent.
func readFileOrDefault(path, defaultVal string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return defaultVal
	}
	content := string(data)
	if content == "" {
		return defaultVal
	}
	return content
}
