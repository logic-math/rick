package prompt

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGenerateEasyPromptFile_GrillingSkillInjected(t *testing.T) {
	rickDir := t.TempDir()
	jobID := "job_test_easy_grilling"

	mainFile, skillFiles, err := GenerateEasyPromptFile(jobID, "test requirement", rickDir, "")
	if err != nil {
		t.Fatalf("GenerateEasyPromptFile failed: %v", err)
	}

	content, err := os.ReadFile(mainFile)
	if err != nil {
		t.Fatalf("failed to read main file: %v", err)
	}
	prompt := string(content)

	if !strings.Contains(prompt, "skill_grilling") {
		t.Error("Expected prompt to contain skill_grilling reference")
	}

	if strings.Contains(prompt, "{{grilling_skill_path}}") {
		t.Error("Expected grilling_skill_path variable to be replaced, found unreplaced {{grilling_skill_path}}")
	}

	doingDir := filepath.Join(rickDir, "jobs", jobID, "doing")
	promptsDir := filepath.Join(doingDir, "prompts")
	grillingFile := filepath.Join(promptsDir, "skill_grilling.md")
	if _, err := os.Stat(grillingFile); os.IsNotExist(err) {
		t.Error("Expected skill_grilling.md to be written in prompts dir")
	}

	found := false
	for _, f := range skillFiles {
		if strings.Contains(f, "skill_grilling") {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected skillFiles to include skill_grilling.md path")
	}
}

func TestGenerateEasyPrompt_DryRun(t *testing.T) {
	rickDir := t.TempDir()

	content, err := GenerateEasyPrompt("测试需求", rickDir, "")
	if err != nil {
		t.Fatalf("GenerateEasyPrompt failed: %v", err)
	}

	if !strings.Contains(content, "skill_grilling.md") {
		t.Error("Expected dry-run prompt to contain skill_grilling.md reference")
	}
	if strings.Contains(content, "{{grilling_skill_path}}") {
		t.Error("Expected grilling_skill_path to be replaced")
	}
	if strings.Contains(content, "{{") {
		t.Errorf("Expected no unreplaced variables, found: %s",
			content[strings.Index(content, "{{"):strings.Index(content, "}}")+2])
	}
	if !strings.Contains(content, "测试需求") {
		t.Error("Expected dry-run prompt to contain the requirement")
	}
}

func TestGenerateEasyLearningPromptFile_LearningVarsInjected(t *testing.T) {
	rickDir := t.TempDir()
	jobID := "job_test_learning_vars"

	// create dirs that the function needs
	doingDir := filepath.Join(rickDir, "jobs", jobID, "doing")
	if err := os.MkdirAll(doingDir, 0755); err != nil {
		t.Fatalf("failed to create doingDir: %v", err)
	}
	// create loops/ and skills/ dirs so LoadLoopsContext can run
	os.MkdirAll(filepath.Join(rickDir, "loops"), 0755)
	os.MkdirAll(filepath.Join(rickDir, "skills"), 0755)

	promptFile, err := GenerateEasyLearningPromptFile(jobID, rickDir)
	if err != nil {
		t.Fatalf("GenerateEasyLearningPromptFile failed: %v", err)
	}

	content, err := os.ReadFile(promptFile)
	if err != nil {
		t.Fatalf("failed to read prompt file: %v", err)
	}
	p := string(content)

	// old vars must not appear as unresolved literals
	for _, old := range []string{"{{wiki_dir}}", "{{tools_dir}}", "{{spec_path}}"} {
		if strings.Contains(p, old) {
			t.Errorf("prompt still contains old variable literal %s", old)
		}
	}

	// new vars must be resolved (paths present, not literal placeholders)
	if strings.Contains(p, "{{loops_dir}}") {
		t.Error("loops_dir not resolved — literal {{loops_dir}} found in output")
	}
	if strings.Contains(p, "{{skills_dir}}") {
		t.Error("skills_dir not resolved — literal {{skills_dir}} found in output")
	}
	if strings.Contains(p, "{{loops_context}}") {
		t.Error("loops_context not resolved — literal {{loops_context}} found in output")
	}

	// loops_context header must appear
	if !strings.Contains(p, "可用的项目 Loops") {
		t.Error("Expected prompt to contain '可用的项目 Loops' from loops_context injection")
	}

	// resolved paths must reference rickDir
	if !strings.Contains(p, filepath.Join(rickDir, "loops")) {
		t.Errorf("Expected prompt to contain loops_dir path %s", filepath.Join(rickDir, "loops"))
	}
	if !strings.Contains(p, filepath.Join(rickDir, "skills")) {
		t.Errorf("Expected prompt to contain skills_dir path %s", filepath.Join(rickDir, "skills"))
	}
}

// TestEasy* tests verify task7 key results: no OKR/SPEC injection, loops_context present.

func TestEasyPrompt_ContainsLoopsContext(t *testing.T) {
	rickDir := t.TempDir()
	os.MkdirAll(filepath.Join(rickDir, "loops"), 0755)

	content, err := GenerateEasyPrompt("需求", rickDir, "")
	if err != nil {
		t.Fatalf("GenerateEasyPrompt failed: %v", err)
	}
	if !strings.Contains(content, "可用的项目 Loops") {
		t.Error("easy prompt must contain '可用的项目 Loops' from loops_context")
	}
	if strings.Contains(content, "{{loops_context}}") {
		t.Error("loops_context must be resolved, found unresolved {{loops_context}}")
	}
}

func TestEasyPrompt_NoOKRorSPECInjection(t *testing.T) {
	rickDir := t.TempDir()

	content, err := GenerateEasyPrompt("需求", rickDir, "")
	if err != nil {
		t.Fatalf("GenerateEasyPrompt failed: %v", err)
	}
	for _, forbidden := range []string{"okr_content", "spec_content", "{{spec_path}}", "{{wiki_dir}}"} {
		if strings.Contains(content, forbidden) {
			t.Errorf("easy prompt must not contain %q", forbidden)
		}
	}
}

func TestEasyPrompt_DebugContentPreserved(t *testing.T) {
	rickDir := t.TempDir()

	content, err := GenerateEasyPrompt("需求", rickDir, "")
	if err != nil {
		t.Fatalf("GenerateEasyPrompt failed: %v", err)
	}
	// debug_content must be resolved (not a literal placeholder)
	if strings.Contains(content, "{{debug_content}}") {
		t.Error("debug_content must be resolved, found unresolved {{debug_content}}")
	}
	// debug section must still be present
	if !strings.Contains(content, "Debug") {
		t.Error("easy prompt must retain debug section")
	}
}

func TestGenerateEasyPromptFile_RequirementAppendInstruction(t *testing.T) {
	rickDir := t.TempDir()

	mainFile, _, err := GenerateEasyPromptFile("job_test_append", "original requirement", rickDir, "")
	if err != nil {
		t.Fatalf("GenerateEasyPromptFile failed: %v", err)
	}

	content, err := os.ReadFile(mainFile)
	if err != nil {
		t.Fatalf("failed to read main file: %v", err)
	}
	prompt := string(content)

	if !strings.Contains(prompt, "追加") {
		t.Error("Expected prompt to contain 追加 instruction for requirement.md")
	}
	if strings.Contains(prompt, "覆写") {
		t.Error("Expected prompt to NOT contain 覆写 (overwrite) instruction")
	}
}
