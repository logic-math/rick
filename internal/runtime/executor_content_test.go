package runtime

import (
	"encoding/json"
	"testing"
)

// TestPiMessageContentStringCompat guards job_53 Bug #1 fix 3: pi's
// message.content wire format is either an array of content parts or a plain
// string (newer pi events). Both must unmarshal; the string form normalizes
// into a single text part. Before the fix the string form failed the whole
// event line's unmarshal and the supervisor dropped it ("skip bad rpc line",
// 488+192 logged drops → missing messages in the web UI).
func TestPiMessageContentStringCompat(t *testing.T) {
	cases := []struct {
		name    string
		payload string
		want    []piContent
	}{
		{
			name:    "array content (legacy/standard form)",
			payload: `{"role":"assistant","content":[{"type":"text","text":"hello"},{"type":"text","text":"world"}]}`,
			want:    []piContent{{Type: "text", Text: "hello"}, {Type: "text", Text: "world"}},
		},
		{
			name:    "string content (newer pi shorthand)",
			payload: `{"role":"assistant","content":"plain string reply"}`,
			want:    []piContent{{Type: "text", Text: "plain string reply"}},
		},
		{
			name:    "empty string content",
			payload: `{"role":"assistant","content":""}`,
			want:    []piContent{{Type: "text", Text: ""}},
		},
		{
			name:    "null content",
			payload: `{"role":"assistant","content":null}`,
			want:    nil,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var m piMessage
			if err := json.Unmarshal([]byte(tc.payload), &m); err != nil {
				t.Fatalf("unmarshal: %v", err)
			}
			if m.Role != "assistant" {
				t.Fatalf("role = %q, want assistant", m.Role)
			}
			if len(m.Content) != len(tc.want) {
				t.Fatalf("content parts = %d (%+v), want %d", len(m.Content), m.Content, len(tc.want))
			}
			for i, w := range tc.want {
				if m.Content[i] != w {
					t.Fatalf("content[%d] = %+v, want %+v", i, m.Content[i], w)
				}
			}
		})
	}
}

// TestPiMessageContentRejectsGarbage ensures malformed content (neither array
// nor string) still errors loudly instead of silently passing.
func TestPiMessageContentRejectsGarbage(t *testing.T) {
	var m piMessage
	if err := json.Unmarshal([]byte(`{"role":"assistant","content":{"type":"text"}}`), &m); err == nil {
		t.Fatalf("expected error for object content, got %+v", m)
	}
}

// TestRpcEventMessageStringContent verifies the end-to-end shape: an rpc
// event line (message_end/turn_end carry a message) with string content must
// parse via ParseEventLine — this is the exact drop seen 488+192 times in
// the supervisor logs.
func TestRpcEventMessageStringContent(t *testing.T) {
	line := `{"type":"turn_end","message":{"role":"assistant","content":"stringified turn end"}}`
	ev, err := ParseEventLine([]byte(line))
	if err != nil {
		t.Fatalf("ParseEventLine: %v", err)
	}
	if ev.Type != "turn_end" || ev.Message == nil {
		t.Fatalf("ev = %+v, want turn_end with message", ev)
	}
	if len(ev.Message.Content) != 1 || ev.Message.Content[0].Text != "stringified turn end" {
		t.Fatalf("content = %+v, want single text part", ev.Message.Content)
	}
}
