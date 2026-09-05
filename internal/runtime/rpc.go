// This file implements the pi `--mode rpc` JSONL protocol client — the wire
// layer rick's web sessions speak to drive a long-lived pi subprocess.
//
// Scope: pure protocol (command construction + event line parsing). Process
// management (spawn, pipes, heartbeat, kill) lives in supervisor.go (task4).
//
// Protocol reference: pi docs/rpc.md (@earendil-works/pi-coding-agent), verified
// against pi v0.84.x. Key wire facts:
//
//   - Commands are JSON objects written to the subprocess stdin, one per line
//     (strict JSONL, LF is the ONLY record delimiter — Node's readline is NOT
//     protocol-compliant because it also splits on U+2028/U+2029, which are
//     valid inside JSON strings; Go's bufio.Scanner splits on LF only and is
//     compliant).
//   - Every command accepts an optional "id" for request/response correlation;
//     the matching response echoes it. rpc.md shows "req-1"-style string ids.
//   - Responses have type "response" with "command" + "success" (+ optional
//     "data"/"error" + echoed "id").
//   - Events are JSON lines on stdout: agent_*, turn_*, message_*, tool_execution_*,
//     bash_execution_update, queue_update, compaction_*, auto_retry_*,
//     summarization_retry_*, extension_error, extension_ui_request.
//   - extension_ui_request (dialog methods select/confirm/input/editor) expects
//     an extension_ui_response written back to stdin with the matching id.
//
// ⚠️ pi session flag semantics (bugs.md, job_30): `--session <path|id>` LOADS
// an existing session ("No session found matching" if absent); `--session-id
// <uuid>` CREATES a new session with that id. The spawn layer (task4
// supervisor) must use --session-id for new web sessions and --session for
// resume. This file never spawns processes, but the distinction is recorded
// here because the protocol layer is where consumers look for pi wire facts.

package runtime

import (
	"encoding/json"
	"fmt"
	"strings"
	"sync/atomic"
)

// --- RPC command construction ---

// rpcCommandID is the shared auto-incrementing request id. rpc.md examples use
// string ids ("req-1"); a per-client counter keeps correlation simple and the
// id optional everywhere (commands without "id" still get responses, just
// without correlation). The counter is process-global so multiple RpcClients
// never emit colliding ids within one rick process.
var rpcCommandID atomic.Int64

// RpcClient builds pi `--mode rpc` JSONL commands. It is a pure builder: it
// holds no pipes and spawns nothing — the supervisor writes the returned bytes
// to the subprocess stdin. Every method returns one complete JSON line (with
// trailing LF), ready to be written verbatim.
//
// Usage:
//
//	c := runtime.NewRpcClient()
//	line, err := c.Prompt("hello")        // {"type":"prompt","message":"hello","id":"req-1"}\n
//	stdin.Write(line)
type RpcClient struct {
	// prefix is prepended to the auto id (e.g. "req-"); empty means bare ints.
	prefix string
}

// NewRpcClient returns a command builder using the default "req-" id prefix
// (mirrors rpc.md's own examples).
func NewRpcClient() *RpcClient {
	return &RpcClient{prefix: "req-"}
}

// nextID allocates the next correlation id ("req-1", "req-2", ...).
func (c *RpcClient) nextID() string {
	return fmt.Sprintf("%s%d", c.prefix, rpcCommandID.Add(1))
}

// BuildCommand is the generic command constructor: it sets "type", merges
// payload fields (string / number / bool / nil values are supported), and
// attaches the auto-incremented correlation id. Fields are emitted with type
// first, then payload keys in sorted order for deterministic output, then id
// last. Returns the JSON line terminated by LF.
//
// A payload key named "type" or "id" is rejected — callers must not fight the
// envelope fields.
func (c *RpcClient) BuildCommand(typ string, payload map[string]any) ([]byte, error) {
	if typ == "" {
		return nil, fmt.Errorf("rpc: command type must not be empty")
	}
	if _, ok := payload["type"]; ok {
		return nil, fmt.Errorf("rpc: payload key %q collides with envelope field", "type")
	}
	if _, ok := payload["id"]; ok {
		return nil, fmt.Errorf("rpc: payload key %q collides with envelope field", "id")
	}

	// Deterministic field order: type, sorted payload keys, id.
	buf := &strings.Builder{}
	buf.WriteString(`{"type":`)
	writeJSONString(buf, typ)
	keys := make([]string, 0, len(payload))
	for k := range payload {
		keys = append(keys, k)
	}
	// insertion-independent ordering: simple lexicographic sort
	for i := 1; i < len(keys); i++ {
		for j := i; j > 0 && keys[j] < keys[j-1]; j-- {
			keys[j], keys[j-1] = keys[j-1], keys[j]
		}
	}
	for _, k := range keys {
		v, err := json.Marshal(payload[k])
		if err != nil {
			return nil, fmt.Errorf("rpc: marshal payload %q: %w", k, err)
		}
		buf.WriteString(",")
		writeJSONString(buf, k)
		buf.WriteString(":")
		buf.Write(v)
	}
	buf.WriteString(`,"id":`)
	writeJSONString(buf, c.nextID())
	buf.WriteString("}\n")
	return []byte(buf.String()), nil
}

// buildWithEnvelope is BuildCommand's internal twin: identical serialization
// (type first, sorted payload keys, id last) but the "id" payload field is
// honored verbatim instead of being rejected/auto-generated. Only the
// extension_ui_response path uses it — every other command goes through
// BuildCommand so the envelope field stays reserved.
func (c *RpcClient) buildWithEnvelope(typ string, payload map[string]any) ([]byte, error) {
	if typ == "" {
		return nil, fmt.Errorf("rpc: command type must not be empty")
	}
	buf := &strings.Builder{}
	buf.WriteString(`{"type":`)
	writeJSONString(buf, typ)
	keys := make([]string, 0, len(payload))
	for k := range payload {
		keys = append(keys, k)
	}
	// insertion-independent ordering: simple lexicographic sort
	for i := 1; i < len(keys); i++ {
		for j := i; j > 0 && keys[j] < keys[j-1]; j-- {
			keys[j], keys[j-1] = keys[j-1], keys[j]
		}
	}
	for _, k := range keys {
		v, err := json.Marshal(payload[k])
		if err != nil {
			return nil, fmt.Errorf("rpc: marshal payload %q: %w", k, err)
		}
		buf.WriteString(",")
		writeJSONString(buf, k)
		buf.WriteString(":")
		buf.Write(v)
	}
	buf.WriteString("}\n")
	return []byte(buf.String()), nil
}

// writeJSONString marshals exactly one JSON string (with quotes) into buf.
func writeJSONString(buf *strings.Builder, s string) {
	b, _ := json.Marshal(s) // json.Marshal on a string never fails
	buf.Write(b)
}

// buildNoPayload emits a command that carries only its type + id.
func (c *RpcClient) buildNoPayload(typ string) ([]byte, error) {
	return c.BuildCommand(typ, nil)
}

// Prompt builds a `prompt` command. streamingBehavior is optional and only
// meaningful while the agent is already streaming: "steer" queues the message
// for delivery after the current assistant turn's tool calls; "followUp"
// waits until the agent stops. Empty string omits the field (pi rejects
// mid-stream prompts without it — callers that need queue semantics should
// use Steer/FollowUp directly).
func (c *RpcClient) Prompt(msg, streamingBehavior string) ([]byte, error) {
	return c.promptLike("prompt", msg, streamingBehavior)
}

// Steer builds a `steer` command: queue a steering message while the agent
// runs (delivered after the current turn's tool calls, before the next LLM
// call). Extension commands are not allowed via steer (use Prompt).
func (c *RpcClient) Steer(msg string) ([]byte, error) {
	return c.promptLike("steer", msg, "")
}

// FollowUp builds a `follow_up` command: queue a message to be processed
// after the agent finishes (no tool calls or steering messages pending).
func (c *RpcClient) FollowUp(msg string) ([]byte, error) {
	return c.promptLike("follow_up", msg, "")
}

// promptLike factors the shared message-carrier shape of
// prompt/steer/follow_up: {"type":…,"message":…[, "streamingBehavior":…]}.
func (c *RpcClient) promptLike(typ, msg, streamingBehavior string) ([]byte, error) {
	if msg == "" {
		return nil, fmt.Errorf("rpc: %s requires a non-empty message", typ)
	}
	payload := map[string]any{"message": msg}
	switch streamingBehavior {
	case "":
		// omit
	case "steer", "followUp":
		payload["streamingBehavior"] = streamingBehavior
	default:
		return nil, fmt.Errorf("rpc: invalid streamingBehavior %q (want \"steer\" or \"followUp\")", streamingBehavior)
	}
	return c.BuildCommand(typ, payload)
}

// Abort builds an `abort` command: abort the current agent operation.
func (c *RpcClient) Abort() ([]byte, error) {
	return c.buildNoPayload("abort")
}

// NewSession builds a `new_session` command: start a fresh session. Optional
// parentSession path enables parent session tracking ("" omits the field).
func (c *RpcClient) NewSession(parentSession string) ([]byte, error) {
	if parentSession == "" {
		return c.buildNoPayload("new_session")
	}
	return c.BuildCommand("new_session", map[string]any{"parentSession": parentSession})
}

// GetState builds a `get_state` command (heartbeat + routing signal: the
// response data carries isStreaming/isCompacting/sessionId/sessionFile).
func (c *RpcClient) GetState() ([]byte, error) {
	return c.buildNoPayload("get_state")
}

// GetMessages builds a `get_messages` command (conversation messages).
func (c *RpcClient) GetMessages() ([]byte, error) {
	return c.buildNoPayload("get_messages")
}

// GetEntries builds a `get_entries` command: session entries in append order.
// since is a durable cursor (last seen entry id); empty string fetches all.
// Unlike get_messages this includes pre-compaction history and abandoned
// branches — the basis for web resume replay.
func (c *RpcClient) GetEntries(since string) ([]byte, error) {
	if since == "" {
		return c.buildNoPayload("get_entries")
	}
	return c.BuildCommand("get_entries", map[string]any{"since": since})
}

// SwitchSession builds a `switch_session` command: load a different session
// file into THIS subprocess (replaces the active session — serial reuse only,
// never concurrent multi-window; cancelled responses carry data.cancelled).
func (c *RpcClient) SwitchSession(sessionPath string) ([]byte, error) {
	if sessionPath == "" {
		return nil, fmt.Errorf("rpc: switch_session requires a sessionPath")
	}
	return c.BuildCommand("switch_session", map[string]any{"sessionPath": sessionPath})
}

// SetSessionName builds a `set_session_name` command (display name for
// session listings; also settable at startup via --name).
func (c *RpcClient) SetSessionName(name string) ([]byte, error) {
	if name == "" {
		return nil, fmt.Errorf("rpc: set_session_name requires a non-empty name")
	}
	return c.BuildCommand("set_session_name", map[string]any{"name": name})
}

// GetSessionStats builds a `get_session_stats` command: token/cost/context
// usage (drives the web session footer).
func (c *RpcClient) GetSessionStats() ([]byte, error) {
	return c.buildNoPayload("get_session_stats")
}

// SetModel builds a `set_model` command: switch the session to a specific
// model (provider + modelId per rpc.md Model section).
func (c *RpcClient) SetModel(provider, modelID string) ([]byte, error) {
	if provider == "" || modelID == "" {
		return nil, fmt.Errorf("rpc: set_model requires provider and modelId")
	}
	return c.BuildCommand("set_model", map[string]any{
		"provider": provider,
		"modelId":  modelID,
	})
}

// CycleModel builds a `cycle_model` command: switch to the next available
// model (response data is null when only one model is available).
func (c *RpcClient) CycleModel() ([]byte, error) {
	return c.buildNoPayload("cycle_model")
}

// GetAvailableModels builds a `get_available_models` command: list all
// configured models (response data carries {"models":[...]}).
func (c *RpcClient) GetAvailableModels() ([]byte, error) {
	return c.buildNoPayload("get_available_models")
}

// SetThinkingLevel builds a `set_thinking_level` command: set the
// reasoning/thinking level ("off" | "minimal" | "low" | "medium" | "high" |
// "xhigh" | "max" — the latter two only when the model supports them).
func (c *RpcClient) SetThinkingLevel(level string) ([]byte, error) {
	if level == "" {
		return nil, fmt.Errorf("rpc: set_thinking_level requires a level")
	}
	return c.BuildCommand("set_thinking_level", map[string]any{"level": level})
}

// GetAvailableThinkingLevels builds a `get_available_thinking_levels`
// command: list the thinking levels supported by the current model
// (response data carries {"levels":[...]}).
func (c *RpcClient) GetAvailableThinkingLevels() ([]byte, error) {
	return c.buildNoPayload("get_available_thinking_levels")
}

// --- extension_ui_response construction ---

// UIResponseKind enumerates the three response shapes of the extension UI
// sub-protocol (dialog methods only: select/confirm/input/editor).
type UIResponseKind int

const (
	// UIResponseValue answers select/input/editor with a chosen/edited value.
	UIResponseValue UIResponseKind = iota
	// UIResponseConfirm answers confirm with a boolean.
	UIResponseConfirm
	// UIResponseCancelled dismisses any dialog (extension receives
	// undefined/false depending on the method).
	UIResponseCancelled
)

// SendUIResponse builds an `extension_ui_response` for a pending
// extension_ui_request. id must match the request id. For UIResponseValue the
// value is the selected option string / edited text; for UIResponseConfirm it
// is the yes/no answer; UIResponseCancelled ignores value.
//
// Note: unlike every other command, the correlation here is the REQUEST's id
// (echoed as a payload field), not an auto-generated one — the response must
// carry exactly the id the request used. This is therefore built through a
// dedicated path instead of BuildCommand (which reserves "id" for itself).
func (c *RpcClient) SendUIResponse(id string, kind UIResponseKind, value any) ([]byte, error) {
	if id == "" {
		return nil, fmt.Errorf("rpc: extension_ui_response requires the request id")
	}
	var body map[string]any
	switch kind {
	case UIResponseValue:
		v, err := json.Marshal(value)
		if err != nil {
			return nil, fmt.Errorf("rpc: marshal ui value: %w", err)
		}
		body = map[string]any{"value": json.RawMessage(v)}
	case UIResponseConfirm:
		b, ok := value.(bool)
		if !ok {
			return nil, fmt.Errorf("rpc: UIResponseConfirm requires a bool value, got %T", value)
		}
		body = map[string]any{"confirmed": b}
	case UIResponseCancelled:
		body = map[string]any{"cancelled": true}
	default:
		return nil, fmt.Errorf("rpc: unknown UIResponseKind %d", kind)
	}
	body["id"] = id
	return c.buildWithEnvelope("extension_ui_response", body)
}

// --- RPC event parsing ---

// RpcEvent is one decoded stdout line of a pi rpc subprocess. The common
// envelope fields are typed; the complete original JSON is preserved in Raw
// so the web layer can relay full fidelity (message_update deltas,
// extension_ui_request dialogs, compaction results, ...) without this
// package chasing every event shape.
type RpcEvent struct {
	Type       string          `json:"type"`
	ID         string          `json:"id"`         // response correlation / extension_ui_request id
	Command    string          `json:"command"`    // responses only
	Success    bool            `json:"success"`    // responses only
	Error      string          `json:"error"`      // responses only, on failure
	SessionID  string          `json:"sessionId"`  // get_state data carries this; events may too
	ToolCallID string          `json:"toolCallId"` // tool_execution_* events
	ToolName   string          `json:"toolName"`   // tool_execution_* events
	Args       json.RawMessage `json:"args"`       // tool_execution_start/update
	Result     json.RawMessage `json:"result"`     // tool_execution_end
	IsError    bool            `json:"isError"`    // tool_execution_end
	Message    *piMessage      `json:"message"`    // message_start/end, turn_end
	Data       json.RawMessage `json:"data"`       // responses: command payload
	Raw        json.RawMessage `json:"-"`          // the complete original event JSON
}

// IsResponse reports whether the event is a command response (type ==
// "response") rather than a streamed agent event.
func (e *RpcEvent) IsResponse() bool { return e != nil && e.Type == "response" }

// IsExtensionUIRequest reports whether the event is an extension dialog /
// notification request that the client may need to answer
// (extension_ui_request — dialog methods expect an extension_ui_response).
func (e *RpcEvent) IsExtensionUIRequest() bool {
	return e != nil && e.Type == "extension_ui_request"
}

// rpcEventTypes is the full event vocabulary of the rpc protocol (rpc.md
// "Event Types" table + response + extension_ui_request). Unknown-but-valid
// JSON lines are still parsed (forward compatibility); this set documents the
// contract and powers test coverage.
var rpcEventTypes = map[string]bool{
	"response":                          true,
	"agent_start":                       true,
	"agent_end":                         true,
	"agent_settled":                     true,
	"turn_start":                        true,
	"turn_end":                          true,
	"message_start":                     true,
	"message_update":                    true,
	"message_end":                       true,
	"bash_execution_update":             true,
	"tool_execution_start":              true,
	"tool_execution_update":             true,
	"tool_execution_end":                true,
	"queue_update":                      true,
	"compaction_start":                  true,
	"compaction_end":                    true,
	"auto_retry_start":                  true,
	"auto_retry_end":                    true,
	"summarization_retry_scheduled":     true,
	"summarization_retry_attempt_start": true,
	"summarization_retry_finished":      true,
	"extension_error":                   true,
	"extension_ui_request":              true,
}

// ParseEventLine decodes one stdout line of a pi rpc subprocess into an
// RpcEvent. line must be a single JSON record WITHOUT the trailing LF
// (bufio.Scanner semantics; see the package comment for why Node readline
// splitting on U+2028/U+2029 is non-compliant but Go is fine). A trailing CR
// (from a pty or CRLF source) is tolerated and stripped.
//
// Empty/whitespace-only lines are an error (the supervisor's scanner should
// skip them before calling, but the parser stays strict). Invalid JSON is an
// error — the caller decides whether to log-skip (executor.go's json-mode
// parser logs and continues; rpc consumers should treat hard errors as wire
// corruption and surface them).
func ParseEventLine(line []byte) (*RpcEvent, error) {
	trimmed := strings.TrimRight(string(line), "\r")
	if strings.TrimSpace(trimmed) == "" {
		return nil, fmt.Errorf("rpc: empty event line")
	}
	var ev RpcEvent
	if err := json.Unmarshal([]byte(trimmed), &ev); err != nil {
		return nil, fmt.Errorf("rpc: parse event line: %w", err)
	}
	if ev.Type == "" {
		return nil, fmt.Errorf("rpc: event line has no type field")
	}
	ev.Raw = json.RawMessage(trimmed)
	return &ev, nil
}
