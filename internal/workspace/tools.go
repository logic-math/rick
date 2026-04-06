package workspace

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ToolInfo holds metadata for a single tool script
type ToolInfo struct {
	Path        string // absolute path to the .py file
	Name        string // filename without extension
	Description string // content of the "# Description: ..." first-line comment
}

// LoadToolsList scans projectRoot/tools/*.py and extracts description from the first line.
// Returns an empty slice (not error) when the tools directory doesn't exist or is empty.
func LoadToolsList(projectRoot string) ([]ToolInfo, error) {
	toolsDir := filepath.Join(projectRoot, "tools")

	info, err := os.Stat(toolsDir)
	if err != nil || !info.IsDir() {
		return nil, nil
	}

	entries, err := os.ReadDir(toolsDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read tools directory: %w", err)
	}

	var tools []ToolInfo
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".py") {
			continue
		}

		filePath := filepath.Join(toolsDir, e.Name())
		desc := extractToolDescription(filePath)
		tools = append(tools, ToolInfo{
			Path:        filePath,
			Name:        strings.TrimSuffix(e.Name(), ".py"),
			Description: desc,
		})
	}

	return tools, nil
}

// extractToolDescription reads the first line of a .py file and extracts the description
// from a "# Description: ..." comment. Returns empty string if not found.
func extractToolDescription(filePath string) string {
	f, err := os.Open(filePath)
	if err != nil {
		return ""
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	if scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "# Description:") {
			return strings.TrimSpace(strings.TrimPrefix(line, "# Description:"))
		}
	}
	return ""
}
