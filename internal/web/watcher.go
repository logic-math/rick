// watcher.go 是 rick web 的自实现轮询 watcher（v1 禁引 fsnotify——不改 go.mod）：
//
//	① ~/.rick/web/dist        变更（debounce 500ms）→ Hub frontend_reload（自迭代自动生效）
//	② 各注册工作区 <ws>/.rick/jobs/*/doing/tasks.json 变更 → Hub jobs_update（看板实时）
//
// H4 硬约束：监听目录不存在 → 跳过并轮询等待目录出现，绝不 fatal——
// 首次启动（干净 HOME）时 ~/.rick/web 尚不存在，服务必须照常起。
//
// tasks.json diff 逻辑复刻 handler.watchTasksJSONIn 的思路（快照 diff），
// 但在 internal/web 内自实现轻量版（写域约束），并多带 workspace_id 维度。
//
// 工作区增删：StartWatchers 持有 WorkspaceRegistry 引用，每轮重新枚举
// 注册工作区（watcher 动态增删的自然实现——无注册回调可用）。
package web

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// watcherInterval is the polling cadence (task spec: 2s mtime 轮询).
const watcherInterval = 2 * time.Second

// frontendReloadDebounce suppresses reload storms while a build is still
// writing files: the event fires only after the tree stays stable for this
// long since the last observed change (task spec: 500ms).
const frontendReloadDebounce = 500 * time.Millisecond

// StartWatchers runs the polling watchers until ctx is cancelled. It never
// returns an error for missing directories (H4) — a missing dist dir is
// polled for appearance, missing workspaces are skipped for that round.
//
//	distDir: the overlay dist root (typically WebStateDir()/dist — may not exist)
//	hub:     events are published here (frontend_reload / jobs_update)
//	ws:      the workspace registry (re-enumerated each round)
func StartWatchers(ctx context.Context, distDir string, hub *Hub, ws *WorkspaceRegistry) {
	go watchFrontend(ctx, distDir, hub)
	go watchJobs(ctx, hub, ws)
}

// watchFrontend polls the overlay dist tree and publishes frontend_reload
// after a debounce window of stability. mtime signature = (name, size, mtime)
// of every regular file (top 2 levels — index.html + assets/* covers the
// Vite layout; deeper trees just make the signature longer, not different).
func watchFrontend(ctx context.Context, distDir string, hub *Hub) {
	var lastSig string
	var pendingChange time.Time
	var pendingSig string
	ticker := time.NewTicker(watcherInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
		// Missing dir: skip (H4) — keep lastSig; a later appearance is a change.
		if distDir == "" {
			continue
		}
		if _, err := os.Lstat(distDir); err != nil {
			continue
		}
		sig := dirSignature(distDir, 2)
		if sig == lastSig {
			// Stable since the last poll: if a change was pending and the
			// debounce window has elapsed, fire the reload.
			if !pendingChange.IsZero() && time.Since(pendingChange) >= frontendReloadDebounce {
				lastSig = pendingSig
				pendingChange = time.Time{}
				hub.Publish(FrontendReloadEvent())
			}
			continue
		}
		// Changed (or first sighting): record and wait for stability.
		if pendingChange.IsZero() {
			pendingChange = time.Now()
			pendingSig = sig
		} else {
			// Still churning: extend the debounce anchor to the newest observation.
			pendingSig = sig
			// Keep the ORIGINAL pendingChange (stability is measured from the
			// first change, not the last: a build writing for >2s must not
			// starve the reload forever).
			if time.Since(pendingChange) >= 5*time.Second {
				// Churned for >5s without a stable poll — flush anyway (a
				// busy build would otherwise delay the reload indefinitely).
				lastSig = sig
				pendingChange = time.Time{}
				hub.Publish(FrontendReloadEvent())
			}
		}
	}
}

// watchJobs polls every registered workspace's jobs/*/doing/tasks.json and
// publishes jobs_update with the per-task diff since the previous sighting.
func watchJobs(ctx context.Context, hub *Hub, ws *WorkspaceRegistry) {
	// state: "<wsID>\x00<jobID>" → last task snapshot
	prev := make(map[string]map[string]string)
	ticker := time.NewTicker(watcherInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
		if ws == nil || hub == nil {
			continue
		}
		for _, w := range ws.List() {
			rickDir := filepath.Join(w.Path, ".rick")
			jobsDir := filepath.Join(rickDir, "jobs")
			entries, err := os.ReadDir(jobsDir)
			if err != nil {
				continue // missing/unreadable: skip this round (H4 同款不 fatal)
			}
			// Track which keys were seen this round for this workspace so
			// removed jobs drop out of the diff base.
			seen := make(map[string]bool)
			for _, e := range entries {
				if !e.IsDir() {
					continue
				}
				jobID := e.Name()
				tasksPath := filepath.Join(jobsDir, jobID, "doing", "tasks.json")
				key := w.ID + "\x00" + jobID
				seen[key] = true
				cur, err := readTaskStatuses(tasksPath)
				if err != nil {
					continue // mid-write / corrupt: skip this poll
				}
				old, had := prev[key]
				if !had {
					prev[key] = cur
					continue // first sighting: baseline, no event
				}
				diff := diffTaskStatuses(old, cur)
				if len(diff) == 0 {
					prev[key] = cur
					continue
				}
				snapshot := make([]TaskBrief, 0, len(cur))
				// Order by task id for a stable snapshot payload.
				ids := make([]string, 0, len(cur))
				for id := range cur {
					ids = append(ids, id)
				}
				sort.Strings(ids)
				for _, id := range ids {
					snapshot = append(snapshot, TaskBrief{TaskID: id, Status: cur[id]})
				}
				hub.Publish(JobsUpdate(w.ID, jobID, diff, snapshot))
				prev[key] = cur
			}
			// Drop keys of jobs that vanished (job dir renamed/removed).
			for key := range prev {
				if seen[key] {
					continue
				}
				if strings.HasPrefix(key, w.ID+"\x00") {
					delete(prev, key)
				}
			}
		}
	}
}

// readTaskStatuses reads tasks.json into {task_id: status} (nil map on any
// error — the caller skips).
func readTaskStatuses(path string) (map[string]string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var tj struct {
		Tasks []struct {
			TaskID string `json:"task_id"`
			Status string `json:"status"`
		} `json:"tasks"`
	}
	if err := json.Unmarshal(data, &tj); err != nil {
		return nil, err
	}
	out := make(map[string]string, len(tj.Tasks))
	for _, t := range tj.Tasks {
		out[t.TaskID] = t.Status
	}
	return out, nil
}

// diffTaskStatuses computes the task transitions between two snapshots.
func diffTaskStatuses(old, cur map[string]string) []TaskDiff {
	var out []TaskDiff
	ids := make([]string, 0, len(cur)+len(old))
	seen := make(map[string]bool, len(cur)+len(old))
	for id := range cur {
		ids = append(ids, id)
		seen[id] = true
	}
	for id := range old {
		if !seen[id] {
			ids = append(ids, id)
		}
	}
	sort.Strings(ids)
	for _, id := range ids {
		from, hadOld := old[id]
		to, hadCur := cur[id]
		switch {
		case hadOld && !hadCur:
			out = append(out, TaskDiff{TaskID: id, From: from, To: "removed"})
		case !hadOld && hadCur:
			out = append(out, TaskDiff{TaskID: id, From: "", To: to})
		case from != to:
			out = append(out, TaskDiff{TaskID: id, From: from, To: to})
		}
	}
	return out
}

// dirSignature builds a stable string fingerprint of a directory tree up to
// maxDepth levels: sorted "name|size|mtime" per regular file. Used by the
// frontend watcher to detect overlay dist changes (npm build rewrites).
func dirSignature(dir string, maxDepth int) string {
	var sb strings.Builder
	var walk func(p string, depth int)
	walk = func(p string, depth int) {
		if depth > maxDepth {
			return
		}
		entries, err := os.ReadDir(p)
		if err != nil {
			return
		}
		names := make([]string, 0, len(entries))
		byName := make(map[string]os.DirEntry, len(entries))
		for _, e := range entries {
			names = append(names, e.Name())
			byName[e.Name()] = e
		}
		sort.Strings(names)
		for _, n := range names {
			e := byName[n]
			full := filepath.Join(p, n)
			if e.IsDir() {
				sb.WriteString("d:" + n + "/;")
				walk(full, depth+1)
				continue
			}
			info, err := e.Info()
			if err != nil {
				continue
			}
			if info.Mode()&os.ModeSymlink != 0 {
				sb.WriteString("l:" + n + ";")
				continue
			}
			rel, err := filepath.Rel(dir, full)
			if err != nil {
				continue
			}
			sb.WriteString(rel + "|" + itoa(info.Size()) + "|" + itoa(info.ModTime().UnixNano()) + ";")
		}
	}
	walk(dir, 1)
	return sb.String()
}

// itoa avoids importing strconv for two tiny call sites (kept dependency-free).
func itoa(n int64) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var buf [24]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}
