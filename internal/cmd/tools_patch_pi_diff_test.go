package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// buildFakePiTree creates a fake pi install: tmp/bin/pi (symlink into a fake
// package root) with the given diff.js content at the pi layout path. Returns
// the bin dir (put on PATH) and the package root.
func buildFakePiTree(t *testing.T, diffSrc string) (binDir, pkgRoot string) {
	t.Helper()
	tmp := t.TempDir()
	pkgRoot = filepath.Join(tmp, "pi-pkg")
	diffDir := filepath.Join(pkgRoot, "dist", "modes", "interactive", "components")
	if err := os.MkdirAll(diffDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(diffDir, "diff.js"), []byte(diffSrc), 0644); err != nil {
		t.Fatal(err)
	}
	cli := filepath.Join(pkgRoot, "dist", "cli.js")
	if err := os.WriteFile(cli, []byte("#!/usr/bin/env node\n"), 0755); err != nil {
		t.Fatal(err)
	}
	binDir = filepath.Join(tmp, "bin")
	if err := os.MkdirAll(binDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(cli, filepath.Join(binDir, "pi")); err != nil {
		t.Fatal(err)
	}
	return binDir, pkgRoot
}

// unpatchedDiffSrc mirrors pi 0.84.x diff.js: two theme.inverse(value) calls.
const unpatchedDiffSrc = `import { theme } from "../theme/theme.js";
function renderIntraLineDiff(oldContent, newContent) {
    for (const part of wordDiff) {
        if (part.removed) {
            removedLine += theme.inverse(value);
        } else if (part.added) {
            addedLine += theme.inverse(value);
        }
    }
}`

func TestPiPackageRoot(t *testing.T) {
	binDir, pkgRoot := buildFakePiTree(t, unpatchedDiffSrc)
	piPath := filepath.Join(binDir, "pi")

	got, err := piPackageRoot(piPath)
	if err != nil {
		t.Fatalf("piPackageRoot: %v", err)
	}
	if got != pkgRoot {
		t.Errorf("piPackageRoot = %q, want %q", got, pkgRoot)
	}

	// Unrecognized layout -> error.
	empty := t.TempDir()
	if _, err := piPackageRoot(filepath.Join(empty, "nonexistent")); err == nil {
		t.Error("expected error for unrecognized pi path")
	}
}

func TestRunPatchPIDiffPatches(t *testing.T) {
	binDir, pkgRoot := buildFakePiTree(t, unpatchedDiffSrc)
	t.Setenv("PATH", binDir)

	var buf bytes.Buffer
	if err := runPatchPIDiff(&buf); err != nil {
		t.Fatalf("runPatchPIDiff: %v", err)
	}
	got, err := os.ReadFile(filepath.Join(pkgRoot, diffComponentRel))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(got), patchPIDiffOld) {
		t.Error("diff.js still contains theme.inverse( after patch")
	}
	if !strings.Contains(string(got), patchPIDiffNew) {
		t.Error("diff.js missing theme.underline( after patch")
	}
	if !strings.Contains(buf.String(), "patched pi diff highlight") {
		t.Errorf("unexpected output: %q", buf.String())
	}
}

func TestRunPatchPIDiffIdempotent(t *testing.T) {
	binDir, pkgRoot := buildFakePiTree(t, unpatchedDiffSrc)
	t.Setenv("PATH", binDir)

	if err := runPatchPIDiff(&bytes.Buffer{}); err != nil {
		t.Fatalf("first run: %v", err)
	}
	afterFirst, err := os.ReadFile(filepath.Join(pkgRoot, diffComponentRel))
	if err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	if err := runPatchPIDiff(&buf); err != nil {
		t.Fatalf("second run: %v", err)
	}
	if !strings.Contains(buf.String(), "already patched") {
		t.Errorf("second run should report already patched, got: %q", buf.String())
	}
	afterSecond, err := os.ReadFile(filepath.Join(pkgRoot, diffComponentRel))
	if err != nil {
		t.Fatal(err)
	}
	if string(afterFirst) != string(afterSecond) {
		t.Error("second run modified diff.js (not idempotent)")
	}
}

func TestRunPatchPIDiffAlreadyPatched(t *testing.T) {
	binDir, _ := buildFakePiTree(t, strings.ReplaceAll(unpatchedDiffSrc, patchPIDiffOld, patchPIDiffNew))
	t.Setenv("PATH", binDir)

	var buf bytes.Buffer
	if err := runPatchPIDiff(&buf); err != nil {
		t.Fatalf("runPatchPIDiff: %v", err)
	}
	if !strings.Contains(buf.String(), "already patched") {
		t.Errorf("expected already-patched report, got: %q", buf.String())
	}
}

func TestRunPatchPIDiffNoInverseCalls(t *testing.T) {
	// pi layout changed upstream: diff.js exists but has no theme.inverse(.
	binDir, _ := buildFakePiTree(t, "export function renderDiff(t) { return t; }\n")
	t.Setenv("PATH", binDir)

	var buf bytes.Buffer
	if err := runPatchPIDiff(&buf); err != nil {
		t.Fatalf("runPatchPIDiff: %v", err)
	}
	if !strings.Contains(buf.String(), "nothing to patch") {
		t.Errorf("expected nothing-to-patch warning, got: %q", buf.String())
	}
}

func TestRunPatchPIDiffPiMissing(t *testing.T) {
	t.Setenv("PATH", t.TempDir()) // no pi on PATH
	var buf bytes.Buffer
	if err := runPatchPIDiff(&buf); err == nil {
		t.Error("expected error when pi is not on PATH")
	}
}
