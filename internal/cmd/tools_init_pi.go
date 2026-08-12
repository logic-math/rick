package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/sunquan/rick/internal/agent/piagent"
)

// piInstallerURL is the official pi installation script.
const piInstallerURL = "https://pi.dev/install.sh"

// tokyoNightPkg is the npm spec for the Tokyo Night theme package rick installs
// for a nicer pi TUI (Tokyo Night color scheme + Powerline status bar).
const tokyoNightPkg = "@wishx127/pi-tokyo-night"

// tokyoNightTheme is the theme name written to settings.json's "theme" field.
const tokyoNightTheme = "tokyo-night-dark"

// tokyoNightPkgFiltered is the managed form of the tokyo-night package entry:
// the package bundles a theme AND a Powerline status-bar extension that
// hard-codes Tokyo Night RGB colors (it does not follow the active theme).
// When rick switches to another theme (e.g. gh-dark-dimmed) the status bar
// would keep rendering Tokyo Night colors — visually "two themes at once".
// rick therefore registers the package with extensions disabled (themes stay
// available), so the status bar falls back to pi's default, which follows the
// active theme. See applyPackageFilter in pi's package-manager.js: an empty
// array explicitly disables all resources of that type.
var tokyoNightPkgFiltered = map[string]any{
	"source":     "npm:" + tokyoNightPkg,
	"extensions": []string{},
}

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
	// Step 0: prerequisite — only when pi is NOT yet installed. pi is a Node.js
	// program (>= 22.19.0) and its installer needs npm, so node/npm must be on
	// PATH before rick can install pi. rick does NOT install node — it is a
	// user-managed environment dependency (keeps rick's guidance simple and
	// respects the user's environment). When pi already exists, the user's
	// environment is assumed ready and this check is skipped.
	if _, err := exec.LookPath("pi"); err != nil {
		if err := requireNodeForPiInstall(); err != nil {
			fmt.Fprintf(os.Stderr, "❌ %v\n", err)
			return err
		}
	}

	// Step 1: pi binary present (install if missing).
	piPath, piNewlyInstalled, err := ensurePI()
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
	if piNewlyInstalled {
		fmt.Printf(" (newly installed)")
	}
	fmt.Println()

	// Step 1.5: rick-managed agent dir + settings bootstrap. rick keeps pi's
	// entire configuration under ~/.rick/pi/agent (injected via
	// PI_CODING_AGENT_DIR at every pi call site), isolated from the user's own
	// ~/.pi. The managed settings.json is bootstrapped with rick's managed
	// defaults: hideThinkingBlock=true (hide thinking blocks — they drown out
	// key information in rick easy/plan sessions), plus a one-time migration
	// of the theme from the legacy ~/.pi/agent/settings.json.
	if err := bootstrapAgentSettings(); err != nil {
		fmt.Fprintf(os.Stderr, "⚠️  managed agent dir: %v\n", err)
		fmt.Fprintf(os.Stderr, "   rick will still run, but pi config isolation may be incomplete.\n")
	} else {
		fmt.Printf("✅ pi agent dir ready: %s (hideThinkingBlock=true)\n", piagent.AgentDir())
	}

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

	// Step 4: Tokyo Night theme. Conservative policy — only adopt the theme when
	// init-pi JUST installed pi this run (a fresh install has no user preference
	// to clobber). If pi already existed, assume the user configured their own
	// theme and leave it untouched. The theme package is still installed either
	// way (so it's available via /settings). Non-fatal.
	if err := ensureTheme(tokyoNightPkg, tokyoNightTheme, piNewlyInstalled); err != nil {
		fmt.Fprintf(os.Stderr, "⚠️  theme: %v\n", err)
		fmt.Fprintf(os.Stderr, "   rick works without it; pi falls back to its default theme.\n")
	} else {
		fmt.Printf("✅ pi theme ready: %s\n", currentTheme())
	}

	// Step 4.5: keep the tokyo-night package registered with its bundled
	// status-bar extension disabled (themes only). See tokyoNightPkgFiltered.
	if err := filterTokyoNightExtension(); err != nil {
		fmt.Fprintf(os.Stderr, "⚠️  tokyo-night status bar: %v\n", err)
	} else {
		fmt.Println("✅ tokyo-night registered with status-bar extension disabled")
	}

	// Step 5: verify all required extensions + theme are actually registered.
	// Final integrity check via `pi list` — catches the case where an install
	// appeared to succeed but the extension is not registered (e.g. the old
	// `pi install <bare-source-dir>` silently wrote to settings.json without the
	// loader recognizing it). Also confirms a theme is set (any non-empty value).
	missing := verifyExtensions()
	if len(missing) > 0 {
		fmt.Fprintf(os.Stderr, "⚠️  verification: these extensions are NOT registered: %s\n", strings.Join(missing, ", "))
		fmt.Fprintf(os.Stderr, "   rick may be degraded. Re-run `rick tools init-pi` or install manually.\n")
	} else {
		fmt.Println("✅ verification: all required extensions registered")
	}
	if cur := currentTheme(); cur == "" {
		fmt.Fprintf(os.Stderr, "⚠️  verification: no theme set in settings.json\n")
	} else {
		fmt.Printf("✅ verification: theme %s active\n", cur)
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

// ensureTheme installs the theme package (if missing). The theme is ACTIVATED
// (written to settings.json) only when adoptTheme is true — which the caller
// passes only when init-pi just installed pi fresh this run. On an existing pi
// install the user's theme is left untouched (they have a preference). The
// package is always installed so the theme is available via /settings regardless.
// Non-fatal on failure.
func ensureTheme(pkg, themeName string, adoptTheme bool) error {
	// Install the theme package if not present.
	if !piListContains(filepath.Base(pkg)) {
		fmt.Printf("⚠️  theme package %s not registered — installing\n", pkg)
		cmd := piCommand("install", "npm:"+pkg)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("pi install npm:%s: %w", pkg, err)
		}
		if !piListContains(filepath.Base(pkg)) {
			return fmt.Errorf("%s still not listed after install", pkg)
		}
	}
	// Activate the theme only when the caller signals a fresh pi install.
	if !adoptTheme {
		if cur := currentTheme(); cur == themeName {
			return nil // already active (e.g. a prior init-pi set it)
		}
		fmt.Printf("✅ theme left as-is: %s (pi pre-existed; assuming user preference)\n", currentTheme())
		return nil
	}
	// Fresh install: adopt the theme if not already active.
	if cur := currentTheme(); cur == themeName {
		return nil
	}
	if err := setTheme(themeName); err != nil {
		return fmt.Errorf("activate theme %s: %w", themeName, err)
	}
	fmt.Printf("⚠️  theme activated: %s\n", themeName)
	return nil
}

// bootstrapAgentSettings ensures the rick-managed pi agent dir exists and its
// settings.json carries rick's managed defaults. On first run in the managed
// dir it seeds theme/packages from the legacy ~/.pi/agent/settings.json (one-
// time migration — extensions themselves are re-installed into the managed dir
// by the steps below). Every later run only merges hideThinkingBlock=true in
// when missing. Non-fatal on failure (callers warn and continue).
func bootstrapAgentSettings() error {
	if err := piagent.EnsureAgentDir(); err != nil {
		return fmt.Errorf("create agent dir %s: %w", piagent.AgentDir(), err)
	}
	path := piSettingsPath()
	if _, err := os.Stat(path); err == nil {
		return ensureHideThinkingBlock(path)
	}

	// First run in the managed dir: seed from legacy ~/.pi if present.
	base := map[string]any{"hideThinkingBlock": true}
	if data, err := os.ReadFile(legacyPiSettingsPath()); err == nil {
		var legacy map[string]any
		if json.Unmarshal(data, &legacy) == nil {
			if t, ok := legacy["theme"].(string); ok && t != "" {
				base["theme"] = t
			}
			// packages are re-installed into the managed dir by the extension
			// steps below; only carry over names we know how to provide so the
			// managed settings never references packages that are not installed.
			if pkgs, ok := legacy["packages"].([]any); ok {
				var kept []any
				for _, p := range pkgs {
					s, _ := p.(string)
					if extensionManagedByRick(s) {
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
	return nil
}

// extensionManagedByRick reports whether a settings.json packages entry is one
// rick itself installs/verifies (so migration can carry it over) as opposed to
// a user's ad-hoc package that would not exist in the isolated dir.
func extensionManagedByRick(pkg string) bool {
	for _, p := range requiredExtensions {
		if strings.Contains(pkg, p) {
			return true
		}
	}
	return strings.Contains(pkg, "pi-tokyo-night")
}

// ensureHideThinkingBlock merges "hideThinkingBlock": true into an existing
// managed settings.json, preserving all other fields.
func ensureHideThinkingBlock(path string) error {
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

// filterTokyoNightExtension rewrites the tokyo-night entry in the managed
// settings.json from a plain string ("npm:@wishx127/pi-tokyo-night") to the
// filtered object form {source, extensions: []}, preserving every other field.
// No-op when the package is already in filtered form or absent. Non-fatal.
func filterTokyoNightExtension() error {
	path := piSettingsPath()
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read %s: %w", path, err)
	}
	var s map[string]any
	if err := json.Unmarshal(data, &s); err != nil {
		return fmt.Errorf("parse %s: %w", path, err)
	}
	pkgs, ok := s["packages"].([]any)
	if !ok {
		return nil // no packages array yet
	}
	source := "npm:" + tokyoNightPkg
	changed := false
	for i, p := range pkgs {
		switch v := p.(type) {
		case string:
			if v == source {
				pkgs[i] = tokyoNightPkgFiltered
				changed = true
			}
		case map[string]any:
			if v["source"] == source {
				if ex, ok := v["extensions"].([]any); ok && len(ex) == 0 {
					return nil // already filtered
				}
				v["extensions"] = []string{}
				pkgs[i] = v
				changed = true
			}
		}
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

// currentTheme reads the "theme" field from the rick-managed settings.json.
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

// setTheme writes the "theme" field in the rick-managed settings.json, preserving
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

// piCommand builds a pi subprocess that runs against rick's managed agent dir
// (PI_CODING_AGENT_DIR=~/.rick/pi/agent). Every `pi install` / `pi list` /
// `pi --version` invocation must use it so installs and checks always act on
// rick's own configuration, never the user's ~/.pi.
func piCommand(args ...string) *exec.Cmd {
	cmd := exec.Command("pi", args...)
	cmd.Env = piagent.AgentEnv()
	return cmd
}

// piSettingsPath returns the rick-managed settings.json
// (~/.rick/pi/agent/settings.json). Respects $HOME and RICK_PI_AGENT_DIR (tests).
func piSettingsPath() string {
	return piagent.SettingsPath()
}

// legacyPiSettingsPath returns the pre-isolation settings.json
// (~/.pi/agent/settings.json), read once during migration so rick's managed
// config starts from the user's existing choices (theme) instead of defaults.
func legacyPiSettingsPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".pi", "agent", "settings.json")
}

// ensurePI returns the path to the pi binary, installing it via the official
// installer if it is not on PATH. The second return is true iff pi was installed
// this call (vs already present). Returns an error only if pi is still missing.
func ensurePI() (string, bool, error) {
	if p, err := exec.LookPath("pi"); err == nil {
		return p, false, nil
	}
	fmt.Println("⚠️  pi not found on PATH — installing via official installer...")
	if err := installPI(); err != nil {
		return "", false, fmt.Errorf("install pi: %w", err)
	}
	p, err := exec.LookPath("pi")
	if err != nil {
		return "", false, fmt.Errorf("pi still not on PATH after install")
	}
	return p, true, nil
}

// requireNodeForPiInstall checks that node and npm are on PATH before rick
// installs pi. pi is a Node.js program (>= 22.19.0) and its installer shells
// out to npm. rick treats node as a user-managed environment dependency — it
// does NOT install node (keeps rick's guidance simple; respects the user's
// environment). Returns a fatal error with install instructions if missing.
func requireNodeForPiInstall() error {
	_, nodeErr := exec.LookPath("node")
	_, npmErr := exec.LookPath("npm")
	if nodeErr == nil && npmErr == nil {
		return nil
	}
	return fmt.Errorf(`pi requires Node.js (>= 22.19.0) and npm to install, but they are not on PATH.

   rick does not install Node.js — it is an environment dependency you manage.
   Install Node.js LTS from https://nodejs.org/ (this includes npm), then re-run:

     rick tools init-pi`)
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
	cmd := piCommand("--version")
	if piPath != "" {
		cmd.Path = piPath
	}
	out, err := cmd.Output()
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
	cmd := piCommand("install", "npm:"+pkg)
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
	out, err := piCommand("list").Output()
	if err != nil {
		return false
	}
	return strings.Contains(string(out), substr)
}
