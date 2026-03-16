package prompt

import (
	"testing"
)

func TestLoadEmbeddedTemplate(t *testing.T) {
	// Create manager with empty template dir (should use embedded)
	pm := NewPromptManager("")

	tests := []string{"plan", "doing", "learning", "test_python"}

	for _, name := range tests {
		t.Run(name, func(t *testing.T) {
			template, err := pm.LoadTemplate(name)
			if err != nil {
				t.Fatalf("LoadTemplate(%s) failed: %v", name, err)
			}

			if template.Name != name {
				t.Errorf("Expected name %s, got %s", name, template.Name)
			}

			if template.Content == "" {
				t.Error("Template content is empty")
			}

			if len(template.Variables) == 0 {
				t.Log("Warning: No variables found in template")
			}

			t.Logf("Template %s loaded successfully, %d variables found", name, len(template.Variables))
		})
	}
}

func TestLoadEmbeddedTemplateCaching(t *testing.T) {
	pm := NewPromptManager("")

	// Load template first time
	template1, err := pm.LoadTemplate("plan")
	if err != nil {
		t.Fatalf("First load failed: %v", err)
	}

	// Load template second time (should be cached)
	template2, err := pm.LoadTemplate("plan")
	if err != nil {
		t.Fatalf("Second load failed: %v", err)
	}

	// Should be the same instance (cached)
	if template1 != template2 {
		t.Error("Template not cached properly")
	}

	if pm.GetCacheSize() != 1 {
		t.Errorf("Expected cache size 1, got %d", pm.GetCacheSize())
	}
}

func TestGetEmbeddedTemplate(t *testing.T) {
	pm := NewPromptManager("")

	tests := []struct {
		name     string
		expected bool
	}{
		{"plan", true},
		{"doing", true},
		{"learning", true},
		{"test_python", true},
		{"nonexistent", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			content := pm.getEmbeddedTemplate(tt.name)
			isEmpty := content == ""

			if tt.expected && isEmpty {
				t.Errorf("Expected embedded template %s to exist", tt.name)
			}

			if !tt.expected && !isEmpty {
				t.Errorf("Expected embedded template %s to not exist", tt.name)
			}
		})
	}
}
