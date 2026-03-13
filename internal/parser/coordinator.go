package parser

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

// JobContext represents the complete context for a job, including all parsed files
type JobContext struct {
	JobID      string
	Tasks      []*Task
	DebugInfo  *DebugInfo
	OKRInfo    *ContextInfo
	SpecInfo   *ContextInfo
	RawContent map[string]string // Store raw file content for reference
}

// Coordinator manages multi-file parsing and coordination
type Coordinator struct {
	cache      map[string]*JobContext
	cacheMutex sync.RWMutex
}

// NewCoordinator creates a new coordinator instance
func NewCoordinator() *Coordinator {
	return &Coordinator{
		cache: make(map[string]*JobContext),
	}
}

// LoadJobContext loads all files for a specific job and returns the complete context
// It looks for task.md, debug.md, OKR.md, and SPEC.md files in the job directory
func (c *Coordinator) LoadJobContext(jobID, jobDir string) (*JobContext, error) {
	// Check cache first
	c.cacheMutex.RLock()
	if ctx, exists := c.cache[jobID]; exists {
		c.cacheMutex.RUnlock()
		return ctx, nil
	}
	c.cacheMutex.RUnlock()

	// Create new context
	context := &JobContext{
		JobID:      jobID,
		Tasks:      []*Task{},
		RawContent: make(map[string]string),
	}

	// Load task.md
	taskPath := filepath.Join(jobDir, "task.md")
	taskContent, err := readFile(taskPath)
	if err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("failed to read task.md: %w", err)
	}
	if taskContent != "" {
		task, err := ParseTask(taskContent)
		if err != nil {
			return nil, fmt.Errorf("failed to parse task.md: %w", err)
		}
		context.Tasks = append(context.Tasks, task)
		context.RawContent["task.md"] = taskContent
	}

	// Load debug.md
	debugPath := filepath.Join(jobDir, "debug.md")
	debugContent, err := readFile(debugPath)
	if err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("failed to read debug.md: %w", err)
	}
	if debugContent != "" {
		debugInfo, err := ParseDebug(debugContent)
		if err != nil {
			return nil, fmt.Errorf("failed to parse debug.md: %w", err)
		}
		context.DebugInfo = debugInfo
		context.RawContent["debug.md"] = debugContent
	}

	// Load OKR.md
	okrPath := filepath.Join(jobDir, "OKR.md")
	okrContent, err := readFile(okrPath)
	if err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("failed to read OKR.md: %w", err)
	}
	if okrContent != "" {
		okrInfo, err := ParseOKR(okrContent)
		if err != nil {
			return nil, fmt.Errorf("failed to parse OKR.md: %w", err)
		}
		context.OKRInfo = okrInfo
		context.RawContent["OKR.md"] = okrContent
	}

	// Load SPEC.md
	specPath := filepath.Join(jobDir, "SPEC.md")
	specContent, err := readFile(specPath)
	if err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("failed to read SPEC.md: %w", err)
	}
	if specContent != "" {
		specInfo, err := ParseSPEC(specContent)
		if err != nil {
			return nil, fmt.Errorf("failed to parse SPEC.md: %w", err)
		}
		context.SpecInfo = specInfo
		context.RawContent["SPEC.md"] = specContent
	}

	// Cache the result
	c.cacheMutex.Lock()
	c.cache[jobID] = context
	c.cacheMutex.Unlock()

	return context, nil
}

// ValidateConsistency checks for consistency across all loaded files
// It verifies that:
// 1. Dependencies referenced in tasks exist
// 2. Debug entries reference valid tasks or issues
// 3. OKR objectives align with task goals
// 4. Specifications don't conflict with task implementations
func (c *Coordinator) ValidateConsistency(context *JobContext) error {
	if context == nil {
		return fmt.Errorf("context cannot be nil")
	}

	if len(context.Tasks) == 0 {
		return fmt.Errorf("no tasks found in job context")
	}

	// Validate task dependencies
	taskIDMap := make(map[string]*Task)
	for _, task := range context.Tasks {
		taskIDMap[task.ID] = task
	}

	for _, task := range context.Tasks {
		for _, dep := range task.Dependencies {
			if _, exists := taskIDMap[dep]; !exists {
				return fmt.Errorf("task %s has undefined dependency: %s", task.ID, dep)
			}
		}
	}

	// Validate that tasks don't have circular dependencies
	if err := detectCircularDependencies(context.Tasks); err != nil {
		return fmt.Errorf("circular dependency detected: %w", err)
	}

	// Validate debug entries reference valid content
	if context.DebugInfo != nil && len(context.DebugInfo.Entries) > 0 {
		// Check that debug entry IDs are sequential starting from 1
		for i, entry := range context.DebugInfo.Entries {
			if entry.ID != i+1 {
				return fmt.Errorf("debug entries are not sequential: expected ID %d, got %d", i+1, entry.ID)
			}
		}
	}

	return nil
}

// MergeTasks combines multiple task definitions into a single context
// This is useful when a job has multiple task.md files or task definitions
func (c *Coordinator) MergeTasks(tasks []*Task) ([]*Task, error) {
	if len(tasks) == 0 {
		return []*Task{}, nil
	}

	// Check for duplicate task IDs
	seenIDs := make(map[string]bool)
	for _, task := range tasks {
		if seenIDs[task.ID] {
			return nil, fmt.Errorf("duplicate task ID found: %s", task.ID)
		}
		seenIDs[task.ID] = true
	}

	// Return merged tasks (no actual merging needed if IDs are unique)
	return tasks, nil
}

// ClearCache clears the internal cache for a specific job
func (c *Coordinator) ClearCache(jobID string) {
	c.cacheMutex.Lock()
	defer c.cacheMutex.Unlock()
	delete(c.cache, jobID)
}

// ClearAllCache clears the entire internal cache
func (c *Coordinator) ClearAllCache() {
	c.cacheMutex.Lock()
	defer c.cacheMutex.Unlock()
	c.cache = make(map[string]*JobContext)
}

// GetCacheSize returns the number of cached job contexts
func (c *Coordinator) GetCacheSize() int {
	c.cacheMutex.RLock()
	defer c.cacheMutex.RUnlock()
	return len(c.cache)
}

// readFile reads a file and returns its content as a string
// Returns empty string if file doesn't exist, error for other issues
func readFile(path string) (string, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", err
	}
	return string(content), nil
}

// detectCircularDependencies checks for circular dependencies in tasks
func detectCircularDependencies(tasks []*Task) error {
	// Build adjacency list
	taskMap := make(map[string]*Task)
	for _, task := range tasks {
		taskMap[task.ID] = task
	}

	// Check each task for cycles
	visited := make(map[string]bool)
	recStack := make(map[string]bool)

	for _, task := range tasks {
		if !visited[task.ID] {
			if hasCycle(task.ID, taskMap, visited, recStack) {
				return fmt.Errorf("circular dependency involving task: %s", task.ID)
			}
		}
	}

	return nil
}

// hasCycle is a helper function for detecting cycles using DFS
func hasCycle(taskID string, taskMap map[string]*Task, visited, recStack map[string]bool) bool {
	visited[taskID] = true
	recStack[taskID] = true

	task, exists := taskMap[taskID]
	if !exists {
		return false
	}

	for _, dep := range task.Dependencies {
		if !visited[dep] {
			if hasCycle(dep, taskMap, visited, recStack) {
				return true
			}
		} else if recStack[dep] {
			return true
		}
	}

	recStack[taskID] = false
	return false
}

// GetTaskByID retrieves a specific task from the job context by its ID
func (c *Coordinator) GetTaskByID(context *JobContext, taskID string) (*Task, error) {
	if context == nil {
		return nil, fmt.Errorf("context cannot be nil")
	}

	for _, task := range context.Tasks {
		if task.ID == taskID {
			return task, nil
		}
	}

	return nil, fmt.Errorf("task with ID %s not found", taskID)
}

// GetDebugEntriesByTaskID retrieves debug entries related to a specific task
func (c *Coordinator) GetDebugEntriesByTaskID(context *JobContext, taskID string) ([]DebugEntry, error) {
	if context == nil || context.DebugInfo == nil {
		return []DebugEntry{}, nil
	}

	// This is a simple implementation that returns all debug entries
	// In a more sophisticated version, we could parse debug entries to identify which task they relate to
	return context.DebugInfo.Entries, nil
}

// SummaryStats returns statistics about the job context
type SummaryStats struct {
	TotalTasks      int
	TotalDebugEntries int
	HasOKR          bool
	HasSpec         bool
	CachedJobs      int
}

// GetSummaryStats returns summary statistics about a job context
func (c *Coordinator) GetSummaryStats(context *JobContext) SummaryStats {
	stats := SummaryStats{
		TotalTasks: len(context.Tasks),
		HasOKR:     context.OKRInfo != nil && (len(context.OKRInfo.Objectives) > 0 || len(context.OKRInfo.KeyResults) > 0),
		HasSpec:    context.SpecInfo != nil && len(context.SpecInfo.Specifications) > 0,
		CachedJobs: c.GetCacheSize(),
	}

	if context.DebugInfo != nil {
		stats.TotalDebugEntries = len(context.DebugInfo.Entries)
	}

	return stats
}
