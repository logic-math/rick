package prompt

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
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
	debuggingFile, err := WriteSkillFile(promptsDir, "skill_systematic_debugging_zh.md", "systematic-debugging-zh")
	if err != nil {
		return "", "", nil, err
	}
	skillFiles := []string{tddFile, debuggingFile}

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
	builder.SetVariable("systematic_debugging_path", debuggingFile)
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

// buildCtxSection returns the context-inheritance instructions for easy.md when ctxPath is set.
// Returns empty string when ctxPath is empty (no inheritance).
func buildCtxSection(ctxPath, localRickDir string) string {
	if ctxPath == "" {
		return ""
	}
	q := "`"
	var sb strings.Builder
	sb.WriteString("---\n\n")
	sb.WriteString("## 继承上下文（--ctx 模式）\n\n")
	sb.WriteString(fmt.Sprintf("本次会话继承了父级 .rick 上下文：%s%s%s\n\n", q, ctxPath, q))
	sb.WriteString("### 第一步：启动 subagent 分析并迁移上下文\n\n")
	sb.WriteString("**在开始处理用户需求之前**，先启动一个 subagent 完成上下文初始化：\n\n")
	sb.WriteString("1. 读取父级 ctx 的以下文件（按优先级）：\n")
	sb.WriteString(fmt.Sprintf("   - %s%s/OKR.md%s\n", q, ctxPath, q))
	sb.WriteString(fmt.Sprintf("   - %s%s/SPEC.md%s\n", q, ctxPath, q))
	sb.WriteString(fmt.Sprintf("   - %s%s/wiki/*.md%s（逐一读取）\n\n", q, ctxPath, q))
	sb.WriteString(fmt.Sprintf("2. 结合当前需求文档，在本地 .rick 目录（%s%s%s）生成：\n", q, localRickDir, q))
	sb.WriteString(fmt.Sprintf("   - %s%s/OKR.md%s — 以父级 OKR 为参考，结合当前需求起草本项目目标\n", q, localRickDir, q))
	sb.WriteString(fmt.Sprintf("   - %s%s/SPEC.md%s — 以父级 SPEC 为架构模板，**剔除环境强相关内容**（见下方约束），补充当前项目技术细节\n\n", q, localRickDir, q))
	sb.WriteString(fmt.Sprintf("3. 对于父级 wiki 中的文档：有通用价值的迁移到本地 %s%s/wiki/%s，项目特定的跳过\n\n", q, localRickDir, q))
	sb.WriteString("### ⚠️ 环境隔离约束（必须遵守）\n\n")
	sb.WriteString("父级 ctx 中以下内容**不得直接复制**，必须以当前环境为准重新配置：\n\n")
	sb.WriteString("| 内容类型 | 处理方式 |\n")
	sb.WriteString("|----------|----------|\n")
	sb.WriteString("| IP 地址、域名、端口 | 留空或标注 `TODO: 填写当前环境地址` |\n")
	sb.WriteString("| 密码、密钥、Token | 删除，标注 `TODO: 从当前环境获取` |\n")
	sb.WriteString("| 文件系统路径（/home/xxx、/opt/xxx） | 留空或标注 `TODO: 确认当前路径` |\n")
	sb.WriteString("| 数据库名、集群名 | 留空或标注 `TODO: 填写当前环境配置` |\n")
	sb.WriteString("| 服务账号、用户名 | 留空或标注 `TODO: 确认当前账号` |\n\n")
	sb.WriteString("**原则**：迁移的是架构知识和流程经验，不是环境配置。保持环境隔离。\n\n")
	sb.WriteString("### 知识查询规则\n\n")
	sb.WriteString("在会话过程中遇到模糊或不确定的信息时：\n")
	sb.WriteString("1. 先查当前 .rick（已迁移的上下文）\n")
	sb.WriteString(fmt.Sprintf("2. 若当前 .rick 无答案，**优先去父级 ctx 路径搜索**：%s%s%s\n", q, ctxPath, q))
	sb.WriteString("3. 找到相关知识后，将其适配当前环境后使用\n\n")
	sb.WriteString("---\n")
	return sb.String()
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
