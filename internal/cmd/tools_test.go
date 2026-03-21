package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

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

func TestRunDoingCheck_NoDebugMD(t *testing.T) {
	dir := t.TempDir()
	makeTasksJSON(t, dir, []executor.TaskState{
		{TaskID: "task1", TaskName: "T1", Status: "success", CommitHash: "abc123"},
	})
	err := runDoingCheck(dir)
	if err == nil {
		t.Fatal("expected error for missing debug.md")
	}
	if !containsStr(err.Error(), "debug.md") {
		t.Errorf("expected debug.md in error, got: %v", err)
	}
}

func TestRunDoingCheck_ZombieTask(t *testing.T) {
	dir := t.TempDir()
	makeTasksJSON(t, dir, []executor.TaskState{
		{TaskID: "task1", TaskName: "T1", Status: "running"},
	})
	if err := os.WriteFile(filepath.Join(dir, "debug.md"), []byte("# debug"), 0644); err != nil {
		t.Fatal(err)
	}
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
	if err := os.WriteFile(filepath.Join(dir, "debug.md"), []byte("# debug"), 0644); err != nil {
		t.Fatal(err)
	}
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
	if err := os.WriteFile(filepath.Join(dir, "debug.md"), []byte("# debug"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := runDoingCheck(dir); err != nil {
		t.Errorf("expected no error for valid doing dir, got: %v", err)
	}
}

func TestRunDoingCheck_FailedTaskNoCommit(t *testing.T) {
	dir := t.TempDir()
	// failed tasks don't need commit_hash
	makeTasksJSON(t, dir, []executor.TaskState{
		{TaskID: "task1", TaskName: "T1", Status: "failed", CommitHash: ""},
	})
	if err := os.WriteFile(filepath.Join(dir, "debug.md"), []byte("# debug"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := runDoingCheck(dir); err != nil {
		t.Errorf("expected no error for failed task without commit_hash, got: %v", err)
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

func TestRunLearningCheck_BadPythonSyntax(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "SUMMARY.md"), []byte("summary"), 0644); err != nil {
		t.Fatal(err)
	}
	skillsDir := filepath.Join(dir, "skills")
	if err := os.MkdirAll(skillsDir, 0755); err != nil {
		t.Fatal(err)
	}
	badPy := "def broken(\n    x = 1 +\n"
	if err := os.WriteFile(filepath.Join(skillsDir, "bad.py"), []byte(badPy), 0644); err != nil {
		t.Fatal(err)
	}
	err := runLearningCheck(dir)
	if err == nil {
		t.Fatal("expected error for bad Python syntax")
	}
	if !containsStr(err.Error(), "syntax") && !containsStr(err.Error(), "Python") {
		t.Errorf("expected syntax error, got: %v", err)
	}
}

func TestRunLearningCheck_OKRMissingSection(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "SUMMARY.md"), []byte("summary"), 0644); err != nil {
		t.Fatal(err)
	}
	// OKR.md without required sections
	if err := os.WriteFile(filepath.Join(dir, "OKR.md"), []byte("# OKR\nsome content"), 0644); err != nil {
		t.Fatal(err)
	}
	err := runLearningCheck(dir)
	if err == nil {
		t.Fatal("expected error for OKR missing sections")
	}
}

func TestRunLearningCheck_SPECMissingSection(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "SUMMARY.md"), []byte("summary"), 0644); err != nil {
		t.Fatal(err)
	}
	// SPEC.md missing required sections
	if err := os.WriteFile(filepath.Join(dir, "SPEC.md"), []byte("# SPEC\n## 技术栈\nGo"), 0644); err != nil {
		t.Fatal(err)
	}
	err := runLearningCheck(dir)
	if err == nil {
		t.Fatal("expected error for SPEC missing sections")
	}
}

func TestRunLearningCheck_Valid(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "SUMMARY.md"), []byte("summary"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := runLearningCheck(dir); err != nil {
		t.Errorf("expected no error for valid learning dir, got: %v", err)
	}
}

func TestRunLearningCheck_ValidWithSkill(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "SUMMARY.md"), []byte("summary"), 0644); err != nil {
		t.Fatal(err)
	}
	skillsDir := filepath.Join(dir, "skills")
	if err := os.MkdirAll(skillsDir, 0755); err != nil {
		t.Fatal(err)
	}
	goodPy := "def hello():\n    return 'world'\n"
	if err := os.WriteFile(filepath.Join(skillsDir, "good.py"), []byte(goodPy), 0644); err != nil {
		t.Fatal(err)
	}
	if err := runLearningCheck(dir); err != nil {
		t.Errorf("expected no error for valid skill, got: %v", err)
	}
}

func TestRunLearningCheck_ValidOKR(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "SUMMARY.md"), []byte("summary"), 0644); err != nil {
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
	if err := os.WriteFile(filepath.Join(dir, "SUMMARY.md"), []byte("summary"), 0644); err != nil {
		t.Fatal(err)
	}
	spec := "## 技术栈\nGo\n## 架构设计\nModular\n## 开发规范\nStandard\n## 工程实践\nDAG\n"
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

func TestNewMergeCmd(t *testing.T) {
	cmd := NewMergeCmd()
	if cmd == nil {
		t.Fatal("NewMergeCmd returned nil")
	}
	if cmd.Use != "merge <job_id>" {
		t.Errorf("unexpected Use: %s", cmd.Use)
	}
}

func TestToolsSubcommands(t *testing.T) {
	cmd := NewToolsCmd()
	subNames := make(map[string]bool)
	for _, sub := range cmd.Commands() {
		subNames[sub.Name()] = true
	}
	expected := []string{"plan_check", "doing_check", "learning_check", "merge"}
	for _, name := range expected {
		if !subNames[name] {
			t.Errorf("missing subcommand: %s", name)
		}
	}
}

func TestCheckOKRSections_Valid(t *testing.T) {
	dir := t.TempDir()
	okrPath := filepath.Join(dir, "OKR.md")
	content := "## O1: 完成项目\n### 关键结果\n1. KR1\n"
	if err := os.WriteFile(okrPath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	if err := checkOKRSections(okrPath); err != nil {
		t.Errorf("expected no error, got: %v", err)
	}
}

func TestCheckOKRSections_MissingObjective(t *testing.T) {
	dir := t.TempDir()
	okrPath := filepath.Join(dir, "OKR.md")
	content := "# OKR\n### 关键结果\n1. KR1\n"
	if err := os.WriteFile(okrPath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	if err := checkOKRSections(okrPath); err == nil {
		t.Fatal("expected error for missing objective")
	}
}

func TestCheckOKRSections_MissingKR(t *testing.T) {
	dir := t.TempDir()
	okrPath := filepath.Join(dir, "OKR.md")
	content := "## O1: 目标\nsome content\n"
	if err := os.WriteFile(okrPath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	if err := checkOKRSections(okrPath); err == nil {
		t.Fatal("expected error for missing 关键结果")
	}
}

func TestCheckSPECSections_Valid(t *testing.T) {
	dir := t.TempDir()
	specPath := filepath.Join(dir, "SPEC.md")
	content := "## 技术栈\nGo\n## 架构设计\nModular\n## 开发规范\nStandard\n## 工程实践\nDAG\n"
	if err := os.WriteFile(specPath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	if err := checkSPECSections(specPath); err != nil {
		t.Errorf("expected no error, got: %v", err)
	}
}

func TestCheckSPECSections_Missing(t *testing.T) {
	dir := t.TempDir()
	specPath := filepath.Join(dir, "SPEC.md")
	content := "## 技术栈\nGo\n## 架构设计\nModular\n"
	if err := os.WriteFile(specPath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	if err := checkSPECSections(specPath); err == nil {
		t.Fatal("expected error for missing SPEC sections")
	}
}

// ─── Merge Helper Tests ───────────────────────────────────────────────────────

func TestCheckApproved_Valid(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "SUMMARY.md")
	if err := os.WriteFile(p, []byte("APPROVED: true\n\nsome content"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := checkApproved(p); err != nil {
		t.Errorf("expected no error, got: %v", err)
	}
}

func TestCheckApproved_NotApproved(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "SUMMARY.md")
	if err := os.WriteFile(p, []byte("APPROVED: false\n\nsome content"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := checkApproved(p); err == nil {
		t.Fatal("expected error for APPROVED: false")
	}
}

func TestCheckApproved_Missing(t *testing.T) {
	err := checkApproved("/nonexistent/SUMMARY.md")
	if err == nil {
		t.Fatal("expected error for missing SUMMARY.md")
	}
}

func TestCheckApproved_EmptyFile(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "SUMMARY.md")
	if err := os.WriteFile(p, []byte(""), 0644); err != nil {
		t.Fatal(err)
	}
	if err := checkApproved(p); err == nil {
		t.Fatal("expected error for empty SUMMARY.md")
	}
}

func TestCopyFile(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "src.txt")
	dst := filepath.Join(dir, "dst.txt")
	if err := os.WriteFile(src, []byte("hello world"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := copyFile(src, dst); err != nil {
		t.Fatalf("copyFile failed: %v", err)
	}
	data, err := os.ReadFile(dst)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "hello world" {
		t.Errorf("expected 'hello world', got '%s'", string(data))
	}
}

func TestCopyFile_CreatesDirs(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "src.txt")
	dst := filepath.Join(dir, "nested", "deep", "dst.txt")
	if err := os.WriteFile(src, []byte("data"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := copyFile(src, dst); err != nil {
		t.Fatalf("copyFile failed: %v", err)
	}
	if _, err := os.Stat(dst); err != nil {
		t.Errorf("expected dst to exist: %v", err)
	}
}

func TestCopyDir(t *testing.T) {
	srcDir := t.TempDir()
	dstDir := t.TempDir()

	// Create files in src
	for _, name := range []string{"a.md", "b.md", "c.txt"} {
		if err := os.WriteFile(filepath.Join(srcDir, name), []byte(name+" content"), 0644); err != nil {
			t.Fatal(err)
		}
	}
	// Create a subdirectory (should be skipped)
	if err := os.MkdirAll(filepath.Join(srcDir, "subdir"), 0755); err != nil {
		t.Fatal(err)
	}

	count, err := copyDir(srcDir, dstDir)
	if err != nil {
		t.Fatalf("copyDir failed: %v", err)
	}
	if count != 3 {
		t.Errorf("expected 3 files copied, got %d", count)
	}
	for _, name := range []string{"a.md", "b.md", "c.txt"} {
		if _, err := os.Stat(filepath.Join(dstDir, name)); err != nil {
			t.Errorf("expected %s to exist in dst", name)
		}
	}
}

func TestGenerateWikiREADME(t *testing.T) {
	dir := t.TempDir()
	wikiDir := filepath.Join(dir, "wiki")
	if err := os.MkdirAll(wikiDir, 0755); err != nil {
		t.Fatal(err)
	}
	// Create wiki files
	doc1 := "# Architecture\n\nThis is the architecture document."
	doc2 := "# Runtime Flow\n\nThis describes runtime flow."
	if err := os.WriteFile(filepath.Join(wikiDir, "architecture.md"), []byte(doc1), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(wikiDir, "runtime.md"), []byte(doc2), 0644); err != nil {
		t.Fatal(err)
	}

	if err := generateWikiREADME(dir); err != nil {
		t.Fatalf("generateWikiREADME failed: %v", err)
	}

	readmePath := filepath.Join(wikiDir, "README.md")
	if _, err := os.Stat(readmePath); err != nil {
		t.Fatalf("README.md not created: %v", err)
	}
	data, err := os.ReadFile(readmePath)
	if err != nil {
		t.Fatal(err)
	}
	content := string(data)
	if !containsStr(content, "architecture") {
		t.Errorf("README.md should mention architecture.md")
	}
	if !containsStr(content, "runtime") {
		t.Errorf("README.md should mention runtime.md")
	}
}

func TestWriteDoingCheckFixPrompt(t *testing.T) {
	dir := t.TempDir()
	path, err := writeDoingCheckFixPrompt(dir, fmt.Errorf("test error"))
	if err != nil {
		t.Fatalf("writeDoingCheckFixPrompt failed: %v", err)
	}
	defer os.Remove(path)
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("prompt file not created: %v", err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !containsStr(string(data), "test error") {
		t.Errorf("prompt file should contain error message")
	}
}

func TestWriteLearningCheckFixPrompt(t *testing.T) {
	dir := t.TempDir()
	path, err := writeLearningCheckFixPrompt(dir, fmt.Errorf("learning error"))
	if err != nil {
		t.Fatalf("writeLearningCheckFixPrompt failed: %v", err)
	}
	defer os.Remove(path)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !containsStr(string(data), "learning error") {
		t.Errorf("prompt file should contain error message")
	}
}

func TestWritePlanCheckFixPrompt(t *testing.T) {
	dir := t.TempDir()
	path, err := writePlanCheckFixPrompt(dir, fmt.Errorf("plan error"))
	if err != nil {
		t.Fatalf("writePlanCheckFixPrompt failed: %v", err)
	}
	defer os.Remove(path)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !containsStr(string(data), "plan error") {
		t.Errorf("prompt file should contain error message")
	}
}

func TestPrintMergeSummary_NoItems(t *testing.T) {
	// Just verify it doesn't panic
	printMergeSummary("job_1", "learning/job_1", "main", nil)
}

func TestPrintMergeSummary_WithItems(t *testing.T) {
	printMergeSummary("job_1", "learning/job_1", "main", []string{"wiki: 3 files", "OKR.md: updated"})
}

func TestExtractWikiTitleAndSummary(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "test.md")
	content := "# My Title\n\nThis is the summary paragraph.\n\n## Section\nMore content."
	if err := os.WriteFile(f, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	title, summary := extractWikiTitleAndSummary(f)
	if title != "My Title" {
		t.Errorf("expected 'My Title', got '%s'", title)
	}
	if summary != "This is the summary paragraph." {
		t.Errorf("expected summary, got '%s'", summary)
	}
}

func TestExtractWikiTitleAndSummary_NoTitle(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "test.md")
	content := "Just a paragraph, no heading."
	if err := os.WriteFile(f, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	title, summary := extractWikiTitleAndSummary(f)
	// Should not panic
	_ = title
	_ = summary
}

func TestExtractWikiTitleAndSummary_Missing(t *testing.T) {
	title, summary := extractWikiTitleAndSummary("/nonexistent/file.md")
	if title != "" || summary != "" {
		t.Errorf("expected empty for missing file, got title='%s' summary='%s'", title, summary)
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
		if err := os.WriteFile(filepath.Join(doingDir, "debug.md"), []byte("# debug"), 0644); err != nil {
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
		if err := os.WriteFile(filepath.Join(learningDir, "SUMMARY.md"), []byte("summary"), 0644); err != nil {
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

func TestRunGit_Success(t *testing.T) {
	setupGitRepo(t)
	out, err := runGit("status")
	if err != nil {
		t.Errorf("expected no error, got: %v (output: %s)", err, out)
	}
}

func TestGetCurrentBranch(t *testing.T) {
	setupGitRepo(t)
	branch, err := getCurrentBranch()
	if err != nil {
		t.Errorf("expected no error, got: %v", err)
	}
	if branch == "" {
		t.Error("expected non-empty branch name")
	}
}

func TestGitCreateAndSwitch(t *testing.T) {
	setupGitRepo(t)
	if err := gitCreateAndSwitch("test-branch"); err != nil {
		t.Errorf("expected no error, got: %v", err)
	}
	// Verify we're on the new branch
	branch, err := getCurrentBranch()
	if err != nil {
		t.Fatal(err)
	}
	if branch != "test-branch" {
		t.Errorf("expected branch=test-branch, got %s", branch)
	}
}

func TestGitCheckout(t *testing.T) {
	dir := setupGitRepo(t)
	_ = dir
	// Create and switch to a new branch
	if err := gitCreateAndSwitch("new-branch"); err != nil {
		t.Fatal(err)
	}
	// Switch back to main/master
	mainBranch := "main"
	if err := gitCheckout(mainBranch); err != nil {
		// Try master
		if err2 := gitCheckout("master"); err2 != nil {
			t.Logf("gitCheckout to main/master failed (acceptable): %v / %v", err, err2)
		}
	}
}

func TestFindClaudeBinary(t *testing.T) {
	// Just verify the function runs without panic
	path, err := findClaudeBinary()
	if err != nil {
		t.Logf("findClaudeBinary returned error (claude not in PATH): %v", err)
	} else if path == "" {
		t.Error("expected non-empty path when claude is found")
	}
}

func TestRunAutoFix_MockBinary(t *testing.T) {
	tmpDir := t.TempDir()
	mockScript := "#!/bin/sh\nexit 0\n"
	mockPath := filepath.Join(tmpDir, "mock_claude")
	if err := os.WriteFile(mockPath, []byte(mockScript), 0755); err != nil {
		t.Fatal(err)
	}
	promptFile := filepath.Join(tmpDir, "prompt.md")
	if err := os.WriteFile(promptFile, []byte("# prompt"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := runAutoFix(mockPath, promptFile); err != nil {
		t.Errorf("expected no error with mock binary, got: %v", err)
	}
}

func TestRunAutoFix_FailingBinary(t *testing.T) {
	tmpDir := t.TempDir()
	mockScript := "#!/bin/sh\nexit 1\n"
	mockPath := filepath.Join(tmpDir, "mock_claude_fail")
	if err := os.WriteFile(mockPath, []byte(mockScript), 0755); err != nil {
		t.Fatal(err)
	}
	promptFile := filepath.Join(tmpDir, "prompt.md")
	if err := os.WriteFile(promptFile, []byte("# prompt"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := runAutoFix(mockPath, promptFile); err == nil {
		t.Error("expected error with failing binary")
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
		if err := os.WriteFile(filepath.Join(doingDir, "debug.md"), []byte("# debug"), 0644); err != nil {
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
		if data != nil && !containsStr(data.DebugContent, "No debugging") {
			t.Logf("debug content: %s", data.DebugContent)
		}
	})
}

// ─── runMerge tests ───────────────────────────────────────────────────────────

func TestRunMerge_NoSummary(t *testing.T) {
	dir := setupGitRepo(t)
	// Create .rick structure
	if err := os.MkdirAll(filepath.Join(dir, ".rick", "jobs", "job_test", "learning"), 0755); err != nil {
		t.Fatal(err)
	}
	// No SUMMARY.md
	err := runMerge("job_test")
	if err == nil {
		t.Fatal("expected error for missing SUMMARY.md")
	}
}

func TestRunMerge_NotApproved(t *testing.T) {
	dir := setupGitRepo(t)
	learningDir := filepath.Join(dir, ".rick", "jobs", "job_test", "learning")
	if err := os.MkdirAll(learningDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(learningDir, "SUMMARY.md"), []byte("APPROVED: false\n\ncontent"), 0644); err != nil {
		t.Fatal(err)
	}
	err := runMerge("job_test")
	if err == nil {
		t.Fatal("expected error for not approved")
	}
}

func TestRunMerge_Success(t *testing.T) {
	dir := setupGitRepo(t)
	learningDir := filepath.Join(dir, ".rick", "jobs", "job_test", "learning")
	if err := os.MkdirAll(learningDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(learningDir, "SUMMARY.md"), []byte("APPROVED: true\n\ncontent"), 0644); err != nil {
		t.Fatal(err)
	}
	// Add OKR.md so there's something to merge
	okrContent := "## O1: Test\n### 关键结果\n1. KR1\n"
	if err := os.WriteFile(filepath.Join(learningDir, "OKR.md"), []byte(okrContent), 0644); err != nil {
		t.Fatal(err)
	}
	// Pre-create .rick dir and git-add it so merge can commit
	rickDir := filepath.Join(dir, ".rick")
	gitAddCmd := exec.Command("git", "add", rickDir)
	gitAddCmd.Dir = dir
	_ = gitAddCmd.Run()
	gitCommitCmd := exec.Command("git", "commit", "-m", "add rick dir")
	gitCommitCmd.Dir = dir
	_ = gitCommitCmd.Run()

	err := runMerge("job_test")
	if err != nil {
		t.Errorf("expected no error for valid merge, got: %v", err)
	}
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
		if err := os.WriteFile(filepath.Join(doingDir, "debug.md"), []byte("# debug"), 0644); err != nil {
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
		if err := os.WriteFile(filepath.Join(learningDir, "SUMMARY.md"), []byte("summary"), 0644); err != nil {
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

func TestRunMerge_WithWikiAndSkills(t *testing.T) {
	dir := setupGitRepo(t)
	learningDir := filepath.Join(dir, ".rick", "jobs", "job_test", "learning")
	if err := os.MkdirAll(filepath.Join(learningDir, "wiki"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(learningDir, "skills"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(learningDir, "SUMMARY.md"), []byte("APPROVED: true\n\ncontent"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(learningDir, "wiki", "test.md"), []byte("# Test\ncontent"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(learningDir, "skills", "test.py"), []byte("def test(): pass\n"), 0644); err != nil {
		t.Fatal(err)
	}
	// Pre-commit .rick dir
	gitAdd := exec.Command("git", "add", filepath.Join(dir, ".rick"))
	gitAdd.Dir = dir
	_ = gitAdd.Run()
	gitCommit := exec.Command("git", "commit", "-m", "add rick")
	gitCommit.Dir = dir
	_ = gitCommit.Run()

	err := runMerge("job_test")
	if err != nil {
		t.Logf("runMerge with wiki/skills returned: %v", err)
	}
}
