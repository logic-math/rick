package prompt

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sunquan/rick/internal/parser"
)

func TestGeneratePlanPrompt_EmptyRequirement(t *testing.T) {
	_, err := GeneratePlanPrompt("", "/tmp/test_plan", "")
	if err == nil {
		t.Error("Expected error for empty requirement, got nil")
	}
}

func TestGeneratePlanPrompt_ContainsRequirement(t *testing.T) {
	prompt, err := GeneratePlanPrompt("implement user auth", "/tmp/test_plan", "")
	if err != nil {
		t.Fatalf("GeneratePlanPrompt failed: %v", err)
	}
	if !strings.Contains(prompt, "implement user auth") {
		t.Error("Expected prompt to contain user requirement")
	}
}

func TestGeneratePlanPrompt_HasLoopsContext(t *testing.T) {
	rickDir := t.TempDir() // empty loops dir → placeholder
	prompt, err := GeneratePlanPrompt("test req", "/tmp/plan", rickDir)
	if err != nil {
		t.Fatalf("GeneratePlanPrompt failed: %v", err)
	}
	if !strings.Contains(prompt, "可用的项目 Loops") {
		t.Error("Expected prompt to contain loops_context header '可用的项目 Loops'")
	}
}

func TestGeneratePlanPrompt_NoOKRSpecRFCVars(t *testing.T) {
	prompt, err := GeneratePlanPrompt("test req", "/tmp/plan", "")
	if err != nil {
		t.Fatalf("GeneratePlanPrompt failed: %v", err)
	}
	for _, banned := range []string{"okr_path", "spec_path", "rfc_paths", "rfc_dir"} {
		if strings.Contains(prompt, banned) {
			t.Errorf("Expected prompt to NOT contain %q", banned)
		}
	}
}

func TestGeneratePlanPrompt_WithLoops(t *testing.T) {
	rickDir := t.TempDir()
	loopsDir := filepath.Join(rickDir, "loops")
	if err := os.MkdirAll(loopsDir, 0755); err != nil {
		t.Fatal(err)
	}
	loopContent := "---\nname: test-loop\ntrigger: when test happens\nscope: project\n---\n# Test Loop\n"
	if err := os.WriteFile(filepath.Join(loopsDir, "loop1.md"), []byte(loopContent), 0644); err != nil {
		t.Fatal(err)
	}
	prompt, err := GeneratePlanPrompt("test req", "/tmp/plan", rickDir)
	if err != nil {
		t.Fatalf("GeneratePlanPrompt failed: %v", err)
	}
	if !strings.Contains(prompt, "test-loop") {
		t.Error("Expected prompt to contain loop name from loops dir")
	}
}

func TestGeneratePlanPrompt_NoUnreplacedVars(t *testing.T) {
	prompt, err := GeneratePlanPrompt("test requirement", "/tmp/plan", "")
	if err != nil {
		t.Fatalf("GeneratePlanPrompt failed: %v", err)
	}
	if strings.Contains(prompt, "{{") || strings.Contains(prompt, "}}") {
		t.Error("Expected all template variables to be replaced")
	}
}

// --- context_helpers tests (functions moved to context_helpers.go) ---

func TestFormatOKRContent_WithData(t *testing.T) {
	okrInfo := &parser.ContextInfo{
		Objectives: []string{"Build Rick CLI", "Improve performance"},
		KeyResults: []string{"Complete 8 modules", "Achieve 90% test coverage"},
	}
	content := formatOKRContent(okrInfo)
	if !strings.Contains(content, "Objectives") {
		t.Error("Expected content to contain 'Objectives'")
	}
	if !strings.Contains(content, "Build Rick CLI") {
		t.Error("Expected content to contain objective")
	}
	if !strings.Contains(content, "Key Results") {
		t.Error("Expected content to contain 'Key Results'")
	}
}

func TestFormatOKRContent_Empty(t *testing.T) {
	content := formatOKRContent(&parser.ContextInfo{})
	if content != "暂无项目 OKR 信息" {
		t.Errorf("Expected default message, got %s", content)
	}
}

func TestFormatOKRContent_Nil(t *testing.T) {
	content := formatOKRContent(nil)
	if content != "暂无项目 OKR 信息" {
		t.Errorf("Expected default message, got %s", content)
	}
}

func TestFormatSPECContent_WithData(t *testing.T) {
	specInfo := &parser.ContextInfo{
		Specifications: []string{"Use Go language", "Support DAG execution"},
	}
	content := formatSPECContent(specInfo)
	if !strings.Contains(content, "Use Go language") {
		t.Error("Expected content to contain specification")
	}
}

func TestFormatSPECContent_Empty(t *testing.T) {
	content := formatSPECContent(&parser.ContextInfo{})
	if content != "暂无项目 SPEC 信息" {
		t.Errorf("Expected default message, got %s", content)
	}
}

func TestFormatCompletedWork_WithHistory(t *testing.T) {
	content := formatCompletedWork([]string{"Module 1 completed", "Module 2 completed"})
	if !strings.Contains(content, "已完成的工作") || !strings.Contains(content, "Module 1 completed") {
		t.Error("Expected content to contain history items")
	}
}

func TestFormatCompletedWork_Empty(t *testing.T) {
	content := formatCompletedWork([]string{})
	if content != "这是项目的第一阶段规划" {
		t.Errorf("Expected default message, got %s", content)
	}
}

func TestGeneratePlanPrompt_HasGrillingSkillPath(t *testing.T) {
	prompt, err := GeneratePlanPrompt("test requirement", "/tmp/plan", "")
	if err != nil {
		t.Fatalf("GeneratePlanPrompt failed: %v", err)
	}
	if !strings.Contains(prompt, "skill_grilling") {
		t.Error("Expected prompt to contain skill_grilling reference")
	}
	if strings.Contains(prompt, "sense_skill_path") {
		t.Error("Expected prompt to NOT contain sense_skill_path")
	}
}

func TestGeneratePlanPrompt_NoUnreplacedGrillingVar(t *testing.T) {
	prompt, err := GeneratePlanPrompt("test requirement", "/tmp/plan", "")
	if err != nil {
		t.Fatalf("GeneratePlanPrompt failed: %v", err)
	}
	if strings.Contains(prompt, "{{grilling_skill_path}}") {
		t.Error("Expected grilling_skill_path variable to be replaced, found unreplaced {{grilling_skill_path}}")
	}
}

func TestGeneratePlanPromptFile_WritesGrillingSkillFile(t *testing.T) {
	planDir := t.TempDir()
	promptFile, _, err := GeneratePlanPromptFile("test requirement", planDir, "")
	if err != nil {
		t.Fatalf("GeneratePlanPromptFile failed: %v", err)
	}
	promptsDir := filepath.Dir(promptFile)
	grillingFile := filepath.Join(promptsDir, "skill_grilling.md")
	if _, err := os.Stat(grillingFile); os.IsNotExist(err) {
		t.Error("Expected skill_grilling.md to be written in prompts dir")
	}
	senseFile := filepath.Join(promptsDir, "skill_sense.md")
	if _, err := os.Stat(senseFile); err == nil {
		t.Error("Expected skill_sense.md to NOT be written in prompts dir")
	}
}
