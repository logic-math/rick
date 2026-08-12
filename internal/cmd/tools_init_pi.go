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

// tokyoNightPkg is the npm spec of the Tokyo Night theme+extension package.
// rick deliberately does NOT install it: the package bundles a Powerline
// status-bar extension with hard-coded Tokyo Night RGB colors that do not
// follow the active theme and pollute rick's agent context. See
// purgeTokyoNight.
const tokyoNightPkg = "@wishx127/pi-tokyo-night"

// tokyoNightThemes are the theme names provided by tokyoNightPkg. When found
// in the managed settings (theme or packages) they are reverted/removed.
var tokyoNightThemes = []string{"tokyo-night-dark", "tokyo-night-light"}

// NewInitPiCmd creates the init-pi subcommand: ensures pi (rick's agent
// runtime) is installed, the subagent + web-access extensions are registered,
// and rick's managed pi config (settings + theme) is in place. Idempotent —
// each step checks before acting and skips what is already done.
func NewInitPiCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "init-pi",
		Short: "Initialize pi (rick's agent runtime) + extensions + theme",
		Long: `Ensure pi is installed, required extensions are registered, and rick's
managed pi config is set up.

rick drives pi (@earendil-works/pi-coding-agent) as its agent runtime. This
command guarantees the runtime is ready: it installs pi if missing (via the
official installer), registers pi-subagents (Sub Agent delegation), registers
pi-web-access (external web search/fetch), keeps the managed config free of the
Tokyo Night package, and sets rick's managed theme. A final verification step
confirms everything is registered.

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
	// Step 0: prerequisite — only when rick's managed pi runtime is NOT yet
	// installed. pi is a Node.js program (>= 22.19.0) and its npm install needs
	// node/npm on PATH, so they must be present before rick can install pi.
	// rick does NOT install node — it is a user-managed environment dependency
	// (keeps rick's guidance simple and respects the user's environment). When
	// the managed runtime already exists, the environment is assumed ready and
	// this check is skipped.
	if !piagent.FileExists(piagent.RuntimeBin()) {
		if err := requireNodeForPiInstall(); err != nil {
			fmt.Fprintf(os.Stderr, "❌ %v\n", err)
			return err
		}
	}

	// Step 1: rick's self-contained pi runtime present (install if missing).
	piPath, piNewlyInstalled, err := ensurePI()
	if err != nil {
		// Fatal: without pi, rick cannot execute any agent command.
		fmt.Fprintf(os.Stderr, "❌ pi is not available and could not be installed: %v\n", err)
		fmt.Fprintf(os.Stderr, "   Install manually: npm install -g @earendil-works/pi-coding-agent\n")
		return err
	}
	ver := piVersion(piPath)
	fmt.Printf("✅ rick pi runtime ready: %s", piPath)
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

	// Step 4: purge the Tokyo Night package from the managed config. rick no
	// longer ships it: the bundled status-bar extension hard-codes Tokyo Night
	// colors (does not follow the active theme) and pollutes rick's agent
	// context. Any existing entry (string or filtered-object form) is removed
	// from packages and a tokyo theme is reverted to pi's default "dark".
	// Non-fatal.
	if err := purgeTokyoNight(); err != nil {
		fmt.Fprintf(os.Stderr, "⚠️  tokyo-night purge: %v\n", err)
		fmt.Fprintf(os.Stderr, "   rick works without it; the package may still be listed.\n")
	} else {
		fmt.Println("✅ tokyo-night purged from managed config (theme/packages)")
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

// bootstrapAgentSettings ensures the rick-managed pi agent dir exists and its
// settings.json carries rick's managed defaults. On first run in the managed
// dir it seeds theme/packages from the legacy ~/.pi/agent/settings.json (one-
// time migration — extensions themselves are re-installed into the managed dir
// by the steps below). Every later run only merges hideThinkingBlock=true in
// when missing, and seeds rick's default theme when no theme is set. Non-fatal
// on failure (callers warn and continue).
func bootstrapAgentSettings() error {
	if err := piagent.EnsureAgentDir(); err != nil {
		return fmt.Errorf("create agent dir %s: %w", piagent.AgentDir(), err)
	}
	path := piSettingsPath()
	if _, err := os.Stat(path); err == nil {
		if err := ensureHideThinkingBlock(path); err != nil {
			return err
		}
		return ensureRickTheme()
	}

	// First run in the managed dir: seed from legacy ~/.pi if present.
	base := map[string]any{"hideThinkingBlock": true}
	if data, err := os.ReadFile(legacyPiSettingsPath()); err == nil {
		var legacy map[string]any
		if json.Unmarshal(data, &legacy) == nil {
			if t, ok := legacy["theme"].(string); ok && t != "" && !containsString(tokyoNightThemes, t) {
				// tokyo-night themes are deliberately not carried over (the
				// package is purged — see purgeTokyoNight).
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
	return ensureRickTheme()
}

// ensureRickTheme seeds rick's default theme when the managed settings has no
// theme yet (fresh machine / fresh managed dir): the embedded rick.json is
// written into the managed themes dir (pi discovers agentDir/themes/*.json)
// and "theme": "rick" is set. An existing theme is never overridden.
func ensureRickTheme() error {
	if cur := currentTheme(); cur != "" {
		return nil
	}
	data, err := embeddedThemes.ReadFile("themes/rick.json")
	if err != nil {
		return fmt.Errorf("read embedded rick theme: %w", err)
	}
	themesDir := filepath.Join(piagent.AgentDir(), "themes")
	if err := os.MkdirAll(themesDir, 0755); err != nil {
		return fmt.Errorf("create themes dir: %w", err)
	}
	if err := os.WriteFile(filepath.Join(themesDir, "rick.json"), data, 0644); err != nil {
		return fmt.Errorf("write rick theme: %w", err)
	}
	if err := setTheme("rick"); err != nil {
		return fmt.Errorf("activate rick theme: %w", err)
	}
	return nil
}

// extensionManagedByRick reports whether a settings.json packages entry is one
// rick itself installs/verifies (so migration can carry it over) as opposed to
// a user's ad-hoc package that would not exist in the isolated dir. tokyo-night
// is deliberately excluded: rick purges it (see purgeTokyoNight).
func extensionManagedByRick(pkg string) bool {
	for _, p := range requiredExtensions {
		if strings.Contains(pkg, p) {
			return true
		}
	}
	return false
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

// purgeTokyoNight removes every trace of the Tokyo Night package from the
// managed settings.json: the packages entry (string or filtered-object form)
// is dropped and a tokyo-night theme is reverted to pi's built-in "dark".
// Other fields (hideThinkingBlock, remaining packages, theme) are preserved.
// Non-fatal — returns nil when the package is absent.
func purgeTokyoNight() error {
	path := piSettingsPath()
	data, err := os.ReadFile(path)
	if err != nil {
		return nil // no managed settings yet — nothing to purge
	}
	var s map[string]any
	if err := json.Unmarshal(data, &s); err != nil {
		return fmt.Errorf("parse %s: %w", path, err)
	}
	source := "npm:" + tokyoNightPkg
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
	if t, ok := s["theme"].(string); ok && containsString(tokyoNightThemes, t) {
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

func containsString(list []string, s string) bool {
	for _, v := range list {
		if v == s {
			return true
		}
	}
	return false
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
	// Prefer rick's self-contained runtime pi; fall back to PATH (e.g. before
	// the runtime is installed). All pi subprocesses get AgentEnv so config
	// stays isolated under ~/.rick/pi/agent.
	bin := "pi"
	if rb := piagent.RuntimeBin(); piagent.FileExists(rb) {
		bin = rb
	}
	cmd := exec.Command(bin, args...)
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

// ensurePI returns the path to rick's self-contained pi runtime binary,
// installing it via npm into ~/.rick/pi/agent/runtime if missing. When an
// existing global pi is on PATH, its version is matched so behavior stays
// identical (the global install itself is never modified). The second return is
// true iff pi was installed this call. Returns an error only if pi is still
// missing.
func ensurePI() (string, bool, error) {
	if bin := piagent.RuntimeBin(); piagent.FileExists(bin) {
		return bin, false, nil
	}
	// Prefer matching the version of an existing global pi (preserves known
	// behavior); otherwise install the latest.
	version := ""
	if p, err := exec.LookPath("pi"); err == nil {
		version = piVersion(p)
	}
	fmt.Printf("⚠️  rick's managed pi runtime missing — installing self-contained pi under %s", piagent.RuntimeDir())
	if version != "" {
		fmt.Printf(" (matching global v%s)", version)
	}
	fmt.Println(" ...")
	if err := installManagedPI(version); err != nil {
		return "", false, fmt.Errorf("install managed pi: %w", err)
	}
	bin := piagent.RuntimeBin()
	if !piagent.FileExists(bin) {
		return "", false, fmt.Errorf("managed pi still missing after install: %s", bin)
	}
	return bin, true, nil
}

// installManagedPI installs the pi package into rick's self-contained runtime
// dir (~/.rick/pi/agent/runtime) via `npm install --prefix`, so rick's pi is
// fully isolated from the user's global/standalone pi. version may be "" for
// latest; a failed pinned install falls back to latest (registry may have
// dropped the exact version).
func installManagedPI(version string) error {
	prefix := piagent.RuntimeDir()
	if err := os.MkdirAll(prefix, 0755); err != nil {
		return fmt.Errorf("create runtime dir: %w", err)
	}
	spec := "@earendil-works/pi-coding-agent"
	if version != "" {
		spec += "@" + version
	}
	cmd := exec.Command("npm", "install", "--prefix", prefix, "--no-fund", "--no-audit", spec)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		if version != "" {
			fmt.Println("⚠️  pinned install failed — retrying with latest version...")
			cmd = exec.Command("npm", "install", "--prefix", prefix, "--no-fund", "--no-audit", "@earendil-works/pi-coding-agent")
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			if err2 := cmd.Run(); err2 != nil {
				return fmt.Errorf("npm install @earendil-works/pi-coding-agent (pinned: %v): %w", err, err2)
			}
			return nil
		}
		return fmt.Errorf("npm install %s: %w", spec, err)
	}
	return nil
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
