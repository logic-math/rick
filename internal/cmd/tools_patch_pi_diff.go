package cmd

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/spf13/cobra"

	"github.com/sunquan/rick/internal/agent/piagent"
)

// diffComponentRel is pi's edit-tool diff renderer. Its intra-line word
// highlight uses ANSI reverse video (theme.inverse = chalk.inverse), which
// paints the changed keyword's background in the line color (red for removed,
// green for added) on top of the already red/green line text. rick rewrites it
// to render changed words bold: the line-level red/green contrast stays and
// keywords keep a visible (bold) cue without any colored background.
const diffComponentRel = "dist/modes/interactive/components/diff.js"

// patchPIDiffCall matches pi's `theme.inverse(<arg>)` calls inside
// renderIntraLineDiff and replaces them with `theme.bold(<arg>)`. The regexp
// form survives future argument renames (currently `value`).
var patchPIDiffCall = regexp.MustCompile(`theme\.inverse\(([^()]*)\)`)

// NewPatchPIDiffCmd creates the patch-pi-diff subcommand: replaces pi's
// reverse-video keyword highlight in edit-tool diffs with bold text (changed
// words stay visible; no colored backgrounds, no underline). It targets rick's
// self-contained pi runtime only (~/.rick/pi/agent/runtime) — the user's
// global/standalone pi install is never touched. Idempotent and safe to re-run
// after a pi upgrade.
func NewPatchPIDiffCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "patch-pi-diff",
		Short: "Patch rick's pi runtime diff highlight (keyword reverse-video bg -> bold)",
		Long: `Patch rick's self-contained pi runtime to stop painting keyword backgrounds.

pi's built-in intra-line diff highlight (introduced 0.84.x) marks changed words
inside a single-line edit with ANSI reverse video, which paints the word's
background in the line color (red for removed, green for added). This command
rewrites the diff.js of rick's OWN pi runtime (~/.rick/pi/agent/runtime) so
changed words render bold — the line-level red/green contrast stays, keywords
keep a visible cue, but get no colored background and no underline.

The user's global/standalone pi install is never modified: rick runs its own
isolated pi copy (installed by rick tools init-pi into ~/.rick/pi/agent/runtime)
and only that copy is patched.

Idempotent: already-patched installs are detected and skipped, so it is safe
to re-run after every pi runtime upgrade (rick tools init-pi also applies it
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

// patchPIDiffFile applies the inverse -> plain-text replacement to pi's diff.js
// and reports whether the file was modified. Already-patched installs and
// upstream layout changes (no theme.inverse calls at all) both yield
// changed=false — a no-op, not an error.
func patchPIDiffFile(diffPath string) (changed bool, err error) {
	src, err := os.ReadFile(diffPath)
	if err != nil {
		return false, fmt.Errorf("read %s: %w", diffPath, err)
	}
	text := string(src)
	if !strings.Contains(text, "theme.inverse(") {
		return false, nil
	}
	text = patchPIDiffCall.ReplaceAllString(text, "theme.bold($1)")
	if err := os.WriteFile(diffPath, []byte(text), 0644); err != nil {
		return false, fmt.Errorf("write %s: %w", diffPath, err)
	}
	return true, nil
}

// runPatchPIDiff locates rick's pi (managed runtime first) and patches its diff
// renderer. It errors only when pi cannot be found or the install layout is
// unrecognized; an already-patched install or an upstream layout change is
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
	diffPath := filepath.Join(root, diffComponentRel)
	changed, err := patchPIDiffFile(diffPath)
	if err != nil {
		return err
	}
	if changed {
		fmt.Fprintf(w, "✅ patched pi diff highlight (keyword reverse-video bg -> bold): %s\n", diffPath)
	} else {
		fmt.Fprintf(w, "✅ nothing to patch (already applied or pi layout changed): %s\n", diffPath)
	}
	return nil
}
