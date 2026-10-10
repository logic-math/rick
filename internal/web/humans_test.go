package web

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

// humansFixture builds <ws>/.rick/draft with a loop_N tree + rfc history
// (gated by gate2: loops/loop_N/{mission.md,judgment.md,research/,lessons/}
// and rfc/*.md must show up in the tree).
func humansFixture(t *testing.T) string {
	t.Helper()
	ws := t.TempDir()
	draft := filepath.Join(ws, ".rick", "draft")
	writeFileT(t, filepath.Join(draft, "rfc", "rfc-2026-01.md"), "# RFC 1")
	writeFileT(t, filepath.Join(draft, "loops", "loop_1", "mission.md"), "# 学习目标")
	writeFileT(t, filepath.Join(draft, "loops", "loop_1", "judgment.md"), "# 判断时间线")
	writeFileT(t, filepath.Join(draft, "loops", "loop_1", "research", "research_1.md"), "# 事实材料")
	writeFileT(t, filepath.Join(draft, "loops", "loop_1", "lessons", "teach_1.md"), "# 教学节 1")
	writeFileT(t, filepath.Join(draft, "loops", "loop_1", "prompts", "sense_loop.md"), "# prompt")
	writeFileT(t, filepath.Join(draft, "loops", "loop_1", "session_id"), "abc-123")
	writeFileT(t, filepath.Join(draft, "loops", "loop_2", "mission.md"), "# 第二个 loop")
	// 非文本扩展名不入树（与 knowledge 树一致）。
	writeFileT(t, filepath.Join(draft, "loops", "loop_1", "raw.png"), "\x89PNG fake binary")
	return ws
}

func TestHumansTree(t *testing.T) {
	ws := humansFixture(t)
	nodes, err := HumansTree(ws)
	if err != nil {
		t.Fatalf("HumansTree: %v", err)
	}
	var paths []string
	for _, n := range nodes {
		paths = append(paths, n.Path)
	}
	// 字典序、draft 根相对；session_id（无扩展名）与 raw.png（二进制）被过滤。
	want := []string{
		"loops/loop_1/judgment.md",
		"loops/loop_1/lessons/teach_1.md",
		"loops/loop_1/mission.md",
		"loops/loop_1/prompts/sense_loop.md",
		"loops/loop_1/research/research_1.md",
		"loops/loop_2/mission.md",
		"rfc/rfc-2026-01.md",
	}
	if len(paths) != len(want) {
		t.Fatalf("tree mismatch:\nwant %v\ngot  %v", want, paths)
	}
	for i := range want {
		if paths[i] != want[i] {
			t.Fatalf("tree[%d]: want %s, got %s (not sorted or extra entry)", i, want[i], paths[i])
		}
	}
	for _, n := range nodes {
		if n.Size <= 0 {
			t.Fatalf("node %s has non-positive size %d", n.Path, n.Size)
		}
	}

	// 符号链接文件永不列出（与 KnowledgeTree 一致）。
	outside := filepath.Join(t.TempDir(), "outside.md")
	if err := os.WriteFile(outside, []byte("# outside"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(ws, ".rick", "draft", "loops", "loop_1", "linked.md")); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}
	nodes, err = HumansTree(ws)
	if err != nil {
		t.Fatal(err)
	}
	for _, n := range nodes {
		if n.Path == "loops/loop_1/linked.md" {
			t.Fatal("symlinked draft file must not be listed")
		}
	}
}

// TestHumansTreeMissingDraft：全新工作区无 .rick/draft → 空树且不报错（KR2d）。
func TestHumansTreeMissingDraft(t *testing.T) {
	// 完全没有 .rick 的目录。
	nodes, err := HumansTree(t.TempDir())
	if err != nil {
		t.Fatalf("HumansTree(missing draft) error: %v", err)
	}
	if len(nodes) != 0 {
		t.Fatalf("want empty tree, got %v", nodes)
	}
	// 有 .rick 但无 draft。
	ws := t.TempDir()
	writeFileT(t, filepath.Join(ws, ".rick", "domain", "x.md"), "# x")
	nodes, err = HumansTree(ws)
	if err != nil {
		t.Fatalf("HumansTree(.rick without draft) error: %v", err)
	}
	if len(nodes) != 0 {
		t.Fatalf("want empty tree, got %v", nodes)
	}
}

func TestHumansReadFile(t *testing.T) {
	ws := humansFixture(t)

	got, err := ReadHumansFile(ws, "loops/loop_1/judgment.md")
	if err != nil {
		t.Fatalf("ReadHumansFile: %v", err)
	}
	if got != "# 判断时间线" {
		t.Fatalf("content = %q", got)
	}

	bad := []struct {
		path   string
		status int
		code   string
	}{
		{"loops/loop_1", 404, "not_found"},  // 目录路径拒绝（仅文件）
		{"loops/loop_1/", 404, "not_found"}, // 目录尾斜杠
		{"loops/loop_1/missing.md", 404, "not_found"},
		{"", 400, "invalid_params"},                      // 空路径
		{"loops/loop_1/\x00bad.md", 400, "invalid_path"}, // NUL 拒绝
	}
	for _, c := range bad {
		_, err := ReadHumansFile(ws, c.path)
		assertWebError(t, err, c.status, c.code)
	}

	// 超限 → file_too_large（400）。
	big := filepath.Join(ws, ".rick", "draft", "loops", "loop_1", "big.md")
	writeFileT(t, big, "")
	if err := os.WriteFile(big, bytes.Repeat([]byte("x"), MaxReadFileSize+1), 0644); err != nil {
		t.Fatal(err)
	}
	_, err = ReadHumansFile(ws, "loops/loop_1/big.md")
	assertWebError(t, err, 400, "file_too_large")

	// 绝对路径直接读（job_43：不做白名单根校验，不引入更严限制）。
	outside := filepath.Join(t.TempDir(), "note.md")
	if err := os.WriteFile(outside, []byte("outside"), 0644); err != nil {
		t.Fatal(err)
	}
	content, err := ReadHumansFile(ws, outside)
	if err != nil {
		t.Fatalf("ReadHumansFile(absolute): %v", err)
	}
	if content != "outside" {
		t.Fatalf("absolute content = %q", content)
	}
}

// TestHumansRoutes 走完整 mux（authWrap + {ws} 注册）验证 KR2b 路由契约。
func TestHumansRoutes(t *testing.T) {
	wsPath := humansFixture(t)
	reg, err := LoadWorkspaceRegistry(filepath.Join(t.TempDir(), "web.json"))
	if err != nil {
		t.Fatalf("registry: %v", err)
	}
	ws, _, err := reg.Add(wsPath, "ws")
	if err != nil {
		t.Fatalf("add workspace: %v", err)
	}
	mux := http.NewServeMux()
	RegisterRoutes(mux, Deps{Token: "test-token", Workspaces: reg})

	do := func(target string, withToken bool) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodGet, target, nil)
		if withToken {
			req.Header.Set("Authorization", "Bearer test-token")
		}
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)
		return rec
	}

	// 无 token → 401（authWrap 与 knowledge 一致）。
	if rec := do("/api/workspaces/"+ws.ID+"/humans/tree", false); rec.Code != http.StatusUnauthorized {
		t.Fatalf("no-token tree status = %d", rec.Code)
	}

	// tree → 200 {"tree":[...]}。
	rec := do("/api/workspaces/"+ws.ID+"/humans/tree", true)
	if rec.Code != http.StatusOK {
		t.Fatalf("tree status = %d (%s)", rec.Code, rec.Body.String())
	}
	var tree treeResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &tree); err != nil {
		t.Fatalf("decode tree: %v", err)
	}
	if len(tree.Tree) != 7 {
		t.Fatalf("tree len = %d: %+v", len(tree.Tree), tree.Tree)
	}

	// file → 200 {"path","content"}。
	rec = do("/api/workspaces/"+ws.ID+"/humans/file?path=loops/loop_1/mission.md", true)
	if rec.Code != http.StatusOK {
		t.Fatalf("file status = %d (%s)", rec.Code, rec.Body.String())
	}
	var body fileResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode file: %v", err)
	}
	if body.Path != "loops/loop_1/mission.md" || body.Content != "# 学习目标" {
		t.Fatalf("file body = %+v", body)
	}

	// 目录路径 → 404。
	if rec := do("/api/workspaces/"+ws.ID+"/humans/file?path=loops/loop_1", true); rec.Code != http.StatusNotFound {
		t.Fatalf("directory path status = %d, want 404", rec.Code)
	}

	// 未注册工作区 → 404。
	if rec := do("/api/workspaces/nope/humans/tree", true); rec.Code != http.StatusNotFound {
		t.Fatalf("unknown workspace status = %d, want 404", rec.Code)
	}
}
