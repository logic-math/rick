package relocate

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

// ImportOptions configures one `rick tools import <archive>` run.
type ImportOptions struct {
	// Archive is the dump archive path (rick-dump-*.tar.gz).
	Archive string
	// Target is the workspace landing directory. Default $PWD.
	Target string
	// StateDir is the target rick state dir. Default: --state-dir >
	// $RICK_STATE_DIR > $HOME/.rick (ladder mirrored from
	// web.ResolveStateDir — relocate cannot import web).
	StateDir string
	// AgentDir is the target pi agent dir. Default: $RICK_PI_AGENT_DIR >
	// <StateDir>/pi/agent (import treats StateDir as the state tree root:
	// everything the archive carries under state/ lands under it).
	AgentDir string
	// Force replaces conflicting destinations (workspace dirs / the whole
	// state dir) after backing them up as <path>.bak-import-<ts>.
	Force bool
	// Now is overridable in tests for deterministic backup suffixes.
	Now time.Time
}

// ImportedWorkspace reports one workspace's import outcome.
type ImportedWorkspace struct {
	Name         string
	OriginPath   string
	DestPath     string
	MarkerID     string
	FilesPlaced  int
	FilesSkipped int
}

// CheckResult is one post-import verification outcome.
type CheckResult struct {
	Name   string
	OK     bool
	Detail string
}

// ImportResult reports what one import did.
type ImportResult struct {
	Archive             string
	SHA256Verified      bool   // sidecar present and matched
	SHA256SkippedReason string // set when no sidecar was found
	Workspaces          []ImportedWorkspace
	StateFilesPlaced    int
	StateFilesSkipped   int
	MergedWebEntries    int // archive workspace entries appended to existing web.json
	MergedSessions      int // archive session entries appended to existing sessions.json
	SessionsPlaced      int // pi session files placed (headers rewritten)
	SessionsSkipped     int
	OrphanSessionDirs   []string // session dirs of cwds not among the imported workspaces (placed as-is)
	ForceBackups        []string
	Checks              []CheckResult
	InstallHint         string // how to install the embedded native rick binary
	NextSteps           []string

	nowStamp string // backup suffix timestamp (one per import run)
}

// Import unpacks a dump archive, merges it into the target machine's
// directories, remaps every path-bound state file, ports the pi session
// history (directory re-escape + session header cwd rewrite), adapts
// platform-bound leftovers, and verifies the result.
//
// Merge semantics (default): existing files are never overwritten — pi
// session files dedupe by filename, sessions.json/web.json entries by id /
// path, and the target's config.json / start-web.sh win when present. With
// Force, conflicting destinations are backed up (<path>.bak-import-<ts>)
// and then replaced by the archive's version.
func Import(opts ImportOptions) (*ImportResult, error) {
	if err := resolveImportDefaults(&opts); err != nil {
		return nil, err
	}
	res := &ImportResult{Archive: opts.Archive, nowStamp: opts.Now.UTC().Format("20060102-150405")}

	// 0. Integrity: verify against the sidecar when present.
	archiveSHA, _, err := hashFile(opts.Archive)
	if err != nil {
		return nil, fmt.Errorf("read archive: %w", err)
	}
	if side, err := os.ReadFile(opts.Archive + ".sha256"); err == nil {
		want := strings.Fields(string(side))
		if len(want) > 0 && want[0] != archiveSHA {
			return nil, fmt.Errorf("归档校验失败：sha256 不匹配（边车 %s ≠ 实际 %s）——文件可能损坏或被改动", want[0], archiveSHA)
		}
		res.SHA256Verified = true
	} else {
		res.SHA256SkippedReason = "未找到 " + filepath.Base(opts.Archive) + ".sha256 边车，跳过完整性校验"
	}

	// 1. Extract everything into staging (failure → clean up, no target
	// pollution).
	staging, err := os.MkdirTemp("", "rick-import-*")
	if err != nil {
		return nil, fmt.Errorf("staging dir: %w", err)
	}
	defer os.RemoveAll(staging)
	af, err := os.Open(opts.Archive)
	if err != nil {
		return nil, err
	}
	err = ExtractArchive(af, staging)
	af.Close()
	if err != nil {
		return nil, fmt.Errorf("extract: %w", err)
	}

	// 2. Manifest.
	mf, err := os.Open(filepath.Join(staging, "manifest.json"))
	if err != nil {
		return nil, fmt.Errorf("归档缺少 manifest.json（不是 rick dump 归档？）：%w", err)
	}
	manifest, err := ReadManifest(mf)
	mf.Close()
	if err != nil {
		return nil, err
	}
	if manifest.Version > DumpVersion {
		return nil, fmt.Errorf("归档版本 v%d 高于本 rick 支持的 v%d（先升级 rick 再 import）", manifest.Version, DumpVersion)
	}

	// 3. Workspace destinations (marker-aware placement keeps re-imports
	// idempotent: a target that already holds this workspace's marker is
	// that workspace's home).
	dests, err := planWorkspaceDestinations(manifest, opts.Target)
	if err != nil {
		return nil, err
	}

	// 4. Remap the archive's state files IN STAGING (paths → target,
	// workspace ids → marker ids) before anything is merged.
	pm := &PathMap{StateOrigin: manifest.StateDirOrigin, StateNew: opts.StateDir}
	for _, d := range dests {
		pm.Workspaces = append(pm.Workspaces, WorkspaceMapping{
			Name: d.Name, OriginPath: d.OriginPath, NewPath: d.DestPath,
			MarkerID: d.MarkerID, LegacyID: d.LegacyID,
		})
	}
	registered, err := remapStagingState(staging, pm)
	if err != nil {
		return nil, err
	}

	// 5. Merge workspaces into their destinations.
	for _, d := range dests {
		iw := ImportedWorkspace{Name: d.Name, OriginPath: d.OriginPath, DestPath: d.DestPath, MarkerID: d.MarkerID}
		src := filepath.Join(staging, "workspaces", d.Name)
		if err := mergeTree(src, d.DestPath, opts.Force, &iw.FilesPlaced, &iw.FilesSkipped, res); err != nil {
			return nil, fmt.Errorf("merge workspace %s: %w", d.Name, err)
		}
		res.Workspaces = append(res.Workspaces, iw)
	}

	// 6. Merge state (default: keep existing; Force: back up the whole
	// state dir and replace). The pi-agent subtree routes to AgentDir.
	if err := mergeState(staging, opts, pm, res); err != nil {
		return nil, err
	}

	// 6b. pi-agent tree (settings/auth/extensions/runtime/npm/themes …)
	// merges into AgentDir; the sessions/ subtree is handled by step 7.
	if err := mergeTreeSkip(filepath.Join(staging, "state", "pi-agent"), opts.AgentDir,
		[]string{"sessions"}, &res.StateFilesPlaced, &res.StateFilesSkipped, res); err != nil {
		return nil, fmt.Errorf("merge pi-agent: %w", err)
	}

	// 7. pi session portability: re-escape session dirs per workspace and
	// rewrite each session file's header cwd.
	if err := portSessions(staging, opts, dests, pm, res); err != nil {
		return nil, fmt.Errorf("port pi sessions: %w", err)
	}

	// 8. Platform adaptation: drop a wrong-platform rg; stage the embedded
	// native rick binary under <StateDir>/bin for the install hint.
	adaptPlatform(opts, manifest, staging, res)

	// 9. Backfill the TARGET's registries to marker ids (covers a target
	// that already had the workspace registered under its legacy id).
	if err := backfillTargetRegistries(opts.StateDir, dests); err != nil {
		return nil, err
	}

	// 10. Verify.
	res.Checks = verifyImport(opts, manifest, dests, registered)
	var failed []string
	for _, c := range res.Checks {
		if !c.OK {
			failed = append(failed, c.Name+": "+c.Detail)
		}
	}
	if len(failed) > 0 {
		return res, fmt.Errorf("导入后验证未通过：\n- %s", strings.Join(failed, "\n- "))
	}

	res.NextSteps = nextSteps(opts, manifest, res)
	return res, nil
}

// workspaceDest is the resolved landing spot of one packed workspace.
type workspaceDest struct {
	Name, OriginPath, DestPath, MarkerID, LegacyID string
}

// planWorkspaceDestinations decides where each packed workspace lands:
// a single workspace goes directly to Target when Target is empty or
// already holds this workspace's marker (idempotent re-import); otherwise
// (multiple workspaces, or a non-empty Target) each lands under
// Target/<name>. A pre-existing Target/<name> holding the same marker is
// merged into; holding a different marker it is still the destination
// (merge keeps existing files; --force replaces after backup).
func planWorkspaceDestinations(m *Manifest, target string) ([]workspaceDest, error) {
	out := make([]workspaceDest, 0, len(m.Workspaces))
	targetEmpty := dirIsEmpty(target)
	for _, w := range m.Workspaces {
		dest := ""
		if len(m.Workspaces) == 1 && (targetEmpty || ReadMarkerID(target) == w.MarkerID) {
			dest = target
		} else {
			dest = filepath.Join(target, w.Name)
		}
		out = append(out, workspaceDest{
			Name: w.Name, OriginPath: w.OriginPath, DestPath: dest,
			MarkerID: w.MarkerID, LegacyID: w.LegacyID,
		})
	}
	return out, nil
}

func dirIsEmpty(p string) bool {
	entries, err := os.ReadDir(p)
	return err != nil || len(entries) == 0
}

// remapStagingState rewrites the archive's path-bound state files inside
// staging: web.json (paths + ids), sessions.json (ids + _prompt_file /
// _method_file), archived.json / job-names.json (id keys), config.json
// (default_workspace), start-web.sh (cd lines). It also returns which
// remapped workspace paths the archive's web.json registered (workspaces
// the origin never registered must not be required to appear in the
// target's registry — verification uses this).
func remapStagingState(staging string, pm *PathMap) (registered map[string]bool, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("remap staging state: %w", err)
		}
	}()
	state := filepath.Join(staging, "state")
	jobs := []struct {
		rel     string
		rewrite func([]byte, *PathMap) ([]byte, error)
	}{
		{"web.json", RewriteWebJSON},
		{"web/sessions.json", RewriteSessionRefs},
		{"web/archived.json", RewriteArchivedIDs},
		{"web/job-names.json", RewriteJobNameKeys},
		{"config.json", RewriteConfigDefaultWorkspace},
	}
	registered = map[string]bool{}
	for _, j := range jobs {
		p := filepath.Join(state, filepath.FromSlash(j.rel))
		data, err := os.ReadFile(p)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, err
		}
		out, err := j.rewrite(data, pm)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", j.rel, err)
		}
		if err := os.WriteFile(p, out, filePermOr(p, 0o644)); err != nil {
			return nil, err
		}
		if j.rel == "web.json" {
			if root, err := decodeJSONPreserving(out); err == nil {
				if list, ok := root["workspaces"].([]any); ok {
					for _, e := range list {
						if entry, ok := e.(map[string]any); ok {
							if path, _ := entry["path"].(string); path != "" {
								registered[path] = true
							}
						}
					}
				}
			}
		}
	}
	// start-web.sh: text, not JSON.
	sp := filepath.Join(state, "start-web.sh")
	if data, err := os.ReadFile(sp); err == nil {
		if out, err := RewriteStartScript(data, pm); err == nil {
			_ = os.WriteFile(sp, out, filePermOr(sp, 0o755))
		}
	}
	return registered, nil
}

// mergeState merges staging/state into the target StateDir. Default: file
// level merge keeping existing files, with content-level merging for the
// registry JSONs (web.json / sessions.json / archived.json /
// job-names.json). Force: back up the whole StateDir, then place the
// archive's version.
func mergeState(staging string, opts ImportOptions, pm *PathMap, res *ImportResult) error {
	src := filepath.Join(staging, "state")
	if !dirExists(src) {
		return nil
	}
	dst := opts.StateDir

	if opts.Force && !dirIsEmpty(dst) {
		backup := dst + ".bak-import-" + opts.Now.UTC().Format("20060102-150405")
		if err := os.Rename(dst, backup); err != nil {
			return fmt.Errorf("备份状态目录 %s → %s: %w", dst, backup, err)
		}
		res.ForceBackups = append(res.ForceBackups, backup)
	}
	if err := os.MkdirAll(dst, 0o755); err != nil {
		return err
	}

	// Content-merged registries: if the target already has one, merge the
	// (already remapped) archive version into it rather than skipping.
	merged := map[string]func(dstData, arcData []byte) ([]byte, int, error){
		"web.json":           mergeWebJSON,
		"web/sessions.json":  mergeSessionsJSON,
		"web/archived.json":  mergeArchivedJSON,
		"web/job-names.json": mergeJobNamesJSON,
	}
	for rel, fn := range merged {
		arcPath := filepath.Join(src, filepath.FromSlash(rel))
		dstPath := filepath.Join(dst, filepath.FromSlash(rel))
		arcData, err := os.ReadFile(arcPath)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return err
		}
		dstData, err := os.ReadFile(dstPath)
		if os.IsNotExist(err) {
			continue // not present on target: plain file merge below places it
		}
		if err != nil {
			return err
		}
		out, added, err := fn(dstData, arcData)
		if err != nil {
			return fmt.Errorf("merge %s: %w", rel, err)
		}
		if err := writeFileAtomic(dstPath, out, filePermOr(dstPath, 0o644)); err != nil {
			return err
		}
		switch rel {
		case "web.json":
			res.MergedWebEntries += added
		case "web/sessions.json":
			res.MergedSessions += added
		}
		// Consumed by the content merge — remove from staging so the file
		// merge below does not try to place (and skip) it again.
		if err := os.Remove(arcPath); err != nil {
			return err
		}
	}
	_ = pm // (pm kept for signature symmetry / future merges)

	// The pi-agent subtree routes to AgentDir, not StateDir/pi/agent.
	return mergeTreeSkip(src, dst, []string{"pi-agent"}, &res.StateFilesPlaced, &res.StateFilesSkipped, res)
}

// mergeTree merges src into dst. Existing files are kept (skipped+counted);
// missing ones are placed (dirs, files with modes, symlinks). With force,
// dst is backed up first (dst.bak-import-<ts>) and replaced wholesale.
func mergeTree(src, dst string, force bool, placed, skipped *int, res *ImportResult) error {
	if !dirExists(src) {
		return nil
	}
	if force && !dirIsEmpty(dst) {
		backup := dst + ".bak-import-" + res.nowStamp
		if err := os.Rename(dst, backup); err != nil {
			return fmt.Errorf("备份 %s → %s: %w", dst, backup, err)
		}
		res.ForceBackups = append(res.ForceBackups, backup)
	}
	return mergeTreeSkip(src, dst, nil, placed, skipped, res)
}

// mergeTreeSkip merges src into dst skipping top-level entries named in
// skip. mergeCount tracks placed/skipped file counts.
func mergeTreeSkip(src, dst string, skip []string, placed, skipped *int, res *ImportResult) error {
	skipSet := map[string]bool{}
	for _, s := range skip {
		skipSet[s] = true
	}
	return filepath.WalkDir(src, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, p)
		if err != nil {
			return err
		}
		if rel == "." {
			return nil
		}
		parts := strings.Split(filepath.ToSlash(rel), "/")
		if len(parts) > 0 && skipSet[parts[0]] {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		target := filepath.Join(dst, filepath.FromSlash(rel))
		fi, err := d.Info()
		if err != nil {
			return err
		}
		existing, statErr := os.Lstat(target)
		switch {
		case fi.IsDir():
			if statErr == nil && !existing.IsDir() {
				*skipped++
				return nil // a file sits where a dir is needed: keep target
			}
			return os.MkdirAll(target, 0o755)
		case fi.Mode()&os.ModeSymlink != 0:
			if statErr == nil {
				*skipped++
				return nil
			}
			link, err := os.Readlink(p)
			if err != nil {
				return err
			}
			return os.Symlink(link, target)
		case fi.Mode().IsRegular():
			if statErr == nil {
				*skipped++
				return nil
			}
			if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
				return err
			}
			if err := copyFile(p, target, fi.Mode().Perm()); err != nil {
				return err
			}
			*placed++
			return nil
		default:
			return nil // sockets/fifos/devices never travel anyway
		}
	})
}

func copyFile(src, dst string, perm os.FileMode) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, perm)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		return err
	}
	return out.Close()
}

// ─── registry content merges ────────────────────────────────────────────────

// mergeWebJSON appends archive workspace entries to dst when their path is
// among the imported destinations and not already registered (dedupe by
// path — target wins). Entries for workspaces the dump did not pack are
// dropped (they would point at paths the target does not have).
func mergeWebJSON(dstData, arcData []byte) ([]byte, int, error) {
	dst, err := decodeJSONPreserving(dstData)
	if err != nil {
		return nil, 0, fmt.Errorf("parse target web.json: %w", err)
	}
	arc, err := decodeJSONPreserving(arcData)
	if err != nil {
		return nil, 0, fmt.Errorf("parse archive web.json: %w", err)
	}
	known := map[string]bool{}
	dstList, _ := dst["workspaces"].([]any)
	for _, e := range dstList {
		if entry, ok := e.(map[string]any); ok {
			if p, _ := entry["path"].(string); p != "" {
				known[p] = true
			}
		}
	}
	added := 0
	arcList, _ := arc["workspaces"].([]any)
	for _, e := range arcList {
		entry, ok := e.(map[string]any)
		if !ok {
			continue
		}
		p, _ := entry["path"].(string)
		if p == "" || known[p] {
			continue
		}
		known[p] = true
		dstList = append(dstList, entry)
		added++
	}
	dst["workspaces"] = dstList
	out, err := encodeJSONIndent(dst)
	return out, added, err
}

// mergeSessionsJSON appends archive session entries whose id is not already
// present in dst (dedupe by id — target wins).
func mergeSessionsJSON(dstData, arcData []byte) ([]byte, int, error) {
	dst, err := decodeJSONPreserving(dstData)
	if err != nil {
		return nil, 0, fmt.Errorf("parse target sessions.json: %w", err)
	}
	arc, err := decodeJSONPreserving(arcData)
	if err != nil {
		return nil, 0, fmt.Errorf("parse archive sessions.json: %w", err)
	}
	known := map[string]bool{}
	dstList, _ := dst["sessions"].([]any)
	for _, e := range dstList {
		if entry, ok := e.(map[string]any); ok {
			if id, _ := entry["id"].(string); id != "" {
				known[id] = true
			}
		}
	}
	added := 0
	arcList, _ := arc["sessions"].([]any)
	for _, e := range arcList {
		entry, ok := e.(map[string]any)
		if !ok {
			continue
		}
		id, _ := entry["id"].(string)
		if id == "" || known[id] {
			continue
		}
		known[id] = true
		dstList = append(dstList, entry)
		added++
	}
	dst["sessions"] = dstList
	out, err := encodeJSONIndent(dst)
	return out, added, err
}

// mergeArchivedJSON merges the per-workspace archive lists (keys not in dst
// are added; existing keys keep the target's list).
func mergeArchivedJSON(dstData, arcData []byte) ([]byte, int, error) {
	return mergeKeyedArrays(dstData, arcData, "archived", "parse archived.json")
}

// mergeJobNamesJSON merges the "<wsID>/<jobID>" name map (target wins per
// key).
func mergeJobNamesJSON(dstData, arcData []byte) ([]byte, int, error) {
	dst, err := decodeJSONPreserving(dstData)
	if err != nil {
		return nil, 0, fmt.Errorf("parse target job-names.json: %w", err)
	}
	arc, err := decodeJSONPreserving(arcData)
	if err != nil {
		return nil, 0, fmt.Errorf("parse archive job-names.json: %w", err)
	}
	dstMap, _ := dst["names"].(map[string]any)
	if dstMap == nil {
		dstMap = map[string]any{}
	}
	added := 0
	if arcMap, ok := arc["names"].(map[string]any); ok {
		for k, v := range arcMap {
			if _, exists := dstMap[k]; exists {
				continue
			}
			dstMap[k] = v
			added++
		}
	}
	dst["names"] = dstMap
	out, err := encodeJSONIndent(dst)
	return out, added, err
}

func mergeKeyedArrays(dstData, arcData []byte, key, what string) ([]byte, int, error) {
	dst, err := decodeJSONPreserving(dstData)
	if err != nil {
		return nil, 0, fmt.Errorf("parse target %s: %w", what, err)
	}
	arc, err := decodeJSONPreserving(arcData)
	if err != nil {
		return nil, 0, fmt.Errorf("parse archive %s: %w", what, err)
	}
	dstMap, _ := dst[key].(map[string]any)
	if dstMap == nil {
		dstMap = map[string]any{}
	}
	added := 0
	if arcMap, ok := arc[key].(map[string]any); ok {
		for k, v := range arcMap {
			if _, exists := dstMap[k]; exists {
				continue
			}
			dstMap[k] = v
			added++
		}
	}
	dst[key] = dstMap
	out, err := encodeJSONIndent(dst)
	return out, added, err
}

// ─── pi session portability ────────────────────────────────────────────────

// portSessions moves the staging pi session history to the target agent
// dir: session dirs of imported workspaces are re-escaped to the target
// workspace path and every session file's header cwd is rewritten (the
// minimal rewrite set validated against pi in research-L2 A4); dirs of
// other cwds (packed via --sessions=all) are placed as-is — their headers
// keep the origin cwd, so resume by --continue is not possible for them,
// and they are reported as orphans.
func portSessions(staging string, opts ImportOptions, dests []workspaceDest, pm *PathMap, res *ImportResult) error {
	srcSessions := filepath.Join(staging, "state", "pi-agent", "sessions")
	if !dirExists(srcSessions) {
		return nil
	}
	dstSessions := filepath.Join(opts.AgentDir, "sessions")
	if err := os.MkdirAll(dstSessions, 0o755); err != nil {
		return err
	}

	// Map escape(origin cwd) → destination cwd.
	rename := map[string]string{}
	for _, d := range dests {
		rename[EscapeDir(d.OriginPath)] = d.DestPath
	}

	entries, err := os.ReadDir(srcSessions)
	if err != nil {
		return err
	}
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		srcDir := filepath.Join(srcSessions, e.Name())
		if newCwd, ok := rename[e.Name()]; ok {
			dstDir := filepath.Join(dstSessions, EscapeDir(newCwd))
			if err := mergeSessionDir(srcDir, dstDir, newCwd, pm, res); err != nil {
				return fmt.Errorf("session dir %s: %w", e.Name(), err)
			}
			continue
		}
		// Orphan cwd (--sessions=all): place as-is.
		dstDir := filepath.Join(dstSessions, e.Name())
		var p, s int
		if err := mergeTreeSkip(srcDir, dstDir, nil, &p, &s, res); err != nil {
			return err
		}
		res.SessionsPlaced += p
		res.SessionsSkipped += s
		res.OrphanSessionDirs = append(res.OrphanSessionDirs, e.Name())
	}
	return nil
}

// sessionFileName matches pi's session file naming: <ISO timestamp>_<uuid>.jsonl
// (subagent artifact dirs and other files inside the session dir are copied
// through untouched).
func sessionFileName(name string) bool {
	if !strings.HasSuffix(name, ".jsonl") {
		return false
	}
	stem := strings.TrimSuffix(name, ".jsonl")
	_, _, ok := strings.Cut(stem, "_")
	return ok && len(stem) > len("2006-01-02T00-00-00-000Z_")
}

// mergeSessionDir merges one session dir to dstDir, rewriting the header
// cwd of true session files (merge semantics: existing files win).
func mergeSessionDir(srcDir, dstDir, newCwd string, pm *PathMap, res *ImportResult) error {
	if err := os.MkdirAll(dstDir, 0o755); err != nil {
		return err
	}
	return filepath.WalkDir(srcDir, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(srcDir, p)
		if err != nil {
			return err
		}
		if rel == "." {
			return nil
		}
		target := filepath.Join(dstDir, rel)
		fi, err := d.Info()
		if err != nil {
			return err
		}
		if _, statErr := os.Lstat(target); statErr == nil {
			res.SessionsSkipped++
			return nil
		}
		switch {
		case fi.IsDir():
			return os.MkdirAll(target, 0o755)
		case fi.Mode().IsRegular():
			if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
				return err
			}
			// Session files at the dir root get their header rewritten;
			// everything else (artifacts) copies verbatim.
			if filepath.Dir(p) == srcDir && sessionFileName(filepath.Base(p)) {
				if err := copyAndRewriteHeader(p, target, newCwd, pm, fi.Mode().Perm()); err != nil {
					return err
				}
			} else if err := copyFile(p, target, fi.Mode().Perm()); err != nil {
				return err
			}
			res.SessionsPlaced++
			return nil
		default:
			return nil
		}
	})
}

func copyAndRewriteHeader(src, dst, newCwd string, pm *PathMap, perm os.FileMode) error {
	if err := copyFile(src, dst, perm); err != nil {
		return err
	}
	return RewriteSessionHeader(dst, newCwd, pm)
}

// ─── platform adaptation ────────────────────────────────────────────────────

// adaptPlatform removes a wrong-platform ripgrep (pi re-downloads the
// correct one) and stages the embedded native rick binary under
// <StateDir>/bin/rick-import-<goos>-<goarch> with an install hint.
func adaptPlatform(opts ImportOptions, manifest *Manifest, staging string, res *ImportResult) {
	rg := filepath.Join(opts.AgentDir, "bin", "rg")
	if data, err := os.ReadFile(rg); err == nil {
		if wrongPlatformBinary(data) {
			_ = os.Remove(rg)
		}
	}
	native := platform{runtime.GOOS, runtime.GOARCH}
	for _, b := range manifest.Binaries {
		if b.GOOS == native.goos && b.GOARCH == native.goarch {
			src := filepath.Join(staging, "bin", b.GOOS+"-"+b.GOARCH, "rick")
			if !fileExists(src) {
				continue
			}
			dstDir := filepath.Join(opts.StateDir, "bin")
			dst := filepath.Join(dstDir, "rick-import-"+b.GOOS+"-"+b.GOARCH)
			if !fileExists(dst) {
				if err := os.MkdirAll(dstDir, 0o755); err == nil {
					if err := copyFile(src, dst, 0o755); err == nil {
						res.InstallHint = fmt.Sprintf("cp %s ~/.local/bin/rick && chmod +x ~/.local/bin/rick", dst)
					}
				}
			} else {
				res.InstallHint = fmt.Sprintf("cp %s ~/.local/bin/rick && chmod +x ~/.local/bin/rick", dst)
			}
			return
		}
	}
}

// wrongPlatformBinary sniffs the executable format of the first bytes and
// compares against what the current platform can run.
func wrongPlatformBinary(data []byte) bool {
	macho := len(data) >= 4 && (data[0] == 0xfe && data[1] == 0xed && (data[2] == 0xfa || data[2] == 0xce) ||
		data[0] == 0xca && data[1] == 0xfe && data[2] == 0xba && data[3] == 0xbe)
	elf := len(data) >= 4 && data[0] == 0x7f && data[1] == 'E' && data[2] == 'L' && data[3] == 'F'
	switch runtime.GOOS {
	case "linux":
		return macho
	case "darwin":
		return elf
	default:
		return macho || elf
	}
}

// ─── target registry backfill ───────────────────────────────────────────────

// backfillTargetRegistries upgrades the TARGET's registries to marker ids
// for the imported workspaces (covers a target state that already knew the
// workspace under a legacy id — e.g. the workspace was registered on the
// target before import at the same path).
func backfillTargetRegistries(stateDir string, dests []workspaceDest) error {
	pm := &PathMap{StateOrigin: stateDir, StateNew: stateDir}
	any := false
	for _, d := range dests {
		if d.MarkerID == "" {
			continue
		}
		pm.Workspaces = append(pm.Workspaces, WorkspaceMapping{
			Name: d.Name, OriginPath: d.DestPath, NewPath: d.DestPath,
			MarkerID: d.MarkerID, LegacyID: LegacyWorkspaceID(d.DestPath),
		})
		any = true
	}
	if !any {
		return nil
	}
	_, err := backfillWorkspaceIDs(stateDir, pm)
	return err
}

// ─── verification ───────────────────────────────────────────────────────────

// verifyImport runs the post-import checklist. Every item lands in the
// result; failures also fail the command (Import aggregates them).
func verifyImport(opts ImportOptions, manifest *Manifest, dests []workspaceDest, registered map[string]bool) []CheckResult {
	var checks []CheckResult
	add := func(name string, ok bool, detail string) {
		checks = append(checks, CheckResult{Name: name, OK: ok, Detail: detail})
	}

	for _, d := range dests {
		name := "workspace " + d.Name
		if !dirExists(d.DestPath) {
			add(name+" 落位", false, "目标目录不存在: "+d.DestPath)
			continue
		}
		add(name+" 落位", true, d.DestPath)
		if d.MarkerID == "" {
			add(name+" marker", true, "归档未携带 marker（legacy id 工作区）")
			continue
		}
		got := ReadMarkerID(d.DestPath)
		add(name+" marker", got == d.MarkerID,
			fmt.Sprintf("marker id %s（期望 %s）", dashIfEmpty(got), d.MarkerID))
	}

	// web.json: every workspace the dump had registered must be
	// registered at its target path with the marker id.
	if data, err := os.ReadFile(filepath.Join(opts.StateDir, "web.json")); err == nil {
		root, err := decodeJSONPreserving(data)
		if err != nil {
			add("web.json 可解析", false, err.Error())
		} else {
			add("web.json 可解析", true, "")
			for _, d := range dests {
				found, id := "", ""
				if list, ok := root["workspaces"].([]any); ok {
					for _, e := range list {
						if entry, ok := e.(map[string]any); ok {
							if p, _ := entry["path"].(string); p == d.DestPath {
								found = p
								id, _ = entry["id"].(string)
								break
							}
						}
					}
				}
				if found == "" {
					if !registered[d.DestPath] {
						add("web.json 注册 "+d.Name, true, "（源机未注册该工作区）")
					} else {
						add("web.json 注册 "+d.Name, false, "未找到 path="+d.DestPath)
					}
					continue
				}
				ok := d.MarkerID == "" || id == d.MarkerID
				add("web.json 注册 "+d.Name, ok,
					fmt.Sprintf("id=%s（期望 %s）", id, dashIfEmpty(d.MarkerID)))
			}
		}
	} else {
		add("web.json 可解析", false, "读取失败: "+err.Error())
	}

	for _, rel := range []string{"web/sessions.json", "config.json"} {
		p := filepath.Join(opts.StateDir, filepath.FromSlash(rel))
		if !fileExists(p) {
			continue
		}
		if data, err := os.ReadFile(p); err == nil {
			_, perr := decodeJSONPreserving(data)
			add(rel+" 可解析", perr == nil, errDetail(perr))
		} else {
			add(rel+" 可解析", false, err.Error())
		}
	}

	// Session dirs exist for workspaces that carried history.
	sessions := filepath.Join(opts.AgentDir, "sessions")
	for _, d := range dests {
		if manifest.SessionsPolicy == SessionsPolicyNone {
			continue
		}
		dir := filepath.Join(sessions, EscapeDir(d.DestPath))
		if dirExists(dir) {
			add("pi 会话目录 "+d.Name, true, dir)
		} else if manifest.SessionsPolicy == SessionsPolicyWorkspaces {
			add("pi 会话目录 "+d.Name, true, "（源机无该工作区会话历史）")
		}
	}
	return checks
}

func errDetail(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

func dashIfEmpty(s string) string {
	if s == "" {
		return "-"
	}
	return s
}

// nextSteps assembles the human guidance printed by the CLI.
func nextSteps(opts ImportOptions, manifest *Manifest, res *ImportResult) []string {
	steps := []string{}
	if res.InstallHint != "" {
		steps = append(steps, "安装 rick："+res.InstallHint)
	}
	steps = append(steps, "启动 web：cd <工作区> && rick web --port 8413（或恢复你的 start-web.sh）")
	hasSlim := false
	for _, e := range manifest.Excluded {
		if strings.Contains(e.Reason, "slim") {
			hasSlim = true
		}
	}
	if hasSlim {
		steps = append(steps, "slim 归档：先跑 rick tools init-pi 重建 pi runtime/扩展")
	}
	if len(res.OrphanSessionDirs) > 0 {
		steps = append(steps, fmt.Sprintf("注意：%d 个会话目录属于未随迁工作区（--sessions=all 带来），其 cwd 已不存在，仅可按 session id 全局检索",
			len(res.OrphanSessionDirs)))
	}
	return steps
}

// resolveImportDefaults validates and fills import option defaults.
func resolveImportDefaults(o *ImportOptions) error {
	if strings.TrimSpace(o.Archive) == "" {
		return fmt.Errorf("import 需要归档路径参数")
	}
	if a, err := filepath.Abs(o.Archive); err == nil {
		o.Archive = a
	}
	if !fileExists(o.Archive) {
		return fmt.Errorf("归档 %s 不存在", o.Archive)
	}
	if o.Target == "" {
		o.Target = mustWd()
	} else if a, err := filepath.Abs(o.Target); err == nil {
		o.Target = a
	}
	if o.StateDir == "" {
		if v := strings.TrimSpace(os.Getenv("RICK_STATE_DIR")); v != "" {
			o.StateDir = v
		} else if home, err := os.UserHomeDir(); err == nil {
			o.StateDir = filepath.Join(home, ".rick")
		}
	} else if a, err := filepath.Abs(o.StateDir); err == nil {
		o.StateDir = a
	}
	// Import treats StateDir as the state tree root: the agent dir lands
	// inside it unless explicitly overridden (keeps a custom --state-dir
	// import fully self-contained — mirrors the dump-side default).
	if o.AgentDir == "" {
		if v := strings.TrimSpace(os.Getenv("RICK_PI_AGENT_DIR")); v != "" {
			o.AgentDir = v
		} else {
			o.AgentDir = filepath.Join(o.StateDir, "pi", "agent")
		}
	} else if a, err := filepath.Abs(o.AgentDir); err == nil {
		o.AgentDir = a
	}
	if o.Now.IsZero() {
		o.Now = time.Now()
	}
	return nil
}
