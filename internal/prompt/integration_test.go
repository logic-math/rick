package prompt

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestIntegration_LoadAllTemplates verifies all templates can be loaded correctly
func TestIntegration_LoadAllTemplates(t *testing.T) {
	templateDir := getTemplateDir()

	pm := NewPromptManager(templateDir)

	templates := []string{"plan", "doing", "test_python", "learning"}
	for _, templateName := range templates {
		t.Run(templateName, func(t *testing.T) {
			template, err := pm.LoadTemplate(templateName)
			if err != nil {
				t.Fatalf("Failed to load template %s: %v", templateName, err)
			}
			if template == nil {
				t.Fatalf("Template %s is nil", templateName)
			}
			if template.Name != templateName {
				t.Errorf("Expected template name %s, got %s", templateName, template.Name)
			}
			if template.Content == "" {
				t.Errorf("Template %s has empty content", templateName)
			}
			if !strings.Contains(template.Content, "{{") {
				t.Errorf("Template %s does not contain any placeholders", templateName)
			}
			if len(template.Variables) == 0 {
				t.Errorf("Template %s has no variables extracted", templateName)
			}
		})
	}
}

// TestIntegration_PromptBuilderWorks verifies the prompt builder can correctly build prompts
func TestIntegration_PromptBuilderWorks(t *testing.T) {
	templateDir := getTemplateDir()
	pm := NewPromptManager(templateDir)

	template, err := pm.LoadTemplate("plan")
	if err != nil {
		t.Fatalf("Failed to load plan template: %v", err)
	}

	builder := NewPromptBuilder(template)
	builder.SetVariable("user_requirement", "Implement feature X")
	builder.SetContext("completed_work", "- Previous task 1")

	prompt, err := builder.Build()
	if err != nil {
		t.Fatalf("Failed to build prompt: %v", err)
	}
	if prompt == "" {
		t.Fatal("Built prompt is empty")
	}
	if strings.Contains(prompt, "{{user_requirement}}") {
		t.Error("user_requirement variable not replaced")
	}
	if !strings.Contains(prompt, "Implement feature X") {
		t.Error("user_requirement value not found in prompt")
	}
}

// TestIntegration_PlanPromptGeneration verifies the plan generator produces output.
func TestIntegration_PlanPromptGeneration(t *testing.T) {
	prompt, err := GeneratePlanPrompt("New requirement", "/tmp/test_plan", "")
	if err != nil {
		t.Fatalf("Failed to generate plan prompt: %v", err)
	}
	if prompt == "" {
		t.Fatal("Plan prompt is empty")
	}
	if !strings.Contains(prompt, "New requirement") {
		t.Error("Plan prompt does not contain user requirement")
	}
}

// Helper function to get template directory
func getTemplateDir() string {
	cwd, _ := os.Getwd()
	if strings.HasSuffix(cwd, "prompt") {
		return filepath.Join(cwd, "templates")
	}
	possiblePaths := []string{
		"internal/prompt/templates",
		"./internal/prompt/templates",
		"../prompt/templates",
	}
	for _, path := range possiblePaths {
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}
	return "internal/prompt/templates"
}
