package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// buildFakePiTree creates a fake pi install: tmp/bin/pi (symlink into a fake
// package root) with diff.js + bash.js at the pi layout paths. Returns the bin
// dir (put on PATH) and the package root.
func buildFakePiTree(t *testing.T, diffSrc, bashSrc string) (binDir, pkgRoot string) {
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
	toolsDir := filepath.Join(pkgRoot, "dist", "core", "tools")
	if err := os.MkdirAll(toolsDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(toolsDir, "bash.js"), []byte(bashSrc), 0644); err != nil {
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

// unpatchedDiffSrc mirrors pi 0.84.x diff.js anchors (extracted verbatim from
// the stock install). JS template literals (backticks) are spliced in via
// interpreted segments.
const unpatchedDiffSrc = `import * as Diff from "diff";
import { theme } from "../theme/theme.js";
export function renderDiff(diffText, _options = {}) {
    const lines = diffText.split("` + "\\n" + `");
    const result = [];
    if (true) {
        removedLine += theme.inverse(value);
        addedLine += theme.inverse(value);
    }
            if (removedLines.length === 1 && addedLines.length === 1) {
                const removed = removedLines[0];
                const added = addedLines[0];
                const { removedLine, addedLine } = renderIntraLineDiff(replaceTabs(removed.content), replaceTabs(added.content));
                result.push(theme.fg("toolDiffRemoved", ` + "`-${removed.lineNum} ${removedLine}`" + `));
                result.push(theme.fg("toolDiffAdded", ` + "`+${added.lineNum} ${addedLine}`" + `));
            }
            else {
                // Show all removed lines first, then all added lines
                for (const removed of removedLines) {
                    result.push(theme.fg("toolDiffRemoved", ` + "`-${removed.lineNum} ${replaceTabs(removed.content)}`" + `));
                }
                for (const added of addedLines) {
                    result.push(theme.fg("toolDiffAdded", ` + "`+${added.lineNum} ${replaceTabs(added.content)}`" + `));
                }
            }
        else if (parsed.prefix === "+") {
            // Standalone added line
            result.push(theme.fg("toolDiffAdded", ` + "`+${parsed.lineNum} ${replaceTabs(parsed.content)}`" + `));
            i++;
        }
        else {
            // Context line
            result.push(theme.fg("toolDiffContext", ` + "` ${parsed.lineNum} ${replaceTabs(parsed.content)}`" + `));
            i++;
        }
}`

// unpatchedBashSrc mirrors pi 0.84.x bash.js formatBashCall anchors.
const unpatchedBashSrc = `import { theme } from "../../modes/interactive/theme/theme.js";
function formatBashCall(args) {
    const command = str(args?.command);
    const timeout = args?.timeout;
    const timeoutSuffix = timeout ? theme.fg("muted", ` + "` (timeout ${timeout}s)`" + `) : "";
    const commandDisplay = command === null ? invalidArgText(theme) : command ? command : theme.fg("toolOutput", "...");
    return theme.fg("toolTitle", theme.bold(` + "`$ ${commandDisplay}`" + `)) + timeoutSuffix;
}`

func TestPiPackageRoot(t *testing.T) {
	binDir, pkgRoot := buildFakePiTree(t, unpatchedDiffSrc, unpatchedBashSrc)
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
	// Isolate the managed runtime dir so FindBinary resolves the fake pi on
	// PATH, never the real managed runtime.
	t.Setenv("RICK_PI_AGENT_DIR", t.TempDir())
	binDir, pkgRoot := buildFakePiTree(t, unpatchedDiffSrc, unpatchedBashSrc)
	t.Setenv("PATH", binDir)

	var buf bytes.Buffer
	if err := runPatchPIDiff(&buf); err != nil {
		t.Fatalf("runPatchPIDiff: %v", err)
	}
	if !strings.Contains(buf.String(), "patch(es) applied") {
		t.Errorf("expected applied report, got: %q", buf.String())
	}

	diffText := readFile(t, filepath.Join(pkgRoot, diffComponentRel))
	if strings.Contains(diffText, "theme.inverse(") {
		t.Error("diff.js still contains theme.inverse( after patch")
	}
	if !strings.Contains(diffText, "removedLine += theme.bold(value);") {
		t.Error("diff.js missing bold keyword patch")
	}
	for _, want := range []string{
		`getLanguageFromPath`, `highlightCode`, `supportsLanguage`,
		`const lang = _options?.filePath`, `const hl = lang && supportsLanguage(lang)`,
		`if (hl) {`, `highlightCode(replaceTabs(removed.content), lang)[0]`,
		`highlightCode(replaceTabs(added.content), lang)[0]`,
		`const addedContent = hl ? highlightCode(replaceTabs(parsed.content), lang)[0] : replaceTabs(parsed.content);`,
		`const contextContent = hl ? highlightCode(replaceTabs(parsed.content), lang)[0] : replaceTabs(parsed.content);`,
	} {
		if !strings.Contains(diffText, want) {
			t.Errorf("diff.js missing syntax-highlight fragment: %q", want)
		}
	}

	bashText := readFile(t, filepath.Join(pkgRoot, bashComponentRel))
	for _, want := range []string{
		`highlightShellCommand`, `supportsLanguage("bash")`, `highlightCode(command, "bash")`,
		`command ? highlightShellCommand(command) : theme.fg("toolOutput", "...")`,
	} {
		if !strings.Contains(bashText, want) {
			t.Errorf("bash.js missing command-highlight fragment: %q", want)
		}
	}
}

func TestRunPatchPIDiffIdempotent(t *testing.T) {
	t.Setenv("RICK_PI_AGENT_DIR", t.TempDir())
	binDir, pkgRoot := buildFakePiTree(t, unpatchedDiffSrc, unpatchedBashSrc)
	t.Setenv("PATH", binDir)

	if err := runPatchPIDiff(&bytes.Buffer{}); err != nil {
		t.Fatalf("first run: %v", err)
	}
	diffAfterFirst := readFile(t, filepath.Join(pkgRoot, diffComponentRel))
	bashAfterFirst := readFile(t, filepath.Join(pkgRoot, bashComponentRel))

	var buf bytes.Buffer
	if err := runPatchPIDiff(&buf); err != nil {
		t.Fatalf("second run: %v", err)
	}
	if !strings.Contains(buf.String(), "already patched") {
		t.Errorf("second run should report already patched, got: %q", buf.String())
	}
	if readFile(t, filepath.Join(pkgRoot, diffComponentRel)) != diffAfterFirst {
		t.Error("second run modified diff.js (not idempotent)")
	}
	if readFile(t, filepath.Join(pkgRoot, bashComponentRel)) != bashAfterFirst {
		t.Error("second run modified bash.js (not idempotent)")
	}
}

func TestRunPatchPIDiffAlreadyPatched(t *testing.T) {
	// Sources with none of the old anchors (already patched or pi changed
	// upstream) are a no-op, not an error.
	t.Setenv("RICK_PI_AGENT_DIR", t.TempDir())
	binDir, pkgRoot := buildFakePiTree(t, "export function renderDiff(t) { return t; }\n",
		"export function formatBashCall(a) { return a; }\n")
	t.Setenv("PATH", binDir)

	var buf bytes.Buffer
	if err := runPatchPIDiff(&buf); err != nil {
		t.Fatalf("runPatchPIDiff: %v", err)
	}
	if !strings.Contains(buf.String(), "already patched") {
		t.Errorf("expected already-patched report, got: %q", buf.String())
	}
	if got := readFile(t, filepath.Join(pkgRoot, diffComponentRel)); !strings.Contains(got, "renderDiff(t)") {
		t.Error("no-op run must not modify diff.js")
	}
}

func TestRunPatchPIDiffPiMissing(t *testing.T) {
	t.Setenv("PATH", t.TempDir()) // no pi on PATH
	t.Setenv("RICK_PI_AGENT_DIR", t.TempDir())
	var buf bytes.Buffer
	if err := runPatchPIDiff(&buf); err == nil {
		t.Error("expected error when pi is not on PATH")
	}
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}
