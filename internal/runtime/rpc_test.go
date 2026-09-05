package runtime

import (
	"encoding/json"
	"strconv"
	"strings"
	"testing"
)

// The fixtures below encode the pi `--mode rpc` wire protocol as documented in
// pi's docs/rpc.md (verified against pi v0.84.x). Command-construction tests
// pin exact JSON bytes (field names, nesting, LF termination); event-parsing
// tests cover the full event vocabulary.

// --- command construction ---

// decodeCommandJSON parses a built command line into a generic map, after
// asserting shape invariants (single line, trailing LF, valid JSON).
func decodeCommandJSON(t *testing.T, line []byte) map[string]any {
	t.Helper()
	s := string(line)
	if !strings.HasSuffix(s, "\n") {
		t.Fatalf("command must end with LF, got %q", s)
	}
	if strings.Count(strings.TrimSuffix(s, "\n"), "\n") != 0 {
		t.Fatalf("command must be exactly one line, got %q", s)
	}
	var m map[string]any
	if err := json.Unmarshal(line, &m); err != nil {
		t.Fatalf("command is not valid JSON: %v\n%s", err, s)
	}
	return m
}

func TestRpcClient_Prompt(t *testing.T) {
	c := NewRpcClient()
	line, err := c.Prompt("Hello, world!", "")
	if err != nil {
		t.Fatalf("Prompt: %v", err)
	}
	m := decodeCommandJSON(t, line)
	if m["type"] != "prompt" {
		t.Errorf("type: want prompt, got %v", m["type"])
	}
	if m["message"] != "Hello, world!" {
		t.Errorf("message: want %q, got %v", "Hello, world!", m["message"])
	}
	if _, ok := m["streamingBehavior"]; ok {
		t.Error("streamingBehavior must be omitted when empty")
	}
	if id, ok := m["id"].(string); !ok || !strings.HasPrefix(id, "req-") {
		t.Errorf("id: want req-* string, got %v", m["id"])
	}
}

func TestRpcClient_PromptWithStreamingBehavior(t *testing.T) {
	c := NewRpcClient()
	line, err := c.Prompt("New instruction", "steer")
	if err != nil {
		t.Fatalf("Prompt: %v", err)
	}
	m := decodeCommandJSON(t, line)
	if m["streamingBehavior"] != "steer" {
		t.Errorf("streamingBehavior: want steer, got %v", m["streamingBehavior"])
	}

	if _, err := c.Prompt("x", "bogus"); err == nil {
		t.Error("invalid streamingBehavior should error")
	}
	if _, err := c.Prompt("", "steer"); err == nil {
		t.Error("empty message should error")
	}
}

func TestRpcClient_SteerAndFollowUp(t *testing.T) {
	c := NewRpcClient()

	line, err := c.Steer("Stop and do this instead")
	if err != nil {
		t.Fatalf("Steer: %v", err)
	}
	m := decodeCommandJSON(t, line)
	if m["type"] != "steer" || m["message"] != "Stop and do this instead" {
		t.Errorf("steer: %v", m)
	}
	if _, ok := m["streamingBehavior"]; ok {
		t.Error("steer must not carry streamingBehavior (it is the queue semantic itself)")
	}

	line, err = c.FollowUp("After you're done, also do this")
	if err != nil {
		t.Fatalf("FollowUp: %v", err)
	}
	m = decodeCommandJSON(t, line)
	if m["type"] != "follow_up" || m["message"] != "After you're done, also do this" {
		t.Errorf("follow_up: %v", m)
	}
}

func TestRpcClient_NoPayloadCommands(t *testing.T) {
	c := NewRpcClient()
	builders := map[string]func() ([]byte, error){
		"abort":                     c.Abort,
		"get_state":                 c.GetState,
		"get_messages":              c.GetMessages,
		"get_session_stats":         c.GetSessionStats,
		"cycle_model":               c.CycleModel,
		"get_available_models":      c.GetAvailableModels,
		"get_available_thinking_levels": c.GetAvailableThinkingLevels,
	}
	for want, build := range builders {
		line := mustCmd(build())
		m := decodeCommandJSON(t, line)
		if m["type"] != want {
			t.Errorf("%s type: want %q, got %v", want, want, m["type"])
		}
		if len(m) != 2 { // type + id only
			t.Errorf("%s: want exactly type+id fields, got %d: %v", want, len(m), m)
		}
	}
}

func TestRpcClient_SetModel(t *testing.T) {
	c := NewRpcClient()
	line := mustCmd(c.SetModel("deepseek", "deepseek-v4-flash"))
	m := decodeCommandJSON(t, line)
	if m["type"] != "set_model" {
		t.Errorf("type: want set_model, got %v", m["type"])
	}
	if m["provider"] != "deepseek" {
		t.Errorf("provider: want deepseek, got %v", m["provider"])
	}
	if m["modelId"] != "deepseek-v4-flash" {
		t.Errorf("modelId: want deepseek-v4-flash, got %v", m["modelId"])
	}
	if len(m) != 4 { // type + provider + modelId + id
		t.Errorf("want exactly 4 fields, got %d: %v", len(m), m)
	}
}

func TestRpcClient_SetModelValidation(t *testing.T) {
	c := NewRpcClient()
	if _, err := c.SetModel("", "model-x"); err == nil {
		t.Error("SetModel with empty provider must error")
	}
	if _, err := c.SetModel("deepseek", ""); err == nil {
		t.Error("SetModel with empty modelID must error")
	}
}

func TestRpcClient_SetThinkingLevel(t *testing.T) {
	c := NewRpcClient()
	line := mustCmd(c.SetThinkingLevel("high"))
	m := decodeCommandJSON(t, line)
	if m["type"] != "set_thinking_level" {
		t.Errorf("type: want set_thinking_level, got %v", m["type"])
	}
	if m["level"] != "high" {
		t.Errorf("level: want high, got %v", m["level"])
	}
	if len(m) != 3 { // type + level + id
		t.Errorf("want exactly 3 fields, got %d: %v", len(m), m)
	}
	if _, err := c.SetThinkingLevel(""); err == nil {
		t.Error("SetThinkingLevel with empty level must error")
	}
}

func TestRpcClient_NewSession(t *testing.T) {
	c := NewRpcClient()

	line, err := c.NewSession("")
	if err != nil {
		t.Fatalf("NewSession: %v", err)
	}
	m := decodeCommandJSON(t, line)
	if m["type"] != "new_session" || len(m) != 2 {
		t.Errorf("bare new_session: %v", m)
	}

	line, err = c.NewSession("/path/to/parent.jsonl")
	if err != nil {
		t.Fatalf("NewSession: %v", err)
	}
	m = decodeCommandJSON(t, line)
	if m["type"] != "new_session" || m["parentSession"] != "/path/to/parent.jsonl" {
		t.Errorf("new_session with parent: %v", m)
	}
}

func TestRpcClient_GetEntries(t *testing.T) {
	c := NewRpcClient()

	line, err := c.GetEntries("")
	if err != nil {
		t.Fatalf("GetEntries: %v", err)
	}
	m := decodeCommandJSON(t, line)
	if m["type"] != "get_entries" || len(m) != 2 {
		t.Errorf("bare get_entries: %v", m)
	}

	line, err = c.GetEntries("abc123")
	if err != nil {
		t.Fatalf("GetEntries: %v", err)
	}
	m = decodeCommandJSON(t, line)
	if m["since"] != "abc123" {
		t.Errorf("since: want abc123, got %v", m["since"])
	}
}

func TestRpcClient_SwitchSession(t *testing.T) {
	c := NewRpcClient()
	line, err := c.SwitchSession("/tmp/sess.jsonl")
	if err != nil {
		t.Fatalf("SwitchSession: %v", err)
	}
	m := decodeCommandJSON(t, line)
	if m["type"] != "switch_session" || m["sessionPath"] != "/tmp/sess.jsonl" {
		t.Errorf("switch_session: %v", m)
	}
	if _, err := c.SwitchSession(""); err == nil {
		t.Error("empty sessionPath should error")
	}
}

func TestRpcClient_SetSessionName(t *testing.T) {
	c := NewRpcClient()
	line, err := c.SetSessionName("my-feature-work")
	if err != nil {
		t.Fatalf("SetSessionName: %v", err)
	}
	m := decodeCommandJSON(t, line)
	if m["type"] != "set_session_name" || m["name"] != "my-feature-work" {
		t.Errorf("set_session_name: %v", m)
	}
	if _, err := c.SetSessionName(""); err == nil {
		t.Error("empty name should error")
	}
}

func TestRpcClient_CommandIDsAreUniqueAndMonotonic(t *testing.T) {
	c := NewRpcClient()
	var prev int64
	for i := 0; i < 5; i++ {
		line := mustCmd(c.GetState())
		m := decodeCommandJSON(t, line)
		id, ok := m["id"].(string)
		if !ok || !strings.HasPrefix(id, "req-") {
			t.Fatalf("id shape: %v", m["id"])
		}
		n, err := strconv.ParseInt(strings.TrimPrefix(id, "req-"), 10, 64)
		if err != nil {
			t.Fatalf("id not numeric: %v", id)
		}
		if i > 0 && n != prev+1 {
			t.Fatalf("ids not monotonic: %d then %d", prev, n)
		}
		prev = n
	}
}

func TestRpcClient_BuildCommandGeneric(t *testing.T) {
	c := NewRpcClient()

	line, err := c.BuildCommand("bash", map[string]any{"command": "ls -la"})
	if err != nil {
		t.Fatalf("BuildCommand: %v", err)
	}
	m := decodeCommandJSON(t, line)
	if m["type"] != "bash" || m["command"] != "ls -la" {
		t.Errorf("bash: %v", m)
	}

	if _, err := c.BuildCommand("", nil); err == nil {
		t.Error("empty type should error")
	}
	if _, err := c.BuildCommand("prompt", map[string]any{"type": "x"}); err == nil {
		t.Error("payload key 'type' should be rejected")
	}
	if _, err := c.BuildCommand("prompt", map[string]any{"id": "x"}); err == nil {
		t.Error("payload key 'id' should be rejected")
	}

	// Values of every JSON kind round-trip.
	line, err = c.BuildCommand("probe", map[string]any{
		"s": "v", "i": 42, "f": 1.5, "b": true, "n": nil,
	})
	if err != nil {
		t.Fatalf("BuildCommand mixed: %v", err)
	}
	m = decodeCommandJSON(t, line)
	if m["s"] != "v" || m["i"] != float64(42) || m["f"] != 1.5 || m["b"] != true || m["n"] != nil {
		t.Errorf("mixed payload: %v", m)
	}
}

func TestRpcClient_CommandsJSONEscapeMessages(t *testing.T) {
	c := NewRpcClient()
	line, err := c.Prompt("line1\nline2 \"quoted\" 中文", "")
	if err != nil {
		t.Fatalf("Prompt: %v", err)
	}
	if strings.Count(string(line), "\n") != 1 {
		t.Fatalf("embedded newline must be escaped, got %d LFs: %q", strings.Count(string(line), "\n"), line)
	}
	m := decodeCommandJSON(t, line)
	if m["message"] != "line1\nline2 \"quoted\" 中文" {
		t.Errorf("escaped message round-trip: %v", m["message"])
	}
}

// --- extension_ui_response ---

func TestRpcClient_SendUIResponse(t *testing.T) {
	c := NewRpcClient()

	// Value response (select/input/editor).
	line, err := c.SendUIResponse("uuid-1", UIResponseValue, "Allow")
	if err != nil {
		t.Fatalf("SendUIResponse value: %v", err)
	}
	m := decodeCommandJSON(t, line)
	if m["type"] != "extension_ui_response" || m["id"] != "uuid-1" || m["value"] != "Allow" {
		t.Errorf("value response: %v", m)
	}
	if _, ok := m["confirmed"]; ok {
		t.Error("value response must not carry confirmed")
	}

	// Confirm response.
	line, err = c.SendUIResponse("uuid-2", UIResponseConfirm, true)
	if err != nil {
		t.Fatalf("SendUIResponse confirm: %v", err)
	}
	m = decodeCommandJSON(t, line)
	if m["confirmed"] != true {
		t.Errorf("confirm response: %v", m)
	}
	if _, ok := m["value"]; ok {
		t.Error("confirm response must not carry value")
	}

	// Cancelled response.
	line, err = c.SendUIResponse("uuid-3", UIResponseCancelled, nil)
	if err != nil {
		t.Fatalf("SendUIResponse cancelled: %v", err)
	}
	m = decodeCommandJSON(t, line)
	if m["cancelled"] != true {
		t.Errorf("cancelled response: %v", m)
	}

	// Errors.
	if _, err := c.SendUIResponse("", UIResponseValue, "x"); err == nil {
		t.Error("empty id should error")
	}
	if _, err := c.SendUIResponse("u", UIResponseConfirm, "not-a-bool"); err == nil {
		t.Error("non-bool confirm value should error")
	}
}

// --- event parsing ---

func TestRpcParseEventLine_Response(t *testing.T) {
	ev, err := ParseEventLine([]byte(`{"id":"req-1","type":"response","command":"prompt","success":true}`))
	if err != nil {
		t.Fatalf("ParseEventLine: %v", err)
	}
	if !ev.IsResponse() {
		t.Error("IsResponse: want true")
	}
	if ev.Command != "prompt" || !ev.Success || ev.ID != "req-1" {
		t.Errorf("response fields: %+v", ev)
	}
}

func TestRpcParseEventLine_ResponseError(t *testing.T) {
	ev, err := ParseEventLine([]byte(`{"type":"response","command":"set_model","success":false,"error":"Model not found: invalid/model"}`))
	if err != nil {
		t.Fatalf("ParseEventLine: %v", err)
	}
	if ev.Success {
		t.Error("success: want false")
	}
	if ev.Error != "Model not found: invalid/model" {
		t.Errorf("error field: %q", ev.Error)
	}
}

func TestRpcParseEventLine_GetStateResponseData(t *testing.T) {
	line := `{"type":"response","command":"get_state","success":true,"data":{"sessionId":"abc123","sessionFile":"/p/s.jsonl","isStreaming":false,"isCompacting":false,"thinkingLevel":"medium"}}`
	ev, err := ParseEventLine([]byte(line))
	if err != nil {
		t.Fatalf("ParseEventLine: %v", err)
	}
	var data map[string]any
	if err := json.Unmarshal(ev.Data, &data); err != nil {
		t.Fatalf("data not valid JSON: %v", err)
	}
	if data["sessionId"] != "abc123" || data["isStreaming"] != false {
		t.Errorf("get_state data: %v", data)
	}
}

func TestRpcParseEventLine_AgentLifecycle(t *testing.T) {
	for _, line := range []string{
		`{"type":"agent_start"}`,
		`{"type":"agent_end","messages":[],"willRetry":false}`,
		`{"type":"agent_settled"}`,
		`{"type":"turn_start"}`,
		`{"type":"turn_end","message":{"role":"assistant","content":[]},"toolResults":[]}`,
	} {
		ev, err := ParseEventLine([]byte(line))
		if err != nil {
			t.Fatalf("ParseEventLine(%s): %v", line, err)
		}
		if !rpcEventTypes[ev.Type] {
			t.Errorf("type %q not in event vocabulary", ev.Type)
		}
	}
}

func TestRpcParseEventLine_MessageEnd(t *testing.T) {
	line := `{"type":"message_end","message":{"role":"assistant","content":[{"type":"text","text":"Hello! How can I help?"}],"timestamp":1733234567890}}`
	ev, err := ParseEventLine([]byte(line))
	if err != nil {
		t.Fatalf("ParseEventLine: %v", err)
	}
	if ev.Message == nil || ev.Message.Role != "assistant" {
		t.Fatalf("message: %+v", ev.Message)
	}
	if len(ev.Message.Content) != 1 || ev.Message.Content[0].Text != "Hello! How can I help?" {
		t.Errorf("content: %+v", ev.Message.Content)
	}
}

func TestRpcParseEventLine_MessageUpdateDelta(t *testing.T) {
	line := `{"type":"message_update","usage":{"input":100,"output":1},"assistantMessageEvent":{"type":"text_delta","contentIndex":0,"delta":"Hello "}}`
	ev, err := ParseEventLine([]byte(line))
	if err != nil {
		t.Fatalf("ParseEventLine: %v", err)
	}
	// delta lives in Raw (full-line preservation) — the web layer relays it verbatim.
	var m map[string]any
	if err := json.Unmarshal(ev.Raw, &m); err != nil {
		t.Fatalf("raw preservation broken: %v", err)
	}
	ame, _ := m["assistantMessageEvent"].(map[string]any)
	if ame == nil || ame["delta"] != "Hello " {
		t.Errorf("assistantMessageEvent: %v", m["assistantMessageEvent"])
	}
}

func TestRpcParseEventLine_ToolExecution(t *testing.T) {
	start, err := ParseEventLine([]byte(`{"type":"tool_execution_start","toolCallId":"call_abc123","toolName":"bash","args":{"command":"ls -la"}}`))
	if err != nil {
		t.Fatalf("start: %v", err)
	}
	if start.ToolCallID != "call_abc123" || start.ToolName != "bash" {
		t.Errorf("start fields: %+v", start)
	}
	var args map[string]any
	if err := json.Unmarshal(start.Args, &args); err != nil || args["command"] != "ls -la" {
		t.Errorf("args: %v (%v)", start.Args, err)
	}

	end, err := ParseEventLine([]byte(`{"type":"tool_execution_end","toolCallId":"call_abc123","toolName":"bash","result":{"content":[{"type":"text","text":"total 48"}],"details":{}},"isError":false}`))
	if err != nil {
		t.Fatalf("end: %v", err)
	}
	if end.IsError {
		t.Error("isError: want false")
	}
	var result map[string]any
	if err := json.Unmarshal(end.Result, &result); err != nil {
		t.Fatalf("result: %v", err)
	}
	if result["content"] == nil {
		t.Errorf("result content: %v", result)
	}
}

func TestRpcParseEventLine_BashExecutionUpdate(t *testing.T) {
	ev, err := ParseEventLine([]byte(`{"type":"bash_execution_update","id":"req-1","delta":"total 48\n"}`))
	if err != nil {
		t.Fatalf("ParseEventLine: %v", err)
	}
	if ev.ID != "req-1" {
		t.Errorf("id (bash correlation): %q", ev.ID)
	}
}

func TestRpcParseEventLine_ExtensionUIRequest(t *testing.T) {
	line := `{"type":"extension_ui_request","id":"uuid-1","method":"select","title":"Allow dangerous command?","options":["Allow","Block"],"timeout":10000}`
	ev, err := ParseEventLine([]byte(line))
	if err != nil {
		t.Fatalf("ParseEventLine: %v", err)
	}
	if !ev.IsExtensionUIRequest() {
		t.Error("IsExtensionUIRequest: want true")
	}
	if ev.ID != "uuid-1" {
		t.Errorf("request id: %q", ev.ID)
	}
	var m map[string]any
	if err := json.Unmarshal(ev.Raw, &m); err != nil {
		t.Fatalf("method/options preserved: %v", err)
	}
	if m["method"] != "select" {
		t.Errorf("method: %v", m["method"])
	}
	opts, ok := m["options"].([]any)
	if !ok || len(opts) != 2 || opts[0] != "Allow" {
		t.Errorf("options: %v", m["options"])
	}
}

func TestRpcParseEventLine_CompactionAndRetry(t *testing.T) {
	for _, line := range []string{
		`{"type":"compaction_start","reason":"threshold"}`,
		`{"type":"compaction_end","reason":"threshold","result":{"summary":"...","firstKeptEntryId":"abc"},"aborted":false,"willRetry":false}`,
		`{"type":"auto_retry_start","attempt":1,"maxAttempts":3,"delayMs":2000,"errorMessage":"529"}`,
		`{"type":"auto_retry_end","success":true,"attempt":2}`,
		`{"type":"queue_update","steering":["a"],"followUp":["b"]}`,
		`{"type":"extension_error","extensionPath":"/x.ts","event":"tool_call","error":"boom"}`,
	} {
		if _, err := ParseEventLine([]byte(line)); err != nil {
			t.Fatalf("ParseEventLine(%s): %v", line, err)
		}
	}
}

func TestRpcParseEventLine_ToleratesTrailingCR(t *testing.T) {
	ev, err := ParseEventLine([]byte("{\"type\":\"agent_settled\"}\r"))
	if err != nil {
		t.Fatalf("CRLF tolerance: %v", err)
	}
	if ev.Type != "agent_settled" {
		t.Errorf("type: %q", ev.Type)
	}
}

func TestRpcParseEventLine_Errors(t *testing.T) {
	cases := []struct {
		name string
		line string
	}{
		{"empty", ""},
		{"whitespace", "   "},
		{"invalid json", `{"type":`},
		{"not an object", `["agent_start"]`},
		{"missing type", `{"sessionId":"s1"}`},
	}
	for _, tc := range cases {
		if _, err := ParseEventLine([]byte(tc.line)); err == nil {
			t.Errorf("%s: expected error, got none", tc.name)
		}
	}
}

func TestRpcEventVocabularyCoversRpcDoc(t *testing.T) {
	// Every documented event type must parse and must be in the vocabulary.
	for typ := range rpcEventTypes {
		line := `{"type":"` + typ + `"}`
		ev, err := ParseEventLine([]byte(line))
		if err != nil {
			t.Fatalf("vocab type %q failed to parse: %v", typ, err)
		}
		if ev.Type != typ {
			t.Errorf("type round-trip: %q != %q", ev.Type, typ)
		}
	}
}

// mustCmd panics if a command builder errors, so it can consume the builder's
// two return values directly (Go only spreads multi-value returns into a
// single-argument call position).
func mustCmd(line []byte, err error) []byte {
	if err != nil {
		panic(err)
	}
	return line
}
