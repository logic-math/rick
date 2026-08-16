package prompt

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
)

// LoadLoopsContext reads all *.md files in loopsDir, extracts name/trigger from
// YAML frontmatter, and returns a formatted loops list for prompt injection.
func LoadLoopsContext(loopsDir string) string {
	header := "## 可用的项目 Loops\n\n"
	placeholder := header + "（暂无项目 Loop 记录）\n"

	entries, err := os.ReadDir(loopsDir)
	if err != nil {
		return placeholder
	}

	var items []string
	for _, e := range entries {
		if e.IsDir() || filepath.Ext(e.Name()) != ".md" {
			continue
		}
		data, err := os.ReadFile(filepath.Join(loopsDir, e.Name()))
		if err != nil {
			log.Printf("warn: LoadLoopsContext: failed to read %s: %v", e.Name(), err)
			continue
		}
		name, trigger := parseFrontmatterNameTrigger(string(data))
		if trigger == "" {
			log.Printf("warn: LoadLoopsContext: no trigger in %s, skipping", e.Name())
			continue
		}
		items = append(items, fmt.Sprintf("- **%s**：%s", name, trigger))
	}

	if len(items) == 0 {
		return placeholder
	}
	return header + strings.Join(items, "\n") + "\n"
}

// LoadSkillsContext reads all *_skill/ directories in skillsDir, extracts name and trigger
// from each skill.md, and returns a formatted skills list for prompt injection.
func LoadSkillsContext(skillsDir string) string {
	header := "## 可用的项目 Skills\n\n"
	placeholder := header + "（暂无项目 Skill 记录）\n"

	entries, err := os.ReadDir(skillsDir)
	if err != nil {
		return placeholder
	}

	var items []string
	for _, e := range entries {
		if !e.IsDir() || !strings.HasSuffix(e.Name(), "_skill") {
			continue
		}
		skillFile := filepath.Join(skillsDir, e.Name(), "skill.md")
		data, err := os.ReadFile(skillFile)
		if err != nil {
			log.Printf("warn: LoadSkillsContext: failed to read %s: %v", skillFile, err)
			continue
		}
		name, trigger := parseSkillNameTrigger(string(data))
		if trigger == "" {
			log.Printf("warn: LoadSkillsContext: no trigger section in %s, skipping", skillFile)
			continue
		}
		items = append(items, fmt.Sprintf("- **%s**：%s", name, trigger))
	}

	if len(items) == 0 {
		return placeholder
	}
	return header + strings.Join(items, "\n") + "\n"
}

// parseSkillNameTrigger extracts skill name from the # heading and the first line of ## 触发场景.
func parseSkillNameTrigger(content string) (name, trigger string) {
	lines := strings.Split(content, "\n")
	var inTrigger bool
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "# skill:") && name == "" {
			rest := strings.TrimPrefix(trimmed, "# skill:")
			if idx := strings.Index(rest, "（"); idx >= 0 {
				name = rest[:idx]
			} else if fields := strings.Fields(rest); len(fields) > 0 {
				name = fields[0]
			}
		}
		if trimmed == "## 触发场景" {
			inTrigger = true
			continue
		}
		if inTrigger && strings.HasPrefix(trimmed, "## ") {
			break
		}
		if inTrigger && trimmed != "" && trigger == "" {
			trigger = strings.TrimRight(trimmed, "：:")
		}
	}
	return name, trigger
}

// parseFrontmatterNameTrigger extracts name and trigger from YAML frontmatter.
func parseFrontmatterNameTrigger(content string) (name, trigger string) {
	if !strings.HasPrefix(content, "---") {
		return "", ""
	}
	rest := content[3:]
	end := strings.Index(rest, "\n---")
	if end < 0 {
		return "", ""
	}
	block := rest[:end]
	for _, line := range strings.Split(block, "\n") {
		line = strings.TrimSpace(line)
		if k, v, ok := strings.Cut(line, ":"); ok {
			k = strings.TrimSpace(k)
			v = strings.TrimSpace(v)
			switch k {
			case "name":
				name = v
			case "trigger":
				trigger = v
			}
		}
	}
	return name, trigger
}

func formatCompletedWork(history []string) string {
	if len(history) == 0 {
		return "这是项目的第一阶段规划"
	}
	var b strings.Builder
	b.WriteString("**已完成的工作:**\n")
	for _, item := range history {
		b.WriteString(fmt.Sprintf("- %s\n", item))
	}
	return b.String()
}
