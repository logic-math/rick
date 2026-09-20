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
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/sunquan/rick/internal/workspace"
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

	// Archived is the web-layer soft-archive store (job 归档：列表默认过滤、
	// 可恢复；不碰 rick job 文件）。nil 时归档接口返回 state_conflict，
	// jobs 列表不过滤（老测试/未注入场景优雅降级）。
	Archived *ArchivedStore

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

// handleBrowseWorkspaces serves GET /api/workspaces/browse?path=<dir> — the
// immediate subdirectories of path that contain a .rick directory (depth 1,
// capped at 50 entries). The frontend uses it for the "选择机器上已有的工作区"
// flow. Non-existent / non-directory paths → 400 invalid_params.
func handleBrowseWorkspaces(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		path := strings.TrimSpace(r.URL.Query().Get("path"))
		if path == "" {
			writeError(w, newWebError(http.StatusBadRequest, "invalid_params", "path must not be empty"))
			return
		}
		fi, err := os.Stat(path)
		if err != nil || !fi.IsDir() {
			writeError(w, newWebError(http.StatusBadRequest, "invalid_params", "path is not a readable directory: %s", path))
			return
		}
		entries, err := os.ReadDir(path)
		if err != nil {
			writeError(w, newWebError(http.StatusInternalServerError, "internal", "read directory: %v", err))
			return
		}
		var found []map[string]string
		for _, e := range entries {
			if !e.IsDir() {
				continue
			}
			rickDir := filepath.Join(path, e.Name(), workspace.RickDirName)
			if st, err := os.Stat(rickDir); err == nil && st.IsDir() {
				found = append(found, map[string]string{"path": filepath.Join(path, e.Name()), "name": e.Name()})
			}
			if len(found) >= 50 {
				break
			}
		}
		writeJSON(w, http.StatusOK, map[string]any{"workspaces": found})
	}
}

// ---- fs 浏览（多级路径选择器——前端目录浏览器数据源）----

// handleFSList serves GET /api/fs/list?path=<dir> — the immediate
// subdirectories of path (non-hidden, sorted by name, capped at 100) for
// multi-level directory navigation. Returns {path, parent, entries:[{path,name}]}
// where parent = filepath.Dir(path) (empty string for the filesystem root).
func handleFSList(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimSpace(r.URL.Query().Get("path"))
	if path == "" {
		writeError(w, newWebError(http.StatusBadRequest, "invalid_params", "path must not be empty"))
		return
	}
	fi, err := os.Stat(path)
	if err != nil || !fi.IsDir() {
		writeError(w, newWebError(http.StatusBadRequest, "invalid_params", "path is not a readable directory: %s", path))
		return
	}
	entries, err := os.ReadDir(path)
	if err != nil {
		writeError(w, newWebError(http.StatusInternalServerError, "internal", "read directory: %v", err))
		return
	}

	parent := filepath.Dir(path)
	if parent == path {
		parent = "" // filesystem root
	}

	var dirs []string
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		if strings.HasPrefix(e.Name(), ".") {
			continue // hidden dirs excluded
		}
		dirs = append(dirs, e.Name())
	}
	sort.Strings(dirs)
	if len(dirs) > 100 {
		dirs = dirs[:100]
	}

	type fsEntry struct {
		Path string `json:"path"`
		Name string `json:"name"`
	}
	out := make([]fsEntry, 0, len(dirs))
	for _, name := range dirs {
		out = append(out, fsEntry{Path: filepath.Join(path, name), Name: name})
	}
	writeJSON(w, http.StatusOK, map[string]any{"path": path, "parent": parent, "entries": out})
}

// handleFSStatus serves GET /api/fs/status?path=<dir> — directory state for
// the create/register decision: {path, exists, is_dir, has_rick}.
func handleFSStatus(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimSpace(r.URL.Query().Get("path"))
	if path == "" {
		writeError(w, newWebError(http.StatusBadRequest, "invalid_params", "path must not be empty"))
		return
	}
	exists, isDir := false, false
	if fi, err := os.Stat(path); err == nil {
		exists = true
		isDir = fi.IsDir()
	}
	hasRick := false
	if exists && isDir {
		if st, err := os.Stat(filepath.Join(path, workspace.RickDirName)); err == nil && st.IsDir() {
			hasRick = true
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{"path": path, "exists": exists, "is_dir": isDir, "has_rick": hasRick})
}

// handleFSMkdir serves POST /api/fs/mkdir {path, name} — creates a
// subdirectory for the in-browser「新建子目录」flow (returns the new path;
// 200 idempotent when it already exists). Name must be a single path segment
// (no / \ . ..).
func handleFSMkdir(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Path string `json:"path"`
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, newWebError(http.StatusBadRequest, "invalid_params", "decode body: %v", err))
		return
	}
	path := strings.TrimSpace(req.Path)
	name := strings.TrimSpace(req.Name)
	if path == "" || name == "" {
		writeError(w, newWebError(http.StatusBadRequest, "invalid_params", "path and name must not be empty"))
		return
	}
	if strings.ContainsAny(name, "/\\") || name == "." || name == ".." {
		writeError(w, newWebError(http.StatusBadRequest, "invalid_params", "invalid directory name"))
		return
	}
	target := filepath.Join(path, name)
	if err := os.Mkdir(target, 0o755); err != nil {
		if os.IsExist(err) {
			writeJSON(w, http.StatusOK, map[string]string{"path": target})
			return
		}
		writeError(w, newWebError(http.StatusInternalServerError, "internal", "mkdir: %v", err))
		return
	}
	writeJSON(w, http.StatusCreated, map[string]string{"path": target})
}

// createWorkspaceDirs bootstraps the minimal .rick structure for a new
// workspace (idempotent MkdirAll — mirrors handler.ensureWorkspaceDirsIn's
// six directories so a freshly created workspace is immediately usable by
// rick commands). Returns the rickDir path.
func createWorkspaceDirs(path string) (string, error) {
	rickDir := filepath.Join(path, workspace.RickDirName)
	dirs := []string{
		rickDir,
		filepath.Join(rickDir, workspace.LoopsDirName),
		filepath.Join(rickDir, workspace.SkillsDirName),
		filepath.Join(rickDir, workspace.DomainDirName),
		filepath.Join(rickDir, workspace.JobsDirName),
		filepath.Join(rickDir, workspace.DreamDirName),
		filepath.Join(rickDir, workspace.DraftDirName),
	}
	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return "", err
		}
	}
	return rickDir, nil
}

// handleCreateWorkspace serves POST /api/workspaces/create {path, name?}:
// creates the directory tree (and workspace parent if missing), bootstraps
// the minimal .rick structure when absent, then registers the workspace
// (201 created / 200 idempotent when already registered).
func handleCreateWorkspace(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Path string `json:"path"`
			Name string `json:"name"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, newWebError(http.StatusBadRequest, "invalid_params", "decode body: %v", err))
			return
		}
		path := strings.TrimSpace(req.Path)
		if path == "" {
			writeError(w, newWebError(http.StatusBadRequest, "invalid_params", "path must not be empty"))
			return
		}
		if err := os.MkdirAll(path, 0o755); err != nil {
			writeError(w, newWebError(http.StatusInternalServerError, "internal", "create workspace dir: %v", err))
			return
		}
		if _, err := createWorkspaceDirs(path); err != nil {
			writeError(w, newWebError(http.StatusInternalServerError, "internal", "init .rick structure: %v", err))
			return
		}
		entry, created, err := deps.Workspaces.Add(path, req.Name)
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

// handleReorderWorkspaces serves PUT /api/workspaces/order {ids:[...]} → 204.
// Reorders the display order of registered workspaces (sidebar drag-sort).
// Validation failures (id set mismatch / unknown / duplicate) → 400
// invalid_order; empty ids is a no-op 204.
func handleReorderWorkspaces(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			IDs []string `json:"ids"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, newWebError(http.StatusBadRequest, "invalid_params", "decode body: %v", err))
			return
		}
		if err := deps.Workspaces.Reorder(req.IDs); err != nil {
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
		includeArchived := r.URL.Query().Get("include_archived") == "true"
		rickDir := ws.Path + "/.rick"
		var manual []string
		if deps.Archived != nil {
			manual = deps.Archived.List(ws.ID)
		}
		// Auto-archive: jobs already processed by dream are archived
		// (archive = dream 转化, job_36 语义).
		dreamArchived := DreamArchivedJobs(rickDir)
		jobs, err := ListJobsFiltered(rickDir, manual, dreamArchived, includeArchived)
		if err != nil {
			writeError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, jobs)
	}
}

// handleArchiveJob serves POST /api/workspaces/{ws}/jobs/{job}/archive →
// 204. Only completed jobs (every task status=success) may be archived;
// archiving is idempotent and never touches rick job files.
func (deps Deps) handleArchiveJob() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ws, ok := deps.workspaceFromPath(w, r)
		if !ok {
			return
		}
		jobID := r.PathValue("job")
		if jobID == "" {
			writeError(w, newWebError(http.StatusBadRequest, "invalid_params", "missing job id"))
			return
		}
		// force=true（前端「关闭」按钮）：允许归档**未完成**的 job——历史遗留的
		// blocked/error/长期 pending 的 job 永远无法满足「全部 success」，否则会永久
		// 滞留在 Jobs 默认列表（用户实测反馈）。关闭只是从列表隐藏（写归档集），
		// 文件不动，且可在归档区恢复；不伪造任何 task 状态。
		force := r.URL.Query().Get("force") == "true"
		if !force {
			done, err := jobIsComplete(ws.Path+"/.rick", jobID)
			if err != nil {
				writeError(w, err)
				return
			}
			if !done {
				writeError(w, newWebError(http.StatusConflict, "state_conflict", "only completed jobs can be archived"))
				return
			}
		} else if _, err := jobRoot(ws.Path+"/.rick", jobID); err != nil {
			// 仍要求 job 真实存在（防止对不存在的 id 写归档标记）
			writeError(w, err)
			return
		}
		if deps.Archived == nil {
			writeError(w, newWebError(http.StatusConflict, "state_conflict", "archive store not configured"))
			return
		}
		if err := deps.Archived.Archive(ws.ID, jobID); err != nil {
			writeError(w, newWebError(http.StatusInternalServerError, "internal", "archive job: %v", err))
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

// handleUnarchiveJob serves POST /api/workspaces/{ws}/jobs/{job}/unarchive
// → 204 (idempotent; unknown archive entry is a no-op).
func (deps Deps) handleUnarchiveJob() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ws, ok := deps.workspaceFromPath(w, r)
		if !ok {
			return
		}
		jobID := r.PathValue("job")
		if jobID == "" {
			writeError(w, newWebError(http.StatusBadRequest, "invalid_params", "missing job id"))
			return
		}
		if deps.Archived == nil {
			writeError(w, newWebError(http.StatusConflict, "state_conflict", "archive store not configured"))
			return
		}
		if err := deps.Archived.Unarchive(ws.ID, jobID); err != nil {
			writeError(w, newWebError(http.StatusInternalServerError, "internal", "unarchive job: %v", err))
			return
		}
		w.WriteHeader(http.StatusNoContent)
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

// handleJobFileList serves GET /api/workspaces/{ws}/jobs/{job}/files — the actual
// file tree under the job's plan/doing/grilling/prompts directories (recursive,
// text files only, 200 max). The frontend previously guessed paths from the
// "convention" (plan/task1.md etc.) which breaks for easy jobs whose layout differs.
func (deps Deps) handleJobFileList() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ws, ok := deps.workspaceFromPath(w, r)
		if !ok {
			return
		}
		jobID := r.PathValue("job")
		if jobID == "" {
			writeError(w, newWebError(http.StatusBadRequest, "invalid_params", "missing job id"))
			return
		}
		rickDir := ws.Path + "/.rick"
		jobRoot, err := jobRoot(rickDir, jobID)
		if err != nil {
			writeError(w, err)
			return
		}
		type FileEntry struct {
			Path string `json:"path"`
			Size int64  `json:"size"`
		}
		var entries []FileEntry
		exts := map[string]bool{".md": true, ".log": true, ".json": true, ".txt": true, ".yaml": true, ".yml": true}
		for _, sub := range []string{"plan", "doing", "grilling", "prompts"} {
			root := filepath.Join(jobRoot, sub)
			filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
				if err != nil || d.IsDir() {
					return nil
				}
				if len(entries) >= 200 {
					return filepath.SkipAll
				}
				ext := strings.ToLower(filepath.Ext(path))
				if !exts[ext] {
					return nil
				}
				rel, _ := filepath.Rel(jobRoot, path)
				st, _ := d.Info()
				size := int64(0)
				if st != nil {
					size = st.Size()
				}
				entries = append(entries, FileEntry{Path: filepath.ToSlash(rel), Size: size})
				return nil
			})
		}
		if entries == nil {
			entries = []FileEntry{}
		}
		writeJSON(w, http.StatusOK, entries)
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
		// 契约「连接建立即发 server_info」：每次连接（含重连）先发一个
		// server_info envelope，客户端拿到全量重建信号后才开始消费事件流。
		// version 与 handleConfig 的 rick_version 同源（Deps.Version）。
		info := ServerInfoEvent(1, deps.Version)
		ServeSSE(w, r, deps.Hub, ServeSSEOptions{InitialInfo: &info})
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
	mux.Handle("GET /api/workspaces/browse", authWrap(deps.Token, http.HandlerFunc(handleBrowseWorkspaces(deps))))
	mux.Handle("POST /api/workspaces/create", authWrap(deps.Token, http.HandlerFunc(handleCreateWorkspace(deps))))
	mux.Handle("DELETE /api/workspaces/{id}", authWrap(deps.Token, http.HandlerFunc(handleRemoveWorkspace(deps))))
	mux.Handle("PUT /api/workspaces/order", authWrap(deps.Token, http.HandlerFunc(handleReorderWorkspaces(deps))))

	// FS browse（多级路径选择器——目录导航/新建子目录/注册与创建判定）。
	mux.Handle("GET /api/fs/list", authWrap(deps.Token, http.HandlerFunc(handleFSList)))
	mux.Handle("GET /api/fs/status", authWrap(deps.Token, http.HandlerFunc(handleFSStatus)))
	mux.Handle("POST /api/fs/mkdir", authWrap(deps.Token, http.HandlerFunc(handleFSMkdir)))

	// Sessions（全命令面——handler 内部已做状态冲突矩阵）。
	if deps.Sessions != nil {
		sm := deps.Sessions
		mux.Handle("GET /api/sessions", authWrap(deps.Token, http.HandlerFunc(sm.ListSessions)))
		mux.Handle("POST /api/sessions", authWrap(deps.Token, http.HandlerFunc(sm.CreateSession)))
		mux.Handle("POST /api/sessions/import", authWrap(deps.Token, http.HandlerFunc(sm.ImportSession)))
		mux.Handle("GET /api/sessions/{id}", authWrap(deps.Token, http.HandlerFunc(sm.GetSession)))
		mux.Handle("POST /api/sessions/{id}/prompt", authWrap(deps.Token, http.HandlerFunc(sm.SessionPrompt)))
		mux.Handle("POST /api/sessions/{id}/steer", authWrap(deps.Token, http.HandlerFunc(sm.SessionSteer)))
		mux.Handle("POST /api/sessions/{id}/abort", authWrap(deps.Token, http.HandlerFunc(sm.SessionAbort)))
		mux.Handle("POST /api/sessions/{id}/close", authWrap(deps.Token, http.HandlerFunc(sm.SessionClose)))
		mux.Handle("POST /api/sessions/{id}/archive", authWrap(deps.Token, http.HandlerFunc(sm.SessionArchive)))
		mux.Handle("POST /api/sessions/{id}/unarchive", authWrap(deps.Token, http.HandlerFunc(sm.SessionUnarchive)))
		mux.Handle("POST /api/sessions/{id}/resume", authWrap(deps.Token, http.HandlerFunc(sm.SessionResume)))
		mux.Handle("POST /api/sessions/{id}/ui_response", authWrap(deps.Token, http.HandlerFunc(sm.SessionUIResponse)))
		mux.Handle("GET /api/sessions/{id}/entries", authWrap(deps.Token, http.HandlerFunc(sm.SessionEntries)))
		mux.Handle("GET /api/sessions/{id}/models", authWrap(deps.Token, http.HandlerFunc(sm.SessionModels)))
		mux.Handle("POST /api/sessions/{id}/model", authWrap(deps.Token, http.HandlerFunc(sm.SessionSetModel)))
		mux.Handle("POST /api/sessions/{id}/thinking", authWrap(deps.Token, http.HandlerFunc(sm.SessionSetThinking)))
		mux.Handle("GET /api/sessions/{id}/prompt", authWrap(deps.Token, http.HandlerFunc(sm.SessionPromptFiles)))
		mux.Handle("GET /api/workspaces/{ws}/sessions/{id}/prompt", authWrap(deps.Token, http.HandlerFunc(sm.SessionPromptFiles)))
	}

	// Jobs / knowledge（{ws} 注册工作区）。
	mux.Handle("GET /api/workspaces/{ws}/jobs", authWrap(deps.Token, http.HandlerFunc(deps.handleListJobs())))
	mux.Handle("POST /api/workspaces/{ws}/jobs/{job}/archive", authWrap(deps.Token, http.HandlerFunc(deps.handleArchiveJob())))
	mux.Handle("POST /api/workspaces/{ws}/jobs/{job}/unarchive", authWrap(deps.Token, http.HandlerFunc(deps.handleUnarchiveJob())))
	mux.Handle("GET /api/workspaces/{ws}/jobs/{job}/tasks", authWrap(deps.Token, http.HandlerFunc(deps.handleReadTasks())))
	mux.Handle("GET /api/workspaces/{ws}/jobs/{job}/file", authWrap(deps.Token, http.HandlerFunc(deps.handleJobFile())))
	mux.Handle("GET /api/workspaces/{ws}/jobs/{job}/files", authWrap(deps.Token, http.HandlerFunc(deps.handleJobFileList())))
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
