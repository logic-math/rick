package claudecode

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
)

func TestExecute_ParseNDJSON(t *testing.T) {
	lines := []string{
		`{"type":"system","session_id":"mock-session-001"}`,
		`{"type":"assistant","message":{"content":[{"type":"tool_use","id":"t1","name":"Bash","input":{"command":"ls"}}]},"session_id":"mock-session-001"}`,
		`{"type":"user","message":{"content":[{"type":"tool_result","tool_use_id":"t1","content":"file1.txt","is_error":false}]},"session_id":"mock-session-001"}`,
		`{"type":"assistant","message":{"content":[{"type":"text","text":"done."}]},"session_id":"mock-session-001"}`,
		`{"type":"result","is_error":false,"duration_ms":10864,"session_id":"mock-session-001"}`,
	}
	ndjson := strings.Join(lines, "\n")

	tmpDir := t.TempDir()
	rawLogPath := tmpDir + "/raw_session.log"

	sess, err := parseStream(strings.NewReader(ndjson), rawLogPath)
	if err != nil {
		t.Fatalf("parseStream error: %v", err)
	}

	if sess.ID() != "mock-session-001" {
		t.Errorf("sessionID: got %q, want %q", sess.ID(), "mock-session-001")
	}
	if len(sess.ToolCalls()) != 1 {
		t.Fatalf("ToolCalls length: got %d, want 1", len(sess.ToolCalls()))
	}
	if sess.ToolCalls()[0].Name != "Bash" {
		t.Errorf("ToolCall name: got %q, want %q", sess.ToolCalls()[0].Name, "Bash")
	}
	if sess.ToolCalls()[0].IsError {
		t.Error("ToolCall IsError: got true, want false")
	}
	if sess.FinalMessage() != "done." {
		t.Errorf("FinalMessage: got %q, want %q", sess.FinalMessage(), "done.")
	}
	if len([]rune(sess.FinalMessage())) > 200 {
		t.Error("FinalMessage exceeds 200 chars")
	}
	if sess.FinalMessageLine() != 4 {
		t.Errorf("FinalMessageLine: got %d, want 4", sess.FinalMessageLine())
	}

	// raw_session.log: 5 lines, each valid JSON
	data, err := os.ReadFile(rawLogPath)
	if err != nil {
		t.Fatalf("read raw_session.log: %v", err)
	}
	fileLines := strings.Split(strings.TrimRight(string(data), "\n"), "\n")
	if len(fileLines) != 5 {
		t.Errorf("raw_session.log lines: got %d, want 5", len(fileLines))
	}
	for i, l := range fileLines {
		var m map[string]interface{}
		if err := json.Unmarshal([]byte(l), &m); err != nil {
			t.Errorf("raw_session.log line %d not valid JSON: %v", i+1, err)
		}
	}
}

func TestExecute_SkipNonJSON(t *testing.T) {
	lines := []string{
		`{"type":"system","session_id":"mock-session-002"}`,
		`{"type":"assistant","message":{"content":[{"type":"tool_use","id":"t1","name":"Read","input":{"path":"file.go"}}]},"session_id":"mock-session-002"}`,
		`not json`,
		`{"type":"user","message":{"content":[{"type":"tool_result","tool_use_id":"t1","content":"ok","is_error":false}]},"session_id":"mock-session-002"}`,
		`{"type":"result","is_error":false,"duration_ms":5000,"session_id":"mock-session-002"}`,
	}
	ndjson := strings.Join(lines, "\n")

	tmpDir := t.TempDir()
	rawLogPath := tmpDir + "/raw_session.log"

	sess, err := parseStream(strings.NewReader(ndjson), rawLogPath)
	if err != nil {
		t.Fatalf("parseStream error: %v", err)
	}

	if sess.ID() != "mock-session-002" {
		t.Errorf("sessionID: got %q, want %q", sess.ID(), "mock-session-002")
	}
	if len(sess.ToolCalls()) != 1 {
		t.Fatalf("ToolCalls length: got %d, want 1", len(sess.ToolCalls()))
	}

	// raw_session.log exists and contains the non-JSON line
	data, err := os.ReadFile(rawLogPath)
	if err != nil {
		t.Fatalf("read raw_session.log: %v", err)
	}
	if len(data) == 0 {
		t.Error("raw_session.log is empty")
	}
	if !strings.Contains(string(data), "not json") {
		t.Error("raw_session.log should contain the non-JSON line")
	}
}
