// supervisor.go implements the pi `--mode rpc` subprocess supervisor: the
// process-management half of rick's web session execution layer (rpc.go is
// the wire half).
//
// One Worker = one `pi --mode rpc` subprocess owning exactly one pi session
// (pi runs a single active session per process — switch_session REPLACES it,
// so concurrent chat windows require one worker each; research-L1-r2). The
// supervisor caps concurrency, probes spawns, heartbeats via get_state,
// reclaims idle workers, and terminates workers by killing the whole process
// GROUP (pi's bash tool leaves grandchildren that would outlive a
// single-pid kill — the Setpgid/-pgid pattern is lifted from pi's own
// subagent extension, as is the SIGTERM→grace→SIGKILL ladder).
//
// Crash recovery is deliberately NOT automatic: when a worker dies the
// OnDead(sessionID, lastEntryID) callback fires and the web layer decides
// whether to respawn (spawn + `--session` + get_entries(since) replay).
// Hidden auto-respawn would mask state transitions the UI must surface.

package runtime

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sync"
	"sync/atomic"
	"syscall"
	"time"
)

// BootstrapMessage re-exports the CLI bootstrap trigger for web sessions.
// cli.go keeps the canonical constant unexported; task11's web session layer
// lives outside this package and needs it to kick spawned rpc sessions with
// the exact same trigger the CLI paths use.
var BootstrapMessage = bootstrapMessage

// Supervisor defaults (mirroring pi's own subagent extension magnitudes).
const (
	DefaultMaxActive         = 8                // pi subagent MAX_PARALLEL_TASKS ballpark
	DefaultIdleTimeout       = 30 * time.Minute // idle web sessions are expensive
	DefaultHeartbeatInterval = 30 * time.Second // get_state liveness cadence
	DefaultProbeTimeout      = 500 * time.Millisecond
	DefaultTermGrace         = 5 * time.Second
	eventChanBuffer          = 256  // consumer-facing channel buffer (spec)
	maxPendingEvents         = 1024 // staging queue cap; overflow drops OLDEST
	stderrTailSize           = 4096 // stderr ring for probe failure messages
	maxEventLineSize         = 8 << 20
)

// ErrMaxActive is returned by Spawn when the supervisor already holds the
// configured maximum of live workers.
var ErrMaxActive = errors.New("supervisor: max active workers reached")

// ErrEmptySessionFlag is returned by Spawn without a session flag value.
var ErrEmptySessionFlag = errors.New("supervisor: SpawnSpec.SessionIDFlag must not be empty")

// SupervisorConfig parameterizes a Supervisor. Zero fields take the defaults
// above; explicit small values are honored (tests use 50ms heartbeats).
type SupervisorConfig struct {
	// MaxActive caps concurrent live workers (default 8).
	MaxActive int
	// IdleTimeout reclaims workers with no activity for this long while not
	// streaming (default 30m; 0 disables idle reaping).
	IdleTimeout time.Duration
	// HeartbeatInterval is the get_state cadence (default 30s; 0 disables).
	HeartbeatInterval time.Duration
	// HeartbeatTimeout is how long a heartbeat may stay unanswered before the
	// worker is declared dead (default 2× HeartbeatInterval).
	HeartbeatTimeout time.Duration
	// ProbeTimeout is the post-spawn liveness window (default 500ms).
	ProbeTimeout time.Duration
	// TermGrace is the SIGTERM→SIGKILL ladder grace (default 5s).
	TermGrace time.Duration
	// PiPath overrides pi binary resolution (tests point it at fake scripts;
	// empty resolves managed runtime → PATH, same as piPathOrDefault).
	PiPath string
	// ExtraArgs are passed through to pi (e.g. --provider/--model/--api-key).
	ExtraArgs []string
	// OnDead fires once per worker death (process exit, heartbeat timeout,
	// idle reap) with the session id and the last known entry cursor. Called
	// on its own goroutine; must not block.
	OnDead func(sessionID, lastEntryID string)
}

// SpawnSpec describes one worker launch.
type SpawnSpec struct {
	// SessionID is rick's web-session identifier (worker registry key).
	SessionID string
	// Dir is the subprocess working directory — the workspace root whose
	// .rick/ the session anchors to. Empty inherits the rick process cwd.
	Dir string
	// MethodFile is injected via --append-system-prompt (method layer).
	MethodFile string
	// PromptFile is injected via --append-system-prompt (instance layer).
	PromptFile string
	// SessionIDFlag is the value passed to --session-id / --session.
	SessionIDFlag string
	// CreateNew selects the pi session flag semantics (bugs.md job_30):
	// true  → `--session-id <flag>`  create a new pi session with that id
	// false → `--session <flag>`     resume the existing pi session (error
	//                                 if no session matches).
	CreateNew bool
}

// WorkerState is the cached result of the latest get_state heartbeat.
type WorkerState struct {
	IsStreaming  bool   `json:"isStreaming"`
	IsCompacting bool   `json:"isCompacting"`
	SessionID    string `json:"sessionId"`
	SessionFile  string `json:"sessionFile"`
	UpdatedAt    time.Time
}

// Supervisor owns a set of Workers. It is safe for concurrent use.
type Supervisor struct {
	cfg SupervisorConfig

	mu      sync.RWMutex
	workers map[string]*Worker
}

// NewSupervisor returns a Supervisor with defaults applied for zero fields.
func NewSupervisor(cfg SupervisorConfig) *Supervisor {
	if cfg.MaxActive <= 0 {
		cfg.MaxActive = DefaultMaxActive
	}
	if cfg.IdleTimeout == 0 {
		cfg.IdleTimeout = DefaultIdleTimeout
	}
	if cfg.HeartbeatInterval == 0 {
		cfg.HeartbeatInterval = DefaultHeartbeatInterval
	}
	if cfg.HeartbeatTimeout <= 0 {
		cfg.HeartbeatTimeout = 2 * cfg.HeartbeatInterval
	}
	if cfg.ProbeTimeout <= 0 {
		cfg.ProbeTimeout = DefaultProbeTimeout
	}
	if cfg.TermGrace <= 0 {
		cfg.TermGrace = DefaultTermGrace
	}
	return &Supervisor{cfg: cfg, workers: make(map[string]*Worker)}
}

// Spawn launches one pi rpc subprocess per spec and probes it alive.
// The worker is registered under spec.SessionID; a duplicate id is rejected.
func (s *Supervisor) Spawn(spec SpawnSpec) (*Worker, error) {
	if spec.SessionIDFlag == "" {
		return nil, ErrEmptySessionFlag
	}
	if spec.SessionID == "" {
		spec.SessionID = spec.SessionIDFlag
	}

	// Concurrency cap: count live workers under the read lock, then
	// re-verify under the write lock on insert (small race window is fine —
	// the insert itself is serialized).
	s.mu.RLock()
	live := 0
	for _, w := range s.workers {
		if !w.IsDead() {
			live++
		}
	}
	if _, dup := s.workers[spec.SessionID]; dup {
		s.mu.RUnlock()
		return nil, fmt.Errorf("supervisor: session %q already has a worker", spec.SessionID)
	}
	s.mu.RUnlock()
	if live >= s.cfg.MaxActive {
		return nil, ErrMaxActive
	}

	piBin := s.cfg.PiPath
	if piBin == "" {
		piBin = piPathOrDefault(nil)
	}

	// Args are assembled as a slice — Go's variadic spread cannot be followed
	// by more positional args (spec note).
	args := make([]string, 0, 4+len(s.cfg.ExtraArgs))
	args = append(args, "--mode", "rpc")
	args = append(args, s.cfg.ExtraArgs...)
	if spec.MethodFile != "" {
		args = append(args, "--append-system-prompt", spec.MethodFile)
	}
	if spec.PromptFile != "" {
		args = append(args, "--append-system-prompt", spec.PromptFile)
	}
	if spec.CreateNew {
		args = append(args, "--session-id", spec.SessionIDFlag)
	} else {
		args = append(args, "--session", spec.SessionIDFlag)
	}

	cmd := exec.Command(piBin, args...)
	if spec.Dir != "" {
		cmd.Dir = spec.Dir
	}
	cmd.Env = AgentEnv()
	// Own process group: pi's bash tool spawns grandchildren that must die
	// with the worker — kill(-pgid) reaches the whole tree.
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("supervisor: stdin pipe: %w", err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("supervisor: stdout pipe: %w", err)
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, fmt.Errorf("supervisor: stderr pipe: %w", err)
	}
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("supervisor: start pi: %w", err)
	}

	w := &Worker{
		sup:       s,
		sessionID: spec.SessionID,
		cmd:       cmd,
		stdin:     stdin,
		client:    NewRpcClient(),
		events:    make(chan *RpcEvent, eventChanBuffer),
		notify:    make(chan struct{}, 1),
		waitCh:    make(chan struct{}),
		deadCh:    make(chan struct{}),
		abandonC:  make(chan struct{}),
		lastActivity: time.Now(),
	}
	w.stderrTail = newByteRing(stderrTailSize)

	// The wait goroutine owns cmd.Wait (exactly once) and declares death on
	// process exit — whichever of scanner-EOF / heartbeat-timeout / wait
	// fires first wins, the rest no-op through deadOnce.
	go func() {
		w.waitErr = cmd.Wait()
		close(w.waitCh)
		w.markDead(fmt.Sprintf("exited: %v", w.waitErr))
	}()

	// stderr drain: keeps the pipe from filling (a full stderr pipe would
	// block the child) and retains a tail for probe-failure diagnostics.
	go func() {
		buf := make([]byte, 1024)
		for {
			n, err := stderr.Read(buf)
			if n > 0 {
				w.stderrTail.Write(buf[:n])
			}
			if err != nil {
				return
			}
		}
	}()

	// Probe: give the process a window to die (exec failures, bad flags).
	select {
	case <-w.waitCh:
		tail := w.StderrTail()
		w.abandon()
		return nil, fmt.Errorf("supervisor: pi exited during probe: %v (stderr: %s)", w.waitErr, tail)
	case <-time.After(s.cfg.ProbeTimeout):
		// alive
	}

	s.mu.Lock()
	if _, dup := s.workers[spec.SessionID]; dup {
		s.mu.Unlock()
		w.Close()
		return nil, fmt.Errorf("supervisor: session %q already has a worker", spec.SessionID)
	}
	if s.liveCountLocked() >= s.cfg.MaxActive {
		s.mu.Unlock()
		w.Close()
		return nil, ErrMaxActive
	}
	s.workers[spec.SessionID] = w
	s.mu.Unlock()

	// Event pipeline: scanner parses stdout lines, intercepts housekeeping
	// responses (get_state / get_entries), stages everything on a queue; the
	// pump drains the queue into the buffered channel. Slow consumers make
	// the queue grow until maxPendingEvents, then the OLDEST events are
	// dropped (overflow counter exposed via Dropped()).
	go w.scanLoop(stdout)
	go w.pumpLoop()

	// Heartbeat: get_state cadence; unanswered heartbeats declare death.
	if s.cfg.HeartbeatInterval > 0 {
		go w.heartbeatLoop(s.cfg.HeartbeatInterval, s.cfg.HeartbeatTimeout)
	}
	// Idle reaping: quiet, non-streaming workers are closed and reported.
	if s.cfg.IdleTimeout > 0 {
		go w.reapLoop(s.cfg.IdleTimeout)
	}
	return w, nil
}

// Get returns the worker registered for sessionID (nil if absent).
func (s *Supervisor) Get(sessionID string) *Worker {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.workers[sessionID]
}

// Count returns the number of live (not dead) workers.
func (s *Supervisor) Count() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.liveCountLocked()
}

func (s *Supervisor) liveCountLocked() int {
	n := 0
	for _, w := range s.workers {
		if !w.IsDead() {
			n++
		}
	}
	return n
}

// Remove drops a worker from the registry (after Close). Idempotent.
func (s *Supervisor) Remove(sessionID string) {
	s.mu.Lock()
	delete(s.workers, sessionID)
	s.mu.Unlock()
}

// CloseAll gracefully closes every worker (server shutdown path).
func (s *Supervisor) CloseAll() {
	s.mu.RLock()
	ws := make([]*Worker, 0, len(s.workers))
	for _, w := range s.workers {
		ws = append(ws, w)
	}
	s.mu.RUnlock()
	for _, w := range ws {
		w.Close()
	}
}

// --- Worker ---

// Worker wraps one live pi rpc subprocess.
type Worker struct {
	sup       *Supervisor
	sessionID string
	cmd       *exec.Cmd
	stdin     io.WriteCloser
	client    *RpcClient

	writeMu sync.Mutex // serializes stdin writes

	mu           sync.Mutex
	closed       bool
	lastActivity time.Time
	lastEntryID  string
	state        *WorkerState
	hbDeadline   time.Time

	events chan *RpcEvent // buffered eventChanBuffer; closed on death
	// queue staging: scanner → pending → pump → events
	queueMu     sync.Mutex
	pending     []*RpcEvent
	supplyDone  bool
	notify      chan struct{} // cap 1 wakeup for the pump
	dropped     atomic.Int64
	abandonC    chan struct{}
	abandonOnce    sync.Once
	closeEventsOnce sync.Once

	waitCh   chan struct{} // closed when cmd.Wait returns
	waitErr  error
	deadCh   chan struct{}
	deadOnce sync.Once
	deadReason atomic.Value // string

	stderrTail *byteRing
}

// SessionID returns the web-session id this worker was spawned for.
func (w *Worker) SessionID() string { return w.sessionID }

// IsDead reports whether the worker has terminated (or was closed).
func (w *Worker) IsDead() bool {
	select {
	case <-w.deadCh:
		return true
	default:
		return false
	}
}

// Reason returns the death cause diagnostics ("exited: ...", "heartbeat timeout", "idle reap").
func (w *Worker) Reason() string {
	if r, ok := w.deadReason.Load().(string); ok {
		return r
	}
	return ""
}

// Dropped returns the number of events dropped from the head of the staging
// queue (slow-consumer overflow). Non-zero means the consumer lost history.
func (w *Worker) Dropped() int64 { return w.dropped.Load() }

// Events is the worker's event stream (pi rpc stdout decoded). The channel
// is closed when the worker dies; unread events remain buffered.
func (w *Worker) Events() <-chan *RpcEvent { return w.events }

// StderrTail returns the last ~4KB of the subprocess stderr.
func (w *Worker) StderrTail() string { return w.stderrTail.String() }

// State returns the latest heartbeat snapshot (nil before the first
// get_state response arrives).
func (w *Worker) State() *WorkerState {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.state == nil {
		return nil
	}
	cp := *w.state
	return &cp
}

// LastEntryID returns the most recent get_entries leafId seen ("" if none) —
// the durable cursor OnDead hands to the web layer for respawn replay.
func (w *Worker) LastEntryID() string {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.lastEntryID
}

// Send writes one pre-built rpc command line to the subprocess stdin.
// Errors mean the pipe is gone (process dying) — the wait goroutine will
// declare death; callers treat Send errors as best-effort.
func (w *Worker) Send(cmd []byte) error {
	if len(cmd) == 0 {
		return nil
	}
	return w.writeLine(cmd)
}

// writeLine writes a command and counts it as user-visible activity (idle
// reaping resets). Heartbeats use writeRaw so supervisor housekeeping never
// keeps an idle worker alive.
func (w *Worker) writeLine(line []byte) error {
	if err := w.writeRaw(line); err != nil {
		return err
	}
	w.touch()
	return nil
}

func (w *Worker) writeRaw(line []byte) error {
	w.writeMu.Lock()
	defer w.writeMu.Unlock()
	if w.stdin == nil {
		return errors.New("supervisor: stdin already closed")
	}
	if _, err := w.stdin.Write(line); err != nil {
		return fmt.Errorf("supervisor: write stdin: %w", err)
	}
	return nil
}

func (w *Worker) touch() {
	w.mu.Lock()
	w.lastActivity = time.Now()
	w.mu.Unlock()
}

// Close terminates the worker: best-effort rpc abort → SIGTERM the process
// group → TermGrace → SIGKILL the group. Idempotent.
func (w *Worker) Close() {
	w.mu.Lock()
	if w.closed {
		w.mu.Unlock()
		return
	}
	w.closed = true
	w.mu.Unlock()

	// Spec ladder: abort first (lets pi flush a final event burst), then TERM.
	if line, err := w.client.Abort(); err == nil {
		_ = w.writeLine(line)
	}

	if w.cmd.Process != nil {
		_ = syscall.Kill(-w.cmd.Process.Pid, syscall.SIGTERM)
	}
	select {
	case <-w.waitCh:
	case <-time.After(w.sup.cfg.TermGrace):
		if w.cmd.Process != nil {
			_ = syscall.Kill(-w.cmd.Process.Pid, syscall.SIGKILL)
		}
		select {
		case <-w.waitCh:
		case <-time.After(2 * time.Second):
			// Refused to die even under KILL; wait goroutine still owns Wait.
		}
	}

	w.writeMu.Lock()
	if w.stdin != nil {
		_ = w.stdin.Close()
		w.stdin = nil
	}
	w.writeMu.Unlock()

	// Release the pump (consumer may still be draining buffered events; the
	// once-guard makes this safe after a natural drain).
	w.abandon()
}

// abandon unblocks the pump's send path after termination.
func (w *Worker) abandon() {
	w.abandonOnce.Do(func() { close(w.abandonC) })
}

// markDead fires the one-shot death transition.
func (w *Worker) markDead(reason string) {
	w.deadOnce.Do(func() {
		w.deadReason.Store(reason)
		w.mu.Lock()
		last := w.lastEntryID
		w.mu.Unlock()
		close(w.deadCh)
		if cb := w.sup.cfg.OnDead; cb != nil {
			go cb(w.sessionID, last)
		}
	})
}

// scanLoop reads stdout lines, decodes them, intercepts housekeeping
// responses (get_state state cache + heartbeat ack, get_entries leafId
// cursor) and stages every event for the pump.
func (w *Worker) scanLoop(stdout io.Reader) {
	sc := bufio.NewScanner(stdout)
	sc.Buffer(make([]byte, 0, 64*1024), maxEventLineSize)
	for sc.Scan() {
		line := sc.Bytes()
		if len(line) == 0 {
			continue
		}
		ev, err := ParseEventLine(line)
		if err != nil {
			// Wire corruption: log to stderr and keep scanning (the json-mode
			// executor has the same skip-and-continue posture).
			fmt.Fprintf(os.Stderr, "[rick] supervisor: skip bad rpc line: %v\n", err)
			continue
		}
		w.intercept(ev)
		w.stage(ev)
	}
	// stdout EOF: the process is gone or closed its stdout. Supply ends; the
	// pump drains the queue and closes the events channel.
	w.endSupply()
}

// intercept caches housekeeping responses without consuming them — events
// still reach the consumer channel (the web layer relays responses too).
func (w *Worker) intercept(ev *RpcEvent) {
	if ev.Type != "response" || !ev.Success || len(ev.Data) == 0 {
		return
	}
	switch ev.Command {
	case "get_state":
		var st WorkerState
		if err := json.Unmarshal(ev.Data, &st); err == nil {
			st.UpdatedAt = time.Now()
			w.mu.Lock()
			w.state = &st
			w.hbDeadline = time.Time{}
			w.mu.Unlock()
		}
	case "get_entries":
		var d struct {
			LeafID string `json:"leafId"`
		}
		if err := json.Unmarshal(ev.Data, &d); err == nil && d.LeafID != "" {
			w.mu.Lock()
			if d.LeafID != w.lastEntryID {
				w.lastEntryID = d.LeafID
			}
			w.mu.Unlock()
		}
	}
}

// stage appends an event to the pending queue (dropping the OLDEST on
// overflow) and wakes the pump. Only non-response events refresh activity:
// command responses (including heartbeat get_state echoes) are wire-level
// housekeeping and must not keep an idle worker alive.
func (w *Worker) stage(ev *RpcEvent) {
	if ev.Type != "response" {
		w.touch()
	}
	w.queueMu.Lock()
	w.pending = append(w.pending, ev)
	if n := len(w.pending); n > maxPendingEvents {
		drop := n - maxPendingEvents
		w.pending = w.pending[drop:]
		w.dropped.Add(int64(drop))
	}
	w.queueMu.Unlock()
	select {
	case w.notify <- struct{}{}:
	default:
	}
}

// endSupply signals that no further events will be staged.
func (w *Worker) endSupply() {
	w.queueMu.Lock()
	w.supplyDone = true
	w.queueMu.Unlock()
	select {
	case w.notify <- struct{}{}:
	default:
	}
}

// pumpLoop moves staged events into the buffered channel. When supply has
// ended and the queue is drained, the channel is closed. Consumers that stop
// reading block the pump here — the queue absorbs up to maxPendingEvents
// events and then drops the oldest (Dropped() reflects the loss).
func (w *Worker) pumpLoop() {
	for {
		w.queueMu.Lock()
		if len(w.pending) == 0 {
			done := w.supplyDone
			w.queueMu.Unlock()
			if done {
				w.closeEventsOnce.Do(func() { close(w.events) })
				return
			}
			select {
			case <-w.notify:
				continue
			case <-w.abandonC:
				w.closeEventsOnce.Do(func() { close(w.events) })
				return
			}
		}
		ev := w.pending[0]
		w.pending = w.pending[1:]
		w.queueMu.Unlock()

		select {
		case w.events <- ev:
		case <-w.abandonC:
			w.closeEventsOnce.Do(func() { close(w.events) })
			return
		}
	}
}

// heartbeatLoop sends get_state on a cadence and declares death when a
// heartbeat stays unanswered past the timeout window.
func (w *Worker) heartbeatLoop(interval, timeout time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-w.deadCh:
			return
		case <-w.abandonC:
			return
		case <-ticker.C:
			w.mu.Lock()
			deadline := w.hbDeadline
			w.mu.Unlock()
			if !deadline.IsZero() && time.Now().After(deadline) {
				w.markDead("heartbeat timeout")
				return
			}
			line, err := w.client.GetState()
			if err != nil {
				continue // builder error; retry next tick
			}
			if err := w.writeRaw(line); err != nil {
				// Pipe gone — the wait goroutine declares death.
				continue
			}
			w.mu.Lock()
			w.hbDeadline = time.Now().Add(timeout)
			w.mu.Unlock()
		}
	}
}

// reapLoop closes workers that stayed idle past IdleTimeout while not
// streaming (e.g. a chat window left open overnight).
func (w *Worker) reapLoop(idle time.Duration) {
	check := idle / 4
	if check < 10*time.Millisecond {
		check = 10 * time.Millisecond
	}
	ticker := time.NewTicker(check)
	defer ticker.Stop()
	for {
		select {
		case <-w.deadCh:
			return
		case <-w.abandonC:
			return
		case <-ticker.C:
			w.mu.Lock()
			last := w.lastActivity
			st := w.state
			closed := w.closed
			w.mu.Unlock()
			if closed {
				return
			}
			streaming := st != nil && st.IsStreaming
			if !streaming && time.Since(last) > idle {
				w.Close()
				w.markDead("idle reap")
				return
			}
		}
	}
}

// --- byteRing: fixed-size stderr tail buffer ---

type byteRing struct {
	mu  sync.Mutex
	buf []byte
	n   int
}

func newByteRing(size int) *byteRing {
	return &byteRing{buf: make([]byte, 0, size)}
}

func (r *byteRing) Write(p []byte) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.buf = append(r.buf, p...)
	if len(r.buf) > cap(r.buf) {
		r.buf = r.buf[len(r.buf)-cap(r.buf):]
	}
}

func (r *byteRing) String() string {
	r.mu.Lock()
	defer r.mu.Unlock()
	return string(r.buf)
}
