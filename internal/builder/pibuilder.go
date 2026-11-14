package builder

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/sunquan/rick/internal/prompt"
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
// （六 agent 在 task9 经 env 职责 3 落盘），返回空列表。
func (p *PIBuilder) BuildAgents(method []Method) ([]AgentDef, error) {
	return []AgentDef{}, nil
}

// BuildPrompt implements RuntimeBuilder: 按 cmd 派发到对应 Build*，返回 instance。
func (p *PIBuilder) BuildPrompt(cmd string, params map[string]string) (string, error) {
	switch cmd {
	case "easy":
		_, instance, err := p.BuildEasy(getParam(params, "requirement"), params)
		return instance, err
	case "human-loop":
		_, instance, err := p.BuildHumanLoop(getParam(params, "topic"), params)
		return instance, err
	case "dream":
		_, instance, err := p.BuildDream(params)
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
	case "easy":
		sop = "Easy 方法：grilling 追问澄清需求 → 按 doing_loop 执行 → 格式检查（easy_check）→ 等待人类指令后执行 learning loop。"
	case "human-loop":
		sop = "Human-loop v7 教学方法（启发 human 原创思考）：SENSE 思考范式（非流程）——问题还原（分清期望与现状；research 递归调研事实至常识性置信信息源，价值性信息上报 human 裁决）→ 构建视角 → 洞察矛盾 → 辩证逆转 → 良质批判。四 agent 重生成（research/syllabus-builder/lesson-writer/recall-review）+ 三个 parent 内联方法论 skill（zpd 档案/feynman 费曼演绎 3 步/exploration_stream 探索流）；核心循环=调研与呈现（ZPD 详略）→ 费曼演绎（human 反讲、模糊点追问、补讲、建档）→ 探索流（可选：识别原创性信息点，跨领域差异性概念扩展，形成原创性表达）；节奏推进与否问 human，档案薄弱点驱动详略；收口蒸馏 anchors.md。"
	case "dream":
		sop = "Dream 方法（9 步）：确认范围 → 加载行为轨迹 → 跨 job 模式识别 → loops 进化 → domain 进化 → skills 进化 → 4 subagent 质量验证 → dream_check → 写 dream log。"
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
	// 委托 prompt 包（单源 v4.4.13）；签名保持 (string, error) 兼容。
	return prompt.LoadDoingLoopContent(domainDir, debugSkillPath), nil
}

// renderInlineSkills reads embedded skills via prompt.ReadEmbeddedSkill and renders
// them as pi 可识别的结构化「skill 内联段」（单文件内聚）。Missing skill returns error.
func renderInlineSkills(skills []string) (string, error) {
	return renderInlineSkillsWithVars(skills, nil)
}

// renderInlineSkillsWithVars renders embedded skills with {{key}} substitution
//（v4.4.12：grilling 含 {{grilling_workdir}}，内联路径同样要替换）。
func renderInlineSkillsWithVars(skills []string, vars map[string]string) (string, error) {
	var sb strings.Builder
	for i, name := range skills {
		content, err := prompt.ReadEmbeddedSkill(name)
		if err != nil {
			return "", err
		}
		for k, v := range vars {
			content = strings.ReplaceAll(content, "{{"+k+"}}", v)
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
	return prompt.BuildRequirementSection(requirement)
}

func buildGrillingSection(grillingFilePath, doingDir string) string {
	// v4.4.12: 编排协议单源——prompt 包是唯一实现（skill:grilling 是唯一协议源），
	// pibuilder 的 dry-run 路径与 easy_prompt.go 的真实路径共用同一函数，杜绝双份漂移。
	return prompt.BuildGrillingSection(grillingFilePath, doingDir)
}

func buildSessionWrapSection(learningLoopPath string) string {
	return prompt.BuildSessionWrapSection(learningLoopPath)
}

func buildCtxSection(ctxPath, localRickDir string) string {
	return prompt.BuildCtxSection(ctxPath, localRickDir)
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
// BuildDoing renders the doing method + instance（task_info_section/debug_context
// 注入路径；orchestration_section 注入 pi workflowScript 编排）。params 约定键：
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

	tmpl, err := p.manager().LoadTemplate("easy")
	if err != nil {
		return "", "", fmt.Errorf("failed to load easy template: %w", err)
	}

	// 单文件内聚：doing_loop + grilling 关键 skill 内容内联进主产物。
	doingLoopContent, err := renderDoingLoop(domainDir, filepath.Join(promptsDir, "skill_debug_skill.md"))
	if err != nil {
		return "", "", fmt.Errorf("failed to inline doing loop: %w", err)
	}
	inlineSkills, err := renderInlineSkillsWithVars([]string{"grilling"}, map[string]string{
		"grilling_workdir": filepath.Join(doingDir, "grilling"),
	})
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
// ---------------------------------------------------------------------------
// Save* (落盘 prompts/ 产物，生成行为与重构前一致)
// ---------------------------------------------------------------------------
// SaveDoingPrompt writes doing_prompt.md + skill_debug_skill.md to
// doingDir/prompts/（task_info_section/debug_context 注入路径；orchestration_section
// SaveEasyPrompt writes easy_prompt.md + skills to doingDir/prompts/.
func (p *PIBuilder) SaveEasyPrompt(jobID, requirement, rickDir, ctxPath string) (promptFile string, method string, skillFiles []string, err error) {
	doingDir := filepath.Join(rickDir, "jobs", jobID, "doing")
	promptsDir, err := prompt.EnsurePromptsDir(doingDir)
	if err != nil {
		return "", "", nil, fmt.Errorf("failed to create prompts dir: %w", err)
	}

	grillingFile, err := prompt.WriteSkillFileWithVars(promptsDir, "skill_grilling.md", "grilling", map[string]string{
		"grilling_workdir": filepath.Join(doingDir, "grilling"),
	})
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
	tmpl, err := p.manager().LoadTemplate("easy")
	if err != nil {
		return "", "", nil, fmt.Errorf("failed to load easy template: %w", err)
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

// SaveHumanLoopPrompt writes the v7 sense_loop prompt + 7 skill files to
// loopDir/prompts/ (task prompts are composed by the parent at dispatch time;
// agent contracts live in the deployed agent system prompts).
func (p *PIBuilder) SaveHumanLoopPrompt(topic, rfcDir, draftDir, loopDir string) (mainFile, method string, err error) {
	mainFile, err = prompt.GenerateHumanLoopPromptFile(topic, rfcDir, draftDir, loopDir, p.manager())
	if err != nil {
		return "", "", err
	}
	return mainFile, commandMethod("human-loop"), nil
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
// writeLearningSkills writes gen-skill/gen-loop/gen-domain + learning_loop to
// learningSkillPaths returns placeholder skill paths for dry-run rendering.
func learningSkillPaths(promptsDir string) map[string]string {
	return map[string]string{
		"gen_skill_path":     filepath.Join(promptsDir, "skill_gen_skill.md"),
		"gen_loop_path":      filepath.Join(promptsDir, "skill_gen_loop.md"),
		"gen_domain_path":    filepath.Join(promptsDir, "skill_gen_domain.md"),
		"learning_loop_path": filepath.Join(promptsDir, "learning_loop.md"),
	}
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
