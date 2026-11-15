package web

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Bug #8 2A 追加（human 裁决 2026-11-15：R2 方案——SSE 推送前端渲染，不改 pi-subagents
// 前台逻辑）：pi-subagents 的 async runner 把进度持久化到
// $TMPDIR/pi-subagents-uid-<uid>/async-subagent-runs/<runId>/status.json
// （AsyncStatus：runId/toolCallId/sessionId/state/steps[{agent,label,status,
// recentTools,turnCount}]）。此处 watcher 轮询这些文件（mtime 变化）→ 投影精简进度
// → hub 按 sessionId 路由发布 subagent_progress SSE 事件 → 前端按 toolCallId 匹配
// subagent 工具卡实时渲染进度（多 subagent 多卡）。

// subagentProgressEvent 是 subagent_progress SSE 事件的负载（前端渲染进度卡的数据源）。
type subagentProgressEvent struct {
	RunID     string                 `json:"runId"`
	ToolCall  string                 `json:"toolCallId"`
	State     string                 `json:"state"`
	Mode      string                 `json:"mode,omitempty"`
	StartedAt int64                  `json:"startedAt,omitempty"`
	UpdatedAt int64                  `json:"lastUpdate,omitempty"`
	Steps     []subagentProgressStep `json:"steps,omitempty"`
}

type subagentProgressStep struct {
	Agent       string `json:"agent"`
	Label       string `json:"label"`
	Status      string `json:"status"`
	Async       bool   `json:"async,omitempty"`
	CurrentTool string `json:"currentTool,omitempty"`
	LastActive  int64  `json:"lastActivityAt,omitempty"`
}

// asyncStatusRaw 是 status.json 的最小投影（pi-subagents AsyncStatus 的字段子集）。
type asyncStatusRaw struct {
	RunID      string `json:"runId"`
	ToolCallID string `json:"toolCallId"`
	SessionID  string `json:"sessionId"`
	Mode       string `json:"mode"`
	State      string `json:"state"`
	StartedAt  int64  `json:"startedAt"`
	LastUpdate int64  `json:"lastUpdate"`
	Steps      []struct {
		Agent          string `json:"agent"`
		Label          string `json:"label"`
		Status         string `json:"status"`
		Async          bool   `json:"async"`
		LastActivityAt int64  `json:"lastActivityAt"`
		RecentTools    []struct {
			Tool string `json:"tool"`
			Args string `json:"args"`
		} `json:"recentTools"`
	} `json:"steps"`
}

// subagentProgressDir 复刻 pi-subagents 的 async-runs 根目录
// （TEMP_ROOT_DIR = $PI_SUBAGENTS_TEMP_ROOT 或 $TMPDIR/pi-subagents-uid-<uid>）。
func subagentProgressDir() string {
	if v := os.Getenv("PI_SUBAGENTS_TEMP_ROOT"); v != "" {
		return filepath.Join(v, "async-subagent-runs")
	}
	return filepath.Join(os.TempDir(), fmt.Sprintf("pi-subagents-uid-%d", os.Getuid()), "async-subagent-runs")
}

// StartSubagentProgressWatcher 启动 async 进度 watcher：轮询（1s）所有
// status.json 的 mtime → 变化即投影发布。sessionId（status.json 里是父会话
// JSONL 完整路径）按 basename 前缀匹配 web 会话的 PISessionID。
func (m *SessionManager) StartSubagentProgressWatcher() {
	go func() {
		dir := subagentProgressDir()
		mt := map[string]time.Time{}
		tk := time.NewTicker(time.Second)
		defer tk.Stop()
		for range tk.C {
			entries, err := os.ReadDir(dir)
			if err != nil {
				continue // 目录不存在（无 async run）——静默跳过
			}
			seen := map[string]bool{}
			for _, e := range entries {
				if !e.IsDir() {
					continue
				}
				sp := filepath.Join(dir, e.Name(), "status.json")
				fi, err := os.Stat(sp)
				if err != nil {
					continue
				}
				seen[sp] = true
				if t, ok := mt[sp]; ok && !fi.ModTime().After(t) {
					continue // 未变化
				}
				mt[sp] = fi.ModTime()
				m.publishSubagentProgress(sp)
			}
			for sp := range mt {
				if !seen[sp] {
					delete(mt, sp) // run 目录已清理
				}
			}
		}
	}()
}

// publishSubagentProgress 读一个 status.json → 投影 → 按父会话路由发布。
func (m *SessionManager) publishSubagentProgress(statusPath string) {
	data, err := os.ReadFile(statusPath)
	if err != nil {
		return
	}
	var st asyncStatusRaw
	if err := json.Unmarshal(data, &st); err != nil || st.ToolCallID == "" {
		return
	}
	ev := subagentProgressEvent{
		RunID:     st.RunID,
		ToolCall:  st.ToolCallID,
		State:     st.State,
		Mode:      st.Mode,
		StartedAt: st.StartedAt,
		UpdatedAt: st.LastUpdate,
	}
	for _, s := range st.Steps {
		cur := ""
		if n := len(s.RecentTools); n > 0 {
			rt := s.RecentTools[n-1]
			args := rt.Args
			if len(args) > 60 {
				args = args[:60] + "…"
			}
			cur = rt.Tool + " " + args
		}
		ev.Steps = append(ev.Steps, subagentProgressStep{
			Agent: s.Agent, Label: s.Label, Status: s.Status, Async: s.Async,
			CurrentTool: cur, LastActive: s.LastActivityAt,
		})
	}
	// 父会话路由：status.sessionId 是 pi 会话 JSONL 完整路径；web 会话的
	// PISessionID（uuid）是其 basename 的尾部段。
	piSession := st.SessionID
	if i := strings.LastIndexAny(piSession, "/\\"); i >= 0 {
		piSession = piSession[i+1:]
	}
	for _, e := range m.sessions.List() {
		if e.PISessionID != "" && strings.Contains(piSession, e.PISessionID) {
			// SessionEvent helper：{event: {...}} 形状（前端 env.data.event 解包进 applyEvent）。
			// 字段 camelCase（与前端 viewModel 的 ev.toolCallId/ev.runId 对齐）。
			m.hub.Publish(SessionEvent(e.ID, nil, map[string]any{
				"type":       "subagent_progress",
				"runId":      ev.RunID,
				"toolCallId": ev.ToolCall,
				"state":      ev.State,
				"steps":      ev.Steps,
				"lastUpdate": ev.UpdatedAt,
			}))
		}
	}
}
