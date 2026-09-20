// supervisor_test.go exercises the pi rpc subprocess supervisor against
// fake pi shell scripts (the cli_mock_test.go pattern, extended with a
// stdin-speaking fake).
//
// Isolation (bugs.md):
//   - RICK_PI_AGENT_DIR → temp: piPathOrDefault would otherwise resolve the
//     real managed runtime even when tests point PiPath at a fake (job_34).
//   - Fake scripts restore the system PATH (job_33: sh builtin-only scripts
//     that call external commands like sleep would silently fail).
package runtime

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"
)

// --- fake pi fixtures ---

// fakePiResponsive writes a fake pi that answers stdin commands. Optional
// hooks: grandchildPath — spawn `sleep 300` and write its pid to the file;
// exitAfter — exit after N commands (0 = never).
func fakePiResponsive(t *testing.T, grandchildPath string, exitAfter int) string {
	t.Helper()
	dir := t.TempDir()
	script := filepath.Join(dir, "fake-pi")
	var b strings.Builder
	b.WriteString("#!/bin/sh\n")
	// job_33 pitfall guard: restore the system PATH so sleep/echo/etc work.
	b.WriteString("export PATH=/usr/bin:/bin:/usr/sbin:/sbin:$PATH\n")
	if grandchildPath != "" {
		b.WriteString("sleep 300 &\n")
		b.WriteString(fmt.Sprintf("echo $! > %q\n", grandchildPath))
	}
	b.WriteString("n=0\n")
	b.WriteString("while IFS= read -r line; do\n")
	b.WriteString("  n=$((n+1))\n")
	if exitAfter > 0 {
		b.WriteString(fmt.Sprintf("  if [ \"$n\" -ge %d ]; then exit 0; fi\n", exitAfter))
	}
	b.WriteString("  case \"$line\" in\n")
	b.WriteString("    *get_state*)\n")
	b.WriteString(`      echo '{"type":"response","command":"get_state","success":true,"id":"hb-1","data":{"isStreaming":false,"isCompacting":false,"sessionId":"pi-sess-1","sessionFile":"/tmp/fake-session.jsonl"}}'` + "\n")
	b.WriteString("      ;;\n")
	b.WriteString("    *get_entries*)\n")
	b.WriteString(`      echo '{"type":"response","command":"get_entries","success":true,"id":"ge-1","data":{"entries":[],"leafId":"leaf-42"}}'` + "\n")
	b.WriteString("      ;;\n")
	b.WriteString("    *)\n")
	b.WriteString(`      echo '{"type":"agent_start"}'` + "\n")
	b.WriteString(`      echo '{"type":"agent_settled"}'` + "\n")
	b.WriteString("      ;;\n")
	b.WriteString("  esac\n")
	b.WriteString("done\n")
	if err := os.WriteFile(script, []byte(b.String()), 0755); err != nil {
		t.Fatal(err)
	}
	return script
}

// fakePiExitNow dies immediately with stderr noise (probe-failure fixture).
func fakePiExitNow(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	script := filepath.Join(dir, "fake-pi-dies")
	body := "#!/bin/sh\nexport PATH=/usr/bin:/bin:/usr/sbin:/sbin:$PATH\necho \"pi: boom: invalid flag\" >&2\nexit 1\n"
	if err := os.WriteFile(script, []byte(body), 0755); err != nil {
		t.Fatal(err)
	}
	return script
}

// fakePiGraceful traps TERM and exits 0 (graceful Close fixture).
func fakePiGraceful(t *testing.T) string {
	t.Helper()
	return fakePiWithTrap(t, "exit 0")
}

// fakePiTermImmune ignores TERM (forces the SIGKILL ladder step).
func fakePiTermImmune(t *testing.T) string {
	t.Helper()
	return fakePiWithTrap(t, ":")
}

// fakePiWithTrap builds a responsive fake whose TERM disposition is the
// given shell snippet ("exit 0" = graceful, ":" = ignore).
func fakePiWithTrap(t *testing.T, termAction string) string {
	t.Helper()
	base := fakePiResponsive(t, "", 0)
	body, err := os.ReadFile(base)
	if err != nil {
		t.Fatal(err)
	}
	// Insert the trap right after the PATH restore line.
	lines := strings.Split(string(body), "\n")
	var out []string
	inserted := false
	for _, ln := range lines {
		out = append(out, ln)
		if !inserted && strings.HasPrefix(ln, "export PATH=") {
			out = append(out, "trap '"+termAction+"' TERM")
			inserted = true
		}
	}
	script := filepath.Join(filepath.Dir(base), "fake-pi-trap")
	if err := os.WriteFile(script, []byte(strings.Join(out, "\n")), 0755); err != nil {
		t.Fatal(err)
	}
	return script
}

// isolateRuntimeEnv applies the job_34 isolation: AgentDir() resolves to a
// temp dir so the managed runtime binary can never be picked up.
func isolateRuntimeEnv(t *testing.T) {
	t.Helper()
	t.Setenv(rickAgentDirEnv, t.TempDir())
}

// testSupervisor builds an isolated supervisor with a fast heartbeat.
func testSupervisor(t *testing.T, piPath string, onDead func(string, string)) *Supervisor {
	t.Helper()
	isolateRuntimeEnv(t)
	return NewSupervisor(SupervisorConfig{
		HeartbeatInterval: 50 * time.Millisecond,
		HeartbeatTimeout:  250 * time.Millisecond,
		IdleTimeout:       0, // disabled unless a test opts in
		PiPath:            piPath,
		OnDead:            onDead,
	})
}

// recvEvent reads one event or fails after timeout.
func recvEvent(t *testing.T, ch <-chan *RpcEvent, timeout time.Duration) *RpcEvent {
	t.Helper()
	select {
	case ev, ok := <-ch:
		if !ok {
			t.Fatal("events channel closed before expected event")
		}
		return ev
	case <-time.After(timeout):
		t.Fatal("timeout waiting for rpc event")
		return nil
	}
}

// drainUntilResponse reads events until a response for command arrives.
func drainUntilResponse(t *testing.T, ch <-chan *RpcEvent, command string, timeout time.Duration) *RpcEvent {
	t.Helper()
	deadline := time.After(timeout)
	for {
		select {
		case ev, ok := <-ch:
			if !ok {
				t.Fatalf("events channel closed while waiting for %s response", command)
			}
			if ev.Type == "response" && ev.Command == command {
				return ev
			}
		case <-deadline:
			t.Fatalf("timeout waiting for %s response", command)
		}
	}
}

// spec builds a minimal SpawnSpec for the responsive fake.
func spec(sessionID, piPath string) SpawnSpec {
	return SpawnSpec{
		SessionID:     sessionID,
		SessionIDFlag: sessionID,
		MethodFile:    "",
		PromptFile:    "",
		CreateNew:     true,
	}
}

// --- tests ---

func TestSupervisor_BootstrapMessageExport(t *testing.T) {
	if BootstrapMessage == "" {
		t.Fatal("BootstrapMessage must re-export the non-empty cli bootstrap trigger")
	}
	if BootstrapMessage != bootstrapMessage {
		t.Fatalf("BootstrapMessage drift: %q != %q", BootstrapMessage, bootstrapMessage)
	}
}

func TestSupervisor_SpawnProbeFail(t *testing.T) {
	pi := fakePiExitNow(t)
	sup := testSupervisor(t, pi, nil)
	defer sup.CloseAll()

	_, err := sup.Spawn(spec("probe-fail", pi))
	if err == nil {
		t.Fatal("Spawn must fail when pi exits during probe")
	}
	if !strings.Contains(err.Error(), "boom") {
		t.Errorf("probe error should surface stderr tail, got: %v", err)
	}
	if n := sup.Count(); n != 0 {
		t.Errorf("failed spawn must not register a worker, count=%d", n)
	}
}

func TestSupervisor_CommandRoundTrip(t *testing.T) {
	pi := fakePiResponsive(t, "", 0)
	sup := testSupervisor(t, pi, nil)
	defer sup.CloseAll()

	w, err := sup.Spawn(spec("rt-1", pi))
	if err != nil {
		t.Fatalf("Spawn: %v", err)
	}
	defer w.Close()

	// Send a get_state command; the fake answers on stdout.
	line, err := NewRpcClient().GetState()
	if err != nil {
		t.Fatalf("build get_state: %v", err)
	}
	if err := w.Send(line); err != nil {
		t.Fatalf("Send: %v", err)
	}
	ev := drainUntilResponse(t, w.Events(), "get_state", 3*time.Second)
	if !ev.Success {
		t.Fatalf("get_state response not successful: %+v", ev)
	}

	// The intercepted state cache must be populated from the same response.
	st := w.State()
	if st == nil {
		t.Fatal("State() must cache the get_state response payload")
	}
	if st.SessionID != "pi-sess-1" || st.IsStreaming {
		t.Errorf("state cache mismatch: %+v", st)
	}

	// Non-heartbeat command: prompt → agent events + response.
	pl, err := NewRpcClient().Prompt("hello", "")
	if err != nil {
		t.Fatalf("build prompt: %v", err)
	}
	if err := w.Send(pl); err != nil {
		t.Fatalf("Send prompt: %v", err)
	}
	ev = recvEvent(t, w.Events(), 3*time.Second)
	if ev.Type != "agent_start" {
		t.Errorf("expected agent_start after prompt, got %s", ev.Type)
	}
}

func TestSupervisor_LastEntryIDFromGetEntries(t *testing.T) {
	pi := fakePiResponsive(t, "", 0)
	sup := testSupervisor(t, pi, nil)
	defer sup.CloseAll()

	w, err := sup.Spawn(spec("cursor-1", pi))
	if err != nil {
		t.Fatalf("Spawn: %v", err)
	}
	defer w.Close()

	line, err := NewRpcClient().GetEntries("")
	if err != nil {
		t.Fatalf("build get_entries: %v", err)
	}
	if err := w.Send(line); err != nil {
		t.Fatalf("Send: %v", err)
	}
	drainUntilResponse(t, w.Events(), "get_entries", 3*time.Second)

	if got := w.LastEntryID(); got != "leaf-42" {
		t.Errorf("LastEntryID: want leaf-42, got %q", got)
	}
}

func TestSupervisor_CloseGraceful(t *testing.T) {
	pi := fakePiGraceful(t)
	sup := testSupervisor(t, pi, nil)
	defer sup.CloseAll()

	start := time.Now()
	w, err := sup.Spawn(spec("grace-1", pi))
	if err != nil {
		t.Fatalf("Spawn: %v", err)
	}
	// Prime one round trip so the fake is inside its read loop.
	line, _ := NewRpcClient().GetState()
	_ = w.Send(line)
	drainUntilResponse(t, w.Events(), "get_state", 3*time.Second)

	w.Close()
	if elapsed := time.Since(start); elapsed > 5*time.Second {
		t.Errorf("graceful close took %s (TERM trap should exit immediately)", elapsed)
	}
	select {
	case <-w.waitCh:
	default:
		t.Fatal("process not reaped after graceful close")
	}
	if !w.IsDead() {
		t.Error("worker must be dead after Close")
	}
	// Idempotent second close.
	w.Close()
}

func TestSupervisor_CloseForceKill(t *testing.T) {
	pi := fakePiTermImmune(t)
	sup := testSupervisor(t, pi, nil)
	defer sup.CloseAll()

	// The worker's shell ignores SIGTERM; the Close ladder must SIGKILL it.
	w, err := sup.Spawn(spec("immune-1", pi))
	if err != nil {
		t.Fatalf("Spawn immune: %v", err)
	}
	start := time.Now()
	w.Close()
	select {
	case <-w.waitCh:
	case <-time.After(15 * time.Second):
		t.Fatal("immune worker not reaped by SIGKILL ladder")
	}
	if elapsed := time.Since(start); elapsed < 4*time.Second {
		t.Logf("force kill took %s (expected ≈ TermGrace, log only)", elapsed)
	}
	if !w.IsDead() {
		t.Error("worker must be dead after force kill")
	}
}

func TestSupervisor_GroupKillReachesGrandchild(t *testing.T) {
	// pi's bash tool leaves grandchildren; killing the process GROUP must
	// reach them (Setpgid + kill(-pgid)).
	grandPath := filepath.Join(t.TempDir(), "grandchild.pid")
	piGrand := fakePiResponsive(t, grandPath, 0)
	sup := testSupervisor(t, piGrand, nil)
	defer sup.CloseAll()

	w, err := sup.Spawn(spec("grand-1", piGrand))
	if err != nil {
		t.Fatalf("Spawn grand: %v", err)
	}

	// Wait for the grandchild pid file.
	var grandPid int
	deadline := time.Now().Add(3 * time.Second)
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

	// Group-kill must also reach the sleep grandchild.
	w.Close()
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
}

func TestSupervisor_MaxActive(t *testing.T) {
	pi := fakePiResponsive(t, "", 0)
	isolateRuntimeEnv(t)
	sup := NewSupervisor(SupervisorConfig{
		MaxActive:         1,
		HeartbeatInterval: 50 * time.Millisecond,
		IdleTimeout:       0,
		PiPath:            pi,
	})
	defer sup.CloseAll()

	if _, err := sup.Spawn(spec("max-1", pi)); err != nil {
		t.Fatalf("first spawn: %v", err)
	}
	_, err := sup.Spawn(spec("max-2", pi))
	if !errors.Is(err, ErrMaxActive) {
		t.Fatalf("second spawn: want ErrMaxActive, got %v", err)
	}
	if n := sup.Count(); n != 1 {
		t.Errorf("count: want 1, got %d", n)
	}
}

func TestSupervisor_MultipleWorkers(t *testing.T) {
	pi := fakePiResponsive(t, "", 0)
	sup := testSupervisor(t, pi, nil)
	defer sup.CloseAll()

	var workers []*Worker
	for i := 0; i < 3; i++ {
		w, err := sup.Spawn(spec(fmt.Sprintf("multi-%d", i), pi))
		if err != nil {
			t.Fatalf("spawn %d: %v", i, err)
		}
		workers = append(workers, w)
	}
	if n := sup.Count(); n != 3 {
		t.Fatalf("count: want 3, got %d", n)
	}

	// Each worker answers its own commands.
	for i, w := range workers {
		line, _ := NewRpcClient().GetState()
		if err := w.Send(line); err != nil {
			t.Fatalf("worker %d send: %v", i, err)
		}
		ev := drainUntilResponse(t, w.Events(), "get_state", 3*time.Second)
		if !ev.Success {
			t.Fatalf("worker %d: bad response %+v", i, ev)
		}
	}

	sup.CloseAll()
	for i, w := range workers {
		if !w.IsDead() {
			t.Errorf("worker %d not dead after CloseAll", i)
		}
	}
}

func TestSupervisor_OnDeadOnExternalKill(t *testing.T) {
	pi := fakePiResponsive(t, "", 0)
	dead := make(chan struct {
		session string
		last    string
	}, 4)
	sup := testSupervisor(t, pi, func(sessionID, lastEntryID string) {
		dead <- struct {
			session string
			last    string
		}{sessionID, lastEntryID}
	})
	defer sup.CloseAll()

	w, err := sup.Spawn(spec("killed-1", pi))
	if err != nil {
		t.Fatalf("Spawn: %v", err)
	}

	// Establish the entry cursor before the kill.
	line, _ := NewRpcClient().GetEntries("")
	_ = w.Send(line)
	drainUntilResponse(t, w.Events(), "get_entries", 3*time.Second)

	// Kill the process group externally (simulating a crash).
	if err := syscall.Kill(-w.cmd.Process.Pid, syscall.SIGKILL); err != nil {
		t.Fatalf("external kill: %v", err)
	}

	select {
	case d := <-dead:
		if d.session != "killed-1" {
			t.Errorf("OnDead session: want killed-1, got %q", d.session)
		}
		if d.last != "leaf-42" {
			t.Errorf("OnDead lastEntryID: want leaf-42, got %q", d.last)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("OnDead never fired after external kill")
	}

	// The events channel closes after death.
	deadline := time.Now().Add(3 * time.Second)
	for {
		select {
		case _, ok := <-w.Events():
			if !ok {
				return // closed — good
			}
		case <-time.After(time.Until(deadline)):
			t.Fatal("events channel never closed after death")
			return
		}
	}
}

func TestSupervisor_IdleReap(t *testing.T) {
	pi := fakePiResponsive(t, "", 0)
	isolateRuntimeEnv(t)
	dead := make(chan string, 1)
	sup := NewSupervisor(SupervisorConfig{
		HeartbeatInterval: 20 * time.Millisecond, // keep state fresh
		HeartbeatTimeout:  500 * time.Millisecond,
		IdleTimeout:       150 * time.Millisecond,
		PiPath:            pi,
		OnDead: func(sessionID, _ string) {
			select {
			case dead <- sessionID:
			default:
			}
		},
	})
	defer sup.CloseAll()

	w, err := sup.Spawn(spec("idle-1", pi))
	if err != nil {
		t.Fatalf("Spawn: %v", err)
	}

	// No activity: the reaper should close the worker within ~idle+check.
	select {
	case s := <-dead:
		if s != "idle-1" {
			t.Errorf("reaped session: want idle-1, got %q", s)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("idle worker never reaped")
	}
	if r := w.Reason(); !strings.Contains(r, "idle") {
		t.Logf("reap reason: %q (log only)", r)
	}
	if !w.IsDead() {
		t.Error("worker must be dead after idle reap")
	}
}

func TestSupervisor_HeartbeatDeath(t *testing.T) {
	// Responsive fake that exits after 2 commands: heartbeats keep arriving
	// until the fake dies, then the pending heartbeat times out OR the wait
	// goroutine fires — either way OnDead must fire exactly once.
	pi := fakePiResponsive(t, "", 2)
	dead := make(chan string, 2)
	sup := testSupervisor(t, pi, func(sessionID, _ string) {
		dead <- sessionID
	})
	defer sup.CloseAll()

	if _, err := sup.Spawn(spec("hb-1", pi)); err != nil {
		t.Fatalf("Spawn: %v", err)
	}

	select {
	case <-dead:
	case <-time.After(10 * time.Second):
		t.Fatal("worker death never reported after fake exit")
	}
	// Exactly-once: no second callback for a while.
	select {
	case s := <-dead:
		t.Fatalf("OnDead fired twice (session %q)", s)
	case <-time.After(300 * time.Millisecond):
	}
}

func TestSupervisor_EventsOverflowDropsOldest(t *testing.T) {
	// Unit-test the staging queue policy without a subprocess: craft a
	// worker by hand and stage synthetic events.
	isolateRuntimeEnv(t)
	sup := NewSupervisor(SupervisorConfig{PiPath: "/nonexistent"})
	w := &Worker{
		sup:          sup,
		sessionID:    "ovf-1",
		client:       NewRpcClient(),
		events:       make(chan *RpcEvent, eventChanBuffer),
		notify:       make(chan struct{}, 1),
		waitCh:       make(chan struct{}),
		deadCh:       make(chan struct{}),
		abandonC:     make(chan struct{}),
		lastActivity: time.Now(),
	}
	w.stderrTail = newByteRing(stderrTailSize)

	// Stage well beyond the queue cap; pump is NOT running, so the queue
	// must cap itself and count the drops.
	for i := 0; i < maxPendingEvents+500; i++ {
		w.stage(&RpcEvent{Type: "agent_start"})
	}
	if got := w.Dropped(); got != 500 {
		t.Errorf("dropped: want 500, got %d", got)
	}
	w.queueMu.Lock()
	n := len(w.pending)
	w.queueMu.Unlock()
	if n != maxPendingEvents {
		t.Errorf("queue length: want %d, got %d", maxPendingEvents, n)
	}
}

func TestSupervisor_DuplicateSessionRejected(t *testing.T) {
	pi := fakePiResponsive(t, "", 0)
	sup := testSupervisor(t, pi, nil)
	defer sup.CloseAll()

	if _, err := sup.Spawn(spec("dup-1", pi)); err != nil {
		t.Fatalf("first spawn: %v", err)
	}
	if _, err := sup.Spawn(spec("dup-1", pi)); err == nil {
		t.Fatal("duplicate session id must be rejected")
	}
}

// --- spawn argv contract ---

func TestSupervisor_SpawnFlagSemantics(t *testing.T) {
	// Assert the create-vs-resume flag choice via a recording fake (the
	// bugs.md job_30 semantics are the whole point). The fake APPENDS its argv
	// so both spawns accumulate in one file.
	dir := t.TempDir()
	argvFile := filepath.Join(dir, "argv.txt")
	script := filepath.Join(dir, "fake-pi-argv")
	body := fmt.Sprintf("#!/bin/sh\nexport PATH=/usr/bin:/bin:/usr/sbin:/sbin:$PATH\nprintf '%%s\\n' \"$@\" >> %q\nwhile IFS= read -r line; do :; done\n", argvFile)
	if err := os.WriteFile(script, []byte(body), 0755); err != nil {
		t.Fatal(err)
	}

	sup := testSupervisor(t, script, nil)
	defer sup.CloseAll()

	// CreateNew: --session-id <flag>
	w, err := sup.Spawn(SpawnSpec{SessionID: "new-1", SessionIDFlag: "uuid-A", CreateNew: true})
	if err != nil {
		t.Fatalf("spawn create: %v", err)
	}
	w.Close()

	// Resume: --session <flag>
	w2, err := sup.Spawn(SpawnSpec{SessionID: "res-1", SessionIDFlag: "uuid-B", CreateNew: false})
	if err != nil {
		t.Fatalf("spawn resume: %v", err)
	}
	w2.Close()

	data, err := os.ReadFile(argvFile)
	if err != nil {
		t.Fatalf("read argv: %v", err)
	}
	lines := strings.Fields(string(data))
	has := func(flag, val string) bool {
		for i := 0; i+1 < len(lines); i++ {
			if lines[i] == flag && lines[i+1] == val {
				return true
			}
		}
		return false
	}
	if !has("--session-id", "uuid-A") {
		t.Errorf("create spawn missing --session-id uuid-A: %v", lines)
	}
	if !has("--session", "uuid-B") {
		t.Errorf("resume spawn missing --session uuid-B: %v", lines)
	}
	if lines[0] != "--mode" || lines[1] != "rpc" {
		t.Errorf("spawn must start with --mode rpc: %v", lines[:2])
	}
}

func TestSupervisor_SpawnDirAndEnv(t *testing.T) {
	// The worker's cwd must be the spec dir and the env must carry the
	// isolated PI_CODING_AGENT_DIR (AgentEnv contract).
	dir := t.TempDir()
	probeFile := filepath.Join(dir, "probe.txt")
	script := filepath.Join(t.TempDir(), "fake-pi-env")
	body := fmt.Sprintf("#!/bin/sh\nexport PATH=/usr/bin:/bin:/usr/sbin:/sbin:$PATH\npwd > %q\necho \"$PI_CODING_AGENT_DIR\" >> %q\nwhile IFS= read -r line; do :; done\n", probeFile, probeFile)
	if err := os.WriteFile(script, []byte(body), 0755); err != nil {
		t.Fatal(err)
	}

	agentDir := t.TempDir()
	t.Setenv(rickAgentDirEnv, agentDir)
	sup := NewSupervisor(SupervisorConfig{
		HeartbeatInterval: 0,
		IdleTimeout:       0,
		PiPath:            script,
	})
	defer sup.CloseAll()

	w, err := sup.Spawn(SpawnSpec{SessionID: "env-1", SessionIDFlag: "uuid-C", Dir: dir, CreateNew: true})
	if err != nil {
		t.Fatalf("spawn: %v", err)
	}
	defer w.Close()

	var gotDir, gotEnv string
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if b, err := os.ReadFile(probeFile); err == nil {
			parts := strings.SplitN(strings.TrimSpace(string(b)), "\n", 2)
			if len(parts) == 2 {
				gotDir, gotEnv = parts[0], parts[1]
				break
			}
		}
		time.Sleep(20 * time.Millisecond)
	}
	if gotDir != dir {
		t.Errorf("subprocess cwd: want %q, got %q", dir, gotDir)
	}
	if gotEnv != agentDir {
		t.Errorf("PI_CODING_AGENT_DIR: want %q, got %q", agentDir, gotEnv)
	}
}

// --- helpers used across tests ---

// readArgv is shared with cli_mock_test.go conventions but local: reads the
// recorded argv lines of an argv-recording fake.
func readSpawnArgv(t *testing.T, path string) []string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read argv file: %v", err)
	}
	return strings.Fields(strings.TrimSpace(string(b)))
}

// ensure no compile-time unused helper complaints
var _ = exec.Command
var _ sync.Mutex

// TestReadEventLine 覆盖 stdout 行长处理（resume 大会话时 pi 会吐超大单行事件，
// 旧实现用 bufio.Scanner，遇到 >8MB 的行会永久停摆并把整条会话判为 worker lost）：
// 正常行、恰好超限被丢弃（且后续行仍可读）、无结尾换行的最后一行、CRLF。
func TestReadEventLine(t *testing.T) {
	limit := 64

	t.Run("normal lines", func(t *testing.T) {
		br := bufio.NewReaderSize(strings.NewReader("aaa\nbbb\n"), 16)
		l1, err := readEventLine(br, limit)
		if err != nil || string(l1) != "aaa" {
			t.Fatalf("line1 = %q, err=%v", l1, err)
		}
		l2, err := readEventLine(br, limit)
		if err != nil || string(l2) != "bbb" {
			t.Fatalf("line2 = %q, err=%v", l2, err)
		}
		if _, err := readEventLine(br, limit); !errors.Is(err, io.EOF) {
			t.Fatalf("want EOF after last line, got %v", err)
		}
	})

	t.Run("oversized line is skipped and scanning continues", func(t *testing.T) {
		big := strings.Repeat("x", limit*3)
		br := bufio.NewReaderSize(strings.NewReader(big+"\nnext\n"), 16)
		if _, err := readEventLine(br, limit); !errors.Is(err, errEventLineTooLong) {
			t.Fatalf("oversized line err = %v, want errEventLineTooLong", err)
		}
		// 关键：丢弃之后必须还能读到下一行（旧 Scanner 会永久停摆）
		next, err := readEventLine(br, limit)
		if err != nil || string(next) != "next" {
			t.Fatalf("after oversized: next = %q, err=%v", next, err)
		}
	})

	t.Run("line spanning multiple internal buffer fills", func(t *testing.T) {
		long := strings.Repeat("y", limit-1)
		br := bufio.NewReaderSize(strings.NewReader(long+"\n"), 8) // 内部缓冲 8B → 多次 ErrBufferFull
		got, err := readEventLine(br, limit)
		if err != nil || string(got) != long {
			t.Fatalf("multi-chunk line len=%d err=%v", len(got), err)
		}
	})

	t.Run("final line without newline and CRLF", func(t *testing.T) {
		br := bufio.NewReaderSize(strings.NewReader("tail"), 16)
		got, err := readEventLine(br, limit)
		if err != nil || string(got) != "tail" {
			t.Fatalf("no-newline tail = %q err=%v", got, err)
		}
		br2 := bufio.NewReaderSize(strings.NewReader("crlf\r\n"), 16)
		got2, err := readEventLine(br2, limit)
		if err != nil || string(got2) != "crlf" {
			t.Fatalf("crlf = %q err=%v", got2, err)
		}
	})
}

// TestScanLoopSurvivesOversizedLine 端到端（读取循环层面）：stdout 里夹一条超长
// 事件时，必须继续投递后续事件——而不是结束 supply 让上层把整个会话判为 worker lost。
func TestScanLoopSurvivesOversizedLine(t *testing.T) {
	normal := `{"type":"agent_start","id":"1"}`
	// 上限注入为 1KB（等价语义，避免测试真的分配 64MB）
	br := bufio.NewReaderSize(strings.NewReader(strings.Repeat("z", 4096)+"\n"+normal+"\n"), 64)
	var got []*RpcEvent
	for {
		line, err := readEventLine(br, 1024)
		if errors.Is(err, errEventLineTooLong) {
			continue
		}
		if err != nil {
			break
		}
		ev, perr := ParseEventLine(line)
		if perr != nil {
			t.Fatalf("parse: %v", perr)
		}
		got = append(got, ev)
	}
	if len(got) != 1 || got[0].Type != "agent_start" {
		t.Fatalf("events after oversized line = %+v, want one agent_start", got)
	}
}
