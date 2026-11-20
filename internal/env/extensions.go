package env

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/sunquan/rick/internal/runtime"
)

// RequiredExtensions is the full set of pi extensions rick depends on,
// derived from PinnedExtensions (single source: the pin table owns both the
// names and the fixed versions).
var RequiredExtensions = pinnedExtensionNames()

// VerifyExtensions runs `pi list` and returns the names of required extensions
// that are NOT registered. Empty result means all present.
func VerifyExtensions() []string {
	var missing []string
	for _, pkg := range RequiredExtensions {
		if !PiListContains(pkg) {
			missing = append(missing, pkg)
		}
	}
	return missing
}

// InstalledExtensionVersion returns the version of an installed extension
// by reading AgentDir()/npm/node_modules/<pkg>/package.json (the actual npm
// install rick manages), or "" when it cannot be determined (not installed /
// unreadable / no version field). pi list prints versions too, but the
// package.json is the deterministic source without shelling out.
func InstalledExtensionVersion(pkg string) string {
	if pkg == "" {
		return ""
	}
	manifest := filepath.Join(runtime.AgentDir(), "npm", "node_modules", pkg, "package.json")
	data, err := os.ReadFile(manifest)
	if err != nil {
		return ""
	}
	var m struct {
		Version string `json:"version"`
	}
	if err := json.Unmarshal(data, &m); err != nil {
		return ""
	}
	return m.Version
}

// EnsureNpmExtension registers an npm-based pi extension if it is not already
// installed, and re-installs the PINNED version when an installed version has
// drifted from the pin table (job_55: pi-subagents 0.89.1 vs pinned 0.51.0
// broke pi 0.84.2 — installs are always version-pinned and drift self-heals).
// pkg is the npm package name; detect is the substring `pi list` is grepped
// for (usually the package name without the npm: prefix). Non-fatal on
// failure — callers warn and continue.
func EnsureNpmExtension(pkg, detect string) error {
	if PiListContains(detect) {
		// Already registered — verify the installed version matches the pin.
		// InstalledExtensionVersion "" (unreadable/missing) is tolerated: the
		// extension is registered, we just cannot verify the version.
		want := PinnedExtensionVersion(pkg)
		if want == "" {
			return nil // not a pinned extension (defensive): registration suffices
		}
		if got := InstalledExtensionVersion(pkg); got != "" && got != want {
			fmt.Printf("⚠️  %s 版本漂移（installed %s ≠ pinned %s）——重装固定版本\n", pkg, got, want)
			if err := installNpmSpec(PinnedExtensionSpec(pkg)); err != nil {
				return err
			}
			if got := InstalledExtensionVersion(pkg); got != want {
				return fmt.Errorf("%s still at %s after pinned reinstall (want %s)", pkg, got, want)
			}
		}
		return nil
	}
	// Not registered — install. Pinned packages install their fixed spec
	// ("npm:<pkg>@<version>"); unpinned names (defensive, not in the pin
	// table) keep the legacy unpinned behavior.
	spec := "npm:" + pkg
	if pinned := PinnedExtensionSpec(pkg); pinned != "" {
		spec = pinned
	}
	fmt.Printf("⚠️  %s not registered — installing via pi install %s\n", pkg, spec)
	if err := installNpmSpec(spec); err != nil {
		return err
	}
	if !PiListContains(detect) {
		return fmt.Errorf("%s still not listed after install", pkg)
	}
	return nil
}

// installNpmSpec runs `pi install <spec>` against rick's managed agent dir
// (inheriting stdio). spec must be the full registration form
// ("npm:pi-subagents@0.51.0").
func installNpmSpec(spec string) error {
	cmd := PiCommand("install", spec)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("pi install %s: %w", spec, err)
	}
	return nil
}

// PiListContains reports whether `pi list` output contains substr. Used to
// detect installed extensions idempotently (EnsureNpmExtension + VerifyExtensions).
func PiListContains(substr string) bool {
	out, err := PiCommand("list").Output()
	if err != nil {
		return false
	}
	return strings.Contains(string(out), substr)
}
