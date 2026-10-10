package prompt

import (
	"fmt"
	"os"
)

// GenerateHumanLoopPrompt generates the human-loop (teacher session) prompt string
// (for dry-run). Returns the main sense_loop prompt content with learning-material
// paths injected.
func GenerateHumanLoopPrompt(topic string, rfcDir string, draftDir string, manager *PromptManager) (string, error) {
	if manager == nil {
		return "", fmt.Errorf("prompt manager cannot be nil")
	}

	mainTmpl, err := manager.LoadTemplate("sense_loop")
	if err != nil {
		return "", fmt.Errorf("failed to load sense_loop template: %w", err)
	}

	loopDir := "<draft>/loops/loop_N"
	builder := NewPromptBuilder(mainTmpl)
	builder.SetVariable("topic", topic)
	builder.SetVariable("rfc_dir", rfcDir)
	builder.SetVariable("draft_dir", draftDir)
	builder.SetVariable("loop_dir", loopDir)
	builder.SetVariable("teach_agent_path", loopDir+"/prompts/teach.md")
	builder.SetVariable("research_agent_path", loopDir+"/prompts/research.md")
	setLearningMaterialVars(builder, loopDir)

	content, err := builder.Build()
	if err != nil {
		return "", fmt.Errorf("failed to build sense_loop prompt: %w", err)
	}

	return content, nil
}

// setLearningMaterialVars injects the learning-material path convention into a
// template builder. Kept in one place so the teacher-session layout stays frozen:
//
//	{{loop_dir}}/mission.md
//	{{loop_dir}}/syllabus.md
//	{{loop_dir}}/research/research_{N}.md
//	{{loop_dir}}/lessons/teach_{N}.md
//	{{loop_dir}}/judgment.md
func setLearningMaterialVars(builder *PromptBuilder, loopDir string) {
	builder.SetVariable("mission_path", loopDir+"/mission.md")
	builder.SetVariable("syllabus_path", loopDir+"/syllabus.md")
	builder.SetVariable("research_dir", loopDir+"/research")
	builder.SetVariable("lessons_dir", loopDir+"/lessons")
	builder.SetVariable("judgment_path", loopDir+"/judgment.md")
}

// GenerateHumanLoopPromptFile generates the human-loop (teacher session) prompt and
// saves all files to loopDir/prompts/ for persistence across sessions.
// Returns mainFile (sense_loop), teachFile, researchFile, and any error.
// The learning-material directories loopDir/research/ and loopDir/lessons/ are
// pre-created so subagents only have to write into them.
func GenerateHumanLoopPromptFile(topic string, rfcDir string, draftDir string, loopDir string, manager *PromptManager) (string, string, string, error) {
	if manager == nil {
		return "", "", "", fmt.Errorf("prompt manager cannot be nil")
	}

	promptsDir := loopDir + "/prompts"
	researchDir := loopDir + "/research"
	lessonsDir := loopDir + "/lessons"

	for _, d := range []string{promptsDir, researchDir, lessonsDir} {
		if err := os.MkdirAll(d, 0755); err != nil {
			return "", "", "", fmt.Errorf("failed to create directory %s: %w", d, err)
		}
	}

	// Write all skill files (sense/teach/research)
	senseSkillFile, err := WriteSkillFile(promptsDir, "skill_sense.md", "sense")
	if err != nil {
		return "", "", "", fmt.Errorf("failed to write sense skill: %w", err)
	}
	teachSkillFile, err := WriteSkillFile(promptsDir, "skill_teach.md", "teach")
	if err != nil {
		return "", "", "", fmt.Errorf("failed to write teach skill: %w", err)
	}
	researchSkillFile, err := WriteSkillFile(promptsDir, "skill_research.md", "research")
	if err != nil {
		return "", "", "", fmt.Errorf("failed to write research skill: %w", err)
	}

	// Build and save teach subagent prompt
	teachTmpl, err := manager.LoadTemplate("teach")
	if err != nil {
		return "", "", "", fmt.Errorf("failed to load teach template: %w", err)
	}
	teachBuilder := NewPromptBuilder(teachTmpl)
	teachBuilder.SetVariable("draft_dir", draftDir)
	teachBuilder.SetVariable("rfc_dir", rfcDir)
	teachBuilder.SetVariable("loop_dir", loopDir)
	teachBuilder.SetVariable("teach_skill_path", teachSkillFile)
	setLearningMaterialVars(teachBuilder, loopDir)
	teachFile, err := teachBuilder.BuildAndSaveToDir("teach", promptsDir)
	if err != nil {
		return "", "", "", fmt.Errorf("failed to save teach: %w", err)
	}

	// Build and save research subagent prompt
	researchTmpl, err := manager.LoadTemplate("research")
	if err != nil {
		return "", "", "", fmt.Errorf("failed to load research template: %w", err)
	}
	researchBuilder := NewPromptBuilder(researchTmpl)
	researchBuilder.SetVariable("draft_dir", draftDir)
	researchBuilder.SetVariable("rfc_dir", rfcDir)
	researchBuilder.SetVariable("loop_dir", loopDir)
	researchBuilder.SetVariable("research_skill_path", researchSkillFile)
	setLearningMaterialVars(researchBuilder, loopDir)
	researchFile, err := researchBuilder.BuildAndSaveToDir("research", promptsDir)
	if err != nil {
		return "", "", "", fmt.Errorf("failed to save research: %w", err)
	}

	// Build and save main sense_loop prompt
	mainTmpl, err := manager.LoadTemplate("sense_loop")
	if err != nil {
		return "", "", "", fmt.Errorf("failed to load sense_loop template: %w", err)
	}

	builder := NewPromptBuilder(mainTmpl)
	builder.SetVariable("topic", topic)
	builder.SetVariable("rfc_dir", rfcDir)
	builder.SetVariable("draft_dir", draftDir)
	builder.SetVariable("loop_dir", loopDir)
	builder.SetVariable("teach_agent_path", teachFile)
	builder.SetVariable("research_agent_path", researchFile)
	setLearningMaterialVars(builder, loopDir)

	mainFile, err := builder.BuildAndSaveToDir("sense_loop", promptsDir)
	if err != nil {
		return "", "", "", fmt.Errorf("failed to build and save sense_loop prompt: %w", err)
	}
	_ = senseSkillFile // sense skill written for subagent reference

	return mainFile, teachFile, researchFile, nil
}
