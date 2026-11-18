package web

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"sync"

	"path/filepath"
)

// sessionNameStoreVersion is the on-disk schema version of session-names.json.
const sessionNameStoreVersion = 1

// SessionNameStore persists user-assigned session display names（「会话名」）to
// ~/.rick/web/session-names.json, keyed "<workspaceID>/<sessionID>".
//
// Bug #4（human 2026-11-14）：所有会话类型都应能重命名（避免会话管理混乱）。
// 既有 JobNameStore 仅覆盖 job 类会话（params.job 存在，PUT jobs/{job}/name
// 还要求 job 目录真实存在）——super/human-loop/dream 等类型无 job 概念，永远
// 无法重命名。会话级别名按会话 id 键控，与 job 概念解耦（与 Bug #2 去 Jobs
// 方向一致）。
//
// 与 JobNameStore 同理由放 web 状态目录（工作区是 git 仓库，展示层别名不进
// 工作区）；并发写 mutex 串行化 + 原子落盘（tmp + rename）。
type SessionNameStore struct {
	path string

	mu    sync.Mutex
	names map[string]string
}

// sessionNameKey is the in-memory/on-disk key for one session's alias.
func sessionNameKey(workspaceID, sessionID string) string {
	return workspaceID + "/" + sessionID
}

// SessionNamesPath returns the session display-name file path
// (~/.rick/web/session-names.json) — Bug #4：所有会话类型可重命名的「会话名」
// 别名（展示层数据，不碰工作区）。
func SessionNamesPath() string {
	return filepath.Join(WebStateDir(), "session-names.json")
}

// LoadSessionNameStore loads the alias file. A missing file yields an empty
// store (not an error); a malformed file is an error.
func LoadSessionNameStore(path string) (*SessionNameStore, error) {
	s := &SessionNameStore{path: path, names: map[string]string{}}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return s, nil
		}
		return nil, fmt.Errorf("read session names %s: %w", path, err)
	}
	if len(strings.TrimSpace(string(data))) == 0 {
		return s, nil
	}
	var file struct {
		Version int               `json:"version"`
		Names   map[string]string `json:"names"`
	}
	if err := json.Unmarshal(data, &file); err != nil {
		return nil, fmt.Errorf("parse session names %s: %w", path, err)
	}
	if file.Names != nil {
		s.names = file.Names
	}
	return s, nil
}

// Get returns the display name for a session ("" when never renamed).
func (s *SessionNameStore) Get(workspaceID, sessionID string) string {
	if s == nil || workspaceID == "" || sessionID == "" {
		return ""
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.names[sessionNameKey(workspaceID, sessionID)]
}

// Set assigns (or clears, when name is empty) the display name for a session.
func (s *SessionNameStore) Set(workspaceID, sessionID, name string) error {
	if s == nil {
		return fmt.Errorf("session name store is not configured")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	key := sessionNameKey(workspaceID, sessionID)
	if name == "" {
		delete(s.names, key)
	} else {
		s.names[key] = name
	}
	return s.saveLocked()
}

// saveLocked persists the store atomically (tmp + rename). Caller holds mu.
func (s *SessionNameStore) saveLocked() error {
	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return fmt.Errorf("mkdir session names dir: %w", err)
	}
	data, err := json.MarshalIndent(struct {
		Version int               `json:"version"`
		Names   map[string]string `json:"names"`
	}{Version: sessionNameStoreVersion, Names: s.names}, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal session names: %w", err)
	}
	tmp := s.path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return fmt.Errorf("write session names tmp: %w", err)
	}
	if err := os.Rename(tmp, s.path); err != nil {
		return fmt.Errorf("rename session names: %w", err)
	}
	return nil
}
