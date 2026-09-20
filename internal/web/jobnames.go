package web

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"unicode/utf8"
)

// jobNameStoreVersion is the on-disk schema version of job-names.json.
const jobNameStoreVersion = 1

// MaxJobNameLen caps a user-assigned job display name (rune count). Longer
// names are rejected rather than silently truncated so the UI can show a
// precise error.
const MaxJobNameLen = 60

// JobNameStore persists user-assigned job display names（「任务名」）to
// ~/.rick/web/job-names.json, keyed "<workspaceID>/<jobID>".
//
// 为什么放在 web 状态目录而不是写进工作区：用户的工作区是 git 仓库，往
// .rick/jobs/<job>/ 里塞额外文件会变成一次意外的仓库改动（还会被 job 文件
// 列表 API 列出来）。别名纯属 web 展示层数据，因此与 sessions.json /
// archived.json 同级存放，删除该文件即回到「显示 job_N」的默认状态。
//
// 并发：HTTP 层可能并发改名 → 内部 mutex 串行化；每次写入原子落盘
// （tmp + rename，同 registry.go）。
type JobNameStore struct {
	path string

	mu    sync.Mutex
	names map[string]string
}

// jobNameKey is the in-memory/on-disk key for one job's alias.
func jobNameKey(workspaceID, jobID string) string {
	return workspaceID + "/" + jobID
}

// LoadJobNameStore loads the alias file. A missing file yields an empty store
// (not an error); a malformed file is an error.
func LoadJobNameStore(path string) (*JobNameStore, error) {
	s := &JobNameStore{path: path, names: map[string]string{}}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return s, nil
		}
		return nil, fmt.Errorf("read job names %s: %w", path, err)
	}
	if len(strings.TrimSpace(string(data))) == 0 {
		return s, nil
	}
	var file struct {
		Version int               `json:"version"`
		Names   map[string]string `json:"names"`
	}
	if err := json.Unmarshal(data, &file); err != nil {
		return nil, fmt.Errorf("parse job names %s: %w", path, err)
	}
	if file.Names != nil {
		s.names = file.Names
	}
	return s, nil
}

// Get returns the display name for a job ("" when never renamed).
func (s *JobNameStore) Get(workspaceID, jobID string) string {
	if s == nil || workspaceID == "" || jobID == "" {
		return ""
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.names[jobNameKey(workspaceID, jobID)]
}

// All returns a copy of the alias map (defensive copy — callers cannot mutate
// store state).
func (s *JobNameStore) All() map[string]string {
	if s == nil {
		return map[string]string{}
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make(map[string]string, len(s.names))
	for k, v := range s.names {
		out[k] = v
	}
	return out
}

// ValidateJobName normalizes and validates a requested display name. An empty
// (or whitespace-only) name means "clear the alias" and is allowed. The
// returned string is the trimmed name to store.
func ValidateJobName(name string) (string, error) {
	trimmed := strings.TrimSpace(name)
	// 名字里的换行/制表会破坏侧栏单行布局 → 折叠成单个空格
	trimmed = strings.Join(strings.Fields(trimmed), " ")
	if trimmed == "" {
		return "", nil
	}
	if utf8.RuneCountInString(trimmed) > MaxJobNameLen {
		return "", newValidationError("invalid_name", "任务名过长（最多 %d 字，当前 %d 字）", MaxJobNameLen, utf8.RuneCountInString(trimmed))
	}
	return trimmed, nil
}

// Set stores (or clears, when name is empty) a job's display name and flushes
// the file atomically. Clearing a non-existent alias is a no-op success.
func (s *JobNameStore) Set(workspaceID, jobID, name string) error {
	if s == nil {
		return fmt.Errorf("job name store not configured")
	}
	if workspaceID == "" || jobID == "" {
		return newValidationError("invalid_params", "workspace id and job id are required")
	}
	normalized, err := ValidateJobName(name)
	if err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	key := jobNameKey(workspaceID, jobID)
	if normalized == "" {
		if _, ok := s.names[key]; !ok {
			return nil
		}
		delete(s.names, key)
	} else {
		if s.names[key] == normalized {
			return nil // 幂等：同值不落盘
		}
		s.names[key] = normalized
	}
	return s.saveLocked()
}

// saveLocked writes the alias file atomically (caller holds s.mu).
func (s *JobNameStore) saveLocked() error {
	if s.path == "" {
		return fmt.Errorf("job name store path is empty")
	}
	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return fmt.Errorf("mkdir for job names: %w", err)
	}
	body, err := json.MarshalIndent(struct {
		Version int               `json:"version"`
		Names   map[string]string `json:"names"`
	}{Version: jobNameStoreVersion, Names: s.names}, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal job names: %w", err)
	}
	tmp, err := os.CreateTemp(filepath.Dir(s.path), ".job-names-*.tmp")
	if err != nil {
		return fmt.Errorf("create temp job names: %w", err)
	}
	tmpName := tmp.Name()
	defer func() { _ = os.Remove(tmpName) }()
	if _, err := tmp.Write(append(body, '\n')); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("write temp job names: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close temp job names: %w", err)
	}
	if err := os.Rename(tmpName, s.path); err != nil {
		return fmt.Errorf("rename job names onto %s: %w", s.path, err)
	}
	return nil
}
