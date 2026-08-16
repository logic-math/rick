package prompt

import (
	"fmt"
	"os"
)

// GenerateHumanLoopPrompt generates the human-loop prompt string (for dry-run).
// Returns the main sense_loop prompt content.
func GenerateHumanLoopPrompt(topic string, rfcDir string, draftDir string, manager *PromptManager) (string, error) {
	if manager == nil {
		return "", fmt.Errorf("prompt manager cannot be nil")
	}

	mainTmpl, err := manager.LoadTemplate("sense_loop")
	if err != nil {
		return "", fmt.Errorf("failed to load sense_loop template: %w", err)
	}

	builder := NewPromptBuilder(mainTmpl)
	builder.SetVariable("topic", topic)
	builder.SetVariable("rfc_dir", rfcDir)
	builder.SetVariable("draft_dir", draftDir)
	builder.SetVariable("loop_dir", "<draft>/loops/loop_N")
	builder.SetVariable("think_agent_path", "<draft>/loops/loop_N/prompts/think.md")
	builder.SetVariable("research_agent_path", "<draft>/loops/loop_N/prompts/research.md")
	builder.SetVariable("exporter_agent_path", "<draft>/loops/loop_N/prompts/exporter.md")
	builder.SetVariable("max_retries", "5")
	builder.SetVariable("max_backflows", "3")

	content, err := builder.Build()
	if err != nil {
		return "", fmt.Errorf("failed to build sense_loop prompt: %w", err)
	}

	return content, nil
}

// GenerateHumanLoopPromptFile generates the human-loop prompt and saves all files to
// loopDir/prompts/ for persistence across sessions.
// Returns mainFile (sense_loop), thinkFile, researchFile, exporterFile, and any error.
func GenerateHumanLoopPromptFile(topic string, rfcDir string, draftDir string, loopDir string, manager *PromptManager) (string, string, string, string, error) {
	if manager == nil {
		return "", "", "", "", fmt.Errorf("prompt manager cannot be nil")
	}

	promptsDir := loopDir + "/prompts"
	briefsDir := loopDir + "/briefs"

	for _, d := range []string{promptsDir, briefsDir} {
		if err := os.MkdirAll(d, 0755); err != nil {
			return "", "", "", "", fmt.Errorf("failed to create directory %s: %w", d, err)
		}
	}

	// Write all skill files (sense/think/research/exporter)
	senseSkillFile, err := WriteSkillFile(promptsDir, "skill_sense.md", "sense")
	if err != nil {
		return "", "", "", "", fmt.Errorf("failed to write sense skill: %w", err)
	}
	thinkSkillFile, err := WriteSkillFile(promptsDir, "skill_think.md", "think")
	if err != nil {
		return "", "", "", "", fmt.Errorf("failed to write think skill: %w", err)
	}
	researchSkillFile, err := WriteSkillFile(promptsDir, "skill_research.md", "research")
	if err != nil {
		return "", "", "", "", fmt.Errorf("failed to write research skill: %w", err)
	}
	exporterSkillFile, err := WriteSkillFile(promptsDir, "skill_exporter.md", "exporter")
	if err != nil {
		return "", "", "", "", fmt.Errorf("failed to write exporter skill: %w", err)
	}

	// Build and save think subagent prompt
	thinkTmpl, err := manager.LoadTemplate("think")
	if err != nil {
		return "", "", "", "", fmt.Errorf("failed to load think template: %w", err)
	}
	thinkBuilder := NewPromptBuilder(thinkTmpl)
	thinkBuilder.SetVariable("draft_dir", draftDir)
	thinkBuilder.SetVariable("rfc_dir", rfcDir)
	thinkBuilder.SetVariable("loop_dir", loopDir)
	thinkBuilder.SetVariable("think_skill_path", thinkSkillFile)
	thinkFile, err := thinkBuilder.BuildAndSaveToDir("think", promptsDir)
	if err != nil {
		return "", "", "", "", fmt.Errorf("failed to save think: %w", err)
	}

	// Build and save research subagent prompt
	researchTmpl, err := manager.LoadTemplate("research")
	if err != nil {
		return "", "", "", "", fmt.Errorf("failed to load research template: %w", err)
	}
	researchBuilder := NewPromptBuilder(researchTmpl)
	researchBuilder.SetVariable("draft_dir", draftDir)
	researchBuilder.SetVariable("rfc_dir", rfcDir)
	researchBuilder.SetVariable("loop_dir", loopDir)
	researchBuilder.SetVariable("research_skill_path", researchSkillFile)
	researchFile, err := researchBuilder.BuildAndSaveToDir("research", promptsDir)
	if err != nil {
		return "", "", "", "", fmt.Errorf("failed to save research: %w", err)
	}

	// Build and save exporter subagent prompt
	exporterTmpl, err := manager.LoadTemplate("exporter")
	if err != nil {
		return "", "", "", "", fmt.Errorf("failed to load exporter template: %w", err)
	}
	exporterBuilder := NewPromptBuilder(exporterTmpl)
	exporterBuilder.SetVariable("draft_dir", draftDir)
	exporterBuilder.SetVariable("rfc_dir", rfcDir)
	exporterBuilder.SetVariable("loop_dir", loopDir)
	exporterBuilder.SetVariable("exporter_skill_path", exporterSkillFile)
	exporterFile, err := exporterBuilder.BuildAndSaveToDir("exporter", promptsDir)
	if err != nil {
		return "", "", "", "", fmt.Errorf("failed to save exporter: %w", err)
	}

	// Build and save main sense_loop prompt
	mainTmpl, err := manager.LoadTemplate("sense_loop")
	if err != nil {
		return "", "", "", "", fmt.Errorf("failed to load sense_loop template: %w", err)
	}

	builder := NewPromptBuilder(mainTmpl)
	builder.SetVariable("topic", topic)
	builder.SetVariable("rfc_dir", rfcDir)
	builder.SetVariable("draft_dir", draftDir)
	builder.SetVariable("loop_dir", loopDir)
	builder.SetVariable("think_agent_path", thinkFile)
	builder.SetVariable("research_agent_path", researchFile)
	builder.SetVariable("exporter_agent_path", exporterFile)
	builder.SetVariable("max_retries", "5")
	builder.SetVariable("max_backflows", "3")
	builder.SetVariable("min_assumptions", "5")

	mainFile, err := builder.BuildAndSaveToDir("sense_loop", promptsDir)
	if err != nil {
		return "", "", "", "", fmt.Errorf("failed to build and save sense_loop prompt: %w", err)
	}
	_ = senseSkillFile // sense skill written for subagent reference

	return mainFile, thinkFile, researchFile, exporterFile, nil
}
