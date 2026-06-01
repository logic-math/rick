package prompt

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// GenerateDreamPromptFile generates the dream phase prompt and saves it to a temporary file.
// jobIDs is the list of jobs to process; rickDir is the .rick directory path.
// Returns the temp file path; caller is responsible for cleanup.
func GenerateDreamPromptFile(jobIDs []string, rickDir string) (string, error) {
	mgr := NewPromptManager()

	tmpl, err := mgr.LoadTemplate("dream")
	if err != nil {
		return "", fmt.Errorf("failed to load dream template: %w", err)
	}

	builder := NewPromptBuilder(tmpl)
	builder.SetVariable("pending_jobs", formatPendingJobs(jobIDs))
	builder.SetVariable("run_logs", loadRunLogs(rickDir))

	content, err := builder.Build()
	if err != nil {
		return "", fmt.Errorf("failed to build dream prompt: %w", err)
	}

	// Append act-path context for each job
	actPathContent := loadActPaths(jobIDs, rickDir)
	if actPathContent != "" {
		content += "\n\n## 行为轨迹（act-path）\n\n" + actPathContent
	}

	// Append core skills: only sense and evolve-skills
	coreSkills := LoadCoreSkills([]string{"sense", "evolve-skills"})
	if coreSkills != "" {
		content += "\n\n## Core Skills\n\n" + coreSkills
	}

	tmpFile, err := os.CreateTemp("", "rick-dream-*.md")
	if err != nil {
		return "", fmt.Errorf("failed to create temp file: %w", err)
	}
	defer tmpFile.Close()

	if _, err := tmpFile.WriteString(content); err != nil {
		os.Remove(tmpFile.Name())
		return "", fmt.Errorf("failed to write dream prompt: %w", err)
	}

	return tmpFile.Name(), nil
}

func formatPendingJobs(jobIDs []string) string {
	if len(jobIDs) == 0 {
		return "（无待处理 jobs）"
	}
	var sb strings.Builder
	for _, id := range jobIDs {
		sb.WriteString("- " + id + "\n")
	}
	return sb.String()
}

func loadRunLogs(rickDir string) string {
	if rickDir == "" {
		return "（无 run logs）"
	}
	dreamDir := filepath.Join(rickDir, "dream")
	entries, err := os.ReadDir(dreamDir)
	if err != nil {
		return "（无 run logs）"
	}

	var logs []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasPrefix(e.Name(), "run_log_") && strings.HasSuffix(e.Name(), ".md") {
			logs = append(logs, e.Name())
		}
	}
	if len(logs) == 0 {
		return "（无 run logs）"
	}
	sort.Strings(logs)

	var sb strings.Builder
	for _, name := range logs {
		content, err := os.ReadFile(filepath.Join(dreamDir, name))
		if err != nil {
			continue
		}
		sb.WriteString("### " + name + "\n\n")
		sb.Write(content)
		sb.WriteString("\n\n")
	}
	return sb.String()
}

func loadActPaths(jobIDs []string, rickDir string) string {
	if len(jobIDs) == 0 || rickDir == "" {
		return ""
	}
	var sb strings.Builder
	for _, jobID := range jobIDs {
		doingTasksDir := filepath.Join(rickDir, "jobs", jobID, "doing", "tasks")
		entries, err := os.ReadDir(doingTasksDir)
		if err != nil {
			continue
		}
		for _, e := range entries {
			if !e.IsDir() {
				continue
			}
			actPath := filepath.Join(doingTasksDir, e.Name(), "act-path.md")
			content, err := os.ReadFile(actPath)
			if err != nil {
				continue
			}
			sb.WriteString(fmt.Sprintf("#### %s/%s/act-path.md\n\n", jobID, e.Name()))
			sb.Write(content)
			sb.WriteString("\n\n")
		}
	}
	return sb.String()
}
