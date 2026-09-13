package web

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"sync"
	"time"
)

// SSE 事件类型常量（api-contract.md SSE 节 envelope.type）。
const (
	EventTypeServerInfo     = "server_info"     // 连接建立即发：{"version","rick_version","time"}
	EventTypeSessionEvent   = "session_event"   // pi rpc 事件透传：{"event":<pi 原始 rpc 事件对象>}
	EventTypeSessionState   = "session_state"   // 会话状态变更：{"status":"...","reason"?}
	EventTypeJobsUpdate     = "jobs_update"     // tasks.json 变更：{"job_id","diff":[...],"snapshot":[...]}
	EventTypeFrontendReload = "frontend_reload" // 覆盖层 dist 变更：{}
)

// Envelope is the SSE event envelope (api-contract.md SSE 节):
//
//	{"seq":N,"type":"...","session_id":"..."|null,"data":{...}}
//
// Seq is assigned by Hub.Publish (globally monotonic, resets on restart).
type Envelope struct {
	Seq       int64           `json:"seq"`
	Type      string          `json:"type"`
	SessionID string          `json:"session_id"`
	Data      json.RawMessage `json:"data"`
}

// Subscription is one SSE client's view of the Hub.
type Subscription struct {
	ch       chan Envelope
	lastSent int64 // highest seq delivered into ch (for diagnostics)
	dropped  int64 // events dropped for this slow consumer
}

// Hub is the single multiplexed SSE event bus: every session's events, jobs
// updates and frontend reloads are published here and broadcast to all
// subscribers. Each event gets a globally monotonic seq; a bounded ring
// buffer keeps the most recent events for Last-Event-ID replay on reconnect.
//
// Concurrency: Publish/Subscribe/Unsubscribe/Stats are safe for concurrent
// use. The buffer default (1000) matches api-contract.md; when a client's
// Last-Event-ID is older than the buffer's window, the replay sends a
// server_info envelope so the client rebuilds full state instead of silently
// missing events (contract: 超限则发 server_info 重建全量状态).
type Hub struct {
	mu   sync.Mutex
	seq  int64
	buf  []Envelope // ring replay buffer, newest last
	max  int        // buffer capacity
	subs map[*Subscription]struct{}

	// diagnostics
	droppedTotal int64 // events dropped across all slow consumers
}

// NewHub creates a Hub with a replay buffer of bufSize events (api-contract
// default 1000; non-positive falls back to 1000).
func NewHub(bufSize int) *Hub {
	if bufSize <= 0 {
		bufSize = 1000
	}
	return &Hub{
		max:  bufSize,
		subs: make(map[*Subscription]struct{}),
	}
}

// Publish assigns the next seq to evt's copy, records it in the replay
// buffer and broadcasts to all subscribers. Slow consumers (full channel)
// drop their oldest buffered event and keep the newest (contract: 丢旧保新).
func (h *Hub) Publish(evt Envelope) {
	h.mu.Lock()
	// Assign seq here (under lock) so seq order == publish order.
	h.seq++
	evt.Seq = h.seq

	// Ring buffer append with eviction of the oldest when over capacity.
	h.buf = append(h.buf, evt)
	if len(h.buf) > h.max {
		// Copy-on-evict to avoid unbounded slice growth (append reuse).
		h.buf = h.buf[len(h.buf)-h.max:]
	}

	for sub := range h.subs {
		sub.deliver(evt)
	}
	h.mu.Unlock()
}

// deliver pushes evt into the subscription channel. Must be called with the
// hub lock held (serializes across subscribers; channel ops are quick).
func (s *Subscription) deliver(evt Envelope) {
	for {
		select {
		case s.ch <- evt:
			s.lastSent = evt.Seq
			return
		default:
			// Slow consumer: drop the oldest queued event to make room,
			// then retry. Dropped count is per-subscription diagnostics.
			select {
			case <-s.ch:
				s.dropped++
			default:
				// Channel emptied between attempts (racing reader) — retry send.
			}
		}
	}
}

// Subscribe registers a new subscriber. lastEventID is the client's
// Last-Event-ID header (0 when absent): events with seq > lastEventID are
// replayed from the ring buffer first, so a reconnecting client catches up
// before live events arrive. If lastEventID is older than the buffer window
// (or from a previous server run), a server_info envelope is delivered in
// place of the gap so the client rebuilds full state.
//
// The returned subscription's channel has capacity 256 (slow consumers drop
// oldest rather than blocking Publish).
func (h *Hub) Subscribe(lastEventID int64) *Subscription {
	sub := &Subscription{ch: make(chan Envelope, 256)}

	h.mu.Lock()
	defer h.mu.Unlock()

	// Replay pass: buffer holds seq-ordered events.
	if lastEventID > 0 {
		if len(h.buf) > 0 && lastEventID >= h.buf[0].Seq-1 {
			// Client's cursor is within the buffer window: replay everything
			// strictly after it. (h.buf[0].Seq-1 is the newest seq NOT in the
			// buffer; a cursor >= that means no gap.)
			for _, evt := range h.buf {
				if evt.Seq > lastEventID {
					sub.deliver(evt)
				}
			}
		} else {
			// Cursor too old (or buffer empty after a restart): signal a full
			// state rebuild instead of silently skipping the gap.
			sub.deliver(Envelope{
				Seq:  h.seq, // keep monotonic; the real next event continues from here
				Type: EventTypeServerInfo,
				Data: json.RawMessage(`{"reason":"replay_overflow","reconnect":true}`),
			})
		}
	}

	h.subs[sub] = struct{}{}
	return sub
}

// Unsubscribe removes a subscriber and lets its channel be GC'd. Idempotent.
func (h *Hub) Unsubscribe(sub *Subscription) {
	if sub == nil {
		return
	}
	h.mu.Lock()
	delete(h.subs, sub)
	h.mu.Unlock()
}

// Events returns the subscription's live event channel.
func (s *Subscription) Events() <-chan Envelope { return s.ch }

// Dropped reports how many events were dropped for this slow consumer.
func (s *Subscription) Dropped() int64 {
	// No lock: diagnostics only; races yield slightly stale values.
	return s.dropped
}

// HubStats is a point-in-time snapshot of hub diagnostics (exposed for
// /api/config or logging per the task spec).
type HubStats struct {
	Seq       int64 // last assigned seq
	Buffered  int   // events currently in the replay buffer
	Clients   int   // active subscribers
	SubsDrops int64 // total events dropped to slow consumers
}

// Stats returns a diagnostics snapshot.
func (h *Hub) Stats() HubStats {
	h.mu.Lock()
	defer h.mu.Unlock()
	total := int64(0)
	for sub := range h.subs {
		total += sub.dropped
	}
	return HubStats{
		Seq:       h.seq,
		Buffered:  len(h.buf),
		Clients:   len(h.subs),
		SubsDrops: total,
	}
}

// ---- SSE HTTP handler ----

// sseHeartbeatInterval is the `: ping` comment cadence (api-contract: 15s).
// Overridable in tests via ServeSSEOptions.
const sseHeartbeatInterval = 15 * time.Second

// ServeSSEOptions tunes ServeSSE (heartbeat cadence + initial envelope).
// The zero value uses the contract defaults. Tests inject a short ticker here.
//
// InitialInfo, when non-nil, is written to the stream immediately after the
// response headers — the contract's「连接建立即发 server_info」. routes.go's
// handleEvents constructs it via ServerInfoEvent so every fresh connection
// (and reconnect) receives a state-rebuild signal without waiting for hub
// traffic or the heartbeat.
type ServeSSEOptions struct {
	Heartbeat  time.Duration
	InitialInfo *Envelope
}

// ServeSSE streams the hub to one HTTP client as text/event-stream:
//
//   - each event: `id: <seq>` + `data: <envelope JSON>` + blank line
//   - heartbeat comment `: ping` every 15s (default) keeps intermediaries
//     from reaping idle connections and detects dead peers
//   - Last-Event-ID header (or ?lastEventID= query param) seeds the replay
//     cursor for reconnecting clients
//   - X-Accel-Buffering: no disables nginx proxy buffering (known pitfall)
//   - disconnect (request context done) unsubscribes and returns
//
// The response starts (and stays) 200 once headers are written; errors after
// that simply end the stream — EventSource reconnects with Last-Event-ID.
func ServeSSE(w http.ResponseWriter, r *http.Request, hub *Hub, opts ServeSSEOptions) {
	if hub == nil {
		http.Error(w, `{"error":{"code":"internal","message":"nil hub"}}`, http.StatusInternalServerError)
		return
	}

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, `{"error":{"code":"internal","message":"streaming unsupported"}}`,
			http.StatusInternalServerError)
		return
	}

	// Reconnect cursor: Last-Event-ID header first, ?lastEventID= fallback.
	lastEventID := int64(0)
	if v := r.Header.Get("Last-Event-ID"); v != "" {
		if n, err := strconv.ParseInt(v, 10, 64); err == nil {
			lastEventID = n
		}
	} else if v := r.URL.Query().Get("lastEventID"); v != "" {
		if n, err := strconv.ParseInt(v, 10, 64); err == nil {
			lastEventID = n
		}
	}

	sub := hub.Subscribe(lastEventID)
	defer hub.Unsubscribe(sub)

	heartbeat := opts.Heartbeat
	if heartbeat <= 0 {
		heartbeat = sseHeartbeatInterval
	}
	ticker := time.NewTicker(heartbeat)
	defer ticker.Stop()

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	// Disable proxy buffering (nginx et al) so events flush immediately.
	w.Header().Set("X-Accel-Buffering", "no")
	w.WriteHeader(http.StatusOK)
	flusher.Flush()

	ctx := r.Context()
	writeEnvelope := func(evt Envelope) bool {
		payload, err := json.Marshal(evt)
		if err != nil {
			// Malformed envelope: log-and-skip rather than killing the stream.
			payload = []byte(fmt.Sprintf(`{"seq":%d,"type":"%s","session_id":%q,"data":{"marshal_error":%q}}`,
				evt.Seq, evt.Type, evt.SessionID, err.Error()))
		}
		if _, err := fmt.Fprintf(w, "id: %d\ndata: %s\n\n", evt.Seq, payload); err != nil {
			return false
		}
		flusher.Flush()
		return true
	}

	// 契约「连接建立即发 server_info」：初始 envelope 先于事件循环写出，
	// 客户端（gate6 的 SSE 探测/前端 sse.ts 的全量刷新信号）在静默期也能
	// 立即收到首个数据行。写失败（客户端已断开）则直接收尾。
	if opts.InitialInfo != nil {
		if !writeEnvelope(*opts.InitialInfo) {
			return
		}
	}

	for {
		select {
		case <-ctx.Done():
			return
		case evt := <-sub.ch:
			if !writeEnvelope(evt) {
				return
			}
		case <-ticker.C:
			// Heartbeat comment line (ignored by EventSource, keeps the
			// connection observable).
			if _, err := fmt.Fprint(w, ": ping\n\n"); err != nil {
				return
			}
			flusher.Flush()
		}
	}
}

// ---- 事件构造辅助（task11/13/14 消费）----

// mustMarshal marshals v for an envelope data payload; on impossible
// failures it degrades to an error marker object rather than panicking.
func mustMarshal(v any) json.RawMessage {
	b, err := json.Marshal(v)
	if err != nil {
		return json.RawMessage(fmt.Sprintf(`{"marshal_error":%q}`, err.Error()))
	}
	return b
}

// SessionEvent wraps a pi rpc event for transparent relay: data is the raw
// rpc event object (contract session_event: {"event":<pi 原始 rpc 事件>}).
// raw must be the original pi event JSON line (runtime.RpcEvent.Raw);
// an empty raw falls back to the typed struct v so callers can pass either.
func SessionEvent(sessionID string, raw json.RawMessage, v any) Envelope {
	if len(raw) > 0 {
		return Envelope{
			Type:      EventTypeSessionEvent,
			SessionID: sessionID,
			Data:      mustMarshal(map[string]json.RawMessage{"event": raw}),
		}
	}
	return Envelope{
		Type:      EventTypeSessionEvent,
		SessionID: sessionID,
		Data:      mustMarshal(map[string]any{"event": v}),
	}
}

// SessionStateEvent reports a session status change (contract session_state:
// {"status":"...","reason"?}).
// SessionBusyEvent broadcasts the server-authoritative streaming flag
// (session_state with "busy": true/false). The frontend renders the input
// area (send vs abort/steer) from this flag — independent of client-side
// event-replay inference (a refresh window can drop agent_start).
func SessionBusyEvent(sessionID, status string, busy bool, reason string) Envelope {
	payload := map[string]any{"status": status, "busy": busy}
	if reason != "" {
		payload["reason"] = reason
	}
	return Envelope{
		Type:      EventTypeSessionState,
		SessionID: sessionID,
		Data:      mustMarshal(payload),
	}
}

func SessionStateEvent(sessionID, status, reason string) Envelope {
	payload := map[string]any{"status": status}
	if reason != "" {
		payload["reason"] = reason
	}
	return Envelope{
		Type:      EventTypeSessionState,
		SessionID: sessionID,
		Data:      mustMarshal(payload),
	}
}

// JobsUpdateEvent reports a tasks.json change (contract jobs_update:
// {"job_id","diff":[{task_id,from,to}],"snapshot":[...]}).
type TaskDiff struct {
	TaskID string `json:"task_id"`
	From   string `json:"from"`
	To     string `json:"to"`
}

// TaskBrief is shared with jobs.go (the jobs listing projection of a task);
// the jobs_update snapshot reuses that shape.

// JobsUpdate builds a jobs_update envelope. snapshot may be nil (diff-only).
func JobsUpdate(workspaceID, jobID string, diff []TaskDiff, snapshot []TaskBrief) Envelope {
	if diff == nil {
		diff = []TaskDiff{}
	}
	if snapshot == nil {
		snapshot = []TaskBrief{}
	}
	return Envelope{
		Type:      EventTypeJobsUpdate,
		SessionID: "", // jobs are workspace-scoped, not session-scoped
		Data: mustMarshal(map[string]any{
			"workspace_id": workspaceID,
			"job_id":       jobID,
			"diff":         diff,
			"snapshot":     snapshot,
		}),
	}
}

// FrontendReloadEvent signals the browser to reload (overlay dist changed).
func FrontendReloadEvent() Envelope {
	return Envelope{
		Type: EventTypeFrontendReload,
		Data: json.RawMessage(`{}`),
	}
}

// ServerInfoEvent is sent when a client connects (and on replay overflow):
// {"version","rick_version","time"} per the contract.
func ServerInfoEvent(version int, rickVersion string) Envelope {
	return Envelope{
		Type: EventTypeServerInfo,
		Data: mustMarshal(map[string]any{
			"version":      version,
			"rick_version": rickVersion,
			"time":         time.Now().Format(time.RFC3339),
		}),
	}
}
