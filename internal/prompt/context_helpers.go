package prompt

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/sunquan/rick/internal/parser"
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

func formatOKRContent(okrInfo *parser.ContextInfo) string {
	if okrInfo == nil || (len(okrInfo.Objectives) == 0 && len(okrInfo.KeyResults) == 0) {
		return "暂无项目 OKR 信息"
	}
	var b strings.Builder
	if len(okrInfo.Objectives) > 0 {
		b.WriteString("**Objectives**:\n")
		for _, obj := range okrInfo.Objectives {
			b.WriteString(fmt.Sprintf("- %s\n", obj))
		}
		b.WriteString("\n")
	}
	if len(okrInfo.KeyResults) > 0 {
		b.WriteString("**Key Results**:\n")
		for _, kr := range okrInfo.KeyResults {
			b.WriteString(fmt.Sprintf("- %s\n", kr))
		}
	}
	return b.String()
}

func formatSPECContent(specInfo *parser.ContextInfo) string {
	if specInfo == nil || len(specInfo.Specifications) == 0 {
		return "暂无项目 SPEC 信息"
	}
	var b strings.Builder
	b.WriteString("**Specifications**:\n")
	for _, spec := range specInfo.Specifications {
		b.WriteString(fmt.Sprintf("- %s\n", spec))
	}
	return b.String()
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
