package prompt

import (
	"os"
	"strings"
	"testing"
)

func TestGenerateHumanLoopPromptFileInjectsDraftDir(t *testing.T) {
	tmpDir := t.TempDir()
	draftDir := "/tmp/test-draft"

	pm := NewPromptManager()

	mainFile, subAgentFiles, err := GenerateHumanLoopPromptFile("test topic", tmpDir, draftDir, pm)
	if err != nil {
		t.Fatalf("GenerateHumanLoopPromptFile() error: %v", err)
	}
	defer func() {
		os.Remove(mainFile)
		for _, f := range subAgentFiles {
			os.Remove(f)
		}
	}()

	// think and express files should contain draftDir, not {{draft_dir}}
	for _, f := range subAgentFiles {
		content, err := os.ReadFile(f)
		if err != nil {
			t.Fatalf("failed to read sub-agent file %s: %v", f, err)
		}
		s := string(content)
		if strings.Contains(s, "{{draft_dir}}") {
			t.Errorf("file %s still contains unreplaced {{draft_dir}}", f)
		}
		if !strings.Contains(s, draftDir) {
			t.Errorf("file %s does not contain draftDir %q", f, draftDir)
		}
	}

	// main file should also not have {{draft_dir}}
	mainContent, err := os.ReadFile(mainFile)
	if err != nil {
		t.Fatalf("failed to read main file: %v", err)
	}
	if strings.Contains(string(mainContent), "{{draft_dir}}") {
		t.Error("main file still contains unreplaced {{draft_dir}}")
	}
}

func TestGenerateHumanLoopPromptInjectsDraftDir(t *testing.T) {
	draftDir := "/tmp/test-draft"
	pm := NewPromptManager()

	content, err := GenerateHumanLoopPrompt("test topic", "/tmp/rfc", draftDir, pm)
	if err != nil {
		t.Fatalf("GenerateHumanLoopPrompt() error: %v", err)
	}

	if strings.Contains(content, "{{draft_dir}}") {
		t.Error("dry-run output contains unreplaced {{draft_dir}}")
	}
	if !strings.Contains(content, draftDir) {
		t.Errorf("dry-run output does not contain draftDir %q", draftDir)
	}
}
