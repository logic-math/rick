package cmd

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// piListContains shells out to `pi list`; test it via a fake pi on PATH that
// prints a canned list, covering both the found and not-found cases. This
// helper backs ensureNpmExtension's idempotency check.
func TestPiListContains(t *testing.T) {
	tmp := t.TempDir()
	// fake pi: prints argv[1] routing — "list" prints canned packages.
	piScript := `#!/bin/sh
case "$1" in
  list) echo "User packages:"; echo "  pi-subagents"; echo "  pi-web-access";;
esac
`
	piPath := filepath.Join(tmp, "pi")
	if err := os.WriteFile(piPath, []byte(piScript), 0755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", tmp)

	if !piListContains("pi-subagents") {
		t.Error("expected piListContains(\"pi-subagents\") = true")
	}
	if !piListContains("pi-web-access") {
		t.Error("expected piListContains(\"pi-web-access\") = true")
	}
	if piListContains("not-installed-pkg") {
		t.Error("expected piListContains(\"not-installed-pkg\") = false")
	}
}

// verifyExtensions uses `pi list` to confirm all expected extensions are
// registered. Test it with a fake pi covering both all-present and
// missing-one cases (no real pi / LLM needed).
func TestVerifyExtensions(t *testing.T) {
	tmp := t.TempDir()

	t.Run("all_present", func(t *testing.T) {
		piScript := `#!/bin/sh
case "$1" in
  list) echo "User packages:"; echo "  pi-subagents"; echo "  pi-web-access";;
esac
`
		writeFakePi(t, tmp, piScript)
		t.Setenv("PATH", tmp)
		missing := verifyExtensions()
		if len(missing) != 0 {
			t.Errorf("expected no missing extensions, got %v", missing)
		}
	})

	t.Run("subagent_missing", func(t *testing.T) {
		piScript := `#!/bin/sh
case "$1" in
  list) echo "User packages:"; echo "  pi-web-access";;
esac
`
		writeFakePi(t, tmp, piScript)
		t.Setenv("PATH", tmp)
		missing := verifyExtensions()
		if len(missing) != 1 || missing[0] != "pi-subagents" {
			t.Errorf("expected [pi-subagents] missing, got %v", missing)
		}
	})

	t.Run("none_present", func(t *testing.T) {
		piScript := `#!/bin/sh
case "$1" in
  list) echo "No packages installed.";;
esac
`
		writeFakePi(t, tmp, piScript)
		t.Setenv("PATH", tmp)
		missing := verifyExtensions()
		if len(missing) != 2 {
			t.Errorf("expected 2 missing, got %v", missing)
		}
	})
}

func writeFakePi(t *testing.T, dir, script string) {
	t.Helper()
	piPath := filepath.Join(dir, "pi")
	if err := os.WriteFile(piPath, []byte(script), 0755); err != nil {
		t.Fatal(err)
	}
}

// setupPiSettings writes a ~/.pi/agent/settings.json with the given theme (or
// no theme if ""), pointing HOME at a temp dir, and returns that HOME.
func setupPiSettings(t *testing.T, theme string) string {
	t.Helper()
	home := t.TempDir()
	agentDir := filepath.Join(home, ".pi", "agent")
	if err := os.MkdirAll(agentDir, 0755); err != nil {
		t.Fatal(err)
	}
	s := map[string]any{"theme": theme, "packages": []string{"npm:pi-subagents"}}
	if theme == "" {
		delete(s, "theme")
	}
	data, _ := json.MarshalIndent(s, "", "  ")
	if err := os.WriteFile(filepath.Join(agentDir, "settings.json"), data, 0600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOME", home)
	return home
}

func TestCurrentTheme_ReadsField(t *testing.T) {
	setupPiSettings(t, "tokyo-night-dark")
	if got := currentTheme(); got != "tokyo-night-dark" {
		t.Errorf("currentTheme: want tokyo-night-dark, got %q", got)
	}
}

func TestCurrentTheme_EmptyWhenUnset(t *testing.T) {
	setupPiSettings(t, "")
	if got := currentTheme(); got != "" {
		t.Errorf("currentTheme: want empty, got %q", got)
	}
}

func TestSetTheme_PreservesOtherFields(t *testing.T) {
	setupPiSettings(t, "dark")
	if err := setTheme("tokyo-night-dark"); err != nil {
		t.Fatalf("setTheme: %v", err)
	}
	if got := currentTheme(); got != "tokyo-night-dark" {
		t.Errorf("after setTheme: want tokyo-night-dark, got %q", got)
	}
	// packages field must survive the rewrite.
	data, err := os.ReadFile(piSettingsPath())
	if err != nil {
		t.Fatal(err)
	}
	var s map[string]any
	if err := json.Unmarshal(data, &s); err != nil {
		t.Fatal(err)
	}
	pkgs, ok := s["packages"].([]any)
	if !ok || len(pkgs) != 1 || pkgs[0] != "npm:pi-subagents" {
		t.Errorf("packages field not preserved: %v", s["packages"])
	}
}

func TestEnsureTheme_NoopWhenAlreadyActive(t *testing.T) {
	// theme already tokyo-night-dark + package present in fake pi list.
	setupPiSettings(t, tokyoNightTheme)
	tmp := t.TempDir()
	piScript := `#!/bin/sh
case "$1" in
  list) echo "pi-tokyo-night";;
esac
`
	writeFakePi(t, tmp, piScript)
	t.Setenv("PATH", tmp)
	if err := ensureTheme(tokyoNightPkg, tokyoNightTheme); err != nil {
		t.Fatalf("ensureTheme (already active): %v", err)
	}
	if currentTheme() != tokyoNightTheme {
		t.Error("theme should remain unchanged")
	}
}

func TestEnsureTheme_AdoptsWhenDefault(t *testing.T) {
	// theme is "dark" (default) → init-pi should adopt tokyo-night-dark.
	setupPiSettings(t, "dark")
	tmp := t.TempDir()
	piScript := `#!/bin/sh
case "$1" in
  list) echo "pi-tokyo-night";;
esac
`
	writeFakePi(t, tmp, piScript)
	t.Setenv("PATH", tmp)
	if err := ensureTheme(tokyoNightPkg, tokyoNightTheme); err != nil {
		t.Fatalf("ensureTheme: %v", err)
	}
	if currentTheme() != tokyoNightTheme {
		t.Errorf("expected theme adopted to %q, got %q", tokyoNightTheme, currentTheme())
	}
}

func TestEnsureTheme_RespectsUserCustomTheme(t *testing.T) {
	// theme is a user-chosen non-default (e.g. "gruvbox") → must NOT override.
	setupPiSettings(t, "gruvbox")
	tmp := t.TempDir()
	piScript := `#!/bin/sh
case "$1" in
  list) echo "pi-tokyo-night";;
esac
`
	writeFakePi(t, tmp, piScript)
	t.Setenv("PATH", tmp)
	if err := ensureTheme(tokyoNightPkg, tokyoNightTheme); err != nil {
		t.Fatalf("ensureTheme: %v", err)
	}
	if currentTheme() != "gruvbox" {
		t.Errorf("user theme should be respected, got %q", currentTheme())
	}
}

func TestIsDefaultTheme(t *testing.T) {
	for _, c := range []string{"", "dark", "light"} {
		if !isDefaultTheme(c) {
			t.Errorf("isDefaultTheme(%q) = false, want true", c)
		}
	}
	for _, c := range []string{"tokyo-night-dark", "gruvbox", "nord"} {
		if isDefaultTheme(c) {
			t.Errorf("isDefaultTheme(%q) = true, want false", c)
		}
	}
}
