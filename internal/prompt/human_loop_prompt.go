package prompt

import (
	"fmt"
	"os"
)

// GenerateHumanLoopPrompt generates the human-loop prompt string (for dry-run).
// Sub-agent paths are replaced with placeholder descriptions.
func GenerateHumanLoopPrompt(topic string, rfcDir string, manager *PromptManager) (string, error) {
	if manager == nil {
		return "", fmt.Errorf("prompt manager cannot be nil")
	}

	thinkTmpl, err := manager.LoadTemplate("human_loop_think")
	if err != nil {
		return "", fmt.Errorf("failed to load human_loop_think template: %w", err)
	}
	learnTmpl, err := manager.LoadTemplate("human_loop_learn")
	if err != nil {
		return "", fmt.Errorf("failed to load human_loop_learn template: %w", err)
	}
	expressTmpl, err := manager.LoadTemplate("human_loop_express")
	if err != nil {
		return "", fmt.Errorf("failed to load human_loop_express template: %w", err)
	}

	_ = thinkTmpl
	_ = learnTmpl
	_ = expressTmpl

	mainTmpl, err := manager.LoadTemplate("human_loop")
	if err != nil {
		return "", fmt.Errorf("failed to load human_loop template: %w", err)
	}

	builder := NewPromptBuilder(mainTmpl)
	builder.SetVariable("topic", topic)
	builder.SetVariable("rfc_dir", rfcDir)
	builder.SetVariable("think_agent_path", "<tmp>/human_loop_think_*.md")
	builder.SetVariable("learn_agent_path", "<tmp>/human_loop_learn_*.md")
	builder.SetVariable("express_agent_path", "<tmp>/human_loop_express_*.md")

	content, err := builder.Build()
	if err != nil {
		return "", fmt.Errorf("failed to build human_loop prompt: %w", err)
	}

	return content, nil
}

// GenerateHumanLoopPromptFile generates the human-loop prompt and saves it to a temporary file.
// It also writes the three sub-agent templates to tmp files and injects their paths.
// Returns mainFile, subAgentFiles (think, learn, express), and any error.
// The caller is responsible for cleaning up all returned files.
func GenerateHumanLoopPromptFile(topic string, rfcDir string, manager *PromptManager) (string, []string, error) {
	if manager == nil {
		return "", nil, fmt.Errorf("prompt manager cannot be nil")
	}

	// Build and save each sub-agent template
	thinkTmpl, err := manager.LoadTemplate("human_loop_think")
	if err != nil {
		return "", nil, fmt.Errorf("failed to load human_loop_think template: %w", err)
	}
	thinkBuilder := NewPromptBuilder(thinkTmpl)
	thinkFile, err := thinkBuilder.BuildAndSave("human_loop_think")
	if err != nil {
		return "", nil, fmt.Errorf("failed to save human_loop_think: %w", err)
	}

	learnTmpl, err := manager.LoadTemplate("human_loop_learn")
	if err != nil {
		cleanupFiles([]string{thinkFile})
		return "", nil, fmt.Errorf("failed to load human_loop_learn template: %w", err)
	}
	learnBuilder := NewPromptBuilder(learnTmpl)
	learnFile, err := learnBuilder.BuildAndSave("human_loop_learn")
	if err != nil {
		cleanupFiles([]string{thinkFile})
		return "", nil, fmt.Errorf("failed to save human_loop_learn: %w", err)
	}

	expressTmpl, err := manager.LoadTemplate("human_loop_express")
	if err != nil {
		cleanupFiles([]string{thinkFile, learnFile})
		return "", nil, fmt.Errorf("failed to load human_loop_express template: %w", err)
	}
	expressBuilder := NewPromptBuilder(expressTmpl)
	expressFile, err := expressBuilder.BuildAndSave("human_loop_express")
	if err != nil {
		cleanupFiles([]string{thinkFile, learnFile})
		return "", nil, fmt.Errorf("failed to save human_loop_express: %w", err)
	}

	subAgentFiles := []string{thinkFile, learnFile, expressFile}

	// Build main prompt with sub-agent paths injected
	mainTmpl, err := manager.LoadTemplate("human_loop")
	if err != nil {
		cleanupFiles(subAgentFiles)
		return "", nil, fmt.Errorf("failed to load human_loop template: %w", err)
	}

	builder := NewPromptBuilder(mainTmpl)
	builder.SetVariable("topic", topic)
	builder.SetVariable("rfc_dir", rfcDir)
	builder.SetVariable("think_agent_path", thinkFile)
	builder.SetVariable("learn_agent_path", learnFile)
	builder.SetVariable("express_agent_path", expressFile)

	mainFile, err := builder.BuildAndSave("human_loop")
	if err != nil {
		cleanupFiles(subAgentFiles)
		return "", nil, fmt.Errorf("failed to build and save human_loop prompt: %w", err)
	}

	return mainFile, subAgentFiles, nil
}

func cleanupFiles(files []string) {
	for _, f := range files {
		if f != "" {
			_ = os.Remove(f)
		}
	}
}
