package handler

import (
	"strings"
	"testing"
)

// TestRenderHumanLoopPrompt 验证 v8 思考门禁提示词构建（update-pi 渲染冒烟的底层）：
// 非空、含 v8 协议核心锚词（五阶段门禁 + orig 启发 + thought_path 唯一产出）、
// 无未替换的 {{placeholder}}、无 v7 教学机制残留。
func TestRenderHumanLoopPrompt(t *testing.T) {
	content, err := RenderHumanLoopPrompt("render smoke test")
	if err != nil {
		t.Fatalf("RenderHumanLoopPrompt: %v", err)
	}
	if strings.TrimSpace(content) == "" {
		t.Fatal("rendered human-loop prompt is empty")
	}
	// v8 核心锚词：五阶段名（human 裁决表述）+ 门禁 + orig + thought_path。
	for _, anchor := range []string{
		"问题还原", "视角生产", "判断矛盾", "辩证逆转", "良质批判",
		"门禁", "orig", "thought_path.md", "推进权",
	} {
		if !strings.Contains(content, anchor) {
			t.Errorf("rendered prompt should contain v8 core anchor %q", anchor)
		}
	}
	// v8 skill 引用：五阶段 + orig + sense canon。
	for _, skill := range []string{
		"skill_sense_reconstruct.md", "skill_sense_perspective.md", "skill_sense_contradiction.md",
		"skill_sense_reversal.md", "skill_sense_critique.md", "skill_orig.md", "skill_sense.md",
	} {
		if !strings.Contains(content, skill) {
			t.Errorf("rendered prompt should reference v8 skill %q", skill)
		}
	}
	// v7 教学机制不得残留。
	for _, gone := range []string{
		"教学会话 v7", "ZPD", "费曼", "探索流", "文件状态机", "价值裁决",
		"mission.md", "values.md", "syllabus", "lesson-writer", "recall-review",
	} {
		if strings.Contains(content, gone) {
			t.Errorf("rendered prompt should not contain v7 teaching keyword %q", gone)
		}
	}
	if strings.Contains(content, "{{") {
		t.Error("rendered prompt contains unreplaced {{placeholder}}")
	}
}
