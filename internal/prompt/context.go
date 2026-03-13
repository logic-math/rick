package prompt

import (
	"fmt"
	"os"
	"strings"
	"sync"

	"github.com/sunquan/rick/internal/parser"
)

// ContextManager manages execution context from various sources (task, debug, OKR, SPEC, history)
type ContextManager struct {
	jobID      string
	Task       *parser.Task
	Debug      *parser.DebugInfo
	OKRInfo    *parser.ContextInfo
	SPECInfo   *parser.ContextInfo
	History    []string
	mu         sync.RWMutex
}

// NewContextManager creates a new ContextManager instance for the given jobID
func NewContextManager(jobID string) *ContextManager {
	return &ContextManager{
		jobID:    jobID,
		Task:     nil,
		Debug:    &parser.DebugInfo{Entries: []parser.DebugEntry{}},
		OKRInfo:  &parser.ContextInfo{},
		SPECInfo: &parser.ContextInfo{},
		History:  []string{},
	}
}

// LoadTask loads task information from a Task struct
func (cm *ContextManager) LoadTask(task *parser.Task) error {
	if task == nil {
		return fmt.Errorf("task cannot be nil")
	}

	cm.mu.Lock()
	defer cm.mu.Unlock()

	cm.Task = task
	return nil
}

// LoadDebugFromContent loads debug information from debug.md content
func (cm *ContextManager) LoadDebugFromContent(debugContent string) error {
	debugInfo, err := parser.ParseDebug(debugContent)
	if err != nil {
		return fmt.Errorf("failed to parse debug content: %w", err)
	}

	cm.mu.Lock()
	defer cm.mu.Unlock()

	cm.Debug = debugInfo
	return nil
}

// LoadDebugFromFile loads debug information from a debug.md file
func (cm *ContextManager) LoadDebugFromFile(filePath string) error {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read debug file: %w", err)
	}

	return cm.LoadDebugFromContent(string(content))
}

// LoadOKRFromContent loads OKR information from OKR.md content
func (cm *ContextManager) LoadOKRFromContent(okrContent string) error {
	okrInfo, err := parser.ParseOKR(okrContent)
	if err != nil {
		return fmt.Errorf("failed to parse OKR content: %w", err)
	}

	cm.mu.Lock()
	defer cm.mu.Unlock()

	cm.OKRInfo = okrInfo
	return nil
}

// LoadOKRFromFile loads OKR information from an OKR.md file
func (cm *ContextManager) LoadOKRFromFile(filePath string) error {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read OKR file: %w", err)
	}

	return cm.LoadOKRFromContent(string(content))
}

// LoadSPECFromContent loads SPEC information from SPEC.md content
func (cm *ContextManager) LoadSPECFromContent(specContent string) error {
	specInfo, err := parser.ParseSPEC(specContent)
	if err != nil {
		return fmt.Errorf("failed to parse SPEC content: %w", err)
	}

	cm.mu.Lock()
	defer cm.mu.Unlock()

	cm.SPECInfo = specInfo
	return nil
}

// LoadSPECFromFile loads SPEC information from a SPEC.md file
func (cm *ContextManager) LoadSPECFromFile(filePath string) error {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read SPEC file: %w", err)
	}

	return cm.LoadSPECFromContent(string(content))
}

// LoadHistory loads execution history from git or other sources
// For now, it accepts a slice of history strings
func (cm *ContextManager) LoadHistory(historyItems []string) error {
	if historyItems == nil {
		historyItems = []string{}
	}

	cm.mu.Lock()
	defer cm.mu.Unlock()

	cm.History = historyItems
	return nil
}

// LoadHistoryFromFile loads execution history from a file
func (cm *ContextManager) LoadHistoryFromFile(filePath string) error {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read history file: %w", err)
	}

	// Parse history file - one line per history item
	historyItems := []string{}
	if len(content) > 0 {
		// Split by newlines and filter empty lines
		lines := strings.Split(string(content), "\n")
		for _, line := range lines {
			trimmed := strings.TrimSpace(line)
			if trimmed != "" {
				historyItems = append(historyItems, trimmed)
			}
		}
	}

	return cm.LoadHistory(historyItems)
}

// GetTask returns the loaded task information
func (cm *ContextManager) GetTask() *parser.Task {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	return cm.Task
}

// GetDebug returns the loaded debug information
func (cm *ContextManager) GetDebug() *parser.DebugInfo {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	return cm.Debug
}

// GetOKRInfo returns the loaded OKR information
func (cm *ContextManager) GetOKRInfo() *parser.ContextInfo {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	return cm.OKRInfo
}

// GetSPECInfo returns the loaded SPEC information
func (cm *ContextManager) GetSPECInfo() *parser.ContextInfo {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	return cm.SPECInfo
}

// GetHistory returns the loaded history items
func (cm *ContextManager) GetHistory() []string {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	// Return a copy to prevent external modification
	result := make([]string, len(cm.History))
	copy(result, cm.History)
	return result
}

// GetJobID returns the job ID
func (cm *ContextManager) GetJobID() string {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	return cm.jobID
}

// IsTaskLoaded returns true if task information has been loaded
func (cm *ContextManager) IsTaskLoaded() bool {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	return cm.Task != nil
}

// HasDebugEntries returns true if there are debug entries
func (cm *ContextManager) HasDebugEntries() bool {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	return len(cm.Debug.Entries) > 0
}

// HasOKRInfo returns true if OKR information has been loaded
func (cm *ContextManager) HasOKRInfo() bool {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	return len(cm.OKRInfo.Objectives) > 0 || len(cm.OKRInfo.KeyResults) > 0
}

// HasSPECInfo returns true if SPEC information has been loaded
func (cm *ContextManager) HasSPECInfo() bool {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	return len(cm.SPECInfo.Specifications) > 0
}

// HasHistory returns true if history has been loaded
func (cm *ContextManager) HasHistory() bool {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	return len(cm.History) > 0
}
