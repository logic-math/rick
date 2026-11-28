// daemon.go implements the daemon-mode pi worker transport: instead of the
// web server holding pi's stdin/stdout pipes directly (server restart =
// stdin EOF = pi exits, killing every in-flight tool call / subagent), each
// pi worker runs behind a small node shim that owns the pipes and exposes a
// unix-socket endpoint:
//
//	Go web (restartable) ── unix socket ──→ shim (own process group) ── pipes ──→ pi --mode rpc
//
// The Go side speaks the exact same JSON-line RPC protocol over the socket
// (net.Conn satisfies Worker's io.ReadWriteCloser contract), so the entire
// event pipeline (scanLoop/intercept/pump/heartbeat) is transport-agnostic
// and reused verbatim. The shim buffers pi's stdout into a ring while the
// server is down and replays it on reconnect (shim_attached marker), so a
// restart loses nothing the pi process emitted in the meantime.
//
// Lifecycle paths (the web layer drives these):
//   - spawnDaemon:  fresh worker (node shim + pi, both in one Setpgid group)
//   - Attach:       reconnect to a live daemon socket after a server restart
//   - DetachAll:    server shutdown — drop connections, keep pi alive
//   - CleanupOrphans: sockets with no wanted session → kill group + remove files
package runtime

import (
	"crypto/sha256"
	_ "embed"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"syscall"
	"time"
)

//go:embed daemon_shim.mjs
var daemonShimSrc string

// DefaultDaemonDialTimeout is the post-spawn window for the shim to bind its
// socket (node startup ~100-300ms; generous for loaded machines).
const DefaultDaemonDialTimeout = 10 * time.Second

// DefaultDaemonSocketDir is the default daemon socket directory
// (<agent-dir>/sockets): it follows RICK_PI_AGENT_DIR, so tests that redirect
// the agent dir are automatically isolated from real daemon sockets.
func DefaultDaemonSocketDir() string {
	return filepath.Join(AgentDir(), "sockets")
}

// daemonDialRetry is the poll cadence inside daemonDial.
const daemonDialRetry = 50 * time.Millisecond

// unixSocketPathMax bounds a daemon socket path (sun_path is 104 bytes on
// macOS / 108 on Linux — an over-long path fails connect with EINVAL).
const unixSocketPathMax = 96

// minStemLen is the floor for a truncated socket stem (id head + hash
// suffix must still fit).
const minStemLen = 18

// daemonLogTail reads the last ~size bytes of a daemon stdio log file
// (diagnostics for probe failures / post-mortem). Missing file → "".
func daemonLogTail(path string, size int) string {
	f, err := os.Open(path)
	if err != nil {
		return ""
	}
	defer f.Close()
	fi, err := f.Stat()
	if err != nil {
		return ""
	}
	var off int64
	if fi.Size() > int64(size) {
		off = fi.Size() - int64(size)
	}
	buf := make([]byte, int64(size))
	n, _ := f.ReadAt(buf, off)
	return string(buf[:n])
}

// ErrDaemonAlive reports a Spawn onto a session whose daemon socket already
// answers — the caller should Attach instead (one session, one pi worker).
var ErrDaemonAlive = errors.New("supervisor: live daemon already exists for session — use Attach")

// socketStem derives the socket filename stem for a session id: the
// sanitized id when it fits, else a stable truncation (id head + sha256
// prefix — deterministic across restarts, collision-negligible).
func (s *Supervisor) socketStem(sessionID string) string {
	name := sanitizeSessionID(sessionID)
	max := unixSocketPathMax - len(s.cfg.DaemonSocketDir) - len("/") - len(".sock")
	if max < minStemLen {
		max = minStemLen
	}
	if len(name) <= max {
		return name
	}
	sum := sha256.Sum256([]byte(sessionID))
	return name[:min(8, len(name))] + "-" + hex.EncodeToString(sum[:8])
}

// socketPath returns the daemon socket path for a session id (sanitized
// and length-guarded — unix sockets cap sun_path at 104/108 bytes on
// macOS/Linux, and a too-long path fails dial with EINVAL).
func (s *Supervisor) socketPath(sessionID string) string {
	return filepath.Join(s.cfg.DaemonSocketDir, s.socketStem(sessionID)+".sock")
}

var daemonPathUnsafe = regexp.MustCompile(`[^a-zA-Z0-9._-]`)

// sanitizeSessionID maps a session id onto a safe filename component.
func sanitizeSessionID(sessionID string) string {
	id := daemonPathUnsafe.ReplaceAllString(sessionID, "_")
	if id == "" {
		return "_"
	}
	return id
}

// materializeShim writes the embedded shim source into the socket dir
// (idempotent: skips the write when the content already matches).
func (s *Supervisor) materializeShim() (string, error) {
	if err := os.MkdirAll(s.cfg.DaemonSocketDir, 0o755); err != nil {
		return "", fmt.Errorf("supervisor: daemon socket dir: %w", err)
	}
	path := filepath.Join(s.cfg.DaemonSocketDir, "daemon_shim.mjs")
	if cur, err := os.ReadFile(path); err == nil && string(cur) == daemonShimSrc {
		return path, nil
	}
	if err := os.WriteFile(path, []byte(daemonShimSrc), 0o755); err != nil {
		return "", fmt.Errorf("supervisor: write daemon shim: %w", err)
	}
	return path, nil
}

// daemonDial connects to a unix socket, retrying until the deadline (the
// shim needs a moment to bind after spawn).
func daemonDial(path string, timeout time.Duration) (net.Conn, error) {
	deadline := time.Now().Add(timeout)
	for {
		conn, err := net.Dial("unix", path)
		if err == nil {
			return conn, nil
		}
		if time.Now().After(deadline) {
			return nil, err
		}
		time.Sleep(daemonDialRetry)
	}
}

// daemonAlive probes a daemon socket with a short window. Connected → the
// caller owns the (closed) probe conn; true means a live daemon answers.
func daemonAlive(socketPath string) bool {
	conn, err := net.DialTimeout("unix", socketPath, 200*time.Millisecond)
	if err != nil {
		return false
	}
	_ = conn.Close()
	return true
}

// readDaemonPgid reads the pgid recorded by the shim next to its socket
// (<socket>.meta.json). Returns 0 when unreadable (kill on Close is then a
// no-op — the socket is dead anyway).
func readDaemonPgid(socketPath string) int {
	data, err := os.ReadFile(socketPath + ".meta.json")
	if err != nil {
		return 0
	}
	var meta struct {
		Pgid int `json:"pgid"`
	}
	if json.Unmarshal(data, &meta) != nil || meta.Pgid <= 0 {
		return 0
	}
	return meta.Pgid
}

// killGroupDetached terminates a process group the supervisor does NOT own
// as a child (attached workers; orphan cleanup). Same SIGTERM→grace→SIGKILL
// ladder as Worker.Close, with group-liveness polling (syscall.Kill(-pgid, 0))
// standing in for cmd.Wait.
//
// Guards: a garbage/stale meta file must never make us signal init (pgid 1
// — kill(-1, TERM) hits every process the user can signal) or ourselves
// (our own process group) — the meta file is best-effort trust.
func killGroupDetached(pgid int, grace time.Duration) {
	if pgid <= 1 {
		return
	}
	if own := syscall.Getpgrp(); pgid == own {
		return
	}
	_ = syscall.Kill(-pgid, syscall.SIGTERM)
	deadline := time.Now().Add(grace)
	for time.Now().Before(deadline) {
		if err := syscall.Kill(-pgid, 0); err != nil {
			return // ESRCH: group gone
		}
		time.Sleep(100 * time.Millisecond)
	}
	_ = syscall.Kill(-pgid, syscall.SIGKILL)
	// KILL settle window: poll a bounded 2s like Worker.Close's fallback.
	deadline = time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if err := syscall.Kill(-pgid, 0); err != nil {
			return
		}
		time.Sleep(100 * time.Millisecond)
	}
}

// spawnDaemon is Spawn's single implementation (human 2026-11-28 单轨化：
// the direct-pipe transport is deleted). Builds the pi argv, wraps it in the
// node shim behind a unix socket, dials it, and gives the pair a probe
// window to die (bad flags / missing node surface as a Spawn error).
func (s *Supervisor) spawnDaemon(spec SpawnSpec) (*Worker, error) {
	if err := s.precheckSpawn(spec); err != nil {
		return nil, err
	}
	// Normalize (precheckSpawn normalizes its own copy for the dup check).
	if spec.SessionID == "" {
		spec.SessionID = spec.SessionIDFlag
	}
	// Unix socket path budget: sun_path caps at 104/108 bytes; the stem
	// length guard cannot help if the DIR alone is too long — fail loudly
	// instead of a cryptic EINVAL on dial.
	if n := len(s.cfg.DaemonSocketDir) + minStemLen + len("/") + len(".sock"); n > 104 {
		return nil, fmt.Errorf("supervisor: daemon socket dir %q too long for a unix socket (%d+%d > 104) — set web_daemon_socket_dir to a shorter path", s.cfg.DaemonSocketDir, len(s.cfg.DaemonSocketDir), minStemLen)
	}
	socketPath := s.socketPath(spec.SessionID)
	// Idempotency: a live daemon already owns this session — never spawn a
	// second pi (one session, one worker). The web layer attaches instead.
	if daemonAlive(socketPath) {
		return nil, fmt.Errorf("%w %q", ErrDaemonAlive, spec.SessionID)
	}
	// Stale socket files from a crashed pair: remove before the shim binds
	// (the shim unlinks on exit, but a SIGKILLed shim leaves residue).
	_ = os.Remove(socketPath)
	_ = os.Remove(socketPath + ".meta.json")
	_ = os.Remove(socketPath + ".log")

	piBin := s.cfg.PiPath
	if piBin == "" {
		piBin = piPathOrDefault(nil)
	}
	nodeBin := s.cfg.NodeBin
	if nodeBin == "" {
		nodeBin = "node"
	}
	shimPath, err := s.materializeShim()
	if err != nil {
		return nil, err
	}

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

	// node shim <sock> -- pi --mode rpc ...  (shim reads NO stdin; pi's
	// stdin/stdout are owned by the shim, pi's stderr is inherited by the
	// shim and forwarded to its own stderr → supervisor stderrTail).
	cmdArgs := append([]string{shimPath, socketPath, "--", piBin}, args...)
	cmd := exec.Command(nodeBin, cmdArgs...)
	if spec.Dir != "" {
		cmd.Dir = spec.Dir
	}
	cmd.Env = AgentEnv()
	// Own process group: the shim is the group leader; pi and every tool
	// call / subagent it spawns share the pgid, so kill(-pgid) reaches the
	// whole tree on explicit close — while a plain server restart (detach)
	// leaves the group untouched.
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	// stdio decoupling（真 e2e 实测修复 2026-11-28）：shim/pi 的 stdout/stderr
	// 绝不能是连回本进程的管道——本进程（web server）死亡时管道断裂，
	// pi 及其子进程任何一次 stderr 写入就 EPIPE 崩溃（dev 实测：detach 正常
	// 执行后 2 秒内 pi 仍死）。全部落日志文件 <stem>.log，进程生命周期与
	// web 完全解耦；探活失败的诊断从文件尾部读取。
	logPath := socketPath + ".log"
	logf, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return nil, fmt.Errorf("supervisor: open daemon log: %w", err)
	}
	cmd.Stdout = logf
	cmd.Stderr = logf
	if err := cmd.Start(); err != nil {
		logf.Close()
		return nil, fmt.Errorf("supervisor: start shim: %w", err)
	}
	// 子进程持有自己的 FD 副本；父侧句柄即可关闭。
	logf.Close()

	w := &Worker{
		sup:          s,
		sessionID:    spec.SessionID,
		cmd:          cmd,
		client:       NewRpcClient(),
		events:       make(chan *RpcEvent, eventChanBuffer),
		notify:       make(chan struct{}, 1),
		waitCh:       make(chan struct{}),
		deadCh:       make(chan struct{}),
		abandonC:     make(chan struct{}),
		lastActivity: time.Now(),
	}
	w.daemonLogPath = logPath

	go func() {
		w.waitErr = cmd.Wait()
		close(w.waitCh)
		w.markDead(fmt.Sprintf("exited: %v", w.waitErr))
	}()

	// Probe: dial the shim's socket (node startup included in the window).
	// A fast shim death (bad node, exec error, full disk) surfaces with its
	// stderr tail; a bind timeout is the fallback error.
	type dialResult struct {
		conn net.Conn
		err  error
	}
	dialCh := make(chan dialResult, 1)
	go func() {
		c, err := daemonDial(socketPath, DefaultDaemonDialTimeout)
		dialCh <- dialResult{conn: c, err: err}
	}()
	var conn net.Conn
	select {
	case <-w.waitCh:
		w.abandon()
		return nil, fmt.Errorf("supervisor: shim exited during probe: %v (stderr: %s)", w.waitErr, w.StderrTail())
	case res := <-dialCh:
		if res.err != nil {
			// Prefer the exit reason when the shim died mid-dial.
			select {
			case <-w.waitCh:
				w.abandon()
				return nil, fmt.Errorf("supervisor: shim exited during probe: %v (stderr: %s)", w.waitErr, w.StderrTail())
			default:
			}
			w.abandon()
			_ = cmd.Process.Kill()
			return nil, fmt.Errorf("supervisor: dial daemon socket: %w", res.err)
		}
		conn = res.conn
	}

	// Settle window（probe 语义，承接 legacy probe 的职责）：dial 成功只证明
	// shim 活着——再给 ProbeTimeout 让「秒死对」（bad flag 的 pi、启动即退的
	// shim）把 waitCh 落下来，调用方拿到同步 Spawn 错误而不是「成功后立刻
	// 异步死」。健康的 shim 在窗口内不会退出，代价只是每次 spawn 多等一个
	// ProbeTimeout（默认 500ms；测试可调小）。
	select {
	case <-w.waitCh:
		w.abandon()
		_ = conn.Close()
		return nil, fmt.Errorf("supervisor: shim exited during probe: %v (stderr: %s)", w.waitErr, w.StderrTail())
	case <-time.After(s.cfg.ProbeTimeout):
		// alive
	}

	s.mu.Lock()
	if _, dup := s.workers[spec.SessionID]; dup {
		s.mu.Unlock()
		_ = conn.Close()
		w.abandon()
		_ = syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
		return nil, fmt.Errorf("supervisor: session %q already has a worker", spec.SessionID)
	}
	if s.liveCountLocked() >= s.cfg.MaxActive {
		s.mu.Unlock()
		_ = conn.Close()
		w.abandon()
		_ = syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
		return nil, ErrMaxActive
	}
	w.stdin = conn
	w.pgid = cmd.Process.Pid
	s.workers[spec.SessionID] = w
	s.mu.Unlock()

	go w.scanLoop(conn)
	go w.pumpLoop()
	if s.cfg.HeartbeatInterval > 0 {
		go w.heartbeatLoop(s.cfg.HeartbeatInterval, s.cfg.HeartbeatTimeout)
	}
	if s.cfg.IdleTimeout > 0 {
		go w.reapLoop(s.cfg.IdleTimeout)
	}
	return w, nil
}

// Attach reconnects to a live daemon worker (server restart path). The
// worker has no child process (the shim belongs to no Go process anymore);
// death detection is scanLoop EOF (shim closes the conn when pi exits) plus
// the heartbeat ladder. Closing kills the recorded process group.
//
// Liveness probe (get_state round trip, DefaultAttachProbeTimeout): a
// SIGKILLed shim's listener can still ACCEPT connections during kernel
// teardown — a bare dial in that window "succeeds" against a phantom that
// never answers (observed: kill → immediate Attach → connect OK → get_state
// 5s timeout → session wrongly degraded). Attach therefore refuses to hand
// out a worker until the daemon actually answers.
func (s *Supervisor) Attach(sessionID string) (*Worker, error) {
	s.mu.RLock()
	if _, dup := s.workers[sessionID]; dup {
		s.mu.RUnlock()
		return nil, fmt.Errorf("supervisor: session %q already has a worker", sessionID)
	}
	s.mu.RUnlock()

	socketPath := s.socketPath(sessionID)
	conn, err := net.DialTimeout("unix", socketPath, 2*time.Second)
	if err != nil {
		return nil, fmt.Errorf("supervisor: attach %s: %w", sessionID, err)
	}
	pgid := readDaemonPgid(socketPath)

	w := &Worker{
		sup:          s,
		sessionID:    sessionID,
		stdin:        conn,
		client:       NewRpcClient(),
		events:       make(chan *RpcEvent, eventChanBuffer),
		notify:       make(chan struct{}, 1),
		waitCh:       make(chan struct{}),
		deadCh:       make(chan struct{}),
		abandonC:     make(chan struct{}),
		lastActivity: time.Now(),
		pgid:         pgid,
		attached:     true,
	}
	w.stderrTail = newByteRing(stderrTailSize)

	s.mu.Lock()
	if _, dup := s.workers[sessionID]; dup {
		s.mu.Unlock()
		_ = conn.Close()
		return nil, fmt.Errorf("supervisor: session %q already has a worker", sessionID)
	}
	s.workers[sessionID] = w
	s.mu.Unlock()

	go w.scanLoop(conn)
	go w.pumpLoop()
	if s.cfg.HeartbeatInterval > 0 {
		go w.heartbeatLoop(s.cfg.HeartbeatInterval, s.cfg.HeartbeatTimeout)
	}
	if s.cfg.IdleTimeout > 0 {
		go w.reapLoop(s.cfg.IdleTimeout)
	}

	// End-to-end liveness probe: the daemon must answer get_state. The
	// response is cached by intercept (Worker.State) and still flows to the
	// events channel (harmless extra envelope for the consumer).
	if line, err := w.client.GetState(); err == nil {
		if err := w.writeRaw(line); err != nil {
			s.dropAttached(w, sessionID, conn)
			return nil, fmt.Errorf("supervisor: attach %s: daemon write failed: %w", sessionID, err)
		}
		deadline := time.Now().Add(DefaultAttachProbeTimeout)
		for w.State() == nil && time.Now().Before(deadline) && !w.IsDead() {
			time.Sleep(20 * time.Millisecond)
		}
	}
	if w.State() == nil {
		reason := "daemon did not answer get_state (phantom or dead)"
		s.dropAttached(w, sessionID, conn)
		return nil, fmt.Errorf("supervisor: attach %s: %s", sessionID, reason)
	}
	return w, nil
}

// DefaultAttachProbeTimeout bounds the Attach get_state liveness probe.
const DefaultAttachProbeTimeout = 2 * time.Second

// dropAttached unwinds a failed Attach (phantom daemon): close the conn,
// release the pump, unregister — leave no half-attached worker behind.
// markDead is called so housekeeping goroutines (heartbeat/reap) exit and
// consumers see the channel close; OnDead firing is fine — the web layer
// treats a failed attach's OnDead like any dead worker (suspended path).
func (s *Supervisor) dropAttached(w *Worker, sessionID string, conn net.Conn) {
	_ = conn.Close()
	w.markDead("attach probe failed")
	w.abandon()
	s.mu.Lock()
	if cur, ok := s.workers[sessionID]; ok && cur == w {
		delete(s.workers, sessionID)
	}
	s.mu.Unlock()
}

// DetachAll drops every worker's socket connection WITHOUT touching the
// daemon processes (server shutdown / upgrade path): pi keeps running, the
// shim buffers its stdout, a restarted server attaches back via Attach.
// The supervisor's worker map is emptied (the processes are no longer ours);
// they are remembered on the detached list for KillDetached (test hygiene).
func (s *Supervisor) DetachAll() {
	s.mu.Lock()
	ws := make([]*Worker, 0, len(s.workers))
	for id, w := range s.workers {
		ws = append(ws, w)
		delete(s.workers, id)
	}
	s.mu.Unlock()
	s.detachedMu.Lock()
	s.detached = append(s.detached, ws...)
	s.detachedMu.Unlock()
	for _, w := range ws {
		w.detach()
	}
}

// KillDetached terminates every daemon left alive by DetachAll. It exists
// for test hygiene (a detached fake pi would otherwise outlive the test
// binary); production never calls it — after a server exit the next
// server's ReconcileOnStart/CleanupOrphansExcept owns the daemons.
func (s *Supervisor) KillDetached() {
	s.detachedMu.Lock()
	ws := s.detached
	s.detached = nil
	s.detachedMu.Unlock()
	for _, w := range ws {
		killGroupDetached(w.pgid, s.cfg.TermGrace)
	}
}

// CleanupSessionFiles removes one session's daemon socket + meta files
// (explicit-close path: the shim unlinks them on exit, but a SIGKILLed
// shim can leave residue). No-op when absent.
func (s *Supervisor) CleanupSessionFiles(sessionID string) {
	socketPath := s.socketPath(sessionID)
	_ = os.Remove(socketPath)
	_ = os.Remove(socketPath + ".meta.json")
	_ = os.Remove(socketPath + ".log")
}

// CleanupOrphansExcept removes daemon sockets whose session id is NOT in
// keep: kills the recorded process group (leftover pi from a closed
// session, or a crashed server's session that was closed on disk) and
// deletes socket + meta files. Returns the stems it cleaned (for logging).
// keep entries are full session ids — the supervisor maps them onto socket
// stems itself (stems can be length-truncated, so callers never derive
// them by hand).
func (s *Supervisor) CleanupOrphansExcept(keep []string) []string {
	keepSet := make(map[string]bool, len(keep))
	for _, id := range keep {
		keepSet[s.socketStem(id)] = true
	}
	entries, err := os.ReadDir(s.cfg.DaemonSocketDir)
	if err != nil {
		return nil
	}
	var cleaned []string
	for _, e := range entries {
		name := e.Name()
		if !strings.HasSuffix(name, ".sock") || e.IsDir() {
			continue
		}
		stem := strings.TrimSuffix(name, ".sock")
		if keepSet[stem] {
			continue
		}
		socketPath := filepath.Join(s.cfg.DaemonSocketDir, name)
		if pgid := readDaemonPgid(socketPath); pgid > 0 {
			killGroupDetached(pgid, s.cfg.TermGrace)
		}
		_ = os.Remove(socketPath)
		_ = os.Remove(socketPath + ".meta.json")
		_ = os.Remove(socketPath + ".log")
		cleaned = append(cleaned, stem)
	}
	return cleaned
}
