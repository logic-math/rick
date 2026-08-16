package prompt

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPromptManager_LoadTemplate(t *testing.T) {
	// Create a temporary directory for templates
	tmpDir, err := os.MkdirTemp("", "rick_prompt_test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a test template file
	templateContent := "This is a test template with {{variable1}} and {{variable2}}"
	templatePath := filepath.Join(tmpDir, "test.md")
	err = os.WriteFile(templatePath, []byte(templateContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test template: %v", err)
	}

	manager := NewPromptManager(tmpDir)

	// Test loading template
	template, err := manager.LoadTemplate("test")
	if err != nil {
		t.Fatalf("Failed to load template: %v", err)
	}

	if template.Name != "test" {
		t.Errorf("Expected template name 'test', got '%s'", template.Name)
	}

	if template.Content != templateContent {
		t.Errorf("Expected content '%s', got '%s'", templateContent, template.Content)
	}

	// Check variables extraction
	if len(template.Variables) != 2 {
		t.Errorf("Expected 2 variables, got %d", len(template.Variables))
	}

	expectedVars := map[string]bool{"variable1": true, "variable2": true}
	for _, v := range template.Variables {
		if !expectedVars[v] {
			t.Errorf("Unexpected variable: %s", v)
		}
	}
}

func TestPromptManager_CacheMechanism(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "rick_prompt_test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	templateContent := "Cached template content"
	templatePath := filepath.Join(tmpDir, "cached.md")
	err = os.WriteFile(templatePath, []byte(templateContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test template: %v", err)
	}

	manager := NewPromptManager(tmpDir)

	// Load template first time
	template1, err := manager.LoadTemplate("cached")
	if err != nil {
		t.Fatalf("Failed to load template: %v", err)
	}

	// Check cache size
	if manager.GetCacheSize() != 1 {
		t.Errorf("Expected cache size 1, got %d", manager.GetCacheSize())
	}

	// Load template second time (should come from cache)
	template2, err := manager.LoadTemplate("cached")
	if err != nil {
		t.Fatalf("Failed to load template: %v", err)
	}

	// Both should point to the same object (from cache)
	if template1 != template2 {
		t.Error("Expected same template object from cache")
	}

	// Cache size should still be 1
	if manager.GetCacheSize() != 1 {
		t.Errorf("Expected cache size 1, got %d", manager.GetCacheSize())
	}
}

func TestPromptManager_LoadNonExistentTemplate(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "rick_prompt_test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	manager := NewPromptManager(tmpDir)

	_, err = manager.LoadTemplate("nonexistent")
	if err == nil {
		t.Error("Expected error when loading non-existent template")
	}
}

func TestPromptManager_ClearCache(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "rick_prompt_test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	templateContent := "Template for cache clear test"
	templatePath := filepath.Join(tmpDir, "clear.md")
	err = os.WriteFile(templatePath, []byte(templateContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test template: %v", err)
	}

	manager := NewPromptManager(tmpDir)

	// Load template
	_, err = manager.LoadTemplate("clear")
	if err != nil {
		t.Fatalf("Failed to load template: %v", err)
	}

	if manager.GetCacheSize() != 1 {
		t.Errorf("Expected cache size 1, got %d", manager.GetCacheSize())
	}

	// Clear cache
	manager.ClearCache()

	if manager.GetCacheSize() != 0 {
		t.Errorf("Expected cache size 0 after clear, got %d", manager.GetCacheSize())
	}
}

func TestExtractVariables(t *testing.T) {
	tests := []struct {
		name      string
		content   string
		expected  []string
		wantCount int
	}{
		{
			name:      "no_variables",
			content:   "No variables here",
			expected:  []string{},
			wantCount: 0,
		},
		{
			name:      "single_variable",
			content:   "Single {{variable}} placeholder",
			expected:  []string{"variable"},
			wantCount: 1,
		},
		{
			name:      "multiple_variables",
			content:   "Multiple {{var1}} and {{var2}} placeholders",
			expected:  []string{"var1", "var2"},
			wantCount: 2,
		},
		{
			name:      "duplicate_variables",
			content:   "Duplicate {{var}} and {{var}} should be deduplicated",
			expected:  []string{"var"},
			wantCount: 1,
		},
		{
			name:      "spaced_variables",
			content:   "Variables with spaces {{ spaced }} should work",
			expected:  []string{"spaced"},
			wantCount: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			vars := extractVariables(tt.content)
			if len(vars) != tt.wantCount {
				t.Errorf("Expected %d variables, got %d: %v", tt.wantCount, len(vars), vars)
			}
		})
	}
}

func TestCoreSkillsEmbed(t *testing.T) {
	t.Run("all_eight_skill_files_non_empty", func(t *testing.T) {
		skills := []string{"sense", "testing", "debug_skill", "gen-skill", "evolve-skills"}
		for _, name := range skills {
			result := LoadCoreSkills([]string{name})
			if result == "" {
				t.Errorf("LoadCoreSkills(%q) returned empty string", name)
			}
		}
	})

	t.Run("multi_skill_contains_separator", func(t *testing.T) {
		result := LoadCoreSkills([]string{"sense", "tdd-zh"})
		if result == "" {
			t.Fatal("LoadCoreSkills returned empty for multiple skills")
		}
		if !strings.Contains(result, "---") {
			t.Errorf("multi-skill result missing '---' separator")
		}
	})

	t.Run("nonexistent_skill_no_panic", func(t *testing.T) {
		result := LoadCoreSkills([]string{"nonexistent-skill-xyz"})
		// should not panic; result may be empty
		_ = result
	})
}

func TestLoadCoreSkills_Grilling(t *testing.T) {
	result := LoadCoreSkills([]string{"grilling"})
	if !strings.Contains(result, "Interview me relentlessly") {
		t.Errorf("grilling skill missing core instruction, got len=%d", len(result))
	}
	if len(result) <= 100 {
		t.Errorf("grilling skill content too short: len=%d", len(result))
	}
}

func TestReadEmbeddedSkill(t *testing.T) {
	t.Run("existing_skill", func(t *testing.T) {
		content, err := ReadEmbeddedSkill("grilling")
		if err != nil {
			t.Fatalf("ReadEmbeddedSkill(grilling) failed: %v", err)
		}
		if !strings.Contains(content, "Grilling") || !strings.Contains(content, "结构化追问") {
			t.Error("grilling content should contain 'Grilling' and '结构化追问'")
		}
	})

	t.Run("missing_skill_returns_error", func(t *testing.T) {
		if _, err := ReadEmbeddedSkill("nonexistent-skill-xyz"); err == nil {
			t.Fatal("expected error for missing skill")
		} else if !strings.Contains(err.Error(), "nonexistent-skill-xyz") {
			t.Errorf("error should name the missing skill, got %v", err)
		}
	})
}

func TestTrimSpace(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"hello", "hello"},
		{"  hello  ", "hello"},
		{"\thello\t", "hello"},
		{"\nhello\n", "hello"},
		{"  hello world  ", "hello world"},
		{"", ""},
		{"   ", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := trimSpace(tt.input)
			if result != tt.expected {
				t.Errorf("Expected '%s', got '%s'", tt.expected, result)
			}
		})
	}
}
