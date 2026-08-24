package handler

import (
	"fmt"
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

	err = Doing("job_test", Options{}, nil)
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

	err = Doing("job_test", Options{}, nil)
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

	err = Doing("job_test", Options{}, nil)
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
	task1 := "# 依赖关系\n无\n# 任务名称\nTask1\n# 任务目标\nGoal\n# 关键结果\n1. KR1\n# 测试方法\nTest\n# 写域\nsrc/a/\n"
	task2 := "# 依赖关系\ntask1\n# 任务名称\nTask2\n# 任务目标\nGoal\n# 关键结果\n1. KR1\n# 测试方法\nTest\n# 写域\nsrc/b/\n"
	if err := os.WriteFile(filepath.Join(planDir, "task1.md"), []byte(task1), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(planDir, "task2.md"), []byte(task2), 0644); err != nil {
		t.Fatal(err)
	}
	// v4.4：每层门禁程序（task1=第1层，task2=第2层）
	gatesDir := filepath.Join(planDir, "gates")
	if err := os.MkdirAll(gatesDir, 0755); err != nil {
		t.Fatal(err)
	}
	for i := 1; i <= 2; i++ {
		gate := "#!/usr/bin/env python3\nimport sys\nsys.exit(0)\n"
		if err := os.WriteFile(filepath.Join(gatesDir, fmt.Sprintf("gate%d.py", i)), []byte(gate), 0644); err != nil {
			t.Fatal(err)
		}
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
	// v4.4.2 测试收敛到层门禁：无 per-task test-worker，task 只 impl-worker（自测）
	if strings.Contains(s, "'task1-test'") {
		t.Error("doing prompt must NOT contain per-task test-worker (tests converged to layer gates)")
	}
	if !strings.Contains(s, "'task1-impl'") {
		t.Error("doing prompt must contain task1-impl (impl-worker with self-test)")
	}
	if !strings.Contains(s, "自测") {
		t.Error("doing prompt must reference self-test (# 测试方法 as process-level guidance)")
	}
	if !strings.Contains(s, "runs.all") {
		t.Error("doing prompt must contain runs.all (level-parallel fanout)")
	}
	if !strings.Contains(s, "level_complete") {
		t.Error("doing prompt must reference level_complete hook tool (layer checkpoint commit)")
	}
	if !strings.Contains(s, "步骤 ③") {
		t.Error("doing prompt must contain per-level step ③ (level_complete checkpoint)")
	}
	// v4.3.1 动态超时：timeoutMs 由工作量估算（≥20min），不再固定 3600000
	if !strings.Contains(s, "timeoutMs: ") {
		t.Error("doing prompt dispatches must carry dynamically estimated timeoutMs")
	}
	// v4.4 分层 pipeline + 层门禁
	if !strings.Contains(s, "门禁判别力验证") {
		t.Error("doing prompt must contain step-1 gate discriminability check")
	}
	if !strings.Contains(s, "gate_cmd") {
		t.Error("doing prompt must pass gate_cmd to level_complete")
	}
	if !strings.Contains(s, "debug 压缩传递") {
		t.Error("doing prompt must contain step-5 debug compression handoff")
	}
	if !strings.Contains(s, "# 写域") == false && strings.Contains(s, "写域互斥") == false {
		t.Error("doing prompt must mention write-domain disjointness")
	}
}
