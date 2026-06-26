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

func TestGeneratePlanPrompt_NoSpecFile(t *testing.T) {
	rickDir := t.TempDir() // empty dir, no SPEC.md
	prompt, err := GeneratePlanPrompt("test req", "/tmp/plan", rickDir)
	if err != nil {
		t.Fatalf("GeneratePlanPrompt failed: %v", err)
	}
	if !strings.Contains(prompt, "暂无") {
		t.Error("Expected prompt to contain 暂无 when no SPEC.md exists")
	}
}

func TestGeneratePlanPrompt_WithSpecFile(t *testing.T) {
	rickDir := t.TempDir()
	specPath := filepath.Join(rickDir, "SPEC.md")
	if err := os.WriteFile(specPath, []byte("# SPEC\n## 技术栈\n- 语言: Go"), 0644); err != nil {
		t.Fatal(err)
	}
	prompt, err := GeneratePlanPrompt("test req", "/tmp/plan", rickDir)
	if err != nil {
		t.Fatalf("GeneratePlanPrompt failed: %v", err)
	}
	if !strings.Contains(prompt, specPath) {
		t.Error("Expected prompt to contain SPEC.md path")
	}
}

func TestGeneratePlanPrompt_WithRFC(t *testing.T) {
	rickDir := t.TempDir()
	rfcDir := filepath.Join(rickDir, "RFC")
	if err := os.MkdirAll(rfcDir, 0755); err != nil {
		t.Fatal(err)
	}
	rfcPath := filepath.Join(rfcDir, "rfc001.md")
	if err := os.WriteFile(rfcPath, []byte("# RFC001\nDecision: use DIP"), 0644); err != nil {
		t.Fatal(err)
	}
	prompt, err := GeneratePlanPrompt("test req", "/tmp/plan", rickDir)
	if err != nil {
		t.Fatalf("GeneratePlanPrompt failed: %v", err)
	}
	if !strings.Contains(prompt, rfcPath) {
		t.Error("Expected prompt to contain RFC file path")
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
