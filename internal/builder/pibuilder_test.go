package builder

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestBuildAgents_Empty(t *testing.T) {
	pb := NewPIBuilder()
	agents, err := pb.BuildAgents(nil)
	if err != nil {
		t.Fatalf("BuildAgents failed: %v", err)
	}
	if len(agents) != 0 {
		t.Errorf("expected empty agents for pi, got %d", len(agents))
	}
}

func TestBuildPrompt_Dispatch(t *testing.T) {
	pb := NewPIBuilder()
	instance, err := pb.BuildPrompt("easy", map[string]string{"requirement": "req"})
	if err != nil {
		t.Fatalf("BuildPrompt easy failed: %v", err)
	}
	if !strings.Contains(instance, "req") {
		t.Error("BuildPrompt easy should include requirement")
	}

	// 已删除类型（job_53 Bug #5）与未知命令同样报错
	for _, gone := range []string{"plan", "doing", "learning", "ctrl"} {
		if _, err := pb.BuildPrompt(gone, nil); err == nil {
			t.Errorf("removed cmd %q must error", gone)
		}
	}
	if _, err := pb.BuildPrompt("unknown", nil); err == nil {
		t.Error("expected error for unknown cmd")
	}
}

func TestBuildEasy_InlinesGrillingContent(t *testing.T) {
	pb := NewPIBuilder()
	rickDir := t.TempDir()
	_, instance, err := pb.BuildEasy("测试需求", map[string]string{
		"rick_dir":  rickDir,
		"doing_dir": filepath.Join(rickDir, "jobs", "job_N", "doing"),
		"job_id":    "job_N",
	})
	if err != nil {
		t.Fatalf("BuildEasy failed: %v", err)
	}
	if !strings.Contains(instance, "Grilling") {
		t.Error("instance should inline grilling content (Grilling)")
	}
	if strings.Contains(instance, "{{") {
		t.Error("instance should have no unreplaced variables")
	}
}
