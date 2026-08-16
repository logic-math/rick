package cmd

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sunquan/rick/internal/env"
)

// fakePiWithList installs a fake pi on PATH whose `list` reads a list file and
// whose `install` appends "npm:<pkg>" to it — enough state to exercise the
// auto-install path of rick tools theme without a real pi/npm/network.
func fakePiWithList(t *testing.T, listFile string) {
	t.Helper()
	dir := t.TempDir()
	// cat is not a shell builtin: restore the system PATH inside the script
	// because tests replace PATH with the fake-pi dir only (echo works either
	// way, which is why plain-echo fakes never hit this).
	script := `#!/bin/sh
export PATH=/usr/bin:/bin:/usr/sbin:/sbin:$PATH
case "$1" in
  list) cat "$FAKE_PI_LIST" 2>/dev/null;;
  install) echo "$2" >> "$FAKE_PI_LIST"; echo "Installed $2";;
esac
`
	if err := os.WriteFile(filepath.Join(dir, "pi"), []byte(script), 0755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir)
	t.Setenv("FAKE_PI_LIST", listFile)
}

// setupPiSettings writes the rick-managed settings.json (~/.rick/pi/agent)
// with the given theme (or no theme if ""), pointing HOME at a temp dir.
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

// readManagedSettings reads the managed settings.json into a map.
func readManagedSettings(t *testing.T) map[string]any {
	t.Helper()
	data, err := os.ReadFile(env.PiSettingsPath())
	if err != nil {
		t.Fatal(err)
	}
	var s map[string]any
	if err := json.Unmarshal(data, &s); err != nil {
		t.Fatal(err)
	}
	return s
}

func TestRunThemeList_ShowsCurrentAndOptions(t *testing.T) {
	setupPiSettings(t, "dark") // managed settings with theme=dark
	listFile := filepath.Join(t.TempDir(), "list.txt")
	if err := os.WriteFile(listFile, []byte(""), 0644); err != nil {
		t.Fatal(err)
	}
	fakePiWithList(t, listFile)

	var buf bytes.Buffer
	if err := runThemeList(&buf); err != nil {
		t.Fatalf("runThemeList: %v", err)
	}
	out := buf.String()
	for _, want := range []string{"Current theme: dark", "dark", "light", "gh-dark", "nightowl", "rick tools theme <name>"} {
		if !strings.Contains(out, want) {
			t.Errorf("list output missing %q:\n%s", want, out)
		}
	}
	if !strings.Contains(out, "not installed") {
		t.Errorf("uninstalled packages should be marked, output:\n%s", out)
	}
}

func TestRunThemeSet_AutoInstallsProvidingPackage(t *testing.T) {
	setupPiSettings(t, "dark")
	listFile := filepath.Join(t.TempDir(), "list.txt")
	if err := os.WriteFile(listFile, []byte(""), 0644); err != nil {
		t.Fatal(err)
	}
	fakePiWithList(t, listFile)

	var buf bytes.Buffer
	if err := runThemeSet("gh-dark", &buf); err != nil {
		t.Fatalf("runThemeSet: %v", err)
	}
	if got := env.CurrentTheme(); got != "gh-dark" {
		t.Errorf("theme should be gh-dark, got %q", got)
	}
	// hideThinkingBlock must survive the theme switch (bootstrap + setTheme).
	s := readManagedSettings(t)
	if s["hideThinkingBlock"] != true {
		t.Errorf("hideThinkingBlock lost during theme switch: %v", s)
	}
	// The providing package was installed into the (fake) managed registry.
	data, err := os.ReadFile(listFile)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "npm:pi-gh-dark-theme") {
		t.Errorf("theme package should have been installed, list file: %q", string(data))
	}
	if !strings.Contains(buf.String(), "theme active: gh-dark") {
		t.Errorf("success message missing: %q", buf.String())
	}
}

func TestRunThemeSet_BuiltinNoInstall(t *testing.T) {
	setupPiSettings(t, "dark")
	listFile := filepath.Join(t.TempDir(), "list.txt")
	if err := os.WriteFile(listFile, []byte(""), 0644); err != nil {
		t.Fatal(err)
	}
	fakePiWithList(t, listFile)

	var buf bytes.Buffer
	if err := runThemeSet("light", &buf); err != nil {
		t.Fatalf("runThemeSet(light): %v", err)
	}
	if got := env.CurrentTheme(); got != "light" {
		t.Errorf("theme should be light, got %q", got)
	}
	// built-in themes need no package install.
	data, _ := os.ReadFile(listFile)
	if len(data) != 0 {
		t.Errorf("built-in theme must not trigger an install, got %q", string(data))
	}
}

func TestRunThemeSet_UnknownTheme(t *testing.T) {
	setupPiSettings(t, "dark")
	listFile := filepath.Join(t.TempDir(), "list.txt")
	if err := os.WriteFile(listFile, []byte(""), 0644); err != nil {
		t.Fatal(err)
	}
	fakePiWithList(t, listFile)

	err := runThemeSet("no-such-theme", &bytes.Buffer{})
	if err == nil {
		t.Fatal("expected error for unknown theme")
	}
	if !strings.Contains(err.Error(), "no-such-theme") || !strings.Contains(err.Error(), "available") {
		t.Errorf("error should name the theme and available options, got: %v", err)
	}
}

func TestRunThemeSet_CustomThemeFromManagedDir(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	themesDir := filepath.Join(home, ".rick", "pi", "agent", "themes")
	if err := os.MkdirAll(themesDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(themesDir, "gruvbox.json"), []byte(`{"name":"gruvbox","vars":{}}`), 0644); err != nil {
		t.Fatal(err)
	}
	listFile := filepath.Join(t.TempDir(), "list.txt")
	if err := os.WriteFile(listFile, []byte(""), 0644); err != nil {
		t.Fatal(err)
	}
	fakePiWithList(t, listFile)

	var buf bytes.Buffer
	if err := runThemeSet("gruvbox", &buf); err != nil {
		t.Fatalf("runThemeSet(gruvbox): %v", err)
	}
	if got := env.CurrentTheme(); got != "gruvbox" {
		t.Errorf("theme should be gruvbox, got %q", got)
	}
	// no package install for custom themes.
	data, _ := os.ReadFile(listFile)
	if len(data) != 0 {
		t.Errorf("custom theme must not trigger install, got %q", string(data))
	}
}

func TestCustomThemeNames_Sorted(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	themeDir := filepath.Join(home, ".rick", "pi", "agent", "themes")
	if err := os.MkdirAll(themeDir, 0755); err != nil {
		t.Fatal(err)
	}
	for _, f := range []string{"zebra.json", "alpha.json", "note.txt"} {
		if err := os.WriteFile(filepath.Join(themeDir, f), []byte("{}"), 0644); err != nil {
			t.Fatal(err)
		}
	}
	names := customThemeNames()
	if len(names) != 2 || names[0] != "alpha" || names[1] != "zebra" {
		t.Errorf("customThemeNames: want [alpha zebra], got %v", names)
	}
}

func TestRunThemeSet_EmbeddedRickTheme(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	listFile := filepath.Join(t.TempDir(), "list.txt")
	if err := os.WriteFile(listFile, []byte(""), 0644); err != nil {
		t.Fatal(err)
	}
	fakePiWithList(t, listFile)

	var buf bytes.Buffer
	if err := runThemeSet("gh-light", &buf); err != nil {
		t.Fatalf("runThemeSet(gh-light): %v", err)
	}
	if got := env.CurrentTheme(); got != "gh-light" {
		t.Errorf("theme should be gh-light, got %q", got)
	}
	// The embedded theme JSON must be written into the managed themes dir
	// (pi discovers agentDir/themes/*.json automatically).
	themeFile := filepath.Join(home, ".rick", "pi", "agent", "themes", "gh-light.json")
	data, err := os.ReadFile(themeFile)
	if err != nil {
		t.Fatalf("embedded theme not written: %v", err)
	}
	if !strings.Contains(string(data), `"name": "gh-light"`) {
		t.Errorf("embedded theme content wrong: %s", string(data)[:100])
	}
	// hideThinkingBlock managed by bootstrap must survive.
	s := readManagedSettings(t)
	if s["hideThinkingBlock"] != true {
		t.Errorf("hideThinkingBlock lost: %v", s)
	}
	// no npm install for embedded themes.
	list, _ := os.ReadFile(listFile)
	if len(list) != 0 {
		t.Errorf("embedded theme must not trigger npm install, got %q", string(list))
	}
}

func TestRunThemeList_MarksEmbeddedThemes(t *testing.T) {
	setupPiSettings(t, "gh-light")
	listFile := filepath.Join(t.TempDir(), "list.txt")
	if err := os.WriteFile(listFile, []byte(""), 0644); err != nil {
		t.Fatal(err)
	}
	fakePiWithList(t, listFile)

	var buf bytes.Buffer
	if err := runThemeList(&buf); err != nil {
		t.Fatalf("runThemeList: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "gh-light") || !strings.Contains(out, "builtin (rick)") {
		t.Errorf("embedded themes should be listed as builtin (rick):\n%s", out)
	}
	if !strings.Contains(out, "gh-dark") || !strings.Contains(out, "not installed") {
		t.Errorf("npm-provided gh-dark should be listed as not installed:\n%s", out)
	}
}
