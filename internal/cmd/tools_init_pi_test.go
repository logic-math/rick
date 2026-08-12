package cmd

import (
	"os"
	"path/filepath"
	"testing"
)

// findSubagentSource uses exec.LookPath("pi") + filepath.EvalSymlinks to walk
// from the pi binary to its package's examples/extensions/subagent dir.
// These tests build a mock pi install tree (a symlink chain like the real one)
// and verify the resolution, without needing a real pi or LLM.

// setupMockPiTree builds:
//
//	tmpDir/
//	  bin/pi -> ../lib/<pkg>/dist/cli.js   (symlink, like ~/.local/bin/pi)
//	  lib/<pkg>/
//	    dist/cli.js                         (fake binary)
//	    examples/extensions/subagent/       (the extension dir)
//
// and returns tmpDir so the caller can prepend tmpDir/bin to PATH.
func setupMockPiTree(t *testing.T) string {
	t.Helper()
	tmp := t.TempDir()

	pkgDir := filepath.Join(tmp, "lib", "@earendil-works", "pi-coding-agent")
	distDir := filepath.Join(pkgDir, "dist")
	if err := os.MkdirAll(distDir, 0755); err != nil {
		t.Fatal(err)
	}
	// fake cli.js (the real pi binary lives here; content irrelevant, but it
	// must be executable so exec.LookPath resolves the bin/pi symlink).
	cliJS := filepath.Join(distDir, "cli.js")
	if err := os.WriteFile(cliJS, []byte("#!/bin/sh\n"), 0755); err != nil {
		t.Fatal(err)
	}
	// subagent extension dir (what findSubagentSource looks for)
	subagentDir := filepath.Join(pkgDir, subagentRelPath)
	if err := os.MkdirAll(subagentDir, 0755); err != nil {
		t.Fatal(err)
	}

	// bin/pi -> ../lib/<pkg>/dist/cli.js  (mimics ~/.local/bin/pi symlink)
	binDir := filepath.Join(tmp, "bin")
	if err := os.MkdirAll(binDir, 0755); err != nil {
		t.Fatal(err)
	}
	piLink := filepath.Join(binDir, "pi")
	if err := os.Symlink(cliJS, piLink); err != nil {
		t.Fatal(err)
	}

	return tmp
}

func TestFindSubagentSource_ResolvesSymlinkChain(t *testing.T) {
	tmp := setupMockPiTree(t)
	t.Setenv("PATH", filepath.Join(tmp, "bin"))

	got, err := findSubagentSource()
	if err != nil {
		t.Fatalf("findSubagentSource: %v", err)
	}
	want := filepath.Join(tmp, "lib", "@earendil-works", "pi-coding-agent", subagentRelPath)
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestFindSubagentSource_ErrorsWhenSubagentDirMissing(t *testing.T) {
	// pi installed but no examples/extensions/subagent (e.g. future Bun binary).
	tmp := setupMockPiTree(t)
	subagentDir := filepath.Join(tmp, "lib", "@earendil-works", "pi-coding-agent", subagentRelPath)
	if err := os.RemoveAll(subagentDir); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", filepath.Join(tmp, "bin"))

	if _, err := findSubagentSource(); err == nil {
		t.Error("expected error when subagent dir is missing")
	}
}

func TestFindSubagentSource_ErrorsWhenPiNotOnPath(t *testing.T) {
	t.Setenv("PATH", "/nonexistent-empty-path")
	if _, err := findSubagentSource(); err == nil {
		t.Error("expected error when pi is not on PATH")
	}
}

// piListContains shells out to `pi list`; test it via a fake pi on PATH that
// prints a canned list, covering both the found and not-found cases.
func TestPiListContains(t *testing.T) {
	tmp := t.TempDir()
	// fake pi: prints argv[1] routing — "list" prints canned packages.
	piScript := `#!/bin/sh
case "$1" in
  list) echo "User packages:"; echo "  some/subagent"; echo "  other/pkg";;
esac
`
	piPath := filepath.Join(tmp, "pi")
	if err := os.WriteFile(piPath, []byte(piScript), 0755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", tmp)

	if !piListContains("subagent") {
		t.Error("expected piListContains(\"subagent\") = true")
	}
	if piListContains("not-installed-pkg") {
		t.Error("expected piListContains(\"not-installed-pkg\") = false")
	}
}
