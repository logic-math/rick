package env

import (
	"os"
	"path/filepath"

	"github.com/sunquan/rick/internal/config"
)

// rickRepoMarker is the file whose presence identifies a rick source clone.
// Parity: internal/super findSuperRoot walks up for the same marker (the
// source_tree pointer recorded here is marker-validated at read time there).
// Forward-slash form; join with filepath.FromSlash at the use site (const).
const rickRepoMarker = "super/superopt/__main__.py"

// recordSourceTreeIfRepo records the rick clone root as the config
// source_tree pointer when the process runs from inside a rick source clone
// (job_55 方案 2): install.sh copies the binary out of the clone, so a
// copied install's repo-root resolution (internal/super resolveRepoRoot —
// cwd walk + exe walk) never finds the clone; the pointer recorded here is
// its fallback. Runs from the init-pi flow (install.sh and manual runs both
// execute init-pi from the clone), so the pointer is recorded exactly when
// an install has a clone to point at.
//
// Semantics:
//   - cwd inside a clone → config source_tree = the clone ROOT (upward walk;
//     subdirectories record the root, not the subdir)
//   - cwd not inside a clone → config untouched (an existing pointer is
//     never cleared or overwritten by an unrelated init-pi run)
//   - last recording wins (a deliberate re-run from a different clone
//     repoints the pointer)
//
// Non-fatal by design (callers ignore the error): a failed recording only
// degrades super resolution to the pre-job_55 behavior.
func recordSourceTreeIfRepo() error {
	cwd, err := os.Getwd()
	if err != nil {
		return err
	}
	root := rickRepoRootFrom(cwd)
	if root == "" {
		return nil // cwd 不在 rick clone 内：不动既有指针（绝不误清）
	}
	cfg, err := config.LoadConfig()
	if err != nil {
		return err
	}
	cfg.SourceTree = root
	return config.SaveConfig(cfg)
}

// rickRepoRootFrom walks up from dir looking for the rick repo marker and
// returns the directory carrying it ("" when none). Simplified mirror of
// internal/super findSuperRoot's upward walk (no symlink evaluation — the
// pointer is marker-validated at read time by the super side anyway).
func rickRepoRootFrom(dir string) string {
	for {
		if _, err := os.Stat(filepath.Join(dir, filepath.FromSlash(rickRepoMarker))); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "" // filesystem root: not found
		}
		dir = parent
	}
}
