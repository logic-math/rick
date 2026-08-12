package cmd

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/spf13/cobra"

	"github.com/sunquan/rick/internal/agent/piagent"
)

// rick patches pi's TUI rendering to its taste. pi exposes only theme tokens,
// not per-renderer behavior, so rick rewrites the two renderers that matter:
//
//   - dist/modes/interactive/components/diff.js — the edit-tool diff:
//     (a) intra-line changed keywords use ANSI reverse video (theme.inverse),
//     painting their background red/green on top of the line color → rick
//     switches them to bold; (b) the diff content is flat colored text →
//     rick adds real syntax highlighting (per file extension, via pi's own
//     highlight.js pipeline used by markdown/read-tool rendering).
//   - dist/core/tools/bash.js — the bash tool block shows the command flat in
//     toolTitle color → rick adds shell syntax highlighting to the command.
//
// Every patch is a {file, old, new} string replacement; each old string
// disappears once applied, so re-running is a no-op (idempotent) and a pi
// upgrade that changes the layout simply skips (warn, non-fatal). Only rick's
// self-contained runtime (~/.rick/pi/agent/runtime) is touched — the user's
// global/standalone pi is never modified.

const (
	// diffComponentRel is pi's edit-tool diff renderer.
	diffComponentRel = "dist/modes/interactive/components/diff.js"
	// bashComponentRel is pi's bash tool renderer (command block title).
	bashComponentRel = "dist/core/tools/bash.js"
)

// piPatch is one idempotent string replacement inside a pi runtime file.
type piPatch struct {
	file string // package-relative path
	old  string
	new  string
	note string
}

// piRuntimePatches is the ordered manifest of rick's pi UI patches.
var piRuntimePatches = []piPatch{
	// --- diff.js: keyword highlight reverse-video -> bold ---
	{
		file: diffComponentRel,
		old:  "theme.inverse(",
		new:  "theme.bold(",
		note: "diff keyword reverse-video bg -> bold",
	},
	// --- diff.js: syntax highlighting (imports + lang resolution + render) ---
	{
		file: diffComponentRel,
		old: `import * as Diff from "diff";
import { theme } from "../theme/theme.js";`,
		new: `import * as Diff from "diff";
import { theme, getLanguageFromPath, highlightCode } from "../theme/theme.js";
import { supportsLanguage } from "../../../utils/syntax-highlight.js";`,
		note: "diff syntax highlight imports",
	},
	{
		file: diffComponentRel,
		old: `export function renderDiff(diffText, _options = {}) {
    const lines = diffText.split("\n");
    const result = [];`,
		new: `export function renderDiff(diffText, _options = {}) {
    const lines = diffText.split("\n");
    const lang = _options?.filePath ? getLanguageFromPath(_options.filePath) : undefined;
    const hl = lang && supportsLanguage(lang);
    const result = [];`,
		note: "diff syntax highlight lang resolution",
	},
	{
		file: diffComponentRel,
		old: `            if (removedLines.length === 1 && addedLines.length === 1) {
                const removed = removedLines[0];
                const added = addedLines[0];
                const { removedLine, addedLine } = renderIntraLineDiff(replaceTabs(removed.content), replaceTabs(added.content));
                result.push(theme.fg("toolDiffRemoved", ` + "`-${removed.lineNum} ${removedLine}`" + `));
                result.push(theme.fg("toolDiffAdded", ` + "`+${added.lineNum} ${addedLine}`" + `));
            }`,
		new: `            if (removedLines.length === 1 && addedLines.length === 1) {
                const removed = removedLines[0];
                const added = addedLines[0];
                if (hl) {
                    const removedLine = highlightCode(replaceTabs(removed.content), lang)[0];
                    const addedLine = highlightCode(replaceTabs(added.content), lang)[0];
                    result.push(theme.fg("toolDiffRemoved", ` + "`-${removed.lineNum} ${removedLine}`" + `));
                    result.push(theme.fg("toolDiffAdded", ` + "`+${added.lineNum} ${addedLine}`" + `));
                } else {
                    const { removedLine, addedLine } = renderIntraLineDiff(replaceTabs(removed.content), replaceTabs(added.content));
                    result.push(theme.fg("toolDiffRemoved", ` + "`-${removed.lineNum} ${removedLine}`" + `));
                    result.push(theme.fg("toolDiffAdded", ` + "`+${added.lineNum} ${addedLine}`" + `));
                }
            }`,
		note: "diff syntax highlight single-line edit",
	},
	{
		file: diffComponentRel,
		old: `                for (const removed of removedLines) {
                    result.push(theme.fg("toolDiffRemoved", ` + "`-${removed.lineNum} ${replaceTabs(removed.content)}`" + `));
                }
                for (const added of addedLines) {
                    result.push(theme.fg("toolDiffAdded", ` + "`+${added.lineNum} ${replaceTabs(added.content)}`" + `));
                }`,
		new: `                for (const removed of removedLines) {
                    const content = hl ? highlightCode(replaceTabs(removed.content), lang)[0] : replaceTabs(removed.content);
                    result.push(theme.fg("toolDiffRemoved", ` + "`-${removed.lineNum} ${content}`" + `));
                }
                for (const added of addedLines) {
                    const content = hl ? highlightCode(replaceTabs(added.content), lang)[0] : replaceTabs(added.content);
                    result.push(theme.fg("toolDiffAdded", ` + "`+${added.lineNum} ${content}`" + `));
                }`,
		note: "diff syntax highlight multi-line edit",
	},
	{
		file: diffComponentRel,
		old: `        else if (parsed.prefix === "+") {
            // Standalone added line
            result.push(theme.fg("toolDiffAdded", ` + "`+${parsed.lineNum} ${replaceTabs(parsed.content)}`" + `));
            i++;
        }
        else {
            // Context line
            result.push(theme.fg("toolDiffContext", ` + "` ${parsed.lineNum} ${replaceTabs(parsed.content)}`" + `));
            i++;
        }`,
		new: `        else if (parsed.prefix === "+") {
            // Standalone added line
            const addedContent = hl ? highlightCode(replaceTabs(parsed.content), lang)[0] : replaceTabs(parsed.content);
            result.push(theme.fg("toolDiffAdded", ` + "`+${parsed.lineNum} ${addedContent}`" + `));
            i++;
        }
        else {
            // Context line
            const contextContent = hl ? highlightCode(replaceTabs(parsed.content), lang)[0] : replaceTabs(parsed.content);
            result.push(theme.fg("toolDiffContext", ` + "` ${parsed.lineNum} ${contextContent}`" + `));
            i++;
        }`,
		note: "diff syntax highlight standalone/context lines",
	},
	// --- bash.js: command line syntax highlighting ---
	{
		file: bashComponentRel,
		old:  `import { theme } from "../../modes/interactive/theme/theme.js";`,
		new: `import { theme, highlightCode } from "../../modes/interactive/theme/theme.js";
import { supportsLanguage } from "../../utils/syntax-highlight.js";`,
		note: "bash command highlight imports",
	},
	{
		// Whole-function replacement so the anchor is consumed (idempotent):
		// adds a shell-highlight helper and routes the command through it.
		file: bashComponentRel,
		old: `function formatBashCall(args) {
    const command = str(args?.command);
    const timeout = args?.timeout;
    const timeoutSuffix = timeout ? theme.fg("muted", ` + "` (timeout ${timeout}s)`" + `) : "";
    const commandDisplay = command === null ? invalidArgText(theme) : command ? command : theme.fg("toolOutput", "...");
    return theme.fg("toolTitle", theme.bold(` + "`$ ${commandDisplay}`" + `)) + timeoutSuffix;
}`,
		new: `function highlightShellCommand(command) {
    return supportsLanguage("bash") ? highlightCode(command, "bash").join("\n") : command;
}
function formatBashCall(args) {
    const command = str(args?.command);
    const timeout = args?.timeout;
    const timeoutSuffix = timeout ? theme.fg("muted", ` + "` (timeout ${timeout}s)`" + `) : "";
    const commandDisplay = command === null ? invalidArgText(theme) : command ? highlightShellCommand(command) : theme.fg("toolOutput", "...");
    return theme.fg("toolTitle", theme.bold(` + "`$ ${commandDisplay}`" + `)) + timeoutSuffix;
}`,
		note: "bash command highlight helper + display",
	},
}

// NewPatchPIDiffCmd creates the patch-pi subcommand (alias patch-pi-diff):
// applies rick's manifest of pi TUI patches — edit-diff keyword highlight
// (reverse video -> bold), edit-diff syntax highlighting, and bash command
// syntax highlighting — to rick's self-contained pi runtime only
// (~/.rick/pi/agent/runtime). The user's global/standalone pi install is never
// touched. Idempotent and safe to re-run after a pi runtime upgrade.
func NewPatchPIDiffCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "patch-pi",
		Aliases: []string{"patch-pi-diff"},
		Short:   "Patch rick's pi runtime TUI rendering (diff highlight + diff/command syntax highlight)",
		Long: `Apply rick's pi TUI rendering patches to rick's self-contained pi runtime.

Patches (all inside ~/.rick/pi/agent/runtime, the user's global pi is untouched):
  1. edit-diff intra-line keywords: ANSI reverse video (colored backgrounds) -> bold
  2. edit-diff content: syntax highlighting by file extension (pi's own
     highlight.js pipeline, same as markdown/read-tool rendering)
  3. bash tool command line: shell syntax highlighting

Idempotent: already-applied patches are detected and skipped, so it is safe to
re-run after every pi runtime upgrade (rick tools init-pi also applies it
non-fatally).`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := runPatchPIDiff(cmd.OutOrStdout()); err != nil {
				return err
			}
			return nil
		},
	}
}

// piPackageRoot resolves the pi install's package root — the directory
// containing diffComponentRel. It follows symlinks (e.g. the npm .bin/pi shim
// -> .../pi-coding-agent/dist/cli.js) and walks up the directory tree so it
// works for any install layout pi might be found in.
func piPackageRoot(piPath string) (string, error) {
	resolved, err := filepath.EvalSymlinks(piPath)
	if err != nil {
		return "", fmt.Errorf("resolve pi path %s: %w", piPath, err)
	}
	dir := filepath.Dir(resolved)
	for i := 0; i < 12; i++ {
		if _, err := os.Stat(filepath.Join(dir, diffComponentRel)); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return "", fmt.Errorf("pi install layout not recognized: %s has no %s", piPath, diffComponentRel)
}

// applyPiRuntimePatches applies the manifest to the pi package at root.
// Returns the number of patches applied and the number skipped (already
// applied or upstream layout changed — a no-op, not an error).
func applyPiRuntimePatches(root string) (applied, skipped int, err error) {
	byFile := map[string][]piPatch{}
	var files []string
	for _, p := range piRuntimePatches {
		if _, ok := byFile[p.file]; !ok {
			files = append(files, p.file)
		}
		byFile[p.file] = append(byFile[p.file], p)
	}
	sort.Strings(files)
	for _, file := range files {
		path := filepath.Join(root, file)
		src, err := os.ReadFile(path)
		if err != nil {
			return applied, skipped, fmt.Errorf("read %s: %w", path, err)
		}
		text := string(src)
		changed := false
		for _, p := range byFile[file] {
			if !strings.Contains(text, p.old) {
				skipped++
				continue
			}
			text = strings.ReplaceAll(text, p.old, p.new)
			applied++
			changed = true
		}
		if changed {
			if err := os.WriteFile(path, []byte(text), 0644); err != nil {
				return applied, skipped, fmt.Errorf("write %s: %w", path, err)
			}
		}
	}
	return applied, skipped, nil
}

// runPatchPIDiff locates rick's pi (managed runtime first) and applies the
// patch manifest. It errors only when pi cannot be found or the install layout
// is unrecognized; already-applied patches or upstream layout changes are
// reported without failing (so init-pi can call this non-fatally).
func runPatchPIDiff(w io.Writer) error {
	piPath, err := piagent.FindBinary(nil)
	if err != nil {
		return err
	}
	root, err := piPackageRoot(piPath)
	if err != nil {
		return err
	}
	applied, skipped, err := applyPiRuntimePatches(root)
	if err != nil {
		return err
	}
	if applied > 0 {
		fmt.Fprintf(w, "✅ patched rick pi runtime: %d patch(es) applied, %d skipped: %s\n", applied, skipped, root)
	} else {
		fmt.Fprintf(w, "✅ rick pi runtime already patched (%d patch(es) skipped): %s\n", skipped, root)
	}
	return nil
}
