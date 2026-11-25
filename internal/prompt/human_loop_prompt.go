package prompt

import (
	"fmt"
	"os"
)

// humanLoopSkills 是 human-loop v8 会话需要复制到 loop 目录 prompts/ 的 skill 清单
// （7 个：SENSE canon + 五个阶段 skill + orig agent 的 system prompt 副本，
// 供会话内按需读取与留痕）。
// v8（job_57）：human-loop 彻底重构为 SENSE 五阶段思考门禁 loop——旧八个教学
// skill 副本（zpd/feynman/exploration_stream/syllabus_builder/lesson_writer/
// recall_review/research）全部下线，改为五阶段 skill（sense_reconstruct/
// sense_perspective/sense_contradiction/sense_reversal/sense_critique）+
// orig agent skill。
var humanLoopSkills = []struct {
	filename  string
	skillName string
}{
	{"skill_sense.md", "sense"},
	{"skill_sense_reconstruct.md", "sense_reconstruct"},
	{"skill_sense_perspective.md", "sense_perspective"},
	{"skill_sense_contradiction.md", "sense_contradiction"},
	{"skill_sense_reversal.md", "sense_reversal"},
	{"skill_sense_critique.md", "sense_critique"},
	{"skill_orig.md", "orig"},
}

// GenerateHumanLoopPrompt generates the human-loop (v8 thinking-gate loop)
// prompt string (for dry-run). Returns the main sense_loop prompt content with
// artifact paths injected.
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
	builder.SetVariable("draft_dir", draftDir)
	builder.SetVariable("loop_dir", loopDir)
	setV8SkillVars(builder, loopDir)
	setV8ArtifactVars(builder, loopDir)

	content, err := builder.Build()
	if err != nil {
		return "", fmt.Errorf("failed to build sense_loop prompt: %w", err)
	}

	return content, nil
}

// setV8SkillVars injects the v8 skill path variables shared by the dry-run and
// file-rendering variants: SENSE canon + five stage skills + orig agent skill.
func setV8SkillVars(builder *PromptBuilder, loopDir string) {
	builder.SetVariable("sense_skill_path", loopDir+"/prompts/skill_sense.md")
	builder.SetVariable("sense_reconstruct_skill_path", loopDir+"/prompts/skill_sense_reconstruct.md")
	builder.SetVariable("sense_perspective_skill_path", loopDir+"/prompts/skill_sense_perspective.md")
	builder.SetVariable("sense_contradiction_skill_path", loopDir+"/prompts/skill_sense_contradiction.md")
	builder.SetVariable("sense_reversal_skill_path", loopDir+"/prompts/skill_sense_reversal.md")
	builder.SetVariable("sense_critique_skill_path", loopDir+"/prompts/skill_sense_critique.md")
	builder.SetVariable("orig_skill_path", loopDir+"/prompts/skill_orig.md")
}

// setV8ArtifactVars injects the v8 artifact path convention into a template
// builder. The v8 human-loop has exactly two artifacts (human 已裁决：唯一产出物
// 是思考轨迹)：
//
//	{{loop_dir}}/thought_path.md   思考轨迹（human 原话 + AI 启发的时间线）
//	{{loop_dir}}/orig/stage{N}_{seq}.md   orig 中间结构（调研素材）
func setV8ArtifactVars(builder *PromptBuilder, loopDir string) {
	builder.SetVariable("thought_path_path", loopDir+"/thought_path.md")
	builder.SetVariable("orig_dir", loopDir+"/orig")
}

// GenerateHumanLoopPromptFile generates the human-loop (v8 thinking-gate loop)
// prompt and saves all files to loopDir/prompts/ for persistence across
// sessions. Returns mainFile (sense_loop) and any error. The v8 artifact
// directories loopDir/orig/ are pre-created and thought_path.md is initialized
// with a header line so the session (and resume) can append directly.
func GenerateHumanLoopPromptFile(topic string, rfcDir string, draftDir string, loopDir string, manager *PromptManager) (string, error) {
	if manager == nil {
		return "", fmt.Errorf("prompt manager cannot be nil")
	}

	promptsDir := loopDir + "/prompts"
	origDir := loopDir + "/orig"

	if err := os.MkdirAll(promptsDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create directory %s: %w", promptsDir, err)
	}
	if err := os.MkdirAll(origDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create directory %s: %w", origDir, err)
	}

	// Write all v8 skill files (canon + five stage skills + orig agent skill).
	for _, s := range humanLoopSkills {
		if _, err := WriteSkillFile(promptsDir, s.filename, s.skillName); err != nil {
			return "", fmt.Errorf("failed to write skill %s: %w", s.skillName, err)
		}
	}

	// Initialize thought_path.md（唯一产出物：思考轨迹）with a header line.
	thoughtPath := loopDir + "/thought_path.md"
	if _, err := os.Stat(thoughtPath); os.IsNotExist(err) {
		header := fmt.Sprintf("# 思考轨迹 — %s\n", topic)
		if err := os.WriteFile(thoughtPath, []byte(header), 0644); err != nil {
			return "", fmt.Errorf("failed to initialize thought_path.md: %w", err)
		}
	}

	// Build and save main sense_loop prompt.
	mainTmpl, err := manager.LoadTemplate("sense_loop")
	if err != nil {
		return "", fmt.Errorf("failed to load sense_loop template: %w", err)
	}

	builder := NewPromptBuilder(mainTmpl)
	builder.SetVariable("topic", topic)
	builder.SetVariable("draft_dir", draftDir)
	builder.SetVariable("loop_dir", loopDir)
	setV8SkillVars(builder, loopDir)
	setV8ArtifactVars(builder, loopDir)

	mainFile, err := builder.BuildAndSaveToDir("sense_loop", promptsDir)
	if err != nil {
		return "", fmt.Errorf("failed to build and save sense_loop prompt: %w", err)
	}

	return mainFile, nil
}
