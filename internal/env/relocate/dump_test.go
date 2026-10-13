package relocate

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

// ─── test fixtures ──────────────────────────────────────────────────────────

// fixtureWS builds a fake workspace: real content (README, executable
// script, symlink, nested dirs, marker-relevant .rick tree) plus garbage
// the default rules must exclude (web/node_modules, bin/, .pi/, .morty/).
func fixtureWS(t *testing.T, root string) string {
	t.Helper()
	ws := filepath.Join(root, "proj")
	mustMk(t, filepath.Join(ws, ".rick", "jobs", "job_1", "plan"))
	write(t, filepath.Join(ws, ".rick", "jobs", "job_1", "plan", "task1.md"), "# task1\n")
	write(t, filepath.Join(ws, ".rick", "jobs", "job_1", "plan", "prompt.md"), "PROMPT")
	write(t, filepath.Join(ws, "README.md"), "# proj\n")
	write(t, filepath.Join(ws, "scripts", "run.sh"), "#!/bin/sh\necho hi\n")
	chmod(t, filepath.Join(ws, "scripts", "run.sh"), 0o755)
	symlink(t, "../README.md", filepath.Join(ws, "scripts", "link"))
	// garbage that default rules exclude
	write(t, filepath.Join(ws, "web", "node_modules", "x", "index.js"), "// huge\n")
	write(t, filepath.Join(ws, "bin", "rick"), "\x00ELF-binary")
	write(t, filepath.Join(ws, ".pi", "cache.json"), "{}")
	write(t, filepath.Join(ws, ".morty", "log.txt"), "x")
	return ws
}

// fixtureState builds a fake rick state dir: web.json registering ws under
// its legacy id, sessions.json referencing legacy id + _prompt_file,
// archived.json / job-names.json keyed by legacy id, config.json, and a
// pi agent dir with auth/settings + one session jsonl.
func fixtureState(t *testing.T, root, ws string) string {
	t.Helper()
	state := filepath.Join(root, "state")
	mustMk(t, state, filepath.Join(state, "web"))
	legacy := LegacyWorkspaceID(ws)
	writeJSON(t, filepath.Join(state, "web.json"), map[string]any{
		"version": 1,
		"workspaces": []any{map[string]any{
			"id": legacy, "path": ws, "name": "proj", "added_at": "2026-01-01T00:00:00Z",
		}},
	})
	writeJSON(t, filepath.Join(state, "web", "sessions.json"), map[string]any{
		"version": 1,
		"sessions": []any{map[string]any{
			"id": "sess-1", "workspace_id": legacy, "type": "easy", "status": "closed",
			"pi_session_id": "pi-0001",
			"params":        map[string]any{"_prompt_file": filepath.Join(ws, ".rick/jobs/job_1/plan/prompt.md")},
			"created_at":    "2026-01-01T00:00:00Z",
		}},
	})
	writeJSON(t, filepath.Join(state, "web", "archived.json"), map[string]any{
		"version": 1, "archived": map[string]any{legacy: []any{"job_9"}},
	})
	writeJSON(t, filepath.Join(state, "web", "job-names.json"), map[string]any{
		"version": 1, "names": map[string]any{legacy + "/job_9": "旧名"},
	})
	writeJSON(t, filepath.Join(state, "config.json"), map[string]any{
		"default_workspace": state, "web_token": "tok123",
	})
	write(t, filepath.Join(state, "start-web.sh"), "#!/bin/bash\ncd "+ws+"\nexec ./bin/rick web --port 8413\n")
	write(t, filepath.Join(state, "web.pid"), "12345")
	write(t, filepath.Join(state, "web.log"), "log\n")
	write(t, filepath.Join(state, "config.json.bak.20260101"), "{}\n")

	// pi agent dir
	agent := filepath.Join(state, "pi", "agent")
	mustMk(t, agent, filepath.Join(agent, "sessions", EscapeDir(ws)))
	writeJSON(t, filepath.Join(agent, "settings.json"), map[string]any{"theme": "rick"})
	write(t, filepath.Join(agent, "auth.json"), `{"deepseek":{"key":"sk-test"}}`)
	sessFile := filepath.Join(agent, "sessions", EscapeDir(ws), "2026-01-01T00-00-00-000Z_pi-0001.jsonl")
	write(t, sessFile, `{"type":"session","version":3,"id":"pi-0001","timestamp":"2026-01-01T00:00:00Z","cwd":"`+ws+`"}`+"\n"+
		`{"type":"message","role":"user","content":"hello"}`+"\n")
	// other-cwd session dir (excluded under policy=workspaces, packed under all)
	other := "/nonexistent/other-cwd"
	mustMk(t, filepath.Join(agent, "sessions", EscapeDir(other)))
	write(t, filepath.Join(agent, "sessions", EscapeDir(other), "2026-01-02T00-00-00-000Z_pi-0002.jsonl"),
		`{"type":"session","version":3,"id":"pi-0002","cwd":"`+other+`"}`+"\n")
	// excluded garbage
	write(t, filepath.Join(agent, "bin", "rg"), "\x7fELF")
	symlink(t, "/workdir/nowhere/catpaw.ts", filepath.Join(agent, "extensions", "catpaw-proxy.ts"))
	write(t, filepath.Join(agent, "runtime", "node_modules", ".bin", "pi"), "#!/bin/sh\n")
	write(t, filepath.Join(agent, "npm", "node_modules", "pi-subagents", "index.js"), "//\n")
	return state
}

// helloModule builds a tiny go module used as BuildDir in tests (fast
// builds, no dependency downloads).
func helloModule(t *testing.T, root string) string {
	t.Helper()
	dir := filepath.Join(root, "hello-src")
	mustMk(t, dir, filepath.Join(dir, "cmd", "rick"))
	write(t, filepath.Join(dir, "go.mod"), "module hello\n\ngo 1.25\n")
	write(t, filepath.Join(dir, "cmd", "rick", "main.go"), "package main\n\nfunc main() { println(\"hi\") }\n")
	return dir
}

func dumpOpts(ws, state, out string, mut func(*DumpOptions)) DumpOptions {
	o := DumpOptions{
		Workspaces: []string{ws},
		Output:     out,
		StateDir:   state,
		AgentDir:   filepath.Join(state, "pi", "agent"),
		BuildDir:   "", // set per test
		Goos:       []string{},
		Now:        time.Date(2026, 10, 13, 12, 0, 0, 0, time.UTC),
	}
	if mut != nil {
		mut(&o)
	}
	return o
}

// ─── options defaults ───────────────────────────────────────────────────────

func TestDumpResolveDefaults(t *testing.T) {
	tmp := t.TempDir()
	ws := fixtureWS(t, tmp)
	state := fixtureState(t, tmp, ws)

	o := DumpOptions{Workspaces: []string{ws}, StateDir: state, Output: filepath.Join(tmp, "out.tar.gz")}
	if err := resolveDumpDefaults(&o); err != nil {
		t.Fatalf("resolve defaults: %v", err)
	}
	if o.Sessions != SessionsPolicyWorkspaces {
		t.Errorf("default sessions policy = %q, want workspaces", o.Sessions)
	}
	if len(o.Goos) != 3 || o.Goos[0] != "linux/amd64" {
		t.Errorf("default platforms = %v", o.Goos)
	}
	if o.Output == "" || !strings.HasSuffix(o.Output, ".tar.gz") {
		t.Errorf("default output = %q", o.Output)
	}
	if o.AgentDir != filepath.Join(state, "pi", "agent") {
		t.Errorf("agent dir = %q, want nested-in-state candidate (it exists)", o.AgentDir)
	}

	// invalid sessions policy
	o2 := DumpOptions{Workspaces: []string{ws}, StateDir: state, Sessions: "bogus"}
	if err := resolveDumpDefaults(&o2); err == nil {
		t.Error("bogus sessions policy accepted")
	}
	// invalid platform
	o3 := DumpOptions{Workspaces: []string{ws}, StateDir: state, Goos: []string{"linux"}}
	if err := resolveDumpDefaults(&o3); err == nil {
		t.Error("platform without goarch accepted")
	}
	// missing workspace
	o4 := DumpOptions{Workspaces: []string{filepath.Join(tmp, "nope")}, StateDir: state}
	if err := resolveDumpDefaults(&o4); err == nil {
		t.Error("missing workspace accepted")
	}
	// missing state dir → Dump errors
	o5 := DumpOptions{Workspaces: []string{ws}, StateDir: filepath.Join(tmp, "nostate")}
	if _, err := Dump(context.Background(), o5); err == nil {
		t.Error("missing state dir accepted")
	}
}

// ─── dump orchestration ─────────────────────────────────────────────────────

func TestDumpArchiveStructureAndBackfill(t *testing.T) {
	tmp := t.TempDir()
	ws := fixtureWS(t, tmp)
	state := fixtureState(t, tmp, ws)
	out := filepath.Join(tmp, "out.tar.gz")

	res, err := Dump(context.Background(), dumpOpts(ws, state, out, nil))
	if err != nil {
		t.Fatalf("dump: %v", err)
	}

	// sidecar
	side, err := os.ReadFile(out + ".sha256")
	if err != nil || !strings.HasPrefix(string(side), res.SHA256) {
		t.Fatalf("sidecar missing/mismatched: %v %q", err, string(side))
	}

	// archive contents
	names := archiveNames(t, out)
	mustContain(t, names, "manifest.json")
	mustContain(t, names, "workspaces/proj/README.md")
	mustContain(t, names, "workspaces/proj/scripts/run.sh")
	mustContain(t, names, "workspaces/proj/scripts/link") // symlink kept
	mustContain(t, names, "state/web.json")
	mustContain(t, names, "state/web/sessions.json")
	mustContain(t, names, "state/config.json")
	mustContain(t, names, "state/start-web.sh")
	mustContain(t, names, "state/pi-agent/settings.json")
	mustContain(t, names, "state/pi-agent/auth.json")
	mustContain(t, names, "state/pi-agent/sessions/"+EscapeDir(ws)+"/2026-01-01T00-00-00-000Z_pi-0001.jsonl")
	// exclusions
	for _, banned := range []string{
		"workspaces/proj/web/node_modules", "workspaces/proj/bin/", "workspaces/proj/.pi/",
		"workspaces/proj/.morty/", "state/web.pid", "state/web.log", "state/config.json.bak.",
		"state/pi-agent/bin/rg", "state/pi-agent/extensions/catpaw-proxy.ts",
	} {
		for _, n := range names {
			if strings.HasPrefix(n, banned) {
				t.Errorf("excluded path leaked into archive: %s", n)
			}
		}
	}
	// sessions policy=workspaces: other-cwd dir absent
	for _, n := range names {
		if strings.HasPrefix(n, "state/pi-agent/sessions/"+EscapeDir("/nonexistent/other-cwd")) {
			t.Errorf("other-cwd session packed under policy=workspaces: %s", n)
		}
	}
	// runtime/npm present (not slim)
	mustContain(t, names, "state/pi-agent/runtime/node_modules/.bin/pi")

	// manifest
	m := readManifest(t, out)
	if m.Format != DumpFormat || m.Version != DumpVersion {
		t.Errorf("manifest identity: %+v", m)
	}
	if len(m.Workspaces) != 1 || m.Workspaces[0].Name != "proj" {
		t.Fatalf("manifest workspaces: %+v", m.Workspaces)
	}
	marker := ReadMarkerID(ws)
	if marker == "" {
		t.Fatal("marker not created in source workspace")
	}
	if m.Workspaces[0].MarkerID != marker {
		t.Errorf("manifest marker id %q != on-disk %q", m.Workspaces[0].MarkerID, marker)
	}
	if m.Workspaces[0].LegacyID != LegacyWorkspaceID(ws) {
		t.Errorf("manifest legacy id wrong")
	}
	if len(m.Excluded) == 0 {
		t.Error("excluded list empty — node_modules etc. should be recorded")
	}
	if m.SessionsPolicy != SessionsPolicyWorkspaces {
		t.Errorf("sessions policy not recorded")
	}

	// source-side backfill: registries upgraded to marker id
	webData := readJSON(t, filepath.Join(state, "web.json"))
	wsList := webData["workspaces"].([]any)
	entry := wsList[0].(map[string]any)
	if entry["id"] != marker {
		t.Errorf("web.json id not backfilled: %v != %s", entry["id"], marker)
	}
	sess := readJSON(t, filepath.Join(state, "web", "sessions.json"))
	s0 := sess["sessions"].([]any)[0].(map[string]any)
	if s0["workspace_id"] != marker {
		t.Errorf("sessions.json workspace_id not backfilled")
	}
	arch := readJSON(t, filepath.Join(state, "web", "archived.json"))
	if _, ok := arch["archived"].(map[string]any)[marker]; !ok {
		t.Errorf("archived.json key not backfilled: %v", arch["archived"])
	}
	jn := readJSON(t, filepath.Join(state, "web", "job-names.json"))
	if _, ok := jn["names"].(map[string]any)[marker+"/job_9"]; !ok {
		t.Errorf("job-names.json key not backfilled: %v", jn["names"])
	}

	// idempotency: second dump — no marker churn, stable ids, same exclude count
	res2, err := Dump(context.Background(), dumpOpts(ws, state, out, nil))
	if err != nil {
		t.Fatalf("second dump: %v", err)
	}
	if res2.MarkersNew != 0 {
		t.Errorf("second dump created %d markers (want 0)", res2.MarkersNew)
	}
	if ReadMarkerID(ws) != marker {
		t.Errorf("marker id changed on second dump")
	}
	if res2.Excluded != res.Excluded {
		t.Errorf("excluded count changed between dumps: %d vs %d", res.Excluded, res2.Excluded)
	}
	if len(res2.Backfilled) != 0 {
		t.Errorf("second dump backfilled again: %v", res2.Backfilled)
	}
	entry2 := readJSON(t, filepath.Join(state, "web.json"))["workspaces"].([]any)[0].(map[string]any)
	if entry2["id"] != marker {
		t.Errorf("web.json id drifted after second dump")
	}
}

func TestDumpSessionsPolicyNoneAndAll(t *testing.T) {
	tmp := t.TempDir()
	ws := fixtureWS(t, tmp)
	state := fixtureState(t, tmp, ws)
	other := EscapeDir("/nonexistent/other-cwd")

	out := filepath.Join(tmp, "none.tar.gz")
	if _, err := Dump(context.Background(), dumpOpts(ws, state, out, func(o *DumpOptions) { o.Sessions = SessionsPolicyNone })); err != nil {
		t.Fatalf("dump none: %v", err)
	}
	for _, n := range archiveNames(t, out) {
		if strings.HasPrefix(n, "state/pi-agent/sessions/") {
			t.Errorf("policy=none packed session: %s", n)
		}
	}

	out = filepath.Join(tmp, "all.tar.gz")
	if _, err := Dump(context.Background(), dumpOpts(ws, state, out, func(o *DumpOptions) { o.Sessions = SessionsPolicyAll })); err != nil {
		t.Fatalf("dump all: %v", err)
	}
	mustContain(t, archiveNames(t, out), "state/pi-agent/sessions/"+other+"/2026-01-02T00-00-00-000Z_pi-0002.jsonl")
}

func TestDumpSlim(t *testing.T) {
	tmp := t.TempDir()
	ws := fixtureWS(t, tmp)
	state := fixtureState(t, tmp, ws)
	out := filepath.Join(tmp, "slim.tar.gz")
	res, err := Dump(context.Background(), dumpOpts(ws, state, out, func(o *DumpOptions) { o.Slim = true }))
	if err != nil {
		t.Fatalf("dump slim: %v", err)
	}
	_ = res
	for _, n := range archiveNames(t, out) {
		if strings.HasPrefix(n, "state/pi-agent/runtime/") || strings.HasPrefix(n, "state/pi-agent/npm/") {
			t.Errorf("slim packed runtime/npm: %s", n)
		}
	}
	m := readManifest(t, out)
	slimmed := false
	for _, e := range m.Excluded {
		if strings.Contains(e.Reason, "slim") {
			slimmed = true
		}
	}
	if !slimmed {
		t.Error("slim exclusions not recorded in manifest")
	}
}

func TestDumpNoExclude(t *testing.T) {
	tmp := t.TempDir()
	ws := fixtureWS(t, tmp)
	state := fixtureState(t, tmp, ws)
	out := filepath.Join(tmp, "all.tar.gz")
	res, err := Dump(context.Background(), dumpOpts(ws, state, out, func(o *DumpOptions) { o.NoExclude = true }))
	if err != nil {
		t.Fatalf("dump: %v", err)
	}
	if res.Excluded != 0 {
		t.Errorf("no-exclude still recorded %d exclusions", res.Excluded)
	}
	mustContain(t, archiveNames(t, out), "workspaces/proj/web/node_modules/x/index.js")
	mustContain(t, archiveNames(t, out), "state/web.pid")
}

// ─── binaries ───────────────────────────────────────────────────────────────

func TestDumpBinariesBuildAndDegrade(t *testing.T) {
	tmp := t.TempDir()
	ws := fixtureWS(t, tmp)
	state := fixtureState(t, tmp, ws)
	src := helloModule(t, tmp)

	// native platform builds from the hello module → embedded
	out := filepath.Join(tmp, "a.tar.gz")
	native := runtime.GOOS + "/" + runtime.GOARCH
	res, err := Dump(context.Background(), dumpOpts(ws, state, out, func(o *DumpOptions) {
		o.BuildDir = src
		o.Goos = []string{native}
	}))
	if err != nil {
		t.Fatalf("dump: %v", err)
	}
	if len(res.Binaries) != 1 || res.Binaries[0] != native {
		t.Fatalf("binaries = %v", res.Binaries)
	}
	mustContain(t, archiveNames(t, out), "bin/"+runtime.GOOS+"-"+runtime.GOARCH+"/rick")
	m := readManifest(t, out)
	if len(m.Binaries) != 1 || m.Binaries[0].SHA256 == "" {
		t.Errorf("manifest binaries: %+v", m.Binaries)
	}

	// fake non-native platform fails to build → warning, dump still succeeds
	out = filepath.Join(tmp, "b.tar.gz")
	res, err = Dump(context.Background(), dumpOpts(ws, state, out, func(o *DumpOptions) {
		o.BuildDir = src
		o.Goos = []string{"fakeos/fakearch"}
	}))
	if err != nil {
		t.Fatalf("dump with only-failing platform: %v", err)
	}
	if len(res.Binaries) != 0 {
		t.Errorf("expected no binaries, got %v", res.Binaries)
	}
	if len(res.Warnings) == 0 {
		t.Error("expected a skip warning")
	}

	// broken module + native → hard error
	broken := filepath.Join(tmp, "broken-src")
	mustMk(t, broken, filepath.Join(broken, "cmd", "rick"))
	write(t, filepath.Join(broken, "go.mod"), "module broken\n\ngo 1.25\n")
	write(t, filepath.Join(broken, "cmd", "rick", "main.go"), "this is not go code\n")
	out = filepath.Join(tmp, "c.tar.gz")
	if _, err := Dump(context.Background(), dumpOpts(ws, state, out, func(o *DumpOptions) {
		o.BuildDir = broken
		o.Goos = []string{native}
	})); err == nil {
		t.Error("native build failure must be an error")
	}

	// no BuildDir → running executable embedded as native
	out = filepath.Join(tmp, "d.tar.gz")
	res, err = Dump(context.Background(), dumpOpts(ws, state, out, func(o *DumpOptions) {
		o.BuildDir = "" // probe finds nothing (ws is not rick source, cwd is the relocate pkg)
		o.Goos = nil    // default set (all non-native) — but with no build dir we take the executable path
	}))
	if err != nil {
		t.Fatalf("dump no-builddir: %v", err)
	}
	if len(res.Binaries) != 1 || res.Binaries[0] != native {
		t.Errorf("no-builddir binaries = %v, want [%s]", res.Binaries, native)
	}
}

// ─── helpers ────────────────────────────────────────────────────────────────

func TestDumpOutputInsideWorkspaceSelfExclusion(t *testing.T) {
	tmp := t.TempDir()
	ws := fixtureWS(t, tmp)
	state := fixtureState(t, tmp, ws)
	out := filepath.Join(ws, "self.tar.gz") // inside the packed workspace
	if _, err := Dump(context.Background(), dumpOpts(ws, state, out, nil)); err != nil {
		t.Fatalf("dump: %v", err)
	}
	for _, n := range archiveNames(t, out) {
		if strings.Contains(n, "self.tar.gz") {
			t.Errorf("archive contains itself: %s", n)
		}
	}
	// second dump over the existing archive: tmp + sidecar excluded, stable
	if _, err := Dump(context.Background(), dumpOpts(ws, state, out, nil)); err != nil {
		t.Fatalf("second dump: %v", err)
	}
}

func archiveNames(t *testing.T, path string) []string {
	t.Helper()
	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("open archive: %v", err)
	}
	defer f.Close()
	gz, err := gzip.NewReader(f)
	if err != nil {
		t.Fatalf("gzip: %v", err)
	}
	tr := tar.NewReader(gz)
	var out []string
	for {
		hdr, err := tr.Next()
		if err != nil {
			break
		}
		out = append(out, strings.TrimSuffix(hdr.Name, "/"))
	}
	return out
}

func readManifest(t *testing.T, path string) *Manifest {
	t.Helper()
	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("open archive: %v", err)
	}
	defer f.Close()
	gz, err := gzip.NewReader(f)
	if err != nil {
		t.Fatalf("gzip: %v", err)
	}
	tr := tar.NewReader(gz)
	for {
		hdr, err := tr.Next()
		if err != nil {
			t.Fatalf("manifest.json not found in archive")
		}
		if strings.TrimSuffix(hdr.Name, "/") == "manifest.json" {
			m, err := ReadManifest(tr)
			if err != nil {
				t.Fatalf("read manifest: %v", err)
			}
			return m
		}
	}
}

func mustContain(t *testing.T, names []string, want string) {
	t.Helper()
	for _, n := range names {
		if n == want {
			return
		}
	}
	t.Errorf("archive missing %s (have %d entries)", want, len(names))
}

func mustMk(t *testing.T, dirs ...string) {
	t.Helper()
	for _, d := range dirs {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", d, err)
		}
	}
}

func write(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir for %s: %v", path, err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func writeJSON(t *testing.T, path string, v any) {
	t.Helper()
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	write(t, path, string(data)+"\n")
}

func readJSON(t *testing.T, path string) map[string]any {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	var m map[string]any
	dec := json.NewDecoder(strings.NewReader(string(data)))
	dec.UseNumber()
	if err := dec.Decode(&m); err != nil {
		t.Fatalf("parse %s: %v", path, err)
	}
	return m
}

func chmod(t *testing.T, path string, mode os.FileMode) {
	t.Helper()
	if err := os.Chmod(path, mode); err != nil {
		t.Fatalf("chmod %s: %v", path, err)
	}
}

func symlink(t *testing.T, target, link string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(link), 0o755); err != nil {
		t.Fatalf("mkdir for %s: %v", link, err)
	}
	if err := os.Symlink(target, link); err != nil {
		t.Fatalf("symlink %s: %v", link, err)
	}
}

// User --exclude patterns prune workspace subtrees on top of the default
// rules; invalid patterns (absolute / escaping) are rejected before any
// byte streams.
func TestDumpExtraExclude(t *testing.T) {
	tmp := t.TempDir()
	ws := fixtureWS(t, tmp)
	write(t, filepath.Join(ws, "bigdata", "work", "ckpt.bin"), "weights")
	write(t, filepath.Join(ws, "bigdata", "keep.txt"), "keep me")
	state := fixtureState(t, tmp, ws)
	out := filepath.Join(tmp, "extra.tar.gz")
	opts := dumpOpts(ws, state, out, func(o *DumpOptions) {
		o.BuildDir = helloModule(t, tmp)
		o.ExtraExclude = []string{"bigdata/work"}
	})
	if _, err := Dump(context.Background(), opts); err != nil {
		t.Fatalf("dump with extra exclude: %v", err)
	}
	// archive must NOT contain bigdata/work but MUST contain bigdata/keep.txt
	list := archiveListing(t, out)
	for _, n := range list {
		if strings.HasPrefix(n, "workspaces/proj/bigdata/work/") {
			t.Errorf("excluded dir leaked into archive: %s", n)
		}
	}
	if !archiveHas(t, out, "workspaces/proj/bigdata/keep.txt") {
		t.Error("sibling of excluded dir must travel")
	}
	// invalid patterns rejected
	for _, bad := range []string{"/abs/path", "../escape", "", "  "} {
		o := dumpOpts(ws, state, filepath.Join(tmp, "x.tar.gz"), nil)
		o.BuildDir = helloModule(t, tmp)
		o.ExtraExclude = []string{bad}
		if _, err := Dump(context.Background(), o); err == nil {
			t.Errorf("pattern %q must be rejected", bad)
		}
	}
}

func archiveListing(t *testing.T, path string) []string {
	t.Helper()
	f, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	gz, err := gzip.NewReader(f)
	if err != nil {
		t.Fatal(err)
	}
	defer gz.Close()
	var out []string
	tr := tar.NewReader(gz)
	for {
		h, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatal(err)
		}
		out = append(out, h.Name)
	}
	return out
}

func archiveHas(t *testing.T, path, name string) bool {
	t.Helper()
	for _, n := range archiveListing(t, path) {
		if n == name {
			return true
		}
	}
	return false
}

// A permission hole inside a packed tree (another user's 700 dir) must not
// fail the dump: the subtree is skipped and recorded in the manifest as
// unreadable (real-machine scenario: rick-dev-home web-search-cache).
func TestDumpUnreadableSkippedNotFailed(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root reads through 0700; test needs a non-root run")
	}
	tmp := t.TempDir()
	ws := fixtureWS(t, tmp)
	hole := filepath.Join(ws, "someone-else-cache")
	mustMk(t, hole)
	write(t, filepath.Join(hole, "secret.txt"), "x")
	chmod(t, hole, 0o000)
	defer chmod(t, hole, 0o755) // cleanup on hosts where TempDir removal needs it
	state := fixtureState(t, tmp, ws)
	out := filepath.Join(tmp, "hole.tar.gz")
	_, err := Dump(context.Background(), dumpOpts(ws, state, out, func(o *DumpOptions) {
		o.BuildDir = helloModule(t, tmp)
	}))
	if err != nil {
		t.Fatalf("dump must survive a permission hole: %v", err)
	}
	// archive must not contain the hole
	if archiveHas(t, out, "workspaces/proj/someone-else-cache/secret.txt") {
		t.Error("unreadable subtree leaked into archive")
	}
	// manifest must record it honestly
	found := false
	f, err := os.Open(out)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	gz, _ := gzip.NewReader(f)
	tr := tar.NewReader(gz)
	for {
		h, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatal(err)
		}
		if h.Name != "manifest.json" {
			continue
		}
		var m Manifest
		if err := json.NewDecoder(tr).Decode(&m); err != nil {
			t.Fatal(err)
		}
		for _, e := range m.Excluded {
			if strings.Contains(e.Path, "someone-else-cache") && e.Kind == "unreadable" {
				found = true
			}
		}
	}
	if !found {
		t.Error("unreadable hole not recorded in manifest exclusions")
	}
	// sibling content still traveled
	if !archiveHas(t, out, "workspaces/proj/README.md") {
		t.Error("readable content lost")
	}
}

// Traverse-able directory holding an UNREADABLE file (ACL/other-user): the
// walk passes but the open fails — the entry must be skipped (no empty tar
// entry) and recorded, the dump must survive.
func TestDumpUnreadableFileSkippedNotFailed(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root reads through everything; test needs a non-root run")
	}
	tmp := t.TempDir()
	ws := fixtureWS(t, tmp)
	dir := filepath.Join(ws, "shared-run")
	mustMk(t, dir)
	write(t, filepath.Join(dir, "secret.md"), "x")
	chmod(t, filepath.Join(dir, "secret.md"), 0o000)
	state := fixtureState(t, tmp, ws)
	out := filepath.Join(tmp, "hole2.tar.gz")
	if _, err := Dump(context.Background(), dumpOpts(ws, state, out, func(o *DumpOptions) {
		o.BuildDir = helloModule(t, tmp)
	})); err != nil {
		t.Fatalf("dump must survive an unreadable file: %v", err)
	}
	if archiveHas(t, out, "workspaces/proj/shared-run/secret.md") {
		t.Error("unreadable file leaked into archive (or left an empty entry)")
	}
	if !archiveHas(t, out, "workspaces/proj/README.md") {
		t.Error("readable content lost")
	}
}
