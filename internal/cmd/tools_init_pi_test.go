package cmd

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
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

// writeFakeBin writes a trivial executable named `name` into dir (a fake node/npm).
func writeFakeBin(t *testing.T, dir, name string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte("#!/bin/sh\nexit 0\n"), 0755); err != nil {
		t.Fatal(err)
	}
}

func TestRequireNodeForPiInstall_BothPresent(t *testing.T) {
	tmp := t.TempDir()
	writeFakeBin(t, tmp, "node")
	writeFakeBin(t, tmp, "npm")
	t.Setenv("PATH", tmp)
	if err := requireNodeForPiInstall(); err != nil {
		t.Errorf("expected nil when node+npm present, got: %v", err)
	}
}

func TestRequireNodeForPiInstall_NodeMissing(t *testing.T) {
	tmp := t.TempDir()
	writeFakeBin(t, tmp, "npm") // npm present, node absent
	t.Setenv("PATH", tmp)
	err := requireNodeForPiInstall()
	if err == nil {
		t.Fatal("expected error when node missing")
	}
	if !strings.Contains(err.Error(), "Node.js") {
		t.Errorf("error should mention Node.js, got: %v", err)
	}
	if !strings.Contains(err.Error(), "nodejs.org") {
		t.Errorf("error should point to nodejs.org, got: %v", err)
	}
}

func TestRequireNodeForPiInstall_NpmMissing(t *testing.T) {
	tmp := t.TempDir()
	writeFakeBin(t, tmp, "node") // node present, npm absent
	t.Setenv("PATH", tmp)
	if err := requireNodeForPiInstall(); err == nil {
		t.Fatal("expected error when npm missing")
	}
}

func TestRequireNodeForPiInstall_BothMissing(t *testing.T) {
	t.Setenv("PATH", "/nonexistent-empty-path")
	if err := requireNodeForPiInstall(); err == nil {
		t.Fatal("expected error when both missing")
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

// runEnsureTheme is a helper that sets up a fake pi (package already listed)
// and calls ensureTheme with the given adopt flag + starting theme.
func runEnsureTheme(t *testing.T, startTheme string, adopt bool) string {
	t.Helper()
	setupPiSettings(t, startTheme)
	tmp := t.TempDir()
	piScript := `#!/bin/sh
case "$1" in
  list) echo "pi-tokyo-night";;
esac
`
	writeFakePi(t, tmp, piScript)
	t.Setenv("PATH", tmp)
	if err := ensureTheme(tokyoNightPkg, tokyoNightTheme, adopt); err != nil {
		t.Fatalf("ensureTheme: %v", err)
	}
	return currentTheme()
}

func TestEnsureTheme_AdoptsOnFreshInstall(t *testing.T) {
	// Fresh pi install (adopt=true): even with a user theme set, adopt tokyo.
	got := runEnsureTheme(t, "gruvbox", true)
	if got != tokyoNightTheme {
		t.Errorf("fresh install should adopt tokyo, got %q", got)
	}
}

func TestEnsureTheme_NoopOnExistingInstall(t *testing.T) {
	// pi pre-existed (adopt=false): leave the user's theme alone, even if it's
	// a default like "dark".
	got := runEnsureTheme(t, "dark", false)
	if got != "dark" {
		t.Errorf("existing install should leave dark as-is, got %q", got)
	}
}

func TestEnsureTheme_NoopOnExistingInstallWithCustomTheme(t *testing.T) {
	// pi pre-existed (adopt=false) + user has gruvbox: must NOT override.
	got := runEnsureTheme(t, "gruvbox", false)
	if got != "gruvbox" {
		t.Errorf("existing install should respect gruvbox, got %q", got)
	}
}

func TestEnsureTheme_AlreadyTokyoIsNoop(t *testing.T) {
	// Already tokyo-night-dark: no-op regardless of adopt flag.
	got := runEnsureTheme(t, tokyoNightTheme, false)
	if got != tokyoNightTheme {
		t.Errorf("already-tokyo should stay, got %q", got)
	}
}
