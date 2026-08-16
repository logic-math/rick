package runtime

import (
	"fmt"
	"os"
	"os/exec"
	"time"

	"github.com/sunquan/rick/internal/config"
)

// ToolCall is a single tool invocation within a runtime session. It replaces
// the deleted internal/agent.ToolCall — the runtime is now the sole owner of
// the session/trace vocabulary (there is no second agent runtime to abstract).
type ToolCall struct {
	Name    string
	Input   string
	Output  string
	Line    int
	IsError bool
}

// Trace is the behavioral trace of a single runtime run. It carries the same
// information the old act-path + session pair held: the session identity, the
// tool-call history, the agent's final message, the raw JSONL log path, the
// self-timed duration, and whether the runtime reported a clean settle.
type Trace struct {
	SessionID    string
	ToolCalls    []ToolCall
	FinalMessage string
	RawLogPath   string
	Duration     time.Duration
	Settled      bool
}

// Runtime is the seam for agent runtimes (pi today, dsh later). Handlers depend
// on this interface rather than a concrete piRuntime, so adding a new runtime
// means implementing and registering the corresponding Runtime without touching the handlers
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
// The runtime contract (task8 Execute→Run switch): returning a non-empty
// sessionID means the run succeeded. If the JSONL stream never produced a
// session id or never emitted agent_settled (the session did not settle), Run
// returns an error so the caller (handler) can apply its retry safety net.
//
// methodText (the method layer) is written to a temp file and injected through
// `--append-system-prompt <methodFile>`. promptFile is passed last as the user
// prompt (instance context). The temp method file is removed on return.
func (r *piRuntime) Run(methodText string, promptFile string, cfg *config.Config) (string, *Trace, error) {
	piBin := r.piPath
	if piBin == "" {
		piBin = piPathOrDefault(cfg)
	}

	// Method layer → temp file → --append-system-prompt <methodFile>.
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

	// Raw JSONL log: the runtime persists the event stream for the trace.
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
	// rick-managed pi config isolation (same as CallCLI).
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

	// Readiness contract: a run that never settled (or produced no session id)
	// is a failure from the caller's perspective.
	if !isSessionReady(sess.sessionID, sess.settled) {
		return "", trace, fmt.Errorf("pi session did not settle (sessionID=%q settled=%v)", sess.sessionID, sess.settled)
	}

	return sess.sessionID, trace, nil
}
