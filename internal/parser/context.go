package parser

import (
	"strings"
)

// ContextInfo represents the context information extracted from OKR.md and SPEC.md
type ContextInfo struct {
	Objectives      []string // 项目目标
	KeyResults      []string // 关键结果
	Specifications  []string // 开发规范
}

// ParseOKR parses OKR.md content and extracts objectives and key results
func ParseOKR(content string) (*ContextInfo, error) {
	if content == "" {
		return &ContextInfo{
			Objectives: []string{},
			KeyResults: []string{},
		}, nil
	}

	info := &ContextInfo{}

	objectives, err := ExtractObjectives(content)
	if err != nil {
		return nil, err
	}
	info.Objectives = objectives

	keyResults, err := ExtractKeyResults(content)
	if err != nil {
		return nil, err
	}
	info.KeyResults = keyResults

	return info, nil
}

// ParseSPEC parses SPEC.md content and extracts specifications
func ParseSPEC(content string) (*ContextInfo, error) {
	if content == "" {
		return &ContextInfo{
			Specifications: []string{},
		}, nil
	}

	info := &ContextInfo{}

	specs, err := ExtractSpecifications(content)
	if err != nil {
		return nil, err
	}
	info.Specifications = specs

	return info, nil
}

// ExtractObjectives extracts objectives from OKR.md content
// Looks for "# 目标" or "# Objectives" section
func ExtractObjectives(content string) ([]string, error) {
	sectionContent := extractSectionContent(content, "# 目标")
	if sectionContent == "" {
		// Try English version
		sectionContent = extractSectionContent(content, "# Objectives")
	}

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

// ExtractKeyResults extracts key results from OKR.md content
// Looks for "# 关键结果" or "# Key Results" section
func ExtractKeyResults(content string) ([]string, error) {
	sectionContent := extractSectionContent(content, "# 关键结果")
	if sectionContent == "" {
		// Try English version
		sectionContent = extractSectionContent(content, "# Key Results")
	}

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

// ExtractSpecifications extracts specifications from SPEC.md content
// Looks for "# 规范" or "# Specifications" section
func ExtractSpecifications(content string) ([]string, error) {
	sectionContent := extractSectionContent(content, "# 规范")
	if sectionContent == "" {
		// Try English version
		sectionContent = extractSectionContent(content, "# Specifications")
	}

	if sectionContent == "" {
		// Try alternative Chinese
		sectionContent = extractSectionContent(content, "# 开发规范")
	}

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
