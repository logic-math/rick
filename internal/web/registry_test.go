package web

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// newTestWorkspace creates a temp directory containing a .rick/ subdir so it
// qualifies as a registrable workspace.
func newTestWorkspace(t *testing.T) string {
	t.Helper()
	ws := t.TempDir()
	if err := os.MkdirAll(filepath.Join(ws, ".rick"), 0755); err != nil {
		t.Fatalf("create test workspace .rick: %v", err)
	}
	return ws
}

// registryPathUnder isolates the registry file under the test's own temp dir
// (no HOME needed — Load* takes an explicit path).
func registryPathUnder(t *testing.T) string {
	t.Helper()
	return filepath.Join(t.TempDir(), "web.json")
}

func TestRegistryWebPaths(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	if got := WebStateDir(); got != filepath.Join(home, ".rick", "web") {
		t.Errorf("WebStateDir: got %q", got)
	}
	if got := WebConfigPath(); got != filepath.Join(home, ".rick", "web.json") {
		t.Errorf("WebConfigPath: got %q", got)
	}
	if got := SessionsPath(); got != filepath.Join(home, ".rick", "web", "sessions.json") {
		t.Errorf("SessionsPath: got %q", got)
	}
	if got := PidPath(); got != filepath.Join(home, ".rick", "web.pid") {
		t.Errorf("PidPath: got %q", got)
	}
}

func TestRegistryWorkspaceAddListGet(t *testing.T) {
	path := registryPathUnder(t)
	r, err := LoadWorkspaceRegistry(path)
	if err != nil {
		t.Fatalf("LoadWorkspaceRegistry on missing file: %v", err)
	}
	if len(r.List()) != 0 {
		t.Fatalf("fresh registry should be empty, got %d", len(r.List()))
	}

	ws := newTestWorkspace(t)
	entry, created, err := r.Add(ws, "my-workspace")
	if err != nil {
		t.Fatalf("Add: %v", err)
	}
	if !created {
		t.Errorf("first Add should report created=true")
	}
	if entry.Name != "my-workspace" {
		t.Errorf("entry.Name: got %q", entry.Name)
	}
	if entry.ID != workspaceID(entry.Path) {
		t.Errorf("entry.ID %q != workspaceID(path) %q", entry.ID, workspaceID(entry.Path))
	}
	if len(entry.ID) != 8 {
		t.Errorf("entry.ID should be 8 hex chars, got %q", entry.ID)
	}
	if entry.Path != ws {
		t.Errorf("entry.Path: got %q want %q", entry.Path, ws)
	}
	if entry.AddedAt.IsZero() {
		t.Errorf("entry.AddedAt not set")
	}

	got, ok := r.Get(entry.ID)
	if !ok || got.Path != ws {
		t.Errorf("Get(%s): ok=%v path=%q", entry.ID, ok, got.Path)
	}
	if _, ok := r.Get("deadbeef"); ok {
		t.Errorf("Get(unknown) should miss")
	}
}

func TestRegistryWorkspaceAddIdempotent(t *testing.T) {
	path := registryPathUnder(t)
	r, _ := LoadWorkspaceRegistry(path)
	ws := newTestWorkspace(t)

	first, created, err := r.Add(ws, "")
	if err != nil {
		t.Fatalf("first Add: %v", err)
	}
	if !created {
		t.Errorf("first Add created=%v, want true", created)
	}
	if first.Name != filepath.Base(ws) {
		t.Errorf("empty name should default to dir base, got %q", first.Name)
	}

	second, created, err := r.Add(ws, "other-name")
	if err != nil {
		t.Fatalf("second Add: %v", err)
	}
	if created {
		t.Errorf("second Add created=true, want false (idempotent)")
	}
	if second.ID != first.ID || second.Name != first.Name {
		t.Errorf("second Add should return the existing entry verbatim, got %+v vs %+v", second, first)
	}
	if len(r.List()) != 1 {
		t.Errorf("idempotent Add must not append, list len=%d", len(r.List()))
	}
}

func TestRegistryWorkspaceAddInvalid(t *testing.T) {
	path := registryPathUnder(t)
	r, _ := LoadWorkspaceRegistry(path)

	plain := t.TempDir() // exists, but no .rick inside
	_, _, err := r.Add(plain, "")
	var ve *ValidationError
	if !errors.As(err, &ve) || ve.Code != "invalid_workspace" {
		t.Fatalf("Add on dir without .rick: want invalid_workspace ValidationError, got %v", err)
	}

	_, _, err = r.Add(filepath.Join(t.TempDir(), "nope"), "")
	if !errors.As(err, &ve) || ve.Code != "invalid_workspace" {
		t.Fatalf("Add on missing dir: want invalid_workspace ValidationError, got %v", err)
	}

	if len(r.List()) != 0 {
		t.Errorf("failed Adds must not register anything, len=%d", len(r.List()))
	}
}

func TestRegistryWorkspaceRemove(t *testing.T) {
	path := registryPathUnder(t)
	r, _ := LoadWorkspaceRegistry(path)
	ws := newTestWorkspace(t)
	entry, _, err := r.Add(ws, "")
	if err != nil {
		t.Fatalf("Add: %v", err)
	}
	if err := r.Remove(entry.ID); err != nil {
		t.Fatalf("Remove: %v", err)
	}
	if len(r.List()) != 0 {
		t.Errorf("after Remove list should be empty, len=%d", len(r.List()))
	}
	err = r.Remove(entry.ID)
	var ve *ValidationError
	if !errors.As(err, &ve) || ve.Code != "not_found" {
		t.Fatalf("Remove unknown id: want not_found, got %v", err)
	}
}

func TestRegistryWorkspaceJSONRoundTrip(t *testing.T) {
	path := registryPathUnder(t)
	r, _ := LoadWorkspaceRegistry(path)
	ws := newTestWorkspace(t)
	entry, _, err := r.Add(ws, "round-trip")
	if err != nil {
		t.Fatalf("Add: %v", err)
	}

	// Reload from disk and verify fields survive (timestamps included).
	r2, err := LoadWorkspaceRegistry(path)
	if err != nil {
		t.Fatalf("reload: %v", err)
	}
	got, ok := r2.Get(entry.ID)
	if !ok {
		t.Fatalf("reloaded registry lost workspace %s", entry.ID)
	}
	if got.Path != entry.Path || got.Name != entry.Name {
		t.Errorf("round trip mismatch: %+v vs %+v", got, entry)
	}
	if !got.AddedAt.Equal(entry.AddedAt) {
		t.Errorf("AddedAt not preserved: %v vs %v", got.AddedAt, entry.AddedAt)
	}

	// On-disk shape must match api-contract.md field names.
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read web.json: %v", err)
	}
	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("web.json is not valid JSON: %v", err)
	}
	if raw["version"] != float64(workspaceRegistryVersion) {
		t.Errorf("web.json version: got %v", raw["version"])
	}
	items, ok := raw["workspaces"].([]any)
	if !ok || len(items) != 1 {
		t.Fatalf("web.json workspaces: %v", raw["workspaces"])
	}
	item := items[0].(map[string]any)
	for _, key := range []string{"id", "path", "name", "added_at"} {
		if _, ok := item[key]; !ok {
			t.Errorf("web.json entry missing field %q (api-contract mismatch)", key)
		}
	}
	// added_at must carry a timezone (RFC3339 with offset — bugs.md job_35).
	if s, _ := item["added_at"].(string); !strings.Contains(s, "+") && !strings.Contains(s, "Z") {
		t.Errorf("added_at %q lacks timezone offset", s)
	}
}

func TestRegistryWorkspaceMalformedFile(t *testing.T) {
	path := registryPathUnder(t)
	if err := os.WriteFile(path, []byte("{not json"), 0644); err != nil {
		t.Fatalf("write malformed file: %v", err)
	}
	if _, err := LoadWorkspaceRegistry(path); err == nil {
		t.Errorf("malformed web.json should fail to load")
	}
}

// TestWorkspaceRegistry_AtomicWrite verifies a failed persist leaves no .tmp
// residue and no half-written registry. The failure is injected by pointing
// the registry at a path whose parent does not exist (write fails).
// Simulating a read-only directory is skipped on platforms where chmod is
// advisory (noted per task spec).
func TestRegistryWorkspaceAtomicWrite(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "web.json")
	r, _ := LoadWorkspaceRegistry(path)
	ws := newTestWorkspace(t)
	if _, _, err := r.Add(ws, "ok"); err != nil {
		t.Fatalf("Add: %v", err)
	}

	// Corrupt the backing location: replace the file with a directory so the
	// tmp write still succeeds but rename fails, exercising the rename-error
	// path and tmp cleanup.
	if err := os.Remove(path); err != nil {
		t.Fatalf("remove registry: %v", err)
	}
	if err := os.Mkdir(path, 0755); err != nil {
		t.Fatalf("replace with dir: %v", err)
	}
	err := r.Remove(r.List()[0].ID)
	if err == nil {
		t.Fatalf("expected rename onto a directory to fail")
	}
	// The failed save must clean up its tmp file.
	if _, err := os.Stat(path + ".tmp"); !os.IsNotExist(err) {
		t.Errorf("tmp residue left behind: %v", err)
	}
}

func TestRegistrySessionLifecycle(t *testing.T) {
	path := filepath.Join(t.TempDir(), "sessions.json")
	r, err := LoadSessionRegistry(path)
	if err != nil {
		t.Fatalf("LoadSessionRegistry: %v", err)
	}

	now := time.Now()
	a := SessionEntry{
		ID: "uuid-a", WorkspaceID: "ws1", Type: SessionTypePlan,
		Params: map[string]any{"requirement": "do it"}, Status: SessionStatusActive,
		PISessionID: "pi-uuid-a", CreatedAt: now,
	}
	b := SessionEntry{
		ID: "uuid-b", WorkspaceID: "ws2", Type: SessionTypeDoing,
		Params: map[string]any{"job": "job_3"}, Status: SessionStatusRunning,
		PISessionID: "pi-uuid-b", CreatedAt: now,
	}
	if err := r.Add(a); err != nil {
		t.Fatalf("Add a: %v", err)
	}
	if err := r.Add(b); err != nil {
		t.Fatalf("Add b: %v", err)
	}

	if got := r.GetByWorkspace("ws1"); len(got) != 1 || got[0].ID != "uuid-a" {
		t.Errorf("GetByWorkspace(ws1): %+v", got)
	}
	if got := r.List(); len(got) != 2 {
		t.Errorf("List len=%d", len(got))
	}
	if _, ok := r.Get("uuid-b"); !ok {
		t.Errorf("Get(uuid-b) missed")
	}

	// Update: running -> closed with ClosedAt set.
	closed := b
	closed.Status = SessionStatusClosed
	closed.ClosedAt = time.Now()
	if err := r.Update(closed); err != nil {
		t.Fatalf("Update: %v", err)
	}
	got, _ := r.Get("uuid-b")
	if got.Status != SessionStatusClosed || got.ClosedAt.IsZero() {
		t.Errorf("Update not persisted: %+v", got)
	}

	// Update unknown id -> not_found.
	err = r.Update(SessionEntry{ID: "nope"})
	var ve *ValidationError
	if !errors.As(err, &ve) || ve.Code != "not_found" {
		t.Fatalf("Update unknown: want not_found, got %v", err)
	}

	// Reload from disk: entries and statuses survive.
	r2, err := LoadSessionRegistry(path)
	if err != nil {
		t.Fatalf("reload: %v", err)
	}
	got, ok := r2.Get("uuid-b")
	if !ok || got.Status != SessionStatusClosed {
		t.Errorf("reload lost update: ok=%v %+v", ok, got)
	}
	// On-disk field names follow api-contract SessionInfo.
	data, _ := os.ReadFile(path)
	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("sessions.json invalid: %v", err)
	}
	items := raw["sessions"].([]any)
	first := items[0].(map[string]any)
	for _, key := range []string{"id", "workspace_id", "type", "params", "status", "pi_session_id", "created_at"} {
		if _, ok := first[key]; !ok {
			t.Errorf("sessions.json entry missing %q", key)
		}
	}
	// closed_at is omitempty by contract ("closed_at"?: ...) — a zero value
	// must be absent, not serialized as 0001-01-01 (omitzero, Go 1.25).
	if _, ok := first["closed_at"]; ok {
		t.Errorf("closed_at must be absent for a never-closed session, got %v", first["closed_at"])
	}
}

func TestRegistryValidateSessionMatrix(t *testing.T) {
	cases := []struct {
		name    string
		typ     string
		params  map[string]any
		wantErr string // "" = valid; otherwise a substring of the expected error
	}{
		// plan
		{"plan ok", SessionTypePlan, map[string]any{"requirement": "r"}, ""},
		{"plan ok with job", SessionTypePlan, map[string]any{"requirement": "r", "job": "job_3"}, ""},
		{"plan missing requirement", SessionTypePlan, map[string]any{}, "requirement"},
		{"plan empty requirement", SessionTypePlan, map[string]any{"requirement": ""}, "requirement"},
		{"plan non-string requirement", SessionTypePlan, map[string]any{"requirement": 42}, "requirement"},
		{"plan non-string job", SessionTypePlan, map[string]any{"requirement": "r", "job": 7}, "job"},
		// easy
		{"easy ok", SessionTypeEasy, map[string]any{"requirement": "r"}, ""},
		{"easy ok ctx", SessionTypeEasy, map[string]any{"requirement": "r", "ctx_path": "/x"}, ""},
		{"easy missing requirement", SessionTypeEasy, map[string]any{}, "requirement"},
		{"easy bad ctx", SessionTypeEasy, map[string]any{"requirement": "r", "ctx_path": 1}, "ctx_path"},
		// ctrl / learning / doing
		{"ctrl ok", SessionTypeCtrl, map[string]any{"job": "job_1"}, ""},
		{"ctrl missing job", SessionTypeCtrl, map[string]any{}, "job"},
		{"learning ok", SessionTypeLearning, map[string]any{"job": "job_2"}, ""},
		{"learning empty job", SessionTypeLearning, map[string]any{"job": ""}, "job"},
		{"doing ok", SessionTypeDoing, map[string]any{"job": "job_9"}, ""},
		{"doing missing job", SessionTypeDoing, map[string]any{}, "job"},
		// human-loop
		{"human-loop ok", SessionTypeHumanLoop, map[string]any{"topic": "t"}, ""},
		{"human-loop missing topic", SessionTypeHumanLoop, map[string]any{}, "topic"},
		// dream
		{"dream ok defaults", SessionTypeDream, map[string]any{}, ""},
		{"dream ok interactive", SessionTypeDream, map[string]any{"job_num": 3, "mode": "interactive"}, ""},
		{"dream ok background", SessionTypeDream, map[string]any{"job_num": 5, "mode": "background"}, ""},
		{"dream zero job_num", SessionTypeDream, map[string]any{"job_num": 0}, "job_num"},
		{"dream negative job_num", SessionTypeDream, map[string]any{"job_num": -2}, "job_num"},
		{"dream fractional job_num", SessionTypeDream, map[string]any{"job_num": 2.5}, "job_num"},
		{"dream string job_num", SessionTypeDream, map[string]any{"job_num": "five"}, "job_num"},
		{"dream bad mode", SessionTypeDream, map[string]any{"mode": "both"}, "mode"},
		{"dream non-string mode", SessionTypeDream, map[string]any{"mode": 1}, "mode"},
		// unknown type
		{"unknown type", "tdd", map[string]any{}, "unknown session type"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateSessionRequest(tc.typ, tc.params)
			if tc.wantErr == "" {
				if err != nil {
					t.Fatalf("expected valid, got %v", err)
				}
				return
			}
			if err == nil {
				t.Fatalf("expected error containing %q, got nil", tc.wantErr)
			}
			var ve *ValidationError
			if !errors.As(err, &ve) {
				t.Fatalf("error is not *ValidationError: %v", err)
			}
			if ve.Code != "invalid_params" {
				t.Errorf("code: got %q want invalid_params", ve.Code)
			}
			if !strings.Contains(err.Error(), tc.wantErr) {
				t.Errorf("error %q does not mention %q", err.Error(), tc.wantErr)
			}
		})
	}
}

func TestRegistryValidatePure(t *testing.T) {
	// Validation is pure: it must not touch the filesystem, so HOME must not
	// matter. Guards against accidental path resolution sneaking in.
	t.Setenv("HOME", t.TempDir())
	if err := ValidateSessionRequest(SessionTypePlan, map[string]any{"requirement": "x"}); err != nil {
		t.Fatalf("pure validation failed: %v", err)
	}
}

// TestManualSmoke is the task spec's 手工 smoke (临时 HOME 下跑一轮
// Add/List/落盘检查 JSON 内容)，kept as a skipped-by-default test so it can
// be run on demand with -run TestManualSmoke -v (and via SMOKE=1 env).
func TestManualSmoke(t *testing.T) {
	if os.Getenv("SMOKE") == "" {
		t.Skip("manual smoke; run with SMOKE=1 go test ./internal/web/ -run TestManualSmoke -v")
	}
	home := t.TempDir()
	t.Setenv("HOME", home)

	ws1 := filepath.Join(home, "ws1")
	ws2 := filepath.Join(home, "ws2")
	os.MkdirAll(filepath.Join(ws1, ".rick"), 0755)
	os.MkdirAll(filepath.Join(ws2, ".rick"), 0755)

	fmt.Println("WebConfigPath:", WebConfigPath())
	fmt.Println("SessionsPath: ", SessionsPath())
	fmt.Println("PidPath:      ", PidPath())

	r, err := LoadWorkspaceRegistry(WebConfigPath())
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	e1, created, err := r.Add(ws1, "first")
	fmt.Printf("Add ws1: id=%s name=%s created=%v err=%v\n", e1.ID, e1.Name, created, err)
	e2, created, err := r.Add(ws2, "")
	fmt.Printf("Add ws2: id=%s name=%s created=%v err=%v\n", e2.ID, e2.Name, created, err)
	_, created, err = r.Add(ws1, "ignored")
	fmt.Printf("Re-add ws1: created=%v err=%v (expect false/nil)\n", created, err)
	os.MkdirAll(filepath.Join(home, "notaws"), 0755)
	_, _, err = r.Add(filepath.Join(home, "notaws"), "")
	fmt.Printf("Add invalid: err=%v (expect invalid_workspace)\n", err)

	sr, err := LoadSessionRegistry(SessionsPath())
	if err != nil {
		t.Fatalf("load sessions: %v", err)
	}
	err = sr.Add(SessionEntry{
		ID: "uuid-1", WorkspaceID: e1.ID, Type: SessionTypeDream,
		Params: map[string]any{"job_num": 5, "mode": "background"},
		Status: SessionStatusRunning, PISessionID: "pi-1",
	})
	fmt.Println("Session Add err:", err)

	data, _ := os.ReadFile(WebConfigPath())
	fmt.Println("---- web.json ----")
	fmt.Println(string(data))
	data, _ = os.ReadFile(SessionsPath())
	fmt.Println("---- sessions.json ----")
	fmt.Println(string(data))
}

// TestRegistryWorkspaceReorder —— 拖拽排序（sidebar drag-sort）持久化：
// 按给定 id 顺序重排、空集合 no-op、集合不匹配/未知 id/重复 id 报 invalid_order
// 且顺序不变、save 失败回滚。
func TestRegistryWorkspaceReorder(t *testing.T) {
	path := registryPathUnder(t)
	r, err := LoadWorkspaceRegistry(path)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	ws1, ws2, ws3 := newTestWorkspace(t), newTestWorkspace(t), newTestWorkspace(t)
	e1, _, err := r.Add(ws1, "alpha")
	if err != nil {
		t.Fatalf("add ws1: %v", err)
	}
	e2, _, err := r.Add(ws2, "beta")
	if err != nil {
		t.Fatalf("add ws2: %v", err)
	}
	e3, _, err := r.Add(ws3, "gamma")
	if err != nil {
		t.Fatalf("add ws3: %v", err)
	}

	// 1) 空集合：no-op，顺序不变
	if err := r.Reorder(nil); err != nil {
		t.Fatalf("reorder empty: %v", err)
	}
	want := []string{e1.ID, e2.ID, e3.ID}
	if got := idsOf(r.List()); !equalStrings(got, want) {
		t.Fatalf("reorder empty changed order: got %v want %v", got, want)
	}

	// 2) 合法重排：[e3, e1, e2] → 顺序变为该序，且持久化（重载验证）
	if err := r.Reorder([]string{e3.ID, e1.ID, e2.ID}); err != nil {
		t.Fatalf("reorder: %v", err)
	}
	want = []string{e3.ID, e1.ID, e2.ID}
	if got := idsOf(r.List()); !equalStrings(got, want) {
		t.Fatalf("reorder order: got %v want %v", got, want)
	}
	reloaded, err := LoadWorkspaceRegistry(path)
	if err != nil {
		t.Fatalf("reload: %v", err)
	}
	if got := idsOf(reloaded.List()); !equalStrings(got, want) {
		t.Fatalf("reorder persisted: got %v want %v", got, want)
	}

	// 3) 集合不匹配（多一个 id）→ invalid_order，顺序不变
	before := idsOf(r.List())
	if err := r.Reorder([]string{e3.ID, e1.ID, e2.ID, "extra-1"}); err == nil {
		t.Fatal("reorder with extra id: expected error")
	}
	if got := idsOf(r.List()); !equalStrings(got, before) {
		t.Fatalf("reorder failed but order changed: got %v", got)
	}

	// 4) 未知 id → invalid_order
	if err := r.Reorder([]string{e3.ID, "unknown-9", e1.ID}); err == nil {
		t.Fatal("reorder with unknown id: expected error")
	}

	// 5) 重复 id → invalid_order
	if err := r.Reorder([]string{e3.ID, e3.ID, e1.ID}); err == nil {
		t.Fatal("reorder with duplicate id: expected error")
	}
}

// idsOf extracts the ordered id slice from workspace entries.
func idsOf(entries []WorkspaceEntry) []string {
	out := make([]string, 0, len(entries))
	for _, e := range entries {
		out = append(out, e.ID)
	}
	return out
}

// equalStrings reports whether two string slices are element-wise equal.
func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// TestRoutes_ReorderWorkspacesHTTP —— 路由层回归锁：PUT /api/workspaces/order
// 经 RegisterRoutes 正确路由到 Reorder（合法 204 / 非法 400 / 无 token 401）。
func TestRoutes_ReorderWorkspacesHTTP(t *testing.T) {
	env := newTestEnv(t)
	mux := http.NewServeMux()
	RegisterRoutes(mux, Deps{
		Hub:        env.hub,
		Sessions:   env.managerWith(t, nil, nil),
		Workspaces: env.workspaces,
		Token:      "test-token",
		Static:     http.NotFoundHandler(),
		Version:    "test",
	})

	ws1 := newTestWorkspace(t)
	e1, _, err := env.workspaces.Add(ws1, "alpha")
	if err != nil {
		t.Fatalf("add ws1: %v", err)
	}
	ws2 := newTestWorkspace(t)
	e2, _, err := env.workspaces.Add(ws2, "beta")
	if err != nil {
		t.Fatalf("add ws2: %v", err)
	}
	// newTestEnv 自带一个已注册工作区（env.wsEntry）——三者构成完整集合。
	base := env.wsEntry.ID
	all := []string{e2.ID, e1.ID, base}

	// 无 token → 401
	req := httptest.NewRequest(http.MethodPut, "/api/workspaces/order",
		strings.NewReader(`{"ids":["`+e2.ID+`","`+e1.ID+`"]}`))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("PUT order without token: got %d, want 401", rec.Code)
	}

	// 合法重排（完整集合 [e2, e1, base]）→ 204 + 顺序变更
	body, _ := json.Marshal(map[string]any{"ids": all})
	req = httptest.NewRequest(http.MethodPut, "/api/workspaces/order",
		strings.NewReader(string(body)))
	req.Header.Set("Authorization", "Bearer test-token")
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("PUT order: got %d, want 204, body=%s", rec.Code, rec.Body.String())
	}
	if got := idsOf(env.workspaces.List()); !equalStrings(got, all) {
		t.Fatalf("reorder via HTTP: got %v want %v", got, all)
	}

	// 非法（未知 id 且缺 base）→ 400
	req = httptest.NewRequest(http.MethodPut, "/api/workspaces/order",
		strings.NewReader(`{"ids":["`+e2.ID+`","bogus"]}`))
	req.Header.Set("Authorization", "Bearer test-token")
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("PUT order invalid: got %d, want 400", rec.Code)
	}
}
