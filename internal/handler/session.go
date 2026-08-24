package handler

import (
	"fmt"
	"os"
	"path/filepath"
)

// --- 统一 --resume 会话恢复（v4.4.9）---
//
// 各阶段（plan/human-loop/doing/easy）启动 pi 会话时用 --session-id <uuid>
// 创建（pi 语义：不存在则创建，存在则恢复），并把 id 落盘到各自目录的
// session_id 文件。--resume 读取该文件恢复同一会话（完整历史+上下文）。
// 助手集中于此，各 handler 复用。

// ensureSessionID 返回 dir/session_id 的会话 id：存在则复用（resume 语义），
// 不存在则生成新 uuid 并落盘。dir 不存在时创建。
func ensureSessionID(dir string) (string, error) {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", fmt.Errorf("create session dir: %w", err)
	}
	path := filepath.Join(dir, "session_id")
	if data, err := os.ReadFile(path); err == nil {
		if id := trimSpace(string(data)); id != "" {
			return id, nil
		}
	}
	id, err := generateUUID()
	if err != nil {
		return "", fmt.Errorf("generate session id: %w", err)
	}
	if err := os.WriteFile(path, []byte(id), 0644); err != nil {
		return "", fmt.Errorf("persist session id: %w", err)
	}
	return id, nil
}

// loadSessionIDStrict 读取 dir/session_id，不存在时报带指引的错误。
func loadSessionIDStrict(dir, what string) (string, error) {
	path := filepath.Join(dir, "session_id")
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("%s 无会话记录（%s 不存在）——先正常运行一次该阶段，再 --resume", what, path)
	}
	id := trimSpace(string(data))
	if id == "" {
		return "", fmt.Errorf("%s 会话记录为空：%s", what, path)
	}
	return id, nil
}

func trimSpace(s string) string {
	start, end := 0, len(s)
	for start < end && (s[start] == ' ' || s[start] == '\n' || s[start] == '\r' || s[start] == '\t') {
		start++
	}
	for end > start && (s[end-1] == ' ' || s[end-1] == '\n' || s[end-1] == '\r' || s[end-1] == '\t') {
		end--
	}
	return s[start:end]
}
