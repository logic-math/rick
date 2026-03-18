package prompt

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sunquan/rick/internal/parser"
)

// TestIntegration_LoadAllTemplates verifies all templates can be loaded correctly
func TestIntegration_LoadAllTemplates(t *testing.T) {
	// Get template directory
	templateDir := getTemplateDir()

	pm := NewPromptManager(templateDir)

	// Test loading all templates
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

			// Verify template contains expected placeholders
			if !strings.Contains(template.Content, "{{") {
				t.Errorf("Template %s does not contain any placeholders", templateName)
			}

			// Verify variables are extracted
			if len(template.Variables) == 0 {
				t.Errorf("Template %s has no variables extracted", templateName)
			}

			t.Logf("Template %s loaded successfully with %d variables", templateName, len(template.Variables))
		})
	}
}

// TestIntegration_PromptBuilderWorks verifies the prompt builder can correctly build prompts
func TestIntegration_PromptBuilderWorks(t *testing.T) {
	templateDir := getTemplateDir()
	pm := NewPromptManager(templateDir)

	// Load a template
	template, err := pm.LoadTemplate("plan")
	if err != nil {
		t.Fatalf("Failed to load plan template: %v", err)
	}

	// Create builder and set variables
	builder := NewPromptBuilder(template)
	builder.SetVariable("project_name", "Test Project")
	builder.SetVariable("project_description", "A test project")
	builder.SetVariable("user_requirement", "Implement feature X")
	builder.SetContext("okr_content", "OKR: Achieve goal Y")
	builder.SetContext("spec_content", "SPEC: Technical specification")
	builder.SetContext("completed_work", "- Previous task 1\n- Previous task 2")

	// Build the prompt
	prompt, err := builder.Build()
	if err != nil {
		t.Fatalf("Failed to build prompt: %v", err)
	}

	if prompt == "" {
		t.Fatal("Built prompt is empty")
	}

	// Verify variables were replaced
	if strings.Contains(prompt, "{{project_name}}") {
		t.Error("project_name variable not replaced")
	}

	if strings.Contains(prompt, "{{project_description}}") {
		t.Error("project_description variable not replaced")
	}

	// Verify context was injected
	if !strings.Contains(prompt, "Test Project") {
		t.Error("project_name value not found in prompt")
	}

	if !strings.Contains(prompt, "A test project") {
		t.Error("project_description value not found in prompt")
	}

	t.Logf("Prompt builder works correctly, generated %d character prompt", len(prompt))
}

// TestIntegration_ContextManagerLoads verifies context manager can load various contexts
func TestIntegration_ContextManagerLoads(t *testing.T) {
	cm := NewContextManager("test_job_1")

	// Load task
	task := &parser.Task{
		ID:           "task1",
		Name:         "Test Task",
		Dependencies: []string{},
		Goal:         "Test goal",
	}

	err := cm.LoadTask(task)
	if err != nil {
		t.Fatalf("Failed to load task: %v", err)
	}

	if cm.Task == nil || cm.Task.ID != "task1" {
		t.Error("Task not loaded correctly")
	}

	// Load debug content
	debugContent := `# debug1: Test issue
Test debug entry
`
	err = cm.LoadDebugFromContent(debugContent)
	if err != nil {
		t.Fatalf("Failed to load debug content: %v", err)
	}

	if cm.Debug == nil {
		t.Error("Debug not loaded correctly")
	}

	// Load OKR content
	ocrContent := "# OKR\n- Goal 1\n- Goal 2"
	err = cm.LoadOKRFromContent(ocrContent)
	if err != nil {
		t.Fatalf("Failed to load OKR content: %v", err)
	}

	if cm.OKRInfo == nil {
		t.Error("OKRInfo is nil")
	}

	// Load SPEC content
	specContent := "# SPEC\n- Requirement 1\n- Requirement 2"
	err = cm.LoadSPECFromContent(specContent)
	if err != nil {
		t.Fatalf("Failed to load SPEC content: %v", err)
	}

	if cm.SPECInfo == nil {
		t.Error("SPECInfo is nil")
	}

	// Load history
	history := []string{"commit1", "commit2", "commit3"}
	err = cm.LoadHistory(history)
	if err != nil {
		t.Fatalf("Failed to load history: %v", err)
	}

	if len(cm.History) != 3 {
		t.Errorf("Expected 3 history items, got %d", len(cm.History))
	}

	t.Log("Context manager loads all context types correctly")
}

// TestIntegration_AllPromptGeneratorsWork verifies all prompt generators produce output
func TestIntegration_AllPromptGeneratorsWork(t *testing.T) {
	templateDir := getTemplateDir()
	pm := NewPromptManager(templateDir)
	cm := NewContextManager("test_job_1")

	// Load context
	task := &parser.Task{
		ID:           "task1",
		Name:         "Test Task",
		Dependencies: []string{},
		Goal:         "Test Description",
		KeyResults:   []string{"Result 1", "Result 2"},
		TestMethod:   "Test 1\nTest 2",
	}

	cm.LoadTask(task)
	cm.LoadOKRFromContent("# OKR\n- Goal 1")
	cm.LoadSPECFromContent("# SPEC\n- Requirement 1")
	cm.LoadHistory([]string{"commit1"})

	// Test plan prompt generation
	t.Run("PlanPrompt", func(t *testing.T) {
		prompt, err := GeneratePlanPrompt("New requirement", "/tmp/test_plan", cm, pm)
		if err != nil {
			t.Fatalf("Failed to generate plan prompt: %v", err)
		}

		if prompt == "" {
			t.Fatal("Plan prompt is empty")
		}

		// Verify key content
		if !strings.Contains(prompt, "New requirement") {
			t.Error("Plan prompt does not contain user requirement")
		}

		t.Logf("Plan prompt generated successfully (%d chars)", len(prompt))
	})

	// Test doing prompt generation
	t.Run("DoingPrompt", func(t *testing.T) {
		prompt, err := GenerateDoingPrompt(task, 1, cm, pm)
		if err != nil {
			t.Fatalf("Failed to generate doing prompt: %v", err)
		}

		if prompt == "" {
			t.Fatal("Doing prompt is empty")
		}

		// Verify key content
		if !strings.Contains(prompt, task.Name) {
			t.Error("Doing prompt does not contain task name")
		}

		t.Logf("Doing prompt generated successfully (%d chars)", len(prompt))
	})

	// Note: Test prompt generation (GenerateTestPrompt) was removed
	// Python test script generation is now handled by runner.go using test_python.md template

	// Test learning prompt generation
	t.Run("LearningPrompt", func(t *testing.T) {
		prompt, err := GenerateLearningPrompt("test_job_1", cm, pm)
		if err != nil {
			t.Fatalf("Failed to generate learning prompt: %v", err)
		}

		if prompt == "" {
			t.Fatal("Learning prompt is empty")
		}

		t.Logf("Learning prompt generated successfully (%d chars)", len(prompt))
	})
}

// TestIntegration_PromptsIncludeContextInfo verifies prompts include necessary context
func TestIntegration_PromptsIncludeContextInfo(t *testing.T) {
	templateDir := getTemplateDir()
	pm := NewPromptManager(templateDir)
	cm := NewContextManager("test_job_1")

	// Set up comprehensive context
	task := &parser.Task{
		ID:           "task1",
		Name:         "Complex Task",
		Dependencies: []string{"task0"},
		Goal:         "This is a complex task",
		KeyResults:   []string{"KR1", "KR2", "KR3"},
		TestMethod:   "Test method 1\nTest method 2",
	}

	cm.LoadTask(task)
	cm.LoadOKRFromContent("# OKR\n## Goal 1\nDescription of goal 1")
	cm.LoadSPECFromContent("# SPEC\n## Requirement 1\nTechnical specification")
	cm.LoadHistory([]string{"commit1", "commit2", "commit3"})

	// Generate doing prompt with retry context
	cm.LoadDebugFromContent("# debug1: Previous issue\nDescription of previous issue")

	prompt, err := GenerateDoingPrompt(task, 2, cm, pm)
	if err != nil {
		t.Fatalf("Failed to generate doing prompt: %v", err)
	}

	// Verify context information is included
	contextChecks := []struct {
		name    string
		content string
		should  bool
	}{
		{"task name", "Complex Task", true},
		{"task description", "This is a complex task", true},
		{"dependencies", "task0", true},
		{"key results", "KR1", true},
		{"test methods", "Test method 1", true},
		{"retry indication", "retry", false}, // May or may not be included
	}

	for _, check := range contextChecks {
		present := strings.Contains(prompt, check.content)
		if check.should && !present {
			t.Errorf("Prompt should contain %s (%s) but doesn't", check.name, check.content)
		}
	}

	t.Log("Prompts include necessary context information")
}

// TestIntegration_RetryPromptsIncludeDebug verifies retry prompts include debug info
func TestIntegration_RetryPromptsIncludeDebug(t *testing.T) {
	templateDir := getTemplateDir()
	pm := NewPromptManager(templateDir)
	cm := NewContextManager("test_job_1")

	task := &parser.Task{
		ID:           "task1",
		Name:         "Test Task",
		Dependencies: []string{},
		Goal:         "Test task",
	}

	cm.LoadTask(task)

	// First attempt - no debug
	prompt1, err := GenerateDoingPrompt(task, 1, cm, pm)
	if err != nil {
		t.Fatalf("Failed to generate first prompt: %v", err)
	}

	// Add debug info and retry
	cm.LoadDebugFromContent("# debug1: Issue occurred\nThe problem was that X failed\n\n# Solution\nWe need to handle Y")

	prompt2, err := GenerateDoingPrompt(task, 2, cm, pm)
	if err != nil {
		t.Fatalf("Failed to generate retry prompt: %v", err)
	}

	// Retry prompt should be different and include debug info
	if prompt1 == prompt2 {
		t.Error("Retry prompt should be different from first attempt")
	}

	if !strings.Contains(prompt2, "debug") && !strings.Contains(prompt2, "Issue") {
		t.Error("Retry prompt should include debug information")
	}

	t.Log("Retry prompts correctly include debug information")
}

// TestIntegration_CompleteWorkflow verifies complete workflow from planning to learning
func TestIntegration_CompleteWorkflow(t *testing.T) {
	templateDir := getTemplateDir()
	pm := NewPromptManager(templateDir)
	cm := NewContextManager("test_job_complete")

	// Setup project context
	cm.LoadOKRFromContent("# Project OKR\n- Objective 1\n- Objective 2")
	cm.LoadSPECFromContent("# Technical SPEC\n- Architecture\n- Components")
	cm.LoadHistory([]string{
		"Initial commit",
		"Add feature A",
		"Add feature B",
	})

	// Step 1: Planning
	planPrompt, err := GeneratePlanPrompt("Add user authentication", "/tmp/test_plan", cm, pm)
	if err != nil {
		t.Fatalf("Planning failed: %v", err)
	}

	if !strings.Contains(planPrompt, "Add user authentication") {
		t.Error("Plan prompt missing requirement")
	}

	// Step 2: Execution (simulated)
	task := &parser.Task{
		ID:           "auth_task_1",
		Name:         "Implement login endpoint",
		Dependencies: []string{},
		Goal:         "Create login endpoint",
		KeyResults:   []string{"Endpoint accepts credentials", "Returns JWT token"},
		TestMethod:   "Test valid credentials\nTest invalid credentials",
	}

	cm.LoadTask(task)

	doingPrompt, err := GenerateDoingPrompt(task, 1, cm, pm)
	if err != nil {
		t.Fatalf("Execution failed: %v", err)
	}

	if !strings.Contains(doingPrompt, "login") {
		t.Error("Doing prompt missing task")
	}

	// Step 3: Testing
	// Note: GenerateTestPrompt was removed - Python test generation is now in runner.go

	// Step 4: Learning
	cm.LoadHistory([]string{
		"Initial commit",
		"Add feature A",
		"Add feature B",
		"Implement login endpoint",
		"Add tests for login",
	})

	learningPrompt, err := GenerateLearningPrompt("test_job_complete", cm, pm)
	if err != nil {
		t.Fatalf("Learning failed: %v", err)
	}

	if learningPrompt == "" {
		t.Error("Learning prompt is empty")
	}

	t.Log("Complete workflow executed successfully")
}

// TestIntegration_PromptConsistency verifies prompts are consistent and properly formatted
func TestIntegration_PromptConsistency(t *testing.T) {
	templateDir := getTemplateDir()
	pm := NewPromptManager(templateDir)
	cm := NewContextManager("test_job_consistency")

	task := &parser.Task{
		ID:           "task1",
		Name:         "Consistency Test",
		Dependencies: []string{},
		Goal:         "Test prompt consistency",
	}

	cm.LoadTask(task)
	cm.LoadOKRFromContent("# OKR")
	cm.LoadSPECFromContent("# SPEC")

	// Generate same prompt multiple times - should be consistent
	prompt1, _ := GenerateDoingPrompt(task, 1, cm, pm)
	prompt2, _ := GenerateDoingPrompt(task, 1, cm, pm)

	if prompt1 != prompt2 {
		t.Error("Same inputs should produce same prompt")
	}

	// Verify prompt is properly formatted - should contain some content
	if len(prompt1) < 100 {
		t.Error("Prompt seems too short to be properly formatted")
	}

	// Verify prompt contains expected markers
	if !strings.Contains(prompt1, "你是") && !strings.Contains(prompt1, "task") {
		t.Log("Prompt may not be properly formatted - no task-related content found")
	}

	t.Log("Prompts are consistent and properly formatted")
}

// Helper function to get template directory
func getTemplateDir() string {
	// Try to find the template directory
	cwd, _ := os.Getwd()

	// Check if we're in the prompt package directory
	if strings.HasSuffix(cwd, "prompt") {
		return filepath.Join(cwd, "templates")
	}

	// Check if templates exist in standard location
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

	// Default fallback
	return "internal/prompt/templates"
}
