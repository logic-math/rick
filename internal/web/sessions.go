// sessions.go 是 rick web 的会话执行心脏（task11）：把 Supervisor（task4 的
// pi rpc 子进程管理）、handler XxxIn core（task2 的路径参数化编排）与 SSE
// Hub（task8 的单流多路复用总线）三线汇合成统一的会话生命周期。
//
// 会话模型（api-contract.md Sessions 节 + design-tree L2 终判）：
//
//	交互型（plan/easy/ctrl/human-loop/learning/dream[interactive]）
//	    → supervisor.Spawn(pi --mode rpc 常驻 worker) + bootstrap prompt
//	后台型（doing/dream[background]）
//	    → goroutine 跑 handler.DoingIn/DreamIn(ctx, ..., progress→Hub)
//	closed 会话
//	    → 离线读 pi session JSONL（不 spawn 进程）；resume 才重新 spawn
//
// 状态机：pending → active|running → closed|error。
// close 幂等（已 closed 仍 202）；resume 仅交互型可（后台型 409）。
//
// 事件泵：每个活跃 worker 一个 goroutine，把 Events() 全量转
// Hub.Publish（session_event 透传 + agent_settled 的 idle 提示）；worker
// 意外死亡（非用户 close）→ 会话落 closed（reason 记录死因）。
//
// prompt 产出刻意绕过会启动交互 CLI 的 handler.XxxIn（web 不能接管终端），
// 直接消费 builder/prompt 的路径参数化导出面（SavePlanPrompt 等）——与
// CLI 同一套提示词产物，只是驱动方式从 TUI 换成 rpc。

package web

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/sunquan/rick/internal/builder"
	"github.com/sunquan/rick/internal/handler"
	"github.com/sunquan/rick/internal/runtime"
	"github.com/sunquan/rick/internal/workspace"
)

// ---- 可注入的后台执行器（测试 fake 化） ----

// DoingRunner runs one doing job in the background (web 的 doing 执行桥).
// 生产实现包 handler.DoingIn；测试注入 fake 以断言 progress→Hub 链路。
type DoingRunner func(ctx context.Context, rickDir, jobID string, progress func(handler.DoingEvent)) error

// DreamRunner runs one dream pass in the background (non-interactive).
type DreamRunner func(ctx context.Context, rickDir string, jobNum int, progress func(handler.DreamEvent)) error

// SessionManager orchestrates the whole session lifecycle. It is safe for
// concurrent use; state transitions are serialized under mu.
type SessionManager struct {
	sessions   *SessionRegistry
	workspaces *WorkspaceRegistry
	sup        *runtime.Supervisor
	hub        *Hub

	doingRunner DoingRunner
	dreamRunner DreamRunner

	// pending routes correlated rpc responses (get_entries) back to the
	// waiting HTTP requesters via the worker event pumps.
	pending *pendingResp

	mu      sync.Mutex
	cancels map[string]context.CancelFunc // background sessions (doing/dream)
	closing map[string]bool               // sessions closed by the user (worker death is not a crash)
	bgWaits map[string]chan struct{}      // closed when the background goroutine exits
}

// NewSessionManager wires the dependencies. doingRunner/dreamRunner may be
// nil — defaults wrap handler.DoingIn / handler.DreamIn with the CLI
// runtime (the same Runtime the composition root injects elsewhere).
func NewSessionManager(sessions *SessionRegistry, workspaces *WorkspaceRegistry, sup *runtime.Supervisor, hub *Hub, rt runtime.Runtime, doing DoingRunner, dream DreamRunner) *SessionManager {
	m := &SessionManager{
		sessions:   sessions,
		workspaces: workspaces,
		sup:        sup,
		hub:        hub,
		pending:    &pendingResp{wait: make(map[string]chan *runtime.RpcEvent)},
		cancels:    make(map[string]context.CancelFunc),
		closing:    make(map[string]bool),
		bgWaits:    make(map[string]chan struct{}),
	}
	m.doingRunner = doing
	if m.doingRunner == nil {
		m.doingRunner = func(ctx context.Context, rickDir, jobID string, progress func(handler.DoingEvent)) error {
			return handler.DoingIn(ctx, rickDir, jobID, handler.Options{}, rt, progress)
		}
	}
	m.dreamRunner = dream
	if m.dreamRunner == nil {
		m.dreamRunner = func(ctx context.Context, rickDir string, jobNum int, progress func(handler.DreamEvent)) error {
			// background=true → ModePrint（非交互、stdout 转发），goroutine 驱动
			return handler.DreamIn(ctx, rickDir, jobNum, true, handler.Options{}, progress)
		}
	}
	return m
}

// ReconcileOnStart 服务启动对账：注册表里 status=active/running 的会话在本次进程
// 重启后没有对应 worker/goroutine（旧进程已死）——统一标记为 error（worker lost
// on server restart），前端据此显示 Resume 按钮恢复（SessionResume 支持 error→active）。
// 这是「重启后发送消息 409 session worker is not alive」的根治（job_36 验收期实测）。
func (m *SessionManager) ReconcileOnStart() {
	for _, e := range m.sessions.List() {
		if e.Status != SessionStatusActive && e.Status != SessionStatusRunning {
			continue
		}
		if e.Type == SessionTypeDoing || e.Type == SessionTypeDream {
			// 后台型 goroutine 随进程消亡——一律标记 error（可重新发起）
			m.updateStatus(e, SessionStatusError, "server restarted: background task lost")
			continue
		}
		w := m.sup.Get(e.ID)
		if w == nil || w.IsDead() {
			m.updateStatus(e, SessionStatusError, "server restarted: worker lost")
		}
	}
}

// ---- HTTP handlers ----

// sessionIDFromPath resolves the {id} path segment. Route patterns
// (task13) register "/api/sessions/{id}/..." so PathValue works in
// production; tests without patterns fall back to parsing the URL.
func sessionIDFromPath(r *http.Request) string {
	if id := r.PathValue("id"); id != "" {
		return id
	}
	// /api/sessions/<id>[/...]
	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(parts) >= 3 && parts[0] == "api" && parts[1] == "sessions" {
		return parts[2]
	}
	return ""
}

// writeJSON marshals v as the response body.
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

// writeError maps an error to the contract error body. WebError and
// ValidationError carry codes; anything else is a 500 internal.
func writeError(w http.ResponseWriter, err error) {
	var we *WebError
	if errors.As(err, &we) {
		writeJSON(w, we.Status, map[string]any{"error": we})
		return
	}
	var ve *ValidationError
	if errors.As(err, &ve) {
		status := http.StatusBadRequest
		switch ve.Code {
		case "not_found":
			status = http.StatusNotFound
		}
		writeJSON(w, status, map[string]any{"error": ve})
		return
	}
	writeJSON(w, http.StatusInternalServerError, map[string]any{
		"error": map[string]string{"code": "internal", "message": err.Error()},
	})
}

// sessionInfo is the wire projection of a SessionEntry (reserved _-prefixed
// params are stripped; contract SessionInfo).
type sessionInfo struct {
	ID          string         `json:"id"`
	WorkspaceID string         `json:"workspace_id"`
	Type        string         `json:"type"`
	Title       string         `json:"title,omitempty"`
	Params      map[string]any `json:"params"`
	Status      string         `json:"status"`
	PISessionID string         `json:"pi_session_id"`
	CreatedAt   time.Time      `json:"created_at"`
	ClosedAt    *time.Time     `json:"closed_at,omitempty"`
	Archived    bool           `json:"archived,omitempty"`
	ArchivedAt  *time.Time     `json:"archived_at,omitempty"`
	// Busy mirrors SessionEntry.Busy (agent streaming this turn) — the
	// authoritative source for the frontend's streaming/idle input state
	// (刷新/重连不得靠客户端事件重放推断：窗口可能丢 agent_start)。
	Busy bool `json:"busy,omitempty"`
}

func toSessionInfo(e SessionEntry) sessionInfo {
	params := make(map[string]any, len(e.Params))
	for k, v := range e.Params {
		if strings.HasPrefix(k, "_") {
			continue // reserved plumbing (prompt files), not client data
		}
		params[k] = v
	}
	info := sessionInfo{
		ID:          e.ID,
		WorkspaceID: e.WorkspaceID,
		Type:        e.Type,
		Title:       e.Title,
		Params:      params,
		Status:      e.Status,
		PISessionID: e.PISessionID,
		CreatedAt:   e.CreatedAt,
		Archived:    e.Archived,
			Busy:        e.Status == SessionStatusActive && e.Busy,
	}
	if !e.ClosedAt.IsZero() {
		closed := e.ClosedAt
		info.ClosedAt = &closed
	}
	if !e.ArchivedAt.IsZero() {
		archived := e.ArchivedAt
		info.ArchivedAt = &archived
	}
	return info
}

// CreateSession handles POST /api/sessions.
func (m *SessionManager) CreateSession(w http.ResponseWriter, r *http.Request) {
	var req struct {
		WorkspaceID string         `json:"workspace_id"`
		Type        string         `json:"type"`
		Params      map[string]any `json:"params"`
		Title       string         `json:"title"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, newWebError(http.StatusBadRequest, "invalid_params", "decode body: %v", err))
		return
	}
	if req.Params == nil {
		req.Params = map[string]any{}
	}
	if err := ValidateSessionRequest(req.Type, req.Params); err != nil {
		writeError(w, err)
		return
	}
	ws, ok := m.workspaces.Get(req.WorkspaceID)
	if !ok {
		writeError(w, newWebError(http.StatusNotFound, "not_found", "workspace %s is not registered", req.WorkspaceID))
		return
	}

	entry, err := m.createSessionLocked(ws, req)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, toSessionInfo(*entry))
}

// createSessionLocked does the creation work.
func (m *SessionManager) createSessionLocked(ws WorkspaceEntry, req struct {
	WorkspaceID string         `json:"workspace_id"`
	Type        string         `json:"type"`
	Params      map[string]any `json:"params"`
	Title       string         `json:"title"`
}) (*SessionEntry, error) {
	rickDir := filepath.Join(ws.Path, ".rick")

	webID, err := newUUID()
	if err != nil {
		return nil, fmt.Errorf("generate session id: %w", err)
	}
	piID, err := newUUID()
	if err != nil {
		return nil, fmt.Errorf("generate pi session id: %w", err)
	}

	entry := SessionEntry{
		ID:          webID,
		WorkspaceID: ws.ID,
		Type:        req.Type,
		Title:       req.Title,
		Params:      cloneParams(req.Params),
		Status:      "pending",
		PISessionID: piID,
		CreatedAt:   time.Now(),
	}

	// Background types never spawn an rpc worker.
	if req.Type == SessionTypeDoing || (req.Type == SessionTypeDream && dreamMode(req.Params) == DreamModeBackground) {
		if err := m.sessions.Add(entry); err != nil {
			return nil, err
		}
		m.startBackground(&entry, rickDir, ws)
		return &entry, nil
	}

	// Interactive types: prepare prompt artifacts, then spawn.
	prep, err := m.prepareInteractive(rickDir, req.Type, req.Params)
	if err != nil {
		return nil, err
	}
	entry.Title = orDefault(entry.Title, prep.title)

	// Persist prompt file paths for resume (reserved params, stripped in the
	// wire projection). The files live inside the workspace and survive
	// restarts; resume re-injects them as system prompts.
	entry.Params["_prompt_file"] = prep.promptFile
	if prep.methodFile != "" {
		entry.Params["_method_file"] = prep.methodFile
	}
	if prep.JobID != "" {
		entry.Params["_job_id"] = prep.JobID
	}

	if err := m.sessions.Add(entry); err != nil {
		return nil, err
	}

	spec := runtime.SpawnSpec{
		SessionID:     webID,
		Dir:           ws.Path, // anchor to workspace root (parent of .rick)
		MethodFile:    prep.methodFile,
		PromptFile:    prep.promptFile,
		SessionIDFlag: piID,
		CreateNew:     true,
	}
	worker, err := m.sup.Spawn(spec)
	if err != nil {
		m.updateStatus(entry, SessionStatusError, "spawn failed: "+err.Error())
		return nil, newWebError(http.StatusInternalServerError, "spawn_failed", "%v", err)
	}

	// Kick the session with the same bootstrap trigger the CLI uses.
	if line, berr := (runtime.NewRpcClient()).Prompt(runtime.BootstrapMessage, ""); berr == nil {
		_ = worker.Send(line)
	}

	m.updateStatus(entry, SessionStatusActive, "")

	// CLI-compat: persist the pi session id where the CLI resume paths look
	// (plan/easy/human-loop have session_id contracts).
	if prep.persistDir != "" {
		_ = os.MkdirAll(prep.persistDir, 0755)
		_ = os.WriteFile(filepath.Join(prep.persistDir, "session_id"), []byte(piID), 0644)
	}

	m.pumpWorker(webID, worker)
	return &entry, nil
}

// startBackground launches a doing/dream background runner in a goroutine.
func (m *SessionManager) startBackground(entry *SessionEntry, rickDir string, ws WorkspaceEntry) {
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})

	m.mu.Lock()
	m.cancels[entry.ID] = cancel
	m.bgWaits[entry.ID] = done
	m.mu.Unlock()

	m.updateStatus(*entry, SessionStatusRunning, "")

	go func() {
		defer close(done)
		var err error
		switch entry.Type {
		case SessionTypeDoing:
			jobID := paramString(entry.Params, "job")
			err = m.doingRunner(ctx, rickDir, jobID, func(ev handler.DoingEvent) {
				m.hub.Publish(JobsUpdate(ws.ID, ev.JobID, []TaskDiff{
					{TaskID: ev.TaskID, From: ev.From, To: ev.To},
				}, nil))
				// doing 的任务态变更同时作为会话事件流出（监控视图主体）。
				m.hub.Publish(Envelope{
					Type:      EventTypeSessionEvent,
					SessionID: entry.ID,
					Data: mustMarshal(map[string]any{
						"kind":    "doing_progress",
						"job_id":  ev.JobID,
						"task_id": ev.TaskID,
						"from":    ev.From,
						"to":      ev.To,
					}),
				})
			})
		case SessionTypeDream:
			jobNum := DreamDefaultJobNum
			if v, ok := paramInt(entry.Params, "job_num"); ok {
				jobNum = v
			}
			err = m.dreamRunner(ctx, rickDir, jobNum, func(ev handler.DreamEvent) {
				m.hub.Publish(Envelope{
					Type:      EventTypeSessionEvent,
					SessionID: entry.ID,
					Data: mustMarshal(map[string]any{
						"kind":    "dream_progress",
						"phase":   ev.Phase,
						"job_ids": ev.JobIDs,
					}),
				})
			})
		}
		if err != nil {
			m.updateStatus(*entry, SessionStatusError, err.Error())
		} else {
			m.updateStatus(*entry, SessionStatusClosed, "completed")
		}
		m.mu.Lock()
		delete(m.cancels, entry.ID)
		m.mu.Unlock()
	}()
}

// ListSessions handles GET /api/sessions?workspace=<ws_id>.
func (m *SessionManager) ListSessions(w http.ResponseWriter, r *http.Request) {
	wsFilter := r.URL.Query().Get("workspace")
	var list []SessionEntry
	if wsFilter != "" {
		if _, ok := m.workspaces.Get(wsFilter); !ok {
			writeError(w, newWebError(http.StatusNotFound, "not_found", "workspace %s is not registered", wsFilter))
			return
		}
		list = m.sessions.GetByWorkspace(wsFilter)
	} else {
		list = m.sessions.List()
	}

	// Archived view (session-level archive, user-initiated): paginated object
	// response ordered by created_at desc. Default view stays a bare array
	// without archived sessions (frontend branches on the archived param).
	if r.URL.Query().Get("archived") == "true" {
		arch := make([]sessionInfo, 0, len(list))
		for _, e := range list {
			if e.Archived {
				arch = append(arch, toSessionInfo(e))
			}
		}
		sort.Slice(arch, func(i, j int) bool {
			return arch[i].CreatedAt.After(arch[j].CreatedAt)
		})
		total := len(arch)
		limit := 50
		if v := r.URL.Query().Get("limit"); v != "" {
			if n, err := strconv.Atoi(v); err == nil && n > 0 {
				limit = n
			}
		}
		if limit > 200 {
			limit = 200
		}
		offset := 0
		if v := r.URL.Query().Get("offset"); v != "" {
			if n, err := strconv.Atoi(v); err == nil && n > 0 {
				offset = n
			}
		}
		page := []sessionInfo{}
		if offset < total {
			end := offset + limit
			if end > total {
				end = total
			}
			page = arch[offset:end]
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"items":  page,
			"total":  total,
			"limit":  limit,
			"offset": offset,
		})
		return
	}

	out := make([]sessionInfo, 0, len(list))
	for _, e := range list {
		if e.Archived {
			continue
		}
		out = append(out, toSessionInfo(e))
	}
	writeJSON(w, http.StatusOK, out)
}

// GetSession handles GET /api/sessions/{id}.
func (m *SessionManager) GetSession(w http.ResponseWriter, r *http.Request) {
	entry, ok := m.lookup(w, r)
	if !ok {
		return
	}
	writeJSON(w, http.StatusOK, toSessionInfo(entry))
}

// SessionPrompt handles POST /api/sessions/{id}/prompt.
func (m *SessionManager) SessionPrompt(w http.ResponseWriter, r *http.Request) {
	m.command(w, r, "prompt", func(worker *runtime.Worker, msg string) error {
		line, err := (runtime.NewRpcClient()).Prompt(msg, "")
		if err != nil {
			return err
		}
		return worker.Send(line)
	})
}

// SessionSteer handles POST /api/sessions/{id}/steer.
func (m *SessionManager) SessionSteer(w http.ResponseWriter, r *http.Request) {
	m.command(w, r, "steer", func(worker *runtime.Worker, msg string) error {
		line, err := (runtime.NewRpcClient()).Steer(msg)
		if err != nil {
			return err
		}
		return worker.Send(line)
	})
}

// command is the shared prompt/steer shape: active-only, streaming-aware.
func (m *SessionManager) command(w http.ResponseWriter, r *http.Request, what string, send func(*runtime.Worker, string) error) {
	entry, ok := m.lookup(w, r)
	if !ok {
		return
	}
	if entry.Status != SessionStatusActive {
		writeError(w, newWebError(http.StatusConflict, "state_conflict", "session is %s, %s requires active", entry.Status, what))
		return
	}
	worker := m.sup.Get(entry.ID)
	if worker == nil || worker.IsDead() {
		// worker 失活（崩溃/心跳超时/重启遗留）——自动标记 error（终止/中断语义），
		// 前端收到 session_state 后切换到 error 态并显示 Resume 引导（用户反馈：
		// 非活跃会话仍显示 active，进入发送就 409——应标记中断+引导 resume）。
		m.markWorkerLost(entry)
		writeError(w, newWebError(http.StatusConflict, "state_conflict", "session worker is not alive"))
		return
	}
	// A streaming session rejects plain prompts (contract: 409, use steer).
	if what == "prompt" {
		if st := worker.State(); st != nil && st.IsStreaming {
			writeError(w, newWebError(http.StatusConflict, "state_conflict", "agent is streaming — use steer"))
			return
		}
	}
	var req struct {
		Message string `json:"message"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, newWebError(http.StatusBadRequest, "invalid_params", "decode body: %v", err))
		return
	}
	if strings.TrimSpace(req.Message) == "" {
		writeError(w, newWebError(http.StatusBadRequest, "invalid_params", "message must not be empty"))
		return
	}
	if err := send(worker, req.Message); err != nil {
		// 管道断裂（worker 刚死）——同样标记 error 而非静默 500
		m.markWorkerLost(entry)
		writeError(w, newWebError(http.StatusConflict, "state_conflict", "session worker is not alive"))
		return
	}
	writeJSON(w, http.StatusAccepted, map[string]any{"ok": true})
}

// SessionAbort handles POST /api/sessions/{id}/abort.
func (m *SessionManager) SessionAbort(w http.ResponseWriter, r *http.Request) {
	entry, ok := m.lookup(w, r)
	if !ok {
		return
	}
	if entry.Status != SessionStatusActive {
		writeError(w, newWebError(http.StatusConflict, "state_conflict", "session is %s, abort requires active", entry.Status))
		return
	}
	worker := m.sup.Get(entry.ID)
	if worker == nil || worker.IsDead() {
		m.markWorkerLost(entry)
		writeError(w, newWebError(http.StatusConflict, "state_conflict", "session worker is not alive"))
		return
	}
	if line, err := (runtime.NewRpcClient()).Abort(); err == nil {
		_ = worker.Send(line)
	}
	writeJSON(w, http.StatusAccepted, map[string]any{"ok": true})
}

// SessionClose handles POST /api/sessions/{id}/close. Idempotent: an already
// closed session still returns 202 (contract).
func (m *SessionManager) SessionClose(w http.ResponseWriter, r *http.Request) {
	entry, ok := m.lookup(w, r)
	if !ok {
		return
	}
	if entry.Status == SessionStatusClosed {
		writeJSON(w, http.StatusAccepted, map[string]any{"ok": true})
		return
	}
	m.closeSession(entry, "user closed")
	writeJSON(w, http.StatusAccepted, map[string]any{"ok": true})
}

// closeSession terminates a session (worker kill / background cancel) and
// records the closed state.
func (m *SessionManager) closeSession(entry SessionEntry, reason string) {
	m.mu.Lock()
	if m.closing[entry.ID] {
		m.mu.Unlock()
		return
	}
	m.closing[entry.ID] = true
	cancel := m.cancels[entry.ID]
	m.mu.Unlock()

	if entry.Type == SessionTypeDoing || entry.Type == SessionTypeDream {
		if cancel != nil {
			cancel()
			// Give the goroutine a moment to observe cancellation; the state
			// transition happens in its error path (context cancelled).
			if done := m.bgWait(entry.ID); done != nil {
				select {
				case <-done:
				case <-time.After(3 * time.Second):
				}
			}
		}
	} else {
		if worker := m.sup.Get(entry.ID); worker != nil {
			worker.Close()
			m.sup.Remove(entry.ID)
		}
	}
	// easy 会话正常关闭 = 需求完成：把合成 tasks.json 标 success（创建时是
	// running——dream 只学 tasks 全 success 的 job，中断的 easy 会话保持
	// running 不被学习/归档）。
	if entry.Type == SessionTypeEasy {
		if jobID := paramString(entry.Params, "_job_id"); jobID != "" {
			if ws, ok := m.workspaces.Get(entry.WorkspaceID); ok {
				if err := updateEasyTasksStatus(ws.Path+"/.rick", jobID, "success"); err != nil {
					fmt.Fprintf(os.Stderr, "[rick-web] easy close: mark tasks success failed (job %s): %v\n", jobID, err)
				}
			}
		}
	}
	m.updateStatus(entry, SessionStatusClosed, reason)
	m.mu.Lock()
	delete(m.closing, entry.ID)
	m.mu.Unlock()
}

// updateEasyTasksStatus sets the synthetic easy task's status in the job's
// doing/tasks.json (running → success on close). Missing file / parse error
// are logged-and-ignored: close must not fail because of housekeeping.
func updateEasyTasksStatus(rickDir, jobID, status string) error {
	tasksPath := filepath.Join(rickDir, "jobs", jobID, "doing", "tasks.json")
	data, err := os.ReadFile(tasksPath)
	if err != nil {
		return err
	}
	var doc map[string]any
	if err := json.Unmarshal(data, &doc); err != nil {
		return err
	}
	tasks, _ := doc["tasks"].([]any)
	if len(tasks) == 0 {
		return fmt.Errorf("easy tasks.json has no tasks: %s", tasksPath)
	}
	task, _ := tasks[0].(map[string]any)
	if task == nil {
		return fmt.Errorf("easy tasks.json task[0] not an object: %s", tasksPath)
	}
	task["status"] = status
	task["updated_at"] = time.Now().Format(time.RFC3339)
	doc["updated_at"] = time.Now().Format(time.RFC3339)
	out, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(tasksPath, out, 0644)
}

// SessionArchive handles POST /api/sessions/{id}/archive → 204 (idempotent).
//
// Session-level archive is a user-initiated action (no agent judgement): the
// user marks a session done and archives it so it leaves the default session
// list. An active/running session is terminated first (same semantics as
// close — worker killed / background cancelled, status → closed). Already
// closed/error sessions are only flagged. Archived sessions stay queryable
// through GET /api/sessions?archived=true and can be restored via unarchive.
func (m *SessionManager) SessionArchive(w http.ResponseWriter, r *http.Request) {
	entry, ok := m.lookup(w, r)
	if !ok {
		return
	}
	if entry.Status == SessionStatusActive || entry.Status == SessionStatusRunning {
		// Terminate the live session first; the state transition to closed is
		// broadcast by closeSession (reason "archived").
		m.closeSession(entry, "archived")
		if e2, ok2 := m.sessions.Get(entry.ID); ok2 {
			entry = e2
		}
	}
	if entry.Archived {
		// Idempotent; backfill a missing archive timestamp (legacy rows that
		// were persisted with archived=true but no time).
		if entry.ArchivedAt.IsZero() {
			entry.ArchivedAt = time.Now()
			_ = m.sessions.Update(entry)
		}
		w.WriteHeader(http.StatusNoContent)
		return
	}
	entry.Archived = true
	entry.ArchivedAt = time.Now()
	if err := m.sessions.Update(entry); err != nil {
		writeError(w, newWebError(http.StatusInternalServerError, "internal", "archive session: %v", err))
		return
	}
	// Broadcast so live UIs refresh (status may be unchanged for already
	// closed/error sessions; archived flag itself is read via GET).
	m.hub.Publish(SessionStateEvent(entry.ID, entry.Status, "archived"))
	w.WriteHeader(http.StatusNoContent)
}

// SessionUnarchive handles POST /api/sessions/{id}/unarchive → 204
// (idempotent): clears the archived flag so the session returns to the
// default list. Status is untouched (a closed session stays closed — the
// frontend offers Resume to re-run it).
func (m *SessionManager) SessionUnarchive(w http.ResponseWriter, r *http.Request) {
	entry, ok := m.lookup(w, r)
	if !ok {
		return
	}
	if !entry.Archived {
		w.WriteHeader(http.StatusNoContent) // idempotent
		return
	}
	entry.Archived = false
	entry.ArchivedAt = time.Time{}
	if err := m.sessions.Update(entry); err != nil {
		writeError(w, newWebError(http.StatusInternalServerError, "internal", "unarchive session: %v", err))
		return
	}
	m.hub.Publish(SessionStateEvent(entry.ID, entry.Status, "unarchived"))
	w.WriteHeader(http.StatusNoContent)
}

// bgWait returns the background goroutine's done channel (nil if none).
func (m *SessionManager) bgWait(id string) chan struct{} {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.bgWaits[id]
}

// SessionResume handles POST /api/sessions/{id}/resume: closed → active by
// re-spawning with `--session <pi_session_id>` (pi resume semantics).
func (m *SessionManager) SessionResume(w http.ResponseWriter, r *http.Request) {
	entry, ok := m.lookup(w, r)
	if !ok {
		return
	}
	switch entry.Type {
	case SessionTypeDoing:
		writeError(w, newWebError(http.StatusConflict, "state_conflict", "doing sessions are monitor-only; re-run via a new doing session"))
		return
	case SessionTypeDream:
		if dreamMode(entry.Params) == DreamModeBackground {
			writeError(w, newWebError(http.StatusConflict, "state_conflict", "background dream sessions cannot be resumed; start a new one"))
			return
		}
	}
	if entry.Status == SessionStatusActive || entry.Status == SessionStatusRunning {
		writeError(w, newWebError(http.StatusConflict, "state_conflict", "session is %s; nothing to resume", entry.Status))
		return
	}
	ws, ok := m.workspaces.Get(entry.WorkspaceID)
	if !ok {
		writeError(w, newWebError(http.StatusNotFound, "not_found", "workspace %s is no longer registered", entry.WorkspaceID))
		return
	}

	spec := runtime.SpawnSpec{
		SessionID:     entry.ID,
		Dir:           ws.Path,
		MethodFile:    paramString(entry.Params, "_method_file"),
		PromptFile:    paramString(entry.Params, "_prompt_file"),
		SessionIDFlag: entry.PISessionID,
		CreateNew:     false, // --session <id>: load the existing pi session
	}
	worker, err := m.sup.Spawn(spec)
	if err != nil {
		m.updateStatus(entry, SessionStatusError, "resume spawn failed: "+err.Error())
		writeError(w, newWebError(http.StatusInternalServerError, "spawn_failed", "%v", err))
		return
	}
	m.updateStatus(entry, SessionStatusActive, "resumed")
	// Resuming an archived session means the user wants it back in active use:
	// clear the archive marker so it returns to the default session list.
	if e2, ok2 := m.sessions.Get(entry.ID); ok2 && e2.Archived {
		e2.Archived = false
		e2.ArchivedAt = time.Time{}
		if err := m.sessions.Update(e2); err != nil {
			fmt.Fprintf(os.Stderr, "[rick-web] session %s unarchive-on-resume failed: %v\n", entry.ID, err)
		}
	}
	m.pumpWorker(entry.ID, worker)
	writeJSON(w, http.StatusAccepted, map[string]any{"ok": true})
}

// SessionUIResponse handles POST /api/sessions/{id}/ui_response: answer a
// pending extension_ui_request (select/confirm/input/editor dialogs).
func (m *SessionManager) SessionUIResponse(w http.ResponseWriter, r *http.Request) {
	entry, ok := m.lookup(w, r)
	if !ok {
		return
	}
	if entry.Status != SessionStatusActive {
		writeError(w, newWebError(http.StatusConflict, "state_conflict", "session is %s, ui_response requires active", entry.Status))
		return
	}
	worker := m.sup.Get(entry.ID)
	if worker == nil || worker.IsDead() {
		writeError(w, newWebError(http.StatusConflict, "state_conflict", "session worker is not alive"))
		return
	}
	var req struct {
		RequestID string `json:"request_id"`
		Value     any    `json:"value"`
		Confirmed *bool  `json:"confirmed"`
		Cancelled bool   `json:"cancelled"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, newWebError(http.StatusBadRequest, "invalid_params", "decode body: %v", err))
		return
	}
	if req.RequestID == "" {
		writeError(w, newWebError(http.StatusBadRequest, "invalid_params", "request_id must not be empty"))
		return
	}

	client := runtime.NewRpcClient()
	var line []byte
	var err error
	switch {
	case req.Cancelled:
		line, err = client.SendUIResponse(req.RequestID, runtime.UIResponseCancelled, nil)
	case req.Confirmed != nil:
		line, err = client.SendUIResponse(req.RequestID, runtime.UIResponseConfirm, *req.Confirmed)
	case req.Value != nil:
		line, err = client.SendUIResponse(req.RequestID, runtime.UIResponseValue, req.Value)
	default:
		writeError(w, newWebError(http.StatusBadRequest, "invalid_params", "one of value / confirmed / cancelled is required"))
		return
	}
	if err != nil {
		writeError(w, newWebError(http.StatusInternalServerError, "internal", "build ui response: %v", err))
		return
	}
	if err := worker.Send(line); err != nil {
		writeError(w, newWebError(http.StatusInternalServerError, "internal", "send ui response: %v", err))
		return
	}
	writeJSON(w, http.StatusAccepted, map[string]any{"ok": true})
}

// entriesResponse is the GET /api/sessions/{id}/entries body (contract:
// {"entries":[...],"leaf_id":"..."|null}).
type entriesResponse struct {
	Entries []json.RawMessage `json:"entries"`
	LeafID  *string           `json:"leaf_id"`
}

// SessionEntries handles GET /api/sessions/{id}/entries?since=<entryId>&limit=N&before=<entryId>.
// Pagination semantics:
//   - since: incremental cursor (strictly-after entries; active-worker rpc passthrough)
//   - before+limit: newest-first paging for rehydration — returns at most `limit`
//     entries that come strictly BEFORE `before` (time order preserved);
//     `limit` alone returns the most recent `limit` entries.
// Three-tier resolution (job_36 fix): ① active + live worker → rpc get_entries
// passthrough (incremental `since` semantics preserved; before/limit applied
// after); ② worker missing / dead / rpc error (browser refresh, reconnect,
// another device, server restart) → fallback to offline parsing of the pi
// session JSONL — the file is appended in real time, so it is the same source
// of truth pi serves; ③ both paths fail → 404/500. Running/closed/error
// sessions go straight to the offline path.
func (m *SessionManager) SessionEntries(w http.ResponseWriter, r *http.Request) {
	entry, ok := m.lookup(w, r)
	if !ok {
		return
	}
	since := r.URL.Query().Get("since")
	before := r.URL.Query().Get("before")
	limit := 0
	if v := r.URL.Query().Get("limit"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 0 {
			writeError(w, newWebError(http.StatusBadRequest, "invalid_params", "limit must be a non-negative integer"))
			return
		}
		if n > maxEntriesPage {
			n = maxEntriesPage
		}
		limit = n
	}

	if entry.Status == SessionStatusActive {
		if resp, err := m.rpcGetEntries(entry.ID, since); err == nil {
			applyPagination(resp, limit, before)
			writeJSON(w, http.StatusOK, resp)
			return
		}
		// Any rpc failure (including errNoWorker — worker not in this
		// process: restarted server, second browser, post-refresh) falls
		// through to offline parsing below. The JSONL on disk is appended
		// live by pi, so the offline read serves full history.
	}

	// Offline: locate the pi session JSONL by uuid filename prefix under the
	// rick-managed sessions tree (AgentDir() — never a hard-coded HOME path,
	// keeping RICK_PI_AGENT_DIR test isolation intact).
	file, err := findSessionJSONL(entry.PISessionID)
	if err != nil {
		writeError(w, newWebError(http.StatusNotFound, "not_found", "%v", err))
		return
	}
	resp, err := parseSessionEntries(file, since, limit, before)
	if err != nil {
		writeError(w, newWebError(http.StatusInternalServerError, "internal", "parse session entries: %v", err))
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

// maxEntriesPage caps a single paginated page (rehydration request).
const maxEntriesPage = 500

// applyPagination trims resp.Entries per before/limit (time order preserved).
// Works on rpc responses as well as offline lists: before trims everything
// at-or-after the cursor, then limit keeps only the trailing `limit` entries.
func applyPagination(resp *entriesResponse, limit int, before string) {
	if resp == nil || resp.Entries == nil {
		return
	}
	all := resp.Entries
	if before != "" {
		cut := -1
		for i, raw := range all {
			var probe struct {
				ID string `json:"id"`
			}
			if err := json.Unmarshal(raw, &probe); err != nil {
				continue
			}
			if probe.ID == before {
				cut = i
				break
			}
		}
		if cut >= 0 {
			all = all[:cut]
		} // cursor not found → keep the whole page (defensive: don't lose data)
	}
	if limit > 0 && len(all) > limit {
		all = all[len(all)-limit:]
	}
	resp.Entries = all
}

var errNoWorker = errors.New("no live worker for session")

// SessionPromptFiles serves GET /api/sessions/{id}/prompt — the full text that
// was fed to the LLM for this session (method + instance system prompts).
// The file paths were recorded at session creation (_prompt_file /
// _method_file params — see createSessionLocked). Files are read through
// readPromptFile, which enforces the workspace-.rick containment, symlink
// rejection and size cap. Missing recorded paths degrade to not_found so the
// frontend can show an explicit hint instead of a broken block.
func (m *SessionManager) SessionPromptFiles(w http.ResponseWriter, r *http.Request) {
	entry, ok := m.lookup(w, r)
	if !ok {
		return
	}
	ws, ok := m.workspaces.Get(entry.WorkspaceID)
	if !ok {
		writeError(w, newWebError(http.StatusNotFound, "not_found", "workspace %s is not registered", entry.WorkspaceID))
		return
	}
	instancePath, _ := entry.Params["_prompt_file"].(string)
	methodPath, _ := entry.Params["_method_file"].(string)

	instance, err := readPromptFile(ws.Path, instancePath)
	if err != nil {
		writeError(w, err)
		return
	}
	method := ""
	if methodPath != "" {
		method, err = readPromptFile(ws.Path, methodPath)
		if we, isWeb := err.(*WebError); isWeb && we.Status != http.StatusNotFound {
			writeError(w, err)
			return
		}
	}
	writeJSON(w, http.StatusOK, map[string]string{
		"method":   method,
		"instance": instance,
	})
}

// pendingResponses routes rpc responses back to waiting requesters.
type pendingResp struct {
	mu   sync.Mutex
	wait map[string]chan *runtime.RpcEvent
}

func (p *pendingResp) register(id string) chan *runtime.RpcEvent {
	ch := make(chan *runtime.RpcEvent, 1)
	p.mu.Lock()
	p.wait[id] = ch
	p.mu.Unlock()
	return ch
}

func (p *pendingResp) release(id string) {
	p.mu.Lock()
	delete(p.wait, id)
	p.mu.Unlock()
}

func (p *pendingResp) deliver(ev *runtime.RpcEvent) {
	if ev == nil || ev.Type != "response" || ev.ID == "" {
		return
	}
	p.mu.Lock()
	ch, ok := p.wait[ev.ID]
	p.mu.Unlock()
	if ok {
		select {
		case ch <- ev:
		default:
		}
	}
}

// rpcGetEntries sends a get_entries command and waits for the correlated
// response (bounded), returning the contract entries projection.
func (m *SessionManager) rpcGetEntries(sessionID, since string) (*entriesResponse, error) {
	worker := m.sup.Get(sessionID)
	if worker == nil || worker.IsDead() {
		return nil, errNoWorker
	}
	client := runtime.NewRpcClient()
	line, err := client.GetEntries(since)
	if err != nil {
		return nil, err
	}
	// Extract the auto-assigned correlation id from the built command.
	var idOnly struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(line, &idOnly); err != nil {
		return nil, err
	}
	ch := m.pending.register(idOnly.ID)
	defer m.pending.release(idOnly.ID)
	if err := worker.Send(line); err != nil {
		return nil, err
	}
	select {
	case ev := <-ch:
		if ev == nil || !ev.Success {
			return nil, fmt.Errorf("get_entries response failed: %s", ev.Error)
		}
		return decodeEntriesPayload(ev.Data)
	case <-time.After(5 * time.Second):
		return nil, fmt.Errorf("get_entries timed out")
	}
}

// rpcRequest sends an rpc command built by build to the session's worker and
// waits for the correlated response (bounded). It returns the response event
// for the caller to interpret. Errors: errNoWorker when the worker is absent
// or dead; a descriptive error when pi reports success:false or the wait
// times out.
func (m *SessionManager) rpcRequest(sessionID string, build func(*runtime.RpcClient) ([]byte, error)) (*runtime.RpcEvent, error) {
	worker := m.sup.Get(sessionID)
	if worker == nil || worker.IsDead() {
		return nil, errNoWorker
	}
	client := runtime.NewRpcClient()
	line, err := build(client)
	if err != nil {
		return nil, err
	}
	var idOnly struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(line, &idOnly); err != nil {
		return nil, err
	}
	ch := m.pending.register(idOnly.ID)
	defer m.pending.release(idOnly.ID)
	if err := worker.Send(line); err != nil {
		return nil, err
	}
	select {
	case ev := <-ch:
		if ev == nil {
			return nil, fmt.Errorf("rpc response channel closed")
		}
		if !ev.Success {
			return nil, fmt.Errorf("%s failed: %s", ev.Command, ev.Error)
		}
		return ev, nil
	case <-time.After(5 * time.Second):
		return nil, fmt.Errorf("rpc command timed out")
	}
}

// SessionModels handles GET /api/sessions/{id}/models — the list of models
// pi can switch to (get_available_models projection: id/name/provider/
// contextWindow). Worker absent (restarted server / second browser) → 409 so
// the frontend can degrade gracefully.
func (m *SessionManager) SessionModels(w http.ResponseWriter, r *http.Request) {
	entry, ok := m.lookup(w, r)
	if !ok {
		return
	}
	if entry.Status != SessionStatusActive {
		writeError(w, newWebError(http.StatusConflict, "state_conflict", "session is %s, model list requires active", entry.Status))
		return
	}
	ev, err := m.rpcRequest(entry.ID, func(c *runtime.RpcClient) ([]byte, error) { return c.GetAvailableModels() })
	if err != nil {
		if err == errNoWorker {
			writeError(w, newWebError(http.StatusConflict, "state_conflict", "session worker is not alive"))
			return
		}
		writeError(w, newWebError(http.StatusInternalServerError, "internal", "get_available_models: %v", err))
		return
	}
	var payload struct {
		Models []ModelInfo `json:"models"`
	}
	if err := json.Unmarshal(ev.Data, &payload); err != nil {
		writeError(w, newWebError(http.StatusInternalServerError, "internal", "decode models payload: %v", err))
		return
	}
	if payload.Models == nil {
		payload.Models = []ModelInfo{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"models": payload.Models, "current": nil})
}

// SessionSetModel handles POST /api/sessions/{id}/model {provider, model_id}:
// switches the session to the given model via set_model.
func (m *SessionManager) SessionSetModel(w http.ResponseWriter, r *http.Request) {
	entry, ok := m.lookup(w, r)
	if !ok {
		return
	}
	if entry.Status != SessionStatusActive {
		writeError(w, newWebError(http.StatusConflict, "state_conflict", "session is %s, model switch requires active", entry.Status))
		return
	}
	var req struct {
		Provider string `json:"provider"`
		ModelID  string `json:"model_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, newWebError(http.StatusBadRequest, "invalid_params", "decode body: %v", err))
		return
	}
	if req.Provider == "" || req.ModelID == "" {
		writeError(w, newWebError(http.StatusBadRequest, "invalid_params", "provider and model_id are required"))
		return
	}
	ev, err := m.rpcRequest(entry.ID, func(c *runtime.RpcClient) ([]byte, error) {
		return c.SetModel(req.Provider, req.ModelID)
	})
	if err != nil {
		if err == errNoWorker {
			writeError(w, newWebError(http.StatusConflict, "state_conflict", "session worker is not alive"))
			return
		}
		// pi-side rejection (bad model id / provider mismatch) → 400 with
		// the upstream error text so the UI can surface it.
		writeError(w, newWebError(http.StatusBadRequest, "model_rejected", "%v", err))
		return
	}
	writeJSON(w, http.StatusAccepted, map[string]any{"ok": true, "model": string(ev.Data)})
}

// SessionSetThinking handles POST /api/sessions/{id}/thinking {level}:
// sets the reasoning/thinking level for the session's current model.
func (m *SessionManager) SessionSetThinking(w http.ResponseWriter, r *http.Request) {
	entry, ok := m.lookup(w, r)
	if !ok {
		return
	}
	if entry.Status != SessionStatusActive {
		writeError(w, newWebError(http.StatusConflict, "state_conflict", "session is %s, thinking switch requires active", entry.Status))
		return
	}
	var req struct {
		Level string `json:"level"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, newWebError(http.StatusBadRequest, "invalid_params", "decode body: %v", err))
		return
	}
	if req.Level == "" {
		writeError(w, newWebError(http.StatusBadRequest, "invalid_params", "level is required"))
		return
	}
	ev, err := m.rpcRequest(entry.ID, func(c *runtime.RpcClient) ([]byte, error) {
		return c.SetThinkingLevel(req.Level)
	})
	if err != nil {
		if err == errNoWorker {
			writeError(w, newWebError(http.StatusConflict, "state_conflict", "session worker is not alive"))
			return
		}
		writeError(w, newWebError(http.StatusBadRequest, "thinking_rejected", "%v", err))
		return
	}
	writeJSON(w, http.StatusAccepted, map[string]any{"ok": true, "level": string(ev.Data)})
}

// ModelInfo is the wire projection of pi's Model object (rpc.md Types
// section) — the fields the web UI needs for display and switching.
type ModelInfo struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	Provider      string `json:"provider"`
	ContextWindow int64  `json:"contextWindow,omitempty"`
}

// decodeEntriesPayload converts a get_entries response data payload
// ({"entries":[...],"leafId":...}) into the wire projection.
func decodeEntriesPayload(data json.RawMessage) (*entriesResponse, error) {
	var payload struct {
		Entries []json.RawMessage `json:"entries"`
		LeafID  *string           `json:"leafId"`
	}
	if err := json.Unmarshal(data, &payload); err != nil {
		return nil, err
	}
	if payload.Entries == nil {
		payload.Entries = []json.RawMessage{}
	}
	return &entriesResponse{Entries: payload.Entries, LeafID: payload.LeafID}, nil
}

// findSessionJSONL locates the pi session file whose name carries the given
// session uuid (files are named <timestamp>_<uuid>.jsonl under
// AgentDir()/sessions/<encoded-workdir>/). The whole sessions tree is
// searched by prefix — workdir encoding is not assumed stable.
func findSessionJSONL(sessionID string) (string, error) {
	if sessionID == "" {
		return "", fmt.Errorf("session has no pi session id")
	}
	root := filepath.Join(runtime.AgentDir(), "sessions")
	var found string
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil // unreadable subtrees are skipped, not fatal
		}
		name := d.Name()
		if strings.HasSuffix(name, ".jsonl") && strings.Contains(name, sessionID) {
			found = path
			return filepath.SkipAll
		}
		return nil
	})
	if err != nil {
		return "", err
	}
	if found == "" {
		return "", fmt.Errorf("no session file found for %s", sessionID)
	}
	return found, nil
}

// parseSessionEntries reads a pi session JSONL file offline and returns the
// entries after the `since` cursor (append order), plus the leaf id (the
// last appended entry — a reasonable leaf for offline reads). The session
// header line (type "session") is excluded, mirroring rpc get_entries.
// When `since` is non-empty but not found among entry ids, the FULL list is
// returned (stale-cursor rebuild semantics).
func parseSessionEntries(file, since string, limit int, before string) (*entriesResponse, error) {
	data, err := os.ReadFile(file)
	if err != nil {
		return nil, err
	}
	all := make([]json.RawMessage, 0, 64)
	var leaf string
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimRight(line, "\r")
		if strings.TrimSpace(line) == "" {
			continue
		}
		var probe struct {
			Type string `json:"type"`
			ID   string `json:"id"`
		}
		if err := json.Unmarshal([]byte(line), &probe); err != nil {
			continue // corrupt line: skip (listing resilience)
		}
		if probe.Type == "session" {
			continue // header
		}
		all = append(all, json.RawMessage(line))
		if probe.ID != "" {
			leaf = probe.ID
		}
	}

	resp := &entriesResponse{Entries: all}
	if leaf != "" {
		resp.LeafID = &leaf
	}
	if since == "" {
		applyPagination(resp, limit, before)
		return resp, nil
	}

	// `since` cursor: return only entries strictly after it. When the cursor
	// id does not exist in this file (stale cursor from a previous server run
	// or a compaction boundary), degrade to the full list — the caller
	// rebuilds full state instead of silently seeing an empty tail.
	for i, raw := range all {
		var probe struct {
			ID string `json:"id"`
		}
		if err := json.Unmarshal(raw, &probe); err != nil {
			continue
		}
		if probe.ID == since {
			resp.Entries = all[i+1:]
			applyPagination(resp, limit, before)
			return resp, nil
		}
	}
	applyPagination(resp, limit, before)
	return resp, nil // cursor not found → full list
}

// pumpWorker relays a worker's event stream into the Hub for the session's
// lifetime: every rpc event becomes a session_event envelope; agent_settled
// additionally becomes a session_state idle hint; worker death (not caused
// by user close) transitions the session to closed.
func (m *SessionManager) pumpWorker(sessionID string, worker *runtime.Worker) {
	go func() {
		for ev := range worker.Events() {
			if ev == nil {
				continue
			}
			m.pending.deliver(ev)
			m.hub.Publish(SessionEvent(sessionID, ev.Raw, ev))
			// Server-authoritative streaming state（SessionEntry.Busy）：agent
			// 开始/结束回合时更新并广播，前端输入区（发送 vs 终止/steer）据此
			// 渲染，不依赖客户端事件重放推断。
			switch ev.Type {
			// 回合边界（pi 常驻 --mode rpc 会话的权威信号）：turn_start → busy，
			// turn_end → idle（agent 回复完等待用户输入）。实测真实会话事件流里
			// agent_settled 在回合结束**并不发送**（74s 窗口 0 次、turn_start/turn_end
			// 7/7 成对），旧实现只认 agent_start/agent_settled → 回合结束后 busy
			// 永久卡 true（用户实测：AI 已等待输入，UI 仍显示「生产中」+终止按钮）。
			case "turn_start", "agent_start":
				m.setBusy(sessionID, true, ev.Type)
			case "turn_end", "agent_settled":
				m.setBusy(sessionID, false, ev.Type)
			case "agent_end":
				if !agentEndWillRetry(ev) {
					m.setBusy(sessionID, false, "agent_end")
				}
			}
		}
		// Channel closed: worker terminated. Distinguish user close (closing
		// map, already handled) from a crash (transition to error here —
		// 终止/中断语义：worker 异常退出=error，前端显示 Resume 恢复；
		// closed 保留给主动 close）。
		m.mu.Lock()
		userClosing := m.closing[sessionID]
		m.mu.Unlock()
		if !userClosing {
			if entry, ok := m.sessions.Get(sessionID); ok && entry.Status == SessionStatusActive {
				m.markWorkerLost(entry)
			}
			m.sup.Remove(sessionID)
		}
	}()
}

// setBusy records the server-authoritative streaming state for a session and
// broadcasts it as a session_state envelope carrying "busy". The flag lives
// only in memory (SessionEntry.Busy, json:"-") — a restart kills workers, so
// there is nothing meaningful to restore.
func (m *SessionManager) setBusy(sessionID string, busy bool, reason string) {
	entry, ok := m.sessions.Get(sessionID)
	if !ok {
		return
	}
	if entry.Busy == busy {
		return // 幂等：重复事件不产生额外广播
	}
	entry.Busy = busy
	_ = m.sessions.Update(entry)
	m.hub.Publish(SessionBusyEvent(sessionID, entry.Status, busy, reason))
}

// agentEndWillRetry reports ev.willRetry==true (agent_end is not terminal when
// the runtime will retry the turn — busy must stay true).
func agentEndWillRetry(ev *runtime.RpcEvent) bool {
	if ev == nil || len(ev.Raw) == 0 {
		return false
	}
	var probe struct {
		WillRetry bool `json:"willRetry"`
	}
	if json.Unmarshal(ev.Raw, &probe) != nil {
		return false
	}
	return probe.WillRetry
}

// markWorkerLost 标记会话为 error（终止/中断语义）：worker 失活（崩溃/心跳超时/
// 重启遗留/发送管道断裂）时调用，前端收到 session_state 后切换到 error 态并显示
// Resume 恢复引导。幂等：仅当当前 status 为 active/running 时生效（已 error/closed 不覆盖）。
func (m *SessionManager) markWorkerLost(entry SessionEntry) {
	if entry.Status != SessionStatusActive && entry.Status != SessionStatusRunning {
		return
	}
	m.updateStatus(entry, SessionStatusError, "worker lost")
}

// updateStatus persists a status transition and broadcasts it on the hub.
func (m *SessionManager) updateStatus(entry SessionEntry, status, reason string) {
	entry.Status = status
	if status == SessionStatusClosed || status == SessionStatusError {
		entry.ClosedAt = time.Now()
	}
	if err := m.sessions.Update(entry); err != nil {
		// Registry persistence failure is logged via stderr; the in-memory
		// transition still proceeds (the hub event is the live truth).
		fmt.Fprintf(os.Stderr, "[rick-web] session %s status update failed: %v\n", entry.ID, err)
	}
	m.hub.Publish(SessionStateEvent(entry.ID, status, reason))
}

// lookup resolves the {id} path segment to a registry entry (404 on miss).
func (m *SessionManager) lookup(w http.ResponseWriter, r *http.Request) (SessionEntry, bool) {
	id := sessionIDFromPath(r)
	if id == "" {
		writeError(w, newWebError(http.StatusBadRequest, "invalid_params", "missing session id"))
		return SessionEntry{}, false
	}
	entry, ok := m.sessions.Get(id)
	if !ok {
		writeError(w, newWebError(http.StatusNotFound, "not_found", "session %s is not registered", id))
		return SessionEntry{}, false
	}
	return entry, true
}

// ---- interactive prompt preparation (per session type) ----

// prep is the artifact bundle an interactive spawn needs.
type prep struct {
	promptFile string
	methodFile string
	persistDir string // CLI-compat session_id file location ("" = none)
	title      string

	// jobID of the rick job this interactive session drives (easy: close-time
	// tasks.json status flip; empty for other types).
	JobID string
}

// prepareInteractive produces the prompt artifacts for a session type using
// the builder/prompt path-parameterized exports (NEVER handler.XxxIn — those
// launch the interactive CLI and would steal the terminal).
func (m *SessionManager) prepareInteractive(rickDir, sessionType string, params map[string]any) (*prep, error) {
	pb := builder.NewPIBuilder()
	switch sessionType {
	case SessionTypePlan:
		requirement := paramString(params, "requirement")
		jobID := paramString(params, "job")
		if jobID == "" {
			var err error
			jobID, err = scanNextJobID(rickDir)
			if err != nil {
				return nil, err
			}
		}
		jobPlanDir := filepath.Join(rickDir, "jobs", jobID, "plan")
		if err := os.MkdirAll(jobPlanDir, 0755); err != nil {
			return nil, fmt.Errorf("create plan dir: %w", err)
		}
		promptFile, method, err := pb.SavePlanPrompt(requirement, jobPlanDir, rickDir)
		if err != nil {
			return nil, fmt.Errorf("build plan prompt: %w", err)
		}
		methodFile, err := writeMethodFile(jobPlanDir, method)
		if err != nil {
			return nil, err
		}
		return &prep{promptFile: promptFile, methodFile: methodFile, persistDir: jobPlanDir, title: "plan " + jobID}, nil

	case SessionTypeEasy:
		requirement := paramString(params, "requirement")
		ctxPath := paramString(params, "ctx_path")
		if ctxPath != "" {
			if _, err := os.Stat(ctxPath); err != nil {
				return nil, newWebError(http.StatusBadRequest, "invalid_params", "ctx_path does not exist: %s", ctxPath)
			}
		}
		jobID, err := scanNextJobID(rickDir)
		if err != nil {
			return nil, err
		}
		doingDir := filepath.Join(rickDir, "jobs", jobID, "doing")
		if err := os.MkdirAll(doingDir, 0755); err != nil {
			return nil, fmt.Errorf("create doing dir: %w", err)
		}
		// Mirror StartEasySession's artifacts (requirement + synthetic
		// tasks.json) so dream/job tooling still discovers the easy job.
		if err := os.WriteFile(filepath.Join(doingDir, "requirement.md"), []byte(requirement), 0644); err != nil {
			return nil, fmt.Errorf("write requirement: %w", err)
		}
		if err := writeEasyTasksJSON(doingDir); err != nil {
			return nil, fmt.Errorf("write tasks.json: %w", err)
		}
		promptFile, method, _, err := pb.SaveEasyPrompt(jobID, requirement, rickDir, ctxPath)
		if err != nil {
			return nil, fmt.Errorf("build easy prompt: %w", err)
		}
		methodFile, err := writeMethodFile(doingDir, method)
		if err != nil {
			return nil, err
		}
		// jobID 经 prep.JobID 携带——createSessionLocked 会写入 entry.Params
		// （reserved _job_id，wire 投影过滤）：SessionClose 时据此把 easy 的
		// tasks.json 标 success（dream 只学完成的 job）。
		return &prep{promptFile: promptFile, methodFile: methodFile, persistDir: doingDir, title: "easy " + jobID, JobID: jobID}, nil

	case SessionTypeCtrl:
		jobID := paramString(params, "job")
		doingDir := filepath.Join(rickDir, "jobs", jobID, "doing")
		if _, err := os.Stat(doingDir); os.IsNotExist(err) {
			return nil, newWebError(http.StatusNotFound, "not_found", "doing directory not found for job %s", jobID)
		}
		promptFile, method, err := pb.SaveCtrlPrompt(jobID, rickDir)
		if err != nil {
			return nil, fmt.Errorf("build ctrl prompt: %w", err)
		}
		methodFile, err := writeMethodFile(doingDir, method)
		if err != nil {
			return nil, err
		}
		return &prep{promptFile: promptFile, methodFile: methodFile, title: "ctrl " + jobID}, nil

	case SessionTypeHumanLoop:
		topic := paramString(params, "topic")
		draftDir, rfcDir, loopDir, err := handler.PrepareHumanLoopDirsIn(rickDir)
		if err != nil {
			return nil, err
		}
		mainFile, _, _, _, method, err := pb.SaveHumanLoopPrompt(topic, rfcDir, draftDir, loopDir)
		if err != nil {
			return nil, fmt.Errorf("build human-loop prompt: %w", err)
		}
		methodFile, err := writeMethodFile(loopDir, method)
		if err != nil {
			return nil, err
		}
		return &prep{promptFile: mainFile, methodFile: methodFile, persistDir: loopDir, title: "human-loop " + filepath.Base(loopDir)}, nil

	case SessionTypeLearning:
		jobID := paramString(params, "job")
		lp, err := buildLearningParams(rickDir, jobID)
		if err != nil {
			return nil, err
		}
		promptFile, method, err := pb.SaveLearningPrompt(lp)
		if err != nil {
			return nil, fmt.Errorf("build learning prompt: %w", err)
		}
		methodFile, err := writeMethodFile(lp.LearningDir, method)
		if err != nil {
			return nil, err
		}
		return &prep{promptFile: promptFile, methodFile: methodFile, title: "learning " + jobID}, nil

	case SessionTypeDream:
		// interactive dream: same prompt as the CLI dream run
		jobNum := DreamDefaultJobNum
		if v, ok := paramInt(params, "job_num"); ok {
			jobNum = v
		}
		// 无待学习 job → 拒绝创建（前端已预检并显示明确提示「已无可学习的 job」，
		// 此处是 API 兜底）。dream 只学习 tasks 全部 success 的已完成 job；已完成
		// 但已 dream 过的（dream_run_{job}_log.md 存在）不再重复学习。
		jobIDs := workspace.SelectPendingJobs(rickDir, jobNum)
		if len(jobIDs) == 0 {
			return nil, newWebError(http.StatusConflict, "no_pending_jobs",
				"no jobs left to dream: every completed job has already been dreamed (see Jobs → Archived); interrupted or unfinished jobs are not eligible")
		}
		promptFile, method, err := pb.SaveDreamPrompt(jobIDs, rickDir)
		if err != nil {
			return nil, fmt.Errorf("build dream prompt: %w", err)
		}
		dreamDir := filepath.Join(rickDir, workspace.DreamDirName)
		methodFile, err := writeMethodFile(dreamDir, method)
		if err != nil {
			return nil, err
		}
		return &prep{promptFile: promptFile, methodFile: methodFile, title: fmt.Sprintf("dream ×%d", len(jobIDs))}, nil
	}
	return nil, newWebError(http.StatusBadRequest, "invalid_params", "unknown session type %q", sessionType)
}

// writeMethodFile persists the method-layer SOP text next to the prompt so
// SpawnSpec can inject it via --append-system-prompt (doing writes method to
// a temp file; web sessions keep it alongside the prompt for resume reuse).
func writeMethodFile(dir, method string) (string, error) {
	if method == "" {
		return "", nil
	}
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", fmt.Errorf("create method dir: %w", err)
	}
	p := filepath.Join(dir, "method.md")
	if err := os.WriteFile(p, []byte(method), 0644); err != nil {
		return "", fmt.Errorf("write method file: %w", err)
	}
	return p, nil
}

// scanNextJobID mirrors handler.nextJobIDIn (unexported): scan <rickDir>/jobs
// for the highest job_N and return N+1 (job_1 when absent).
func scanNextJobID(rickDir string) (string, error) {
	jobsDir := filepath.Join(rickDir, workspace.JobsDirName)
	entries, err := os.ReadDir(jobsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return "job_1", nil
		}
		return "", fmt.Errorf("read jobs directory: %w", err)
	}
	maxN := 0
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		var n int
		if _, err := fmt.Sscanf(e.Name(), "job_%d", &n); err == nil && n > 0 && n <= 9999 && n > maxN {
			maxN = n
		}
	}
	return fmt.Sprintf("job_%d", maxN+1), nil
}

// buildLearningParams assembles builder.LearningParams from the job's
// on-disk state (a path-parameterized port of handler's unexported
// collectExecutionDataIn + buildLearningPrompt pairing).
func buildLearningParams(rickDir, jobID string) (builder.LearningParams, error) {
	jobDir := filepath.Join(rickDir, "jobs", jobID)
	doingDir := filepath.Join(jobDir, "doing")
	if _, err := os.Stat(doingDir); os.IsNotExist(err) {
		return builder.LearningParams{}, newWebError(http.StatusNotFound, "not_found", "doing directory not found for job %s (has it been executed?)", jobID)
	}
	learningDir := filepath.Join(jobDir, "learning")
	if err := os.MkdirAll(learningDir, 0755); err != nil {
		return builder.LearningParams{}, fmt.Errorf("create learning dir: %w", err)
	}

	lp := builder.LearningParams{
		JobID:        jobID,
		RickDir:      rickDir,
		LearningDir:  learningDir,
		DebugContent: workspace.LoadDebugContext(doingDir),
		DebugDir:     filepath.Join(doingDir, "debug"),
	}

	// tasks.json is required (mirrors collectExecutionDataIn).
	tasksJSONPath := filepath.Join(doingDir, "tasks.json")
	if _, err := os.Stat(tasksJSONPath); err != nil {
		return builder.LearningParams{}, newWebError(http.StatusNotFound, "not_found", "tasks.json not found: %s", tasksJSONPath)
	}
	tj, err := workspace.LoadTasksJSON(tasksJSONPath)
	if err != nil {
		return builder.LearningParams{}, fmt.Errorf("load tasks.json: %w", err)
	}
	for _, task := range tj.Tasks {
		lp.TaskResults = append(lp.TaskResults, builder.LearningResult{
			TaskID:     task.TaskID,
			TaskName:   task.TaskName,
			Status:     task.Status,
			CommitHash: task.CommitHash,
			Attempts:   task.Attempts,
		})
	}

	// plan task files
	if files, err := filepath.Glob(filepath.Join(jobDir, "plan", "task*.md")); err == nil {
		lp.TaskMDPaths = files
	}

	// native trajectory: subagent artifacts + doing parent session id
	repoRoot := filepath.Dir(rickDir)
	if metas, err := filepath.Glob(filepath.Join(repoRoot, ".pi", "subagents", "artifacts", "*_meta.json")); err == nil {
		lp.ActPathFiles = append(lp.ActPathFiles, metas...)
	}
	if sid, err := os.ReadFile(filepath.Join(doingDir, "session_id")); err == nil && len(strings.TrimSpace(string(sid))) > 0 {
		lp.ActPathFiles = append(lp.ActPathFiles, "session:"+strings.TrimSpace(string(sid)))
	}
	return lp, nil
}

// writeEasyTasksJSON mirrors handler.writeEasyTasksJSON (unexported): a
// synthetic success task so dream discovers the easy job.
func writeEasyTasksJSON(doingDir string) error {
	now := time.Now().Format(time.RFC3339)
	doc := map[string]any{
		"version":    "1.0",
		"created_at": now,
		"updated_at": now,
		"tasks": []map[string]any{{
			"task_id":      "easy_session",
			"task_name":    "Easy Mode Session",
			"task_file":    "",
			// 会话进行中（用户 close 时才标 success——dream 只学 tasks 全 success
			// 的 job；旧值创建即 success，导致**中断的 easy job 也被 dream 学习**
			// 并归档（用户实测）。CLI 侧 handler.writeEasyTasksJSON 同病，另行修复。
			"status":       "running",
			"dependencies": []string{},
			"attempts":     1,
			"created_at":   now,
			"updated_at":   now,
		}},
	}
	data, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(doingDir, "tasks.json"), data, 0644)
}

// ---- small param helpers ----

func paramString(params map[string]any, key string) string {
	v, ok := params[key]
	if !ok || v == nil {
		return ""
	}
	s, ok := v.(string)
	if !ok {
		return ""
	}
	return s
}

func paramInt(params map[string]any, key string) (int, bool) {
	v, ok := params[key]
	if !ok || v == nil {
		return 0, false
	}
	switch n := v.(type) {
	case float64:
		return int(n), true
	case int:
		return n, true
	case int64:
		return int(n), true
	}
	return 0, false
}

func cloneParams(params map[string]any) map[string]any {
	out := make(map[string]any, len(params))
	for k, v := range params {
		out[k] = v
	}
	return out
}

func orDefault(s, def string) string {
	if s == "" {
		return def
	}
	return s
}

func dreamMode(params map[string]any) string {
	if m := paramString(params, "mode"); m != "" {
		return m
	}
	return DreamModeBackground
}

// newUUID generates a v4 uuid (crypto/rand; mirrors handler.generateUUID —
// unexported there, duplicated here to stay in the write domain).
func newUUID() (string, error) {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		b[0:4], b[4:6], b[6:8], b[8:10], b[10:16]), nil
}
