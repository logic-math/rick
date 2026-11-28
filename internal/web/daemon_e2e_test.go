// daemon_e2e_test.go exercises the daemon-mode web architecture end-to-end
// at the process level (human 2026-11-28): a server "restart" (ShutdownWorkers
// → fresh SessionManager+Supervisor over the SAME registries and socket dir)
// must re-attach to live pi daemons — sessions stay active, the pi processes
// (with their in-flight tool calls) never die, and events pi emitted while
// the server was down are replayed. Exception paths: explicit close kills
// the whole pi group; a crashed pi falls back to suspended; idempotency
// guarantees one pi worker per session.
package web

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/sunquan/rick/internal/handler"
	"github.com/sunquan/rick/internal/runtime"
)

// daemonWebEnv is newTestEnv's daemon-mode variant: the supervisor runs
// workers behind the node shim on a SHORT socket dir (unix sun_path caps at
// 104 bytes), and the fake pi can run a long multi-event task ("spin").
type daemonWebEnv struct {
	sessions     *SessionRegistry
	sessionsPath string
	workspaces   *WorkspaceRegistry
	hub          *Hub
	sup          *runtime.Supervisor
	socketDir    string
	wsEntry      WorkspaceEntry
	rickDir      string
	piLog        string
	piPath       string
}

// requireNodeWeb skips when node is missing (the shim is a node script).
func requireNodeWeb(t *testing.T) string {
	t.Helper()
	node, err := exec.LookPath("node")
	if err != nil {
		t.Skip("node not found in PATH — daemon e2e tests need the shim runtime")
	}
	return node
}

// fakePiWebDaemon writes a responsive fake pi that logs stdin, answers
// get_state/get_entries, and on a prompt containing "spin" emits 10
// tool_call_update events 100ms apart before turn_end (a long task that
// survives a server restart mid-flight).
func fakePiWebDaemon(t *testing.T, logPath string) string {
	t.Helper()
	dir := t.TempDir()
	script := filepath.Join(dir, "fake-pi-daemon")
	var b strings.Builder
	b.WriteString("#!/bin/sh\n")
	b.WriteString("export PATH=/usr/bin:/bin:/usr/sbin:/sbin:$PATH\n")
	b.WriteString("while IFS= read -r line; do\n")
	b.WriteString(fmt.Sprintf("  printf '%%s\\n' \"$line\" >> %q\n", logPath))
	b.WriteString("  case \"$line\" in\n")
	b.WriteString("    *get_state*)\n")
	b.WriteString(`      rid=$(printf '%s' "$line" | sed -n 's/.*"id":"\([^"]*\)".*/\1/p')` + "\n")
	b.WriteString(`      echo "{\"type\":\"response\",\"command\":\"get_state\",\"success\":true,\"id\":\"$rid\",\"data\":{\"isStreaming\":false,\"isCompacting\":false,\"sessionId\":\"pi-sess-1\",\"sessionFile\":\"/tmp/fake.jsonl\"}}"` + "\n")
	b.WriteString("      ;;\n")
	b.WriteString("    *get_entries*)\n")
	b.WriteString(`      rid=$(printf '%s' "$line" | sed -n 's/.*"id":"\([^"]*\)".*/\1/p')` + "\n")
	b.WriteString(`      echo "{\"type\":\"response\",\"command\":\"get_entries\",\"success\":true,\"id\":\"$rid\",\"data\":{\"entries\":[],\"leafId\":\"leaf-42\"}}"` + "\n")
	b.WriteString("      ;;\n")
	b.WriteString("    *spin*)\n")
	// The turn runs in a BACKGROUND subshell (real pi processes commands
	// concurrently — `void handleInputLine(line)` is not awaited — so
	// get_state must stay answerable WHILE the turn streams). A foreground
	// spin would block the read loop past the Attach probe window.
	b.WriteString("      (\n")
	b.WriteString("      i=0\n")
	b.WriteString("      while [ $i -lt 10 ]; do\n")
	b.WriteString("        i=$((i+1))\n")
	b.WriteString(`        echo "{\"type\":\"tool_call_update\",\"seq\":$i}"` + "\n")
	b.WriteString("        sleep 0.1\n")
	b.WriteString("      done\n")
	b.WriteString(`      echo '{"type":"turn_end"}'` + "\n")
	b.WriteString("      ) &\n")
	b.WriteString("      ;;\n")
	b.WriteString("    *)\n")
	b.WriteString(`      echo '{"type":"agent_start"}'` + "\n")
	b.WriteString(`      echo '{"type":"agent_settled"}'` + "\n")
	b.WriteString("      ;;\n")
	b.WriteString("  esac\n")
	b.WriteString("done\n")
	if err := os.WriteFile(script, []byte(b.String()), 0o755); err != nil {
		t.Fatal(err)
	}
	return script
}

// newDaemonWebEnv builds the isolated daemon-mode stack. The sessions
// registry lives at a FIXED path (the "restart" reloads it), the supervisor
// runs in daemon mode on a short socket dir.
func newDaemonWebEnv(t *testing.T) *daemonWebEnv {
	t.Helper()
	t.Setenv("HOME", t.TempDir())
	t.Setenv("RICK_PI_AGENT_DIR", t.TempDir())

	ws := t.TempDir()
	rickDir := filepath.Join(ws, ".rick")
	for _, d := range []string{"jobs", "loops", "skills", "domain", "draft", "dream"} {
		if err := os.MkdirAll(filepath.Join(rickDir, d), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	socketDir, err := os.MkdirTemp("", "rickwebd")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(socketDir) })

	piLog := filepath.Join(t.TempDir(), "stdin.log")
	pi := fakePiWebDaemon(t, piLog)

	stateDir := t.TempDir()
	sessionsPath := filepath.Join(stateDir, "sessions.json")
	sessions, err := LoadSessionRegistry(sessionsPath)
	if err != nil {
		t.Fatal(err)
	}
	workspaces, err := LoadWorkspaceRegistry(filepath.Join(stateDir, "web.json"))
	if err != nil {
		t.Fatal(err)
	}
	entry, created, err := workspaces.Add(ws, "daemon-ws")
	if err != nil || !created {
		t.Fatalf("register workspace: %v created=%v", err, created)
	}

	env := &daemonWebEnv{
		sessions:     sessions,
		sessionsPath: sessionsPath,
		workspaces:   workspaces,
		hub:          NewHub(128),
		socketDir:    socketDir,
		wsEntry:      entry,
		rickDir:      rickDir,
		piLog:        piLog,
		piPath:       pi,
	}
	env.sup = env.newSupervisor(t)
	return env
}

// newSupervisor builds a daemon-mode supervisor over the shared socket dir
// (each "server restart" builds its own — exactly like a new process).
func (e *daemonWebEnv) newSupervisor(t *testing.T) *runtime.Supervisor {
	t.Helper()
	sup := runtime.NewSupervisor(runtime.SupervisorConfig{
		HeartbeatInterval: 0,
		IdleTimeout:       0,
		PiPath:            e.piPath,
		NodeBin:           requireNodeWeb(t),
		DaemonSocketDir:   e.socketDir,
	})
	t.Cleanup(func() {
		sup.CloseAll()
		sup.KillDetached()
		time.Sleep(100 * time.Millisecond)
	})
	return sup
}

// manager builds a SessionManager over the given live pieces (a restart
// passes freshly-loaded registries + a fresh supervisor + a fresh hub).
func (e *daemonWebEnv) manager(t *testing.T, sessions *SessionRegistry, workspaces *WorkspaceRegistry, sup *runtime.Supervisor, hub *Hub) *SessionManager {
	t.Helper()
	dream := func(ctx context.Context, rickDir string, jobNum int, progress func(handler.DreamEvent)) error {
		<-ctx.Done()
		return ctx.Err()
	}
	return NewSessionManager(sessions, workspaces, sup, hub, nil, dream)
}

// createEasy creates an easy session via the HTTP handler.
func (e *daemonWebEnv) createEasy(t *testing.T, m *SessionManager) string {
	t.Helper()
	id, code, body := createSessionViaHTTP(t, m, e.wsEntry.ID, "easy", map[string]any{
		"requirement": "daemon e2e",
	})
	if code != http.StatusCreated {
		t.Fatalf("create easy session: %d %v", code, body)
	}
	return id
}

// socketMeta finds the single daemon meta file in the socket dir and returns
// its pgid (exactly one daemon is expected to be alive).
func (e *daemonWebEnv) socketMeta(t *testing.T) int {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for {
		metas, _ := filepath.Glob(filepath.Join(e.socketDir, "*.meta.json"))
		if len(metas) == 1 {
			data, err := os.ReadFile(metas[0])
			if err != nil {
				t.Fatal(err)
			}
			var meta struct {
				Pgid int `json:"pgid"`
			}
			if json.Unmarshal(data, &meta) != nil || meta.Pgid <= 0 {
				t.Fatalf("bad meta file %s: %s", metas[0], data)
			}
			return meta.Pgid
		}
		if time.Now().After(deadline) {
			t.Fatalf("expected exactly 1 daemon meta file, found %d in %s", len(metas), e.socketDir)
		}
		time.Sleep(50 * time.Millisecond)
	}
}

// waitStatus polls the registry until the session reaches status.
func waitStatus(t *testing.T, sessions *SessionRegistry, id, status string) SessionEntry {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for {
		if e, ok := sessions.Get(id); ok && e.Status == status {
			return e
		}
		if time.Now().After(deadline) {
			e, _ := sessions.Get(id)
			got := ""
			if e.ID != "" {
				got = e.Status
			}
			t.Fatalf("session %s status = %q, want %q", id, got, status)
		}
		time.Sleep(50 * time.Millisecond)
	}
}

// groupAliveWeb reports whether a process group answers kill(-pgid, 0).
func groupAliveWeb(pgid int) bool {
	return syscall.Kill(-pgid, 0) == nil
}

// TestDaemonE2E_RestartContinuity: create → spin (long task) → ShutdownWorkers
// (server exit) → NEW manager over reloaded registries → ReconcileOnStart
// re-attaches → session STAYS active, pi never died, spin events from the
// detached window are replayed, and fresh prompts still work.
func TestDaemonE2E_RestartContinuity(t *testing.T) {
	env := newDaemonWebEnv(t)
	m1 := env.manager(t, env.sessions, env.workspaces, env.sup, env.hub)
	coll1 := startCollector(env.hub)
	defer coll1.stop()

	id := env.createEasy(t, m1)
	waitLog(t, env.piLog, "开始：按系统提示词中的 rick 协议立即执行你的职责")

	// Kick off the long task; wait until at least one spin event reached us.
	rec := post(t, m1.SessionPrompt, "/api/sessions/"+id+"/prompt", map[string]any{"message": "spin"})
	if rec.Code != http.StatusOK && rec.Code != http.StatusAccepted {
		t.Fatalf("prompt spin: %d %s", rec.Code, rec.Body.String())
	}
	if len(coll1.waitMatching(t, 5*time.Second, func(e Envelope) bool {
		return e.Type == EventTypeSessionEvent && e.SessionID == id && strings.Contains(string(e.Data), "tool_call_update")
	})) == 0 {
		t.Fatal("no spin events before shutdown")
	}

	// Server exit: detach (pi keeps running, spin keeps producing).
	m1.ShutdownWorkers()
	if w := env.sup.Get(id); w != nil {
		t.Fatal("ShutdownWorkers left the worker registered")
	}
	pgid := env.socketMeta(t)

	// Restart: fresh supervisor + fresh hub + reloaded registries.
	sessions2, err := LoadSessionRegistry(env.sessionsPath)
	if err != nil {
		t.Fatal(err)
	}
	sup2 := env.newSupervisor(t)
	hub2 := NewHub(128)
	m2 := env.manager(t, sessions2, env.workspaces, sup2, hub2)
	coll2 := startCollector(hub2)
	defer coll2.stop()

	m2.ReconcileOnStart()

	// The session re-attached and stayed ACTIVE — never suspended.
	if w := sup2.Get(id); w == nil || w.IsDead() {
		t.Fatal("no live re-attached worker after ReconcileOnStart")
	}
	entry := waitStatus(t, sessions2, id, SessionStatusActive)
	if entry.PISessionID == "" {
		t.Fatal("PISessionID lost across restart")
	}
	if !groupAliveWeb(pgid) {
		t.Fatal("pi process group died across the restart — detach must never kill pi")
	}

	// Continuity: the events pi emitted while the server was down (the
	// remainder of the 10-event spin) were buffered by the shim and replayed
	// on attach — they must arrive on the NEW hub.
	if len(coll2.waitMatching(t, 5*time.Second, func(e Envelope) bool {
		return e.Type == EventTypeSessionEvent && e.SessionID == id && strings.Contains(string(e.Data), "tool_call_update")
	})) == 0 {
		t.Fatal("spin events emitted during the detached window were not replayed")
	}
	// The spin turn completed while we were away/re-attached.
	if len(coll2.waitMatching(t, 5*time.Second, func(e Envelope) bool {
		return e.Type == EventTypeSessionEvent && e.SessionID == id && strings.Contains(string(e.Data), "turn_end")
	})) == 0 {
		t.Fatal("no turn_end after re-attach")
	}

	// Fresh commands still work through the re-attached daemon.
	rec = post(t, m2.SessionPrompt, "/api/sessions/"+id+"/prompt", map[string]any{"message": "hello after restart"})
	if rec.Code != http.StatusOK && rec.Code != http.StatusAccepted {
		t.Fatalf("prompt after re-attach: %d %s", rec.Code, rec.Body.String())
	}
	if len(coll2.waitMatching(t, 5*time.Second, func(e Envelope) bool {
		return e.Type == EventTypeSessionEvent && e.SessionID == id && strings.Contains(string(e.Data), "agent_settled")
	})) == 0 {
		t.Fatal("no round-trip events after re-attach")
	}
	// Wait for the fake to log the post-restart prompt (delivery proof on the
	// actual stdin, not just the hub).
	waitLog(t, env.piLog, "hello after restart")
}

// TestDaemonE2E_ExplicitCloseKillsPi: user close terminates the ENTIRE pi
// process group, removes the socket files, records closed — and a later
// restart does not resurrect it.
func TestDaemonE2E_ExplicitCloseKillsPi(t *testing.T) {
	env := newDaemonWebEnv(t)
	m1 := env.manager(t, env.sessions, env.workspaces, env.sup, env.hub)
	id := env.createEasy(t, m1)
	pgid := env.socketMeta(t)
	if !groupAliveWeb(pgid) {
		t.Fatal("pi group must be alive before close")
	}

	rec := post(t, m1.SessionClose, "/api/sessions/"+id+"/close", nil)
	if rec.Code != http.StatusAccepted {
		t.Fatalf("close: %d %s", rec.Code, rec.Body.String())
	}
	if groupAliveWeb(pgid) {
		t.Fatal("pi process group survived explicit session close")
	}
	// The whole process GROUP died — the shim does not linger either.
	entry := waitStatus(t, env.sessions, id, SessionStatusClosed)
	if entry.PISessionID == "" {
		t.Fatal("close must not erase the pi session id")
	}
	deadline := time.Now().Add(3 * time.Second)
	for len(mustGlob(t, filepath.Join(env.socketDir, "*.sock"))) > 0 || len(mustGlob(t, filepath.Join(env.socketDir, "*.meta.json"))) > 0 {
		if time.Now().After(deadline) {
			t.Fatal("socket/meta files survived explicit close")
		}
		time.Sleep(50 * time.Millisecond)
	}

	// Restart: the closed session must not come back (and cleanup finds
	// nothing to attach).
	sessions2, err := LoadSessionRegistry(env.sessionsPath)
	if err != nil {
		t.Fatal(err)
	}
	sup2 := env.newSupervisor(t)
	hub2 := NewHub(128)
	m2 := env.manager(t, sessions2, env.workspaces, sup2, hub2)
	m2.ReconcileOnStart()
	if e, ok := sessions2.Get(id); !ok || e.Status != SessionStatusClosed {
		t.Fatalf("closed session resurrected: %+v", e)
	}
	if w := sup2.Get(id); w != nil {
		t.Fatal("worker spawned for a closed session")
	}
	if !groupAliveWeb(pgid) {
		return // already dead — good
	}
	t.Fatal("pi group alive after restart reconcile")
}

func mustGlob(t *testing.T, pattern string) []string {
	t.Helper()
	matches, _ := filepath.Glob(pattern)
	return matches
}

// TestDaemonE2E_NoDuplicateWorker: after a restart re-attach, further
// ensure/continue calls must REUSE the live daemon — exactly one pi process
// per session ever exists.
func TestDaemonE2E_NoDuplicateWorker(t *testing.T) {
	env := newDaemonWebEnv(t)
	m1 := env.manager(t, env.sessions, env.workspaces, env.sup, env.hub)
	id := env.createEasy(t, m1)
	pgid := env.socketMeta(t)

	// Restart + re-attach.
	m1.ShutdownWorkers()
	sessions2, err := LoadSessionRegistry(env.sessionsPath)
	if err != nil {
		t.Fatal(err)
	}
	sup2 := env.newSupervisor(t)
	hub2 := NewHub(128)
	m2 := env.manager(t, sessions2, env.workspaces, sup2, hub2)
	m2.ReconcileOnStart()
	if w := sup2.Get(id); w == nil || w.IsDead() {
		t.Fatal("re-attach failed")
	}

	// The continue endpoint on the re-attached session must be idempotent
	// (already active → no second worker).
	rec := post(t, m2.SessionContinue, "/api/sessions/"+id+"/continue", nil)
	if rec.Code != http.StatusOK && rec.Code != http.StatusAccepted {
		t.Fatalf("continue after re-attach: %d %s", rec.Code, rec.Body.String())
	}

	// And a THIRD manager (another restart) attaching while ensure runs
	// concurrently must still yield exactly one live daemon group.
	sessions3, err := LoadSessionRegistry(env.sessionsPath)
	if err != nil {
		t.Fatal(err)
	}
	sup3 := env.newSupervisor(t)
	hub3 := NewHub(128)
	m3 := env.manager(t, sessions3, env.workspaces, sup3, hub3)
	m3.ReconcileOnStart()
	if w := sup3.Get(id); w == nil || w.IsDead() {
		t.Fatal("second re-attach failed")
	}
	if !groupAliveWeb(pgid) {
		t.Fatal("original pi group died — re-attach must reuse the SAME daemon")
	}
	// Exactly one socket pair on disk.
	socks := mustGlob(t, filepath.Join(env.socketDir, "*.sock"))
	if len(socks) != 1 {
		t.Fatalf("expected exactly 1 daemon socket, found %d: %v", len(socks), socks)
	}
}

// TestDaemonE2E_CrashedPiSuspends: pi crashes while the server is DOWN → the
// restarted server's reconcile cannot attach → the session falls back to
// suspended (the documented recovery path), and the dead socket files are
// cleaned up as orphans.
func TestDaemonE2E_CrashedPiSuspends(t *testing.T) {
	env := newDaemonWebEnv(t)
	m1 := env.manager(t, env.sessions, env.workspaces, env.sup, env.hub)
	id := env.createEasy(t, m1)
	pgid := env.socketMeta(t)

	// Server exit (detach; registry keeps active), then pi CRASHES (the
	// whole group — shim included — SIGKILLed while nobody supervises).
	m1.ShutdownWorkers()
	if err := syscall.Kill(-pgid, syscall.SIGKILL); err != nil {
		t.Fatalf("kill crashed pi: %v", err)
	}

	sessions2, err := LoadSessionRegistry(env.sessionsPath)
	if err != nil {
		t.Fatal(err)
	}
	sup2 := env.newSupervisor(t)
	hub2 := NewHub(128)
	m2 := env.manager(t, sessions2, env.workspaces, sup2, hub2)
	m2.ReconcileOnStart()

	// Attach failed → suspended (NOT error, NOT active).
	waitStatus(t, sessions2, id, SessionStatusSuspended)
	if w := sup2.Get(id); w != nil {
		t.Fatal("worker registered for a crashed pi session")
	}
	// Orphan cleanup: the dead socket files are gone.
	deadline := time.Now().Add(3 * time.Second)
	for len(mustGlob(t, filepath.Join(env.socketDir, "*.sock"))) > 0 {
		if time.Now().After(deadline) {
			t.Fatal("dead daemon socket survived reconcile orphan cleanup")
		}
		time.Sleep(50 * time.Millisecond)
	}
}

// TestDaemonE2E_LegacyUnaffected: with NO DaemonSocketDir (legacy transport)
// the classic flow still works — create, prompt round trip, close.
func TestDaemonE2E_LegacyUnaffected(t *testing.T) {
	env := newTestEnv(t)
	m := env.managerWith(t, nil)
	coll := startCollector(env.hub)
	defer coll.stop()

	id, code, body := createSessionViaHTTP(t, m, env.wsEntry.ID, "easy", map[string]any{
		"requirement": "legacy still works",
	})
	if code != http.StatusCreated {
		t.Fatalf("create legacy session: %d %v", code, body)
	}
	waitLog(t, env.piLog, "开始：按系统提示词中的 rick 协议立即执行你的职责")
	if w := env.sup.Get(id); w == nil || w.IsDead() {
		t.Fatal("legacy worker missing")
	}
	if len(coll.waitMatching(t, 3*time.Second, func(e Envelope) bool {
		return e.Type == EventTypeSessionEvent && e.SessionID == id
	})) == 0 {
		t.Fatal("legacy session events missing")
	}
	rec := post(t, m.SessionPrompt, "/api/sessions/"+id+"/prompt", map[string]any{"message": "hi"})
	if rec.Code != http.StatusOK && rec.Code != http.StatusAccepted {
		t.Fatalf("legacy prompt: %d %s", rec.Code, rec.Body.String())
	}
	// Legacy shutdown still kills everything (suspendRunning + CloseAll).
	m.ShutdownWorkers()
	entry := waitStatus(t, env.sessions, id, SessionStatusSuspended)
	if entry.Busy {
		t.Fatal("legacy suspend must clear Busy")
	}
}
