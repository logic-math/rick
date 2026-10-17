// supervisor_test.go exercises the PySupervisor against fake scheduler
// shell scripts (the internal/runtime/supervisor_test.go pattern, adapted to
// the file-heartbeat contract of `python3 -m superopt`).
//
// Isolation (bugs.md):
//   - Fake scripts restore the system PATH (job_33: sh scripts that call
//     external commands like sleep/date would silently fail).
//   - No PATH-fake pi interference applies here (we spawn our own fakes via
//     PythonPath), but tests never touch the managed pi runtime.
//
// Timing: heartbeats use 50ms intervals / 250ms timeouts so every case
// completes in seconds (spec: "心跳超时用例用短 interval：50ms/200ms 级").
package super

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"
)

// --- fake scheduler fixtures ---

// wsArgvParser extracts the --workdir value from "$@" (shared sh snippet).
const wsArgvParser = `WS=""; prev=""
for a in "$@"; do
  if [ "$prev" = "--workdir" ]; then WS="$a"; fi
  prev="$a"
done
mkdir -p "$WS"
`

// fakePyBeating writes a valid heartbeat every 50ms, optionally spawns a
// `sleep 300` grandchild (writing its pid) and optionally traps TERM.
func fakePyBeating(t *testing.T, grandchildPath, termTrap string) string {
	t.Helper()
	var b strings.Builder
	b.WriteString("#!/bin/sh\n")
	// job_33 pitfall guard: restore the system PATH so sleep/date work.
	b.WriteString("export PATH=/usr/bin:/bin:/usr/sbin:/sbin:$PATH\n")
	b.WriteString(wsArgvParser)
	if termTrap != "" {
		b.WriteString(fmt.Sprintf("trap '%s' TERM\n", termTrap))
	}
	if grandchildPath != "" {
		b.WriteString("sleep 300 &\n")
		b.WriteString(fmt.Sprintf("echo $! > %q\n", grandchildPath))
	}
	b.WriteString("n=0\n")
	b.WriteString("while :; do\n")
	b.WriteString("  n=$((n+1))\n")
	b.WriteString(`  printf '{"ts":"%s","seq":%d,"state":"running"}\n' "$(date -u +%Y-%m-%dT%H:%M:%SZ)" "$n" > "$WS/heartbeat.json"` + "\n")
	b.WriteString("  sleep 0.05\n")
	b.WriteString("done\n")
	return writeFakeScript(t, "fake-py-beating", b.String())
}

// fakePyStall writes ONE valid heartbeat (seq=7) and then goes silent —
// the heartbeat_timeout fixture.
func fakePyStall(t *testing.T) string {
	t.Helper()
	var b strings.Builder
	b.WriteString("#!/bin/sh\n")
	b.WriteString("export PATH=/usr/bin:/bin:/usr/sbin:/sbin:$PATH\n")
	b.WriteString(wsArgvParser)
	b.WriteString(`printf '{"ts":"%s","seq":7,"state":"running"}\n' "$(date -u +%Y-%m-%dT%H:%M:%SZ)" > "$WS/heartbeat.json"` + "\n")
	b.WriteString("sleep 300\n")
	return writeFakeScript(t, "fake-py-stall", b.String())
}

// fakePyGarbageBeats writes one valid beat (seq=1), then keeps REWRITING the
// file with invalid JSON — mtime keeps advancing (a live process) but the
// content is a half-written/corrupt tail. The supervisor must tolerate the
// parse failures (keep lastSeq=1) and only a real timeout may kill it.
func fakePyGarbageBeats(t *testing.T) string {
	t.Helper()
	var b strings.Builder
	b.WriteString("#!/bin/sh\n")
	b.WriteString("export PATH=/usr/bin:/bin:/usr/sbin:/sbin:$PATH\n")
	b.WriteString(wsArgvParser)
	b.WriteString(`printf '{"ts":"%s","seq":1,"state":"running"}\n' "$(date -u +%Y-%m-%dT%H:%M:%SZ)" > "$WS/heartbeat.json"` + "\n")
	b.WriteString("sleep 0.1\n")
	b.WriteString("while :; do\n")
	b.WriteString(`  printf '{"ts":"%s","seq"' "$(date -u +%Y-%m-%dT%H:%M:%SZ)" > "$WS/heartbeat.json"` + "\n")
	b.WriteString("  sleep 0.05\n")
	b.WriteString("done\n")
	return writeFakeScript(t, "fake-py-garbage", b.String())
}

// fakePyExitNow dies immediately with stderr noise (exit_nonzero fixture).
func fakePyExitNow(t *testing.T) string {
	t.Helper()
	body := "#!/bin/sh\nexport PATH=/usr/bin:/bin:/usr/sbin:/sbin:$PATH\necho \"superopt: boom: bad config\" >&2\nexit 1\n"
	return writeFakeScript(t, "fake-py-exit", body)
}

// fakePyRecorder records its argv, cwd and PYTHONPATH, then sleeps — the
// spawn-contract fixture.
func fakePyRecorder(t *testing.T, argvFile, envFile string) string {
	t.Helper()
	body := fmt.Sprintf("#!/bin/sh\nexport PATH=/usr/bin:/bin:/usr/sbin:/sbin:$PATH\n"+
		"printf '%%s\\n' \"$@\" > %q\npwd > %q\necho \"$PYTHONPATH\" >> %q\nsleep 300\n",
		argvFile, envFile, envFile)
	return writeFakeScript(t, "fake-py-recorder", body)
}

func writeFakeScript(t *testing.T, name, body string) string {
	t.Helper()
	script := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(script, []byte(body), 0755); err != nil {
		t.Fatal(err)
	}
	return script
}

// --- helpers ---

// testSupervisor builds a supervisor with a fast heartbeat ladder.
//
// Timing note: subprocess startup on this machine (go test spawning sh
// fakes while the dev-web instance and parallel job workers run) was
// measured at up to ~300ms before the first heartbeat write, so the
// timeout carries generous headroom (800ms) over the spec's "50ms interval /
// 200ms-竳 timeout" intent — every case still completes in seconds.
func testSupervisor(t *testing.T, pythonPath string, onDead func(string, int64)) *PySupervisor {
	t.Helper()
	return NewPySupervisor(PySupervisorConfig{
		HeartbeatInterval: 50 * time.Millisecond,
		HeartbeatTimeout:  800 * time.Millisecond,
		TermGrace:         400 * time.Millisecond,
		PythonPath:        pythonPath,
		OnDead:            onDead,
	})
}

// newWS makes a workspace directory (the fake scripts write heartbeats there).
func newWS(t *testing.T) string {
	t.Helper()
	ws := filepath.Join(t.TempDir(), "ws")
	if err := os.MkdirAll(ws, 0755); err != nil {
		t.Fatal(err)
	}
	return ws
}

// waitDead polls IsDead until true or the deadline.
func waitDead(t *testing.T, w *PyWorker, d time.Duration) {
	t.Helper()
	deadline := time.Now().Add(d)
	for time.Now().Before(deadline) {
		if w.IsDead() {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("worker %s not dead after %s", w.SessionID(), d)
}

type deadReport struct {
	ws      string
	lastSeq int64
}

// --- tests ---

// TestPySupervisor_Defaults asserts the zero-config cadence matches the
// py-side config.py DEFAULTS (30s / 90s / 5s / python3).
func TestPySupervisor_Defaults(t *testing.T) {
	s := NewPySupervisor(PySupervisorConfig{})
	if s.cfg.HeartbeatInterval != 30*time.Second {
		t.Errorf("HeartbeatInterval default = %v, want 30s", s.cfg.HeartbeatInterval)
	}
	if s.cfg.HeartbeatTimeout != 90*time.Second {
		t.Errorf("HeartbeatTimeout default = %v, want 90s", s.cfg.HeartbeatTimeout)
	}
	if s.cfg.TermGrace != 5*time.Second {
		t.Errorf("TermGrace default = %v, want 5s", s.cfg.TermGrace)
	}
	if s.cfg.PythonPath != "python3" {
		t.Errorf("PythonPath default = %q, want python3", s.cfg.PythonPath)
	}
}

// TestPySupervisor_StartAndHeartbeatsAdvance: fake_py_ok — Start succeeds,
// the worker stays alive, and the heartbeat file updates (seq advances).
func TestPySupervisor_StartAndHeartbeatsAdvance(t *testing.T) {
	pi := fakePyBeating(t, "", "")
	dead := make(chan deadReport, 4)
	sup := testSupervisor(t, pi, func(ws string, seq int64) { dead <- deadReport{ws, seq} })
	defer sup.StopAll()

	ws := newWS(t)
	w, err := sup.Start(ws)
	if err != nil {
		t.Fatalf("Start: %v", err)
	}
	if w.IsDead() {
		t.Fatalf("worker must start alive (reason=%q)", w.Reason())
	}
	if w.SessionID() != ws {
		t.Errorf("SessionID: want %q, got %q", ws, w.SessionID())
	}
	if w.Pid() <= 0 {
		t.Errorf("Pid must be positive after Start, got %d", w.Pid())
	}

	// The heartbeat file must exist and the observed seq must advance
	// (subprocess startup carries headroom under load).
	first := int64(-1)
	deadline := time.Now().Add(4 * time.Second)
	for time.Now().Before(deadline) {
		if hb := w.LastHeartbeat(); hb != nil {
			first = hb.Seq
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if first < 1 {
		t.Fatalf("no heartbeat observed (LastHeartbeat=nil after 3s)")
	}
	advanced := false
	deadline = time.Now().Add(4 * time.Second)
	for time.Now().Before(deadline) {
		if hb := w.LastHeartbeat(); hb != nil && hb.Seq > first {
			advanced = true
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if !advanced {
		t.Fatalf("heartbeat seq never advanced past %d", first)
	}
	if hb := w.LastHeartbeat(); hb != nil && hb.State != "running" {
		t.Errorf("heartbeat state: want running, got %q", hb.State)
	}
	// The file itself is on disk (the py-side contract).
	if _, err := os.Stat(filepath.Join(ws, "heartbeat.json")); err != nil {
		t.Errorf("heartbeat.json missing: %v", err)
	}
}

// TestPySupervisor_CloseKillsWholeGroup: fake_py_ok + grandchild — Close
// must reap the process AND its sleep grandchild (Setpgid + kill(-pgid)).
func TestPySupervisor_CloseKillsWholeGroup(t *testing.T) {
	ws := newWS(t)
	grandPath := filepath.Join(t.TempDir(), "grandchild.pid")
	pi := fakePyBeating(t, grandPath, "")
	sup := testSupervisor(t, pi, nil)
	defer sup.StopAll()

	w, err := sup.Start(ws)
	if err != nil {
		t.Fatalf("Start: %v", err)
	}

	// Wait for the grandchild pid file (subprocess startup headroom).
	var grandPid int
	deadline := time.Now().Add(8 * time.Second)
	for time.Now().Before(deadline) {
		if b, err := os.ReadFile(grandPath); err == nil {
			if n, err := fmt.Sscanf(strings.TrimSpace(string(b)), "%d", &grandPid); err == nil && n == 1 && grandPid > 0 {
				break
			}
		}
		time.Sleep(20 * time.Millisecond)
	}
	if grandPid <= 0 {
		t.Fatal("grandchild pid never appeared")
	}

	if err := w.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	waitDead(t, w, 5*time.Second)

	// Group-kill must also reach the sleep grandchild.
	reaped := false
	deadline = time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if err := syscall.Kill(grandPid, 0); err != nil {
			reaped = true
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	if !reaped {
		t.Errorf("grandchild %d survived the group kill", grandPid)
	}
	// Idempotent second close.
	if err := w.Close(); err != nil {
		t.Errorf("second Close must be a no-op, got %v", err)
	}
}

// TestPySupervisor_HeartbeatTimeoutDeclaresDead: fake_py_stall — one beat,
// then silence → IsDead, Reason=heartbeat_timeout, OnDead carries the last
// seq, exactly once.
func TestPySupervisor_HeartbeatTimeoutDeclaresDead(t *testing.T) {
	pi := fakePyStall(t)
	dead := make(chan deadReport, 4)
	sup := testSupervisor(t, pi, func(ws string, seq int64) { dead <- deadReport{ws, seq} })
	defer sup.StopAll()

	ws := newWS(t)
	w, err := sup.Start(ws)
	if err != nil {
		t.Fatalf("Start: %v", err)
	}

	select {
	case d := <-dead:
		if d.ws != ws {
			t.Errorf("OnDead ws: want %q, got %q", ws, d.ws)
		}
		if d.lastSeq != 7 {
			t.Errorf("OnDead lastSeq: want 7, got %d", d.lastSeq)
		}
	case <-time.After(8 * time.Second):
		t.Fatal("OnDead never fired after heartbeat timeout")
	}

	waitDead(t, w, 3*time.Second)
	if r := w.Reason(); !strings.Contains(r, "heartbeat_timeout") {
		t.Errorf("Reason: want heartbeat_timeout, got %q", r)
	}
	// Exactly-once: no second callback for a while.
	select {
	case d := <-dead:
		t.Fatalf("OnDead fired twice (ws=%q seq=%d)", d.ws, d.lastSeq)
	case <-time.After(300 * time.Millisecond):
	}
}

// TestPySupervisor_ImmediateExit: fake_py_exit — the scheduler dies at
// once → IsDead, Reason=exit_nonzero, OnDead fires.
func TestPySupervisor_ImmediateExit(t *testing.T) {
	pi := fakePyExitNow(t)
	dead := make(chan deadReport, 4)
	sup := testSupervisor(t, pi, func(ws string, seq int64) { dead <- deadReport{ws, seq} })
	defer sup.StopAll()

	ws := newWS(t)
	w, err := sup.Start(ws)
	if err != nil {
		t.Fatalf("Start: %v", err)
	}

	select {
	case d := <-dead:
		if d.ws != ws {
			t.Errorf("OnDead ws: want %q, got %q", ws, d.ws)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("OnDead never fired after immediate exit")
	}
	waitDead(t, w, 3*time.Second)
	if r := w.Reason(); !strings.Contains(r, "exit_nonzero") {
		t.Errorf("Reason: want exit_nonzero, got %q", r)
	}
	// The stderr tail must surface the fake's noise (probe diagnostics).
	if tail := w.StderrTail(0); !strings.Contains(tail, "boom") {
		t.Errorf("StderrTail should contain the crash noise, got %q", tail)
	}
}

// TestPySupervisor_HalfWrittenHeartbeatTolerated: garbage beats keep the
// mtime advancing (a live, actively-writing process) — parse failures must
// NOT kill the worker, and the last VALID seq must survive to OnDead.
func TestPySupervisor_HalfWrittenHeartbeatTolerated(t *testing.T) {
	pi := fakePyGarbageBeats(t)
	dead := make(chan deadReport, 4)
	sup := testSupervisor(t, pi, func(ws string, seq int64) { dead <- deadReport{ws, seq} })
	defer sup.StopAll()

	ws := newWS(t)
	w, err := sup.Start(ws)
	if err != nil {
		t.Fatalf("Start: %v", err)
	}

	// 1.6s is 2× the 800ms timeout: had the parse failure been treated as
	// death (or the garbage tail masked the beats), the worker would already
	// be gone.
	time.Sleep(1600 * time.Millisecond)
	if w.IsDead() {
		t.Fatalf("worker must tolerate half-written heartbeats (reason=%q)", w.Reason())
	}
	// The last valid seq (1) is retained even while garbage flows.
	if hb := w.LastHeartbeat(); hb == nil || hb.Seq != 1 {
		t.Errorf("LastHeartbeat: want seq 1 retained, got %+v", hb)
	}

	w.Close()
	select {
	case d := <-dead:
		if d.lastSeq != 1 {
			t.Errorf("OnDead lastSeq: want 1 (last VALID seq), got %d", d.lastSeq)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("OnDead never fired after Close")
	}
}

// TestPySupervisor_StaleHeartbeatIsNotLife: a heartbeat.json left over from
// a PREVIOUS run (mtime before spawn) must not extend the stall worker's
// life beyond the timeout.
func TestPySupervisor_StaleHeartbeatIsNotLife(t *testing.T) {
	// A mute fake that never writes a heartbeat (the stale file on disk is
	// the ONLY heartbeat the supervisor ever sees — and it predates spawn).
	var b strings.Builder
	b.WriteString("#!/bin/sh\nexport PATH=/usr/bin:/bin:/usr/sbin:/sbin:$PATH\n")
	b.WriteString(wsArgvParser)
	b.WriteString("sleep 300\n")
	mute := writeFakeScript(t, "fake-py-mute", b.String())

	dead := make(chan deadReport, 4)
	sup := testSupervisor(t, mute, func(ws string, seq int64) { dead <- deadReport{ws, seq} })
	defer sup.StopAll()

	ws := newWS(t)
	// Pre-existing heartbeat from an earlier run, mtime an hour old.
	stale := filepath.Join(ws, "heartbeat.json")
	old := time.Now().Add(-time.Hour)
	if err := os.WriteFile(stale, []byte(`{"ts":"2026-10-17T00:00:00Z","seq":99,"state":"running"}`), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(stale, old, old); err != nil {
		t.Fatal(err)
	}

	w, err := sup.Start(ws)
	if err != nil {
		t.Fatalf("Start: %v", err)
	}
	select {
	case <-dead:
		// died by timeout — good
	case <-time.After(6 * time.Second):
		t.Fatal("stale heartbeat kept a mute worker alive past the timeout")
	}
	if r := w.Reason(); !strings.Contains(r, "heartbeat_timeout") {
		t.Errorf("Reason: want heartbeat_timeout, got %q", r)
	}
}

// TestPySupervisor_CloseGracefulTermNoEscalation: TERM handled within the
// grace window must NOT escalate to SIGKILL (Close returns well inside
// TermGrace).
func TestPySupervisor_CloseGracefulTermNoEscalation(t *testing.T) {
	pi := fakePyBeating(t, "", "exit 0")
	sup := NewPySupervisor(PySupervisorConfig{
		HeartbeatInterval: 50 * time.Millisecond,
		HeartbeatTimeout:  800 * time.Millisecond,
		TermGrace:         1500 * time.Millisecond,
		PythonPath:        pi,
	})
	defer sup.StopAll()

	ws := newWS(t)
	w, err := sup.Start(ws)
	if err != nil {
		t.Fatalf("Start: %v", err)
	}
	// Let at least one heartbeat land so the fake is inside its loop.
	time.Sleep(150 * time.Millisecond)

	start := time.Now()
	if err := w.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	if elapsed := time.Since(start); elapsed >= 1500*time.Millisecond {
		t.Errorf("graceful close took %s — TERM was not honored (SIGKILL escalation?)", elapsed)
	}
	waitDead(t, w, 3*time.Second)
	if r := w.Reason(); r != "closed" {
		t.Errorf("Reason after Close: want closed, got %q", r)
	}
}

// TestPySupervisor_CloseForceKillAfterGrace: a TERM-immune scheduler must
// be reaped by the SIGKILL rung once TermGrace lapses.
func TestPySupervisor_CloseForceKillAfterGrace(t *testing.T) {
	pi := fakePyBeating(t, "", ":") // ignore TERM
	sup := NewPySupervisor(PySupervisorConfig{
		HeartbeatInterval: 50 * time.Millisecond,
		HeartbeatTimeout:  800 * time.Millisecond,
		TermGrace:         400 * time.Millisecond,
		PythonPath:        pi,
	})
	defer sup.StopAll()

	ws := newWS(t)
	w, err := sup.Start(ws)
	if err != nil {
		t.Fatalf("Start: %v", err)
	}
	start := time.Now()
	if err := w.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	elapsed := time.Since(start)
	// TERM was ignored → the ladder must have waited ≈ TermGrace then KILLed.
	if elapsed < 350*time.Millisecond {
		t.Logf("force kill took %s (expected ≈ TermGrace 400ms; log only)", elapsed)
	}
	waitDead(t, w, 3*time.Second)
	if !w.IsDead() {
		t.Fatal("TERM-immune worker must be dead after the SIGKILL rung")
	}
}

// TestPySupervisor_SpawnContract: argv must be exactly
// [-m superopt --workdir <ws>], cwd the resolved repo root, PYTHONPATH
// anchored at <repoRoot>/super (module resolution contract).
func TestPySupervisor_SpawnContract(t *testing.T) {
	dir := t.TempDir()
	argvFile := filepath.Join(dir, "argv.txt")
	envFile := filepath.Join(dir, "env.txt")
	pi := fakePyRecorder(t, argvFile, envFile)
	sup := NewPySupervisor(PySupervisorConfig{
		HeartbeatInterval: 50 * time.Millisecond,
		HeartbeatTimeout:  10 * time.Second, // keep the recorder alive
		PythonPath:        pi,
	})
	defer sup.StopAll()

	ws := newWS(t)
	w, err := sup.Start(ws)
	if err != nil {
		t.Fatalf("Start: %v", err)
	}
	defer w.Close()

	var argv []string
	var pwd, pythonPath string
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if b, err := os.ReadFile(argvFile); err == nil && len(b) > 0 {
			argv = strings.Fields(string(b))
		}
		if b, err := os.ReadFile(envFile); err == nil {
			if lines := strings.Split(strings.TrimSpace(string(b)), "\n"); len(lines) >= 2 {
				pwd, pythonPath = lines[0], lines[1]
				break
			}
		}
		time.Sleep(20 * time.Millisecond)
	}
	if len(argv) < 4 {
		t.Fatalf("argv never recorded: %v", argv)
	}
	want := []string{"-m", "superopt", "--workdir", ws}
	if !equalStrings(argv, want) {
		t.Errorf("argv: want %v, got %v", want, argv)
	}
	if pwd != resolveRepoRoot() {
		t.Errorf("subprocess cwd: want %q, got %q", resolveRepoRoot(), pwd)
	}
	if pythonPath == "" {
		t.Errorf("PYTHONPATH must be set for -m superopt resolution")
	} else if !strings.HasSuffix(pythonPath, string(filepath.Separator)+"super") {
		t.Errorf("PYTHONPATH must end in /super, got %q", pythonPath)
	}
}

// TestPySupervisor_MultiWorkspace: concurrent ws → Count/Get/StopAll.
func TestPySupervisor_MultiWorkspace(t *testing.T) {
	pi := fakePyBeating(t, "", "")
	sup := testSupervisor(t, pi, nil)

	wsA, wsB := newWS(t), newWS(t)
	wA, err := sup.Start(wsA)
	if err != nil {
		t.Fatalf("Start A: %v", err)
	}
	wB, err := sup.Start(wsB)
	if err != nil {
		t.Fatalf("Start B: %v", err)
	}
	if n := sup.Count(); n != 2 {
		t.Fatalf("Count: want 2, got %d", n)
	}
	for _, tc := range []struct {
		ws string
		w  *PyWorker
	}{{wsA, wA}, {wsB, wB}} {
		got, ok := sup.Get(tc.ws)
		if !ok || got != tc.w {
			t.Errorf("Get(%q) must return the spawned worker", tc.ws)
		}
	}
	if _, ok := sup.Get(filepath.Join(t.TempDir(), "nope")); ok {
		t.Error("Get on an unknown ws must miss")
	}

	sup.StopAll()
	waitDead(t, wA, 5*time.Second)
	waitDead(t, wB, 5*time.Second)
	if n := sup.Count(); n != 0 {
		t.Errorf("Count after StopAll: want 0, got %d", n)
	}
	if _, ok := sup.Get(wsA); ok {
		t.Error("StopAll must clear the registry")
	}
	// StopAll on an empty supervisor is a no-op.
	sup.StopAll()
}

// TestPySupervisor_DuplicateLiveRejectedDeadReplaced: a LIVE duplicate ws
// is rejected; a DEAD one is replaced (the OnDead respawn path).
func TestPySupervisor_DuplicateLiveRejectedDeadReplaced(t *testing.T) {
	pi := fakePyBeating(t, "", "")
	sup := testSupervisor(t, pi, nil)
	defer sup.StopAll()

	ws := newWS(t)
	w1, err := sup.Start(ws)
	if err != nil {
		t.Fatalf("first Start: %v", err)
	}
	if _, err := sup.Start(ws); !errors.Is(err, ErrLiveWorkerExists) {
		t.Fatalf("second Start while live: want ErrLiveWorkerExists, got %v", err)
	}
	if n := sup.Count(); n != 1 {
		t.Errorf("Count after rejected duplicate: want 1, got %d", n)
	}

	// Kill it; the dead entry must not block a respawn.
	if err := w1.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	waitDead(t, w1, 5*time.Second)

	w2, err := sup.Start(ws)
	if err != nil {
		t.Fatalf("respawn after death must be allowed: %v", err)
	}
	got, ok := sup.Get(ws)
	if !ok || got != w2 {
		t.Error("Get after respawn must return the NEW worker")
	}
	if n := sup.Count(); n != 1 {
		t.Errorf("Count after respawn: want 1, got %d", n)
	}
}

// TestPySupervisor_ConcurrentStarts: parallel Starts of distinct
// workspaces register exactly N workers (registry race check).
func TestPySupervisor_ConcurrentStarts(t *testing.T) {
	pi := fakePyBeating(t, "", "")
	sup := testSupervisor(t, pi, nil)
	defer sup.StopAll()

	const n = 6
	var wg sync.WaitGroup
	errs := make([]error, n)
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			_, errs[i] = sup.Start(newWS(t))
		}(i)
	}
	wg.Wait()
	for i, err := range errs {
		if err != nil {
			t.Fatalf("Start %d: %v", i, err)
		}
	}
	if got := sup.Count(); got != n {
		t.Errorf("Count: want %d, got %d", n, got)
	}
}

// TestPySupervisor_EmptyWsRejected: Start("") must fail without spawning.
func TestPySupervisor_EmptyWsRejected(t *testing.T) {
	sup := testSupervisor(t, "/nonexistent/python", nil)
	defer sup.StopAll()
	if _, err := sup.Start(""); !errors.Is(err, ErrEmptyWs) {
		t.Fatalf("Start(\"\"): want ErrEmptyWs, got %v", err)
	}
}

func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
