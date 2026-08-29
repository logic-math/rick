package cmd

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/sunquan/rick/internal/config"
	"github.com/sunquan/rick/internal/handler"
)

// webPidFile resolves the singleton pid file under an isolated HOME.
func webPidFile(t *testing.T) string {
	t.Helper()
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatal(err)
	}
	return filepath.Join(home, ".rick", "web.pid")
}

func TestWebSingletonRejectsLiveInstance(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	// Occupy the pid file with OUR pid (a live process — the test itself).
	pidFile := webPidFile(t)
	if err := os.MkdirAll(filepath.Dir(pidFile), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(pidFile, []byte(strconv.Itoa(os.Getpid())), 0644); err != nil {
		t.Fatal(err)
	}

	serveCalled := false
	err := handler.Web(handler.WebOptions{Port: 0}, func(ctx context.Context, opts handler.WebOptions, token string, cfg *config.Config) error {
		serveCalled = true
		return nil
	})
	if !errors.Is(err, handler.ErrWebAlreadyRunning) {
		t.Fatalf("want ErrWebAlreadyRunning, got %v", err)
	}
	if serveCalled {
		t.Fatal("serve must not run when singleton check fails")
	}
}

func TestWebSingletonClearsStalePidFile(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	// Stale pid file: a pid that is definitely not alive (but parseable).
	pidFile := webPidFile(t)
	if err := os.MkdirAll(filepath.Dir(pidFile), 0755); err != nil {
		t.Fatal(err)
	}
	// 2^31-1 area: extremely unlikely to be a live process; if it were,
	// kill(pid,0) permission semantics still treat it as alive and the test
	// would flake — use a pid from a finished child process instead.
	dead := spawnDeadPid(t)
	if err := os.WriteFile(pidFile, []byte(strconv.Itoa(dead)), 0644); err != nil {
		t.Fatal(err)
	}

	err := handler.Web(handler.WebOptions{Port: 0, Token: "x"}, func(ctx context.Context, opts handler.WebOptions, token string, cfg *config.Config) error {
		return nil // immediate clean exit
	})
	if err != nil {
		t.Fatalf("stale pid file should be cleared and Web proceed: %v", err)
	}
	// pid file written by this run and removed on exit (defer).
	if _, statErr := os.Stat(pidFile); !os.IsNotExist(statErr) {
		t.Errorf("pid file should be removed after clean exit, stat err=%v", statErr)
	}
}

// spawnDeadPid forks a child that exits immediately and returns its pid.
func spawnDeadPid(t *testing.T) int {
	t.Helper()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	type pidMsg struct{ pid int }
	done := make(chan pidMsg, 1)
	go func() {
		buf := make([]byte, 32)
		n, _ := r.Read(buf)
		p, _ := strconv.Atoi(strings.TrimSpace(string(buf[:n])))
		done <- pidMsg{p}
	}()
	proc, err := os.StartProcess("/bin/true", []string{"/bin/true"}, &os.ProcAttr{Files: []*os.File{nil, w, w}})
	if err != nil {
		w.Close()
		t.Skipf("cannot spawn dead pid helper: %v", err)
	}
	w.Close()
	state, err := proc.Wait()
	if err != nil || !state.Success() {
		t.Skipf("dead pid helper did not exit cleanly: %v %v", state, err)
	}
	// The pid never gets written by /bin/true; use proc.Pid directly.
	msg := <-done
	_ = msg // pipe read may race; use proc.Pid
	deadPid := proc.Pid
	if deadPid <= 0 {
		t.Skip("invalid dead pid")
	}
	return deadPid
}

func TestWebTokenAutoGenerateWrittenBack(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	var gotToken string
	err := handler.Web(handler.WebOptions{Port: 0}, func(ctx context.Context, opts handler.WebOptions, token string, cfg *config.Config) error {
		gotToken = token
		return nil
	})
	if err != nil {
		t.Fatalf("Web: %v", err)
	}

	// Token delivered to serve...
	if len(gotToken) != 32 {
		t.Fatalf("auto-generated token should be 32 hex chars, got %q", gotToken)
	}
	// ...and persisted to config.json (written back).
	cfg, err := config.LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
	if cfg.WebToken != gotToken {
		t.Fatalf("config web_token %q != served token %q", cfg.WebToken, gotToken)
	}
}

func TestWebTokenFlagOverridesConfig(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	// Seed a config token; the flag must win.
	seed, err := config.LoadConfig() // creates default config file
	if err != nil {
		t.Fatal(err)
	}
	seed.WebToken = "config-token"
	if err := config.SaveConfig(seed); err != nil {
		t.Fatal(err)
	}

	var gotToken string
	err = handler.Web(handler.WebOptions{Token: "flag-token"}, func(ctx context.Context, opts handler.WebOptions, token string, cfg *config.Config) error {
		gotToken = token
		return nil
	})
	if err != nil {
		t.Fatalf("Web: %v", err)
	}
	if gotToken != "flag-token" {
		t.Fatalf("flag token should override config: got %q", gotToken)
	}
	// Config must remain untouched (flag does not write back).
	cfg, _ := config.LoadConfig()
	if cfg.WebToken != "config-token" {
		t.Fatalf("config token must not be overwritten by flag run: %q", cfg.WebToken)
	}
}

func TestWebTokenConfigUsedWithoutFlag(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	seed, err := config.LoadConfig()
	if err != nil {
		t.Fatal(err)
	}
	seed.WebToken = "from-config"
	if err := config.SaveConfig(seed); err != nil {
		t.Fatal(err)
	}

	var gotToken string
	err = handler.Web(handler.WebOptions{}, func(ctx context.Context, opts handler.WebOptions, token string, cfg *config.Config) error {
		gotToken = token
		return nil
	})
	if err != nil {
		t.Fatalf("Web: %v", err)
	}
	if gotToken != "from-config" {
		t.Fatalf("config token should be used: got %q", gotToken)
	}
}

func TestNewWebCmdHelpSurface(t *testing.T) {
	cmd := NewWebCmd("test-version")
	if cmd == nil {
		t.Fatal("NewWebCmd returned nil")
	}
	// Subcommands present.
	subs := map[string]bool{}
	for _, sub := range cmd.Commands() {
		subs[sub.Name()] = true
	}
	for _, want := range []string{"customize", "reset"} {
		if !subs[want] {
			t.Errorf("web cmd missing subcommand %q", want)
		}
	}
	// Flags surface: --port / --listen / --token.
	for _, want := range []string{"port", "listen", "token"} {
		if cmd.Flags().Lookup(want) == nil {
			t.Errorf("web cmd missing --%s flag", want)
		}
	}
}
