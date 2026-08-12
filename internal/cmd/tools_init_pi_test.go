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
	// Isolate HOME so rick's managed pi runtime (agent/runtime) resolves under
	// a temp dir and piCommand falls back to the fake pi on PATH instead of the
	// real managed runtime.
	t.Setenv("HOME", t.TempDir())
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
	// Isolate HOME so piCommand prefers the PATH fake pi, never the real
	// managed runtime (which would read the real ~/.rick/pi/agent config).
	t.Setenv("HOME", t.TempDir())
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

// setupPiSettings writes the rick-managed settings.json (~/.rick/pi/agent)
// with the given theme (or no theme if ""), pointing HOME at a temp dir, and
// returns that HOME.
func setupPiSettings(t *testing.T, theme string) string {
	t.Helper()
	home := t.TempDir()
	agentDir := filepath.Join(home, ".rick", "pi", "agent")
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

// setupLegacyPiSettings writes a legacy ~/.pi/agent/settings.json (pre-
// isolation layout) for migration tests.
func setupLegacyPiSettings(t *testing.T, theme string) string {
	t.Helper()
	home := t.TempDir()
	agentDir := filepath.Join(home, ".pi", "agent")
	if err := os.MkdirAll(agentDir, 0755); err != nil {
		t.Fatal(err)
	}
	s := map[string]any{
		"theme":    theme,
		"packages": []string{"npm:pi-subagents", "npm:user-random-thing"},
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

// --- bootstrapAgentSettings (config isolation + hideThinkingBlock) ---

// readManagedSettings reads the managed settings.json into a map.
func readManagedSettings(t *testing.T) map[string]any {
	t.Helper()
	data, err := os.ReadFile(piSettingsPath())
	if err != nil {
		t.Fatal(err)
	}
	var s map[string]any
	if err := json.Unmarshal(data, &s); err != nil {
		t.Fatal(err)
	}
	return s
}

func TestBootstrapAgentSettings_FreshNoLegacy(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	if err := bootstrapAgentSettings(); err != nil {
		t.Fatalf("bootstrapAgentSettings: %v", err)
	}
	if _, err := os.Stat(filepath.Join(home, ".rick", "pi", "agent")); err != nil {
		t.Errorf("managed agent dir not created: %v", err)
	}
	s := readManagedSettings(t)
	if s["hideThinkingBlock"] != true {
		t.Errorf("hideThinkingBlock should be true, got %v", s["hideThinkingBlock"])
	}
	// fresh managed dir seeds rick's default theme.
	if s["theme"] != "rick" {
		t.Errorf("fresh dir should default to rick theme, got %v", s["theme"])
	}
	if _, err := os.Stat(filepath.Join(home, ".rick", "pi", "agent", "themes", "rick.json")); err != nil {
		t.Errorf("embedded rick theme should be written to managed themes dir: %v", err)
	}
}

func TestBootstrapAgentSettings_MigratesLegacyThemeAndManagedPackages(t *testing.T) {
	setupLegacyPiSettings(t, "gruvbox")
	if err := bootstrapAgentSettings(); err != nil {
		t.Fatalf("bootstrapAgentSettings: %v", err)
	}
	s := readManagedSettings(t)
	if s["hideThinkingBlock"] != true {
		t.Errorf("hideThinkingBlock should be true, got %v", s["hideThinkingBlock"])
	}
	if s["theme"] != "gruvbox" {
		t.Errorf("theme should migrate from legacy, got %v", s["theme"])
	}
	pkgs, ok := s["packages"].([]any)
	if !ok {
		t.Fatalf("packages missing: %v", s)
	}
	// rick-managed packages carried over; user ad-hoc package dropped (it is
	// not installed in the isolated dir and would fail to load).
	if len(pkgs) != 1 || pkgs[0] != "npm:pi-subagents" {
		t.Errorf("packages should keep only rick-managed ones, got %v", pkgs)
	}
}

func TestBootstrapAgentSettings_DoesNotMigrateTokyoNight(t *testing.T) {
	// tokyo-night is deliberately purged from the managed config (bundled
	// status-bar extension hard-codes Tokyo Night colors and pollutes rick's
	// agent context) — its theme and package must not carry over.
	setupLegacyPiSettings(t, "tokyo-night-dark")
	if err := bootstrapAgentSettings(); err != nil {
		t.Fatalf("bootstrapAgentSettings: %v", err)
	}
	s := readManagedSettings(t)
	if s["theme"] != "rick" {
		t.Errorf("tokyo theme must not migrate; fall back to rick default, got %v", s["theme"])
	}
	for _, p := range s["packages"].([]any) {
		if strings.Contains(p.(string), "tokyo") {
			t.Errorf("tokyo-night package must not migrate, got %v", s["packages"])
		}
	}
}

func TestBootstrapAgentSettings_AddsHideThinkingBlockWhenMissing(t *testing.T) {
	setupPiSettings(t, "dark") // managed settings exists, no hideThinkingBlock
	if err := bootstrapAgentSettings(); err != nil {
		t.Fatalf("bootstrapAgentSettings: %v", err)
	}
	s := readManagedSettings(t)
	if s["hideThinkingBlock"] != true {
		t.Errorf("hideThinkingBlock should be added, got %v", s["hideThinkingBlock"])
	}
	if s["theme"] != "dark" {
		t.Errorf("existing theme must be preserved, got %v", s["theme"])
	}
}

func TestBootstrapAgentSettings_NoopWhenAlreadyManaged(t *testing.T) {
	home := t.TempDir()
	agentDir := filepath.Join(home, ".rick", "pi", "agent")
	if err := os.MkdirAll(agentDir, 0755); err != nil {
		t.Fatal(err)
	}
	managed := map[string]any{"hideThinkingBlock": true, "theme": "tokyo-night-dark"}
	data, _ := json.MarshalIndent(managed, "", "  ")
	if err := os.WriteFile(filepath.Join(agentDir, "settings.json"), data, 0600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOME", home)
	if err := bootstrapAgentSettings(); err != nil {
		t.Fatalf("bootstrapAgentSettings: %v", err)
	}
	s := readManagedSettings(t)
	if len(s) != 2 || s["theme"] != "tokyo-night-dark" {
		t.Errorf("managed settings should be untouched, got %v", s)
	}
}

// --- purgeTokyoNight (remove tokyo-night package + theme traces) ---

func TestPurgeTokyoNight_RemovesStringEntryAndRevertsTheme(t *testing.T) {
	home := t.TempDir()
	agentDir := filepath.Join(home, ".rick", "pi", "agent")
	if err := os.MkdirAll(agentDir, 0755); err != nil {
		t.Fatal(err)
	}
	managed := map[string]any{
		"hideThinkingBlock": true,
		"theme":             "tokyo-night-dark",
		"packages":          []string{"npm:pi-subagents", "npm:@wishx127/pi-tokyo-night"},
	}
	data, _ := json.MarshalIndent(managed, "", "  ")
	if err := os.WriteFile(filepath.Join(agentDir, "settings.json"), data, 0600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOME", home)

	if err := purgeTokyoNight(); err != nil {
		t.Fatalf("purgeTokyoNight: %v", err)
	}
	s := readManagedSettings(t)
	pkgs := s["packages"].([]any)
	if len(pkgs) != 1 || pkgs[0] != "npm:pi-subagents" {
		t.Errorf("tokyo-night should be removed from packages, got %v", pkgs)
	}
	if s["theme"] != "dark" {
		t.Errorf("tokyo theme should revert to dark, got %v", s["theme"])
	}
	if s["hideThinkingBlock"] != true {
		t.Errorf("hideThinkingBlock lost: %v", s)
	}
}

func TestPurgeTokyoNight_RemovesFilteredObjectForm(t *testing.T) {
	home := t.TempDir()
	agentDir := filepath.Join(home, ".rick", "pi", "agent")
	if err := os.MkdirAll(agentDir, 0755); err != nil {
		t.Fatal(err)
	}
	managed := map[string]any{
		"hideThinkingBlock": true,
		"theme":             "tokyo-night-light",
		"packages": []any{
			map[string]any{"source": "npm:@wishx127/pi-tokyo-night", "extensions": []any{}},
			"npm:pi-web-access",
		},
	}
	data, _ := json.MarshalIndent(managed, "", "  ")
	if err := os.WriteFile(filepath.Join(agentDir, "settings.json"), data, 0600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOME", home)

	if err := purgeTokyoNight(); err != nil {
		t.Fatalf("purgeTokyoNight: %v", err)
	}
	s := readManagedSettings(t)
	pkgs := s["packages"].([]any)
	if len(pkgs) != 1 || pkgs[0] != "npm:pi-web-access" {
		t.Errorf("filtered-object tokyo-night should be removed, got %v", pkgs)
	}
	if s["theme"] != "dark" {
		t.Errorf("tokyo-light should revert to dark, got %v", s["theme"])
	}
}

func TestPurgeTokyoNight_AbsentNoop(t *testing.T) {
	home := t.TempDir()
	agentDir := filepath.Join(home, ".rick", "pi", "agent")
	if err := os.MkdirAll(agentDir, 0755); err != nil {
		t.Fatal(err)
	}
	managed := map[string]any{
		"hideThinkingBlock": true,
		"theme":             "gh-dark-dimmed",
		"packages":          []string{"npm:pi-subagents"},
	}
	data, _ := json.MarshalIndent(managed, "", "  ")
	path := filepath.Join(agentDir, "settings.json")
	if err := os.WriteFile(path, data, 0600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOME", home)

	if err := purgeTokyoNight(); err != nil {
		t.Fatalf("purgeTokyoNight: %v", err)
	}
	after, _ := os.ReadFile(path)
	if string(after) != string(data) {
		t.Errorf("settings should be untouched, got:\n%s", string(after))
	}
}
