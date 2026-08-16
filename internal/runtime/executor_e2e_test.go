package runtime

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/sunquan/rick/internal/actpath"
	"github.com/sunquan/rick/internal/agent"
)

// End-to-end test: runtime.Execute() execs a mock pi binary that emits a real
// pi `--mode json` event stream (calibrated against pi v0.84.1 output), and the
// full rick-side pipeline — exec → parseStream → AgentSession → actpath — must
// work. Only the LLM is mocked; every line of rick's pi integration runs for real.

// realToolCallStream mirrors the structure of a genuine pi tool-calling turn:
// session header → agent/turn start → user message echo → assistant message →
// tool_execution_start (Read) → tool_execution_end → final assistant text →
// turn_end/agent_end/agent_settled. Field names/shapes match real pi output.
const realToolCallStream = `{"type":"session","version":3,"id":"e2e-sess-001","timestamp":"2026-08-11T12:00:00.000Z","cwd":"/tmp"}
{"type":"agent_start"}
{"type":"turn_start"}
{"type":"message_start","message":{"role":"user","content":[{"type":"text","text":"read the file"}],"timestamp":1786449858483}}
{"type":"message_end","message":{"role":"user","content":[{"type":"text","text":"read the file"}],"timestamp":1786449858483}}
{"type":"message_start","message":{"role":"assistant","content":[]}}
{"type":"tool_execution_start","toolCallId":"call_e2e_1","toolName":"Read","args":{"file_path":"/tmp/e2e_target.txt"}}
{"type":"tool_execution_end","toolCallId":"call_e2e_1","result":"calibration payload","isError":false}
{"type":"message_end","message":{"role":"assistant","content":[{"type":"text","text":"The file contains: calibration payload"}]}}
{"type":"turn_end","toolResults":[]}
{"type":"agent_end","willRetry":false}
{"type":"agent_settled"}
`

// writeMockPiEmitting writes a mock pi binary that prints the given stream to stdout.
func writeMockPiEmitting(t *testing.T, path, stream string) {
	t.Helper()
	// Mock binary prints the fixture stream verbatim (one line per print so it is
	// valid JSONL), regardless of args. This stands in for `pi --mode json <file>`.
	script := "#!/bin/sh\ncat <<'RICK_PI_EOF'\n" + stream + "RICK_PI_EOF\n"
	if err := os.WriteFile(path, []byte(script), 0755); err != nil {
		t.Fatal(err)
	}
}

func TestExecute_EndToEndRealSchema(t *testing.T) {
	tmpDir := t.TempDir()
	mockPi := filepath.Join(tmpDir, "mock_pi")
	writeMockPiEmitting(t, mockPi, realToolCallStream)

	// A prompt file pi would read (content irrelevant — mock ignores it).
	promptFile := filepath.Join(tmpDir, "prompt.md")
	if err := os.WriteFile(promptFile, []byte("# do the task"), 0644); err != nil {
		t.Fatal(err)
	}

	workspace := tmpDir
	exec := NewExecutor(mockPi)

	sess, err := exec.Execute(promptFile, "task_e2e", workspace, "raw_session_e2e.log")
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	if sess == nil {
		t.Fatal("Execute returned nil session")
	}

	// ID() from the session header's `id` field.
	if got := sess.ID(); got != "e2e-sess-001" {
		t.Errorf("ID(): want e2e-sess-001, got %q", got)
	}

	// Duration self-timed (pi emits no duration_ms); agent_settled was present.
	if sess.Duration() <= 0 {
		t.Error("Duration(): expected positive (self-timed)")
	}

	// FinalMessage: assistant turn only — NOT the user's "read the file" echo.
	if got, want := sess.FinalMessage(), "The file contains: calibration payload"; got != want {
		t.Errorf("FinalMessage(): want %q, got %q", want, got)
	}

	// ToolCalls: the Read tool, with input (args) + output (result) + isError=false.
	calls := sess.ToolCalls()
	if len(calls) != 1 {
		t.Fatalf("ToolCalls(): want 1, got %d (%+v)", len(calls), calls)
	}
	c := calls[0]
	if c.Name != "Read" {
		t.Errorf("tool name: want Read, got %q", c.Name)
	}
	if c.Output != "calibration payload" {
		t.Errorf("tool output: want %q, got %q", "calibration payload", c.Output)
	}
	if c.IsError {
		t.Error("tool IsError: want false")
	}
	if c.Input == "" {
		t.Error("tool Input: expected non-empty (args JSON)")
	}
	if c.Line == 0 {
		t.Error("tool Line: expected non-zero (line in raw log)")
	}

	// Raw log written to disk and contains the tool events.
	rawLog := filepath.Join(workspace, "tasks", "task_e2e", "raw_session_e2e.log")
	if sess.RawLogPath() != rawLog {
		t.Errorf("RawLogPath(): want %q, got %q", rawLog, sess.RawLogPath())
	}
	data, err := os.ReadFile(rawLog)
	if err != nil {
		t.Fatalf("read raw log: %v", err)
	}
	if !contains(string(data), "tool_execution_end") {
		t.Error("raw log should contain tool_execution_end line")
	}

	// Error count derives from tool IsError flags (all false here). GetErrorCount
	// is a piSession method (not on the AgentSession interface), so assert the
	// concrete type — same package, direct cast.
	ps, ok := sess.(*piSession)
	if !ok {
		t.Fatalf("expected *piSession, got %T", sess)
	}
	if got := ps.GetErrorCount(); got != 0 {
		t.Errorf("GetErrorCount(): want 0, got %d", got)
	}
}

func TestExecute_ToolErrorPropagated(t *testing.T) {
	// A turn where the tool call fails (isError:true) — GetErrorCount must reflect it.
	stream := `{"type":"session","id":"s2"}
{"type":"tool_execution_start","toolCallId":"c1","toolName":"Write","args":{"path":"/nope"}}
{"type":"tool_execution_end","toolCallId":"c1","result":"permission denied","isError":true}
{"type":"agent_settled"}
`
	tmpDir := t.TempDir()
	mockPi := filepath.Join(tmpDir, "mock_pi")
	writeMockPiEmitting(t, mockPi, stream)

	sess, err := NewExecutor(mockPi).Execute(filepath.Join(tmpDir, "p.md"), "task_err", tmpDir, "raw.log")
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	calls := sess.ToolCalls()
	if len(calls) != 1 {
		t.Fatalf("ToolCalls: want 1, got %d", len(calls))
	}
	if !calls[0].IsError {
		t.Error("expected tool IsError=true")
	}
	ps, ok := sess.(*piSession)
	if !ok {
		t.Fatalf("expected *piSession, got %T", sess)
	}
	if got := ps.GetErrorCount(); got != 1 {
		t.Errorf("GetErrorCount: want 1, got %d", got)
	}
}

func TestExecute_ActPathGeneration(t *testing.T) {
	// Full pipeline: Execute → AgentSession → actpath.Generate writes act-path.md.
	// Proves the parsed session is consumable by the downstream doing-loop consumer.
	tmpDir := t.TempDir()
	mockPi := filepath.Join(tmpDir, "mock_pi")
	writeMockPiEmitting(t, mockPi, realToolCallStream)

	sess, err := NewExecutor(mockPi).Execute(filepath.Join(tmpDir, "p.md"), "task_ap", tmpDir, "raw.log")
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}

	taskDir := filepath.Join(tmpDir, "tasks", "task_ap")
	actPath := filepath.Join(taskDir, "act-path.md")
	if err := actpath.Generate(sess, actPath); err != nil {
		t.Fatalf("actpath.Generate: %v", err)
	}
	body, err := os.ReadFile(actPath)
	if err != nil {
		t.Fatalf("read act-path.md: %v", err)
	}
	s := string(body)
	if !contains(s, "Read") {
		t.Error("act-path.md should list the Read tool call")
	}
	if !contains(s, "e2e-sess-001") {
		t.Error("act-path.md should reference the session id")
	}
}

// contains is a local substring helper (avoids importing strings just for this).
func contains(s, sub string) bool {
	return len(sub) == 0 || (len(s) >= len(sub) && indexOf(s, sub) >= 0)
}

func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}

// satisfy the agent package import (actpath/agent referenced for clarity of pipeline).
var _ agent.AgentSession = (*piSession)(nil)
