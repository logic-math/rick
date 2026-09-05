package web

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// ---- fs 浏览接口（多级路径选择器数据源）----

func TestRoutesFSList(t *testing.T) {
	e := newServerTestEnv(t, "tk", nil, nil)

	root := t.TempDir()
	for _, d := range []string{"a", "b", "c", "c/nested", ".hidden"} {
		if err := os.MkdirAll(filepath.Join(root, d), 0755); err != nil {
			t.Fatal(err)
		}
	}
	// a plain file must be excluded
	if err := os.WriteFile(filepath.Join(root, "afile.txt"), []byte("x"), 0644); err != nil {
		t.Fatal(err)
	}

	resp, body := e.req(t, "GET", "/api/fs/list?path="+root, "", "tk", "")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("fs/list: status %d body %s", resp.StatusCode, body)
	}
	var out struct {
		Path    string `json:"path"`
		Parent  string `json:"parent"`
		Entries []struct {
			Path string `json:"path"`
			Name string `json:"name"`
		} `json:"entries"`
	}
	if err := json.Unmarshal([]byte(body), &out); err != nil {
		t.Fatalf("fs/list body: %s", body)
	}
	if out.Path != root {
		t.Fatalf("fs/list path: got %q want %q", out.Path, root)
	}
	var names []string
	for _, en := range out.Entries {
		names = append(names, en.Name)
	}
	// sorted a,b,c; .hidden excluded; afile.txt excluded
	if got := strings.Join(names, ","); got != "a,b,c" {
		t.Fatalf("fs/list names: got %q want %q", got, "a,b,c")
	}
	// entries carry absolute paths
	if len(out.Entries) > 0 && out.Entries[0].Path != filepath.Join(root, "a") {
		t.Fatalf("fs/list entry path: got %q want %q", out.Entries[0].Path, filepath.Join(root, "a"))
	}

	// nested listing (c contains nested)
	resp2, body2 := e.req(t, "GET", "/api/fs/list?path="+filepath.Join(root, "c"), "", "tk", "")
	if resp2.StatusCode != http.StatusOK || !strings.Contains(body2, "nested") {
		t.Fatalf("fs/list nested: status %d body %s", resp2.StatusCode, body2)
	}

	// missing path → 400 invalid_params
	resp3, body3 := e.req(t, "GET", "/api/fs/list?path="+filepath.Join(root, "nope"), "", "tk", "")
	if resp3.StatusCode != http.StatusBadRequest || !strings.Contains(body3, "invalid_params") {
		t.Fatalf("fs/list missing: status %d body %s", resp3.StatusCode, body3)
	}

	// empty path → 400
	resp4, _ := e.req(t, "GET", "/api/fs/list", "", "tk", "")
	if resp4.StatusCode != http.StatusBadRequest {
		t.Fatalf("fs/list empty: status %d", resp4.StatusCode)
	}
}

func TestRoutesFSStatus(t *testing.T) {
	e := newServerTestEnv(t, "tk", nil, nil)

	root := t.TempDir()
	ws := filepath.Join(root, "proj")
	if err := os.MkdirAll(filepath.Join(ws, ".rick"), 0755); err != nil {
		t.Fatal(err)
	}
	plain := filepath.Join(root, "plain")
	if err := os.MkdirAll(plain, 0755); err != nil {
		t.Fatal(err)
	}

	// workspace (has .rick)
	resp, body := e.req(t, "GET", "/api/fs/status?path="+ws, "", "tk", "")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status ws: status %d body %s", resp.StatusCode, body)
	}
	var st struct {
		Exists  bool `json:"exists"`
		IsDir   bool `json:"is_dir"`
		HasRick bool `json:"has_rick"`
	}
	if err := json.Unmarshal([]byte(body), &st); err != nil {
		t.Fatalf("status ws body: %s", body)
	}
	if !st.Exists || !st.IsDir || !st.HasRick {
		t.Fatalf("status ws: %+v", st)
	}

	// plain dir → exists, is_dir, no .rick
	_, body2 := e.req(t, "GET", "/api/fs/status?path="+plain, "", "tk", "")
	var st2 struct {
		Exists  bool `json:"exists"`
		IsDir   bool `json:"is_dir"`
		HasRick bool `json:"has_rick"`
	}
	if err := json.Unmarshal([]byte(body2), &st2); err != nil {
		t.Fatalf("status plain body: %s", body2)
	}
	if !st2.Exists || !st2.IsDir || st2.HasRick {
		t.Fatalf("status plain: %+v", st2)
	}

	// missing → exists=false
	_, body3 := e.req(t, "GET", "/api/fs/status?path="+filepath.Join(root, "nope"), "", "tk", "")
	var st3 struct {
		Exists bool `json:"exists"`
	}
	if err := json.Unmarshal([]byte(body3), &st3); err != nil {
		t.Fatalf("status missing body: %s", body3)
	}
	if st3.Exists {
		t.Fatalf("status missing: exists should be false")
	}

	// empty path → 400
	resp4, _ := e.req(t, "GET", "/api/fs/status", "", "tk", "")
	if resp4.StatusCode != http.StatusBadRequest {
		t.Fatalf("status empty: status %d", resp4.StatusCode)
	}
}

func TestRoutesFSMkdir(t *testing.T) {
	e := newServerTestEnv(t, "tk", nil, nil)

	parent := t.TempDir()
	resp, body := e.req(t, "POST", "/api/fs/mkdir", fmt.Sprintf(`{"path":%q,"name":"newdir"}`, parent), "tk", "")
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("mkdir: status %d body %s", resp.StatusCode, body)
	}
	var out struct {
		Path string `json:"path"`
	}
	if err := json.Unmarshal([]byte(body), &out); err != nil {
		t.Fatalf("mkdir body: %s", body)
	}
	if out.Path != filepath.Join(parent, "newdir") {
		t.Fatalf("mkdir path: got %q", out.Path)
	}
	fi, err := os.Stat(out.Path)
	if err != nil || !fi.IsDir() {
		t.Fatalf("mkdir: created dir missing (err %v)", err)
	}

	// idempotent (already exists → 200)
	resp2, _ := e.req(t, "POST", "/api/fs/mkdir", fmt.Sprintf(`{"path":%q,"name":"newdir"}`, parent), "tk", "")
	if resp2.StatusCode != http.StatusOK {
		t.Fatalf("mkdir idempotent: status %d", resp2.StatusCode)
	}

	// invalid name (path traversal) → 400
	resp3, body3 := e.req(t, "POST", "/api/fs/mkdir", fmt.Sprintf(`{"path":%q,"name":"../evil"}`, parent), "tk", "")
	if resp3.StatusCode != http.StatusBadRequest || !strings.Contains(body3, "invalid_params") {
		t.Fatalf("mkdir traversal: status %d body %s", resp3.StatusCode, body3)
	}

	// empty name → 400
	resp4, _ := e.req(t, "POST", "/api/fs/mkdir", fmt.Sprintf(`{"path":%q,"name":""}`, parent), "tk", "")
	if resp4.StatusCode != http.StatusBadRequest {
		t.Fatalf("mkdir empty: status %d", resp4.StatusCode)
	}
}
