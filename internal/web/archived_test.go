// archived_test.go covers the web-layer job soft-archive: store
// persistence (idempotent archive/unarchive), the filtered listing view
// (default hides archived jobs; include_archived=true marks them), the
// completion gate (only fully-success jobs may be archived), and the
// archive/unarchive HTTP routes.
package web

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/sunquan/rick/internal/workspace"
)

// writeJobTasks writes a minimal tasks.json for rickDir/jobs/jobID/doing/,
// so ListJobs / jobIsComplete have something to read.
func writeJobTasks(t *testing.T, rickDir, jobID string, statuses ...string) {
	t.Helper()
	doingDir := filepath.Join(rickDir, workspace.JobsDirName, jobID, workspace.DoingDirName)
	if err := os.MkdirAll(doingDir, 0o755); err != nil {
		t.Fatal(err)
	}
	type task struct {
		TaskID   string `json:"task_id"`
		Status   string `json:"status"`
		TaskName string `json:"task_name"`
	}
	tasks := make([]task, 0, len(statuses))
	for i, st := range statuses {
		tasks = append(tasks, task{TaskID: "task" + string(rune('1'+i)), Status: st, TaskName: "t"})
	}
	data := map[string]any{
		"version":    "1.0",
		"updated_at": "2026-08-31T12:00:00+08:00",
		"tasks":      tasks,
	}
	b, err := json.Marshal(data)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(doingDir, "tasks.json"), b, 0o644); err != nil {
		t.Fatal(err)
	}
}

// --- ArchivedStore unit ---

func TestArchivedStore_PersistenceAndIdempotence(t *testing.T) {
	path := filepath.Join(t.TempDir(), "archived.json")

	// Missing file → empty store, no error.
	s, err := LoadArchived(path)
	if err != nil {
		t.Fatalf("LoadArchived on missing file: %v", err)
	}
	if s.List("ws-a") != nil && len(s.List("ws-a")) != 0 {
		t.Fatalf("empty store should list nothing, got %v", s.List("ws-a"))
	}
	if s.IsArchived("ws-a", "job_1") {
		t.Fatal("nothing archived yet")
	}

	// Archive idempotent.
	if err := s.Archive("ws-a", "job_1"); err != nil {
		t.Fatalf("archive job_1: %v", err)
	}
	if err := s.Archive("ws-a", "job_1"); err != nil {
		t.Fatalf("re-archive job_1 (idempotent): %v", err)
	}
	if err := s.Archive("ws-a", "job_2"); err != nil {
		t.Fatalf("archive job_2: %v", err)
	}
	got := s.List("ws-a")
	if len(got) != 2 || got[0] != "job_1" || got[1] != "job_2" {
		t.Fatalf("List(ws-a) = %v, want [job_1 job_2]", got)
	}

	// Reload from disk → persisted.
	s2, err := LoadArchived(path)
	if err != nil {
		t.Fatalf("reload archived: %v", err)
	}
	if !s2.IsArchived("ws-a", "job_1") || !s2.IsArchived("ws-a", "job_2") {
		t.Fatal("persisted archive entries missing after reload")
	}

	// Unarchive idempotent + per-workspace isolation.
	if err := s2.Unarchive("ws-a", "job_1"); err != nil {
		t.Fatalf("unarchive job_1: %v", err)
	}
	if err := s2.Unarchive("ws-a", "job_1"); err != nil {
		t.Fatalf("re-unarchive job_1 (idempotent): %v", err)
	}
	if err := s2.Unarchive("ws-b", "job_9"); err != nil { // unknown ws/job → no-op
		t.Fatalf("unarchive unknown entry: %v", err)
	}
	if s2.IsArchived("ws-a", "job_1") {
		t.Fatal("job_1 should be unarchived")
	}
	if !s2.IsArchived("ws-a", "job_2") {
		t.Fatal("job_2 should stay archived")
	}
	if s2.IsArchived("ws-b", "job_9") {
		t.Fatal("ws-b must be untouched")
	}
}

// --- Listing view ---

func TestListJobsFiltered_DefaultHidesArchived_IncludeMarks(t *testing.T) {
	env := newTestEnv(t)
	writeJobTasks(t, env.rickDir, "job_1", "success")
	writeJobTasks(t, env.rickDir, "job_2", "success")
	writeJobTasks(t, env.rickDir, "job_3", "pending")

	archived := []string{"job_2"}

	// Default: only in-flight jobs — job_1 (done auto-archive) and job_2
	// (manual) hidden, job_3 (pending) visible.
	jobs, err := ListJobsFiltered(env.rickDir, archived, nil, false)
	if err != nil {
		t.Fatalf("ListJobsFiltered default: %v", err)
	}
	ids := jobIDs(jobs)
	if len(ids) != 1 || ids[0] != "job_3" {
		t.Fatalf("default view ids = %v, want [job_3] (job_1 done-archived, job_2 manual)", ids)
	}
	for _, j := range jobs {
		if j.Archived {
			t.Fatalf("default view must not mark Archived, got job %s", j.JobID)
		}
	}

	// include_archived=true: all three, each with the right source.
	all, err := ListJobsFiltered(env.rickDir, archived, nil, true)
	if err != nil {
		t.Fatalf("ListJobsFiltered include: %v", err)
	}
	if len(all) != 3 {
		t.Fatalf("include view = %d jobs, want 3", len(all))
	}
	byID := map[string]JobSummary{}
	for _, j := range all {
		byID[j.JobID] = j
	}
	if !byID["job_1"].Archived || byID["job_1"].ArchivedBy != "done" {
		t.Fatalf("job_1 include view = %+v, want archived=done (auto-archived completed)", byID["job_1"])
	}
	if !byID["job_2"].Archived || byID["job_2"].ArchivedBy != "manual" {
		t.Fatalf("job_2 include view = %+v, want archived=manual", byID["job_2"])
	}
	if byID["job_3"].Archived {
		t.Fatal("job_3 must not be archived")
	}

	// done auto-archive is independent of the manual set: no manual/dream
	// entries still filters completed jobs out of the default view.
	nilJobs, err := ListJobsFiltered(env.rickDir, nil, nil, false)
	if err != nil {
		t.Fatalf("ListJobsFiltered nil archived: %v", err)
	}
	if ids := jobIDs(nilJobs); len(ids) != 1 || ids[0] != "job_3" {
		t.Fatalf("nil archived default view = %v, want [job_3] (done archive still applies)", ids)
	}
}

func TestJobIsComplete_Gate(t *testing.T) {
	env := newTestEnv(t)
	writeJobTasks(t, env.rickDir, "done", "success", "success")
	writeJobTasks(t, env.rickDir, "running", "success", "pending")
	writeJobTasks(t, env.rickDir, "empty") // zero tasks

	if done, err := jobIsComplete(env.rickDir, "done"); err != nil || !done {
		t.Fatalf("all-success job should be complete (done=%v err=%v)", done, err)
	}
	if done, err := jobIsComplete(env.rickDir, "running"); err != nil || done {
		t.Fatalf("job with pending task must not be complete (done=%v err=%v)", done, err)
	}
	if done, err := jobIsComplete(env.rickDir, "empty"); err != nil || done {
		t.Fatalf("empty-tasks job must not be complete (done=%v err=%v)", done, err)
	}
	if _, err := jobIsComplete(env.rickDir, "nope"); err == nil {
		t.Fatal("unknown job should error")
	}
}

// --- HTTP routes ---

// archivedRoutes returns a ServeMux with a real RegisteredStore wired, for
// exercising archive/unarchive/list through the full route layer.
func archivedRoutes(t *testing.T) (*http.ServeMux, *ArchivedStore, *testEnv) {
	t.Helper()
	env := newTestEnv(t)
	archived, err := LoadArchived(filepath.Join(t.TempDir(), "archived.json"))
	if err != nil {
		t.Fatal(err)
	}
	mux := http.NewServeMux()
	RegisterRoutes(mux, Deps{
		Hub:        env.hub,
		Sessions:   nil,
		Workspaces: env.workspaces,
		Token:      "test-token",
		Static:     http.NotFoundHandler(),
		Version:    "test",
		Archived:   archived,
	})
	return mux, archived, env
}

func authPost(mux *http.ServeMux, url string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPost, url, nil)
	req.Header.Set("Authorization", "Bearer test-token")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	return rec
}

func authGet(mux *http.ServeMux, url string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, url, nil)
	req.Header.Set("Authorization", "Bearer test-token")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	return rec
}

func TestArchiveRoutes_ArchiveListUnarchive(t *testing.T) {
	mux, archived, env := archivedRoutes(t)
	writeJobTasks(t, env.rickDir, "job_1", "success")
	writeJobTasks(t, env.rickDir, "job_2", "pending")
	wsID := env.wsEntry.ID

	// Archive a completed job → 204.
	if rec := authPost(mux, "/api/workspaces/"+wsID+"/jobs/job_1/archive"); rec.Code != http.StatusNoContent {
		t.Fatalf("archive job_1: got %d, want 204; body=%s", rec.Code, rec.Body.String())
	}
	if !archived.IsArchived(wsID, "job_1") {
		t.Fatal("job_1 should be archived in store")
	}

	// Archive idempotent → 204 again.
	if rec := authPost(mux, "/api/workspaces/"+wsID+"/jobs/job_1/archive"); rec.Code != http.StatusNoContent {
		t.Fatalf("re-archive job_1: got %d, want 204", rec.Code)
	}

	// Archive an incomplete job → 409 state_conflict.
	if rec := authPost(mux, "/api/workspaces/"+wsID+"/jobs/job_2/archive"); rec.Code != http.StatusConflict {
		t.Fatalf("archive incomplete job: got %d, want 409; body=%s", rec.Code, rec.Body.String())
	}

	// Default list: job_1 hidden, job_2 visible.
	var list []JobSummary
	if rec := authGet(mux, "/api/workspaces/"+wsID+"/jobs"); rec.Code != http.StatusOK {
		t.Fatalf("list jobs: got %d", rec.Code)
	} else if err := json.Unmarshal(rec.Body.Bytes(), &list); err != nil {
		t.Fatalf("decode list: %v", err)
	}
	if len(list) != 1 || list[0].JobID != "job_2" {
		t.Fatalf("default list = %v, want only job_2 (job_1 archived)", jobIDs(list))
	}

	// include_archived=true: both, job_1 flagged.
	var all []JobSummary
	if rec := authGet(mux, "/api/workspaces/"+wsID+"/jobs?include_archived=true"); rec.Code != http.StatusOK {
		t.Fatalf("list all jobs: got %d", rec.Code)
	} else if err := json.Unmarshal(rec.Body.Bytes(), &all); err != nil {
		t.Fatalf("decode all list: %v", err)
	}
	if len(all) != 2 {
		t.Fatalf("include_archived list = %d jobs, want 2", len(all))
	}
	var found bool
	for _, j := range all {
		if j.JobID == "job_1" && !j.Archived {
			t.Fatal("job_1 must carry archived=true in include view")
		}
		if j.JobID == "job_1" {
			found = true
		}
	}
	if !found {
		t.Fatal("job_1 missing from include view")
	}

	// Unarchive → 204; job_1 is still hidden because completed jobs are
	// auto-archived (done) — unarchiving only lifts the manual marker.
	if rec := authPost(mux, "/api/workspaces/"+wsID+"/jobs/job_1/unarchive"); rec.Code != http.StatusNoContent {
		t.Fatalf("unarchive job_1: got %d, want 204", rec.Code)
	}
	var after []JobSummary
	_ = json.Unmarshal(authGet(mux, "/api/workspaces/"+wsID+"/jobs").Body.Bytes(), &after)
	if len(after) != 1 || after[0].JobID != "job_2" {
		t.Fatalf("after unarchive list = %v, want [job_2] (job_1 completed → still done-archived)", jobIDs(after))
	}
	// Include view: job_1 carries archived_by=done now (manual marker lifted).
	var all2 []JobSummary
	_ = json.Unmarshal(authGet(mux, "/api/workspaces/"+wsID+"/jobs?include_archived=true").Body.Bytes(), &all2)
	for _, j := range all2 {
		if j.JobID == "job_1" && (j.ArchivedBy != "done" || !j.Archived) {
			t.Fatalf("job_1 after unarchive = %+v, want archived=done", j)
		}
	}
}

func TestArchiveRoutes_UnknownJobOrWorkspace(t *testing.T) {
	mux, _, env := archivedRoutes(t)

	// Unknown job → 404 (jobRoot not_found).
	if rec := authPost(mux, "/api/workspaces/"+env.wsEntry.ID+"/jobs/ghost/archive"); rec.Code != http.StatusNotFound {
		t.Fatalf("archive unknown job: got %d, want 404", rec.Code)
	}

	// Unknown workspace → 404.
	if rec := authPost(mux, "/api/workspaces/nope/jobs/job_1/archive"); rec.Code != http.StatusNotFound {
		t.Fatalf("archive on unknown workspace: got %d, want 404", rec.Code)
	}

	// Missing token → 401.
	req := httptest.NewRequest(http.MethodPost, "/api/workspaces/"+env.wsEntry.ID+"/jobs/job_1/archive", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("no token: got %d, want 401", rec.Code)
	}
}

// jobIDs extracts job ids from a listing (test helper).
func jobIDs(jobs []JobSummary) []string {
	out := make([]string, 0, len(jobs))
	for _, j := range jobs {
		out = append(out, j.JobID)
	}
	return out
}

// writeDreamLog writes a dream_run_{job_id}_log.md into rickDir/dream/,
// marking jobID as dream-processed (auto-archive signal).
func writeDreamLog(t *testing.T, rickDir, jobID string) {
	t.Helper()
	dreamDir := filepath.Join(rickDir, workspace.DreamDirName)
	if err := os.MkdirAll(dreamDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dreamDir, "dream_run_"+jobID+"_log.md"), []byte("# dream log\n"), 0o644); err != nil {
		t.Fatal(err)
	}
}

// TestDreamArchivedJobs_ScansDreamLogs verifies DreamArchivedJobs reports
// exactly the jobs with a dream_run_{job_id}_log.md under .rick/dream/.
func TestDreamArchivedJobs_ScansDreamLogs(t *testing.T) {
	env := newTestEnv(t)
	rickDir := env.rickDir
	writeDreamLog(t, rickDir, "job_3")
	writeDreamLog(t, rickDir, "job_7")

	got := DreamArchivedJobs(rickDir)
	if !got["job_3"] || !got["job_7"] {
		t.Fatalf("DreamArchivedJobs = %v, want job_3 and job_7", got)
	}
	if got["job_1"] {
		t.Fatal("job_1 reported dream-archived without a dream log")
	}
}

// TestListJobsFiltered_DreamAutoArchive verifies the merged archive view:
// a dream-processed job is hidden from the default listing with
// ArchivedBy=dream in the include view; manual archiving still works and
// marks ArchivedBy=manual; dream wins when both apply.
func TestListJobsFiltered_DreamAutoArchive(t *testing.T) {
	env := newTestEnv(t)
	rickDir := env.rickDir
	writeJobTasks(t, rickDir, "job_1", "success")
	writeJobTasks(t, rickDir, "job_2", "success")
	writeJobTasks(t, rickDir, "job_3", "success")

	// job_2 is dream-processed (auto-archived); job_3 is manually archived.
	writeDreamLog(t, rickDir, "job_2")

	store := &ArchivedStore{data: archivedFile{Version: 1, Archived: map[string][]string{"ws1": {"job_3"}}}}

	// Default view: all three are completed → all auto-archived (job_1 done,
	// job_2 dream, job_3 manual) → empty default view.
	got, err := ListJobsFiltered(rickDir, store.List("ws1"), DreamArchivedJobs(rickDir), false)
	if err != nil {
		t.Fatal(err)
	}
	if ids := jobIDs(got); len(ids) != 0 {
		t.Fatalf("default view = %v, want [] (all completed → auto-archived)", ids)
	}

	// Include view: all three back with archive flags/sources (dream wins
	// over manual wins over done).
	all, err := ListJobsFiltered(rickDir, store.List("ws1"), DreamArchivedJobs(rickDir), true)
	if err != nil {
		t.Fatal(err)
	}
	byID := map[string]JobSummary{}
	for _, j := range all {
		byID[j.JobID] = j
	}
	if !byID["job_2"].Archived || byID["job_2"].ArchivedBy != "dream" {
		t.Fatalf("job_2 include view = %+v, want archived=dream", byID["job_2"])
	}
	if !byID["job_3"].Archived || byID["job_3"].ArchivedBy != "manual" {
		t.Fatalf("job_3 include view = %+v, want archived=manual", byID["job_3"])
	}
	if !byID["job_1"].Archived || byID["job_1"].ArchivedBy != "done" {
		t.Fatalf("job_1 include view = %+v, want archived=done (auto)", byID["job_1"])
	}

	// Dream wins when both apply: job_2 manually archived too → still "dream".
	if err := store.Archive("ws1", "job_2"); err != nil {
		t.Fatal(err)
	}
	both, err := ListJobsFiltered(rickDir, store.List("ws1"), DreamArchivedJobs(rickDir), true)
	if err != nil {
		t.Fatal(err)
	}
	for _, j := range both {
		if j.JobID == "job_2" && j.ArchivedBy != "dream" {
			t.Fatalf("job_2 with both sources: ArchivedBy=%q, want dream", j.ArchivedBy)
		}
	}
}
