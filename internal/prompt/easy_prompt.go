package prompt

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// GenerateEasyPromptFile generates the easy mode interactive prompt using the shared doing.md template.
// ctxPath is optional; when non-empty the prompt includes ctx-inheritance instructions.
func GenerateEasyPromptFile(jobID, requirement, rickDir, ctxPath string) (string, []string, error) {
	doingDir := filepath.Join(rickDir, "jobs", jobID, "doing")

	promptsDir, err := EnsurePromptsDir(doingDir)
	if err != nil {
		return "", nil, fmt.Errorf("failed to create prompts dir: %w", err)
	}
	grillingFile, err := WriteSkillFile(promptsDir, "skill_grilling.md", "grilling")
	if err != nil {
		return "", nil, err
	}
	// Write learning loop and its dependencies upfront so the agent can trigger
	// learning at any point during the session.
	projectRoot, _ := os.Getwd()
	rickBinPath := filepath.Join(projectRoot, "bin", "rick")
	learningDir := filepath.Join(rickDir, "jobs", jobID, "learning")
	loopsDir := filepath.Join(rickDir, "loops")
	skillsDir := filepath.Join(rickDir, "skills")

	genSkillFile, err := WriteSkillFile(promptsDir, "skill_gen_skill.md", "gen-skill")
	if err != nil {
		return "", nil, err
	}
	genLoopFile, err := WriteSkillFile(promptsDir, "skill_gen_loop.md", "gen-loop")
	if err != nil {
		return "", nil, err
	}
	learningLoopFile, err := WriteSkillFileWithVars(promptsDir, "learning_loop.md", "learning_loop", map[string]string{
		"job_id":         jobID,
		"learning_dir":   learningDir,
		"loops_dir":      loopsDir,
		"skills_dir":     skillsDir,
		"rick_bin_path":  rickBinPath,
		"gen_skill_path": genSkillFile,
		"gen_loop_path":  genLoopFile,
	})
	if err != nil {
		return "", nil, err
	}

	debugSkillFile, err := WriteSkillFile(promptsDir, "skill_debug_skill.md", "debug_skill")
	if err != nil {
		return "", nil, err
	}

	skillFiles := []string{grillingFile, genSkillFile, genLoopFile, learningLoopFile, debugSkillFile}

	debugContext := loadDebugContextLocal(doingDir)
	if debugContext == "" {
		debugContext = "暂无（首次会话）"
	}

	domainDir := filepath.Join(rickDir, "domain")

	mgr := NewPromptManager()
	tmpl, err := mgr.LoadTemplate("doing")
	if err != nil {
		return "", nil, fmt.Errorf("failed to load doing template: %w", err)
	}

	builder := NewPromptBuilder(tmpl)
	builder.SetVariable("task_info_section", "")
	builder.SetVariable("requirement", buildRequirementSection(requirement))
	builder.SetVariable("grilling_section", buildGrillingSection(grillingFile, doingDir))
	builder.SetVariable("import_ctx_content", buildCtxSection(ctxPath, rickDir))
	builder.SetVariable("loops_context", LoadLoopsContext(loopsDir))
	builder.SetVariable("skills_context", LoadSkillsContext(skillsDir))
	builder.SetVariable("debug_context", debugContext)
	builder.SetVariable("doing_loop_content", loadDoingLoopContent(domainDir, debugSkillFile))
	builder.SetVariable("loop_step_header", "## 第二步：执行 Doing Loop")
	builder.SetVariable("session_wrap_section", buildSessionWrapSection(learningLoopFile))
	builder.SetVariable("orchestration_section", "")
	builder.SetVariable("rick_bin_path", rickBinPath)
	builder.SetVariable("job_id", jobID)

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
	tmpl, err := mgr.LoadTemplate("doing")
	if err != nil {
		return "", fmt.Errorf("failed to load doing template: %w", err)
	}

	projectRoot, _ := os.Getwd()
	rickBinPath := filepath.Join(projectRoot, "bin", "rick")
	promptsDir := filepath.Join(rickDir, "jobs", "job_N", "doing", "prompts")
	loopsDir := filepath.Join(rickDir, "loops")
	domainDir := filepath.Join(rickDir, "domain")

	builder := NewPromptBuilder(tmpl)
	builder.SetVariable("task_info_section", "")
	builder.SetVariable("requirement", buildRequirementSection(requirement))
	builder.SetVariable("grilling_section", buildGrillingSection(filepath.Join(promptsDir, "skill_grilling.md"), ""))
	builder.SetVariable("import_ctx_content", buildCtxSection(ctxPath, rickDir))
	builder.SetVariable("loops_context", LoadLoopsContext(loopsDir))
	builder.SetVariable("skills_context", LoadSkillsContext(filepath.Join(rickDir, "skills")))
	builder.SetVariable("debug_context", "暂无（首次会话）")
	builder.SetVariable("doing_loop_content", loadDoingLoopContent(domainDir, filepath.Join(promptsDir, "skill_debug_skill.md")))
	builder.SetVariable("loop_step_header", "## 第二步：执行 Doing Loop")
	builder.SetVariable("session_wrap_section", buildSessionWrapSection(filepath.Join(promptsDir, "learning_loop.md")))
	builder.SetVariable("orchestration_section", "")
	builder.SetVariable("rick_bin_path", rickBinPath)
	builder.SetVariable("job_id", "job_N")

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

	loopsDir := filepath.Join(rickDir, "loops")
	skillsDir := filepath.Join(rickDir, "skills")
	projectRoot, _ := os.Getwd()
	rickBinPath := filepath.Join(projectRoot, "bin", "rick")

	genSkillFile, err := WriteSkillFile(promptsDir, "skill_gen_skill.md", "gen-skill")
	if err != nil {
		return "", fmt.Errorf("failed to write gen-skill: %w", err)
	}
	genLoopFile, err := WriteSkillFile(promptsDir, "skill_gen_loop.md", "gen-loop")
	if err != nil {
		return "", fmt.Errorf("failed to write gen-loop: %w", err)
	}
	genDomainFile, err := WriteSkillFile(promptsDir, "skill_gen_domain.md", "gen-domain")
	if err != nil {
		return "", fmt.Errorf("failed to write gen-domain: %w", err)
	}

	domainDir := filepath.Join(rickDir, "domain")
	learningLoopFile, err := WriteSkillFileWithVars(promptsDir, "learning_loop.md", "learning_loop", map[string]string{
		"job_id":          jobID,
		"learning_dir":    learningDir,
		"loops_dir":       loopsDir,
		"skills_dir":      skillsDir,
		"domain_dir":      domainDir,
		"rick_bin_path":   rickBinPath,
		"gen_skill_path":  genSkillFile,
		"gen_loop_path":   genLoopFile,
		"gen_domain_path": genDomainFile,
	})
	if err != nil {
		return "", fmt.Errorf("failed to write learning_loop skill: %w", err)
	}

	builder := NewPromptBuilder(tmpl)
	builder.SetVariable("job_id", jobID)
	builder.SetVariable("loops_context", LoadLoopsContext(loopsDir))
	builder.SetVariable("learning_loop_path", learningLoopFile)

	debugContext := loadDebugContextLocal(doingDir)
	if debugContext == "" {
		debugContext = "（本次 job 无 debug 记录）"
	}
	builder.SetVariable("debug_content", debugContext)

	builder.SetVariable("task_md_files", "  （easy 模式无 task*.md 文件）")
	builder.SetVariable("act_path_files", "  （easy 模式无 act-path.md 文件）")
	builder.SetVariable("task_execution_results", buildEasyTaskResults(doingDir))
	builder.SetVariable("rick_bin_path", rickBinPath)

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

// buildRequirementSection wraps the user requirement in a prompt section.
func buildRequirementSection(requirement string) string {
	if requirement == "" {
		return ""
	}
	return fmt.Sprintf("## 用户需求\n\n%s\n", requirement)
}

// buildGrillingSection builds the grilling prompt block with the actual skill file path.
// doingDir is used for the requirement.md write-back path; pass "" in dry-run.
func buildGrillingSection(grillingFilePath, doingDir string) string {
	writeBack := ""
	if doingDir != "" {
		writeBack = fmt.Sprintf("\n**Grilling 结束后**，将澄清结论追加到 `%s/requirement.md`（只追加，不替换）。\n", doingDir)
	}
	return fmt.Sprintf("## 第一步：Grilling 追问（需求澄清）\n\n在正式开始工作之前，必须先执行结构化追问，将需求澄清到可落实的代码路径或具体方案。\n\n**加载并执行 skill:grilling**：`%s`%s\n", grillingFilePath, writeBack)
}

// buildSessionWrapSection returns the learning trigger section injected at the end of easy prompts.
func buildSessionWrapSection(learningLoopPath string) string {
	return fmt.Sprintf(`---

## 第四步：执行 Learning Loop

⚠️ **必须等待人类明确说"执行 learning"后，才能启动 Learning Loop。禁止自动触发。**

格式检查通过后，向人类汇报完成情况并停止，等待人类指令。
人类确认后，启动子 Agent 执行 Learning Loop：

`+"`%s`", learningLoopPath)
}

// buildCtxSection renders skills/import_ctx.md with ctxPath and localRickDir substituted.
// Returns empty string when ctxPath is empty (no inheritance).
func buildCtxSection(ctxPath, localRickDir string) string {
	if ctxPath == "" {
		return ""
	}
	raw := LoadCoreSkills([]string{"import_ctx"})
	if raw == "" {
		return ""
	}
	// Strip YAML frontmatter (---\n...\n---\n)
	content := stripYAMLFrontmatter(raw)
	content = strings.ReplaceAll(content, "{{ctx_path}}", ctxPath)
	content = strings.ReplaceAll(content, "{{local_rick_dir}}", localRickDir)
	return content
}

// stripYAMLFrontmatter removes the leading ---...--- YAML block if present.
func stripYAMLFrontmatter(s string) string {
	if !strings.HasPrefix(s, "---\n") {
		return s
	}
	rest := s[4:]
	idx := strings.Index(rest, "\n---\n")
	if idx < 0 {
		return s
	}
	return strings.TrimLeft(rest[idx+5:], "\n")
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
				summary, status := extractBugFrontmatter(string(data))
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

// extractBugFrontmatter parses YAML frontmatter (between --- markers) and
// extracts summary and status fields (the thin replacement for the deleted
// internal/parser.ExtractBugFrontmatter).
func extractBugFrontmatter(content string) (summary, status string) {
	lines := strings.Split(content, "\n")
	inFrontmatter := false
	started := false
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "---" {
			if !started {
				inFrontmatter = true
				started = true
				continue
			}
			if inFrontmatter {
				break
			}
		}
		if !inFrontmatter {
			continue
		}
		if strings.HasPrefix(trimmed, "summary:") {
			summary = strings.Trim(strings.TrimSpace(strings.TrimPrefix(trimmed, "summary:")), `"'`)
		} else if strings.HasPrefix(trimmed, "status:") {
			status = strings.Trim(strings.TrimSpace(strings.TrimPrefix(trimmed, "status:")), `"'`)
		}
	}
	return summary, status
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
