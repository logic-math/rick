package executor

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/sunquan/rick/internal/parser"
)

// LoadDebugDirSummaries scans {workspaceDir}/debug/, reads bug*.md in lexicographic order,
// and returns a multi-line string of frontmatter summaries.
func LoadDebugDirSummaries(workspaceDir string) string {
	if workspaceDir == "" {
		return ""
	}
	debugDir := filepath.Join(workspaceDir, "debug")
	entries, err := os.ReadDir(debugDir)
	if err != nil {
		return ""
	}

	var files []string
	for _, e := range entries {
		name := e.Name()
		if !e.IsDir() && strings.HasPrefix(name, "bug") && strings.HasSuffix(name, ".md") {
			files = append(files, name)
		}
	}
	if len(files) == 0 {
		return ""
	}
	sort.Strings(files)

	var sb strings.Builder
	for _, name := range files {
		data, err := os.ReadFile(filepath.Join(debugDir, name))
		if err != nil {
			continue
		}
		summary, status := parser.ExtractBugFrontmatter(string(data))
		sb.WriteString(fmt.Sprintf("- [%s] summary: %s | status: %s\n", name, summary, status))
	}
	return sb.String()
}

// LoadDebugContext is the unified entry point for all debug context loading.
// Prefers bug*.md frontmatter summaries from {workspaceDir}/debug/;
// falls back to {workspaceDir}/debug.md when debug/ is empty or absent.
// Returns empty string (not panic) when workspaceDir is empty or does not exist.
//
// TODO(2026-08): remove fallback to debug.md after full migration to debug/ dir format.
func LoadDebugContext(workspaceDir string) string {
	if workspaceDir == "" {
		return ""
	}
	if _, err := os.Stat(workspaceDir); os.IsNotExist(err) {
		return ""
	}

	summaries := LoadDebugDirSummaries(workspaceDir)
	if summaries != "" {
		return summaries
	}

	// TODO(2026-08): remove fallback after full migration
	data, err := os.ReadFile(filepath.Join(workspaceDir, "debug.md"))
	if err != nil {
		return ""
	}
	return string(data)
}
