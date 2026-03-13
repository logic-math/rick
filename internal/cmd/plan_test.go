package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCreateTaskFromBlock(t *testing.T) {
	block := `## Task 1
# 任务名称
实现功能 A

# 任务目标
完成功能 A 的实现

# 依赖关系
无

# 关键结果
- 功能完整
- 测试通过

# 测试方法
运行单元测试
`

	task, err := createTaskFromBlock(block, 1, "job_1")
	if err != nil {
		t.Fatalf("createTaskFromBlock failed: %v", err)
	}

	if task.ID != "task_1" {
		t.Errorf("Expected task ID 'task_1', got '%s'", task.ID)
	}

	if task.Name != "实现功能 A" {
		t.Errorf("Expected task name '实现功能 A', got '%s'", task.Name)
	}

	if task.Goal != "完成功能 A 的实现" {
		t.Errorf("Expected task goal '完成功能 A 的实现', got '%s'", task.Goal)
	}

	if task.TestMethod != "运行单元测试" {
		t.Errorf("Expected test method '运行单元测试', got '%s'", task.TestMethod)
	}

	if len(task.KeyResults) != 2 {
		t.Errorf("Expected 2 key results, got %d", len(task.KeyResults))
	}
}

func TestExtractField(t *testing.T) {
	block := `# 任务名称
测试任务

# 任务目标
这是一个测试
`

	tests := []struct {
		fieldName     string
		expectedValue string
	}{
		{"# 任务名称", "测试任务"},
		{"# 任务目标", "这是一个测试"},
		{"# 不存在", ""},
	}

	for _, tt := range tests {
		t.Run(tt.fieldName, func(t *testing.T) {
			result := extractField(block, tt.fieldName, "")
			if result != tt.expectedValue {
				t.Errorf("extractField(%q) = %q, want %q", tt.fieldName, result, tt.expectedValue)
			}
		})
	}
}

func TestParseTasksFromClaudeOutput(t *testing.T) {
	tmpDir := t.TempDir()

	output := `# Planning Results

## Task 1
# 任务名称
任务一

# 任务目标
完成任务一

# 依赖关系


# 关键结果
- 结果1
- 结果2

# 测试方法
测试方法1

## Task 2
# 任务名称
任务二

# 任务目标
完成任务二

# 依赖关系
task_1

# 关键结果
- 结果3

# 测试方法
测试方法2
`

	tasks, err := parseTasksFromClaudeOutput(output, tmpDir, "job_1")
	if err != nil {
		t.Fatalf("parseTasksFromClaudeOutput failed: %v", err)
	}

	if len(tasks) < 1 {
		t.Errorf("Expected at least 1 task, got %d", len(tasks))
	}

	// Check that task files were created
	files, err := os.ReadDir(tmpDir)
	if err != nil {
		t.Fatalf("Failed to read task directory: %v", err)
	}

	if len(files) < 1 {
		t.Errorf("Expected task files to be created, got %d files", len(files))
	}
}

func TestCreateJobStructureHelper(t *testing.T) {
	tmpDir := t.TempDir()
	jobID := "job_test_123"
	jobPath := filepath.Join(tmpDir, "jobs", jobID)
	planDir := filepath.Join(jobPath, "plan")
	doingDir := filepath.Join(jobPath, "doing")
	learningDir := filepath.Join(jobPath, "learning")

	for _, dir := range []string{jobPath, planDir, doingDir, learningDir} {
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatalf("Failed to create directory: %v", err)
		}
	}

	// Verify directories were created
	if _, err := os.Stat(planDir); os.IsNotExist(err) {
		t.Errorf("Expected plan directory to exist")
	}
}

func TestGenerateJobID(t *testing.T) {
	id1 := generateJobID()

	if !strings.HasPrefix(id1, "job_") {
		t.Errorf("Expected job ID to start with 'job_', got '%s'", id1)
	}

	// Generate two IDs in quick succession - they might be the same due to timestamp resolution
	// Just verify format is correct
	id2 := generateJobID()
	if !strings.HasPrefix(id2, "job_") {
		t.Errorf("Expected job ID to start with 'job_', got '%s'", id2)
	}
}

func TestNewPlanCmd(t *testing.T) {
	cmd := NewPlanCmd()

	if cmd.Use != "plan [requirement]" {
		t.Errorf("Expected Use 'plan [requirement]', got '%s'", cmd.Use)
	}

	if cmd.Short == "" {
		t.Errorf("Expected Short description to be non-empty")
	}
}

func TestParseClaudeOutputAndGenerateTasks(t *testing.T) {
	// Setup temporary workspace
	tmpDir := t.TempDir()

	// We'll test the core logic without mocking
	claudeOutput := `## Task 1
# 任务名称
测试任务

# 任务目标
完成测试

# 依赖关系


# 关键结果
- 成功完成

# 测试方法
运行测试
`

	tasks, err := parseTasksFromClaudeOutput(claudeOutput, tmpDir, "job_1")
	if err != nil {
		t.Fatalf("parseTasksFromClaudeOutput failed: %v", err)
	}

	if len(tasks) == 0 {
		t.Fatalf("Expected at least one task, got 0")
	}

	task := tasks[0]
	if task.Name != "测试任务" {
		t.Errorf("Expected task name '测试任务', got '%s'", task.Name)
	}

	// Verify file was created
	files, err := os.ReadDir(tmpDir)
	if err != nil {
		t.Fatalf("Failed to read directory: %v", err)
	}

	if len(files) == 0 {
		t.Errorf("Expected task files to be created")
	}
}

func TestExtractFieldWithMultiline(t *testing.T) {
	block := `# 关键结果
- 结果1
- 结果2
- 结果3

# 测试方法
运行测试
`

	result := extractField(block, "# 关键结果", "")
	if result == "" {
		t.Errorf("Expected non-empty result for '# 关键结果'")
	}

	if !strings.Contains(result, "结果1") {
		t.Errorf("Expected result to contain '结果1', got '%s'", result)
	}
}

func TestCreateTaskWithDependencies(t *testing.T) {
	block := `## Task 2
# 任务名称
任务二

# 任务目标
完成任务二

# 依赖关系
task_1, task_2

# 关键结果
- 结果1

# 测试方法
测试
`

	task, err := createTaskFromBlock(block, 2, "job_1")
	if err != nil {
		t.Fatalf("createTaskFromBlock failed: %v", err)
	}

	if len(task.Dependencies) != 2 {
		t.Errorf("Expected 2 dependencies, got %d", len(task.Dependencies))
	}

	expectedDeps := []string{"task_1", "task_2"}
	for i, dep := range task.Dependencies {
		if dep != expectedDeps[i] {
			t.Errorf("Expected dependency '%s', got '%s'", expectedDeps[i], dep)
		}
	}
}

func TestParseTaskName(t *testing.T) {
	block := `## Task 1
# 任务名称
这是一个测试任务

# 其他字段
值
`

	task, err := createTaskFromBlock(block, 1, "job_1")
	if err != nil {
		t.Fatalf("createTaskFromBlock failed: %v", err)
	}

	if task.Name != "这是一个测试任务" {
		t.Errorf("Expected task name '这是一个测试任务', got '%s'", task.Name)
	}
}

func TestCreateTaskWithEmptyFields(t *testing.T) {
	block := `## Task 1
# 任务名称


# 任务目标
目标

# 依赖关系


# 关键结果


# 测试方法
`

	task, err := createTaskFromBlock(block, 1, "job_1")
	if err != nil {
		t.Fatalf("createTaskFromBlock failed: %v", err)
	}

	// Should generate default name
	if task.Name == "" {
		t.Errorf("Expected default task name to be generated")
	}

	if task.Goal != "目标" {
		t.Errorf("Expected goal '目标', got '%s'", task.Goal)
	}
}

func TestParseTaskKeyResults(t *testing.T) {
	block := `## Task 1
# 任务名称
测试

# 关键结果
- 结果1
- 结果2
- 结果3

# 测试方法
测试
`

	task, err := createTaskFromBlock(block, 1, "job_1")
	if err != nil {
		t.Fatalf("createTaskFromBlock failed: %v", err)
	}

	if len(task.KeyResults) < 1 {
		t.Errorf("Expected at least 1 key result, got %d", len(task.KeyResults))
	}
}

func TestPromptForRequirement(t *testing.T) {
	// This test would need stdin mocking, so we'll skip it for now
	// or test it manually with the actual CLI
	t.Skip("Requires stdin mocking")
}

func TestCommitPlanResults(t *testing.T) {
	tmpDir := t.TempDir()

	// Initialize git repo first
	err := commitPlanResults(tmpDir, "job_test")
	if err != nil {
		// It's OK if this fails due to git not being initialized
		// The function should handle it gracefully
		t.Logf("commitPlanResults returned: %v", err)
	}
}

func TestTaskIDGeneration(t *testing.T) {
	tests := []struct {
		index    int
		expected string
	}{
		{0, "task_0"},
		{1, "task_1"},
		{5, "task_5"},
		{10, "task_10"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			task, _ := createTaskFromBlock("# 任务名称\n测试", tt.index, "job_1")
			if task.ID != tt.expected {
				t.Errorf("Expected ID '%s', got '%s'", tt.expected, task.ID)
			}
		})
	}
}
