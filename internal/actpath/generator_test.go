package actpath

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/sunquan/rick/internal/agent"
)

type mockSession struct {
	id               string
	duration         time.Duration
	toolCalls        []agent.ToolCall
	finalMessage     string
	finalMessageLine int
	rawLogPath       string
}

func (m *mockSession) ID() string                  { return m.id }
func (m *mockSession) Duration() time.Duration     { return m.duration }
func (m *mockSession) ToolCalls() []agent.ToolCall { return m.toolCalls }
func (m *mockSession) FinalMessage() string        { return m.finalMessage }
func (m *mockSession) FinalMessageLine() int       { return m.finalMessageLine }
func (m *mockSession) RawLogPath() string          { return m.rawLogPath }

func TestGenerate_Format(t *testing.T) {
	dir := t.TempDir()
	outFile := filepath.Join(dir, "act-path.md")

	sess := &mockSession{
		id:       "test-session-1",
		duration: 5 * time.Second,
		toolCalls: []agent.ToolCall{
			{Name: "bash", Input: "ls", Line: 5, IsError: false},
			{Name: "read", Input: "file.go", Line: 10, IsError: true},
		},
		finalMessage:     "done",
		finalMessageLine: 42,
		rawLogPath:       "/tmp/raw_session_coding.log",
	}

	if err := Generate(sess, outFile); err != nil {
		t.Fatalf("Generate error: %v", err)
	}

	content, err := os.ReadFile(outFile)
	if err != nil {
		t.Fatalf("read output: %v", err)
	}
	s := string(content)

	for _, want := range []string{
		"## 执行摘要",
		"## 行为轨迹",
		"## Agent 最终输出",
		"报错次数: 1",
		"[L",
		"raw_session_coding.log:42",
	} {
		if !strings.Contains(s, want) {
			t.Errorf("output missing %q\ncontent:\n%s", want, s)
		}
	}

	// FinalMessage truncation: long message should be truncated to ≤ 200 chars
	longMsg := strings.Repeat("x", 300)
	sess2 := &mockSession{
		id:               "s2",
		duration:         time.Second,
		toolCalls:        nil,
		finalMessage:     longMsg,
		finalMessageLine: 1,
		rawLogPath:       "/tmp/raw_session_coding.log",
	}
	outFile2 := filepath.Join(dir, "act-path2.md")
	if err := Generate(sess2, outFile2); err != nil {
		t.Fatalf("Generate error (truncation): %v", err)
	}
	content2, _ := os.ReadFile(outFile2)
	idx := strings.Index(string(content2), "## Agent 最终输出")
	if idx == -1 {
		t.Fatal("missing Agent 最终输出 section in truncation test")
	}
	section := string(content2)[idx:]
	xCount := strings.Count(section, "x")
	if xCount > maxFinalMessageLen {
		t.Errorf("FinalMessage not truncated: found %d 'x' chars, want ≤%d", xCount, maxFinalMessageLen)
	}
}

func TestGenerate_EmptyToolCalls(t *testing.T) {
	dir := t.TempDir()
	outFile := filepath.Join(dir, "act-path.md")

	sess := &mockSession{
		id:               "s",
		duration:         time.Second,
		toolCalls:        []agent.ToolCall{},
		finalMessage:     "ok",
		finalMessageLine: 1,
		rawLogPath:       "/tmp/raw_session_coding.log",
	}

	if err := Generate(sess, outFile); err != nil {
		t.Fatalf("Generate error: %v", err)
	}

	content, _ := os.ReadFile(outFile)
	s := string(content)

	if !strings.Contains(s, "## 行为轨迹") {
		t.Error("missing 行为轨迹 section")
	}
	if !strings.Contains(s, "| 行号 | 工具 |") {
		t.Error("missing table header row")
	}
	if strings.Contains(s, "[L") {
		t.Error("unexpected tool call row in empty session")
	}
}

func TestGenerate_CreatesDir(t *testing.T) {
	dir := t.TempDir()
	outFile := filepath.Join(dir, "new_subdir", "nested", "act-path.md")

	sess := &mockSession{
		id:               "s",
		duration:         time.Second,
		toolCalls:        nil,
		finalMessage:     "ok",
		finalMessageLine: 1,
		rawLogPath:       "/tmp/raw_session_coding.log",
	}

	if err := Generate(sess, outFile); err != nil {
		t.Fatalf("Generate error: %v", err)
	}

	if _, err := os.Stat(outFile); err != nil {
		t.Errorf("output file not created: %v", err)
	}
}
