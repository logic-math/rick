package prompt

import (
	"os"
	"testing"

	"github.com/sunquan/rick/internal/parser"
)

func TestNewContextManager(t *testing.T) {
	jobID := "job_1"
	cm := NewContextManager(jobID)

	if cm == nil {
		t.Fatal("NewContextManager returned nil")
	}

	if cm.GetJobID() != jobID {
		t.Errorf("Expected jobID %s, got %s", jobID, cm.GetJobID())
	}

	if cm.IsTaskLoaded() {
		t.Error("Task should not be loaded initially")
	}

	if cm.HasDebugEntries() {
		t.Error("Debug entries should be empty initially")
	}

	if cm.HasOKRInfo() {
		t.Error("OKR info should be empty initially")
	}

	if cm.HasSPECInfo() {
		t.Error("SPEC info should be empty initially")
	}

	if cm.HasHistory() {
		t.Error("History should be empty initially")
	}
}

func TestLoadTask(t *testing.T) {
	cm := NewContextManager("job_1")

	task := &parser.Task{
		ID:         "task1",
		Name:       "Test Task",
		Goal:       "Test goal",
		KeyResults: []string{"result1", "result2"},
		TestMethod: "test method",
	}

	err := cm.LoadTask(task)
	if err != nil {
		t.Fatalf("LoadTask failed: %v", err)
	}

	if !cm.IsTaskLoaded() {
		t.Error("Task should be loaded")
	}

	loadedTask := cm.GetTask()
	if loadedTask.ID != task.ID {
		t.Errorf("Expected task ID %s, got %s", task.ID, loadedTask.ID)
	}

	if loadedTask.Name != task.Name {
		t.Errorf("Expected task name %s, got %s", task.Name, loadedTask.Name)
	}
}

func TestLoadTaskNil(t *testing.T) {
	cm := NewContextManager("job_1")

	err := cm.LoadTask(nil)
	if err == nil {
		t.Error("LoadTask should fail with nil task")
	}
}

func TestLoadDebugFromContent(t *testing.T) {
	cm := NewContextManager("job_1")

	debugContent := `**调试日志**:
- debug1: 问题1, 复现1, 猜想1, 验证1, 修复1, 进展1
- debug2: 问题2, 复现2, 猜想2, 验证2, 修复2, 进展2`

	err := cm.LoadDebugFromContent(debugContent)
	if err != nil {
		t.Fatalf("LoadDebugFromContent failed: %v", err)
	}

	if !cm.HasDebugEntries() {
		t.Error("Debug entries should not be empty")
	}

	debugInfo := cm.GetDebug()
	if len(debugInfo.Entries) != 2 {
		t.Errorf("Expected 2 debug entries, got %d", len(debugInfo.Entries))
	}

	if debugInfo.Entries[0].Phenomenon != "问题1" {
		t.Errorf("Expected phenomenon '问题1', got '%s'", debugInfo.Entries[0].Phenomenon)
	}
}

func TestLoadDebugFromFile(t *testing.T) {
	cm := NewContextManager("job_1")

	// Create temporary debug file
	tmpFile, err := os.CreateTemp("", "debug_*.md")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	debugContent := `**调试日志**:
- debug1: 问题1, 复现1, 猜想1, 验证1, 修复1, 进展1`

	if _, err := tmpFile.WriteString(debugContent); err != nil {
		t.Fatalf("Failed to write to temp file: %v", err)
	}
	tmpFile.Close()

	err = cm.LoadDebugFromFile(tmpFile.Name())
	if err != nil {
		t.Fatalf("LoadDebugFromFile failed: %v", err)
	}

	if !cm.HasDebugEntries() {
		t.Error("Debug entries should not be empty")
	}
}

func TestLoadDebugFromFileNotFound(t *testing.T) {
	cm := NewContextManager("job_1")

	err := cm.LoadDebugFromFile("/nonexistent/path/debug.md")
	if err == nil {
		t.Error("LoadDebugFromFile should fail for nonexistent file")
	}
}

func TestLoadOKRFromContent(t *testing.T) {
	cm := NewContextManager("job_1")

	okrContent := `# 目标
- 目标1
- 目标2

# 关键结果
- 结果1
- 结果2`

	err := cm.LoadOKRFromContent(okrContent)
	if err != nil {
		t.Fatalf("LoadOKRFromContent failed: %v", err)
	}

	if !cm.HasOKRInfo() {
		t.Error("OKR info should not be empty")
	}

	okrInfo := cm.GetOKRInfo()
	if len(okrInfo.Objectives) != 2 {
		t.Errorf("Expected 2 objectives, got %d", len(okrInfo.Objectives))
	}

	if len(okrInfo.KeyResults) != 2 {
		t.Errorf("Expected 2 key results, got %d", len(okrInfo.KeyResults))
	}
}

func TestLoadOKRFromFile(t *testing.T) {
	cm := NewContextManager("job_1")

	// Create temporary OKR file
	tmpFile, err := os.CreateTemp("", "okr_*.md")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	okrContent := `# 目标
- 目标1

# 关键结果
- 结果1`

	if _, err := tmpFile.WriteString(okrContent); err != nil {
		t.Fatalf("Failed to write to temp file: %v", err)
	}
	tmpFile.Close()

	err = cm.LoadOKRFromFile(tmpFile.Name())
	if err != nil {
		t.Fatalf("LoadOKRFromFile failed: %v", err)
	}

	if !cm.HasOKRInfo() {
		t.Error("OKR info should not be empty")
	}
}

func TestLoadSPECFromContent(t *testing.T) {
	cm := NewContextManager("job_1")

	specContent := `# 规范
- 规范1
- 规范2`

	err := cm.LoadSPECFromContent(specContent)
	if err != nil {
		t.Fatalf("LoadSPECFromContent failed: %v", err)
	}

	if !cm.HasSPECInfo() {
		t.Error("SPEC info should not be empty")
	}

	specInfo := cm.GetSPECInfo()
	if len(specInfo.Specifications) != 2 {
		t.Errorf("Expected 2 specifications, got %d", len(specInfo.Specifications))
	}
}

func TestLoadSPECFromFile(t *testing.T) {
	cm := NewContextManager("job_1")

	// Create temporary SPEC file
	tmpFile, err := os.CreateTemp("", "spec_*.md")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	specContent := `# 规范
- 规范1`

	if _, err := tmpFile.WriteString(specContent); err != nil {
		t.Fatalf("Failed to write to temp file: %v", err)
	}
	tmpFile.Close()

	err = cm.LoadSPECFromFile(tmpFile.Name())
	if err != nil {
		t.Fatalf("LoadSPECFromFile failed: %v", err)
	}

	if !cm.HasSPECInfo() {
		t.Error("SPEC info should not be empty")
	}
}

func TestLoadHistory(t *testing.T) {
	cm := NewContextManager("job_1")

	historyItems := []string{
		"commit1: 初始化项目",
		"commit2: 添加功能",
	}

	err := cm.LoadHistory(historyItems)
	if err != nil {
		t.Fatalf("LoadHistory failed: %v", err)
	}

	if !cm.HasHistory() {
		t.Error("History should not be empty")
	}

	history := cm.GetHistory()
	if len(history) != 2 {
		t.Errorf("Expected 2 history items, got %d", len(history))
	}

	if history[0] != "commit1: 初始化项目" {
		t.Errorf("Expected 'commit1: 初始化项目', got '%s'", history[0])
	}
}

func TestLoadHistoryNil(t *testing.T) {
	cm := NewContextManager("job_1")

	err := cm.LoadHistory(nil)
	if err != nil {
		t.Fatalf("LoadHistory should accept nil: %v", err)
	}

	if cm.HasHistory() {
		t.Error("History should be empty after loading nil")
	}
}

func TestLoadHistoryFromFile(t *testing.T) {
	cm := NewContextManager("job_1")

	// Create temporary history file
	tmpFile, err := os.CreateTemp("", "history_*.txt")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	historyContent := "commit1: 初始化项目\ncommit2: 添加功能"

	if _, err := tmpFile.WriteString(historyContent); err != nil {
		t.Fatalf("Failed to write to temp file: %v", err)
	}
	tmpFile.Close()

	err = cm.LoadHistoryFromFile(tmpFile.Name())
	if err != nil {
		t.Fatalf("LoadHistoryFromFile failed: %v", err)
	}

	if !cm.HasHistory() {
		t.Error("History should not be empty")
	}

	history := cm.GetHistory()
	if len(history) != 2 {
		t.Errorf("Expected 2 history items, got %d", len(history))
	}
}

func TestLoadHistoryFromFileNotFound(t *testing.T) {
	cm := NewContextManager("job_1")

	err := cm.LoadHistoryFromFile("/nonexistent/path/history.txt")
	if err == nil {
		t.Error("LoadHistoryFromFile should fail for nonexistent file")
	}
}

func TestContextManagerThreadSafety(t *testing.T) {
	cm := NewContextManager("job_1")

	task := &parser.Task{
		ID:   "task1",
		Name: "Test Task",
		Goal: "Test goal",
	}

	// Load task
	cm.LoadTask(task)

	// Concurrent reads should not panic
	done := make(chan bool)

	for i := 0; i < 10; i++ {
		go func() {
			cm.GetTask()
			cm.GetDebug()
			cm.GetOKRInfo()
			cm.GetSPECInfo()
			cm.GetHistory()
			done <- true
		}()
	}

	for i := 0; i < 10; i++ {
		<-done
	}
}

func TestGetHistoryCopy(t *testing.T) {
	cm := NewContextManager("job_1")

	historyItems := []string{"item1", "item2"}
	cm.LoadHistory(historyItems)

	history := cm.GetHistory()
	history[0] = "modified"

	// Original should not be modified
	history2 := cm.GetHistory()
	if history2[0] != "item1" {
		t.Error("GetHistory should return a copy, not the original")
	}
}

func TestEmptyContextOperations(t *testing.T) {
	cm := NewContextManager("job_1")

	// Test with empty content
	err := cm.LoadDebugFromContent("")
	if err != nil {
		t.Fatalf("LoadDebugFromContent should handle empty content: %v", err)
	}

	err = cm.LoadOKRFromContent("")
	if err != nil {
		t.Fatalf("LoadOKRFromContent should handle empty content: %v", err)
	}

	err = cm.LoadSPECFromContent("")
	if err != nil {
		t.Fatalf("LoadSPECFromContent should handle empty content: %v", err)
	}

	// All should still return false for Has* methods
	if cm.HasDebugEntries() {
		t.Error("HasDebugEntries should return false for empty content")
	}

	if cm.HasOKRInfo() {
		t.Error("HasOKRInfo should return false for empty content")
	}

	if cm.HasSPECInfo() {
		t.Error("HasSPECInfo should return false for empty content")
	}
}

func TestMultipleLoads(t *testing.T) {
	cm := NewContextManager("job_1")

	// Load task first time
	task1 := &parser.Task{
		ID:   "task1",
		Name: "Task 1",
		Goal: "Goal 1",
	}
	cm.LoadTask(task1)

	if cm.GetTask().Name != "Task 1" {
		t.Error("First load failed")
	}

	// Load task second time (should overwrite)
	task2 := &parser.Task{
		ID:   "task2",
		Name: "Task 2",
		Goal: "Goal 2",
	}
	cm.LoadTask(task2)

	if cm.GetTask().Name != "Task 2" {
		t.Error("Second load should overwrite first")
	}

	if cm.GetTask().ID != "task2" {
		t.Error("Task ID should be updated")
	}
}

func TestContextManagerGetters(t *testing.T) {
	cm := NewContextManager("job_1")

	task := &parser.Task{ID: "task1", Name: "Test"}
	cm.LoadTask(task)

	// Test all getters
	if cm.GetTask() == nil {
		t.Error("GetTask should not return nil")
	}

	if cm.GetDebug() == nil {
		t.Error("GetDebug should not return nil")
	}

	if cm.GetOKRInfo() == nil {
		t.Error("GetOKRInfo should not return nil")
	}

	if cm.GetSPECInfo() == nil {
		t.Error("GetSPECInfo should not return nil")
	}

	history := cm.GetHistory()
	if history == nil {
		t.Error("GetHistory should not return nil")
	}

	if len(history) != 0 {
		t.Error("GetHistory should return empty slice initially")
	}
}
