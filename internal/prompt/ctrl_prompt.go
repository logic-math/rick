package prompt

import (
	"fmt"
	"os"
	"path/filepath"
)

// GenerateCtrlPromptFile generates the ctrl monitoring prompt for a given job.
// The prompt is persisted to doing/prompts/ctrl_prompt.md and its path is returned.
func GenerateCtrlPromptFile(jobID, rickDir string) (string, error) {
	doingDir := filepath.Join(rickDir, "jobs", jobID, "doing")
	planDir := filepath.Join(rickDir, "jobs", jobID, "plan")
	tasksJSONPath := filepath.Join(doingDir, "tasks.json")

	if _, err := os.Stat(doingDir); os.IsNotExist(err) {
		return "", fmt.Errorf("doing directory not found for job %s: %s", jobID, doingDir)
	}

	tasksJSONContent := readFileOrDefault(tasksJSONPath, `{"tasks": []}`)

	promptsDir, err := EnsurePromptsDir(doingDir)
	if err != nil {
		return "", fmt.Errorf("failed to create prompts dir: %w", err)
	}

	mgr := NewPromptManager()
	tmpl, err := mgr.LoadTemplate("ctrl")
	if err != nil {
		return "", fmt.Errorf("failed to load ctrl template: %w", err)
	}

	builder := NewPromptBuilder(tmpl)
	builder.SetVariable("job_id", jobID)
	builder.SetVariable("doing_dir", doingDir)
	builder.SetVariable("plan_dir", planDir)
	builder.SetVariable("tasks_json_path", tasksJSONPath)
	builder.SetVariable("tasks_json_content", tasksJSONContent)

	content, err := builder.Build()
	if err != nil {
		return "", fmt.Errorf("failed to build ctrl prompt: %w", err)
	}

	promptFile := filepath.Join(promptsDir, "ctrl_prompt.md")
	if err := os.WriteFile(promptFile, []byte(content), 0644); err != nil {
		return "", fmt.Errorf("failed to write ctrl prompt: %w", err)
	}

	return promptFile, nil
}
