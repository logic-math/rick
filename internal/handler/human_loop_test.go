package handler

import (
	"strings"
	"testing"
)

// TestRenderHumanLoopPrompt 验证 SENSE 提示词构建（update-pi 渲染冒烟的底层）：
// 非空、含协议标题、且无未替换的 {{placeholder}}。
func TestRenderHumanLoopPrompt(t *testing.T) {
	content, err := RenderHumanLoopPrompt("render smoke test")
	if err != nil {
		t.Fatalf("RenderHumanLoopPrompt: %v", err)
	}
	if strings.TrimSpace(content) == "" {
		t.Fatal("rendered human-loop prompt is empty")
	}
	if !strings.Contains(content, "sense loop") {
		t.Error("rendered prompt should contain the sense loop protocol header")
	}
	if strings.Contains(content, "{{") {
		t.Error("rendered prompt contains unreplaced {{placeholder}}")
	}
}
