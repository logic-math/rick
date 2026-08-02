package prompt

import (
	"os"
	"strings"
	"testing"
)

func TestGenerateHumanLoopPromptFileInjectsDraftDir(t *testing.T) {
	tmpDir := t.TempDir()
	draftDir := tmpDir + "/draft"
	rfcDir := draftDir + "/rfc"
	loopDir := draftDir + "/loops/loop_1"
	promptsDir := loopDir + "/prompts"

	pm := NewPromptManager()

	mainFile, thinkFile, researchFile, exporterFile, err := GenerateHumanLoopPromptFile("test topic", rfcDir, draftDir, loopDir, pm)
	if err != nil {
		t.Fatalf("GenerateHumanLoopPromptFile() error: %v", err)
	}

	for _, f := range []string{mainFile, thinkFile, researchFile, exporterFile} {
		if !strings.HasPrefix(f, promptsDir) {
			t.Errorf("file %s is not under promptsDir %s", f, promptsDir)
		}
	}

	for _, f := range []string{thinkFile, researchFile, exporterFile} {
		content, err := os.ReadFile(f)
		if err != nil {
			t.Fatalf("failed to read %s: %v", f, err)
		}
		s := string(content)
		for _, v := range []string{"{{draft_dir}}", "{{rfc_dir}}", "{{loop_dir}}"} {
			if strings.Contains(s, v) {
				t.Errorf("file %s still contains unreplaced %s", f, v)
			}
		}
		if !strings.Contains(s, draftDir) {
			t.Errorf("file %s does not contain draftDir %q", f, draftDir)
		}
	}

	senseSkillFile := promptsDir + "/skill_sense.md"
	if _, err := os.Stat(senseSkillFile); os.IsNotExist(err) {
		t.Errorf("skill_sense.md not written to prompts dir: %s", senseSkillFile)
	}
	briefsDir := loopDir + "/briefs"
	if _, err := os.Stat(briefsDir); os.IsNotExist(err) {
		t.Errorf("briefs dir not created: %s", briefsDir)
	}

	mainContent, err := os.ReadFile(mainFile)
	if err != nil {
		t.Fatalf("failed to read main file: %v", err)
	}
	mainStr := string(mainContent)
	for _, v := range []string{"{{draft_dir}}", "{{rfc_dir}}", "{{think_agent_path}}", "{{research_agent_path}}", "{{exporter_agent_path}}"} {
		if strings.Contains(mainStr, v) {
			t.Errorf("main file still contains unreplaced %s", v)
		}
	}
}

func TestGenerateHumanLoopPromptInjectsDraftDir(t *testing.T) {
	draftDir := "/tmp/test-draft"
	rfcDir := draftDir + "/rfc"
	pm := NewPromptManager()

	content, err := GenerateHumanLoopPrompt("test topic", rfcDir, draftDir, pm)
	if err != nil {
		t.Fatalf("GenerateHumanLoopPrompt() error: %v", err)
	}

	if strings.Contains(content, "{{draft_dir}}") {
		t.Error("dry-run output contains unreplaced {{draft_dir}}")
	}
	if strings.Contains(content, "{{think_agent_path}}") {
		t.Error("dry-run output contains unreplaced {{think_agent_path}}")
	}
}

func TestSenseLoopTemplateHasThreePhases(t *testing.T) {
	pm := NewPromptManager()
	tpl, err := pm.LoadTemplate("sense_loop")
	if err != nil {
		t.Fatalf("LoadTemplate(sense_loop) error: %v", err)
	}
	for _, kw := range []string{"调研", "简报", "选项", "S1", "E1", "E2"} {
		if !strings.Contains(tpl.Content, kw) {
			t.Errorf("sense_loop.md missing keyword %q", kw)
		}
	}
}

func TestSenseLoopTemplateHasSENSESteps(t *testing.T) {
	pm := NewPromptManager()
	tpl, err := pm.LoadTemplate("sense_loop")
	if err != nil {
		t.Fatalf("LoadTemplate(sense_loop) error: %v", err)
	}
	for _, kw := range []string{"Subject", "Perspective", "Judgment", "Reverse", "Critique"} {
		if !strings.Contains(tpl.Content, kw) {
			t.Errorf("sense_loop.md missing SENSE step %q", kw)
		}
	}
}

func TestSenseLoopTemplateHasTwoProducts(t *testing.T) {
	pm := NewPromptManager()
	tpl, err := pm.LoadTemplate("sense_loop")
	if err != nil {
		t.Fatalf("LoadTemplate(sense_loop) error: %v", err)
	}
	for _, kw := range []string{"judgment.md", "briefs", "{{rfc_dir}}", "{{draft_dir}}"} {
		if !strings.Contains(tpl.Content, kw) {
			t.Errorf("sense_loop.md missing keyword %q", kw)
		}
	}
	for _, obsolete := range []string{"loops.md", "progress.md"} {
		if strings.Contains(tpl.Content, obsolete) {
			t.Errorf("sense_loop.md should not contain obsolete product %q", obsolete)
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

func TestSenseLoopTemplateHasDispatch(t *testing.T) {
	pm := NewPromptManager()
	tpl, err := pm.LoadTemplate("sense_loop")
	if err != nil {
		t.Fatalf("LoadTemplate(sense_loop) error: %v", err)
	}
	for _, kw := range []string{"{{think_agent_path}}", "{{research_agent_path}}", "{{exporter_agent_path}}", "批判门禁", "假设"} {
		if !strings.Contains(tpl.Content, kw) {
			t.Errorf("sense_loop.md missing keyword %q", kw)
		}
	}
}

func TestSenseLoopTemplateForcesHumanAtEveryStep(t *testing.T) {
	pm := NewPromptManager()
	tpl, err := pm.LoadTemplate("sense_loop")
	if err != nil {
		t.Fatalf("LoadTemplate(sense_loop) error: %v", err)
	}
	for _, kw := range []string{"不可跳过", "S1", "E1", "E2"} {
		if !strings.Contains(tpl.Content, kw) {
			t.Errorf("sense_loop.md missing enforcement keyword %q", kw)
		}
	}
}

func TestSenseLoopTemplateHasFiveDispatchElements(t *testing.T) {
	pm := NewPromptManager()
	tpl, err := pm.LoadTemplate("sense_loop")
	if err != nil {
		t.Fatalf("LoadTemplate(sense_loop) error: %v", err)
	}
	for _, kw := range []string{"主题", "草稿路径", "前序判断", "任务派发", "结果核验"} {
		if !strings.Contains(tpl.Content, kw) {
			t.Errorf("sense_loop.md missing dispatch element %q", kw)
		}
	}
}

func TestSenseLoopTemplateHasMaxRetries(t *testing.T) {
	pm := NewPromptManager()
	tpl, err := pm.LoadTemplate("sense_loop")
	if err != nil {
		t.Fatalf("LoadTemplate(sense_loop) error: %v", err)
	}
	if !strings.Contains(tpl.Content, "{{max_retries}}") {
		t.Error("sense_loop.md missing {{max_retries}} variable for configurable retry limit")
	}
}

func TestThinkTemplateHasPipeline(t *testing.T) {
	pm := NewPromptManager()
	tpl, err := pm.LoadTemplate("think")
	if err != nil {
		t.Fatalf("LoadTemplate(think) error: %v", err)
	}
	for _, kw := range []string{"打分", "最高风险"} {
		if !strings.Contains(tpl.Content, kw) {
			t.Errorf("think.md missing pipeline keyword %q", kw)
		}
	}
}

func TestThinkTemplateDeletesExplicitAssumptionDuty(t *testing.T) {
	pm := NewPromptManager()
	tpl, err := pm.LoadTemplate("think")
	if err != nil {
		t.Fatalf("LoadTemplate(think) error: %v", err)
	}
	for _, forbidden := range []string{"价值性假设生成", "显式职责"} {
		if strings.Contains(tpl.Content, forbidden) {
			t.Errorf("think.md should not contain explicit duty %q (D-R3)", forbidden)
		}
	}
}

func TestResearchTemplateHasBFS(t *testing.T) {
	pm := NewPromptManager()
	tpl, err := pm.LoadTemplate("research")
	if err != nil {
		t.Fatalf("LoadTemplate(research) error: %v", err)
	}
	for _, kw := range []string{"BFS", "事实性假设"} {
		if !strings.Contains(tpl.Content, kw) {
			t.Errorf("research.md missing keyword %q", kw)
		}
	}
}

func TestResearchTemplateHasR7Rule(t *testing.T) {
	pm := NewPromptManager()
	tpl, err := pm.LoadTemplate("research")
	if err != nil {
		t.Fatalf("LoadTemplate(research) error: %v", err)
	}
	if !strings.Contains(tpl.Content, "上报列表") {
		t.Error("research.md missing R7 rule: 上报列表 for unresearchable facts")
	}
}

func TestExporterTemplateHasTwoPhase(t *testing.T) {
	pm := NewPromptManager()
	tpl, err := pm.LoadTemplate("exporter")
	if err != nil {
		t.Fatalf("LoadTemplate(exporter) error: %v", err)
	}
	for _, kw := range []string{"大纲", "内容"} {
		if !strings.Contains(tpl.Content, kw) {
			t.Errorf("exporter.md missing two-phase keyword %q", kw)
		}
	}
}
