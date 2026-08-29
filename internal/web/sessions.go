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
	closing map[string]bool                // sessions closed by the user (worker death is not a crash)
	bgWaits map[string]chan struct{}       // closed when the background goroutine exits
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
	}
	if !e.ClosedAt.IsZero() {
		closed := e.ClosedAt
		info.ClosedAt = &closed
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
						"kind":     "doing_progress",
						"job_id":   ev.JobID,
						"task_id":  ev.TaskID,
						"from":     ev.From,
						"to":       ev.To,
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
						"kind":     "dream_progress",
						"phase":    ev.Phase,
						"job_ids":  ev.JobIDs,
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
	out := make([]sessionInfo, 0, len(list))
	for _, e := range list {
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
		writeError(w, newWebError(http.StatusInternalServerError, "internal", "send %s: %v", what, err))
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
	m.updateStatus(entry, SessionStatusClosed, reason)
	m.mu.Lock()
	delete(m.closing, entry.ID)
	m.mu.Unlock()
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

// SessionEntries handles GET /api/sessions/{id}/entries?since=<entryId>.
// Active sessions relay rpc get_entries; closed sessions parse the pi
// session JSONL offline (no spawn).
func (m *SessionManager) SessionEntries(w http.ResponseWriter, r *http.Request) {
	entry, ok := m.lookup(w, r)
	if !ok {
		return
	}
	since := r.URL.Query().Get("since")

	if entry.Status == SessionStatusActive {
		if resp, err := m.rpcGetEntries(entry.ID, since); err == nil {
			writeJSON(w, http.StatusOK, resp)
			return
		} else if !errors.Is(err, errNoWorker) {
			// Fall through to offline parsing on rpc failure — the session
			// file on disk is the same source of truth pi serves.
			_ = err
		} else {
			writeError(w, newWebError(http.StatusConflict, "state_conflict", "session worker is not alive"))
			return
		}
	}

	// Offline: locate the pi session JSONL by uuid filename prefix under the
	// rick-managed sessions tree (AgentDir() — never a hard-coded HOME path,
	// keeping RICK_PI_AGENT_DIR test isolation intact).
	file, err := findSessionJSONL(entry.PISessionID)
	if err != nil {
		writeError(w, newWebError(http.StatusNotFound, "not_found", "%v", err))
		return
	}
	resp, err := parseSessionEntries(file, since)
	if err != nil {
		writeError(w, newWebError(http.StatusInternalServerError, "internal", "parse session entries: %v", err))
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

var errNoWorker = errors.New("no live worker for session")

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
func parseSessionEntries(file, since string) (*entriesResponse, error) {
	data, err := os.ReadFile(file)
	if err != nil {
		return nil, err
	}
	resp := &entriesResponse{Entries: []json.RawMessage{}}
	var sinceSeen bool = since == ""
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
		if !sinceSeen {
			if probe.ID == since {
				sinceSeen = true
			}
			continue
		}
		resp.Entries = append(resp.Entries, json.RawMessage(line))
		if probe.ID != "" {
			leaf = probe.ID
		}
	}
	if leaf != "" {
		resp.LeafID = &leaf
	}
	return resp, nil
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
			if ev.Type == "agent_settled" {
				m.hub.Publish(SessionStateEvent(sessionID, SessionStatusActive, "agent_settled"))
			}
		}
		// Channel closed: worker terminated. Distinguish user close (closing
		// map, already handled) from a crash (transition to closed here).
		m.mu.Lock()
		userClosing := m.closing[sessionID]
		m.mu.Unlock()
		if !userClosing {
			if entry, ok := m.sessions.Get(sessionID); ok && entry.Status == SessionStatusActive {
				m.updateStatus(entry, SessionStatusClosed, "worker exited: "+worker.Reason())
			}
			m.sup.Remove(sessionID)
		}
	}()
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
	promptFile  string
	methodFile  string
	persistDir  string // CLI-compat session_id file location ("" = none)
	title       string
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
		return &prep{promptFile: promptFile, methodFile: methodFile, persistDir: doingDir, title: "easy " + jobID}, nil

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
		jobIDs := workspace.SelectPendingJobs(rickDir, jobNum)
		if len(jobIDs) == 0 {
			return nil, newWebError(http.StatusConflict, "state_conflict", "no pending completed jobs to dream")
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
			"status":       "success",
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
