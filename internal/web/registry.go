package web

import (
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// Session type constants (api-contract.md Sessions 节)：会话类型 = rick cmd。
const (
	SessionTypePlan      = "plan"
	SessionTypeEasy      = "easy"
	SessionTypeCtrl      = "ctrl"
	SessionTypeHumanLoop = "human-loop"
	SessionTypeLearning  = "learning"
	SessionTypeDream     = "dream"
	SessionTypeDoing     = "doing"
)

// Session status constants (api-contract.md SessionInfo.status).
const (
	SessionStatusActive  = "active"  // 交互型：rpc worker 存活
	SessionStatusRunning = "running" // 后台型：goroutine 执行中
	SessionStatusClosed  = "closed"  // 已关闭（可离线浏览 / resume）
	SessionStatusError   = "error"   // 执行失败
)

// Dream mode constants (api-contract.md dream params).
const (
	DreamModeInteractive = "interactive"
	DreamModeBackground  = "background" // 默认
)

// DreamDefaultJobNum is the default dream job_num (api-contract.md).
const DreamDefaultJobNum = 5

// workspaceRegistryVersion is the on-disk schema version of web.json.
const workspaceRegistryVersion = 1

// ValidationError describes a client-input problem with a stable snake_case
// code (api-contract.md error body: {"error":{"code","message"}}). HTTP
// mapping happens in the routes layer (task13).
type ValidationError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

func newValidationError(code, format string, args ...any) *ValidationError {
	return &ValidationError{Code: code, Message: fmt.Sprintf(format, args...)}
}

// WorkspaceEntry is one registered workspace (a directory containing .rick/).
// JSON tags follow api-contract.md Workspaces 节 verbatim.
type WorkspaceEntry struct {
	ID      string    `json:"id"`
	Path    string    `json:"path"`
	Name    string    `json:"name"`
	AddedAt time.Time `json:"added_at"`
}

// WorkspaceRegistry is the machine-level list of registered workspaces,
// persisted atomically to ~/.rick/web.json. Not safe for concurrent use
// without external locking; the HTTP layer serializes mutations.
type WorkspaceRegistry struct {
	path    string
	Version int              `json:"version"`
	Items   []WorkspaceEntry `json:"workspaces"`
}

// LoadWorkspaceRegistry loads the registry from path. A missing file yields
// an empty registry (not an error); a malformed file is an error.
func LoadWorkspaceRegistry(path string) (*WorkspaceRegistry, error) {
	r := &WorkspaceRegistry{path: path, Version: workspaceRegistryVersion}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return r, nil
		}
		return nil, fmt.Errorf("read workspace registry: %w", err)
	}
	if err := json.Unmarshal(data, r); err != nil {
		return nil, fmt.Errorf("parse workspace registry %s: %w", path, err)
	}
	r.path = path
	return r, nil
}

// workspaceID derives the stable workspace id: first 8 hex chars of sha1(path).
func workspaceID(path string) string {
	sum := sha1.Sum([]byte(path))
	return hex.EncodeToString(sum[:])[:8]
}

// Add registers the workspace at path (which must exist and contain a .rick/
// directory). It is idempotent: registering an already-known path returns the
// existing entry with created=false. The created flag lets the HTTP layer
// distinguish 201 (created) from 200 (already present) per api-contract.md.
func (r *WorkspaceRegistry) Add(path, name string) (WorkspaceEntry, bool, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return WorkspaceEntry{}, false, newValidationError("invalid_workspace", "cannot resolve path %q: %v", path, err)
	}
	// The workspace root must exist and contain a .rick directory.
	rickDir := filepath.Join(abs, ".rick")
	if st, err := os.Stat(rickDir); err != nil || !st.IsDir() {
		return WorkspaceEntry{}, false, newValidationError("invalid_workspace", "path %s does not contain a .rick directory", abs)
	}
	// Idempotent: same path returns the existing entry.
	for _, e := range r.Items {
		if e.Path == abs {
			return e, false, nil
		}
	}
	if name == "" {
		name = filepath.Base(abs)
	}
	entry := WorkspaceEntry{
		ID:      workspaceID(abs),
		Path:    abs,
		Name:    name,
		AddedAt: time.Now(),
	}
	r.Items = append(r.Items, entry)
	if err := r.save(); err != nil {
		// Roll back the in-memory append so a failed write does not leave
		// the registry claiming a workspace that is not on disk.
		r.Items = r.Items[:len(r.Items)-1]
		return WorkspaceEntry{}, false, fmt.Errorf("persist workspace registry: %w", err)
	}
	return entry, true, nil
}

// Remove deletes the workspace with the given id. Removing an unknown id is
// an error (not_found). Removing does not touch anything on disk besides the
// registry file itself (api-contract.md: 活跃会话保留)。注销=仅从 rick 移除
// 注册、目录与 job 文件完整保留（可随时重新 Add 回来）。
func (r *WorkspaceRegistry) Remove(id string) error {
	for i, e := range r.Items {
		if e.ID == id {
			r.Items = append(r.Items[:i], r.Items[i+1:]...)
			if err := r.save(); err != nil {
				return fmt.Errorf("persist workspace registry: %w", err)
			}
			return nil
		}
	}
	return newValidationError("not_found", "workspace %s is not registered", id)
}

// Reorder reorders the registered workspaces to match the given id order
// (sidebar drag-sort persistence). The ids must be a permutation of the
// currently registered ids: same length, every id present, no duplicates —
// otherwise a 400 invalid_order error is returned and the order is untouched.
// An empty ids slice is a no-op (nil). Persists atomically with rollback on
// save failure. List() afterwards returns entries in the new order.
func (r *WorkspaceRegistry) Reorder(ids []string) error {
	if len(ids) == 0 {
		return nil
	}
	if len(ids) != len(r.Items) {
		return newValidationError("invalid_order", "expected %d workspace ids, got %d", len(r.Items), len(ids))
	}
	byID := make(map[string]WorkspaceEntry, len(r.Items))
	for _, e := range r.Items {
		byID[e.ID] = e
	}
	seen := make(map[string]bool, len(ids))
	next := make([]WorkspaceEntry, 0, len(ids))
	for _, id := range ids {
		e, ok := byID[id]
		if !ok {
			return newValidationError("invalid_order", "unknown workspace id %q", id)
		}
		if seen[id] {
			return newValidationError("invalid_order", "duplicate workspace id %q", id)
		}
		seen[id] = true
		next = append(next, e)
	}
	old := r.Items
	r.Items = next
	if err := r.save(); err != nil {
		r.Items = old // rollback so a failed write never claims a reordered state
		return fmt.Errorf("persist workspace registry: %w", err)
	}
	return nil
}

// List returns the registered workspaces (oldest first).
func (r *WorkspaceRegistry) List() []WorkspaceEntry {
	out := make([]WorkspaceEntry, len(r.Items))
	copy(out, r.Items)
	return out
}

// Get returns the workspace with the given id and whether it was found.
func (r *WorkspaceRegistry) Get(id string) (WorkspaceEntry, bool) {
	for _, e := range r.Items {
		if e.ID == id {
			return e, true
		}
	}
	return WorkspaceEntry{}, false
}

// save writes the registry atomically (MkdirAll parent + tmp file + rename)
// so a crash never leaves a half-written web.json behind, and a first run on
// a machine without ~/.rick yet does not fail.
func (r *WorkspaceRegistry) save() error {
	if r.path == "" {
		return errors.New("workspace registry has no backing path")
	}
	data, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal workspace registry: %w", err)
	}
	return writeFileAtomic(r.path, data, 0644)
}

// SessionEntry is one persisted session. JSON tags follow api-contract.md
// SessionInfo verbatim; ClosedAt is omitempty because it is absent until the
// session closes. Archived/ArchivedAt record a user-initiated manual archive
// (session-level archive — finished sessions are hidden from the default list
// but queryable via the paginated archived view and restorable).
type SessionEntry struct {
	ID          string         `json:"id"`
	WorkspaceID string         `json:"workspace_id"`
	Type        string         `json:"type"`
	Title       string         `json:"title,omitempty"`
	Params      map[string]any `json:"params"`
	Status      string         `json:"status"`
	PISessionID string         `json:"pi_session_id"`
	CreatedAt   time.Time      `json:"created_at"`
	ClosedAt    time.Time      `json:"closed_at,omitzero"`
	Archived    bool           `json:"archived,omitempty"`
	ArchivedAt  time.Time      `json:"archived_at,omitzero"`

	// Busy = the agent is currently streaming a turn (agent_start seen, not yet
	// settled). Server-authoritative UI state: the frontend must not infer it
	// from client-side event replay (a refresh/reconnect window can drop
	// agent_start → 输入框错误地回到 idle/发送 态). Deliberately NOT persisted
	// (json:"-") — a server restart kills all workers, so a stale busy flag on
	// disk would be a lie; ReconcileOnStart marks those sessions error anyway.
	Busy bool `json:"-"`
}

// sessionRegistryVersion is the on-disk schema version of sessions.json.
const sessionRegistryVersion = 1

// SessionRegistry persists session metadata across rick web restarts, so the
// session list (and resume) survives the server process. As with the
// workspace registry, mutations are serialized by the HTTP layer.
type SessionRegistry struct {
	path    string
	Version int            `json:"version"`
	Items   []SessionEntry `json:"sessions"`
}

// LoadSessionRegistry loads the session registry from path. A missing file
// yields an empty registry; a malformed file is an error.
func LoadSessionRegistry(path string) (*SessionRegistry, error) {
	r := &SessionRegistry{path: path, Version: sessionRegistryVersion}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return r, nil
		}
		return nil, fmt.Errorf("read session registry: %w", err)
	}
	if err := json.Unmarshal(data, r); err != nil {
		return nil, fmt.Errorf("parse session registry %s: %w", path, err)
	}
	r.path = path
	return r, nil
}

// Add appends a new session entry and persists it. The caller supplies the ID
// (uuid) and PISessionID (the pi --session-id used at spawn time); validation
// of type/params must have happened before (ValidateSessionRequest).
func (r *SessionRegistry) Add(entry SessionEntry) error {
	r.Items = append(r.Items, entry)
	if err := r.save(); err != nil {
		r.Items = r.Items[:len(r.Items)-1]
		return fmt.Errorf("persist session registry: %w", err)
	}
	return nil
}

// Update replaces the entry with the same ID (status transitions, title
// changes, ...). Updating an unknown ID is a not_found error.
func (r *SessionRegistry) Update(entry SessionEntry) error {
	for i, e := range r.Items {
		if e.ID == entry.ID {
			r.Items[i] = entry
			if err := r.save(); err != nil {
				return fmt.Errorf("persist session registry: %w", err)
			}
			return nil
		}
	}
	return newValidationError("not_found", "session %s is not registered", entry.ID)
}

// List returns all sessions (oldest first).
func (r *SessionRegistry) List() []SessionEntry {
	out := make([]SessionEntry, len(r.Items))
	copy(out, r.Items)
	return out
}

// Get returns the session with the given id and whether it was found.
func (r *SessionRegistry) Get(id string) (SessionEntry, bool) {
	for _, e := range r.Items {
		if e.ID == id {
			return e, true
		}
	}
	return SessionEntry{}, false
}

// GetByWorkspace returns the sessions anchored to the given workspace id.
func (r *SessionRegistry) GetByWorkspace(workspaceID string) []SessionEntry {
	var out []SessionEntry
	for _, e := range r.Items {
		if e.WorkspaceID == workspaceID {
			out = append(out, e)
		}
	}
	return out
}

func (r *SessionRegistry) save() error {
	if r.path == "" {
		return errors.New("session registry has no backing path")
	}
	data, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal session registry: %w", err)
	}
	return writeFileAtomic(r.path, data, 0644)
}

// writeFileAtomic writes data to path atomically: create parent directories,
// write to a sibling tmp file, then rename over the target. On rename failure
// the tmp file is removed so no residue is left behind.
func writeFileAtomic(path string, data []byte, perm os.FileMode) error {
	if dir := filepath.Dir(path); dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("create parent dir %s: %w", dir, err)
		}
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, perm); err != nil {
		return fmt.Errorf("write %s: %w", tmp, err)
	}
	if err := os.Rename(tmp, path); err != nil {
		os.Remove(tmp)
		return fmt.Errorf("rename %s -> %s: %w", tmp, path, err)
	}
	return nil
}

// ValidateSessionRequest checks a create-session request against the
// api-contract.md params table. Missing optional fields are valid (defaults
// are applied by the session manager when it creates the entry); fields that
// are present but wrong (wrong type, empty required string, bad enum) are
// invalid_params errors.
//
//	params per type:
//	  plan:       requirement (required, non-empty string), job (optional string)
//	  easy:       requirement (required), ctx_path (optional string)
//	  ctrl:       job (required string)
//	  human-loop: topic (required string)
//	  learning:   job (required string)
//	  dream:      job_num (optional positive number, default 5),
//	              mode (optional "interactive"|"background", default background)
//	  doing:      job (required string)
func ValidateSessionRequest(sessionType string, params map[string]any) error {
	strParam := func(key string) (string, bool) {
		v, ok := params[key]
		if !ok || v == nil {
			return "", false
		}
		s, ok := v.(string)
		return s, ok
	}
	requireStr := func(key string) error {
		s, present := strParam(key)
		if !present {
			return newValidationError("invalid_params", "%s.%s must be a string", sessionType, key)
		}
		if s == "" {
			return newValidationError("invalid_params", "%s.%s must not be empty", sessionType, key)
		}
		return nil
	}

	switch sessionType {
	case SessionTypePlan:
		if err := requireStr("requirement"); err != nil {
			return err
		}
		if _, present := params["job"]; present {
			if _, ok := strParam("job"); !ok {
				return newValidationError("invalid_params", "plan.job must be a string")
			}
		}
	case SessionTypeEasy:
		if err := requireStr("requirement"); err != nil {
			return err
		}
		if _, present := params["ctx_path"]; present {
			if _, ok := strParam("ctx_path"); !ok {
				return newValidationError("invalid_params", "easy.ctx_path must be a string")
			}
		}
	case SessionTypeCtrl, SessionTypeLearning, SessionTypeDoing:
		if err := requireStr("job"); err != nil {
			return err
		}
	case SessionTypeHumanLoop:
		if err := requireStr("topic"); err != nil {
			return err
		}
	case SessionTypeDream:
		if v, present := params["job_num"]; present && v != nil {
			f, ok := toFloat(v)
			if !ok || f <= 0 || f != float64(int(f)) {
				return newValidationError("invalid_params", "dream.job_num must be a positive integer")
			}
		}
		if v, present := params["mode"]; present && v != nil {
			s, ok := v.(string)
			if !ok || (s != DreamModeInteractive && s != DreamModeBackground) {
				return newValidationError("invalid_params", "dream.mode must be %q or %q", DreamModeInteractive, DreamModeBackground)
			}
		}
	default:
		return newValidationError("invalid_params", "unknown session type %q", sessionType)
	}
	return nil
}

// toFloat accepts any JSON number flavor (float64 from encoding/json, or
// json.Number) and reports whether it is a plain number.
func toFloat(v any) (float64, bool) {
	switch n := v.(type) {
	case float64:
		return n, true
	case int:
		return float64(n), true
	case int64:
		return float64(n), true
	case json.Number:
		f, err := n.Float64()
		return f, err == nil
	default:
		return 0, false
	}
}
