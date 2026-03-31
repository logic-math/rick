package prompt

import (
	"fmt"
)

// GenerateHumanLoopPromptFile generates the human-loop prompt and saves it to a temporary file.
// It injects topic and rfcDir into the human_loop template.
// Returns the file path and any error. The caller is responsible for cleaning up the temporary file.
func GenerateHumanLoopPromptFile(topic string, rfcDir string, manager *PromptManager) (string, error) {
	if manager == nil {
		return "", fmt.Errorf("prompt manager cannot be nil")
	}

	template, err := manager.LoadTemplate("human_loop")
	if err != nil {
		return "", fmt.Errorf("failed to load human_loop template: %w", err)
	}

	builder := NewPromptBuilder(template)
	builder.SetVariable("topic", topic)
	builder.SetVariable("rfc_dir", rfcDir)

	promptFile, err := builder.BuildAndSave("human_loop")
	if err != nil {
		return "", fmt.Errorf("failed to build and save human_loop prompt: %w", err)
	}

	return promptFile, nil
}
