// Package runtime wraps rick's agent runtime calls (pi today, dsh later).
//
// pi is a Node.js/TypeScript coding agent (@earendil-works/pi-coding-agent) that
// rick shells out to as its agent runtime. rick acts as a bootloader: it
// deterministically assembles prompts/files and drives pi, which provides the
// actual agentic execution.
//
// This file implements the pi `--mode json` event-stream parser. pi's event
// schema differs from claude code's stream-json NDJSON: fields are camelCase,
// termination is the `agent_settled` event (no `result` line), tool calls are
// `tool_execution_start`/`tool_execution_end` events (not content blocks), and
// pi emits no duration (rick self-times).
package runtime

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"time"
)

// --- JSONL event types ---

type piEvent struct {
	Type       string          `json:"type"`
	ID         string          `json:"id"`
	SessionID  string          `json:"sessionId"`
	ToolCallID string          `json:"toolCallId"`
	ToolName   string          `json:"toolName"`
	Args       json.RawMessage `json:"args"`
	Result     json.RawMessage `json:"result"`
	IsError    bool            `json:"isError"`
	Message    *piMessage      `json:"message"`
}

type piMessage struct {
	Role    string      `json:"role"`
	Content []piContent `json:"content"`
}

type piContent struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

// --- private session ---

type piSession struct {
	sessionID        string
	toolCalls        []ToolCall
	toolCallIDs      []string
	finalMessage     string
	finalMessageLine int
	rawLogPath       string
	duration         time.Duration
	startTime        time.Time
	settled          bool
}

func (s *piSession) ID() string               { return s.sessionID }
func (s *piSession) Duration() time.Duration  { return s.duration }
func (s *piSession) ToolCalls() []ToolCall    { return s.toolCalls }
func (s *piSession) FinalMessage() string     { return s.finalMessage }
func (s *piSession) FinalMessageLine() int    { return s.finalMessageLine }
func (s *piSession) RawLogPath() string       { return s.rawLogPath }
func (s *piSession) GetRawLogPath() string    { return s.rawLogPath }

func (s *piSession) GetErrorCount() int {
	count := 0
	for _, tc := range s.toolCalls {
		if tc.IsError {
			count++
		}
	}
	return count
}

// isSessionReady reports whether a parsed pi session is ready to serve as the
// runtime's result: a non-empty session ID plus the agent_settled termination
// signal. A session without agent_settled is not an error at parse time
// (parseStream still returns the partial session with fallback timing) — the
// caller (Run) decides readiness.
func isSessionReady(sessionID string, settled bool) bool {
	return sessionID != "" && settled
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
	scanner.Buffer(make([]byte, 64*1024*1024), 64*1024*1024)
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

		if ev.SessionID != "" {
			sess.sessionID = ev.SessionID
		} else if ev.ID != "" && sess.sessionID == "" {
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
			sess.toolCalls = append(sess.toolCalls, ToolCall{
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

	if !sess.settled {
		sess.duration = time.Since(sess.startTime)
	}

	return sess, scanner.Err()
}

// matchToolCall resolves a tool_execution_end event to its start. Prefers an
// exact toolCallId match; otherwise falls back to the most recent tool call
// without a recorded output.
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
