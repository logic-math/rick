package web

import (
	"encoding/json"
	"strings"
	"testing"
)

// job_53 迭代 4：injectCompactionEstimates 的 chars/4 估算注入（human 反馈：
// 历史压缩卡显示压缩后 token + 压缩比例）。
func TestInjectCompactionEstimates(t *testing.T) {
	mkMsg := func(id, text string) json.RawMessage {
		return json.RawMessage(`{"type":"message","id":"` + id + `","message":{"role":"user","content":"` + text + `"}}`)
	}
	lines := []json.RawMessage{
		mkMsg("m1", strings.Repeat("a", 400)), // 100 tokens
		mkMsg("m2", strings.Repeat("b", 40)),  // 10 tokens
		json.RawMessage(`{"type":"compaction","id":"c1","summary":"` + strings.Repeat("s", 200) + `","firstKeptEntryId":"m2","tokensBefore":110}`),
		mkMsg("m3", strings.Repeat("c", 8000)), // kept 之后的（不计入 c1 估算）
	}
	injectCompactionEstimates(lines)
	var c1 struct {
		Summary              string `json:"summary"`
		FirstKeptEntryID     string `json:"firstKeptEntryId"`
		EstimatedTokensAfter *int64 `json:"estimatedTokensAfter"`
	}
	if err := json.Unmarshal(lines[2], &c1); err != nil {
		t.Fatalf("unmarshal compaction: %v", err)
	}
	if c1.EstimatedTokensAfter == nil {
		t.Fatalf("estimatedTokensAfter not injected")
	}
	// 期望：summary 200/4=50 + kept（m2 起 → c1 前）40/4=10 = 60
	if *c1.EstimatedTokensAfter != 60 {
		t.Fatalf("estimate = %d, want 60 (summary 50 + kept 10)", *c1.EstimatedTokensAfter)
	}
	// 幂等：重复注入不叠加
	injectCompactionEstimates(lines)
	var again struct {
		EstimatedTokensAfter int64 `json:"estimatedTokensAfter"`
	}
	if err := json.Unmarshal(lines[2], &again); err != nil || again.EstimatedTokensAfter != 60 {
		t.Fatalf("idempotency broken: %v %+v", err, again)
	}
}

// kept 边界缺失（窗口受限/旧文件）→ 跳过注入（前端降级显示压缩前规模）。
func TestInjectCompactionEstimatesMissingKeptBoundary(t *testing.T) {
	lines := []json.RawMessage{
		json.RawMessage(`{"type":"message","id":"m1","message":{"role":"user","content":"hi"}}`),
		json.RawMessage(`{"type":"compaction","id":"c1","summary":"s","firstKeptEntryId":"ghost","tokensBefore":5}`),
	}
	injectCompactionEstimates(lines)
	if strings.Contains(string(lines[1]), "estimatedTokensAfter") {
		t.Fatalf("missing boundary must skip injection")
	}
}

// content string 简写形态（pi 新版）参与估算。
func TestMessageCharsEstimateStringContent(t *testing.T) {
	raw := json.RawMessage(`{"type":"message","message":{"role":"assistant","content":"hello world"}}`)
	if got := messageCharsEstimate(raw); got != 11 {
		t.Fatalf("string content chars = %d, want 11", got)
	}
}
