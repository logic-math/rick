package handler

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDoing_NoJobDir(t *testing.T) {
	orig, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(orig) }()

	if err := os.MkdirAll(filepath.Join(dir, ".rick"), 0755); err != nil {
		t.Fatal(err)
	}

	err = Doing("job_test", Options{})
	if err == nil {
		t.Fatal("expected error for missing job dir")
	}
}

func TestDoing_NoPlanDir(t *testing.T) {
	orig, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(orig) }()

	if err := os.MkdirAll(filepath.Join(dir, ".rick", "jobs", "job_test"), 0755); err != nil {
		t.Fatal(err)
	}

	err = Doing("job_test", Options{})
	if err == nil {
		t.Fatal("expected error for missing plan dir")
	}
}

func TestDoing_NoTasks(t *testing.T) {
	orig, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(orig) }()

	if err := os.MkdirAll(filepath.Join(dir, ".rick", "jobs", "job_test", "plan"), 0755); err != nil {
		t.Fatal(err)
	}

	err = Doing("job_test", Options{})
	if err == nil {
		t.Fatal("expected error for no tasks")
	}
}

func TestDoingDryRun_Orchestration(t *testing.T) {
	orig, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(orig) }()

	planDir := filepath.Join(dir, ".rick", "jobs", "job_1", "plan")
	if err := os.MkdirAll(planDir, 0755); err != nil {
		t.Fatal(err)
	}
	task1 := "# 依赖关系\n无\n# 任务名称\nTask1\n# 任务目标\nGoal\n# 关键结果\n1. KR1\n# 测试方法\nTest\n"
	task2 := "# 依赖关系\ntask1\n# 任务名称\nTask2\n# 任务目标\nGoal\n# 关键结果\n1. KR1\n# 测试方法\nTest\n"
	if err := os.WriteFile(filepath.Join(planDir, "task1.md"), []byte(task1), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(planDir, "task2.md"), []byte(task2), 0644); err != nil {
		t.Fatal(err)
	}

	if err := DoingDryRun("job_1"); err != nil {
		t.Fatalf("DoingDryRun: %v", err)
	}

	// The prompt was written to doing/prompts/doing_prompt.md; verify the
	// orchestration content exists on disk (DoingDryRun prints to stdout).
	promptFile := filepath.Join(dir, ".rick", "jobs", "job_1", "doing", "prompts", "doing_prompt.md")
	content, err := os.ReadFile(promptFile)
	if err != nil {
		t.Fatalf("read prompt file: %v", err)
	}
	s := string(content)
	if !strings.Contains(s, "workflowScript") {
		t.Error("doing prompt must contain workflowScript orchestration")
	}
	if !strings.Contains(s, "runs.run('task1'") {
		t.Error("doing prompt must contain runs.run('task1'")
	}
}
