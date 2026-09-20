package web

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

// newWorkspaceFileEnv builds a workspace with a file at
// .rick/jobs/job_1/doing/requirement.md plus a file OUTSIDE the workspace, and
// returns a Deps wired for handleWorkspaceFile.
func newWorkspaceFileEnv(t *testing.T) (Deps, WorkspaceEntry, string) {
	t.Helper()
	root := t.TempDir()
	wsPath := filepath.Join(root, "ws")
	jobDoing := filepath.Join(wsPath, ".rick", "jobs", "job_1", "doing")
	if err := os.MkdirAll(jobDoing, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(jobDoing, "requirement.md"), []byte("# 需求\n做一个 env\n"), 0o644); err != nil {
		t.Fatalf("write md: %v", err)
	}
	// 工作区根的相对路径文件（`debug/notes.md` 这类聊天里常见的写法）
	if err := os.MkdirAll(filepath.Join(wsPath, "debug"), 0o755); err != nil {
		t.Fatalf("mkdir debug: %v", err)
	}
	if err := os.WriteFile(filepath.Join(wsPath, "debug", "notes.md"), []byte("notes"), 0o644); err != nil {
		t.Fatalf("write notes: %v", err)
	}
	// 工作区之外（同级目录）：必须拒绝
	outside := filepath.Join(root, "secret.md")
	if err := os.WriteFile(outside, []byte("secret"), 0o644); err != nil {
		t.Fatalf("write outside: %v", err)
	}

	reg, err := LoadWorkspaceRegistry(filepath.Join(root, "web.json"))
	if err != nil {
		t.Fatalf("load registry: %v", err)
	}
	ws, _, err := reg.Add(wsPath, "ws")
	if err != nil {
		t.Fatalf("add workspace: %v", err)
	}
	return Deps{Workspaces: reg}, ws, outside
}

func TestWorkspaceFileRead(t *testing.T) {
	deps, ws, outside := newWorkspaceFileEnv(t)

	get := func(path string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodGet, "/api/workspaces/"+ws.ID+"/file?path="+path, nil)
		req.SetPathValue("ws", ws.ID)
		rec := httptest.NewRecorder()
		deps.handleWorkspaceFile()(rec, req)
		return rec
	}

	// 工作区相对路径（.rick 下）
	rec := get(".rick%2Fjobs%2Fjob_1%2Fdoing%2Frequirement.md")
	if rec.Code != http.StatusOK {
		t.Fatalf("relative .rick path status = %d (%s)", rec.Code, rec.Body.String())
	}
	var body fileResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body.Content == "" || body.Path != ".rick/jobs/job_1/doing/requirement.md" {
		t.Fatalf("body = %+v", body)
	}

	// 工作区根的相对路径（`debug/notes.md`）
	if rec := get("debug%2Fnotes.md"); rec.Code != http.StatusOK {
		t.Fatalf("relative root path status = %d (%s)", rec.Code, rec.Body.String())
	}

	// 绝对路径（工作区内）
	if rec := get(ws.Path + "%2Fdebug%2Fnotes.md"); rec.Code != http.StatusOK {
		t.Fatalf("absolute in-workspace path status = %d", rec.Code)
	}

	// file:// URL（阅读器内点击 file: 链接）—— 用绝对路径构造
	fileURL := "file%3A%2F%2F" + ws.Path + "%2Fdebug%2Fnotes.md"
	if rec := get(fileURL); rec.Code != http.StatusOK {
		t.Fatalf("file:// URL status = %d (%s)", rec.Code, rec.Body.String())
	}

	// 锚点/行号后缀被剥掉（`x.md#L3`、`x.md?raw=1`）
	if rec := get(".rick%2Fjobs%2Fjob_1%2Fdoing%2Frequirement.md%23L3"); rec.Code != http.StatusOK {
		t.Fatalf("anchor suffix status = %d (%s)", rec.Code, rec.Body.String())
	}
	if rec := get("debug%2Fnotes.md%3Fraw%3D1"); rec.Code != http.StatusOK {
		t.Fatalf("query suffix status = %d", rec.Code)
	}

	// ---- 安全边界 ----
	// 绝对路径在工作区之外 → invalid_path（这是本端点的核心防线）
	if rec := get(outside); rec.Code != http.StatusBadRequest {
		t.Fatalf("outside absolute path status = %d, want 400; body=%s", rec.Code, rec.Body.String())
	}
	// 目录穿越
	if rec := get("%2E%2E%2Fsecret.md"); rec.Code != http.StatusBadRequest {
		t.Fatalf("traversal status = %d, want 400", rec.Code)
	}
	if rec := get("%2Fetc%2Fpasswd"); rec.Code != http.StatusBadRequest {
		t.Fatalf("absolute /etc/passwd status = %d, want 400", rec.Code)
	}
	// 目录不是文件
	if rec := get(".rick%2Fjobs%2Fjob_1%2Fdoing"); rec.Code != http.StatusNotFound {
		t.Fatalf("directory status = %d, want 404", rec.Code)
	}
	// 缺失
	if rec := get("debug%2Fnope.md"); rec.Code != http.StatusNotFound {
		t.Fatalf("missing file status = %d, want 404", rec.Code)
	}
	// 空路径
	if rec := get(""); rec.Code != http.StatusBadRequest {
		t.Fatalf("empty path status = %d, want 400", rec.Code)
	}
	// 符号链接（HTTP 暴露面：拒绝）
	link := filepath.Join(ws.Path, "debug", "link.md")
	if err := os.Symlink(outside, link); err == nil {
		if rec := get("debug%2Flink.md"); rec.Code != http.StatusBadRequest {
			t.Fatalf("symlink status = %d, want 400", rec.Code)
		}
	}
}

// TestWorkspaceRelativeFileResolution 直接覆盖路径解析（无需 HTTP）：
// file:// 前缀、锚点剥离、绝对/相对、允许根（工作区 + ~/.rick）与拒绝分支。
func TestWorkspaceRelativeFileResolution(t *testing.T) {
	wsPath := "/workdir/example/WS"
	home, _ := os.UserHomeDir()

	cases := []struct {
		in      string
		wantRel string
		wantErr bool
	}{
		{in: ".rick/jobs/job_1/plan/task1.md", wantRel: ".rick/jobs/job_1/plan/task1.md"},
		{in: "debug/notes.md", wantRel: "debug/notes.md"},
		{in: "/workdir/example/WS/debug/notes.md", wantRel: "debug/notes.md"},
		{in: "file:///workdir/example/WS/debug/notes.md", wantRel: "debug/notes.md"},
		{in: "debug/notes.md#L10", wantRel: "debug/notes.md"},
		{in: "debug/notes.md?raw=1", wantRel: "debug/notes.md"},
		{in: filepath.Join(home, ".rick", "web", "sessions.json"), wantRel: "web/sessions.json"},
		{in: "../../etc/passwd", wantErr: true},
		{in: "/etc/passwd", wantErr: true},
		{in: "/workdir/example/WS", wantErr: true}, // 工作区根自身不是文件
		{in: "  ", wantErr: true},
	}
	for _, tc := range cases {
		got, err := workspaceRelativeFile(wsPath, tc.in)
		if tc.wantErr {
			if err == nil {
				t.Errorf("%q: expected error, got rel=%q", tc.in, got.rel)
			}
			continue
		}
		if err != nil {
			t.Errorf("%q: unexpected error %v", tc.in, err)
			continue
		}
		if got.rel != tc.wantRel {
			t.Errorf("%q: rel = %q, want %q", tc.in, got.rel, tc.wantRel)
		}
	}
}
