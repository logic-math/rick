package prompt

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

// PromptTemplate represents a prompt template with variables
type PromptTemplate struct {
	Name      string
	Content   string
	Variables []string
}

// PromptManager manages prompt templates with caching
type PromptManager struct {
	templateDir string
	cache       map[string]*PromptTemplate
	mu          sync.RWMutex
}

// NewPromptManager creates a new PromptManager instance
func NewPromptManager(templateDir string) *PromptManager {
	return &PromptManager{
		templateDir: templateDir,
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

	// Load from file
	templatePath := filepath.Join(pm.templateDir, name+".md")
	content, err := os.ReadFile(templatePath)
	if err != nil {
		return nil, fmt.Errorf("failed to load template %s: %w", name, err)
	}

	template := &PromptTemplate{
		Name:      name,
		Content:   string(content),
		Variables: extractVariables(string(content)),
	}

	// Store in cache
	pm.mu.Lock()
	pm.cache[name] = template
	pm.mu.Unlock()

	return template, nil
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
