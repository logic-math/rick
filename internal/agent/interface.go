package agent

import (
	"time"
)

// ToolCall represents a single tool invocation within an agent session.
type ToolCall struct {
	Name    string
	Input   string
	Output  string
	Line    int // line number in raw log
	IsError bool
}

// AgentSession provides read access to the results of a completed agent execution.
type AgentSession interface {
	ID() string
	Duration() time.Duration
	ToolCalls() []ToolCall
	FinalMessage() string
	FinalMessageLine() int
	RawLogPath() string
}

// AgentExecutor runs a prompt file and returns a session capturing the execution.
type AgentExecutor interface {
	Execute(promptFile, taskID string) (AgentSession, error)
}
