// Package piagent integrates rick with the pi coding agent (https://pi.dev).
//
// pi is a Node.js/TypeScript coding agent (@earendil-works/pi-coding-agent) that
// rick shells out to as its agent runtime. rick acts as a bootloader: it
// deterministically assembles prompts/files and drives pi, which provides the
// actual agentic execution. See RFC rfc-rick-pi-迁移的价值基础与架构定位-2026-08-10.
//
// This file implements the AgentExecutor interface (the structured doing.go path)
// using pi's `--mode json` output: a JSONL event stream over stdout. pi's event
// schema differs from claude code's stream-json NDJSON (see research-5-N2 /
// research-7-N4): fields are camelCase, termination is the `agent_settled` event
// (no `result` line), tool calls are `tool_execution_start`/`tool_execution_end`
// events (not content blocks), and pi emits no duration (rick self-times).
package piagent

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/sunquan/rick/internal/agent"
)

// Executor runs the pi binary in `--mode json` mode and parses its JSONL event stream.
type Executor struct {
	piPath    string
	extraArgs []string // extra pi flags (e.g. --provider/--model/--api-key) from config
}

// NewExecutor creates a pi agent executor. piPath may be empty (resolve "pi" via
// PATH). extraArgs are passed through to pi before the prompt file — use them to
// configure provider/model/api-key (pi does not read these from env vars).
func NewExecutor(piPath string, extraArgs ...string) *Executor {
	return &Executor{piPath: piPath, extraArgs: extraArgs}
}

// Execute runs pi for a single prompt file and returns the parsed session.
// It shells out to `pi --mode json [extraArgs] <promptFile>` (per-prompt subprocess,
// isomorphic to the previous claude `stream-json` invocation). pi has no permission
// popups, so no `--dangerously-skip-permissions` flag is needed.
func (e *Executor) Execute(promptFile, taskID, workspaceDir, logFileName string) (agent.AgentSession, error) {
	dir := filepath.Join(workspaceDir, "tasks", taskID)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("mkdir %s: %w", dir, err)
	}
	if logFileName == "" {
		logFileName = "raw_session_coding.log"
	}
	rawLogPath, err := filepath.Abs(filepath.Join(dir, logFileName))
	if err != nil {
		return nil, err
	}

	piBin := e.piPath
	if piBin == "" {
		piBin = "pi"
	}

	// pi json mode: JSONL event stream over stdout (session header + events + agent_settled).
	args := append([]string{"--mode", "json"}, e.extraArgs...)
	args = append(args, promptFile)
	cmd := exec.Command(piBin, args...)
	// rick-managed pi config isolation (same as CallCLI): pi reads its settings/
	// extensions/themes from ~/.rick/pi/agent, not the user's ~/.pi.
	cmd.Env = AgentEnv()
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	if err := cmd.Start(); err != nil {
		return nil, err
	}

	sess, parseErr := parseStream(stdout, rawLogPath)
	cmd.Wait() //nolint
	return sess, parseErr
}

// --- JSONL event types ---
//
// pi emits one JSON object per line. The schema below is derived from the
// research briefs (research-5-N2 / research-7-N4, which read pi's source:
// rpc-types.ts, agent-session.ts, agent-loop.ts). Field names are camelCase.
// CAUTION: the exact wrapper shape was captured from source that is no longer
// available; calibrate against real `pi --mode json` output once pi is installed.

type piEvent struct {
	Type       string          `json:"type"`       // "session"/"agent_start"/"message_end"/"tool_execution_start"/"tool_execution_end"/"agent_settled"/...
	ID         string          `json:"id"`         // session header id
	SessionID  string          `json:"sessionId"`  // alternative session id (camelCase)
	ToolCallID string          `json:"toolCallId"` // tool events
	ToolName   string          `json:"toolName"`   // tool_execution_start
	Args       json.RawMessage `json:"args"`       // tool_execution_start
	Result     json.RawMessage `json:"result"`     // tool_execution_end
	IsError    bool            `json:"isError"`    // tool_execution_end
	Message    *piMessage      `json:"message"`    // message_end
}

type piMessage struct {
	Role    string      `json:"role"` // "user" / "assistant" — only assistant text is the final message
	Content []piContent `json:"content"`
}

type piContent struct {
	Type string `json:"type"` // "text"
	Text string `json:"text"`
}

// --- private session ---

type piSession struct {
	sessionID        string
	toolCalls        []agent.ToolCall
	toolCallIDs      []string // parallel to toolCalls, for matching end→start by toolCallId
	finalMessage     string
	finalMessageLine int
	rawLogPath       string
	duration         time.Duration
	startTime        time.Time
	settled          bool
}

func (s *piSession) ID() string                  { return s.sessionID }
func (s *piSession) Duration() time.Duration     { return s.duration }
func (s *piSession) ToolCalls() []agent.ToolCall { return s.toolCalls }
func (s *piSession) FinalMessage() string        { return s.finalMessage }
func (s *piSession) FinalMessageLine() int       { return s.finalMessageLine }
func (s *piSession) RawLogPath() string          { return s.rawLogPath }

func (s *piSession) GetRawLogPath() string { return s.rawLogPath }
func (s *piSession) GetErrorCount() int {
	count := 0
	for _, tc := range s.toolCalls {
		if tc.IsError {
			count++
		}
	}
	return count
}

// --- core parser ---

func parseStream(r io.Reader, rawLogPath string) (*piSession, error) {
	f, err := os.OpenFile(rawLogPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return nil, fmt.Errorf("open raw log %s: %w", rawLogPath, err)
	}
	defer f.Close()

	sess := &piSession{rawLogPath: rawLogPath, startTime: time.Now()}
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 64*1024*1024), 64*1024*1024) // 64MB per line for large tool outputs
	lineNo := 0

	for scanner.Scan() {
		line := scanner.Text()
		lineNo++
		fmt.Fprintln(f, line)

		var ev piEvent
		if err := json.Unmarshal([]byte(line), &ev); err != nil {
			preview := line
			if len(preview) > 60 {
				preview = preview[:60]
			}
			log.Printf("warn: skip non-json line %d: %s", lineNo, preview)
			continue
		}

		// Track session id from any line that carries one (header / state events).
		if ev.SessionID != "" {
			sess.sessionID = ev.SessionID
		} else if ev.ID != "" && sess.sessionID == "" {
			// First-line session header typically exposes `id` without a sessionId.
			sess.sessionID = ev.ID
		}

		switch ev.Type {
		case "agent_settled":
			sess.settled = true
			sess.duration = time.Since(sess.startTime)
		case "message_end":
			if ev.Message == nil {
				continue
			}
			// Only the assistant's final message is the agent's output; pi also
			// emits message_end for the user turn (echoing the prompt), which we
			// must not mistake for the agent's reply.
			if ev.Message.Role != "assistant" {
				continue
			}
			for _, c := range ev.Message.Content {
				if c.Type == "text" && c.Text != "" {
					sess.finalMessage = truncate(c.Text, 200)
					sess.finalMessageLine = lineNo
				}
			}
		case "tool_execution_start":
			input := truncate(string(ev.Args), 300)
			sess.toolCalls = append(sess.toolCalls, agent.ToolCall{
				Name:  ev.ToolName,
				Input: input,
				Line:  lineNo,
			})
			sess.toolCallIDs = append(sess.toolCallIDs, ev.ToolCallID)
		case "tool_execution_end":
			output := rawToString(ev.Result)
			output = truncate(output, 300)
			idx := matchToolCall(sess, ev.ToolCallID)
			if idx >= 0 {
				sess.toolCalls[idx].Output = output
				sess.toolCalls[idx].IsError = ev.IsError
			}
		}
	}

	// If pi never emitted agent_settled (e.g. crashed or non-termination), still
	// record elapsed time so Duration() is meaningful.
	if !sess.settled {
		sess.duration = time.Since(sess.startTime)
	}

	return sess, scanner.Err()
}

// matchToolCall resolves a tool_execution_end event to the tool_execution_start
// it pairs with. It prefers an exact toolCallId match; if none (or empty id), it
// falls back to the most recent tool call without a recorded output — mirroring
// the previous claude parser's "last tool call" behavior.
func matchToolCall(sess *piSession, toolCallID string) int {
	if toolCallID != "" {
		for i, id := range sess.toolCallIDs {
			if id == toolCallID {
				return i
			}
		}
	}
	for i := len(sess.toolCalls) - 1; i >= 0; i-- {
		if sess.toolCalls[i].Output == "" {
			return i
		}
	}
	if len(sess.toolCalls) > 0 {
		return len(sess.toolCalls) - 1
	}
	return -1
}

// rawToString unwraps a JSON RawMessage that may hold a JSON string or an
// arbitrary JSON value, returning a plain string for logging/truncation.
func rawToString(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}
	var s string
	if json.Unmarshal(raw, &s) == nil {
		return s
	}
	return string(raw)
}

func truncate(s string, n int) string {
	r := []rune(s)
	if len(r) > n {
		return string(r[:n])
	}
	return s
}
