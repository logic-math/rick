package web

import (
	"encoding/json"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// TestShutdownSnapshotRoundTrip 覆盖关停快照的磁盘契约（原子写 + 读回 + 空快照
// 也有意义：显式空快照表示「本次关停没有被打断的会话」）。
func TestShutdownSnapshotRoundTrip(t *testing.T) {
	dir := t.TempDir()

	// 缺文件不是错误（首次启动）
	snap, err := ReadShutdownSnapshot(dir)
	if err != nil {
		t.Fatalf("read missing snapshot: %v", err)
	}
	if len(snap.Sessions) != 0 {
		t.Fatalf("missing snapshot should be empty, got %+v", snap.Sessions)
	}

	want := ShutdownSnapshot{
		StoppedAt: time.Now().UTC().Truncate(time.Second),
		PID:       1234,
		Reason:    SuspendReasonRelease,
		Sessions: []SuspendRecord{{
			SessionID:   "s1",
			Type:        SessionTypeEasy,
			WorkspaceID: "ws1",
			JobID:       "job_9",
			PISessionID: "pi-1",
			LastEntryID: "e42",
			Busy:        true,
			At:          time.Now().UTC().Truncate(time.Second),
		}},
	}
	if err := WriteShutdownSnapshot(dir, want); err != nil {
		t.Fatalf("write snapshot: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, ShutdownSnapshotFile)); err != nil {
		t.Fatalf("snapshot file missing: %v", err)
	}
	got, err := ReadShutdownSnapshot(dir)
	if err != nil {
		t.Fatalf("read snapshot: %v", err)
	}
	if got.Version != recoveryReportVersion || got.Reason != SuspendReasonRelease || got.PID != 1234 {
		t.Fatalf("snapshot header = %+v", got)
	}
	if len(got.Sessions) != 1 || got.Sessions[0].LastEntryID != "e42" || !got.Sessions[0].Busy {
		t.Fatalf("snapshot records = %+v", got.Sessions)
	}

	// 空快照也必须落盘（可审计事实）
	if err := WriteShutdownSnapshot(dir, ShutdownSnapshot{Reason: SuspendReasonRelease}); err != nil {
		t.Fatalf("write empty snapshot: %v", err)
	}
	got, err = ReadShutdownSnapshot(dir)
	if err != nil {
		t.Fatalf("read empty snapshot: %v", err)
	}
	if got.Sessions == nil || len(got.Sessions) != 0 {
		t.Fatalf("empty snapshot sessions = %+v, want empty non-nil", got.Sessions)
	}
	// 原子性副作用检查：不留 .tmp
	if _, err := os.Stat(filepath.Join(dir, ShutdownSnapshotFile+".tmp")); !os.IsNotExist(err) {
		t.Fatal("snapshot 落盘留下了 .tmp 残留")
	}
}

// TestRecoveryReportRoundTrip 覆盖恢复报告：读不到/损坏 → 空报告（绝不 500）、
// 三栏永远是数组（JSON 契约）、写入原子。
func TestRecoveryReportRoundTrip(t *testing.T) {
	dir := t.TempDir()

	// 报告是展示层数据：缺失/损坏都必须降级为空报告，而不是报错
	rep := ReadRecoveryReport(dir)
	if len(rep.Suspended) != 0 || len(rep.Recovered) != 0 || len(rep.Failed) != 0 {
		t.Fatalf("missing report should be empty: %+v", rep)
	}
	if err := os.WriteFile(filepath.Join(dir, RecoveryReportFile), []byte("{not json"), 0644); err != nil {
		t.Fatal(err)
	}
	if rep = ReadRecoveryReport(dir); len(rep.Suspended) != 0 {
		t.Fatalf("corrupt report should degrade to empty, got %+v", rep)
	}

	want := newRecoveryReport()
	want.Suspended = append(want.Suspended, RecoveryItem{ID: "s1", Type: SessionTypeDoing, JobID: "job_1", Reason: "x"})
	want.Recovered = append(want.Recovered, RecoveryItem{ID: "s2"})
	want.Failed = append(want.Failed, RecoveryItem{ID: "s3", Reason: "boom"})
	if err := WriteRecoveryReport(dir, want); err != nil {
		t.Fatalf("write report: %v", err)
	}
	got := ReadRecoveryReport(dir)
	if len(got.Suspended) != 1 || got.Suspended[0].ID != "s1" {
		t.Fatalf("suspended = %+v", got.Suspended)
	}
	if len(got.Recovered) != 1 || len(got.Failed) != 1 || got.Failed[0].Reason != "boom" {
		t.Fatalf("recovered/failed = %+v %+v", got.Recovered, got.Failed)
	}
}

// TestSuspendRunningMarksAndSnapshots 覆盖关停路径核心：只把 active/running 标成
// suspended（closed/error 不动）、写关停快照（带 worker 的 last entry id）、
// 且**不碰** worker（收尸由 CloseAll 负责，顺序在 ShutdownWorkers 里固定）。
func TestSuspendRunningMarksAndSnapshots(t *testing.T) {
	env := newTestEnv(t)
	m := env.managerWith(t, nil, nil)

	// 一个真活的 worker（fake pi），用于验证 LastEntryID 进快照
	activeID, _, _ := createSessionViaHTTP(t, m, env.wsEntry.ID, "plan", map[string]any{"requirement": "r"})
	waitLog(t, env.piLog, "开始")

	// 一个后台型 running 会话（fake runner 停住）
	runningID, code, _ := createSessionViaHTTP(t, m, env.wsEntry.ID, "doing", map[string]any{"job": "job_1"})
	if code != 201 {
		t.Fatalf("create doing: %d", code)
	}
	// 一个已结束的会话（不得被改成 suspended）
	closed := SessionEntry{ID: "closed-1", WorkspaceID: env.wsEntry.ID, Type: SessionTypePlan,
		Status: SessionStatusClosed, Params: map[string]any{}, CreatedAt: time.Now()}
	if err := env.sessions.Add(closed); err != nil {
		t.Fatal(err)
	}
	failed := SessionEntry{ID: "err-1", WorkspaceID: env.wsEntry.ID, Type: SessionTypeEasy,
		Status: SessionStatusError, Params: map[string]any{}, CreatedAt: time.Now()}
	if err := env.sessions.Add(failed); err != nil {
		t.Fatal(err)
	}

	recs := m.suspendRunning(SuspendReasonRelease)
	if len(recs) != 2 {
		t.Fatalf("suspendRunning records = %d, want 2 (%+v)", len(recs), recs)
	}
	byID := map[string]SuspendRecord{}
	for _, r := range recs {
		byID[r.SessionID] = r
	}
	if r, ok := byID[activeID]; !ok || r.Type != SessionTypePlan || r.PISessionID == "" || r.WorkspaceID != env.wsEntry.ID {
		t.Fatalf("active 记录不完整: %+v", r)
	}
	if r, ok := byID[runningID]; !ok || r.JobID != "job_1" {
		t.Fatalf("doing 记录应带 job id: %+v", r)
	}

	// 状态落盘
	for id, want := range map[string]string{
		activeID:   SessionStatusSuspended,
		runningID:  SessionStatusSuspended,
		"closed-1": SessionStatusClosed,
		"err-1":    SessionStatusError,
	} {
		e, _ := env.sessions.Get(id)
		if e.Status != want {
			t.Fatalf("%s status = %q, want %q", id, e.Status, want)
		}
		if want == SessionStatusSuspended && e.LastReason != SuspendReasonRelease {
			t.Fatalf("%s LastReason = %q, want %q", id, e.LastReason, SuspendReasonRelease)
		}
	}

	// 快照落盘且能读回
	snap, err := ReadShutdownSnapshot(m.stateDir())
	if err != nil {
		t.Fatalf("read snapshot: %v", err)
	}
	if snap.Reason != SuspendReasonRelease || len(snap.Sessions) != 2 {
		t.Fatalf("snapshot = %+v", snap)
	}
	if snap.PID != os.Getpid() {
		t.Fatalf("snapshot pid = %d, want %d", snap.PID, os.Getpid())
	}

	// 关停路径不得顺手收 worker（顺序契约：ShutdownWorkers 先 suspend 再 CloseAll）
	if w := env.sup.Get(activeID); w == nil || w.IsDead() {
		t.Fatal("suspendRunning 不应杀掉 worker（收尸是 CloseAll 的职责）")
	}
}

// TestShutdownWorkersSuspendsThenCloses 覆盖 ShutdownWorkers 的顺序契约：
// 先挂起（状态落盘 + 快照）再收 worker，且收尸之后状态仍是 suspended
// （worker 死亡回调会把 active/running 刷成 error —— 必须不发生）。
func TestShutdownWorkersSuspendsThenCloses(t *testing.T) {
	env := newTestEnv(t)
	m := env.managerWith(t, nil, nil)

	id, _, _ := createSessionViaHTTP(t, m, env.wsEntry.ID, "plan", map[string]any{"requirement": "r"})
	waitLog(t, env.piLog, "开始")

	m.ShutdownWorkers()

	e, ok := env.sessions.Get(id)
	if !ok {
		t.Fatal("session vanished")
	}
	if e.Status != SessionStatusSuspended {
		t.Fatalf("after ShutdownWorkers status = %q, want suspended（不能被死亡回调刷成 error）", e.Status)
	}
	if e.LastReason != SuspendReasonRelease {
		t.Fatalf("LastReason = %q, want %q", e.LastReason, SuspendReasonRelease)
	}
	if w := env.sup.Get(id); w != nil && !w.IsDead() {
		t.Fatal("ShutdownWorkers 应已收掉 worker")
	}
	if _, err := os.Stat(filepath.Join(m.stateDir(), ShutdownSnapshotFile)); err != nil {
		t.Fatalf("关停快照缺失: %v", err)
	}
}

// TestRecoveryReportRecordsHumanActions 覆盖报告的三栏流转：挂起 → 人工恢复成功
// 则从 suspended 移入 recovered；失败则进 failed 并保留在 suspended。
func TestRecoveryReportRecordsHumanActions(t *testing.T) {
	env := newTestEnv(t)
	m := env.managerWith(t, nil, nil)

	suspendedEntry := SessionEntry{ID: "s-1", WorkspaceID: env.wsEntry.ID, Type: SessionTypePlan,
		Status: SessionStatusSuspended, Title: "t", Params: map[string]any{"requirement": "x"}, CreatedAt: time.Now()}
	if err := env.sessions.Add(suspendedEntry); err != nil {
		t.Fatal(err)
	}
	m.refreshSuspendedReport()
	if rep := m.ReadRecoveryReport(); len(rep.Suspended) != 1 || rep.Suspended[0].ID != "s-1" {
		t.Fatalf("refreshSuspendedReport = %+v", rep)
	}

	m.recordRecovered(recoveryItemOf(suspendedEntry, "resumed by human"))
	rep := m.ReadRecoveryReport()
	if len(rep.Suspended) != 0 {
		t.Fatalf("恢复后不应仍在 suspended: %+v", rep.Suspended)
	}
	if len(rep.Recovered) != 1 || rep.Recovered[0].ID != "s-1" || rep.Recovered[0].Type != SessionTypePlan {
		t.Fatalf("recovered = %+v", rep.Recovered)
	}
	// 幂等：同一会话重复记录不重复入列
	m.recordRecovered(recoveryItemOf(suspendedEntry, "resumed by human"))
	if rep = m.ReadRecoveryReport(); len(rep.Recovered) != 1 {
		t.Fatalf("recovered 应去重: %+v", rep.Recovered)
	}

	m.recordFailed(recoveryItemOf(suspendedEntry, "spawn failed"))
	if rep = m.ReadRecoveryReport(); len(rep.Failed) != 1 || rep.Failed[0].Reason != "spawn failed" {
		t.Fatalf("failed = %+v", rep.Failed)
	}
}

// TestHandleRecoveryReport 覆盖 GET /api/recovery 的 HTTP 契约：永远 200 + 四个
// 必备字段（前端横幅据此渲染）。
func TestHandleRecoveryReport(t *testing.T) {
	env := newTestEnv(t)
	m := env.managerWith(t, nil, nil)

	rec := get(t, m.HandleRecoveryReport, "/api/recovery")
	if rec.Code != 200 {
		t.Fatalf("GET /api/recovery = %d, want 200", rec.Code)
	}
	body := decodeJSON(t, rec)
	for _, k := range []string{"at", "suspended", "recovered", "failed"} {
		if _, ok := body[k]; !ok {
			t.Fatalf("恢复报告缺字段 %q: %v", k, body)
		}
	}
	for _, k := range []string{"suspended", "recovered", "failed"} {
		if _, ok := body[k].([]any); !ok {
			t.Fatalf("%s 必须是数组（JSON 契约），got %T", k, body[k])
		}
	}
}

// decodeJSON 解一个 JSON 响应体为 map（断言字段存在性用）。
func decodeJSON(t *testing.T, rec *httptest.ResponseRecorder) map[string]any {
	t.Helper()
	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode body %q: %v", rec.Body.String(), err)
	}
	return body
}
