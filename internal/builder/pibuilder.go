package builder

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/sunquan/rick/internal/prompt"
	"github.com/sunquan/rick/internal/workspace"
)

// PIBuilder is the pi 统一入口 (builder 三件之 pibuilder). It composes the
// command-specific sub-builders (plan/doing/easy/human-loop/ctrl/dream/learning)
// and produces two artifacts per command:
//
//	method   — command-specific method layer (rick 全局方法前缀 + 命令特定方法),
//	           injected into pi's system prompt via --append-system-prompt.
//	instance — job context / paths, injected as the user prompt file.
//
// PromptBuilder / PromptManager / template embed 保留为底层能力（internal/prompt）。
type PIBuilder struct {
	mgr *prompt.PromptManager
}

// NewPIBuilder returns a PIBuilder with the embedded template manager.
func NewPIBuilder() *PIBuilder {
	return &PIBuilder{mgr: prompt.NewPromptManager()}
}

func (p *PIBuilder) manager() *prompt.PromptManager {
	if p.mgr == nil {
		p.mgr = prompt.NewPromptManager()
	}
	return p.mgr
}

// Name implements RuntimeBuilder.
func (p *PIBuilder) Name() string { return "pi" }

// BuildAgents implements RuntimeBuilder. pi 当前不注册额外自定义 agent
// （think/research/exporter 在 task9 经 env 职责 3 落盘），返回空列表。
func (p *PIBuilder) BuildAgents(method []Method) ([]AgentDef, error) {
	return []AgentDef{}, nil
}

// BuildPrompt implements RuntimeBuilder: 按 cmd 派发到对应 Build*，返回 instance。
func (p *PIBuilder) BuildPrompt(cmd string, params map[string]string) (string, error) {
	switch cmd {
	case "plan":
		_, instance, err := p.BuildPlan(getParam(params, "requirement"), params)
		return instance, err
	case "doing":
		_, instance, err := p.BuildDoing(getParam(params, "task_id"), params)
		return instance, err
	case "easy":
		_, instance, err := p.BuildEasy(getParam(params, "requirement"), params)
		return instance, err
	case "human-loop":
		_, instance, err := p.BuildHumanLoop(getParam(params, "topic"), params)
		return instance, err
	case "ctrl":
		_, instance, err := p.BuildCtrl(params)
		return instance, err
	case "dream":
		_, instance, err := p.BuildDream(params)
		return instance, err
	case "learning":
		_, instance, err := p.BuildLearning(params)
		return instance, err
	default:
		return "", fmt.Errorf("unknown cmd %q", cmd)
	}
}

// ---------------------------------------------------------------------------
// method layer
// ---------------------------------------------------------------------------

const rickMethodPrefix = "你是 rick 的 coding agent。rick = 方法（loops/skills/domain 体系），你 = 实现。工作前先读 domain 事实，执行中遵循 loop 流程，按需加载 skills；每个任务以「测试方法」的可验证断言为完成标准。"

// commandMethod returns the method-layer system prompt for a command.
func commandMethod(cmd string) string {
	sop := ""
	switch cmd {
	case "plan":
		sop = "Plan 方法（9 步 SOP）：1 Domain 加载 → 2 Loops 上下文初始化 → 3 探索项目 → 4 grilling 追问 → 5 方案设计 → 6 任务分解 → 7 六维评审（6 subagent 串行）→ 8 plan_check → 9 输出 task.md。"
	case "doing":
		sop = "Doing 方法：按 doing_loop 执行（Step 0 domain 搜索 + loop 匹配 → Step 1 确认全局目标 → Step 2 读上下文压缩 → Step 3 sub agent ANALYZE/RED/GREEN/DEBUG/REFACTOR/COMMIT → Step 4 产出评估 → Step 5 停止标准）。"
	case "easy":
		sop = "Easy 方法：grilling 追问澄清需求 → 按 doing_loop 执行 → 格式检查（easy_check）→ 等待人类指令后执行 learning loop。"
	case "human-loop":
		sop = "SENSE 方法（5 阶段）：S 问题确认 → E 视角生成 → N1 矛盾生成 → N2 主要矛盾判断 → S-R 辩证逆转 → EC 良知批判（human 自判）；允许反向回流。"
	case "ctrl":
		sop = "Ctrl 方法：监控 doing 进度，读取 tasks.json 与 raw_session_coding.log 汇报状态，接受人类干预指令（展示计划后征得确认再写文件）。"
	case "dream":
		sop = "Dream 方法（9 步）：确认范围 → 加载行为轨迹 → 跨 job 模式识别 → loops 进化 → domain 进化 → skills 进化 → 4 subagent 质量验证 → dream_check → 写 dream log。"
	case "learning":
		sop = "Learning 方法：读取 job 执行轨迹与 debug 记录，沉淀可复用 loops/skills 与 domain 事实，写 SUMMARY.md，跑 learning_check 后完成 draft 同步。"
	default:
		sop = "遵循 rick 方法体系（loops/skills/domain）执行当前命令。"
	}
	return rickMethodPrefix + "\n\n" + sop
}

// ---------------------------------------------------------------------------
// shared helpers (路径注入)
// ---------------------------------------------------------------------------

func getParam(params map[string]string, key string) string {
	if params == nil {
		return ""
	}
	return params[key]
}

func resolveRickBinPath() string {
	projectRoot, err := os.Getwd()
	if err != nil {
		return "rick"
	}
	localBin := filepath.Join(projectRoot, "bin", "rick")
	if _, err := os.Stat(localBin); err == nil {
		return localBin
	}
	return "rick"
}

func extractJobID(dirPath string) string {
	parts := strings.Split(filepath.ToSlash(dirPath), "/")
	for i := len(parts) - 1; i >= 0; i-- {
		if strings.HasPrefix(parts[i], "job_") {
			return parts[i]
		}
	}
	return "job_N"
}

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

// renderDoingLoop reads the doing_loop skill via prompt.ReadEmbeddedSkill（跨包访问
// 未导出的 skillsFS），strips YAML frontmatter, and substitutes the domain/debug-skill
// paths. Missing skill returns error（非静默空内容）。
func renderDoingLoop(domainDir, debugSkillPath string) (string, error) {
	raw, err := prompt.ReadEmbeddedSkill("doing_loop")
	if err != nil {
		return "", err
	}
	content := stripYAMLFrontmatter(raw)
	content = strings.ReplaceAll(content, "{{domain_dir}}", domainDir)
	return strings.ReplaceAll(content, "{{debug_skill_path}}", debugSkillPath), nil
}

// renderInlineSkills reads embedded skills via prompt.ReadEmbeddedSkill and renders
// them as pi 可识别的结构化「skill 内联段」（单文件内聚）。Missing skill returns error.
func renderInlineSkills(skills []string) (string, error) {
	var sb strings.Builder
	for i, name := range skills {
		content, err := prompt.ReadEmbeddedSkill(name)
		if err != nil {
			return "", err
		}
		if i > 0 {
			sb.WriteString("\n\n")
		}
		sb.WriteString("### skill:")
		sb.WriteString(name)
		sb.WriteString("\n\n")
		sb.WriteString(stripYAMLFrontmatter(content))
	}
	return sb.String(), nil
}

// taskFilePath returns the plan/taskN.md path injected into task_info_section.
func taskFilePath(planDir, taskID string) string {
	if taskID == "" {
		taskID = "taskN"
	}
	if planDir == "" {
		return filepath.Join("<plan_dir>", taskID+".md")
	}
	return filepath.Join(planDir, taskID+".md")
}

// debugDirPath returns the doing/debug/ path injected into debug_context.
func debugDirPath(doingDir string) string {
	if doingDir == "" {
		return filepath.Join("<doing_dir>", "debug")
	}
	return filepath.Join(doingDir, "debug")
}

func resolvePromptsDir(doingDir string) (string, error) {
	if doingDir == "" {
		return os.MkdirTemp("", "rick-doing-prompts-*")
	}
	return prompt.EnsurePromptsDir(doingDir)
}

func buildRequirementSection(requirement string) string {
	if requirement == "" {
		return ""
	}
	return fmt.Sprintf("## 用户需求\n\n%s\n", requirement)
}

func buildGrillingSection(grillingFilePath, doingDir string) string {
	writeBack := ""
	if doingDir != "" {
		writeBack = fmt.Sprintf("\n**Grilling 结束后**，将澄清结论追加到 `%s/requirement.md`（只追加，不替换）。\n", doingDir)
	}
	return fmt.Sprintf("## 第一步：Grilling 追问（需求澄清）\n\n在正式开始工作之前，必须先执行结构化追问，将需求澄清到可落实的代码路径或具体方案。\n\n**加载并执行 skill:grilling**：`%s`%s\n", grillingFilePath, writeBack)
}

func buildSessionWrapSection(learningLoopPath string) string {
	return fmt.Sprintf(`---

## 第四步：执行 Learning Loop

⚠️ **必须等待人类明确说"执行 learning"后，才能启动 Learning Loop。禁止自动触发。**

格式检查通过后，向人类汇报完成情况并停止，等待人类指令。
人类确认后，启动子 Agent 执行 Learning Loop：

`+"`%s`", learningLoopPath)
}

func buildCtxSection(ctxPath, localRickDir string) string {
	if ctxPath == "" {
		return ""
	}
	raw := prompt.LoadCoreSkills([]string{"import_ctx"})
	if raw == "" {
		return ""
	}
	content := stripYAMLFrontmatter(raw)
	content = strings.ReplaceAll(content, "{{ctx_path}}", ctxPath)
	content = strings.ReplaceAll(content, "{{local_rick_dir}}", localRickDir)
	return content
}

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

// ---------------------------------------------------------------------------
// Build* (纯渲染：返回 method + instance，不落盘)
// ---------------------------------------------------------------------------

// BuildPlan renders the plan method + instance. 空 requirement 返回 error。
func (p *PIBuilder) BuildPlan(requirement string, params map[string]string) (method string, instance string, err error) {
	if requirement == "" {
		return "", "", fmt.Errorf("requirement cannot be empty")
	}

	rickDir := getParam(params, "rick_dir")
	jobPlanDir := getParam(params, "job_plan_dir")
	if jobPlanDir == "" {
		jobPlanDir = filepath.Join(rickDir, "jobs", "job_N", "plan")
	}

	tmpl, err := p.manager().LoadTemplate("plan")
	if err != nil {
		return "", "", fmt.Errorf("failed to load plan template: %w", err)
	}

	// 单文件内聚：关键 skill 内容内联进主产物（而非散落路径引用）。缺内联源时
	// 返回 error（非 panic、非静默产出空内容）。
	inlineSkills, err := renderInlineSkills([]string{"grilling", "tdd-zh", "testing-anti-patterns-zh"})
	if err != nil {
		return "", "", fmt.Errorf("failed to inline plan skills: %w", err)
	}

	b := prompt.NewPromptBuilder(tmpl)
	b.SetVariable("loops_context", prompt.LoadLoopsContext(filepath.Join(rickDir, "loops")))
	b.SetVariable("domain_dir", filepath.Join(rickDir, "domain"))
	b.SetVariable("user_requirement", requirement)
	b.SetVariable("job_plan_dir", jobPlanDir)
	b.SetVariable("rick_bin_path", resolveRickBinPath())
	b.SetVariable("job_id", extractJobID(jobPlanDir))
	b.SetVariable("grilling_skill_path", "<tmp>/rick-plan-prompts/skill_grilling.md")
	b.SetVariable("tdd_skill_path", "<tmp>/rick-plan-skill-tdd-zh-*.md")
	b.SetVariable("testing_anti_patterns_path", "<tmp>/rick-plan-skill-testing-anti-patterns-zh-*.md")

	instance, err = b.Build()
	if err != nil {
		return "", "", fmt.Errorf("failed to build plan prompt: %w", err)
	}

	// 内联段追加在主产物末尾，作为 pi 可识别的结构化「skill 内联段」。
	instance += "\n\n---\n\n## 内联技能（单文件内聚）\n\n" + inlineSkills
	return commandMethod("plan"), instance, nil
}

// BuildDoing renders the doing method + instance（task_info_section/debug_context
// 注入路径；orchestration_section 注入 pi workflowScript 编排）。params 约定键：
// rick_dir/plan_dir/doing_dir/job_id。taskID 可选（task_info_section 的路径片段）。
func (p *PIBuilder) BuildDoing(taskID string, params map[string]string) (method string, instance string, err error) {
	rickDir := getParam(params, "rick_dir")
	planDir := getParam(params, "plan_dir")
	doingDir := getParam(params, "doing_dir")
	jobID := getParam(params, "job_id")
	if jobID == "" {
		jobID = "job_N"
	}

	tmpl, err := p.manager().LoadTemplate("doing")
	if err != nil {
		return "", "", fmt.Errorf("failed to load doing template: %w", err)
	}

	doingLoopContent, err := renderDoingLoop(filepath.Join(rickDir, "domain"), "<doing-prompts>/skill_debug_skill.md")
	if err != nil {
		return "", "", fmt.Errorf("failed to inline doing loop: %w", err)
	}

	b := prompt.NewPromptBuilder(tmpl)
	b.SetVariable("task_info_section", taskFilePath(planDir, taskID))
	b.SetVariable("requirement", "")
	b.SetVariable("grilling_section", "")
	b.SetVariable("import_ctx_content", "")
	b.SetVariable("session_wrap_section", "")
	b.SetVariable("loops_context", prompt.LoadLoopsContext(filepath.Join(rickDir, "loops")))
	b.SetVariable("skills_context", prompt.LoadSkillsContext(filepath.Join(rickDir, "skills")))
	b.SetVariable("doing_loop_content", doingLoopContent)
	b.SetVariable("loop_step_header", "## 第一步：执行 Doing Loop")
	b.SetVariable("debug_context", debugDirPath(doingDir))
	b.SetVariable("orchestration_section", buildOrchestrationSection(doingDir, planDir))
	b.SetVariable("rick_bin_path", resolveRickBinPath())
	b.SetVariable("job_id", jobID)

	instance, err = b.Build()
	if err != nil {
		return "", "", fmt.Errorf("failed to build doing prompt: %w", err)
	}
	return commandMethod("doing"), instance, nil
}

// BuildEasy renders the easy method + instance（复用 doing 模板，debug_context 注入路径）。
func (p *PIBuilder) BuildEasy(requirement string, params map[string]string) (method string, instance string, err error) {
	if requirement == "" {
		requirement = "<requirement>"
	}
	rickDir := getParam(params, "rick_dir")
	doingDir := getParam(params, "doing_dir")
	ctxPath := getParam(params, "ctx_path")
	jobID := getParam(params, "job_id")
	if jobID == "" {
		jobID = "job_N"
	}
	promptsDir := filepath.Join(doingDir, "prompts")
	domainDir := filepath.Join(rickDir, "domain")

	tmpl, err := p.manager().LoadTemplate("doing")
	if err != nil {
		return "", "", fmt.Errorf("failed to load doing template: %w", err)
	}

	// 单文件内聚：doing_loop + grilling 关键 skill 内容内联进主产物。
	doingLoopContent, err := renderDoingLoop(domainDir, filepath.Join(promptsDir, "skill_debug_skill.md"))
	if err != nil {
		return "", "", fmt.Errorf("failed to inline doing loop: %w", err)
	}
	inlineSkills, err := renderInlineSkills([]string{"grilling"})
	if err != nil {
		return "", "", fmt.Errorf("failed to inline easy skills: %w", err)
	}

	b := prompt.NewPromptBuilder(tmpl)
	b.SetVariable("task_info_section", "")
	b.SetVariable("requirement", buildRequirementSection(requirement))
	b.SetVariable("grilling_section", buildGrillingSection(filepath.Join(promptsDir, "skill_grilling.md"), ""))
	b.SetVariable("import_ctx_content", buildCtxSection(ctxPath, rickDir))
	b.SetVariable("loops_context", prompt.LoadLoopsContext(filepath.Join(rickDir, "loops")))
	b.SetVariable("skills_context", prompt.LoadSkillsContext(filepath.Join(rickDir, "skills")))
	b.SetVariable("debug_context", debugDirPath(doingDir))
	b.SetVariable("doing_loop_content", doingLoopContent)
	b.SetVariable("loop_step_header", "## 第二步：执行 Doing Loop")
	b.SetVariable("session_wrap_section", buildSessionWrapSection(filepath.Join(promptsDir, "learning_loop.md")))
	b.SetVariable("orchestration_section", "")
	b.SetVariable("rick_bin_path", resolveRickBinPath())
	b.SetVariable("job_id", jobID)

	instance, err = b.Build()
	if err != nil {
		return "", "", fmt.Errorf("failed to build easy prompt: %w", err)
	}

	// 内联段追加在主产物末尾，作为 pi 可识别的结构化「skill 内联段」。
	instance += "\n\n---\n\n## 内联技能（单文件内聚）\n\n" + inlineSkills
	return commandMethod("easy"), instance, nil
}

// BuildHumanLoop renders the SENSE method + instance（dry-run 占位路径）。
func (p *PIBuilder) BuildHumanLoop(topic string, params map[string]string) (method string, instance string, err error) {
	rfcDir := getParam(params, "rfc_dir")
	draftDir := getParam(params, "draft_dir")
	if draftDir == "" {
		draftDir = "<draft>"
	}
	if rfcDir == "" {
		rfcDir = filepath.Join(draftDir, "rfc")
	}
	instance, err = prompt.GenerateHumanLoopPrompt(topic, rfcDir, draftDir, p.manager())
	if err != nil {
		return "", "", fmt.Errorf("failed to build human-loop prompt: %w", err)
	}
	return commandMethod("human-loop"), instance, nil
}

// BuildCtrl renders the ctrl method + instance。
func (p *PIBuilder) BuildCtrl(params map[string]string) (method string, instance string, err error) {
	jobID := getParam(params, "job_id")
	rickDir := getParam(params, "rick_dir")
	if jobID == "" {
		jobID = "job_N"
	}
	doingDir := filepath.Join(rickDir, "jobs", jobID, "doing")
	planDir := filepath.Join(rickDir, "jobs", jobID, "plan")
	tasksJSONPath := filepath.Join(doingDir, "tasks.json")

	tmpl, err := p.manager().LoadTemplate("ctrl")
	if err != nil {
		return "", "", fmt.Errorf("failed to load ctrl template: %w", err)
	}

	b := prompt.NewPromptBuilder(tmpl)
	b.SetVariable("job_id", jobID)
	b.SetVariable("doing_dir", doingDir)
	b.SetVariable("plan_dir", planDir)
	b.SetVariable("tasks_json_path", tasksJSONPath)
	b.SetVariable("tasks_json_content", readFileOrDefault(tasksJSONPath, `{"tasks": []}`))

	instance, err = b.Build()
	if err != nil {
		return "", "", fmt.Errorf("failed to build ctrl prompt: %w", err)
	}
	return commandMethod("ctrl"), instance, nil
}

// BuildDream renders the dream method + instance。params["job_ids"] 为逗号分隔列表。
func (p *PIBuilder) BuildDream(params map[string]string) (method string, instance string, err error) {
	rickDir := getParam(params, "rick_dir")
	jobIDs := splitJobIDs(getParam(params, "job_ids"))
	instance, err = prompt.GenerateDreamPrompt(jobIDs, rickDir)
	if err != nil {
		return "", "", fmt.Errorf("failed to build dream prompt: %w", err)
	}
	return commandMethod("dream"), instance, nil
}

// BuildLearning renders the learning method + instance（dry-run 占位 skill 路径）。
func (p *PIBuilder) BuildLearning(params map[string]string) (method string, instance string, err error) {
	lp := LearningParams{
		JobID:   getParam(params, "job_id"),
		RickDir: getParam(params, "rick_dir"),
	}
	if lp.JobID == "" {
		lp.JobID = "job_N"
	}
	lp.LearningDir = filepath.Join(lp.RickDir, "jobs", lp.JobID, "learning")
	lp.PromptsDir = filepath.Join(lp.LearningDir, "prompts")
	lp.DebugDir = filepath.Join(lp.RickDir, "jobs", lp.JobID, "doing", "debug")

	instance, err = p.renderLearning(lp, learningSkillPaths(lp.PromptsDir))
	if err != nil {
		return "", "", err
	}
	return commandMethod("learning"), instance, nil
}

// ---------------------------------------------------------------------------
// Save* (落盘 prompts/ 产物，生成行为与重构前一致)
// ---------------------------------------------------------------------------

// SavePlanPrompt writes plan_prompt.md + skills to jobPlanDir/prompts/.
func (p *PIBuilder) SavePlanPrompt(requirement, jobPlanDir, rickDir string) (promptFile string, method string, err error) {
	if requirement == "" {
		return "", "", fmt.Errorf("requirement cannot be empty")
	}
	promptFile, _, err = prompt.GeneratePlanPromptFile(requirement, jobPlanDir, rickDir)
	if err != nil {
		return "", "", err
	}
	return promptFile, commandMethod("plan"), nil
}

// SaveDoingPrompt writes doing_prompt.md + skill_debug_skill.md to
// doingDir/prompts/（task_info_section/debug_context 注入路径；orchestration_section
// 注入 pi workflowScript 编排）。返回单一 job 级编排提示词。
func (p *PIBuilder) SaveDoingPrompt(doingDir, planDir, rickDir, jobID string) (promptFile string, method string, skillFiles []string, err error) {
	if jobID == "" {
		jobID = "job_N"
	}

	promptsDir, err := resolvePromptsDir(doingDir)
	if err != nil {
		return "", "", nil, fmt.Errorf("failed to resolve prompts dir: %w", err)
	}

	debugSkillFile, err := prompt.WriteSkillFile(promptsDir, "skill_debug_skill.md", "debug_skill")
	if err != nil {
		return "", "", nil, err
	}
	skillFiles = []string{debugSkillFile}

	tmpl, err := p.manager().LoadTemplate("doing")
	if err != nil {
		return "", "", nil, fmt.Errorf("failed to load doing template: %w", err)
	}

	doingLoopContent, err := renderDoingLoop(filepath.Join(rickDir, "domain"), debugSkillFile)
	if err != nil {
		return "", "", nil, fmt.Errorf("failed to inline doing loop: %w", err)
	}

	b := prompt.NewPromptBuilder(tmpl)
	b.SetVariable("task_info_section", taskFilePath(planDir, ""))
	b.SetVariable("requirement", "")
	b.SetVariable("grilling_section", "")
	b.SetVariable("import_ctx_content", "")
	b.SetVariable("session_wrap_section", "")
	b.SetVariable("loops_context", prompt.LoadLoopsContext(filepath.Join(rickDir, "loops")))
	b.SetVariable("skills_context", prompt.LoadSkillsContext(filepath.Join(rickDir, "skills")))
	b.SetVariable("doing_loop_content", doingLoopContent)
	b.SetVariable("loop_step_header", "## 第一步：执行 Doing Loop")
	b.SetVariable("debug_context", debugDirPath(doingDir))
	b.SetVariable("orchestration_section", buildOrchestrationSection(doingDir, planDir))
	b.SetVariable("rick_bin_path", resolveRickBinPath())
	b.SetVariable("job_id", jobID)

	promptFile = filepath.Join(promptsDir, "doing_prompt.md")
	if err := b.SaveToFile(promptFile); err != nil {
		return "", "", nil, fmt.Errorf("failed to save doing prompt: %w", err)
	}
	return promptFile, commandMethod("doing"), skillFiles, nil
}

// SaveEasyPrompt writes easy_prompt.md + skills to doingDir/prompts/.
func (p *PIBuilder) SaveEasyPrompt(jobID, requirement, rickDir, ctxPath string) (promptFile string, method string, skillFiles []string, err error) {
	doingDir := filepath.Join(rickDir, "jobs", jobID, "doing")
	promptsDir, err := prompt.EnsurePromptsDir(doingDir)
	if err != nil {
		return "", "", nil, fmt.Errorf("failed to create prompts dir: %w", err)
	}

	grillingFile, err := prompt.WriteSkillFile(promptsDir, "skill_grilling.md", "grilling")
	if err != nil {
		return "", "", nil, err
	}
	genSkillFile, err := prompt.WriteSkillFile(promptsDir, "skill_gen_skill.md", "gen-skill")
	if err != nil {
		return "", "", nil, err
	}
	genLoopFile, err := prompt.WriteSkillFile(promptsDir, "skill_gen_loop.md", "gen-loop")
	if err != nil {
		return "", "", nil, err
	}
	learningDir := filepath.Join(rickDir, "jobs", jobID, "learning")
	loopsDir := filepath.Join(rickDir, "loops")
	skillsDir := filepath.Join(rickDir, "skills")
	learningLoopFile, err := prompt.WriteSkillFileWithVars(promptsDir, "learning_loop.md", "learning_loop", map[string]string{
		"job_id":         jobID,
		"learning_dir":   learningDir,
		"loops_dir":      loopsDir,
		"skills_dir":     skillsDir,
		"rick_bin_path":  resolveRickBinPath(),
		"gen_skill_path": genSkillFile,
		"gen_loop_path":  genLoopFile,
	})
	if err != nil {
		return "", "", nil, err
	}
	debugSkillFile, err := prompt.WriteSkillFile(promptsDir, "skill_debug_skill.md", "debug_skill")
	if err != nil {
		return "", "", nil, err
	}
	skillFiles = []string{grillingFile, genSkillFile, genLoopFile, learningLoopFile, debugSkillFile}

	domainDir := filepath.Join(rickDir, "domain")
	tmpl, err := p.manager().LoadTemplate("doing")
	if err != nil {
		return "", "", nil, fmt.Errorf("failed to load doing template: %w", err)
	}

	doingLoopContent, err := renderDoingLoop(domainDir, debugSkillFile)
	if err != nil {
		return "", "", nil, fmt.Errorf("failed to inline doing loop: %w", err)
	}

	b := prompt.NewPromptBuilder(tmpl)
	b.SetVariable("task_info_section", "")
	b.SetVariable("requirement", buildRequirementSection(requirement))
	b.SetVariable("grilling_section", buildGrillingSection(grillingFile, doingDir))
	b.SetVariable("import_ctx_content", buildCtxSection(ctxPath, rickDir))
	b.SetVariable("loops_context", prompt.LoadLoopsContext(loopsDir))
	b.SetVariable("skills_context", prompt.LoadSkillsContext(skillsDir))
	b.SetVariable("debug_context", debugDirPath(doingDir))
	b.SetVariable("doing_loop_content", doingLoopContent)
	b.SetVariable("loop_step_header", "## 第二步：执行 Doing Loop")
	b.SetVariable("session_wrap_section", buildSessionWrapSection(learningLoopFile))
	b.SetVariable("orchestration_section", "")
	b.SetVariable("rick_bin_path", resolveRickBinPath())
	b.SetVariable("job_id", jobID)

	promptFile = filepath.Join(promptsDir, "easy_prompt.md")
	if err := b.SaveToFile(promptFile); err != nil {
		return "", "", nil, fmt.Errorf("failed to save easy prompt: %w", err)
	}
	return promptFile, commandMethod("easy"), skillFiles, nil
}

// SaveHumanLoopPrompt writes sense_loop/think/research/exporter + skills to loopDir/prompts/.
func (p *PIBuilder) SaveHumanLoopPrompt(topic, rfcDir, draftDir, loopDir string) (mainFile, thinkFile, researchFile, exporterFile, method string, err error) {
	mainFile, thinkFile, researchFile, exporterFile, err = prompt.GenerateHumanLoopPromptFile(topic, rfcDir, draftDir, loopDir, p.manager())
	if err != nil {
		return "", "", "", "", "", err
	}
	return mainFile, thinkFile, researchFile, exporterFile, commandMethod("human-loop"), nil
}

// SaveCtrlPrompt writes ctrl_prompt.md to doingDir/prompts/.
func (p *PIBuilder) SaveCtrlPrompt(jobID, rickDir string) (promptFile string, method string, err error) {
	promptFile, err = prompt.GenerateCtrlPromptFile(jobID, rickDir)
	if err != nil {
		return "", "", err
	}
	return promptFile, commandMethod("ctrl"), nil
}

// SaveDreamPrompt writes dream_prompt.md + skills to .rick/dream/prompts/.
func (p *PIBuilder) SaveDreamPrompt(jobIDs []string, rickDir string) (promptFile string, method string, err error) {
	promptFile, _, err = prompt.GenerateDreamPromptFile(jobIDs, rickDir)
	if err != nil {
		return "", "", err
	}
	return promptFile, commandMethod("dream"), nil
}

// ---------------------------------------------------------------------------
// learning
// ---------------------------------------------------------------------------

// LearningResult is one task row for the learning prompt's task results table.
type LearningResult struct {
	TaskID     string
	TaskName   string
	Status     string
	CommitHash string
	Attempts   int
}

// LearningParams carries the learning command's instance inputs.
type LearningParams struct {
	JobID        string
	RickDir      string
	LearningDir  string
	PromptsDir   string
	DebugContent string
	DebugDir     string
	TaskMDPaths  []string
	ActPathFiles []string
	TaskResults  []LearningResult
}

// SaveLearningPrompt writes learning_prompt.md + skills to promptsDir/.
func (p *PIBuilder) SaveLearningPrompt(lp LearningParams) (promptFile string, method string, err error) {
	promptsDir := lp.PromptsDir
	if promptsDir == "" {
		promptsDir, err = prompt.EnsurePromptsDir(lp.LearningDir)
		if err != nil {
			return "", "", fmt.Errorf("failed to create prompts dir: %w", err)
		}
	}

	skillPaths, err := p.writeLearningSkills(lp, promptsDir)
	if err != nil {
		return "", "", err
	}
	instance, err := p.renderLearning(lp, skillPaths)
	if err != nil {
		return "", "", err
	}

	promptFile = filepath.Join(promptsDir, "learning_prompt.md")
	if err := os.WriteFile(promptFile, []byte(instance), 0644); err != nil {
		return "", "", fmt.Errorf("failed to save learning prompt: %w", err)
	}
	return promptFile, commandMethod("learning"), nil
}

// writeLearningSkills writes gen-skill/gen-loop/gen-domain + learning_loop to
// promptsDir and returns their resolved paths.
func (p *PIBuilder) writeLearningSkills(lp LearningParams, promptsDir string) (map[string]string, error) {
	loopsDir := filepath.Join(lp.RickDir, "loops")
	skillsDir := filepath.Join(lp.RickDir, "skills")
	domainDir := filepath.Join(lp.RickDir, "domain")

	genSkillFile, err := prompt.WriteSkillFile(promptsDir, "skill_gen_skill.md", "gen-skill")
	if err != nil {
		return nil, fmt.Errorf("failed to write gen-skill: %w", err)
	}
	genLoopFile, err := prompt.WriteSkillFile(promptsDir, "skill_gen_loop.md", "gen-loop")
	if err != nil {
		return nil, fmt.Errorf("failed to write gen-loop: %w", err)
	}
	genDomainFile, err := prompt.WriteSkillFile(promptsDir, "skill_gen_domain.md", "gen-domain")
	if err != nil {
		return nil, fmt.Errorf("failed to write gen-domain: %w", err)
	}
	learningLoopFile, err := prompt.WriteSkillFileWithVars(promptsDir, "learning_loop.md", "learning_loop", map[string]string{
		"job_id":          lp.JobID,
		"learning_dir":    lp.LearningDir,
		"loops_dir":       loopsDir,
		"skills_dir":      skillsDir,
		"domain_dir":      domainDir,
		"rick_bin_path":   resolveRickBinPath(),
		"gen_skill_path":  genSkillFile,
		"gen_loop_path":   genLoopFile,
		"gen_domain_path": genDomainFile,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to write learning_loop skill: %w", err)
	}
	return map[string]string{
		"gen_skill_path":     genSkillFile,
		"gen_loop_path":      genLoopFile,
		"gen_domain_path":    genDomainFile,
		"learning_loop_path": learningLoopFile,
	}, nil
}

// learningSkillPaths returns placeholder skill paths for dry-run rendering.
func learningSkillPaths(promptsDir string) map[string]string {
	return map[string]string{
		"gen_skill_path":     filepath.Join(promptsDir, "skill_gen_skill.md"),
		"gen_loop_path":      filepath.Join(promptsDir, "skill_gen_loop.md"),
		"gen_domain_path":    filepath.Join(promptsDir, "skill_gen_domain.md"),
		"learning_loop_path": filepath.Join(promptsDir, "learning_loop.md"),
	}
}

// renderLearning renders the learning template with the given skill paths.
func (p *PIBuilder) renderLearning(lp LearningParams, skillPaths map[string]string) (string, error) {
	tmpl, err := p.manager().LoadTemplate("learning")
	if err != nil {
		return "", fmt.Errorf("failed to load learning template: %w", err)
	}

	b := prompt.NewPromptBuilder(tmpl)
	b.SetVariable("job_id", lp.JobID)
	b.SetVariable("loops_context", prompt.LoadLoopsContext(filepath.Join(lp.RickDir, "loops")))
	b.SetVariable("learning_loop_path", skillPaths["learning_loop_path"])

	// debug 记录：优先注入 doing/debug/ 路径（路径注入），无路径时回退为摘要文本。
	if lp.DebugDir != "" {
		b.SetVariable("debug_content", lp.DebugDir)
	} else if lp.DebugContent != "" {
		b.SetVariable("debug_content", lp.DebugContent)
	} else {
		b.SetVariable("debug_content", "（本次 job 无 debug 记录）")
	}

	if len(lp.TaskMDPaths) > 0 {
		var sb strings.Builder
		for _, path := range lp.TaskMDPaths {
			sb.WriteString(fmt.Sprintf("  - `%s`\n", path))
		}
		b.SetVariable("task_md_files", strings.TrimRight(sb.String(), "\n"))
	} else {
		b.SetVariable("task_md_files", "  （无 task*.md 文件）")
	}

	if len(lp.ActPathFiles) > 0 {
		var sb strings.Builder
		for _, path := range lp.ActPathFiles {
			sb.WriteString(fmt.Sprintf("  - `%s`\n", path))
		}
		b.SetVariable("act_path_files", strings.TrimRight(sb.String(), "\n"))
	} else {
		b.SetVariable("act_path_files", "  （无 act-path.md 文件）")
	}

	b.SetVariable("task_execution_results", formatTaskResults(lp.TaskResults))
	b.SetVariable("rick_bin_path", resolveRickBinPath())

	draftDir, err := workspace.GetDraftDir()
	if err != nil {
		draftDir = ""
	}
	b.SetVariable("draft_dir", draftDir)

	instance, err := b.Build()
	if err != nil {
		return "", fmt.Errorf("failed to build learning prompt: %w", err)
	}
	return instance, nil
}

func formatTaskResults(results []LearningResult) string {
	if len(results) == 0 {
		return "无任务元信息\n"
	}
	var sb strings.Builder
	sb.WriteString("| Task ID | 任务名称 | 状态 | Commit Hash | 重试次数 |\n")
	sb.WriteString("|---------|---------|------|-------------|----------|\n")
	for _, r := range results {
		hash := r.CommitHash
		if hash == "" {
			hash = "N/A"
		} else if len(hash) > 8 {
			hash = hash[:8]
		}
		sb.WriteString(fmt.Sprintf("| %s | %s | %s | %s | %d |\n", r.TaskID, r.TaskName, r.Status, hash, r.Attempts))
	}
	return sb.String()
}

func splitJobIDs(s string) []string {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if t := strings.TrimSpace(p); t != "" {
			out = append(out, t)
		}
	}
	return out
}
