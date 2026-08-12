package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/spf13/cobra"
)

// piInstallerURL is the official pi installation script.
const piInstallerURL = "https://pi.dev/install.sh"

// NewInitPiCmd creates the init-pi subcommand: ensures pi (rick's agent
// runtime) is installed and the subagent + web-access extensions are registered.
// Idempotent — each step checks before acting and skips what is already done.
func NewInitPiCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "init-pi",
		Short: "Initialize pi (rick's agent runtime) + subagent extension",
		Long: `Ensure pi is installed and the subagent + web-access extensions are registered.

rick drives pi (@earendil-works/pi-coding-agent) as its agent runtime. This
command guarantees the runtime is ready: it installs pi if missing (via the
official installer), then registers pi's subagent extension (enables rick's
Sub Agent per-iteration delegation) and the pi-web-access extension (enables
external web search/fetch).

Idempotent: every step checks first and skips what is already satisfied.
Non-fatal: a missing extension does not block rick; pi being entirely missing
is the only fatal condition.

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

// runInitPi runs the two-step initialization and returns an error only when pi
// is unavailable and rick therefore cannot run. Missing extensions are warned
// about, not fatal.
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

	// Step 4: verify all required extensions are actually registered. This is a
	// final integrity check via `pi list` — it catches the case where an install
	// appeared to succeed but the extension is not registered (e.g. the old
	// `pi install <bare-source-dir>` silently wrote to settings.json without the
	// loader recognizing it).
	missing := verifyExtensions()
	if len(missing) > 0 {
		fmt.Fprintf(os.Stderr, "⚠️  verification: these extensions are NOT registered: %s\n", strings.Join(missing, ", "))
		fmt.Fprintf(os.Stderr, "   rick may be degraded. Re-run `rick tools init-pi` or install manually.\n")
	} else {
		fmt.Println("✅ verification: all required extensions registered")
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
