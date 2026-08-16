package env

import (
	"fmt"
	"os"
	"strings"
)

// RequiredExtensions is the full set of pi extensions rick depends on.
var RequiredExtensions = []string{"pi-subagents", "pi-web-access"}

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

// EnsureNpmExtension registers an npm-based pi extension if it is not already
// installed. pkg is the npm spec passed to `pi install` (e.g. "pi-web-access");
// detect is the substring `pi list` is grepped for (usually the package name
// without the npm: prefix). Non-fatal on failure — callers warn and continue.
func EnsureNpmExtension(pkg, detect string) error {
	if PiListContains(detect) {
		return nil // already registered
	}
	fmt.Printf("⚠️  %s not registered — installing via pi install npm:%s\n", pkg, pkg)
	cmd := PiCommand("install", "npm:"+pkg)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("pi install npm:%s: %w", pkg, err)
	}
	if !PiListContains(detect) {
		return fmt.Errorf("%s still not listed after install", pkg)
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
