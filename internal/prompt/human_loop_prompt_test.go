package prompt

import (
	"os"
	"strings"
	"testing"
)

// forbiddenTeachMethodology 列出 human 明确要求从 teach 中删除的方法论/机制关键词。
// teach 只讲 SENSE 五关注面的「概念本质」，不得写回这些内容。
var forbiddenTeachMethodology = []string{
	"三维打分",
	"系统论描述符",
	"4 维打分",
	"think_top_n",
	"反向回流",
	"max_backflows",
	"node/input/output/inner/edge",
}

func TestGenerateHumanLoopPromptFileWritesThreeFiles(t *testing.T) {
	tmpDir := t.TempDir()
	draftDir := tmpDir + "/draft"
	rfcDir := draftDir + "/rfc"
	loopDir := draftDir + "/loops/loop_1"
	promptsDir := loopDir + "/prompts"

	pm := NewPromptManager()

	mainFile, teachFile, researchFile, err := GenerateHumanLoopPromptFile("test topic", rfcDir, draftDir, loopDir, pm)
	if err != nil {
		t.Fatalf("GenerateHumanLoopPromptFile() error: %v", err)
	}

	for _, f := range []string{mainFile, teachFile, researchFile} {
		if !strings.HasPrefix(f, promptsDir) {
			t.Errorf("file %s is not under promptsDir %s", f, promptsDir)
		}
	}

	for _, f := range []string{mainFile, teachFile, researchFile} {
		content, err := os.ReadFile(f)
		if err != nil {
			t.Fatalf("failed to read %s: %v", f, err)
		}
		s := string(content)
		if strings.Contains(s, "{{") {
			t.Errorf("file %s still contains an unreplaced template variable", f)
		}
		if !strings.Contains(s, draftDir) {
			t.Errorf("file %s does not contain draftDir %q", f, draftDir)
		}
	}

	// Skills: sense + teach + research（不再有 skill_think/skill_exporter）。
	for _, skill := range []string{"skill_sense.md", "skill_teach.md", "skill_research.md"} {
		if _, err := os.Stat(promptsDir + "/" + skill); os.IsNotExist(err) {
			t.Errorf("skill file not written to prompts dir: %s", skill)
		}
	}

	// 学习材料目录预建：research/ 与 lessons/。
	for _, d := range []string{loopDir + "/research", loopDir + "/lessons"} {
		if _, err := os.Stat(d); os.IsNotExist(err) {
			t.Errorf("learning material dir not created: %s", d)
		}
	}
}

func TestGenerateHumanLoopPromptDryRunNoUnreplacedVars(t *testing.T) {
	draftDir := "/tmp/test-draft"
	rfcDir := draftDir + "/rfc"
	pm := NewPromptManager()

	content, err := GenerateHumanLoopPrompt("test topic", rfcDir, draftDir, pm)
	if err != nil {
		t.Fatalf("GenerateHumanLoopPrompt() error: %v", err)
	}

	if strings.Contains(content, "{{") {
		t.Errorf("dry-run output contains an unreplaced template variable")
	}
	// 学习材料路径约定注入。
	for _, want := range []string{"mission.md", "syllabus.md", "lessons", "research", "judgment.md"} {
		if !strings.Contains(content, want) {
			t.Errorf("dry-run output missing learning-material path %q", want)
		}
	}
	if strings.Contains(content, "exporter") {
		t.Error("dry-run output still contains exporter")
	}
}

func TestSenseLoopTemplateIsTeacherSession(t *testing.T) {
	pm := NewPromptManager()
	tpl, err := pm.LoadTemplate("sense_loop")
	if err != nil {
		t.Fatalf("LoadTemplate(sense_loop) error: %v", err)
	}

	// 新协议关键词：teach + research + 学习材料路径约定 + 文件状态机 + 退出准则。
	for _, kw := range []string{"teach", "research", "mission.md", "syllabus", "教学大纲", "lessons", "judgment.md", "文件状态机", "学习目标", "考核标准", "退出准则"} {
		if !strings.Contains(tpl.Content, kw) {
			t.Errorf("sense_loop.md missing keyword %q", kw)
		}
	}

	// 已删除的机械化机制不得回流。
	for _, gone := range []string{"exporter", "阶段 1:S", "N1:", "反向回流", "三维打分", "系统论描述符", "max_retries", "max_backflows", "think_agent_path"} {
		if strings.Contains(tpl.Content, gone) {
			t.Errorf("sense_loop.md should not contain removed mechanism %q", gone)
		}
	}

	// 派发显式语法 + 交付门禁。
	for _, kw := range []string{"workflowScript", "timeoutMs: 3600000", "runs.run"} {
		if !strings.Contains(tpl.Content, kw) {
			t.Errorf("sense_loop.md missing dispatch keyword %q", kw)
		}
	}
}

func TestSenseLoopJudgmentIsHumanOnly(t *testing.T) {
	pm := NewPromptManager()
	tpl, err := pm.LoadTemplate("sense_loop")
	if err != nil {
		t.Fatalf("LoadTemplate(sense_loop) error: %v", err)
	}
	if !strings.Contains(tpl.Content, "human 原话") && !strings.Contains(tpl.Content, "human 判断") {
		t.Error("sense_loop.md missing human-only constraint for judgment.md")
	}
	if !strings.Contains(tpl.Content, "AI 推理") {
		t.Error("sense_loop.md missing prohibition of AI reasoning in judgment.md")
	}
}

func TestTeachTemplateHasSenseAspectsAndDiscipline(t *testing.T) {
	pm := NewPromptManager()
	tpl, err := pm.LoadTemplate("teach")
	if err != nil {
		t.Fatalf("LoadTemplate(teach) error: %v", err)
	}
	skill, err := ReadEmbeddedSkill("teach")
	if err != nil {
		t.Fatalf("ReadEmbeddedSkill(teach) error: %v", err)
	}
	all := tpl.Content + "\n" + skill

	// SENSE 五关注面 + 概念本质（human 要求：只讲本质、不讲方法论）。
	for _, kw := range []string{"还原事实", "呈现视角", "洞察矛盾", "启发逆转", "良质批判", "本质"} {
		if !strings.Contains(all, kw) {
			t.Errorf("teach template/skill missing SENSE aspect %q", kw)
		}
	}

	// 教学纪律 + 产出契约。
	for _, kw := range []string{"一课一事", "lessons/", "teach_", "隐式", "primary source", "working memory", "教学大纲", "核心练习方法", "还原原创思考"} {
		if !strings.Contains(all, kw) {
			t.Errorf("teach template/skill missing teaching keyword %q", kw)
		}
	}

	// 已删方法论不得回流。
	for _, gone := range forbiddenTeachMethodology {
		if strings.Contains(all, gone) {
			t.Errorf("teach template/skill should not contain removed methodology %q", gone)
		}
	}
}

func TestResearchTemplateHasDiligenceAndGaps(t *testing.T) {
	pm := NewPromptManager()
	tpl, err := pm.LoadTemplate("research")
	if err != nil {
		t.Fatalf("LoadTemplate(research) error: %v", err)
	}
	for _, kw := range []string{"尽调树", "MECE", "下钻", "信源", "权重", "置信度", "subagent", "research_", "Gaps", "domain-survey"} {
		if !strings.Contains(tpl.Content, kw) {
			t.Errorf("research.md missing keyword %q", kw)
		}
	}
	// 信源等级 A/B/C。
	if !strings.Contains(tpl.Content, "等级") || !strings.Contains(tpl.Content, "A") {
		t.Error("research.md missing source-grade A/B/C contract")
	}
	for _, obsolete := range []string{"BFS", "全量收集", "QUEUE", "exporter"} {
		if strings.Contains(tpl.Content, obsolete) {
			t.Errorf("research.md should not contain obsolete concept %q", obsolete)
		}
	}
	skill, err := ReadEmbeddedSkill("research")
	if err != nil {
		t.Fatalf("ReadEmbeddedSkill(research) error: %v", err)
	}
	if !strings.Contains(skill, "R7 上报") {
		t.Error("research skill missing R7 上报 rule for unresearchable facts")
	}
	if !strings.Contains(skill, "Gaps") {
		t.Error("research skill missing ## Gaps contract")
	}
}
