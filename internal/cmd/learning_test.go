package cmd

import (
	"strings"
	"testing"

	"github.com/sunquan/rick/internal/executor"
)

func TestBuildLearningPrompt(t *testing.T) {
	// Create test data
	data := &ExecutionData{
		JobID:        "job_0",
		DebugContent: "Test debug content",
		TasksJSON: &executor.TasksJSON{
			Version: "1.0",
			Tasks: []executor.TaskState{
				{
					TaskID:     "task1",
					TaskName:   "Test Task",
					TaskFile:   "task1.md",
					Status:     "success",
					CommitHash: "abc123",
					Attempts:   1,
				},
			},
		},
	}
	
	// Build prompt
	prompt, err := buildLearningPrompt(data, "/tmp/test-learning")
	if err != nil {
		t.Fatalf("buildLearningPrompt failed: %v", err)
	}

	// Verify prompt contains expected sections
	if prompt == "" {
		t.Fatal("Prompt should not be empty")
	}

	// Check for key sections
	expectedSections := []string{
		"task1",
		"abc123",
	}
	
	for _, section := range expectedSections {
		if !strings.Contains(prompt, section) {
			t.Errorf("Prompt missing expected section: %s", section)
		}
	}
}

func TestExecutionDataStruct(t *testing.T) {
	data := &ExecutionData{
		JobID:        "job_0",
		DebugContent: "test",
		TasksJSON:    nil,
	}
	
	if data.JobID != "job_0" {
		t.Errorf("Expected JobID 'job_0', got '%s'", data.JobID)
	}
	
	if data.DebugContent != "test" {
		t.Errorf("Expected DebugContent 'test', got '%s'", data.DebugContent)
	}
}
