package web

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

// 恢复报告的磁盘契约（human 2026-11-28 单轨化后）：
//
//	<stateDir>/recovery-report.json  恢复报告（挂起清单 + 已恢复 + 失败）
//
// Daemon 单轨（关停 = detach，pi 全程活着）后，关停不再打断任何会话——
// 旧的 suspend.json「关停快照」（关停那一刻谁在跑的「遗嘱」）随之废弃并
// 删除。挂起的唯一来源变成「pi 真的死了」（进程崩溃/被杀）——注册表
// status=suspended 即权威，ReconcileOnStart 会探活恢复 live daemon。
//
// ⚠️ 不做自动续跑（human 裁决 J-L6-6/J-L6-7：实测 pi 不修复悬挂
// toolCall → 自动续跑会重复副作用）；真正的恢复动作只有一个人工入口：
// POST /api/sessions/{id}/continue（auto_resume_on_start 只做 spawn 恢复兜底）。
const (
	// RecoveryReportFile 是恢复报告文件名（位于 state dir 根）。
	RecoveryReportFile = "recovery-report.json"
	// recoveryReportVersion 是恢复报告的 schema 版本。
	recoveryReportVersion = 1
)

// SuspendReasonRestart 是「服务重启/pi 消亡」导致的挂起原因。
const SuspendReasonRestart = "server restart"

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

func recoveryReportPath(stateDir string) string {
	return filepath.Join(stateDir, RecoveryReportFile)
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
// report (报告是展示层的，读不到绝不能让接口 500)。

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
