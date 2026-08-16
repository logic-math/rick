package workspace

import (
	"os"
	"path/filepath"
	"testing"
)

func writeTasksJSONFile(t *testing.T, path string, tasks string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(tasks), 0644); err != nil {
		t.Fatal(err)
	}
}

func TestJobNumber(t *testing.T) {
	cases := []struct {
		in   string
		want int
	}{
		{"job_1", 1},
		{"job_10", 10},
		{"job_42", 42},
		{"job_abc", 0},
		{"", 0},
	}
	for _, c := range cases {
		if got := JobNumber(c.in); got != c.want {
			t.Errorf("JobNumber(%q) = %d, want %d", c.in, got, c.want)
		}
	}
}

func TestDiscoverCompletedJobs(t *testing.T) {
	rickDir := t.TempDir()

	// job_1: all success → completed
	writeTasksJSONFile(t, filepath.Join(rickDir, "jobs", "job_1", "doing", "tasks.json"),
		`{"version":"1.0","tasks":[{"task_id":"t1","status":"success"}]}`)
	// job_2: one pending → excluded
	writeTasksJSONFile(t, filepath.Join(rickDir, "jobs", "job_2", "doing", "tasks.json"),
		`{"version":"1.0","tasks":[{"task_id":"t1","status":"success"},{"task_id":"t2","status":"pending"}]}`)
	// job_10: all success → completed (tests numeric sort vs lexicographic)
	writeTasksJSONFile(t, filepath.Join(rickDir, "jobs", "job_10", "doing", "tasks.json"),
		`{"version":"1.0","tasks":[{"task_id":"t1","status":"success"}]}`)
	// job_3: empty tasks → excluded
	writeTasksJSONFile(t, filepath.Join(rickDir, "jobs", "job_3", "doing", "tasks.json"),
		`{"version":"1.0","tasks":[]}`)

	got := DiscoverCompletedJobs(rickDir)
	want := []string{"job_1", "job_10"}
	if len(got) != len(want) {
		t.Fatalf("DiscoverCompletedJobs = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("DiscoverCompletedJobs[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestGetDreamProcessedJobs(t *testing.T) {
	rickDir := t.TempDir()
	dreamDir := filepath.Join(rickDir, "dream")
	if err := os.MkdirAll(dreamDir, 0755); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"dream_run_job_1_log.md", "dream_run_job_2_log.md", "other.md"} {
		if err := os.WriteFile(filepath.Join(dreamDir, name), []byte("x"), 0644); err != nil {
			t.Fatal(err)
		}
	}

	got := GetDreamProcessedJobs(rickDir)
	if !got["job_1"] || !got["job_2"] {
		t.Errorf("expected job_1 and job_2 to be processed, got %v", got)
	}
	if got["other"] {
		t.Errorf("other.md should not be treated as a processed job, got %v", got)
	}
}

func TestSelectPendingJobs(t *testing.T) {
	rickDir := t.TempDir()

	// job_1: completed, not dreamed → pending
	writeTasksJSONFile(t, filepath.Join(rickDir, "jobs", "job_1", "doing", "tasks.json"),
		`{"version":"1.0","tasks":[{"task_id":"t1","status":"success"}]}`)
	// job_2: completed but dreamed → excluded
	writeTasksJSONFile(t, filepath.Join(rickDir, "jobs", "job_2", "doing", "tasks.json"),
		`{"version":"1.0","tasks":[{"task_id":"t1","status":"success"}]}`)
	// job_3: incomplete → excluded
	writeTasksJSONFile(t, filepath.Join(rickDir, "jobs", "job_3", "doing", "tasks.json"),
		`{"version":"1.0","tasks":[{"task_id":"t1","status":"pending"}]}`)
	// job_4: completed, not dreamed → pending
	writeTasksJSONFile(t, filepath.Join(rickDir, "jobs", "job_4", "doing", "tasks.json"),
		`{"version":"1.0","tasks":[{"task_id":"t1","status":"success"}]}`)

	dreamDir := filepath.Join(rickDir, "dream")
	if err := os.MkdirAll(dreamDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dreamDir, "dream_run_job_2_log.md"), []byte("x"), 0644); err != nil {
		t.Fatal(err)
	}

	// Sorted ascending, truncates to jobNum.
	got := SelectPendingJobs(rickDir, 5)
	want := []string{"job_1", "job_4"}
	if len(got) != len(want) {
		t.Fatalf("SelectPendingJobs = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("SelectPendingJobs[%d] = %q, want %q", i, got[i], want[i])
		}
	}

	// Truncation: jobNum=1 returns only the first pending job.
	got = SelectPendingJobs(rickDir, 1)
	if len(got) != 1 || got[0] != "job_1" {
		t.Errorf("SelectPendingJobs(rickDir, 1) = %v, want [job_1]", got)
	}
}
