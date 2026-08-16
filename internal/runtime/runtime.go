package runtime

import (
	"fmt"
	"os"
	"os/exec"
	"time"

	"github.com/sunquan/rick/internal/agent"
	"github.com/sunquan/rick/internal/config"
)

// Trace is the behavioral trace of a single runtime run. It carries the same
// information the old act-path + session pair held: the session identity, the
// tool-call history, the agent's final message, the raw JSONL log path, the
// self-timed duration, and whether the runtime reported a clean settle.
type Trace struct {
	SessionID    string
	ToolCalls    []agent.ToolCall
	FinalMessage string
	RawLogPath   string
	Duration     time.Duration
	Settled      bool
}

// Runtime is the seam for agent runtimes (pi today, dsh later). Handlers depend
// on this interface rather than a concrete piRuntime, so adding a new runtime
// means implementing and registering dshRuntime without touching the handlers
// or the method-layer templates.
//
// methodText is the method-layer system prompt (rick 方法描述): it is injected
// before the session via `--append-system-prompt <methodFile>` so pi keeps its
// default skeleton. promptFile is the instance context (user prompt).
type Runtime interface {
	Name() string
	Run(methodText string, promptFile string, cfg *config.Config) (sessionID string, trace *Trace, err error)
}

// piRuntime implements Runtime for the pi coding agent.
type piRuntime struct {
	piPath    string
	extraArgs []string
}

// NewPiRuntime constructs the pi runtime implementation. piPath may be empty
// (resolve via config / managed runtime / PATH); extraArgs are passed through to
// pi (e.g. --provider/--model/--api-key).
func NewPiRuntime(piPath string, extraArgs ...string) *piRuntime {
	return &piRuntime{piPath: piPath, extraArgs: extraArgs}
}

// Name returns the runtime's identifier ("pi").
func (r *piRuntime) Name() string { return "pi" }

// Run launches pi for one prompt and returns the parsed session id + trace.
//
// methodText (the method layer) is written to a temp file and injected through
// `--append-system-prompt <methodFile>` — a session-preparation injection that
// preserves pi's default skeleton and avoids passing long text inline. The temp
// file is created by the runtime and removed on return (defer cleanup, deleted
// once used). promptFile is passed last as the user prompt (instance context).
//
// This is the runtime contract's skeleton: it is not yet wired into handlers
// (task8 performs the Execute→Run switch). The AgentExecutor-compatible Execute
// below remains the wired entry point for the structured doing path.
func (r *piRuntime) Run(methodText string, promptFile string, cfg *config.Config) (string, *Trace, error) {
	piBin := r.piPath
	if piBin == "" {
		piBin = piPathOrDefault(cfg)
	}

	// Method layer → temp file → --append-system-prompt <methodFile>. The runtime
	// owns the temp file and deletes it once pi has consumed it (defer).
	var methodFile string
	if methodText != "" {
		f, err := os.CreateTemp("", "rick-method-*.md")
		if err != nil {
			return "", nil, fmt.Errorf("create method temp file: %w", err)
		}
		methodFile = f.Name()
		if _, err := f.WriteString(methodText); err != nil {
			f.Close()
			os.Remove(methodFile)
			return "", nil, fmt.Errorf("write method temp file: %w", err)
		}
		if err := f.Close(); err != nil {
			os.Remove(methodFile)
			return "", nil, fmt.Errorf("close method temp file: %w", err)
		}
		defer os.Remove(methodFile)
	}

	// Raw JSONL log: parseStream persists the event stream for the trace. The
	// skeleton keeps it in a temp file (task8 routes it to the job dir).
	rawLog, err := os.CreateTemp("", "rick-raw-*.log")
	if err != nil {
		return "", nil, fmt.Errorf("create raw log temp file: %w", err)
	}
	rawLogPath := rawLog.Name()
	rawLog.Close()
	defer os.Remove(rawLogPath)

	merged := mergeExtraArgs(cfg, r.extraArgs)
	args := make([]string, 0, len(merged)+6)
	args = append(args, "--mode", "json")
	args = append(args, merged...)
	if methodFile != "" {
		args = append(args, "--append-system-prompt", methodFile)
	}
	if promptFile != "" {
		args = append(args, promptFile)
	}

	cmd := exec.Command(piBin, args...)
	// rick-managed pi config isolation (same as CallCLI / Execute).
	cmd.Env = AgentEnv()
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return "", nil, err
	}
	if err := cmd.Start(); err != nil {
		return "", nil, err
	}

	sess, parseErr := parseStream(stdout, rawLogPath)
	cmd.Wait() //nolint
	if parseErr != nil {
		return "", nil, parseErr
	}

	trace := &Trace{
		SessionID:    sess.sessionID,
		ToolCalls:    sess.toolCalls,
		FinalMessage: sess.finalMessage,
		RawLogPath:   sess.rawLogPath,
		Duration:     sess.duration,
		Settled:      sess.settled,
	}
	return sess.sessionID, trace, nil
}
