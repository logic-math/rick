// daemon_test.go exercises the daemon-mode transport (daemon.go): each pi
// worker runs behind a node shim on a unix socket, so a supervisor
// restart (DetachAll → new Supervisor → Attach) leaves pi ALIVE with its
// in-flight tool calls, and the shim replays the buffered stdout on
// re-attach. Requires node (a rick environment dependency, same as the
// real shim).
//
// Isolation: isolateRuntimeEnv redirects AgentDir (and tests set a temp
// DaemonSocketDir) so no real daemon socket is ever touched.
package runtime

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync/atomic"
	"syscall"
	"testing"
	"time"
)

// requireNode skips the test when node is unavailable (the shim is a node
// script — a rick install without node cannot run daemon mode at all).
func requireNode(t *testing.T) string {
	t.Helper()
	node, err := exec.LookPath("node")
	if err != nil {
		t.Skip("node not found in PATH — daemon tests need the shim runtime")
	}
	return node
}

// daemonTestSupervisor builds an isolated daemon-mode supervisor. The
// socket dir must be SHORT: unix sun_path caps at 104 bytes on macOS and
// t.TempDir() paths are already ~90 — os.MkdirTemp under the system tmp
// keeps it ~62, leaving room for the (length-guarded) stem.
func daemonTestSupervisor(t *testing.T, piPath string, onDead func(string, string)) *Supervisor {
	t.Helper()
	isolateRuntimeEnv(t)
	socketDir, err := os.MkdirTemp("", "rickd")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(socketDir) })
	sup := NewSupervisor(SupervisorConfig{
		HeartbeatInterval: 0, // deterministic tests: no heartbeat races
		IdleTimeout:       0,
		PiPath:            piPath,
		NodeBin:           requireNode(t),
		DaemonSocketDir:   socketDir,
		OnDead:            onDead,
	})
	t.Cleanup(func() {
		// Daemon workers must die with the test: registered ones via
		// CloseAll, detached daemons (survivors of a simulated restart)
		// via KillDetached (test hygiene — production relies on the next
		// server's CleanupOrphansExcept instead).
		sup.CloseAll()
		sup.KillDetached()
		time.Sleep(100 * time.Millisecond)
	})
	return sup
}

// daemonSpec builds a minimal daemon SpawnSpec.
func daemonSpec(sessionID string) SpawnSpec {
	return SpawnSpec{
		SessionID:     sessionID,
		SessionIDFlag: sessionID,
		CreateNew:     true,
	}
}

// fakePiDaemon is a responsive fake pi that logs every stdin line to logPath
// and, on the special "spin" command, emits N heartbeat events spaced 100ms
// apart (simulating a long-running tool call that keeps producing output
// across the detach window).
func fakePiDaemon(t *testing.T, logPath string) string {
	t.Helper()
	dir := t.TempDir()
	script := filepath.Join(dir, "fake-pi-daemon")
	var b strings.Builder
	b.WriteString("#!/bin/sh\n")
	b.WriteString("export PATH=/usr/bin:/bin:/usr/sbin:/sbin:$PATH\n")
	b.WriteString(fmt.Sprintf("printf '%%s\\n' \"$$\" > %q\n", logPath+".pid"))
	b.WriteString("while IFS= read -r line; do\n")
	b.WriteString(fmt.Sprintf("  printf '%%s\\n' \"$line\" >> %q\n", logPath))
	b.WriteString("  case \"$line\" in\n")
	b.WriteString("    *get_state*)\n")
	b.WriteString(`      echo '{"type":"response","command":"get_state","success":true,"id":"hb-1","data":{"isStreaming":false,"isCompacting":false,"sessionId":"pi-sess-1","sessionFile":"/tmp/fake.jsonl"}}'` + "\n")
	b.WriteString("      ;;\n")
	b.WriteString("    *get_entries*)\n")
	b.WriteString(`      echo '{"type":"response","command":"get_entries","success":true,"id":"ge-1","data":{"entries":[],"leafId":"leaf-42"}}'` + "\n")
	b.WriteString("      ;;\n")
	b.WriteString("    *spin*)\n")
	// The turn runs in a BACKGROUND subshell (real pi processes commands
	// concurrently — `void handleInputLine(line)` is not awaited — so
	// get_state must stay answerable WHILE the turn streams). A foreground
	// spin would block the read loop past the Attach probe window.
	b.WriteString("      (\n")
	b.WriteString("      i=0\n")
	b.WriteString("      while [ $i -lt 20 ]; do\n")
	b.WriteString("        i=$((i+1))\n")
	b.WriteString(`        echo "{\"type\":\"tool_call_update\",\"seq\":$i}"` + "\n")
	b.WriteString("        sleep 0.1\n")
	b.WriteString("      done\n")
	b.WriteString("      echo '{\"type\":\"turn_end\"}'\n")
	b.WriteString("      ) &\n")
	b.WriteString("      ;;\n")
	b.WriteString("    *)\n")
	b.WriteString(`      echo '{"type":"turn_start"}'` + "\n")
	b.WriteString(`      echo '{"type":"turn_end"}'` + "\n")
	b.WriteString("      ;;\n")
	b.WriteString("  esac\n")
	b.WriteString("done\n")
	if err := os.WriteFile(script, []byte(b.String()), 0o755); err != nil {
		t.Fatal(err)
	}
	return script
}

// groupAlive reports whether the process group answers kill(-pgid, 0).
func groupAlive(pgid int) bool {
	return syscall.Kill(-pgid, 0) == nil
}

// TestDaemonSpawnPromptRoundTrip: spawn → prompt round trip → DetachAll →
// pi STILL ALIVE (the whole point) → spin a long task → detach mid-flight →
// new supervisor Attach → buffered events replayed → new round trip works.
func TestDaemonSpawnPromptRoundTrip(t *testing.T) {
	logPath := filepath.Join(t.TempDir(), "stdin.log")
	pi := fakePiDaemon(t, logPath)
	sup := daemonTestSupervisor(t, pi, nil)

	w, err := sup.Spawn(daemonSpec("sess-roundtrip"))
	if err != nil {
		t.Fatalf("spawn: %v", err)
	}
	client := NewRpcClient()
	line, _ := client.Prompt("hello", "")
	if err := w.Send(line); err != nil {
		t.Fatalf("send: %v", err)
	}
	// First event after attach is the shim's replay marker — drain until
	// the prompt round-trip events.
	ev := drainUntil(t, w.Events(), func(ev *RpcEvent) bool { return ev.Type == "turn_end" }, 10*time.Second)
	if ev == nil {
		t.Fatal("no turn_end from round trip")
	}

	// Kick off the long task, wait for the first spin event, then detach.
	line, _ = client.Prompt("spin", "")
	if err := w.Send(line); err != nil {
		t.Fatalf("send spin: %v", err)
	}
	if ev := drainUntil(t, w.Events(), func(ev *RpcEvent) bool { return ev.Type == "tool_call_update" && ev.Command == "" }, 10*time.Second); ev == nil {
		t.Fatal("no spin event before detach")
	}

	pgid := w.pgid
	if pgid == 0 {
		t.Fatal("daemon worker has no pgid")
	}

	// Server restart: detach — the supervisor forgets the worker but the
	// process group must survive.
	sup.DetachAll()
	if got := sup.Get("sess-roundtrip"); got != nil {
		t.Fatal("DetachAll left the worker registered")
	}
	// Give the spin task time to finish emitting while detached.
	time.Sleep(700 * time.Millisecond)
	if !groupAlive(pgid) {
		t.Fatal("pi process group died on DetachAll — detach must never kill")
	}

	// New server: attach back to the live daemon.
	sup2 := daemonTestSupervisor(t, pi, nil)
	// reuse the same socket dir (the daemon's sockets live there)
	sup2.cfg.DaemonSocketDir = sup.cfg.DaemonSocketDir
	w2, err := sup2.Attach("sess-roundtrip")
	if err != nil {
		t.Fatalf("attach: %v", err)
	}
	if w2.pgid != pgid {
		t.Fatalf("attach pgid %d != spawn pgid %d (not the same daemon)", w2.pgid, pgid)
	}

	// Replay: the events pi emitted while we were detached must arrive.
	// 20 spin events total; 1 was consumed before the detach → 19 remain
	// (buffered + live-forwarded after reattach).
	spinSeqs := 0
	sawMarker := false
	deadline := time.After(10 * time.Second)
	for spinSeqs < 19 {
		select {
		case ev, ok := <-w2.Events():
			if !ok {
				t.Fatalf("events channel closed with only %d/19 spin events", spinSeqs)
			}
			if ev.Type == "shim_attached" {
				sawMarker = true
			}
			if ev.Type == "tool_call_update" && strings.Contains(string(ev.Raw), "seq") {
				spinSeqs++
			}
		case <-deadline:
			t.Fatalf("timeout: only %d/19 spin events replayed", spinSeqs)
		}
	}
	if !sawMarker {
		t.Fatal("shim_attached replay marker missing")
	}

	// Post-attach round trip: the daemon answers fresh commands.
	line, _ = client.Prompt("after-reattach", "")
	if err := w2.Send(line); err != nil {
		t.Fatalf("send after reattach: %v", err)
	}
	if ev := drainUntil(t, w2.Events(), func(ev *RpcEvent) bool { return ev.Type == "turn_end" }, 10*time.Second); ev == nil {
		t.Fatal("no turn_end after reattach round trip")
	}

	// Explicit close kills the whole group.
	w2.Close()
	if groupAlive(pgid) {
		t.Fatal("process group survived explicit Close")
	}
}

// drainUntil reads events until pred matches (nil on timeout).
func drainUntil(t *testing.T, ch <-chan *RpcEvent, pred func(*RpcEvent) bool, timeout time.Duration) *RpcEvent {
	t.Helper()
	deadline := time.After(timeout)
	for {
		select {
		case ev, ok := <-ch:
			if !ok {
				return nil
			}
			if pred(ev) {
				return ev
			}
		case <-deadline:
			return nil
		}
	}
}

// TestDaemonCloseKillsGroup: explicit Close terminates the whole process
// group (shim + pi) and the socket files are gone.
func TestDaemonCloseKillsGroup(t *testing.T) {
	pi := fakePiDaemon(t, filepath.Join(t.TempDir(), "log"))
	sup := daemonTestSupervisor(t, pi, nil)

	w, err := sup.Spawn(daemonSpec("sess-close"))
	if err != nil {
		t.Fatalf("spawn: %v", err)
	}
	pgid := w.pgid
	if !groupAlive(pgid) {
		t.Fatal("group dead before close")
	}
	w.Close()
	if groupAlive(pgid) {
		t.Fatal("group alive after Close")
	}
	socketPath := filepath.Join(sup.cfg.DaemonSocketDir, "sess-close.sock")
	if _, err := os.Stat(socketPath); !os.IsNotExist(err) {
		t.Fatalf("socket file survived Close: %v", err)
	}
	if _, err := os.Stat(socketPath + ".meta.json"); !os.IsNotExist(err) {
		t.Fatal("meta file survived Close")
	}
}

// TestDaemonSpawnOnLiveSocketFails: a live daemon for a session means Spawn
// must refuse (one session, one pi worker — never a second one).
func TestDaemonSpawnOnLiveSocketFails(t *testing.T) {
	pi := fakePiDaemon(t, filepath.Join(t.TempDir(), "log"))
	sup := daemonTestSupervisor(t, pi, nil)

	if _, err := sup.Spawn(daemonSpec("sess-dup")); err != nil {
		t.Fatalf("spawn: %v", err)
	}
	// Same supervisor (registry hit) and a fresh supervisor (socket probe
	// hit) must both refuse.
	if _, err := sup.Spawn(daemonSpec("sess-dup")); err == nil {
		t.Fatal("duplicate spawn on same supervisor accepted")
	}
	sup2 := daemonTestSupervisor(t, pi, nil)
	sup2.cfg.DaemonSocketDir = sup.cfg.DaemonSocketDir
	_, err := sup2.Spawn(daemonSpec("sess-dup"))
	if err == nil {
		t.Fatal("spawn onto live daemon socket accepted")
	}
	if !strings.Contains(err.Error(), "live daemon") {
		t.Fatalf("unexpected error for live-socket spawn: %v", err)
	}
}

// TestDaemonPiDeathClosesWorker: killing pi (SIGKILL the group) makes the
// shim exit → the worker dies with "daemon connection closed" and the
// socket files are cleaned up.
func TestDaemonPiDeathClosesWorker(t *testing.T) {
	var dead atomic.Bool
	var reason atomic.Value
	pi := fakePiDaemon(t, filepath.Join(t.TempDir(), "log"))
	sup := daemonTestSupervisor(t, pi, func(sid, last string) {
		_ = sid
		reason.Store(sid)
		dead.Store(true)
	})

	w, err := sup.Spawn(daemonSpec("sess-pi-dies"))
	if err != nil {
		t.Fatalf("spawn: %v", err)
	}
	// External kill: the whole group (shim + pi) dies at once.
	if err := syscall.Kill(-w.pgid, syscall.SIGKILL); err != nil {
		t.Fatalf("kill group: %v", err)
	}
	deadline := time.After(10 * time.Second)
	for !w.IsDead() {
		select {
		case <-deadline:
			t.Fatal("worker not dead 10s after pi group kill")
		case <-time.After(50 * time.Millisecond):
		}
	}
	if !dead.Load() {
		t.Fatal("OnDead did not fire for externally killed pi")
	}
	// SIGKILL cannot run the shim's exit handler → socket/meta files stay
	// behind until the next server's CleanupOrphansExcept — assert THAT path
	// (the production cleanup), not the impossible unlink-on-SIGKILL.
	socketPath := filepath.Join(sup.cfg.DaemonSocketDir, "sess-pi-dies.sock")
	if _, err := os.Stat(socketPath); err != nil {
		t.Fatalf("fixture sanity: socket file already gone: %v", err)
	}
	cleaned := sup.CleanupOrphansExcept(nil)
	if len(cleaned) != 1 || cleaned[0] != "sess-pi-dies" {
		t.Fatalf("CleanupOrphansExcept = %v, want [sess-pi-dies]", cleaned)
	}
	if _, err := os.Stat(socketPath); !os.IsNotExist(err) {
		t.Fatal("socket file survived CleanupOrphansExcept after pi death")
	}
	if _, err := os.Stat(socketPath + ".meta.json"); !os.IsNotExist(err) {
		t.Fatal("meta file survived CleanupOrphansExcept after pi death")
	}
}

// TestCleanupOrphans: leftover sockets with no wanted session are killed +
// removed; wanted ones (about to be attached) are left alone.
func TestCleanupOrphans(t *testing.T) {
	pi := fakePiDaemon(t, filepath.Join(t.TempDir(), "log"))
	sup := daemonTestSupervisor(t, pi, nil)

	// A live daemon nobody wants (session closed on disk).
	w, err := sup.Spawn(daemonSpec("sess-orphan"))
	if err != nil {
		t.Fatalf("spawn: %v", err)
	}
	orphanPgid := w.pgid
	// Detach so CleanupOrphansExcept sees a raw socket (not a registered worker).
	sup.DetachAll()

	// A live daemon someone wants (kept for a later attach).
	w2, err := sup.Spawn(daemonSpec("sess-wanted"))
	if err != nil {
		t.Fatalf("spawn wanted: %v", err)
	}
	wantedPgid := w2.pgid
	sup.DetachAll()

	// A dead socket pair (no process, just files).
	deadSocket := filepath.Join(sup.cfg.DaemonSocketDir, "sess-dead.sock")
	if err := os.WriteFile(deadSocket, []byte{}, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(deadSocket+".meta.json", []byte(`{"pgid":2000000000}`), 0o644); err != nil {
		t.Fatal(err)
	}

	cleaned := sup.CleanupOrphansExcept([]string{"sess-wanted"})
	if len(cleaned) != 2 {
		t.Fatalf("expected 2 cleaned (orphan + dead), got %v", cleaned)
	}

	// The orphan group is dead; the wanted one lives.
	if groupAlive(orphanPgid) {
		t.Fatal("orphan daemon survived CleanupOrphans")
	}
	if !groupAlive(wantedPgid) {
		t.Fatal("wanted daemon killed by CleanupOrphans")
	}
	if _, err := os.Stat(deadSocket); !os.IsNotExist(err) {
		t.Fatal("dead socket file survived CleanupOrphans")
	}
	// Wanted socket file still there.
	if _, err := os.Stat(filepath.Join(sup.cfg.DaemonSocketDir, "sess-wanted.sock")); err != nil {
		t.Fatalf("wanted socket file removed: %v", err)
	}
	// Kill the survivor for teardown hygiene.
	_ = syscall.Kill(-wantedPgid, syscall.SIGKILL)
}

// TestDaemonStdioDecoupledToLogFile 回归守护（dev 真 e2e 实测 2026-11-28）：
// shim/pi 的 stdout/stderr 必须路由到 <stem>.log 文件，绝不能是连回本进程的
// 管道——web 进程死亡时管道断裂，pi 任何一次 stderr 写入就 EPIPE 崩溃
// （dev 实测：detach 正常执行后 2 秒内 pi 仍死）。shim 启动必写
// "[pi-daemon-shim] listening" 到 stderr——日志文件含该行即证明
// stderr→文件路由成立（管道实现下该文件恒为空，测试立刻红）。
func TestDaemonStdioDecoupledToLogFile(t *testing.T) {
	logPath := filepath.Join(t.TempDir(), "stdin.log")
	pi := fakePiDaemon(t, logPath)
	sup := daemonTestSupervisor(t, pi, nil)

	w, err := sup.Spawn(daemonSpec("sess-stdio"))
	if err != nil {
		t.Fatalf("spawn: %v", err)
	}
	t.Cleanup(func() { w.Close() })

	// Round trip proves the daemon is serving RPC before inspecting stdio.
	client := NewRpcClient()
	line, _ := client.Prompt("hello", "")
	if err := w.Send(line); err != nil {
		t.Fatalf("send: %v", err)
	}
	if ev := drainUntil(t, w.Events(), func(ev *RpcEvent) bool { return ev.Type == "turn_end" }, 10*time.Second); ev == nil {
		t.Fatal("no turn_end from round trip")
	}

	socketPath := sup.socketPath("sess-stdio")
	data, err := os.ReadFile(socketPath + ".log")
	if err != nil {
		t.Fatalf("read daemon log: %v", err)
	}
	if !strings.Contains(string(data), "pi-daemon-shim") {
		t.Fatalf("daemon log missing shim stderr output (stdio not decoupled to file?): %q", string(data))
	}
}
