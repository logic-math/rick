package piagent

import (
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
	// With an empty PiPath, FindBinary falls back to PATH lookup. Point PATH at a
	// directory that does not contain pi to prove the pre-flight error fires
	// (environment-independent: works whether or not pi is actually installed).
	t.Setenv("PATH", "/nonexistent-empty-path")
	cfg := &config.Config{PiPath: ""}
	_, err := FindBinary(cfg)
	if err == nil {
		t.Error("expected error when pi is neither configured nor on PATH")
	}
}

func TestPiPathOrDefault(t *testing.T) {
	if got := piPathOrDefault(&config.Config{PiPath: "/x/pi"}); got != "/x/pi" {
		t.Errorf("configured: want /x/pi, got %q", got)
	}
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
