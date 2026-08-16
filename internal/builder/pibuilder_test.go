package builder

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestBuildPlan_EmptyRequirement(t *testing.T) {
	pb := NewPIBuilder()
	_, _, err := pb.BuildPlan("", nil)
	if err == nil {
		t.Fatal("expected error for empty requirement")
	}
	if !strings.Contains(err.Error(), "requirement cannot be empty") {
		t.Fatalf("expected error to contain 'requirement cannot be empty', got %v", err)
	}
}

func TestBuildPlan_MethodAndInstance(t *testing.T) {
	pb := NewPIBuilder()
	method, instance, err := pb.BuildPlan("implement login", map[string]string{
		"rick_dir":     t.TempDir(),
		"job_plan_dir": filepath.Join(t.TempDir(), "jobs", "job_1", "plan"),
	})
	if err != nil {
		t.Fatalf("BuildPlan failed: %v", err)
	}
	if method == "" {
		t.Error("expected non-empty method")
	}
	if instance == "" {
		t.Error("expected non-empty instance")
	}
	if !strings.Contains(instance, "implement login") {
		t.Error("instance should contain requirement")
	}
	if strings.Contains(instance, "{{") {
		t.Error("instance should have no unreplaced variables")
	}
}

func TestBuildDoing_PathInjection(t *testing.T) {
	pb := NewPIBuilder()
	rickDir := t.TempDir()
	doingDir := filepath.Join(rickDir, "jobs", "job_1", "doing")
	planDir := filepath.Join(rickDir, "jobs", "job_1", "plan")

	_, instance, err := pb.BuildDoing("task1", map[string]string{
		"rick_dir":  rickDir,
		"plan_dir":  planDir,
		"doing_dir": doingDir,
		"job_id":    "job_1",
	})
	if err != nil {
		t.Fatalf("BuildDoing failed: %v", err)
	}

	// task_info_section 与 debug_context 变量值应为路径片段，而非正文。
	if !strings.Contains(instance, filepath.Join(planDir, "task1.md")) {
		t.Error("instance should inject task_info_section as plan/task1.md path")
	}
	if !strings.Contains(instance, filepath.Join(doingDir, "debug")) {
		t.Error("instance should inject debug_context as doing/debug path")
	}
	if strings.Contains(instance, "{{") {
		t.Error("instance should have no unreplaced variables")
	}
}

func TestBuildDoing_EmptyTaskIDFallsBack(t *testing.T) {
	pb := NewPIBuilder()
	_, instance, err := pb.BuildDoing("", map[string]string{
		"rick_dir": t.TempDir(),
		"plan_dir": filepath.Join(t.TempDir(), "plan"),
	})
	if err != nil {
		t.Fatalf("BuildDoing failed: %v", err)
	}
	if !strings.Contains(instance, "taskN.md") {
		t.Error("expected taskN.md fallback path in instance")
	}
}

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
	instance, err := pb.BuildPrompt("plan", map[string]string{"requirement": "req"})
	if err != nil {
		t.Fatalf("BuildPrompt plan failed: %v", err)
	}
	if !strings.Contains(instance, "req") {
		t.Error("BuildPrompt plan should include requirement")
	}

	if _, err := pb.BuildPrompt("unknown", nil); err == nil {
		t.Error("expected error for unknown cmd")
	}
}

func TestBuildPlan_InlinesGrillingContent(t *testing.T) {
	pb := NewPIBuilder()
	_, instance, err := pb.BuildPlan("implement login", map[string]string{
		"rick_dir":     t.TempDir(),
		"job_plan_dir": filepath.Join(t.TempDir(), "jobs", "job_1", "plan"),
	})
	if err != nil {
		t.Fatalf("BuildPlan failed: %v", err)
	}
	// 单文件内聚：grilling skill 内容内联进主产物（而非仅 skill_grilling.md 路径引用）。
	if !strings.Contains(instance, "Grilling") {
		t.Error("instance should inline grilling content (Grilling)")
	}
	if !strings.Contains(instance, "结构化追问") {
		t.Error("instance should inline grilling content (结构化追问)")
	}
	if !strings.Contains(instance, "内联技能") {
		t.Error("instance should have a structured inline-skills section")
	}
	if strings.Contains(instance, "{{") {
		t.Error("instance should have no unreplaced variables")
	}
}

func TestBuildDoing_InlinesDoingLoopContent(t *testing.T) {
	pb := NewPIBuilder()
	rickDir := t.TempDir()
	_, instance, err := pb.BuildDoing("task1", map[string]string{
		"rick_dir":  rickDir,
		"plan_dir":  filepath.Join(rickDir, "jobs", "job_1", "plan"),
		"doing_dir": filepath.Join(rickDir, "jobs", "job_1", "doing"),
		"job_id":    "job_1",
	})
	if err != nil {
		t.Fatalf("BuildDoing failed: %v", err)
	}
	// doing_loop 内容内联（非路径引用），含 Step 0 domain 搜索等正文。
	if !strings.Contains(instance, "Step 0") || !strings.Contains(instance, "Step 1") {
		t.Error("instance should inline doing_loop content (Step 0/Step 1)")
	}
	if strings.Contains(instance, "{{") {
		t.Error("instance should have no unreplaced variables")
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
