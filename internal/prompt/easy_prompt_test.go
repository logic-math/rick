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
