package piagent

import (
	"os"
	"path/filepath"
)

// rickAgentDirEnv overrides where rick keeps its managed pi configuration.
// Tests set it to a temp dir so no real ~/.rick/pi is touched. It is NOT
// PI_CODING_AGENT_DIR: that is the env var pi itself reads, which rick
// injects into every pi subprocess (see AgentEnv).
const rickAgentDirEnv = "RICK_PI_AGENT_DIR"

// AgentDir returns the directory where rick manages pi's entire configuration
// (~/.rick/pi/agent). rick injects this path as PI_CODING_AGENT_DIR into every
// pi subprocess, so pi reads/writes its settings, extensions, themes and
// packages exclusively under ~/.rick/pi — fully isolated from the user's own
// ~/.pi (the user's standalone pi sessions keep their own config; rick's pi
// sessions are self-contained and cannot be polluted by user-installed
// extensions/skills).
func AgentDir() string {
	if d := os.Getenv(rickAgentDirEnv); d != "" {
		return d
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".rick", "pi", "agent")
}

// SettingsPath returns the path of the rick-managed pi settings.json.
func SettingsPath() string {
	return filepath.Join(AgentDir(), "settings.json")
}

// RuntimeDir returns the npm prefix of rick's self-contained pi runtime
// (agent/runtime). rick installs its own copy of the @earendil-works/
// pi-coding-agent package here so it can patch pi's UI behavior (e.g. the diff
// keyword highlight) without touching the user's global/standalone pi install.
// Keeping it under AgentDir gives tests the same RICK_PI_AGENT_DIR isolation.
func RuntimeDir() string {
	return filepath.Join(AgentDir(), "runtime")
}

// RuntimeBin returns the pi binary shim of the managed runtime
// (agent/runtime/node_modules/.bin/pi). Empty/invalid until installed.
func RuntimeBin() string {
	return filepath.Join(RuntimeDir(), "node_modules", ".bin", "pi")
}

// FileExists reports whether path exists and is not a directory.
func FileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

// EnsureAgentDir creates the rick-managed pi agent directory (and parents).
func EnsureAgentDir() error {
	return os.MkdirAll(AgentDir(), 0755)
}

// AgentEnv returns the environment for pi subprocesses: the inherited
// environment plus PI_CODING_AGENT_DIR pointing at rick's managed agent dir.
// This is what makes the configuration isolation effective at every call site
// (CallCLI for interactive plan/easy/ctrl/human-loop, Executor for doing's
// --mode json, and rick's own `pi install`/`pi list`/`pi --version` runs).
func AgentEnv() []string {
	return append(os.Environ(), "PI_CODING_AGENT_DIR="+AgentDir())
}
