package prompt

import (
	"fmt"
	"os"
)

// humanLoopSkills 是 human-loop v7 会话需要复制到 loop 目录 prompts/ 的 skill 清单
// （SENSE canon + research + 五个 v7 教学 agent 的 system prompt 副本，供会话内
// 按需读取与留痕）。skill 文件名（下划线）与 pi agent 名（连字符）不同源部署。
var humanLoopSkills = []struct {
	filename  string
	skillName string
}{
	{"skill_sense.md", "sense"},
	{"skill_research.md", "research"},
	{"skill_zpd_assessor.md", "zpd_assessor"},
	{"skill_syllabus_builder.md", "syllabus_builder"},
	{"skill_lesson_writer.md", "lesson_writer"},
	{"skill_originality_sparker.md", "originality_sparker"},
	{"skill_recall_review.md", "recall_review"},
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
	builder.SetVariable("sense_skill_path", loopDir+"/prompts/skill_sense.md")
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
//	{{loop_dir}}/values.md
//	{{loop_dir}}/syllabus.md
//	{{loop_dir}}/zpd.md
//	{{loop_dir}}/anchors.md
//	{{loop_dir}}/research/research_{N}.md
//	{{loop_dir}}/lessons/teach_{N}.md + spark_{N}.md
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
// to write into them. v7：不再渲染 teach/research task prompt——六 agent 契约在
// 部署的 agent system prompt（templates/skills/*.md）里，task 文本由 parent 动态组装。
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

	// Write all v7 skill files (sense + research + 5 teaching agents).
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
	builder.SetVariable("sense_skill_path", promptsDir+"/skill_sense.md")
	setLearningMaterialVars(builder, loopDir)

	mainFile, err := builder.BuildAndSaveToDir("sense_loop", promptsDir)
	if err != nil {
		return "", fmt.Errorf("failed to build and save sense_loop prompt: %w", err)
	}

	return mainFile, nil
}
