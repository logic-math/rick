package handler

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

// core_test.go verifies the job_36 task2 refactor: handler cores are
// path-parameterized (rickDir explicit) instead of cwd-resolved, so web
// sessions can target a workspace different from the process cwd.

func TestNextJobIDIn(t *testing.T) {
	dir := t.TempDir()
	rickDir := filepath.Join(dir, ".rick")

	// Empty workspace (no jobs dir) → job_1
	got, err := nextJobIDIn(rickDir)
	if err != nil {
		t.Fatalf("nextJobIDIn on empty workspace: %v", err)
	}
	if got != "job_1" {
		t.Errorf("empty workspace: want job_1, got %s", got)
	}

	// job_3 + job_10 exist → job_11 (max + 1, not count + 1)
	jobsDir := filepath.Join(rickDir, "jobs")
	for _, id := range []string{"job_3", "job_10"} {
		if err := os.MkdirAll(filepath.Join(jobsDir, id), 0755); err != nil {
			t.Fatal(err)
		}
	}
	got, err = nextJobIDIn(rickDir)
	if err != nil {
		t.Fatalf("nextJobIDIn: %v", err)
	}
	if got != "job_11" {
		t.Errorf("want job_11, got %s", got)
	}

	// Non-directory entries are ignored
	if err := os.WriteFile(filepath.Join(jobsDir, "job_99.md"), []byte("x"), 0644); err != nil {
		t.Fatal(err)
	}
	got, err = nextJobIDIn(rickDir)
	if err != nil {
		t.Fatalf("nextJobIDIn with file entry: %v", err)
	}
	if got != "job_11" {
		t.Errorf("file entries must be ignored: want job_11, got %s", got)
	}
}

func TestNextJobIDIn_IndependentOfWorkspace(t *testing.T) {
	// Two distinct workspaces allocate ids from their own jobs dirs — the
	// cwd-based NextJobID could not do this.
	a := filepath.Join(t.TempDir(), "wsA", ".rick")
	b := filepath.Join(t.TempDir(), "wsB", ".rick")
	for _, rickDir := range []string{a, b} {
		if err := os.MkdirAll(filepath.Join(rickDir, "jobs", "job_5"), 0755); err != nil {
			t.Fatal(err)
		}
	}
	for _, rickDir := range []string{a, b} {
		got, err := nextJobIDIn(rickDir)
		if err != nil {
			t.Fatalf("nextJobIDIn(%s): %v", rickDir, err)
		}
		if got != "job_6" {
			t.Errorf("nextJobIDIn(%s): want job_6, got %s", rickDir, got)
		}
	}
}

func TestEnsureWorkspaceDirsIn(t *testing.T) {
	rickDir := filepath.Join(t.TempDir(), "ws", ".rick")
	if err := ensureWorkspaceDirsIn(rickDir); err != nil {
		t.Fatalf("ensureWorkspaceDirsIn: %v", err)
	}
	// Same six directories as workspace.New()
	for _, sub := range []string{"", "loops", "skills", "domain", "jobs", "dream"} {
		p := filepath.Join(rickDir, sub)
		if info, err := os.Stat(p); err != nil || !info.IsDir() {
			t.Errorf("expected directory %s (err=%v)", p, err)
		}
	}
	// Idempotent
	if err := ensureWorkspaceDirsIn(rickDir); err != nil {
		t.Fatalf("ensureWorkspaceDirsIn second run: %v", err)
	}
}

func TestWorkspaceRoot(t *testing.T) {
	if got := workspaceRoot("/a/b/.rick"); got != "/a/b" {
		t.Errorf("workspaceRoot: want /a/b, got %s", got)
	}
}

func TestDoingIn_ExplicitRickDir(t *testing.T) {
	// doingCore resolves the job directory from the explicit rickDir, not the
	// process cwd: run from an unrelated cwd, target a workspace whose job
	// exists, and observe the doing-dir (not plan-dir) error message.
	dir := t.TempDir()
	rickDir := filepath.Join(dir, ".rick")

	// cwd is an unrelated empty directory with its own .rick (would produce
	// "job directory not found" if doing used cwd).
	other := t.TempDir()
	orig, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(other); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(orig) }()

	// Target workspace has the job dir but no plan dir → plan error proves
	// rickDir (not cwd) resolution.
	if err := os.MkdirAll(filepath.Join(rickDir, "jobs", "job_1"), 0755); err != nil {
		t.Fatal(err)
	}

	err = DoingIn(context.Background(), rickDir, "job_1", Options{}, nil, nil)
	if err == nil {
		t.Fatal("expected error for missing plan dir")
	}
	want := "plan directory not found: " + filepath.Join(rickDir, "jobs", "job_1", "plan")
	if err.Error() != want {
		t.Errorf("error must reference the explicit rickDir:\n want: %s\n got:  %s", want, err.Error())
	}

	// Empty rickDir is rejected
	if err := DoingIn(context.Background(), "", "job_1", Options{}, nil, nil); err == nil {
		t.Error("empty rickDir must be rejected")
	}
}

func TestDoingIn_CancelledBeforeRun(t *testing.T) {
	// A cancelled context surfaces before the first pi attempt.
	dir := t.TempDir()
	rickDir := filepath.Join(dir, ".rick")
	if err := os.MkdirAll(filepath.Join(rickDir, "jobs", "job_1", "plan"), 0755); err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := DoingIn(ctx, rickDir, "job_1", Options{}, nil, nil)
	if err == nil {
		t.Fatal("expected cancellation error")
	}
}

func TestCtrlIn_Validation(t *testing.T) {
	if err := CtrlIn("", "job_1", Options{}); err == nil {
		t.Error("empty rickDir must be rejected")
	}
	dir := t.TempDir()
	rickDir := filepath.Join(dir, ".rick")
	if err := os.MkdirAll(rickDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := CtrlIn(rickDir, "", Options{}); err == nil {
		t.Error("empty jobID must be rejected")
	}
}

func TestPlanIn_Validation(t *testing.T) {
	if err := PlanIn("", "req", Options{}); err == nil {
		t.Error("empty rickDir must be rejected")
	}
}

func TestPrepareHumanLoopDirsIn(t *testing.T) {
	rickDir := filepath.Join(t.TempDir(), ".rick")
	draftDir, rfcDir, loopDir, err := PrepareHumanLoopDirsIn(rickDir)
	if err != nil {
		t.Fatalf("PrepareHumanLoopDirsIn: %v", err)
	}
	if draftDir != filepath.Join(rickDir, "draft") {
		t.Errorf("draftDir: %s", draftDir)
	}
	if rfcDir != filepath.Join(rickDir, "draft", "rfc") {
		t.Errorf("rfcDir: %s", rfcDir)
	}
	if loopDir != filepath.Join(rickDir, "draft", "loops", "loop_1") {
		t.Errorf("first loop should be loop_1, got %s", loopDir)
	}
	// Directory tree ensured
	for _, sub := range []string{"draft", "draft/rfc", "draft/concepts", "draft/human-learning", "draft/loops"} {
		if _, err := os.Stat(filepath.Join(rickDir, sub)); err != nil {
			t.Errorf("expected %s: %v", sub, err)
		}
	}
	// Second allocation without a materialized loop_1 dir returns loop_1
	// again (loop dirs are created by the session flow, e.g. ensureSessionID)
	_, _, loopDir2, err := PrepareHumanLoopDirsIn(rickDir)
	if err != nil {
		t.Fatalf("PrepareHumanLoopDirsIn second: %v", err)
	}
	if filepath.Base(loopDir2) != "loop_1" {
		t.Errorf("unmaterialized loop_1 must reallocate loop_1, got %s", loopDir2)
	}

	// Once loop_1 exists (session flow created it), the next allocation is loop_2
	if err := os.MkdirAll(filepath.Join(rickDir, "draft", "loops", "loop_1"), 0755); err != nil {
		t.Fatal(err)
	}
	_, _, loopDir3, err := PrepareHumanLoopDirsIn(rickDir)
	if err != nil {
		t.Fatalf("PrepareHumanLoopDirsIn third: %v", err)
	}
	if filepath.Base(loopDir3) != "loop_2" {
		t.Errorf("after loop_1 exists, next should be loop_2, got %s", loopDir3)
	}
}

func TestDiffTaskSnapshots(t *testing.T) {
	last := taskSnapshot{
		"task1": {Status: "success", CommitHash: "aaa"},
		"task2": {Status: "running", CommitHash: ""},
	}
	cur := taskSnapshot{
		"task1": {Status: "success", CommitHash: "bbb"}, // commit churn on rerun
		"task2": {Status: "success", CommitHash: "ccc"}, // running → success
		"task3": {Status: "pending", CommitHash: ""},    // new
	}
	changes := diffTaskSnapshots(last, cur)
	if len(changes) != 3 {
		t.Fatalf("want 3 changes, got %d: %+v", len(changes), changes)
	}
	// Sorted by task id
	if changes[0].id != "task1" || changes[0].from != "success" || changes[0].to != "success" || changes[0].hash != "bbb" {
		t.Errorf("change[0] = %+v", changes[0])
	}
	if changes[1].id != "task2" || changes[1].from != "running" || changes[1].to != "success" {
		t.Errorf("change[1] = %+v", changes[1])
	}
	if changes[2].id != "task3" || changes[2].from != "new" || changes[2].to != "pending" {
		t.Errorf("change[2] = %+v", changes[2])
	}

	// No changes → empty
	if got := diffTaskSnapshots(cur, cur); len(got) != 0 {
		t.Errorf("identical snapshots must diff empty, got %+v", got)
	}
}

func TestEmitDoingAndDream(t *testing.T) {
	// nil callbacks are safe no-ops
	emitDoing(nil, DoingEvent{JobID: "job_1", TaskID: "task1", From: "running", To: "success"})
	emitDream(nil, DreamEvent{Phase: "selected"})

	var got []DoingEvent
	emitDoing(func(ev DoingEvent) { got = append(got, ev) }, DoingEvent{TaskID: "task1"})
	if len(got) != 1 || got[0].TaskID != "task1" {
		t.Errorf("emitDoing callback: %+v", got)
	}
}

func TestDreamIn_SelectedEventAndEmpty(t *testing.T) {
	// No pending jobs → returns nil, no selected event (no dream dir state
	// needed beyond .rick skeleton).
	dir := t.TempDir()
	rickDir := filepath.Join(dir, ".rick")
	if err := os.MkdirAll(filepath.Join(rickDir, "jobs"), 0755); err != nil {
		t.Fatal(err)
	}

	var events []DreamEvent
	err := DreamIn(context.Background(), rickDir, 5, false, Options{}, func(ev DreamEvent) {
		events = append(events, ev)
	})
	if err != nil {
		t.Fatalf("DreamIn with no pending jobs: %v", err)
	}
	if len(events) != 0 {
		t.Errorf("no pending jobs must emit no events, got %+v", events)
	}

	// Empty rickDir rejected
	if err := DreamIn(context.Background(), "", 5, false, Options{}, nil); err == nil {
		t.Error("empty rickDir must be rejected")
	}
}
