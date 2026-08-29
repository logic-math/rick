package web

import (
	"bufio"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"
)

// readSSELine reads one logical SSE record (up to a blank line) from the
// response stream. Returns the raw block; callers parse id:/data: lines.
func readSSELine(t *testing.T, br *bufio.Reader, deadline time.Duration) (string, bool) {
	t.Helper()
	type result struct {
		block string
		ok    bool
	}
	done := make(chan result, 1)
	go func() {
		var sb strings.Builder
		for {
			line, err := br.ReadString('\n')
			sb.WriteString(line)
			if err != nil {
				done <- result{sb.String(), false}
				return
			}
			if line == "\n" || line == "\r\n" {
				done <- result{sb.String(), true}
				return
			}
		}
	}()
	select {
	case r := <-done:
		return r.block, r.ok
	case <-time.After(deadline):
		return "", false
	}
}

// parseSSEBlock extracts the id and data payload from one SSE record.
func parseSSEBlock(t *testing.T, block string) (seq int64, data string) {
	t.Helper()
	for _, line := range strings.Split(block, "\n") {
		line = strings.TrimRight(line, "\r")
		switch {
		case strings.HasPrefix(line, "id:"):
			if n, err := parseSeq(line[3:]); err == nil {
				seq = n
			}
		case strings.HasPrefix(line, "data:"):
			data = strings.TrimSpace(line[5:])
		}
	}
	return seq, data
}

func parseSeq(s string) (int64, error) {
	return strconv.ParseInt(strings.TrimSpace(s), 10, 64)
}

// TestSSEPublishReachSubscriber：订阅后 Publish 的事件可达（httptest 全链路）。
func TestSSEPublishReachSubscriber(t *testing.T) {
	hub := NewHub(0) // default 1000
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ServeSSE(w, r, hub, ServeSSEOptions{Heartbeat: time.Hour})
	}))
	defer srv.Close()

	resp, err := srv.Client().Get(srv.URL)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	defer resp.Body.Close()
	if ct := resp.Header.Get("Content-Type"); !strings.HasPrefix(ct, "text/event-stream") {
		t.Fatalf("content-type: %q", ct)
	}
	if ab := resp.Header.Get("X-Accel-Buffering"); ab != "no" {
		t.Fatalf("X-Accel-Buffering: %q", ab)
	}

	br := bufio.NewReader(resp.Body)

	// Publish two events; both must arrive in order with seq 1,2.
	hub.Publish(FrontendReloadEvent())
	hub.Publish(SessionStateEvent("s1", "active", ""))

	for want := int64(1); want <= 2; want++ {
		block, ok := readSSELine(t, br, 3*time.Second)
		if !ok {
			t.Fatalf("event %d: stream ended/timeout", want)
		}
		seq, data := parseSSEBlock(t, block)
		if seq != want {
			t.Fatalf("event seq: want %d got %d (block=%q)", want, seq, block)
		}
		var env Envelope
		if err := json.Unmarshal([]byte(data), &env); err != nil {
			t.Fatalf("envelope unmarshal: %v (data=%q)", err, data)
		}
		if env.Seq != want {
			t.Fatalf("envelope.seq: want %d got %d", want, env.Seq)
		}
		if want == 2 && env.Type != EventTypeSessionState {
			t.Fatalf("event 2 type: got %q", env.Type)
		}
		if want == 2 && env.SessionID != "s1" {
			t.Fatalf("event 2 session_id: got %q", env.SessionID)
		}
	}
}

// TestSSEEnvelopeSerialization：envelope JSON 形状符合契约（snake_case 字段）。
func TestSSEEnvelopeSerialization(t *testing.T) {
	env := Envelope{
		Seq:       7,
		Type:      EventTypeSessionEvent,
		SessionID: "abc",
		Data:      json.RawMessage(`{"event":{"type":"message_end"}}`),
	}
	b, err := json.Marshal(env)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	s := string(b)
	for _, want := range []string{`"seq":7`, `"type":"session_event"`, `"session_id":"abc"`} {
		if !strings.Contains(s, want) {
			t.Errorf("envelope JSON missing %s: %s", want, s)
		}
	}
}

// TestSSELastEventIDReplay：重连带 Last-Event-ID 只补 seq 之后的。
func TestSSELastEventIDReplay(t *testing.T) {
	hub := NewHub(100)

	// Publish 5 events with no subscriber: seq 1..5 sit in the buffer.
	for i := 0; i < 5; i++ {
		hub.Publish(SessionStateEvent("s", "active", ""))
	}

	sub := hub.Subscribe(3) // client saw up to seq 3
	got := drain(t, sub.ch, 2, 2*time.Second)
	if len(got) != 2 {
		t.Fatalf("replay length: want 2 got %d (%v)", len(got), got)
	}
	if got[0].Seq != 4 || got[1].Seq != 5 {
		t.Fatalf("replay seqs: want [4 5] got %v", got)
	}

	// Fresh client (cursor 0 = first connection, no Last-Event-ID): NO replay
	// (state is rebuilt via REST, not via SSE replay — first-connection semantics).
	sub2 := hub.Subscribe(0)
	got2 := drain(t, sub2.ch, 5, 300*time.Millisecond)
	if len(got2) != 0 {
		t.Fatalf("fresh client should get no replay, got %d events", len(got2))
	}
	// ...but the buffer is intact for a reconnecting client.
	sub3 := hub.Subscribe(0)
	_ = sub3
	hub.Publish(SessionStateEvent("s", "active", "")) // seq 6
	got3 := drain(t, sub3.ch, 1, 2*time.Second)
	if len(got3) != 1 || got3[0].Seq != 6 {
		t.Fatalf("live event after fresh subscribe: got %v", got3)
	}
}

// TestSSEReplayOverflow：游标老于缓冲窗口 → 发 server_info 重建信号而非静默缺口。
func TestSSEReplayOverflow(t *testing.T) {
	hub := NewHub(3) // tiny buffer
	for i := 0; i < 5; i++ {
		hub.Publish(FrontendReloadEvent())
	} // buffer now holds seq 3,4,5

	sub := hub.Subscribe(1) // cursor 1 → older than window (buffer starts at 3)
	got := drain(t, sub.ch, 1, 2*time.Second)
	if len(got) != 1 {
		t.Fatalf("overflow signal length: got %d", len(got))
	}
	if got[0].Type != EventTypeServerInfo {
		t.Fatalf("overflow signal type: got %q", got[0].Type)
	}
	if !strings.Contains(string(got[0].Data), "replay_overflow") {
		t.Fatalf("overflow signal data: %s", got[0].Data)
	}

	// Cursor within window still replays normally.
	sub2 := hub.Subscribe(3)
	got2 := drain(t, sub2.ch, 2, 2*time.Second)
	if len(got2) != 2 || got2[0].Seq != 4 || got2[1].Seq != 5 {
		t.Fatalf("in-window replay after overflow: got %v", got2)
	}
}

// TestSSEMultiSubscriberBroadcast：多订阅者广播。
func TestSSEMultiSubscriberBroadcast(t *testing.T) {
	hub := NewHub(0)
	a := hub.Subscribe(0)
	b := hub.Subscribe(0)

	hub.Publish(FrontendReloadEvent())

	for name, sub := range map[string]*Subscription{"a": a, "b": b} {
		got := drain(t, sub.ch, 1, 2*time.Second)
		if len(got) != 1 || got[0].Seq != 1 {
			t.Fatalf("subscriber %s: got %v", name, got)
		}
	}

	// Unsubscribe a; further publishes only reach b.
	hub.Unsubscribe(a)
	hub.Publish(FrontendReloadEvent())
	if got := drain(t, a.ch, 1, 500*time.Millisecond); len(got) != 0 {
		t.Fatalf("unsubscribed a still got events: %v", got)
	}
	if got := drain(t, b.ch, 1, 2*time.Second); len(got) != 1 {
		t.Fatalf("b missing event after a unsubscribed: %v", got)
	}

	// Hub stats reflect one client left.
	if st := hub.Stats(); st.Clients != 1 {
		t.Fatalf("stats clients: want 1 got %d", st.Clients)
	}
}

// TestSSESlowConsumerDropsOldest：慢消费者 chan 满丢旧不阻塞 Publish。
func TestSSESlowConsumerDropsOldest(t *testing.T) {
	hub := NewHub(0)
	sub := hub.Subscribe(0) // never drains (ch cap 256)

	// Publish 300 events (beyond channel capacity): Publish must not block.
	done := make(chan struct{})
	go func() {
		defer close(done)
		for i := 0; i < 300; i++ {
			hub.Publish(FrontendReloadEvent())
		}
	}()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("Publish blocked on slow consumer")
	}

	// The slow consumer's channel holds its newest 256 events; the oldest 44
	// were dropped and counted.
	if n := len(sub.ch); n != 256 {
		t.Fatalf("channel fill: want 256 got %d", n)
	}
	if d := sub.Dropped(); d != 44 {
		t.Fatalf("dropped count: want 44 got %d", d)
	}
	if st := hub.Stats(); st.SubsDrops != 44 {
		t.Fatalf("stats drops: want 44 got %d", st.SubsDrops)
	}

	// Newest event must be the last one in the channel (drop-oldest order).
	last := drain(t, sub.ch, 256, 2*time.Second)
	if last[255].Seq != 300 {
		t.Fatalf("newest seq in channel: want 300 got %d", last[255].Seq)
	}
}

// TestSSEHeartbeat：短 ticker 注入下心跳注释行可见。
func TestSSEHeartbeat(t *testing.T) {
	hub := NewHub(0)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ServeSSE(w, r, hub, ServeSSEOptions{Heartbeat: 30 * time.Millisecond})
	}))
	defer srv.Close()

	resp, err := srv.Client().Get(srv.URL)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	defer resp.Body.Close()
	br := bufio.NewReader(resp.Body)

	block, ok := readSSELine(t, br, 3*time.Second)
	if !ok {
		t.Fatal("no heartbeat block received")
	}
	if !strings.HasPrefix(block, ": ping") {
		t.Fatalf("heartbeat block: %q", block)
	}
}

// TestSSELastEventIDHeader：HTTP 层 Last-Event-ID 头驱动重放（端到端）。
func TestSSELastEventIDHeader(t *testing.T) {
	hub := NewHub(0)

	// Pre-publish two events before any client connects.
	hub.Publish(FrontendReloadEvent())
	hub.Publish(FrontendReloadEvent())

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ServeSSE(w, r, hub, ServeSSEOptions{Heartbeat: time.Hour})
	}))
	defer srv.Close()

	req, _ := http.NewRequest("GET", srv.URL, nil)
	req.Header.Set("Last-Event-ID", "1")
	resp, err := srv.Client().Do(req)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	defer resp.Body.Close()
	br := bufio.NewReader(resp.Body)

	block, ok := readSSELine(t, br, 3*time.Second)
	if !ok {
		t.Fatal("no replay event received")
	}
	seq, _ := parseSSEBlock(t, block)
	if seq != 2 {
		t.Fatalf("replay via header: want seq 2 got %d (block=%q)", seq, block)
	}
}

// TestSSEDisconnectCleanup：客户端断开（请求 ctx done）→ Unsubscribe 生效（无 goroutine/订阅泄漏）。
func TestSSEDisconnectCleanup(t *testing.T) {
	hub := NewHub(0)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ServeSSE(w, r, hub, ServeSSEOptions{Heartbeat: 20 * time.Millisecond})
	}))
	defer srv.Close()

	resp, err := srv.Client().Get(srv.URL)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}

	// Consume the first heartbeat so the handler is fully in its loop.
	br := bufio.NewReader(resp.Body)
	if _, ok := readSSELine(t, br, 3*time.Second); !ok {
		t.Fatal("no initial block")
	}

	resp.Body.Close() // disconnect

	// Handler unregisters on context cancellation (poll stats).
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if hub.Stats().Clients == 0 {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	if n := hub.Stats().Clients; n != 0 {
		t.Fatalf("subscribers after disconnect: %d (leak)", n)
	}
}

// TestSSEEventConstructors：事件构造辅助的形状。
func TestSSEEventConstructors(t *testing.T) {
	// SessionEvent 透传 raw。
	env := SessionEvent("s1", json.RawMessage(`{"type":"agent_start"}`), nil)
	if env.Type != EventTypeSessionEvent || env.SessionID != "s1" {
		t.Fatalf("SessionEvent: %+v", env)
	}
	if !strings.Contains(string(env.Data), `"event":{"type":"agent_start"}`) {
		t.Fatalf("SessionEvent data: %s", env.Data)
	}

	// SessionEvent falls back to typed struct when raw empty.
	env = SessionEvent("s1", nil, map[string]string{"type": "agent_end"})
	if !strings.Contains(string(env.Data), `"event":{"type":"agent_end"}`) {
		t.Fatalf("SessionEvent typed fallback data: %s", env.Data)
	}

	// SessionStateEvent with reason.
	env = SessionStateEvent("s1", "closed", "user closed")
	if !strings.Contains(string(env.Data), `"status":"closed"`) ||
		!strings.Contains(string(env.Data), `"reason":"user closed"`) {
		t.Fatalf("SessionStateEvent data: %s", env.Data)
	}

	// JobsUpdate shape.
	env = JobsUpdate("ws1", "job_5", []TaskDiff{{TaskID: "task1", From: "pending", To: "running"}},
		[]TaskBrief{{TaskID: "task1", Name: "n", Status: "running"}})
	if !strings.Contains(string(env.Data), `"job_id":"job_5"`) ||
		!strings.Contains(string(env.Data), `"diff":[{"task_id":"task1","from":"pending","to":"running"}]`) {
		t.Fatalf("JobsUpdate data: %s", env.Data)
	}
	if env.SessionID != "" {
		t.Fatalf("JobsUpdate session_id should be empty: %q", env.SessionID)
	}

	// FrontendReloadEvent empty object.
	env = FrontendReloadEvent()
	if string(env.Data) != "{}" {
		t.Fatalf("FrontendReloadEvent data: %s", env.Data)
	}

	// ServerInfoEvent fields.
	env = ServerInfoEvent(1, "3.1.5")
	if !strings.Contains(string(env.Data), `"rick_version":"3.1.5"`) {
		t.Fatalf("ServerInfoEvent data: %s", env.Data)
	}
}

// TestSSESeqMonotonicConcurrent：并发 Publish 下 seq 全局单调无重复。
func TestSSESeqMonotonicConcurrent(t *testing.T) {
	hub := NewHub(0)
	sub := hub.Subscribe(0)

	const writers, each = 8, 50
	var wg sync.WaitGroup
	for i := 0; i < writers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < each; j++ {
				hub.Publish(FrontendReloadEvent())
				time.Sleep(time.Millisecond) // pace: keep the concurrent drainer fed (cap 256 < 400)
			}
		}()
	}

	// Concurrent drain so the subscriber never hits the 256 cap (slow-consumer
	// drops are covered separately in TestSSESlowConsumerDropsOldest).
	got := make([]Envelope, 0, writers*each)
	var mu sync.Mutex
	go func() {
		for evt := range sub.ch {
			mu.Lock()
			got = append(got, evt)
			mu.Unlock()
		}
	}()
	wg.Wait()
	// Give the drainer a moment to tail the channel, then detach. Note: the
	// drainer goroutine exits with the test (Unsubscribe does not close the
	// channel — closing would panic Publish racing a send).
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		mu.Lock()
		n := len(got)
		mu.Unlock()
		if n == writers*each {
			break
		}
		time.Sleep(5 * time.Millisecond)
	}
	hub.Unsubscribe(sub)
	mu.Lock()
	defer mu.Unlock()
	if len(got) != writers*each {
		t.Fatalf("events: want %d got %d", writers*each, len(got))
	}
	seen := make(map[int64]bool)
	last := int64(0)
	for _, evt := range got {
		if evt.Seq <= last {
			t.Fatalf("seq not monotonic: %d after %d", evt.Seq, last)
		}
		if seen[evt.Seq] {
			t.Fatalf("duplicate seq %d", evt.Seq)
		}
		seen[evt.Seq] = true
		last = evt.Seq
	}
	if st := hub.Stats(); st.Seq != int64(writers*each) || st.Buffered != writers*each {
		t.Fatalf("stats: %+v", st)
	}
}

// drain reads up to n envelopes from ch (waiting up to deadline overall).
func drain(t *testing.T, ch <-chan Envelope, n int, deadline time.Duration) []Envelope {
	t.Helper()
	var out []Envelope
	timer := time.NewTimer(deadline)
	defer timer.Stop()
	for len(out) < n {
		select {
		case evt := <-ch:
			out = append(out, evt)
		case <-timer.C:
			return out
		}
	}
	return out
}
