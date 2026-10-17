// Package super implements the process-supervision half of rick's `super`
// mode: a PySupervisor launches the stdlib-only Python evolution scheduler
// (`python3 -m superopt --workdir <ws>`) in its own process group, watches a
// file-based heartbeat, and reaps the whole tree through a
// SIGTERM→grace→SIGKILL ladder.
//
// The design mirrors internal/runtime's pi rpc supervisor (process-group
// kill ladder, heartbeat-driven death, exactly-once OnDead crash callback)
// but is deliberately NEW code (job_47/task10, J7 ruling): the pi
// supervisor's Spawn speaks the pi `--mode rpc` wire protocol, which a plain
// Python scheduler cannot answer. The heartbeat here is a JSON *file* the
// Python side rewrites on a cadence (`<ws>/heartbeat.json` ->
// {"ts","seq","state"}), not an RPC round-trip; a half-written file is
// tolerated (parse failure keeps the last good snapshot — only the timeout
// declares death).
//
// Crash recovery is callback-driven, not automatic: when a worker dies
// (process exit, heartbeat timeout) OnDead(ws, lastSeq) fires exactly once
// and the caller (task11's handler layer) decides whether to respawn — the
// scheduler replays its state from the events.jsonl cursor.
package super

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"
)

// Defaults mirror the py-side config.py DEFAULTS (job_47 task4) so the two
// languages agree on cadence unless explicitly overridden: 30s heartbeat
// interval, 90s timeout (3×), 5s TERM grace.
const (
	DefaultHeartbeatInterval = 30 * time.Second
	DefaultHeartbeatTimeout  = 90 * time.Second
	DefaultTermGrace         = 5 * time.Second
	DefaultPythonPath        = "python3"

	// pyTailSize bounds the retained stdout/stderr rings (diagnostics for
	// probe failures and post-mortem debugging).
	pyTailSize = 4096
)

// ErrLiveWorkerExists is returned by Start when the workspace already has a
// live worker registered. Dead workers do NOT block a re-Start — they are
// replaced (the OnDead respawn path relies on this).
var ErrLiveWorkerExists = errors.New("super: workspace already has a live worker")

// ErrEmptyWs is returned by Start without a workspace path.
var ErrEmptyWs = errors.New("super: workspace path must not be empty")

// PySupervisorConfig parameterizes a PySupervisor. Zero fields take the
// defaults above; explicit small values are honored (tests use 50ms
// heartbeats).
type PySupervisorConfig struct {
	// HeartbeatInterval is the heartbeat-file check cadence (default 30s).
	HeartbeatInterval time.Duration
	// HeartbeatTimeout is how long the heartbeat file may stay stale
	// (unchanged mtime AND unchanged seq) before the worker is declared
	// dead (default 90s).
	HeartbeatTimeout time.Duration
	// TermGrace is the SIGTERM→SIGKILL ladder grace (default 5s).
	TermGrace time.Duration
	// PythonPath is the interpreter used to launch the scheduler
	// (default "python3"; tests point it at fake scripts).
	PythonPath string
	// OnDead fires exactly once per worker death (process exit, heartbeat
	// timeout) with the workspace path and the last known heartbeat seq —
	// the events.jsonl cursor for respawn replay. Called on its own
	// goroutine; must not block. Restart POLICY lives in the caller
	// (task11), not here.
	OnDead func(ws string, lastSeq int64)
}

// Heartbeat is the parsed snapshot of <ws>/heartbeat.json.
type Heartbeat struct {
	Ts         time.Time `json:"-"` // parsed "ts" (zero when unparseable)
	Seq        int64     `json:"-"` // last valid "seq"
	State      string    `json:"-"` // py-side scheduler state (idle/running/…)
	ObservedAt time.Time `json:"-"` // when the supervisor saw this content
}

// PySupervisor owns a set of PyWorkers, one per super workspace. Safe for
// concurrent use.
type PySupervisor struct {
	cfg PySupervisorConfig

	mu      sync.RWMutex
	workers map[string]*PyWorker
}

// NewPySupervisor returns a PySupervisor with defaults applied for zero
// fields.
func NewPySupervisor(cfg PySupervisorConfig) *PySupervisor {
	if cfg.HeartbeatInterval <= 0 {
		cfg.HeartbeatInterval = DefaultHeartbeatInterval
	}
	if cfg.HeartbeatTimeout <= 0 {
		cfg.HeartbeatTimeout = DefaultHeartbeatTimeout
	}
	if cfg.TermGrace <= 0 {
		cfg.TermGrace = DefaultTermGrace
	}
	if cfg.PythonPath == "" {
		cfg.PythonPath = DefaultPythonPath
	}
	return &PySupervisor{cfg: cfg, workers: make(map[string]*PyWorker)}
}

// Start launches `python3 -m superopt --workdir <ws>` as a new process group
// (Setpgid) with cwd at the rick repo root and PYTHONPATH anchored at
// <repoRoot>/super so the `superopt` package resolves regardless of where
// the rick process was invoked from. The worker is registered under ws; a
// LIVE duplicate is rejected, a dead one is replaced (respawn path).
func (s *PySupervisor) Start(ws string) (*PyWorker, error) {
	if ws == "" {
		return nil, ErrEmptyWs
	}

	// Fast path rejection under the read lock (insert re-verifies below).
	s.mu.RLock()
	if old, ok := s.workers[ws]; ok && !old.IsDead() {
		s.mu.RUnlock()
		return nil, fmt.Errorf("%w: %s", ErrLiveWorkerExists, ws)
	}
	s.mu.RUnlock()

	repoRoot := resolveRepoRoot()
	cmd := exec.Command(s.cfg.PythonPath, "-m", "superopt", "--workdir", ws)
	cmd.Dir = repoRoot
	cmd.Env = pyEnv(repoRoot)
	// Own process group: the scheduler's pi agent subprocesses are
	// grandchildren that must die with the worker — kill(-pgid) reaches
	// the whole tree.
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("super: stdout pipe: %w", err)
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, fmt.Errorf("super: stderr pipe: %w", err)
	}
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("super: start scheduler: %w", err)
	}

	w := &PyWorker{
		sup:       s,
		ws:        ws,
		cmd:       cmd,
		startTime: time.Now(),
		waitCh:    make(chan struct{}),
		deadCh:    make(chan struct{}),
	}
	w.lastBeatObserved = w.startTime
	w.outRing = newByteRing(pyTailSize)
	w.errRing = newByteRing(pyTailSize)

	// The wait goroutine owns cmd.Wait (exactly once) and declares death on
	// process exit — whichever of wait / heartbeat-timeout fires first wins,
	// the rest no-op through deadOnce.
	go func() {
		w.waitErr = cmd.Wait()
		close(w.waitCh)
		w.refreshHeartbeatFinal() // freshest seq for the OnDead cursor
		w.markDead(exitReason(w.waitErr, w.isClosed()))
	}()

	// Drain both pipes into rings: an undrained pipe would fill (~64KB) and
	// BLOCK the scheduler; the tails power probe-failure diagnostics.
	go drainTo(stdout, w.outRing)
	go drainTo(stderr, w.errRing)

	// Heartbeat monitor: file-based liveness, timeout declares death and
	// runs the kill ladder.
	go w.monitorLoop(s.cfg.HeartbeatInterval, s.cfg.HeartbeatTimeout)

	s.mu.Lock()
	if old, ok := s.workers[ws]; ok && !old.IsDead() {
		s.mu.Unlock()
		_ = w.Close() // late duplicate discovery: reap our own spawn
		return nil, fmt.Errorf("%w: %s", ErrLiveWorkerExists, ws)
	}
	s.workers[ws] = w
	s.mu.Unlock()
	return w, nil
}

// Get returns the worker registered for ws (nil if absent). Dead workers
// stay retrievable until replaced by a re-Start or cleared by StopAll.
func (s *PySupervisor) Get(ws string) (*PyWorker, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	w, ok := s.workers[ws]
	return w, ok
}

// Count returns the number of live (not dead) workers.
func (s *PySupervisor) Count() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	n := 0
	for _, w := range s.workers {
		if !w.IsDead() {
			n++
		}
	}
	return n
}

// StopAll gracefully closes every worker and clears the registry.
func (s *PySupervisor) StopAll() {
	s.mu.Lock()
	ws := make([]*PyWorker, 0, len(s.workers))
	for _, w := range s.workers {
		ws = append(ws, w)
	}
	s.workers = make(map[string]*PyWorker)
	s.mu.Unlock()
	for _, w := range ws {
		if err := w.Close(); err != nil {
			fmt.Fprintf(os.Stderr, "[rick] super: stop %s: %v\n", w.ws, err)
		}
	}
}

// --- PyWorker ---

// PyWorker wraps one live scheduler subprocess (its whole process group).
type PyWorker struct {
	sup       *PySupervisor
	ws        string
	cmd       *exec.Cmd
	startTime time.Time

	mu               sync.Mutex
	closed           bool
	hb               *Heartbeat // last VALID parsed snapshot (nil until first)
	lastSeq          int64      // last valid seq — survives parse failures
	lastBeatObserved time.Time
	lastMtime        time.Time

	waitCh     chan struct{} // closed when cmd.Wait returns
	waitErr    error
	deadCh     chan struct{}
	deadOnce   sync.Once
	deadReason atomic.Value // string

	outRing *byteRing
	errRing *byteRing
}

// SessionID returns the workspace path this worker was spawned for.
func (w *PyWorker) SessionID() string { return w.ws }

// StartTime returns the spawn time.
func (w *PyWorker) StartTime() time.Time { return w.startTime }

// Pid returns the scheduler process id (0 before Start).
func (w *PyWorker) Pid() int {
	if w.cmd.Process == nil {
		return 0
	}
	return w.cmd.Process.Pid
}

// IsDead reports whether the worker has terminated (or was closed).
func (w *PyWorker) IsDead() bool {
	select {
	case <-w.deadCh:
		return true
	default:
		return false
	}
}

// Reason returns the death cause ("exit_nonzero: …", "exited: 0",
// "heartbeat_timeout", "closed").
func (w *PyWorker) Reason() string {
	if r, ok := w.deadReason.Load().(string); ok {
		return r
	}
	return ""
}

// StdoutTail returns the last n bytes of the scheduler stdout ring (whole
// ring when n <= 0).
func (w *PyWorker) StdoutTail(n int) string { return w.outRing.tail(n) }

// StderrTail returns the last n bytes of the scheduler stderr ring (whole
// ring when n <= 0).
func (w *PyWorker) StderrTail(n int) string { return w.errRing.tail(n) }

// LastHeartbeat returns a copy of the last valid heartbeat snapshot
// (nil before the first parseable one).
func (w *PyWorker) LastHeartbeat() *Heartbeat {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.hb == nil {
		return nil
	}
	cp := *w.hb
	return &cp
}

// Close terminates the worker: SIGTERM the process group → TermGrace →
// SIGKILL the group. Idempotent; returns an error only if the process
// refuses to die even under SIGKILL.
func (w *PyWorker) Close() error {
	w.mu.Lock()
	if w.closed {
		w.mu.Unlock()
		return nil
	}
	w.closed = true
	w.mu.Unlock()
	return w.ladder()
}

// ladder is the kill escalation shared by Close and heartbeat-death.
func (w *PyWorker) ladder() error {
	if w.cmd.Process != nil {
		_ = syscall.Kill(-w.cmd.Process.Pid, syscall.SIGTERM)
	}
	select {
	case <-w.waitCh:
		return nil
	case <-time.After(w.sup.cfg.TermGrace):
		if w.cmd.Process != nil {
			_ = syscall.Kill(-w.cmd.Process.Pid, syscall.SIGKILL)
		}
		select {
		case <-w.waitCh:
			return nil
		case <-time.After(2 * time.Second):
			return fmt.Errorf("super: worker %s refused to die after SIGKILL", w.ws)
		}
	}
}

func (w *PyWorker) isClosed() bool {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.closed
}

// markDead fires the one-shot death transition: stores the reason, then
// OnDead(ws, lastSeq) on its own goroutine (non-blocking contract).
func (w *PyWorker) markDead(reason string) {
	w.deadOnce.Do(func() {
		w.deadReason.Store(reason)
		w.mu.Lock()
		seq := w.lastSeq
		w.mu.Unlock()
		close(w.deadCh)
		// 诊断可见性：死亡原因落 stderr，与 runtime supervisor 同约定。
		fmt.Fprintf(os.Stderr, "[rick] super: worker %s died: %s (last_seq=%d)\n",
			w.ws, reason, seq)
		if cb := w.sup.cfg.OnDead; cb != nil {
			go cb(w.ws, seq)
		}
	})
}

// monitorLoop checks the heartbeat file on a cadence. A "beat" is an mtime
// change of <ws>/heartbeat.json (content parsed best-effort: a half-written
// file keeps the last good seq — only the timeout declares death). A
// heartbeat whose mtime predates the spawn is stale residue from a previous
// run and does NOT count as life. No beat for HeartbeatTimeout → death by
// "heartbeat_timeout" + kill ladder.
func (w *PyWorker) monitorLoop(interval, timeout time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-w.deadCh:
			return
		case <-ticker.C:
			if w.observeHeartbeat() {
				continue
			}
			w.mu.Lock()
			last := w.lastBeatObserved
			w.mu.Unlock()
			if time.Since(last) > timeout {
				w.markDead("heartbeat_timeout")
				_ = w.ladder()
				return
			}
		}
	}
}

// observeHeartbeat stats (and best-effort parses) the heartbeat file and
// reports whether a new beat was observed.
func (w *PyWorker) observeHeartbeat() bool {
	path := filepath.Join(w.ws, "heartbeat.json")
	info, err := os.Stat(path)
	if err != nil {
		return false // absent/unreadable: not a beat
	}
	mtime := info.ModTime()
	w.mu.Lock()
	defer w.mu.Unlock()
	if mtime.Before(w.startTime) || mtime.Equal(w.lastMtime) {
		return false // stale pre-spawn residue, or no change
	}
	w.lastMtime = mtime
	w.lastBeatObserved = time.Now()
	if data, err := os.ReadFile(path); err == nil {
		var raw struct {
			Ts    string `json:"ts"`
			Seq   int64  `json:"seq"`
			State string `json:"state"`
		}
		if json.Unmarshal(data, &raw) == nil {
			if raw.Seq >= w.lastSeq {
				w.lastSeq = raw.Seq
			}
			snap := Heartbeat{Seq: raw.Seq, State: raw.State, ObservedAt: w.lastBeatObserved}
			if t, err := time.Parse(time.RFC3339, raw.Ts); err == nil {
				snap.Ts = t
			}
			w.hb = &snap
		}
		// 解析失败（半写/损坏）＝沿用上一次（lastSeq / lastMtime 已更新），
		// 只有超时才判死——不在这里宣告死亡。
	}
	return true
}

// refreshHeartbeatFinal does one last best-effort read on process exit so
// OnDead carries the freshest possible events cursor.
func (w *PyWorker) refreshHeartbeatFinal() {
	path := filepath.Join(w.ws, "heartbeat.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return
	}
	var raw struct {
		Seq   int64  `json:"seq"`
		State string `json:"state"`
		Ts    string `json:"ts"`
	}
	if json.Unmarshal(data, &raw) != nil {
		return // half-written tail: keep the last good snapshot
	}
	w.mu.Lock()
	defer w.mu.Unlock()
	if raw.Seq > w.lastSeq {
		w.lastSeq = raw.Seq
		snap := Heartbeat{Seq: raw.Seq, State: raw.State, ObservedAt: time.Now()}
		if t, err := time.Parse(time.RFC3339, raw.Ts); err == nil {
			snap.Ts = t
		}
		w.hb = &snap
	}
}

// exitReason classifies a process exit for Reason().
func exitReason(err error, closed bool) string {
	if closed {
		return "closed"
	}
	var ee *exec.ExitError
	if errors.As(err, &ee) {
		if code := ee.ExitCode(); code > 0 {
			return fmt.Sprintf("exit_nonzero: %d", code)
		}
	}
	if err == nil {
		return "exited: 0"
	}
	return fmt.Sprintf("exited: %v", err)
}

// --- repo-root resolution & env ---

// resolveRepoRoot walks up from the process cwd looking for the repo that
// contains super/superopt/__main__.py (the module we spawn); falls back to
// the process cwd when the package is not found (tests, external installs).
func resolveRepoRoot() string {
	start, err := os.Getwd()
	if err != nil {
		return "."
	}
	dir := start
	for {
		if st, err := os.Stat(filepath.Join(dir, "super", "superopt", "__main__.py")); err == nil && !st.IsDir() {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return start // filesystem root: fall back to the process cwd
		}
		dir = parent
	}
}

// pyEnv builds the scheduler environment: inherited env plus PYTHONPATH with
// <repoRoot>/super prepended so `python3 -m superopt` resolves the package.
func pyEnv(repoRoot string) []string {
	entry := filepath.Join(repoRoot, "super")
	env := os.Environ()
	out := make([]string, 0, len(env)+1)
	for _, kv := range env {
		if strings.HasPrefix(kv, "PYTHONPATH=") {
			old := strings.TrimPrefix(kv, "PYTHONPATH=")
			if old != "" {
				entry = entry + string(os.PathListSeparator) + old
			}
			continue
		}
		out = append(out, kv)
	}
	return append(out, "PYTHONPATH="+entry)
}

// --- pipe draining & byte ring ---

func drainTo(r io.Reader, ring *byteRing) {
	buf := make([]byte, 1024)
	for {
		n, err := r.Read(buf)
		if n > 0 {
			ring.Write(buf[:n])
		}
		if err != nil {
			return
		}
	}
}

// byteRing is a fixed-size tail buffer (stderr/stdout diagnostics).
type byteRing struct {
	mu  sync.Mutex
	buf []byte
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

// tail returns the last n bytes (whole buffer when n <= 0).
func (r *byteRing) tail(n int) string {
	r.mu.Lock()
	defer r.mu.Unlock()
	if n <= 0 || n > len(r.buf) {
		return string(r.buf)
	}
	return string(r.buf[len(r.buf)-n:])
}
