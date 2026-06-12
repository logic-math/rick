package parser

import "strings"

// ExtractBugFrontmatter parses YAML frontmatter (between --- markers) and extracts
// summary and status fields. Returns empty strings when frontmatter is absent or fields missing.
func ExtractBugFrontmatter(content string) (summary, status string) {
	lines := strings.Split(content, "\n")
	inFrontmatter := false
	started := false
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "---" {
			if !started {
				inFrontmatter = true
				started = true
				continue
			}
			if inFrontmatter {
				break
			}
		}
		if !inFrontmatter {
			continue
		}
		if strings.HasPrefix(trimmed, "summary:") {
			v := strings.TrimSpace(strings.TrimPrefix(trimmed, "summary:"))
			summary = strings.Trim(v, `"'`)
		} else if strings.HasPrefix(trimmed, "status:") {
			v := strings.TrimSpace(strings.TrimPrefix(trimmed, "status:"))
			status = strings.Trim(v, `"'`)
		}
	}
	return
}
