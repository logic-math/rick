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

// SessionDraftStore persists per-key draft texts（草稿）to
// ~/.rick/web/drafts.json, keyed by draft key（sessionID 或 "new-session-<type>"）.
//
// Bug #5 迭代5b（human 2026-11-14）：草稿存服务端——跨设备输入不中断
//（localStorage 只在本机浏览器；服务端存储 + 前端防抖 PUT 高频更新）。
//
// 高频写入：PUT 前端防抖 500ms，服务端内存 map + 互斥串行落盘（文件小，
// 单次全量写延迟可忽略；草稿容忍度高，无需 per-key 文件）。
type SessionDraftStore struct {
	path string

	mu     sync.Mutex
	drafts map[string]string
}

// DraftsPath returns the on-disk location of drafts.json.
func DraftsPath() string {
	return filepath.Join(WebStateDir(), "drafts.json")
}

// LoadSessionDraftStore loads the draft file. A missing file yields an empty
// store (not an error); a malformed file is an error.
func LoadSessionDraftStore(path string) (*SessionDraftStore, error) {
	s := &SessionDraftStore{path: path, drafts: map[string]string{}}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return s, nil
		}
		return nil, fmt.Errorf("read drafts %s: %w", path, err)
	}
	if len(strings.TrimSpace(string(data))) == 0 {
		return s, nil
	}
	var file struct {
		Version int               `json:"version"`
		Drafts  map[string]string `json:"drafts"`
	}
	if err := json.Unmarshal(data, &file); err != nil {
		return nil, fmt.Errorf("parse drafts %s: %w", path, err)
	}
	if file.Drafts != nil {
		s.drafts = file.Drafts
	}
	return s, nil
}

// MaxDraftLen caps a single draft (rune count)——防滥用（超长即拒绝）。
const MaxDraftLen = 16 * 1024

// validateDraftKey：key 限长 + 字符集（字母数字-_:.，无路径分隔符——纯 map 键
// 安全，但约束住避免畸形键）。
func validateDraftKey(key string) error {
	if key == "" || len(key) > 128 {
		return newValidationError("invalid_params", "draft key must be 1-128 bytes")
	}
	for _, r := range key {
		ok := r == '-' || r == '_' || r == ':' || r == '.' ||
			(r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9')
		if !ok {
			return newValidationError("invalid_params", "draft key has invalid character %q", string(r))
		}
	}
	return nil
}

// Get returns the draft for a key ("" when absent).
func (s *SessionDraftStore) Get(key string) string {
	if s == nil || key == "" {
		return ""
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.drafts[key]
}

// Set assigns (or clears, when content is empty) the draft for a key.
func (s *SessionDraftStore) Set(key, content string) error {
	if s == nil {
		return fmt.Errorf("draft store is not configured")
	}
	if utf8.RuneCountInString(content) > MaxDraftLen {
		return newValidationError("invalid_params", "draft too long (max %d runes)", MaxDraftLen)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if content == "" {
		delete(s.drafts, key)
	} else {
		s.drafts[key] = content
	}
	return s.saveLocked()
}

// saveLocked persists the store atomically (tmp + rename). Caller holds mu.
func (s *SessionDraftStore) saveLocked() error {
	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return fmt.Errorf("mkdir drafts dir: %w", err)
	}
	data, err := json.MarshalIndent(struct {
		Version int               `json:"version"`
		Drafts  map[string]string `json:"drafts"`
	}{Version: 1, Drafts: s.drafts}, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal drafts: %w", err)
	}
	tmp := s.path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return fmt.Errorf("write drafts tmp: %w", err)
	}
	if err := os.Rename(tmp, s.path); err != nil {
		return fmt.Errorf("rename drafts: %w", err)
	}
	return nil
}
