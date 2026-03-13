package prompt

import (
	"fmt"
	"strings"
)

// PromptBuilder builds prompts by substituting variables and context into templates
type PromptBuilder struct {
	Template  *PromptTemplate
	Variables map[string]string
	Context   map[string]interface{}
}

// NewPromptBuilder creates a new PromptBuilder instance with the given template
func NewPromptBuilder(template *PromptTemplate) *PromptBuilder {
	return &PromptBuilder{
		Template:  template,
		Variables: make(map[string]string),
		Context:   make(map[string]interface{}),
	}
}

// SetVariable sets a template variable
func (pb *PromptBuilder) SetVariable(key, value string) *PromptBuilder {
	pb.Variables[key] = value
	return pb
}

// SetContext sets a context value
func (pb *PromptBuilder) SetContext(key string, value interface{}) *PromptBuilder {
	pb.Context[key] = value
	return pb
}

// Build constructs the final prompt by replacing variables and injecting context
func (pb *PromptBuilder) Build() (string, error) {
	if pb.Template == nil {
		return "", fmt.Errorf("template is not set")
	}

	result := pb.Template.Content

	// Replace variables first
	result = pb.replaceVariables(result)

	// Inject context
	result = pb.injectContext(result)

	return result, nil
}

// replaceVariables replaces {{variable}} placeholders with their values
func (pb *PromptBuilder) replaceVariables(content string) string {
	result := content

	// Replace each variable
	for key, value := range pb.Variables {
		placeholder := "{{" + key + "}}"
		result = strings.ReplaceAll(result, placeholder, value)
	}

	return result
}

// injectContext injects context values into the prompt
// Context values are converted to strings and injected into the prompt
func (pb *PromptBuilder) injectContext(content string) string {
	result := content

	// Inject each context value
	for key, value := range pb.Context {
		placeholder := "{{" + key + "}}"
		valueStr := fmt.Sprintf("%v", value)
		result = strings.ReplaceAll(result, placeholder, valueStr)
	}

	return result
}

// GetVariables returns the list of variables that need to be set
func (pb *PromptBuilder) GetVariables() []string {
	if pb.Template == nil {
		return []string{}
	}
	return pb.Template.Variables
}

// GetMissingVariables returns variables that haven't been set
func (pb *PromptBuilder) GetMissingVariables() []string {
	var missing []string
	for _, variable := range pb.GetVariables() {
		if _, exists := pb.Variables[variable]; !exists {
			if _, existsInContext := pb.Context[variable]; !existsInContext {
				missing = append(missing, variable)
			}
		}
	}
	return missing
}
