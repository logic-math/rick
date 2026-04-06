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

// LoadSkillsIndex reads the content of rickDir/skills/index.md and returns it as a string.
// Returns empty string (not error) when the file doesn't exist.
func LoadSkillsIndex(rickDir string) (string, error) {
	indexPath := filepath.Join(rickDir, SkillsDirName, "index.md")
	data, err := os.ReadFile(indexPath)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", fmt.Errorf("failed to read skills index: %w", err)
	}
	return string(data), nil
}

// GenerateSkillsIndex regenerates rickDir/skills/index.md from all .py files.
func GenerateSkillsIndex(rickDir string) error {
	skills, err := LoadSkillsList(rickDir)
	if err != nil {
		return err
	}

	skillsDir := filepath.Join(rickDir, SkillsDirName)
	if err := os.MkdirAll(skillsDir, 0755); err != nil {
		return fmt.Errorf("failed to create skills directory: %w", err)
	}

	var sb strings.Builder
	sb.WriteString("# Skills Index\n\n")
	sb.WriteString("本目录包含可在 doing 阶段调用的 Python 脚本工具。\n\n")
	sb.WriteString("## 可用 Skills\n\n")
	sb.WriteString("| 文件 | 描述 | 触发场景 |\n")
	sb.WriteString("|------|------|----------|\n")

	for _, s := range skills {
		sb.WriteString(fmt.Sprintf("| %s.py | %s | |\n", s.Name, s.Description))
	}

	sb.WriteString("\n## 调用方式\n\n")
	sb.WriteString("```bash\npython3 .rick/skills/<filename>.py\n```\n")

	indexPath := filepath.Join(skillsDir, "index.md")
	return os.WriteFile(indexPath, []byte(sb.String()), 0644)
}

// GenerateSkillsREADME is an alias for GenerateSkillsIndex for backward compatibility.
// Deprecated: Use GenerateSkillsIndex instead.
func GenerateSkillsREADME(rickDir string) error {
	return GenerateSkillsIndex(rickDir)
}
