package prompt

import (
	"embed"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

// PromptTemplate represents a prompt template with variables
type PromptTemplate struct {
	Name      string
	Content   string
	Variables []string
}

// Embedded templates
var (
	//go:embed templates/plan.md
	planTemplate string

	//go:embed templates/doing.md
	doingTemplate string

	//go:embed templates/learning.md
	learningTemplate string

	//go:embed templates/test_python.md
	testPythonTemplate string

	//go:embed templates/human_loop.md
	humanLoopTemplate string

	//go:embed templates/human_loop_think.md
	humanLoopThinkTemplate string

	//go:embed templates/human_loop_learn.md
	humanLoopLearnTemplate string

	//go:embed templates/human_loop_express.md
	humanLoopExpressTemplate string

	//go:embed templates/dream.md
	dreamTemplate string

	//go:embed templates/easy.md
	easyTemplate string

	//go:embed templates/easy_ctx.md
	easyCtxTemplate string

	//go:embed templates/ctrl.md
	ctrlTemplate string

	//go:embed templates/skills
	skillsFS embed.FS
)

// PromptManager manages prompt templates with caching
type PromptManager struct {
	templateDir string
	cache       map[string]*PromptTemplate
	mu          sync.RWMutex
}

// NewPromptManager creates a new PromptManager instance.
// An optional templateDir can be provided; if omitted or empty, embedded templates are used.
func NewPromptManager(templateDir ...string) *PromptManager {
	dir := ""
	if len(templateDir) > 0 {
		dir = templateDir[0]
	}
	return &PromptManager{
		templateDir: dir,
		cache:       make(map[string]*PromptTemplate),
	}
}

// LoadTemplate loads a template from file with caching
func (pm *PromptManager) LoadTemplate(name string) (*PromptTemplate, error) {
	// Check cache first
	pm.mu.RLock()
	if template, exists := pm.cache[name]; exists {
		pm.mu.RUnlock()
		return template, nil
	}
	pm.mu.RUnlock()

	var content string

	// If templateDir is empty or file doesn't exist, use embedded templates
	if pm.templateDir == "" {
		content = pm.getEmbeddedTemplate(name)
		if content == "" {
			return nil, fmt.Errorf("embedded template %s not found", name)
		}
	} else {
		// Try to load from file
		templatePath := filepath.Join(pm.templateDir, name+".md")
		fileContent, err := os.ReadFile(templatePath)
		if err != nil {
			// Fallback to embedded template
			content = pm.getEmbeddedTemplate(name)
			if content == "" {
				return nil, fmt.Errorf("failed to load template %s: %w", name, err)
			}
		} else {
			content = string(fileContent)
		}
	}

	template := &PromptTemplate{
		Name:      name,
		Content:   content,
		Variables: extractVariables(content),
	}

	// Store in cache
	pm.mu.Lock()
	pm.cache[name] = template
	pm.mu.Unlock()

	return template, nil
}

// getEmbeddedTemplate returns the embedded template content by name
func (pm *PromptManager) getEmbeddedTemplate(name string) string {
	switch name {
	case "plan":
		return planTemplate
	case "doing":
		return doingTemplate
	case "learning":
		return learningTemplate
	case "test_python":
		return testPythonTemplate
	case "human_loop":
		return humanLoopTemplate
	case "human_loop_think":
		return humanLoopThinkTemplate
	case "human_loop_learn":
		return humanLoopLearnTemplate
	case "human_loop_express":
		return humanLoopExpressTemplate
	case "dream":
		return dreamTemplate
	case "easy":
		return easyTemplate
	case "easy_ctx":
		return easyCtxTemplate
	case "ctrl":
		return ctrlTemplate
	default:
		return ""
	}
}

// extractVariables extracts {{variable}} placeholders from template content
func extractVariables(content string) []string {
	var variables []string
	seen := make(map[string]bool)

	// Simple regex-free extraction of {{variable}} patterns
	for i := 0; i < len(content)-3; i++ {
		if content[i:i+2] == "{{" {
			// Find closing }}
			for j := i + 2; j < len(content)-1; j++ {
				if content[j:j+2] == "}}" {
					variable := content[i+2 : j]
					// Trim whitespace
					variable = trimSpace(variable)
					if variable != "" && !seen[variable] {
						variables = append(variables, variable)
						seen[variable] = true
					}
					i = j + 1
					break
				}
			}
		}
	}

	return variables
}

// trimSpace removes leading and trailing whitespace
func trimSpace(s string) string {
	start := 0
	end := len(s)

	for start < end && (s[start] == ' ' || s[start] == '\t' || s[start] == '\n' || s[start] == '\r') {
		start++
	}

	for end > start && (s[end-1] == ' ' || s[end-1] == '\t' || s[end-1] == '\n' || s[end-1] == '\r') {
		end--
	}

	return s[start:end]
}

// ClearCache clears the template cache
func (pm *PromptManager) ClearCache() {
	pm.mu.Lock()
	pm.cache = make(map[string]*PromptTemplate)
	pm.mu.Unlock()
}

// GetCacheSize returns the number of cached templates
func (pm *PromptManager) GetCacheSize() int {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	return len(pm.cache)
}

// LoadCoreSkills reads embedded skill files by name and returns their concatenated content.
// Names like "tdd/testing-anti-patterns" map to templates/skills/tdd/testing-anti-patterns.md.
// All other names map to templates/skills/{name}.md.
// Missing files are logged as warnings and skipped — no panic.
func LoadCoreSkills(names []string) string {
	var parts []string
	for _, name := range names {
		path := "templates/skills/" + name + ".md"
		content, err := skillsFS.ReadFile(path)
		if err != nil {
			log.Printf("warn: core skill not found: %s (%v)", path, err)
			continue
		}
		if len(content) > 0 {
			parts = append(parts, string(content))
		}
	}
	return strings.Join(parts, "\n\n---\n\n")
}

// WriteSkillTmpFile writes a core skill's content to a named tmp file and returns its path.
// Caller is responsible for cleanup via os.Remove.
func WriteSkillTmpFile(pattern string, skillName string) (string, error) {
	content := LoadCoreSkills([]string{skillName})
	f, err := os.CreateTemp("", pattern)
	if err != nil {
		return "", fmt.Errorf("failed to create skill tmp file: %w", err)
	}
	defer f.Close()
	if _, err := f.WriteString(content); err != nil {
		os.Remove(f.Name())
		return "", fmt.Errorf("failed to write skill tmp file: %w", err)
	}
	return f.Name(), nil
}

// EnsurePromptsDir creates and returns the prompts/ subdirectory inside baseDir.
func EnsurePromptsDir(baseDir string) (string, error) {
	dir := filepath.Join(baseDir, "prompts")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", fmt.Errorf("failed to create prompts directory: %w", err)
	}
	return dir, nil
}

// WriteSkillFile writes an embedded skill to dir/filename (persistent, no cleanup needed).
func WriteSkillFile(dir, filename, skillName string) (string, error) {
	return WriteSkillFileWithVars(dir, filename, skillName, nil)
}

// WriteSkillFileWithVars writes an embedded skill with {{key}} variable substitution.
func WriteSkillFileWithVars(dir, filename, skillName string, vars map[string]string) (string, error) {
	content := LoadCoreSkills([]string{skillName})
	if content == "" {
		return "", fmt.Errorf("embedded skill %q not found", skillName)
	}
	for k, v := range vars {
		content = strings.ReplaceAll(content, "{{"+k+"}}", v)
	}
	path := filepath.Join(dir, filename)
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		return "", fmt.Errorf("failed to write skill file: %w", err)
	}
	return path, nil
}
