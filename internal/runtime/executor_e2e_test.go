package runtime

import (
	"os"
	"path/filepath"
	"testing"
)

// End-to-end test: runtime.Run() execs a mock pi binary that emits a real
// pi `--mode json` event stream (calibrated against pi v0.84.1 output), and the
// full rick-side pipeline — exec → parseStream → Trace — must work. Only the
// LLM is mocked; every line of rick's pi integration runs for real.

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

func writeMockPiEmitting(t *testing.T, path, stream string) {
	t.Helper()
	script := "#!/bin/sh\ncat <<'RICK_PI_EOF'\n" + stream + "RICK_PI_EOF\n"
	if err := os.WriteFile(path, []byte(script), 0755); err != nil {
		t.Fatal(err)
	}
}

func TestRun_EndToEndRealSchema(t *testing.T) {
	tmpDir := t.TempDir()
	mockPi := filepath.Join(tmpDir, "mock_pi")
	writeMockPiEmitting(t, mockPi, realToolCallStream)

	promptFile := filepath.Join(tmpDir, "prompt.md")
	if err := os.WriteFile(promptFile, []byte("# do the task"), 0644); err != nil {
		t.Fatal(err)
	}

	sessionID, trace, err := NewPiRuntime(mockPi).Run("", promptFile, nil)
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if sessionID != "e2e-sess-001" {
		t.Errorf("sessionID: want e2e-sess-001, got %q", sessionID)
	}
	if trace == nil {
		t.Fatal("trace is nil")
	}
	if !trace.Settled {
		t.Error("trace.Settled: want true")
	}
	if trace.Duration <= 0 {
		t.Error("trace.Duration: expected positive (self-timed)")
	}
	if trace.FinalMessage != "The file contains: calibration payload" {
		t.Errorf("FinalMessage: want %q, got %q", "The file contains: calibration payload", trace.FinalMessage)
	}

	calls := trace.ToolCalls
	if len(calls) != 1 {
		t.Fatalf("ToolCalls: want 1, got %d (%+v)", len(calls), calls)
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
		t.Error("tool Line: expected non-zero")
	}
}

func TestRun_ToolErrorPropagated(t *testing.T) {
	stream := `{"type":"session","id":"s2"}
{"type":"tool_execution_start","toolCallId":"c1","toolName":"Write","args":{"path":"/nope"}}
{"type":"tool_execution_end","toolCallId":"c1","result":"permission denied","isError":true}
{"type":"agent_settled"}
`
	tmpDir := t.TempDir()
	mockPi := filepath.Join(tmpDir, "mock_pi")
	writeMockPiEmitting(t, mockPi, stream)

	_, trace, err := NewPiRuntime(mockPi).Run("", filepath.Join(tmpDir, "p.md"), nil)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	calls := trace.ToolCalls
	if len(calls) != 1 {
		t.Fatalf("ToolCalls: want 1, got %d", len(calls))
	}
	if !calls[0].IsError {
		t.Error("expected tool IsError=true")
	}
}
