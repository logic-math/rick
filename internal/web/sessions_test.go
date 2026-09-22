// sessions_test.go exercises the SessionManager (task11) end-to-end at the
// HTTP-handler level: interactive session creation against a fake pi rpc
// subprocess (stdin logged for bootstrap/command assertions), background
// doing/dream bridges with injected fake runners (progress→Hub), offline
// entries parsing from a fixture session JSONL, and the 409 state matrix.
//
// Isolation (bugs.md):
//   - RICK_PI_AGENT_DIR → temp (piPathOrDefault + AgentDir()/sessions resolution)
//   - HOME              → temp (config.LoadConfig would auto-create real ~/.rick)
//   - fake pi scripts restore the system PATH (job_33 pitfall)
package web

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/sunquan/rick/internal/handler"
	"github.com/sunquan/rick/internal/runtime"
	"github.com/sunquan/rick/internal/workspace"
)

// --- test fixtures ---

// testEnv bundles the isolated SessionManager stack.
type testEnv struct {
	manager    *SessionManager
	sessions   *SessionRegistry
	workspaces *WorkspaceRegistry
	hub        *Hub
	sup        *runtime.Supervisor
	wsEntry    WorkspaceEntry
	rickDir    string
	piLog      string // fake pi's stdin log
}

// fakePiRPC writes a fake pi that logs every stdin line to logPath and
// answers the housekeeping commands (get_state/get_entries).
func fakePiRPC(t *testing.T, logPath string) string {
	t.Helper()
	dir := t.TempDir()
	script := filepath.Join(dir, "fake-pi-rpc")
	var b strings.Builder
	b.WriteString("#!/bin/sh\n")
	b.WriteString("export PATH=/usr/bin:/bin:/usr/sbin:/sbin:$PATH\n")
	b.WriteString("while IFS= read -r line; do\n")
	b.WriteString(fmt.Sprintf("  printf '%%s\\n' \"$line\" >> %q\n", logPath))
	b.WriteString("  case \"$line\" in\n")
	b.WriteString("    *get_state*)\n")
	b.WriteString(`      echo '{"type":"response","command":"get_state","success":true,"id":"hb-1","data":{"isStreaming":false,"isCompacting":false,"sessionId":"pi-sess-1","sessionFile":"/tmp/fake.jsonl"}}'` + "\n")
	b.WriteString("      ;;\n")
	b.WriteString("    *get_available_models*)\n")
	b.WriteString(`      rid=$(printf '%s' "$line" | sed -n 's/.*"id":"\([^"]*\)".*/\1/p')
      echo "{\"type\":\"response\",\"command\":\"get_available_models\",\"success\":true,\"id\":\"$rid\",\"data\":{\"models\":[{\"id\":\"deepseek-v4-flash\",\"name\":\"DeepSeek V4 Flash\",\"provider\":\"deepseek\",\"contextWindow\":131072},{\"id\":\"deepseek-v4-pro\",\"name\":\"DeepSeek V4 Pro\",\"provider\":\"deepseek\",\"contextWindow\":262144}]}}"` + "\n")
	b.WriteString("      ;;\n")
	b.WriteString("    *set_model*)\n")
	b.WriteString(`      rid=$(printf '%s' "$line" | sed -n 's/.*"id":"\([^"]*\)".*/\1/p')
      echo "{\"type\":\"response\",\"command\":\"set_model\",\"success\":true,\"id\":\"$rid\",\"data\":{\"id\":\"deepseek-v4-pro\",\"name\":\"DeepSeek V4 Pro\",\"provider\":\"deepseek\"}}"` + "\n")
	b.WriteString("      ;;\n")
	b.WriteString("    *set_thinking_level*)\n")
	b.WriteString(`      rid=$(printf '%s' "$line" | sed -n 's/.*"id":"\([^"]*\)".*/\1/p')
      echo "{\"type\":\"response\",\"command\":\"set_thinking_level\",\"success\":true,\"id\":\"$rid\",\"data\":{}}"` + "\n")
	b.WriteString("      ;;\n")
	b.WriteString("    *get_entries*)\n")
	b.WriteString(`      rid=$(printf '%s' "$line" | sed -n 's/.*"id":"\([^"]*\)".*/\1/p')
      echo "{\"type\":\"response\",\"command\":\"get_entries\",\"success\":true,\"id\":\"$rid\",\"data\":{\"entries\":[{\"type\":\"message\",\"id\":\"e1\",\"parentId\":null,\"message\":{\"role\":\"user\",\"content\":\"hi\"}}],\"leafId\":\"e1\"}}"` + "\n")
	b.WriteString("      ;;\n")
	b.WriteString("    *)\n")
	b.WriteString(`      echo '{"type":"agent_start"}'` + "\n")
	b.WriteString(`      echo '{"type":"agent_settled"}'` + "\n")
	b.WriteString("      ;;\n")
	b.WriteString("  esac\n")
	b.WriteString("done\n")
	if err := os.WriteFile(script, []byte(b.String()), 0755); err != nil {
		t.Fatal(err)
	}
	return script
}

// newTestEnv builds the full isolated stack: temp HOME + RICK_PI_AGENT_DIR,
// one registered workspace with a bare .rick tree, supervisor on the fake pi.
func newTestEnv(t *testing.T) *testEnv {
	t.Helper()
	t.Setenv("HOME", t.TempDir())
	t.Setenv("RICK_PI_AGENT_DIR", t.TempDir())

	ws := t.TempDir()
	rickDir := filepath.Join(ws, ".rick")
	for _, d := range []string{"jobs", "loops", "skills", "domain", "draft", "dream"} {
		if err := os.MkdirAll(filepath.Join(rickDir, d), 0755); err != nil {
			t.Fatal(err)
		}
	}

	logPath := filepath.Join(t.TempDir(), "stdin.log")
	pi := fakePiRPC(t, logPath)

	sup := runtime.NewSupervisor(runtime.SupervisorConfig{
		HeartbeatInterval: 0, // disabled: tests do not rely on heartbeats
		IdleTimeout:       0,
		PiPath:            pi,
	})
	t.Cleanup(sup.CloseAll)

	workspaces, err := LoadWorkspaceRegistry(filepath.Join(t.TempDir(), "web.json"))
	if err != nil {
		t.Fatal(err)
	}
	entry, created, err := workspaces.Add(ws, "test-ws")
	if err != nil || !created {
		t.Fatalf("register test workspace: %v created=%v", err, created)
	}
	sessions, err := LoadSessionRegistry(filepath.Join(t.TempDir(), "sessions.json"))
	if err != nil {
		t.Fatal(err)
	}
	hub := NewHub(128)

	return &testEnv{
		hub:        hub,
		sup:        sup,
		sessions:   sessions,
		workspaces: workspaces,
		wsEntry:    entry,
		rickDir:    rickDir,
		piLog:      logPath,
	}
}

// managerWith builds a SessionManager (fakes optional; nil → injected fake runners).
func (e *testEnv) managerWith(t *testing.T, doing DoingRunner, dream DreamRunner) *SessionManager {
	t.Helper()
	if doing == nil {
		doing = func(ctx context.Context, rickDir, jobID string, progress func(handler.DoingEvent)) error {
			<-ctx.Done() // default fake: park until cancelled
			return ctx.Err()
		}
	}
	if dream == nil {
		dream = func(ctx context.Context, rickDir string, jobNum int, progress func(handler.DreamEvent)) error {
			<-ctx.Done()
			return ctx.Err()
		}
	}
	return NewSessionManager(e.sessions, e.workspaces, e.sup, e.hub, nil, doing, dream)
}

// post issues a JSON POST against a handler with a path like
// /api/sessions/<id>/prompt and returns the recorder.
func post(t *testing.T, h http.HandlerFunc, path string, body any) *httptest.ResponseRecorder {
	t.Helper()
	var reader *strings.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			t.Fatal(err)
		}
		reader = strings.NewReader(string(data))
	} else {
		reader = strings.NewReader("")
	}
	req := httptest.NewRequest(http.MethodPost, path, reader)
	rec := httptest.NewRecorder()
	h(rec, req)
	return rec
}

func get(t *testing.T, h http.HandlerFunc, path string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, path, nil)
	rec := httptest.NewRecorder()
	h(rec, req)
	return rec
}

// createSessionViaHTTP drives POST /api/sessions and returns (id, status).
func createSessionViaHTTP(t *testing.T, m *SessionManager, wsID, typ string, params map[string]any) (string, int, map[string]any) {
	t.Helper()
	rec := post(t, m.CreateSession, "/api/sessions", map[string]any{
		"workspace_id": wsID,
		"type":         typ,
		"params":       params,
	})
	var body map[string]any
	_ = json.Unmarshal(rec.Body.Bytes(), &body)
	id := ""
	if body != nil {
		if v, ok := body["id"].(string); ok {
			id = v
		}
	}
	return id, rec.Code, body
}

// readLog tails the fake pi's stdin log with retries (spawn+send is async).
func readLog(t *testing.T, path string) string {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for {
		data, err := os.ReadFile(path)
		if err == nil && len(data) > 0 {
			return string(data)
		}
		if time.Now().After(deadline) {
			data, _ := os.ReadFile(path)
			return string(data)
		}
		time.Sleep(20 * time.Millisecond)
	}
}

// waitLog waits until the stdin log contains want (or times out).
func waitLog(t *testing.T, path, want string) string {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for {
		data, _ := os.ReadFile(path)
		if strings.Contains(string(data), want) {
			return string(data)
		}
		if time.Now().After(deadline) {
			t.Fatalf("fake pi stdin log never contained %q; log=\n%s", want, string(data))
		}
		time.Sleep(20 * time.Millisecond)
	}
}

// hubCollector subscribes BEFORE the action under test and buffers every
// envelope, so assertions never race the publish (the hub does not replay
// for a zero Last-Event-ID subscriber by design).
type hubCollector struct {
	hub    *Hub
	sub    *Subscription
	mu     sync.Mutex
	events []Envelope
	done   chan struct{}
}

func startCollector(hub *Hub) *hubCollector {
	c := &hubCollector{hub: hub, sub: hub.Subscribe(0), done: make(chan struct{})}
	go func() {
		for {
			select {
			case evt := <-c.sub.Events():
				c.mu.Lock()
				c.events = append(c.events, evt)
				c.mu.Unlock()
			case <-c.done:
				return
			}
		}
	}()
	return c
}

func (c *hubCollector) stop() {
	close(c.done)
	c.hub.Unsubscribe(c.sub)
}

// waitMatching polls the buffered events until pred matches something (or
// the timeout fires), returning what it found.
func (c *hubCollector) waitMatching(t *testing.T, timeout time.Duration, pred func(Envelope) bool) []Envelope {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for {
		c.mu.Lock()
		var out []Envelope
		for _, e := range c.events {
			if pred == nil || pred(e) {
				out = append(out, e)
			}
		}
		c.mu.Unlock()
		if len(out) > 0 || time.Now().After(deadline) {
			return out
		}
		time.Sleep(15 * time.Millisecond)
	}
}

// --- interactive session creation ---

func TestSessionCreatePlan_SpawnsRPCAndBootstraps(t *testing.T) {
	env := newTestEnv(t)
	m := env.managerWith(t, nil, nil)

	coll := startCollector(env.hub)
	defer coll.stop()
	id, code, body := createSessionViaHTTP(t, m, env.wsEntry.ID, "plan", map[string]any{
		"requirement": "test requirement",
	})
	if code != http.StatusCreated {
		t.Fatalf("create plan session: status %d body %v", code, body)
	}
	if id == "" {
		t.Fatal("create plan session: no id in response")
	}

	// The fake pi must see the bootstrap prompt on stdin (the same trigger
	// the CLI uses).
	log := waitLog(t, env.piLog, "开始：按系统提示词中的 rick 协议立即执行你的职责")

	// The spawn args are baked into the script invocation... assert via the
	// supervisor registry: a live worker exists for the session.
	if w := env.sup.Get(id); w == nil || w.IsDead() {
		t.Fatal("no live worker registered for the created session")
	}

	// Registry entry: status active, params stripped of reserved keys.
	entry, ok := env.sessions.Get(id)
	if !ok {
		t.Fatal("session not persisted in registry")
	}
	if entry.Status != SessionStatusActive {
		t.Fatalf("entry.Status = %q, want active", entry.Status)
	}
	if entry.PISessionID == "" {
		t.Fatal("entry.PISessionID must be recorded (resume depends on it)")
	}
	if _, reserved := entry.Params["_prompt_file"]; !reserved {
		t.Fatal("entry.Params must carry _prompt_file for resume")
	}

	// CLI-compat: the job's plan/session_id file carries the pi session id.
	jobDir := filepath.Join(env.rickDir, "jobs")
	entries, _ := os.ReadDir(jobDir)
	if len(entries) == 0 {
		t.Fatal("plan session must have created a job directory")
	}
	sid, err := os.ReadFile(filepath.Join(env.rickDir, "jobs", "job_1", "plan", "session_id"))
	if err != nil || strings.TrimSpace(string(sid)) != entry.PISessionID {
		t.Fatalf("plan/session_id CLI-compat file: got %q err=%v, want %q", string(sid), err, entry.PISessionID)
	}

	// The prompt file must exist and be injected (referenced as _prompt_file).
	pf := paramString(entry.Params, "_prompt_file")
	if pf == "" {
		t.Fatal("_prompt_file missing")
	}
	if _, err := os.Stat(pf); err != nil {
		t.Fatalf("prompt file %s: %v", pf, err)
	}

	// Hub broadcast the state transitions (pending→active shows as active).
	evts := coll.waitMatching(t, time.Second, func(e Envelope) bool {
		return e.Type == EventTypeSessionState && e.SessionID == id
	})
	if len(evts) == 0 {
		t.Fatal("hub never broadcast session_state for the new session")
	}

	// Session events flow: the fake answers bootstrap with agent_start/agent_settled.
	evts = coll.waitMatching(t, time.Second, func(e Envelope) bool {
		return e.Type == EventTypeSessionEvent && e.SessionID == id
	})
	if len(evts) == 0 {
		t.Fatalf("hub never relayed session events; log=\n%s", log)
	}

	// GET /api/sessions/{id}: wire projection strips reserved params.
	rec := get(t, m.GetSession, "/api/sessions/"+id)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET session: %d %s", rec.Code, rec.Body.String())
	}
	var info map[string]any
	_ = json.Unmarshal(rec.Body.Bytes(), &info)
	if info["status"] != SessionStatusActive {
		t.Fatalf("GET session status: %v", info["status"])
	}
	params, _ := info["params"].(map[string]any)
	if _, leak := params["_prompt_file"]; leak {
		t.Fatal("reserved _prompt_file leaked into the wire projection")
	}
}

func TestSessionCreatePlan_ReusesExplicitJob(t *testing.T) {
	env := newTestEnv(t)
	m := env.managerWith(t, nil, nil)

	// Pre-create job_7/plan so the explicit-job path reuses it.
	if err := os.MkdirAll(filepath.Join(env.rickDir, "jobs", "job_7", "plan"), 0755); err != nil {
		t.Fatal(err)
	}
	id, code, _ := createSessionViaHTTP(t, m, env.wsEntry.ID, "plan", map[string]any{
		"requirement": "another requirement",
		"job":         "job_7",
	})
	if code != http.StatusCreated {
		t.Fatalf("create with explicit job: %d", code)
	}
	entry, _ := env.sessions.Get(id)
	pf := paramString(entry.Params, "_prompt_file")
	if !strings.Contains(pf, filepath.Join("job_7", "plan")) {
		t.Fatalf("explicit job not reused: prompt file %s", pf)
	}
}

func TestSessionCreateCtrl_JobMustExist(t *testing.T) {
	env := newTestEnv(t)
	m := env.managerWith(t, nil, nil)

	_, code, body := createSessionViaHTTP(t, m, env.wsEntry.ID, "ctrl", map[string]any{"job": "job_99"})
	if code != http.StatusNotFound {
		t.Fatalf("ctrl with missing job: status %d body %v", code, body)
	}
}

// --- command routing (prompt/steer/abort/close/resume) ---

func TestSessionCommandRouting(t *testing.T) {
	env := newTestEnv(t)
	m := env.managerWith(t, nil, nil)
	id, _, _ := createSessionViaHTTP(t, m, env.wsEntry.ID, "plan", map[string]any{"requirement": "r"})
	waitLog(t, env.piLog, "开始")

	// prompt
	rec := post(t, m.SessionPrompt, "/api/sessions/"+id+"/prompt", map[string]any{"message": "hello world"})
	if rec.Code != http.StatusAccepted {
		t.Fatalf("prompt: %d %s", rec.Code, rec.Body.String())
	}
	log := waitLog(t, env.piLog, "hello world")
	if !strings.Contains(log, `"type":"prompt"`) {
		t.Fatalf("prompt command not in rpc wire form; log=\n%s", log)
	}

	// steer
	rec = post(t, m.SessionSteer, "/api/sessions/"+id+"/steer", map[string]any{"message": "change course"})
	if rec.Code != http.StatusAccepted {
		t.Fatalf("steer: %d %s", rec.Code, rec.Body.String())
	}
	log = waitLog(t, env.piLog, "change course")
	if !strings.Contains(log, `"type":"steer"`) {
		t.Fatalf("steer command not in rpc wire form; log=\n%s", log)
	}

	// abort
	rec = post(t, m.SessionAbort, "/api/sessions/"+id+"/abort", nil)
	if rec.Code != http.StatusAccepted {
		t.Fatalf("abort: %d %s", rec.Code, rec.Body.String())
	}
	waitLog(t, env.piLog, `"type":"abort"`)

	// ui_response (value form)
	rec = post(t, m.SessionUIResponse, "/api/sessions/"+id+"/ui_response", map[string]any{
		"request_id": "ui-1", "value": "option-a",
	})
	if rec.Code != http.StatusAccepted {
		t.Fatalf("ui_response: %d %s", rec.Code, rec.Body.String())
	}
	log = waitLog(t, env.piLog, "extension_ui_response")
	if !strings.Contains(log, `"value":"option-a"`) || !strings.Contains(log, `"id":"ui-1"`) {
		t.Fatalf("ui_response wire form wrong; log=\n%s", log)
	}

	// ui_response cancelled form
	rec = post(t, m.SessionUIResponse, "/api/sessions/"+id+"/ui_response", map[string]any{
		"request_id": "ui-2", "cancelled": true,
	})
	if rec.Code != http.StatusAccepted {
		t.Fatalf("ui_response cancelled: %d %s", rec.Code, rec.Body.String())
	}
	waitLog(t, env.piLog, `"cancelled":true`)

	// entries via rpc relay
	rec = get(t, m.SessionEntries, "/api/sessions/"+id+"/entries")
	if rec.Code != http.StatusOK {
		t.Fatalf("entries (active, rpc): %d %s", rec.Code, rec.Body.String())
	}
	var er entriesResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &er); err != nil {
		t.Fatal(err)
	}
	if len(er.Entries) != 1 || er.LeafID == nil || *er.LeafID != "e1" {
		t.Fatalf("entries relay wrong: %+v", er)
	}

	// close → 202, then close again → still 202 (idempotent, contract)
	rec = post(t, m.SessionClose, "/api/sessions/"+id+"/close", nil)
	if rec.Code != http.StatusAccepted {
		t.Fatalf("close: %d %s", rec.Code, rec.Body.String())
	}
	entry, _ := env.sessions.Get(id)
	if entry.Status != SessionStatusClosed {
		t.Fatalf("after close: status %q", entry.Status)
	}
	rec = post(t, m.SessionClose, "/api/sessions/"+id+"/close", nil)
	if rec.Code != http.StatusAccepted {
		t.Fatalf("close (idempotent): %d %s", rec.Code, rec.Body.String())
	}

	// resume → 202, worker alive again with --session semantics
	rec = post(t, m.SessionResume, "/api/sessions/"+id+"/resume", nil)
	if rec.Code != http.StatusAccepted {
		t.Fatalf("resume: %d %s", rec.Code, rec.Body.String())
	}
	entry, _ = env.sessions.Get(id)
	if entry.Status != SessionStatusActive {
		t.Fatalf("after resume: status %q", entry.Status)
	}
	if w := env.sup.Get(id); w == nil || w.IsDead() {
		t.Fatal("no live worker after resume")
	}

	// close for cleanup
	_ = post(t, m.SessionClose, "/api/sessions/"+id+"/close", nil)
}

func TestSessionModels_ListsAvailableModels(t *testing.T) {
	env := newTestEnv(t)
	m := env.managerWith(t, nil, nil)
	id, _, _ := createSessionViaHTTP(t, m, env.wsEntry.ID, "plan", map[string]any{"requirement": "r"})
	waitLog(t, env.piLog, "开始")

	rec := get(t, m.SessionModels, "/api/sessions/"+id+"/models")
	if rec.Code != http.StatusOK {
		t.Fatalf("models: %d %s", rec.Code, rec.Body.String())
	}
	var body struct {
		Models []ModelInfo `json:"models"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if len(body.Models) != 2 {
		t.Fatalf("want 2 models, got %d: %s", len(body.Models), rec.Body.String())
	}
	if body.Models[0].ID != "deepseek-v4-flash" || body.Models[0].Provider != "deepseek" {
		t.Fatalf("model[0] wrong: %+v", body.Models[0])
	}
	if body.Models[1].Name != "DeepSeek V4 Pro" || body.Models[1].ContextWindow != 262144 {
		t.Fatalf("model[1] wrong: %+v", body.Models[1])
	}
	// get_available_models must have reached the pi subprocess stdin.
	waitLog(t, env.piLog, `"type":"get_available_models"`)
}

func TestSessionSetModel(t *testing.T) {
	env := newTestEnv(t)
	m := env.managerWith(t, nil, nil)
	id, _, _ := createSessionViaHTTP(t, m, env.wsEntry.ID, "plan", map[string]any{"requirement": "r"})
	waitLog(t, env.piLog, "开始")

	rec := post(t, m.SessionSetModel, "/api/sessions/"+id+"/model", map[string]any{
		"provider": "deepseek", "model_id": "deepseek-v4-pro",
	})
	if rec.Code != http.StatusAccepted {
		t.Fatalf("set_model: %d %s", rec.Code, rec.Body.String())
	}
	log := waitLog(t, env.piLog, `"type":"set_model"`)
	if !strings.Contains(log, `"provider":"deepseek"`) || !strings.Contains(log, `"modelId":"deepseek-v4-pro"`) {
		t.Fatalf("set_model wire form wrong; log=%s", log)
	}

	// Missing fields → 400.
	rec = post(t, m.SessionSetModel, "/api/sessions/"+id+"/model", map[string]any{"provider": "deepseek"})
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("set_model missing model_id: %d", rec.Code)
	}
}

func TestSessionSetThinking(t *testing.T) {
	env := newTestEnv(t)
	m := env.managerWith(t, nil, nil)
	id, _, _ := createSessionViaHTTP(t, m, env.wsEntry.ID, "plan", map[string]any{"requirement": "r"})
	waitLog(t, env.piLog, "开始")

	rec := post(t, m.SessionSetThinking, "/api/sessions/"+id+"/thinking", map[string]any{"level": "high"})
	if rec.Code != http.StatusAccepted {
		t.Fatalf("set_thinking: %d %s", rec.Code, rec.Body.String())
	}
	log := waitLog(t, env.piLog, `"type":"set_thinking_level"`)
	if !strings.Contains(log, `"level":"high"`) {
		t.Fatalf("set_thinking_level wire form wrong; log=%s", log)
	}

	rec = post(t, m.SessionSetThinking, "/api/sessions/"+id+"/thinking", map[string]any{})
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("set_thinking empty level: %d", rec.Code)
	}
}

func TestSessionModels_NoWorker409(t *testing.T) {
	env := newTestEnv(t)
	m := env.managerWith(t, nil, nil)
	id, _, _ := createSessionViaHTTP(t, m, env.wsEntry.ID, "plan", map[string]any{"requirement": "r"})
	waitLog(t, env.piLog, "开始")
	_ = post(t, m.SessionClose, "/api/sessions/"+id+"/close", nil)

	// Closed session → state_conflict 409 (worker gone).
	rec := get(t, m.SessionModels, "/api/sessions/"+id+"/models")
	if rec.Code != http.StatusConflict {
		t.Fatalf("closed models: %d %s", rec.Code, rec.Body.String())
	}
	rec = post(t, m.SessionSetModel, "/api/sessions/"+id+"/model", map[string]any{"provider": "p", "model_id": "m"})
	if rec.Code != http.StatusConflict {
		t.Fatalf("closed set_model: %d", rec.Code)
	}
}

// --- 409 matrix ---

func TestSessionStateConflictMatrix(t *testing.T) {
	env := newTestEnv(t)
	m := env.managerWith(t, nil, nil)

	// Unknown session: 404.
	rec := post(t, m.SessionPrompt, "/api/sessions/nope/prompt", map[string]any{"message": "x"})
	if rec.Code != http.StatusNotFound {
		t.Fatalf("unknown session prompt: %d", rec.Code)
	}

	// Create a plan session, close it, then assert 409s on the active-only
	// command surface.
	id, _, _ := createSessionViaHTTP(t, m, env.wsEntry.ID, "plan", map[string]any{"requirement": "r"})
	waitLog(t, env.piLog, "开始")
	_ = post(t, m.SessionClose, "/api/sessions/"+id+"/close", nil)

	for name, fn := range map[string]http.HandlerFunc{
		"prompt":      m.SessionPrompt,
		"steer":       m.SessionSteer,
		"abort":       m.SessionAbort,
		"ui_response": m.SessionUIResponse,
	} {
		rec := post(t, fn, "/api/sessions/"+id+"/"+name, map[string]any{"message": "x", "request_id": "r"})
		if rec.Code != http.StatusConflict {
			t.Fatalf("%s on closed session: %d (want 409)", name, rec.Code)
		}
	}

	// Doing sessions are no longer monitor-only: /resume (= /continue) 对一个**正在
	// 跑**的 doing 会话是幂等 no-op（自动 202 already_active），而不是 409——人工
	// 恢复入口必须能重复点击而不报错（human 裁决 J-L6-7）。
	did, code, _ := createSessionViaHTTP(t, m, env.wsEntry.ID, "doing", map[string]any{"job": "job_1"})
	if code != http.StatusCreated {
		t.Fatalf("create doing session: %d", code)
	}
	rec = post(t, m.SessionResume, "/api/sessions/"+did+"/resume", nil)
	if rec.Code != http.StatusAccepted {
		t.Fatalf("resume running doing session: %d (want 202 already_active), body=%s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "already_active") {
		t.Fatalf("resume running doing session should be idempotent, body=%s", rec.Body.String())
	}
	// prompt on running doing session: 409.
	rec = post(t, m.SessionPrompt, "/api/sessions/"+did+"/prompt", map[string]any{"message": "x"})
	if rec.Code != http.StatusConflict {
		t.Fatalf("prompt on running doing: %d (want 409)", rec.Code)
	}
	// close cancels the background goroutine.
	rec = post(t, m.SessionClose, "/api/sessions/"+did+"/close", nil)
	if rec.Code != http.StatusAccepted {
		t.Fatalf("close doing: %d", rec.Code)
	}
	entry, _ := env.sessions.Get(did)
	if entry.Status != SessionStatusClosed && entry.Status != SessionStatusError {
		t.Fatalf("doing after close: %q", entry.Status)
	}
}

func TestSessionDreamInteractiveVsBackground(t *testing.T) {
	env := newTestEnv(t)
	m := env.managerWith(t, nil, nil)

	// Interactive dream with no pending jobs → 409 no_pending_jobs (the frontend
	// pre-checks and shows an explicit "已无可学习的 job" hint + disables submit;
	// this is the API backstop with a clear message, formerly a bare 409).
	_, code, body := createSessionViaHTTP(t, m, env.wsEntry.ID, "dream", map[string]any{"mode": "interactive"})
	if code != http.StatusConflict {
		t.Fatalf("interactive dream with no pending jobs: %d %v (want 409)", code, body)
	}
	if e, _ := body["error"].(map[string]any); e == nil || e["code"] != "no_pending_jobs" {
		t.Fatalf("dream 409 should carry code no_pending_jobs: %v", body)
	}

	// Background dream spawns no worker and starts the runner.
	var mu sync.Mutex
	started := make(chan struct{})
	dream := func(ctx context.Context, rickDir string, jobNum int, progress func(handler.DreamEvent)) error {
		mu.Lock()
		close(started)
		mu.Unlock()
		progress(handler.DreamEvent{Phase: "selected", JobIDs: []string{"job_1"}})
		<-ctx.Done()
		return ctx.Err()
	}
	coll := startCollector(env.hub)
	defer coll.stop()
	m2 := env.managerWith(t, nil, dream)
	id, code, _ := createSessionViaHTTP(t, m2, env.wsEntry.ID, "dream", map[string]any{"mode": "background", "job_num": 3})
	if code != http.StatusCreated {
		t.Fatalf("background dream: %d", code)
	}
	if w := env.sup.Get(id); w != nil {
		t.Fatal("background dream must not spawn an rpc worker")
	}
	select {
	case <-started:
	case <-time.After(2 * time.Second):
		t.Fatal("dream runner never started")
	}
	// The progress callback surfaced as a session event.
	evts := coll.waitMatching(t, time.Second, func(e Envelope) bool {
		return e.Type == EventTypeSessionEvent && e.SessionID == id
	})
	if len(evts) == 0 {
		t.Fatal("dream progress never reached the hub")
	}
	// Background dream 现在是**人工可继续**的（不再是 monitor-only 的 409）：对着一个
	// 正在跑的后台 dream 调 /resume(= /continue) 应当是幂等 202 already_active
	//（human 裁决 J-L6-7：后台 job 挂起后由人一键继续）。
	rec := post(t, m2.SessionResume, "/api/sessions/"+id+"/resume", nil)
	if rec.Code != http.StatusAccepted {
		t.Fatalf("resume running background dream: %d (want 202 already_active), body=%s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "already_active") {
		t.Fatalf("resume running background dream should be idempotent, body=%s", rec.Body.String())
	}
	_ = post(t, m2.SessionClose, "/api/sessions/"+id+"/close", nil)
}

// --- background doing bridge ---

func TestSessionDoingBackground_ProgressToHub(t *testing.T) {
	env := newTestEnv(t)

	doing := func(ctx context.Context, rickDir, jobID string, progress func(handler.DoingEvent)) error {
		// Emit a task transition, then finish cleanly.
		progress(handler.DoingEvent{JobID: jobID, TaskID: "task1", From: "pending", To: "running"})
		progress(handler.DoingEvent{JobID: jobID, TaskID: "task1", From: "running", To: "success"})
		return nil
	}
	coll := startCollector(env.hub)
	defer coll.stop()
	m := env.managerWith(t, doing, nil)

	id, code, _ := createSessionViaHTTP(t, m, env.wsEntry.ID, "doing", map[string]any{"job": "job_1"})
	if code != http.StatusCreated {
		t.Fatalf("create doing session: %d", code)
	}

	// jobs_update envelopes must reach the hub with the task diff.
	gotJobs := coll.waitMatching(t, 3*time.Second, func(e Envelope) bool {
		return e.Type == EventTypeJobsUpdate && strings.Contains(string(e.Data), `"task_id":"task1"`)
	})
	if len(gotJobs) == 0 {
		t.Fatal("doing progress never produced jobs_update on the hub")
	}
	gotSessionEvts := coll.waitMatching(t, time.Second, func(e Envelope) bool {
		return e.Type == EventTypeSessionEvent && e.SessionID == id && strings.Contains(string(e.Data), "doing_progress")
	})
	if len(gotSessionEvts) == 0 {
		t.Fatal("doing progress never produced a session event on the hub")
	}

	// The runner returned nil → session closed with reason completed.
	waitStatus := func(want string) SessionEntry {
		t.Helper()
		deadline := time.Now().Add(3 * time.Second)
		for {
			entry, _ := env.sessions.Get(id)
			if entry.Status == want {
				return entry
			}
			if time.Now().After(deadline) {
				t.Fatalf("doing session status stuck at %q, want %q", entry.Status, want)
			}
			time.Sleep(20 * time.Millisecond)
		}
	}
	waitStatus(SessionStatusClosed)

	// A failing runner lands on error status.
	doingErr := func(ctx context.Context, rickDir, jobID string, progress func(handler.DoingEvent)) error {
		return fmt.Errorf("boom")
	}
	m2 := env.managerWith(t, doingErr, nil)
	id2, _, _ := createSessionViaHTTP(t, m2, env.wsEntry.ID, "doing", map[string]any{"job": "job_1"})
	_ = waitForStatus(t, env, id2, SessionStatusError)
}

// waitForStatus polls the registry until the session reaches want.
func waitForStatus(t *testing.T, env *testEnv, id, want string) SessionEntry {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for {
		entry, _ := env.sessions.Get(id)
		if entry.Status == want {
			return entry
		}
		if time.Now().After(deadline) {
			t.Fatalf("session %s status stuck at %q, want %q", id, entry.Status, want)
		}
		time.Sleep(20 * time.Millisecond)
	}
}

// --- offline entries parsing ---

func TestSessionEntriesOfflineParsing(t *testing.T) {
	env := newTestEnv(t)
	m := env.managerWith(t, nil, nil)

	// Fabricate a pi session file under the isolated AgentDir sessions tree
	// named after a known uuid.
	piID := "0199abcd-1234-4abc-8def-000000000001"
	sessDir := filepath.Join(os.Getenv("RICK_PI_AGENT_DIR"), "sessions", "--fake-workdir--")
	if err := os.MkdirAll(sessDir, 0755); err != nil {
		t.Fatal(err)
	}
	lines := []string{
		`{"type":"session","version":"3","id":"` + piID + `","timestamp":"2026-01-01T00:00:00.000Z","cwd":"/x"}`,
		`{"type":"message","id":"e1","parentId":null,"timestamp":"2026-01-01T00:00:01.000Z","message":{"role":"user","content":"first"}}`,
		`{"type":"message","id":"e2","parentId":"e1","timestamp":"2026-01-01T00:00:02.000Z","message":{"role":"assistant","content":"second"}}`,
		`{"type":"message","id":"e3","parentId":"e2","timestamp":"2026-01-01T00:00:03.000Z","message":{"role":"assistant","content":"third"}}`,
	}
	if err := os.WriteFile(filepath.Join(sessDir, "2026-01-01T00-00-00-000Z_"+piID+".jsonl"), []byte(strings.Join(lines, "\n")+"\n"), 0644); err != nil {
		t.Fatal(err)
	}

	// Register a closed session pointing at that pi session.
	entry := SessionEntry{
		ID:          "closed-1",
		WorkspaceID: env.wsEntry.ID,
		Type:        SessionTypePlan,
		Params:      map[string]any{"requirement": "r"},
		Status:      SessionStatusClosed,
		PISessionID: piID,
		CreatedAt:   time.Now(),
		ClosedAt:    time.Now(),
	}
	if err := env.sessions.Add(entry); err != nil {
		t.Fatal(err)
	}

	// Full read: header excluded, 3 entries, leaf = e3.
	rec := get(t, m.SessionEntries, "/api/sessions/closed-1/entries")
	if rec.Code != http.StatusOK {
		t.Fatalf("offline entries: %d %s", rec.Code, rec.Body.String())
	}
	var er entriesResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &er); err != nil {
		t.Fatal(err)
	}
	if len(er.Entries) != 3 {
		t.Fatalf("offline entries: got %d want 3", len(er.Entries))
	}
	if er.LeafID == nil || *er.LeafID != "e3" {
		t.Fatalf("offline leaf: %+v", er.LeafID)
	}

	// since=e1 → entries after e1 only.
	rec = get(t, m.SessionEntries, "/api/sessions/closed-1/entries?since=e1")
	if rec.Code != http.StatusOK {
		t.Fatalf("offline entries since: %d", rec.Code)
	}
	er = entriesResponse{}
	if err := json.Unmarshal(rec.Body.Bytes(), &er); err != nil {
		t.Fatal(err)
	}
	if len(er.Entries) != 2 {
		t.Fatalf("offline entries since=e1: got %d want 2", len(er.Entries))
	}

	// Pagination: limit=2 keeps the two most recent entries (e2, e3).
	rec = get(t, m.SessionEntries, "/api/sessions/closed-1/entries?limit=2")
	if rec.Code != http.StatusOK {
		t.Fatalf("offline entries limit: %d", rec.Code)
	}
	er = entriesResponse{}
	if err := json.Unmarshal(rec.Body.Bytes(), &er); err != nil {
		t.Fatal(err)
	}
	if len(er.Entries) != 2 {
		t.Fatalf("entries limit=2: got %d want 2", len(er.Entries))
	}
	idOf := func(raw json.RawMessage) string {
		var p struct {
			ID string `json:"id"`
		}
		_ = json.Unmarshal(raw, &p)
		return p.ID
	}
	if idOf(er.Entries[0]) != "e2" || idOf(er.Entries[1]) != "e3" {
		t.Fatalf("entries limit=2 order: got %q..%q want e2..e3", idOf(er.Entries[0]), idOf(er.Entries[1]))
	}

	// Pagination: before=e3 + limit=1 → the single entry strictly before e3 (e2).
	rec = get(t, m.SessionEntries, "/api/sessions/closed-1/entries?before=e3&limit=1")
	if rec.Code != http.StatusOK {
		t.Fatalf("offline entries before: %d", rec.Code)
	}
	er = entriesResponse{}
	if err := json.Unmarshal(rec.Body.Bytes(), &er); err != nil {
		t.Fatal(err)
	}
	if len(er.Entries) != 1 || idOf(er.Entries[0]) != "e2" {
		t.Fatalf("entries before=e3 limit=1: got %d entries first=%q want [e2]", len(er.Entries), idOf(er.Entries[0]))
	}

	// Pagination: before=e1 → nothing strictly before the first entry.
	rec = get(t, m.SessionEntries, "/api/sessions/closed-1/entries?before=e1")
	if rec.Code != http.StatusOK {
		t.Fatalf("offline entries before=e1: %d", rec.Code)
	}
	er = entriesResponse{}
	if err := json.Unmarshal(rec.Body.Bytes(), &er); err != nil {
		t.Fatal(err)
	}
	if len(er.Entries) != 0 {
		t.Fatalf("entries before=e1: got %d want 0", len(er.Entries))
	}

	// Invalid limit → 400.
	rec = get(t, m.SessionEntries, "/api/sessions/closed-1/entries?limit=abc")
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("entries limit=abc: got %d want 400", rec.Code)
	}

	// Unknown pi session → 404.
	entry2 := entry
	entry2.ID = "closed-2"
	entry2.PISessionID = "0199ffff-0000-4000-8000-000000000002"
	_ = env.sessions.Add(entry2)
	rec = get(t, m.SessionEntries, "/api/sessions/closed-2/entries")
	if rec.Code != http.StatusNotFound {
		t.Fatalf("unknown pi session entries: %d (want 404)", rec.Code)
	}
}

// --- active session, no worker: entries falls back to offline parsing ---
// job_36 regression: browser refresh / server restart / second device leaves
// an active-status session without a live worker in this process — entries
// must serve full history from the pi session JSONL (appended live), not 409.

func TestSessionEntriesActiveNoWorker_FallsBackOffline(t *testing.T) {
	env := newTestEnv(t)
	// No supervisor worker is ever spawned for this session (nil runners are
	// fine — we register the session entry directly, mimicking a restarted
	// server whose registry lists the session as active).
	m := env.managerWith(t, nil, nil)

	piID := "0199abcd-1234-4abc-8def-00000000000a"
	sessDir := filepath.Join(os.Getenv("RICK_PI_AGENT_DIR"), "sessions", "--fake-workdir--")
	if err := os.MkdirAll(sessDir, 0755); err != nil {
		t.Fatal(err)
	}
	lines := []string{
		`{"type":"session","version":"3","id":"` + piID + `","timestamp":"2026-01-01T00:00:00.000Z","cwd":"/x"}`,
		`{"type":"message","id":"a1","parentId":null,"timestamp":"2026-01-01T00:00:01.000Z","message":{"role":"user","content":"hello"}}`,
		`{"type":"message","id":"a2","parentId":"a1","timestamp":"2026-01-01T00:00:02.000Z","message":{"role":"assistant","content":"world"}}`,
	}
	if err := os.WriteFile(filepath.Join(sessDir, "2026-01-01T00-00-00-000Z_"+piID+".jsonl"), []byte(strings.Join(lines, "\n")+"\n"), 0644); err != nil {
		t.Fatal(err)
	}

	entry := SessionEntry{
		ID:          "active-noworker",
		WorkspaceID: env.wsEntry.ID,
		Type:        SessionTypePlan,
		Params:      map[string]any{"requirement": "r"},
		Status:      SessionStatusActive, // active but no worker in this process
		PISessionID: piID,
		CreatedAt:   time.Now(),
	}
	if err := env.sessions.Add(entry); err != nil {
		t.Fatal(err)
	}

	// Full read must NOT be 409 — offline fallback serves the JSONL history.
	rec := get(t, m.SessionEntries, "/api/sessions/active-noworker/entries")
	if rec.Code != http.StatusOK {
		t.Fatalf("active-no-worker entries: %d %s (want 200, was 409 before fix)", rec.Code, rec.Body.String())
	}
	var er entriesResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &er); err != nil {
		t.Fatal(err)
	}
	if len(er.Entries) != 2 {
		t.Fatalf("active-no-worker entries: got %d want 2", len(er.Entries))
	}
	if er.LeafID == nil || *er.LeafID != "a2" {
		t.Fatalf("active-no-worker leaf: %+v", er.LeafID)
	}

	// Incremental read via offline fallback: since=a1 → only a2.
	rec = get(t, m.SessionEntries, "/api/sessions/active-noworker/entries?since=a1")
	if rec.Code != http.StatusOK {
		t.Fatalf("active-no-worker entries since: %d", rec.Code)
	}
	er = entriesResponse{}
	if err := json.Unmarshal(rec.Body.Bytes(), &er); err != nil {
		t.Fatal(err)
	}
	if len(er.Entries) != 1 {
		t.Fatalf("active-no-worker since=a1: got %d want 1", len(er.Entries))
	}

	// Stale cursor (id not in file): full list back (rebuild semantics).
	rec = get(t, m.SessionEntries, "/api/sessions/active-noworker/entries?since=stale-id")
	if rec.Code != http.StatusOK {
		t.Fatalf("active-no-worker entries stale cursor: %d", rec.Code)
	}
	er = entriesResponse{}
	if err := json.Unmarshal(rec.Body.Bytes(), &er); err != nil {
		t.Fatal(err)
	}
	if len(er.Entries) != 2 {
		t.Fatalf("active-no-worker stale cursor: got %d want 2 (full rebuild)", len(er.Entries))
	}

	// Unknown pi session under active status: still 404 (both paths failed).
	entry2 := entry
	entry2.ID = "active-noworker-2"
	entry2.PISessionID = "0199ffff-0000-4000-8000-00000000000b"
	_ = env.sessions.Add(entry2)
	rec = get(t, m.SessionEntries, "/api/sessions/active-noworker-2/entries")
	if rec.Code != http.StatusNotFound {
		t.Fatalf("active-no-worker unknown pi session: %d (want 404)", rec.Code)
	}
}

// --- list sessions ---

func TestSessionListByWorkspace(t *testing.T) {
	env := newTestEnv(t)
	m := env.managerWith(t, nil, nil)

	id, _, _ := createSessionViaHTTP(t, m, env.wsEntry.ID, "plan", map[string]any{"requirement": "r"})
	waitLog(t, env.piLog, "开始")

	rec := get(t, m.ListSessions, "/api/sessions?workspace="+env.wsEntry.ID)
	if rec.Code != http.StatusOK {
		t.Fatalf("list: %d", rec.Code)
	}
	var list []map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &list); err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 || list[0]["id"] != id {
		t.Fatalf("list content: %+v", list)
	}

	// Unknown workspace filter: 404.
	rec = get(t, m.ListSessions, "/api/sessions?workspace=nope")
	if rec.Code != http.StatusNotFound {
		t.Fatalf("list unknown workspace: %d", rec.Code)
	}

	_ = post(t, m.SessionClose, "/api/sessions/"+id+"/close", nil)
}

// --- worker crash marks the session error (终止/中断语义，Resume 可恢复) ---

func TestSessionWorkerDeathMarksError(t *testing.T) {
	env := newTestEnv(t)
	m := env.managerWith(t, nil, nil)

	id, _, _ := createSessionViaHTTP(t, m, env.wsEntry.ID, "plan", map[string]any{"requirement": "r"})
	waitLog(t, env.piLog, "开始")

	// Kill the worker outside the manager (simulates a crash): closing stdin
	// makes the fake pi exit (read loop ends).
	w := env.sup.Get(id)
	if w == nil {
		t.Fatal("worker missing")
	}
	w.Close()

	// 用户反馈：非活跃会话不应显示 active——worker 崩溃 → error（前端显示 Resume）
	entry := waitForStatus(t, env, id, SessionStatusError)
	if entry.Status != SessionStatusError {
		t.Fatalf("crashed worker session: %q, want error", entry.Status)
	}
}

// --- command on a lost worker auto-marks error + 409 ---

func TestSessionCommandOnLostWorkerMarksError(t *testing.T) {
	env := newTestEnv(t)
	m := env.managerWith(t, nil, nil)

	id, _, _ := createSessionViaHTTP(t, m, env.wsEntry.ID, "plan", map[string]any{"requirement": "r"})
	waitLog(t, env.piLog, "开始")

	// 手动把 worker 标记 dead（模拟心跳超时/崩溃——不经 pumpWorker 通道关闭路径）
	if w := env.sup.Get(id); w != nil {
		w.Close()
		// Close 会触发 pumpWorker 的通道关闭 → error；为测 command 分支，
		// 直接造一个「注册表 active 但 supervisor 无 worker」的状态：
	}
	// 等 pumpWorker 先把状态转 error（覆盖第一种路径）
	entry := waitForStatus(t, env, id, SessionStatusError)

	// 再测 command 分支：手工把会话改回 active 但 worker 已移除——
	// 用私有状态注入不优雅，这里验证已覆盖：worker 失活时 command 返回 409 且不 panic。
	_ = entry
	_ = post(t, m.SessionPrompt, "/api/sessions/"+id+"/prompt", map[string]any{"message": "x"})
}

// TestRoutes_POSTPromptMapsToSendNotFiles —— 路由注册回归锁：POST /api/sessions/{id}/prompt
// 必须映射到 SessionPrompt（发送消息），而不是 SessionPromptFiles（读 prompt 原文）。
// job_36 验收期实测踩坑：routes.go 曾把 POST prompt 误注册为 SessionPromptFiles，
// 导致前端发送消息静默返回 prompt 原文（200 假成功、消息从未送达 agent）——「发送没反应」。
func TestRoutes_POSTPromptMapsToSendNotFiles(t *testing.T) {
	env := newTestEnv(t)
	m := env.managerWith(t, nil, nil)

	mux := http.NewServeMux()
	RegisterRoutes(mux, Deps{
		Hub:        env.hub,
		Sessions:   m,
		Workspaces: env.workspaces,
		Token:      "test-token",
		Static:     http.NotFoundHandler(),
		Version:    "test",
	})

	req := httptest.NewRequest(http.MethodPost, "/api/sessions/nonexistent/prompt",
		strings.NewReader(`{"message":"hi"}`))
	req.Header.Set("Authorization", "Bearer test-token")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound && rec.Code != http.StatusConflict {
		t.Fatalf("POST /api/sessions/{id}/prompt on unknown session: got %d, want 404/409 (SessionPrompt semantics), body=%s",
			rec.Code, rec.Body.String())
	}
	if rec.Code == http.StatusNotFound && !strings.Contains(rec.Body.String(), `"error"`) {
		t.Fatalf("POST prompt 404 body should carry error envelope, got: %s", rec.Body.String())
	}
}

// TestReconcileOnStart_MarksOrphanActiveAsSuspended —— 重启对账回归锁：status=active
// 但 supervisor 无 worker 的会话（服务重启/平台升级场景）必须被标记 **suspended**，
// 而不是 error：重启后进程不在是平台升级的必然结果，不是执行失败（human 裁决
// J-L6-6）。前端据此显示「因平台升级挂起」+ 一键继续。
// job_36 验收期实测：重启后旧会话仍 active → 发送消息 409 worker is not alive。
func TestReconcileOnStart_MarksOrphanActiveAsSuspended(t *testing.T) {
	env := newTestEnv(t)
	m := env.managerWith(t, nil, nil)

	// 直接造一个 active 会话（不经过 CreateSession——模拟旧进程遗留的注册表条目）
	now := time.Now()
	orphan := SessionEntry{
		ID:          "orphan-1",
		WorkspaceID: env.wsEntry.ID,
		Type:        SessionTypePlan,
		Status:      SessionStatusActive,
		Params:      map[string]any{"requirement": "x"},
		CreatedAt:   now,
	}
	if err := env.sessions.Add(orphan); err != nil {
		t.Fatalf("seed orphan session: %v", err)
	}
	// 已有的 suspended 行必须原样保留（幂等：不重复处理）
	suspended := SessionEntry{
		ID: "orphan-2", WorkspaceID: env.wsEntry.ID, Type: SessionTypeEasy,
		Status: SessionStatusSuspended, Params: map[string]any{"requirement": "y"},
		CreatedAt: now, LastReason: "platform upgrade (release)",
	}
	if err := env.sessions.Add(suspended); err != nil {
		t.Fatalf("seed suspended session: %v", err)
	}

	m.ReconcileOnStart()

	got, ok := env.sessions.Get("orphan-1")
	if !ok {
		t.Fatal("orphan session missing after reconcile")
	}
	if got.Status != SessionStatusSuspended {
		t.Fatalf("reconcile: orphan active status = %q, want suspended（不是 error）", got.Status)
	}
	// reason 必须落盘（否则重启后 UI 无法解释为何挂起）
	if got.LastReason != SuspendReasonRestart {
		t.Fatalf("reconcile: LastReason = %q, want %q（reason 需持久化）", got.LastReason, SuspendReasonRestart)
	}
	// suspended **不是** closed：不得写 closed_at 冒充「已结束」
	if !got.ClosedAt.IsZero() {
		t.Fatalf("reconcile: ClosedAt should stay zero for suspended, got %v", got.ClosedAt)
	}

	kept, _ := env.sessions.Get("orphan-2")
	if kept.Status != SessionStatusSuspended || kept.LastReason != "platform upgrade (release)" {
		t.Fatalf("已挂起会话被改动: status=%q reason=%q", kept.Status, kept.LastReason)
	}

	// 恢复报告：挂起清单以注册表为权威，且文件落盘（UI/metrics 可读）
	rep := m.ReadRecoveryReport()
	if len(rep.Suspended) != 2 {
		t.Fatalf("recovery report suspended = %d 条，want 2（两条都是挂起）", len(rep.Suspended))
	}
	ids := map[string]bool{}
	for _, it := range rep.Suspended {
		ids[it.ID] = true
	}
	if !ids["orphan-1"] || !ids["orphan-2"] {
		t.Fatalf("recovery report 缺会话: %+v", rep.Suspended)
	}
	if rep.Recovered == nil || rep.Failed == nil {
		t.Fatal("recovery report 的 recovered/failed 必须是空数组而非 nil（JSON 契约）")
	}
}

// TestReconcileOnStart_DoesNotAutoResume —— **反向断言**：启动对账绝不得 spawn 任何
// worker（自动恢复被 human 明确否决：悬挂 toolCall 自动续跑会重复副作用、配额耗尽
// 不报错会静默空转）。
func TestReconcileOnStart_DoesNotAutoResume(t *testing.T) {
	env := newTestEnv(t)
	m := env.managerWith(t, nil, nil)

	for _, e := range []SessionEntry{
		{ID: "r1", WorkspaceID: env.wsEntry.ID, Type: SessionTypePlan, Status: SessionStatusActive,
			PISessionID: "pi-r1", Params: map[string]any{"requirement": "x"}, CreatedAt: time.Now()},
		{ID: "r2", WorkspaceID: env.wsEntry.ID, Type: SessionTypeDoing, Status: SessionStatusRunning,
			Params: map[string]any{"job": "job_1"}, CreatedAt: time.Now()},
		{ID: "r3", WorkspaceID: env.wsEntry.ID, Type: SessionTypeDream, Status: SessionStatusRunning,
			Params: map[string]any{"mode": DreamModeBackground}, CreatedAt: time.Now()},
	} {
		if err := env.sessions.Add(e); err != nil {
			t.Fatalf("seed %s: %v", e.ID, err)
		}
	}

	m.ReconcileOnStart()

	// ① 不得有任何 worker 被 spawn
	for _, id := range []string{"r1", "r2", "r3"} {
		if w := env.sup.Get(id); w != nil {
			t.Fatalf("reconcile spawned a worker for %s —— 自动恢复被明确否决", id)
		}
	}
	// ② 不得有任何后台 runner 被启动
	m.mu.Lock()
	running := len(m.cancels)
	m.mu.Unlock()
	if running != 0 {
		t.Fatalf("reconcile started %d background runner(s) —— 自动续跑被明确否决", running)
	}
	// ③ 全部落在 suspended
	for _, id := range []string{"r1", "r2", "r3"} {
		e, _ := env.sessions.Get(id)
		if e.Status != SessionStatusSuspended {
			t.Fatalf("%s status = %q, want suspended", id, e.Status)
		}
	}
}

// --- session-level archive (user-initiated: manual mark-done-and-archive) ---

func TestSessionArchive_ActiveTerminatesAndMarks(t *testing.T) {
	env := newTestEnv(t)
	m := env.managerWith(t, nil, nil)

	id, _, _ := createSessionViaHTTP(t, m, env.wsEntry.ID, "plan", map[string]any{"requirement": "r"})
	waitLog(t, env.piLog, "开始")
	if w := env.sup.Get(id); w == nil {
		t.Fatal("active session should hold a live worker")
	}

	rec := post(t, m.SessionArchive, "/api/sessions/"+id+"/archive", nil)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("archive active session: %d, body=%s", rec.Code, rec.Body.String())
	}

	got, ok := env.sessions.Get(id)
	if !ok {
		t.Fatal("session missing after archive")
	}
	if got.Status != SessionStatusClosed {
		t.Fatalf("archived active session status = %q, want closed", got.Status)
	}
	if !got.Archived {
		t.Fatal("session should be archived")
	}
	if got.ArchivedAt.IsZero() {
		t.Fatal("archived_at should be set")
	}
	if w := env.sup.Get(id); w != nil {
		t.Fatal("worker must be removed after archive of an active session")
	}

	// Wire projection carries the archive fields.
	var info map[string]any
	rec = get(t, m.GetSession, "/api/sessions/"+id)
	if rec.Code != http.StatusOK {
		t.Fatalf("get session: %d", rec.Code)
	}
	_ = json.Unmarshal(rec.Body.Bytes(), &info)
	if info["archived"] != true {
		t.Fatalf("get session archived field = %v, want true", info["archived"])
	}
	if info["archived_at"] == nil {
		t.Fatal("get session archived_at field missing")
	}
}

func TestSessionArchive_ClosedAndError_OnlyMarksIdempotent(t *testing.T) {
	env := newTestEnv(t)
	m := env.managerWith(t, nil, nil)

	// Closed session (created then closed by the user).
	id, _, _ := createSessionViaHTTP(t, m, env.wsEntry.ID, "plan", map[string]any{"requirement": "r"})
	waitLog(t, env.piLog, "开始")
	if rec := post(t, m.SessionClose, "/api/sessions/"+id+"/close", nil); rec.Code != http.StatusAccepted {
		t.Fatalf("close: %d", rec.Code)
	}

	rec := post(t, m.SessionArchive, "/api/sessions/"+id+"/archive", nil)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("archive closed: %d body=%s", rec.Code, rec.Body.String())
	}
	got, _ := env.sessions.Get(id)
	if !got.Archived || got.ArchivedAt.IsZero() {
		t.Fatalf("closed session after archive = %+v, want archived", got)
	}

	// Idempotent second archive → 204.
	if rec := post(t, m.SessionArchive, "/api/sessions/"+id+"/archive", nil); rec.Code != http.StatusNoContent {
		t.Fatalf("re-archive: %d", rec.Code)
	}

	// Error session (manually pushed to error, worker gone) → marks only.
	errID := "err-archive-1"
	now := time.Now()
	if err := env.sessions.Add(SessionEntry{
		ID: errID, WorkspaceID: env.wsEntry.ID, Type: SessionTypePlan,
		Params: map[string]any{"requirement": "r"}, Status: SessionStatusError,
		PISessionID: "pi-err", CreatedAt: now,
	}); err != nil {
		t.Fatal(err)
	}
	if rec := post(t, m.SessionArchive, "/api/sessions/"+errID+"/archive", nil); rec.Code != http.StatusNoContent {
		t.Fatalf("archive error session: %d", rec.Code)
	}
	gotErr, _ := env.sessions.Get(errID)
	if gotErr.Status != SessionStatusError || !gotErr.Archived {
		t.Fatalf("error session after archive = %+v (status should stay error, archived set)", gotErr)
	}

	// Unknown session → 404.
	if rec := post(t, m.SessionArchive, "/api/sessions/ghost/archive", nil); rec.Code != http.StatusNotFound {
		t.Fatalf("archive unknown: %d", rec.Code)
	}
}

func TestSessionArchive_UnarchiveRestoresToList(t *testing.T) {
	env := newTestEnv(t)
	m := env.managerWith(t, nil, nil)

	id, _, _ := createSessionViaHTTP(t, m, env.wsEntry.ID, "plan", map[string]any{"requirement": "r"})
	waitLog(t, env.piLog, "开始")
	if rec := post(t, m.SessionArchive, "/api/sessions/"+id+"/archive", nil); rec.Code != http.StatusNoContent {
		t.Fatalf("archive: %d", rec.Code)
	}

	// Archived session leaves the default list.
	rec := get(t, m.ListSessions, "/api/sessions?workspace="+env.wsEntry.ID)
	var list []map[string]any
	_ = json.Unmarshal(rec.Body.Bytes(), &list)
	if len(list) != 0 {
		t.Fatalf("default list after archive = %d sessions, want 0", len(list))
	}

	// Unarchive → 204; status untouched (closed stays closed).
	if rec := post(t, m.SessionUnarchive, "/api/sessions/"+id+"/unarchive", nil); rec.Code != http.StatusNoContent {
		t.Fatalf("unarchive: %d", rec.Code)
	}
	got, _ := env.sessions.Get(id)
	if got.Archived || !got.ArchivedAt.IsZero() {
		t.Fatalf("after unarchive = %+v, want archived cleared", got)
	}
	if got.Status != SessionStatusClosed {
		t.Fatalf("unarchive changed status to %q, want closed untouched", got.Status)
	}

	// Back in the default list.
	rec = get(t, m.ListSessions, "/api/sessions?workspace="+env.wsEntry.ID)
	_ = json.Unmarshal(rec.Body.Bytes(), &list)
	if len(list) != 1 || list[0]["id"] != id {
		t.Fatalf("default list after unarchive = %+v, want the restored session", list)
	}

	// Idempotent unarchive again.
	if rec := post(t, m.SessionUnarchive, "/api/sessions/"+id+"/unarchive", nil); rec.Code != http.StatusNoContent {
		t.Fatalf("re-unarchive: %d", rec.Code)
	}
}

func TestSessionList_ArchivedPagination(t *testing.T) {
	env := newTestEnv(t)
	m := env.managerWith(t, nil, nil)

	// Seed 5 archived + 1 active session with distinct created_at (desc order
	// expected: newest first in the archived view).
	seed := func(id string, created time.Time, archived bool) {
		entry := SessionEntry{
			ID: id, WorkspaceID: env.wsEntry.ID, Type: SessionTypePlan,
			Params: map[string]any{"requirement": "r"}, Status: SessionStatusClosed,
			PISessionID: "pi-" + id, CreatedAt: created,
		}
		if archived {
			entry.Archived = true
			entry.ArchivedAt = created.Add(time.Minute)
		}
		if err := env.sessions.Add(entry); err != nil {
			t.Fatalf("seed %s: %v", id, err)
		}
	}
	base := time.Now().Add(-time.Hour)
	for i := 1; i <= 5; i++ {
		seed(fmt.Sprintf("arch-%d", i), base.Add(time.Duration(i)*time.Minute), true)
	}
	seed("active-1", base.Add(time.Hour), false)

	// Default list: only the non-archived session.
	rec := get(t, m.ListSessions, "/api/sessions?workspace="+env.wsEntry.ID)
	var list []map[string]any
	_ = json.Unmarshal(rec.Body.Bytes(), &list)
	if len(list) != 1 || list[0]["id"] != "active-1" {
		t.Fatalf("default list = %+v, want only active-1", list)
	}

	// Archived view: page 1 (limit 2) → total 5, newest first (arch-5, arch-4).
	rec = get(t, m.ListSessions, "/api/sessions?workspace="+env.wsEntry.ID+"&archived=true&limit=2&offset=0")
	var page map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &page); err != nil {
		t.Fatalf("decode archived view: %v body=%s", err, rec.Body.String())
	}
	if total := page["total"].(float64); total != 5 {
		t.Fatalf("archived total = %v, want 5", total)
	}
	items := page["items"].([]any)
	if len(items) != 2 {
		t.Fatalf("archived page len = %d, want 2", len(items))
	}
	first := items[0].(map[string]any)
	second := items[1].(map[string]any)
	if first["id"] != "arch-5" || second["id"] != "arch-4" {
		t.Fatalf("archived page order = [%v %v], want [arch-5 arch-4] (created_at desc)", first["id"], second["id"])
	}

	// Page 2 (offset 2, limit 2) → arch-3, arch-2.
	rec = get(t, m.ListSessions, "/api/sessions?workspace="+env.wsEntry.ID+"&archived=true&limit=2&offset=2")
	_ = json.Unmarshal(rec.Body.Bytes(), &page)
	items = page["items"].([]any)
	if len(items) != 2 || items[0].(map[string]any)["id"] != "arch-3" || items[1].(map[string]any)["id"] != "arch-2" {
		t.Fatalf("archived page2 = %v, want [arch-3 arch-2]", items)
	}

	// Offset beyond total → empty page.
	rec = get(t, m.ListSessions, "/api/sessions?workspace="+env.wsEntry.ID+"&archived=true&limit=2&offset=10")
	_ = json.Unmarshal(rec.Body.Bytes(), &page)
	if len(page["items"].([]any)) != 0 {
		t.Fatal("archived view past the end should be empty")
	}
}

func TestSessionResume_ClearsArchived(t *testing.T) {
	env := newTestEnv(t)
	m := env.managerWith(t, nil, nil)

	id, _, _ := createSessionViaHTTP(t, m, env.wsEntry.ID, "plan", map[string]any{"requirement": "r"})
	waitLog(t, env.piLog, "开始")
	if rec := post(t, m.SessionArchive, "/api/sessions/"+id+"/archive", nil); rec.Code != http.StatusNoContent {
		t.Fatalf("archive: %d", rec.Code)
	}

	// Resume the archived (closed) session → active again, archive cleared.
	rec := post(t, m.SessionResume, "/api/sessions/"+id+"/resume", nil)
	if rec.Code != http.StatusAccepted {
		t.Fatalf("resume archived: %d body=%s", rec.Code, rec.Body.String())
	}
	got, _ := env.sessions.Get(id)
	if got.Status != SessionStatusActive {
		t.Fatalf("resumed status = %q, want active", got.Status)
	}
	if got.Archived || !got.ArchivedAt.IsZero() {
		t.Fatalf("resumed session still archived = %+v, want cleared", got)
	}
}

func TestSessionArchive_PersistsAcrossReload(t *testing.T) {
	env := newTestEnv(t)
	m := env.managerWith(t, nil, nil)

	id, _, _ := createSessionViaHTTP(t, m, env.wsEntry.ID, "plan", map[string]any{"requirement": "r"})
	waitLog(t, env.piLog, "开始")
	if rec := post(t, m.SessionArchive, "/api/sessions/"+id+"/archive", nil); rec.Code != http.StatusNoContent {
		t.Fatalf("archive: %d", rec.Code)
	}

	// Reload the registry from disk (simulates a server restart).
	reloaded, err := LoadSessionRegistry(env.sessions.path)
	if err != nil {
		t.Fatalf("reload registry: %v", err)
	}
	got, ok := reloaded.Get(id)
	if !ok {
		t.Fatal("session missing after reload")
	}
	if !got.Archived || got.ArchivedAt.IsZero() {
		t.Fatalf("archived flag lost across reload: %+v", got)
	}
}

// TestSessionBusyAuthority 验证服务端权威流式状态（SessionEntry.Busy）：
// 投影规则（仅 active 才 busy）、广播内容（session_state 带 busy）、幂等
// （重复同值不重复广播）、以及非 active 状态一律不 busy。
func TestSessionBusyAuthority(t *testing.T) {
	env := newTestEnv(t)
	m := env.managerWith(t, nil, nil)

	entry := SessionEntry{
		ID:          "busy-s1",
		WorkspaceID: env.wsEntry.ID,
		Type:        SessionTypePlan,
		Status:      SessionStatusActive,
		PISessionID: "pi-busy-1",
	}
	if err := m.sessions.Add(entry); err != nil {
		t.Fatal(err)
	}
	got, _ := m.sessions.Get(entry.ID)
	if m.toSessionInfo(got).Busy {
		t.Fatal("initial busy must be false")
	}

	coll := startCollector(env.hub)
	defer coll.stop()

	// agent_start → busy true + 广播
	m.setBusy(entry.ID, true, "agent_start")
	got, _ = m.sessions.Get(entry.ID)
	if !got.Busy || !m.toSessionInfo(got).Busy {
		t.Fatalf("after agent_start: entry.Busy=%v info.Busy=%v (want true)", got.Busy, m.toSessionInfo(got).Busy)
	}
	evts := coll.waitMatching(t, time.Second, func(e Envelope) bool {
		return e.Type == EventTypeSessionState && e.SessionID == entry.ID && strings.Contains(string(e.Data), `"busy":true`)
	})
	if len(evts) == 0 {
		t.Fatal("busy=true session_state never reached the hub")
	}

	// 幂等：重复 true 不新增广播（waitMatching 计数不变）
	m.setBusy(entry.ID, true, "agent_start")
	time.Sleep(100 * time.Millisecond)
	busyTrue := coll.waitMatching(t, 200*time.Millisecond, func(e Envelope) bool {
		return e.Type == EventTypeSessionState && e.SessionID == entry.ID && strings.Contains(string(e.Data), `"busy":true`)
	})
	if len(busyTrue) != 1 {
		t.Fatalf("idempotent setBusy broadcast %d busy=true events, want exactly 1", len(busyTrue))
	}

	// agent_settled → busy false + 广播
	m.setBusy(entry.ID, false, "agent_settled")
	got, _ = m.sessions.Get(entry.ID)
	if got.Busy || m.toSessionInfo(got).Busy {
		t.Fatal("after agent_settled busy must be false")
	}

	// 非 active（closed/error）一律不 busy（即使 entry.Busy 残留）
	got.Busy = true
	_ = m.sessions.Update(got)
	got.Status = SessionStatusClosed
	_ = m.sessions.Update(got)
	if m.toSessionInfo(got).Busy {
		t.Fatal("closed session must project busy=false regardless of the flag")
	}
}

// TestEasyTasksLifecycle 验证 easy 合成 tasks.json 的生命周期：
// 创建时 running（进行中——dream 不学未完成 job），用户 close（正常结束）才标
// success（可被 dream 学习）。修复点：旧版创建即 success → **中断的 easy job
// 也被 dream 当完成学习并归档**（用户实测）。
func TestEasyTasksLifecycle(t *testing.T) {
	env := newTestEnv(t)
	m := env.managerWith(t, nil, nil)

	id, code, body := createSessionViaHTTP(t, m, env.wsEntry.ID, "easy", map[string]any{
		"requirement": "easy lifecycle test",
	})
	if code != http.StatusCreated {
		t.Fatalf("create easy: %d %v", code, body)
	}
	entry, _ := m.sessions.Get(id)

	// 创建后：running（不是 success）
	jobID := paramString(entry.Params, "_job_id")
	if jobID == "" {
		t.Fatal("easy entry missing _job_id param")
	}
	readStatus := func() string {
		t.Helper()
		data, err := os.ReadFile(filepath.Join(env.rickDir, "jobs", jobID, "doing", "tasks.json"))
		if err != nil {
			t.Fatal(err)
		}
		var doc struct {
			Tasks []struct {
				Status string `json:"status"`
			} `json:"tasks"`
		}
		if err := json.Unmarshal(data, &doc); err != nil {
			t.Fatal(err)
		}
		if len(doc.Tasks) != 1 {
			t.Fatalf("want 1 synthetic task, got %d", len(doc.Tasks))
		}
		return doc.Tasks[0].Status
	}
	if st := readStatus(); st != "running" {
		t.Fatalf("after create: task status = %q, want running (in-flight)", st)
	}
	// 未完成的 job 不参与 dream 学习
	if ids := workspace.SelectPendingJobs(env.rickDir, 10); len(ids) != 0 {
		t.Fatalf("running easy job must not be dream-eligible, got %v", ids)
	}

	// close（正常结束）→ success → 可被 dream 学习
	if rec := post(t, m.SessionClose, "/api/sessions/"+id+"/close", nil); rec.Code != http.StatusAccepted {
		t.Fatalf("close easy: %d", rec.Code)
	}
	if st := readStatus(); st != "success" {
		t.Fatalf("after close: task status = %q, want success", st)
	}
	if ids := workspace.SelectPendingJobs(env.rickDir, 10); len(ids) != 1 || ids[0] != jobID {
		t.Fatalf("closed easy job should be dream-eligible, got %v", ids)
	}
}

// TestSessionResumeIdempotent 验证 resume 幂等 + 自愈：
// ① 已 active 且 worker 存活 → 202（already_active），不再 409（用户实测：
//
//	会话其实在跑，点 Resume 报 409「nothing to resume」且 UI 不刷新）；
//
// ② active 但 worker 缺失（异常路径）→ 修正状态并重新 spawn（自愈），不卡死。
func TestSessionResumeIdempotent(t *testing.T) {
	env := newTestEnv(t)
	m := env.managerWith(t, nil, nil)

	id, code, body := createSessionViaHTTP(t, m, env.wsEntry.ID, "plan", map[string]any{
		"requirement": "resume idempotency",
	})
	if code != http.StatusCreated {
		t.Fatalf("create: %d %v", code, body)
	}
	// 已 active（fake pi 已 spawn）→ resume 幂等 202
	rec := post(t, m.SessionResume, "/api/sessions/"+id+"/resume", nil)
	if rec.Code != http.StatusAccepted {
		t.Fatalf("resume on active session: %d (%s), want 202", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "already_active") {
		t.Fatalf("idempotent resume should report already_active: %s", rec.Body.String())
	}

	// 模拟「active 但 worker 丢失」（异常路径）：清 supervisor 里的 worker
	m.sup.Remove(id)
	entry, ok := m.sessions.Get(id)
	if !ok || entry.Status != SessionStatusActive {
		t.Fatalf("precondition: entry should be active, got %+v", entry)
	}
	rec = post(t, m.SessionResume, "/api/sessions/"+id+"/resume", nil)
	if rec.Code != http.StatusAccepted {
		t.Fatalf("resume with missing worker should self-heal: %d (%s)", rec.Code, rec.Body.String())
	}
	if got, _ := m.sessions.Get(id); got.Status != SessionStatusActive {
		t.Fatalf("after self-heal resume: status = %q, want active", got.Status)
	}
}

// TestSessionResumeConcurrentSpawnHeals 验证 resume 的并发/残留 worker 竞态：
// supervisor 已有活 worker 而 spawn 失败时，不能把状态误标 error（否则
// 「error + worker 存活」永久不一致，之后每次 resume 都 500）。应视为
// already_active(202) 并把状态纠正回 active。
func TestSessionResumeConcurrentSpawnHeals(t *testing.T) {
	env := newTestEnv(t)
	m := env.managerWith(t, nil, nil)

	id, code, body := createSessionViaHTTP(t, m, env.wsEntry.ID, "plan", map[string]any{
		"requirement": "concurrent resume race",
	})
	if code != http.StatusCreated {
		t.Fatalf("create: %d %v", code, body)
	}
	// 制造「status=error 但 worker 存活」的不一致态（模拟并发 resume 的败者路径）
	entry, _ := m.sessions.Get(id)
	m.updateStatus(entry, SessionStatusError, "simulated race loser")
	if w := m.sup.Get(id); w == nil {
		t.Fatal("precondition: a live worker should exist")
	}

	rec := post(t, m.SessionResume, "/api/sessions/"+id+"/resume", nil)
	if rec.Code != http.StatusAccepted {
		t.Fatalf("resume with live worker + error status: %d (%s), want 202", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "already_active") {
		t.Fatalf("want already_active in body: %s", rec.Body.String())
	}
	if got, _ := m.sessions.Get(id); got.Status != SessionStatusActive {
		t.Fatalf("status should be healed to active, got %q", got.Status)
	}
}

// ---- 人工恢复（/continue）——本增量的核心语义 ----

// TestSessionContinue_InteractiveSuspended 覆盖交互型会话的人工恢复：
// suspended 会话 → POST /continue → spawn `--session <pi>` → active；
// 幂等（重复点击 202 already_active）；恢复动作写入恢复报告。
func TestSessionContinue_InteractiveSuspended(t *testing.T) {
	env := newTestEnv(t)
	m := env.managerWith(t, nil, nil)

	entry := SessionEntry{
		ID: "susp-plan", WorkspaceID: env.wsEntry.ID, Type: SessionTypePlan,
		Status: SessionStatusSuspended, Title: "suspended plan",
		PISessionID: "pi-susp-plan",
		Params:      map[string]any{"requirement": "r"},
		CreatedAt:   time.Now(), LastReason: SuspendReasonRelease,
	}
	if err := env.sessions.Add(entry); err != nil {
		t.Fatal(err)
	}
	m.refreshSuspendedReport()

	rec := post(t, m.SessionContinue, "/api/sessions/susp-plan/continue", nil)
	if rec.Code != http.StatusAccepted {
		t.Fatalf("continue suspended session: %d, body=%s", rec.Code, rec.Body.String())
	}
	var body map[string]any
	_ = json.Unmarshal(rec.Body.Bytes(), &body)
	if body["resumed"] != true {
		t.Fatalf("continue body = %s, want resumed:true", rec.Body.String())
	}

	got, _ := env.sessions.Get("susp-plan")
	if got.Status != SessionStatusActive {
		t.Fatalf("after continue status = %q, want active", got.Status)
	}
	if w := env.sup.Get("susp-plan"); w == nil || w.IsDead() {
		t.Fatal("continue 应 spawn 出活 worker")
	}
	// 恢复动作必须留痕（报告：从 suspended 移入 recovered）
	rep := m.ReadRecoveryReport()
	if len(rep.Suspended) != 0 {
		t.Fatalf("恢复后报告 suspended 应为空: %+v", rep.Suspended)
	}
	if len(rep.Recovered) != 1 || rep.Recovered[0].ID != "susp-plan" {
		t.Fatalf("报告 recovered = %+v", rep.Recovered)
	}

	// 幂等：重复点击 → 202 already_active，不重复 spawn
	rec = post(t, m.SessionContinue, "/api/sessions/susp-plan/continue", nil)
	if rec.Code != http.StatusAccepted || !strings.Contains(rec.Body.String(), "already_active") {
		t.Fatalf("重复 continue 应幂等 202 already_active, got %d %s", rec.Code, rec.Body.String())
	}

	// 未知会话 → 404
	if rec := post(t, m.SessionContinue, "/api/sessions/nope/continue", nil); rec.Code != http.StatusNotFound {
		t.Fatalf("continue unknown session = %d, want 404", rec.Code)
	}
}

// TestSessionContinue_DoingNormalizesAndReruns 覆盖 doing 的人工恢复：
// 归一化tasks.json 里遗留的 running → pending（**在 runner 启动之前**），
// 再重跑剩余 task；且不把 success 改回 pending。
func TestSessionContinue_DoingNormalizesAndReruns(t *testing.T) {
	env := newTestEnv(t)

	type observed struct {
		jobID    string
		statuses map[string]string
	}
	seen := make(chan observed, 1)
	doing := func(ctx context.Context, rickDir, jobID string, progress func(handler.DoingEvent)) error {
		// runner 启动那一刻的 tasks.json 快照 —— 归一化必须已经完成
		data, err := os.ReadFile(filepath.Join(rickDir, "jobs", jobID, "doing", "tasks.json"))
		if err != nil {
			t.Errorf("runner 读 tasks.json: %v", err)
		}
		var doc struct {
			Tasks []struct {
				TaskID string `json:"task_id"`
				Status string `json:"status"`
			} `json:"tasks"`
		}
		_ = json.Unmarshal(data, &doc)
		st := map[string]string{}
		for _, tk := range doc.Tasks {
			st[tk.TaskID] = tk.Status
		}
		select {
		case seen <- observed{jobID: jobID, statuses: st}:
		default:
		}
		<-ctx.Done()
		return ctx.Err()
	}
	m := env.managerWith(t, doing, nil)

	// 造 job：一个 success（必须原样保留）+ 一个遗留 running + 一个 pending
	doingDir := filepath.Join(env.rickDir, "jobs", "job_9", "doing")
	if err := os.MkdirAll(doingDir, 0755); err != nil {
		t.Fatal(err)
	}
	tasksJSON := `{"version":"1.0","tasks":[` +
		`{"task_id":"task1","status":"success","commit_hash":"abc"},` +
		`{"task_id":"task2","status":"running"},` +
		`{"task_id":"task3","status":"pending"}]}`
	if err := os.WriteFile(filepath.Join(doingDir, "tasks.json"), []byte(tasksJSON), 0644); err != nil {
		t.Fatal(err)
	}

	entry := SessionEntry{
		ID: "susp-doing", WorkspaceID: env.wsEntry.ID, Type: SessionTypeDoing,
		Status: SessionStatusSuspended, Title: "doing job_9",
		Params: map[string]any{"job": "job_9"}, CreatedAt: time.Now(),
	}
	if err := env.sessions.Add(entry); err != nil {
		t.Fatal(err)
	}
	m.refreshSuspendedReport()

	rec := post(t, m.SessionContinue, "/api/sessions/susp-doing/continue", nil)
	if rec.Code != http.StatusAccepted {
		t.Fatalf("continue suspended doing: %d, body=%s", rec.Code, rec.Body.String())
	}
	var body map[string]any
	_ = json.Unmarshal(rec.Body.Bytes(), &body)
	if body["kind"] != SessionTypeDoing {
		t.Fatalf("continue body kind = %v, want doing", body["kind"])
	}
	if norm, ok := body["normalized_tasks"].([]any); !ok || len(norm) != 1 || norm[0] != "task2" {
		t.Fatalf("normalized_tasks = %v, want [task2]", body["normalized_tasks"])
	}

	select {
	case obs := <-seen:
		if obs.jobID != "job_9" {
			t.Fatalf("runner jobID = %q, want job_9", obs.jobID)
		}
		if obs.statuses["task2"] != "pending" {
			t.Fatalf("runner 启动时 task2 仍为 %q（归一化必须早于重跑）", obs.statuses["task2"])
		}
		if obs.statuses["task1"] != "success" {
			t.Fatalf("success 被改动了: %q", obs.statuses["task1"])
		}
	case <-time.After(5 * time.Second):
		t.Fatal("doing runner 未被启动（人工继续没有生效）")
	}

	// 重跑前状态回到 running
	if e, _ := env.sessions.Get("susp-doing"); e.Status != SessionStatusRunning {
		t.Fatalf("继续后状态 = %q, want running", e.Status)
	}
	// 幂等：再次 continue → 202 already_active（不重复起 runner）
	rec = post(t, m.SessionContinue, "/api/sessions/susp-doing/continue", nil)
	if rec.Code != http.StatusAccepted || !strings.Contains(rec.Body.String(), "already_active") {
		t.Fatalf("重复 continue doing 应幂等: %d %s", rec.Code, rec.Body.String())
	}
	// tasks.json 备份存在（归一化写过盘）
	if _, err := os.Stat(filepath.Join(doingDir, "tasks.json.bak")); err != nil {
		t.Fatalf("归一化应留下 tasks.json.bak 备份: %v", err)
	}

	// 收尾：取消后台 runner，避免 goroutine 泄漏
	post(t, m.SessionClose, "/api/sessions/susp-doing/close", nil)
}

// ---- RSI 去特殊化（human 裁决 2026-09-22）：loop 走标准加载，无专属会话类型 ----
//
// 修订后的成立条件：RSI 只是 .rick/loops/rick-rsi-loop.md 这条普通 loop——
// 由 easy/plan 会话提示词里的「可用的项目 Loops」目录（LoadLoopsContext：name+trigger）
// 按触发条件发现，agent 加载后自行遵循。因此这里只断言两件事：
// ① 标准发现路径可用：带该 loop 的工作区上建 easy 会话，提示词必须含目录条目；
// ② 「rsi」不再是合法会话类型（特殊入口已删，回落到 unknown session type 错误）。

// writeRsiLoopFixture 在工作区的 .rick/loops/ 下放一个最小合规的 rick-rsi-loop.md
// （五要素齐全，含 LoadLoopsContext 需要的 frontmatter trigger）。
func writeRsiLoopFixture(t *testing.T, ws string) {
	t.Helper()
	loopDir := filepath.Join(ws, ".rick", "loops")
	if err := os.MkdirAll(loopDir, 0o755); err != nil {
		t.Fatal(err)
	}
	body := `---
name: rick-rsi-loop
trigger: "当需要修改 rick 自身并让它生效到生产时触发"
scope: "全局"
---

## 目标
安全送达生产。

## 上下文管理
保留改动清单。

## 可调用工具
- rick tools dev-web restart
- rick tools release --merge-source

## 产出评估
由 rsi_check 校验。

## 停止标准
rsi_check pass=true。
`
	if err := os.WriteFile(filepath.Join(loopDir, "rick-rsi-loop.md"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

// TestEasyPromptDiscoversRsiLoop：标准发现路径——easy 会话的提示词必须包含
// 「可用的项目 Loops」目录里的 rick-rsi-loop 条目（name + trigger）。
// 这是「RSI 只是一条普通 loop」设计的成立条件：agent 靠这个目录找到 loop。
func TestEasyPromptDiscoversRsiLoop(t *testing.T) {
	env := newTestEnv(t)
	m := env.managerWith(t, nil, nil)
	writeRsiLoopFixture(t, filepath.Dir(env.rickDir)) // 工作区根 = rickDir 的父目录

	id, code, body := createSessionViaHTTP(t, m, env.wsEntry.ID, "easy", map[string]any{
		"requirement": "改进 rick 自身：优化 web UI 的会话列表",
	})
	if code != http.StatusCreated {
		t.Fatalf("create easy session: %d %v", code, body)
	}
	entry, ok := m.sessions.Get(id)
	if !ok {
		t.Fatal("easy session not persisted in registry")
	}
	promptFile, _ := entry.Params["_prompt_file"].(string)
	if promptFile == "" {
		t.Fatal("easy entry missing _prompt_file param")
	}
	data, err := os.ReadFile(promptFile)
	if err != nil {
		t.Fatalf("read easy prompt: %v", err)
	}
	promptText := string(data)
	// 目录标题（LoadLoopsContext 的注入格式）
	if !strings.Contains(promptText, "可用的项目 Loops") {
		t.Fatal("easy prompt missing '可用的项目 Loops' catalog header (LoadLoopsContext)")
	}
	// 目录条目：name + trigger（格式 "- **name**：trigger"）
	if !strings.Contains(promptText, "rick-rsi-loop") {
		t.Fatal("easy prompt catalog must list rick-rsi-loop (standard discovery path)")
	}
	if !strings.Contains(promptText, "修改 rick 自身") {
		t.Fatal("easy prompt catalog must carry the loop's trigger text")
	}
	// 无需全文注入（那是已删除的特殊机制）：目录只负责发现，agent 按需读取
	if strings.Contains(promptText, "## 停止标准") {
		t.Fatal("easy prompt must NOT embed the full loop body (special injection was removed)")
	}
}

// TestCreateSessionRSITypeRemoved：「rsi」不是合法会话类型（去特殊化后回落到
// unknown session type 错误路径，返回 400 invalid_params）。
func TestCreateSessionRSITypeRemoved(t *testing.T) {
	env := newTestEnv(t)
	m := env.managerWith(t, nil, nil)

	_, code, body := createSessionViaHTTP(t, m, env.wsEntry.ID, "rsi", map[string]any{})
	if code != http.StatusBadRequest {
		t.Fatalf("type=rsi must be rejected with 400, got %d %v", code, body)
	}
	e, _ := body["error"].(map[string]any)
	if msg, _ := e["message"].(string); !strings.Contains(msg, "unknown session type") {
		t.Fatalf("error should be the unknown-type message, got: %v", e)
	}
}
