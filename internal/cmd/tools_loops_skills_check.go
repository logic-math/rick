package cmd

import (
	"os"
	"path/filepath"
	"strings"
)

// runLoopsAndSkillsCheck validates .rick/loops/*.md and .rick/skills/*.md format.
// Returns a list of error strings (empty = pass). Skips README.md in each dir.
// Directories that don't exist are silently skipped.
func runLoopsAndSkillsCheck(rickDir string) []string {
	var errs []string

	loopSections := []string{
		"## 目标",
		"## 上下文管理",
		"## 可调用工具",
		"## 产出评估",
		"## 停止标准",
	}
	errs = append(errs, checkMarkdownDir(rickDir, "loops", []string{"name", "trigger"}, loopSections)...)

	skillSections := []string{
		"## When to Use",
		"## Procedure",
		"## Pitfalls",
		"## Verification",
	}
	errs = append(errs, checkMarkdownDir(rickDir, "skills", []string{"name", "description"}, skillSections)...)

	return errs
}

// checkMarkdownDir scans dirName under rickDir for *.md files (excluding README.md),
// validates frontmatter fields and body sections, and returns error strings.
func checkMarkdownDir(rickDir, dirName string, requiredFields, requiredSections []string) []string {
	dir := filepath.Join(rickDir, dirName)
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return nil
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return []string{dirName + ": failed to read directory: " + err.Error()}
	}

	var errs []string
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
			continue
		}
		if filepath.Base(e.Name()) == "README.md" {
			continue
		}

		filePath := filepath.Join(dir, e.Name())
		raw, err := os.ReadFile(filePath)
		if err != nil {
			errs = append(errs, dirName+"/"+e.Name()+": failed to read file: "+err.Error())
			continue
		}
		content := string(raw)
		prefix := dirName + "/" + e.Name()
		errs = append(errs, checkMarkdownFile(prefix, content, requiredFields, requiredSections)...)
	}
	return errs
}

// checkMarkdownFile validates frontmatter and body sections for a single file.
func checkMarkdownFile(prefix, content string, requiredFields, requiredSections []string) []string {
	var errs []string

	fm, body, ok := parseFrontmatter(content)
	if !ok {
		errs = append(errs, prefix+": missing frontmatter (file must start with ---)")
		// Still check sections on the full content so all errors are reported
		for _, section := range requiredSections {
			if !strings.Contains(content, section) {
				errs = append(errs, prefix+": missing section '"+section+"'")
			}
		}
		return errs
	}

	for _, field := range requiredFields {
		val, exists := fm[field]
		if !exists || strings.TrimSpace(val) == "" {
			errs = append(errs, prefix+": missing '"+field+"' field in frontmatter")
		}
	}

	for _, section := range requiredSections {
		if !strings.Contains(body, section) {
			errs = append(errs, prefix+": missing section '"+section+"'")
		}
	}

	return errs
}

// parseFrontmatter extracts key-value pairs from YAML frontmatter delimited by ---
// Returns (fields, body, ok). ok=false if frontmatter is absent.
func parseFrontmatter(content string) (map[string]string, string, bool) {
	if !strings.HasPrefix(content, "---") {
		return nil, content, false
	}
	rest := content[3:]
	// Skip optional \r\n or \n after opening ---
	rest = strings.TrimLeft(rest, "\r\n")

	// Find closing ---
	idx := strings.Index(rest, "\n---")
	if idx == -1 {
		// Try without leading newline (edge case: --- immediately after opening)
		if strings.HasPrefix(rest, "---") {
			return map[string]string{}, rest[3:], true
		}
		return nil, content, false
	}

	fmBlock := rest[:idx]
	body := rest[idx+4:] // skip \n---

	fields := make(map[string]string)
	for _, line := range strings.Split(fmBlock, "\n") {
		line = strings.TrimRight(line, "\r")
		colonIdx := strings.Index(line, ":")
		if colonIdx < 0 {
			continue
		}
		key := strings.TrimSpace(line[:colonIdx])
		value := strings.TrimSpace(line[colonIdx+1:])
		// Strip surrounding quotes from value
		if len(value) >= 2 && value[0] == '"' && value[len(value)-1] == '"' {
			value = value[1 : len(value)-1]
		}
		if key != "" {
			fields[key] = value
		}
	}
	return fields, body, true
}
