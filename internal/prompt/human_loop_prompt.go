package prompt

import (
	"fmt"
	"os"
)

// humanLoopSkills 是 human-loop v7.x 会话需要复制到 loop 目录 prompts/ 的 skill 清单
// （8 个：SENSE canon + 三个 parent 内联方法论 + 四个重生型 agent 的 system prompt
// 副本，供会话内按需读取与留痕）。
// v7.2（job_50）：交互型环节收回 parent 会话内联执行——skill_zpd/skill_feynman/
// skill_exploration_stream 是 parent 的方法论文件；agent 收缩为 4 个重生型
// （research/syllabus-builder/lesson-writer/recall-review）。
var humanLoopSkills = []struct {
	filename  string
	skillName string
}{
	{"skill_sense.md", "sense"},
	{"skill_zpd.md", "zpd"},
	{"skill_feynman.md", "feynman"},
	{"skill_exploration_stream.md", "exploration_stream"},
	{"skill_syllabus_builder.md", "syllabus_builder"},
	{"skill_lesson_writer.md", "lesson_writer"},
	{"skill_recall_review.md", "recall_review"},
	{"skill_research.md", "research"},
}

// GenerateHumanLoopPrompt generates the human-loop (v7 teaching session) prompt
// string (for dry-run). Returns the main sense_loop prompt content with
// learning-material paths injected.
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
	setSkillVars(builder, loopDir)
	setLearningMaterialVars(builder, loopDir)

	content, err := builder.Build()
	if err != nil {
		return "", fmt.Errorf("failed to build sense_loop prompt: %w", err)
	}

	return content, nil
}

// setSkillVars injects the four methodology-skill path variables shared by the
// dry-run and file-rendering variants: SENSE canon + three parent inline
// methodology files (zpd/feynman/exploration_stream).
func setSkillVars(builder *PromptBuilder, loopDir string) {
	builder.SetVariable("sense_skill_path", loopDir+"/prompts/skill_sense.md")
	builder.SetVariable("zpd_skill_path", loopDir+"/prompts/skill_zpd.md")
	builder.SetVariable("feynman_skill_path", loopDir+"/prompts/skill_feynman.md")
	builder.SetVariable("exploration_skill_path", loopDir+"/prompts/skill_exploration_stream.md")
}

// setLearningMaterialVars injects the learning-material path convention into a
// template builder. Kept in one place so the teacher-session layout stays frozen:
//
//	{{loop_dir}}/mission.md
//	{{loop_dir}}/values.md
//	{{loop_dir}}/syllabus.md
//	{{loop_dir}}/zpd.md
//	{{loop_dir}}/anchors.md
//	{{loop_dir}}/research/research_{N}.md
//	{{loop_dir}}/lessons/teach_{N}.md
//	{{loop_dir}}/judgment.md
func setLearningMaterialVars(builder *PromptBuilder, loopDir string) {
	builder.SetVariable("mission_path", loopDir+"/mission.md")
	builder.SetVariable("values_path", loopDir+"/values.md")
	builder.SetVariable("syllabus_path", loopDir+"/syllabus.md")
	builder.SetVariable("zpd_path", loopDir+"/zpd.md")
	builder.SetVariable("anchors_path", loopDir+"/anchors.md")
	builder.SetVariable("research_dir", loopDir+"/research")
	builder.SetVariable("lessons_dir", loopDir+"/lessons")
	builder.SetVariable("judgment_path", loopDir+"/judgment.md")
}

// GenerateHumanLoopPromptFile generates the human-loop (v7 teaching session)
// prompt and saves all files to loopDir/prompts/ for persistence across sessions.
// Returns mainFile (sense_loop) and any error. The learning-material directories
// loopDir/research/ and loopDir/lessons/ are pre-created so subagents only have
// to write into them. v7.2：8 个 skill 副本（canon+3 内联方法论+4 agent 契约）；
// agent task 文本由 parent 动态组装，费曼/ZPD/探索流不派发（parent 内联执行）。
func GenerateHumanLoopPromptFile(topic string, rfcDir string, draftDir string, loopDir string, manager *PromptManager) (string, error) {
	if manager == nil {
		return "", fmt.Errorf("prompt manager cannot be nil")
	}

	promptsDir := loopDir + "/prompts"
	researchDir := loopDir + "/research"
	lessonsDir := loopDir + "/lessons"

	for _, d := range []string{promptsDir, researchDir, lessonsDir} {
		if err := os.MkdirAll(d, 0755); err != nil {
			return "", fmt.Errorf("failed to create directory %s: %w", d, err)
		}
	}

	// Write all v7.2 skill files (canon + 3 parent inline methodology + 4 agent skills).
	for _, s := range humanLoopSkills {
		if _, err := WriteSkillFile(promptsDir, s.filename, s.skillName); err != nil {
			return "", fmt.Errorf("failed to write skill %s: %w", s.skillName, err)
		}
	}

	// Build and save main sense_loop prompt.
	mainTmpl, err := manager.LoadTemplate("sense_loop")
	if err != nil {
		return "", fmt.Errorf("failed to load sense_loop template: %w", err)
	}

	builder := NewPromptBuilder(mainTmpl)
	builder.SetVariable("topic", topic)
	builder.SetVariable("rfc_dir", rfcDir)
	builder.SetVariable("draft_dir", draftDir)
	builder.SetVariable("loop_dir", loopDir)
	setSkillVars(builder, loopDir)
	setLearningMaterialVars(builder, loopDir)

	mainFile, err := builder.BuildAndSaveToDir("sense_loop", promptsDir)
	if err != nil {
		return "", fmt.Errorf("failed to build and save sense_loop prompt: %w", err)
	}

	return mainFile, nil
}
