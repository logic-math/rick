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

	mainFile, senseFile, err := GenerateHumanLoopPromptFile("test topic", rfcDir, draftDir, loopDir, pm)
	if err != nil {
		t.Fatalf("GenerateHumanLoopPromptFile() error: %v", err)
	}

	for _, f := range []string{mainFile, senseFile} {
		if !strings.HasPrefix(f, promptsDir) {
			t.Errorf("file %s is not under promptsDir %s", f, promptsDir)
		}
	}

	content, err := os.ReadFile(senseFile)
	if err != nil {
		t.Fatalf("failed to read sense file: %v", err)
	}
	s := string(content)
	for _, v := range []string{"{{draft_dir}}", "{{rfc_dir}}", "{{sense_skill_path}}", "{{loop_dir}}"} {
		if strings.Contains(s, v) {
			t.Errorf("sense file still contains unreplaced %s", v)
		}
	}
	if !strings.Contains(s, draftDir) {
		t.Errorf("sense file does not contain draftDir %q", draftDir)
	}
	// sense skill file and briefs dir must exist
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
	for _, v := range []string{"{{draft_dir}}", "{{rfc_dir}}", "{{sense_agent_path}}"} {
		if strings.Contains(string(mainContent), v) {
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
	if strings.Contains(content, "{{sense_agent_path}}") {
		t.Error("dry-run output contains unreplaced {{sense_agent_path}}")
	}
}

func TestSenseSubagentTemplateHasThreePhases(t *testing.T) {
	pm := NewPromptManager()
	tpl, err := pm.LoadTemplate("sense_subagent")
	if err != nil {
		t.Fatalf("LoadTemplate(sense_subagent) error: %v", err)
	}
	for _, kw := range []string{"调研", "简报", "选项", "S1", "E1", "E2"} {
		if !strings.Contains(tpl.Content, kw) {
			t.Errorf("sense_subagent.md missing keyword %q", kw)
		}
	}
}

func TestSenseSubagentTemplateHasSENSESteps(t *testing.T) {
	pm := NewPromptManager()
	tpl, err := pm.LoadTemplate("sense_subagent")
	if err != nil {
		t.Fatalf("LoadTemplate(sense_subagent) error: %v", err)
	}
	for _, kw := range []string{"Subject", "Perspective", "Judgment", "Reverse", "Critique"} {
		if !strings.Contains(tpl.Content, kw) {
			t.Errorf("sense_subagent.md missing SENSE step %q", kw)
		}
	}
}

func TestSenseSubagentTemplateHasOutputFiles(t *testing.T) {
	pm := NewPromptManager()
	tpl, err := pm.LoadTemplate("sense_subagent")
	if err != nil {
		t.Fatalf("LoadTemplate(sense_subagent) error: %v", err)
	}
	for _, kw := range []string{"judgment.md", "loops.md", "progress.md", "{{rfc_dir}}", "{{draft_dir}}"} {
		if !strings.Contains(tpl.Content, kw) {
			t.Errorf("sense_subagent.md missing keyword %q", kw)
		}
	}
}

func TestSenseSubagentJudgmentIsHumanOnly(t *testing.T) {
	pm := NewPromptManager()
	tpl, err := pm.LoadTemplate("sense_subagent")
	if err != nil {
		t.Fatalf("LoadTemplate(sense_subagent) error: %v", err)
	}
	// judgment.md must emphasize human-only
	if !strings.Contains(tpl.Content, "human 原话") && !strings.Contains(tpl.Content, "human 判断") {
		t.Error("sense_subagent.md missing human-only constraint for judgment.md")
	}
	// must forbid AI reasoning in judgment.md
	if !strings.Contains(tpl.Content, "AI 推理") {
		t.Error("sense_subagent.md missing prohibition of AI reasoning in judgment.md")
	}
}

func TestSenseSubagentLoopsIsConceptMap(t *testing.T) {
	pm := NewPromptManager()
	tpl, err := pm.LoadTemplate("sense_subagent")
	if err != nil {
		t.Fatalf("LoadTemplate(sense_subagent) error: %v", err)
	}
	for _, kw := range []string{"概念地图", "掌握程度"} {
		if !strings.Contains(tpl.Content, kw) {
			t.Errorf("sense_subagent.md missing concept-map keyword %q in loops.md section", kw)
		}
	}
}

func TestHumanLoopMainTemplateHasSenseAgentPath(t *testing.T) {
	pm := NewPromptManager()
	tpl, err := pm.LoadTemplate("human_loop")
	if err != nil {
		t.Fatalf("LoadTemplate(human_loop) error: %v", err)
	}
	for _, kw := range []string{"{{sense_agent_path}}", "批判门禁", "假设"} {
		if !strings.Contains(tpl.Content, kw) {
			t.Errorf("human_loop.md missing keyword %q", kw)
		}
	}
}

func TestHumanLoopMainTemplateForcesHumanAtEveryStep(t *testing.T) {
	pm := NewPromptManager()
	tpl, err := pm.LoadTemplate("human_loop")
	if err != nil {
		t.Fatalf("LoadTemplate(human_loop) error: %v", err)
	}
	// must enforce human judgment at every step
	for _, kw := range []string{"不可跳过", "S1", "E1", "E2"} {
		if !strings.Contains(tpl.Content, kw) {
			t.Errorf("human_loop.md missing enforcement keyword %q", kw)
		}
	}
}
