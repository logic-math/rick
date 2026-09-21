package web

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// 挂起/恢复的磁盘契约（设计依据 research-L6 §3.2 / §3.6）：
//
//	<stateDir>/suspend.json          关停快照（旧进程的「遗嘱」：关停那一刻谁在跑）
//	<stateDir>/recovery-report.json  恢复报告（挂起清单 + 已恢复 + 失败）
//
// ⚠️ 本增量**不做自动恢复**（human 裁决 J-L6-6/J-L6-7：实测 pi 不修复悬挂
// toolCall → 自动续跑会重复副作用；配额耗尽不报错 → 自动恢复会静默空转）。
// 因此这两个文件的职责被刻意收窄为：
//
//	① 关停快照 = 「平台升级」这一动作的可审计证据 + 供人/脚本查看谁被打断；
//	② 恢复报告 = 让 UI 明确「谁因平台升级挂起」、并记录人工恢复的结果。
//
// 真正的恢复动作只有一个人工入口：POST /api/sessions/{id}/continue。
const (
	// ShutdownSnapshotFile 是关停快照文件名（位于 state dir 根）。
	ShutdownSnapshotFile = "suspend.json"
	// RecoveryReportFile 是恢复报告文件名（位于 state dir 根）。
	RecoveryReportFile = "recovery-report.json"
	// recoveryReportVersion 是恢复报告的 schema 版本。
	recoveryReportVersion = 1
)

// SuspendReasonRelease 是「人类主动提升/重启平台」导致的挂起原因。
const SuspendReasonRelease = "platform upgrade (release)"

// SuspendReasonRestart 是「服务重启（非 release 路径）」导致的挂起原因。
const SuspendReasonRestart = "server restart"

// SuspendRecord is one session that was running when the server shut down.
// Mirrors the resume-relevant fields of SessionEntry plus the worker's durable
// cursor (research-L6 §3.2：快照比「重启后反推」可靠得多).
type SuspendRecord struct {
	SessionID   string    `json:"session_id"`
	Type        string    `json:"type"`
	WorkspaceID string    `json:"workspace_id,omitempty"`
	JobID       string    `json:"job_id,omitempty"`
	PISessionID string    `json:"pi_session_id,omitempty"`
	LastEntryID string    `json:"last_entry_id,omitempty"`
	Busy        bool      `json:"busy,omitempty"`
	At          time.Time `json:"at"`
}

// ShutdownSnapshot is the atomic "last will" written before workers are killed.
// It is written even when nothing is running (an explicit empty snapshot is
// meaningful: 「本次关停没有任何在跑的会话」).
type ShutdownSnapshot struct {
	Version   int             `json:"version"`
	StoppedAt time.Time       `json:"stopped_at"`
	PID       int             `json:"pid,omitempty"`
	Reason    string          `json:"reason,omitempty"`
	Sessions  []SuspendRecord `json:"sessions"`
}

// RecoveryItem is one row of the recovery report (UI 可读).
type RecoveryItem struct {
	ID     string    `json:"id"`
	Type   string    `json:"type,omitempty"`
	Title  string    `json:"title,omitempty"`
	JobID  string    `json:"job_id,omitempty"`
	Reason string    `json:"reason,omitempty"`
	At     time.Time `json:"at"`
}

// RecoveryReport answers「谁被挂起了 / 谁已被人工恢复 / 哪些恢复失败」。
type RecoveryReport struct {
	Version   int            `json:"version"`
	At        time.Time      `json:"at"`
	Suspended []RecoveryItem `json:"suspended"`
	Recovered []RecoveryItem `json:"recovered"`
	Failed    []RecoveryItem `json:"failed"`
}

// newRecoveryReport returns a report with non-nil (JSON `[]`) slices.
func newRecoveryReport() RecoveryReport {
	return RecoveryReport{
		Version:   recoveryReportVersion,
		At:        time.Now(),
		Suspended: []RecoveryItem{},
		Recovered: []RecoveryItem{},
		Failed:    []RecoveryItem{},
	}
}

func shutdownSnapshotPath(stateDir string) string {
	return filepath.Join(stateDir, ShutdownSnapshotFile)
}

func recoveryReportPath(stateDir string) string {
	return filepath.Join(stateDir, RecoveryReportFile)
}

// WriteShutdownSnapshot persists the snapshot atomically (tmp+rename).
func WriteShutdownSnapshot(stateDir string, snap ShutdownSnapshot) error {
	if stateDir == "" {
		return fmt.Errorf("write shutdown snapshot: empty state dir")
	}
	if snap.Version == 0 {
		snap.Version = recoveryReportVersion
	}
	if snap.StoppedAt.IsZero() {
		snap.StoppedAt = time.Now()
	}
	if snap.Sessions == nil {
		snap.Sessions = []SuspendRecord{}
	}
	data, err := json.MarshalIndent(snap, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal shutdown snapshot: %w", err)
	}
	return writeFileAtomic(shutdownSnapshotPath(stateDir), append(data, '\n'), 0644)
}

// ReadShutdownSnapshot loads the last snapshot; a missing file is not an error.
func ReadShutdownSnapshot(stateDir string) (ShutdownSnapshot, error) {
	var snap ShutdownSnapshot
	if stateDir == "" {
		return snap, nil
	}
	data, err := os.ReadFile(shutdownSnapshotPath(stateDir))
	if err != nil {
		if os.IsNotExist(err) {
			return snap, nil
		}
		return snap, fmt.Errorf("read shutdown snapshot: %w", err)
	}
	if len(strings.TrimSpace(string(data))) == 0 {
		return snap, nil
	}
	if err := json.Unmarshal(data, &snap); err != nil {
		return snap, fmt.Errorf("parse shutdown snapshot: %w", err)
	}
	return snap, nil
}

// WriteRecoveryReport persists the report atomically.
func WriteRecoveryReport(stateDir string, rep RecoveryReport) error {
	if stateDir == "" {
		return fmt.Errorf("write recovery report: empty state dir")
	}
	if rep.Version == 0 {
		rep.Version = recoveryReportVersion
	}
	if rep.At.IsZero() {
		rep.At = time.Now()
	}
	if rep.Suspended == nil {
		rep.Suspended = []RecoveryItem{}
	}
	if rep.Recovered == nil {
		rep.Recovered = []RecoveryItem{}
	}
	if rep.Failed == nil {
		rep.Failed = []RecoveryItem{}
	}
	data, err := json.MarshalIndent(rep, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal recovery report: %w", err)
	}
	return writeFileAtomic(recoveryReportPath(stateDir), append(data, '\n'), 0644)
}

// ReadRecoveryReport loads the report; a missing/corrupt file yields an empty
// report (报告是展示层的，读不到绝不能让接口 500）。
func ReadRecoveryReport(stateDir string) RecoveryReport {
	rep := newRecoveryReport()
	if stateDir == "" {
		return rep
	}
	data, err := os.ReadFile(recoveryReportPath(stateDir))
	if err != nil {
		return rep
	}
	var loaded RecoveryReport
	if err := json.Unmarshal(data, &loaded); err != nil {
		return rep
	}
	if loaded.Suspended == nil {
		loaded.Suspended = []RecoveryItem{}
	}
	if loaded.Recovered == nil {
		loaded.Recovered = []RecoveryItem{}
	}
	if loaded.Failed == nil {
		loaded.Failed = []RecoveryItem{}
	}
	if loaded.Version == 0 {
		loaded.Version = recoveryReportVersion
	}
	return loaded
}

// ---- SessionManager 侧 ----

// stateDir 是当前实例的状态根（由会话注册表位置反推；见 SessionRegistry.StateDir）。
func (m *SessionManager) stateDir() string {
	if m == nil || m.sessions == nil {
		return ""
	}
	return m.sessions.StateDir()
}

// ReadRecoveryReport exposes the persisted report for GET /api/recovery.
func (m *SessionManager) ReadRecoveryReport() RecoveryReport {
	return ReadRecoveryReport(m.stateDir())
}

// suspendRunning 是关停路径的核心：把「当前在跑」的会话写进关停快照并标为
// suspended（**写盘**，不是只发事件），返回快照里的记录。必须在
// Supervisor.CloseAll 之前调用——否则 worker 死亡回调会先把状态刷成 error。
//
// 语义细节：即使没有任何会话在跑也会落一份空快照（显式空快照 = 本次关停没有
// 被打断的会话，是可审计的事实）。
func (m *SessionManager) suspendRunning(reason string) []SuspendRecord {
	if m == nil {
		return nil
	}
	records := make([]SuspendRecord, 0, 4)
	now := time.Now()
	for _, e := range m.sessions.List() {
		if e.Status != SessionStatusActive && e.Status != SessionStatusRunning {
			continue
		}
		rec := SuspendRecord{
			SessionID:   e.ID,
			Type:        e.Type,
			WorkspaceID: e.WorkspaceID,
			JobID:       jobParamOf(e),
			PISessionID: e.PISessionID,
			Busy:        e.Busy,
			At:          now,
		}
		if w := m.sup.Get(e.ID); w != nil {
			rec.LastEntryID = w.LastEntryID()
		}
		records = append(records, rec)
		// 先落状态再收 worker：CloseAll 之后 pump 的 markWorkerLost 只会作用于
		// active/running（suspended 不在其列），因此这里的状态不会被覆盖。
		m.updateStatus(e, SessionStatusSuspended, reason)
	}
	if err := WriteShutdownSnapshot(m.stateDir(), ShutdownSnapshot{
		Version:   recoveryReportVersion,
		StoppedAt: now,
		PID:       os.Getpid(),
		Reason:    reason,
		Sessions:  records,
	}); err != nil {
		fmt.Fprintf(os.Stderr, "[rick-web] write shutdown snapshot failed: %v\n", err)
	}
	return records
}

// refreshSuspendedReport 重建恢复报告的 suspended 清单（启动对账后调用）：
// suspended 的权威来源是**注册表**（重启后谁是挂起的一目了然），已恢复/失败两栏
// 是「本次进程生命周期内的人工动作」记录，启动即清零。
func (m *SessionManager) refreshSuspendedReport() {
	if m == nil || m.sessions == nil {
		return
	}
	rep := newRecoveryReport()
	for _, e := range m.sessions.List() {
		if e.Status != SessionStatusSuspended {
			continue
		}
		rep.Suspended = append(rep.Suspended, RecoveryItem{
			ID:     e.ID,
			Type:   e.Type,
			Title:  e.Title,
			JobID:  jobParamOf(e),
			Reason: e.LastReason,
			At:     e.CreatedAt,
		})
	}
	if err := WriteRecoveryReport(m.stateDir(), rep); err != nil {
		fmt.Fprintf(os.Stderr, "[rick-web] write recovery report failed: %v\n", err)
	}
}

// recordRecovered 把一次成功的人工恢复写进报告（从 suspended 移入 recovered）。
func (m *SessionManager) recordRecovered(item RecoveryItem) {
	if m == nil {
		return
	}
	if item.At.IsZero() {
		item.At = time.Now()
	}
	rep := m.ReadRecoveryReport()
	rep.Suspended = removeRecoveryItem(rep.Suspended, item.ID)
	rep.Recovered = upsertRecoveryItem(rep.Recovered, item)
	rep.At = time.Now()
	if err := WriteRecoveryReport(m.stateDir(), rep); err != nil {
		fmt.Fprintf(os.Stderr, "[rick-web] update recovery report failed: %v\n", err)
	}
}

// recordFailed 把一次失败的人工恢复写进报告（保留在 suspended 里，另记 failed）。
func (m *SessionManager) recordFailed(item RecoveryItem) {
	if m == nil {
		return
	}
	if item.At.IsZero() {
		item.At = time.Now()
	}
	rep := m.ReadRecoveryReport()
	rep.Failed = upsertRecoveryItem(rep.Failed, item)
	rep.At = time.Now()
	if err := WriteRecoveryReport(m.stateDir(), rep); err != nil {
		fmt.Fprintf(os.Stderr, "[rick-web] update recovery report failed: %v\n", err)
	}
}

func removeRecoveryItem(items []RecoveryItem, id string) []RecoveryItem {
	out := make([]RecoveryItem, 0, len(items))
	for _, it := range items {
		if it.ID != id {
			out = append(out, it)
		}
	}
	return out
}

func upsertRecoveryItem(items []RecoveryItem, item RecoveryItem) []RecoveryItem {
	for i := range items {
		if items[i].ID == item.ID {
			items[i] = item
			return items
		}
	}
	return append(items, item)
}

// recoveryItemOf 把一个会话投影成报告条目。
func recoveryItemOf(e SessionEntry, reason string) RecoveryItem {
	return RecoveryItem{
		ID:     e.ID,
		Type:   e.Type,
		Title:  e.Title,
		JobID:  jobParamOf(e),
		Reason: reason,
		At:     time.Now(),
	}
}

// HandleRecoveryReport serves GET /api/recovery → 200 RecoveryReport.
//
// 报告永远可读（文件缺失/损坏 → 空报告，绝不 500）：它是前端「平台升级后谁被
// 挂起」横幅的唯一数据源，读不到也必须让页面正常渲染。
func (m *SessionManager) HandleRecoveryReport(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, m.ReadRecoveryReport())
}
