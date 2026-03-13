package parser

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNewCoordinator(t *testing.T) {
	coord := NewCoordinator()
	if coord == nil {
		t.Fatal("NewCoordinator returned nil")
	}
	if coord.GetCacheSize() != 0 {
		t.Errorf("expected cache size 0, got %d", coord.GetCacheSize())
	}
}

func TestLoadJobContext_WithTaskOnly(t *testing.T) {
	// Create temporary directory structure
	tmpDir := t.TempDir()
	jobDir := filepath.Join(tmpDir, "job_1")
	if err := os.MkdirAll(jobDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create task.md
	taskContent := `# 依赖关系
无

# 任务名称
测试任务

# 任务目标
实现测试功能

# 关键结果
- 结果1
- 结果2

# 测试方法
- 测试步骤1
- 测试步骤2
`
	taskPath := filepath.Join(jobDir, "task.md")
	if err := os.WriteFile(taskPath, []byte(taskContent), 0644); err != nil {
		t.Fatal(err)
	}

	coord := NewCoordinator()
	context, err := coord.LoadJobContext("job_1", jobDir)
	if err != nil {
		t.Fatalf("LoadJobContext failed: %v", err)
	}

	if context.JobID != "job_1" {
		t.Errorf("expected job ID job_1, got %s", context.JobID)
	}
	if len(context.Tasks) != 1 {
		t.Errorf("expected 1 task, got %d", len(context.Tasks))
	}
	if context.Tasks[0].Name != "测试任务" {
		t.Errorf("expected task name '测试任务', got '%s'", context.Tasks[0].Name)
	}
}

func TestLoadJobContext_WithAllFiles(t *testing.T) {
	tmpDir := t.TempDir()
	jobDir := filepath.Join(tmpDir, "job_1")
	if err := os.MkdirAll(jobDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create task.md
	taskContent := `# 依赖关系
无

# 任务名称
完整测试任务

# 任务目标
实现完整功能

# 关键结果
- 结果1

# 测试方法
- 测试步骤1
`
	if err := os.WriteFile(filepath.Join(jobDir, "task.md"), []byte(taskContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Create debug.md
	debugContent := `**调试日志**:
- debug1: 问题描述, 复现步骤, 猜想, 验证, 修复, 已修复
`
	if err := os.WriteFile(filepath.Join(jobDir, "debug.md"), []byte(debugContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Create OKR.md
	okrContent := `# 目标
- 目标1
- 目标2

# 关键结果
- KR1
- KR2
`
	if err := os.WriteFile(filepath.Join(jobDir, "OKR.md"), []byte(okrContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Create SPEC.md
	specContent := `# 规范
- 规范1
- 规范2
`
	if err := os.WriteFile(filepath.Join(jobDir, "SPEC.md"), []byte(specContent), 0644); err != nil {
		t.Fatal(err)
	}

	coord := NewCoordinator()
	context, err := coord.LoadJobContext("job_1", jobDir)
	if err != nil {
		t.Fatalf("LoadJobContext failed: %v", err)
	}

	if context.JobID != "job_1" {
		t.Errorf("expected job ID job_1, got %s", context.JobID)
	}
	if len(context.Tasks) != 1 {
		t.Errorf("expected 1 task, got %d", len(context.Tasks))
	}
	if context.DebugInfo == nil {
		t.Error("expected DebugInfo to be loaded")
	}
	if len(context.DebugInfo.Entries) != 1 {
		t.Errorf("expected 1 debug entry, got %d", len(context.DebugInfo.Entries))
	}
	if context.OKRInfo == nil {
		t.Error("expected OKRInfo to be loaded")
	}
	if len(context.OKRInfo.Objectives) != 2 {
		t.Errorf("expected 2 objectives, got %d", len(context.OKRInfo.Objectives))
	}
	if context.SpecInfo == nil {
		t.Error("expected SpecInfo to be loaded")
	}
	if len(context.SpecInfo.Specifications) != 2 {
		t.Errorf("expected 2 specifications, got %d", len(context.SpecInfo.Specifications))
	}
}

func TestLoadJobContext_Caching(t *testing.T) {
	tmpDir := t.TempDir()
	jobDir := filepath.Join(tmpDir, "job_1")
	if err := os.MkdirAll(jobDir, 0755); err != nil {
		t.Fatal(err)
	}

	taskContent := `# 依赖关系
无

# 任务名称
缓存测试

# 任务目标
测试缓存

# 关键结果
- 结果1

# 测试方法
- 测试步骤1
`
	if err := os.WriteFile(filepath.Join(jobDir, "task.md"), []byte(taskContent), 0644); err != nil {
		t.Fatal(err)
	}

	coord := NewCoordinator()

	// First load
	context1, err := coord.LoadJobContext("job_1", jobDir)
	if err != nil {
		t.Fatalf("first LoadJobContext failed: %v", err)
	}

	if coord.GetCacheSize() != 1 {
		t.Errorf("expected cache size 1 after first load, got %d", coord.GetCacheSize())
	}

	// Second load should use cache
	context2, err := coord.LoadJobContext("job_1", jobDir)
	if err != nil {
		t.Fatalf("second LoadJobContext failed: %v", err)
	}

	if coord.GetCacheSize() != 1 {
		t.Errorf("expected cache size 1 after second load, got %d", coord.GetCacheSize())
	}

	// Both should be the same object (from cache)
	if context1 != context2 {
		t.Error("expected same context object from cache")
	}

	// Test cache clearing
	coord.ClearCache("job_1")
	if coord.GetCacheSize() != 0 {
		t.Errorf("expected cache size 0 after clear, got %d", coord.GetCacheSize())
	}

	// Test clear all cache
	coord.LoadJobContext("job_1", jobDir)
	coord.LoadJobContext("job_2", jobDir)
	if coord.GetCacheSize() != 2 {
		t.Errorf("expected cache size 2, got %d", coord.GetCacheSize())
	}

	coord.ClearAllCache()
	if coord.GetCacheSize() != 0 {
		t.Errorf("expected cache size 0 after clear all, got %d", coord.GetCacheSize())
	}
}

func TestValidateConsistency_ValidContext(t *testing.T) {
	context := &JobContext{
		JobID: "job_1",
		Tasks: []*Task{
			{
				ID:           "task1",
				Name:         "Task 1",
				Goal:         "Goal 1",
				TestMethod:   "Test 1",
				Dependencies: []string{},
			},
			{
				ID:           "task2",
				Name:         "Task 2",
				Goal:         "Goal 2",
				TestMethod:   "Test 2",
				Dependencies: []string{"task1"},
			},
		},
	}

	coord := NewCoordinator()
	err := coord.ValidateConsistency(context)
	if err != nil {
		t.Errorf("ValidateConsistency failed for valid context: %v", err)
	}
}

func TestValidateConsistency_UndefinedDependency(t *testing.T) {
	context := &JobContext{
		JobID: "job_1",
		Tasks: []*Task{
			{
				ID:           "task1",
				Name:         "Task 1",
				Goal:         "Goal 1",
				TestMethod:   "Test 1",
				Dependencies: []string{"task_nonexistent"},
			},
		},
	}

	coord := NewCoordinator()
	err := coord.ValidateConsistency(context)
	if err == nil {
		t.Error("expected error for undefined dependency")
	}
}

func TestValidateConsistency_CircularDependency(t *testing.T) {
	context := &JobContext{
		JobID: "job_1",
		Tasks: []*Task{
			{
				ID:           "task1",
				Name:         "Task 1",
				Goal:         "Goal 1",
				TestMethod:   "Test 1",
				Dependencies: []string{"task2"},
			},
			{
				ID:           "task2",
				Name:         "Task 2",
				Goal:         "Goal 2",
				TestMethod:   "Test 2",
				Dependencies: []string{"task1"},
			},
		},
	}

	coord := NewCoordinator()
	err := coord.ValidateConsistency(context)
	if err == nil {
		t.Error("expected error for circular dependency")
	}
}

func TestValidateConsistency_EmptyContext(t *testing.T) {
	context := &JobContext{
		JobID: "job_1",
		Tasks: []*Task{},
	}

	coord := NewCoordinator()
	err := coord.ValidateConsistency(context)
	if err == nil {
		t.Error("expected error for empty context")
	}
}

func TestValidateConsistency_NilContext(t *testing.T) {
	coord := NewCoordinator()
	err := coord.ValidateConsistency(nil)
	if err == nil {
		t.Error("expected error for nil context")
	}
}

func TestMergeTasks_Success(t *testing.T) {
	tasks := []*Task{
		{
			ID:   "task1",
			Name: "Task 1",
		},
		{
			ID:   "task2",
			Name: "Task 2",
		},
	}

	coord := NewCoordinator()
	merged, err := coord.MergeTasks(tasks)
	if err != nil {
		t.Fatalf("MergeTasks failed: %v", err)
	}

	if len(merged) != 2 {
		t.Errorf("expected 2 tasks, got %d", len(merged))
	}
}

func TestMergeTasks_DuplicateID(t *testing.T) {
	tasks := []*Task{
		{
			ID:   "task1",
			Name: "Task 1",
		},
		{
			ID:   "task1",
			Name: "Task 1 Duplicate",
		},
	}

	coord := NewCoordinator()
	_, err := coord.MergeTasks(tasks)
	if err == nil {
		t.Error("expected error for duplicate task ID")
	}
}

func TestMergeTasks_Empty(t *testing.T) {
	coord := NewCoordinator()
	merged, err := coord.MergeTasks([]*Task{})
	if err != nil {
		t.Fatalf("MergeTasks failed: %v", err)
	}

	if len(merged) != 0 {
		t.Errorf("expected 0 tasks, got %d", len(merged))
	}
}

func TestGetTaskByID(t *testing.T) {
	context := &JobContext{
		JobID: "job_1",
		Tasks: []*Task{
			{
				ID:   "task1",
				Name: "Task 1",
			},
			{
				ID:   "task2",
				Name: "Task 2",
			},
		},
	}

	coord := NewCoordinator()

	task, err := coord.GetTaskByID(context, "task1")
	if err != nil {
		t.Fatalf("GetTaskByID failed: %v", err)
	}

	if task.ID != "task1" {
		t.Errorf("expected task ID task1, got %s", task.ID)
	}

	_, err = coord.GetTaskByID(context, "nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent task")
	}
}

func TestGetSummaryStats(t *testing.T) {
	context := &JobContext{
		JobID: "job_1",
		Tasks: []*Task{
			{
				ID:   "task1",
				Name: "Task 1",
			},
		},
		DebugInfo: &DebugInfo{
			Entries: []DebugEntry{
				{ID: 1, Phenomenon: "Issue 1"},
			},
		},
		OKRInfo: &ContextInfo{
			Objectives: []string{"Obj 1"},
		},
		SpecInfo: &ContextInfo{
			Specifications: []string{"Spec 1"},
		},
	}

	coord := NewCoordinator()
	stats := coord.GetSummaryStats(context)

	if stats.TotalTasks != 1 {
		t.Errorf("expected 1 task, got %d", stats.TotalTasks)
	}
	if stats.TotalDebugEntries != 1 {
		t.Errorf("expected 1 debug entry, got %d", stats.TotalDebugEntries)
	}
	if !stats.HasOKR {
		t.Error("expected HasOKR to be true")
	}
	if !stats.HasSpec {
		t.Error("expected HasSpec to be true")
	}
}

func TestGetDebugEntriesByTaskID(t *testing.T) {
	context := &JobContext{
		JobID: "job_1",
		DebugInfo: &DebugInfo{
			Entries: []DebugEntry{
				{ID: 1, Phenomenon: "Issue 1"},
				{ID: 2, Phenomenon: "Issue 2"},
			},
		},
	}

	coord := NewCoordinator()
	entries, err := coord.GetDebugEntriesByTaskID(context, "task1")
	if err != nil {
		t.Fatalf("GetDebugEntriesByTaskID failed: %v", err)
	}

	if len(entries) != 2 {
		t.Errorf("expected 2 debug entries, got %d", len(entries))
	}
}

func TestGetDebugEntriesByTaskID_NoDebugInfo(t *testing.T) {
	context := &JobContext{
		JobID: "job_1",
	}

	coord := NewCoordinator()
	entries, err := coord.GetDebugEntriesByTaskID(context, "task1")
	if err != nil {
		t.Fatalf("GetDebugEntriesByTaskID failed: %v", err)
	}

	if len(entries) != 0 {
		t.Errorf("expected 0 debug entries, got %d", len(entries))
	}
}

func TestLoadJobContext_MissingFiles(t *testing.T) {
	tmpDir := t.TempDir()
	jobDir := filepath.Join(tmpDir, "job_1")
	if err := os.MkdirAll(jobDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create only task.md, other files missing
	taskContent := `# 依赖关系
无

# 任务名称
测试任务

# 任务目标
实现测试功能

# 关键结果
- 结果1

# 测试方法
- 测试步骤1
`
	if err := os.WriteFile(filepath.Join(jobDir, "task.md"), []byte(taskContent), 0644); err != nil {
		t.Fatal(err)
	}

	coord := NewCoordinator()
	context, err := coord.LoadJobContext("job_1", jobDir)
	if err != nil {
		t.Fatalf("LoadJobContext failed: %v", err)
	}

	// Should load successfully with only task.md
	if len(context.Tasks) != 1 {
		t.Errorf("expected 1 task, got %d", len(context.Tasks))
	}
	if context.DebugInfo != nil {
		t.Error("expected DebugInfo to be nil")
	}
	if context.OKRInfo != nil {
		t.Error("expected OKRInfo to be nil")
	}
	if context.SpecInfo != nil {
		t.Error("expected SpecInfo to be nil")
	}
}

func TestCircularDependencies_SelfReference(t *testing.T) {
	context := &JobContext{
		JobID: "job_1",
		Tasks: []*Task{
			{
				ID:           "task1",
				Name:         "Task 1",
				Goal:         "Goal 1",
				TestMethod:   "Test 1",
				Dependencies: []string{"task1"},
			},
		},
	}

	coord := NewCoordinator()
	err := coord.ValidateConsistency(context)
	if err == nil {
		t.Error("expected error for self-referencing dependency")
	}
}

func TestCircularDependencies_LongChain(t *testing.T) {
	context := &JobContext{
		JobID: "job_1",
		Tasks: []*Task{
			{
				ID:           "task1",
				Name:         "Task 1",
				Goal:         "Goal 1",
				TestMethod:   "Test 1",
				Dependencies: []string{"task2"},
			},
			{
				ID:           "task2",
				Name:         "Task 2",
				Goal:         "Goal 2",
				TestMethod:   "Test 2",
				Dependencies: []string{"task3"},
			},
			{
				ID:           "task3",
				Name:         "Task 3",
				Goal:         "Goal 3",
				TestMethod:   "Test 3",
				Dependencies: []string{"task1"},
			},
		},
	}

	coord := NewCoordinator()
	err := coord.ValidateConsistency(context)
	if err == nil {
		t.Error("expected error for circular dependency chain")
	}
}
