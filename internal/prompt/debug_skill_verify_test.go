package prompt

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDebugSkillFileWritten(t *testing.T) {
	rickDir := t.TempDir()
	_, skillFiles, err := GenerateEasyPromptFile("job_verify", "test req", rickDir, "")
	if err != nil {
		t.Fatalf("GenerateEasyPromptFile failed: %v", err)
	}

	promptsDir := filepath.Join(rickDir, "jobs", "job_verify", "doing", "prompts")

	t.Log("=== skillFiles returned ===")
	foundInSkillFiles := false
	for _, f := range skillFiles {
		t.Logf("  %s", filepath.Base(f))
		if filepath.Base(f) == "skill_debug_skill.md" {
			foundInSkillFiles = true
		}
	}

	t.Log("=== files in doing/prompts/ ===")
	entries, _ := os.ReadDir(promptsDir)
	for _, e := range entries {
		t.Logf("  %s", e.Name())
	}

	debugSkillPath := filepath.Join(promptsDir, "skill_debug_skill.md")
	if _, err := os.Stat(debugSkillPath); err == nil {
		data, _ := os.ReadFile(debugSkillPath)
		t.Logf("✅ skill_debug_skill.md EXISTS on disk (first 60 bytes): %s", string(data[:60]))
	} else {
		t.Error("❌ skill_debug_skill.md NOT FOUND on disk")
	}

	if !foundInSkillFiles {
		t.Error("❌ skill_debug_skill.md not in returned skillFiles list")
	}
}
