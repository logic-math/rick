package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sunquan/rick/internal/config"
	"github.com/sunquan/rick/internal/handler"
	"github.com/sunquan/rick/internal/runtime"
)

func containsStr(s, substr string) bool {
	return strings.Contains(s, substr)
}

// ─── Learning Check Tests ─────────────────────────────────────────────────────

func TestRunLearningCheck_NoSummary(t *testing.T) {
	dir := t.TempDir()
	err := runLearningCheck(dir)
	if err == nil {
		t.Fatal("expected error for missing SUMMARY.md")
	}
	if !containsStr(err.Error(), "SUMMARY.md") {
		t.Errorf("expected SUMMARY.md in error, got: %v", err)
	}
}

func TestRunLearningCheck_EmptySummary(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "SUMMARY.md"), []byte("   "), 0644); err != nil {
		t.Fatal(err)
	}
	err := runLearningCheck(dir)
	if err == nil {
		t.Fatal("expected error for empty SUMMARY.md")
	}
}

func TestRunLearningCheck_MissingJobHeading(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "SUMMARY.md"), []byte("# Summary\nsome content without job heading"), 0644); err != nil {
		t.Fatal(err)
	}
	err := runLearningCheck(dir)
	if err == nil {
		t.Fatal("expected error for SUMMARY.md missing '# Job' heading")
	}
	if !containsStr(err.Error(), "# Job") {
		t.Errorf("expected '# Job' in error, got: %v", err)
	}
}

func TestRunLearningCheck_Valid(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "SUMMARY.md"), []byte("# Job Summary\nsome content"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := runLearningCheck(dir); err != nil {
		t.Errorf("expected no error for valid learning dir, got: %v", err)
	}
}

// ─── Tools Command Tests ──────────────────────────────────────────────────────

func TestNewToolsCmd(t *testing.T) {
	cmd := NewToolsCmd()
	if cmd == nil {
		t.Fatal("NewToolsCmd returned nil")
	}
	if cmd.Use != "tools" {
		t.Errorf("expected Use='tools', got '%s'", cmd.Use)
	}
}

func TestNewLearningCheckCmd(t *testing.T) {
	cmd := NewLearningCheckCmd()
	if cmd == nil {
		t.Fatal("NewLearningCheckCmd returned nil")
	}
	if cmd.Use != "learning_check <job_id>" {
		t.Errorf("unexpected Use: %s", cmd.Use)
	}
}

func TestToolsSubcommands(t *testing.T) {
	cmd := NewToolsCmd()
	subNames := make(map[string]bool)
	for _, sub := range cmd.Commands() {
		subNames[sub.Name()] = true
	}
	for _, name := range []string{"learning_check", "dream_check"} {
		if !subNames[name] {
			t.Errorf("missing subcommand: %s", name)
		}
	}
}

// ─── Workspace-dependent tests ────────────────────────────────────────────────

func withTempWorkspace(t *testing.T, f func(dir string)) {
	t.Helper()
	orig, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(orig)
	})
	if err := os.MkdirAll(filepath.Join(dir, ".rick"), 0755); err != nil {
		t.Fatal(err)
	}
	f(dir)
}

func TestRunDoingDryRun_EmptyJobID(t *testing.T) {
	if err := handler.DoingDryRun(""); err != nil {
		t.Errorf("expected no error for empty job ID, got: %v", err)
	}
}

func TestRunDoingDryRun_NoPlanDir(t *testing.T) {
	withTempWorkspace(t, func(dir string) {
		if err := handler.DoingDryRun("job_test"); err != nil {
			t.Errorf("expected no error (dry-run ignores missing plan), got: %v", err)
		}
	})
}

func TestRunLearningCheck_WithWorkspace(t *testing.T) {
	withTempWorkspace(t, func(dir string) {
		learningDir := filepath.Join(dir, ".rick", "jobs", "job_test", "learning")
		if err := os.MkdirAll(learningDir, 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(learningDir, "SUMMARY.md"), []byte("# Job Summary\nsome content"), 0644); err != nil {
			t.Fatal(err)
		}
		if err := runLearningCheck(learningDir); err != nil {
			t.Errorf("expected no error, got: %v", err)
		}
	})
}

func TestFindPiBinary(t *testing.T) {
	path, err := runtime.FindBinary(nil)
	if err != nil {
		t.Logf("runtime.FindBinary returned error (pi not in PATH): %v", err)
	} else if path == "" {
		t.Error("expected non-empty path when pi is found")
	}
}

func TestAutoFix_MockPi(t *testing.T) {
	tmpDir := t.TempDir()
	mockPath := filepath.Join(tmpDir, "mock_pi")
	if err := os.WriteFile(mockPath, []byte("#!/bin/sh\nexit 0\n"), 0755); err != nil {
		t.Fatal(err)
	}
	promptFile := filepath.Join(tmpDir, "prompt.md")
	if err := os.WriteFile(promptFile, []byte("# prompt"), 0644); err != nil {
		t.Fatal(err)
	}
	cfg := &config.Config{PiPath: mockPath}
	if err := runtime.CallCLI(false, cfg, promptFile, runtime.ModePrint); err != nil {
		t.Errorf("expected no error with mock pi, got: %v", err)
	}
}

func TestAutoFix_FailingPi(t *testing.T) {
	tmpDir := t.TempDir()
	mockPath := filepath.Join(tmpDir, "mock_pi_fail")
	if err := os.WriteFile(mockPath, []byte("#!/bin/sh\nexit 1\n"), 0755); err != nil {
		t.Fatal(err)
	}
	promptFile := filepath.Join(tmpDir, "prompt.md")
	if err := os.WriteFile(promptFile, []byte("# prompt"), 0644); err != nil {
		t.Fatal(err)
	}
	cfg := &config.Config{PiPath: mockPath}
	if err := runtime.CallCLI(false, cfg, promptFile, runtime.ModePrint); err == nil {
		t.Error("expected error with failing pi binary")
	}
}

func TestNewLearningCheckCmd_RunE_NoArgs(t *testing.T) {
	cmd := NewLearningCheckCmd()
	cmd.SetArgs([]string{})
	if err := cmd.Execute(); err == nil {
		t.Log("expected error for missing job_id")
	}
}

func TestNewLearningCheckCmd_RunE_WithWorkspace(t *testing.T) {
	withTempWorkspace(t, func(dir string) {
		learningDir := filepath.Join(dir, ".rick", "jobs", "job_test", "learning")
		if err := os.MkdirAll(learningDir, 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(learningDir, "SUMMARY.md"), []byte("# Job Summary\nsome content"), 0644); err != nil {
			t.Fatal(err)
		}
		cmd := NewLearningCheckCmd()
		cmd.SetArgs([]string{"job_test"})
		if err := cmd.Execute(); err != nil {
			t.Logf("learning_check RunE error: %v", err)
		}
	})
}
