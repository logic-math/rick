package cmd

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNewLearningCmd(t *testing.T) {
	cmd := NewLearningCmd()

	if cmd == nil {
		t.Fatal("NewLearningCmd returned nil")
	}

	if cmd.Use != "learning [job_id]" {
		t.Errorf("Expected Use='learning [job_id]', got '%s'", cmd.Use)
	}

	if cmd.Short == "" {
		t.Error("Short description is empty")
	}

	if cmd.Long == "" {
		t.Error("Long description is empty")
	}
}

func TestLearningCmdFlags(t *testing.T) {
	cmd := NewLearningCmd()

	// Check that --job flag exists
	jobFlag := cmd.Flags().Lookup("job")
	if jobFlag == nil {
		t.Fatal("--job flag not found")
	}

	if jobFlag.Usage == "" {
		t.Error("--job flag has no usage description")
	}
}

func TestLoadExecutionResults(t *testing.T) {
	// Create temporary directory
	tmpDir := t.TempDir()

	// Create execution log
	logPath := filepath.Join(tmpDir, "execution.log")
	logContent := `Execution Summary
Total Tasks: 5
Successful Tasks: 4
Failed Tasks: 1`
	if err := os.WriteFile(logPath, []byte(logContent), 0644); err != nil {
		t.Fatalf("Failed to create execution log: %v", err)
	}

	// Load execution results
	results, err := loadExecutionResults(tmpDir, "test_job")
	if err != nil {
		t.Fatalf("loadExecutionResults failed: %v", err)
	}

	if results.JobID != "test_job" {
		t.Errorf("Expected JobID='test_job', got '%s'", results.JobID)
	}

	if results.TotalTasks != 5 {
		t.Errorf("Expected TotalTasks=5, got %d", results.TotalTasks)
	}

	if results.SuccessfulTasks != 4 {
		t.Errorf("Expected SuccessfulTasks=4, got %d", results.SuccessfulTasks)
	}

	if results.FailedTasks != 1 {
		t.Errorf("Expected FailedTasks=1, got %d", results.FailedTasks)
	}

	if results.ExecutionLog == "" {
		t.Error("ExecutionLog is empty")
	}
}

func TestLoadExecutionResultsWithDebugRecords(t *testing.T) {
	// Create temporary directory
	tmpDir := t.TempDir()

	// Create debug.md
	debugPath := filepath.Join(tmpDir, "debug.md")
	debugContent := `# Debug Records

- debug1: Issue 1, reproduction, hypothesis, verification, fix, pending
- debug2: Issue 2, reproduction, hypothesis, verification, fix, resolved`
	if err := os.WriteFile(debugPath, []byte(debugContent), 0644); err != nil {
		t.Fatalf("Failed to create debug.md: %v", err)
	}

	// Load execution results
	results, err := loadExecutionResults(tmpDir, "test_job")
	if err != nil {
		t.Fatalf("loadExecutionResults failed: %v", err)
	}

	if results.DebugRecords == "" {
		t.Error("DebugRecords is empty")
	}

	if !contains(results.DebugRecords, "Issue 1") {
		t.Error("DebugRecords does not contain expected content")
	}
}

func TestGenerateLearningPrompt(t *testing.T) {
	results := &ExecutionResults{
		JobID:           "test_job",
		TotalTasks:      5,
		SuccessfulTasks: 4,
		FailedTasks:     1,
		ExecutionLog:    "Sample execution log",
	}

	prompt, err := generateLearningPrompt("test_job", nil, results)
	if err != nil {
		t.Fatalf("generateLearningPrompt failed: %v", err)
	}

	if prompt == "" {
		t.Error("Generated prompt is empty")
	}

	if !contains(prompt, "test_job") {
		t.Error("Prompt does not contain job ID")
	}

	if !contains(prompt, "5") {
		t.Error("Prompt does not contain task count")
	}

	if !contains(prompt, "Learning Task") {
		t.Error("Prompt does not contain learning task section")
	}
}

func TestExtractKeyInsights(t *testing.T) {
	learningResult := `# Learning Summary

## Key Achievements
- Achievement 1
- Achievement 2

## Lessons Learned
- Lesson 1
- Lesson 2`

	insights := extractKeyInsights(learningResult)
	if insights == "" {
		t.Error("Extracted insights is empty")
	}

	if !contains(insights, "Achievement") {
		t.Error("Insights does not contain achievement section")
	}
}

func TestExtractImplementationNotes(t *testing.T) {
	learningResult := `# Learning Summary

## Recommendations
- Recommendation 1
- Recommendation 2

## Technical Improvements
- Improvement 1
- Improvement 2`

	notes := extractImplementationNotes(learningResult)
	if notes == "" {
		t.Error("Extracted notes is empty")
	}

	if !contains(notes, "Recommendation") {
		t.Error("Notes does not contain recommendation section")
	}
}

func TestAppendToFile(t *testing.T) {
	// Create temporary directory
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "test.md")

	// Create initial file
	initialContent := "# Initial Content\n"
	if err := os.WriteFile(filePath, []byte(initialContent), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Append content
	appendContent := "## Appended Content\n"
	if err := appendToFile(filePath, appendContent); err != nil {
		t.Fatalf("appendToFile failed: %v", err)
	}

	// Read file and verify
	data, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("Failed to read test file: %v", err)
	}

	content := string(data)
	if !contains(content, "Initial Content") {
		t.Error("Initial content not found")
	}

	if !contains(content, "Appended Content") {
		t.Error("Appended content not found")
	}
}

func TestAppendToFileCreate(t *testing.T) {
	// Create temporary directory
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "new_file.md")

	// Append to non-existent file (should create it)
	content := "# New File Content\n"
	if err := appendToFile(filePath, content); err != nil {
		t.Fatalf("appendToFile failed: %v", err)
	}

	// Verify file was created
	data, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("Failed to read created file: %v", err)
	}

	if !contains(string(data), "New File Content") {
		t.Error("File content not found")
	}
}

func TestUpdateDocumentation(t *testing.T) {
	// Create temporary directory structure
	tmpDir := t.TempDir()
	learningDir := filepath.Join(tmpDir, "learning")
	if err := os.MkdirAll(learningDir, 0755); err != nil {
		t.Fatalf("Failed to create learning directory: %v", err)
	}

	// Create OKR.md and SPEC.md
	okriPath := filepath.Join(tmpDir, "OKR.md")
	specPath := filepath.Join(tmpDir, "SPEC.md")
	if err := os.WriteFile(okriPath, []byte("# OKR\n"), 0644); err != nil {
		t.Fatalf("Failed to create OKR.md: %v", err)
	}
	if err := os.WriteFile(specPath, []byte("# SPEC\n"), 0644); err != nil {
		t.Fatalf("Failed to create SPEC.md: %v", err)
	}

	// Update documentation
	learningResult := "# Learning Results\n\n## Key Achievements\n- Achievement 1\n\n## Recommendations\n- Recommendation 1"
	if err := updateDocumentation(tmpDir, learningResult, learningDir); err != nil {
		t.Fatalf("updateDocumentation failed: %v", err)
	}

	// Verify learning summary was saved
	summaryPath := filepath.Join(learningDir, "learning_summary.md")
	if _, err := os.Stat(summaryPath); os.IsNotExist(err) {
		t.Error("Learning summary file not created")
	}

	// Verify OKR.md was updated
	okriData, err := os.ReadFile(okriPath)
	if err != nil {
		t.Fatalf("Failed to read OKR.md: %v", err)
	}
	if !contains(string(okriData), "Learning Insights") {
		t.Error("OKR.md not updated with learning insights")
	}

	// Verify SPEC.md was updated
	specData, err := os.ReadFile(specPath)
	if err != nil {
		t.Fatalf("Failed to read SPEC.md: %v", err)
	}
	if !contains(string(specData), "Implementation Notes") {
		t.Error("SPEC.md not updated with implementation notes")
	}
}

// contains is a helper function to check if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		func() bool {
			for i := 0; i <= len(s)-len(substr); i++ {
				if s[i:i+len(substr)] == substr {
					return true
				}
			}
			return false
		}())
}

