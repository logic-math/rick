package web

import (
	"encoding/json"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"
)

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
	want.Suspended = append(want.Suspended, RecoveryItem{ID: "s1", Type: SessionTypeEasy, JobID: "job_1", Reason: "x"})
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

// TestRecoveryReportRecordsHumanActions 覆盖报告的三栏流转：挂起 → 人工恢复成功
// 则从 suspended 移入 recovered；失败则进 failed 并保留在 suspended。
func TestRecoveryReportRecordsHumanActions(t *testing.T) {
	env := newTestEnv(t)
	m := env.managerWith(t, nil)

	suspendedEntry := SessionEntry{ID: "s-1", WorkspaceID: env.wsEntry.ID, Type: SessionTypeEasy,
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
	if len(rep.Recovered) != 1 || rep.Recovered[0].ID != "s-1" || rep.Recovered[0].Type != SessionTypeEasy {
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
	m := env.managerWith(t, nil)

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
