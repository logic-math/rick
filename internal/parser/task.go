package parser

import (
	"fmt"
	"strings"
)

// Task represents a task extracted from task.md
type Task struct {
	ID           string   // task1, task2, ...
	Name         string   // 任务名称
	Goal         string   // 任务目标
	KeyResults   []string // 关键结果列表
	TestMethod   string   // 测试方法
	Dependencies []string // 依赖的 task IDs
}

// ParseTask parses a complete task.md content and returns a Task struct
func ParseTask(content string) (*Task, error) {
	task := &Task{}

	// Parse dependencies
	deps, err := ParseDependencies(content)
	if err != nil {
		return nil, fmt.Errorf("failed to parse dependencies: %w", err)
	}
	task.Dependencies = deps

	// Parse task name
	name, err := ParseTaskName(content)
	if err != nil {
		return nil, fmt.Errorf("failed to parse task name: %w", err)
	}
	task.Name = name

	// Parse goal
	goal, err := ParseGoal(content)
	if err != nil {
		return nil, fmt.Errorf("failed to parse goal: %w", err)
	}
	task.Goal = goal

	// Parse key results
	keyResults, err := ParseKeyResults(content)
	if err != nil {
		return nil, fmt.Errorf("failed to parse key results: %w", err)
	}
	task.KeyResults = keyResults

	// Parse test method
	testMethod, err := ParseTestMethod(content)
	if err != nil {
		return nil, fmt.Errorf("failed to parse test method: %w", err)
	}
	task.TestMethod = testMethod

	return task, nil
}

// ParseDependencies extracts dependencies from "# 依赖关系" section
func ParseDependencies(content string) ([]string, error) {
	heading := extractSectionContent(content, "# 依赖关系")
	if heading == "" {
		return []string{}, nil
	}

	// Split by comma and trim whitespace
	parts := strings.Split(heading, ",")
	var deps []string
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		// Skip empty strings and keywords that mean "no dependencies"
		if trimmed == "" || isNoDependency(trimmed) {
			continue
		}
		deps = append(deps, trimmed)
	}
	return deps, nil
}

// isNoDependency checks if a string represents "no dependency"
func isNoDependency(s string) bool {
	lower := strings.ToLower(s)
	noDeps := []string{"无", "none", "null", "nil", "n/a", "na", "-"}
	for _, noDep := range noDeps {
		if lower == noDep {
			return true
		}
	}
	return false
}

// ParseTaskName extracts task name from "# 任务名称" section
func ParseTaskName(content string) (string, error) {
	heading := extractSectionContent(content, "# 任务名称")
	if heading == "" {
		return "", fmt.Errorf("missing '# 任务名称' section")
	}
	return strings.TrimSpace(heading), nil
}

// ParseGoal extracts goal from "# 任务目标" section
func ParseGoal(content string) (string, error) {
	heading := extractSectionContent(content, "# 任务目标")
	if heading == "" {
		return "", fmt.Errorf("missing '# 任务目标' section")
	}
	return strings.TrimSpace(heading), nil
}

// ParseKeyResults extracts key results from "# 关键结果" section
func ParseKeyResults(content string) ([]string, error) {
	sectionContent := extractSectionContent(content, "# 关键结果")
	if sectionContent == "" {
		return []string{}, nil
	}

	// Extract list items by parsing lines starting with "- " or "* " or numbered lists
	var items []string
	lines := strings.Split(sectionContent, "\n")
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		// Handle bullet points (- or *)
		if strings.HasPrefix(trimmed, "- ") {
			items = append(items, strings.TrimPrefix(trimmed, "- "))
		} else if strings.HasPrefix(trimmed, "* ") {
			items = append(items, strings.TrimPrefix(trimmed, "* "))
		} else if len(trimmed) > 0 && trimmed[0] >= '0' && trimmed[0] <= '9' {
			// Handle numbered lists (1. 2. etc)
			parts := strings.SplitN(trimmed, ". ", 2)
			if len(parts) == 2 {
				items = append(items, parts[1])
			}
		}
	}
	return items, nil
}

// ParseTestMethod extracts test method from "# 测试方法" section
func ParseTestMethod(content string) (string, error) {
	sectionContent := extractSectionContent(content, "# 测试方法")
	if sectionContent == "" {
		return "", nil
	}

	// Extract list items by parsing lines starting with "- " or "* " or numbered lists
	var items []string
	lines := strings.Split(sectionContent, "\n")
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		// Handle bullet points (- or *)
		if strings.HasPrefix(trimmed, "- ") {
			items = append(items, strings.TrimPrefix(trimmed, "- "))
		} else if strings.HasPrefix(trimmed, "* ") {
			items = append(items, strings.TrimPrefix(trimmed, "* "))
		} else if len(trimmed) > 0 && trimmed[0] >= '0' && trimmed[0] <= '9' {
			// Handle numbered lists (1. 2. etc)
			parts := strings.SplitN(trimmed, ". ", 2)
			if len(parts) == 2 {
				items = append(items, parts[1])
			}
		}
	}

	if len(items) == 0 {
		return "", nil
	}
	return strings.Join(items, "\n"), nil
}

// extractSectionContent extracts content between two section headings
// It finds the section starting with the given heading and returns content until the next heading
func extractSectionContent(content string, sectionHeading string) string {
	lines := strings.Split(content, "\n")
	var result []string
	foundSection := false
	var currentLevel int

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Check if this is the section we're looking for
		if trimmed == sectionHeading {
			foundSection = true
			// Determine the heading level
			currentLevel = strings.Count(strings.TrimPrefix(trimmed, " "), "#")
			continue
		}

		// If we found the section, collect lines until we hit another heading of the same or higher level
		if foundSection {
			// Check if we hit another heading
			if strings.HasPrefix(trimmed, "#") {
				headingLevel := strings.Count(strings.TrimPrefix(trimmed, " "), "#")
				// If this heading is at the same level or higher (lower number), we've reached the next section
				if headingLevel <= currentLevel {
					break
				}
			}
			// Skip empty lines at the beginning
			if len(result) > 0 || trimmed != "" {
				result = append(result, line)
			}
		}
	}

	// Remove leading empty lines
	for len(result) > 0 && strings.TrimSpace(result[0]) == "" {
		result = result[1:]
	}

	// Remove trailing empty lines
	for len(result) > 0 && strings.TrimSpace(result[len(result)-1]) == "" {
		result = result[:len(result)-1]
	}

	return strings.Join(result, "\n")
}

// ValidateTask validates that a Task has all required fields
func ValidateTask(task *Task) error {
	if task.Name == "" {
		return fmt.Errorf("task name is required")
	}
	if task.Goal == "" {
		return fmt.Errorf("task goal is required")
	}
	if len(task.KeyResults) == 0 {
		return fmt.Errorf("task key results are required")
	}
	if task.TestMethod == "" {
		return fmt.Errorf("task test method is required")
	}
	return nil
}
