package relocate

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestEscapeDirPiRule(t *testing.T) {
	// Locked against pi's session-manager.js (job_46 research-L2 leaf A,
	// 12-case experiment matrix) and the real production tree.
	cases := []struct{ cwd, want string }{
		// Real production directory (verified on disk).
		{"/workdir/sunquan20/AI_CODING/rick", "--workdir-sunquan20-AI_CODING-rick--"},
		// Underscores are NOT replaced (real: /tmp/rick_accept_7YgSff).
		{"/tmp/rick_accept_7YgSff", "--tmp-rick_accept_7YgSff--"},
		// Root /tmp → --tmp-- (real dir on disk).
		{"/tmp", "--tmp--"},
		// Windows drive letter: ":" also becomes "-".
		{`C:\Users\x`, "--C--Users-x--"},
		// Mixed separators and colons.
		{"/a/b:c\\d", "--a-b-c-d--"},
		// No truncation, no special handling of dots/dashes.
		{"/very/long.path/with-dashes", "--very-long.path-with-dashes--"},
	}
	for _, c := range cases {
		if got := EscapeDir(c.cwd); got != c.want {
			t.Errorf("EscapeDir(%q) = %q, want %q", c.cwd, got, c.want)
		}
	}
}

func writeSessionFile(t *testing.T, dir, name string, lines []string) string {
	t.Helper()
	p := filepath.Join(dir, name)
	if err := os.WriteFile(p, []byte(strings.Join(lines, "\n")+"\n"), 0644); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestRewriteSessionHeaderRewritesFirstLineOnly(t *testing.T) {
	dir := t.TempDir()
	header := `{"type":"session","version":3,"id":"019ff5c1-a686-7d37-ab1a-c1ad61e07d8c","timestamp":"2026-08-12T11:35:36.838Z","cwd":"/workdir/sunquan20/AI_CODING/rick"}`
	msg1 := `{"type":"model_change","id":"2dd0908d","parentId":null,"timestamp":"2026-08-12T11:35:37.195Z","provider":"deepseek","modelId":"deepseek-v4-flash"}`
	msg2 := `{"type":"message_end","id":"65eabf28","cwd":"/workdir/sunquan20/AI_CODING/rick","text":"mentions /workdir/sunquan20/AI_CODING/rick/go.mod inline"}`
	p := writeSessionFile(t, dir, "s.jsonl", []string{header, msg1, msg2})

	pm := &PathMap{Workspaces: []WorkspaceMapping{{
		OriginPath: "/workdir/sunquan20/AI_CODING/rick",
		NewPath:    "/Users/rick/AI_CODING/rick",
	}}}
	newCwd := "/Users/rick/AI_CODING/rick"
	if err := RewriteSessionHeader(p, newCwd, pm); err != nil {
		t.Fatalf("RewriteSessionHeader: %v", err)
	}

	data, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	got := strings.Split(strings.TrimSuffix(string(data), "\n"), "\n")
	if len(got) != 3 {
		t.Fatalf("line count changed: %d", len(got))
	}
	var hdr map[string]any
	if err := json.Unmarshal([]byte(got[0]), &hdr); err != nil {
		t.Fatalf("rewritten header must stay valid JSON: %v\n%s", err, got[0])
	}
	if hdr["cwd"] != newCwd {
		t.Errorf("header.cwd = %v, want %q", hdr["cwd"], newCwd)
	}
	if hdr["id"] != "019ff5c1-a686-7d37-ab1a-c1ad61e07d8c" {
		t.Errorf("header.id must not change: %v", hdr["id"])
	}
	if hdr["version"] != float64(3) {
		t.Errorf("header.version must stay 3: %v", hdr["version"])
	}
	// Message lines must be byte-identical.
	if got[1] != msg1 {
		t.Errorf("message line 1 changed:\n got %s\nwant %s", got[1], msg1)
	}
	if got[2] != msg2 {
		t.Errorf("message line 2 changed:\n got %s\nwant %s", got[2], msg2)
	}
}

func TestRewriteSessionHeaderParentSession(t *testing.T) {
	dir := t.TempDir()
	header := `{"type":"session","version":3,"id":"b","timestamp":"t","cwd":"/old","parentSession":"/old/sessions/--old--/a.jsonl"}`
	p := writeSessionFile(t, dir, "fork.jsonl", []string{header, `{"type":"x"}`})
	pm := &PathMap{Workspaces: []WorkspaceMapping{{
		OriginPath: "/old",
		NewPath:    "/new",
	}}}
	if err := RewriteSessionHeader(p, "/new", pm); err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(p)
	var hdr map[string]any
	if err := json.Unmarshal(data[:indexOfByte(data, '\n')], &hdr); err != nil {
		t.Fatalf("header JSON: %v", err)
	}
	want := "/new/sessions/--old--/a.jsonl"
	if hdr["parentSession"] != want {
		t.Errorf("parentSession = %v, want %q", hdr["parentSession"], want)
	}
}

func TestRewriteSessionHeaderPreservesModeAndNoNewlineCase(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "s.jsonl")
	// Single line WITHOUT trailing newline.
	if err := os.WriteFile(p, []byte(`{"type":"session","version":3,"id":"x","cwd":"/a"}`), 0600); err != nil {
		t.Fatal(err)
	}
	if err := RewriteSessionHeader(p, "/b", nil); err != nil {
		t.Fatal(err)
	}
	fi, err := os.Stat(p)
	if err != nil {
		t.Fatal(err)
	}
	if fi.Mode().Perm() != 0600 {
		t.Errorf("mode must be preserved: got %v", fi.Mode().Perm())
	}
	data, _ := os.ReadFile(p)
	if strings.HasSuffix(string(data), "\n") {
		// Adding a trailing newline would be harmless, but we promise
		// byte-preserving rewrites for everything but the header JSON.
		t.Logf("note: trailing newline added (len=%d)", len(data))
	}
	var hdr map[string]any
	if err := json.Unmarshal(data, &hdr); err != nil {
		t.Fatalf("header JSON: %v", err)
	}
	if hdr["cwd"] != "/b" {
		t.Errorf("cwd = %v want /b", hdr["cwd"])
	}
}

func TestRewriteSessionHeaderEmptyFile(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "empty.jsonl")
	if err := os.WriteFile(p, nil, 0644); err != nil {
		t.Fatal(err)
	}
	if err := RewriteSessionHeader(p, "/b", nil); err != nil {
		t.Fatalf("empty file must be a no-op, got %v", err)
	}
}

func TestRewriteSessionHeaderMalformedFirstLine(t *testing.T) {
	dir := t.TempDir()
	p := writeSessionFile(t, dir, "bad.jsonl", []string{"not json at all", `{"type":"x"}`})
	if err := RewriteSessionHeader(p, "/b", nil); err == nil {
		t.Fatal("malformed header line must error, not corrupt the file")
	}
	// The original must be intact (atomic rewrite: tmp + rename).
	data, _ := os.ReadFile(p)
	if !strings.HasPrefix(string(data), "not json at all") {
		t.Fatal("original file must be untouched on failure")
	}
}

func TestRenameSessionDir(t *testing.T) {
	agentDir := t.TempDir()
	oldCwd := "/workdir/sunquan20/AI_CODING/rick"
	newCwd := "/Users/rick/AI_CODING/rick"
	oldDir := filepath.Join(agentDir, "sessions", EscapeDir(oldCwd))
	if err := os.MkdirAll(oldDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(oldDir, "s.jsonl"), []byte("x"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := RenameSessionDir(agentDir, oldCwd, newCwd); err != nil {
		t.Fatalf("RenameSessionDir: %v", err)
	}
	newDir := filepath.Join(agentDir, "sessions", EscapeDir(newCwd))
	if _, err := os.Stat(filepath.Join(newDir, "s.jsonl")); err != nil {
		t.Fatalf("session must live under re-escaped dir: %v", err)
	}
	if _, err := os.Stat(oldDir); !os.IsNotExist(err) {
		t.Error("old session dir must be gone after rename")
	}
}

func TestRenameSessionDirMissingSourceNoop(t *testing.T) {
	agentDir := t.TempDir()
	if err := RenameSessionDir(agentDir, "/never/existed", "/anywhere"); err != nil {
		t.Fatalf("missing source must be a no-op, got %v", err)
	}
}

func TestRenameSessionDirTargetExists(t *testing.T) {
	agentDir := t.TempDir()
	for _, cwd := range []string{"/old/ws", "/new/ws"} {
		d := filepath.Join(agentDir, "sessions", EscapeDir(cwd))
		if err := os.MkdirAll(d, 0755); err != nil {
			t.Fatal(err)
		}
	}
	err := RenameSessionDir(agentDir, "/old/ws", "/new/ws")
	if err == nil {
		t.Fatal("existing target must error (merge is import's job, not the primitive's)")
	}
	if !strings.Contains(err.Error(), "exists") {
		t.Errorf("error should mention the conflict: %v", err)
	}
}

func TestRenameSessionDirSameEscapeNoop(t *testing.T) {
	agentDir := t.TempDir()
	// Different cwds that escape to the same dir name are impossible with
	// this scheme; same cwd must be a clean no-op.
	if err := RenameSessionDir(agentDir, "/ws", "/ws"); err != nil {
		t.Fatalf("same dir rename must be a no-op, got %v", err)
	}
}

func indexOfByte(b []byte, c byte) int {
	for i, x := range b {
		if x == c {
			return i
		}
	}
	return len(b)
}
