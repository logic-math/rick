package piagent

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sunquan/rick/internal/config"
)

// These tests exercise CallCLI end-to-end against a mock pi binary (a shell
// script that records its argv). They verify that CallCLI actually execs the
// resolved binary with the argument list produced by buildArgs — coverage that
// the buildArgs unit tests alone cannot provide. Migrated from the previous
// cmd/plan_test.go tests of the removed callClaudeCodeCLI helper.

// writeMockPi writes a mock binary at mockPath that writes its argv (one per
// line) to argsFile and exits 0.
func writeMockPi(t *testing.T, mockPath, argsFile string) {
	t.Helper()
	mockScript := fmt.Sprintf("#!/bin/sh\nprintf '%%s\\n' \"$@\" > %s\nexit 0\n", argsFile)
	if err := os.WriteFile(mockPath, []byte(mockScript), 0755); err != nil {
		t.Fatal(err)
	}
}

func TestCallCLI_MockBinaryArgv(t *testing.T) {
	tmpDir := t.TempDir()
	argsFile := filepath.Join(tmpDir, "args.txt")
	mockPath := filepath.Join(tmpDir, "mock_pi")
	writeMockPi(t, mockPath, argsFile)
	cfg := &config.Config{PiPath: mockPath}

	t.Run("promptFile_nonempty", func(t *testing.T) {
		promptFile := filepath.Join(tmpDir, "test.md")
		if err := os.WriteFile(promptFile, []byte("# prompt"), 0644); err != nil {
			t.Fatal(err)
		}
		// Interactive: pi --session <id> <promptFile> (no -p)
		if err := CallCLI(false, cfg, promptFile, ModeInteractive, "--session", "abc"); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		lines := readArgv(t, argsFile)
		want := []string{"--session", "abc", promptFile}
		assertArgv(t, lines, want)
	})

	t.Run("promptFile_empty", func(t *testing.T) {
		// Resume path: promptFile omitted, only flags. pi --session <id>
		if err := CallCLI(false, cfg, "", ModeInteractive, "--session", "xyz"); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		lines := readArgv(t, argsFile)
		want := []string{"--session", "xyz"}
		assertArgv(t, lines, want)
	})

	t.Run("print_mode_prepends_p", func(t *testing.T) {
		promptFile := filepath.Join(tmpDir, "p.md")
		if err := os.WriteFile(promptFile, []byte("# p"), 0644); err != nil {
			t.Fatal(err)
		}
		if err := CallCLI(false, cfg, promptFile, ModePrint); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		lines := readArgv(t, argsFile)
		want := []string{"-p", promptFile}
		assertArgv(t, lines, want)
	})
}

func TestCallCLI_FailingBinary(t *testing.T) {
	tmpDir := t.TempDir()
	mockPath := filepath.Join(tmpDir, "mock_pi_fail")
	if err := os.WriteFile(mockPath, []byte("#!/bin/sh\nexit 1\n"), 0755); err != nil {
		t.Fatal(err)
	}
	cfg := &config.Config{PiPath: mockPath}

	promptFile := filepath.Join(tmpDir, "prompt.md")
	if err := os.WriteFile(promptFile, []byte("# Test prompt"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := CallCLI(false, cfg, promptFile, ModeInteractive); err == nil {
		t.Error("expected error with failing mock pi binary")
	}
}

func readArgv(t *testing.T, argsFile string) []string {
	t.Helper()
	data, err := os.ReadFile(argsFile)
	if err != nil {
		t.Fatal(err)
	}
	got := strings.TrimSpace(string(data))
	if got == "" {
		return nil
	}
	return strings.Split(got, "\n")
}

func assertArgv(t *testing.T, got, want []string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("argv mismatch:\ngot:  %v\nwant: %v", got, want)
	}
	for i, w := range want {
		if got[i] != w {
			t.Errorf("argv[%d]: got %q, want %q", i, got[i], w)
		}
	}
}
