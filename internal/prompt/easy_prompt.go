package prompt

import (
	"fmt"
	"os"
	"path/filepath"
)

// GenerateEasyPromptFile generates the easy mode interactive prompt and learning prompt.
// Both files are persisted to doingDir (not tmp) so they survive session exits.
// ctxPath is optional; when non-empty the prompt includes ctx-inheritance instructions.
// Returns mainFile (easy_prompt.md), learningFile (learning_prompt.md), skill tmp files, error.
func GenerateEasyPromptFile(jobID, requirement, rickDir, ctxPath string) (string, string, []string, error) {
	doingDir := filepath.Join(rickDir, "jobs", jobID, "doing")

	// Write skill files to doing/prompts/ (persistent)
	promptsDir, err := EnsurePromptsDir(doingDir)
	if err != nil {
		return "", "", nil, fmt.Errorf("failed to create prompts dir: %w", err)
	}
	tddFile, err := WriteSkillFile(promptsDir, "skill_tdd_zh.md", "tdd-zh")
	if err != nil {
		return "", "", nil, err
	}
	debugSkillFile, err := WriteSkillFile(promptsDir, "skill_debug_skill.md", "debug_skill")
	if err != nil {
		return "", "", nil, err
	}
	senseFile, err := WriteSkillFile(promptsDir, "skill_sense.md", "sense")
	if err != nil {
		return "", "", nil, err
	}
	skillFiles := []string{tddFile, debugSkillFile, senseFile}

	// Load context (embedded in main prompt, read latest at session start)
	okrContent := readFileOrDefault(filepath.Join(rickDir, "OKR.md"), "暂无 OKR")
	specContent := readFileOrDefault(filepath.Join(rickDir, "SPEC.md"), "暂无 SPEC")
	debugContent := readFileOrDefault(filepath.Join(doingDir, "debug.md"), "暂无（首次会话）")

	// Build main easy prompt
	mgr := NewPromptManager()
	tmpl, err := mgr.LoadTemplate("easy")
	if err != nil {
		return "", "", nil, fmt.Errorf("failed to load easy template: %w", err)
	}

	projectRoot, _ := os.Getwd()
	rickBinPath := filepath.Join(projectRoot, "bin", "rick")
	learningPromptPath := filepath.Join(promptsDir, "easy_learning_prompt.md")

	builder := NewPromptBuilder(tmpl)
	builder.SetVariable("okr_content", okrContent)
	builder.SetVariable("spec_content", specContent)
	builder.SetVariable("debug_content", debugContent)
	builder.SetVariable("requirement", requirement)
	builder.SetVariable("doing_dir", doingDir)
	builder.SetVariable("tdd_skill_path", tddFile)
	builder.SetVariable("debug_skill_path", debugSkillFile)
	builder.SetVariable("sense_skill_path", senseFile)
	builder.SetVariable("learning_prompt_path", learningPromptPath)
	builder.SetVariable("rick_bin_path", rickBinPath)
	builder.SetVariable("job_id", jobID)
	builder.SetVariable("ctx_section", buildCtxSection(ctxPath, rickDir))

	mainContent, err := builder.Build()
	if err != nil {
		return "", "", nil, fmt.Errorf("failed to build easy prompt: %w", err)
	}

	// Persist main prompt to doingDir/prompts/easy_prompt.md
	mainFile := filepath.Join(promptsDir, "easy_prompt.md")
	if err := os.WriteFile(mainFile, []byte(mainContent), 0644); err != nil {
		return "", "", nil, fmt.Errorf("failed to write easy prompt: %w", err)
	}

	// Generate and persist learning prompt
	learningContent := buildEasyLearningPrompt(jobID, rickDir, doingDir, rickBinPath)
	if err := os.WriteFile(learningPromptPath, []byte(learningContent), 0644); err != nil {
		return "", "", nil, fmt.Errorf("failed to write learning prompt: %w", err)
	}

	return mainFile, learningPromptPath, skillFiles, nil
}

// buildEasyLearningPrompt creates the learning prompt content for easy mode.
// Uses file path references so it reads fresh data when executed.
// Writes directly to .rick/wiki/, .rick/tools/, .rick/SPEC.md — no merge step.
func buildEasyLearningPrompt(jobID, rickDir, doingDir, rickBinPath string) string {
	learningDir := filepath.Join(filepath.Dir(doingDir), "learning")
	debugPath := filepath.Join(doingDir, "debug.md")
	specPath := filepath.Join(rickDir, "SPEC.md")
	okrPath := filepath.Join(rickDir, "OKR.md")
	wikiDir := filepath.Join(rickDir, "wiki")
	toolsDir := filepath.Join(rickDir, "tools")
	q := "`"

	lines := []string{
		"# Rick Easy Mode Learning",
		"",
		"你是一个资深技术专家，对本次 easy 会话的执行过程进行学习和知识沉淀。",
		"",
		"## 执行上下文",
		"",
		fmt.Sprintf("**Job**: %s（easy 模式）", jobID),
		"",
		"### 数据来源（请读取以下文件）",
		"",
		fmt.Sprintf("- **debug.md（行为轨迹与问题记录）**: %s%s%s", q, debugPath, q),
		fmt.Sprintf("- **OKR**: %s%s%s", q, okrPath, q),
		fmt.Sprintf("- **SPEC.md**: %s%s%s", q, specPath, q),
		"",
		"---",
		"",
		"## ⚠️ 执行 SOP",
		"",
		"### Step 1：读取并分析 debug.md",
		"",
		"读取 debug.md 文件，分析：",
		"- 每个 debug 条目的根因与解决方案",
		"- 跨问题的共性模式",
		"- 未解决的问题",
		"",
		"### Step 2：提取可复用 Tools",
		"",
		"**YOU MUST declare: \"I will use skill:gen-skill.\" Before writing any tool.**",
		"",
		"从 debug.md 中识别可复用模式，提取为 Python 工具：",
		"- ✅ 纯函数，确定性输入输出",
		"- ✅ 跨场景通用",
		"- ✅ 支持 --test 自验证",
		"",
		fmt.Sprintf("直接写入：%s%s/*.py%s", q, toolsDir, q),
		"",
		"### Step 3：沉淀 Skills（wiki 文档）",
		"",
		"为每个可复用模式生成 wiki 文档（触发场景/预期效果/使用方法）。",
		"",
		fmt.Sprintf("直接写入：%s%s/*.md%s", q, wikiDir, q),
		"",
		"### Step 4：更新 SPEC.md",
		"",
		fmt.Sprintf("直接更新 %s%s%s（in-place），将新 wiki 文档注册到技能列表，SPEC ≤ 512 行。", q, specPath, q),
		"",
		"### Step 5：生成 SUMMARY.md",
		"",
		fmt.Sprintf("在 %s%s%s 下生成 SUMMARY.md：", q, learningDir, q),
		"",
		"`APPROVED: true` 开头，包含执行概述、关键成就、问题教训、知识沉淀清单。",
		"",
		"### Step 6：运行 learning_check",
		"",
		"```bash",
		fmt.Sprintf("%s tools learning_check %s", rickBinPath, jobID),
		"```",
		"",
		"失败则修复后重新运行。",
		"",
		"---",
		"",
		"## ⚠️ 约束",
		"",
		"1. 必须先读取 debug.md 再生成 SUMMARY.md",
		"2. Step 2 必须声明使用 gen-skill",
		fmt.Sprintf("3. wiki/tools/SPEC 直接写入 .rick/：%s%s%s、%s%s%s、%s%s%s", q, wikiDir, q, q, toolsDir, q, q, specPath, q),
		fmt.Sprintf("4. SUMMARY.md 写入 learning 目录：%s%s%s", q, learningDir, q),
	}

	result := ""
	for _, l := range lines {
		result += l + "\n"
	}
	return result
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
