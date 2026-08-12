package piagent

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// The fixtures below encode the pi `--mode json` event schema as captured by the
// research briefs (research-5-N2 / research-7-N4, which read pi source:
// rpc-types.ts, agent-session.ts). The exact wrapper shape must be calibrated
// against real `pi --mode json` output once pi is installed; tests here pin the
// parser's contract so calibration is a localized fix.

// piSessionStream is a representative pi json-mode event stream, calibrated
// against real `pi --mode json` output (session header → agent_start → turn →
// user message echo → assistant message → agent_settled).
const piSessionStream = `{"type":"session","version":3,"id":"sess_abc123","cwd":"/tmp"}
{"type":"agent_start"}
{"type":"turn_start"}
{"type":"message_start","message":{"role":"user","content":[{"type":"text","text":"do the thing"}]}}
{"type":"message_end","message":{"role":"user","content":[{"type":"text","text":"do the thing"}]}}
{"type":"message_start","message":{"role":"assistant","content":[]}}
{"type":"tool_execution_start","toolCallId":"call_1","toolName":"Read","args":{"file_path":"/tmp/foo.go"}}
{"type":"tool_execution_end","toolCallId":"call_1","result":"package main","isError":false}
{"type":"message_end","message":{"role":"assistant","content":[{"type":"text","text":"Task complete."}]}}
{"type":"turn_end","toolResults":[]}
{"type":"agent_end","willRetry":false}
{"type":"agent_settled"}
`

func TestParseStream_BasicSession(t *testing.T) {
	sess := mustParse(t, piSessionStream)

	if sess.ID() != "sess_abc123" {
		t.Errorf("session id: want sess_abc123, got %q", sess.ID())
	}
	if !sess.settled {
		t.Error("expected settled=true after agent_settled")
	}
	if sess.Duration() <= 0 {
		t.Error("expected positive duration after agent_settled")
	}

	calls := sess.ToolCalls()
	if len(calls) != 1 {
		t.Fatalf("tool calls: want 1, got %d", len(calls))
	}
	wantCall := calls[0]
	if wantCall.Name != "Read" {
		t.Errorf("tool name: want Read, got %q", wantCall.Name)
	}
	if wantCall.Output != "package main" {
		t.Errorf("tool output: want %q, got %q", "package main", wantCall.Output)
	}
	if wantCall.IsError {
		t.Error("tool IsError: want false")
	}

	if sess.FinalMessage() != "Task complete." {
		t.Errorf("final message: want %q, got %q", "Task complete.", sess.FinalMessage())
	}
	if sess.FinalMessageLine() == 0 {
		t.Error("expected non-zero final message line")
	}
}

func TestParseStream_ToolError(t *testing.T) {
	stream := `{"type":"session","id":"s1"}
{"type":"tool_execution_start","toolCallId":"c1","toolName":"Write","args":{"path":"/nope"}}
{"type":"tool_execution_end","toolCallId":"c1","result":"permission denied","isError":true}
{"type":"agent_settled"}
`
	sess := mustParse(t, stream)
	calls := sess.ToolCalls()
	if len(calls) != 1 {
		t.Fatalf("tool calls: want 1, got %d", len(calls))
	}
	if !calls[0].IsError {
		t.Error("expected tool IsError=true")
	}
	if got := sess.GetErrorCount(); got != 1 {
		t.Errorf("error count: want 1, got %d", got)
	}
}

func TestParseStream_NoAgentSettled(t *testing.T) {
	// pi crashed / did not terminate: duration should still be derived from start time.
	stream := `{"type":"session","id":"s1"}
{"type":"tool_execution_start","toolCallId":"c1","toolName":"Read","args":{}}
`
	sess := mustParse(t, stream)
	if sess.settled {
		t.Error("did not expect settled without agent_settled")
	}
	if sess.Duration() <= 0 {
		t.Error("expected fallback duration even without agent_settled")
	}
}

func TestParseStream_FinalMessageTruncation(t *testing.T) {
	long := strings.Repeat("x", 500)
	stream := `{"type":"session","id":"s1"}
{"type":"message_end","message":{"role":"assistant","content":[{"type":"text","text":"` + long + `"}]}}
{"type":"agent_settled"}
`
	sess := mustParse(t, stream)
	if got := sess.FinalMessage(); len([]rune(got)) != 200 {
		t.Errorf("final message: want 200 runes, got %d", len([]rune(got)))
	}
}

func TestParseStream_ToolOutputObject(t *testing.T) {
	// tool result may be a JSON object, not a string — parser must fall back to raw.
	stream := `{"type":"session","id":"s1"}
{"type":"tool_execution_start","toolCallId":"c1","toolName":"Bash","args":{"cmd":"ls"}}
{"type":"tool_execution_end","toolCallId":"c1","result":{"stdout":"a.go"},"isError":false}
{"type":"agent_settled"}
`
	sess := mustParse(t, stream)
	calls := sess.ToolCalls()
	if len(calls) != 1 {
		t.Fatalf("tool calls: want 1, got %d", len(calls))
	}
	if calls[0].Output == "" {
		t.Error("expected non-empty output for object result")
	}
}

func TestParseStream_RawLogWritten(t *testing.T) {
	rawLogPath := filepath.Join(t.TempDir(), "raw.log")
	sess := mustParseAt(t, piSessionStream, rawLogPath)

	if sess.RawLogPath() != rawLogPath {
		t.Errorf("raw log path: want %q, got %q", rawLogPath, sess.RawLogPath())
	}
	data, err := os.ReadFile(rawLogPath)
	if err != nil {
		t.Fatalf("read raw log: %v", err)
	}
	if !strings.Contains(string(data), "agent_settled") {
		t.Error("raw log should contain agent_settled line")
	}
}

func TestParseStream_UserMessageNotFinalMessage(t *testing.T) {
	// Regression: pi emits message_end for BOTH the user turn (echoing the prompt)
	// and the assistant turn. The user's text must NOT become FinalMessage.
	stream := `{"type":"session","id":"s1"}
{"type":"message_end","message":{"role":"user","content":[{"type":"text","text":"USER PROMPT TEXT"}]}}
{"type":"message_end","message":{"role":"assistant","content":[{"type":"text","text":"assistant reply"}]}}
{"type":"agent_settled"}
`
	sess := mustParse(t, stream)
	if got := sess.FinalMessage(); got != "assistant reply" {
		t.Errorf("FinalMessage: want %q (assistant), got %q — user message leaked through", "assistant reply", got)
	}
}

func TestParseStream_NonJSONLineSkipped(t *testing.T) {
	stream := `{"type":"session","id":"s1"}
this is not json
{"type":"agent_settled"}
`
	sess := mustParse(t, stream)
	if !sess.settled {
		t.Error("non-json line should be skipped, agent_settled still parsed")
	}
}

// mustParse parses stream into a session via a temp raw log, failing the test on error.
func mustParse(t *testing.T, stream string) *piSession {
	t.Helper()
	return mustParseAt(t, stream, filepath.Join(t.TempDir(), "raw.log"))
}

func mustParseAt(t *testing.T, stream, rawLogPath string) *piSession {
	t.Helper()
	sess, err := parseStream(strings.NewReader(stream), rawLogPath)
	if err != nil {
		t.Fatalf("parseStream: %v", err)
	}
	if sess == nil {
		t.Fatal("parseStream returned nil session")
	}
	// Duration is time.Since(startTime); ensure it is non-zero even on fast machines.
	time.Sleep(time.Millisecond)
	return sess
}
