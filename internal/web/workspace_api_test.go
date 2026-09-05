package web

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// ---- browse / create workspaces ----

func TestRoutesBrowseWorkspaces(t *testing.T) {
	e := newServerTestEnv(t, "tk", nil, nil)

	// Build a parent dir with two .rick children + one plain dir.
	parent := t.TempDir()
	wsA := filepath.Join(parent, "proj-a")
	wsB := filepath.Join(parent, "proj-b")
	plain := filepath.Join(parent, "not-a-project")
	for _, d := range []string{filepath.Join(wsA, ".rick"), filepath.Join(wsB, ".rick"), plain} {
		if err := os.MkdirAll(d, 0755); err != nil {
			t.Fatal(err)
		}
	}

	resp, body := e.req(t, "GET", "/api/workspaces/browse?path="+parent, "", "tk", "")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("browse: status %d body %s", resp.StatusCode, body)
	}
	var out struct {
		Workspaces []struct {
			Path string `json:"path"`
			Name string `json:"name"`
		} `json:"workspaces"`
	}
	if err := json.Unmarshal([]byte(body), &out); err != nil {
		t.Fatalf("browse body: %s", body)
	}
	if len(out.Workspaces) != 2 {
		t.Fatalf("browse: want 2 .rick dirs, got %d (%s)", len(out.Workspaces), body)
	}
	got := map[string]bool{}
	for _, w := range out.Workspaces {
		got[w.Name] = true
	}
	if !got["proj-a"] || !got["proj-b"] {
		t.Fatalf("browse names: %+v", got)
	}

	// Missing path → 400.
	resp, body = e.req(t, "GET", "/api/workspaces/browse?path="+filepath.Join(parent, "nope"), "", "tk", "")
	if resp.StatusCode != http.StatusBadRequest || !strings.Contains(body, "invalid_params") {
		t.Fatalf("browse missing: status %d body %s", resp.StatusCode, body)
	}
}

func TestRoutesCreateWorkspace(t *testing.T) {
	e := newServerTestEnv(t, "tk", nil, nil)

	newPath := filepath.Join(t.TempDir(), "brand-new")
	resp, body := e.req(t, "POST", "/api/workspaces/create", fmt.Sprintf(`{"path":%q}`, newPath), "tk", "")
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("create: status %d body %s", resp.StatusCode, body)
	}
	var entry WorkspaceEntry
	if err := json.Unmarshal([]byte(body), &entry); err != nil {
		t.Fatalf("create body: %s", body)
	}
	// .rick structure bootstrapped with the standard six dirs.
	for _, sub := range []string{"", "jobs", "draft", "domain", "loops", "skills", "dream"} {
		dir := filepath.Join(newPath, ".rick", sub)
		fi, err := os.Stat(dir)
		if err != nil || !fi.IsDir() {
			t.Fatalf("bootstrap missing %s (err %v)", dir, err)
		}
	}

	// Existing dir without .rick → bootstrapped + registered.
	existing := t.TempDir()
	resp, body = e.req(t, "POST", "/api/workspaces/create", fmt.Sprintf(`{"path":%q}`, existing), "tk", "")
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("create existing: status %d body %s", resp.StatusCode, body)
	}
	if _, err := os.Stat(filepath.Join(existing, ".rick", "jobs")); err != nil {
		t.Fatalf("existing dir not bootstrapped: %v", err)
	}

	// Idempotent re-create → 200.
	resp, _ = e.req(t, "POST", "/api/workspaces/create", fmt.Sprintf(`{"path":%q}`, newPath), "tk", "")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("idempotent create: status %d", resp.StatusCode)
	}

	// Workspace is listed.
	resp, body = e.req(t, "GET", "/api/workspaces", "", "tk", "")
	if resp.StatusCode != http.StatusOK || !strings.Contains(body, "brand-new") {
		t.Fatalf("list after create: %s", body)
	}
}

// ---- session prompt files ----

// TestRoutesSessionPrompt verifies GET /api/sessions/{id}/prompt serves the
// recorded _prompt_file/_method_file contents with security constraints.
func TestRoutesSessionPrompt(t *testing.T) {
	e := newServerTestEnv(t, "tk", nil, nil)

	// Build a workspace with .rick and register it.
	wsRoot := t.TempDir()
	rickDir := filepath.Join(wsRoot, ".rick")
	if err := os.MkdirAll(filepath.Join(rickDir, "jobs", "job_1", "plan", "prompts"), 0755); err != nil {
		t.Fatal(err)
	}
	resp, body := e.req(t, "POST", "/api/workspaces", fmt.Sprintf(`{"path":%q}`, wsRoot), "tk", "")
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("register ws: %d %s", resp.StatusCode, body)
	}
	var ws WorkspaceEntry
	if err := json.Unmarshal([]byte(body), &ws); err != nil {
		t.Fatal(err)
	}

	// Write the prompt + method files.
	promptPath := filepath.Join(rickDir, "jobs", "job_1", "plan", "prompts", "plan_prompt.md")
	methodPath := filepath.Join(rickDir, "jobs", "job_1", "plan", "method.md")
	if err := os.WriteFile(promptPath, []byte("INSTANCE_CONTENT"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(methodPath, []byte("METHOD_CONTENT"), 0644); err != nil {
		t.Fatal(err)
	}

	// Register a session entry with the recorded paths (like createSessionLocked).
	sessionID := "sess-prompt-test-0001"
	if err := e.sessions.Add(SessionEntry{
		ID:          sessionID,
		WorkspaceID: ws.ID,
		Type:        SessionTypePlan,
		Params: map[string]any{
			"_prompt_file": promptPath,
			"_method_file": methodPath,
		},
		Status:    SessionStatusActive,
		CreatedAt: mustTime(t),
	}); err != nil {
		t.Fatal(err)
	}

	resp, body = e.req(t, "GET", "/api/sessions/"+sessionID+"/prompt", "", "tk", "")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("prompt: status %d body %s", resp.StatusCode, body)
	}
	var out struct {
		Method   string `json:"method"`
		Instance string `json:"instance"`
	}
	if err := json.Unmarshal([]byte(body), &out); err != nil {
		t.Fatalf("prompt body: %s", body)
	}
	if out.Method != "METHOD_CONTENT" || out.Instance != "INSTANCE_CONTENT" {
		t.Fatalf("prompt contents: method=%q instance=%q", out.Method, out.Instance)
	}
}

// TestRoutesSessionPromptTraversalRejected verifies a session whose recorded
// prompt path escapes the workspace .rick tree is refused (not served).
func TestRoutesSessionPromptTraversalRejected(t *testing.T) {
	e := newServerTestEnv(t, "tk", nil, nil)

	wsRoot := t.TempDir()
	rickDir := filepath.Join(wsRoot, ".rick")
	if err := os.MkdirAll(rickDir, 0755); err != nil {
		t.Fatal(err)
	}
	resp, body := e.req(t, "POST", "/api/workspaces", fmt.Sprintf(`{"path":%q}`, wsRoot), "tk", "")
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("register ws: %d %s", resp.StatusCode, body)
	}
	var ws WorkspaceEntry
	if err := json.Unmarshal([]byte(body), &ws); err != nil {
		t.Fatal(err)
	}

	// Prompt file recorded OUTSIDE the workspace (e.g. /etc or a sibling).
	outside := filepath.Join(t.TempDir(), "evil.md")
	if err := os.WriteFile(outside, []byte("SECRET"), 0644); err != nil {
		t.Fatal(err)
	}

	sessionID := "sess-prompt-traversal-0001"
	if err := e.sessions.Add(SessionEntry{
		ID:          sessionID,
		WorkspaceID: ws.ID,
		Type:        SessionTypePlan,
		Params:      map[string]any{"_prompt_file": outside},
		Status:      SessionStatusActive,
		CreatedAt:   mustTime(t),
	}); err != nil {
		t.Fatal(err)
	}

	resp, body = e.req(t, "GET", "/api/sessions/"+sessionID+"/prompt", "", "tk", "")
	if resp.StatusCode != http.StatusBadRequest || !strings.Contains(body, "invalid_path") {
		t.Fatalf("traversal: want 400 invalid_path, got %d %s", resp.StatusCode, body)
	}
}

// TestRoutesSessionPromptMissingFile verifies 404 when the recorded prompt
// file does not exist.
func TestRoutesSessionPromptMissingFile(t *testing.T) {
	e := newServerTestEnv(t, "tk", nil, nil)

	wsRoot := t.TempDir()
	if err := os.MkdirAll(filepath.Join(wsRoot, ".rick"), 0755); err != nil {
		t.Fatal(err)
	}
	resp, body := e.req(t, "POST", "/api/workspaces", fmt.Sprintf(`{"path":%q}`, wsRoot), "tk", "")
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("register ws: %d %s", resp.StatusCode, body)
	}
	var ws WorkspaceEntry
	if err := json.Unmarshal([]byte(body), &ws); err != nil {
		t.Fatal(err)
	}

	sessionID := "sess-prompt-missing-0001"
	if err := e.sessions.Add(SessionEntry{
		ID:          sessionID,
		WorkspaceID: ws.ID,
		Type:        SessionTypePlan,
		Params:      map[string]any{"_prompt_file": filepath.Join(wsRoot, ".rick", "nope.md")},
		Status:      SessionStatusActive,
		CreatedAt:   mustTime(t),
	}); err != nil {
		t.Fatal(err)
	}

	resp, body = e.req(t, "GET", "/api/sessions/"+sessionID+"/prompt", "", "tk", "")
	if resp.StatusCode != http.StatusNotFound || !strings.Contains(body, "not_found") {
		t.Fatalf("missing: want 404, got %d %s", resp.StatusCode, body)
	}
}

func mustTime(t *testing.T) time.Time {
	return time.Now()
}
