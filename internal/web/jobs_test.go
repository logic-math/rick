package web

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// writeFileT writes content creating parent dirs.
func writeFileT(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatalf("mkdir %s: %v", path, err)
	}
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

// jobsFixture builds a workspace .rick tree:
//
//	jobs/job_1  tasks.json (older, 2 tasks) + plan/task1.md + doing/{debug/log.md, raw_session_coding.log, prompts/x.md}
//	jobs/job_2  tasks.json (newer, 1 task)
//	jobs/job_3  tasks.json (corrupt JSON — must be skipped by the listing)
//	jobs/job_4  plan-only（有 plan/task*.md，无 doing/tasks.json）→ 列为
//	            stage=planned（必须能被执行 doing/ctrl 的会话表单选到）
func jobsFixture(t *testing.T) string {
	t.Helper()
	rick := filepath.Join(t.TempDir(), ".rick")
	writeFileT(t, filepath.Join(rick, "jobs", "job_1", "doing", "tasks.json"), fmt.Sprintf(`{
  "version": "1.0",
  "created_at": "2026-08-01T09:00:00+08:00",
  "updated_at": "2026-08-01T10:00:00+08:00",
  "tasks": [
    {"task_id": "task1", "task_name": "定义 spec", "task_file": "task1.md", "status": "success", "dependencies": null, "attempts": 0, "commit_hash": "abc123", "created_at": "2026-08-01T09:00:00+08:00", "updated_at": "2026-08-01T09:30:00+08:00"},
    {"task_id": "task2", "task_name": "产出实现", "task_file": "task2.md", "status": "pending", "dependencies": ["task1"], "attempts": 0, "commit_hash": "", "created_at": "2026-08-01T09:00:00+08:00", "updated_at": "2026-08-01T09:00:00+08:00"}
  ]
}`))
	writeFileT(t, filepath.Join(rick, "jobs", "job_1", "plan", "task1.md"), "# plan task1")
	writeFileT(t, filepath.Join(rick, "jobs", "job_1", "doing", "debug", "log.md"), "debug log")
	writeFileT(t, filepath.Join(rick, "jobs", "job_1", "doing", "raw_session_coding.log"), "raw session")
	writeFileT(t, filepath.Join(rick, "jobs", "job_1", "doing", "prompts", "x.md"), "prompt")
	writeFileT(t, filepath.Join(rick, "jobs", "job_2", "doing", "tasks.json"), `{
  "version": "1.0",
  "updated_at": "2026-08-02T12:00:00+08:00",
  "tasks": [
    {"task_id": "task1", "task_name": "新任务", "status": "running", "commit_hash": "def456"}
  ]
}`)
	writeFileT(t, filepath.Join(rick, "jobs", "job_3", "doing", "tasks.json"), `{not valid json`)
	writeFileT(t, filepath.Join(rick, "jobs", "job_4", "plan", "task1.md"), "# plan only")
	return rick
}

// knowledgeFixture builds .rick/{domain,loops,skills} with nested dirs, a
// binary to skip, and no other roots.
func knowledgeFixture(t *testing.T) string {
	t.Helper()
	rick := filepath.Join(t.TempDir(), ".rick")
	writeFileT(t, filepath.Join(rick, "domain", "bugs.md"), "# Bugs Domain")
	writeFileT(t, filepath.Join(rick, "domain", "arch.md"), "# Architecture")
	writeFileT(t, filepath.Join(rick, "domain", "logo.png"), "\x89PNG fake binary")
	writeFileT(t, filepath.Join(rick, "loops", "tdd.md"), "# TDD loop")
	writeFileT(t, filepath.Join(rick, "loops", "deprecated", "old.md"), "# old loop")
	writeFileT(t, filepath.Join(rick, "skills", "foo_skill", "skill.md"), "# foo skill")
	writeFileT(t, filepath.Join(rick, "skills", "foo_skill", "helper.py"), "print('hi')")
	writeFileT(t, filepath.Join(rick, "skills", "gates", "index.ts"), "export {}")
	return rick
}

func assertWebError(t *testing.T, err error, wantStatus int, wantCode string) {
	t.Helper()
	if err == nil {
		t.Fatalf("want error %s(%d), got nil", wantCode, wantStatus)
	}
	var we *WebError
	if !errors.As(err, &we) {
		t.Fatalf("want *WebError, got %T: %v", err, err)
	}
	if we.Code != wantCode || we.Status != wantStatus {
		t.Fatalf("want %s(%d), got %s(%d): %v", wantCode, wantStatus, we.Code, we.Status, we)
	}
}

func TestJobsListJobsOrderingAndFields(t *testing.T) {
	rick := jobsFixture(t)
	jobs, err := ListJobs(rick)
	if err != nil {
		t.Fatalf("ListJobs: %v", err)
	}
	// job_3 是损坏 JSON → 跳过；job_4 只有 plan（无 doing/tasks.json）→ 以
	// stage=planned 列出（plan 完成但未 doing 的 job 必须可被选来执行）。
	if len(jobs) != 3 {
		t.Fatalf("want 3 jobs, got %d: %+v", len(jobs), jobs)
	}
	byID := map[string]JobSummary{}
	for _, j := range jobs {
		byID[j.JobID] = j
	}
	if byID["job_4"].Stage != "planned" || len(byID["job_4"].Tasks) != 0 {
		t.Fatalf("job_4 should be stage=planned with no tasks: %+v", byID["job_4"])
	}
	if byID["job_1"].Stage != "doing" || byID["job_2"].Stage != "doing" {
		t.Fatalf("job_1/job_2 should be stage=doing: %+v", byID)
	}
	if _, skipped := byID["job_3"]; skipped {
		t.Fatal("job_3 (corrupt JSON) must be skipped")
	}
	// 排序按 updated_at desc：job_2（08-02）应排在 job_1（08-01）之前。
	idxOf := func(id string) int {
		for i, j := range jobs {
			if j.JobID == id {
				return i
			}
		}
		return -1
	}
	if idxOf("job_2") == -1 || idxOf("job_1") == -1 || idxOf("job_2") > idxOf("job_1") {
		t.Fatalf("want job_2 before job_1 (updated_at desc): %v", []string{jobs[0].JobID, jobs[1].JobID, jobs[2].JobID})
	}
	want := []TaskBrief{
		{TaskID: "task1", Name: "定义 spec", Status: "success", CommitHash: "abc123"},
		{TaskID: "task2", Name: "产出实现", Status: "pending", CommitHash: ""},
	}
	got := byID["job_1"].Tasks
	if len(got) != len(want) {
		t.Fatalf("job_1 task count: want %d, got %d", len(want), len(got))
	}
	for i, w := range want {
		if got[i] != w {
			t.Fatalf("task[%d]: want %+v, got %+v", i, w, got[i])
		}
	}
	// JSON projection must use contract field names.
	b, err := json.Marshal(byID["job_1"])
	if err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{`"job_id":"job_1"`, `"updated_at":`, `"tasks":[{"task_id":"task1","name":"定义 spec","status":"success","commit_hash":"abc123"}`} {
		if !strings.Contains(string(b), key) {
			t.Fatalf("json projection missing %s: %s", key, b)
		}
	}
}

func TestJobsListJobsEmptyAndInvalidWorkspace(t *testing.T) {
	// Empty workspace (rick dir exists, no jobs yet) → empty list, no error.
	rick := filepath.Join(t.TempDir(), ".rick")
	if err := os.MkdirAll(rick, 0755); err != nil {
		t.Fatal(err)
	}
	jobs, err := ListJobs(rick)
	if err != nil {
		t.Fatalf("ListJobs on empty workspace: %v", err)
	}
	if jobs == nil || len(jobs) != 0 {
		t.Fatalf("want empty non-nil list, got %#v", jobs)
	}
	nodes, err := KnowledgeTree(rick)
	if err != nil {
		t.Fatalf("KnowledgeTree on empty workspace: %v", err)
	}
	if len(nodes) != 0 {
		t.Fatalf("want empty tree, got %d nodes", len(nodes))
	}

	// Nonexistent rick dir → invalid_workspace (400).
	missing := filepath.Join(t.TempDir(), "nope")
	_, err = ListJobs(missing)
	assertWebError(t, err, 400, "invalid_workspace")
	_, err = KnowledgeTree(missing)
	assertWebError(t, err, 400, "invalid_workspace")
	_, err = ReadTasks(missing, "job_1")
	assertWebError(t, err, 400, "invalid_workspace")
	_, err = ReadJobFile(missing, "job_1", "plan/task1.md")
	assertWebError(t, err, 400, "invalid_workspace")
	_, err = ReadKnowledgeFile(missing, "domain/bugs.md")
	assertWebError(t, err, 400, "invalid_workspace")
}

func TestJobsReadTasks(t *testing.T) {
	rick := jobsFixture(t)
	raw, err := ReadTasks(rick, "job_1")
	if err != nil {
		t.Fatalf("ReadTasks: %v", err)
	}
	var probe struct {
		Tasks []map[string]any `json:"tasks"`
	}
	if err := json.Unmarshal(raw, &probe); err != nil {
		t.Fatalf("returned bytes are not valid JSON: %v", err)
	}
	if len(probe.Tasks) != 2 {
		t.Fatalf("want 2 tasks in raw passthrough, got %d", len(probe.Tasks))
	}

	// Unknown job → not_found.
	_, err = ReadTasks(rick, "job_9")
	assertWebError(t, err, 404, "not_found")

	// Path-shaped job ids → invalid_path.
	for _, bad := range []string{"..", ".", "../x", "a/b", "", "job_1 "} {
		_, err = ReadTasks(rick, bad)
		assertWebError(t, err, 400, "invalid_path")
	}
}

func TestJobsReadJobFileWhitelistAndTraversal(t *testing.T) {
	rick := jobsFixture(t)

	ok := map[string]string{
		"plan/task1.md":                "# plan task1",
		"doing/debug/log.md":           "debug log",
		"doing/raw_session_coding.log": "raw session",
		"doing/prompts/x.md":           "prompt",
	}
	for path, want := range ok {
		got, err := ReadJobFile(rick, "job_1", path)
		if err != nil {
			t.Fatalf("ReadJobFile(%q): %v", path, err)
		}
		if got != want {
			t.Fatalf("ReadJobFile(%q) = %q, want %q", path, got, want)
		}
	}

	bad := []struct {
		path   string
		status int
		code   string
	}{
		{"../config.json", 400, "invalid_path"},
		{"/etc/passwd", 400, "invalid_path"},
		{"doing/../../x", 400, "invalid_path"},
		{"plan/../../etc/passwd", 400, "invalid_path"},
		{"..", 400, "invalid_path"},
		{"", 400, "invalid_path"},
		{"learning/x.md", 400, "invalid_path"},
		{"draft/rfc/x.md", 400, "invalid_path"},
		{"plan", 400, "invalid_path"}, // bare root, no file
		{"plan/missing.md", 404, "not_found"},
		{"doing/prompts", 404, "not_found"}, // directory
	}
	for _, c := range bad {
		_, err := ReadJobFile(rick, "job_1", c.path)
		assertWebError(t, err, c.status, c.code)
	}

	// Symlink component → rejected (HTTP-exposed surface).
	outside := filepath.Join(t.TempDir(), "secret.txt")
	if err := os.WriteFile(outside, []byte("secret"), 0644); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(rick, "jobs", "job_1", "plan", "link.md")
	if err := os.Symlink(outside, link); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}
	_, err := ReadJobFile(rick, "job_1", "plan/link.md")
	assertWebError(t, err, 400, "invalid_path")

	// Oversized file → file_too_large.
	big := filepath.Join(rick, "jobs", "job_1", "plan", "big.log")
	if err := os.WriteFile(big, bytes.Repeat([]byte("x"), MaxReadFileSize+1), 0644); err != nil {
		t.Fatal(err)
	}
	_, err = ReadJobFile(rick, "job_1", "plan/big.log")
	assertWebError(t, err, 400, "file_too_large")

	// Path-shaped job id → invalid_path even for a well-formed rel path.
	_, err = ReadJobFile(rick, "../job_1", "plan/task1.md")
	assertWebError(t, err, 400, "invalid_path")
}

func TestJobsKnowledgeTree(t *testing.T) {
	rick := knowledgeFixture(t)
	nodes, err := KnowledgeTree(rick)
	if err != nil {
		t.Fatalf("KnowledgeTree: %v", err)
	}
	var paths []string
	for _, n := range nodes {
		paths = append(paths, n.Path)
	}
	want := []string{
		"domain/arch.md",
		"domain/bugs.md",
		"loops/deprecated/old.md",
		"loops/tdd.md",
		"skills/foo_skill/helper.py",
		"skills/foo_skill/skill.md",
		"skills/gates/index.ts",
	}
	if len(paths) != len(want) {
		t.Fatalf("tree mismatch:\nwant %v\ngot  %v", want, paths)
	}
	for i := range want {
		if paths[i] != want[i] {
			t.Fatalf("tree[%d]: want %s, got %s (not sorted or extra entry)", i, want[i], paths[i])
		}
	}
	// Sizes present and positive.
	for _, n := range nodes {
		if n.Size <= 0 {
			t.Fatalf("node %s has non-positive size %d", n.Path, n.Size)
		}
	}

	// A symlinked knowledge file is never listed.
	outside := filepath.Join(t.TempDir(), "outside.md")
	if err := os.WriteFile(outside, []byte("# outside"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(rick, "domain", "linked.md")); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}
	nodes, err = KnowledgeTree(rick)
	if err != nil {
		t.Fatal(err)
	}
	for _, n := range nodes {
		if n.Path == "domain/linked.md" {
			t.Fatal("symlinked knowledge file must not be listed")
		}
	}
}

func TestJobsReadKnowledgeFile(t *testing.T) {
	rick := knowledgeFixture(t)

	got, err := ReadKnowledgeFile(rick, "domain/bugs.md")
	if err != nil {
		t.Fatalf("ReadKnowledgeFile: %v", err)
	}
	if got != "# Bugs Domain" {
		t.Fatalf("content = %q", got)
	}

	bad := []struct {
		path   string
		status int
		code   string
	}{
		{"../web.json", 400, "invalid_path"},
		{"/etc/passwd", 400, "invalid_path"},
		{"domain/../domain/../../x", 400, "invalid_path"},
		{"jobs/../domain/bugs.md", 400, "invalid_path"}, // any ".." element is rejected per contract
		{"draft/rfc/x.md", 400, "invalid_path"},
		{"domain/logo.png", 400, "invalid_path"}, // binary ext
		{"domain/missing.md", 404, "not_found"},
		{"domain", 400, "invalid_path"},
		{"", 400, "invalid_path"},
	}
	for _, c := range bad {
		_, err := ReadKnowledgeFile(rick, c.path)
		assertWebError(t, err, c.status, c.code)
	}

	// Symlink → invalid_path.
	outside := filepath.Join(t.TempDir(), "secret.md")
	if err := os.WriteFile(outside, []byte("secret"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(rick, "domain", "linked.md")); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}
	_, err = ReadKnowledgeFile(rick, "domain/linked.md")
	assertWebError(t, err, 400, "invalid_path")

	// Oversized → file_too_large.
	writeFileT(t, filepath.Join(rick, "domain", "big.md"), "")
	if err := os.WriteFile(filepath.Join(rick, "domain", "big.md"), bytes.Repeat([]byte("x"), MaxReadFileSize+1), 0644); err != nil {
		t.Fatal(err)
	}
	_, err = ReadKnowledgeFile(rick, "domain/big.md")
	assertWebError(t, err, 400, "file_too_large")
}

func TestJobsListJobsSortTiebreakAndZeroTime(t *testing.T) {
	rick := filepath.Join(t.TempDir(), ".rick")
	// Same updated_at → job id ascending; empty/legacy timestamps → zero time (last).
	writeFileT(t, filepath.Join(rick, "jobs", "job_9", "doing", "tasks.json"), `{"updated_at":"2026-08-01T10:00:00+08:00","tasks":[{"task_id":"t","task_name":"n","status":"success"}]}`)
	writeFileT(t, filepath.Join(rick, "jobs", "job_2", "doing", "tasks.json"), `{"updated_at":"2026-08-01T10:00:00+08:00","tasks":[{"task_id":"t","task_name":"n","status":"success"}]}`)
	writeFileT(t, filepath.Join(rick, "jobs", "job_1", "doing", "tasks.json"), `{"updated_at":"","tasks":[]}`)                    // empty → zero time
	writeFileT(t, filepath.Join(rick, "jobs", "job_0", "doing", "tasks.json"), `{"updated_at":"2026-08-01T09:00:00","tasks":[]}`) // no timezone → zero time (no fake UTC precision)
	jobs, err := ListJobs(rick)
	if err != nil {
		t.Fatal(err)
	}
	if len(jobs) != 4 {
		t.Fatalf("want 4 jobs, got %d", len(jobs))
	}
	wantOrder := []string{"job_2", "job_9", "job_0", "job_1"}
	for i, w := range wantOrder {
		if jobs[i].JobID != w {
			t.Fatalf("order[%d]: want %s, got %s (all: %v)", i, w, jobs[i].JobID, jobs)
		}
	}
	// job_0 (no timezone) and job_1 (empty) both fall to zero time, sorting last.
	if !jobs[2].UpdatedAt.IsZero() || !jobs[3].UpdatedAt.IsZero() {
		t.Fatalf("legacy timestamps should be zero time, got %v / %v", jobs[2].UpdatedAt, jobs[3].UpdatedAt)
	}
}

// TestJobsSmokeRealRepo runs against the real repo .rick (read-only). Opt-in
// via RICK_WEB_SMOKE=1 — the gate runs it to prove the layer on real data.
func TestJobsSmokeRealRepo(t *testing.T) {
	if os.Getenv("RICK_WEB_SMOKE") != "1" {
		t.Skip("set RICK_WEB_SMOKE=1 to run the real-repo smoke check")
	}
	repo, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	rick := filepath.Join(repo, ".rick")

	jobs, err := ListJobs(rick)
	if err != nil {
		t.Fatalf("ListJobs(real repo): %v", err)
	}
	if len(jobs) == 0 {
		t.Fatal("real repo has jobs with doing/tasks.json — listing must not be empty")
	}
	found := false
	for _, j := range jobs {
		if j.JobID == "job_36" {
			found = true
			if len(j.Tasks) == 0 {
				t.Fatalf("job_36 has tasks in tasks.json, got empty projection: %+v", j)
			}
		}
	}
	if !found {
		t.Logf("note: job_36 not present (fresh checkout?) — %d jobs listed", len(jobs))
	}

	nodes, err := KnowledgeTree(rick)
	if err != nil {
		t.Fatalf("KnowledgeTree(real repo): %v", err)
	}
	hasBugs := false
	for _, n := range nodes {
		if n.Path == "domain/bugs.md" {
			hasBugs = true
		}
	}
	if !hasBugs {
		t.Fatal("real repo knowledge tree must contain domain/bugs.md")
	}
	content, err := ReadKnowledgeFile(rick, "domain/bugs.md")
	if err != nil {
		t.Fatalf("ReadKnowledgeFile(real repo): %v", err)
	}
	if !strings.HasPrefix(content, "# Bugs Domain") {
		t.Fatalf("domain/bugs.md content mismatch: %q", content[:40])
	}
}
