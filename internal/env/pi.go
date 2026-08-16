package env

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/sunquan/rick/internal/runtime"
)

// PiCommand builds a pi subprocess that runs against rick's managed agent dir
// (PI_CODING_AGENT_DIR=~/.rick/pi/agent). Every `pi install` / `pi list` /
// `pi --version` invocation must use it so installs and checks always act on
// rick's own configuration, never the user's ~/.pi.
func PiCommand(args ...string) *exec.Cmd {
	// Prefer rick's self-contained runtime pi; fall back to PATH (e.g. before
	// the runtime is installed). All pi subprocesses get AgentEnv so config
	// stays isolated under ~/.rick/pi/agent.
	bin := "pi"
	if rb := runtime.RuntimeBin(); runtime.FileExists(rb) {
		bin = rb
	}
	cmd := exec.Command(bin, args...)
	cmd.Env = runtime.AgentEnv()
	return cmd
}

// EnsurePI returns the path to rick's self-contained pi runtime binary,
// installing it via npm into ~/.rick/pi/agent/runtime if missing. When an
// existing global pi is on PATH, its version is matched so behavior stays
// identical (the global install itself is never modified). The second return is
// true iff pi was installed this call. Returns an error only if pi is still
// missing.
func EnsurePI() (string, bool, error) {
	if bin := runtime.RuntimeBin(); runtime.FileExists(bin) {
		return bin, false, nil
	}
	// Prefer matching the version of an existing global pi (preserves known
	// behavior); otherwise install the latest.
	version := ""
	if p, err := exec.LookPath("pi"); err == nil {
		version = PiVersion(p)
	}
	fmt.Printf("⚠️  rick's managed pi runtime missing — installing self-contained pi under %s", runtime.RuntimeDir())
	if version != "" {
		fmt.Printf(" (matching global v%s)", version)
	}
	fmt.Println(" ...")
	if err := InstallManagedPI(version); err != nil {
		return "", false, fmt.Errorf("install managed pi: %w", err)
	}
	bin := runtime.RuntimeBin()
	if !runtime.FileExists(bin) {
		return "", false, fmt.Errorf("managed pi still missing after install: %s", bin)
	}
	return bin, true, nil
}

// InstallManagedPI installs the pi package into rick's self-contained runtime
// dir (~/.rick/pi/agent/runtime) via `npm install --prefix`, so rick's pi is
// fully isolated from the user's global/standalone pi. version may be "" for
// latest; a failed pinned install falls back to latest (registry may have
// dropped the exact version).
func InstallManagedPI(version string) error {
	prefix := runtime.RuntimeDir()
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

// RequireNodeForPiInstall checks that node and npm are on PATH before rick
// installs pi. pi is a Node.js program (>= 22.19.0) and its installer shells
// out to npm. rick treats node as a user-managed environment dependency — it
// does NOT install node (keeps rick's guidance simple; respects the user's
// environment). Returns a fatal error with install instructions if missing.
func RequireNodeForPiInstall() error {
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

// PiVersion returns pi's version string (e.g. "0.84.1"), or "" if it cannot be
// determined. Best-effort — init-pi does not gate on version.
func PiVersion(piPath string) string {
	cmd := PiCommand("--version")
	if piPath != "" {
		cmd.Path = piPath
	}
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}
