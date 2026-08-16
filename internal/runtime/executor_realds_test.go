package runtime

import (
	"os"
	"strings"
	"testing"
)

// TestParseStream_RealDeepSeekToolCall runs the parser against a genuine
// deepseek-v4-flash tool-calling turn (captured 2026-08-11 via pi --mode json).
// Ground truth: toolName is lowercase "read"; tool_execution_end.result is a
// JSON OBJECT {"content":[{"type":"text","text":...}]}, not a string.
func TestParseStream_RealDeepSeekToolCall(t *testing.T) {
	stream := `{"type":"session","version":3,"id":"real-ds-001","cwd":"/tmp"}
{"type":"agent_start"}
{"type":"turn_start"}
{"type":"message_start","message":{"role":"user","content":[{"type":"text","text":"Read /tmp/pi_calib_target.txt"}]}}
{"type":"message_end","message":{"role":"user","content":[{"type":"text","text":"Read /tmp/pi_calib_target.txt"}]}}
{"type":"message_start","message":{"role":"assistant","content":[]}}
{"type":"tool_execution_start","toolCallId":"call_00_LM4T","toolName":"read","args":{"path":"/tmp/pi_calib_target.txt"}}
{"type":"tool_execution_end","toolCallId":"call_00_LM4T","toolName":"read","result":{"content":[{"type":"text","text":"hello from deepseek calibration\n"}]},"isError":false}
{"type":"message_end","message":{"role":"assistant","content":[{"type":"text","text":"The file contains hello from deepseek calibration"}]}}
{"type":"turn_end","toolResults":[]}
{"type":"agent_end","willRetry":false}
{"type":"agent_settled"}
`
	rawLog := "/tmp/rick_real_raw_test.log"
	defer os.Remove(rawLog)
	sess, err := parseStream(strings.NewReader(stream), rawLog)
	if err != nil {
		t.Fatalf("parseStream: %v", err)
	}
	if sess.ID() != "real-ds-001" {
		t.Errorf("ID: want real-ds-001, got %q", sess.ID())
	}
	calls := sess.ToolCalls()
	if len(calls) != 1 {
		t.Fatalf("toolCalls: want 1, got %d", len(calls))
	}
	c := calls[0]
	if c.Name != "read" {
		t.Errorf("toolName: want read (lowercase), got %q", c.Name)
	}
	if c.IsError {
		t.Error("IsError: want false")
	}
	if c.Output == "" {
		t.Error("Output: expected non-empty (object result unwrapped)")
	}
	if !strings.Contains(c.Output, "hello from deepseek") {
		t.Errorf("Output should contain file text, got %q", c.Output)
	}
	wantPrefix := "The file contains"
	if !strings.HasPrefix(sess.FinalMessage(), wantPrefix) {
		t.Errorf("FinalMessage: want prefix %q, got %q", wantPrefix, sess.FinalMessage())
	}
}
