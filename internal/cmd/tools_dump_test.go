package cmd

import (
	"strings"
	"testing"

	"github.com/sunquan/rick/internal/env/relocate"
)

// ─── dump / import command registration & wiring ────────────────────────────

func TestNewDumpCmd(t *testing.T) {
	cmd := NewDumpCmd("test-version")
	if cmd == nil {
		t.Fatal("NewDumpCmd returned nil")
	}
	if cmd.Use != "dump" {
		t.Errorf("expected Use='dump', got '%s'", cmd.Use)
	}
	// key flags exist
	for _, f := range []string{"workspace", "output", "sessions", "slim", "goos", "no-exclude", "state-dir"} {
		if cmd.Flags().Lookup(f) == nil {
			t.Errorf("dump flag missing: %s", f)
		}
	}
	// default sessions policy + platform list
	if got := cmd.Flags().Lookup("sessions").DefValue; got != relocate.SessionsPolicyWorkspaces {
		t.Errorf("sessions default = %q, want %q", got, relocate.SessionsPolicyWorkspaces)
	}
	if got := cmd.Flags().Lookup("goos").DefValue; !strings.Contains(got, "darwin/arm64") {
		t.Errorf("goos default = %q, want the cross-platform set", got)
	}
}

func TestNewImportCmd(t *testing.T) {
	cmd := NewImportCmd()
	if cmd == nil {
		t.Fatal("NewImportCmd returned nil")
	}
	if cmd.Use != "import <archive>" {
		t.Errorf("unexpected Use: %s", cmd.Use)
	}
	for _, f := range []string{"target", "state-dir", "force"} {
		if cmd.Flags().Lookup(f) == nil {
			t.Errorf("import flag missing: %s", f)
		}
	}
	// positional arg contract: exactly one (the archive)
	if cmd.Args == nil {
		t.Error("import must validate its positional args")
	}
}

func TestToolsCmdListsDumpImport(t *testing.T) {
	cmd := NewToolsCmd("test")
	subNames := map[string]bool{}
	for _, sub := range cmd.Commands() {
		subNames[sub.Name()] = true
	}
	for _, name := range []string{"dump", "import", "theme", "init-pi", "release", "rsi_check"} {
		if !subNames[name] {
			t.Errorf("tools missing subcommand: %s", name)
		}
	}
	// Long help documents the new commands (discoverability contract from
	// tools_integration_test.sh scenario 11)
	if !strings.Contains(cmd.Long, "dump") || !strings.Contains(cmd.Long, "import") {
		t.Error("tools Long help does not mention dump/import")
	}
	if !strings.Contains(cmd.Long, "theme") {
		t.Error("tools Long help does not mention theme (known drift — fix included)")
	}
}

func TestDumpCmdRejectsPositionalArgs(t *testing.T) {
	cmd := NewDumpCmd("test")
	cmd.SetArgs([]string{"some/path"})
	if err := cmd.Execute(); err == nil {
		t.Error("dump must reject positional args (use --workspace)")
	}
}
