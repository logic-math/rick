package runtime

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/sunquan/rick/internal/config"
)

func TestBuildArgs_InteractiveWithFile(t *testing.T) {
	got := buildArgs(ModeInteractive, "/tmp/prompt.md")
	want := []string{"/tmp/prompt.md"}
	assertArgs(t, got, want)
}

func TestBuildArgs_PrintWithFile(t *testing.T) {
	got := buildArgs(ModePrint, "/tmp/prompt.md")
	want := []string{"-p", "/tmp/prompt.md"}
	assertArgs(t, got, want)
}

func TestBuildArgs_InteractiveWithExtraArgs(t *testing.T) {
	// easy.go session path: pi --session <id> <mainFile>
	got := buildArgs(ModeInteractive, "main.md", "--session", "sess_123")
	want := []string{"--session", "sess_123", "main.md"}
	assertArgs(t, got, want)
}

func TestBuildArgs_ResumeEmptyFile(t *testing.T) {
	// easy.go resume path: promptFile omitted, only flags. pi --session <id>
	got := buildArgs(ModeInteractive, "", "--session", "sess_123")
	want := []string{"--session", "sess_123"}
	assertArgs(t, got, want)
}

func TestBuildArgs_PrintNoFile(t *testing.T) {
	got := buildArgs(ModePrint, "", "--session", "s1")
	want := []string{"-p", "--session", "s1"}
	assertArgs(t, got, want)
}

func TestFindBinary_ConfiguredPath(t *testing.T) {
	cfg := &config.Config{PiPath: "/custom/bin/pi"}
	got, err := FindBinary(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "/custom/bin/pi" {
		t.Errorf("want /custom/bin/pi, got %q", got)
	}
}

func TestFindBinary_FallsBackToPathLookup(t *testing.T) {
	// With an empty PiPath and no managed runtime, FindBinary falls back to
	// PATH lookup. Isolate the managed runtime dir (RICK_PI_AGENT_DIR) so it is
	// empty, then point PATH at a directory that does not contain pi to prove
	// the pre-flight error fires (environment-independent).
	t.Setenv(rickAgentDirEnv, t.TempDir())
	t.Setenv("PATH", "/nonexistent-empty-path")
	cfg := &config.Config{PiPath: ""}
	_, err := FindBinary(cfg)
	if err == nil {
		t.Error("expected error when pi is neither configured, in the managed runtime, nor on PATH")
	}
}

func TestFindBinary_PrefersManagedRuntime(t *testing.T) {
	// When rick's self-contained runtime is installed, FindBinary returns it
	// even though PATH has another pi.
	agentDir := t.TempDir()
	runtimeBin := filepath.Join(agentDir, "runtime", "node_modules", ".bin", "pi")
	if err := os.MkdirAll(filepath.Dir(runtimeBin), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(runtimeBin, []byte("#!/bin/sh\n"), 0755); err != nil {
		t.Fatal(err)
	}
	t.Setenv(rickAgentDirEnv, agentDir)
	t.Setenv("PATH", "/nonexistent-empty-path")

	got, err := FindBinary(&config.Config{PiPath: ""})
	if err != nil {
		t.Fatalf("FindBinary: %v", err)
	}
	if got != runtimeBin {
		t.Errorf("FindBinary = %q, want managed runtime %q", got, runtimeBin)
	}

	// cfg.PiPath still wins over the managed runtime.
	cfg := &config.Config{PiPath: "/custom/bin/pi"}
	if got, err := FindBinary(cfg); err != nil || got != "/custom/bin/pi" {
		t.Errorf("configured path must win: got %q err %v", got, err)
	}
}

func TestPiPathOrDefault(t *testing.T) {
	if got := piPathOrDefault(&config.Config{PiPath: "/x/pi"}); got != "/x/pi" {
		t.Errorf("configured: want /x/pi, got %q", got)
	}

	// Without a managed runtime, falls back to "pi" on PATH. Isolate the
	// managed runtime dir so the test is environment-independent.
	t.Setenv(rickAgentDirEnv, t.TempDir())
	if got := piPathOrDefault(&config.Config{PiPath: ""}); got != "pi" {
		t.Errorf("empty: want pi, got %q", got)
	}
	if got := piPathOrDefault(nil); got != "pi" {
		t.Errorf("nil cfg: want pi, got %q", got)
	}
}

func TestMergeExtraArgs(t *testing.T) {
	// cfg.PiExtraArgs first (global), then per-call extraArgs.
	cfg := &config.Config{PiExtraArgs: []string{"--provider", "deepseek", "--model", "deepseek-v4-flash"}}
	got := mergeExtraArgs(cfg, []string{"--session", "s1"})
	want := []string{"--provider", "deepseek", "--model", "deepseek-v4-flash", "--session", "s1"}
	assertArgs(t, got, want)

	// nil cfg / no PiExtraArgs -> just per-call args.
	if got := mergeExtraArgs(nil, []string{"--session", "s1"}); len(got) != 2 || got[0] != "--session" {
		t.Errorf("nil cfg: want [--session s1], got %v", got)
	}
	if got := mergeExtraArgs(&config.Config{}, []string{"--session", "s1"}); len(got) != 2 {
		t.Errorf("empty PiExtraArgs: want [--session s1], got %v", got)
	}
}

func assertArgs(t *testing.T, got, want []string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("args: want %v, got %v", want, got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("args[%d]: want %q, got %q", i, want[i], got[i])
		}
	}
}
