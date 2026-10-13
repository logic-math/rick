package relocate

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

const (
	hostGOOS   = runtime.GOOS
	hostGOARCH = runtime.GOARCH
)

// dumpFixture performs a full dump of the fixture environment and returns
// the archive path.
func dumpFixture(t *testing.T, tmp, ws, state string, mut func(*DumpOptions)) string {
	t.Helper()
	out := filepath.Join(tmp, "dump.tar.gz")
	if _, err := Dump(context.Background(), dumpOpts(ws, state, out, mut)); err != nil {
		t.Fatalf("dump fixture: %v", err)
	}
	return out
}

// ─── import: happy path (cross-path simulation) ────────────────────────────

func TestImportCrossPathFullChain(t *testing.T) {
	tmp := t.TempDir()
	ws := fixtureWS(t, tmp)
	state := fixtureState(t, tmp, ws)
	archive := dumpFixture(t, tmp, ws, state, nil)

	// "Another machine": different root, empty target + state.
	home := t.TempDir()
	target := filepath.Join(home, "new-ws")
	stateDir := filepath.Join(home, "state")
	importOpts := ImportOptions{
		Archive:  archive,
		Target:   target,
		StateDir: stateDir,
		AgentDir: filepath.Join(stateDir, "pi", "agent"),
		Now:      time.Date(2026, 10, 13, 12, 0, 0, 0, time.UTC),
	}
	res, err := Import(importOpts)
	if err != nil {
		t.Fatalf("import: %v (checks: %+v)", err, res.Checks)
	}

	// workspace landed directly at target (single ws, empty target)
	if !fileExists(filepath.Join(target, "README.md")) {
		t.Error("workspace not placed at target")
	}
	// content, modes, symlinks survived
	if b, _ := os.ReadFile(filepath.Join(target, "README.md")); string(b) != "# proj\n" {
		t.Errorf("README content wrong: %q", b)
	}
	if fi, err := os.Lstat(filepath.Join(target, "scripts", "run.sh")); err != nil || fi.Mode().Perm() != 0o755 {
		t.Errorf("run.sh mode lost: %v %v", err, fi)
	}
	if fi, err := os.Lstat(filepath.Join(target, "scripts", "link")); err != nil || fi.Mode()&os.ModeSymlink == 0 {
		t.Errorf("symlink lost: %v", err)
	}
	// excluded garbage did not travel
	if fileExists(filepath.Join(target, "web", "node_modules", "x", "index.js")) {
		t.Error("node_modules traveled")
	}
	if fileExists(filepath.Join(target, "bin", "rick")) {
		t.Error("workspace bin/ traveled")
	}

	marker := ReadMarkerID(ws) // marker identical across machines
	if got := ReadMarkerID(target); got != marker {
		t.Errorf("marker id at target = %q, want %q", got, marker)
	}

	// web.json remapped: path → target, id → marker (stable!)
	web := readJSON(t, filepath.Join(stateDir, "web.json"))
	entry := web["workspaces"].([]any)[0].(map[string]any)
	if entry["path"] != target {
		t.Errorf("web.json path = %v, want %s", entry["path"], target)
	}
	if entry["id"] != marker {
		t.Errorf("web.json id = %v, want marker %s", entry["id"], marker)
	}

	// sessions.json: workspace_id → marker, _prompt_file → target path
	sess := readJSON(t, filepath.Join(stateDir, "web", "sessions.json"))
	s0 := sess["sessions"].([]any)[0].(map[string]any)
	if s0["workspace_id"] != marker {
		t.Errorf("sessions workspace_id = %v", s0["workspace_id"])
	}
	params := s0["params"].(map[string]any)
	if params["_prompt_file"] != filepath.Join(target, ".rick/jobs/job_1/plan/prompt.md") {
		t.Errorf("_prompt_file = %v", params["_prompt_file"])
	}

	// archived / job-names keys remapped
	arch := readJSON(t, filepath.Join(stateDir, "web", "archived.json"))
	if _, ok := arch["archived"].(map[string]any)[marker]; !ok {
		t.Errorf("archived key not remapped: %v", arch["archived"])
	}
	jn := readJSON(t, filepath.Join(stateDir, "web", "job-names.json"))
	if _, ok := jn["names"].(map[string]any)[marker+"/job_9"]; !ok {
		t.Errorf("job-names key not remapped: %v", jn["names"])
	}

	// config: default_workspace remapped to new state dir
	cfg := readJSON(t, filepath.Join(stateDir, "config.json"))
	if cfg["default_workspace"] != stateDir {
		t.Errorf("default_workspace = %v, want %s", cfg["default_workspace"], stateDir)
	}
	if cfg["web_token"] != "tok123" {
		t.Errorf("web_token lost: %v", cfg["web_token"])
	}

	// start-web.sh cd line remapped
	sw, _ := os.ReadFile(filepath.Join(stateDir, "start-web.sh"))
	if !strings.Contains(string(sw), "cd "+target) {
		t.Errorf("start-web.sh cd not remapped:\n%s", sw)
	}

	// pi sessions: dir re-escaped to target cwd, header cwd rewritten
	sessDir := filepath.Join(stateDir, "pi", "agent", "sessions", EscapeDir(target))
	if !dirExists(sessDir) {
		t.Fatalf("session dir not re-escaped: %s", sessDir)
	}
	hdr := firstLine(t, filepath.Join(sessDir, "2026-01-01T00-00-00-000Z_pi-0001.jsonl"))
	if !strings.Contains(hdr, `"cwd":"`+target+`"`) {
		t.Errorf("session header cwd not rewritten: %s", hdr)
	}
	// second line untouched
	lines := readLines(t, filepath.Join(sessDir, "2026-01-01T00-00-00-000Z_pi-0001.jsonl"))
	if len(lines) < 2 || !strings.Contains(lines[1], "hello") {
		t.Errorf("session payload lines lost: %v", lines)
	}
	// origin-escaped dir must NOT exist at target
	if dirExists(filepath.Join(stateDir, "pi", "agent", "sessions", EscapeDir(ws))) {
		t.Error("origin-escaped session dir still present")
	}

	// runtime/npm traveled (non-slim) and pid/log junk did not
	if !fileExists(filepath.Join(stateDir, "pi", "agent", "runtime", "node_modules", ".bin", "pi")) {
		t.Error("pi runtime not placed")
	}
	if fileExists(filepath.Join(stateDir, "web.pid")) {
		t.Error("web.pid placed")
	}
	if !res.SHA256Verified {
		t.Error("sha256 sidecar not verified")
	}
}

// ─── placement: multi workspace / non-empty target ─────────────────────────

func TestImportPlacementRules(t *testing.T) {
	tmp := t.TempDir()
	// two workspaces sharing even a basename to exercise name dedup
	wsA := fixtureWS(t, tmp)                   // .../proj
	wsB := filepath.Join(tmp, "other", "proj") // same basename
	mustMk(t, wsB)
	write(t, filepath.Join(wsB, "hello.txt"), "B\n")
	state := fixtureState(t, tmp, wsA)

	archive := filepath.Join(tmp, "multi.tar.gz")
	if _, err := Dump(context.Background(), DumpOptions{
		Workspaces: []string{wsA, wsB},
		Output:     archive,
		StateDir:   state,
		AgentDir:   filepath.Join(state, "pi", "agent"),
		Goos:       []string{},
		Now:        time.Date(2026, 10, 13, 12, 0, 0, 0, time.UTC),
	}); err != nil {
		t.Fatalf("dump multi: %v", err)
	}
	m := readManifest(t, archive)
	if len(m.Workspaces) != 2 || m.Workspaces[1].Name != "proj-2" {
		t.Fatalf("name dedup failed: %+v", m.Workspaces)
	}

	// multi → each lands under Target/<name>
	target := t.TempDir()
	res, err := Import(ImportOptions{Archive: archive, Target: target,
		StateDir: filepath.Join(target, "..", "state2"), Now: time.Date(2026, 10, 13, 12, 0, 0, 0, time.UTC)})
	if err != nil {
		t.Fatalf("import multi: %v", err)
	}
	if !fileExists(filepath.Join(target, "proj", "README.md")) {
		t.Error("wsA not at target/proj")
	}
	if !fileExists(filepath.Join(target, "proj-2", "hello.txt")) {
		t.Error("wsB not at target/proj-2")
	}

	// single ws + non-empty unrelated target → Target/<name> subdir
	tmp2 := t.TempDir()
	ws := fixtureWS(t, tmp2)
	state2 := fixtureState(t, tmp2, ws)
	arch2 := dumpFixture(t, tmp2, ws, state2, nil)
	target2 := t.TempDir()
	write(t, filepath.Join(target2, "unrelated.txt"), "keep me\n")
	res, err = Import(ImportOptions{Archive: arch2, Target: target2,
		StateDir: filepath.Join(target2, "..", "state3"), Now: time.Date(2026, 10, 13, 12, 0, 0, 0, time.UTC)})
	if err != nil {
		t.Fatalf("import into non-empty target: %v", err)
	}
	if !fileExists(filepath.Join(target2, "proj", "README.md")) {
		t.Error("single ws into non-empty target should land at target/proj")
	}
	if !fileExists(filepath.Join(target2, "unrelated.txt")) {
		t.Error("unrelated target content destroyed")
	}
	if len(res.Workspaces) != 1 || res.Workspaces[0].DestPath != filepath.Join(target2, "proj") {
		t.Errorf("dest wrong: %+v", res.Workspaces)
	}
}

// ─── merge semantics ────────────────────────────────────────────────────────

func TestImportMergeKeepsExisting(t *testing.T) {
	tmp := t.TempDir()
	ws := fixtureWS(t, tmp)
	state := fixtureState(t, tmp, ws)
	archive := dumpFixture(t, tmp, ws, state, nil)

	target := filepath.Join(tmp, "t1")
	stateDir := filepath.Join(tmp, "s1")
	if _, err := Import(ImportOptions{Archive: archive, Target: target, StateDir: stateDir,
		AgentDir: filepath.Join(stateDir, "pi", "agent"), Now: time.Date(2026, 10, 13, 12, 0, 0, 0, time.UTC)}); err != nil {
		t.Fatalf("first import: %v", err)
	}

	// target-side modifications that must survive a re-import
	write(t, filepath.Join(target, "README.md"), "# CHANGED LOCALLY\n")
	write(t, filepath.Join(target, "new-local.txt"), "local\n")

	res, err := Import(ImportOptions{Archive: archive, Target: target, StateDir: stateDir,
		AgentDir: filepath.Join(stateDir, "pi", "agent"), Now: time.Date(2026, 10, 13, 12, 0, 0, 0, time.UTC)})
	if err != nil {
		t.Fatalf("re-import: %v", err)
	}
	if b, _ := os.ReadFile(filepath.Join(target, "README.md")); string(b) != "# CHANGED LOCALLY\n" {
		t.Error("existing file overwritten by merge")
	}
	if !fileExists(filepath.Join(target, "new-local.txt")) {
		t.Error("local-only file removed")
	}
	if res.Workspaces[0].FilesSkipped == 0 {
		t.Errorf("re-import should skip existing files (skipped=%d)", res.Workspaces[0].FilesSkipped)
	}
	if res.Workspaces[0].FilesPlaced != 0 {
		t.Errorf("re-import placed %d files (want 0 — everything exists)", res.Workspaces[0].FilesPlaced)
	}

	// registry merges: appending to an existing (different) state
	other := t.TempDir()
	ws2 := fixtureWS(t, other)
	state2 := fixtureState(t, other, ws2)
	arch2 := dumpFixture(t, other, ws2, state2, nil)
	res, err = Import(ImportOptions{Archive: arch2, Target: t.TempDir(), StateDir: stateDir,
		AgentDir: filepath.Join(stateDir, "pi", "agent"), Now: time.Date(2026, 10, 13, 12, 0, 0, 0, time.UTC)})
	if err != nil {
		t.Fatalf("import into occupied state: %v", err)
	}
	web := readJSON(t, filepath.Join(stateDir, "web.json"))
	list := web["workspaces"].([]any)
	if len(list) != 2 {
		t.Errorf("web.json should hold both workspaces, has %d", len(list))
	}
	sess := readJSON(t, filepath.Join(stateDir, "web", "sessions.json"))
	if got := len(sess["sessions"].([]any)); got != 1 {
		t.Errorf("sessions merged to %d (same session id sess-1 in both fixtures — dedupe should keep 1)", got)
	}
	if res.MergedWebEntries != 1 {
		t.Errorf("MergedWebEntries = %d, want 1", res.MergedWebEntries)
	}
}

func TestImportForceBacksUpAndReplaces(t *testing.T) {
	tmp := t.TempDir()
	ws := fixtureWS(t, tmp)
	state := fixtureState(t, tmp, ws)
	archive := dumpFixture(t, tmp, ws, state, nil)

	target := filepath.Join(tmp, "t")
	stateDir := filepath.Join(tmp, "s")
	now := time.Date(2026, 10, 13, 12, 0, 0, 0, time.UTC)
	if _, err := Import(ImportOptions{Archive: archive, Target: target, StateDir: stateDir,
		AgentDir: filepath.Join(stateDir, "pi", "agent"), Now: now}); err != nil {
		t.Fatalf("first import: %v", err)
	}
	write(t, filepath.Join(target, "README.md"), "# LOCAL EDIT\n")

	res, err := Import(ImportOptions{Archive: archive, Target: target, StateDir: stateDir,
		AgentDir: filepath.Join(stateDir, "pi", "agent"), Force: true, Now: now})
	if err != nil {
		t.Fatalf("force import: %v", err)
	}
	if b, _ := os.ReadFile(filepath.Join(target, "README.md")); string(b) != "# proj\n" {
		t.Error("force should replace the workspace with the archive version")
	}
	wantWS := target + ".bak-import-20261013-120000"
	if !fileExists(filepath.Join(wantWS, "README.md")) {
		t.Errorf("workspace backup missing at %s", wantWS)
	}
	if b, _ := os.ReadFile(filepath.Join(wantWS, "README.md")); string(b) != "# LOCAL EDIT\n" {
		t.Error("backup should hold the pre-force state")
	}
	wantState := stateDir + ".bak-import-20261013-120000"
	if !dirExists(wantState) {
		t.Errorf("state backup missing at %s", wantState)
	}
	if len(res.ForceBackups) == 0 {
		t.Error("ForceBackups empty")
	}
}

// ─── corrupt / foreign archives ─────────────────────────────────────────────

func TestImportRejectsTamperedArchive(t *testing.T) {
	tmp := t.TempDir()
	ws := fixtureWS(t, tmp)
	state := fixtureState(t, tmp, ws)
	archive := dumpFixture(t, tmp, ws, state, nil)

	// flip a byte in the middle of the archive
	data, _ := os.ReadFile(archive)
	data[len(data)/2] ^= 0xFF
	tampered := filepath.Join(tmp, "tampered.tar.gz")
	os.WriteFile(tampered, data, 0o644)

	if _, err := Import(ImportOptions{Archive: tampered, Target: t.TempDir(),
		StateDir: t.TempDir(), Now: time.Date(2026, 10, 13, 12, 0, 0, 0, time.UTC)}); err == nil {
		t.Error("tampered archive accepted (sha256 sidecar should catch it)")
	}

	// foreign archive without manifest
	foreign := filepath.Join(tmp, "foreign.tar.gz")
	os.WriteFile(foreign, []byte{}, 0o644)
	if _, err := Import(ImportOptions{Archive: foreign, Target: t.TempDir(),
		StateDir: t.TempDir(), Now: time.Date(2026, 10, 13, 12, 0, 0, 0, time.UTC)}); err == nil {
		t.Error("foreign archive accepted")
	}
}

// ─── sessions=all orphan dirs ───────────────────────────────────────────────

func TestImportSessionsAllOrphans(t *testing.T) {
	tmp := t.TempDir()
	ws := fixtureWS(t, tmp)
	state := fixtureState(t, tmp, ws)
	archive := dumpFixture(t, tmp, ws, state, func(o *DumpOptions) { o.Sessions = SessionsPolicyAll })

	target := filepath.Join(tmp, "t")
	stateDir := filepath.Join(tmp, "s")
	res, err := Import(ImportOptions{Archive: archive, Target: target, StateDir: stateDir,
		AgentDir: filepath.Join(stateDir, "pi", "agent"), Now: time.Date(2026, 10, 13, 12, 0, 0, 0, time.UTC)})
	if err != nil {
		t.Fatalf("import sessions=all: %v", err)
	}
	// other-cwd session dir placed as-is (orphan), header untouched
	od := filepath.Join(stateDir, "pi", "agent", "sessions", EscapeDir("/nonexistent/other-cwd"))
	if !fileExists(filepath.Join(od, "2026-01-02T00-00-00-000Z_pi-0002.jsonl")) {
		t.Error("orphan session dir not placed")
	}
	hdr := firstLine(t, filepath.Join(od, "2026-01-02T00-00-00-000Z_pi-0002.jsonl"))
	if !strings.Contains(hdr, "/nonexistent/other-cwd") {
		t.Errorf("orphan header should keep origin cwd: %s", hdr)
	}
	if len(res.OrphanSessionDirs) != 1 {
		t.Errorf("OrphanSessionDirs = %v", res.OrphanSessionDirs)
	}
}

// ─── platform adaptation ────────────────────────────────────────────────────

func TestImportAdaptPlatformRemovesWrongRG(t *testing.T) {
	tmp := t.TempDir()
	ws := fixtureWS(t, tmp)
	state := fixtureState(t, tmp, ws)
	// plant a foreign-format rg into the archive by hand: dump with
	// --no-exclude keeps the fixture's ELF-looking rg
	archive := dumpFixture(t, tmp, ws, state, func(o *DumpOptions) { o.NoExclude = true })

	target := filepath.Join(tmp, "t")
	stateDir := filepath.Join(tmp, "s")
	rg := filepath.Join(stateDir, "pi", "agent", "bin", "rg")
	_, err := Import(ImportOptions{Archive: archive, Target: target, StateDir: stateDir,
		AgentDir: filepath.Join(stateDir, "pi", "agent"), Now: time.Date(2026, 10, 13, 12, 0, 0, 0, time.UTC)})
	if err != nil {
		t.Fatalf("import: %v", err)
	}
	// The fixture rg holds ELF bytes: native on a linux host (kept — pi
	// can exec it), wrong-platform on darwin (must be removed).
	elfIsWrong := wrongPlatformBinary([]byte{0x7f, 'E', 'L', 'F'})
	if _, err := os.Lstat(rg); err == nil && elfIsWrong {
		t.Error("wrong-platform rg survived import")
	} else if os.IsNotExist(err) && !elfIsWrong {
		t.Error("native rg removed by mistake")
	}
}

// ─── install hint for embedded binary ──────────────────────────────────────

func TestImportInstallHint(t *testing.T) {
	tmp := t.TempDir()
	ws := fixtureWS(t, tmp)
	state := fixtureState(t, tmp, ws)
	src := helloModule(t, tmp)
	archive := dumpFixture(t, tmp, ws, state, func(o *DumpOptions) {
		o.BuildDir = src
		o.Goos = []string{hostGOOS + "/" + hostGOARCH}
	})

	target := filepath.Join(tmp, "t")
	stateDir := filepath.Join(tmp, "s")
	res, err := Import(ImportOptions{Archive: archive, Target: target, StateDir: stateDir,
		AgentDir: filepath.Join(stateDir, "pi", "agent"), Now: time.Date(2026, 10, 13, 12, 0, 0, 0, time.UTC)})
	if err != nil {
		t.Fatalf("import: %v", err)
	}
	want := filepath.Join(stateDir, "bin", "rick-import-"+hostGOOS+"-"+hostGOARCH)
	if !fileExists(want) {
		t.Errorf("native binary not staged at %s", want)
	}
	if !strings.Contains(res.InstallHint, want) {
		t.Errorf("install hint = %q", res.InstallHint)
	}
}

// ─── helpers ────────────────────────────────────────────────────────────────

func firstLine(t *testing.T, path string) string {
	t.Helper()
	lines := readLines(t, path)
	if len(lines) == 0 {
		t.Fatalf("%s is empty", path)
	}
	return lines[0]
}

func readLines(t *testing.T, path string) []string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	s := strings.TrimRight(string(data), "\n")
	if s == "" {
		return nil
	}
	return strings.Split(s, "\n")
}
