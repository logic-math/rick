package cmd

import (
	"strings"
	"testing"

	"github.com/sunquan/rick/internal/prompt"
)

// TestLearningTemplateHasDraftDir verifies the embedded learning template declares {{draft_dir}}.
func TestLearningTemplateHasDraftDir(t *testing.T) {
	pm := prompt.NewPromptManager("")
	tmpl, err := pm.LoadTemplate("learning")
	if err != nil {
		t.Fatalf("failed to load learning template: %v", err)
	}
	if !strings.Contains(tmpl.Content, "{{draft_dir}}") {
		t.Errorf("learning.md template does not contain '{{draft_dir}}'")
	}
}
