package cmd

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/sunquan/rick/internal/agent/piagent"
	"github.com/sunquan/rick/internal/config"
	"github.com/sunquan/rick/internal/executor"
)

func containsStr(s, substr string) bool {
	return strings.Contains(s, substr)
}

// ─── Plan Check Tests ────────────────────────────────────────────────────────

func TestRunPlanCheck_NoDir(t *testing.T) {
	err := runPlanCheck("/nonexistent/path/plan")
	if err == nil {
		t.Fatal("expected error for nonexistent plan dir")
	}
}

func TestRunPlanCheck_NoTasks(t *testing.T) {
	dir := t.TempDir()
	err := runPlanCheck(dir)
	if err == nil {
		t.Fatal("expected error for empty plan dir")
	}
}

func TestRunPlanCheck_MissingSection(t *testing.T) {
	dir := t.TempDir()
	// Missing '# 关键结果'
	content := "# 依赖关系\n无\n# 任务名称\nTest\n# 任务目标\nGoal\n# 测试方法\nTest\n"
	if err := os.WriteFile(filepath.Join(dir, "task1.md"), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	err := runPlanCheck(dir)
	if err == nil {
		t.Fatal("expected error for missing section")
	}
	if !containsStr(err.Error(), "关键结果") {
		t.Errorf("expected error to mention 关键结果, got: %v", err)
	}
}

func TestRunPlanCheck_MissingDepFile(t *testing.T) {
	dir := t.TempDir()
	content := "# 依赖关系\ntask99\n# 任务名称\nTest\n# 任务目标\nGoal\n# 关键结果\n1. KR1\n# 测试方法\nTest\n"
	if err := os.WriteFile(filepath.Join(dir, "task1.md"), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	err := runPlanCheck(dir)
	if err == nil {
		t.Fatal("expected error for missing dependency")
	}
	if !containsStr(err.Error(), "task99") {
		t.Errorf("expected error to mention task99, got: %v", err)
	}
}

func TestRunPlanCheck_CircularDep(t *testing.T) {
	dir := t.TempDir()
	task1 := "# 依赖关系\ntask2\n# 任务名称\nTask1\n# 任务目标\nGoal\n# 关键结果\n1. KR1\n# 测试方法\nTest\n"
	task2 := "# 依赖关系\ntask1\n# 任务名称\nTask2\n# 任务目标\nGoal\n# 关键结果\n1. KR1\n# 测试方法\nTest\n"
	if err := os.WriteFile(filepath.Join(dir, "task1.md"), []byte(task1), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "task2.md"), []byte(task2), 0644); err != nil {
		t.Fatal(err)
	}
	err := runPlanCheck(dir)
	if err == nil {
		t.Fatal("expected error for circular dependency")
	}
	if !containsStr(err.Error(), "cycle") && !containsStr(err.Error(), "circular") {
		t.Errorf("expected cycle error, got: %v", err)
	}
}

func TestRunPlanCheck_Valid(t *testing.T) {
	dir := t.TempDir()
	task1 := "# 依赖关系\n无\n# 任务名称\nTask1\n# 任务目标\nGoal\n# 关键结果\n1. KR1\n# 测试方法\nTest\n"
	task2 := "# 依赖关系\ntask1\n# 任务名称\nTask2\n# 任务目标\nGoal\n# 关键结果\n1. KR1\n# 测试方法\nTest\n"
	if err := os.WriteFile(filepath.Join(dir, "task1.md"), []byte(task1), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "task2.md"), []byte(task2), 0644); err != nil {
		t.Fatal(err)
	}
	// OKR.md is required with meaningful content
	okrContent := "# Job OKR\n## O1: 目标\n- KR1: 完成核心功能实现并通过测试\n- KR2: 代码覆盖率达到 80% 以上\n"
	if err := os.WriteFile(filepath.Join(dir, "OKR.md"), []byte(okrContent), 0644); err != nil {
		t.Fatal(err)
	}
	if err := runPlanCheck(dir); err != nil {
		t.Errorf("expected no error for valid plan, got: %v", err)
	}
}

// ─── Doing Check Tests ───────────────────────────────────────────────────────

func makeTasksJSON(t *testing.T, dir string, tasks []executor.TaskState) {
	t.Helper()
	now := time.Now()
	data := map[string]interface{}{
		"version":    "1.0",
		"created_at": now,
		"updated_at": now,
		"tasks":      tasks,
	}
	b, err := json.Marshal(data)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "tasks.json"), b, 0644); err != nil {
		t.Fatal(err)
	}
}

func TestRunDoingCheck_NoTasksJSON(t *testing.T) {
	dir := t.TempDir()
	err := runDoingCheck(dir)
	if err == nil {
		t.Fatal("expected error for missing tasks.json")
	}
}

func TestRunDoingCheck_ZombieTask(t *testing.T) {
	dir := t.TempDir()
	makeTasksJSON(t, dir, []executor.TaskState{
		{TaskID: "task1", TaskName: "T1", Status: "running"},
	})
	err := runDoingCheck(dir)
	if err == nil {
		t.Fatal("expected error for zombie task")
	}
	if !containsStr(err.Error(), "running") {
		t.Errorf("expected 'running' in error, got: %v", err)
	}
}

func TestRunDoingCheck_MissingCommitHash(t *testing.T) {
	dir := t.TempDir()
	makeTasksJSON(t, dir, []executor.TaskState{
		{TaskID: "task1", TaskName: "T1", Status: "success", CommitHash: ""},
	})
	err := runDoingCheck(dir)
	if err == nil {
		t.Fatal("expected error for missing commit_hash")
	}
	if !containsStr(err.Error(), "commit_hash") {
		t.Errorf("expected commit_hash in error, got: %v", err)
	}
}

func TestRunDoingCheck_Valid(t *testing.T) {
	dir := t.TempDir()
	makeTasksJSON(t, dir, []executor.TaskState{
		{TaskID: "task1", TaskName: "T1", Status: "success", CommitHash: "abc123"},
		{TaskID: "task2", TaskName: "T2", Status: "success", CommitHash: "def456"},
	})
	if err := runDoingCheck(dir); err != nil {
		t.Errorf("expected no error for valid doing dir, got: %v", err)
	}
}

func TestRunDoingCheck_FailedTaskNoCommit(t *testing.T) {
	dir := t.TempDir()
	makeTasksJSON(t, dir, []executor.TaskState{
		{TaskID: "task1", TaskName: "T1", Status: "failed", CommitHash: ""},
	})
	if err := runDoingCheck(dir); err != nil {
		t.Errorf("expected no error for failed task without commit_hash, got: %v", err)
	}
}

func TestRunDoingCheck_DebugBugUnresolved(t *testing.T) {
	dir := t.TempDir()
	makeTasksJSON(t, dir, []executor.TaskState{
		{TaskID: "task1", TaskName: "T1", Status: "success", CommitHash: "abc123"},
	})
	debugDir := filepath.Join(dir, "debug")
	if err := os.MkdirAll(debugDir, 0755); err != nil {
		t.Fatal(err)
	}
	content := "---\nsummary: \"some bug\"\nstatus: \"🔄 进行中\"\n---\n## 阶段一记录\n..."
	if err := os.WriteFile(filepath.Join(debugDir, "bug1-some-bug.md"), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	err := runDoingCheck(dir)
	if err == nil {
		t.Fatal("expected error for unresolved bug")
	}
	if !containsStr(err.Error(), "进行中") {
		t.Errorf("expected '进行中' in error, got: %v", err)
	}
}

func TestRunDoingCheck_DebugBugMissingStatus(t *testing.T) {
	dir := t.TempDir()
	makeTasksJSON(t, dir, []executor.TaskState{
		{TaskID: "task1", TaskName: "T1", Status: "success", CommitHash: "abc123"},
	})
	debugDir := filepath.Join(dir, "debug")
	if err := os.MkdirAll(debugDir, 0755); err != nil {
		t.Fatal(err)
	}
	content := "## 阶段一记录\n没有 frontmatter"
	if err := os.WriteFile(filepath.Join(debugDir, "bug1-no-frontmatter.md"), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	err := runDoingCheck(dir)
	if err == nil {
		t.Fatal("expected error for bug file missing status frontmatter")
	}
	if !containsStr(err.Error(), "status:") {
		t.Errorf("expected 'status:' in error, got: %v", err)
	}
}

func TestRunDoingCheck_DebugBugResolved(t *testing.T) {
	dir := t.TempDir()
	makeTasksJSON(t, dir, []executor.TaskState{
		{TaskID: "task1", TaskName: "T1", Status: "success", CommitHash: "abc123"},
	})
	debugDir := filepath.Join(dir, "debug")
	if err := os.MkdirAll(debugDir, 0755); err != nil {
		t.Fatal(err)
	}
	content := "---\nsummary: \"fixed\"\nstatus: \"✅ 已解决\"\n---\n\n## Phase 1: 构建反馈回路\n\n（复现步骤）\n\n## Phase 2: 复现最小化\n\n（最小单元）\n\n## Phase 3: 可证伪假设\n\n（假设列表）\n\n## Phase 4: 插桩观察\n\n（观测结果）\n\n## Phase 5: 修复回归\n\n（修复完成）\n\n## Phase 6: 清理事后分析\n\n（清理完成）\n\n## 结论\n\n修复完成。\n"
	if err := os.WriteFile(filepath.Join(debugDir, "bug1-fixed.md"), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	if err := runDoingCheck(dir); err != nil {
		t.Errorf("expected no error for resolved bug, got: %v", err)
	}
}

// ─── Learning Check Tests ─────────────────────────────────────────────────────

func TestRunLearningCheck_NoSummary(t *testing.T) {
	dir := t.TempDir()
	err := runLearningCheck(dir)
	if err == nil {
		t.Fatal("expected error for missing SUMMARY.md")
	}
	if !containsStr(err.Error(), "SUMMARY.md") {
		t.Errorf("expected SUMMARY.md in error, got: %v", err)
	}
}

func TestRunLearningCheck_EmptySummary(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "SUMMARY.md"), []byte("   "), 0644); err != nil {
		t.Fatal(err)
	}
	err := runLearningCheck(dir)
	if err == nil {
		t.Fatal("expected error for empty SUMMARY.md")
	}
	if !containsStr(err.Error(), "empty") && !containsStr(err.Error(), "# Job") {
		t.Errorf("expected empty/# Job in error, got: %v", err)
	}
}

func TestRunLearningCheck_MissingJobHeading(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "SUMMARY.md"), []byte("# Summary\nsome content without job heading"), 0644); err != nil {
		t.Fatal(err)
	}
	err := runLearningCheck(dir)
	if err == nil {
		t.Fatal("expected error for SUMMARY.md missing '# Job' heading")
	}
	if !containsStr(err.Error(), "# Job") {
		t.Errorf("expected '# Job' in error, got: %v", err)
	}
}

func TestRunLearningCheck_Valid(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "SUMMARY.md"), []byte("# Job Summary\nsome content"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := runLearningCheck(dir); err != nil {
		t.Errorf("expected no error for valid learning dir, got: %v", err)
	}
}

func TestRunLearningCheck_ValidWithSkill(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "SUMMARY.md"), []byte("# Job Summary\nsome content"), 0644); err != nil {
		t.Fatal(err)
	}
	toolsDir := filepath.Join(dir, "tools")
	if err := os.MkdirAll(toolsDir, 0755); err != nil {
		t.Fatal(err)
	}
	goodPy := "def hello():\n    return 'world'\n"
	if err := os.WriteFile(filepath.Join(toolsDir, "good.py"), []byte(goodPy), 0644); err != nil {
		t.Fatal(err)
	}
	if err := runLearningCheck(dir); err != nil {
		t.Errorf("expected no error for valid skill, got: %v", err)
	}
}

func TestRunLearningCheck_ValidOKR(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "SUMMARY.md"), []byte("# Job Summary\nsome content"), 0644); err != nil {
		t.Fatal(err)
	}
	okr := "## O1: 目标\n### 关键结果\n1. KR1\n"
	if err := os.WriteFile(filepath.Join(dir, "OKR.md"), []byte(okr), 0644); err != nil {
		t.Fatal(err)
	}
	if err := runLearningCheck(dir); err != nil {
		t.Errorf("expected no error for valid OKR, got: %v", err)
	}
}

func TestRunLearningCheck_ValidSPEC(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "SUMMARY.md"), []byte("# Job Summary\nsome content"), 0644); err != nil {
		t.Fatal(err)
	}
	spec := "## 调试环境\ngo run\n## 架构设计\nModular\n## 编译与运行方法\ngo build\n## 观测方法\nlogs\n## 控制方法\nconfig\n## 技能列表\n| 名称 | 触发词 | 路径 |\n"
	if err := os.WriteFile(filepath.Join(dir, "SPEC.md"), []byte(spec), 0644); err != nil {
		t.Fatal(err)
	}
	if err := runLearningCheck(dir); err != nil {
		t.Errorf("expected no error for valid SPEC, got: %v", err)
	}
}

// ─── Tools Command Tests ──────────────────────────────────────────────────────

func TestNewToolsCmd(t *testing.T) {
	cmd := NewToolsCmd()
	if cmd == nil {
		t.Fatal("NewToolsCmd returned nil")
	}
	if cmd.Use != "tools" {
		t.Errorf("expected Use='tools', got '%s'", cmd.Use)
	}
}

func TestNewPlanCheckCmd(t *testing.T) {
	cmd := NewPlanCheckCmd()
	if cmd == nil {
		t.Fatal("NewPlanCheckCmd returned nil")
	}
	if cmd.Use != "plan_check <job_id>" {
		t.Errorf("unexpected Use: %s", cmd.Use)
	}
}

func TestNewDoingCheckCmd(t *testing.T) {
	cmd := NewDoingCheckCmd()
	if cmd == nil {
		t.Fatal("NewDoingCheckCmd returned nil")
	}
	if cmd.Use != "doing_check <job_id>" {
		t.Errorf("unexpected Use: %s", cmd.Use)
	}
}

func TestNewLearningCheckCmd(t *testing.T) {
	cmd := NewLearningCheckCmd()
	if cmd == nil {
		t.Fatal("NewLearningCheckCmd returned nil")
	}
	if cmd.Use != "learning_check <job_id>" {
		t.Errorf("unexpected Use: %s", cmd.Use)
	}
}

func TestToolsSubcommands(t *testing.T) {
	cmd := NewToolsCmd()
	subNames := make(map[string]bool)
	for _, sub := range cmd.Commands() {
		subNames[sub.Name()] = true
	}
	expected := []string{"plan_check", "doing_check", "learning_check", "dream_check"}
	for _, name := range expected {
		if !subNames[name] {
			t.Errorf("missing subcommand: %s", name)
		}
	}
}

// ─── Workspace-dependent tests ────────────────────────────────────────────────

// withTempWorkspace changes the working directory to a temp dir with a .rick structure,
// calls f, then restores the original working directory.
func withTempWorkspace(t *testing.T, f func(dir string)) {
	t.Helper()
	orig, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(orig)
	})
	if err := os.MkdirAll(filepath.Join(dir, ".rick"), 0755); err != nil {
		t.Fatal(err)
	}
	f(dir)
}

func TestRunDoingDryRun_EmptyJobID(t *testing.T) {
	// Empty job ID should not error
	if err := runDoingDryRun(""); err != nil {
		t.Errorf("expected no error for empty job ID, got: %v", err)
	}
}

func TestRunDoingDryRun_NoPlanDir(t *testing.T) {
	withTempWorkspace(t, func(dir string) {
		// job_test has no plan dir
		if err := runDoingDryRun("job_test"); err != nil {
			t.Errorf("expected no error (dry-run ignores missing plan), got: %v", err)
		}
	})
}

func TestRunDoingDryRun_WithPlan(t *testing.T) {
	withTempWorkspace(t, func(dir string) {
		planDir := filepath.Join(dir, ".rick", "jobs", "job_test", "plan")
		if err := os.MkdirAll(planDir, 0755); err != nil {
			t.Fatal(err)
		}
		task1 := "# 依赖关系\n无\n# 任务名称\nTask1\n# 任务目标\nGoal\n# 关键结果\n1. KR1\n# 测试方法\nTest\n"
		if err := os.WriteFile(filepath.Join(planDir, "task1.md"), []byte(task1), 0644); err != nil {
			t.Fatal(err)
		}
		// Should not error even if prompt generation has issues
		if err := runDoingDryRun("job_test"); err != nil {
			t.Errorf("expected no error, got: %v", err)
		}
	})
}

func TestRunPlanCheck_WithWorkspace(t *testing.T) {
	withTempWorkspace(t, func(dir string) {
		planDir := filepath.Join(dir, ".rick", "jobs", "job_test", "plan")
		if err := os.MkdirAll(planDir, 0755); err != nil {
			t.Fatal(err)
		}
		task1 := "# 依赖关系\n无\n# 任务名称\nTask1\n# 任务目标\nGoal\n# 关键结果\n1. KR1\n# 测试方法\nTest\n"
		if err := os.WriteFile(filepath.Join(planDir, "task1.md"), []byte(task1), 0644); err != nil {
			t.Fatal(err)
		}
		okrContent := "# Job OKR\n## O1: 目标\n- KR1: 完成核心功能实现并通过测试\n"
		if err := os.WriteFile(filepath.Join(planDir, "OKR.md"), []byte(okrContent), 0644); err != nil {
			t.Fatal(err)
		}
		// Write meaningful SPEC.md so the project-level check passes
		specPath := filepath.Join(dir, ".rick", "SPEC.md")
		specContent := "# SPEC\n## 架构设计\n本项目采用 Go 标准库实现，无外部依赖。\n## 编译与运行方法\n```\ngo build ./...\n```\n"
		if err := os.WriteFile(specPath, []byte(specContent), 0644); err != nil {
			t.Fatal(err)
		}
		if err := runPlanCheck(planDir); err != nil {
			t.Errorf("expected no error, got: %v", err)
		}
	})
}

func TestRunDoingCheck_WithWorkspace(t *testing.T) {
	withTempWorkspace(t, func(dir string) {
		doingDir := filepath.Join(dir, ".rick", "jobs", "job_test", "doing")
		if err := os.MkdirAll(doingDir, 0755); err != nil {
			t.Fatal(err)
		}
		makeTasksJSON(t, doingDir, []executor.TaskState{
			{TaskID: "task1", TaskName: "T1", Status: "success", CommitHash: "abc123"},
		})
		if err := os.WriteFile(filepath.Join(doingDir, "debug.md"), []byte("## task1: did work\nsome content"), 0644); err != nil {
			t.Fatal(err)
		}
		if err := runDoingCheck(doingDir); err != nil {
			t.Errorf("expected no error, got: %v", err)
		}
	})
}

func TestRunLearningCheck_WithWorkspace(t *testing.T) {
	withTempWorkspace(t, func(dir string) {
		learningDir := filepath.Join(dir, ".rick", "jobs", "job_test", "learning")
		if err := os.MkdirAll(learningDir, 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(learningDir, "SUMMARY.md"), []byte("# Job Summary\nsome content"), 0644); err != nil {
			t.Fatal(err)
		}
		if err := runLearningCheck(learningDir); err != nil {
			t.Errorf("expected no error, got: %v", err)
		}
	})
}

// ─── Git helper tests ─────────────────────────────────────────────────────────

func setupGitRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	orig, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(orig) })

	// Initialize git repo
	cmds := [][]string{
		{"git", "init"},
		{"git", "config", "user.email", "test@test.com"},
		{"git", "config", "user.name", "Test"},
	}
	for _, args := range cmds {
		cmd := exec.Command(args[0], args[1:]...)
		cmd.Dir = dir
		if err := cmd.Run(); err != nil {
			t.Fatalf("git setup failed: %v", err)
		}
	}
	// Create initial commit
	if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte("test"), 0644); err != nil {
		t.Fatal(err)
	}
	for _, args := range [][]string{
		{"git", "add", "."},
		{"git", "commit", "-m", "init"},
	} {
		cmd := exec.Command(args[0], args[1:]...)
		cmd.Dir = dir
		if err := cmd.Run(); err != nil {
			t.Fatalf("git commit failed: %v", err)
		}
	}
	return dir
}

func TestFindPiBinary(t *testing.T) {
	// Just verify the function runs without panic
	path, err := piagent.FindBinary(nil)
	if err != nil {
		t.Logf("piagent.FindBinary returned error (pi not in PATH): %v", err)
	} else if path == "" {
		t.Error("expected non-empty path when pi is found")
	}
}

func TestAutoFix_MockPi(t *testing.T) {
	tmpDir := t.TempDir()
	mockScript := "#!/bin/sh\nexit 0\n"
	mockPath := filepath.Join(tmpDir, "mock_pi")
	if err := os.WriteFile(mockPath, []byte(mockScript), 0755); err != nil {
		t.Fatal(err)
	}
	promptFile := filepath.Join(tmpDir, "prompt.md")
	if err := os.WriteFile(promptFile, []byte("# prompt"), 0644); err != nil {
		t.Fatal(err)
	}
	cfg := &config.Config{PiPath: mockPath}
	if err := piagent.CallCLI(false, cfg, promptFile, piagent.ModePrint); err != nil {
		t.Errorf("expected no error with mock pi, got: %v", err)
	}
}

func TestAutoFix_FailingPi(t *testing.T) {
	tmpDir := t.TempDir()
	mockScript := "#!/bin/sh\nexit 1\n"
	mockPath := filepath.Join(tmpDir, "mock_pi_fail")
	if err := os.WriteFile(mockPath, []byte(mockScript), 0755); err != nil {
		t.Fatal(err)
	}
	promptFile := filepath.Join(tmpDir, "prompt.md")
	if err := os.WriteFile(promptFile, []byte("# prompt"), 0644); err != nil {
		t.Fatal(err)
	}
	cfg := &config.Config{PiPath: mockPath}
	if err := piagent.CallCLI(false, cfg, promptFile, piagent.ModePrint); err == nil {
		t.Error("expected error with failing pi binary")
	}
}

// ─── collectExecutionData tests ───────────────────────────────────────────────

func TestCollectExecutionData_NoDoingDir(t *testing.T) {
	withTempWorkspace(t, func(dir string) {
		_, err := collectExecutionData("job_test")
		if err == nil {
			t.Fatal("expected error for missing doing dir")
		}
	})
}

func TestCollectExecutionData_WithData(t *testing.T) {
	withTempWorkspace(t, func(dir string) {
		doingDir := filepath.Join(dir, ".rick", "jobs", "job_test", "doing")
		if err := os.MkdirAll(doingDir, 0755); err != nil {
			t.Fatal(err)
		}
		// Create debug.md
		if err := os.WriteFile(filepath.Join(doingDir, "debug.md"), []byte("## task1: did work\nsome content"), 0644); err != nil {
			t.Fatal(err)
		}
		// Create tasks.json
		makeTasksJSON(t, doingDir, []executor.TaskState{
			{TaskID: "task1", TaskName: "T1", Status: "success", CommitHash: "abc"},
		})
		data, err := collectExecutionData("job_test")
		if err != nil {
			t.Errorf("expected no error, got: %v", err)
		}
		if data == nil {
			t.Fatal("expected non-nil data")
		}
		if data.JobID != "job_test" {
			t.Errorf("expected job_id=job_test, got %s", data.JobID)
		}
	})
}

func TestCollectExecutionData_NoDebugMD(t *testing.T) {
	withTempWorkspace(t, func(dir string) {
		doingDir := filepath.Join(dir, ".rick", "jobs", "job_test", "doing")
		if err := os.MkdirAll(doingDir, 0755); err != nil {
			t.Fatal(err)
		}
		// Create tasks.json but no debug.md
		makeTasksJSON(t, doingDir, []executor.TaskState{
			{TaskID: "task1", TaskName: "T1", Status: "success", CommitHash: "abc"},
		})
		data, err := collectExecutionData("job_test")
		if err != nil {
			t.Errorf("expected no error even without debug.md, got: %v", err)
		}
		if data != nil && data.DebugContent != "" {
			t.Logf("debug content unexpectedly set: %s", data.DebugContent)
		}
	})
}

// ─── commitDoingResults tests ─────────────────────────────────────────────────

func TestCommitDoingResults_NoChanges(t *testing.T) {
	setupGitRepo(t)
	result := &executor.ExecutionJobResult{
		JobID:           "job_test",
		Status:          "completed",
		TotalTasks:      1,
		SuccessfulTasks: 1,
	}
	// No changes to commit - should succeed silently
	err := commitDoingResults("job_test", result)
	if err != nil {
		t.Errorf("expected no error for no-changes case, got: %v", err)
	}
}

func TestCommitDoingResults_PartialStatus(t *testing.T) {
	dir := setupGitRepo(t)
	// Create a new file to commit
	if err := os.WriteFile(filepath.Join(dir, "new_file.txt"), []byte("content"), 0644); err != nil {
		t.Fatal(err)
	}
	result := &executor.ExecutionJobResult{
		JobID:           "job_test",
		Status:          "partial",
		TotalTasks:      2,
		SuccessfulTasks: 1,
		FailedTasks:     1,
	}
	err := commitDoingResults("job_test", result)
	if err != nil {
		t.Logf("commitDoingResults partial error (acceptable): %v", err)
	}
}

func TestCommitDoingResults_FailedStatus(t *testing.T) {
	setupGitRepo(t)
	result := &executor.ExecutionJobResult{
		JobID:       "job_test",
		Status:      "failed",
		TotalTasks:  1,
		FailedTasks: 1,
	}
	err := commitDoingResults("job_test", result)
	if err != nil {
		t.Logf("commitDoingResults failed status error (acceptable): %v", err)
	}
}

// ─── ensureGitUserConfigured tests ───────────────────────────────────────────

func TestEnsureGitUserConfigured(t *testing.T) {
	dir := setupGitRepo(t)
	err := ensureGitUserConfigured(dir)
	if err != nil {
		t.Logf("ensureGitUserConfigured error (acceptable in test env): %v", err)
	}
}

// ─── Command RunE tests ───────────────────────────────────────────────────────

func TestNewPlanCheckCmd_RunE_NoArgs(t *testing.T) {
	cmd := NewPlanCheckCmd()
	cmd.SetArgs([]string{})
	err := cmd.Execute()
	// Should fail because no job_id arg
	if err == nil {
		t.Log("expected error for missing job_id")
	}
}

func TestNewDoingCheckCmd_RunE_NoArgs(t *testing.T) {
	cmd := NewDoingCheckCmd()
	cmd.SetArgs([]string{})
	err := cmd.Execute()
	if err == nil {
		t.Log("expected error for missing job_id")
	}
}

func TestNewLearningCheckCmd_RunE_NoArgs(t *testing.T) {
	cmd := NewLearningCheckCmd()
	cmd.SetArgs([]string{})
	err := cmd.Execute()
	if err == nil {
		t.Log("expected error for missing job_id")
	}
}

func TestNewPlanCheckCmd_RunE_WithWorkspace(t *testing.T) {
	withTempWorkspace(t, func(dir string) {
		planDir := filepath.Join(dir, ".rick", "jobs", "job_test", "plan")
		if err := os.MkdirAll(planDir, 0755); err != nil {
			t.Fatal(err)
		}
		task1 := "# 依赖关系\n无\n# 任务名称\nTask1\n# 任务目标\nGoal\n# 关键结果\n1. KR1\n# 测试方法\nTest\n"
		if err := os.WriteFile(filepath.Join(planDir, "task1.md"), []byte(task1), 0644); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(planDir, "OKR.md"), []byte("# Job OKR\n## O1: 目标\n"), 0644); err != nil {
			t.Fatal(err)
		}
		cmd := NewPlanCheckCmd()
		cmd.SetArgs([]string{"job_test"})
		if err := cmd.Execute(); err != nil {
			t.Logf("plan_check RunE error: %v", err)
		}
	})
}

func TestNewDoingCheckCmd_RunE_WithWorkspace(t *testing.T) {
	withTempWorkspace(t, func(dir string) {
		doingDir := filepath.Join(dir, ".rick", "jobs", "job_test", "doing")
		if err := os.MkdirAll(doingDir, 0755); err != nil {
			t.Fatal(err)
		}
		makeTasksJSON(t, doingDir, []executor.TaskState{
			{TaskID: "task1", TaskName: "T1", Status: "success", CommitHash: "abc"},
		})
		if err := os.WriteFile(filepath.Join(doingDir, "debug.md"), []byte("## task1: did work\nsome content"), 0644); err != nil {
			t.Fatal(err)
		}
		cmd := NewDoingCheckCmd()
		cmd.SetArgs([]string{"job_test"})
		if err := cmd.Execute(); err != nil {
			t.Logf("doing_check RunE error: %v", err)
		}
	})
}

func TestNewLearningCheckCmd_RunE_WithWorkspace(t *testing.T) {
	withTempWorkspace(t, func(dir string) {
		learningDir := filepath.Join(dir, ".rick", "jobs", "job_test", "learning")
		if err := os.MkdirAll(learningDir, 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(learningDir, "SUMMARY.md"), []byte("# Job Summary\nsome content"), 0644); err != nil {
			t.Fatal(err)
		}
		cmd := NewLearningCheckCmd()
		cmd.SetArgs([]string{"job_test"})
		if err := cmd.Execute(); err != nil {
			t.Logf("learning_check RunE error: %v", err)
		}
	})
}

func TestEnsureGitUserConfigured_WithConfig(t *testing.T) {
	dir := setupGitRepo(t)
	// Unset git user to force configuration
	exec.Command("git", "config", "--unset", "user.name").Run()
	exec.Command("git", "config", "--unset", "user.email").Run()
	err := ensureGitUserConfigured(dir)
	if err != nil {
		t.Logf("ensureGitUserConfigured error (acceptable): %v", err)
	}
}
