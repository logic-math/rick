package claudecode

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

// ClaudeCodeExecutor runs claude CLI and parses its stream-json output.
type ClaudeCodeExecutor struct {
	claudePath string
}

func NewExecutor(claudePath string) *ClaudeCodeExecutor {
	return &ClaudeCodeExecutor{claudePath: claudePath}
}

func (e *ClaudeCodeExecutor) Execute(promptFile, taskID string) (agent.AgentSession, error) {
	dir := filepath.Join("doing", "tasks", taskID)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("mkdir %s: %w", dir, err)
	}
	rawLogPath, err := filepath.Abs(filepath.Join(dir, "raw_session.log"))
	if err != nil {
		return nil, err
	}

	cmd := exec.Command(e.claudePath, "-p", "--output-format", "stream-json", "--verbose",
		"--dangerously-skip-permissions", promptFile)
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

// --- NDJSON types ---

type ndLine struct {
	Type       string     `json:"type"`
	SessionID  string     `json:"session_id"`
	Message    *ndMessage `json:"message,omitempty"`
	IsError    bool       `json:"is_error"`
	DurationMS int64      `json:"duration_ms"`
}

type ndMessage struct {
	Content []ndContent `json:"content"`
}

type ndContent struct {
	Type      string          `json:"type"`
	ID        string          `json:"id"`
	Name      string          `json:"name"`
	Input     json.RawMessage `json:"input"`
	Text      string          `json:"text"`
	ToolUseID string          `json:"tool_use_id"`
	Content   json.RawMessage `json:"content"`
	IsError   bool            `json:"is_error"`
}

// --- private session ---

type claudeSession struct {
	sessionID        string
	toolCalls        []agent.ToolCall
	finalMessage     string
	finalMessageLine int
	rawLogPath       string
	duration         time.Duration
}

func (s *claudeSession) ID() string                  { return s.sessionID }
func (s *claudeSession) Duration() time.Duration     { return s.duration }
func (s *claudeSession) ToolCalls() []agent.ToolCall { return s.toolCalls }
func (s *claudeSession) FinalMessage() string        { return s.finalMessage }
func (s *claudeSession) FinalMessageLine() int       { return s.finalMessageLine }
func (s *claudeSession) RawLogPath() string          { return s.rawLogPath }

func (s *claudeSession) GetRawLogPath() string { return s.rawLogPath }
func (s *claudeSession) GetErrorCount() int {
	count := 0
	for _, tc := range s.toolCalls {
		if tc.IsError {
			count++
		}
	}
	return count
}

// --- core parser ---

func parseStream(r io.Reader, rawLogPath string) (*claudeSession, error) {
	f, err := os.OpenFile(rawLogPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return nil, fmt.Errorf("open raw log %s: %w", rawLogPath, err)
	}
	defer f.Close()

	sess := &claudeSession{rawLogPath: rawLogPath}
	scanner := bufio.NewScanner(r)
	lineNo := 0

	for scanner.Scan() {
		line := scanner.Text()
		lineNo++
		fmt.Fprintln(f, line)

		var nd ndLine
		if err := json.Unmarshal([]byte(line), &nd); err != nil {
			preview := line
			if len(preview) > 60 {
				preview = preview[:60]
			}
			log.Printf("warn: skip non-json line %d: %s", lineNo, preview)
			continue
		}

		switch nd.Type {
		case "system":
			sess.sessionID = nd.SessionID
		case "result":
			if nd.SessionID != "" {
				sess.sessionID = nd.SessionID
			}
			sess.duration = time.Duration(nd.DurationMS) * time.Millisecond
		case "assistant":
			if nd.Message == nil {
				continue
			}
			for _, c := range nd.Message.Content {
				switch c.Type {
				case "tool_use":
					input := string(c.Input)
					input = truncate(input, 300)
					sess.toolCalls = append(sess.toolCalls, agent.ToolCall{
						Name:  c.Name,
						Input: input,
						Line:  lineNo,
					})
				case "text":
					sess.finalMessage = truncate(c.Text, 200)
					sess.finalMessageLine = lineNo
				}
			}
		case "user":
			if nd.Message == nil {
				continue
			}
			for _, c := range nd.Message.Content {
				if c.Type == "tool_result" && len(sess.toolCalls) > 0 {
					last := &sess.toolCalls[len(sess.toolCalls)-1]
					var output string
					if json.Unmarshal(c.Content, &output) != nil {
						output = string(c.Content)
					}
					last.Output = truncate(output, 300)
					if c.IsError {
						last.IsError = true
					}
				}
			}
		}
	}

	return sess, scanner.Err()
}

func truncate(s string, n int) string {
	r := []rune(s)
	if len(r) > n {
		return string(r[:n])
	}
	return s
}
