package relocate

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

// DefaultDumpPlatforms are the platforms cross-compiled into a dump by
// default (dump --goos overrides).
var DefaultDumpPlatforms = []string{"linux/amd64", "darwin/arm64", "darwin/amd64"}

// DumpOptions configures one `rick tools dump` run. Zero-value fields fall
// back to their documented defaults (resolveDumpDefaults fills them in).
type DumpOptions struct {
	// Workspaces are absolute workspace paths to pack. CLI fills [$PWD].
	Workspaces []string
	// Output is the archive path. Default rick-dump-<ts>.tar.gz in CWD.
	Output string
	// Sessions selects which pi session dirs travel along:
	// SessionsPolicyNone / SessionsPolicyWorkspaces / SessionsPolicyAll.
	Sessions string
	// Slim drops pi-agent runtime/ and npm/ (target re-runs init-pi).
	Slim bool
	// Goos is the platform list ("goos/goarch" each). Default
	// DefaultDumpPlatforms; nil means default, non-nil empty means none.
	Goos []string
	// NoExclude disables the default exclusion rules.
	NoExclude bool
	// StateDir is the source rick state dir. Default: $RICK_STATE_DIR >
	// $HOME/.rick (same ladder as web.ResolveStateDir, mirrored here
	// because relocate cannot import web — import cycle).
	StateDir string
	// AgentDir is the source pi agent dir. Default: $RICK_PI_AGENT_DIR >
	// <StateDir>/pi/agent (when it exists) > $HOME/.rick/pi/agent.
	AgentDir string
	// BuildDir is the rick source tree used to cross-compile embedded
	// binaries. Default: probed (a packed workspace containing cmd/rick >
	// CWD). Empty after probe → embed the running executable as the
	// native-platform binary.
	BuildDir string
	// RickVersion is recorded in the manifest and new marker files.
	RickVersion string
	// Now is overridable in tests for deterministic timestamps.
	Now time.Time
}

// DumpResult reports what one dump produced.
type DumpResult struct {
	ArchivePath string
	SHA256      string
	Files       int
	Bytes       int64 // payload (uncompressed) bytes
	ArchiveSize int64 // on-disk archive size
	Workspaces  []string
	MarkersNew  int      // markers created this run
	Backfilled  []string // source state files rewritten (id → marker)
	Excluded    int      // recorded exclusion hits (rules + slim)
	Binaries    []string // "goos/goarch" of embedded binaries
	Warnings    []string // non-fatal issues (skipped platforms, …)
	AgentPacked bool     // pi agent dir was found and packed
	StatePacked bool     // state dir was found and packed
}

// platform is one parsed "goos/goarch" pair.
type platform struct{ goos, goarch string }

func (p platform) String() string { return p.goos + "/" + p.goarch }

// Dump packs the configured workspaces plus rick's state (config, web
// registries, pi agent dir incl. session history) into one tar.gz with an
// embedded manifest, and writes a <output>.sha256 sidecar.
//
// Source-side effects (both idempotent): every workspace gets a marker
// (.rick-workspace.json) when missing, and the source state's web.json /
// sessions.json / archived.json / job-names.json are upgraded to marker ids
// when they still reference legacy path-hash ids (backfill — doing it at the
// "leaving" moment keeps the source machine consistent and its sessions
// reachable under the marker id).
func Dump(ctx context.Context, opts DumpOptions) (*DumpResult, error) {
	if err := resolveDumpDefaults(&opts); err != nil {
		return nil, err
	}
	res := &DumpResult{Workspaces: opts.Workspaces}

	stateDir, agentDir := opts.StateDir, opts.AgentDir
	res.StatePacked = dirExists(stateDir)
	res.AgentPacked = dirExists(agentDir)
	if !res.StatePacked {
		return nil, fmt.Errorf("state dir %s 不存在（用 --state-dir 显式指定，或设 RICK_STATE_DIR）", stateDir)
	}

	// 1. Markers + manifest workspace list + source-side backfill.
	manifest := &Manifest{
		Format:         DumpFormat,
		Version:        DumpVersion,
		CreatedAt:      opts.Now,
		Origin:         originInfo(),
		StateDirOrigin: stateDir,
		SessionsPolicy: opts.Sessions,
		RickVersion:    opts.RickVersion,
	}
	pm := &PathMap{StateOrigin: stateDir, StateNew: stateDir}
	names := dedupeArchiveNames(opts.Workspaces)
	for i, ws := range opts.Workspaces {
		if _, created, err := EnsureMarker(ws, opts.RickVersion); err != nil {
			return nil, fmt.Errorf("workspace %s: %w", ws, err)
		} else if created {
			res.MarkersNew++
		}
		markerID := ReadMarkerID(ws)
		manifest.Workspaces = append(manifest.Workspaces, WorkspaceInfo{
			Name:       names[i],
			OriginPath: ws,
			MarkerID:   markerID,
			LegacyID:   LegacyWorkspaceID(ws),
		})
		// Backfill mappings are identity-shaped (this machine's paths) with
		// the legacy id of *this* path — RewriteWebJSON switches the id,
		// RewriteSessionRefs repoints sha1(path)→marker references.
		pm.Workspaces = append(pm.Workspaces, WorkspaceMapping{
			Name: names[i], OriginPath: ws, NewPath: ws,
			MarkerID: markerID, LegacyID: LegacyWorkspaceID(ws),
		})
	}
	files, err := backfillWorkspaceIDs(stateDir, pm)
	if err != nil {
		return nil, fmt.Errorf("source backfill: %w", err)
	}
	res.Backfilled = files

	// 2. Embedded binaries (built before the archive: they are items).
	bins, binTmp, warns, err := buildBinaries(ctx, opts)
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(binTmp)
	res.Warnings = append(res.Warnings, warns...)
	for _, b := range bins {
		res.Binaries = append(res.Binaries, b.platform.String())
		manifest.Binaries = append(manifest.Binaries, BinaryInfo{
			GOOS: b.platform.goos, GOARCH: b.platform.goarch, SHA256: b.sha256, Size: b.size,
		})
	}

	// 3. Items + exclusion collection.
	matcher := NewMatcher(DefaultRules(), opts.NoExclude)
	collector := newExclusionCollector()
	tmpOut := opts.Output + ".tmp"
	sidecar := opts.Output + ".sha256"
	// Silent structural exclusions: the dump's own artifacts (never
	// recorded — they are the tool, not the state) and the state↔pi-agent
	// overlap (the agent dir is archived separately as state/pi-agent).
	silent := newSilentExcluder(tmpOut, opts.Output, sidecar)

	items := make([]Item, 0, len(opts.Workspaces)+2)
	for i, ws := range opts.Workspaces {
		items = append(items, Item{
			SourcePath:  ws,
			ArchivePath: "workspaces/" + names[i],
			Match:       collector.matchFunc(RootWorkspace, ws, matcher, silent.forPath()),
		})
	}

	stateMatch := collector.matchFunc(RootState, stateDir, matcher, silent.forPath())
	if res.AgentPacked {
		if rel, err := filepath.Rel(stateDir, agentDir); err == nil && isSubRel(rel) {
			agentRel := filepath.ToSlash(rel)
			inner := stateMatch
			stateMatch = func(rel string) bool {
				if rel == agentRel || strings.HasPrefix(rel, agentRel+"/") {
					return false // packed separately as state/pi-agent
				}
				return inner(rel)
			}
		}
	}
	items = append(items, Item{SourcePath: stateDir, ArchivePath: "state", Match: stateMatch})

	if res.AgentPacked {
		base := collector.matchFunc(RootPIAgent, agentDir, matcher, silent.forPath())
		items = append(items, Item{
			SourcePath:  agentDir,
			ArchivePath: "state/pi-agent",
			Match:       piAgentMatch(base, opts.Sessions, opts.Workspaces, opts.Slim, collector, agentDir),
		})
	}
	for _, b := range bins {
		items = append(items, Item{
			SourcePath:  b.path,
			ArchivePath: "bin/" + b.platform.goos + "-" + b.platform.goarch + "/rick",
		})
	}

	// 4. Prescan exclusions so manifest.json (the archive's first entry)
	// already knows them. WriteArchive re-applies the same match funcs —
	// same inputs, same verdicts; the collector dedups the double walk.
	for _, it := range items {
		if err := prescanItem(it); err != nil {
			return nil, fmt.Errorf("prescan %s: %w", it.SourcePath, err)
		}
	}
	manifest.Excluded = collector.entries()
	res.Excluded = len(manifest.Excluded)

	// 5. Write the archive.
	mf, err := os.CreateTemp("", "rick-dump-manifest-*.json")
	if err != nil {
		return nil, fmt.Errorf("temp manifest: %w", err)
	}
	tmpManifest := mf.Name()
	defer os.Remove(tmpManifest)
	if err := WriteManifest(mf, manifest); err != nil {
		mf.Close()
		return nil, err
	}
	if err := mf.Close(); err != nil {
		return nil, fmt.Errorf("close temp manifest: %w", err)
	}

	f, err := os.OpenFile(tmpOut, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
	if err != nil {
		return nil, fmt.Errorf("create %s: %w", tmpOut, err)
	}
	sha, filesN, bytesN, err := WriteArchive(f,
		append([]Item{{SourcePath: tmpManifest, ArchivePath: "manifest.json"}}, items...), nil)
	if werr := f.Close(); err == nil {
		err = werr
	}
	if err != nil {
		os.Remove(tmpOut)
		return nil, fmt.Errorf("write archive: %w", err)
	}
	if err := os.Rename(tmpOut, opts.Output); err != nil {
		os.Remove(tmpOut)
		return nil, fmt.Errorf("rename to %s: %w", opts.Output, err)
	}
	st, err := os.Stat(opts.Output)
	if err != nil {
		return nil, err
	}
	if err := os.WriteFile(sidecar, []byte(sha+"  "+filepath.Base(opts.Output)+"\n"), 0o644); err != nil {
		return nil, fmt.Errorf("write %s: %w", sidecar, err)
	}

	res.ArchivePath, res.SHA256 = opts.Output, sha
	res.Files, res.Bytes, res.ArchiveSize = filesN, bytesN, st.Size()
	return res, nil
}

// resolveDumpDefaults validates and fills option defaults.
func resolveDumpDefaults(o *DumpOptions) error {
	if len(o.Workspaces) == 0 {
		wd, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("resolve cwd: %w", err)
		}
		o.Workspaces = []string{wd}
	}
	seen := map[string]bool{}
	abs := make([]string, 0, len(o.Workspaces))
	for _, w := range o.Workspaces {
		a, err := filepath.Abs(w)
		if err != nil {
			return fmt.Errorf("resolve workspace %q: %w", w, err)
		}
		if !dirExists(a) {
			return fmt.Errorf("workspace %s 不存在", a)
		}
		if seen[a] {
			continue
		}
		seen[a] = true
		abs = append(abs, a)
	}
	o.Workspaces = abs

	switch o.Sessions {
	case "":
		o.Sessions = SessionsPolicyWorkspaces
	case SessionsPolicyNone, SessionsPolicyWorkspaces, SessionsPolicyAll:
	default:
		return fmt.Errorf("--sessions 取值无效：%q（none|workspaces|all）", o.Sessions)
	}

	if o.Now.IsZero() {
		o.Now = time.Now()
	}
	if o.Output == "" {
		o.Output = filepath.Join(mustWd(), "rick-dump-"+o.Now.UTC().Format("20060102-150405")+".tar.gz")
	} else if a, err := filepath.Abs(o.Output); err == nil {
		o.Output = a
	}

	if o.Goos == nil {
		o.Goos = append([]string{}, DefaultDumpPlatforms...)
	} else {
		for _, g := range o.Goos {
			if _, _, err := parsePlatform(g); err != nil {
				return err
			}
		}
	}

	if o.StateDir == "" {
		o.StateDir = defaultSourceStateDir()
	} else if a, err := filepath.Abs(o.StateDir); err == nil {
		o.StateDir = a
	}
	if o.AgentDir == "" {
		o.AgentDir = defaultSourceAgentDir(o.StateDir)
	} else if a, err := filepath.Abs(o.AgentDir); err == nil {
		o.AgentDir = a
	}
	if o.BuildDir == "" {
		o.BuildDir = probeBuildDir(o.Workspaces)
	}
	return nil
}

// defaultSourceStateDir mirrors web.ResolveStateDir's ladder without
// importing web (cycle): RICK_STATE_DIR > $HOME/.rick.
func defaultSourceStateDir() string {
	if v := strings.TrimSpace(os.Getenv("RICK_STATE_DIR")); v != "" {
		return v
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".rick")
}

// defaultSourceAgentDir resolves the source pi agent dir: RICK_PI_AGENT_DIR
// (the override runtime.AgentDir honors) > <stateDir>/pi/agent when it
// exists > $HOME/.rick/pi/agent. The middle candidate keeps custom
// --state-dir dumps self-contained (their agent dir usually nests inside).
func defaultSourceAgentDir(stateDir string) string {
	if v := strings.TrimSpace(os.Getenv("RICK_PI_AGENT_DIR")); v != "" {
		return v
	}
	if stateDir != "" {
		if nested := filepath.Join(stateDir, "pi", "agent"); dirExists(nested) {
			return nested
		}
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".rick", "pi", "agent")
}

// probeBuildDir finds a rick source tree to build embedded binaries from:
// a packed workspace that is a rick checkout, else CWD, else "".
func probeBuildDir(workspaces []string) string {
	for _, ws := range workspaces {
		if isRickSource(ws) {
			return ws
		}
	}
	if wd := mustWd(); isRickSource(wd) {
		return wd
	}
	return ""
}

func isRickSource(dir string) bool {
	return fileExists(filepath.Join(dir, "go.mod")) && dirExists(filepath.Join(dir, "cmd", "rick"))
}

func mustWd() string {
	wd, err := os.Getwd()
	if err != nil {
		return "."
	}
	return wd
}

func dirExists(p string) bool {
	fi, err := os.Stat(p)
	return err == nil && fi.IsDir()
}

func fileExists(p string) bool {
	fi, err := os.Stat(p)
	return err == nil && !fi.IsDir()
}

func parsePlatform(s string) (string, string, error) {
	goos, goarch, ok := strings.Cut(strings.TrimSpace(s), "/")
	if !ok || goos == "" || goarch == "" {
		return "", "", fmt.Errorf("--goos 平台格式无效：%q（应为 goos/goarch，如 linux/amd64）", s)
	}
	return goos, goarch, nil
}

func originInfo() OriginInfo {
	hostname, _ := os.Hostname()
	u := ""
	if cur, err := user.Current(); err == nil {
		u = cur.Username
	}
	home, _ := os.UserHomeDir()
	return OriginInfo{Hostname: hostname, OS: runtime.GOOS, Arch: runtime.GOARCH, User: u, Home: home}
}

// dedupeArchiveNames assigns each workspace a unique archive directory name
// (basename, then basename-2, …) — two packed workspaces may share a
// basename and archive paths must not collide.
func dedupeArchiveNames(workspaces []string) []string {
	taken := map[string]bool{}
	out := make([]string, 0, len(workspaces))
	for _, ws := range workspaces {
		base := filepath.Base(ws)
		name := base
		for i := 2; taken[name]; i++ {
			name = fmt.Sprintf("%s-%d", base, i)
		}
		taken[name] = true
		out = append(out, name)
	}
	return out
}

// isSubRel reports whether rel (from filepath.Rel) is a strict subpath —
// i.e. the agent dir really nests inside the state dir.
func isSubRel(rel string) bool {
	return rel != "." && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

// ─── exclusion collection ───────────────────────────────────────────────────

// exclusionCollector records rule hits once per (root, source, rel) path —
// the same match funcs run in prescan and in WriteArchive's walk and must
// not double-count.
type exclusionCollector struct {
	seen  map[string]ExcludedEntry
	order []ExcludedEntry
}

func newExclusionCollector() *exclusionCollector {
	return &exclusionCollector{seen: map[string]ExcludedEntry{}}
}

func (c *exclusionCollector) record(root Root, sourcePath, rel string, r Rule) {
	key := string(root) + "|" + sourcePath + "|" + rel
	if _, dup := c.seen[key]; dup {
		return
	}
	e := ExcludedEntry{
		Path:   filepath.Join(sourcePath, filepath.FromSlash(rel)),
		Root:   string(root),
		Kind:   r.Kind,
		Reason: r.Reason,
	}
	c.seen[key] = e
	c.order = append(c.order, e)
}

func (c *exclusionCollector) entries() []ExcludedEntry {
	out := make([]ExcludedEntry, len(c.order))
	copy(out, c.order)
	return out
}

// matchFunc builds the keep-semantics hook for one archive root: silent
// paths (dump outputs) drop first, then rule hits are recorded and dropped,
// everything else is kept.
func (c *exclusionCollector) matchFunc(root Root, sourcePath string, matcher *Matcher, silent func(abs string) bool) func(string) bool {
	return func(rel string) bool {
		if silent != nil && silent(filepath.Join(sourcePath, filepath.FromSlash(rel))) {
			return false
		}
		if rule, hit := matcher.Match(root, rel); hit {
			c.record(root, sourcePath, rel, rule)
			return false
		}
		return true
	}
}

// silentExcluder silently skips the dump's own artifacts when they land
// inside a collected root (the archive streams into <output>.tmp which may
// sit inside a workspace being packed — the growing file must not archive
// itself, and none of these are "left behind" state, so nothing is recorded).
type silentExcluder struct {
	paths map[string]bool
}

func newSilentExcluder(paths ...string) *silentExcluder {
	s := &silentExcluder{paths: map[string]bool{}}
	for _, p := range paths {
		if p != "" {
			s.paths[filepath.Clean(p)] = true
		}
	}
	return s
}

func (s *silentExcluder) forPath() func(abs string) bool {
	if s == nil {
		return nil
	}
	return func(abs string) bool { return s.paths[filepath.Clean(abs)] }
}

// piAgentMatch layers the pi-agent-root concerns over the base rule match:
// the sessions policy filter and slim-mode runtime/npm exclusion.
func piAgentMatch(base func(string) bool, policy string, workspaces []string, slim bool, collector *exclusionCollector, agentDir string) func(string) bool {
	wanted := map[string]bool{}
	for _, ws := range workspaces {
		wanted[EscapeDir(ws)] = true
	}
	slimRule := Rule{Root: RootPIAgent, Kind: KindRegenerable, Reason: "slim 模式：目标机 rick tools init-pi 重装"}
	return func(rel string) bool {
		if rel == "sessions" {
			return policy != SessionsPolicyNone
		}
		if strings.HasPrefix(rel, "sessions/") {
			switch policy {
			case SessionsPolicyNone:
				return false
			case SessionsPolicyWorkspaces:
				dir, _, _ := strings.Cut(strings.TrimPrefix(rel, "sessions/"), "/")
				return wanted[dir]
			}
			// SessionsPolicyAll falls through to the rule matcher.
		}
		if slim {
			if rel == "runtime" || strings.HasPrefix(rel, "runtime/") ||
				rel == "npm" || strings.HasPrefix(rel, "npm/") {
				collector.record(RootPIAgent, agentDir, rel, slimRule)
				return false
			}
		}
		return base(rel)
	}
}

// prescanItem walks one item's source applying its match func to record
// exclusions ahead of the archive write (manifest.json — the archive's
// first entry — must already know them). Mirrors writeItem's walk
// semantics; single-file items carry nothing excludable.
func prescanItem(it Item) error {
	abs, err := filepath.Abs(it.SourcePath)
	if err != nil {
		return err
	}
	fi, err := os.Lstat(abs)
	if err != nil {
		return err
	}
	if !fi.IsDir() {
		return nil
	}
	return filepath.WalkDir(abs, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(abs, p)
		if err != nil {
			return err
		}
		if rel == "." {
			return nil
		}
		if it.Match != nil && !it.Match(filepath.ToSlash(rel)) {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		return nil
	})
}

// ─── embedded binaries ──────────────────────────────────────────────────────

type builtBinary struct {
	platform platform
	path     string
	sha256   string
	size     int64
}

// buildBinaries cross-compiles rick for each requested platform out of
// BuildDir. Contract: a failed build is a warning (platform skipped) except
// for the native platform, which is an error; with no BuildDir at all, the
// running executable is embedded as the native binary (a dump made from an
// installed rick still carries rick). Returns the binaries, the temp dir
// holding them (caller removes), and warnings.
func buildBinaries(ctx context.Context, opts DumpOptions) ([]builtBinary, string, []string, error) {
	if len(opts.Goos) == 0 {
		return nil, "", []string{"未请求任何平台二进制（--goos 为空）"}, nil
	}
	native := platform{runtime.GOOS, runtime.GOARCH}

	if opts.BuildDir == "" {
		self, err := os.Executable()
		if err != nil {
			return nil, "", []string{"找不到 rick 源码树，且无法定位当前可执行文件：归档不内嵌 rick 二进制"}, nil
		}
		sha, size, err := hashFile(self)
		if err != nil {
			return nil, "", nil, fmt.Errorf("hash %s: %w", self, err)
		}
		return []builtBinary{{platform: native, path: self, sha256: sha, size: size}}, "", nil, nil
	}

	tmp, err := os.MkdirTemp("", "rick-dump-bin-*")
	if err != nil {
		return nil, "", nil, fmt.Errorf("temp bin dir: %w", err)
	}
	var out []builtBinary
	var warns []string
	for _, g := range opts.Goos {
		goos, goarch, err := parsePlatform(g)
		if err != nil {
			os.RemoveAll(tmp)
			return nil, "", nil, err
		}
		p := platform{goos, goarch}
		bin := filepath.Join(tmp, p.goos+"-"+p.goarch, "rick")
		if err := os.MkdirAll(filepath.Dir(bin), 0o755); err != nil {
			os.RemoveAll(tmp)
			return nil, "", nil, err
		}
		cmd := exec.CommandContext(ctx, "go", "build", "-o", bin, "./cmd/rick")
		cmd.Dir = opts.BuildDir
		cmd.Env = append(os.Environ(), "GOOS="+p.goos, "GOARCH="+p.goarch, "CGO_ENABLED=0")
		outb, err := cmd.CombinedOutput()
		if err != nil {
			if p == native {
				os.RemoveAll(tmp)
				return nil, "", nil, fmt.Errorf("本机平台 %s 构建失败：%v\n%s", p, err, string(outb))
			}
			warns = append(warns, fmt.Sprintf("平台 %s 构建失败，已跳过：%v", p, err))
			continue
		}
		sha, size, err := hashFile(bin)
		if err != nil {
			os.RemoveAll(tmp)
			return nil, "", nil, fmt.Errorf("hash %s: %w", bin, err)
		}
		out = append(out, builtBinary{platform: p, path: bin, sha256: sha, size: size})
	}
	return out, tmp, warns, nil
}

func hashFile(p string) (string, int64, error) {
	f, err := os.Open(p)
	if err != nil {
		return "", 0, err
	}
	defer f.Close()
	h := sha256.New()
	n, err := io.Copy(h, f)
	if err != nil {
		return "", 0, err
	}
	return hex.EncodeToString(h.Sum(nil)), n, nil
}

// ─── source-side backfill ──────────────────────────────────────────────────

// backfillWorkspaceIDs upgrades a machine's state files to marker ids: for
// every mapping whose workspace is registered in web.json with a different
// id, web.json's id switches to the marker id and every workspace_id
// reference in sessions.json / archived.json / job-names.json follows. Files
// are rewritten only when an id actually changes (idempotent — a second run
// is a no-op). pm's mappings must carry NewPath == the machine's actual
// workspace path (identity on dump, target path on import) and LegacyID ==
// LegacyWorkspaceID(NewPath).
func backfillWorkspaceIDs(stateDir string, pm *PathMap) ([]string, error) {
	webPath := filepath.Join(stateDir, "web.json")
	data, err := os.ReadFile(webPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil // workspace not registered — nothing to backfill
		}
		return nil, fmt.Errorf("read web.json: %w", err)
	}
	root, err := decodeJSONPreserving(data)
	if err != nil {
		return nil, fmt.Errorf("parse web.json: %w", err)
	}
	need := false
	if wsAny, ok := root["workspaces"].([]any); ok {
		for _, e := range wsAny {
			entry, ok := e.(map[string]any)
			if !ok {
				continue
			}
			path, _ := entry["path"].(string)
			for _, w := range pm.Workspaces {
				if path != w.NewPath || w.MarkerID == "" {
					continue
				}
				if id, _ := entry["id"].(string); id != w.MarkerID {
					need = true
				}
			}
		}
	}
	if !need {
		return nil, nil
	}
	var rewritten []string
	perm := filePermOr(webPath, 0o644)
	out, err := RewriteWebJSON(data, pm)
	if err != nil {
		return nil, err
	}
	if err := writeFileAtomic(webPath, out, perm); err != nil {
		return nil, err
	}
	rewritten = append(rewritten, webPath)
	// Reference files follow the id switch (each is optional on disk).
	for _, f := range []struct {
		name    string
		rewrite func([]byte, *PathMap) ([]byte, error)
	}{
		{"web/sessions.json", RewriteSessionRefs},
		{"web/archived.json", RewriteArchivedIDs},
		{"web/job-names.json", RewriteJobNameKeys},
	} {
		p := filepath.Join(stateDir, filepath.FromSlash(f.name))
		d, err := os.ReadFile(p)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return rewritten, fmt.Errorf("read %s: %w", p, err)
		}
		out, err := f.rewrite(d, pm)
		if err != nil {
			return rewritten, fmt.Errorf("rewrite %s: %w", p, err)
		}
		if err := writeFileAtomic(p, out, filePermOr(p, 0o644)); err != nil {
			return rewritten, err
		}
		rewritten = append(rewritten, p)
	}
	return rewritten, nil
}

func filePermOr(p string, def os.FileMode) os.FileMode {
	if fi, err := os.Stat(p); err == nil {
		return fi.Mode().Perm()
	}
	return def
}
