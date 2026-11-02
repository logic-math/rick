package handler

import (
	"strings"
	"testing"
)

// TestRenderHumanLoopPrompt 验证 v7 教学提示词构建（update-pi 渲染冒烟的底层）：
// 非空、含 v7 协议标题与核心节锚词、且无未替换的 {{placeholder}}。
func TestRenderHumanLoopPrompt(t *testing.T) {
	content, err := RenderHumanLoopPrompt("render smoke test")
	if err != nil {
		t.Fatalf("RenderHumanLoopPrompt: %v", err)
	}
	if strings.TrimSpace(content) == "" {
		t.Fatal("rendered human-loop prompt is empty")
	}
	if !strings.Contains(content, "教学会话 v7") {
		t.Error("rendered prompt should contain the v7 teaching session protocol header")
	}
	// v7.2 核心节锚词：SENSE 第一步新名 + 核心循环机制（费曼演绎内联+探索流）+ 五步编排。
	for _, anchor := range []string{"问题还原", "费曼", "ZPD", "探索流", "文件状态机", "价值裁决"} {
		if !strings.Contains(content, anchor) {
			t.Errorf("rendered prompt should contain v7 core anchor %q", anchor)
		}
	}
	// 六 agent 名（连字符 pi agent 名）。
	// v7.2：四个重生型 agent + 三个 parent 内联方法论 skill。
	for _, agent := range []string{"syllabus-builder", "lesson-writer", "recall-review", "research"} {
		if !strings.Contains(content, agent) {
			t.Errorf("rendered prompt should reference v7.2 agent %q", agent)
		}
	}
	for _, skill := range []string{"skill_zpd.md", "skill_feynman.md", "skill_exploration_stream.md"} {
		if !strings.Contains(content, skill) {
			t.Errorf("rendered prompt should reference parent inline methodology skill %q", skill)
		}
	}
	if strings.Contains(content, "{{") {
		t.Error("rendered prompt contains unreplaced {{placeholder}}")
	}
}
