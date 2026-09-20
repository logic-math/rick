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

// newJobNameStoreEnv builds a throwaway store on disk plus a minimal workspace
// tree containing job_1/job_2 (so the rename handler's existence check passes).
func newJobNameStoreEnv(t *testing.T) (*JobNameStore, string, WorkspaceEntry) {
	t.Helper()
	root := t.TempDir()
	wsPath := filepath.Join(root, "ws")
	for _, job := range []string{"job_1", "job_2"} {
		doing := filepath.Join(wsPath, ".rick", "jobs", job, "doing")
		if err := os.MkdirAll(doing, 0o755); err != nil {
			t.Fatalf("mkdir job: %v", err)
		}
		// doing/session_id 让 ListJobs 能发现该 job（stage=started）
		if err := os.WriteFile(filepath.Join(doing, "session_id"), []byte("pi-session-"+job+"\n"), 0o644); err != nil {
			t.Fatalf("write session_id: %v", err)
		}
	}
	if err := os.MkdirAll(filepath.Join(wsPath, ".rick"), 0o755); err != nil {
		t.Fatalf("mkdir .rick: %v", err)
	}
	store, err := LoadJobNameStore(filepath.Join(root, "job-names.json"))
	if err != nil {
		t.Fatalf("load store: %v", err)
	}
	return store, filepath.Join(root, "job-names.json"), WorkspaceEntry{ID: "ws1", Path: wsPath, Name: "ws"}
}

// TestJobNameStoreRoundTrip 验证别名的持久化与清除语义：Set 后 Get 可见、
// 落盘可被重新加载（进程重启不丢）、空名 = 清除、未知 key 返回空。
func TestJobNameStoreRoundTrip(t *testing.T) {
	store, path, _ := newJobNameStoreEnv(t)

	if got := store.Get("ws1", "job_1"); got != "" {
		t.Fatalf("fresh store Get = %q, want empty", got)
	}
	if err := store.Set("ws1", "job_1", "  BERT   环境搭建  "); err != nil {
		t.Fatalf("set: %v", err)
	}
	// 前后空白剔除 + 内部连续空白折叠（防换行/制表破坏侧栏单行布局）
	if got := store.Get("ws1", "job_1"); got != "BERT 环境搭建" {
		t.Fatalf("Get after set = %q, want %q", got, "BERT 环境搭建")
	}
	if store.Get("ws1", "job_2") != "" {
		t.Fatal("other job must stay unnamed")
	}
	if store.Get("ws2", "job_1") != "" {
		t.Fatal("别名按 workspace 隔离——另一个 workspace 的同名 job 不受影响")
	}

	reloaded, err := LoadJobNameStore(path)
	if err != nil {
		t.Fatalf("reload: %v", err)
	}
	if got := reloaded.Get("ws1", "job_1"); got != "BERT 环境搭建" {
		t.Fatalf("reloaded Get = %q, want persisted name", got)
	}

	// 清除（空串 / 纯空白）→ 回到未命名
	if err := store.Set("ws1", "job_1", "   "); err != nil {
		t.Fatalf("clear: %v", err)
	}
	if got := store.Get("ws1", "job_1"); got != "" {
		t.Fatalf("Get after clear = %q, want empty", got)
	}
	// 清除未命名的 job 是幂等成功
	if err := store.Set("ws1", "job_2", ""); err != nil {
		t.Fatalf("clearing a never-named job must succeed: %v", err)
	}

	// 超长名字被拒绝（而不是静默截断）
	long := strings.Repeat("字", MaxJobNameLen+1)
	if err := store.Set("ws1", "job_2", long); err == nil {
		t.Fatal("over-long name must be rejected")
	}
	if _, err := ValidateJobName(long); err == nil {
		t.Fatal("ValidateJobName must reject over-long names")
	}
	if _, err := ValidateJobName(strings.Repeat("字", MaxJobNameLen)); err != nil {
		t.Fatalf("name at the limit must pass: %v", err)
	}
}

// TestJobNameNilStoreSafe 验证未注入 store 时的优雅降级（老测试/内嵌场景）：
// Get 返回空而不是 panic。
func TestJobNameNilStoreSafe(t *testing.T) {
	var store *JobNameStore
	if got := store.Get("ws1", "job_1"); got != "" {
		t.Fatalf("nil store Get = %q, want empty", got)
	}
	if len(store.All()) != 0 {
		t.Fatal("nil store All must be empty")
	}
	if err := store.Set("ws1", "job_1", "x"); err == nil {
		t.Fatal("nil store Set must error")
	}
}

// TestRenameJobHandler 验证 PUT /name 的 HTTP 契约：200 + {"job_id","name"}、
// job 不存在 404、格式错误 400、以及改名后**会话投影带上 job_name**（侧栏据此
// 显示任务名）。
func TestRenameJobHandler(t *testing.T) {
	store, _, ws := newJobNameStoreEnv(t)

	sessions, err := LoadSessionRegistry(filepath.Join(t.TempDir(), "sessions.json"))
	if err != nil {
		t.Fatalf("load sessions: %v", err)
	}
	reg, err := LoadWorkspaceRegistry(filepath.Join(t.TempDir(), "web.json"))
	if err != nil {
		t.Fatalf("load workspaces: %v", err)
	}
	added, _, err := reg.Add(ws.Path, ws.Name)
	if err != nil {
		t.Fatalf("add workspace: %v", err)
	}
	ws = added // 注册表返回真实 id（sha1 of path）
	m := NewSessionManager(sessions, reg, nil, nil, nil, nil, nil)
	m.SetJobNames(store)

	deps := Deps{Workspaces: reg, Sessions: m, JobNames: store}

	do := func(method, target, body string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(method, target, strings.NewReader(body))
		req.SetPathValue("ws", ws.ID)
		// 路径参数由 mux 注入；单测里手动补上 /jobs/{job}/name 的 job
		if parts := strings.Split(target, "/"); len(parts) >= 6 {
			req.SetPathValue("job", parts[5])
		}
		rec := httptest.NewRecorder()
		deps.handleRenameJob()(rec, req)
		return rec
	}

	rec := do(http.MethodPut, "/api/workspaces/ws1/jobs/job_1/name", `{"name":"修复流式闪烁"}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("rename status = %d, want 200 (%s)", rec.Code, rec.Body.String())
	}
	var got map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if got["job_id"] != "job_1" || got["name"] != "修复流式闪烁" {
		t.Fatalf("rename response = %v", got)
	}

	// 会话投影带上 job_name（侧栏优先显示它）
	entry := SessionEntry{
		ID: "s1", WorkspaceID: ws.ID, Type: SessionTypeDoing, Status: SessionStatusClosed,
		Params: map[string]any{"job": "job_1"},
	}
	if info := m.toSessionInfo(entry); info.JobName != "修复流式闪烁" {
		t.Fatalf("toSessionInfo job_name = %q, want the alias", info.JobName)
	}
	// 无 job 参数的会话不受影响
	if info := m.toSessionInfo(SessionEntry{ID: "s2", WorkspaceID: ws.ID, Type: SessionTypePlan}); info.JobName != "" {
		t.Fatalf("job-less session job_name = %q, want empty", info.JobName)
	}

	// 列表响应也带 name（Jobs 页展示）
	req := httptest.NewRequest(http.MethodGet, "/api/workspaces/ws1/jobs", nil)
	req.SetPathValue("ws", ws.ID)
	lrec := httptest.NewRecorder()
	deps.handleListJobs()(lrec, req)
	if lrec.Code != http.StatusOK {
		t.Fatalf("list jobs status = %d", lrec.Code)
	}
	var jobs []JobSummary
	if err := json.Unmarshal(lrec.Body.Bytes(), &jobs); err != nil {
		t.Fatalf("decode jobs: %v", err)
	}
	found := false
	for _, j := range jobs {
		if j.JobID == "job_1" {
			found = true
			if j.Name != "修复流式闪烁" {
				t.Fatalf("job_1 Name = %q, want the alias", j.Name)
			}
		}
	}
	if !found {
		t.Fatal("job_1 missing from list")
	}

	// job 不存在 → 404（不给幽灵 job 起名）
	if rec := do(http.MethodPut, "/api/workspaces/ws1/jobs/job_404/name", `{"name":"x"}`); rec.Code != http.StatusNotFound {
		t.Fatalf("unknown job status = %d, want 404", rec.Code)
	}
	// 坏 JSON → 400
	if rec := do(http.MethodPut, "/api/workspaces/ws1/jobs/job_1/name", `{`); rec.Code != http.StatusBadRequest {
		t.Fatalf("bad body status = %d, want 400", rec.Code)
	}
	// 错误响应体遵循契约 {"error":{"code","message"}}（小写——前端据此显示中文原因）
	rec = do(http.MethodPut, "/api/workspaces/ws1/jobs/job_404/name", `{"name":"x"}`)
	var errBody struct {
		Error struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &errBody); err != nil {
		t.Fatalf("decode error body: %v", err)
	}
	if errBody.Error.Code != "not_found" || errBody.Error.Message == "" {
		t.Fatalf("error body = %+v, want lowercase code/message", errBody)
	}
}

// TestJobParamBackfillAndProjection 验证 job 归属的可发现性修复：
// ① 历史行只有保留字段 _job_id（web 早期创建的 easy 会话）→ 投影出的 params.job
//
//	必须补齐（前端据此显示 job 徽标 / 「任务名」重命名入口 / Jobs 页关联）；
//
// ② BackfillJobParams 把公开的 params.job 落盘，且幂等（已有值不覆盖）。
func TestJobParamBackfillAndProjection(t *testing.T) {
	reg, err := LoadSessionRegistry(filepath.Join(t.TempDir(), "sessions.json"))
	if err != nil {
		t.Fatalf("load sessions: %v", err)
	}
	legacy := SessionEntry{
		ID: "legacy-1", WorkspaceID: "ws1", Type: SessionTypeEasy, Status: SessionStatusClosed,
		Params: map[string]any{"_job_id": "job_72", "requirement": "算 MFU"},
	}
	if err := reg.Add(legacy); err != nil {
		t.Fatalf("add: %v", err)
	}
	m := NewSessionManager(reg, nil, nil, nil, nil, nil, nil)

	// ① 投影补齐（读取路径：无需迁移也能被前端关联）
	info := m.toSessionInfo(legacy)
	if got, _ := info.Params["job"].(string); got != "job_72" {
		t.Fatalf("projected params.job = %v, want job_72", info.Params["job"])
	}
	if _, leaked := info.Params["_job_id"]; leaked {
		t.Fatal("reserved _job_id must not leak into the wire projection")
	}
	// 有 job 的行不再被别名查找漏掉
	if info := m.toSessionInfo(legacy); info.JobName != "" { // store 未注入 → 空
		t.Fatalf("unexpected job_name %q", info.JobName)
	}

	// ② 回填落盘 + 幂等
	m.BackfillJobParams()
	got, _ := reg.Get("legacy-1")
	if v, _ := got.Params["job"].(string); v != "job_72" {
		t.Fatalf("after backfill params.job = %v, want job_72", got.Params["job"])
	}
	// 已有显式 job 的行不被覆盖
	explicit := SessionEntry{
		ID: "explicit-1", WorkspaceID: "ws1", Type: SessionTypeDoing, Status: SessionStatusClosed,
		Params: map[string]any{"job": "job_1", "_job_id": "job_9"},
	}
	if err := reg.Add(explicit); err != nil {
		t.Fatalf("add explicit: %v", err)
	}
	m.BackfillJobParams()
	gotExplicit, _ := reg.Get("explicit-1")
	if v, _ := gotExplicit.Params["job"].(string); v != "job_1" {
		t.Fatalf("explicit job overwritten: %v", gotExplicit.Params["job"])
	}

	// 无 job 的行（plan 等）保持无 job
	bare := SessionEntry{ID: "bare-1", WorkspaceID: "ws1", Type: SessionTypePlan, Status: SessionStatusClosed}
	if info := m.toSessionInfo(bare); info.Params["job"] != nil {
		t.Fatalf("job-less session must not gain a job param: %v", info.Params["job"])
	}
}
