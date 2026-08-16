package env

import (
	"embed"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/sunquan/rick/internal/runtime"
)

// TokyoNightPkg is the npm spec of the Tokyo Night theme+extension package.
// rick deliberately does NOT install it: the package bundles a Powerline
// status-bar extension with hard-coded Tokyo Night RGB colors that do not
// follow the active theme and pollute rick's agent context. See PurgeTokyoNight.
const TokyoNightPkg = "@wishx127/pi-tokyo-night"

// TokyoNightThemes are the theme names provided by TokyoNightPkg. When found
// in the managed settings (theme or packages) they are reverted/removed.
var TokyoNightThemes = []string{"tokyo-night-dark", "tokyo-night-light"}

//go:embed themes/*.json
var embeddedThemes embed.FS

// PiSettingsPath returns the rick-managed settings.json
// (~/.rick/pi/agent/settings.json). Respects $HOME and RICK_PI_AGENT_DIR (tests).
func PiSettingsPath() string {
	return runtime.SettingsPath()
}

// LegacyPiSettingsPath returns the pre-isolation settings.json
// (~/.pi/agent/settings.json), read once during migration so rick's managed
// config starts from the user's existing choices (theme) instead of defaults.
func LegacyPiSettingsPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".pi", "agent", "settings.json")
}

// BootstrapAgentSettings ensures the rick-managed pi agent dir exists and its
// settings.json carries rick's managed defaults. On first run in the managed
// dir it seeds theme/packages from the legacy ~/.pi/agent/settings.json (one-
// time migration — extensions themselves are re-installed into the managed dir
// by the steps below). Every later run only merges hideThinkingBlock=true in
// when missing, and seeds rick's default theme when no theme is set. Non-fatal
// on failure (callers warn and continue).
func BootstrapAgentSettings() error {
	if err := runtime.EnsureAgentDir(); err != nil {
		return fmt.Errorf("create agent dir %s: %w", runtime.AgentDir(), err)
	}
	path := PiSettingsPath()
	if _, err := os.Stat(path); err == nil {
		if err := EnsureHideThinkingBlock(path); err != nil {
			return err
		}
		return EnsureRickTheme()
	}

	// First run in the managed dir: seed from legacy ~/.pi if present.
	base := map[string]any{"hideThinkingBlock": true}
	if data, err := os.ReadFile(LegacyPiSettingsPath()); err == nil {
		var legacy map[string]any
		if json.Unmarshal(data, &legacy) == nil {
			if t, ok := legacy["theme"].(string); ok && t != "" && !ContainsString(TokyoNightThemes, t) {
				// tokyo-night themes are deliberately not carried over (the
				// package is purged — see PurgeTokyoNight).
				base["theme"] = t
			}
			// packages are re-installed into the managed dir by the extension
			// steps below; only carry over names we know how to provide so the
			// managed settings never references packages that are not installed.
			if pkgs, ok := legacy["packages"].([]any); ok {
				var kept []any
				for _, p := range pkgs {
					s, _ := p.(string)
					if ExtensionManagedByRick(s) {
						kept = append(kept, s)
					}
				}
				if len(kept) > 0 {
					base["packages"] = kept
				}
			}
		}
	}
	out, err := json.MarshalIndent(base, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal managed settings: %w", err)
	}
	if err := os.WriteFile(path, out, 0600); err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}
	return EnsureRickTheme()
}

// EnsureRickTheme seeds rick's default theme when the managed settings has no
// theme yet (fresh machine / fresh managed dir): the embedded rick.json is
// written into the managed themes dir (pi discovers agentDir/themes/*.json)
// and "theme": "rick" is set. An existing theme is never overridden.
func EnsureRickTheme() error {
	if cur := CurrentTheme(); cur != "" {
		return nil
	}
	if err := WriteEmbeddedTheme("themes/rick.json"); err != nil {
		return fmt.Errorf("write rick theme: %w", err)
	}
	if err := SetTheme("rick"); err != nil {
		return fmt.Errorf("activate rick theme: %w", err)
	}
	return nil
}

// WriteEmbeddedTheme writes an embedded rick theme file (themes/<name>.json)
// into the managed themes dir (pi discovers agentDir/themes/*.json). The
// written file name is the base of embedPath, e.g. "themes/rick.json" →
// "rick.json".
func WriteEmbeddedTheme(embedPath string) error {
	data, err := embeddedThemes.ReadFile(embedPath)
	if err != nil {
		return fmt.Errorf("read embedded theme %s: %w", embedPath, err)
	}
	themesDir := filepath.Join(runtime.AgentDir(), "themes")
	if err := os.MkdirAll(themesDir, 0755); err != nil {
		return fmt.Errorf("create themes dir: %w", err)
	}
	if err := os.WriteFile(filepath.Join(themesDir, filepath.Base(embedPath)), data, 0644); err != nil {
		return fmt.Errorf("write theme %s: %w", filepath.Base(embedPath), err)
	}
	return nil
}

// ExtensionManagedByRick reports whether a settings.json packages entry is one
// rick itself installs/verifies (so migration can carry it over) as opposed to
// a user's ad-hoc package that would not exist in the isolated dir. tokyo-night
// is deliberately excluded: rick purges it (see PurgeTokyoNight).
func ExtensionManagedByRick(pkg string) bool {
	for _, p := range RequiredExtensions {
		if strings.Contains(pkg, p) {
			return true
		}
	}
	return false
}

// EnsureHideThinkingBlock merges "hideThinkingBlock": true into an existing
// managed settings.json, preserving all other fields.
func EnsureHideThinkingBlock(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read %s: %w", path, err)
	}
	var s map[string]any
	if err := json.Unmarshal(data, &s); err != nil {
		return fmt.Errorf("parse %s: %w", path, err)
	}
	if v, ok := s["hideThinkingBlock"].(bool); ok && v {
		return nil // already managed
	}
	s["hideThinkingBlock"] = true
	out, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal settings: %w", err)
	}
	if err := os.WriteFile(path, out, 0600); err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}
	return nil
}

// PurgeTokyoNight removes every trace of the Tokyo Night package from the
// managed settings.json: the packages entry (string or filtered-object form)
// is dropped and a tokyo-night theme is reverted to pi's built-in "dark".
// Other fields (hideThinkingBlock, remaining packages, theme) are preserved.
// Non-fatal — returns nil when the package is absent.
func PurgeTokyoNight() error {
	path := PiSettingsPath()
	data, err := os.ReadFile(path)
	if err != nil {
		return nil // no managed settings yet — nothing to purge
	}
	var s map[string]any
	if err := json.Unmarshal(data, &s); err != nil {
		return fmt.Errorf("parse %s: %w", path, err)
	}
	source := "npm:" + TokyoNightPkg
	changed := false
	if pkgs, ok := s["packages"].([]any); ok {
		kept := pkgs[:0]
		for _, p := range pkgs {
			drop := false
			switch v := p.(type) {
			case string:
				drop = v == source
			case map[string]any:
				drop = v["source"] == source
			}
			if drop {
				changed = true
				continue
			}
			kept = append(kept, p)
		}
		if len(kept) == 0 {
			delete(s, "packages")
		} else {
			s["packages"] = kept
		}
	}
	if t, ok := s["theme"].(string); ok && ContainsString(TokyoNightThemes, t) {
		s["theme"] = "dark"
		changed = true
	}
	if !changed {
		return nil
	}
	out, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal settings: %w", err)
	}
	if err := os.WriteFile(path, out, 0600); err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}
	return nil
}

// ContainsString reports whether list contains s.
func ContainsString(list []string, s string) bool {
	for _, v := range list {
		if v == s {
			return true
		}
	}
	return false
}

// CurrentTheme reads the "theme" field from the rick-managed settings.json.
// "" if unset or unreadable.
func CurrentTheme() string {
	data, err := os.ReadFile(PiSettingsPath())
	if err != nil {
		return ""
	}
	var s map[string]any
	if err := json.Unmarshal(data, &s); err != nil {
		return ""
	}
	if t, ok := s["theme"].(string); ok {
		return t
	}
	return ""
}

// SetTheme writes the "theme" field in the rick-managed settings.json, preserving
// all other fields.
func SetTheme(theme string) error {
	path := PiSettingsPath()
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read %s: %w", path, err)
	}
	var s map[string]any
	if err := json.Unmarshal(data, &s); err != nil {
		return fmt.Errorf("parse %s: %w", path, err)
	}
	s["theme"] = theme
	out, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal settings: %w", err)
	}
	if err := os.WriteFile(path, out, 0600); err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}
	return nil
}
