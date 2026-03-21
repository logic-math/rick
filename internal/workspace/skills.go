package workspace

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// SkillInfo holds metadata for a single skill file
type SkillInfo struct {
	Path        string // absolute path to the .py file
	Name        string // filename without extension
	Description string // content of the "# Description: ..." first-line comment
}

// LoadSkillsList scans rickDir/skills/*.py and extracts description from the first line.
// Returns an empty slice (not error) when the skills directory doesn't exist or is empty.
func LoadSkillsList(rickDir string) ([]SkillInfo, error) {
	skillsDir := filepath.Join(rickDir, SkillsDirName)

	info, err := os.Stat(skillsDir)
	if err != nil || !info.IsDir() {
		return nil, nil
	}

	entries, err := os.ReadDir(skillsDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read skills directory: %w", err)
	}

	var skills []SkillInfo
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".py") {
			continue
		}

		filePath := filepath.Join(skillsDir, e.Name())
		desc := extractSkillDescription(filePath)
		skills = append(skills, SkillInfo{
			Path:        filePath,
			Name:        strings.TrimSuffix(e.Name(), ".py"),
			Description: desc,
		})
	}

	return skills, nil
}

// extractSkillDescription reads the first line of a .py file and extracts the description
// from a "# Description: ..." comment. Returns empty string if not found.
func extractSkillDescription(filePath string) string {
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

// GenerateSkillsREADME regenerates rickDir/skills/README.md from all .py files.
func GenerateSkillsREADME(rickDir string) error {
	skills, err := LoadSkillsList(rickDir)
	if err != nil {
		return err
	}

	skillsDir := filepath.Join(rickDir, SkillsDirName)
	if err := os.MkdirAll(skillsDir, 0755); err != nil {
		return fmt.Errorf("failed to create skills directory: %w", err)
	}

	var sb strings.Builder
	sb.WriteString("# Skills\n\n")
	sb.WriteString("| 文件 | 描述 |\n")
	sb.WriteString("|------|------|\n")

	for _, s := range skills {
		sb.WriteString(fmt.Sprintf("| %s.py | %s |\n", s.Name, s.Description))
	}

	readmePath := filepath.Join(skillsDir, "README.md")
	return os.WriteFile(readmePath, []byte(sb.String()), 0644)
}
