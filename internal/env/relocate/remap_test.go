package relocate

import (
	"encoding/json"
	"strings"
	"testing"
)

func testPathMap() *PathMap {
	return &PathMap{
		Workspaces: []WorkspaceMapping{{
			Name:       "rick",
			OriginPath: "/workdir/sunquan20/AI_CODING/rick",
			NewPath:    "/Users/rick/AI_CODING/rick",
			MarkerID:   "0123456789abcdef0123456789abcdef",
			LegacyID:   "a8ceb938",
		}},
		StateOrigin: "/home/hadoop-recsys/.rick",
		StateNew:    "/Users/rick/.rick",
	}
}

func TestMapPathSegmentAware(t *testing.T) {
	pm := testPathMap()
	cases := []struct {
		in   string
		want string
		hit  bool
	}{
		// Workspace exact and subtree.
		{"/workdir/sunquan20/AI_CODING/rick", "/Users/rick/AI_CODING/rick", true},
		{"/workdir/sunquan20/AI_CODING/rick/.rick/jobs/job_46/doing/x.md", "/Users/rick/AI_CODING/rick/.rick/jobs/job_46/doing/x.md", true},
		// Segment boundary: /rick-dev must NOT match /rick.
		{"/workdir/sunquan20/AI_CODING/rick-dev/web.go", "", false},
		// State dir mapping.
		{"/home/hadoop-recsys/.rick/web/sessions.json", "/Users/rick/.rick/web/sessions.json", true},
		// Unrelated path.
		{"/etc/hostname", "", false},
	}
	for _, c := range cases {
		got, hit := pm.MapPath(c.in)
		if hit != c.hit {
			t.Errorf("MapPath(%q) hit=%v want %v", c.in, hit, c.hit)
			continue
		}
		if hit && got != c.want {
			t.Errorf("MapPath(%q) = %q want %q", c.in, got, c.want)
		}
	}
}

func TestMapPathLongestPrefixWins(t *testing.T) {
	pm := &PathMap{
		Workspaces: []WorkspaceMapping{
			{OriginPath: "/a", NewPath: "/A", LegacyID: "aaaaaaaa"},
			{OriginPath: "/a/b", NewPath: "/B", LegacyID: "bbbbbbbb"}, // longer
		},
	}
	if got, _ := pm.MapPath("/a/b/c.txt"); got != "/B/c.txt" {
		t.Errorf("longest workspace prefix must win: got %q", got)
	}
	if got, _ := pm.MapPath("/a/x.txt"); got != "/A/x.txt" {
		t.Errorf("shorter prefix fallback: got %q", got)
	}
}

func TestMapPathNilAndEmpty(t *testing.T) {
	var pm *PathMap
	if got, hit := pm.MapPath("/x"); hit || got != "/x" {
		t.Errorf("nil PathMap must pass through: %q %v", got, hit)
	}
	empty := &PathMap{}
	if got, hit := empty.MapPath("/x"); hit || got != "/x" {
		t.Errorf("empty PathMap must pass through: %q %v", got, hit)
	}
}

func TestLegacyWorkspaceID(t *testing.T) {
	// The pre-marker id scheme: first 8 hex of sha1(path). Must stay stable
	// forever — it is the key old sessions.json entries reference.
	if got := LegacyWorkspaceID("/workdir/sunquan20/AI_CODING/rick"); got != "a8ceb938" {
		t.Errorf("LegacyWorkspaceID drift: got %q want a8ceb938", got)
	}
	if len(LegacyWorkspaceID("/x")) != 8 {
		t.Error("legacy id must be 8 hex chars")
	}
}

func TestRewriteWebJSON(t *testing.T) {
	in := `{
  "version": 1,
  "workspaces": [
    {"id": "a8ceb938", "path": "/workdir/sunquan20/AI_CODING/rick", "name": "RICK-RSI", "added_at": "2026-09-22T14:53:04.868011969+08:00"},
    {"id": "79a79993", "path": "/workdir/sunquan20/HUMANS", "name": "HUMANS", "added_at": "2026-10-08T18:53:30.852802081+08:00"}
  ]
}`
	out, err := RewriteWebJSON([]byte(in), testPathMap())
	if err != nil {
		t.Fatalf("RewriteWebJSON: %v", err)
	}
	var doc struct {
		Version    int `json:"version"`
		Workspaces []struct {
			ID   string `json:"id"`
			Path string `json:"path"`
			Name string `json:"name"`
		} `json:"workspaces"`
	}
	if err := json.Unmarshal(out, &doc); err != nil {
		t.Fatalf("output must be valid JSON: %v\n%s", err, out)
	}
	if doc.Version != 1 {
		t.Errorf("version must roundtrip: %d", doc.Version)
	}
	if len(doc.Workspaces) != 2 {
		t.Fatalf("workspace count changed: %d", len(doc.Workspaces))
	}
	ws := doc.Workspaces[0]
	if ws.Path != "/Users/rick/AI_CODING/rick" {
		t.Errorf("mapped path: got %q", ws.Path)
	}
	if ws.ID != "0123456789abcdef0123456789abcdef" {
		t.Errorf("id must switch to marker id: got %q", ws.ID)
	}
	if ws.Name != "RICK-RSI" {
		t.Errorf("name must survive: got %q", ws.Name)
	}
	// Unmapped workspace passes through untouched.
	other := doc.Workspaces[1]
	if other.Path != "/workdir/sunquan20/HUMANS" || other.ID != "79a79993" {
		t.Errorf("unmapped entry must pass through: %+v", other)
	}
}

func TestRewriteWebJSONNoMarkerFallsBackToLegacyHashOfNewPath(t *testing.T) {
	pm := testPathMap()
	pm.Workspaces[0].MarkerID = "" // no marker known
	in := `{"version":1,"workspaces":[{"id":"a8ceb938","path":"/workdir/sunquan20/AI_CODING/rick","name":"r","added_at":"2026-01-01T00:00:00Z"}]}`
	out, err := RewriteWebJSON([]byte(in), pm)
	if err != nil {
		t.Fatal(err)
	}
	var doc struct {
		Workspaces []struct {
			ID   string `json:"id"`
			Path string `json:"path"`
		} `json:"workspaces"`
	}
	if err := json.Unmarshal(out, &doc); err != nil {
		t.Fatal(err)
	}
	want := LegacyWorkspaceID("/Users/rick/AI_CODING/rick")
	if doc.Workspaces[0].ID != want {
		t.Errorf("no-marker id must be legacy hash of NEW path: got %q want %q", doc.Workspaces[0].ID, want)
	}
}

func TestRewriteWebJSONInvalid(t *testing.T) {
	if _, err := RewriteWebJSON([]byte("{nope"), testPathMap()); err == nil {
		t.Fatal("invalid JSON must error")
	}
	if _, err := RewriteWebJSON([]byte(`{"version":1}`), testPathMap()); err != nil {
		t.Fatalf("registry without workspaces must pass through: %v", err)
	}
}

func TestRewriteSessionRefs(t *testing.T) {
	in := `{
  "version": 1,
  "sessions": [
    {"id": "s1", "workspace_id": "a8ceb938", "type": "easy", "status": "closed",
     "pi_session_id": "019ff5c1-a686-7d37-ab1a-c1ad61e07d8c",
     "params": {"requirement": "x", "_prompt_file": "/workdir/sunquan20/AI_CODING/rick/.rick/jobs/job_46/doing/prompts/easy_prompt.md", "_method_file": "/workdir/sunquan20/AI_CODING/rick/.rick/jobs/job_46/doing/method.md"},
     "created_at": "2026-10-11T19:08:46+08:00"},
    {"id": "s2", "workspace_id": "67250c1c", "type": "plan", "status": "closed",
     "params": {"requirement": "y"},
     "created_at": "2026-09-01T00:00:00+08:00"},
    {"id": "s3", "workspace_id": "0123456789abcdef0123456789abcdef", "type": "easy", "status": "active",
     "params": {"requirement": "z", "_prompt_file": "/workdir/sunquan20/AI_CODING/rick/.rick/jobs/job_1/doing/prompts/easy_prompt.md"},
     "created_at": "2026-10-12T00:00:00+08:00"}
  ]
}`
	out, err := RewriteSessionRefs([]byte(in), testPathMap())
	if err != nil {
		t.Fatalf("RewriteSessionRefs: %v", err)
	}
	type sessRow struct {
		ID          string         `json:"id"`
		WorkspaceID string         `json:"workspace_id"`
		Params      map[string]any `json:"params"`
	}
	var doc struct {
		Sessions []sessRow `json:"sessions"`
	}
	if err := json.Unmarshal(out, &doc); err != nil {
		t.Fatalf("output must be valid JSON: %v\n%s", err, out)
	}
	byID := map[string]sessRow{}
	for _, s := range doc.Sessions {
		byID[s.ID] = s
	}
	// s1: legacy id -> marker id, prompt/method files remapped.
	s1 := byID["s1"]
	if s1.WorkspaceID != "0123456789abcdef0123456789abcdef" {
		t.Errorf("s1 workspace_id: got %q", s1.WorkspaceID)
	}
	if s1.Params["_prompt_file"] != "/Users/rick/AI_CODING/rick/.rick/jobs/job_46/doing/prompts/easy_prompt.md" {
		t.Errorf("s1 _prompt_file: got %v", s1.Params["_prompt_file"])
	}
	if s1.Params["_method_file"] != "/Users/rick/AI_CODING/rick/.rick/jobs/job_46/doing/method.md" {
		t.Errorf("s1 _method_file: got %v", s1.Params["_method_file"])
	}
	if s1.Params["requirement"] != "x" {
		t.Errorf("s1 requirement must survive: %v", s1.Params["requirement"])
	}
	// s2: ghost id (no mapping) must stay untouched — never guess.
	if byID["s2"].WorkspaceID != "67250c1c" {
		t.Errorf("ghost id must pass through: got %q", byID["s2"].WorkspaceID)
	}
	// s3: already marker id — stays marker id (idempotent), prompt file still remapped.
	s3 := byID["s3"]
	if s3.WorkspaceID != "0123456789abcdef0123456789abcdef" {
		t.Errorf("s3 idempotent remap: got %q", s3.WorkspaceID)
	}
	if s3.Params["_prompt_file"] != "/Users/rick/AI_CODING/rick/.rick/jobs/job_1/doing/prompts/easy_prompt.md" {
		t.Errorf("s3 _prompt_file: got %v", s3.Params["_prompt_file"])
	}
}

func TestRewriteSessionRefsMissingParams(t *testing.T) {
	in := `{"version":1,"sessions":[{"id":"s","workspace_id":"a8ceb938","status":"closed"}]}`
	out, err := RewriteSessionRefs([]byte(in), testPathMap())
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(out), "0123456789abcdef0123456789abcdef") {
		t.Errorf("session without params must still get id remapped:\n%s", out)
	}
}

func TestRewriteArchivedIDs(t *testing.T) {
	in := `{"version":1,"archived":{"a8ceb938":["job_44","job_45"],"67250c1c":["job_1"]}}`
	out, err := RewriteArchivedIDs([]byte(in), testPathMap())
	if err != nil {
		t.Fatal(err)
	}
	var doc struct {
		Archived map[string][]string `json:"archived"`
	}
	if err := json.Unmarshal(out, &doc); err != nil {
		t.Fatal(err)
	}
	if _, ok := doc.Archived["0123456789abcdef0123456789abcdef"]; !ok {
		t.Error("legacy key must be renamed to marker id")
	}
	if len(doc.Archived["0123456789abcdef0123456789abcdef"]) != 2 {
		t.Error("archived job list must survive")
	}
	if _, ok := doc.Archived["a8ceb938"]; ok {
		t.Error("legacy key must be gone")
	}
	if _, ok := doc.Archived["67250c1c"]; !ok {
		t.Error("ghost key must pass through")
	}
}

func TestRewriteJobNameKeys(t *testing.T) {
	in := `{"version":1,"names":{"a8ceb938/job_44":"重构","67250c1c/job_1":"旧机遗留"}}`
	out, err := RewriteJobNameKeys([]byte(in), testPathMap())
	if err != nil {
		t.Fatal(err)
	}
	var doc struct {
		Names map[string]string `json:"names"`
	}
	if err := json.Unmarshal(out, &doc); err != nil {
		t.Fatal(err)
	}
	if doc.Names["0123456789abcdef0123456789abcdef/job_44"] != "重构" {
		t.Errorf("job name key must remap: %+v", doc.Names)
	}
	if _, ok := doc.Names["a8ceb938/job_44"]; ok {
		t.Error("legacy key must be gone")
	}
	if doc.Names["67250c1c/job_1"] != "旧机遗留" {
		t.Error("ghost key must pass through")
	}
}

func TestRewriteConfigDefaultWorkspace(t *testing.T) {
	in := `{
  "max_retries": 5,
  "runtime": "pi",
  "pi_path": "",
  "pi_extra_args": ["--provider", "deepseek", "--model", "deepseek-flash"],
  "default_workspace": "/home/hadoop-recsys/.rick",
  "git": {"user_name": "logic-math", "user_email": "logic.math.code@gmail.com"},
  "web_token": "c2f268ce02fda0d02a7804278bbba0ca"
}`
	out, err := RewriteConfigDefaultWorkspace([]byte(in), testPathMap())
	if err != nil {
		t.Fatal(err)
	}
	var doc map[string]any
	if err := json.Unmarshal(out, &doc); err != nil {
		t.Fatalf("output must be valid JSON: %v", err)
	}
	if doc["default_workspace"] != "/Users/rick/.rick" {
		t.Errorf("default_workspace: got %v", doc["default_workspace"])
	}
	if doc["max_retries"] != float64(5) {
		t.Errorf("other fields must survive: max_retries=%v", doc["max_retries"])
	}
	if doc["web_token"] != "c2f268ce02fda0d02a7804278bbba0ca" {
		t.Errorf("web_token must survive: %v", doc["web_token"])
	}
	args, _ := doc["pi_extra_args"].([]any)
	if len(args) != 4 || args[0] != "--provider" || args[1] != "deepseek" {
		t.Errorf("pi_extra_args must survive: %v", args)
	}
}

func TestRewriteConfigDefaultWorkspaceUnmapped(t *testing.T) {
	in := `{"default_workspace": "/elsewhere/.rick"}`
	out, err := RewriteConfigDefaultWorkspace([]byte(in), testPathMap())
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(out), "/elsewhere/.rick") {
		t.Errorf("unmapped default_workspace must pass through:\n%s", out)
	}
	// No default_workspace key at all.
	out2, err := RewriteConfigDefaultWorkspace([]byte(`{"max_retries": 5}`), testPathMap())
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(out2), `"max_retries": 5`) {
		t.Errorf("config without the key must roundtrip:\n%s", out2)
	}
}

func TestRewriteStartScript(t *testing.T) {
	in := `#!/bin/bash
# rick web 启动脚本（单端口 8413）
export http_proxy=http://10.229.18.27:8412
export https_proxy=http://10.229.18.27:8412
export no_proxy=localhost,127.0.0.1
cd /workdir/sunquan20/AI_CODING/rick
exec ./bin/rick web --listen 0.0.0.0 --port 8413
`
	out, err := RewriteStartScript([]byte(in), testPathMap())
	if err != nil {
		t.Fatalf("RewriteStartScript: %v", err)
	}
	s := string(out)
	if !strings.Contains(s, "cd /Users/rick/AI_CODING/rick") {
		t.Errorf("cd line must be remapped:\n%s", s)
	}
	if strings.Contains(s, "cd /workdir/sunquan20/AI_CODING/rick") {
		t.Errorf("old cd line must be gone:\n%s", s)
	}
	// Proxy lines must be untouched (target machine's network business).
	if !strings.Contains(s, "export http_proxy=http://10.229.18.27:8412") {
		t.Errorf("proxy lines must survive:\n%s", s)
	}
	if !strings.Contains(s, "exec ./bin/rick web --listen 0.0.0.0 --port 8413") {
		t.Errorf("exec line must survive:\n%s", s)
	}
}

func TestRewriteStartScriptQuotedAndComments(t *testing.T) {
	in := "cd \"/workdir/sunquan20/AI_CODING/rick\" # workspace root\ncd /unmapped/path\n"
	out, err := RewriteStartScript([]byte(in), testPathMap())
	if err != nil {
		t.Fatal(err)
	}
	s := string(out)
	if !strings.Contains(s, `cd "/Users/rick/AI_CODING/rick" # workspace root`) {
		t.Errorf("quoted cd with trailing comment must remap cleanly:\n%s", s)
	}
	if !strings.Contains(s, "cd /unmapped/path") {
		t.Errorf("unmapped cd must survive:\n%s", s)
	}
}
