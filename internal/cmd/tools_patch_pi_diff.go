package cmd

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/sunquan/rick/internal/agent/piagent"
)

// diffComponentRel is pi's edit-tool diff renderer. Its intra-line word
// highlight uses ANSI reverse video (theme.inverse = chalk.inverse), which
// paints the changed keyword's background in the line color (red for removed,
// green for added) on top of the already red/green line text. rick patches it
// to underline so only the line-level red/green contrast remains and keywords
// no longer carry colored backgrounds.
const diffComponentRel = "dist/modes/interactive/components/diff.js"

// patchPIDiffOld/New are the exact replacements applied to pi's diff.js
// (renderIntraLineDiff: `removedLine += theme.inverse(value);` /
// `addedLine += theme.inverse(value);`). theme.underline is the sibling style
// helper (chalk.underline) — visible emphasis without a background block.
const (
	patchPIDiffOld = "theme.inverse("
	patchPIDiffNew = "theme.underline("
)

// NewPatchPIDiffCmd creates the patch-pi-diff subcommand: replaces pi's
// reverse-video keyword highlight in edit-tool diffs with underline (keeps the
// red/green line contrast, drops the colored keyword backgrounds). Idempotent
// and safe to re-run after a pi upgrade.
func NewPatchPIDiffCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "patch-pi-diff",
		Short: "Patch pi TUI diff highlight: keyword reverse-video bg -> underline",
		Long: `Patch pi's edit-tool diff renderer to stop painting keyword backgrounds.

pi's built-in intra-line diff highlight (introduced 0.84.x) marks changed words
inside a single-line edit with ANSI reverse video, which paints the word's
background in the line color (red for removed, green for added). This command
rewrites pi's dist/modes/interactive/components/diff.js so changed words are
underlined instead — the line-level red/green contrast stays, but keywords no
longer get colored backgrounds.

Idempotent: already-patched installs are detected and skipped, so it is safe
to re-run after every pi upgrade (rick tools init-pi also applies it
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
// containing diffComponentRel. It follows symlinks (e.g. ~/.local/bin/pi ->
// .../pi-coding-agent/dist/cli.js) and walks up the directory tree so it works
// for any install layout pi might be found in.
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

// patchPIDiffFile applies the inverse -> underline replacement to pi's diff.js.
// Returns:
//   - patched=true     replacement was written
//   - alreadyDone=true install was already patched (no-op)
//   - otherwise        pi's diff.js no longer contains the old call (layout
//     changed upstream — nothing to patch, not an error)
func patchPIDiffFile(diffPath string) (patched, alreadyDone bool, err error) {
	src, err := os.ReadFile(diffPath)
	if err != nil {
		return false, false, fmt.Errorf("read %s: %w", diffPath, err)
	}
	text := string(src)
	if strings.Contains(text, patchPIDiffNew) && !strings.Contains(text, patchPIDiffOld) {
		return false, true, nil
	}
	if !strings.Contains(text, patchPIDiffOld) {
		return false, false, nil
	}
	text = strings.ReplaceAll(text, patchPIDiffOld, patchPIDiffNew)
	if err := os.WriteFile(diffPath, []byte(text), 0644); err != nil {
		return false, false, fmt.Errorf("write %s: %w", diffPath, err)
	}
	return true, false, nil
}

// runPatchPIDiff locates pi and patches its diff renderer. It errors only when
// pi cannot be found or the install layout is unrecognized; an already-patched
// install or an upstream layout change is reported without failing (so init-pi
// can call this non-fatally).
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
	patched, alreadyDone, err := patchPIDiffFile(diffPath)
	if err != nil {
		return err
	}
	switch {
	case alreadyDone:
		fmt.Fprintf(w, "✅ pi diff highlight already patched (inverse -> underline): %s\n", diffPath)
	case patched:
		fmt.Fprintf(w, "✅ patched pi diff highlight (inverse video -> underline): %s\n", diffPath)
	default:
		fmt.Fprintf(w, "⚠️  no theme.inverse( calls found in %s — pi layout changed; nothing to patch\n", diffPath)
	}
	return nil
}
