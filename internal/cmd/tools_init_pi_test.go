package cmd

import (
	"os"
	"path/filepath"
	"testing"
)

// piListContains shells out to `pi list`; test it via a fake pi on PATH that
// prints a canned list, covering both the found and not-found cases. This
// helper backs ensureNpmExtension's idempotency check.
func TestPiListContains(t *testing.T) {
	tmp := t.TempDir()
	// fake pi: prints argv[1] routing — "list" prints canned packages.
	piScript := `#!/bin/sh
case "$1" in
  list) echo "User packages:"; echo "  pi-subagents"; echo "  pi-web-access";;
esac
`
	piPath := filepath.Join(tmp, "pi")
	if err := os.WriteFile(piPath, []byte(piScript), 0755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", tmp)

	if !piListContains("pi-subagents") {
		t.Error("expected piListContains(\"pi-subagents\") = true")
	}
	if !piListContains("pi-web-access") {
		t.Error("expected piListContains(\"pi-web-access\") = true")
	}
	if piListContains("not-installed-pkg") {
		t.Error("expected piListContains(\"not-installed-pkg\") = false")
	}
}

// verifyExtensions uses `pi list` to confirm all expected extensions are
// registered. Test it with a fake pi covering both all-present and
// missing-one cases (no real pi / LLM needed).
func TestVerifyExtensions(t *testing.T) {
	tmp := t.TempDir()

	t.Run("all_present", func(t *testing.T) {
		piScript := `#!/bin/sh
case "$1" in
  list) echo "User packages:"; echo "  pi-subagents"; echo "  pi-web-access";;
esac
`
		writeFakePi(t, tmp, piScript)
		t.Setenv("PATH", tmp)
		missing := verifyExtensions()
		if len(missing) != 0 {
			t.Errorf("expected no missing extensions, got %v", missing)
		}
	})

	t.Run("subagent_missing", func(t *testing.T) {
		piScript := `#!/bin/sh
case "$1" in
  list) echo "User packages:"; echo "  pi-web-access";;
esac
`
		writeFakePi(t, tmp, piScript)
		t.Setenv("PATH", tmp)
		missing := verifyExtensions()
		if len(missing) != 1 || missing[0] != "pi-subagents" {
			t.Errorf("expected [pi-subagents] missing, got %v", missing)
		}
	})

	t.Run("none_present", func(t *testing.T) {
		piScript := `#!/bin/sh
case "$1" in
  list) echo "No packages installed.";;
esac
`
		writeFakePi(t, tmp, piScript)
		t.Setenv("PATH", tmp)
		missing := verifyExtensions()
		if len(missing) != 2 {
			t.Errorf("expected 2 missing, got %v", missing)
		}
	})
}

func writeFakePi(t *testing.T, dir, script string) {
	t.Helper()
	piPath := filepath.Join(dir, "pi")
	if err := os.WriteFile(piPath, []byte(script), 0755); err != nil {
		t.Fatal(err)
	}
}
