package piagent

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestAgentDir_UnderHomeRickPi(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv(rickAgentDirEnv, "")
	want := filepath.Join(home, ".rick", "pi", "agent")
	if got := AgentDir(); got != want {
		t.Errorf("AgentDir: want %q, got %q", want, got)
	}
}

func TestAgentDir_EnvOverride(t *testing.T) {
	want := t.TempDir()
	t.Setenv(rickAgentDirEnv, want)
	if got := AgentDir(); got != want {
		t.Errorf("AgentDir with %s: want %q, got %q", rickAgentDirEnv, want, got)
	}
}

func TestSettingsPath_UnderAgentDir(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv(rickAgentDirEnv, "")
	if got := SettingsPath(); got != filepath.Join(home, ".rick", "pi", "agent", "settings.json") {
		t.Errorf("SettingsPath: got %q", got)
	}
}

func TestEnsureAgentDir_CreatesDir(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "nested", "pi", "agent")
	t.Setenv(rickAgentDirEnv, dir)
	if err := EnsureAgentDir(); err != nil {
		t.Fatalf("EnsureAgentDir: %v", err)
	}
	if st, err := os.Stat(dir); err != nil || !st.IsDir() {
		t.Errorf("agent dir not created: %v", err)
	}
}

func TestAgentEnv_ContainsIsolationVar(t *testing.T) {
	dir := t.TempDir()
	t.Setenv(rickAgentDirEnv, dir)
	env := AgentEnv()
	found := false
	for _, kv := range env {
		if kv == "PI_CODING_AGENT_DIR="+dir {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("AgentEnv missing PI_CODING_AGENT_DIR=%s; got %v", dir, env)
	}
	// Inherited env must survive (prepend, not replace).
	if !containsEnvPrefix(env, "PATH=") {
		t.Errorf("AgentEnv dropped inherited PATH")
	}
}

func containsEnvPrefix(env []string, prefix string) bool {
	for _, kv := range env {
		if strings.HasPrefix(kv, prefix) {
			return true
		}
	}
	return false
}
