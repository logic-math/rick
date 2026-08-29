// routes.go 挂载 rick web 的全部 HTTP 路由（api-contract.md 单源）。
//
// 设计要点：
//   - Go 1.22 http.ServeMux pattern 语法（"POST /api/sessions" / "/api/sessions/{id}/..."），
//     方法与路径参数都在 mux 层强制——handler 内不再二次判断方法。
//   - 认证经 authWrap：/api/health 与静态资源豁免（探活与首屏），其余全部
//     走 TokenAuth（Bearer header 或 ?token= query——EventSource 场景）。
//   - GET /api/workspaces 的 jobs_count 由本层 join ListJobs 补齐（数据层
//     registry.go 不读 jobs——职责分层：routes 是 join 层）。
//   - 错误统一 JSON：writeError（sessions.go）已按 WebError/ValidationError 映射
//     状态码；404 JSON 兜底在 mux "/" 根上。
//   - CORS 不开：同源部署（静态资源与 API 同一 origin）。
package web

import (
	"encoding/json"
	"io/fs"
	"net/http"
	"strings"
)

// CustomizeFunc deploys the frontend customization scaffold (env 层实现，
// task14 接入). Returns whether the scaffold was newly created.
type CustomizeFunc func() (bool, error)

// ResetFunc removes the customization overlay (env 层实现).
type ResetFunc func() error

// Deps is everything RegisterRoutes needs. task14 的 NewWebCmd(version)
// 组装并注入；Version 流向 /api/config 的 rick_version。
type Deps struct {
	Hub        *Hub
	Sessions   *SessionManager
	Workspaces *WorkspaceRegistry
	Token      string
	Static     http.Handler
	Version    string

	// Customize/Reset 以函数注入解耦 env 层（task14 前可为 nil——nil 时
	// customize/reset 返回 501 not implemented，测试注入 fake）。
	Customize CustomizeFunc
	Reset     ResetFunc
}

// authWrap wraps a handler with token auth (see auth.go: empty token
// disables auth entirely — local development mode).
func authWrap(token string, next http.Handler) http.Handler {
	return TokenAuth(token)(next)
}

// ---- /api/config & /api/health ----

type configResponse struct {
	Version      int    `json:"version"`
	RickVersion  string `json:"rick_version"`
	Port         string `json:"port"`
	AuthRequired bool   `json:"auth_required"`
}

// handleConfig serves GET /api/config (contract Server 节). Port is derived
// from the request host when the config carries none.
func handleConfig(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		port := ""
		if h := r.Host; h != "" {
			if i := strings.LastIndex(h, ":"); i >= 0 {
				port = h[i+1:]
			}
		}
		writeJSON(w, http.StatusOK, configResponse{
			Version:      1,
			RickVersion:  deps.Version,
			Port:         port,
			AuthRequired: deps.Token != "",
		})
	}
}

func handleHealth(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// ---- workspaces ----

// workspaceWithCount is the listing projection: WorkspaceEntry + jobs_count
// joined by ListJobs (read failure → -1, per task spec: 计 -1 或省略).
type workspaceWithCount struct {
	WorkspaceEntry
	JobsCount int `json:"jobs_count"`
}

// handleListWorkspaces serves GET /api/workspaces.
func handleListWorkspaces(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		list := deps.Workspaces.List()
		out := make([]workspaceWithCount, 0, len(list))
		for _, e := range list {
			count := -1
			if jobs, err := ListJobs(e.Path + "/.rick"); err == nil {
				count = len(jobs)
			}
			out = append(out, workspaceWithCount{WorkspaceEntry: e, JobsCount: count})
		}
		writeJSON(w, http.StatusOK, out)
	}
}

// handleAddWorkspace serves POST /api/workspaces (201 created / 200 idempotent).
func handleAddWorkspace(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Path string `json:"path"`
			Name string `json:"name"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, newWebError(http.StatusBadRequest, "invalid_params", "decode body: %v", err))
			return
		}
		if strings.TrimSpace(req.Path) == "" {
			writeError(w, newWebError(http.StatusBadRequest, "invalid_params", "path must not be empty"))
			return
		}
		entry, created, err := deps.Workspaces.Add(req.Path, req.Name)
		if err != nil {
			writeError(w, err)
			return
		}
		status := http.StatusOK
		if created {
			status = http.StatusCreated
		}
		writeJSON(w, status, entry)
	}
}

// handleRemoveWorkspace serves DELETE /api/workspaces/{id} → 204.
func handleRemoveWorkspace(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		if id == "" {
			writeError(w, newWebError(http.StatusBadRequest, "invalid_params", "missing workspace id"))
			return
		}
		if err := deps.Workspaces.Remove(id); err != nil {
			writeError(w, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

// ---- jobs / knowledge ----

// workspaceFromPath resolves {ws} to a registered workspace (404 on miss).
func (deps Deps) workspaceFromPath(w http.ResponseWriter, r *http.Request) (WorkspaceEntry, bool) {
	id := r.PathValue("ws")
	if id == "" {
		writeError(w, newWebError(http.StatusBadRequest, "invalid_params", "missing workspace id"))
		return WorkspaceEntry{}, false
	}
	e, ok := deps.Workspaces.Get(id)
	if !ok {
		writeError(w, newWebError(http.StatusNotFound, "not_found", "workspace %s is not registered", id))
		return WorkspaceEntry{}, false
	}
	return e, true
}

func (deps Deps) handleListJobs() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ws, ok := deps.workspaceFromPath(w, r)
		if !ok {
			return
		}
		jobs, err := ListJobs(ws.Path + "/.rick")
		if err != nil {
			writeError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, jobs)
	}
}

func (deps Deps) handleReadTasks() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ws, ok := deps.workspaceFromPath(w, r)
		if !ok {
			return
		}
		raw, err := ReadTasks(ws.Path+"/.rick", r.PathValue("job"))
		if err != nil {
			writeError(w, err)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(raw)
	}
}

// fileResponse is the jobs/knowledge file read body (contract: {"path","content"}).
type fileResponse struct {
	Path    string `json:"path"`
	Content string `json:"content"`
}

func (deps Deps) handleJobFile() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ws, ok := deps.workspaceFromPath(w, r)
		if !ok {
			return
		}
		content, err := ReadJobFile(ws.Path+"/.rick", r.PathValue("job"), r.URL.Query().Get("path"))
		if err != nil {
			writeError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, fileResponse{Path: r.URL.Query().Get("path"), Content: content})
	}
}

type treeResponse struct {
	Tree []FileNode `json:"tree"`
}

func (deps Deps) handleKnowledgeTree() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ws, ok := deps.workspaceFromPath(w, r)
		if !ok {
			return
		}
		nodes, err := KnowledgeTree(ws.Path + "/.rick")
		if err != nil {
			writeError(w, err)
			return
		}
		if nodes == nil {
			nodes = []FileNode{}
		}
		writeJSON(w, http.StatusOK, treeResponse{Tree: nodes})
	}
}

func (deps Deps) handleKnowledgeFile() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ws, ok := deps.workspaceFromPath(w, r)
		if !ok {
			return
		}
		content, err := ReadKnowledgeFile(ws.Path+"/.rick", r.URL.Query().Get("path"))
		if err != nil {
			writeError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, fileResponse{Path: r.URL.Query().Get("path"), Content: content})
	}
}

// ---- web customize / reset（env 层函数注入）----

func handleWebCustomize(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		if deps.Customize == nil {
			writeError(w, newWebError(http.StatusNotImplemented, "not_implemented", "customize is not wired in this build"))
			return
		}
		scaffolded, err := deps.Customize()
		if err != nil {
			writeError(w, newWebError(http.StatusInternalServerError, "internal", "customize: %v", err))
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"ok": true, "scaffolded": scaffolded})
	}
}

func handleWebReset(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		if deps.Reset == nil {
			writeError(w, newWebError(http.StatusNotImplemented, "not_implemented", "reset is not wired in this build"))
			return
		}
		if err := deps.Reset(); err != nil {
			writeError(w, newWebError(http.StatusInternalServerError, "internal", "reset: %v", err))
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"ok": true})
	}
}

// ---- SSE ----

func (deps Deps) handleEvents() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ServeSSE(w, r, deps.Hub, ServeSSEOptions{})
	}
}

// ---- 注册 ----

// RegisterRoutes mounts every api-contract.md endpoint onto mux. Static
// serves the SPA (embed baseline / overlay); it is mounted on "/" last so
// the more specific /api/ patterns win (ServeMux longest-prefix rule).
//
// Nil-safe: Sessions/Hub may be nil only in static-only tests; handlers
// dereferencing them are still registered (the test server never routes to
// them) — production always wires the full Deps via NewServer.
func RegisterRoutes(mux *http.ServeMux, deps Deps) {
	// Health & config: health is auth-free (probe), config requires auth.
	mux.HandleFunc("GET /api/health", handleHealth)
	mux.Handle("GET /api/config", authWrap(deps.Token, http.HandlerFunc(handleConfig(deps))))

	// Workspaces.
	mux.Handle("GET /api/workspaces", authWrap(deps.Token, http.HandlerFunc(handleListWorkspaces(deps))))
	mux.Handle("POST /api/workspaces", authWrap(deps.Token, http.HandlerFunc(handleAddWorkspace(deps))))
	mux.Handle("DELETE /api/workspaces/{id}", authWrap(deps.Token, http.HandlerFunc(handleRemoveWorkspace(deps))))

	// Sessions（全命令面——handler 内部已做状态冲突矩阵）。
	if deps.Sessions != nil {
		sm := deps.Sessions
		mux.Handle("GET /api/sessions", authWrap(deps.Token, http.HandlerFunc(sm.ListSessions)))
		mux.Handle("POST /api/sessions", authWrap(deps.Token, http.HandlerFunc(sm.CreateSession)))
		mux.Handle("GET /api/sessions/{id}", authWrap(deps.Token, http.HandlerFunc(sm.GetSession)))
		mux.Handle("POST /api/sessions/{id}/prompt", authWrap(deps.Token, http.HandlerFunc(sm.SessionPrompt)))
		mux.Handle("POST /api/sessions/{id}/steer", authWrap(deps.Token, http.HandlerFunc(sm.SessionSteer)))
		mux.Handle("POST /api/sessions/{id}/abort", authWrap(deps.Token, http.HandlerFunc(sm.SessionAbort)))
		mux.Handle("POST /api/sessions/{id}/close", authWrap(deps.Token, http.HandlerFunc(sm.SessionClose)))
		mux.Handle("POST /api/sessions/{id}/resume", authWrap(deps.Token, http.HandlerFunc(sm.SessionResume)))
		mux.Handle("POST /api/sessions/{id}/ui_response", authWrap(deps.Token, http.HandlerFunc(sm.SessionUIResponse)))
		mux.Handle("GET /api/sessions/{id}/entries", authWrap(deps.Token, http.HandlerFunc(sm.SessionEntries)))
	}

	// Jobs / knowledge（{ws} 注册工作区）。
	mux.Handle("GET /api/workspaces/{ws}/jobs", authWrap(deps.Token, http.HandlerFunc(deps.handleListJobs())))
	mux.Handle("GET /api/workspaces/{ws}/jobs/{job}/tasks", authWrap(deps.Token, http.HandlerFunc(deps.handleReadTasks())))
	mux.Handle("GET /api/workspaces/{ws}/jobs/{job}/file", authWrap(deps.Token, http.HandlerFunc(deps.handleJobFile())))
	mux.Handle("GET /api/workspaces/{ws}/knowledge/tree", authWrap(deps.Token, http.HandlerFunc(deps.handleKnowledgeTree())))
	mux.Handle("GET /api/workspaces/{ws}/knowledge/file", authWrap(deps.Token, http.HandlerFunc(deps.handleKnowledgeFile())))

	// Web 管理（env 层函数注入，task14 前可为 nil → 501）。
	mux.Handle("POST /api/web/customize", authWrap(deps.Token, http.HandlerFunc(handleWebCustomize(deps))))
	mux.Handle("POST /api/web/reset", authWrap(deps.Token, http.HandlerFunc(handleWebReset(deps))))

	// SSE（token 走 query——EventSource 无法带 header；authWrap 双通道）。
	mux.Handle("GET /api/events", authWrap(deps.Token, http.HandlerFunc(deps.handleEvents())))

	// 静态资源 + SPA fallback（免认证：首屏加载先于 token 输入）。
	if deps.Static != nil {
		mux.Handle("/", deps.Static)
	}
}

// StaticOnlyMux builds a mux with just the static handler + health (used by
// static-focused tests and as the minimal embedding surface).
func StaticOnlyMux(webFS fs.FS) *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/health", handleHealth)
	mux.Handle("/", StaticHandler(webFS, ""))
	return mux
}
