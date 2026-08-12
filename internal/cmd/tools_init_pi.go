package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

// piInstallerURL is the official pi installation script.
const piInstallerURL = "https://pi.dev/install.sh"

// tokyoNightPkg is the npm spec for the Tokyo Night theme package rick installs
// for a nicer pi TUI (Tokyo Night color scheme + Powerline status bar).
const tokyoNightPkg = "@wishx127/pi-tokyo-night"

// tokyoNightTheme is the theme name written to settings.json's "theme" field.
const tokyoNightTheme = "tokyo-night-dark"

// NewInitPiCmd creates the init-pi subcommand: ensures pi (rick's agent
// runtime) is installed, the subagent + web-access extensions are registered,
// and the Tokyo Night theme is activated. Idempotent — each step checks before
// acting and skips what is already done.
func NewInitPiCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "init-pi",
		Short: "Initialize pi (rick's agent runtime) + extensions + theme",
		Long: `Ensure pi is installed, required extensions are registered, and the
Tokyo Night theme is activated.

rick drives pi (@earendil-works/pi-coding-agent) as its agent runtime. This
command guarantees the runtime is ready: it installs pi if missing (via the
official installer), registers pi-subagents (Sub Agent delegation), registers
pi-web-access (external web search/fetch), and activates the Tokyo Night theme
(nicer TUI). A final verification step confirms everything is registered.

Idempotent: every step checks first and skips what is already satisfied.
Non-fatal: a missing extension/theme does not block rick; pi being entirely
missing is the only fatal condition.

Exit codes:
  0  pi environment ready (or ready enough to run rick)
  1  pi could not be installed/found (rick cannot run)`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := runInitPi(); err != nil {
				fmt.Fprintf(os.Stderr, "❌ init-pi failed: %v\n", err)
				os.Exit(1)
			}
			return nil
		},
	}
}

// runInitPi runs the initialization and returns an error only when pi is
// unavailable and rick therefore cannot run. Missing extensions/theme are
// warned about, not fatal.
func runInitPi() error {
	// Step 1: pi binary present (install if missing).
	piPath, err := ensurePI()
	if err != nil {
		// Fatal: without pi, rick cannot execute any agent command.
		fmt.Fprintf(os.Stderr, "❌ pi is not available and could not be installed: %v\n", err)
		fmt.Fprintf(os.Stderr, "   Install pi manually: curl -fsSL %s | sh\n", piInstallerURL)
		return err
	}
	ver := piVersion(piPath)
	fmt.Printf("✅ pi found: %s", piPath)
	if ver != "" {
		fmt.Printf(" (v%s)", ver)
	}
	fmt.Println()

	// Step 2: subagent extension registered (install if missing). Non-fatal.
	// pi-subagents is the official npm extension providing the `subagent` tool
	// (single/parallel/chain delegation with isolated context). The bare .ts
	// example in the pi package has no package.json and `pi install <path>`
	// silently fails to register it — only the npm package works.
	if err := ensureNpmExtension("pi-subagents", "pi-subagents"); err != nil {
		fmt.Fprintf(os.Stderr, "⚠️  subagent extension: %v\n", err)
		fmt.Fprintf(os.Stderr, "   rick works without it, but Sub Agent delegation is unavailable.\n")
	} else {
		fmt.Println("✅ pi subagent extension ready")
	}

	// Step 3: web-access extension registered (install if missing). Non-fatal.
	if err := ensureNpmExtension("pi-web-access", "web-access"); err != nil {
		fmt.Fprintf(os.Stderr, "⚠️  web-access extension: %v\n", err)
		fmt.Fprintf(os.Stderr, "   rick works without it, but external web search/fetch is unavailable.\n")
	} else {
		fmt.Println("✅ pi web-access extension ready")
	}

	// Step 4: Tokyo Night theme installed + activated. Non-fatal (cosmetic).
	if err := ensureTheme(tokyoNightPkg, tokyoNightTheme); err != nil {
		fmt.Fprintf(os.Stderr, "⚠️  theme: %v\n", err)
		fmt.Fprintf(os.Stderr, "   rick works without it; pi falls back to its default theme.\n")
	} else {
		fmt.Printf("✅ pi theme ready: %s\n", tokyoNightTheme)
	}

	// Step 5: verify all required extensions + theme are actually registered.
	// Final integrity check via `pi list` — catches the case where an install
	// appeared to succeed but the extension is not registered (e.g. the old
	// `pi install <bare-source-dir>` silently wrote to settings.json without the
	// loader recognizing it). Also confirms the theme field is set.
	missing := verifyExtensions()
	if len(missing) > 0 {
		fmt.Fprintf(os.Stderr, "⚠️  verification: these extensions are NOT registered: %s\n", strings.Join(missing, ", "))
		fmt.Fprintf(os.Stderr, "   rick may be degraded. Re-run `rick tools init-pi` or install manually.\n")
	} else {
		fmt.Println("✅ verification: all required extensions registered")
	}
	if cur := currentTheme(); cur != tokyoNightTheme {
		fmt.Fprintf(os.Stderr, "⚠️  verification: theme is %q (expected %q)\n", cur, tokyoNightTheme)
	} else {
		fmt.Printf("✅ verification: theme %s active\n", tokyoNightTheme)
	}

	fmt.Println("✅ pi environment ready")
	return nil
}

// requiredExtensions is the full set of pi extensions rick depends on.
var requiredExtensions = []string{"pi-subagents", "pi-web-access"}

// verifyExtensions runs `pi list` and returns the names of required extensions
// that are NOT registered. Empty result means all present.
func verifyExtensions() []string {
	var missing []string
	for _, pkg := range requiredExtensions {
		if !piListContains(pkg) {
			missing = append(missing, pkg)
		}
	}
	return missing
}

// ensureTheme installs the theme package (if missing) and activates the theme
// in settings.json (if not already the active theme). Non-fatal on failure.
func ensureTheme(pkg, themeName string) error {
	// Install the theme package if not present.
	if !piListContains(filepath.Base(pkg)) {
		fmt.Printf("⚠️  theme package %s not registered — installing\n", pkg)
		cmd := exec.Command("pi", "install", "npm:"+pkg)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("pi install npm:%s: %w", pkg, err)
		}
		if !piListContains(filepath.Base(pkg)) {
			return fmt.Errorf("%s still not listed after install", pkg)
		}
	}
	// Activate the theme in settings.json if not already active.
	if cur := currentTheme(); cur == themeName {
		return nil // already active
	}
	if err := setTheme(themeName); err != nil {
		return fmt.Errorf("activate theme %s: %w", themeName, err)
	}
	fmt.Printf("⚠️  theme activated: %s\n", themeName)
	return nil
}

// currentTheme reads the "theme" field from ~/.pi/agent/settings.json. Returns
// "" if unset or unreadable.
func currentTheme() string {
	data, err := os.ReadFile(piSettingsPath())
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

// setTheme writes the "theme" field in ~/.pi/agent/settings.json, preserving
// all other fields.
func setTheme(theme string) error {
	path := piSettingsPath()
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

// piSettingsPath returns ~/.pi/agent/settings.json. Respects $HOME (tests).
func piSettingsPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".pi", "agent", "settings.json")
}

// ensurePI returns the path to the pi binary, installing it via the official
// installer if it is not on PATH. Returns an error only if pi is still missing.
func ensurePI() (string, error) {
	if p, err := exec.LookPath("pi"); err == nil {
		return p, nil
	}
	fmt.Println("⚠️  pi not found on PATH — installing via official installer...")
	if err := installPI(); err != nil {
		return "", fmt.Errorf("install pi: %w", err)
	}
	p, err := exec.LookPath("pi")
	if err != nil {
		return "", fmt.Errorf("pi still not on PATH after install")
	}
	return p, nil
}

// installPI runs the official pi installer (curl | sh).
func installPI() error {
	cmdStr := fmt.Sprintf("curl -fsSL %s | sh", piInstallerURL)
	cmd := exec.Command("sh", "-c", cmdStr)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("installer failed: %w", err)
	}
	return nil
}

// piVersion returns pi's version string (e.g. "0.84.1"), or "" if it cannot be
// determined. Best-effort — init-pi does not gate on version.
func piVersion(piPath string) string {
	out, err := exec.Command(piPath, "--version").Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

// ensureNpmExtension registers an npm-based pi extension if it is not already
// installed. pkg is the npm spec passed to `pi install` (e.g. "pi-web-access");
// detect is the substring `pi list` is grepped for (usually the package name
// without the npm: prefix). Non-fatal on failure — callers warn and continue.
func ensureNpmExtension(pkg, detect string) error {
	if piListContains(detect) {
		return nil // already registered
	}
	fmt.Printf("⚠️  %s not registered — installing via pi install npm:%s\n", pkg, pkg)
	cmd := exec.Command("pi", "install", "npm:"+pkg)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("pi install npm:%s: %w", pkg, err)
	}
	if !piListContains(detect) {
		return fmt.Errorf("%s still not listed after install", pkg)
	}
	return nil
}

// piListContains reports whether `pi list` output contains substr. Used to
// detect installed extensions idempotently (ensureNpmExtension + verifyExtensions).
func piListContains(substr string) bool {
	out, err := exec.Command("pi", "list").Output()
	if err != nil {
		return false
	}
	return strings.Contains(string(out), substr)
}
