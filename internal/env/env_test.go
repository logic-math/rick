package env

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// ---------------------------------------------------------------------------
// test helpers（测试工具，不导出为生产 API）
// ---------------------------------------------------------------------------

// writeFakePi writes an executable fake `pi` script into dir.
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

// readManagedSettings reads the managed settings.json into a map.
func readManagedSettings(t *testing.T) map[string]any {
	t.Helper()
	data, err := os.ReadFile(PiSettingsPath())
	if err != nil {
		t.Fatal(err)
	}
	var s map[string]any
	if err := json.Unmarshal(data, &s); err != nil {
		t.Fatal(err)
	}
	return s
}

// ---------------------------------------------------------------------------
// extensions（职责 2）
// ---------------------------------------------------------------------------

// TestPiListContains shells out to `pi list`; test it via a fake pi on PATH.
func TestPiListContains(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	tmp := t.TempDir()
	piScript := `#!/bin/sh
case "$1" in
  list) echo "User packages:"; echo "  pi-subagents"; echo "  pi-web-access";;
esac
`
	writeFakePi(t, tmp, piScript)
	t.Setenv("PATH", tmp)

	if !PiListContains("pi-subagents") {
		t.Error(`expected PiListContains("pi-subagents") = true`)
	}
	if !PiListContains("pi-web-access") {
		t.Error(`expected PiListContains("pi-web-access") = true`)
	}
	if PiListContains("not-installed-pkg") {
		t.Error(`expected PiListContains("not-installed-pkg") = false`)
	}
}

// TestVerifyExtensions uses `pi list` to confirm all expected extensions are
// registered (all-present / missing-one / none-present).
func TestVerifyExtensions(t *testing.T) {
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
		if missing := VerifyExtensions(); len(missing) != 0 {
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
		missing := VerifyExtensions()
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
		if missing := VerifyExtensions(); len(missing) != 2 {
			t.Errorf("expected 2 missing, got %v", missing)
		}
	})
}

// TestCheckEcosystemExtensions_Missing verifies the "不就绪即列出" contract:
// when an extension is missing, the check returns a non-empty slice naming it.
func TestCheckEcosystemExtensions_Missing(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	tmp := t.TempDir()
	piScript := `#!/bin/sh
case "$1" in
  list) echo "User packages:"; echo "  pi-subagents";;
esac
`
	writeFakePi(t, tmp, piScript)
	t.Setenv("PATH", tmp)
	missing := CheckEcosystemExtensions()
	if len(missing) == 0 || !ContainsString(missing, "pi-web-access") {
		t.Errorf("CheckEcosystemExtensions should report pi-web-access missing, got %v", missing)
	}
}

// ---------------------------------------------------------------------------
// pi install（职责 1）
// ---------------------------------------------------------------------------

func TestRequireNodeForPiInstall_BothPresent(t *testing.T) {
	tmp := t.TempDir()
	writeFakeBin(t, tmp, "node")
	writeFakeBin(t, tmp, "npm")
	t.Setenv("PATH", tmp)
	if err := RequireNodeForPiInstall(); err != nil {
		t.Errorf("expected nil when node+npm present, got: %v", err)
	}
}

func TestRequireNodeForPiInstall_NodeMissing(t *testing.T) {
	tmp := t.TempDir()
	writeFakeBin(t, tmp, "npm")
	t.Setenv("PATH", tmp)
	err := RequireNodeForPiInstall()
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
	writeFakeBin(t, tmp, "node")
	t.Setenv("PATH", tmp)
	if err := RequireNodeForPiInstall(); err == nil {
		t.Fatal("expected error when npm missing")
	}
}

func TestRequireNodeForPiInstall_BothMissing(t *testing.T) {
	t.Setenv("PATH", "/nonexistent-empty-path")
	if err := RequireNodeForPiInstall(); err == nil {
		t.Fatal("expected error when both missing")
	}
}

// ---------------------------------------------------------------------------
// settings/theme（迁移自 tools_init_pi.go）
// ---------------------------------------------------------------------------

func TestCurrentTheme_ReadsField(t *testing.T) {
	setupPiSettings(t, "tokyo-night-dark")
	if got := CurrentTheme(); got != "tokyo-night-dark" {
		t.Errorf("CurrentTheme: want tokyo-night-dark, got %q", got)
	}
}

func TestCurrentTheme_EmptyWhenUnset(t *testing.T) {
	setupPiSettings(t, "")
	if got := CurrentTheme(); got != "" {
		t.Errorf("CurrentTheme: want empty, got %q", got)
	}
}

func TestSetTheme_PreservesOtherFields(t *testing.T) {
	setupPiSettings(t, "dark")
	if err := SetTheme("tokyo-night-dark"); err != nil {
		t.Fatalf("SetTheme: %v", err)
	}
	if got := CurrentTheme(); got != "tokyo-night-dark" {
		t.Errorf("after SetTheme: want tokyo-night-dark, got %q", got)
	}
	data, err := os.ReadFile(PiSettingsPath())
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

func TestBootstrapAgentSettings_FreshNoLegacy(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	if err := BootstrapAgentSettings(); err != nil {
		t.Fatalf("BootstrapAgentSettings: %v", err)
	}
	if _, err := os.Stat(filepath.Join(home, ".rick", "pi", "agent")); err != nil {
		t.Errorf("managed agent dir not created: %v", err)
	}
	s := readManagedSettings(t)
	if s["hideThinkingBlock"] != true {
		t.Errorf("hideThinkingBlock should be true, got %v", s["hideThinkingBlock"])
	}
	if s["theme"] != "rick" {
		t.Errorf("fresh dir should default to rick theme, got %v", s["theme"])
	}
	if _, err := os.Stat(filepath.Join(home, ".rick", "pi", "agent", "themes", "rick.json")); err != nil {
		t.Errorf("embedded rick theme should be written to managed themes dir: %v", err)
	}
}

func TestBootstrapAgentSettings_MigratesLegacyThemeAndManagedPackages(t *testing.T) {
	setupLegacyPiSettings(t, "gruvbox")
	if err := BootstrapAgentSettings(); err != nil {
		t.Fatalf("BootstrapAgentSettings: %v", err)
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
	if len(pkgs) != 1 || pkgs[0] != "npm:pi-subagents" {
		t.Errorf("packages should keep only rick-managed ones, got %v", pkgs)
	}
}

func TestBootstrapAgentSettings_DoesNotMigrateTokyoNight(t *testing.T) {
	setupLegacyPiSettings(t, "tokyo-night-dark")
	if err := BootstrapAgentSettings(); err != nil {
		t.Fatalf("BootstrapAgentSettings: %v", err)
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
	setupPiSettings(t, "dark")
	if err := BootstrapAgentSettings(); err != nil {
		t.Fatalf("BootstrapAgentSettings: %v", err)
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
	if err := BootstrapAgentSettings(); err != nil {
		t.Fatalf("BootstrapAgentSettings: %v", err)
	}
	s := readManagedSettings(t)
	if len(s) != 2 || s["theme"] != "tokyo-night-dark" {
		t.Errorf("managed settings should be untouched, got %v", s)
	}
}

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

	if err := PurgeTokyoNight(); err != nil {
		t.Fatalf("PurgeTokyoNight: %v", err)
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

	if err := PurgeTokyoNight(); err != nil {
		t.Fatalf("PurgeTokyoNight: %v", err)
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

	if err := PurgeTokyoNight(); err != nil {
		t.Fatalf("PurgeTokyoNight: %v", err)
	}
	after, _ := os.ReadFile(path)
	if string(after) != string(data) {
		t.Errorf("settings should be untouched, got:\n%s", string(after))
	}
}

// ---------------------------------------------------------------------------
// 职责 3：DeployRickCustomizations
// ---------------------------------------------------------------------------

func TestDeployRickCustomizations(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	src := t.TempDir()
	if err := os.MkdirAll(filepath.Join(src, "rick-gates"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(src, "rick-gates", "helper.py"), []byte("#!/usr/bin/env python3\nprint('ok')\n"), 0755); err != nil {
		t.Fatal(err)
	}
	skillDir := filepath.Join(src, "fake_binary_script_skill")
	if err := os.MkdirAll(skillDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(skillDir, "skill.md"), []byte("# skill:fake\n"), 0644); err != nil {
		t.Fatal(err)
	}
	// 非 skill 目录（无 skill.md）不应被复制。
	readmeDir := filepath.Join(src, "README")
	if err := os.MkdirAll(readmeDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(readmeDir, "README.md"), []byte("# readme\n"), 0644); err != nil {
		t.Fatal(err)
	}

	if err := deployRickCustomizations(src); err != nil {
		t.Fatalf("deployRickCustomizations: %v", err)
	}

	// rick-gates → extensions/rick-gates/helper.py
	gates := filepath.Join(home, ".rick", "pi", "agent", "extensions", "rick-gates", "helper.py")
	if _, err := os.Stat(gates); err != nil {
		t.Errorf("rick-gates not deployed: %v", err)
	}
	// skill → skills/<name>/skill.md
	skill := filepath.Join(home, ".rick", "pi", "agent", "skills", "fake_binary_script_skill", "skill.md")
	if _, err := os.Stat(skill); err != nil {
		t.Errorf("rick skill not deployed: %v", err)
	}
	// README（非 skill 目录）不应被复制。
	if _, err := os.Stat(filepath.Join(home, ".rick", "pi", "agent", "skills", "README")); err == nil {
		t.Errorf("non-skill dir should not be deployed")
	}

	// 幂等：再跑一次不报错。
	if err := deployRickCustomizations(src); err != nil {
		t.Errorf("second deploy should be idempotent: %v", err)
	}
}

// ---------------------------------------------------------------------------
// 职责 4：就绪 check
// ---------------------------------------------------------------------------

// TestIsPIReady covers the "ready" boundary: all four check functions return
// empty and IsPIReady returns ok=true with no missing points.
func TestIsPIReady(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	// fake pi on PATH（版本 + 两个生态扩展）。
	tmp := t.TempDir()
	piScript := `#!/bin/sh
case "$1" in
  --version) echo "0.84.1" ;;
  list) echo "User packages:"; echo "  pi-subagents"; echo "  pi-web-access" ;;
  *) exit 0 ;;
esac
exit 0
`
	writeFakePi(t, tmp, piScript)
	t.Setenv("PATH", tmp)

	// deploy rick-gates so CheckRickHooks passes.
	src := t.TempDir()
	if err := os.MkdirAll(filepath.Join(src, "rick-gates"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(src, "rick-gates", "helper.py"), []byte("#!/usr/bin/env python3\n"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := deployRickCustomizations(src); err != nil {
		t.Fatalf("deployRickCustomizations: %v", err)
	}

	ok, missing := IsPIReady()
	if !ok || len(missing) != 0 {
		t.Errorf("IsPIReady = (%v, %v), want (true, [])", ok, missing)
	}
	checks := map[string]func() []string{
		"CheckPIInstalled":         CheckPIInstalled,
		"CheckEcosystemExtensions": CheckEcosystemExtensions,
		"CheckRickAgents":          CheckRickAgents,
		"CheckRickHooks":           CheckRickHooks,
	}
	for name, check := range checks {
		if m := check(); len(m) != 0 {
			t.Errorf("%s = %v, want empty (ready)", name, m)
		}
	}
}

// TestCheckPIInstalled_Missing verifies a non-ready report when pi is absent.
func TestCheckPIInstalled_Missing(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("PATH", "/nonexistent-empty-path")
	if m := CheckPIInstalled(); len(m) == 0 {
		t.Errorf("CheckPIInstalled should report missing pi, got %v", m)
	}
}

// TestCheckRickHooks_Missing verifies a non-ready report before deployment.
func TestCheckRickHooks_Missing(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	if m := CheckRickHooks(); len(m) == 0 {
		t.Errorf("CheckRickHooks should report missing rick-gates, got %v", m)
	}
}

// TestCheckRickAgents_NoRequiredAgents verifies the task3 no-op contract:
// with no required agents registered yet, the check is ready.
func TestCheckRickAgents_NoRequiredAgents(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	if m := CheckRickAgents(); len(m) != 0 {
		t.Errorf("CheckRickAgents should be ready with no required agents, got %v", m)
	}
}

// TestRuntimeEnv_Seam verifies piEnv satisfies RuntimeEnv and its methods
// dispatch to the package-level functions.
func TestRuntimeEnv_Seam(t *testing.T) {
	var _ RuntimeEnv = NewPiEnv()
	pi := NewPiEnv()

	// 隔离 cwd，使 DeployCustomizations 在无 .rick/skills 源时容错返回 nil。
	orig, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	empty := t.TempDir()
	if err := os.Chdir(empty); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(orig) }()
	t.Setenv("HOME", t.TempDir())

	if err := pi.DeployCustomizations(); err != nil {
		t.Errorf("DeployCustomizations should be tolerant, got %v", err)
	}
	// CheckReady 汇总四职责，返回未就绪点清单（可能非空，类型正确即可）。
	_ = pi.CheckReady()
}
