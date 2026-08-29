package handler

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/sunquan/rick/internal/workspace"
)

// core.go 承载「cwd 解析」与「路径参数化辅助」的共享层。
//
// v4.5（job_36 task2）：handler 从「cwd 隐式解析 rickDir」重构为「显式路径参数
// core + CLI 包装」。规则：
//   - 每个 cmd 的完整逻辑落在 xxxCore(rickDir, ...)（或导出的 XxxIn）里，core 内
//     一律 filepath.Join(rickDir, ...) 派生路径，禁止调用 workspace.GetRickDir /
//     NextJobID / GetJobPlanDir / GetDraftDir / GetJobsDir 等 cwd 依赖函数；
//   - CLI 包装（Plan/Easy/... 签名不变）经 rickDirFromCwd 解析后进 core；
//   - web 入口（rick web 多工作区）经导出的 XxxIn(rickDir, ...) 进 core，
//     rickDir 来自 session 锚定的工作区（workspace 根 + "/.rick"）。
//
// 本文件持有 handler 包内唯一一处 GetRickDir 调用点（CLI 语义的唯一出口），
// 供全包 19 个 CLI 包装复用——web 路径零 cwd 依赖。

// rickDirFromCwd resolves the .rick directory from the process working
// directory. It is the single cwd-resolution point for all CLI wrappers in
// this package; cores and web-facing XxxIn variants never call it.
func rickDirFromCwd() (string, error) {
	return workspace.GetRickDir()
}

// nextJobIDIn scans <rickDir>/jobs and returns the next job_N id (path-
// parameterized replacement for workspace.NextJobID, which resolves from cwd).
// If no jobs exist yet, returns "job_1".
func nextJobIDIn(rickDir string) (string, error) {
	jobsDir := filepath.Join(rickDir, workspace.JobsDirName)
	entries, err := os.ReadDir(jobsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return "job_1", nil
		}
		return "", fmt.Errorf("failed to read jobs directory: %w", err)
	}

	maxN := 0
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		var n int
		if _, err := fmt.Sscanf(e.Name(), "job_%d", &n); err == nil && n > 0 && n <= 9999 && n > maxN {
			maxN = n
		}
	}
	return fmt.Sprintf("job_%d", maxN+1), nil
}

// nextLoopIDIn scans <draftDir>/loops and returns the next loop_N id.
// workspace.NextLoopID is already path-parameterized, so this is a thin alias
// keeping the In-naming convention for the core helpers.
func nextLoopIDIn(draftDir string) (string, error) {
	return workspace.NextLoopID(draftDir)
}

// ensureWorkspaceDirsIn mirrors workspace.New()'s directory bootstrap for an
// explicit rickDir (workspace.New resolves from cwd and has no path-
// parameterized constructor). Idempotent MkdirAll, same six directories.
func ensureWorkspaceDirsIn(rickDir string) error {
	dirs := []string{
		rickDir,
		filepath.Join(rickDir, workspace.LoopsDirName),
		filepath.Join(rickDir, workspace.SkillsDirName),
		filepath.Join(rickDir, workspace.DomainDirName),
		filepath.Join(rickDir, workspace.JobsDirName),
		filepath.Join(rickDir, workspace.DreamDirName),
	}
	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to ensure directory %s: %w", dir, err)
		}
	}
	return nil
}

// workspaceRoot returns the workspace root (the parent of the .rick
// directory). Web sessions anchor their pi subprocesses here.
func workspaceRoot(rickDir string) string {
	return filepath.Dir(rickDir)
}

// DoingEvent is a task status transition observed while a doing run is in
// progress. Web (rick web) consumes these through the progress callback to
// stream jobs_update events over SSE; the CLI callback prints the same
// transitions to stdout (unchanged behavior).
type DoingEvent struct {
	JobID  string
	TaskID string
	From   string
	To     string
}

// DreamEvent reports dream run progress (pending-job selection / completion).
// Web streams these as session events; the CLI ignores them (its informational
// prints live inside dreamCore and are unchanged).
type DreamEvent struct {
	Phase  string   // "selected" | "completed"
	JobIDs []string // jobs selected for this dream run
}

// emitDoing invokes a progress callback if non-nil (web SSE streaming).
func emitDoing(progress func(DoingEvent), ev DoingEvent) {
	if progress != nil {
		progress(ev)
	}
}

// emitDream invokes a progress callback if non-nil (web SSE streaming).
func emitDream(progress func(DreamEvent), ev DreamEvent) {
	if progress != nil {
		progress(ev)
	}
}
