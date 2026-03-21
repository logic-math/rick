package parser

import (
	"strings"
	"testing"
)

// Sample task.md content for testing
const sampleTaskMD = `# 依赖关系
task1, task2

# 任务名称
实现 JSON 格式输出

# 任务目标
为日志系统添加 JSON 格式输出支持，使日志可以被机器解析

# 关键结果
- 实现 JSON 格式化函数
- 支持所有日志字段的序列化
- 提供格式切换配置

# 测试方法
- 运行单元测试验证 JSON 格式正确
- 测试各种日志级别的输出
- 验证性能无明显下降
`

// Sample task.md without dependencies
const sampleTaskMDNoDeps = `# 依赖关系

# 任务名称
基础任务

# 任务目标
完成基础功能

# 关键结果
- 结果1
- 结果2

# 测试方法
- 测试步骤1
- 测试步骤2
`

// Sample task.md with empty key results
const sampleTaskMDEmptyKeyResults = `# 依赖关系
task1

# 任务名称
任务标题

# 任务目标
任务目标描述

# 关键结果

# 测试方法
- 测试步骤1
`

func TestParseTask(t *testing.T) {
	task, err := ParseTask(sampleTaskMD)
	if err != nil {
		t.Fatalf("ParseTask failed: %v", err)
	}

	if task.Name != "实现 JSON 格式输出" {
		t.Errorf("Expected task name '实现 JSON 格式输出', got '%s'", task.Name)
	}

	if task.Goal != "为日志系统添加 JSON 格式输出支持，使日志可以被机器解析" {
		t.Errorf("Expected specific goal, got '%s'", task.Goal)
	}

	if len(task.Dependencies) != 2 {
		t.Errorf("Expected 2 dependencies, got %d", len(task.Dependencies))
	}

	if task.Dependencies[0] != "task1" || task.Dependencies[1] != "task2" {
		t.Errorf("Expected dependencies [task1, task2], got %v", task.Dependencies)
	}

	if len(task.KeyResults) != 3 {
		t.Errorf("Expected 3 key results, got %d", len(task.KeyResults))
	}

	if !strings.Contains(task.TestMethod, "单元测试") {
		t.Errorf("Expected test method to contain '单元测试', got '%s'", task.TestMethod)
	}
}

func TestParseDependencies(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		expected []string
	}{
		{
			name:     "With dependencies",
			content:  "# 依赖关系\ntask1, task2, task3",
			expected: []string{"task1", "task2", "task3"},
		},
		{
			name:     "Single dependency",
			content:  "# 依赖关系\ntask1",
			expected: []string{"task1"},
		},
		{
			name:     "No dependencies",
			content:  "# 依赖关系\n",
			expected: []string{},
		},
		{
			name:     "With spaces",
			content:  "# 依赖关系\n  task1  ,  task2  ",
			expected: []string{"task1", "task2"},
		},
		{
			name:     "Missing section",
			content:  "# 任务名称\nSome task",
			expected: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			deps, err := ParseDependencies(tt.content)
			if err != nil {
				t.Fatalf("ParseDependencies failed: %v", err)
			}

			if len(deps) != len(tt.expected) {
				t.Errorf("Expected %d dependencies, got %d", len(tt.expected), len(deps))
			}

			for i, dep := range deps {
				if i >= len(tt.expected) || dep != tt.expected[i] {
					t.Errorf("Dependency mismatch at index %d: expected '%s', got '%s'", i, tt.expected[i], dep)
				}
			}
		})
	}
}

func TestParseTaskName(t *testing.T) {
	tests := []struct {
		name      string
		content   string
		expected  string
		shouldErr bool
	}{
		{
			name:      "Valid task name",
			content:   "# 任务名称\n实现 JSON 格式输出",
			expected:  "实现 JSON 格式输出",
			shouldErr: false,
		},
		{
			name:      "Task name with spaces",
			content:   "# 任务名称\n  任务标题  ",
			expected:  "任务标题",
			shouldErr: false,
		},
		{
			name:      "Missing task name",
			content:   "# 依赖关系\ntask1",
			expected:  "",
			shouldErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			name, err := ParseTaskName(tt.content)
			if (err != nil) != tt.shouldErr {
				t.Errorf("Expected error: %v, got: %v", tt.shouldErr, err)
			}
			if name != tt.expected {
				t.Errorf("Expected '%s', got '%s'", tt.expected, name)
			}
		})
	}
}

func TestParseGoal(t *testing.T) {
	tests := []struct {
		name      string
		content   string
		expected  string
		shouldErr bool
	}{
		{
			name:      "Valid goal",
			content:   "# 任务目标\n为日志系统添加 JSON 格式输出支持",
			expected:  "为日志系统添加 JSON 格式输出支持",
			shouldErr: false,
		},
		{
			name:      "Goal with multiple lines",
			content:   "# 任务目标\n第一行\n第二行",
			expected:  "第一行\n第二行",
			shouldErr: false,
		},
		{
			name:      "Missing goal",
			content:   "# 任务名称\n任务标题",
			expected:  "",
			shouldErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			goal, err := ParseGoal(tt.content)
			if (err != nil) != tt.shouldErr {
				t.Errorf("Expected error: %v, got: %v", tt.shouldErr, err)
			}
			if goal != tt.expected {
				t.Errorf("Expected '%s', got '%s'", tt.expected, goal)
			}
		})
	}
}

func TestParseKeyResults(t *testing.T) {
	tests := []struct {
		name      string
		content   string
		minCount  int
		shouldErr bool
	}{
		{
			name:      "Valid key results with dashes",
			content:   "# 关键结果\n- 结果1\n- 结果2\n- 结果3",
			minCount:  3,
			shouldErr: false,
		},
		{
			name:      "Single key result",
			content:   "# 关键结果\n- 唯一结果",
			minCount:  1,
			shouldErr: false,
		},
		{
			name:      "Empty key results",
			content:   "# 关键结果\n",
			minCount:  0,
			shouldErr: false,
		},
		{
			name:      "Missing key results section",
			content:   "# 任务名称\n任务标题",
			minCount:  0,
			shouldErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results, err := ParseKeyResults(tt.content)
			if (err != nil) != tt.shouldErr {
				t.Errorf("Expected error: %v, got: %v", tt.shouldErr, err)
			}
			if len(results) < tt.minCount {
				t.Errorf("Expected at least %d results, got %d", tt.minCount, len(results))
			}
		})
	}
}

func TestParseTestMethod(t *testing.T) {
	tests := []struct {
		name      string
		content   string
		shouldErr bool
		hasContent bool
	}{
		{
			name:      "Valid test method with dashes",
			content:   "# 测试方法\n- 测试步骤1\n- 测试步骤2",
			shouldErr: false,
			hasContent: true,
		},
		{
			name:      "Empty test method",
			content:   "# 测试方法\n",
			shouldErr: false,
			hasContent: false,
		},
		{
			name:      "Missing test method section",
			content:   "# 任务名称\n任务标题",
			shouldErr: false,
			hasContent: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			method, err := ParseTestMethod(tt.content)
			if (err != nil) != tt.shouldErr {
				t.Errorf("Expected error: %v, got: %v", tt.shouldErr, err)
			}
			hasContent := method != ""
			if hasContent != tt.hasContent {
				t.Errorf("Expected hasContent: %v, got: %v", tt.hasContent, hasContent)
			}
		})
	}
}

func TestExtractSectionContent(t *testing.T) {
	tests := []struct {
		name      string
		content   string
		heading   string
		expected  string
	}{
		{
			name: "Extract between two headings",
			content: `# 依赖关系
task1, task2

# 任务名称
实现功能`,
			heading:  "# 依赖关系",
			expected: "task1, task2",
		},
		{
			name: "Extract last section",
			content: `# 任务名称
实现功能

# 测试方法
1. 测试步骤1`,
			heading:  "# 测试方法",
			expected: "1. 测试步骤1",
		},
		{
			name: "Section with empty content",
			content: `# 依赖关系

# 任务名称
实现功能`,
			heading:  "# 依赖关系",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractSectionContent(tt.content, tt.heading)
			if result != tt.expected {
				t.Errorf("Expected '%s', got '%s'", tt.expected, result)
			}
		})
	}
}

func TestValidateTask(t *testing.T) {
	tests := []struct {
		name      string
		task      *Task
		shouldErr bool
	}{
		{
			name: "Valid task",
			task: &Task{
				Name:       "Task Name",
				Goal:       "Task Goal",
				TestMethod: "Test Method",
			},
			shouldErr: false,
		},
		{
			name: "Missing name",
			task: &Task{
				Goal:       "Task Goal",
				TestMethod: "Test Method",
			},
			shouldErr: true,
		},
		{
			name: "Missing goal",
			task: &Task{
				Name:       "Task Name",
				TestMethod: "Test Method",
			},
			shouldErr: true,
		},
		{
			name: "Missing test method",
			task: &Task{
				Name: "Task Name",
				Goal: "Task Goal",
			},
			shouldErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateTask(tt.task)
			if (err != nil) != tt.shouldErr {
				t.Errorf("Expected error: %v, got: %v", tt.shouldErr, err)
			}
		})
	}
}

func TestParseTaskWithRealExample(t *testing.T) {
	task, err := ParseTask(sampleTaskMD)
	if err != nil {
		t.Fatalf("ParseTask failed: %v", err)
	}

	// Validate all fields
	err = ValidateTask(task)
	if err != nil {
		t.Fatalf("ValidateTask failed: %v", err)
	}

	if len(task.Dependencies) != 2 {
		t.Errorf("Expected 2 dependencies, got %d", len(task.Dependencies))
	}

	if len(task.KeyResults) != 3 {
		t.Errorf("Expected 3 key results, got %d", len(task.KeyResults))
	}

	if task.Name == "" || task.Goal == "" || task.TestMethod == "" {
		t.Error("Task has empty required fields")
	}
}

func TestParseTaskWithNoDependencies(t *testing.T) {
	task, err := ParseTask(sampleTaskMDNoDeps)
	if err != nil {
		t.Fatalf("ParseTask failed: %v", err)
	}

	if len(task.Dependencies) != 0 {
		t.Errorf("Expected 0 dependencies, got %d", len(task.Dependencies))
	}

	if task.Name != "基础任务" {
		t.Errorf("Expected task name '基础任务', got '%s'", task.Name)
	}
}

func TestParseTaskWithEmptyKeyResults(t *testing.T) {
	task, err := ParseTask(sampleTaskMDEmptyKeyResults)
	if err != nil {
		t.Fatalf("ParseTask failed: %v", err)
	}

	if len(task.KeyResults) != 0 {
		t.Errorf("Expected 0 key results, got %d", len(task.KeyResults))
	}
}

func BenchmarkParseTask(b *testing.B) {
	for i := 0; i < b.N; i++ {
		ParseTask(sampleTaskMD)
	}
}

func BenchmarkParseDependencies(b *testing.B) {
	for i := 0; i < b.N; i++ {
		ParseDependencies("# 依赖关系\ntask1, task2, task3")
	}
}

func TestParseDependencies_ChineseNone(t *testing.T) {
	content := `# 依赖关系
无

# 任务名称
测试任务
`
	deps, err := ParseDependencies(content)
	if err != nil {
		t.Fatalf("ParseDependencies failed: %v", err)
	}

	if len(deps) != 0 {
		t.Errorf("Expected 0 dependencies for '无', got %d: %v", len(deps), deps)
	}
}

func TestIsNoDependency(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{"无", true},
		{"None", true},
		{"none", true},
		{"NONE", true},
		{"null", true},
		{"nil", true},
		{"n/a", true},
		{"N/A", true},
		{"-", true},
		{"task1", false},
		{"", false},
	}

	for _, tt := range tests {
		result := isNoDependency(tt.input)
		if result != tt.expected {
			t.Errorf("isNoDependency(%q) = %v, want %v", tt.input, result, tt.expected)
		}
	}
}
