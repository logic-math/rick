package web

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
)

// Bug #4（human 2026-11-14）：所有会话类型可重命名会话名。
func TestSessionNameStoreRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "session-names.json")
	s, err := LoadSessionNameStore(path)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if got := s.Get("ws1", "sess1"); got != "" {
		t.Fatalf("fresh store should be empty, got %q", got)
	}
	if err := s.Set("ws1", "sess1", "矩阵优化专项"); err != nil {
		t.Fatalf("set: %v", err)
	}
	if got := s.Get("ws1", "sess1"); got != "矩阵优化专项" {
		t.Fatalf("get = %q", got)
	}
	// 持久化往返
	s2, err := LoadSessionNameStore(path)
	if err != nil {
		t.Fatalf("reload: %v", err)
	}
	if got := s2.Get("ws1", "sess1"); got != "矩阵优化专项" {
		t.Fatalf("persisted get = %q", got)
	}
	// 清除
	if err := s2.Set("ws1", "sess1", ""); err != nil {
		t.Fatalf("clear: %v", err)
	}
	if got := s2.Get("ws1", "sess1"); got != "" {
		t.Fatalf("cleared get = %q", got)
	}
}

// rename handler：会话名写入 + 空串清除 + 投影（name 字段）。
func TestSessionRenameHandler(t *testing.T) {
	env := newTestEnv(t) // 既有测试环境装配（registry+supervisor+workspace）
	sm := env.managerWith(t, nil, nil)
	sm.SetSessionNames(&SessionNameStore{path: filepath.Join(t.TempDir(), "session-names.json"), names: map[string]string{}})

	// dream 类型（无 loop/skill 前置文件校验——重命名与类型无关，验证非 job 类型即可）
	rec := post(t, sm.CreateSession, "/api/sessions", map[string]any{
		"workspace_id": env.wsEntry.ID,
		"type":         "dream",
		"params":       map[string]any{"requirement": "梦境巡检"},
	})
	if rec.Code != http.StatusCreated {
		t.Fatalf("create status = %d body %s", rec.Code, rec.Body.String())
	}
	var created struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &created); err != nil || created.ID == "" {
		t.Fatalf("create body parse: %v %+v", err, created)
	}

	// 重命名（PUT，复用 post helper 的形态——手动构造 PUT）
	rr := httptest.NewRecorder()
	body, _ := json.Marshal(map[string]string{"name": "矩阵优化专项"})
	req := httptest.NewRequest(http.MethodPut, "/api/sessions/"+created.ID+"/name", strings.NewReader(string(body)))
	sm.SessionRename(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("rename status = %d, body %s", rr.Code, rr.Body.String())
	}
	// 投影带 name：列表接口验证
	rec2 := get(t, sm.ListSessions, "/api/sessions")
	if rec2.Code != http.StatusOK {
		t.Fatalf("list status = %d", rec2.Code)
	}
	if !strings.Contains(rec2.Body.String(), "\"name\":\"矩阵优化专项\"") {
		t.Fatalf("projected name missing in list: %s", rec2.Body.String()[:400])
	}
	// 清除
	rr2 := httptest.NewRecorder()
	req2 := httptest.NewRequest(http.MethodPut, "/api/sessions/"+created.ID+"/name", strings.NewReader(`{"name":""}`))
	sm.SessionRename(rr2, req2)
	if rr2.Code != http.StatusOK {
		t.Fatalf("clear status = %d", rr2.Code)
	}
	rec3 := get(t, sm.ListSessions, "/api/sessions")
	if strings.Contains(rec3.Body.String(), "矩阵优化专项") {
		t.Fatalf("cleared name still projected")
	}
}
