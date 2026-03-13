package parser

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// DebugEntry represents a single debug record
type DebugEntry struct {
	ID         int    // debug1, debug2, ...
	Phenomenon string // 现象
	Reproduce  string // 复现
	Hypothesis string // 猜想
	Verify     string // 验证
	Fix        string // 修复
	Progress   string // 进展
}

// DebugInfo represents a collection of debug entries
type DebugInfo struct {
	Entries []DebugEntry
}

// ParseDebug parses debug.md content and returns DebugInfo
func ParseDebug(content string) (*DebugInfo, error) {
	debugInfo := &DebugInfo{
		Entries: []DebugEntry{},
	}

	if content == "" {
		return debugInfo, nil
	}

	// Parse line by line to find debug entries
	lines := strings.Split(content, "\n")
	pattern := regexp.MustCompile(`^\s*-\s*debug(\d+):\s*(.+)$`)

	for _, line := range lines {
		matches := pattern.FindStringSubmatch(line)
		if len(matches) == 3 {
			idStr := matches[1]
			entryText := matches[2]

			id, err := strconv.Atoi(idStr)
			if err != nil {
				continue
			}

			// Parse the entry text (comma-separated values)
			entry := parseDebugEntryText(id, entryText)
			debugInfo.Entries = append(debugInfo.Entries, entry)
		}
	}

	return debugInfo, nil
}

// parseDebugEntryText parses the comma-separated debug entry text
func parseDebugEntryText(id int, text string) DebugEntry {
	entry := DebugEntry{
		ID: id,
	}

	// Split by comma and trim whitespace
	parts := strings.Split(text, ",")
	if len(parts) > 0 {
		entry.Phenomenon = strings.TrimSpace(parts[0])
	}
	if len(parts) > 1 {
		entry.Reproduce = strings.TrimSpace(parts[1])
	}
	if len(parts) > 2 {
		entry.Hypothesis = strings.TrimSpace(parts[2])
	}
	if len(parts) > 3 {
		entry.Verify = strings.TrimSpace(parts[3])
	}
	if len(parts) > 4 {
		entry.Fix = strings.TrimSpace(parts[4])
	}
	if len(parts) > 5 {
		entry.Progress = strings.TrimSpace(parts[5])
	}

	return entry
}

// AppendDebug appends a new debug entry to the content
func AppendDebug(content string, entry DebugEntry) string {
	// Generate the debug entry line
	entryLine := GenerateDebugEntry(entry.ID, entry.Phenomenon, entry.Reproduce,
		entry.Hypothesis, entry.Verify, entry.Fix, entry.Progress)

	// If content is empty, start with the entry
	if strings.TrimSpace(content) == "" {
		return "**调试日志**:\n" + entryLine
	}

	// Find the debug log section
	if strings.Contains(content, "**调试日志**") {
		// Append to the existing debug log section
		return content + "\n" + entryLine
	}

	// If no debug log section, add it
	return content + "\n\n**调试日志**:\n" + entryLine
}

// GetDebugCount returns the number of debug entries in the content
func GetDebugCount(content string) int {
	debugInfo, err := ParseDebug(content)
	if err != nil {
		return 0
	}
	return len(debugInfo.Entries)
}

// GenerateDebugEntry generates a debug entry line in the format:
// - debugN: phenomenon, reproduce, hypothesis, verify, fix, progress
func GenerateDebugEntry(id int, phenomenon, reproduce, hypothesis, verify, fix, progress string) string {
	return fmt.Sprintf("- debug%d: %s, %s, %s, %s, %s, %s",
		id, phenomenon, reproduce, hypothesis, verify, fix, progress)
}

// GetNextDebugID returns the next available debug ID based on existing entries
func GetNextDebugID(content string) int {
	debugInfo, err := ParseDebug(content)
	if err != nil || len(debugInfo.Entries) == 0 {
		return 1
	}

	// Find the maximum ID and return ID+1
	maxID := 0
	for _, entry := range debugInfo.Entries {
		if entry.ID > maxID {
			maxID = entry.ID
		}
	}

	return maxID + 1
}

// GetDebugByID retrieves a debug entry by its ID
func GetDebugByID(content string, id int) (*DebugEntry, error) {
	debugInfo, err := ParseDebug(content)
	if err != nil {
		return nil, err
	}

	for _, entry := range debugInfo.Entries {
		if entry.ID == id {
			return &entry, nil
		}
	}

	return nil, fmt.Errorf("debug entry with ID %d not found", id)
}

// UpdateDebugEntry updates an existing debug entry
func UpdateDebugEntry(content string, id int, phenomenon, reproduce, hypothesis, verify, fix, progress string) (string, error) {
	debugInfo, err := ParseDebug(content)
	if err != nil {
		return content, err
	}

	// Find and update the entry
	found := false
	for i, entry := range debugInfo.Entries {
		if entry.ID == id {
			debugInfo.Entries[i] = DebugEntry{
				ID:         id,
				Phenomenon: phenomenon,
				Reproduce:  reproduce,
				Hypothesis: hypothesis,
				Verify:     verify,
				Fix:        fix,
				Progress:   progress,
			}
			found = true
			break
		}
	}

	if !found {
		return content, fmt.Errorf("debug entry with ID %d not found", id)
	}

	// Reconstruct the debug log section
	debugLogSection := "**调试日志**:\n"
	for _, entry := range debugInfo.Entries {
		debugLogSection += GenerateDebugEntry(entry.ID, entry.Phenomenon, entry.Reproduce,
			entry.Hypothesis, entry.Verify, entry.Fix, entry.Progress) + "\n"
	}
	debugLogSection = strings.TrimSuffix(debugLogSection, "\n")

	// Replace the debug log section in content
	if strings.Contains(content, "**调试日志**") {
		// Find the start and end of the debug log section
		startIdx := strings.Index(content, "**调试日志**")
		// Find the next section or end of content
		endIdx := len(content)
		nextSectionIdx := strings.Index(content[startIdx+1:], "\n**")
		if nextSectionIdx != -1 {
			endIdx = startIdx + 1 + nextSectionIdx
		}

		return content[:startIdx] + debugLogSection + content[endIdx:], nil
	}

	return content + "\n\n" + debugLogSection, nil
}
