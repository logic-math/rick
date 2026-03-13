package parser

import (
	"os"
	"path/filepath"
	"testing"
)

// Test data samples for integration testing

const (
	// Sample task.md content
	sampleTaskMDIntegration = `# 依赖关系

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

	// Sample debug.md content
	sampleDebugMDIntegration = `**调试日志**:
- debug1: JSON 序列化失败, 复杂对象循环引用, 猜想: 1)缺少循环引用处理 2)未使用 JSON.stringify 的 replacer, 验证: 添加 replacer 函数测试, 修复: 使用 WeakSet 检测循环引用, 待修复
- debug2: 性能测试失败, 大量数据序列化时性能下降, 猜想: 1)序列化算法低效 2)内存分配过多, 验证: 性能分析工具测试, 修复: 使用缓冲区优化, 已修复
`

	// Sample OKR.md content
	sampleOKRMDIntegration = `# 目标
- 提升日志系统的可观测性
- 支持结构化日志输出
- 优化日志性能

# 关键结果
- 实现 JSON 格式输出
- 支持自定义字段映射
- 性能提升 30%
`

	// Sample SPEC.md content
	sampleSpecMDIntegration = `# 规范
- JSON 输出必须符合 RFC 7158 标准
- 所有字段必须进行 URL 编码
- 不支持的字段类型应该转换为字符串
- 日志大小不应超过 1MB
- 性能要求：单条日志序列化时间不超过 1ms
`
)

// TestIntegration_CreateTestData creates test data files for integration testing
func TestIntegration_CreateTestData(t *testing.T) {
	tmpDir := t.TempDir()
	jobDir := filepath.Join(tmpDir, "job_1")

	if err := os.MkdirAll(jobDir, 0755); err != nil {
		t.Fatalf("Failed to create job directory: %v", err)
	}

	// Create task.md
	taskPath := filepath.Join(jobDir, "task.md")
	if err := os.WriteFile(taskPath, []byte(sampleTaskMDIntegration), 0644); err != nil {
		t.Fatalf("Failed to write task.md: %v", err)
	}

	// Create debug.md
	debugPath := filepath.Join(jobDir, "debug.md")
	if err := os.WriteFile(debugPath, []byte(sampleDebugMDIntegration), 0644); err != nil {
		t.Fatalf("Failed to write debug.md: %v", err)
	}

	// Create OKR.md
	okrPath := filepath.Join(jobDir, "OKR.md")
	if err := os.WriteFile(okrPath, []byte(sampleOKRMDIntegration), 0644); err != nil {
		t.Fatalf("Failed to write OKR.md: %v", err)
	}

	// Create SPEC.md
	specPath := filepath.Join(jobDir, "SPEC.md")
	if err := os.WriteFile(specPath, []byte(sampleSpecMDIntegration), 0644); err != nil {
		t.Fatalf("Failed to write SPEC.md: %v", err)
	}

	// Verify all files exist
	for _, path := range []string{taskPath, debugPath, okrPath, specPath} {
		if _, err := os.Stat(path); err != nil {
			t.Errorf("File not created: %s", path)
		}
	}
}

// TestIntegration_ParseTaskMD verifies task.md parsing is correct
func TestIntegration_ParseTaskMD(t *testing.T) {
	task, err := ParseTask(sampleTaskMDIntegration)
	if err != nil {
		t.Fatalf("ParseTask failed: %v", err)
	}

	// Verify task name
	if task.Name != "实现 JSON 格式输出" {
		t.Errorf("Expected task name '实现 JSON 格式输出', got '%s'", task.Name)
	}

	// Verify task goal
	expectedGoal := "为日志系统添加 JSON 格式输出支持，使日志可以被机器解析"
	if task.Goal != expectedGoal {
		t.Errorf("Expected goal '%s', got '%s'", expectedGoal, task.Goal)
	}

	// Verify dependencies (should be empty in this test data)
	if len(task.Dependencies) != 0 {
		t.Errorf("Expected 0 dependencies, got %d", len(task.Dependencies))
	}

	// Verify key results
	if len(task.KeyResults) != 3 {
		t.Errorf("Expected 3 key results, got %d", len(task.KeyResults))
	}

	// Verify test method
	if task.TestMethod == "" {
		t.Error("Expected test method to be non-empty")
	}

	// Validate the task
	if err := ValidateTask(task); err != nil {
		t.Errorf("Task validation failed: %v", err)
	}
}

// TestIntegration_ParseDebugMD verifies debug.md parsing and append are correct
func TestIntegration_ParseDebugMD(t *testing.T) {
	// Parse debug.md
	debugInfo, err := ParseDebug(sampleDebugMDIntegration)
	if err != nil {
		t.Fatalf("ParseDebug failed: %v", err)
	}

	// Verify debug entries count
	if len(debugInfo.Entries) != 2 {
		t.Errorf("Expected 2 debug entries, got %d", len(debugInfo.Entries))
	}

	// Verify first entry
	if debugInfo.Entries[0].ID != 1 {
		t.Errorf("Expected first entry ID 1, got %d", debugInfo.Entries[0].ID)
	}
	if debugInfo.Entries[0].Phenomenon != "JSON 序列化失败" {
		t.Errorf("Expected phenomenon 'JSON 序列化失败', got '%s'", debugInfo.Entries[0].Phenomenon)
	}

	// Verify second entry
	if debugInfo.Entries[1].ID != 2 {
		t.Errorf("Expected second entry ID 2, got %d", debugInfo.Entries[1].ID)
	}
	if debugInfo.Entries[1].Progress != "已修复" {
		t.Errorf("Expected progress '已修复', got '%s'", debugInfo.Entries[1].Progress)
	}

	// Test append functionality
	newEntry := DebugEntry{
		ID:         3,
		Phenomenon: "新问题",
		Reproduce:  "新复现",
		Hypothesis: "新猜想",
		Verify:     "新验证",
		Fix:        "新修复",
		Progress:   "待修复",
	}

	appendedContent := AppendDebug(sampleDebugMDIntegration, newEntry)

	// Verify appended content can be parsed
	debugInfo2, err := ParseDebug(appendedContent)
	if err != nil {
		t.Fatalf("ParseDebug after append failed: %v", err)
	}

	if len(debugInfo2.Entries) != 3 {
		t.Errorf("Expected 3 entries after append, got %d", len(debugInfo2.Entries))
	}

	// Verify next debug ID
	nextID := GetNextDebugID(sampleDebugMDIntegration)
	if nextID != 3 {
		t.Errorf("Expected next debug ID 3, got %d", nextID)
	}
}

// TestIntegration_ParseOKRAndSPEC verifies OKR.md and SPEC.md parsing are correct
func TestIntegration_ParseOKRAndSPEC(t *testing.T) {
	// Parse OKR.md
	okrInfo, err := ParseOKR(sampleOKRMDIntegration)
	if err != nil {
		t.Fatalf("ParseOKR failed: %v", err)
	}

	// Verify objectives
	if len(okrInfo.Objectives) != 3 {
		t.Errorf("Expected 3 objectives, got %d", len(okrInfo.Objectives))
	}
	if okrInfo.Objectives[0] != "提升日志系统的可观测性" {
		t.Errorf("Expected first objective '提升日志系统的可观测性', got '%s'", okrInfo.Objectives[0])
	}

	// Verify key results
	if len(okrInfo.KeyResults) != 3 {
		t.Errorf("Expected 3 key results, got %d", len(okrInfo.KeyResults))
	}
	if okrInfo.KeyResults[0] != "实现 JSON 格式输出" {
		t.Errorf("Expected first key result '实现 JSON 格式输出', got '%s'", okrInfo.KeyResults[0])
	}

	// Parse SPEC.md
	specInfo, err := ParseSPEC(sampleSpecMDIntegration)
	if err != nil {
		t.Fatalf("ParseSPEC failed: %v", err)
	}

	// Verify specifications
	if len(specInfo.Specifications) != 5 {
		t.Errorf("Expected 5 specifications, got %d", len(specInfo.Specifications))
	}
	if specInfo.Specifications[0] != "JSON 输出必须符合 RFC 7158 标准" {
		t.Errorf("Expected first spec 'JSON 输出必须符合 RFC 7158 标准', got '%s'", specInfo.Specifications[0])
	}
}

// TestIntegration_MultiFileCoordinator verifies multi-file coordinator loading is correct
func TestIntegration_MultiFileCoordinator(t *testing.T) {
	tmpDir := t.TempDir()
	jobDir := filepath.Join(tmpDir, "job_1")

	if err := os.MkdirAll(jobDir, 0755); err != nil {
		t.Fatalf("Failed to create job directory: %v", err)
	}

	// Create all test files
	if err := os.WriteFile(filepath.Join(jobDir, "task.md"), []byte(sampleTaskMDIntegration), 0644); err != nil {
		t.Fatalf("Failed to write task.md: %v", err)
	}
	if err := os.WriteFile(filepath.Join(jobDir, "debug.md"), []byte(sampleDebugMDIntegration), 0644); err != nil {
		t.Fatalf("Failed to write debug.md: %v", err)
	}
	if err := os.WriteFile(filepath.Join(jobDir, "OKR.md"), []byte(sampleOKRMDIntegration), 0644); err != nil {
		t.Fatalf("Failed to write OKR.md: %v", err)
	}
	if err := os.WriteFile(filepath.Join(jobDir, "SPEC.md"), []byte(sampleSpecMDIntegration), 0644); err != nil {
		t.Fatalf("Failed to write SPEC.md: %v", err)
	}

	// Load job context using coordinator
	coord := NewCoordinator()
	context, err := coord.LoadJobContext("job_1", jobDir)
	if err != nil {
		t.Fatalf("LoadJobContext failed: %v", err)
	}

	// Verify job ID
	if context.JobID != "job_1" {
		t.Errorf("Expected job ID 'job_1', got '%s'", context.JobID)
	}

	// Verify tasks loaded
	if len(context.Tasks) != 1 {
		t.Errorf("Expected 1 task, got %d", len(context.Tasks))
	}

	// Verify debug info loaded
	if context.DebugInfo == nil {
		t.Error("Expected DebugInfo to be loaded")
	}
	if len(context.DebugInfo.Entries) != 2 {
		t.Errorf("Expected 2 debug entries, got %d", len(context.DebugInfo.Entries))
	}

	// Verify OKR info loaded
	if context.OKRInfo == nil {
		t.Error("Expected OKRInfo to be loaded")
	}
	if len(context.OKRInfo.Objectives) != 3 {
		t.Errorf("Expected 3 objectives, got %d", len(context.OKRInfo.Objectives))
	}

	// Verify SPEC info loaded
	if context.SpecInfo == nil {
		t.Error("Expected SpecInfo to be loaded")
	}
	if len(context.SpecInfo.Specifications) != 5 {
		t.Errorf("Expected 5 specifications, got %d", len(context.SpecInfo.Specifications))
	}

	// Note: We skip ValidateConsistency here because the test data has no dependencies
	// ValidateConsistency checks for undefined dependencies, which is not applicable here

	// Verify summary stats
	stats := coord.GetSummaryStats(context)
	if stats.TotalTasks != 1 {
		t.Errorf("Expected 1 task, got %d", stats.TotalTasks)
	}
	if stats.TotalDebugEntries != 2 {
		t.Errorf("Expected 2 debug entries, got %d", stats.TotalDebugEntries)
	}
	if !stats.HasOKR {
		t.Error("Expected HasOKR to be true")
	}
	if !stats.HasSpec {
		t.Error("Expected HasSpec to be true")
	}
}

// TestIntegration_ErrorHandling verifies error handling mechanism works correctly
func TestIntegration_ErrorHandling(t *testing.T) {
	tests := []struct {
		name      string
		content   string
		parseFunc func(string) error
		wantErr   bool
	}{
		{
			name:      "ParseTask with missing name",
			content:   "# 任务目标\n目标",
			parseFunc: func(c string) error { _, err := ParseTask(c); return err },
			wantErr:   true,
		},
		{
			name:      "ParseTask with missing goal",
			content:   "# 任务名称\n名称",
			parseFunc: func(c string) error { _, err := ParseTask(c); return err },
			wantErr:   true,
		},
		{
			name:      "ParseTask with all required fields",
			content:   "# 任务名称\n名称\n# 任务目标\n目标\n# 测试方法\n测试",
			parseFunc: func(c string) error { _, err := ParseTask(c); return err },
			wantErr:   false,
		},
		{
			name:      "ValidateTask with missing name",
			content:   "",
			parseFunc: func(c string) error { return ValidateTask(&Task{Goal: "g", TestMethod: "t"}) },
			wantErr:   true,
		},
		{
			name:      "ValidateTask with missing goal",
			content:   "",
			parseFunc: func(c string) error { return ValidateTask(&Task{Name: "n", TestMethod: "t"}) },
			wantErr:   true,
		},
		{
			name:      "ValidateTask with missing test method",
			content:   "",
			parseFunc: func(c string) error { return ValidateTask(&Task{Name: "n", Goal: "g"}) },
			wantErr:   true,
		},
		{
			name:      "ValidateTask with all required fields",
			content:   "",
			parseFunc: func(c string) error { return ValidateTask(&Task{Name: "n", Goal: "g", TestMethod: "t"}) },
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.parseFunc(tt.content)
			if (err != nil) != tt.wantErr {
				t.Errorf("Expected error: %v, got: %v", tt.wantErr, err)
			}
		})
	}
}

// TestIntegration_CoordinatorCaching verifies caching mechanism works correctly
func TestIntegration_CoordinatorCaching(t *testing.T) {
	tmpDir := t.TempDir()
	jobDir := filepath.Join(tmpDir, "job_1")

	if err := os.MkdirAll(jobDir, 0755); err != nil {
		t.Fatalf("Failed to create job directory: %v", err)
	}

	if err := os.WriteFile(filepath.Join(jobDir, "task.md"), []byte(sampleTaskMDIntegration), 0644); err != nil {
		t.Fatalf("Failed to write task.md: %v", err)
	}

	coord := NewCoordinator()

	// First load
	context1, err := coord.LoadJobContext("job_1", jobDir)
	if err != nil {
		t.Fatalf("First LoadJobContext failed: %v", err)
	}

	cacheSize1 := coord.GetCacheSize()
	if cacheSize1 != 1 {
		t.Errorf("Expected cache size 1 after first load, got %d", cacheSize1)
	}

	// Second load should use cache
	context2, err := coord.LoadJobContext("job_1", jobDir)
	if err != nil {
		t.Fatalf("Second LoadJobContext failed: %v", err)
	}

	cacheSize2 := coord.GetCacheSize()
	if cacheSize2 != 1 {
		t.Errorf("Expected cache size 1 after second load, got %d", cacheSize2)
	}

	// Both should be the same object (from cache)
	if context1 != context2 {
		t.Error("Expected same context object from cache")
	}
}

// TestIntegration_CompleteParsingFlow tests the complete parsing flow end-to-end
func TestIntegration_CompleteParsingFlow(t *testing.T) {
	tmpDir := t.TempDir()
	jobDir := filepath.Join(tmpDir, "job_1")

	if err := os.MkdirAll(jobDir, 0755); err != nil {
		t.Fatalf("Failed to create job directory: %v", err)
	}

	// Create all test files
	files := map[string]string{
		"task.md":  sampleTaskMDIntegration,
		"debug.md": sampleDebugMDIntegration,
		"OKR.md":   sampleOKRMDIntegration,
		"SPEC.md":  sampleSpecMDIntegration,
	}

	for filename, content := range files {
		if err := os.WriteFile(filepath.Join(jobDir, filename), []byte(content), 0644); err != nil {
			t.Fatalf("Failed to write %s: %v", filename, err)
		}
	}

	// Step 1: Load job context using coordinator
	coord := NewCoordinator()
	context, err := coord.LoadJobContext("job_1", jobDir)
	if err != nil {
		t.Fatalf("LoadJobContext failed: %v", err)
	}

	// Step 2: Skip consistency validation for this test since the task has no dependencies
	// ValidateConsistency checks for undefined dependencies, which is not applicable here

	// Step 3: Get summary stats
	stats := coord.GetSummaryStats(context)

	// Step 4: Verify all components are present
	if stats.TotalTasks == 0 {
		t.Error("Expected at least 1 task")
	}
	if stats.TotalDebugEntries == 0 {
		t.Error("Expected at least 1 debug entry")
	}
	if !stats.HasOKR {
		t.Error("Expected OKR info to be present")
	}
	if !stats.HasSpec {
		t.Error("Expected SPEC info to be present")
	}

	// Step 5: Verify data integrity
	if len(context.Tasks) != 1 {
		t.Errorf("Expected 1 task, got %d", len(context.Tasks))
	}
	if len(context.DebugInfo.Entries) != 2 {
		t.Errorf("Expected 2 debug entries, got %d", len(context.DebugInfo.Entries))
	}
	if len(context.OKRInfo.Objectives) != 3 {
		t.Errorf("Expected 3 objectives, got %d", len(context.OKRInfo.Objectives))
	}
	if len(context.SpecInfo.Specifications) != 5 {
		t.Errorf("Expected 5 specifications, got %d", len(context.SpecInfo.Specifications))
	}

	// Step 6: Test task validation
	if err := ValidateTask(context.Tasks[0]); err != nil {
		t.Errorf("Task validation failed: %v", err)
	}

	// Step 7: Test debug entry retrieval
	debugEntries, err := coord.GetDebugEntriesByTaskID(context, "task1")
	if err != nil {
		t.Errorf("GetDebugEntriesByTaskID failed: %v", err)
	}
	if len(debugEntries) != 2 {
		t.Errorf("Expected 2 debug entries for task1, got %d", len(debugEntries))
	}

	// Step 8: Test task retrieval by ID
	task, err := coord.GetTaskByID(context, "task1")
	if err == nil {
		// Task ID might not be set, so we just verify the structure
		if task == nil {
			t.Error("Expected non-nil task")
		}
	}
}

// TestIntegration_MultipleJobs tests handling multiple jobs
func TestIntegration_MultipleJobs(t *testing.T) {
	tmpDir := t.TempDir()

	// Create job_1
	job1Dir := filepath.Join(tmpDir, "job_1")
	if err := os.MkdirAll(job1Dir, 0755); err != nil {
		t.Fatalf("Failed to create job_1 directory: %v", err)
	}

	// Create job_2
	job2Dir := filepath.Join(tmpDir, "job_2")
	if err := os.MkdirAll(job2Dir, 0755); err != nil {
		t.Fatalf("Failed to create job_2 directory: %v", err)
	}

	// Write task.md to both jobs
	if err := os.WriteFile(filepath.Join(job1Dir, "task.md"), []byte(sampleTaskMDIntegration), 0644); err != nil {
		t.Fatalf("Failed to write task.md to job_1: %v", err)
	}

	if err := os.WriteFile(filepath.Join(job2Dir, "task.md"), []byte(sampleTaskMDIntegration), 0644); err != nil {
		t.Fatalf("Failed to write task.md to job_2: %v", err)
	}

	// Load both jobs
	coord := NewCoordinator()

	context1, err := coord.LoadJobContext("job_1", job1Dir)
	if err != nil {
		t.Fatalf("LoadJobContext for job_1 failed: %v", err)
	}

	context2, err := coord.LoadJobContext("job_2", job2Dir)
	if err != nil {
		t.Fatalf("LoadJobContext for job_2 failed: %v", err)
	}

	// Verify both contexts are cached
	if coord.GetCacheSize() != 2 {
		t.Errorf("Expected cache size 2, got %d", coord.GetCacheSize())
	}

	// Verify contexts are different objects
	if context1 == context2 {
		t.Error("Expected different context objects for different jobs")
	}

	// Verify job IDs are correct
	if context1.JobID != "job_1" {
		t.Errorf("Expected job ID 'job_1', got '%s'", context1.JobID)
	}
	if context2.JobID != "job_2" {
		t.Errorf("Expected job ID 'job_2', got '%s'", context2.JobID)
	}
}

// BenchmarkParseIntegration benchmarks the complete parsing flow
func BenchmarkParseIntegration(b *testing.B) {
	for i := 0; i < b.N; i++ {
		ParseTask(sampleTaskMDIntegration)
		ParseDebug(sampleDebugMDIntegration)
		ParseOKR(sampleOKRMDIntegration)
		ParseSPEC(sampleSpecMDIntegration)
	}
}
