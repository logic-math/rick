package prompt

import (
	"fmt"
	"os"
)

// GenerateHumanLoopPrompt generates the human-loop prompt string (for dry-run).
func GenerateHumanLoopPrompt(topic string, rfcDir string, draftDir string, manager *PromptManager) (string, error) {
	if manager == nil {
		return "", fmt.Errorf("prompt manager cannot be nil")
	}

	mainTmpl, err := manager.LoadTemplate("human_loop")
	if err != nil {
		return "", fmt.Errorf("failed to load human_loop template: %w", err)
	}

	builder := NewPromptBuilder(mainTmpl)
	builder.SetVariable("topic", topic)
	builder.SetVariable("rfc_dir", rfcDir)
	builder.SetVariable("draft_dir", draftDir)
	builder.SetVariable("loop_dir", "<draft>/loops/loop_N")
	builder.SetVariable("sense_agent_path", "<draft>/loops/loop_N/prompts/sense_subagent.md")

	content, err := builder.Build()
	if err != nil {
		return "", fmt.Errorf("failed to build human_loop prompt: %w", err)
	}

	return content, nil
}

// GenerateHumanLoopPromptFile generates the human-loop prompt and saves all files to
// loopDir/prompts/ for persistence across sessions.
// Returns mainFile, senseAgentFile, and any error.
func GenerateHumanLoopPromptFile(topic string, rfcDir string, draftDir string, loopDir string, manager *PromptManager) (string, string, error) {
	if manager == nil {
		return "", "", fmt.Errorf("prompt manager cannot be nil")
	}

	promptsDir := loopDir + "/prompts"
	briefsDir := loopDir + "/briefs"

	for _, d := range []string{promptsDir, briefsDir} {
		if err := os.MkdirAll(d, 0755); err != nil {
			return "", "", fmt.Errorf("failed to create directory %s: %w", d, err)
		}
	}

	senseSkillFile, err := WriteSkillFile(promptsDir, "skill_sense.md", "sense")
	if err != nil {
		return "", "", fmt.Errorf("failed to write sense skill: %w", err)
	}

	senseTmpl, err := manager.LoadTemplate("sense_subagent")
	if err != nil {
		return "", "", fmt.Errorf("failed to load sense_subagent template: %w", err)
	}
	senseBuilder := NewPromptBuilder(senseTmpl)
	senseBuilder.SetVariable("draft_dir", draftDir)
	senseBuilder.SetVariable("rfc_dir", rfcDir)
	senseBuilder.SetVariable("loop_dir", loopDir)
	senseBuilder.SetVariable("sense_skill_path", senseSkillFile)
	senseFile, err := senseBuilder.BuildAndSaveToDir("sense_subagent", promptsDir)
	if err != nil {
		return "", "", fmt.Errorf("failed to save sense_subagent: %w", err)
	}

	mainTmpl, err := manager.LoadTemplate("human_loop")
	if err != nil {
		return "", "", fmt.Errorf("failed to load human_loop template: %w", err)
	}

	builder := NewPromptBuilder(mainTmpl)
	builder.SetVariable("topic", topic)
	builder.SetVariable("rfc_dir", rfcDir)
	builder.SetVariable("draft_dir", draftDir)
	builder.SetVariable("loop_dir", loopDir)
	builder.SetVariable("sense_agent_path", senseFile)

	mainFile, err := builder.BuildAndSaveToDir("human_loop", promptsDir)
	if err != nil {
		return "", "", fmt.Errorf("failed to build and save human_loop prompt: %w", err)
	}

	return mainFile, senseFile, nil
}
