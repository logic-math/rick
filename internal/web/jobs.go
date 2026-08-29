package web

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/sunquan/rick/internal/workspace"
)

// MaxReadFileSize caps any single file served through the jobs/knowledge
// read APIs (2MB). raw_session_coding.log and friends can grow large; the
// frontend renders a "too large" hint instead of the body.
const MaxReadFileSize = 2 << 20

// WebError is the data-layer error surfaced to the HTTP layer (task13 maps
// it directly): Status + stable snake_case Code feed the api-contract error
// body {"error":{"code","message"}}.
type WebError struct {
	Code    string
	Message string
	Status  int
}

func (e *WebError) Error() string {
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

func newWebError(status int, code, format string, args ...any) *WebError {
	return &WebError{Code: code, Message: fmt.Sprintf(format, args...), Status: status}
}

func errInvalidPath(format string, args ...any) *WebError {
	return newWebError(http.StatusBadRequest, "invalid_path", format, args...)
}

func errNotFound(format string, args ...any) *WebError {
	return newWebError(http.StatusNotFound, "not_found", format, args...)
}

// TaskBrief is the per-task projection of a job's tasks.json
// (api-contract.md Jobs 节: task_id/name/status/commit_hash).
type TaskBrief struct {
	TaskID     string `json:"task_id"`
	Name       string `json:"name"`
	Status     string `json:"status"`
	CommitHash string `json:"commit_hash"`
}

// JobSummary is one job's listing entry (api-contract.md Jobs 节).
type JobSummary struct {
	JobID     string      `json:"job_id"`
	UpdatedAt time.Time   `json:"updated_at"`
	Tasks     []TaskBrief `json:"tasks"`
}

// FileNode is one knowledge file entry (api-contract.md Knowledge 节).
type FileNode struct {
	Path string `json:"path"`
	Size int64  `json:"size"`
}

// jobTasksFile is the minimal tasks.json reader for ListJobs: top-level
// updated_at + the task fields the listing projects (same pattern as
// workspace's unexported thinTaskState, extended with name/commit_hash).
type jobTasksFile struct {
	UpdatedAt string         `json:"updated_at"`
	Tasks     []jobTaskEntry `json:"tasks"`
}

type jobTaskEntry struct {
	TaskID     string `json:"task_id"`
	TaskName   string `json:"task_name"`
	Status     string `json:"status"`
	CommitHash string `json:"commit_hash"`
}

// ListJobs scans <rickDir>/jobs/*/doing/tasks.json and returns one
// JobSummary per parseable job, sorted by updated_at descending (job id
// ascending as tie-break). Unparseable/corrupt files are skipped — a
// listing must not fail because one job's tasks.json is mid-write.
func ListJobs(rickDir string) ([]JobSummary, error) {
	if err := requireRickDir(rickDir); err != nil {
		return nil, err
	}
	pattern := filepath.Join(rickDir, workspace.JobsDirName, "*", workspace.DoingDirName, "tasks.json")
	files, err := filepath.Glob(pattern)
	if err != nil {
		return nil, fmt.Errorf("glob jobs under %s: %w", rickDir, err)
	}
	out := make([]JobSummary, 0, len(files))
	for _, f := range files {
		data, err := os.ReadFile(f)
		if err != nil {
			continue
		}
		var tj jobTasksFile
		if err := json.Unmarshal(data, &tj); err != nil {
			continue
		}
		// .rick/jobs/<jobID>/doing/tasks.json → <jobID>
		jobID := filepath.Base(filepath.Dir(filepath.Dir(f)))
		summary := JobSummary{
			JobID:     jobID,
			UpdatedAt: parseTimeLenient(tj.UpdatedAt),
			Tasks:     make([]TaskBrief, 0, len(tj.Tasks)),
		}
		for _, t := range tj.Tasks {
			summary.Tasks = append(summary.Tasks, TaskBrief{
				TaskID:     t.TaskID,
				Name:       t.TaskName,
				Status:     t.Status,
				CommitHash: t.CommitHash,
			})
		}
		out = append(out, summary)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].UpdatedAt.Equal(out[j].UpdatedAt) {
			return out[i].JobID < out[j].JobID
		}
		return out[i].UpdatedAt.After(out[j].UpdatedAt)
	})
	return out, nil
}

// ReadTasks returns the job's tasks.json verbatim (api-contract: "tasks.json
// 原文"). The bytes are hook-written; the data layer does not re-shape them.
func ReadTasks(rickDir, jobID string) (json.RawMessage, error) {
	root, err := jobRoot(rickDir, jobID)
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(filepath.Join(root, workspace.DoingDirName, "tasks.json"))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, errNotFound("no tasks.json for job %s", jobID)
		}
		return nil, fmt.Errorf("read tasks.json for job %s: %w", jobID, err)
	}
	return json.RawMessage(data), nil
}

// ReadJobFile serves a file from the job directory, restricted to the
// plan/** and doing/** whitelist (api-contract Jobs 节: debug/, act-path.md,
// raw_session_coding.log all live under doing/). Path traversal, absolute
// paths, symlink components and files over MaxReadFileSize are rejected.
func ReadJobFile(rickDir, jobID, relPath string) (string, error) {
	root, err := jobRoot(rickDir, jobID)
	if err != nil {
		return "", err
	}
	clean, err := sanitizeWhitelistPath(relPath, []string{workspace.PlanDirName, workspace.DoingDirName})
	if err != nil {
		return "", err
	}
	return readWhitelistedFile(root, clean)
}

// KnowledgeTree enumerates the knowledge files under
// .rick/{domain,loops,skills} (recursively) with relative slash paths and
// sizes, sorted by path. Only whitelisted text extensions are listed;
// symlinks and unreadable entries are skipped; missing roots are tolerated.
func KnowledgeTree(rickDir string) ([]FileNode, error) {
	if err := requireRickDir(rickDir); err != nil {
		return nil, err
	}
	nodes := make([]FileNode, 0, 64)
	for _, root := range KnowledgeRoots() {
		base := filepath.Join(rickDir, root)
		// Errors are swallowed per-entry: a tree listing must not fail on
		// one unreadable dir (missing root included).
		_ = filepath.WalkDir(base, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return nil
			}
			if d.Type()&fs.ModeSymlink != 0 {
				return nil // never traverse or list symlinks
			}
			if d.IsDir() {
				return nil
			}
			if !knowledgeExtAllowed(path) {
				return nil
			}
			info, ierr := d.Info()
			if ierr != nil {
				return nil
			}
			rel, rerr := filepath.Rel(rickDir, path)
			if rerr != nil {
				return nil
			}
			nodes = append(nodes, FileNode{Path: filepath.ToSlash(rel), Size: info.Size()})
			return nil
		})
	}
	sort.Slice(nodes, func(i, j int) bool { return nodes[i].Path < nodes[j].Path })
	return nodes, nil
}

// ReadKnowledgeFile serves a knowledge file, restricted to the
// domain/ loops/ skills/ roots, whitelisted text extensions, no symlink
// components, MaxReadFileSize cap (api-contract Knowledge 节).
func ReadKnowledgeFile(rickDir, relPath string) (string, error) {
	if err := requireRickDir(rickDir); err != nil {
		return "", err
	}
	clean, err := sanitizeWhitelistPath(relPath, KnowledgeRoots())
	if err != nil {
		return "", err
	}
	if !knowledgeExtAllowed(clean) {
		return "", errInvalidPath("file type not allowed: %s", relPath)
	}
	return readWhitelistedFile(rickDir, clean)
}

// KnowledgeRoots returns the whitelisted knowledge roots (api-contract:
// domain/ loops/ skills/), in stable order.
func KnowledgeRoots() []string {
	return []string{workspace.DomainDirName, workspace.LoopsDirName, workspace.SkillsDirName}
}

// knowledgeExts is the text-extension whitelist for the knowledge tree and
// file reads (api-contract: .md/.json/.txt/.yaml/.py — plus .ts, which the
// repo's skills carry, e.g. rick-gates/index.ts).
var knowledgeExts = map[string]bool{
	".md":   true,
	".json": true,
	".txt":  true,
	".yaml": true,
	".py":   true,
	".ts":   true,
}

func knowledgeExtAllowed(path string) bool {
	return knowledgeExts[strings.ToLower(filepath.Ext(path))]
}

// requireRickDir validates the workspace's .rick directory itself.
func requireRickDir(rickDir string) error {
	if fi, err := os.Stat(rickDir); err != nil || !fi.IsDir() {
		return newWebError(http.StatusBadRequest, "invalid_workspace", "workspace .rick directory not found: %s", rickDir)
	}
	return nil
}

// validJobID rejects path-shaped job ids ("..", "a/b", "../x"): the id is
// interpolated into a filesystem path, so it must be a single safe segment.
func validJobID(jobID string) bool {
	if jobID == "" || strings.HasPrefix(jobID, ".") {
		return false
	}
	for _, r := range jobID {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '_', r == '-':
		default:
			return false
		}
	}
	return true
}

// jobRoot resolves and validates <rickDir>/jobs/<jobID>, returning the job
// directory. Errors: invalid_workspace (bad rickDir), invalid_path (path-
// shaped jobID), not_found (unknown job).
func jobRoot(rickDir, jobID string) (string, error) {
	if err := requireRickDir(rickDir); err != nil {
		return "", err
	}
	if !validJobID(jobID) {
		return "", errInvalidPath("invalid job id: %q", jobID)
	}
	root := filepath.Join(rickDir, workspace.JobsDirName, jobID)
	if fi, err := os.Stat(root); err != nil || !fi.IsDir() {
		return "", errNotFound("job %s not found", jobID)
	}
	return root, nil
}

// sanitizeWhitelistPath normalizes relPath and enforces the whitelist
// roots: the cleaned path must stay inside one of roots/ (no traversal, no
// absolute path, no empty). Returns the cleaned slash path.
func sanitizeWhitelistPath(relPath string, roots []string) (string, error) {
	if relPath == "" {
		return "", errInvalidPath("empty path")
	}
	if filepath.IsAbs(relPath) {
		return "", errInvalidPath("absolute paths are not allowed: %s", relPath)
	}
	cleaned := filepath.ToSlash(filepath.Clean(relPath))
	// Contract literal: any ".." element in the raw input is rejected — even
	// when cleaning would land back inside the whitelist (predictability
	// beats cleverness on an HTTP-exposed surface).
	for _, part := range strings.Split(filepath.ToSlash(relPath), "/") {
		if part == ".." {
			return "", errInvalidPath("path escapes the whitelisted roots: %s", relPath)
		}
	}
	for _, part := range strings.Split(cleaned, "/") {
		if part == ".." {
			return "", errInvalidPath("path escapes the whitelisted roots: %s", relPath)
		}
	}
	allowed := false
	for _, r := range roots {
		if strings.HasPrefix(cleaned, r+"/") {
			allowed = true
			break
		}
	}
	if !allowed {
		return "", errInvalidPath("path must be under %s/: %s", strings.Join(roots, "/ or "), relPath)
	}
	return cleaned, nil
}

// readWhitelistedFile reads root/rel after rejecting symlink components
// (Lstat walk — this is an HTTP-exposed surface), directories as the final
// component, and oversized files. Note the walk is not TOCTOU-hardened
// against concurrent swaps: rick web is a single-user localhost service.
func readWhitelistedFile(root, rel string) (string, error) {
	parts := strings.Split(rel, "/")
	cur := root
	var lastSize int64
	for i, part := range parts {
		cur = filepath.Join(cur, part)
		fi, err := os.Lstat(cur)
		if err != nil {
			return "", errNotFound("no such file: %s", rel)
		}
		if fi.Mode()&os.ModeSymlink != 0 {
			return "", errInvalidPath("symlink components are not allowed: %s", rel)
		}
		if fi.IsDir() && i == len(parts)-1 {
			return "", errNotFound("not a regular file: %s", rel)
		}
		if i == len(parts)-1 {
			lastSize = fi.Size()
		}
	}
	if lastSize > MaxReadFileSize {
		return "", newWebError(http.StatusBadRequest, "file_too_large",
			"file %s is %d bytes (limit %d)", rel, lastSize, MaxReadFileSize)
	}
	data, err := os.ReadFile(filepath.Join(root, rel))
	if err != nil {
		if os.IsNotExist(err) {
			return "", errNotFound("no such file: %s", rel)
		}
		return "", fmt.Errorf("read %s: %w", rel, err)
	}
	return string(data), nil
}

// parseTimeLenient parses tasks.json timestamps as RFC3339 (the repo
// convention, timezone included). Anything else — empty, missing offset
// (legacy files, see bugs.md) — yields the zero time, which sorts last
// without pretending to know the author's timezone.
func parseTimeLenient(s string) time.Time {
	if s == "" {
		return time.Time{}
	}
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t
	}
	return time.Time{}
}
