package web

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// newWorkspaceFileEnv builds a workspace with a file at
// .rick/jobs/job_1/doing/requirement.md plus files OUTSIDE the workspace
//（同目录 secret.md），and returns a Deps wired for handleWorkspaceFile.
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
	// 工作区之外（同级目录）：去白名单后**可读**（human 裁决 2026-10-01）
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

	// 绝对路径（工作区内）→ display 为相对路径
	rec = get(ws.Path + "%2Fdebug%2Fnotes.md")
	if rec.Code != http.StatusOK {
		t.Fatalf("absolute in-workspace path status = %d", rec.Code)
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode abs-in-ws: %v", err)
	}
	if body.Path != "debug/notes.md" {
		t.Fatalf("in-ws display = %q, want debug/notes.md", body.Path)
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

	// ---- 去白名单（human 裁决）：任意路径直接读 ----
	// 工作区之外的绝对路径 → 200（KR14a 核心语义）
	rec = get(outside)
	if rec.Code != http.StatusOK {
		t.Fatalf("outside absolute path status = %d, want 200; body=%s", rec.Code, rec.Body.String())
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode outside: %v", err)
	}
	if body.Content != "secret" || body.Path != outside {
		t.Fatalf("outside body = %+v", body)
	}

	// 目录穿越到工作区外 → 200（解析后是合法文件路径）
	if rec := get("%2E%2E%2Fsecret.md"); rec.Code != http.StatusOK {
		t.Fatalf("traversal status = %d, want 200; body=%s", rec.Code, rec.Body.String())
	}

	// 系统文件绝对路径（/etc/hostname 在 Linux CI 环境恒存在且小）→ 200
	if _, err := os.Stat("/etc/hostname"); err == nil {
		if rec := get("%2Fetc%2Fhostname"); rec.Code != http.StatusOK {
			t.Fatalf("/etc/hostname status = %d, want 200; body=%s", rec.Code, rec.Body.String())
		}
	} else {
		t.Logf("/etc/hostname 不存在（非 Linux？），跳过该断言: %v", err)
	}

	// 目录穿越到工作区外 → 200（解析后是合法文件路径；「../../secret.md」
	// 之类的任意穿越不再受根判定限制——同上，无白名单语义）

	// 符号链接 → 跟随放行（KR14b：读链接目标，不再拒绝）
	link := filepath.Join(ws.Path, "debug", "link.md")
	if err := os.Symlink(outside, link); err == nil {
		if rec := get("debug%2Flink.md"); rec.Code != http.StatusOK {
			t.Fatalf("symlink status = %d, want 200", rec.Code)
		}
	} else {
		t.Logf("symlink 创建失败（无权限？），跳过: %v", err)
	}

	// ---- 保留的基本防御（KR14b）----
	// 目录不是文件 → 404
	if rec := get(".rick%2Fjobs%2Fjob_1%2Fdoing"); rec.Code != http.StatusNotFound {
		t.Fatalf("directory status = %d, want 404", rec.Code)
	}
	// 缺失 → 404
	if rec := get("debug%2Fnope.md"); rec.Code != http.StatusNotFound {
		t.Fatalf("missing file status = %d, want 404", rec.Code)
	}
	// 空路径 → 400
	if rec := get(""); rec.Code != http.StatusBadRequest {
		t.Fatalf("empty path status = %d, want 400", rec.Code)
	}
	// 超大文件 → 400
	big := filepath.Join(t.TempDir(), "big.txt")
	if err := os.WriteFile(big, make([]byte, MaxReadFileSize+1), 0o644); err != nil {
		t.Fatalf("write big: %v", err)
	}
	if rec := get(big); rec.Code != http.StatusBadRequest {
		t.Fatalf("oversized status = %d, want 400; body=%s", rec.Code, rec.Body.String())
	}
	// NUL → 400
	if rec := get("debug%00notes.md"); rec.Code != http.StatusBadRequest {
		t.Fatalf("NUL path status = %d, want 400", rec.Code)
	}
}

// TestResolveWorkspaceFilePath 直接覆盖路径解析（无需 HTTP）：file:// 前缀、
// 锚点/查询剥离、绝对/相对（相对按 workspace 根解析）、NUL 拒绝——KR14a。
func TestResolveWorkspaceFilePath(t *testing.T) {
	wsPath := "/workdir/example/WS"

	cases := []struct {
		in   string
		want string
		err  bool
	}{
		{in: ".rick/jobs/job_1/plan/task1.md", want: "/workdir/example/WS/.rick/jobs/job_1/plan/task1.md"},
		{in: "debug/notes.md", want: "/workdir/example/WS/debug/notes.md"},
		{in: "/workdir/example/WS/debug/notes.md", want: "/workdir/example/WS/debug/notes.md"},
		{in: "file:///workdir/example/WS/debug/notes.md", want: "/workdir/example/WS/debug/notes.md"},
		{in: "debug/notes.md#L10", want: "/workdir/example/WS/debug/notes.md"},
		{in: "debug/notes.md?raw=1", want: "/workdir/example/WS/debug/notes.md"},
		{in: "/etc/hostname", want: "/etc/hostname"},               // 任意绝对路径：直接通过
		{in: "../../etc/passwd", want: "/workdir/etc/passwd"},      // 相对穿越：解析为 ws 祖先下的路径
		{in: "  ", err: true},                              // 空白
		{in: "debug/notes.md\x00tail", err: true},          // NUL
	}
	for _, tc := range cases {
		got, err := resolveWorkspaceFilePath(wsPath, tc.in)
		if tc.err {
			if err == nil {
				t.Errorf("%q: expected error, got abs=%q", tc.in, got)
			}
			continue
		}
		if err != nil {
			t.Errorf("%q: unexpected error %v", tc.in, err)
			continue
		}
		if got != tc.want {
			t.Errorf("%q: abs = %q, want %q", tc.in, got, tc.want)
		}
	}
}

// TestWorkspaceFileNoWhitelist 端到端断言去白名单语义（KR14a）：未注册的
// 任意目录路径可读——曾经的 TestWorkspaceFileCrossWorkspaceRoots 白名单
// 三态（注册 ws 200 / 未注册 400 / ExtraFileRoots 200）已被 human 裁决整体
// 否决，本测试只剩「都能读」。
func TestWorkspaceFileNoWhitelist(t *testing.T) {
	deps, ws, _ := newWorkspaceFileEnv(t)

	// 未注册的独立目录树（模拟 MOGIC_HOME / 任意位置）
	other := t.TempDir()
	otherMD := filepath.Join(other, "list.md")
	if err := os.WriteFile(otherMD, []byte("跨工作区内容"), 0o644); err != nil {
		t.Fatalf("write other md: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/workspaces/"+ws.ID+"/file?path="+otherMD, nil)
	req.SetPathValue("ws", ws.ID)
	rec := httptest.NewRecorder()
	deps.handleWorkspaceFile()(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("unregistered path status = %d, want 200; body=%s", rec.Code, rec.Body.String())
	}
	var body fileResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body.Content != "跨工作区内容" {
		t.Fatalf("content = %q", body.Content)
	}
	if body.Path != otherMD {
		t.Fatalf("display = %q, want absolute %q（工作区外原样回显）", body.Path, otherMD)
	}
}

// TestReadFileBasicSymlinkFollowed：symlink 指向的文件可读（KR14b——
// os.Stat 跟随链接；曾经的 Lstat 逐组件拒绝已随白名单移除）。
func TestReadFileBasicSymlinkFollowed(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "target.txt")
	if err := os.WriteFile(target, []byte("via-link"), 0o644); err != nil {
		t.Fatalf("write target: %v", err)
	}
	link := filepath.Join(dir, "link.txt")
	if err := os.Symlink(target, link); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}
	got, err := readFileBasic(link)
	if err != nil {
		t.Fatalf("readFileBasic(link) error: %v", err)
	}
	if got != "via-link" {
		t.Fatalf("content = %q", got)
	}
	// 目录仍拒绝
	if _, err := readFileBasic(dir); err == nil || !strings.Contains(err.Error(), "not a regular file") {
		t.Fatalf("readFileBasic(dir) should reject, got %v", err)
	}
}
